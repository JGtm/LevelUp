package service

// filters_cascade_test.go — tests de compatibilité des filtres en cascade.
//
// Couvre les 4 niveaux : ExperienceTypes → Playlists → Modes → Maps.
//
// Scénarios :
//  1. Experience types dynamiques (seuls les types présents dans les données sont disponibles).
//  2. Options de playlist cascadées depuis le filtre d'expérience.
//  3. Options de mode cascadées depuis expérience + playlist.
//  4. Options de map cascadées depuis expérience + playlist + mode.
//  5. Zombie expérience : type sélectionné absent → playlists/modes/maps vides.
//  6. Zombie playlist : playlist incompatible → modes/maps vides.
//  7. Zombie mode : mode incompatible → maps vides.
//  8. Zombie map : map absente après cascade complète → TotalMatchesAfterFilters=0.
//  9. Chaîne complète compatible (tous les 4 niveaux → 1 match).
// 10. Session + cascade : session réduit les lignes temporelles, types d'expérience réduits.
// 11. Session zombie expérience : type non présent dans la session.
// 12. Session + cascade playlist+mode+map.
// 13. Deux types d'expérience sélectionnés (sélection partielle).
// 14. Données vides : aucune option disponible.

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

func mkFilterRow(matchID string, opts ...func(*domain.FilterMatchRow)) domain.FilterMatchRow {
	r := domain.FilterMatchRow{MatchID: matchID}
	for _, o := range opts {
		o(&r)
	}
	return r
}

func fWithPlaylist(pl string) func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) { r.PlaylistName = strPtr(pl) }
}

func fWithMode(m string) func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) { r.PairName = strPtr(m) }
}

func fWithMap(m string) func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) { r.MapName = strPtr(m) }
}

// fWithVariant renseigne le game_variant (EN + FR) SANS pair_name : reproduit le
// modèle Halo 5 où le mode vient du game_variant (pair_id/pair_name NULL).
func fWithVariant(en, fr string) func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) {
		r.GameVariantName = strPtr(en)
		r.GameVariantNameFR = strPtr(fr)
	}
}

func fWithRanked() func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) { r.IsRanked = true }
}

func fWithFirefight() func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) { r.IsFirefight = true }
}

func fWithSession(lbl string) func(*domain.FilterMatchRow) {
	return func(r *domain.FilterMatchRow) {
		r.SessionLabel = strPtr(lbl)
		t := time.Now()
		r.StartTime = &t
	}
}

// hasLabel returns true if lv contains a LabelValue with the given value.
func hasLabel(lv []domain.LabelValue, val string) bool {
	for _, l := range lv {
		if l.Value == val {
			return true
		}
	}
	return false
}

// labelValues extracts the .Value strings from a LabelValue slice.
func labelValues(lv []domain.LabelValue) []string {
	out := make([]string, len(lv))
	for i, l := range lv {
		out[i] = l.Value
	}
	return out
}

// ---------------------------------------------------------------------------
// 1. Experience types — dynamiques
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_ExperienceType_OnlyUnrankedPresent(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("CTF")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.ExperienceTypes) != 1 {
		t.Fatalf("expected 1 experience type, got %d: %v", len(avail.ExperienceTypes), labelValues(avail.ExperienceTypes))
	}
	if avail.ExperienceTypes[0].Value != "PVP non classé" {
		t.Errorf("expected 'PVP non classé', got %q", avail.ExperienceTypes[0].Value)
	}
}

func TestBuildAvailableOptions_ExperienceType_AllThreePresent(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1"),                   // unranked
		mkFilterRow("m2", fWithRanked()),    // ranked
		mkFilterRow("m3", fWithFirefight()), // PvE
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.ExperienceTypes) != 3 {
		t.Fatalf("expected 3 experience types, got %d: %v", len(avail.ExperienceTypes), labelValues(avail.ExperienceTypes))
	}
	// Ordre canonique : "PVP non classé", "PVP classé", "PVE"
	expected := []string{"PVP non classé", "PVP classé", "PVE"}
	for i, want := range expected {
		if avail.ExperienceTypes[i].Value != want {
			t.Errorf("ExperienceTypes[%d]: got %q, want %q", i, avail.ExperienceTypes[i].Value, want)
		}
	}
}

func TestBuildAvailableOptions_ExperienceType_RankedOnly(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Slayer")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.ExperienceTypes) != 1 || avail.ExperienceTypes[0].Value != "PVP classé" {
		t.Errorf("expected ['PVP classé'], got %v", labelValues(avail.ExperienceTypes))
	}
}

// ---------------------------------------------------------------------------
// 2. Playlist cascade depuis le filtre d'expérience
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Playlist_FilteredByExperience(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer")),                   // unranked
		mkFilterRow("m2", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Oddball")), // ranked
	}
	// Filtre : seulement PVP classé
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP classé"},
	})
	if len(avail.Playlists) != 1 || avail.Playlists[0].Value != "Ranked Arena" {
		t.Errorf("expected Playlists=['Ranked Arena'], got %v", labelValues(avail.Playlists))
	}
	if hasLabel(avail.Playlists, "Quick Play") {
		t.Error("Quick Play should not appear when experience filter = PVP classé")
	}
}

func TestBuildAvailableOptions_Playlist_NoExperienceFilter_AllShown(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play")),
		mkFilterRow("m2", fWithRanked(), fWithPlaylist("Ranked Arena")),
		mkFilterRow("m3", fWithFirefight(), fWithPlaylist("Custom Games")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.Playlists) != 3 {
		t.Errorf("expected 3 playlists (no filter), got %d: %v", len(avail.Playlists), labelValues(avail.Playlists))
	}
}

func TestBuildAvailableOptions_Playlist_PvEExperienceFilter(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play")),                  // unranked
		mkFilterRow("m2", fWithFirefight(), fWithPlaylist("PvE Co-op")), // PvE
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVE"},
	})
	if len(avail.Playlists) != 1 || avail.Playlists[0].Value != "PvE Co-op" {
		t.Errorf("expected Playlists=['PvE Co-op'], got %v", labelValues(avail.Playlists))
	}
}

// ---------------------------------------------------------------------------
// 3. Mode cascade depuis expérience + playlist
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Mode_FilteredByPlaylist(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("CTF")),
		mkFilterRow("m3", fWithPlaylist("Social"), fWithMode("SWAT")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Quick Play"},
	})
	if len(avail.Modes) != 2 {
		t.Fatalf("expected 2 modes (Slayer+CTF), got %d: %v", len(avail.Modes), labelValues(avail.Modes))
	}
	if !hasLabel(avail.Modes, "Slayer") || !hasLabel(avail.Modes, "CTF") {
		t.Errorf("expected Slayer and CTF in modes, got %v", labelValues(avail.Modes))
	}
	if hasLabel(avail.Modes, "SWAT") {
		t.Error("SWAT should not appear (belongs to Social, not Quick Play)")
	}
}

func TestBuildAvailableOptions_Mode_FilteredByExperienceAndPlaylist(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Slayer")),
		mkFilterRow("m2", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Oddball")),
		mkFilterRow("m3", fWithPlaylist("Quick Play"), fWithMode("CTF")), // unranked → excluded by exp filter
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP classé"},
		Playlists:       []string{"Ranked Arena"},
	})
	if len(avail.Modes) != 2 {
		t.Fatalf("expected 2 modes, got %d: %v", len(avail.Modes), labelValues(avail.Modes))
	}
	if !hasLabel(avail.Modes, "Slayer") || !hasLabel(avail.Modes, "Oddball") {
		t.Errorf("expected Slayer and Oddball, got %v", labelValues(avail.Modes))
	}
	if hasLabel(avail.Modes, "CTF") {
		t.Error("CTF should not appear (unranked row excluded by experience filter)")
	}
}

// ---------------------------------------------------------------------------
// 4. Map cascade depuis expérience + playlist + mode
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Map_FilteredByMode(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Streets")),
		mkFilterRow("m3", fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Bazaar")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Quick Play"},
		Modes:     []string{"Slayer"},
	})
	if len(avail.Maps) != 2 {
		t.Fatalf("expected 2 maps (Aquarius+Streets), got %d: %v", len(avail.Maps), labelValues(avail.Maps))
	}
	if !hasLabel(avail.Maps, "Aquarius") || !hasLabel(avail.Maps, "Streets") {
		t.Errorf("expected Aquarius and Streets, got %v", labelValues(avail.Maps))
	}
	if hasLabel(avail.Maps, "Bazaar") {
		t.Error("Bazaar should not appear (belongs to CTF, not Slayer)")
	}
}

func TestBuildAvailableOptions_Map_FilteredByAllLevels(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Oddball"), fWithMap("Recharge")),
		mkFilterRow("m3", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Streets")), // unranked
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP classé"},
		Playlists:       []string{"Ranked Arena"},
		Modes:           []string{"Slayer"},
	})
	if len(avail.Maps) != 1 || avail.Maps[0].Value != "Aquarius" {
		t.Errorf("expected Maps=['Aquarius'], got %v", labelValues(avail.Maps))
	}
}

// ---------------------------------------------------------------------------
// 5. Zombie expérience : type sélectionné absent des données → playlists vides
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Zombie_ExperienceNotInData(t *testing.T) {
	// Données : uniquement PvP non classé
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Streets")),
	}
	// Utilisateur a sélectionné "PVP classé" (zombie — absent des données)
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP classé"},
	})
	// ExperienceTypes disponibles : seulement "PVP non classé" (ce qui est dans les données)
	if len(avail.ExperienceTypes) != 1 || avail.ExperienceTypes[0].Value != "PVP non classé" {
		t.Errorf("ExperienceTypes should only contain 'PVP non classé', got %v", labelValues(avail.ExperienceTypes))
	}
	// Playlists, Modes, Maps vides car l'exp filter n'a rien retenu
	if len(avail.Playlists) != 0 {
		t.Errorf("Playlists should be empty (zombie experience), got %v", labelValues(avail.Playlists))
	}
	if len(avail.Modes) != 0 {
		t.Errorf("Modes should be empty (zombie experience), got %v", labelValues(avail.Modes))
	}
	if len(avail.Maps) != 0 {
		t.Errorf("Maps should be empty (zombie experience), got %v", labelValues(avail.Maps))
	}
}

func TestResolveFilters_Zombie_ExperienceNotInData_CountZero(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play")),
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{ExperienceTypes: []string{"PVP classé"}},
	})
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 matches (zombie experience), got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

// ---------------------------------------------------------------------------
// 6. Zombie playlist : playlist incompatible → modes/maps vides
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Zombie_PlaylistNotInData(t *testing.T) {
	// Données : Quick Play uniquement
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
	}
	// Utilisateur a "Ranked Arena" sélectionné (zombie)
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Ranked Arena"},
	})
	// Playlists disponibles : Quick Play (pas Ranked Arena)
	if len(avail.Playlists) != 1 || avail.Playlists[0].Value != "Quick Play" {
		t.Errorf("Playlists should be ['Quick Play'], got %v", labelValues(avail.Playlists))
	}
	// Modes et maps vides (aucun match ne correspond à Ranked Arena)
	if len(avail.Modes) != 0 {
		t.Errorf("Modes should be empty (zombie playlist), got %v", labelValues(avail.Modes))
	}
	if len(avail.Maps) != 0 {
		t.Errorf("Maps should be empty (zombie playlist), got %v", labelValues(avail.Maps))
	}
}

func TestResolveFilters_Zombie_PlaylistNotInData_CountZero(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play")),
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Playlists: []string{"Ranked Arena"}},
	})
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 matches (zombie playlist), got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

// ---------------------------------------------------------------------------
// 7. Zombie mode : mode incompatible avec la playlist → maps vides
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_Zombie_ModeNotInPlaylist(t *testing.T) {
	// Données : Quick Play contient Slayer et CTF
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Streets")),
	}
	// Utilisateur a "SWAT" sélectionné (zombie — pas dans Quick Play)
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Quick Play"},
		Modes:     []string{"SWAT"},
	})
	// Modes disponibles : Slayer et CTF (pas SWAT)
	if hasLabel(avail.Modes, "SWAT") {
		t.Error("SWAT should not appear in available modes (zombie)")
	}
	if !hasLabel(avail.Modes, "Slayer") || !hasLabel(avail.Modes, "CTF") {
		t.Errorf("Slayer and CTF should be in available modes, got %v", labelValues(avail.Modes))
	}
	// Maps vides car SWAT ne donne aucun résultat
	if len(avail.Maps) != 0 {
		t.Errorf("Maps should be empty (zombie mode), got %v", labelValues(avail.Maps))
	}
}

func TestResolveFilters_Zombie_ModeNotInPlaylist_CountZero(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer")),
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Quick Play"},
			Modes:     []string{"SWAT"},
		},
	})
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 matches (zombie mode), got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

// ---------------------------------------------------------------------------
// 8. Zombie map : map absente du filtre mode → count=0
// ---------------------------------------------------------------------------

func TestResolveFilters_Zombie_MapNotInMode_CountZero(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Streets")),
	}
	// Utilisateur a "Bazaar" sélectionné (zombie — pas dans Slayer)
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Quick Play"},
			Modes:     []string{"Slayer"},
			Maps:      []string{"Bazaar"},
		},
	})
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 matches (zombie map), got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

func TestBuildAvailableOptions_Zombie_MapNotInMode_MapsShowReal(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Streets")),
	}
	// Maps n'est pas utilisé dans buildAvailableOptions pour réduire mapOpts —
	// les maps disponibles sont celles obtenues après exp+playlist+mode.
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		Playlists: []string{"Quick Play"},
		Modes:     []string{"Slayer"},
	})
	if len(avail.Maps) != 2 {
		t.Fatalf("expected 2 available maps, got %d: %v", len(avail.Maps), labelValues(avail.Maps))
	}
	if !hasLabel(avail.Maps, "Aquarius") || !hasLabel(avail.Maps, "Streets") {
		t.Errorf("expected Aquarius and Streets, got %v", labelValues(avail.Maps))
	}
}

// ---------------------------------------------------------------------------
// 9. Chaîne complète compatible (tous les 4 niveaux → 1 match)
// ---------------------------------------------------------------------------

func TestResolveFilters_FullCompatibleCascade_OnlyOneMatch(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),                  // correspond
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Streets")),                   // même mode/playlist mais map diff
		mkFilterRow("m3", fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Aquarius")),                     // même map mais mode diff
		mkFilterRow("m4", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Slayer"), fWithMap("Aquarius")), // même tout sauf exp
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			ExperienceTypes: []string{"PVP non classé"},
			Playlists:       []string{"Quick Play"},
			Modes:           []string{"Slayer"},
			Maps:            []string{"Aquarius"},
		},
	})
	if res.Counts.TotalMatchesAfterFilters != 1 {
		t.Errorf("expected 1 match (full compatible cascade), got %d", res.Counts.TotalMatchesAfterFilters)
	}
	if res.Counts.TotalMatchesBeforeFilters != 4 {
		t.Errorf("expected 4 matches before filters, got %d", res.Counts.TotalMatchesBeforeFilters)
	}
}

func TestBuildAvailableOptions_FullCompatibleCascade_OptionsCorrect(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Streets")),
		mkFilterRow("m3", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Oddball"), fWithMap("Bazaar")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP non classé"},
		Playlists:       []string{"Quick Play"},
		Modes:           []string{"Slayer"},
	})
	// ExperienceTypes : 2 types présents dans les données (pas PvE)
	if len(avail.ExperienceTypes) != 2 {
		t.Errorf("expected 2 experience types, got %v", labelValues(avail.ExperienceTypes))
	}
	// Playlists : seulement Quick Play (exp filter = unranked → exclut Ranked Arena)
	if len(avail.Playlists) != 1 || avail.Playlists[0].Value != "Quick Play" {
		t.Errorf("expected Playlists=['Quick Play'], got %v", labelValues(avail.Playlists))
	}
	// Modes : Slayer + CTF (dans Quick Play unranked)
	if len(avail.Modes) != 2 {
		t.Errorf("expected 2 modes, got %v", labelValues(avail.Modes))
	}
	// Maps : seulement Aquarius (Quick Play + Slayer)
	if len(avail.Maps) != 1 || avail.Maps[0].Value != "Aquarius" {
		t.Errorf("expected Maps=['Aquarius'], got %v", labelValues(avail.Maps))
	}
}

// ---------------------------------------------------------------------------
// 10. Session + cascade : session réduit les types d'expérience disponibles
// ---------------------------------------------------------------------------

func TestResolveFilters_Session_ReducesExperienceTypes(t *testing.T) {
	const session = "2026-04-21 19h"
	rows := []domain.FilterMatchRow{
		// Dans la session : uniquement unranked
		mkFilterRow("m1", fWithSession(session), fWithPlaylist("Quick Play"), fWithMode("Slayer")),
		mkFilterRow("m2", fWithSession(session), fWithPlaylist("Quick Play"), fWithMode("CTF")),
		// Hors session : ranked (ne doit pas influencer les options disponibles)
		mkFilterRow("m3", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Oddball")),
	}
	// Pas de startTime sur m3 — mais le filter mode est "sessions" avec label
	t3 := time.Now()
	rows[2].StartTime = &t3

	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		FilterMode: "sessions",
		Sessions: domain.SessionsFilter{
			PickedSessions: []string{session},
		},
	})
	// Seulement les rows de la session sont dans temporal
	if res.Counts.TotalMatchesBeforeFilters != 3 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 3", res.Counts.TotalMatchesBeforeFilters)
	}
	if res.Counts.TotalMatchesAfterFilters != 2 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 2 (session only)", res.Counts.TotalMatchesAfterFilters)
	}
	// ExperienceTypes disponibles : seulement "PVP non classé" (la session ne contient pas de ranked)
	if len(res.AvailableOptions.ExperienceTypes) != 1 {
		t.Fatalf("expected 1 experience type in session, got %d: %v",
			len(res.AvailableOptions.ExperienceTypes), labelValues(res.AvailableOptions.ExperienceTypes))
	}
	if res.AvailableOptions.ExperienceTypes[0].Value != "PVP non classé" {
		t.Errorf("expected 'PVP non classé', got %q", res.AvailableOptions.ExperienceTypes[0].Value)
	}
}

// ---------------------------------------------------------------------------
// 11. Session zombie expérience : type non présent dans la session
// ---------------------------------------------------------------------------

func TestResolveFilters_Session_Zombie_ExperienceNotInSession(t *testing.T) {
	const session = "2026-04-21 19h"
	rows := []domain.FilterMatchRow{
		// Session : unranked seulement
		mkFilterRow("m1", fWithSession(session), fWithPlaylist("Quick Play")),
		// Hors session : ranked
		func() domain.FilterMatchRow {
			r := mkFilterRow("m3", fWithRanked(), fWithPlaylist("Ranked Arena"))
			t := time.Now()
			r.StartTime = &t
			return r
		}(),
	}
	// Utilisateur a sélectionné "PVP classé" + session → combo zombie
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		FilterMode: "sessions",
		Sessions: domain.SessionsFilter{
			PickedSessions: []string{session},
		},
		Cascade: domain.CascadeFilter{
			ExperienceTypes: []string{"PVP classé"},
		},
	})
	// "PVP classé" n'est pas dans la session → zombie
	if hasLabel(res.AvailableOptions.ExperienceTypes, "PVP classé") {
		t.Error("'PVP classé' should not be in available experience types (not in session)")
	}
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 matches (zombie exp + session), got %d", res.Counts.TotalMatchesAfterFilters)
	}
	// Playlists vides car le filtre exp ("PVP classé") ne match rien dans la session
	if len(res.AvailableOptions.Playlists) != 0 {
		t.Errorf("Playlists should be empty (zombie exp), got %v", labelValues(res.AvailableOptions.Playlists))
	}
}

// ---------------------------------------------------------------------------
// 12. Session + cascade playlist + mode + map
// ---------------------------------------------------------------------------

func TestResolveFilters_Session_CascadeAllLevels_OnlySessionMatchesContribute(t *testing.T) {
	const session = "2026-04-21 19h"
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithSession(session), fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius")),
		mkFilterRow("m2", fWithSession(session), fWithPlaylist("Quick Play"), fWithMode("CTF"), fWithMap("Streets")),
		// Hors session — même playlist/mode/map
		func() domain.FilterMatchRow {
			r := mkFilterRow("m3", fWithPlaylist("Quick Play"), fWithMode("Slayer"), fWithMap("Aquarius"))
			t := time.Now()
			r.StartTime = &t
			return r
		}(),
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		FilterMode: "sessions",
		Sessions: domain.SessionsFilter{
			PickedSessions: []string{session},
		},
		Cascade: domain.CascadeFilter{
			Playlists: []string{"Quick Play"},
			Modes:     []string{"Slayer"},
			Maps:      []string{"Aquarius"},
		},
	})
	// Seulement m1 correspond (session + Quick Play + Slayer + Aquarius)
	if res.Counts.TotalMatchesAfterFilters != 1 {
		t.Errorf("expected 1 match (session+cascade), got %d", res.Counts.TotalMatchesAfterFilters)
	}
	// Options maps disponibles dans la session pour Quick Play+Slayer : Aquarius seulement
	if len(res.AvailableOptions.Maps) != 1 || res.AvailableOptions.Maps[0].Value != "Aquarius" {
		t.Errorf("expected Maps=['Aquarius'] in session, got %v", labelValues(res.AvailableOptions.Maps))
	}
}

// ---------------------------------------------------------------------------
// 13. Deux types d'expérience sélectionnés (sélection partielle)
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_TwoExperienceTypes_ExcludesPvE(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1", fWithPlaylist("Quick Play"), fWithMode("Slayer")),
		mkFilterRow("m2", fWithRanked(), fWithPlaylist("Ranked Arena"), fWithMode("Slayer")),
		mkFilterRow("m3", fWithFirefight(), fWithPlaylist("PvE Co-op"), fWithMode("Firefight")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{
		ExperienceTypes: []string{"PVP non classé", "PVP classé"},
	})
	// Playlists : Quick Play + Ranked Arena (PvE exclu)
	if hasLabel(avail.Playlists, "PvE Co-op") {
		t.Error("PvE Co-op should not appear when experience filter excludes PvE")
	}
	if !hasLabel(avail.Playlists, "Quick Play") || !hasLabel(avail.Playlists, "Ranked Arena") {
		t.Errorf("expected Quick Play and Ranked Arena, got %v", labelValues(avail.Playlists))
	}
}

func TestResolveFilters_TwoExperienceTypes_CorrectCount(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("m1"),                   // unranked
		mkFilterRow("m2", fWithRanked()),    // ranked
		mkFilterRow("m3", fWithFirefight()), // PvE
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{
			ExperienceTypes: []string{"PVP non classé", "PVP classé"},
		},
	})
	// m1 + m2 correspondent (PvE exclu)
	if res.Counts.TotalMatchesAfterFilters != 2 {
		t.Errorf("expected 2 matches (unranked+ranked), got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

// ---------------------------------------------------------------------------
// 14. Données vides : aucune option disponible
// ---------------------------------------------------------------------------

func TestBuildAvailableOptions_EmptyRows_AllOptionsEmpty(t *testing.T) {
	avail := buildAvailableOptions(nil, domain.CascadeFilter{})
	if len(avail.ExperienceTypes) != 0 {
		t.Errorf("expected 0 experience types for empty rows, got %v", labelValues(avail.ExperienceTypes))
	}
	if len(avail.Playlists) != 0 {
		t.Errorf("expected 0 playlists, got %v", labelValues(avail.Playlists))
	}
	if len(avail.Modes) != 0 {
		t.Errorf("expected 0 modes, got %v", labelValues(avail.Modes))
	}
	if len(avail.Maps) != 0 {
		t.Errorf("expected 0 maps, got %v", labelValues(avail.Maps))
	}
}

func TestResolveFilters_EmptyRows_ZeroCounts(t *testing.T) {
	res := ResolveFiltersFromRows(nil, domain.FilterContextInput{})
	if res.Counts.TotalMatchesBeforeFilters != 0 {
		t.Errorf("expected 0 before, got %d", res.Counts.TotalMatchesBeforeFilters)
	}
	if res.Counts.TotalMatchesAfterFilters != 0 {
		t.Errorf("expected 0 after, got %d", res.Counts.TotalMatchesAfterFilters)
	}
}

// ---------------------------------------------------------------------------
// 15. Halo 5 — mode dérivé du game_variant (pair_name NULL)
// ---------------------------------------------------------------------------

// TestBuildAvailableOptions_H5VariantMode_ModesFromVariant : sur un titre sans
// pair (H5), le mode est dérivé du game_variant (FR préféré) → la catégorie Modes
// n'est plus vide. C'est le symptôme corrigé (filtres L2 grisés sur Halo 5).
func TestBuildAvailableOptions_H5VariantMode_ModesFromVariant(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("h5a", fWithPlaylist("Team Arena"), fWithVariant("Team Slayer", "Assassin en équipe"), fWithMap("Truth")),
		mkFilterRow("h5b", fWithPlaylist("Team Arena"), fWithVariant("Capture the Flag", "Capture du drapeau"), fWithMap("Coliseum")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.Modes) != 2 {
		t.Fatalf("expected 2 modes from variant, got %d: %v", len(avail.Modes), labelValues(avail.Modes))
	}
	if !hasLabel(avail.Modes, "Assassin en équipe") || !hasLabel(avail.Modes, "Capture du drapeau") {
		t.Errorf("expected FR variant modes, got %v", labelValues(avail.Modes))
	}
}

// TestResolveFilters_H5VariantMode_CascadeMatchesRow : sélectionner un mode dérivé
// du game_variant filtre bien les rows (Value d'option == clé de filtrage, garanti
// par le chokepoint modeUI).
func TestResolveFilters_H5VariantMode_CascadeMatchesRow(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("h5a", fWithPlaylist("Team Arena"), fWithVariant("Team Slayer", "Assassin en équipe"), fWithMap("Truth")),
		mkFilterRow("h5b", fWithPlaylist("Team Arena"), fWithVariant("Capture the Flag", "Capture du drapeau"), fWithMap("Coliseum")),
	}
	res := ResolveFiltersFromRows(rows, domain.FilterContextInput{
		Cascade: domain.CascadeFilter{Modes: []string{"Assassin en équipe"}},
	})
	if res.Counts.TotalMatchesBeforeFilters != 2 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 2", res.Counts.TotalMatchesBeforeFilters)
	}
	if res.Counts.TotalMatchesAfterFilters != 1 {
		t.Errorf("TotalMatchesAfterFilters = %d, want 1 (mode dérivé du game_variant)", res.Counts.TotalMatchesAfterFilters)
	}
}

// TestBuildAvailableOptions_InfinitePairWins_VariantIgnored : contre-épreuve
// Infinite — quand le pair_name est rempli, il prime et le game_variant est
// ignoré (le fallback ne s'active que pair absent) → comportement inchangé.
func TestBuildAvailableOptions_InfinitePairWins_VariantIgnored(t *testing.T) {
	rows := []domain.FilterMatchRow{
		mkFilterRow("inf1", fWithPlaylist("Quick Play"), fWithMode("Arena:Slayer"),
			fWithVariant("Team Slayer", "Assassin en équipe"), fWithMap("Streets")),
	}
	avail := buildAvailableOptions(rows, domain.CascadeFilter{})
	if len(avail.Modes) != 1 || avail.Modes[0].Value != "Slayer" {
		t.Errorf("expected Modes=['Slayer'] (pair prime, variant ignoré), got %v", labelValues(avail.Modes))
	}
}
