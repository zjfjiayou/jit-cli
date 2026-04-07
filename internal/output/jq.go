package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

func ApplyJQ(payload any, expr string) ([]any, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid --jq expression: %w", err)
	}

	normalized, err := normalizePayload(payload)
	if err != nil {
		return nil, err
	}

	iter := query.Run(normalized)
	var results []any
	for {
		value, ok := iter.Next()
		if !ok {
			return results, nil
		}
		if evalErr, ok := value.(error); ok {
			return nil, fmt.Errorf("--jq evaluation error: %w", evalErr)
		}
		results = append(results, value)
	}
}

func WriteJQ(w io.Writer, payload any, expr string, pretty bool) error {
	results, err := ApplyJQ(payload, expr)
	if err != nil {
		return err
	}

	for _, value := range results {
		switch typed := value.(type) {
		case string:
			if _, err := fmt.Fprintln(w, typed); err != nil {
				return err
			}
		default:
			if err := WriteJSON(w, typed, pretty); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizePayload(payload any) (any, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload for jq: %w", err)
	}
	data = bytes.TrimSpace(data)

	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, fmt.Errorf("normalize payload for jq: %w", err)
	}
	return normalized, nil
}
