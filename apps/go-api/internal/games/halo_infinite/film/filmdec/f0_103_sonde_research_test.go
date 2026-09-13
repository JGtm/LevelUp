package filmdec

// f0_103_sonde_research_test.go — LOT F.0 : LA SONDE, c est-a-dire les valeurs BRUTES.
//
// Un taux de resolution nul ne dit pas POURQUOI il est nul : la reference peut etre lue a la
// mauvaise largeur, porter une base, designer une autre chose que la vie de l objet, ou ne
// rien designer du tout. Cette sonde ecrit, evenement par evenement, les index et generations
// lus A COTE des creations `ti=37` contemporaines. C est la piece a regarder avant toute
// conclusion.
//
// LECTURE SEULE. Gardes et commande : `f0_103_contexte_research_test.go`.
// `F0_SONDE_N` borne le nombre d evenements detailles par film (defaut 20).

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"
)

// f0SondeN borne le detail.
func f0SondeN() int {
	if v := os.Getenv("F0_SONDE_N"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 20
}

// TestF0Sonde103 ecrit les valeurs brutes : refs du 103, creations contemporaines, bandes.
func TestF0Sonde103(t *testing.T) {
	root, ids := f0Films(t)
	cat := f0Catalogue(t)
	cartes := f0Cartes(t, cat)
	n := f0SondeN()
	for _, id := range ids {
		e, ok := cartes[id]
		if !ok {
			continue
		}
		f := f0Charge(t, root, id, e)
		t.Logf("")
		t.Logf("######## SONDE FILM %s ########", id)
		f0SondeBandes(t, f)
		cre := append([]EquipmentCreation(nil), f.Creations...)
		sort.SliceStable(cre, func(i, j int) bool { return cre[i].TimestampUS < cre[j].TimestampUS })
		for i, ev := range f.Ev103 {
			if i >= n {
				t.Logf("  ... (%d autres evenements)", len(f.Ev103)-n)
				break
			}
			t.Logf("  103 @%d ms (chunk %d, paquet %d, pos %d) refs %s",
				(int64(ev.TsUS)-int64(f.BaseUS))/1000, ev.Chunk, ev.Paquet, ev.Pos,
				f0RefsTexte(ev.Refs))
			for _, l := range f0CreationsAutour(cre, ev.TsUS, 1_000_000) {
				t.Logf("        %s", l)
			}
		}
	}
}

// f0RefsTexte formate les trois references.
func f0RefsTexte(refs [3]r7RefVal) string {
	out := ""
	for i, r := range refs {
		if i > 0 {
			out += " | "
		}
		if !r.Present {
			out += fmt.Sprintf("ref%d=absente(dom %d)", i, r.Dom)
			continue
		}
		out += fmt.Sprintf("ref%d=idx %d gen %d (dom %d, w %d)", i, r.Index, r.Gen, r.Dom, r.Width)
	}
	return out
}

// f0CreationsAutour rend les creations `ti=37` a moins de `rayon` µs de l instant donne.
func f0CreationsAutour(cre []EquipmentCreation, at uint64, rayon uint64) []string {
	var out []string
	for _, c := range cre {
		d := int64(c.TimestampUS) - int64(at)
		if d < -int64(rayon) || d > int64(rayon) {
			continue
		}
		out = append(out, fmt.Sprintf("creation ti=37 slot %d gen %d id 0x%08x a %+d ms",
			c.Slot, c.Gen, uint32(c.MPPVal[MPPWord32]), d/1000))
	}
	if len(out) == 0 {
		out = append(out, "aucune creation ti=37 a +/- 1 s")
	}
	return out
}

// f0SondeBandes ecrit les bornes des index en jeu : celles des creations lues, celles des
// references du 103. Deux bandes disjointes diraient a elles seules que la reference ne
// designe pas l objet cree.
func f0SondeBandes(t *testing.T, f f0Film) {
	t.Helper()
	minC, maxC := ^uint32(0), uint32(0)
	gens := map[uint32]int{}
	for _, c := range f.Creations {
		if c.Slot < minC {
			minC = c.Slot
		}
		if c.Slot > maxC {
			maxC = c.Slot
		}
		gens[c.Gen]++
	}
	minR, maxR := ^uint64(0), uint64(0)
	gensR := map[uint32]int{}
	for _, ev := range f.Ev103 {
		for _, r := range ev.Refs {
			if !r.Present {
				continue
			}
			if r.Index < minR {
				minR = r.Index
			}
			if r.Index > maxR {
				maxR = r.Index
			}
			gensR[r.Gen]++
		}
	}
	t.Logf("  bandes : creations ti=37 slots [%d..%d] (%d records, generations %v) · "+
		"references du 103 index [%d..%d] (generations %v)",
		minC, maxC, len(f.Creations), gens, minR, maxR, gensR)
}
