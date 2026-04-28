package app

import (
	"context"
	"io"

	"github.com/difof/umm/internal/actions"
	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/resultfmt"
)

func handleNormalSelection(ctx context.Context, cfg cli.RootConfig, appConfig ummconfig.Config, results []resultfmt.Result, noUI bool, in io.Reader, out io.Writer, errOut io.Writer) error {
	if len(results) == 0 {
		return nil
	}

	switch cfg.Action {
	case cli.ActionAsk:
		return actions.PromptAction(ctx, appConfig, results, false, in, out, errOut)
	case cli.ActionSystem:
		return actions.OpenWithSystem(ctx, results)
	case cli.ActionStat:
		return actions.RenderPathStats(out, cfg.StatMode, results)
	default:
		if cfg.SearchMode == cli.SearchModeOnlyDirname {
			if noUI {
				return actions.PrintPaths(out, results[:1])
			}
			return actions.PrintPaths(out, results)
		}

		editorResults := results
		if noUI {
			editorResults = results[:1]
		}
		return actions.OpenInEditor(ctx, appConfig, editorResults, in, out, errOut)
	}
}

func handleGitSelection(ctx context.Context, cfg cli.RootConfig, appConfig ummconfig.Config, results []resultfmt.Result, noUI bool, in io.Reader, out io.Writer, errOut io.Writer) error {
	switch cfg.Action {
	case cli.ActionAsk:
		return actions.PromptAction(ctx, appConfig, results, true, in, out, errOut)
	case cli.ActionSystem:
		tracked := actions.TrackedFileSubset(results)
		if noUI && len(tracked) > 1 {
			tracked = tracked[:1]
		}
		return actions.OpenWithSystem(ctx, tracked)
	case cli.ActionStat, cli.ActionDefault:
		return actions.RenderGitStats(out, results)
	default:
		return actions.RenderGitStats(out, results)
	}
}
