package migration

// steps_shared_social.go — migrations ciblant shared_social.duckdb.
//
// Cette base centralise les données utilisateur/social qui n'appartiennent
// ni aux stats de match (shared_matches_v2) ni aux enrichissements joueur (stats.duckdb) :
//   - media_files       : fichiers médias de tous les joueurs
//   - media_match_assoc : associations média↔match
//   - media_likes       : likes sociaux sur les médias
//   - match_favorites   : matchs mis en favoris par joueur

import "database/sql"

func init() {
	Register(Migration{
		Name:        "create_base_shared_social_schema",
		TargetDB:    TargetSharedSocial,
		Description: "Tables de base shared_social.duckdb : médias, likes, favoris",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				CREATE TABLE IF NOT EXISTS media_files (
					id                   VARCHAR PRIMARY KEY,
					player_slug          VARCHAR NOT NULL,
					file_path            VARCHAR NOT NULL UNIQUE,
					file_name            VARCHAR NOT NULL,
					kind                 VARCHAR NOT NULL DEFAULT 'video',
					file_hash            VARCHAR,
					file_size            INTEGER DEFAULT 0,
					thumbnail_path       VARCHAR,
					capture_end_utc      TIMESTAMP,
					discord_notified_at  TIMESTAMP,
					liked                BOOLEAN DEFAULT FALSE,
					liked_at             TIMESTAMP,
					created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_mf_player_slug ON media_files(player_slug);
				CREATE INDEX IF NOT EXISTS idx_mf_kind ON media_files(kind);
				CREATE INDEX IF NOT EXISTS idx_mf_created ON media_files(created_at);

				CREATE TABLE IF NOT EXISTS media_match_associations (
					media_file_id  VARCHAR NOT NULL,
					match_id       VARCHAR NOT NULL,
					delta_seconds  INTEGER,
					created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (media_file_id, match_id)
				);
				CREATE INDEX IF NOT EXISTS idx_mma_match ON media_match_associations(match_id);

				CREATE TABLE IF NOT EXISTS media_likes (
					media_path      VARCHAR NOT NULL,
					liker_slug      VARCHAR NOT NULL,
					liker_gamertag  VARCHAR,
					liked_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (media_path, liker_slug)
				);
				CREATE INDEX IF NOT EXISTS idx_ml_media_path ON media_likes(media_path);

				CREATE TABLE IF NOT EXISTS match_favorites (
					player_slug   VARCHAR NOT NULL,
					match_id      VARCHAR NOT NULL,
					favorited_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (player_slug, match_id)
				);
				CREATE INDEX IF NOT EXISTS idx_mfav_player ON match_favorites(player_slug);
			`)
		},
	})
}
