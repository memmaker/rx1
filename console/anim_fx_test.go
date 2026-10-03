package console

import (
	"rx1/foundation"
	"rx1/geometry"
	"testing"
)

type unseenGame struct{ foundation.GameForUI }

func (unseenGame) IsVisibleToPlayer(geometry.Point) bool { return false }
func (unseenGame) IsExplored(geometry.Point) bool        { return false }

// A switched off animation plays no frame, yet its done and its follow-ups' run, and what follows them still plays.
func TestNoAnimationRunsDoneAndFollowUpsAtOnce(t *testing.T) {
	a := NewAnimator()
	var calls []string
	first := noAnimation(func() { calls = append(calls, "first") })
	second := noAnimation(func() { calls = append(calls, "second") })
	shown := NewCoverAnimation(geometry.Point{}, foundation.TextIcon{Rune: 'x'}, 2, nil)
	first.SetFollowUp([]foundation.Animation{second})
	second.SetFollowUp([]foundation.Animation{shown})
	a.AddAnimation(first.(TextAnimation))
	a.Flush()
	a.Tick()
	if len(calls) != 2 || a.animationState[geometry.Point{}].Rune != 'x' {
		t.Fatalf("calls %v, drawn %v", calls, a.animationState)
	}
}

// Every effect draws something on some frame and ends.
func TestEveryEffectDraws(t *testing.T) {
	u := &UI{game: unseenGame{}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec"), settings: &foundation.Configuration{AnimationsEnabled: true, AnimateEffects: true}}
	for _, name := range []string{"haste", "slow", "levitate", "see_invisible", "blind", "hallucinate", "polymorph", "detect_food", "detect_magic",
		"detect_monsters", "detect_traps", "light", "darkness", "raise_level", "sleep", "laughter", "red_glow", "cancel", "hold", "invisible"} {
		anim := u.GetAnimEffect(name, geometry.Point{X: 10, Y: 10}, nil, nil).(*FxAnimation)
		drew := false
		for i := 0; !anim.IsDone(); i++ {
			drew = drew || len(anim.GetDrawables()) > 0
			if anim.NextFrame(); i > 200 {
				t.Fatalf("%s never ends", name)
			}
		}
		if !drew || anim.frames < 6 {
			t.Errorf("%s: drew %v in %d frames", name, drew, anim.frames)
		}
	}
}
