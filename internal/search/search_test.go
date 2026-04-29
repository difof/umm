package search

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ummruntime "github.com/difof/umm/internal/runtime"
)

func TestQuery(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle one\n")
	writeFile(t, filepath.Join(root, "nested", "root.go"), "package main\n")
	writeFile(t, filepath.Join(root, ".hidden", "secret.txt"), "needle hidden\n")
	writeFile(t, filepath.Join(root, "cmd", "tool.txt"), "command\n")
	if err := os.MkdirAll(filepath.Join(root, "emptydir"), 0o755); err != nil {
		t.Fatalf("MkdirAll emptydir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll hidden-empty: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skipme"), 0o755); err != nil {
		t.Fatalf("MkdirAll skipme: %v", err)
	}

	t.Run("default search finds content", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeDefault}
		results, err := Query(t.Context(), cfg, "needle", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected content results")
		}
	})

	t.Run("content search accepts single-dash patterns", func(t *testing.T) {
		writeFile(t, filepath.Join(root, "dash.txt"), "-foo line\n")

		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeDefault}
		results, err := Query(t.Context(), cfg, "-foo", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected single-dash content result")
		}
	})

	t.Run("content search accepts double-dash patterns", func(t *testing.T) {
		writeFile(t, filepath.Join(root, "double-dash.txt"), "--help text\n")

		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeDefault}
		results, err := Query(t.Context(), cfg, "--help", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected double-dash content result")
		}
	})

	t.Run("filename search finds file path only", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyFilename}
		results, err := Query(t.Context(), cfg, "root\\.go", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != "root.go" {
			t.Fatalf("unexpected filename-only results: %#v", results)
		}
	})

	t.Run("dirname search returns directory paths", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		results, err := Query(t.Context(), cfg, "cmd", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != "cmd" {
			t.Fatalf("unexpected dirname results: %#v", results)
		}
	})

	t.Run("dirname search finds empty directories", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		results, err := Query(t.Context(), cfg, "emptydir", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != "emptydir" {
			t.Fatalf("unexpected empty-dir results: %#v", results)
		}
	})

	t.Run("dirname search excludes hidden directories by default", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		results, err := Query(t.Context(), cfg, "hidden-empty", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected hidden dir to be excluded, got %#v", results)
		}
	})

	t.Run("dirname search includes hidden directories when enabled", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname, Hidden: true}
		results, err := Query(t.Context(), cfg, "hidden-empty", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != ".hidden-empty" {
			t.Fatalf("unexpected hidden dirname results: %#v", results)
		}
	})

	t.Run("dirname search honors excludes", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname, Excludes: []string{"skipme/**", "skipme/"}}
		results, err := Query(t.Context(), cfg, "skipme", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected excluded dirname to be skipped, got %#v", results)
		}
	})

	t.Run("dirname search reports unreadable directories without failing", func(t *testing.T) {
		locked := filepath.Join(root, "locked")
		if err := os.MkdirAll(filepath.Join(locked, "child"), 0o755); err != nil {
			t.Fatalf("MkdirAll locked: %v", err)
		}
		if err := os.Chmod(locked, 0o000); err != nil {
			t.Fatalf("Chmod locked: %v", err)
		}
		defer func() {
			_ = os.Chmod(locked, 0o755)
		}()
		if _, err := os.ReadDir(locked); err == nil {
			t.Skip("test environment can still read chmod 000 directories")
		}

		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		var warnings bytes.Buffer
		results, err := QueryWithErrorOutput(t.Context(), cfg, "locked|cmd", true, &warnings)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected dirname results despite unreadable subtree")
		}
		if !strings.Contains(warnings.String(), locked) {
			t.Fatalf("expected warning output to mention %q, got %q", locked, warnings.String())
		}
	})

	t.Run("dirname search prunes gitignored unreadable directories", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, filepath.Join(repo, ".gitignore"), "ignored/\n")
		if err := os.MkdirAll(filepath.Join(repo, "visible", "child"), 0o755); err != nil {
			t.Fatalf("MkdirAll visible: %v", err)
		}
		ignored := filepath.Join(repo, "ignored")
		if err := os.MkdirAll(filepath.Join(ignored, "child"), 0o755); err != nil {
			t.Fatalf("MkdirAll ignored: %v", err)
		}
		runGitCommand(t, repo, "init")
		if err := os.Chmod(ignored, 0o000); err != nil {
			t.Fatalf("Chmod ignored: %v", err)
		}
		defer func() {
			_ = os.Chmod(ignored, 0o755)
		}()
		if _, err := os.ReadDir(ignored); err == nil {
			t.Skip("test environment can still read chmod 000 directories")
		}

		cfg := ummruntime.RootConfig{Root: repo, SearchMode: ummruntime.SearchModeOnlyDirname}
		var warnings bytes.Buffer
		results, err := QueryWithErrorOutput(t.Context(), cfg, "ignored|visible", true, &warnings)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected visible dirname results")
		}
		for _, result := range results {
			if strings.Contains(result.Path, ignored) {
				t.Fatalf("expected gitignored path %q to be pruned from results, got %#v", ignored, results)
			}
		}
		if strings.Contains(warnings.String(), ignored) {
			t.Fatalf("expected gitignored path %q to be pruned before warnings, got %q", ignored, warnings.String())
		}
	})

	t.Run("dirname query results are sorted", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(root, "zzz"), 0o755); err != nil {
			t.Fatalf("MkdirAll zzz: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(root, "aaa"), 0o755); err != nil {
			t.Fatalf("MkdirAll aaa: %v", err)
		}

		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		results, err := Query(t.Context(), cfg, ".", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		for i := 1; i < len(results); i++ {
			if results[i-1].Display > results[i].Display {
				t.Fatalf("expected sorted dirname results, got %#v", results)
			}
		}
	})

	t.Run("hidden search excludes dot paths by default", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyFilename}
		results, err := Query(t.Context(), cfg, "secret\\.txt", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected hidden path to be excluded, got %#v", results)
		}
	})

	t.Run("hidden flag includes dot paths", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyFilename, Hidden: true}
		results, err := Query(t.Context(), cfg, "secret\\.txt", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected hidden path result, got %#v", results)
		}
	})
}

func TestEmitLinesHighlightsMatches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "nested", "root.go"), "package main\n")
	writeFile(t, filepath.Join(root, "cmd", "tool.txt"), "command\n")
	writeFile(t, filepath.Join(root, "a.txt"), "needle one\n")

	t.Run("filename matches are highlighted", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyFilename}
		var out bytes.Buffer
		if err := EmitLines(t.Context(), cfg, `root\.go`, &out); err != nil {
			t.Fatalf("EmitLines returned error: %v", err)
		}
		if !strings.Contains(out.String(), matchStartANSI+"root.go"+matchResetANSI) {
			t.Fatalf("expected filename highlight, got %q", out.String())
		}
	})

	t.Run("dirname matches are highlighted", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeOnlyDirname}
		var out bytes.Buffer
		if err := EmitLines(t.Context(), cfg, `cmd`, &out); err != nil {
			t.Fatalf("EmitLines returned error: %v", err)
		}
		if !strings.Contains(out.String(), matchStartANSI+"cmd"+matchResetANSI) {
			t.Fatalf("expected dirname highlight, got %q", out.String())
		}
	})

	t.Run("content matches are highlighted", func(t *testing.T) {
		cfg := ummruntime.RootConfig{Root: root, SearchMode: ummruntime.SearchModeDefault}
		var out bytes.Buffer
		if err := EmitLines(t.Context(), cfg, `needle`, &out); err != nil {
			t.Fatalf("EmitLines returned error: %v", err)
		}
		if !strings.Contains(out.String(), matchStartANSI+"needle"+matchResetANSI) {
			t.Fatalf("expected content highlight, got %q", out.String())
		}
	})
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
