package replay

// zone_states_gauge_test.go — LA JAUGE EN DIRECT (schema 17), sur des enregistrements CONSTRUITS :
// la serie est ALLEGEE comme ecrit (un point par variation >= 0,02, OU par seconde de rampe, ET le
// DERNIER point de chaque rampe toujours — trois tests, un par clause, qui ECHOUENT si la clause
// est retiree), elle est monotone en T, dans [0, 1], et VIDE hors rampe ; elle sort du calque en
// Bastion, et JAMAIS sur une colline (KOTH).
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
// (une par frame de 100 ms, un pas de 1/60 d'unite, soit 0,0167 < 0,02) ne publie qu'un
// point sur deux : 31, premier et dernier compris. Retirer le seuil de variation en publierait 61
// — c'est exactement ce que ce test refuse.
func TestZoneGaugeSerieAllegeeParVariation(t *testing.T) {
	ss := gaugeSamples(61, 100, 1, zoneGaugeQuantZero, zoneGaugeQuantUnit/60)
	wins := []zoneGaugeWindow{{t0: 100, t1: 160}}
	pts := zoneGaugeSeriesOf(ss, wins, zoneGaugeGapFrames(100))
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
// toutes les 0,5 s, un pas de 0,005 d'unite) ne franchirait le seuil de variation que tous
// les 2 s ; la clause de la seconde publie un point toutes les deux emissions : 11 sur 21, et
// jamais plus d'une seconde entre deux points. Retirer la clause en publierait 6.
func TestZoneGaugeSerieReEmetChaqueSeconde(t *testing.T) {
	ss := gaugeSamples(21, 0, 5, zoneGaugeQuantZero, zoneGaugeQuantUnit/200)
	wins := []zoneGaugeWindow{{t0: 0, t1: 100}}
	gap := zoneGaugeGapFrames(100)
	pts := zoneGaugeSeriesOf(ss, wins, gap)
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

// TestZoneGaugeDernierPointToujoursPublie — LA CLAUSE DU DERNIER POINT (`!last` dans
// appendGaugeWindow). Le SOMMET d'une rampe se gagne souvent par un pas COURT, et les deux clauses
// d'allegement l'ecarteraient : sur `7344d24f` (zone 1), la jauge monte de 0,028 toutes les deux
// frames jusqu'a 0,976, puis atteint 0,991 la frame SUIVANTE — 0,015 de variation (sous 0,02) une
// frame apres le point precedent (sous la seconde). Sans `!last`, l'arc s'arreterait a 0,976 et le
// sommet reellement atteint ne serait jamais publie ; le film le porte, l'oeil le lit.
//
// Le test verifie AUSSI que le dernier pas reste sous LES DEUX seuils : sans cette garde, une
// retouche des valeurs rendrait le cas vacant (le point passerait par la clause de variation) et
// la clause `!last` redeviendrait non couverte sans que rien ne rougisse.
func TestZoneGaugeDernierPointToujoursPublie(t *testing.T) {
	gap := zoneGaugeGapFrames(100)
	ss := []zoneSample{
		{t: 1060, v: gaugeQ(805)}, {t: 1062, v: gaugeQ(833)}, {t: 1064, v: gaugeQ(862)},
		{t: 1066, v: gaugeQ(891)}, {t: 1068, v: gaugeQ(919)}, {t: 1070, v: gaugeQ(948)},
		{t: 1072, v: gaugeQ(976)}, {t: 1073, v: gaugeQ(991)},
	}
	wins := []zoneGaugeWindow{{t0: 1060, t1: 1073}}
	pts := zoneGaugeSeriesOf(ss, wins, gap)
	checkGaugeSeries(t, pts, wins)
	if len(pts) != len(ss) {
		t.Fatalf("%d points publies pour %d emissions, attendu %d (chaque pas de 0,028 franchit le "+
			"seuil, et le SOMMET est publie meme sous les seuils) : %v", len(pts), len(ss), len(ss), pts)
	}
	last, avant := pts[len(pts)-1], pts[len(pts)-2]
	if last != (GaugePoint{T: 1073, V: 0.991}) {
		t.Errorf("dernier point %+v, attendu le SOMMET {1073 0.991} — sans la clause `!last` la serie "+
			"s'arreterait a %+v : %v", last, avant, pts)
	}
	if d := last.V - avant.V; d >= float32(zoneGaugeMinDeltaMilli)/zoneGaugeMilli {
		t.Errorf("le dernier pas vaut %.4f, au-dessus du seuil de variation : le cas ne prouve plus "+
			"rien sur la clause `!last` (la clause de variation suffirait)", d)
	}
	if d := last.T - avant.T; d >= gap {
		t.Errorf("le dernier pas suit de %d frames (gap = %d) : le cas ne prouve plus rien sur la "+
			"clause `!last` (la clause de la seconde suffirait)", d, gap)
	}
}

// TestZoneGaugeRienHorsRampe : les emissions hors des fenetres ne sont JAMAIS publiees, et une
// fenetre publie toujours son depart et son sommet.
func TestZoneGaugeRienHorsRampe(t *testing.T) {
	ss := gaugeSamples(31, 90, 1, zoneGaugeQuantZero, 2_000)
	wins := []zoneGaugeWindow{{t0: 100, t1: 104}, {t0: 110, t1: 112}}
	pts := zoneGaugeSeriesOf(ss, wins, zoneGaugeGapFrames(100))
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
	if zoneGaugeSeriesOf(ss, nil, 10) != nil {
		t.Errorf("sans rampe, la serie doit etre nil")
	}
	if zoneGaugeSeriesOf(nil, wins, 10) != nil {
		t.Errorf("sans emission, la serie doit etre nil")
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
	// La zone 0 (slot 10) : deux rampes de trois emissions (0,001, 0,2, puis 0,9 et 0,95 sur
	// l'echelle du jeu). Trois points par rampe (depart, milieu, sommet), et le sommet de la
	// seconde (0,95) est le meme que le `progress` de son intervalle.
	got := states[0].Gauge
	want := []GaugePoint{{T: 96, V: 0.001}, {T: 98, V: 0.2}, {T: 100, V: 0.9},
		{T: 296, V: 0.001}, {T: 298, V: 0.2}, {T: 300, V: 0.95}}
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
	if want := gaugeProgressOf(gaugeQ(950)); top == nil || *top != want {
		t.Errorf("progress de l'intervalle a 101 = %v, attendu %v (le sommet de la seconde rampe)", top, want)
	}
}

// TestZoneStatesCollineNePublieAucuneJauge — LE CALQUE, en KOTH : la colline active ne porte
// AUCUNE serie de jauge, et la couverture le dit (gaugePoints = 0). Le tag 3 y est un compteur de
// transfert d'environ une seconde, pas la progression de garde (lot C-ter volet 1) : une serie
// montrerait un arc qui se remplit en une seconde a chaque prise — credible et faux.
func TestZoneStatesCollineNePublieAucuneJauge(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900)...)
	reads = append(reads, zoneRampAt(40, 400, 900)...)
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
	for _, st := range states {
		if st.Gauge != nil {
			t.Errorf("colline %d : serie de jauge publiee (%d points) — aucune n'est attendue en KOTH",
				st.ZoneRef, len(st.Gauge))
		}
	}
	if cov.GaugePoints != 0 {
		t.Errorf("gaugePoints = %d, attendu 0 sur un film a colline", cov.GaugePoints)
	}
}

// TestZoneGaugeRetourAZeroFermeLaRampe — LE POINT DE FIN. La premiere emission qui suit une rampe,
// si elle est un retour au zero du jeu, est publiee a sa frame avec v = 0 : c'est ainsi que le
// client sait que la capture est finie (aboutie ou abandonnee) et non figee. Une emission qui
// suit en MONTANT (capture qui reprend apres un blocage) n'ajoute rien ; un retour a zero deja
// depart de la rampe suivante n'entre pas deux fois.
func TestZoneGaugeRetourAZeroFermeLaRampe(t *testing.T) {
	gap := zoneGaugeGapFrames(100)
	// Une rampe 0,001 -> 0,5 (frames 10..14), abandonnee : retour a zero 14 frames plus tard.
	ss := gaugeSamples(5, 10, 1, gaugeQ(1), zoneGaugeQuantUnit/8)
	ss = append(ss, zoneSample{t: 28, v: zoneGaugeQuantZero})
	pts := zoneGaugeSeriesOf(ss, []zoneGaugeWindow{{t0: 10, t1: 14}}, gap)
	if n := len(pts); n < 3 || pts[n-1] != (GaugePoint{T: 28, V: 0}) {
		t.Fatalf("serie %v : attendu un dernier point {28 0}, le retour a zero qui ferme la rampe", pts)
	}
	// La meme rampe, FIGEE puis reprise : l'emission suivante est plus haute, rien n'est ajoute.
	reprise := gaugeSamples(5, 10, 1, gaugeQ(1), zoneGaugeQuantUnit/8)
	reprise = append(reprise, zoneSample{t: 300, v: gaugeQ(600)})
	pts = zoneGaugeSeriesOf(reprise, []zoneGaugeWindow{{t0: 10, t1: 14}}, gap)
	if n := len(pts); pts[n-1].T != 14 {
		t.Fatalf("serie %v : aucun point ne doit suivre le sommet 14 quand la jauge reprend plus haut", pts)
	}
	// Deux rampes que separe un retour a zero qui est aussi le DEPART de la seconde : un seul point.
	deux := gaugeSamples(5, 10, 1, gaugeQ(1), zoneGaugeQuantUnit/8)
	deux = append(deux, zoneSample{t: 15, v: zoneGaugeQuantZero})
	deux = append(deux, gaugeSamples(4, 40, 1, gaugeQ(200), zoneGaugeQuantUnit/8)...)
	pts = zoneGaugeSeriesOf(deux, []zoneGaugeWindow{{t0: 10, t1: 14}, {t0: 15, t1: 43}}, gap)
	zeros := 0
	for _, p := range pts {
		if p.T == 15 {
			zeros++
			if p.V != 0 {
				t.Errorf("le point de la frame 15 vaut %v, attendu 0 (retour a zero)", p.V)
			}
		}
	}
	if zeros != 1 {
		t.Errorf("%d point(s) a la frame 15, attendu 1 : %v", zeros, pts)
	}
	checkGaugeSeries(t, pts, []zoneGaugeWindow{{t0: 10, t1: 14}, {t0: 15, t1: 43}})
}
