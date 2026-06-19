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
	sr    *H5ServiceRecordResponse
	srErr error
}

func (f *fakeSource) GetServiceRecords(_ context.Context, _, _ string) (*H5ServiceRecordResponse, error) {
	return f.sr, f.srErr
}

func (f *fakeSource) GetPlayerMatches(_ context.Context, _ string, _, _ int) (*H5MatchesResponse, error) {
	return nil, errors.New("non utilise")
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
	// Phase 1a honnete : seul career.progression est cable.
	if !caps.Has(games.CapCareerProgression) {
		t.Errorf("career.progression devrait etre disponible (LoadCareerSnapshot cable)")
	}
	// match.history / citations = not_exposed tant que stub (pas de Has()==true menteur).
	if caps.Has(games.CapMatchHistory) {
		t.Errorf("match.history devrait etre not_exposed (LoadMatchSummaries stub)")
	}
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
