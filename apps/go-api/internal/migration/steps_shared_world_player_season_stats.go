package migration

// steps_shared_world_player_season_stats.go — stats agrégées par joueur du
// classement mondial, par saison CSR et par playlist (Phase B du plan
// PLAN_WORLD_LEADERBOARD_ENRICHED.md).
//
// Alimentée par l'agrégateur (Phase C) + le backfill (Phase D) qui paginent
// l'historique des joueurs du top-100 (résolus via PeopleHub), bucketent leurs
// matchs par (season_id, playlist_id) et somment les compteurs bruts.
//
// Conception append-only (règle ART, cf. ADR 0019 + CLAUDE.md ; identique à
// world_csr_leaderboard_snapshots) :
//   - PK technique `id` (séquence) + colonne `written_at`.
//   - Toute écriture = INSERT pur (jamais UPDATE/UPSERT) → pas de bug ART.
//   - Lecture via la vue `world_player_season_stats_latest` (dernière ligne par
//     (title_slug, gamertag, season_id, playlist_id)).
//
// Compteurs BRUTS uniquement : win_rate / kda / *_per_min sont DÉRIVÉS au moment
// de la lecture (query layer ou API), jamais stockés (cf. plan § schéma).
//
// Attribution (Phase A) : season_id = saison CSR du match (via MatchInfo.SeasonId,
// format Csr/Seasons/CsrSeasonX-Y), playlist_id = MatchInfo.Playlist.AssetId.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_world_player_season_stats",
		TargetDB:    TargetShared,
		Description: "Crée world_player_season_stats (append-only) + vue _latest : stats joueur du classement mondial par saison CSR x playlist (compteurs bruts, ratios dérivés à la lecture)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE SEQUENCE IF NOT EXISTS wpss_seq START 1;

				CREATE TABLE IF NOT EXISTS world_player_season_stats (
					id           BIGINT PRIMARY KEY DEFAULT nextval('wpss_seq'),
					title_slug   VARCHAR NOT NULL DEFAULT 'halo_infinite',
					gamertag     VARCHAR NOT NULL,
					season_id    VARCHAR NOT NULL,
					playlist_id  VARCHAR NOT NULL,
					match_count  INTEGER NOT NULL DEFAULT 0,
					win_count    INTEGER NOT NULL DEFAULT 0,
					loss_count   INTEGER NOT NULL DEFAULT 0,
					tie_count    INTEGER NOT NULL DEFAULT 0,
					dnf_count    INTEGER NOT NULL DEFAULT 0,
					kills        BIGINT NOT NULL DEFAULT 0,
					deaths       BIGINT NOT NULL DEFAULT 0,
					assists      BIGINT NOT NULL DEFAULT 0,
					playtime_s   BIGINT NOT NULL DEFAULT 0,
					medal_count  BIGINT NOT NULL DEFAULT 0,
					computed_at  TIMESTAMP,
					written_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX IF NOT EXISTS idx_wpss_lookup
					ON world_player_season_stats(title_slug, season_id, playlist_id, gamertag, written_at);

				CREATE OR REPLACE VIEW world_player_season_stats_latest AS
					SELECT *
					FROM world_player_season_stats
					QUALIFY ROW_NUMBER() OVER (
						PARTITION BY title_slug, gamertag, season_id, playlist_id
						ORDER BY written_at DESC, id DESC
					) = 1;
			`)
		},
	})
}
