package migrations

// steps_shared_widen_scores.go — match_registry.team_{0,1}_score : SMALLINT -> INTEGER.
//
// Derive de schema constatee le 2026-09-16 : la DDL du code declare INTEGER, les bases reelles
// (creees avant) portaient SMALLINT, et un score d'equipe > 32 767 (Bapteme du feu) faisait
// REJETER le match a l'INSERT — pour tous les joueurs. Fichier dedie : steps_shared_core.go
// depasse deja le seuil de 500 lignes (dette gelee, ne pas l'accroitre).
//
// L'ALTER COLUMN de DuckDB 1.5.5 echoue (« Cannot alter entry ... there are entries that depend
// on it ») des qu'un index SECONDAIRE existe sur la table, meme sur une autre colonne — et
// match_registry en porte (idx_mr_start_time, idx_match_registry_season_id). Le helper
// migration.AlterColumnTypeIfNeeded depose ces index (DDL relevee dans duckdb_indexes()),
// elargit, puis les recree. Revue adversariale du 2026-09-16 (P0) : la premiere version ne le
// faisait pas et aurait echoue a chaque boot, bloquant en cascade les migrations pve/social.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

var widenMatchRegistryTeamScoresStep = migration.Migration{
	Name:        "widen_match_registry_team_scores",
	TargetDB:    migration.TargetShared,
	Description: "match_registry.team_{0,1}_score : SMALLINT -> INTEGER (idempotent, index secondaires deposes puis recrees) — un score d'equipe > 32 767 (Bapteme du feu) faisait REJETER le match a l'INSERT, pour tous les joueurs",
	ApplySchema: widenMatchRegistryTeamScores,
}

func widenMatchRegistryTeamScores(db *sql.DB) error {
	existe, err := migration.TableExists(db, "match_registry")
	if err != nil {
		return fmt.Errorf("widen_match_registry_team_scores: %w", err)
	}
	if !existe {
		return nil
	}
	var elargies []string
	for _, colonne := range []string{"team_0_score", "team_1_score"} {
		change, err := migration.AlterColumnTypeIfNeeded(db, "match_registry", colonne, "INTEGER")
		if err != nil {
			return fmt.Errorf("widen_match_registry_team_scores: %w", err)
		}
		if change {
			elargies = append(elargies, colonne)
		}
	}
	if len(elargies) > 0 {
		slog.InfoContext(migration.BootCtx(), "migration: match_registry scores d'equipe elargis en INTEGER",
			"colonnes", elargies)
	}
	return nil
}
