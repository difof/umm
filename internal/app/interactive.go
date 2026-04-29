package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/resultfmt"
	ummruntime "github.com/difof/umm/internal/runtime"
	ummtheme "github.com/difof/umm/internal/theme"
)

func runNormalInteractive(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, errOut io.Writer) ([]resultfmt.Result, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	reloadCommand := buildEmitSearchCommand(exe, cfg)
	previewCommand := shellQuote(exe) + " preview {1} {2}"
	input := startNormalSearchInput(ctx, exe, cfg, errOut)
	keybindArgs, err := buildBindArgs(appConfig.Keybinds.Normal.Bind, ummconfig.KeybindTemplateData{
		ReloadCommand:  reloadCommand,
		PreviewCommand: previewCommand,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	themeArgs, err := resolveThemeArgs(appConfig.Theme, ummtheme.RenderOverrides{
		Prompt:        "> Search: ",
		Info:          "inline",
		PreviewWindow: "top:60%",
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	args := []string{
		"--ansi",
		"--disabled",
		"--query=" + cfg.Pattern,
		"--delimiter=\t",
		"--with-nth=3..",
		"--preview=" + previewCommand,
	}
	args = append(args, themeArgs...)
	args = append(args, keybindArgs...)
	if !cfg.NoMulti {
		args = append(args, "--multi")
	}

	output, err := execx.InteractiveOutputWithInput(ctx, "", nil, input, errOut, "fzf", args...)
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

func runGitInteractive(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, errOut io.Writer) ([]resultfmt.Result, bool, error) {
	results, err := gitsearch.Aggregate(ctx, cfg, appConfig)
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
	keybindArgs, err := buildBindArgs(appConfig.Keybinds.Git.Bind, ummconfig.KeybindTemplateData{PreviewCommand: previewCommand})
	if err != nil {
		return nil, false, errors.Wrap(err)
	}
	themeArgs, err := resolveThemeArgs(appConfig.Theme, ummtheme.RenderOverrides{
		Prompt:        "> Git: ",
		Info:          "inline",
		PreviewWindow: "top:60%",
	})
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	args := []string{
		"--ansi",
		"--delimiter=\t",
		"--with-nth=3..",
		"--header=" + buildGitHeader(cfg.GitModes),
		"--header-first",
		"--query=" + cfg.Pattern,
		"--preview=" + previewCommand,
	}
	args = append(args, themeArgs...)
	if len(appConfig.Keybinds.Git.ExpectKeys) > 0 {
		args = append(args, "--expect="+strings.Join(appConfig.Keybinds.Git.ExpectKeys, ","))
	}
	args = append(args, keybindArgs...)
	if !cfg.NoMulti {
		args = append(args, "--multi")
	}

	output, err := execx.InteractiveOutputWithInput(ctx, "", nil, bytes.NewReader(buffer.Bytes()), errOut, "fzf", args...)
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
	if len(parts) > 1 && contains(appConfig.Keybinds.Git.ExpectKeys, parts[0]) {
		ctrlO = true
		output = parts[1]
	}

	results, err = resultfmt.DecodeLines(output)
	if err != nil {
		return nil, false, errors.Wrap(err)
	}

	return results, ctrlO, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildBindArgs(binds []string, data ummconfig.KeybindTemplateData) ([]string, error) {
	args := make([]string, 0, len(binds)*2)
	for _, bind := range binds {
		rendered, err := ummconfig.RenderString("bind", bind, data)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		args = append(args, "--bind", rendered)
	}
	return args, nil
}

func buildEmitSearchCommand(exe string, cfg ummruntime.RootConfig) string {
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

func buildEmitReloadArgs(cfg ummruntime.RootConfig) []string {
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

func buildEmitArgs(cfg ummruntime.RootConfig, patternStdin bool) []string {
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

func startNormalSearchInput(ctx context.Context, exe string, cfg ummruntime.RootConfig, errOut io.Writer) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		args := buildEmitArgs(cfg, true)
		err := execx.Run(ctx, "", nil, strings.NewReader(cfg.Pattern), writer, errOut, exe, args...)
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

func resolveThemeArgs(name string, overrides ummtheme.RenderOverrides) ([]string, error) {
	configDir, err := ummconfig.ResolveConfigDir()
	if err != nil {
		return nil, errors.Wrap(err)
	}
	catalog, err := ummtheme.Discover(configDir)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	entry, err := catalog.Resolve(name)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	args, err := ummtheme.Render(entry.Theme, overrides)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return args, nil
}
