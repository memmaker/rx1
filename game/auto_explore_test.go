package game

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

// Auto explore must finish a level instead of bouncing between two tiles.
func TestAutoExploreTerminates(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	for n := 0; n < 30; n++ {
		g := NewGameState(stubUI{}, cfg)
		g.GotoDungeonLevel(1+n%10, StairsBoth, true)
		for _, o := range g.gridMap.Objects() { // traps would teleport or hurt the player
			g.gridMap.RemoveObject(o)
		}
		steps := 0
		clearMonsters := func() bool { // wandering monsters spawn mid-run
			for _, a := range g.gridMap.Actors() {
				if a != g.Player {
					g.gridMap.RemoveActor(a)
				}
			}
			return true
		}
		for clearMonsters() && g.AutoExploreStep() || g.gridMap.IsItemAt(g.Player.Position()) {
			if g.gridMap.IsItemAt(g.Player.Position()) {
				g.gridMap.RemoveItemAt(g.Player.Position())
			}
			if steps++; steps > 3000 {
				t.Fatalf("level %d: auto explore still going after %d steps", 1+n%10, steps)
			}
		}
	}
}

func (stubUI) AnimatePending() {}
func (stubUI) ForgetActors()   {}
func (stubUI) ActorMoved(foundation.ActorForUI, geometry.Point, bool) {
}
func (stubUI) EndAnimatedAction(bool)   {}
func (stubUI) AfterAnimations(f func()) { f() }
func (stubUI) GetAnimMove(foundation.ActorForUI, geometry.Point, geometry.Point) foundation.Animation {
	return stubAnim{} // the game always asks a move to update the map
}

type stubAnim struct{}

func (stubAnim) IsDone() bool                                                 { return true }
func (stubAnim) SetFollowUp([]foundation.Animation)                           {}
func (stubAnim) RequestMapUpdateOnFinish()                                    {}
func (stubUI) GetAnimDamage(geometry.Point, int, func()) foundation.Animation { return nil }
func (stubUI) GetAnimBackgroundColor(geometry.Point, string, int, func()) foundation.Animation {
	return nil
}
func (stubUI) GetAnimExplosion([]geometry.Point, func()) foundation.Animation { return nil }
func (stubUI) GetAnimWakeUp(geometry.Point, func()) foundation.Animation      { return nil }
func (stubUI) GetAnimEffect(string, geometry.Point, []geometry.Point, func()) foundation.Animation {
	return nil
}
