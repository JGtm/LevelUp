package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// keyframe_datums.go — LA TABLE DE DATUMS PAR SLOT, LUE DANS L IMAGE-CLE (lot 5.16.2/5.16.4).
//
// # LE MODELE, ET IL EST CELUI DU JEU
//
// La branche VIVE de la boucle de records de la vue B (`FUN_1406cd128` avec
// `DAT_14474cd78 != 0`, sa valeur dans l image) dispatche par `FUN_1406cbaa0`, et son cas
// DELTA ne lit un corps QUE si le slot figure dans la TABLE DE DATUMS du decodeur partage :
//
//	t = *(decodeur + 0x20) ; n = (*(decodeur + 0x28) - t) / 200
//	si (eid == 0xffffffff || n <= slot || *(uint *)(slot * 200 + t) != eid) :
//	      code = 3 - (*(int *)(DAT_144c1cfa8 + 4) != 2)      <- 2 ou 3, ZERO bit lu
//	sinon : selecteur de baseline, puis FUN_1406caad8, qui lit l ARCHETYPE en
//	        *(int *)(slot * 200 + t + 4) et appelle l iterateur FUN_14076cb60
//
// Le monde hors ligne n avait PAS cette table : `rejetDeVue` transcrivait la garde de la
// branche 0 (`vue[0x38]`, pas de 0xa0, eid ET type), celle que le jeu n emprunte que pour
// l aller-retour d etat de `FUN_1428e24bc`. Mesure du lot 5.15 : la vue B sortait sur ce rejet
// 23 452 fois contre 400 terminateurs sur `bfecd02b`, et 21 988 des 22 112 rejets lisibles
// portaient sur un slot que le monde n avait JAMAIS lie.
//
// # LA SOURCE DE LA TABLE, PAR ADRESSE : L IMAGE-CLE EST SON DUMP
//
//	FUN_142f2e174   slot 0x10 de la vtable de vue — parcourt la table de SA vue
//	                (`vue+0x38` a `vue+0x40`) sous le bitmap `vue+0x58`, et rend UN MOT de
//	                32 bits par entite VIVANTE : `vue+8 << 0x1e | slot & 0x1fff | genre`
//	FUN_142f2c658   serialise chaque mot selon son genre : FUN_142f303bc (1),
//	                FUN_142f304a8 (2), FUN_142f30610 (3 = l etat complet)
//	FUN_142f30610   ecrit l en-tete par FUN_142f2c754(writer, 3, eid, archetype), ou
//	                archetype = *(int *)(*(vue[0x20] + 0x120) + 4 + slot * 0x18)
//
// L en-tete que le film porte est donc `[id:32][field:26][ti:6]`, et c est exactement celui que
// `readKeyframeHeader` / `kfAnchorFromID` lisent.
//
// # POURQUOI CETTE LECTURE-CI, ET PAS LA MARCHE D IMAGE-CLE EXISTANTE
//
// `WalkKeyframeWorld` est un BALAYEUR d ancres : il suit la CHAINE des records (frontiere par
// frontiere, slots croissants, fenetre de recherche de 120 000 bits) parce qu il rend aussi les
// POSITIONS de chaque record, dont les lecteurs d etat ont besoin. Cette chaine se coupe :
// mesure du lot 5.16.2 sur `dad793c7`, le chunk 1 rend 123 records (dernier slot 122, bit
// 139 754 sur 1 028 032) alors que le MEME payload declare le slot 1298 (`id 0x40000512`,
// `field26 = 0`, `ti = 47`) au bit 279 659 — 139 841 bits plus loin, au-dela de la fenetre.
//
// Or le modele du jeu n a besoin QUE de `slot -> archetype`. Cette table-ci se lit donc a
// POSITION LIBRE : les gardes de `kfAnchorFromID` a chaque position de bit, SANS la contrainte
// de croissance qui appartient a une marche de records. Un slot vu avec DEUX archetypes est
// AMBIGU : il est COMPTE, et la table en retient un par la regle de coincidence de la grammaire
// (cf. [TableDeDatums] ; jusqu au 2026-09-27 cette phrase disait qu il n y entrait pas, ce que
// le code ne faisait plus depuis la contrainte de croissance du lot 5.16.4).
//
// HORS LIGNE — jamais depuis un chemin de requete.

// TableDeDatums rend la table `slot -> typeIndex` que porte le payload d un paquet d image-cle.
//
// # DEUX CONTRAINTES, LES DEUX PRISES DE L ECRIVAIN
//
//  1. LES GARDES D EN-TETE (`kfAnchorFromID`) : slot sous le cardinal de la table, `field26` nul,
//     `ti` sous le cap objet de 50 — et les deux bits de tete non nuls, garde de la marche et non
//     regle de l ecrivain : l allocateur `FUN_142f2e598` pose aussi la valeur 0
//     (`gen = (gen+1)&3`), le seul identifiant nul est `0xffffffff`.
//  2. LA CROISSANCE DES SLOTS. `FUN_142f2e174` parcourt la table de sa vue par INDEX CROISSANT
//     (`uVar9` de 0 vers le cardinal, sous le bitmap `vue+0x58`) et serialise une entree par
//     entite vivante : les entrees du payload sont donc en SLOTS CROISSANTS. On retient la plus
//     LONGUE sous-suite de candidats croissante en slot le long du payload — ce qui elimine les
//     coincidences de 64 bits qui, isolees, passent les gardes.
//
// Sans la contrainte 2 la table est SALE, et la mesure le dit : 1 913 slots ambigus sur
// `bfecd02b` et quatre paquets de plus en DEBORDEMENT. Avec elle, la croissance mesuree par
// l ecrivain fait le tri.
//
// `ambigus` compte les SLOTS AMBIGUS : ceux dont un candidat ecarte disait AUTRE CHOSE que la
// table — un slot vu avec deux archetypes (la table en retient un, par la regle de coincidence
// ci-dessous), ou un slot que la croissance a ecarte en entier. Deux lectures identiques d un meme
// slot ne sont pas une ambiguite. Un compteur muet cacherait une table sale (J10.5, GA1-4 : il
// comptait les CANDIDATS ecartes, la ou ses lecteurs annoncent des slots).
//
// LA REGLE DE COINCIDENCE EST CELLE DE LA GRAMMAIRE : a slot egal, la generation la plus basse,
// puis le candidat le plus TOT dans le payload — l ordre de [kfCand.betterThan] a slot egal. La
// suite croissante gardait la coincidence la plus TARDIVE (J10.5, GA1-4).
func TableDeDatums(pay []byte) (table map[uint32]uint32, ambigus int) {
	return tableDesCandidats(candidatsDeDatum(pay))
}

// tableDesCandidats est [TableDeDatums] sur des candidats deja lus.
func tableDesCandidats(cands []candidatDeDatum) (table map[uint32]uint32, ambigus int) {
	garde := plusLongueSuiteCroissante(cands)
	table = make(map[uint32]uint32, len(garde))
	for _, i := range garde {
		table[cands[i].slot] = cands[i].ti
	}
	return table, slotsAmbigus(cands, table)
}

// slotsAmbigus compte les slots dont un candidat contredit la table : absent d elle, ou retenu
// avec un autre archetype.
func slotsAmbigus(cands []candidatDeDatum, table map[uint32]uint32) int {
	ambigus := map[uint32]bool{}
	for _, c := range cands {
		if ti, retenu := table[c.slot]; !retenu || ti != c.ti {
			ambigus[c.slot] = true
		}
	}
	return len(ambigus)
}

// candidatDeDatum est un en-tete de record d image-cle candidat : sa position en bits, son slot
// et son archetype.
type candidatDeDatum struct {
	bit  int
	slot uint32
	ti   uint32
	gen  int
}

// candidatsDeDatum rend, EN ORDRE DE BIT, tous les en-tetes candidats du payload : les gardes de
// `kfAnchorFromID` a chaque position, sans contrainte de croissance (elle est appliquee ensuite).
func candidatsDeDatum(pay []byte) []candidatDeDatum {
	total := len(pay) * 8
	out := make([]candidatDeDatum, 0, 1024)
	for q := 0; q+64 <= total; q++ {
		slot, ti, gen, ok := kfAnchorFromID(pay, q, source.BitsBourres(pay, q, 32), -1, total)
		if !ok {
			continue
		}
		//nolint:gosec // slot < kfTableCap et ti < kfArchMax par les gardes de kfAnchorFromID
		out = append(out, candidatDeDatum{bit: q, slot: uint32(slot), ti: uint32(ti), gen: gen})
	}
	return out
}

// plusLongueSuiteCroissante rend les INDEX de la plus longue sous-suite de `cands` strictement
// croissante en slot (les candidats sont deja en ordre de bit). Patience sorting, O(n log n).
// A slot egal, la queue en place ne cede qu a une generation plus basse : le premier candidat
// gagne, comme a l election de la marche ([kfCand.betterThan]).
func plusLongueSuiteCroissante(cands []candidatDeDatum) []int {
	if len(cands) == 0 {
		return nil
	}
	queues := make([]int, 0, len(cands)) // queues[l] = index du dernier element d une suite de l+1
	pred := make([]int, len(cands))
	for i := range cands {
		pred[i] = -1
		lo, hi := 0, len(queues)
		for lo < hi { // premiere queue dont le slot est >= celui du candidat
			mid := (lo + hi) / 2
			if cands[queues[mid]].slot < cands[i].slot {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo < len(queues) && cands[queues[lo]].slot == cands[i].slot &&
			cands[i].gen >= cands[queues[lo]].gen {
			continue // coincidence : la queue en place la precede, a generation egale ou plus basse
		}
		if lo > 0 {
			pred[i] = queues[lo-1]
		}
		if lo == len(queues) {
			queues = append(queues, i)
		} else {
			queues[lo] = i
		}
	}
	out := make([]int, len(queues))
	for k, i := len(queues)-1, queues[len(queues)-1]; k >= 0; k, i = k-1, pred[i] {
		out[k] = i
	}
	return out
}

// LierTableDeDatums pose dans `w` les liaisons de la table de datums des paquets d image-cle d un
// chunk, SANS ecraser une liaison deja posee (un record d image-cle marche ou un NEW propre dit
// davantage : il porte aussi la position et la generation). Rend le nombre de liaisons posees et
// le nombre de slots ambigus (cf. [TableDeDatums]).
//
// C est le geste qui donne au monde hors ligne le modele de la branche vive : l archetype d un
// slot que la chaine de l image-cle n a pas atteint et qu aucun NEW n a declare.
func LierTableDeDatums(w *World, data []byte, pks []FilmPacket) (posees, ambigus int) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		table, amb := TableDeDatums(pk.Payload(data))
		ambigus += amb
		posees += lierLesDatums(w, table)
	}
	return posees, ambigus
}

// lierLesDatums pose les liaisons d UNE table de datums, sans ecraser une liaison deja posee, et
// rend le nombre de liaisons posees.
func lierLesDatums(w *World, table map[uint32]uint32) int {
	posees := 0
	for slot, ti := range table {
		if _, lie := w.ArchetypeForSlot(slot); lie {
			continue
		}
		w.BindDatum(slot, ti)
		posees++
	}
	return posees
}
