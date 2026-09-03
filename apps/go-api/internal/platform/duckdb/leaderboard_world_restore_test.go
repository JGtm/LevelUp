//go:build integration

// leaderboard_world_restore_test.go — sélection du MEILLEUR lot historique et cycle
// de restauration (CLI -restore-best). Rejoue la situation réelle : un lot dégradé a
// été capturé après un lot sain, la vue _latest sert donc le dégradé ; la
// restauration doit ré-INSÉRER le lot sain (append-only, jamais de DELETE/UPDATE) et
// devenir un no-op au second passage.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// restoreTestBatch fabrique un lot de n entrées dont `withXUID` portent un xuid.
func restoreTestBatch(key WorldCSRBatchKey, n, withXUID int, at time.Time, tag string) []domain.LeaderboardEntry {
	out := make([]domain.LeaderboardEntry, 0, n)
	for i := 0; i < n; i++ {
		e := domain.LeaderboardEntry{
			Season: key.SeasonID, Playlist: key.PlaylistID, Rank: i + 1,
			Gamertag: fmt.Sprintf("%s%d", tag, i+1), CSRValue: 2000 - i,
			Tier: "Onyx", FetchedAt: at,
		}
		if i < withXUID {
			e.XUID = fmt.Sprintf("253500000000%04d", i+1)
		}
		out = append(out, e)
	}
	return out
}

func servedStatsOrFail(t *testing.T, db *sql.DB, key WorldCSRBatchKey) WorldCSRBatchStats {
	t.Helper()
	stats, ok, err := WorldCSRServedBatchStats(context.Background(), db, key.TitleSlug, key.SeasonID, key.PlaylistID)
	if err != nil || !ok {
		t.Fatalf("lecture lot servi: stats=%+v ok=%v err=%v", stats, ok, err)
	}
	return stats
}

// TestWorldCSRBestBatch_RestoreCycle couvre le cycle complet du -restore-best :
// détection du meilleur lot, verdict de la règle D1, ré-insertion, puis idempotence.
func TestWorldCSRBestBatch_RestoreCycle(t *testing.T) {
	db := openMemDB(t).SQLDb()
	applyWorldLeaderboardMigration(t, db)
	ctx := context.Background()
	key := WorldCSRBatchKey{TitleSlug: "halo_infinite", SeasonID: "csrseason13-2", PlaylistID: "pl-arena"}

	tSain := time.Date(2026, 7, 3, 4, 0, 0, 0, time.UTC)
	tDegrade := time.Date(2026, 7, 7, 4, 0, 0, 0, time.UTC)

	// Le lot SAIN (ancien) puis le lot DÉGRADÉ (récent) : la vue sert le dégradé.
	if _, err := InsertWorldCSRSnapshot(ctx, db, key.TitleSlug, restoreTestBatch(key, 10, 10, tSain, "Sain")); err != nil {
		t.Fatalf("insert lot sain: %v", err)
	}
	if _, err := InsertWorldCSRSnapshot(ctx, db, key.TitleSlug, restoreTestBatch(key, 3, 0, tDegrade, "Degrade")); err != nil {
		t.Fatalf("insert lot dégradé: %v", err)
	}
	if served := servedStatsOrFail(t, db, key); served.Rows != 3 || served.WithXUID != 0 {
		t.Fatalf("lot servi initial = %+v, attendu {Rows:3 WithXUID:0} (le dégradé masque le sain)", served)
	}

	// Le couple doit être découvert par le balayage du CLI.
	pairs, err := WorldCSRSeasonPlaylistPairs(ctx, db, key.TitleSlug)
	if err != nil {
		t.Fatalf("WorldCSRSeasonPlaylistPairs: %v", err)
	}
	if len(pairs) != 1 || pairs[0] != key {
		t.Fatalf("couples = %+v, attendu [%+v]", pairs, key)
	}

	// Meilleur lot = le sain (plus d'xuid, plus de lignes), malgré son antériorité.
	best, ok, err := WorldCSRBestBatch(ctx, db, key)
	if err != nil || !ok {
		t.Fatalf("WorldCSRBestBatch: best=%+v ok=%v err=%v", best, ok, err)
	}
	if best.Stats.Rows != 10 || best.Stats.WithXUID != 10 {
		t.Errorf("meilleur lot = %+v, attendu {Rows:10 WithXUID:10}", best.Stats)
	}
	if !best.FetchedAt.Equal(tSain) {
		t.Errorf("meilleur lot daté %v, attendu %v (le sain, pas le plus récent)", best.FetchedAt, tSain)
	}

	// Verdict D1 : le lot SERVI serait refusé face au meilleur → restauration due.
	if reason := DegradedBatchReason(best.Stats, servedStatsOrFail(t, db, key)); reason == "" {
		t.Fatal("la règle ne réclame pas la restauration alors que le lot servi est effondré")
	}

	// Restauration : ré-INSERT du meilleur lot avec un instant frais commun.
	entries, err := WorldCSRBatchEntries(ctx, db, key, best.FetchedAt)
	if err != nil {
		t.Fatalf("WorldCSRBatchEntries: %v", err)
	}
	if len(entries) != 10 {
		t.Fatalf("entrées du meilleur lot = %d, attendu 10", len(entries))
	}
	tRestore := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	for i := range entries {
		entries[i].FetchedAt = tRestore
	}
	if _, err := InsertWorldCSRSnapshot(ctx, db, key.TitleSlug, entries); err != nil {
		t.Fatalf("ré-insert restauration: %v", err)
	}

	// La vue sert de nouveau le contenu SAIN.
	served := servedStatsOrFail(t, db, key)
	if served.Rows != 10 || served.WithXUID != 10 {
		t.Errorf("lot servi après restauration = %+v, attendu {Rows:10 WithXUID:10}", served)
	}
	var topGT, topXUID string
	if err := db.QueryRowContext(ctx,
		`SELECT gamertag, COALESCE(xuid, '') FROM world_csr_leaderboard_latest
		 WHERE title_slug = ? AND season_id = ? AND playlist_id = ? AND rank = 1`,
		key.TitleSlug, key.SeasonID, key.PlaylistID).Scan(&topGT, &topXUID); err != nil {
		t.Fatalf("lecture rang 1: %v", err)
	}
	if topGT != "Sain1" || topXUID == "" {
		t.Errorf("rang 1 servi = (%q, xuid=%q), attendu le contenu sain identifié", topGT, topXUID)
	}

	// Append-only : rien n'a été supprimé ni modifié (10 + 3 + 10).
	var raw int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM world_csr_leaderboard_snapshots`).Scan(&raw); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	if raw != 23 {
		t.Errorf("lignes brutes = %d, attendu 23 (append-only : aucun DELETE/UPDATE)", raw)
	}

	// Idempotence : au second passage, servi == meilleur → la règle ne réclame rien.
	best2, ok2, err := WorldCSRBestBatch(ctx, db, key)
	if err != nil || !ok2 {
		t.Fatalf("WorldCSRBestBatch (2e passage): %v", err)
	}
	if !best2.FetchedAt.Equal(tRestore) {
		t.Errorf("meilleur lot après restauration daté %v, attendu %v (le lot restauré)", best2.FetchedAt, tRestore)
	}
	if reason := DegradedBatchReason(best2.Stats, servedStatsOrFail(t, db, key)); reason != "" {
		t.Errorf("2e passage : la règle réclame encore une restauration (%q) — le -restore-best boucherait", reason)
	}
}
