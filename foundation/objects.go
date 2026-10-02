package foundation

import (
	"math/rand"
	"strings"
)

type ObjectCategory int

const (
	ObjectExplodingTrap ObjectCategory = iota
	ObjectSlowTrap
	ObjectTeleportTrap
	ObjectDartTrap
	ObjectArrowTrap
	ObjectDescendTrap
	ObjectBearTrap
	ObjectSleepTrap
	ObjectRustTrap
	ObjectMysteryTrap
)

func RandomObjectCategory() ObjectCategory {
	return ObjectCategory(rand.Intn(int(ObjectMysteryTrap) + 1))
}

func GetAllTrapCategories() []ObjectCategory {
	return []ObjectCategory{
		ObjectExplodingTrap,
		ObjectSlowTrap,
		ObjectTeleportTrap,
		ObjectDartTrap,
		ObjectArrowTrap,
		ObjectDescendTrap,
		ObjectBearTrap,
		ObjectSleepTrap,
		ObjectRustTrap,
		ObjectMysteryTrap,
	}
}

func (o ObjectCategory) String() string {
	switch o {
	case ObjectExplodingTrap:
		return "Exploding Trap"
	case ObjectSlowTrap:
		return "Slow Trap"
	case ObjectTeleportTrap:
		return "Teleport Trap"
	case ObjectDartTrap:
		return "Dart Trap"
	case ObjectArrowTrap:
		return "Arrow Trap"
	case ObjectDescendTrap:
		return "Descend Trap"
	case ObjectBearTrap:
		return "Bear Trap"
	case ObjectSleepTrap:
		return "Sleeping Gas Trap"
	case ObjectRustTrap:
		return "Rust Trap"
	case ObjectMysteryTrap:
		return "Mystery Trap"

	default:
		return "Unknown"
	}
}

func ObjectCategoryFromString(s string) ObjectCategory {
	switch strings.TrimPrefix(s, "Object") { // themes use both spellings
	case "ExplodingTrap":
		return ObjectExplodingTrap
	case "SlowTrap":
		return ObjectSlowTrap
	case "TeleportTrap":
		return ObjectTeleportTrap
	case "DartTrap":
		return ObjectDartTrap
	case "ArrowTrap":
		return ObjectArrowTrap
	case "DescendTrap":
		return ObjectDescendTrap
	case "BearTrap":
		return ObjectBearTrap
	case "SleepTrap":
		return ObjectSleepTrap
	case "RustTrap":
		return ObjectRustTrap
	case "MysteryTrap":
		return ObjectMysteryTrap
	default:
		return -1
	}
}

func (o ObjectCategory) ZapEffect() string {
	switch o {
	case ObjectExplodingTrap:
		return "explode"
	case ObjectSlowTrap:
		return "slow_target"
	case ObjectTeleportTrap:
		return "teleport_target_away"
	case ObjectDartTrap:
		return "trap_dart"
	case ObjectArrowTrap:
		return "trap_arrow"
	case ObjectDescendTrap:
		return "force_descend_target"
	case ObjectBearTrap:
		return "trap_bear"
	case ObjectSleepTrap:
		return "trap_sleep"
	case ObjectRustTrap:
		return "trap_rust"
	case ObjectMysteryTrap:
		return "trap_mystery"
	default:
		return ""
	}
}

// IsRogueTrap: Rogue's traps stay after they fired, rx1's own exploding and slow traps are used up.
func (o ObjectCategory) IsRogueTrap() bool {
	return o != ObjectExplodingTrap && o != ObjectSlowTrap
}

func (o ObjectCategory) IsTrap() bool {
	return true
}
