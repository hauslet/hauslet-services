package templates

import "embed"

// FS contains calendar email templates.
//
//go:embed *.html
var FS embed.FS
