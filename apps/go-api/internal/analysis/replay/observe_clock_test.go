package replay

// observe_clock_test.go — LE CHRONOMETRE DES BALAYAGES NE CHANGE RIEN A CE QUI EST OBSERVE.
//
// La duree par balayage (PLAN_CUISSON_PERF §3 D5) se lit dans l'ECART entre deux appels a
// l'observateur, sans ajouter un site d'appel a BuildFromFilm. Le risque de cette economie est
// ailleurs : que la mesure s'immisce dans ce que l'observateur voit. Ces cas verrouillent donc
// la STRUCTURE — memes etapes, memes valeurs, meme ordre, horloge qui avance sur les balayages
// et seulement sur eux. AUCUNE DUREE N'EST TESTEE : un seuil de temps en test est un test qui
// rougit sur une machine chargee.

import (
	"slices"
	"testing"
	"time"
)

// TestObserveClockNAltereNiEtapesNiValeurs : avec une horloge armee, l'observateur recoit
// exactement ce qu'il recevrait sans elle.
func TestObserveClockNAltereNiEtapesNiValeurs(t *testing.T) {
	var vues []string
	var valeurs []any
	o := Options{Observe: func(step string, v any) {
		vues = append(vues, step)
		valeurs = append(valeurs, v)
	}}
	o.clock = &stepClock{last: time.Now()}

	attendues := []string{"positions", "heldWeaponChanges", "heldWeaponChanges.stats", "clockOrigin"}
	for i, step := range attendues {
		o.observe(step, i)
	}
	if !slices.Equal(vues, attendues) {
		t.Fatalf("etapes observees = %v, attendu %v — l'horloge a modifie la sequence", vues, attendues)
	}
	for i := range valeurs {
		if valeurs[i] != i {
			t.Fatalf("valeur de l'etape %q = %v, attendu %d", attendues[i], valeurs[i], i)
		}
	}
}

// TestObserveClockAvanceHorsStats : l'horloge avance sur un balayage, et PAS sur la seconde
// sortie du meme balayage (`.stats`) — sinon le balayage suivant serait mesure depuis un
// instant qui n'est celui d'aucun travail.
func TestObserveClockAvanceHorsStats(t *testing.T) {
	// Une date FRANCHEMENT anterieure : deux `time.Now()` consecutifs peuvent rendre le meme
	// instant sur une horloge a faible resolution, et le test dirait alors n'importe quoi.
	avant := time.Now().Add(-time.Hour)
	o := Options{clock: &stepClock{last: avant}}

	o.observe("positions", nil)
	apres := o.clock.last
	if !apres.After(avant) {
		t.Fatalf("l'horloge n'a pas avance sur un balayage (%v -> %v)", avant, apres)
	}
	o.observe("heldWeaponChanges.stats", nil)
	if !o.clock.last.Equal(apres) {
		t.Fatalf("l'horloge a avance sur une etape .stats (%v -> %v)", apres, o.clock.last)
	}
}

// TestObserveSansClockNeMesureRien : le champ est nil hors BuildFromFilm (BuildFromPositions,
// tests, appelants directs) — aucun appel ne doit paniquer pour autant.
func TestObserveSansClockNeMesureRien(t *testing.T) {
	var vues int
	o := Options{Observe: func(string, any) { vues++ }}
	if o.clock != nil {
		t.Fatal("clock devrait etre nil sur des Options non armees")
	}
	o.observe("positions", nil)
	o.observe("positions.stats", nil)
	if vues != 2 {
		t.Fatalf("%d etape(s) observee(s), attendu 2", vues)
	}
}
