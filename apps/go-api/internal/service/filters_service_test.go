package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/testutil"
)

// --- mock ---

type mockFiltersRepo struct {
	rows []domain.FilterMatchRow
	err  error
}

func (m *mockFiltersRepo) LoadMatchesForFilters(_ context.Context) ([]domain.FilterMatchRow, error) {
	return m.rows, m.err
}
func (m *mockFiltersRepo) GetMatchCount(_ context.Context) (int, error) { return len(m.rows), nil }
func (m *mockFiltersRepo) GetPlayerMatchCount(_ context.Context) (int, error) {
	return len(m.rows), nil
}
func (m *mockFiltersRepo) GetAvailablePlaylists(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}
func (m *mockFiltersRepo) GetAvailableMaps(_ context.Context) ([]domain.LabelValue, error) {
	return nil, nil
}

// --- tests FiltersService ---

func TestFiltersService_Resolve_OK(t *testing.T) {
	now := time.Now()
	repo := &mockFiltersRepo{
		rows: []domain.FilterMatchRow{
			{MatchID: "m1", StartTime: &now, MapName: strPtr("Aquarius"), PairName: strPtr("Slayer"), PlaylistName: strPtr("Ranked Arena")},
			{MatchID: "m2", StartTime: &now, MapName: strPtr("Streets"), PairName: strPtr("CTF"), PlaylistName: strPtr("Quick Play")},
		},
	}
	svc := NewFiltersService(repo)

	resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{FilterMode: "period"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Counts.TotalMatchesBeforeFilters != 2 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 2", resp.Counts.TotalMatchesBeforeFilters)
	}
}

// TestFiltersService_Resolve_ExperienceLabelsLocaleAware prouve GH5-2 : le LABEL
// des options d'expérience est localisé (EN sous locale EN) tandis que la VALUE
// reste FR dans les deux locales (contrat cascade/substring intact).
func TestFiltersService_Resolve_ExperienceLabelsLocaleAware(t *testing.T) {
	now := time.Now()
	repo := &mockFiltersRepo{
		rows: []domain.FilterMatchRow{
			{MatchID: "pve", StartTime: &now, IsFirefight: true},
			{MatchID: "ranked", StartTime: &now, IsRanked: true},
			{MatchID: "unranked", StartTime: &now},
		},
	}
	svc := NewFiltersService(repo)

	// VALUE FR canonique → LABEL EN attendu.
	wantEN := map[string]string{
		expTypePVPUnranked: "Unranked PvP",
		expTypePVPRanked:   "Ranked PvP",
		expTypePVE:         "PvE",
	}

	// Locale EN : Label localisé, Value FR.
	respEN, err := svc.Resolve(ctxkeys.WithLocale(context.Background(), "en"), domain.FilterContextInput{FilterMode: "period"})
	if err != nil {
		t.Fatalf("resolve EN: %v", err)
	}
	optsEN := respEN.AvailableOptions.ExperienceTypes
	if len(optsEN) != 3 {
		t.Fatalf("EN: %d options d'expérience, want 3", len(optsEN))
	}
	for _, o := range optsEN {
		if _, isFRValue := wantEN[o.Value]; !isFRValue {
			t.Errorf("EN: Value %q n'est pas une VALUE FR canonique (la Value NE doit PAS être localisée)", o.Value)
		}
		if o.Label != wantEN[o.Value] {
			t.Errorf("EN: Value %q → Label %q, want %q", o.Value, o.Label, wantEN[o.Value])
		}
	}

	// Locale FR (défaut) : Label == Value (FR) dans les deux champs.
	respFR, err := svc.Resolve(ctxkeys.WithLocale(context.Background(), "fr"), domain.FilterContextInput{FilterMode: "period"})
	if err != nil {
		t.Fatalf("resolve FR: %v", err)
	}
	for _, o := range respFR.AvailableOptions.ExperienceTypes {
		if o.Label != o.Value {
			t.Errorf("FR: Label %q != Value %q (attendu identique sous locale FR)", o.Label, o.Value)
		}
	}

	// Chemin emptyResolved (0 row) : localisé aussi (3 options, count 0).
	emptySvc := NewFiltersService(&mockFiltersRepo{rows: []domain.FilterMatchRow{}})
	respEmpty, err := emptySvc.Resolve(ctxkeys.WithLocale(context.Background(), "en"), domain.FilterContextInput{FilterMode: "period"})
	if err != nil {
		t.Fatalf("resolve empty EN: %v", err)
	}
	for _, o := range respEmpty.AvailableOptions.ExperienceTypes {
		if o.Label != wantEN[o.Value] {
			t.Errorf("empty EN: Value %q → Label %q, want %q", o.Value, o.Label, wantEN[o.Value])
		}
	}
}

func TestFiltersService_Resolve_Empty(t *testing.T) {
	repo := &mockFiltersRepo{rows: []domain.FilterMatchRow{}}
	svc := NewFiltersService(repo)

	resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Counts.TotalMatchesBeforeFilters != 0 {
		t.Errorf("TotalMatchesBeforeFilters = %d, want 0", resp.Counts.TotalMatchesBeforeFilters)
	}
}

func TestFilteredMatchIDs_OrderedDescAndContextFiltered(t *testing.T) {
	t1 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", StartTime: &t1, IsWithFriends: false},
		{MatchID: "m3", StartTime: &t3, IsWithFriends: false},
		{MatchID: "m2", StartTime: &t2, IsWithFriends: true}, // squad
	}

	// Contexte solo : exclut m2, ordonné start_time DESC.
	ids := FilteredMatchIDs(rows, domain.FilterContextInput{FilterMode: "period", MatchContext: domain.MatchContextSolo})
	want := []string{"m3", "m1"}
	if len(ids) != len(want) {
		t.Fatalf("FilteredMatchIDs solo = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("FilteredMatchIDs solo[%d] = %q, want %q (full: %v)", i, ids[i], want[i], ids)
		}
	}

	// Contexte squad : ne garde que m2.
	squad := FilteredMatchIDs(rows, domain.FilterContextInput{FilterMode: "period", MatchContext: domain.MatchContextSquad})
	if len(squad) != 1 || squad[0] != "m2" {
		t.Errorf("FilteredMatchIDs squad = %v, want [m2]", squad)
	}

	// Ensemble vide → nil (bouton masqué côté front).
	if got := FilteredMatchIDs(nil, domain.FilterContextInput{}); got != nil {
		t.Errorf("FilteredMatchIDs(nil) = %v, want nil", got)
	}
}

func TestFiltersService_Resolve_Error(t *testing.T) {
	repo := &mockFiltersRepo{err: errors.New("fail")}
	svc := NewFiltersService(repo)

	_, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
	if err == nil {
		t.Error("expected error")
	}
}

// TestFiltersService_Resolve_Empty_JSONShape vérifie la régression du crash
// front 2026-05-27 sur FilterOmnibar : quand totalBefore=0, emptyResolved()
// renvoyait Playlists/Modes/Maps en nil → JSON "playlists":null → opts.filter()
// crashait. Aujourd'hui les 4 slices sont initialisées à []. Ce test catche
// toute régression future via testutil.RequireNoNilSlicesWithoutOmitempty.
func TestFiltersService_Resolve_Empty_JSONShape(t *testing.T) {
	repo := &mockFiltersRepo{rows: []domain.FilterMatchRow{}}
	svc := NewFiltersService(repo)

	resp, err := svc.Resolve(context.Background(), domain.FilterContextInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1) Garde générique via reflection : aucun slice non-omitempty ne doit être nil.
	testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)

	// 2) Garde ciblée : le JSON sérialisé ne doit contenir aucun "<champ>":null
	//    pour les 4 champs critiques d'AvailableOptions consommés par FilterOmnibar.
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	s := string(b)
	for _, k := range []string{"playlists", "modes", "maps", "experience_types"} {
		needle := `"` + k + `":null`
		if strings.Contains(s, needle) {
			t.Errorf("JSON contains %q — front crashera sur .filter()/.map() (cf. FilterOmnibar.tsx)", needle)
		}
	}
}
