package dungen

import (
	"rx1/geometry"
	"strings"
)

// The themed rooms of NetHack 5.0's dat/themerms.lua that need nothing but walls and floor:
// the shaped maps (L, T, Z, cross, clover, circles, blocked center), pillars, and rooms inside
// rooms (Fake Delphi, room in a room, huge room with another room inside, nesting rooms).
// NetHack lists each rotation by hand; here every map is turned and mirrored at random.
// Not ported: the fills (ice, traps, garden, statuary, ...), the mausoleum (a sealed closet),
// the water-surrounded vault and the twin shops.
var nhShapes = []string{`
-----xxx
|...|xxx
|...|xxx
|...----
|......|
|......|
--------`, `
xxx-----xxx
xxx|...|xxx
xxx|...|xxx
----...----
|.........|
|.........|
|.........|
-----------`, `
xxx-----
xxx|...|
xxx|...|
----...|
|......|
|......|
|......|
|...----
|...|xxx
|...|xxx
-----xxx`, `
xxx-----xxx
xxx|...|xxx
xxx|...|xxx
----...----
|.........|
|.........|
|.........|
----...----
xxx|...|xxx
xxx|...|xxx
xxx-----xxx`, `
-----x-----
|...|x|...|
|...---...|
|.........|
---.....---
xx|.....|xx
---.....---
|.........|
|...---...|
|...|x|...|
-----x-----`, `
xx---xx
x--.--x
--...--
|.....|
--...--
x--.--x
xx---xx`, `
xx-----xx
x--...--x
--.....--
|.......|
|.......|
|.......|
--.....--
x--...--x
xx-----xx`, `
xxx-----xxx
x---...---x
x-.......-x
--.......--
|.........|
|.........|
|.........|
--.......--
x-.......-x
x---...---x
xxx-----xxx`, `
-----------
|.........|
|.........|
|.........|
|...---...|
|...---...|
|...---...|
|.........|
|.........|
|.........|
-----------`}

// createThemedRoom makes one of the themed rooms, or reports false when there was no space for it.
func (g *NetHackGenerator) createThemedRoom() bool {
	switch kind := g.rn2(len(nhShapes) + 5); {
	case kind < len(nhShapes):
		return g.createShapedRoom(nhShapes[kind])
	case kind == len(nhShapes):
		return g.createPillarRoom()
	default:
		return g.createNestedRoom(kind - len(nhShapes) - 1)
	}
}

// lit is NetHack's litstate_rnd: rooms get darker with depth.
func (g *NetHackGenerator) lit() bool { return g.rn2(1+g.level)+1 < 11 && g.rn2(77) != 0 }

// createShapedRoom is a des.map room: the map is claimed as it is, the 'x' stays rock.
func (g *NetHackGenerator) createShapedRoom(shape string) bool {
	rows := strings.Split(strings.TrimSpace(shape), "\n")
	grid := make([][]byte, len(rows))
	for y, row := range rows {
		grid[y] = []byte(row)
	}
	for turns := g.rn2(4); turns > 0; turns-- {
		turned := make([][]byte, len(grid[0]))
		for x := range turned {
			turned[x] = make([]byte, len(grid))
			for y := range grid {
				turned[x][len(grid)-1-y] = grid[y][x]
			}
		}
		grid = turned
	}
	if g.rn2(2) == 0 {
		for _, row := range grid {
			for i, j := 0, len(row)-1; i < j; i, j = i+1, j-1 {
				row[i], row[j] = row[j], row[i]
			}
		}
	}
	h, w := len(grid), len(grid[0])
	xabs, yabs, _, _, r1, ok := g.pickSpot(w-3, h-3, true)
	if !ok {
		return false
	}
	x0, y0 := xabs-1, yabs-1
	g.splitRects(r1, nhRect{x0, y0, x0 + w - 1, y0 + h - 1})

	var floor, walls []geometry.Point
	for y, row := range grid {
		for x, c := range row {
			p := geometry.Point{X: x0 + x, Y: y0 + y}
			switch c {
			case '.':
				floor = append(floor, p)
			case '-', '|':
				walls = append(walls, p)
			}
		}
	}
	g.addIrregularRoom(floor, walls)
	return true
}

// addIrregularRoom makes a room of loose floor and wall tiles, e.g. an L; its bounds are those of the floor.
func (g *NetHackGenerator) addIrregularRoom(floor, walls []geometry.Point) *nhRoom {
	b := geometry.NewRect(floor[0].X, floor[0].Y, floor[0].X+1, floor[0].Y+1)
	room := &DungeonRoom{floorTiles: make(map[geometry.Point]bool), wallTiles: walls, doors: make(map[geometry.Point]bool)}
	for _, p := range floor {
		room.floorTiles[p] = true
		b = geometry.NewRect(min(b.Min.X, p.X), min(b.Min.Y, p.Y), max(b.Max.X, p.X+1), max(b.Max.Y, p.Y+1))
	}
	room.bounds = b
	room.SetLit(g.lit())
	g.m.AddRoomAndSetTiles(room)
	for _, ps := range [][]geometry.Point{walls, floor} {
		for _, p := range ps {
			g.claimed[p.X+p.Y*g.width] = true
		}
	}
	nr := &nhRoom{lx: b.Min.X, ly: b.Min.Y, hx: b.Max.X - 1, hy: b.Max.Y - 1, irregular: true, room: room}
	g.rooms = append(g.rooms, nr)
	return nr
}

// createPillarRoom is the 'Pillars' room: 10x10 with 2x2 blocks of wall.
func (g *NetHackGenerator) createPillarRoom() bool {
	xabs, yabs, dx, dy, r1, ok := g.pickSpot(9, 9, true)
	if !ok {
		return false
	}
	g.splitRects(r1, nhRect{xabs - 1, yabs - 1, xabs + dx + 1, yabs + dy + 1})
	g.addRoom(xabs, yabs, xabs+dx, yabs+dy)
	room := g.rooms[len(g.rooms)-1]
	for _, px := range []int{2, 6} {
		for _, py := range []int{2, 6} {
			for _, o := range []geometry.Point{{}, {X: 1}, {Y: 1}, {X: 1, Y: 1}} {
				g.wallOff(room, geometry.Point{X: xabs + px + o.X, Y: yabs + py + o.Y})
			}
		}
	}
	room.irregular = true
	return true
}

// wallOff turns a floor tile of the room into wall.
func (g *NetHackGenerator) wallOff(r *nhRoom, p geometry.Point) {
	delete(r.room.floorTiles, p)
	r.room.wallTiles = append(r.room.wallTiles, p)
	g.m.SetWall(p.X, p.Y)
}

// createNestedRoom makes a big room with a room inside it, which has a door of its own. kind 0 is the
// 11x9 Fake Delphi, 1 'Room in a room', 2 'Huge room with another room inside', 3 'Nesting rooms'.
func (g *NetHackGenerator) createNestedRoom(kind int) bool {
	w, h := 11, 9
	switch kind {
	case 1:
		w, h = 6+g.rn2(8), 5+g.rn2(3)
	case 2:
		w, h = 11+g.rn2(10), 8+g.rn2(5)
	case 3:
		w, h = 9+g.rn2(4), 9+g.rn2(4)
	}
	xabs, yabs, dx, dy, r1, ok := g.pickSpot(w-1, h-1, true)
	if !ok {
		return false
	}
	g.splitRects(r1, nhRect{xabs - 1, yabs - 1, xabs + dx + 1, yabs + dy + 1})
	g.addRoom(xabs, yabs, xabs+dx, yabs+dy)
	outer := g.rooms[len(g.rooms)-1]
	outer.irregular = true // the room inside takes floor away from it

	inner := nhRect{xabs + 4, yabs + 3, xabs + 6, yabs + 5} // Fake Delphi, floor of the inner room
	if kind != 0 {
		iw := 1 + g.rn2(w-4)
		ih := 1 + g.rn2(h-4)
		if kind == 3 {
			iw, ih = w/2+g.rn2(w-1-w/2), h/2+g.rn2(h-1-h/2)
		}
		if iw > w-4 || ih > h-4 {
			iw, ih = min(iw, w-4), min(ih, h-4)
		}
		ix, iy := 2+g.rn2(w-iw-3), 2+g.rn2(h-ih-3)
		inner = nhRect{xabs + ix, yabs + iy, xabs + ix + iw - 1, yabs + iy + ih - 1}
	}
	if kind == 2 && g.rn2(10) == 0 {
		return true // 10% of the huge rooms are empty
	}
	g.nestRoom(outer, inner, kind == 3)
	return true
}

// nestRoom puts a room with the given floor into the outer room, with a door or two on the way to it.
func (g *NetHackGenerator) nestRoom(outer *nhRoom, floor nhRect, nestAgain bool) {
	for x := floor.lx - 1; x <= floor.hx+1; x++ {
		for y := floor.ly - 1; y <= floor.hy+1; y++ {
			delete(outer.room.floorTiles, geometry.Point{X: x, Y: y})
			g.m.SetWall(x, y)
		}
	}
	room := NewDungeonRoomFromRect(geometry.NewRect(floor.lx, floor.ly, floor.hx+1, floor.hy+1))
	room.SetLit(outer.room.IsLit())
	g.m.AddRoomAndSetTiles(room)
	inner := &nhRoom{lx: floor.lx, ly: floor.ly, hx: floor.hx, hy: floor.hy, room: room}

	if nestAgain {
		w, h := floor.hx-floor.lx+1, floor.hy-floor.ly+1
		if w >= 5 && h >= 5 {
			iw, ih := 1+g.rn2(w-4), 1+g.rn2(h-4)
			ix, iy := floor.lx+2+g.rn2(w-iw-3), floor.ly+2+g.rn2(h-ih-3)
			g.nestRoom(inner, nhRect{ix, iy, ix + iw - 1, iy + ih - 1}, false)
		}
	}
	for doors := 1 + b2i(g.rn2(100) < 15); doors > 0; doors-- {
		for try := 0; try < 20; try++ {
			side := g.rn2(4)
			if p, ok := g.finddpos(side, inner); ok && outer.room.FloorContains(p.Sub(inward[side])) {
				g.dodoor(p, inner)
				break
			}
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
