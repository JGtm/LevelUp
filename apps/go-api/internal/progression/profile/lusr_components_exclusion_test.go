package profile

// lusr_components_exclusion_test.go — la composante LUSR d'un match exclu ne pèse plus
// dans les agrégats du profil.
//
// Fixture sur fichier migré (schéma réel : vues _latest et étapes d'enrichissement).

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

const exclusionComponent = "win_factor"

func newExclusionPlayerDB(t *testing.T) *duckdb.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	raw, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open stats: %v", err)
	}
	if err := migration.RunForDB(raw, migration.TargetPlayer); err != nil {
		_ = raw.Close()
		t.Fatalf("migrate stats: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close stats: %v", err)
	}
	db, err := duckdb.OpenReadWrite(path)
	if err != nil {
		t.Fatalf("reopen stats: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func seedComponentValue(t *testing.T, db *duckdb.DB, matchID string, value float64, at time.Time) {
	t.Helper()
	if _, err := db.Exec(context.Background(),
		`INSERT INTO lusr_component_history (match_id, component_name, value, weight, computed_at)
		 VALUES (?, ?, ?, 0.1, ?)`, matchID, exclusionComponent, value, at); err != nil {
		t.Fatalf("insert component %s: %v", matchID, err)
	}
}

func seedExclusion(t *testing.T, db *duckdb.DB, matchID string) {
	t.Helper()
	if _, err := db.Exec(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, is_excluded, stage) VALUES (?, TRUE, 'exclusion')`,
		matchID); err != nil {
		t.Fatalf("insert exclusion %s: %v", matchID, err)
	}
}

func TestLoadLUSRComponentsBreakdown_ExcludedMatchDoesNotWeigh(t *testing.T) {
	db := newExclusionPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedComponentValue(t, db, "m1", 0.2, day)
	seedComponentValue(t, db, "m2", 0.4, day.Add(24*time.Hour))
	seedComponentValue(t, db, "m3", 0.9, day.Add(48*time.Hour))
	seedExclusion(t, db, "m3")

	rows, err := NewService(db).loadLUSRComponentsBreakdown(context.Background(), "u")
	if err != nil {
		t.Fatalf("loadLUSRComponentsBreakdown: %v", err)
	}
	got, ok := rows[exclusionComponent]
	if !ok {
		t.Fatalf("composante %s absente : %v", exclusionComponent, rows)
	}
	if math.Abs(got.currentAvg-0.3) > 1e-9 {
		t.Errorf("moyenne = %v, want 0.3 (m3 exclu ; avec lui : 0.5)", got.currentAvg)
	}
}

func TestLoadLUSRComponentsBreakdown_NoExclusionKeepsEveryMatch(t *testing.T) {
	db := newExclusionPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedComponentValue(t, db, "m1", 0.2, day)
	seedComponentValue(t, db, "m2", 0.4, day.Add(24*time.Hour))
	seedComponentValue(t, db, "m3", 0.9, day.Add(48*time.Hour))

	rows, err := NewService(db).loadLUSRComponentsBreakdown(context.Background(), "u")
	if err != nil {
		t.Fatalf("loadLUSRComponentsBreakdown: %v", err)
	}
	if got := rows[exclusionComponent].currentAvg; math.Abs(got-0.5) > 1e-9 {
		t.Errorf("moyenne = %v, want 0.5 (aucun match exclu : liste vide, aucune clause)", got)
	}
}

func TestLusrComponentsQuery_ClauseOnlyWhenExcluded(t *testing.T) {
	q, args := lusrComponentsQuery(nil)
	if len(args) != 0 || strings.Contains(q, "NOT IN") {
		t.Errorf("liste vide : args=%v, clause NOT IN présente=%v ; want aucune clause", args, strings.Contains(q, "NOT IN"))
	}
	q, args = lusrComponentsQuery([]string{"a", "b"})
	if len(args) != 2 || !strings.Contains(q, "match_id NOT IN (?, ?)") {
		t.Errorf("deux exclus : args=%v, want 2 paramètres et `match_id NOT IN (?, ?)`\n%s", args, q)
	}
}

// breakExclusionRead remplace la vue des enrichissements par une vue dont la lecture échoue
// à l'exécution (conversion impossible), sans message de schéma absent : la lecture des
// exclusions échoue alors que la table des composantes reste lisible.
func breakExclusionRead(t *testing.T, db *duckdb.DB) {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE exclusion_boom (v VARCHAR)`,
		`INSERT INTO exclusion_boom VALUES ('x')`,
		`CREATE OR REPLACE VIEW player_match_enrichment_latest AS
		 SELECT CAST(v AS INTEGER)::VARCHAR AS match_id, TRUE AS is_excluded FROM exclusion_boom`,
	} {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// Une lecture des exclusions en échec remonte une erreur : servir les composantes en
// laissant peser un match exclu serait une réponse fausse, pas dégradée.
func TestLoadLUSRComponentsBreakdown_ExclusionReadFailureIsReturned(t *testing.T) {
	db := newExclusionPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedComponentValue(t, db, "m1", 0.2, day)
	seedComponentValue(t, db, "m2", 0.4, day.Add(24*time.Hour))
	seedComponentValue(t, db, "m3", 0.9, day.Add(48*time.Hour))
	seedExclusion(t, db, "m3")
	breakExclusionRead(t, db)

	rows, err := NewService(db).loadLUSRComponentsBreakdown(context.Background(), "u")
	if err == nil {
		t.Fatalf("erreur nil, want l'échec de lecture des exclusions ; composantes servies : %v", rows)
	}
	if !strings.Contains(err.Error(), "lecture des matchs exclus") {
		t.Errorf("erreur = %v, want le contexte « lecture des matchs exclus »", err)
	}
	if len(rows) != 0 {
		t.Errorf("composantes servies malgré l'échec : %v", rows)
	}
}

// Une base sans la vue des enrichissements ne porte aucune exclusion : le calcul continue
// sur tous les matchs (contrat de skill.LoadExcludedMatchIDs).
func TestLoadLUSRComponentsBreakdown_MissingEnrichmentViewMeansNoExclusion(t *testing.T) {
	db := newExclusionPlayerDB(t)
	day := time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC)
	seedComponentValue(t, db, "m1", 0.2, day)
	seedComponentValue(t, db, "m2", 0.4, day.Add(24*time.Hour))
	if _, err := db.Exec(context.Background(), `DROP VIEW player_match_enrichment_latest`); err != nil {
		t.Fatalf("drop view: %v", err)
	}

	rows, err := NewService(db).loadLUSRComponentsBreakdown(context.Background(), "u")
	if err != nil {
		t.Fatalf("loadLUSRComponentsBreakdown: %v", err)
	}
	if got := rows[exclusionComponent].currentAvg; math.Abs(got-0.3) > 1e-9 {
		t.Errorf("moyenne = %v, want 0.3 (aucune exclusion lisible)", got)
	}
}
