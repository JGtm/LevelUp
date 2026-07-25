//go:build cgo

package wire

// post_sync_deltas_snapshot_medals_test.go — contrat d'INVALIDITÉ du set
// EarnedMedalIDs (contre-revue V7.2, 2026-07-25).
//
// Enjeu : un set TRONQUÉ côté snapshot « before » fait passer des médailles déjà
// connues pour inédites → fausses notifications « médaille inédite ». Toute
// lecture qui ne peut pas garantir un set COMPLET doit donc rendre nil, ce que la
// garde cold-start de emitMedalFirstEarned traduit en seed silencieux.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// newMedalsEarnedDB ouvre une DuckDB mémoire avec medals_earned seedée.
func newMedalsEarnedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE medals_earned (
		match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT, count INTEGER)`); err != nil {
		t.Fatalf("create medals_earned: %v", err)
	}
	return db
}

// TestLoadEarnedMedalIDs_Aggregate : agrégat correct (SUM(count) > 0 par
// medal_name_id, scopé xuid) et exclusion des médailles à somme nulle.
func TestLoadEarnedMedalIDs_Aggregate(t *testing.T) {
	db := newMedalsEarnedDB(t)
	for _, q := range []string{
		`INSERT INTO medals_earned VALUES ('m1','x1',10,1)`,
		`INSERT INTO medals_earned VALUES ('m2','x1',10,2)`, // même médaille, 2 matchs
		`INSERT INTO medals_earned VALUES ('m1','x1',20,1)`,
		`INSERT INTO medals_earned VALUES ('m3','x1',30,0)`, // SUM=0 → exclue
		`INSERT INTO medals_earned VALUES ('m1','x2',99,5)`, // autre joueur → exclue
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	got := loadEarnedMedalIDs(context.Background(), db, "x1")
	if got == nil {
		t.Fatal("set nil alors que la lecture est saine (nil est réservé à l'INVALIDITÉ)")
	}
	if len(got) != 2 {
		t.Fatalf("got %d médailles, want 2 (10 et 20) : %v", len(got), got)
	}
	for _, id := range []int64{10, 20} {
		if _, ok := got[id]; !ok {
			t.Errorf("médaille %d absente du set", id)
		}
	}
	if _, ok := got[30]; ok {
		t.Error("médaille 30 (SUM(count)=0) ne doit pas figurer dans le set")
	}
	if _, ok := got[99]; ok {
		t.Error("médaille d'un AUTRE xuid ne doit pas figurer dans le set")
	}
}

// TestLoadEarnedMedalIDs_EmptyIsValid : 0 ligne sur une requête saine = joueur
// sans médaille, PAS une invalidité — set vide non-nil.
func TestLoadEarnedMedalIDs_EmptyIsValid(t *testing.T) {
	db := newMedalsEarnedDB(t)
	got := loadEarnedMedalIDs(context.Background(), db, "inconnu")
	if got == nil {
		t.Fatal("aucune ligne ≠ invalidité : le set doit être vide non-nil")
	}
	if len(got) != 0 {
		t.Fatalf("got %d entrées, want 0", len(got))
	}
}

// TestLoadEarnedMedalIDs_MissingTableIsInvalid : table absente (titre sans
// capability médailles) → nil, jamais un set partiel.
func TestLoadEarnedMedalIDs_MissingTableIsInvalid(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if got := loadEarnedMedalIDs(context.Background(), db, "x1"); got != nil {
		t.Fatalf("table absente : want nil (invalide), got %v", got)
	}
}

// TestLoadEarnedMedalIDs_CancelledContextIsInvalid : lecture interrompue
// (contexte annulé) → nil. C'est le cas que l'ancien code avalait : il rendait un
// set PARTIEL, et le diff « médaille inédite » du cycle suivant notifiait des
// médailles déjà connues.
func TestLoadEarnedMedalIDs_CancelledContextIsInvalid(t *testing.T) {
	db := newMedalsEarnedDB(t)
	if _, err := db.Exec(`INSERT INTO medals_earned VALUES ('m1','x1',10,1)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := loadEarnedMedalIDs(ctx, db, "x1"); got != nil {
		t.Fatalf("lecture interrompue : want nil (invalide), got %v", got)
	}
}

// NB : le bouclage « set nil ⇒ aucune notification » est déjà couvert par
// TestEmitMedalFirstEarned_ColdStartSeedsSilently (post_sync_medal_first_earned_test.go) —
// nil est précisément l'entrée que ce contrat d'invalidité produit.
