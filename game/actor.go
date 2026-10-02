package game

import (
	"fmt"
	"image/color"
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/rpg"
)

// Rogue 3.6 stats. arm is the natural armor class, lower is better.
type Stats struct {
	Str, MaxStr int
	Lvl, Exp    int
	HP, MaxHP   int
	Arm         int
	Dmg         string // "1d4", or one roll per attack "1d8/1d8/3d10"
	FP, MaxFP   int    // fatigue, spent on tactics
}

type Actor struct {
	internalName string
	name         string
	stats        Stats
	onChange     func()
	position     geometry.Point

	inventory *Inventory
	equipment *Equipment

	statusFlags *foundation.MapFlags

	intrinsicZapEffects    []string
	intrinsicUseEffects    []string
	intrinsicHitEffects    []HitEffect
	intrinsicStruckEffects []HitEffect
	intrinsicGazeEffects   []string
	disguise               foundation.ItemCategory

	icon       rune
	color      string
	timeEnergy int
}

func (a *Actor) Color() string {
	return a.color
}

// NewPlayer starts like Rogue: Str 16, level 1, 12 hp, AC 10, 1d4 bare-handed.
func NewPlayer(name string, playerIcon rune, playerColor string) *Actor {
	player := NewActor(name, playerIcon, playerColor)
	player.stats = Stats{Str: 16, MaxStr: 16, Lvl: 1, HP: 12, MaxHP: 12, Arm: 10, Dmg: "1d4", FP: 3, MaxFP: 3}
	return player
}

func NewActor(name string, icon rune, color string) *Actor {
	return &Actor{
		name:        name,
		icon:        icon,
		color:       color,
		inventory:   NewInventory(23),
		equipment:   NewEquipment(),
		statusFlags: foundation.NewMapFlags(),
		stats:       Stats{Lvl: 1, HP: 1, MaxHP: 1, Arm: 10, Dmg: "1d2"},
	}
}

func (a *Actor) changed() {
	if a.onChange != nil {
		a.onChange()
	}
}

func (a *Actor) Icon() rune {
	return a.icon
}

func (a *Actor) IsStunned() bool {
	return a.HasFlag(foundation.FlagStun)
}

func (a *Actor) Position() geometry.Point {
	return a.position
}
func (a *Actor) SetPosition(pos geometry.Point) {
	a.position = pos
}

func (a *Actor) Name() string {
	if a.HasFlag(foundation.FlagInvisible) {
		return "something"
	}
	return a.name
}

func (a *Actor) IsDrawn(playerCanSeeInvisible bool) bool {
	return !a.HasFlag(foundation.FlagInvisible) || playerCanSeeInvisible
}

func (a *Actor) GetInventory() *Inventory {
	return a.inventory
}

func (a *Actor) GetEquipment() *Equipment {
	return a.equipment
}

func (a *Actor) GetFlags() *foundation.MapFlags {
	return a.statusFlags
}

func (a *Actor) HasFlag(flag foundation.ActorFlag) bool {
	return a.statusFlags.IsSet(flag) || a.GetEquipment().ContainsFlag(flag)
}

func (a *Actor) IsSleeping() bool {
	return a.HasFlag(foundation.FlagSleep)
}

func (a *Actor) WakeUp() {
	a.statusFlags.Unset(foundation.FlagSleep)
}

func (a *Actor) SetIntrinsicZapEffects(effects []string) {
	a.intrinsicZapEffects = effects
}

func (a *Actor) SetIntrinsicUseEffects(effects []string) {
	a.intrinsicUseEffects = effects
}

func (a *Actor) GetIntrinsicZapEffects() []string {
	return a.intrinsicZapEffects
}

func (a *Actor) SetIntrinsicHitEffects(effects []HitEffect) {
	a.intrinsicHitEffects = effects
}

func (a *Actor) GetIntrinsicHitEffects() []HitEffect {
	return a.intrinsicHitEffects
}

func (a *Actor) SetIntrinsicGazeEffects(effects []string) {
	a.intrinsicGazeEffects = effects
}

func (a *Actor) GetIntrinsicGazeEffects() []string {
	return a.intrinsicGazeEffects
}

// Disguise returns the item category a mimic is pretending to be.
func (a *Actor) Disguise() (foundation.ItemCategory, bool) {
	return a.disguise, a.HasFlag(foundation.FlagDisguised)
}

func (a *Actor) SetIntrinsicStruckEffects(effects []HitEffect) {
	a.intrinsicStruckEffects = effects
}

func (a *Actor) GetIntrinsicStruckEffects() []HitEffect {
	return a.intrinsicStruckEffects
}

func (a *Actor) GetIntrinsicUseEffects() []string {
	return a.intrinsicUseEffects
}
func (a *Actor) AddGold(i int) {
	a.statusFlags.Increase(foundation.FlagGold, i)
}

func (a *Actor) RemoveLevelStatusEffects() {
	a.statusFlags.Unset(foundation.FlagSeeFood)
	a.statusFlags.Unset(foundation.FlagSeeMagic)
	a.statusFlags.Unset(foundation.FlagSeeTraps)
}

func (a *Actor) GetInternalName() string {
	return a.internalName
}

func (a *Actor) SetInternalName(name string) {
	a.internalName = name
}

func (a *Actor) TextIcon(bg color.RGBA, getColor func(string) color.RGBA) foundation.TextIcon {
	return foundation.TextIcon{
		Rune: a.icon,
		Fg:   getColor(a.color),
		Bg:   bg,
	}
}

func (a *Actor) IsLaunching(missile *Item) bool {
	return missile.IsMissile() && a.GetEquipment().HasMissileLauncherEquippedForMissile(missile.GetWeapon())
}

func (a *Actor) HasGold(price int) bool {
	return a.statusFlags.Get(foundation.FlagGold) >= price
}

func (a *Actor) RemoveGold(price int) {
	a.statusFlags.Decrease(foundation.FlagGold, price)
}

func (a *Actor) GetGold() int {
	return a.statusFlags.Get(foundation.FlagGold)
}

func (a *Actor) NeedsHealing() bool {
	return a.GetHitPoints() < a.GetHitPointsMax()
}

func (a *Actor) IsHungry() bool {
	return a.statusFlags.Get(foundation.FlagHunger) > 0
}

func (a *Actor) Satiate() {
	a.statusFlags.Unset(foundation.FlagHunger)
	a.statusFlags.Unset(foundation.FlagTurnsSinceEating)
}

func (a *Actor) SetSleeping() {
	flags := a.GetFlags()
	flags.Set(foundation.FlagSleep)
	flags.Unset(foundation.FlagAwareOfPlayer)
	flags.Unset(foundation.FlagScared)
}

func (a *Actor) SetAware() {
	flags := a.GetFlags()
	flags.Unset(foundation.FlagSleep)
	flags.Set(foundation.FlagAwareOfPlayer)
}

func (a *Actor) IsBlind() bool {
	return a.HasFlag(foundation.FlagBlind)
}

func (a *Actor) AddTimeEnergy(timeSpent int) {
	a.timeEnergy += timeSpent
}

func (a *Actor) HasEnergyForActions() bool {
	return a.timeEnergy >= a.timeNeededForActions()
}

func (a *Actor) timeNeededForActions() int {
	speed := a.GetBasicSpeed()
	timeNeeded := 100 / speed
	return timeNeeded
}

func (a *Actor) SpendTimeEnergy() {
	a.timeEnergy -= a.timeNeededForActions()
}

func (a *Actor) AfterTurn() {
	a.GetEquipment().AfterTurn()
}

func (a *Actor) GetListInfo() string {
	return fmt.Sprintf("%s HP: %d/%d Dmg: %s Armor: %d", a.name, a.stats.HP, a.stats.MaxHP, a.stats.Dmg, a.GetArmor())
}

// GetArmor is the armor value that is shown, higher is better: how far the armor class is below the unarmored 10.
func (a *Actor) GetArmor() int {
	return 10 - a.GetArmorClass()
}

// GetArmorClass is Rogue's AC, lower is better. Worn armor and rings of protection lower it.
// It is only used for the to-hit roll, never shown.
func (a *Actor) GetArmorClass() int {
	ac := a.stats.Arm
	for _, armorPiece := range a.GetEquipment().GetArmor() {
		ac = min(ac, 10) - armorPiece.GetArmor().GetProtection()
	}
	return ac - a.GetEquipment().GetStatModifier(rpg.StatArmor)
}

func (a *Actor) IsAlive() bool {
	return a.stats.HP > 0
}

func (a *Actor) TakeDamage(amount int) {
	a.stats.HP -= amount
	a.changed()
}

func (a *Actor) Heal(amount int) {
	a.stats.HP = min(a.stats.MaxHP, a.stats.HP+amount)
	a.changed()
}

func (a *Actor) GetIntrinsicDamageAsString() string {
	return a.stats.Dmg
}

// GetMelee is the to-hit and damage bonus plus the damage dice of a melee attack (Rogue's roll_em).
func (a *Actor) GetMelee(enemyInternalName string) (hplus, dplus int, dmg string) {
	hplus = a.GetEquipment().GetStatModifier(rpg.StatToHit)
	dplus = a.GetEquipment().GetStatModifier(rpg.StatDamage)
	if a.GetEquipment().HasMeleeWeaponEquipped() {
		weapon := a.GetEquipment().GetMainWeapon(MeleeAttack).GetWeapon()
		vh, vd := weapon.GetVorpalBonus(enemyInternalName)
		return hplus + vh + weapon.hitPlus, dplus + vd + weapon.damagePlus, weapon.damageDice.String()
	}
	return hplus, dplus, a.stats.Dmg
}

// GetThrowing: a thrown item uses its thrown damage, a missile shot from its launcher adds the launcher's plusses.
func (a *Actor) GetThrowing(enemyInternalName string, missile *Item) (hplus, dplus int, dmg string) {
	hplus = a.GetEquipment().GetStatModifier(rpg.StatToHit)
	dplus = a.GetEquipment().GetStatModifier(rpg.StatDamage)
	if !missile.IsWeapon() {
		return hplus, dplus, missile.GetThrowDamageDice().String()
	}
	weapon := missile.GetWeapon()
	vh, vd := weapon.GetVorpalBonus(enemyInternalName)
	hplus, dplus = hplus+vh+weapon.hitPlus, dplus+vd+weapon.damagePlus
	if a.IsLaunching(missile) {
		launcher := a.GetEquipment().GetMissileLauncher().GetWeapon()
		return hplus + launcher.hitPlus, dplus + launcher.damagePlus, weapon.damageDice.String()
	}
	return hplus, dplus, missile.GetThrowDamageDice().String()
}

func (a *Actor) GetHitPoints() int {
	return a.stats.HP
}

func (a *Actor) GetHitPointsMax() int {
	return a.stats.MaxHP
}

func (a *Actor) GetLevel() int {
	return a.stats.Lvl
}

func (a *Actor) GetExperience() int {
	return a.stats.Exp
}

// GetBasicSpeed: everyone moves at the same pace in Rogue, unless hasted or slowed.
func (a *Actor) GetBasicSpeed() int {
	if a.HasFlag(foundation.FlagSlow) {
		return 5
	} else if a.HasFlag(foundation.FlagHaste) {
		return 20
	}
	return 10
}

func (a *Actor) GetDetailInfo() []string {
	s := a.stats
	result := []string{
		fmt.Sprintf("Name: %s", a.Name()),
		"",
		fmt.Sprintf("Level: %d", s.Lvl),
		fmt.Sprintf("Exp:   %d", s.Exp),
		fmt.Sprintf("HP:    %d/%d", s.HP, s.MaxHP),
		fmt.Sprintf("Str:   %d/%d", a.GetStrength(), s.MaxStr),
		fmt.Sprintf("Armor: %d", a.GetArmor()),
	}
	if s.MaxFP > 0 {
		result = append(result, fmt.Sprintf("FP:    %d/%d", s.FP, s.MaxFP))
	}
	_, dplus, dmg := a.GetMelee("")
	result = append(result, fmt.Sprintf("Dmg:   %s%+d", dmg, dplus))

	if len(a.intrinsicUseEffects) > 0 || len(a.intrinsicZapEffects) > 0 {
		result = append(result, "", "> Abilities:")
		result = append(result, a.intrinsicUseEffects...)
		result = append(result, a.intrinsicZapEffects...)
	}
	return result
}

func (a *Actor) GetFatiguePoints() int {
	return a.stats.FP
}

func (a *Actor) GetFatiguePointsMax() int {
	return a.stats.MaxFP
}

func (a *Actor) LooseFatigue(amount int) {
	a.stats.FP = max(0, a.stats.FP-amount)
	a.changed()
}

func (a *Actor) AddFatiguePoints(amount int) {
	a.stats.FP = min(a.stats.MaxFP, a.stats.FP+amount)
	a.changed()
}

// GetStrength includes rings of strength.
func (a *Actor) GetStrength() int {
	return a.stats.Str + a.GetEquipment().GetStatModifier(rpg.StatStrength)
}

// ChangeStrength is Rogue's chg_str: max strength only ever goes up.
func (a *Actor) ChangeStrength(amount int) {
	a.stats.Str = min(18, max(3, a.stats.Str+amount))
	a.stats.MaxStr = max(a.stats.MaxStr, a.stats.Str)
	a.changed()
}

func (a *Actor) RestoreStrength() {
	a.stats.Str = max(a.stats.Str, a.stats.MaxStr)
	a.changed()
}

// AddExperience returns the new level when one was reached (Rogue's check_level).
func (a *Actor) AddExperience(amount int) (newLevel int, leveledUp bool) {
	a.stats.Exp += amount
	lvl := rpg.LevelForExp(a.stats.Exp)
	if lvl > a.stats.Lvl {
		add := rpg.NewDice(lvl-a.stats.Lvl, 10, 0).Roll()
		a.stats.MaxHP += add
		a.stats.HP += add
		a.stats.Lvl = lvl
		leveledUp = true
	}
	a.changed()
	return lvl, leveledUp
}

// RaiseLevel: just enough experience for the next level.
func (a *Actor) RaiseLevel() int {
	lvl, _ := a.AddExperience(rpg.ExpLevels[min(a.stats.Lvl, len(rpg.ExpLevels))-1] + 1 - a.stats.Exp)
	return lvl
}

// DrainLevel loses one experience level and its hit points.
func (a *Actor) DrainLevel() {
	if a.stats.Lvl > 1 {
		a.stats.Lvl--
		a.stats.Exp = 0
		if a.stats.Lvl > 1 {
			a.stats.Exp = rpg.ExpLevels[a.stats.Lvl-2] + 1
		}
	} else {
		a.stats.Exp = 0
	}
	lost := rpg.NewDice(1, 10, 0).Roll()
	a.stats.MaxHP = max(1, a.stats.MaxHP-lost)
	a.stats.HP = min(a.stats.HP, a.stats.MaxHP)
	a.changed()
}

func (a *Actor) DrainMaxHP(amount int) {
	a.stats.MaxHP = max(1, a.stats.MaxHP-amount)
	a.stats.HP = min(a.stats.HP, a.stats.MaxHP)
	a.changed()
}

// CanPerceivePlayer: Rogue monsters notice you two times out of three.
func (a *Actor) CanPerceivePlayer() bool {
	return rand.Intn(3) != 0
}
