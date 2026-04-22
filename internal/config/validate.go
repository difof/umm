package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/execx"
)

func Validate(cfg Config) error {
	if err := validateGit(cfg.Git); err != nil {
		return errors.Wrap(err)
	}
	if err := validateKeybinds(cfg.Keybinds); err != nil {
		return errors.Wrap(err)
	}
	if err := validateEditors(cfg.Editors); err != nil {
		return errors.Wrap(err)
	}
	if err := validatePreview(cfg.Preview); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func Check(ctx context.Context) (CheckReport, error) {
	path, exists, err := FindUserPath()
	if err != nil {
		return CheckReport{}, errors.Wrap(err)
	}

	report := CheckReport{Path: path, UserExists: exists}
	if !exists {
		return report, nil
	}

	data, err := osReadFile(path)
	if err != nil {
		return CheckReport{}, errors.Wrap(err)
	}

	raw, err := decodeRaw(data)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report, nil
	}

	cfg := mergeIntoDefaults(raw)
	if err := Validate(cfg); err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report, nil
	}

	report.Warnings = append(report.Warnings, commandWarnings(cfg, raw)...)
	report.Errors = append(report.Errors, keybindErrors(ctx, cfg)...)
	report.Warnings = append(report.Warnings, keybindWarnings()...)
	sort.Strings(report.Warnings)
	sort.Strings(report.Errors)

	return report, nil
}

func RuntimeWarnings(cfg Config) []string {
	warnings := []string{}
	warnings = append(warnings, warnIfMissing("preview.file", cfg.Preview.File.Cmd)...)
	warnings = append(warnings, warnIfMissing("preview.diff", cfg.Preview.Diff.Cmd)...)
	warnings = append(warnings, warnIfMissing("preview.tree", cfg.Preview.Tree.Cmd)...)
	sort.Strings(warnings)
	return warnings
}

func validateGit(cfg GitConfig) error {
	for _, mode := range cfg.DefaultModes {
		if _, ok := validGitModes[mode]; !ok {
			return errors.Newf("invalid git.default-modes entry %q", mode)
		}
	}

	limits := map[string]int{
		"git.limits.commits":                cfg.Limits.Commits,
		"git.limits.branches":               cfg.Limits.Branches,
		"git.limits.tags":                   cfg.Limits.Tags,
		"git.limits.reflog":                 cfg.Limits.Reflog,
		"git.limits.stashes":                cfg.Limits.Stashes,
		"git.limits.tracked":                cfg.Limits.Tracked,
		"git.limits.preview-branch-commits": cfg.Limits.PreviewBranchCommits,
	}
	for name, value := range limits {
		if value < 0 {
			return errors.Newf("%s must be non-negative", name)
		}
	}

	return nil
}

func validateKeybinds(cfg KeybindsConfig) error {
	for _, group := range [][]string{cfg.Normal.Bind, cfg.Git.Bind, cfg.Git.ExpectKeys} {
		for _, value := range group {
			if strings.TrimSpace(value) == "" {
				return errors.New("keybind entries must not be empty")
			}
		}
	}

	dummy := KeybindTemplateData{ReloadCommand: "umm __emit-search --pattern {q}", PreviewCommand: "umm preview {1} {2}"}
	for _, value := range cfg.Normal.Bind {
		if _, err := RenderString("normal-bind", value, dummy); err != nil {
			return errors.Wrap(err)
		}
	}
	for _, value := range cfg.Git.Bind {
		if _, err := RenderString("git-bind", value, dummy); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func validateEditors(editors map[string]Editor) error {
	for name, editor := range editors {
		if strings.TrimSpace(name) == "" {
			return errors.New("editor names must not be empty")
		}
		if strings.TrimSpace(editor.Cmd) == "" {
			return errors.Newf("editors.%s.cmd must not be empty", name)
		}
		if err := ValidateArgs(editor.Args, PathTemplateData{}); err != nil {
			return errors.Wrapf(err, "validate editors.%s.args", name)
		}
		if err := ValidateArgs(editor.FirstTarget, dummyPathData(true)); err != nil {
			return errors.Wrapf(err, "validate editors.%s.first-target", name)
		}
		if err := ValidateArgs(editor.RestTarget, dummyPathData(false)); err != nil {
			return errors.Wrapf(err, "validate editors.%s.rest-target", name)
		}
	}
	return nil
}

func validatePreview(cfg PreviewConfig) error {
	if err := validateCommand("preview.file", cfg.File, dummyPathData(true)); err != nil {
		return errors.Wrap(err)
	}
	if err := validateCommand("preview.diff", cfg.Diff, dummyDiffData()); err != nil {
		return errors.Wrap(err)
	}
	if err := validateCommand("preview.tree", cfg.Tree, dummyPathData(false)); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func validateCommand(name string, command Command, data any) error {
	if command.Cmd == "" && len(command.Args) == 0 {
		return nil
	}
	if strings.TrimSpace(command.Cmd) == "" {
		return errors.Newf("%s.cmd must not be empty when args are configured", name)
	}
	if err := ValidateArgs(command.Args, data); err != nil {
		return errors.Wrapf(err, "validate %s.args", name)
	}
	return nil
}

func dummyPathData(hasLine bool) PathTemplateData {
	line := 0
	start := 1
	end := 200
	lineRange := ":200"
	if hasLine {
		line = 12
		start = 2
		end = 32
		lineRange = "2:32"
	}
	return PathTemplateData{
		Path:      "/tmp/example.txt",
		Line:      line,
		HasLine:   hasLine,
		StartLine: start,
		EndLine:   end,
		LineRange: lineRange,
	}
}

func dummyDiffData() DiffTemplateData {
	return DiffTemplateData{
		Repo:    "/tmp/repo",
		GitType: "commit",
		GitRef:  "abc123",
		Path:    "/tmp/repo/file.txt",
		Display: "commit: abc123",
		Summary: "commit: abc123",
	}
}

func commandWarnings(cfg Config, raw RawConfig) []string {
	warnings := []string{}
	defaults := Defaults()

	if raw.Preview != nil {
		warnings = append(warnings, warnIfMissing("preview.file", cfg.Preview.File.Cmd)...)
		warnings = append(warnings, warnIfMissing("preview.diff", cfg.Preview.Diff.Cmd)...)
		warnings = append(warnings, warnIfMissing("preview.tree", cfg.Preview.Tree.Cmd)...)
	}

	for name := range raw.Editors {
		effective := cfg.Editors[name]
		if def, ok := defaults.Editors[name]; ok && reflect.DeepEqual(def, effective) {
			continue
		}
		warnings = append(warnings, warnIfMissing(fmt.Sprintf("editors.%s", name), effective.Cmd)...)
	}

	return warnings
}

func warnIfMissing(name string, command string) []string {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	if _, err := exec.LookPath(command); err != nil {
		return []string{fmt.Sprintf("%s command %q is not resolvable on PATH", name, command)}
	}
	return nil
}

func keybindWarnings() []string {
	if _, err := exec.LookPath("fzf"); err != nil {
		return []string{"fzf is not resolvable on PATH; skipping keybind parser validation"}
	}
	return nil
}

func keybindErrors(ctx context.Context, cfg Config) []string {
	if _, err := exec.LookPath("fzf"); err != nil {
		return nil
	}

	errorsList := []string{}
	if message := validateKeybindModeWithFZF(ctx, cfg.Keybinds.Normal.Bind, nil); message != "" {
		errorsList = append(errorsList, "normal keybinds: "+message)
	}
	if message := validateKeybindModeWithFZF(ctx, cfg.Keybinds.Git.Bind, cfg.Keybinds.Git.ExpectKeys); message != "" {
		errorsList = append(errorsList, "git keybinds: "+message)
	}
	return errorsList
}

func validateKeybindModeWithFZF(ctx context.Context, binds []string, expect []string) string {
	args := []string{"--filter", "x"}
	for _, bind := range binds {
		rendered, err := RenderString("fzf-bind", bind, KeybindTemplateData{ReloadCommand: "true", PreviewCommand: "cat {}"})
		if err != nil {
			return err.Error()
		}
		args = append(args, "--bind", rendered)
	}
	if len(expect) > 0 {
		args = append(args, "--expect", strings.Join(expect, ","))
	}

	output, err := execx.CombinedOutput(ctx, "", nil, strings.NewReader("x\n"), "fzf", args...)
	if err != nil {
		return strings.TrimSpace(output)
	}
	return ""
}

var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}
