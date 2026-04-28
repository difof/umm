package umm

import (
	"fmt"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cmdhelp"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/spf13/cobra"
)

func BuildKeybindsCmd() *cobra.Command {
	keybindsCmd := &cobra.Command{
		Use:   "keybinds",
		Short: "Show effective keybind maps and docs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printKeybindsDoc(cmd)
		},
	}
	keybindsCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		if err := printKeybindsDoc(cmd); err != nil {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
	})
	return keybindsCmd
}

func printKeybindsDoc(cmd *cobra.Command) (err error) {
	defer errors.Recover(&err)

	loaded := errors.MustResult(ummconfig.LoadEffective())
	err = cmdhelp.Write(cmd.OutOrStdout(), keybindsHelpDoc(loaded))
	return errors.Wrap(err)
}

func keybindsHelpDoc(loaded ummconfig.LoadResult) cmdhelp.Document {
	intro := []string{
		"This command shows the effective keymap after loading built-in defaults and applying any user config overrides.",
	}
	if loaded.UserExists {
		intro = append(intro,
			fmt.Sprintf("Source: %s", loaded.Path),
			"Configured bind lists replace the built-in lists instead of extending them.",
		)
	} else {
		intro = append(intro,
			fmt.Sprintf("Source: built-in defaults only; no user config file was found at %s", loaded.Path),
			"Create or inspect overrides with umm config dump, umm config show, or by editing umm.yml directly.",
		)
	}

	return cmdhelp.Document{
		Title:   "Keybinds Reference",
		Intro:   intro,
		Example: keybindOverrideExample(),
		Sections: []cmdhelp.Section{
			{
				Title: "Current Normal Keymap",
				Body: []string{
					"Raw fzf --bind entries used for the standard interactive picker.",
				},
				Code: formatBindLines(loaded.Config.Keybinds.Normal.Bind),
				Extras: []cmdhelp.LabelLine{
					{Label: "templates", Text: ummconfig.KeybindTemplateVariablesText(ummconfig.KeybindModeNormal)},
				},
			},
			{
				Title: "Current Git Keymap",
				Body: []string{
					"Raw fzf --bind entries used for git-mode pickers.",
				},
				Code: formatBindLines(loaded.Config.Keybinds.Git.Bind),
				Extras: []cmdhelp.LabelLine{
					{Label: "expect-keys", Text: formatExpectKeys(loaded.Config.Keybinds.Git.ExpectKeys)},
					{Label: "templates", Text: ummconfig.KeybindBindTemplateHelp(ummconfig.KeybindModeGit)},
				},
			},
			{
				Title: "Semantics",
				Fields: []cmdhelp.Field{
					{Path: "keybinds.normal.bind", What: "Replaces the built-in normal-mode bind list with raw fzf bind expressions.", Values: "list of strings in fzf KEY:ACTION, EVENT:ACTION, or chained KEY:ACTION+ACTION form."},
					{Path: "keybinds.git.expect-keys", What: "Registers direct-return keys through fzf --expect for git mode.", Values: "list of fzf key names such as ctrl-o or alt-enter.", Extras: []cmdhelp.LabelLine{{Label: "precedence", Text: "If a key appears in both expect-keys and git.bind, the expect key wins."}}},
					{Path: "keybinds.git.bind", What: "Replaces the built-in git-mode bind list with raw fzf bind expressions.", Values: "list of strings in the same format as keybinds.normal.bind."},
				},
			},
			{
				Title: "Template Variables",
				Body: []string{
					"Bind strings are rendered as Go templates before they are passed to fzf.",
				},
				Extras: []cmdhelp.LabelLine{
					{Label: "normal bind", Text: ummconfig.KeybindTemplateVariablesText(ummconfig.KeybindModeNormal)},
					{Label: "git bind", Text: ummconfig.KeybindTemplateVariablesText(ummconfig.KeybindModeGit)},
					{Label: "validation", Text: "Run umm config check to validate templates and, when fzf is installed, local key names and bind syntax."},
				},
			},
		},
	}
}

func keybindOverrideExample() *cmdhelp.Example {
	return &cmdhelp.Example{
		Title: "Override Example",
		Lines: []string{
			"keybinds:",
			"  normal:",
			"    bind:",
			"      - 'change:reload:sleep 0.05; {{.ReloadCommand}}'",
			"      - 'ctrl-/:change-preview-window(right,70%|down,40%,border-horizontal|hidden|right)'",
			"  git:",
			"    expect-keys:",
			"      - ctrl-o",
			"      - alt-enter",
			"    bind:",
			"      - 'ctrl-/:toggle-preview'",
			"      - 'ctrl-p:change-preview({{.PreviewCommand}})'",
		},
	}
}

func formatBindLines(values []string) []string {
	if len(values) == 0 {
		return []string{"  (none)"}
	}
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, "  - "+value)
	}
	return lines
}

func formatExpectKeys(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}
