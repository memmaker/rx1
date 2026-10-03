package console

import "testing"

// A theme with %rec: tiles draws map, items and monsters with the tile font's runes (U+E000 + icon of the mapping rec).
func TestTilesThemeOverlaysIcons(t *testing.T) {
	theme := NewThemeFromFile("../data_rx1/themes/tiles.rec")
	if !theme.IsTiles() {
		t.Fatal("not a tiles theme")
	}
	if r := theme.GetIconForMap("TileFloor").Rune; r < 0xE000 || r > 0xE000+1313 {
		t.Errorf("floor rune %U, want a tile", r)
	}
	if theme.monsterIcons["bat"].Rune < 0xE000 || theme.monsterIcons["bat"].Fg.A == 0 {
		t.Errorf("bat: %+v, want a tile with a colour", theme.monsterIcons["bat"])
	}
	if theme.monsterAnim["bat"] != theme.monsterIcons["bat"].Rune+monsterCols {
		t.Errorf("bat animation frame %U, want the tile one row down", theme.monsterAnim["bat"])
	}
	// the player sprites: every weapon and shield pair in two frames, the nearest one for what has no sprite
	if s := theme.sprites[[2]string{"dagger", "buckler"}]; s[0] < 0xE000+1314 || s[1] != s[0]+1 {
		t.Errorf("dagger and buckler sprite %U %U", s[0], s[1])
	}
	if theme.PlayerSprite("", "") == 0 || theme.PlayerSprite("unknown", "large_shield") != theme.PlayerSprite("", "large_shield") {
		t.Error("sprite fallback")
	}
	if plain := NewThemeFromFile("../data_rx1/themes/fancy.rec"); plain.IsTiles() || plain.GetIconForMap("TileFloor").Rune >= 0xE000 {
		t.Error("fancy must stay glyphs")
	}
}
