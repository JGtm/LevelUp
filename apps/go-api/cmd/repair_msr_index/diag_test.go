//go:build cgo

package main

// diag_test.go — la désynchronisation d'index ART n'est PAS reproductible sur
// commande (c'est un défaut amont, duckdb#23645). Comme pour le lot PSA, on teste
// donc ce qui est testable, et rien d'autre :
//   - la RÈGLE DE COMPARAISON (scan vs lookup, et l'union des index à reconstruire) ;
//   - la NON-RÉGRESSION sur une base saine : le diagnostic ne crie pas au loup, et
//     la réparation ne touche aucune donnée.
//
// La fixture est bâtie par les MIGRATIONS RÉELLES du titre : ce sont leurs index et
// leur DDL que l'outil capture et rejoue, une DDL recopiée ici dériverait.

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
	"levelup/go-api/internal/platform/duckdb/indexcheck"
)

func TestMain(m *testing.M) {
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	os.Exit(m.Run())
}

// newFixtureDB crée une player DB migrée, peuplée de lignes match_skill_rank
// couvrant les trois axes indexés (plusieurs chaînes, plusieurs rating_type,
// plusieurs versions d'un même match) plus une chaîne NULL.
func newFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player): %v", err)
	}

	base := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	rows := []struct {
		matchID, ratingType string
		group               any
		writtenAt           time.Time
	}{
		{"m1", "LUSR", "arena_slayer", base},
		{"m1", "LUSR", "arena_slayer", base.Add(time.Hour)}, // 2e passe du même match
		{"m1", "LUSR_V2", "arena_slayer", base},
		{"m2", "LUSR", "btb", base},
		{"m3", "LUSR", "h5_arena", base}, // chaîne étrangère, présente en prod
		{"m4", "CSR", nil, base},         // playlist_group NULL : clé non interrogeable
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
			VALUES (?, ?, 1200, ?, ?, ?)`,
			r.matchID, r.ratingType, r.group, base, r.writtenAt); err != nil {
			t.Fatalf("insert %s/%s: %v", r.matchID, r.ratingType, err)
		}
	}
	return db
}

func TestDiagnoseAll_HealthyFixtureHasNoDivergence(t *testing.T) {
	db := newFixtureDB(t)
	ctx := context.Background()

	reports, err := diagnoseAll(ctx, db)
	if err != nil {
		t.Fatalf("diagnoseAll: %v", err)
	}
	if len(reports) != len(msrAxes) {
		t.Fatalf("rapports = %d, want %d (un par axe indexé)", len(reports), len(msrAxes))
	}
	for _, r := range reports {
		if !r.OK() {
			t.Errorf("axe %q : %d écart(s) sur une base SAINE (faux positif) : %+v",
				r.Axis, len(r.Divergences), r.Divergences)
		}
		if r.ScannedRows != r.IndexedRows {
			t.Errorf("axe %q : scan=%d indexé=%d — les deux comptages doivent coïncider",
				r.Axis, r.ScannedRows, r.IndexedRows)
		}
		if r.Keys == 0 {
			t.Errorf("axe %q : aucune clé sondée — le diagnostic ne prouve rien", r.Axis)
		}
	}

	// L'axe playlist_group ignore la ligne à chaîne NULL (non interrogeable par
	// égalité) et ne la compte pas comme un écart.
	if got := reports[0].NullKeys; got != 1 {
		t.Errorf("clés NULL sur playlist_group = %d, want 1", got)
	}
	if got := reports[0].ScannedRows; got != 5 {
		t.Errorf("lignes scannées sur playlist_group = %d, want 5 (6 - la ligne NULL)", got)
	}

	if n := countDivergentAxes(reports); n != 0 {
		t.Errorf("axes divergents = %d, want 0", n)
	}
	if names := indexesToRebuild(reports); len(names) != 0 {
		t.Errorf("index à reconstruire = %v, want aucun sur base saine", names)
	}
}

// TestIndexesToRebuild_ComparisonRule — la règle de décision, isolée des données :
// un axe sans divergence ne déclenche rien, un axe en écart déclenche SES index,
// et l'union est dédoublonnée et ordonnée.
func TestIndexesToRebuild_ComparisonRule(t *testing.T) {
	diverging := []indexcheck.Divergence{{Key: []string{"h5_arena"}, Scanned: 1826, Indexed: 22}}

	cases := []struct {
		name    string
		reports []indexcheck.Report
		want    []string
	}{
		{
			name:    "tous les axes sains",
			reports: []indexcheck.Report{{Axis: "a"}, {Axis: "b"}, {Axis: "c"}},
			want:    nil,
		},
		{
			name: "playlist_group en écart (le cas JGtm du 2026-09-13)",
			reports: []indexcheck.Report{
				{Axis: "playlist_group", Divergences: diverging},
				{Axis: "rating_type"},
				{Axis: "triplet"},
			},
			want: []string{"idx_msr_playlist"},
		},
		{
			name: "deux axes en écart → union ordonnée",
			reports: []indexcheck.Report{
				{Axis: "playlist_group", Divergences: diverging},
				{Axis: "rating_type", Divergences: diverging},
				{Axis: "triplet"},
			},
			want: []string{"idx_msr_playlist", "idx_msr_rating_type"},
		},
		{
			name: "tous les axes en écart",
			reports: []indexcheck.Report{
				{Axis: "playlist_group", Divergences: diverging},
				{Axis: "rating_type", Divergences: diverging},
				{Axis: "triplet", Divergences: diverging},
			},
			want: []string{"idx_msr_match_lookup", "idx_msr_playlist", "idx_msr_rating_type"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := indexesToRebuild(c.reports)
			if len(got) != len(c.want) {
				t.Fatalf("index = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("index = %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestRepairIndexes_LeavesDataUntouched — non-régression : sur une base saine, la
// réparation reconstruit la DDL capturée, ne perd aucune ligne et laisse les index
// en place. C'est le garde qui compte : l'outil ne doit JAMAIS toucher aux données.
func TestRepairIndexes_LeavesDataUntouched(t *testing.T) {
	db := newFixtureDB(t)
	ctx := context.Background()

	before, err := countRows(ctx, db)
	if err != nil {
		t.Fatalf("countRows: %v", err)
	}
	idxBefore, err := existingIndexes(ctx, db)
	if err != nil {
		t.Fatalf("existingIndexes: %v", err)
	}
	if len(idxBefore) < 3 {
		t.Fatalf("index posés par les migrations = %v, want ≥ 3", idxBefore)
	}
	ddls, err := captureIndexDDL(ctx, db)
	if err != nil {
		t.Fatalf("captureIndexDDL: %v", err)
	}
	for _, name := range []string{"idx_msr_playlist", "idx_msr_rating_type", "idx_msr_match_lookup"} {
		if _, ok := ddls[name]; !ok {
			t.Fatalf("DDL de %s non capturée dans la base ; capturées = %v", name, keysOf(ddls))
		}
	}

	if err := repairIndexes(ctx, db, []string{"idx_msr_playlist", "idx_msr_rating_type", "idx_msr_match_lookup"}); err != nil {
		t.Fatalf("repairIndexes: %v", err)
	}

	after, err := countRows(ctx, db)
	if err != nil {
		t.Fatalf("countRows après: %v", err)
	}
	if after != before {
		t.Errorf("lignes avant=%d après=%d — la réparation ne doit toucher AUCUNE donnée", before, after)
	}
	idxAfter, err := existingIndexes(ctx, db)
	if err != nil {
		t.Fatalf("existingIndexes après: %v", err)
	}
	if len(idxAfter) != len(idxBefore) {
		t.Errorf("index avant=%v après=%v — la reconstruction doit tout reposer", idxBefore, idxAfter)
	}
	reports, err := diagnoseAll(ctx, db)
	if err != nil {
		t.Fatalf("diagnoseAll après: %v", err)
	}
	if n := countDivergentAxes(reports); n != 0 {
		t.Errorf("axes divergents après réparation = %d, want 0", n)
	}
}

// TestRepairIndexes_RefusesUnknownIndex — un index absent de la base n'a pas de DDL
// capturable : l'outil refuse plutôt que d'inventer une DDL de remplacement.
func TestRepairIndexes_RefusesUnknownIndex(t *testing.T) {
	db := newFixtureDB(t)
	err := repairIndexes(context.Background(), db, []string{"idx_msr_inexistant"})
	if err == nil {
		t.Fatal("un index sans DDL capturée doit être refusé, pas recréé de mémoire")
	}
	if !strings.Contains(err.Error(), "steps_player_match_skill_rank.go") {
		t.Errorf("l'erreur doit renvoyer à l'autorité de la DDL ; err = %v", err)
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestRepairIndexes_LeavesViewsUntouched — garde C.9 : la réparation d'index ne
// touche QUE des index. DROP/CREATE INDEX n'a aucune raison d'affecter une vue, mais
// c'est exactement ce qu'on a cru du swap de la purge avant de le mesurer : on le
// vérifie plutôt que de le supposer. Les vues sont posées par les migrations réelles.
func TestRepairIndexes_LeavesViewsUntouched(t *testing.T) {
	db := newFixtureDB(t)
	ctx := context.Background()

	before, err := dependentViewNames(ctx, db)
	if err != nil {
		t.Fatalf("dependentViewNames: %v", err)
	}
	if len(before) < 2 {
		t.Fatalf("vues avant = %v, want ≥ 2 (les deux vues de lecture de match_skill_rank)", before)
	}

	if err := repairIndexes(ctx, db, []string{"idx_msr_playlist", "idx_msr_rating_type", "idx_msr_match_lookup"}); err != nil {
		t.Fatalf("repairIndexes: %v", err)
	}

	after, err := dependentViewNames(ctx, db)
	if err != nil {
		t.Fatalf("dependentViewNames après: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("vues avant=%v après=%v — la réparation d'index ne doit toucher aucune vue", before, after)
	}
	for _, name := range after {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+name+`"`).Scan(&n); err != nil {
			t.Errorf("la vue %s est illisible après réparation des index : %v", name, err)
		}
	}
}

// TestEnsureReadViews_RestoresADroppedView — le drapeau -ensure-views repose une vue
// perdue APRÈS que son step a été inscrit au ledger : le runner ne rejouerait jamais
// ce step, la vue ne reviendrait donc jamais d'elle-même.
func TestEnsureReadViews_RestoresADroppedView(t *testing.T) {
	db := newFixtureDB(t)
	ctx := context.Background()

	before, err := dependentViewNames(ctx, db)
	if err != nil {
		t.Fatalf("dependentViewNames: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP VIEW IF EXISTS match_skill_rank_latest_by_type`); err != nil {
		t.Fatalf("DROP VIEW: %v", err)
	}
	if names, _ := dependentViewNames(ctx, db); len(names) != len(before)-1 {
		t.Fatalf("la vue n'a pas été retirée pour le test ; vues = %v", names)
	}

	if err := ensureReadViews(ctx, db); err != nil {
		t.Fatalf("ensureReadViews: %v", err)
	}

	after, err := dependentViewNames(ctx, db)
	if err != nil {
		t.Fatalf("dependentViewNames après: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("vues après -ensure-views = %v, want %v", after, before)
	}
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank_latest_by_type`).Scan(&n); err != nil {
		t.Errorf("la vue reposée est illisible : %v", err)
	}
	// Idempotent : une seconde passe ne casse rien (CREATE OR REPLACE).
	if err := ensureReadViews(ctx, db); err != nil {
		t.Errorf("seconde passe: %v", err)
	}
}
