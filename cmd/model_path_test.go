package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestModelListUsesDerivedJitORMApp(t *testing.T) {
	var resolveCalls int
	var got APIRequest

	rt := mockRuntime{
		resolveAppFn: func(_ context.Context, profile string, appOverride string) (string, error) {
			resolveCalls++
			if appOverride != "" {
				return appOverride, nil
			}
			return "wanyun/JitAi", nil
		},
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{
				Raw: json.RawMessage(`{"errcode":0,"data":[]}`),
			}, nil
		},
	}

	code, _, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"model", "list",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, errOut)
	}
	if resolveCalls == 0 {
		t.Fatalf("expected ResolveApp to be called")
	}
	if got.App != "wanyun/JitORM" {
		t.Fatalf("expected app wanyun/JitORM, got %s", got.App)
	}
	if got.Endpoint != modelSvcGetModelList {
		t.Fatalf("expected endpoint %s, got %s", modelSvcGetModelList, got.Endpoint)
	}
}
