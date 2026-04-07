package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootHelpFlag(t *testing.T) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		os.Stdout = originalStdout
	}()
	os.Stdout = writer

	root := NewRootCmd(NewDefaultFactory())
	root.SetArgs([]string{"--help"})
	execErr := root.Execute()
	_ = writer.Close()
	os.Stdout = originalStdout
	if execErr != nil {
		t.Fatalf("Execute() error = %v", execErr)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "Available Commands:") {
		t.Fatalf("unexpected help output: %q", out)
	}
}
