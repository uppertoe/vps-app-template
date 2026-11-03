// Package app embeds the files the binary needs from the repository root:
// the manifest, and the compliance folder in which a health app's
// attestation is recorded.
package app

import (
	"embed"
	"io/fs"
	"path"

	"github.com/uppertoe/vps-app-kit/manifest"
)

//go:embed app.yaml
var manifestYAML []byte

//go:embed compliance
var compliance embed.FS

// Manifest parses the embedded app.yaml. A malformed manifest stops the app
// at start-up and fails CI.
func Manifest() (*manifest.Manifest, error) { return manifest.Parse(manifestYAML) }

// HealthUnattested reports whether the app holds health information without
// a recorded attestation, in which case every page carries a warning banner.
func HealthUnattested(m *manifest.Manifest) bool {
	if m.DataClass != manifest.DataHealth {
		return false
	}
	if m.HealthAttestation == "" {
		return true
	}
	_, err := fs.Stat(compliance, path.Clean(m.HealthAttestation))
	return err != nil
}
