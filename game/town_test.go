package game

import (
	"math/rand"
	"rx1/foundation"
	"testing"
)

func TestBlacksmithSellsRustyCopies(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	ware := rusty(g.NewItemFromName("mace"))
	if ware.weapon.hitPlus != -1 || rusty(g.NewItemFromName("ring_mail")).armor.plus != -1 {
		t.Fatal("rusty is not -1")
	}
	g.Player.AddGold(itemPrice(ware))
	before := len(g.Player.GetInventory().Items())
	g.buyItemFromVendor(ware, itemPrice(ware), 1)
	items := g.Player.GetInventory().Items()
	bought := items[len(items)-1]
	if len(items) != before+1 || bought == ware || bought.Name() != "-1,-1 rusty mace" || bought.weapon.damagePlus != -1 {
		t.Fatalf("bought %q", bought.Name())
	}
}

func TestGeneralStoreSellsArrowBundles(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	arrows := func() (n int) {
		for _, i := range g.Player.GetInventory().Items() {
			if i.internalName == "arrow" {
				n++
			}
		}
		return
	}
	start := arrows()
	g.Player.RemoveGold(g.Player.GetGold())
	g.Player.AddGold(25)
	g.buyItemFromVendor(g.NewItemFromName("arrow"), 2, 12)
	g.buyItemFromVendor(g.NewItemFromName("arrow"), 2, 0) // 1 gold left: buys none
	if arrows() != start+12 || g.Player.GetGold() != 1 {
		t.Fatalf("got %d arrows, %d gold", arrows()-start, g.Player.GetGold())
	}
}

func TestPotionOfLifeRaisesMaxHP(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	potion := g.NewItemFromName("potion_life")
	if itemPrice(potion) != 500 || potion.Name() != "potion of life" {
		t.Fatalf("%q costs %d", potion.Name(), itemPrice(potion))
	}
	before := g.Player.GetHitPointsMax()
	GetAllUseEffects()[potion.GetUseEffectName()](g, g.Player)
	if g.Player.GetHitPointsMax() != before+1 {
		t.Fatal("max HP did not rise")
	}
}

func TestPotionOfLifeIsNeverFound(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	random := rand.New(rand.NewSource(1))
	for range 2000 {
		if pickWeighted(random, g.dataDefinitions.Items[foundation.ItemCategoryPotions]).InternalName == "potion_life" {
			t.Fatal("potion of life spawned")
		}
	}
}

// Rogue's new_thing odds: 10% of weapons cursed and 5% blessed, 20%/8% of armor
func TestNewThingCursesLikeRogue(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	random := rand.New(rand.NewSource(1))
	var weapons, cursedW, blessedW, armor, cursedA, blessedA int
	for range 200000 {
		item := g.rogueNewThing(random, 5)
		switch {
		case item.IsWeapon() && !item.IsMissile():
			weapons++
			if item.weapon.hitPlus < 0 && item.IsStuck() {
				cursedW++
			} else if item.weapon.hitPlus > 0 {
				blessedW++
			}
		case item.IsArmor():
			armor++
			if item.armor.plus < 0 && item.IsStuck() {
				cursedA++
			} else if item.armor.plus > 0 {
				blessedA++
			}
		}
	}
	near := func(n, of, want int) bool { p := 100 * n / of; return p >= want-1 && p <= want+1 }
	if !near(cursedW, weapons, 10) || !near(blessedW, weapons, 5) || !near(cursedA, armor, 20) || !near(blessedA, armor, 8) {
		t.Fatalf("weapons %d/%d/%d armor %d/%d/%d", weapons, cursedW, blessedW, armor, cursedA, blessedA)
	}
}

func TestTeleportRingIsStuckAndBetterArrowsStackApart(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	random := rand.New(rand.NewSource(1))
	for found := false; !found; {
		if item := g.rogueNewThing(random, 5); item.internalName == "ring_teleportation" {
			found = true
			if !item.IsStuck() || g.Player.GetEquipment().CanUnequip(item) {
				t.Fatal("teleportation ring is not cursed")
			}
		}
	}
	plain, better := g.NewItemFromName("arrow"), g.NewItemFromName("arrow")
	better.weapon.damagePlus = 2
	if plain.CanStackWith(better) || !plain.CanStackWith(g.NewItemFromName("arrow")) {
		t.Fatal("arrows stack by their plus")
	}
}

func TestPlusShowsOnlyWhenKnown(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	mace := g.NewItemFromName("mace")
	mace.weapon.damagePlus = 2
	if mace.Name() != "mace" {
		t.Fatalf("unknown shows %q", mace.Name())
	}
	mace.isKnown = true
	if mace.Name() != "+0,+2 mace" {
		t.Fatalf("known shows %q", mace.Name())
	}
}

func TestStartingKitIsRogues(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	names := map[string]int{}
	for _, i := range g.Player.GetInventory().Items() {
		names[i.Name()]++
	}
	if names["+1,+1 mace"] != 1 || names["+1 ring mail"] != 1 || names["+1,+0 short bow"] != 1 || names["ration of food"] != 1 || names["+0,+0 arrow"] < 25 || names["+0,+0 arrow"] > 39 {
		t.Fatalf("kit %v", names)
	}
	if hplus, dplus, _ := g.Player.GetMelee(""); hplus != 1 || dplus != 1 {
		t.Fatalf("melee %+d,%+d", hplus, dplus)
	}
}

func TestEnchantWeaponFlipsBetweenHitAndDamage(t *testing.T) {
	w := &WeaponInfo{}
	for range 100 {
		w.AddEnchantment()
	}
	if w.hitPlus+w.damagePlus != 100 || w.hitPlus < 30 || w.damagePlus < 30 {
		t.Fatalf("%+d,%+d", w.hitPlus, w.damagePlus)
	}
}
