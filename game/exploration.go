package game

import "rx1/geometry"

// applyLightExploration explores what the player sees, see canPlayerSee.
// ponytail: checks the whole map each time, fine for 80x23; walk playerFoV.Visibles and their neighbours if maps grow.
func (g *GameState) applyLightExploration() {
	g.gridMap.UpdateFieldOfView(g.playerFoV, g.Player.Position(), g.visionRange)
	for y := 0; y < g.gridMap.GetHeight(); y++ {
		for x := 0; x < g.gridMap.GetWidth(); x++ {
			if pos := (geometry.Point{X: x, Y: y}); g.canPlayerSee(pos) {
				g.gridMap.SetExplored(pos)
			}
		}
	}
}
