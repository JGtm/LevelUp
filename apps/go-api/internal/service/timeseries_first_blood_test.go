package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/legacymatch"
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
	m1Start := time.Date(2026, 4, 21, 20, 0, 0, 0, time.UTC)
	matches := []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: m1Start, MapNameFR: "Aquarius FR", MapName: "Aquarius", PairNameFR: "Assassin"},
		{MatchID: "m2", StartTime: m1Start.Add(time.Hour), MapName: "Bazaar"},
		{MatchID: "m3", StartTime: m1Start.Add(2 * time.Hour)},
	}
	got := buildSoloFirstBlood("Madina", rows, matches)
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
	// DEC-4 (retours utilisateur 2026-08-29) : carte/mode/date accompagnent
	// désormais chaque point (tooltip web sans uuid).
	if pts[0].MapUI != "Aquarius FR" {
		t.Errorf("m1 carte want le FR prioritaire, got %q", pts[0].MapUI)
	}
	if pts[0].ModeUI != "Assassin" {
		t.Errorf("m1 mode want %q, got %q", "Assassin", pts[0].ModeUI)
	}
	if !pts[0].StartTime.Equal(m1Start) {
		t.Errorf("m1 start_time want %v, got %v", m1Start, pts[0].StartTime)
	}
	if pts[1].MapUI != "Bazaar" {
		t.Errorf("m2 carte want repli EN (pas de FR fournie), got %q", pts[1].MapUI)
	}
	if pts[2].ModeUI != "" {
		t.Errorf("m3 sans pair/variant : mode want vide, got %q", pts[2].ModeUI)
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
	if got := buildSoloFirstBlood("Madina", rows, nil); got != nil {
		t.Errorf("want nil sans aucun événement, got %#v", got)
	}
	if got := buildSoloFirstBlood("", rows, nil); got != nil {
		t.Errorf("want nil sans gamertag, got %#v", got)
	}
	if got := buildSoloFirstBlood("Madina", nil, nil); got != nil {
		t.Errorf("want nil sans row, got %#v", got)
	}
}
