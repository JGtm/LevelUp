package migration

// steps_world_csr_leaderboard.go — table de snapshots du classement CSR mondial.
//
// Source : pages publiques Halo Waypoint
//   https://www.halowaypoint.com/halo-infinite/leaderboards/{seasonId}/{playlistId}?page=N
// scrapées par le job cmd/snapshot-world-leaderboard (cf. plan Classement).
//
// Conception append-only dès la création (règle ART, cf. ADR 0019 + CLAUDE.md) :
//   - PK technique `id` (séquence) + colonne `written_at`.
//   - Toute écriture est un INSERT pur (jamais UPDATE/UPSERT).
//   - La lecture courante passe par la vue `world_csr_leaderboard_latest`
//     (dernier snapshot par (season_id, playlist_id, rank)).
//
// Le tier n'est pas publié par Halo Waypoint (seul le CSR numérique l'est) :
// `tier_derived` est calculé côté Go au moment du snapshot à partir de csr_value.

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_world_csr_leaderboard_snapshots",
		TargetDB:    TargetShared,
		Description: "Crée world_csr_leaderboard_snapshots (append-only) + vue _latest pour le classement CSR mondial scrapé depuis Halo Waypoint",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE SEQUENCE IF NOT EXISTS wcl_seq START 1;

				CREATE TABLE IF NOT EXISTS world_csr_leaderboard_snapshots (
					id            BIGINT PRIMARY KEY DEFAULT nextval('wcl_seq'),
					season_id     VARCHAR NOT NULL,
					playlist_id   VARCHAR NOT NULL,
					rank          INTEGER NOT NULL,
					gamertag      VARCHAR NOT NULL,
					csr_value     INTEGER NOT NULL,
					tier_derived  VARCHAR,
					fetched_at    TIMESTAMP,
					written_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);

				CREATE INDEX IF NOT EXISTS idx_wcl_lookup
					ON world_csr_leaderboard_snapshots(season_id, playlist_id, rank, written_at);

				CREATE OR REPLACE VIEW world_csr_leaderboard_latest AS
					SELECT *
					FROM world_csr_leaderboard_snapshots
					QUALIFY ROW_NUMBER() OVER (
						PARTITION BY season_id, playlist_id, rank
						ORDER BY written_at DESC, id DESC
					) = 1;
			`)
		},
	})
}
