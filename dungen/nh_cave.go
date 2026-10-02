package dungen

import (
	"math/rand"
	"rx1/geometry"
)

// NetHackCaveGenerator is a port of NetHack 5.0's mkmap.c as used by the Gnomish Mines filler
// (smoothed, joined): 40% of the map is filled at random, then cellular automata turn the noise
// into caves, tiny pockets are dropped, and the remaining pockets are joined by tunnels.
//
//  1. Pass one: a tile with at most 2 neighbours (of 8) dies, one with 5 or more is born.
//  2. Pass two: a tile with exactly 5 neighbours dies.
//  3. Pass three, twice: a tile with fewer than 3 neighbours dies (smoothing).
//
// Each pocket is a wall-less room of its own, as in NetHack; the tunnels belong to no room.
// The rooms are lit all together or not at all. The walls (wallify) are not ported: rock is rock.
type NetHackCaveGenerator struct {
	random        *rand.Rand
	width, height int
	level         int
}

func NewNetHackCaveGenerator(random *rand.Rand, mapCols, mapRows, level int) *NetHackCaveGenerator {
	return &NetHackCaveGenerator{random: random, width: mapCols, height: mapRows, level: level}
}

func (g *NetHackCaveGenerator) Generate() *DungeonMap {
	for {
		if m := g.try(); len(m.rooms) >= 2 {
			return m
		}
	}
}

func (g *NetHackCaveGenerator) try() *DungeonMap {
	w, h := g.width, g.height
	cols, rows := w-2, h-1 // mkmap.c WIDTH and HEIGHT; tiles x 2..cols, y 1..rows-1 are in play
	floor := make([]bool, w*h)
	at := func(x, y int) bool {
		if x <= 0 || y < 0 || x > cols || y >= rows {
			return false
		}
		return floor[x+y*w]
	}
	neighbours := func(x, y int) (n int) {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if (dx != 0 || dy != 0) && at(x+dx, y+dy) {
					n++
				}
			}
		}
		return n
	}
	// pass runs the rule on all tiles; with buffered the changes only show after the pass.
	pass := func(buffered bool, rule func(alive bool, n int) bool) {
		next := slicesClone(floor)
		for x := 2; x <= cols; x++ {
			for y := 1; y < rows; y++ {
				next[x+y*w] = rule(floor[x+y*w], neighbours(x, y))
				if !buffered {
					floor[x+y*w] = next[x+y*w]
				}
			}
		}
		if buffered {
			floor = next
		}
	}

	for count, limit := 0, cols*rows*2/5; count < limit; {
		x, y := 2+rn2(g.random, cols-1), 1+rn2(g.random, rows-1)
		if !floor[x+y*w] {
			floor[x+y*w] = true
			count++
		}
	}
	pass(false, func(alive bool, n int) bool {
		switch {
		case n <= 2:
			return false
		case n >= 5:
			return true
		}
		return alive
	})
	pass(true, func(alive bool, n int) bool { return alive && n != 5 })
	for i := 0; i < 2; i++ {
		pass(true, func(alive bool, n int) bool { return alive && n >= 3 })
	}

	// pockets (4-connected, unlike NetHack's 8: rx1 may forbid diagonal moves); the ones of 3 tiles or less are filled in
	type pocket struct {
		tiles          []geometry.Point
		lx, ly, hx, hy int
	}
	var pockets []*pocket
	seen := make([]bool, w*h)
	for x := 2; x <= cols; x++ {
		for y := 1; y < rows; y++ {
			if !floor[x+y*w] || seen[x+y*w] {
				continue
			}
			p := &pocket{lx: x, ly: y, hx: x, hy: y}
			seen[x+y*w] = true
			queue := []geometry.Point{{X: x, Y: y}}
			for len(queue) > 0 {
				t := queue[0]
				queue = queue[1:]
				p.tiles = append(p.tiles, t)
				p.lx, p.hx, p.ly, p.hy = min(p.lx, t.X), max(p.hx, t.X), min(p.ly, t.Y), max(p.hy, t.Y)
				for _, d := range cardinalDirections {
					if n := t.Add(d); at(n.X, n.Y) && !seen[n.X+n.Y*w] {
						seen[n.X+n.Y*w] = true
						queue = append(queue, n)
					}
				}
			}
			if len(p.tiles) > 3 {
				pockets = append(pockets, p)
				continue
			}
			for _, t := range p.tiles {
				floor[t.X+t.Y*w] = false
			}
		}
	}

	// join_map: dig from a random tile of one pocket to one of the next
	somexy := func(p *pocket) geometry.Point { return p.tiles[g.random.Intn(len(p.tiles))] }
	for croom, i := 0, 1; i < len(pockets); i++ {
		c, c2 := pockets[croom], pockets[i]
		digCorridor(g.random, w, h, somexy(c), somexy(c2), false,
			func(x, y int) bool { return !floor[x+y*w] },
			func(x, y int) bool { return floor[x+y*w] },
			func(x, y int) { floor[x+y*w] = true })
		if c2.lx > c.hx || ((c2.ly > c.hy || c2.hy < c.ly) && g.random.Intn(3) != 0) {
			croom = i
		}
	}

	m := NewDungeonMap(w, h)
	for i, f := range floor {
		if f {
			m.SetRoom(i%w, i/w)
		}
	}
	lit := g.rn2(1+g.level)+1 < 11 && g.rn2(77) != 0
	for _, p := range pockets {
		room := NewDungeonRoomFromTiles(geometry.NewRect(p.lx, p.ly, p.hx+1, p.hy+1), p.tiles)
		room.SetLit(lit)
		m.rooms = append(m.rooms, room)
	}
	if len(m.rooms) >= 2 {
		down := g.rn2(len(m.rooms))
		up := (down + 1 + g.rn2(len(m.rooms)-1)) % len(m.rooms)
		m.SetStairsDown(somexy(pockets[down]))
		m.SetStairsUp(somexy(pockets[up]))
	}
	return m
}

func (g *NetHackCaveGenerator) rn2(n int) int { return rn2(g.random, n) }

func slicesClone(s []bool) []bool { return append([]bool(nil), s...) }
