// Package analysis â€” home_kpis.go : KPIs globaux (ComputeKPIs / ComputeTrend),
// hero card (BuildHeroCard) et identitÃ© Spartan (BuildSpartanIdentity)
// pour la page d'accueil legacy.
package analysis

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

// ---------------------------------------------------------------------------

func dominantKey(counts map[string]int) (string, int) {
	var bestID string
	var bestCount int
	for id, n := range counts {
		if n > bestCount {
			bestID, bestCount = id, n
		}
	}
	return bestID, bestCount
}

// ---------------------------------------------------------------------------
// ComputeTrend â€” fenÃªtre glissante

// BuildSpartanIdentity construit le bloc identitaire compact de la home.
//
// ranks (peut Ãªtre nil) est consultÃ© pour rÃ©soudre RankTitle + NextRankTitle dans
// la locale demandÃ©e. Si nil ou si l'entrÃ©e est absente du catalog, fallback sur
// raw.RankName (libellÃ© prÃ©-construit cÃ´tÃ© player DB) puis sur "Rank N".
func BuildSpartanIdentity(raw *domain.HomeSpartanIdentityRow, locale string, ranks *mappings.RankCatalog) *domain.HomeSpartanIdentity {
	if raw == nil {
		return nil
	}

	identity := &domain.HomeSpartanIdentity{}
	if raw.SpartanID != nil {
		identity.SpartanID = copyOptionalString(raw.SpartanID)
	}
	if raw.BannerImageURL != nil {
		identity.BannerImageURL = copyOptionalString(raw.BannerImageURL)
	}
	if raw.EmblemImageURL != nil {
		identity.EmblemImageURL = copyOptionalString(raw.EmblemImageURL)
	}
	if raw.BackdropImageURL != nil {
		identity.BackdropImageURL = copyOptionalString(raw.BackdropImageURL)
	}
	if peak := buildHomeSkillPeak(raw.HighestCSR, locale); peak != nil {
		identity.HighestCSR = peak
	}
	if peak := buildHomeSkillPeak(raw.HighestLUSR, locale); peak != nil {
		identity.HighestLUSR = peak
	}
	if rank := buildHomeCareerRank(raw, locale, ranks); rank != nil {
		identity.CareerRank = rank
	}

	if identity.SpartanID == nil && identity.CareerRank == nil && identity.BannerImageURL == nil && identity.EmblemImageURL == nil && identity.BackdropImageURL == nil && identity.HighestCSR == nil && identity.HighestLUSR == nil {
		return nil
	}
	return identity
}

// buildHomeSkillPeak projette le row repo vers le DTO JSON home.
//
// Avant mai 2026 : retournait nil dès que RatingValue ≤ 0, ce qui masquait
// la phase de placement (rating effectif = 0 tant que les 10 matchs ne sont
// pas faits) ET les joueurs qui n'avaient pas encore de CSR fetché. Le front
// affichait alors faussement "En placement" via une heuristique côté UI.
//
// Désormais : on préserve le summary si on a au moins un signal exploitable
// (BadgeImageURL = unranked_N.png en placement, ou rating + tier en matured).
// Le front utilise MeasurementMatchesRemaining pour différencier les états
// sans deviner.
func buildHomeSkillPeak(raw *domain.HomeSkillPeakRow, locale string) *domain.HomeSkillPeakSummary {
	if raw == nil {
		return nil
	}
	if raw.RatingValue <= 0 && raw.BadgeImageURL == nil && raw.TierLabel == nil {
		return nil
	}
	summary := &domain.HomeSkillPeakSummary{
		RatingValue:   raw.RatingValue,
		TierLabel:     copyOptionalString(raw.TierLabel),
		BadgeImageURL: copyOptionalString(raw.BadgeImageURL),
	}
	if raw.MeasurementMatchesRemaining != nil {
		rem := *raw.MeasurementMatchesRemaining
		summary.MeasurementMatchesRemaining = &rem
	}
	if raw.PlacementTotal != nil {
		total := *raw.PlacementTotal
		summary.PlacementTotal = &total
	}
	// Date d'obtention du pic (nil = non sourçable : placement, match hors
	// registry, ou pic CSR all-time Waypoint). Propagée telle quelle au DTO JSON.
	if raw.PeakAchievedAt != nil {
		achieved := *raw.PeakAchievedAt
		summary.PeakAchievedAt = &achieved
	}
	// Bande de progression (à droite du rating) : uniquement en phase matured
	// (placement terminé). Progression ORDINALE via le sous-palier (indépendante
	// de l'échelle CSR vs LUSR). Onyx → barre pleine ; sans rang → pas de bande.
	isMatured := raw.MeasurementMatchesRemaining != nil && *raw.MeasurementMatchesRemaining == 0
	if isMatured {
		if pct, ok := SkillTierBand(raw.Tier, raw.SubTier); ok {
			summary.TierProgressPct = &pct
			// Extrémité droite de la barre : sous-palier suivant (nil pour Onyx).
			summary.NextTierLabel = NextSubTierLabel(raw.Tier, raw.SubTier, normalizeHomeLocale(locale) != "en")
		}
	}
	return summary
}

func buildHomeCareerRank(raw *domain.HomeSpartanIdentityRow, locale string, ranks *mappings.RankCatalog) *domain.HomeCareerRankSummary {
	if raw == nil || raw.RankNumber <= 0 {
		return nil
	}

	loc := normalizeHomeLocale(locale)

	// PrioritÃ© : RankCatalog (metadata.duckdb / GameCMS) > RankName (libellÃ©
	// prÃ©-build cÃ´tÃ© player DB) > RankTier > "Rang N". Le fallback player-DB
	// couvre les cas oÃ¹ le SemanticAdapter n'est pas injectÃ© dans le HomeService
	// (tests, mode dÃ©gradÃ©).
	title := rankSubRoman(lookupRankLabel(ranks, raw.RankNumber, loc))
	if title == "" {
		title = rankSubRoman(strings.TrimSpace(optionalStringValue(raw.RankName)))
	}
	if title == "" {
		title = strings.TrimSpace(optionalStringValue(raw.RankTier))
	}
	if title == "" {
		if loc == "en" {
			title = fmt.Sprintf("Rank %d", raw.RankNumber)
		} else {
			title = fmt.Sprintf("Rang %d", raw.RankNumber)
		}
	}

	// is_max_rank fiable : l'API Halo (RewardTrack.IsMaxRank) ne marque pas
	// toujours le dernier rang (Héros) comme max. On le dérive du catalog —
	// title-agnostic, sans magic number : un rang PRÉSENT dans le catalog mais
	// SANS rang suivant (id+1 absent) est le rang max. Le garde Get() évite un
	// faux positif si le rang du joueur n'est pas dans le catalog.
	isMax := raw.IsMaxRank
	if !isMax && ranks != nil {
		if _, has := ranks.Get(raw.RankNumber); has {
			if _, hasNext := ranks.Next(raw.RankNumber); !hasNext {
				isMax = true
			}
		}
	}

	var nextTitle string
	if !isMax && ranks != nil {
		if next, ok := ranks.Next(raw.RankNumber); ok {
			label, _ := next.FullLabel(loc)
			nextTitle = rankSubRoman(strings.TrimSpace(label))
		}
	}

	// Fallback xp_for_next : la source (DB locale) ne stocke pas toujours le seuil —
	// cas cible Explorer suivie, où career_progression renvoie 0 → barre à 0 %. On
	// lit alors le seuil du catalog (career_ranks.xp_required du rang courant), MÊME
	// valeur que CareerLiveRepo.EnrichFromMetadata utilisé par le chemin live/Home.
	xpForNext := raw.XPForNextRank
	if xpForNext <= 0 && !isMax && ranks != nil {
		if e, ok := ranks.Get(raw.RankNumber); ok && e.XPRequired > 0 {
			xpForNext = e.XPRequired
		}
	}

	// XP de carrière CUMULÉE : Σ seuils des rangs précédents + progression intra-rang.
	// Au rang max (Héros), la progression intra-rang vaut 0 → c'est ce total (« le
	// grand nombre ») qu'on affiche à la place. 0 si le catalog n'a pas les seuils.
	totalXP := raw.CurrentXP
	if ranks != nil && raw.RankNumber > 1 {
		totalXP += ranks.CumulativeXPRequired(raw.RankNumber - 1)
	}

	return &domain.HomeCareerRankSummary{
		RankNumber:        raw.RankNumber,
		RankTitle:         title,
		NextRankTitle:     nextTitle,
		RankImageURL:      copyOptionalString(raw.RankImageURL),
		AdornmentImageURL: copyOptionalString(raw.AdornmentImageURL),
		CurrentXP:         raw.CurrentXP,
		XPForNextRank:     xpForNext,
		TotalXP:           totalXP,
		ProgressPct:       computeHomeCareerProgressPct(raw.CurrentXP, xpForNext, isMax),
		IsMaxRank:         isMax,
	}
}

// lookupRankLabel retourne le libellÃ© localisÃ© du rang via le catalog, ou ""
// si ranks est nil ou si l'entrÃ©e est absente.
// rankSubRoman convertit tout sous-rang arabe isolé (1–6) en chiffre romain.
// Gère les positions finale ("Or 3") et médiane ("Général 2 Platine").
func rankSubRoman(label string) string {
	roman := [7]string{"", "I", "II", "III", "IV", "V", "VI"}
	out := label
	for d := byte('1'); d <= '6'; d++ {
		r := roman[d-'0']
		out = strings.ReplaceAll(out, " "+string(d)+" ", " "+r+" ")
		if strings.HasSuffix(out, " "+string(d)) {
			out = out[:len(out)-1] + r
		}
	}
	return out
}

func lookupRankLabel(ranks *mappings.RankCatalog, rankID int, locale string) string {
	if ranks == nil {
		return ""
	}
	label, ok := ranks.FullLabel(rankID, locale)
	if !ok {
		return ""
	}
	return strings.TrimSpace(label)
}

func computeHomeCareerProgressPct(currentXP, xpForNext int, isMaxRank bool) float64 {
	if isMaxRank {
		return 100.0
	}
	if xpForNext <= 0 {
		return 0.0
	}
	pct := float64(currentXP) / float64(xpForNext) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	return round2(pct)
}
