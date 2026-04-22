package preview

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ummconfig "github.com/difof/umm/internal/config"
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
	if err := Run(t.Context(), ummconfig.Defaults(), "file", fileMeta, &fileOut); err != nil {
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
	if err := Run(t.Context(), ummconfig.Defaults(), "dir", dirMeta, &dirOut); err != nil {
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
	if err := Run(t.Context(), ummconfig.Defaults(), "diff", meta, &out); err != nil {
		t.Fatalf("Run diff preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), "initial commit") {
		t.Fatalf("expected diff preview to contain commit text, got %q", out.String())
	}
}

func TestRunFilePreviewWithLongLine(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "long.txt")
	longLine := strings.Repeat("x", 300000)
	if err := os.WriteFile(file, []byte(longLine+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", t.TempDir()); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: file, Line: 1})
	if err != nil {
		t.Fatalf("EncodeMeta file: %v", err)
	}
	var out bytes.Buffer
	if err := Run(t.Context(), ummconfig.Defaults(), "file", meta, &out); err != nil {
		t.Fatalf("Run file preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), strings.Repeat("x", 64)) {
		t.Fatalf("expected long-line preview output, got %q", out.String())
	}
}

func TestRunFilePreviewDoesNotFallBackToCat(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "many.txt")
	lines := make([]string, 0, 260)
	for i := 1; i <= 260; i++ {
		lines = append(lines, strings.Repeat("x", 4)+" "+itoa(i))
	}
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	shimDir := t.TempDir()
	shim := filepath.Join(shimDir, "cat")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nprintf 'CAT_FALLBACK_USED\\n'\n"), 0o755); err != nil {
		t.Fatalf("WriteFile shim: %v", err)
	}
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: file})
	if err != nil {
		t.Fatalf("EncodeMeta file: %v", err)
	}
	var out bytes.Buffer
	if err := Run(t.Context(), ummconfig.Defaults(), "file", meta, &out); err != nil {
		t.Fatalf("Run file preview returned error: %v", err)
	}

	text := out.String()
	if strings.Contains(text, "CAT_FALLBACK_USED") {
		t.Fatalf("expected internal bounded preview, got cat fallback output %q", text)
	}
	if strings.Contains(text, "260") {
		t.Fatalf("expected preview to be capped before line 260, got %q", text)
	}
}

func TestRunUsesConfiguredFilePreviewCommand(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	shimDir := t.TempDir()
	writeExecutable(t, filepath.Join(shimDir, "preview-file"), "#!/bin/sh\nprintf 'CUSTOM FILE %s\\n' \"$1\"\n")
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: file, Line: 1})
	if err != nil {
		t.Fatalf("EncodeMeta file: %v", err)
	}
	appConfig := ummconfig.Defaults()
	appConfig.Preview.File = ummconfig.Command{Cmd: "preview-file", Args: []string{"{{.Path}}"}}

	var out bytes.Buffer
	if err := Run(t.Context(), appConfig, "file", meta, &out); err != nil {
		t.Fatalf("Run file preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), "CUSTOM FILE "+file) {
		t.Fatalf("expected custom preview output, got %q", out.String())
	}
}

func TestRunFallsBackWhenConfiguredPreviewCommandIsMissing(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", t.TempDir()); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: file, Line: 2})
	if err != nil {
		t.Fatalf("EncodeMeta file: %v", err)
	}
	appConfig := ummconfig.Defaults()
	appConfig.Preview.File = ummconfig.Command{Cmd: "missing-preview", Args: []string{"{{.Path}}"}}

	var out bytes.Buffer
	if err := Run(t.Context(), appConfig, "file", meta, &out); err != nil {
		t.Fatalf("Run file preview returned error: %v", err)
	}
	text := out.String()
	if strings.Contains(text, "Warning:") || !strings.Contains(text, "world") {
		t.Fatalf("expected silent fallback preview content, got %q", text)
	}
}

func TestRunUsesConfiguredTreePreviewCommand(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	shimDir := t.TempDir()
	writeExecutable(t, filepath.Join(shimDir, "preview-tree"), "#!/bin/sh\nprintf 'TREE %s\\n' \"$1\"\n")
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Path: dir})
	if err != nil {
		t.Fatalf("EncodeMeta dir: %v", err)
	}
	appConfig := ummconfig.Defaults()
	appConfig.Preview.Tree = ummconfig.Command{Cmd: "preview-tree", Args: []string{"{{.Path}}"}}

	var out bytes.Buffer
	if err := Run(t.Context(), appConfig, "dir", meta, &out); err != nil {
		t.Fatalf("Run dir preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), "TREE "+dir) {
		t.Fatalf("expected custom tree preview output, got %q", out.String())
	}
}

func TestRunUsesConfiguredDiffPreviewCommand(t *testing.T) {
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
	shimDir := t.TempDir()
	writeExecutable(t, filepath.Join(shimDir, "preview-diff"), "#!/bin/sh\ncat\n")
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("LookPath git: %v", err)
	}
	writeExecutable(t, filepath.Join(shimDir, "git"), "#!/bin/sh\nexec \""+gitPath+"\" \"$@\"\n")
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })
	if err := os.Setenv("PATH", shimDir); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Repo: root, GitType: "commit", GitRef: commit})
	if err != nil {
		t.Fatalf("EncodeMeta diff: %v", err)
	}
	appConfig := ummconfig.Defaults()
	appConfig.Preview.Diff = ummconfig.Command{Cmd: "preview-diff", Args: []string{}}

	var out bytes.Buffer
	if err := Run(t.Context(), appConfig, "diff", meta, &out); err != nil {
		t.Fatalf("Run diff preview returned error: %v", err)
	}
	if !strings.Contains(out.String(), "initial commit") {
		t.Fatalf("expected custom diff output from stdin, got %q", out.String())
	}
}

func TestRunDiffPreviewUsesConfiguredBranchLimit(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	file := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(file, []byte("one\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "one")
	if err := os.WriteFile(file, []byte("two\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, root, "commit", "-am", "two")
	if err := os.WriteFile(file, []byte("three\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGit(t, root, "commit", "-am", "three")
	runGit(t, root, "checkout", "-b", "feature")

	meta, err := resultfmt.EncodeMeta(resultfmt.Result{Repo: root, GitType: "branch", GitRef: "feature"})
	if err != nil {
		t.Fatalf("EncodeMeta diff: %v", err)
	}
	appConfig := ummconfig.Defaults()
	appConfig.Git.Limits.PreviewBranchCommits = 1

	var out bytes.Buffer
	if err := Run(t.Context(), appConfig, "diff", meta, &out); err != nil {
		t.Fatalf("Run diff preview returned error: %v", err)
	}
	text := strings.TrimSpace(out.String())
	lines := strings.Split(text, "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "three") {
		t.Fatalf("expected only one branch commit preview line, got %q", text)
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

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
