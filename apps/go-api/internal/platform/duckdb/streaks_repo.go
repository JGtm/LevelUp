// Package duckdb — streaks_repo.go : persistance DuckDB des streaks (V2 Ascension).
//
// Implémente streaks.Repo. Stockage : `streak` table dans stats.duckdb (par joueur).
//
// Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §7.1.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/progression/streaks"
)

// StreaksRepo persiste les streaks d'un joueur dans sa stats.duckdb.
type StreaksRepo struct {
	db *DB
}

// NewStreaksRepo construit le repo.
func NewStreaksRepo(db *DB) *StreaksRepo {
	return &StreaksRepo{db: db}
}

// Compile-time assertion.
var _ streaks.Repo = (*StreaksRepo)(nil)

const streaksSelectColumns = `
SELECT id, user_id, title_slug, type, started_at,
       current_length, best_length, last_increment_at, threshold,
       shields_used, shields_available, status, broken_at
FROM streak`

// GetActive retourne la streak active (status != broken) pour (user, title, type),
// ou nil si aucune. Si plusieurs lignes correspondent (cas pathologique), la plus
// récente par started_at est retournée.
func (r *StreaksRepo) GetActive(ctx context.Context, userID, titleSlug string, st streaks.StreakType) (*streaks.Streak, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, streaksSelectColumns+`
		WHERE user_id = ? AND title_slug = ? AND type = ? AND status != 'broken'
		ORDER BY started_at DESC LIMIT 1`,
		userID, titleSlug, string(st),
	)
	s, err := scanStreak(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("StreaksRepo.GetActive: %w", err)
	}
	return &s, nil
}

// Upsert crée ou met à jour une streak. INSERT ... ON CONFLICT (id) DO UPDATE.
func (r *StreaksRepo) Upsert(ctx context.Context, s streaks.Streak) error {
	if s.ID == "" {
		return fmt.Errorf("StreaksRepo.Upsert: empty ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO streak (
			id, user_id, title_slug, type, started_at,
			current_length, best_length, last_increment_at, threshold,
			shields_used, shields_available, status, broken_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT (id) DO UPDATE SET
			current_length    = excluded.current_length,
			best_length       = excluded.best_length,
			last_increment_at = excluded.last_increment_at,
			threshold         = excluded.threshold,
			shields_used      = excluded.shields_used,
			shields_available = excluded.shields_available,
			status            = excluded.status,
			broken_at         = excluded.broken_at
	`,
		s.ID, s.UserID, s.TitleSlug, string(s.Type), s.StartedAt,
		s.CurrentLength, s.BestLength, nullableTime(s.LastIncrementAt), nullableFloat(s.Threshold),
		s.ShieldsUsed, s.ShieldsAvailable, string(s.Status), nullableTime(s.BrokenAt),
	)
	if err != nil {
		return fmt.Errorf("StreaksRepo.Upsert: %w", err)
	}
	return nil
}

// List retourne toutes les streaks d'un joueur sur un titre (active + historique).
func (r *StreaksRepo) List(ctx context.Context, userID, titleSlug string) ([]streaks.Streak, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, streaksSelectColumns+`
		WHERE user_id = ? AND title_slug = ?
		ORDER BY started_at DESC`,
		userID, titleSlug,
	)
	if err != nil {
		return nil, fmt.Errorf("StreaksRepo.List: %w", err)
	}
	defer rows.Close()
	var out []streaks.Streak
	for rows.Next() {
		s, err := scanStreak(rows)
		if err != nil {
			return nil, fmt.Errorf("StreaksRepo.List scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// scanStreak parse une ligne de la table streak en streaks.Streak.
// rowScanner est défini dans prestige_player_helpers.go.
func scanStreak(row rowScanner) (streaks.Streak, error) {
	var (
		s             streaks.Streak
		streakType    string
		streakStatus  string
		lastIncrement sql.NullTime
		threshold     sql.NullFloat64
		brokenAt      sql.NullTime
	)
	err := row.Scan(
		&s.ID, &s.UserID, &s.TitleSlug, &streakType, &s.StartedAt,
		&s.CurrentLength, &s.BestLength, &lastIncrement, &threshold,
		&s.ShieldsUsed, &s.ShieldsAvailable, &streakStatus, &brokenAt,
	)
	if err != nil {
		return streaks.Streak{}, err
	}
	s.Type = streaks.StreakType(streakType)
	s.Status = streaks.StreakStatus(streakStatus)
	if lastIncrement.Valid {
		s.LastIncrementAt = &lastIncrement.Time
	}
	if threshold.Valid {
		s.Threshold = &threshold.Float64
	}
	if brokenAt.Valid {
		s.BrokenAt = &brokenAt.Time
	}
	return s, nil
}

// nullableTime convertit un *time.Time en sql.NullTime (nil → NULL).
func nullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// nullableFloat convertit un *float64 en sql.NullFloat64 (nil → NULL).
func nullableFloat(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
