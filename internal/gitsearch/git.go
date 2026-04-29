package gitsearch

import (
	"context"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/difof/errors"
	ummconfig "github.com/difof/umm/internal/config"
	"github.com/difof/umm/internal/execx"
	"github.com/difof/umm/internal/resultfmt"
	ummruntime "github.com/difof/umm/internal/runtime"
)

var errCollectLimitReached = errors.New("git result limit reached")

func Query(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config, query string, strict bool) ([]resultfmt.Result, error) {
	results, err := Aggregate(ctx, cfg, appConfig)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return filterResults(results, query, strict)
}

func Aggregate(ctx context.Context, cfg ummruntime.RootConfig, appConfig ummconfig.Config) ([]resultfmt.Result, error) {
	results := []resultfmt.Result{}
	limits := appConfig.Git.Limits

	modeSet := map[string]struct{}{}
	for _, mode := range cfg.GitModes {
		modeSet[mode] = struct{}{}
	}

	if _, ok := modeSet["commit"]; ok {
		items, err := collectCommits(ctx, cfg.Root, limits.Commits)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["branch"]; ok {
		items, err := collectBranches(ctx, cfg.Root, limits.Branches)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["tags"]; ok {
		items, err := collectTags(ctx, cfg.Root, limits.Tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["reflog"]; ok {
		items, err := collectReflog(ctx, cfg.Root, limits.Reflog)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["stash"]; ok {
		items, err := collectStashes(ctx, cfg.Root, limits.Stashes)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		results = append(results, items...)
	}
	if _, ok := modeSet["tracked"]; ok {
		items, err := collectTrackedFiles(ctx, cfg.Root, limits.Tracked)
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

func collectCommits(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	args := []string{"-C", root, "log", "--format=%H\t%h\t%cs\t%s"}
	if limit > 0 {
		args = append(args, "-"+strconv.Itoa(limit))
	}
	args = append(args, "--all")

	output, err := execx.Output(ctx, root, nil, "git", args...)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
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

func collectBranches(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	args := []string{"-C", root, "for-each-ref"}
	if limit > 0 {
		args = append(args, "--count="+strconv.Itoa(limit))
	}
	args = append(args, "--format=%(refname:short)\t%(objectname:short)\t%(subject)\t%(HEAD)", "refs/heads", "refs/remotes")

	output, err := execx.Output(ctx, root, nil, "git", args...)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	results := []resultfmt.Result{}
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
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

func collectTags(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	args := []string{"-C", root, "for-each-ref"}
	if limit > 0 {
		args = append(args, "--count="+strconv.Itoa(limit))
	}
	args = append(args, "--format=%(refname:short)\t%(subject)", "refs/tags")

	output, err := execx.Output(ctx, root, nil, "git", args...)
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

func collectReflog(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	args := []string{"-C", root, "reflog", "--format=%gd\t%h\t%gs\t%cr"}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	output, err := execx.Output(ctx, root, nil, "git", args...)
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

func collectStashes(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	args := []string{"-C", root, "stash", "list", "--format=%gd\t%gs"}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	output, err := execx.Output(ctx, root, nil, "git", args...)
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

func collectTrackedFiles(ctx context.Context, root string, limit int) ([]resultfmt.Result, error) {
	results := []resultfmt.Result{}
	_, err := execx.StreamLines(ctx, root, nil, nil, "git", []string{"-C", root, "ls-files"}, func(line []byte) error {
		if limit > 0 && len(results) >= limit {
			return errCollectLimitReached
		}

		value := strings.TrimSpace(string(line))
		if value == "" {
			return nil
		}

		path := filepath.Join(root, filepath.FromSlash(value))
		display := "file:    " + filepath.ToSlash(value)
		results = append(results, resultfmt.Result{
			Kind:        resultfmt.KindGit,
			PreviewMode: "file",
			Display:     display,
			Path:        path,
			Repo:        root,
			GitType:     "tracked",
			GitRef:      filepath.ToSlash(value),
			Summary:     display,
		})
		return nil
	})
	if err != nil && !errors.Is(err, errCollectLimitReached) {
		return nil, errors.Wrap(err)
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
