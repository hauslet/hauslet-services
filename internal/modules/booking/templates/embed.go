package templates

import "embed"

// FS contains booking email templates.
//
//go:embed *.html
var FS embed.FS
