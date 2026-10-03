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

// emergeFromWall: the wall fades into the monster, which steps into the room if it can and attacks.
func (g *GameState) emergeFromWall(a *Actor) {
	room := g.getPlayerRoom()
	wall := a.Position()
	if room == nil || !room.FloorContains(g.Player.Position()) || !room.ContainsIncludingWalls(wall) {
		return
	}
	a.GetFlags().Unset(foundation.FlagInvisible)
	a.GetFlags().Set(foundation.FlagAwareOfPlayer)
	a.GetFlags().Set(foundation.FlagChase)
	g.msg(foundation.HiLite("%s steps out of the wall", a.Name()))
	uncloak, _ := g.ui.GetAnimUncloakAtPosition(a, wall)
	exits := slices.DeleteFunc(lurkerExits(room, wall), g.gridMap.IsActorAt)
	if len(exits) > 0 { // with its way out blocked it fights from the wall
		floor := exits[rand.Intn(len(exits))]
		move := g.ui.GetAnimMove(a, wall, floor)
		g.gridMap.MoveActor(a, floor)
		if uncloak != nil && move != nil {
			move.RequestMapUpdateOnFinish()
			uncloak.SetFollowUp([]foundation.Animation{move})
		}
	}
	if uncloak != nil { // nil: animations switched off
		g.ui.AddAnimations([]foundation.Animation{uncloak})
	}
}

// fleeIntoStone: D&D xorn, when the fight goes against it, sinks into the rock and moves away through it.
func (g *GameState) fleeIntoStone(a *Actor) bool {
	best, bestDist := a.Position(), geometry.DistanceChebyshev(a.Position(), g.Player.Position())
	if g.gridMap.IsTileWalkable(best) {
		bestDist = -1 // still on the floor: any stone will do
	}
	for _, n := range g.gridMap.NeighborsAll(a.Position(), func(p geometry.Point) bool {
		return g.gridMap.Contains(p) && !g.gridMap.IsTileWalkable(p) && !g.gridMap.IsActorAt(p)
	}) {
		if d := geometry.DistanceChebyshev(n, g.Player.Position()); d > bestDist {
			best, bestDist = n, d
		}
	}
	if best == a.Position() {
		return false
	}
	if !a.HasFlag(foundation.FlagScared) {
		a.GetFlags().Set(foundation.FlagScared)
		g.msg(foundation.HiLite("%s sinks into the stone", a.Name()))
	}
	g.gridMap.ForceMoveActor(a, best) // wall crawlers pass into rock like this (phaseTowards)
	return true
}
