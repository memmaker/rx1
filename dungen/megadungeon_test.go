package dungen

import (
	"math/rand"
	"rx1/geometry"
	"testing"
	"time"
)

// Every mega dungeon is one connected piece without dead ends, and every door leads somewhere.
func TestMegaDungeonIsConnected(t *testing.T) {
	var total time.Duration
	for seed := int64(1); seed <= 40; seed++ {
		start := time.Now()
		m := NewMegaDungeonGenerator(rand.New(rand.NewSource(seed))).Generate(160, 70)
		total += time.Since(start)
		if len(m.rooms) < 10 {
			t.Fatalf("seed %d: only %d rooms", seed, len(m.rooms))
		}
		walkable := 0
		for i := range m.tiles {
			if m.tiles[i] != Wall {
				walkable++
			}
		}
		seen := map[geometry.Point]bool{}
		open := []geometry.Point{m.rooms[0].GetAbsoluteFloorTiles()[0]}
		seen[open[0]] = true
		for len(open) > 0 {
			pos := open[len(open)-1]
			open = open[:len(open)-1]
			if _, isDeadEnd := m.IsDeadEnd(pos); isDeadEnd {
				t.Fatalf("seed %d: dead end at %v", seed, pos)
			}
			neighbours := m.GetFilteredCardinalNeighbours(pos, m.IsWalkable)
			if m.IsDoorAt(pos) && len(neighbours) != 2 {
				t.Fatalf("seed %d: door at %v has %d ways", seed, pos, len(neighbours))
			}
			for _, n := range neighbours {
				if !seen[n] {
					seen[n] = true
					open = append(open, n)
				}
			}
		}
		if len(seen) != walkable {
			t.Fatalf("seed %d: reached %d of %d walkable tiles", seed, len(seen), walkable)
		}
		for _, room := range m.rooms {
			for _, wall := range room.GetWalls() {
				if !m.IsWallAt(wall) {
					t.Fatalf("seed %d: room wall at %v is not a wall", seed, wall)
				}
			}
		}
	}
	t.Logf("%v per dungeon", total/40)
}
