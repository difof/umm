package editor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/execx"
)

type Target struct {
	Path string
	Line int
}

type Command struct {
	Name string
	Args []string
}

func Resolve() (Command, error) {
	raw := os.Getenv("EDITOR")
	if raw == "" {
		raw = "nvim"
	}

	return Parse(raw)
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
	args = append(args, BuildArgs(command.Name, targets[0].Path, targets[0].Line)...)
	for _, target := range targets[1:] {
		args = append(args, target.Path)
	}

	if err := execx.Run(ctx, "", nil, os.Stdin, os.Stdout, os.Stderr, command.Name, args...); err != nil {
		return errors.Wrap(err)
	}

	return nil
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
