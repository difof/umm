package app

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/resultfmt"
)

func TestBuildGitHeader(t *testing.T) {
	t.Run("empty modes shows all", func(t *testing.T) {
		got := buildGitHeader(nil)
		want := "Git modes: all"
		if got != want {
			t.Fatalf("buildGitHeader() = %q, want %q", got, want)
		}
	})

	t.Run("joins selected modes", func(t *testing.T) {
		got := buildGitHeader([]string{"commit", "branch", "tracked"})
		want := "Git modes: commit, branch, tracked"
		if got != want {
			t.Fatalf("buildGitHeader() = %q, want %q", got, want)
		}
	})
}

func TestBuildEmitArgs(t *testing.T) {
	cfg := cli.RootConfig{
		Root:         "/tmp/project",
		Excludes:     []string{"*.tmp", "vendor/**"},
		Hidden:       true,
		NoFilename:   true,
		OnlyDirname:  true,
		OnlyFilename: false,
		MaxDepth:     3,
		Pattern:      "needle",
	}

	got := buildEmitArgs(cfg, false)
	want := []string{
		"__emit-search",
		"--pattern", "needle",
		"--root", "/tmp/project",
		"--exclude", "*.tmp",
		"--exclude", "vendor/**",
		"--hidden",
		"--no-filename",
		"--only-dirname",
		"--max-depth", "3",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("buildEmitArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildBindArgsRendersTemplates(t *testing.T) {
	args, err := buildBindArgs([]string{"change:reload:sleep 0.05; {{.ReloadCommand}}", "ctrl-o:execute({{.PreviewCommand}})"}, ummconfig.KeybindTemplateData{
		ReloadCommand:  "umm __emit-search --pattern {q}",
		PreviewCommand: "umm preview {1} {2}",
	})
	if err != nil {
		t.Fatalf("buildBindArgs returned error: %v", err)
	}
	want := []string{"--bind", "change:reload:sleep 0.05; umm __emit-search --pattern {q}", "--bind", "ctrl-o:execute(umm preview {1} {2})"}
	if !slices.Equal(args, want) {
		t.Fatalf("buildBindArgs() = %#v, want %#v", args, want)
	}
}

func TestRunGitNoUISystemRequiresTrackedFiles(t *testing.T) {
	err := runGitNoUI(t.Context(), cli.RootConfig{Action: cli.ActionSystem}, ummconfig.Defaults(), []resultfmt.Result{{GitType: "commit", GitRef: "abc"}})
	if err == nil || !strings.Contains(err.Error(), "no tracked file results available") {
		t.Fatalf("runGitNoUI() error = %v, want tracked-file error", err)
	}
}

func TestRunNormalNoUIAskRoutesToStat(t *testing.T) {
	t.Setenv("UMM_TEST_OPEN_ASK_CHOICE", "stat")
	root := t.TempDir()
	first := writeFile(t, root, "a.txt", "one\n")
	second := writeFile(t, root, "nested/b.txt", "two\n")

	results := []resultfmt.Result{{Path: first}, {Path: second}}
	output := captureStdout(t, func() {
		if err := runNormalNoUI(t.Context(), cli.RootConfig{Action: cli.ActionAsk}, ummconfig.Defaults(), results); err != nil {
			t.Fatalf("runNormalNoUI() returned error: %v", err)
		}
	})

	if !strings.Contains(output, first) || !strings.Contains(output, second) {
		t.Fatalf("expected stat output for both selections, got %q", output)
	}
}

func TestRunNormalNoUISystemUsesFirstResult(t *testing.T) {
	root := t.TempDir()
	first := writeFile(t, root, "a.txt", "one\n")
	second := writeFile(t, root, "b.txt", "two\n")
	shimDir := t.TempDir()
	logPath := filepath.Join(shimDir, "open.log")
	writeExecutable(t, filepath.Join(shimDir, "open"), "#!/bin/sh\nprintf '%s\n' \"$1\" >> \""+logPath+"\"\n")
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	err := runNormalNoUI(t.Context(), cli.RootConfig{Action: cli.ActionSystem}, ummconfig.Defaults(), []resultfmt.Result{{Path: first}, {Path: second}})
	if err != nil {
		t.Fatalf("runNormalNoUI() returned error: %v", err)
	}

	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile log: %v", err)
	}
	lines := strings.Fields(string(logged))
	if len(lines) != 1 || lines[0] != first {
		t.Fatalf("system open paths = %#v, want only %q", lines, first)
	}
}

func TestRunRootNormalNoUIIntegration(t *testing.T) {
	if !deps.Has("rg") {
		t.Skip("rg is required for normal mode integration test")
	}

	root := t.TempDir()
	path := writeFile(t, root, "nested/needle.txt", "alpha\nneedle\nomega\n")
	cfg := cli.RootConfig{
		Root:       root,
		Pattern:    "needle",
		NoUI:       true,
		SearchMode: cli.SearchModeDefault,
		Action:     cli.ActionStat,
		StatMode:   cli.StatModeList,
	}

	output := captureStdout(t, func() {
		if err := RunRoot(t.Context(), cfg, ummconfig.Defaults()); err != nil {
			t.Fatalf("RunRoot() returned error: %v", err)
		}
	})

	if !strings.Contains(output, path) {
		t.Fatalf("expected normal mode output to contain %q, got %q", path, output)
	}
}

func TestRunRootGitNoUIIntegration(t *testing.T) {
	if !deps.Has("git") {
		t.Skip("git is required for git mode integration test")
	}

	root := t.TempDir()
	runGitCommand(t, root, "init")
	runGitCommand(t, root, "config", "user.email", "test@example.com")
	runGitCommand(t, root, "config", "user.name", "Test User")
	path := writeFile(t, root, "tracked.txt", "hello\n")
	runGitCommand(t, root, "add", ".")
	runGitCommand(t, root, "commit", "-m", "initial")

	cfg := cli.RootConfig{
		Root:       root,
		Pattern:    "tracked",
		NoUI:       true,
		SearchMode: cli.SearchModeGit,
		Action:     cli.ActionDefault,
		GitModes:   []string{"tracked"},
	}

	output := captureStdout(t, func() {
		if err := RunRoot(t.Context(), cfg, ummconfig.Defaults()); err != nil {
			t.Fatalf("RunRoot() returned error: %v", err)
		}
	})

	if !strings.Contains(output, path) {
		t.Fatalf("expected git mode output to contain %q, got %q", path, output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = old })

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}
	os.Stdout = old

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close: %v", err)
	}
	return buffer.String()
}

func writeFile(t *testing.T, root string, rel string, content string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	return path
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
