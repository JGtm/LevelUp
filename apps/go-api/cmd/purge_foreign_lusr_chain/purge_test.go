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

	// La vue et les index sont reposés : sans eux, les lecteurs applicatifs
	// (match_skill_rank_latest) casseraient et les scans repartiraient en full table.
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM duckdb_views() WHERE view_name = 'match_skill_rank_latest'`); n != 1 {
		t.Errorf("vue match_skill_rank_latest = %d, want 1", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM match_skill_rank_latest`); n != 3 {
		t.Errorf("lignes servies par la vue = %d, want 3", n)
	}
	if n := countRows(t, db,
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'match_skill_rank'`); n < 3 {
		t.Errorf("index sur match_skill_rank = %d, want ≥ 3 (idx_msr_match_lookup, _rating_type, _playlist)", n)
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

// TestCensus_ForcesScanAndMatchesLookupOnHealthyDB — sur une base saine, le compte
// par scan forcé et le compte par lookup indexé coïncident, et le pré-vol passe.
// C'est la non-régression du durcissement C.8 : le recensement lit désormais par
// scan (`playlist_group || ”`), il doit continuer de rendre le compte exact.
func TestCensus_ForcesScanAndMatchesLookupOnHealthyDB(t *testing.T) {
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
	if c.ForeignIndexed != c.ForeignRaw {
		t.Errorf("lookup=%d scan=%d : les deux comptages doivent coïncider sur une base saine",
			c.ForeignIndexed, c.ForeignRaw)
	}
	if c.indexMismatch() {
		t.Error("indexMismatch() vrai sur une base saine (faux positif)")
	}
	if err := checkIndexCoherence(ctx, path, "h5_arena", c, true); err != nil {
		t.Errorf("le pré-vol doit passer sur une base saine ; err = %v", err)
	}
}

// TestCheckIndexCoherence_BlocksCommitOnDesync — la règle de refus, isolée des
// données (la désynchronisation ART n'est pas reproductible sur commande) : en
// dry-run l'écart est signalé sans bloquer, en -commit il interdit toute écriture
// et nomme l'outil de réparation.
func TestCheckIndexCoherence_BlocksCommitOnDesync(t *testing.T) {
	// Les comptes réels mesurés sur la base de JGtm le 2026-09-13.
	desync := chainCensus{TotalRows: 12000, ForeignRaw: 1826, ForeignIndexed: 22}
	ctx := context.Background()
	const dbPath = "data/titles/halo_infinite/players/JGtm/stats.duckdb"

	if err := checkIndexCoherence(ctx, dbPath, "h5_arena", desync, false); err != nil {
		t.Errorf("en dry-run l'écart se SIGNALE sans bloquer ; err = %v", err)
	}

	err := checkIndexCoherence(ctx, dbPath, "h5_arena", desync, true)
	if err == nil {
		t.Fatal("en -commit, un index désynchronisé doit INTERDIRE l'écriture")
	}
	for _, want := range []string{"repair_msr_index", "1826", "22", "h5_arena"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("le refus doit mentionner %q pour être actionnable ; err = %v", want, err)
		}
	}
}
