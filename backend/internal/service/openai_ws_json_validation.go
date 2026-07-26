package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrJSONDuplicateObjectKey = errors.New("duplicate JSON object key")

type openAIWSJSONRootType struct {
	present  bool
	value    string
	isString bool
}

// ValidateJSONNoDuplicateObjectKeys validates one complete JSON value and
// rejects duplicate object keys at every nesting level. Keys are compared
// after JSON string decoding, so escaped equivalents such as "type" and
// "t\u0079pe" are duplicates.
func ValidateJSONNoDuplicateObjectKeys(payload []byte) error {
	_, err := validateJSONNoDuplicateObjectKeys(payload, false)
	return err
}

// ValidateOpenAIWSClientFrameJSON validates an OpenAI WebSocket client frame
// before callers inspect fields with first-match JSON helpers.
func ValidateOpenAIWSClientFrameJSON(payload []byte) (string, error) {
	rootType, err := validateJSONNoDuplicateObjectKeys(payload, true)
	if err != nil {
		return "", err
	}
	if !rootType.present {
		return "", nil
	}
	if !rootType.isString {
		return "", errors.New("websocket request type must be a string")
	}
	return rootType.value, nil
}

// ValidateOpenAIWSFirstClientFrameJSON additionally enforces the first-frame
// protocol contract: it must be exactly one response.create JSON object.
func ValidateOpenAIWSFirstClientFrameJSON(payload []byte) error {
	eventType, err := ValidateOpenAIWSClientFrameJSON(payload)
	if err != nil {
		return err
	}
	if eventType != "response.create" {
		return errors.New("first websocket message must have type response.create")
	}
	return nil
}

func validateJSONNoDuplicateObjectKeys(payload []byte, requireRootObject bool) (openAIWSJSONRootType, error) {
	var rootType openAIWSJSONRootType
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	firstToken, err := decoder.Token()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return rootType, errors.New("empty JSON payload")
		}
		return rootType, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if requireRootObject {
		delim, ok := firstToken.(json.Delim)
		if !ok || delim != '{' {
			return rootType, errors.New("websocket request payload must be a JSON object")
		}
	}
	if err := walkJSONValueNoDuplicateKeys(decoder, firstToken, true, &rootType); err != nil {
		return rootType, err
	}
	if trailing, trailingErr := decoder.Token(); trailingErr == nil {
		return rootType, fmt.Errorf("invalid JSON payload: unexpected trailing token %v", trailing)
	} else if !errors.Is(trailingErr, io.EOF) {
		return rootType, fmt.Errorf("invalid JSON payload: %w", trailingErr)
	}
	return rootType, nil
}

func walkJSONValueNoDuplicateKeys(
	decoder *json.Decoder,
	token json.Token,
	root bool,
	rootType *openAIWSJSONRootType,
) error {
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}

	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("%w: %q", ErrJSONDuplicateObjectKey, key)
			}
			seen[key] = struct{}{}

			valueToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON value for key %q: %w", key, err)
			}
			if root && key == "type" {
				rootType.present = true
				rootType.value, rootType.isString = valueToken.(string)
			}
			if err := walkJSONValueNoDuplicateKeys(decoder, valueToken, false, rootType); err != nil {
				return err
			}
		}
		closeToken, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("invalid JSON object: %w", err)
		}
		if closeDelim, ok := closeToken.(json.Delim); !ok || closeDelim != '}' {
			return errors.New("invalid JSON object")
		}
		return nil
	case '[':
		for decoder.More() {
			valueToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON array value: %w", err)
			}
			if err := walkJSONValueNoDuplicateKeys(decoder, valueToken, false, rootType); err != nil {
				return err
			}
		}
		closeToken, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("invalid JSON array: %w", err)
		}
		if closeDelim, ok := closeToken.(json.Delim); !ok || closeDelim != ']' {
			return errors.New("invalid JSON array")
		}
		return nil
	default:
		return fmt.Errorf("invalid JSON delimiter %q", delim)
	}
}
