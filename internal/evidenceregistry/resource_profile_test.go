package evidenceregistry

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResourcePageProfiles(t *testing.T) {
	for _, tc := range []struct {
		name      string
		profile   ResourceProfile
		requested uint32
		fallback  uint32
		wantSize  uint32
		wantMedia MediaPosture
		wantErr   bool
	}{
		{name: "normal default", fallback: 50, wantSize: 50, wantMedia: MediaRefOnly},
		{name: "normal explicit", profile: ResourceNormal, requested: MaxPageSize, fallback: 50, wantSize: MaxPageSize, wantMedia: MediaRefOnly},
		{name: "normal over limit", profile: ResourceNormal, requested: MaxPageSize + 1, fallback: 50, wantErr: true},
		{name: "lowmem default clamps", profile: ResourceLowMem, fallback: 100, wantSize: MaxLowMemPageSize, wantMedia: MediaOmittedNonessential},
		{name: "lowmem requested clamps", profile: ResourceLowMem, requested: MaxPageSize, fallback: 50, wantSize: MaxLowMemPageSize, wantMedia: MediaOmittedNonessential},
		{name: "unknown", profile: "emergency-plus", requested: 1, fallback: 50, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile, size, media, err := resourcePage(tc.profile, tc.requested, tc.fallback)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resourcePage error=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			wantProfile := tc.profile
			if wantProfile == "" {
				wantProfile = ResourceNormal
			}
			if profile != wantProfile || size != tc.wantSize || media != tc.wantMedia {
				t.Fatalf("resourcePage=(%s,%d,%s) want=(%s,%d,%s)", profile, size, media, wantProfile, tc.wantSize, tc.wantMedia)
			}
		})
	}
}

func TestListLowMemCannotBeOverriddenByLargerPage(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, Config{Path: filepath.Join(t.TempDir(), "index.sqlite3"), ProjectRef: "project:uiai-engine"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manifest := loadGoldenManifest(t)
	if _, err := store.Index(ctx, IndexInput{Manifest: manifest, ManifestSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}); err != nil {
		t.Fatal(err)
	}
	page, err := store.List(ctx, Query{ResourceProfile: ResourceLowMem, PageSize: MaxPageSize})
	if err != nil {
		t.Fatal(err)
	}
	if page.ResourceProfile != ResourceLowMem || page.MediaPosture != MediaOmittedNonessential || page.PageSize != MaxLowMemPageSize || len(page.Rows) != 1 {
		t.Fatalf("unexpected lowmem page: %#v", page)
	}
}
