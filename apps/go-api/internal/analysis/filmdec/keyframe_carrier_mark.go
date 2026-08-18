package filmdec

import "fmt"

// keyframe_carrier_mark.go — LE MARQUEUR DE PORTAGE, lu dans le record de bipede des
// images-cles.
//
// CE QUE C'EST, ET SURTOUT CE QUE CE N'EST PAS. C'est une suite de 32 bits qui apparait dans
// l'emprise du record d'image-cle d'un joueur QUAND il porte le drapeau, et pratiquement
// jamais sinon. Ce n'est PAS un identifiant d'arme : il ne porte pas le suffixe d'identifiant
// `weap` (0 fois sur 83 occurrences), il n'a donc aucun nom de tag, et le canal des armes
// tenues des paquets delta ne le voit jamais (0 sur 68 284 lectures). Il ne NOMME rien : il
// DIT que ce joueur-la porte quelque chose, a cet instant-la.
//
// LA MESURE QUI L'ETABLIT (plan `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`,
// phase 0, item 0.1 — balayage SANS predicat de toutes les fenetres de 32 bits de l'emprise,
// confronte aux fenetres de portage de l'oracle CTF nomme) :
//
//	37 / 38 fenetres de portage le portent          97,4 %   (seuil ecrit >= 90 %)
//	temoin (un slot NON porteur, memes instants)     2,6 %   (seuil ecrit <= 5 %)
//
// sur trois films CTF (`64e8adfa`, `530820e5`, `53ce4390`). Controle POSITIF de l'emprise
// balayee : 74 a 78 % des records portent au moins une famille d'arme CONNUE. Controle croise :
// aucune famille d'arme connue ne separe le portage (MA40 87,5 % en portage contre 60,4 % hors).
//
// LES QUATRE VALEURS SONT UN SEUL MOTIF. Le balayage rend 0x00010005, 0x0002000B, 0x00040017 et
// 0x0008002F : c'est la MEME suite de bits lue a quatre decalages (le champ n'est pas aligne sur
// un octet, et une fenetre glissante de 32 bits la voit quatre fois). Les tester toutes les
// quatre est donc la lecture, pas quatre lectures.
//
// CE MARQUEUR N'EST PAS UNIVERSEL ENTRE MODES : il est TOTALEMENT ABSENT du film Oddball mesure
// (0 porteur sur 26 images-cles). Il sert de CONTROLE au portage de drapeau publie par le rejeu,
// jamais de source pour un autre objet.

// carrierMarkViews — les quatre vues decalees du marqueur de portage (cf. en-tete).
var carrierMarkViews = map[uint32]bool{
	0x00010005: true,
	0x0002000B: true,
	0x00040017: true,
	0x0008002F: true,
}

// CarrierMark est UN record de bipede d'image-cle qui porte le marqueur.
type CarrierMark struct {
	// TimestampUS est l'horodatage du paquet d'image-cle — MEME horloge que
	// BipedPosition.TimestampUS.
	TimestampUS uint64
	// Slot est le slot du biped porteur (celui des trajectoires).
	Slot uint32
}

// CarrierMarkScan porte les marques ET le denominateur qui permet de les juger : sans les
// instants d'image-cle, « 3 portages confirmes » ne dit pas si les autres ont ete observes.
type CarrierMarkScan struct {
	// Marks : les records de bipede portant le marqueur, dans l'ordre du film.
	Marks []CarrierMark
	// KeyframeUS : l'horodatage de CHAQUE image-cle balayee, y compris celles sans marque.
	KeyframeUS []uint64
	// Records / BipedRecords : records d'image-cle balayes, et ceux d'archetype biped.
	Records, BipedRecords int
}

// ScanFilmCarrierMarks balaye les images-cles du film de dir et rend les records de bipede
// portant le marqueur de portage.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.
func ScanFilmCarrierMarks(dir string) (CarrierMarkScan, error) {
	n := CountFilmChunks(dir)
	var out CarrierMarkScan
	read := 0
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		read++
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			out.KeyframeUS = append(out.KeyframeUS, p.TimestampUS)
			out.appendMarksOf(p.Payload(chunk), p.TimestampUS)
		}
	}
	if read == 0 {
		return CarrierMarkScan{}, fmt.Errorf("aucun chunk film lisible dans %s", dir)
	}
	return out, nil
}

// appendMarksOf balaye UN payload d'image-cle. Le balayage est celui des armes portees
// (`familiesByRecord`) : meme fenetre glissante de 32 bits, meme attribution au record qui
// contient le PREMIER bit de la fenetre. Seul le jeu de valeurs cherchees change — c'est
// pourquoi ce fichier n'a pas son propre lecteur de bits.
func (s *CarrierMarkScan) appendMarksOf(pay []byte, ts uint64) {
	recs := WalkKeyframeWorld(pay)
	s.Records += len(recs)
	for _, r := range recs {
		if r.TI == keyframeBipedTI {
			s.BipedRecords++
		}
	}
	for _, r := range familiesByRecord(pay, carrierMarkViews, keyframeBipedTI) {
		if r.Rec.Slot < 0 {
			continue
		}
		s.Marks = append(s.Marks, CarrierMark{TimestampUS: ts, Slot: uint32(r.Rec.Slot)})
	}
}
