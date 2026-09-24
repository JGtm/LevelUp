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
//	     naissance ecartee laisse la place au record de creation suivant du meme (slot, gen), s il
//	     existe ET s il ne suit pas la fin de la vie (`vehicleSpawnsByLife` retient le plus precoce
//	     de ceux-la : une fausse naissance anterieure l emportait sur la vraie, et donnait une
//	     famille de chassis inconnue ; un record posterieur a la fin est celui d un objet ulterieur).
//	F-2  `repli_echantillon_vehicule_au_travers_d_un_silence_ecarte` : un vehicule ne se deplace
//	     pas pendant un silence de replication. Mesure : 565 silences de plus de `lifeGapUS` entre
//	     deux echantillons d une meme vie, 21 avec un deplacement de plus de 2 m, et les 21
//	     touchent un echantillon aberrant ; les 544 autres ne deplacent rien. F-2 n ecarte qu un
//	     sejour COURT (`vehicleSejourAberrantMax`) et seulement quand une preuve departage ; sinon
//	     il laisse les deux sejours publies et compte son refus (`silencesNonTranches`).
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

// vehicleSejourAberrantMax est la taille maximale, en echantillons BRUTS du film, d un sejour que
// F-2 peut ecarter. Au-dela, F-2 n ecarte RIEN : il compte un silence non tranche.
//
// MESURE (2026-09-24, 111 documents reconstruits depuis les faits persistes, F-1 debranche pour
// voir TOUS les faux en-tetes) : les sejours aberrants font 1 echantillon (9 cas) ou 2 (1 cas,
// 233 ms). La borne laisse une marge d un echantillon. Sans elle, F-2 ecartait le plus court des
// deux sejours QUELLE QUE SOIT sa taille : une vraie trajectoire reprise apres un silence de plus
// de `lifeGapUS` a plus de 2 m disparaissait en bloc (revue adverse du 2026-09-24).
const vehicleSejourAberrantMax = 3

// bilanSilences est ce que F-2 a fait sur un document : les echantillons ecartes, et les silences
// avec deplacement qu il a refuse de trancher.
type bilanSilences struct{ ecartes, nonTranches int }

// ecarterLesSejoursAuTraversDUnSilence applique F-2 vie par vie et rend le nuage par slot sans
// les echantillons ecartes, avec le bilan. `spawns` porte la naissance de chaque vie (apres F-1) :
// c est la voisine gauche de son premier sejour.
func ecarterLesSejoursAuTraversDUnSilence(bySlot map[uint32][]grammar.BipedPosition,
	lives []vehicleLife, spawns map[types.EquipmentLifeKey]types.EquipmentCreation, fb *fallback.Compteur,
) (map[uint32][]grammar.BipedPosition, bilanSilences) {
	ecartes := map[uint32]map[int]bool{}
	var b bilanSilences
	for _, l := range lives {
		pos := bySlot[l.key.Slot]
		d := departage{pos: pos, sejours: sejoursDeLaVie(pos, l)}
		if sp, ok := spawns[l.key]; ok && sp.TimestampUS > 0 {
			d.naissance = &sp
		}
		tombes, refus := d.sejoursEcartes()
		b.nonTranches += refus
		for _, s := range tombes {
			if ecartes[l.key.Slot] == nil {
				ecartes[l.key.Slot] = map[int]bool{}
			}
			for i := s.debut; i <= s.fin; i++ {
				ecartes[l.key.Slot][i] = true
				b.ecartes++
			}
		}
	}
	if b.ecartes == 0 {
		return bySlot, b
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
	fb.DeclencheN(fallback.NomEchantillonVehiculeAuTraversDUnSilenceEcarte, b.ecartes)
	return out, b
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

// soutien est ce que l AUTRE voisin d un sejour dit de lui : il le contredit, il n existe pas, ou il
// le soutient. L ordre des valeurs est celui de la preuve.
type soutien int

const (
	soutienContredit soutien = iota
	soutienInconnu
	soutienSoutenu
)

// victime designe le sejour que F-2 ecarte parmi deux sejours qui se contredisent.
type victime int

const (
	victimeAucune victime = iota
	victimePrecedent
	victimeSuivant
)

// departage porte ce dont F-2 a besoin pour trancher dans UNE vie : son nuage (celui du slot, trie),
// ses sejours, et sa naissance lue (nil sans record de creation).
type departage struct {
	pos       []grammar.BipedPosition
	sejours   []sejourDeVehicule
	naissance *types.EquipmentCreation
}

// sejoursEcartes rend les sejours qu il faut ecarter pour qu aucun silence ne deplace le vehicule,
// et le nombre de silences avec deplacement laisses tels quels.
//
// DE DEUX SEJOURS QUI SE CONTREDISENT, L UN EST ABERRANT, et c est une PREUVE qui le designe, dans
// cet ordre : celui que son AUTRE voisin contredit (la naissance de la vie est la voisine gauche du
// premier sejour — elle est lue, et elle ne ment pas sur la position d un vehicule qui n a pas
// encore bouge) ; a soutien egal, le plus court. S il n y a pas de preuve (taille et soutien
// egaux), ou si le sejour designe depasse `vehicleSejourAberrantMax`, F-2 REFUSE DE TRANCHER : les
// deux restent publies, et le refus se compte — jamais « le plus tardif » par defaut.
//
// Le sejour garde est ensuite confronte au suivant : une excursion (aller puis retour) tombe donc
// en un seul geste, et ses deux voisins, qui s accordent, restent.
func (d departage) sejoursEcartes() (out []sejourDeVehicule, refus int) {
	var gardes []sejourDeVehicule
	for k, s := range d.sejours {
		for {
			if len(gardes) == 0 {
				gardes = append(gardes, s)
				break
			}
			prec := gardes[len(gardes)-1]
			if !deplaceAuTraversDuSilence(d.pos[prec.fin], d.pos[s.debut]) {
				gardes = append(gardes, s)
				break
			}
			v := d.victime(gardes, k)
			if v == victimePrecedent {
				out = append(out, prec)
				gardes = gardes[:len(gardes)-1]
				continue
			}
			if v == victimeSuivant {
				out = append(out, s)
			} else {
				refus++
				gardes = append(gardes, s)
			}
			break
		}
	}
	return out, refus
}

// victime tranche entre le dernier sejour garde et le sejour `k` qui le contredit.
func (d departage) victime(gardes []sejourDeVehicule, k int) victime {
	prec, s := gardes[len(gardes)-1], d.sejours[k]
	sp, ss := d.soutienDuPrecedent(gardes), d.soutienDuSuivant(k)
	v := victimeAucune
	switch {
	case sp < ss:
		v = victimePrecedent
	case ss < sp:
		v = victimeSuivant
	case prec.taille() < s.taille():
		v = victimePrecedent
	case s.taille() < prec.taille():
		v = victimeSuivant
	}
	if (v == victimePrecedent && prec.taille() > vehicleSejourAberrantMax) ||
		(v == victimeSuivant && s.taille() > vehicleSejourAberrantMax) {
		return victimeAucune
	}
	return v
}

// soutienDuPrecedent : le sejour garde avant lui, ou, pour le premier sejour, la naissance de la
// vie quand elle le precede.
func (d departage) soutienDuPrecedent(gardes []sejourDeVehicule) soutien {
	debut := d.pos[gardes[len(gardes)-1].debut]
	if len(gardes) > 1 {
		return soutienEntre(d.pos[gardes[len(gardes)-2].fin], debut)
	}
	if d.naissance == nil || d.naissance.TimestampUS > debut.TimestampUS {
		return soutienInconnu
	}
	n := grammar.BipedPosition{X: d.naissance.X, Y: d.naissance.Y, Z: d.naissance.Z,
		TimestampUS: d.naissance.TimestampUS, HasWorld: true}
	return soutienEntre(n, debut)
}

// soutienDuSuivant : le sejour qui suit le sejour `k` dans la vie.
func (d departage) soutienDuSuivant(k int) soutien {
	if k+1 >= len(d.sejours) {
		return soutienInconnu
	}
	return soutienEntre(d.pos[d.sejours[k].fin], d.pos[d.sejours[k+1].debut])
}

// soutienEntre dit si deux positions d une meme vie s accordent : au travers d un silence de plus
// de `lifeGapUS`, a au plus `vehicleSilenceMoveM` (un vehicule ne bouge pas pendant un silence) ;
// sans silence, a une vitesse physique (`continuiteVitesseMaxMPS`, cf. emprise_jouee.go).
func soutienEntre(a, b grammar.BipedPosition) soutien {
	dt := int64(b.TimestampUS) - int64(a.TimestampUS)
	if dt < 0 {
		dt = -dt
	}
	d := dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z})
	borne := continuiteVitesseMaxMPS * max(float64(dt)/1e6, continuitePlancherS)
	if dt > lifeGapUS {
		borne = vehicleSilenceMoveM
	}
	if d <= borne {
		return soutienSoutenu
	}
	return soutienContredit
}

// deplaceAuTraversDuSilence dit si deux echantillons consecutifs d une vie sont separes par un
// silence de plus de `lifeGapUS` ET par un deplacement de plus de `vehicleSilenceMoveM`.
func deplaceAuTraversDuSilence(a, b grammar.BipedPosition) bool {
	if int64(b.TimestampUS)-int64(a.TimestampUS) <= lifeGapUS {
		return false
	}
	return dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z}) > vehicleSilenceMoveM
}
