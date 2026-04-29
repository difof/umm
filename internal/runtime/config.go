package runtime

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

func (cfg RootConfig) UsesRG() bool {
	return cfg.SearchMode == SearchModeDefault || cfg.SearchMode == SearchModeOnlyFilename
}
