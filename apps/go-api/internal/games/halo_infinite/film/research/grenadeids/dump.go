//go:build research

package grenadeids

// dump.go — DESCENDRE JUSQU AUX BITS.
//
// Quand deux builds rendent des histogrammes differents a la meme position, l arithmetique sur
// les histogrammes ne tranche pas : elle propose des hypotheses qui collent toutes. Ce qui
// tranche, c est la SUITE DE BITS elle-meme, lue au meme decalage sur un film de chaque build et
// posee cote a cote. C est la methode que `bit_projectile_research_test.go` emploie deja pour la
// position des projectiles ; elle est reprise ici telle quelle.
//
// L instrument releve, pour les premieres occurrences d un identifiant donne derriere le
// marqueur, la tranche de bits qui entoure le marqueur — trente-deux bits AVANT (pour voir le
// sixieme bit d index et ce qui precede) et deux cent vingt-quatre APRES. Rien n est interprete
// ici : la comparaison se fait dans la note.

import (
	"fmt"
	"io"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// dumpAvant / dumpApres : la fenetre relevee autour du marqueur, en bits.
const (
	dumpAvant = 32
	dumpApres = 224
)

// Tranche est la suite de bits qui entoure UN marqueur.
type Tranche struct {
	Occurrence
	// Bits porte les `dumpAvant + dumpApres` bits, en binaire, le marqueur commencant au
	// caractere `dumpAvant`.
	Bits string
}

// releverTranche lit la fenetre autour d un marqueur.
func releverTranche(pay []byte, o Occurrence) Tranche {
	var b strings.Builder
	b.Grow(dumpAvant + dumpApres)
	for i := -dumpAvant; i < dumpApres; i++ {
		b.WriteByte(byte('0' + grammar.PeekBits(pay, o.BitPos+i, 1)))
	}
	return Tranche{Occurrence: o, Bits: b.String()}
}

// EcrireTranches publie les tranches relevees, en blocs de huit bits pour que les frontieres
// d octet se voient, avec une reglette de decalages relatifs au marqueur.
func EcrireTranches(w io.Writer, r *Releve) {
	if len(r.Tranches) == 0 {
		return
	}
	fmt.Fprintf(w, "   BITS  %d tranche(s) relevee(s) ; position 0 = debut du marqueur, negatif = avant\n",
		len(r.Tranches))
	for _, t := range r.Tranches {
		fmt.Fprintf(w, "     ti=%d idx=%d ts=%d chunk=%d paquet=%d bit=%d\n",
			t.TypeIndex, t.Index, t.TimestampUS, t.Chunk, t.Paquet, t.BitPos)
		for i := 0; i < len(t.Bits); i += 64 {
			fin := i + 64
			if fin > len(t.Bits) {
				fin = len(t.Bits)
			}
			fmt.Fprintf(w, "       %+5d  %s\n", i-dumpAvant, espacerParOctet(t.Bits[i:fin]))
		}
	}
}

// espacerParOctet insere une espace tous les huit caracteres.
func espacerParOctet(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 8 {
		if i > 0 {
			b.WriteByte(' ')
		}
		fin := i + 8
		if fin > len(s) {
			fin = len(s)
		}
		b.WriteString(s[i:fin])
	}
	return b.String()
}
