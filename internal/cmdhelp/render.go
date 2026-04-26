package cmdhelp

import (
	"io"
	"strings"
)

type RenderOptions struct {
	Color bool
}

func Render(doc Document, opts RenderOptions) string {
	style := helpStyle{color: opts.Color}
	lines := []string{}

	if doc.Title != "" {
		lines = append(lines, style.heading(doc.Title))
	}
	lines = append(lines, doc.Intro...)

	if doc.Example != nil {
		lines = appendSectionGap(lines)
		if doc.Example.Title != "" {
			lines = append(lines, style.heading(doc.Example.Title))
		}
		for _, line := range doc.Example.Lines {
			lines = append(lines, style.code(line))
		}
	}

	for _, section := range doc.Sections {
		lines = appendSectionGap(lines)
		if section.Title != "" {
			lines = append(lines, style.heading(section.Title))
		}
		lines = append(lines, section.Body...)
		for _, line := range section.Code {
			lines = append(lines, style.code(line))
		}
		for _, extra := range section.Extras {
			lines = append(lines, style.labeled(extra.Label, extra.Text))
		}
		if len(section.Fields) > 0 && (len(section.Body) > 0 || len(section.Code) > 0 || len(section.Extras) > 0) {
			lines = append(lines, "")
		}
		for _, field := range section.Fields {
			lines = append(lines, renderField(style, field)...)
		}
	}

	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func renderField(style helpStyle, field Field) []string {
	lines := []string{style.field(field.Path)}
	if field.What != "" {
		lines = append(lines, style.labeled("what", field.What))
	}
	if field.Values != "" {
		lines = append(lines, style.labeled("values", field.Values))
	}
	for _, extra := range field.Extras {
		lines = append(lines, style.labeled(extra.Label, extra.Text))
	}
	return append(lines, "")
}

func appendSectionGap(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	if lines[len(lines)-1] == "" {
		return lines
	}
	return append(lines, "")
}

func Write(out io.Writer, doc Document) error {
	_, err := io.WriteString(out, Render(doc, RenderOptions{Color: isTerminalWriter(out)}))
	return err
}

type helpStyle struct {
	color bool
}

func (s helpStyle) heading(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[1;36m" + value + "\x1b[0m"
}

func (s helpStyle) field(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[1;33m" + value + "\x1b[0m"
}

func (s helpStyle) labeled(label string, text string) string {
	prefix := "  " + strings.TrimSuffix(label, ":") + ":"
	if !s.color {
		return prefix + " " + text
	}
	return "  \x1b[2m" + strings.TrimSuffix(label, ":") + ":\x1b[0m " + text
}

func (s helpStyle) code(value string) string {
	if !s.color {
		return value
	}
	return "\x1b[38;5;110m" + value + "\x1b[0m"
}
