package config

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/difof/errors"
	"gopkg.in/yaml.v3"
)

func LoadEffective() (LoadResult, error) {
	path, exists, err := FindUserPath()
	if err != nil {
		return LoadResult{}, errors.Wrap(err)
	}

	result := LoadResult{Path: path, UserExists: exists, Config: Defaults()}
	if !exists {
		result.Config = applyThemeEnvOverride(result.Config)
		if err := Validate(result.Config); err != nil {
			return LoadResult{}, errors.Wrap(err)
		}
		return result, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{}, errors.Wrap(err)
	}
	result.ConfigBytes = append([]byte(nil), data...)

	raw, err := decodeRaw(data)
	if err != nil {
		return LoadResult{}, errors.Wrapf(err, "load config %s", path)
	}
	result.RawUser = raw
	result.Config = mergeIntoDefaults(raw)
	result.Config = applyThemeEnvOverride(result.Config)

	if err := Validate(result.Config); err != nil {
		return LoadResult{}, errors.Wrapf(err, "validate config %s", path)
	}

	return result, nil
}

func Marshal(cfg Config) ([]byte, error) {
	buffer := bytes.Buffer{}
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return nil, errors.Wrap(err)
	}
	if err := encoder.Close(); err != nil {
		return nil, errors.Wrap(err)
	}

	return buffer.Bytes(), nil
}

func WriteDefaults(path string) error {
	data, err := DefaultFile()
	if err != nil {
		return errors.Wrap(err)
	}

	if err := writeFile(path, data); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func WriteDefaultsForTheme(path string, theme string) error {
	data, err := DefaultFileForTheme(theme)
	if err != nil {
		return errors.Wrap(err)
	}

	if err := writeFile(path, data); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func DefaultFile() ([]byte, error) {
	return DefaultFileForTheme(Defaults().Theme)
}

func DefaultFileForTheme(theme string) ([]byte, error) {
	cfg := Defaults()
	cfg.Theme = theme
	buffer := bytes.Buffer{}
	buffer.WriteString(defaultFileHeader)

	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(starterConfig{
		Theme:    cfg.Theme,
		Git:      cfg.Git,
		Keybinds: cfg.Keybinds,
	}); err != nil {
		return nil, errors.Wrap(err)
	}
	if err := encoder.Close(); err != nil {
		return nil, errors.Wrap(err)
	}

	buffer.WriteString("\n")
	buffer.WriteString(defaultFileExamples)

	return buffer.Bytes(), nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func decodeRaw(data []byte) (RawConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var raw RawConfig
	if err := decoder.Decode(&raw); err != nil {
		return RawConfig{}, errors.Wrap(err)
	}

	return raw, nil
}

type starterConfig struct {
	Theme    string         `yaml:"theme"`
	Git      GitConfig      `yaml:"git"`
	Keybinds KeybindsConfig `yaml:"keybinds"`
}

const defaultFileHeader = `# umm configuration
#
# This file is optional. When omitted, umm uses the same defaults shown here.
# Built-in editor profiles already cover common editors such as nvim, vim, vi, nano,
# micro, emacs, emacsclient, code, cursor, agy, subl, and sublime_text.
#
# Config semantics:
# - theme selects the active fzf presentation theme by exact name.
# - git.default-modes applies only when --git is set and --git-mode was not passed.
# - all git limit fields use 0 to mean unlimited.
# - keybinds.normal.bind and keybinds.git.bind replace the built-in bind lists.
# - keybinds.git.expect-keys is passed to fzf --expect for direct-open style keys.
# - expect-keys takes precedence over bind when the same key appears in both places.
# - preview.file/diff/tree run custom commands first; if missing or failing, umm warns
#   once at command startup and then falls back to the built-in preview flow.
# - built-in themes are listed with the "umm theme list" command.
# - dumped or custom user themes live in the sibling themes/ directory.
#
# Supported template variables:
# - editors first-target/rest-target, preview.file args, preview.tree args:
#   Path, Line, HasLine, StartLine, EndLine, LineRange
# - preview.diff args:
#   Repo, GitType, GitRef, Path, Display, Summary
# - keybinds bind entries:
#   ReloadCommand, PreviewCommand
#
# Keybind syntax:
# - bind entries use raw fzf syntax exactly: KEY:ACTION, EVENT:ACTION, or chains such as
#   KEY:ACTION+ACTION. Nested actions like reload(...), execute(...), preview(...),
#   change-preview-window(...), transform(...), and become(...) are supported.
# - expect-keys uses the same key names accepted by fzf --expect.
#
# Key names accepted by fzf include:
# - any single character
# - ctrl-<char>, alt-<char>
# - alt-up, alt-down, alt-left, alt-right, alt-enter, alt-space,
#   alt-backspace (alt-bspace, alt-bs)
# - tab, shift-tab (btab), esc, del/delete, up, down, left, right, home, end,
#   insert, page-up (pgup), page-down (pgdn)
# - shift-up, shift-down, shift-left, shift-right, shift-delete
# - alt-shift-up, alt-shift-down, alt-shift-left, alt-shift-right
# - left-click, right-click, double-click, scroll-up, scroll-down,
#   preview-scroll-up, preview-scroll-down, shift-left-click, shift-right-click,
#   shift-scroll-up, shift-scroll-down
#
# Events accepted by fzf include:
# - start, load, resize, result, change, focus, multi, one, zero,
#   backward-eof, jump, jump-cancel, click-header
#
# Actions accepted by fzf include:
# - abort, accept, accept-non-empty, accept-or-print-query
# - backward-char, backward-delete-char, backward-delete-char/eof,
#   backward-kill-word, backward-word
# - become(...), beginning-of-line, bell, cancel
# - change-border-label(...), change-ghost(...), change-header(...),
#   change-header-label(...), change-input-label(...), change-list-label(...),
#   change-multi, change-multi(...), change-nth(...), change-pointer(...),
#   change-preview(...), change-preview-label(...), change-preview-window(...),
#   change-prompt(...), change-query(...)
# - clear-screen, clear-multi, clear-query, close
# - delete-char, delete-char/eof, deselect, deselect-all, disable-search,
#   down, enable-search, end-of-line, exclude, exclude-multi
# - execute(...), execute-silent(...), first, forward-char, forward-word, ignore,
#   jump, kill-line, kill-word, last, next-history, next-selected
# - page-down, page-up, half-page-down, half-page-up
# - hide-header, hide-input, hide-preview
# - offset-down, offset-up, offset-middle, pos(...)
# - prev-history, prev-selected, preview(...), preview-down, preview-up,
#   preview-page-down, preview-page-up, preview-half-page-down,
#   preview-half-page-up, preview-bottom, preview-top
# - print(...), put, put(...), refresh-preview, rebind(...), reload(...),
#   reload-sync(...), replace-query, search(...), select, select-all,
#   show-header, show-input, show-preview
# - toggle, toggle-all, toggle-bind, toggle-header, toggle-hscroll, toggle-input,
#   toggle-in, toggle-multi-line, toggle-out, toggle-preview,
#   toggle-preview-wrap, toggle-search, toggle-sort, toggle-track,
#   toggle-track-current, toggle-wrap, toggle+down, toggle+up
# - track-current, transform(...), transform-border-label(...),
#   transform-ghost(...), transform-header(...), transform-header-label(...),
#   transform-input-label(...), transform-list-label(...), transform-nth(...),
#   transform-pointer(...), transform-preview-label(...), transform-prompt(...),
#   transform-query(...), transform-search(...)
# - unbind(...), unix-line-discard, unix-word-rubout, untrack-current, up, yank
# - each transform* action also has a matching bg-transform* variant in fzf

`

const defaultFileExamples = `# Define custom editor aliases here when built-in editor handling is not enough.
# The alias is matched against the first token of $EDITOR and its basename.
# editors:
#   zed:
#     cmd: zed
#     args:
#       - --wait
#     first-target:
#       - '{{.Path}}:{{.Line}}'
#     rest-target:
#       - '{{.Path}}'

# Override built-in previewers with external commands if you want.
# If the configured command is missing or fails, umm falls back and warns.
# preview:
#   file:
#     cmd: bat
#     args:
#       - --paging=never
#       - --color=always
#       - --style=numbers,header
#       - --line-range
#       - '{{.LineRange}}'
#       - '{{.Path}}'
#   diff:
#     cmd: delta
#     args: []
#   tree:
#     cmd: eza
#     args:
#       - --tree
#       - '{{.Path}}'

# Example keybind overrides:
# keybinds:
#   normal:
#     bind:
#       - 'change:reload:sleep 0.05; {{.ReloadCommand}}'
#       - 'ctrl-/:change-preview-window(right,70%|down,40%,border-horizontal|hidden|right)'
#       - 'ctrl-y:execute-silent(echo {} | pbcopy)'
#       - 'ctrl-o:accept'
#   git:
#     expect-keys:
#       - ctrl-o
#       - alt-enter
#     bind:
#       - 'ctrl-/:toggle-preview'
#       - 'ctrl-r:reload({{.ReloadCommand}})'
#       - 'tab:toggle+down,shift-tab:toggle+up'
`

func mergeIntoDefaults(raw RawConfig) Config {
	cfg := Defaults()

	if raw.Theme != nil {
		cfg.Theme = *raw.Theme
	}

	if raw.Git != nil {
		if raw.Git.DefaultModes != nil {
			cfg.Git.DefaultModes = append([]string(nil), raw.Git.DefaultModes...)
		}
		if raw.Git.Limits != nil {
			mergeGitLimits(&cfg.Git.Limits, *raw.Git.Limits)
		}
	}

	if raw.Keybinds != nil {
		if raw.Keybinds.Normal != nil && raw.Keybinds.Normal.Bind != nil {
			cfg.Keybinds.Normal.Bind = append([]string(nil), raw.Keybinds.Normal.Bind...)
		}
		if raw.Keybinds.Git != nil {
			if raw.Keybinds.Git.ExpectKeys != nil {
				cfg.Keybinds.Git.ExpectKeys = append([]string(nil), raw.Keybinds.Git.ExpectKeys...)
			}
			if raw.Keybinds.Git.Bind != nil {
				cfg.Keybinds.Git.Bind = append([]string(nil), raw.Keybinds.Git.Bind...)
			}
		}
	}

	if raw.Editors != nil {
		if cfg.Editors == nil {
			cfg.Editors = map[string]Editor{}
		}
		for name, editor := range raw.Editors {
			merged := cfg.Editors[name]
			if editor.Cmd != nil {
				merged.Cmd = *editor.Cmd
			}
			if editor.Args != nil {
				merged.Args = append([]string(nil), editor.Args...)
			}
			if editor.FirstTarget != nil {
				merged.FirstTarget = append([]string(nil), editor.FirstTarget...)
			}
			if editor.RestTarget != nil {
				merged.RestTarget = append([]string(nil), editor.RestTarget...)
			}
			cfg.Editors[name] = merged
		}
	}

	if raw.Preview != nil {
		if raw.Preview.File != nil {
			mergeCommand(&cfg.Preview.File, *raw.Preview.File)
		}
		if raw.Preview.Diff != nil {
			mergeCommand(&cfg.Preview.Diff, *raw.Preview.Diff)
		}
		if raw.Preview.Tree != nil {
			mergeCommand(&cfg.Preview.Tree, *raw.Preview.Tree)
		}
	}

	return cfg
}

func mergeGitLimits(dst *GitLimitsConfig, raw RawGitLimitsConfig) {
	if raw.Commits != nil {
		dst.Commits = *raw.Commits
	}
	if raw.Branches != nil {
		dst.Branches = *raw.Branches
	}
	if raw.Tags != nil {
		dst.Tags = *raw.Tags
	}
	if raw.Reflog != nil {
		dst.Reflog = *raw.Reflog
	}
	if raw.Stashes != nil {
		dst.Stashes = *raw.Stashes
	}
	if raw.Tracked != nil {
		dst.Tracked = *raw.Tracked
	}
	if raw.PreviewBranchCommits != nil {
		dst.PreviewBranchCommits = *raw.PreviewBranchCommits
	}
}

func mergeCommand(dst *Command, raw RawCommand) {
	if raw.Cmd != nil {
		dst.Cmd = *raw.Cmd
	}
	if raw.Args != nil {
		dst.Args = append([]string(nil), raw.Args...)
	}
}
