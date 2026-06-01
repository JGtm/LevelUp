//go:build cgo

// Package sync — film_retry_policy_test.go : verrouille la politique de marquage
// events_loaded sur film absent (404). Un film simplement RETARDÉ (match récent)
// ne doit pas être marqué définitif, sinon perte définitive des events de combat
// (cf. .ai/HANDOFF_sync_combat_completion.md).
package sync

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newFilmRetrySharedDB crée une shared DB fichier avec match_registry. Le match
// est inséré avec le start_time fourni (zéro time.Time → start_time NULL).
func newFilmRetrySharedDB(t *testing.T, matchID string, startTime time.Time) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "film_retry.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			events_loaded BOOLEAN DEFAULT FALSE,
			backfill_completed BIGINT DEFAULT 0
		)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if startTime.IsZero() {
		_, err = db.ExecContext(context.Background(),
			`INSERT INTO match_registry (match_id, events_loaded) VALUES (?, FALSE)`, matchID)
	} else {
		_, err = db.ExecContext(context.Background(),
			`INSERT INTO match_registry (match_id, start_time, events_loaded) VALUES (?, ?, FALSE)`,
			matchID, startTime)
	}
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	return db
}

func eventsLoaded(t *testing.T, db *sql.DB, matchID string) bool {
	t.Helper()
	var loaded bool
	if err := db.QueryRow(
		`SELECT events_loaded FROM match_registry WHERE match_id = ?`, matchID).Scan(&loaded); err != nil {
		t.Fatalf("scan events_loaded: %v", err)
	}
	return loaded
}

// TestNoFilm_RecentMatch_NotMarked : un 404 sur un match récent (< filmRetryWindow)
// ne doit PAS marquer events_loaded → le match reste dans le retry set.
func TestNoFilm_RecentMatch_NotMarked(t *testing.T) {
	const matchID = "m-recent"
	db := newFilmRetrySharedDB(t, matchID, time.Now().Add(-1*time.Hour))
	mock := &mockHaloClient{highlightChunkFound: false}

	if err := ProcessHighlightEvents(context.Background(), mock, db, nil, matchID, &domain.SyncResult{}); err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}
	if eventsLoaded(t, db, matchID) {
		t.Error("match récent + 404 : events_loaded ne doit PAS être TRUE (film peut-être pas encore propagé)")
	}
}

// TestNoFilm_OldMatch_Marked : un 404 sur un match ancien (> filmRetryWindow,
// défaut 30 jours) est définitif → events_loaded=TRUE pour sortir du retry set.
func TestNoFilm_OldMatch_Marked(t *testing.T) {
	const matchID = "m-old"
	// 40 jours : au-delà du défaut 30 jours (films conservés ~plusieurs mois ;
	// au-delà on suppose le film réellement expiré). Cf. defaultFilmRetryWindow.
	db := newFilmRetrySharedDB(t, matchID, time.Now().Add(-40*24*time.Hour))
	mock := &mockHaloClient{highlightChunkFound: false}

	if err := ProcessHighlightEvents(context.Background(), mock, db, nil, matchID, &domain.SyncResult{}); err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}
	if !eventsLoaded(t, db, matchID) {
		t.Error("match ancien + 404 : events_loaded devrait être TRUE (film définitivement absent)")
	}
}

// TestNoFilm_UnknownStartTime_Marked : start_time NULL → comportement legacy
// (définitif). Verrouille la compat avec golden_test / bitmask_honesty_test qui
// insèrent sans start_time.
func TestNoFilm_UnknownStartTime_Marked(t *testing.T) {
	const matchID = "m-null"
	db := newFilmRetrySharedDB(t, matchID, time.Time{})
	mock := &mockHaloClient{highlightChunkFound: false}

	if err := ProcessHighlightEvents(context.Background(), mock, db, nil, matchID, &domain.SyncResult{}); err != nil {
		t.Fatalf("ProcessHighlightEvents: %v", err)
	}
	if !eventsLoaded(t, db, matchID) {
		t.Error("start_time NULL + 404 : events_loaded devrait être TRUE (legacy/définitif)")
	}
}

// TestFilmRetryWindow_DefaultIsThirtyDays : le défaut couvre largement toute
// panne d'écriture combat réaliste (films Halo conservés plusieurs mois).
func TestFilmRetryWindow_DefaultIsThirtyDays(t *testing.T) {
	t.Setenv("LEVELUP_FILM_RETRY_WINDOW_HOURS", "")
	if got := filmRetryWindow(); got != 30*24*time.Hour {
		t.Errorf("filmRetryWindow défaut = %v, want 720h (30j)", got)
	}
}

// TestFilmRetryWindow_EnvOverride : override par LEVELUP_FILM_RETRY_WINDOW_HOURS
// (entier d'heures valide) ; valeur invalide → défaut.
func TestFilmRetryWindow_EnvOverride(t *testing.T) {
	t.Setenv("LEVELUP_FILM_RETRY_WINDOW_HOURS", "72")
	if got := filmRetryWindow(); got != 72*time.Hour {
		t.Errorf("override 72 = %v, want 72h", got)
	}
	t.Setenv("LEVELUP_FILM_RETRY_WINDOW_HOURS", "0") // <= 0 ignoré
	if got := filmRetryWindow(); got != 30*24*time.Hour {
		t.Errorf("override 0 → défaut attendu, got %v", got)
	}
	t.Setenv("LEVELUP_FILM_RETRY_WINDOW_HOURS", "abc") // invalide → défaut
	if got := filmRetryWindow(); got != 30*24*time.Hour {
		t.Errorf("override invalide → défaut attendu, got %v", got)
	}
}
