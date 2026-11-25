package buildinfo

import (
	"runtime/debug"
	"strings"
	"testing"
)

// #295: a plain `go build` must surface its commit and commit time from
// embedded VCS build settings instead of printing "unknown".
func TestVCSBuildInfoFromSettings(t *testing.T) {
	cases := []struct {
		name    string
		setting []buildSetting
		commit  string
		built   string
		ok      bool
	}{
		{
			name: "git checkout",
			setting: []buildSetting{
				{Key: "vcs.revision", Value: "0031ad8abcdef1234567890abcdef1234567890"},
				{Key: "vcs.time", Value: "2026-11-25T21:07:00Z"},
			},
			commit: "0031ad8",
			built:  "2026-11-25T21:07:00Z",
			ok:     true,
		},
		{
			name: "no vcs settings",
			setting: []buildSetting{
				{Key: "GOARCH", Value: "arm64"},
			},
			ok: false,
		},
		{
			name: "revision only",
			setting: []buildSetting{
				{Key: "vcs.revision", Value: "0031ad8abcdef"},
			},
			ok: false,
		},
		{
			name: "short revision rejected",
			setting: []buildSetting{
				{Key: "vcs.revision", Value: "abc"},
				{Key: "vcs.time", Value: "2026-11-25T21:07:00Z"},
			},
			ok: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			commit, built, ok := vcsFromSettings(tc.setting)
			if ok != tc.ok {
				t.Fatalf("ok=%v, want %v", ok, tc.ok)
			}
			if commit != tc.commit || built != tc.built {
				t.Fatalf("got (%q, %q), want (%q, %q)", commit, built, tc.commit, tc.built)
			}
		})
	}
}

func TestVCSBuildInfoSeam(t *testing.T) {
	orig := buildInfo
	defer func() { buildInfo = orig }()
	buildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "f07cb89abcdef1234567890abcdef123456789"},
				{Key: "vcs.time", Value: "2026-11-25T21:30:00Z"},
			},
		}, true
	}
	commit, built, ok := VCSBuildInfo()
	if !ok || commit != "f07cb89" || built != "2026-11-25T21:30:00Z" {
		t.Fatalf("VCSBuildInfo=%q %q %v, want f07cb89 2026-11-25T21:30:00Z true", commit, built, ok)
	}

	buildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	if _, _, ok := VCSBuildInfo(); ok {
		t.Fatal("missing build info must not report VCS identity")
	}
}

// Guards the honest-output contract: no surface ever claims a commit when
// nothing was recorded.
func TestCommitDefaultEmpty(t *testing.T) {
	if strings.Contains(Commit, "unknown") {
		t.Fatalf("Commit must default empty, got %q", Commit)
	}
}
