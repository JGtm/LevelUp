package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

func sessionRowsForTest() []legacymatch.StatsMatchRow {
	now := time.Now()
	labelA := "S-A"
	return []legacymatch.StatsMatchRow{
		{MatchID: "m1", StartTime: now, SessionLabel: &labelA},
		{MatchID: "m2", StartTime: now.Add(time.Minute), SessionLabel: &labelA},
	}
}

// ADR 0024 Couche B : une session demandée explicitement mais introuvable dans
// le périmètre renvoie session_not_found, au lieu d'une page vide 200 trompeuse.
func TestSessionPage_RequestedSessionNotFound(t *testing.T) {
	statsRepo := &mockSessionPageStatsRepo{matches: []legacymatch.StatsMatchRow{}}
	svc := NewSessionPageService(statsRepo).
		WithPlayerMatchesRepo(newStatsMockFromRows(sessionRowsForTest(), nil), "halo_infinite", "Test")

	missing := "S-MISSING"
	_, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &missing})
	if err == nil {
		t.Fatal("attendu session_not_found, obtenu nil")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "session_not_found" {
		t.Fatalf("attendu APIError session_not_found, obtenu %v", err)
	}
}

// Une session existante ne doit jamais déclencher session_not_found.
func TestSessionPage_RequestedSessionExists_NoNotFound(t *testing.T) {
	statsRepo := &mockSessionPageStatsRepo{matches: []legacymatch.StatsMatchRow{}}
	svc := NewSessionPageService(statsRepo).
		WithPlayerMatchesRepo(newStatsMockFromRows(sessionRowsForTest(), nil), "halo_infinite", "Test")

	existing := "S-A"
	_, err := svc.GetPage(context.Background(), domain.SessionPageRequest{SessionLabel: &existing})
	var apiErr *domain.APIError
	if errors.As(err, &apiErr) && apiErr.Code == "session_not_found" {
		t.Fatalf("session existante ne doit pas renvoyer session_not_found, obtenu %v", err)
	}
}
