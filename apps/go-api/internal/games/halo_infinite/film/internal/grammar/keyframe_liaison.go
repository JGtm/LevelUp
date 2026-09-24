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

// LiaisonDUnChunk : ce que la liaison des images-cles d un chunk a fait au monde.
type LiaisonDUnChunk struct {
	// Datums / Ambigus : les liaisons que la table de datums a posees, et ses candidats ecartes.
	Datums, Ambigus int
	// Oubliees : les liaisons retirees parce qu aucune lecture de l image-cle ne porte leur slot.
	Oubliees int
}

// lierLeChunkAuMonde pose sur le monde ce que les images-cles d un chunk declarent — en
// commencant par oublier ce qu elles ne portent plus (cf. l en-tete). Les images-cles du chunk
// se jouent dans l ordre du chunk : la suivante est l etat d un instant plus tardif.
func lierLeChunkAuMonde(w *World, marche MarcheDImageCle, data []byte, pks []FilmPacket) LiaisonDUnChunk {
	var out LiaisonDUnChunk
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		pay := pk.Payload(data)
		mp := marche.Marcher(pay)
		table, ambigus := TableDeDatums(pay)
		out.Ambigus += ambigus
		portes := make(map[uint32]bool, len(mp.Records)+len(table)+len(mp.Ecartes))
		for _, r := range mp.Records {
			portes[uint32(r.Slot)] = true //nolint:gosec // slot borne par le walker d image-cle
		}
		for _, r := range mp.Ecartes {
			portes[uint32(r.Slot)] = true //nolint:gosec // idem
		}
		for s := range table {
			portes[s] = true
		}
		out.Oubliees += w.OublierLesSlotsNonPortes(portes)
		for _, r := range mp.Records {
			//nolint:gosec // slot, TI et Gen viennent du walker d image-cle, bornes par construction
			w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
		}
		out.Datums += lierLesDatums(w, table)
	}
	return out
}
