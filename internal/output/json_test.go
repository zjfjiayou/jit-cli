package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSONPretty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, map[string]any{"errcode": 0, "data": map[string]any{"name": "alice"}}, true)
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "\n  \"data\"") {
		t.Fatalf("WriteJSON() output is not pretty formatted: %q", out)
	}
}

func TestWriteCLIError(t *testing.T) {
	var buf bytes.Buffer
	err := WriteCLIError(&buf, "profile_not_found", "profile 'demo' does not exist", nil)
	if err != nil {
		t.Fatalf("WriteCLIError() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "\"error\":\"profile_not_found\"") {
		t.Fatalf("unexpected cli error output: %q", out)
	}
}
