package editor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/execx"
)

type Target struct {
	Path string
	Line int
}

func Resolve() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return "nvim"
	}

	return editor
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

func Open(ctx context.Context, editor string, targets []Target) error {
	if len(targets) == 0 {
		return errors.New("no editor targets to open")
	}

	args := BuildArgs(editor, targets[0].Path, targets[0].Line)
	for _, target := range targets[1:] {
		args = append(args, target.Path)
	}

	if err := execx.Run(ctx, "", nil, os.Stdin, os.Stdout, os.Stderr, editor, args...); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
