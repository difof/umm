package gitsearch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/resultfmt"
)

func TestAggregateAndQuery(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeGitFile(t, root, "tracked.txt", "hello\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial commit")
	runGit(t, root, "checkout", "-b", "feature/test")
	runGit(t, root, "tag", "v1.0.0")
	writeGitFile(t, root, "tracked.txt", "hello\nchange\n")
	runGit(t, root, "stash", "push", "-m", "temp stash")

	cfg := cli.RootConfig{Root: root, GitModes: cli.AllGitModes}
	results, err := Aggregate(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}

	assertHasGitType(t, results, "commit")
	assertHasGitType(t, results, "branch")
	assertHasGitType(t, results, "tag")
	assertHasGitType(t, results, "reflog")
	assertHasGitType(t, results, "stash")
	assertHasGitType(t, results, "tracked")

	filtered, err := Query(t.Context(), cfg, `tag:\s+v1\.0\.0`, true)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].GitType != "tag" {
		t.Fatalf("unexpected filtered results: %#v", filtered)
	}
}

func assertHasGitType(t *testing.T, results []resultfmt.Result, gitType string) {
	t.Helper()
	for _, result := range results {
		if result.GitType == gitType {
			return
		}
	}
	t.Fatalf("expected git type %q in results", gitType)
}

func writeGitFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
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
