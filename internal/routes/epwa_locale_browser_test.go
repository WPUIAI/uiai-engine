package routes

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

// Opt-in browser proof uses real package assembly and serving, not a mocked API.
// The caller supplies an owner-owned output directory to retain DOM/pixel evidence.
func TestEPWALocalizedBrowser(t *testing.T) {
	output := os.Getenv("UIAI_EPWA_BROWSER_PROOF_DIR")
	if output == "" {
		t.Skip("set UIAI_EPWA_BROWSER_PROOF_DIR for real Chromium proof")
	}
	chrome, err := exec.LookPath("chromium-browser")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	root := evidenceShareDir(cfg)
	generic, err := evidenceshare.AssembleGeneric(root, evidenceshare.GenericInput{ArtifactRef: "artifact:browser-fixture", Revision: 1, Title: "Original source title", Kind: "report", MediaType: "application/json", Extension: "json", Payload: []byte("{\"fixture\":true}"), CapturedAt: time.Now().UTC(), Scope: completeEvidenceScope()})
	if err != nil {
		t.Fatal(err)
	}
	pixels := image.NewRGBA(image.Rect(0, 0, 320, 180))
	for y := 0; y < 180; y++ {
		for x := 0; x < 320; x++ {
			pixels.Set(x, y, color.RGBA{uint8(x % 256), uint8(y), 160, 255})
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, pixels); err != nil {
		t.Fatal(err)
	}
	screenshot, err := evidenceshare.Assemble(root, evidenceshare.Input{Screenshot: data.Bytes(), Format: "png", Width: 320, Height: 180, CapturedAt: time.Now().UTC(), Scope: completeEvidenceScope()})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountEvidenceShare(router, cfg)
	server := httptest.NewServer(router)
	defer server.Close()
	for kind, directory := range map[string]string{"generic": generic.Directory, "screenshot": screenshot.Directory} {
		for _, language := range []string{"en", "es", "ar"} {
			t.Run(kind+"-"+language, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
				defer cancel()
				name := kind + "-" + language
				url := server.URL + "/share/" + filepath.Base(directory) + "/?lang=" + language
				cmd := exec.CommandContext(ctx, "node", "../../scripts/epwa-browser-proof.mjs", url, output, name, chrome)
				if log, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("Chromium: %v: %s", err, log)
				}
				dom, err := os.ReadFile(filepath.Join(output, name+"-390.html"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(dom), "lang=\""+language+"\"") {
					t.Fatal("requested locale not applied")
				}
				if language == "ar" && !strings.Contains(string(dom), "dir=\"rtl\"") {
					t.Fatal("RTL not applied")
				}
				if kind == "generic" {
					labels := map[string]string{"en": "Read-only artifact loaded", "es": "Artefacto de solo lectura cargado", "ar": "تم تحميل القطعة للقراءة فقط"}
					if !strings.Contains(string(dom), labels[language]) || !strings.Contains(string(dom), "Original source title") {
						t.Fatal("generic record did not render localized UI with original evidence")
					}
				} else if !strings.Contains(string(dom), "screenshot.png") {
					t.Fatal("screenshot record missing")
				}
			})
		}
	}
}
