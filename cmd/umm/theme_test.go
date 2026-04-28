package umm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeListShowsBuiltinsAndShadowedUserOverride(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: lattice-dark\n")
	writeTestFile(t, filepath.Join(xdg, "umm", "themes", "lattice-dark.yml"), "name: lattice-dark\nvariant: dark\nfzf:\n  info: hidden\n")

	cmd := BuildThemeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := stdout.String()
	if !strings.Contains(text, "name") || !strings.Contains(text, "variant") || !strings.Contains(text, "origin") || !strings.Contains(text, "status") {
		t.Fatalf("expected list header, got %q", text)
	}
	if !strings.Contains(text, "lattice-dark") || !strings.Contains(text, "shadowed") || !strings.Contains(text, "effective") {
		t.Fatalf("expected effective and shadowed rows, got %q", text)
	}
}

func TestThemeListContinuesWhenUserThemeCannotBeRead(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	themesDir := filepath.Join(xdg, "umm", "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(filepath.Join(xdg, "missing.yml"), filepath.Join(themesDir, "broken.yml")); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}

	cmd := BuildThemeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := stdout.String()
	if !strings.Contains(text, "lattice-dark") || !strings.Contains(text, "broken") || !strings.Contains(text, "invalid") {
		t.Fatalf("expected builtins plus invalid unreadable entry, got %q", text)
	}
}

func TestThemeListWarnsAndOmitsActiveWhenConfigLoadFails(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: [\n")

	cmd := BuildThemeCmd()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if strings.Contains(stdout.String(), "active") {
		t.Fatalf("expected no active theme when config load fails, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning: active theme could not be determined") {
		t.Fatalf("expected config-load warning, got %q", stderr.String())
	}
}

func TestThemeSetCreatesStarterConfigWhenMissing(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	cmd := BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"set", "lattice-light"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(xdg, "umm", "umm.yml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "theme: lattice-light") {
		t.Fatalf("expected created config to include selected theme, got %q", string(data))
	}
}

func TestThemeSetPreservesComments(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(xdg, "umm", "umm.yml")
	writeTestFile(t, path, "# user config\ngit:\n  # preferred mode\n  default-modes:\n    - tracked\n")

	cmd := BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"set", "lattice-light"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "# user config") || !strings.Contains(text, "# preferred mode") {
		t.Fatalf("expected comments to survive theme set, got %q", text)
	}
	if !strings.Contains(text, "theme: lattice-light") {
		t.Fatalf("expected updated theme, got %q", text)
	}
}

func TestThemeSetFailsForInvalidExistingConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  default-modes:\n    - nope\n")

	cmd := BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"set", "lattice-light"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestThemeDumpWritesBuiltInTheme(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	cmd := BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump", "lattice-dark"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	path := filepath.Join(xdg, "umm", "themes", "lattice-dark.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "name: lattice-dark") {
		t.Fatalf("expected dumped built-in theme, got %q", string(data))
	}
}

func TestThemeDumpRequiresForceToOverwrite(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(xdg, "umm", "themes", "lattice-dark.yml")
	writeTestFile(t, path, "old\n")

	cmd := BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump", "lattice-dark"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected overwrite error")
	}

	cmd = BuildThemeCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump", "lattice-dark", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("forced dump returned error: %v", err)
	}
}

func TestThemeHelpIncludesSchemaReference(t *testing.T) {
	cmd := BuildThemeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := stdout.String()
	checks := []string{
		"Theme File Schema",
		"Minimal Example",
		"Field Reference",
		"name",
		"  what: Public exact theme name",
		"  values: required; lowercase kebab-case",
		"fzf.preview-border",
		"values: optional enum:",
		"fzf.color.entries",
		"  what: YAML map of individual fzf UI color slots",
		"  values: optional mapping; format is <slot>: <color-spec>",
		"  example: fg: \"#62ff94\"",
		"  slots: main/text: fg, bg",
		"  all keys: alt-bg",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected help output to contain %q, got %q", check, text)
		}
	}
}
