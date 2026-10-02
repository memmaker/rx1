package foundation

import (
	"image/color"
	"rx1/util"
)

// LightInfo describes the player's light for the UI
type LightInfo struct {
	Radius int // 0 = no working light
	Color  color.RGBA
	// Pattern is "fire", "smooth" or "" (steady); DelayMs is the time per flicker frame
	Pattern string
	DelayMs int
}

var firePattern = []float64{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 0.85, 0.92, 0.9, 0.90, 0.92, 0.96, 0.97, 0.99, 1.0, 1.0, 1.0, 0.9, 1.0, 1.0, 1.0, 1.0, 0.9, 0.93, 0.98}

// smooth: 40 steps 1.0 -> 0.7 -> 1.0
var smoothPattern = func() []float64 {
	p := make([]float64, 40)
	for i := range p {
		p[i] = 0.7 + 0.3*float64(abs(i-20))/20
	}
	return p
}()

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// LightReaches says if a light with radius r reaches a tile at the squared distance.
// A light of radius 1 also reaches the diagonal neighbours, weakly (see LightFalloff).
func LightReaches(distSquared, r int) bool {
	return distSquared <= r*r || r == 1 && distSquared == 2
}

// LightFalloff is the brightness at distance d of a light with radius r
func LightFalloff(d, r float64) float64 {
	if r == 1 && d > 1 {
		return 0.3
	}
	return min(max(1-util.EaseInExpo(d/(r+1)), 0.16), 1.0)
}

// LightFlicker is the brightness factor of the whole lit area at time nowMs.
func (l LightInfo) LightFlicker(nowMs int64) float64 {
	pattern := firePattern
	switch l.Pattern {
	case "smooth":
		pattern = smoothPattern
	case "fire":
	default:
		return 1.0
	}
	if l.DelayMs <= 0 {
		return 1.0
	}
	return pattern[int(nowMs/int64(l.DelayMs))%len(pattern)]
}
