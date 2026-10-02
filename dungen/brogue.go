package dungen

import (
	"math/rand"
	"rx1/geometry"
	"slices"
)

// BrogueGenerator is a port of digDungeon from Brogue CE's Architect.c:
//
//  1. carveDungeon: a first room (the entrance room on level 1), then up to 35 rooms of random shapes,
//     some behind a hallway, each fitted against a wall of what is dug already (attachRooms).
//  2. addLoops: a wall between two floors more than 20 steps apart by foot becomes a doorway.
//  3. designLakes / fillLakes: blobs of deep water, lava or chasm wherever they do not cut the level
//     in two, with a wreath of shallow water or chasm edge.
//  4. the luminescent fungus and fungus forest autogenerators.
//  5. removeDiagonalOpenings, cleanUpLakeBoundaries, bridges over chasms, finishDoors (secret doors).
//
// The room types, frequencies and depth adjustments are those of Brogue's DP_BASIC profiles; Brogue's
// amulet level (26) is rx1's last level too.
// ponytail: brimstone, machines (vaults, altars, keys) and the other autogenerators (grass, bones,
// rubble, columns, crystal walls...) are not ported.
type BrogueGenerator struct {
	random        *rand.Rand
	width, height int
	level         int
}

func NewBrogueGenerator(random *rand.Rand, mapCols, mapRows, level int) *BrogueGenerator {
	return &BrogueGenerator{random: random, width: mapCols, height: mapRows, level: level}
}

const brogueAmuletLevel = 26

// the dungeon layer of the carving grid
const (
	bGranite = iota
	bFloor
	bDoor
	bSecretDoor
)

// the liquid and surface layers
const (
	bNothing = iota
	bDeepWater
	bShallowWater
	bLava
	bChasm
	bChasmEdge
	bBridge
	bFungus
	bFungusForest
)

type brogueLevel struct {
	*BrogueGenerator
	dungeon, liquid, surface []int
	roomOf                   []int // index+1 of the room (with its hallway) a tile was dug for
	rooms                    int
}

func (g *BrogueGenerator) Generate() *DungeonMap {
	for {
		if m := g.try(); m != nil {
			return m
		}
	}
}

func (g *BrogueGenerator) rnd(lo, hi int) int { return lo + g.random.Intn(hi-lo+1) }
func (g *BrogueGenerator) percent(p int) bool { return g.random.Intn(100) < p }

func (l *brogueLevel) in(x, y int) bool { return x >= 0 && y >= 0 && x < l.width && y < l.height }

// blocks is T_PATHING_BLOCKER: walls and the lakes.
func (l *brogueLevel) blocks(i int) bool {
	switch l.liquid[i] {
	case bDeepWater, bLava, bChasm:
		return true
	}
	return l.dungeon[i] == bGranite
}

func (g *BrogueGenerator) try() *DungeonMap {
	n := g.width * g.height
	l := &brogueLevel{BrogueGenerator: g, dungeon: make([]int, n), liquid: make([]int, n), surface: make([]int, n),
		roomOf: make([]int, n)}
	l.carve()
	l.addLoops(20)
	for i, d := range l.dungeon {
		if d == bDoor && !g.percent(60) {
			l.dungeon[i] = bFloor
		}
	}
	l.fillLakes(l.designLakes())
	l.autogenerate(bFungus, 7, 15, -300, 70, 14, 60, 8)
	l.autogenerate(bFungusForest, 13, 30, -600, 50, 12, 100, 45)
	l.removeDiagonalOpenings()
	l.cleanUpLakeBoundaries()
	for l.buildABridge() {
	}
	l.finishDoors()
	return l.toMap()
}

// --- carving ---

// brogueGrid is a scratch room design, sized like the level.
type brogueGrid struct {
	cells         []int
	width, height int
}

func (l *brogueLevel) newGrid() *brogueGrid {
	return &brogueGrid{cells: make([]int, l.width*l.height), width: l.width, height: l.height}
}
func (gr *brogueGrid) in(x, y int) bool { return x >= 0 && y >= 0 && x < gr.width && y < gr.height }
func (gr *brogueGrid) at(x, y int) int {
	if !gr.in(x, y) {
		return 0
	}
	return gr.cells[x+y*gr.width]
}
func (gr *brogueGrid) set(x, y, v int) {
	if gr.in(x, y) {
		gr.cells[x+y*gr.width] = v
	}
}
func (gr *brogueGrid) rect(x, y, w, h, v int) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			gr.set(i, j, v)
		}
	}
}
func (gr *brogueGrid) circle(cx, cy, radius, v int) {
	for i := cx - radius; i <= cx+radius; i++ {
		for j := cy - radius; j <= cy+radius; j++ {
			if (i-cx)*(i-cx)+(j-cy)*(j-cy) < radius*radius+radius {
				gr.set(i, j, v)
			}
		}
	}
}

var bDirs = [4]geometry.Point{{Y: -1}, {Y: 1}, {X: -1}, {X: 1}} // Brogue's UP, DOWN, LEFT, RIGHT: dir^1 is the opposite
var invalidSite = geometry.Point{X: -1, Y: -1}

// roomFrequencies is DP_BASIC (or DP_BASIC_FIRST_ROOM) adjusted for the depth:
// cross, small symmetrical cross, small, circular, chunky, cave, cavern, entrance room; and the hallway chance.
func (l *brogueLevel) roomFrequencies(first bool) ([8]int, int) {
	descent := min(max(100*(l.level-1)/(brogueAmuletLevel-1), 0), 100)
	if first {
		if l.level == 1 {
			return [8]int{7: 1}, 0
		}
		return [8]int{10, 0, 0, 3, 7, 10, 10 + 50*descent/100, 0}, 0
	}
	return [8]int{2 + 20*(100-descent)/100, 1 + 10*(100-descent)/100, 1, 1 + 7*(100-descent)/100, 7, 1 + 10*descent/100, 0, 0},
		10 + 80*(100-descent)/100
}

func (l *brogueLevel) carve() {
	freq, _ := l.roomFrequencies(true)
	first := l.newGrid()
	l.designRandomRoom(first, false, nil, freq)
	l.insertRoom(first, geometry.Point{})

	freq, corridorChance := l.roomFrequencies(false)
	order := l.random.Perm(l.width * l.height)
	for built, attempts := 0, 0; built < 35 && attempts < 35; attempts++ {
		room := l.newGrid()
		var doorSites [4]geometry.Point
		l.designRandomRoom(room, attempts <= 35-5 && l.percent(corridorChance), &doorSites, freq)
		for _, i := range order {
			x, y := i%l.width, i/l.width
			dir := l.directionOfDoorSite(x, y)
			if dir < 0 {
				continue
			}
			site := doorSites[dir^1]
			offset := geometry.Point{X: x - site.X, Y: y - site.Y}
			if site != invalidSite && l.roomFitsAt(room, offset) {
				l.insertRoom(room, offset)
				l.dungeon[x+y*l.width] = bDoor
				built++
				break
			}
		}
	}
}

// directionOfDoorSite: the outward direction of a door at (x, y) on the level, or -1.
func (l *brogueLevel) directionOfDoorSite(x, y int) int {
	return doorSiteDirection(l.in, func(x, y int) bool { return l.dungeon[x+y*l.width] != bGranite }, x, y)
}

func doorSiteDirection(in func(x, y int) bool, floor func(x, y int) bool, x, y int) int {
	if !in(x, y) || floor(x, y) {
		return -1
	}
	solution := -1
	for dir, d := range bDirs {
		nx, ny, ox, oy := x+d.X, y+d.Y, x-d.X, y-d.Y
		if in(ox, oy) && in(nx, ny) && floor(ox, oy) {
			if solution >= 0 {
				return -1
			}
			solution = dir
		}
	}
	return solution
}

func (l *brogueLevel) roomFitsAt(room *brogueGrid, offset geometry.Point) bool {
	for i, v := range room.cells {
		if v == 0 {
			continue
		}
		x, y := i%room.width+offset.X, i/room.width+offset.Y
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if !l.in(x+dx, y+dy) || l.dungeon[x+dx+(y+dy)*l.width] != bGranite {
					return false
				}
			}
		}
	}
	return true
}

// insertRoom copies the room in, with its hallway (the cells of 2).
func (l *brogueLevel) insertRoom(room *brogueGrid, offset geometry.Point) {
	l.rooms++
	for i, v := range room.cells {
		if v == 0 {
			continue
		}
		x, y := i%room.width+offset.X, i/room.width+offset.Y
		if !l.in(x, y) {
			continue
		}
		j := x + y*l.width
		l.dungeon[j] = bFloor
		l.roomOf[j] = l.rooms
	}
}

func (l *brogueLevel) designRandomRoom(grid *brogueGrid, attachHallway bool, doorSites *[4]geometry.Point, freq [8]int) {
	sum := 0
	for _, f := range freq {
		sum += f
	}
	kind, roll := 0, l.random.Intn(sum)
	for ; kind < len(freq); kind++ {
		if roll < freq[kind] {
			break
		}
		roll -= freq[kind]
	}
	w, h := l.width, l.height
	switch kind {
	case 0: // cross room
		roomWidth := l.rnd(3, 12)
		roomX := l.rnd(max(0, w/2-(roomWidth-1)), min(w, w/2))
		roomWidth2 := l.rnd(4, 20)
		roomX2 := (roomX + roomWidth/2 + l.rnd(0, 2) + l.rnd(0, 2) - 3) - roomWidth2/2
		roomHeight := l.rnd(3, 7)
		roomY := h/2 - roomHeight
		roomHeight2 := l.rnd(2, 5)
		roomY2 := h/2 - roomHeight2 - (l.rnd(0, 2) + l.rnd(0, 1))
		grid.rect(roomX-5, roomY+5, roomWidth, roomHeight, 1)
		grid.rect(roomX2-5, roomY2+5, roomWidth2, roomHeight2, 1)
	case 1: // small symmetrical cross room
		majorWidth, majorHeight := l.rnd(4, 8), l.rnd(4, 5)
		minorWidth, minorHeight := l.rnd(3, 4), 3
		if majorHeight%2 == 0 {
			minorWidth--
		}
		if majorWidth%2 == 0 {
			minorHeight--
		}
		grid.rect((w-majorWidth)/2, (h-minorHeight)/2, majorWidth, minorHeight, 1)
		grid.rect((w-minorWidth)/2, (h-majorHeight)/2, minorWidth, majorHeight, 1)
	case 2: // small room
		width, height := l.rnd(3, 6), l.rnd(2, 4)
		grid.rect((w-width)/2, (h-height)/2, width, height, 1)
	case 3: // circular room
		radius := l.rnd(2, 4)
		if l.percent(5) {
			radius = l.rnd(4, 10)
		}
		grid.circle(w/2, h/2, radius, 1)
		if radius > 6 && l.percent(50) {
			grid.circle(w/2, h/2, l.rnd(3, radius-3), 0)
		}
	case 4: // chunky room
		grid.circle(w/2, h/2, 2, 1)
		minX, maxX, minY, maxY := w/2-3, w/2+3, h/2-3, h/2+3
		for i, chunks := 0, l.rnd(2, 8); i < chunks; {
			x, y := l.rnd(minX, maxX), l.rnd(minY, maxY)
			if grid.at(x, y) != 0 {
				grid.circle(x, y, 2, 1)
				i++
				minX, maxX = max(1, min(x-3, minX)), min(w-2, max(x+3, maxX))
				minY, maxY = max(1, min(y-3, minY)), min(h-2, max(y+3, maxY))
			}
		}
	case 5: // cave
		switch l.rnd(0, 2) {
		case 0:
			l.designCavern(grid, 3, 12, 4, 8)
		case 1:
			l.designCavern(grid, 3, 12, 15, h-2)
		case 2:
			l.designCavern(grid, 20, 27, 4, 8) // 27 is DROWS-2 of Brogue's map, used as a width
		}
	case 6: // cavern; Brogue's CAVE_MIN_WIDTH 50 and CAVE_MIN_HEIGHT 20 of its 79x29 map
		l.designCavern(grid, 50*w/79, w-2, 20*h/29, h-2)
	case 7: // entrance room: the big upside-down T at the bottom of level 1
		grid.rect(w/2-8/2-1, h-10-2, 8, 10, 1)
		grid.rect(w/2-20/2-1, h-5-2, 20, 5, 1)
	}
	if doorSites != nil {
		l.chooseRandomDoorSites(grid, doorSites)
		if attachHallway {
			l.attachHallwayTo(grid, doorSites)
		}
	}
}

func (l *brogueLevel) designCavern(grid *brogueGrid, minWidth, maxWidth, minHeight, maxHeight int) {
	blob, bx, by, bw, bh := l.createBlob(5, minWidth, minHeight, maxWidth, maxHeight, 55, "ffffffttt", "ffffttttt")
	dx, dy := (l.width-bw)/2-bx, (l.height-bh)/2-by
	for i, v := range blob.cells {
		if v != 0 {
			grid.set(i%blob.width+dx, i/blob.width+dy, 1)
		}
	}
}

// createBlob is createBlobOnGrid: cellular automata on noise until the biggest blob is big enough,
// which alone is kept. It returns the blob and its bounds.
func (l *brogueLevel) createBlob(rounds, minWidth, minHeight, maxWidth, maxHeight, seeded int, birth, survival string) (*brogueGrid, int, int, int, int) {
	for {
		grid := l.newGrid()
		for i := 0; i < maxWidth; i++ {
			for j := 0; j < maxHeight; j++ {
				if l.percent(seeded) {
					grid.set(i, j, 1)
				}
			}
		}
		for k := 0; k < rounds; k++ {
			old := append([]int(nil), grid.cells...)
			for i := range grid.cells {
				x, y, count := i%grid.width, i/grid.width, 0
				if x > maxWidth || y > maxHeight { // with 3 neighbours at most out there, nothing is born
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					for dy := -1; dy <= 1; dy++ {
						if (dx != 0 || dy != 0) && grid.in(x+dx, y+dy) && old[x+dx+(y+dy)*grid.width] != 0 {
							count++
						}
					}
				}
				switch {
				case old[i] == 0 && birth[count] == 't':
					grid.cells[i] = 1
				case old[i] != 0 && survival[count] == 't':
				default:
					grid.cells[i] = 0
				}
			}
		}
		// number the blobs from 2, keep the biggest
		top, topSize := 0, 0
		for i := range grid.cells {
			if grid.cells[i] != 1 {
				continue
			}
			number, size := 2+i, 0
			for queue := []int{i}; len(queue) > 0; queue = queue[1:] {
				c := queue[0]
				if grid.cells[c] != 1 {
					continue
				}
				grid.cells[c] = number
				size++
				x, y := c%grid.width, c/grid.width
				for _, d := range bDirs {
					if grid.at(x+d.X, y+d.Y) == 1 {
						queue = append(queue, x+d.X+(y+d.Y)*grid.width)
					}
				}
			}
			if size > topSize {
				top, topSize = number, size
			}
		}
		minX, minY, maxX, maxY := grid.width, grid.height, -1, -1
		for i, v := range grid.cells {
			if v == top && top != 0 {
				grid.cells[i] = 1
				x, y := i%grid.width, i/grid.width
				minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			} else {
				grid.cells[i] = 0
			}
		}
		if maxX-minX+1 >= minWidth && maxY-minY+1 >= minHeight {
			return grid, minX, minY, maxX - minX + 1, maxY - minY + 1
		}
	}
}

func (l *brogueLevel) chooseRandomDoorSites(room *brogueGrid, doorSites *[4]geometry.Point) {
	candidates := [4][]geometry.Point{}
	for i := range room.cells {
		x, y := i%room.width, i/room.width
		dir := doorSiteDirection(room.in, func(x, y int) bool { return room.at(x, y) == 1 }, x, y)
		if dir < 0 {
			continue
		}
		ok := true
		for k, nx, ny := 0, x+bDirs[dir].X, y+bDirs[dir].Y; k < 10 && room.in(nx, ny) && ok; k, nx, ny = k+1, nx+bDirs[dir].X, ny+bDirs[dir].Y {
			ok = room.at(nx, ny) == 0
		}
		if ok {
			candidates[dir] = append(candidates[dir], geometry.Point{X: x, Y: y})
		}
	}
	for dir, c := range candidates {
		doorSites[dir] = invalidSite
		if len(c) > 0 {
			doorSites[dir] = c[l.random.Intn(len(c))]
		}
	}
}

func (l *brogueLevel) attachHallwayTo(room *brogueGrid, doorSites *[4]geometry.Point) {
	const hMax, vMax = 15, 9
	dir := -1
	for _, d := range l.random.Perm(4) {
		if s := doorSites[d]; s != invalidSite && room.in(s.X+bDirs[d].X*hMax, s.Y+bDirs[d].Y*vMax) {
			dir = d
			break
		}
	}
	if dir < 0 {
		return
	}
	length := l.rnd(5, hMax)
	if dir < 2 {
		length = l.rnd(2, vMax)
	}
	x, y := doorSites[dir].X, doorSites[dir].Y
	for i := 0; i < length; i++ {
		room.set(x, y, 2)
		x, y = x+bDirs[dir].X, y+bDirs[dir].Y
	}
	x, y = min(max(x-bDirs[dir].X, 0), room.width-1), min(max(y-bDirs[dir].Y, 0), room.height-1)
	oblique := l.percent(15)
	for d2, d := range bDirs {
		nx, ny := x+d.X, y+d.Y
		if (d2 != dir && !oblique) || !room.in(nx, ny) || room.at(nx, ny) != 0 {
			doorSites[d2] = invalidSite
		} else {
			doorSites[d2] = geometry.Point{X: nx, Y: ny}
		}
	}
}

// distances is a breadth-first search over what is not blocked, -1 where it does not reach.
func (l *brogueLevel) distances(from int, blocked func(i int) bool) []int {
	dist := make([]int, len(l.dungeon))
	for i := range dist {
		dist[i] = -1
	}
	dist[from] = 0
	for queue := []int{from}; len(queue) > 0; queue = queue[1:] {
		c := queue[0]
		x, y := c%l.width, c/l.width
		for _, d := range bDirs {
			if nx, ny := x+d.X, y+d.Y; l.in(nx, ny) {
				if n := nx + ny*l.width; dist[n] < 0 && !blocked(n) {
					dist[n] = dist[c] + 1
					queue = append(queue, n)
				}
			}
		}
	}
	return dist
}

func (l *brogueLevel) addLoops(minimumDistance int) {
	wall := func(i int) bool { return l.dungeon[i] == bGranite }
	for _, i := range l.random.Perm(len(l.dungeon)) {
		if !wall(i) {
			continue
		}
		x, y := i%l.width, i/l.width
		for _, d := range [2]geometry.Point{{X: 1}, {Y: 1}} {
			if !l.in(x+d.X, y+d.Y) || !l.in(x-d.X, y-d.Y) {
				continue
			}
			a, b := x+d.X+(y+d.Y)*l.width, x-d.X+(y-d.Y)*l.width
			if l.dungeon[a] == bFloor && l.dungeon[b] == bFloor {
				if dist := l.distances(a, wall)[b]; dist < 0 || dist > minimumDistance {
					l.dungeon[i] = bDoor
					break
				}
			}
		}
	}
}

// --- lakes ---

// designLakes returns the lake map: blobs of 30x15 down to 20x10, each placed only where every dry tile
// can still be reached from every other.
func (l *brogueLevel) designLakes() []bool {
	lakes := make([]bool, len(l.dungeon))
	for maxHeight, maxWidth := 15, 30; maxHeight >= 10; maxHeight, maxWidth = maxHeight-1, maxWidth-2 {
		blob, bx, by, bw, bh := l.createBlob(5, 4, 4, maxWidth, min(maxHeight, l.height-4), 55, "ffffftttt", "ffffttttt")
		for k := 0; k < 20; k++ {
			x := l.rnd(1-bx, l.width-bw-bx-2)
			y := l.rnd(1-by, l.height-bh-by-2)
			inBlob := func(i int) bool { return blob.at(i%l.width-x, i/l.width-y) != 0 }
			if l.disconnected(func(i int) bool { return l.blocks(i) || lakes[i] || inBlob(i) }) {
				continue
			}
			for i := range lakes {
				if inBlob(i) {
					lakes[i] = true
					l.dungeon[i] = bFloor
				}
			}
			break
		}
	}
	return lakes
}

// disconnected: not every tile that is not blocked can be reached from the first one.
func (l *brogueLevel) disconnected(blocked func(i int) bool) bool {
	start := -1
	for i := range l.dungeon {
		if !blocked(i) {
			start = i
			break
		}
	}
	if start < 0 {
		return true
	}
	dist := l.distances(start, blocked)
	for i := range dist {
		if dist[i] < 0 && !blocked(i) {
			return true
		}
	}
	return false
}

// liquidType: lava from LavaFromLevel, never a chasm on the last level. Brogue's brimstone (from level 17) is
// not ported, so the roll is between the other three.
func (l *brogueLevel) liquidType() (deep, shallow, width int) {
	randMin := 0
	if l.level < LavaFromLevel {
		randMin = 1
	}
	roll := l.rnd(randMin, 2)
	if l.level >= brogueAmuletLevel {
		roll = 1
	}
	switch roll {
	case 0:
		return bLava, bNothing, 0
	case 1:
		return bDeepWater, bShallowWater, 2
	}
	return bChasm, bChasmEdge, 1
}

func (l *brogueLevel) fillLakes(lakes []bool) {
	for start := range lakes {
		if !lakes[start] {
			continue
		}
		deep, shallow, width := l.liquidType()
		wreath := make([]bool, len(lakes))
		// fillLake: the lake and every other lake tile within 4 of it
		for stack := []int{start}; len(stack) > 0; {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			x, y := c%l.width, c/l.width
			for i := x - 4; i <= x+4; i++ {
				for j := y - 4; j <= y+4; j++ {
					if n := i + j*l.width; l.in(i, j) && lakes[n] {
						lakes[n] = false
						l.liquid[n] = deep
						wreath[n] = true
						stack = append(stack, n)
					}
				}
			}
		}
		if shallow == bNothing {
			continue
		}
		for c, w := range wreath {
			if !w {
				continue
			}
			x, y := c%l.width, c/l.width
			for k := x - width; k <= x+width; k++ {
				for m := y - width; m <= y+width; m++ {
					if n := k + m*l.width; l.in(k, m) && l.liquid[n] == bNothing && (x-k)*(x-k)+(y-m)*(y-m) <= width*width {
						l.liquid[n] = shallow
						if l.dungeon[n] == bDoor {
							l.dungeon[n] = bFloor
						}
					}
				}
			}
		}
	}
}

// copyTile copies all layers of a tile, as Brogue does when it opens a corner or a lake boundary.
func (l *brogueLevel) copyTile(to, from int) {
	l.dungeon[to], l.liquid[to], l.surface[to] = l.dungeon[from], l.liquid[from], l.surface[from]
	l.roomOf[to] = l.roomOf[from]
}

// cleanUpLakeBoundaries knocks down a wall or lake tile between two tiles of the same other lake.
func (l *brogueLevel) cleanUpLakeBoundaries() {
	lake := func(i int) int { // the kind of lake that blocks the way, 0 for none
		switch l.liquid[i] {
		case bDeepWater, bLava, bChasm:
			return l.liquid[i]
		}
		return 0
	}
	for failsafe, reverse, changed := 100, true, true; changed && failsafe > 0; failsafe-- {
		changed, reverse = false, !reverse
		for a := 1; a < l.width-1; a++ {
			i := a
			if reverse {
				i = l.width - 1 - a
			}
			for b := 1; b < l.height-1; b++ {
				j := b
				if reverse {
					j = l.height - 1 - b
				}
				c := i + j*l.width
				if lake(c) == 0 && l.dungeon[c] != bGranite {
					continue
				}
				from := -1
				if k := lake(c - 1); k != 0 && k != lake(c) && k == lake(c+1) {
					from = c + 1
				} else if k := lake(c - l.width); k != 0 && k != lake(c) && k == lake(c+l.width) {
					from = c + l.width
				}
				if from >= 0 {
					changed = true
					l.copyTile(c, from)
				}
			}
		}
	}
}

// removeDiagonalOpenings: of two open tiles that touch only at a corner, one of the walls between them is opened.
func (l *brogueLevel) removeDiagonalOpenings() {
	open := func(x, y int) bool { return l.dungeon[x+y*l.width] != bGranite }
	for removed := true; removed; {
		removed = false
		for i := 0; i < l.width-1; i++ {
			for j := 0; j < l.height-1; j++ {
				for k := 0; k <= 1; k++ {
					if open(i+k, j) && !open(i+1-k, j) && !open(i+k, j+1) && open(i+1-k, j+1) {
						x1, x2, y1 := i+k, i+1-k, j+1
						if l.percent(50) {
							x1, x2, y1 = i+1-k, i+k, j
						}
						l.copyTile(x1+y1*l.width, x2+y1*l.width)
						removed = true
					}
				}
			}
		}
	}
}

// buildABridge lays one bridge over a chasm where it saves enough of a walk, false if there is none.
func (l *brogueLevel) buildABridge() bool {
	ratioX := 100 + (100+100*l.level/9)*l.rnd(10, 20)/10
	ratioY := 100 + (400+100*l.level/18)*l.rnd(10, 20)/10
	chasm := func(x, y int) bool { return l.liquid[x+y*l.width] == bChasm }
	wall := func(x, y int) bool { return l.dungeon[x+y*l.width] == bGranite }
	land := func(x, y int) bool { return !l.blocks(x + y*l.width) }
	for _, i := range l.random.Perm(l.width) {
		for _, j := range l.random.Perm(l.height) {
			if i == 0 || j == 0 || i == l.width-1 || j == l.height-1 || !land(i, j) {
				continue
			}
			for _, d := range [2]geometry.Point{{X: 1}, {Y: 1}} {
				ratio := ratioX
				if d.Y != 0 {
					ratio = ratioY
				}
				side := geometry.Point{X: d.Y, Y: d.X}
				exposed, k := false, 1
				for ; l.in(i+k*d.X, j+k*d.Y); k++ {
					x, y := i+k*d.X, j+k*d.Y
					if !chasm(x, y) || !l.in(x-side.X, y-side.Y) || !l.in(x+side.X, y+side.Y) ||
						!(chasm(x-side.X, y-side.Y) || wall(x-side.X, y-side.Y)) || !(chasm(x+side.X, y+side.Y) || wall(x+side.X, y+side.Y)) {
						break
					}
					if !wall(x-side.X, y-side.Y) && !wall(x+side.X, y+side.Y) {
						exposed = true
					}
				}
				x, y := i+k*d.X, j+k*d.Y
				if !l.in(x, y) || k <= 3 || !exposed || !land(x, y) {
					continue
				}
				if dist := l.distances(i+j*l.width, l.blocks)[x+y*l.width]; dist >= 0 && 100*dist/k <= ratio {
					continue
				}
				for s := 0; s <= k; s++ { // with Brogue's BRIDGE_EDGE at both ends
					l.liquid[i+s*d.X+(j+s*d.Y)*l.width] = bBridge
				}
				return true
			}
		}
	}
	return false
}

// finishDoors: a door with a way around it or with blockers on three sides goes; deeper doors are more often secret.
func (l *brogueLevel) finishDoors() {
	secretChance := min(max((l.level-1)*67/(brogueAmuletLevel-1), 0), 67)
	passable := func(i int) bool { return l.dungeon[i] != bGranite }
	for i := 1; i < l.width-1; i++ {
		for j := 1; j < l.height-1; j++ {
			c := i + j*l.width
			if l.dungeon[c] != bDoor {
				continue
			}
			blockers := 0
			for _, n := range []int{c + 1, c - 1, c + l.width, c - l.width} {
				if l.blocks(n) {
					blockers++
				}
			}
			switch {
			case (passable(c+1) || passable(c-1)) && (passable(c+l.width) || passable(c-l.width)), blockers >= 3:
				l.dungeon[c] = bFloor
			case l.percent(secretChance):
				l.dungeon[c] = bSecretDoor
			}
		}
	}
}

// --- fungus ---

// autogenerate is runAutogenerators for one dungeon feature on dry floor: from minDepth, the number is
// (intercept + depth * slope) / 100, one more while a roll under frequency succeeds; each spreads as in
// spawnMapDF, from startProb percent less probDecr every step.
func (l *brogueLevel) autogenerate(surface, minDepth, frequency, intercept, slope, maxNumber, startProb, probDecr int) {
	if l.level < minDepth {
		return
	}
	count := min((intercept+l.level*slope)/100, maxNumber)
	for l.percent(frequency) && count < maxNumber {
		count++
	}
	dryFloor := func(i int) bool {
		return l.dungeon[i] == bFloor && l.liquid[i] == bNothing && l.surface[i] == bNothing
	}
	for n := 0; n < count; n++ {
		var spots []int
		for i := range l.dungeon {
			if dryFloor(i) {
				spots = append(spots, i)
			}
		}
		if len(spots) == 0 {
			return
		}
		start := spots[l.random.Intn(len(spots))]
		spawned := []int{start}
		seen := map[int]bool{start: true}
		front := spawned
		for prob := startProb; len(front) > 0 && prob > 0; prob -= probDecr {
			var next []int
			for _, c := range front {
				x, y := c%l.width, c/l.width
				for _, d := range bDirs {
					if nx, ny := x+d.X, y+d.Y; l.in(nx, ny) && l.dungeon[nx+ny*l.width] != bGranite && !seen[nx+ny*l.width] && l.percent(prob) {
						seen[nx+ny*l.width] = true
						next = append(next, nx+ny*l.width)
					}
				}
			}
			spawned = append(spawned, next...)
			front = next
		}
		for _, c := range spawned {
			if dryFloor(c) { // DFF_BLOCKED_BY_OTHER_LAYERS
				l.surface[c] = surface
			}
		}
	}
}

// --- the finished map ---

func (l *brogueLevel) toMap() *DungeonMap {
	m := NewDungeonMap(l.width, l.height)
	for i := range l.dungeon {
		p := geometry.Point{X: i % l.width, Y: i / l.width}
		switch l.dungeon[i] {
		case bGranite:
			continue
		case bDoor, bSecretDoor:
			m.SetDoor(p.X, p.Y)
			if l.dungeon[i] == bSecretDoor {
				m.secretDoors[p] = true // looks like wall until found
			}
			continue
		}
		tile := Room
		switch l.liquid[i] {
		case bDeepWater:
			tile = DeepWater
		case bShallowWater:
			tile = ShallowWater
		case bLava:
			tile = Lava
		case bChasm:
			tile = Chasm
		case bChasmEdge:
			tile = ChasmEdge
		case bBridge:
			tile = Bridge
		}
		switch {
		case tile != Room && tile != Corridor:
		case l.surface[i] == bFungus:
			tile = Fungus
		case l.surface[i] == bFungusForest:
			tile = FungusForest
		}
		m.tiles[i] = tile
	}

	// a lake belongs to the room it cuts into (one in solid rock belongs to none), a doorway left
	// without a door to a room next to it
	owner := append([]int(nil), l.roomOf...)
	var queue []int
	for i, o := range owner {
		if o > 0 {
			queue = append(queue, i)
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		c := queue[0]
		for _, d := range bDirs {
			if x, y := c%l.width+d.X, c/l.width+d.Y; l.in(x, y) {
				if n := x + y*l.width; owner[n] == 0 && l.dungeon[n] == bFloor {
					owner[n] = owner[c]
					queue = append(queue, n)
				}
			}
		}
	}
	tilesOf := make([][]geometry.Point, l.rooms+1)
	for i, o := range owner {
		if o > 0 && m.tiles[i] != Wall && m.tiles[i] != Door {
			tilesOf[o] = append(tilesOf[o], geometry.Point{X: i % l.width, Y: i / l.width})
		}
	}

	// the walls of a room are finishWalls: the rock and doors around it
	lit := false // Brogue has no lit rooms: light comes from lava, fungus and the player
	var entrance *DungeonRoom
	var roomTiles [][]geometry.Point
	for r, tiles := range tilesOf {
		if len(tiles) == 0 {
			continue
		}
		bounds := geometry.NewRect(l.width, l.height, 0, 0)
		walls := map[geometry.Point]bool{}
		for _, p := range tiles {
			bounds = geometry.NewRect(min(bounds.Min.X, p.X), min(bounds.Min.Y, p.Y), max(bounds.Max.X, p.X+1), max(bounds.Max.Y, p.Y+1))
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if w := p.Add(geometry.Point{X: dx, Y: dy}); m.Contains(w) && (m.GetTileAt(w) == Wall || m.GetTileAt(w) == Door) {
						walls[w] = true
					}
				}
			}
		}
		room := NewDungeonRoomFromTiles(bounds, tiles)
		for _, w := range keysSorted(walls, l.width) {
			room.wallTiles = append(room.wallTiles, w)
			if m.GetTileAt(w) == Door && !m.secretDoors[w] {
				room.SetDoor(w)
			}
		}
		room.SetLit(lit)
		m.rooms = append(m.rooms, room)
		roomTiles = append(roomTiles, tiles)
		if r == 1 && l.level == 1 {
			entrance = room
		}
	}
	if len(m.rooms) < 2 {
		return nil
	}
	// stairs go on plain floor
	floorOf := func(room int) (geometry.Point, bool) {
		var floor []geometry.Point
		for _, p := range roomTiles[room] {
			if t := m.GetTileAt(p); t == Room || t == Corridor {
				floor = append(floor, p)
			}
		}
		if len(floor) == 0 {
			return geometry.Point{}, false
		}
		return floor[l.random.Intn(len(floor))], true
	}
	down := l.random.Intn(len(m.rooms))
	up := (down + 1 + l.random.Intn(len(m.rooms)-1)) % len(m.rooms)
	if entrance != nil {
		up, down = 0, 1+l.random.Intn(len(m.rooms)-1)
	}
	downAt, okDown := floorOf(down)
	upAt, okUp := floorOf(up)
	if foot := (geometry.Point{X: l.width/2 - 1, Y: l.height - 3}); entrance != nil && m.GetTileAt(foot) == Room {
		upAt = foot // the way out is at the foot of the entrance room
	}
	if !okDown || !okUp {
		return nil
	}
	m.SetStairsDown(downAt)
	m.SetStairsUp(upAt)
	return m
}

// keysSorted lists the points in reading order, for the same level from the same seed.
func keysSorted(set map[geometry.Point]bool, width int) []geometry.Point {
	result := make([]geometry.Point, 0, len(set))
	for p := range set {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b geometry.Point) int { return (a.X + a.Y*width) - (b.X + b.Y*width) })
	return result
}
