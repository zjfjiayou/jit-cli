package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthWhoamiCallsRuntime(t *testing.T) {
	var got UserInfoInput
	rt := mockRuntime{
		getCurrUserInfoFn: func(_ context.Context, input UserInfoInput) (map[string]any, error) {
			got = input
			return map[string]any{
				"raw": json.RawMessage(`{"data":{"user":{"userId":"admin123"}}}`),
			}, nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{
		"--profile", "demo",
		"--app", "whwy/mmm",
		"auth", "whoami",
	}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if got.Profile != "demo" || got.App != "whwy/mmm" {
		t.Fatalf("unexpected whoami input: %+v", got)
	}
	if !strings.Contains(out, `"userId":"admin123"`) {
		t.Fatalf("unexpected stdout: %s", out)
	}
}
