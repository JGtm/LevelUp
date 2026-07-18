package prestigetuning

import (
	"reflect"
	"testing"
)

func TestGrammarView_Lookups(t *testing.T) {
	g := NewGrammarView(map[string][]string{
		"accuracy": {"last_n_matches:10", "rolling_days:7"},
		"kda":      {"session"},
	})
	if !g.HasMetric("accuracy") || g.HasMetric("unknown") {
		t.Errorf("HasMetric incorrect")
	}
	if !g.HasWindow("accuracy", "rolling_days:7") || g.HasWindow("accuracy", "rolling_days:30") {
		t.Errorf("HasWindow incorrect")
	}
	if !g.HasWindow("kda", "session") {
		t.Errorf("fenêtre session non reconnue")
	}
	if got := g.Metrics(); !reflect.DeepEqual(got, []string{"accuracy", "kda"}) {
		t.Errorf("Metrics = %v, want [accuracy kda] triées", got)
	}
}

func TestGrammarView_DefensiveCopy(t *testing.T) {
	src := map[string][]string{"accuracy": {"last_n_matches:10"}}
	g := NewGrammarView(src)
	delete(src, "accuracy") // mutation externe post-construction
	if !g.HasMetric("accuracy") {
		t.Errorf("la vue doit être insensible aux mutations de la map source")
	}
}
