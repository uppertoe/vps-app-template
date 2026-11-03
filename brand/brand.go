// Package brand holds the app's branding: names, colours, fonts, the logo
// and the icons. The template ships a neutral brand with every slot filled,
// so an unbranded app builds and deploys. Replace any file to rebrand; a
// file you leave alone keeps the neutral default.
package brand

import (
	"embed"
	"io/fs"

	"gopkg.in/yaml.v3"
)

//go:embed brand.css logo.svg icon.svg icon-32.png icon-180.png fonts
var files embed.FS

//go:embed brand.yaml
var brandYAML []byte

// Info is brand.yaml.
type Info struct {
	Organisation string `yaml:"organisation"`
	ContactEmail string `yaml:"contact_email"`
}

// Load parses brand.yaml.
func Load() (Info, error) {
	var i Info
	err := yaml.Unmarshal(brandYAML, &i)
	return i, err
}

// Files are served under /brand/ with content-hashed URLs.
func Files() fs.FS { return files }
