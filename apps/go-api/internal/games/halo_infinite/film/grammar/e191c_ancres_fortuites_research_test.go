//go:build research

package grammar

// e191c_ancres_fortuites_research_test.go — LOT 1.9.1 bis, PAS 2 : LA FERMETURE UNE FOIS LES
// ANCRES FORTUITES RETIREES.
//
// # LA QUESTION, ET POURQUOI ELLE SE POSE MAINTENANT
//
// La fermeture se mesure contre une FRONTIERE : le premier bit du record SUIVANT, tel que le
// balayeur d ancres (`WalkKeyframeWorld`) le rend. Ce balayeur accepte une ancre sur un filtre
// de forme (`kfAnchorFromID` : generation 1..3, slot croissant, mot de 32 bits a `q+32`
// inferieur a 50) — un filtre FORT, mais pas exact. La sonde [D] du meme lot vient de montrer
// qu il laisse passer des ancres fortuites : `ti=4` rend 33 records sur 105 dont le residu vaut
// -42 552 ou -69 552 bits, et `ti=8`, qui n a AUCUN composant, en rend deux a -3 727. Ces
// records-la ne peuvent pas fermer, et ils ne le peuvent pas parce qu ils n existent pas.
//
// UN RECORD FORTUIT COMPTE DEUX FOIS CONTRE LA MESURE : il ne ferme pas lui-meme, ET il donne
// une FAUSSE FRONTIERE au vrai record qui le precede, qui ne peut donc pas fermer non plus.
//
// # LE FILTRE, ET SA SOURCE
//
// `FUN_142e2bfd0` lit `n1` a une position FIXE depuis le debut du record (108 bits, la fin de
// l en-tete par entite) : c est une TAILLE DE TAMPON, donc une CONSTANTE par archetype et par
// build (`default_state_n2_constant_test.go` le documente et s en sert deja pour ecarter les
// memes ancres). Un record dont le `n1` s ecarte du modal de son archetype sur la bobine n est
// pas un record de cet archetype.
//
// LE FILTRE EST APPLIQUE AVANT LE CALCUL DES FRONTIERES, pas apres : les records retenus sont
// re-apparies entre eux, exactement comme le jeu les lirait.
//
// CE QUE CA NE PROUVE PAS : `n1` modal n est pas une preuve d existence, c est une preuve de
// COHERENCE. Un ecart mesure entre les deux populations dit que le denominateur du pas 1 etait
// pollue ; il ne donne aucune grammaire (D13).
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cAncresFortuites$' -v -count=1

import (
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cN1Bit est la position de `n1` depuis le debut du record : la fin de l en-tete par entite.
const e191cN1Bit = keyframeFullStateHeaderBits

// e191cRec est un record d image-cle avec son `n1`, avant tout appariement.
type e191cRec struct {
	Bit, TI int
	N1      uint64
}

// e191cRecords rend tous les records d un payload avec leur `n1`.
func e191cRecords(pay []byte) []e191cRec {
	toutes := keyframeBornesToutes(pay)
	out := make([]e191cRec, 0, len(toutes))
	for _, b := range toutes {
		out = append(out, e191cRec{Bit: b.Bit, TI: b.TI, N1: kfReadBits(pay, b.Bit+e191cN1Bit, 32)})
	}
	return out
}

// e191cModal rend, par archetype, la valeur de `n1` la plus frequente sur toute la bobine.
func e191cModal(parTI map[int]map[uint64]int) map[int]uint64 {
	modal := map[int]uint64{}
	for ti, hist := range parTI {
		meilleure, meilleurN := uint64(0), -1
		for v, n := range hist {
			if n > meilleurN || (n == meilleurN && v < meilleure) {
				meilleure, meilleurN = v, n
			}
		}
		modal[ti] = meilleure
	}
	return modal
}

// TestE191cAncresFortuites compare la fermeture des cinq archetypes objet AVANT et APRES le
// retrait des ancres dont le `n1` s ecarte du modal de leur archetype.
func TestE191cAncresFortuites(t *testing.T) {
	t.Logf("######## PAS 2 — FERMETURE DES CINQ ARCHETYPES, ANCRES FORTUITES RETIREES ########")
	brut, filtre := map[int]*e191cFermeture{}, map[int]*e191cFermeture{}
	retires, total := 0, 0
	for _, court := range closureMiniFilms() {
		r, tt := e191cUneBobine(t, court, brut, filtre)
		retires, total = retires+r, total+tt
	}
	t.Logf("  ancres retirees : %d sur %d records (%.2f %%)", retires, total,
		100*float64(retires)/float64(max(total, 1)))
	t.Logf("  %-8s %18s %18s", "ti", "AVANT filtre", "APRES filtre")
	var sBrut, sFiltre e191cFermeture
	for _, ti := range e191cCinq {
		a, b := e191cOu(brut, ti), e191cOu(filtre, ti)
		sBrut.add(a)
		sFiltre.add(b)
		t.Logf("  ti=%-5d %7d/%-6d %5.2f%%  %7d/%-6d %5.2f%%", ti, a.Fermes, a.Bornes, a.pct(),
			b.Fermes, b.Bornes, b.pct())
	}
	t.Logf("  LES CINQ  %7d/%-6d %5.2f%%  %7d/%-6d %5.2f%%", sBrut.Fermes, sBrut.Bornes, sBrut.pct(),
		sFiltre.Fermes, sFiltre.Bornes, sFiltre.pct())
}

// e191cOu rend la mesure d un archetype, ou la mesure vide.
func e191cOu(m map[int]*e191cFermeture, ti int) e191cFermeture {
	if f := m[ti]; f != nil {
		return *f
	}
	return e191cFermeture{}
}

// e191cUneBobine mesure une bobine dans les deux regimes et rend (ancres retirees, records vus).
func e191cUneBobine(t *testing.T, court string, brut, filtre map[int]*e191cFermeture) (int, int) {
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
	pays := e191cPayloads(fc)
	hist := map[int]map[uint64]int{}
	for _, pay := range pays {
		for _, r := range e191cRecords(pay) {
			if hist[r.TI] == nil {
				hist[r.TI] = map[uint64]int{}
			}
			hist[r.TI][r.N1]++
		}
	}
	modal := e191cModal(hist)
	retires, total := 0, 0
	for _, pay := range pays {
		recs := e191cRecords(pay)
		total += len(recs)
		gardes := make([]e191cRec, 0, len(recs))
		for _, r := range recs {
			if r.N1 != modal[r.TI] {
				retires++
				continue
			}
			gardes = append(gardes, r)
		}
		e191cCompter(pay, reg, recs, brut)
		e191cCompter(pay, reg, gardes, filtre)
	}
	return retires, total
}

// e191cCompter apparie les records RETENUS entre eux et compte la fermeture par archetype.
func e191cCompter(pay []byte, reg *Registry, recs []e191cRec, dst map[int]*e191cFermeture) {
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	for i := 0; i+1 < len(recs); i++ {
		if dst[recs[i].TI] == nil {
			dst[recs[i].TI] = &e191cFermeture{}
		}
		f := dst[recs[i].TI]
		f.Bornes++
		tr := WalkKeyframeFullState(pay, recs[i].Bit, reg, contexteDInstrument())
		if tr.DesyncAt < 0 && tr.EndBit == recs[i+1].Bit {
			f.Fermes++
		}
	}
}
