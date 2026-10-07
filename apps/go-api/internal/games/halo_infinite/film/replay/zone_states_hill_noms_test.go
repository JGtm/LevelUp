package replay

// zone_states_hill_noms_test.go — LE DESIGNATEUR DE COLLINE ET LE PREMIER CONTACT AVEC L OBJET DE
// MODE, DESIGNES PAR LE NOM (zone_states_hill.go), sur des series CONSTRUITES.

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// serieColline monte deux candidats designateurs : le slot 50 (deux bascules), dont le VOISIN ne
// parle qu une fois, et le slot 70 (trois bascules), dont le voisin parle deux fois — la regle de
// voisinage elirait 70. L objet de mode du slot 50 a son proprietaire au slot 60 (premier contact
// a la frame 10) et sa jauge au slot 62, loin de lui.
func serieColline() zoneSeries {
	ser := serieTemoin(map[uint32][]zoneSample{})
	ser.desig[50] = []zoneSample{{t: 100, v: 7}, {t: 200, v: 8}}
	ser.desig[70] = []zoneSample{{t: 300, v: 1}, {t: 301, v: 2}, {t: 302, v: 3}}
	ser.owner[51] = []zoneSample{{t: 5, v: 0}}
	ser.owner[60] = []zoneSample{{t: 10, v: zoneNeutralOwner}, {t: 150, v: 1}}
	ser.gauge[62] = []zoneSample{{t: 40, v: zoneGaugeQuantZero}}
	ser.owner[71] = []zoneSample{{t: 2, v: 0}, {t: 250, v: 1}}
	return ser
}

// TestCollineDesignateurParLeNom : le nom de cle designe le slot 50, que la regle de voisinage
// ecarte ; le premier contact se date sur les slots NOMMES de son bloc (60 a la frame 10), pas sur
// ses voisins (51 a la frame 5).
func TestCollineDesignateurParLeNom(t *testing.T) {
	ser := serieColline()
	ser.noms = nomsDe(map[uint32]uint32{50: blocA.cle, 60: blocA.proprietaire, 62: blocA.jauge})
	d, ok := hillDesignatorOf(ser)
	if !ok || d.slot != 50 || d.parVoisinage {
		t.Fatalf("designateur %+v (%v) : attendu le slot 50 par le nom", d, ok)
	}
	if d.first != 10 {
		t.Errorf("premier contact a la frame %d, attendu 10 (proprietaire nomme du bloc)", d.first)
	}
}

// TestCollineDesignateurParVoisinageFauteDeNom : sans nom au vocabulaire, la regle de voisinage
// repond (slot 70, premier contact a la frame 2 sur son voisin), et le dit.
func TestCollineDesignateurParVoisinageFauteDeNom(t *testing.T) {
	d, ok := hillDesignatorOf(serieColline())
	if !ok || d.slot != 70 || !d.parVoisinage || d.first != 2 {
		t.Fatalf("designateur %+v (%v) : attendu le slot 70 par voisinage, contact a 2", d, ok)
	}
}

// TestCollineDesignateurParVoisinageSeCompte : le repli se compte sur la cuisson.
func TestCollineDesignateurParVoisinageSeCompte(t *testing.T) {
	in, c := hillDesignatorCase(true)
	fb := fallback.NouveauCompteur()
	c.fb = fb
	if _, cov := buildZoneStates(context.Background(), in, c); cov.Method != ZoneMethodDesignator {
		t.Fatalf("methode %q, attendu %q", cov.Method, ZoneMethodDesignator)
	}
	if n := fb.Compte(fallback.NomCollineDesignateurParVoisinage); n != 1 {
		t.Fatalf("repli compte %d fois, attendu 1", n)
	}
}
