package gitsearch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/resultfmt"
	ummruntime "github.com/difof/umm/internal/runtime"
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

	cfg := ummruntime.RootConfig{Root: root, GitModes: []string{"commit", "branch", "tags", "reflog", "stash", "tracked"}}
	results, err := Aggregate(t.Context(), cfg, ummconfig.Defaults())
	if err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}

	assertHasGitType(t, results, "commit")
	assertHasGitType(t, results, "branch")
	assertHasGitType(t, results, "tag")
	assertHasGitType(t, results, "reflog")
	assertHasGitType(t, results, "stash")
	assertHasGitType(t, results, "tracked")

	filtered, err := Query(t.Context(), cfg, ummconfig.Defaults(), `tag:\s+v1\.0\.0`, true)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].GitType != "tag" {
		t.Fatalf("unexpected filtered results: %#v", filtered)
	}
}

func TestAggregateRespectsConfiguredLimits(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeGitFile(t, root, "one.txt", "one\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "one")
	writeGitFile(t, root, "two.txt", "two\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "two")

	appConfig := ummconfig.Defaults()
	appConfig.Git.Limits.Commits = 1
	appConfig.Git.Limits.Tracked = 1
	cfg := ummruntime.RootConfig{Root: root, GitModes: []string{"commit", "tracked"}}

	results, err := Aggregate(t.Context(), cfg, appConfig)
	if err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}

	commits := 0
	tracked := 0
	for _, result := range results {
		switch result.GitType {
		case "commit":
			commits++
		case "tracked":
			tracked++
		}
	}
	if commits != 1 || tracked != 1 {
		t.Fatalf("unexpected limited counts: commits=%d tracked=%d results=%#v", commits, tracked, results)
	}
}

func TestAggregateRespectsBranchLimit(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeGitFile(t, root, "tracked.txt", "hello\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	runGit(t, root, "checkout", "-b", "feature/a")
	runGit(t, root, "checkout", "-b", "feature/b")
	runGit(t, root, "checkout", "master")

	appConfig := ummconfig.Defaults()
	appConfig.Git.Limits.Branches = 1
	cfg := ummruntime.RootConfig{Root: root, GitModes: []string{"branch"}}

	results, err := Aggregate(t.Context(), cfg, appConfig)
	if err != nil {
		t.Fatalf("Aggregate returned error: %v", err)
	}
	if len(results) != 1 || results[0].GitType != "branch" {
		t.Fatalf("expected one limited branch result, got %#v", results)
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
