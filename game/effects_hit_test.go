package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"rx1/recfile"
	"rx1/util"
	"testing"
)

func TestHitEffectsParse(t *testing.T) {
	defs := MonsterDefsFromRecords(recfile.Read(util.MustOpen("../data_rx1/definitions/monsters.rec")))
	known := GetAllHitEffects()
	n := 0
	for _, d := range defs {
		for _, e := range append(d.HitEffects, d.StruckEffects...) {
			if _, ok := known[e.Name]; !ok || e.Chance <= 0 {
				t.Errorf("%s: bad hit effect %+v", d.Name, e)
			}
			n++
		}
		for _, gaze := range d.GazeEffects {
			if gaze != "confuse" && gaze != "scare" {
				t.Errorf("%s: unknown gaze effect %s", d.Name, gaze)
			}
		}
	}
	if n != 23 {
		t.Errorf("expected 23 hit effects, got %d", n)
	}
}

func TestFlytrapGripGrowsAndEyeParalysesWhenItHits(t *testing.T) {
	g := newMagicTestGame()
	trap := NewActor("flytrap", 'f', "Green")
	hp := g.Player.GetHitPoints()
	flytrapHold(g, trap, g.Player)
	flytrapHold(g, trap, g.Player) // 1 + 2 hp, Rogue's vf_hit
	if !g.Player.HasFlag(foundation.FlagHeld) || g.Player.GetHitPoints() != hp-3 {
		t.Fatalf("held=%v hp %d -> %d", g.Player.HasFlag(foundation.FlagHeld), hp, g.Player.GetHitPoints())
	}
	for _, d := range g.dataDefinitions.Monsters {
		if d.Name == "floating eye" && (len(d.StruckEffects) != 0 || len(d.HitEffects) != 1) {
			t.Errorf("eye should paralyse on its own hit only: %+v", d)
		}
	}
}

func TestRustArmorSkipsLeather(t *testing.T) {
	g := newMagicTestGame()
	leather := g.NewItemFromName("leather_armor")
	g.giveAndTryEquipItem(g.Player, leather)
	before := leather.GetArmor().GetProtection()
	rustArmor(g, nil, g.Player)
	if leather.GetArmor().GetProtection() != before {
		t.Error("leather armor must not rust")
	}
}

func TestBreathLine(t *testing.T) {
	o := geometry.Point{X: 10, Y: 10}
	if !inBreathLine(o, geometry.Point{X: 16, Y: 10}) || !inBreathLine(o, geometry.Point{X: 13, Y: 7}) ||
		inBreathLine(o, geometry.Point{X: 17, Y: 10}) || inBreathLine(o, geometry.Point{X: 12, Y: 11}) {
		t.Error("breath line mismatch")
	}
}
