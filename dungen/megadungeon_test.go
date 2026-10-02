package dungen

import (
	"math/rand"
	"rx1/geometry"
	"testing"
	"time"
)

// Every mega dungeon is one connected piece without dead ends, and the rooms keep their walls.
func TestMegaDungeonIsConnected(t *testing.T) {
	var total time.Duration
	for seed := int64(1); seed <= 40; seed++ {
		generator := NewMegaDungeonGenerator(rand.New(rand.NewSource(seed)), 300)
		if seed%2 == 0 { // Hauberk's goblin stronghold
			generator.WindingPercent, generator.RoomExtraSize = 70, 1
		}
		start := time.Now()
		m := generator.Generate(160, 70)
		total += time.Since(start)
		if len(m.rooms) < 10 {
			t.Fatalf("seed %d: only %d rooms", seed, len(m.rooms))
		}
		exits := func(pos geometry.Point) (open []geometry.Point) {
			for _, dir := range cardinalDirections {
				if m.IsWalkable(pos.Add(dir)) {
					open = append(open, pos.Add(dir))
				}
			}
			return open
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
			if len(exits(pos)) < 2 {
				t.Fatalf("seed %d: dead end at %v", seed, pos)
			}
			for _, n := range exits(pos) {
				if !seen[n] {
					seen[n] = true
					open = append(open, n)
				}
			}
		}
		if len(seen) != walkable {
			t.Fatalf("seed %d: reached %d of %d walkable tiles", seed, len(seen), walkable)
		}
		for i, room := range m.rooms {
			for _, wall := range room.GetWalls() {
				if !m.IsWallAt(wall) {
					t.Fatalf("seed %d: room wall at %v is not a wall", seed, wall)
				}
			}
			for _, other := range m.rooms[:i] {
				if rectDistance(room.bounds, other.bounds) <= 0 {
					t.Fatalf("seed %d: rooms %v and %v overlap or touch", seed, room.bounds, other.bounds)
				}
			}
		}
	}
	t.Logf("%v per dungeon", total/40)
}

// To look at one: go test ./dungen -run TestMegaDungeonPrint -v
func TestMegaDungeonPrint(t *testing.T) {
	if !testing.Verbose() {
		t.Skip()
	}
	generator := NewMegaDungeonGenerator(rand.New(rand.NewSource(3)), 300)
	generator.WindingPercent, generator.RoomExtraSize = 70, 1
	generator.Generate(119, 45).Print()
}
