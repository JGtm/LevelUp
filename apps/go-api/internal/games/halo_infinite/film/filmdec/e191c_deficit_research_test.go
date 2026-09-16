//go:build research

package filmdec

// e191c_deficit_research_test.go — LOT 1.9.1 bis, PAS 3 QUATER, VOIE (B) : COMBIEN DE BITS
// MANQUENT, ET SUR QUELS ARCHETYPES ?
//
// # LA QUESTION, POSEE SANS DEVINER QUEL CHAMP
//
// L oracle `n2` dit que l etat par defaut des films anciens est plus court. Le balayage des
// largeurs MPP attribuait l ecart a `lead`/`index` faute d autres molettes ; la relecture de
// `FUN_14080cfe8` a montre que le bloc MPP est INVARIANT. Plutot que de deviner QUEL champ
// manque, cet instrument mesure DE COMBIEN DE BITS la fin de l etat par defaut se deplace :
// `n2` est lu a la position portee PLUS un decalage, et on regarde quel decalage le rend
// constant.
//
// C est le meme oracle, sans hypothese de structure : un decalage `-3` qui rend `n2` constant a
// 0,99 sur les cinq films anciens ET laisse `0` constant sur les deux recents dit exactement
// « trois bits de moins, et seulement la-bas ».
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cDeficit$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e191cDeltas est le voisinage balaye autour de la fin PORTEE de l etat par defaut.
var e191cDeltas = []int{-6, -5, -4, -3, -2, -1, 0, 1, 2}

// e191cN2AvecDelta rend la part modale de `n2` lu `delta` bits apres la fin portee de l etat.
func e191cN2AvecDelta(ancres []e191cAncre, ti int, delta int) (float64, uint64) {
	hist := map[uint64]int{}
	for _, a := range ancres {
		br := LecteurSur(a.Pay)
		br.SetBitPos(a.Bit + keyframeFullStateHeaderBits)
		n1 := int32(br.ReadBits(keyframeFullStateSizeBits)) //nolint:gosec // 32 bits
		if n1 > 0 {
			consumeKeyframeDefaultState(br, uint32(ti)) //nolint:gosec // index d archetype
		}
		hist[kfReadBits(a.Pay, br.BitPos()+delta, 32)]++
	}
	meilleure, n := uint64(0), 0
	for v, c := range hist {
		if c > n || (c == n && v < meilleure) {
			meilleure, n = v, c
		}
	}
	if len(ancres) == 0 {
		return 0, 0
	}
	return float64(n) / float64(len(ancres)), meilleure
}

// TestE191cDeficit colle, par bobine et par archetype, la part modale de `n2` pour chaque
// decalage du voisinage.
func TestE191cDeficit(t *testing.T) {
	t.Logf("######## PAS 3 QUATER (B) — LE DEFICIT EN BITS DE L ETAT PAR DEFAUT ########")
	t.Logf("  part des records dont `n2` prend la valeur modale, par decalage depuis la fin portee")
	for _, court := range closureMiniFilms() {
		e191cDeficitBobine(t, court)
	}
}

// e191cDeficitBobine colle une ligne par archetype.
func e191cDeficitBobine(t *testing.T, court string) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	build := "(sans section)"
	if p, e := BuildProfileFromFilm(film); e == nil {
		build = p.Build
	}
	parTI := map[int][]e191cAncre{}
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			parTI[b.TI] = append(parTI[b.TI], e191cAncre{Pay: pay, Bit: b.Bit})
		}
	}
	for _, ti := range []int{37, 38, 42} {
		anc := parTI[ti]
		if len(anc) == 0 {
			continue
		}
		ligne := ""
		meilleur, meilleurD := 0.0, 0
		for _, d := range e191cDeltas {
			part, _ := e191cN2AvecDelta(anc, ti, d)
			if part > meilleur {
				meilleur, meilleurD = part, d
			}
			ligne += fmt.Sprintf(" %+d:%.2f", d, part)
		}
		t.Logf("  %-10s %-12s ti=%-2d (%4d) meilleur %+d a %.3f |%s",
			court, build, ti, len(anc), meilleurD, meilleur, ligne)
	}
}

// LE RESULTAT DU PAS 3 QUATER (B), ET IL REFUTE LE CADRAGE « TROIS BITS DE MOINS » (2026-09-16).
//
// Part modale de `n2` par decalage, `ti=37` (les deltas balayes vont de -6 a +2) :
//
//	fb1a1a72  HI_1_13_0   +0 -> 1,000   (les autres decalages : 0,58 a 0,72)
//	bcb6d393  HI_1_12_0   +0 -> 0,949
//	a521164d  HI_1_4_1    PLAT de 0,27 a 0,32 — aucun decalage ne ressort
//	60ae07c4  HI_1_8_0    PLAT a 0,52
//	11de8353  HI_1_9_0    0,49 partout, +2 -> 0,578
//	111fa685  HI_1_10_0   0,43 partout, +2 -> 0,545
//	e5adf7b2  HI_1_11_0   0,47 partout, +2 -> 0,581
//
// DEUX CHOSES EN SORTENT, ET LA SECONDE CORRIGE UNE LECTURE PRECEDENTE.
//
//	1. LE PORTAGE EST EXACT SUR LES FILMS RECENTS : `+0` rend 1,000 sur `ti=37` et `ti=42` de
//	   `fb1a1a72`. La lecture n a pas besoin d etre touchee la.
//	2. AUCUN DECALAGE PUR NE MARCHE SUR LES FILMS ANCIENS. Or le decoupage `8/3` du pas 3
//	   atteignait 0,988 a 0,996 — et il deplace la fin de l etat de exactement trois bits. Si
//	   l ecart etait un simple raccourcissement, `-3` aurait rendu le meme score : il rend 0,27
//	   a 0,49. **Le `8/3` n est donc PAS un decalage, c est une autre ANALYSE** : le bloc MPP
//	   porte des champs a longueur dependante des donnees (`R(3)` de compte puis boucle, portes
//	   sur valeurs lues), si bien que changer une largeur de tete change les valeurs lues
//	   ensuite, donc les branches prises, donc la longueur totale.
//
// CONCLUSION : la difference des films anciens est STRUCTURELLE (une branche, un champ present
// ou absent), pas un offset. Chercher « les trois bits » comme une constante est une impasse ;
// c est la CONDITION qu il faut trouver chez l ecrivain.
//
// ATTENTION EN RELISANT CE TABLEAU : `ti=38` rend 1,000 a TOUS les decalages sur les deux films
// recents — son `n2` y est constant quoi qu on lise, donc il ne discrimine rien et ne doit pas
// etre compte comme une confirmation.
