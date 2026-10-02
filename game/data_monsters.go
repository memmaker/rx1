package game

import (
	"rx1/foundation"
	"rx1/recfile"
	"rx1/rpg"
)

type HitEffect struct {
	Name   string
	Chance int // percent
}

type MonsterDef struct {
	Name         string
	InternalName string
	Icon         rune
	Color        string

	// Rogue 3.6 monster stats: hit points are level d8
	Level         int
	Armor         int    // armor class, lower is better
	Damage        string // "1d8/1d8/3d10": one roll per attack
	Exp           int
	ZapEffects    []string
	UseEffects    []string
	HitEffects    []HitEffect
	StruckEffects []HitEffect
	GazeEffects   []string
	DungeonLevel  int
	Flags         *foundation.MapFlags

	CarryChance int
	Gold        rpg.Dice
}

func MonsterDefsFromRecords(records []recfile.Record) []MonsterDef {
	var monsters []MonsterDef
	for _, record := range records {
		monsterDef := NewMonsterDefFromRecord(record)
		monsters = append(monsters, monsterDef)
	}
	return monsters
}

func NewMonsterDefFromRecord(record recfile.Record) MonsterDef {
	monsterDef := MonsterDef{
		Flags: foundation.NewMapFlags(),
	}
	for _, field := range record {
		switch field.Name {
		case "name":
			monsterDef.Name = field.Value
		case "internal_name":
			monsterDef.InternalName = field.Value
		case "letter":
			monsterDef.Icon = []rune(field.Value)[0]
		case "color":
			monsterDef.Color = field.Value
		case "level":
			monsterDef.Level = field.AsInt()
		case "armor":
			monsterDef.Armor = field.AsInt()
		case "damage":
			monsterDef.Damage = field.Value
		case "exp":
			monsterDef.Exp = field.AsInt()
		case "zap_effect":
			monsterDef.ZapEffects = append(monsterDef.ZapEffects, field.Value)
		case "hit_effect":
			fields := field.AsList("|")
			monsterDef.HitEffects = append(monsterDef.HitEffects, HitEffect{Name: fields[0].Value, Chance: fields[1].AsInt()})
		case "struck_effect":
			fields := field.AsList("|")
			monsterDef.StruckEffects = append(monsterDef.StruckEffects, HitEffect{Name: fields[0].Value, Chance: fields[1].AsInt()})
		case "gaze_effect":
			monsterDef.GazeEffects = append(monsterDef.GazeEffects, field.Value)
		case "use_effect":
			monsterDef.UseEffects = append(monsterDef.UseEffects, field.Value)
		case "dlvl":
			monsterDef.DungeonLevel = field.AsInt()
		case "gold":
			monsterDef.Gold = rpg.ParseDice(field.Value)
		case "carry_chance":
			monsterDef.CarryChance = field.AsInt()
		case "flags":
			for _, mFlag := range field.AsList("|") {
				monsterDef.Flags.Set(foundation.ActorFlagFromString(mFlag.Value))
			}
		default:
			println("WARNING: Unknown field: " + field.Name)
		}
	}
	return monsterDef
}
