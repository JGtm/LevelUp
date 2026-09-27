package sync

// schema_msr_views_test.go — garde-rail R1 de la revue finitions (2026-09-13).
//
// Une player DB créée par EnsurePlayerSchema SEUL (chemin d'onboarding entre deux boots,
// player DB Halo 5 hors de la boucle de migration du boot) doit porter les DEUX vues de
// lecture de match_skill_rank : `match_skill_rank_latest` et
// `match_skill_rank_latest_by_type`. Avant ce garde, la seconde n'était posée que par la
// migration `player_msr_view_latest_by_type_v1`, et Q8LUSRHistoryPlayer (page Carrière)
// tombait en `Catalog Error` sur une base non migrée.
//
// L'invariant symétrique (Ensure = no-op sur une DB migrée) est tenu par
// schema_authority_test.go ; celui-ci couvre l'autre sens : Ensure seul suffit.

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func TestEnsurePlayerSchema_PosesLesDeuxVuesMatchSkillRank(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}

	views := map[string]string{}
	rows, err := db.QueryContext(t.Context(),
		`SELECT view_name, sql FROM duckdb_views() WHERE NOT internal AND view_name LIKE 'match_skill_rank_latest%'`)
	if err != nil {
		t.Fatalf("duckdb_views: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			t.Fatalf("scan: %v", err)
		}
		views[name] = ddl
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	for _, want := range []string{"match_skill_rank_latest", "match_skill_rank_latest_by_type"} {
		if _, ok := views[want]; !ok {
			t.Errorf("vue %q absente après EnsurePlayerSchema seul (vues présentes : %v)", want, keysOf(views))
		}
	}
	byType := strings.ToLower(views["match_skill_rank_latest_by_type"])
	if !strings.Contains(byType, "partition by match_id, rating_type") {
		t.Errorf("match_skill_rank_latest_by_type doit partitionner par (match_id, rating_type) — DDL : %s", byType)
	}
	if strings.Contains(byType, "'csr'") {
		t.Errorf("match_skill_rank_latest_by_type ne doit PAS arbitrer CSR contre LUSR — DDL : %s", byType)
	}

	// Le lecteur de la page Carrière doit pouvoir interroger la vue sur une base vide.
	var n int
	if err := db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM match_skill_rank_latest_by_type WHERE rating_type <> 'LUSR_V2'`).Scan(&n); err != nil {
		t.Fatalf("lecture de match_skill_rank_latest_by_type : %v", err)
	}
	if n != 0 {
		t.Errorf("base vide : attendu 0 ligne, obtenu %d", n)
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ── Retrait des index ART secondaires sur une VRAIE player DB (plan backlog 2026-09-26,
// lot B3.9) ────────────────────────────────────────────────────────────────────────────
//
// envPlayerDBCopy désigne une COPIE de player DB, JAMAIS l'original : copie faite serveur
// arrêté, sans fichier .wal à côté de l'original, hors de l'arborescence data/titles/.
// Sans la variable, le test est sauté (gates et CI). Exemple, depuis apps/go-api :
//
//	$env:LEVELUP_B3_PLAYER_DB_COPY = "$env:TEMP\backlog-b3\stats.duckdb"
//	go test -count=1 ./internal/sync/ -run TestRetiredARTIndexes_RealPlayerDBCopy -v
const envPlayerDBCopy = "LEVELUP_B3_PLAYER_DB_COPY"

// retiredARTIndexNames : les index secondaires retirés des player DB (MSR 2026-09-27,
// PSA 2026-09-20) — noms tenus par migration.PlayerRetiredARTIndexesDropSQL.
var retiredARTIndexNames = []string{
	"idx_msr_match_lookup", "idx_msr_rating_type", "idx_msr_playlist",
	"idx_psa_match", "idx_psa_category", "idx_psa_gen",
}

// playerDBInventory : ce que la migration et le soin ne doivent PAS changer (lignes par
// table hors journal schema_migrations, vues et leurs lignes), plus les index secondaires
// retirés encore présents.
type playerDBInventory struct {
	rows    map[string]int // table → lignes
	views   map[string]int // vue → lignes
	retired []string       // index retirés présents
}

func inventoryPlayerDB(t *testing.T, db *sql.DB) playerDBInventory {
	t.Helper()
	inv := playerDBInventory{rows: map[string]int{}, views: map[string]int{}}
	count := func(kind, query string, dst map[string]int) {
		for _, n := range queryNames(t, db, query) {
			var c int
			if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM "`+n+`"`).Scan(&c); err != nil {
				t.Fatalf("lecture de la %s %s : %v", kind, n, err)
			}
			dst[n] = c
		}
	}
	count("table", `SELECT table_name FROM duckdb_tables() WHERE schema_name = 'main' AND table_name <> 'schema_migrations'`, inv.rows)
	count("vue", `SELECT view_name FROM duckdb_views() WHERE schema_name = 'main' AND NOT internal`, inv.views)
	present := map[string]bool{}
	for _, n := range queryNames(t, db, `SELECT index_name FROM duckdb_indexes() WHERE schema_name = 'main'`) {
		present[n] = true
	}
	for _, n := range retiredARTIndexNames {
		if present[n] {
			inv.retired = append(inv.retired, n)
		}
	}
	return inv
}

func queryNames(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatalf("%s : %v", query, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan : %v", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows : %v", err)
	}
	sort.Strings(out)
	return out
}

// verifyRetiredARTIndexConvergence applique à la base de `path` la chaîne de migrations
// player PUIS le soin d'ouverture, et exige : plus aucun index retiré, mêmes lignes par
// table, mêmes vues avec les mêmes lignes.
func verifyRetiredARTIndexConvergence(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s : %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })

	before := inventoryPlayerDB(t, db)
	t.Logf("AVANT : index retirés présents %v ; %d tables, %d vues ; match_skill_rank=%d, "+
		"personal_score_awards=%d, match_skill_rank_latest=%d, match_skill_rank_latest_by_type=%d",
		before.retired, len(before.rows), len(before.views), before.rows["match_skill_rank"],
		before.rows["personal_score_awards"], before.views["match_skill_rank_latest"],
		before.views["match_skill_rank_latest_by_type"])
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player) : %v", err)
	}
	if err := EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema : %v", err)
	}
	after := inventoryPlayerDB(t, db)
	t.Logf("APRÈS : index retirés présents %v ; %d tables, %d vues", after.retired, len(after.rows), len(after.views))

	if len(after.retired) > 0 {
		t.Errorf("index retirés encore présents après migrations + soin : %v", after.retired)
	}
	for _, pair := range []struct {
		kind          string
		before, after map[string]int
	}{{"table", before.rows, after.rows}, {"vue", before.views, after.views}} {
		for name, n := range pair.before {
			if m, ok := pair.after[name]; !ok || m != n {
				t.Errorf("%s %s : %d lignes avant, %d après (présente après : %v)", pair.kind, name, n, m, ok)
			}
		}
	}
}

// TestRetiredARTIndexes_RealPlayerDBCopy — B3.9 : la vérification sur une COPIE réelle.
func TestRetiredARTIndexes_RealPlayerDBCopy(t *testing.T) {
	path := os.Getenv(envPlayerDBCopy)
	if path == "" {
		t.Skipf("%s non défini : vérification sur copie réelle non demandée", envPlayerDBCopy)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("chemin %s : %v", path, err)
	}
	if strings.Contains(strings.ToLower(filepath.ToSlash(abs)), "/data/titles/") {
		t.Fatalf("REFUS : %s est dans l'arborescence data/titles/ — ce test écrit dans la base, "+
			"il ne s'applique qu'à une COPIE", abs)
	}
	if _, err := os.Stat(abs + ".wal"); err == nil {
		t.Fatalf("REFUS : %s.wal existe — la copie n'est pas cohérente sans son WAL", abs)
	}
	verifyRetiredARTIndexConvergence(t, abs)
}

// TestRetiredARTIndexes_SyntheticPreRetirementDB — le même contrôle sur une base
// antérieure au retrait fabriquée ici : chaîne migrée, les six index recréés, et le step
// MSR effacé de schema_migrations (état d'une base de prod avant le 2026-09-27). Prouve que
// la migration, pas seulement le soin, retire les index d'une base EXISTANTE.
func TestRetiredARTIndexes_SyntheticPreRetirementDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open : %v", err)
	}
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB(player) : %v", err)
	}
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_msr_match_lookup ON match_skill_rank(match_id, rating_type, written_at)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_rating_type ON match_skill_rank(rating_type)`,
		`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON match_skill_rank(playlist_group)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_match ON personal_score_awards(match_id)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category)`,
		`CREATE INDEX IF NOT EXISTS idx_psa_gen ON personal_score_awards(match_id, xuid, generation_id)`,
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, playlist_group)
		 VALUES ('m1', 'LUSR', 1200, 'arena_slayer'), ('m1', 'LUSR_V2', 1203, 'arena_slayer'),
		        ('m2', 'CSR', 1500, NULL)`,
		`DELETE FROM schema_migrations WHERE name = 'drop_msr_secondary_art_indexes_v1'`,
		`CHECKPOINT`,
	} {
		if _, err := db.ExecContext(t.Context(), stmt); err != nil {
			t.Fatalf("fabrication de la base antérieure (%.60s) : %v", stmt, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close : %v", err)
	}
	verifyRetiredARTIndexConvergence(t, path)
}
