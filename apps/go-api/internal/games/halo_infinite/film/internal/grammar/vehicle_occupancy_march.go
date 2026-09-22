package grammar

// vehicle_occupancy_march.go — L'EMBARQUEMENT LU DANS LE FILM (lot 5.10).
//
// L'ÉCRIVAIN, RELU EN LECTURE SEULE LE 2026-09-21 (`FUN_140c1e4d0`, image base 140000000). Le
// composant `i10 object-parent-state` écrit ses champs dans le bloc du composant, et DEUX
// d'entre eux portent l'attachement — ce sont ceux que les SENTINELLES désignent, parce que la
// branche LIBRE les efface explicitement :
//
//	+0x274  l'identifiant du PARENT (FUN_1406d3140, catégorie 1)  -> 0xffffffff quand libre
//	+0x3a0  un entier de SIX bits derrière un bit de signe        -> 0xffff quand absent
//
// Le reste de la branche attachée est une pose relative (un triplet 3 x R(16), une vitesse) et
// des drapeaux : rien qui puisse nommer un siège.
//
// CE QUE LA MESURE DIT (film `4f77afc1`, Flood Gulch, carte installée, oracle de contenu tenu —
// `i21` à 69,6 % sur 321 335 records `ti=35`) :
//
//	branche ATTACHÉE sur la bande bipède                    83 lectures, 47 slots
//	dont le parent tombe sur un slot `ti=40`, base 0x200    48  (57,8 %)
//	                                          base 0x300     4  ( 4,8 %)
//	                                          base 0        1  ( 1,2 %)
//	SIÈGE (+0x3a0) sur ces parents véhicule                 43 lues, 5 muettes
//	                                       dont {0, 1, 2}   42  — conducteur, passager, tourelle
//
// LA BASE EST MESURÉE, PAS SUPPOSÉE : `readQuantStat` rend la valeur SANS la base de sa
// catégorie (cf. `varwidth.go`, qui explique pourquoi elle n'est pas portée à la source) ; la
// table de `FUN_140d10bb0` n'en propose que quatre, et une seule fait atterrir les parents sur
// des véhicules.
//
// CE N'EST PAS UNE SOURCE D'ÉPISODES, ET C'EST ÉCRIT : le chemin delta ne porte que ce qui
// CHANGE, donc `i10` y est une TRANSITION (48 montées nommées quand le document publie 86
// épisodes). L'épisode reste construit par le trou de position et les événements ; ce canal-ci
// donne le SIÈGE, qui n'était lisible nulle part ailleurs (D1 du lot 5.5 : le champ `R(6)` de
// l'événement rend 153 fois `0` pour 100 tirs de tourelle).
//
// PUR au sens du paquet : la récolte ne lit pas le film, elle range ce que la marche des morts
// vient de traverser — même passe, aucun décodage supplémentaire.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// parentHandleBase est la BASE de la catégorie 1 de `FUN_1406d3140` (`FUN_140d10bb0` : base
// 0x200, plage 0x1FFF - 0x200), celle que le dépôt n'ajoute pas à la source. Mesure du
// 2026-09-21 : elle fait atterrir 57,8 % des parents sur un slot `ti=40` contre 4,8 % (0x300),
// 1,2 % (0) et 0,0 % (0x400) — cf. l'en-tête.
const parentHandleBase = 0x200

// parentHandleValueMask isole la valeur du handle : `readQuantStat` range les DEUX bits de queue
// (la génération) dans les bits 31-30.
const parentHandleValueMask = 0x3FFFFFFF

// occupancyFromRecord rend la lecture d'occupation d'un record de BIPÈDE, s'il en porte une.
//
// LE RATTACHEMENT EST EXACT : la valeur vient du composant lui-même (couche de capture), jamais
// d'un voisinage de position de bit.
func occupancyFromRecord(r *FrameRecord, atUS uint64) (types.VehicleOccupancy, bool) {
	if r.TypeIndex != BipedTypeIndex {
		return types.VehicleOccupancy{}, false
	}
	for _, c := range r.Trace.Comps {
		st, ok := c.ParentOf()
		if !ok {
			continue
		}
		out := types.VehicleOccupancy{
			TimestampUS: atUS, Slot: r.Slot, Gen: r.ID >> 30, Attached: st.Attached,
			HasSeat: st.HasTail6, Seat: st.Tail6,
		}
		if st.Attached {
			out.ParentSlot = (st.Quant16 & parentHandleValueMask) + parentHandleBase
			out.ParentGen = st.Quant16 >> 30
		}
		return out, true
	}
	return types.VehicleOccupancy{}, false
}

// dedupOccupancy trie et déduplique les lectures : la marche à trois vues republie le MÊME
// record, et deux vues d'un même paquet rendraient deux fois la même montée à bord.
func dedupOccupancy(in []types.VehicleOccupancy) []types.VehicleOccupancy {
	if len(in) == 0 {
		return nil
	}
	sort.SliceStable(in, func(i, j int) bool {
		switch {
		case in[i].TimestampUS != in[j].TimestampUS:
			return in[i].TimestampUS < in[j].TimestampUS
		case in[i].Slot != in[j].Slot:
			return in[i].Slot < in[j].Slot
		default:
			return in[i].ParentSlot < in[j].ParentSlot
		}
	})
	type key struct {
		at                   uint64
		slot, parent         uint32
		attached, hasSeat    bool
		seat, gen, parentGen uint32
	}
	seen := make(map[key]bool, len(in))
	out := make([]types.VehicleOccupancy, 0, len(in))
	for _, o := range in {
		k := key{o.TimestampUS, o.Slot, o.ParentSlot, o.Attached, o.HasSeat, o.Seat, o.Gen, o.ParentGen}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, o)
	}
	return out
}
