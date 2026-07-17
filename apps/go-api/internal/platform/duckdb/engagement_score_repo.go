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

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/port"
)

// EngagementScoreRepo implemente port.EngagementScoreRepository.
type EngagementScoreRepo struct {
	pdb *PlayerDB

	// responseBinsExists memoize le check d'existence de engagement_response_bins
	// (scan information_schema) le temps de vie du repo (cree par requete via
	// NewEngagementScoreRepo). Sans cache, LoadResponseBins scannait
	// information_schema A CHAQUE appel → ~1 scan par match dans GetTimeseries
	// (E1 revue 2026-07). Le schema d'une DB deja ouverte ne change pas sous ce
	// handle (mono-writer + ensure additif au boot, lot A4/C) : un cache par
	// handle est sur. Non concurrent : repo utilise en sequentiel par le service.
	responseBinsExists *bool
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
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, q, args...)
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

	// coef_team_share non lu (D5, colonne inerte). Seul coef_lobby_share sert.
	const q = `
		SELECT coef_lobby_share, n_matches, last_updated
		FROM engagement_coefficients
		WHERE xuid = ? AND mode_category = ?
	`
	var coef domain.EngagementCoefficient
	coef.XUID = xuid
	coef.ModeCategory = modeCategory

	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, q, xuid, modeCategory)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadEngagementCoefficient: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(
		&coef.CoefLobbyShare,
		&coef.NMatches,
		&coef.LastUpdated,
	); err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadEngagementCoefficient: %w", err)
	}
	return &coef, nil
}

// SaveEngagementScore persiste le score, le residu brut et la confidence dans
// player_match_enrichment. Append-only #23046 : INSERT pur stage='engagement'
// (plus d'UPDATE). mode_category est repris de la vue _latest (scalar subquery)
// pour ne pas l'écraser à NULL — il est posé par le sync engagement. Ce repo HTTP
// n'a actuellement aucun caller de prod (port + mock seulement) ; la conversion
// satisfait le garde-rail append-only et reste correcte si re-câblé.
//
// Si EngagementScore est nil (cas insufficient_history), on persiste le residu et
// la confidence mais on laisse engagement_score a NULL (reset légitime, préservé
// par le merge-on-read par-groupe).
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

	// Update les paces seulement si les colonnes existent (migration Phase
	// recompute coefs appliquee). Sinon, fallback sur l'ancien UPDATE 3-col.
	hasPaces := r.pacesColumnsExist(ctx)
	var (
		q    string
		args []any
	)
	var scoreArg any
	if result.EngagementScore != nil {
		scoreArg = *result.EngagementScore
	}
	if hasPaces {
		q = `
			INSERT INTO player_match_enrichment
				(match_id, engagement_score, engagement_score_brut, engagement_score_confidence,
				 engagement_pace_player, engagement_pace_team, engagement_pace_lobby, engagement_player_activity,
				 mode_category, stage)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?,
				(SELECT mode_category FROM player_match_enrichment_latest WHERE match_id = ?), 'engagement')
		`
		args = []any{
			matchID,
			scoreArg,
			result.ResidualBrut,
			result.Confidence,
			result.MeanPaceJoueur,
			result.MeanPaceTeam,
			result.MeanPaceLobby,
			result.PlayerActivity,
			matchID,
		}
	} else {
		q = `
			INSERT INTO player_match_enrichment
				(match_id, engagement_score, engagement_score_brut, engagement_score_confidence,
				 mode_category, stage)
			VALUES (?, ?, ?, ?,
				(SELECT mode_category FROM player_match_enrichment_latest WHERE match_id = ?), 'engagement')
		`
		args = []any{
			matchID,
			scoreArg,
			result.ResidualBrut,
			result.Confidence,
			matchID,
		}
	}

	// E5 (revue 2026-07) : ExecRecovered (Reopen+retry) — écriture player-DB tolérant
	// un Reopen concurrent (« database is closed »), comme les lectures du sweep.
	if _, err := r.pdb.Player.ExecRecovered(ctx, q, args...); err != nil {
		return fmt.Errorf("EngagementScoreRepo.SaveEngagementScore: %w", err)
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

	// P2 (revue 2026-06-01) : sérialise ce chemin HTTP admin avec le post-sync
	// (qui recompute les mêmes coefs via saveCoefficient SOUS le lease KindPlayer)
	// et les CLI. Sans ce lease, l'UPSERT ON CONFLICT (xuid, mode_category) DO UPDATE
	// n'était sûr que par l'effet de bord MaxOpenConns(1) du cache de connexion.
	// Unique caller = RecomputeCoefficients (admin), jamais un contexte déjà leasé →
	// pas de réentrance. ErrDBLocked → 503 côté handler.
	w, err := r.pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("EngagementScoreRepo.SaveEngagementCoefficient: lease: %w", err)
	}
	defer w.Release()

	updated := coef.LastUpdated
	if updated.IsZero() {
		updated = time.Now().UTC()
	}

	// coef_team_share : colonne NOT NULL conservée mais INERTE (D5) — on y écrit
	// 1.0 neutre, plus jamais lue (l'attendu est ancré lobby + bins).
	const inertTeamShare = 1.0

	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT, qui réécrit via
	// l'index ART de la PK). engagement_coefficients : PK (xuid, mode_category), pas
	// d'index secondaire muté. Sous lease KindPlayer (sérialisé), basse fréquence.
	if err = r.pdb.Player.UpsertNoConflict(ctx,
		`SELECT 1 FROM engagement_coefficients WHERE xuid = ? AND mode_category = ?`,
		[]any{coef.XUID, coef.ModeCategory},
		`UPDATE engagement_coefficients SET coef_team_share = ?, coef_lobby_share = ?, n_matches = ?, last_updated = ?
		 WHERE xuid = ? AND mode_category = ?`,
		[]any{inertTeamShare, coef.CoefLobbyShare, coef.NMatches, updated, coef.XUID, coef.ModeCategory},
		`INSERT INTO engagement_coefficients (xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		[]any{coef.XUID, coef.ModeCategory, inertTeamShare, coef.CoefLobbyShare, coef.NMatches, updated},
	); err != nil {
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

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("EngagementScoreRepo.LoadMatchIntensity: %w", err)
	}
	defer release()

	var intensity sql.NullFloat64
	err = db.QueryRowContext(ctx,
		`SELECT match_intensity FROM match_registry WHERE match_id = ?`,
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
		FROM player_match_enrichment_latest
		WHERE match_id = ?
	`
	var has bool
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, q, matchID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("EngagementScoreRepo.HasEngagementScore: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(&has); err != nil {
		return false, fmt.Errorf("EngagementScoreRepo.HasEngagementScore: %w", err)
	}
	return has, nil
}

// LoadRatioSamples charge les paces moyennes des derniers matchs PvP du
// joueur sur une categorie de mode, sous forme de RatioSample pour le calcul
// du coefficient (cf. temporal.ComputeEngagementCoefficient).
//
// L'ordre est match_id DESC (les plus recents d'abord). Le filtrage outliers
// est fait cote algo, ici on retourne tous les samples avec paces non-null.
func (r *EngagementScoreRepo) LoadRatioSamples(
	ctx context.Context,
	xuid, modeCategory string,
	limit int,
) ([]temporal.RatioSample, error) {
	if xuid == "" || modeCategory == "" {
		return nil, errors.New("EngagementScoreRepo.LoadRatioSamples: xuid and modeCategory required")
	}
	if limit <= 0 {
		limit = temporal.DefaultRatioSampleLimit
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if !r.pacesColumnsExist(ctx) {
		return nil, port.ErrEngagementUnavailable
	}

	const q = `
		SELECT
			match_id,
			COALESCE(engagement_pace_player, 0),
			COALESCE(engagement_pace_team, 0),
			COALESCE(engagement_pace_lobby, 0),
			COALESCE(engagement_player_activity, 0)
		FROM player_match_enrichment_latest
		WHERE mode_category = ?
		  AND engagement_pace_team IS NOT NULL
		ORDER BY match_id DESC
		LIMIT ?
	`
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, q, modeCategory, limit)
	if err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadRatioSamples: query: %w", err)
	}
	defer rows.Close()

	out := make([]temporal.RatioSample, 0, limit)
	for rows.Next() {
		var s temporal.RatioSample
		if err := rows.Scan(
			&s.MatchID,
			&s.PaceJoueur,
			&s.PaceTeam,
			&s.PaceLobby,
			&s.PlayerActivity,
		); err != nil {
			return nil, fmt.Errorf("EngagementScoreRepo.LoadRatioSamples: scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("EngagementScoreRepo.LoadRatioSamples: rows: %w", err)
	}
	return out, nil
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
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_score'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// pacesColumnsExist verifie que les colonnes engagement_pace_* existent dans
// player_match_enrichment (migration Phase recompute coefs).
func (r *EngagementScoreRepo) pacesColumnsExist(ctx context.Context) bool {
	if r.pdb == nil || r.pdb.ReadDB() == nil {
		return false
	}
	var count int
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'player_match_enrichment'
		  AND column_name = 'engagement_pace_team'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
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
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'match_registry'
		  AND column_name = 'match_intensity'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
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
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'engagement_coefficients'
	`)
	if err != nil {
		return false
	}
	defer rows.Close()
	if err := rows.Scan(&count); err != nil {
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
FROM player_match_enrichment_latest
WHERE mode_category = ?
  AND engagement_score_brut IS NOT NULL
`)
	args = append(args, f.ModeCategory)

	if f.ExcludeMatchID != "" {
		sb.WriteString("  AND match_id <> ?\n")
		args = append(args, f.ExcludeMatchID)
	}

	sb.WriteString("ORDER BY match_id DESC\nLIMIT ?")
	args = append(args, f.Limit)

	return sb.String(), args
}
