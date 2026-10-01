package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/geometry"
	"rx1/gridmap"
	"testing"
)

// Populates many generated levels without a UI to catch panics and empty levels.
func TestRogueLevelPopulation(t *testing.T) {
	g := &GameState{
		dataDefinitions: GetDataDefinitions("../data_rx1"),
		identification:  NewIdentificationKnowledge(),
		usedDocuments:   make(map[string]bool),
		Player:          NewPlayer("tester", '@', "White"),
	}
	items, monsters := 0, 0
	for seed := int64(0); seed < 200; seed++ {
		level := int(seed%26) + 1
		g.deepestDungeonLevelPlayerReached = level
		random := rand.New(rand.NewSource(seed))
		dungeon := dungen.NewRogueGenerator(random, 80, 23, level).Generate()
		w, h := dungeon.GetSize()
		newMap := gridmap.NewEmptyMap[*Actor, *Item, *Object](w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if dungeon.GetTile(x, y) != dungen.Wall {
					newMap.SetTile(geometry.Point{X: x, Y: y}, gridmap.Tile{IsWalkable: true, IsTransparent: true})
				}
			}
		}
		g.spawnEntities(random, level, newMap, dungeon)
		items += len(newMap.Items())
		monsters += len(newMap.Actors())
	}
	t.Logf("avg items %.1f, monsters %.1f per level", float64(items)/200, float64(monsters)/200)
	if monsters == 0 || items == 0 {
		t.Fatal("levels are empty")
	}
}
