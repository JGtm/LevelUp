package analysis

// weapon_range_test.go — l'agrégat de portée, ses bornes et sa convention de signe.
//
// LES BORNES SONT L'ESSENTIEL DE CE FICHIER. Deux points décident du sens produit et se
// trompent silencieusement s'ils ne sont pas épinglés : (1) exactement ±1,0 m est « à
// niveau », pas « au-dessus » ; (2) côté VICTIME, le dénivelé s'inverse — sinon « où je
// meurs, d'en haut ou d'en bas » répondrait la position du tueur.

import (
	"math"
	"testing"
)

const epsPortee = 1e-9

// kilFrag fabrique un frag mesuré ; les tests ne construisent jamais la structure à la main
// pour que l'ordre des champs (distance / dénivelé) ne puisse pas s'inverser à la lecture.
func kilFrag(arme string, side Side, distM, dz float64) MeasuredKill {
	return MeasuredKill{WeaponKey: arme, Side: side, DistanceM: distM, DeltaZ: dz}
}

// serie fabrique n frags de même arme et même côté, distances 1..n, dénivelé nul.
func serie(arme string, side Side, n int) []MeasuredKill {
	out := make([]MeasuredKill, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, kilFrag(arme, side, float64(i), 0))
	}
	return out
}

// TestWeaponRangeAggregateNominalDeuxCotes : la même arme lue des deux côtés donne DEUX
// lignes distinctes, chacune avec ses percentiles.
func TestWeaponRangeAggregateNominalDeuxCotes(t *testing.T) {
	kills := append(serie("br75", SideKiller, 8), serie("br75", SideVictim, 8)...)
	rows, sum := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 2 {
		t.Fatalf("2 lignes attendues (un côté chacune), obtenu %d : %+v", len(rows), rows)
	}
	if sum.BelowThreshold != 0 || sum.MeasuredBelowThreshold != 0 {
		t.Fatalf("aucune arme ne devait être écartée, résumé = %+v", sum)
	}
	for _, r := range rows {
		if r.Measured != 8 {
			t.Fatalf("%s/%s : 8 mesures attendues, obtenu %d", r.WeaponKey, r.Side, r.Measured)
		}
		// distances 1..8 : p10 = 1,7 · médiane = 4,5 · p90 = 7,3 (interpolation linéaire).
		for _, c := range []struct {
			nom  string
			got  float64
			want float64
		}{{"p10", r.P10, 1.7}, {"médiane", r.Median, 4.5}, {"p90", r.P90, 7.3}} {
			if math.Abs(c.got-c.want) > 1e-9 {
				t.Fatalf("%s/%s %s = %v, attendu %v", r.WeaponKey, r.Side, c.nom, c.got, c.want)
			}
		}
		if r.Level != 8 || r.Above != 0 || r.Below != 0 {
			t.Fatalf("%s/%s : dénivelé nul = 8 à niveau, obtenu %+v", r.WeaponKey, r.Side, r)
		}
	}
}

// TestWeaponRangeAggregateSeuilNonAtteint : sous le seuil, la ligne n'est PAS publiée — et le
// résumé la rend connaissable (D9 : jamais un silence).
func TestWeaponRangeAggregateSeuilNonAtteint(t *testing.T) {
	kills := append(serie("br75", SideKiller, 8), serie("sniper", SideKiller, 3)...)
	kills = append(kills, serie("sniper", SideVictim, 2)...)
	rows, sum := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 1 || rows[0].WeaponKey != "br75" {
		t.Fatalf("seul br75 devait être publié, obtenu %+v", rows)
	}
	if sum.BelowThreshold != 2 {
		t.Fatalf("2 couples sous le seuil attendus, obtenu %d", sum.BelowThreshold)
	}
	if sum.BelowThresholdBySide[SideKiller] != 1 || sum.BelowThresholdBySide[SideVictim] != 1 {
		t.Fatalf("ventilation par côté attendue 1/1, obtenu %+v", sum.BelowThresholdBySide)
	}
	if sum.MeasuredBelowThreshold != 5 {
		t.Fatalf("5 frags écartés attendus (3 + 2), obtenu %d", sum.MeasuredBelowThreshold)
	}
}

// TestWeaponRangeAggregateUneArme : le cas minimal — une arme, un côté, exactement le seuil.
func TestWeaponRangeAggregateUneArme(t *testing.T) {
	rows, sum := WeaponRangeAggregate(serie("ravager", SideKiller, WeaponRangeMinMeasured),
		WeaponRangeMinMeasured)
	if len(rows) != 1 {
		t.Fatalf("1 ligne attendue, obtenu %d", len(rows))
	}
	if rows[0].Measured != WeaponRangeMinMeasured {
		t.Fatalf("le seuil est FERMÉ : %d mesures doivent publier, obtenu %+v",
			WeaponRangeMinMeasured, rows[0])
	}
	if sum.BelowThreshold != 0 {
		t.Fatalf("rien ne devait être écarté, obtenu %+v", sum)
	}
}

// TestWeaponRangeAggregateDoublons : des frags STRICTEMENT identiques ne se dédupliquent pas
// — deux frags à la même distance avec la même arme sont deux frags, et la médiane d'une
// série constante est cette constante.
func TestWeaponRangeAggregateDoublons(t *testing.T) {
	kills := make([]MeasuredKill, 0, 10)
	for i := 0; i < 10; i++ {
		kills = append(kills, kilFrag("mangler", SideVictim, 6.5, 0.25))
	}
	rows, _ := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 1 || rows[0].Measured != 10 {
		t.Fatalf("10 doublons = 1 ligne de 10 mesures, obtenu %+v", rows)
	}
	r := rows[0]
	if math.Abs(r.P10-6.5) > epsPortee || math.Abs(r.Median-6.5) > epsPortee ||
		math.Abs(r.P90-6.5) > epsPortee {
		t.Fatalf("série constante : les trois percentiles valent 6,5, obtenu %+v", r)
	}
	if r.Level != 10 {
		t.Fatalf("dénivelé 0,25 m = à niveau pour les 10, obtenu %+v", r)
	}
}

// TestWeaponRangeDeniveleBornes : les trois classes, AUX BORNES, des deux côtés.
//
// LE CŒUR DU TEST : +1,0 m et -1,0 m EXACTEMENT sont « à niveau » — il faut DÉPASSER le
// mètre pour changer de classe (D4). Et côté victime le signe s'inverse : le même frag
// physique (`killer_z - victim_z = +2`) est « d'en haut » pour le tueur et « d'en bas » pour
// la victime, parce que la question est toujours « MOI, où étais-je ? ».
func TestWeaponRangeDeniveleBornes(t *testing.T) {
	// Un jeu de 8 dénivelés bruts couvrant les deux bornes et les deux dépassements. Il est
	// VOLONTAIREMENT ASYMÉTRIQUE (trois positifs francs, un seul négatif franc) : sur un jeu
	// symétrique, une inversion de signe oubliée passerait le test sans se voir.
	dzs := []float64{2, 1.0000001, 1.0, 0.5, 0, -1.0, -1.0000001, 5}
	for _, c := range []struct {
		side                Side
		above, level, below int
	}{
		// Côté tueur : dz tel quel. Au-dessus = {2 ; 1,0000001 ; 5} ; en dessous = {-1,0000001} ;
		// à niveau = {1,0 ; 0,5 ; 0 ; -1,0} — les deux bornes exactes y sont.
		{SideKiller, 3, 4, 1},
		// Côté victime : dz INVERSÉ. Les deux comptes s'échangent, « à niveau » ne bouge pas.
		{SideVictim, 1, 4, 3},
	} {
		t.Run(string(c.side), func(t *testing.T) {
			kills := make([]MeasuredKill, 0, len(dzs))
			for _, dz := range dzs {
				kills = append(kills, kilFrag("br75", c.side, 5, dz))
			}
			rows, _ := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
			if len(rows) != 1 {
				t.Fatalf("1 ligne attendue, obtenu %+v", rows)
			}
			r := rows[0]
			if r.Above != c.above || r.Level != c.level || r.Below != c.below {
				t.Fatalf("côté %s : haut/niveau/bas = %d/%d/%d, attendu %d/%d/%d",
					c.side, r.Above, r.Level, r.Below, c.above, c.level, c.below)
			}
			if r.Above+r.Level+r.Below != r.Measured {
				t.Fatalf("les trois classes doivent partitionner l'effectif : %+v", r)
			}
		})
	}
}

// TestWeaponRangeDeniveleCoteVictimeInverse : la convention de signe, isolée et explicite —
// le MÊME frag physique classé des deux côtés doit tomber dans des classes OPPOSÉES.
func TestWeaponRangeDeniveleCoteVictimeInverse(t *testing.T) {
	const brut = 3.0 // killer_z - victim_z : le tueur est 3 m au-dessus de sa victime
	var kills []MeasuredKill
	for i := 0; i < WeaponRangeMinMeasured; i++ {
		kills = append(kills, kilFrag("sniper", SideKiller, 20, brut))
		kills = append(kills, kilFrag("sniper", SideVictim, 20, brut))
	}
	rows, _ := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 2 {
		t.Fatalf("2 lignes attendues, obtenu %+v", rows)
	}
	byside := map[Side]WeaponRange{}
	for _, r := range rows {
		byside[r.Side] = r
	}
	if k := byside[SideKiller]; k.Above != WeaponRangeMinMeasured || k.Below != 0 {
		t.Fatalf("côté tueur : tout doit être « d'en haut », obtenu %+v", k)
	}
	if v := byside[SideVictim]; v.Below != WeaponRangeMinMeasured || v.Above != 0 {
		t.Fatalf("côté victime : le même frag est « d'en bas », obtenu %+v", v)
	}
}

// TestWeaponRangeTriDeterministe : médiane croissante, puis clé d'arme, puis côté.
func TestWeaponRangeTriDeterministe(t *testing.T) {
	var kills []MeasuredKill
	for i := 0; i < WeaponRangeMinMeasured; i++ {
		kills = append(kills, kilFrag("sniper", SideKiller, 30, 0))
		kills = append(kills, kilFrag("shotgun", SideKiller, 3, 0))
		// deux armes à MÉDIANE ÉGALE : c'est la clé puis le côté qui départagent.
		kills = append(kills, kilFrag("br75", SideVictim, 10, 0))
		kills = append(kills, kilFrag("br75", SideKiller, 10, 0))
		kills = append(kills, kilFrag("ak", SideKiller, 10, 0))
	}
	rows, _ := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	got := make([]string, 0, len(rows))
	for _, r := range rows {
		got = append(got, r.WeaponKey+"/"+string(r.Side))
	}
	want := []string{"shotgun/killer", "ak/killer", "br75/killer", "br75/victim", "sniper/killer"}
	if len(got) != len(want) {
		t.Fatalf("ordre attendu %v, obtenu %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordre attendu %v, obtenu %v", want, got)
		}
	}
}

// TestWeaponRangeAggregateEntreeVideEtSeuilAbsurde : aucune entrée = aucune ligne ; un seuil
// nul publie la ligne d'un groupe d'UNE mesure — jamais une ligne sans mesure, un groupe
// naissant toujours d'au moins un frag.
func TestWeaponRangeAggregateEntreeVideEtSeuilAbsurde(t *testing.T) {
	rows, sum := WeaponRangeAggregate(nil, WeaponRangeMinMeasured)
	if len(rows) != 0 || sum.BelowThreshold != 0 {
		t.Fatalf("entrée vide : ni ligne ni écart, obtenu %+v / %+v", rows, sum)
	}
	rows, _ = WeaponRangeAggregate([]MeasuredKill{kilFrag("br75", SideKiller, 4, 0)}, 0)
	if len(rows) != 1 || rows[0].Measured != 1 {
		t.Fatalf("seuil 0 : une mesure publie une ligne, obtenu %+v", rows)
	}
}

// TestWeaponRangeAggregateDistancesNonTriees : LES PERCENTILES EXIGENT UNE SÉRIE TRIÉE, et
// rien ne garantit que les frags arrivent dans l'ordre des distances — ils arrivent dans
// l'ordre du kill-feed. Le groupe reçoit donc ici des distances volontairement mêlées et les
// trois attendus sont calculés À LA MAIN (convention « type 7 », celle de `quantile_cont`) :
//
//	série triée : 2, 4, 7, 9, 12, 15, 30, 40   (n = 8, donc n-1 = 7 intervalles)
//	p10 : rang 0,7  -> 2 + 0,7 x (4 - 2)   = 3,4
//	p50 : rang 3,5  -> 9 + 0,5 x (12 - 9)  = 10,5
//	p90 : rang 6,3  -> 30 + 0,3 x (40 - 30) = 33,0
//
// Sans le tri, les trois valeurs seraient 13,4 · 9,5 · 8,5 : le test rougit à la première
// disparition de `sort.Float64s` (revue du 2026-09-06 — la mutation passait inaperçue).
func TestWeaponRangeAggregateDistancesNonTriees(t *testing.T) {
	var kills []MeasuredKill
	for _, d := range []float64{40, 2, 9, 15, 4, 30, 7, 12} {
		kills = append(kills, kilFrag("sniper", SideKiller, d, 0))
	}
	rows, sum := WeaponRangeAggregate(kills, WeaponRangeMinMeasured)
	if len(rows) != 1 || sum.BelowThreshold != 0 {
		t.Fatalf("une ligne attendue au seuil exact : %+v / %+v", rows, sum)
	}
	for _, c := range []struct {
		nom  string
		got  float64
		want float64
	}{
		{"p10", rows[0].P10, 3.4},
		{"médiane", rows[0].Median, 10.5},
		{"p90", rows[0].P90, 33.0},
	} {
		if math.Abs(c.got-c.want) > epsPortee {
			t.Errorf("%s = %v, attendu %v (série non triée à l'entrée)", c.nom, c.got, c.want)
		}
	}
}

// TestWeaponRangeAggregateNeMutePasLEntree : la fonction est PURE — elle ne réordonne ni ne
// modifie la tranche reçue.
func TestWeaponRangeAggregateNeMutePasLEntree(t *testing.T) {
	kills := []MeasuredKill{
		kilFrag("br75", SideKiller, 9, 1),
		kilFrag("br75", SideKiller, 2, -3),
		kilFrag("sniper", SideKiller, 40, 0),
	}
	avant := append([]MeasuredKill(nil), kills...)
	WeaponRangeAggregate(kills, 1)
	for i := range avant {
		if kills[i] != avant[i] {
			t.Fatalf("entrée mutée au rang %d : %+v -> %+v", i, avant[i], kills[i])
		}
	}
}

// TestPercentileLinearBornes : n=1, n=2, valeurs égales, p=0, p=100, série vide.
func TestPercentileLinearBornes(t *testing.T) {
	for _, c := range []struct {
		nom    string
		sorted []float64
		p      float64
		want   float64
	}{
		{"série vide rend 0", nil, 50, 0},
		{"n=1 p=0", []float64{7}, 0, 7},
		{"n=1 p=50", []float64{7}, 50, 7},
		{"n=1 p=100", []float64{7}, 100, 7},
		{"n=2 p=0", []float64{2, 6}, 0, 2},
		{"n=2 p=50 interpole", []float64{2, 6}, 50, 4},
		{"n=2 p=10 interpole", []float64{2, 6}, 10, 2.4},
		{"n=2 p=90 interpole", []float64{2, 6}, 90, 5.6},
		{"n=2 p=100", []float64{2, 6}, 100, 6},
		{"valeurs égales", []float64{3, 3, 3, 3}, 10, 3},
		{"valeurs égales p90", []float64{3, 3, 3, 3}, 90, 3},
		{"n=4 p=50 entre les deux du milieu", []float64{1, 2, 4, 8}, 50, 3},
		{"p négatif ramené au minimum", []float64{1, 2, 4}, -10, 1},
		{"p au-delà de 100 ramené au maximum", []float64{1, 2, 4}, 150, 4},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := percentileLinear(c.sorted, c.p); math.Abs(got-c.want) > epsPortee {
				t.Fatalf("percentileLinear(%v, %v) = %v, attendu %v", c.sorted, c.p, got, c.want)
			}
		})
	}
}

// TestPercentileLinearMedianeCoincideAvecMedianFloat : la médiane interpolée et le helper
// canonique `MedianFloat` doivent donner le même nombre — sans quoi la page afficherait deux
// médianes différentes de la même série selon le chemin de calcul.
func TestPercentileLinearMedianeCoincideAvecMedianFloat(t *testing.T) {
	for _, s := range [][]float64{{5}, {1, 3}, {1, 2, 3}, {1, 2, 3, 4}, {2, 2, 9, 9, 9}} {
		if got, want := percentileLinear(s, 50), MedianFloat(s); math.Abs(got-want) > epsPortee {
			t.Fatalf("médiane de %v : percentileLinear = %v, MedianFloat = %v", s, got, want)
		}
	}
}
