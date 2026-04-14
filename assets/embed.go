package assets

import "embed"

// DefaultsFS contains bootstrap files copied into the user config directory.
//
//go:embed defaults/**
var DefaultsFS embed.FS
