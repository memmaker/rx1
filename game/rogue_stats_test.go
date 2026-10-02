package game

import (
	"rx1/foundation"
	"testing"
)

// Rogue 5.4 stomach(): hungry below 300, weak below 150, fainting at 0, dead below -850.
func TestHungerStagesFaintAndStarvation(t *testing.T) {
	g := newTestGame()
	stats := &g.Player.stats
	stats.FoodLeft = 301
	g.digest()
	if stats.FoodLeft != 300 || g.Player.IsHungry() {
		t.Fatalf("food %d", stats.FoodLeft)
	}
	g.digest()
	if !g.Player.IsHungry() || g.GetHudFlags()[foundation.FlagHungry] != 1 {
		t.Fatal("should be hungry")
	}
	stats.FoodLeft = 150
	g.digest()
	if g.GetHudFlags()[foundation.FlagWeak] != 1 {
		t.Fatal("should be weak")
	}
	stats.FoodLeft = 0
	fainted := false
	for i := 0; i < 500 && !fainted; i++ {
		g.noCommand = 0
		g.digest()
		fainted = g.noCommand >= 4 && g.noCommand <= 11
	}
	if !fainted || g.GetHudFlags()[foundation.FlagFaint] != 1 {
		t.Fatal("should faint now and then")
	}
	stats.FoodLeft = -850
	g.digest()
	if !g.Player.IsAlive() {
		t.Fatal("still alive at -850")
	}
	g.digest()
	if g.Player.IsAlive() {
		t.Fatal("should starve below -850")
	}
}

func TestEatingAndRingHunger(t *testing.T) {
	g := newTestGame()
	stats := &g.Player.stats
	for i := 0; i < 100; i++ {
		stats.FoodLeft = -20
		g.Player.Eat()
		if stats.FoodLeft < 1100 || stats.FoodLeft > 1499 {
			t.Fatalf("ration gave %d", stats.FoodLeft)
		}
	}
	stats.FoodLeft = 1900
	g.Player.Eat()
	if stats.FoodLeft != 2000 {
		t.Fatalf("stomach holds 2000, got %d", stats.FoodLeft)
	}
	g.Player.GetEquipment().Equip(g.NewItemFromName("ring_regeneration"))
	g.Player.GetEquipment().Equip(g.NewItemFromName("ring_stealth"))
	if eat := g.Player.GetEquipment().RingFood(); eat != 3 {
		t.Fatalf("regeneration 2 + stealth 1, got %d", eat)
	}
}

func TestHealingPotions(t *testing.T) {
	g := newTestGame()
	p := g.Player
	p.stats.Lvl, p.stats.HP, p.stats.MaxHP = 3, 1, 50
	p.GetFlags().Set(foundation.FlagBlind)
	heal(g, p)
	if p.GetHitPoints() < 4 || p.GetHitPoints() > 13 || p.IsBlind() || p.GetHitPointsMax() != 50 {
		t.Fatalf("roll(3,4) healing: hp %d blind %v", p.GetHitPoints(), p.IsBlind())
	}
	p.stats.HP, p.stats.MaxHP = 10, 10
	heal(g, p)
	if p.GetHitPoints() != 11 || p.GetHitPointsMax() != 11 {
		t.Fatalf("overheal raises max by one: %d/%d", p.GetHitPoints(), p.GetHitPointsMax())
	}
	p.stats.HP, p.stats.MaxHP = 10, 10
	extraHeal(g, p)
	if p.GetHitPointsMax() < 11 || p.GetHitPointsMax() > 12 || p.GetHitPoints() != p.GetHitPointsMax() {
		t.Fatalf("extra overheal: %d/%d", p.GetHitPoints(), p.GetHitPointsMax())
	}
}

func TestGainStrengthRestoresThenAddsOne(t *testing.T) {
	g := newTestGame()
	p := g.Player
	p.ChangeStrength(-3)
	if p.stats.Str != 13 || p.stats.MaxStr != 16 {
		t.Fatalf("str %d/%d", p.stats.Str, p.stats.MaxStr)
	}
	gainStrength(g, p)
	if p.stats.Str != 17 || p.stats.MaxStr != 17 {
		t.Fatalf("restore then +1: %d/%d", p.stats.Str, p.stats.MaxStr)
	}
	p.ChangeStrength(100)
	if p.stats.Str != 31 {
		t.Fatalf("max strength 31, got %d", p.stats.Str)
	}
}

func TestStealthRingHalvesWaking(t *testing.T) {
	g := newTestGame()
	monster := g.NewEnemyFromDef(g.dataDefinitions.Monsters[0])
	count := func() int {
		n := 0
		for i := 0; i < 6000; i++ {
			if g.noticesPlayerAsleep(monster) {
				n++
			}
		}
		return n
	}
	plain := count()
	g.Player.GetEquipment().Equip(g.NewItemFromName("ring_stealth"))
	if stealthy := count(); stealthy > plain*6/10 || stealthy < plain*4/10 {
		t.Fatalf("plain %d stealthy %d", plain, stealthy)
	}
}

// Rogue 5.4 score: a tenth of the gold is lost on death, the pack is sold on escape.
func TestScore(t *testing.T) {
	g := newTestGame()
	g.Player.AddGold(1000 - g.Player.GetGold())
	if g.finalScore(false) != 900 {
		t.Fatalf("death %d", g.finalScore(false))
	}
	base := g.finalScore(true)
	inv := g.Player.GetInventory()
	inv.Add(g.NewItemFromName("potion_healing"))
	if got := g.finalScore(true) - base; got != 130 && got != 65 {
		t.Fatalf("healing potion worth %d", got)
	}
	ration := g.finalScore(true)
	inv.Add(g.NewItemFromName("food_ration"))
	if g.finalScore(true)-ration != 2 {
		t.Fatal("a ration is worth 2")
	}
}
