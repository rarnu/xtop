// Package version holds the xtop version string. It is kept in a small,
// standalone package so both the TUI about dialog and the CLI --version flag
// can share the same value.
package version

// Version is the current xtop release version. It may be overridden at link
// time with -ldflags "-X xtop/internal/version.Version=<value>".
var Version = "0.1.0"
