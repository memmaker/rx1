package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

type stubUI struct{ foundation.GameUI }

func (stubUI) SetGame(foundation.GameForUI)         {}
func (stubUI) UpdateVisibleEnemies()                {}
func (stubUI) UpdateStats()                         {}
func (stubUI) UpdateLogWindow()                     {}
func (stubUI) UpdateInventory()                     {}
func (stubUI) AfterPlayerMoved(foundation.MoveInfo) {}
func (stubUI) InitDungeonUI()                       {}
func (stubUI) AddAnimations([]foundation.Animation) {}

func TestLightOnlyLightsLineOfSight(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	for n := 0; n < 40; n++ {
		g := NewGameState(stubUI{}, cfg)
		g.GotoDungeonLevel(1, StairsBoth, true)
		m := g.gridMap
		// walk the player over every walkable tile and check light vs. line of sight
		for y := 0; y < m.GetHeight(); y++ {
			for x := 0; x < m.GetWidth(); x++ {
				p := geometry.Point{X: x, Y: y}
				if !m.IsWalkable(p) || m.IsActorAt(p) {
					continue
				}
				g.Player.SetPosition(p)
				g.exploreMap()
				r := g.playerLightRadius()
				for yy := 0; yy < m.GetHeight(); yy++ {
					for xx := 0; xx < m.GetWidth(); xx++ {
						q := geometry.Point{X: xx, Y: yy}
						if !g.canPlayerSee(q) || q == p || g.IsLit(q) {
							continue
						}
						if geometry.DistanceSquared(p, q) > r*r {
							t.Fatalf("%s sees %s beyond radius %d", p, q, r)
						}
						if !m.IsLineOfSightClear(p, q) && geometry.DistanceChebyshev(p, q) > 1 {
							t.Fatalf("seed %d: %v sees %v without LOS (radius %d)", n, p, q, r)
						}
					}
				}
			}
		}
	}
}
