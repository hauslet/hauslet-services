package templates

import "embed"

// FS contains auth-owned email templates.
//go:embed *.html
var FS embed.FS
