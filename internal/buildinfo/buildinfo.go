// Package buildinfo carries the binary version and build-time, injectable at
// release via -ldflags:
//
//	-X github.com/WPUIAI/uiai-engine/internal/buildinfo.Version=2.0.25
//	-X github.com/WPUIAI/uiai-engine/internal/buildinfo.BuildTime=...
//
// Runtime surfaces (health, root info) report these values so drift/rollback
// automation trusts the machine surface (#317).
package buildinfo

import "runtime/debug"

// buildSetting aliases the debug build-setting record so vcsFromSettings stays
// testable without a real binary build.
type buildSetting = debug.BuildSetting

// buildInfo is a seam over runtime/debug.ReadBuildInfo for tests.
var buildInfo = func() (*debug.BuildInfo, bool) {
	info, ok := debug.ReadBuildInfo()
	return info, ok
}

var (
	// Version defaults to "dev" for unstamped builds; cmd/uiai-engine
	// propagates the canonical in-source version literal here when no release
	// ldflags stamp was injected, so unstamped binaries never claim a released
	// version they were not built from.
	Version   = "dev"
	BuildTime = "unknown"
	// Commit identifies the exact source revision of a plain `go build`
	// (#295), derived from the binary's embedded VCS build settings; release
	// ldflags stamps do not set it.
	Commit = ""
)

// VCSBuildInfo derives the short commit and commit time from the binary's
// embedded build settings (#295: `go build` records vcs.revision and
// vcs.time when building from a git checkout without -buildvcs=false).
func VCSBuildInfo() (commit, builtAt string, ok bool) {
	info, valid := buildInfo()
	if !valid {
		return "", "", false
	}
	return vcsFromSettings(info.Settings)
}

// vcsFromSettings extracts vcs.revision/vcs.time from build settings; a
// revision shorter than 7 chars is not a usable commit id.
func vcsFromSettings(settings []buildSetting) (string, string, bool) {
	var rev, when string
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			when = s.Value
		}
	}
	if len(rev) < 7 || when == "" {
		return "", "", false
	}
	return rev[:7], when, true
}
