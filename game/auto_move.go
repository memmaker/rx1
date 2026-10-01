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
	start := player.Position()
	isKnownFloor := func(p geometry.Point) bool { return m.Contains(p) && m.IsExplored(p) && m.IsWalkable(p) }
	isFrontier := func(p geometry.Point) bool {
		return len(m.NeighborsAll(p, func(n geometry.Point) bool { return m.Contains(n) && !m.IsExplored(n) })) > 0
	}
	// breadth-first search; firstStep remembers which neighbour of start each tile was reached through
	firstStep := map[geometry.Point]geometry.Point{start: start}
	queue := []geometry.Point{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current != start && isFrontier(current) {
			return g.autoExploreMove(start, firstStep[current])
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
	g.msg(foundation.Msg("There is nothing left to explore"))
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
