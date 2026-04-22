package umm

import (
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/app"
	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/spf13/cobra"
)

func BuildRootCmd(workingDir string) *cobra.Command {
	options := cli.RawRootOptions{}

	rootCmd := &cobra.Command{
		Use:   "umm",
		Short: "Ultimate Multi-file Matcher",
		Long: `umm is a wrapper around ripgrep and fzf for straightforward fuzzy finding files, directories, and git objects.

Use "umm --help" for more information.`,
		Example: strings.Join([]string{
			"  umm",
			"  umm --root ~/src --pattern TODO",
			"  umm --root ~/src --pattern root\\.go --only-filename --no-ui",
			"  umm --root ~/src --pattern cmd --only-dirname",
			"  umm --root ~/src --pattern TODO --only-stat lite",
			"  umm --root ~/repo --git --git-mode commit,tracked",
			"  umm --root ~/repo --git --no-ui --pattern 'tag:\\s+v1'",
		}, "\n"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRootCmd(cmd, options)
		},
	}

	rootCmd.Flags().StringVarP(&options.Root, "root", "r", workingDir, "root search path")
	rootCmd.Flags().StringVarP(&options.Pattern, "pattern", "p", "", "regexp initial pattern")
	rootCmd.Flags().StringSliceVarP(&options.Excludes, "exclude", "e", nil, "file/dir exclusion glob")
	rootCmd.Flags().BoolVarP(&options.Hidden, "hidden", "a", false, "search hidden and ignored files")
	rootCmd.Flags().BoolVar(&options.NoFilename, "no-filename", false, "do not include filenames in search")
	rootCmd.Flags().BoolVarP(&options.OnlyFilename, "only-filename", "f", false, "only search file paths")
	rootCmd.Flags().BoolVarP(&options.OnlyDirname, "only-dirname", "d", false, "only search dir names")
	rootCmd.Flags().BoolVarP(&options.Git, "git", "g", false, "git object search mode")
	rootCmd.Flags().StringSliceVar(&options.GitModes, "git-mode", nil, "git search modes (commit,branch,tags,reflog,stash,tracked)")
	rootCmd.Flags().UintVarP(&options.MaxDepth, "max-depth", "m", 0, "max search depth. zero means no limit")
	rootCmd.Flags().BoolVarP(&options.NoUI, "no-ui", "n", false, "run without the interactive search picker")
	rootCmd.Flags().BoolVarP(&options.NoMulti, "no-multi", "s", false, "disable multi-select")
	rootCmd.Flags().BoolVarP(&options.OpenAsk, "open-ask", "q", false, "after selection, show an action picker")
	rootCmd.Flags().BoolVarP(&options.OpenSys, "open-sys", "o", false, "open selected result using the system handler")
	rootCmd.Flags().StringVar(&options.OnlyStat, "only-stat", "", "show stat output instead of opening (full,lite,list)")

	_ = rootCmd.RegisterFlagCompletionFunc("git-mode", cobra.FixedCompletions(cli.AllGitModes, cobra.ShellCompDirectiveNoFileComp))
	_ = rootCmd.RegisterFlagCompletionFunc("only-stat", cobra.FixedCompletions(cli.AllStatModes, cobra.ShellCompDirectiveNoFileComp))

	return rootCmd
}

func runRootCmd(cmd *cobra.Command, options cli.RawRootOptions) (err error) {
	defer errors.Recover(&err)

	loaded := errors.MustResult(ummconfig.LoadEffective())
	for _, warning := range ummconfig.RuntimeWarnings(loaded.Config) {
		_, _ = cmd.ErrOrStderr().Write([]byte("warning: " + warning + "\n"))
	}
	options.DefaultGitModes = loaded.Config.Git.DefaultModes
	options.GitModesExplicit = cmd.Flags().Changed("git-mode")

	runtime := errors.MustResult(cli.NormalizeRootOptions(options))
	if err := app.RunRoot(cmd.Context(), runtime, loaded.Config); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
