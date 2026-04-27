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
	return stats
}
