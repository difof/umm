package app

import "testing"

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
