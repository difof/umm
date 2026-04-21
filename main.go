package main

import (
	"fmt"
	"os"

	"github.com/difof/errors"
	"github.com/difof/umm/cmd/umm"
	"github.com/difof/umm/internal/version"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Stacktrace(err))
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
		errcfg := []errors.StacktraceOption{
			errors.StacktraceWithFunctionFormat(errors.StacktraceFunctionFuncOnly),
			errors.StacktraceWithTrimFilePath(true),
		}

		fmt.Fprintln(os.Stderr, errors.Stacktrace(err, errcfg...))
		os.Exit(1)
	}
}
