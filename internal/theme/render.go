package theme

import (
	"sort"
	"strings"

	"github.com/difof/errors"
)

type RenderOverrides struct {
	Prompt        string
	Info          string
	PreviewWindow string
}

func Render(theme Theme, overrides RenderOverrides) ([]string, error) {
	if err := Validate(theme); err != nil {
		return nil, errors.Wrap(err)
	}

	args := []string{}
	appendFlag := func(flag string, value string) {
		if value == "" {
			return
		}
		args = append(args, flag+"="+value)
	}

	appendFlag("--style", string(theme.FZF.Style))
	appendFlag("--layout", string(theme.FZF.Layout))
	appendFlag("--height", theme.FZF.Height)
	appendFlag("--min-height", theme.FZF.MinHeight)
	appendFlag("--popup", theme.FZF.Popup)
	appendFlag("--margin", theme.FZF.Margin)
	appendFlag("--padding", theme.FZF.Padding)
	appendFlag("--border", string(theme.FZF.Border))
	appendFlag("--list-border", string(theme.FZF.ListBorder))
	appendFlag("--input-border", string(theme.FZF.InputBorder))
	appendFlag("--preview-border", string(theme.FZF.PreviewBorder))
	appendFlag("--header-border", string(theme.FZF.HeaderBorder))
	appendFlag("--header-lines-border", string(theme.FZF.HeaderLinesBorder))
	appendFlag("--footer-border", string(theme.FZF.FooterBorder))
	appendFlag("--border-label", theme.FZF.BorderLabel)
	appendFlag("--border-label-pos", theme.FZF.BorderLabelPos)
	appendFlag("--list-label", theme.FZF.ListLabel)
	appendFlag("--list-label-pos", theme.FZF.ListLabelPos)
	appendFlag("--input-label", theme.FZF.InputLabel)
	appendFlag("--input-label-pos", theme.FZF.InputLabelPos)
	appendFlag("--header-label", theme.FZF.HeaderLabel)
	appendFlag("--header-label-pos", theme.FZF.HeaderLabelPos)
	appendFlag("--footer-label", theme.FZF.FooterLabel)
	appendFlag("--footer-label-pos", theme.FZF.FooterLabelPos)
	appendFlag("--preview-label", theme.FZF.PreviewLabel)
	appendFlag("--preview-label-pos", theme.FZF.PreviewLabelPos)
	appendFlag("--info", firstNonEmpty(theme.FZF.Info, overrides.Info))
	appendFlag("--prompt", firstNonEmpty(theme.FZF.Prompt, overrides.Prompt))
	appendFlag("--ghost", theme.FZF.Ghost)
	appendFlag("--separator", theme.FZF.Separator)
	appendFlag("--pointer", theme.FZF.Pointer)
	appendFlag("--marker", theme.FZF.Marker)
	appendFlag("--marker-multi-line", theme.FZF.MarkerMultiLine)
	appendFlag("--gutter", theme.FZF.Gutter)
	appendFlag("--gutter-raw", theme.FZF.GutterRaw)
	appendFlag("--scrollbar", theme.FZF.Scrollbar)
	appendFlag("--ellipsis", theme.FZF.Ellipsis)
	appendFlag("--wrap-sign", theme.FZF.WrapSign)
	appendFlag("--preview-wrap-sign", theme.FZF.PreviewWrapSign)
	appendFlag("--preview-window", firstNonEmpty(theme.FZF.PreviewWindow, overrides.PreviewWindow))

	if color := renderColor(theme.FZF.Color); color != "" {
		appendFlag("--color", color)
	}

	return args, nil
}

func renderColor(theme ColorTheme) string {
	parts := []string{}
	if theme.Base != "" {
		parts = append(parts, string(theme.Base))
	}
	keys := make([]string, 0, len(theme.Entries))
	for key := range theme.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, key+":"+theme.Entries[key])
	}
	return strings.Join(parts, ",")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
