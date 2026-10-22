//go:build integration

package main

// cmd_backfill_killsource_resumes_integration_test.go — LA PASSE CRÉDIT DU BACKFILL ÉCRIT LES
// RÉSUMÉS PAR MATCH (ADR 0039).
//
// `levelup backfill-killsource` écrit ses passes par l'écrivain du journal, qui pose leurs
// résumés dans la même transaction. La passe crédit (SQL -> SQL, sans film) le montre sans
// fixture : la tête résume la passe que la vue sert ; une erreur base pendant l'écriture des
// résumés n'en laisse rien, et la commande la compte au lieu de s'arrêter.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/testutil"
)

// baseCreditAvecDuel : une base migrée, un match au registre et un duel de trois morts dans ses
// événements (deux joueurs, kills et morts appariés à moins de 5 ms).
func baseCreditAvecDuel(t *testing.T, matchID string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry (match_id) VALUES (?)`, matchID); err != nil {
		t.Fatalf("registre: %v", err)
	}
	for _, p := range [][2]string{{"1001", "Alpha"}, {"1002", "Bravo"}} {
		if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, ?, ?)`,
			matchID, p[0], p[1]); err != nil {
			t.Fatalf("participant: %v", err)
		}
	}
	for _, e := range []struct {
		xuid, typ string
		ms        int
	}{
		{"1001", "kill", 1000}, {"1002", "death", 1002},
		{"1002", "kill", 5000}, {"1001", "death", 5001},
		{"1001", "kill", 9000}, {"1002", "death", 9003},
	} {
		if _, err := db.Exec(`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid)
			VALUES (?, ?, ?, ?)`, matchID, e.typ, e.ms, e.xuid); err != nil {
			t.Fatalf("événement: %v", err)
		}
	}
	return db
}

func TestBackfillCredit_EcritLesResumesDeSaPasse(t *testing.T) {
	db := baseCreditAvecDuel(t, "m-backfill")
	if err := passeDuCredit(context.Background(), db, killsourceOptions{}, nil); err != nil {
		t.Fatalf("passeDuCredit: %v", err)
	}
	e := testutil.ExigerResumeDeLaPasseCourante(t, db, "m-backfill")
	if e.LignesJournal != 3 {
		t.Fatalf("%d morts écrites, attendu 3", e.LignesJournal)
	}
}

func TestBackfillCredit_ErreurDesResumesAnnuleLaPasse(t *testing.T) {
	db := baseCreditAvecDuel(t, "m-backfill-ko")
	testutil.CasserLaTableDesTetes(t, db)
	if err := passeDuCredit(context.Background(), db, killsourceOptions{}, nil); err != nil {
		t.Fatalf("passeDuCredit: %v (une erreur de match se compte, elle n'arrête pas la passe)", err)
	}
	testutil.ExigerAucunePasse(t, db, "m-backfill-ko")
}
