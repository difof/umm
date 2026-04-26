package theme

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/difof/errors"
	"gopkg.in/yaml.v3"
)

var themeNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var validColorEntries = map[string]struct{}{
	"alt-bg":            {},
	"alt-gutter":        {},
	"bg":                {},
	"bg+":               {},
	"border":            {},
	"current-bg":        {},
	"current-fg":        {},
	"current-hl":        {},
	"disabled":          {},
	"fg":                {},
	"fg+":               {},
	"footer":            {},
	"footer-bg":         {},
	"footer-border":     {},
	"footer-fg":         {},
	"footer-label":      {},
	"gap-line":          {},
	"ghost":             {},
	"gutter":            {},
	"header":            {},
	"header-bg":         {},
	"header-border":     {},
	"header-fg":         {},
	"header-label":      {},
	"hl":                {},
	"hl+":               {},
	"info":              {},
	"input-bg":          {},
	"input-border":      {},
	"input-fg":          {},
	"input-label":       {},
	"label":             {},
	"list-bg":           {},
	"list-border":       {},
	"list-fg":           {},
	"list-label":        {},
	"marker":            {},
	"nomatch":           {},
	"nth":               {},
	"pointer":           {},
	"preview-bg":        {},
	"preview-border":    {},
	"preview-fg":        {},
	"preview-label":     {},
	"preview-scrollbar": {},
	"prompt":            {},
	"query":             {},
	"scrollbar":         {},
	"selected-bg":       {},
	"selected-fg":       {},
	"selected-hl":       {},
	"separator":         {},
	"spinner":           {},
}

func Decode(data []byte) (Theme, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var parsed Theme
	if err := decoder.Decode(&parsed); err != nil {
		return Theme{}, errors.Wrap(err)
	}
	if err := Validate(parsed); err != nil {
		return Theme{}, errors.Wrap(err)
	}

	return parsed, nil
}

func Validate(theme Theme) error {
	if !themeNamePattern.MatchString(strings.TrimSpace(theme.Name)) {
		return errors.Newf("theme.name must match %q", themeNamePattern.String())
	}
	if err := validateEnum("theme.variant", string(theme.Variant), []string{string(VariantLight), string(VariantDark)}); err != nil {
		return errors.Wrap(err)
	}
	if err := validateFZFTheme(theme.FZF); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func validateFZFTheme(theme FZFTheme) error {
	if err := validateOptionalEnum("theme.fzf.style", string(theme.Style), []string{string(StyleDefault), string(StyleMinimal), string(StyleFull)}); err != nil {
		return errors.Wrap(err)
	}
	if err := validateOptionalEnum("theme.fzf.layout", string(theme.Layout), []string{string(LayoutDefault), string(LayoutReverse), string(LayoutReverseList)}); err != nil {
		return errors.Wrap(err)
	}
	if err := validateOptionalEnum("theme.fzf.border", string(theme.Border), validBorderStyles()); err != nil {
		return errors.Wrap(err)
	}
	for field, value := range map[string]string{
		"theme.fzf.list-border":         string(theme.ListBorder),
		"theme.fzf.input-border":        string(theme.InputBorder),
		"theme.fzf.preview-border":      string(theme.PreviewBorder),
		"theme.fzf.header-border":       string(theme.HeaderBorder),
		"theme.fzf.header-lines-border": string(theme.HeaderLinesBorder),
		"theme.fzf.footer-border":       string(theme.FooterBorder),
	} {
		if err := validateOptionalEnum(field, value, validBorderStyles()); err != nil {
			return errors.Wrap(err)
		}
	}
	if err := validateOptionalInfoStyle(theme.Info); err != nil {
		return errors.Wrap(err)
	}
	if err := validateColorTheme(theme.Color); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func validateColorTheme(theme ColorTheme) error {
	if err := validateOptionalEnum("theme.fzf.color.base", string(theme.Base), []string{string(BaseColorDark), string(BaseColorLight), string(BaseColorBase16), string(BaseColor16), string(BaseColorBW)}); err != nil {
		return errors.Wrap(err)
	}
	for key, value := range theme.Entries {
		if _, ok := validColorEntries[key]; !ok {
			return errors.Newf("theme.fzf.color.entries contains unsupported key %q", key)
		}
		if strings.TrimSpace(value) == "" {
			return errors.Newf("theme.fzf.color.entries.%s must not be empty", key)
		}
	}

	return nil
}

func validateOptionalInfoStyle(value string) error {
	if value == "" {
		return nil
	}
	if value == string(InfoDefault) || value == string(InfoRight) || value == string(InfoHidden) || value == string(InfoInline) || value == string(InfoInlineRight) {
		return nil
	}
	if strings.HasPrefix(value, string(InfoInline)+":") || strings.HasPrefix(value, string(InfoInlineRight)+":") {
		return nil
	}
	return errors.Newf("theme.fzf.info must be one of %s", strings.Join([]string{"default", "right", "hidden", "inline", "inline:<prefix>", "inline-right", "inline-right:<prefix>"}, ", "))
}

func validateEnum(field string, value string, allowed []string) error {
	if err := validateOptionalEnum(field, value, allowed); err != nil {
		return errors.Wrap(err)
	}
	if value == "" {
		return errors.Newf("%s must not be empty", field)
	}
	return nil
}

func validateOptionalEnum(field string, value string, allowed []string) error {
	if value == "" {
		return nil
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return errors.Newf("%s must be one of %s", field, strings.Join(allowed, ", "))
}

func validBorderStyles() []string {
	values := []string{
		string(BorderRounded),
		string(BorderSharp),
		string(BorderBold),
		string(BorderDouble),
		string(BorderDashed),
		string(BorderBlock),
		string(BorderThinBlock),
		string(BorderHorizontal),
		string(BorderVertical),
		string(BorderLine),
		string(BorderTop),
		string(BorderBottom),
		string(BorderLeft),
		string(BorderRight),
		string(BorderInline),
		string(BorderNone),
	}
	sort.Strings(values)
	return values
}
