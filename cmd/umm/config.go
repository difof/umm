package umm

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/spf13/cobra"
)

func BuildConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage configuration",
	}

	configCmd.AddCommand(
		buildConfigShowCmd(),
		buildConfigDumpCmd(),
		buildConfigCheckCmd(),
	)

	return configCmd
}

func buildConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the effective configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShowCmd(cmd)
		},
	}
}

func buildConfigDumpCmd() *cobra.Command {
	force := false
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Write the default user configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigDumpCmd(cmd, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config without prompting")
	return cmd
}

func buildConfigCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate the user configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigCheckCmd(cmd)
		},
	}
}

func runConfigShowCmd(cmd *cobra.Command) (err error) {
	defer errors.Recover(&err)

	loaded := errors.MustResult(ummconfig.LoadEffective())
	data := errors.MustResult(ummconfig.Marshal(loaded.Config))
	_, err = cmd.OutOrStdout().Write(data)
	return errors.Wrap(err)
}

func runConfigDumpCmd(cmd *cobra.Command, force bool) (err error) {
	defer errors.Recover(&err)

	path := errors.MustResult(ummconfig.ResolveWritePath())
	if _, statErr := os.Stat(path); statErr == nil && !force {
		if !allowTestConfirm() && (!isTTY(os.Stdin) || !isTTY(os.Stderr)) {
			return errors.New("config file exists; rerun with --force to overwrite outside an interactive terminal")
		}
		confirmed := errors.MustResult(confirmOverwrite(path))
		if !confirmed {
			return nil
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return errors.Wrap(statErr)
	}

	if err := ummconfig.WriteDefaults(path); err != nil {
		return errors.Wrap(err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", path)
	return errors.Wrap(err)
}

func runConfigCheckCmd(cmd *cobra.Command) (err error) {
	defer errors.Recover(&err)

	report := errors.MustResult(ummconfig.Check(cmd.Context()))
	if !report.UserExists {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "No user config file found at %s\n", report.Path)
		return errors.Wrap(err)
	}

	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning); err != nil {
			return errors.Wrap(err)
		}
	}
	for _, item := range report.Errors {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "error: %s\n", item); err != nil {
			return errors.Wrap(err)
		}
	}

	if !report.Valid() {
		return errors.New("configuration is invalid")
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Config is valid: %s\n", report.Path)
	return errors.Wrap(err)
}

func isTTY(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

type confirmItem struct {
	key   string
	title string
}

func (item confirmItem) Title() string       { return item.title }
func (item confirmItem) Description() string { return "" }
func (item confirmItem) FilterValue() string { return item.title }

type confirmModel struct {
	list   list.Model
	choice string
}

func confirmOverwrite(path string) (bool, error) {
	if choice := os.Getenv("UMM_TEST_CONFIG_DUMP_CONFIRM"); choice != "" {
		return choice == "overwrite", nil
	}

	items := []list.Item{
		confirmItem{key: "overwrite", title: "Overwrite"},
		confirmItem{key: "cancel", title: "Cancel"},
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	model := list.New(items, delegate, 80, 8)
	model.Title = "Overwrite existing config? " + path
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetFilteringEnabled(false)
	model.SetShowHelp(false)
	model.DisableQuitKeybindings()

	program := tea.NewProgram(confirmModel{list: model}, tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr))
	result, err := program.Run()
	if err != nil {
		return false, errors.Wrap(err)
	}

	return result.(confirmModel).choice == "overwrite", nil
}

func allowTestConfirm() bool {
	return os.Getenv("UMM_TEST_CONFIG_DUMP_CONFIRM") != ""
}

func (model confirmModel) Init() tea.Cmd {
	return nil
}

func (model confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := model.list.SelectedItem().(confirmItem); ok {
				model.choice = item.key
			}
			return model, tea.Quit
		case "esc", "ctrl+c", "q":
			model.choice = "cancel"
			return model, tea.Quit
		}
	}

	var cmd tea.Cmd
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}

func (model confirmModel) View() string {
	return model.list.View()
}
