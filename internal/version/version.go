// Package version holds the build-time version string.
package version

// Version is set at build time via -ldflags "-X pathrelay/internal/version.Version=...".
var Version = "dev"
