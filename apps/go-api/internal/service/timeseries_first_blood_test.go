package service

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

func msPtr(v int64) *int64 { return &v }

// TestBuildSoloFirstBlood_ProjectsMatchPoints : la projection produit UNE série
// (solo), une entrée par row dans l'ordre reçu, et convertit les millisecondes en
// secondes arrondies au dixième. Un événement absent reste nil (le match compte
// quand même dans le dénominateur affiché côté front).
func TestBuildSoloFirstBlood_ProjectsMatchPoints(t *testing.T) {
	rows := []narrative.FirstEventsRow{
		{MatchID: "m1", FirstKillMS: msPtr(12345), FirstDeathMS: msPtr(60000)},
		{MatchID: "m2", FirstKillMS: nil, FirstDeathMS: msPtr(4000)},
		{MatchID: "m3", FirstKillMS: nil, FirstDeathMS: nil},
	}
	got := buildSoloFirstBlood("Madina", rows)
	if len(got) != 1 || got[0].Player != "Madina" {
		t.Fatalf("want 1 série pour Madina, got %#v", got)
	}
	pts := got[0].Matches
	if len(pts) != 3 {
		t.Fatalf("want 3 points (dont un sans événement), got %d", len(pts))
	}
	if pts[0].MatchID != "m1" || pts[0].FirstKillSec == nil || *pts[0].FirstKillSec != 12.3 {
		t.Errorf("m1 premier frag want 12.3s, got %#v", pts[0])
	}
	if pts[0].FirstDeathSec == nil || *pts[0].FirstDeathSec != 60 {
		t.Errorf("m1 première mort want 60s, got %v", pts[0].FirstDeathSec)
	}
	if pts[1].FirstKillSec != nil {
		t.Errorf("m2 sans premier frag doit rester nil, got %v", pts[1].FirstKillSec)
	}
	if pts[2].FirstKillSec != nil || pts[2].FirstDeathSec != nil {
		t.Errorf("m3 sans événement doit rester nil/nil, got %#v", pts[2])
	}
}

// TestBuildSoloFirstBlood_NilWhenNoEvent : un scope où aucun match ne porte de
// premier frag ni de première mort n'expose PAS de bloc (le front rend son état
// vide plutôt qu'une grille de lanes muettes).
func TestBuildSoloFirstBlood_NilWhenNoEvent(t *testing.T) {
	rows := []narrative.FirstEventsRow{{MatchID: "m1"}, {MatchID: "m2"}}
	if got := buildSoloFirstBlood("Madina", rows); got != nil {
		t.Errorf("want nil sans aucun événement, got %#v", got)
	}
	if got := buildSoloFirstBlood("", rows); got != nil {
		t.Errorf("want nil sans gamertag, got %#v", got)
	}
	if got := buildSoloFirstBlood("Madina", nil); got != nil {
		t.Errorf("want nil sans row, got %#v", got)
	}
}
