package console

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every colour a definition names exists in every theme; a missing one is drawn black on black.
func TestThemesHaveDefinitionColors(t *testing.T) {
	re := regexp.MustCompile(`(?m)^color: *(\S+)`)
	defs, _ := filepath.Glob("../data_rx1/definitions/*.rec")
	themes, _ := filepath.Glob("../data_rx1/themes/*.rec")
	if len(defs) == 0 || len(themes) == 0 {
		t.Fatal("no data files")
	}
	for _, def := range defs {
		data, _ := os.ReadFile(def)
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			for _, file := range themes {
				if _, ok := NewThemeFromFile(file).colorDefs[strings.ToLower(m[1])]; !ok {
					t.Errorf("%s: %s is not in %s", filepath.Base(def), m[1], filepath.Base(file))
				}
			}
		}
	}
}
