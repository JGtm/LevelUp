// Package service - synthesis_service_builders.go : builders highlights +
// detailed stats + top weapons + combat profile + fun stats (canonical
// path). Decoupe de synthesis_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// logFragDistribution émet les compteurs d'agrégation d'une FragDistribution (Debug)
// et SIGNALE (Warn, jamais avalé) un sur-comptage (Σ classes attribuées > total) :
// anomalie de données qui rend le résidu « Non attribué » impossible à calculer
// (l'invariant a n'est alors pas tenu). Helper PARTAGÉ par toutes les surfaces du
// package service qui construisent une FragDistribution (Synthesis, Timeseries,
// Sessions, Match view) — règle ≤2 copies. `surface` préfixe les messages
// ("synthesis"/"timeseries"/"session page"/"match view"). La page Escouade (package
// service/teammates, qui ne peut pas importer son parent service) émet son propre log
// structuré local ("teammates_frag_distribution_built") — cf. squadFragClassesByPlayer.
func logFragDistribution(ctx context.Context, surface, title, player string, fd domain.FragDistribution) {
	sumClasses, sumRoles, unattributed := 0, 0, 0
	for _, c := range fd.Classes {
		sumClasses += c.Kills
		sumRoles += len(c.Roles)
		if c.Class == domain.FragClassUnattributed {
			unattributed = c.Kills
		}
	}
	slog.DebugContext(ctx, surface+": frag distribution built",
		"title", title, "player", player,
		"total_kills", fd.TotalKills, "class_count", len(fd.Classes),
		"role_count", sumRoles, "unattributed", unattributed)
	if sumClasses > fd.TotalKills {
		slog.WarnContext(ctx, surface+": frag distribution over-count (résidu négatif clampé)",
			"title", title, "player", player,
			"sum_classes", sumClasses, "total_kills", fd.TotalKills)
	}
}

func buildHighlightsPreviewCanonical(rows []canonical.PlayerMatchRow) domain.SynthesisHighlightsPreview {
	if len(rows) == 0 {
		return domain.SynthesisHighlightsPreview{
			TopByKills:    []domain.SynthesisMatchHighlight{},
			TopByKDA:      []domain.SynthesisMatchHighlight{},
			WorstByDeaths: []domain.SynthesisMatchHighlight{},
		}
	}
	toHighlight := func(r canonical.PlayerMatchRow) domain.SynthesisMatchHighlight {
		k, d := 0, 0
		if r.Self.Kills != nil {
			k = *r.Self.Kills
		}
		if r.Self.Deaths != nil {
			d = *r.Self.Deaths
		}
		// Outcome canonical â†' int Halo pour le DTO inchangÃ©.
		var outcome int
		switch r.Self.Outcome {
		case canonical.OutcomeWin:
			outcome = domain.OutcomeWin
		case canonical.OutcomeLoss:
			outcome = domain.OutcomeLoss
		case canonical.OutcomeTie:
			outcome = domain.OutcomeDraw
		case canonical.OutcomeDNF:
			outcome = domain.OutcomeDNF
		}
		return domain.SynthesisMatchHighlight{
			MatchID:   r.Summary.MatchID,
			Kills:     k,
			Deaths:    d,
			KDA:       r.Self.KDA,
			Outcome:   outcome,
			PerfScore: r.Enrichment.PerformanceScore,
		}
	}

	topByKills := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		ak, bk := 0, 0
		if a.Self.Kills != nil {
			ak = *a.Self.Kills
		}
		if b.Self.Kills != nil {
			bk = *b.Self.Kills
		}
		return ak > bk
	})
	topByKDA := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		av := 0.0
		if a.Self.KDA != nil {
			av = *a.Self.KDA
		}
		bv := 0.0
		if b.Self.KDA != nil {
			bv = *b.Self.KDA
		}
		return av > bv
	})
	worstByDeaths := topNByFuncCanonical(rows, highlightTopN, func(a, b canonical.PlayerMatchRow) bool {
		ad, bd := 0, 0
		if a.Self.Deaths != nil {
			ad = *a.Self.Deaths
		}
		if b.Self.Deaths != nil {
			bd = *b.Self.Deaths
		}
		return ad > bd
	})

	toSlice := func(src []canonical.PlayerMatchRow) []domain.SynthesisMatchHighlight {
		out := make([]domain.SynthesisMatchHighlight, len(src))
		for i, r := range src {
			out[i] = toHighlight(r)
		}
		return out
	}
	return domain.SynthesisHighlightsPreview{
		TopByKills:    toSlice(topByKills),
		TopByKDA:      toSlice(topByKDA),
		WorstByDeaths: toSlice(worstByDeaths),
	}
}

// topNByFuncCanonical est la variante canonical de topNByFunc.
func topNByFuncCanonical(rows []canonical.PlayerMatchRow, n int, less func(a, b canonical.PlayerMatchRow) bool) []canonical.PlayerMatchRow {
	cp := make([]canonical.PlayerMatchRow, len(rows))
	copy(cp, rows)
	for i := 0; i < n && i < len(cp); i++ {
		minIdx := i
		for j := i + 1; j < len(cp); j++ {
			if less(cp[j], cp[minIdx]) {
				minIdx = j
			}
		}
		cp[i], cp[minIdx] = cp[minIdx], cp[i]
	}
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

// buildSynthesisDetailedStatsFromCanonical agrège les métriques détaillées depuis les rows canoniques.
// Combat : headshot/grenade/melee/power kills, max killing spree.
// Tir : shots fired/hit.
// Dégâts : damage dealt/taken.
//
// provideSpree : false quand le titre ne porte pas le max killing spree (Halo 5) →
// MaxKillingSpree reste nil (le front masque la stat) au lieu d'exposer un 0.
func buildSynthesisDetailedStatsFromCanonical(rows []canonical.PlayerMatchRow, provideSpree bool) domain.SynthesisDetailedStats {
	stats := domain.SynthesisDetailedStats{}
	var maxSpree int
	for _, r := range rows {
		if r.Self.HeadshotKills != nil {
			stats.TotalHeadshotKills += *r.Self.HeadshotKills
		}
		if r.Self.PerfectKills != nil {
			stats.TotalPerfectKills += *r.Self.PerfectKills
		}
		if r.Self.GrenadeKills != nil {
			stats.TotalGrenadeKills += *r.Self.GrenadeKills
		}
		if r.Self.MeleeKills != nil {
			stats.TotalMeleeKills += *r.Self.MeleeKills
		}
		if r.Self.PowerWeaponKills != nil {
			stats.TotalPowerWeaponKills += *r.Self.PowerWeaponKills
		}
		// Mécaniques natives Halo 5 (0 hors h5). Cumul sur le scope → cards cumulées.
		if r.Self.AssassinationKills != nil {
			stats.TotalAssassinations += *r.Self.AssassinationKills
		}
		if r.Self.GroundPoundKills != nil {
			stats.TotalGroundPoundKills += *r.Self.GroundPoundKills
		}
		if r.Self.ShoulderBashKills != nil {
			stats.TotalShoulderBashKills += *r.Self.ShoulderBashKills
		}
		if r.Self.MaxKillingSpree != nil && *r.Self.MaxKillingSpree > maxSpree {
			maxSpree = *r.Self.MaxKillingSpree
		}
		if r.Self.ShotsFired != nil {
			stats.TotalShotsFired += *r.Self.ShotsFired
		}
		if r.Self.ShotsHit != nil {
			stats.TotalShotsHit += *r.Self.ShotsHit
		}
		if r.Self.DamageDealt != nil {
			stats.TotalDamageDealt += float64(*r.Self.DamageDealt)
		}
		if r.Self.DamageTaken != nil {
			stats.TotalDamageTaken += float64(*r.Self.DamageTaken)
		}
		if r.Self.TimePlayed != nil {
			stats.TotalTimePlayedSeconds += *r.Self.TimePlayed
		}
	}
	if provideSpree {
		stats.MaxKillingSpree = &maxSpree
	}
	return stats
}

// synthesisWeaponChartTopN plafonne le nombre de barres des DEUX graphes d'armes
// « Frags par arme » (buildTopWeaponKills) ET « Précision par arme »
// (buildWeaponAccuracy) — même limitation par titre (demande utilisateur B1 :
// « limiter de la même manière », Frags par arme = référence). C'est un cap de
// COMPTE (top N après tri par la métrique du graphe), pas un seuil de volume :
// l'exclusion volume-based reste refusée (cf. buildWeaponAccuracy).
const synthesisWeaponChartTopN = 20

// buildSynthesisFunStatsFromAwards agrege les fun stats depuis personal_score_awards.
// buildTopWeaponKills filtre les rows sans label (weapon ID non résolu), trie
// par kills desc et retourne les top N entrées.
func buildTopWeaponKills(rows []port.WeaponKillRow, n int) []domain.SynthesisWeaponKillEntry {
	resolved := make([]port.WeaponKillRow, 0, len(rows))
	for _, r := range rows {
		if r.Label != "" && !r.IsGrenadeMelee {
			resolved = append(resolved, r)
		}
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].Kills > resolved[j].Kills })
	if len(resolved) > n {
		resolved = resolved[:n]
	}
	out := make([]domain.SynthesisWeaponKillEntry, len(resolved))
	for i, r := range resolved {
		// Class/Role portés depuis le registre (résolus dans la même passe
		// ResolveRoles) pour recolorer le breakdown par arme par classe (P1.5).
		out[i] = domain.SynthesisWeaponKillEntry{Label: r.Label, Kills: r.Kills, Class: r.Class, Role: r.Role}
	}
	return out
}

// buildWeaponAccuracy construit le classement précision par arme : armes
// effectivement tirées (Label résolu ET ShotsFired > 0 — AUCUN seuil de volume,
// conformément à la demande utilisateur). Accuracy = landed / fired en unité
// 0..1. Tri par précision décroissante (tie-break Label alpha pour un ordre
// stable), PUIS cap top N (n = synthesisWeaponChartTopN) — même limitation que
// « Frags par arme » (buildTopWeaponKills), demande utilisateur B1. Le cap est un
// plafond de COMPTE appliqué après tri (identique au reference), pas un filtre de
// volume. nil si aucune arme valide (→ champ omis de la réponse).
func buildWeaponAccuracy(rows []port.WeaponAccuracyRow, n int) []domain.SynthesisWeaponAccuracyEntry {
	out := make([]domain.SynthesisWeaponAccuracyEntry, 0, len(rows))
	for _, r := range rows {
		if r.Label == "" || r.ShotsFired <= 0 {
			continue
		}
		out = append(out, domain.SynthesisWeaponAccuracyEntry{
			Label:       r.Label,
			ShotsFired:  r.ShotsFired,
			ShotsLanded: r.ShotsLanded,
			Accuracy:    float64(r.ShotsLanded) / float64(r.ShotsFired),
		})
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Accuracy != out[j].Accuracy {
			return out[i].Accuracy > out[j].Accuracy
		}
		return out[i].Label < out[j].Label
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// buildCombatProfileFromCanonical agrège OC + DR depuis les rows canoniques filtrés
// et construit le CombatProfileBlock (descripteurs gérés par ClassifyCombatProfile).
// Retourne nil si aucun row valide (matchCount == 0).
func buildCombatProfileFromCanonical(rows []canonical.PlayerMatchRow, effectiveHpToKill float64) *domain.CombatProfileBlock {
	if len(rows) == 0 {
		return nil
	}
	var paceRatioSum float64
	var paceRatioCount int
	// Agrégats volume-pondérés (Σ sur tous les matchs) : base commune au rendement
	// (OC = 225·(Σkills+Σassists/3)/Σdégâts) et au dégâts/frag affiché. Pas de
	// moyenne des ratios par match (décrochait du chiffre affiché).
	var totalDmgDealt, totalDmgTaken float64
	var totalKills, totalAssists, totalDeaths int
	for _, r := range rows {
		if r.Self.DamageDealt != nil && r.Self.DamageTaken != nil {
			totalDmgDealt += float64(*r.Self.DamageDealt)
			totalDmgTaken += float64(*r.Self.DamageTaken)
			if r.Self.Kills != nil {
				totalKills += *r.Self.Kills
			}
			if r.Self.Assists != nil {
				totalAssists += *r.Self.Assists
			}
			if r.Self.Deaths != nil {
				totalDeaths += *r.Self.Deaths
			}
		}
		if r.Enrichment.EngagementPaceRatio != nil {
			paceRatioSum += *r.Enrichment.EngagementPaceRatio
			paceRatioCount++
		}
	}
	cy := analysis.ComputeCombatYieldFloat(float64(totalKills), float64(totalAssists), totalDmgDealt, totalDmgTaken, float64(totalDeaths), effectiveHpToKill)
	avgOC := cy.OffensiveConversion
	avgDR := cy.DefensiveResistance
	var avgPaceRatio *float64
	if paceRatioCount > 0 {
		v := paceRatioSum / float64(paceRatioCount)
		avgPaceRatio = &v
	}
	block := analysis.ClassifyCombatProfile(avgOC, avgDR, avgPaceRatio, len(rows))
	// Pas de damage_taken (ex. Halo 5, totalDmgTaken==0) → DR=0 trompeur : on
	// neutralise l'axe défensif (sinon « fragile » pour tous) et les dégâts/mort.
	if totalDmgTaken <= 0 {
		block.StyleDefensive = nil
	}
	// Dégâts par frag-équivalent (frags + assists/3) : aligné sur OC. DmgPerDeath brut.
	if v := analysis.DamagePerFragEquivalent(totalDmgDealt, float64(totalKills), float64(totalAssists)); v > 0 {
		block.DmgPerKill = &v
	}
	if totalDeaths > 0 && totalDmgTaken > 0 {
		v := totalDmgTaken / float64(totalDeaths)
		block.DmgPerDeath = &v
	}
	return &block
}

// Mappings award_name : betrayed_player -> betrayals, self_destruction -> suicides,
// destroyed_* -> vehicles_destroyed, hijacked_* -> hijacks.
func buildSynthesisFunStatsFromAwards(
	ctx context.Context,
	repo port.PersonalScoreAwardsRepository,
	titleSlug string,
	matchIDs []string,
	playerXUID string,
) (domain.SynthesisDetailedStats, error) {
	stats := domain.SynthesisDetailedStats{}

	rows, err := repo.LoadPersonalScoreAwards(ctx, titleSlug, port.PersonalScoreAwardsFilters{
		MatchIDs: matchIDs,
		XUIDs:    []string{playerXUID},
	})
	if err != nil {
		return stats, err
	}

	for _, row := range rows {
		switch row.AwardName {
		case "betrayed_player":
			stats.TotalBetrayals += row.Total
		case "self_destruction":
			stats.TotalSuicides += row.Total
		default:
			// destroyed_* et hijacked_* patterns
			if strings.HasPrefix(row.AwardName, "destroyed_") {
				stats.TotalVehiclesDestroyed += row.Total
			} else if strings.HasPrefix(row.AwardName, "hijacked_") {
				stats.TotalHijacks += row.Total
			}
		}
	}

	return stats, nil
}
