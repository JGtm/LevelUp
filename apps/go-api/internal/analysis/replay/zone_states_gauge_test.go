package replay

// zone_states_gauge_test.go — LA JAUGE EN DIRECT (schema 17), sur des enregistrements CONSTRUITS :
// la serie est ALLEGEE comme ecrit (un point par variation >= 0,02 OU par seconde de rampe — deux
// tests, un par clause, qui ECHOUENT si la clause est retiree), elle est monotone en T, dans
// [0, 1], et VIDE hors rampe ; et elle sort bien du calque, en Bastion comme en KOTH.
//
// Les fabriques partagees vivent dans `zone_states_test.go`.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// gaugeSamples fabrique `n` emissions aux frames t0 + i*pas, de valeur brute v0 + i*dv.
func gaugeSamples(n, t0, pas int, v0, dv uint64) []zoneSample {
	out := make([]zoneSample, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, zoneSample{t: t0 + i*pas, v: v0 + uint64(i)*dv})
	}
	return out
}

// checkGaugeSeries verifie les invariants de TOUTE serie publiee : T strictement croissant, V
// dans [0, 1], et chaque point DANS une fenetre.
func checkGaugeSeries(t *testing.T, pts []GaugePoint, wins []zoneGaugeWindow) {
	t.Helper()
	for i, p := range pts {
		if i > 0 && p.T <= pts[i-1].T {
			t.Errorf("point %d : T=%d n'est pas strictement apres T=%d", i, p.T, pts[i-1].T)
		}
		if p.V < 0 || p.V > 1 {
			t.Errorf("point %d : V=%v hors de [0, 1]", i, p.V)
		}
		dedans := false
		for _, w := range wins {
			if p.T >= w.t0 && p.T <= w.t1 {
				dedans = true
			}
		}
		if !dedans {
			t.Errorf("point %d : T=%d hors de toute rampe — rien ne doit etre publie hors rampe", i, p.T)
		}
	}
}

// TestZoneGaugeSerieAllegeeParVariation — LA CLAUSE DES 0,02. Une rampe rapide de 61 emissions
// (une par frame de 100 ms, un pas de 1/60 de l'echelle, soit 0,0167 < 0,02) ne publie qu'un
// point sur deux : 31, premier et dernier compris. Retirer le seuil de variation en publierait 61
// — c'est exactement ce que ce test refuse.
func TestZoneGaugeSerieAllegeeParVariation(t *testing.T) {
	ss := gaugeSamples(61, 100, 1, 0, 1)
	scale := zoneGaugeOf(ss)
	wins := []zoneGaugeWindow{{t0: 100, t1: 160}}
	pts := zoneGaugeSeriesOf(ss, scale, wins, zoneGaugeGapFrames(100))
	if len(pts) != 31 {
		t.Fatalf("%d points publies pour 61 emissions, attendu 31 (un point par variation >= 0,02) : %v",
			len(pts), pts)
	}
	checkGaugeSeries(t, pts, wins)
	if pts[0].T != 100 || pts[0].V != 0 {
		t.Errorf("premier point %+v, attendu le DEPART de la rampe {100 0}", pts[0])
	}
	if last := pts[len(pts)-1]; last.T != 160 || last.V != 1 {
		t.Errorf("dernier point %+v, attendu le SOMMET de la rampe {160 1}", last)
	}
	for i := 1; i < len(pts); i++ {
		if d := pts[i].V - pts[i-1].V; d < 0.02-1e-6 {
			t.Errorf("points %d et %d : variation %.4f sous 0,02", i-1, i, d)
		}
	}
}

// TestZoneGaugeSerieReEmetChaqueSeconde — LA CLAUSE DE LA SECONDE. Une rampe LENTE (une emission
// toutes les 0,5 s, un pas de 0,005 de l'echelle) ne franchirait le seuil de variation que tous
// les 2 s ; la clause de la seconde publie un point toutes les deux emissions : 11 sur 21, et
// jamais plus d'une seconde entre deux points. Retirer la clause en publierait 6.
func TestZoneGaugeSerieReEmetChaqueSeconde(t *testing.T) {
	ss := gaugeSamples(21, 0, 5, 0, 10)
	// L'ECHELLE EST CELLE DE LA ZONE, PAS DE CETTE RAMPE : une autre montee du meme slot est
	// allee bien plus haut, et cette rampe-ci n'en parcourt qu'un dixieme.
	scale := zoneGauge{low: 0, high: 2000, seen: true}
	wins := []zoneGaugeWindow{{t0: 0, t1: 100}}
	gap := zoneGaugeGapFrames(100)
	pts := zoneGaugeSeriesOf(ss, scale, wins, gap)
	if len(pts) != 11 {
		t.Fatalf("%d points publies pour 21 emissions, attendu 11 (un point par seconde de rampe) : %v",
			len(pts), pts)
	}
	checkGaugeSeries(t, pts, wins)
	for i := 1; i < len(pts); i++ {
		if pts[i].T-pts[i-1].T > gap {
			t.Errorf("points %d et %d : %d frames sans point, plus d'une seconde", i-1, i,
				pts[i].T-pts[i-1].T)
		}
	}
}

// TestZoneGaugeRienHorsRampe : les emissions hors des fenetres ne sont JAMAIS publiees, et une
// fenetre publie toujours son depart et son sommet.
func TestZoneGaugeRienHorsRampe(t *testing.T) {
	ss := gaugeSamples(31, 90, 1, 0, 100)
	scale := zoneGaugeOf(ss)
	wins := []zoneGaugeWindow{{t0: 100, t1: 104}, {t0: 110, t1: 112}}
	pts := zoneGaugeSeriesOf(ss, scale, wins, zoneGaugeGapFrames(100))
	checkGaugeSeries(t, pts, wins)
	want := []int{100, 101, 102, 103, 104, 110, 111, 112}
	if len(pts) != len(want) {
		t.Fatalf("%d points, attendu %d (%v) : %v", len(pts), len(want), want, pts)
	}
	for i, p := range pts {
		if p.T != want[i] {
			t.Errorf("point %d : T=%d, attendu %d", i, p.T, want[i])
		}
	}
	if zoneGaugeSeriesOf(ss, scale, nil, 10) != nil {
		t.Errorf("sans rampe, la serie doit etre nil")
	}
	if zoneGaugeSeriesOf(ss, zoneGauge{}, wins, 10) != nil {
		t.Errorf("sans echelle (jauge jamais vue), la serie doit etre nil")
	}
}

// TestZoneGaugeTStrictementCroissant : deux emissions sur la MEME frame ne rendent qu'un point
// (le dernier), et un point anterieur au precedent est ecarte — l'escalier ne revient jamais en
// arriere.
func TestZoneGaugeTStrictementCroissant(t *testing.T) {
	var out []GaugePoint
	out = pushGaugePoint(out, GaugePoint{T: 10, V: 0.1})
	out = pushGaugePoint(out, GaugePoint{T: 10, V: 0.2})
	out = pushGaugePoint(out, GaugePoint{T: 12, V: 0.3})
	out = pushGaugePoint(out, GaugePoint{T: 11, V: 0.9})
	if len(out) != 2 || out[0] != (GaugePoint{T: 10, V: 0.2}) || out[1] != (GaugePoint{T: 12, V: 0.3}) {
		t.Fatalf("serie %v, attendu [{10 0.2} {12 0.3}]", out)
	}
	merged := mergeGaugePoints([]GaugePoint{{T: 5, V: 0.5}, {T: 1, V: 0.1}, {T: 5, V: 0.6}, {T: 3, V: 0.3}})
	if len(merged) != 3 || merged[0].T != 1 || merged[1].T != 3 || merged[2] != (GaugePoint{T: 5, V: 0.6}) {
		t.Fatalf("fusion %v, attendu [{1 0.1} {3 0.3} {5 0.6}]", merged)
	}
}

// TestZoneStatesPublieLaJaugeEnDirect — LE CALQUE, en Bastion : chaque zone publiee porte la
// serie de TOUTES les rampes de son slot de jauge, sur l'echelle de `progress`, et la couverture
// en compte les points.
func TestZoneStatesPublieLaJaugeEnDirect(t *testing.T) {
	in, c := bastionCase()
	states, cov := buildZoneStates(in, c)
	if len(states) != 2 {
		t.Fatalf("%d zone(s) publiee(s), attendu 2", len(states))
	}
	total := 0
	for _, st := range states {
		if len(st.Gauge) == 0 {
			t.Fatalf("zone %d : aucune jauge en direct publiee", st.ZoneRef)
		}
		total += len(st.Gauge)
	}
	if cov.GaugePoints != total {
		t.Errorf("coverage.zones.gaugePoints = %d, attendu %d (la somme des points publies)",
			cov.GaugePoints, total)
	}
	// La zone 0 (slot 10) : deux rampes de trois emissions culminant a 900 000 puis 950 000, sur
	// une echelle [1 000 ; 950 000]. Trois points par rampe (depart, milieu a 0,21, sommet), et
	// le sommet de la seconde vaut 1 — le meme que le `progress` de son intervalle.
	got := states[0].Gauge
	want := []GaugePoint{{T: 96, V: 0}, {T: 98, V: 0.21}, {T: 100, V: 0.947},
		{T: 296, V: 0}, {T: 298, V: 0.21}, {T: 300, V: 1}}
	if len(got) != len(want) {
		t.Fatalf("zone 0 : serie %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("zone 0 : point %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
	checkGaugeSeries(t, got, []zoneGaugeWindow{{t0: 96, t1: 100}, {t0: 296, t1: 300}})
	// Le sommet de l'intervalle [101 ; 300] de la zone 0 (equipe 0, jusqu'a la reprise) EST le
	// dernier point de la jauge (la seconde rampe culmine a 300, juste AVANT la bascule du canal
	// de propriete a 301) : meme echelle, meme valeur.
	var top *float32
	for _, sp := range states[0].Spans {
		if sp.T0 == 101 {
			top = sp.Progress
		}
	}
	if top == nil || *top != 1 {
		t.Errorf("progress de l'intervalle a 101 = %v, attendu 1 (le sommet de la seconde rampe)", top)
	}
}

// TestZoneStatesCollinePublieLaJaugeEnDirect — LE CALQUE, en KOTH : la colline active porte la
// serie de la rampe que la grappe a posee sur elle, et rien de la rampe posee sur une autre.
func TestZoneStatesCollinePublieLaJaugeEnDirect(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900_000)...)
	reads = append(reads, zoneRampAt(40, 400, 900_000)...)
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0)) // zone 1 pendant la premiere montee
	}
	for f := 396; f <= 400; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0)) // zone 0 pendant la seconde
	}
	states, cov := buildZoneStates(in, zoneTestCtx(nil, []Track{track("2533", pts...)}))
	if len(states) != 2 || cov.Method != ZoneMethodPositions {
		t.Fatalf("%d zone(s), methode %q — attendu 2 et %q", len(states), cov.Method, ZoneMethodPositions)
	}
	byRef := map[int][]GaugePoint{}
	for _, st := range states {
		byRef[st.ZoneRef] = st.Gauge
	}
	if g := byRef[1]; len(g) != 3 || g[0].T != 96 || g[2].T != 100 || g[2].V != 1 {
		t.Errorf("colline 1 : serie %v, attendu la premiere rampe [96 ; 100] culminant a 1", g)
	}
	if g := byRef[0]; len(g) != 3 || g[0].T != 396 || g[2].T != 400 || g[2].V != 1 {
		t.Errorf("colline 0 : serie %v, attendu la seconde rampe [396 ; 400] culminant a 1", g)
	}
	if cov.GaugePoints != 6 {
		t.Errorf("gaugePoints = %d, attendu 6", cov.GaugePoints)
	}
}
