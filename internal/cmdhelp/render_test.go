package cmdhelp

import (
	"strings"
	"testing"
)

func TestRenderPlainText(t *testing.T) {
	doc := Document{
		Title: "Config File Schema",
		Intro: []string{"Documents the supported umm.yml fields."},
		Example: &Example{
			Title: "Minimal Example",
			Lines: []string{"theme: lattice-dark"},
		},
		Sections: []Section{{
			Title: "Field Reference",
			Body:  []string{"Only the most important fields are listed here."},
			Code:  []string{"theme: lattice-dark"},
			Extras: []LabelLine{{
				Label: "note",
				Text:  "Keep command-specific docs out of the renderer.",
			}},
			Fields: []Field{{
				Path:   "theme",
				What:   "Selects the active theme.",
				Values: "required string",
				Extras: []LabelLine{{Label: "example", Text: "lattice-dark"}},
			}},
		}},
	}

	got := Render(doc, RenderOptions{})
	wantChecks := []string{
		"Config File Schema",
		"Documents the supported umm.yml fields.",
		"Minimal Example",
		"theme: lattice-dark",
		"Field Reference",
		"theme: lattice-dark",
		"  note: Keep command-specific docs out of the renderer.",
		"theme",
		"  what: Selects the active theme.",
		"  values: required string",
		"  example: lattice-dark",
	}
	for _, want := range wantChecks {
		if !strings.Contains(got, want) {
			t.Fatalf("expected render output to contain %q, got %q", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("expected plain output without ANSI, got %q", got)
	}
}

func TestRenderColor(t *testing.T) {
	doc := Document{
		Title: "Theme File Schema",
		Example: &Example{
			Title: "Minimal Example",
			Lines: []string{"theme: lattice-dark"},
		},
		Sections: []Section{{
			Title: "Field Reference",
			Fields: []Field{{
				Path:   "theme",
				What:   "Selects the active theme.",
				Values: "required string",
			}},
		}},
	}

	got := Render(doc, RenderOptions{Color: true})
	checks := []string{
		"\x1b[1;36mTheme File Schema\x1b[0m",
		"\x1b[38;5;110mtheme: lattice-dark\x1b[0m",
		"\x1b[1;33mtheme\x1b[0m",
		"\x1b[2mwhat:\x1b[0m Selects the active theme.",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("expected colored render output to contain %q, got %q", check, got)
		}
	}
}
