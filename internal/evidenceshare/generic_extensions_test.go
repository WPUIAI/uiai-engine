package evidenceshare

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAssembleGenericSupportsRuntimeAndMediaExtensions(t *testing.T) {
	cases := []struct {
		extension string
		mediaType string
	}{
		{"wasm", "application/wasm"},
		{"js", "application/javascript"},
		{"gif", "image/gif"},
	}
	for _, tc := range cases {
		t.Run(tc.extension, func(t *testing.T) {
			result, err := AssembleGeneric(t.TempDir(), GenericInput{
				ArtifactRef: "artifact:" + tc.extension, Revision: 1, Title: "Generic " + tc.extension,
				Kind: "runtime_binary", MediaType: tc.mediaType, Extension: tc.extension,
				Payload: []byte("immutable-" + tc.extension), CapturedAt: time.Now().UTC(), Scope: completeScope(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := ValidateGenericPackage(result.Directory, result.PackageID); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(result.Directory, "payload."+tc.extension)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
