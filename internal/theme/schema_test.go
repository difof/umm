package theme

import "testing"

func TestValidateAcceptsRepresentativeTheme(t *testing.T) {
	theme := Theme{
		Name:    "lattice-dark",
		Variant: VariantDark,
		FZF: FZFTheme{
			Style:         StyleFull,
			Layout:        LayoutReverse,
			Border:        BorderRounded,
			Info:          "inline",
			Prompt:        "> Search: ",
			PreviewWindow: "top,60%,border-horizontal",
			Color: ColorTheme{
				Base: BaseColorDark,
				Entries: map[string]string{
					"fg":             "#62ff94",
					"bg":             "#0a0e0a",
					"border":         "#8ca391",
					"prompt":         "#2eff6a",
					"pointer":        "#2eff6a",
					"preview-border": "#8ca391",
				},
			},
		},
	}

	if err := Validate(theme); err != nil {
		at := "Validate"
		t.Fatalf("%s returned error: %v", at, err)
	}
}

func TestValidateRejectsInvalidName(t *testing.T) {
	err := Validate(Theme{Name: "Lattice Dark", Variant: VariantDark})
	if err == nil {
		t.Fatal("expected invalid name error")
	}
}

func TestValidateRejectsInvalidVariant(t *testing.T) {
	err := Validate(Theme{Name: "lattice-dark", Variant: "nope"})
	if err == nil {
		t.Fatal("expected invalid variant error")
	}
}

func TestValidateRejectsInvalidColorKey(t *testing.T) {
	err := Validate(Theme{
		Name:    "lattice-dark",
		Variant: VariantDark,
		FZF: FZFTheme{
			Color: ColorTheme{Entries: map[string]string{"wat": "#fff"}},
		},
	})
	if err == nil {
		t.Fatal("expected invalid color key error")
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	_, err := Decode([]byte("name: lattice-dark\nvariant: dark\nextra: true\n"))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}
