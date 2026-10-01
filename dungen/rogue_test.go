package dungen

import (
	"math/rand"
	"rx1/geometry"
	"testing"
)

// Every walkable tile must be reachable from the down stairs (secret tiles count as found).
func TestRogueGeneratorConnected(t *testing.T) {
	for seed := int64(0); seed < 300; seed++ {
		level := int(seed%26) + 1
		m := NewRogueGenerator(rand.New(rand.NewSource(seed)), 80, 23, level).Generate()
		var start geometry.Point
		walkable := 0
		for y := 0; y < m.height; y++ {
			for x := 0; x < m.width; x++ {
				switch m.GetTile(x, y) {
				case Wall:
				case StairsDown:
					start = geometry.Point{X: x, Y: y}
					walkable++
				default:
					walkable++
				}
			}
		}
		seen := map[geometry.Point]bool{start: true}
		queue := []geometry.Point{start}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, d := range []geometry.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
				n := p.Add(d)
				if m.Contains(n) && !seen[n] && m.GetTileAt(n) != Wall {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		if len(seen) != walkable {
			t.Errorf("seed %d level %d: %d of %d tiles reachable", seed, level, len(seen), walkable)
			m.Print()
		}
	}
}
