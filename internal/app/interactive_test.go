package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	ummconfig "github.com/difof/umm/internal/config"
	ummtheme "github.com/difof/umm/internal/theme"
)

func TestResolveThemeArgsUsesConfiguredOverride(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	userDir := filepath.Join(xdg, "umm", "themes")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	data := []byte("name: lattice-dark\nvariant: dark\nfzf:\n  info: hidden\n  separator: '='\n")
	if err := os.WriteFile(filepath.Join(userDir, "lattice-dark.yml"), data, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	args, err := resolveThemeArgs(ummconfig.Defaults().Theme, ummtheme.RenderOverrides{Prompt: "> Search: ", Info: "inline", PreviewWindow: "top:60%"})
	if err != nil {
		t.Fatalf("resolveThemeArgs returned error: %v", err)
	}
	want := []string{"--info=hidden", "--prompt=> Search: ", "--separator==", "--preview-window=top:60%"}
	if !slices.Equal(args, want) {
		t.Fatalf("resolveThemeArgs() = %#v, want %#v", args, want)
	}
}
