package grammar

// keyframe_liaison.go — CE QU UNE IMAGE-CLE FAIT AU MONDE D UNE MARCHE DE TRAMES (lot D-fix des
// retours du rejeu, 2026-09-24).
//
// # L IMAGE-CLE EST L ETAT COMPLET, ET ELLE DIT AUSSI QUI N EST PLUS LA
//
// `FUN_142f2e174` parcourt la table de sa vue et rend UN MOT PAR ENTITE VIVANTE (cf.
// `keyframe_datums.go`) : une image-cle porte toutes les entites vivantes a son instant, et
// seulement elles. Les deux marches de trames qui lient le monde depuis les images-cles (etats
// de mouvement, dotations de naissance) ne lisaient d elle que ce qu elle PORTE ; une liaison
// posee par le flux (record NEW propre, [World.BindFull]) ne tombait que sur un record DEL LU.
// Qu un DEL ne soit pas lu — paquet desynchronise avant lui, paquet a evenements non localise —
// et la liaison du mort restait : le prochain occupant du slot se lisait sous l archetype du
// mort, et ses composants ne se decodaient plus jusqu a la premiere image-cle qui le porte.
//
// MESURE (`a0c36016`, lot D-fix) : un objet `ti 30` ne au chunk 27 sur le slot 649, lie par son
// NEW ; son DEL n est pas lu ; le bipede ne sur ce slot au chunk 38 s y decode sous `ti 30` et
// perd ses etats de mouvement jusqu a l image-cle du chunk 39 — cinq vies (649 a 653), vingt-
// quatre intervalles. La marche de M3 le revelait : parce qu elle lie la chaine ENTIERE des
// images-cles, les paquets delta vont plus loin et atteignent le NEW du mort, que la marche
// tronquee n atteignait pas.
//
// # LA REGLE
//
// A chaque image-cle, AVANT ses liaisons, une liaison du monde dont le slot n est porte ni par
// la chaine de ses records, ni par sa table de datums, ni par un candidat que la marche a ECARTE
// (un record possiblement perdu : son absence ne prouve rien, principe du lot D-fix) est
// OUBLIEE. Aucun seuil, aucune fenetre : c est la grammaire de l ecrivain. Un vivant oublie a
// tort (absent des trois lectures) se relie par l anticipation a son prochain delta — la
// premiere image-cle ulterieure qui le declare. Le compte est rendu ([LiaisonDUnChunk.Oubliees]).

// # LE VERDICT DES NEW REFUSES (constat DFIX-R6 de la revue adverse, 2026-09-24)
//
// Un NEW qui contredit une entite vivante est refuse ([contreditUneEntiteVivante]) : c est une
// lecture fausse, SAUF quand la croyance du monde est perimee — le DEL du vivant n a pas ete lu, et
// le NEW est une vraie creation, perdue jusqu a l image-cle suivante. Les deux cas ne se separent
// pas au decodage ; l image-cle suivante les separe : elle redonne au slot l archetype du vivant
// (lecture fausse confirmee), celui du NEW (creation perdue), ou aucun des deux (indecis). Chaque
// refus recoit son verdict a la PREMIERE image-cle du chunk suivant, avant l oubli et les liaisons.

import (
	"maps"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// neufRefuse : un NEW refuse, en attente du verdict de l image-cle suivante.
type neufRefuse struct {
	slot, neuf, vivant uint32
}

// jugerLesNeufsRefuses rend le verdict des refus en attente contre les archetypes que l image-cle
// suivante declare (`declares` : chaine de ses records, puis table de datums).
func (o *Observation) jugerLesNeufsRefuses(declares map[uint32]uint32) {
	if o == nil {
		return
	}
	for _, n := range o.neufsRefuses {
		ti, porte := declares[n.slot]
		switch {
		case porte && ti == n.vivant:
			o.NeufsRefusesLecturesFausses++
		case porte && ti == n.neuf:
			o.NeufsRefusesCreationsPerdues++
		default:
			o.NeufsRefusesIndecis++
		}
	}
	o.neufsRefuses = o.neufsRefuses[:0]
}

// solderLesNeufsRefuses classe INDECIS les refus qu aucune image-cle n a suivis (fin du film).
func (o *Observation) solderLesNeufsRefuses() {
	if o == nil {
		return
	}
	o.NeufsRefusesIndecis += len(o.neufsRefuses)
	o.neufsRefuses = o.neufsRefuses[:0]
}

// LiaisonDUnChunk : ce que la liaison des images-cles d un chunk a fait au monde.
type LiaisonDUnChunk struct {
	// Datums / Ambigus : les liaisons que la table de datums a posees, et ses candidats ecartes.
	Datums, Ambigus int
	// Oubliees : les liaisons retirees parce qu aucune lecture de l image-cle ne porte leur slot.
	Oubliees int
}

// declarationDImageCle est ce qu UNE image-cle declare au monde : la chaine de ses records et les
// candidats que la marche d ancres a ecartes ([MarcheDePayload]), et sa table de datums, lue a
// position libre ([TableDeDatums] : la recuperation que la liaison [lecture.LiaisonDatum] marque).
type declarationDImageCle struct {
	ancres  MarcheDePayload
	table   map[uint32]uint32
	ambigus int
}

// declarerLImageCle lit ce que l image-cle de payload `pay`, de marche d ancres `mp`, declare.
func declarerLImageCle(pay []byte, mp MarcheDePayload) declarationDImageCle {
	table, ambigus := TableDeDatums(pay)
	return declarationDImageCle{ancres: mp, table: table, ambigus: ambigus}
}

// liaisonDesImagesCles est la liaison des images-cles au monde, LUE dans la phase des images-cles
// (un preliminaire de la marche des trames, `marche_trames_preliminaires.go`) et POSEE chunk par
// chunk pendant la marche des trames ([lierLesImagesClesDuChunk]).
type liaisonDesImagesCles struct {
	// parChunk : les declarations des images-cles de chaque chunk, dans l ordre du chunk.
	parChunk map[int][]declarationDImageCle
}

// recevoir garde ce que l image-cle `p`, de marche d ancres `mp`, declare.
func (l *liaisonDesImagesCles) recevoir(p *lecture.Paquet, mp MarcheDePayload) {
	l.parChunk[p.Chunk] = append(l.parChunk[p.Chunk], declarerLImageCle(p.Payload, mp))
}

// rendre rend les declarations du chunk `c` et les oublie : un chunk se lie une fois.
func (l *liaisonDesImagesCles) rendre(c int) []declarationDImageCle {
	d := l.parChunk[c]
	delete(l.parChunk, c)
	return d
}

// lierLesImagesClesDuChunk pose sur le monde ce que les images-cles d un chunk declarent — en
// commencant par oublier ce qu elles ne portent plus (cf. l en-tete). Les images-cles du chunk
// se jouent dans l ordre du chunk : la suivante est l etat d un instant plus tardif.
func lierLesImagesClesDuChunk(w *World, decls []declarationDImageCle, obs *Observation) LiaisonDUnChunk {
	var out LiaisonDUnChunk
	for i, d := range decls {
		mp := d.ancres
		out.Ambigus += d.ambigus
		portes := make(map[uint32]bool, len(mp.Records)+len(d.table)+len(mp.Ecartes))
		for _, r := range mp.Records {
			portes[uint32(r.Slot)] = true //nolint:gosec // slot borne par le walker d image-cle
		}
		for _, r := range mp.Ecartes {
			portes[uint32(r.Slot)] = true //nolint:gosec // idem
		}
		for s := range d.table {
			portes[s] = true
		}
		if i == 0 {
			obs.jugerLesNeufsRefuses(archetypesDeclares(mp.Records, d.table))
		}
		out.Oubliees += w.OublierLesSlotsNonPortes(portes)
		for _, r := range mp.Records {
			//nolint:gosec // slot, TI et Gen viennent du walker d image-cle, bornes par construction
			w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
		out.Datums += lierLesDatums(w, d.table)
	}
	return out
}

// archetypesDeclares rend l archetype que l image-cle donne a chaque slot : la chaine de ses records
// d abord, la table de datums pour les slots que la chaine n a pas atteints.
func archetypesDeclares(recs []KeyframeRec, table map[uint32]uint32) map[uint32]uint32 {
	out := make(map[uint32]uint32, len(recs)+len(table))
	maps.Copy(out, table)
	for _, r := range recs {
		out[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // slot et TI bornes par le walker
	}
	return out
}
