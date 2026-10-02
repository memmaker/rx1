package game

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/rpg"
	"strings"
)

// The save file is text, one section per line: "<name> <crc32 hex> <json>". A section that fails its checksum
// (bit rot, a cut-off last line) is skipped and the game's fresh defaults stand in for it. Sections are tolerant JSON:
// unknown fields and sections are ignored, missing ones keep their new-game value, items and flags are stored by name
// and dropped when the game no longer knows them. Only the hero section is required.
//
// ponytail: a checkpoint, not a snapshot. Levels are not stored: loading builds a fresh level at the saved depth, so
// monsters, floor items and exploration are lost. Store the map (tiles, actors, objects' closures) if that matters.
// Bump saveHeader and migrate in the section decoders when a field changes meaning rather than shape.
const (
	saveFile   = "save.rx1"
	saveHeader = "rx1save 1"
)

type saveItem struct {
	Name      string   `json:"name"`            // internal name, the definition supplies the rest
	Label     string   `json:"label,omitempty"` // display name, only when it differs from the definition (rusty)
	Charges   int      `json:"charges"`
	StatBonus int      `json:"stat_bonus"`
	Stat      rpg.Stat `json:"stat,omitempty"`
	Stuck     int      `json:"stuck,omitempty"`
	Known     bool     `json:"known,omitempty"`
	Found     bool     `json:"found,omitempty"`
	Bundle    int      `json:"bundle,omitempty"`
	HitPlus   int      `json:"hit_plus,omitempty"`
	DamPlus   int      `json:"damage_plus,omitempty"`
	Vorpal    string   `json:"vorpal,omitempty"`
	ArmorPlus int      `json:"armor_plus,omitempty"`
	Equipped  bool     `json:"equipped,omitempty"`
}

type savePlayer struct {
	Stats Stats             `json:"stats"`
	Flags map[string]int    `json:"flags"`
	Items []json.RawMessage `json:"items"` // each decoded alone: one bad item costs one item
}

type saveWorld struct {
	Depth             int             `json:"depth"`
	Turns             int             `json:"turns"`
	Deepest           int             `json:"deepest"`
	LevelsWithoutFood int             `json:"levels_without_food"`
	LightsRolledUpTo  int             `json:"lights_rolled_up_to"`
	StarGlass         bool            `json:"star_glass"`
	MorningStar       bool            `json:"morning_star"`
	Genocided         map[string]bool `json:"genocided"`
	SecretDepth       int             `json:"secret_depth"`
	SecretVisited     bool            `json:"secret_visited"`
	LevelStyles       []string        `json:"level_styles"`
	WanderTurn        int             `json:"wander_turn"`
	WanderCooldown    int             `json:"wander_cooldown"`
	NoMove            int             `json:"no_move"`
	NoCommand         int             `json:"no_command"`
	UsedDocuments     map[string]bool `json:"used_documents"`
}

type saveIdent struct {
	Identified map[string]bool   `json:"identified"`
	Potions    map[string]string `json:"potions"`
	Rings      map[string]string `json:"rings"`
	Scrolls    map[string]string `json:"scrolls"`
	Wands      map[string]string `json:"wands"`
}

// savedFlags names the hero's persistent flags. The numbers are iota order, which a new flag in the middle would shift.
var savedFlags = map[string]foundation.ActorFlag{
	"sleep": foundation.FlagSleep, "stun": foundation.FlagStun, "slow": foundation.FlagSlow, "haste": foundation.FlagHaste,
	"held": foundation.FlagHeld, "fly": foundation.FlagFly, "gold": foundation.FlagGold, "cancel": foundation.FlagCancel,
	"blind": foundation.FlagBlind, "confused": foundation.FlagConfused, "invisible": foundation.FlagInvisible,
	"see_monsters": foundation.FlagSeeMonsters, "see_invisible": foundation.FlagSeeInvisible,
	"hallucinating": foundation.FlagHallucinating, "poisoned": foundation.FlagPoisoned,
	"turns_since_eating": foundation.FlagTurnsSinceEating, "hunger": foundation.FlagHunger,
	"slow_digestion": foundation.FlagSlowDigestion, "regenerating": foundation.FlagRegenerating,
}

func (g *GameState) toSaveItems(inv *Inventory) []json.RawMessage {
	var out []json.RawMessage
	for _, i := range inv.Items() {
		s := saveItem{Name: i.internalName, Charges: i.charges, StatBonus: i.statBonus, Stat: i.stat, Stuck: i.stuckTurns,
			Known: i.isKnown, Found: i.found, Bundle: i.bundle, Equipped: g.Player.GetEquipment().IsEquipped(i)}
		if def, ok := g.dataDefinitions.FindItemDef(i.internalName); ok && def.Name != i.name {
			s.Label = i.name
		}
		if i.weapon != nil {
			s.HitPlus, s.DamPlus, s.Vorpal = i.weapon.hitPlus, i.weapon.damagePlus, i.weapon.vorpalEnemy
		}
		if i.armor != nil {
			s.ArmorPlus = i.armor.plus
		}
		raw, _ := json.Marshal(s)
		out = append(out, raw)
	}
	return out
}

func (g *GameState) fromSaveItem(raw json.RawMessage) (*Item, bool, bool) {
	var s saveItem
	if lenientUnmarshal(raw, &s) != nil {
		return nil, false, false
	}
	def, ok := g.dataDefinitions.FindItemDef(s.Name)
	if !ok {
		return nil, false, false
	}
	i := NewItem(def, g.identification)
	i.charges, i.statBonus, i.stuckTurns, i.isKnown, i.found, i.bundle = s.Charges, s.StatBonus, s.Stuck, s.Known, s.Found, s.Bundle
	if s.Stat != rpg.StatNone {
		i.stat = s.Stat
	}
	if s.Label != "" {
		i.name = s.Label
	}
	if i.weapon != nil {
		i.weapon.hitPlus, i.weapon.damagePlus, i.weapon.vorpalEnemy = s.HitPlus, s.DamPlus, s.Vorpal
	}
	if i.armor != nil {
		i.armor.plus = s.ArmorPlus
	}
	return i, s.Equipped, true
}

// lenientUnmarshal keeps what fits: a field of the wrong type is skipped, the rest of the section is still read.
func lenientUnmarshal(data []byte, into any) error {
	err := json.Unmarshal(data, into)
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return nil
	}
	return err
}

func (g *GameState) encodeSave() []byte {
	flags := map[string]int{}
	for name, flag := range savedFlags {
		if v := g.Player.GetFlags().Get(flag); v != 0 {
			flags[name] = v
		}
	}
	styles := make([]string, len(g.levelStyles))
	for i, s := range g.levelStyles {
		styles[i] = s.String()
	}
	id := g.identification
	var stash []json.RawMessage
	if g.stash != nil {
		stash = g.toSaveItems(g.stash)
	}
	sections := []struct {
		name string
		data any
	}{
		{"player", savePlayer{g.Player.stats, flags, g.toSaveItems(g.Player.GetInventory())}},
		{"world", saveWorld{g.currentDungeonLevel, g.TurnsTaken, g.deepestDungeonLevelPlayerReached, g.levelsWithoutFood,
			g.lightsRolledUpTo, g.starGlassSpawned, g.morningStarSpawned, g.genocided, g.secretLevelDepth, g.secretLevelVisited,
			styles, g.wanderingMonsterTurn, g.wanderingCooldown, g.noMove, g.noCommand, g.usedDocuments}},
		{"ident", saveIdent{id.identifiedItemTypes, id.potionMap, id.ringMap, id.scrollMap, id.wandMap}},
		{"stash", stash},
	}
	var b strings.Builder
	b.WriteString(saveHeader + "\n")
	for _, s := range sections {
		raw, _ := json.Marshal(s.data)
		fmt.Fprintf(&b, "%s %08x %s\n", s.name, crc32.ChecksumIEEE(raw), raw)
	}
	return []byte(b.String())
}

// writeFileAtomic: a crash leaves the old file or the new one, never half of one. The old file is kept as .bak.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync() // on disk before it replaces anything
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		if err := os.Rename(path, path+".bak"); err != nil {
			os.Remove(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		return err // the .bak still holds the last good save, and load reads it
	}
	if d, err := os.Open(filepath.Dir(path)); err == nil { // make the renames durable
		d.Sync()
		d.Close()
	}
	return nil
}

// readSections returns the sections of a save file whose checksum holds.
func readSections(path string) map[string][]byte {
	sections := map[string][]byte{}
	f, err := os.Open(path)
	if err != nil {
		return sections
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		if name, rest, ok := strings.Cut(strings.TrimRight(line, "\r\n"), " "); ok && name != "rx1save" {
			if sum, data, ok := strings.Cut(rest, " "); ok && fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(data))) == sum {
				sections[name] = []byte(data)
			}
		}
		if err != nil {
			break
		}
	}
	return sections
}

func (g *GameState) SaveGame() {
	if err := writeFileAtomic(saveFile, g.encodeSave()); err != nil {
		g.msg(foundation.Msg("Saving failed: " + err.Error()))
		return
	}
	g.msg(foundation.Msg("Game saved"))
}

func (g *GameState) LoadGame() {
	sections := readSections(saveFile + ".bak") // the newer file wins section by section
	for name, data := range readSections(saveFile) {
		sections[name] = data
	}
	if err := g.restore(sections); err != nil {
		g.msg(foundation.Msg("Loading failed: " + err.Error()))
		return
	}
	g.msg(foundation.Msg("Game loaded"))
}

// restore starts a new game, lays the saved sections over it and enters the saved depth. It changes nothing when the
// hero section is missing.
func (g *GameState) restore(sections map[string][]byte) error {
	var p savePlayer
	if raw, ok := sections["player"]; !ok || lenientUnmarshal(raw, &p) != nil {
		return errors.New("no readable save file")
	}
	g.init()
	if p.Stats.MaxHP > 0 { // a hero without hit points would be dead on arrival: keep the new-game stats then
		g.Player.stats = p.Stats
	}
	flags := map[foundation.ActorFlag]int{}
	for name, v := range p.Flags {
		if flag, ok := savedFlags[name]; ok {
			flags[flag] = v
		}
	}
	g.Player.GetFlags().Init(flags)

	inv, eq := g.Player.GetInventory(), g.Player.GetEquipment()
	invChanged, eqChanged := inv.onChanged, eq.onChanged
	inv.onChanged, eq.onChanged = nil, nil // no UI updates before there is a level
	inv.items, eq.slots = nil, map[foundation.EquipSlot]*Item{}
	g.restoreIdentification(sections)
	for _, raw := range p.Items {
		if item, equipped, ok := g.fromSaveItem(raw); ok {
			inv.Add(item)
			if equipped {
				eq.Equip(item)
			}
		}
	}
	inv.onChanged, eq.onChanged = invChanged, eqChanged

	if raw, ok := sections["stash"]; ok {
		var items []json.RawMessage
		if lenientUnmarshal(raw, &items) == nil && len(items) > 0 {
			g.stash = NewInventory(99)
			for _, r := range items {
				if item, _, ok := g.fromSaveItem(r); ok {
					g.stash.Add(item)
				}
			}
		}
	}

	w := saveWorld{g.currentDungeonLevel, g.TurnsTaken, g.deepestDungeonLevelPlayerReached, g.levelsWithoutFood,
		g.lightsRolledUpTo, g.starGlassSpawned, g.morningStarSpawned, g.genocided, g.secretLevelDepth, g.secretLevelVisited,
		nil, g.wanderingMonsterTurn, g.wanderingCooldown, g.noMove, g.noCommand, g.usedDocuments}
	if raw, ok := sections["world"]; ok {
		lenientUnmarshal(raw, &w)
	}
	g.TurnsTaken, g.deepestDungeonLevelPlayerReached, g.levelsWithoutFood = w.Turns, w.Deepest, w.LevelsWithoutFood
	g.lightsRolledUpTo, g.starGlassSpawned, g.morningStarSpawned = w.LightsRolledUpTo, w.StarGlass, w.MorningStar
	g.secretLevelDepth, g.secretLevelVisited = w.SecretDepth, w.SecretVisited
	g.wanderingMonsterTurn, g.wanderingCooldown, g.noMove, g.noCommand = w.WanderTurn, w.WanderCooldown, w.NoMove, w.NoCommand
	if w.Genocided != nil {
		g.genocided = w.Genocided
	}
	if w.UsedDocuments != nil {
		g.usedDocuments = w.UsedDocuments
	}
	for i, name := range w.LevelStyles { // a changed number of levels keeps the new plan for the rest
		for _, s := range dungen.AllLevelStyles {
			if i < len(g.levelStyles) && s.String() == name {
				g.levelStyles[i] = s
			}
		}
	}

	g.gridMap = nil // nothing to leave: the saved level is built anew
	g.ui.InitDungeonUI()
	depth := min(max(w.Depth, 0), g.maximumDungeonLevel)
	if depth == 0 {
		g.GotoNamedLevel("town")
	} else {
		g.levelsWithoutFood-- // entering a new level counts one
		g.gotoLevel(depth, StairsBoth, false, false)
	}
	g.ui.UpdateInventory()
	return nil
}

// restoreIdentification keeps the new game's random names for anything the save does not know (a new potion).
func (g *GameState) restoreIdentification(sections map[string][]byte) {
	raw, ok := sections["ident"]
	if !ok {
		return
	}
	var s saveIdent
	lenientUnmarshal(raw, &s)
	id := g.identification
	overlay := func(into, from map[string]string) {
		for name := range into {
			if v, ok := from[name]; ok && v != "" {
				into[name] = v
			}
		}
	}
	overlay(id.potionMap, s.Potions)
	overlay(id.ringMap, s.Rings)
	overlay(id.scrollMap, s.Scrolls)
	overlay(id.wandMap, s.Wands)
	for name := range s.Identified {
		id.identifiedItemTypes[name] = true
	}
}
