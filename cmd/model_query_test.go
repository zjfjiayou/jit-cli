package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func requireJSONMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return body
}

func requireClassMethodArgDict(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	body := requireJSONMap(t, raw)
	if body["methodType"] != "cls" {
		t.Fatalf("methodType = %#v, want cls", body["methodType"])
	}

	argDict, ok := body["argDict"].(map[string]any)
	if !ok {
		t.Fatalf("argDict missing: %#v", body)
	}
	return argDict
}

func assertValidationFailure(t *testing.T, args []string, want string) {
	t.Helper()

	apiCalled := false
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			apiCalled = true
			return APIResponse{}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, args, "", rt)
	if code != ExitCLIError {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitCLIError, code, errOut)
	}
	if apiCalled {
		t.Fatalf("expected validation failure before API call")
	}
	if !strings.Contains(errOut, want) {
		t.Fatalf("stderr = %s, want substring %s", errOut, want)
	}
}

func TestModelQueryUsesAIQueryPayload(t *testing.T) {
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
		"--fields", `["id","name"]`,
		"--order", `[["id",-1]]`,
		"--page", "2",
		"--size", "5",
		"--level", "3",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != "models/Customer/aiQuery" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	argDict := requireClassMethodArgDict(t, got.Body)
	if argDict["qfilter"] != `Q("name", "=", "Alice")` {
		t.Fatalf("qfilter = %#v, want raw Q expression", argDict["qfilter"])
	}
	if argDict["page"] != float64(2) || argDict["size"] != float64(5) || argDict["level"] != float64(3) {
		t.Fatalf("unexpected pagination payload: %#v", argDict)
	}

	fieldList, ok := argDict["fieldList"].([]any)
	if !ok || len(fieldList) != 2 || fieldList[0] != "id" || fieldList[1] != "name" {
		t.Fatalf("fieldList = %#v, want [id name]", argDict["fieldList"])
	}

	orderList, ok := argDict["orderList"].([]any)
	if !ok || len(orderList) != 1 {
		t.Fatalf("orderList = %#v, want one order item", argDict["orderList"])
	}
	firstOrder, ok := orderList[0].([]any)
	if !ok || len(firstOrder) != 2 || firstOrder[0] != "id" || firstOrder[1] != float64(-1) {
		t.Fatalf("orderList[0] = %#v, want [id -1]", orderList[0])
	}
}

func TestModelQueryDefaultsOptionalArgsToNilAndDefaults(t *testing.T) {
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

	argDict := requireClassMethodArgDict(t, got.Body)

	for _, key := range []string{"qfilter", "fieldList", "orderList"} {
		if value, exists := argDict[key]; !exists || value != nil {
			t.Fatalf("%s = %#v, want nil", key, argDict[key])
		}
	}
	if argDict["page"] != float64(1) || argDict["size"] != float64(20) || argDict["level"] != float64(2) {
		t.Fatalf("unexpected defaults: %#v", argDict)
	}
}

func TestModelAnalyzeUsesAISelectEndpoint(t *testing.T) {
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
		"model", "analyze", `Select([F("id")], From(["models.Customer"]), Limit(0, 10))`,
		"--limit", "20",
		"--offset", "5",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != modelSvcAISelect {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	body := requireJSONMap(t, got.Body)
	if body["tql"] != `Select([F("id")], From(["models.Customer"]), Limit(0, 10))` {
		t.Fatalf("tql = %#v, want raw expr", body["tql"])
	}
	if body["limit"] != float64(20) || body["offset"] != float64(5) {
		t.Fatalf("unexpected pagination payload: %#v", body)
	}
}

func TestModelCreateUsesAICreatePayload(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "create", "models.Customer",
		"--data", `{"name":"Alice"}`,
		"--trigger-event", "0",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != "models/Customer/aiCreate" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	argDict := requireClassMethodArgDict(t, got.Body)
	data, ok := argDict["data"].(map[string]any)
	if !ok || data["name"] != "Alice" {
		t.Fatalf("data = %#v, want object with name=Alice", argDict["data"])
	}
	if argDict["triggerEvent"] != float64(0) {
		t.Fatalf("triggerEvent = %#v, want 0", argDict["triggerEvent"])
	}
}

func TestModelUpdateUsesAIUpdatePayload(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "update", "models.Customer",
		"--filter", `Q("id","=",1)`,
		"--data", `{"name":"Bob"}`,
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != "models/Customer/aiUpdate" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	argDict := requireClassMethodArgDict(t, got.Body)
	if argDict["qfilter"] != `Q("id","=",1)` {
		t.Fatalf("qfilter = %#v, want raw Q expression", argDict["qfilter"])
	}
	updateData, ok := argDict["updateData"].(map[string]any)
	if !ok || updateData["name"] != "Bob" {
		t.Fatalf("updateData = %#v, want object with name=Bob", argDict["updateData"])
	}
	if argDict["triggerEvent"] != float64(1) {
		t.Fatalf("triggerEvent = %#v, want 1", argDict["triggerEvent"])
	}
}

func TestModelDeleteUsesAIDeletePayload(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{Raw: json.RawMessage(`{"errcode":0}`)}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "delete", "models.Customer",
		"--filter", `Q("id","=",1)`,
		"--trigger-event", "0",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}

	if got.Endpoint != "models/Customer/aiDelete" {
		t.Fatalf("unexpected endpoint: %+v", got)
	}

	argDict := requireClassMethodArgDict(t, got.Body)
	if argDict["qfilter"] != `Q("id","=",1)` {
		t.Fatalf("qfilter = %#v, want raw Q expression", argDict["qfilter"])
	}
	if argDict["triggerEvent"] != float64(0) {
		t.Fatalf("triggerEvent = %#v, want 0", argDict["triggerEvent"])
	}
}

func TestModelWriteCommandsValidateRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "create missing data",
			args: []string{"--profile", "demo", "model", "create", "models.Customer"},
			want: `"error":"missing_data"`,
		},
		{
			name: "update missing filter",
			args: []string{"--profile", "demo", "model", "update", "models.Customer", "--data", `{"name":"Bob"}`},
			want: `"error":"missing_filter"`,
		},
		{
			name: "update missing data",
			args: []string{"--profile", "demo", "model", "update", "models.Customer", "--filter", `Q("id","=",1)`},
			want: `"error":"missing_data"`,
		},
		{
			name: "delete missing filter",
			args: []string{"--profile", "demo", "model", "delete", "models.Customer"},
			want: `"error":"missing_filter"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationFailure(t, tt.args, tt.want)
		})
	}
}

func TestModelCommandsValidateJSONShape(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "query fields must be array",
			args: []string{"--profile", "demo", "model", "query", "models.Customer", "--fields", `{"id":true}`},
			want: `"error":"invalid_fields"`,
		},
		{
			name: "query order must be array",
			args: []string{"--profile", "demo", "model", "query", "models.Customer", "--order", `{"id":-1}`},
			want: `"error":"invalid_order"`,
		},
		{
			name: "create data must be object",
			args: []string{"--profile", "demo", "model", "create", "models.Customer", "--data", `[]`},
			want: `"error":"invalid_data"`,
		},
		{
			name: "update data must be object",
			args: []string{"--profile", "demo", "model", "update", "models.Customer", "--filter", `Q("id","=",1)`, "--data", `[]`},
			want: `"error":"invalid_data"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidationFailure(t, tt.args, tt.want)
		})
	}
}
