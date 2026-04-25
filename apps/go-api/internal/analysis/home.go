// Package analysis — home.go : algorithmes purs pour la page d'accueil Mission Control.
//
// Fonctions stateless : entrée = slices de domain rows, sortie = blocs JSON.
// Aucun accès DB, aucun import Streamlit.
package analysis

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Constantes outcome (codes numériques Halo Infinite)
// ---------------------------------------------------------------------------

const (
	homeOutcomeWin                = 2
	homeOutcomeLoss               = 3
	homeOutcomeTie                = 1
	homeOutcomeDNF                = 4
	homeDominanceDomination       = 1
	homeDominanceHumiliation      = 2
	homeDominanceRemontada        = 3
	homeDominanceDebacle          = 4
	homeDominanceCounterRemontada = 5
)

var homeOutcomeLabels = map[int]string{
	homeOutcomeWin:  "Victoire",
	homeOutcomeLoss: "Défaite",
	homeOutcomeTie:  "Égalité",
	homeOutcomeDNF:  "Abandon",
}

var homeOutcomeLabelsEN = map[int]string{
	homeOutcomeWin:  "Victory",
	homeOutcomeLoss: "Defeat",
	homeOutcomeTie:  "Tie",
	homeOutcomeDNF:  "DNF",
}

var homeOutcomeTones = map[int]string{
	homeOutcomeWin:  "win",
	homeOutcomeLoss: "loss",
	homeOutcomeTie:  "tie",
	homeOutcomeDNF:  "dnf",
}

var homeUUIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// labelFR retourne fr si non vide, sinon en.
func labelFR(fr, en string) string {
	if fr != "" {
		return fr
	}
	return en
}

func normalizeHomeLocale(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "en") {
		return "en"
	}
	return "fr"
}

func labelForLocale(locale, fr, en string) string {
	if normalizeHomeLocale(locale) == "en" {
		if strings.TrimSpace(en) != "" {
			return en
		}
		return fr
	}
	return labelFR(fr, en)
}

func outcomeLabelForLocale(outcome int, locale string) string {
	if normalizeHomeLocale(locale) == "en" {
		if label, ok := homeOutcomeLabelsEN[outcome]; ok {
			return label
		}
		return "Match"
	}
	if label, ok := homeOutcomeLabels[outcome]; ok {
		return label
	}
	return "Match"
}

func buildHomeScoreLabel(match domain.HomeMatchRow) *string {
	if match.Team0Score < 0 || match.Team1Score < 0 {
		return nil
	}

	leftScore := match.Team0Score
	rightScore := match.Team1Score
	if match.TeamID == 1 {
		leftScore = match.Team1Score
		rightScore = match.Team0Score
	}

	label := fmt.Sprintf("%d-%d", leftScore, rightScore)
	return &label
}

func buildHomeNarrativeBadges(dominanceFlag int) []string {
	switch dominanceFlag {
	case homeDominanceDomination:
		return []string{"dominant"}
	case homeDominanceHumiliation:
		return []string{"humiliation"}
	case homeDominanceRemontada:
		return []string{"remontada"}
	case homeDominanceDebacle:
		return []string{"debacle"}
	case homeDominanceCounterRemontada:
		return []string{"contre_remontada"}
	default:
		return nil
	}
}

// normalizeHomeModeLabel est un alias interne vers NormalizeModeLabel.
// Conservé pour ne pas casser les appelants internes au package.
func normalizeHomeModeLabel(raw string, mapLabels ...string) string {
	return NormalizeModeLabel(raw, mapLabels...)
}

// ---------------------------------------------------------------------------
// ComputeKPIs — KPIs globaux
// ---------------------------------------------------------------------------

// ComputeKPIs calcule les KPIs globaux depuis les matchs chargés.
// totalMatches est le nombre réel de matchs du joueur (pas limité par le LIMIT SQL).
func ComputeKPIs(matches []domain.HomeMatchRow, totalMatches int) domain.HeroKPIs {
	if len(matches) == 0 {
		return domain.HeroKPIs{}
	}
	var wins, losses, draws, dnfs int
	var ratioSum, ratioCount float64
	var kdaSum, kdaCount float64
	var accSum, accCount float64
	var totalPlaytime int
	var offSum, offCount float64
	var defSum, defCount float64
	playlistCounts := make(map[string]int)
	playlistNames := make(map[string]string)

	for _, m := range matches {
		switch m.Outcome {
		case homeOutcomeWin:
			wins++
		case homeOutcomeLoss:
			losses++
		case homeOutcomeTie:
			draws++
		case homeOutcomeDNF:
			dnfs++
		}
		if m.Ratio != nil {
			ratioSum += *m.Ratio
			ratioCount++
		}
		if m.KDA != nil {
			kdaSum += *m.KDA
			kdaCount++
		}
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accCount++
		}
		if m.TimePlayedSecs != nil {
			totalPlaytime += *m.TimePlayedSecs
		}
		if m.DamageDealt != nil && m.DamageTaken != nil {
			cy := ComputeCombatYield(m.Kills, m.Assists, *m.DamageDealt, *m.DamageTaken, m.Deaths)
			if cy.OffensiveConversion > 0 {
				offSum += cy.OffensiveConversion
				offCount++
			}
			if cy.DefensiveResistance > 0 {
				defSum += cy.DefensiveResistance
				defCount++
			}
		}
		if name := labelFR(m.PlaylistNameFR, m.PlaylistName); name != "" && !homeUUIDRe.MatchString(name) && m.PlaylistID != "" {
			playlistCounts[m.PlaylistID]++
			if _, seen := playlistNames[m.PlaylistID]; !seen {
				playlistNames[m.PlaylistID] = name
			}
		}
	}

	total := len(matches)
	kpis := domain.HeroKPIs{
		WinRate:           float64(wins) / float64(total),
		TotalMatches:      totalMatches,
		Wins:              wins,
		Draws:             draws,
		DNFs:              dnfs,
		Losses:            losses,
		TotalPlaytimeSecs: totalPlaytime,
	}
	if ratioCount > 0 {
		v := round2(ratioSum / ratioCount)
		kpis.GlobalRatio = &v
	}
	if kdaCount > 0 {
		v := round2(kdaSum / kdaCount)
		kpis.AvgKDA = &v
	}
	if accCount > 0 {
		v := round1(accSum / accCount)
		kpis.AvgAccuracy = &v
	}
	if offCount > 0 {
		v := round2(offSum / offCount)
		kpis.AvgOffensiveConversion = &v
	}
	if defCount > 0 {
		v := round2(defSum / defCount)
		kpis.AvgDefensiveResistance = &v
	}
	if bestID, bestCount := dominantKey(playlistCounts); bestID != "" {
		kpis.FavoritePlaylistName = playlistNames[bestID]
		kpis.FavoritePlaylistCount = bestCount
	}
	return kpis
}

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
// ComputeTrend — fenêtre glissante
// ---------------------------------------------------------------------------

// ComputeTrend calcule la tendance entre les [0:window] et [window:2*window] derniers matchs.
// Retourne nil si pas assez de données.
func ComputeTrend(matches []domain.HomeMatchRow, window int) *domain.HeroTrend {
	if len(matches) < window+1 {
		return nil
	}
	// matches est déjà trié DESC (les plus récents en premier, venant de Q26).
	recent := matches[:window]
	prev := matches[window : window+window]
	if len(prev) == 0 {
		return nil
	}

	trend := &domain.HeroTrend{}

	rCurr, rPrev := meanRatio(recent), meanRatio(prev)
	if rCurr != nil && rPrev != nil {
		v := round3(*rCurr - *rPrev)
		trend.RatioDelta = &v
	}

	aCurr, aPrev := meanAccuracy(recent), meanAccuracy(prev)
	if aCurr != nil && aPrev != nil {
		v := round2(*aCurr - *aPrev)
		trend.AccuracyDelta = &v
	}

	wrCurr := winRate(recent)
	wrPrev := winRate(prev)
	v := round4(wrCurr - wrPrev)
	trend.WinRateDelta = &v

	return trend
}

// ---------------------------------------------------------------------------
// BuildHeroCard — assemblage hero card
// ---------------------------------------------------------------------------

// BuildHeroCard construit le hero card pour un joueur.
// totalMatches est le nombre réel de matchs (sans LIMIT SQL).
func BuildHeroCard(matches []domain.HomeMatchRow, gamertag string, totalMatches int) domain.HomeHeroCard {
	kpis := ComputeKPIs(matches, totalMatches)
	trend := ComputeTrend(matches, 5)
	return domain.HomeHeroCard{PlayerName: gamertag, KPIs: kpis, Trend: trend}
}

// BuildSpartanIdentity construit le bloc identitaire compact de la home.
func BuildSpartanIdentity(raw *domain.HomeSpartanIdentityRow, locale string) *domain.HomeSpartanIdentity {
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
	if peak := buildHomeSkillPeak(raw.HighestCSR); peak != nil {
		identity.HighestCSR = peak
	}
	if peak := buildHomeSkillPeak(raw.HighestLUSR); peak != nil {
		identity.HighestLUSR = peak
	}
	if rank := buildHomeCareerRank(raw, locale); rank != nil {
		identity.CareerRank = rank
	}

	if identity.SpartanID == nil && identity.CareerRank == nil && identity.BannerImageURL == nil && identity.EmblemImageURL == nil && identity.BackdropImageURL == nil && identity.HighestCSR == nil && identity.HighestLUSR == nil {
		return nil
	}
	return identity
}

func buildHomeSkillPeak(raw *domain.HomeSkillPeakRow) *domain.HomeSkillPeakSummary {
	if raw == nil || raw.RatingValue <= 0 {
		return nil
	}
	return &domain.HomeSkillPeakSummary{
		RatingValue:   raw.RatingValue,
		TierLabel:     copyOptionalString(raw.TierLabel),
		BadgeImageURL: copyOptionalString(raw.BadgeImageURL),
	}
}

func buildHomeCareerRank(raw *domain.HomeSpartanIdentityRow, locale string) *domain.HomeCareerRankSummary {
	if raw == nil || raw.RankNumber <= 0 {
		return nil
	}

	title := strings.TrimSpace(labelForLocale(locale, optionalStringValue(raw.RankTitleFR), optionalStringValue(raw.RankTitleEN)))
	if title == "" {
		title = strings.TrimSpace(optionalStringValue(raw.RankName))
	}
	if title == "" {
		title = strings.TrimSpace(optionalStringValue(raw.RankTier))
	}
	if title == "" {
		if normalizeHomeLocale(locale) == "en" {
			title = fmt.Sprintf("Rank %d", raw.RankNumber)
		} else {
			title = fmt.Sprintf("Rang %d", raw.RankNumber)
		}
	}

	nextTitle := strings.TrimSpace(labelForLocale(locale, optionalStringValue(raw.NextRankTitleFR), optionalStringValue(raw.NextRankTitleEN)))

	return &domain.HomeCareerRankSummary{
		RankNumber:        raw.RankNumber,
		RankTitle:         title,
		NextRankTitle:     nextTitle,
		RankImageURL:      copyOptionalString(raw.RankImageURL),
		AdornmentImageURL: copyOptionalString(raw.AdornmentImageURL),
		CurrentXP:         raw.CurrentXP,
		XPForNextRank:     raw.XPForNextRank,
		ProgressPct:       computeHomeCareerProgressPct(raw.CurrentXP, raw.XPForNextRank, raw.IsMaxRank),
		IsMaxRank:         raw.IsMaxRank,
	}
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

func copyOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ---------------------------------------------------------------------------
// selectHighlightWindow — fenêtre de sessions similaires
// ---------------------------------------------------------------------------

// selectHighlightWindow sélectionne les matchs de la dernière session et des
// 4 sessions les plus récentes ayant la même composition (IsWithFriends) et
// la même playlist dominante (SkillPlaylistGroup).
// Fallback : si moins de 5 sessions similaires existent, toutes les sessions
// disponibles sont retournées. Les matchs sans SessionLabel ne font pas partie
// de la fenêtre calculée.
func selectHighlightWindow(matches []domain.HomeMatchRow) []domain.HomeMatchRow {
	if len(matches) == 0 {
		return nil
	}

	type sessionEntry struct {
		label         string
		isWithFriends bool
		playlistGroup string
		indices       []int
	}

	sessionOrder := []string{}
	sessionMap := map[string]*sessionEntry{}

	for i, m := range matches {
		if m.SessionLabel == nil {
			continue
		}
		lbl := *m.SessionLabel
		if _, exists := sessionMap[lbl]; !exists {
			sessionMap[lbl] = &sessionEntry{
				label:         lbl,
				isWithFriends: m.IsWithFriends,
			}
			sessionOrder = append(sessionOrder, lbl)
		}
		sessionMap[lbl].indices = append(sessionMap[lbl].indices, i)
	}

	if len(sessionOrder) == 0 {
		// Aucun match avec session_label : fallback sur les 50 premiers.
		if len(matches) > 50 {
			return matches[:50]
		}
		return matches
	}

	// Calculer la playlist dominante de chaque session.
	for _, lbl := range sessionOrder {
		entry := sessionMap[lbl]
		freq := map[string]int{}
		for _, idx := range entry.indices {
			if matches[idx].SkillPlaylistGroup != nil && *matches[idx].SkillPlaylistGroup != "" {
				freq[*matches[idx].SkillPlaylistGroup]++
			}
		}
		best, bestCount := "", 0
		for pg, cnt := range freq {
			if cnt > bestCount {
				bestCount = cnt
				best = pg
			}
		}
		entry.playlistGroup = best
	}

	// Session de référence = la plus récente (sessionOrder[0]).
	ref := sessionMap[sessionOrder[0]]

	// Collecter jusqu'à 5 sessions similaires (même composition + même playlist).
	collected := []string{}
	for _, lbl := range sessionOrder {
		e := sessionMap[lbl]
		if e.isWithFriends == ref.isWithFriends && e.playlistGroup == ref.playlistGroup {
			collected = append(collected, lbl)
			if len(collected) >= 5 {
				break
			}
		}
	}

	labelSet := map[string]bool{}
	for _, lbl := range collected {
		labelSet[lbl] = true
	}

	var window []domain.HomeMatchRow
	for _, m := range matches {
		if m.SessionLabel != nil && labelSet[*m.SessionLabel] {
			window = append(window, m)
		}
	}
	return window
}

// ---------------------------------------------------------------------------
// BuildHighlights — faits marquants
// ---------------------------------------------------------------------------

// highlightPerfColor retourne le niveau de couleur d'un score de performance.
// Les seuils sont identiques à ceux de perf-color.ts côté frontend.
func highlightPerfColor(perf float64) string {
	switch {
	case perf >= 80:
		return "perf-excellent"
	case perf >= 65:
		return "perf-good"
	case perf >= 50:
		return "perf-ok"
	case perf >= 35:
		return "perf-low"
	default:
		return "perf-bad"
	}
}

// highlightKDAColor retourne la couleur sémantique d'un FDA/KDA.
func highlightKDAColor(kda float64) string {
	switch {
	case kda > 1:
		return "positive"
	case kda >= 0:
		return "neutral"
	default:
		return "negative"
	}
}

// bestKDAMatch retourne le match avec le KDA le plus élevé dans la slice.
func bestKDAMatch(matches []domain.HomeMatchRow) *domain.HomeMatchRow {
	var best *domain.HomeMatchRow
	for i := range matches {
		if matches[i].KDA == nil {
			continue
		}
		if best == nil || *matches[i].KDA > *best.KDA {
			best = &matches[i]
		}
	}
	return best
}

// bestMMRUnderdogWin retourne le match victoire où le joueur avait le plus grand
// désavantage MMR (enemy_mmr - team_mmr maximal).
func bestMMRUnderdogWin(matches []domain.HomeMatchRow) *domain.HomeMatchRow {
	var best *domain.HomeMatchRow
	bestDelta := 0.0
	for i := range matches {
		m := &matches[i]
		if m.Outcome != homeOutcomeWin || m.TeamMMR == nil || m.EnemyMMR == nil {
			continue
		}
		delta := *m.EnemyMMR - *m.TeamMMR
		if best == nil || delta > bestDelta {
			bestDelta = delta
			best = m
		}
	}
	return best
}

// BuildHighlights construit les faits marquants depuis les matchs récents.
// La fenêtre est déterminée par selectHighlightWindow (dernière session + 4 similaires).
func BuildHighlights(matches []domain.HomeMatchRow) []domain.HighlightItem {
	if len(matches) == 0 {
		return nil
	}

	window := selectHighlightWindow(matches)
	if len(window) == 0 {
		return nil
	}

	var highlights []domain.HighlightItem

	// Highlight 1 : Performance moyenne.
	{
		var sum float64
		var n int
		for _, m := range window {
			if m.PerformanceScore != nil {
				sum += *m.PerformanceScore
				n++
			}
		}
		if n > 0 {
			avg := sum / float64(n)
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.perf_avg",
				Value:      fmt.Sprintf("%.0f", avg),
				ValueColor: highlightPerfColor(avg),
			})
		}
	}

	// Highlight 2 : Delta rang (LUSR ou CSR).
	{
		var deltaSum float64
		var n int
		csr, lusr := 0, 0
		for _, m := range window {
			if m.SkillRatingDelta != nil {
				deltaSum += *m.SkillRatingDelta
				n++
			}
			switch strings.ToUpper(m.SkillRatingType) {
			case "CSR":
				csr++
			case "LUSR":
				lusr++
			}
		}
		if n > 0 {
			titleKey := "highlight.title.skill_delta_lusr"
			if csr > lusr {
				titleKey = "highlight.title.skill_delta_csr"
			}
			sign := ""
			if deltaSum > 0 {
				sign = "+"
			}
			color := "neutral"
			if deltaSum > 0 {
				color = "positive"
			} else if deltaSum < 0 {
				color = "negative"
			}
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   titleKey,
				Value:      fmt.Sprintf("%s%.0fpts", sign, deltaSum),
				ValueColor: color,
			})
		}
	}

	// Highlight 3 : Plus belle victoire (plus grand désavantage MMR surmonté).
	{
		best := bestMMRUnderdogWin(window)
		if best != nil {
			// delta négatif = désavantage pour le joueur (team_mmr < enemy_mmr)
			delta := *best.TeamMMR - *best.EnemyMMR
			sign := ""
			if delta > 0 {
				sign = "+"
			}
			// Couleur : négatif = rouge (désavantage), ≈ 0 = bleu (équilibré), positif = vert (avantage).
			const mmrNeutralThreshold = 25.0
			color := "neutral"
			if delta > mmrNeutralThreshold {
				color = "positive"
			} else if delta < -mmrNeutralThreshold {
				color = "negative"
			}
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.best_underdog_win",
				Value:      fmt.Sprintf("%s%.0f MMR", sign, delta),
				Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
				ValueColor: color,
			})
		}
	}

	// Highlight 4 : Pic FDA récent (meilleur KDA sur toutes les sessions sélectionnées).
	{
		best := bestKDAMatch(window)
		if best != nil && best.KDA != nil {
			highlights = append(highlights, domain.HighlightItem{
				TitleKey:   "highlight.title.kda_peak",
				Value:      fmt.Sprintf("%.2f", *best.KDA),
				Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
				ValueColor: highlightKDAColor(*best.KDA),
			})
		}
	}

	// Highlight 5 : Maîtrise — tuile à défilement (tirs à la tête · frags parfaits · précision moyenne).
	if maitrise := buildMaitriseHighlight(window); maitrise != nil {
		highlights = append(highlights, *maitrise)
	}

	// Highlight 6 : Stats par min. — tuile à défilement (frags · morts · assistances par minute).
	if perMin := buildPerMinuteHighlight(window); perMin != nil {
		highlights = append(highlights, *perMin)
	}

	// Highlight 7 : Volume — nombre de parties et contexte FDA moyen / taux de victoire.
	{
		var kdaSum float64
		var kdaN, wins int
		for _, m := range window {
			if m.KDA != nil {
				kdaSum += *m.KDA
				kdaN++
			}
			if m.Outcome == homeOutcomeWin {
				wins++
			}
		}
		wr := int(math.Round(float64(wins) / float64(len(window)) * 100))
		params := map[string]any{"wr": wr}
		detailKey := "highlight.detail.volume_wr"
		if kdaN > 0 {
			detailKey = "highlight.detail.volume_kda_wr"
			params["kda"] = fmt.Sprintf("%.2f", kdaSum/float64(kdaN))
		}
		highlights = append(highlights, domain.HighlightItem{
			TitleKey:     "highlight.title.volume",
			Value:        fmt.Sprintf("%d", len(window)),
			DetailKey:    detailKey,
			DetailParams: params,
		})
	}

	// Highlight 8 : Série — tuile à défilement (folie meurtrière max · victoires consécutives · carte fétiche).
	if serie := buildSerieHighlight(window); serie != nil {
		highlights = append(highlights, *serie)
	}

	return highlights
}

// buildMaitriseHighlight assemble la tuile « Maîtrise » (3 slides à défilement) :
//  1. Tirs à la tête (somme) sur la fenêtre
//  2. Frags parfaits (somme)
//  3. Précision moyenne (moyenne arithmétique des matchs renseignés)
//
// Retourne nil si aucun slide exploitable.
func buildMaitriseHighlight(window []domain.HomeMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide

	// Slide 1 : Tirs à la tête (somme).
	var hsSum, perfSum int
	for _, m := range window {
		hsSum += m.HeadshotKills
		perfSum += m.PerfectKills
	}
	hsColor := "neutral"
	if hsSum > 0 {
		hsColor = "positive"
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey:   "highlight.slide.headshots",
		Value:      fmt.Sprintf("%d", hsSum),
		ValueColor: hsColor,
	})

	// Slide 2 : Frags parfaits (somme).
	perfColor := "neutral"
	if perfSum > 0 {
		perfColor = "positive"
	}
	slides = append(slides, domain.HighlightSlide{
		LabelKey:   "highlight.slide.perfect_kills",
		Value:      fmt.Sprintf("%d", perfSum),
		ValueColor: perfColor,
	})

	// Slide 3 : Précision moyenne.
	var accSum float64
	var accN int
	for _, m := range window {
		if m.Accuracy != nil {
			accSum += *m.Accuracy
			accN++
		}
	}
	if accN > 0 {
		avg := accSum / float64(accN)
		// Seuils alignés sur Performance globale (HomePage.tsx) : > 55 vert, ≥ 40 ambre, sinon rouge.
		color := "negative"
		if avg > 55 {
			color = "positive"
		} else if avg >= 40 {
			color = "warning"
		}
		slides = append(slides, domain.HighlightSlide{
			LabelKey:   "highlight.slide.accuracy",
			Value:      fmt.Sprintf("%.0f%%", avg),
			ValueColor: color,
		})
	}

	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey:   "highlight.title.mastery",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

// buildPerMinuteHighlight assemble la tuile « Stats par min. » (3 slides à défilement) :
//  1. Frags / min (kills cumulés / minutes jouées cumulées)
//  2. Morts / min
//  3. Assistances / min
//
// Seuls les matchs avec time_played_seconds > 0 sont comptés. Retourne nil si aucun match exploitable.
func buildPerMinuteHighlight(window []domain.HomeMatchRow) *domain.HighlightItem {
	var totalSecs float64
	var kills, deaths, assists, n int
	for _, m := range window {
		if m.TimePlayedSecs == nil || *m.TimePlayedSecs <= 0 {
			continue
		}
		totalSecs += float64(*m.TimePlayedSecs)
		kills += m.Kills
		deaths += m.Deaths
		assists += m.Assists
		n++
	}
	if n == 0 || totalSecs <= 0 {
		return nil
	}
	minutes := totalSecs / 60.0
	slides := []domain.HighlightSlide{
		{LabelKey: "highlight.slide.kills", Value: fmt.Sprintf("%.2f", float64(kills)/minutes), ValueColor: "neutral"},
		{LabelKey: "highlight.slide.deaths", Value: fmt.Sprintf("%.2f", float64(deaths)/minutes), ValueColor: "neutral"},
		{LabelKey: "highlight.slide.assists", Value: fmt.Sprintf("%.2f", float64(assists)/minutes), ValueColor: "neutral"},
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey:   "highlight.title.per_minute",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

// buildSerieHighlight assemble la tuile « Série » (3 slides à défilement) :
//  1. Folie meurtrière (max) sur la fenêtre
//  2. Victoires consécutives (plus longue séquence)
//  3. Carte fétiche (meilleur taux de victoire, min 2 parties)
//
// Retourne nil si aucun slide n'a pu être calculé.
func buildSerieHighlight(window []domain.HomeMatchRow) *domain.HighlightItem {
	var slides []domain.HighlightSlide

	// Slide 1 : Folie meurtrière max.
	if s := sliceBestKillingSpree(window); s != nil {
		slides = append(slides, *s)
	}
	// Slide 2 : Plus longue série de victoires.
	if s := sliceBestWinStreak(window); s != nil {
		slides = append(slides, *s)
	}
	// Slide 3 : Carte fétiche.
	if s := sliceFavoriteMap(window); s != nil {
		slides = append(slides, *s)
	}

	if len(slides) == 0 {
		return nil
	}
	first := slides[0]
	return &domain.HighlightItem{
		TitleKey:   "highlight.title.serie",
		Value:      first.Value,
		Detail:     first.Detail,
		ValueColor: first.ValueColor,
		Slides:     slides,
	}
}

func sliceBestKillingSpree(window []domain.HomeMatchRow) *domain.HighlightSlide {
	var best *domain.HomeMatchRow
	bestVal := 0
	for i := range window {
		m := &window[i]
		if m.MaxKillingSpree == nil || *m.MaxKillingSpree <= 0 {
			continue
		}
		if best == nil || *m.MaxKillingSpree > bestVal {
			bestVal = *m.MaxKillingSpree
			best = m
		}
	}
	if best == nil {
		return nil
	}
	return &domain.HighlightSlide{
		LabelKey:   "highlight.slide.killing_spree_max",
		Value:      fmt.Sprintf("%d", bestVal),
		Detail:     fmt.Sprintf("%s · %s", labelFR(best.MapNameFR, best.MapName), labelFR(best.PairNameFR, best.PairName)),
		ValueColor: "positive",
	}
}

func sliceBestWinStreak(window []domain.HomeMatchRow) *domain.HighlightSlide {
	if len(window) == 0 {
		return nil
	}
	// window est triée start_time DESC : on parcourt en inverse pour ordre chronologique.
	best, cur := 0, 0
	for i := len(window) - 1; i >= 0; i-- {
		if window[i].Outcome == homeOutcomeWin {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	if best == 0 {
		return nil
	}
	color := "neutral"
	if best >= 3 {
		color = "positive"
	}
	return &domain.HighlightSlide{
		LabelKey:     "highlight.slide.win_streak",
		Value:        fmt.Sprintf("%d", best),
		DetailKey:    "highlight.detail.win_streak",
		DetailParams: map[string]any{"count": best},
		ValueColor:   color,
	}
}

func sliceFavoriteMap(window []domain.HomeMatchRow) *domain.HighlightSlide {
	type stat struct {
		name   string
		nameFR string
		plays  int
		wins   int
	}
	byMap := map[string]*stat{}
	for _, m := range window {
		if m.MapID == "" {
			continue
		}
		s, ok := byMap[m.MapID]
		if !ok {
			s = &stat{name: m.MapName, nameFR: m.MapNameFR}
			byMap[m.MapID] = s
		}
		s.plays++
		if m.Outcome == homeOutcomeWin {
			s.wins++
		}
	}
	var best *stat
	bestWR := -1.0
	for _, s := range byMap {
		if s.plays < 2 {
			continue
		}
		wr := float64(s.wins) / float64(s.plays)
		// Départage : WR, puis nombre de parties (plus = plus fiable).
		if wr > bestWR || (wr == bestWR && best != nil && s.plays > best.plays) {
			bestWR = wr
			best = s
		}
	}
	if best == nil {
		return nil
	}
	color := "neutral"
	if bestWR >= 0.6 {
		color = "positive"
	} else if bestWR < 0.4 {
		color = "negative"
	}
	return &domain.HighlightSlide{
		LabelKey:  "highlight.slide.favorite_map",
		Value:     labelFR(best.nameFR, best.name),
		DetailKey: "highlight.detail.favorite_map",
		DetailParams: map[string]any{
			"wins":   best.wins,
			"losses": best.plays - best.wins,
			"wr":     int(bestWR*100 + 0.5),
		},
		ValueColor: color,
	}
}

// ---------------------------------------------------------------------------
// BuildRecentMatches — timeline récente
// ---------------------------------------------------------------------------

// mapPNGNames contient les noms de maps (EN) dont l'image locale est au format PNG.
// Tous les autres noms utilisent le format JPEG par défaut.
var mapPNGNames = map[string]struct{}{
	"Aquarius":                 {},
	"Aquarius - Ranked":        {},
	"Bazaar":                   {},
	"Behemoth":                 {},
	"Breaker":                  {},
	"Breaker Heavies":          {},
	"Catalyst":                 {},
	"Deadlock":                 {},
	"Deadlock Heavies":         {},
	"Highpower":                {},
	"Highpower Heavies":        {},
	"Highpower Sentry Defense": {},
	"Launch Site":              {},
	"Recharge":                 {},
	"Recharge - Ranked":        {},
	"Streets":                  {},
	"Streets - Ranked":         {},
}

// mapStaticImagePath retourne l'URL relative de l'image de map servie par /static/maps/.
// Le nom de la map est encodé pour les espaces et caractères spéciaux.
func mapStaticImagePath(mapName string) string {
	mapName = strings.TrimSpace(mapName)
	if mapName == "" || homeUUIDRe.MatchString(mapName) {
		return ""
	}
	ext := ".jpg"
	if _, ok := mapPNGNames[mapName]; ok {
		ext = ".png"
	}
	// Encoder les espaces manuellement — net/url.PathEscape encode aussi "/" ce qu'on ne veut pas.
	encoded := ""
	for _, c := range mapName {
		if c == ' ' {
			encoded += "%20"
		} else {
			encoded += string(c)
		}
	}
	return "/static/maps/" + encoded + ext
}

// BuildRecentMatches construit la liste des derniers matchs pour la timeline.
func BuildRecentMatches(matches []domain.HomeMatchRow, limit int) []domain.RecentMatchItem {
	return BuildRecentMatchesForLocale(matches, limit, "fr")
}

// BuildRecentMatchesForLocale construit la liste des derniers matchs pour la locale demandée.
func BuildRecentMatchesForLocale(matches []domain.HomeMatchRow, limit int, locale string) []domain.RecentMatchItem {
	return BuildRecentMatchesWithFavoritesForLocale(matches, limit, nil, locale)
}

// BuildRecentMatchesWithFavorites construit la liste des derniers matchs avec le flag favori.
// favoriteIDs est un set de match_id favoris (nil = social repo indisponible).
func BuildRecentMatchesWithFavorites(matches []domain.HomeMatchRow, limit int, favoriteIDs map[string]bool) []domain.RecentMatchItem {
	return BuildRecentMatchesWithFavoritesForLocale(matches, limit, favoriteIDs, "fr")
}

// BuildRecentMatchesWithFavoritesForLocale construit la liste des derniers matchs avec le flag favori
// en choisissant les labels selon la langue active de l'interface.
func BuildRecentMatchesWithFavoritesForLocale(matches []domain.HomeMatchRow, limit int, favoriteIDs map[string]bool, locale string) []domain.RecentMatchItem {
	if len(matches) == 0 {
		return nil
	}
	locale = normalizeHomeLocale(locale)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	items := make([]domain.RecentMatchItem, 0, len(matches))
	for _, m := range matches {
		if m.MatchID == "" {
			continue
		}
		label := outcomeLabelForLocale(m.Outcome, locale)
		tone := outcomeTone(m.Outcome)
		ratioStr := "-"
		if m.Ratio != nil {
			ratioStr = fmt.Sprintf("%.2f", *m.Ratio)
		}
		accStr := "-"
		if m.Accuracy != nil {
			accStr = fmt.Sprintf("%.0f%%", *m.Accuracy)
		}
		t := m.StartTime

		mapUI := labelForLocale(locale, m.MapNameFR, m.MapName)
		modeUI := normalizeHomeModeLabel(labelForLocale(locale, m.PairNameFR, m.PairName), m.MapNameFR, m.MapName)
		var playlistUI *string
		if m.PlaylistName != "" || m.PlaylistNameFR != "" {
			playlist := labelForLocale(locale, m.PlaylistNameFR, m.PlaylistName)
			playlistUI = &playlist
		}
		scoreLabel := buildHomeScoreLabel(m)
		narrativeBadges := buildHomeNarrativeBadges(m.DominanceFlag)

		kills := m.Kills
		deaths := m.Deaths
		assists := m.Assists

		var perfScoreRel *int
		if m.PerformanceScore != nil {
			v := int(math.Round(*m.PerformanceScore))
			perfScoreRel = &v
		}

		var skillRatingVal *int
		var skillRatingType *string
		if m.SkillRatingValue != nil {
			v := int(math.Round(*m.SkillRatingValue))
			skillRatingVal = &v
			t := m.SkillRatingType
			skillRatingType = &t
		}

		// Enrichissement skill tier : passer les champs nullable directement.
		var skillTierLabel *string
		if m.SkillTierLabel != nil && *m.SkillTierLabel != "" {
			skillTierLabel = m.SkillTierLabel
		}
		var skillPlaylistGroup *string
		if m.SkillPlaylistGroup != nil && *m.SkillPlaylistGroup != "" {
			skillPlaylistGroup = m.SkillPlaylistGroup
		}

		// Progression dans le tier : approximation sur une fenêtre de 50 pts.
		var skillProgressPct *float64
		var skillPointsInTier *int
		if m.SkillRatingValue != nil {
			const tierSize = 50.0
			pts := math.Mod(*m.SkillRatingValue, tierSize)
			if pts < 0 {
				pts += tierSize
			}
			pct := pts / tierSize * 100.0
			skillProgressPct = &pct
			p := int(math.Round(pts))
			skillPointsInTier = &p
		}

		var offConv, defRes *float64
		dd := float64PtrVal(m.DamageDealt)
		dt := float64PtrVal(m.DamageTaken)
		if dd > 0 || dt > 0 {
			cy := ComputeCombatYield(m.Kills, m.Assists, dd, dt, m.Deaths)
			if dd > 0 {
				offConv = &cy.OffensiveConversion
			}
			if dt > 0 && m.Deaths > 0 {
				defRes = &cy.DefensiveResistance
			}
		}

		isFav := favoriteIDs[m.MatchID]

		// Précision brute (de HomeMatchRow, déjà en %).
		var accuracy *float64
		if m.Accuracy != nil {
			accuracy = m.Accuracy
		}

		// Solo / Escouade
		isWithFriends := m.IsWithFriends
		iwf := &isWithFriends

		// Préférer l'asset local /static/maps pour les maps connues ; fallback cache-aside sinon.
		mapImageURL := buildMapImageURL("halo_infinite", m.MapID, m.MapName, m.MapNameFR)

		items = append(items, domain.RecentMatchItem{
			MatchID:                  m.MatchID,
			Title:                    fmt.Sprintf("%s · %s", label, mapUI),
			Detail:                   fmt.Sprintf("%s · KD %s · %s", modeUI, ratioStr, accStr),
			StartedAt:                &t,
			OutcomeLabel:             label,
			OutcomeTone:              tone,
			ScoreLabel:               scoreLabel,
			NarrativeBadges:          narrativeBadges,
			IsFavorite:               isFav,
			MapUI:                    &mapUI,
			ModeUI:                   &modeUI,
			PlaylistUI:               playlistUI,
			MapImageURL:              mapImageURL,
			Kills:                    &kills,
			Deaths:                   &deaths,
			Assists:                  &assists,
			PerformanceScoreRelative: perfScoreRel,
			OffensiveConversion:      offConv,
			DefensiveResistance:      defRes,
			DamageDealt:              m.DamageDealt,
			DamageTaken:              m.DamageTaken,
			SkillRatingValue:         skillRatingVal,
			SkillRatingType:          skillRatingType,
			SkillTierLabel:           skillTierLabel,
			SkillRatingDelta:         m.SkillRatingDelta,
			SkillPlaylistGroup:       skillPlaylistGroup,
			SkillRankImageURL:        m.SkillRankImageURL,
			SkillProgressPct:         skillProgressPct,
			SkillPointsInTier:        skillPointsInTier,
			KDA:                      m.KDA,
			DurationSecs:             m.TimePlayedSecs,
			Accuracy:                 accuracy,
			AvgLifeSecs:              m.AvgLifeSeconds,
			TeamMMR:                  m.TeamMMR,
			EnemyMMR:                 m.EnemyMMR,
			DeltaMMR:                 mmrDelta(m.TeamMMR, m.EnemyMMR),
			IsWithFriends:            iwf,
			RankInTeam:               m.RankInTeam,
			HeadshotKills:            intPtrIfPos(m.HeadshotKills),
			PerfectKills:             intPtrIfPos(m.PerfectKills),
		})
	}
	return items
}

// buildMapImageURL construit l'URL d'image d'une map.
// Priorité au fichier statique local quand le nom de map est connu ; fallback sur le cache-aside UUID sinon.
func buildMapImageURL(titleID, mapID, mapName, mapNameFR string) *string {
	if localPath := mapStaticImagePath(mapName); localPath != "" {
		return &localPath
	}
	if localPath := mapStaticImagePath(mapNameFR); localPath != "" {
		return &localPath
	}
	if mapID == "" {
		return nil
	}
	url := fmt.Sprintf("/api/v1/assets/maps/%s/%s/image", titleID, mapID)
	return &url
}

// mmrDelta calcule team_mmr - enemy_mmr ; retourne nil si l'un ou l'autre est absent.
func mmrDelta(team, enemy *float64) *float64 {
	if team == nil || enemy == nil {
		return nil
	}
	v := *team - *enemy
	return &v
}

// float64PtrVal retourne la valeur pointée ou 0 si nil.
func float64PtrVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// ---------------------------------------------------------------------------
// BuildSessionSummaries / BuildSessionSummary — résumés de sessions
// ---------------------------------------------------------------------------

// BuildSessionSummaries construit la liste des N dernières sessions (solo ou escouade),
// triées de la plus récente à la plus ancienne.
func BuildSessionSummaries(
	matches []domain.HomeMatchRow,
	sessions []domain.HomeSessionRow,
	squadMode bool,
	limit int,
) []domain.SessionSummaryItem {
	if len(sessions) == 0 || len(matches) == 0 {
		return nil
	}

	// Filtrer par mode.
	var filtered []domain.HomeSessionRow
	for _, s := range sessions {
		if s.IsWithFriends == squadMode && s.SessionLabel != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Collecter les labels distincts triés par date décroissante.
	labels := distinctSessionLabels(filtered)

	// Construire le résumé pour chaque label, jusqu'à la limite.
	var result []domain.SessionSummaryItem
	for _, lbl := range labels {
		if limit > 0 && len(result) >= limit {
			break
		}
		// Rassembler les match_ids de ce label.
		matchIDSet := make(map[string]bool)
		for _, s := range filtered {
			if s.SessionLabel != nil && *s.SessionLabel == lbl {
				matchIDSet[s.MatchID] = true
			}
		}
		// Filtrer les matchs.
		var sessionMatches []domain.HomeMatchRow
		for _, m := range matches {
			if matchIDSet[m.MatchID] {
				sessionMatches = append(sessionMatches, m)
			}
		}
		if len(sessionMatches) == 0 {
			continue
		}

		// Compter les outcomes.
		var wins, losses, draws, dnfs int
		for _, m := range sessionMatches {
			switch m.Outcome {
			case domain.OutcomeWin:
				wins++
			case domain.OutcomeLoss:
				losses++
			case domain.OutcomeDraw:
				draws++
			default:
				dnfs++
			}
		}

		// Performance joueur : toujours la moyenne des PerformanceScore personnels.
		var avgPlayerPerf *float64
		{
			var sum float64
			var count int
			for _, m := range sessionMatches {
				if m.PerformanceScore != nil {
					sum += *m.PerformanceScore
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgPlayerPerf = &v
			}
		}

		// Performance équipe : uniquement en mode escouade.
		var avgTeamPerf *float64
		if squadMode {
			var scores []*float64
			var winRates, kdas, kills []float64
			for _, m := range sessionMatches {
				scores = append(scores, m.PerformanceScore)
				wr := 0.0
				if m.Outcome == domain.OutcomeWin {
					wr = 100.0
				} else if m.Outcome == domain.OutcomeDraw {
					wr = 50.0
				}
				winRates = append(winRates, wr)
				kda := 0.0
				if m.KDA != nil {
					kda = *m.KDA
				}
				kdas = append(kdas, kda)
				kills = append(kills, float64(m.Kills))
			}
			sq := ComputeSquadPerformanceScore(scores, winRates, kdas, kills)
			avgTeamPerf = sq.Score
		}

		// K/D moyen sur la session.
		var avgKDA *float64
		{
			var sum float64
			var count int
			for _, m := range sessionMatches {
				if m.KDA != nil {
					sum += *m.KDA
					count++
				}
			}
			if count > 0 {
				v := round1(sum / float64(count))
				avgKDA = &v
			}
		}

		// Mode dominant : pair (map+mode) le plus joué sur la session (nom FR).
		var dominantMode *string
		{
			freq := make(map[string]int)
			for _, m := range sessionMatches {
				if m.PairNameFR != "" {
					freq[m.PairNameFR]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantMode = &best
			}
		}

		// Playlist dominante : playlist FR la plus jouée sur la session.
		var dominantPlaylist *string
		{
			freq := make(map[string]int)
			for _, m := range sessionMatches {
				name := m.PlaylistNameFR
				if name == "" {
					name = m.PlaylistName
				}
				if name != "" {
					freq[name]++
				}
			}
			var best string
			var bestCount int
			for name, cnt := range freq {
				if cnt > bestCount || (cnt == bestCount && name < best) {
					best = name
					bestCount = cnt
				}
			}
			if best != "" {
				dominantPlaylist = &best
			}
		}

		kpis := ComputeKPIs(sessionMatches, len(sessionMatches))
		item := domain.SessionSummaryItem{
			SessionLabel:         lbl,
			MatchCount:           len(sessionMatches),
			WinRate:              kpis.WinRate,
			GlobalRatio:          kpis.GlobalRatio,
			Wins:                 wins,
			Losses:               losses,
			Draws:                draws,
			DNFs:                 dnfs,
			AvgPlayerPerformance: avgPlayerPerf,
			AvgTeamPerformance:   avgTeamPerf,
			AvgKDA:               avgKDA,
			DominantPlaylist:     dominantPlaylist,
			DominantMode:         dominantMode,
		}
		if earliest := earliestStartTime(sessionMatches); earliest != nil {
			item.StartedAt = earliest
		}
		if ended := latestEndTime(sessionMatches); ended != nil {
			item.EndedAt = ended
		}
		result = append(result, item)
	}
	return result
}

// distinctSessionLabels retourne les labels distincts triés par start_time DESC.
func distinctSessionLabels(sessions []domain.HomeSessionRow) []string {
	// Calculer le start_time max par label.
	labelTimes := make(map[string]time.Time)
	for _, s := range sessions {
		if s.SessionLabel == nil || *s.SessionLabel == "" {
			continue
		}
		lbl := *s.SessionLabel
		if s.StartTime != nil {
			if t, ok := labelTimes[lbl]; !ok || s.StartTime.After(t) {
				labelTimes[lbl] = *s.StartTime
			}
		} else {
			if _, ok := labelTimes[lbl]; !ok {
				labelTimes[lbl] = time.Time{}
			}
		}
	}
	labels := make([]string, 0, len(labelTimes))
	for lbl := range labelTimes {
		labels = append(labels, lbl)
	}
	sort.Slice(labels, func(i, j int) bool {
		return labelTimes[labels[i]].After(labelTimes[labels[j]])
	})
	return labels
}

// BuildSessionSummary construit le résumé de la dernière session solo ou escouade.
func BuildSessionSummary(
	matches []domain.HomeMatchRow,
	sessions []domain.HomeSessionRow,
	squadMode bool,
) *domain.SessionSummaryItem {
	if len(sessions) == 0 || len(matches) == 0 {
		return nil
	}

	// Filtrer les sessions par mode.
	var filtered []domain.HomeSessionRow
	for _, s := range sessions {
		if s.IsWithFriends == squadMode && s.SessionLabel != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	// Trouver le label de la session la plus récente.
	latestLabel := latestSessionLabel(filtered)
	if latestLabel == "" {
		return nil
	}

	// Rassembler les match_ids de cette session.
	matchIDSet := make(map[string]bool)
	for _, s := range filtered {
		if s.SessionLabel != nil && *s.SessionLabel == latestLabel {
			matchIDSet[s.MatchID] = true
		}
	}
	if len(matchIDSet) == 0 {
		return nil
	}

	// Filtrer les matchs de la session.
	var sessionMatches []domain.HomeMatchRow
	for _, m := range matches {
		if matchIDSet[m.MatchID] {
			sessionMatches = append(sessionMatches, m)
		}
	}
	if len(sessionMatches) == 0 {
		return nil
	}

	kpis := ComputeKPIs(sessionMatches, len(sessionMatches))
	item := &domain.SessionSummaryItem{
		SessionLabel: latestLabel,
		MatchCount:   len(sessionMatches),
		WinRate:      kpis.WinRate,
		GlobalRatio:  kpis.GlobalRatio,
	}
	// Trouver le start_time le plus ancien de la session.
	if earliest := earliestStartTime(sessionMatches); earliest != nil {
		item.StartedAt = earliest
	}
	return item
}

// ---------------------------------------------------------------------------
// BuildRecentMedia — médias récents
// ---------------------------------------------------------------------------

// BuildRecentMedia transforme les lignes DuckDB en items de médias récents.
func BuildRecentMedia(media []domain.HomeMediaRow, limit int) []domain.RecentMediaItem {
	if len(media) == 0 {
		return nil
	}
	if len(media) > limit {
		media = media[:limit]
	}
	items := make([]domain.RecentMediaItem, 0, len(media))
	for _, m := range media {
		if m.FileName == "" {
			continue
		}
		items = append(items, domain.RecentMediaItem{
			Basename:       m.FileName,
			MatchID:        m.MatchID,
			MatchStartTime: m.MatchStartTime,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func meanRatio(matches []domain.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Ratio != nil {
			sum += *m.Ratio
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func meanAccuracy(matches []domain.HomeMatchRow) *float64 {
	var sum, count float64
	for _, m := range matches {
		if m.Accuracy != nil {
			sum += *m.Accuracy
			count++
		}
	}
	if count == 0 {
		return nil
	}
	v := sum / count
	return &v
}

func winRate(matches []domain.HomeMatchRow) float64 {
	if len(matches) == 0 {
		return 0
	}
	var wins int
	for _, m := range matches {
		if m.Outcome == homeOutcomeWin {
			wins++
		}
	}
	return float64(wins) / float64(len(matches))
}

func bestRatioMatch(matches []domain.HomeMatchRow) *domain.HomeMatchRow {
	var best *domain.HomeMatchRow
	for i := range matches {
		if matches[i].Ratio == nil {
			continue
		}
		if best == nil || *matches[i].Ratio > *best.Ratio {
			best = &matches[i]
		}
	}
	return best
}

func outcomeLabel(code int) string {
	if l, ok := homeOutcomeLabels[code]; ok {
		return l
	}
	return "DNF"
}

func outcomeTone(code int) string {
	if t, ok := homeOutcomeTones[code]; ok {
		return t
	}
	return "dnf"
}

func latestSessionLabel(sessions []domain.HomeSessionRow) string {
	// Trier par start_time DESC, prendre le premier session_label.
	sorted := make([]domain.HomeSessionRow, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartTime == nil {
			return false
		}
		if sorted[j].StartTime == nil {
			return true
		}
		return sorted[i].StartTime.After(*sorted[j].StartTime)
	})
	for _, s := range sorted {
		if s.SessionLabel != nil && *s.SessionLabel != "" {
			return *s.SessionLabel
		}
	}
	return ""
}

func earliestStartTime(matches []domain.HomeMatchRow) *time.Time {
	var earliest *time.Time
	for i := range matches {
		t := matches[i].StartTime
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
	}
	return earliest
}

// intPtrIfPos retourne un pointeur vers v si v > 0, nil sinon.
func intPtrIfPos(v int) *int {
	if v > 0 {
		return &v
	}
	return nil
}

// latestEndTime retourne l'heure de fin estimée du dernier match de la session.
func latestEndTime(matches []domain.HomeMatchRow) *time.Time {
	var latest *domain.HomeMatchRow
	for i := range matches {
		if latest == nil || matches[i].StartTime.After(latest.StartTime) {
			latest = &matches[i]
		}
	}
	if latest == nil {
		return nil
	}
	if latest.TimePlayedSecs != nil && *latest.TimePlayedSecs > 0 {
		t := latest.StartTime.Add(time.Duration(*latest.TimePlayedSecs) * time.Second)
		return &t
	}
	t := latest.StartTime
	return &t
}
