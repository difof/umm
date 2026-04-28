package search

import (
	"context"
	"io"
	"sort"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/resultfmt"
)

type resultEmitter func(resultfmt.Result) error

const (
	matchStartANSI = "\x1b[1;33m"
	matchResetANSI = "\x1b[0m"
)

func Query(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	return QueryWithErrorOutput(ctx, cfg, query, strict, nil)
}

func QueryWithErrorOutput(ctx context.Context, cfg cli.RootConfig, query string, strict bool, errOut io.Writer) ([]resultfmt.Result, error) {
	results := []resultfmt.Result{}
	err := emitQuery(ctx, cfg, query, strict, errOut, func(result resultfmt.Result) error {
		results = append(results, result)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if cfg.SearchMode == cli.SearchModeOnlyDirname {
		sort.Slice(results, func(i int, j int) bool {
			return results[i].Display < results[j].Display
		})
	}

	return results, nil
}

func EmitLines(ctx context.Context, cfg cli.RootConfig, query string, out io.Writer) error {
	return EmitLinesWithErrorOutput(ctx, cfg, query, out, nil)
}

func EmitLinesWithErrorOutput(ctx context.Context, cfg cli.RootConfig, query string, out io.Writer, errOut io.Writer) error {
	return emitQuery(ctx, cfg, query, false, errOut, func(result resultfmt.Result) error {
		result.Display = highlightDisplay(result.Display, query)

		line, err := resultfmt.EncodeLine(result)
		if err != nil {
			return errors.Wrap(err)
		}

		if _, err := io.WriteString(out, line+"\n"); err != nil {
			return errors.Wrap(err)
		}

		return nil
	})
}

func emitQuery(ctx context.Context, cfg cli.RootConfig, query string, strict bool, errOut io.Writer, emit resultEmitter) error {
	switch cfg.SearchMode {
	case cli.SearchModeOnlyDirname:
		return emitDirnames(ctx, cfg, query, strict, errOut, emit)
	case cli.SearchModeOnlyFilename:
		return emitFilenames(ctx, cfg, query, strict, emit)
	case cli.SearchModeDefault:
		return emitDefault(ctx, cfg, query, strict, emit)
	default:
		return errors.Newf("unsupported search mode %q", cfg.SearchMode)
	}
}

func emitDefault(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	if query == "" {
		if cfg.NoFilename {
			return nil
		}

		return emitFilenames(ctx, cfg, query, strict, emit)
	}

	if err := emitContent(ctx, cfg, query, strict, emit); err != nil {
		return errors.Wrap(err)
	}
	if cfg.NoFilename {
		return nil
	}

	return emitFilenames(ctx, cfg, query, strict, emit)
}
