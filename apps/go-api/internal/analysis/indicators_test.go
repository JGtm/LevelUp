// Package analysis — indicators_test.go : tests table-driven exhaustifs sur
// les indicateurs canoniques (ADR 0006).
//
// Politique transverse : tests de non-régression nominatifs. Toute modification
// d'un seuil métier (PerfTier) ou d'une formule (CombatEfficiency/KDR/WinRate/
// Accuracy) requiert ré-évaluation de ce fichier ET du miroir front
// (instances.test.ts).
package analysis

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// ─── CombatEfficiency ───────────────────────────────────────────────────

// CombatEfficiency = (kills+assists)/max(1,deaths) — métrique interne du perf
// score, PAS le KDA affiché (qui est un net (k+a/3)−d, possiblement négatif).
func TestCombatEfficiency(t *testing.T) {
	cases := []struct {
		name    string
		k, a, d int
		want    float64
	}{
		{"happy path 5/2/3 = 7/3", 5, 2, 3, 7.0 / 3.0},
		{"deaths=0 -> floor at 1", 5, 2, 0, 7.0},
		{"deaths=1", 5, 2, 1, 7.0},
		{"all zero", 0, 0, 0, 0.0},
		{"only deaths", 0, 0, 5, 0.0},
		{"high deaths", 1, 0, 100, 0.01},
		{"reference 15/4/6 = 19/6 ~3.17", 15, 4, 6, 19.0 / 6.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CombatEfficiency(tc.k, tc.a, tc.d)
			if !almostEqual(got, tc.want) {
				t.Errorf("CombatEfficiency(%d,%d,%d) = %v, want %v", tc.k, tc.a, tc.d, got, tc.want)
			}
		})
	}
}

// ─── KDR ────────────────────────────────────────────────────────────────

func TestKDR(t *testing.T) {
	cases := []struct {
		name string
		k, d int
		want float64
	}{
		{"happy path 10/5", 10, 5, 2.0},
		{"deaths=0 -> floor at 1", 10, 0, 10.0},
		{"deaths=1", 10, 1, 10.0},
		{"all zero", 0, 0, 0.0},
		{"only deaths", 0, 5, 0.0},
		{"sub-unitary 4/8", 4, 8, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := KDR(tc.k, tc.d)
			if !almostEqual(got, tc.want) {
				t.Errorf("KDR(%d,%d) = %v, want %v", tc.k, tc.d, got, tc.want)
			}
		})
	}
}

// ─── WinRate ────────────────────────────────────────────────────────────

func TestWinRate(t *testing.T) {
	cases := []struct {
		name        string
		wins, total int
		want        float64
	}{
		{"happy path 50/100", 50, 100, 0.5},
		{"100% wins", 10, 10, 1.0},
		{"0% wins", 0, 10, 0.0},
		{"total=0 -> 0", 5, 0, 0.0},
		{"total negative -> 0", 5, -1, 0.0},
		{"55%", 55, 100, 0.55},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WinRate(tc.wins, tc.total)
			if !almostEqual(got, tc.want) {
				t.Errorf("WinRate(%d,%d) = %v, want %v", tc.wins, tc.total, got, tc.want)
			}
		})
	}
}

// ─── Accuracy ───────────────────────────────────────────────────────────

func TestAccuracy(t *testing.T) {
	cases := []struct {
		name        string
		hits, fired int
		want        float64
	}{
		{"happy path 42/100", 42, 100, 0.42},
		{"100% hits", 10, 10, 1.0},
		{"0% hits", 0, 10, 0.0},
		{"fired=0 -> 0", 5, 0, 0.0},
		{"fired negative -> 0", 5, -1, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Accuracy(tc.hits, tc.fired)
			if !almostEqual(got, tc.want) {
				t.Errorf("Accuracy(%d,%d) = %v, want %v", tc.hits, tc.fired, got, tc.want)
			}
		})
	}
}

// ─── KillsPerGame / DeathsPerGame ───────────────────────────────────────

func TestKillsPerGame(t *testing.T) {
	cases := []struct {
		name           string
		kills, matches int
		want           float64
	}{
		{"happy path 100/10", 100, 10, 10.0},
		{"matches=0 -> 0", 5, 0, 0.0},
		{"sub-unitary", 1, 4, 0.25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := KillsPerGame(tc.kills, tc.matches)
			if !almostEqual(got, tc.want) {
				t.Errorf("KillsPerGame(%d,%d) = %v, want %v", tc.kills, tc.matches, got, tc.want)
			}
		})
	}
}

func TestDeathsPerGame(t *testing.T) {
	cases := []struct {
		name            string
		deaths, matches int
		want            float64
	}{
		{"happy path 80/10", 80, 10, 8.0},
		{"matches=0 -> 0", 5, 0, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeathsPerGame(tc.deaths, tc.matches)
			if !almostEqual(got, tc.want) {
				t.Errorf("DeathsPerGame(%d,%d) = %v, want %v", tc.deaths, tc.matches, got, tc.want)
			}
		})
	}
}

// ─── PerfTier — table de vérité complète ────────────────────────────────

// Miroir Vitest : apps/web/src/lib/accessibility/scales/__tests__/instances.test.ts
// (TestPerfScale_TruthTable). Toute modification doit synchroniser les 2.
func TestPerfTier_TruthTable(t *testing.T) {
	cases := []struct {
		name  string
		score float64
		want  Tier
	}{
		{"100 = TierExcellent", 100, TierExcellent},
		{"85 = TierExcellent (Excellent)", 85, TierExcellent},
		{"80 = TierExcellent (borne basse exacte)", 80, TierExcellent},
		{"79.9 = TierBon", 79.9, TierBon},
		{"70 = TierBon (Bon)", 70, TierBon},
		{"65 = TierBon (borne basse exacte)", 65, TierBon},
		{"64.9 = TierCorrect", 64.9, TierCorrect},
		{"55 = TierCorrect (Correct)", 55, TierCorrect},
		{"50 = TierCorrect (borne basse exacte)", 50, TierCorrect},
		{"49.9 = TierFaible", 49.9, TierFaible},
		{"40 = TierFaible (Faible)", 40, TierFaible},
		{"35 = TierFaible (borne basse exacte)", 35, TierFaible},
		{"34.9 = TierMauvais", 34.9, TierMauvais},
		{"20 = TierMauvais (Mauvais)", 20, TierMauvais},
		{"0 = TierMauvais", 0, TierMauvais},
		{"-10 = TierMauvais (negative ok)", -10, TierMauvais},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PerfTier(tc.score)
			if got != tc.want {
				t.Errorf("PerfTier(%v) = %v, want %v", tc.score, got, tc.want)
			}
		})
	}
}

func TestTier_Token(t *testing.T) {
	cases := []struct {
		tier Tier
		want string
	}{
		{TierExcellent, "perf-tier-1"},
		{TierBon, "perf-tier-2"},
		{TierCorrect, "perf-tier-3"},
		{TierFaible, "perf-tier-4"},
		{TierMauvais, "perf-tier-5"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.tier.Token()
			if got != tc.want {
				t.Errorf("Tier(%d).Token() = %q, want %q", tc.tier, got, tc.want)
			}
		})
	}
}
