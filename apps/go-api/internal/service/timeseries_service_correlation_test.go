package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// ---------------------------------------------------------------------------
// buildCorrelationPoints (types étendus)
// ---------------------------------------------------------------------------

func TestBuildCorrelationPoints_AllTypes(t *testing.T) {
	win := 2
	acc := 0.55
	dur := 600
	kda := 2.5
	mmrTeam := 1500.0
	mmrEnemy := 1600.0
	matches := []legacymatch.StatsMatchRow{
		{
			Kills:             10,
			Deaths:            5,
			Assists:           2,
			Outcome:           &win,
			Accuracy:          &acc,
			TimePlayedSeconds: &dur,
			KDA:               &kda,
			TeamMMR:           &mmrTeam,
			EnemyMMR:          &mmrEnemy,
		},
	}
	points := buildCorrelationPoints(context.Background(), matches)
	if len(points) == 0 {
		t.Fatal("expected non-empty correlation points")
	}
	// P7.1 : Label composite remplacé par MetricXKey + MetricYKey.
	pairs := make(map[string]bool)
	for _, p := range points {
		pairs[p.MetricXKey+"_vs_"+p.MetricYKey] = true
	}
	for _, want := range []string{"kills_vs_kd_ratio", "kills_vs_deaths", "lifespan_vs_kills"} {
		if !pairs[want] {
			t.Errorf("missing pair %q", want)
		}
	}
}

// TestBuildCorrelationPoints_LifespanPrefersRealValue couvre D-09 (V721-14a) :
// buildCorrelationPoints calculait son propre proxy time_played/(morts+1) au lieu
// de réutiliser matchAvgLifeSeconds (même helper que buildLifeBuckets) — l'histogramme
// et le nuage racontaient deux histoires différentes de la même métrique. La valeur
// RÉELLE (avg_life_seconds) doit l'emporter sur le proxy quand elle est disponible.
func TestBuildCorrelationPoints_LifespanPrefersRealValue(t *testing.T) {
	real := 12.3 // très éloignée du proxy (600/(9+1) = 60) pour détecter un retour au proxy
	tp := 600
	matches := []legacymatch.StatsMatchRow{
		{Kills: 4, Deaths: 9, AvgLifeSeconds: &real, TimePlayedSeconds: &tp},
	}
	points := buildCorrelationPoints(context.Background(), matches)
	got := findCorrelationPair(t, points, "lifespan", tsMetricKeyKills)
	if got.XValue != 12.3 {
		t.Errorf("lifespan: want valeur RÉELLE 12.3, got %v (proxy aurait donné 60)", got.XValue)
	}
}

// TestBuildCorrelationPoints_LifespanFallbackWhenRealMissing couvre le repli : sans
// AvgLifeSeconds (matchs antérieurs à la colonne), le proxy historique reste utilisé.
func TestBuildCorrelationPoints_LifespanFallbackWhenRealMissing(t *testing.T) {
	tp := 600
	matches := []legacymatch.StatsMatchRow{
		{Kills: 4, Deaths: 9, TimePlayedSeconds: &tp}, // AvgLifeSeconds absent
	}
	points := buildCorrelationPoints(context.Background(), matches)
	got := findCorrelationPair(t, points, "lifespan", tsMetricKeyKills)
	if got.XValue != 60 { // 600 / (9 + 1), proxy historique
		t.Errorf("lifespan repli: want proxy 60, got %v", got.XValue)
	}
}

func findCorrelationPair(t *testing.T, points []domain.CorrelationDataPair, xKey, yKey string) domain.CorrelationDataPair {
	t.Helper()
	for _, p := range points {
		if p.MetricXKey == xKey && p.MetricYKey == yKey {
			return p
		}
	}
	t.Fatalf("missing pair %s_vs_%s", xKey, yKey)
	return domain.CorrelationDataPair{}
}
