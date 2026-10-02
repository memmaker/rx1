package rpg

import (
	"math/rand"
	"strings"
)

// Rogue 5.4 combat rules (fight.c, misc.c).

// Swing is Rogue's to-hit roll: d20 + plusses must reach 21 - level - armor class.
func Swing(atLvl, defArm, wplus int) bool {
	return rand.Intn(20)+1+wplus >= 21-atLvl-defArm
}

// Rogue 5.4 strength tables (fight.c), indexed by strength 0..31.
var strPlus = [32]int{
	-7, -6, -5, -4, -3, -2, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1,
	1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3,
}

var addDam = [32]int{
	-7, -6, -5, -4, -3, -2, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 2, 3,
	3, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 6,
}

// MaxStrength is the top of the strength tables.
const MaxStrength = 31

// StrPlus is the to-hit bonus from strength.
func StrPlus(str int) int {
	return strPlus[min(MaxStrength, max(0, str))]
}

// AddDam is the damage bonus from strength.
func AddDam(str int) int {
	return addDam[min(MaxStrength, max(0, str))]
}

// Rogue 5.4 hunger (rogue.h): food_left counts down one per turn.
const (
	HungerTime   = 1300 // a ration, and the food the hero starts with
	StomachSize  = 2000
	HungryAt     = 300 // 2 * MORETIME
	WeakAt       = 150 // MORETIME
	StarveTime   = 850 // dead when food_left drops below -StarveTime
	RationSpread = 400
)

// Saving throw categories.
const (
	VsPoison = 0
	VsMagic  = 3
)

// Save is Rogue's saving throw: d20 >= 14 + which - level/2.
func Save(lvl, which int) bool {
	return rand.Intn(20)+1 >= 14+which-lvl/2
}

// RollAttacks rolls each "XdY" of a Rogue damage string like "1d8/1d8/3d10" as a separate attack.
// hit is called per attack and returns whether it connected; the damage of each hit is summed.
func RollAttacks(dmg string, hit func() bool, dplus int) (damage int, didHit bool) {
	for _, part := range strings.Split(dmg, "/") {
		if !hit() {
			continue
		}
		didHit = true
		damage += max(0, ParseDice(part).Roll()+dplus)
	}
	return damage, didHit
}

// ExpLevels are the experience points needed for each new level.
var ExpLevels = []int{10, 20, 40, 80, 160, 320, 640, 1300, 2600, 5200, 13000, 26000, 50000, 100000, 200000, 400000, 800000, 2000000, 4000000, 8000000}

// LevelForExp is the level a character with exp experience points has.
func LevelForExp(exp int) int {
	for i, need := range ExpLevels {
		if need > exp {
			return i + 1
		}
	}
	return len(ExpLevels) + 1
}

// ExpAdd is the bonus experience a monster is worth on top of its base value.
func ExpAdd(lvl, maxHP int) int {
	mod := maxHP / 6
	if lvl == 1 {
		mod = maxHP / 8
	}
	if lvl > 9 {
		mod *= 20
	} else if lvl > 6 {
		mod *= 4
	}
	return mod
}

// Stat is what a ring or cursed item adds to.
type Stat string

const (
	StatNone     Stat = ""
	StatStrength Stat = "strength"
	StatToHit    Stat = "to_hit"
	StatDamage   Stat = "damage"
	StatArmor    Stat = "armor"
)

func StatFromString(s string) Stat {
	switch st := Stat(strings.ToLower(s)); st {
	case StatStrength, StatToHit, StatDamage, StatArmor:
		return st
	}
	panic("Unknown stat: " + s)
}

func GetRandomStat() Stat {
	return []Stat{StatStrength, StatToHit, StatDamage, StatArmor}[rand.Intn(4)]
}
