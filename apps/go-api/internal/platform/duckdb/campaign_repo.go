package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/campaign"
)

// campaign_repo.go — accès DuckDB pour ImprovementCampaign (V1 §4.5).
//
// Toutes les méthodes acceptent un context et utilisent timeout court.

// CampaignRepo implémente l'accès stats.duckdb pour les campagnes d'amélioration.
type CampaignRepo struct{ db PlayerReadHandle }

// NewCampaignRepo construit le repo depuis une connexion Player.
func NewCampaignRepo(db *DB) *CampaignRepo { return &CampaignRepo{db: NewPlayerReadHandle(db)} }

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
	_, err := r.db.ExecRecovered(ctx, `
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
	rows, err := r.db.QueryRowRecovered(ctx, campaignSelectColumns+" WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return campaign.ImprovementCampaign{}, campaign.ErrNotFound
	}
	if err != nil {
		return campaign.ImprovementCampaign{}, err
	}
	defer rows.Close()
	return scanCampaign(rows)
}

// GetActive retourne la campagne active du joueur (1 max attendue par titre).
// Retourne ErrNotFound si aucune campagne active.
func (r *CampaignRepo) GetActive(ctx context.Context, userID, titleSlug string) (campaign.ImprovementCampaign, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryRowRecovered(ctx, campaignSelectColumns+`
		WHERE user_id = ? AND title_slug = ? AND status = 'active'
		ORDER BY started_at DESC LIMIT 1
	`, userID, titleSlug)
	if errors.Is(err, sql.ErrNoRows) {
		return campaign.ImprovementCampaign{}, campaign.ErrNotFound
	}
	if err != nil {
		return campaign.ImprovementCampaign{}, err
	}
	defer rows.Close()
	return scanCampaign(rows)
}

// ListEnded liste les campagnes closes (completed/abandoned) du joueur sur un
// titre, les plus récentes d'abord. Scopé user_id + title_slug ; tri par
// ended_at desc (NULLS LAST par sécurité — une campagne close a normalement un
// ended_at renseigné).
func (r *CampaignRepo) ListEnded(ctx context.Context, userID, titleSlug string) ([]campaign.ImprovementCampaign, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, campaignSelectColumns+`
		WHERE user_id = ? AND title_slug = ?
		  AND status IN ('completed', 'abandoned')
		ORDER BY ended_at DESC NULLS LAST, started_at DESC
	`, userID, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []campaign.ImprovementCampaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateStatus change le status (+ ended_at si terminé).
func (r *CampaignRepo) UpdateStatus(ctx context.Context, id string, status campaign.CampaignStatus, endedAt *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.db.ExecRecovered(ctx, `
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
	_, err := r.db.ExecRecovered(ctx, `
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
	rows, err := r.db.QueryRecovered(ctx, `
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
		_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET campaign_id = NULL WHERE id = ?`, challengeID)
		return err
	}
	_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET campaign_id = ? WHERE id = ?`, campaignID, challengeID)
	return err
}

// scanCampaign convertit une ligne en ImprovementCampaign. Helper interne.
func scanCampaign(row RowScanner) (campaign.ImprovementCampaign, error) {
	var c campaign.ImprovementCampaign
	var (
		axisKind, status               string
		endedAt, lastEvaluatedAt       sql.NullTime
		currentRaw, currentLOWESS, mwp sql.NullFloat64
		autoReason                     string
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
// reçoit un *PlayerDB pour accéder à SharedReader
// (queries shared coordonnées avec le SharedDBProvider) et à pdb.Player
// (lusr_component_history).
//
// Mapping V1 axis → expression SQL :
//
//   - radar.combat          : kills
//   - radar.survival        : kills - deaths (clampé à 0)
//   - radar.support         : assists
//   - radar.score           : personal_score
//   - radar.objective       : 0 (V1 — mapping award→axis title-specific reporté)
//   - radar.impact          : max_killing_spree * 10
//   - lusr_component.*      : lusr_component_history (PLAYER) ⨝ match_registry (SHARED)
type CampaignSampleProvider struct{ pdb *PlayerDB }

// NewCampaignSampleProvider construit le provider depuis un *PlayerDB.
func NewCampaignSampleProvider(pdb *PlayerDB) *CampaignSampleProvider {
	return &CampaignSampleProvider{pdb: pdb}
}

// LoadAxisSamples charge les valeurs de l'axe pour les matchs du joueur dans
// la fenêtre, filtré par playlist_group si != "all".
//
// Pour les axes `lusr_component.*` (V2 §1) : lit `lusr_component_history` au
// lieu de `match_participants`. Permet le suivi fiable des 8 composantes
// LUSR (déjà normalisées [0,1] par le scoring engine).
func (p *CampaignSampleProvider) LoadAxisSamples(
	ctx context.Context,
	userID, _titleSlug, axis string, axisKind campaign.AxisKind,
	playlistGroup string,
	since, until time.Time,
) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if axisKind == campaign.AxisKindLUSRComponent {
		return p.loadLUSRComponentSamples(ctx, axis, playlistGroup, since, until)
	}

	expr, supported := axisValueExpression(axis, axisKind)
	if !supported {
		return nil, nil
	}
	q := "SELECT " + expr + ` AS val
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND mr.start_time >= ? AND mr.start_time <= ?`
	args := []any{userID, since, until}
	if playlistGroup != "" && playlistGroup != "all" {
		q += ` AND mr.playlist_id = ?`
		args = append(args, playlistGroup)
	}
	q += ` ORDER BY mr.start_time ASC`

	db, release, err := p.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
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

// loadLUSRComponentSamples lit la chronologie d'une composante LUSR depuis
// `lusr_component_history` (table joueur), joint à `match_registry` (shared)
// pour le filtre playlist et la chronologie.
//
// split cross-DB.
//
//	Étape 1 (SharedReader) : SELECT match_id, start_time FROM match_registry
//	  WHERE start_time IN window [AND playlist_id = ?]
//	Étape 2 (pdb.Player) : SELECT match_id, value FROM lusr_component_history_latest
//	  WHERE component_name = ? AND match_id IN (...)
//	Étape 3 (Go) : merge en respectant l'ordre chronologique de l'étape 1.
//
// lusrMatchTS : (match_id, start_time) pour ordre chronologique.
type lusrMatchTS struct {
	matchID string
	ts      time.Time
}

func (p *CampaignSampleProvider) loadLUSRComponentSamples(
	ctx context.Context,
	component, playlistGroup string,
	since, until time.Time,
) ([]float64, error) {
	matches, err := p.loadLUSRMatchIDsInWindow(ctx, component, playlistGroup, since, until)
	if err != nil || len(matches) == 0 {
		return nil, err
	}

	matchIDs := make([]string, 0, len(matches))
	for _, m := range matches {
		matchIDs = append(matchIDs, m.matchID)
	}
	values, err := p.loadLUSRValuesByMatch(ctx, component, matchIDs)
	if err != nil {
		return nil, err
	}

	// Ressort selon l'ordre chronologique des matchs.
	var out []float64
	for _, m := range matches {
		if v, ok := values[m.matchID]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// loadLUSRMatchIDsInWindow charge les match_ids ordonnés chronologiquement depuis shared.
func (p *CampaignSampleProvider) loadLUSRMatchIDsInWindow(
	ctx context.Context, component, playlistGroup string, since, until time.Time,
) ([]lusrMatchTS, error) {
	sharedQ := `
		SELECT match_id, start_time
		FROM match_registry
		WHERE start_time >= ? AND start_time <= ?
	`
	sharedArgs := []any{since, until}
	if playlistGroup != "" && playlistGroup != "all" {
		sharedQ += ` AND playlist_id = ?`
		sharedArgs = append(sharedArgs, playlistGroup)
	}
	sharedQ += ` ORDER BY start_time ASC`

	sharedDB, release, err := p.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "campaign: loadLUSRComponentSamples shared reader indisponible (best-effort, retour vide)",
			"component", component, "err", err)
		return nil, nil //nolint:nilerr
	}
	defer release()

	sharedRows, err := sharedDB.QueryContext(ctx, sharedQ, sharedArgs...)
	if err != nil {
		slog.WarnContext(ctx, "campaign: loadLUSRComponentSamples shared query failed (best-effort, retour vide)",
			"component", component, "err", err)
		return nil, nil //nolint:nilerr
	}
	defer sharedRows.Close()
	var matches []lusrMatchTS
	for sharedRows.Next() {
		var m lusrMatchTS
		if err := sharedRows.Scan(&m.matchID, &m.ts); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// loadLUSRValuesByMatch charge map[match_id]value depuis player.lusr_component_history.
func (p *CampaignSampleProvider) loadLUSRValuesByMatch(
	ctx context.Context, component string, matchIDs []string,
) (map[string]float64, error) {
	playerQ := fmt.Sprintf(`
		SELECT match_id, value
		FROM lusr_component_history_latest
		WHERE component_name = ? AND match_id IN (%s)`, Placeholders(len(matchIDs)))
	args := make([]any, 0, 1+len(matchIDs))
	args = append(args, component)
	args = append(args, ToAnySlice(matchIDs)...)
	rows, err := p.pdb.Player.QueryRecovered(ctx, playerQ, args...)
	if err != nil {
		slog.WarnContext(ctx, "campaign: loadLUSRComponentSamples player query failed (best-effort, retour vide)",
			"component", component, "match_count", len(matchIDs), "err", err)
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	values := make(map[string]float64, len(matchIDs))
	for rows.Next() {
		var mid string
		var v sql.NullFloat64
		if err := rows.Scan(&mid, &v); err != nil {
			return nil, err
		}
		if v.Valid {
			values[mid] = v.Float64
		}
	}
	return values, rows.Err()
}

// axisValueExpression mappe (axis, kind=radar) → expression SQL sur
// match_participants. V1 = heuristique pragmatique pour les axes radar.
// Le mapping fin via awards.toml arrive au V2 commit-2.
//
// Note : kind=lusr_component est traité séparément par loadLUSRComponentSamples
// qui lit `lusr_component_history` directement (V2 commit-1).
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
			return "0.0", true // placeholder V1 — mapping awards→axis arrive V2 commit-2
		}
	}
	return "", false
}
