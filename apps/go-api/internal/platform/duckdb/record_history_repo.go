// Package duckdb — record_history_repo.go : timeline des PB battus.
//
// RecordHistoryRepo  → table `record_history` dans stats.duckdb (par joueur).
//
// Append-only : à chaque PB battu, une nouvelle ligne. Pas de mise à jour
// ni suppression — c'est l'historique des PB qui sert à l'affichage timeline
// (cf. plan §5.3).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/progression/records"
)

// RecordHistoryRepo persiste l'historique des PB dans stats.duckdb (par joueur).
type RecordHistoryRepo struct {
	db PlayerReadHandle
}

// NewRecordHistoryRepo construit le repo.
func NewRecordHistoryRepo(db *DB) *RecordHistoryRepo {
	return &RecordHistoryRepo{db: NewPlayerReadHandle(db)}
}

// Compile-time assertion.
var _ records.HistoryRepo = (*RecordHistoryRepo)(nil)

// defaultHistoryLimit est la limite par défaut quand l'appelant n'en spécifie
// pas (limit <= 0). Empêche de tirer toute la timeline d'un joueur prolifique.
const defaultHistoryLimit = 100

// Append insère une entrée d'historique. Idempotent au sens où l'ID est
// supposé unique (UUID) — pas de protection si l'appelant réutilise un ID.
func (r *RecordHistoryRepo) Append(ctx context.Context, h records.RecordHistory) error {
	if h.ID == "" {
		return fmt.Errorf("RecordHistoryRepo.Append: empty ID")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
		INSERT INTO record_history (id, user_id, title_slug, metric, period, value, achieved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		h.ID, h.UserID, h.TitleSlug, h.Metric, string(h.Period), h.Value, h.AchievedAt,
	)
	if err != nil {
		return fmt.Errorf("RecordHistoryRepo.Append: %w", err)
	}
	return nil
}

// ListRecent retourne les `limit` entrées les plus récentes pour (user, title),
// triées par achieved_at DESC. Si limit <= 0, defaultHistoryLimit est utilisé.
func (r *RecordHistoryRepo) ListRecent(ctx context.Context, userID, titleSlug string, limit int) ([]records.RecordHistory, error) {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT id, user_id, title_slug, metric, period, value, achieved_at
		FROM record_history
		WHERE user_id = ? AND title_slug = ?
		ORDER BY achieved_at DESC
		LIMIT ?
	`,
		userID, titleSlug, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("RecordHistoryRepo.ListRecent: %w", err)
	}
	defer rows.Close()
	var out []records.RecordHistory
	for rows.Next() {
		var (
			h          records.RecordHistory
			periodStr  string
			achievedAt sql.NullTime
		)
		if err := rows.Scan(&h.ID, &h.UserID, &h.TitleSlug, &h.Metric, &periodStr, &h.Value, &achievedAt); err != nil {
			return nil, fmt.Errorf("RecordHistoryRepo.ListRecent scan: %w", err)
		}
		h.Period = records.RecordPeriod(periodStr)
		if achievedAt.Valid {
			h.AchievedAt = achievedAt.Time
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
