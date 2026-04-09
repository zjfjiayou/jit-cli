package cmd

import (
	"strings"
	"testing"
)

func TestAppHelpListsSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"app", "--help"})

	for _, name := range []string{"refresh", "get", "ls"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	for _, name := range []string{"info", "elements", "functions"} {
		if strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to exclude %q, got: %s", name, out)
		}
	}
}

func TestAppRefreshHelpRendersStructuredDescription(t *testing.T) {
	out := runHelpForTest(t, []string{"app", "refresh", "--help"})
	if !strings.Contains(out, "副作用：") || !strings.Contains(out, "数据来源：") {
		t.Fatalf("expected refresh help to contain structured sections, got: %s", out)
	}
	if !strings.Contains(out, "示例:") || !strings.Contains(out, "jit app refresh --dry-run") {
		t.Fatalf("expected refresh help examples to be rendered, got: %s", out)
	}
}
