package replay

// equipment_placement_ends.go — LA FIN D'AFFICHAGE OBSERVÉE d'une pose (schéma 28).
//
// Extrait de `equipment_placements.go` au lot 1.9.1 (2026-09-15) pour tenir le seuil de
// 500 lignes du dépôt. DÉPLACEMENT PUR : aucune ligne de logique n'a changé.
//
// CE QU'ELLE MESURE, ET CE QU'ELLE NE MESURE PAS : `T1` est la fin du MOUVEMENT d'un objet, pas
// sa disparition (cf. le contrat d'[EquipmentPlacement]). La fin d'affichage, elle, est bornée
// par le RECENSEMENT des images-clés — dernière image qui voit l'objet, première qui ne le voit
// plus — exactement comme la chaîne des socles (`gwPickupBoundsFrom`, un seul exemplaire).

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// placEnd est la fin d'affichage observée d'UNE pose, sur l'axe du document.
type placEnd struct {
	until, untilMax int
	end             string
}

// placementEnds borne la disparition de chaque pose par le recensement des images-clés —
// mêmes règles que la chaîne des socles (`gwPickupBoundsFrom`, un seul exemplaire), la vie
// d'une clé étant fermée par la pose SUIVANTE de la même clé (le pool de clés reboucle) — et, depuis
// le lot J5.4 (RB2-3), par la pose suivante de son SLOT, quelle que soit sa génération
// ([naissancesDObjetParSlot]).
//
// Rendu INDEXÉ sur `raw` : l'appelant filtre les poses hors axe après coup, l'index doit
// survivre au filtre.
func placementEnds(
	raw []types.EquipmentPlacement, census grammar.WorldObjectKeyframes, clock replayClock,
) []placEnd {
	byLife := map[types.LifeKey][]int{}
	for i, p := range raw {
		byLife[p.Life] = append(byLife[p.Life], i)
	}
	// +1 : la fenêtre de recensement (`gwPickupSeenWithin`) est EXCLUSIVE sur sa borne haute.
	// À la fin de film exacte, la DERNIÈRE image-clé serait retranchée et une pose encore
	// recensée à cette image-clé sortirait « disparue » au lieu d'« ouverte ».
	filmEnd := census.LastTimeUS() + 1
	// Toute pose balayee est publiee : elle est donc RETENUE, et elle borne la precedente de son slot.
	parSlot := naissancesDObjetParSlot{}
	for _, p := range raw {
		parSlot.ajouter(p.Life.Slot, naissanceDObjet{tUS: p.T0US, gen: p.Life.Gen, retenue: true})
	}
	parSlot.trier()
	out := make([]placEnd, len(raw))
	for life, idxs := range byLife {
		// Tri total (J12.1, DT-9) : clé (T0US, index dans raw) ; l'index est unique et suit
		// l'ordre du film, et la pose suivante borne la fin de vie de la précédente.
		slices.SortFunc(idxs, func(a, b int) int { return cmp.Or(cmp.Compare(raw[a].T0US, raw[b].T0US), cmp.Compare(a, b)) })
		for j, i := range idxs {
			lifeEnd := filmEnd
			if j+1 < len(idxs) {
				lifeEnd = raw[idxs[j+1]].T0US
			}
			lifeEnd = parSlot.finDeVie(life, raw[i].T0US, lifeEnd)
			seen := gwPickupSeenWithin(census.SeenUS[life], raw[i].T0US, lifeEnd)
			b := gwPickupBoundsFrom(raw[i].T0US, lifeEnd, filmEnd, census.TimesUS, seen)
			switch {
			case b.NeverPicked || b.NoLaterKF:
				out[i] = placEnd{
					until: clock.frames - 1, untilMax: clock.frames - 1,
					end: GroundWeaponEndOpen,
				}
			default:
				e := placEnd{
					until:    clampFrame(frameOf(b.LowUS, clock.origin, clock.step), clock.frames),
					untilMax: clampFrame(frameOf(b.HighUS, clock.origin, clock.step), clock.frames),
					end:      GroundWeaponEndSeen,
				}
				if e.untilMax < e.until {
					e.untilMax = e.until
				}
				out[i] = e
			}
		}
	}
	return out
}
