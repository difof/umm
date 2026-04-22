package config

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/difof/errors"
)

type PathTemplateData struct {
	Path      string
	Line      int
	HasLine   bool
	StartLine int
	EndLine   int
	LineRange string
}

type DiffTemplateData struct {
	Repo    string
	GitType string
	GitRef  string
	Path    string
	Display string
	Summary string
}

type KeybindTemplateData struct {
	ReloadCommand  string
	PreviewCommand string
}

func RenderArgs(parts []string, data any) ([]string, error) {
	rendered := make([]string, 0, len(parts))
	for index, part := range parts {
		value, err := renderString(fmt.Sprintf("arg-%d", index), part, data)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		rendered = append(rendered, value)
	}

	return rendered, nil
}

func ValidateArgs(parts []string, data any) error {
	_, err := RenderArgs(parts, data)
	if err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func RenderString(name string, value string, data any) (string, error) {
	return renderString(name, value, data)
}

func renderString(name string, value string, data any) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(value)
	if err != nil {
		return "", errors.Wrapf(err, "parse template %q", value)
	}

	buffer := bytes.Buffer{}
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", errors.Wrapf(err, "execute template %q", value)
	}

	return buffer.String(), nil
}
