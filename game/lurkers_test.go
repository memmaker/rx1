package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"slices"
	"testing"
)

func TestWallLurkersStepOutWhenHeroEnters(t *testing.T) {
	g := newTestGame()
	g.wizardLevelStyle = new(dungen.LevelStyle) // walled rooms
	g.GotoDungeonLevel(10, StairsBoth, true)
	rooms := slices.DeleteFunc(slices.Clone(g.dungeonLayout.AllRooms()), func(r *dungen.DungeonRoom) bool { return len(r.GetWalls()) == 0 })
	room, elsewhere := rooms[0], rooms[1]
	for _, a := range slices.Clone(g.gridMap.Actors()) {
		if a != g.Player && room.ContainsIncludingWalls(a.Position()) {
			g.gridMap.RemoveActor(a)
		}
	}
	def, _ := g.monsterDefByInternalName("xeroc_2")
	g.placeWallLurkers(rand.New(rand.NewSource(1)), g.gridMap, room, def)
	var lurkers []*Actor
	for _, a := range g.gridMap.Actors() {
		if g.lurksInWall(a) {
			lurkers = append(lurkers, a)
		}
	}
	if len(lurkers) < 2 {
		t.Fatalf("%d lurkers in the walls", len(lurkers))
	}
	for _, p := range elsewhere.GetAbsoluteFloorTiles() { // the hero is elsewhere
		if !g.gridMap.IsActorAt(p) {
			g.gridMap.MoveActor(g.Player, p)
			break
		}
	}
	g.aiAct(lurkers[0]) // so it stays hidden
	if !g.lurksInWall(lurkers[0]) {
		t.Fatal("a lurker should wait while the room is empty")
	}
	for _, p := range room.GetAbsoluteFloorTiles() {
		if !g.gridMap.IsActorAt(p) {
			g.gridMap.MoveActor(g.Player, p)
			break
		}
	}
	for _, a := range lurkers {
		g.aiAct(a)
		if g.lurksInWall(a) || a.HasFlag(foundation.FlagInvisible) {
			t.Fatal("a lurker should come out of the wall")
		}
	}
}

func TestXornResistsAndFleesIntoStone(t *testing.T) {
	g := newTestGame()
	g.wizardLevelStyle = new(dungen.LevelStyle)
	g.GotoDungeonLevel(10, StairsBoth, true)
	def, _ := g.monsterDefByInternalName("xorn")
	xorn := g.NewEnemyFromDef(def)
	if fireDamage(xorn, 10) != 0 {
		t.Fatal("fire should not harm a xorn")
	}
	rooms := slices.DeleteFunc(slices.Clone(g.dungeonLayout.AllRooms()), func(r *dungen.DungeonRoom) bool { return len(r.GetWalls()) == 0 })
	room := rooms[0]
	for _, a := range slices.Clone(g.gridMap.Actors()) {
		if a != g.Player && room.ContainsIncludingWalls(a.Position()) {
			g.gridMap.RemoveActor(a)
		}
	}
	for _, w := range room.GetWalls() {
		if exits := lurkerExits(room, w); len(exits) > 0 {
			g.gridMap.AddActor(xorn, exits[0])
			break
		}
	}
	xorn.WakeUp()
	xorn.GetFlags().Set(foundation.FlagAwareOfPlayer)
	xorn.stats.HP, xorn.stats.MaxHP = 1, 30
	g.aiAct(xorn)
	if g.gridMap.IsTileWalkable(xorn.Position()) || !xorn.HasFlag(foundation.FlagScared) {
		t.Fatal("a beaten xorn should sink into the stone")
	}
}
