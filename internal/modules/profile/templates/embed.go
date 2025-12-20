package templates

import "embed"

// FS contains profile-owned email templates.
//
//go:embed *.html
var FS embed.FS
