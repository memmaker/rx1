package game

import (
	"fmt"
	"image/color"
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
	"strings"

	"codeberg.org/tslocum/cview"
)

type WeaponInfo struct {
	damageDice       rpg.Dice
	hitPlus          int // Rogue's o_hplus
	damagePlus       int // Rogue's o_dplus
	weaponType       WeaponType
	launchedWithType WeaponType
	vorpalEnemy      string
}

func (i *WeaponInfo) Vorpalize(enemy string) {
	i.vorpalEnemy = enemy
}

func (i *WeaponInfo) GetDamageDice() rpg.Dice {
	return i.damageDice.WithBonus(i.damagePlus)
}
func (i *WeaponInfo) GetVorpalBonus(enemyName string) (int, int) {
	if i.vorpalEnemy != "" {
		if i.vorpalEnemy == enemyName {
			return 4, 4
		}
		return 1, 1
	}
	return 0, 0
}

// Corrode lowers the damage bonus, down to -3.
func (i *WeaponInfo) Corrode() bool {
	if i.damagePlus <= -3 {
		return false
	}
	i.damagePlus--
	return true
}

// AddEnchantment is Rogue's coin flip: +1 to hit or +1 damage
func (i *WeaponInfo) AddEnchantment() {
	if rand.Intn(2) == 0 {
		i.hitPlus++
	} else {
		i.damagePlus++
	}
}

func (i *WeaponInfo) IsEnchanted() bool {
	return i.hitPlus > 0 || i.damagePlus > 0
}

func (i *WeaponInfo) IsLaunchedWith(category WeaponType) bool {
	return i.launchedWithType == category
}

func (i *WeaponInfo) GetWeaponType() WeaponType {
	return i.weaponType
}

func (i *WeaponInfo) IsVorpal() bool {
	return i.vorpalEnemy != ""
}

// ArmorInfo: protection is how much the armor lowers the AC of 10, like Rogue 5.4 shows it.
type ArmorInfo struct {
	protection int
	plus       int
}

func (i *ArmorInfo) GetProtection() int {
	return i.protection + i.plus
}

func (i *ArmorInfo) AddEnchantment() {
	i.plus++
}

func (i *ArmorInfo) IsEnchanted() bool {
	return i.plus > 0
}

type Item struct {
	name          string
	internalName  string
	position      geometry.Point
	category      foundation.ItemCategory
	weapon        *WeaponInfo
	armor         *ArmorInfo
	useEffectName string
	zapEffectName string
	charges       int
	slot          foundation.EquipSlot

	id        *IdentificationKnowledge
	stat      rpg.Stat
	statBonus int

	equipFlag    foundation.ActorFlag
	stuckTurns   int // cursed: cannot be taken off for this many more equipped turns
	thrownDamage rpg.Dice
	isKnown      bool
	found        bool // Rogue ISFOUND: a scare monster scroll that was picked up once
	text         string
	description  string // rules text from the .rec file

	light foundation.LightInfo // Radius is the full radius, see LightRadius()

	bundle int // missiles lying on the floor with this one, picked up together
}

// copyOfMissile: one more of the same missile
func (i *Item) copyOfMissile() *Item {
	c := *i
	w := *i.weapon
	c.weapon, c.bundle = &w, 0
	return &c
}

func (i *Item) IsLight() bool {
	return i.light.Radius > 0
}

// LightRadius is 0 once the fuel is burnt out (charges -1 = infinite)
func (i *Item) LightRadius() int {
	if i.charges == 0 {
		return 0
	}
	return i.light.Radius
}

func (i *Item) InventoryNameWithColorsAndShortcut(lineColorCode string) string {
	return fmt.Sprintf("%c - %s", i.Shortcut(), i.InventoryNameWithColors(lineColorCode))
}

func (i *Item) Shortcut() rune {
	return -1
}

func (i *Item) DisplayLength() int {
	return cview.TaggedStringWidth(i.InventoryNameWithColorsAndShortcut(""))
}

func (i *Item) GetListInfo() string {
	return fmt.Sprintf("%s", i.name)
}

// Description: stats and rules; an unidentified item tells nothing beyond its looks
func (i *Item) Description() string {
	lines := []string{i.InventoryNameWithColors(""), " " + i.category.String()}
	if i.IsMagic() && !i.id.IsItemIdentified(i.internalName) {
		return strings.Join(append(lines, "", " Unidentified: use it, or have it identified."), "\n")
	}
	if i.IsWeapon() && i.thrownDamage.NotZero() {
		lines = append(lines, " thrown: "+i.thrownDamage.ShortString())
	}
	if i.IsRing() && i.stat != "" {
		lines = append(lines, fmt.Sprintf(" %s %+d", i.stat, i.statBonus))
	}
	if (i.IsWeapon() || i.IsArmor()) && !i.isKnown {
		lines = append(lines, " enchantment unknown")
	}
	if i.description != "" {
		lines = append(lines, "", cview.Escape(i.description))
	}
	return strings.Join(lines, "\n")
}

func (i *Item) InventoryNameWithColors(colorCode string) string {
	line := cview.Escape(i.Name())
	if i.IsWeapon() {
		line = cview.Escape(fmt.Sprintf("%s (%s)", i.Name(), i.weapon.GetDamageDice().ShortString()))
	}
	if i.IsArmor() {
		line = cview.Escape(fmt.Sprintf("%s [%+d]", i.Name(), i.armor.GetProtection()))
	}
	if i.IsRing() && i.charges > 1 && i.id.IsItemIdentified(i.internalName) {
		line = cview.Escape(fmt.Sprintf("%s (%d turns)", i.Name(), i.charges))
	}
	if i.IsWand() && i.id.IsItemIdentified(i.internalName) {
		line = cview.Escape(fmt.Sprintf("%s (%d charges)", i.Name(), i.charges))
	}
	if i.IsLight() && i.charges >= 0 {
		line = cview.Escape(fmt.Sprintf("%s (%d turns)", i.Name(), i.charges))
	}
	return colorCode + line + "[-]"
}

func (i *Item) SetPosition(pos geometry.Point) {
	i.position = pos
}

func (i *Item) Position() geometry.Point {
	return i.position
}

func (i *Item) Name() string {
	name := i.name
	if i.IsGold() {
		name = fmt.Sprintf("%d gold", i.charges)
	}

	if i.IsPotion() && !i.id.IsItemIdentified(i.internalName) {
		flavor := i.id.GetPotionColor(i.internalName)
		name = fmt.Sprintf("%s potion", flavor)
	}

	if i.IsScroll() && !i.id.IsItemIdentified(i.internalName) {
		flavor := i.id.GetScrollName(i.internalName)
		name = fmt.Sprintf("scroll of '%s'", flavor)
	}

	if i.IsWand() && !i.id.IsItemIdentified(i.internalName) {
		flavor := i.id.GetWandMaterial(i.internalName)
		name = fmt.Sprintf("%s wand", flavor)
	}

	if i.IsRing() && !i.id.IsItemIdentified(i.internalName) {
		flavor := i.id.GetRingStone(i.internalName)
		name = fmt.Sprintf("%s ring", flavor)
	}

	if i.isKnown && i.IsWeapon() {
		name = fmt.Sprintf("%+d,%+d %s", i.weapon.hitPlus, i.weapon.damagePlus, name)
	}
	if i.isKnown && i.IsArmor() {
		name = fmt.Sprintf("%+d %s", i.armor.plus, name)
	}

	if i.IsStuck() && i.isKnown {
		name = fmt.Sprintf("*%d* %s", i.stuckTurns, name)
	}

	if i.statBonus != 0 && (i.isKnown || i.id.IsItemIdentified(i.internalName)) {
		name = fmt.Sprintf("%s [%+d %s]", name, i.statBonus, strings.ReplaceAll(string(i.stat), "_", " "))
	}

	return name
}

func (i *Item) IsThrowable() bool {
	return true
}

func (i *Item) IsUsableOrZappable() bool {
	return i.IsUsable() || i.zapEffectName != ""
}

func (i *Item) IsUsable() bool {
	return i.useEffectName != "" || i.IsDocument()
}

func (i *Item) IsDocument() bool {
	return i.category == foundation.ItemCategoryDocuments
}

func (i *Item) GetUseEffectName() string {
	return i.useEffectName
}

func (i *Item) GetZapEffectName() string {
	return i.zapEffectName
}

func (i *Item) IsZappable() bool {
	return i.zapEffectName != ""
}

func (i *Item) Color() color.RGBA {
	return color.RGBA{255, 255, 255, 255}
}

func (i *Item) CanStackWith(other *Item) bool {
	if i.name != other.name || i.category != other.category {
		return false
	}

	if (i.IsWeapon() && !i.IsGroupWeapon()) || i.IsArmor() || i.IsRing() || (other.IsWeapon() && !other.IsGroupWeapon()) || other.IsArmor() || other.IsRing() {
		return false
	}

	if i.useEffectName != other.useEffectName || i.zapEffectName != other.zapEffectName {
		return false
	}

	if i.charges != other.charges || i.found != other.found || i.IsGroupWeapon() && (i.weapon.hitPlus != other.weapon.hitPlus || i.weapon.damagePlus != other.weapon.damagePlus) {
		return false
	}

	return true
}

func (i *Item) SlotName() foundation.EquipSlot {
	return i.slot
}

func (i *Item) IsEquippable() bool {
	return i.slot != foundation.SlotNameNotEquippable
}

func (i *Item) IsMeleeWeapon() bool {
	return i.IsWeapon() && (i.slot == foundation.SlotNameOneHandedWeapon || i.slot == foundation.SlotNameTwoHandedWeapon)
}

func (i *Item) IsRangedWeapon() bool {
	return i.IsWeapon() && i.slot == foundation.SlotNameMissileLauncher
}

func (i *Item) IsArmor() bool {
	return i.armor != nil
}

func (i *Item) IsTwoHandedWeapon() bool {
	return i.IsWeapon() && i.slot == foundation.SlotNameTwoHandedWeapon
}

func (i *Item) IsWeapon() bool {
	return i.weapon != nil
}

func (i *Item) GetCategory() foundation.ItemCategory {
	return i.category
}

func (i *Item) GetWeapon() *WeaponInfo {
	return i.weapon
}

func (i *Item) GetArmor() *ArmorInfo {
	return i.armor
}

func (i *Item) IsPotion() bool {
	return i.category == foundation.ItemCategoryPotions
}

func (i *Item) IsGold() bool {
	return i.category == foundation.ItemCategoryGold
}

func (i *Item) IsScareMonster() bool {
	return i.internalName == "scare_monster"
}

func (i *Item) GetCharges() int {
	return i.charges
}

func (i *Item) IsFood() bool {
	return i.category == foundation.ItemCategoryFood
}

func (i *Item) IsMagic() bool { // potions, scrolls, wands & weapons/armor with plusses

	isConsumableMagic := i.IsPotion() || i.IsWand() || i.IsScroll()

	isEnchantedWeapon := i.IsWeapon() && i.weapon.IsEnchanted()

	isEnchantedArmor := i.IsArmor() && i.armor.IsEnchanted()

	isMagicRing := i.IsRing()

	return isConsumableMagic || isEnchantedWeapon || isEnchantedArmor || isMagicRing
}

func (i *Item) IsWand() bool {
	return i.category == foundation.ItemCategoryWands
}

func (i *Item) IsScroll() bool {
	return i.category == foundation.ItemCategoryScrolls
}

func (i *Item) IsRing() bool {
	return i.category == foundation.ItemCategoryRings
}

func (i *Item) GetInternalName() string {
	return i.internalName
}

func (i *Item) GetStatBonus(stat rpg.Stat) int {

	if i.stat == stat {
		return i.statBonus
	}
	return 0
}
func (i *Item) GetEquipFlag() foundation.ActorFlag {
	if i.IsRing() && i.charges == 0 {
		return foundation.FlagNone
	}
	return i.equipFlag
}

// IsGroupWeapon: missiles and daggers come in stacking groups (Rogue's ISMANY and dagger o_count)
func (i *Item) IsGroupWeapon() bool {
	return i.IsMissile() || i.IsWeapon() && i.GetWeapon().GetWeaponType() == ItemTypeDagger
}

func (i *Item) IsMissile() bool {
	return i.IsWeapon() && i.GetWeapon().GetWeaponType().IsMissile()
}

func (i *Item) GetThrowDamageDice() rpg.Dice {
	return i.thrownDamage
}

func (i *Item) SetCharges(amount int) {
	i.charges = amount
}

func (i *Item) AfterEquippedTurn() {
	if i.IsRing() && i.charges > 0 {
		i.charges--
		i.isKnown = true
	}
	if i.IsStuck() {
		i.stuckTurns--
		i.isKnown = true
	}
}

func (i *Item) IsStuck() bool {
	return i.stuckTurns > 0
}

func (i *Item) IsCursed() bool {
	return i.IsStuck() || i.statBonus < 0
}

func (i *Item) RemoveCurse() {
	if !i.IsCursed() {
		return
	}
	blessing := rand.Intn(100) < 4

	i.stuckTurns = 0
	if i.statBonus < 0 {
		if blessing {
			i.statBonus = rand.Intn(3) + 1
		} else {
			i.statBonus = 0
		}
	}
}

// Rust lowers the armor's protection by one; returns false if there is nothing left to rust.
func (i *ArmorInfo) Rust() bool {
	if i.GetProtection() <= 0 {
		return false
	}
	i.plus--
	return true
}
