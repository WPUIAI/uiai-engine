package routes

import (
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/media"
)

// #298: media helpers disable cleanly when media.script_dir is absent —
// typed refusal naming the missing configuration, never a host-path fallback.
func TestMediaHelpersDisabledWithoutScriptDir(t *testing.T) {
	d := &mediaDeps{cfg: &config.Config{}}
	job := &media.Job{ID: "t298", Device: "iphone-15", URLs: []string{"https://example.com/"}}
	_, err := d.produceMockup(job)
	if err == nil || !strings.Contains(err.Error(), "media.script_dir not configured") {
		t.Fatalf("mockup without script_dir must refuse cleanly, got %v", err)
	}
	_, err = d.produceGIF(job)
	if err == nil || !strings.Contains(err.Error(), "media.script_dir not configured") {
		t.Fatalf("gif without script_dir must refuse cleanly, got %v", err)
	}
}
