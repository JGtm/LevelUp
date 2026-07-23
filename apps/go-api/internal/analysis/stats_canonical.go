// Package analysis â€” stats_canonical.go : entry-points canonical-aware partagÃ©s
// par les services consommant `legacymatch.StatsMatchRow` (Stats, Timeseries,
// SessionCompare, SessionPage). P4.3c, ADR 0011.
//
// **StratÃ©gie pragmatique** : un converter unique `StatsMatchRowFromCanonical`
// expose la conversion canonical â†’ legacy. Les services consomment ce
// converter via leurs branches `useCanonical`. Cela retire la duplication du
// converter qui vivait prÃ©cÃ©demment dans `service/stats_service.go`.
//
// **TODO P4.3 finale** : porter les analyses (buildWinLossTab, buildCumulTab,
// buildKDABuckets, computeRegressionStats, extractSessionLabels,
// buildCompareEntry, buildSessionDetailRows, etc.) Ã  canonical et retirer ces
// converters + le type legacy `legacymatch.StatsMatchRow`. BloquÃ© tant que :
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
// Converter canonical â†’ StatsMatchRow (public, partagÃ©)
// =============================================================================

// StatsMatchRowFromCanonical convertit canonical.PlayerMatchRow vers le
// format legacymatch.StatsMatchRow consommÃ© par les fonctions d'analyse legacy
// (buildWinLossTab, buildCumulTab, buildKDABuckets, etc.).
//
// Mapping selon ADR 0011 :
//   - K/D/A et perfs : depuis Self.
//   - SkillSnapshot KillsExpected / DeathsExpected : depuis SkillSnapshot.
//   - Outcome canonical â†’ int Halo (Win=2, Loss=3, Tie=1, DNF=4).
//   - PlaylistName : depuis Summary.Playlist.DefaultLabel.
//   - PairName / PairNameFR : depuis Summary.PairMode (pair_name / pair_name_fr).
//   - MapName / MapNameFR : depuis Summary.Map.
//   - IsFirefight : depuis Summary.IsPvE.
//   - MedalExploitScore / OffensiveConversion / DefensiveResistance : dÃ©rivÃ©s
//     LevelUp non couverts par canonical (cf. P4_GAP_ANALYSIS.md), restent nil.
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

// StatsMatchRowsFromCanonical : version slice partagÃ©e par les 4 services
// consommant StatsMatchRow.
func StatsMatchRowsFromCanonical(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) []legacymatch.StatsMatchRow {
	out := make([]legacymatch.StatsMatchRow, len(rows))
	for i, r := range rows {
		out[i] = StatsMatchRowFromCanonical(r, effectiveHpToKill)
	}
	return out
}

// =============================================================================
// Entry-points canonical pour analyses stats partagÃ©es (P4.3 finale)
// =============================================================================
//
// StratÃ©gie : chaque *FromCanonical convertit en []StatsMatchRow via
// StatsMatchRowsFromCanonical puis dÃ©lÃ¨gue Ã  la fonction legacy. La logique
// mÃ©tier (ComputePerformanceSeries, ComputeSkillRatingsBatch, etc.) reste
// UNE source de vÃ©ritÃ© cÃ´tÃ© legacy.
//
// **Justification pragmatique** : la chaÃ®ne d'appel ComputePerformanceSeries â†’
// ComputeRelativePerformanceScore â†’ applyBotBonus â†’ ... fait plusieurs
// centaines de lignes. Un port full canonical apporterait 0 valeur mÃ©tier
// (zÃ©ro changement de comportement) tout en doublant la maintenance. Les
// converters encapsulÃ©s ici permettent de retirer la conversion service-level.

// ComputePerformanceSeriesFromCanonical : entry-point canonical pour
// ComputePerformanceSeries.
func ComputePerformanceSeriesFromCanonical(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) []*float64 {
	return ComputePerformanceSeries(StatsMatchRowsFromCanonical(rows, effectiveHpToKill))
}

// =============================================================================
// Converter canonical â†’ SynthesisMatchRow (P4.3 finale, partagÃ© squad/teammates)
// =============================================================================

// SynthesisMatchRowFromCanonical convertit canonical.PlayerMatchRow vers
// legacymatch.SynthesisMatchRow consommÃ© par les helpers internes des services
// squad / teammates. Permet Ã  ces services de migrer canonical-only sans
// rÃ©Ã©crire ~300 lignes d'internals (extractSynthesisSessionLabels,
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

// SynthesisMatchRowsFromCanonical : version slice partagÃ©e par teammates/squad.
func SynthesisMatchRowsFromCanonical(rows []canonical.PlayerMatchRow) []legacymatch.SynthesisMatchRow {
	out := make([]legacymatch.SynthesisMatchRow, len(rows))
	for i, r := range rows {
		out[i] = SynthesisMatchRowFromCanonical(r)
	}
	return out
}
