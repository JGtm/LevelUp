// Package analysis — elevation_cloud_test.go : le nuage « distance × dénivelé » (D25, T5).
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. LE SIGNE, AUX DEUX BORNES. C'est le point le plus facile à inverser de tout le chantier :
//     côté tueur `DeltaZM` vaut le dénivelé brut, côté victime son OPPOSÉ, pour que les deux
//     côtés répondent à la MÊME question (« moi, étais-je au-dessus ? »).
//  2. AUCUNE PERTE : un point par frag mesuré, pas de plafond, pas d'échantillonnage.
//  3. Les quantiles d'un côté ne regardent JAMAIS l'autre côté.
//  4. Un côté vide rend un résumé à `N == 0` — et pas un quartile inventé.
package analysis

import (
	"math"
	"testing"
)

const epsCloud = 1e-9

func elevKill(match string, ms int64, weapon string, side Side, dist, dz float64) MeasuredKill {
	return MeasuredKill{MatchID: match, TimeMS: ms, WeaponKey: weapon, Side: side, DistanceM: dist, DeltaZ: dz}
}

// TestElevationCloudSigneParCote — le contrat du signe, aux deux bornes.
func TestElevationCloudSigneParCote(t *testing.T) {
	cloud := BuildElevationCloud([]MeasuredKill{
		elevKill("m1", 10, "br", SideKiller, 12, 3.5),
		elevKill("m1", 20, "ar", SideVictim, 8, 3.5),
	})
	if len(cloud.Kills) != 1 || len(cloud.Deaths) != 1 {
		t.Fatalf("un point par cote attendu, got %d/%d", len(cloud.Kills), len(cloud.Deaths))
	}
	if got := cloud.Kills[0].DeltaZM; math.Abs(got-3.5) > epsCloud {
		t.Errorf("cote tueur : dz brut attendu 3.5, got %v", got)
	}
	// Le MÊME dénivelé brut, lu côté victime, vaut son opposé : le tueur était au-dessus,
	// donc MOI, la victime, j'étais en dessous.
	if got := cloud.Deaths[0].DeltaZM; math.Abs(got+3.5) > epsCloud {
		t.Errorf("cote victime : dz oppose attendu -3.5, got %v", got)
	}
}

// TestElevationCloudAucunePerte — un point par frag, aucun plafond.
func TestElevationCloudAucunePerte(t *testing.T) {
	in := make([]MeasuredKill, 0, 2000)
	for i := 0; i < 2000; i++ {
		in = append(in, elevKill("m1", int64(i), "br", SideKiller, float64(i%40), float64(i%7)-3))
	}
	cloud := BuildElevationCloud(in)
	if len(cloud.Kills) != 2000 {
		t.Fatalf("2000 points attendus (aucun echantillonnage), got %d", len(cloud.Kills))
	}
	if cloud.KillsSummary.N != 2000 {
		t.Errorf("N doit compter tous les points, got %d", cloud.KillsSummary.N)
	}
}

// TestElevationCloudQuantilesParCote — les quartiles d'un côté ignorent l'autre.
func TestElevationCloudQuantilesParCote(t *testing.T) {
	in := []MeasuredKill{}
	for _, d := range []float64{10, 20, 30, 40, 50} {
		in = append(in, elevKill("m1", int64(d), "br", SideKiller, d, 1))
	}
	// Les morts sont volontairement à des distances bien plus grandes : si les quartiles des
	// frags bougeaient, c'est que les deux côtés auraient été mélangés.
	for _, d := range []float64{100, 200, 300} {
		in = append(in, elevKill("m1", int64(d), "ar", SideVictim, d, -2))
	}
	cloud := BuildElevationCloud(in)
	k := cloud.KillsSummary
	if math.Abs(k.DistanceP25-20) > epsCloud || math.Abs(k.DistanceP50-30) > epsCloud ||
		math.Abs(k.DistanceP75-40) > epsCloud {
		t.Errorf("quartiles de distance des frags : 20/30/40 attendus, got %v/%v/%v",
			k.DistanceP25, k.DistanceP50, k.DistanceP75)
	}
	if math.Abs(k.DeltaZP50-1) > epsCloud {
		t.Errorf("mediane de denivele des frags : 1 attendu, got %v", k.DeltaZP50)
	}
	if got := cloud.DeathsSummary.DeltaZP50; math.Abs(got-2) > epsCloud {
		// dz brut -2 côté victime => +2 de mon point de vue (j'étais au-dessus).
		t.Errorf("mediane de denivele des morts : +2 attendu (signe pose), got %v", got)
	}
	if cloud.DeathsSummary.N != 3 {
		t.Errorf("N des morts : 3 attendu, got %d", cloud.DeathsSummary.N)
	}
}

// TestElevationCloudCoteVide — pas de quartile inventé sur un côté sans mesure.
func TestElevationCloudCoteVide(t *testing.T) {
	cloud := BuildElevationCloud([]MeasuredKill{elevKill("m1", 1, "br", SideKiller, 10, 0)})
	if len(cloud.Deaths) != 0 || cloud.DeathsSummary.N != 0 {
		t.Fatalf("cote sans mesure : vide attendu, got %d points / N=%d",
			len(cloud.Deaths), cloud.DeathsSummary.N)
	}
	if cloud.DeathsSummary.DistanceP50 != 0 || cloud.DeathsSummary.DeltaZP50 != 0 {
		t.Errorf("cote vide : resume a zero attendu, got %+v", cloud.DeathsSummary)
	}
}

// TestElevationCloudOrdreDeterministe — deux appels, le même tableau.
func TestElevationCloudOrdreDeterministe(t *testing.T) {
	in := []MeasuredKill{
		elevKill("m2", 5, "br", SideKiller, 1, 0),
		elevKill("m1", 9, "ar", SideKiller, 2, 0),
		elevKill("m1", 3, "br", SideKiller, 3, 0),
	}
	cloud := BuildElevationCloud(in)
	want := []struct {
		match string
		ms    int64
	}{{"m1", 3}, {"m1", 9}, {"m2", 5}}
	for i, w := range want {
		if cloud.Kills[i].MatchID != w.match || cloud.Kills[i].TimeMS != w.ms {
			t.Fatalf("point %d : %s/%d attendu, got %s/%d", i, w.match, w.ms,
				cloud.Kills[i].MatchID, cloud.Kills[i].TimeMS)
		}
	}
	// L'entrée n'est pas mutée : l'appelant la réutilise pour la portée par arme.
	if in[0].MatchID != "m2" {
		t.Errorf("l'entree a ete triee en place, got %s en tete", in[0].MatchID)
	}
}
