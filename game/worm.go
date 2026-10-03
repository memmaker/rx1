package game

import (
	"rx1/foundation"
	"rx1/geometry"
)

// The purple worm is a snake of segments: each segment is an actor on the map that
// passes blows on to the head, follows it step by step, and vanishes when it dies.

const wormStartLength = 2

func newSegment(head *Actor) *Actor {
	seg := NewActor("purple worm tail", '~', head.color)
	seg.SetInternalName("purple_worm_tail")
	seg.GetFlags().Init(head.GetFlags().UnderlyingCopy()) // tunnels where the head tunnels
	seg.stats = head.stats                                // the armor a blow must beat
	seg.head = head
	head.tail = append(head.tail, seg)
	return seg
}

// dragTail moves every segment into the cell the one ahead just left; a growing worm adds one at the end.
func (g *GameState) dragTail(head *Actor, vacated geometry.Point) {
	if geometry.DistanceChebyshev(head.Position(), vacated) > 1 { // teleported: the body follows
		g.coilTail(head)
		return
	}
	for _, seg := range head.tail {
		next := seg.Position()
		g.gridMap.MoveActor(seg, vacated)
		vacated = next
	}
	if head.HasFlag(foundation.FlagGrow) && !g.gridMap.IsActorAt(vacated) {
		head.GetFlags().Decrement(foundation.FlagGrow)
		g.gridMap.ForceSpawnActorInWall(newSegment(head), vacated)
	}
}

func (g *GameState) onMap(a *Actor) bool {
	return a.IsAlive() && g.gridMap.IsActorAt(a.Position()) && g.gridMap.ActorAt(a.Position()) == a
}

// coilTail lays the segments down one after another around the head's new spot.
func (g *GameState) coilTail(head *Actor) {
	prev := head.Position()
	for _, seg := range head.tail {
		for _, n := range g.gridMap.NeighborsAll(prev, func(p geometry.Point) bool {
			return !g.gridMap.IsActorAt(p) && g.gridMap.IsWalkableFor(p, seg)
		}) {
			g.gridMap.MoveActor(seg, n)
			prev = n
			break
		}
	}
}

// actorAt is the actor at pos, the whole worm for one of its segments.
func (g *GameState) actorAt(pos geometry.Point) *Actor {
	a := g.gridMap.ActorAt(pos)
	if a != nil && a.head != nil {
		return a.head
	}
	return a
}
