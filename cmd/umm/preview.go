package umm

import (
	"github.com/difof/errors"
	"github.com/difof/umm/internal/app"
	"github.com/spf13/cobra"
)

func BuildPreviewCmd() *cobra.Command {
	previewCmd := &cobra.Command{
		Use:    "preview <mode> <meta>",
		Short:  "Internal preview helper",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreviewCmd(cmd, args[0], args[1])
		},
	}

	return previewCmd
}

func runPreviewCmd(cmd *cobra.Command, mode string, meta string) (err error) {
	defer errors.Recover(&err)

	if err := app.RunPreview(cmd.Context(), mode, meta, cmd.OutOrStdout()); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
