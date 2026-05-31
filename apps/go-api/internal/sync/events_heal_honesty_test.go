//go:build cgo

// Package sync — events_heal_honesty_test.go : garde-fou anti-régression sur
// l'honnêteté des compteurs du self-heal events.
//
// Contexte (incident mai 2026, cf. .ai/HANDOFF_sync_combat_completion.md) :
// quand `shared_matches_v2` était ouverte en read-only par la connexion de
// complétion post-sync, `InsertHighlightEvents` échouait à CHAQUE cycle, mais
// l'échec était comptabilisé en `no_film` — donnant l'illusion d'un film
// simplement absent. Le bug est resté invisible 31h. Ce test verrouille la
// distinction : un échec RÉEL (réseau / write RO) ne doit JAMAIS être compté
// comme no_film.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newHealTestSharedDB crée une shared DB fichier minimale (match_registry avec
// 1 match events_loaded=FALSE) pour exercer healEventsForRecentMatches sans
// réseau. Fichier (pas :memory:) car le pool database/sql peut ouvrir
// plusieurs connexions et :memory: en donnerait une distincte par conn.
func newHealTestSharedDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared_heal_test.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			events_loaded BOOLEAN DEFAULT FALSE,
			backfill_completed BIGINT DEFAULT 0
		)`,
		`INSERT INTO match_registry (match_id, start_time, events_loaded)
			VALUES ('m-heal-1', TIMESTAMP '2026-05-30 13:00:00', FALSE)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(context.Background(), s); err != nil {
			t.Fatalf("exec schema: %v\nSQL: %s", err, s)
		}
	}
	return db
}

// TestHealEvents_FetchError_NotCountedAsNoFilm : un échec de fetch (proxy d'un
// échec réseau OU d'une écriture shared en read-only) ne doit pas gonfler
// no_film. Avant le fix, ce cas tombait dans `default: noFilm++`.
func TestHealEvents_FetchError_NotCountedAsNoFilm(t *testing.T) {
	db := newHealTestSharedDB(t)
	client := &mockHaloClient{getHighlightErr: errors.New("network boom")}

	healed, noFilm, err := healEventsForRecentMatches(context.Background(), db, nil, client, 10)
	if err != nil {
		t.Fatalf("healEvents: erreur propagée alors que best-effort attendu: %v", err)
	}
	if healed != 0 {
		t.Errorf("healed=%d, want 0", healed)
	}
	if noFilm != 0 {
		t.Errorf("noFilm=%d, want 0 — un échec réel ne doit JAMAIS être compté en no_film", noFilm)
	}
}

// TestHealEvents_GenuineNoFilm_CountedAsNoFilm : contrôle anti-sur-correction —
// un film réellement absent (found=false, sans erreur) reste compté en no_film
// et marque events_loaded=TRUE pour ne pas reboucler.
func TestHealEvents_GenuineNoFilm_CountedAsNoFilm(t *testing.T) {
	db := newHealTestSharedDB(t)
	client := &mockHaloClient{} // highlightChunkData nil + getHighlightErr nil → found=false, sans erreur

	healed, noFilm, err := healEventsForRecentMatches(context.Background(), db, nil, client, 10)
	if err != nil {
		t.Fatalf("healEvents: %v", err)
	}
	if healed != 0 {
		t.Errorf("healed=%d, want 0", healed)
	}
	if noFilm != 1 {
		t.Errorf("noFilm=%d, want 1 — film absent légitime doit compter en no_film", noFilm)
	}

	var loaded bool
	if err := db.QueryRow(
		`SELECT events_loaded FROM match_registry WHERE match_id = 'm-heal-1'`,
	).Scan(&loaded); err != nil {
		t.Fatalf("scan events_loaded: %v", err)
	}
	if !loaded {
		t.Error("events_loaded devrait être TRUE après un no-film définitif")
	}
}
