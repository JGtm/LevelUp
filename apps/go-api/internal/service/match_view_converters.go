// Package service — converters analysis -> domain pour MatchView.
//
// Helpers de conversion entre les types de la couche analysis/ (stateless,
// chart-agnostic) et les types domain.* exposés par l'API Match View.
// Extrait de match_view_service.go (audit #1 god files).
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
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

// buildScoreboardObjective mappe ObjectiveRaw (Q12 LEFT JOIN) vers le DTO
// MatchScoreboardObjective. Retourne nil si aucun bloc objectif n'est présent
// (Slayer / titre non supporté → table vide) — le front n'affiche alors pas de
// section. Ne recopie QUE les champs du bloc du mode joué (les autres restent nil,
// omitempty côté JSON) — data-driven par mode.
func buildScoreboardObjective(o domain.ObjectiveRaw) *domain.MatchScoreboardObjective {
	if !o.HasObjective() {
		return nil
	}
	out := &domain.MatchScoreboardObjective{}
	if o.HasCTF() {
		out.FlagCaptures = o.FlagCaptures
		out.FlagCaptureAssists = o.FlagCaptureAssists
		out.FlagGrabs = o.FlagGrabs
		out.FlagSecures = o.FlagSecures
		out.FlagSteals = o.FlagSteals
		out.FlagReturns = o.FlagReturns
		out.FlagCarriersKilled = o.FlagCarriersKilled
		out.FlagReturnersKilled = o.FlagReturnersKilled
		out.KillsAsFlagCarrier = o.KillsAsFlagCarrier
		out.KillsAsFlagReturner = o.KillsAsFlagReturner
		out.TimeAsFlagCarrierSeconds = o.TimeAsFlagCarrierSeconds
	}
	if o.HasZones() {
		out.ZoneCaptures = o.ZoneCaptures
		out.ZoneSecures = o.ZoneSecures
		out.ZoneOffensiveKills = o.ZoneOffensiveKills
		out.ZoneDefensiveKills = o.ZoneDefensiveKills
		out.ZoneScoringTicks = o.ZoneScoringTicks
		out.TimeInZonesSeconds = o.TimeInZonesSeconds
	}
	if o.HasOddball() {
		out.KillsAsSkullCarrier = o.KillsAsSkullCarrier
		out.SkullCarriersKilled = o.SkullCarriersKilled
		out.SkullGrabs = o.SkullGrabs
		out.SkullScoringTicks = o.SkullScoringTicks
		out.TimeAsSkullCarrierSeconds = o.TimeAsSkullCarrierSeconds
		out.LongestTimeAsSkullCarrierSeconds = o.LongestTimeAsSkullCarrierSeconds
	}
	if o.HasStockpile() {
		out.KillsAsPowerSeedCarrier = o.KillsAsPowerSeedCarrier
		out.PowerSeedCarriersKilled = o.PowerSeedCarriersKilled
		out.PowerSeedsDeposited = o.PowerSeedsDeposited
		out.PowerSeedsStolen = o.PowerSeedsStolen
		out.TimeAsPowerSeedCarrierSeconds = o.TimeAsPowerSeedCarrierSeconds
		out.TimeAsPowerSeedDriverSeconds = o.TimeAsPowerSeedDriverSeconds
	}
	if o.HasExtraction() {
		out.ExtractionConversionsCompleted = o.ExtractionConversionsCompleted
		out.ExtractionConversionsDenied = o.ExtractionConversionsDenied
		out.ExtractionInitiationsCompleted = o.ExtractionInitiationsCompleted
		out.ExtractionInitiationsDenied = o.ExtractionInitiationsDenied
		out.SuccessfulExtractions = o.SuccessfulExtractions
	}
	if o.HasVip() {
		out.KillsAsVip = o.KillsAsVip
		out.VipKills = o.VipKills
		out.VipAssists = o.VipAssists
		out.TimesSelectedAsVip = o.TimesSelectedAsVip
		out.MaxKillingSpreeAsVip = o.MaxKillingSpreeAsVip
		out.TimeAsVipSeconds = o.TimeAsVipSeconds
		out.LongestTimeAsVipSeconds = o.LongestTimeAsVipSeconds
	}
	// ASSAUT : le seul bloc qui ne vient pas de `match_objective_stats_latest` mais du FILM
	// (l'API 343 n'en publie aucune pour ce mode). Il est chargé par une seconde requête,
	// gatée par la capability `film.bomb_stats` : absente, aucun de ces champs n'est renseigné
	// et `HasBomb()` est faux — pas de bloc, pas de section côté web.
	if o.HasBomb() {
		out.BombDetonations = o.BombDetonations
		out.BombArms = o.BombArms
		out.BombGrabs = o.BombGrabs
		out.TimeAsBombCarrierSeconds = o.TimeAsBombCarrierSeconds
		out.BombCarriersKilled = o.BombCarriersKilled
	}
	return out
}

// indexBulkMedalsByXUID indexe les médailles bulk par XUID pour O(1) lookup.
// Icône title-aware via static.MedalImage (comme convertMedals du résumé) : PNG pour
// Halo Infinite, sprite (feuille + offset) pour Halo 5. L'ancien chemin
// assetURL.MedalImageURL renvoyait "" pour H5 (pas de PNG par-médaille) → médailles
// vides dans le drawer scoreboard (GH-5a). titleSlug tiré de l'adapter.
func indexBulkMedalsByXUID(
	bulkMedals []domain.BulkMedalRaw,
	assetURL games.TitleAssetURLAdapter,
	hint int,
) map[string][]domain.PlayerMedalRow {
	out := make(map[string][]domain.PlayerMedalRow, hint)
	titleSlug := ""
	if assetURL != nil {
		titleSlug = assetURL.TitleSlug()
	}
	for _, m := range bulkMedals {
		row := domain.PlayerMedalRow{
			MedalID:    m.MedalID,
			Count:      m.Count,
			Label:      m.Label,
			Difficulty: m.Difficulty,
		}
		if titleSlug != "" {
			png, sp := static.MedalImage(titleSlug, m.MedalID)
			if sp != nil {
				row.SpriteSheet, row.SpriteLeft, row.SpriteTop, row.SpriteWidth, row.SpriteHeight =
					sp.SheetURL, sp.Left, sp.Top, sp.Width, sp.Height
			} else {
				row.ImageURL = png
			}
		}
		out[m.XUID] = append(out[m.XUID], row)
	}
	return out
}

// indexBulkWeaponsByXUID indexe les armes bulk par XUID pour O(1) lookup.
// ImageURL résolu via TitleAssetURLAdapter, keyé par weapon_id (l'identifiant
// natif du titre, pas le libellé — cf. games.TitleAssetURLAdapter).
func indexBulkWeaponsByXUID(
	bulkWeapons []domain.BulkWeaponKillRaw,
	assetURL games.TitleAssetURLAdapter,
	hint int,
) map[string][]domain.PlayerWeaponKillRow {
	out := make(map[string][]domain.PlayerWeaponKillRow, hint)
	for _, w := range bulkWeapons {
		var imgURL string
		var tinted bool
		if assetURL != nil {
			imgURL = assetURL.WeaponImageURL(w.WeaponID)
			tinted = imgURL != "" && assetURL.WeaponImageIsTinted(w.WeaponID)
		}
		out[w.XUID] = append(out[w.XUID], domain.PlayerWeaponKillRow{
			WeaponID:    w.WeaponID,
			Kills:       w.Kills,
			Label:       w.WeaponLabel,
			ImageURL:    imgURL,
			ImageTinted: tinted,
		})
	}
	return out
}
