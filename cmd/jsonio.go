package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func writeJSON(out io.Writer, value any) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func writeRawJSON(out io.Writer, raw json.RawMessage) error {
	if len(raw) == 0 {
		_, err := out.Write([]byte("null\n"))
		return err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		_, err := out.Write([]byte("null\n"))
		return err
	}
	if _, err := out.Write(trimmed); err != nil {
		return err
	}
	_, err := out.Write([]byte("\n"))
	return err
}

func rawPayload(resp map[string]any) json.RawMessage {
	if resp == nil {
		return nil
	}
	raw, _ := resp["raw"].(json.RawMessage)
	return raw
}

func writeResponsePayload(out io.Writer, resp map[string]any) error {
	if raw := rawPayload(resp); len(raw) > 0 {
		return writeRawJSON(out, raw)
	}
	return writeJSON(out, resp)
}

func writeRawPayload(out io.Writer, resp map[string]any) error {
	return writeRawJSON(out, rawPayload(resp))
}

func parseJSONArg(dataArg string, in io.Reader) (json.RawMessage, error) {
	source := strings.TrimSpace(dataArg)
	switch source {
	case "":
		source = "{}"
	case "@-":
		body, err := io.ReadAll(in)
		if err != nil {
			return nil, fmt.Errorf("read stdin failed: %w", err)
		}
		source = strings.TrimSpace(string(body))
		if source == "" {
			source = "{}"
		}
	}
	if !json.Valid([]byte(source)) {
		return nil, fmt.Errorf("invalid json payload")
	}
	return json.RawMessage(source), nil
}

func parseJSONValue(dataArg string) (any, error) {
	raw := strings.TrimSpace(dataArg)
	if raw == "" {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseErrCode(raw json.RawMessage) (int, bool) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, false
	}
	value, ok := payload["errcode"]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case json.Number:
		i, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(typed)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func readTokenFromStdin(in io.Reader) (string, error) {
	tokenRaw, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(tokenRaw)), nil
}
