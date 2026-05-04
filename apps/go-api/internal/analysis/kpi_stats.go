// Package analysis — kpi_stats.go : agregation des KPIs personnels d'un joueur
// depuis ses canonical.PlayerMatchRow.
//
// Reproduit la logique compute_kpi_stats() Python (src/ui/components/kpi.py)
// pour le bandeau "Mes stats" affiche en tete de page Squad et MatchView (cf.
// PLAN_SQUAD_GO_PORTAGE § 1.1, P2).
package analysis

import (
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
func ComputeKPIStats(rows []canonical.PlayerMatchRow) domain.KPIStats {
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
