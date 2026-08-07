// Package duckdb - media_repo_writes.go : SetMediaMatchAssociation +
// MediaExists + SetMediaLikeAtomic + ToggleSharedLike + GetMediaLikers +
// queryConfig + joinStrings. Decoupe de media_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
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
		`SELECT id FROM media_files
		 WHERE (file_path = ? OR file_name = ?) AND `+MediaVisiblePredicate("")+`
		 LIMIT 1`,
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
			VALUES (?, ?, 0, TRUE, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`, mediaID, matchID); err != nil {
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

// queryConfig porte le scoping player_slug du pipeline média Q37. Appelée
// uniquement depuis loadMediaCandidates, DERRIÈRE ses deux gardes
// (pdb.SharedSocial != nil et pdb.Gamertag != "") — le repli « config vide »
// (schéma legacy) a été supprimé le 2026-08-03 en même temps que la branche SQL
// legacy qu'il servait à sélectionner ; c'est le garde Gamertag du caller qui
// couvre désormais le cas d'un PlayerDB sans gamertag.
func (r *MediaRepo) queryConfig() mediaQueryConfig {
	return mediaQueryConfig{playerSlug: r.pdb.Gamertag, viewerSlug: r.viewer()}
}

// qMediaVisibleExists : lookup d'existence d'un média VISIBLE par son file_path.
// Partagé par MediaExists (chemin legacy) et SetMediaLikeAtomic (dans la TX) —
// les deux doivent répondre pareil, sinon le chemin atomique et le chemin legacy
// divergeraient sur ce qui vaut 404.
var qMediaVisibleExists = `SELECT 1 FROM media_files
	WHERE file_path = ? AND ` + MediaVisiblePredicate("") + ` LIMIT 1`

// MediaExists indique si le média existe ET est visible (non supprimé).
//
// Cette méthode a remplacé SetMediaLike le 2026-08-04 : `media_files.liked`
// était une colonne GLOBALE (un seul booléen partagé par tous les viewers), donc
// le like d'un coéquipier allumait le cœur de tout le monde. L'état du like vit
// désormais UNIQUEMENT dans media_likes_history / media_likes_latest, par liker.
// Ce qui restait de l'ancien UPDATE — la détection « file_path inconnu → 404 » —
// est tout ce que le caller a encore besoin de savoir avant d'écrire l'event.
func (r *MediaRepo) MediaExists(ctx context.Context, filePath string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.socialDB().QueryRowRecovered(ctx, qMediaVisibleExists, filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("MediaExists: %w", err)
	}
	defer rows.Close()
	return true, nil
}

// SetMediaLikeAtomic écrit l'event de like DU LIKER dans media_likes_history
// (append-only), après avoir vérifié que le média existe et est visible — le
// tout dans la transaction fournie par le caller (si exec est un *sql.Tx).
//
// Retourne false si le file_path est inconnu ou supprimé : le caller traduit en
// 404 et AUCUN event n'est écrit (media_likes_history est append-only — un event
// inséré par erreur ne pourrait plus être retiré).
//
// Si likerSlug est vide, rien n'est écrit : depuis le passage du like au
// par-viewer (2026-08-04) un like sans liker n'a plus aucun support de stockage
// (media_files.liked, le support global, a été droppée du schéma). Le handler
// garantit un liker non vide (session, ou propriétaire de la page à défaut).
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
	// Vérification d'existence DANS la transaction : elle remplace le rowsAffected
	// de l'ancien UPDATE media_files.liked et conserve exactement sa sémantique
	// (média supprimé ou inconnu → like inopérant, 404).
	var one int
	err := exec.QueryRowContext(ctx, qMediaVisibleExists, filePath).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		slog.WarnContext(ctx, "SetMediaLikeAtomic: file_path absent (ou supprimé) en DB",
			"file_path", filePath, "liked", liked)
		return false, nil // file_path inconnu — caller traduit en 404 sans écrire d'event
	}
	if err != nil {
		return false, fmt.Errorf("SetMediaLikeAtomic lookup: %w", err)
	}

	if likerSlug == "" {
		slog.WarnContext(ctx, "SetMediaLikeAtomic: aucun liker — like non persistable",
			"file_path", filePath, "liked", liked)
		return true, nil
	}

	// APPEND-ONLY : like/unlike = INSERT pur d'event dans media_likes_history.
	//
	// liked_at en UTC EXPLICITE (lot S6) : la route NOMINALE de la meme colonne
	// (persist.SharedSocialPersister.AddLike) pose `time.Now().UTC()`. Un
	// `CURRENT_TIMESTAMP` nu rendrait ici un TIMESTAMPTZ coerce par le fuseau de SESSION,
	// soit deux horloges sur une seule colonne : le tri d'affichage des likers
	// (GetMediaLikers, ORDER BY liked_at) melangerait alors deux referentiels, et la ligne
	// serait incoherente avec son propre written_at, deja canonique.
	if liked {
		_, err := exec.ExecContext(ctx, `
			INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
			VALUES (?, ?, ?, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
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
	// liked_at en UTC explicite, meme raison qu'en route nominale ci-dessus (lot S6).
	if liked {
		_, err := r.socialDB().Exec(ctx, `
			INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
			VALUES (?, ?, ?, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
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
