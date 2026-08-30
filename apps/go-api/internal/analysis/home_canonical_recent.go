// Package analysis â€” home_canonical_recent.go : BuildRecentMatchesWithFavorites
// canonical (P4.3 finale). Construit la liste des matchs rÃ©cents + score label.
package analysis

import (
	"fmt"
	"math"
	"strings"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// BuildRecentMatchesWithFavoritesFromCanonical : full canonical (P4.3 finale).
//
// Lit Map/Playlist/GameVariant labels via Summary.AssetReference.Labels.
// PairName (composite Halo-only) substituÃ© par GameVariant FR/Default.
// Ce que la couche service injecte — locale, létalité du titre, résolveur de badge de
// palier, table des variantes qui se décident aux manches — voyage dans
// RecentMatchesOptions, ci-dessous.

// RecentMatchesOptions regroupe ce que la couche service injecte dans la projection des
// tuiles d'accueil : la locale, la constante de létalité du titre, le résolveur de badge de
// palier, et la table des variantes qui se décident aux manches.
//
// Un struct plutôt que quatre paramètres de plus : la fonction en portait déjà six, au-delà
// du seuil du dépôt (5), et chaque ajout aggravait la dette. Groupés, ils la ramènent SOUS
// le seuil.
type RecentMatchesOptions struct {
	Locale            string
	EffectiveHpToKill float64
	// SkillBadgeURL : (tier EN capitalisé, sous-palier 0..6 ; 0 = Onyx) -> URL. Nil → aucune
	// image, le front retombe sur le libellé texte.
	SkillBadgeURL func(tierEN string, subTier int) string
	// RoundsDecide : game_variant_name -> le RÉSULTAT se lit en manches (ADR 0032).
	// Nil/absente → lecture en points, comme avant.
	RoundsDecide map[string]bool
}

func BuildRecentMatchesWithFavoritesFromCanonical(
	rows []canonical.PlayerMatchRow,
	limit int,
	favoriteIDs map[string]bool,
	opts RecentMatchesOptions,
) []domain.RecentMatchItem {
	if len(rows) == 0 {
		return nil
	}
	locale := normalizeHomeLocale(opts.Locale)
	effectiveHpToKill := opts.EffectiveHpToKill
	skillBadgeURL := opts.SkillBadgeURL
	if len(rows) > limit {
		rows = rows[:limit]
	}
	items := make([]domain.RecentMatchItem, 0, len(rows))
	for _, r := range rows {
		if r.Summary.MatchID == "" {
			continue
		}
		// Outcome canonical â†’ int Halo pour les helpers existants.
		outcome := canonicalOutcomeToInt(r.Self.Outcome)
		label := outcomeLabelForLocale(outcome, locale)
		tone := outcomeTone(outcome)

		// FDA (KDA canonique fourni par l'API ; pas un calcul custom).
		// Le label dans Detail est "FDA" en FR (cf. fields.toml::kda).
		kdaStr := "-"
		if r.Self.KDA != nil {
			kdaStr = fmt.Sprintf("%.2f", *r.Self.KDA)
		}
		t := r.Summary.StartedAtUTC

		mapName, mapNameFR := assetLabels(r.Summary.Map)
		// modeLabels privilégie PairMode (FR/EN dispo en DB) puis fallback
		// GameVariant (FR jamais peuplé). Bug #7.
		modeEN, modeFR := modeLabels(r)
		playlistName, playlistNameFR := assetLabels(r.Summary.Playlist)

		mapUI := labelForLocale(locale, mapNameFR, mapName)
		modeUI := normalizeHomeModeLabel(labelForLocale(locale, modeFR, modeEN), mapNameFR, mapName)
		var playlistUI *string
		if playlistName != "" || playlistNameFR != "" {
			playlist := labelForLocale(locale, playlistNameFR, playlistName)
			playlistUI = &playlist
		}

		// Score label : reconstruit depuis Summary.Teams — points ou MANCHES.
		scoreLabel := buildScoreLabelCanonical(r, opts.RoundsDecide)
		scoreStr := "-"
		if scoreLabel != nil {
			scoreStr = *scoreLabel
		}
		narrativeBadges := buildHomeNarrativeBadges(int(r.Enrichment.DominanceFlag))

		kills := derefIntZero(r.Self.Kills)
		deaths := derefIntZero(r.Self.Deaths)
		assists := derefIntZero(r.Self.Assists)

		var perfScoreRel *int
		if r.Enrichment.PerformanceScore != nil {
			v := int(math.Round(*r.Enrichment.PerformanceScore))
			perfScoreRel = &v
		}

		// SkillSnapshot : data fields + label + badge URL.
		var skillRatingVal *int
		var skillRatingType *string
		var skillRatingDelta *float64
		var skillPlaylistGroup *string
		var skillProgressPct *float64
		var skillPointsInTier *int
		var skillTierLabelVal *string
		var skillBadgeURLVal *string
		if ss := r.Enrichment.SkillSnapshot; ss != nil {
			if ss.RatingValue != nil {
				v := int(math.Round(*ss.RatingValue))
				skillRatingVal = &v
				typ := string(ss.RatingType)
				skillRatingType = &typ
				if typ == "CSR" {
					const tierSize = 50.0
					pts := math.Mod(*ss.RatingValue, tierSize)
					if pts < 0 {
						pts += tierSize
					}
					pct := pts / tierSize * 100.0
					skillProgressPct = &pct
					pInt := int(math.Round(pts))
					skillPointsInTier = &pInt
				} else if p, ok := skillv2.LegacyContinuousSubTierProgress(*ss.RatingValue); ok {
					// LUSR : rating_value continu → position dans le sous-palier (grille
					// legacy à largeurs variables ; l'ancien mod-50 était faux pour LUSR).
					pct := p * 100.0
					skillProgressPct = &pct
				}
			}
			skillRatingDelta = ss.Delta
			if ss.PlaylistGroup != nil && *ss.PlaylistGroup != "" {
				skillPlaylistGroup = ss.PlaylistGroup
			}
			// Badge URL depuis TierCode (EN) ; label depuis TierCodeFR si locale == "fr".
			if ss.TierCode != nil && *ss.TierCode != "" {
				tierForLabel := *ss.TierCode
				if locale == "fr" && ss.TierCodeFR != nil && *ss.TierCodeFR != "" {
					tierForLabel = *ss.TierCodeFR
				}
				skillTierLabelVal, skillBadgeURLVal = buildCanonicalSkillBadge(tierForLabel, *ss.TierCode, ss.SubTier, skillBadgeURL)
			}
		}

		// Combat yield depuis canonical (DamageDealt/DamageTaken int â†’ float64).
		var offConv, defRes *float64
		var dmgDealtPtr, dmgTakenPtr *float64
		if r.Self.DamageDealt != nil {
			v := float64(*r.Self.DamageDealt)
			dmgDealtPtr = &v
		}
		if r.Self.DamageTaken != nil {
			v := float64(*r.Self.DamageTaken)
			dmgTakenPtr = &v
		}
		dd := float64PtrVal(dmgDealtPtr)
		dt := float64PtrVal(dmgTakenPtr)
		if dd > 0 || dt > 0 {
			cy := ComputeCombatYield(kills, assists, dd, dt, deaths, effectiveHpToKill)
			if dd > 0 {
				offConv = &cy.OffensiveConversion
			}
			if dt > 0 && deaths > 0 {
				defRes = &cy.DefensiveResistance
			}
		}

		isFav := favoriteIDs[r.Summary.MatchID]

		isWithFriends := r.Enrichment.IsWithFriends
		iwf := &isWithFriends

		// MapImageURL est résolu par HomeRepo.EnrichCanonicalAssetTranslations
		// via map_images_registry (pattern asset kinds, lookup par map_id).
		// Hydraté sur Map.IconURL au moment de l'enrichissement ; ici on fait
		// juste un passe-plat. Empty string → nil pour omitempty côté JSON.
		var mapImageURL *string
		if m := r.Summary.Map; m != nil && strings.TrimSpace(m.IconURL) != "" {
			u := m.IconURL
			mapImageURL = &u
		}

		items = append(items, domain.RecentMatchItem{
			MatchID:                  r.Summary.MatchID,
			Title:                    fmt.Sprintf("%s · %s", label, mapUI),
			Detail:                   fmt.Sprintf("%s · FDA %s · %s", modeUI, kdaStr, scoreStr),
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
			DamageDealt:              dmgDealtPtr,
			DamageTaken:              dmgTakenPtr,
			SkillRatingValue:         skillRatingVal,
			SkillRatingType:          skillRatingType,
			SkillTierLabel:           skillTierLabelVal,
			SkillRatingDelta:         skillRatingDelta,
			SkillPlaylistGroup:       skillPlaylistGroup,
			SkillRankImageURL:        skillBadgeURLVal,
			SkillProgressPct:         skillProgressPct,
			SkillPointsInTier:        skillPointsInTier,
			KDA:                      r.Self.KDA,
			DurationSecs:             r.Self.TimePlayed,
			Accuracy:                 r.Self.Accuracy,
			AvgLifeSecs:              r.Self.AvgLifeSeconds,
			TeamMMR:                  r.Enrichment.TeamMMR,
			EnemyMMR:                 r.Enrichment.EnemyMMR,
			DeltaMMR:                 mmrDelta(r.Enrichment.TeamMMR, r.Enrichment.EnemyMMR),
			IsWithFriends:            iwf,
			RankInTeam:               r.Self.RankInMatch,
			HeadshotKills:            intPtrIfPos(derefIntZero(r.Self.HeadshotKills)),
			PerfectKills:             intPtrIfPos(derefIntZero(r.Self.PerfectKills)),
			SessionLabel:             r.Enrichment.SessionLabel,
			IsRanked:                 r.Summary.IsRanked != nil && *r.Summary.IsRanked,
		})
	}
	return items
}

// buildScoreLabelCanonical : reconstruit le score "X-Y" depuis Summary.Teams
// + Self.TeamID (Ã©quivalent canonical de buildHomeScoreLabel).
func buildScoreLabelCanonical(r canonical.PlayerMatchRow, roundsDecide map[string]bool) *string {
	var t0, t1 *canonical.TeamSnapshot
	for i := range r.Summary.Teams {
		switch r.Summary.Teams[i].TeamID {
		case 0:
			t0 = &r.Summary.Teams[i]
		case 1:
			t1 = &r.Summary.Teams[i]
		}
	}
	if t0 == nil || t1 == nil {
		return nil
	}
	mine, theirs := t0, t1
	if r.Self.TeamID != nil && *r.Self.TeamID == 1 {
		mine, theirs = t1, t0
	}
	// Délégation à la source unique (cf. team_score_display.go) : la MÊME règle que la vue
	// match, l'historique et l'escouade — une tuile d'accueil et la page du match ne peuvent
	// pas afficher deux nombres différents pour le même Oddball.
	label := TeamScoreLabel(TeamScoreInput{
		MyPoints: mine.Score, EnemyPoints: theirs.Score,
		MyRoundsWon: mine.RoundsWon, EnemyRoundsWon: theirs.RoundsWon,
		RoundsTotal:  r.Summary.RoundsTotal,
		RoundsDecide: roundsDecide[strings.TrimSpace(variantNameOf(r))],
	})
	if label == "" {
		return nil
	}
	return &label
}

// variantNameOf rend le nom de variante canonique du match, ou "" — la clé de la table
// [rounds_decide]. Une variante inconnue n'est déclarée nulle part, donc points.
func variantNameOf(r canonical.PlayerMatchRow) string {
	if r.Summary.GameVariant == nil {
		return ""
	}
	return r.Summary.GameVariant.DefaultLabel
}
