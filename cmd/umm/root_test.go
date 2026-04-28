package umm

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootUsesConfigDefaultGitModes(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeRootTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  default-modes:\n    - tracked\n")

	repo := initRepo(t)
	path := filepath.Join(repo, "tracked.txt")

	cmd := BuildRootCmd(repo)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--root", repo, "--git", "--no-ui", "--pattern", "tracked"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), path) {
		t.Fatalf("expected tracked-file output, got %q", stdout.String())
	}
}

func TestRootExplicitGitModeOverridesConfigDefault(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	writeRootTestFile(t, filepath.Join(xdg, "umm", "umm.yml"), "git:\n  default-modes:\n    - tracked\n")

	repo := initRepo(t)
	runRootGit(t, repo, "tag", "v1.0.0")

	cmd := BuildRootCmd(repo)
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--root", repo, "--git", "--git-mode", "tags", "--no-ui", "--pattern", "v1.0.0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "tag:") {
		t.Fatalf("expected tag output, got %q", stdout.String())
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runRootGit(t, repo, "init")
	runRootGit(t, repo, "config", "user.email", "test@example.com")
	runRootGit(t, repo, "config", "user.name", "Test User")
	writeRootTestFile(t, filepath.Join(repo, "tracked.txt"), "hello\n")
	runRootGit(t, repo, "add", ".")
	runRootGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runRootGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeRootTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
