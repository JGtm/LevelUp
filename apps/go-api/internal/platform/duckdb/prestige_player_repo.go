// Package duckdb — repositories Prestige côté joueur (stats.duckdb).
//
// 5 structs distinctes pour respecter les interfaces du package prestige
// dont les méthodes Create/Get/List collisionnent en signature de retour.

package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/prestige"
)

// ─────────── ChallengeRepo ───────────

// PrestigeChallengeRepo implémente prestige.ChallengeRepo.
type PrestigeChallengeRepo struct{ db *DB }

// NewPrestigeChallengeRepo construit le repo.
func NewPrestigeChallengeRepo(db *DB) *PrestigeChallengeRepo {
	return &PrestigeChallengeRepo{db: db}
}

// Compile-time assertion.
var _ prestige.ChallengeRepo = (*PrestigeChallengeRepo)(nil)

func (r *PrestigeChallengeRepo) Create(ctx context.Context, c prestige.Challenge) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO challenge (
			id, user_id, title_slug, arc_id, position, template_id,
			metric, target, target_per_member, window_type, window_value,
			cadence, eval_type, mode, tier, data_tier, label, status,
			created_at, committed_at, completed_at, expired_at, abandoned_at,
			last_palier_recompute_at, is_private
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		c.ID, c.UserID, c.TitleSlug, nullableStr(c.ArcID), c.Position, nullableStr(c.TemplateID),
		c.Metric, c.Target, c.TargetPerMember, string(c.WindowType), c.WindowValue,
		string(c.Cadence), string(c.EvalType), string(c.Mode), string(c.Tier), string(c.DataTier),
		c.Label, string(c.Status),
		c.CreatedAt, c.CommittedAt, c.CompletedAt, c.ExpiredAt, c.AbandonedAt,
		c.LastPalierRecomputeAt, c.IsPrivate,
	)
	if err != nil {
		return fmt.Errorf("ChallengeRepo.Create: %w", err)
	}
	return nil
}

func (r *PrestigeChallengeRepo) Get(ctx context.Context, id string) (prestige.Challenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, challengeSelectColumns+" WHERE id = ?", id)
	c, err := scanChallenge(row)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.Challenge{}, prestige.ErrChallengeNotFound
	}
	return c, err
}

func (r *PrestigeChallengeRepo) List(ctx context.Context, f prestige.ChallengeFilter) ([]prestige.Challenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	q, args := buildChallengeListQuery(f)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo.List: %w", err)
	}
	defer rows.Close()
	var out []prestige.Challenge
	for rows.Next() {
		c, err := scanChallenge(rows)
		if err != nil {
			return nil, fmt.Errorf("ChallengeRepo.List scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PrestigeChallengeRepo) UpdateStatus(ctx context.Context, id string, status prestige.ChallengeStatus, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	col := timestampColumnFor(status)
	if col == "" {
		_, err := r.db.Exec(ctx, `UPDATE challenge SET status = ? WHERE id = ?`, string(status), id)
		return err
	}
	q := fmt.Sprintf(`UPDATE challenge SET status = ?, %s = ? WHERE id = ?`, col)
	_, err := r.db.Exec(ctx, q, string(status), at, id)
	return err
}

func (r *PrestigeChallengeRepo) UpdateLabel(ctx context.Context, id, label string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `UPDATE challenge SET label = ? WHERE id = ?`, label, id)
	return err
}

func (r *PrestigeChallengeRepo) UpdateTarget(ctx context.Context, id string, target float64, tier prestige.Tier, dataTier prestige.DataTier, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		UPDATE challenge
		SET target = ?, tier = ?, data_tier = ?, last_palier_recompute_at = ?
		WHERE id = ?
	`, target, string(tier), string(dataTier), at, id)
	return err
}

func (r *PrestigeChallengeRepo) CountActiveByCadence(ctx context.Context, userID, titleSlug string, cadence prestige.Cadence) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND cadence = ? AND status = 'active'
	`, userID, titleSlug, string(cadence)).Scan(&n)
	return n, err
}

func (r *PrestigeChallengeRepo) CountActiveTotal(ctx context.Context, userID, titleSlug string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND status = 'active'
	`, userID, titleSlug).Scan(&n)
	return n, err
}

func (r *PrestigeChallengeRepo) CountCreatedSince(ctx context.Context, userID, titleSlug string, mode prestige.ChallengeMode, since time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND mode = ? AND created_at >= ?
	`, userID, titleSlug, string(mode), since).Scan(&n)
	return n, err
}

// ─────────── ArcRepo ───────────

// PrestigeArcRepo implémente prestige.ArcRepo.
type PrestigeArcRepo struct{ db *DB }

func NewPrestigeArcRepo(db *DB) *PrestigeArcRepo { return &PrestigeArcRepo{db: db} }

var _ prestige.ArcRepo = (*PrestigeArcRepo)(nil)

func (r *PrestigeArcRepo) Create(ctx context.Context, a prestige.Arc) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO arc (id, user_id, title_slug, title, description, is_preset, preset_id, created_at, completed_at)
		VALUES (?,?,?,?,?,?,?,?,?)
	`, a.ID, a.UserID, a.TitleSlug, a.Title, a.Description, a.IsPreset, nullableStr(a.PresetID), a.CreatedAt, a.CompletedAt)
	return err
}

func (r *PrestigeArcRepo) Get(ctx context.Context, id string) (prestige.Arc, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, title_slug, title, COALESCE(description, ''),
		       is_preset, COALESCE(preset_id, ''), created_at, completed_at
		FROM arc WHERE id = ?`, id)
	a, err := scanArc(row)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.Arc{}, prestige.ErrArcNotFound
	}
	return a, err
}

func (r *PrestigeArcRepo) ListByUser(ctx context.Context, userID, titleSlug string) ([]prestige.Arc, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, title_slug, title, COALESCE(description, ''),
		       is_preset, COALESCE(preset_id, ''), created_at, completed_at
		FROM arc WHERE user_id = ? AND title_slug = ? ORDER BY created_at DESC
	`, userID, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.Arc
	for rows.Next() {
		a, err := scanArc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PrestigeArcRepo) MarkCompleted(ctx context.Context, id string, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `UPDATE arc SET completed_at = ? WHERE id = ?`, at, id)
	return err
}

// ─────────── MomentCardRepo ───────────

// PrestigeMomentCardRepo implémente prestige.MomentCardRepo.
type PrestigeMomentCardRepo struct{ db *DB }

func NewPrestigeMomentCardRepo(db *DB) *PrestigeMomentCardRepo {
	return &PrestigeMomentCardRepo{db: db}
}

var _ prestige.MomentCardRepo = (*PrestigeMomentCardRepo)(nil)

func (r *PrestigeMomentCardRepo) Create(ctx context.Context, mc prestige.MomentCard) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO moment_card (id, challenge_id, blob_path, created_at) VALUES (?, ?, ?, ?)
	`, mc.ID, mc.ChallengeID, mc.BlobPath, mc.CreatedAt)
	return err
}

func (r *PrestigeMomentCardRepo) GetByChallenge(ctx context.Context, challengeID string) (prestige.MomentCard, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, `
		SELECT id, challenge_id, COALESCE(blob_path, ''), created_at
		FROM moment_card WHERE challenge_id = ?`, challengeID)
	var mc prestige.MomentCard
	err := row.Scan(&mc.ID, &mc.ChallengeID, &mc.BlobPath, &mc.CreatedAt)
	return mc, err
}

func (r *PrestigeMomentCardRepo) ListRecent(ctx context.Context, userID, titleSlug string, limit int) ([]prestige.MomentCard, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT mc.id, mc.challenge_id, COALESCE(mc.blob_path, ''), mc.created_at
		FROM moment_card mc
		JOIN challenge c ON c.id = mc.challenge_id
		WHERE c.user_id = ? AND c.title_slug = ?
		ORDER BY mc.created_at DESC LIMIT ?
	`, userID, titleSlug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []prestige.MomentCard
	for rows.Next() {
		var mc prestige.MomentCard
		if err := rows.Scan(&mc.ID, &mc.ChallengeID, &mc.BlobPath, &mc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, mc)
	}
	return out, rows.Err()
}

// ─────────── TelemetryRepo ───────────

// PrestigeTelemetryRepo implémente prestige.TelemetryRepo.
type PrestigeTelemetryRepo struct{ db *DB }

func NewPrestigeTelemetryRepo(db *DB) *PrestigeTelemetryRepo { return &PrestigeTelemetryRepo{db: db} }

var _ prestige.TelemetryRepo = (*PrestigeTelemetryRepo)(nil)

func (r *PrestigeTelemetryRepo) Emit(ctx context.Context, ev prestige.PrestigeTelemetry) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO prestige_telemetry (
			id, user_id, challenge_id, event_type, palier, stretch_ratio,
			baseline_value, mode, cadence, eval_type, time_since_create_seconds, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		ev.ID, ev.UserID, nullableStr(ev.ChallengeID), ev.EventType,
		nullableStr(string(ev.Palier)), ev.StretchRatio, ev.BaselineValue,
		nullableStr(string(ev.Mode)), nullableStr(string(ev.Cadence)),
		nullableStr(string(ev.EvalType)), ev.TimeSinceCreateSeconds, ev.CreatedAt,
	)
	return err
}

// ─────────── BaselineStateRepo ───────────

// PrestigeBaselineStateRepo implémente prestige.BaselineStateRepo.
type PrestigeBaselineStateRepo struct{ db *DB }

func NewPrestigeBaselineStateRepo(db *DB) *PrestigeBaselineStateRepo {
	return &PrestigeBaselineStateRepo{db: db}
}

var _ prestige.BaselineStateRepo = (*PrestigeBaselineStateRepo)(nil)

func (r *PrestigeBaselineStateRepo) Get(ctx context.Context, userID, titleSlug, metric string) (prestige.BaselineState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, `
		SELECT user_id, title_slug, metric, last_match_at, is_stale,
		       recovery_matches_remaining, updated_at
		FROM baseline_state WHERE user_id = ? AND title_slug = ? AND metric = ?
	`, userID, titleSlug, metric)
	var st prestige.BaselineState
	err := row.Scan(&st.UserID, &st.TitleSlug, &st.Metric, &st.LastMatchAt,
		&st.IsStale, &st.RecoveryMatchesRemaining, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.BaselineState{
			UserID: userID, TitleSlug: titleSlug, Metric: metric,
		}, nil
	}
	return st, err
}

func (r *PrestigeBaselineStateRepo) Upsert(ctx context.Context, st prestige.BaselineState) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO baseline_state (user_id, title_slug, metric, last_match_at,
		                            is_stale, recovery_matches_remaining, updated_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT (user_id, title_slug, metric) DO UPDATE SET
			last_match_at = EXCLUDED.last_match_at,
			is_stale = EXCLUDED.is_stale,
			recovery_matches_remaining = EXCLUDED.recovery_matches_remaining,
			updated_at = EXCLUDED.updated_at
	`, st.UserID, st.TitleSlug, st.Metric, st.LastMatchAt,
		st.IsStale, st.RecoveryMatchesRemaining, st.UpdatedAt)
	return err
}
