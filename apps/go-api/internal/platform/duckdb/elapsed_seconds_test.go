// elapsed_seconds_test.go — estimateur de durée de jeu observée (:memory:).
// Couvre : médiane des participants complets, repli MAX quand aucun ne l'est,
// insensibilité aux quitteurs, table absente (dégradation best-effort).
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newElapsedTestDB crée une DB :memory: avec un match_participants minimal.
func newElapsedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE match_participants (
			match_id              VARCHAR NOT NULL,
			xuid                  VARCHAR NOT NULL,
			time_played_seconds   INTEGER,
			present_at_beginning  BOOLEAN,
			present_at_completion BOOLEAN
		)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func seedParticipant(t *testing.T, db *sql.DB, matchID, xuid string, played int, begin, complete any) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO match_participants
		   (match_id, xuid, time_played_seconds, present_at_beginning, present_at_completion)
		 VALUES (?, ?, ?, ?, ?)`,
		matchID, xuid, played, begin, complete); err != nil {
		t.Fatalf("seed %s/%s: %v", matchID, xuid, err)
	}
}

// Médiane des participants complets ; les quitteurs (present_at_completion=false)
// ne doivent PAS tirer l'estimation vers le bas.
func TestLoadElapsedSecondsByMatch_MedianOfFullGamePlayers(t *testing.T) {
	db := newElapsedTestDB(t)
	// 3 joueurs complets : 760 / 763 / 766 → médiane 763.
	seedParticipant(t, db, "m1", "a", 760, true, true)
	seedParticipant(t, db, "m1", "b", 763, true, true)
	seedParticipant(t, db, "m1", "c", 766, true, true)
	// 2 quitteurs très courts + 1 retardataire : ignorés par le FILTER.
	seedParticipant(t, db, "m1", "d", 120, true, false)
	seedParticipant(t, db, "m1", "e", 90, true, false)
	seedParticipant(t, db, "m1", "f", 300, false, true)

	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m1"})
	if got["m1"] != 763 {
		t.Errorf("elapsed m1 = %d, want 763 (médiane des joueurs complets)", got["m1"])
	}
}

// Aucun participant complet (colonnes de participation jamais backfillées) →
// repli MAX(time_played_seconds).
func TestLoadElapsedSecondsByMatch_FallbackToMax(t *testing.T) {
	db := newElapsedTestDB(t)
	seedParticipant(t, db, "m2", "a", 640, nil, nil)
	seedParticipant(t, db, "m2", "b", 712, nil, nil)
	seedParticipant(t, db, "m2", "c", 500, nil, nil)
	// Participation explicitement partielle : ne qualifie pas non plus.
	seedParticipant(t, db, "m2", "d", 300, true, false)

	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m2"})
	if got["m2"] != 712 {
		t.Errorf("elapsed m2 = %d, want 712 (repli MAX)", got["m2"])
	}
}

// Médiane paire = moyenne des 2 valeurs centrales, arrondie.
func TestLoadElapsedSecondsByMatch_EvenMedianRounded(t *testing.T) {
	db := newElapsedTestDB(t)
	seedParticipant(t, db, "m3", "a", 720, true, true)
	seedParticipant(t, db, "m3", "b", 725, true, true) // médiane = 722.5 → 723 (arrondi)
	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m3"})
	if got["m3"] != 723 {
		t.Errorf("elapsed m3 = %d, want 723", got["m3"])
	}
}

// Plusieurs matchs en un appel + match inconnu absent de la map.
func TestLoadElapsedSecondsByMatch_MultipleAndMissing(t *testing.T) {
	db := newElapsedTestDB(t)
	seedParticipant(t, db, "m4", "a", 700, true, true)
	seedParticipant(t, db, "m5", "a", 900, true, true)

	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m4", "m5", "absent"})
	if got["m4"] != 700 || got["m5"] != 900 {
		t.Errorf("elapsed m4=%d m5=%d, want 700/900", got["m4"], got["m5"])
	}
	if _, ok := got["absent"]; ok {
		t.Error("un match sans participant ne doit pas apparaître dans la map")
	}
}

// Temps joué NULL / nul → pas d'estimation (pas d'entrée), jamais de 0 exposé.
func TestLoadElapsedSecondsByMatch_NoUsableTime(t *testing.T) {
	db := newElapsedTestDB(t)
	if _, err := db.Exec(
		`INSERT INTO match_participants (match_id, xuid, time_played_seconds) VALUES ('m6', 'a', NULL)`); err != nil {
		t.Fatalf("seed null: %v", err)
	}
	seedParticipant(t, db, "m7", "a", 0, true, true)

	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m6", "m7"})
	if _, ok := got["m6"]; ok {
		t.Error("time_played NULL ne doit pas produire d'estimation")
	}
	if _, ok := got["m7"]; ok {
		t.Error("time_played = 0 ne doit pas produire d'estimation")
	}
}

// Cas mono-match (Match View).
func TestLoadElapsedSecondsForMatch(t *testing.T) {
	db := newElapsedTestDB(t)
	seedParticipant(t, db, "m8", "a", 763, true, true)

	if secs, ok := LoadElapsedSecondsForMatch(context.Background(), db, "m8"); !ok || secs != 763 {
		t.Errorf("LoadElapsedSecondsForMatch(m8) = (%d, %v), want (763, true)", secs, ok)
	}
	if secs, ok := LoadElapsedSecondsForMatch(context.Background(), db, "inconnu"); ok || secs != 0 {
		t.Errorf("match inconnu = (%d, %v), want (0, false)", secs, ok)
	}
}

// Table absente (DB non migrée) → dégradation best-effort : map vide, pas d'erreur.
func TestLoadElapsedSecondsByMatch_DegradesWhenTableMissing(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	got := LoadElapsedSecondsByMatch(context.Background(), db, []string{"m1"})
	if len(got) != 0 {
		t.Errorf("table absente : got %d entrées, want 0 (dégradation)", len(got))
	}
}

// Entrées vides : aucun appel SQL, map vide.
func TestLoadElapsedSecondsByMatch_EmptyInput(t *testing.T) {
	if got := LoadElapsedSecondsByMatch(context.Background(), nil, []string{"m1"}); len(got) != 0 {
		t.Errorf("db nil : got %d entrées, want 0", len(got))
	}
	db := newElapsedTestDB(t)
	if got := LoadElapsedSecondsByMatch(context.Background(), db, nil); len(got) != 0 {
		t.Errorf("matchIDs vide : got %d entrées, want 0", len(got))
	}
}
