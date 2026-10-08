package game

import (
	"container/heap"
	"slices"

	"rx1/foundation"
	"rx1/geometry"
)

func (g *GameState) RunPlayer(direction geometry.CompassDirection, isStarting bool) bool {
	player := g.Player
	if player.HasFlag(foundation.FlagConfused) {
		g.msg(foundation.Msg("You cannot run while confused"))
		return false
	}

	if len(g.GetVisibleEnemies()) > 0 {
		g.msg(foundation.Msg("You cannot run while enemies are near"))
		return false
	}

	currentPos := player.Position()
	targetPos := currentPos.Add(direction.ToPoint())
	currentMap := g.gridMap

	if !isStarting {
		if !currentMap.IsTileWalkable(targetPos) && !direction.IsDiagonal() {

			leftDir := direction.TurnLeftBy90()
			rightDir := direction.TurnRightBy90()
			leftTarget := currentPos.Add(leftDir.ToPoint())
			rightTarget := currentPos.Add(rightDir.ToPoint())
			isFreeLeft := currentMap.IsTileWalkable(leftTarget)
			isFreeRight := currentMap.IsTileWalkable(rightTarget)

			if (isFreeLeft && isFreeRight) || (!isFreeLeft && !isFreeRight) {
				return false
			}
			if isFreeLeft {
				targetPos = leftTarget
			} else if isFreeRight {
				targetPos = rightTarget
			}
		}
	}

	if !currentMap.IsCurrentlyPassable(targetPos) {
		return false
	}

	currentDirection := targetPos.Sub(currentPos).ToDirection()
	if g.dungeonLayout != nil && g.dungeonLayout.IsCorridor(currentPos) && g.dungeonLayout.IsDoorAt(targetPos) {
		firstRoomTileAfterDoor := targetPos.Add(currentDirection.ToPoint())
		if !currentMap.IsExplored(firstRoomTileAfterDoor) {
			return false
		}
	}
	g.playerMove(targetPos)

	g.ui.AfterPlayerMoved(foundation.MoveInfo{
		Direction: currentDirection,
		OldPos:    currentPos,
		NewPos:    targetPos,
		Mode:      foundation.PlayerMoveModeRun,
	})
	return true
}

func (g *GameState) IsPlayerOnStairs(down bool) bool {
	t := g.gridMap.GetCell(g.Player.Position()).TileType
	return down && t.IsStairsDown() || !down && t.IsStairsUp()
}

// TravelToStairsStep walks one step toward the nearest explored stairs (down or up); false when
// the player arrived, nothing is known or travel has to stop.
func (g *GameState) TravelToStairsStep(down bool) bool {
	m := g.gridMap
	isGoal := func(p geometry.Point) bool {
		t := m.GetCell(p).TileType
		return m.IsExplored(p) && (down && t.IsStairsDown() || !down && t.IsStairsUp())
	}
	if g.IsPlayerOnStairs(down) || len(g.GetVisibleEnemies()) > 0 || g.Player.HasFlag(foundation.FlagConfused) {
		return false
	}
	if g.travelAlongPath(isGoal) {
		return true
	}
	if !g.IsPlayerOnStairs(down) { // no step and not arrived: no known path
		direction := "up"
		if down {
			direction = "down"
		}
		g.msg(foundation.Msg("You don't know the way to the stairs " + direction))
	}
	return false
}

// AutoExploreStep walks one step toward the nearest explored tile that borders unexplored space.
func (g *GameState) AutoExploreStep() bool {
	player := g.Player
	if player.HasFlag(foundation.FlagConfused) {
		g.msg(foundation.Msg("You cannot explore while confused"))
		return false
	}
	if len(g.GetVisibleEnemies()) > 0 {
		g.msg(foundation.Msg("You cannot explore while enemies are near"))
		return false
	}
	m := g.gridMap
	if g.exploreVisitedMap != m {
		g.exploreVisited, g.exploreVisitedMap, g.exploreSeen = map[geometry.Point]bool{}, m, nil
	}
	g.exploreVisited[player.Position()] = true
	g.spotNewThings() // what is in view when exploring starts is not a discovery
	isFrontier := func(p geometry.Point) bool {
		// a tile we stood on without revealing its neighbours never will; skipping it stops the back-and-forth
		return !g.exploreVisited[p] && len(m.NeighborsAll(p, func(n geometry.Point) bool { return m.Contains(n) && !m.IsExplored(n) })) > 0
	}
	if !g.autoMoveToward(isFrontier, true) {
		return false
	}
	if g.spotNewThings() || len(g.GetVisibleEnemies()) > 0 {
		g.msg(foundation.Msg("You stop exploring: you spot something"))
		return false
	}
	return true
}

// spotNewThings remembers the items and visible traps in view; true if any of them were not seen before.
func (g *GameState) spotNewThings() bool {
	if g.exploreSeen == nil {
		g.exploreSeen = map[any]bool{}
	}
	found := false
	spot := func(thing any) {
		if !g.exploreSeen[thing] {
			g.exploreSeen[thing], found = true, true
		}
	}
	for _, item := range g.playerVisibleItemsByDistance() {
		spot(item)
	}
	for _, object := range g.gridMap.Objects() {
		if object.IsTrap() && !object.IsHidden() && g.canPlayerSee(object.Position()) {
			spot(object)
		}
	}
	return found
}

// travelAlongPath takes one step along the cheapest A* path over known floor to the closest goal
// tile, with diagonals costing 1.4 straight steps so routes follow the straight line; false if
// none is reachable or the player stops on an item. The path is planned once and only re-planned
// when the player left it, the level changed or the next tile stopped being known floor.
func (g *GameState) travelAlongPath(isGoal func(geometry.Point) bool) bool {
	m := g.gridMap
	start := g.Player.Position()
	isKnownFloor := func(p geometry.Point) bool {
		return m.Contains(p) && m.IsExplored(p) && m.IsWalkable(p) && !m.IsObviousHazardAt(p)
	}
	path := g.travelPath
	if g.travelPathMap != m || len(path) < 2 || path[0] != start || !isGoal(path[len(path)-1]) || !isKnownFloor(path[1]) {
		var goals []geometry.Point
		for y := 0; y < m.GetHeight(); y++ {
			for x := 0; x < m.GetWidth(); x++ {
				if p := (geometry.Point{X: x, Y: y}); p != start && isGoal(p) && isKnownFloor(p) {
					goals = append(goals, p)
				}
			}
		}
		path = astarPath(start, goals, func(p geometry.Point) []geometry.Point { return m.NeighborsAll(p, isKnownFloor) })
	}
	g.travelPath, g.travelPathMap = nil, m
	if len(path) < 2 || !g.autoExploreMove(start, path[1]) {
		return false
	}
	g.travelPath = path[1:]
	return len(path) > 2 // false once the goal is reached
}

// stepCost is 10 for a straight step and 14 for a diagonal one.
func stepCost(a, b geometry.Point) int {
	if a.X != b.X && a.Y != b.Y {
		return 14
	}
	return 10
}

// octile is the exact stepCost distance on an empty grid, so it never overestimates.
func octile(a, b geometry.Point) int {
	dx, dy := geometry.Abs(a.X-b.X), geometry.Abs(a.Y-b.Y)
	return 10*max(dx, dy) + 4*min(dx, dy)
}

type astarItem struct {
	p          geometry.Point
	rank, cost int
}

type astarQueue []astarItem

func (q astarQueue) Len() int           { return len(q) }
func (q astarQueue) Less(i, j int) bool { return q[i].rank < q[j].rank }
func (q astarQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *astarQueue) Push(x any)        { *q = append(*q, x.(astarItem)) }
func (q *astarQueue) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// astarPath returns the cheapest path from start to the nearest of goals (both included), nil if none.
// ponytail: heuristic scans every goal per node, fine for a handful of stairs.
func astarPath(start geometry.Point, goals []geometry.Point, neighbors func(geometry.Point) []geometry.Point) []geometry.Point {
	if len(goals) == 0 {
		return nil
	}
	h := func(p geometry.Point) int {
		best := octile(p, goals[0])
		for _, goal := range goals[1:] {
			best = min(best, octile(p, goal))
		}
		return best
	}
	cost := map[geometry.Point]int{start: 0}
	parent := map[geometry.Point]geometry.Point{}
	queue := &astarQueue{{start, h(start), 0}}
	for queue.Len() > 0 {
		cur := heap.Pop(queue).(astarItem)
		if cur.cost > cost[cur.p] {
			continue // stale entry
		}
		if h(cur.p) == 0 {
			path := []geometry.Point{cur.p}
			for p := cur.p; p != start; {
				p = parent[p]
				path = append(path, p)
			}
			slices.Reverse(path)
			return path
		}
		for _, next := range neighbors(cur.p) {
			c := cur.cost + stepCost(cur.p, next)
			if old, seen := cost[next]; !seen || c < old {
				cost[next], parent[next] = c, cur.p
				heap.Push(queue, astarItem{next, c + h(next), c})
			}
		}
	}
	return nil
}

// autoMoveToward takes one step along a known-floor path to the nearest goal tile; false if none is reachable.
func (g *GameState) autoMoveToward(isGoal func(geometry.Point) bool, isExploring bool) bool {
	m := g.gridMap
	start := g.Player.Position()
	isKnownFloor := func(p geometry.Point) bool {
		return m.Contains(p) && m.IsExplored(p) && m.IsWalkable(p) && !m.IsObviousHazardAt(p) // no stroll into a chasm
	}
	// breadth-first search; firstStep remembers which neighbour of start each tile was reached through
	firstStep := map[geometry.Point]geometry.Point{start: start}
	queue := []geometry.Point{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && isGoal(current) {
			return g.autoExploreMove(start, firstStep[current]) && !(firstStep[current] == current && !isExploring)
		}
		for _, next := range m.NeighborsAll(current, isKnownFloor) {
			if _, seen := firstStep[next]; seen {
				continue
			}
			if current == start {
				firstStep[next] = next
			} else {
				firstStep[next] = firstStep[current]
			}
			queue = append(queue, next)
		}
	}
	if isExploring {
		g.msg(foundation.Msg("There is nothing left to explore"))
	}
	return false
}

func (g *GameState) autoExploreMove(currentPos, targetPos geometry.Point) bool {
	if !g.gridMap.IsCurrentlyPassable(targetPos) {
		return false
	}
	g.playerMove(targetPos)
	g.ui.AfterPlayerMoved(foundation.MoveInfo{
		Direction: targetPos.Sub(currentPos).ToDirection(),
		OldPos:    currentPos,
		NewPos:    targetPos,
		Mode:      foundation.PlayerMoveModeRun,
	})
	return !g.gridMap.IsItemAt(g.Player.Position()) // stop on items so the player sees them
}
