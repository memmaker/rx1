package dungen

import (
	"math/rand"
	"rx1/geometry"
)

// NetHackMazeGenerator is a port of NetHack 5.0's mkmaze.c create_maze: a randomized
// depth-first maze on a grid of cells two tiles apart, optionally without dead ends, scaled
// up so that passages are CorridorWidth wide and walls WallThickness thick.
//
// Like NetHack's makemaz, NewNetHackMazeGenerator picks one of two kinds at random: the plain
// 1/1 maze, or a maze of random widths of which one in five has its dead ends removed.
// Set the fields to choose; -1 rolls NetHack's random width/thickness.
//
// The maze is split into 3x3 blocks, each block a wall-less room like Rogue's maze rooms, so
// that "in the same room" stays local. Not ported: wallification, the maze special levels.
type NetHackMazeGenerator struct {
	random         *rand.Rand
	width, height  int
	CorridorWidth  int // 1..5, -1 for random
	WallThickness  int // 1..5, -1 for random
	RemoveDeadEnds bool
}

func NewNetHackMazeGenerator(random *rand.Rand, mapCols, mapRows int) *NetHackMazeGenerator {
	g := &NetHackMazeGenerator{random: random, width: mapCols, height: mapRows, CorridorWidth: 1, WallThickness: 1}
	if rn2(random, 2) == 0 {
		g.CorridorWidth, g.WallThickness, g.RemoveDeadEnds = -1, -1, rn2(random, 5) == 0
	}
	return g
}

func (g *NetHackMazeGenerator) Generate() *DungeonMap {
	w, h := g.width, g.height
	xmax, ymax := (w-1)&^1, (h-1)&^1 // x_maze_max, y_maze_max
	corrwid, wallthick := g.CorridorWidth, g.WallThickness
	if corrwid == -1 {
		corrwid = 1 + rn2(g.random, 4)
	}
	if wallthick == -1 {
		wallthick = 1 + rn2(g.random, 4) - corrwid
	}
	wallthick = min(max(wallthick, 1), 5)
	corrwid = min(max(corrwid, 1), 5)
	scale := corrwid + wallthick
	rdx, rdy := xmax/scale, ymax/scale
	mx, my := rdx*2, rdy*2 // the bounds of the unscaled maze

	open := make([]bool, w*h)
	okay := func(x, y int, dir geometry.Point) bool {
		x, y = x+2*dir.X, y+2*dir.Y
		return x >= 3 && y >= 3 && x <= mx && y <= my && !open[x+y*w]
	}
	start := geometry.Point{X: 3 + 2*rn2(g.random, mx>>1-1), Y: 3 + 2*rn2(g.random, my>>1-1)}
	open[start.X+start.Y*w] = true
	for stack := []geometry.Point{start}; len(stack) > 0; {
		cell := stack[len(stack)-1]
		var dirs []geometry.Point
		for _, d := range cardinalDirections {
			if okay(cell.X, cell.Y, d) {
				dirs = append(dirs, d)
			}
		}
		if len(dirs) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		d := dirs[g.random.Intn(len(dirs))]
		open[cell.X+d.X+(cell.Y+d.Y)*w] = true
		next := geometry.Point{X: cell.X + 2*d.X, Y: cell.Y + 2*d.Y}
		open[next.X+next.Y*w] = true
		stack = append(stack, next)
	}

	if g.RemoveDeadEnds {
		inBounds := func(x, y int) bool { return x >= 2 && y >= 2 && x < mx && y < my }
		for x := 2; x < mx; x++ {
			for y := 2; y < my; y++ {
				if !open[x+y*w] || x%2 == 0 || y%2 == 0 {
					continue
				}
				var walled []geometry.Point // walls to a cell that is open
				blocked := 0
				for _, d := range cardinalDirections {
					if !inBounds(x+d.X, y+d.Y) || !inBounds(x+2*d.X, y+2*d.Y) {
						blocked++
					} else if !open[x+d.X+(y+d.Y)*w] && open[x+2*d.X+(y+2*d.Y)*w] {
						walled = append(walled, d)
						blocked++
					}
				}
				if blocked >= 3 && len(walled) > 0 {
					d := walled[g.random.Intn(len(walled))]
					open[x+d.X+(y+d.Y)*w] = true
				}
			}
		}
	}

	if scale > 2 {
		small := slicesClone(open)
		clear(open)
		rx := 2
		for x := 2; rx < xmax; x++ {
			sx := corrwid
			if x%2 == 0 {
				sx = wallthick
				if x == 2 || x == rdx*2 {
					sx = 1
				}
			}
			ry := 2
			for y := 2; ry < ymax; y++ {
				sy := corrwid
				if y%2 == 0 {
					sy = wallthick
					if y == 2 || y == rdy*2 {
						sy = 1
					}
				}
				for dx := 0; dx < sx && rx+dx < xmax; dx++ {
					for dy := 0; dy < sy && ry+dy < ymax; dy++ {
						open[rx+dx+(ry+dy)*w] = x < w && y < h && small[x+y*w]
					}
				}
				ry += sy
			}
			rx += sx
		}
	}

	m := NewDungeonMap(w, h)
	blocks := make(map[int][]geometry.Point)
	bw, bh := (w-1)/3, h/3
	for i, o := range open {
		if o {
			x, y := i%w, i/w
			m.SetCorridor(x, y)
			blocks[min(x/bw, 2)+3*min(y/bh, 2)] = append(blocks[min(x/bw, 2)+3*min(y/bh, 2)], geometry.Point{X: x, Y: y})
		}
	}
	var tiles []geometry.Point
	for b := 0; b < 9; b++ {
		if len(blocks[b]) == 0 {
			continue
		}
		bx, by := b%3*bw, b/3*bh
		m.rooms = append(m.rooms, NewDungeonRoomFromTiles(geometry.NewRect(bx, by, bx+bw, by+bh), blocks[b]))
		tiles = append(tiles, blocks[b]...)
	}
	down := tiles[g.random.Intn(len(tiles))]
	up := tiles[g.random.Intn(len(tiles))]
	for up == down {
		up = tiles[g.random.Intn(len(tiles))]
	}
	m.SetStairsDown(down)
	m.SetStairsUp(up)
	return m
}
