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
	room := g.dungeonLayout.AllRooms()[0]
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
	g.aiAct(lurkers[0]) // the hero is elsewhere: it stays hidden
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
		if g.lurksInWall(a) || a.HasFlag(foundation.FlagInvisible) || !room.FloorContains(a.Position()) {
			t.Fatal("a lurker should step out of the wall into the room")
		}
	}
}
