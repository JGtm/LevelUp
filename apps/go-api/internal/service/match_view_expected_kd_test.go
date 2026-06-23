package service

import (
	"math"
	"testing"
)

// TestOlsPredictAt_PerfectLinear : sur une relation parfaitement linéaire
// kills = 2 + 0.02·durée, la prédiction doit retomber exactement dessus.
func TestOlsPredictAt_PerfectLinear(t *testing.T) {
	var durs, kills []float64
	for d := 300.0; d <= 800.0; d += 50 {
		durs = append(durs, d)
		kills = append(kills, 2+0.02*d) // pente connue
	}
	got := olsPredictAt(durs, kills, 600)
	want := 2 + 0.02*600
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("olsPredictAt = %.4f, want %.4f", got, want)
	}
}

// TestOlsPredictAt_NoVarianceFallsBackToMean : durées identiques → pas de pente
// estimable → retombe sur la moyenne de y.
func TestOlsPredictAt_NoVarianceFallsBackToMean(t *testing.T) {
	durs := []float64{500, 500, 500, 500}
	kills := []float64{8, 12, 10, 14} // moyenne 11
	got := olsPredictAt(durs, kills, 700)
	if math.Abs(got-11) > 1e-9 {
		t.Fatalf("olsPredictAt sans variance = %.4f, want 11 (moyenne)", got)
	}
}

// TestPredictKDFromDuration_MinSamples : sous le seuil (10), ok=false.
func TestPredictKDFromDuration_MinSamples(t *testing.T) {
	durs := []float64{300, 400, 500}
	if _, _, ok := predictKDFromDuration(durs, durs, durs, 600); ok {
		t.Fatal("predictKDFromDuration devrait refuser <10 échantillons")
	}
}

// TestPredictKDFromDuration_ScalesWithDuration : un match plus long doit attendre
// PLUS de frags (le cœur du modèle count∝durée). Plancher 0 respecté.
func TestPredictKDFromDuration_ScalesWithDuration(t *testing.T) {
	var durs, kills, deaths []float64
	for i := 0; i < 30; i++ {
		d := 300.0 + float64(i)*20 // 300..880s
		durs = append(durs, d)
		kills = append(kills, 0.02*d) // 6..17.6
		deaths = append(deaths, 0.015*d)
	}
	ekShort, _, ok := predictKDFromDuration(durs, kills, deaths, 300)
	if !ok {
		t.Fatal("predictKDFromDuration ok=false inattendu")
	}
	ekLong, edLong, ok := predictKDFromDuration(durs, kills, deaths, 900)
	if !ok {
		t.Fatal("predictKDFromDuration ok=false inattendu (long)")
	}
	if ekLong <= ekShort {
		t.Fatalf("frags attendus devraient monter avec la durée : court=%.2f long=%.2f", ekShort, ekLong)
	}
	if edLong < 0 || ekLong < 0 {
		t.Fatalf("attendus négatifs interdits : kills=%.2f deaths=%.2f", ekLong, edLong)
	}
}
