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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"
)

type HeroGradients struct {
	Left  string
	Right string
}

var (
	DefaultHeroEdgeLeftGrad  string
	DefaultHeroEdgeRightGrad string

	defaultGradientsOnce sync.Once
	heroGradientCache    = make(map[string]HeroGradients)
	heroGradientCacheMu  sync.RWMutex
	httpClient           = &http.Client{
		Timeout: 3 * time.Second,
	}
)

func init() {
	initDefaultHeroGradients()
}

// initDefaultHeroGradients reads --neutral-300 (Light mode) and --neutral-700 (Dark mode)
// from layout/styles/color-palette.css, constructs synthetic 1x1 images for each,
// and runs them through CalculateHeroGradients so that the fallback hero-color
// is extracted by the exact same Go pipeline.
func initDefaultHeroGradients() {
	defaultGradientsOnce.Do(func() {
		lightColor := parsePaletteNeutralColor("--neutral-300", color.NRGBA{R: 212, G: 212, B: 212, A: 255})
		darkColor := parsePaletteNeutralColor("--neutral-700", color.NRGBA{R: 64, G: 64, B: 64, A: 255})

		lightImg := image.NewNRGBA(image.Rect(0, 0, 100, 100))
		darkImg := image.NewNRGBA(image.Rect(0, 0, 100, 100))
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				lightImg.SetNRGBA(x, y, lightColor)
				darkImg.SetNRGBA(x, y, darkColor)
			}
		}

		lightLeft, lightRight := CalculateHeroGradients(lightImg)
		darkLeft, darkRight := CalculateHeroGradients(darkImg)

		// Extract the rgb(...) color extracted by CalculateHeroGradients
		lightLeftRGB := extractFirstRGB(lightLeft, "rgb(212, 212, 212)")
		lightRightRGB := extractFirstRGB(lightRight, "rgb(212, 212, 212)")
		darkLeftRGB := extractFirstRGB(darkLeft, "rgb(64, 64, 64)")
		darkRightRGB := extractFirstRGB(darkRight, "rgb(64, 64, 64)")

		// Use light-dark() so the browser selects the Go-extracted Light (--neutral-300) or Dark (--neutral-700)
		// gradient color automatically based on color-scheme: light dark.
		leftStop := fmt.Sprintf("light-dark(%s, %s)", lightLeftRGB, darkLeftRGB)
		rightStop := fmt.Sprintf("light-dark(%s, %s)", lightRightRGB, darkRightRGB)

		DefaultHeroEdgeLeftGrad = fmt.Sprintf("linear-gradient(to bottom, %s 0%%, %s 35%%, %s 70%%, %s 100%%)", leftStop, leftStop, leftStop, leftStop)
		DefaultHeroEdgeRightGrad = fmt.Sprintf("linear-gradient(to bottom, %s 0%%, %s 35%%, %s 70%%, %s 100%%)", rightStop, rightStop, rightStop, rightStop)
	})
}

func extractFirstRGB(gradStr, fallback string) string {
	idx := strings.Index(gradStr, "rgb(")
	if idx == -1 {
		return fallback
	}
	end := strings.Index(gradStr[idx:], ")")
	if end == -1 {
		return fallback
	}
	return gradStr[idx : idx+end+1]
}

// parsePaletteNeutralColor parses an achromatic oklch(L 0 0) variable from layout/styles/color-palette.css
// and converts it to sRGB color.NRGBA.
func parsePaletteNeutralColor(varName string, fallback color.NRGBA) color.NRGBA {
	candidates := []string{
		"layout/styles/color-palette.css",
		"../layout/styles/color-palette.css",
	}
	var content []byte
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			content = b
			break
		}
	}
	if len(content) == 0 {
		return fallback
	}

	pattern := regexp.QuoteMeta(varName) + `\s*:\s*oklch\(\s*([0-9.]+)`
	re := regexp.MustCompile(pattern)
	m := re.FindSubmatch(content)
	if len(m) < 2 {
		return fallback
	}

	L, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return fallback
	}

	// Convert achromatic OKLCH (L, C=0, H=0) -> OKLab (L, a=0, b=0) -> Linear sRGB -> sRGB
	lLinear := L * L * L
	srgb := linearToSRGB(lLinear)
	v := uint8(math.Round(math.Max(0, math.Min(1, srgb)) * 255.0))
	return color.NRGBA{R: v, G: v, B: v, A: 255}
}

func linearToSRGB(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
}

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
