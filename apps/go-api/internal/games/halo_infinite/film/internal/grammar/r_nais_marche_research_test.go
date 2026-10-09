//go:build research

package grammar

// r_nais_marche_research_test.go — CHANTIER « NAIS » (R-L1 (a), (b), (d) et R-P6, 2026-10-02) :
// la MARCHE et le JUGE du chantier. Un instrument de recherche : aucun fichier de production n est
// touche, aucune sortie ne change.
//
//	rnMarcher  la marche de la campagne ([cmMarcher]), avec un crochet AVANT chaque paquet delta,
//	           qui recoit le monde tel que le paquet le trouvera (pour juger une occurrence d en-tete
//	           NEW sur le monde d avant son paquet, puis le restaurer) ;
//	rnJuge     le juge des invariants de l ecrivain ([cmContredit]) : paquets fermes, fermes sains,
//	           records utiles fermes dans des paquets sains, et, contre la reference, paquets gagnes
//	           (dont contredits) et perdus (dont sains en reference).

import (
	"fmt"
	"strings"
)

// rnAvant est le crochet d avant paquet : `j` est le rang du paquet delta dans son chunk (celui
// que [cmCollecteur] donne a ses paquets).
type rnAvant func(c, j int, pk FilmPacket, pay []byte, w *World)

// rnMarcher est [cmMarcher] avec le crochet d avant paquet ; la valeur de retour est la meme.
func rnMarcher(f *cmFilm, v cmVariante, e cmEcouteur, avant rnAvant) (FrameClosureReport, *Observation) {
	md := marcheDetaillee{mesureDesTrames: nouvelleMesureDesTrames(f.reg, f.utiles, f.cfg)}
	obs := md.cfg.Obs
	monde := NewWorld(f.reg)
	monde.PoserTableAnticipee(ConstruireTableAnticipee(f.fc))
	marche := f.fc.MarcheDImageCle()
	poser := func(cle [2]int) {
		for _, l := range v.oracle[cle] {
			monde.BindFull(l.eid, l.ti)
		}
	}
	for _, c := range f.fc.ChunkNumbers() {
		data, pks, ok := f.fc.ChunkAt(c)
		if !ok {
			continue
		}
		monde.PoserChunkCourant(c)
		lierLeChunkAuMonde(monde, marche, data, pks, obs)
		if e != nil {
			e.debutDeChunk(c, data, pks, monde)
		}
		poser([2]int{c, -1})
		tete := debutDeLaListeSous(f.fc.grammaireDeLaVueA()) // la cuisson : la fin de la vue A, puis le localisateur
		if v.tete != nil {
			tete = v.tete(c)
		}
		j := 0
		for _, pk := range pks {
			if pk.Type == PacketTypeDelta && pk.Size >= 1 && avant != nil {
				avant(c, j, pk, pk.Payload(data), monde)
			}
			p := cmPaquetDe(&md, tete, c, pk, data, monde)
			if p == nil {
				continue
			}
			j++
			if e != nil {
				e.paquet(c, p, monde)
			}
			poser([2]int{c, pk.Index})
		}
	}
	obs.solderLesNeufsRefuses()
	if e != nil {
		e.finDeFilm()
	}
	return md.rapport(), obs
}

// rnJuge juge une marche ; `ref` nil : la marche jugee EST la reference (elle remplit refFerme et
// refSain, que les autres juges lisent).
type rnJuge struct {
	chk                         *cmCollecteur
	ref                         *rnJuge
	refFerme, refSain           map[[2]int]bool
	fermes, sains, utilesSains  int
	gagnes, gagnesContre        int
	perdus, perdusSains         int
	gagnesSains, gagnesSainsUti int
}

func rnNouveauJuge(f *cmFilm, b *cmBlocs, ref *rnJuge) *rnJuge {
	return &rnJuge{chk: nouveauCollecteur(f, b, nil), ref: ref, refFerme: map[[2]int]bool{}, refSain: map[[2]int]bool{}}
}

func (j *rnJuge) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) { j.chk.chunk = c }

func (j *rnJuge) paquet(_ int, p *cmPaquet, _ *World) {
	cle := [2]int{p.d.Chunk, p.d.Index}
	sain := false
	if p.d.Fermee {
		j.fermes++
		sain = len(cmContredit(j.chk, p)) == 0
		if sain {
			j.sains++
			j.utilesSains += p.utilesFermes
		}
	}
	if j.ref == nil {
		j.refFerme[cle], j.refSain[cle] = p.d.Fermee, sain
		return
	}
	avant := j.ref.refFerme[cle]
	switch {
	case p.d.Fermee && !avant:
		j.gagnes++
		if !sain {
			j.gagnesContre++
		} else {
			j.gagnesSains++
			j.gagnesSainsUti += p.utilesFermes
		}
	case !p.d.Fermee && avant:
		j.perdus++
		if j.ref.refSain[cle] {
			j.perdusSains++
		}
	}
}

func (j *rnJuge) finDeFilm() {}

// rnTeteVariante : l en-tete des lignes de variantes.
const rnTeteVariante = "film\tbuild\tvariante\tliaisons\tpaquets\tfermes\tfermes_sains\tutiles_fermes\t" +
	"utiles_sains_fermes\tutiles_lus\thors_cadre\tgagnes\tgagnes_contredits\tgagnes_sains\tperdus\tperdus_sains"

// rnLigneVariante met une marche jugee en ligne.
func rnLigneVariante(f *cmFilm, nom string, nl int, r FrameClosureReport, j *rnJuge) string {
	return rnTab(f.id, f.build, nom, nl, r.Paquets, r.PaquetsFermes, j.sains, r.Utiles.RecordsFermes,
		j.utilesSains, r.Utiles.Records, r.Bloquants[CauseHorsCadre].Paquets, j.gagnes, j.gagnesContre,
		j.gagnesSains, j.perdus, j.perdusSains)
}

// rnTab assemble une ligne TSV.
func rnTab(v ...any) string {
	s := make([]string, len(v))
	for i, x := range v {
		s[i] = fmt.Sprint(x)
	}
	return strings.Join(s, "\t")
}
