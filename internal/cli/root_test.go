package cli

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeRootOptions(t *testing.T) {
	root := t.TempDir()

	t.Run("requires pattern for no-ui", func(t *testing.T) {
		_, err := NormalizeRootOptions(RawRootOptions{Root: root, NoUI: true})
		if err == nil {
			t.Fatal("expected error for no-ui without pattern")
		}
	})

	t.Run("rejects git plus exclude", func(t *testing.T) {
		_, err := NormalizeRootOptions(RawRootOptions{Root: root, Git: true, Excludes: []string{"vendor/**"}})
		if err == nil {
			t.Fatal("expected git conflict error")
		}
	})

	t.Run("rejects invalid stat mode", func(t *testing.T) {
		_, err := NormalizeRootOptions(RawRootOptions{Root: root, Pattern: "x", OnlyStat: "wat"})
		if err == nil {
			t.Fatal("expected invalid stat mode error")
		}
	})

	t.Run("defaults git modes to all", func(t *testing.T) {
		cfg, err := NormalizeRootOptions(RawRootOptions{Root: root, Git: true})
		if err != nil {
			t.Fatalf("NormalizeRootOptions returned error: %v", err)
		}

		if !reflect.DeepEqual(cfg.GitModes, AllGitModes) {
			t.Fatalf("git modes = %v, want %v", cfg.GitModes, AllGitModes)
		}
	})

	t.Run("normalizes repeated and comma git modes", func(t *testing.T) {
		cfg, err := NormalizeRootOptions(RawRootOptions{Root: root, Git: true, GitModes: []string{"commit,tracked", "branch"}})
		if err != nil {
			t.Fatalf("NormalizeRootOptions returned error: %v", err)
		}

		want := []string{"commit", "tracked", "branch"}
		if !reflect.DeepEqual(cfg.GitModes, want) {
			t.Fatalf("git modes = %v, want %v", cfg.GitModes, want)
		}
	})

	t.Run("classifies dirname action and root", func(t *testing.T) {
		cfg, err := NormalizeRootOptions(RawRootOptions{Root: root, OnlyDirname: true})
		if err != nil {
			t.Fatalf("NormalizeRootOptions returned error: %v", err)
		}

		if cfg.SearchMode != SearchModeOnlyDirname {
			t.Fatalf("search mode = %q, want %q", cfg.SearchMode, SearchModeOnlyDirname)
		}
		if cfg.Root != filepath.Clean(root) {
			t.Fatalf("root = %q, want %q", cfg.Root, filepath.Clean(root))
		}
	})
}
