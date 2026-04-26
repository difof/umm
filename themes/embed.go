package themes

import "embed"

// FS exposes the committed built-in theme catalog.
//
//go:embed *.yml
var FS embed.FS
