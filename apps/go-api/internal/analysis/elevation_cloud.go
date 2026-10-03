// Package analysis — elevation_cloud.go : LE NUAGE BRUT « DISTANCE × DÉNIVELÉ » (décision D25).
//
// # CE QUE CE FICHIER PRODUIT
//
// Un point PAR FRAG MESURÉ — aucune agrégation, aucun binning, aucun plafond — plus quatre
// quantiles par côté. C'est le contrat de la proposition T5 de la maquette
// `.ai/V7.5/MAQUETTE_DENIVELE_V2_2026-09-22.html` : « rien n'est agrégé, chaque frag reste un
// point, donc les cas extrêmes survivent ». Un sous-échantillonnage rendrait deux dessins
// différents pour la même fenêtre — un échantillon n'est pas une mesure.
//
// # LE SIGNE EST POSÉ ICI, UNE FOIS, ET JAMAIS RECALCULÉ EN AVAL
//
// `DeltaZM` répond toujours à la même question, des deux côtés : « MOI, étais-je au-dessus de
// l'autre ? ». Positif = j'étais au-dessus, pour un frag COMME POUR UNE MORT. C'est
// `signedElevation` (weapon_range.go) qui l'établit, et c'est la seule écriture de cette
// convention dans le dépôt — le front reçoit un nombre déjà signé et ne le retouche pas.
//
// # LA DIMENSION ARME NE COMPTE PAS (D23-b)
//
// Aucun regroupement par arme, donc aucun seuil de publication : ce nuage parle de POSITION,
// pas d'outil. La clé d'arme reste sur le point, pour l'infobulle seulement.
package analysis

import "sort"

// ElevationPoint est UN frag mesuré, vu comme un point du plan « distance × dénivelé ».
type ElevationPoint struct {
	// DistanceM : distance 3D tueur <-> victime, en mètres (abscisse).
	DistanceM float64
	// DeltaZM : dénivelé SIGNÉ DU POINT DE VUE DU JOUEUR (ordonnée). Positif = j'étais
	// au-dessus. Voir l'en-tête : il ne se recalcule nulle part ailleurs.
	DeltaZM float64
	// MatchID, TimeMS, WeaponKey identifient le frag pour l'infobulle et le rejeu. La clé
	// d'arme est une clé de REGISTRE, jamais un `source_tag` brut ; c'est l'appelant qui la
	// traduit en libellé (adaptateur sémantique).
	MatchID   string
	TimeMS    int64
	WeaponKey string
}

// ElevationSideSummary : les quantiles d'UN côté, plus son effectif.
//
// P25/P50/P75 EN DISTANCE ET EN DÉNIVELÉ, SÉPARÉMENT — ce ne sont pas les coordonnées de
// points existants mais deux quantiles marginaux, qui dessinent le rectangle de masse du
// nuage (« halo » de la maquette) et son point médian. Un effectif nul rend un résumé à zéro
// et `N == 0` : c'est `N` qui dit s'il y a quelque chose à tracer, jamais les valeurs.
type ElevationSideSummary struct {
	DistanceP25, DistanceP50, DistanceP75 float64
	DeltaZP25, DeltaZP50, DeltaZP75       float64
	N                                     int
}

// ElevationCloud : les deux côtés du nuage et leurs résumés.
type ElevationCloud struct {
	Kills, Deaths               []ElevationPoint
	KillsSummary, DeathsSummary ElevationSideSummary
}

// BuildElevationCloud projette des frags mesurés en nuage. PUR : l'entrée n'est ni triée ni
// mutée.
//
// ORDRE DE SORTIE DÉTERMINISTE (match, instant, arme) : deux appels sur la même fenêtre
// rendent le même tableau, donc le même dessin et le même diff de test.
func BuildElevationCloud(kills []MeasuredKill) ElevationCloud {
	var cloud ElevationCloud
	for _, k := range kills {
		p := ElevationPoint{
			DistanceM: k.DistanceM,
			DeltaZM:   signedElevation(k),
			MatchID:   k.MatchID,
			TimeMS:    k.TimeMS,
			WeaponKey: k.WeaponKey,
		}
		if k.Side == SideVictim {
			cloud.Deaths = append(cloud.Deaths, p)
			continue
		}
		cloud.Kills = append(cloud.Kills, p)
	}
	sortElevationPoints(cloud.Kills)
	sortElevationPoints(cloud.Deaths)
	cloud.KillsSummary = summarizeElevation(cloud.Kills)
	cloud.DeathsSummary = summarizeElevation(cloud.Deaths)
	return cloud
}

// sortElevationPoints impose l'ordre de sortie.
func sortElevationPoints(pts []ElevationPoint) {
	sort.SliceStable(pts, func(i, j int) bool {
		if pts[i].MatchID != pts[j].MatchID {
			return pts[i].MatchID < pts[j].MatchID
		}
		if pts[i].TimeMS != pts[j].TimeMS {
			return pts[i].TimeMS < pts[j].TimeMS
		}
		return pts[i].WeaponKey < pts[j].WeaponKey
	})
}

// summarizeElevation rend les quantiles d'un côté.
//
// `percentileLinear` EST RÉUTILISÉ TEL QUEL (weapon_range.go) : c'est la définition « type 7 »
// déjà retenue pour les bâtons p10→p90 de la portée. Deux définitions de quantile sur la même
// page auraient fait diverger le halo de son propre bâton.
func summarizeElevation(pts []ElevationPoint) ElevationSideSummary {
	if len(pts) == 0 {
		return ElevationSideSummary{}
	}
	dist := make([]float64, 0, len(pts))
	dz := make([]float64, 0, len(pts))
	for _, p := range pts {
		dist = append(dist, p.DistanceM)
		dz = append(dz, p.DeltaZM)
	}
	sort.Float64s(dist)
	sort.Float64s(dz)
	return ElevationSideSummary{
		DistanceP25: percentileLinear(dist, 25),
		DistanceP50: percentileLinear(dist, 50),
		DistanceP75: percentileLinear(dist, 75),
		DeltaZP25:   percentileLinear(dz, 25),
		DeltaZP50:   percentileLinear(dz, 50),
		DeltaZP75:   percentileLinear(dz, 75),
		N:           len(pts),
	}
}
