package game

import "testing"

func TestLoadDocuments(t *testing.T) {
	docs := LoadDocuments("../data_rx1/lore")
	if len(docs) < 10 {
		t.Fatalf("expected the history plus many stories, got %d documents", len(docs))
	}
	seen := map[string]bool{}
	for _, d := range docs {
		if d.Text == "" || d.Name == "" || seen[d.InternalName] {
			t.Fatalf("bad document %+v", d)
		}
		seen[d.InternalName] = true
	}
}
