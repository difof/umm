package app

import (
	"context"
	"io"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/preview"
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
