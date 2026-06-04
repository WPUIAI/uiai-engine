package routes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// VisionState holds the current page state.
type VisionState struct {
	URL      string                 `json:"url"`
	Title    string                 `json:"title"`
	Viewport map[string]int         `json:"viewport"`
	Analysis map[string]interface{} `json:"analysis"`
	Files    map[string]string      `json:"files,omitempty"`
	Timing   map[string]int64       `json:"timing,omitempty"`
}

// lastCapture stores the most recent screenshot for diff comparisons.
var lastCapture struct {
	mu   sync.Mutex
	data []byte
	url  string
	ts   time.Time
}

// MountVisionInteractive registers all vision interactive routes.
func MountVisionInteractive(r chi.Router, cfg *config.Config, pool *vision.Pool) {
	if pool == nil {
		return
	}

	// Core endpoints
	r.Get("/state", handleState(pool))
	r.Post("/capture", handleCapture(pool))
	r.Post("/look", handleLook(pool))
	r.Get("/look", handleLook(pool))
	r.Post("/inject", handleInject(pool))
	r.Get("/el", handleElement(pool))
	r.Post("/multi", handleMulti(pool))
	r.Get("/multi", handleMulti(pool))
	r.Get("/diff", handleDiff(pool))
	r.Get("/viewport", handleViewport(pool))

	// Added: analyze, critique, regression
	r.Get("/analyze", handleAnalyze(pool))
	r.Post("/critique", handleCritiqueCapture(pool))
	r.Post("/regression", handleRegression(pool))
}

// ═══════════════════════════════════════════════════════════
// CORE ENDPOINTS
// ═══════════════════════════════════════════════════════════

func handleState(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		state := buildState(page)
		writeJSON(w, 200, state)
	}
}

func handleCapture(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		data, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		analysis := runDOMAnalysis(page)
		storeCapture(data, "")

		result := map[string]interface{}{
			"screenshot": b64(data),
			"viewport":   map[string]int{"width": 1280, "height": 800},
			"analysis":   analysis,
			"timing": map[string]int64{
				"capture_ms": time.Since(start).Milliseconds(),
			},
		}

		writeJSON(w, 200, result)
	}
}

func handleLook(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageParam := r.URL.Query().Get("page")
		if pageParam == "" {
			// Try body for POST
			var body struct {
				Page string `json:"page"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			pageParam = body.Page
		}
		if pageParam == "" {
			writeJSON(w, 400, map[string]string{"error": "page param required"})
			return
		}

		targetURL := resolvePageURL(pageParam)

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		if err := page.Navigate(targetURL); err != nil {
			writeJSON(w, 500, map[string]string{"error": "navigation failed: " + err.Error()})
			return
		}

		page.Timeout(30 * time.Second).WaitLoad()

		data, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		title := evalStr(page, `() => document.title`)
		analysis := runDOMAnalysis(page)
		storeCapture(data, targetURL)

		result := map[string]interface{}{
			"url":        targetURL,
			"title":      title,
			"screenshot": b64(data),
			"viewport":   map[string]int{"width": 1280, "height": 800},
			"analysis":   analysis,
			"timing": map[string]int64{
				"look_ms": time.Since(start).Milliseconds(),
			},
		}

		writeJSON(w, 200, result)
	}
}

func handleInject(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CSS string `json:"css"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		css := body.CSS
		if css == "" {
			css = r.URL.Query().Get("css")
		}
		if css == "" {
			writeJSON(w, 400, map[string]string{"error": "css param required"})
			return
		}

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		// Remove previous injection, inject new CSS
		injectJS := `(css) => {
			const old = document.getElementById('wpuiai-injected');
			if (old) old.remove();
			const s = document.createElement('style');
			s.textContent = css;
			s.id = 'wpuiai-injected';
			document.head.appendChild(s);
			return 1;
		}`
		_, err = page.Eval(injectJS, css)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "inject failed: " + err.Error()})
			return
		}

		time.Sleep(100 * time.Millisecond)

		data, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		analysis := runDOMAnalysis(page)
		storeCapture(data, "")

		result := map[string]interface{}{
			"injected":   1,
			"inject_ms":  time.Since(start).Milliseconds(),
			"screenshot": b64(data),
			"analysis":   analysis,
		}

		writeJSON(w, 200, result)
	}
}

func handleElement(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sel := r.URL.Query().Get("sel")
		if sel == "" {
			writeJSON(w, 400, map[string]string{"error": "sel param required"})
			return
		}

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		// Try to find element and take element-level screenshot
		el, err := page.Element(sel)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "element not found: " + sel})
			return
		}

		// Get element info
		box, _ := el.Shape()
		tag := evalStr(page, `(sel) => { const e = document.querySelector(sel); return e ? e.tagName.toLowerCase() : '' }`, sel)
		text := evalStr(page, `(sel) => { const e = document.querySelector(sel); return e ? e.textContent.substring(0,200).trim() : '' }`, sel)

		// Element screenshot
		data, err := el.Screenshot(proto.PageCaptureScreenshotFormatPng, 85)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "element screenshot failed: " + err.Error()})
			return
		}

		var bounds interface{}
		if box != nil && len(box.Quads) > 0 {
			bounds = box.Quads[0]
		}

		result := map[string]interface{}{
			"selector":   sel,
			"tag":        tag,
			"text":       text,
			"bounds":     bounds,
			"screenshot": b64(data),
			"timing": map[string]int64{
				"element_ms": time.Since(start).Milliseconds(),
			},
		}

		writeJSON(w, 200, result)
	}
}

func handleMulti(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()
		viewports := []struct {
			name   string
			width  int
			height int
		}{
			{"desktop", 1440, 900},
			{"tablet", 768, 1024},
			{"mobile", 375, 812},
		}

		result := make(map[string]interface{})
		for _, vp := range viewports {
			page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
				Width: vp.width, Height: vp.height, DeviceScaleFactor: 1,
			})
			time.Sleep(200 * time.Millisecond)

			data, _ := screenshotPNG(page)
			analysis := runDOMAnalysis(page)

			result[vp.name] = map[string]interface{}{
				"screenshot": b64(data),
				"viewport":   map[string]int{"width": vp.width, "height": vp.height},
				"analysis":   analysis,
			}
		}

		// Restore desktop viewport
		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 1440, Height: 900, DeviceScaleFactor: 1,
		})

		result["timing"] = map[string]int64{
			"multi_ms": time.Since(start).Milliseconds(),
		}

		writeJSON(w, 200, result)
	}
}

func handleDiff(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		// Take current screenshot
		current, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		lastCapture.mu.Lock()
		previous := lastCapture.data
		prevURL := lastCapture.url
		prevTS := lastCapture.ts
		lastCapture.mu.Unlock()

		if previous == nil {
			storeCapture(current, "")
			writeJSON(w, 200, map[string]interface{}{
				"diff_pixels":  0,
				"has_previous": false,
				"note":         "No previous capture stored. Current frame saved.",
			})
			return
		}

		// Simple byte-level diff (pixel-accurate diff requires image decoding)
		diffBytes := 0
		minLen := len(previous)
		if len(current) < minLen {
			minLen = len(current)
		}
		for i := 0; i < minLen; i++ {
			if previous[i] != current[i] {
				diffBytes++
			}
		}
		diffBytes += abs(len(current) - len(previous))

		// Estimate pixel diff (PNG compressed, so byte diff ≈ pixel diff * compression ratio)
		estimatedPixels := diffBytes / 4 // rough estimate (RGBA)

		storeCapture(current, "")

		writeJSON(w, 200, map[string]interface{}{
			"diff_pixels":  estimatedPixels,
			"diff_bytes":   diffBytes,
			"has_previous": true,
			"previous_url": prevURL,
			"previous_ts":  prevTS.Format(time.RFC3339),
		})
	}
}

func handleViewport(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		width, _ := strconv.Atoi(r.URL.Query().Get("w"))
		height, _ := strconv.Atoi(r.URL.Query().Get("h"))
		if width == 0 {
			width = 1280
		}
		if height == 0 {
			height = 800
		}

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: width, Height: height, DeviceScaleFactor: 1,
		})

		writeJSON(w, 200, map[string]interface{}{
			"viewport": map[string]int{"width": width, "height": height},
		})
	}
}

// ═══════════════════════════════════════════════════════════
// NEW ENDPOINTS: analyze, critique, regression
// ═══════════════════════════════════════════════════════════

// handleAnalyze returns DOM analysis without taking a screenshot.
func handleAnalyze(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		url := evalStr(page, `() => window.location.href`)
		title := evalStr(page, `() => document.title`)
		analysis := runDOMAnalysis(page)

		// Count issues
		issueCount := 0
		for key, val := range analysis {
			if key == "headings" || key == "links" || key == "buttons" {
				continue
			}
			switch v := val.(type) {
			case int:
				issueCount += v
			case float64:
				issueCount += int(v)
			}
		}

		writeJSON(w, 200, map[string]interface{}{
			"url":      url,
			"title":    title,
			"issues":   issueCount,
			"analysis": analysis,
		})
	}
}

// handleCritiqueCapture navigates, captures desktop + optional mobile + DOM analysis.
// POST body: { "page": "url-or-slug", "mobile": true }
func handleCritiqueCapture(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Page   string `json:"page"`
			Mobile bool   `json:"mobile"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.Page == "" {
			body.Page = r.URL.Query().Get("page")
		}
		if body.Page == "" {
			writeJSON(w, 400, map[string]string{"error": "page param required"})
			return
		}

		targetURL := resolvePageURL(body.Page)

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		// Navigate (desktop viewport)
		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width: 1440, Height: 900, DeviceScaleFactor: 1,
		})
		if err := page.Navigate(targetURL); err != nil {
			writeJSON(w, 500, map[string]string{"error": "navigation failed: " + err.Error()})
			return
		}
		page.Timeout(30 * time.Second).WaitLoad()

		desktopData, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "desktop screenshot failed: " + err.Error()})
			return
		}

		title := evalStr(page, `() => document.title`)
		analysis := runDOMAnalysis(page)
		storeCapture(desktopData, targetURL)

		result := map[string]interface{}{
			"page":  body.Page,
			"url":   targetURL,
			"title": title,
			"desktop": map[string]interface{}{
				"screenshot": b64(desktopData),
				"viewport":   map[string]int{"width": 1440, "height": 900},
			},
			"analysis": analysis,
		}

		// Mobile capture if requested
		if body.Mobile {
			page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
				Width: 375, Height: 812, DeviceScaleFactor: 2,
			})
			time.Sleep(300 * time.Millisecond)

			mobileData, err := screenshotPNG(page)
			if err == nil {
				mobileAnalysis := runDOMAnalysis(page)
				result["mobile"] = map[string]interface{}{
					"screenshot": b64(mobileData),
					"viewport":   map[string]int{"width": 375, "height": 812},
					"analysis":   mobileAnalysis,
				}
			}

			// Restore desktop viewport
			page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
				Width: 1440, Height: 900, DeviceScaleFactor: 1,
			})
		}

		// Issue summary
		issueSummary := map[string]int{}
		totalIssues := 0
		for cat, val := range analysis {
			if cat == "headings" || cat == "links" || cat == "buttons" {
				continue
			}
			switch v := val.(type) {
			case int:
				if v > 0 {
					issueSummary[cat] = v
					totalIssues += v
				}
			case float64:
				iv := int(v)
				if iv > 0 {
					issueSummary[cat] = iv
					totalIssues += iv
				}
			}
		}

		result["issue_summary"] = issueSummary
		result["total_issues"] = totalIssues
		result["elapsed_ms"] = time.Since(start).Milliseconds()

		writeJSON(w, 200, result)
	}
}

// handleRegression captures current state and compares to previous.
// POST body: { "page": "url-or-slug", "threshold": 500 }
func handleRegression(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Page      string `json:"page"`
			Threshold int    `json:"threshold"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.Page == "" {
			body.Page = r.URL.Query().Get("page")
		}
		if body.Page == "" {
			writeJSON(w, 400, map[string]string{"error": "page param required"})
			return
		}
		if body.Threshold == 0 {
			body.Threshold = 500
		}

		targetURL := resolvePageURL(body.Page)

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		// Navigate and capture
		if err := page.Navigate(targetURL); err != nil {
			writeJSON(w, 500, map[string]string{"error": "navigation failed: " + err.Error()})
			return
		}
		page.Timeout(30 * time.Second).WaitLoad()

		current, err := screenshotPNG(page)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		analysis := runDOMAnalysis(page)

		// Compare to previous
		lastCapture.mu.Lock()
		previous := lastCapture.data
		lastCapture.mu.Unlock()

		diffPixels := 0
		hasPrevious := previous != nil

		if hasPrevious {
			diffBytes := 0
			minLen := len(previous)
			if len(current) < minLen {
				minLen = len(current)
			}
			for i := 0; i < minLen; i++ {
				if previous[i] != current[i] {
					diffBytes++
				}
			}
			diffBytes += abs(len(current) - len(previous))
			diffPixels = diffBytes / 4
		}

		regression := hasPrevious && diffPixels > body.Threshold

		storeCapture(current, targetURL)

		writeJSON(w, 200, map[string]interface{}{
			"page":         body.Page,
			"url":          targetURL,
			"regression":   regression,
			"diff_pixels":  diffPixels,
			"threshold":    body.Threshold,
			"has_previous": hasPrevious,
			"analysis":     analysis,
			"elapsed_ms":   time.Since(start).Milliseconds(),
		})
	}
}

// ═══════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════

func b64(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func screenshotPNG(page *rod.Page) ([]byte, error) {
	q := 85
	return page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatPng,
		Quality: &q,
	})
}

func evalStr(page *rod.Page, js string, args ...interface{}) string {
	var result *proto.RuntimeRemoteObject
	var err error
	if len(args) > 0 {
		result, err = page.Eval(js, args...)
	} else {
		result, err = page.Eval(js)
	}
	if err != nil || result == nil {
		return ""
	}
	return result.Value.Str()
}

func buildState(page *rod.Page) VisionState {
	url := evalStr(page, `() => window.location.href`)
	title := evalStr(page, `() => document.title`)

	viewportJSON := evalStr(page, `() => JSON.stringify({width: window.innerWidth, height: window.innerHeight, scrollHeight: document.body.scrollHeight})`)
	viewportMap := make(map[string]int)
	json.Unmarshal([]byte(viewportJSON), &viewportMap)

	analysis := runDOMAnalysis(page)

	return VisionState{
		URL:      url,
		Title:    title,
		Viewport: viewportMap,
		Analysis: analysis,
	}
}

func storeCapture(data []byte, url string) {
	lastCapture.mu.Lock()
	defer lastCapture.mu.Unlock()
	lastCapture.data = data
	lastCapture.url = url
	lastCapture.ts = time.Now()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func resolvePageURL(page string) string {
	pageMap := map[string]string{
		"wpuiai":          "https://wpuiai.com",
		"wpuiai-settings": "https://wpuiai.com/wp-admin/admin.php?page=wpuiai",
	}

	if mapped, ok := pageMap[page]; ok {
		return mapped
	}

	if strings.HasPrefix(page, "http://") || strings.HasPrefix(page, "https://") {
		return page
	}

	return "https://wpuiai.com/" + page
}

func runDOMAnalysis(page *rod.Page) map[string]interface{} {
	analysis := make(map[string]interface{})

	checks := []struct {
		name  string
		check string
	}{
		{"brokenImages", `Array.from(document.images).filter(i => !i.complete || i.naturalWidth === 0).length`},
		{"emptyContainers", `Array.from(document.querySelectorAll('[class*="container"],[class*="section"],[class*="block-"]')).filter(e => e.innerHTML.trim().length < 50).length`},
		{"links", `document.querySelectorAll('a[href]').length`},
		{"buttons", `document.querySelectorAll('button,.wp-block-button a').length`},
		{"images", `document.images.length`},
		{"forms", `document.forms.length`},
	}

	for _, check := range checks {
		result, err := page.Eval("() => " + check.check)
		if err == nil && result != nil {
			if n, ok := result.Value.MarshalJSON(); ok == nil {
				var count int
				json.Unmarshal(n, &count)
				analysis[check.name] = count
			}
		}
	}

	// Headings
	headingsResult, _ := page.Eval(`() => JSON.stringify(Array.from(document.querySelectorAll('h1,h2,h3')).slice(0, 10).map(e => e.tagName + ': ' + e.textContent.trim().substring(0,80)))`)
	if headingsResult != nil {
		if s := headingsResult.Value.Str(); s != "" {
			var headings []string
			json.Unmarshal([]byte(s), &headings)
			analysis["headings"] = headings
		}
	}

	// Page metrics
	metricsJS := `() => JSON.stringify({
		scrollHeight: document.body.scrollHeight,
		clientHeight: document.documentElement.clientHeight,
		scrollWidth: document.body.scrollWidth,
		clientWidth: document.documentElement.clientWidth
	})`
	metricsResult, _ := page.Eval(metricsJS)
	if metricsResult != nil {
		if s := metricsResult.Value.Str(); s != "" {
			var metrics map[string]int
			json.Unmarshal([]byte(s), &metrics)
			analysis["pageMetrics"] = metrics
		}
	}

	return analysis
}
