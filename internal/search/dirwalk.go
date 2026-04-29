package search

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/execx"
	ummruntime "github.com/difof/umm/internal/runtime"
)

type dirCandidate struct {
	abs string
	rel string
}

type dirJob struct {
	abs   string
	rel   string
	depth int
}

type dirWalkWarning struct {
	Path string
	Err  error
}

type dirWalkReport struct {
	Warnings []dirWalkWarning
}

func walkDirs(ctx context.Context, cfg ummruntime.RootConfig, visit func(dirCandidate) error) (dirWalkReport, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	report := dirWalkReport{}
	useGitIgnorePrune := !cfg.Hidden && isGitWorktree(ctx, cfg.Root)

	jobs := make(chan dirJob, dirWalkWorkerCount()*2)
	results := make(chan dirCandidate, dirWalkWorkerCount()*8)
	errs := make(chan error, 1)

	var pending sync.WaitGroup
	var workers sync.WaitGroup
	var warningsMu sync.Mutex
	var failOnce sync.Once
	recordWarning := func(path string, err error) {
		warningsMu.Lock()
		report.Warnings = append(report.Warnings, dirWalkWarning{Path: path, Err: err})
		warningsMu.Unlock()
	}
	recordFailure := func(err error) {
		failOnce.Do(func() {
			select {
			case errs <- err:
			default:
			}
			cancel()
		})
	}

	processDir := func(job dirJob) {
		entries, err := os.ReadDir(job.abs)
		if err != nil {
			recordWarning(job.abs, err)
			return
		}

		children := make([]dirCandidate, 0, len(entries))
		childJobs := make([]dirJob, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !cfg.Hidden && isHiddenName(name) {
				continue
			}

			childRel := name
			if job.rel != "" {
				childRel = job.rel + "/" + name
			}
			childDepth := job.depth + 1
			if cfg.MaxDepth > 0 && childDepth > cfg.MaxDepth {
				continue
			}
			if excluded(cfg.Excludes, childRel, true) {
				continue
			}

			childAbs := filepath.Join(job.abs, name)
			children = append(children, dirCandidate{abs: childAbs, rel: childRel})
			childJobs = append(childJobs, dirJob{abs: childAbs, rel: childRel, depth: childDepth})
		}

		ignored := map[string]struct{}{}
		if useGitIgnorePrune {
			ignored, err = ignoredDirKeys(ctx, cfg.Root, children)
			if err != nil {
				recordFailure(err)
				return
			}
		}

		for i, candidate := range children {
			if _, ok := ignored[candidate.rel]; ok {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case results <- candidate:
			}

			pending.Add(1)
			select {
			case <-ctx.Done():
				pending.Done()
				return
			case jobs <- childJobs[i]:
			}
		}
	}

	worker := func() {
		defer workers.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-jobs:
				if !ok {
					return
				}
				processDir(job)
				pending.Done()
			}
		}
	}

	workerCount := dirWalkWorkerCount()
	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	pending.Add(1)
	jobs <- dirJob{abs: cfg.Root, rel: "", depth: 0}

	go func() {
		pending.Wait()
		close(jobs)
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	var visitErr error
	for candidate := range results {
		if visitErr != nil {
			continue
		}
		if err := visit(candidate); err != nil {
			visitErr = err
			recordFailure(err)
		}
	}

	select {
	case err := <-errs:
		return dirWalkReport{}, errors.Wrap(err)
	default:
	}
	if err := ctx.Err(); err != nil && err != context.Canceled {
		return dirWalkReport{}, errors.Wrap(err)
	}
	sort.Slice(report.Warnings, func(i int, j int) bool {
		return report.Warnings[i].Path < report.Warnings[j].Path
	})

	return report, nil
}

func filterIgnoredDirCandidates(ctx context.Context, root string, candidates []dirCandidate) ([]dirCandidate, error) {
	if !isGitWorktree(ctx, root) {
		return candidates, nil
	}

	ignored, err := ignoredDirKeys(ctx, root, candidates)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	filtered := make([]dirCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := strings.TrimSuffix(candidate.rel, "/")
		if _, ok := ignored[key]; ok {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return filtered, nil
}

func isGitWorktree(ctx context.Context, root string) bool {
	_, err := execx.Output(ctx, root, nil, "git", "-C", root, "rev-parse", "--show-toplevel")
	return err == nil
}

func ignoredDirKeys(ctx context.Context, root string, candidates []dirCandidate) (map[string]struct{}, error) {
	if len(candidates) == 0 {
		return map[string]struct{}{}, nil
	}

	var input strings.Builder
	for _, candidate := range candidates {
		key := strings.TrimSuffix(candidate.rel, "/")
		if key == "" {
			continue
		}
		input.WriteString(key)
		input.WriteByte('/')
		input.WriteByte('\n')
	}
	if input.Len() == 0 {
		return map[string]struct{}{}, nil
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

	return ignored, nil
}

func dirWalkWorkerCount() int {
	workers := runtime.GOMAXPROCS(0) * 4
	if workers < 4 {
		workers = 4
	}
	if workers > 32 {
		workers = 32
	}
	return workers
}

func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}
