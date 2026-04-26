package theme

import (
	"slices"
	"testing"
)

func TestRenderUsesOverridesWhenThemeLeavesFieldsUnset(t *testing.T) {
	args, err := Render(Theme{
		Name:    "lattice-dark",
		Variant: VariantDark,
		FZF: FZFTheme{
			Info:          "",
			PreviewWindow: "",
			Color: ColorTheme{
				Base: BaseColorDark,
				Entries: map[string]string{
					"prompt": "#2eff6a",
					"bg":     "#0a0e0a",
				},
			},
		},
	}, RenderOverrides{Prompt: "> Git: ", Info: "inline", PreviewWindow: "top:60%"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	want := []string{
		"--info=inline",
		"--prompt=> Git: ",
		"--preview-window=top:60%",
		"--color=dark,bg:#0a0e0a,prompt:#2eff6a",
	}
	if !slices.Equal(args, want) {
		t.Fatalf("Render() = %#v, want %#v", args, want)
	}
}
