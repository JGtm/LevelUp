package narrative

// objective_support.go — LES ACTIONS DE SOUTIEN d'un mode à objectif : celles d'un joueur qui
// escorte, retourne, sécurise et abat les porteurs sans prendre lui-même l'objectif. Elles
// donnent une mesure d'objectif PROPRE au soutien, indépendante du temps de port (le terme de
// tenue de l'indice r_f est nul pour qui ne porte pas) : ObjectiveSupportPoints, dont la part
// d'équipe (ObjectiveTeamShare.SupportShare) règle le poids du soutien dans la note de
// performance à objectif. Les poids restent ceux de ObjectiveFamilyActionWeights.
//
// Deux sortes d'actions de soutien :
//   - SANS frag (retours, sécurisations, passes de capture, conversions refusées...) : poids
//     plein, elles ne comptent nulle part ailleurs ;
//   - qui sont des FRAGS (porteurs abattus, frags de zone...) : poids réduit de
//     ObjectiveSupportFragWeight, parce que le combat les compte déjà.
//
// La table ne porte QUE des noms de colonnes de ObjectiveFamilyActionWeights (garde-rail :
// objective_share_test.go) : aucune colonne neuve, aucun poids de plus.

// ObjectiveSupportFragWeight : le facteur des actions de soutien qui sont des frags.
const ObjectiveSupportFragWeight = 0.5

// ObjectiveSupportColumns : par famille, les colonnes d'action de soutien (frags compris).
//
//   - Drapeau : retours, sécurisations, passes de capture, porteurs et retourneurs abattus ;
//   - zones (Colline, Bases) : sécurisations, frags offensifs et défensifs de zone ;
//   - Oddball : porteurs du crâne abattus ;
//   - Stockpile : porteurs de graine abattus ;
//   - Extraction : conversions adverses refusées ; VIP : assistances sur le VIP.
//
// Ni les captures, vols, prises de zone, dépôts, conversions réussies (prendre) n'y figurent.
var ObjectiveSupportColumns = map[ObjectiveFamily][]string{
	FamilyCTF: {
		objectiveColFlagReturns, objectiveColFlagSecures, objectiveColFlagCaptureAssists,
		objectiveColFlagCarriersKilled, objectiveColFlagReturnersKilled,
	},
	FamilyZonesKOTH: {
		objectiveColZoneSecures, objectiveColZoneOffensiveKills, objectiveColZoneDefensiveKills,
	},
	FamilyZonesStrongholds: {
		objectiveColZoneSecures, objectiveColZoneOffensiveKills, objectiveColZoneDefensiveKills,
	},
	FamilyOddball:    {objectiveColSkullCarriersKilled},
	FamilyStockpile:  {objectiveColPowerSeedCarriersKilled},
	FamilyExtraction: {objectiveColExtractionConversionsDenied},
	FamilyVIP:        {objectiveColVipAssists},
}

// objectiveSupportFragColumns : parmi les colonnes de soutien, celles qui sont des FRAGS
// (garde-rail : sous-ensemble de ObjectiveSupportColumns, noms de frags).
var objectiveSupportFragColumns = map[string]bool{
	objectiveColFlagCarriersKilled:      true,
	objectiveColFlagReturnersKilled:     true,
	objectiveColZoneOffensiveKills:      true,
	objectiveColZoneDefensiveKills:      true,
	objectiveColSkullCarriersKilled:     true,
	objectiveColPowerSeedCarriersKilled: true,
}

// ObjectiveSupportPoints : actions de SOUTIEN pondérées d'un agrégat d'un match : poids de
// ObjectiveFamilyActionWeights, réduits de ObjectiveSupportFragWeight pour les frags.
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
		if objectiveSupportFragColumns[col] {
			w *= ObjectiveSupportFragWeight
		}
		points += w * v / float64(st.Matches)
	}
	return points
}
