package replay

// positions_porte_vehicules.go — LA PORTE DES POSITIONS DE VEHICULE (lot M1 des retours du rejeu,
// 2026-09-23 ; annexe `.ai/V7.5/retours_rejeu_2026-09-23/RAPPORT_positions_limbe.md`).
//
// # LE DEFAUT, MESURE (111 documents)
//
// Les memes faux en-tetes que chez les joueurs (cf. positions_porte.go) atteignent le calque des
// vehicules par deux balayages : le nuage de positions `ti=40` et les records de creation. Le
// client tenait le dernier echantillon et INTERPOLAIT entre deux : un Mongoose immobile au point
// d apparition « partait seul » vers un echantillon lu 70 s plus tard a 193 m de la, puis revenait
// (81c02726, slot 770). Au parc : 11 allers-retours (1 344 s d affichage fautif), 5 vies fantomes
// dont tous les echantillons sont hors de la carte (1 948 s), 11 naissances hors de la carte
// (3 684 s au faux point d apparition).
//
// # DEUX REPLIS NOMMES, COMPTES, ET LEUR CRITERE DE RETRAIT
//
//	F-1  `repli_position_hors_emprise_ecartee` : echantillon ou naissance hors de l emprise jouee
//	     du film ET sans continuite physique avec une position dans l emprise (une chute dans un
//	     vide reste publiee) — la MEME emprise que les joueurs, mesuree une fois
//	     (emprise_jouee.go). Une
//	     naissance ecartee laisse la place au record de creation suivant de la meme vie, s il
//	     existe (`vehicleSpawnsByLife` retient le plus precoce : une fausse naissance anterieure
//	     l emportait sur la vraie, et donnait une famille de chassis inconnue).
//	F-2  `repli_echantillon_vehicule_au_travers_d_un_silence_ecarte` : un vehicule ne se deplace
//	     pas pendant un silence de replication. Mesure : 565 silences de plus de `lifeGapUS` entre
//	     deux echantillons d une meme vie, 21 avec un deplacement de plus de 2 m, et les 21
//	     touchent un echantillon aberrant ; les 544 autres ne deplacent rien.
//
// Une vie sans position restante (ni naissance ni echantillon) n est plus publiee : c est le
// `NoPosition` qui existait deja (cf. vehicle_tracks.go).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// vehicleSilenceMoveM est le deplacement, en metres (3D), au-dela duquel deux echantillons d une
// meme vie separes par un silence de plus de `lifeGapUS` se contredisent. MESURE (2026-09-23,
// 111 documents) : les 544 silences sans echantillon aberrant deplacent le vehicule de 2 m au plus,
// les 21 autres de 25 m au moins.
const vehicleSilenceMoveM = 2.0

// ecarterVehiculesHorsEmprise rend une copie du balayage sans les echantillons ni les records de
// creation hors de l emprise jouee (F-1), et les deux comptes. Le balayage d entree n est pas
// modifie.
func ecarterVehiculesHorsEmprise(scan VehicleScan, e empriseJouee, fb *fallback.Compteur,
) (out VehicleScan, echantillons, naissances int) {
	out = scan
	if !scan.Scanned || !e.armee {
		return out, 0, 0
	}
	triees := append([]grammar.BipedPosition(nil), scan.Positions...)
	sort.SliceStable(triees, func(i, j int) bool { return triees[i].TimestampUS < triees[j].TimestampUS })
	rejets := rejetsParSlot(triees, e)
	out.Positions = make([]grammar.BipedPosition, 0, len(triees))
	gardees := map[uint32][]grammar.BipedPosition{}
	for i, p := range triees {
		if rejets[i] {
			echantillons++
			continue
		}
		out.Positions = append(out.Positions, p)
		if p.HasWorld {
			gardees[p.Slot] = append(gardees[p.Slot], p)
		}
	}
	out.Creations = make([]types.EquipmentCreation, 0, len(scan.Creations))
	for _, c := range scan.Creations {
		if e.rejette(c.X, c.Y, c.Z) && !naissanceRelieeAuVol(c, gardees[c.Slot]) {
			naissances++
			continue
		}
		out.Creations = append(out.Creations, c)
	}
	ecartes := echantillons + naissances
	fb.DeclencheN(fallback.NomPositionHorsEmpriseEcartee, ecartes)
	return out, echantillons, naissances
}

// naissanceRelieeAuVol dit si une naissance hors de l emprise est reliee, par continuite, a un
// echantillon RETENU de son slot (cf. emprise_jouee.go) — un vehicule cree hors de la zone jouee
// (largue d en haut) puis replique jusqu a elle n est pas un artefact. `pos` est trie par instant.
func naissanceRelieeAuVol(c types.EquipmentCreation, pos []grammar.BipedPosition) bool {
	for _, p := range pos {
		if seRelient([3]float32{c.X, c.Y, c.Z}, [3]float32{p.X, p.Y, p.Z}, c.TimestampUS, p.TimestampUS) {
			return true
		}
	}
	return false
}

// sejourDeVehicule est une suite d echantillons d une vie sans silence de plus de `lifeGapUS`,
// en indices INCLUS dans la liste des positions du slot.
type sejourDeVehicule struct{ debut, fin int }

func (s sejourDeVehicule) taille() int { return s.fin - s.debut + 1 }

// ecarterLesSejoursAuTraversDUnSilence applique F-2 vie par vie et rend le nuage par slot sans
// les echantillons ecartes, avec leur nombre.
func ecarterLesSejoursAuTraversDUnSilence(bySlot map[uint32][]grammar.BipedPosition,
	lives []vehicleLife, fb *fallback.Compteur,
) (map[uint32][]grammar.BipedPosition, int) {
	ecartes := map[uint32]map[int]bool{}
	n := 0
	for _, l := range lives {
		pos := bySlot[l.key.Slot]
		for _, s := range sejoursEcartes(pos, sejoursDeLaVie(pos, l)) {
			if ecartes[l.key.Slot] == nil {
				ecartes[l.key.Slot] = map[int]bool{}
			}
			for i := s.debut; i <= s.fin; i++ {
				ecartes[l.key.Slot][i] = true
				n++
			}
		}
	}
	if n == 0 {
		return bySlot, 0
	}
	out := make(map[uint32][]grammar.BipedPosition, len(bySlot))
	for slot, pos := range bySlot {
		drop := ecartes[slot]
		if len(drop) == 0 {
			out[slot] = pos
			continue
		}
		garde := make([]grammar.BipedPosition, 0, len(pos)-len(drop))
		for i, p := range pos {
			if !drop[i] {
				garde = append(garde, p)
			}
		}
		out[slot] = garde
	}
	fb.DeclencheN(fallback.NomEchantillonVehiculeAuTraversDUnSilenceEcarte, n)
	return out, n
}

// sejoursDeLaVie decoupe les echantillons de la fenetre d une vie en sejours de replication.
func sejoursDeLaVie(pos []grammar.BipedPosition, l vehicleLife) []sejourDeVehicule {
	var out []sejourDeVehicule
	for i, p := range pos {
		if p.TimestampUS < l.loUS || p.TimestampUS > l.hiUS {
			continue
		}
		n := len(out)
		if n > 0 && out[n-1].fin == i-1 && int64(p.TimestampUS)-int64(pos[i-1].TimestampUS) <= lifeGapUS {
			out[n-1].fin = i
			continue
		}
		out = append(out, sejourDeVehicule{debut: i, fin: i})
	}
	return out
}

// sejoursEcartes rend les sejours qu il faut ecarter pour qu aucun silence ne deplace le vehicule.
//
// DE DEUX SEJOURS QUI SE CONTREDISENT, L UN EST ABERRANT : on ecarte le plus court (un faux
// en-tete est un echantillon isole — 1 a 3 au parc) ; a egalite, celui que son AUTRE voisin ne
// soutient pas ; a egalite encore, le plus tardif. Le sejour garde est ensuite confronte au
// suivant : une excursion (aller puis retour) tombe donc en un seul geste, et ses deux voisins,
// qui s accordent, restent.
func sejoursEcartes(pos []grammar.BipedPosition, sejours []sejourDeVehicule) []sejourDeVehicule {
	var gardes, out []sejourDeVehicule
	for k := 0; k < len(sejours); k++ {
		s := sejours[k]
		for {
			if len(gardes) == 0 {
				gardes = append(gardes, s)
				break
			}
			prec := gardes[len(gardes)-1]
			if !deplaceAuTraversDuSilence(pos[prec.fin], pos[s.debut]) {
				gardes = append(gardes, s)
				break
			}
			if !ecarterLePrecedent(pos, gardes, sejours, k) {
				out = append(out, s)
				break
			}
			out = append(out, prec)
			gardes = gardes[:len(gardes)-1]
		}
	}
	return out
}

// ecarterLePrecedent tranche entre le dernier sejour garde et le sejour `k` qui le contredit.
func ecarterLePrecedent(pos []grammar.BipedPosition, gardes, sejours []sejourDeVehicule, k int) bool {
	prec, s := gardes[len(gardes)-1], sejours[k]
	if prec.taille() != s.taille() {
		return prec.taille() < s.taille()
	}
	precSoutenu := len(gardes) > 1 && !deplaceAuTraversDuSilence(pos[gardes[len(gardes)-2].fin], pos[prec.debut])
	sSoutenu := k+1 < len(sejours) && !deplaceAuTraversDuSilence(pos[s.fin], pos[sejours[k+1].debut])
	return sSoutenu && !precSoutenu
}

// deplaceAuTraversDuSilence dit si deux echantillons consecutifs d une vie sont separes par un
// silence de plus de `lifeGapUS` ET par un deplacement de plus de `vehicleSilenceMoveM`.
func deplaceAuTraversDuSilence(a, b grammar.BipedPosition) bool {
	if int64(b.TimestampUS)-int64(a.TimestampUS) <= lifeGapUS {
		return false
	}
	return dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z}) > vehicleSilenceMoveM
}
