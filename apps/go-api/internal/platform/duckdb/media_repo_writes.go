// Package duckdb - media_repo_writes.go : SetMediaMatchAssociation +
// SetMediaLike + SetMediaLikeAtomic + ToggleSharedLike + GetMediaLikers +
// queryConfig + joinStrings. Decoupe de media_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func (r *MediaRepo) SetMediaMatchAssociation(ctx context.Context, filePath, matchID string) (mapName, modeName *string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Récupérer l'id du média (match flexible : file_path exact OU basename).
	basename := filepath.Base(filePath)
	var mediaID int64
	if err := r.socialDB().QueryRow(ctx,
		`SELECT id FROM media_files WHERE file_path = ? OR file_name = ? LIMIT 1`,
		filePath, basename,
	).Scan(&mediaID); err != nil {
		return nil, nil, fmt.Errorf("SetMediaMatchAssociation: media not found: %w", err)
	}

	// ADR 0021 Phase 3.2 : route via SocialPersister (TX atomique DELETE+INSERT
	// + CHECKPOINT garanti). Fallback legacy si Persister non wired (tests).
	if r.pdb.SocialPersister != nil {
		if err := r.pdb.SocialPersister.SetMediaMatchAssociation(ctx, mediaID, matchID); err != nil {
			return nil, nil, fmt.Errorf("SetMediaMatchAssociation persister: %w", err)
		}
	} else {
		// Fallback legacy APPEND-ONLY : 1 INSERT d'event manuel dans _history (plus de
		// DELETE → ExecRecovered inutile, Exec simple suffit). is_manual=TRUE : la vue
		// donne priorité à la correction utilisateur sur l'auto.
		if _, err := r.socialDB().Exec(ctx, `INSERT INTO media_match_associations_history
			(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
			VALUES (?, ?, 0, TRUE, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, mediaID, matchID); err != nil {
			return nil, nil, fmt.Errorf("insert manual assoc event: %w", err)
		}
		_ = CheckpointSharedSocial(ctx, r.socialDB())
	}

	// récupérer map/mode du nouveau match via SharedReader.
	var mapN, pairN sql.NullString
	if db, release, err := r.pdb.SharedReadDB().Get(ctx); err == nil {
		_ = db.QueryRowContext(ctx, `
			SELECT COALESCE(r.map_name_fr, r.map_name), COALESCE(r.pair_name_fr, r.pair_name)
			FROM match_registry r WHERE r.match_id = ? LIMIT 1
		`, matchID).Scan(&mapN, &pairN)
		release()
	}
	if mapN.Valid {
		s := strings.TrimSpace(mapN.String)
		if s != "" {
			mapName = &s
		}
	}
	if pairN.Valid {
		s := strings.TrimSpace(pairN.String)
		if s != "" {
			modeName = &s
		}
	}
	return mapName, modeName, nil
}

func (r *MediaRepo) queryConfig() mediaQueryConfig {
	if r.pdb.SharedSocial != nil {
		return mediaQueryConfig{playerSlug: r.pdb.Gamertag}
	}
	return mediaQueryConfig{}
}

// SetMediaLike persiste l'état liked d'un média dans media_files.
//
// ADR 0021 Phase 3.2 : route via SocialPersister (TX atomique + CHECKPOINT
// garanti). Fallback legacy si Persister nil (tests).
//
// Note : quand WriterAcquirer est wired, le service utilise SetMediaLikeAtomic
// via LeasedWriter — chemin parallèle qui combine media_files.liked +
// media_likes (likers sociaux) en 1 TX.
func (r *MediaRepo) SetMediaLike(ctx context.Context, filePath string, liked bool) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if r.pdb.SocialPersister != nil {
		return r.pdb.SocialPersister.SetMediaLiked(ctx, filePath, liked)
	}

	// Fallback legacy : UPDATE direct + CHECKPOINT explicite.
	result, err := r.socialDB().ExecRecovered(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		return false, fmt.Errorf("SetMediaLike: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("SetMediaLike rows affected: %w", err)
	}
	_ = CheckpointSharedSocial(ctx, r.socialDB())
	return rowsAffected > 0, nil
}

// SetMediaLikeAtomic exécute en une seule transaction (si exec est un *sql.Tx)
// le UPDATE media_files.liked + l'INSERT/DELETE media_likes correspondant.
//
// Si likerSlug est vide, seul le UPDATE media_files.liked est exécuté
// (pas de ligne dans media_likes côté shared).
//
// Retourne true si la ligne media_files a été mise à jour (file_path existe).
//
// Cette méthode est l'usage canonique côté MediaService quand un
// WriterAcquirer est configuré : le service ouvre une *sql.Tx via
// LeasedWriter.BeginTx, l'injecte ici comme port.DBExecutor → atomicité.
//
// Cf. commit 6 du refactor leased-writer-enforcement (résout P3).
func (r *MediaRepo) SetMediaLikeAtomic(
	ctx context.Context,
	exec port.DBExecutor,
	filePath, likerSlug, likerGamertag string,
	liked bool,
) (bool, error) {
	result, err := exec.ExecContext(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		return false, fmt.Errorf("SetMediaLikeAtomic update: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("SetMediaLikeAtomic rows affected: %w", err)
	}
	if rowsAffected == 0 {
		slog.WarnContext(ctx, "SetMediaLikeAtomic: 0 rows — file_path absent en DB",
			"file_path", filePath, "liked", liked)
		return false, nil // file_path inconnu — caller traduit en 404 sans toucher media_likes
	}

	if likerSlug == "" {
		return true, nil
	}

	// APPEND-ONLY : like/unlike = INSERT pur d'event dans media_likes_history.
	if liked {
		_, err := exec.ExecContext(ctx, `
			INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
			VALUES (?, ?, ?, TRUE, CURRENT_TIMESTAMP)
		`, filePath, likerSlug, likerGamertag)
		if err != nil {
			return true, fmt.Errorf("SetMediaLikeAtomic add event media_likes_history: %w", err)
		}
		return true, nil
	}

	_, err = exec.ExecContext(ctx, `
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, NULL, FALSE, NULL)
	`, filePath, likerSlug)
	if err != nil {
		return true, fmt.Errorf("SetMediaLikeAtomic remove event media_likes_history: %w", err)
	}
	return true, nil
}

// ToggleSharedLike écrit ou supprime un like dans media_likes (shared DB).
//
// ADR 0022 : route via r.pdb.SocialPersister (CHECKPOINT garanti + TX
// atomique INSERT OR IGNORE — pas d'UPSERT, donc pas de pression ART).
// Fallback legacy si Persister pas wired.
func (r *MediaRepo) ToggleSharedLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string, liked bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Route nominale : Persister.
	if r.pdb.SocialPersister != nil {
		if liked {
			return r.pdb.SocialPersister.AddLike(ctx, mediaPath, likerSlug, likerGamertag)
		}
		return r.pdb.SocialPersister.RemoveLike(ctx, mediaPath, likerSlug)
	}

	// Fallback legacy (tests sans wiring). CHECKPOINT immédiat post-write
	// (ADR 0021 Phase 3.2) pour ne pas laisser de WAL non-flushé.
	// APPEND-ONLY : like/unlike = INSERT pur d'event (is_liked TRUE/FALSE) dans
	// media_likes_history. Plus de DELETE ni ON CONFLICT → surface ART éliminée
	// (Exec simple suffit). CHECKPOINT conservé (durabilité, ADR 0021 Phase 3.2).
	if liked {
		_, err := r.socialDB().Exec(ctx, `
			INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
			VALUES (?, ?, ?, TRUE, CURRENT_TIMESTAMP)
		`, mediaPath, likerSlug, likerGamertag)
		if err != nil {
			return err
		}
		_ = CheckpointSharedSocial(ctx, r.socialDB())
		return nil
	}
	_, err := r.socialDB().Exec(ctx, `
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, NULL, FALSE, NULL)
	`, mediaPath, likerSlug)
	if err != nil {
		return err
	}
	_ = CheckpointSharedSocial(ctx, r.socialDB())
	return nil
}

// GetMediaLikers retourne pour chaque media_path ses likers (max 3 noms + total).
func (r *MediaRepo) GetMediaLikers(ctx context.Context, mediaPaths []string) (map[string]domain.MediaLikersInfo, error) {
	if len(mediaPaths) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Construire le placeholder IN (?, ?, ...)
	placeholders := make([]string, len(mediaPaths))
	args := make([]any, len(mediaPaths))
	for i, p := range mediaPaths {
		placeholders[i] = "?"
		args[i] = p
	}

	// Lecture de l'état courant via la vue append-only (dernier event par
	// (media_path, liker_slug)), likers actifs uniquement (is_liked=TRUE).
	q := `SELECT media_path, liker_gamertag, ROW_NUMBER() OVER (PARTITION BY media_path ORDER BY liked_at) AS rn,
		COUNT(*) OVER (PARTITION BY media_path) AS total
	FROM media_likes_latest
	WHERE is_liked = TRUE AND media_path IN (` + joinStrings(placeholders) + `)
	ORDER BY media_path, liked_at`

	rows, err := r.socialDB().QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetMediaLikers: %w", err)
	}
	defer rows.Close()

	result := make(map[string]domain.MediaLikersInfo)
	for rows.Next() {
		var path, gamertag string
		var rn, total int
		if err := rows.Scan(&path, &gamertag, &rn, &total); err != nil {
			return nil, err
		}
		info := result[path]
		info.Total = total
		if rn <= 3 {
			info.Names = append(info.Names, gamertag)
		}
		result[path] = info
	}
	return result, rows.Err()
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
