package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthLoginWithTokenFlag(t *testing.T) {
	var got AuthLoginInput
	rt := mockRuntime{
		authLoginFn: func(_ context.Context, input AuthLoginInput) (map[string]any, error) {
			got = input
			return map[string]any{
				"user": map[string]any{"name": "fx"},
			}, nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{
		"auth", "login",
		"--server", "http://127.0.0.1:8080",
		"--app", "wanyun/JitAi",
		"--profile", "demo",
		"--token", "jit_pat_tokenid_secret",
	}, "", rt)

	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, errOut)
	}
	if got.Server != "http://127.0.0.1:8080" || got.App != "wanyun/JitAi" || got.Profile != "demo" || got.Token != "jit_pat_tokenid_secret" {
		t.Fatalf("unexpected login input: %+v", got)
	}
	if !strings.Contains(out, `"user"`) {
		t.Fatalf("expected user json in stdout, got: %s", out)
	}
}

func TestAuthLoginDryRunRendersRawPreview(t *testing.T) {
	var got AuthLoginInput
	rt := mockRuntime{
		authLoginFn: func(_ context.Context, input AuthLoginInput) (map[string]any, error) {
			got = input
			return map[string]any{
				"raw": json.RawMessage(`{"method":"POST","url":"http://127.0.0.1:8080/api/wanyun/JitAuth/corps/services/MemberSvc/getCurrUserInfo"}`),
			}, nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{
		"--dry-run",
		"auth", "login",
		"--server", "http://127.0.0.1:8080",
		"--app", "wanyun/JitAi",
		"--token", "jit_pat_tokenid_secret",
	}, "", rt)

	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d, stderr=%s", code, errOut)
	}
	if !got.DryRun {
		t.Fatalf("expected DryRun to be passed to runtime")
	}
	if !strings.Contains(out, `"method":"POST"`) {
		t.Fatalf("expected raw dry-run preview in stdout, got: %s", out)
	}
}
