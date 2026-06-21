// Package sync — aggregates.go : rafraîchissement des vues matérialisées.
//
// Portage de src/data/sync/_aggregates.py.
// Recrée (DROP+CREATE) les vues matérialisées dans la player DB
// et les vues SQL garanties dans la shared DB.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

// MaterializedView décrit une vue matérialisée à recréer.
type MaterializedView struct {
	Name  string
	Query string
}

// playerMaterializedViews contient les vues matérialisées joueur (mv_*).
var playerMaterializedViews = []MaterializedView{
	{
		Name: "mv_player_matches",
		Query: `
		SELECT
			pme.match_id,
			pme.performance_score,
			pme.session_id,
			pme.session_label,
			pme.is_with_friends
		FROM player_match_enrichment_latest pme
		WHERE pme.performance_score IS NOT NULL`,
	},
	{
		Name: "mv_map_stats",
		Query: `
		SELECT
			msr.playlist_group,
			COUNT(*) AS match_count,
			AVG(msr.rating_value) AS avg_rating,
			MAX(msr.rating_value) AS peak_rating
		FROM match_skill_rank msr
		WHERE msr.rating_type = 'LUSR'
		GROUP BY msr.playlist_group`,
	},
}

// refreshAggregates recrée les vues matérialisées dans la player DB.
// Best-effort : tente toutes les vues même si l'une échoue. Retourne
// (created, failed, err) où err = errors.Join des échecs par-vue (nil si aucun
// échec) — permet au caller de loguer un warn agrégé corrélé au lieu de perdre
// les warns par-vue.
func refreshAggregates(ctx context.Context, playerDB *sql.DB) (created, failed int, err error) {
	var viewErrs []error
	for _, mv := range playerMaterializedViews {
		if verr := recreateMaterializedView(ctx, playerDB, mv); verr != nil {
			slog.WarnContext(ctx, "aggregates: échec vue matérialisée", "view", mv.Name, "err", verr)
			viewErrs = append(viewErrs, fmt.Errorf("%s: %w", mv.Name, verr))
			continue
		}
		created++
	}
	return created, len(viewErrs), errors.Join(viewErrs...)
}

// RefreshAggregates est l'export public de refreshAggregates pour les
// callers hors-package qui doivent rebuild mv_player_matches après un UPDATE
// direct (cf. friends_recompute.go §4 plan Squad/Sessions).
func RefreshAggregates(ctx context.Context, playerDB *sql.DB) (created, failed int, err error) {
	return refreshAggregates(ctx, playerDB)
}

// recreateMaterializedView exécute DROP TABLE IF EXISTS + CREATE TABLE AS SELECT.
func recreateMaterializedView(ctx context.Context, db *sql.DB, mv MaterializedView) error {
	drop := fmt.Sprintf("DROP TABLE IF EXISTS %s", mv.Name)
	if _, err := db.ExecContext(ctx, drop); err != nil {
		return fmt.Errorf("drop %s: %w", mv.Name, err)
	}
	create := fmt.Sprintf("CREATE TABLE %s AS %s", mv.Name, mv.Query)
	if _, err := db.ExecContext(ctx, create); err != nil {
		return fmt.Errorf("create %s: %w", mv.Name, err)
	}
	return nil
}
