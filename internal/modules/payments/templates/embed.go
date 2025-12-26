package templates

import "embed"

// FS contains payment-owned email templates.
//
//go:embed *.html
var FS embed.FS
