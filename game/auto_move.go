package game

import (
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
	if g.autoMoveToward(isGoal, false) {
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
		if !object.IsHidden() && g.canPlayerSee(object.Position()) {
			spot(object)
		}
	}
	return found
}

// autoMoveToward takes one step along a known-floor path to the nearest goal tile; false if none is reachable.
func (g *GameState) autoMoveToward(isGoal func(geometry.Point) bool, isExploring bool) bool {
	m := g.gridMap
	start := g.Player.Position()
	isKnownFloor := func(p geometry.Point) bool { return m.Contains(p) && m.IsExplored(p) && m.IsWalkable(p) }
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
