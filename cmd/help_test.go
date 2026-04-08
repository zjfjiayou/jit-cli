package cmd

import (
	"strings"
	"testing"
)

func TestRootHelpFlag(t *testing.T) {
	out := runHelpForTest(t, []string{"--help"})
	if !strings.Contains(out, "用法:") || !strings.Contains(out, "可用命令:") {
		t.Fatalf("unexpected help output: %q", out)
	}
}
