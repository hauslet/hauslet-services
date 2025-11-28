package logger

import (
	"github.com/fatih/color"
	"github.com/go-pkgz/lgr"
)

// New creates a new logger with colorized output
// This logger is configured for development with colored console output
func New() *lgr.Logger {
	colorizer := lgr.Mapper{
		ErrorFunc:  func(s string) string { return color.New(color.FgHiRed).Sprint(s) },
		WarnFunc:   func(s string) string { return color.New(color.FgHiYellow).Sprint(s) },
		InfoFunc:   func(s string) string { return color.New(color.FgHiWhite).Sprint(s) },
		DebugFunc:  func(s string) string { return color.New(color.FgWhite).Sprint(s) },
		CallerFunc: func(s string) string { return color.New(color.FgBlue).Sprint(s) },
		TimeFunc:   func(s string) string { return color.New(color.FgCyan).Sprint(s) },
	}

	return lgr.New(
		lgr.Msec,
		lgr.LevelBraces,
		lgr.Map(colorizer),
		lgr.Debug,
	)
}

// NewWithOptions creates a logger with custom options
// Use this if you need more control over logger configuration
func NewWithOptions(opts ...lgr.Option) *lgr.Logger {
	return lgr.New(opts...)
}

// NewProduction creates a logger suitable for production use
// No colors, structured logging, minimal debug output
func NewProduction() *lgr.Logger {
	return lgr.New(
		lgr.Msec,
		lgr.LevelBraces,
		// No colorizer for production
	)
}
