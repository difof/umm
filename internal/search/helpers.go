package search

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

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

func itoa(v int) string {
	return strconv.Itoa(v)
}
