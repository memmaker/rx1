package dungen

import (
	"math/rand"
	"rx1/geometry"
)

// RogueGenerator is a port of Rogue 5.4's rooms.c (do_rooms, do_maze) and
// passages.c (do_passages, conn, door, putpass). Only the map size is variable.
// Populating the level (gold, monsters, items, traps) happens in game.spawnEntities.
type RogueGenerator struct {
	mapWidth, mapHeight int
	level               int
	random              *rand.Rand
}

// rogueRoom mirrors Rogue's struct room: pos/max include the walls.
type rogueRoom struct {
	pos, max geometry.Point
	gone     bool
	dark     bool
	maze     bool
	room     *DungeonRoom
}

func NewRogueGenerator(random *rand.Rand, mapCols, mapRows, level int) *RogueGenerator {
	return &RogueGenerator{random: random, mapWidth: mapCols, mapHeight: mapRows, level: level}
}

// rnd is Rogue's rnd(): 0..n-1, and 0 for n <= 0.
func (r *RogueGenerator) rnd(n int) int {
	if n <= 0 {
		return 0
	}
	return r.random.Intn(n)
}

func (r *RogueGenerator) Generate() *DungeonMap {
	m := NewDungeonMap(r.mapWidth, r.mapHeight)
	rooms := r.doRooms(m)
	r.doPassages(m, rooms)
	r.placeStairs(m)
	return m
}

func (r *RogueGenerator) rndRoom(rooms []rogueRoom) int {
	for {
		rm := r.rnd(len(rooms))
		if !rooms[rm].gone {
			return rm
		}
	}
}

func (r *RogueGenerator) doRooms(m *DungeonMap) []rogueRoom {
	// Rogue: bsze = NUMCOLS/3, NUMLINES/3 on an 80x24 screen. (w-1)/3 keeps the
	// rightmost wall inside the map for any width and equals 26 for 80 columns.
	bsze := geometry.Point{X: (r.mapWidth - 1) / 3, Y: r.mapHeight / 3}
	rooms := make([]rogueRoom, 9)

	leftOut := r.rnd(4)
	for i := 0; i < leftOut; i++ {
		rooms[r.rndRoom(rooms)].gone = true
	}

	for i := range rooms {
		rp := &rooms[i]
		top := geometry.Point{X: (i%3)*bsze.X + 1, Y: (i / 3) * bsze.Y}
		if rp.gone {
			for {
				rp.pos.X = top.X + r.rnd(bsze.X-2) + 1
				rp.pos.Y = top.Y + r.rnd(bsze.Y-2) + 1
				if rp.pos.Y > 0 && rp.pos.Y < r.mapHeight-1 {
					break
				}
			}
			continue
		}
		if r.rnd(10) < r.level-1 {
			rp.dark = true
			if r.rnd(15) == 0 {
				rp.maze = true
			}
		}
		if rp.maze {
			rp.max = geometry.Point{X: bsze.X - 1, Y: bsze.Y - 1}
			rp.pos = top
			if rp.pos.X == 1 {
				rp.pos.X = 0
			}
			if rp.pos.Y == 0 {
				rp.pos.Y++
				rp.max.Y--
			}
			r.doMaze(m, rp)
		} else {
			for {
				rp.max.X = r.rnd(bsze.X-4) + 4
				rp.max.Y = r.rnd(bsze.Y-4) + 4
				rp.pos.X = top.X + r.rnd(bsze.X-rp.max.X)
				rp.pos.Y = top.Y + r.rnd(bsze.Y-rp.max.Y)
				if rp.pos.Y != 0 {
					break
				}
			}
			floor := geometry.NewRect(rp.pos.X+1, rp.pos.Y+1, rp.pos.X+rp.max.X-1, rp.pos.Y+rp.max.Y-1)
			rp.room = NewDungeonRoomFromRect(floor)
			m.AddRoomAndSetTiles(rp.room)
		}
		rp.room.SetLit(!rp.dark)
	}
	return rooms
}

// doMaze digs a maze of passages filling the room's cell (rooms.c do_maze/dig).
func (r *RogueGenerator) doMaze(m *DungeonMap, rp *rogueRoom) {
	passages := make(map[geometry.Point]bool)
	put := func(p geometry.Point) {
		if m.Contains(p) {
			passages[p] = true
			m.SetCorridor(p.X, p.Y)
		}
	}
	maxY, maxX := rp.max.Y, rp.max.X
	startY, startX := rp.pos.Y, rp.pos.X
	isPass := func(y, x int) bool { return passages[geometry.Point{X: x + startX, Y: y + startY}] }

	var dig func(y, x int)
	dig = func(y, x int) {
		deltas := []geometry.Point{{X: 0, Y: 2}, {X: 0, Y: -2}, {X: 2, Y: 0}, {X: -2, Y: 0}}
		for {
			cnt := 0
			nextY, nextX := 0, 0
			for _, d := range deltas {
				newY, newX := y+d.Y, x+d.X
				if newY < 0 || newY > maxY || newX < 0 || newX > maxX {
					continue
				}
				if isPass(newY, newX) {
					continue
				}
				cnt++
				if r.rnd(cnt) == 0 {
					nextY, nextX = newY, newX
				}
			}
			if cnt == 0 {
				return
			}
			var between geometry.Point
			if nextY == y {
				between.Y = y + startY
				if nextX-x < 0 {
					between.X = nextX + startX + 1
				} else {
					between.X = nextX + startX - 1
				}
			} else {
				between.X = x + startX
				if nextY-y < 0 {
					between.Y = nextY + startY + 1
				} else {
					between.Y = nextY + startY - 1
				}
			}
			put(between)
			put(geometry.Point{X: nextX + startX, Y: nextY + startY})
			dig(nextY, nextX)
		}
	}
	startCellY := (r.rnd(maxY) / 2) * 2
	startCellX := (r.rnd(maxX) / 2) * 2
	put(geometry.Point{X: startCellX + startX, Y: startCellY + startY})
	dig(startCellY, startCellX)

	tiles := make([]geometry.Point, 0, len(passages))
	for p := range passages {
		tiles = append(tiles, p)
	}
	rp.room = NewDungeonRoomFromTiles(geometry.NewRect(rp.pos.X, rp.pos.Y, rp.pos.X+rp.max.X+1, rp.pos.Y+rp.max.Y+1), tiles)
	m.rooms = append(m.rooms, rp.room)
}

// isConnectable is Rogue's rdes[].conn table: rooms next to each other in the 3x3 grid.
func isConnectable(a, b int) bool {
	ax, ay, bx, by := a%3, a/3, b%3, b/3
	return (ay == by && (ax-bx == 1 || bx-ax == 1)) || (ax == bx && (ay-by == 1 || by-ay == 1))
}

func (r *RogueGenerator) doPassages(m *DungeonMap, rooms []rogueRoom) {
	const maxRooms = 9
	var isConn [maxRooms][maxRooms]bool
	var inGraph [maxRooms]bool

	roomCount := 1
	r1 := r.rnd(maxRooms)
	inGraph[r1] = true
	for roomCount < maxRooms {
		j, r2 := 0, -1
		for i := 0; i < maxRooms; i++ {
			if isConnectable(r1, i) && !inGraph[i] {
				j++
				if r.rnd(j) == 0 {
					r2 = i
				}
			}
		}
		if j == 0 {
			for {
				r1 = r.rnd(maxRooms)
				if inGraph[r1] {
					break
				}
			}
			continue
		}
		inGraph[r2] = true
		r.conn(m, rooms, r1, r2)
		isConn[r1][r2], isConn[r2][r1] = true, true
		roomCount++
	}

	for extra := r.rnd(5); extra > 0; extra-- {
		r1 = r.rnd(maxRooms)
		j, r2 := 0, -1
		for i := 0; i < maxRooms; i++ {
			if isConnectable(r1, i) && !isConn[r1][i] {
				j++
				if r.rnd(j) == 0 {
					r2 = i
				}
			}
		}
		if j != 0 {
			r.conn(m, rooms, r1, r2)
			isConn[r1][r2], isConn[r2][r1] = true, true
		}
	}
}

// conn draws a corridor between two neighbouring grid rooms (passages.c conn).
func (r *RogueGenerator) conn(m *DungeonMap, rooms []rogueRoom, r1, r2 int) {
	rm := min(r1, r2)
	down := max(r1, r2) != rm+1
	rpf := &rooms[rm]
	var rpt *rogueRoom
	var del, turnDelta, spos, epos geometry.Point
	var distance, turnDistance int

	// Maze rooms have no fixed wall line; retry until the spot is a maze passage (Rogue loops forever).
	pick := func(rp *rogueRoom, f func() geometry.Point) geometry.Point {
		p := f()
		for tries := 0; rp.maze && !m.IsCorridor(p) && tries < 1000; tries++ {
			p = f()
		}
		return p
	}

	if down {
		rpt = &rooms[rm+3]
		del = geometry.Point{Y: 1}
		spos, epos = rpf.pos, rpt.pos
		if !rpf.gone {
			spos = pick(rpf, func() geometry.Point {
				return geometry.Point{X: rpf.pos.X + r.rnd(rpf.max.X-2) + 1, Y: rpf.pos.Y + rpf.max.Y - 1}
			})
		}
		if !rpt.gone {
			epos = pick(rpt, func() geometry.Point {
				return geometry.Point{X: rpt.pos.X + r.rnd(rpt.max.X-2) + 1, Y: rpt.pos.Y}
			})
		}
		distance = abs(spos.Y-epos.Y) - 1
		turnDelta = geometry.Point{X: sign(epos.X - spos.X)}
		turnDistance = abs(spos.X - epos.X)
	} else {
		rpt = &rooms[rm+1]
		del = geometry.Point{X: 1}
		spos, epos = rpf.pos, rpt.pos
		if !rpf.gone {
			spos = pick(rpf, func() geometry.Point {
				return geometry.Point{X: rpf.pos.X + rpf.max.X - 1, Y: rpf.pos.Y + r.rnd(rpf.max.Y-2) + 1}
			})
		}
		if !rpt.gone {
			epos = pick(rpt, func() geometry.Point {
				return geometry.Point{X: rpt.pos.X, Y: rpt.pos.Y + r.rnd(rpt.max.Y-2) + 1}
			})
		}
		distance = abs(spos.X-epos.X) - 1
		turnDelta = geometry.Point{Y: sign(epos.Y - spos.Y)}
		turnDistance = abs(spos.Y - epos.Y)
	}

	turnSpot := r.rnd(distance-1) + 1

	if !rpf.gone {
		r.door(m, rpf, spos)
	} else {
		r.putPass(m, spos)
	}
	if !rpt.gone {
		r.door(m, rpt, epos)
	} else {
		r.putPass(m, epos)
	}

	curr := spos
	for distance > 0 {
		curr = curr.Add(del)
		if distance == turnSpot {
			for ; turnDistance > 0; turnDistance-- {
				r.putPass(m, curr)
				curr = curr.Add(turnDelta)
			}
		}
		r.putPass(m, curr)
		distance--
	}
}

// putPass adds a passage, sometimes secret (passages.c putpass).
func (r *RogueGenerator) putPass(m *DungeonMap, p geometry.Point) {
	if !m.Contains(p) {
		return
	}
	if m.GetTileAt(p) == Room || m.GetTileAt(p) == Door {
		return // don't turn a room into corridor where a passage grazes it
	}
	m.SetCorridor(p.X, p.Y)
	if r.rnd(10)+1 < r.level && r.rnd(40) == 0 {
		m.secretPassages[p] = true
	}
}

// door adds a door, sometimes secret (passages.c door). Maze rooms get no door.
func (r *RogueGenerator) door(m *DungeonMap, rp *rogueRoom, p geometry.Point) {
	if rp.maze {
		return
	}
	m.SetDoor(p.X, p.Y)
	if r.rnd(10)+1 < r.level && r.rnd(5) == 0 {
		m.secretDoors[p] = true // the room keeps its wall here until found
		return
	}
	rp.room.SetDoor(p)
}

// placeStairs puts both staircases on random floor spots (new_level.c find_floor).
// Rogue has only the down stairs; rx1 adds an up staircase the same way.
func (r *RogueGenerator) placeStairs(m *DungeonMap) {
	down := r.findFloor(m)
	m.SetStairsDown(down)
	up := r.findFloor(m)
	for up == down {
		up = r.findFloor(m)
	}
	m.SetStairsUp(up)
}

func (r *RogueGenerator) findFloor(m *DungeonMap) geometry.Point {
	for {
		room := m.rooms[r.rnd(len(m.rooms))]
		p := room.GetRandomAbsoluteFloorPosition(r.random)
		if t := m.GetTileAt(p); t == Room || t == Corridor {
			return p
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	if v < 0 {
		return -1
	}
	return 1
}
