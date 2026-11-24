package migrations

// shared_highlight_events_dedupe_test.go — shared_highlight_events_dedupe_v1 joué PAR LE RUNNER :
// il retire les lignes écrites deux fois sur Halo Infinite et ne touche pas à Halo 5, dont les
// lignes identiques sont légitimes.

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

const stepDedupeHighlightEvents = "shared_highlight_events_dedupe_v1"

// baseAvecHighlightEventsDoubles : base partagée neuve migrée par le runner pour `slug`, puis
// remplie d'un match recopié en bloc et ré-armée pour le step (ligne du ledger retirée : écriture
// de TEST sur une base jetable).
func baseAvecHighlightEventsDoubles(t *testing.T, slug string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB(%s) base neuve: %v", slug, err)
	}
	for i := 0; i < 2; i++ {
		for _, stmt := range []string{
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint) VALUES ('m', 'kill', 1000, 'x1', 50)`,
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint) VALUES ('m', 'medal', 2000, 'x1', 50)`,
		} {
			if _, err := db.Exec(stmt); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE name = ?`, stepDedupeHighlightEvents); err != nil {
		t.Fatalf("ré-armement: %v", err)
	}
	return db
}

func TestHighlightEventsDedupe_ParLeRunner_HaloInfinite(t *testing.T) {
	db := baseAvecHighlightEventsDoubles(t, migration.DefaultSlug)
	if err := migration.RunForTitleDB(db, migration.DefaultSlug, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM highlight_events`); n != 2 {
		t.Errorf("highlight_events = %d lignes, attendu 2 (le match recopié perd sa seconde moitié)", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stepDedupeHighlightEvents); n != 1 {
		t.Errorf("ledger : %d inscription(s), attendu 1", n)
	}
}

func TestHighlightEventsDedupe_ParLeRunner_Halo5NonTouche(t *testing.T) {
	db := baseAvecHighlightEventsDoubles(t, slugSansBit)
	if err := migration.RunForTitleDB(db, slugSansBit, migration.TargetShared); err != nil {
		t.Fatalf("RunForTitleDB: %v", err)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM highlight_events`); n != 4 {
		t.Errorf("highlight_events = %d lignes, attendu 4 (Halo 5 : lignes identiques légitimes, rien retiré)", n)
	}
	if n := scanRegistre(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, stepDedupeHighlightEvents); n != 0 {
		t.Error("le step ne doit ni s'appliquer ni s'inscrire hors de son titre")
	}
}
