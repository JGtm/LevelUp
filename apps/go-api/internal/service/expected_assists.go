// Package service — expected_assists.go : résolution des assists attendus
// (modèle personnel OLS → fallback populationnel → nil), version unitaire et
// version batch per-mode. Partagé par la Match View (is_me), Timeseries et
// Sessions. L'arithmétique est factorisée ici (règle CLAUDE.md n°6 ≤2 copies).
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// assistsModelReader résout le modèle personnel OLS d'assists attendus d'un mode
// (player DB). Sous-API de port.MatchViewRepository (interface segregation).
type assistsModelReader interface {
	GetPlayerAssistsModel(ctx context.Context, gameVariantName string) (*domain.PlayerAssistsModel, error)
}

// assistsCoefReader résout le fallback populationnel (slope/intercept) d'un mode
// (metadata). Sous-API de port.MetadataRepository.
type assistsCoefReader interface {
	GetAssistsCoef(ctx context.Context, gameVariantName string) (slope, intercept float64, err error)
}

// statsMMRDelta retourne team_mmr − enemy_mmr (0 si l'un des deux manque).
func statsMMRDelta(m *legacymatch.StatsMatchRow) float64 {
	if m.TeamMMR != nil && m.EnemyMMR != nil {
		return *m.TeamMMR - *m.EnemyMMR
	}
	return 0
}

// computeExpectedAssistsBatch résout expected_assists pour le joueur suivi (is_me)
// sur un lot de matchs. Chaîne D8 : modèle personnel OLS → fallback populationnel →
// nil. Le modèle est résolu UNE SEULE FOIS par mode rencontré (game_variant_name),
// pas par match. Retourne match_id → assists attendus (absent = non résolu → traité
// comme terme 0 par analysis.ExpectedFDA). models nil → nil (dégradation propre).
func computeExpectedAssistsBatch(
	ctx context.Context,
	models assistsModelReader,
	coefs assistsCoefReader,
	matches []legacymatch.StatsMatchRow,
) map[string]*float64 {
	if models == nil || len(matches) == 0 {
		return nil
	}
	out := make(map[string]*float64, len(matches))
	personal := make(map[string]*domain.PlayerAssistsModel)
	personalDone := make(map[string]bool)
	type popCoef struct {
		slope, intercept float64
		ok               bool
	}
	pop := make(map[string]popCoef)
	popDone := make(map[string]bool)

	for i := range matches {
		m := &matches[i]
		mode := m.GameVariantName
		if !personalDone[mode] {
			pm, err := models.GetPlayerAssistsModel(ctx, mode)
			if err != nil {
				slog.WarnContext(ctx, "expected_assists_personal_model_failed", "mode", mode, "err", err)
			}
			personal[mode] = pm // pm peut être nil (mode sans modèle)
			personalDone[mode] = true
		}
		if pm := personal[mode]; pm != nil {
			v := analysis.ApplyPersonalAssistsModel(pm, float64(m.Kills), float64(m.Deaths),
				derefFloat64(m.DamageDealt), derefFloat64(m.DamageTaken), statsMMRDelta(m))
			out[m.MatchID] = &v
			continue
		}
		if coefs == nil {
			continue
		}
		if !popDone[mode] {
			slope, intercept, err := coefs.GetAssistsCoef(ctx, mode)
			if err != nil {
				slog.WarnContext(ctx, "expected_assists_pop_coef_failed", "mode", mode, "err", err)
				pop[mode] = popCoef{}
			} else {
				pop[mode] = popCoef{slope: slope, intercept: intercept, ok: true}
			}
			popDone[mode] = true
		}
		if pc := pop[mode]; pc.ok {
			ps := 0.0
			if m.PersonalScore != nil {
				ps = float64(*m.PersonalScore)
			}
			sh := 0.0
			if m.ShotsHit != nil {
				sh = float64(*m.ShotsHit)
			}
			v := analysis.ApplyPopulationalAssists(pc.slope, pc.intercept, ps, sh)
			out[m.MatchID] = &v
		}
	}
	return out
}
