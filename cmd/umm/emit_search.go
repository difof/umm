package umm

import (
	"io"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/app"
	"github.com/spf13/cobra"
)

func BuildEmitSearchCmd() *cobra.Command {
	options := rawRootOptions{}
	patternStdin := false

	emitCmd := &cobra.Command{
		Use:    "__emit-search",
		Short:  "Internal search emitter for fzf reloads",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEmitSearchCmd(cmd, options)
		},
	}

	emitCmd.Flags().StringVarP(&options.Root, "root", "r", ".", "root search path")
	emitCmd.Flags().StringVarP(&options.Pattern, "pattern", "p", "", "regexp search pattern")
	emitCmd.Flags().StringSliceVarP(&options.Excludes, "exclude", "e", nil, "file/dir exclusion glob")
	emitCmd.Flags().BoolVarP(&options.Hidden, "hidden", "a", false, "search hidden and ignored files")
	emitCmd.Flags().BoolVar(&options.NoFilename, "no-filename", false, "do not include filenames in search")
	emitCmd.Flags().BoolVarP(&options.OnlyFilename, "only-filename", "f", false, "only search file paths")
	emitCmd.Flags().BoolVarP(&options.OnlyDirname, "only-dirname", "d", false, "only search dir names")
	emitCmd.Flags().UintVarP(&options.MaxDepth, "max-depth", "m", 0, "max search depth. zero means no limit")
	emitCmd.Flags().BoolVar(&patternStdin, "pattern-stdin", false, "read the search pattern from stdin")
	_ = emitCmd.Flags().MarkHidden("pattern-stdin")

	return emitCmd
}

func runEmitSearchCmd(cmd *cobra.Command, options rawRootOptions) (err error) {
	defer errors.Recover(&err)

	patternStdin, err := cmd.Flags().GetBool("pattern-stdin")
	if err != nil {
		return errors.Wrap(err)
	}
	if patternStdin {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return errors.Wrap(err)
		}
		options.Pattern = strings.TrimRight(string(data), "\r\n")
	}

	config := errors.MustResult(normalizeEmitterOptions(options))
	if err := app.EmitSearch(cmd.Context(), config, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
