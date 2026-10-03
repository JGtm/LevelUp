package grammar

// marche_images_cles.go — LA PHASE IMAGES-CLES DE LA REPRESENTATION INTERMEDIAIRE (ADR 0037 IR-2,
// IR-3).
//
// # LA MARCHE DE PRODUCTION, PAS UNE SECONDE MARCHE
//
// Les records d un paquet d image-cle sont les ancres de la marche du film
// ([FilmContext.MarcheDImageCle] : la memoire du contexte rend un payload deja marche sans le
// remarcher) ; l etat complet de chacun est traverse sous le cadre que la production lit
// ([WalkKeyframeFullState]) ; la frontiere d un record est l ancre qui le suit
// ([keyframeBornesDe]). Rien n est relu ni recopie. La carte de fermeture des images-cles
// ([KeyframeClosure]) consomme cette phase.
//
// # LA RECUPERATION EST MARQUEE
//
// Un record dont l ancre a ete ELUE par le repli nomme `repli_ancre_d_image_cle_par_election`
// porte la liaison [lecture.LiaisonImageCleElue] ; une ancre atteinte de proche en proche (voisin,
// saut de largeur, recalage sur l en-tete exact d un bipede), [lecture.LiaisonImageCle]
// (ADR 0037 IR-6).
//
// # LA PREUVE D UN RECORD D IMAGE-CLE
//
// Un record est FERME quand la traversee de son etat complet finit exactement sur l ancre
// suivante : la seule preuve qu on ait que toutes ses largeurs sont justes (`keyframe_closure.go`).
// Sinon il est NON PROUVE — un composant sans lecteur l arrete, la traversee manque la frontiere,
// ou c est le dernier record du paquet, qui n a pas de frontiere. Un paquet d image-cle ne porte
// pas de verdict ([lecture.VerdictNonRendu]) : chacun de ses records porte sa preuve.
//
// # DUREE DE VIE
//
// Le paquet rendu est l arene de la marche : il n est valide que pendant le tour qui le rend.

import (
	"iter"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ImagesCles rend la PHASE IMAGES-CLES du film : chaque paquet d image-cle, dans l ordre du flux,
// range dans la structure de lecture — un record d etat complet par ancre, ses composants, son
// etendue et sa preuve. Les crochets qui interpretent pendant la traversee sont ceux de
// l observation du contexte ([FilmContext.Observation]). La phase ne tient pas de monde : la table
// d entites du paquet n est pas renseignee.
//
// Le decoupage MPP du format du film est pose sur le contexte pour la duree de l iteration
// ([InstallFilmFormatMPP]), comme pour la mesure de fermeture, et restaure a la sortie, arret
// anticipe compris. Le paquet rendu n est valide que pendant le tour qui le rend.
func (c *FilmContext) ImagesCles() iter.Seq2[*lecture.Paquet, error] {
	return func(rendre func(*lecture.Paquet, error) bool) {
		reg, err := c.Registry()
		if err != nil {
			rendre(nil, err)
			return
		}
		if restaurer, err := InstallFilmFormatMPP(c); err == nil {
			defer restaurer()
		}
		m := marcheDesImagesCles{reg: reg, ctx: c.ContexteDeLecture(), marche: c.MarcheDImageCle()}
		for _, num := range c.ChunkNumbers() {
			data, pks, ok := c.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				m.marcherLePaquet(num, pk, data)
				if !rendre(&m.paquet, nil) {
					return
				}
			}
		}
	}
}

// marcheDesImagesCles porte l etat d UNE marche des images-cles d un film : le registre, le
// contexte de lecture, la marche d ancres du film et l arene du paquet en cours.
type marcheDesImagesCles struct {
	reg    *Registry
	ctx    ContexteDeLecture
	marche MarcheDImageCle
	paquet lecture.Paquet
}

// marcherLePaquet range UN paquet d image-cle dans l arene : ses ancres, dans l ordre des bits, et
// l etat complet de chacune.
func (m *marcheDesImagesCles) marcherLePaquet(chunk int, pk FilmPacket, data []byte) {
	p := &m.paquet
	viderLePaquet(p)
	p.Chunk, p.Index, p.Type, p.TS, p.Payload = chunk, pk.Index, pk.Type, pk.TimestampUS, pk.Payload(data)
	recs := m.marche.Records(p.Payload)
	for i, b := range keyframeBornesDe(recs) { // trie `recs` par bit : recs[i] est l ancre de b
		tr := WalkKeyframeFullState(p.Payload, b.Bit, m.reg, m.ctx)
		premier := uint32(len(p.Comps)) //nolint:gosec // l arene d un paquet tient sur 32 bits
		for k := range tr.Comps {
			p.Comps = append(p.Comps, composantLu(&tr, k))
		}
		tete := uint32(recs[i].Gen) //nolint:gosec // deux bits
		p.Records = append(p.Records, lecture.Record{
			Genre: lecture.GenreEtatComplet, Vue: uint8(tete), Liaison: liaisonDeLAncre(recs[i]),
			Preuve: preuveDeLEtatComplet(tr, b), TI: int16(b.TI), Desync: int16(tr.DesyncAt), //nolint:gosec // archetype < 50, index de composant < 64
			Vie:   types.LifeKey{Slot: uint32(b.Slot), Gen: tete}, //nolint:gosec // slot < 8192
			Debut: uint32(b.Bit), Bits: uint32(tr.EndBit - b.Bit), //nolint:gosec // positions d un payload
			Masque: tr.Mask,
			Comps:  [2]uint32{premier, uint32(len(p.Comps))}, //nolint:gosec // idem
		})
	}
}

// liaisonDeLAncre rend la provenance de l identite d un record d image-cle : son ancre atteinte de
// proche en proche, ou ELUE par le repli nomme `repli_ancre_d_image_cle_par_election` — la
// recuperation que la structure marque (ADR 0037 IR-6).
func liaisonDeLAncre(r KeyframeRec) lecture.Liaison {
	if r.Elue {
		return lecture.LiaisonImageCleElue
	}
	return lecture.LiaisonImageCle
}

// preuveDeLEtatComplet rend la preuve d un record d image-cle : ferme quand la traversee de son
// etat complet finit exactement sur l ancre suivante, non prouve sinon.
func preuveDeLEtatComplet(tr EntityTrace, b keyframeBorne) lecture.Preuve {
	if b.Want >= 0 && tr.DesyncAt < 0 && tr.EndBit == b.Want {
		return lecture.PreuveFerme
	}
	return lecture.PreuveNonProuve
}
