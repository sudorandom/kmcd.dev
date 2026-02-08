package main

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func BenchmarkAddVignette(b *testing.B) {
	// Setup a dummy image similar to production size (1200x630)
	// Use NRGBA because production code uses imaging.Blur which returns NRGBA.
	width, height := 1200, 630
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	// Fill with some color
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{100, 150, 200, 255}}, image.Point{}, draw.Src)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addVignette(img)
	}
}
