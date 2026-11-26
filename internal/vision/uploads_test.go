package vision

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// #301 upload slice: the agent supplies file bytes; the engine writes them
// into the governed session file area and sets them on the page's file
// input. The chosen name is visible to the next read, non-file inputs are
// refused, and the file is served like any other session file.
func TestUploadFileToPageInput(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser upload regression")
	}
	payload := "uploaded-report-payload\n" + strings.Repeat("data ", 500)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body>
		<input id="attach" type="file">
		<div id="picked">none</div>
		<input id="notfile" type="text">
		<script>document.getElementById('attach').addEventListener('change', function(){ document.getElementById('picked').textContent = 'picked:' + this.files[0].name + ':' + this.files[0].size; });</script>
		</body></html>`)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	snap, err := session.UploadFile("#attach", "report.txt", base64.StdEncoding.EncodeToString([]byte(payload)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if snap == nil || snap.Size == 0 {
		t.Fatal("upload must return a capture")
	}

	res, rerr := session.page.Eval(`() => document.getElementById('picked').textContent`)
	if rerr != nil || !strings.Contains(res.Value.Str(), "picked:report.txt:"+fmt.Sprint(len(payload))) {
		t.Fatalf("upload not visible on the page: %v %v", rerr, res.Value.Str())
	}
	// The input itself carries the file.
	res2, rerr2 := session.page.Eval(`() => document.getElementById('attach').files[0].name`)
	if rerr2 != nil || res2.Value.Str() != "report.txt" {
		t.Fatalf("file input does not carry the upload: %v %v", rerr2, res2.Value.Str())
	}

	// The uploaded file is also a governed session file.
	files := session.Downloads()
	found := false
	for _, f := range files {
		if f.Name == "report.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("uploaded file must appear in the session file list: %+v", files)
	}
	served, serr := session.DownloadFileBytes("report.txt")
	if serr != nil || string(served) != payload {
		t.Fatalf("uploaded bytes not served back identically: %v", serr)
	}

	// Non-file inputs are refused.
	if _, err := session.UploadFile("#notfile", "x.txt", base64.StdEncoding.EncodeToString([]byte("x"))); err == nil {
		t.Fatal("upload to a non-file input must be refused")
	}
	// Bad base64 is refused.
	if _, err := session.UploadFile("#attach", "x.txt", "not-base64!!"); err == nil {
		t.Fatal("invalid base64 upload must be refused")
	}
}
