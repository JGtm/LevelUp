package grammar

// zone_state_scan_images_cles.go — LA VOIE IMAGE-CLE DES PROPRIETES RESEAU DE ti=13.
//
// # POURQUOI UNE SECONDE VOIE
//
// Une propriete reseau n'est emise dans une trame delta qu'a son CHANGEMENT. Une valeur posee
// avant le premier paquet du film et jamais changee depuis — la base qu'une variante de Bastion
// donne a chaque camp au coup d'envoi — n'apparait donc dans AUCUNE trame delta tant qu'elle ne
// bouge pas. Seule l'image-cle, qui est le vidage de l'etat complet de chaque entite, la porte.
//
// # LE CHEMIN : UN CANAL DE LA PHASE DES IMAGES-CLES (ADR 0037 IR-3, IR-4)
//
// Le canal declare son interet pour la valeur SCALAIRE de ti=13 (`i1`, le variant en mode A) dans
// la phase des images-cles ; la distribution parcourt donc l'etat complet des records de ti=13
// sous le cadre que la production lit ([WalkKeyframeFullState]) et range chaque occurrence avec
// son etendue. Le canal relit la valeur A L ETENDUE de l'occurrence avec le deserialiseur de
// production, et exige que la relecture finisse exactement ou la marche a pose la fin de
// l'occurrence : deux lectures des memes bits par le meme lecteur doivent s'accorder, sinon rien
// n'est retenu.
//
// LES VARIANTS PAR JOUEUR (`i2..i33`) NE SONT PAS INTERPRETES : la marche les traverse (ils
// bornent le record), aucun consommateur ne lit leur etat initial.
//
// # SEULS LES RECORDS FERMES PARLENT
//
// Un record d'image-cle est FERME quand la traversee de son etat complet finit exactement sur
// l'ancre suivante ([lecture.PreuveFerme]) : c'est la seule preuve que toutes ses largeurs sont
// justes. Un record casse (composant sans lecteur), non prouve (frontiere manquee, dernier record
// du paquet) ou refuse a la relecture ne laisse AUCUNE lecture ; chaque cas se compte.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// canalDesProprietesGerees lit, dans la phase des images-cles, la valeur scalaire de chaque record
// ti=13 ferme.
type canalDesProprietesGerees struct {
	fc   *FilmContext
	arch Archetype
	sc   *ManagedPropertyScan
	// obs est l'observation de la relecture : son crochet depose la valeur dans `cur`.
	obs *Observation
	cur ManagedPropertyRead
	got bool
}

// scanKeyframeManagedProperties joue la phase des images-cles du film pour le canal de ti=13 et
// range ses lectures et ses comptes dans `sc`.
func scanKeyframeManagedProperties(fc *FilmContext, arch Archetype, sc *ManagedPropertyScan) {
	c := &canalDesProprietesGerees{fc: fc, arch: arch, sc: sc}
	c.obs = NouvelleObservation()
	c.obs.ManagedPropertyHook = func(f ManagedPropertyField, values []uint64) {
		if len(values) == 0 {
			return
		}
		c.cur.Field, c.cur.Tag = f, int(values[0])
		c.cur.Value, c.cur.HasValue = 0, false
		if len(values) > 1 {
			c.cur.Value, c.cur.HasValue = values[1], true
		}
		c.got = true
	}
	distribuerLesImagesClesSeules(fc, []Canal{c})
}

func (*canalDesProprietesGerees) Interets() []Interet {
	return []Interet{{Phase: PhaseImagesCles, TI: ManagedPropertyTypeIndex, Composant: compManagedObjectProperty}}
}

func (*canalDesProprietesGerees) Clore(BilanDeMarche) {}

// ImageCle lit les records ti=13 d'UN paquet d'image-cle et compte chacun.
func (c *canalDesProprietesGerees) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	ctx := ContexteDeLecture{Profil: c.fc.ProfilDeBalayage(), Obs: c.obs}
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != ManagedPropertyTypeIndex || r.Desync == lecture.CorpsNonParcouru {
			continue
		}
		c.sc.KeyRecords++
		switch {
		case r.Desync >= 0:
			c.sc.KeyBroken++
		case r.Preuve != lecture.PreuveFerme:
			c.sc.KeyUnproven++
		default:
			reads, ok := c.relire(p, r, ctx)
			if !ok {
				c.sc.KeyRefused++
				continue
			}
			c.sc.KeyClosed++
			c.sc.KeyReads = append(c.sc.KeyReads, reads...)
		}
	}
}

// relire relit, a son etendue, chaque occurrence interpretee du record ferme `r` et rend ses
// lectures ; faux quand une relecture ne finit pas ou la marche a pose la fin de l'occurrence.
//
// UNE LECTURE D'IMAGE-CLE EST CHAINEE PAR CONSTRUCTION : son record finit exactement sur l'ancre
// du suivant, ce qui est plus que le temoin de chainage d'une lecture delta.
func (c *canalDesProprietesGerees) relire(p *lecture.Paquet, r *lecture.Record, ctx ContexteDeLecture) (
	[]ManagedPropertyRead, bool,
) {
	var out []ManagedPropertyRead
	for _, oc := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if oc.Etat != lecture.EtatInterprete {
			continue
		}
		idx := int(oc.Index)
		br := LecteurSur(p.Payload)
		br.PoserContexte(ctx)
		br.SetBitPos(int(oc.Debut))
		c.got = false
		_, _, porte := consumeByName(br, c.arch.component(idx), ManagedPropertyTypeIndex, c.arch.Level(idx))
		if !porte {
			return nil, false
		}
		consumeCorruptionCheck(br)
		if br.BitPos() != int(oc.Debut+oc.Bits) {
			return nil, false
		}
		if c.got {
			c.cur.Slot, c.cur.TimestampUS = r.Vie.Slot, p.TS
			c.cur.FilmIndex, c.cur.Chained = ManagedPropertyFilmIndex(idx), true
			out = append(out, c.cur)
		}
	}
	return out, true
}
