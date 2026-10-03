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

// animLight is a light an animation casts this frame.
type animLight struct {
	pos              geometry.Point
	color            color.RGBA
	radius, strength float64
}

// lightEmitter is an animation that lights up the map around it; the animator shines its lights over everything drawn.
type lightEmitter interface {
	GetLights() []animLight
}

// shine adds light of strength k to the icon: lights add up, as the glow of lava and fungus does.
func shine(icon foundation.TextIcon, light color.RGBA, k float64) foundation.TextIcon {
	add := func(c, l uint8, f float64) uint8 { return uint8(min(float64(c)+float64(l)*f, 255)) }
	icon.Fg = color.RGBA{add(icon.Fg.R, light.R, k), add(icon.Fg.G, light.G, k), add(icon.Fg.B, light.B, k), 255}
	icon.Bg = color.RGBA{add(icon.Bg.R, light.R, k/3), add(icon.Bg.G, light.G, k/3), add(icon.Bg.B, light.B, k/3), 255}
	return icon
}

// fxFalloff is the game's light falloff without its floor (the dimmest a tile in the player's light gets),
// so the light of an effect fades out to nothing at its edge.
func fxFalloff(d, r float64) float64 {
	return max(foundation.LightFalloff(d, r)-0.16, 0) / 0.84
}

// shineLight lights the tiles around l: shown gives what a tile shows now, false where nothing is known.
func shineLight(l animLight, shown func(geometry.Point) (foundation.TextIcon, bool), put func(geometry.Point, foundation.TextIcon)) {
	reach := int(l.radius) + 1
	for y := -reach; y <= reach; y++ {
		for x := -reach; x <= reach; x++ {
			p := l.pos.Add(geometry.Point{X: x, Y: y})
			d := geometry.Distance(l.pos, p)
			if d > l.radius+1 {
				continue
			}
			if icon, ok := shown(p); ok {
				put(p, shine(icon, l.color, fxFalloff(d, l.radius)*l.strength))
			}
		}
	}
}

// noAnimation stands in for a switched off animation: it plays no frame, but runs done and its follow-ups.
func noAnimation(done func()) foundation.Animation { return NewFxAnimation(0, nil, done) }

// fxLayer is one part of an effect; the layers of an effect play at the same time.
type fxLayer struct {
	frames int
	draw   func(frame int, put func(geometry.Point, foundation.TextIcon))
	// shine lights what is shown at the tiles it touches; lights shine after all layers have drawn
	shine func(frame int, shown func(geometry.Point) foundation.TextIcon, put func(geometry.Point, foundation.TextIcon))
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
	return fxLayer{frames: maxDist + len(palette), draw: func(f int, put func(geometry.Point, foundation.TextIcon)) {
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
	return fxLayer{frames: steps, draw: func(f int, put func(geometry.Point, foundation.TextIcon)) {
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
	return fxLayer{frames: (len(origins)-1)*spacing + len(colors), draw: func(f int, put func(geometry.Point, foundation.TextIcon)) {
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
	return fxLayer{frames: frames, draw: func(f int, put func(geometry.Point, foundation.TextIcon)) {
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

// lit shines a light from center over the cells: radius and strength per frame, fading with distance as the
// game's lights do, its colour added to what the tiles show, as glows add up (Brogue).
func (u *UI) lit(center geometry.Point, cells []geometry.Point, light color.RGBA, radius []float64, strength []float64) fxLayer {
	return fxLayer{frames: len(strength), shine: func(f int, shown func(geometry.Point) foundation.TextIcon, put func(geometry.Point, foundation.TextIcon)) {
		for _, p := range cells {
			if d := geometry.Distance(center, p); d <= radius[f]+1 {
				put(p, shine(shown(p), light, fxFalloff(d, radius[f])*strength[f]))
			}
		}
	}}
}

// glow is a soft light in the effect's colour that swells and fades over frames.
func (u *UI) glow(center geometry.Point, light color.RGBA, frames int) fxLayer {
	up := max(frames/3, 1)
	strength := append(ramp(0, 0.7, up), ramp(0.7, 0, frames-up)...)
	radius := make([]float64, frames)
	for i := range radius {
		radius[i] = 1 + 2*strength[i]
	}
	return u.lit(center, disc(center, 4), light, radius, strength)
}

// ramp is n steps from a to b.
func ramp(a, b float64, n int) []float64 {
	steps := make([]float64, n)
	for i := range steps {
		steps[i] = a + (b-a)*float64(i)/float64(max(n-1, 1))
	}
	return steps
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
		layers = []fxLayer{u.glow(center, color.RGBA{255, 230, 60, 255}, 14),
			u.orbit(center, []rune("»›"), []string{"White", "Yellow", "Yellow", "DarkYellow", "Brown"}, 3, false),
			u.pulse(center, hold(3, u.bgPalette("Black", "Yellow", "DarkYellow", "Brown", "VeryDarkGray"))),
		}
	case "heal":
		layers = []fxLayer{
			u.glow(center, color.RGBA{60, 255, 100, 255}, 16),
			u.rise(ring(center, 1), []rune("+"), []string{"White", "LightGreen", "LightGreen", "Green", "Green", "DarkGray"}, 0),
		}
	case "extra_heal":
		layers = []fxLayer{
			u.glow(center, color.RGBA{120, 255, 160, 255}, 22),
			u.rise(append(ring(center, 1), ring(center, 2)...), []rune("+✚"), []string{"White", "White", "LightGreen", "LightGreen", "Green", "Green", "DarkGray"}, 0),
			u.wave(center, disc(center, 5), false, []foundation.TextIcon{u.icon(keepRune, "White", ""), u.icon(keepRune, "LightGreen", ""), u.icon(keepRune, "Green", "")}),
		}
	case "gain_strength":
		layers = []fxLayer{
			u.glow(center, color.RGBA{255, 130, 30, 255}, 16),
			u.pulse(center, hold(2, u.bgPalette("Black", "Yellow", "Orange", "Red", "Orange", "Brown"))),
			u.wave(center, disc(center, 3), false, hold(2, []foundation.TextIcon{u.icon('!', "Yellow", ""), u.icon('!', "Orange", ""), u.icon('∙', "Red", "")})),
		}
	case "gain_max_hp":
		layers = []fxLayer{
			u.glow(center, color.RGBA{255, 60, 90, 255}, 16),
			u.rise([]geometry.Point{center, center.Add(geometry.Point{X: -1}), center.Add(geometry.Point{X: 1}), center}, []rune("♥"), []string{"White", "LightRed", "LightRed", "Red", "Red", "Brown"}, 0),
		}
	case "slow":
		layers = []fxLayer{u.glow(center, color.RGBA{60, 110, 255, 255}, 14),
			u.orbit(center, []rune("·∙"), []string{"VeryLightBlue", "LightBlue", "Blue", "DirtyBlue"}, 2, true),
			u.wave(center, disc(center, 2), true, hold(2, u.bgPalette("", "DirtyBlue", "ChasmEdge", "ChasmEdge"))),
		}
	case "levitate":
		var origins []geometry.Point
		for _, x := range []int{0, -2, 2, -1, 1, 0} {
			origins = append(origins, center.Add(geometry.Point{X: x, Y: 1}))
		}
		layers = []fxLayer{u.glow(center, color.RGBA{120, 220, 255, 255}, 14),
			u.rise(origins, []rune("↑˄∙↑"), []string{"White", "LightCyan", "LightCyan", "LightBlue", "DirtyBlue", "ChasmEdge"}, 0),
			u.pulse(center, hold(3, u.bgPalette("Black", "VeryLightBlue", "LightCyan", "LightBlue", "DirtyBlue"))),
		}
	case "see_invisible":
		layers = []fxLayer{u.glow(center, color.RGBA{80, 255, 255, 255}, 14),
			u.wave(center, disc(center, 5), false, []foundation.TextIcon{u.icon('◊', "White", ""), u.icon('◊', "LightCyan", ""), u.icon('∙', "Cyan", "")}),
			u.pulse(center, hold(2, []foundation.TextIcon{u.icon('ʘ', "White", ""), u.icon('ʘ', "LightCyan", ""), u.icon('ʘ', "White", ""), u.icon('ʘ', "LightCyan", ""), u.icon('ʘ', "Cyan", ""), u.icon('o', "Cyan", "")})),
		}
	case "blind":
		layers = []fxLayer{
			u.wave(center, disc(center, 4), true, hold(2, []foundation.TextIcon{u.icon('░', "DarkGray", ""), u.icon('▒', "VeryDarkGray", "Black"), u.icon('░', "VeryDarkGray", "Black"), u.icon(' ', "Black", "Black")})),
		}
	case "hallucinate":
		layers = []fxLayer{u.glow(center, color.RGBA{255, 80, 255, 255}, 14), u.glitch(disc(center, 3), 18, []rune("@&%$#?!*~§¤ΦΩ♣♠♥♦☺"))}
	case "polymorph":
		layers = []fxLayer{u.glow(center, color.RGBA{230, 60, 255, 255}, 14),
			u.glitch([]geometry.Point{center}, 14, []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")),
			u.wave(center, disc(center, 2), false, hold(2, []foundation.TextIcon{u.icon('∙', "White", ""), u.icon('∙', "LightMagenta", ""), u.icon('∙', "Magenta", "")})),
		}
	case "detect_food":
		layers = []fxLayer{u.glow(center, color.RGBA{255, 170, 80, 255}, 14), u.sonar(center, 9, "LightBrown", "Brown")}
	case "detect_magic":
		layers = []fxLayer{u.glow(center, color.RGBA{230, 60, 255, 255}, 14), u.sonar(center, 9, "LightMagenta", "Magenta")}
	case "detect_monsters":
		layers = []fxLayer{u.glow(center, color.RGBA{255, 50, 40, 255}, 14), u.sonar(center, 9, "LightRed", "Red")}
	case "detect_traps":
		layers = []fxLayer{u.glow(center, color.RGBA{255, 230, 60, 255}, 14), u.sonar(center, 9, "Yellow", "DarkYellow")}
	case "light":
		// a warm light swells from the user until it fills the room, flares and settles into the room's own light
		maxDist := 0.0
		for _, p := range area {
			maxDist = max(maxDist, geometry.Distance(center, p))
		}
		radius := append(ramp(0, maxDist, 10), ramp(maxDist, maxDist, 8)...)
		strength := append(ramp(0.75, 0.75, 10), ramp(0.75, 0, 8)...)
		layers = []fxLayer{u.lit(center, area, color.RGBA{255, 220, 140, 255}, radius, strength)}
	case "darkness":
		layers = []fxLayer{u.wave(center, area, false, []foundation.TextIcon{u.icon('▒', "Black", ""), u.icon('▓', "VeryDarkGray", "Black"), u.icon('▒', "VeryDarkGray", "Black"), u.icon('░', "VeryDarkGray", "Black"), u.icon(keepRune, "VeryDarkGray", "Black")})}
	case "raise_level":
		layers = []fxLayer{u.glow(center, color.RGBA{255, 210, 80, 255}, 14),
			u.rise(ring(center, 1), []rune("*+↑"), []string{"White", "Yellow", "Yellow", "DarkYellow", "Brown"}, 0),
			u.pulse(center, hold(3, u.bgPalette("Black", "White", "Yellow", "LightBrown", "DarkYellow", "Brown"))),
			u.wave(center, disc(center, 6), false, []foundation.TextIcon{u.icon('∙', "Yellow", ""), u.icon('∙', "DarkYellow", "")}),
		}
	case "sleep":
		layers = []fxLayer{u.glow(center, color.RGBA{80, 120, 255, 255}, 14), u.rise([]geometry.Point{center, center, center, center}, []rune("zZzZ"), []string{"VeryLightBlue", "LightBlue", "LightBlue", "DirtyBlue", "DirtyBlue", "DarkGray", "VeryDarkGray"}, 1)}
	case "laughter":
		var origins []geometry.Point
		for range 8 {
			origins = append(origins, center.Add(geometry.Point{X: rand.Intn(11) - 5, Y: rand.Intn(5) - 1}))
		}
		layers = []fxLayer{u.glow(center, color.RGBA{220, 60, 220, 255}, 14), u.rise(origins, []rune("haHa"), []string{"LightMagenta", "LightMagenta", "Magenta", "Magenta", "DarkGray"}, 0)}
	case "red_glow":
		// two heartbeats of red light, then it settles into a faint glow
		strength := append(append(append(ramp(0, 1, 4), ramp(1, 0.3, 4)...), append(ramp(0.3, 1, 3), ramp(1, 0.3, 4)...)...), ramp(0.3, 0, 5)...)
		radius := make([]float64, len(strength))
		for i := range radius {
			radius[i] = 1 + 2*strength[i]
		}
		layers = []fxLayer{u.lit(center, disc(center, 4), color.RGBA{255, 30, 20, 255}, radius, strength)}
	case "cancel":
		layers = []fxLayer{u.glow(center, color.RGBA{200, 200, 200, 255}, 14),
			u.wave(center, disc(center, 4), true, []foundation.TextIcon{u.icon('×', "White", ""), u.icon('×', "LightGray", ""), u.icon('∙', "DarkGray", "")}),
			u.pulse(center, append(hold(6, []foundation.TextIcon{u.icon(keepRune, "", "")}), hold(2, u.bgPalette("Black", "White", "LightGray", "Gray", "DarkGray"))...)),
		}
	case "hold":
		layers = []fxLayer{u.glow(center, color.RGBA{120, 255, 255, 255}, 14),
			u.wave(center, disc(center, 3), true, []foundation.TextIcon{u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('∙', "Cyan", "")}),
			u.wave(center, ring(center, 1), false, append(hold(4, []foundation.TextIcon{u.icon(keepRune, "", "")}), hold(2, []foundation.TextIcon{u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('#', "White", ""), u.icon('#', "LightCyan", ""), u.icon('#', "Cyan", "")})...)),
		}
	case "invisible":
		layers = []fxLayer{u.glow(center, color.RGBA{180, 180, 200, 255}, 14),
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
	coversCenter := effect == "polymorph" || effect == "invisible" // the change of who stands there is the effect
	drawn := map[geometry.Point]foundation.TextIcon{}
	shown := func(p geometry.Point) foundation.TextIcon {
		if icon, ok := drawn[p]; ok {
			return icon
		}
		return u.tileIcon(p)
	}
	return NewFxAnimation(frames, func(f int, put func(geometry.Point, foundation.TextIcon)) {
		clear(drawn)
		keep := func(p geometry.Point, icon foundation.TextIcon) { drawn[p] = icon }
		for _, l := range layers {
			if f < l.frames && l.draw != nil {
				l.draw(f, keep)
			}
		}
		for _, l := range layers {
			if f < l.frames && l.shine != nil {
				l.shine(f, shown, keep)
			}
		}
		if icon, ok := drawn[center]; ok && !coversCenter {
			icon.Rune = u.tileIcon(center).Rune // whoever stands there stays in sight, lit by the effect
			drawn[center] = icon
		}
		for p, icon := range drawn {
			put(p, icon)
		}
	}, done)
}
