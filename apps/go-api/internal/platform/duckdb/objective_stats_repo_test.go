//go:build cgo

// Package duckdb — objective_stats_repo_test.go : couverture d'ObjectiveStatsRepo
// (PLAN_V72_OBJECTIVE_STATS P4), jusqu'ici SANS aucun test malgré 10 SUM et une
// dégradation silencieuse (contre-revue V7.2).
//
// Deux propriétés à verrouiller :
//   - l'AGRÉGAT (SUM par xuid sur match_objective_stats_latest, filtré xuid IN +
//     match_id IN) — une colonne mal mappée passerait inaperçue ;
//   - la DÉGRADATION : vue absente (DB non migrée / titre sans capability) ⇒
//     map vide + nil, JAMAIS d'échec dur d'une page.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// memSharedReader : SharedReader minimal branché sur une *sql.DB in-memory.
// Évite de construire un PlayerDB complet — le repo ne consomme que
// pdb.SharedReadDB().
type memSharedReader struct{ db *sql.DB }

func (r memSharedReader) Get(_ context.Context) (*sql.DB, func(), error) {
	return r.db, func() {}, nil
}

// newObjectiveStatsRepoOnMem construit un repo lisant une DuckDB mémoire.
// migrate=true applique les VRAIES migrations shared (table append-only +
// vue _latest) ; false laisse la DB nue pour tester la dégradation.
func newObjectiveStatsRepoOnMem(t *testing.T, migrate bool) (*ObjectiveStatsRepo, *sql.DB) {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if migrate {
		// Le provider de steps title-owned est câblé par le TestMain du package
		// (migration_provider_test.go) — même chaîne qu'au boot.
		if err := migration.RunForDB(db, migration.TargetShared); err != nil {
			t.Fatalf("RunForDB(Shared): %v", err)
		}
	}
	pdb := &PlayerDB{SharedReader: memSharedReader{db: db}}
	return NewObjectiveStatsRepo(pdb), db
}

// insertObjectiveRow insère une ligne match_objective_stats (append-only).
func insertObjectiveRow(t *testing.T, db *sql.DB, matchID, xuid string, cols map[string]any) {
	t.Helper()
	names := "match_id, xuid"
	placeholders := "?, ?"
	args := []any{matchID, xuid}
	for k, v := range cols {
		names += ", " + k
		placeholders += ", ?"
		args = append(args, v)
	}
	q := "INSERT INTO match_objective_stats (" + names + ") VALUES (" + placeholders + ")"
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("insert objective row: %v\nSQL: %s", err, q)
	}
}

// TestObjectiveStatsRepo_Aggregate : SUM par xuid sur les 10 colonnes servies,
// scope (xuid IN, match_id IN) respecté.
func TestObjectiveStatsRepo_Aggregate(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)

	// Joueur 1 : deux matchs dans le scope, CTF + Zones + Oddball.
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{
		"flag_captures": 2, "flag_returns": 1, "flag_steals": 3,
		"time_as_flag_carrier_seconds": 40.5,
		"zone_captures":                4, "zone_secures": 1, "time_in_zones_seconds": 12.25,
		"skull_grabs": 5, "time_as_skull_carrier_seconds": 8.0,
	})
	insertObjectiveRow(t, db, "m2", "x1", map[string]any{
		"flag_captures": 1, "flag_returns": 2, "flag_steals": 0,
		"time_as_flag_carrier_seconds": 9.5,
		"zone_captures":                1, "zone_secures": 2, "time_in_zones_seconds": 0.75,
		"skull_grabs": 0, "time_as_skull_carrier_seconds": 2.0,
	})
	// Match HORS scope (ne doit pas être sommé).
	insertObjectiveRow(t, db, "m_hors_scope", "x1", map[string]any{"flag_captures": 100})
	// Autre joueur : entrée distincte dans le résultat.
	insertObjectiveRow(t, db, "m1", "x2", map[string]any{"zone_captures": 7})
	// Joueur hors scope xuid.
	insertObjectiveRow(t, db, "m1", "x3", map[string]any{"flag_captures": 50})

	got, err := repo.LoadAggregatedByXUID(context.Background(),
		[]string{"m1", "m2"}, []string{"x1", "x2"})
	if err != nil {
		t.Fatalf("LoadAggregatedByXUID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d xuids, want 2 (x1, x2) : %v", len(got), got)
	}

	a := got["x1"]
	if a == nil {
		t.Fatal("agrégat x1 absent")
	}
	checks := []struct {
		field string
		got   float64
		want  float64
	}{
		{"FlagCaptures", float64(a.FlagCaptures), 3},
		{"FlagReturns", float64(a.FlagReturns), 3},
		{"FlagSteals", float64(a.FlagSteals), 3},
		{"FlagCarrierSeconds", a.FlagCarrierSeconds, 50.0},
		{"ZoneCaptures", float64(a.ZoneCaptures), 5},
		{"ZoneSecures", float64(a.ZoneSecures), 3},
		{"ZoneSeconds", a.ZoneSeconds, 13.0},
		{"SkullGrabs", float64(a.SkullGrabs), 5},
		{"SkullCarrierSeconds", a.SkullCarrierSeconds, 10.0},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("x1.%s = %v, want %v (SUM hors match hors scope)", c.field, c.got, c.want)
		}
	}

	b := got["x2"]
	if b == nil || b.ZoneCaptures != 7 {
		t.Errorf("agrégat x2 = %+v, want ZoneCaptures=7", b)
	}
	if _, ok := got["x3"]; ok {
		t.Error("x3 hors scope xuid ne doit pas figurer dans le résultat")
	}
}

// TestObjectiveStatsRepo_LatestViewWins : la lecture passe par la vue _latest
// (règle ART n°2) — une ré-écriture append-only remplace la valeur, elle ne
// s'additionne pas à l'ancienne.
func TestObjectiveStatsRepo_LatestViewWins(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	if _, err := db.Exec(`
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 1, TIMESTAMP '2026-01-01 00:00:00');
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 4, TIMESTAMP '2026-01-02 00:00:00');
	`); err != nil {
		t.Fatalf("seed append-only: %v", err)
	}

	got, err := repo.LoadAggregatedByXUID(context.Background(), []string{"m1"}, []string{"x1"})
	if err != nil {
		t.Fatalf("LoadAggregatedByXUID: %v", err)
	}
	if got["x1"] == nil {
		t.Fatal("agrégat x1 absent")
	}
	if got["x1"].FlagCaptures != 4 {
		t.Errorf("FlagCaptures = %d, want 4 (dernière version via la vue _latest, pas 1+4=5)",
			got["x1"].FlagCaptures)
	}
}

// TestObjectiveStatsRepo_EmptyScopeShortCircuits : scope vide → map vide sans
// requête (ni erreur).
func TestObjectiveStatsRepo_EmptyScopeShortCircuits(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, true)
	for _, c := range []struct {
		name     string
		matchIDs []string
		xuids    []string
	}{
		{"aucun match", nil, []string{"x1"}},
		{"aucun xuid", []string{"m1"}, nil},
		{"les deux vides", nil, nil},
	} {
		got, err := repo.LoadAggregatedByXUID(context.Background(), c.matchIDs, c.xuids)
		if err != nil {
			t.Errorf("%s: err = %v, want nil", c.name, err)
		}
		if len(got) != 0 {
			t.Errorf("%s: got %v, want map vide", c.name, got)
		}
	}
}

// TestObjectiveStatsRepo_MissingViewDegrades : vue absente (DB non migrée, titre
// sans la capability) → map vide + err nil. La page qui consomme le repo reste
// servie SANS bloc objectifs, jamais en erreur.
func TestObjectiveStatsRepo_MissingViewDegrades(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, false) // aucune migration → pas de vue

	got, err := repo.LoadAggregatedByXUID(context.Background(), []string{"m1"}, []string{"x1"})
	if err != nil {
		t.Fatalf("dégradation attendue silencieuse, err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want map vide", got)
	}
}

// TestObjectiveStatsRepo_AllZeroRowsOmitted : une ligne à zéro partout
// (mode sans objectif, colonnes NULL) ne produit PAS d'entrée — IsEmpty filtre,
// le front n'affiche donc pas un bloc « Objectifs » entièrement à 0.
func TestObjectiveStatsRepo_AllZeroRowsOmitted(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	insertObjectiveRow(t, db, "m1", "x1", map[string]any{"flag_captures": 0})

	got, err := repo.LoadAggregatedByXUID(context.Background(), []string{"m1"}, []string{"x1"})
	if err != nil {
		t.Fatalf("LoadAggregatedByXUID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want map vide (agrégat entièrement nul → entrée omise)", got)
	}
}
