// Package duckdb — repositories Prestige côté joueur (stats.duckdb).
//
// 5 structs distinctes pour respecter les interfaces du package prestige
// dont les méthodes Create/Get/List collisionnent en signature de retour.

package prestige

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// ─────────── ChallengeRepo ───────────

// PrestigeChallengeRepo implémente prestige.ChallengeRepo.
type PrestigeChallengeRepo struct{ db duckdb.PlayerReadHandle }

// NewPrestigeChallengeRepo construit le repo.
func NewPrestigeChallengeRepo(db *duckdb.DB) *PrestigeChallengeRepo {
	return &PrestigeChallengeRepo{db: duckdb.NewPlayerReadHandle(db)}
}

// Compile-time assertion.
var _ prestige.ChallengeRepo = (*PrestigeChallengeRepo)(nil)

func (r *PrestigeChallengeRepo) Create(ctx context.Context, c prestige.Challenge) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
		INSERT INTO challenge (
			id, user_id, title_slug, arc_id, position, template_id,
			metric, target, target_per_member, window_type, window_value,
			cadence, eval_type, mode, tier, data_tier, label, status,
			created_at, committed_at, completed_at, expired_at, abandoned_at,
			last_palier_recompute_at, is_private, source
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		c.ID, c.UserID, c.TitleSlug, duckdb.NullableStr(c.ArcID), c.Position, duckdb.NullableStr(c.TemplateID),
		c.Metric, c.Target, c.TargetPerMember, string(c.WindowType), c.WindowValue,
		string(c.Cadence), string(c.EvalType), string(c.Mode), string(c.Tier), string(c.DataTier),
		c.Label, string(c.Status),
		c.CreatedAt, c.CommittedAt, c.CompletedAt, c.ExpiredAt, c.AbandonedAt,
		c.LastPalierRecomputeAt, c.IsPrivate, duckdb.NullableStr(c.Source),
	)
	if err != nil {
		return fmt.Errorf("ChallengeRepo.Create: %w", err)
	}
	return nil
}

func (r *PrestigeChallengeRepo) Get(ctx context.Context, id string) (prestige.Challenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, challengeSelectColumns+" WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.Challenge{}, prestige.ErrChallengeNotFound
	}
	if err != nil {
		return prestige.Challenge{}, err
	}
	defer rows.Close()
	return scanChallenge(rows)
}

func (r *PrestigeChallengeRepo) List(ctx context.Context, f prestige.ChallengeFilter) ([]prestige.Challenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	q, args := buildChallengeListQuery(f)
	rows, err := r.db.QueryRecovered(ctx, q, args...)
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
		_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET status = ? WHERE id = ?`, string(status), id)
		return err
	}
	q := fmt.Sprintf(`UPDATE challenge SET status = ?, %s = ? WHERE id = ?`, col)
	_, err := r.db.ExecRecovered(ctx, q, string(status), at, id)
	return err
}

func (r *PrestigeChallengeRepo) UpdateLabel(ctx context.Context, id, label string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET label = ? WHERE id = ?`, label, id)
	return err
}

func (r *PrestigeChallengeRepo) UpdateTarget(
	ctx context.Context,
	id string,
	target float64,
	tier prestige.Tier,
	dataTier prestige.DataTier,
	at time.Time,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
		UPDATE challenge
		SET target = ?, tier = ?, data_tier = ?, last_palier_recompute_at = ?
		WHERE id = ?
	`, target, string(tier), string(dataTier), at, id)
	return err
}

func (r *PrestigeChallengeRepo) CountActiveByCadence(ctx context.Context, userID, titleSlug string, cadence prestige.Cadence) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND cadence = ? AND status = 'active'
	`, userID, titleSlug, string(cadence))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	err = rows.Scan(&n)
	return n, err
}

func (r *PrestigeChallengeRepo) CountActiveTotal(ctx context.Context, userID, titleSlug string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND status = 'active'
	`, userID, titleSlug)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	err = rows.Scan(&n)
	return n, err
}

func (r *PrestigeChallengeRepo) CountCreatedSince(
	ctx context.Context,
	userID, titleSlug string,
	mode prestige.ChallengeMode,
	since time.Time,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM challenge
		WHERE user_id = ? AND title_slug = ? AND mode = ? AND created_at >= ?
	`, userID, titleSlug, string(mode), since)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	err = rows.Scan(&n)
	return n, err
}

func (r *PrestigeChallengeRepo) DetachFromArc(ctx context.Context, arcID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET arc_id = NULL WHERE arc_id = ?`, arcID)
	return err
}

func (r *PrestigeChallengeRepo) DeleteByArc(ctx context.Context, arcID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `DELETE FROM challenge WHERE arc_id = ?`, arcID)
	return err
}

// ─────────── ArcRepo ───────────

// PrestigeArcRepo implémente prestige.ArcRepo.
type PrestigeArcRepo struct{ db duckdb.PlayerReadHandle }

func NewPrestigeArcRepo(db *duckdb.DB) *PrestigeArcRepo {
	return &PrestigeArcRepo{db: duckdb.NewPlayerReadHandle(db)}
}

var (
	_ prestige.ArcRepo       = (*PrestigeArcRepo)(nil)
	_ prestige.ArcTitlesRepo = (*PrestigeArcRepo)(nil)
)

func (r *PrestigeArcRepo) Create(ctx context.Context, a prestige.Arc) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := r.db.ExecRecovered(ctx, `
		INSERT INTO arc (id, user_id, title_slug, title, description, is_preset, preset_id, created_at, completed_at)
		VALUES (?,?,?,?,?,?,?,?,?)
	`, a.ID, a.UserID, a.TitleSlug, a.Title, a.Description, a.IsPreset, duckdb.NullableStr(a.PresetID), a.CreatedAt, a.CompletedAt); err != nil {
		return err
	}
	// Invariant cross-titre : 1 ligne (arc.id, arc.title_slug) par arc. La voie
	// arc_titles reste ainsi un sur-ensemble des lectures mono-titre (cf.
	// PLAN_CROSS_TITLE_ARCS_BACKEND Phase 2). ON CONFLICT DO NOTHING = idempotent.
	_, err := r.db.ExecRecovered(ctx, `
		INSERT INTO arc_titles (arc_id, title_slug) VALUES (?, ?) ON CONFLICT DO NOTHING
	`, a.ID, a.TitleSlug)
	return err
}

// ArcTitles retourne les title_slug couverts par un arc via la table de jointure
// arc_titles, avec fallback sur arc.title_slug si la jointure est vide (garde-fou
// pré-backfill). Voir prestige.ArcTitlesRepo.
func (r *PrestigeArcRepo) ArcTitles(ctx context.Context, arcID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `SELECT title_slug FROM arc_titles WHERE arc_id = ? ORDER BY title_slug`, arcID)
	if err != nil {
		return nil, fmt.Errorf("ArcTitles: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("ArcTitles scan: %w", err)
		}
		out = append(out, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		// Fallback pré-backfill : titre primaire de l'arc.
		frows, err := r.db.QueryRowRecovered(ctx, `SELECT title_slug FROM arc WHERE id = ?`, arcID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, prestige.ErrArcNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("ArcTitles fallback: %w", err)
		}
		defer frows.Close()
		var primary string
		if err := frows.Scan(&primary); err != nil {
			return nil, fmt.Errorf("ArcTitles fallback: %w", err)
		}
		return []string{primary}, nil
	}
	return out, nil
}

// ArcsByTitle liste les arcs d'un joueur couvrant un titre via arc_titles. En
// mono-titre (1 ligne/arc) le résultat est strictement équivalent à l'ancienne
// requête ListByUser sur arc.title_slug. Voir prestige.ArcTitlesRepo.
func (r *PrestigeArcRepo) ArcsByTitle(ctx context.Context, userID, titleSlug string) ([]prestige.Arc, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT a.id, a.user_id, a.title_slug, a.title, COALESCE(a.description, ''),
		       a.is_preset, COALESCE(a.preset_id, ''), a.created_at, a.completed_at
		FROM arc a
		JOIN arc_titles act ON act.arc_id = a.id
		WHERE a.user_id = ? AND act.title_slug = ?
		ORDER BY a.created_at DESC
	`, userID, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("ArcsByTitle: %w", err)
	}
	defer rows.Close()
	var out []prestige.Arc
	for rows.Next() {
		a, err := scanArc(rows)
		if err != nil {
			return nil, fmt.Errorf("ArcsByTitle scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PrestigeArcRepo) Get(ctx context.Context, id string) (prestige.Arc, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT id, user_id, title_slug, title, COALESCE(description, ''),
		       is_preset, COALESCE(preset_id, ''), created_at, completed_at
		FROM arc WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.Arc{}, prestige.ErrArcNotFound
	}
	if err != nil {
		return prestige.Arc{}, err
	}
	defer rows.Close()
	return scanArc(rows)
}

func (r *PrestigeArcRepo) ListByUser(ctx context.Context, userID, titleSlug string) ([]prestige.Arc, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
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
	_, err := r.db.ExecRecovered(ctx, `UPDATE arc SET completed_at = ? WHERE id = ?`, at, id)
	return err
}

func (r *PrestigeArcRepo) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `DELETE FROM arc WHERE id = ?`, id)
	return err
}

// ─────────── MomentCardRepo ───────────

// PrestigeMomentCardRepo implémente prestige.MomentCardRepo.
type PrestigeMomentCardRepo struct{ db duckdb.PlayerReadHandle }

func NewPrestigeMomentCardRepo(db *duckdb.DB) *PrestigeMomentCardRepo {
	return &PrestigeMomentCardRepo{db: duckdb.NewPlayerReadHandle(db)}
}

var _ prestige.MomentCardRepo = (*PrestigeMomentCardRepo)(nil)

func (r *PrestigeMomentCardRepo) Create(ctx context.Context, mc prestige.MomentCard) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
		INSERT INTO moment_card (id, challenge_id, blob_path, created_at) VALUES (?, ?, ?, ?)
	`, mc.ID, mc.ChallengeID, mc.BlobPath, mc.CreatedAt)
	return err
}

func (r *PrestigeMomentCardRepo) GetByChallenge(ctx context.Context, challengeID string) (prestige.MomentCard, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT id, challenge_id, COALESCE(blob_path, ''), created_at
		FROM moment_card WHERE challenge_id = ?`, challengeID)
	if err != nil {
		// Inclut sql.ErrNoRows : contrat inchangé, le caller le mappe lui-même.
		return prestige.MomentCard{}, err
	}
	defer rows.Close()
	var mc prestige.MomentCard
	err = rows.Scan(&mc.ID, &mc.ChallengeID, &mc.BlobPath, &mc.CreatedAt)
	return mc, err
}

func (r *PrestigeMomentCardRepo) ListRecent(ctx context.Context, userID, titleSlug string, limit int) ([]prestige.MomentCard, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryRecovered(ctx, `
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
type PrestigeTelemetryRepo struct{ db duckdb.PlayerReadHandle }

func NewPrestigeTelemetryRepo(db *duckdb.DB) *PrestigeTelemetryRepo {
	return &PrestigeTelemetryRepo{db: duckdb.NewPlayerReadHandle(db)}
}

var _ prestige.TelemetryRepo = (*PrestigeTelemetryRepo)(nil)

func (r *PrestigeTelemetryRepo) Emit(ctx context.Context, ev prestige.PrestigeTelemetry) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
		INSERT INTO prestige_telemetry (
			id, user_id, challenge_id, event_type, palier, stretch_ratio,
			baseline_value, mode, cadence, eval_type, time_since_create_seconds, source, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		ev.ID, ev.UserID, duckdb.NullableStr(ev.ChallengeID), ev.EventType,
		duckdb.NullableStr(string(ev.Palier)), ev.StretchRatio, ev.BaselineValue,
		duckdb.NullableStr(string(ev.Mode)), duckdb.NullableStr(string(ev.Cadence)),
		duckdb.NullableStr(string(ev.EvalType)), ev.TimeSinceCreateSeconds,
		duckdb.NullableStr(ev.Source), ev.CreatedAt,
	)
	return err
}

// ─────────── BaselineStateRepo ───────────

// PrestigeBaselineStateRepo implémente prestige.BaselineStateRepo.
type PrestigeBaselineStateRepo struct{ db duckdb.PlayerReadHandle }

func NewPrestigeBaselineStateRepo(db *duckdb.DB) *PrestigeBaselineStateRepo {
	return &PrestigeBaselineStateRepo{db: duckdb.NewPlayerReadHandle(db)}
}

var _ prestige.BaselineStateRepo = (*PrestigeBaselineStateRepo)(nil)

func (r *PrestigeBaselineStateRepo) Get(ctx context.Context, userID, titleSlug, metric string) (prestige.BaselineState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT user_id, title_slug, metric, last_match_at, is_stale,
		       recovery_matches_remaining, updated_at
		FROM baseline_state WHERE user_id = ? AND title_slug = ? AND metric = ?
	`, userID, titleSlug, metric)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.BaselineState{
			UserID: userID, TitleSlug: titleSlug, Metric: metric,
		}, nil
	}
	if err != nil {
		return prestige.BaselineState{}, err
	}
	defer rows.Close()
	var st prestige.BaselineState
	err = rows.Scan(&st.UserID, &st.TitleSlug, &st.Metric, &st.LastMatchAt,
		&st.IsStale, &st.RecoveryMatchesRemaining, &st.UpdatedAt)
	return st, err
}

func (r *PrestigeBaselineStateRepo) Upsert(ctx context.Context, st prestige.BaselineState) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT, qui réécrit via
	// l'index ART de la PK). État "dernière valeur observée" sans index secondaire →
	// l'UPDATE ne touche aucune colonne indexée.
	return r.db.UpsertNoConflict(ctx,
		`SELECT 1 FROM baseline_state WHERE user_id = ? AND title_slug = ? AND metric = ?`,
		[]any{st.UserID, st.TitleSlug, st.Metric},
		`UPDATE baseline_state SET last_match_at = ?, is_stale = ?, recovery_matches_remaining = ?, updated_at = ?
		 WHERE user_id = ? AND title_slug = ? AND metric = ?`,
		[]any{st.LastMatchAt, st.IsStale, st.RecoveryMatchesRemaining, st.UpdatedAt, st.UserID, st.TitleSlug, st.Metric},
		`INSERT INTO baseline_state (user_id, title_slug, metric, last_match_at, is_stale, recovery_matches_remaining, updated_at)
		 VALUES (?,?,?,?,?,?,?)`,
		[]any{st.UserID, st.TitleSlug, st.Metric, st.LastMatchAt, st.IsStale, st.RecoveryMatchesRemaining, st.UpdatedAt},
	)
}
