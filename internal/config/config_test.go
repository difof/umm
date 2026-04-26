package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ummtheme "github.com/difof/umm/internal/theme"
)

func TestResolveWritePathPrefersXDG(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)

	path, err := ResolveWritePath()
	if err != nil {
		t.Fatalf("ResolveWritePath returned error: %v", err)
	}
	want := filepath.Join(xdg, "umm", "umm.yml")
	if path != want {
		t.Fatalf("ResolveWritePath() = %q, want %q", path, want)
	}
}

func TestResolveConfigDirPrefersXDG(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)

	path, err := ResolveConfigDir()
	if err != nil {
		t.Fatalf("ResolveConfigDir returned error: %v", err)
	}
	want := filepath.Join(xdg, "umm")
	if path != want {
		t.Fatalf("ResolveConfigDir() = %q, want %q", path, want)
	}
}

func TestFindUserPathPrefersExistingXDG(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)

	xdgPath := filepath.Join(xdg, "umm", "umm.yml")
	homePath := filepath.Join(home, ".config", "umm", "umm.yml")
	writeConfigFile(t, xdgPath, "git:\n  default-modes: [tracked]\n")
	writeConfigFile(t, homePath, "git:\n  default-modes: [commit]\n")

	path, exists, err := FindUserPath()
	if err != nil {
		t.Fatalf("FindUserPath returned error: %v", err)
	}
	if !exists || path != xdgPath {
		t.Fatalf("FindUserPath() = (%q, %v), want (%q, true)", path, exists, xdgPath)
	}
}

func TestFindUserPathFallsBackToHomeConfig(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)

	homePath := filepath.Join(home, ".config", "umm", "umm.yml")
	writeConfigFile(t, homePath, "git:\n  default-modes: [commit]\n")

	path, exists, err := FindUserPath()
	if err != nil {
		t.Fatalf("FindUserPath returned error: %v", err)
	}
	if !exists || path != homePath {
		t.Fatalf("FindUserPath() = (%q, %v), want (%q, true)", path, exists, homePath)
	}
}

func TestLoadEffectiveDefaultsWhenMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	loaded, err := LoadEffective()
	if err != nil {
		t.Fatalf("LoadEffective returned error: %v", err)
	}
	if loaded.UserExists {
		t.Fatal("expected no user config")
	}
	if got := strings.Join(loaded.Config.Git.DefaultModes, ","); got != strings.Join(AllGitModes, ",") {
		t.Fatalf("default git modes = %q, want %q", got, strings.Join(AllGitModes, ","))
	}
	if loaded.Config.Theme != ummtheme.DefaultName {
		t.Fatalf("default theme = %q, want %q", loaded.Config.Theme, ummtheme.DefaultName)
	}
}

func TestLoadEffectiveMergesPartialConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	path := filepath.Join(xdg, "umm", "umm.yml")
	writeConfigFile(t, path, "git:\n  default-modes:\n    - tracked\npreview:\n  file:\n    cmd: bat\n    args:\n      - --line-range\n      - '{{.LineRange}}'\n      - '{{.Path}}'\n")

	loaded, err := LoadEffective()
	if err != nil {
		t.Fatalf("LoadEffective returned error: %v", err)
	}
	if !loaded.UserExists {
		t.Fatal("expected user config to exist")
	}
	if got := strings.Join(loaded.Config.Git.DefaultModes, ","); got != "tracked" {
		t.Fatalf("merged git modes = %q, want %q", got, "tracked")
	}
	if loaded.Config.Theme != ummtheme.DefaultName {
		t.Fatalf("default theme should be preserved, got %q", loaded.Config.Theme)
	}
	if loaded.Config.Git.Limits.PreviewBranchCommits != 10 {
		t.Fatalf("expected default preview branch commits to be preserved, got %d", loaded.Config.Git.Limits.PreviewBranchCommits)
	}
	if loaded.Config.Preview.File.Cmd != "bat" {
		t.Fatalf("preview.file.cmd = %q, want bat", loaded.Config.Preview.File.Cmd)
	}
	if len(loaded.Config.Keybinds.Normal.Bind) == 0 {
		t.Fatal("expected default keybinds to be preserved")
	}
}

func TestLoadEffectiveRejectsUnknownField(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	writeConfigFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  bad-field: true\n")
	if _, err := LoadEffective(); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestLoadEffectiveRejectsInvalidYAML(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())

	writeConfigFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git: [\n")
	if _, err := LoadEffective(); err == nil {
		t.Fatal("expected invalid-yaml error")
	}
}

func TestValidateRejectsBadTemplateField(t *testing.T) {
	cfg := Defaults()
	cfg.Preview.File = Command{Cmd: "bat", Args: []string{"{{.Nope}}"}}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected template validation error")
	}
}

func TestCheckReportsWarningsAndKeybindErrors(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	shimDir := t.TempDir()
	t.Setenv("PATH", shimDir)
	writeExecutable(t, filepath.Join(shimDir, "fzf"), "#!/bin/sh\nprev=''\nfor arg in \"$@\"; do\n  if [ \"$prev\" = \"--bind\" ]; then\n    case \"$arg\" in\n      *BROKEN*)\n        printf 'invalid bind\\n' >&2\n        exit 1\n        ;;\n    esac\n  fi\n  prev=\"$arg\"\ndone\nexit 0\n")

	writeConfigFile(t, filepath.Join(xdg, "umm", "umm.yml"), "keybinds:\n  normal:\n    bind:\n      - 'BROKEN'\npreview:\n  file:\n    cmd: missing-preview\n    args:\n      - '{{.Path}}'\n")

	report, err := Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if report.Valid() {
		t.Fatal("expected invalid report")
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "missing-preview") {
		t.Fatalf("expected missing-command warning, got %#v", report.Warnings)
	}
	if len(report.Errors) == 0 || !strings.Contains(strings.Join(report.Errors, "\n"), "invalid bind") {
		t.Fatalf("expected keybind parser error, got %#v", report.Errors)
	}
}

func TestCheckMissingConfigSucceeds(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	report, err := Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !report.Valid() || report.UserExists {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestMarshalUsesHyphenatedKeys(t *testing.T) {
	data, err := Marshal(Defaults())
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "default-modes:") || !strings.Contains(text, "expect-keys:") {
		t.Fatalf("expected hyphenated keys in YAML, got %q", text)
	}
	if strings.Contains(text, "default_modes") || strings.Contains(text, "expect_keys") {
		t.Fatalf("expected no underscore keys in YAML, got %q", text)
	}
}

func TestValidateRejectsEmptyTheme(t *testing.T) {
	cfg := Defaults()
	cfg.Theme = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected theme validation error")
	}
}

func TestCheckReportsMissingThemeAndInvalidUserTheme(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeConfigFile(t, filepath.Join(xdg, "umm", "umm.yml"), "theme: missing-theme\n")
	writeConfigFile(t, filepath.Join(xdg, "umm", "themes", "broken.yml"), "name: broken\nvariant: nope\n")

	report, err := Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	text := strings.Join(report.Errors, "\n")
	if !strings.Contains(text, "missing-theme") || !strings.Contains(text, "broken.yml") {
		t.Fatalf("expected theme errors, got %#v", report.Errors)
	}
}

func TestUpdateThemeBytesPreservesComments(t *testing.T) {
	updated, err := updateThemeBytes([]byte("# top\ngit:\n  # keep\n  default-modes:\n    - tracked\n"), "lattice-light")
	if err != nil {
		t.Fatalf("updateThemeBytes returned error: %v", err)
	}
	text := string(updated)
	if !strings.Contains(text, "# top") || !strings.Contains(text, "# keep") {
		t.Fatalf("expected comments to survive, got %q", text)
	}
	if !strings.Contains(text, "theme: lattice-light") {
		t.Fatalf("expected inserted theme, got %q", text)
	}
}

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
