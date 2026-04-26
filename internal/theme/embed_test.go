package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuiltins(t *testing.T) {
	builtins, err := LoadBuiltins()
	if err != nil {
		t.Fatalf("LoadBuiltins returned error: %v", err)
	}
	if len(builtins) != 74 {
		t.Fatalf("expected full built-in catalog, got %d", len(builtins))
	}
	if builtins[0].Theme.Name == "" || len(builtins[0].Raw) == 0 {
		t.Fatalf("expected populated built-in entry, got %#v", builtins[0])
	}
	foundDefault := false
	foundForgeHub := false
	for _, builtin := range builtins {
		if builtin.Theme.Name == "lattice-dark" {
			foundDefault = true
			if builtin.Theme.FZF.PreviewBorder != BorderLine {
				t.Fatalf("expected lattice-dark preview border to be line, got %#v", builtin.Theme.FZF.PreviewBorder)
			}
		}
		if builtin.Theme.Name == "forge-hub-light" {
			foundForgeHub = true
		}
		if builtin.Theme.Variant == VariantDark {
			if builtin.Theme.FZF.PreviewBorder == "" {
				t.Fatalf("expected dark theme %q to set preview border", builtin.Theme.Name)
			}
			if builtin.Theme.FZF.PreviewBorder == BorderRounded {
				t.Fatalf("expected dark theme %q to avoid rounded preview border", builtin.Theme.Name)
			}
		}
	}
	if !foundDefault || !foundForgeHub {
		t.Fatalf("expected representative built-ins, got %#v", builtins)
	}
}

func TestDiscoverPrefersUserOverride(t *testing.T) {
	configDir := t.TempDir()
	userDir := UserDir(configDir)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	override := []byte("name: lattice-dark\nvariant: dark\nfzf:\n  info: inline\n  separator: '='\n")
	path := filepath.Join(userDir, "lattice-dark.yml")
	if err := os.WriteFile(path, override, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	catalog, err := Discover(configDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	entry, err := catalog.Resolve("lattice-dark")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if entry.Origin != OriginUser {
		t.Fatalf("expected user override, got %#v", entry)
	}
	if entry.Theme.FZF.Separator != "=" {
		t.Fatalf("expected overridden separator, got %#v", entry.Theme.FZF)
	}

	entries := catalog.Entries()
	shadowedBuiltin := false
	for _, item := range entries {
		if item.Name == "lattice-dark" && item.Origin == OriginBuiltin && item.Shadowed {
			shadowedBuiltin = true
		}
	}
	if !shadowedBuiltin {
		t.Fatalf("expected shadowed built-in entry, got %#v", entries)
	}
}

func TestDiscoverInvalidUserOverrideBlocksResolution(t *testing.T) {
	configDir := t.TempDir()
	userDir := UserDir(configDir)
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	path := filepath.Join(userDir, "lattice-dark.yml")
	if err := os.WriteFile(path, []byte("name: lattice-dark\nvariant: nope\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	catalog, err := Discover(configDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if _, err := catalog.Resolve("lattice-dark"); err == nil {
		t.Fatal("expected invalid user override to block resolution")
	}
}
