package dungen

import (
	"math/rand"
	"rx1/geometry"
	"slices"
)

const (
	nhXLim     = 4  // rect.c: free space kept around a room
	nhYLim     = 3  //
	nhMaxRooms = 40 // MAXNROFROOMS

	// nhThemedPercent is how many rooms try to be a themed shape. NetHack's themerms.lua makes
	// about 4% of them, which is hardly ever seen on a map of 8 rooms.
	nhThemedPercent = 30
)

func rn2(random *rand.Rand, n int) int {
	if n <= 0 {
		return 0
	}
	return random.Intn(n)
}

type nhRect struct{ lx, ly, hx, hy int }

// nhRoom mirrors struct mkroom: lx..hx and ly..hy are the floor, the walls are one tile outside.
type nhRoom struct {
	lx, ly, hx, hy int
	doors          int
	irregular      bool // not a rectangle: the walls are not on the edge of lx..hx, ly..hy
	smeq           int  // rooms with equal smeq are known to be connected
	room           *DungeonRoom
}

// NetHackGenerator is a port of the classic NetHack 5.0 level: mklev.c (makerooms, makecorridors,
// make_niches), rect.c (free rectangles), sp_lev.c (create_room, check_room, dig_corridor).
//
// Rooms are placed in random free rectangles; each room splits the rectangle it took into
// smaller ones. The rooms are sorted left to right and joined 0-1, 1-2, ..., then 0-2, 1-3, ...
// where still unconnected, then every unconnected pair, then four or more extra corridors that
// may end blind. Niches are closets behind a (mostly secret) door.
//
// Not ported: themed rooms (themerms.lua), vaults, special rooms (mkroom.c), boulders in blind
// corridors and the trapdoor/teleport niches. Populating the level is game.spawnEntities' job.
type NetHackGenerator struct {
	random        *rand.Rand
	width, height int
	level         int
	m             *DungeonMap
	claimed       []bool // room floor and walls: not solid rock any more
	rects         []nhRect
	rooms         []*nhRoom
}

func NewNetHackGenerator(random *rand.Rand, mapCols, mapRows, level int) *NetHackGenerator {
	return &NetHackGenerator{random: random, width: mapCols, height: mapRows, level: level}
}

func (g *NetHackGenerator) Generate() *DungeonMap {
	g.m = NewDungeonMap(g.width, g.height)
	g.claimed = make([]bool, g.width*g.height)
	g.rects = []nhRect{{0, 0, g.width - 1, g.height - 1}}
	g.rooms = nil

	g.makeRooms()
	slices.SortStableFunc(g.rooms, func(a, b *nhRoom) int { return a.lx - b.lx })
	for i, r := range g.rooms {
		r.smeq = i
	}
	g.placeStairs()
	g.makeCorridors()
	g.makeNiches()
	return g.m
}

func (g *NetHackGenerator) rn2(n int) int { return rn2(g.random, n) }

func (g *NetHackGenerator) isStone(x, y int) bool {
	return g.m.GetTile(x, y) == Wall && !g.claimed[x+y*g.width]
}

func (g *NetHackGenerator) isCorridor(x, y int) bool { return g.m.GetTile(x, y) == Corridor }

func (g *NetHackGenerator) makeRooms() {
	for len(g.rooms) < nhMaxRooms-1 && len(g.rects) > 0 {
		if g.rn2(100) < nhThemedPercent && g.createThemedRoom() {
			continue
		}
		if !g.createRoom() {
			break
		}
	}
}

// createRoom is sp_lev.c create_room for a totally random room.
func (g *NetHackGenerator) createRoom() bool {
	xabs, yabs, dx, dy, r1, ok := g.pickSpot(-1, -1, false)
	if !ok {
		return false
	}
	g.splitRects(r1, nhRect{xabs - 1, yabs - 1, xabs + dx + 1, yabs + dy + 1})
	g.addRoom(xabs, yabs, xabs+dx, yabs+dy)
	return true
}

// pickSpot is the search of create_room: a random free rectangle and a place in it for a room whose
// floor is (dx+1)x(dy+1); dx < 0 rolls a random size. An exact room (the themed ones) is not moved
// or shrunk by checkRoom, there either is space or not.
func (g *NetHackGenerator) pickSpot(wantDx, wantDy int, exact bool) (xabs, yabs, dx, dy int, r1 nhRect, ok bool) {
	for try := 0; try <= 100 && len(g.rects) > 0; try++ {
		r1 = g.rects[g.rn2(len(g.rects))]
		dx, dy = wantDx, wantDy
		if dx < 0 {
			dxRange := 8
			if r1.hx-r1.lx > 28 {
				dxRange = 12
			}
			dx = 2 + g.rn2(dxRange)
			dy = 2 + g.rn2(4)
			if dx*dy > 50 {
				dy = 50 / dx
			}
		}
		xborder, yborder := nhXLim+1, nhYLim+1
		if r1.lx > 0 && r1.hx < g.width-1 {
			xborder = 2 * nhXLim
		}
		if r1.ly > 0 && r1.hy < g.height-1 {
			yborder = 2 * nhYLim
		}
		if r1.hx-r1.lx < dx+3+xborder || r1.hy-r1.ly < dy+3+yborder {
			continue
		}
		xmin, ymin, xfloor, yfloor := 3, 2, nhXLim, nhYLim
		if r1.lx > 0 {
			xmin = r1.lx
		} else {
			xfloor = 3
		}
		if r1.ly > 0 {
			ymin = r1.ly
		} else {
			yfloor = 2
		}
		xabs = r1.lx + xfloor + g.rn2(r1.hx-xmin-dx-xborder+1)
		yabs = r1.ly + yfloor + g.rn2(r1.hy-ymin-dy-yborder+1)
		if !exact && r1.ly == 0 && r1.hy >= g.height-1 && (len(g.rooms) == 0 || g.rn2(len(g.rooms)) == 0) && yabs+dy > g.height/2 {
			yabs = 2 + g.rn2(3)
			if len(g.rooms) < 4 && dy > 1 {
				dy--
			}
		}
		sx, sdx, sy, sdy := xabs, dx, yabs, dy
		if !g.checkRoom(&xabs, &dx, &yabs, &dy, exact) || (exact && (sx != xabs || sdx != dx || sy != yabs || sdy != dy)) {
			continue
		}
		return xabs, yabs, dx, dy, r1, true
	}
	return 0, 0, 0, 0, nhRect{}, false
}

// checkRoom is sp_lev.c check_room: it shrinks the room away from rock that is already used, or gives up.
func (g *NetHackGenerator) checkRoom(lowx, ddx, lowy, ddy *int, exact bool) bool {
	hix, hiy := *lowx+*ddx, *lowy+*ddy
	*lowx = max(*lowx, 3)
	*lowy = max(*lowy, 2)
	hix = min(hix, g.width-3)
	hiy = min(hiy, g.height-3)
chk:
	for {
		if hix <= *lowx || hiy <= *lowy {
			return false
		}
		for x := *lowx - nhXLim; x <= hix+nhXLim; x++ {
			if x <= 0 || x >= g.width {
				continue
			}
			for y := max(*lowy-nhYLim, 0); y <= min(hiy+nhYLim, g.height-1); y++ {
				if !g.claimed[x+y*g.width] {
					continue
				}
				if exact || g.rn2(3) == 0 {
					return false
				}
				if x < *lowx {
					*lowx = x + nhXLim + 1
				} else {
					hix = x - nhXLim - 1
				}
				if y < *lowy {
					*lowy = y + nhYLim + 1
				} else {
					hiy = y - nhYLim - 1
				}
				continue chk
			}
		}
		break
	}
	*ddx, *ddy = hix-*lowx, hiy-*lowy
	return true
}

// addRoom is mklev.c add_room.
func (g *NetHackGenerator) addRoom(lx, ly, hx, hy int) {
	room := NewDungeonRoomFromRect(geometry.NewRect(lx, ly, hx+1, hy+1))
	room.SetLit(g.lit())
	g.m.AddRoomAndSetTiles(room)
	for x := lx - 1; x <= hx+1; x++ {
		for y := ly - 1; y <= hy+1; y++ {
			g.claimed[x+y*g.width] = true
		}
	}
	g.rooms = append(g.rooms, &nhRoom{lx: lx, ly: ly, hx: hx, hy: hy, room: room})
}

func nhIntersect(r1, r2 nhRect) (nhRect, bool) {
	if r2.lx > r1.hx || r2.ly > r1.hy || r2.hx < r1.lx || r2.hy < r1.ly {
		return nhRect{}, false
	}
	r3 := nhRect{max(r1.lx, r2.lx), max(r1.ly, r2.ly), min(r1.hx, r2.hx), min(r1.hy, r2.hy)}
	return r3, r3.lx <= r3.hx && r3.ly <= r3.hy
}

func (g *NetHackGenerator) removeRect(r nhRect) {
	if i := slices.Index(g.rects, r); i >= 0 {
		g.rects[i] = g.rects[len(g.rects)-1]
		g.rects = g.rects[:len(g.rects)-1]
	}
}

func (g *NetHackGenerator) addRect(r nhRect) {
	for _, o := range g.rects { // already inside another free rect
		if r.lx >= o.lx && r.ly >= o.ly && r.hx <= o.hx && r.hy <= o.hy {
			return
		}
	}
	g.rects = append(g.rects, r)
}

// splitRects is rect.c split_rects: r1 is a free rect that contains the used r2; replace it by the free rest.
func (g *NetHackGenerator) splitRects(r1, r2 nhRect) {
	old := r1
	g.removeRect(r1)
	for i := len(g.rects) - 1; i >= 0; i-- {
		if i >= len(g.rects) { // the recursion removed several rects
			continue
		}
		if r, ok := nhIntersect(g.rects[i], r2); ok {
			g.splitRects(g.rects[i], r)
		}
	}
	border := func(atEdge bool, lim int) int {
		if atEdge {
			return lim + 1 + 4
		}
		return 2*lim + 4
	}
	if r2.ly-old.ly-1 > border(old.hy >= g.height-1, nhYLim) {
		r := old
		r.hy = r2.ly - 2
		g.addRect(r)
	}
	if r2.lx-old.lx-1 > border(old.hx >= g.width-1, nhXLim) {
		r := old
		r.hx = r2.lx - 2
		g.addRect(r)
	}
	if old.hy-r2.hy-1 > border(old.ly <= 0, nhYLim) {
		r := old
		r.ly = r2.hy + 2
		g.addRect(r)
	}
	if old.hx-r2.hx-1 > border(old.lx <= 0, nhXLim) {
		r := old
		r.lx = r2.hx + 2
		g.addRect(r)
	}
}

// somexy is a random floor tile of the room itself, not of a room inside it.
func (g *NetHackGenerator) somexy(r *nhRoom) geometry.Point {
	for {
		p := geometry.Point{X: r.lx + g.rn2(r.hx-r.lx+1), Y: r.ly + g.rn2(r.hy-r.ly+1)}
		if r.room.FloorContains(p) && g.m.GetTileAt(p) == Room {
			return p
		}
	}
}

// placeStairs is makelevel's: down stairs in a random room, up stairs in another one if there is one.
func (g *NetHackGenerator) placeStairs() {
	down := g.rn2(len(g.rooms))
	g.m.SetStairsDown(g.somexy(g.rooms[down]))
	up := down
	if len(g.rooms) > 1 {
		up = g.rn2(len(g.rooms) - 1)
		if up >= down {
			up++
		}
	}
	g.m.SetStairsUp(g.somexy(g.rooms[up]))
}

func (g *NetHackGenerator) bydoor(p geometry.Point) bool {
	for _, d := range cardinalDirections {
		if n := p.Add(d); g.m.Contains(n) && g.m.GetTileAt(n) == Door {
			return true
		}
	}
	return false
}

// okdoor is mklev.c okdoor: a room wall next to something walkable, and not next to another door.
func (g *NetHackGenerator) okdoor(p geometry.Point) bool {
	if g.m.GetTileAt(p) != Wall || !g.claimed[p.X+p.Y*g.width] || g.bydoor(p) {
		return false
	}
	for _, d := range cardinalDirections {
		if n := p.Add(d); g.m.Contains(n) && g.m.GetTileAt(n) != Wall {
			return true
		}
	}
	return false
}

// inward is the direction from a wall on side 0 N, 1 E, 2 S, 3 W of a room into the room.
var inward = [4]geometry.Point{{Y: 1}, {X: -1}, {Y: -1}, {X: 1}}

// goodDoorPos is mklev.c good_rm_wall_doorpos: a wall (or door) of this room with floor behind it.
func (g *NetHackGenerator) goodDoorPos(p geometry.Point, side int, r *nhRoom) bool {
	if !g.m.Contains(p) || (g.m.GetTileAt(p) != Wall && g.m.GetTileAt(p) != Door) || !g.claimed[p.X+p.Y*g.width] || g.bydoor(p) {
		return false
	}
	return r.room.FloorContains(p.Add(inward[side]))
}

// finddposShift is mklev.c finddpos_shift: an irregular room has its walls away from the edge of its
// bounds, so walk from the edge into the room until there is a wall to put the door in.
func (g *NetHackGenerator) finddposShift(p geometry.Point, side int, r *nhRoom) (geometry.Point, bool) {
	if g.goodDoorPos(p, side, r) {
		return p, true
	}
	if !r.irregular {
		return p, false
	}
	free := func(q geometry.Point) bool { return g.m.Contains(q) && (g.isStone(q.X, q.Y) || g.isCorridor(q.X, q.Y)) }
	for q := p; free(q); {
		q = q.Add(inward[side])
		if g.goodDoorPos(q, side, r) {
			return q, true
		}
		if !free(q) || q.X < r.lx || q.X > r.hx || q.Y < r.ly || q.Y > r.hy {
			break
		}
	}
	return p, false
}

// finddpos is mklev.c finddpos: a door position on the given side of the room (dir 0 N, 1 E, 2 S, 3 W).
func (g *NetHackGenerator) finddpos(dir int, r *nhRoom) (geometry.Point, bool) {
	x1, x2, y1, y2 := r.lx, r.hx, r.ly, r.hy
	switch dir {
	case 0:
		y1, y2 = r.ly-1, r.ly-1
	case 2:
		y1, y2 = r.hy+1, r.hy+1
	case 3:
		x1, x2 = r.lx-1, r.lx-1
	case 1:
		x1, x2 = r.hx+1, r.hx+1
	}
	for try := 0; try < 20; try++ {
		p := geometry.Point{X: x1 + g.rn2(x2-x1+1), Y: y1 + g.rn2(y2-y1+1)}
		if q, ok := g.finddposShift(p, dir, r); ok {
			return q, true
		}
	}
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			if q, ok := g.finddposShift(geometry.Point{X: x, Y: y}, dir, r); ok {
				return q, true
			}
		}
	}
	return geometry.Point{X: x1, Y: y1}, false
}

// maybeSDoor is mklev.c maybe_sdoor: nothing is secret on the first two levels.
func (g *NetHackGenerator) maybeSDoor(chance int) bool {
	return g.level > 2 && g.rn2(max(2, chance)) == 0
}

// dodoor is mklev.c dodoor: a door that is secret now and then.
func (g *NetHackGenerator) dodoor(p geometry.Point, r *nhRoom) {
	g.placeDoor(p, r, g.maybeSDoor(8))
}

// placeDoor is mklev.c dosdoor. A door that exists already is made an ordinary one.
func (g *NetHackGenerator) placeDoor(p geometry.Point, r *nhRoom, secret bool) {
	if g.m.GetTileAt(p) == Door {
		delete(g.m.secretDoors, p)
		r.room.SetDoor(p)
		return
	}
	g.m.SetDoor(p.X, p.Y)
	r.doors++
	if secret {
		g.m.secretDoors[p] = true // the room keeps its wall here until found
		return
	}
	r.room.SetDoor(p)
}

func (g *NetHackGenerator) join(a, b int, nxcor bool) {
	croom, troom := g.rooms[a], g.rooms[b]
	var dx, dy int
	var cc, tt geometry.Point
	var ok1, ok2 bool
	switch {
	case troom.lx > croom.hx:
		dx = 1
		cc, ok1 = g.finddpos(1, croom)
		tt, ok2 = g.finddpos(3, troom)
	case troom.hy < croom.ly:
		dy = -1
		cc, ok1 = g.finddpos(0, croom)
		tt, ok2 = g.finddpos(2, troom)
	case troom.hx < croom.lx:
		dx = -1
		cc, ok1 = g.finddpos(3, croom)
		tt, ok2 = g.finddpos(1, troom)
	default:
		dy = 1
		cc, ok1 = g.finddpos(2, croom)
		tt, ok2 = g.finddpos(0, troom)
	}
	if !ok1 || !ok2 {
		return
	}
	step := geometry.Point{X: dx, Y: dy}
	org, dest := cc.Add(step), tt.Sub(step)
	if nxcor && !g.isStone(org.X, org.Y) {
		return
	}
	npoints, ok := digCorridor(g.random, g.width, g.height, org, dest, nxcor, g.isStone, g.isCorridor, func(x, y int) {
		g.m.SetCorridor(x, y)
		if g.maybeSDoor(100) {
			g.m.secretPassages[geometry.Point{X: x, Y: y}] = true
		}
	})
	if npoints > 0 && (g.okdoor(cc) || !nxcor) {
		g.dodoor(cc, croom)
	}
	if !ok {
		return
	}
	if g.okdoor(tt) || !nxcor {
		g.dodoor(tt, troom)
	}
	if croom.smeq < troom.smeq {
		troom.smeq = croom.smeq
	} else {
		croom.smeq = troom.smeq
	}
}

// makeCorridors is mklev.c makecorridors.
func (g *NetHackGenerator) makeCorridors() {
	n := len(g.rooms)
	for a := 0; a < n-1; a++ {
		g.join(a, a+1, false)
		if g.rn2(50) == 0 {
			break
		}
	}
	for a := 0; a < n-2; a++ {
		if g.rooms[a].smeq != g.rooms[a+2].smeq {
			g.join(a, a+2, false)
		}
	}
	for a, any := 0, true; any && a < n; a++ {
		any = false
		for b := 0; b < n; b++ {
			if g.rooms[a].smeq != g.rooms[b].smeq {
				g.join(a, b, false)
				any = true
			}
		}
	}
	if n > 2 {
		for i := g.rn2(n) + 4; i > 0; i-- {
			a := g.rn2(n)
			b := g.rn2(n - 2)
			if b >= a {
				b += 2
			}
			g.join(a, b, true)
		}
	}
}

// makeNiches is mklev.c make_niches without the trap niches: closets one tile deep above or below a room.
func (g *NetHackGenerator) makeNiches() {
	for ct := g.rn2(len(g.rooms)>>1+1) + 1; ct > 0; ct-- {
		g.makeNiche()
	}
}

func (g *NetHackGenerator) makeNiche() {
	for vct := 0; vct < 8; vct++ {
		room := g.rooms[g.rn2(len(g.rooms))]
		if room.doors == 1 && g.rn2(5) != 0 {
			continue
		}
		dy, side := 1, 2
		if g.rn2(2) == 0 {
			dy, side = -1, 0
		}
		door, ok := g.finddpos(side, room)
		niche := geometry.Point{X: door.X, Y: door.Y + dy}
		if !ok || !g.m.Contains(niche) || !g.isStone(niche.X, niche.Y) || g.m.GetTileAt(door.Add(geometry.Point{Y: -dy})) != Room {
			continue
		}
		if g.rn2(4) == 0 { // secret corridor behind a secret door
			g.m.SetCorridor(niche.X, niche.Y)
			g.m.secretPassages[niche] = true
			g.placeDoor(door, room, true)
		} else if g.rn2(7) != 0 {
			g.m.SetCorridor(niche.X, niche.Y)
			g.placeDoor(door, room, g.rn2(5) != 0)
		}
		return // the sealed closet with the scroll of teleportation is left out: rx1 has no digging
	}
}

// digCorridor is sp_lev.c dig_corridor: walk from org to dest, turning towards it, through back
// (rock) and fore (already dug) tiles, calling carve for every back tile. A blind corridor
// (nxcor) may stop early, so ok is false although npoints can be positive.
func digCorridor(random *rand.Rand, width, height int, org, dest geometry.Point, nxcor bool,
	isBack, isFore func(x, y int) bool, carve func(x, y int)) (npoints int, ok bool) {
	xx, yy, tx, ty := org.X, org.Y, dest.X, dest.Y
	if xx <= 0 || yy <= 0 || tx <= 0 || ty <= 0 || xx > width-1 || tx > width-1 || yy > height-1 || ty > height-1 {
		return 0, false
	}
	var dx, dy int
	switch {
	case tx > xx:
		dx = 1
	case ty > yy:
		dy = 1
	case tx < xx:
		dx = -1
	default:
		dy = -1
	}
	passable := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < width && y < height && (isBack(x, y) || isFore(x, y))
	}
	xx -= dx
	yy -= dy
	for cct := 0; xx != tx || yy != ty; cct++ {
		if cct > 500 || (nxcor && rn2(random, 35) == 0) {
			return npoints, false
		}
		xx += dx
		yy += dy
		if xx >= width-1 || xx <= 0 || yy <= 0 || yy >= height-1 {
			return npoints, false
		}
		if isBack(xx, yy) {
			npoints++
			carve(xx, yy)
		} else if !isFore(xx, yy) {
			return npoints, false
		}

		dix, diy := abs(xx-tx), abs(yy-ty)
		if dix > diy && diy != 0 && rn2(random, dix-diy+1) == 0 {
			dix = 0
		} else if diy > dix && dix != 0 && rn2(random, diy-dix+1) == 0 {
			diy = 0
		}
		if dy != 0 && dix > diy {
			ddx := sign(tx - xx)
			if passable(xx+ddx, yy) {
				dx, dy = ddx, 0
				continue
			}
		} else if dx != 0 && diy > dix {
			ddy := sign(ty - yy)
			if passable(xx, yy+ddy) {
				dx, dy = 0, ddy
				continue
			}
		}
		if passable(xx+dx, yy+dy) {
			continue
		}
		if dx != 0 {
			dx, dy = 0, sign(ty-yy)
		} else {
			dx, dy = sign(tx-xx), 0
		}
		if passable(xx+dx, yy+dy) {
			continue
		}
		dx, dy = -dx, -dy
	}
	return npoints, true
}
