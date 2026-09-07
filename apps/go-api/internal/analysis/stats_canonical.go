// Package analysis — stats_canonical.go : entry-points canonical-aware partagés
// par les services consommant `legacymatch.StatsMatchRow` (Stats, Timeseries,
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
// converters + le type legacy `legacymatch.StatsMatchRow`. Bloqué tant que :
//   - `port.StatsRepository.LoadStatsMatches` retourne du legacy
//   - Des analyses tierces (parallel agent squad/teammates) consomment encore
//     `legacymatch.StatsMatchRow`.
package analysis

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// =============================================================================
// Converter canonical → StatsMatchRow (public, partagé)
// =============================================================================

// StatsMatchRowFromCanonical convertit canonical.PlayerMatchRow vers le
// format legacymatch.StatsMatchRow consommé par les fonctions d'analyse legacy
// (buildWinLossTab, buildCumulTab, buildKDABuckets, etc.).
//
// Mapping selon ADR 0011 :
//   - K/D/A et perfs : depuis Self.
//   - SkillSnapshot KillsExpected / DeathsExpected : depuis SkillSnapshot.
//   - Outcome canonical → int Halo (Win=2, Loss=3, Tie=1, DNF=4).
//   - PlaylistName : depuis Summary.Playlist.DefaultLabel.
//   - PairName / PairNameFR : depuis Summary.PairMode (pair_name / pair_name_fr).
//   - MapName / MapNameFR : depuis Summary.Map.
//   - IsFirefight : depuis Summary.IsPvE.
//   - MedalExploitScore : dérivé LevelUp non couvert par canonical
//     (cf. P4_GAP_ANALYSIS.md), reste nil dans ce converter.
//   - OffensiveConversion / DefensiveResistance : dérivés LevelUp CALCULÉS
//     ici (via ComputeCombatYield, cf. plus bas) — nil seulement quand aucune
//     donnée de dégâts n'est disponible pour le match (D-11 V721-14a :
//     l'ancien commentaire affirmait à  tort qu'ils restaient toujours nil).
func StatsMatchRowFromCanonical(r canonical.PlayerMatchRow, effectiveHpToKill float64) legacymatch.StatsMatchRow {
	out := legacymatch.StatsMatchRow{
		MatchID:            r.Summary.MatchID,
		StartTime:          r.Summary.StartedAtUTC,
		KDA:                r.Self.KDA,
		Accuracy:           r.Self.Accuracy,
		PerfScoreComputed:  r.Enrichment.PerformanceScore,
		SessionID:          r.Enrichment.SessionID,
		SessionLabel:       r.Enrichment.SessionLabel,
		TimePlayedSeconds:  r.Self.TimePlayed,
		AvgLifeSeconds:     r.Self.AvgLifeSeconds,
		TeamMMR:            r.Enrichment.TeamMMR,
		EnemyMMR:           r.Enrichment.EnemyMMR,
		PersonalScore:      r.Self.PersonalScore,
		Rank:               r.Self.RankInMatch,
		TeamID:             r.Self.TeamID,
		MaxKillingSpree:    r.Self.MaxKillingSpree,
		HeadshotKills:      r.Self.HeadshotKills,
		PerfectKills:       r.Self.PerfectKills,
		MeleeKills:         r.Self.MeleeKills,
		GrenadeKills:       r.Self.GrenadeKills,
		PowerWeaponKills:   r.Self.PowerWeaponKills,
		AssassinationKills: r.Self.AssassinationKills,
		GroundPoundKills:   r.Self.GroundPoundKills,
		ShoulderBashKills:  r.Self.ShoulderBashKills,
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
	// Stats attendues natives (Self) : source de l'écart au FDA attendu. Depuis
	// MatchParticipant (populé quel que soit le skill snapshot), pas SkillSnapshot
	// (nil sans rating). ShotsHit alimente le fallback populationnel des assists attendus.
	out.KillsExpected = r.Self.KillsExpected
	out.DeathsExpected = r.Self.DeathsExpected
	out.ShotsHit = r.Self.ShotsHit
	// DominanceFlag : enrichissement narratif déjà persisté (player_match_enrichment)
	// et chargé par le projecteur canonical. 0 = aucun badge (titre sans timeline de
	// score, ou match non traité par le pipeline post-sync) → aucun marqueur front.
	out.DominanceFlag = int(r.Enrichment.DominanceFlag)
	if r.Self.DamageDealt != nil {
		v := float64(*r.Self.DamageDealt)
		out.DamageDealt = &v
	}
	if r.Self.DamageTaken != nil {
		v := float64(*r.Self.DamageTaken)
		out.DamageTaken = &v
	}
	// OC/DR : dérivés LevelUp (rendement offensif / résistance défensive) calculés
	// depuis dégâts + K/D/A via ComputeCombatYield. Sans ça avg_oc/avg_dr (KPI
	// Rendement/Résistance) et le nuage OC/DR de la page session restaient vides.
	if out.DamageDealt != nil || out.DamageTaken != nil {
		dd, dt := 0.0, 0.0
		if out.DamageDealt != nil {
			dd = *out.DamageDealt
		}
		if out.DamageTaken != nil {
			dt = *out.DamageTaken
		}
		cy := ComputeCombatYield(out.Kills, out.Assists, dd, dt, out.Deaths, effectiveHpToKill)
		if cy.OffensiveConversion > 0 {
			v := cy.OffensiveConversion
			out.OffensiveConversion = &v
		}
		if cy.DefensiveResistance > 0 {
			v := cy.DefensiveResistance
			out.DefensiveResistance = &v
		}
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
	if r.Summary.IsPvE != nil {
		out.IsFirefight = *r.Summary.IsPvE
	}
	if r.Summary.Playlist != nil {
		out.PlaylistName = r.Summary.Playlist.DefaultLabel
		if fr, ok := r.Summary.Playlist.Labels["fr"]; ok && fr != "" {
			out.PlaylistNameFR = fr
		}
	}
	if r.Summary.Map != nil {
		out.MapName = r.Summary.Map.DefaultLabel
		if fr, ok := r.Summary.Map.Labels["fr"]; ok {
			out.MapNameFR = fr
		}
	}
	if r.Summary.PairMode != nil {
		out.PairName = r.Summary.PairMode.DefaultLabel
		if fr, ok := r.Summary.PairMode.Labels["fr"]; ok {
			out.PairNameFR = fr
		}
	}
	// GameVariant : source de repli pour le mode FR. Contrairement aux paires
	// (asset_translations[pair] = EN pour toutes les langues), les game_variant SONT
	// localisés ("Assassin en équipe : Arène"). buildSessionDetailRows s'en sert quand
	// la cascade pair n'a pas produit de FR — aligné sur le converter Home.
	if r.Summary.GameVariant != nil {
		out.GameVariantName = r.Summary.GameVariant.DefaultLabel
		if fr, ok := r.Summary.GameVariant.Labels["fr"]; ok && fr != "" {
			out.GameVariantNameFR = fr
		}
	}
	if r.Enrichment.SkillSnapshot != nil {
		// KillsExpected/DeathsExpected proviennent désormais de Self (ci-dessus) —
		// populés indépendamment du skill snapshot.
		out.SkillRatingValue = r.Enrichment.SkillSnapshot.RatingValue
		out.SkillRatingType = string(r.Enrichment.SkillSnapshot.RatingType)
		out.SkillPlaylistGroup = r.Enrichment.SkillSnapshot.PlaylistGroup
		out.SkillSeasonID = r.Enrichment.SkillSnapshot.SeasonID
		out.SkillMeasurementRemaining = r.Enrichment.SkillSnapshot.MeasurementRemaining
		out.SkillRatingDelta = r.Enrichment.SkillSnapshot.Delta
		out.SkillExpectedWinProb = r.Enrichment.SkillSnapshot.ExpectedWinProb
		out.SkillTierCode = r.Enrichment.SkillSnapshot.TierCode
		out.SkillTierCodeFR = r.Enrichment.SkillSnapshot.TierCodeFR
		out.SkillSubTier = r.Enrichment.SkillSnapshot.SubTier
	}
	out.IsWithFriends = r.Enrichment.IsWithFriends
	out.EngagementScoreBrut = r.Enrichment.EngagementScoreBrut
	out.EngagementPaceRatio = r.Enrichment.EngagementPaceRatio
	return out
}

// StatsMatchRowsFromCanonical : version slice partagée par les 4 services
// consommant StatsMatchRow.
func StatsMatchRowsFromCanonical(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) []legacymatch.StatsMatchRow {
	out := make([]legacymatch.StatsMatchRow, len(rows))
	for i, r := range rows {
		out[i] = StatsMatchRowFromCanonical(r, effectiveHpToKill)
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
func ComputePerformanceSeriesFromCanonical(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) []*float64 {
	return ComputePerformanceSeries(StatsMatchRowsFromCanonical(rows, effectiveHpToKill))
}

// =============================================================================
// Converter canonical → SynthesisMatchRow (P4.3 finale, partagé squad/teammates)
// =============================================================================

// SynthesisMatchRowFromCanonical convertit canonical.PlayerMatchRow vers
// legacymatch.SynthesisMatchRow consommé par les helpers internes des services
// squad / teammates. Permet à ces services de migrer canonical-only sans
// réécrire ~300 lignes d'internals (extractSynthesisSessionLabels,
// filterSynthesisByCascade, buildTeammateRow, etc.).
func SynthesisMatchRowFromCanonical(r canonical.PlayerMatchRow) legacymatch.SynthesisMatchRow {
	out := legacymatch.SynthesisMatchRow{
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
func SynthesisMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []legacymatch.SynthesisMatchRow {
	out := make([]legacymatch.SynthesisMatchRow, len(rows))
	for i, r := range rows {
		out[i] = SynthesisMatchRowFromCanonical(r)
	}
	return out
}
