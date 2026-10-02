package dungen

import "math/rand"

// LevelStyle is the kind of generator that makes a dungeon level.
type LevelStyle int

const (
	StyleRogue  LevelStyle = iota // Rogue's 3x3 grid of rooms
	StyleRooms                    // NetHack's rooms and corridors, with themed room shapes
	StyleCave                     // NetHack's Gnomish Mines caves
	StyleMaze                     // NetHack's mazes
	StyleBrogue                   // Brogue's accreted rooms
)

var AllLevelStyles = []LevelStyle{StyleRogue, StyleRooms, StyleCave, StyleMaze, StyleBrogue}

// styleWeights is how often a level has the style when nothing forces it.
var styleWeights = map[LevelStyle]int{StyleRogue: 30, StyleRooms: 25, StyleCave: 15, StyleMaze: 10, StyleBrogue: 20}

func (s LevelStyle) String() string {
	return [...]string{"Rogue rooms", "NetHack rooms", "Cave", "Maze", "Brogue"}[s]
}

// Generate makes a level of the given style on a map of the given size.
func Generate(style LevelStyle, random *rand.Rand, mapCols, mapRows, level int) *DungeonMap {
	switch style {
	case StyleRooms:
		return NewNetHackGenerator(random, mapCols, mapRows, level).Generate()
	case StyleCave:
		return withLava(random, NewNetHackCaveGenerator(random, mapCols, mapRows, level).Generate(), level)
	case StyleBrogue:
		return withLava(random, NewBrogueGenerator(random, mapCols, mapRows, level).Generate(), level)
	case StyleMaze:
		return NewNetHackMazeGenerator(random, mapCols, mapRows).Generate()
	}
	return NewRogueGenerator(random, mapCols, mapRows, level).Generate()
}

// rogueOpeningLevels are always the classic Rogue style.
const rogueOpeningLevels = 2

// PlanLevelStyles rolls the style of each of the levels (index 0 is level 1) by weight, then gives
// every style that did not come up to a level of a style that did come up more than once, so that
// a whole run through the dungeon sees all of them.
func PlanLevelStyles(random *rand.Rand, levels int) []LevelStyle {
	total := 0
	for _, s := range AllLevelStyles {
		total += styleWeights[s]
	}
	plan := make([]LevelStyle, levels)
	count := map[LevelStyle]int{}
	for i := 0; i < min(rogueOpeningLevels, levels); i++ {
		plan[i] = StyleRogue
		count[StyleRogue]++
	}
	for i := rogueOpeningLevels; i < levels; i++ {
		roll := random.Intn(total)
		for _, s := range AllLevelStyles {
			if roll -= styleWeights[s]; roll < 0 {
				plan[i] = s
				break
			}
		}
		count[plan[i]]++
	}
	for _, missing := range AllLevelStyles {
		for count[missing] == 0 && levels >= len(AllLevelStyles)+rogueOpeningLevels {
			if i := rogueOpeningLevels + random.Intn(levels-rogueOpeningLevels); count[plan[i]] > 1 {
				count[plan[i]]--
				plan[i] = missing
				count[missing]++
			}
		}
	}
	return plan
}
