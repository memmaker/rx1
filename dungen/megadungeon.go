package dungen

import (
	"math/rand"
	"rx1/geometry"
)

// based on https://journal.stuffwithstuff.com/2014/12/21/rooms-and-mazes/
// https://github.com/munificent/hauberk/blob/db360d9efa714efb6d937c31953ef849c7394a39/lib/src/content/dungeon.dart

// MegaDungeonGenerator packs the map with rooms, fills the space between them with winding corridors
// and then opens doors until everything is connected.
type MegaDungeonGenerator struct {
	randomSource           *rand.Rand
	roomTries              int
	minRoomSize            int
	straightPassageChance  float64
	imperfectConnectChance float64 // chance per connection for a second way in
	roomRatioInterval      float64 // rooms are up to 1+interval times longer than wide, or the other way round
	doorDistance           int     // an extra door keeps this distance to all other doors

	regionOf   []int          // per tile: the room or corridor system it belongs to, 0 for rock
	regionRoom []*DungeonRoom // per region: its room, nil for corridors
}

// connector is a wall tile between two regions: a door there connects them.
type connector struct {
	pos  geometry.Point
	a, b int
}

func NewMegaDungeonGenerator(source *rand.Rand) *MegaDungeonGenerator {
	return &MegaDungeonGenerator{
		randomSource:           source,
		roomTries:              source.Intn(500) + 300,
		minRoomSize:            source.Intn(4) + 2,
		straightPassageChance:  source.Float64(),
		imperfectConnectChance: source.Float64(),
		doorDistance:           source.Intn(6) + 2,
		roomRatioInterval:      source.Float64() * 0.5,
	}
}

func (c *MegaDungeonGenerator) Generate(width, height int) *DungeonMap {
	// rooms and corridors sit on odd coordinates, with a wall around the map
	if width%2 == 0 {
		width--
	}
	if height%2 == 0 {
		height--
	}
	m := NewDungeonMap(width, height)
	c.regionOf = make([]int, width*height)
	c.regionRoom = []*DungeonRoom{nil} // region 0 is rock

	c.addRooms(m)
	c.addCorridors(m)
	doors := c.connectRegions(m)
	for _, door := range doors {
		m.SetDoor(door.pos.X, door.pos.Y)
	}
	// corridors that lead nowhere go, and then the doors that only led to them
	for removed := true; removed; {
		m.FillDeadEnds(c.randomSource)
		removed = false
		for _, door := range doors {
			if m.IsDoorAt(door.pos) && len(m.GetFilteredCardinalNeighbours(door.pos, m.IsWalkable)) < 2 {
				m.SetWall(door.pos.X, door.pos.Y)
				removed = true
			}
		}
	}
	for _, door := range doors {
		if !m.IsDoorAt(door.pos) {
			continue
		}
		for _, region := range []int{door.a, door.b} {
			if room := c.regionRoom[region]; room != nil {
				room.SetDoor(door.pos) // the room has no wall here
			}
		}
	}
	return m
}

func (c *MegaDungeonGenerator) addRegion(m *DungeonMap, room *DungeonRoom, tiles []geometry.Point) {
	c.regionRoom = append(c.regionRoom, room)
	for _, pos := range tiles {
		c.regionOf[pos.X+pos.Y*m.width] = len(c.regionRoom) - 1
	}
}

func (c *MegaDungeonGenerator) addRooms(m *DungeonMap) {
	for i := 0; i < c.roomTries; i++ {
		roomSize := makeOddForSize(c.randomSource, c.minRoomSize, c.randomSource.Intn(5)+c.minRoomSize)
		aspectRatio := c.randomSource.Float64()*c.roomRatioInterval + 1
		roomWidth := roomSize
		roomHeight := roomSize
		if c.randomSource.Intn(2) == 0 {
			roomWidth = makeOddForSize(c.randomSource, c.minRoomSize, int(float64(roomWidth)*aspectRatio))
		} else {
			roomHeight = makeOddForSize(c.randomSource, c.minRoomSize, int(float64(roomHeight)*aspectRatio))
		}
		x := makeOdd(c.randomSource, c.randomSource.Intn(max(2, m.width-roomWidth-1))+1)
		y := makeOdd(c.randomSource, c.randomSource.Intn(max(2, m.height-roomHeight-1))+1)

		room := NewDungeonRoomFromRect(geometry.NewRect(x, y, x+roomWidth, y+roomHeight))
		if !m.CanPlaceRoomRestrictive(room) {
			continue
		}
		room.SetLit(c.randomSource.Intn(4) == 0)
		m.AddRoomAndSetTiles(room)
		c.addRegion(m, room, room.GetAbsoluteFloorTiles())
	}
}

// addCorridors grows a maze from every spot that is still solid rock. One pass is enough:
// rock that is not free now never becomes free again.
func (c *MegaDungeonGenerator) addCorridors(m *DungeonMap) {
	m.TraverseTilesRandomly(c.randomSource, func(pos geometry.Point) {
		if pos.X >= 1 && pos.Y >= 1 && pos.X < m.width-1 && pos.Y < m.height-1 && isCompletelyFree(m, pos) {
			c.addRegion(m, nil, fillMaze(m, pos, c.straightPassageChance, c.randomSource))
		}
	})
}

// connectRegions returns the doors that connect everything that can be connected to one random room.
// What cannot be reached from there is filled up again.
func (c *MegaDungeonGenerator) connectRegions(m *DungeonMap) []connector {
	if len(m.rooms) == 0 {
		return nil
	}
	// merged regions share a root
	root := make([]int, len(c.regionRoom))
	for i := range root {
		root[i] = i
	}
	find := func(region int) int {
		for root[region] != region {
			root[region] = root[root[region]]
			region = root[region]
		}
		return region
	}
	start := m.rooms[c.randomSource.Intn(len(m.rooms))].GetAbsoluteFloorTiles()[0]
	main := c.regionOf[start.X+start.Y*m.width]

	connectors := c.findConnectors(m)
	var doors []connector
	isDoor := make(map[geometry.Point]bool)
	addDoor := func(door connector) {
		doors = append(doors, door)
		isDoor[door.pos] = true
		root[find(door.a)], root[find(door.b)] = main, main
	}
	hasDoorNextTo := func(pos geometry.Point) bool {
		return isDoor[pos.Add(geometry.Point{X: 1})] || isDoor[pos.Add(geometry.Point{X: -1})] ||
			isDoor[pos.Add(geometry.Point{Y: 1})] || isDoor[pos.Add(geometry.Point{Y: -1})]
	}
	var candidates, apart []connector
	for {
		// the connectors from what is connected so far to something that is not
		candidates, apart = candidates[:0], apart[:0]
		for _, con := range connectors {
			if (find(con.a) == main) != (find(con.b) == main) {
				candidates = append(candidates, con)
				if !hasDoorNextTo(con.pos) {
					apart = append(apart, con)
				}
			}
		}
		if len(candidates) == 0 {
			break
		}
		if len(apart) > 0 { // no two doors side by side, if it can be helped
			candidates = apart
		}
		addDoor(candidates[c.randomSource.Intn(len(candidates))])
		if c.randomSource.Float64() < c.imperfectConnectChance {
			for tries := 0; tries < 5; tries++ {
				extra := candidates[c.randomSource.Intn(len(candidates))]
				if c.hasMinDistToDoors(doors, extra.pos) {
					addDoor(extra)
					break
				}
			}
		}
	}

	for i, region := range c.regionOf {
		if region != 0 && find(region) != main {
			m.tiles[i] = Wall
		}
	}
	connectedRooms := m.rooms[:0]
	for region, room := range c.regionRoom {
		if room != nil && find(region) == main {
			connectedRooms = append(connectedRooms, room)
		}
	}
	m.rooms = connectedRooms
	return doors
}

// findConnectors returns the wall tiles that have different regions on two opposite sides and wall on the other two.
func (c *MegaDungeonGenerator) findConnectors(m *DungeonMap) []connector {
	var connectors []connector
	side := func(pos geometry.Point) int { // the region there, 0 if a door must not lead there
		region := c.regionOf[pos.X+pos.Y*m.width]
		if room := c.regionRoom[region]; room != nil && room.IsCornerPosition(pos) {
			return 0
		}
		return region
	}
	for y := 1; y < m.height-1; y++ {
		for x := 1; x < m.width-1; x++ {
			pos := geometry.Point{X: x, Y: y}
			if c.regionOf[x+y*m.width] != 0 {
				continue
			}
			north, south := side(pos.Add(geometry.Point{Y: -1})), side(pos.Add(geometry.Point{Y: 1}))
			east, west := side(pos.Add(geometry.Point{X: 1})), side(pos.Add(geometry.Point{X: -1}))
			if north != 0 && south != 0 && north != south &&
				m.IsWallAt(pos.Add(geometry.Point{X: 1})) && m.IsWallAt(pos.Add(geometry.Point{X: -1})) {
				connectors = append(connectors, connector{pos, north, south})
			} else if east != 0 && west != 0 && east != west &&
				m.IsWallAt(pos.Add(geometry.Point{Y: 1})) && m.IsWallAt(pos.Add(geometry.Point{Y: -1})) {
				connectors = append(connectors, connector{pos, east, west})
			}
		}
	}
	return connectors
}

func (c *MegaDungeonGenerator) hasMinDistToDoors(doors []connector, pos geometry.Point) bool {
	for _, door := range doors {
		if geometry.DistanceManhattan(pos, door.pos) < c.doorDistance {
			return false
		}
	}
	return true
}

// fillMaze carves a winding corridor system into the rock around start and returns its tiles.
func fillMaze(m *DungeonMap, start geometry.Point, straightChance float64, rnd *rand.Rand) []geometry.Point {
	openList := []geometry.Point{start}
	carved := []geometry.Point{start}
	m.SetCorridor(start.X, start.Y)
	prevDirection := geometry.Point{X: 1, Y: 0}
	carve := func(from, to geometry.Point) bool {
		if !isCompletelyFreeForCarving(m, from, to) {
			return false
		}
		m.SetCorridor(to.X, to.Y)
		carved = append(carved, to)
		openList = append(openList, from, to)
		prevDirection = to.Sub(from)
		return true
	}
	for len(openList) > 0 {
		pos := openList[len(openList)-1]
		openList = openList[:len(openList)-1]

		if rnd.Float64() < straightChance && carve(pos, pos.Add(prevDirection)) {
			continue
		}
		neighbours := m.GetFilteredCardinalNeighbours(pos, func(p geometry.Point) bool {
			return p.X > 0 && p.Y > 0 && p.X < m.width-1 && p.Y < m.height-1
		})
		for _, randomIndex := range rnd.Perm(len(neighbours)) {
			if carve(pos, neighbours[randomIndex]) {
				break
			}
		}
	}
	return carved
}

func isCompletelyFree(m *DungeonMap, pos geometry.Point) bool {
	return m.IsWallAt(pos) && len(m.GetAllFilteredNeighbours(pos, m.IsWallAt)) == 8
}

// isCompletelyFreeForCarving: stepping from to keeps a wall to everything but from.
func isCompletelyFreeForCarving(m *DungeonMap, from geometry.Point, to geometry.Point) bool {
	direction := to.Sub(from)
	left := direction.RotateLeft()
	right := direction.RotateRight()
	for _, pos := range []geometry.Point{to, to.Add(direction), to.Add(left), to.Add(right), to.Add(left).Add(direction), to.Add(right).Add(direction)} {
		if !m.Contains(pos) || !m.IsWallAt(pos) {
			return false
		}
	}
	return true
}

func makeOddForSize(rnd *rand.Rand, minValue, value int) int {
	if value%2 == 0 {
		if value-1 < minValue || rnd.Intn(2) == 0 {
			value++
		} else {
			value--
		}
	}
	return value
}

func makeOdd(rnd *rand.Rand, value int) int {
	if value%2 == 0 {
		if rnd.Intn(2) == 0 {
			value++
		} else {
			value--
		}
	}
	return value
}
