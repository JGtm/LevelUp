package powerpos

// reglage_v2.go — LE REGLAGE V2 (item 2bis.B du plan des positions de force, 2026-09-20).
//
// FIGE le 2026-09-20 sur les distributions de TROIS cartes de calibrage — recharge,
// aquarius, streets — SANS regarder l'oracle ; les neuf autres cartes servent de
// validation a l'etape 2bis.D. Chaque valeur est justifiee par un chiffre dans
// .ai/V7.5/positions_de_force/MESURE_EMPIRIQUE_V2_2026-09-20.md, section « Reglage v2
// fige ». Il ne se retouche pas apres le verdict v2 (D12 : meme regle que ReglageV1).
//
// # CE QUI CHANGE DE NATURE PAR RAPPORT A LA V1
//
//   - DEUX AXES ANGULAIRES A POIDS EGAUX. Couverture (dispersion des directions de tir) et
//     abri (1 - dispersion des directions d'ou l'on meurt) sont ANTICORRELES a -0,70 /
//     -0,73 / -0,54 sur les trois cartes : un lieu ouvert tue et meurt de partout. A poids
//     egaux, la composante commune (« ouverture ») s'annule et il ne reste que ce qu'un
//     lieu a d'ASYMETRIQUE — il voit beaucoup et n'est vu que de peu. C'est la propriete
//     que les guides decrivent, et la v1 ne la mesurait pas.
//   - LES POIDS SE DEDUISENT DE L'ETALEMENT MESURE de chaque axe (p90 - p10 par disque,
//     moyenne des trois cartes) : poids nominal = part voulue / etalement. Sans cette
//     regle, la portee (etalement 0,92 : sa normalisation p50-p90 sature) pesait en v1
//     autant que l'avantage (0,17) avec un poids cinq fois moindre.
//   - L'AVANTAGE EST PONDERE PAR LE RANG DU TUEUR (D10a) et retrograde d'une part sur
//     deux a une part sur cinq et demi : le verdict v1 a montre qu'il retrouve les bases.
//   - LA SELECTION EST UNE HYSTERESIS (amorce / croissance) SUIVIE D'UNE FERMETURE
//     MORPHOLOGIQUE ET COMPTEE EN 8-CONNEXITE : six des treize zones manquees par la v1
//     l'etaient par fragmentation (amas de 2 a 10 cellules pour 12 exigees), deux autres a
//     moins de 0,01 du seuil unique.

// ReglageV2 rend le reglage v2 FIGE le 2026-09-20.
func ReglageV2() Reglage {
	return Reglage{
		// Inchange depuis la v1 : le disque de 2 m, le plancher de 3 matchs et les 40
		// engagements sont des conditions de MESURE, pas des choix de score.
		RayonLissageM:        2.0,
		PlancherMatchs:       3,
		MinEngagementsDisque: 40,
		ForcePrior:           40,
		DeniveleReferenceM:   1.5,

		// Parts voulues : asymetrie 2, avantage 1, hauteur 1, intensite 0,5, portee 0,5.
		// Etalements mesures (moyenne recharge / aquarius / streets) : asymetrie 0,14,
		// avantage 0,17, hauteur 0,22, intensite 0,20, portee 0,92. Poids normalises a 1.
		PoidsCouverture: 0.26,
		PoidsAbri:       0.26,
		PoidsAvantage:   0.21,
		PoidsHauteur:    0.16,
		PoidsIntensite:  0.09,
		PoidsPortee:     0.02,

		// 20 directions : la moitie du plancher de 40 engagements — un disque au plancher
		// partage a egalite entre kills et morts en a exactement 20 de chaque cote. Le p10
		// mesure vaut 47 a 71 directions par disque : l'axe est presque toujours lu.
		MinDirections:      20,
		UtiliseRangPondere: true,

		// Selection : amorce au p95 de la carte (les maxima francs), croissance au p90
		// (un lieu s'etend tant que le score reste dans le decile superieur — au p80,
		// mesure : des lieux de 150 m2 sur streets, une salle entiere), fermeture
		// de rayon 1 (un trou d'une cellule n'est pas un mur), composantes en 8-connexite
		// APRES fermeture, 10 cellules minimum (2,5 m2), 8 positions au plus.
		QuantileSeuil:          0.90,
		SeuilScoreMin:          0.57,
		QuantileAmorce:         0.95,
		QuantileCroissance:     0.90,
		FermetureRayonCellules: 1,
		Connexite8:             true,
		TailleMiniComposante:   10,
		MaxComposantes:         8,
	}
}
