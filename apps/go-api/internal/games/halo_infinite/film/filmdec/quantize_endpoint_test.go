package filmdec

import (
	"math"
	"testing"
)

// TestDequantEndpointExtremities — LE DÉTECTEUR. Si l'une de ces trois égalités n'est pas
// EXACTE (pas « proche »), la formule portée de FUN_1406d84b4 est fausse et aucune valeur
// de santé/bouclier ne doit être publiée.
func TestDequantEndpointExtremities(t *testing.T) {
	cases := []struct {
		name                string
		q                   uint64
		min, max            float32
		w                   uint
		excl, endpointExact bool
		want                float32
	}{
		// SANTÉ i4 : W=8, excl=1 -> levels=255, endpointExact=1.
		{"sante q=0 -> min exact", 0, VitalityBodyMin, VitalityBodyMax, 8, true, true, -1},
		{"sante q=levels-1=254 -> max exact", 254, VitalityBodyMin, VitalityBodyMax, 8, true, true, 1},
		{"sante q=127 -> 0.0 EXACT (point milieu)", 127, VitalityBodyMin, VitalityBodyMax, 8, true, true, 0},
		// BOUCLIER i5 : W=8, excl=0 -> levels=256, endpointExact=1.
		{"bouclier q=0 -> 0 exact", 0, VitalityShieldMin, VitalityShieldMax, 8, false, true, 0},
		{"bouclier q=levels-1=255 -> 4.0 exact", 255, VitalityShieldMin, VitalityShieldMax, 8, false, true, 4},
		// HORLOGE : W=16, excl=0, endpointExact=1, [0, 36000].
		{"horloge q=0 -> 0", 0, 0, RoundTimerMax, 16, false, true, 0},
		{"horloge q=65535 -> 36000", 65535, 0, RoundTimerMax, 16, false, true, 36000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DequantEndpoint(c.q, c.min, c.max, c.w, c.excl, c.endpointExact)
			if got != c.want { // égalité EXACTE voulue : pas de tolérance
				t.Fatalf("DequantEndpoint(q=%d) = %v (bits %08x), attendu %v exact",
					c.q, got, math.Float32bits(got), c.want)
			}
		})
	}
}

// TestDequantEndpointMidpointNeedsTheExclRule prouve que la règle du point milieu n'est PAS
// redondante en flottant : sans elle, la branche endpointExact seule rate 0.0 pour la santé.
// C'est la justification de la ligne `if excl && 2*q == levels-1` — sans ce test, un futur
// lecteur pourrait la supprimer en la croyant algébriquement inutile (elle l'est en réels,
// pas en float32).
func TestDequantEndpointMidpointNeedsTheExclRule(t *testing.T) {
	const q, levels = 127, 255
	step2 := (VitalityBodyMax - VitalityBodyMin) / float32(levels-2)
	naive := float32(q-1)*step2 + (step2*0.5 + VitalityBodyMin) // branche endpointExact seule
	if naive == 0 {
		t.Skip("cette plateforme rend déjà 0 exact sans la règle du milieu — test sans objet")
	}
	if got := DequantEndpoint(q, VitalityBodyMin, VitalityBodyMax, 8, true, true); got != 0 {
		t.Fatalf("la règle du milieu ne s'applique pas : got %v", got)
	}
	t.Logf("sans la règle du milieu : %v (bits %08x) — non nul, d'où la branche en double",
		naive, math.Float32bits(naive))
}

// TestDequantEndpointNonExactPathMatchesMidBucket : quand endpointExact est à 0, la formule
// dégénère en « milieu de bucket », la même que quantize.go — c'est le cas de la fraction
// R(12) de weapon-state-ammo (min=0, max=1). Vérifie qu'on n'a pas mélangé les deux modes.
func TestDequantEndpointNonExactPathMatchesMidBucket(t *testing.T) {
	const w = 12
	for _, q := range []uint64{0, 1, 2047, 4095} {
		step := float32(1) / float32(uint64(1)<<w)
		want := (float32(q)*step + 0) + step*0.5
		if got := DequantEndpoint(q, 0, 1, w, false, false); got != want {
			t.Fatalf("q=%d : got %v, attendu %v (milieu de bucket)", q, got, want)
		}
	}
}

// TestDequantEndpointMonotone : la déquantification doit être strictement croissante en q
// sur tout le domaine. Un ordre cassé signalerait une erreur de branche (par exemple le
// point milieu appliqué hors de sa condition).
func TestDequantEndpointMonotone(t *testing.T) {
	check := func(name string, min, max float32, w uint, excl, exact bool, last uint64) {
		t.Helper()
		prev := DequantEndpoint(0, min, max, w, excl, exact)
		for q := uint64(1); q <= last; q++ {
			v := DequantEndpoint(q, min, max, w, excl, exact)
			if !(v > prev) {
				t.Fatalf("%s : q=%d rend %v <= %v (q=%d)", name, q, v, prev, q-1)
			}
			prev = v
		}
		if prev != max {
			t.Fatalf("%s : le dernier quantum rend %v, attendu %v", name, prev, max)
		}
	}
	check("sante", VitalityBodyMin, VitalityBodyMax, 8, true, true, 254)
	check("bouclier", VitalityShieldMin, VitalityShieldMax, 8, false, true, 255)
}

// TestDequantEndpointHealthFractionSemantics documente (et fige) la conversion en fraction
// de vie affichée : la plage sérialisée est [-1, +1] et la barre affiche clamp(v, 0, 1).
func TestDequantEndpointHealthFractionSemantics(t *testing.T) {
	cases := map[uint64]float32{0: 0, 127: 0, 254: 1}
	for q, want := range cases {
		if got := HealthFraction(DequantEndpoint(q, VitalityBodyMin, VitalityBodyMax, 8, true, true)); got != want {
			t.Fatalf("HealthFraction(q=%d) = %v, attendu %v", q, got, want)
		}
	}
	if got := HealthFraction(DequantEndpoint(190, VitalityBodyMin, VitalityBodyMax, 8, true, true)); got <= 0 || got >= 1 {
		t.Fatalf("q=190 devrait donner une fraction strictement intérieure, got %v", got)
	}
}
