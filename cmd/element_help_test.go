package cmd

import (
	"strings"
	"testing"
)

func TestElementHelpListsSubcommands(t *testing.T) {
	out := runHelpForTest(t, []string{"element", "--help"})

	for _, name := range []string{"ls", "get", "save", "apply", "build", "search", "knowledge"} {
		if !strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to contain %q, got: %s", name, out)
		}
	}
	for _, name := range []string{"list", "info", "exec", "remove"} {
		if strings.Contains(out, "\n  "+name+" ") {
			t.Fatalf("expected help to exclude %q, got: %s", name, out)
		}
	}
}

func TestElementSaveHelpRendersStructuredDescription(t *testing.T) {
	out := runHelpForTest(t, []string{"element", "save", "--help"})
	if !strings.Contains(out, "副作用：") || !strings.Contains(out, "如果它不适合：") {
		t.Fatalf("expected save help to contain structured sections, got: %s", out)
	}
	if !strings.Contains(out, "示例:") || !strings.Contains(out, "jit element save services.DemoSvc") {
		t.Fatalf("expected save help examples to be rendered, got: %s", out)
	}
}

func TestElementHelpUsesClearCommandDescriptions(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"element", "build", "--help"}, "构建一个或多个指定 fullName 的元素"},
		{[]string{"element", "search", "--help"}, "位置参数是正则表达式 pattern。"},
		{[]string{"element", "knowledge", "--help"}, "后端优先调用元素的 `knowledges()`"},
	}

	for _, tc := range cases {
		out := runHelpForTest(t, tc.args)
		if !strings.Contains(out, tc.want) {
			t.Fatalf("expected help %v to contain %q, got: %s", tc.args, tc.want, out)
		}
	}
}
