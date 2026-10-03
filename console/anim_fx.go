package console

import (
	"image/color"
	"math"
	"math/rand"
	"rx1/foundation"
	"rx1/geometry"
)

// FxAnimation draws each frame with a function, so every tile can look different: waves, orbits, rising sparks.
// With no frames it is done at once and only calls done.
type FxAnimation struct {
	*BaseAnimation
	frame, frames int
	draw          func(frame int, put func(geometry.Point, foundation.TextIcon))
	drawables     map[geometry.Point]foundation.TextIcon
}

func NewFxAnimation(frames int, draw func(frame int, put func(geometry.Point, foundation.TextIcon)), done func()) *FxAnimation {
	a := &FxAnimation{BaseAnimation: &BaseAnimation{done: done}, frames: frames, draw: draw, drawables: map[geometry.Point]foundation.TextIcon{}}
	a.paint()
	return a
}

func (a *FxAnimation) paint() {
	clear(a.drawables)
	if a.frame < a.frames {
		a.draw(a.frame, func(p geometry.Point, icon foundation.TextIcon) { a.drawables[p] = icon })
	}
}

func (a *FxAnimation) GetPriority() int { return 1 }

func (a *FxAnimation) GetDrawables() map[geometry.Point]foundation.TextIcon { return a.drawables }

func (a *FxAnimation) NextFrame() {
	if a.IsDone() {
		return
	}
	a.frame++
	a.paint()
	if a.frame >= a.frames {
		a.onFinishedOrCancelled()
	}
}

func (a *FxAnimation) IsDone() bool { return a.frame >= a.frames || a.finishedOrCancelled }

// noAnimation stands in for a switched off animation: it plays no frame, but runs done and its follow-ups.
func noAnimation(done func()) foundation.Animation { return NewFxAnimation(0, nil, done) }

// fxLayer is one part of an effect; the layers of an effect play at the same time.
type fxLayer struct {
	frames int
	draw   func(frame int, put func(geometry.Point, foundation.TextIcon))
}

// keepRune in a palette entry keeps the rune of the map tile underneath.
const keepRune = rune(0)

// clockwise neighbours, starting north
var orbitRing = []geometry.Point{{X: 0, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}, {X: -1, Y: 1}, {X: -1, Y: 0}, {X: -1, Y: -1}}

func (u *UI) tileIcon(p geometry.Point) foundation.TextIcon {
	if icon, ok := u.mapLookup(p); ok {
		return icon
	}
	return foundation.TextIcon{Rune: ' ', Fg: u.color("Black"), Bg: u.currentTheme.GetUIColor(UIColorMapDefaultBackground)}
}

func (u *UI) color(name string) color.RGBA { return u.currentTheme.GetColorByName(name) }

// icon names its colours; an empty name keeps the colour of the map tile underneath.
func (u *UI) icon(r rune, fg, bg string) foundation.TextIcon {
	icon := foundation.TextIcon{Rune: r}
	if fg != "" {
		icon.Fg = u.color(fg)
	}
	if bg != "" {
		icon.Bg = u.color(bg)
	}
	return icon
}

// fxDist is the distance on screen: a cell is about twice as high as it is wide, so ripples come out round.
func fxDist(a, b geometry.Point) int {
	return int(math.Round(math.Hypot(float64(a.X-b.X)/2, float64(a.Y-b.Y))))
}

// disc is the cells within radius of center on screen.
func disc(center geometry.Point, radius int) []geometry.Point {
	var cells []geometry.Point
	for _, p := range square(center, radius*2) {
		if fxDist(center, p) <= radius {
			cells = append(cells, p)
		}
	}
	return cells
}

// onTile fills in what a palette entry leaves open: keepRune and colours without alpha come from the map tile.
func (u *UI) onTile(p geometry.Point, icon foundation.TextIcon) foundation.TextIcon {
	if icon.Rune != keepRune && icon.Fg.A != 0 && icon.Bg.A != 0 {
		return icon
	}
	tile := u.tileIcon(p)
	if icon.Rune == keepRune {
		icon.Rune = tile.Rune
	}
	if icon.Fg.A == 0 {
		icon.Fg = tile.Fg
	}
	if icon.Bg.A == 0 {
		icon.Bg = tile.Bg
	}
	return icon
}

// wave lights the cells one after the other by their distance to center, each running through the palette.
// Inward starts at the rim.
func (u *UI) wave(center geometry.Point, cells []geometry.Point, inward bool, palette []foundation.TextIcon) fxLayer {
	maxDist := 0
	for _, p := range cells {
		maxDist = max(maxDist, fxDist(center, p))
	}
	return fxLayer{maxDist + len(palette), func(f int, put func(geometry.Point, foundation.TextIcon)) {
		for _, p := range cells {
			delay := fxDist(center, p)
			if inward {
				delay = maxDist - delay
			}
			if k := f - delay; k >= 0 && k < len(palette) {
				put(p, u.onTile(p, palette[k]))
			}
		}
	}}
}

// orbit sends a glowing comet around center, its tail fading through colors.
func (u *UI) orbit(center geometry.Point, runes []rune, colors []string, laps int, counterClockwise bool) fxLayer {
	steps := laps * len(orbitRing)
	return fxLayer{steps, func(f int, put func(geometry.Point, foundation.TextIcon)) {
		for t, c := range colors {
			i := f - t
			if i < 0 {
				break
			}
			if counterClockwise {
				i = -i
			}
			i = ((i % 8) + 8) % 8
			p := center.Add(orbitRing[i])
			put(p, foundation.TextIcon{Rune: runes[(f+t)%len(runes)], Fg: u.color(c), Bg: u.tileIcon(p).Bg})
		}
	}}
}

// rise lets sparks float up from the origins one after the other, fading through colors.
func (u *UI) rise(origins []geometry.Point, runes []rune, colors []string, drift int) fxLayer {
	const spacing = 2
	return fxLayer{(len(origins)-1)*spacing + len(colors), func(f int, put func(geometry.Point, foundation.TextIcon)) {
		for i, o := range origins {
			age := f - i*spacing
			if age < 0 || age >= len(colors) {
				continue
			}
			p := o.Add(geometry.Point{X: drift * age / 2, Y: -age})
			put(p, foundation.TextIcon{Rune: runes[i%len(runes)], Fg: u.color(colors[age]), Bg: u.tileIcon(p).Bg})
		}
	}}
}

// glitch scrambles the cells with random runes in random colors.
func (u *UI) glitch(cells []geometry.Point, frames int, runes []rune) fxLayer {
	return fxLayer{frames, func(f int, put func(geometry.Point, foundation.TextIcon)) {
		for _, p := range cells {
			if rand.Intn(3) > 0 {
				put(p, foundation.TextIcon{Rune: runes[rand.Intn(len(runes))], Fg: u.currentTheme.GetRandomColor(), Bg: u.tileIcon(p).Bg})
			}
		}
	}}
}

// pulse flashes the center through the palette, keeping the rune of what stands there.
func (u *UI) pulse(center geometry.Point, palette []foundation.TextIcon) fxLayer {
	return u.wave(center, []geometry.Point{center}, false, palette)
}

func square(center geometry.Point, radius int) []geometry.Point {
	var cells []geometry.Point
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			cells = append(cells, center.Add(geometry.Point{X: x, Y: y}))
		}
	}
	return cells
}

func ring(center geometry.Point, radius int) []geometry.Point {
	var cells []geometry.Point
	for _, p := range square(center, radius) {
		if geometry.DistanceChebyshev(center, p) == radius {
			cells = append(cells, p)
		}
	}
	return cells
}

// bgPalette is the tile's own rune on a run of background colors, the rune drawn in fg.
func (u *UI) bgPalette(fg string, bgs ...string) []foundation.TextIcon {
	palette := make([]foundation.TextIcon, len(bgs))
	for i, bg := range bgs {
		palette[i] = u.icon(keepRune, fg, bg)
	}
	return palette
}

func (u *UI) runePalette(r rune, bg string, fgs ...string) []foundation.TextIcon {
	palette := make([]foundation.TextIcon, len(fgs))
	for i, fg := range fgs {
		palette[i] = u.icon(r, fg, bg)
	}
	return palette
}

// hold repeats every entry of the palette n times, for slower changes.
func hold(n int, palette []foundation.TextIcon) []foundation.TextIcon {
	var held []foundation.TextIcon
	for _, icon := range palette {
		for range n {
			held = append(held, icon)
		}
	}
	return held
}

// sonar sends a bright ring out over a disc, leaving tinted runes behind it.
func (u *UI) sonar(center geometry.Point, radius int, lead string, trail string) fxLayer {
	return u.wave(center, disc(center, radius), false, []foundation.TextIcon{
		u.icon(keepRune, "Black", lead), u.icon(keepRune, lead, trail), u.icon(keepRune, lead, ""), u.icon(keepRune, trail, ""),
	})
}

// GetAnimEffect plays the named effect at center; area is the room or region it concerns, if any.
func (u *UI) GetAnimEffect(effect string, center geometry.Point, area []geometry.Point, done func()) foundation.Animation {
	if !u.settings.AnimationsEnabled || !u.settings.AnimateEffects {
		return noAnimation(done)
	}
	if len(area) == 0 {
		area = disc(center, 4)
	}
	var layers []fxLayer
	switch effect {
	case "haste":
		layers = []fxLayer{
			u.orbit(center, []rune("»›"), []string{"White", "Yellow", "Yellow", "DarkYellow", "Brown"}, 3, false),
			u.pulse(center, hold(3, u.bgPalette("Black", "Yellow", "DarkYellow", "Brown", "VeryDarkGray"))),
		}
	case "slow":
		layers = []fxLayer{
			u.orbit(center, []rune("·∙"), []string{"VeryLightBlue", "LightBlue", "Blue", "DirtyBlue"}, 2, true),
			u.wave(center, disc(center, 2), true, hold(2, u.bgPalette("", "DirtyBlue", "ChasmEdge", "ChasmEdge"))),
		}
	case "levitate":
		var origins []geometry.Point
		for _, x := range []int{0, -2, 2, -1, 1, 0} {
			origins = append(origins, center.Add(geometry.Point{X: x, Y: 1}))
		}
		layers = []fxLayer{
			u.rise(origins, []rune("↑˄∙↑"), []string{"White", "LightCyan", "LightCyan", "LightBlue", "DirtyBlue", "ChasmEdge"}, 0),
			u.pulse(center, hold(3, u.bgPalette("Black", "VeryLightBlue", "LightCyan", "LightBlue", "DirtyBlue"))),
		}
	case "see_invisible":
		layers = []fxLayer{
			u.wave(center, disc(center, 5), false, []foundation.TextIcon{u.icon('◊', "White", ""), u.icon('◊', "LightCyan", ""), u.icon('∙', "Cyan", "")}),
			u.pulse(center, hold(2, []foundation.TextIcon{u.icon('ʘ', "White", ""), u.icon('ʘ', "LightCyan", ""), u.icon('ʘ', "White", ""), u.icon('ʘ', "LightCyan", ""), u.icon('ʘ', "Cyan", ""), u.icon('o', "Cyan", "")})),
		}
	case "blind":
		layers = []fxLayer{
			u.wave(center, disc(center, 4), true, hold(2, []foundation.TextIcon{u.icon('░', "DarkGray", ""), u.icon('▒', "VeryDarkGray", "Black"), u.icon('░', "VeryDarkGray", "Black"), u.icon(' ', "Black", "Black")})),
		}
	case "hallucinate":
		layers = []fxLayer{u.glitch(disc(center, 3), 18, []rune("@&%$#?!*~§¤ΦΩ♣♠♥♦☺"))}
	case "polymorph":
		layers = []fxLayer{
			u.glitch([]geometry.Point{center}, 14, []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")),
			u.wave(center, disc(center, 2), false, hold(2, []foundation.TextIcon{u.icon('∙', "White", ""), u.icon('∙', "LightMagenta", ""), u.icon('∙', "Magenta", "")})),
		}
	case "detect_food":
		layers = []fxLayer{u.sonar(center, 9, "LightBrown", "Brown")}
	case "detect_magic":
		layers = []fxLayer{u.sonar(center, 9, "LightMagenta", "Magenta")}
	case "detect_monsters":
		layers = []fxLayer{u.sonar(center, 9, "LightRed", "Red")}
	case "detect_traps":
		layers = []fxLayer{u.sonar(center, 9, "Yellow", "DarkYellow")}
	case "light":
		layers = []fxLayer{u.wave(center, area, false, []foundation.TextIcon{u.icon(keepRune, "Black", "White"), u.icon(keepRune, "Black", "Yellow"), u.icon(keepRune, "Brown", "LightBrown"), u.icon(keepRune, "Yellow", "DarkYellow"), u.icon(keepRune, "Yellow", "Brown"), u.icon(keepRune, "Yellow", "")})}
	case "darkness":
		layers = []fxLayer{u.wave(center, area, false, []foundation.TextIcon{u.icon('▒', "Black", ""), u.icon('▓', "VeryDarkGray", "Black"), u.icon('▒', "VeryDarkGray", "Black"), u.icon('░', "VeryDarkGray", "Black"), u.icon(keepRune, "VeryDarkGray", "Black")})}
	case "raise_level":
		layers = []fxLayer{
			u.rise(ring(center, 1), []rune("*+↑"), []string{"White", "Yellow", "Yellow", "DarkYellow", "Brown"}, 0),
			u.pulse(center, hold(3, u.bgPalette("Black", "White", "Yellow", "LightBrown", "DarkYellow", "Brown"))),
			u.wave(center, disc(center, 6), false, []foundation.TextIcon{u.icon('∙', "Yellow", ""), u.icon('∙', "DarkYellow", "")}),
		}
	case "sleep":
		layers = []fxLayer{u.rise([]geometry.Point{center, center, center, center}, []rune("zZzZ"), []string{"VeryLightBlue", "LightBlue", "LightBlue", "DirtyBlue", "DirtyBlue", "DarkGray", "VeryDarkGray"}, 1)}
	case "laughter":
		var origins []geometry.Point
		for range 8 {
			origins = append(origins, center.Add(geometry.Point{X: rand.Intn(11) - 5, Y: rand.Intn(5) - 1}))
		}
		layers = []fxLayer{u.rise(origins, []rune("haHa"), []string{"LightMagenta", "LightMagenta", "Magenta", "Magenta", "DarkGray"}, 0)}
	case "red_glow":
		layers = []fxLayer{
			u.pulse(center, hold(2, u.bgPalette("Black", "Red", "LightRed", "Red", "LightRed", "Red", "Brown", "VeryDarkGray"))),
			u.wave(center, disc(center, 2), false, hold(2, []foundation.TextIcon{u.icon('∙', "LightRed", ""), u.icon('∙', "Red", ""), u.icon('∙', "Brown", "")})),
		}
	case "cancel":
		layers = []fxLayer{
			u.wave(center, disc(center, 4), true, []foundation.TextIcon{u.icon('×', "White", ""), u.icon('×', "LightGray", ""), u.icon('∙', "DarkGray", "")}),
			u.pulse(center, append(hold(6, []foundation.TextIcon{u.icon(keepRune, "", "")}), hold(2, u.bgPalette("Black", "White", "LightGray", "Gray", "DarkGray"))...)),
		}
	case "hold":
		layers = []fxLayer{
			u.wave(center, disc(center, 3), true, []foundation.TextIcon{u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('∙', "Cyan", "")}),
			u.wave(center, ring(center, 1), false, append(hold(4, []foundation.TextIcon{u.icon(keepRune, "", "")}), hold(2, []foundation.TextIcon{u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('#', "Cyan", "")})...)),
		}
	case "invisible":
		layers = []fxLayer{
			u.wave(center, disc(center, 3), false, []foundation.TextIcon{u.icon('∙', "White", ""), u.icon('∙', "LightGray", ""), u.icon('.', "DarkGray", "")}),
			u.pulse(center, hold(2, []foundation.TextIcon{u.icon(keepRune, "White", ""), u.icon(keepRune, "LightGray", ""), u.icon(keepRune, "Gray", ""), u.icon(keepRune, "DarkGray", ""), u.icon(keepRune, "VeryDarkGray", ""), u.icon('∙', "VeryDarkGray", "")})),
		}
	default:
		return noAnimation(done)
	}
	frames := 0
	for _, l := range layers {
		frames = max(frames, l.frames)
	}
	return NewFxAnimation(frames, func(f int, put func(geometry.Point, foundation.TextIcon)) {
		for _, l := range layers {
			if f < l.frames {
				l.draw(f, put)
			}
		}
	}, done)
}
