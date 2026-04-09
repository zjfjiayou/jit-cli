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
	if !strings.Contains(out, "适合 AI Agent、脚本和排障场景") {
		t.Fatalf("expected root long help to be rendered, got: %q", out)
	}
	if !strings.Contains(out, "术语说明：") || !strings.Contains(out, "示例:") {
		t.Fatalf("expected structured sections and examples, got: %q", out)
	}
}

func TestAuthHelpRendersLongDescription(t *testing.T) {
	out := runHelpForTest(t, []string{"auth", "--help"})
	if !strings.Contains(out, "profile 会保存 server、default_app 和本地 token 存储位置") {
		t.Fatalf("expected auth long help to be rendered, got: %q", out)
	}
	if !strings.Contains(out, "默认上下文：") || !strings.Contains(out, "如果它不适合：") {
		t.Fatalf("expected auth structured sections to be rendered, got: %q", out)
	}
}
