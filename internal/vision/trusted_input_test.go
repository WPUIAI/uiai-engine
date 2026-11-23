package vision

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// This is synthetic silence, never customer media or generated narration.
func trustedInputFixture(t *testing.T) *Session {
	t.Helper()
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for native browser input regression")
	}
	var wav bytes.Buffer
	wav.WriteString("RIFF")
	binary.Write(&wav, binary.LittleEndian, uint32(8036))
	wav.WriteString("WAVEfmt ")
	binary.Write(&wav, binary.LittleEndian, uint32(16))
	binary.Write(&wav, binary.LittleEndian, uint16(1))
	binary.Write(&wav, binary.LittleEndian, uint16(1))
	binary.Write(&wav, binary.LittleEndian, uint32(8000))
	binary.Write(&wav, binary.LittleEndian, uint32(8000))
	binary.Write(&wav, binary.LittleEndian, uint16(1))
	binary.Write(&wav, binary.LittleEndian, uint16(8))
	wav.WriteString("data")
	binary.Write(&wav, binary.LittleEndian, uint32(8000))
	wav.Write(bytes.Repeat([]byte{128}, 8000))
	html := fmt.Sprintf(`<!doctype html><title>Synthetic trusted input</title>
<button id="play">Play synthetic audio</button><button id="key">Keyboard activation</button>
<audio id="audio" loop preload="auto" src="data:audio/wav;base64,%s"></audio>
<script>window.metrics={trusted:false,active:false,resolved:false,rejected:'',keyClicks:0,keyTrusted:false,ups:0};
const a=document.querySelector('#audio');document.querySelector('#play').onclick=e=>{metrics.trusted=e.isTrusted;metrics.active=navigator.userActivation.isActive;a.play().then(()=>metrics.resolved=true).catch(e=>metrics.rejected=e.name);};
const k=document.querySelector('#key');k.onclick=e=>{metrics.keyClicks++;metrics.keyTrusted=e.isTrusted;};k.onkeyup=()=>metrics.ups++;</script>`, base64.StdEncoding.EncodeToString(wav.Bytes()))
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	t.Cleanup(site.Close)
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	manager := NewSessionManager(pool)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func TestTrustedClickStartsDecodedAudio(t *testing.T) {
	session := trustedInputFixture(t)
	if _, err := session.Click("#play"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	result, err := session.page.Eval(`() => JSON.stringify({...window.metrics,paused:document.querySelector('#audio').paused,time:document.querySelector('#audio').currentTime,error:document.querySelector('#audio').error?.code??0})`)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Trusted, Active, Resolved, Paused bool
		Time                              float64
		Error                             int
		Rejected                          string
	}
	if err := json.Unmarshal([]byte(result.Value.Str()), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Trusted || !got.Active || !got.Resolved || got.Paused || got.Time <= 0 || got.Error != 0 || got.Rejected != "" {
		t.Fatalf("native click did not start decoded audio: %+v", got)
	}
}

func TestNamedSpaceCompletesNativeStroke(t *testing.T) {
	session := trustedInputFixture(t)
	button, err := session.page.Element("#key")
	if err != nil {
		t.Fatal(err)
	}
	if err = button.Focus(); err != nil {
		t.Fatal(err)
	}
	if _, err = session.Press("Space"); err != nil {
		t.Fatal(err)
	}
	result, err := session.page.Eval(`() => JSON.stringify(window.metrics)`)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		KeyClicks  int
		KeyTrusted bool
		Ups        int
	}
	if err = json.Unmarshal([]byte(result.Value.Str()), &got); err != nil {
		t.Fatal(err)
	}
	if got.KeyClicks != 1 || !got.KeyTrusted || got.Ups != 1 {
		t.Fatalf("named Space did not complete exactly one trusted stroke: %+v", got)
	}
}
