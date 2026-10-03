// Package web embeds the Management Center build produced from this directory
// so the control panel ships inside the server binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/management.html
var assets embed.FS

const managementAssetPath = "dist/management.html"

// ManagementHTML returns the embedded control panel build, or false when the
// frontend has not been built yet (`cd web && bun run build`).
func ManagementHTML() ([]byte, bool) {
	data, err := fs.ReadFile(assets, managementAssetPath)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// HasManagementHTML reports whether this binary carries a built control panel.
func HasManagementHTML() bool {
	_, ok := ManagementHTML()
	return ok
}
