package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"testing"
)

// Rogue's diag_ok: no diagonal step or attack past a wall corner or through a doorway.
func TestDiagonalOK(t *testing.T) {
	g := trapTestGame(t)
	from := g.Player.Position()
	to := from.Add(geometry.Point{X: 1, Y: 1})
	floor := gridmap.Tile{Feature: foundation.TileCorridorFloor, IsWalkable: true, IsTransparent: true}
	for _, p := range []geometry.Point{from, to, {X: from.X + 1, Y: from.Y}, {X: from.X, Y: from.Y + 1}} {
		g.gridMap.SetTile(p, floor)
	}
	if !g.gridMap.DiagonalOK(from, to) {
		t.Fatal("open floor allows the diagonal")
	}
	g.gridMap.SetTile(geometry.Point{X: from.X + 1, Y: from.Y}, gridmap.Tile{Feature: foundation.TileWall})
	if g.gridMap.DiagonalOK(from, to) {
		t.Fatal("a wall corner blocks the diagonal")
	}
	g.gridMap.SetTile(geometry.Point{X: from.X + 1, Y: from.Y}, floor)
	g.gridMap.SetTile(to, gridmap.Tile{Feature: foundation.TileDoorOpen, IsWalkable: true})
	if g.gridMap.DiagonalOK(from, to) {
		t.Fatal("a doorway blocks the diagonal")
	}
}

// Rogue's runto: any damage wakes the monster and sets it after the hero.
func TestDamageAlwaysWakes(t *testing.T) {
	g := trapTestGame(t)
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	monster.GetFlags().Set(foundation.FlagSleep)
	monster.stats.HP, monster.stats.MaxHP = 100, 100
	g.gridMap.AddActor(monster, g.Player.Position().Add(geometry.Point{X: 2}))
	for i := 0; i < 50; i++ {
		monster.GetFlags().Set(foundation.FlagSleep)
		g.damageActor("test", monster, 1)
		if monster.IsSleeping() || !monster.HasFlag(foundation.FlagAwareOfPlayer) {
			t.Fatal("damage must always wake the monster and make it aware")
		}
	}
}
