package webui

import "embed"

// Dist is the built admin UI. CI / make build copy web/dist here before compiling.
//
//go:embed all:dist
var Dist embed.FS
