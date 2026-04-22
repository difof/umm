package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/difof/errors"
)

type SearchMode string

const (
	SearchModeDefault      SearchMode = "default"
	SearchModeOnlyFilename SearchMode = "only-filename"
	SearchModeOnlyDirname  SearchMode = "only-dirname"
	SearchModeGit          SearchMode = "git"
)

type ActionKind string

const (
	ActionDefault ActionKind = "default"
	ActionAsk     ActionKind = "ask"
	ActionSystem  ActionKind = "system"
	ActionStat    ActionKind = "stat"
)

type StatMode string

const (
	StatModeFull StatMode = "full"
	StatModeLite StatMode = "lite"
	StatModeList StatMode = "list"
)

var AllGitModes = []string{"commit", "branch", "tags", "reflog", "stash", "tracked"}
var AllStatModes = []string{"full", "lite", "list"}

var validGitModes = map[string]struct{}{
	"commit":  {},
	"branch":  {},
	"tags":    {},
	"reflog":  {},
	"stash":   {},
	"tracked": {},
}

var validStatModes = map[StatMode]struct{}{
	StatModeFull: {},
	StatModeLite: {},
	StatModeList: {},
}

type RawRootOptions struct {
	Root             string
	Pattern          string
	Excludes         []string
	Hidden           bool
	NoFilename       bool
	OnlyFilename     bool
	OnlyDirname      bool
	Git              bool
	GitModes         []string
	MaxDepth         uint
	NoUI             bool
	NoMulti          bool
	OpenAsk          bool
	OpenSys          bool
	OnlyStat         string
	DefaultGitModes  []string
	GitModesExplicit bool
}

type RootConfig struct {
	Root         string
	Pattern      string
	Excludes     []string
	Hidden       bool
	MaxDepth     int
	NoUI         bool
	NoMulti      bool
	SearchMode   SearchMode
	GitModes     []string
	Action       ActionKind
	StatMode     StatMode
	OpenAsk      bool
	OpenSys      bool
	NoFilename   bool
	OnlyFilename bool
	OnlyDirname  bool
}

func NormalizeRootOptions(raw RawRootOptions) (cfg RootConfig, err error) {
	defer errors.Recover(&err)

	return normalize(raw)
}

func NormalizeEmitterOptions(raw RawRootOptions) (cfg RootConfig, err error) {
	defer errors.Recover(&err)

	return normalize(raw)
}

func normalize(raw RawRootOptions) (cfg RootConfig, err error) {
	defer errors.Recover(&err)

	if raw.NoFilename && raw.OnlyFilename {
		return cfg, errors.New("--no-filename cannot be used with --only-filename")
	}
	if raw.NoFilename && raw.OnlyDirname {
		return cfg, errors.New("--no-filename cannot be used with --only-dirname")
	}
	if raw.OnlyFilename && raw.OnlyDirname {
		return cfg, errors.New("--only-filename cannot be used with --only-dirname")
	}
	if raw.OpenAsk && raw.OpenSys {
		return cfg, errors.New("--open-ask cannot be used with --open-sys")
	}
	if raw.OpenAsk && raw.OnlyStat != "" {
		return cfg, errors.New("--open-ask cannot be used with --only-stat")
	}
	if raw.OpenSys && raw.OnlyStat != "" {
		return cfg, errors.New("--open-sys cannot be used with --only-stat")
	}
	if raw.NoUI && raw.Pattern == "" {
		return cfg, errors.New("--pattern is required when using --no-ui")
	}

	root := raw.Root
	if root == "" {
		root = "."
	}
	root = errors.MustResult(filepath.Abs(root))

	info := errors.MustResult(os.Stat(root))
	if !info.IsDir() {
		return cfg, errors.Newf("root path is not a directory: %s", root)
	}

	defaultGitModes := raw.DefaultGitModes
	if len(defaultGitModes) == 0 {
		defaultGitModes = AllGitModes
	}
	gitModes := errors.MustResult(parseGitModes(raw.GitModes, defaultGitModes, raw.GitModesExplicit))
	if raw.Git {
		if len(raw.Excludes) > 0 {
			return cfg, errors.New("--exclude cannot be used with --git")
		}
		if raw.Hidden {
			return cfg, errors.New("--hidden cannot be used with --git")
		}
		if raw.MaxDepth > 0 {
			return cfg, errors.New("--max-depth cannot be used with --git")
		}
		if raw.NoFilename {
			return cfg, errors.New("--no-filename cannot be used with --git")
		}
		if raw.OnlyFilename {
			return cfg, errors.New("--only-filename cannot be used with --git")
		}
		if raw.OnlyDirname {
			return cfg, errors.New("--only-dirname cannot be used with --git")
		}
	}

	statMode := StatMode("")
	if raw.OnlyStat != "" {
		statMode = StatMode(strings.ToLower(raw.OnlyStat))
		if _, ok := validStatModes[statMode]; !ok {
			return cfg, errors.Newf("invalid --only-stat value %q; expected one of: %s", raw.OnlyStat, strings.Join(AllStatModes, ", "))
		}
	}

	action := ActionDefault
	switch {
	case raw.OnlyStat != "":
		action = ActionStat
	case raw.OpenAsk:
		action = ActionAsk
	case raw.OpenSys:
		action = ActionSystem
	}

	searchMode := SearchModeDefault
	switch {
	case raw.Git:
		searchMode = SearchModeGit
	case raw.OnlyDirname:
		searchMode = SearchModeOnlyDirname
	case raw.OnlyFilename:
		searchMode = SearchModeOnlyFilename
	}

	cfg.Root = root
	cfg.Pattern = raw.Pattern
	cfg.Excludes = append([]string(nil), raw.Excludes...)
	cfg.Hidden = raw.Hidden
	cfg.MaxDepth = int(raw.MaxDepth)
	cfg.NoUI = raw.NoUI
	cfg.NoMulti = raw.NoMulti
	cfg.SearchMode = searchMode
	cfg.GitModes = gitModes
	cfg.Action = action
	cfg.StatMode = statMode
	cfg.OpenAsk = raw.OpenAsk
	cfg.OpenSys = raw.OpenSys
	cfg.NoFilename = raw.NoFilename
	cfg.OnlyFilename = raw.OnlyFilename
	cfg.OnlyDirname = raw.OnlyDirname

	return cfg, nil
}

func parseGitModes(raw []string, defaults []string, explicit bool) ([]string, error) {
	if len(raw) == 0 && !explicit {
		modes := make([]string, len(defaults))
		copy(modes, defaults)
		return modes, nil
	}

	seen := map[string]struct{}{}
	modes := make([]string, 0, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := validGitModes[part]; !ok {
				return nil, errors.Newf("invalid git mode %q; expected one of: %s", part, strings.Join(AllGitModes, ", "))
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			modes = append(modes, part)
		}
	}

	if len(modes) == 0 {
		modes = make([]string, len(defaults))
		copy(modes, defaults)
	}

	return modes, nil
}

func (cfg RootConfig) UsesRG() bool {
	return cfg.SearchMode == SearchModeDefault || cfg.SearchMode == SearchModeOnlyFilename
}
