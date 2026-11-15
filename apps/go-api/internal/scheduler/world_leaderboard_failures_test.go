//go:build cgo
// +build cgo

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
)

// Une panne 5xx du service Halo sur les deux playlists d'une passe écrit UNE ligne WARN de synthèse,
// aucune ERROR par playlist.
func TestWorldLeaderboardCron_Panne5xxUneLigneDeSynthese(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{
		season:   "csrseason13-2",
		fetchErr: fmt.Errorf("FetchCSRLeaderboard page 1: %w", errors.New("statut HTTP 500")),
	}
	c := newTestCron(provider, scraper)
	logs := captureLogs(t)

	c.RunOnce(context.Background())

	if scraper.fetchCalls != 2 {
		t.Fatalf("scrapes = %d, attendu 2 (une par playlist)", scraper.fetchCalls)
	}
	if n := logs.countAtLevel(slog.LevelError, "scrape"); n != 0 {
		t.Errorf("%d ERROR de scrape, attendu 0 pour une panne 5xx", n)
	}
	if n := logs.countAtLevel(slog.LevelWarn, "scrape de playlists échoué"); n != 1 {
		t.Errorf("%d lignes de synthèse WARN, attendu 1", n)
	}
}

// Un échec qui n'est pas une panne du service distant (page illisible) reste une ERROR, une seule
// par passe.
func TestWorldLeaderboardCron_EchecHorsPanneResteUneErreur(t *testing.T) {
	provider, _ := newSharedProviderForTest(t)
	scraper := &stubScraper{season: "csrseason13-2", fetchErr: errors.New("markup illisible")}
	c := newTestCron(provider, scraper)
	logs := captureLogs(t)

	c.RunOnce(context.Background())

	if n := logs.countAtLevel(slog.LevelError, "scrape de playlists échoué"); n != 1 {
		t.Errorf("%d lignes de synthèse ERROR, attendu 1", n)
	}
}
