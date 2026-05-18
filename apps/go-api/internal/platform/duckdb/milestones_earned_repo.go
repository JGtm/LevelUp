// Package duckdb — milestones_earned_repo.go : milestones débloqués par joueur
// (table `milestone_earned` dans stats.duckdb).
//
// PK composite (user_id, title_slug, milestone_id) garantit l'idempotence :
// un milestone n'est débloqué qu'une seule fois par joueur par titre.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/progression/milestones"
)

// MilestoneEarnedRepo persiste les milestones débloqués (stats.duckdb par joueur).
type MilestoneEarnedRepo struct {
	db *DB
}

// NewMilestoneEarnedRepo construit le repo.
func NewMilestoneEarnedRepo(db *DB) *MilestoneEarnedRepo {
	return &MilestoneEarnedRepo{db: db}
}

// Compile-time assertion.
var _ milestones.EarnedRepo = (*MilestoneEarnedRepo)(nil)

// IsEarned retourne true si l'enregistrement existe.
func (r *MilestoneEarnedRepo) IsEarned(ctx context.Context, userID, titleSlug, milestoneID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var dummy string
	err := r.db.QueryRow(ctx, `
		SELECT milestone_id FROM milestone_earned
		WHERE user_id = ? AND title_slug = ? AND milestone_id = ?
		LIMIT 1
	`, userID, titleSlug, milestoneID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("MilestoneEarnedRepo.IsEarned: %w", err)
	}
	return true, nil
}

// Append insère un débloquage. INSERT ON CONFLICT DO NOTHING pour idempotence
// (un milestone déjà débloqué reste avec son earned_at original).
func (r *MilestoneEarnedRepo) Append(ctx context.Context, e milestones.Earned) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	earnedAt := e.EarnedAt
	if earnedAt.IsZero() {
		earnedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO milestone_earned (user_id, title_slug, milestone_id, earned_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, title_slug, milestone_id) DO NOTHING
	`,
		e.UserID, e.TitleSlug, e.MilestoneID, earnedAt,
	)
	if err != nil {
		return fmt.Errorf("MilestoneEarnedRepo.Append: %w", err)
	}
	return nil
}

// ListByUser retourne tous les milestones débloqués (DESC sur earned_at).
func (r *MilestoneEarnedRepo) ListByUser(ctx context.Context, userID, titleSlug string) ([]milestones.Earned, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, `
		SELECT user_id, title_slug, milestone_id, earned_at
		FROM milestone_earned
		WHERE user_id = ? AND title_slug = ?
		ORDER BY earned_at DESC
	`, userID, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("MilestoneEarnedRepo.ListByUser: %w", err)
	}
	defer rows.Close()
	var out []milestones.Earned
	for rows.Next() {
		var e milestones.Earned
		if err := rows.Scan(&e.UserID, &e.TitleSlug, &e.MilestoneID, &e.EarnedAt); err != nil {
			return nil, fmt.Errorf("MilestoneEarnedRepo.ListByUser scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
