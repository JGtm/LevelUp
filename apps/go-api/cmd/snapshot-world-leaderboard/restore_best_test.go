//go:build cgo

// restore_best_test.go — orchestration du mode -restore-best sur une vraie shared DB
// migrée. Ce que le test verrouille, parce que ce sont les promesses faites à
// l'opérateur avant une exécution en production :
//   - le DRY-RUN n'écrit RIEN (aucune ligne ajoutée, lot servi inchangé) ;
//   - l'exécution restaure le lot sain via un ré-INSERT (append-only : le lot dégradé
//     reste en table) et la vue _latest sert de nouveau le bon contenu ;
//   - un second passage est un NO-OP (« déjà au meilleur »), donc rejouable sans
//     empiler des copies à chaque lancement.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// openTestSharedDB ouvre une shared DB temporaire migrée (même chemin d'ouverture que
// le job réel : openSharedRW + RunForDB).
func openTestSharedDB(t *testing.T) *sql.DB {
	t.Helper()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	db, err := openSharedRW(filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migration shared: %v", err)
	}
	return db
}

func testBatch(key duckdb.WorldCSRBatchKey, n, withXUID int, at time.Time, tag string) []domain.LeaderboardEntry {
	out := make([]domain.LeaderboardEntry, 0, n)
	for i := 0; i < n; i++ {
		e := domain.LeaderboardEntry{
			Season: key.SeasonID, Playlist: key.PlaylistID, Rank: i + 1,
			Gamertag: tag, CSRValue: 2000 - i, Tier: "Onyx", FetchedAt: at,
		}
		if i < withXUID {
			e.XUID = "2535000000000001"
		}
		out = append(out, e)
	}
	return out
}

func rawCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM world_csr_leaderboard_snapshots`).Scan(&n); err != nil {
		t.Fatalf("count brut: %v", err)
	}
	return n
}

func servedOf(t *testing.T, db *sql.DB, key duckdb.WorldCSRBatchKey) duckdb.WorldCSRBatchStats {
	t.Helper()
	stats, _, err := duckdb.WorldCSRServedBatchStats(context.Background(), db, key.TitleSlug, key.SeasonID, key.PlaylistID)
	if err != nil {
		t.Fatalf("lecture lot servi: %v", err)
	}
	return stats
}

func TestRestoreBestBatches_DryRunThenExecuteThenNoop(t *testing.T) {
	db := openTestSharedDB(t)
	ctx := context.Background()
	log := slog.Default()
	key := duckdb.WorldCSRBatchKey{TitleSlug: "halo_infinite", SeasonID: "csrseason13-2", PlaylistID: "pl-arena"}

	// Lot sain (2026-07-03) puis lot dégradé (2026-07-07) : c'est le dégradé qui est servi.
	if _, err := duckdb.InsertWorldCSRSnapshot(ctx, db, key.TitleSlug,
		testBatch(key, 10, 10, time.Date(2026, 7, 3, 4, 0, 0, 0, time.UTC), "Sain")); err != nil {
		t.Fatalf("insert lot sain: %v", err)
	}
	if _, err := duckdb.InsertWorldCSRSnapshot(ctx, db, key.TitleSlug,
		testBatch(key, 3, 0, time.Date(2026, 7, 7, 4, 0, 0, 0, time.UTC), "Degrade")); err != nil {
		t.Fatalf("insert lot dégradé: %v", err)
	}
	if s := servedOf(t, db, key); s.Rows != 3 || s.WithXUID != 0 {
		t.Fatalf("lot servi initial = %+v, attendu {Rows:3 WithXUID:0}", s)
	}

	// 1. DRY-RUN : annonce la restauration, n'écrit rien.
	rawBefore := rawCount(t, db)
	restored, already, failed, err := restoreBestBatches(ctx, log, db, key.TitleSlug, false)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if restored != 1 || already != 0 || failed != 0 {
		t.Errorf("dry-run → restored=%d already=%d failed=%d, attendu 1/0/0", restored, already, failed)
	}
	if got := rawCount(t, db); got != rawBefore {
		t.Errorf("dry-run a écrit %d ligne(s) — il ne doit RIEN écrire", got-rawBefore)
	}
	if s := servedOf(t, db, key); s.Rows != 3 {
		t.Errorf("dry-run a changé le lot servi (%+v)", s)
	}

	// 2. EXÉCUTION : le lot sain redevient servi, l'historique est préservé.
	restored, already, failed, err = restoreBestBatches(ctx, log, db, key.TitleSlug, true)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if restored != 1 || already != 0 || failed != 0 {
		t.Errorf("execute → restored=%d already=%d failed=%d, attendu 1/0/0", restored, already, failed)
	}
	if s := servedOf(t, db, key); s.Rows != 10 || s.WithXUID != 10 {
		t.Errorf("lot servi après restauration = %+v, attendu {Rows:10 WithXUID:10}", s)
	}
	if got := rawCount(t, db); got != rawBefore+10 {
		t.Errorf("lignes brutes = %d, attendu %d (ré-INSERT append-only du lot sain)", got, rawBefore+10)
	}

	// 3. SECOND PASSAGE : rien à faire, aucune ligne supplémentaire.
	rawAfterRestore := rawCount(t, db)
	restored, already, failed, err = restoreBestBatches(ctx, log, db, key.TitleSlug, true)
	if err != nil {
		t.Fatalf("2e passage: %v", err)
	}
	if restored != 0 || already != 1 || failed != 0 {
		t.Errorf("2e passage → restored=%d already=%d failed=%d, attendu 0/1/0 (idempotent)", restored, already, failed)
	}
	if got := rawCount(t, db); got != rawAfterRestore {
		t.Errorf("2e passage a écrit %d ligne(s) — le mode doit être rejouable sans empiler", got-rawAfterRestore)
	}
}
