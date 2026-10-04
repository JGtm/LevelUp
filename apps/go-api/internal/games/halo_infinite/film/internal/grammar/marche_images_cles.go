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
// # UN CORPS N EST PARCOURU QUE S IL EST LU
//
// L iterateur [FilmContext.ImagesCles] parcourt l etat complet de chaque record (sa mesure de
// fermeture les veut tous). Une distribution ([Distribuer]) ne parcourt que les corps des
// archetypes qu un de ses canaux interprete : les autres records gardent leur identite, leur ancre
// et leur liaison, sans composant ni preuve ([lecture.CorpsNonParcouru]). C est ce qui laisse a un
// canal qui ne lit que des identites le cout de la seule marche d ancres, et ce qui lui permet de
// lire un film sans registre : la marche d ancres n en a pas besoin, la traversee d un corps si.
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
// d entites du paquet n est pas renseignee. Sans canal, aucune occurrence n est interpretee
// ([Distribuer] declare les interets).
//
// Le decoupage MPP du format du film ([EnTete.MPP]) est pose sur le contexte pour la duree de
// l iteration, comme pour la mesure de fermeture, et restaure a la sortie, arret anticipe compris.
// Le paquet rendu n est valide que pendant le tour qui le rend.
func (c *FilmContext) ImagesCles() iter.Seq2[*lecture.Paquet, error] {
	return func(rendre func(*lecture.Paquet, error) bool) {
		reg, err := c.Registry()
		if err != nil {
			rendre(nil, err)
			return
		}
		m, restaurer := c.nouvelleMarcheDesImagesCles(demandeDImagesCles{reg: reg}, true)
		defer restaurer()
		m.parcourir(func(p *lecture.Paquet) bool { return rendre(p, nil) })
	}
}

// marcheDesImagesCles porte l etat d UNE marche des images-cles d un film : le contexte, le
// registre (nil quand aucun corps n est a parcourir et que le film n en a pas), le contexte de
// lecture, la marche d ancres du film, les interets des canaux, l arene du paquet en cours et la
// marche d ancres de ce paquet.
type marcheDesImagesCles struct {
	fc       *FilmContext
	reg      *Registry
	ctx      ContexteDeLecture
	marche   MarcheDImageCle
	interets interetsResolus
	// tousLesCorps : la marche parcourt l etat complet de chaque record ([FilmContext.ImagesCles]),
	// et pas seulement ceux des archetypes interpretes.
	tousLesCorps bool
	// chunks : les numeros des chunks marches, dans l ordre ; nil = ceux du film
	// ([FilmContext.ChunkNumbers]).
	chunks []int
	paquet lecture.Paquet
	ancres MarcheDePayload
}

// nouvelleMarcheDesImagesCles prepare la phase des images-cles selon ce que la distribution demande
// (registre, interets, chunks). Quand un corps peut etre parcouru, elle pose le decoupage MPP du
// format ([EnTete.MPP]) sur le contexte et prend son contexte de lecture ; elle rend la
// restauration du contexte.
func (c *FilmContext) nouvelleMarcheDesImagesCles(d demandeDImagesCles, tousLesCorps bool) (
	*marcheDesImagesCles, func(),
) {
	m := &marcheDesImagesCles{fc: c, reg: d.reg, interets: d.interets, tousLesCorps: tousLesCorps, chunks: d.chunks}
	if d.marche != nil {
		m.marche = *d.marche
	} else {
		m.marche = c.MarcheDImageCle()
	}
	restaurer := func() {}
	if m.reg == nil || (!tousLesCorps && len(m.interets) == 0) {
		return m, restaurer
	}
	if mpp := c.EnTete().MPP; mpp.Provenance != lecture.ProvenanceNonRenseignee {
		prev := c.PoserMPP(mpp.Valeur)
		restaurer = func() { c.PoserMPP(prev) }
	}
	m.ctx = c.ContexteDeLecture()
	return m, restaurer
}

// parcourt dit si la marche parcourt l etat complet des records de l archetype `ti`.
func (m *marcheDesImagesCles) parcourt(ti int) bool {
	return m.reg != nil && (m.tousLesCorps || m.interets.parcourt(ti))
}

// parcourir range chaque paquet d image-cle des chunks de la marche (ceux du film quand
// [marcheDesImagesCles.chunks] est nil), dans l ordre du flux, et le rend a `rendre` ; faux arrete
// la marche. Elle rend le nombre de chunks qu elle a pu lire.
func (m *marcheDesImagesCles) parcourir(rendre func(*lecture.Paquet) bool) int {
	chunks := m.chunks
	if chunks == nil {
		chunks = m.fc.ChunkNumbers()
	}
	lus := 0
	for _, num := range chunks {
		data, pks, ok := m.fc.ChunkAt(num)
		if !ok {
			continue
		}
		lus++
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			m.marcherLePaquet(num, pk, data)
			if !rendre(&m.paquet) {
				return lus
			}
		}
	}
	return lus
}

// marcherLePaquet range UN paquet d image-cle dans l arene : ses ancres, dans l ordre des bits, et
// l etat complet de chacune dont le corps est a parcourir.
func (m *marcheDesImagesCles) marcherLePaquet(chunk int, pk FilmPacket, data []byte) {
	p := &m.paquet
	viderLePaquet(p)
	p.Chunk, p.Index, p.Type, p.TS, p.Payload = chunk, pk.Index, pk.Type, pk.TimestampUS, pk.Payload(data)
	m.ancres = m.marche.Marcher(p.Payload)
	recs := m.ancres.Records
	// Le tri par bit ne deplace rien : la marche avance strictement (cf. [keyframeBornesDe]), et
	// l ordre des ancres reste celui de la marche. recs[i] est l ancre de b.
	for i, b := range keyframeBornesDe(recs) {
		tete := uint32(recs[i].Gen)     //nolint:gosec // deux bits
		premier := uint32(len(p.Comps)) //nolint:gosec // l arene d un paquet tient sur 32 bits
		r := lecture.Record{
			Genre: lecture.GenreEtatComplet, Vue: uint8(tete), Liaison: liaisonDeLAncre(recs[i]),
			Preuve: lecture.PreuveNonProuve, TI: int16(b.TI), Desync: lecture.CorpsNonParcouru, //nolint:gosec // archetype < 50
			Vie:   types.LifeKey{Slot: uint32(b.Slot), Gen: tete},    //nolint:gosec // slot < 8192
			Debut: uint32(b.Bit), Comps: [2]uint32{premier, premier}, //nolint:gosec // position d un payload
		}
		if m.parcourt(b.TI) {
			m.parcourirLeCorps(&r, b)
		}
		p.Records = append(p.Records, r)
	}
}

// parcourirLeCorps traverse l etat complet du record `r`, d ancre `b`, et range ses composants.
func (m *marcheDesImagesCles) parcourirLeCorps(r *lecture.Record, b keyframeBorne) {
	p := &m.paquet
	tr := WalkKeyframeFullState(p.Payload, b.Bit, m.reg, m.ctx)
	for k := range tr.Comps {
		p.Comps = append(p.Comps, composantLu(&tr, k, m.interets.contient(b.TI, tr.Comps[k].Index)))
	}
	r.Preuve, r.Desync, r.Masque = preuveDeLEtatComplet(tr, b), int16(tr.DesyncAt), tr.Mask //nolint:gosec // index de composant < 64
	r.Bits = uint32(tr.EndBit - b.Bit)                                                      //nolint:gosec // positions d un payload
	r.Comps[1] = uint32(len(p.Comps))                                                       //nolint:gosec // l arene d un paquet tient sur 32 bits
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
