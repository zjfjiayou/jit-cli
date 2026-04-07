package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAPICommandDryRun(t *testing.T) {
	var got APIRequest
	rt := mockRuntime{
		callAPIFn: func(_ context.Context, req APIRequest) (APIResponse, error) {
			got = req
			return APIResponse{
				Raw: json.RawMessage(`{"dryRun":true,"url":"http://127.0.0.1:8080/api/wanyun/JitAi/services/JitAISvc/sendMessage"}`),
			}, nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"--app", "wanyun/JitAi",
		"--dry-run",
		"api", "services/JitAISvc/sendMessage",
		"--data", `{"message":"hello"}`,
	}, "", rt)

	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, errOut)
	}
	if !got.DryRun {
		t.Fatalf("expected dryRun request")
	}
	if got.Endpoint != "services/JitAISvc/sendMessage" || got.App != "wanyun/JitAi" {
		t.Fatalf("unexpected request: %+v", got)
	}
	if !strings.Contains(out, `"dryRun":true`) {
		t.Fatalf("expected dry run payload in stdout, got: %s", out)
	}
}
