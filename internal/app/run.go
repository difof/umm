package app

import (
	"context"
	"os"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/actions"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/resultfmt"
	"github.com/difof/umm/internal/search"
)

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

func buildGitHeader(modes []string) string {
	if len(modes) == 0 {
		return "Git modes: all"
	}

	return "Git modes: " + strings.Join(modes, ", ")
}
