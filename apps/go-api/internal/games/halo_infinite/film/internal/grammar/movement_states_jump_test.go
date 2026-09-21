package grammar

// movement_states_jump_test.go — LA DERIVATION DU SAUT, SANS FILM (lot 5.9.4).
//
// Ces tests ne lisent aucun octet de film : ils posent des suites de lectures de vitesse a la
// main et verifient que la segmentation, l integration et la fenetre de hauteur font ce que la
// chronique annonce. Un test qui exigerait un film ne tournerait pas en CI.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestSautDeriveReconnaitUneMonteeALaBonneHauteur : une montee de 0,85 m est retenue.
func TestSautDeriveReconnaitUneMonteeALaBonneHauteur(t *testing.T) {
	// 5 m/s tenus 170 ms = 0,85 m. La lecture de fermeture porte une vitesse sous le seuil.
	vs := []jumpVelSample{
		{ts: 0, vz: 0},
		{ts: 100_000, vz: 5.0},
		{ts: 270_000, vz: 0.1},
		{ts: 300_000, vz: 0},
	}
	eps := episodesDeMontee(7, vs)
	if len(eps) != 1 {
		t.Fatalf("episodes = %d, attendu 1 (%+v)", len(eps), eps)
	}
	e := eps[0]
	if e.slot != 7 || e.t0 != 100_000 || e.t1 != 270_000 {
		t.Errorf("bornes = slot %d [%d, %d], attendu slot 7 [100000, 270000]", e.slot, e.t0, e.t1)
	}
	if got := e.haut; got < 0.84 || got > 0.86 {
		t.Errorf("hauteur = %.3f m, attendu ~0,85", got)
	}
	if !hauteurDeSaut(e.haut) {
		t.Errorf("hauteur %.3f refusee par la fenetre du saut", e.haut)
	}
}

// TestSautDeriveEcarteLesAutresHauteurs : une chute de canon a homme et une oscillation de
// marche tombent hors de la fenetre. C EST LE POINT DE LA METHODE — l episode aerien existe,
// mais seule la HAUTEUR en fait un saut.
func TestSautDeriveEcarteLesAutresHauteurs(t *testing.T) {
	cas := []struct {
		nom    string
		h      float64
		retenu bool
	}{
		{"oscillation de marche", 0.20, false},
		{"pres de la borne basse", 0.77, true},
		{"saut du Spartan", 0.85, true},
		{"pres de la borne haute", 0.93, true},
		{"juste au-dessus", 0.95, false},
		{"rampe ou propulsion", 2.40, false},
	}
	for _, c := range cas {
		if got := hauteurDeSaut(c.h); got != c.retenu {
			t.Errorf("%s (%.3f m) : retenu = %v, attendu %v", c.nom, c.h, got, c.retenu)
		}
	}
}

// TestSautDeriveRefuseUnEpisodeOuvert : sans lecture de fermeture, pas d intervalle — la fin
// n est pas mesuree et la hauteur est tronquee par le silence.
func TestSautDeriveRefuseUnEpisodeOuvert(t *testing.T) {
	vs := []jumpVelSample{{ts: 0, vz: 5.0}, {ts: 170_000, vz: 5.0}}
	if eps := episodesDeMontee(1, vs); len(eps) != 0 {
		t.Fatalf("episodes = %d, attendu 0 (un episode ouvert ne se publie pas) : %+v", len(eps), eps)
	}
}

// TestSautDeriveNIntegrePasUnSilence : au-dela de la borne de tenue, la lecture ne vaut plus
// rien. Sans cette borne, un silence de dix secondes a 5 m/s fabriquerait cinquante metres.
func TestSautDeriveNIntegrePasUnSilence(t *testing.T) {
	vs := []jumpVelSample{
		{ts: 0, vz: 5.0},
		{ts: 10_000_000, vz: 5.0},
		{ts: 10_170_000, vz: 0},
	}
	eps := episodesDeMontee(1, vs)
	if len(eps) != 1 {
		t.Fatalf("episodes = %d, attendu 1", len(eps))
	}
	if got := eps[0].haut; got < 0.84 || got > 0.86 {
		t.Errorf("hauteur = %.3f m, attendu ~0,85 (le silence de 10 s ne compte pas)", got)
	}
}

// TestSautDeriveEcarteLesDoublonsDInstant : le chemin d inference republie la meme lecture au
// meme instant ; les bornes d episode ne doivent pas dependre de l ordre de parcours.
func TestSautDeriveEcarteLesDoublonsDInstant(t *testing.T) {
	vs := []jumpVelSample{
		{ts: 100_000, vz: 5.0},
		{ts: 100_000, vz: 5.0},
		{ts: 270_000, vz: 0},
		{ts: 270_000, vz: 0},
	}
	eps := episodesDeMontee(3, vs)
	if len(eps) != 1 || eps[0].t0 != 100_000 || eps[0].t1 != 270_000 {
		t.Fatalf("episodes = %+v, attendu un seul [100000, 270000]", eps)
	}
}

// TestSautDeriveNommeSonGenre est un RATCHET : le genre publie doit porter le mot « derive »
// dans sa valeur meme. Le renommer en `jump` ferait passer un calcul pour une lecture — c est
// la condition posee par l utilisateur le 2026-09-21, et elle se tient par un test.
func TestSautDeriveNommeSonGenre(t *testing.T) {
	if types.MovementJumpDerived != "jumpDerived" {
		t.Fatalf("genre du saut derive = %q, attendu \"jumpDerived\" : un genre `jump` NU est "+
			"reserve a une lecture du film, et la chaine du declencheur n est pas trouvee "+
			"(lot 5.9.2, FUN_1408b2f90)", types.MovementJumpDerived)
	}
	if types.SpartanJumpHeightM != 0.85 {
		t.Fatalf("SpartanJumpHeightM = %v, attendu 0.85 (mesure du lot 5.7.5, deux films)",
			types.SpartanJumpHeightM)
	}
}
