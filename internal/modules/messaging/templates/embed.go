package templates

import "embed"

// FS contains messaging email templates.
//
//go:embed *.html
var FS embed.FS
