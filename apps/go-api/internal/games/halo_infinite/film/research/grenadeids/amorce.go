//go:build research

package grenadeids

// amorce.go — MESURER L AMORCE AU LIEU DE LA SUPPOSER (lot 3.3.1, 2026-09-17).
//
// POURQUOI CETTE PASSE EXISTE. Les passes A a D partent toutes d un MARQUEUR : elles cherchent
// un motif connu, puis regardent ce qui le suit. Sur la majeure 31 (`50247b26`), ce point de
// depart manque — le motif de 24 bits ne compte que 13 occurrences sur 22 Mio de flux delta,
// alors que l identifiant de la grenade a fragmentation y apparait 79 fois contre 0,17 attendue
// par hasard. Les lancers sont donc ECRITS, et c est l AMORCE qui n est ni celle de 23 bits ni
// celle de 24.
//
// LA MESURE RENVERSE L ANCRAGE : on part des occurrences ABSOLUES d un identifiant connu (passe
// B) et on lit CE QUI LES PRECEDE, largeur par largeur. Pour la bonne largeur, la valeur lue est
// CONSTANTE — c est la definition d une amorce. Pour toute autre, elle porte un bit de donnee et
// se disperse.
//
// Rien n est interprete ici : la passe publie, par largeur, la valeur modale et sa part.

import (
	"fmt"
	"io"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// largeurAmorceMin / largeurAmorceMax : les largeurs d amorce sondees, en bits. La grammaire
// connue en porte deux (23 et 24) ; la fenetre est ouverte de part et d autre pour qu une
// troisieme se voie.
const (
	largeurAmorceMin = 16
	largeurAmorceMax = 28
)

// noterAmorce accumule, pour UNE occurrence absolue d un identifiant connu, la valeur qui la
// precede a chaque largeur sondee.
func noterAmorce(pay []byte, bp int, r *Releve) {
	if r.AmorceParLargeur == nil {
		r.AmorceParLargeur = map[int]map[uint64]int{}
	}
	for w := largeurAmorceMin; w <= largeurAmorceMax; w++ {
		if bp-w < 0 {
			continue
		}
		h := r.AmorceParLargeur[w]
		if h == nil {
			h = map[uint64]int{}
			r.AmorceParLargeur[w] = h
		}
		h[grammar.PeekBits(pay, bp-w, w)]++
	}
}

// EcrireAmorce publie, par largeur, la valeur modale de ce qui precede un identifiant connu et
// sa part. Une amorce est une CONSTANTE : la part de la modale la designe sans interpretation.
func EcrireAmorce(w io.Writer, r *Releve) {
	if len(r.AmorceParLargeur) == 0 {
		return
	}
	largeurs := make([]int, 0, len(r.AmorceParLargeur))
	for l := range r.AmorceParLargeur {
		largeurs = append(largeurs, l)
	}
	sort.Ints(largeurs)
	fmt.Fprintln(w, "   passe E  ce qui PRECEDE un identifiant connu, largeur par largeur (valeur modale et sa part)")
	for _, l := range largeurs {
		val, n, total := modale(r.AmorceParLargeur[l])
		fmt.Fprintf(w, "     largeur=%d  modale=0x%0*X  %d/%d\n", l, (l+3)/4, val, n, total)
	}
}

// modale rend la valeur la plus frequente d un histogramme, son compte et le total.
func modale(h map[uint64]int) (uint64, int, int) {
	var val uint64
	n, total := 0, 0
	for v, c := range h {
		total += c
		if c > n || (c == n && v < val) {
			val, n = v, c
		}
	}
	return val, n, total
}

// LE CHAMP D INDEX AUTEUR, MESURE DEPUIS L IDENTIFIANT ET NON DEPUIS LE MARQUEUR (lot 3.3.1).
//
// Sur la majeure 31 le marqueur de production ne se trouve pas : l ancrage sur le marqueur ne
// peut donc pas servir. L identifiant, lui, se trouve — par la passe B. La fenetre ci-dessous
// balaye donc les decalages DEPUIS LE DEBUT DE L IDENTIFIANT, et publie pour chacun la
// dispersion (nombre de valeurs distinctes et maximum), qui est le critere fort : un champ
// d index de joueur prend autant de valeurs qu il y a de lanceurs, une suite de zeros n en
// prend qu une.

// indexDepuisIdMin / indexDepuisIdMax : la fenetre de decalages sondee depuis l identifiant.
// L identifiant fait 32 bits, le champ d index vient apres.
const (
	indexDepuisIdMin = 32
	indexDepuisIdMax = 128
)

// noterIndexDepuisIdentifiant accumule la dispersion du champ de cinq bits a chaque decalage
// depuis le debut d un identifiant connu.
func noterIndexDepuisIdentifiant(pay []byte, bp int, r *Releve) {
	if r.IndexDepuisIdentifiant == nil {
		r.IndexDepuisIdentifiant = map[int]*CompteIndex{}
	}
	for d := indexDepuisIdMin; d <= indexDepuisIdMax; d++ {
		c := r.IndexDepuisIdentifiant[d]
		if c == nil {
			c = &CompteIndex{}
			r.IndexDepuisIdentifiant[d] = c
		}
		c.Total++
		v := grammar.PeekBits(pay, bp+d, indexAuteurBits)
		c.Vues |= 1 << uint(v&0x1F)
		if v <= indexAuteurMax {
			c.Dans0a7++
		}
	}
}

// EcrireIndexDepuisIdentifiant publie les decalages les plus disperses.
func EcrireIndexDepuisIdentifiant(w io.Writer, r *Releve, top int) {
	if len(r.IndexDepuisIdentifiant) == 0 {
		return
	}
	type ligne struct {
		d int
		c CompteIndex
	}
	var lignes []ligne
	for d, c := range r.IndexDepuisIdentifiant {
		lignes = append(lignes, ligne{d: d, c: *c})
	}
	sort.Slice(lignes, func(i, j int) bool {
		a, b := lignes[i], lignes[j]
		if a.c.Distincts() != b.c.Distincts() {
			return a.c.Distincts() > b.c.Distincts()
		}
		return a.d < b.d
	})
	fmt.Fprintln(w, "   passe E  dispersion du champ de 5 bits par decalage DEPUIS l identifiant")
	for i, l := range lignes {
		if top > 0 && i >= top {
			break
		}
		fmt.Fprintf(w, "     depuis_id=%+d  distincts=%d max=%d dans0a7=%d/%d\n",
			l.d, l.c.Distincts(), l.c.Max(), l.c.Dans0a7, l.c.Total)
	}
}
