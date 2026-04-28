package umm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cmdhelp"
	ummconfig "github.com/difof/umm/internal/config"
	ummtheme "github.com/difof/umm/internal/theme"
	"github.com/spf13/cobra"
)

func BuildThemeCmd() *cobra.Command {
	themeCmd := &cobra.Command{
		Use:   "theme",
		Short: "Inspect and manage fzf themes",
		Long: strings.Join([]string{
			"Inspect and manage fzf themes.",
			"",
			"Use \"umm theme --help\" to see the command help plus the full theme file schema reference.",
		}, "\n"),
	}
	cmdhelp.AttachAppendix(themeCmd, themeHelpDoc())

	themeCmd.AddCommand(
		buildThemeListCmd(),
		buildThemeSetCmd(),
		buildThemeDumpCmd(),
	)

	return themeCmd
}

func buildThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in and user themes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runThemeListCmd(cmd)
		},
	}
}

func buildThemeSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name>",
		Short: "Set the active theme by exact name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeSetCmd(cmd, args[0])
		},
	}
}

func buildThemeDumpCmd() *cobra.Command {
	force := false
	cmd := &cobra.Command{
		Use:   "dump <name>",
		Short: "Dump a built-in theme into the user theme directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeDumpCmd(cmd, args[0], force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing user theme file")
	return cmd
}

func runThemeListCmd(cmd *cobra.Command) (err error) {
	defer errors.Recover(&err)

	selectedTheme := ""
	if loaded, loadErr := ummconfig.LoadEffective(); loadErr == nil {
		selectedTheme = loaded.Config.Theme
	} else {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "warning: active theme could not be determined: %v\n", loadErr); err != nil {
			return errors.Wrap(err)
		}
	}
	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	catalog := errors.MustResult(ummtheme.Discover(configDir))

	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "name\tvariant\torigin\tstatus")
	for _, entry := range catalog.Entries() {
		statuses := themeStatuses(entry, selectedTheme)
		_, err = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", entry.Name, entry.Variant, entry.Origin, strings.Join(statuses, ","))
		if err != nil {
			return errors.Wrap(err)
		}
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func runThemeSetCmd(cmd *cobra.Command, name string) (err error) {
	defer errors.Recover(&err)

	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	catalog := errors.MustResult(ummtheme.Discover(configDir))
	_, err = catalog.Resolve(name)
	if err != nil {
		return errors.Wrap(err)
	}

	path := errors.MustResult(ummconfig.SetTheme(name))
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", path)
	return errors.Wrap(err)
}

func runThemeDumpCmd(cmd *cobra.Command, name string, force bool) (err error) {
	defer errors.Recover(&err)

	builtins := errors.MustResult(ummtheme.LoadBuiltins())
	var builtin *ummtheme.BuiltinFile
	for i := range builtins {
		if builtins[i].Theme.Name == name {
			builtin = &builtins[i]
			break
		}
	}
	if builtin == nil {
		return errors.Newf("built-in theme %q was not found", name)
	}

	configDir := errors.MustResult(ummconfig.ResolveConfigDir())
	path := filepath.Join(ummtheme.UserDir(configDir), name+".yml")
	if _, statErr := os.Stat(path); statErr == nil && !force {
		return errors.Newf("theme file exists: %s (rerun with --force to overwrite)", path)
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return errors.Wrap(statErr)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err)
	}
	if err := os.WriteFile(path, builtin.Raw, 0o644); err != nil {
		return errors.Wrap(err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", path)
	return errors.Wrap(err)
}

func themeStatuses(entry ummtheme.Entry, selectedTheme string) []string {
	statuses := []string{}
	if entry.Effective && entry.Name == selectedTheme {
		statuses = append(statuses, "active")
	}
	if entry.Name == ummtheme.DefaultName {
		statuses = append(statuses, "default")
	}
	if entry.Effective && entry.Origin == ummtheme.OriginUser {
		statuses = append(statuses, "effective")
	}
	if entry.Shadowed {
		statuses = append(statuses, "shadowed")
	}
	if entry.Invalid {
		statuses = append(statuses, "invalid")
	}
	return statuses
}
