package email

import "embed"

// layoutFS contains shared email layouts owned by the platform layer.
//go:embed templates/*.html
var layoutFS embed.FS
