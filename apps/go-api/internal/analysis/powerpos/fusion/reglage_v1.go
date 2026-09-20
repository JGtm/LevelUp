package fusion

import "levelup/go-api/internal/analysis/powerpos"

// reglage_v1.go — LE REGLAGE DE FUSION V1, FIGE LE 2026-09-20 (item 2bis.D, seconde
// moitie), gagnant du balayage de calibrage sur Recharge, Aquarius et Streets
// (`cmd/mappower-build/verdict_fusion_calibrage_research_test.go`, table complete dans
// `.ai/V7.5/positions_de_force/fusion_2026-09-20/_calibrage_fusion.md`). C'est le SEUL
// reglage du chantier choisi en regardant l'oracle — sur les cartes de calibrage seulement,
// jamais sur celles de validation (plan D11). Il ne se retouche pas apres le verdict de
// fusion (D12) : une retouche = un nouveau balayage, un nouveau document.
//
// # CE QUE LE BALAYAGE A DIT (112 candidats, tri rappel puis precision puis pieges purs)
//
//   - a = 0,6 / b = 0,4 : la geometrie pese un peu plus que l'empirique. A a = 1 (geometrie
//     seule sous la meme selection) le rappel moyen tombe a 0,70 ; a a = 0,3 a 0,82 ; a
//     a = 0,6 il vaut 0,89 (aquarius 1,00, recharge 0,67, streets 1,00). La fusion bat
//     chacune des deux voies seules, et ce n'est pas une frontiere de la grille.
//   - emp absent 0,25 (une cellule du sol sans engagement mesure est PENALISEE, pas
//     neutre) et geo absent 0 (une cellule scorable hors du sol derive ne recoit rien de
//     la geometrie) : les deux valeurs les plus severes gagnent — la fusion cherche les
//     cellules que les DEUX voies connaissent.
//   - amorce p90, croissance p80 : plus bas que la v2 (p95 / p90). Le score fusionne est
//     plus lisse que le score v2 (la geometrie n'a pas de bruit binomial), et une
//     croissance au p80 n'y produit pas les salles de 150 m2 que la v2 produisait ; le
//     verdict de fusion (section 6, couloirs) le controle.
//   - precision moyenne du gagnant sur le calibrage : 0,49 (< 0,60). Le rappel etait le
//     premier critere ecrit ; la precision se juge sur la validation.
//
// # LE PLANCHER ABSOLU
//
// SeuilScoreMin est pose APRES le gel, sous l'amorce la plus basse mesuree sur les trois
// cartes de calibrage avec ce reglage (meme recette que la v2 : un filet qui ne mord sur
// aucune carte de calibrage). Il ne fait pas partie du balayage. Amorces mesurees le
// 2026-09-20 (p90 du score fusionne) : recharge 0,759, aquarius 0,714, streets 0,742 —
// plancher 0,70. Il est pose AVANT tout regard sur les cartes de validation ; s'il y mord
// (une amorce de validation sous 0,70), le verdict le dit.

// ReglageFusionV1 rend le reglage de fusion fige le 2026-09-20.
func ReglageFusionV1() Reglage {
	return Reglage{
		PoidsGeo:  0.6,
		PoidsEmp:  0.4,
		EmpAbsent: 0.25,
		GeoAbsent: 0,
		Selection: powerpos.Reglage{
			QuantileAmorce:         0.90,
			QuantileCroissance:     0.80,
			SeuilScoreMin:          0.70,
			FermetureRayonCellules: 1,
			Connexite8:             true,
			TailleMiniComposante:   10,
			MaxComposantes:         8,
		},
	}
}
