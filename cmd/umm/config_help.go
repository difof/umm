package umm

import (
	"strconv"
	"strings"

	"github.com/difof/umm/internal/cmdhelp"
	ummconfig "github.com/difof/umm/internal/config"
)

func configHelpDoc() cmdhelp.Document {
	return cmdhelp.Document{
		Title: "Config File Schema",
		Intro: []string{
			"User config lives at $XDG_CONFIG_HOME/umm/umm.yml or ~/.config/umm/umm.yml.",
			"This appendix documents the supported umm.yml structure and the nested fields users most often edit.",
		},
		Example: &cmdhelp.Example{
			Title: "Minimal Example",
			Lines: []string{
				"theme: lattice-dark",
				"git:",
				"  default-modes:",
				"    - commit",
				"    - tracked",
				"preview:",
				"  file:",
				"    cmd: bat",
				"    args:",
				"      - --paging=never",
				"      - '{{.LineRange}}'",
				"      - '{{.Path}}'",
			},
		},
		Sections: []cmdhelp.Section{
			{
				Title: "Field Reference",
				Fields: []cmdhelp.Field{
					{Path: "theme", What: "Selects the active fzf presentation theme by exact name.", Values: "required string; use a built-in or user theme name from umm theme list."},
					{Path: "git.default-modes", What: "Sets the default git object groups when --git is enabled without an explicit --git-mode flag.", Values: "list of any of " + quotedValues(ummconfig.AllGitModes) + ".", Extras: []cmdhelp.LabelLine{{Label: "notes", Text: "This list replaces the built-in defaults instead of merging with them."}}},
					{Path: "git.limits.commits", What: "Caps how many commit results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.branches", What: "Caps how many branch results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.tags", What: "Caps how many tag results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.reflog", What: "Caps how many reflog results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.stashes", What: "Caps how many stash results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.tracked", What: "Caps how many tracked-file results are loaded for git mode.", Values: limitValues()},
					{Path: "git.limits.preview-branch-commits", What: "Limits how many commits are shown in branch preview summaries.", Values: limitValues()},
					{Path: "keybinds.normal.bind", What: "Overrides the normal picker bind list with raw fzf --bind expressions.", Values: "list of raw fzf bind strings such as KEY:ACTION, EVENT:ACTION, or KEY:ACTION+ACTION.", Extras: []cmdhelp.LabelLine{{Label: "templates", Text: "Bind strings may reference {{.ReloadCommand}} and {{.PreviewCommand}}."}}},
					{Path: "keybinds.git.expect-keys", What: "Passes direct-open keys through fzf --expect in git mode.", Values: "list of raw fzf key names such as ctrl-o or alt-enter.", Extras: []cmdhelp.LabelLine{{Label: "notes", Text: "If the same key is present in git.bind, expect-keys takes precedence."}}},
					{Path: "keybinds.git.bind", What: "Overrides the git picker bind list with raw fzf --bind expressions.", Values: "list of raw fzf bind strings; same syntax and template variables as keybinds.normal.bind."},
					{Path: "editors.<name>.cmd", What: "Sets the executable used when the matching editor alias is detected from $EDITOR.", Values: "required non-empty command name or absolute path when the editor entry is present."},
					{Path: "editors.<name>.args", What: "Adds extra argv segments before any rendered target arguments.", Values: "list of strings; path-template rendering is applied to each item and empty results are dropped."},
					{Path: "editors.<name>.first-target", What: "Builds the target argv for the first selected result, including line-aware forms.", Values: "list of path-template strings; can include conditional line handling such as {{if .HasLine}}...{{end}}."},
					{Path: "editors.<name>.rest-target", What: "Builds the target argv for additional selected results after the first one.", Values: "list of path-template strings; usually simpler path-only arguments for multi-open flows."},
					{Path: "preview.file.cmd", What: "Overrides the built-in file preview command.", Values: "optional command string; when left empty, umm keeps the built-in file preview."},
					{Path: "preview.file.args", What: "Supplies argv for the custom file preview command.", Values: "list of path-template strings such as '{{.LineRange}}' and '{{.Path}}'.", Extras: []cmdhelp.LabelLine{{Label: "notes", Text: "If the configured command is missing or fails, umm warns once and falls back."}}},
					{Path: "preview.diff.cmd", What: "Overrides the built-in git diff preview command.", Values: "optional command string; when left empty, umm keeps the built-in diff preview."},
					{Path: "preview.diff.args", What: "Supplies argv for the custom diff preview command.", Values: "list of diff-template strings such as '{{.Repo}}', '{{.GitRef}}', '{{.Path}}', or '{{.Summary}}'."},
					{Path: "preview.tree.cmd", What: "Overrides the built-in tree preview command.", Values: "optional command string; when left empty, umm keeps the built-in tree preview."},
					{Path: "preview.tree.args", What: "Supplies argv for the custom tree preview command.", Values: "list of path-template strings; usually the selected path plus any tree tool flags."},
				},
			},
			{
				Title: "Template Variables",
				Body:  []string{"Template-aware args use Go template syntax with missing keys treated as errors."},
				Extras: []cmdhelp.LabelLine{
					{Label: "path args", Text: "Path, Line, HasLine, StartLine, EndLine, LineRange."},
					{Label: "diff args", Text: "Repo, GitType, GitRef, Path, Display, Summary."},
					{Label: "keybinds", Text: "ReloadCommand, PreviewCommand."},
				},
			},
			{
				Title: "Keybind Semantics",
				Body: []string{
					"Bind entries use raw fzf syntax exactly, including chained actions and nested actions such as reload(...), execute(...), preview(...), transform(...), and become(...).",
				},
				Extras: []cmdhelp.LabelLine{
					{Label: "replacement", Text: "keybinds.normal.bind and keybinds.git.bind replace the built-in bind lists instead of extending them."},
					{Label: "expect keys", Text: "keybinds.git.expect-keys is passed to fzf --expect, so matching keys bypass bind actions and return directly."},
					{Label: "validation", Text: "Run umm config check to validate templates and, when fzf is installed, local key names and bind syntax."},
				},
			},
		},
	}
}

func quotedValues(values []string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, strconv.Quote(value))
	}
	return strings.Join(items, ", ")
}

func limitValues() string {
	return "optional non-negative integer; 0 means unlimited."
}
