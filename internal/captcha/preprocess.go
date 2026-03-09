package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"golang.org/x/image/draw"
)

// Preprocess cleans a captcha image for better OCR/VLM accuracy.
// Returns base64-encoded PNG.
func Preprocess(imgBase64, imgType string, cfg *PreprocessConfig) (string, error) {
	if cfg == nil {
		return imgBase64, nil // no preprocessing requested
	}

	raw, err := base64.StdEncoding.DecodeString(imgBase64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	// 1. Upscale
	if cfg.Upscale > 1 {
		img = upscale(img, cfg.Upscale)
	}

	// 2. Grayscale
	gray := toGrayscale(img)

	// 3. Threshold → binary
	if cfg.Threshold > 0 {
		gray = threshold(gray, uint8(cfg.Threshold))
	}

	// 4. Morphological open (remove thin lines)
	if cfg.MorphologyKernel > 0 {
		gray = morphOpen(gray, cfg.MorphologyKernel)
	}

	// 5. Connected component filter (remove grid fragments)
	if cfg.ComponentMinArea > 0 {
		gray = filterComponents(gray, cfg.ComponentMinArea, cfg.ComponentMaxAspect)
	}

	// Encode result
	var buf bytes.Buffer
	if err := png.Encode(&buf, gray); err != nil {
		return "", fmt.Errorf("encode png: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// upscale resizes image by factor using nearest-neighbor (preserves pixel edges).
func upscale(src image.Image, factor int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx()*factor, bounds.Dy()*factor
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// toGrayscale converts any image to 8-bit grayscale.
func toGrayscale(src image.Image) *image.Gray {
	bounds := src.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			// ITU-R BT.601 luma
			lum := uint8((19595*r + 38470*g + 7471*b + 1<<15) >> 24)
			gray.SetGray(x, y, color.Gray{Y: lum})
		}
	}
	return gray
}

// threshold binarizes: pixels darker than thresh become black (ink), rest white.
// Result is inverted binary: ink=255, bg=0 (for component analysis).
func threshold(src *image.Gray, thresh uint8) *image.Gray {
	bounds := src.Bounds()
	out := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if src.GrayAt(x, y).Y < thresh {
				out.SetGray(x, y, color.Gray{Y: 255}) // dark pixel = ink
			} else {
				out.SetGray(x, y, color.Gray{Y: 0}) // light pixel = background
			}
		}
	}
	return out
}

// morphOpen performs binary morphological opening (erode then dilate)
// to remove thin structures like grid lines.
func morphOpen(src *image.Gray, kernel int) *image.Gray {
	eroded := erode(src, kernel)
	return dilate(eroded, kernel)
}

func erode(src *image.Gray, k int) *image.Gray {
	bounds := src.Bounds()
	out := image.NewGray(bounds)
	half := k / 2
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			minVal := uint8(255)
			for dy := -half; dy <= half; dy++ {
				for dx := -half; dx <= half; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
						v := src.GrayAt(nx, ny).Y
						if v < minVal {
							minVal = v
						}
					} else {
						minVal = 0
					}
				}
			}
			out.SetGray(x, y, color.Gray{Y: minVal})
		}
	}
	return out
}

func dilate(src *image.Gray, k int) *image.Gray {
	bounds := src.Bounds()
	out := image.NewGray(bounds)
	half := k / 2
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			maxVal := uint8(0)
			for dy := -half; dy <= half; dy++ {
				for dx := -half; dx <= half; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
						v := src.GrayAt(nx, ny).Y
						if v > maxVal {
							maxVal = v
						}
					}
				}
			}
			out.SetGray(x, y, color.Gray{Y: maxVal})
		}
	}
	return out
}

// filterComponents removes connected components that are too small or too elongated.
// Input: inverted binary (ink=255, bg=0). Output: same format.
func filterComponents(src *image.Gray, minArea, maxAspect int) *image.Gray {
	if maxAspect <= 0 {
		maxAspect = 8
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Flatten to 2D slice for flood-fill labeling
	pixels := make([][]uint8, h)
	for y := 0; y < h; y++ {
		pixels[y] = make([]uint8, w)
		for x := 0; x < w; x++ {
			pixels[y][x] = src.GrayAt(x+bounds.Min.X, y+bounds.Min.Y).Y
		}
	}

	labels := make([][]int, h)
	for y := 0; y < h; y++ {
		labels[y] = make([]int, w)
	}

	type compInfo struct {
		area                     int
		minX, minY, maxX, maxY   int
	}
	var components []compInfo
	label := 0

	// 8-connected flood fill labeling
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if pixels[y][x] == 255 && labels[y][x] == 0 {
				label++
				info := compInfo{minX: w, minY: h, maxX: 0, maxY: 0}
				stack := []struct{ x, y int }{{x, y}}
				for len(stack) > 0 {
					p := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if p.x < 0 || p.x >= w || p.y < 0 || p.y >= h {
						continue
					}
					if pixels[p.y][p.x] != 255 || labels[p.y][p.x] != 0 {
						continue
					}
					labels[p.y][p.x] = label
					info.area++
					if p.x < info.minX { info.minX = p.x }
					if p.y < info.minY { info.minY = p.y }
					if p.x > info.maxX { info.maxX = p.x }
					if p.y > info.maxY { info.maxY = p.y }

					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if dx == 0 && dy == 0 { continue }
							stack = append(stack, struct{ x, y int }{p.x + dx, p.y + dy})
						}
					}
				}
				components = append(components, info)
			}
		}
	}

	// Build set of labels to keep
	keepLabels := map[int]bool{}
	for i, c := range components {
		if c.area < minArea {
			continue
		}
		cw := c.maxX - c.minX + 1
		ch := c.maxY - c.minY + 1
		aspect := cw
		if ch > cw { aspect = ch }
		minor := cw
		if ch < cw { minor = ch }
		if minor == 0 { minor = 1 }
		if aspect/minor > maxAspect {
			continue // too elongated — likely a grid line
		}
		keepLabels[i+1] = true // labels are 1-indexed
	}

	// Rebuild output — invert back to normal (ink=dark, bg=white)
	out := image.NewGray(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if keepLabels[labels[y][x]] {
				out.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{Y: 0}) // ink = black
			} else {
				out.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{Y: 255}) // bg = white
			}
		}
	}
	return out
}
