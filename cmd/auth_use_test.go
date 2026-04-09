package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestAuthUseOutputsResolvedProfileName(t *testing.T) {
	var gotSelector string
	rt := mockRuntime{
		authUseFn: func(_ context.Context, profile string) (string, error) {
			gotSelector = profile
			return "demo", nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{"auth", "use", "0"}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if gotSelector != "0" {
		t.Fatalf("AuthUse() selector = %q, want 0", gotSelector)
	}
	if !strings.Contains(out, `"current_profile":"demo"`) {
		t.Fatalf("unexpected stdout: %s", out)
	}
}
