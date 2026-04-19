package gitsearch

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/resultfmt"
)

func Query(ctx context.Context, cfg cli.RootConfig, query string, strict bool) ([]resultfmt.Result, error) {
	results, err := Aggregate(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return filterResults(results, query, strict)
}

func Aggregate(ctx context.Context, cfg cli.RootConfig) ([]resultfmt.Result, error) {
	results := []resultfmt.Result{}

	modeSet := map[string]struct{}{}
	for _, mode := range cfg.GitModes {
		modeSet[mode] = struct{}{}
	}

	if _, ok := modeSet["commit"]; ok {
		items, err := collectCommits(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["branch"]; ok {
		items, err := collectBranches(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["tags"]; ok {
		items, err := collectTags(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["reflog"]; ok {
		items, err := collectReflog(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["stash"]; ok {
		items, err := collectStashes(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["tracked"]; ok {
		items, err := collectTrackedFiles(ctx, cfg.Root)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}

	return results, nil
}

func ValidateRepo(ctx context.Context, root string) error {
	if _, err := execx.Output(ctx, root, nil, "git", "-C", root, "rev-parse", "--git-dir"); err != nil {
		return errors.Newf("not a git repository: %s", root)
	}

	return nil
}

func TrackedFileSubset(results []resultfmt.Result) []resultfmt.Result {
	subset := []resultfmt.Result{}
	for _, result := range results {
		if result.GitType != "tracked" || result.Path == "" {
			continue
		}
		subset = append(subset, result)
	}

	return subset
}

func filterResults(results []resultfmt.Result, query string, strict bool) ([]resultfmt.Result, error) {
	matcher, err := compileSmartRegex(query)
	if err != nil {
		if strict {
			return nil, errors.Wrap(err)
		}
		return nil, nil
	}

	filtered := []resultfmt.Result{}
	for _, result := range results {
		if matcher.MatchString(result.Display) {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

func collectCommits(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "log", "--format=%H\t%h\t%cs\t%s", "--all")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		display := "commit:  " + parts[1] + " " + parts[2] + " " + parts[3]
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "diff",
			Display:     display,
			Repo:        root,
			GitType:     "commit",
			GitRef:      parts[0],
			Summary:     display,
		})
	}

	return results, nil
}

func collectBranches(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "for-each-ref", "--format=%(refname:short)\t%(objectname:short)\t%(subject)\t%(HEAD)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		current := strings.TrimSpace(parts[3]) == "*"
		prefix := "branch:  "
		if current {
			prefix += "* "
		}
		display := prefix + name
		if subject := strings.TrimSpace(parts[2]); subject != "" {
			display += " " + subject
		}
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "diff",
			Display:     display,
			Repo:        root,
			GitType:     "branch",
			GitRef:      name,
			Summary:     display,
			Current:     current,
		})
	}

	return results, nil
}

func collectTags(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "tag", "-l", "--format=%(refname:short)\t%(subject)")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 0 {
			continue
		}
		display := "tag:     " + parts[0]
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			display += " " + parts[1]
		}
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "diff",
			Display:     display,
			Repo:        root,
			GitType:     "tag",
			GitRef:      parts[0],
			Summary:     display,
		})
	}

	return results, nil
}

func collectReflog(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "reflog", "--format=%gd\t%h\t%gs\t%cr")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		display := "reflog:  " + parts[0] + " " + parts[1] + " " + parts[2]
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "diff",
			Display:     display,
			Repo:        root,
			GitType:     "reflog",
			GitRef:      parts[0],
			Summary:     display,
			SubValue:    parts[1],
		})
	}

	return results, nil
}

func collectStashes(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "stash", "list", "--format=%gd\t%gs")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		display := "stash:   " + parts[0] + " " + parts[1]
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "diff",
			Display:     display,
			Repo:        root,
			GitType:     "stash",
			GitRef:      parts[0],
			Summary:     display,
		})
	}

	return results, nil
}

func collectTrackedFiles(ctx context.Context, root string) ([]resultfmt.Result, error) {
	output, err := execx.Output(ctx, root, nil, "git", "-C", root, "ls-files")
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(line))
		display := "file:    " + filepath.ToSlash(line)
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "file",
			Display:     display,
			Path:        path,
			Repo:        root,
			GitType:     "tracked",
			GitRef:      filepath.ToSlash(line),
			Summary:     display,
		})
	}

	return results, nil
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
