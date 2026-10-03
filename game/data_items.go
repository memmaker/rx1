package game

import (
	"fmt"
	"image/color"
	"rx1/foundation"
	"rx1/recfile"
	"rx1/rpg"
	"strings"
)

type WeaponType int

var weaponTypeNames = []string{"", "sword", "club", "axe", "dagger", "spear", "bow", "arrow", "crossbow", "bolt", "dart", "rapier", "whip"}

// String is the weapon_type word of weapons.rec, "" for ItemTypeUnknown.
func (t WeaponType) String() string { return weaponTypeNames[t] }

func (t WeaponType) IsMissile() bool {
	return t == ItemTypeArrow || t == ItemTypeBolt || t == ItemTypeDart
}

const (
	ItemTypeUnknown WeaponType = iota
	ItemTypeSword
	ItemTypeClub
	ItemTypeAxe
	ItemTypeDagger
	ItemTypeSpear
	ItemTypeBow
	ItemTypeArrow
	ItemTypeCrossbow
	ItemTypeBolt
	ItemTypeDart
	ItemTypeRapier
	ItemTypeWhip
)

type WeaponDef struct {
	DamageDice          rpg.Dice
	Type                WeaponType
	LaunchedWithType    WeaponType
	ShotMaxRange        int
	ShotMinRange        int
	ShotHalfDamageRange int
	ShotAccuracy        int
}

func (w WeaponDef) IsValid() bool {
	return w.Type != ItemTypeUnknown && w.DamageDice.NotZero()
}

type ArmorDef struct {
	Protection int
}

type ItemDef struct {
	Name         string
	InternalName string
	Chance       int // relative weight within its category (Rogue's o_prob)
	Worth        int // Rogue's oi_worth, for the score

	Slot foundation.EquipSlot

	WeaponDef WeaponDef
	ArmorDef  ArmorDef

	ThrowDamageDice rpg.Dice

	UseEffect string
	ZapEffect string

	Stat      rpg.Stat
	StatBonus rpg.Dice

	Charges  rpg.Dice
	Category foundation.ItemCategory

	AlwaysIDOnUse bool
	EquipFlag     foundation.ActorFlag

	Text string // documents only

	Description string // the item's rules, shown in the inventory once identified

	LightRadius       int
	LightColor        color.RGBA
	LightPattern      string
	LightFrameDelayMs int
}

func (i ItemDef) IsValidArmor() bool {
	return i.Slot.IsArmorSlot()
}

func (i ItemDef) IsValidWeapon() bool {
	return i.Slot.IsWeaponSlot() && i.WeaponDef.IsValid()
}

func ItemDefsFromRecords(otherRecords []recfile.Record) []ItemDef {
	var items []ItemDef
	for _, record := range otherRecords {
		itemDef := NewItemDefFromRecord(record)
		items = append(items, itemDef)
	}
	return items
}

func NewItemDefFromRecord(record recfile.Record) ItemDef {
	itemDef := ItemDef{}

	for _, field := range record {
		switch field.Name {
		case "name":
			itemDef.Name = field.Value
		case "internal_name":
			itemDef.InternalName = field.Value
		case "chance":
			itemDef.Chance = field.AsInt()
		case "worth":
			itemDef.Worth = field.AsInt()
		case "category":
			itemDef.Category = foundation.ItemCategoryFromString(field.Value)
		case "slot":
			itemDef.Slot = foundation.ItemSlotFromString(field.Value)
		case "weapon_type":
			itemDef.WeaponDef.Type = WeaponTypeFromString(field.Value)
		case "weapon_launched_with_type":
			itemDef.WeaponDef.LaunchedWithType = WeaponTypeFromString(field.Value)
		case "weapon_damage":
			itemDef.WeaponDef.DamageDice = rpg.ParseDice(field.Value)
		case "armor":
			itemDef.ArmorDef.Protection = field.AsInt()
		case "thrown_damage":
			itemDef.ThrowDamageDice = rpg.ParseDice(field.Value)
		case "shot_max_range":
			itemDef.WeaponDef.ShotMaxRange = field.AsInt()
		case "shot_min_range":
			itemDef.WeaponDef.ShotMinRange = field.AsInt()
		case "shot_half_damage_range":
			itemDef.WeaponDef.ShotHalfDamageRange = field.AsInt()
		case "shot_accuracy":
			itemDef.WeaponDef.ShotAccuracy = field.AsInt()
		case "use_effect":
			if useEffectExists(field.Value) {
				itemDef.UseEffect = field.Value
			} else {
				panic("Invalid use effect: " + field.Value)
			}
		case "zap_effect":
			if zapEffectExists(field.Value) {
				itemDef.ZapEffect = field.Value
			} else {
				panic("Invalid zap effect: " + field.Value)
			}
		case "charges":
			itemDef.Charges = rpg.ParseDice(field.Value)
		case "always_id_on_use":
			itemDef.AlwaysIDOnUse = field.AsBool()
		case "stat":
			itemDef.Stat = rpg.StatFromString(field.Value)
		case "stat_bonus":
			itemDef.StatBonus = rpg.ParseDice(field.Value)
		case "light_radius":
			itemDef.LightRadius = field.AsInt()
		case "light_color":
			var r, g, b uint8
			fmt.Sscan(field.Value, &r, &g, &b)
			itemDef.LightColor = color.RGBA{R: r, G: g, B: b, A: 255}
		case "light_flicker_pattern":
			itemDef.LightPattern = field.Value
		case "light_flicker_frame_delay":
			itemDef.LightFrameDelayMs = field.AsInt()
		case "description":
			itemDef.Description = field.Value
		case "equip_flag":
			itemDef.EquipFlag = foundation.ActorFlagFromString(field.Value)
		}
	}

	return itemDef
}

func WeaponTypeFromString(value string) WeaponType {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "sword":
		return ItemTypeSword
	case "club":
		return ItemTypeClub
	case "axe":
		return ItemTypeAxe
	case "dagger":
		return ItemTypeDagger
	case "spear":
		return ItemTypeSpear
	case "bow":
		return ItemTypeBow
	case "arrow":
		return ItemTypeArrow
	case "crossbow":
		return ItemTypeCrossbow
	case "bolt":
		return ItemTypeBolt
	case "dart":
		return ItemTypeDart
	case "rapier":
		return ItemTypeRapier
	case "whip":
		return ItemTypeWhip
	}
	panic("Invalid weapon type: " + value)
}
