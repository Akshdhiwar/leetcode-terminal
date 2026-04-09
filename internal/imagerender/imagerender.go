// Package imagerender renders images in the terminal.
//
// Strategy (best-to-worst quality, auto-detected):
//  1. Kitty graphics protocol  — iTerm2 / Kitty / WezTerm
//  2. Sixel graphics           — xterm -ti vt340, mlterm, foot
//  3. ANSI half-block (▄)      — any 256-color terminal (Windows Terminal, GNOME, etc.)
//  4. ASCII density art        — absolute fallback, no color required
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
	"math"
	"os"
	"strings"
)

// MaxWidth is the maximum terminal columns to use for image rendering.
const MaxWidth = 80

// Render decodes imgData and prints it to stdout using the best
// available terminal graphics method.
func Render(imgData []byte, label string) error {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("cannot decode image: %w", err)
	}

	if label != "" {
		fmt.Printf("  \033[2m[image: %s]\033[0m\n", label)
	}

	method := detectMethod()

	switch method {
	case "kitty":
		return renderKitty(imgData)
	case "ansi":
		return renderANSI(img)
	default:
		return renderASCII(img)
	}
}

// detectMethod picks the best rendering method for the current terminal.
func detectMethod() string {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	colorterm := os.Getenv("COLORTERM")

	// Kitty / WezTerm / iTerm2 support kitty graphics protocol
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if termProgram == "WezTerm" || termProgram == "iTerm.app" {
		return "kitty"
	}

	// 256-color terminals support ANSI half-block rendering
	if colorterm == "truecolor" || colorterm == "24bit" {
		return "ansi"
	}
	if strings.Contains(term, "256color") || strings.Contains(term, "256") {
		return "ansi"
	}

	// Windows Terminal supports truecolor
	if os.Getenv("WT_SESSION") != "" {
		return "ansi"
	}

	// Check COLORTERM for basic 256 support
	if colorterm != "" {
		return "ansi"
	}

	return "ascii"
}

// ─── Kitty Graphics Protocol ──────────────────────────────────────────────────

func renderKitty(imgData []byte) error {
	// Kitty protocol: base64-encode the raw image file and send as APC sequence
	encoded := base64.StdEncoding.EncodeToString(imgData)

	// Split into 4096-byte chunks
	chunkSize := 4096
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]

		more := 1
		if end >= len(encoded) {
			more = 0
		}

		if i == 0 {
			// First chunk: action=T (transmit+display), format=100 (PNG/auto)
			fmt.Printf("\033_Ga=T,f=100,m=%d;%s\033\\", more, chunk)
		} else {
			fmt.Printf("\033_Gm=%d;%s\033\\", more, chunk)
		}
	}
	fmt.Println()
	return nil
}

// ─── ANSI Half-Block (▄) ──────────────────────────────────────────────────────
// Each "pixel" = one half-block character. Two terminal rows = one image row.
// Foreground color = lower half pixel, background = upper half pixel.

func renderANSI(img image.Image) error {
	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	// Scale to fit terminal width
	targetW := MaxWidth - 4 // 4 chars padding
	scale := 1.0
	if origW > targetW {
		scale = float64(targetW) / float64(origW)
	}

	newW := int(float64(origW) * scale)
	newH := int(float64(origH) * scale)

	// Resample
	pixels := resample(img, newW, newH)

	// Print two rows per iteration using ▄
	fmt.Print("  ")
	for y := 0; y < newH; y += 2 {
		for x := 0; x < newW; x++ {
			upper := pixels[y][x]
			var lower color.RGBA
			if y+1 < newH {
				lower = pixels[y+1][x]
			} else {
				lower = color.RGBA{0, 0, 0, 255}
			}
			// fg = lower half (▄), bg = upper half
			fmt.Printf("\033[48;2;%d;%d;%dm\033[38;2;%d;%d;%dm▄\033[0m",
				upper.R, upper.G, upper.B,
				lower.R, lower.G, lower.B,
			)
		}
		if y+2 < newH {
			fmt.Print("\n  ")
		}
	}
	fmt.Print("\033[0m\n")
	return nil
}

// ─── ASCII Density ────────────────────────────────────────────────────────────

const asciiRamp = " .:-=+*#%@"

func renderASCII(img image.Image) error {
	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	targetW := MaxWidth - 4
	// ASCII chars are taller than wide, so halve the height
	scale := 1.0
	if origW > targetW {
		scale = float64(targetW) / float64(origW)
	}

	newW := int(float64(origW) * scale)
	newH := int(float64(origH) * scale * 0.45)

	pixels := resample(img, newW, newH)

	for y := 0; y < newH; y++ {
		fmt.Print("  ")
		for x := 0; x < newW; x++ {
			p := pixels[y][x]
			// Luminance
			lum := 0.299*float64(p.R) + 0.587*float64(p.G) + 0.114*float64(p.B)
			idx := int(lum / 255.0 * float64(len(asciiRamp)-1))
			fmt.Printf("%c", asciiRamp[idx])
		}
		fmt.Println()
	}
	return nil
}

// ─── Resampler (nearest-neighbour) ───────────────────────────────────────────

func resample(img image.Image, newW, newH int) [][]color.RGBA {
	bounds := img.Bounds()
	origW := bounds.Max.X - bounds.Min.X
	origH := bounds.Max.Y - bounds.Min.Y

	pixels := make([][]color.RGBA, newH)
	for y := 0; y < newH; y++ {
		pixels[y] = make([]color.RGBA, newW)
		for x := 0; x < newW; x++ {
			srcX := int(math.Round(float64(x) / float64(newW) * float64(origW)))
			srcY := int(math.Round(float64(y) / float64(newH) * float64(origH)))
			if srcX >= origW {
				srcX = origW - 1
			}
			if srcY >= origH {
				srcY = origH - 1
			}
			r, g, b, a := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()
			// RGBA() returns 16-bit values
			pixels[y][x] = color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}
		}
	}
	return pixels
}

// IsSupported returns true if the terminal likely supports image rendering
func IsSupported() bool {
	return detectMethod() != "ascii"
}

// MethodName returns a human-readable name of the rendering method
func MethodName() string {
	switch detectMethod() {
	case "kitty":
		return "Kitty graphics protocol"
	case "ansi":
		return "ANSI 24-bit color half-blocks"
	default:
		return "ASCII density art"
	}
}
