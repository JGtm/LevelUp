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
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT milestone_id FROM milestone_earned
		WHERE user_id = ? AND title_slug = ? AND milestone_id = ?
		LIMIT 1
	`, userID, titleSlug, milestoneID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("MilestoneEarnedRepo.IsEarned: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(&dummy); err != nil {
		return false, fmt.Errorf("MilestoneEarnedRepo.IsEarned: %w", err)
	}
	return true, nil
}

// Append insère un débloquage (idempotent : earned_at original conservé).
//
// Pattern ART-safe SELECT-then-INSERT (CLAUDE.md) : un milestone se débloque une
// seule fois → "insert if not exists". Évite l'ON CONFLICT, qui déclenche le
// chemin "delete from index" de DuckDB (bug ART "Failed to delete all rows from
// index") et échoue en Binder Error si l'index a été invalidé par un crash ART
// antérieur. Fréquence basse (post-sync par joueur).
func (r *MilestoneEarnedRepo) Append(ctx context.Context, e milestones.Earned) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	earnedAt := e.EarnedAt
	if earnedAt.IsZero() {
		earnedAt = time.Now().UTC()
	}
	// E5 (revue 2026-07) : QueryRow/Exec PLATS sur le handle player pdb.Player (champ
	// nu r.db) → invisibles au garde-rail grep mais atteignables par la race « database
	// is closed » d'un Reopen concurrent. Routés vers les variantes *Recovered.
	var exists bool
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM milestone_earned
			WHERE user_id = ? AND title_slug = ? AND milestone_id = ?
		)`, e.UserID, e.TitleSlug, e.MilestoneID)
	if err != nil {
		return fmt.Errorf("MilestoneEarnedRepo.Append exists: %w", err)
	}
	if err := rows.Scan(&exists); err != nil {
		_ = rows.Close()
		return fmt.Errorf("MilestoneEarnedRepo.Append exists scan: %w", err)
	}
	_ = rows.Close()
	if exists {
		return nil
	}
	if _, err := r.db.ExecRecovered(ctx, `
		INSERT INTO milestone_earned (user_id, title_slug, milestone_id, earned_at)
		VALUES (?, ?, ?, ?)
	`, e.UserID, e.TitleSlug, e.MilestoneID, earnedAt); err != nil {
		return fmt.Errorf("MilestoneEarnedRepo.Append: %w", err)
	}
	return nil
}

// ListByUser retourne tous les milestones débloqués (DESC sur earned_at).
func (r *MilestoneEarnedRepo) ListByUser(ctx context.Context, userID, titleSlug string) ([]milestones.Earned, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
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
