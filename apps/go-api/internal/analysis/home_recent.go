// Package analysis â€” home_recent.go : projection des matchs rÃ©cents pour la
// timeline home (BuildRecentMatches*) + helpers nullable et URL d'image map.
package analysis

import (
	"fmt"
	"math"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// BuildRecentMatches â€” timeline rÃ©cente
// ---------------------------------------------------------------------------

// BuildRecentMatches construit la liste des derniers matchs pour la timeline.
func BuildRecentMatches(matches []legacymatch.HomeMatchRow, limit int) []domain.RecentMatchItem {
	return BuildRecentMatchesForLocale(matches, limit, "fr")
}

// BuildRecentMatchesForLocale construit la liste des derniers matchs pour la locale demandÃ©e.
func BuildRecentMatchesForLocale(matches []legacymatch.HomeMatchRow, limit int, locale string) []domain.RecentMatchItem {
	return BuildRecentMatchesWithFavoritesForLocale(matches, limit, nil, locale)
}

// BuildRecentMatchesWithFavorites construit la liste des derniers matchs avec le flag favori.
// favoriteIDs est un set de match_id favoris (nil = social repo indisponible).
func BuildRecentMatchesWithFavorites(matches []legacymatch.HomeMatchRow, limit int, favoriteIDs map[string]bool) []domain.RecentMatchItem {
	return BuildRecentMatchesWithFavoritesForLocale(matches, limit, favoriteIDs, "fr")
}

// BuildRecentMatchesWithFavoritesForLocale construit la liste des derniers matchs avec le flag favori
// en choisissant les labels selon la langue active de l'interface.
func BuildRecentMatchesWithFavoritesForLocale(matches []legacymatch.HomeMatchRow, limit int, favoriteIDs map[string]bool, locale string) []domain.RecentMatchItem {
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
		// FDA (KDA canonique fourni par l'API ; pas un calcul custom).
		// Le label dans Detail est "FDA" en FR (cf. fields.toml::kda).
		kdaStr := "-"
		if m.KDA != nil {
			kdaStr = fmt.Sprintf("%.2f", *m.KDA)
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

		// Progression dans le tier : approximation sur une fenÃªtre de 50 pts.
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

		// PrÃ©cision brute (de HomeMatchRow, dÃ©jÃ  en %).
		var accuracy *float64
		if m.Accuracy != nil {
			accuracy = m.Accuracy
		}

		// Solo / Escouade
		isWithFriends := m.IsWithFriends
		iwf := &isWithFriends

		// MapImageURL est résolue par HomeRepo via map_images_registry (pattern
		// asset kinds, lookup par map_id). Empty string → nil pour le frontend.
		mapImageURL := mapImageURLFromRegistry(m.MapImageURL)

		items = append(items, domain.RecentMatchItem{
			MatchID:                  m.MatchID,
			Title:                    fmt.Sprintf("%s · %s", label, mapUI),
			Detail:                   fmt.Sprintf("%s · FDA %s · %s", modeUI, kdaStr, accStr),
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

// mapImageURLFromRegistry retourne *string si la home_repo a résolu une URL
// depuis map_images_registry, nil sinon. Pas de fallback name-based : un
// map_id absent du registry signale un asset à indexer via cmd/migrate-static-maps,
// pas une URL à fabriquer côté analyse (le name peut être un UUID brut ou un
// label localisé qui ne correspond à aucun fichier sur disque).
func mapImageURLFromRegistry(localPath string) *string {
	if strings.TrimSpace(localPath) == "" {
		return nil
	}
	return &localPath
}

// mmrDelta calcule team_mmr - enemy_mmr ; retourne nil si l'un ou l'autre est absent.
func mmrDelta(team, enemy *float64) *float64 {
	if team == nil || enemy == nil {
		return nil
	}
	v := *team - *enemy
	return &v
}

// float64PtrVal retourne la valeur pointÃ©e ou 0 si nil.
func float64PtrVal(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// intPtrIfPos retourne un pointeur vers v si v > 0, nil sinon.
func intPtrIfPos(v int) *int {
	if v > 0 {
		return &v
	}
	return nil
}
