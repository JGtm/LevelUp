//go:build research

package grammar

// e191c_prefixe_v_research_test.go — LOT 1.9.1 bis, PAS 3 TER : LE PREFIXE `V` EST-IL LA CLE ?
//
// # L HYPOTHESE
//
// Les etats par defaut commencent par `V` = `R(1)` [si 1 : `R(8)`] (lot 1.3). C est une VERSION
// lue DANS LE RECORD, donc dans le film — exactement la forme que le fait tranche par
// l utilisateur appelle (« le film ne depend que de lui-meme »). Si sa valeur differe entre les
// cinq films courts et les deux longs, c est elle la cle, et non la table par type de la
// section 2.
//
// # CE QUE L INSTRUMENT MESURE
//
// Pour chaque bobine et chaque record de `ti=37`, `ti=38` et `ti=42` : le bit de porte de `V` et,
// s il est mis, ses huit bits de valeur, lus a la position EXACTE ou l etat par defaut commence
// (record + 108 d en-tete + 32 de `n1`). Pour `ti=37`, dont l etat par defaut enchaine DEUX
// prefixes (le sien puis celui de `ti=36`), le second est lu aussi.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cPrefixeV$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cDebutEtat est la position du premier bit de l etat par defaut, depuis le debut du record.
const e191cDebutEtat = keyframeFullStateHeaderBits + keyframeFullStateSizeBits

// e191cLireV lit un prefixe `V` a la position `pos` et rend sa valeur lisible et sa largeur.
func e191cLireV(pay []byte, pos int) (string, int) {
	if kfReadBits(pay, pos, 1) == 0 {
		return "absent", 1
	}
	return fmt.Sprintf("%d", kfReadBits(pay, pos+1, 8)), 9
}

// TestE191cPrefixeV colle la distribution des prefixes `V` par bobine et par archetype.
func TestE191cPrefixeV(t *testing.T) {
	t.Logf("######## PAS 3 TER — LE PREFIXE `V` DES ETATS PAR DEFAUT, BOBINE PAR BOBINE ########")
	t.Logf("  (position : record + %d bits = en-tete 108 + mot de taille 32)", e191cDebutEtat)
	for _, court := range closureMiniFilms() {
		e191cPrefixeVBobine(t, court)
	}
}

// e191cPrefixeVBobine colle une ligne par archetype pour une bobine.
func e191cPrefixeVBobine(t *testing.T, court string) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	build := "(sans section)"
	if _, d0 := readChunk00(t, dir); len(d0) > 0 {
		if id, e := ReadFilmIdentity(d0); e == nil {
			build = id.Build
		}
	}
	for _, ti := range []int{37, 38, 42} {
		un, deux := map[string]int{}, map[string]int{}
		for _, pay := range e191cPayloads(fc) {
			for _, b := range keyframeBornes(pay) {
				if b.TI != ti {
					continue
				}
				v1, w := e191cLireV(pay, b.Bit+e191cDebutEtat)
				un[v1]++
				if ti == 37 { // ti=37 enchaine son `V` puis celui de ti=36
					v2, _ := e191cLireV(pay, b.Bit+e191cDebutEtat+w)
					deux[v2]++
				}
			}
		}
		if len(un) == 0 {
			continue
		}
		ligne := fmt.Sprintf("  %-10s %-12s ti=%-2d  V1=%s", court, build, ti, e191cHisto(un))
		if len(deux) > 0 {
			ligne += "   V2=" + e191cHisto(deux)
		}
		t.Logf("%s", ligne)
	}
}

// e191cHisto colle un histogramme trie par frequence decroissante.
func e191cHisto(m map[string]int) string {
	type kv struct {
		k string
		n int
	}
	l := make([]kv, 0, len(m))
	for k, n := range m {
		l = append(l, kv{k, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].k < l[j].k
	})
	out := ""
	for i, e := range l {
		if i >= 4 {
			out += " ..."
			break
		}
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s×%d", e.k, e.n)
	}
	return out
}

// LE RESULTAT DU PAS 3 TER : LE PREFIXE `V` N EST PAS LA CLE (2026-09-16).
//
// Mesure sur les sept bobines — IDENTIQUE PARTOUT, sans une exception :
//
//	ti=37  V1 = absent (bit de porte a 0) sur 100 % des records, V2 = absent de meme
//	ti=38  V1 = 3      sur 1 303 / 2 098 / 2 262 / 1 365 / 1 044 / 1 837 / 2 147 records
//	ti=42  V1 = 2      sur 696 / 250 / 151 / 141 / 186 / 420 / 216 records
//
// Le prefixe `V` ne separe donc RIEN : ni les cinq films « courts » des deux « longs », ni quoi
// que ce soit d autre. C est un negatif MESURE, et il ferme la piste (3) du pas 3 ter.
//
// AU PASSAGE, UN FAIT UTILE : le `V` de `ti=37` coute UN bit, pas neuf — sa porte est a zero sur
// tous les records des sept bobines, et celle de l etat par defaut de `ti=36` qu il enchaine
// aussi. L etat par defaut de `ti=37` commence donc par DEUX bits de prefixe.
