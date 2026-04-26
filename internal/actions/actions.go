package actions

import (
	"context"
	"io"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/editor"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/gitsearch"
	"github.com/difof/umm/internal/resultfmt"
)

func OpenInEditor(ctx context.Context, appConfig ummconfig.Config, results []resultfmt.Result, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	targets := editorTargets(results)

	if len(targets) == 0 {
		return errors.New("no compatible file results to open in editor")
	}

	command, err := editor.Resolve(appConfig)
	if err != nil {
		return errors.Wrap(err)
	}

	if err := deps.Require(command.Name, "editor open"); err != nil {
		return errors.Wrap(err)
	}

	return editor.Open(ctx, command, targets, stdin, stdout, stderr)
}

func OpenWithSystem(ctx context.Context, results []resultfmt.Result) error {
	command, err := deps.SystemOpenCommand()
	if err != nil {
		return errors.Wrap(err)
	}
	if err := deps.Require(command, "system open"); err != nil {
		return errors.Wrap(err)
	}

	paths := uniquePaths(results)
	if len(paths) == 0 {
		return errors.New("no compatible results to open with system handler")
	}

	for _, path := range paths {
		if err := execx.Run(ctx, "", nil, nil, io.Discard, io.Discard, command, path); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func PrintPaths(out io.Writer, results []resultfmt.Result) error {
	for _, result := range results {
		if result.Path == "" {
			continue
		}
		if _, err := io.WriteString(out, result.Path+"\n"); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func TrackedFileSubset(results []resultfmt.Result) []resultfmt.Result {
	return gitsearch.TrackedFileSubset(results)
}

func FirstTrackedFile(results []resultfmt.Result) []resultfmt.Result {
	tracked := TrackedFileSubset(results)
	if len(tracked) == 0 {
		return nil
	}
	return tracked[:1]
}
