package migration

// steps_shared_social_likes_append_only.go — bascule media_likes en APPEND-ONLY.
//
// AVANT : ToggleSharedLike / SetMediaLikeAtomic / persistLikes retiraient un like par
// `DELETE FROM media_likes` et ajoutaient par `INSERT … ON CONFLICT DO UPDATE` — deux
// surfaces ART sur shared_social.duckdb (handle RW PARTAGÉ concurrent : un FATAL fait
// tomber toute l'app).
//
// APRÈS : chaque like/unlike = un INSERT pur d'event dans media_likes_history
// (is_liked TRUE/FALSE). PK technique id BIGINT → aucune pression ART. État courant lu
// via media_likes_latest (dernier event par (media_path, liker_slug), filtré is_liked=TRUE).
//
// NB (2026-08-04) : media_files.liked / liked_at, l'ancien support GLOBAL du cœur,
// ont été retirées du schéma — le like est PAR LIKER et n'a plus d'autre support que
// media_likes_history. Cf. drop_media_files_liked_columns_v1
// (games/halo_infinite/migrations/steps_shared_social.go).
//
// Pattern calqué sur match_favorites (table sœur _history + vue + backfill, sans CTAS).

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func init() {
	Register(Migration{
		Name:        "shared_social_likes_append_only_v1",
		TargetDB:    TargetSharedSocial,
		Description: "media_likes append-only : table _history (event is_liked) + vue _latest + backfill — élimine DELETE + ON CONFLICT (surface ART shared_social)",
		ApplySchema: applyLikesAppendOnly,
	})
}

func applyLikesAppendOnly(db *sql.DB) error {
	ctx := bootCtx()

	hasHistory, err := tableExists(db, "media_likes_history")
	if err != nil {
		return fmt.Errorf("likes append-only: check history: %w", err)
	}
	if hasHistory {
		return nil
	}

	stmts := []string{
		`CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1`,
		`CREATE TABLE media_likes_history (
			id             BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path     VARCHAR NOT NULL,
			liker_slug     VARCHAR NOT NULL,
			liker_gamertag VARCHAR,
			is_liked       BOOLEAN NOT NULL,
			liked_at       TIMESTAMP,
			written_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mlh_lookup ON media_likes_history(media_path, liker_slug, written_at DESC)`,
		`CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (
				PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC
			) = 1`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("likes append-only: step (%s): %w", firstWords(s, 3), err)
		}
	}

	// Backfill : lignes legacy = likes actifs → event is_liked=TRUE.
	if hasLegacy, _ := tableExists(db, "media_likes"); hasLegacy {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at)
			SELECT media_path, liker_slug, liker_gamertag, TRUE, liked_at, COALESCE(liked_at, CURRENT_TIMESTAMP)
			FROM media_likes`); err != nil {
			return fmt.Errorf("likes append-only: backfill: %w", err)
		}
	}

	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_likes_history`).Scan(&n)
	slog.InfoContext(ctx, "media_likes append-only: migration appliquée", "rows_backfilled", n)
	return nil
}
