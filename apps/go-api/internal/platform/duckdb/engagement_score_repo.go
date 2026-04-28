// Package duckdb — engagement_score_repo.go : implementation DuckDB de
// port.EngagementScoreRepository.
//
// Persistence repartie :
//   - Score par match (0-100, residu brut, confidence) : colonnes dans
//     player.player_match_enrichment (engagement_score, engagement_score_brut,
//     engagement_score_confidence)
//   - Intensity match (caracteristique objective) : colonne match_intensity
//     dans shared.match_registry
//   - Coefficients perso (team_share, lobby_share) : table dediee
//     engagement_coefficients dans la player DB
//
// La migration Phase 2 du plan ajoute ces colonnes/tables. Tant qu'elle n'est
// pas appliquee, les methodes detectent l'absence via information_schema et
// retournent port.ErrEngagementUnavailable.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// EngagementScoreRepo implemente port.EngagementScoreRepository.
type EngagementScoreRepo struct {
	pdb *PlayerDB
}

// NewEngagementScoreRepo cree un repo lie a un PlayerDB.
func NewEngagementScoreRepo(pdb *PlayerDB) *EngagementScoreRepo {
	return &EngagementScoreRepo{pdb: pdb}
}

// LoadPlayerHistory charge les residus bruts des N derniers matchs du joueur
// sur la categorie de mode demandee, ordre chronologique decroissant.
func (r *EngagementScoreRepo) LoadPlayerHistory(
	ctx context.Context,
	filter port.EngagementHistoryFilter,
) ([]domain.HistoricalEngagementBrut, error) {
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadPlayerHistory: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if !r.engagementColumnsExist(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	q, args := buildEngagementHistoryQuery(filter)
	rows, err := r.pdb.ReadDB().Query(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "EngagementScoreRepo: history query failed",
			"xuid", filter.XUID, "mode", filter.ModeCategory, "err", err)
		return nil, fmt.Errorf("EngagementScoreRepo.LoadPlayerHistory: query: %w", err)
	}
	defer rows.Close()

	out := make([]domain.HistoricalEngagementBrut, 0, filter.Limit)
	for rows.Next() {
		var h domain.HistoricalEngagementBrut
		if err := rows.Scan(&h.MatchID, &h.Brut); err != nil {
			return nil, fmt.Errorf("EngagementScoreRepo.LoadPlayerHistory: scan: %w", err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadPlayerHistory: rows: %w", err)
	}
	return out, nil
}

// LoadEngagementCoefficient charge le couple (CoefTeamShare, CoefLobbyShare)
// stocke pour (xuid, mode_category). Retourne (nil, nil) si non present.
func (r *EngagementScoreRepo) LoadEngagementCoefficient(
	ctx context.Context,
	xuid, modeCategory string,
) (*domain.EngagementCoefficient, error) {
	if xuid == "" || modeCategory == "" {
		return nil, errors.New("EngagementScoreRepo.LoadEngagementCoefficient: xuid and modeCategory required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.coefficientsTableExists(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT coef_team_share, coef_lobby_share, n_matches, last_updated
		FROM engagement_coefficients
		WHERE xuid = ? AND mode_category = ?
	`
	var coef domain.EngagementCoefficient
	coef.XUID = xuid
	coef.ModeCategory = modeCategory

	err := r.pdb.ReadDB().QueryRow(ctx, q, xuid, modeCategory).Scan(
		&coef.CoefTeamShare,
		&coef.CoefLobbyShare,
		&coef.NMatches,
		&coef.LastUpdated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadEngagementCoefficient: %w", err)
	}
	return &coef, nil
}

// SaveEngagementScore persiste le score, le residu brut et la confidence
// dans player_match_enrichment (UPDATE).
//
// Si EngagementScore est nil (cas insufficient_history), on persiste le
// residu et la confidence mais on laisse engagement_score a NULL.
func (r *EngagementScoreRepo) SaveEngagementScore(
	ctx context.Context,
	xuid, matchID string,
	result domain.EngagementScoreResult,
) error {
	if xuid == "" || matchID == "" {
		return errors.New("EngagementScoreRepo.SaveEngagementScore: xuid and matchID required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.engagementColumnsExist(ctx) {
		return port.ErrEngagementUnavailable
	}

	const q = `
		UPDATE player_match_enrichment
		SET
			engagement_score = ?,
			engagement_score_brut = ?,
			engagement_score_confidence = ?
		WHERE xuid = ? AND match_id = ?
	`
	var scoreArg any
	if result.EngagementScore != nil {
		scoreArg = *result.EngagementScore
	}

	res, err := r.pdb.Player.Exec(ctx, q,
		scoreArg,
		result.ResidualBrut,
		result.Confidence,
		xuid,
		matchID,
	)
	if err != nil {
		return fmt.Errorf("EngagementScoreRepo.SaveEngagementScore: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		// Pas d'enrichment row pour ce match — phenomene attendu pour les
		// matchs non-encore enrichis. Le sync createra la row puis ressayera.
		slog.DebugContext(ctx, "EngagementScoreRepo: no enrichment row updated",
			"xuid", xuid, "match_id", matchID)
	}
	return nil
}

// SaveEngagementCoefficient persiste / met a jour les coefs (UPSERT).
func (r *EngagementScoreRepo) SaveEngagementCoefficient(
	ctx context.Context,
	coef domain.EngagementCoefficient,
) error {
	if coef.XUID == "" || coef.ModeCategory == "" {
		return errors.New("EngagementScoreRepo.SaveEngagementCoefficient: XUID and ModeCategory required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.coefficientsTableExists(ctx) {
		return port.ErrEngagementUnavailable
	}

	// DuckDB supporte INSERT OR REPLACE via "ON CONFLICT DO UPDATE".
	const q = `
		INSERT INTO engagement_coefficients (
			xuid, mode_category, coef_team_share, coef_lobby_share,
			n_matches, last_updated
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (xuid, mode_category) DO UPDATE SET
			coef_team_share = EXCLUDED.coef_team_share,
			coef_lobby_share = EXCLUDED.coef_lobby_share,
			n_matches = EXCLUDED.n_matches,
			last_updated = EXCLUDED.last_updated
	`
	updated := coef.LastUpdated
	if updated.IsZero() {
		updated = time.Now().UTC()
	}

	_, err := r.pdb.Player.Exec(ctx, q,
		coef.XUID,
		coef.ModeCategory,
		coef.CoefTeamShare,
		coef.CoefLobbyShare,
		coef.NMatches,
		updated,
	)
	if err != nil {
		return fmt.Errorf("EngagementScoreRepo.SaveEngagementCoefficient: %w", err)
	}
	return nil
}

// SaveMatchIntensity persiste l'intensite d'un match dans shared.match_registry.
//
// Note : shared est attache en READ_ONLY sur la connexion player. L'ecriture
// passe par une connexion temporaire RW au shared DB. En pratique, le sync
// engine a deja une connexion RW au shared (cf sync/writes.go) ; ce repo est
// destine au scenario "calcul a la volee" lequel est rare. Pour le sync, le
// pipeline doit batcher les ecritures via sa propre connexion shared RW.
//
// Phase 1.5 : on documente la limitation et on retourne ErrEngagementUnavailable
// si la connexion shared n'est pas disponible en RW. L'integration sync (Phase 3)
// passera par le sync engine qui dispose de la connexion appropriee.
func (r *EngagementScoreRepo) SaveMatchIntensity(
	ctx context.Context,
	matchID string,
	intensity float64,
) error {
	if matchID == "" {
		return errors.New("EngagementScoreRepo.SaveMatchIntensity: matchID required")
	}

	// Sentinel : la persistence shared writes passe par sync engine.
	// Cf plan d'implementation Phase 3 "Integration sync/backfill".
	return port.ErrEngagementUnavailable
}

// LoadMatchIntensity lit l'intensite stockee dans shared.match_registry.
func (r *EngagementScoreRepo) LoadMatchIntensity(
	ctx context.Context,
	matchID string,
) (float64, bool, error) {
	if matchID == "" {
		return 0, false, errors.New("EngagementScoreRepo.LoadMatchIntensity: matchID required")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !r.matchIntensityColumnExists(ctx) {
		return 0, false, port.ErrEngagementUnavailable
	}

	var intensity sql.NullFloat64
	err := r.pdb.ReadDB().QueryRow(ctx,
		`SELECT match_intensity FROM shared.match_registry WHERE match_id = ?`,
		matchID,
	).Scan(&intensity)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("EngagementScoreRepo.LoadMatchIntensity: %w", err)
	}
	if !intensity.Valid {
		return 0, false, nil
	}
	return intensity.Float64, true, nil
}

// HasEngagementScore retourne true si un score (non NULL) est deja persiste
// pour (xuid, match_id). Permet au sync de skip les matchs deja traites.
func (r *EngagementScoreRepo) HasEngagementScore(
	ctx context.Context,
	xuid, matchID string,
) (bool, error) {
	if xuid == "" || matchID == "" {
		return false, errors.New("EngagementScoreRepo.HasEngagementScore: xuid and matchID required")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !r.engagementColumnsExist(ctx) {
		return false, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT engagement_score IS NOT NULL
		FROM player_match_enrichment
		WHERE xuid = ? AND match_id = ?
	`
	var has bool
	err := r.pdb.ReadDB().QueryRow(ctx, q, xuid, matchID).Scan(&has)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("EngagementScoreRepo.HasEngagementScore: %w", err)
	}
	return has, nil
}

// LoadMatchEngagementContext charge les metadonnees d'un match (Phase 4 API).
func (r *EngagementScoreRepo) LoadMatchEngagementContext(
	ctx context.Context,
	matchID, xuid string,
) (*port.MatchEngagementContext, error) {
	if matchID == "" || xuid == "" {
		return nil, errors.New("EngagementScoreRepo.LoadMatchEngagementContext: matchID and xuid required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT
			mr.match_id,
			COALESCE(EPOCH_MS(mr.start_time_utc), EPOCH_MS(mr.start_time)),
			COALESCE(EPOCH_MS(mr.end_time_utc), EPOCH_MS(mr.end_time)),
			COALESCE(mr.is_ranked, FALSE),
			COALESCE(mr.is_pve, FALSE),
			COALESCE(mp.team_id, 0),
			COALESCE(mp.personal_score, 0),
			COALESCE(mp.kills, 0),
			COALESCE(mp.assists, 0)
		FROM shared.match_registry mr
		JOIN shared.match_participants mp ON mr.match_id = mp.match_id
		WHERE mr.match_id = ? AND mp.xuid = ?
	`
	var mctx port.MatchEngagementContext
	err := r.pdb.ReadDB().QueryRow(ctx, q, matchID, xuid).Scan(
		&mctx.MatchID,
		&mctx.StartTimeMS,
		&mctx.EndTimeMS,
		&mctx.IsRanked,
		&mctx.IsPvE,
		&mctx.TargetTeamID,
		&mctx.PersonalScore,
		&mctx.Kills,
		&mctx.Assists,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("LoadMatchEngagementContext: %w", err)
	}

	// Charger NTeam et NHumansLobby separement.
	const sizeQ = `
		SELECT
			SUM(CASE WHEN team_id = ? AND COALESCE(is_bot, FALSE) = FALSE THEN 1 ELSE 0 END),
			SUM(CASE WHEN COALESCE(is_bot, FALSE) = FALSE THEN 1 ELSE 0 END)
		FROM shared.match_participants WHERE match_id = ?
	`
	var nTeam, nLobby sql.NullInt64
	_ = r.pdb.ReadDB().QueryRow(ctx, sizeQ, mctx.TargetTeamID, matchID).Scan(&nTeam, &nLobby)
	mctx.NTeam = int(nTeam.Int64)
	mctx.NHumansLobby = int(nLobby.Int64)
	mctx.IsTeamMode = mctx.NTeam > 1

	return &mctx, nil
}

// LoadEventsForMatch charge tous les events highlight_events d'un match.
func (r *EngagementScoreRepo) LoadEventsForMatch(
	ctx context.Context,
	matchID string,
) ([]canonical.HighlightEvent, error) {
	if matchID == "" {
		return nil, errors.New("EngagementScoreRepo.LoadEventsForMatch: matchID required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	const q = `
		SELECT match_id, event_type, COALESCE(time_ms, 0), COALESCE(xuid, '')
		FROM shared.highlight_events
		WHERE match_id = ?
		ORDER BY time_ms ASC
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, matchID)
	if err != nil {
		return nil, fmt.Errorf("LoadEventsForMatch: %w", err)
	}
	defer rows.Close()

	out := make([]canonical.HighlightEvent, 0)
	for rows.Next() {
		var ev canonical.HighlightEvent
		if err := rows.Scan(&ev.MatchID, &ev.EventType, &ev.TimeMS, &ev.XUID); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// LoadTeamXUIDs charge les XUIDs des coequipiers humains (joueur cible exclu).
func (r *EngagementScoreRepo) LoadTeamXUIDs(
	ctx context.Context,
	matchID string,
	teamID int,
	targetXUID string,
) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const q = `
		SELECT xuid FROM shared.match_participants
		WHERE match_id = ?
		  AND team_id = ?
		  AND COALESCE(is_bot, FALSE) = FALSE
		  AND xuid <> ?
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, matchID, teamID, targetXUID)
	if err != nil {
		return nil, fmt.Errorf("LoadTeamXUIDs: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err == nil {
			out[x] = true
		}
	}
	return out, rows.Err()
}

// LoadAllCoefficients charge tous les coefficients du joueur, toutes categories.
func (r *EngagementScoreRepo) LoadAllCoefficients(
	ctx context.Context,
	xuid string,
) ([]domain.EngagementCoefficient, error) {
	if xuid == "" {
		return nil, errors.New("EngagementScoreRepo.LoadAllCoefficients: xuid required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !r.coefficientsTableExists(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT xuid, mode_category, coef_team_share, coef_lobby_share,
		       n_matches, last_updated
		FROM engagement_coefficients
		WHERE xuid = ?
		ORDER BY mode_category
	`
	rows, err := r.pdb.ReadDB().Query(ctx, q, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadAllCoefficients: %w", err)
	}
	defer rows.Close()

	out := make([]domain.EngagementCoefficient, 0)
	for rows.Next() {
		var c domain.EngagementCoefficient
		if err := rows.Scan(
			&c.XUID, &c.ModeCategory, &c.CoefTeamShare, &c.CoefLobbyShare,
			&c.NMatches, &c.LastUpdated,
		); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// =============================================================================
// Helpers de gating (information_schema)
// =============================================================================

// engagementColumnsExist verifie que les colonnes engagement_score* existent
// dans player_match_enrichment.
func (r *EngagementScoreRepo) engagementColumnsExist(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_score'
	`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// matchIntensityColumnExists verifie la colonne shared.match_registry.match_intensity.
func (r *EngagementScoreRepo) matchIntensityColumnExists(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'match_registry'
		  AND column_name = 'match_intensity'
	`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// coefficientsTableExists verifie l'existence de engagement_coefficients.
func (r *EngagementScoreRepo) coefficientsTableExists(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	err := r.pdb.ReadDB().QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'engagement_coefficients'
	`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// =============================================================================
// Query builder
// =============================================================================

// buildEngagementHistoryQuery compose le SELECT de l'historique des residus.
// Le filtre ModeCategory s'appuie sur la colonne mode_category de
// player_match_enrichment ou est calcule cote SQL via les flags is_ranked /
// is_pve. Phase 1.5 : on suppose la presence d'une colonne mode_category
// (sinon a ajouter via la migration Phase 2 ou un SELECT plus complexe).
func buildEngagementHistoryQuery(f port.EngagementHistoryFilter) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, 4)

	sb.WriteString(`
SELECT match_id, engagement_score_brut
FROM player_match_enrichment
WHERE xuid = ?
  AND mode_category = ?
  AND engagement_score_brut IS NOT NULL
`)
	args = append(args, f.XUID, f.ModeCategory)

	if f.ExcludeMatchID != "" {
		sb.WriteString("  AND match_id <> ?\n")
		args = append(args, f.ExcludeMatchID)
	}

	sb.WriteString("ORDER BY match_id DESC\nLIMIT ?")
	args = append(args, f.Limit)

	return sb.String(), args
}
