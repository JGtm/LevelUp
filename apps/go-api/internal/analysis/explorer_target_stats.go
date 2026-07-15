// Package analysis — explorer_target_stats.go : agrégation des stats du
// joueur cible sur l'échantillon des matchs joués en commun avec le user
// connecté (cf. Explorer mode Joueur, encart "Profil joueur cible").
//
// Toutes les opérations sont stateless et déterministes (pas d'accès DB,
// pas d'IO). Réutilise ComputeCombatYield (combat_yield.go) pour les ratios
// OffensiveConversion / DefensiveResistance.
package analysis

import "levelup/go-api/internal/domain"

// BuildSampleStats agrège les totaux bruts retournés par le repo en un bloc
// ExplorerTargetSampleStats prêt à être sérialisé.
//
// Tous les ratios sont retournés en pointer : nil signifie "indisponible"
// quand le dénominateur est nul (ex : deaths=0 → KDR indéfini, shots_fired=0
// → accuracy indéfinie). Le frontend rend "—" dans ce cas.
//
// Retourne nil quand sampleSize ≤ 0 ou agg == nil — l'encart masque alors la
// section "Sur N matchs joués ensemble".
//
// Le KDA per-match est NET ((k + a/3) − d, peut être négatif) pour TOUS les titres
// (Infinite : valeur CoreStats "KDA" d'API ; Halo 5 : FDA NET native). L'agrégat est
// donc la MOYENNE NETTE ((Σk + Σa/3) − Σd)/N, identique pour tous les titres —
// JAMAIS le quotient (k+a/3)/d (qui serait un BUG). KDR (k/d) reste toujours calculé.
func BuildSampleStats(
	agg *domain.ParticipantStatsAggregate,
	medals *domain.MedalCountsAggregate,
	sampleSize int,
	effectiveHpToKill float64,
) *domain.ExplorerTargetSampleStats {
	if agg == nil || sampleSize <= 0 {
		return nil
	}

	stats := &domain.ExplorerTargetSampleStats{
		SampleSize:       sampleSize,
		Kills:            agg.Kills,
		Deaths:           agg.Deaths,
		Assists:          agg.Assists,
		Wins:             agg.Wins,
		Losses:           agg.Losses,
		Draws:            agg.Draws,
		ShotsFired:       agg.ShotsFired,
		ShotsHit:         agg.ShotsHit,
		DamageDealt:      int(agg.DamageDealt),
		DamageTaken:      int(agg.DamageTaken),
		HeadshotKills:    agg.HeadshotKills,
		MeleeKills:       agg.MeleeKills,
		PowerWeaponKills: agg.PowerWeaponKills,
		GrenadeKills:     agg.GrenadeKills,
	}

	if medals != nil {
		stats.TotalMedals = medals.Total
		stats.UniqueMedals = medals.Unique
		stats.PerfectKills = medals.PerfectKills
	}

	// KDR = kills / deaths (non ambigu, toujours calculé).
	if agg.Deaths > 0 {
		kdr := float64(agg.Kills) / float64(agg.Deaths)
		stats.KDR = &kdr
	}
	// KDA agrégé = moyenne NETTE pour TOUS les titres (le KDA par match est figé à
	// l'ingestion et NET ; on agrège, on ne refabrique pas la forme par match).
	// Helper canonique AggregateKDA (ADR 0006) : ((k + a/3) − d)/N (sampleSize > 0
	// garanti ci-dessus), peut être négatif — JAMAIS le quotient (k+a/3)/d.
	kda := AggregateKDA(agg.Kills, agg.Assists, agg.Deaths, sampleSize)
	stats.KDA = &kda

	// WinRate = wins / (wins + losses + draws). On exclut les DNF du
	// dénominateur — convention alignée sur le reste du produit (cf. compare).
	played := agg.Wins + agg.Losses + agg.Draws
	if played > 0 {
		wr := float64(agg.Wins) / float64(played)
		stats.WinRate = &wr
	}

	// Accuracy = hits / fired.
	if agg.ShotsFired > 0 {
		acc := float64(agg.ShotsHit) / float64(agg.ShotsFired)
		stats.Accuracy = &acc
	}

	// HeadshotRate = headshots / kills.
	if agg.Kills > 0 {
		hr := float64(agg.HeadshotKills) / float64(agg.Kills)
		stats.HeadshotRate = &hr
	}

	// Rendement combat : réutilise la formule canonique.
	yield := ComputeCombatYield(agg.Kills, agg.Assists, agg.DamageDealt, agg.DamageTaken, agg.Deaths, effectiveHpToKill)
	if yield.OffensiveConversion > 0 {
		oc := yield.OffensiveConversion
		stats.OffensiveConversion = &oc
	}
	if yield.DefensiveResistance > 0 {
		dr := yield.DefensiveResistance
		stats.DefensiveResistance = &dr
	}

	// Cadence par minute (même KPI dérivé que la page Coéquipiers, teammates.14).
	setPerMinuteCadence(stats, agg)

	// Score Halo moyen par match (AVG(personal_score)).
	if sampleSize > 0 {
		avg := float64(agg.PersonalScore) / float64(sampleSize)
		stats.AvgPersonalScore = &avg
	}

	return stats
}

// setPerMinuteCadence dérive frags/morts/assists par minute depuis le temps
// joué cumulé (time_played_seconds). Laisse les pointeurs nil si la durée
// cumulée est nulle (dégradation gracieuse, comme la page Coéquipiers).
func setPerMinuteCadence(stats *domain.ExplorerTargetSampleStats, agg *domain.ParticipantStatsAggregate) {
	if agg.TimePlayedSeconds <= 0 {
		return
	}
	minutes := float64(agg.TimePlayedSeconds) / 60.0
	kpm := float64(agg.Kills) / minutes
	dpm := float64(agg.Deaths) / minutes
	apm := float64(agg.Assists) / minutes
	stats.KillsPerMin = &kpm
	stats.DeathsPerMin = &dpm
	stats.AssistsPerMin = &apm
}
