package preview

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/deps"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/resultfmt"
)

func Run(ctx context.Context, appConfig ummconfig.Config, mode string, meta string, out io.Writer) error {
	result, err := resultfmt.DecodeMeta(meta)
	if err != nil {
		return errors.Wrap(err)
	}

	switch mode {
	case "file":
		renderFilePreview(ctx, out, appConfig, result)
	case "dir":
		renderDirPreview(ctx, out, appConfig, result)
	case "diff":
		renderDiffPreview(ctx, out, appConfig, result)
	default:
		_, _ = io.WriteString(out, fmt.Sprintf("unknown preview mode: %s\n", mode))
	}

	return nil
}

func renderFilePreview(ctx context.Context, out io.Writer, appConfig ummconfig.Config, result resultfmt.Result) {
	if result.Path == "" {
		_, _ = io.WriteString(out, "Error: file preview needs a file path\n")
		return
	}

	if _, err := os.Stat(result.Path); err != nil {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not preview file %s\n", result.Path))
		return
	}

	if tryConfiguredPathPreview(ctx, out, appConfig.Preview.File, result, false) {
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

	renderInternalFile(out, result.Path, result.Line)
}

func renderDirPreview(ctx context.Context, out io.Writer, appConfig ummconfig.Config, result resultfmt.Result) {
	if result.Path == "" {
		_, _ = io.WriteString(out, "Error: directory preview needs a directory path\n")
		return
	}

	info, err := os.Stat(result.Path)
	if err != nil || !info.IsDir() {
		_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not preview directory %s\n", result.Path))
		return
	}

	if tryConfiguredPathPreview(ctx, out, appConfig.Preview.Tree, result, true) {
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

func renderDiffPreview(ctx context.Context, out io.Writer, appConfig ummconfig.Config, result resultfmt.Result) {
	if result.Repo == "" || result.GitType == "" {
		_, _ = io.WriteString(out, "Error: git preview needs repository metadata\n")
		return
	}

	if result.GitType == "branch" {
		gitArgs := []string{"-C", result.Repo, "log", "--oneline", "--color=always"}
		if appConfig.Git.Limits.PreviewBranchCommits > 0 {
			gitArgs = append(gitArgs, fmt.Sprintf("-%d", appConfig.Git.Limits.PreviewBranchCommits))
		}
		gitArgs = append(gitArgs, result.GitRef)
		output, err := execx.CombinedOutput(ctx, result.Repo, nil, nil, "git", gitArgs...)
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

	if tryConfiguredDiffPreview(ctx, out, appConfig.Preview.Diff, result, raw) {
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

func tryConfiguredPathPreview(ctx context.Context, out io.Writer, command ummconfig.Command, result resultfmt.Result, tree bool) bool {
	if strings.TrimSpace(command.Cmd) == "" {
		return false
	}
	if !deps.Has(command.Cmd) {
		return false
	}

	args, err := ummconfig.RenderArgs(command.Args, pathTemplateData(result.Path, result.Line))
	if err != nil {
		warnPreviewFallback(out, err.Error())
		return false
	}
	if err := execx.Run(ctx, "", nil, nil, out, io.Discard, command.Cmd, args...); err != nil {
		kind := "file"
		if tree {
			kind = "tree"
		}
		warnPreviewFallback(out, fmt.Sprintf("configured %s preview command %q failed", kind, command.Cmd))
		return false
	}
	return true
}

func tryConfiguredDiffPreview(ctx context.Context, out io.Writer, command ummconfig.Command, result resultfmt.Result, raw []byte) bool {
	if strings.TrimSpace(command.Cmd) == "" {
		return false
	}
	if !deps.Has(command.Cmd) {
		return false
	}

	args, err := ummconfig.RenderArgs(command.Args, ummconfig.DiffTemplateData{
		Repo:    result.Repo,
		GitType: result.GitType,
		GitRef:  result.GitRef,
		Path:    result.Path,
		Display: result.Display,
		Summary: result.Summary,
	})
	if err != nil {
		warnPreviewFallback(out, err.Error())
		return false
	}
	if err := execx.Run(ctx, "", nil, bytes.NewReader(raw), out, io.Discard, command.Cmd, args...); err != nil {
		warnPreviewFallback(out, fmt.Sprintf("configured diff preview command %q failed", command.Cmd))
		return false
	}
	return true
}

func pathTemplateData(path string, line int) ummconfig.PathTemplateData {
	hasLine := line > 0
	start := 1
	end := 200
	lineRange := ":200"
	if hasLine {
		start = line - 10
		if start < 1 {
			start = 1
		}
		end = line + 20
		lineRange = fmt.Sprintf("%d:%d", start, end)
	}
	return ummconfig.PathTemplateData{
		Path:      path,
		Line:      line,
		HasLine:   hasLine,
		StartLine: start,
		EndLine:   end,
		LineRange: lineRange,
	}
}

func warnPreviewFallback(out io.Writer, message string) {
	_, _ = io.WriteString(out, "Warning: "+message+"; falling back to built-in preview\n\n")
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
		if err != nil && !errors.Is(err, io.EOF) {
			_, _ = io.WriteString(out, fmt.Sprintf("Error: Could not read preview file %s\n", path))
			return
		}
		if text == "" && errors.Is(err, io.EOF) {
			break
		}

		lineNumber++
		if lineNumber < start {
			if errors.Is(err, io.EOF) {
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

		if errors.Is(err, io.EOF) {
			break
		}
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}
