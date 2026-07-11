// Package sync — engagement_recompute.go : recompute des coefficients
// d'engagement personnels (mediane glissante des paces).
//
// Reference plan : .ai/V7/PLAN_ENGAGEMENT_IMPLEMENTATION.md §4.4 + thought_log
// 2026-05-05 "Engagement long-term".
//
// Pipeline :
//   - Pour chaque mode_category PvP supporte
//   - Charge les paces des N derniers matchs (loadRatioSamples)
//   - Calcule la mediane via temporal.ComputeEngagementCoefficient
//   - Persiste via saveCoefficient (UPSERT)
//
// Doit etre appele APRES batchComputeEngagementScores : c'est le compute qui
// renseigne les colonnes engagement_pace_* utilisees ici.
//
// Skip silencieux si la migration "add_engagement_pace_columns" n'est pas
// appliquee (colonnes paces absentes) — laisse le coef cold-start a 1.0.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// engagementCoefModes liste les categories de mode pour lesquelles on recompute un
// coefficient. Source unique domain.EngagementCoefModes (K1n) — partagee avec le
// service admin (service.engagementCoefModesService).
var engagementCoefModes = domain.EngagementCoefModes()

// batchRecomputeCoefficients recalcule et persiste le coef lobby global (+ les
// bins de reponse via recomputeResponseBins) du joueur pour chaque mode_category
// PvP, depuis la mediane glissante des paces stockees dans player_match_enrichment.
//
// Doit etre appele APRES batchComputeEngagementScores (les paces sont
// renseignees par le compute, le recompute lit ensuite l'historique).
//
// Skip silencieux si :
//   - colonnes engagement_pace_* absentes (migration recompute coefs non appliquee)
//   - table engagement_coefficients absente (migration Phase 2 non appliquee)
//   - moins de temporal.MinMatchesForCoef samples valides pour la categorie
//
// Retourne le nombre de coefs persistes (0 a len(engagementCoefModes)).
//
// siblings ; aujourd'hui best-effort (logs warn sur erreurs internes, jamais
// remonté au caller).
//
//nolint:unparam // err maintenu en signature pour cohérence avec batchCompute*
func batchRecomputeCoefficients(
	ctx context.Context,
	playerDB *sql.DB,
	xuid string,
) (int, error) {
	if !pacesColumnsAvailable(ctx, playerDB) {
		slog.DebugContext(ctx, "engagement coefs: paces columns absent, skip recompute",
			"xuid", xuid)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_unavailable_skips_total")
		return 0, nil
	}
	if !coefficientsTableAvailable(ctx, playerDB) {
		slog.DebugContext(ctx, "engagement coefs: coefficients table absent, skip recompute",
			"xuid", xuid)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_unavailable_skips_total")
		return 0, nil
	}

	now := time.Now().UTC()
	updated := 0
	for _, mode := range engagementCoefModes {
		if recomputeMode(ctx, playerDB, xuid, mode, now) {
			updated++
		}
	}
	return updated, nil
}

// recomputeMode traite un (xuid, mode) : load samples → compute → save.
// Retourne true si un coef a ete persiste, false sinon (skip ou erreur deja
// loggee). Extraite de batchRecomputeCoefficients pour respecter la limite
// 80L par fonction.
func recomputeMode(
	ctx context.Context,
	playerDB *sql.DB,
	xuid, mode string,
	now time.Time,
) bool {
	samples, err := loadRatioSamples(ctx, playerDB, xuid, mode, temporal.DefaultRatioSampleLimit)
	if err != nil {
		slog.WarnContext(ctx, "engagement coefs: load samples failed",
			"xuid", xuid, "mode", mode, "err", err)
		return false
	}
	result, err := temporal.ComputeEngagementCoefficient(samples)
	if errors.Is(err, temporal.ErrInsufficientCoefHistory) {
		slog.DebugContext(ctx, "engagement coefs: cold-start (insufficient samples)",
			"xuid", xuid, "mode", mode, "n_samples", len(samples))
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_coef_skipped_insufficient_history")
		return false
	}
	if err != nil {
		slog.WarnContext(ctx, "engagement coefs: compute failed",
			"xuid", xuid, "mode", mode, "err", err)
		return false
	}
	if err := saveCoefficient(ctx, playerDB, xuid, mode, result, now); err != nil {
		slog.ErrorContext(ctx, "engagement coefs: save failed",
			"xuid", xuid, "mode", mode, "err", err)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_coef_save_error_total")
		return false
	}
	// Bins de reponse (modele lobby-anchored v2) : memes samples, best-effort.
	// Absence de table ou historique insuffisant = non bloquant (l'attendu
	// retombe sur le coef lobby global).
	recomputeResponseBins(ctx, playerDB, xuid, mode, samples, now)

	slog.DebugContext(ctx, "engagement coefs: updated",
		"xuid", xuid, "mode", mode,
		"coef_lobby", result.CoefLobbyShare,
		"n_matches", result.NMatches, "n_rejected", result.NRejected)
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_coef_recomputed_total")
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_coef_lobby_bucket_"+coefBucket(result.CoefLobbyShare))
	return true
}

// recomputeResponseBins calcule et persiste les bins de reponse (terciles
// d'intensite) pour (xuid, mode) depuis les memes samples que le coef. Best
// effort : skip silencieux si la table est absente (migration non appliquee) ou
// si l'historique est insuffisant pour former des terciles.
func recomputeResponseBins(
	ctx context.Context,
	playerDB *sql.DB,
	xuid, mode string,
	samples []temporal.RatioSample,
	now time.Time,
) {
	if !responseBinsTableAvailable(ctx, playerDB) {
		return
	}
	binsResult, err := temporal.ComputeEngagementResponseBins(samples)
	if errors.Is(err, temporal.ErrInsufficientBinHistory) {
		return
	}
	if err != nil {
		slog.WarnContext(ctx, "engagement bins: compute failed",
			"xuid", xuid, "mode", mode, "err", err)
		return
	}
	if err := saveResponseBins(ctx, playerDB, xuid, mode, binsResult.Bins, now); err != nil {
		slog.ErrorContext(ctx, "engagement bins: save failed",
			"xuid", xuid, "mode", mode, "err", err)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_bins_save_error_total")
		return
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), "engagement_bins_recomputed_total")
}

// responseBinsTableAvailable verifie la presence de engagement_response_bins.
func responseBinsTableAvailable(ctx context.Context, playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'engagement_response_bins'
	`).Scan(&count)
	return err == nil && count > 0
}

// saveResponseBins UPSERT (SELECT-then-UPDATE-or-INSERT, ART-safe) les bins de
// reponse. Serialise sous lease KindPlayer cote caller (comme saveCoefficient).
func saveResponseBins(
	ctx context.Context,
	playerDB *sql.DB,
	xuid, modeCategory string,
	bins []domain.EngagementIntensityBin,
	now time.Time,
) error {
	for _, b := range bins {
		var dummy int
		err := playerDB.QueryRowContext(ctx,
			`SELECT 1 FROM engagement_response_bins WHERE xuid = ? AND mode_category = ? AND intensity_bin = ?`,
			xuid, modeCategory, b.Bin).Scan(&dummy)
		switch {
		case err == nil:
			_, err = playerDB.ExecContext(ctx, `UPDATE engagement_response_bins
				SET lower_bound = ?, upper_bound = ?, coef_lobby = ?, n_matches = ?, last_updated = ?
				WHERE xuid = ? AND mode_category = ? AND intensity_bin = ?`,
				b.LowerBound, b.UpperBound, b.CoefLobby, b.NMatches, now, xuid, modeCategory, b.Bin)
		case errors.Is(err, sql.ErrNoRows):
			_, err = playerDB.ExecContext(ctx, `INSERT INTO engagement_response_bins
				(xuid, mode_category, intensity_bin, lower_bound, upper_bound, coef_lobby, n_matches, last_updated)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				xuid, modeCategory, b.Bin, b.LowerBound, b.UpperBound, b.CoefLobby, b.NMatches, now)
		}
		if err != nil {
			return fmt.Errorf("saveResponseBins bin %s: %w", b.Bin, err)
		}
	}
	return nil
}

// coefBucket retourne le bucket nomme du coef pour les metriques expvar.
// Tranches de 0.2 sur [CoefMin, CoefMax]. Permet de detecter les derives
// statistiques sans necessiter Prometheus.
func coefBucket(coef float64) string {
	switch {
	case coef < 0.5:
		return "lt_0_5"
	case coef < 0.7:
		return "0_5_to_0_7"
	case coef < 0.9:
		return "0_7_to_0_9"
	case coef < 1.1:
		return "0_9_to_1_1"
	case coef < 1.3:
		return "1_1_to_1_3"
	case coef < 1.5:
		return "1_3_to_1_5"
	case coef < 2.0:
		return "1_5_to_2_0"
	default:
		return "gte_2_0"
	}
}

// coefficientsTableAvailable verifie la presence de engagement_coefficients
// dans la player DB.
func coefficientsTableAvailable(ctx context.Context, playerDB *sql.DB) bool {
	var count int
	err := playerDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name = 'engagement_coefficients'
	`).Scan(&count)
	return err == nil && count > 0
}

// loadRatioSamples lit les paces des derniers N matchs PvP du joueur pour
// une categorie de mode. Renvoie une slice (potentiellement vide).
func loadRatioSamples(
	ctx context.Context,
	playerDB *sql.DB,
	xuid, modeCategory string,
	limit int,
) ([]temporal.RatioSample, error) {
	// Note : player_match_enrichment est dans la player DB (1 DB par joueur)
	// donc pas de colonne xuid — `xuid` est implicite. On ne filtre que par
	// mode_category et la présence des paces.
	_ = xuid
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
	rows, err := playerDB.QueryContext(ctx, q, modeCategory, limit)
	if err != nil {
		return nil, fmt.Errorf("loadRatioSamples query: %w", err)
	}
	defer rows.Close()

	out := make([]temporal.RatioSample, 0, limit)
	for rows.Next() {
		var s temporal.RatioSample
		if err := rows.Scan(&s.MatchID, &s.PaceJoueur, &s.PaceTeam, &s.PaceLobby, &s.PlayerActivity); err != nil {
			return nil, fmt.Errorf("loadRatioSamples scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// inertTeamShare : valeur ecrite dans la colonne coef_team_share, conservee NOT
// NULL mais INERTE (D5, modele lobby-anchored — plus calculee ni lue). La colonne
// reste en place (pas de DROP COLUMN sur les player DB) ; on y ecrit 1.0 neutre.
const inertTeamShare = 1.0

// saveCoefficient UPSERT le coefficient calcule dans engagement_coefficients.
func saveCoefficient(
	ctx context.Context,
	playerDB *sql.DB,
	xuid, modeCategory string,
	result *temporal.CoefficientResult,
	now time.Time,
) error {
	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT, qui réécrit via
	// l'index ART de la PK). engagement_coefficients : PK (xuid, mode_category), pas
	// d'index secondaire muté. Sérialisé sous lease KindPlayer côté caller.
	var dummy int
	err := playerDB.QueryRowContext(ctx,
		`SELECT 1 FROM engagement_coefficients WHERE xuid = ? AND mode_category = ?`,
		xuid, modeCategory).Scan(&dummy)
	switch {
	case err == nil:
		_, err = playerDB.ExecContext(ctx, `UPDATE engagement_coefficients
			SET coef_team_share = ?, coef_lobby_share = ?, n_matches = ?, last_updated = ?
			WHERE xuid = ? AND mode_category = ?`,
			inertTeamShare, result.CoefLobbyShare, result.NMatches, now, xuid, modeCategory)
	case errors.Is(err, sql.ErrNoRows):
		_, err = playerDB.ExecContext(ctx, `INSERT INTO engagement_coefficients
			(xuid, mode_category, coef_team_share, coef_lobby_share, n_matches, last_updated)
			VALUES (?, ?, ?, ?, ?, ?)`,
			xuid, modeCategory, inertTeamShare, result.CoefLobbyShare, result.NMatches, now)
	}
	return err
}
