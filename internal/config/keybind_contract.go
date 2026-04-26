package config

import "strings"

type GitKeybindTemplateData struct {
	PreviewCommand string
}

type KeybindModeName string

const (
	KeybindModeNormal KeybindModeName = "normal"
	KeybindModeGit    KeybindModeName = "git"
)

type keybindModeContract struct {
	templateVariables []string
	templateData      any
}

var keybindModeContracts = map[KeybindModeName]keybindModeContract{
	KeybindModeNormal: {
		templateVariables: []string{"ReloadCommand", "PreviewCommand"},
		templateData: KeybindTemplateData{
			ReloadCommand:  "umm __emit-search --pattern {q}",
			PreviewCommand: "umm preview {1} {2}",
		},
	},
	KeybindModeGit: {
		templateVariables: []string{"PreviewCommand"},
		templateData:      GitKeybindTemplateData{PreviewCommand: "umm preview {1} {2}"},
	},
}

func KeybindTemplateVariables(mode KeybindModeName) []string {
	variables := keybindContract(mode).templateVariables
	return append([]string(nil), variables...)
}

func KeybindTemplateVariablesText(mode KeybindModeName) string {
	return strings.Join(KeybindTemplateVariables(mode), ", ") + "."
}

func KeybindBindTemplateHelp(mode KeybindModeName) string {
	if mode == KeybindModeGit {
		return "Bind strings may reference {{.PreviewCommand}} only."
	}
	return "Bind strings may reference {{.ReloadCommand}} and {{.PreviewCommand}}."
}

func KeybindTemplateDataForMode(mode KeybindModeName) any {
	return keybindContract(mode).templateData
}

func keybindContract(mode KeybindModeName) keybindModeContract {
	if contract, ok := keybindModeContracts[mode]; ok {
		return contract
	}
	return keybindModeContracts[KeybindModeNormal]
}
