package gridmap

import (
	"fmt"
	"math"
	"math/rand"
	"rx1/geometry"
	"rx1/util"
	"sort"
)

type MapObject interface {
	Position() geometry.Point
	SetPosition(geometry.Point)
}

type MapActor interface {
	MapObject
}
type MapObjectWithProperties[ActorType interface {
	comparable
	MapActor
}] interface {
	MapObject
	IsWalkable(person ActorType) bool
	IsTransparent() bool
	IsPassableForProjectile() bool
}
type GridMap[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}] struct {
	cells         []MapCell[ActorType, ItemType, ObjectType]
	allActors     []ActorType
	actorListener func(actor ActorType, onMap bool) // told about every actor placed, moved or removed
	allItems      []ItemType
	allObjects    []ObjectType

	mapWidth  int
	mapHeight int

	pathfinder *geometry.PathRange

	cardinalMovementOnly bool
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetCardinalMovementOnly(cardinalMovementOnly bool) {
	m.cardinalMovementOnly = cardinalMovementOnly
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetTile(position geometry.Point, mapTile Tile) {
	if !m.Contains(position) {
		return
	}
	index := position.Y*m.mapWidth + position.X
	m.cells[index].TileType = mapTile
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveItemAt(position geometry.Point) {
	m.RemoveItem(m.ItemAt(position))
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveObject(obj ObjectType) {
	m.cells[obj.Position().Y*m.mapWidth+obj.Position().X] = m.cells[obj.Position().Y*m.mapWidth+obj.Position().X].WithObjectHereRemoved(obj)
	for i := len(m.allObjects) - 1; i >= 0; i-- {
		if m.allObjects[i] == obj {
			m.allObjects = append(m.allObjects[:i], m.allObjects[i+1:]...)
			return
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) CellAt(location geometry.Point) MapCell[ActorType, ItemType, ObjectType] {
	return m.cells[m.mapWidth*location.Y+location.X]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ItemAt(location geometry.Point) ItemType {
	return *m.cells[m.mapWidth*location.Y+location.X].Item
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsItemAt(location geometry.Point) bool {
	return m.cells[m.mapWidth*location.Y+location.X].Item != nil
}

// SetActorListener sets the one function that hears about every actor placed, moved (onMap) or removed (!onMap),
// after the map has changed. The display follows actors with it.
func (m *GridMap[ActorType, ItemType, ObjectType]) SetActorListener(listener func(actor ActorType, onMap bool)) {
	m.actorListener = listener
}

func (m *GridMap[ActorType, ItemType, ObjectType]) actorChanged(actor ActorType, onMap bool) {
	if m.actorListener != nil {
		m.actorListener(actor, onMap)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveItem(item ItemType, to geometry.Point) {
	m.cells[item.Position().Y*m.mapWidth+item.Position().X] = m.cells[item.Position().Y*m.mapWidth+item.Position().X].WithItemHereRemoved(item)
	item.SetPosition(to)
	m.cells[to.Y*m.mapWidth+to.X] = m.cells[to.Y*m.mapWidth+to.X].WithItem(item)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetRandomFreeAndSafeNeighbor(source *rand.Rand, location geometry.Point) geometry.Point {
	freeNeighbors := m.GetFilteredNeighbors(location, func(p geometry.Point) bool {
		return m.Contains(p) && m.IsCurrentlyPassable(p) && !m.IsObviousHazardAt(p)
	})
	if len(freeNeighbors) == 0 {
		return location
	}
	return freeNeighbors[source.Intn(len(freeNeighbors))]
}

type SetOfPoints map[geometry.Point]bool

func (s *SetOfPoints) Pop() geometry.Point {
	for k := range *s {
		delete(*s, k)
		return k
	}
	return geometry.Point{}
}
func (s *SetOfPoints) Contains(p geometry.Point) bool {
	_, ok := (*s)[p]
	return ok
}

func (s *SetOfPoints) ToSlice() []geometry.Point {
	result := make([]geometry.Point, 0)
	for k := range *s {
		result = append(result, k)
	}
	return result
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFreeCellsForDistribution(position geometry.Point, neededCellCount int, freePredicate func(p geometry.Point) bool) []geometry.Point {
	foundFreeCells := make(SetOfPoints)
	currentPosition := position
	openList := make(SetOfPoints)
	closedList := make(SetOfPoints)
	closedList[currentPosition] = true

	for _, neighbor := range m.GetFilteredNeighbors(currentPosition, m.IsTileWalkable) {
		openList[neighbor] = true
	}
	for len(foundFreeCells) < neededCellCount && len(openList) > 0 {
		freeNeighbors := m.GetFilteredNeighbors(currentPosition, freePredicate)
		for _, neighbor := range freeNeighbors {
			foundFreeCells[neighbor] = true
		}
		// pop from open list
		pop := openList.Pop()
		currentPosition = pop
		for _, neighbor := range m.GetFilteredNeighbors(currentPosition, m.IsTileWalkable) {
			if !closedList.Contains(neighbor) {
				openList[neighbor] = true
			}
		}
		closedList[currentPosition] = true
	}

	freeCells := foundFreeCells.ToSlice()
	sort.Slice(freeCells, func(i, j int) bool {
		return geometry.DistanceSquared(freeCells[i], position) < geometry.DistanceSquared(freeCells[j], position)
	})
	return freeCells
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveItem(item ItemType) {
	m.cells[item.Position().Y*m.mapWidth+item.Position().X] = m.cells[item.Position().Y*m.mapWidth+item.Position().X].WithItemHereRemoved(item)
	for i := len(m.allItems) - 1; i >= 0; i-- {
		if m.allItems[i] == item {
			m.allItems = append(m.allItems[:i], m.allItems[i+1:]...)
			return
		}
	}
}

// WavePropagationFrom spreads from pos through everything walkable, around corners but not through walls.
// waves[d] holds the tiles that are d steps away.
func (m *GridMap[ActorType, ItemType, ObjectType]) WavePropagationFrom(pos geometry.Point, steps int) [][]geometry.Point {
	waves := make([][]geometry.Point, steps+1)
	for _, node := range m.pathfinder.DijkstraMap(m.getDijkstraMapperWithActorsNotBlocking(), []geometry.Point{pos}, steps*10+9) {
		waves[node.Cost/10] = append(waves[node.Cost/10], node.P)
	}
	return waves
}

type DijkstraMapper struct {
	neighbors func(geometry.Point) []geometry.Point
	cost      func(geometry.Point, geometry.Point) int
}

func (d DijkstraMapper) Neighbors(point geometry.Point) []geometry.Point {
	return d.neighbors(point)
}

func (d DijkstraMapper) Cost(point geometry.Point, point2 geometry.Point) int {
	return d.cost(point, point2)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) getDijkstraMapperWithActorsNotBlocking() DijkstraMapper {
	return DijkstraMapper{
		neighbors: func(point geometry.Point) []geometry.Point {
			return m.GetFilteredNeighbors(point, func(p geometry.Point) bool {
				return m.Contains(p) && m.IsWalkable(p) && m.DiagonalOK(point, p)
			})
		},
		cost: func(point geometry.Point, point2 geometry.Point) int {
			return int(geometry.Distance(point, point2) * 10)
		},
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) getDijkstraMapper(passable func(pos geometry.Point) bool) DijkstraMapper {
	return DijkstraMapper{
		neighbors: func(point geometry.Point) []geometry.Point {
			return m.GetFilteredNeighbors(point, func(p geometry.Point) bool {
				return m.Contains(p) && passable(p) && m.DiagonalOK(point, p)
			})
		},
		cost: func(point geometry.Point, point2 geometry.Point) int {
			return int(geometry.Distance(point, point2) * 10)
		},
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetDijkstraMapWithActorsNotBlocking(start geometry.Point, maxCost int) map[geometry.Point]int {
	nodes := m.pathfinder.DijkstraMap(m.getDijkstraMapperWithActorsNotBlocking(), []geometry.Point{start}, maxCost*10)

	dijkstraMap := make(map[geometry.Point]int)
	for _, v := range nodes {
		dijkstraMap[v.P] = v.Cost
	}
	return dijkstraMap
}
func (m *GridMap[ActorType, ItemType, ObjectType]) GetDijkstraMap(start geometry.Point, maxCost int, passable func(point geometry.Point) bool) map[geometry.Point]int {
	nodes := m.pathfinder.DijkstraMap(m.getDijkstraMapper(passable), []geometry.Point{start}, maxCost*10)

	dijkstraMap := make(map[geometry.Point]int)
	for _, v := range nodes {
		dijkstraMap[v.P] = v.Cost
	}
	return dijkstraMap
}
func NewMapFromString[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}](width, height int, inputString string, mapper func(gridMap *GridMap[ActorType, ItemType, ObjectType], icon rune, pos geometry.Point)) *GridMap[ActorType, ItemType, ObjectType] {
	emptyMap := NewEmptyMap[ActorType, ItemType, ObjectType](width, height)
	size := width * height
	if len(inputString) < size {
		return emptyMap
	}
	for i := 0; i < size; i++ {
		icon := rune(inputString[i])
		mapper(emptyMap, icon, geometry.Point{X: i % width, Y: i / width})
	}
	return emptyMap
}

// update for entities:
// call update for every updatable entity (genMap, AllItems, AllObjects, tiles)
// default: just return
// entities have an internal schedule, waiting for ticks to happen

func NewEmptyMap[ActorType interface {
	comparable
	MapActor
}, ItemType interface {
	comparable
	MapObject
}, ObjectType interface {
	comparable
	MapObjectWithProperties[ActorType]
}](width, height int) *GridMap[ActorType, ItemType, ObjectType] {
	pathRange := geometry.NewPathRange(geometry.NewRect(0, 0, width, height))
	m := &GridMap[ActorType, ItemType, ObjectType]{
		cells:      make([]MapCell[ActorType, ItemType, ObjectType], width*height),
		allActors:  make([]ActorType, 0),
		allItems:   make([]ItemType, 0),
		allObjects: make([]ObjectType, 0),
		mapWidth:   width,
		mapHeight:  height,
		pathfinder: pathRange,
	}
	return m
}

func (m *GridMap[ActorType, ItemType, ObjectType]) FillTile(tile Tile) {
	for i := range m.cells {
		m.cells[i].TileType = tile
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetCell(p geometry.Point) MapCell[ActorType, ItemType, ObjectType] {
	return m.cells[p.X+p.Y*m.mapWidth]
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RemoveActor(actor ActorType) bool {
	m.cells[actor.Position().X+actor.Position().Y*m.mapWidth] = m.cells[actor.Position().X+actor.Position().Y*m.mapWidth].WithActorHereRemoved(actor)
	m.actorChanged(actor, false)
	for i := len(m.allActors) - 1; i >= 0; i-- {
		if m.allActors[i] == actor {
			m.allActors = append(m.allActors[:i], m.allActors[i+1:]...)
			return true
		}
	}
	return false
}

// MoveActor Should only be called my the model, so we can ensure that a HUD IsDone will follow
func (m *GridMap[ActorType, ItemType, ObjectType]) MoveActor(actor ActorType, newPos geometry.Point) {
	m.MoveActorFrom(actor, actor.Position(), newPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveActorFrom(actor ActorType, from, to geometry.Point) {
	if !m.Contains(to) {
		return
	}
	if !m.IsWalkableFor(to, actor) {
		return
	}
	if m.Contains(from) {
		m.cells[from.X+from.Y*m.mapWidth] = m.cells[from.X+from.Y*m.mapWidth].WithActorHereRemoved(actor)
	}
	actor.SetPosition(to)
	m.cells[to.X+to.Y*m.mapWidth] = m.cells[to.X+to.Y*m.mapWidth].WithActor(actor)
	m.actorChanged(actor, true)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveObject(obj ObjectType, newPos geometry.Point) {
	m.cells[obj.Position().X+obj.Position().Y*m.mapWidth] = m.cells[obj.Position().X+obj.Position().Y*m.mapWidth].WithObjectHereRemoved(obj)
	obj.SetPosition(newPos)
	m.cells[newPos.X+newPos.Y*m.mapWidth] = m.cells[newPos.X+newPos.Y*m.mapWidth].WithObject(obj)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Fill(mapCell MapCell[ActorType, ItemType, ObjectType]) {
	for i := range m.cells {
		m.cells[i] = mapCell
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTransparent(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}

	if objectAt, ok := m.TryGetObjectAt(p); ok && !objectAt.IsTransparent() {
		return false
	}

	return m.GetCell(p).TileType.IsTransparent
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTileWalkable(point geometry.Point) bool {
	if !m.Contains(point) {
		return false
	}
	return m.GetCell(point).TileType.IsWalkable
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Contains(dest geometry.Point) bool {
	return dest.X >= 0 && dest.X < m.mapWidth && dest.Y >= 0 && dest.Y < m.mapHeight
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsActorAt(location geometry.Point) bool {
	if !m.Contains(location) {
		return false
	}
	return m.cells[location.X+location.Y*m.mapWidth].Actor != nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ActorAt(location geometry.Point) ActorType {
	return *m.cells[location.X+location.Y*m.mapWidth].Actor
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsObjectAt(location geometry.Point) bool {
	return m.cells[location.X+location.Y*m.mapWidth].Object != nil
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ObjectAt(location geometry.Point) ObjectType {
	return *m.cells[location.X+location.Y*m.mapWidth].Object
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Actors() []ActorType {
	return m.allActors
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Items() []ItemType {
	return m.allItems
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Objects() []ObjectType {
	return m.allObjects
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFilteredNeighbors(pos geometry.Point, filter func(geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	filtered := neighbors.All(pos, filter)
	return filtered
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFilteredNeighborsForMovement(pos geometry.Point, filter func(geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	var result []geometry.Point
	if m.cardinalMovementOnly {
		result = neighbors.Cardinal(pos, filter)
	} else {
		result = neighbors.All(pos, func(p geometry.Point) bool { return filter(p) && m.DiagonalOK(pos, p) })
	}
	return result
}

// DiagonalOK is Rogue's diag_ok: no diagonal step or attack around a corner or through a doorway.
func (m *GridMap[ActorType, ItemType, ObjectType]) DiagonalOK(from, to geometry.Point) bool {
	if from.X == to.X || from.Y == to.Y {
		return true
	}
	if !m.IsTileWalkable(geometry.Point{X: from.X, Y: to.Y}) || !m.IsTileWalkable(geometry.Point{X: to.X, Y: from.Y}) {
		return false
	}
	return !m.GetCell(from).TileType.IsDoor() && !m.GetCell(to).TileType.IsDoor()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) displaceActor(a ActorType, position geometry.Point) {
	free := m.GetFreeCellsForDistribution(position, 1, func(p geometry.Point) bool {
		return m.CanPlaceActorHere(p)
	})
	if len(free) == 0 {
		return
	}
	freePos := free[0]
	m.MoveActor(a, freePos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddItemWithDisplacement(a ItemType, targetPos geometry.Point) {
	if !m.Contains(targetPos) {
		return
	}
	if m.CanPlaceItemHere(targetPos) {
		m.AddItem(a, targetPos)
		return
	}
	free := m.GetFreeCellsForDistribution(targetPos, 1, func(p geometry.Point) bool {
		return m.CanPlaceItemHere(p)
	})
	if len(free) == 0 {
		println("WARNING: Could not find a free spot for item")
		return
	}
	freePos := free[0]
	m.AddItem(a, freePos)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) GetJPSPath(start geometry.Point, end geometry.Point, isWalkable func(geometry.Point) bool) []geometry.Point {
	if !isWalkable(end) {
		end = m.getNearestFreeNeighbor(start, end, isWalkable)
	}
	return m.pathfinder.JPSPath([]geometry.Point{}, start, end, isWalkable, !m.cardinalMovementOnly)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) getNearestFreeNeighbor(origin, pos geometry.Point, isFree func(geometry.Point) bool) geometry.Point {
	dist := math.MaxInt32
	nearest := pos
	for _, neighbor := range m.NeighborsCardinal(pos, isFree) {
		d := geometry.DistanceManhattan(origin, neighbor)
		if d < dist {
			dist = d
			nearest = neighbor
		}
	}
	return nearest
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsCurrentlyPassable(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}
	return m.IsWalkable(p) && (!m.IsActorAt(p)) //&& !knownAsBlocked
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsWalkable(p geometry.Point) bool {
	if !m.Contains(p) {
		return false
	}
	var noActor ActorType
	if m.IsObjectAt(p) && (!m.ObjectAt(p).IsWalkable(noActor)) {
		return false
	}
	cellAt := m.GetCell(p)
	return cellAt.TileType.IsWalkable
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsObviousHazardAt(p geometry.Point) bool {
	return m.IsDamagingTileAt(p)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) IsWalkableFor(p geometry.Point, person ActorType) bool {
	if !m.Contains(p) {
		return false
	}

	if m.IsActorAt(p) && m.ActorAt(p) != person {
		return false
	}

	if m.IsObjectAt(p) && (!m.ObjectAt(p).IsWalkable(person)) {
		return false
	}

	cellAt := m.GetCell(p)
	return cellAt.TileType.IsWalkable

}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsExplored(pos geometry.Point) bool {
	return m.GetCell(pos).IsExplored
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetExplored(pos geometry.Point) {
	if !m.Contains(pos) {
		return
	}
	m.cells[pos.X+pos.Y*m.mapWidth].IsExplored = true
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetLit(pos geometry.Point, value bool) {
	if !m.Contains(pos) {
		return
	}
	m.cells[pos.X+pos.Y*m.mapWidth].IsLit = value
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Apply(f func(cell MapCell[ActorType, ItemType, ObjectType]) MapCell[ActorType, ItemType, ObjectType]) {
	for i, cell := range m.cells {
		m.cells[i] = f(cell)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MapSize() geometry.Point {
	return geometry.Point{X: m.mapWidth, Y: m.mapHeight}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) NeighborsAll(pos geometry.Point, filter func(p geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.All(pos, filter)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) NeighborsCardinal(pos geometry.Point, filter func(p geometry.Point) bool) []geometry.Point {
	neighbors := geometry.Neighbors{}
	return neighbors.Cardinal(pos, filter)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsPassableForProjectile(p geometry.Point) bool {
	isTileWalkable := m.IsTileWalkable(p)
	isActorOnTile := m.IsActorAt(p)
	isObjectOnTile := m.IsObjectAt(p)
	isObjectBlocking := false
	if isObjectOnTile {
		objectOnTile := m.ObjectAt(p)
		isObjectBlocking = !objectOnTile.IsPassableForProjectile()
	}
	return isTileWalkable && !isActorOnTile && !isObjectBlocking
}

// BresenhamLine returns a list of points that are on the line between source and destination.
// NOTE: Will remove the source point from the list
func (m *GridMap[ActorType, ItemType, ObjectType]) BresenhamLine(source geometry.Point, destination geometry.Point, isBlocking func(mapPos geometry.Point) bool) []geometry.Point {
	los := geometry.BresenhamLine(source, destination, func(x, y int) bool {
		p := geometry.Point{X: x, Y: y}
		if !m.Contains(p) {
			return false
		}
		if p == source {
			return true
		}
		if isBlocking == nil && isBlocking(p) {
			return false
		}
		return m.IsWalkable(p)
	})
	withoutSource := los[1:]
	return withoutSource
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetAllExplored() {
	for y := 0; y < m.mapHeight; y++ {
		for x := 0; x < m.mapWidth; x++ {
			m.cells[y*m.mapWidth+x].IsExplored = true
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RandomSpawnPosition() geometry.Point {
	for {
		x := rand.Intn(m.mapWidth)
		y := rand.Intn(m.mapHeight)
		pos := geometry.Point{X: x, Y: y}
		if m.IsEmptyNonSpecialFloor(pos) {
			return pos
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsDamagingTileAt(p geometry.Point) bool {
	tileType := m.CellAt(p).TileType
	return tileType.IsChasm() || tileType.IsLava()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetActorAt(pos geometry.Point) (ActorType, bool) {
	var noActor ActorType
	isActorAt := m.IsActorAt(pos)
	if !isActorAt {
		return noActor, false
	}
	return m.ActorAt(pos), isActorAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetObjectAt(pos geometry.Point) (ObjectType, bool) {
	var noObject ObjectType
	isObjectAt := m.IsObjectAt(pos)
	if !isObjectAt {
		return noObject, false
	}
	return m.ObjectAt(pos), isObjectAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) TryGetItemAt(pos geometry.Point) (ItemType, bool) {
	var noItem ItemType
	isItemAt := m.IsItemAt(pos)
	if !isItemAt {
		return noItem, false
	}
	return m.ItemAt(pos), isItemAt
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddActor(actor ActorType, spawnPos geometry.Point) {
	if m.IsActorAt(spawnPos) {
		return
	}
	m.allActors = append(m.allActors, actor)
	m.MoveActor(actor, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddObject(object ObjectType, spawnPos geometry.Point) {
	if m.IsObjectAt(spawnPos) {
		return
	}
	m.allObjects = append(m.allObjects, object)
	m.MoveObject(object, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddItem(item ItemType, spawnPos geometry.Point) {
	if m.IsItemAt(spawnPos) {
		return
	}
	m.allItems = append(m.allItems, item)
	m.MoveItem(item, spawnPos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) UpdateFieldOfView(fov *geometry.FOV, fovPosition geometry.Point, visionRange int) {
	visionRangeSquared := visionRange * visionRange
	var fovRange = geometry.NewRect(-visionRange, -visionRange, visionRange+1, visionRange+1)
	fov.SetRange(fovRange.Add(fovPosition).Intersect(geometry.NewRect(0, 0, m.mapWidth, m.mapHeight)))

	fov.SSCVisionMap(fovPosition, visionRange, false, func(p geometry.Point) bool {
		if !m.Contains(p) {
			return false
		}
		return m.IsTransparent(p) && geometry.DistanceSquared(p, fovPosition) <= visionRangeSquared
	})
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFilteredActorsInRadius(location geometry.Point, radius int, filter func(actor ActorType) bool) []ActorType {
	return m.iterateActors(location, radius, filter)
}
func (m *GridMap[ActorType, ItemType, ObjectType]) iterateActors(location geometry.Point, radius int, filter func(actor ActorType) bool) []ActorType {
	result := make([]ActorType, 0)
	for _, actor := range m.allActors {
		if geometry.Distance(location, actor.Position()) <= float64(radius) && filter(actor) {
			result = append(result, actor)
		}
	}
	return result
}

func (m *GridMap[ActorType, ItemType, ObjectType]) Print() {
	// walls and floors only
	for y := 0; y < m.mapHeight; y++ {
		for x := 0; x < m.mapWidth; x++ {
			pos := geometry.Point{X: x, Y: y}
			if m.IsTileWalkable(pos) {
				fmt.Printf(".")
			} else {
				fmt.Printf("#")
			}
		}
		fmt.Printf("\n")
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) CanPlaceActorHere(pos geometry.Point) bool {
	return m.IsWalkable(pos) && !m.IsActorAt(pos) && !m.IsObviousHazardAt(pos) && !m.IsTileSpecial(pos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) CanPlaceItemHere(pos geometry.Point) bool {
	return m.IsWalkable(pos) && !m.IsItemAt(pos) && !m.IsTileSpecial(pos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) MoveDistance(pOne geometry.Point, pTwo geometry.Point) int {
	if m.cardinalMovementOnly {
		return geometry.DistanceManhattan(pOne, pTwo)
	}
	return geometry.DistanceChebyshev(pOne, pTwo)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) RayCast(origin, direction geometry.PointF, isBlockingRay func(geometry.Point) bool) util.HitInfo2D {
	direction = direction.Normalize()
	hitInfo := util.Raycast2D(origin.X, origin.Y, direction.X, direction.Y, func(x, y int64) bool {
		currentMapCell := geometry.Point{X: int(x), Y: int(y)}
		if currentMapCell == origin.ToPoint() {
			return false
		}
		return isBlockingRay(currentMapCell)
	})
	return hitInfo
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ReflectingRayCast(origin, direction geometry.PointF, maxReflections int, isBlockingRay func(geometry.Point) bool) []util.HitInfo2D {
	direction = direction.Normalize()
	hitInfos := util.ReflectingRaycast2D(origin.X, origin.Y, direction.X, direction.Y, maxReflections, func(x, y int64) bool {
		currentMapCell := geometry.Point{X: int(x), Y: int(y)}
		return isBlockingRay(currentMapCell)
	})
	return hitInfos
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ChainedRayCast(origin, direction geometry.PointF, isBlockingRay func(geometry.Point) bool, nextTarget func(geometry.Point) (bool, geometry.Point)) []util.HitInfo2D {
	direction = direction.Normalize()
	hitInfos := util.ChainedRaycast2D(origin.X, origin.Y, direction.X, direction.Y, func(x, y int64) bool {
		currentMapCell := geometry.Point{X: int(x), Y: int(y)}
		return isBlockingRay(currentMapCell)

	}, func(x, y int64) (bool, int, int) {
		currentMapCell := geometry.Point{X: int(x), Y: int(y)}
		target, point := nextTarget(currentMapCell)
		return target, point.X, point.Y
	})
	return hitInfos
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetWidth() int {
	return m.mapWidth
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetHeight() int {
	return m.mapHeight
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTileLit(pos geometry.Point) bool {
	if !m.Contains(pos) {
		return false
	}
	return m.GetCell(pos).IsLit
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsEmptyNonSpecialFloor(pos geometry.Point) bool {
	return m.Contains(pos) && m.IsTileWalkable(pos) && !m.IsActorAt(pos) && !m.IsItemAt(pos) && !m.IsObjectAt(pos) && !m.IsTileSpecial(pos)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetListExplored(tiles []geometry.Point, value bool) {
	for _, tile := range tiles {
		m.SetExplored(tile)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetMoveOnPlayerDijkstraMap(from geometry.Point, towardsPlayer bool, dijkstraMap map[geometry.Point]int) geometry.Point {
	if dijkstraMap == nil {
		return from
	}
	currentDistanceToPlayer, exist := dijkstraMap[from]
	if !exist {
		return from
	}
	var compareFunc func(a, b int) bool
	if towardsPlayer {
		compareFunc = func(a, b int) bool { return a < b }
	} else {
		compareFunc = func(a, b int) bool { return a > b }
	}
	currentMap := m
	neighbors := currentMap.GetFilteredNeighborsForMovement(from, func(pos geometry.Point) bool { // choose a possible next step
		if !currentMap.IsCurrentlyPassable(pos) { // only walk on passable tiles
			return false
		}
		neighborDist, neighborExists := dijkstraMap[pos]
		if !neighborExists {
			return false
		}
		return compareFunc(neighborDist, currentDistanceToPlayer) // depends on rolling direction on our dijkstra map
	})

	if len(neighbors) == 0 {
		return from
	}
	// we found some locations we can move to, that also bring us closer to the player
	// which of these is the closest to the player?

	nearestDist := math.MaxInt
	if !towardsPlayer {
		nearestDist = 0
	}
	nearestPos := from

	for _, neighbor := range neighbors {
		neighborDist, _ := dijkstraMap[neighbor]
		if compareFunc(neighborDist, nearestDist) {
			nearestDist = neighborDist
			nearestPos = neighbor
		}
	}

	return nearestPos
}

func (m *GridMap[ActorType, ItemType, ObjectType]) HasWalkableNeighbor(point geometry.Point) bool {
	neighbors := m.NeighborsAll(point, m.IsTileWalkable)
	return len(neighbors) > 0
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetLitByFilter(filter func(pos geometry.Point) bool) {
	for y := 0; y < m.mapHeight; y++ {
		for x := 0; x < m.mapWidth; x++ {
			pos := geometry.Point{X: x, Y: y}
			if filter(pos) {
				m.SetLit(pos, true)
			}
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetLitMulti(tiles []geometry.Point) {
	for _, tile := range tiles {
		m.SetLit(tile, true)
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsLineOfSightClear(source geometry.Point, dest geometry.Point) bool {
	direction := dest.Sub(source).ToCenteredPointF()
	hitInfo := m.RayCast(source.ToCenteredPointF(), direction, func(point geometry.Point) bool {
		if point == source {
			return true
		}
		return !m.IsCurrentlyPassable(point)
	})
	return int(hitInfo.ColliderGridPosition[0]) == dest.X && int(hitInfo.ColliderGridPosition[1]) == dest.Y
}

func (m *GridMap[ActorType, ItemType, ObjectType]) AddActorWithDisplacement(actor ActorType, position geometry.Point) {
	if m.CanPlaceActorHere(position) {
		m.AddActor(actor, position)
	} else {
		m.allActors = append(m.allActors, actor)
		m.displaceActor(actor, position)
	}
}

// ForceMoveActor ignores walkability, for monsters that pass through rock.
func (m *GridMap[ActorType, ItemType, ObjectType]) ForceMoveActor(actor ActorType, to geometry.Point) {
	from := actor.Position()
	m.cells[from.X+from.Y*m.mapWidth] = m.cells[from.X+from.Y*m.mapWidth].WithActorHereRemoved(actor)
	actor.SetPosition(to)
	m.cells[to.X+to.Y*m.mapWidth] = m.cells[to.X+to.Y*m.mapWidth].WithActor(actor)
	m.actorChanged(actor, true)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) ForceSpawnActorInWall(actor ActorType, to geometry.Point) {
	m.allActors = append(m.allActors, actor)
	actor.SetPosition(to)
	m.cells[to.X+to.Y*m.mapWidth] = m.cells[to.X+to.Y*m.mapWidth].WithActor(actor)
	m.actorChanged(actor, true)
}

func (m *GridMap[ActorType, ItemType, ObjectType]) IsTileSpecial(pos geometry.Point) bool {
	return m.GetCell(pos).TileType.IsSpecial()
}

func (m *GridMap[ActorType, ItemType, ObjectType]) SetAllLit() {
	for y := 0; y < m.mapHeight; y++ {
		for x := 0; x < m.mapWidth; x++ {
			m.SetLit(geometry.Point{X: x, Y: y}, true)
		}
	}
}

func (m *GridMap[ActorType, ItemType, ObjectType]) GetFirstWallCardinalInDirection(origin geometry.Point, dir geometry.CompassDirection) geometry.Point {
	for i := 1; i < max(m.mapWidth, m.mapHeight); i++ {
		pos := origin.Add(dir.ToPoint().Mul(i))
		if !m.Contains(pos) {
			return pos
		}
		if !m.IsTileWalkable(pos) {
			return pos
		}
	}
	return origin
}
