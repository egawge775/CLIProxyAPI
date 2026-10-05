package managementasset

import (
	webassets "github.com/router-for-me/CLIProxyAPI/v8/web"
)

// EmbeddedManagementHTML returns the control panel built from web/ and embedded
// in this binary, or false when the frontend has not been built yet.
func EmbeddedManagementHTML() ([]byte, bool) {
	return webassets.ManagementHTML()
}

// HasEmbeddedManagementHTML reports whether this binary serves a locally built panel.
func HasEmbeddedManagementHTML() bool {
	return webassets.HasManagementHTML()
}
