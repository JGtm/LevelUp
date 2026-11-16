package narrative

// objective_raw.go — DU BLOC OBJECTIF D'UN PARTICIPANT AU VOCABULAIRE DU NOYAU : la famille de mode d'une
// ligne `domain.ObjectiveRaw` et ses colonnes de `match_objective_stats`. Pur, comme le reste du paquet.
//
// La famille se décide AUSSI en SQL (`objectiveFamilyCaseSQL`, platform/duckdb) pour les lectures qui ne
// passent pas par la feuille de match : les deux expressions de la même règle sont tenues égales par le
// test de parité de platform/duckdb (TestObjectiveFamilyCaseSQL_ParityWithGoFunction), un cas par
// famille et un cas sans objectif.

import "levelup/go-api/internal/domain"

// ObjectiveFamilyOfRaw classe une ligne objectif (1 match, 1 joueur) dans sa famille de mode. Même
// précédence de blocs et même split KOTH / Strongholds (zone_scoring_ticks > 0, décision 3) que
// l'expression SQL `objectiveFamilyCaseSQL`. false = aucun bloc reconnu (Slayer, titre sans
// match.objective.stats).
func ObjectiveFamilyOfRaw(o domain.ObjectiveRaw) (ObjectiveFamily, bool) {
	switch {
	case o.HasZones():
		if rawInt(o.ZoneScoringTicks) > 0 {
			return FamilyZonesKOTH, true
		}
		return FamilyZonesStrongholds, true
	case o.HasCTF():
		return FamilyCTF, true
	case o.HasOddball():
		return FamilyOddball, true
	case o.HasStockpile():
		return FamilyStockpile, true
	case o.HasExtraction():
		return FamilyExtraction, true
	case o.HasVip():
		return FamilyVIP, true
	}
	return "", false
}

// ObjectiveColumnValuesOf porte les valeurs de toutes les colonnes d'action et de durée du bloc objectif
// par nom de colonne de match_objective_stats : c'est la famille de la ligne qui décide lesquelles
// comptent. Les noms DOIVENT couvrir les tables de ce paquet (garde-rail :
// TestObjectiveColumnValues_CoverFamilyTables) ; une colonne absente vaudrait 0 sans bruit. Un champ nil
// vaut 0.
func ObjectiveColumnValuesOf(o domain.ObjectiveRaw) map[string]float64 {
	return map[string]float64{
		// CTF
		objectiveColFlagCaptures: rawInt(o.FlagCaptures), objectiveColFlagCaptureAssists: rawInt(o.FlagCaptureAssists),
		objectiveColFlagSteals: rawInt(o.FlagSteals), objectiveColFlagSecures: rawInt(o.FlagSecures),
		objectiveColFlagReturns:        rawInt(o.FlagReturns),
		objectiveColFlagCarriersKilled: rawInt(o.FlagCarriersKilled), objectiveColFlagReturnersKilled: rawInt(o.FlagReturnersKilled),
		objectiveColTimeAsFlagCarrier: rawFloat(o.TimeAsFlagCarrierSeconds),
		// Zones
		objectiveColZoneCaptures: rawInt(o.ZoneCaptures), objectiveColZoneSecures: rawInt(o.ZoneSecures),
		objectiveColZoneOffensiveKills: rawInt(o.ZoneOffensiveKills), objectiveColZoneDefensiveKills: rawInt(o.ZoneDefensiveKills),
		objectiveColZoneScoringTicks: rawInt(o.ZoneScoringTicks), objectiveColTimeInZones: rawFloat(o.TimeInZonesSeconds),
		// Oddball
		objectiveColSkullGrabs: rawInt(o.SkullGrabs), objectiveColSkullCarriersKilled: rawInt(o.SkullCarriersKilled),
		objectiveColSkullScoringTicks:  rawInt(o.SkullScoringTicks),
		objectiveColTimeAsSkullCarrier: rawFloat(o.TimeAsSkullCarrierSeconds),
		// Stockpile
		objectiveColPowerSeedsDeposited: rawInt(o.PowerSeedsDeposited), objectiveColPowerSeedsStolen: rawInt(o.PowerSeedsStolen),
		objectiveColPowerSeedCarriersKilled: rawInt(o.PowerSeedCarriersKilled),
		objectiveColTimeAsPowerSeedCarrier:  rawFloat(o.TimeAsPowerSeedCarrierSeconds),
		objectiveColTimeAsPowerSeedDriver:   rawFloat(o.TimeAsPowerSeedDriverSeconds),
		// Extraction
		objectiveColExtractionConversionsCompleted: rawInt(o.ExtractionConversionsCompleted),
		objectiveColSuccessfulExtractions:          rawInt(o.SuccessfulExtractions),
		objectiveColExtractionInitiationsCompleted: rawInt(o.ExtractionInitiationsCompleted),
		objectiveColExtractionConversionsDenied:    rawInt(o.ExtractionConversionsDenied),
		// VIP
		objectiveColVipKills: rawInt(o.VipKills), objectiveColVipAssists: rawInt(o.VipAssists),
		ObjectiveColKillsAsVIP:         rawInt(o.KillsAsVip),
		objectiveColTimeAsVip:          rawFloat(o.TimeAsVipSeconds),
		ObjectiveColTimesSelectedAsVIP: rawInt(o.TimesSelectedAsVip),
	}
}

func rawInt(p *int) float64 {
	if p == nil {
		return 0
	}
	return float64(*p)
}

func rawFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
