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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func DefaultFile() ([]byte, error) {
	cfg := Defaults()
	buffer := bytes.Buffer{}
	buffer.WriteString(defaultFileHeader)

	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(starterConfig{
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
	Git      GitConfig      `yaml:"git"`
	Keybinds KeybindsConfig `yaml:"keybinds"`
}

const defaultFileHeader = `# umm configuration
#
# This file is optional. When omitted, umm uses the same defaults shown here.
# Built-in editor profiles already cover common editors such as nvim, vim, vi, nano,
# micro, emacs, emacsclient, code, cursor, agy, subl, and sublime_text.

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
`

func mergeIntoDefaults(raw RawConfig) Config {
	cfg := Defaults()

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
