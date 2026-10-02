package game

import (
	"fmt"
	"os"
	"path"
	"rx1/foundation"
	"strings"

	"codeberg.org/tslocum/cview"
)

// LoadDocuments turns the lore texts into readable floor items:
// the history is one document, stories.txt holds one story per "%%"-separated chunk (the first chunk is its heading).
func LoadDocuments(loreDir string) []ItemDef {
	var defs []ItemDef
	if history, err := os.ReadFile(path.Join(loreDir, "history.of.rogue.txt")); err == nil {
		defs = append(defs, newDocumentDef("torn page", "document_history", string(history)))
	}
	if stories, err := os.ReadFile(path.Join(loreDir, "stories.txt")); err == nil {
		for i, story := range SplitStories(string(stories)) {
			defs = append(defs, newDocumentDef("old letter", fmt.Sprintf("document_story_%d", i+1), story))
		}
	}
	return defs
}

func SplitStories(text string) []string {
	chunks := strings.Split(text, "\n%%\n")
	var stories []string
	for _, chunk := range chunks[1:] {
		if chunk = strings.TrimSpace(chunk); chunk != "" {
			stories = append(stories, chunk)
		}
	}
	return stories
}

func newDocumentDef(kind, internalName, text string) ItemDef {
	text = strings.TrimSpace(text)
	title, _, _ := strings.Cut(text, "\n")
	if runes := []rune(title); len(runes) > 32 {
		title = strings.TrimSpace(string(runes[:32])) + "…"
	}
	return ItemDef{
		Name:         fmt.Sprintf("%s: %s", kind, title),
		InternalName: internalName,
		Category:     foundation.ItemCategoryDocuments,
		Text:         text,
	}
}

// documentLines word-wraps a document for the text modal (Esc closes it, arrows/PgUp/PgDn scroll).
func documentLines(title, text string) []string {
	lines := []string{cview.Escape(title), ""}
	for _, line := range cview.WordWrap(text, 70) {
		lines = append(lines, cview.Escape(strings.TrimRight(line, " ")))
	}
	return lines
}
