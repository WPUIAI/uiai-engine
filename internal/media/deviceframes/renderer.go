package deviceframes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
)

type Manifest struct {
	Version   int           `json:"version"`
	Sources   []Source      `json:"sources"`
	Frames    []FrameConfig `json:"frames"`
	Generated string        `json:"generated_at,omitempty"`
}

type Source struct {
	Name          string `json:"name"`
	Repo          string `json:"repo"`
	Commit        string `json:"commit"`
	LicenseStatus string `json:"license_status"`
	License       string `json:"license,omitempty"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Output struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FrameConfig struct {
	FrameID      string `json:"frame_id"`
	Title        string `json:"title,omitempty"`
	Source       string `json:"source"`
	SourceRef    string `json:"source_ref"`
	License      string `json:"license"`
	SVG          string `json:"svg"`
	SafeArea     Rect   `json:"safe_area"`
	Output       Output `json:"output"`
	CornerRadius int    `json:"corner_radius,omitempty"`
	Active       bool   `json:"active"`
}

type RenderRequest struct {
	FrameID string
	Image   []byte
	Fit     string // cover|contain
	Format  string // png|jpeg|jpg
	Quality int
	Scale   int
}

type RenderResult struct {
	Image  []byte
	Width  int
	Height int
	Format string
	Source string
}

type Renderer struct {
	manifestPath string
	baseDir      string
	mu           sync.RWMutex
	manifest     *Manifest
	frameCache   map[string]image.Image
}

func NewRenderer(manifestPath string) (*Renderer, error) {
	if manifestPath == "" {
		manifestPath = discoverManifestPath()
	}
	m, base, err := loadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return &Renderer{
		manifestPath: manifestPath,
		baseDir:      base,
		manifest:     m,
		frameCache:   map[string]image.Image{},
	}, nil
}

func discoverManifestPath() string {
	candidates := []string{
		"internal/media/deviceframes/manifest.json",
		"/home/wpuiai/uiai-engine/internal/media/deviceframes/manifest.json",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func loadManifest(path string) (*Manifest, string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- manifest path is resolved from bundled device-frame assets.
	if err != nil {
		return nil, "", fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, "", fmt.Errorf("parse manifest: %w", err)
	}
	if len(m.Frames) == 0 {
		return nil, "", fmt.Errorf("manifest has no frames")
	}
	return &m, filepath.Dir(path), nil
}

func (r *Renderer) Catalog() []FrameConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]FrameConfig, 0, len(r.manifest.Frames))
	for _, f := range r.manifest.Frames {
		if f.Active {
			out = append(out, f)
		}
	}
	return out
}

func (r *Renderer) Render(req RenderRequest) (*RenderResult, error) {
	if req.FrameID == "" {
		return nil, fmt.Errorf("frameId required")
	}
	if len(req.Image) == 0 {
		return nil, fmt.Errorf("image required")
	}
	if req.Fit == "" {
		req.Fit = "cover"
	}
	if req.Format == "" {
		req.Format = "png"
	}
	if req.Quality <= 0 {
		req.Quality = 90
	}
	if req.Scale <= 0 {
		req.Scale = 1
	}

	frameCfg, err := r.findFrame(req.FrameID)
	if err != nil {
		return nil, err
	}
	frameImg, err := r.loadFrameImage(frameCfg)
	if err != nil {
		return nil, err
	}

	srcImg, _, err := image.Decode(bytes.NewReader(req.Image))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	canvas := image.NewRGBA(frameImg.Bounds())
	draw.Draw(canvas, canvas.Bounds(), frameImg, image.Point{}, draw.Src)

	target := image.Rect(
		frameCfg.SafeArea.X,
		frameCfg.SafeArea.Y,
		frameCfg.SafeArea.X+frameCfg.SafeArea.W,
		frameCfg.SafeArea.Y+frameCfg.SafeArea.H,
	)
	fitted := fitImage(srcImg, frameCfg.SafeArea.W, frameCfg.SafeArea.H, req.Fit)

	if frameCfg.CornerRadius > 0 {
		fitted = applyRoundedMask(fitted, frameCfg.CornerRadius)
	}

	// center inside target area
	x := target.Min.X + (target.Dx()-fitted.Bounds().Dx())/2
	y := target.Min.Y + (target.Dy()-fitted.Bounds().Dy())/2
	draw.Draw(canvas, image.Rect(x, y, x+fitted.Bounds().Dx(), y+fitted.Bounds().Dy()), fitted, image.Point{}, draw.Over)

	outImg := image.Image(canvas)
	if req.Scale > 1 {
		scaled := image.NewRGBA(image.Rect(0, 0, canvas.Bounds().Dx()*req.Scale, canvas.Bounds().Dy()*req.Scale))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), canvas, canvas.Bounds(), draw.Over, nil)
		outImg = scaled
	}

	buf := &bytes.Buffer{}
	format := strings.ToLower(req.Format)
	switch format {
	case "jpg", "jpeg":
		format = "jpeg"
		err = jpeg.Encode(buf, outImg, &jpeg.Options{Quality: clamp(req.Quality, 50, 100)})
	default:
		format = "png"
		err = png.Encode(buf, outImg)
	}
	if err != nil {
		return nil, fmt.Errorf("encode output: %w", err)
	}

	return &RenderResult{
		Image:  buf.Bytes(),
		Width:  outImg.Bounds().Dx(),
		Height: outImg.Bounds().Dy(),
		Format: format,
		Source: fmt.Sprintf("%s@%s", frameCfg.Source, frameCfg.SourceRef),
	}, nil
}

func (r *Renderer) findFrame(id string) (*FrameConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, f := range r.manifest.Frames {
		if f.Active && f.FrameID == id {
			fc := f
			return &fc, nil
		}
	}
	return nil, fmt.Errorf("unknown frameId: %s", id)
}

func (r *Renderer) loadFrameImage(fc *FrameConfig) (image.Image, error) {
	r.mu.RLock()
	cached, ok := r.frameCache[fc.FrameID]
	r.mu.RUnlock()
	if ok {
		return cached, nil
	}

	svgPath := fc.SVG
	if !filepath.IsAbs(svgPath) {
		svgPath = filepath.Join(r.baseDir, svgPath)
	}
	if _, err := os.Stat(svgPath); err != nil {
		return nil, fmt.Errorf("svg not found: %s", svgPath)
	}
	cmd := exec.Command("convert", svgPath, "png:-") // #nosec G204 -- svgPath is from bundled device-frame assets and output is stdout.
	pngBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rasterize svg: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("decode rasterized frame: %w", err)
	}

	r.mu.Lock()
	r.frameCache[fc.FrameID] = img
	r.mu.Unlock()
	return img, nil
}

func fitImage(src image.Image, w, h int, fit string) image.Image {
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	if sw <= 0 || sh <= 0 || w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	scaleX := float64(w) / float64(sw)
	scaleY := float64(h) / float64(sh)
	scale := math.Max(scaleX, scaleY)
	if strings.EqualFold(fit, "contain") {
		scale = math.Min(scaleX, scaleY)
	}
	newW := int(math.Round(float64(sw) * scale))
	newH := int(math.Round(float64(sh) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), src, src.Bounds(), draw.Over, nil)

	if strings.EqualFold(fit, "cover") {
		// center crop to target size
		x0 := (newW - w) / 2
		y0 := (newH - h) / 2
		crop := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(crop, crop.Bounds(), resized, image.Point{X: x0, Y: y0}, draw.Src)
		return crop
	}
	return resized
}

func applyRoundedMask(src image.Image, radius int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if radius <= 0 {
		return src
	}
	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	r2 := float64(radius * radius)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			inCorner := false
			cx, cy := 0, 0
			switch {
			case x < radius && y < radius:
				inCorner, cx, cy = true, radius, radius
			case x >= w-radius && y < radius:
				inCorner, cx, cy = true, w-radius-1, radius
			case x < radius && y >= h-radius:
				inCorner, cx, cy = true, radius, h-radius-1
			case x >= w-radius && y >= h-radius:
				inCorner, cx, cy = true, w-radius-1, h-radius-1
			}
			if !inCorner {
				continue
			}
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy > r2 {
				out.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return out
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
