package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const pricingOverridesFileName = "model_pricing_overrides.json"

// applyPricingOverrides layers operator-maintained, exact-model prices over a
// freshly parsed snapshot. Call after the existing fallback merge, before
// publishing the snapshot. The remote cache body and hash must remain unchanged.
//
// The file uses the LiteLLM model-ID-to-entry format. A top-level string $schema
// and unknown per-entry metadata (for example source/sources) are permitted.
// Prices are replaced as a group; omitted capabilities and model metadata are
// inherited from the previous entry, while explicitly supplied metadata wins.
// Every model is validated before any replacement is made. An error returns the
// original map unchanged, so the caller can retain its previous valid snapshot.
func (s *PricingService) applyPricingOverrides(data map[string]*LiteLLMModelPricing) (map[string]*LiteLLMModelPricing, error) {
	if s == nil || s.cfg == nil || strings.TrimSpace(s.cfg.Pricing.DataDir) == "" {
		return data, nil
	}
	filePath := filepath.Join(s.cfg.Pricing.DataDir, pricingOverridesFileName)
	body, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return data, nil
	}
	if err != nil {
		return data, fmt.Errorf("read pricing overrides: %w", err)
	}
	overrides, err := s.parsePricingOverrides(body)
	if err != nil {
		return data, fmt.Errorf("invalid pricing overrides %s: %w", filePath, err)
	}
	if len(overrides) == 0 {
		return data, nil
	}
	var rawEntries map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawEntries); err != nil {
		return data, fmt.Errorf("read validated pricing override metadata: %w", err)
	}
	merged := make(map[string]*LiteLLMModelPricing, len(data)+len(overrides))
	baseModels := make([]string, 0, len(data))
	for model, pricing := range data {
		merged[model] = pricing
		baseModels = append(baseModels, model)
	}
	sort.Strings(baseModels)
	for model, pricing := range overrides {
		previous, exact := data[model]
		if !exact {
			// Prefer an exact-case entry; otherwise pick a stable case-insensitive
			// match when the base source contains multiple spellings of an ID.
			for _, existing := range baseModels {
				if strings.EqualFold(existing, model) {
					previous = data[existing]
					break
				}
			}
		}
		fields, err := pricingOverrideObject(rawEntries[model])
		if err != nil {
			return data, fmt.Errorf("read validated metadata for %q: %w", model, err)
		}
		inheritPricingOverrideMetadata(pricing, previous, fields)
		// Missing prices never inherit stale values or presence bits. Lookups
		// ignore case, so remove differently cased copies of the same ID.
		for existing := range merged {
			if strings.EqualFold(existing, model) {
				delete(merged, existing)
			}
		}
		merged[model] = pricing
	}
	return merged, nil
}

func inheritPricingOverrideMetadata(pricing, previous *LiteLLMModelPricing, fields map[string]json.RawMessage) {
	if previous == nil {
		return
	}
	value := reflect.ValueOf(pricing).Elem()
	oldValue := reflect.ValueOf(previous).Elem()
	typeInfo := value.Type()
	for i := 0; i < value.NumField(); i++ {
		name := strings.Split(typeInfo.Field(i).Tag.Get("json"), ",")[0]
		metadata := name == "litellm_provider" || name == "mode" || name == "max_input_tokens" || name == "max_output_tokens" ||
			(strings.HasPrefix(name, "supports_") && name != "supports_service_tier")
		if _, supplied := fields[name]; metadata && !supplied {
			value.Field(i).Set(oldValue.Field(i))
		}
	}
}

func (s *PricingService) parsePricingOverrides(body []byte) (map[string]*LiteLLMModelPricing, error) {
	entries, err := pricingOverrideObject(body)
	if err != nil {
		return nil, err
	}
	seenModels := make([]string, 0, len(entries))
	for model, rawEntry := range entries {
		if model == "$schema" {
			var schema string
			if bytes.Equal(bytes.TrimSpace(rawEntry), []byte("null")) || json.Unmarshal(rawEntry, &schema) != nil {
				return nil, fmt.Errorf("$schema must be a string")
			}
			delete(entries, model)
			continue
		}
		if model == "sample_spec" || strings.TrimSpace(model) == "" || strings.TrimSpace(model) != model {
			return nil, fmt.Errorf("invalid model ID %q", model)
		}
		for _, seen := range seenModels {
			if strings.EqualFold(seen, model) {
				return nil, fmt.Errorf("case-insensitive duplicate model IDs %q and %q", seen, model)
			}
		}
		seenModels = append(seenModels, model)
		fields, err := pricingOverrideObject(rawEntry)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", model, err)
		}
		// encoding/json accepts case-insensitive struct-field aliases. Require
		// canonical names for known fields so alternate casing cannot hide a
		// second, conflicting price or bypass explicit-null validation.
		entryType := reflect.TypeOf(LiteLLMRawEntry{})
		for i := 0; i < entryType.NumField(); i++ {
			canonical := strings.Split(entryType.Field(i).Tag.Get("json"), ",")[0]
			for name := range fields {
				if name != canonical && strings.EqualFold(name, canonical) {
					return nil, fmt.Errorf("model %q: use canonical field %q instead of %q", model, canonical, name)
				}
			}
		}
		var entry LiteLLMRawEntry
		if err := json.Unmarshal(rawEntry, &entry); err != nil {
			return nil, fmt.Errorf("model %q: %w", model, err)
		}
		if err := validatePricingOverrideNumbers(entry, fields); err != nil {
			return nil, fmt.Errorf("model %q: %w", model, err)
		}
		if entry.InputCostPerToken == nil && entry.OutputCostPerToken == nil && entry.OutputCostPerImage == nil && entry.OutputCostPerImageToken == nil && entry.InputCostPerImageToken == nil {
			return nil, fmt.Errorf("model %q must specify an input/output token or image price", model)
		}
	}
	if len(entries) == 0 {
		return map[string]*LiteLLMModelPricing{}, nil
	}
	// Reuse the canonical conversion only after validating every entry, keeping
	// explicit-zero presence bits and image-only TokenPricingAbsent behavior.
	validatedBody, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("encode validated pricing overrides: %w", err)
	}
	return s.parsePricingData(validatedBody)
}

// pricingOverrideObject rejects non-objects, duplicate keys and trailing JSON.
// Rejecting duplicates prevents a later value from hiding an invalid price or
// silently selecting a different model definition during review.
func pricingOverrideObject(body []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("read JSON object: %w", err)
	}
	if start != json.Delim('{') {
		return nil, fmt.Errorf("expected a JSON object")
	}
	result := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("read JSON key: %w", err)
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string JSON key")
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate JSON key %q", key)
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("read JSON field %q: %w", key, err)
		}
		result[key] = raw
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("close JSON object: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("unexpected content after JSON object")
	}
	return result, nil
}

func validatePricingOverrideNumbers(entry LiteLLMRawEntry, fields map[string]json.RawMessage) error {
	// All numeric fields in LiteLLMRawEntry are optional pointers. Inspect the
	// canonical type so new price fields cannot accidentally bypass validation.
	value := reflect.ValueOf(entry)
	typeInfo := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() != reflect.Pointer {
			continue
		}
		name := strings.Split(typeInfo.Field(i).Tag.Get("json"), ",")[0]
		if field.IsNil() {
			if _, present := fields[name]; present {
				return fmt.Errorf("%s cannot be null; omit an unknown price or limit", name)
			}
			continue
		}
		switch number := field.Elem(); number.Kind() {
		case reflect.Float64:
			n := number.Float()
			if math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
				return fmt.Errorf("%s must be finite and non-negative", name)
			}
		case reflect.Int, reflect.Int64:
			if number.Int() < 0 {
				return fmt.Errorf("%s must be non-negative", name)
			}
		}
	}
	return nil
}
