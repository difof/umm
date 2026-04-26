package search

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

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
	args = append(args, "-e", query, "--", cfg.Root)

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

func emitDirnames(ctx context.Context, cfg cli.RootConfig, query string, strict bool, warningOut io.Writer, emit resultEmitter) error {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return errors.Wrap(err)
		}
		return nil
	}

	useGitIgnoreFilter := !cfg.Hidden
	if useGitIgnoreFilter {
		matches := []dirCandidate{}
		report, err := walkDirs(ctx, cfg, func(candidate dirCandidate) error {
			if !matcher.MatchString(candidate.rel) {
				return nil
			}
			matches = append(matches, candidate)
			return nil
		})
		if err != nil {
			return errors.Wrap(err)
		}

		filtered, err := filterIgnoredDirCandidates(ctx, cfg.Root, matches)
		if err != nil {
			return errors.Wrap(err)
		}

		for _, candidate := range filtered {
			if err := emit(resultfmt.Result{
				Kind:        resultfmt.KindDir,
				PreviewMode: "dir",
				Display:     candidate.rel,
				Path:        candidate.abs,
			}); err != nil {
				return errors.Wrap(err)
			}
		}

		reportDirWalkWarnings(warningOut, report.Warnings)
		return nil
	}

	report, err := walkDirs(ctx, cfg, func(candidate dirCandidate) error {
		if !matcher.MatchString(candidate.rel) {
			return nil
		}
		return emit(resultfmt.Result{
			Kind:        resultfmt.KindDir,
			PreviewMode: "dir",
			Display:     candidate.rel,
			Path:        candidate.abs,
		})
	})
	if err != nil {
		return errors.Wrap(err)
	}

	reportDirWalkWarnings(warningOut, report.Warnings)
	return nil
}

func reportDirWalkWarnings(out io.Writer, warnings []dirWalkWarning) {
	if len(warnings) == 0 || out == nil {
		return
	}

	const maxDetails = 5
	lines := []string{fmt.Sprintf("warning: skipped %d unreadable directorie(s) during dirname search", len(warnings))}
	for _, warning := range warnings[:min(len(warnings), maxDetails)] {
		lines = append(lines, fmt.Sprintf("  %s: %v", warning.Path, warning.Err))
	}
	if len(warnings) > maxDetails {
		lines = append(lines, fmt.Sprintf("  ... and %d more", len(warnings)-maxDetails))
	}
	_, _ = io.WriteString(out, strings.Join(lines, "\n")+"\n")
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

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
