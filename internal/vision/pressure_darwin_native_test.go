//go:build darwin

package vision

import (
	"os"
	"testing"
)

// This must execute on macOS: cross-compilation and Linux parser fixtures do not
// prove the native ps adapter can observe a real running process.
func TestDarwinNativeRSSMeasuresRunningProcess(t *testing.T) {
	if rss := treeRSSKB(os.Getpid()); rss <= 0 {
		t.Fatal("native Darwin RSS sampler returned no observation for the running test process")
	}
	if rss := treeRSSKB(0); rss != 0 {
		t.Fatal("invalid process id was advertised as a measured process")
	}
}
