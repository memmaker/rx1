package util

import (
	"image/color"
)

func LerpColorRGBA(start, end color.RGBA, percent float64) color.RGBA {
	r1, g1, b1, a1 := start.R, start.G, start.B, start.A
	r2, g2, b2, a2 := end.R, end.G, end.B, end.A
	return color.RGBA{
		R: uint8(float64(r1) + float64(r2-r1)*percent),
		G: uint8(float64(g1) + float64(g2-g1)*percent),
		B: uint8(float64(b1) + float64(b2-b1)*percent),
		A: uint8(float64(a1) + float64(a2-a1)*percent),
	}
}

func SetBrightness(start color.RGBA, percent float64) color.RGBA {
	r1, g1, b1 := start.R, start.G, start.B
	return color.RGBA{
		R: uint8(float64(r1) * percent),
		G: uint8(float64(g1) * percent),
		B: uint8(float64(b1) * percent),
		A: uint8(255),
	}
}
