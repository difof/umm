package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/resultfmt"
)

func runNormalInteractive(ctx context.Context, cfg cli.RootConfig) ([]resultfmt.Result, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	reloadCommand := buildEmitSearchCommand(exe, cfg)
	previewCommand := shellQuote(exe) + " preview {1} {2}"
	input := startNormalSearchInput(ctx, exe, cfg)

	args := []string{
		"--ansi",
		"--disabled",
		"--query=" + cfg.Pattern,
		"--delimiter=\t",
		"--with-nth=3..",
		"--prompt=> Search: ",
		"--info=inline",
		"--preview=" + previewCommand,
		"--preview-window=top:60%",
		"--bind", "change:reload:sleep 0.05; " + reloadCommand,
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

	output, err := execx.InteractiveOutputWithInput(ctx, "", nil, input, "fzf", args...)
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
		"--ansi",
		"--delimiter=\t",
		"--with-nth=3..",
		"--header=" + buildGitHeader(cfg.GitModes),
		"--header-first",
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
	parts := []string{shellQuote(exe)}
	for _, arg := range buildEmitReloadArgs(cfg) {
		if arg == "{q}" {
			parts = append(parts, arg)
			continue
		}
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func buildEmitReloadArgs(cfg cli.RootConfig) []string {
	args := []string{"__emit-search", "--pattern", "{q}", "--root", cfg.Root}
	for _, pattern := range cfg.Excludes {
		args = append(args, "--exclude", pattern)
	}
	if cfg.Hidden {
		args = append(args, "--hidden")
	}
	if cfg.NoFilename {
		args = append(args, "--no-filename")
	}
	if cfg.OnlyFilename {
		args = append(args, "--only-filename")
	}
	if cfg.OnlyDirname {
		args = append(args, "--only-dirname")
	}
	if cfg.MaxDepth > 0 {
		args = append(args, "--max-depth", itoa(cfg.MaxDepth))
	}

	return args
}

func buildEmitArgs(cfg cli.RootConfig, patternStdin bool) []string {
	args := []string{"__emit-search"}
	if patternStdin {
		args = append(args, "--pattern-stdin")
	} else if cfg.Pattern != "" {
		args = append(args, "--pattern", cfg.Pattern)
	}
	args = append(args, "--root", cfg.Root)
	for _, pattern := range cfg.Excludes {
		args = append(args, "--exclude", pattern)
	}
	if cfg.Hidden {
		args = append(args, "--hidden")
	}
	if cfg.NoFilename {
		args = append(args, "--no-filename")
	}
	if cfg.OnlyFilename {
		args = append(args, "--only-filename")
	}
	if cfg.OnlyDirname {
		args = append(args, "--only-dirname")
	}
	if cfg.MaxDepth > 0 {
		args = append(args, "--max-depth", itoa(cfg.MaxDepth))
	}

	return args
}

func startNormalSearchInput(ctx context.Context, exe string, cfg cli.RootConfig) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		args := buildEmitArgs(cfg, true)
		err := execx.Run(ctx, "", nil, strings.NewReader(cfg.Pattern), writer, io.Discard, exe, args...)
		if err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()

	return reader
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
