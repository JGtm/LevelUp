//go:build cgo

package migrations

// steps_player_drop_player_secondary_indexes_test.go — verrouille le retrait des quatre
// derniers index secondaires des tables append-only joueur (idx_lch_component,
// idx_lch_match, idx_pme_match_lookup, idx_pcs_lookup), plan des recommandations du
// 2026-10-09, lot C2. Mesure : migration/psa_index_repro_player_planprobe_test.go (tag
// psarepro) — aucune lecture de production ne les emprunte, aucune ne ralentit sans eux.

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/migration"
)

const dropPlayerSecondaryStepName = "drop_player_secondary_art_indexes_v1"

// playerSecondaryLegacyIndexDDL : les quatre index tels que les posaient les autorités
// avant le 2026-10-09 — l'état d'une player DB de prod, ou rouverte par un binaire ancien.
var playerSecondaryLegacyIndexDDL = []string{
	`CREATE INDEX IF NOT EXISTS idx_lch_component ON lusr_component_history(component_name)`,
	`CREATE INDEX IF NOT EXISTS idx_lch_match ON lusr_component_history(match_id)`,
	`CREATE INDEX IF NOT EXISTS idx_pme_match_lookup ON player_match_enrichment(match_id, written_at)`,
	`CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at)`,
}

func playerSecondaryIndexCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name IN
		('lusr_component_history', 'player_match_enrichment', 'player_csr_snapshots')`).Scan(&n); err != nil {
		t.Fatalf("duckdb_indexes(): %v", err)
	}
	return n
}

func applyDropPlayerSecondaryIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, m := range migration.All() {
		if m.Name != dropPlayerSecondaryStepName {
			continue
		}
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("%s: %v", dropPlayerSecondaryStepName, err)
		}
		return
	}
	t.Fatalf("step %s absent du registre global", dropPlayerSecondaryStepName)
}

// TestPlayerChainNeCreeAucunIndexSecondaireAppendOnly : une player DB NEUVE, construite
// par la seule chaîne de migrations, ne porte aucun index secondaire sur les trois tables.
func TestPlayerChainNeCreeAucunIndexSecondaireAppendOnly(t *testing.T) {
	db := migratedPlayerDBFile(t)
	if n := playerSecondaryIndexCount(t, db); n != 0 {
		t.Errorf("%d index secondaire(s) sur lusr_component_history / player_match_enrichment / "+
			"player_csr_snapshots d'une player DB fraîchement migrée, attendu 0", n)
	}
}

// TestDropPlayerSecondaryARTIndexes_RetireEtPreserve : le step retire les quatre index
// d'une DB EXISTANTE qui les porte, ne touche ni aux lignes ni aux vues, laisse un INSERT
// sans id fonctionner, et un second passage ne casse pas.
func TestDropPlayerSecondaryARTIndexes_RetireEtPreserve(t *testing.T) {
	db := migratedPlayerDBFile(t)
	for _, stmt := range playerSecondaryLegacyIndexDDL {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("pose de l'index legacy: %v", err)
		}
	}
	if n := playerSecondaryIndexCount(t, db); n != 4 {
		t.Fatalf("préalable : %d index posés, attendu 4 — le test ne mordrait pas", n)
	}
	for _, q := range []string{
		`INSERT INTO lusr_component_history (match_id, component_name, value, weight) VALUES
			('m1', 'kda', 0.5, 1), ('m1', 'kda', 0.6, 1), ('m2', 'acc', 0.4, 1)`,
		`INSERT INTO player_match_enrichment (match_id, stage, performance_score) VALUES
			('m1', 'live', 50), ('m1', 'perf', 60), ('m2', 'live', 40)`,
		`INSERT INTO player_csr_snapshots (playlist_id, season_id, current_value) VALUES
			('p1', 's1', 1200), ('p1', 's1', 1250), ('p2', 's1', 900)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	read := func() map[string]int {
		out := map[string]int{}
		for _, rel := range []string{
			"lusr_component_history", "lusr_component_history_latest",
			"player_match_enrichment", "player_match_enrichment_latest",
			"player_csr_snapshots", "player_csr_snapshots_latest",
		} {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + rel).Scan(&n); err != nil {
				t.Fatalf("%s: %v", rel, err)
			}
			out[rel] = n
		}
		return out
	}
	before := read()

	applyDropPlayerSecondaryIndexes(t, db)
	if n := playerSecondaryIndexCount(t, db); n != 0 {
		t.Errorf("%d index toujours présents après %s", n, dropPlayerSecondaryStepName)
	}
	after := read()
	for rel, n := range before {
		if after[rel] != n {
			t.Errorf("%s : %d lignes avant, %d après (le step ne touche que la DDL d'index)", rel, n, after[rel])
		}
	}
	if _, err := db.Exec(`INSERT INTO player_match_enrichment (match_id, stage) VALUES ('m3', 'live')`); err != nil {
		t.Errorf("INSERT sans id après le step (PK/séquence intactes ?): %v", err)
	}
	applyDropPlayerSecondaryIndexes(t, db) // idempotence
}
