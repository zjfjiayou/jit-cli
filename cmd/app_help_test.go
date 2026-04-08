package cmd

import (
	"strings"
	"testing"
)

func TestAppHelpListsSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"app", "--help"})

	for _, name := range []string{"refresh", "info", "elements"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	if strings.Contains(out, "\n  functions ") {
		t.Fatalf("expected help to exclude %q, got: %s", "functions", out)
	}
}
