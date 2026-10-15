package grammar

// jeu_darmes_refuse_test.go — LE RETOUR FAUX DE `FUN_1406d01fc` ARRETE LA BOUCLE D ETAT COMPLET
// (revue D1.4.6, constat 2 ; [arreterSiJeuDArmesRefuse]).
//
// Mutations jouees le 2026-10-09, chacune rouge puis retiree : la garde de portee retiree (le jeu
// refuse arrete aussi hors de la portee) ; la comparaison `Principal == Second` retiree.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// jeuDArmesEcrit rend les bits d un jeu d armes : R(3) de demande, puis deux `FUN_1406d00ec`
// (R(1) a 0 puis R(2) la valeur ; R(1) a 1 pour -1).
func jeuDArmesEcrit(principal, second int) []byte {
	var w bitWriter
	w.bits(5, 3)
	for _, v := range []int{principal, second} {
		if v < 0 {
			w.bit(1)
			continue
		}
		w.bit(0)
		w.bits(uint64(v), 2) //nolint:gosec // 0..3
	}
	w.bits(0, 8) // bourrage
	return w.buf
}

func TestLeJeuDArmesRefuseArreteLaBoucleDEtatComplet(t *testing.T) {
	cas := []struct {
		nom               string
		principal, second int
		portee            bool
		arret             bool
	}{
		{"meme emplacement des deux mains, sous la portee", 1, 1, true, true},
		{"meme emplacement des deux mains, hors portee", 1, 1, false, false},
		{"ambidextrie normale, sous la portee", 0, 1, true, false},
		{"aucun emplacement, sous la portee", -1, -1, true, false},
		{"seconde main seule, sous la portee", -1, 2, true, false},
	}
	for _, c := range cas {
		br := LecteurSur(jeuDArmesEcrit(c.principal, c.second))
		if c.portee {
			sousLaPortee(br)
		}
		var publie bool
		br.obs = &Observation{DesiredWeaponSetHook: func(JeuDArmes) { publie = true }}
		consumeBipedDesiredWeaponSet(br)
		if got := br.arret == lecture.ArretJeuDArmesRefuse; got != c.arret {
			t.Errorf("%s : arret %v, attendu %v", c.nom, got, c.arret)
		}
		if publie == c.arret {
			t.Errorf("%s : publication %v alors que l arret vaut %v", c.nom, publie, c.arret)
		}
		if nomDeLArret(lecture.ArretJeuDArmesRefuse) != "jeu_d_armes_refuse" {
			t.Errorf("nom de la cause : %q", nomDeLArret(lecture.ArretJeuDArmesRefuse))
		}
	}
}
