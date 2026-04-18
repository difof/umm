package version

import "strings"

var (
	Version   = "dev"
	Commit    = ""
	BuildDate = ""
)

func VersionString() string {
	parts := []string{Version}
	if Commit != "" {
		parts = append(parts, Commit)
	}
	if BuildDate != "" {
		parts = append(parts, BuildDate)
	}

	return strings.Join(parts, " ")
}
