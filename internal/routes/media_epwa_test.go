package routes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/media"
)

func TestMediaJobResponseNeverExposesRawPathOrLegacyURL(t *testing.T) {
	job := &media.Job{
		ID: "media-1", Type: media.TypeDeviceMockup, Status: media.StatusComplete,
		ResultPath: "/tmp/raw.png", ResultURL: "https://raw.example/raw.png", CreatedAt: time.Now().UTC(),
		EPWADelivery: &epwadelivery.Delivery{
			State: epwadelivery.StateReady,
			EPWA: epwadelivery.EPWABinding{
				RecordURL:   "https://evidence.example/api/screenshot/share/package/",
				PortableURL: "https://evidence.example/api/screenshot/share/package/portable.zip",
			},
		},
	}
	body, err := json.Marshal(mediaJobResponse(job))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, forbidden := range []string{"/tmp/raw.png", "raw.example", "result_path", "result_url", "delivery_scope", "delivery_base_url"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("media response exposed %q: %s", forbidden, text)
		}
	}
	for _, required := range []string{"epwa_delivery", "artifact_url", "portable_url", "withheld_by_mandatory_epwa_delivery"} {
		if !strings.Contains(text, required) {
			t.Fatalf("media response missing %q: %s", required, text)
		}
	}
}

func TestMediaOutputDeliveryIsReadyOnlyWithHTTPSAndCompleteScope(t *testing.T) {
	dataDir := t.TempDir()
	outputPath := filepath.Join(dataDir, "frame.png")
	if err := os.WriteFile(outputPath, []byte("not-real-png-but-immutable"), 0o600); err != nil {
		t.Fatal(err)
	}
	deps := &mediaDeps{cfg: &config.Config{Storage: config.StorageConfig{DataDir: dataDir}}}
	job := &media.Job{
		ID: "media-ready", Type: media.TypeDeviceMockup, DeliveryScope: completeEvidenceScope(),
		DeliveryBaseURL: "https://evidence.example/base/",
	}
	delivery, err := deps.publishMediaOutput(job, outputPath, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if delivery.State != epwadelivery.StateReady {
		t.Fatalf("state=%s recovery=%s", delivery.State, delivery.RecoveryRef)
	}
	if !strings.HasPrefix(delivery.EPWA.RecordURL, "https://evidence.example/") || !strings.HasSuffix(delivery.EPWA.PortableURL, "/portable.zip") {
		t.Fatalf("unexpected EPWA URLs: %#v", delivery.EPWA)
	}

	job.ID = "media-blocked"
	job.DeliveryScope = completeEvidenceScope()
	job.DeliveryScope.WorkItems = nil
	blocked, err := deps.publishMediaOutput(job, outputPath, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if blocked.State != epwadelivery.StateBlocked || blocked.EPWA.RecordURL != "" || blocked.EPWA.PortableURL != "" {
		t.Fatalf("incomplete scope did not fail closed: %#v", blocked)
	}
}
