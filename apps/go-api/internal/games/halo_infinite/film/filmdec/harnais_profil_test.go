package filmdec

// harnais_profil_test.go — LE PROFIL DE BALAYAGE DES INSTRUMENTS (lot 2.3 du PLAN_DECODEUR_FILM).
//
// # CE QUE CE FICHIER REMPLACE
//
// Jusqu au lot 2.3, un instrument posait les largeurs d axe d une carte dans une VARIABLE DE
// PAQUET (`SetWorldObjectPrecisionFromLayout`, `SetMPPWidths`, `SetRecordStateParam`) et TOUS
// les lecteurs de bits du processus les prenaient — c est ce qui obligeait le decodage a
// passer sous un verrou. Le profil voyage desormais avec le contexte du film
// ([FilmContext.ProfilDeBalayage]) et avec le lecteur ([Lecteur.PoserProfil]) : un
// instrument ouvre SON contexte et y pose ce qu il mesure.
//
// CE FICHIER NE GARDE AUCUN ETAT ENTRE DEUX TESTS : ce sont deux fonctions pures, et c est la
// propriete qui permet a deux films de se decoder en parallele.

import "testing"

// profilDeCarte rend le profil de balayage d une carte : l invariant, plus les largeurs d axe
// du chemin world-object que le decoupage `lay` porte.
func profilDeCarte(lay I0Layout) ProfilDeBalayage {
	p := ProfilDeBalayageParDefaut()
	p.PoserLargeursObjetDuMondeDepuisDecoupage(lay)
	return p
}

// contexteDuFilm est [ContexteDeFilm] avec l arret de test qui va bien.
func contexteDuFilm(t *testing.T, dir string) (*FilmContext, I0Layout) {
	t.Helper()
	fc, lay, err := ContexteDeFilm(dir)
	if err != nil {
		t.Fatalf("contexte du film %s : %v", dir, err)
	}
	return fc, lay
}

// profilDInstrument : le profil que les HARNAIS DE MESURE posent pour le film courant.
//
// # POURQUOI CET ETAT SURVIT, ET OU IL EST BORNE
//
// Les mesures d image-cle (`kf35b*`, `imagecle_*`, `e191c_*`) enchainent une dizaine de marches
// par film a travers des helpers qui ne se passent pas de profil ; leur en donner un a chacun
// serait la descente du pas 5, pas ce lot. Elles partagent donc un profil de HARNAIS.
//
// IL VIT DANS UN FICHIER `_test.go`, ET C EST LA FRONTIERE. La production de `filmdec` n a plus
// AUCUNE variable de paquet mutable depuis le lot 2.3 — c est ce que le ratchet
// `archlint/filmdec_package_vars_test.go` mesure, et il ne compte que les fichiers NON-test.
// Deux films peuvent donc se decoder en parallele en production ; les harnais de mesure, eux,
// sont sequentiels par construction (un test Go ne s execute en parallele que sur demande, et
// aucun de ceux-ci ne la fait).
var profilDInstrument = ProfilDeBalayageParDefaut()

// poserProfilDInstrument installe le profil du harnais et rend sa restauration.
func poserProfilDInstrument(p ProfilDeBalayage) func() {
	prev := profilDInstrument
	profilDInstrument = p
	return func() { profilDInstrument = prev }
}

// poserBasculeDInstrument pose une ou plusieurs bascules de grammaire sur le profil du harnais
// et rend sa restauration. C est ce que les douze reglages publics de `filmdec` faisaient avant
// le lot 2.3, mais borne aux fichiers de test.
func poserBasculeDInstrument(f func(*GrammaireBalayage)) func() {
	prev := profilDInstrument
	f(&profilDInstrument.Grammaire)
	return func() { profilDInstrument = prev }
}

// poserLargeurDInstrument pose (largeur >= 0) ou retire (largeur < 0) une largeur de saut sur le
// profil du harnais : `calibree` remplace le deserialiseur d un composant, `bouchon` donne une
// largeur provisoire a un composant non porte.
func poserLargeurDInstrument(genre, nom string, largeur int) {
	table := &profilDInstrument.Grammaire.LargeursCalibrees
	if genre == "bouchon" {
		table = &profilDInstrument.Grammaire.LargeursBouchon
	}
	if largeur < 0 {
		delete(*table, nom)
		return
	}
	if *table == nil {
		*table = map[string]int{}
	}
	(*table)[nom] = largeur
}
