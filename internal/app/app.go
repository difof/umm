package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/actions"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/preview"
	"github.com/difof/umm/internal/resultfmt"
	"github.com/difof/umm/internal/search"
)

func RunRoot(ctx context.Context, cfg cli.RootConfig) (err error) {
	defer errors.Recover(&err)

	if cfg.SearchMode == cli.SearchModeGit {
		return runGit(ctx, cfg)
	}

	return runNormal(ctx, cfg)
}

func RunPreview(ctx context.Context, mode string, meta string, out io.Writer) error {
	return preview.Run(ctx, mode, meta, out)
}

func EmitSearch(ctx context.Context, cfg cli.RootConfig, out io.Writer) error {
	return search.EmitLines(ctx, cfg, cfg.Pattern, out)
}

func runNormal(ctx context.Context, cfg cli.RootConfig) error {
	if cfg.UsesRG() {
		if err := deps.Require("rg", "normal search"); err != nil {
			return errors.Wrap(err)
		}
	}

	if cfg.NoUI {
		results, err := search.Query(ctx, cfg, cfg.Pattern, true)
		if err != nil {
			return errors.Wrap(err)
		}
		if len(results) == 0 {
			return errors.Newf("no matches found for pattern %q", cfg.Pattern)
		}
		return runNormalNoUI(ctx, cfg, results)
	}

	if err := deps.Require("fzf", "interactive search"); err != nil {
		return errors.Wrap(err)
	}

	results, err := runNormalInteractive(ctx, cfg)
	if err != nil {
		return errors.Wrap(err)
	}
	if len(results) == 0 {
		return nil
	}

	return handleNormalSelection(ctx, cfg, results, false)
}

func runGit(ctx context.Context, cfg cli.RootConfig) error {
	if err := deps.Require("git", "git search"); err != nil {
		return errors.Wrap(err)
	}
	if err := gitsearch.ValidateRepo(ctx, cfg.Root); err != nil {
		return errors.Wrap(err)
	}

	if cfg.NoUI {
		results, err := gitsearch.Query(ctx, cfg, cfg.Pattern, true)
		if err != nil {
			return errors.Wrap(err)
		}
		if len(results) == 0 {
			return errors.Newf("no git matches found for pattern %q", cfg.Pattern)
		}
		return runGitNoUI(ctx, cfg, results)
	}

	if err := deps.Require("fzf", "interactive git search"); err != nil {
		return errors.Wrap(err)
	}

	results, ctrlO, err := runGitInteractive(ctx, cfg)
	if err != nil {
		return errors.Wrap(err)
	}
	if len(results) == 0 {
		return nil
	}

	if ctrlO {
		tracked := actions.TrackedFileSubset(results)
		if len(tracked) == 0 {
			return errors.New("Ctrl+O in git mode requires at least one tracked file selection")
		}
		return actions.OpenInEditor(ctx, tracked)
	}

	return handleGitSelection(ctx, cfg, results, false)
}

func runNormalNoUI(ctx context.Context, cfg cli.RootConfig, results []resultfmt.Result) error {
	if cfg.Action == cli.ActionStat {
		return actions.RenderPathStats(os.Stdout, cfg.StatMode, results)
	}

	if cfg.Action == cli.ActionAsk {
		return handleNormalSelection(ctx, cfg, results, true)
	}

	if cfg.Action == cli.ActionSystem {
		return handleNormalSelection(ctx, cfg, results[:1], true)
	}

	return handleNormalSelection(ctx, cfg, results[:1], true)
}

func runGitNoUI(ctx context.Context, cfg cli.RootConfig, results []resultfmt.Result) error {
	if cfg.Action == cli.ActionAsk {
		return handleGitSelection(ctx, cfg, results, true)
	}

	if cfg.Action == cli.ActionSystem {
		firstTracked := actions.FirstTrackedFile(results)
		if len(firstTracked) == 0 {
			return errors.New("no tracked file results available for open action")
		}
		return handleGitSelection(ctx, cfg, firstTracked, true)
	}

	return handleGitSelection(ctx, cfg, results, true)
}

func handleNormalSelection(ctx context.Context, cfg cli.RootConfig, results []resultfmt.Result, noUI bool) error {
	if len(results) == 0 {
		return nil
	}

	switch cfg.Action {
	case cli.ActionAsk:
		return actions.PromptAction(ctx, results, false)
	case cli.ActionSystem:
		return actions.OpenWithSystem(ctx, results)
	case cli.ActionStat:
		return actions.RenderPathStats(os.Stdout, cfg.StatMode, results)
	default:
		if cfg.SearchMode == cli.SearchModeOnlyDirname {
			if noUI {
				return actions.PrintPaths(os.Stdout, results[:1])
			}
			return actions.PrintPaths(os.Stdout, results)
		}

		editorResults := results
		if noUI {
			editorResults = results[:1]
		}
		return actions.OpenInEditor(ctx, editorResults)
	}
}

func handleGitSelection(ctx context.Context, cfg cli.RootConfig, results []resultfmt.Result, noUI bool) error {
	switch cfg.Action {
	case cli.ActionAsk:
		return actions.PromptAction(ctx, results, true)
	case cli.ActionSystem:
		tracked := actions.TrackedFileSubset(results)
		if noUI && len(tracked) > 1 {
			tracked = tracked[:1]
		}
		return actions.OpenWithSystem(ctx, tracked)
	case cli.ActionStat, cli.ActionDefault:
		return actions.RenderGitStats(os.Stdout, results)
	default:
		return actions.RenderGitStats(os.Stdout, results)
	}
}

func runNormalInteractive(ctx context.Context, cfg cli.RootConfig) ([]resultfmt.Result, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	reloadCommand := buildEmitSearchCommand(exe, cfg)
	previewCommand := shellQuote(exe) + " preview {1} {2}"

	args := []string{
		"--disabled",
		"--query=" + cfg.Pattern,
		"--delimiter=\t",
		"--with-nth=3..",
		"--prompt=> Search: ",
		"--info=inline",
		"--preview=" + previewCommand,
		"--preview-window=top:60%",
		"--bind", "change:reload:sleep 0.05; " + reloadCommand,
		"--bind", "start:reload:" + reloadCommand,
		"--bind", "ctrl-g:last",
		"--bind", "ctrl-b:first",
		"--bind", "alt-g:preview-top",
		"--bind", "alt-b:preview-bottom",
		"--bind", "shift-up:preview-up",
		"--bind", "shift-down:preview-down",
		"--bind", "alt-u:preview-half-page-up",
		"--bind", "alt-d:preview-half-page-down",
		"--bind", "ctrl-u:half-page-up",
		"--bind", "ctrl-d:half-page-down",
		"--bind", "ctrl-o:accept",
	}
	if !cfg.NoMulti {
		args = append(args,
			"--multi",
			"--bind", "tab:toggle+down,shift-tab:toggle+up",
		)
	}

	output, err := execx.InteractiveOutput(ctx, "", nil, "fzf", args...)
	if err != nil {
		if code, ok := execx.ExitCode(err); ok && (code == 1 || code == 130) && strings.TrimSpace(output) == "" {
			return nil, nil
		}
		return nil, errors.Wrap(err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	results, err := resultfmt.DecodeLines(output)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return results, nil
}

func runGitInteractive(ctx context.Context, cfg cli.RootConfig) ([]resultfmt.Result, bool, error) {
	results, err := gitsearch.Aggregate(ctx, cfg)
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	buffer := bytes.Buffer{}
	for _, result := range results {
		line, err := resultfmt.EncodeLine(result)
		if err != nil {
			return nil, false, errors.Wrap(err)
		}
		buffer.WriteString(line)
		buffer.WriteByte('\n')
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, false, errors.Wrap(err)
	}
	previewCommand := shellQuote(exe) + " preview {1} {2}"

	args := []string{
		"--delimiter=\t",
		"--with-nth=3..",
		"--prompt=> Git: ",
		"--info=inline",
		"--query=" + cfg.Pattern,
		"--preview=" + previewCommand,
		"--preview-window=top:60%",
		"--expect=ctrl-o",
		"--bind", "ctrl-/:toggle-preview",
		"--bind", "ctrl-g:last",
		"--bind", "ctrl-b:first",
		"--bind", "alt-g:preview-top",
		"--bind", "alt-b:preview-bottom",
		"--bind", "shift-up:preview-up",
		"--bind", "shift-down:preview-down",
		"--bind", "alt-u:preview-half-page-up",
		"--bind", "alt-d:preview-half-page-down",
		"--bind", "ctrl-u:half-page-up",
		"--bind", "ctrl-d:half-page-down",
	}
	if !cfg.NoMulti {
		args = append(args,
			"--multi",
			"--bind", "tab:toggle+down,shift-tab:toggle+up",
		)
	}

	output, err := execx.InteractiveOutputWithInput(ctx, "", nil, bytes.NewReader(buffer.Bytes()), "fzf", args...)
	if err != nil {
		if code, ok := execx.ExitCode(err); ok && (code == 1 || code == 130) && strings.TrimSpace(output) == "" {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, false, nil
	}

	ctrlO := false
	parts := strings.SplitN(output, "\n", 2)
	if len(parts) > 1 && parts[0] == "ctrl-o" {
		ctrlO = true
		output = parts[1]
	}

	results, err = resultfmt.DecodeLines(output)
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	return results, ctrlO, nil
}

func buildEmitSearchCommand(exe string, cfg cli.RootConfig) string {
	parts := []string{"printf '%s' {q} |", shellQuote(exe), "__emit-search", "--pattern-stdin", "--root", shellQuote(cfg.Root)}
	for _, pattern := range cfg.Excludes {
		parts = append(parts, "--exclude", shellQuote(pattern))
	}
	if cfg.Hidden {
		parts = append(parts, "--hidden")
	}
	if cfg.NoFilename {
		parts = append(parts, "--no-filename")
	}
	if cfg.OnlyFilename {
		parts = append(parts, "--only-filename")
	}
	if cfg.OnlyDirname {
		parts = append(parts, "--only-dirname")
	}
	if cfg.MaxDepth > 0 {
		parts = append(parts, "--max-depth", itoa(cfg.MaxDepth))
	}

	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
