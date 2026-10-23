package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func TestGenericActiveContentIsAttachmentSandboxed(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	request := httptest.NewRequest(http.MethodPost, "https://evidence.example/report", nil)
	delivery, err := publishGenericEPWA(request, cfg, evidenceshare.GenericInput{
		ArtifactRef: "report:active-content", Revision: 1, Title: "Active content report",
		Kind: "generated_report", MediaType: "text/html", Extension: "html",
		Payload:    []byte(`<script>document.body.dataset.executed="true"</script><p>payload</p>`),
		CapturedAt: time.Now().UTC(), Scope: completeEvidenceScope(),
	})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountEvidenceShare(router, cfg)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/share/"+delivery.EPWA.PackageID+"/payload.html", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if disposition := recorder.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment;") {
		t.Fatalf("active content is not attachment-only: %q", disposition)
	}
	if policy := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "sandbox") || !strings.Contains(policy, "default-src 'none'") {
		t.Fatalf("active content missing sandbox policy: %q", policy)
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("active content missing nosniff")
	}
}
