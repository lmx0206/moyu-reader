// Package version exposes the application version string.
package version

// Version is the application version. It defaults to "dev" for local builds and
// is overridden at release time via the linker:
//
//	go build -ldflags "-X moyureader/internal/version.Version=0.5.1"
//
// The Release workflow injects the git tag (without its leading "v"), so a
// published binary reports its real version.
var Version = "dev"
