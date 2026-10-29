package replay

// zone_states_gauge_descent_test.go — L ALLEGEMENT DE LA JAUGE MESURE UNE VARIATION, PAS UNE
// MONTEE (constat RB1-8 de l audit du 2026-09-24, lot J9.7).
//
// LE DEFAUT : la clause des 0,02 comparait `m - lastM` a son seuil. Une DESCENTE donne une
// difference negative, toujours sous le seuil : elle n etait publiee qu au bout d une seconde (la
// clause de l ecart) ou au dernier point de la fenetre. L escalier du client tenait donc la valeur
// HAUTE pendant la seconde ou la jauge redescendait. Le meme allegement sert la jauge de RETOUR DU
// DRAPEAU (`flag_return_gauge.go`), dont la valeur redescend quand les defenseurs sortent de la zone.

import "testing"

// TestZoneGaugeDescentePublieeDansLaSeconde — LE POINT DU LOT. Une descente de 0,05 une frame
// apres le point precedent est une variation de plus de 0,02 : elle est publiee a SA frame.
//
// MUTATION : `m-lastM < zoneGaugeMinDeltaMilli` (la difference signee) dans `appendGaugeThinned`
// rougit ce test.
func TestZoneGaugeDescentePublieeDansLaSeconde(t *testing.T) {
	ss := []zoneSample{
		{t: 0, v: gaugeQ(500)}, {t: 1, v: gaugeQ(600)}, {t: 2, v: gaugeQ(550)},
		{t: 3, v: gaugeQ(560)}, {t: 4, v: gaugeQ(700)},
	}
	wins := []zoneGaugeWindow{{t0: 0, t1: 4}}
	pts, _ := appendGaugeThinned(nil, ss, 0, 4, zoneGaugeGapFrames(100))
	checkGaugeSeries(t, pts, wins)
	veut := []GaugePoint{{T: 0, V: 0.5}, {T: 1, V: 0.6}, {T: 2, V: 0.55}, {T: 4, V: 0.7}}
	if len(pts) != len(veut) {
		t.Fatalf("points %v, attendu %v — la descente de 0,05 a la frame 2 doit etre publiee", pts, veut)
	}
	for i := range veut {
		if pts[i] != veut[i] {
			t.Errorf("point %d : %+v, attendu %+v (serie %v)", i, pts[i], veut[i], pts)
		}
	}
}
