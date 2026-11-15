// Package service — match_view_radar_objective.go : l'axe « Objectif » du radar de la vue Match,
// la note d'objectif DU MATCH SEUL (narrative.ObjectiveScoreOf, 0-100), sans aucune comparaison à
// l'historique du joueur. La feuille de match porte déjà le bloc objectif de CHAQUE participant (les
// deux camps) : la part du joueur dans l'objectif de son équipe se calcule sur elle, sans requête.
package service

import (
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// matchObjectiveScore rend la note d'objectif de `xuid` pour ce match depuis la feuille de match :
// prise et soutien compris, mêmes poids que la note de performance (frags de soutien à
// narrative.ObjectiveSupportFragWeight). ok == false — l'axe est alors retiré, jamais 0 — quand la
// feuille n'a aucune ligne d'objectif (Slayer, Halo 5) ou que le joueur n'y a ni ligne ni camp.
func matchObjectiveScore(scoreboard []domain.ScoreboardRaw, xuid string) (float64, bool) {
	rows := make([]narrative.ObjectiveTeamRow, 0, len(scoreboard))
	for i := range scoreboard {
		if row, ok := objectiveTeamRowFromScoreboard(scoreboard[i]); ok {
			rows = append(rows, row)
		}
	}
	return narrative.ObjectiveMatchScore(rows, xuid)
}

// objectiveTeamRowFromScoreboard ramène un participant de la feuille de match au vocabulaire du noyau
// (narrative.ObjectiveTeamRow). ok == false : pas de bloc objectif reconnu (mode sans objectif) ou pas
// d'identité.
func objectiveTeamRowFromScoreboard(row domain.ScoreboardRaw) (narrative.ObjectiveTeamRow, bool) {
	if row.XUID == "" || !row.Obj.HasObjective() {
		return narrative.ObjectiveTeamRow{}, false
	}
	fam, ok := objectiveFamilyFromRaw(row.Obj)
	if !ok {
		return narrative.ObjectiveTeamRow{}, false
	}
	out := narrative.ObjectiveTeamRow{
		XUID: row.XUID, TeamID: narrative.NoObjectiveTeam, Family: fam,
		Values: objectiveColumnValues(row.Obj),
	}
	if row.TeamID != nil {
		out.TeamID = *row.TeamID
	}
	if row.TimePlayed != nil {
		out.TimePlayedSeconds = *row.TimePlayed
	}
	return out, true
}

// objectiveFamilyFromRaw classe une ligne objectif (1 match, 1 joueur) dans sa
// famille de mode. Même précédence de blocs et même split KOTH/Strongholds
// (zone_scoring_ticks > 0, décision 3) que objectiveFamilyCaseSQL côté repo.
// false = aucun bloc reconnu (Slayer, titre sans match.objective.stats).
func objectiveFamilyFromRaw(o domain.ObjectiveRaw) (narrative.ObjectiveFamily, bool) {
	switch {
	case o.HasZones():
		if derefInt(o.ZoneScoringTicks) > 0 {
			return narrative.FamilyZonesKOTH, true
		}
		return narrative.FamilyZonesStrongholds, true
	case o.HasCTF():
		return narrative.FamilyCTF, true
	case o.HasOddball():
		return narrative.FamilyOddball, true
	case o.HasStockpile():
		return narrative.FamilyStockpile, true
	case o.HasExtraction():
		return narrative.FamilyExtraction, true
	case o.HasVip():
		return narrative.FamilyVIP, true
	}
	return "", false
}

// objectiveColumnValues porte les valeurs de toutes les colonnes d'action et de durée du bloc
// objectif par nom de colonne de match_objective_stats : c'est la famille de la ligne qui décide
// lesquelles comptent. Les noms DOIVENT couvrir les tables de narrative (garde-rail :
// TestObjectiveColumnValues_CoverFamilyTables) ; une colonne absente vaudrait 0 sans bruit.
func objectiveColumnValues(o domain.ObjectiveRaw) map[string]float64 {
	fi := func(p *int) float64 { return float64(derefInt(p)) }
	return map[string]float64{
		// CTF
		colFlagCaptures: fi(o.FlagCaptures), "flag_capture_assists": fi(o.FlagCaptureAssists),
		"flag_steals": fi(o.FlagSteals), "flag_secures": fi(o.FlagSecures), "flag_returns": fi(o.FlagReturns),
		"flag_carriers_killed": fi(o.FlagCarriersKilled), "flag_returners_killed": fi(o.FlagReturnersKilled),
		"time_as_flag_carrier_seconds": derefFloat64(o.TimeAsFlagCarrierSeconds),
		// Zones
		"zone_captures": fi(o.ZoneCaptures), "zone_secures": fi(o.ZoneSecures),
		"zone_offensive_kills": fi(o.ZoneOffensiveKills), "zone_defensive_kills": fi(o.ZoneDefensiveKills),
		"zone_scoring_ticks": fi(o.ZoneScoringTicks), "time_in_zones_seconds": derefFloat64(o.TimeInZonesSeconds),
		// Oddball
		"skull_grabs": fi(o.SkullGrabs), "skull_carriers_killed": fi(o.SkullCarriersKilled),
		"skull_scoring_ticks":           fi(o.SkullScoringTicks),
		"time_as_skull_carrier_seconds": derefFloat64(o.TimeAsSkullCarrierSeconds),
		// Stockpile
		"power_seeds_deposited": fi(o.PowerSeedsDeposited), "power_seeds_stolen": fi(o.PowerSeedsStolen),
		"power_seed_carriers_killed":         fi(o.PowerSeedCarriersKilled),
		"time_as_power_seed_carrier_seconds": derefFloat64(o.TimeAsPowerSeedCarrierSeconds),
		"time_as_power_seed_driver_seconds":  derefFloat64(o.TimeAsPowerSeedDriverSeconds),
		// Extraction
		"extraction_conversions_completed": fi(o.ExtractionConversionsCompleted),
		"successful_extractions":           fi(o.SuccessfulExtractions),
		"extraction_initiations_completed": fi(o.ExtractionInitiationsCompleted),
		"extraction_conversions_denied":    fi(o.ExtractionConversionsDenied),
		// VIP
		"vip_kills": fi(o.VipKills), "vip_assists": fi(o.VipAssists),
		narrative.ObjectiveColKillsAsVIP:         fi(o.KillsAsVip),
		"time_as_vip_seconds":                    derefFloat64(o.TimeAsVipSeconds),
		narrative.ObjectiveColTimesSelectedAsVIP: fi(o.TimesSelectedAsVip),
	}
}

// colFlagCaptures : nom de la colonne des captures de drapeau de match_objective_stats.
const colFlagCaptures = "flag_captures"
