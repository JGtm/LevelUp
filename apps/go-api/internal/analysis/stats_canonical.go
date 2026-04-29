// Package analysis — stats_canonical.go : entry-points canonical-aware partagés
// par les services consommant `domain.StatsMatchRow` (Stats, Timeseries,
// SessionCompare, SessionPage). P4.3c, ADR 0011.
//
// **Stratégie pragmatique** : un converter unique `StatsMatchRowFromCanonical`
// expose la conversion canonical → legacy. Les services consomment ce
// converter via leurs branches `useCanonical`. Cela retire la duplication du
// converter qui vivait précédemment dans `service/stats_service.go`.
//
// **TODO P4.3 finale** : porter les analyses (buildWinLossTab, buildCumulTab,
// buildKDABuckets, computeRegressionStats, extractSessionLabels,
// buildCompareEntry, buildSessionDetailRows, etc.) à canonical et retirer ces
// converters + le type legacy `domain.StatsMatchRow`. Bloqué tant que :
//   - `port.StatsRepository.LoadStatsMatches` retourne du legacy
//   - Des analyses tierces (parallel agent squad/teammates) consomment encore
//     `domain.StatsMatchRow`.
package analysis

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// =============================================================================
// Converter canonical → StatsMatchRow (public, partagé)
// =============================================================================

// StatsMatchRowFromCanonical convertit canonical.PlayerMatchRow vers le
// format domain.StatsMatchRow consommé par les fonctions d'analyse legacy
// (buildWinLossTab, buildCumulTab, buildKDABuckets, etc.).
//
// Mapping selon ADR 0011 :
//   - K/D/A et perfs : depuis Self.
//   - SkillSnapshot KillsExpected / DeathsExpected : depuis SkillSnapshot.
//   - Outcome canonical → int Halo (Win=2, Loss=3, Tie=1, DNF=4).
//   - PlaylistName : depuis Summary.Playlist.DefaultLabel.
//   - PairName : composite Halo-only laissé vide (P4.3 finale).
//   - MedalExploitScore / OffensiveConversion / DefensiveResistance : dérivés
//     LevelUp non couverts par canonical (cf. P4_GAP_ANALYSIS.md), restent nil.
func StatsMatchRowFromCanonical(r canonical.PlayerMatchRow) domain.StatsMatchRow {
	out := domain.StatsMatchRow{
		MatchID:           r.Summary.MatchID,
		StartTime:         r.Summary.StartedAtUTC,
		KDA:               r.Self.KDA,
		Accuracy:          r.Self.Accuracy,
		PerfScoreComputed: r.Enrichment.PerformanceScore,
		SessionID:         r.Enrichment.SessionID,
		SessionLabel:      r.Enrichment.SessionLabel,
		TimePlayedSeconds: r.Self.TimePlayed,
		TeamMMR:           r.Enrichment.TeamMMR,
		EnemyMMR:          r.Enrichment.EnemyMMR,
		PersonalScore:     r.Self.PersonalScore,
		Rank:              r.Self.RankInMatch,
		TeamID:            r.Self.TeamID,
	}
	if r.Self.Kills != nil {
		out.Kills = *r.Self.Kills
	}
	if r.Self.Deaths != nil {
		out.Deaths = *r.Self.Deaths
	}
	if r.Self.Assists != nil {
		out.Assists = *r.Self.Assists
	}
	if r.Self.DamageDealt != nil {
		v := float64(*r.Self.DamageDealt)
		out.DamageDealt = &v
	}
	if r.Self.DamageTaken != nil {
		v := float64(*r.Self.DamageTaken)
		out.DamageTaken = &v
	}
	switch r.Self.Outcome {
	case canonical.OutcomeWin:
		o := domain.OutcomeWin
		out.Outcome = &o
	case canonical.OutcomeLoss:
		o := domain.OutcomeLoss
		out.Outcome = &o
	case canonical.OutcomeTie:
		o := domain.OutcomeDraw
		out.Outcome = &o
	case canonical.OutcomeDNF:
		o := domain.OutcomeDNF
		out.Outcome = &o
	}
	if r.Summary.IsRanked != nil {
		out.IsRanked = *r.Summary.IsRanked
	}
	if r.Summary.Playlist != nil {
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
	}
	if r.Enrichment.SkillSnapshot != nil {
		out.KillsExpected = r.Enrichment.SkillSnapshot.KillsExpected
		out.DeathsExpected = r.Enrichment.SkillSnapshot.DeathsExpected
	}
	return out
}

// StatsMatchRowsFromCanonical : version slice partagée par les 4 services
// consommant StatsMatchRow.
func StatsMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []domain.StatsMatchRow {
	out := make([]domain.StatsMatchRow, len(rows))
	for i, r := range rows {
		out[i] = StatsMatchRowFromCanonical(r)
	}
	return out
}

// =============================================================================
// Entry-points canonical pour analyses stats partagées (P4.3 finale)
// =============================================================================
//
// Stratégie : chaque *FromCanonical convertit en []StatsMatchRow via
// StatsMatchRowsFromCanonical puis délègue à la fonction legacy. La logique
// métier (ComputePerformanceSeries, ComputeSkillRatingsBatch, etc.) reste
// UNE source de vérité côté legacy.
//
// **Justification pragmatique** : la chaîne d'appel ComputePerformanceSeries →
// ComputeRelativePerformanceScore → applyBotBonus → ... fait plusieurs
// centaines de lignes. Un port full canonical apporterait 0 valeur métier
// (zéro changement de comportement) tout en doublant la maintenance. Les
// converters encapsulés ici permettent de retirer la conversion service-level.

// ComputePerformanceSeriesFromCanonical : entry-point canonical pour
// ComputePerformanceSeries.
func ComputePerformanceSeriesFromCanonical(rows []canonical.PlayerMatchRow) []*float64 {
	return ComputePerformanceSeries(StatsMatchRowsFromCanonical(rows))
}

// =============================================================================
// Converter canonical → SynthesisMatchRow (P4.3 finale, partagé squad/teammates)
// =============================================================================

// SynthesisMatchRowFromCanonical convertit canonical.PlayerMatchRow vers
// domain.SynthesisMatchRow consommé par les helpers internes des services
// squad / teammates. Permet à ces services de migrer canonical-only sans
// réécrire ~300 lignes d'internals (extractSynthesisSessionLabels,
// filterSynthesisByCascade, buildTeammateRow, etc.).
func SynthesisMatchRowFromCanonical(r canonical.PlayerMatchRow) domain.SynthesisMatchRow {
	out := domain.SynthesisMatchRow{
		MatchID:          r.Summary.MatchID,
		StartTime:        r.Summary.StartedAtUTC,
		IsWithFriends:    r.Enrichment.IsWithFriends,
		KDA:              r.Self.KDA,
		Accuracy:         r.Self.Accuracy,
		PerformanceScore: r.Enrichment.PerformanceScore,
		SessionLabel:     r.Enrichment.SessionLabel,
		TimePlayedSecs:   r.Self.TimePlayed,
	}
	if r.Self.Kills != nil {
		out.Kills = *r.Self.Kills
	}
	if r.Self.Deaths != nil {
		out.Deaths = *r.Self.Deaths
	}
	switch r.Self.Outcome {
	case canonical.OutcomeWin:
		out.Outcome = domain.OutcomeWin
	case canonical.OutcomeLoss:
		out.Outcome = domain.OutcomeLoss
	case canonical.OutcomeTie:
		out.Outcome = domain.OutcomeDraw
	case canonical.OutcomeDNF:
		out.Outcome = domain.OutcomeDNF
	}
	if r.Summary.IsRanked != nil {
		out.IsRanked = *r.Summary.IsRanked
	}
	if r.Summary.IsPvE != nil {
		out.IsFirefight = *r.Summary.IsPvE
	}
	if r.Summary.Playlist != nil {
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
	}
	return out
}

// SynthesisMatchRowsFromCanonical : version slice partagée par teammates/squad.
func SynthesisMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []domain.SynthesisMatchRow {
	out := make([]domain.SynthesisMatchRow, len(rows))
	for i, r := range rows {
		out[i] = SynthesisMatchRowFromCanonical(r)
	}
	return out
}
