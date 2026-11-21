// Package buildinfo carries the binary version and build-time, injectable at
// release via -ldflags:
//
//	-X github.com/WPUIAI/uiai-engine/internal/buildinfo.Version=2.0.25
//	-X github.com/WPUIAI/uiai-engine/internal/buildinfo.BuildTime=...
//
// Runtime surfaces (health, root info) report these values so drift/rollback
// automation trusts the machine surface (#317).
package buildinfo

var (
	// Version defaults to "dev" for unstamped builds; cmd/uiai-engine
	// propagates the canonical in-source version literal here when no release
	// ldflags stamp was injected, so unstamped binaries never claim a released
	// version they were not built from.
	Version   = "dev"
	BuildTime = "unknown"
)
