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

// TestBuildAvailableOptions_ExperienceFiltersPlaylistsModesMaps cadenasse le
// comportement attendu sur le terrain : si l'utilisateur coche Experience=PVE,
// les Playlists/Modes/Maps non-PVE doivent disparaître complètement de la
// liste retournée (pas juste avoir count=0).
//
// Bug observé : sur un dataset avec 2 matchs PVE et 50+ matchs PVP, cocher
// PVE laissait toutes les options PVP visibles côté UI car la liste backend
// les retournait avec un count > 0.
func TestBuildAvailableOptions_ExperienceFiltersPlaylistsModesMaps(t *testing.T) {
	rows := []domain.FilterMatchRow{
		// 2 matchs PVE — Firefight sur Sanctuary
		{MatchID: "pve1", PairName: strPtr("Firefight"), PairNameFR: strPtr("Firefight"), MapName: strPtr("Sanctuary"), MapNameFR: strPtr("Sanctuary"), PlaylistName: strPtr("Firefight"), IsFirefight: true},
		{MatchID: "pve2", PairName: strPtr("Firefight"), PairNameFR: strPtr("Firefight"), MapName: strPtr("Sanctuary"), MapNameFR: strPtr("Sanctuary"), PlaylistName: strPtr("Firefight"), IsFirefight: true},
		// 5 matchs PVP — Slayer/CTF/Strongholds sur Streets/Bazaar/Aquarius
		makeFilterRow("pvp1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("pvp2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("pvp3", "CTF", "Streets", "Arena", false, false),
		makeFilterRow("pvp4", "CTF", "Bazaar", "BTB", false, false),
		makeFilterRow("pvp5", "Strongholds", "Aquarius", "BTB", false, false),
	}

	got := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVE"},
	})

	// Playlists : seule "Firefight" doit apparaître.
	playlistVals := make([]string, 0, len(got.Playlists))
	for _, p := range got.Playlists {
		playlistVals = append(playlistVals, p.Value)
	}
	if len(got.Playlists) != 1 || got.Playlists[0].Value != "Firefight" {
		t.Errorf("Playlists avec Experience=[PVE] : want [Firefight], got %v", playlistVals)
	}

	// Modes : seul "Firefight" doit apparaître.
	modeVals := make([]string, 0, len(got.Modes))
	for _, m := range got.Modes {
		modeVals = append(modeVals, m.Value)
	}
	if len(got.Modes) != 1 || got.Modes[0].Value != "Firefight" {
		t.Errorf("Modes avec Experience=[PVE] : want [Firefight], got %v", modeVals)
	}

	// Maps : seule "Sanctuary" doit apparaître.
	mapVals := make([]string, 0, len(got.Maps))
	for _, m := range got.Maps {
		mapVals = append(mapVals, m.Value)
	}
	if len(got.Maps) != 1 || got.Maps[0].Value != "Sanctuary" {
		t.Errorf("Maps avec Experience=[PVE] : want [Sanctuary], got %v", mapVals)
	}
}

// TestBuildAvailableOptions_PlaylistFiltersDownStream : cocher une playlist
// doit masquer les modes/cartes non joués sur cette playlist.
func TestBuildAvailableOptions_PlaylistFiltersDownStream(t *testing.T) {
	rows := []domain.FilterMatchRow{
		makeFilterRow("a1", "Slayer", "Streets", "Arena", false, false),
		makeFilterRow("a2", "Slayer", "Bazaar", "Arena", false, false),
		makeFilterRow("b1", "Strongholds", "Aquarius", "BTB", false, false),
	}

	got := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Arena"},
	})

	if len(got.Modes) != 1 || got.Modes[0].Value != "Slayer" {
		vals := []string{}
		for _, m := range got.Modes {
			vals = append(vals, m.Value)
		}
		t.Errorf("Modes avec Playlists=[Arena] : want [Slayer], got %v", vals)
	}
	mapVals := make([]string, 0, len(got.Maps))
	for _, m := range got.Maps {
		mapVals = append(mapVals, m.Value)
	}
	if len(got.Maps) != 2 {
		t.Errorf("Maps avec Playlists=[Arena] : want 2 (Bazaar + Streets), got %d %v", len(got.Maps), mapVals)
	}
}

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

// ─── BuildSeasonCounts ────────────────────────────────────────────────────

func TestBuildSeasonCounts_BasicWindowCount(t *testing.T) {
	// 3 saisons disjointes : S1 [2022-01, 2022-04), S2 [2022-04, 2022-07), S3 [2022-07, 2022-10)
	s1End := time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	s2End := time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)
	s3End := time.Date(2022, 10, 1, 0, 0, 0, 0, time.UTC)
	seasons := []SeasonWindow{
		{ID: "s1", Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), End: &s1End},
		{ID: "s2", Start: time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC), End: &s2End},
		{ID: "s3", Start: time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC), End: &s3End},
	}
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", StartTime: tsPtr(time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC))}, // s1
		{MatchID: "m2", StartTime: tsPtr(time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC))},  // s1
		{MatchID: "m3", StartTime: tsPtr(time.Date(2022, 5, 10, 0, 0, 0, 0, time.UTC))}, // s2
		{MatchID: "m4", StartTime: tsPtr(time.Date(2022, 8, 20, 0, 0, 0, 0, time.UTC))}, // s3
		{MatchID: "m5", StartTime: tsPtr(time.Date(2022, 9, 30, 0, 0, 0, 0, time.UTC))}, // s3
	}
	got := BuildSeasonCounts(rows, domain.CascadeFilter{}, seasons)
	want := map[string]int{"s1": 2, "s2": 1, "s3": 2}
	if len(got) != 3 {
		t.Fatalf("expected 3 season counts, got %d", len(got))
	}
	for _, c := range got {
		if c.Count != want[c.SeasonID] {
			t.Errorf("season %q: count=%d, want %d", c.SeasonID, c.Count, want[c.SeasonID])
		}
	}
}

func TestBuildSeasonCounts_CascadeApplied(t *testing.T) {
	s1End := time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)
	seasons := []SeasonWindow{
		{ID: "s1", Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), End: &s1End},
	}
	rows := []domain.FilterMatchRow{
		// 2 matchs Slayer + 3 matchs CTF, tous dans s1.
		{MatchID: "m1", StartTime: tsPtr(time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC)), PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer")},
		{MatchID: "m2", StartTime: tsPtr(time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC)), PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer")},
		{MatchID: "m3", StartTime: tsPtr(time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)), PairName: strPtr("CTF"), PairNameFR: strPtr("CTF")},
		{MatchID: "m4", StartTime: tsPtr(time.Date(2022, 5, 1, 0, 0, 0, 0, time.UTC)), PairName: strPtr("CTF"), PairNameFR: strPtr("CTF")},
		{MatchID: "m5", StartTime: tsPtr(time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)), PairName: strPtr("CTF"), PairNameFR: strPtr("CTF")},
	}
	got := BuildSeasonCounts(rows, domain.CascadeFilter{Modes: []string{"Slayer"}}, seasons)
	if len(got) != 1 {
		t.Fatalf("expected 1 season count, got %d", len(got))
	}
	if got[0].Count != 2 {
		t.Errorf("s1 avec cascade Modes=[Slayer]: count=%d, want 2", got[0].Count)
	}
}

func TestBuildSeasonCounts_OpenSeason(t *testing.T) {
	// Saison ouverte : Start = 2022-01-01, End = nil → tout match >= Start compte.
	seasons := []SeasonWindow{
		{ID: "s_open", Start: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), End: nil},
	}
	rows := []domain.FilterMatchRow{
		{MatchID: "m_before", StartTime: tsPtr(time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC))}, // exclu
		{MatchID: "m_in1", StartTime: tsPtr(time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC))},
		{MatchID: "m_in2", StartTime: tsPtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))},
		{MatchID: "m_far_future", StartTime: tsPtr(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))},
	}
	got := BuildSeasonCounts(rows, domain.CascadeFilter{}, seasons)
	if got[0].Count != 3 {
		t.Errorf("saison ouverte: count=%d, want 3 (tous les matchs >= Start)", got[0].Count)
	}
}

func TestBuildSeasonCounts_NoSeasonsInRegistry(t *testing.T) {
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", StartTime: tsPtr(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))},
	}
	got := BuildSeasonCounts(rows, domain.CascadeFilter{}, nil)
	if got != nil {
		t.Errorf("seasons nil → BuildSeasonCounts doit retourner nil (pas []), got %v", got)
	}
	got = BuildSeasonCounts(rows, domain.CascadeFilter{}, []SeasonWindow{})
	if got != nil {
		t.Errorf("seasons vide → BuildSeasonCounts doit retourner nil, got %v", got)
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
		// Session B : 2 matchs Slayer/Streets (≥ minListedSessionMatches pour être listée).
		{MatchID: "m3", SessionLabel: &lblB, SessionID: &sidB, PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer"), MapName: strPtr("Streets")},
		{MatchID: "m4", SessionLabel: &lblB, SessionID: &sidB, PairName: strPtr("Slayer"), PairNameFR: strPtr("Slayer"), MapName: strPtr("Streets")},
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

	// Avec cascade Modes=[Slayer] : Session A perd le CTF (filtered=1), Session B garde ses 2 matchs (filtered=2).
	got = buildSessionOptions(rows, domain.CascadeFilter{Modes: []string{"Slayer"}})
	for _, s := range got.AllSessions {
		switch s.Label {
		case lblA:
			if s.MatchCount != 2 || s.MatchCountFiltered != 1 {
				t.Errorf("Session A: want MatchCount=2, MatchCountFiltered=1, got %+v", s)
			}
		case lblB:
			if s.MatchCount != 2 || s.MatchCountFiltered != 2 {
				t.Errorf("Session B: want MatchCount=2, MatchCountFiltered=2, got %+v", s)
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
