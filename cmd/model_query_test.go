package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestModelQueryUsesClassMethodPayload(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
		resolveAppFn: func(_ context.Context, _, _ string) (string, error) {
			return "whwy/mmm", nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "query", "models.Customer",
		"--filter", `Q("name", "=", "Alice")`,
		"--page", "2",
		"--size", "5",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != "models/Customer/query" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	var body map[string]any
	if err := json.Unmarshal(got.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["methodType"] != "cls" {
		t.Fatalf("methodType = %#v, want cls", body["methodType"])
	}

	argDict, ok := body["argDict"].(map[string]any)
	if !ok {
		t.Fatalf("argDict missing: %#v", body)
	}
	if argDict["filter"] != `Q("name", "=", "Alice")` {
		t.Fatalf("filter = %#v, want raw Q expression", argDict["filter"])
	}
	if argDict["page"] != float64(2) || argDict["size"] != float64(5) {
		t.Fatalf("unexpected pagination payload: %#v", argDict)
	}
}

func TestModelQueryDefaultsToNilFilter(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "query", "models.Customer",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	var body map[string]any
	if err := json.Unmarshal(got.Body, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	argDict, ok := body["argDict"].(map[string]any)
	if !ok {
		t.Fatalf("argDict missing: %#v", body)
	}
	if value, exists := argDict["filter"]; !exists || value != nil {
		t.Fatalf("filter = %#v, want nil", argDict["filter"])
	}
}
