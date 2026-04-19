package search

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/difof/errors"
	"github.com/difof/umm/internal/cli"
	"github.com/difof/umm/internal/execx"
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

func walkDirs(ctx context.Context, cfg cli.RootConfig, visit func(dirCandidate) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan dirJob, dirWalkWorkerCount()*2)
	results := make(chan dirCandidate, dirWalkWorkerCount()*8)
	errs := make(chan error, 1)

	var pending sync.WaitGroup
	var workers sync.WaitGroup

	processDir := func(job dirJob) {
		entries, err := os.ReadDir(job.abs)
		if err != nil {
			return
		}

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
			candidate := dirCandidate{abs: childAbs, rel: childRel}

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
			case jobs <- dirJob{abs: childAbs, rel: childRel, depth: childDepth}:
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
			cancel()
			select {
			case errs <- err:
			default:
			}
		}
	}

	if visitErr != nil {
		return errors.Wrap(visitErr)
	}
	if err := ctx.Err(); err != nil && err != context.Canceled {
		return errors.Wrap(err)
	}

	return nil
}

func filterIgnoredDirCandidates(ctx context.Context, root string, candidates []dirCandidate) ([]dirCandidate, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	if _, err := execx.Output(ctx, root, nil, "git", "-C", root, "rev-parse", "--show-toplevel"); err != nil {
		return candidates, nil
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
