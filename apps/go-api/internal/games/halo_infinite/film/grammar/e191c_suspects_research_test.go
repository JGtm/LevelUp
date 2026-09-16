//go:build research

package grammar

// e191c_suspects_research_test.go — LOT 1.9.1 bis, PAS 2 BIS : LA LISTE COURTE DES COMPOSANTS
// QUI PEUVENT ENCORE ETRE FAUX.
//
// # CE QUE LA CARTE DU GOLDEN VIENT DE MONTRER
//
// Les archetypes qui ne ferment pas se separent en DEUX classes, et la colonne « bloquant » du
// golden 0.A.3 les separe sans ambiguite :
//
//	(A) DESYNCHRONISATION — un composant SANS LECTEUR est nomme, et il arrete la marche.
//	    ti=0, 2, 3, 5, 9, 10, 11, 12, 19, 25, 26, 27, 34, 35, 43, 45, 47. Leur correction est
//	    un PORTAGE (lot 3.6), pas une largeur.
//	(B) AUCUNE DESYNCHRONISATION, et pourtant la marche rate la frontiere : **ti=13, 21, 37,
//	    38, 41, 42** — SIX archetypes, et deux d entre eux (13 et 21) ne portent meme pas
//	    `object-position-component`. C est LA population a expliquer, et elle est bien plus
//	    petite que « la famille objet » du pas 1.
//
// # CE QUE CET INSTRUMENT CALCULE
//
// Un record qui FERME a lu juste TOUS ses composants : les composants des archetypes qui
// ferment sont donc blanchis. La liste courte des suspects est :
//
//	{ composants des six archetypes de la classe (B) } \ { composants des archetypes qui ferment }
//
// C est une preuve bornante, pas une correlation : un composant blanchi a ete lu au bit pres
// dans au moins un record qui atterrit exactement sur la frontiere.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cListeCourte$' -v -count=1

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cSansDesync sont les archetypes de la classe (B) : aucun composant sans lecteur, et
// pourtant la marche ne ferme pas (colonne « bloquant » VIDE dans `keyframe_closure.golden`).
var e191cSansDesync = []int{13, 21, 37, 38, 41, 42}

// TestE191cListeCourte publie la liste courte des composants encore suspects.
func TestE191cListeCourte(t *testing.T) {
	blanchis, suspects := map[string]int{}, map[string][]int{}
	for _, court := range closureMiniFilms() {
		e191cUnFilmListe(t, court, blanchis, suspects)
	}
	noms := make([]string, 0, len(suspects))
	for n := range suspects {
		if blanchis[n] == 0 {
			noms = append(noms, n)
		}
	}
	sort.Strings(noms)
	t.Logf("######## PAS 2 BIS — LA LISTE COURTE DES COMPOSANTS ENCORE SUSPECTS ########")
	t.Logf("  population (B) : archetypes %v — aucune desynchronisation, aucune fermeture", e191cSansDesync)
	t.Logf("  %d composants distincts y vivent ; %d sont BLANCHIS par un archetype qui ferme",
		len(suspects), len(suspects)-len(noms))
	t.Logf("  SUSPECTS (%d) — jamais lus dans un record qui ferme :", len(noms))
	for _, n := range noms {
		tis := suspects[n]
		sort.Ints(tis)
		t.Logf("     %-52s dans ti=%v", n, e191cUnique(tis))
	}
}

// e191cUnFilmListe accumule, pour une bobine, les composants blanchis et les suspects.
func e191cUnFilmListe(t *testing.T, court string, blanchis map[string]int, suspects map[string][]int) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	stats, err := KeyframeClosure(fc)
	if err != nil {
		t.Fatalf("KeyframeClosure %s : %v", court, err)
	}
	for ti, s := range stats {
		i := int(ti) //nolint:gosec // index d archetype
		arch, ok := reg.Archetype(i)
		if !ok {
			continue
		}
		f := e191cFermeture{Fermes: s.Closed, Bornes: s.Total}
		if f.Bornes > 0 && f.pct() >= e191cSeuilBlanchi {
			for _, n := range arch.Components {
				blanchis[n]++
			}
		}
		if e191cDansB(i) {
			for _, n := range arch.Components {
				suspects[n] = append(suspects[n], i)
			}
		}
	}
}

// e191cDansB dit si l archetype appartient a la population (B).
func e191cDansB(ti int) bool {
	for _, v := range e191cSansDesync {
		if v == ti {
			return true
		}
	}
	return false
}

// e191cUnique rend la liste triee sans doublon.
func e191cUnique(in []int) []int {
	out := in[:0:0]
	for i, v := range in {
		if i == 0 || v != in[i-1] {
			out = append(out, v)
		}
	}
	return out
}
