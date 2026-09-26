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
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
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
