package app

import (
	"context"
	"os"

	"github.com/difof/umm/internal/actions"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/resultfmt"
)

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
