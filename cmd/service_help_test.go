package cmd

import (
	"strings"
	"testing"
)

func TestServiceHelpListsSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"service", "--help"})

	for _, name := range []string{"list", "exec"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
}
