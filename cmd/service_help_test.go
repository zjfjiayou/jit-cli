package cmd

import (
	"strings"
	"testing"
)

func TestServiceHelpListsSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"service", "--help"})

	for _, name := range []string{"ls", "call"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	for _, name := range []string{"list", "exec"} {
		if strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to exclude %q, got: %s", name, out)
		}
	}
}

func TestServiceCallHelpRendersStructuredDescription(t *testing.T) {
	out := runHelpForTest(t, []string{"service", "call", "--help"})
	if !strings.Contains(out, "校验行为：") || !strings.Contains(out, "如果它不适合：") {
		t.Fatalf("expected call help to contain structured sections, got: %s", out)
	}
	if !strings.Contains(out, "示例:") || !strings.Contains(out, "jit service call corps.services.MemberSvc getCurrUserInfo") {
		t.Fatalf("expected call help examples to be rendered, got: %s", out)
	}
}
