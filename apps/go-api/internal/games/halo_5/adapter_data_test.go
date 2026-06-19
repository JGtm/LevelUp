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

func mustServiceRecord(t *testing.T) *H5ServiceRecordResponse {
	t.Helper()
	var sr H5ServiceRecordResponse
	if err := json.Unmarshal([]byte(fixtureServiceRecord), &sr); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return &sr
}

func TestAdapter_TitleSlugAndCapabilities(t *testing.T) {
	a := NewDataAdapter(&fakeSource{}, nil)
	if a.TitleSlug() != "halo_5" {
		t.Errorf("TitleSlug = %q, want halo_5", a.TitleSlug())
	}
	caps := a.Capabilities()
	if !caps.Has(games.CapMatchHistory) {
		t.Errorf("match.history devrait etre disponible")
	}
	if caps.Has(games.CapCitationsEngine) {
		t.Errorf("citations.engine devrait etre not_exposed (decision B)")
	}
}

func TestAdapter_LoadPlayerStats_Live(t *testing.T) {
	a := NewDataAdapter(&fakeSource{sr: mustServiceRecord(t)}, nil)
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
	a := NewDataAdapter(&fakeSource{sr: mustServiceRecord(t)}, nil)
	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap.RankName == nil || *snap.RankName != "Diamant 5" {
		t.Errorf("RankName = %v, want Diamant 5", snap.RankName)
	}
}

func TestAdapter_NilSource_NotSupported(t *testing.T) {
	a := NewDataAdapter(nil, nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("source nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
	if _, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("source nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
}

func TestAdapter_NotFound_EmptyNotError(t *testing.T) {
	notFound := &HTTPError{StatusCode: http.StatusNotFound, URL: "x", Err: errors.New("absent")}
	a := NewDataAdapter(&fakeSource{srErr: notFound}, nil)
	s, err := a.LoadPlayerStats(context.Background(), "Inconnu", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("404 ne doit pas etre une erreur metier : %v", err)
	}
	if s == nil || s.MatchesPlayed != 0 || s.Identity.Gamertag != "Inconnu" {
		t.Errorf("404 -> stats vides avec identite, got %+v", s)
	}
}

func TestAdapter_TransportError_Propagated(t *testing.T) {
	a := NewDataAdapter(&fakeSource{srErr: errors.New("boom reseau")}, nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); err == nil {
		t.Error("erreur transport (non-404) doit etre propagee")
	}
}
