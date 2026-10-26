// Package service — match_history_service_briefing.go : construction du bandeau
// de briefing de l'Explorer (mode Matchs), servi au-dessus du tableau des matchs
// filtrés quand la requête pose include_briefing=true.
//
// Le socle (KPIs personnels du scope) réutilise les canonical KPIs déjà calculés
// (kpisFromScoped). Les modules baseline / dimensions / tendance sont bâtis sur
// les MatchHistoryRawRow : celles-ci portent déjà les libellés FR résolus
// (MapNameFR / PairNameFR / PlaylistName coalescé) — les canonical rows chargées
// par LoadPlayerMatches ne sont PAS enrichies FR, les utiliser afficherait des
// libellés anglais sous locale FR. Les raw rows « full history » sont en outre
// déjà post-exclusions manuelles (= baseline DEC-3). Le module « Classement »
// segmente les raw rows PAR CHAÎNE (rating_type, playlist_group) et délègue le calcul
// à analysis.ComputeRankProgressionByChain (DEC-RANK-BE).
//
// Contrat produit : PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md (DEC-1..8) +
// PLAN_EXPLORER_BRIEFING_V4_2026-07.md (Classement par chaîne).
package service

import (
	"context"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/service/mapgroups"
)

// Seuils nommés du briefing (DEC-4 : pas de magic numbers).
const (
	// MinBriefingModulesMatches est le seuil sous lequel le briefing est en
	// échantillon faible (LowSample=true) : les blocs qui COMPARENT (écarts à la
	// baseline, cartes « Par… », « Par contexte ») ne sont pas émis ; le socle, le
	// Classement, l'arme favorite, les séries et les moments forts le sont. Aligné
	// sur analysis.MinMatchesForRelative (activation du score relatif).
	MinBriefingModulesMatches = analysis.MinMatchesForRelative
	// MinDimensionGroupMatches est le nombre minimal de matchs d'un groupe (carte/
	// mode/playlist) pour avoir une note et entrer en top/flop. Une carte « Par… »
	// compare au moins deux groupes qualifiés ; en deçà elle est servie en état
	// vide (Insufficient).
	MinDimensionGroupMatches = 5
	// dimensionTopFlopCount : nombre d'entrées top ET flop par dimension (DEC-8).
	dimensionTopFlopCount = 3
	// MinRankedChainMatches est le seuil de pertinence d'une chaîne de Classement
	// (rating_type, playlist_group). Propre au module Classement : indépendant de
	// MinDimensionGroupMatches (une chaîne de rang se juge sur une progression, pas
	// sur un groupe comparé).
	MinRankedChainMatches = 10
	// RankedChainMaxCount est le nombre max de chaînes de Classement affichées
	// (parité dimensionTopFlopCount) ; ajustable en revue visuelle.
	RankedChainMaxCount = 3
)

// buildExplorerBriefing assemble le bandeau de briefing du scope filtré.
//
//   - filtered : sous-ensemble filtré (scope) en raw rows (libellés FR + start_time).
//   - allRaw   : historique complet post-exclusions (baseline DEC-3) en raw rows.
//
// Retourne nil si le scope est vide. Toujours calculés, même en échantillon faible
// (< MinBriefingModulesMatches) : Classement (si rankedCapable), Arme favorite,
// Séries, Moments forts, car ils décrivent le scope sans le comparer. Seulement hors
// échantillon faible : Baseline, Dimensions, ContextSplit et MinGroupMatches (seuil
// servi au front pour ses libellés d'état vide).
func (s *MatchHistoryService) buildExplorerBriefing(
	ctx context.Context,
	filtered, allRaw []domain.MatchHistoryRawRow,
) *domain.ExplorerBriefing {
	if len(filtered) == 0 {
		return nil
	}
	b := &domain.ExplorerBriefing{Scope: buildBriefingScope(filtered)}
	b.PeriodStart, b.PeriodEnd = scopePeriod(filtered)
	b.LowSample = len(filtered) < MinBriefingModulesMatches
	if !b.LowSample {
		b.Baseline = buildBriefingBaseline(filtered, allRaw)
		b.Dimensions = buildBriefingDimensions(filtered, allRaw, mapgroups.Load(ctx, s.mapGroups))
		// Split solo/escouade : aucun gate capability (P-7), omission si un côté est vide.
		b.ContextSplit = buildBriefingContextSplit(filtered)
		b.MinGroupMatches = MinDimensionGroupMatches
	}
	if s.rankedCapable {
		b.Ranked = buildBriefingRanked(ctx, filtered)
	}
	// Arme favorite : SEUL module du briefing qui interroge la base (agrégat des
	// frags par arme sur les matchs du scope). Best-effort strict — repo absent,
	// capability manquante ou erreur → bloc nil, le reste du briefing est servi.
	b.Weapons = buildBriefingWeapons(ctx, s.weaponKillsRepo, s.titleSlug, s.weaponKillsXUID, filtered)
	// Séries + moments forts : calculés sur TOUT le scope filtré (P-9), omission
	// si non pertinent (aucune row datée / tous les compteurs à zéro).
	b.Streaks = buildBriefingStreaks(filtered)
	b.Dominance = buildBriefingDominance(filtered)
	return b
}

// scopePeriod retourne les bornes temporelles (min/max start_time non nil) du scope.
func scopePeriod(rows []domain.MatchHistoryRawRow) (start, end *time.Time) {
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		t := *r.StartTime
		if start == nil || t.Before(*start) {
			v := t
			start = &v
		}
		if end == nil || t.After(*end) {
			v := t
			end = &v
		}
	}
	return start, end
}

// buildBriefingScope agrège le socle du sous-ensemble filtré (raw rows) — même
// source que le tableau, pour un compteur/bilan/indicateurs cohérents. Porte aussi
// les agrégats socle V4/V5 : durée totale + pics (FDA, MMR d'équipe, rang par
// système) + bornes min/max des triptyques FDA & Perf (DP-1/DEC-MINMAX).
func buildBriefingScope(scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingScope {
	if len(scope) == 0 {
		return nil
	}
	a := aggregateRawStats(scope)
	return &domain.ExplorerBriefingScope{
		Matches:              a.matches,
		Wins:                 a.wins,
		Losses:               a.losses,
		Ties:                 a.ties,
		DNF:                  a.dnf,
		WinRate:              analysis.WinRate(a.wins, a.matches),
		KDA:                  a.kda,
		MinKDA:               minScopeFloat(scope, func(r *domain.MatchHistoryRawRow) *float64 { return r.KDA }),
		AvgPerf:              a.perf,
		MinPerf:              minScopeFloat(scope, func(r *domain.MatchHistoryRawRow) *float64 { return r.PerformanceScore }),
		MaxPerf:              maxScopeFloat(scope, func(r *domain.MatchHistoryRawRow) *float64 { return r.PerformanceScore }),
		TotalDurationSeconds: sumScopeDurations(scope),
		PeakKDA:              maxScopeFloat(scope, func(r *domain.MatchHistoryRawRow) *float64 { return r.KDA }),
		PeakTeamMMR:          maxScopeFloat(scope, func(r *domain.MatchHistoryRawRow) *float64 { return r.TeamMMR }),
		PeakRanks:            scopePeakRanks(scope),
	}
}

// sumScopeDurations somme les durées (r.DurationSeconds) des matchs du scope. nil si
// aucune durée disponible (le socle « Durée totale » est alors omis côté front).
func sumScopeDurations(scope []domain.MatchHistoryRawRow) *int {
	var total int
	var found bool
	for i := range scope {
		if scope[i].DurationSeconds != nil {
			total += *scope[i].DurationSeconds
			found = true
		}
	}
	if !found {
		return nil
	}
	return &total
}

// maxScopeFloat retourne le maximum d'un champ *float64 sur le scope (nil si le champ
// est absent partout). Sert au Pic FDA (KDA natif par match), au Pic MMR (DEC-PEAK) et
// à la borne HAUTE du triptyque Perf (MaxPerf, DP-1/DEC-MINMAX).
func maxScopeFloat(scope []domain.MatchHistoryRawRow, get func(*domain.MatchHistoryRawRow) *float64) *float64 {
	var best *float64
	for i := range scope {
		v := get(&scope[i])
		if v == nil {
			continue
		}
		if best == nil || *v > *best {
			b := *v
			best = &b
		}
	}
	return best
}

// minScopeFloat retourne le minimum d'un champ *float64 sur le scope (nil si le champ
// est absent partout). Symétrique de maxScopeFloat ; sert aux bornes BASSES des
// triptyques FDA & Perf (MinKDA / MinPerf, DP-1/DEC-MINMAX).
func minScopeFloat(scope []domain.MatchHistoryRawRow, get func(*domain.MatchHistoryRawRow) *float64) *float64 {
	var best *float64
	for i := range scope {
		v := get(&scope[i])
		if v == nil {
			continue
		}
		if best == nil || *v < *best {
			b := *v
			best = &b
		}
	}
	return best
}

// scopePeakRanks retourne le meilleur palier ATTEINT par système de rating sur le
// scope (Pic rang). Pour chaque rating_type présent, argmax des rows notées par
// (analysis.CSRTierOrdinal(SkillTier EN), SubTier) ; TierLabel = SkillTierLabel de la
// row gagnante. Ordre déterministe LUSR puis CSR. 0/1/2 entrées (DEC-PEAK/DEC-PEAKRANK).
func scopePeakRanks(scope []domain.MatchHistoryRawRow) []domain.ExplorerBriefingPeakRank {
	type peak struct {
		ordinal, sub int
		label        string
	}
	best := map[string]peak{}
	for i := range scope {
		r := &scope[i]
		if r.SkillRatingType == nil || r.SkillTier == nil || r.SkillTierLabel == nil {
			continue
		}
		ord := analysis.CSRTierOrdinal(*r.SkillTier)
		if ord == 0 {
			continue // palier EN inconnu → non classable
		}
		sub := 0
		if r.SubTier != nil {
			sub = *r.SubTier
		}
		kind := strings.ToLower(strings.TrimSpace(*r.SkillRatingType))
		if cur, ok := best[kind]; !ok || ord > cur.ordinal || (ord == cur.ordinal && sub > cur.sub) {
			best[kind] = peak{ordinal: ord, sub: sub, label: *r.SkillTierLabel}
		}
	}
	if len(best) == 0 {
		return nil
	}
	out := make([]domain.ExplorerBriefingPeakRank, 0, len(best))
	for _, kind := range []string{string(canonical.RatingTypeLUSR), string(canonical.RatingTypeCSR)} {
		if p, ok := best[kind]; ok {
			out = append(out, domain.ExplorerBriefingPeakRank{RatingType: kind, TierLabel: p.label})
		}
	}
	return out
}

// buildBriefingBaseline compare le scope à l'historique complet (post-exclusions).
// Deltas signés = valeur(scope) − valeur(baseline). nil si l'un des deux est vide.
func buildBriefingBaseline(scope, all []domain.MatchHistoryRawRow) *domain.ExplorerBriefingBaseline {
	if len(scope) == 0 || len(all) == 0 {
		return nil
	}
	base := aggregateRawStats(all)
	sc := aggregateRawStats(scope)
	baseWR := analysis.WinRate(base.wins, base.matches)
	scopeWR := analysis.WinRate(sc.wins, sc.matches)
	out := &domain.ExplorerBriefingBaseline{
		Matches:      base.matches,
		WinRate:      baseWR,
		KDA:          base.kda,
		AvgPerf:      base.perf,
		DeltaWinRate: scopeWR - baseWR,
		DeltaKDA:     sc.kda - base.kda,
	}
	if sc.perf != nil && base.perf != nil {
		d := *sc.perf - *base.perf
		out.DeltaPerf = &d
	}
	return out
}

// rawAgg agrège les compteurs socle d'un ensemble de raw rows.
type rawAgg struct {
	matches, wins, losses, ties, dnf int
	kda                              float64
	perf                             *float64 // nil si aucun score de perf
}

// aggregateRawStats agrège matchs, outcomes, KDA agrégat ADR 0006 et perf moyenne.
func aggregateRawStats(rows []domain.MatchHistoryRawRow) rawAgg {
	var k, a, d int
	var perfSum float64
	var perfN int
	out := rawAgg{}
	for _, r := range rows {
		out.matches++
		k += r.Kills
		a += r.Assists
		d += r.Deaths
		switch r.Outcome {
		case domain.OutcomeWin:
			out.wins++
		case domain.OutcomeLoss:
			out.losses++
		case domain.OutcomeDraw:
			out.ties++
		case domain.OutcomeDNF:
			out.dnf++
		}
		if r.PerformanceScore != nil {
			perfSum += *r.PerformanceScore
			perfN++
		}
	}
	out.kda = analysis.AggregateKDA(k, a, d, out.matches)
	if perfN > 0 {
		avg := perfSum / float64(perfN)
		out.perf = &avg
	}
	return out
}

// coalesceStr retourne le premier pointeur non nil / non vide, "" sinon. Unique
// helper de coalescence *string→string du package service (l'ancien `coalesce`
// 2-args a été fusionné ici — résultats identiques ; F4, revue 2026-07-17).
// Garde-rail : internal/archlint/no_local_str_coalesce_test.go.
func coalesceStr(ptrs ...*string) string {
	for _, p := range ptrs {
		if p != nil && *p != "" {
			return *p
		}
	}
	return ""
}
