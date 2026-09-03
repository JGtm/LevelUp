//go:build cgo
// +build cgo

// world_leaderboard_quality_test.go — garde-fou D1 : un lot de scrape dégradé ne doit
// jamais remplacer le lot servi. Rejoue l'incident du 2026-07-07 (86 lignes sans xuid
// ayant masqué 200 lignes intégralement identifiées) sur le harnais du cron, avec une
// vraie shared DuckDB migrée : c'est la vue _latest elle-même qui doit continuer à
// servir le lot sain après le cycle.
//
// Package interne (scheduler) : réutilise newTestCron / newSharedProviderForTest /
// stubScraper de world_leaderboard_cron_test.go.
package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

const qualityTestSeason = "csrseason13-3"

// servedT0 / candidateT1 : le lot candidat doit être PLUS RÉCENT que le lot servi,
// sinon la vue _latest (max(fetched_at)) continuerait de servir l'ancien lot et les
// tests d'acceptation ne prouveraient rien.
var (
	servedT0    = time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)
	candidateT1 = servedT0.Add(24 * time.Hour)
)

// batchEntries fabrique un lot de n entrées dont `withXUID` portent un xuid.
func batchEntries(n, withXUID int, at time.Time, tag string) []domain.LeaderboardEntry {
	out := make([]domain.LeaderboardEntry, 0, n)
	for i := 0; i < n; i++ {
		e := domain.LeaderboardEntry{
			Season: qualityTestSeason, Rank: i + 1,
			Gamertag: fmt.Sprintf("%s%d", tag, i+1), CSRValue: 2000 - i, FetchedAt: at,
		}
		if i < withXUID {
			e.XUID = fmt.Sprintf("253500000000%04d", i+1)
		}
		out = append(out, e)
	}
	return out
}

// seedServedBatch persiste un lot SERVI pour une playlist (état d'avant-cycle).
func seedServedBatch(t *testing.T, db *sql.DB, playlist string, rows, withXUID int) {
	t.Helper()
	entries := batchEntries(rows, withXUID, servedT0, "Sain")
	for i := range entries {
		entries[i].Playlist = playlist
	}
	if _, err := duckdb.InsertWorldCSRSnapshot(context.Background(), db, "halo_infinite", entries); err != nil {
		t.Fatalf("seed lot servi (%s): %v", playlist, err)
	}
}

// servedBatchOf lit ce que la vue _latest sert RÉELLEMENT pour une playlist.
func servedBatchOf(t *testing.T, db *sql.DB, playlist string) (rows, withXUID int) {
	t.Helper()
	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(xuid, '') <> '' THEN 1 ELSE 0 END), 0)
		FROM world_csr_leaderboard_latest
		WHERE title_slug = 'halo_infinite' AND season_id = ? AND playlist_id = ?`,
		qualityTestSeason, playlist).Scan(&rows, &withXUID)
	if err != nil {
		t.Fatalf("lecture lot servi (%s): %v", playlist, err)
	}
	return rows, withXUID
}

// rawRowsOf compte les lignes BRUTES (append-only) d'une playlist : prouve qu'un
// refus n'écrit rien du tout, pas seulement qu'il n'est pas servi.
func rawRowsOf(t *testing.T, db *sql.DB, playlist string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM world_csr_leaderboard_snapshots WHERE season_id = ? AND playlist_id = ?`,
		qualityTestSeason, playlist).Scan(&n); err != nil {
		t.Fatalf("count brut (%s): %v", playlist, err)
	}
	return n
}

// newQualityCron : cron sur une seule playlist, fraîcheur neutralisée (un lot servi
// pré-existant rendrait sinon le cycle « déjà frais » et rien ne serait scrapé).
func newQualityCron(t *testing.T, provider sharedprovider.Provider, scraper LeaderboardScraperPort, playlists ...string) *WorldLeaderboardCron {
	t.Helper()
	c := newTestCron(provider, scraper)
	pls := append([]string(nil), playlists...)
	c.playlists = func() []string { return pls }
	c.freshness = time.Nanosecond
	return c
}

// TestWorldLeaderboardCron_RefusesCollapsedVolume (D1, volume) : un candidat sous la
// moitié des lignes servies est refusé — rien n'est écrit et la vue continue de servir
// le lot sain.
func TestWorldLeaderboardCron_RefusesCollapsedVolume(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	seedServedBatch(t, db, "pl-a", 10, 10)

	scraper := &stubScraper{
		season:  qualityTestSeason,
		entries: batchEntries(4, 4, candidateT1, "Tronque"), // 4/10 = 40 % < 50 %
	}
	c := newQualityCron(t, provider, scraper, "pl-a")
	before := observability.LoadCounter(worldBatchRefusedMetric)

	c.RunOnce(context.Background())

	if got := rawRowsOf(t, db, "pl-a"); got != 10 {
		t.Errorf("lignes brutes = %d, attendu 10 (le lot refusé ne doit RIEN écrire)", got)
	}
	rows, xuid := servedBatchOf(t, db, "pl-a")
	if rows != 10 || xuid != 10 {
		t.Errorf("lot servi = (%d lignes, %d xuid), attendu (10, 10) — le lot sain doit rester servi", rows, xuid)
	}
	if got := observability.LoadCounter(worldBatchRefusedMetric); got != before+1 {
		t.Errorf("compteur de refus = %d, attendu %d", got, before+1)
	}
}

// TestWorldLeaderboardCron_RefusesXUIDCollapse (D1, identification) : à volume égal,
// un candidat sans AUCUN xuid face à un lot servi massivement identifié est refusé.
// C'est exactement l'incident du 2026-07-07 (0 xuid écrasant 100 % de xuid).
func TestWorldLeaderboardCron_RefusesXUIDCollapse(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	seedServedBatch(t, db, "pl-a", 10, 10) // couverture servie = 100 % (>= 90 %)

	scraper := &stubScraper{
		season:  qualityTestSeason,
		entries: batchEntries(10, 0, candidateT1, "SansXuid"), // même volume, 0 xuid
	}
	c := newQualityCron(t, provider, scraper, "pl-a")

	c.RunOnce(context.Background())

	if got := rawRowsOf(t, db, "pl-a"); got != 10 {
		t.Errorf("lignes brutes = %d, attendu 10 (candidat sans xuid refusé)", got)
	}
	rows, xuid := servedBatchOf(t, db, "pl-a")
	if rows != 10 || xuid != 10 {
		t.Errorf("lot servi = (%d lignes, %d xuid), attendu (10, 10)", rows, xuid)
	}
}

// TestWorldLeaderboardCron_AcceptsHealthyBatches : le garde-fou ne doit RIEN bloquer
// de légitime — ni une première capture (aucun lot servi), ni une croissance normale.
func TestWorldLeaderboardCron_AcceptsHealthyBatches(t *testing.T) {
	t.Run("première capture", func(t *testing.T) {
		provider, db := newSharedProviderForTest(t)
		scraper := &stubScraper{
			season:  qualityTestSeason,
			entries: batchEntries(10, 10, candidateT1, "Neuf"),
		}
		c := newQualityCron(t, provider, scraper, "pl-a")

		c.RunOnce(context.Background())

		rows, xuid := servedBatchOf(t, db, "pl-a")
		if rows != 10 || xuid != 10 {
			t.Errorf("lot servi = (%d, %d), attendu (10, 10) — aucun lot servi ne peut refuser le premier", rows, xuid)
		}
	})

	t.Run("croissance normale", func(t *testing.T) {
		provider, db := newSharedProviderForTest(t)
		seedServedBatch(t, db, "pl-a", 10, 10)
		scraper := &stubScraper{
			season:  qualityTestSeason,
			entries: batchEntries(12, 12, candidateT1, "Frais"),
		}
		c := newQualityCron(t, provider, scraper, "pl-a")

		c.RunOnce(context.Background())

		rows, xuid := servedBatchOf(t, db, "pl-a")
		if rows != 12 || xuid != 12 {
			t.Errorf("lot servi = (%d, %d), attendu (12, 12) — un lot sain doit remplacer le précédent", rows, xuid)
		}
	})
}

// TestWorldLeaderboardCron_PartialRefusalKeepsHealthyPlaylist (D1, refus partiel) :
// une playlist dégradée est écartée SANS empêcher la persistance des autres — le
// cycle va au bout et le classement sain de la playlist refusée reste servi.
func TestWorldLeaderboardCron_PartialRefusalKeepsHealthyPlaylist(t *testing.T) {
	provider, db := newSharedProviderForTest(t)
	seedServedBatch(t, db, "pl-degradee", 10, 10)
	seedServedBatch(t, db, "pl-saine", 10, 10)

	scraper := &stubScraper{
		season: qualityTestSeason,
		entriesByPlaylist: map[string][]domain.LeaderboardEntry{
			"pl-degradee": batchEntries(3, 3, candidateT1, "Tronque"), // 30 % → refusé
			"pl-saine":    batchEntries(12, 12, candidateT1, "Frais"), // sain → accepté
		},
	}
	c := newQualityCron(t, provider, scraper, "pl-degradee", "pl-saine")
	rec := captureLogs(t)

	c.RunOnce(context.Background())

	// Playlist refusée : aucune écriture, lot sain toujours servi.
	if got := rawRowsOf(t, db, "pl-degradee"); got != 10 {
		t.Errorf("pl-degradee : %d lignes brutes, attendu 10 (rien inséré)", got)
	}
	if rows, xuid := servedBatchOf(t, db, "pl-degradee"); rows != 10 || xuid != 10 {
		t.Errorf("pl-degradee servie = (%d, %d), attendu (10, 10)", rows, xuid)
	}
	// Playlist saine : le cycle est allé au bout et a persisté son lot.
	if rows, xuid := servedBatchOf(t, db, "pl-saine"); rows != 12 || xuid != 12 {
		t.Errorf("pl-saine servie = (%d, %d), attendu (12, 12) — un refus partiel ne doit pas avorter le cycle", rows, xuid)
	}
	// Un seul refus logué (WARN), pas une erreur de cycle.
	if got := rec.countAtLevel(slog.LevelWarn, "lot dégradé REFUSÉ"); got != 1 {
		t.Errorf("WARN de refus = %d, attendu 1", got)
	}
	if got := rec.countAtLevel(slog.LevelError, ""); got != 0 {
		t.Errorf("ERROR = %d, attendu 0 (un refus est un skip, jamais une erreur de cycle)", got)
	}
}

// TestDegradedBatchReason couvre les BORNES de la décision D1 sur la fonction pure :
// exactement 50 % accepté, juste en dessous refusé ; seuil de couverture xuid à 90 %.
func TestDegradedBatchReason(t *testing.T) {
	cases := []struct {
		name           string
		served, candid duckdb.WorldCSRBatchStats
		wantRefused    bool
	}{
		{"volume exactement 50 %", stats(100, 100), stats(50, 50), false},
		{"volume juste sous 50 %", stats(100, 100), stats(49, 49), true},
		{"volume effondré (incident 07/07)", stats(200, 200), stats(86, 0), true},
		{"croissance", stats(100, 100), stats(120, 120), false},
		{"xuid effondré, servi à 100 %", stats(100, 100), stats(100, 0), true},
		{"xuid effondré, servi à 90 %", stats(100, 90), stats(100, 0), true},
		{"xuid effondré mais servi à 89 % (sous le seuil)", stats(100, 89), stats(100, 0), false},
		{"xuid partiel (pas zéro) toléré", stats(100, 100), stats(100, 5), false},
		{"servi sans xuid, candidat sans xuid", stats(100, 0), stats(100, 0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason := duckdb.DegradedBatchReason(tc.served, tc.candid)
			if refused := reason != ""; refused != tc.wantRefused {
				t.Errorf("servi=%+v candidat=%+v → refusé=%v (%q), attendu refusé=%v",
					tc.served, tc.candid, refused, reason, tc.wantRefused)
			}
		})
	}
}

func stats(rows, withXUID int) duckdb.WorldCSRBatchStats {
	return duckdb.WorldCSRBatchStats{Rows: rows, WithXUID: withXUID}
}
