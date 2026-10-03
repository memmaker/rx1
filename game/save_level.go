package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"strings"
)

// One section per dungeon level ("lvl.<depth>.<secret>"): a damaged level is rebuilt when the hero next enters it,
// the others are not affected. Town is a prefab and is not stored.

type saveActor struct {
	Name       string                  `json:"name"` // internal name, the definition supplies the rest
	Pos        geometry.Point          `json:"pos"`
	Stats      Stats                   `json:"stats"`
	Flags      map[string]int          `json:"flags"`
	Items      []json.RawMessage       `json:"items,omitempty"`
	HoldHits   int                     `json:"hold_hits,omitempty"`
	TimeEnergy int                     `json:"time_energy,omitempty"`
	Tail       []geometry.Point        `json:"tail,omitempty"`
	Disguise   foundation.ItemCategory `json:"disguise,omitempty"` // ponytail: the enum number, a mimic looks odd if the enum shifts
}

type saveFloorItem struct {
	saveItem
	X, Y int
}

type saveTrap struct {
	Kind   string         `json:"kind"` // ObjectCategory.String()
	Pos    geometry.Point `json:"pos"`
	Hidden bool           `json:"hidden"`
	Alive  bool           `json:"alive"`
	Drawn  bool           `json:"drawn"`
}

type saveSecret struct {
	Pos  geometry.Point
	Tile gridmap.Tile
}

type saveGlow struct {
	Pos geometry.Point
	C   color.RGBA
}

type saveLevel struct {
	Depth        int                `json:"depth"`
	Secret       bool               `json:"secret"`
	Style        string             `json:"style"`
	W            int                `json:"w"`
	H            int                `json:"h"`
	Palette      []gridmap.Tile     `json:"palette"`
	Cells        []int              `json:"cells"`    // palette index per cell
	Explored     string             `json:"explored"` // '0'/'1' per cell
	Lit          string             `json:"lit"`
	Layout       *dungen.DungeonMap `json:"layout"`
	Secrets      []saveSecret       `json:"secrets,omitempty"`
	SecretStairs geometry.Point     `json:"secret_stairs"`
	Glow         []saveGlow         `json:"glow,omitempty"`
	Actors       []saveActor        `json:"actors,omitempty"`
	Items        []json.RawMessage  `json:"items,omitempty"`
	Traps        []saveTrap         `json:"traps,omitempty"`
}

func levelSectionName(depth int, secret bool) string {
	return fmt.Sprintf("lvl.%d.%t", depth, secret)
}

func bitString(n int, f func(i int) bool) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = '0'
		if f(i) {
			b[i] = '1'
		}
	}
	return string(b)
}

func (g *GameState) encodeLevel(depth int, secret bool, v *visitedLevel) saveLevel {
	m := v.gridMap
	size := m.MapSize()
	l := saveLevel{Depth: depth, Secret: secret, Style: v.style.String(), W: size.X, H: size.Y, Layout: v.dungeonLayout,
		SecretStairs: v.secretStairs, Cells: make([]int, size.X*size.Y)}
	palette := map[gridmap.Tile]int{}
	cell := func(i int) gridmap.MapCell[*Actor, *Item, *Object] {
		return m.GetCell(geometry.Point{X: i % size.X, Y: i / size.X})
	}
	for i := range l.Cells {
		tile := cell(i).TileType
		idx, ok := palette[tile]
		if !ok {
			idx = len(l.Palette)
			palette[tile] = idx
			l.Palette = append(l.Palette, tile)
		}
		l.Cells[i] = idx
	}
	l.Explored = bitString(len(l.Cells), func(i int) bool { return cell(i).IsExplored })
	l.Lit = bitString(len(l.Cells), func(i int) bool { return cell(i).IsLit })
	for p, t := range v.secrets {
		l.Secrets = append(l.Secrets, saveSecret{p, t})
	}
	for p, c := range v.glowing {
		l.Glow = append(l.Glow, saveGlow{p, c})
	}
	for _, a := range m.Actors() {
		if a == g.Player || !a.IsAlive() || a.head != nil {
			continue
		}
		var tail []geometry.Point
		for _, seg := range a.tail {
			tail = append(tail, seg.Position())
		}
		l.Actors = append(l.Actors, saveActor{a.internalName, a.Position(), a.stats, flagsToSave(a.GetFlags()),
			g.toSaveItems(a.GetInventory()), a.holdHits, a.timeEnergy, tail, a.disguise})
	}
	for _, i := range m.Items() {
		raw, _ := json.Marshal(saveFloorItem{g.toSaveItem(i), i.Position().X, i.Position().Y})
		l.Items = append(l.Items, raw)
	}
	for _, o := range m.Objects() {
		l.Traps = append(l.Traps, saveTrap{o.category.String(), o.Position(), o.isHidden, o.isAlive, o.isDrawn})
	}
	return l
}

func styleFromName(name string) dungen.LevelStyle {
	for _, s := range dungen.AllLevelStyles {
		if s.String() == name {
			return s
		}
	}
	return dungen.StyleRogue
}

// decodeLevel rebuilds a level. It refuses a section whose map does not hang together; single monsters, items and
// traps the game no longer knows are left out. current is the level the hero returns to: the UI follows its actors.
func (g *GameState) decodeLevel(l saveLevel, current bool) (*visitedLevel, error) {
	n := l.W * l.H
	if l.W <= 0 || l.H <= 0 || len(l.Cells) != n || len(l.Explored) != n || len(l.Lit) != n || l.Layout == nil {
		return nil, fmt.Errorf("level %d does not fit its size", l.Depth)
	}
	if w, h := l.Layout.GetSize(); w != l.W || h != l.H {
		return nil, fmt.Errorf("level %d: layout and map differ in size", l.Depth)
	}
	m := gridmap.NewEmptyMap[*Actor, *Item, *Object](l.W, l.H)
	m.SetCardinalMovementOnly(!g.config.DiagonalMovementEnabled)
	for i, idx := range l.Cells {
		if idx < 0 || idx >= len(l.Palette) {
			return nil, fmt.Errorf("level %d: bad tile %d", l.Depth, idx)
		}
		p := geometry.Point{X: i % l.W, Y: i / l.W}
		m.SetTile(p, l.Palette[idx])
		if l.Explored[i] == '1' {
			m.SetExplored(p)
		}
		m.SetLit(p, l.Lit[i] == '1')
	}
	if current {
		g.watchActors(m)
	}
	free := func(p geometry.Point) bool { return m.Contains(p) && !m.IsActorAt(p) }
	for _, sa := range l.Actors {
		def, ok := g.dataDefinitions.FindMonsterDef(sa.Name)
		if !ok || !free(sa.Pos) {
			continue
		}
		a := g.newEnemy(def, false)
		if sa.Stats.MaxHP > 0 {
			a.stats = sa.Stats
		}
		if sa.Flags != nil {
			a.GetFlags().Init(flagsFromSave(sa.Flags))
		}
		for _, raw := range sa.Items {
			if item, _, ok := g.fromSaveItem(raw); ok {
				a.GetInventory().Add(item)
			}
		}
		a.holdHits, a.timeEnergy, a.disguise = sa.HoldHits, sa.TimeEnergy, sa.Disguise
		m.ForceSpawnActorInWall(a, sa.Pos) // keeps a monster inside rock where it was
		for _, p := range sa.Tail {
			if free(p) {
				m.ForceSpawnActorInWall(newSegment(a), p)
			}
		}
	}
	for _, raw := range l.Items {
		var at struct{ X, Y int }
		if item, _, ok := g.fromSaveItem(raw); ok && json.Unmarshal(raw, &at) == nil && m.Contains(geometry.Point{X: at.X, Y: at.Y}) {
			m.AddItem(item, geometry.Point{X: at.X, Y: at.Y})
		}
	}
	for _, st := range l.Traps {
		for _, kind := range foundation.GetAllTrapCategories() {
			if kind.String() == st.Kind && m.Contains(st.Pos) {
				trap := g.NewTrap(kind)
				trap.isHidden, trap.isAlive, trap.isDrawn = st.Hidden, st.Alive, st.Drawn
				m.AddObject(trap, st.Pos)
			}
		}
	}
	if !current {
		g.listenToActors(m) // after filling: these actors are not on the screen
	}
	v := &visitedLevel{gridMap: m, dungeonLayout: l.Layout, style: styleFromName(l.Style), secrets: map[geometry.Point]gridmap.Tile{},
		secretStairs: l.SecretStairs, glowing: map[geometry.Point]color.RGBA{}}
	for _, s := range l.Secrets {
		v.secrets[s.Pos] = s.Tile
	}
	for _, gl := range l.Glow {
		v.glowing[gl.Pos] = gl.C
	}
	return v, nil
}

// levelSections encodes every level the hero has seen, the current one included.
func (g *GameState) levelSections() map[string]saveLevel {
	out := map[string]saveLevel{}
	for key, v := range g.levels {
		out[levelSectionName(key.level, key.secret)] = g.encodeLevel(key.level, key.secret, v)
	}
	if g.gridMap != nil && g.currentDungeonLevel > 0 && g.dungeonLayout != nil {
		cur := &visitedLevel{g.gridMap, g.dungeonLayout, g.levelStyle, g.secrets, g.secretStairs, g.glowing}
		out[levelSectionName(g.currentDungeonLevel, g.inSecretLevel)] = g.encodeLevel(g.currentDungeonLevel, g.inSecretLevel, cur)
	}
	return out
}

// restoreLevels fills g.levels from the sections that decode; the current level is built last so the UI follows it.
func (g *GameState) restoreLevels(sections map[string][]byte, depth int, secret bool) {
	g.levels = map[levelKey]*visitedLevel{}
	for name, raw := range sections {
		var l saveLevel
		if !strings.HasPrefix(name, "lvl.") || lenientUnmarshal(raw, &l) != nil {
			continue
		}
		current := l.Depth == depth && l.Secret == secret
		if v, err := g.decodeLevel(l, current); err == nil {
			g.levels[levelKey{l.Depth, l.Secret}] = v
		}
	}
}
