package game

import (
	"math/bits"
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// Rooms of the secret level share walls: where a wall tile points, the wall goes on (or is opened),
// and some of the tiles are junctions.
func TestSharedWallsJoinUp(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	g.GotoSecretLevel()

	arms := make(map[foundation.TileType]int)
	for mask, tile := range wallByArms {
		if tile != "" {
			arms[tile] = mask
		}
	}
	armsAt := func(pos geometry.Point) (int, bool) {
		if !g.gridMap.Contains(pos) {
			return 0, false
		}
		mask, isWall := arms[g.gridMap.GetCell(pos).TileType.Feature]
		return mask, isWall
	}
	steps := []geometry.Point{{Y: -1}, {X: 1}, {Y: 1}, {X: -1}} // as the arm bits: north, east, south, west
	junctions := 0
	size := g.gridMap.MapSize()
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			pos := geometry.Point{X: x, Y: y}
			mask, isWall := armsAt(pos)
			if bits.OnesCount(uint(mask)) > 2 {
				junctions++
			}
			for i, step := range steps {
				other, otherIsWall := armsAt(pos.Add(step))
				if isWall && otherIsWall && (mask>>i)&1 != (other>>((i+2)%4))&1 {
					t.Fatalf("the walls at %v and %v do not join up: %s, %s", pos, pos.Add(step),
						wallByArms[mask], wallByArms[other])
				}
				if isWall && mask>>i&1 == 1 && !otherIsWall && !g.gridMap.GetCell(pos.Add(step)).TileType.IsWalkable {
					t.Fatalf("the wall at %v (%s) points at %v, where there is neither wall nor opening", pos, wallByArms[mask], pos.Add(step))
				}
			}
		}
	}
	if junctions == 0 {
		t.Fatal("no shared walls on the secret level")
	}
}
