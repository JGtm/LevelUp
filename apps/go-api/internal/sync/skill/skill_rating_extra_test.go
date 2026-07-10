// Package sync — skill_rating_extra_test.go : tests unitaires des fonctions LUSR pures.
//
// Couvre applyInactivityDecay, computeCompositeScore, estimateIndividualMU,
// computeEnemyStrength, GetLUSRChain (dispatcher), GetTierForRating,
// FormatTierLabel, ClampF, sigmoidRatio.
package skill

import (
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// applyInactivityDecay
// ─────────────────────────────────────────────────────────────────────────────

func TestApplyInactivityDecay_BelowThreshold(t *testing.T) {
	sigma := 100.0
	got := applyInactivityDecay(sigma, 0.5)
	if got != sigma {
		t.Errorf("below threshold: got %v, want %v (unchanged)", got, sigma)
	}
}

func TestApplyInactivityDecay_AtThreshold(t *testing.T) {
	sigma := 100.0
	got := applyInactivityDecay(sigma, InactivityThresholdDay)
	if got != sigma {
		t.Errorf("at threshold: got %v, want %v", got, sigma)
	}
}

func TestApplyInactivityDecay_AboveThreshold(t *testing.T) {
	sigma := 100.0
	got := applyInactivityDecay(sigma, 5.0)
	expected := sigma + InactivitySigmaPerDay*(5.0-InactivityThresholdDay)
	if math.Abs(got-expected) > 0.01 {
		t.Errorf("5 days: got %v, want %v", got, expected)
	}
}

func TestApplyInactivityDecay_MaxDays(t *testing.T) {
	sigma := 100.0
	got := applyInactivityDecay(sigma, 100.0) // bien au-delà du max
	maxExpected := sigma + InactivitySigmaPerDay*(float64(MaxInactivityDays)-InactivityThresholdDay)
	if got > MaxSigma {
		t.Errorf("should be capped at MaxSigma (%v), got %v", MaxSigma, got)
	}
	if math.Abs(got-maxExpected) > 0.01 {
		t.Errorf("capped at max: got %v, want %v", got, maxExpected)
	}
}

func TestApplyInactivityDecay_NeverBelowMinSigma(t *testing.T) {
	got := applyInactivityDecay(MinSigma, 0)
	if got < MinSigma {
		t.Errorf("got %v < MinSigma %v", got, MinSigma)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// computeCompositeScore
// ─────────────────────────────────────────────────────────────────────────────

func TestCompositeScore_AllZeros(t *testing.T) {
	row := &compositeMatchRow{}
	got := computeCompositeScore(row, nil, nil, nil, nil, nil, nil)
	if got != 0.5 {
		t.Errorf("all zeros: got %v, want 0.5", got)
	}
}

func TestCompositeScore_ClearWin(t *testing.T) {
	outcome := 2 // WIN
	row := &compositeMatchRow{
		Kills:          20,
		Deaths:         5,
		KillsExpected:  10,
		DeathsExpected: 10,
		Outcome:        &outcome,
		DamageDealt:    5000,
		DamageTaken:    2000,
		Accuracy:       55.0,
	}
	avgAcc := 45.0
	avgDE := 0.5
	got := computeCompositeScore(row, &avgAcc, nil, &avgDE, nil, nil, nil)
	if got <= 0.5 {
		t.Errorf("clear win with above-average stats: got %v, want > 0.5", got)
	}
}

func TestCompositeScore_ClearLoss(t *testing.T) {
	outcome := 3 // LOSS
	row := &compositeMatchRow{
		Kills:          3,
		Deaths:         15,
		KillsExpected:  10,
		DeathsExpected: 10,
		Outcome:        &outcome,
		DamageDealt:    1000,
		DamageTaken:    5000,
		Accuracy:       25.0,
	}
	avgAcc := 45.0
	avgDE := 0.5
	got := computeCompositeScore(row, &avgAcc, nil, &avgDE, nil, nil, nil)
	if got >= 0.5 {
		t.Errorf("clear loss with below-average stats: got %v, want < 0.5", got)
	}
}

func TestCompositeScore_InRange(t *testing.T) {
	outcome := 2
	row := &compositeMatchRow{
		Kills: 10, Deaths: 10, KillsExpected: 10, DeathsExpected: 10,
		Outcome: &outcome, DamageDealt: 3000, DamageTaken: 3000, Accuracy: 45,
	}
	avgAcc := 45.0
	got := computeCompositeScore(row, &avgAcc, nil, nil, nil, nil, nil)
	if got < 0 || got > 1 {
		t.Errorf("score out of [0,1]: %v", got)
	}
}

func TestCompositeScore_DNF(t *testing.T) {
	outcome := 4 // DNF
	row := &compositeMatchRow{
		Kills: 5, Deaths: 5, KillsExpected: 10, DeathsExpected: 10,
		Outcome: &outcome, DamageDealt: 2000, DamageTaken: 2000,
	}
	got := computeCompositeScore(row, nil, nil, nil, nil, nil, nil)
	if got < 0 || got > 1 {
		t.Errorf("DNF score out of range: %v", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// estimateIndividualMU
// ─────────────────────────────────────────────────────────────────────────────

func TestEstimateIndividualMU_ZeroStd(t *testing.T) {
	got := estimateIndividualMU(10, 10, 0, 1500)
	if got != 1500 {
		t.Errorf("zero std: got %v, want 1500", got)
	}
}

func TestEstimateIndividualMU_AboveAvg(t *testing.T) {
	got := estimateIndividualMU(15, 10, 3, 1500)
	if got <= 1500 {
		t.Errorf("above average KE should raise MU, got %v", got)
	}
}

func TestEstimateIndividualMU_BelowAvg(t *testing.T) {
	got := estimateIndividualMU(5, 10, 3, 1500)
	if got >= 1500 {
		t.Errorf("below average KE should lower MU, got %v", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// computeEnemyStrength
// ─────────────────────────────────────────────────────────────────────────────

func TestComputeEnemyStrength_EmptyKEs(t *testing.T) {
	mu, sigma := computeEnemyStrength(nil, 10, 3, 1500)
	if mu != 1500 {
		t.Errorf("empty: muOpp = %v, want 1500", mu)
	}
	if sigma != DefaultOpponentSigma {
		t.Errorf("empty: sigmaOpp = %v, want %v", sigma, DefaultOpponentSigma)
	}
}

func TestComputeEnemyStrength_SingleKE(t *testing.T) {
	mu, sigma := computeEnemyStrength([]float64{12}, 10, 3, 1500)
	if mu <= 1499 {
		t.Errorf("single KE above avg: muOpp = %v, should be near or above 1500", mu)
	}
	if sigma != DefaultOpponentSigma {
		t.Errorf("single KE: sigmaOpp = %v, want %v", sigma, DefaultOpponentSigma)
	}
}

func TestComputeEnemyStrength_MultipleKEs(t *testing.T) {
	kes := []float64{5, 10, 15, 20}
	mu, sigma := computeEnemyStrength(kes, 10, 5, 1500)
	_ = mu
	if sigma <= 0 || sigma > DefaultOpponentSigma {
		t.Errorf("sigma should be in (0, %v], got %v", DefaultOpponentSigma, sigma)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetLUSRChain
// ─────────────────────────────────────────────────────────────────────────────

func TestGetLUSRChain(t *testing.T) {
	cases := []struct {
		pairName string
		want     string
	}{
		// Exclus
		{"Ranked:Slayer on Aquarius", ""},
		{"Ranked:CTF on Recharge", ""},
		{"Firefight:King of the Hill on Argyle", ""},
		{"Gruntpocalypse:Slayer on Deadlock", ""},
		// BTB
		{"BTB:Slayer on Fragmentation", LUSRChainBTB},
		{"BTB Heavies:CTF on Highpower", LUSRChainBTB},
		// Chaos — catégorie Fiesta/SuperFiesta/HuskyRaid
		{"Fiesta:Slayer on Bazaar", LUSRChainChaos},
		{"Super Fiesta:Slayer on Catalyst", LUSRChainChaos},
		{"Husky Raid:CTF on Pharaoh", LUSRChainChaos},
		{"Super Husky Raid:CTF on Pharaoh", LUSRChainChaos},
		{"Castle Wars", LUSRChainChaos},
		// Chaos — Other avec keywords
		{"Infection:Slayer on Bazaar", LUSRChainChaos},
		{"Griffball", LUSRChainChaos},
		{"Rocket Hog Race:BTB on Highpower", LUSRChainChaos},
		{"Action Sack:Slayer on Recharge", LUSRChainChaos},
		{"Event:Last Spartan Standing on Fragmentation", LUSRChainChaos},
		// arena_slayer — Other fallback
		{"Rumble Pit:Slayer on Bazaar", LUSRChainArenaSlayer},
		{"Custom:Unknown on MapX", LUSRChainArenaSlayer},
		// arena_slayer — Assassin
		{"Arena:Slayer on Bazaar", LUSRChainArenaSlayer},
		{"Arena:Team Slayer on Bazaar", LUSRChainArenaSlayer},
		{"Arena:Attrition on Live Fire", LUSRChainArenaSlayer},
		{"Arena:Elimination on Bazaar", LUSRChainArenaSlayer},
		{"Tactical:Slayer on Recharge", LUSRChainArenaSlayer},
		{"Community:Team Slayer on Solution", LUSRChainArenaSlayer},
		// arena_objectif — Assassin
		{"Arena:CTF on Recharge", LUSRChainArenaObjectif},
		{"Arena:Neutral Flag CTF on Live Fire", LUSRChainArenaObjectif},
		{"Arena:One Flag CTF on Highpower", LUSRChainArenaObjectif},
		{"Arena:Strongholds on Streets", LUSRChainArenaObjectif},
		{"Arena:Oddball on Aquarius", LUSRChainArenaObjectif},
		{"Arena:King of the Hill on Catalyst", LUSRChainArenaObjectif},
		{"Arena:Total Control on Fragmentation", LUSRChainArenaObjectif},
		{"Arena:Land Grab on Recharge", LUSRChainArenaObjectif},
		{"Arena:Extraction on Live Fire", LUSRChainArenaObjectif},
		{"Arena:Stockpile on Deadlock", LUSRChainArenaObjectif},
		{"BTB:CTF on Highpower", LUSRChainBTB}, // CTF dans BTB → btb (pas arena_objectif)
		// pair_name vide → arena_slayer (fallback safe)
		{"", LUSRChainArenaSlayer},
	}
	for _, tc := range cases {
		t.Run(tc.pairName, func(t *testing.T) {
			got := GetLUSRChain(tc.pairName)
			if got != tc.want {
				t.Errorf("GetLUSRChain(%q) = %q, want %q", tc.pairName, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GetTierForRating / FormatTierLabel
// ─────────────────────────────────────────────────────────────────────────────

func TestGetTierForRating_Table(t *testing.T) {
	cases := []struct {
		rating   float64
		wantTier string
		wantSub  int
	}{
		{1000, "Bronze", 1},
		{1100, "Bronze", 4},
		{1200, "Silver", 1},
		{1500, "Gold", 4},
		{1750, "Platinum", 5},
		{1900, "Diamond", 4},
		{2100, "Onyx", 0},
	}
	for _, c := range cases {
		tier, sub := GetTierForRating(c.rating)
		if tier == nil {
			t.Errorf("rating %v: got nil tier", c.rating)
			continue
		}
		if tier.Name != c.wantTier {
			t.Errorf("rating %v: tier = %q, want %q", c.rating, tier.Name, c.wantTier)
		}
		if sub != c.wantSub {
			t.Errorf("rating %v: sub = %d, want %d", c.rating, sub, c.wantSub)
		}
	}
}

func TestGetTierForRating_BelowMin(t *testing.T) {
	tier, _ := GetTierForRating(500)
	if tier != nil {
		t.Error("rating 500 should be nil tier")
	}
}

func TestFormatTierLabel_Bronze(t *testing.T) {
	got := FormatTierLabel(1050)
	if got != "Bronze II" {
		t.Errorf("got %q, want Bronze II", got)
	}
}

func TestFormatTierLabel_Onyx(t *testing.T) {
	got := FormatTierLabel(2100)
	if got != "Onyx" {
		t.Errorf("got %q, want Onyx", got)
	}
}

func TestFormatTierLabel_Unranked(t *testing.T) {
	got := FormatTierLabel(100)
	if got != "Non classé" {
		t.Errorf("got %q, want Non classé", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ClampF / sigmoidRatio
// (containsI/toLowerASCII déplacés dans skillchain — MT-15)
// ─────────────────────────────────────────────────────────────────────────────

func TestClampF(t *testing.T) {
	cases := []struct {
		v, lo, hi, want float64
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 10, 0},
	}
	for _, c := range cases {
		got := ClampF(c.v, c.lo, c.hi)
		if got != c.want {
			t.Errorf("ClampF(%v, %v, %v) = %v, want %v", c.v, c.lo, c.hi, got, c.want)
		}
	}
}

func TestSigmoidRatio_Normal(t *testing.T) {
	got := sigmoidRatio(10, 10) // ratio=1 → sigmoid=0.5
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("sigmoidRatio(10,10) = %v, want ~0.5", got)
	}
}

func TestSigmoidRatio_ZeroDenom(t *testing.T) {
	got := sigmoidRatio(10, 0)
	if got != 0.5 {
		t.Errorf("zero denom: got %v, want 0.5", got)
	}
}

func TestSigmoidRatio_HighRatio(t *testing.T) {
	got := sigmoidRatio(100, 1) // ratio=100 → near 1.0
	if got < 0.95 {
		t.Errorf("high ratio: got %v, want > 0.95", got)
	}
}
