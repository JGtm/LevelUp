// Package service — engagement_admin_service.go : RecomputeCoefficients
// (admin endpoint POST /engagement/recompute_coefficients).
//
// Decoupe de engagement_player_service.go pour respecter la limite 500L
// (cf. arch-rules § Modularité).
package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// RecomputeReport est le resultat d'un appel admin a RecomputeCoefficients.
type RecomputeReport struct {
	ModesUpdated    []string `json:"modes_updated"`
	NCoefsPersisted int      `json:"n_coefs_persisted"`
	ModesSkipped    []string `json:"modes_skipped,omitempty"` // ex. insufficient_history
}

// engagementCoefModesService liste les categories de mode prises en charge par
// l'admin RecomputeCoefficients. Source unique domain.EngagementCoefModes (K1n) —
// meme liste que sync.engagementCoefModes.
var engagementCoefModesService = domain.EngagementCoefModes()

// RecomputeCoefficients force le recalcul des coefficients perso du joueur
// pour toutes les categories de mode supportees, depuis les paces persistees
// dans player_match_enrichment.
//
// Retourne un rapport detaillant les modes mis a jour. Erreurs sentinelles :
//   - port.ErrEngagementUnavailable : migration non appliquee → 503 cote handler
//
// Les modes "insufficient_history" ne sont pas une erreur : on les liste dans
// ModesSkipped et le caller decide de l'UX (ex. afficher "<10 matchs sur ce mode").
func (s *PlayerEngagementService) RecomputeCoefficients(ctx context.Context) (*RecomputeReport, error) {
	if s.xuid == "" {
		return nil, errors.New("PlayerEngagementService.RecomputeCoefficients: xuid required")
	}
	report := &RecomputeReport{
		ModesUpdated: make([]string, 0, len(engagementCoefModesService)),
		ModesSkipped: make([]string, 0),
	}
	for _, mode := range engagementCoefModesService {
		if err := s.recomputeAdminMode(ctx, mode, report); err != nil {
			// Erreur fatale (migration absente) — abort tot.
			return nil, err
		}
	}
	return report, nil
}

// recomputeAdminMode traite un (xuid, mode) : load samples → compute → save.
// Retourne port.ErrEngagementUnavailable si la migration est absente (le
// caller propage en 503). Les autres echecs sont enregistres dans le report
// (ModesSkipped) sans interrompre le traitement des modes restants.
func (s *PlayerEngagementService) recomputeAdminMode(
	ctx context.Context,
	mode string,
	report *RecomputeReport,
) error {
	samples, err := s.repo.LoadRatioSamples(ctx, s.xuid, mode, temporal.DefaultRatioSampleLimit)
	if err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return port.ErrEngagementUnavailable
		}
		slog.WarnContext(ctx, "RecomputeCoefficients: load samples failed",
			"xuid", s.xuid, "mode", mode, "err", err)
		report.ModesSkipped = append(report.ModesSkipped, mode+":load_failed")
		return nil
	}
	result, err := temporal.ComputeEngagementCoefficient(samples)
	if errors.Is(err, temporal.ErrInsufficientCoefHistory) {
		report.ModesSkipped = append(report.ModesSkipped, mode+":insufficient_history")
		return nil
	}
	if err != nil {
		slog.WarnContext(ctx, "RecomputeCoefficients: compute failed",
			"xuid", s.xuid, "mode", mode, "err", err)
		report.ModesSkipped = append(report.ModesSkipped, mode+":compute_failed")
		return nil
	}
	coef := domain.EngagementCoefficient{
		XUID:           s.xuid,
		Gamertag:       s.gamertag,
		ModeCategory:   mode,
		CoefTeamShare:  result.CoefTeamShare,
		CoefLobbyShare: result.CoefLobbyShare,
		NMatches:       result.NMatches,
		LastUpdated:    time.Now().UTC(),
	}
	if err := s.repo.SaveEngagementCoefficient(ctx, coef); err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return port.ErrEngagementUnavailable
		}
		slog.ErrorContext(ctx, "RecomputeCoefficients: save failed",
			"xuid", s.xuid, "mode", mode, "err", err)
		report.ModesSkipped = append(report.ModesSkipped, mode+":save_failed")
		return nil
	}
	// Bins de reponse (lobby-anchored v2) : memes samples, best-effort. Absence
	// de table ou historique insuffisant = non bloquant (l'attendu retombe sur
	// le coef lobby global).
	s.saveResponseBinsAdmin(ctx, mode, samples)
	report.ModesUpdated = append(report.ModesUpdated, mode)
	report.NCoefsPersisted++
	return nil
}

// saveResponseBinsAdmin calcule et persiste les bins de reponse pour (xuid, mode)
// depuis les memes samples que le coef. Best effort : une table absente ou un
// historique insuffisant ne fait pas echouer le recompute admin.
func (s *PlayerEngagementService) saveResponseBinsAdmin(
	ctx context.Context,
	mode string,
	samples []temporal.RatioSample,
) {
	binsResult, err := temporal.ComputeEngagementResponseBins(samples)
	if errors.Is(err, temporal.ErrInsufficientBinHistory) {
		return
	}
	if err != nil {
		slog.WarnContext(ctx, "RecomputeCoefficients: bins compute failed",
			"xuid", s.xuid, "mode", mode, "err", err)
		return
	}
	bins := domain.EngagementResponseBins{XUID: s.xuid, ModeCategory: mode, Bins: binsResult.Bins}
	if err := s.repo.SaveResponseBins(ctx, bins); err != nil {
		if errors.Is(err, port.ErrEngagementUnavailable) {
			return // table absente : non bloquant
		}
		slog.WarnContext(ctx, "RecomputeCoefficients: bins save failed",
			"xuid", s.xuid, "mode", mode, "err", err)
	}
}
