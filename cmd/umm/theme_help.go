package umm

import (
	"sort"
	"strconv"
	"strings"

	"github.com/difof/umm/internal/cmdhelp"
	ummtheme "github.com/difof/umm/internal/theme"
)

func themeHelpDoc() cmdhelp.Document {
	return cmdhelp.Document{
		Title: "Theme File Schema",
		Intro: []string{
			"Theme files are YAML and live in built-ins or under the user themes/ directory.",
			"Each field below includes one line for what it controls and one line for supported values or formats.",
		},
		Example: &cmdhelp.Example{
			Title: "Minimal Example",
			Lines: []string{
				"  name: lattice-dark",
				"  variant: dark",
				"  fzf:",
				"    border: sharp",
				"    preview-border: line",
				"    color:",
				"      base: dark",
				"      entries:",
				"        fg: \"#62ff94\"",
				"        bg: \"#0a0e0a\"",
			},
		},
		Sections: []cmdhelp.Section{{
			Title:  "Field Reference",
			Fields: themeFieldDocs(),
		}},
	}
}

func themeFieldDocs() []cmdhelp.Field {
	return []cmdhelp.Field{
		{Path: "name", What: "Public exact theme name used by config, UMM_THEME, and theme commands.", Values: "required; lowercase kebab-case; must match the file basename for built-ins and user overrides."},
		{Path: "variant", What: "Declares whether the theme is intended for light or dark presentation.", Values: enumValues(ummtheme.VariantLight, ummtheme.VariantDark)},
		{Path: "fzf.style", What: "Applies an fzf style preset for the overall picker chrome.", Values: enumValues(ummtheme.StyleDefault, ummtheme.StyleMinimal, ummtheme.StyleFull)},
		{Path: "fzf.layout", What: "Controls the list order/layout mode used by the picker.", Values: enumValues(ummtheme.LayoutDefault, ummtheme.LayoutReverse, ummtheme.LayoutReverseList)},
		{Path: "fzf.height", What: "Sets picker height when using a non-fullscreen layout.", Values: "optional string; any fzf height value such as 40%, 20, ~80%, or calc-like fzf forms accepted by your local fzf."},
		{Path: "fzf.min-height", What: "Sets the lower height bound for adaptive layouts.", Values: "optional string; fzf-native height format such as 10, 30%, or other supported min-height values."},
		{Path: "fzf.popup", What: "Enables popup mode and configures popup geometry.", Values: "optional string; raw fzf popup value such as centered, 80%,border-rounded, or other local fzf popup syntax."},
		{Path: "fzf.margin", What: "Adds outer spacing around the picker.", Values: "optional string; raw fzf margin syntax such as 1, 1,2, 5%, or 1,2,3,4."},
		{Path: "fzf.padding", What: "Adds inner spacing inside the picker border.", Values: "optional string; raw fzf padding syntax such as 1, 1,2, 5%, or 1,2,3,4."},
		{Path: "fzf.border", What: "Sets the main picker border style.", Values: borderStyleValues()},
		{Path: "fzf.list-border", What: "Sets the result list pane border style when the local fzf supports split borders.", Values: borderStyleValues()},
		{Path: "fzf.input-border", What: "Sets the input prompt pane border style.", Values: borderStyleValues()},
		{Path: "fzf.preview-border", What: "Sets the preview pane border style independently from the main border.", Values: borderStyleValues() + "; use line or sharp if you do not want rounded preview borders."},
		{Path: "fzf.header-border", What: "Sets the header pane border style.", Values: borderStyleValues()},
		{Path: "fzf.header-lines-border", What: "Sets the divider style for header lines in the local fzf version that supports it.", Values: borderStyleValues()},
		{Path: "fzf.footer-border", What: "Sets the footer pane border style.", Values: borderStyleValues()},
		{Path: "fzf.border-label", What: "Sets the text label shown on the main picker border.", Values: "optional string; any literal label text."},
		{Path: "fzf.border-label-pos", What: "Positions the main border label.", Values: "optional string; fzf-native label position such as 0, -3, 4:bottom, or center-like numeric offsets accepted by local fzf."},
		{Path: "fzf.list-label", What: "Sets the label shown on the list pane border.", Values: "optional string; any literal label text."},
		{Path: "fzf.list-label-pos", What: "Positions the list pane border label.", Values: "optional string; same position format as fzf.border-label-pos."},
		{Path: "fzf.input-label", What: "Sets the label shown on the input pane border.", Values: "optional string; any literal label text."},
		{Path: "fzf.input-label-pos", What: "Positions the input pane border label.", Values: "optional string; same position format as fzf.border-label-pos."},
		{Path: "fzf.header-label", What: "Sets the label shown on the header border.", Values: "optional string; any literal label text."},
		{Path: "fzf.header-label-pos", What: "Positions the header border label.", Values: "optional string; same position format as fzf.border-label-pos."},
		{Path: "fzf.footer-label", What: "Sets the label shown on the footer border.", Values: "optional string; any literal label text."},
		{Path: "fzf.footer-label-pos", What: "Positions the footer border label.", Values: "optional string; same position format as fzf.border-label-pos."},
		{Path: "fzf.preview-label", What: "Sets the label shown on the preview border.", Values: "optional string; any literal label text."},
		{Path: "fzf.preview-label-pos", What: "Positions the preview border label.", Values: "optional string; same position format as fzf.border-label-pos."},
		{Path: "fzf.info", What: "Controls where fzf shows the match and selection info text.", Values: infoValues()},
		{Path: "fzf.prompt", What: "Overrides the interactive prompt text when set.", Values: "optional string; any prompt literal such as > Search: or > Git: ."},
		{Path: "fzf.ghost", What: "Sets ghost text shown in the input area by fzf when supported.", Values: "optional string; any literal ghost text."},
		{Path: "fzf.separator", What: "Sets the separator glyph used by fzf.", Values: "optional string; one or more visible characters such as -, ─, or ::."},
		{Path: "fzf.pointer", What: "Sets the active row pointer glyph.", Values: "optional string; one or more visible characters such as >, ▌, or ->."},
		{Path: "fzf.marker", What: "Sets the multi-select marker glyph.", Values: "optional string; one or more visible characters such as *, ▌, or ┃."},
		{Path: "fzf.marker-multi-line", What: "Sets the multiline selection marker when supported by fzf.", Values: "optional string; one or more visible characters used for multiline marks."},
		{Path: "fzf.gutter", What: "Sets the gutter glyph or text used by fzf.", Values: "optional string; one or more visible characters."},
		{Path: "fzf.gutter-raw", What: "Passes raw gutter text through without fzf glyph normalization when supported.", Values: "optional string; raw fzf gutter payload."},
		{Path: "fzf.scrollbar", What: "Sets the scrollbar glyph used by fzf panes.", Values: "optional string; one or more visible characters such as |, │, or █."},
		{Path: "fzf.ellipsis", What: "Sets the truncation marker used for clipped text.", Values: "optional string; one or more visible characters such as ..., …, or >>>."},
		{Path: "fzf.wrap-sign", What: "Sets the continuation marker for wrapped list lines.", Values: "optional string; one or more visible characters."},
		{Path: "fzf.preview-wrap-sign", What: "Sets the continuation marker for wrapped preview lines.", Values: "optional string; one or more visible characters."},
		{Path: "fzf.preview-window", What: "Controls preview position, size, border mode, wrapping, and other preview window behavior.", Values: "optional string; raw fzf preview-window syntax such as top:60%, right,50%,border-left, up,30%,wrap, or hidden."},
		{Path: "fzf.color.base", What: "Sets the base fzf color scheme before explicit color entries are applied.", Values: enumValues(ummtheme.BaseColorDark, ummtheme.BaseColorLight, ummtheme.BaseColorBase16, ummtheme.BaseColor16, ummtheme.BaseColorBW)},
		{Path: "fzf.color.entries", What: "YAML map of individual fzf UI color slots to color specs after the base scheme is applied.", Values: "optional mapping; format is <slot>: <color-spec>, where the slot names the UI element and the color spec is raw fzf color syntax such as #62ff94, 123, or bold:#62ff94.", Extras: colorEntriesHelpExtras()},
	}
}

func colorEntriesHelpExtras() []cmdhelp.LabelLine {
	return []cmdhelp.LabelLine{
		{Label: "example", Text: `fg: "#62ff94", bg: "#0a0e0a", prompt: "bold:#2eff6a", preview-border: "#8ca391"`},
		{Label: "slots", Text: "main/text: fg, bg, fg+, bg+, hl, hl+, nth, query, current-fg, current-bg, current-hl, selected-fg, selected-bg, selected-hl, disabled, nomatch."},
		{Label: "slots", Text: "chrome: prompt, pointer, marker, spinner, separator, scrollbar, gap-line, border, label, gutter, alt-gutter, alt-bg, ghost."},
		{Label: "slots", Text: "panes: list-bg, list-fg, list-border, list-label, input-bg, input-fg, input-border, input-label, header, header-bg, header-fg, header-border, header-label, footer, footer-bg, footer-fg, footer-border, footer-label, preview-bg, preview-fg, preview-border, preview-label, preview-scrollbar."},
		{Label: "all keys", Text: strings.Join(validThemeColorEntryKeys(), ", ") + "."},
	}
}

func enumValues[T ~string](values ...T) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, strconv.Quote(string(value)))
	}
	return strings.Join(items, ", ")
}

func borderStyleValues() string {
	return "optional enum: " + strings.Join(validBorderStyleNames(), ", ")
}

func infoValues() string {
	return "optional string; one of \"default\", \"right\", \"hidden\", \"inline\", \"inline-right\", or prefixed forms like \"inline:TEXT\" and \"inline-right:TEXT\""
}

func validBorderStyleNames() []string {
	values := []string{
		string(ummtheme.BorderRounded),
		string(ummtheme.BorderSharp),
		string(ummtheme.BorderBold),
		string(ummtheme.BorderDouble),
		string(ummtheme.BorderDashed),
		string(ummtheme.BorderBlock),
		string(ummtheme.BorderThinBlock),
		string(ummtheme.BorderHorizontal),
		string(ummtheme.BorderVertical),
		string(ummtheme.BorderLine),
		string(ummtheme.BorderTop),
		string(ummtheme.BorderBottom),
		string(ummtheme.BorderLeft),
		string(ummtheme.BorderRight),
		string(ummtheme.BorderInline),
		string(ummtheme.BorderNone),
	}
	sort.Strings(values)
	for i := range values {
		values[i] = strconv.Quote(values[i])
	}
	return values
}

func validThemeColorEntryKeys() []string {
	keys := []string{
		"alt-bg",
		"alt-gutter",
		"bg",
		"bg+",
		"border",
		"current-bg",
		"current-fg",
		"current-hl",
		"disabled",
		"fg",
		"fg+",
		"footer",
		"footer-bg",
		"footer-border",
		"footer-fg",
		"footer-label",
		"gap-line",
		"ghost",
		"gutter",
		"header",
		"header-bg",
		"header-border",
		"header-fg",
		"header-label",
		"hl",
		"hl+",
		"info",
		"input-bg",
		"input-border",
		"input-fg",
		"input-label",
		"label",
		"list-bg",
		"list-border",
		"list-fg",
		"list-label",
		"marker",
		"nomatch",
		"nth",
		"pointer",
		"preview-bg",
		"preview-border",
		"preview-fg",
		"preview-label",
		"preview-scrollbar",
		"prompt",
		"query",
		"scrollbar",
		"selected-bg",
		"selected-fg",
		"selected-hl",
		"separator",
		"spinner",
	}
	sort.Strings(keys)
	return keys
}
