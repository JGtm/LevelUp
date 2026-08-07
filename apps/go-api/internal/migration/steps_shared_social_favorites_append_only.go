package migration

// steps_shared_social_favorites_append_only.go — bascule match_favorites en
// APPEND-ONLY (doctrine : zéro DELETE, tout horodaté, état lu via vue _latest).
//
// AVANT : ToggleMatchFavorite retirait un favori par `DELETE FROM match_favorites`
// (persistFavorites / social_repo fallback). Un DELETE per-row retire l'entrée de
// la PK ART (player_slug, match_id) ET de idx_mfav_player → surface du bug DuckDB
// "Failed to delete all rows from index" sur shared_social.duckdb (handle RW
// PARTAGÉ process-wide concurrent = un FATAL fait tomber TOUTE l'app).
//
// APRÈS : chaque toggle = un INSERT pur dans match_favorites_history portant un flag
// is_favorite (TRUE = ajout, FALSE = retrait). PK technique id BIGINT (séquence) →
// aucune pression ART. L'état courant = dernier event par (player_slug, match_id),
// lu via la vue match_favorites_latest (QUALIFY ROW_NUMBER, comme match_skill_rank_latest).
// Plus aucun DELETE/UPDATE/ON CONFLICT → pansement ExecRecovered inutile sur ce chemin.
//
// Pattern calqué sur create_player_records_history_append_only (même DB, table sœur
// + vue + backfill, sans CTAS-swap destructif). Idempotent : no-op si _history existe.

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_favorites_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "match_favorites append-only : table _history (event is_favorite) + vue _latest + backfill — élimine le DELETE (surface ART shared_social)",
		ApplySchema: applyFavoritesAppendOnly,
	})
}

func applyFavoritesAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "match_favorites_history")
	if err != nil {
		return fmt.Errorf("favorites append-only: check history: %w", err)
	}
	if hasHistory {
		return nil // déjà migré
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS match_favorites_history_id_seq START 1`,
		`CREATE TABLE match_favorites_history (
			id           BIGINT PRIMARY KEY DEFAULT nextval('match_favorites_history_id_seq'),
			player_slug  VARCHAR NOT NULL,
			match_id     VARCHAR NOT NULL,
			is_favorite  BOOLEAN NOT NULL,
			favorited_at TIMESTAMP,
			written_at   TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)`,
		// Index secondaire NON NULL-bearing (player_slug + match_id NOT NULL),
		// alimenté uniquement par INSERT → jamais de retrait/relocation d'entrée ART.
		`CREATE INDEX IF NOT EXISTS idx_mfh_lookup ON match_favorites_history(player_slug, match_id, written_at DESC)`,
		`CREATE OR REPLACE VIEW match_favorites_latest AS
			SELECT id, player_slug, match_id, is_favorite, favorited_at, written_at
			FROM match_favorites_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY player_slug, match_id
				ORDER BY written_at DESC, id DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("favorites append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	// Backfill : toutes les lignes legacy = favoris actifs → un event is_favorite=TRUE.
	if hasLegacy, _ := tableExists(db, "match_favorites"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO match_favorites_history (player_slug, match_id, is_favorite, favorited_at, written_at)
			SELECT player_slug, match_id, TRUE, favorited_at, COALESCE(favorited_at, CURRENT_TIMESTAMP)
			FROM match_favorites`); err != nil {
			return fmt.Errorf("favorites append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_favorites_history`).Scan(&n)
	slog.InfoContext(ctx, "match_favorites append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
