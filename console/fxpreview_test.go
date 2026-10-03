package console

// Temporary: renders every effect to an animated HTML page for tuning. Run with FXPREVIEW=out.html.

import (
	"fmt"
	"image/color"
	"os"
	"rx1/foundation"
	"rx1/geometry"
	"strings"
	"testing"
)

type roomGame struct {
	foundation.GameForUI
	player foundation.ActorForUI
}

var previewCenter = geometry.Point{X: 12, Y: 7}

// the player stands in the middle of a lit room, looked up as the game does
func (roomGame) IsVisibleToPlayer(p geometry.Point) bool {
	return roomGame{}.MapAt(p) != foundation.TileEmpty
}
func (roomGame) GetHudFlags() map[foundation.ActorFlag]int { return nil }
func (roomGame) GetPlayerPosition() geometry.Point         { return previewCenter }
func (roomGame) PlayerWielding() (string, string) { return "", "" }
func (roomGame) GetPlayerLight() (foundation.LightInfo, bool) {
	return foundation.LightInfo{}, false
}
func (roomGame) GlowAt(geometry.Point) (color.RGBA, bool)          { return color.RGBA{}, false }
func (roomGame) ObjectAt(geometry.Point) foundation.ObjectCategory { return -1 }
func (g roomGame) ActorAt(p geometry.Point) foundation.ActorForUI {
	if p == previewCenter {
		return g.player
	}
	return nil
}
func (g roomGame) TopEntityAt(p geometry.Point, actor foundation.ActorForUI) foundation.EntityType {
	if actor != nil {
		return foundation.EntityTypeActor
	}
	return foundation.EntityTypeWorldTile
}
func (roomGame) IsExplored(geometry.Point) bool { return true }
func (roomGame) IsLit(geometry.Point) bool      { return true }
func (roomGame) MapAt(p geometry.Point) foundation.TileType {
	switch {
	case p.X < 2 || p.X > 22 || p.Y < 1 || p.Y > 13:
		return foundation.TileEmpty
	case p.X == 2 || p.X == 22 || p.Y == 1 || p.Y == 13:
		return foundation.TileWall
	}
	return foundation.TileFloor
}

func TestFxPreview(t *testing.T) {
	out := os.Getenv("FXPREVIEW")
	if out == "" {
		t.Skip()
	}
	u := &UI{game: roomGame{player: &atActor{}}, currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec"), settings: &foundation.Configuration{AnimationsEnabled: true, AnimateEffects: true, AnimateProjectiles: true}}
	center := geometry.Point{X: 12, Y: 7}
	var room []geometry.Point
	for y := 2; y <= 12; y++ {
		for x := 3; x <= 21; x++ {
			room = append(room, geometry.Point{X: x, Y: y})
		}
	}
	hex := func(c interface{ RGBA() (r, g, b, a uint32) }) string {
		r, g, b, _ := c.RGBA()
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	cell := func(i foundation.TextIcon) string {
		r := i.Rune
		if r == 0 {
			r = ' '
		}
		return fmt.Sprintf("[%q,%q,%q]", string(r), hex(i.Fg), hex(i.Bg))
	}
	var js strings.Builder
	js.WriteString("const FX={")
	names := strings.Fields("haste slow levitate see_invisible blind hallucinate polymorph detect_food detect_magic detect_monsters detect_traps light darkness raise_level sleep laughter red_glow cancel hold invisible heal extra_heal gain_strength gain_max_hp")
	for _, name := range names {
		var area []geometry.Point
		if name == "light" || name == "darkness" {
			area = room
		}
		anim := u.GetAnimEffect(name, center, area, nil).(*FxAnimation)
		fmt.Fprintf(&js, "%q:[", name)
		for !anim.IsDone() {
			js.WriteString("{")
			for p, i := range anim.GetDrawables() {
				fmt.Fprintf(&js, "\"%d,%d\":%s,", p.X, p.Y, cell(i))
			}
			js.WriteString("},")
			anim.NextFrame()
		}
		js.WriteString("],")
	}
	// the older animations, played by the animator so their lights shine
	var line []geometry.Point
	for x := 4; x <= 20; x++ {
		line = append(line, geometry.Point{X: x, Y: 7})
	}
	scenes := map[string]func() foundation.Animation{
		"fire_ray": func() foundation.Animation {
			a, _ := u.GetAnimProjectileWithTrail(' ', []string{"White", "Yellow", "LightRed", "Red"}, line, nil)
			return a
		},
		"cold_ray": func() foundation.Animation {
			a, _ := u.GetAnimProjectileWithTrail('☼', []string{"White", "White", "LightCyan", "LightBlue", "Blue"}, line, nil)
			return a
		},
		"lightning": func() foundation.Animation {
			a, _ := u.GetAnimProjectileWithTrail(' ', []string{"White", "Yellow", "Yellow", "Yellow"}, line, nil)
			return a
		},
		"magic_missile": func() foundation.Animation {
			a, _ := u.GetAnimProjectile('°', "LightGreen", line[0], line[len(line)-1], nil)
			return a
		},
		"explosion": func() foundation.Animation { return u.GetAnimExplosion(disc(center, 2), nil) },
	}
	u.animator = NewAnimator()
	u.animator.lookup = u.mapLookup
	for _, name := range []string{"fire_ray", "cold_ray", "lightning", "magic_missile", "explosion"} {
		u.animator.AddAnimation(scenes[name]().(TextAnimation))
		u.animator.Flush()
		fmt.Fprintf(&js, "%q:[", name)
		for u.animator.Tick(); u.animator.IsBusy() || len(u.animator.animationState) > 0; u.animator.Tick() {
			js.WriteString("{")
			for p, i := range u.animator.animationState {
				fmt.Fprintf(&js, "\"%d,%d\":%s,", p.X, p.Y, cell(i))
			}
			js.WriteString("},")
		}
		js.WriteString("],")
	}
	js.WriteString("};const BASE={")
	for y := 0; y < 15; y++ {
		for x := 0; x < 25; x++ {
			p := geometry.Point{X: x, Y: y}
			i := u.tileIcon(p)
			fmt.Fprintf(&js, "\"%d,%d\":%s,", x, y, cell(i))
		}
	}
	js.WriteString("};")
	html := `<!doctype html><meta charset=utf-8><title>FX preview</title><style>
body{background:#111;color:#ccc;font:14px monospace;margin:16px}
#root{display:flex;flex-wrap:wrap}.fx{margin:6px}.g{line-height:1;font-size:11px;white-space:pre;background:#000}
.g span{display:inline-block;width:.62em;text-align:center}</style>
<label>delay ms <input id=d type=number value=45 style=width:4em></label><div id=root></div><script>` + js.String() + `
const root=document.getElementById('root');const views={};
for(const n in FX){const d=document.createElement('div');d.className='fx';d.innerHTML='<div>'+n+' ('+FX[n].length+')</div><div class=g></div>';root.appendChild(d);views[n]={g:d.lastChild,f:0,pause:0}}
function draw(n){const v=views[n],fr=FX[n][v.f]||{};let h='';for(let y=0;y<15;y++){for(let x=0;x<25;x++){const c=fr[x+','+y]||BASE[x+','+y];h+='<span style="color:'+c[1]+';background:'+c[2]+'">'+c[0].replace('<','&lt;')+'</span>'}h+='\n'}v.g.innerHTML=h}
function tick(){for(const n in views){const v=views[n];if(v.pause>0){v.pause--;}else{v.f++;if(v.f>=FX[n].length){v.f=FX[n].length;v.pause=15}}if(v.pause===1)v.f=0;draw(n)}setTimeout(tick,+document.getElementById('d').value)}tick();
</script>`
	if err := os.WriteFile(out, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
}
