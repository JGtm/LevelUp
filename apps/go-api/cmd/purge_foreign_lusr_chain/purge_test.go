//go:build cgo

package main

// purge_test.go — la fixture est bâtie par les MIGRATIONS RÉELLES du titre, jamais par
// une DDL recopiée : une DDL de test recopiée dérive du schéma de prod sans que rien ne
// le signale, et c'est précisément le schéma (PK technique, index, vue _latest) que
// cette purge doit reposer à l'identique.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}

// newFixturePlayerDB crée une player DB migrée contenant 3 lignes saines et 2 lignes de
// chaîne étrangère, puis referme le handle (l'outil rouvre la base en RW exclusif).
func newFixturePlayerDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}
	start := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	rows := []struct {
		matchID, ratingType string
		group               any
	}{
		{"m1", "LUSR", "arena_slayer"},
		{"m2", "LUSR", "btb"},
		{"m3", "CSR", nil}, // playlist_group NULL : doit SURVIVRE (`<>` nu la jetterait)
		{"m4", "LUSR", "h5_arena"},
		{"m4", "LUSR_V2", "h5_arena"},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time)
			VALUES (?, ?, 1200, ?, ?)`, r.matchID, r.ratingType, r.group, start); err != nil {
			t.Fatalf("insert %s/%s: %v", r.matchID, r.ratingType, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture: %v", err)
	}
	return path
}

func reopen(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("reopen %s: %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// closeDB ferme explicitement un handle avant que l'outil rouvre la base en RW
// exclusif (le Cleanup de reopen refermera sans effet : Close est idempotent).
func closeDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func TestPurgeForeignLUSRChain_DryRunChangesNothing(t *testing.T) {
	path := newFixturePlayerDB(t)
	if err := run(context.Background(), path, "h5_arena", true, false); err != nil {
		t.Fatalf("run dry-run: %v", err)
	}
	db := reopen(t, path)
	if n := countRows(t, db, `SELECT COUNT(*) FROM match_skill_rank`); n != 5 {
		t.Errorf("lignes après dry-run = %d, want 5 (le dry-run n'écrit RIEN)", n)
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM match_skill_rank WHERE playlist_group = 'h5_arena'`); n != 2 {
		t.Errorf("lignes h5_arena après dry-run = %d, want 2", n)
	}
}

func TestPurgeForeignLUSRChain_CommitRebuildsWithoutForeignRows(t *testing.T) {
	path := newFixturePlayerDB(t)
	if err := run(context.Background(), path, "h5_arena", false, true); err != nil {
		t.Fatalf("run commit: %v", err)
	}
	db := reopen(t, path)

	if n := countRows(t, db, `SELECT COUNT(*) FROM match_skill_rank`); n != 3 {
		t.Errorf("lignes après purge = %d, want 3 (5 - 2 étrangères)", n)
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM match_skill_rank WHERE playlist_group = 'h5_arena'`); n != 0 {
		t.Errorf("lignes h5_arena restantes = %d, want 0", n)
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM match_skill_rank WHERE playlist_group IS NULL`); n != 1 {
		t.Errorf("ligne à playlist_group NULL = %d, want 1 (IS DISTINCT FROM, pas `<>`)", n)
	}

	// La vue est reposée : sans elle, les lecteurs applicatifs (match_skill_rank_latest)
	// casseraient. (Les index secondaires n'existent plus depuis le 2026-09-27.)
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM duckdb_views() WHERE view_name = 'match_skill_rank_latest'`); n != 1 {
		t.Errorf("vue match_skill_rank_latest = %d, want 1", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM match_skill_rank_latest`); n != 3 {
		t.Errorf("lignes servies par la vue = %d, want 3", n)
	}

	// La PK technique et son DEFAULT : un INSERT sans id doit continuer de marcher.
	if _, err := db.Exec(`INSERT INTO match_skill_rank
		(match_id, rating_type, rating_value, playlist_group) VALUES ('m5', 'LUSR', 1300, 'btb')`); err != nil {
		t.Errorf("INSERT post-purge (PK/séquence/défauts reposés ?): %v", err)
	}

	// Idempotence : une seconde passe ne trouve plus rien et ne casse rien.
	if err := run(context.Background(), path, "h5_arena", false, true); err != nil {
		t.Errorf("seconde passe: %v", err)
	}
}

func TestPurgeForeignLUSRChain_RefusesEmptyArguments(t *testing.T) {
	if err := run(context.Background(), "", "h5_arena", true, false); err == nil {
		t.Error("-db vide doit être refusé")
	}
	if err := run(context.Background(), newFixturePlayerDB(t), "", false, true); err == nil {
		t.Error("-chain vide doit être refusé (purge d'une chaîne non nommée)")
	}
}

// TestCensus_ForcesScanOnHealthyDB — non-régression du durcissement C.8 : le recensement
// lit par scan (`playlist_group || ”`), il doit rendre le compte exact.
func TestCensus_ForcesScanOnHealthyDB(t *testing.T) {
	path := newFixturePlayerDB(t)
	db := reopen(t, path)
	ctx := context.Background()

	c, err := censusForeignChain(ctx, db, "h5_arena")
	if err != nil {
		t.Fatalf("censusForeignChain: %v", err)
	}
	if c.ForeignRaw != 2 {
		t.Errorf("lignes étrangères par scan = %d, want 2", c.ForeignRaw)
	}
	if got := c.ForeignByRatingType; got["LUSR"] != 1 || got["LUSR_V2"] != 1 {
		t.Errorf("ventilation par rating_type = %v, want LUSR:1 LUSR_V2:1", got)
	}
}

// dependentViewNames liste les vues non internes qui référencent match_skill_rank.
func dependentViewNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT view_name FROM duckdb_views()
		WHERE internal = FALSE AND sql IS NOT NULL AND lower(sql) LIKE '%match_skill_rank%'
		ORDER BY view_name`)
	if err != nil {
		t.Fatalf("duckdb_views(): %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, n)
	}
	return out
}

// TestPurge_KeepsAllDependentViews — garde C.9 (2026-09-13).
//
// Le swap DROP+RENAME ne doit perdre AUCUNE vue. L'outil ne capturait que le nom
// `match_skill_rank_latest` ; depuis C.3 bis la table en porte deux, et une vue
// filtrée par nom disparaît en silence de l'outil — le trou ne se voit qu'au premier
// lecteur qui tombe sur « table does not exist ». Les deux vues sont posées ICI par
// les MIGRATIONS RÉELLES, pas par une DDL de test : c'est la seule façon que le test
// suive l'ajout d'une troisième vue.
func TestPurge_KeepsAllDependentViews(t *testing.T) {
	path := newFixturePlayerDB(t)

	db := reopen(t, path)
	before := dependentViewNames(t, db)
	if len(before) < 2 {
		t.Fatalf("vues dépendantes avant purge = %v, want ≥ 2 "+
			"(match_skill_rank_latest ET match_skill_rank_latest_by_type)", before)
	}
	for _, want := range []string{"match_skill_rank_latest", "match_skill_rank_latest_by_type"} {
		found := false
		for _, n := range before {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("la migration n'a pas posé %s ; vues = %v", want, before)
		}
	}
	closeDB(t, db)

	if err := run(context.Background(), path, "h5_arena", false, true); err != nil {
		t.Fatalf("run commit: %v", err)
	}

	db = reopen(t, path)
	after := dependentViewNames(t, db)
	if len(after) != len(before) {
		t.Fatalf("vues après purge = %v, want %v (aucune vue ne doit être perdue)", after, before)
	}
	// Chaque vue doit être INTERROGEABLE, pas seulement présente au catalogue : une
	// vue laissée liée à l'ancienne table serait listée mais casserait à la lecture.
	for _, name := range after {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + name + `"`).Scan(&n); err != nil {
			t.Errorf("la vue %s est listée mais illisible après le swap : %v", name, err)
		}
	}
	// Et elle doit servir la donnée PURGÉE (3 lignes saines, 0 étrangère).
	if n := countRows(t, db, `SELECT COUNT(*) FROM match_skill_rank_latest_by_type`); n != 3 {
		t.Errorf("match_skill_rank_latest_by_type sert %d lignes, want 3", n)
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM match_skill_rank_latest_by_type WHERE playlist_group = 'h5_arena'`); n != 0 {
		t.Errorf("la vue _by_type sert encore %d ligne(s) h5_arena", n)
	}
}

// TestCaptureDependentViews_SeesEveryViewOnTheTable — LE garde de C.9.
//
// C'est ici que se joue le défaut, pas en bout de chaîne : mesuré sur DuckDB, une vue
// dépendante SURVIT au `DROP TABLE` et se re-lie à la table recréée par le RENAME —
// une vue oubliée par la capture reste donc présente par accident, et l'assertion
// d'état final ne voit rien. Ce qu'il faut cadenasser, c'est que l'outil VOIE toutes
// les vues : filtre sur le SQL (`LIKE '%match_skill_rank%'`), jamais sur un nom.
// Les vues sont posées par les MIGRATIONS RÉELLES : une troisième vue ajoutée demain
// fait échouer ce test si la capture l'ignore.
func TestCaptureDependentViews_SeesEveryViewOnTheTable(t *testing.T) {
	db := reopen(t, newFixturePlayerDB(t))

	captured, err := captureDependentViews(context.Background(), db)
	if err != nil {
		t.Fatalf("captureDependentViews: %v", err)
	}
	got := map[string]string{}
	for _, v := range captured {
		got[v.name] = v.ddl
	}
	for _, want := range dependentViewNames(t, db) {
		ddl, ok := got[want]
		if !ok {
			t.Errorf("vue %q NON capturée — elle serait perdue par le swap ; capturées = %v",
				want, viewNames(captured))
			continue
		}
		if !strings.Contains(strings.ToLower(ddl), "match_skill_rank") {
			t.Errorf("DDL capturée pour %q ne référence pas la table : %q", want, ddl)
		}
	}
	if len(captured) != len(dependentViewNames(t, db)) {
		t.Errorf("capturées = %v, vues réelles = %v", viewNames(captured), dependentViewNames(t, db))
	}
}
