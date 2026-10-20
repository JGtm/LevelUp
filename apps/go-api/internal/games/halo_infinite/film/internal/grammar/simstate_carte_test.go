package grammar

// simstate_carte_test.go — LE GARDE-RAIL DE LA BASCULE `SimStateComplet` (lot 5.3.3-a).
//
// CE QU IL EPINGLE, ET POURQUOI IL EXISTE. La bascule declare `i60 simulation-state`
// entierement decode ; son critere ECRIT est « que le chemin absolu d i0 tire ses trois largeurs
// de la CARTE du match ». Elle etait posee A LA MAIN par des instruments, donc jamais en
// production. Ce fichier fixe les DEUX moities de la regle :
//
//	AVEC carte    les deux portes (constructeur sous catalogue, geste d installation des
//	              largeurs) la levent ;
//	SANS carte    le defaut GLOBAL reste faux — `NewFilmContext` sert les enveloppes
//	              `ScanFilm*(dir)`, ou les largeurs ne viennent pas de la carte, et un defaut
//	              leve y lirait `i60` au-dela de ce que le critere autorise.
//
// UN DEFAUT GLOBAL LEVE EST LA PANNE QUE CE TEST ATTRAPE : elle ne se verrait sur aucun gate de
// compilation, et elle desynchroniserait les instruments sans carte.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// carteDeTestSimState rend une entree de catalogue PORTANT ses largeurs d axe (les valeurs de
// `snowbound`, la carte de `bfecd02b` sur laquelle la levee a ete mesuree).
func carteDeTestSimState() profile.MapQuantEntry {
	return profile.MapQuantEntry{
		Module:     "snowbound-de-test",
		Min:        [3]float32{-64, -64, -16},
		Max:        [3]float32{64, 64, 48},
		AxisWidths: [3]uint{13, 13, 12},
	}
}

func TestSimStateCompletSuitLaCarte(t *testing.T) {
	carte := carteDeTestSimState()
	if !carte.Layout().Valid() {
		t.Fatalf("fixture invalide : le decoupage de l entree de test doit etre valide, %+v",
			carte.Layout())
	}
	sansLargeurs := carte
	sansLargeurs.AxisWidths = [3]uint{}

	cas := []struct {
		nom  string
		bal  ProfilDeBalayage
		veut bool
	}{
		{"defaut global", ProfilDeBalayageParDefaut(), false},
		{"contexte sans carte", NewFilmContext(nil).ProfilDeBalayage(), false},
		{"contexte sous catalogue, carte sans largeurs",
			NewFilmContextForMap(nil, &sansLargeurs, nil).ProfilDeBalayage(), false},
		{"contexte sous catalogue, entree nulle",
			NewFilmContextForMap(nil, nil, nil).ProfilDeBalayage(), false},
		{"contexte sous catalogue, carte avec largeurs",
			NewFilmContextForMap(nil, &carte, nil).ProfilDeBalayage(), true},
	}
	for _, c := range cas {
		if got := c.bal.Grammaire.SimStateComplet; got != c.veut {
			t.Errorf("%s : SimStateComplet = %v, attendu %v", c.nom, got, c.veut)
		}
	}
}

// TestSimStateCompletSuitLeGesteDesLargeurs epingle la SECONDE porte : le geste qui installe les
// largeurs de la carte sur un profil deja construit — celui de `killsource.ProfilDeDepartPourCarte`
// et de `replay.installWorldObjectPrecision`. Sans lui, la bascule posee a la construction du
// contexte serait EFFACEE en production : `replay.poserProfilPuisCarte` remplace le profil ENTIER
// par celui que `killsource` a calibre, et ce profil-la n est pas passe par le constructeur.
func TestSimStateCompletSuitLeGesteDesLargeurs(t *testing.T) {
	carte := carteDeTestSimState()

	p := ProfilDeBalayageParDefaut()
	if p.Grammaire.SimStateComplet {
		t.Fatalf("le profil par defaut ne doit pas porter la bascule")
	}
	p.PoserLargeursObjetDuMondeDepuisDecoupage(carte.Layout())
	if !p.Grammaire.SimStateComplet {
		t.Errorf("largeurs de carte installees : SimStateComplet attendu vrai")
	}

	// DECOUPAGE NON DETECTE : le geste garde le defaut plutot que d installer des zeros, et il
	// ne doit donc RIEN declarer non plus.
	q := ProfilDeBalayageParDefaut()
	q.PoserLargeursObjetDuMondeDepuisDecoupage(profile.I0Layout{})
	if q.Grammaire.SimStateComplet {
		t.Errorf("decoupage nul : SimStateComplet attendu faux")
	}
}
