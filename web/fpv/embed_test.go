package fpv

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEmbeddedFPVAssetsMatchCanonicalWriter(t *testing.T) {
	raw, ok := Asset("fpv-assets.json")
	if !ok {
		t.Fatal("asset manifest missing")
	}
	var manifest struct {
		Assets map[string]struct {
			SHA256 string `json:"sha256"`
		} `json:"assets"`
	}
	if json.Unmarshal(raw, &manifest) != nil {
		t.Fatal("asset manifest invalid")
	}
	for _, name := range []string{"fpv.js", "fpv.css"} {
		data, ok := Asset(name)
		if !ok || fmt.Sprintf("%x", sha256.Sum256(data)) != manifest.Assets[name].SHA256 {
			t.Fatal("generated asset digest mismatch")
		}
	}
	if _, ok := Asset("../index.html"); ok {
		t.Fatal("unlisted asset exposed")
	}
	script, _ := Asset("fpv.js")
	for _, required := range []string{"fps=${Math.max", "text('latencyMs', 'Not measured')", "text('fps', 'Live')"} {
		if !strings.Contains(string(script), required) {
			t.Fatal("quality override or truthful native-stream telemetry missing")
		}
	}
}
