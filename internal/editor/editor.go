package editor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/execx"
)

type Target struct {
	Path string
	Line int
}

type Command struct {
	Name    string
	Args    []string
	Profile *ummconfig.Editor
}

func Resolve(appConfig ummconfig.Config) (Command, error) {
	raw := os.Getenv("EDITOR")
	if raw == "" {
		raw = "nvim"
	}

	parsed, err := Parse(raw)
	if err != nil {
		return Command{}, errors.Wrap(err)
	}

	profile, ok := lookupEditor(appConfig.Editors, parsed.Name)
	if !ok {
		return parsed, nil
	}

	resolved := profile
	return Command{
		Name:    resolved.Cmd,
		Args:    append(append([]string(nil), parsed.Args...), resolved.Args...),
		Profile: &resolved,
	}, nil
}

func Parse(raw string) (Command, error) {
	fields, err := splitShellWords(raw)
	if err != nil {
		return Command{}, errors.Wrapf(err, "parse editor command")
	}
	if len(fields) == 0 {
		return Command{}, errors.New("editor command is empty")
	}

	return Command{
		Name: fields[0],
		Args: append([]string(nil), fields[1:]...),
	}, nil
}

func BuildArgs(editor string, file string, line int) []string {
	editorName := filepath.Base(editor)
	hasLine := line > 0

	switch editorName {
	case "vim", "vi", "nvim", "nano", "micro", "emacs", "emacsclient":
		if hasLine {
			return []string{"+" + strconv.Itoa(line), file}
		}
		return []string{file}
	case "code", "code-insiders", "cursor", "agy":
		if hasLine {
			return []string{"--goto", file + ":" + strconv.Itoa(line)}
		}
		return []string{file}
	case "subl", "sublime_text":
		if hasLine {
			return []string{file + ":" + strconv.Itoa(line)}
		}
		return []string{file}
	default:
		if hasLine {
			return []string{"+" + strconv.Itoa(line), file}
		}
		return []string{file}
	}
}

func Open(ctx context.Context, command Command, targets []Target) error {
	if len(targets) == 0 {
		return errors.New("no editor targets to open")
	}

	args := append([]string(nil), command.Args...)
	if command.Profile != nil {
		firstArgs, err := renderTargetArgs(command.Profile.FirstTarget, targets[0])
		if err != nil {
			return errors.Wrap(err)
		}
		args = append(args, firstArgs...)
		for _, target := range targets[1:] {
			targetArgs, err := renderTargetArgs(command.Profile.RestTarget, target)
			if err != nil {
				return errors.Wrap(err)
			}
			args = append(args, targetArgs...)
		}
	} else {
		args = append(args, BuildArgs(command.Name, targets[0].Path, targets[0].Line)...)
		for _, target := range targets[1:] {
			args = append(args, target.Path)
		}
	}

	if err := execx.Run(ctx, "", nil, os.Stdin, os.Stdout, os.Stderr, command.Name, args...); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func lookupEditor(editors map[string]ummconfig.Editor, name string) (ummconfig.Editor, bool) {
	if editor, ok := editors[name]; ok {
		return editor, true
	}
	base := filepath.Base(name)
	editor, ok := editors[base]
	return editor, ok
}

func renderTargetArgs(parts []string, target Target) ([]string, error) {
	hasLine := target.Line > 0
	start := 1
	end := 200
	lineRange := ":200"
	if hasLine {
		start = target.Line - 10
		if start < 1 {
			start = 1
		}
		end = target.Line + 20
		lineRange = strconv.Itoa(start) + ":" + strconv.Itoa(end)
	}

	args, err := ummconfig.RenderArgs(parts, ummconfig.PathTemplateData{
		Path:      target.Path,
		Line:      target.Line,
		HasLine:   hasLine,
		StartLine: start,
		EndLine:   end,
		LineRange: lineRange,
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return args, nil
}

func splitShellWords(input string) ([]string, error) {
	fields := []string{}
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		fields = append(fields, current.String())
		current.Reset()
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				current.WriteRune(r)
			}
		case inDouble:
			switch r {
			case '"':
				inDouble = false
			case '\\':
				escaped = true
			default:
				current.WriteRune(r)
			}
		default:
			switch {
			case unicode.IsSpace(r):
				flush()
			case r == '\\':
				escaped = true
			case r == '\'':
				inSingle = true
			case r == '"':
				inDouble = true
			default:
				current.WriteRune(r)
			}
		}
	}

	if escaped {
		return nil, errors.New("unterminated escape in editor command")
	}
	if inSingle || inDouble {
		return nil, errors.New("unterminated quote in editor command")
	}

	flush()
	return fields, nil
}
