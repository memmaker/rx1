package console

import "testing"

// A theme with %rec: tiles draws map, items and monsters with the tile font's runes (U+E000 + icon of the mapping rec).
func TestTilesThemeOverlaysIcons(t *testing.T) {
	theme := NewThemeFromFile("../data_rx1/themes/tiles.rec")
	if !theme.IsTiles() {
		t.Fatal("not a tiles theme")
	}
	if r := theme.GetIconForMap("TileFloor").Rune; r < 0xE000 || r > 0xE000+1183 {
		t.Errorf("floor rune %U, want a tile", r)
	}
	if theme.monsterIcons["bat"].Rune < 0xE000 || theme.monsterIcons["bat"].Fg.A == 0 {
		t.Errorf("bat: %+v, want a tile with a colour", theme.monsterIcons["bat"])
	}
	if plain := NewThemeFromFile("../data_rx1/themes/fancy.rec"); plain.IsTiles() || plain.GetIconForMap("TileFloor").Rune >= 0xE000 {
		t.Error("fancy must stay glyphs")
	}
}
