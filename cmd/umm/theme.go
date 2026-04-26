package umm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	ummtheme "github.com/difof/umm/internal/theme"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func BuildThemeCmd() *cobra.Command {
	themeCmd := &cobra.Command{
		Use:   "theme",
		Short: "Inspect and manage fzf themes",
		Long: strings.Join([]string{
			"Inspect and manage fzf themes.",
			"",
			"Use \"umm theme --help\" to see the command help plus the full theme file schema reference.",
		}, "\n"),
	}
	defaultHelp := themeCmd.HelpFunc()
	themeCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		defaultHelp(cmd, args)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), buildThemeHelpBlock(cmd.OutOrStdout()))
	})

	themeCmd.AddCommand(
		buildThemeListCmd(),
		buildThemeSetCmd(),
		buildThemeDumpCmd(),
	)

	return themeCmd
}

type themeFieldDoc struct {
	Path   string
	What   string
	Values string
	Extra  []themeFieldExtra
}

type themeFieldExtra struct {
	Label string
	Text  string
}

func buildThemeHelpBlock(out io.Writer) string {
	style := newHelpStyle(out)
	lines := []string{
		"",
		style.heading("Theme File Schema"),
		"Theme files are YAML and live in built-ins or under the user themes/ directory.",
		"Each field below includes one line for what it controls and one line for supported values or formats.",
		"",
		style.heading("Minimal Example"),
		style.code("  name: lattice-dark"),
		style.code("  variant: dark"),
		style.code("  fzf:"),
		style.code("    border: sharp"),
		style.code("    preview-border: line"),
		style.code("    color:"),
		style.code("      base: dark"),
		style.code("      entries:"),
		style.code("        fg: \"#62ff94\""),
		style.code("        bg: \"#0a0e0a\""),
		"",
		style.heading("Field Reference"),
	}

	for _, doc := range themeFieldDocs() {
		lines = append(lines,
			style.field(doc.Path),
			style.label("what:")+" "+doc.What,
			style.label("values:")+" "+doc.Values,
		)
		for _, extra := range doc.Extra {
			lines = append(lines, style.label(extra.Label+": ")+extra.Text)
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

type helpStyle struct {
	color bool
}

func newHelpStyle(out io.Writer) helpStyle {
	file, ok := out.(*os.File)
	if !ok {
		return helpStyle{}
	}
	return helpStyle{color: isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())}
}

func (s helpStyle) heading(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[1;36m" + value + "\x1b[0m"
}

func (s helpStyle) field(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[1;33m" + value + "\x1b[0m"
}

func (s helpStyle) label(value string) string {
	if !s.color {
		return "  " + value
	}
	return "  \x1b[2m" + value + "\x1b[0m"
}

func (s helpStyle) code(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[38;5;110m" + value + "\x1b[0m"
}

func themeFieldDocs() []themeFieldDoc {
	return []themeFieldDoc{
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
		{Path: "fzf.color.entries", What: "YAML map of individual fzf UI color slots to color specs after the base scheme is applied.", Values: "optional mapping; format is <slot>: <color-spec>, where the slot names the UI element and the color spec is raw fzf color syntax such as #62ff94, 123, or bold:#62ff94.", Extra: colorEntriesHelpExtras()},
	}
}

func colorEntriesHelpExtras() []themeFieldExtra {
	return []themeFieldExtra{
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

func buildThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in and user themes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runThemeListCmd(cmd)
		},
	}
}

func buildThemeSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set the active theme by exact name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeSetCmd(cmd, args[0])
		},
	}
}

func buildThemeDumpCmd() *cobra.Command {
	force := false
	cmd := &cobra.Command{
		Use:   "dump <name>",
		Short: "Dump a built-in theme into the user theme directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeDumpCmd(cmd, args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing user theme file")
	return cmd
}

func runThemeListCmd(cmd *cobra.Command) (err error) {
	defer errors.Recover(&err)

	selectedTheme := ummconfig.Defaults().Theme
	if loaded, loadErr := ummconfig.LoadEffective(); loadErr == nil {
		selectedTheme = loaded.Config.Theme
	}
	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	catalog := errors.MustResult(ummtheme.Discover(configDir))

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "name\tvariant\torigin\tstatus")
	for _, entry := range catalog.Entries() {
		statuses := themeStatuses(entry, selectedTheme)
		_, err = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", entry.Name, entry.Variant, entry.Origin, strings.Join(statuses, ","))
		if err != nil {
			return errors.Wrap(err)
		}
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func runThemeSetCmd(cmd *cobra.Command, name string) (err error) {
	defer errors.Recover(&err)

	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	catalog := errors.MustResult(ummtheme.Discover(configDir))
	_, err = catalog.Resolve(name)
	if err != nil {
		return errors.Wrap(err)
	}

	path := errors.MustResult(ummconfig.SetTheme(name))
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", path)
	return errors.Wrap(err)
}

func runThemeDumpCmd(cmd *cobra.Command, name string, force bool) (err error) {
	defer errors.Recover(&err)

	builtins := errors.MustResult(ummtheme.LoadBuiltins())
	var builtin *ummtheme.BuiltinFile
	for i := range builtins {
		if builtins[i].Theme.Name == name {
			builtin = &builtins[i]
			break
		}
	}
	if builtin == nil {
		return errors.Newf("built-in theme %q was not found", name)
	}

	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	path := filepath.Join(ummtheme.UserDir(configDir), name+".yml")
	if _, statErr := os.Stat(path); statErr == nil && !force {
		return errors.Newf("theme file exists: %s (rerun with --force to overwrite)", path)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return errors.Wrap(statErr)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err)
	}
	if err := os.WriteFile(path, builtin.Raw, 0o644); err != nil {
		return errors.Wrap(err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", path)
	return errors.Wrap(err)
}

func themeStatuses(entry ummtheme.Entry, selectedTheme string) []string {
	statuses := []string{}
	if entry.Effective && entry.Name == selectedTheme {
		statuses = append(statuses, "active")
	}
	if entry.Name == ummtheme.DefaultName {
		statuses = append(statuses, "default")
	}
	if entry.Effective && entry.Origin == ummtheme.OriginUser {
		statuses = append(statuses, "effective")
	}
	if entry.Shadowed {
		statuses = append(statuses, "shadowed")
	}
	if entry.Invalid {
		statuses = append(statuses, "invalid")
	}
	return statuses
}
