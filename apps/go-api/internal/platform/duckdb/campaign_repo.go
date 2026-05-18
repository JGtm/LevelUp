package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"levelup/go-api/internal/campaign"
)

// campaign_repo.go — accès DuckDB pour ImprovementCampaign (V1 §4.5).
//
// Toutes les méthodes acceptent un context et utilisent timeout court.

// CampaignRepo implémente l'accès stats.duckdb pour les campagnes d'amélioration.
type CampaignRepo struct{ db *DB }

// NewCampaignRepo construit le repo depuis une connexion Player.
func NewCampaignRepo(db *DB) *CampaignRepo { return &CampaignRepo{db: db} }

const campaignSelectColumns = `
	SELECT
		id, user_id, title_slug, axis, axis_kind, started_at, ended_at,
		status, playlist_group, snapshot_value, snapshot_sample,
		current_value_raw, current_value_lowess, matches_since_start,
		last_evaluated_at, mann_whitney_p, progression_confirmed,
		auto_closure_suggested, COALESCE(auto_closure_reason, '')
	FROM improvement_campaign`

// Insert persiste une nouvelle campagne. Échoue si l'ID existe déjà.
func (r *CampaignRepo) Insert(ctx context.Context, c campaign.ImprovementCampaign) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		INSERT INTO improvement_campaign (
			id, user_id, title_slug, axis, axis_kind, started_at,
			status, playlist_group, snapshot_value, snapshot_sample,
			matches_since_start, progression_confirmed, auto_closure_suggested
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`,
		c.ID, c.UserID, c.TitleSlug, c.Axis, string(c.AxisKind), c.StartedAt,
		string(c.Status), c.PlaylistGroup, c.SnapshotValue, c.SnapshotSample,
		c.MatchesSinceStart, c.ProgressionConfirmed, c.AutoClosureSuggested,
	)
	return err
}

// GetByID lit une campagne par ID.
func (r *CampaignRepo) GetByID(ctx context.Context, id string) (campaign.ImprovementCampaign, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, campaignSelectColumns+" WHERE id = ?", id)
	c, err := scanCampaign(row)
	if errors.Is(err, sql.ErrNoRows) {
		return campaign.ImprovementCampaign{}, campaign.ErrNotFound
	}
	return c, err
}

// GetActive retourne la campagne active du joueur (1 max attendue par titre).
// Retourne ErrNotFound si aucune campagne active.
func (r *CampaignRepo) GetActive(ctx context.Context, userID, titleSlug string) (campaign.ImprovementCampaign, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	row := r.db.QueryRow(ctx, campaignSelectColumns+`
		WHERE user_id = ? AND title_slug = ? AND status = 'active'
		ORDER BY started_at DESC LIMIT 1
	`, userID, titleSlug)
	c, err := scanCampaign(row)
	if errors.Is(err, sql.ErrNoRows) {
		return campaign.ImprovementCampaign{}, campaign.ErrNotFound
	}
	return c, err
}

// UpdateStatus change le status (+ ended_at si terminé).
func (r *CampaignRepo) UpdateStatus(ctx context.Context, id string, status campaign.CampaignStatus, endedAt *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		UPDATE improvement_campaign
		SET status = ?, ended_at = ?
		WHERE id = ?
	`, string(status), endedAt, id)
	return err
}

// UpdateEvaluation persiste l'état recalculé après EvaluateCampaign.
func (r *CampaignRepo) UpdateEvaluation(ctx context.Context, id string, eval campaign.Evaluation) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `
		UPDATE improvement_campaign
		SET current_value_raw = ?, current_value_lowess = ?,
		    matches_since_start = ?, last_evaluated_at = ?,
		    mann_whitney_p = ?, progression_confirmed = ?,
		    auto_closure_suggested = ?, auto_closure_reason = ?
		WHERE id = ?
	`,
		eval.CurrentRaw, eval.CurrentLOWESS,
		eval.MatchesSinceStart, eval.EvaluatedAt,
		eval.MannWhitneyP, eval.ProgressionConfirmed,
		eval.AutoClosureSuggested, eval.AutoClosureReason,
		id,
	)
	return err
}

// LinkedChallengeIDs retourne les IDs des défis liés à la campagne.
func (r *CampaignRepo) LinkedChallengeIDs(ctx context.Context, campaignID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.db.Query(ctx, `
		SELECT id FROM challenge WHERE campaign_id = ? ORDER BY created_at DESC
	`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LinkChallenge tague un challenge avec un campaign_id.
// Si campaignID = "" → unlink (set NULL).
func (r *CampaignRepo) LinkChallenge(ctx context.Context, challengeID, campaignID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if campaignID == "" {
		_, err := r.db.Exec(ctx, `UPDATE challenge SET campaign_id = NULL WHERE id = ?`, challengeID)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE challenge SET campaign_id = ? WHERE id = ?`, campaignID, challengeID)
	return err
}

// scanCampaign convertit une ligne en ImprovementCampaign. Helper interne.
func scanCampaign(row rowScanner) (campaign.ImprovementCampaign, error) {
	var c campaign.ImprovementCampaign
	var (
		axisKind, status                       string
		endedAt, lastEvaluatedAt               sql.NullTime
		currentRaw, currentLOWESS, mwp         sql.NullFloat64
		autoReason                             string
	)
	err := row.Scan(
		&c.ID, &c.UserID, &c.TitleSlug, &c.Axis, &axisKind, &c.StartedAt, &endedAt,
		&status, &c.PlaylistGroup, &c.SnapshotValue, &c.SnapshotSample,
		&currentRaw, &currentLOWESS, &c.MatchesSinceStart,
		&lastEvaluatedAt, &mwp, &c.ProgressionConfirmed,
		&c.AutoClosureSuggested, &autoReason,
	)
	if err != nil {
		return c, err
	}
	c.AxisKind = campaign.AxisKind(axisKind)
	c.Status = campaign.CampaignStatus(status)
	if endedAt.Valid {
		t := endedAt.Time
		c.EndedAt = &t
	}
	if lastEvaluatedAt.Valid {
		t := lastEvaluatedAt.Time
		c.LastEvaluatedAt = &t
	}
	if currentRaw.Valid {
		v := currentRaw.Float64
		c.CurrentValueRaw = &v
	}
	if currentLOWESS.Valid {
		v := currentLOWESS.Float64
		c.CurrentValueLOWESS = &v
	}
	if mwp.Valid {
		v := mwp.Float64
		c.MannWhitneyP = &v
	}
	c.AutoClosureReason = strings.TrimSpace(autoReason)
	return c, nil
}

// ─── SampleProvider ────────────────────────────────────────────────────────

// CampaignSampleProvider implémente campaign.SampleProvider en lisant
// match_participants (shared) joint à match_registry.
//
// Mapping V1 axis → expression SQL :
//
//   - radar.combat          : kills
//   - radar.survival        : kills - deaths (clampé à 0)
//   - radar.support         : assists
//   - radar.score           : personal_score
//   - radar.objective       : 0 (V1 — mapping award→axis title-specific reporté)
//   - radar.impact          : max_killing_spree * 10
//   - lusr_component.*      : 0 (V1 — table lusr_component_history pas créée,
//     même note que profile.loadLUSRComponentsBreakdown)
type CampaignSampleProvider struct{ db *DB }

// NewCampaignSampleProvider construit le provider sur stats.duckdb.
func NewCampaignSampleProvider(db *DB) *CampaignSampleProvider {
	return &CampaignSampleProvider{db: db}
}

// LoadAxisSamples charge les valeurs de l'axe pour les matchs du joueur dans
// la fenêtre, filtré par playlist_group si != "all".
func (p *CampaignSampleProvider) LoadAxisSamples(
	ctx context.Context,
	userID, _titleSlug, axis string, axisKind campaign.AxisKind,
	playlistGroup string,
	since, until time.Time,
) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	expr, supported := axisValueExpression(axis, axisKind)
	if !supported {
		// V1 : axe non supporté → samples vides (ne crash pas la campagne).
		return nil, nil
	}
	q := "SELECT " + expr + ` AS val
		FROM shared.match_participants mp
		JOIN shared.match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time >= ? AND mr.start_time <= ?`
	args := []any{userID, since, until}
	if playlistGroup != "" && playlistGroup != "all" {
		q += ` AND mr.playlist_id = ?`
		args = append(args, playlistGroup)
	}
	q += ` ORDER BY mr.start_time ASC`
	rows, err := p.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v sql.NullFloat64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			out = append(out, v.Float64)
		}
	}
	return out, rows.Err()
}

// axisValueExpression mappe (axis, kind) → expression SQL sur
// match_participants. V1 = heuristique pragmatique. V2 = lecture depuis
// personal_score_awards / lusr_component_history.
func axisValueExpression(axis string, kind campaign.AxisKind) (string, bool) {
	if kind == campaign.AxisKindRadar {
		switch axis {
		case "combat":
			return "CAST(mp.kills AS DOUBLE)", true
		case "survival":
			return "GREATEST(CAST(mp.kills AS DOUBLE) - CAST(mp.deaths AS DOUBLE), 0)", true
		case "support":
			return "CAST(mp.assists AS DOUBLE)", true
		case "score":
			return "CAST(mp.personal_score AS DOUBLE)", true
		case "impact":
			return "CAST(mp.max_killing_spree AS DOUBLE) * 10", true
		case "objective":
			return "0.0", true // placeholder V1 — mapping awards→axis reporté
		}
	}
	// lusr_component : placeholder V1 (table history pas créée).
	if kind == campaign.AxisKindLUSRComponent {
		return "0.0", true
	}
	return "", false
}
