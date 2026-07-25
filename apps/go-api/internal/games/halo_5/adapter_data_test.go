package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// fakeSource implemente h5Source sans reseau (injecte des reponses / erreurs).
type fakeSource struct {
	sr         *H5ServiceRecordResponse
	srErr      error
	ev         *h5MatchEventsResponse
	evErr      error
	matches    *H5MatchesResponse
	matchesErr error
	carnage    *H5CarnageResponse
	carnageErr error
}

func (f *fakeSource) GetServiceRecords(_ context.Context, _, _ string) (*H5ServiceRecordResponse, error) {
	return f.sr, f.srErr
}

func (f *fakeSource) GetPlayerMatches(_ context.Context, _ string, _, _ int) (*H5MatchesResponse, error) {
	return f.matches, f.matchesErr
}

func (f *fakeSource) GetMatchCarnage(_ context.Context, _, _ string) (*H5CarnageResponse, error) {
	return f.carnage, f.carnageErr
}

func (f *fakeSource) GetMatchEvents(_ context.Context, _ string) (*h5MatchEventsResponse, error) {
	return f.ev, f.evErr
}

// srcFactory enveloppe une source fixe en SourceFactory (le token de prod vient du
// ctx ; en test on court-circuite).
func srcFactory(s h5Source) SourceFactory {
	return func(context.Context) (h5Source, error) { return s, nil }
}

func mustServiceRecord(t *testing.T) *H5ServiceRecordResponse {
	t.Helper()
	var sr H5ServiceRecordResponse
	if err := json.Unmarshal([]byte(fixtureServiceRecord), &sr); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return &sr
}

func TestAdapter_TitleSlugAndCapabilities(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil)
	if a.TitleSlug() != "halo_5" {
		t.Errorf("TitleSlug = %q, want halo_5", a.TitleSlug())
	}
	caps := a.Capabilities()
	// career.progression cable (LoadCareerSnapshot).
	if !caps.Has(games.CapCareerProgression) {
		t.Errorf("career.progression devrait etre disponible (LoadCareerSnapshot cable)")
	}
	// match.history = supported (AXE A : LoadMatchSummaries lit le shared h5 local).
	if !caps.Has(games.CapMatchHistory) {
		t.Errorf("match.history devrait etre supported (LoadMatchSummaries cable shared h5)")
	}
	// citations = not_exposed tant que stub (pas de Has()==true menteur).
	if caps.Has(games.CapCitationsEngine) {
		t.Errorf("citations.engine devrait etre not_exposed (decision B)")
	}
}

func TestAdapter_NilFactory_CapabilitiesDegraded(t *testing.T) {
	// Pas de source-factory -> rien n'est servable -> toutes les capabilities a not_exposed.
	caps := NewDataAdapter(nil, nil).Capabilities()
	for k, v := range caps {
		if v != games.CapNotExposed {
			t.Errorf("factory nil : capability %q = %q, want not_exposed", k, v)
		}
	}
}

func TestAdapter_LoadPlayerStats_Live(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{sr: mustServiceRecord(t)}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("LoadPlayerStats: %v", err)
	}
	if s.MatchesPlayed != 3 || s.Kills != 20 || s.Deaths != 39 {
		t.Errorf("stats inattendues : %+v", s)
	}
	if s.Identity.XUID != "" || s.Identity.Gamertag != "JGtm" {
		t.Errorf("identite gamertag-keyee attendue : %+v", s.Identity)
	}
}

func TestAdapter_LoadCareerSnapshot_Live(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{sr: mustServiceRecord(t)}), nil)
	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap.RankName == nil || *snap.RankName != "Diamant 5" {
		t.Errorf("RankName = %v, want Diamant 5", snap.RankName)
	}
}

func TestAdapter_NilFactory_NotSupported(t *testing.T) {
	a := NewDataAdapter(nil, nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("factory nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
	if _, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("factory nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
}

func TestAdapter_NotFound_EmptyNotError(t *testing.T) {
	notFound := &HTTPError{StatusCode: http.StatusNotFound, URL: "x", Err: errors.New("absent")}
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: notFound}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "Inconnu", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("404 ne doit pas etre une erreur metier : %v", err)
	}
	if s == nil || s.MatchesPlayed != 0 || s.Identity.Gamertag != "Inconnu" {
		t.Errorf("404 -> stats vides avec identite, got %+v", s)
	}
}

func TestAdapter_TokenExpired401_GracefulEmpty(t *testing.T) {
	// Un 401 (token expire/tourne) sur un endpoint read-only ne doit pas casser la
	// page : degradation gracieuse (vide identite-seule + warn), pas erreur dure.
	unauth := &HTTPError{StatusCode: http.StatusUnauthorized, URL: "x", Err: errors.New("expire")}
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: unauth}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("401 doit degrader gracieusement, pas erreur dure : %v", err)
	}
	if s == nil || s.MatchesPlayed != 0 {
		t.Errorf("401 -> stats vides, got %+v", s)
	}
}

func TestAdapter_TransportError_Propagated(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: errors.New("boom reseau")}), nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); err == nil {
		t.Error("erreur transport (non-404/401) doit etre propagee")
	}
}

// TestNewSpartanTokenSource_NoToken : la factory de prod echoue proprement si le
// contexte ne porte pas de SpartanToken (pas de panique).
func TestNewSpartanTokenSource_NoToken(t *testing.T) {
	if _, err := NewSpartanTokenSource(context.Background()); err == nil {
		t.Error("attendu une erreur quand le SpartanToken est absent du contexte")
	}
}

// fakeMatchHistorySource implémente MatchHistorySource sans DB (in-memory).
// Capture les matchIDs reçus pour vérifier le passage de paramètres.
type fakeMatchHistorySource struct {
	summaries []canonical.MatchSummary
	err       error
	gotIDs    []string
}

func (f *fakeMatchHistorySource) GetMatchSummaries(_ context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	f.gotIDs = matchIDs
	return f.summaries, f.err
}

// TestAdapter_LoadMatchSummaries_FromSource : AXE A — l'adapter délègue à la source
// d'historique LOCAL (shared h5) et renvoie les MatchSummary projetés tels quels.
func TestAdapter_LoadMatchSummaries_FromSource(t *testing.T) {
	win := canonical.OutcomeWin
	rk := true
	want := []canonical.MatchSummary{
		{MatchID: "m1", MatchType: canonical.MatchTypeRanked, Outcome: win, IsRanked: &rk},
		{MatchID: "m2", Outcome: canonical.OutcomeLoss},
		{MatchID: "m3", Outcome: canonical.OutcomeTie},
	}
	fake := &fakeMatchHistorySource{summaries: want}
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil).WithMatchHistorySource(fake)

	got, err := a.LoadMatchSummaries(context.Background(), []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("LoadMatchSummaries: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i := range want {
		if got[i].MatchID != want[i].MatchID {
			t.Errorf("ordre/ID #%d = %q, want %q", i, got[i].MatchID, want[i].MatchID)
		}
		if got[i].Outcome != want[i].Outcome {
			t.Errorf("Outcome #%d = %q, want %q", i, got[i].Outcome, want[i].Outcome)
		}
	}
	// Mapping canonique préservé (ranked + outcome win sur le 1er).
	if got[0].MatchType != canonical.MatchTypeRanked || got[0].IsRanked == nil || !*got[0].IsRanked {
		t.Errorf("MatchSummary canonique #0 KO: %+v", got[0])
	}
	// Les matchIDs sont propagés à la source telle quelle.
	if len(fake.gotIDs) != 3 {
		t.Errorf("matchIDs propagés = %v, want 3", fake.gotIDs)
	}
}

// TestAdapter_LoadMatchSummaries_NilSource : sans source d'historique câblée (cas du
// builder global live-only sans PlayerDB), LoadMatchSummaries dégrade proprement.
func TestAdapter_LoadMatchSummaries_NilSource(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil) // pas de WithMatchHistorySource
	got, err := a.LoadMatchSummaries(context.Background(), nil)
	if got != nil {
		t.Errorf("source nil → summaries nil, got %+v", got)
	}
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("source nil → attendu ErrCapabilityNotSupported, got %v", err)
	}
}

// TestAdapter_LoadMatchSummaries_SourceError : une erreur de la source (DB) est
// propagée (pas une dégradation silencieuse — c'est une panne de lecture locale).
func TestAdapter_LoadMatchSummaries_SourceError(t *testing.T) {
	fake := &fakeMatchHistorySource{err: errors.New("boom shared")}
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil).WithMatchHistorySource(fake)
	if _, err := a.LoadMatchSummaries(context.Background(), nil); err == nil {
		t.Error("erreur source doit être propagée")
	} else if errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("erreur source ≠ ErrCapabilityNotSupported, got %v", err)
	}
}
