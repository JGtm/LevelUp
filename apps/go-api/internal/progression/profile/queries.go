package profile

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games"
)

// queries.go — calculs de BuildProfile() (Sections A1/A2/B/C V1) sur les lectures de port.ProfileStore.
//
// Toutes les lectures acceptent une fenêtre [since, until] cohérente avec la fenêtre LOWESS du caller ; le
// SQL (SharedReader ADR 0016, fragment timezone canonique, exclusion Campagne) vit dans le dépôt de
// platform/duckdb/profilerepo.

const queryTimeout = 10 * time.Second

// countMatchesInWindow retourne le nombre de matchs distincts du joueur dans la fenêtre.
func (s *Service) countMatchesInWindow(ctx context.Context, userID string, since, until time.Time) (int, error) {
	if s.store == nil {
		return 0, fmt.Errorf("countMatchesInWindow: PlayerDB non initialisée")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	return s.store.CountWindowMatches(ctx, userID, since, until)
}

// computeRadarAxes calcule les 6 axes radar narrative en agrégeant les stats
// du joueur sur la fenêtre.
//
// Deux sources de signal :
//   - match_participants : agrégats Halo (kills/deaths/assists/score/spree)
//     → fournit les axes combat / survival / support / score / impact.
//   - personal_score_awards : awards spécifiques (flag_captured, zone_secured,
//     destroyed_*, etc.) → fournit l'axe objective + enrichit support/impact
//     via le mapping awards.toml (V2 §2).
//
// Sans AwardMappingSet injecté, l'axe Objective reste à 0 (fallback V1).
// Avec mapping, l'axe Objective devient un signal réel et les axes
// support/impact gagnent en granularité.
func (s *Service) computeRadarAxes(ctx context.Context, userID string, since, until time.Time) (map[narrative.ParticipationAxis]float64, error) {
	out, matchCount, err := s.computeRadarAxesBase(ctx, userID, since, until)
	if err != nil {
		return nil, err
	}
	if matchCount == 0 {
		return nil, nil
	}
	// Enrichissement awards : objective + boosts ciblés sur support/impact.
	if s.awards != nil {
		s.applyAwardsRadarAxes(ctx, userID, since, until, matchCount, out)
	}
	return out, nil
}

// computeRadarAxesBase calcule les axes via match_participants (heuristique
// V1 conservée). Retourne aussi le matchCount pour normaliser les awards.
func (s *Service) computeRadarAxesBase(
	ctx context.Context, userID string, since, until time.Time,
) (map[narrative.ParticipationAxis]float64, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	base, err := s.store.RadarBase(ctx, userID, since, until)
	if err != nil {
		return nil, 0, err
	}
	out := map[narrative.ParticipationAxis]float64{}
	if base.MatchCount == 0 {
		return out, 0, nil
	}
	if base.AvgKills != nil {
		out[narrative.AxisCombat] = *base.AvgKills
	}
	if base.AvgKills != nil && base.AvgDeaths != nil {
		survival := *base.AvgKills - *base.AvgDeaths
		if survival < 0 {
			survival = 0
		}
		out[narrative.AxisSurvival] = survival
	}
	if base.AvgAssists != nil {
		out[narrative.AxisSupport] = *base.AvgAssists
	}
	if base.AvgScore != nil {
		out[narrative.AxisScore] = *base.AvgScore
	}
	if base.AvgKillingSpree != nil {
		out[narrative.AxisImpact] = *base.AvgKillingSpree * 10
	}
	return out, base.MatchCount, nil
}

// windowMatchIDs retourne les match_id du registre partagé dans la fenêtre
// (Campagne exclue). Source commune de applyAwardsRadarAxes (Phase A) et de
// computeObjectiveIndexAxis — règle ≤ 2 copies. Best-effort : erreur → nil.
func (s *Service) windowMatchIDs(ctx context.Context, since, until time.Time) []string {
	ids, err := s.store.WindowMatchIDs(ctx, since, until)
	if err != nil {
		slog.WarnContext(ctx, "profile: matchs de la fenêtre illisibles, section dégradée", "err", err)
		return nil
	}
	return ids
}

// applyAwardsRadarAxes lit les awards du joueur dans la fenêtre, somme par
// award_name, applique le mapping (axes + weight) et accumule par axe en
// moyenne par match. Mutates out in place. L'axe OBJECTIVE est ignoré ici :
// il est alimenté par l'index par opportunité (computeObjectiveIndexAxis) —
// awards.toml reste la source des AUTRES axes (support/impact/…), décision 1
// du plan PLAN_AXE_OBJECTIFS_INDEX.
//
// Split cross-DB (ADR 0016) : Phase A charge les match_ids dans la fenêtre
// (windowMatchIDs), Phase B agrège les awards du joueur sur ces matchs.
func (s *Service) applyAwardsRadarAxes(
	ctx context.Context, userID string, since, until time.Time,
	matchCount int, out map[narrative.ParticipationAxis]float64,
) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	matchIDs := s.windowMatchIDs(ctx, since, until)
	if len(matchIDs) == 0 {
		// Dégrade gracieusement comme la version legacy (shared indispo /
		// fenêtre vide → on garde la base V1 sans erreur).
		return
	}

	totals, err := s.store.AwardTotals(ctx, userID, matchIDs)
	if err != nil {
		// Les cumuls lus avant l'erreur s'appliquent quand même (lecture partielle).
		slog.WarnContext(ctx, "profile: awards du radar illisibles ou partiels", "err", err)
	}
	for _, t := range totals {
		mapping, ok := s.awards.Get(t.Name)
		if !ok {
			continue
		}
		contribution := float64(t.Total) * mapping.Weight / float64(matchCount)
		for _, axis := range mapping.Axes {
			if narrative.ParticipationAxis(axis) == narrative.AxisObjective {
				continue // remplacé par l'index par opportunité
			}
			out[narrative.ParticipationAxis(axis)] += contribution
		}
	}
}

// computeObjectiveIndexAxis calcule l'axe « Objectifs » par opportunité sur la
// fenêtre : agrégats par famille (match_objective_stats_latest) →
// narrative.ComputeObjectiveIndex. Retourne (raw, n_obj) ; n_obj == 0 (titre
// sans capability match.objective.stats, fenêtre sans match à objectif, erreur
// best-effort) → l'axe est RETIRÉ du radar par le caller (aggregateNarrative).
func (s *Service) computeObjectiveIndexAxis(
	ctx context.Context, userID string, since, until time.Time,
) (float64, int) {
	if s.store == nil || userID == "" {
		return 0, 0
	}
	// Gating capability (décision 4) — jamais de slug== (ratchet no_slug_comparison).
	if !games.ProvidesObjectiveStats(s.store.TitleSlug()) {
		return 0, 0
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	matchIDs := s.windowMatchIDs(ctx, since, until)
	if len(matchIDs) == 0 {
		return 0, 0
	}
	inputs, err := s.store.ObjectiveIndexInputs(ctx, matchIDs, []string{userID})
	if err != nil {
		slog.WarnContext(ctx, "profile: index des objectifs illisible, axe retiré", "err", err)
		return 0, 0
	}
	return narrative.ComputeObjectiveIndex(inputs[userID])
}

// computeEngagementSimple calcule un EngagementSnapshot heuristique :
//
//   - score = min(100, matches_per_day_avg × 25)
//   - tier  = low / regular / high / intense par paliers
//   - max_gap_days = plus longue interruption entre deux matchs
func (s *Service) computeEngagementSimple(ctx context.Context, userID string, since, until time.Time) (EngagementSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	var snap EngagementSnapshot

	times, err := s.store.MatchStartTimes(ctx, userID, since, until)
	if err != nil {
		return snap, err
	}
	if len(times) == 0 {
		return snap, nil
	}
	windowDays := until.Sub(since).Hours() / 24
	if windowDays <= 0 {
		windowDays = 1
	}
	snap.MatchesPerDayAvg = float64(len(times)) / windowDays
	snap.Score = snap.MatchesPerDayAvg * 25
	if snap.Score > 100 {
		snap.Score = 100
	}
	// max gap entre matchs consécutifs.
	var maxGap time.Duration
	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1])
		if gap > maxGap {
			maxGap = gap
		}
	}
	snap.MaxGapDays = int(maxGap.Hours() / 24)
	snap.Tier = engagementTier(snap.Score)
	snap.RegularityCoach = "profile.engagement." + snap.Tier
	return snap, nil
}

func engagementTier(score float64) string {
	switch {
	case score < 20:
		return "low"
	case score < 50:
		return "regular"
	case score < 80:
		return "high"
	default:
		return "intense"
	}
}

// componentRow capture l'agrégat d'une composante LUSR.
type componentRow struct {
	currentAvg float64 // moyenne sur la fenêtre (0..1)
	top20      float64 // top 20% personnel sur la fenêtre
	trend      float64 // pente LOWESS simplifiée (diff dernière - première)
}

// loadLUSRComponentsBreakdown calcule, pour chaque composante LUSR :
//
//   - current_avg : moyenne brute sur les matchs du joueur (toutes les valeurs
//     sont déjà normalisées 0..1 par le scoring engine).
//   - top20 : seuil du quintile haut (top 20% personnel = QUANTILE_CONT(0.8)).
//   - trend : amélioration récente (différence entre la moyenne des 10 derniers
//     matchs et la moyenne des 10 plus anciens dans la fenêtre — ≥ 20 matchs
//     requis pour significativité).
//
// Les matchs exclus (is_excluded) ne pèsent pas ; une erreur de lecture des exclusions est remontée. Si la
// table est vide ou la requête échoue, la map est vide silencieusement (UI dégrade gracieusement avec
// current=0/top20=0/target). Le SQL est celui de profilerepo.LUSRComponents.
func (s *Service) loadLUSRComponentsBreakdown(ctx context.Context) (map[string]componentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := make(map[string]componentRow, 8)
	aggregates, err := s.store.LUSRComponents(ctx)
	if err != nil {
		return out, err
	}
	for _, a := range aggregates {
		if a.N == 0 {
			continue
		}
		row := componentRow{}
		if a.Avg != nil {
			row.currentAvg = *a.Avg
		}
		if a.Top20 != nil {
			row.top20 = *a.Top20
		}
		if a.LastAvg != nil && a.FirstAvg != nil {
			row.trend = *a.LastAvg - *a.FirstAvg
		}
		out[a.Name] = row
	}
	return out, nil
}
