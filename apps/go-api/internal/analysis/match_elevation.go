// Package analysis — match_elevation.go : LE DÉNIVELÉ D'UN MATCH, AU GRAIN DU FRAG.
//
// Frère de weapon_range.go, et son CONTRAIRE d'échelle : celui-là agrège des centaines de
// frags en une ligne par (arme, côté) ; celui-ci n'agrège RIEN. Un match donne 10 à 25 frags,
// c'est-à-dire la densité d'un nuage lisible — le seuil de mesure, les percentiles et le
// binning n'y ont aucun objet (décision D24, 2026-09-22).
//
// CE QU'IL PARTAGE AVEC SON FRÈRE, ET C'EST LE POINT : `signedElevation`. Le signe du
// dénivelé est le point le plus facile à inverser du dépôt ; il est écrit une seule fois, et
// les deux lectures l'empruntent. Recopier « if côté victime alors -dz » ici aurait fait une
// seconde vérité à tenir en phase.
//
// AUCUNE LECTURE DE BASE, AUCUN LIBELLÉ FABRIQUÉ : les lignes arrivent déjà mesurées et déjà
// nommées (le repo a résolu l'arme par le référentiel du titre). Ce fichier ne fait que
// choisir un point de vue, trier, et rendre une médiane.
package analysis

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// BuildMatchElevation assemble le bloc `combat_tab.elevation` pour UN match.
//
// viewerXUID est le joueur de la page : ses frags et ses morts forment `Kills`, tous signés
// de SON point de vue. Les frags des autres forment `Lobby`, signés du leur (côté tueur) —
// un fond de comparaison, cf. la doctrine du DTO.
//
// totalKills est le nombre de frags que le TABLEAU DES SCORES compte au joueur : il n'est pas
// recalculé ici (les lignes reçues sont, par construction, les seules MESURÉES — en déduire
// un total serait le confondre avec la couverture qu'il sert justement à révéler).
//
// nil quand aucune ligne ne concerne le joueur ET que le lobby est vide : il n'y a alors rien
// à tracer, et un bloc présent mais vide ferait afficher un nuage sans points.
func BuildMatchElevation(
	rows []domain.MatchElevationKillRaw, viewerXUID string, totalKills int,
) *domain.MatchElevationBlock {
	if len(rows) == 0 || viewerXUID == "" {
		return nil
	}
	mine := make([]domain.MatchElevationKill, 0, len(rows))
	lobby := make([]domain.MatchElevationKill, 0, len(rows))
	measured := 0
	for _, r := range rows {
		switch {
		case r.KillerXUID == viewerXUID:
			measured++
			mine = append(mine, elevationPoint(r, SideKiller, domain.MatchElevationSideKill, r.VictimGamertag))
		case r.VictimXUID == viewerXUID && r.VictimXUID != "":
			mine = append(mine, elevationPoint(r, SideVictim, domain.MatchElevationSideDeath, r.KillerGamertag))
		default:
			lobby = append(lobby, elevationPoint(r, SideKiller, domain.MatchElevationSideKill, r.VictimGamertag))
		}
	}
	if len(mine) == 0 && len(lobby) == 0 {
		return nil
	}
	sortElevationKills(mine)
	sortElevationKills(lobby)
	return &domain.MatchElevationBlock{
		Kills:              mine,
		Lobby:              lobby,
		LobbyMedianDeltaZM: medianDeltaZ(lobby),
		MeasuredKills:      measured,
		TotalKills:         totalKills,
	}
}

// elevationPoint ramène une ligne brute au point de vue demandé. `side` est le côté au sens
// de l'AGRÉGAT (il pilote le signe, via `signedElevation`) ; `publishedSide` est le mot du
// CONTRAT ; `opponent` est l'autre joueur, que l'appelant choisit parce que lui seul sait de
// quel côté il lit.
func elevationPoint(
	r domain.MatchElevationKillRaw, side Side, publishedSide, opponent string,
) domain.MatchElevationKill {
	return domain.MatchElevationKill{
		DistanceM: r.DistanceM,
		DeltaZM:   signedElevation(MeasuredKill{Side: side, DeltaZ: r.DeltaZ}),
		TimeMS:    r.TimeMS,
		Weapon:    r.Weapon,
		WeaponEN:  r.WeaponEN,
		Side:      publishedSide,
		Opponent:  opponent,
	}
}

// sortElevationKills rend la sortie DÉTERMINISTE : l'ordre du match (l'instant), puis la
// distance pour départager deux frags au même instant sur des joueurs différents.
func sortElevationKills(ks []domain.MatchElevationKill) {
	sort.SliceStable(ks, func(i, j int) bool {
		if ks[i].TimeMS != ks[j].TimeMS {
			return ks[i].TimeMS < ks[j].TimeMS
		}
		return ks[i].DistanceM < ks[j].DistanceM
	})
}

// medianDeltaZ rend la médiane des dénivelés, en mètres. 0 sur une liste vide — la valeur
// d'un repère que l'appelant ne trace pas, cf. la doctrine du champ.
//
// MÉDIANE PAR INTERPOLATION, comme partout dans le paquet (`percentileLinear`) : un repère
// qui sauterait d'une valeur d'échantillon à l'autre selon la parité de l'effectif se lirait
// comme une instabilité de la mesure.
func medianDeltaZ(ks []domain.MatchElevationKill) float64 {
	if len(ks) == 0 {
		return 0
	}
	dz := make([]float64, 0, len(ks))
	for _, k := range ks {
		dz = append(dz, k.DeltaZM)
	}
	sort.Float64s(dz)
	return percentileLinear(dz, 50)
}
