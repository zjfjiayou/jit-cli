package cmd

import (
	"strings"
	"testing"
)

func TestModelHelpListsSimplifiedSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"model", "--help"})

	for _, name := range []string{"list", "info", "select", "query"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	for _, name := range []string{"meta", "create", "update", "delete"} {
		if strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to exclude %q, got: %s", name, out)
		}
	}
}
