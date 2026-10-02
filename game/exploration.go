package game

import (
	"rx1/dungen"
	"rx1/geometry"
)

func (g *GameState) applyLightRoomExploration(playerRoom *dungen.DungeonRoom) {
	allTiles := playerRoom.GetAbsoluteRoomTiles()
	g.gridMap.SetListExplored(allTiles, true)
}

// applyLightExploration explores what the player's own light reaches, see canPlayerSee
func (g *GameState) applyLightExploration() {
	g.gridMap.SetExplored(g.Player.Position())
	if radius := g.playerLightRadius(); radius > 0 {
		g.gridMap.UpdateFieldOfView(g.playerFoV, g.Player.Position(), radius)
		for y := -radius; y <= radius; y++ {
			for x := -radius; x <= radius; x++ {
				if pos := g.Player.Position().Add(geometry.Point{X: x, Y: y}); g.gridMap.Contains(pos) && g.seenByOwnLight(pos) {
					g.gridMap.SetExplored(pos)
				}
			}
		}
	}
}
