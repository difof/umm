package preview

import (
	"bufio"
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/resultfmt"
)

func Run(ctx context.Context, mode string, meta string, out io.Writer) error {
	result, err := resultfmt.DecodeMeta(meta)
	if err != nil {
		return errors.Wrap(err)
	}

	switch mode {
	case "file":
		renderFilePreview(ctx, out, result)
	case "dir":
		renderDirPreview(out, result)
	case "diff":
		renderDiffPreview(ctx, out, result)
	default:
		_, _ = io.WriteString(out, fmt.Sprintf("unknown preview mode: %s\n", mode))
	}

	return nil
}

func renderFilePreview(ctx context.Context, out io.Writer, result resultfmt.Result) {
	if result.Path == "" {
		_, _ = io.WriteString(out, "Error: file preview needs a file path\n")
		return
	}

	if _, err := os.Stat(result.Path); err != nil {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not preview file %s\n", result.Path))
		return
	}

	if deps.Has("bat") {
		args := []string{"--paging=never", "--color=always", "--style=numbers,header"}
		if result.Line > 0 {
			start := result.Line - 10
			if start < 1 {
				start = 1
			}
			end := result.Line + 20
			args = append(args, "--highlight-line", itoa(result.Line), "--line-range", fmt.Sprintf("%d:%d", start, end))
		} else {
			args = append(args, "--line-range", ":200")
		}
		args = append(args, result.Path)
		if err := execx.Run(ctx, "", nil, nil, out, io.Discard, "bat", args...); err == nil {
			return
		}
	}

	if result.Line <= 0 && deps.Has("cat") {
		if err := execx.Run(ctx, "", nil, nil, out, io.Discard, "cat", result.Path); err == nil {
			return
		}
	}

	renderInternalFile(out, result.Path, result.Line)
}

func renderDirPreview(out io.Writer, result resultfmt.Result) {
	if result.Path == "" {
		_, _ = io.WriteString(out, "Error: directory preview needs a directory path\n")
		return
	}

	info, err := os.Stat(result.Path)
	if err != nil || !info.IsDir() {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not preview directory %s\n", result.Path))
		return
	}

	maxDepth := 2
	maxLines := 200
	count := 0

	_, _ = io.WriteString(out, filepath.Base(result.Path)+"/\n")
	_ = filepath.WalkDir(result.Path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == result.Path {
			return nil
		}
		if count >= maxLines {
			return io.EOF
		}

		rel, err := filepath.Rel(result.Path, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if depth > maxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		indent := strings.Repeat("  ", depth)
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		_, _ = io.WriteString(out, indent+name+"\n")
		count++
		return nil
	})
}

func renderDiffPreview(ctx context.Context, out io.Writer, result resultfmt.Result) {
	if result.Repo == "" || result.GitType == "" {
		_, _ = io.WriteString(out, "Error: git preview needs repository metadata\n")
		return
	}

	if result.GitType == "branch" {
		output, err := execx.CombinedOutput(ctx, result.Repo, nil, nil, "git", "-C", result.Repo, "log", "--oneline", "--color=always", "-10", result.GitRef)
		if err != nil && strings.TrimSpace(output) == "" {
			_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not show branch %s\n", result.GitRef))
			return
		}
		_, _ = io.WriteString(out, output)
		return
	}

	gitColor := "always"
	pager := "internal"
	if deps.Has("delta") {
		gitColor = "never"
		pager = "delta"
	} else if deps.Has("bat") {
		pager = "bat"
	} else if deps.Has("cat") {
		pager = "cat"
	}

	var gitArgs []string
	switch result.GitType {
	case "commit", "tag", "reflog":
		gitArgs = []string{"-C", result.Repo, "show", "--color=" + gitColor, result.GitRef}
	case "stash":
		gitArgs = []string{"-C", result.Repo, "stash", "show", "-p", "--color=" + gitColor, result.GitRef}
	default:
		_, _ = io.WriteString(out, fmt.Sprintf("Unsupported git preview type: %s\n", result.GitType))
		return
	}

	raw, err := execx.OutputBytes(ctx, result.Repo, nil, nil, "git", gitArgs...)
	if err != nil {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not show %s %s\n", result.GitType, result.GitRef))
		return
	}

	switch pager {
	case "delta":
		if err := execx.Run(ctx, "", nil, bytes.NewReader(raw), out, io.Discard, "delta"); err == nil {
			return
		}
	case "bat":
		if err := execx.Run(ctx, "", nil, bytes.NewReader(raw), out, io.Discard, "bat", "--paging=never", "--style=numbers,changes", "--language=diff", "--color=always"); err == nil {
			return
		}
	case "cat":
		_, _ = out.Write(raw)
		return
	}

	_, _ = out.Write(raw)
}

func renderInternalFile(out io.Writer, path string, line int) {
	file, err := os.Open(path)
	if err != nil {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not preview file %s\n", path))
		return
	}
	defer file.Close()

	start := 1
	end := 200
	if line > 0 {
		start = line - 10
		if start < 1 {
			start = 1
		}
		end = line + 20
	}

	reader := bufio.NewReader(file)
	lineNumber := 0
	for {
		text, err := reader.ReadString('\n')
		if err != nil && !stderrors.Is(err, io.EOF) {
			_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not read preview file %s\n", path))
			return
		}
		if text == "" && stderrors.Is(err, io.EOF) {
			break
		}

		lineNumber++
		if lineNumber < start {
			if stderrors.Is(err, io.EOF) {
				break
			}
			continue
		}
		if lineNumber > end {
			break
		}
		text = strings.TrimRight(text, "\r\n")

		prefix := fmt.Sprintf("%4d ", lineNumber)
		if lineNumber == line && line > 0 {
			prefix = fmt.Sprintf(">%4d ", lineNumber)
		}
		_, _ = io.WriteString(out, prefix+text+"\n")

		if stderrors.Is(err, io.EOF) {
			break
		}
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
