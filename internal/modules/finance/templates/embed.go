package templates

import "embed"

// FS contains finance-owned email templates.
//
//go:embed *.html
var FS embed.FS
