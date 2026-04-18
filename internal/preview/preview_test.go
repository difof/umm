package preview

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/difof/umm/internal/resultfmt"
)

func TestRunFileAndDirPreview(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir", "child.txt"), []byte("child\n"), 0o644); err != nil {
		t.Fatalf("WriteFile child: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", t.TempDir()); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	fileMeta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: file, Line: 2})
	if err != nil {
		t.Fatalf("EncodeMeta file: %v", err)
	}
	var fileOut bytes.Buffer
	if err := Run(t.Context(), "file", fileMeta, &fileOut); err != nil {
		t.Fatalf("Run file preview returned error: %v", err)
	}
	if !strings.Contains(fileOut.String(), "two") {
		t.Fatalf("expected file preview to contain target line, got %q", fileOut.String())
	}

	dirMeta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: filepath.Join(root, "dir")})
	if err != nil {
		t.Fatalf("EncodeMeta dir: %v", err)
	}
	var dirOut bytes.Buffer
	if err := Run(t.Context(), "dir", dirMeta, &dirOut); err != nil {
		t.Fatalf("Run dir preview returned error: %v", err)
	}
	if !strings.Contains(dirOut.String(), "child.txt") {
		t.Fatalf("expected dir preview to contain child entry, got %q", dirOut.String())
	}
}

func TestRunDiffPreview(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	file := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(file, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial commit")
	commit := strings.TrimSpace(runGitOutput(t, root, "rev-parse", "HEAD"))

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("LookPath git: %v", err)
	}
	shimDir := t.TempDir()
	shim := filepath.Join(shimDir, "git")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nexec \""+gitPath+"\" \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile shim: %v", err)
	}
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Repo: root, GitType: "commit", GitRef: commit})
	if err != nil {
		t.Fatalf("EncodeMeta diff: %v", err)
	}
	var out bytes.Buffer
	if err := Run(t.Context(), "diff", meta, &out); err != nil {
		t.Fatalf("Run diff preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), "initial commit") {
		t.Fatalf("expected diff preview to contain commit text, got %q", out.String())
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
