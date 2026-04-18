package umm

import (
	"reflect"
	"testing"
)

func TestNormalizeGitModeInput(t *testing.T) {
	t.Run("splits comma separated values", func(t *testing.T) {
		got := normalizeGitModeInput([]string{"commit,tracked", "branch"})
		want := []string{"commit", "tracked", "branch"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("normalizeGitModeInput returned %v, want %v", got, want)
		}
	})

	t.Run("drops empty entries", func(t *testing.T) {
		got := normalizeGitModeInput([]string{"commit,,tracked", "", "branch"})
		want := []string{"commit", "tracked", "branch"}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("normalizeGitModeInput returned %v, want %v", got, want)
		}
	})
}
