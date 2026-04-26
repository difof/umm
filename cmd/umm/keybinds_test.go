package umm

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeybindsCommandMatchesHelpOutput(t *testing.T) {
	cmd := BuildKeybindsCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	runText := stdout.String()

	cmd = BuildKeybindsCmd()
	stdout.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute --help returned error: %v", err)
	}
	helpText := stdout.String()

	if runText != helpText {
		t.Fatalf("expected identical output for run and help\nrun:\n%s\nhelp:\n%s", runText, helpText)
	}

	checks := []string{
		"Keybinds Reference",
		"Current Normal Keymap",
		"Current Git Keymap",
		"Semantics",
		"Template Variables",
		"change:reload:sleep 0.05; {{.ReloadCommand}}",
		"ctrl-/:toggle-preview",
		"expect-keys: ctrl-o",
	}
	for _, check := range checks {
		if !strings.Contains(runText, check) {
			t.Fatalf("expected keybinds output to contain %q, got %q", check, runText)
		}
	}
}

func TestKeybindsCommandShowsConfiguredOverrides(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "keybinds:\n  normal:\n    bind:\n      - 'ctrl-y:execute-silent(echo {} | pbcopy)'\n  git:\n    expect-keys:\n      - alt-enter\n    bind:\n      - 'ctrl-p:toggle-preview'\n")

	cmd := BuildKeybindsCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	text := stdout.String()
	checks := []string{
		"Source: " + filepath.Join(xdg, "umm", "umm.yml"),
		"ctrl-y:execute-silent(echo {} | pbcopy)",
		"expect-keys: alt-enter",
		"ctrl-p:toggle-preview",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected configured keybinds output to contain %q, got %q", check, text)
		}
	}
	normalSection := sectionText(text, "Current Normal Keymap", "Current Git Keymap")
	if strings.Contains(normalSection, "change:reload:sleep 0.05; {{.ReloadCommand}}") {
		t.Fatalf("expected custom normal bind list to replace defaults in current section, got %q", normalSection)
	}
	gitSection := sectionText(text, "Current Git Keymap", "Semantics")
	if strings.Contains(gitSection, "ctrl-/:toggle-preview") {
		t.Fatalf("expected custom git bind list to replace defaults in current section, got %q", gitSection)
	}
}

func sectionText(text string, start string, end string) string {
	startIndex := strings.Index(text, start)
	if startIndex == -1 {
		return text
	}
	section := text[startIndex:]
	endIndex := strings.Index(section, end)
	if endIndex == -1 {
		return section
	}
	return section[:endIndex]
}
