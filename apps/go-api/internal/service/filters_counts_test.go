// filters_counts_test.go — tests pour la sémantique OR des counts par option
// (LabelValue.Count) + counts presets + SessionOption.MatchCountFiltered.
package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ─── Helpers de fabrication de FilterMatchRow ───────────────────────────────

func tsPtr(t time.Time) *time.Time { return &t }

// makeRow construit une ligne minimale.
func makeFilterRow(matchID, mode, mapName, playlist string, isRanked, isFirefight bool) domain.FilterMatchRow {
	return domain.FilterMatchRow{
		MatchID:      matchID,
		PairName:     strPtr(mode),
		PairNameFR:   strPtr(mode),
		MapName:      strPtr(mapName),
		MapNameFR:    strPtr(mapName),
		PlaylistName: strPtr(playlist),
		IsRanked:     isRanked,
		IsFirefight:  isFirefight,
	}
}

// ─── uniqueLabelValuesWithORCounts ─────────────────────────────────────────

func TestUniqueLabelValuesWithORCounts_NoSelection(t *testing.T) {
	rows := []domain.FilterMatchRow{
		makeFilterRow("m1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("m2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("m3", "CTF", "Streets", "Arena", false, false),
	}
	identity := func(rs []domain.FilterMatchRow) []domain.FilterMatchRow { return rs }

	got := uniqueLabelValuesWithORCounts(rows, modeUI, nil, identity)
	if len(got) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(got))
	}
	// Ordre alphabétique → CTF avant Slayer.
	if got[0].Value != "CTF" || got[0].Count != 1 {
		t.Errorf("CTF: got %+v, want Count=1", got[0])
	}
	if got[1].Value != "Slayer" || got[1].Count != 2 {
		t.Errorf("Slayer: got %+v, want Count=2", got[1])
	}
}

func TestUniqueLabelValuesWithORCounts_WithSelection_OR(t *testing.T) {
	// Sélection [Slayer] : pour Slayer (cochée), count = matchs avec mode IN [Slayer] = 2.
	// Pour CTF (non coché), count = matchs avec mode IN [Slayer, CTF] = 3.
	rows := []domain.FilterMatchRow{
		makeFilterRow("m1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("m2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("m3", "CTF", "Streets", "Arena", false, false),
	}
	identity := func(rs []domain.FilterMatchRow) []domain.FilterMatchRow { return rs }

	got := uniqueLabelValuesWithORCounts(rows, modeUI, []string{"Slayer"}, identity)
	if len(got) != 2 {
		t.Fatalf("expected 2 modes")
	}
	if got[0].Value != "CTF" || got[0].Count != 3 {
		t.Errorf("CTF (non coché): want OR-count=3, got %+v", got[0])
	}
	if got[1].Value != "Slayer" || got[1].Count != 2 {
		t.Errorf("Slayer (coché): want count=2 (sélection actuelle), got %+v", got[1])
	}
}

func TestUniqueLabelValuesWithORCounts_DownFilterAppliedToCount(t *testing.T) {
	// Pour la cat Modes, le filtre DOWN (Maps) doit être appliqué au count.
	// rows : Slayer/Streets, Slayer/Bazaar, CTF/Streets
	// avec Maps = [Streets] :
	//   Slayer count = matchs(mode=Slayer, map=Streets) = 1
	//   CTF count    = matchs(mode=CTF, map=Streets)    = 1
	rows := []domain.FilterMatchRow{
		makeFilterRow("m1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("m2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("m3", "CTF", "Streets", "Arena", false, false),
	}
	mapsFilter := []string{"Streets"}
	applyMaps := func(rs []domain.FilterMatchRow) []domain.FilterMatchRow {
		return filterBySet(rs, mapsFilter, mapUI)
	}

	got := uniqueLabelValuesWithORCounts(rows, modeUI, nil, applyMaps)
	if len(got) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(got))
	}
	if got[0].Value != "CTF" || got[0].Count != 1 {
		t.Errorf("CTF + Maps=[Streets]: want 1, got %+v", got[0])
	}
	if got[1].Value != "Slayer" || got[1].Count != 1 {
		t.Errorf("Slayer + Maps=[Streets]: want 1, got %+v", got[1])
	}
}

// ─── buildAvailableOptions — bout à bout ──────────────────────────────────

func TestBuildAvailableOptions_CountsAreOR(t *testing.T) {
	rows := []domain.FilterMatchRow{
		makeFilterRow("m1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("m2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("m3", "CTF", "Streets", "Arena", false, false),
		makeFilterRow("m4", "CTF", "Bazaar", "BTB", false, false),
	}
	got := buildAvailableOptions(rows, domain.CascadeFilter{Modes: []string{"Slayer"}})

	// Modes : 2 options.
	if len(got.Modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(got.Modes))
	}
	// CTF non coché : count = matchs avec mode IN [Slayer, CTF] = 4.
	// Slayer coché : count = matchs avec mode IN [Slayer] = 2.
	for _, m := range got.Modes {
		switch m.Value {
		case "CTF":
			if m.Count != 4 {
				t.Errorf("CTF count: want 4 (OR), got %d", m.Count)
			}
		case "Slayer":
			if m.Count != 2 {
				t.Errorf("Slayer count: want 2 (selection actuelle), got %d", m.Count)
			}
		}
	}
}

// ─── buildPeriodPresetCounts ──────────────────────────────────────────────

func TestBuildPeriodPresetCounts(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	rows := []domain.FilterMatchRow{
		// Match d'aujourd'hui → entre dans tous les presets sauf "all" trivialement.
		{MatchID: "m_today", StartTime: tsPtr(now.AddDate(0, 0, -1))},
		// Match de 10 jours → entre dans 30j et 90j.
		{MatchID: "m_10d", StartTime: tsPtr(now.AddDate(0, 0, -10))},
		// Match de 60 jours → entre dans 90j seulement.
		{MatchID: "m_60d", StartTime: tsPtr(now.AddDate(0, 0, -60))},
		// Match de 100 jours → entre seulement dans "all".
		{MatchID: "m_100d", StartTime: tsPtr(now.AddDate(0, 0, -100))},
	}
	got := buildPeriodPresetCounts(rows, domain.CascadeFilter{}, now)

	expected := map[string]int{"7d": 1, "30d": 2, "90d": 3, "all": 4}
	if len(got) != len(expected) {
		t.Fatalf("expected %d presets, got %d", len(expected), len(got))
	}
	for _, p := range got {
		want, ok := expected[p.PresetID]
		if !ok {
			t.Errorf("unexpected preset %q", p.PresetID)
			continue
		}
		if p.Count != want {
			t.Errorf("preset %q: want %d, got %d", p.PresetID, want, p.Count)
		}
	}
}

// ─── buildSessionOptions — MatchCountFiltered ──────────────────────────────

func TestBuildSessionOptions_MatchCountFiltered(t *testing.T) {
	lblA, lblB := "Session A", "Session B"
	sidA, sidB := "sa", "sb"
	rows := []domain.FilterMatchRow{
		// Session A : 2 matchs, dont 1 Slayer/Streets et 1 CTF/Bazaar.
		{MatchID: "m1", SessionLabel: &lblA, SessionID: &sidA, PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer"), MapName: strPtr("Streets")},
		{MatchID: "m2", SessionLabel: &lblA, SessionID: &sidA, PairName: strPtr("CTF"), PairNameFR: strPtr("CTF"), MapName: strPtr("Bazaar")},
		// Session B : 1 match Slayer/Streets.
		{MatchID: "m3", SessionLabel: &lblB, SessionID: &sidB, PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer"), MapName: strPtr("Streets")},
	}

	// Sans cascade : MatchCountFiltered == MatchCount.
	got := buildSessionOptions(rows, domain.CascadeFilter{})
	if len(got.AllSessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got.AllSessions))
	}
	for _, s := range got.AllSessions {
		if s.MatchCount != s.MatchCountFiltered {
			t.Errorf("session %q: MatchCount=%d != MatchCountFiltered=%d", s.Label, s.MatchCount, s.MatchCountFiltered)
		}
	}

	// Avec cascade Modes=[Slayer] : Session A perd le CTF (filtered=1), Session B garde son match (filtered=1).
	got = buildSessionOptions(rows, domain.CascadeFilter{Modes: []string{"Slayer"}})
	for _, s := range got.AllSessions {
		switch s.Label {
		case lblA:
			if s.MatchCount != 2 || s.MatchCountFiltered != 1 {
				t.Errorf("Session A: want MatchCount=2, MatchCountFiltered=1, got %+v", s)
			}
		case lblB:
			if s.MatchCount != 1 || s.MatchCountFiltered != 1 {
				t.Errorf("Session B: want MatchCount=1, MatchCountFiltered=1, got %+v", s)
			}
		}
	}

	// Avec cascade Modes=[Oddball] (aucun match) : MatchCountFiltered=0 partout.
	got = buildSessionOptions(rows, domain.CascadeFilter{Modes: []string{"Oddball"}})
	for _, s := range got.AllSessions {
		if s.MatchCountFiltered != 0 {
			t.Errorf("session %q with incompatible cascade: want MatchCountFiltered=0, got %d", s.Label, s.MatchCountFiltered)
		}
	}
}
