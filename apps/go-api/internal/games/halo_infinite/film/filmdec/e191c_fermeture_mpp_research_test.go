//go:build research

package filmdec

// e191c_fermeture_mpp_research_test.go — LOT 1.9.1 bis, PAS 3 : LA FERMETURE ARBITRE ENTRE LES
// DEUX ORACLES.
//
// # LA CONTRADICTION QU IL FAUT TRANCHER
//
// L oracle `n2` (le mot de taille lu APRES l etat par defaut) designe `lead=8 index=3` sur les
// cinq builds <= HI_1_11_0, avec une part modale de 0,988 a 0,996 contre 0,304 a 0,522 pour les
// largeurs de l ecrivain. Mais quand on POSE ce decoupage, la FERMETURE DESCEND : 246 -> 182
// sur les cinq archetypes objet, et `ti=38` perd 62 records.
//
// Les deux oracles ne peuvent pas avoir raison ensemble. `n2` est FAIBLE (une valeur qui revient
// souvent peut revenir pour une autre raison) ; la fermeture est STRICTE (`EndBit == Want` au
// bit pres, sur tout le record). Le chantier dit que la fermeture est « le seul oracle qui
// prouve une largeur de composant sans capture live » : c est donc elle qui tranche.
//
// # CE QUE CET INSTRUMENT MESURE
//
// La FERMETURE, bobine par bobine, pour un petit voisinage de decoupages autour des largeurs de
// l ecrivain. Il n appelle PAS `KeyframeClosure` — qui installe desormais le profil du build —
// mais rejoue la marche lui-meme, decoupage impose.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cFermetureMPP$' -v -count=1

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e191cLeads / e191cIndexes : le voisinage balaye. Le balayage large (16 x 16) a deja eu lieu
// contre `n2` ; ici on veut la fermeture, qui coute une marche complete par record.
var (
	e191cLeads   = []int{7, 8, 9, 10}
	e191cIndexes = []int{3, 4, 5}
)

// TestE191cFermetureMPP colle, par bobine, la fermeture obtenue pour chaque decoupage.
func TestE191cFermetureMPP(t *testing.T) {
	t.Logf("######## PAS 3 — LA FERMETURE CONTRE LES DECOUPAGES MPP, BOBINE PAR BOBINE ########")
	t.Logf("  (somme des archetypes qui portent le bloc MPP : ti=36, 37, 38, 39, 42, 43)")
	for _, court := range closureMiniFilms() {
		e191cFermetureBobine(t, court)
	}
}

// e191cPorteMPP sont les archetypes dont l etat par defaut contient le bloc MPP.
var e191cPorteMPP = map[int]bool{36: true, 37: true, 38: true, 39: true, 42: true, 43: true}

// e191cFermetureBobine colle une ligne par decoupage pour une bobine.
func e191cFermetureBobine(t *testing.T, court string) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre %s : %v", court, err)
	}
	build := "(sans section)"
	if p, e := BuildProfileFromFilm(film); e == nil {
		build = p.Build + " profil " + p.MPP.String()
	}
	type borne struct {
		Pay       []byte
		Bit, Want int
		TI        int
	}
	var bornes []borne
	for _, pay := range e191cPayloads(fc) {
		for _, b := range keyframeBornes(pay) {
			if e191cPorteMPP[b.TI] {
				bornes = append(bornes, borne{Pay: pay, Bit: b.Bit, Want: b.Want, TI: b.TI})
			}
		}
	}
	ligne := ""
	for _, l := range e191cLeads {
		for _, i := range e191cIndexes {
			p := contexteDInstrument()
			p.Profil.MPP = MPPWidths{Lead: l, Index: i}
			n := 0
			for _, b := range bornes {
				tr := WalkKeyframeFullState(b.Pay, b.Bit, reg, p)
				if tr.DesyncAt < 0 && tr.EndBit == b.Want {
					n++
				}
			}
			marque := " "
			if l == 9 && i == 5 {
				marque = "*"
			}
			ligne += " " + itoaN(l) + "/" + itoaN(i) + marque + "=" + itoaN(n)
		}
	}
	t.Logf("  %-10s %-26s %5d bornes :%s", court, build, len(bornes), ligne)
}

// itoaN formate un entier court.
func itoaN(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
