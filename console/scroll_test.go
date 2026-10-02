package console

import "testing"

func TestScrollAxis(t *testing.T) {
	for _, c := range []struct{ scroll, player, window, mapSize, want int }{
		{0, 40, 80, 80, 0},     // the map fits: never scroll
		{7, 3, 100, 80, 0},     // window larger than the map
		{0, 70, 80, 159, 0},    // inside the margin: stay
		{0, 76, 80, 159, 36},   // close to the right edge: centre on the player
		{36, 150, 80, 159, 79}, // clamped to the end of the map
		{36, 38, 80, 159, 0},   // close to the left edge, clamped to the start
	} {
		if got := scrollAxis(c.scroll, c.player, c.window, c.mapSize); got != c.want {
			t.Errorf("scrollAxis(%d, %d, %d, %d) = %d, want %d", c.scroll, c.player, c.window, c.mapSize, got, c.want)
		}
	}
}
