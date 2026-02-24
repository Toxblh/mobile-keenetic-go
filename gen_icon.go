//go:build ignore

// Run: go run gen_icon.go
// Generates Icon.png (1024x1024) required by `fyne package`.

package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const size = 1024
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	cx, cy := size/2, size/2

	fill(img, color.RGBA{0, 0, 0, 0}) // transparent background
	drawCircle(img, cx, cy, 480, color.RGBA{30, 30, 35, 255})   // dark outer ring
	drawCircle(img, cx, cy, 360, color.RGBA{80, 170, 255, 255}) // blue inner circle
	drawCircle(img, cx, cy, 200, color.RGBA{30, 30, 35, 255})   // dark center
	drawCircle(img, cx, cy, 130, color.RGBA{80, 170, 255, 255}) // blue dot

	f, err := os.Create("Icon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func fill(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawCircle(img *image.RGBA, cx, cy int, r float64, c color.RGBA) {
	r2 := r * r
	b := img.Bounds()
	minX := max(b.Min.X, int(math.Floor(float64(cx)-r)))
	maxX := min(b.Max.X-1, int(math.Ceil(float64(cx)+r)))
	minY := max(b.Min.Y, int(math.Floor(float64(cy)-r)))
	maxY := min(b.Max.Y-1, int(math.Ceil(float64(cy)+r)))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x-cx) + 0.5
			dy := float64(y-cy) + 0.5
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
