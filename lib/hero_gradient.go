package lib

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"
)

// Default hero edge gradients matching the design system
const (
	DefaultHeroEdgeLeftGrad  = "linear-gradient(to bottom, #8b9d9c 0%, #958a84 35%, #7e695d 70%, #644739 100%)"
	DefaultHeroEdgeRightGrad = "linear-gradient(to bottom, #262423 0%, #2d2b29 35%, #2a2826 70%, #1c1b1a 100%)"
)

type HeroGradients struct {
	Left  string
	Right string
}

var (
	heroGradientCache   = make(map[string]HeroGradients)
	heroGradientCacheMu sync.RWMutex
	httpClient          = &http.Client{
		Timeout: 3 * time.Second,
	}
)

// ExtractHeroGradients calculates left and right vertical 4-tier edge gradients from an image.
// It accepts local file paths or external HTTP/HTTPS URLs.
// If an image cannot be fetched or decoded, it safely returns default gradients without crashing.
func ExtractHeroGradients(imageSrc string) (string, string) {
	trimmed := strings.TrimSpace(imageSrc)
	if trimmed == "" {
		return DefaultHeroEdgeLeftGrad, DefaultHeroEdgeRightGrad
	}

	heroGradientCacheMu.RLock()
	cached, ok := heroGradientCache[trimmed]
	heroGradientCacheMu.RUnlock()
	if ok {
		return cached.Left, cached.Right
	}

	img, err := loadImage(trimmed)
	if err != nil {
		fmt.Printf("warning: hero 이미지 로드 실패 (%s): %v. 기본 그라디언트를 사용합니다.\n", trimmed, err)
		heroGradientCacheMu.Lock()
		heroGradientCache[trimmed] = HeroGradients{Left: DefaultHeroEdgeLeftGrad, Right: DefaultHeroEdgeRightGrad}
		heroGradientCacheMu.Unlock()
		return DefaultHeroEdgeLeftGrad, DefaultHeroEdgeRightGrad
	}

	leftGrad, rightGrad := CalculateHeroGradients(img)

	heroGradientCacheMu.Lock()
	heroGradientCache[trimmed] = HeroGradients{Left: leftGrad, Right: rightGrad}
	heroGradientCacheMu.Unlock()

	return leftGrad, rightGrad
}

func loadImage(imageSrc string) (image.Image, error) {
	if strings.HasPrefix(imageSrc, "//") {
		imageSrc = "https:" + imageSrc
	}

	if strings.HasPrefix(imageSrc, "http://") || strings.HasPrefix(imageSrc, "https://") {
		req, err := http.NewRequest("GET", imageSrc, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "image/webp,image/png,image/jpeg;q=0.9,image/*;q=0.8")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		img, _, err := image.Decode(io.LimitReader(resp.Body, 15*1024*1024))
		return img, err
	}

	// Try resolving local path (strip query params or hash if any)
	cleanPath := strings.SplitN(strings.SplitN(imageSrc, "?", 2)[0], "#", 2)[0]
	candidates := []string{
		strings.TrimPrefix(cleanPath, "/"),
		cleanPath,
		filepath.Join(".", strings.TrimPrefix(cleanPath, "/")),
		filepath.Join("public", strings.TrimPrefix(cleanPath, "/")),
		filepath.Join("assets", filepath.Base(cleanPath)),
		filepath.Join("..", strings.TrimPrefix(cleanPath, "/")),
		filepath.Join("..", "assets", filepath.Base(cleanPath)),
	}

	var foundPath string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			foundPath = c
			break
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("file not found in candidates: %s", imageSrc)
	}

	f, err := os.Open(foundPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", foundPath, err)
	}

	return img, nil
}

// CalculateHeroGradients computes the vertical 4-tier edge gradients from an image.Image.
func CalculateHeroGradients(img image.Image) (string, string) {
	if img == nil {
		return DefaultHeroEdgeLeftGrad, DefaultHeroEdgeRightGrad
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return DefaultHeroEdgeLeftGrad, DefaultHeroEdgeRightGrad
	}

	sampleW := int(float64(w) * 0.05)
	if sampleW < 1 {
		sampleW = 1
	}
	if sampleW > 20 {
		sampleW = 20
	}

	y1 := int(math.Floor(float64(h) * 0.25))
	y2 := int(math.Floor(float64(h) * 0.55))
	y3 := int(math.Floor(float64(h) * 0.85))

	sampleSlice := func(xStart, yStart, yEnd int) string {
		if yEnd <= yStart {
			yEnd = yStart + 1
		}
		if yEnd > h {
			yEnd = h
		}
		if yStart >= h {
			yStart = h - 1
		}

		xEnd := xStart + sampleW
		if xEnd > w {
			xEnd = w
		}
		if xStart < 0 {
			xStart = 0
		}

		var totalR, totalG, totalB, totalA uint64
		for y := yStart; y < yEnd; y++ {
			for x := xStart; x < xEnd; x++ {
				c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
				nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
				if nrgba.A == 0 {
					continue
				}
				a := uint64(nrgba.A)
				totalR += uint64(nrgba.R) * a
				totalG += uint64(nrgba.G) * a
				totalB += uint64(nrgba.B) * a
				totalA += a
			}
		}

		if totalA == 0 {
			return "rgb(28, 27, 26)"
		}

		avgR := int(math.Round(float64(totalR) / float64(totalA)))
		avgG := int(math.Round(float64(totalG) / float64(totalA)))
		avgB := int(math.Round(float64(totalB) / float64(totalA)))
		return fmt.Sprintf("rgb(%d, %d, %d)", avgR, avgG, avgB)
	}

	lTop := sampleSlice(0, 0, y1)
	lMidTop := sampleSlice(0, y1, y2)
	lMidBot := sampleSlice(0, y2, y3)
	lBot := sampleSlice(0, y3, h)

	rTop := sampleSlice(w-sampleW, 0, y1)
	rMidTop := sampleSlice(w-sampleW, y1, y2)
	rMidBot := sampleSlice(w-sampleW, y2, y3)
	rBot := sampleSlice(w-sampleW, y3, h)

	leftGrad := fmt.Sprintf("linear-gradient(to bottom, %s 0%%, %s 35%%, %s 70%%, %s 100%%)", lTop, lMidTop, lMidBot, lBot)
	rightGrad := fmt.Sprintf("linear-gradient(to bottom, %s 0%%, %s 35%%, %s 70%%, %s 100%%)", rTop, rMidTop, rMidBot, rBot)

	return leftGrad, rightGrad
}

// Suppress unused imports check
var _ io.Reader
