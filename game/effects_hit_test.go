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
	if n != 25 {
		t.Errorf("expected 25 hit effects, got %d", n)
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

// D&D ochre jelly rules: the slime halves do half damage, and lightning divides it instead of hurting it.
func TestSlimeSplitHalvesDamage(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("slime")
	slime := g.NewEnemyFromDef(def)
	slime.stats.HP = 10
	g.gridMap.AddActor(slime, g.Player.Position().Add(geometry.Point{X: 2}))
	if !splits(slime) {
		t.Fatal("slime should split")
	}
	split(g, slime, g.Player)
	if slime.stats.HP != 5 || slime.stats.Dmg != "1d2" {
		t.Errorf("after split: hp %d dmg %s", slime.stats.HP, slime.stats.Dmg)
	}
}

// D&D trolls: a killing blow of fire or acid keeps them dead.
func TestFinalBlowCancelsRevive(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("troll")
	troll := g.NewEnemyFromDef(def)
	fireDamage(troll, troll.GetHitPoints()-1)
	if !troll.HasFlag(foundation.FlagRevive) {
		t.Fatal("a non-lethal burn must not cancel the revive")
	}
	fireDamage(troll, troll.GetHitPoints())
	if troll.HasFlag(foundation.FlagRevive) {
		t.Error("a lethal burn should cancel the revive")
	}
	aq, _ := g.monsterDefByInternalName("aquator")
	if !isAcidic(g.NewEnemyFromDef(aq)) {
		t.Error("the aquator strikes with acid")
	}
}

// D&D wraith: mundane weapons do half damage, enchanted ones full.
func TestWraithHalvesMundaneDamage(t *testing.T) {
	g := newMagicTestGame()
	def, _ := g.monsterDefByInternalName("wraith")
	wraith := g.NewEnemyFromDef(def)
	mace := g.NewItemFromName("mace")
	if got := mundaneHalf(wraith, mace, 8); got != 4 {
		t.Errorf("mundane mace: %d, want 4", got)
	}
	mace.GetWeapon().AddEnchantment()
	if got := mundaneHalf(wraith, mace, 8); got != 8 {
		t.Errorf("enchanted mace: %d, want 8", got)
	}
}
