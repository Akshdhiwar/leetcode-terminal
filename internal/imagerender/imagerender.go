// Package imagerender renders images in the terminal using the best available method.
//
// Quality tiers (auto-detected):
//  1. Kitty graphics protocol — Kitty, WezTerm, iTerm2 (native image, best)
//  2. ANSI 24-bit half-blocks — Windows Terminal, GNOME, VS Code (Lanczos resampled)
//  3. ASCII density art       — any terminal (Lanczos + luminance mapping)
//
// Lanczos3 resampling (via github.com/nfnt/resize) gives photo-quality
// downscaling — same algorithm used by Photoshop and ffmpeg.
package imagerender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
)

// Render decodes imgData and prints it inline using the best available method.
func Render(imgData []byte, label string) error {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("cannot decode image: %w", err)
	}

	method := detectMethod()

	if label != "" {
		fmt.Printf("  \033[2m[%s]\033[0m\n", label)
	}

	switch method {
	case "kitty":
		return renderKitty(imgData)
	case "ansi":
		return renderANSI(img)
	default:
		return renderASCII(img)
	}
}

func terminalWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 20 {
			return n
		}
	}
	if os.Getenv("WT_SESSION") != "" {
		return 220
	}
	return 160
}

func detectMethod() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	tp := os.Getenv("TERM_PROGRAM")
	if tp == "WezTerm" || tp == "iTerm.app" {
		return "kitty"
	}
	if os.Getenv("WT_SESSION") != "" {
		return "ansi"
	}
	ct := os.Getenv("COLORTERM")
	if ct == "truecolor" || ct == "24bit" {
		return "ansi"
	}
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") {
		return "ansi"
	}
	if tp == "vscode" || tp == "Hyper" {
		return "ansi"
	}
	if ct != "" {
		return "ansi"
	}
	return "ascii"
}

// ─── Kitty Protocol ───────────────────────────────────────────────────────────

func renderKitty(imgData []byte) error {
	encoded := base64.StdEncoding.EncodeToString(imgData)
	const chunkSize = 4096
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		more := 1
		if end >= len(encoded) {
			more = 0
		}
		if i == 0 {
			fmt.Printf("\033_Ga=T,f=100,m=%d;%s\033\\", more, encoded[i:end])
		} else {
			fmt.Printf("\033_Gm=%d;%s\033\\", more, encoded[i:end])
		}
	}
	fmt.Println()
	return nil
}

// ─── ANSI Half-Block Renderer ─────────────────────────────────────────────────
// Uses Lanczos3 for highest quality downscaling.
// Each terminal character = 2 vertical image pixels (▄ half-block trick).

func renderANSI(img image.Image) error {
	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	// Target width: 85% of terminal, capped at 200 columns
	maxCols := int(float64(terminalWidth()) * 0.85)
	if maxCols > 200 {
		maxCols = 200
	}
	if maxCols < 20 {
		maxCols = 20
	}

	// Terminal chars are ~2× taller than wide.
	// We need newW columns and newH*2 pixel rows (2 rows per char row).
	// Aspect: newH_chars = newH_px / 2
	// So: newH_px / 2 / newW = origH / origW  →  newH_px = newW * origH / origW * 2 * 0.5
	// The 0.5 corrects for the 2:1 char aspect ratio.
	newW := maxCols
	if origW < newW {
		newW = origW
	}
	// pixel height for half-block rendering (2 px per char row, chars are 2:1)
	newH := newW * origH / origW // this is already in pixel rows
	// round to even
	if newH%2 != 0 {
		newH++
	}

	// Lanczos3 resize — sharpest downscaling available
	resized := resize.Resize(uint(newW), uint(newH), img, resize.Lanczos3)

	fmt.Printf("  \033[2m╔%s╗\033[0m\n", strings.Repeat("═", newW))

	for y := 0; y < newH; y += 2 {
		fmt.Print("  \033[2m║\033[0m")
		for x := 0; x < newW; x++ {
			upper := blendWhite(toRGBA(resized.At(x, y)))
			var lower color.RGBA
			if y+1 < newH {
				lower = blendWhite(toRGBA(resized.At(x, y+1)))
			} else {
				lower = color.RGBA{255, 255, 255, 255}
			}
			fmt.Printf("\033[48;2;%d;%d;%dm\033[38;2;%d;%d;%dm▄\033[0m",
				upper.R, upper.G, upper.B,
				lower.R, lower.G, lower.B,
			)
		}
		fmt.Printf("\033[2m║\033[0m\n")
	}

	fmt.Printf("  \033[2m╚%s╝\033[0m\n", strings.Repeat("═", newW))
	return nil
}

// ─── ASCII Density ────────────────────────────────────────────────────────────

const asciiDense = `$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\|()1{}[]?-_+~<>i!lI;:,"^` + "`" + `'. `

func renderASCII(img image.Image) error {
	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	maxCols := terminalWidth() - 6
	newW := maxCols
	if origW < newW {
		newW = origW
	}
	// Correct for char aspect ratio
	newH := newW * origH / origW * 10 / 22

	resized := resize.Resize(uint(newW), uint(newH), img, resize.Lanczos3)
	ramp := []rune(asciiDense)

	fmt.Printf("  %s\n", strings.Repeat("─", newW))
	for y := 0; y < newH; y++ {
		fmt.Print("  ")
		for x := 0; x < newW; x++ {
			c := blendWhite(toRGBA(resized.At(x, y)))
			lum := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
			// Map: bright → sparse chars (end of ramp), dark → dense chars (start)
			idx := int((1.0 - lum/255.0) * float64(len(ramp)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(ramp) {
				idx = len(ramp) - 1
			}
			fmt.Printf("%c", ramp[idx])
		}
		fmt.Println()
	}
	fmt.Printf("  %s\n", strings.Repeat("─", newW))
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func blendWhite(c color.RGBA) color.RGBA {
	if c.A == 255 {
		return c
	}
	a := float64(c.A) / 255.0
	return color.RGBA{
		R: uint8(float64(c.R)*a + 255*(1-a)),
		G: uint8(float64(c.G)*a + 255*(1-a)),
		B: uint8(float64(c.B)*a + 255*(1-a)),
		A: 255,
	}
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func IsSupported() bool      { return detectMethod() != "ascii" }
func MethodName() string {
	switch detectMethod() {
	case "kitty":
		return "Kitty graphics protocol (lossless)"
	case "ansi":
		return "ANSI 24-bit half-blocks + Lanczos3"
	default:
		return "ASCII density art + Lanczos3"
	}
}
