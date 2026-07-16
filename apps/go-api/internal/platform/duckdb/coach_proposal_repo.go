// Package duckdb — coach_proposal_repo.go : proposals du coach_advisor
// (table `coach_proposal` dans stats.duckdb par joueur). Cf. ADR 0020 Phase 3.
//
// Pattern d'écriture (cf. CLAUDE.md §"Phase 4 ART") : INSERT pour Create,
// UPDATE explicite pour les transitions de status. Pas d'ON CONFLICT DO UPDATE
// sur colonnes mutables.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

const (
	coachProposalReadTimeout  = 10 * time.Second
	coachProposalWriteTimeout = 5 * time.Second
)

// CoachProposalRepo persiste les proposals du coach_advisor (stats.duckdb).
type CoachProposalRepo struct {
	db *DB
}

// NewCoachProposalRepo construit le repo. db doit pointer sur la stats.duckdb
// du joueur courant.
func NewCoachProposalRepo(db *DB) *CoachProposalRepo {
	return &CoachProposalRepo{db: db}
}

// Compile-time assertion.
var _ coach_advisor.Repo = (*CoachProposalRepo)(nil)

// Create insère une nouvelle proposal. Si Status est vide, applique 'pending'.
func (r *CoachProposalRepo) Create(ctx context.Context, p coach_advisor.Proposal) error {
	ctx, cancel := context.WithTimeout(ctx, coachProposalWriteTimeout)
	defer cancel()

	if p.Status == "" {
		p.Status = coach_advisor.ProposalPending
	}
	createdAt := p.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO coach_proposal (
			id, user_id, title_slug, kind,
			template_id, challenges_spec_json, suggested_tier,
			source_signal, source_metric, radar_axis, strength, origin,
			reason_key_en, reason_key_fr, reason_params,
			status, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID, p.UserID, p.TitleSlug, string(p.Kind),
		nullableString(p.TemplateID), nullableString(p.ChallengesSpec), nullableString(string(p.SuggestedTier)),
		string(p.SourceSignal), nullableString(p.SourceMetric), nullableString(p.RadarAxis), p.Strength, string(p.Origin),
		nullableString(p.ReasonKeyEN), nullableString(p.ReasonKeyFR), nullableString(p.ReasonParams),
		string(p.Status), createdAt, nullableExpiresAt(p.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("CoachProposalRepo.Create: %w", err)
	}
	return nil
}

// Get retourne une proposal par id, ErrProposalNotFound si absent.
func (r *CoachProposalRepo) Get(ctx context.Context, id string) (coach_advisor.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, coachProposalReadTimeout)
	defer cancel()

	rows, err := r.db.QueryRowRecovered(ctx, baseSelectCoachProposal+" WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return coach_advisor.Proposal{}, coach_advisor.ErrProposalNotFound
	}
	if err != nil {
		return coach_advisor.Proposal{}, fmt.Errorf("CoachProposalRepo.Get: %w", err)
	}
	defer rows.Close()
	p, err := scanCoachProposal(rows)
	if err != nil {
		return coach_advisor.Proposal{}, fmt.Errorf("CoachProposalRepo.Get: %w", err)
	}
	return p, nil
}

// ListByUser retourne les proposals filtrées par status (vide = tous).
func (r *CoachProposalRepo) ListByUser(ctx context.Context, userID, titleSlug string, status coach_advisor.ProposalStatus) ([]coach_advisor.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, coachProposalReadTimeout)
	defer cancel()

	q := baseSelectCoachProposal + " WHERE user_id = ? AND title_slug = ?"
	args := []any{userID, titleSlug}
	if status != "" {
		q += " AND status = ?"
		args = append(args, string(status))
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.db.QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("CoachProposalRepo.ListByUser: %w", err)
	}
	defer rows.Close()
	return scanCoachProposalRows(rows)
}

// ListPendingBySignalScope retourne les pending qui matchent (metric, axis).
// Au moins l'un des deux paramètres doit matcher (OR logique) — supersession.
func (r *CoachProposalRepo) ListPendingBySignalScope(ctx context.Context, userID, titleSlug, metric, axis string) ([]coach_advisor.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, coachProposalReadTimeout)
	defer cancel()

	rows, err := r.db.QueryRecovered(ctx, baseSelectCoachProposal+`
		WHERE user_id = ?
		  AND title_slug = ?
		  AND status = 'pending'
		  AND (source_metric = ? OR radar_axis = ?)
		ORDER BY created_at DESC
	`, userID, titleSlug, metric, axis)
	if err != nil {
		return nil, fmt.Errorf("CoachProposalRepo.ListPendingBySignalScope: %w", err)
	}
	defer rows.Close()
	return scanCoachProposalRows(rows)
}

// ListPendingByAxis retourne les pending sur ce radar_axis (obsolescence).
func (r *CoachProposalRepo) ListPendingByAxis(ctx context.Context, userID, titleSlug, axis string) ([]coach_advisor.Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, coachProposalReadTimeout)
	defer cancel()

	rows, err := r.db.QueryRecovered(ctx, baseSelectCoachProposal+`
		WHERE user_id = ?
		  AND title_slug = ?
		  AND status = 'pending'
		  AND radar_axis = ?
		ORDER BY created_at DESC
	`, userID, titleSlug, axis)
	if err != nil {
		return nil, fmt.Errorf("CoachProposalRepo.ListPendingByAxis: %w", err)
	}
	defer rows.Close()
	return scanCoachProposalRows(rows)
}

// MarkAccepted positionne status='accepted', resolved_at=now, resolved_ref=ref.
func (r *CoachProposalRepo) MarkAccepted(ctx context.Context, id, ref string, now time.Time) error {
	return r.updateStatus(ctx, id, "MarkAccepted", `
		UPDATE coach_proposal
		SET status = 'accepted', resolved_at = ?, resolved_ref = ?
		WHERE id = ?
	`, now.UTC(), ref, id)
}

// MarkDismissed positionne status='dismissed', resolved_at=now.
func (r *CoachProposalRepo) MarkDismissed(ctx context.Context, id string, now time.Time) error {
	return r.updateStatus(ctx, id, "MarkDismissed", `
		UPDATE coach_proposal
		SET status = 'dismissed', resolved_at = ?
		WHERE id = ?
	`, now.UTC(), id)
}

// MarkSuperseded positionne status='superseded', superseded_at=now, superseded_by=newID.
// Idempotent : aucun effet si déjà superseded.
func (r *CoachProposalRepo) MarkSuperseded(ctx context.Context, id, newID string, now time.Time) error {
	return r.updateStatus(ctx, id, "MarkSuperseded", `
		UPDATE coach_proposal
		SET status = 'superseded', superseded_at = ?, superseded_by = ?
		WHERE id = ? AND status = 'pending'
	`, now.UTC(), newID, id)
}

// MarkObsoleted positionne status='obsoleted', obsoleted_at=now.
func (r *CoachProposalRepo) MarkObsoleted(ctx context.Context, id string, now time.Time) error {
	return r.updateStatus(ctx, id, "MarkObsoleted", `
		UPDATE coach_proposal
		SET status = 'obsoleted', obsoleted_at = ?
		WHERE id = ? AND status = 'pending'
	`, now.UTC(), id)
}

func (r *CoachProposalRepo) updateStatus(ctx context.Context, id, op, query string, args ...any) error {
	ctx, cancel := context.WithTimeout(ctx, coachProposalWriteTimeout)
	defer cancel()
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("CoachProposalRepo.%s: %w", op, err)
	}
	return nil
}

const baseSelectCoachProposal = `
	SELECT id, user_id, title_slug, kind,
	       template_id, challenges_spec_json, suggested_tier,
	       source_signal, source_metric, radar_axis, strength, origin,
	       reason_key_en, reason_key_fr, reason_params,
	       status, created_at, expires_at, resolved_at, resolved_ref,
	       superseded_by, superseded_at, obsoleted_at
	FROM coach_proposal
`

// scanCoachProposal lit une ligne (RowScanner défini dans sql_scan_helpers.go).
func scanCoachProposal(s RowScanner) (coach_advisor.Proposal, error) {
	var (
		p              coach_advisor.Proposal
		kind           string
		templateID     sql.NullString
		challengesSpec sql.NullString
		suggestedTier  sql.NullString
		sourceMetric   sql.NullString
		radarAxis      sql.NullString
		origin         string
		reasonKeyEN    sql.NullString
		reasonKeyFR    sql.NullString
		reasonParams   sql.NullString
		status         string
		expiresAt      sql.NullTime
		resolvedAt     sql.NullTime
		resolvedRef    sql.NullString
		supersededBy   sql.NullString
		supersededAt   sql.NullTime
		obsoletedAt    sql.NullTime
		sourceSignal   string
	)
	err := s.Scan(
		&p.ID, &p.UserID, &p.TitleSlug, &kind,
		&templateID, &challengesSpec, &suggestedTier,
		&sourceSignal, &sourceMetric, &radarAxis, &p.Strength, &origin,
		&reasonKeyEN, &reasonKeyFR, &reasonParams,
		&status, &p.CreatedAt, &expiresAt, &resolvedAt, &resolvedRef,
		&supersededBy, &supersededAt, &obsoletedAt,
	)
	if err != nil {
		return p, err
	}
	p.Kind = coach_advisor.ProposalKind(kind)
	p.TemplateID = templateID.String
	p.ChallengesSpec = challengesSpec.String
	p.SuggestedTier = prestige.Tier(suggestedTier.String)
	p.SourceSignal = coach_advisor.SignalKind(sourceSignal)
	p.SourceMetric = sourceMetric.String
	p.RadarAxis = radarAxis.String
	p.Origin = coach_advisor.ProposalOrigin(origin)
	p.ReasonKeyEN = reasonKeyEN.String
	p.ReasonKeyFR = reasonKeyFR.String
	p.ReasonParams = reasonParams.String
	p.Status = coach_advisor.ProposalStatus(status)
	if expiresAt.Valid {
		t := expiresAt.Time
		p.ExpiresAt = &t
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		p.ResolvedAt = &t
	}
	p.ResolvedRef = resolvedRef.String
	p.SupersededBy = supersededBy.String
	if supersededAt.Valid {
		t := supersededAt.Time
		p.SupersededAt = &t
	}
	if obsoletedAt.Valid {
		t := obsoletedAt.Time
		p.ObsoletedAt = &t
	}
	return p, nil
}

func scanCoachProposalRows(rows *sql.Rows) ([]coach_advisor.Proposal, error) {
	var out []coach_advisor.Proposal
	for rows.Next() {
		p, err := scanCoachProposal(rows)
		if err != nil {
			return nil, fmt.Errorf("CoachProposalRepo: scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// nullableExpiresAt convertit *time.Time en sql.NullTime UTC (NULL si nil).
// (Existe en plus de nullableTime de streaks_repo.go car le ExpiresAt doit être
// stocké en UTC pour cohérence — pas une simple recopie du pointer.)
func nullableExpiresAt(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}
