package umm

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigShowPrintsEffectiveConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  default-modes:\n    - tracked\n")

	cmd := BuildConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"show"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	text := stdout.String()
	if !strings.Contains(text, "default-modes:") || !strings.Contains(text, "- tracked") {
		t.Fatalf("unexpected show output: %q", text)
	}
}

func TestConfigDumpCreatesDefaults(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	cmd := BuildConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	path := filepath.Join(xdg, "umm", "umm.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "theme: lattice-dark") {
		t.Fatalf("expected explicit theme in dumped config, got %q", text)
	}
	if !strings.Contains(text, "default-modes:") {
		t.Fatalf("expected dumped defaults, got %q", string(data))
	}
	if !strings.Contains(text, "# editors:") || !strings.Contains(text, "# preview:") {
		t.Fatalf("expected commented examples in dumped config, got %q", text)
	}
	if !strings.Contains(text, "umm theme list") || !strings.Contains(text, "themes/") {
		t.Fatalf("expected theme guidance comments in dumped config, got %q", text)
	}
	if !strings.Contains(text, "LineRange") || !strings.Contains(text, "ReloadCommand") || !strings.Contains(text, "toggle-preview") {
		t.Fatalf("expected inline config reference comments, got %q", text)
	}
	if strings.Contains(text, "editors:\n  nvim:") {
		t.Fatalf("expected concise starter config, got %q", text)
	}
}

func TestConfigDumpOverwritesWithForce(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(xdg, "umm", "umm.yml")
	writeTestFile(t, path, "old: true\n")

	cmd := BuildConfigCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "old: true") {
		t.Fatalf("expected file to be overwritten, got %q", string(data))
	}
}

func TestConfigDumpOverwriteCancelPreservesFile(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UMM_TEST_CONFIG_DUMP_CONFIRM", "cancel")
	path := filepath.Join(xdg, "umm", "umm.yml")
	writeTestFile(t, path, "old: true\n")

	cmd := BuildConfigCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(data) != "old: true\n" {
		t.Fatalf("expected file to remain unchanged, got %q", string(data))
	}
}

func TestConfigDumpOverwriteConfirmedReplacesFile(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UMM_TEST_CONFIG_DUMP_CONFIRM", "overwrite")
	path := filepath.Join(xdg, "umm", "umm.yml")
	writeTestFile(t, path, "old: true\n")

	cmd := BuildConfigCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"dump"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "old: true") {
		t.Fatalf("expected overwritten defaults, got %q", string(data))
	}
}

func TestConfigCheckMissingFileSucceeds(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	cmd := BuildConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"check"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No user config file found") {
		t.Fatalf("unexpected check output: %q", stdout.String())
	}
}

func TestConfigCheckInvalidConfigFails(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  default-modes:\n    - nope\n")

	cmd := BuildConfigCmd()
	cmd.SetOut(&bytes.Buffer{})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"check"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid config error")
	}
	if !strings.Contains(stderr.String(), "invalid git.default-modes") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestConfigHelpIncludesSchemaReference(t *testing.T) {
	cmd := BuildConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	text := stdout.String()
	checks := []string{
		"Config File Schema",
		"Minimal Example",
		"theme",
		"git.default-modes",
		"git.limits.preview-branch-commits",
		"keybinds.normal.bind",
		"keybinds.git.expect-keys",
		"editors.<name>.cmd",
		"preview.file.args",
		"Template Variables",
		"Keybind Semantics",
		"  path args: Path, Line, HasLine, StartLine, EndLine, LineRange.",
		"  validation: Run umm config check",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Fatalf("expected help output to contain %q, got %q", check, text)
		}
	}
	if strings.Contains(text, "UMM_THEME") {
		t.Fatalf("expected config appendix to omit UMM_THEME, got %q", text)
	}
}

func TestConfigSubcommandHelpDoesNotIncludeSchemaReference(t *testing.T) {
	cmd := BuildConfigCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"show", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	text := stdout.String()
	if strings.Contains(text, "Config File Schema") {
		t.Fatalf("expected subcommand help without config appendix, got %q", text)
	}
	if !strings.Contains(text, "Show the effective configuration") {
		t.Fatalf("expected show help output, got %q", text)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
