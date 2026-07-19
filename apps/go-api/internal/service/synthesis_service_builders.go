// Package service - synthesis_service_builders.go : builders highlights +
// detailed stats + top weapons + combat profile + fun stats (canonical
// path). Decoupe de synthesis_service.go (god-file split, refactor 2026-05-27).
package service

import (
	"context"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

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

// buildKillsByRole agrège les frags par rôle de combat (row.Role renseigné par le
// registre quand ResolveRoles=true). Les rows sans rôle (arme non mappée au
// registre) sont ignorées — title-agnostic, dégradation propre. Trié par kills desc
// (tie-break alpha pour un ordre stable). nil si aucun rôle résolu (→ champ omis).
func buildKillsByRole(rows []port.WeaponKillRow) []domain.SynthesisRoleKillEntry {
	byRole := make(map[string]int, 9)
	for _, r := range rows {
		if r.Role == "" || r.IsGrenadeMelee {
			continue
		}
		byRole[r.Role] += r.Kills
	}
	if len(byRole) == 0 {
		return nil
	}
	out := make([]domain.SynthesisRoleKillEntry, 0, len(byRole))
	for role, kills := range byRole {
		out = append(out, domain.SynthesisRoleKillEntry{Role: role, Kills: kills})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kills != out[j].Kills {
			return out[i].Kills > out[j].Kills
		}
		return out[i].Role < out[j].Role
	})
	return out
}

// canonicalFragClassOrder fixe l'ordre déterministe des classes du sunburst v2
// (§4 du plan). Toute sortie de buildFragDistribution suit cet ordre.
var canonicalFragClassOrder = []string{
	domain.FragClassShoulder, domain.FragClassSidearm, domain.FragClassHeavy,
	domain.FragClassMelee, domain.FragClassGrenade, domain.FragClassSpartanAbility,
	domain.FragClassUnattributed,
}

// gunFragClasses = classes dont la ventilation par arme vient du REGISTRE (estimé).
// Les autres classes (melee/grenade/spartan) sont servies par les totaux API, et
// les buckets non-combat H5 (vehicle/turret/…) retombent dans « Non attribué ».
var gunFragClasses = map[string]bool{
	domain.FragClassShoulder: true,
	domain.FragClassSidearm:  true,
	domain.FragClassHeavy:    true,
}

// buildFragDistribution assemble la répartition hiérarchique des frags (sunburst v2).
// Builder PUR (aucune IO, aucun log — le câblage service loggue les compteurs).
//
// Provenance (anti-double-source, §2) : classes gun shoulder/sidearm/heavy + rôles
// d'arme = registre (rows, Authoritative=false) ; classes melee/grenade/
// spartan_ability + total = stats API canoniques (Authoritative=true) ; unattributed
// = totalKills − Σ classes (résidu, ajouté si > 0). hasMechanics gate spartan_ability
// et le niveau 2 de Mêlée (invariant d — cap off ⇒ pas de spartan + Mêlée feuille).
func buildFragDistribution(
	rows []port.WeaponKillRow,
	stats domain.SynthesisDetailedStats,
	totalKills int,
	hasMechanics bool,
) domain.FragDistribution {
	byClass := make(map[string]domain.FragClassEntry, len(canonicalFragClassOrder))
	for _, e := range buildGunFragClasses(rows) {
		byClass[e.Class] = e
	}
	for _, e := range buildAPIFragClasses(stats, hasMechanics) {
		byClass[e.Class] = e
	}
	// Non attribué = résidu calculé (invariant a : Σ classes == total ; invariant c :
	// clamp >= 0). Un dépassement (Σ attribué > total) est une anomalie de données —
	// non ajouté ici, signalé par le câblage service (jamais avalé).
	sum := 0
	for _, e := range byClass {
		sum += e.Kills
	}
	if unattr := totalKills - sum; unattr > 0 {
		byClass[domain.FragClassUnattributed] = domain.FragClassEntry{
			Class: domain.FragClassUnattributed, Kills: unattr, Authoritative: false,
		}
	}
	classes := make([]domain.FragClassEntry, 0, len(byClass))
	for _, c := range canonicalFragClassOrder {
		if e, ok := byClass[c]; ok {
			classes = append(classes, e)
		}
	}
	return domain.FragDistribution{TotalKills: totalKills, Classes: classes}
}

// buildGunFragClasses agrège les classes gun (shoulder/sidearm/heavy) + leurs rôles
// depuis le registre (rows). Exclut les sentinels grenade/melee et les rows sans
// class/role résolu (dégradation propre, comme buildKillsByRole).
func buildGunFragClasses(rows []port.WeaponKillRow) []domain.FragClassEntry {
	type acc struct {
		kills  int
		byRole map[string]int
	}
	agg := make(map[string]*acc, len(gunFragClasses))
	for _, r := range rows {
		if r.IsGrenadeMelee || r.Class == "" || r.Role == "" || !gunFragClasses[r.Class] {
			continue
		}
		a := agg[r.Class]
		if a == nil {
			a = &acc{byRole: make(map[string]int)}
			agg[r.Class] = a
		}
		a.kills += r.Kills
		a.byRole[r.Role] += r.Kills
	}
	out := make([]domain.FragClassEntry, 0, len(agg))
	for class, a := range agg {
		if a.kills <= 0 {
			continue
		}
		out = append(out, domain.FragClassEntry{
			Class: class, Kills: a.kills, Authoritative: false,
			Roles: rolesFromMap(a.byRole, class),
		})
	}
	return out
}

// rolesFromMap trie les rôles (kills desc, tie-break alpha → ordre stable) et
// replie en FEUILLE une classe dont l'unique rôle porte son propre nom (ex. Poing/
// sidearm — D4 : pas de niveau 2 significatif).
func rolesFromMap(byRole map[string]int, class string) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, len(byRole))
	for role, kills := range byRole {
		if kills <= 0 {
			continue
		}
		roles = append(roles, domain.FragRoleEntry{Role: role, Kills: kills})
	}
	if len(roles) == 1 && roles[0].Role == class {
		return nil
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Kills != roles[j].Kills {
			return roles[i].Kills > roles[j].Kills
		}
		return roles[i].Role < roles[j].Role
	})
	return roles
}

// buildAPIFragClasses construit les classes servies par les totaux API canoniques
// (Authoritative=true) : Mêlée, Grenade, et — si hasMechanics — Capacités spartanes.
func buildAPIFragClasses(stats domain.SynthesisDetailedStats, hasMechanics bool) []domain.FragClassEntry {
	out := make([]domain.FragClassEntry, 0, 3)
	if mk := stats.TotalMeleeKills; mk > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassMelee, Kills: mk, Authoritative: true,
			Roles: meleeRoles(mk, stats.TotalAssassinations, hasMechanics),
		})
	}
	if gk := stats.TotalGrenadeKills; gk > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassGrenade, Kills: gk, Authoritative: true,
		})
	}
	if hasMechanics {
		if sk := stats.TotalGroundPoundKills + stats.TotalShoulderBashKills; sk > 0 {
			out = append(out, domain.FragClassEntry{
				Class: domain.FragClassSpartanAbility, Kills: sk, Authoritative: true,
				Roles: spartanRoles(stats.TotalGroundPoundKills, stats.TotalShoulderBashKills),
			})
		}
	}
	return out
}

// meleeRoles construit le niveau 2 de la classe Mêlée. FEUILLE sur Infinite (pas de
// mécanique native → invariant d). Sur H5 (hasMechanics) : Assassinat (total API) +
// Corps-à-corps direct.
//
// FORMULE PROVISOIRE : direct_melee = melee_total − assassination suppose
// assassination ⊆ melee. À CONFIRMER par la probe H5 (gate G2.3, phase P2) — si les
// deux compteurs sont disjoints, corriger ici. Cf. PLAN_FRAG_DISTRIBUTION_V2 §2.
// Clamp assassination ≤ melee pour garantir l'invariant (b) et un résidu ≥ 0.
func meleeRoles(meleeKills, assassinations int, hasMechanics bool) []domain.FragRoleEntry {
	if !hasMechanics {
		return nil
	}
	assass := assassinations
	if assass > meleeKills {
		assass = meleeKills
	}
	if assass < 0 {
		assass = 0
	}
	direct := meleeKills - assass
	roles := make([]domain.FragRoleEntry, 0, 2)
	if assass > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleAssassination, Kills: assass})
	}
	if direct > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleDirectMelee, Kills: direct})
	}
	return roles
}

// spartanRoles construit le niveau 2 des Capacités spartanes (ordre sémantique fixe :
// Frappe au sol puis Charge d'épaule). Σ rôles == kills de la classe (invariant b).
func spartanRoles(groundPound, shoulderBash int) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, 2)
	if groundPound > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleGroundPound, Kills: groundPound})
	}
	if shoulderBash > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleShoulderBash, Kills: shoulderBash})
	}
	return roles
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
