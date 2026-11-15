package narrative

// objective_support.go — LES ACTIONS DE SOUTIEN d'un mode à objectif : celles d'un joueur qui
// escorte, retourne et abat les porteurs sans prendre lui-même l'objectif. Elles servent la
// note de performance à objectif de deux façons (cf. ObjectiveActionPointsBoosted et la part
// de soutien de ObjectiveTeamShare) ; les radars et profils continuent de lire
// ObjectiveFamilyActionWeights tels quels.
//
// La table ne porte QUE des noms de colonnes de ObjectiveFamilyActionWeights (garde-rail :
// objective_share_test.go) : aucune colonne neuve, aucun poids de plus.

// ObjectiveSupportColumns : par famille, les colonnes d'action de soutien.
//
//   - Drapeau : passes de capture, retours, porteurs abattus, retourneurs abattus ;
//   - zones (Colline, Bases) : frags offensifs et défensifs de zone ;
//   - Oddball : porteurs du crâne abattus ;
//   - Stockpile : porteurs de graine abattus ;
//   - Extraction : conversions adverses refusées ; VIP : assistances sur le VIP.
//
// Ni les captures, vols, prises de zone, dépôts, conversions réussies (prendre), ni les
// sécurisations (tenir son propre objectif) n'y figurent.
var ObjectiveSupportColumns = map[ObjectiveFamily][]string{
	FamilyCTF: {
		objectiveColFlagCaptureAssists, objectiveColFlagReturns,
		objectiveColFlagCarriersKilled, objectiveColFlagReturnersKilled,
	},
	FamilyZonesKOTH:        {objectiveColZoneOffensiveKills, objectiveColZoneDefensiveKills},
	FamilyZonesStrongholds: {objectiveColZoneOffensiveKills, objectiveColZoneDefensiveKills},
	FamilyOddball:          {objectiveColSkullCarriersKilled},
	FamilyStockpile:        {objectiveColPowerSeedCarriersKilled},
	FamilyExtraction:       {objectiveColExtractionConversionsDenied},
	FamilyVIP:              {objectiveColVipAssists},
}

// objectiveSupportSet rend les colonnes de soutien d'une famille en ensemble.
func objectiveSupportSet(fam ObjectiveFamily) map[string]bool {
	cols := ObjectiveSupportColumns[fam]
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c] = true
	}
	return set
}

// ObjectiveSupportPoints : actions de SOUTIEN pondérées (poids de ObjectiveFamilyActionWeights,
// non relevés) d'un agrégat d'un match.
func ObjectiveSupportPoints(fam ObjectiveFamily, st ObjectiveFamilyStats) float64 {
	if st.Matches <= 0 {
		return 0
	}
	var points float64
	for _, col := range ObjectiveSupportColumns[fam] {
		v := st.ColumnSums[col]
		if v <= 0 {
			continue
		}
		w := ObjectiveFamilyActionWeights[fam][col]
		if col == ObjectiveColKillsAsVIP {
			points += w * v / max(1, st.TimesSelectedAsVIP)
			continue
		}
		points += w * v / float64(st.Matches)
	}
	return points
}
