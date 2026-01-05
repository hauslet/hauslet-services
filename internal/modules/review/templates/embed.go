package templates

import "embed"

// FS contains review email templates.
//
//go:embed *.html
var FS embed.FS
