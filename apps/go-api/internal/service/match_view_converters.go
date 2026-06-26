// Package service — converters analysis -> domain pour MatchView.
//
// Helpers de conversion entre les types de la couche analysis/ (stateless,
// chart-agnostic) et les types domain.* exposés par l'API Match View.
// Extrait de match_view_service.go (audit #1 god files).
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// convertTugBinsToDomain convertit les bins analysis -> domain pour le chart
// tug-of-war (allies vs ennemis), avec split delta positif (allies) / negatif
// (ennemis).
func convertTugBinsToDomain(bins []analysis.TugOfWarBin) []domain.MatchTugOfWarBin {
	out := make([]domain.MatchTugOfWarBin, 0, len(bins))
	for _, b := range bins {
		allyKills := 0
		enemyKills := 0
		if b.Delta > 0 {
			allyKills = b.Delta
		} else {
			enemyKills = -b.Delta
		}
		out = append(out, domain.MatchTugOfWarBin{
			BinStart:   int(b.BinStartMS / 1000),
			BinEnd:     int(b.BinEndMS / 1000),
			TeamKills:  allyKills,
			EnemyKills: enemyKills,
			NetKills:   b.CumDelta,
		})
	}
	return out
}

// convertImpactBadgesToDomain convertit les badges analysis -> domain avec
// pointer optionnel sur TimeMS (nil si event horodaté absent).
func convertImpactBadgesToDomain(badges []analysis.ImpactBadge) []domain.MatchImpactBadge {
	out := make([]domain.MatchImpactBadge, 0, len(badges))
	for _, b := range badges {
		badge := domain.MatchImpactBadge{
			Key:        b.BadgeKey,
			Label:      b.BadgeFR,
			PlayerXUID: b.PlayerXUID,
		}
		if b.TimeMS > 0 {
			t := b.TimeMS
			badge.TimeMS = &t
		}
		out = append(out, badge)
	}
	return out
}

// convertKDPointsToDomain convertit les points KD timeline analysis -> domain
// (cum kills / cum deaths, secondes plutot que ms).
func convertKDPointsToDomain(points []analysis.KDTimelinePoint) []domain.MatchKDTimelinePoint {
	out := make([]domain.MatchKDTimelinePoint, 0, len(points))
	for _, p := range points {
		out = append(out, domain.MatchKDTimelinePoint{
			TimeSeconds: int(p.TimeMS / 1000),
			Kills:       p.CumKills,
			Deaths:      p.CumDeaths,
		})
	}
	return out
}

// computeScoreboardRowCombatYield calcule les 4 pointeurs combat yield du
// scoreboard row (offensive_conversion, defensive_resistance, damage_per_kill,
// damage_per_death). Retourne nil pour les pointeurs dont les données source
// sont absentes — semantically "non calculable".
//
// OC (offensive_conversion) ne dépend QUE de damage_dealt : on l'émet dès que
// DamageDealt est présent (title-agnostic — Halo 5 porte damage_dealt mais pas
// damage_taken). DR (defensive_resistance) exige damage_taken : reste N/A pour
// les titres sans cette donnée (légitime). ComputeCombatYield est déjà découplé
// (OC si damageDlt>0, DR si damageTkn>0 && deaths>0) → on passe damageTkn=0
// quand DamageTaken est nil pour ne calculer que l'OC.
func computeScoreboardRowCombatYield(s domain.ScoreboardRaw, effectiveHpToKill float64) (oc, dr, dpk, dpd *float64) {
	if s.DamageDealt != nil {
		damageTkn := 0.0
		if s.DamageTaken != nil {
			damageTkn = *s.DamageTaken
		}
		cy := analysis.ComputeCombatYield(s.Kills, s.Assists, *s.DamageDealt, damageTkn, s.Deaths, effectiveHpToKill)
		oc = &cy.OffensiveConversion
		if s.DamageTaken != nil {
			if s.Deaths == 0 {
				inf := -1.0 // sentinel ∞ — jamais de valeur négative réelle
				dr = &inf
			} else {
				dr = &cy.DefensiveResistance
			}
		}
	}
	if s.DamageDealt != nil && s.Kills > 0 {
		v := *s.DamageDealt / float64(s.Kills)
		dpk = &v
	}
	if s.DamageTaken != nil && s.Deaths > 0 {
		v := *s.DamageTaken / float64(s.Deaths)
		dpd = &v
	}
	return oc, dr, dpk, dpd
}

// indexBulkMedalsByXUID indexe les médailles bulk par XUID pour O(1) lookup.
// ImageURL résolu via TitleAssetURLAdapter (medals = ID numérique).
func indexBulkMedalsByXUID(
	bulkMedals []domain.BulkMedalRaw,
	assetURL games.TitleAssetURLAdapter,
	hint int,
) map[string][]domain.PlayerMedalRow {
	out := make(map[string][]domain.PlayerMedalRow, hint)
	for _, m := range bulkMedals {
		var imgURL string
		if assetURL != nil {
			imgURL = assetURL.MedalImageURL(uint64(m.MedalID)) //nolint:gosec
		}
		out[m.XUID] = append(out[m.XUID], domain.PlayerMedalRow{
			MedalID:    m.MedalID,
			Count:      m.Count,
			Label:      m.Label,
			ImageURL:   imgURL,
			Difficulty: m.Difficulty,
		})
	}
	return out
}

// indexBulkWeaponsByXUID indexe les armes bulk par XUID pour O(1) lookup.
// ImageURL résolu via TitleAssetURLAdapter (weapons = name_en → fichier slug).
func indexBulkWeaponsByXUID(
	bulkWeapons []domain.BulkWeaponKillRaw,
	assetURL games.TitleAssetURLAdapter,
	hint int,
) map[string][]domain.PlayerWeaponKillRow {
	out := make(map[string][]domain.PlayerWeaponKillRow, hint)
	for _, w := range bulkWeapons {
		var imgURL string
		if assetURL != nil {
			imgURL = assetURL.WeaponImageURL(w.NameEN)
		}
		out[w.XUID] = append(out[w.XUID], domain.PlayerWeaponKillRow{
			WeaponID: w.WeaponID,
			Kills:    w.Kills,
			Label:    w.WeaponLabel,
			ImageURL: imgURL,
		})
	}
	return out
}
