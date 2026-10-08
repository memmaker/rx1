package game

import (
	"rx1/geometry"
	"testing"
)

// On an open grid the 1.4 diagonal cost gives the cheapest mix: 3 diagonals and 7 straight steps for (10,3).
func TestAstarPathPrefersCheapestMix(t *testing.T) {
	open := func(p geometry.Point) []geometry.Point {
		var out []geometry.Point
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if q := (geometry.Point{X: p.X + dx, Y: p.Y + dy}); q != p && q.X >= 0 && q.Y >= 0 && q.X < 20 && q.Y < 20 {
					out = append(out, q)
				}
			}
		}
		return out
	}
	start, far, near := geometry.Point{X: 0, Y: 0}, geometry.Point{X: 19, Y: 19}, geometry.Point{X: 10, Y: 3}
	path := astarPath(start, []geometry.Point{far, near}, open)
	if len(path) != 11 || path[0] != start || path[10] != near {
		t.Fatalf("want 11-point path to the nearer goal, got %v", path)
	}
	total := 0
	for i := 1; i < len(path); i++ {
		total += stepCost(path[i-1], path[i])
	}
	if total != 3*14+7*10 {
		t.Fatalf("path cost %d, want %d", total, 3*14+7*10)
	}
}
