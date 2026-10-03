package game

import (
	"rx1/foundation"
	"testing"
)

// A shield takes the off hand: it adds to armor, and it and a two-handed weapon push each other out.
func TestShieldOffHand(t *testing.T) {
	e := NewEquipment()
	shield := &Item{slot: foundation.SlotNameShield, armor: &ArmorInfo{}}
	sword := &Item{slot: foundation.SlotNameTwoHandedWeapon, weapon: &WeaponInfo{}}
	e.Equip(shield)
	if !e.IsEquipped(shield) || len(e.GetArmor()) != 1 {
		t.Fatal("shield not worn as armor")
	}
	if r := e.GetItemsToReplace(sword); len(r) != 1 || r[0] != shield {
		t.Fatalf("two-handed weapon should replace the shield, got %v", r)
	}
	e.Equip(sword)
	if e.IsEquipped(shield) {
		t.Fatal("shield still worn with a two-handed weapon")
	}
	e.Equip(shield)
	if e.IsEquipped(sword) || !e.IsEquipped(shield) {
		t.Fatal("shield should take the off hand back from the two-handed weapon")
	}
	e.UnEquip(shield)
	if e.IsEquipped(shield) {
		t.Fatal("shield not removed")
	}
}
