package game

import (
	"rx1/geometry"
	"testing"
)

// Each Brogue weapon pattern hurts the right monsters; they're unaware, so every attack is a sneak attack and hits.
func TestWeaponPatterns(t *testing.T) {
	east, west, north := geometry.Point{X: 1}, geometry.Point{X: -1}, geometry.Point{Y: -1}
	cases := []struct {
		weapon  string
		monster []geometry.Point // offsets from the player
		bump    geometry.Point   // direction the player moves
		hurt    []bool
	}{
		{"spear", []geometry.Point{east, {X: 2}}, east, []bool{true, true}},
		{"axe", []geometry.Point{east, west, north}, east, []bool{true, true, true}},
		{"long_sword", []geometry.Point{east, west}, east, []bool{true, false}},
		{"rapier", []geometry.Point{{X: 2}}, east, []bool{true}},
		{"whip", []geometry.Point{{X: 4}}, east, []bool{true}},
	}
	for _, c := range cases {
		g := newTestGame()
		g.GotoNamedLevel("town")
		weapon := g.NewItemFromName(c.weapon)
		g.Player.GetInventory().Add(weapon)
		g.Player.GetEquipment().Equip(weapon)
		start := g.Player.Position()
		var monsters []*Actor
		for _, off := range c.monster {
			m := g.NewEnemyFromDef(g.dataDefinitions.Monsters[len(g.dataDefinitions.Monsters)-1])
			g.gridMap.AddActor(m, start.Add(off))
			monsters = append(monsters, m)
		}
		hp := make([]int, len(monsters))
		for i, m := range monsters {
			hp[i] = m.GetHitPoints()
		}
		g.updatePlayerFoVAndApplyExploration()
		g.ManualMovePlayer(c.bump.ToDirection())
		for i, m := range monsters {
			if got := m.GetHitPoints() < hp[i] || !m.IsAlive(); got != c.hurt[i] {
				t.Errorf("%s: monster %d hurt=%v, want %v", c.weapon, i, got, c.hurt[i])
			}
		}
		if c.weapon == "rapier" && g.Player.Position() != start.Add(east) {
			t.Errorf("rapier: the lunge didn't move the player")
		}
	}
}
