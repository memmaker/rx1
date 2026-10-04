package console

import (
	"rx1/foundation"
	"strings"
	"testing"
)

// a repeated plain message (an attack's) shows its count, as a highlighted one does
func TestRepeatedMessageShowsCount(t *testing.T) {
	u := &UI{currentTheme: NewThemeFromFile("../data_rx1/themes/fancy.rec")}
	m := foundation.Msg("You miss the bat")
	m.Repetitions = 1
	if s := u.ToColoredText(m, 1); !strings.HasSuffix(s, "(x2)") {
		t.Errorf("got %q", s)
	}
}
