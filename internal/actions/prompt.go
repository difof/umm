package actions

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/editor"
	"github.com/difof/umm/internal/resultfmt"
)

func PromptAction(ctx context.Context, appConfig ummconfig.Config, results []resultfmt.Result, isGit bool) error {
	items := buildPromptItems(results, isGit)
	if len(items) == 0 {
		return errors.New("no actions are available for the selected results")
	}

	if _, err := fmt.Fprintln(os.Stderr, promptSelectionSummary(results, isGit)); err != nil {
		return errors.Wrap(err)
	}

	choice, err := promptChoice(items)
	if err != nil {
		return errors.Wrap(err)
	}
	switch choice {
	case "", "cancel":
		return nil
	case "editor":
		targets := results
		if isGit {
			targets = TrackedFileSubset(results)
		} else {
			targets = editorCompatible(results)
		}
		return OpenInEditor(ctx, appConfig, targets)
	case "system":
		targets := results
		if isGit {
			targets = TrackedFileSubset(results)
		}
		return OpenWithSystem(ctx, targets)
	case "stat":
		if isGit {
			return RenderGitStats(os.Stdout, results)
		}
		return RenderPathStats(os.Stdout, cli.StatModeFull, results)
	default:
		return errors.Newf("unsupported action %q", choice)
	}
}

func promptChoice(items []promptItem) (string, error) {
	if choice := os.Getenv("UMM_TEST_OPEN_ASK_CHOICE"); choice != "" {
		return choice, nil
	}

	program := tea.NewProgram(
		newPromptModel(items),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)
	model, err := program.Run()
	if err != nil {
		return "", errors.Wrap(err)
	}

	return model.(promptModel).choice, nil
}

type promptItem struct {
	key         string
	title       string
	description string
}

func (item promptItem) Title() string       { return item.title }
func (item promptItem) Description() string { return item.description }
func (item promptItem) FilterValue() string { return item.title }

type promptModel struct {
	list   list.Model
	choice string
}

func newPromptModel(items []promptItem) promptModel {
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, item)
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	height := len(items) + 6
	if height < 8 {
		height = 8
	}
	model := list.New(listItems, delegate, 72, height)
	model.Title = "Select action"
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetFilteringEnabled(false)
	model.SetShowHelp(false)
	model.DisableQuitKeybindings()

	return promptModel{list: model}
}

func (model promptModel) Init() tea.Cmd {
	return nil
}

func (model promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := model.list.SelectedItem().(promptItem); ok {
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

func (model promptModel) View() string {
	return model.list.View()
}

func buildPromptItems(results []resultfmt.Result, isGit bool) []promptItem {
	items := []promptItem{}
	if len(editorCompatibleForPrompt(results, isGit)) > 0 {
		items = append(items, promptItem{key: "editor", title: "Editor", description: "Open compatible files in $EDITOR"})
	}
	if len(systemCompatible(results, isGit)) > 0 {
		items = append(items, promptItem{key: "system", title: "System", description: "Open compatible results with the system handler"})
	}
	items = append(items, promptItem{key: "stat", title: "Stat", description: "Print stat output"})
	items = append(items, promptItem{key: "cancel", title: "Cancel", description: "Do nothing"})
	return items
}

func editorCompatibleForPrompt(results []resultfmt.Result, isGit bool) []resultfmt.Result {
	if isGit {
		return TrackedFileSubset(results)
	}
	return editorCompatible(results)
}

func editorCompatible(results []resultfmt.Result) []resultfmt.Result {
	filtered := []resultfmt.Result{}
	for _, result := range results {
		if result.Path == "" || result.Kind == resultfmt.KindDir {
			continue
		}
		filtered = append(filtered, result)
	}
	return filtered
}

func systemCompatible(results []resultfmt.Result, isGit bool) []resultfmt.Result {
	if isGit {
		return TrackedFileSubset(results)
	}
	filtered := []resultfmt.Result{}
	for _, result := range results {
		if result.Path == "" {
			continue
		}
		filtered = append(filtered, result)
	}
	return filtered
}

func promptSelectionSummary(results []resultfmt.Result, isGit bool) string {
	lines := []string{}
	if isGit {
		tracked := TrackedFileSubset(results)
		if len(tracked) > 0 {
			lines = append(lines, "Open-compatible tracked files:")
			lines = append(lines, formatPathLines(tracked, 8)...)
		} else {
			lines = append(lines, "Selected git objects:")
			lines = append(lines, formatDisplayLines(results, 8)...)
		}
	} else {
		targets := editorTargets(results)
		if len(targets) > 0 {
			lines = append(lines, "Open-compatible files:")
			lines = append(lines, formatEditorTargetLines(targets, 8)...)
		} else {
			lines = append(lines, "Selected targets:")
			lines = append(lines, formatPathLines(results, 8)...)
		}
	}

	return strings.Join(lines, "\n")
}

func formatPathLines(results []resultfmt.Result, limit int) []string {
	paths := uniquePaths(results)
	if len(paths) == 0 {
		return []string{"  (none)"}
	}

	lines := make([]string, 0, min(len(paths), limit)+1)
	for _, path := range paths[:min(len(paths), limit)] {
		lines = append(lines, "  - "+path)
	}
	if len(paths) > limit {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(paths)-limit))
	}
	return lines
}

func formatEditorTargetLines(targets []editor.Target, limit int) []string {
	if len(targets) == 0 {
		return []string{"  (none)"}
	}

	lines := make([]string, 0, min(len(targets), limit)+1)
	for _, target := range targets[:min(len(targets), limit)] {
		value := target.Path
		if target.Line > 0 {
			value = fmt.Sprintf("%s:%d", target.Path, target.Line)
		}
		lines = append(lines, "  - "+value)
	}
	if len(targets) > limit {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(targets)-limit))
	}

	return lines
}

func formatDisplayLines(results []resultfmt.Result, limit int) []string {
	if len(results) == 0 {
		return []string{"  (none)"}
	}

	lines := make([]string, 0, min(len(results), limit)+1)
	for _, result := range results[:min(len(results), limit)] {
		value := strings.TrimSpace(result.Display)
		if value == "" {
			value = strings.TrimSpace(result.Summary)
		}
		if value == "" {
			continue
		}
		lines = append(lines, "  - "+value)
	}
	if len(results) > limit {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(results)-limit))
	}
	if len(lines) == 0 {
		return []string{"  (none)"}
	}
	return lines
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func editorTargets(results []resultfmt.Result) []editor.Target {
	seen := map[string]struct{}{}
	targets := make([]editor.Target, 0, len(results))
	for _, result := range results {
		if result.Path == "" || result.Kind == resultfmt.KindDir {
			continue
		}
		if _, ok := seen[result.Path]; ok {
			continue
		}
		seen[result.Path] = struct{}{}
		targets = append(targets, editor.Target{Path: result.Path, Line: result.Line})
	}

	return targets
}
