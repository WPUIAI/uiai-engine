package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func fillOTPTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.TwoFactor.Enabled = true
	cfg.TwoFactor.Profiles = map[string]config.TwoFactorProfile{
		"probe": {Provider: "totp", Secret: "JBSWY3DPEHPK3PXP", Digits: 6, Period: 5},
	}
	return cfg
}

func otpProbeServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input id="otp" autocomplete="one-time-code" inputmode="numeric"></body></html>`))
	}))
}

// #84 acceptance: an OTP-only wall raises the otp_input challenge class and
// enters the pending lease; fill_otp is admitted while pending, resolves the
// code at the fill moment through the local two-factor custody point (never
// before), refuses floors larger than the period, and a successful fill
// clears the pending challenge in-band. The resolved code never appears in
// the response payload.
func TestFillOTPRoundTripBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for fill_otp browser tests")
	}
	site := otpProbeServer()
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()

	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	challenge, err := session.DetectAuthChallenge()
	if err != nil || challenge == nil {
		t.Fatalf("otp wall not detected: %v %v", err, challenge)
	}
	if challenge.Class != "otp_input" {
		t.Fatalf("challenge class is %q, want otp_input", challenge.Class)
	}
	if session.LeaseState() != vision.LeaseAwaitingAuth {
		t.Fatalf("otp wall did not enter the pending lease: %s", session.LeaseState())
	}

	router := chi.NewRouter()
	MountSessionRoutes(router, fillOTPTestConfig(), manager)

	// Ordinary ops stay blocked; fill_otp is admitted while pending.
	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, httptest.NewRequest("POST", "/"+session.ID+"/eval", strings.NewReader(`{"js":"return 1;"}`)))
	if blocked.Code != http.StatusConflict || !strings.Contains(blocked.Body.String(), "auth_challenge_pending") {
		t.Fatalf("ordinary op not typed-pending: status=%d body=%.150s", blocked.Code, blocked.Body.String())
	}

	// A floor at or above the period can never be satisfied and is refused.
	tooLarge := httptest.NewRecorder()
	router.ServeHTTP(tooLarge, httptest.NewRequest("POST", "/"+session.ID+"/fill_otp",
		strings.NewReader(`{"selector":"#otp","profile":"probe","floor_s":5}`)))
	if tooLarge.Code != http.StatusBadRequest || !strings.Contains(tooLarge.Body.String(), "otp_floor_too_large") {
		t.Fatalf("floor >= period not refused: status=%d body=%.150s", tooLarge.Code, tooLarge.Body.String())
	}

	// Unknown profile is typed-refused, never a silent fill.
	unknown := httptest.NewRecorder()
	router.ServeHTTP(unknown, httptest.NewRequest("POST", "/"+session.ID+"/fill_otp",
		strings.NewReader(`{"selector":"#otp","profile":"missing"}`)))
	if unknown.Code != http.StatusBadRequest || !strings.Contains(unknown.Body.String(), "otp_unavailable") {
		t.Fatalf("unknown profile not typed-refused: status=%d body=%.150s", unknown.Code, unknown.Body.String())
	}

	// Fill-time resolution: the code is resolved now, not before; below-floor
	// validity waits for the next window and re-resolves (forced boundary).
	resolved := httptest.NewRecorder()
	router.ServeHTTP(resolved, httptest.NewRequest("POST", "/"+session.ID+"/fill_otp",
		strings.NewReader(`{"selector":"#otp","profile":"probe","floor_s":3}`)))
	body := resolved.Body.String()
	if resolved.Code != http.StatusOK || !strings.Contains(body, `"filled":true`) {
		t.Fatalf("fill_otp did not succeed: status=%d body=%.200s", resolved.Code, body)
	}
	if strings.Contains(body, `"code"`) {
		t.Fatalf("fill_otp response leaked a code channel: %.200s", body)
	}

	// The field actually received a 6-digit code (checked in-page, never echoed).
	valueOK, _, err := session.Eval(`return document.getElementById('otp').value.length === 6 && /^[0-9]{6}$/.test(document.getElementById('otp').value);`)
	if err != nil || valueOK != "true" {
		t.Fatalf("otp field not filled with a 6-digit code: %v %q", err, valueOK)
	}

	// A successful fill clears the pending challenge in-band; ops resume.
	if session.LeaseState() != vision.LeaseActive || session.CurrentAuthChallenge() != nil {
		t.Fatalf("pending otp challenge not cleared after fill: %s %v", session.LeaseState(), session.CurrentAuthChallenge())
	}
	after := httptest.NewRecorder()
	router.ServeHTTP(after, httptest.NewRequest("POST", "/"+session.ID+"/eval", strings.NewReader(`{"js":"return 1;"}`)))
	if after.Code == http.StatusConflict || strings.Contains(after.Body.String(), "auth_challenge_pending") {
		t.Fatalf("ops still blocked after in-band otp resolution: status=%d", after.Code)
	}
}

// #84 acceptance: fill_otp succeeds across a FORCED period boundary - the
// first resolution lands below the floor, the fill waits for the next
// window, re-resolves, and fills the fresh code (attempts == 2).
func TestFillOTPForcedWindowBoundaryBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for fill_otp browser tests")
	}
	site := otpProbeServer()
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()

	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := session.DetectAuthChallenge(); err != nil {
		t.Fatalf("detect: %v", err)
	}

	// Align to the last second of a 5s window so the first resolution is
	// guaranteed to land below the floor (expires_in == 1 < floor 3).
	period := 5
	for i := 0; i < 12; i++ {
		if time.Now().Unix()%int64(period) == int64(period)-1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	router := chi.NewRouter()
	MountSessionRoutes(router, fillOTPTestConfig(), manager)
	resolved := httptest.NewRecorder()
	router.ServeHTTP(resolved, httptest.NewRequest("POST", "/"+session.ID+"/fill_otp",
		strings.NewReader(`{"selector":"#otp","profile":"probe","floor_s":3}`)))
	body := resolved.Body.String()
	if resolved.Code != http.StatusOK || !strings.Contains(body, `"filled":true`) {
		t.Fatalf("forced-boundary fill failed: status=%d body=%.200s", resolved.Code, body)
	}
	if !strings.Contains(body, `"attempts":2`) {
		t.Fatalf("forced boundary did not take the wait-and-refetch path: %.200s", body)
	}
	valueOK, _, err := session.Eval(`return document.getElementById('otp').value.length === 6 && /^[0-9]{6}$/.test(document.getElementById('otp').value);`)
	if err != nil || valueOK != "true" {
		t.Fatalf("otp field not filled with the fresh code: %v %q", err, valueOK)
	}
	if session.LeaseState() != vision.LeaseActive {
		t.Fatalf("lease not restored after in-band fill: %s", session.LeaseState())
	}
}
