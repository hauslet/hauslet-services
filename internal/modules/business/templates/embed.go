package templates

import "embed"

// FS contains business-owned email templates.
//go:embed *.html
var FS embed.FS
