package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// decodeOpenAIJSONUseNumber decodes one JSON document while preserving number
// tokens as json.Number. Responses/WebSocket payloads may contain identifiers
// larger than the float64 safe-integer range; decoding through float64 would
// silently change those identifiers during compatibility normalization.
func decodeOpenAIJSONUseNumber(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}
