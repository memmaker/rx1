package game

import (
	"rx1/dungen"
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
		for _, pos := range g.playerFoV.Visibles {
			g.gridMap.SetExplored(pos)
		}
	}
}
