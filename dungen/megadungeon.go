package dungen

import (
	"math/rand"
	"rx1/geometry"
	"slices"
)

// MegaDungeonGenerator is a port of the dungeon generator of Hauberk, see
// https://journal.stuffwithstuff.com/2014/12/21/rooms-and-mazes/
// https://github.com/munificent/hauberk/blob/db360d9efa714efb6d937c31953ef849c7394a39/lib/src/content/dungeon.dart
//
// Starting with solid rock, it works like so:
//
//  1. Place a number of randomly sized and positioned rooms. A room that overlaps or touches
//     an existing room is discarded.
//  2. Fill the remaining solid areas with mazes.
//  3. Find every "connector": a solid tile that is adjacent to two unconnected regions.
//  4. Open random connectors until all regions are joined. Now and then a connector between two
//     regions that are joined already is opened too, so that the dungeon is not singly connected.
//  5. Remove the dead ends by filling in every open tile that is closed on three sides.
type MegaDungeonGenerator struct {
	randomSource *rand.Rand
	NumRoomTries int
	// ExtraConnectorChance is the inverse chance of adding a connector between two regions that have
	// already been joined. Increasing this leads to more loosely connected dungeons.
	ExtraConnectorChance int
	RoomExtraSize        int // increasing this allows rooms to be larger
	WindingPercent       int

	m *DungeonMap
	// regions holds, for each open position, the index of the connected region it is part of, -1 for rock
	regions       []int
	currentRegion int            // the index of the current region being carved
	regionRoom    []*DungeonRoom // per region: its room, nil for a maze
}

var cardinalDirections = []geometry.Point{{Y: -1}, {X: 1}, {Y: 1}, {X: -1}}

// NewMegaDungeonGenerator has the defaults of the reference, which leaves the number of room tries open.
func NewMegaDungeonGenerator(source *rand.Rand, numRoomTries int) *MegaDungeonGenerator {
	return &MegaDungeonGenerator{randomSource: source, NumRoomTries: numRoomTries, ExtraConnectorChance: 20}
}

// Generate needs an odd size, an even one is made one smaller.
func (c *MegaDungeonGenerator) Generate(width, height int) *DungeonMap {
	if width%2 == 0 {
		width--
	}
	if height%2 == 0 {
		height--
	}
	c.m = NewDungeonMap(width, height)
	c.regions = make([]int, width*height)
	for i := range c.regions {
		c.regions[i] = -1
	}
	c.currentRegion = -1
	c.regionRoom = nil

	c.addRooms()

	// Fill in all of the empty space with mazes.
	for y := 1; y < height; y += 2 {
		for x := 1; x < width; x += 2 {
			if c.m.GetTile(x, y) == Wall {
				c.growMaze(geometry.Point{X: x, Y: y})
			}
		}
	}

	junctions := c.connectRegions()
	c.removeDeadEnds()
	c.describeRooms(junctions)
	return c.m
}

// describeRooms is not part of the reference: it tells the rooms what the game wants to know about them.
func (c *MegaDungeonGenerator) describeRooms(junctions []geometry.Point) {
	for _, junction := range junctions {
		if c.m.IsWallAt(junction) {
			continue
		}
		for _, dir := range cardinalDirections {
			if region := c.regionAt(junction.Add(dir)); region >= 0 && c.regionRoom[region] != nil {
				c.regionRoom[region].SetDoor(junction)
			}
		}
	}
	for _, room := range c.m.rooms {
		// a junction can also open the corner of a room's wall, from one corridor to another
		room.wallTiles = slices.DeleteFunc(room.wallTiles, func(pos geometry.Point) bool { return !c.m.IsWallAt(pos) })
		room.SetLit(c.randomSource.Intn(4) == 0)
	}
}

func (c *MegaDungeonGenerator) regionAt(pos geometry.Point) int {
	return c.regions[pos.X+pos.Y*c.m.width]
}

// growMaze is the "growing tree" algorithm from http://www.astrolog.org/labyrnth/algrithm.htm
func (c *MegaDungeonGenerator) growMaze(start geometry.Point) {
	cells := []geometry.Point{start}
	var lastDir geometry.Point

	c.startRegion(nil)
	c.carve(start, Corridor)

	for len(cells) > 0 {
		cell := cells[len(cells)-1]

		// See which adjacent cells are open.
		var unmadeCells []geometry.Point
		for _, dir := range cardinalDirections {
			if c.canCarve(cell, dir) {
				unmadeCells = append(unmadeCells, dir)
			}
		}

		if len(unmadeCells) == 0 {
			// No adjacent uncarved cells, this path has ended.
			cells = cells[:len(cells)-1]
			lastDir = geometry.Point{}
			continue
		}
		// Based on how "windy" passages are, try to prefer carving in the same direction.
		dir := lastDir
		if !slices.Contains(unmadeCells, lastDir) || c.randomSource.Intn(100) <= c.WindingPercent {
			dir = unmadeCells[c.randomSource.Intn(len(unmadeCells))]
		}
		c.carve(cell.Add(dir), Corridor)
		c.carve(cell.Add(dir).Add(dir), Corridor)

		cells = append(cells, cell.Add(dir).Add(dir))
		lastDir = dir
	}
}

// addRooms places rooms ignoring the existing maze corridors.
func (c *MegaDungeonGenerator) addRooms() {
	var rooms []geometry.Rect
	for i := 0; i < c.NumRoomTries; i++ {
		// Pick a random room size. The funny math here does two things:
		// - It makes sure rooms are odd-sized to line up with maze.
		// - It avoids creating rooms that are too rectangular: too tall and narrow or too wide and flat.
		size := (1+c.randomSource.Intn(2+c.RoomExtraSize))*2 + 1
		rectangularity := c.randomSource.Intn(1+size/2) * 2
		width, height := size, size
		if c.randomSource.Intn(2) == 0 {
			width += rectangularity
		} else {
			height += rectangularity
		}
		if c.m.width-width < 2 || c.m.height-height < 2 {
			continue // not in the reference: the map is too small for this room
		}

		x := c.randomSource.Intn((c.m.width-width)/2)*2 + 1
		y := c.randomSource.Intn((c.m.height-height)/2)*2 + 1

		room := geometry.NewRect(x, y, x+width, y+height)
		if slices.ContainsFunc(rooms, func(other geometry.Rect) bool { return rectDistance(room, other) <= 0 }) {
			continue
		}
		rooms = append(rooms, room)

		dungeonRoom := NewDungeonRoomFromRect(room)
		c.m.rooms = append(c.m.rooms, dungeonRoom)
		c.startRegion(dungeonRoom)
		for ry := y; ry < y+height; ry++ {
			for rx := x; rx < x+width; rx++ {
				c.carve(geometry.Point{X: rx, Y: ry}, Room)
			}
		}
	}
}

// rectDistance is the minimum length that a corridor would have to be to go from one rect to the other.
// If the two are adjacent, it is zero. If they overlap, it is -1.
func rectDistance(a, b geometry.Rect) int {
	gap := func(aMin, aMax, bMin, bMax int) int {
		if aMin >= bMax {
			return aMin - bMax
		}
		if aMax <= bMin {
			return bMin - aMax
		}
		return -1
	}
	vertical, horizontal := gap(a.Min.Y, a.Max.Y, b.Min.Y, b.Max.Y), gap(a.Min.X, a.Max.X, b.Min.X, b.Max.X)
	if vertical == -1 {
		return horizontal
	}
	if horizontal == -1 {
		return vertical
	}
	return horizontal + vertical
}

// connectRegions returns the junctions it opened.
func (c *MegaDungeonGenerator) connectRegions() []geometry.Point {
	// Find all of the tiles that can connect two (or more) regions.
	type connector struct {
		pos     geometry.Point
		regions []int
	}
	var connectors []connector
	for y := 1; y < c.m.height-1; y++ {
		for x := 1; x < c.m.width-1; x++ {
			pos := geometry.Point{X: x, Y: y}
			// Can't already be part of a region.
			if c.m.GetTile(x, y) != Wall {
				continue
			}
			var regions []int
			for _, dir := range cardinalDirections {
				if region := c.regionAt(pos.Add(dir)); region >= 0 && !slices.Contains(regions, region) {
					regions = append(regions, region)
				}
			}
			if len(regions) >= 2 {
				connectors = append(connectors, connector{pos, regions})
			}
		}
	}

	// Keep track of which regions have been merged. This maps an original
	// region index to the one it has been merged to.
	merged := make([]int, c.currentRegion+1)
	for i := range merged {
		merged[i] = i
	}
	openRegions := len(merged)
	// the regions a connector joins by now
	mergedRegions := func(con connector) []int {
		var regions []int
		for _, region := range con.regions {
			if !slices.Contains(regions, merged[region]) {
				regions = append(regions, merged[region])
			}
		}
		return regions
	}

	var junctions []geometry.Point
	addJunction := func(pos geometry.Point) {
		// the reference places closed doors, open doors and bare openings; this game knows one kind of door
		// extra connectors are never cleared from around a junction, so one can land next to a door: open it bare
		nextToDoor := slices.ContainsFunc(junctions, func(j geometry.Point) bool {
			d := j.Sub(pos)
			return d.X*d.X+d.Y*d.Y < 4 && c.m.IsDoorAt(j)
		})
		if nextToDoor || c.randomSource.Intn(4) == 0 && c.randomSource.Intn(3) != 0 {
			c.m.SetCorridor(pos.X, pos.Y)
		} else {
			c.m.SetDoor(pos.X, pos.Y)
		}
		junctions = append(junctions, pos)
	}

	// Keep connecting regions until we're down to one.
	// Not in the reference, which fails then: stop when no connector is left.
	for openRegions > 1 && len(connectors) > 0 {
		chosen := connectors[c.randomSource.Intn(len(connectors))]

		// Carve the connection.
		addJunction(chosen.pos)

		// Merge the connected regions. We'll pick one region (arbitrarily) and
		// map all of the other regions to its index.
		// The reference strikes the first region from the open ones too if the connector touches it twice,
		// and then stops before everything is joined; here every region counts once.
		regions := mergedRegions(chosen)
		dest, sources := regions[0], regions[1:]

		// Merge all of the affected regions. We have to look at *all* of the
		// regions because other regions may have previously been merged with
		// some of the ones we're merging now.
		for i := range merged {
			if slices.Contains(sources, merged[i]) {
				merged[i] = dest
			}
		}

		// The sources are no longer in use.
		openRegions -= len(sources)

		// Remove any connectors that aren't needed anymore.
		connectors = slices.DeleteFunc(connectors, func(con connector) bool {
			// Don't allow connectors right next to each other.
			if d := chosen.pos.Sub(con.pos); d.X*d.X+d.Y*d.Y < 4 {
				return true
			}
			// If the connector no longer spans different regions, we don't need it.
			if len(mergedRegions(con)) > 1 {
				return false
			}
			// This connector isn't needed, but connect it occasionally so that the
			// dungeon isn't singly-connected.
			if c.randomSource.Intn(c.ExtraConnectorChance) == 0 {
				addJunction(con.pos)
			}
			return true
		})
	}

	// Not in the reference: what could not be joined to the first region is filled up again.
	if openRegions > 1 {
		for i, region := range c.regions {
			if region >= 0 && merged[region] != merged[0] {
				c.m.tiles[i] = Wall
			}
		}
		c.m.rooms = slices.DeleteFunc(c.m.rooms, func(room *DungeonRoom) bool {
			return c.m.IsWallAt(room.bounds.Min)
		})
	}
	return junctions
}

func (c *MegaDungeonGenerator) removeDeadEnds() {
	for done := false; !done; {
		done = true
		for y := 1; y < c.m.height-1; y++ {
			for x := 1; x < c.m.width-1; x++ {
				if c.m.GetTile(x, y) == Wall {
					continue
				}
				// If it only has one exit, it's a dead end.
				exits := 0
				for _, dir := range cardinalDirections {
					if c.m.GetTile(x+dir.X, y+dir.Y) != Wall {
						exits++
					}
				}
				if exits != 1 {
					continue
				}
				done = false
				c.m.SetWall(x, y)
			}
		}
	}
}

// canCarve gets whether or not an opening can be carved from the given starting cell at pos
// to the adjacent cell facing direction: the destination must be in bounds and not open yet.
func (c *MegaDungeonGenerator) canCarve(pos, direction geometry.Point) bool {
	// Must end in bounds.
	if !c.m.Contains(geometry.Point{X: pos.X + direction.X*3, Y: pos.Y + direction.Y*3}) {
		return false
	}
	// Destination must not be open.
	return c.m.GetTile(pos.X+direction.X*2, pos.Y+direction.Y*2) == Wall
}

func (c *MegaDungeonGenerator) startRegion(room *DungeonRoom) {
	c.currentRegion++
	c.regionRoom = append(c.regionRoom, room)
}

func (c *MegaDungeonGenerator) carve(pos geometry.Point, tile DungeonTile) {
	c.m.tiles[pos.X+pos.Y*c.m.width] = tile
	c.regions[pos.X+pos.Y*c.m.width] = c.currentRegion
}
