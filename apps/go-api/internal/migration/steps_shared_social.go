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

	// Ajout player_slug à media_files Go (schéma créé par ensureMediaTables sans cette colonne).
	Register(Migration{
		Name:        "add_player_slug_to_media_files",
		TargetDB:    TargetSharedSocial,
		Description: "Ajoute player_slug à media_files si absente (schéma Go ops/media.go)",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `ALTER TABLE media_files ADD COLUMN IF NOT EXISTS player_slug VARCHAR;`)
		},
	})

	// Ajout file_name à media_files Go.
	Register(Migration{
		Name:        "add_file_name_to_media_files",
		TargetDB:    TargetSharedSocial,
		Description: "Ajoute file_name à media_files si absente",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `ALTER TABLE media_files ADD COLUMN IF NOT EXISTS file_name VARCHAR;`)
		},
	})

	// Ajout thumbnail_path, capture_end_utc, status, mtime à media_files Go.
	Register(Migration{
		Name:        "add_missing_columns_to_media_files",
		TargetDB:    TargetSharedSocial,
		Description: "Ajoute thumbnail_path, capture_end_utc, status, mtime à media_files si absentes",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE media_files ADD COLUMN IF NOT EXISTS thumbnail_path VARCHAR;
				ALTER TABLE media_files ADD COLUMN IF NOT EXISTS capture_end_utc TIMESTAMPTZ;
				ALTER TABLE media_files ADD COLUMN IF NOT EXISTS status VARCHAR;
				ALTER TABLE media_files ADD COLUMN IF NOT EXISTS mtime TIMESTAMPTZ;
			`)
		},
	})

	// Sprint 2026-04 : flag is_manual pour distinguer associations auto vs réassociées
	// manuellement par l'utilisateur. Permet de préserver les corrections lors d'un
	// reassociate global (DELETE WHERE NOT is_manual).
	Register(Migration{
		Name:        "add_is_manual_to_media_match_associations",
		TargetDB:    TargetSharedSocial,
		Description: "Ajoute is_manual BOOLEAN à media_match_associations pour tracer les réassociations manuelles",
		ApplySchema: func(db *sql.DB) error {
			return execScript(db, `
				ALTER TABLE media_match_associations ADD COLUMN IF NOT EXISTS is_manual BOOLEAN DEFAULT FALSE;
			`)
		},
	})
}
