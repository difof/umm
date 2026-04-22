package app

import (
	"context"
	"io"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/preview"
	"github.com/difof/umm/internal/search"
)

func RunRoot(ctx context.Context, cfg cli.RootConfig, appConfig ummconfig.Config) (err error) {
	defer errors.Recover(&err)

	if cfg.SearchMode == cli.SearchModeGit {
		return runGit(ctx, cfg, appConfig)
	}

	return runNormal(ctx, cfg, appConfig)
}

func RunPreview(ctx context.Context, appConfig ummconfig.Config, mode string, meta string, out io.Writer) error {
	return preview.Run(ctx, appConfig, mode, meta, out)
}

func EmitSearch(ctx context.Context, cfg cli.RootConfig, out io.Writer) error {
	return search.EmitLines(ctx, cfg, cfg.Pattern, out)
}
