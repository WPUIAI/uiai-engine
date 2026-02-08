package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

type VisionState struct {
	URL      string                 `json:"url"`
	Title    string                 `json:"title"`
	Viewport map[string]int         `json:"viewport"`
	Analysis map[string]interface{} `json:"analysis"`
	Files    map[string]string      `json:"files,omitempty"`
	Timing   map[string]int64       `json:"timing,omitempty"`
}

func MountVisionInteractive(r chi.Router, cfg *config.Config, pool *vision.Pool) {
	if pool == nil {
		return
	}

	r.Get("/state", handleState(pool))
	r.Post("/capture", handleCapture(pool))
	r.Get("/look", handleLook(pool))
	r.Post("/inject", handleInject(pool))
	r.Get("/el", handleElement(pool))
	r.Get("/multi", handleMulti(pool))
	r.Get("/diff", handleDiff(pool))
	r.Get("/viewport", handleViewport(pool))
}

func handleState(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		// Get current URL and title
		currentURL, _ := page.Eval(`() => window.location.href`)
		title, _ := page.Eval(`() => document.title`)

		// Get viewport
		viewport, _ := page.Eval(`() => ({width: window.innerWidth, height: window.innerHeight, scrollHeight: document.body.scrollHeight})`)

		// Parse viewport
		viewportMap := make(map[string]int)
		if vp, ok := viewport.Value.MarshalJSON(); ok == nil {
			json.Unmarshal(vp, &viewportMap)
		}

		// Run DOM analysis
		analysis := runDOMAnalysis(page)

		state := VisionState{
			URL:      getString(currentURL),
			Title:    getString(title),
			Viewport: viewportMap,
			Analysis: analysis,
		}

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

		// Take screenshot
		data, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: gson(85),
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		// Run DOM analysis
		analysis := runDOMAnalysis(page)

		result := map[string]interface{}{
			"screenshot": base64Encode(data),
			"width":      1280,
			"height":     800,
			"viewport":   map[string]int{"width": 1280, "height": 800},
			"analysis":   analysis,
			"timing": map[string]int64{
				"capture": time.Since(start).Milliseconds(),
			},
		}

		writeJSON(w, 200, result)
	}
}

func handleLook(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageParam := r.URL.Query().Get("page")
		if pageParam == "" {
			writeJSON(w, 400, map[string]string{"error": "page param required"})
			return
		}

		// Resolve page to URL
		targetURL := resolvePageURL(pageParam)

		page, err := pool.GetPage()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "failed to get page"})
			return
		}
		defer pool.ReleasePage(page)

		start := time.Now()

		// Navigate
		if err := page.Navigate(targetURL); err != nil {
			writeJSON(w, 500, map[string]string{"error": "navigation failed: " + err.Error()})
			return
		}

		// Wait for load
		page.Timeout(30 * time.Second).WaitLoad()

		// Take screenshot
		data, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: gson(85),
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		// Get title
		title, _ := page.Eval(`() => document.title`)

		// Run DOM analysis
		analysis := runDOMAnalysis(page)

		result := map[string]interface{}{
			"url":       targetURL,
			"title":     getString(title),
			"screenshot": base64Encode(data),
			"viewport":  map[string]int{"width": 1280, "height": 800},
			"analysis":  analysis,
			"timing": map[string]int64{
				"look": time.Since(start).Milliseconds(),
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}

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

		// Inject CSS
		injectJS := "(css) => { const s = document.createElement('style'); s.textContent = css; s.id = 'wpuiai-injected'; document.head.appendChild(s); return 1; }"
		_, err = page.Eval(injectJS, css)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "inject failed: " + err.Error()})
			return
		}

		// Small delay for CSS to apply
		time.Sleep(100 * time.Millisecond)

		// Take screenshot
		data, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: gson(85),
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "screenshot failed: " + err.Error()})
			return
		}

		// Run analysis
		analysis := runDOMAnalysis(page)

		result := map[string]interface{}{
			"injected":   1,
			"inject_ms":  time.Since(start).Milliseconds(),
			"screenshot": base64Encode(data),
			"analysis":  analysis,
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

		// Get element info (simplified - just find the selector)
		elInfo, _ := page.Eval("() => { const e = document.querySelector('" + sel + "'); if(!e) return null; return {tag: e.tagName, text: e.textContent.substring(0,100)}; }")

		result := map[string]interface{}{
			"selector": sel,
			"element":  getString(elInfo),
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

		// Desktop
		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: 1440, Height: 900, DeviceScaleFactor: 1})
		time.Sleep(200 * time.Millisecond)
		desktopData, _ := page.Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.PageCaptureScreenshotFormatPng, Quality: gson(85)})

		// Tablet
		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: 768, Height: 1024, DeviceScaleFactor: 1})
		time.Sleep(200 * time.Millisecond)
		tabletData, _ := page.Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.PageCaptureScreenshotFormatPng, Quality: gson(85)})

		// Mobile
		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: 375, Height: 812, DeviceScaleFactor: 1})
		time.Sleep(200 * time.Millisecond)
		mobileData, _ := page.Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.PageCaptureScreenshotFormatPng, Quality: gson(85)})

		result := map[string]interface{}{
			"desktop": map[string]interface{}{
				"screenshot": base64Encode(desktopData),
				"viewport":   map[string]int{"width": 1440, "height": 900},
			},
			"tablet": map[string]interface{}{
				"screenshot": base64Encode(tabletData),
				"viewport":  map[string]int{"width": 768, "height": 1024},
			},
			"mobile": map[string]interface{}{
				"screenshot": base64Encode(mobileData),
				"viewport":  map[string]int{"width": 375, "height": 812},
			},
		}

		writeJSON(w, 200, result)
	}
}

func handleDiff(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simplified: just return that diff requires storing previous state
		writeJSON(w, 200, map[string]interface{}{
			"diffPixels":  0,
			"note":        "diff requires storing previous capture - not yet implemented",
			"implemented": false,
		})
	}
}

func handleViewport(pool *vision.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wParam := r.URL.Query().Get("w")
		hParam := r.URL.Query().Get("h")

		width, _ := strconv.Atoi(wParam)
		height, _ := strconv.Atoi(hParam)

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

		page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: width, Height: height, DeviceScaleFactor: 1})

		writeJSON(w, 200, map[string]interface{}{
			"viewport": map[string]int{"width": width, "height": height},
		})
	}
}

func base64Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return strings.TrimPrefix(encodeBase64(data), "/9j/")
}

func encodeBase64(b []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, len(b)*2)
	for i := 0; i < len(b); i++ {
		result = append(result, base64Chars[b[i]>>2])
		result = append(result, base64Chars[(b[i]&0x03)<<4])
		if i+1 < len(b) {
			result = append(result, base64Chars[b[i+1]>>4])
			result = append(result, base64Chars[(b[i+1]&0x0F)<<2])
		} else {
			result = append(result, '=')
			result = append(result, '=')
		}
		i++
	}
	return string(result)
}

func getString(v *proto.RuntimeRemoteObject) string {
	if v == nil {
		return ""
	}
	return v.Value.Str()
}

func gson(v int) *int {
	return &v
}

func resolvePageURL(page string) string {
	// Handle special page names
	pageMap := map[string]string{
		"wpuiai":           "https://wpuiai.com",
		"wpuiai-settings": "https://wpuiai.com/wp-admin/admin.php?page=wpuiai",
	}

	if mapped, ok := pageMap[page]; ok {
		return mapped
	}

	// If it looks like a URL, return as-is
	if strings.HasPrefix(page, "http://") || strings.HasPrefix(page, "https://") {
		return page
	}

	// Assume it's a page on wpuiai.com
	return "https://wpuiai.com/" + page
}

func runDOMAnalysis(page *rod.Page) map[string]interface{} {
	analysis := make(map[string]interface{})

	// Basic DOM checks
	checks := []struct {
		name  string
		check string
	}{
		{"brokenImages", `Array.from(document.images).filter(i => !i.complete || i.naturalWidth === 0).length`},
		{"emptyContainers", `Array.from(document.querySelectorAll('[class*="container"],[class*="section"],[class*="block-"]')).filter(e => e.innerHTML.trim().length < 50).length`},
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
	headingsResult, _ := page.Eval(`() => Array.from(document.querySelectorAll('h1,h2,h3')).slice(0, 10).map(e => e.tagName + ': ' + e.textContent.trim().substring(0,80))`)
	if headingsResult != nil {
		if s := headingsResult.Value.Str(); s != "" {
			analysis["headings"] = s
		}
	}

	// Links
	linksResult, _ := page.Eval(`() => document.querySelectorAll('a[href]').length`)
	if linksResult != nil {
		if n, ok := linksResult.Value.MarshalJSON(); ok == nil {
			var count int
			json.Unmarshal(n, &count)
			analysis["links"] = count
		}
	}

	// Buttons
	buttonsResult, _ := page.Eval(`() => document.querySelectorAll('button,.wp-block-button a').length`)
	if buttonsResult != nil {
		if n, ok := buttonsResult.Value.MarshalJSON(); ok == nil {
			var count int
			json.Unmarshal(n, &count)
			analysis["buttons"] = count
		}
	}

	return analysis
}
