package console

import (
	"image/color"
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
		"detect_monsters", "detect_traps", "light", "darkness", "raise_level", "sleep", "laughter", "red_glow", "cancel", "hold", "invisible", "heal", "extra_heal", "gain_strength", "gain_max_hp"} {
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

// A bolt lights the tiles beside its path in its colour, and nothing where the map shows nothing.
func TestBoltCastsLight(t *testing.T) {
	floor := foundation.TextIcon{Rune: '.', Fg: color.RGBA{100, 100, 100, 255}, Bg: color.RGBA{10, 10, 10, 255}}
	a := NewAnimator()
	a.lookup = func(p geometry.Point) (foundation.TextIcon, bool) { return floor, p.Y >= 0 }
	bolt := NewProjectileAnimation([]geometry.Point{{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}}, foundation.TextIcon{Rune: '*'}, nil, nil)
	bolt.SetLight(color.RGBA{255, 0, 0, 255})
	a.AddAnimation(bolt)
	a.Flush()
	a.Tick()
	beside := a.animationState[geometry.Point{X: 0, Y: 2}]
	if beside.Bg.R <= floor.Bg.R || beside.Bg.G != floor.Bg.G {
		t.Fatalf("beside the bolt: %+v", beside)
	}
	if _, drawn := a.animationState[geometry.Point{X: 0, Y: -1}]; drawn {
		t.Fatal("lit a tile the map knows nothing of")
	}
}

// Whoever an effect plays on stays in sight, unless changing them is the effect.
func TestEffectsKeepTheActorInSight(t *testing.T) {
	u := &UI{game: roomGame{player: &atActor{}}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec"), settings: &foundation.Configuration{AnimationsEnabled: true, AnimateEffects: true}}
	for _, name := range []string{"haste", "slow", "levitate", "see_invisible", "blind", "hallucinate", "detect_food", "detect_magic", "detect_monsters", "detect_traps",
		"light", "darkness", "raise_level", "sleep", "laughter", "red_glow", "cancel", "hold", "heal", "extra_heal", "gain_strength", "gain_max_hp"} {
		anim := u.GetAnimEffect(name, previewCenter, nil, nil).(*FxAnimation)
		for f := 0; !anim.IsDone(); f++ {
			if icon, ok := anim.GetDrawables()[previewCenter]; ok && icon.Rune != '@' {
				t.Fatalf("%s frame %d covers the actor with %c", name, f, icon.Rune)
			}
			anim.NextFrame()
		}
	}
}

// A faster projectile shows on every cell of its path, it just stays on each for fewer ticks.
func TestFastProjectileVisitsEveryCell(t *testing.T) {
	path := make([]geometry.Point, 10)
	for i := range path {
		path[i] = geometry.Point{X: i}
	}
	fly := func(speed float64) (ticks int, seen map[geometry.Point]bool) {
		a := NewAnimator()
		a.SetSubTicks(6)
		p := NewProjectileAnimation(path, foundation.TextIcon{Rune: '/'}, nil, nil)
		p.SetSpeed(speed)
		a.AddAnimation(p)
		a.Flush()
		seen = map[geometry.Point]bool{}
		for a.Tick(); a.IsBusy(); a.Tick() {
			for pos := range a.animationState {
				seen[pos] = true
			}
			ticks++
		}
		return ticks, seen
	}
	normal, _ := fly(0)
	fast, seen := fly(1.5)
	if fast*3 != normal*2 || len(seen) != len(path) {
		t.Fatalf("ticks %d at 1.5 vs %d normal, saw %d of %d cells", fast, normal, len(seen), len(path))
	}
}
