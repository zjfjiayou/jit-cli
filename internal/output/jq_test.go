package output

import (
	"bytes"
	"testing"
)

func TestApplyJQ(t *testing.T) {
	payload := map[string]any{
		"errcode": 0,
		"data": map[string]any{
			"user": map[string]any{
				"name": "alice",
			},
		},
	}

	results, err := ApplyJQ(payload, ".data.user.name")
	if err != nil {
		t.Fatalf("ApplyJQ() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0] != "alice" {
		t.Fatalf("result = %v, want alice", results[0])
	}
}

func TestWriteJQ(t *testing.T) {
	payload := map[string]any{
		"items": []map[string]any{
			{"id": 1},
			{"id": 2},
		},
	}

	var buf bytes.Buffer
	err := WriteJQ(&buf, payload, ".items[].id", false)
	if err != nil {
		t.Fatalf("WriteJQ() error = %v", err)
	}

	got := buf.String()
	if got != "1\n2\n" {
		t.Fatalf("WriteJQ() output = %q, want %q", got, "1\n2\n")
	}
}

func TestApplyJQSupportsPipeSelectAndIndex(t *testing.T) {
	payload := map[string]any{
		"items": []map[string]any{
			{"id": 1},
			{"id": 2},
		},
	}

	results, err := ApplyJQ(payload, `.items | map(select(.id > 1)) | .[0].id`)
	if err != nil {
		t.Fatalf("ApplyJQ() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	got, ok := results[0].(float64)
	if !ok {
		t.Fatalf("result type = %T, want float64", results[0])
	}
	if got != 2 {
		t.Fatalf("result = %v, want 2", got)
	}
}
