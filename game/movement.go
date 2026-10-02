package game

import (
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
)

func (g *GameState) ManualMovePlayer(direction geometry.CompassDirection) {
	if !g.config.DiagonalMovementEnabled && direction.IsDiagonal() {
		return
	}

	player := g.Player
	oldPos := player.Position()

	// adapted from: https://github.com/memmaker/rogue-pc-modern-C/blob/582340fcaef32dd91595721efb2d5db41ff3cb05/src/move.c#L56
	if player.HasFlag(foundation.FlagConfused) && rand.Intn(5) != 0 {
		direction = geometry.RandomDirection()
	}

	newPos := oldPos.Add(direction.ToPoint())

	if !g.gridMap.Contains(newPos) || !g.gridMap.IsTileWalkable(newPos) {
		if !g.config.WallSlide {
			return
		}
		// check for wallslide
		var forwardLeft, forwardRight geometry.Point
		var forwardLeftTest, forwardRightTest geometry.Point
		if direction.IsDiagonal() {
			dirVec := direction.ToPoint()
			leftDir := geometry.Point{X: 0, Y: dirVec.Y}
			rightDir := geometry.Point{X: dirVec.X, Y: 0}
			forwardLeft = oldPos.Add(leftDir)
			forwardRight = oldPos.Add(rightDir)
			forwardLeftTest = forwardLeft
			forwardRightTest = forwardRight
		} else { // cardinal case
			rightDir := direction.TurnRightBy90()
			leftDir := direction.TurnLeftBy90()
			forwardLeft = oldPos.Add(leftDir.ToPoint())
			forwardRight = oldPos.Add(rightDir.ToPoint())

			// the step is sideways: the diagonal one past the wall in front is never allowed (diag_ok)
			forwardLeftTest = forwardLeft.Add(direction.ToPoint())
			forwardRightTest = forwardRight.Add(direction.ToPoint())
		}

		if g.gridMap.IsCurrentlyPassable(forwardLeftTest) && !g.gridMap.IsTileWalkable(forwardRightTest) {
			newPos = forwardLeft // we can slide :)
		} else if g.gridMap.IsCurrentlyPassable(forwardRightTest) && !g.gridMap.IsTileWalkable(forwardLeftTest) {
			newPos = forwardRight // we can slide :)
		} else {
			return // no slide :(, no move
		}
	}

	if !g.gridMap.DiagonalOK(oldPos, newPos) { // Rogue's diag_ok, for moves and attacks alike
		return
	}

	if actorAt, exists := g.gridMap.TryGetActorAt(newPos); exists {
		g.playerAttack(actorAt)
		return
	}

	if g.tryReachAttack(newPos.Sub(oldPos)) {
		return
	}

	if objectAt, exists := g.gridMap.TryGetObjectAt(newPos); exists && !objectAt.IsWalkable(g.Player) {
		g.msg(foundation.Msg("You bump into something"))
		return
	}
	direction = newPos.Sub(oldPos).ToDirection()
	g.playerMove(newPos)
	g.ui.AfterPlayerMoved(foundation.MoveInfo{
		Direction: direction,
		OldPos:    oldPos,
		NewPos:    newPos,
		Mode:      foundation.PlayerMoveModeManual,
	})
}
func (g *GameState) afterPlayerMoved() {
	// explore the map
	// print "You see.." message
	if g.gridMap.IsItemAt(g.Player.Position()) && g.config.AutoPickup {
		g.PickupItem()
	}

	if _, isSecret := g.secrets[g.Player.Position()]; isSecret {
		g.revealSecret(g.Player.Position(), false)
	}
	g.msg(g.GetMapInfoForMovement(g.Player.Position()))
	switch g.gridMap.GetCell(g.Player.Position()).TileType.Feature {
	case foundation.TileChasm: // Brogue's chasms lead to the level below
		g.msg(foundation.HiLite("You plunge downward into the chasm!"))
		g.QueueActionAfterAnimation(g.descendToRandomLocation)
	case foundation.TileFungusForest: // trampled, it no longer hides what is behind it
		g.gridMap.SetTile(g.Player.Position(), gridmap.Tile{Feature: foundation.TileFungus, DefinedDescription: "trampled fungal foliage", IsWalkable: true, IsTransparent: true})
	}
	g.exploreMap()
	g.updateDijkstraMap()

	if g.Player.HasFlag(foundation.FlagCurseTeleportitis) && rand.Intn(100) < 2 {
		g.ui.AddAnimations(OneAnimation(teleportWithAnimation(g, g.Player, g.gridMap.RandomSpawnPosition())))
	}
}

func (g *GameState) updateDijkstraMap() {
	g.playerDijkstraMap = g.gridMap.GetDijkstraMapWithActorsNotBlocking(g.Player.Position(), 1000)
}

func (g *GameState) exploreMap() {
	if g.currentDungeonLevel == 0 {
		return
	}
	if g.dungeonLayout == nil { // no dungeon as a base
		g.updatePlayerFoVAndApplyExploration()
		return
	}

	playerRoom := g.getPlayerRoom()

	if playerRoom != nil && playerRoom.IsLit() {
		if g.TurnsTaken-playerRoom.LastSeenTurn > 50 {
			g.checkTilesForHiddenObjects(playerRoom.GetAbsoluteFloorTiles())
			playerRoom.LastSeenTurn = g.TurnsTaken
		}
	}
	g.applyLightExploration()
}
