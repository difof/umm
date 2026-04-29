package actions

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/resultfmt"
	ummruntime "github.com/difof/umm/internal/runtime"
)

func RenderPathStats(out io.Writer, mode ummruntime.StatMode, results []resultfmt.Result) error {
	results = uniqueByPath(results)
	for index, result := range results {
		if result.Path == "" {
			continue
		}

		if mode == ummruntime.StatModeList {
			if _, err := io.WriteString(out, result.Path+"\n"); err != nil {
				return errors.Wrap(err)
			}
			continue
		}

		info, err := os.Stat(result.Path)
		if err != nil {
			return errors.Wrap(err)
		}

		if mode == ummruntime.StatModeLite {
			line := fmt.Sprintf("%s %s %s %s\n", fileTypeLabel(info), humanSize(info), info.ModTime().Format(time.RFC3339), result.Path)
			if _, err := io.WriteString(out, line); err != nil {
				return errors.Wrap(err)
			}
			continue
		}

		block := strings.Join([]string{
			"Path: " + result.Path,
			"Type: " + fileTypeLabel(info),
			"Size: " + humanSize(info),
			"Mode: " + info.Mode().String(),
			"Modified: " + info.ModTime().Format(time.RFC3339),
		}, "\n")
		if _, err := io.WriteString(out, block+"\n"); err != nil {
			return errors.Wrap(err)
		}
		if index < len(results)-1 {
			if _, err := io.WriteString(out, "\n"); err != nil {
				return errors.Wrap(err)
			}
		}
	}

	return nil
}

func RenderGitStats(out io.Writer, results []resultfmt.Result) error {
	for _, result := range results {
		if result.GitType == "tracked" {
			if err := RenderPathStats(out, ummruntime.StatModeLite, []resultfmt.Result{result}); err != nil {
				return errors.Wrap(err)
			}
			continue
		}

		line := strings.TrimSpace(result.Summary)
		if line == "" {
			line = strings.TrimSpace(result.Display)
		}
		if line == "" {
			continue
		}
		if _, err := io.WriteString(out, line+"\n"); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func uniquePaths(results []resultfmt.Result) []string {
	seen := map[string]struct{}{}
	paths := []string{}
	for _, result := range results {
		if result.Path == "" {
			continue
		}
		if _, ok := seen[result.Path]; ok {
			continue
		}
		seen[result.Path] = struct{}{}
		paths = append(paths, result.Path)
	}
	return paths
}

func uniqueByPath(results []resultfmt.Result) []resultfmt.Result {
	seen := map[string]struct{}{}
	filtered := make([]resultfmt.Result, 0, len(results))
	for _, result := range results {
		if result.Path == "" {
			continue
		}
		if _, ok := seen[result.Path]; ok {
			continue
		}
		seen[result.Path] = struct{}{}
		filtered = append(filtered, result)
	}

	return filtered
}

func fileTypeLabel(info os.FileInfo) string {
	if info.IsDir() {
		return "dir"
	}
	return "file"
}

func humanSize(info os.FileInfo) string {
	if info.IsDir() {
		return "-"
	}

	const unit = 1024
	size := float64(info.Size())
	if size < unit {
		return fmt.Sprintf("%d B", info.Size())
	}

	div, exp := float64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", size/div, "KMGTPE"[exp])
}
