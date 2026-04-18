package search

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/execx"
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

func Query(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	switch cfg.SearchMode {
	case cli.SearchModeOnlyDirname:
		return searchDirnames(ctx, cfg, query, strict)
	case cli.SearchModeOnlyFilename:
		return searchFilenames(ctx, cfg, query, strict)
	case cli.SearchModeDefault:
		return searchDefault(ctx, cfg, query, strict)
	default:
		return nil, errors.Newf("unsupported search mode %q", cfg.SearchMode)
	}
}

func EmitLines(ctx context.Context, cfg cli.RootConfig, query string, out io.Writer) error {
	results, err := Query(ctx, cfg, query, false)
	if err != nil {
		return errors.Wrap(err)
	}

	for _, result := range results {
		line, err := resultfmt.EncodeLine(result)
		if err != nil {
			return errors.Wrap(err)
		}

		if _, err := io.WriteString(out, line+"\n"); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func searchDefault(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	contentResults, err := searchContent(ctx, cfg, query, strict)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	pathResults, err := searchFilenames(ctx, cfg, query, strict)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := make([]resultfmt.Result, 0, len(contentResults)+len(pathResults))
	results = append(results, contentResults...)
	results = append(results, pathResults...)
	return results, nil
}

func searchContent(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
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

	output, err := execx.CombinedOutput(ctx, cfg.Root, nil, nil, "rg", args...)
	if err != nil {
		if code, ok := execx.ExitCode(err); ok {
			switch code {
			case 1:
				return nil, nil
			case 2:
				if !strict {
					return nil, nil
				}
			}
		}

		if !strict {
			return nil, nil
		}

		trimmed := strings.TrimSpace(output)
		if trimmed != "" {
			return nil, errors.New(trimmed)
		}
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event rgJSONLine
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if strict {
				return nil, errors.Wrapf(err, "parse ripgrep json")
			}
			continue
		}
		if event.Type != "match" {
			continue
		}

		path := event.Data.Path.Text
		lineNumber := event.Data.LineNumber
		text := strings.TrimRight(event.Data.Lines.Text, "\r\n")
		display := relDisplay(cfg.Root, path) + ":" + itoa(lineNumber) + ":" + text
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindFile,
			PreviewMode: "file",
			Display:     display,
			Path:        path,
			Line:        lineNumber,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err)
	}

	return results, nil
}

func searchFilenames(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return nil, errors.Wrap(err)
		}
		return nil, nil
	}

	files, err := listFiles(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, path := range files {
		rel := relDisplay(cfg.Root, path)
		if !matcher.MatchString(rel) {
			continue
		}

		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindFile,
			PreviewMode: "file",
			Display:     rel,
			Path:        path,
		})
	}

	return results, nil
}

func searchDirnames(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return nil, errors.Wrap(err)
		}
		return nil, nil
	}

	files, err := listFiles(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	seen := map[string]struct{}{}
	results := []resultfmt.Result{}
	for _, filePath := range files {
		current := filepath.Dir(filePath)
		for current != cfg.Root && current != "." && current != string(filepath.Separator) {
			rel := relDisplay(cfg.Root, current)
			if rel == "." || rel == "" {
				break
			}
			if _, ok := seen[current]; ok {
				current = filepath.Dir(current)
				continue
			}
			seen[current] = struct{}{}

			if cfg.MaxDepth > 0 && depth(rel) > cfg.MaxDepth {
				current = filepath.Dir(current)
				continue
			}
			if !matcher.MatchString(filepath.ToSlash(rel)) {
				current = filepath.Dir(current)
				continue
			}

			results = append(results, resultfmt.Result{
				Kind:        resultfmt.KindDir,
				PreviewMode: "dir",
				Display:     rel,
				Path:        current,
			})

			current = filepath.Dir(current)
		}
	}

	return results, nil
}

func listFiles(ctx context.Context, cfg cli.RootConfig) ([]string, error) {
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

	output, err := execx.Output(ctx, cfg.Root, nil, "rg", args...)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	files := []string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}

	return files, nil
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

func itoa(v int) string {
	return strconv.Itoa(v)
}
