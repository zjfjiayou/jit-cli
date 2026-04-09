package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestAuthRmCallsRuntime(t *testing.T) {
	var gotProfile string
	rt := mockRuntime{
		authRemoveFn: func(_ context.Context, profile string) error {
			gotProfile = profile
			return nil
		},
	}

	code, out, errOut := runCmdForTest(t, []string{"auth", "rm", "demo"}, "", rt)
	if code != ExitOK {
		t.Fatalf("expected exit %d, got %d, stderr=%s", ExitOK, code, errOut)
	}
	if gotProfile != "demo" {
		t.Fatalf("AuthRemove() profile = %q, want demo", gotProfile)
	}
	if !strings.Contains(out, `"profile":"demo"`) || !strings.Contains(out, `"ok":true`) {
		t.Fatalf("unexpected stdout: %s", out)
	}
}

func TestAuthHelpListsRmSubcommand(t *testing.T) {
	out := runHelpForTest(t, []string{"auth", "--help"})
	for _, name := range []string{"login", "whoami", "logout", "rm", "ls", "use"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	for _, name := range []string{"status", "list"} {
		if strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to exclude %q, got: %s", name, out)
		}
	}
}
