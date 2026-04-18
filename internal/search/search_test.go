package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/difof/umm/internal/cli"
)

func TestQuery(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "needle one\n")
	writeFile(t, filepath.Join(root, "nested", "root.go"), "package main\n")
	writeFile(t, filepath.Join(root, ".hidden", "secret.txt"), "needle hidden\n")
	writeFile(t, filepath.Join(root, "cmd", "tool.txt"), "command\n")

	t.Run("default search finds content", func(t *testing.T) {
		cfg := cli.RootConfig{Root: root, SearchMode: cli.SearchModeDefault}
		results, err := Query(t.Context(), cfg, "needle", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected content results")
		}
	})

	t.Run("filename search finds file path only", func(t *testing.T) {
		cfg := cli.RootConfig{Root: root, SearchMode: cli.SearchModeOnlyFilename}
		results, err := Query(t.Context(), cfg, "root\\.go", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != "root.go" {
			t.Fatalf("unexpected filename-only results: %#v", results)
		}
	})

	t.Run("dirname search returns directory paths", func(t *testing.T) {
		cfg := cli.RootConfig{Root: root, SearchMode: cli.SearchModeOnlyDirname}
		results, err := Query(t.Context(), cfg, "cmd", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 || filepath.Base(results[0].Path) != "cmd" {
			t.Fatalf("unexpected dirname results: %#v", results)
		}
	})

	t.Run("hidden search excludes dot paths by default", func(t *testing.T) {
		cfg := cli.RootConfig{Root: root, SearchMode: cli.SearchModeOnlyFilename}
		results, err := Query(t.Context(), cfg, "secret\\.txt", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected hidden path to be excluded, got %#v", results)
		}
	})

	t.Run("hidden flag includes dot paths", func(t *testing.T) {
		cfg := cli.RootConfig{Root: root, SearchMode: cli.SearchModeOnlyFilename, Hidden: true}
		results, err := Query(t.Context(), cfg, "secret\\.txt", true)
		if err != nil {
			t.Fatalf("Query returned error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected hidden path result, got %#v", results)
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
