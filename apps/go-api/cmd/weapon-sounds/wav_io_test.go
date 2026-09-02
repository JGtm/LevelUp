package main

import (
	"math"
	"testing"
)

// TestVersStereoNePartagePasLesCanaux garde le correctif du 2026-09-02 : la montee mono ->
// stereo doit COPIER l'echantillonnage. Quand les deux canaux pointaient sur le meme
// tableau, `appliquerGain` le multipliait deux fois et le gain de chemin arrivait au carre
// (+7 dB rendus a +14 dB) — sur les seules couches mono, donc invisible sans mesure.
func TestVersStereoNePartagePasLesCanaux(t *testing.T) {
	mono := &audio{Taux: 48000, Canaux: 1, Ech: [][]float64{{0.5, -0.5}}}
	st := versStereo(mono)
	appliquerGain(st, -6)
	attendu := 0.5 * math.Pow(10, -6.0/20)
	for c := range st.Ech {
		if math.Abs(st.Ech[c][0]-attendu) > 1e-9 {
			t.Fatalf("canal %d : %.6f, attendu %.6f (gain applique deux fois ?)", c, st.Ech[c][0], attendu)
		}
	}
	if mono.Ech[0][0] != 0.5 {
		t.Fatalf("la source mono a ete modifiee : %.6f", mono.Ech[0][0])
	}
}

// TestBouclerEtCouperTiennentLaDuree : un corps de boucle repete puis coupe doit rendre
// exactement la duree demandee, sans reallouer le contenu de la source.
func TestBouclerEtCouperTiennentLaDuree(t *testing.T) {
	src := &audio{Taux: 4, Canaux: 1, Ech: [][]float64{{1, 2, 3}}}
	out := couper(boucler(src, 8), 8)
	if out.nEch() != 8 {
		t.Fatalf("duree %d echantillons, attendu 8", out.nEch())
	}
	for i, v := range []float64{1, 2, 3, 1, 2, 3, 1, 2} {
		if out.Ech[0][i] != v {
			t.Fatalf("echantillon %d = %v, attendu %v", i, out.Ech[0][i], v)
		}
	}
}

// TestDecalerInsereDuSilence : l'offset de couche (`InitialDelay`) doit produire un vrai
// silence en tete, pas un decalage circulaire.
func TestDecalerInsereDuSilence(t *testing.T) {
	src := &audio{Taux: 4, Canaux: 1, Ech: [][]float64{{1, 1}}}
	out := decaler(src, 2)
	if out.nEch() != 4 || out.Ech[0][0] != 0 || out.Ech[0][1] != 0 || out.Ech[0][2] != 1 {
		t.Fatalf("decalage incorrect : %v", out.Ech[0])
	}
}
