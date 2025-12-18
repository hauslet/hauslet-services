package templates

import "embed"

// FS contains property-owned email templates.
//
//go:embed *.html
var FS embed.FS
