package dungen

import (
	"math/rand"
	"rx1/geometry"
)

// LavaFromLevel is the first level with lava lakes and rivers, on cave and Brogue levels only.
const LavaFromLevel = 13

func withLava(random *rand.Rand, m *DungeonMap, level int) *DungeonMap {
	if level >= LavaFromLevel {
		addLava(random, m)
	}
	return m
}

// addLava pours a few lava lakes and maybe a river over the ground of the level. A tile only turns
// to lava when every walkable tile can still be reached from every other one, so the stairs stay
// connected; the tiles left out become fords and islands.
func addLava(random *rand.Rand, m *DungeonMap) {
	var cells []geometry.Point
	for lakes := 1 + random.Intn(3); lakes > 0; lakes-- {
		center := geometry.Point{X: random.Intn(m.width), Y: random.Intn(m.height)}
		radius := 2 + random.Intn(3)
		for y := -radius; y <= radius; y++ {
			for x := -radius; x <= radius; x++ {
				if x*x+y*y <= radius*radius-random.Intn(radius+1) { // a ragged shore
					cells = append(cells, center.Add(geometry.Point{X: x, Y: y}))
				}
			}
		}
	}
	if random.Intn(2) == 0 { // a river from the west edge to the east, wandering up and down
		for x, y := 0, random.Intn(m.height); x < m.width; x++ {
			y = min(max(y+random.Intn(3)-1, 1), m.height-2)
			cells = append(cells, geometry.Point{X: x, Y: y}, geometry.Point{X: x, Y: y + 1})
		}
	}
	unreachable := m.unreachableTiles() // a Brogue lake may already wall off some chasm
	for _, p := range cells {
		if !m.Contains(p) || (m.GetTileAt(p) != Room && m.GetTileAt(p) != Corridor) {
			continue
		}
		room := m.GetRoomAt(p)
		if room != nil && len(room.floorTiles) == 1 {
			continue // every room keeps some ground
		}
		old := m.GetTileAt(p)
		m.tiles[p.X+p.Y*m.width] = Lava
		if m.unreachableTiles() != unreachable {
			m.tiles[p.X+p.Y*m.width] = old
			continue
		}
		if room != nil {
			delete(room.floorTiles, p)
		}
	}
}

// unreachableTiles counts the walkable tiles that cannot be reached from the down stairs with cardinal moves.
// ponytail: a flood fill per lava tile, O(lava * map), fine for 80x23; union-find if maps grow.
func (m *DungeonMap) unreachableTiles() int {
	var start geometry.Point
	walkable := 0
	for i, t := range m.tiles {
		if t == StairsDown {
			start = geometry.Point{X: i % m.width, Y: i / m.width}
		}
		if t.IsWalkable() {
			walkable++
		}
	}
	seen := map[geometry.Point]bool{start: true}
	for queue := []geometry.Point{start}; len(queue) > 0; queue = queue[1:] {
		for _, d := range cardinalDirections {
			if q := queue[0].Add(d); m.Contains(q) && !seen[q] && m.GetTileAt(q).IsWalkable() {
				seen[q] = true
				queue = append(queue, q)
			}
		}
	}
	return walkable - len(seen)
}
