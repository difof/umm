package main

import (
	"fmt"
	"os"

	difoferrors "github.com/difof/errors"
	"github.com/difof/umm/cmd/umm"
	"github.com/difof/umm/internal/version"
)

func firstErrorMessage(err error) string {
	entry := difoferrors.Expand(err)
	if entry == nil {
		return ""
	}

	var walk func(*difoferrors.ErrorEntry) string
	walk = func(entry *difoferrors.ErrorEntry) string {
		if entry == nil {
			return ""
		}

		if entry.Resolved.Message != "" {
			return entry.Resolved.Message
		}

		for _, child := range entry.Children {
			if message := walk(child); message != "" {
				return message
			}
		}

		return ""
	}

	message := walk(entry)
	if message == "" {
		return err.Error()
	}

	return message
}

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	rootCmd := umm.BuildRootCmd(workingDir)
	rootCmd.AddCommand(
		umm.BuildPreviewCmd(),
		umm.BuildEmitSearchCmd(),
	)

	rootCmd.Version = version.VersionString()
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", firstErrorMessage(err))
		os.Exit(1)
	}
}
