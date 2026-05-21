// Package sync — aggregates.go : rafraîchissement des vues matérialisées.
//
// Portage de src/data/sync/_aggregates.py.
// Recrée (DROP+CREATE) les vues matérialisées dans la player DB
// et les vues SQL garanties dans la shared DB.
package sync

import (
	"context"
	"database/sql"
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
		FROM player_match_enrichment pme
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

// sharedSQLViews contient les vues SQL à recréer dans shared DB.
var sharedSQLViews = []MaterializedView{
	{
		Name: "v_gamertag_lookup",
		Query: `
		SELECT
			COALESCE(xa.xuid, mp.xuid) AS xuid,
			COALESCE(xa.gamertag, mp.gamertag) AS gamertag,
			xa.last_seen
		FROM xuid_aliases xa
		FULL OUTER JOIN (
			SELECT DISTINCT xuid, gamertag
			FROM match_participants
			WHERE gamertag IS NOT NULL
		) mp ON xa.xuid = mp.xuid`,
	},
	{
		Name: "v_match_full",
		Query: `
		SELECT
			mr.*
		FROM match_registry mr`,
	},
}

// refreshAggregates recrée les vues matérialisées dans la player DB.
// Retourne le nombre de vues créées.
func refreshAggregates(ctx context.Context, playerDB *sql.DB) (int, error) { //nolint:unparam // error toujours nil, conservé pour évolution
	count := 0
	for _, mv := range playerMaterializedViews {
		if err := recreateMaterializedView(ctx, playerDB, mv); err != nil {
			slog.Warn("aggregates: échec vue matérialisée", "view", mv.Name, "err", err)
			continue
		}
		count++
	}
	return count, nil
}

// RefreshAggregates est l'export public de refreshAggregates pour les
// callers hors-package qui doivent rebuild mv_player_matches après un UPDATE
// direct (cf. friends_recompute.go §4 plan Squad/Sessions).
func RefreshAggregates(ctx context.Context, playerDB *sql.DB) (int, error) {
	return refreshAggregates(ctx, playerDB)
}

// refreshSharedViews recrée les vues SQL dans la shared DB (idempotent).
func refreshSharedViews(ctx context.Context, sharedDB *sql.DB) (int, error) { //nolint:unparam // error toujours nil, conservé pour évolution
	count := 0
	for _, v := range sharedSQLViews {
		if err := recreateSQLView(ctx, sharedDB, v); err != nil {
			slog.Warn("aggregates: échec shared view", "view", v.Name, "err", err)
			continue
		}
		count++
	}
	return count, nil
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

// recreateSQLView exécute CREATE OR REPLACE VIEW.
func recreateSQLView(ctx context.Context, db *sql.DB, v MaterializedView) error {
	stmt := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s", v.Name, v.Query)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("view %s: %w", v.Name, err)
	}
	return nil
}
