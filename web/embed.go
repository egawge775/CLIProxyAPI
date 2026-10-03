// Package web embeds the Management Center build produced from this directory
// so the control panel can ship inside the server binary.
//
// `dist/` is a build output: it exists only after `cd web && bun run build`.
// The embed therefore targets the optional `all:dist` tree - a checkout without
// a build still compiles and falls back to the upstream release panel.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

const managementAssetPath = "dist/management.html"

// ManagementHTML returns the embedded control panel build, or false when the
// frontend has not been built in this checkout.
func ManagementHTML() ([]byte, bool) {
	data, err := fs.ReadFile(assets, managementAssetPath)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// HasManagementHTML reports whether this binary carries a locally built control panel.
func HasManagementHTML() bool {
	_, ok := ManagementHTML()
	return ok
}
