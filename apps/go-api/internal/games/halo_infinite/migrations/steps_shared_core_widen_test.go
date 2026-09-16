//go:build cgo

package migrations

// steps_shared_core_widen_test.go — LA DÉRIVE DE SCHÉMA DE match_registry, ET SA RÉPARATION.
//
// CE QUE CE TEST PROTÈGE (mesuré le 2026-09-16). La DDL du code déclarait
// `team_0_score / team_1_score INTEGER` ; les bases de production, créées par une DDL
// antérieure, portaient SMALLINT — et `CREATE TABLE IF NOT EXISTS` ne corrige jamais une
// colonne existante. Deux matchs de Baptême du feu ont été REJETÉS à l'INSERT
// (`Type INT64 with value 120267 ... INT16`) : un score d'équipe dépasse 32 767. Un match
// rejeté du registre est perdu POUR TOUS LES JOUEURS, pas seulement pour celui qui
// synchronisait.
//
// Le test construit la table avec la DDL LEGACY (SMALLINT + PK sur match_id, exactement la
// forme des bases réelles), joue l'étape, et exige : INTEGER × 2, lignes intactes, INSERT à
// 120 267 accepté, second passage no-op.
//
// AUCUNE BASE SOUS data/ N'EST OUVERTE : tout se joue en `:memory:`.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ddlLegacyMatchRegistry — la forme réelle des bases d'avant la correction : PK sur match_id
// (index ART) et scores d'équipe en SMALLINT.
const ddlLegacyMatchRegistry = `
CREATE TABLE match_registry (
	match_id VARCHAR PRIMARY KEY,
	start_time TIMESTAMP,
	team_0_score SMALLINT,
	team_1_score SMALLINT,
	team_0_ps_score INTEGER,
	team_1_ps_score INTEGER,
	player_count SMALLINT DEFAULT 0
)`

func typeColonne(t *testing.T, db *sql.DB, table, colonne string) string {
	t.Helper()
	var declare string
	err := db.QueryRowContext(context.Background(),
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = ? AND column_name = ?",
		table, colonne).Scan(&declare)
	if err != nil {
		t.Fatalf("lecture du type de %s.%s : %v", table, colonne, err)
	}
	return declare
}

func TestWidenMatchRegistryTeamScores(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("ouverture duckdb mémoire : %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, ddlLegacyMatchRegistry); err != nil {
		t.Fatalf("DDL legacy : %v", err)
	}
	// La FORME REELLE de la table : un index secondaire (add_shared_performance_indexes pose
	// idx_mr_start_time). Sans lui ce test etait vert alors que l'ALTER echouait en prod
	// (revue adversariale du 2026-09-16, P0).
	if _, err := db.ExecContext(ctx, "CREATE INDEX idx_mr_start_time ON match_registry(start_time)"); err != nil {
		t.Fatalf("index secondaire : %v", err)
	}
	for _, id := range []string{"match-a", "match-b", "match-c"} {
		if _, err := db.ExecContext(ctx,
			"INSERT INTO match_registry (match_id, team_0_score, team_1_score) VALUES (?, ?, ?)",
			id, 50, 42); err != nil {
			t.Fatalf("insertion %s : %v", id, err)
		}
	}
	// Contrôle négatif : avant la migration, le gros score est REJETÉ. Si ce rejet
	// n'arrivait plus, le test ne prouverait rien.
	if _, err := db.ExecContext(ctx,
		"INSERT INTO match_registry (match_id, team_0_score, team_1_score) VALUES (?, ?, ?)",
		"match-firefight", 120267, 0); err == nil {
		t.Fatal("la DDL legacy a accepté 120267 en SMALLINT — la fixture ne reproduit plus la dérive")
	}

	if err := widenMatchRegistryTeamScores(db); err != nil {
		t.Fatalf("migration : %v", err)
	}

	for _, colonne := range []string{"team_0_score", "team_1_score"} {
		if got := typeColonne(t, db, "match_registry", colonne); got != "INTEGER" {
			t.Errorf("%s = %s après migration, attendu INTEGER", colonne, got)
		}
	}

	var nbIndex int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'match_registry' AND index_name = 'idx_mr_start_time'").
		Scan(&nbIndex); err != nil || nbIndex != 1 {
		t.Fatalf("idx_mr_start_time absent apres migration (n=%d, err=%v) : l'index doit etre recree", nbIndex, err)
	}

	var lignes int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&lignes); err != nil {
		t.Fatalf("comptage : %v", err)
	}
	if lignes != 3 {
		t.Errorf("lignes = %d après migration, attendu 3 (aucune perte)", lignes)
	}
	var score0, score1 int
	if err := db.QueryRowContext(ctx,
		"SELECT team_0_score, team_1_score FROM match_registry WHERE match_id = 'match-a'").
		Scan(&score0, &score1); err != nil {
		t.Fatalf("relecture match-a : %v", err)
	}
	if score0 != 50 || score1 != 42 {
		t.Errorf("match-a = (%d, %d), attendu (50, 42) — valeurs altérées par l'élargissement", score0, score1)
	}

	// Le match qui était rejeté passe maintenant.
	if _, err := db.ExecContext(ctx,
		"INSERT INTO match_registry (match_id, team_0_score, team_1_score) VALUES (?, ?, ?)",
		"match-firefight", 120267, 0); err != nil {
		t.Fatalf("insertion d'un score d'équipe > 32 767 après migration : %v", err)
	}

	// Idempotence : second passage = no-op, types et lignes inchangés.
	if err := widenMatchRegistryTeamScores(db); err != nil {
		t.Fatalf("second passage de la migration : %v", err)
	}
	if got := typeColonne(t, db, "match_registry", "team_0_score"); got != "INTEGER" {
		t.Errorf("team_0_score = %s après second passage, attendu INTEGER", got)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&lignes); err != nil {
		t.Fatalf("comptage après second passage : %v", err)
	}
	if lignes != 4 {
		t.Errorf("lignes = %d après second passage, attendu 4", lignes)
	}
}

// TestWidenMatchRegistryTeamScores_TableAbsente — la migration tourne aussi sur une base qui
// n'a pas encore match_registry (ordre des étapes, base neuve) : no-op, pas d'erreur.
func TestWidenMatchRegistryTeamScores_TableAbsente(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("ouverture duckdb mémoire : %v", err)
	}
	defer db.Close()

	if err := widenMatchRegistryTeamScores(db); err != nil {
		t.Errorf("migration sur base sans match_registry : %v, attendu nil", err)
	}
}
