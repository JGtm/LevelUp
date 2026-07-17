//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// seedRelationsWhatsNew crée un schéma Q28 minimal avec des timestamps RELATIFS à
// `now` pour exercer les colonnes encounters_30d / prev_seen_before_window du
// volet « Quoi de neuf ». Le repo calcule son cutoff via time.Now().UTC() ; `now`
// doit donc être proche du temps réel (l'appelant passe time.Now().UTC()).
//
// Relations seedées (chacune >= 2 matchs communs, HAVING satisfait) :
//   - RevyPlayer : matchs à now-200 j et now-5 j  → RAVIVÉE
//     (encounters_30d = 1 ; prev_seen_before_window = now-200 j)
//   - RegPlayer  : matchs à now-10 j et now-20 j  → NON ravivée / régulière
//     (encounters_30d = 2 ; prev_seen_before_window = NULL)
func seedRelationsWhatsNew(t *testing.T, db *DB, now time.Time) {
	t.Helper()
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER, kda DOUBLE)`,
		`CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMPTZ, start_time TIMESTAMP, pair_name VARCHAR, map_name VARCHAR, map_name_fr VARCHAR)`,
		`CREATE TABLE killer_victim_pairs (match_id VARCHAR, killer_xuid VARCHAR, victim_xuid VARCHAR, kill_count INTEGER)`,
		`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE VIEW v_gamertag_lookup AS SELECT xuid, gamertag FROM xuid_aliases`,
	} {
		if _, err := db.Exec(ctx, ddl); err != nil {
			t.Fatalf("seedRelationsWhatsNew DDL: %v\nSQL: %s", err, ddl)
		}
	}
	day := 24 * time.Hour
	if _, err := db.Exec(ctx, `INSERT INTO match_participants VALUES
		('rv1','xuidMe',0,2,1.0), ('rv1','xuidRevy',0,2,1.0),
		('rv2','xuidMe',0,2,1.0), ('rv2','xuidRevy',0,2,1.0),
		('rg1','xuidMe',0,2,1.0), ('rg1','xuidReg',0,2,1.0),
		('rg2','xuidMe',0,2,1.0), ('rg2','xuidReg',0,2,1.0)`); err != nil {
		t.Fatalf("seedRelationsWhatsNew participants: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO match_registry VALUES
		('rv1', ?, NULL, 'Slayer', 'Aquarius', NULL),
		('rv2', ?, NULL, 'Slayer', 'Aquarius', NULL),
		('rg1', ?, NULL, 'Slayer', 'Aquarius', NULL),
		('rg2', ?, NULL, 'Slayer', 'Aquarius', NULL)`,
		now.Add(-200*day), now.Add(-5*day), now.Add(-10*day), now.Add(-20*day)); err != nil {
		t.Fatalf("seedRelationsWhatsNew registry: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO xuid_aliases VALUES
		('xuidMe','MePlayer'), ('xuidRevy','RevyPlayer'), ('xuidReg','RegPlayer')`); err != nil {
		t.Fatalf("seedRelationsWhatsNew aliases: %v", err)
	}
}

func rowByGamertag(rows []domain.RelationRawRow, gt string) *domain.RelationRawRow {
	for i := range rows {
		if rows[i].Gamertag == gt {
			return &rows[i]
		}
	}
	return nil
}

// assertWhatsNewColumns vérifie encounters_30d / prev_seen_before_window sur la
// relation ravivée (Revy) et la régulière (Reg), quel que soit le template.
func assertWhatsNewColumns(t *testing.T, rows []domain.RelationRawRow, now time.Time) {
	t.Helper()
	day := 24 * time.Hour
	revy := rowByGamertag(rows, "RevyPlayer")
	reg := rowByGamertag(rows, "RegPlayer")
	if revy == nil || reg == nil {
		t.Fatalf("missing Revy/Reg rows: %+v", rows)
	}
	// Ravivée : 1 rencontre récente (now-5 j), la précédente à now-200 j.
	if revy.Encounters30d != 1 {
		t.Fatalf("Revy encounters_30d=%d want 1", revy.Encounters30d)
	}
	if revy.PrevSeenBeforeWindow == nil {
		t.Fatal("Revy prev_seen_before_window must be set (now-200 j)")
	} else if gap := now.Sub(*revy.PrevSeenBeforeWindow); gap < 199*day || gap > 201*day {
		t.Fatalf("Revy prev_seen gap=%v want ~200 j", gap)
	}
	// Régulière : 2 rencontres récentes (now-10 j, now-20 j), aucune antérieure.
	if reg.Encounters30d != 2 {
		t.Fatalf("Reg encounters_30d=%d want 2", reg.Encounters30d)
	}
	if reg.PrevSeenBeforeWindow != nil {
		t.Fatalf("Reg prev_seen_before_window=%v want nil (aucune rencontre < fenêtre)", reg.PrevSeenBeforeWindow)
	}
}

// TestCareerRepo_GetRelations_WhatsNewColumns : les colonnes du volet « Quoi de
// neuf » sont correctes sur le template NON scopé (Q28RelationsTpl) ET le
// template scopé (Q28RelationsScopedTpl).
func TestCareerRepo_GetRelations_WhatsNewColumns(t *testing.T) {
	now := time.Now().UTC()

	db := openMemDB(t)
	seedRelationsWhatsNew(t, db, now)
	pdb := &PlayerDB{Player: db, Shared: db, XUID: "xuidMe", Gamertag: "MePlayer"}
	repo := NewCareerRepo(pdb)
	ctx := context.Background()

	// Template NON scopé (scope nil).
	nonScoped, err := repo.GetRelations(ctx, nil)
	if err != nil {
		t.Fatalf("GetRelations(nil): %v", err)
	}
	assertWhatsNewColumns(t, nonScoped, now)

	// Template SCOPÉ (scope = tous les matchs → mêmes 2 relations).
	scoped, err := repo.GetRelations(ctx, []string{"rv1", "rv2", "rg1", "rg2"})
	if err != nil {
		t.Fatalf("GetRelations(scoped): %v", err)
	}
	assertWhatsNewColumns(t, scoped, now)
}
