// Package analysis — kpi_stats.go : agregation des KPIs personnels d'un joueur
// depuis ses canonical.PlayerMatchRow.
//
// Reproduit la logique compute_kpi_stats() Python (src/ui/components/kpi.py)
// pour le bandeau "Mes stats" affiche en tete de page Squad et MatchView (cf.
// PLAN_SQUAD_GO_PORTAGE § 1.1, P2).
package analysis

import (
	"math"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// ComputeKPIStats agrege les KPIs personnels depuis un slice de PlayerMatchRow.
//
// Les ratios "par minute" sont calcules sur TotalPlaySeconds (somme des
// TimePlayed de chaque match) plutot que sur la duree totale des matchs (qui
// inclurait l'attente de respawn). Cette convention Python est preservee.
//
// AvgAccuracy est exprime en pourcent (0..100). Si le champ Accuracy n'est
// renseigne sur aucun match, retourne 0 (pas de fallback).
//
// Outcomes compte uniquement les outcomes connus (Win/Loss/Tie/DNF). Les
// outcomes vides ou inconnus ne sont pas comptes.
func ComputeKPIStats(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) domain.KPIStats {
	stats := domain.KPIStats{}
	if len(rows) == 0 {
		return stats
	}

	stats.MatchesCount = len(rows)

	var totalKills, totalDeaths, totalAssists int
	var totalLifeSeconds float64
	var lifeSamples int
	var totalAccuracy float64
	var accuracySamples int
	var perfSum float64
	var perfCount int
	var totalDmgDealt, totalDmgTaken float64
	var paceRatioSum float64
	var paceRatioCount int
	// Buckets par RatingType pour le delta de rang. On accumule les deltas
	// pour chaque type rencontre puis on retient le type majoritaire en
	// sortie (cf. RankDelta.Kind — exclusivite metier au sein d'un scope coherent).
	type rankBucket struct {
		sum   float64
		count int
	}
	rankBuckets := map[canonical.RatingType]*rankBucket{}

	for _, r := range rows {
		if r.Self.TimePlayed != nil {
			stats.TotalPlaySeconds += int64(*r.Self.TimePlayed)
		}
		if r.Self.Kills != nil {
			totalKills += *r.Self.Kills
		}
		if r.Self.Deaths != nil {
			totalDeaths += *r.Self.Deaths
		}
		if r.Self.Assists != nil {
			totalAssists += *r.Self.Assists
		}
		if r.Self.AvgLifeSeconds != nil {
			totalLifeSeconds += *r.Self.AvgLifeSeconds
			lifeSamples++
		}
		if r.Self.Accuracy != nil {
			totalAccuracy += *r.Self.Accuracy
			accuracySamples++
		}
		if r.Enrichment.PerformanceScore != nil {
			perfSum += *r.Enrichment.PerformanceScore
			perfCount++
		}
		if r.Self.DamageDealt != nil && r.Self.DamageTaken != nil {
			totalDmgDealt += float64(*r.Self.DamageDealt)
			totalDmgTaken += float64(*r.Self.DamageTaken)
		}
		if r.Enrichment.EngagementPaceRatio != nil {
			paceRatioSum += *r.Enrichment.EngagementPaceRatio
			paceRatioCount++
		}
		if snap := r.Enrichment.SkillSnapshot; snap != nil && snap.Delta != nil && snap.RatingType != "" {
			b, ok := rankBuckets[snap.RatingType]
			if !ok {
				b = &rankBucket{}
				rankBuckets[snap.RatingType] = b
			}
			b.sum += *snap.Delta
			b.count++
		}
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			stats.Outcomes.Wins++
		case canonical.OutcomeLoss:
			stats.Outcomes.Losses++
		case canonical.OutcomeTie:
			stats.Outcomes.Ties++
		case canonical.OutcomeDNF:
			stats.Outcomes.DNF++
		}
	}

	n := float64(stats.MatchesCount)
	stats.KillsPerGame = float64(totalKills) / n
	stats.DeathsPerGame = float64(totalDeaths) / n
	stats.AssistsPerGame = float64(totalAssists) / n
	stats.AvgMatchSeconds = float64(stats.TotalPlaySeconds) / n

	if stats.TotalPlaySeconds > 0 {
		minutes := float64(stats.TotalPlaySeconds) / 60.0
		stats.KillsPerMinute = float64(totalKills) / minutes
		stats.DeathsPerMinute = float64(totalDeaths) / minutes
		stats.AssistsPerMinute = float64(totalAssists) / minutes
	}
	if accuracySamples > 0 {
		stats.AvgAccuracy = totalAccuracy / float64(accuracySamples)
	}
	if lifeSamples > 0 {
		stats.AvgLifeSeconds = totalLifeSeconds / float64(lifeSamples)
	}
	if perfCount > 0 {
		avg := math.Round(perfSum/float64(perfCount)*10) / 10
		stats.PerformanceScore = &avg
	} else {
		// Fallback KD : clamp(50 + 10*(KD-1), 0, 100) — miroir computeOneCard Go.
		kd := float64(totalKills)
		if totalDeaths > 0 {
			kd = float64(totalKills) / float64(totalDeaths)
		}
		fallback := math.Round(math.Min(math.Max(50+10*(kd-1), 0), 100)*10) / 10
		stats.PerformanceScore = &fallback
	}
	// Rendement / résistance : AGRÉGAT volume-pondéré sur les totaux (pas une moyenne
	// des ratios par match). OC garde les assists (frag-équivalent). Calculé sur les
	// MÊMES totaux que DmgPerKill ci-dessous → % = 225 / DmgPerKill exactement.
	avgOC := 0.0
	avgDR := 0.0
	cy := ComputeCombatYieldFloat(float64(totalKills), float64(totalAssists), totalDmgDealt, totalDmgTaken, float64(totalDeaths), effectiveHpToKill)
	if cy.OffensiveConversion > 0 {
		avgOC = math.Round(cy.OffensiveConversion*100) / 100
		stats.AvgOffensiveConversion = &avgOC
	}
	if cy.DefensiveResistance > 0 {
		avgDR = math.Round(cy.DefensiveResistance*100) / 100
		stats.AvgDefensiveResistance = &avgDR
	}
	if avgOC > 0 || avgDR > 0 {
		var avgPaceRatio *float64
		if paceRatioCount > 0 {
			v := paceRatioSum / float64(paceRatioCount)
			avgPaceRatio = &v
		}
		block := ClassifyCombatProfile(avgOC, avgDR, avgPaceRatio, stats.MatchesCount)
		// Pas de damage_taken (ex. Halo 5, totalDmgTaken==0) → DR=0 trompeur : on
		// neutralise l'axe défensif (sinon « fragile » pour TOUS les joueurs) et
		// les dégâts/mort, plutôt que d'afficher 0. Title-agnostic (data-driven).
		if totalDmgTaken <= 0 {
			block.StyleDefensive = nil
		}
		// Dégâts par frag-équivalent (frags + assists/3) : aligné sur OC. DmgPerDeath brut.
		if v := DamagePerFragEquivalent(totalDmgDealt, float64(totalKills), float64(totalAssists)); v > 0 {
			block.DmgPerKill = &v
		}
		if totalDeaths > 0 && totalDmgTaken > 0 {
			v := totalDmgTaken / float64(totalDeaths)
			block.DmgPerDeath = &v
		}
		stats.CombatProfile = &block
	}
	if len(rankBuckets) > 0 {
		// Type majoritaire : le bucket avec le plus de matchs.
		// En egalite, csr l'emporte (priorite competitive).
		var bestKind canonical.RatingType
		var best *rankBucket
		for kind, b := range rankBuckets {
			switch {
			case best == nil:
				bestKind, best = kind, b
			case b.count > best.count:
				bestKind, best = kind, b
			case b.count == best.count && kind == canonical.RatingTypeCSR:
				bestKind, best = kind, b
			}
		}
		stats.RankDelta = &domain.RankDelta{
			Kind:  string(bestKind),
			Value: best.sum,
			Count: best.count,
		}
	}
	return stats
}

// ComputeTeamAvgKPIs calcule la moyenne arithmetique des KPI individuels
// (un par joueur de l'escouade) pour servir de reference aux fleches de
// tendance ▲/▼ du SessionBriefing.
//
// Comparaison intra-session (vs moyenne d'equipe sur le scope filtre), PAS
// vs all-time du joueur. Les outcomes (wins/losses/...) sont mis a zero — la
// moyenne d'outcomes n'a pas de sens metier.
//
// Retourne nil si la map est vide ou ne contient que des entrees nil.
func ComputeTeamAvgKPIs(perXuid map[string]*domain.KPIStats) *domain.KPIStats {
	var valid []*domain.KPIStats
	for _, k := range perXuid {
		if k != nil {
			valid = append(valid, k)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	n := float64(len(valid))
	avg := &domain.KPIStats{}
	var totalPlaySec int64
	for _, k := range valid {
		avg.MatchesCount += k.MatchesCount
		totalPlaySec += k.TotalPlaySeconds
		avg.AvgMatchSeconds += k.AvgMatchSeconds
		avg.KillsPerGame += k.KillsPerGame
		avg.KillsPerMinute += k.KillsPerMinute
		avg.DeathsPerGame += k.DeathsPerGame
		avg.DeathsPerMinute += k.DeathsPerMinute
		avg.AssistsPerGame += k.AssistsPerGame
		avg.AssistsPerMinute += k.AssistsPerMinute
		avg.AvgAccuracy += k.AvgAccuracy
		avg.AvgLifeSeconds += k.AvgLifeSeconds
	}
	avg.MatchesCount = int(float64(avg.MatchesCount) / n)
	avg.TotalPlaySeconds = int64(float64(totalPlaySec) / n)
	avg.AvgMatchSeconds /= n
	avg.KillsPerGame /= n
	avg.KillsPerMinute /= n
	avg.DeathsPerGame /= n
	avg.DeathsPerMinute /= n
	avg.AssistsPerGame /= n
	avg.AssistsPerMinute /= n
	avg.AvgAccuracy /= n
	avg.AvgLifeSeconds /= n
	// Outcomes laisses a zero (sans signification en moyenne).
	return avg
}
