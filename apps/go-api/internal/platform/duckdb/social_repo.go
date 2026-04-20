// Package duckdb — social_repo.go : accès DB pour les données sociales (favoris).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// SocialRepo implémente port.SocialRepository.
type SocialRepo struct {
	pdb *PlayerDB
}

// NewSocialRepo crée un SocialRepo pour un joueur.
func NewSocialRepo(pdb *PlayerDB) *SocialRepo {
	return &SocialRepo{pdb: pdb}
}

// socialConn retourne SharedSocial si disponible, sinon nil (opération no-op).
func (r *SocialRepo) socialConn() *DB {
	return r.pdb.SharedSocial
}

// ToggleMatchFavorite bascule l'état favori d'un match pour un joueur.
func (r *SocialRepo) ToggleMatchFavorite(ctx context.Context, playerSlug, matchID string, favorited bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := r.socialConn()
	if db == nil {
		slog.WarnContext(ctx, "social_repo: SharedSocial non disponible, favori ignoré",
			"player", playerSlug, "match_id", matchID)
		return fmt.Errorf("shared_social.duckdb non disponible")
	}

	if favorited {
		_, err := db.Exec(ctx, `
			INSERT INTO match_favorites (player_slug, match_id, favorited_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (player_slug, match_id) DO NOTHING
		`, playerSlug, matchID)
		if err != nil {
			return fmt.Errorf("ToggleMatchFavorite insert: %w", err)
		}
		return nil
	}
	_, err := db.Exec(ctx, `
		DELETE FROM match_favorites WHERE player_slug = ? AND match_id = ?
	`, playerSlug, matchID)
	if err != nil {
		return fmt.Errorf("ToggleMatchFavorite delete: %w", err)
	}
	return nil
}

// IsMatchFavorite indique si un match est en favori pour un joueur.
func (r *SocialRepo) IsMatchFavorite(ctx context.Context, playerSlug, matchID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := r.socialConn()
	if db == nil {
		return false, nil
	}

	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM match_favorites WHERE player_slug = ? AND match_id = ?
	`, playerSlug, matchID).Scan(&count)
	return count > 0, err
}

// GetFavoriteMatchIDs retourne l'ensemble des match_id mis en favoris par un joueur.
// Retourne nil si shared_social n'est pas disponible (dégradation silencieuse).
func (r *SocialRepo) GetFavoriteMatchIDs(ctx context.Context, playerSlug string) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := r.socialConn()
	if db == nil {
		return nil, nil
	}

	rows, err := db.Query(ctx, `SELECT match_id FROM match_favorites WHERE player_slug = ?`, playerSlug)
	if err != nil {
		return nil, fmt.Errorf("GetFavoriteMatchIDs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var matchID string
		if err := rows.Scan(&matchID); err != nil {
			return nil, fmt.Errorf("GetFavoriteMatchIDs scan: %w", err)
		}
		result[matchID] = true
	}
	return result, rows.Err()
}
