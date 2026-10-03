package game

import (
	"math/rand"
	"rx1/dungen"
	"rx1/foundation"
	"rx1/geometry"
	"rx1/gridmap"
	"slices"
)

// Xeroc mk ii: a pack that hides in a room's walls, invisible, and steps out
// of them once the hero stands inside the room.

// placeWallLurkers hides 2-4 of them in the walls of the room, each next to its floor.
func (g *GameState) placeWallLurkers(random *rand.Rand, newMap *gridmap.GridMap[*Actor, *Item, *Object], room *dungen.DungeonRoom, def MonsterDef) {
	var spots []geometry.Point
	for _, w := range room.GetWalls() {
		if !newMap.IsActorAt(w) && len(lurkerExits(room, w)) > 0 {
			spots = append(spots, w)
		}
	}
	random.Shuffle(len(spots), func(i, j int) { spots[i], spots[j] = spots[j], spots[i] })
	for _, w := range spots[:min(len(spots), 2+rnd(random, 3))] {
		newMap.ForceSpawnActorInWall(g.NewEnemyFromDef(def), w)
	}
}

// lurkerExits are the room's floor tiles a lurker in wall w can step onto.
func lurkerExits(room *dungen.DungeonRoom, w geometry.Point) []geometry.Point {
	var exits []geometry.Point
	for _, d := range []geometry.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
		if room.FloorContains(w.Add(d)) {
			exits = append(exits, w.Add(d))
		}
	}
	return exits
}

func (g *GameState) lurksInWall(a *Actor) bool {
	return a.GetInternalName() == "xeroc_2" && a.HasFlag(foundation.FlagInvisible) && !g.gridMap.IsTileWalkable(a.Position())
}

// emergeFromWall: the wall fades into the monster, which steps into the room and attacks.
func (g *GameState) emergeFromWall(a *Actor) {
	room := g.getPlayerRoom()
	wall := a.Position()
	if room == nil || !room.FloorContains(g.Player.Position()) || !room.ContainsIncludingWalls(wall) {
		return
	}
	exits := slices.DeleteFunc(lurkerExits(room, wall), g.gridMap.IsActorAt)
	if len(exits) == 0 {
		return
	}
	floor := exits[rand.Intn(len(exits))]
	a.GetFlags().Unset(foundation.FlagInvisible)
	a.GetFlags().Set(foundation.FlagAwareOfPlayer)
	a.GetFlags().Set(foundation.FlagChase)
	g.msg(foundation.HiLite("%s steps out of the wall", a.Name()))
	uncloak, _ := g.ui.GetAnimUncloakAtPosition(a, wall)
	move := g.ui.GetAnimMove(a, wall, floor)
	g.gridMap.MoveActor(a, floor)
	if uncloak == nil || move == nil { // animations switched off
		return
	}
	move.RequestMapUpdateOnFinish()
	uncloak.SetFollowUp([]foundation.Animation{move})
	g.ui.AddAnimations([]foundation.Animation{uncloak})
}
