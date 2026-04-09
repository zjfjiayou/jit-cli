package cmd

import (
	"strings"
)

func helpText(lines ...string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func helpSection(title string, lines ...string) string {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return ""
	}

	block := []string{trimmedTitle + "："}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		block = append(block, "  "+trimmed)
	}
	return strings.Join(block, "\n")
}

func helpSections(sections ...string) string {
	blocks := make([]string, 0, len(sections))
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		blocks = append(blocks, trimmed)
	}
	return strings.Join(blocks, "\n\n")
}

type helpExample struct {
	Description string
	Command     string
}

func helpExamples(items ...helpExample) string {
	lines := make([]string, 0, len(items)*2)
	for _, item := range items {
		description := strings.TrimSpace(item.Description)
		command := strings.TrimSpace(item.Command)
		if description != "" {
			lines = append(lines, "  "+description)
		}
		if command != "" {
			lines = append(lines, "    "+command)
		}
	}
	return strings.Join(lines, "\n")
}
