package actions

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/resultfmt"
)

func TestRenderPathStatsDeduplicatesPaths(t *testing.T) {
	path := t.TempDir()
	results := []resultfmt.Result{{Path: path}, {Path: path}}

	var out bytes.Buffer
	if err := RenderPathStats(&out, cli.StatModeLite, results); err != nil {
		t.Fatalf("RenderPathStats returned error: %v", err)
	}

	if count := strings.Count(strings.TrimSpace(out.String()), "\n"); count != 0 {
		t.Fatalf("expected one stat line, got output %q", out.String())
	}
}

func TestBuildPromptItems(t *testing.T) {
	t.Run("directory results disable editor", func(t *testing.T) {
		items := buildPromptItems([]resultfmt.Result{{Kind: resultfmt.KindDir, Path: "/tmp/dir"}}, false)
		for _, item := range items {
			if item.key == "editor" {
				t.Fatal("did not expect editor item for directory result")
			}
		}
	})

	t.Run("git non-file results only show stat and cancel", func(t *testing.T) {
		items := buildPromptItems([]resultfmt.Result{{Kind: resultfmt.KindGit, GitType: "commit", Summary: "commit: test"}}, true)
		got := []string{}
		for _, item := range items {
			got = append(got, item.key)
		}
		want := []string{"stat", "cancel"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("prompt items = %v, want %v", got, want)
		}
	})
}

func TestPromptModelSelection(t *testing.T) {
	model := newPromptModel([]promptItem{{key: "stat", title: "Stat", description: "Print stat output"}})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(promptModel)
	if final.choice != "stat" {
		t.Fatalf("choice = %q, want stat", final.choice)
	}
}

func TestPromptSelectionSummary(t *testing.T) {
	t.Run("normal paths are shown", func(t *testing.T) {
		summary := promptSelectionSummary([]resultfmt.Result{{Path: "/tmp/a.txt", Line: 12}, {Path: "/tmp/b.txt"}}, false)
		if !strings.Contains(summary, "/tmp/a.txt:12") || !strings.Contains(summary, "/tmp/b.txt") {
			t.Fatalf("expected summary to list selected paths, got %q", summary)
		}
	})

	t.Run("git tracked files prefer open-compatible list", func(t *testing.T) {
		summary := promptSelectionSummary([]resultfmt.Result{{GitType: "commit", Display: "commit: abc"}, {GitType: "tracked", Path: "/tmp/tracked.txt"}}, true)
		if !strings.Contains(summary, "Open-compatible tracked files:") || !strings.Contains(summary, "/tmp/tracked.txt") {
			t.Fatalf("expected tracked-file summary, got %q", summary)
		}
	})

	t.Run("editor targets are unique by path", func(t *testing.T) {
		targets := editorTargets([]resultfmt.Result{{Path: "/tmp/a.txt", Line: 3}, {Path: "/tmp/a.txt", Line: 10}, {Path: "/tmp/b.txt", Line: 1}})
		if len(targets) != 2 {
			t.Fatalf("expected two unique targets, got %#v", targets)
		}
		if targets[0].Path != "/tmp/a.txt" || targets[0].Line != 3 {
			t.Fatalf("expected first target to preserve first line, got %#v", targets[0])
		}
	})
}

func TestPromptActionRoutesSummaryAndStatToProvidedWriters(t *testing.T) {
	t.Setenv("UMM_TEST_OPEN_ASK_CHOICE", "stat")
	path := t.TempDir()
	results := []resultfmt.Result{{Path: path}}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := PromptAction(t.Context(), ummconfig.Defaults(), results, false, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("PromptAction returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Open-compatible files:") {
		t.Fatalf("expected summary on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Path: "+path) {
		t.Fatalf("expected stat output on stdout, got %q", stdout.String())
	}
}
