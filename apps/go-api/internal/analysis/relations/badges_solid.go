package relations

import "time"

// Tokens couleur des nouveaux badges (style solid). Réutilisent la famille
// narrative-encounter-* (déjà déclarée front + palettes) pour la teinte ; le
// rendu plein vs teinté est piloté par le champ Style côté front.
const (
	colorTokenDuoGagnant    = "narrative-encounter-duo-gagnant"
	colorTokenCameleon      = "narrative-encounter-cameleon"
	colorTokenDeLongueDate  = "narrative-encounter-de-longue-date"
	colorTokenRecrue        = "narrative-encounter-recrue"
	colorTokenProieFavorite = "narrative-encounter-proie-favorite"
)

// solidBadges calcule les nouveaux badges (style solid) dans un ordre stable :
// duo_gagnant → cameleon → de_longue_date → recrue → proie_favorite.
func solidBadges(s RelationStats, now time.Time) []Badge {
	out := make([]Badge, 0, 5)
	if b := duoGagnantBadge(s); b != nil {
		out = append(out, *b)
	}
	if b := cameleonBadge(s); b != nil {
		out = append(out, *b)
	}
	if b := deLongueDateBadge(s, now); b != nil {
		out = append(out, *b)
	}
	if b := recrueBadge(s, now); b != nil {
		out = append(out, *b)
	}
	if b := proieFavoriteBadge(s); b != nil {
		out = append(out, *b)
	}
	return out
}

// duoGagnantBadge : teammate_win_rate >= 0.60 ET teammate_matches >= 10.
func duoGagnantBadge(s RelationStats) *Badge {
	if s.TeammateWinRate == nil || s.TeammateMatches < DuoGagnantMinTeammateMatches {
		return nil
	}
	if *s.TeammateWinRate < DuoGagnantWinRateThreshold {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.duo_gagnant",
		ColorToken: colorTokenDuoGagnant,
		Style:      BadgeStyleSolid,
		Detail:     map[string]any{"teammate_win_rate": *s.TeammateWinRate},
	}
}

// cameleonBadge : min(teammate,enemy)/total >= 0.40 ET total >= 10.
func cameleonBadge(s RelationStats) *Badge {
	if s.TotalMatches < CameleonMinTotalMatches || s.TotalMatches == 0 {
		return nil
	}
	minSide := s.TeammateMatches
	if s.EnemyMatches < minSide {
		minSide = s.EnemyMatches
	}
	ratio := float64(minSide) / float64(s.TotalMatches)
	if ratio < CameleonMixRatioThreshold {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.cameleon",
		ColorToken: colorTokenCameleon,
		Style:      BadgeStyleSolid,
		Detail:     map[string]any{"mix_ratio": ratio},
	}
}

// deLongueDateBadge : first_seen > 6 mois OU total_matches >= 80.
func deLongueDateBadge(s RelationStats, now time.Time) *Badge {
	oldEnough := false
	if s.FirstSeen != nil && !s.FirstSeen.IsZero() {
		ageDays := now.Sub(*s.FirstSeen).Hours() / 24
		if ageDays > DeLongueDateMinMonths*monthsToDaysApprox {
			oldEnough = true
		}
	}
	if !oldEnough && s.TotalMatches < DeLongueDateMinTotalMatches {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.de_longue_date",
		ColorToken: colorTokenDeLongueDate,
		Style:      BadgeStyleSolid,
		Detail:     map[string]any{"total_matches": s.TotalMatches},
	}
}

// recrueBadge : first_seen < 30 jours ET total_matches >= 4.
func recrueBadge(s RelationStats, now time.Time) *Badge {
	if s.FirstSeen == nil || s.FirstSeen.IsZero() {
		return nil
	}
	if s.TotalMatches < RecrueMinTotalMatches {
		return nil
	}
	ageDays := now.Sub(*s.FirstSeen).Hours() / 24
	if ageDays >= RecrueMaxDays {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.recrue",
		ColorToken: colorTokenRecrue,
		Style:      BadgeStyleSolid,
		Detail:     map[string]any{"first_seen_days": ageDays},
	}
}

// proieFavoriteBadge : duel_ratio > 1.5 ET enemy_matches >= 6.
func proieFavoriteBadge(s RelationStats) *Badge {
	if s.DuelRatio == nil || s.EnemyMatches < ProieFavoriteMinEnemyMatches {
		return nil
	}
	if *s.DuelRatio <= ProieFavoriteDuelRatioThreshold {
		return nil
	}
	return &Badge{
		LabelKey:   "narrative.encounter.proie_favorite",
		ColorToken: colorTokenProieFavorite,
		Style:      BadgeStyleSolid,
		Detail:     map[string]any{"duel_ratio": *s.DuelRatio},
	}
}
