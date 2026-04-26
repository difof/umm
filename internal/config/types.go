package config

import ummtheme "github.com/difof/umm/internal/theme"

type Config struct {
	Theme    string            `yaml:"theme"`
	Git      GitConfig         `yaml:"git"`
	Keybinds KeybindsConfig    `yaml:"keybinds"`
	Editors  map[string]Editor `yaml:"editors"`
	Preview  PreviewConfig     `yaml:"preview"`
}

type GitConfig struct {
	DefaultModes []string        `yaml:"default-modes"`
	Limits       GitLimitsConfig `yaml:"limits"`
}

type GitLimitsConfig struct {
	Commits              int `yaml:"commits"`
	Branches             int `yaml:"branches"`
	Tags                 int `yaml:"tags"`
	Reflog               int `yaml:"reflog"`
	Stashes              int `yaml:"stashes"`
	Tracked              int `yaml:"tracked"`
	PreviewBranchCommits int `yaml:"preview-branch-commits"`
}

type KeybindsConfig struct {
	Normal KeybindMode    `yaml:"normal"`
	Git    GitKeybindMode `yaml:"git"`
}

type KeybindMode struct {
	Bind []string `yaml:"bind"`
}

type GitKeybindMode struct {
	ExpectKeys []string `yaml:"expect-keys"`
	Bind       []string `yaml:"bind"`
}

type Editor struct {
	Cmd         string   `yaml:"cmd"`
	Args        []string `yaml:"args"`
	FirstTarget []string `yaml:"first-target"`
	RestTarget  []string `yaml:"rest-target"`
}

type PreviewConfig struct {
	File Command `yaml:"file"`
	Diff Command `yaml:"diff"`
	Tree Command `yaml:"tree"`
}

type Command struct {
	Cmd  string   `yaml:"cmd"`
	Args []string `yaml:"args"`
}

type RawConfig struct {
	Theme    *string              `yaml:"theme"`
	Git      *RawGitConfig        `yaml:"git"`
	Keybinds *RawKeybindsConfig   `yaml:"keybinds"`
	Editors  map[string]RawEditor `yaml:"editors"`
	Preview  *RawPreviewConfig    `yaml:"preview"`
}

type RawGitConfig struct {
	DefaultModes []string            `yaml:"default-modes"`
	Limits       *RawGitLimitsConfig `yaml:"limits"`
}

type RawGitLimitsConfig struct {
	Commits              *int `yaml:"commits"`
	Branches             *int `yaml:"branches"`
	Tags                 *int `yaml:"tags"`
	Reflog               *int `yaml:"reflog"`
	Stashes              *int `yaml:"stashes"`
	Tracked              *int `yaml:"tracked"`
	PreviewBranchCommits *int `yaml:"preview-branch-commits"`
}

type RawKeybindsConfig struct {
	Normal *RawKeybindMode    `yaml:"normal"`
	Git    *RawGitKeybindMode `yaml:"git"`
}

type RawKeybindMode struct {
	Bind []string `yaml:"bind"`
}

type RawGitKeybindMode struct {
	ExpectKeys []string `yaml:"expect-keys"`
	Bind       []string `yaml:"bind"`
}

type RawEditor struct {
	Cmd         *string  `yaml:"cmd"`
	Args        []string `yaml:"args"`
	FirstTarget []string `yaml:"first-target"`
	RestTarget  []string `yaml:"rest-target"`
}

type RawPreviewConfig struct {
	File *RawCommand `yaml:"file"`
	Diff *RawCommand `yaml:"diff"`
	Tree *RawCommand `yaml:"tree"`
}

type RawCommand struct {
	Cmd  *string  `yaml:"cmd"`
	Args []string `yaml:"args"`
}

type LoadResult struct {
	Path        string
	UserExists  bool
	Config      Config
	RawUser     RawConfig
	ConfigBytes []byte
}

type CheckReport struct {
	Path       string
	UserExists bool
	Errors     []string
	Warnings   []string
}

func (report CheckReport) Valid() bool {
	return len(report.Errors) == 0
}

var AllGitModes = []string{"commit", "branch", "tags", "reflog", "stash", "tracked"}

var validGitModes = map[string]struct{}{
	"commit":  {},
	"branch":  {},
	"tags":    {},
	"reflog":  {},
	"stash":   {},
	"tracked": {},
}

func Defaults() Config {
	return Config{
		Theme: ummtheme.DefaultName,
		Git: GitConfig{
			DefaultModes: append([]string(nil), AllGitModes...),
			Limits: GitLimitsConfig{
				PreviewBranchCommits: 10,
			},
		},
		Keybinds: KeybindsConfig{
			Normal: KeybindMode{Bind: []string{
				"change:reload:sleep 0.05; {{.ReloadCommand}}",
				"ctrl-g:last",
				"ctrl-b:first",
				"alt-g:preview-top",
				"alt-b:preview-bottom",
				"shift-up:preview-up",
				"shift-down:preview-down",
				"alt-u:preview-half-page-up",
				"alt-d:preview-half-page-down",
				"ctrl-u:half-page-up",
				"ctrl-d:half-page-down",
				"ctrl-o:accept",
				"tab:toggle+down,shift-tab:toggle+up",
			}},
			Git: GitKeybindMode{
				ExpectKeys: []string{"ctrl-o"},
				Bind: []string{
					"ctrl-/:toggle-preview",
					"ctrl-g:last",
					"ctrl-b:first",
					"alt-g:preview-top",
					"alt-b:preview-bottom",
					"shift-up:preview-up",
					"shift-down:preview-down",
					"alt-u:preview-half-page-up",
					"alt-d:preview-half-page-down",
					"ctrl-u:half-page-up",
					"ctrl-d:half-page-down",
					"tab:toggle+down,shift-tab:toggle+up",
				},
			},
		},
		Editors: defaultEditors(),
		Preview: PreviewConfig{
			File: Command{Args: []string{}},
			Diff: Command{Args: []string{}},
			Tree: Command{Args: []string{}},
		},
	}
}

func defaultEditors() map[string]Editor {
	return map[string]Editor{
		"vim":           defaultLineEditor("vim"),
		"vi":            defaultLineEditor("vi"),
		"nvim":          defaultLineEditor("nvim"),
		"nano":          defaultLineEditor("nano"),
		"micro":         defaultLineEditor("micro"),
		"emacs":         defaultLineEditor("emacs"),
		"emacsclient":   defaultLineEditor("emacsclient"),
		"code":          defaultGotoEditor("code"),
		"code-insiders": defaultGotoEditor("code-insiders"),
		"cursor":        defaultGotoEditor("cursor"),
		"agy":           defaultGotoEditor("agy"),
		"subl":          defaultPathLineEditor("subl"),
		"sublime_text":  defaultPathLineEditor("sublime_text"),
	}
}

func defaultLineEditor(cmd string) Editor {
	return Editor{
		Cmd:         cmd,
		Args:        []string{},
		FirstTarget: []string{"{{if .HasLine}}+{{.Line}}{{end}}", "{{.Path}}"},
		RestTarget:  []string{"{{.Path}}"},
	}
}

func defaultGotoEditor(cmd string) Editor {
	return Editor{
		Cmd:         cmd,
		Args:        []string{},
		FirstTarget: []string{"{{if .HasLine}}--goto{{end}}", "{{if .HasLine}}{{.Path}}:{{.Line}}{{else}}{{.Path}}{{end}}"},
		RestTarget:  []string{"{{.Path}}"},
	}
}

func defaultPathLineEditor(cmd string) Editor {
	return Editor{
		Cmd:         cmd,
		Args:        []string{},
		FirstTarget: []string{"{{if .HasLine}}{{.Path}}:{{.Line}}{{else}}{{.Path}}{{end}}"},
		RestTarget:  []string{"{{.Path}}"},
	}
}
