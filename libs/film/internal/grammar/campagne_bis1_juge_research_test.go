//go:build research

package grammar

// campagne_bis1_juge_research_test.go — compagnon de `campagne_bis1_research_test.go` (MESURES
// BIS 1 DE LA CAMPAGNE DE GRAMMAIRE, 2026-10-01) : les rangs de rejet, le diffuseur de marche, le
// juge des invariants de l ecrivain et le denominateur des entrees de controle (item 1.3).
// Deplacement pur, decoupe pour le seuil de 500 lignes par fichier ; aucun comportement change.

import (
	"fmt"
	"strings"
)

// cmRangDeRejet : un eid rejete tel que la sonde M1 l a range.
type cmRangDeRejet struct {
	region, distance, desync string
	lie                      bool
	compte                   cmCompte
}

// lierParRegion range une liaison d oracle sous la region de sa naissance trouvee.
func (c *cmCollecteur) lierParRegion(reg string, cle [2]int, l cmLiaison) {
	if c.parRegion == nil {
		c.parRegion = map[string]map[[2]int][]cmLiaison{}
	}
	if c.parRegion[reg] == nil {
		c.parRegion[reg] = map[[2]int][]cmLiaison{}
	}
	c.parRegion[reg][cle] = append(c.parRegion[reg][cle], l)
}

// noterRang garde la region, la distance et le NEW desynchronise anterieur d un eid rejete.
func (c *cmCollecteur) noterRang(r *cmRejet, best *cmTrouve, reg string, ok bool) {
	x := cmRangDeRejet{region: reg, distance: "sans naissance trouvee", desync: "aucun NEW desynchronise anterieur",
		compte: r.compte}
	x.compte.n = 1
	if best != nil {
		x.distance = classeDeDistance(r.premier - best.j)
		x.lie = ok && r.premier-best.j <= 3
	}
	if r.desync != "" {
		x.desync = r.desync
	}
	c.rangs = append(c.rangs, x)
}

// cmMux diffuse la marche a plusieurs ecouteurs.
type cmMux []cmEcouteur

func (m cmMux) debutDeChunk(c int, data []byte, pks []FilmPacket, w *World) {
	for _, e := range m {
		e.debutDeChunk(c, data, pks, w)
	}
}

func (m cmMux) paquet(c int, p *cmPaquet, w *World) {
	for _, e := range m {
		e.paquet(c, p, w)
	}
}

func (m cmMux) finDeFilm() {
	for _, e := range m {
		e.finDeFilm()
	}
}

// cmContredit rend les invariants de l ecrivain qu un paquet contredit (ordre, masque, vue C),
// par le collecteur prive `chk` (ses tables ne sont pas publiees).
func cmContredit(chk *cmCollecteur, p *cmPaquet) []string {
	if p.d.DebutVueB < 0 {
		return nil
	}
	chk.contre = chk.contre[:0]
	chk.ordre(p)
	chk.masques(p)
	chk.vueC(p)
	return append([]string(nil), chk.contre...)
}

// cmJuge juge une marche contre la reference : paquets gagnes / perdus / conserves, et les
// invariants de l ecrivain sur chaque paquet ferme. `ref` nil : la marche jugee EST la reference.
type cmJuge struct {
	ref map[[2]int]bool
	// contreRef / utilesRef : par paquet ferme de la reference, s il contredit un invariant et
	// ses records utiles fermes (remplis par le juge de la reference, lus par les autres).
	contreRef            map[[2]int]bool
	utilesRef            map[[2]int]int
	chk                  *cmCollecteur
	t                    cmTables
	gagnes, perdus       int
	gagnesContredits     int
	fermes, fermesContre int
	entreesUtiles        int
	entrees              int
}

func cmNouveauJuge(f *cmFilm, b *cmBlocs, ref map[[2]int]bool) *cmJuge {
	return &cmJuge{ref: ref, chk: nouveauCollecteur(f, b, nil), t: cmTables{},
		contreRef: map[[2]int]bool{}, utilesRef: map[[2]int]int{}}
}

// etatRef : le paquet tel que la reference l a juge.
func (j *cmJuge) etatRef(cle [2]int) string {
	if j.contreRef[cle] {
		return "ref contredit"
	}
	return "ref sain"
}

func (j *cmJuge) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) { j.chk.chunk = c }

func (j *cmJuge) paquet(_ int, p *cmPaquet, _ *World) {
	cleP := [2]int{p.d.Chunk, p.d.Index}
	avant := j.ref[cleP]
	if !p.d.Fermee {
		if j.ref != nil && avant {
			j.perdus++
			j.t.add("juge", cmJoindre("perdu", j.etatRef(cleP)), cmCompte{n: 1, enJeu: j.utilesRef[cleP]})
		}
		return
	}
	j.fermes++
	for _, e := range p.d.VueC.Entrees {
		j.entrees++
		if e.Bloc {
			j.entreesUtiles++
		}
	}
	statut := "conserve"
	switch {
	case j.ref == nil:
		statut = "reference"
	case !avant:
		statut = "gagne"
		j.gagnes++
	}
	contre := cmContredit(j.chk, p)
	if j.ref == nil {
		j.contreRef[cleP], j.utilesRef[cleP] = len(contre) > 0, p.utilesFermes
	}
	if statut == "conserve" {
		etat := "variante sain"
		if len(contre) > 0 {
			etat = "variante contredit"
		}
		j.t.add("juge", cmJoindre("conserve", j.etatRef(cleP), etat), cmCompte{n: 1, enJeu: p.utilesFermes - j.utilesRef[cleP]})
	}
	cle := "aucun invariant contredit"
	if len(contre) > 0 {
		j.fermesContre++
		if statut == "gagne" {
			j.gagnesContredits++
		}
		cle = "contredit : " + strings.Join(contre, " + ")
	}
	j.t.add("juge", cmJoindre(statut, cle, fmt.Sprintf("%d record(s)", min(len(p.recs), 3))), cmCompte{n: 1, enJeu: p.utilesFermes})
}

func (j *cmJuge) finDeFilm() {}

// cmEntrees : l item 1.3 — les entrees de controle des vues C fermees, et les joueurs du film.
type cmEntrees struct {
	sieges   map[int]bool
	refus    string
	vus      map[int]bool
	ti9      int
	vusChunk map[int]bool
	pqChunk  int
	// sommes sur les paquets delta marches
	paquets, fermes, entrees, utiles, horsSieges int
	sommeTi9, sommeChunk                         int
	t                                            cmTables
}

func cmNouvellesEntrees(f *cmFilm) *cmEntrees {
	e := &cmEntrees{sieges: map[int]bool{}, vus: map[int]bool{}, t: cmTables{}}
	tab, _ := ScanFilmPlayerTable(f.fc.Film())
	e.refus = string(tab.Refusal)
	if e.refus == "" {
		e.refus = "lue"
	}
	for _, s := range tab.Seats {
		e.sieges[s.FilmIndex] = true
	}
	return e
}

func (e *cmEntrees) debutDeChunk(_ int, _ []byte, _ []FilmPacket, w *World) {
	e.finirChunk()
	e.ti9, e.vusChunk, e.pqChunk = 0, map[int]bool{}, 0
	for _, s := range w.slots {
		if s.TypeIndex == 9 {
			e.ti9++
		}
	}
}

func (e *cmEntrees) finirChunk() {
	e.sommeChunk += e.pqChunk * len(e.vusChunk)
	e.pqChunk = 0
}

// cmEcart classe un ecart (entrees - joueurs).
func cmEcart(d int) string {
	switch {
	case d < -2:
		return "< -2"
	case d > 1:
		return "> +1"
	}
	return fmt.Sprintf("%+d", d)
}

func (e *cmEntrees) paquet(_ int, p *cmPaquet, _ *World) {
	e.paquets++
	e.pqChunk++
	e.sommeTi9 += e.ti9
	if !p.d.Fermee {
		return
	}
	e.fermes++
	for _, x := range p.d.VueC.Entrees {
		e.entrees++
		e.vus[x.Index] = true
		e.vusChunk[x.Index] = true
		if !e.sieges[x.Index] {
			e.horsSieges++
		}
		if x.Bloc {
			e.utiles++
		}
	}
	e.t.un("ecart_ti9", cmEcart(len(p.d.VueC.Entrees)-e.ti9))
	if len(e.sieges) > 0 {
		e.t.un("ecart_sieges", cmEcart(len(p.d.VueC.Entrees)-len(e.sieges)))
	}
	if len(p.d.VueC.Entrees) > 32 {
		e.t.un("regle", "plus de 32 entrees")
	}
}

func (e *cmEntrees) finDeFilm() { e.finirChunk() }

// lignes rend les mesures du film.
func (e *cmEntrees) lignes(id, build string) []string {
	union := map[int]bool{}
	for k := range e.sieges {
		union[k] = true
	}
	for k := range e.vus {
		union[k] = true
	}
	var out []string
	add := func(m string, v int) { out = append(out, fmt.Sprintf("%s\t%s\t%s\t%d", id, build, m, v)) }
	add("table du film "+e.refus, 1)
	add("sieges", len(e.sieges))
	add("index vus (vues C fermees)", len(e.vus))
	add("sieges U index vus", len(union))
	add("paquets", e.paquets)
	add("paquets fermes", e.fermes)
	add("entrees fermees", e.entrees)
	add("entrees utiles fermees", e.utiles)
	add("entrees fermees hors sieges", e.horsSieges)
	add("D sieges", e.paquets*len(e.sieges))
	add("D sieges U vus", e.paquets*len(union))
	add("D ti9 au debut du chunk", e.sommeTi9)
	add("D index vus dans le chunk", e.sommeChunk)
	for nom, m := range e.t {
		for _, k := range cmCles(m) {
			add(nom+" "+k, m[k].n)
		}
	}
	return out
}
