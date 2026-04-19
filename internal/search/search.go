package search

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/jsonx"
	"github.com/difof/umm/internal/resultfmt"
)

type rgJSONLine struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
	} `json:"data"`
}

type resultEmitter func(resultfmt.Result) error

const (
	matchStartANSI = "\x1b[1;33m"
	matchResetANSI = "\x1b[0m"
)

func Query(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	results := []resultfmt.Result{}
	err := emitQuery(ctx, cfg, query, strict, func(result resultfmt.Result) error {
		results = append(results, result)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return results, nil
}

func EmitLines(ctx context.Context, cfg cli.RootConfig, query string, out io.Writer) error {
	return emitQuery(ctx, cfg, query, false, func(result resultfmt.Result) error {
		result.Display = highlightDisplay(result.Display, query)

		line, err := resultfmt.EncodeLine(result)
		if err != nil {
			return errors.Wrap(err)
		}

		if _, err := io.WriteString(out, line+"\n"); err != nil {
			return errors.Wrap(err)
		}

		return nil
	})
}

func emitQuery(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	switch cfg.SearchMode {
	case cli.SearchModeOnlyDirname:
		return emitDirnames(ctx, cfg, query, strict, emit)
	case cli.SearchModeOnlyFilename:
		return emitFilenames(ctx, cfg, query, strict, emit)
	case cli.SearchModeDefault:
		return emitDefault(ctx, cfg, query, strict, emit)
	default:
		return errors.Newf("unsupported search mode %q", cfg.SearchMode)
	}
}

func emitDefault(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	if query == "" {
		if cfg.NoFilename {
			return nil
		}

		return emitFilenames(ctx, cfg, query, strict, emit)
	}

	if err := emitContent(ctx, cfg, query, strict, emit); err != nil {
		return errors.Wrap(err)
	}
	if cfg.NoFilename {
		return nil
	}

	return emitFilenames(ctx, cfg, query, strict, emit)
}

func emitContent(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	args := []string{"--json", "--line-number", "--no-heading", "--smart-case"}
	if cfg.MaxDepth > 0 {
		args = append(args, "--max-depth", itoa(cfg.MaxDepth))
	}
	for _, pattern := range cfg.Excludes {
		args = append(args, "--glob", "!"+pattern)
	}
	if cfg.Hidden {
		args = append(args, "--hidden", "--no-ignore")
	}
	args = append(args, query, cfg.Root)

	stderr, err := execx.StreamLines(ctx, cfg.Root, nil, nil, "rg", args, func(line []byte) error {
		trimmedLine := strings.TrimSpace(string(line))
		if trimmedLine == "" {
			return nil
		}

		var event rgJSONLine
		if err := jsonx.Fast.Unmarshal([]byte(trimmedLine), &event); err != nil {
			if strict {
				return errors.Wrapf(err, "parse ripgrep json")
			}
			return nil
		}
		if event.Type != "match" {
			return nil
		}

		path := event.Data.Path.Text
		lineNumber := event.Data.LineNumber
		text := strings.TrimRight(event.Data.Lines.Text, "\r\n")
		return emit(resultfmt.Result{
			Kind:        resultfmt.KindFile,
			PreviewMode: "file",
			Display:     relDisplay(cfg.Root, path) + ":" + itoa(lineNumber) + ":" + text,
			Path:        path,
			Line:        lineNumber,
		})
	})
	if err != nil {
		if code, ok := execx.ExitCode(err); ok {
			switch code {
			case 1:
				return nil
			case 2:
				if !strict {
					return nil
				}
			}
		}

		if !strict {
			return nil
		}

		trimmed := strings.TrimSpace(stderr)
		if trimmed != "" {
			return errors.New(trimmed)
		}
		return errors.Wrap(err)
	}

	return nil
}

func emitFilenames(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return errors.Wrap(err)
		}
		return nil
	}

	_, err = execx.StreamLines(ctx, cfg.Root, nil, nil, "rg", buildFilesArgs(cfg), func(line []byte) error {
		path := strings.TrimSpace(string(line))
		if path == "" {
			return nil
		}

		rel := relDisplay(cfg.Root, path)
		if !matcher.MatchString(rel) {
			return nil
		}

		return emit(resultfmt.Result{
			Kind:        resultfmt.KindFile,
			PreviewMode: "file",
			Display:     rel,
			Path:        path,
		})
	})
	if err != nil {
		if code, ok := execx.ExitCode(err); ok {
			if code == 1 || code == 2 {
				return nil
			}
		}
		return errors.Wrap(err)
	}

	return nil
}

func emitDirnames(ctx context.Context, cfg cli.RootConfig, query string, strict bool, emit resultEmitter) error {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return errors.Wrap(err)
		}
		return nil
	}

	dirs, err := listDirs(ctx, cfg)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, dir := range dirs {
		rel := relDisplay(cfg.Root, dir)
		if !matcher.MatchString(filepath.ToSlash(rel)) {
			continue
		}
		if err := emit(resultfmt.Result{
			Kind:        resultfmt.KindDir,
			PreviewMode: "dir",
			Display:     rel,
			Path:        dir,
		}); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func listDirs(ctx context.Context, cfg cli.RootConfig) ([]string, error) {
	seen := map[string]struct{}{}
	dirs := []string{}

	err := filepath.WalkDir(cfg.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == cfg.Root {
			return nil
		}

		rel := relDisplay(cfg.Root, path)
		if cfg.MaxDepth > 0 && depth(rel) > cfg.MaxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if hiddenPath(rel) && !cfg.Hidden {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if excluded(cfg.Excludes, rel, entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}

		if _, ok := seen[path]; ok {
			return nil
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if cfg.Hidden {
		return dirs, nil
	}

	filtered, err := filterIgnoredDirs(ctx, cfg.Root, dirs)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return filtered, nil
}

func listFiles(ctx context.Context, cfg cli.RootConfig) ([]string, error) {
	files := []string{}
	_, err := execx.StreamLines(ctx, cfg.Root, nil, nil, "rg", buildFilesArgs(cfg), func(line []byte) error {
		path := strings.TrimSpace(string(line))
		if path == "" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		if code, ok := execx.ExitCode(err); ok {
			if code == 1 || code == 2 {
				return files, nil
			}
		}
		return nil, errors.Wrap(err)
	}

	return files, nil
}

func buildFilesArgs(cfg cli.RootConfig) []string {
	args := []string{"--files"}
	if cfg.MaxDepth > 0 {
		args = append(args, "--max-depth", itoa(cfg.MaxDepth))
	}
	for _, pattern := range cfg.Excludes {
		args = append(args, "--glob", "!"+pattern)
	}
	if cfg.Hidden {
		args = append(args, "--hidden", "--no-ignore")
	}
	args = append(args, cfg.Root)
	return args
}

func compileSmartRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return regexp.Compile(".*")
	}

	if hasUpper(pattern) {
		return regexp.Compile(pattern)
	}

	return regexp.Compile("(?i)" + pattern)
}

func highlightDisplay(display string, query string) string {
	if query == "" || display == "" {
		return display
	}

	matcher, err := compileSmartRegex(query)
	if err != nil {
		return display
	}

	ranges := matcher.FindAllStringIndex(display, -1)
	if len(ranges) == 0 {
		return display
	}

	merged := mergeRanges(ranges)
	if len(merged) == 0 {
		return display
	}

	var builder strings.Builder
	last := 0
	for _, match := range merged {
		if match[0] > last {
			builder.WriteString(display[last:match[0]])
		}
		builder.WriteString(matchStartANSI)
		builder.WriteString(display[match[0]:match[1]])
		builder.WriteString(matchResetANSI)
		last = match[1]
	}
	if last < len(display) {
		builder.WriteString(display[last:])
	}

	return builder.String()
}

func mergeRanges(ranges [][]int) [][]int {
	filtered := make([][]int, 0, len(ranges))
	for _, match := range ranges {
		if len(match) != 2 || match[0] >= match[1] {
			continue
		}
		filtered = append(filtered, []int{match[0], match[1]})
	}
	if len(filtered) == 0 {
		return nil
	}

	sort.Slice(filtered, func(i int, j int) bool {
		if filtered[i][0] == filtered[j][0] {
			return filtered[i][1] < filtered[j][1]
		}
		return filtered[i][0] < filtered[j][0]
	})

	merged := [][]int{filtered[0]}
	for _, current := range filtered[1:] {
		last := merged[len(merged)-1]
		if current[0] <= last[1] {
			if current[1] > last[1] {
				last[1] = current[1]
			}
			continue
		}
		merged = append(merged, current)
	}

	return merged
}

func hiddenPath(rel string) bool {
	if rel == "." || rel == "" {
		return false
	}

	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}

	return false
}

func hasUpper(value string) bool {
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}

	return false
}

func excluded(patterns []string, rel string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	candidates := []string{rel}
	if isDir && !strings.HasSuffix(rel, "/") {
		candidates = append(candidates, rel+"/")
	}

	for _, pattern := range patterns {
		for _, candidate := range candidates {
			matched, err := doublestar.Match(pattern, candidate)
			if err == nil && matched {
				return true
			}
		}

		trimmed := strings.TrimSuffix(pattern, "/")
		if trimmed != pattern {
			for _, candidate := range candidates {
				if candidate == trimmed || strings.HasPrefix(candidate, trimmed+"/") {
					return true
				}
			}
		}
	}

	return false
}

func relDisplay(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}

func depth(rel string) int {
	rel = strings.Trim(filepath.ToSlash(rel), "/")
	if rel == "" || rel == "." {
		return 0
	}

	return strings.Count(rel, "/") + 1
}

func filterIgnoredDirs(ctx context.Context, root string, dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return dirs, nil
	}

	if _, err := execx.Output(ctx, root, nil, "git", "-C", root, "rev-parse", "--show-toplevel"); err != nil {
		return dirs, nil
	}

	var input strings.Builder
	for _, dir := range dirs {
		rel := relDisplay(root, dir)
		key := strings.TrimSuffix(filepath.ToSlash(rel), "/")
		if key == "." || key == "" {
			continue
		}
		input.WriteString(key)
		input.WriteByte('/')
		input.WriteByte('\n')
	}

	ignoredOutput, err := execx.CombinedOutput(ctx, root, nil, strings.NewReader(input.String()), "git", "-C", root, "check-ignore", "--stdin")
	ignored := map[string]struct{}{}
	if err != nil {
		if code, ok := execx.ExitCode(err); !ok || code != 1 {
			return nil, errors.Wrap(err)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(ignoredOutput), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "/"))
		if line == "" {
			continue
		}
		ignored[line] = struct{}{}
	}

	filtered := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		key := strings.TrimSuffix(filepath.ToSlash(relDisplay(root, dir)), "/")
		if _, ok := ignored[key]; ok {
			continue
		}
		filtered = append(filtered, dir)
	}

	return filtered, nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
