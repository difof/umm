package app

import (
	"context"
	"io"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/actions"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/resultfmt"
	ummruntime "github.com/difof/umm/internal/runtime"
	"github.com/difof/umm/internal/search"
)

func runNormal(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, in io.Reader, out io.Writer, errOut io.Writer) error {
	if cfg.UsesRG() {
		if err := deps.Require("rg", "normal search"); err != nil {
			return errors.Wrap(err)
		}
	}

	if cfg.NoUI {
		results, err := search.QueryWithErrorOutput(ctx, cfg, cfg.Pattern, true, errOut)
		if err != nil {
			return errors.Wrap(err)
		}
		if len(results) == 0 {
			return errors.Newf("no matches found for pattern %q", cfg.Pattern)
		}
		return runNormalNoUI(ctx, cfg, appConfig, results, in, out, errOut)
	}

	if err := deps.Require("fzf", "interactive search"); err != nil {
		return errors.Wrap(err)
	}

	results, err := runNormalInteractive(ctx, cfg, appConfig, errOut)
	if err != nil {
		return errors.Wrap(err)
	}
	if len(results) == 0 {
		return nil
	}

	return handleNormalSelection(ctx, cfg, appConfig, results, false, in, out, errOut)
}

func runGit(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, in io.Reader, out io.Writer, errOut io.Writer) error {
	if err := deps.Require("git", "git search"); err != nil {
		return errors.Wrap(err)
	}
	if err := gitsearch.ValidateRepo(ctx, cfg.Root); err != nil {
		return errors.Wrap(err)
	}

	if cfg.NoUI {
		results, err := gitsearch.Query(ctx, cfg, appConfig, cfg.Pattern, true)
		if err != nil {
			return errors.Wrap(err)
		}
		if len(results) == 0 {
			return errors.Newf("no git matches found for pattern %q", cfg.Pattern)
		}
		return runGitNoUI(ctx, cfg, appConfig, results, in, out, errOut)
	}

	if err := deps.Require("fzf", "interactive git search"); err != nil {
		return errors.Wrap(err)
	}

	results, ctrlO, err := runGitInteractive(ctx, cfg, appConfig, errOut)
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
		return actions.OpenInEditor(ctx, appConfig, tracked, in, out, errOut)
	}

	return handleGitSelection(ctx, cfg, appConfig, results, false, in, out, errOut)
}

func runNormalNoUI(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, results []resultfmt.Result, in io.Reader, out io.Writer, errOut io.Writer) error {
	if cfg.Action == ummruntime.ActionStat {
		return actions.RenderPathStats(out, cfg.StatMode, results)
	}

	if cfg.Action == ummruntime.ActionAsk {
		return handleNormalSelection(ctx, cfg, appConfig, results, true, in, out, errOut)
	}

	if cfg.Action == ummruntime.ActionSystem {
		return handleNormalSelection(ctx, cfg, appConfig, results[:1], true, in, out, errOut)
	}

	return handleNormalSelection(ctx, cfg, appConfig, results[:1], true, in, out, errOut)
}

func runGitNoUI(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, results []resultfmt.Result, in io.Reader, out io.Writer, errOut io.Writer) error {
	if cfg.Action == ummruntime.ActionAsk {
		return handleGitSelection(ctx, cfg, appConfig, results, true, in, out, errOut)
	}

	if cfg.Action == ummruntime.ActionSystem {
		firstTracked := actions.FirstTrackedFile(results)
		if len(firstTracked) == 0 {
			return errors.New("no tracked file results available for open action")
		}
		return handleGitSelection(ctx, cfg, appConfig, firstTracked, true, in, out, errOut)
	}

	return handleGitSelection(ctx, cfg, appConfig, results, true, in, out, errOut)
}

func buildGitHeader(modes []string) string {
	if len(modes) == 0 {
		return "Git modes: all"
	}

	return "Git modes: " + strings.Join(modes, ", ")
}
