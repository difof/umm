package app

import (
	"context"
	"io"
	"os"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/preview"
	ummruntime "github.com/difof/umm/internal/runtime"
	"github.com/difof/umm/internal/search"
)

func RunRoot(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config) (err error) {
	return RunRootWithIO(ctx, cfg, appConfig, nil, nil, nil)
}

func RunRootWithIO(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, in io.Reader, out io.Writer, errOut io.Writer) (err error) {
	defer errors.Recover(&err)
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}

	if cfg.SearchMode == ummruntime.SearchModeGit {
		return runGit(ctx, cfg, appConfig, in, out, errOut)
	}

	return runNormal(ctx, cfg, appConfig, in, out, errOut)
}

func RunPreview(ctx context.Context, appConfig ummconfig.Config, mode string, meta string, out io.Writer) error {
	return preview.Run(ctx, appConfig, mode, meta, out)
}

func EmitSearch(ctx context.Context, cfg ummruntime.RootConfig, out io.Writer, errOut io.Writer) error {
	return search.EmitLinesWithErrorOutput(ctx, cfg, cfg.Pattern, out, errOut)
}
