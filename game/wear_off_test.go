package game

import (
	"rx1/foundation"
	"testing"
)

// Every timed effect runs out and says so, once.
func TestTimedEffectsWearOffWithAMessage(t *testing.T) {
	cfg := foundation.NewDefaultConfiguration()
	cfg.DataRootDir = "../data_rx1"
	g := NewGameState(stubUI{}, cfg)
	for _, effect := range timedEffects {
		g.Player.GetFlags().Increase(effect.flag, 2)
	}
	for turn := 1; turn <= 3; turn++ {
		g.logBuffer = nil
		g.decrementStatusEffects()
		wantMessages := 0
		if turn == 2 {
			wantMessages = len(timedEffects)
		}
		if len(g.logBuffer) != wantMessages {
			t.Fatalf("turn %d: %d messages, want %d: %v", turn, len(g.logBuffer), wantMessages, g.logBuffer)
		}
	}
	for _, effect := range timedEffects {
		if g.Player.HasFlag(effect.flag) {
			t.Fatalf("flag %v is still set", effect.flag)
		}
	}
}
