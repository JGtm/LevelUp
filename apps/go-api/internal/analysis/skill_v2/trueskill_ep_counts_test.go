package skill_v2

import (
	"math"
	"testing"
)

// trueskill_ep_counts_test.go : tests Phase 3c — discrimination intra-squad
// grâce aux kills/deaths comme observations (TS2 §8).
//
// Le test crucial : un match 2v2 où la team A gagne. Player A0 a 20 kills /
// 4 deaths, player A1 a 4 kills / 20 deaths. Sans le signal individuel
// (UpdateTwoTeamEP plain), leurs μ bougent à l'identique (TS classique ne
// peut pas les distinguer). AVEC counts, A0 doit voir un Δμ nettement plus
// grand que A1 — c'est ce qui résoudra le verdict Phase 1d (Madina
// sous-évaluée vs Choco/JGtm sur-évalués).

// pf64 helper pour pointer-to-float64 littéral.
func pf64(v float64) *float64 { return &v }

func TestPhase3c_CountsDiscriminateTeammates(t *testing.T) {
	p := DefaultPriors()

	// 2 teammates ex aequo en priors. Team A bat Team B.
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}

	// Baseline : sans counts.
	noCountsA, _, err := UpdateTwoTeamEP(m, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeamEP baseline: %v", err)
	}
	// Sans counts, A0 et A1 doivent avoir EXACTEMENT le même μ (priors identiques,
	// pas de signal individuel).
	if math.Abs(noCountsA[0].Mu-noCountsA[1].Mu) > 1e-6 {
		t.Errorf("baseline (no counts) : A0=%v, A1=%v devraient être identiques",
			noCountsA[0].Mu, noCountsA[1].Mu)
	}

	// Avec counts : A0 a beaucoup de kills, peu de deaths ; A1 inverse.
	counts := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(20), Deaths: pf64(4)}, // A0 carry
			{Kills: pf64(4), Deaths: pf64(20)}, // A1 passif
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(8), Deaths: pf64(12)},
			{Kills: pf64(8), Deaths: pf64(12)},
		},
	}
	withCountsA, _, err := UpdateTwoTeamWithCountsEP(m, counts, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeamWithCountsEP: %v", err)
	}

	// Le test crucial : A0 (le carry) doit avoir μ STRICTEMENT > A1 (le passif).
	if withCountsA[0].Mu <= withCountsA[1].Mu {
		t.Errorf("DISCRIMINATION FAILED: A0 μ=%v, A1 μ=%v — counts ne départagent pas",
			withCountsA[0].Mu, withCountsA[1].Mu)
	}

	// Sanity : les deux restent au-dessus du prior (l'équipe a gagné).
	if withCountsA[0].Mu <= p.Mu0 {
		t.Errorf("A0 μ=%v <= prior μ0=%v — winning team devrait monter", withCountsA[0].Mu, p.Mu0)
	}
	// A1 peut éventuellement descendre (forte deaths) ou rester ~ priors.

	// Magnitude check : l'écart A0/A1 devrait être > 1.0 sur l'échelle native
	// TS (μ ≈ 25, σ ≈ 8). Sinon le signal n'est pas suffisamment exploité.
	gap := withCountsA[0].Mu - withCountsA[1].Mu
	if gap < 1.0 {
		t.Errorf("écart A0/A1 = %v, attendu > 1.0 (signal kills/deaths trop faible)", gap)
	}
	t.Logf("OK : A0 μ=%.3f σ=%.3f, A1 μ=%.3f σ=%.3f, écart=%.3f",
		withCountsA[0].Mu, withCountsA[0].Sigma, withCountsA[1].Mu, withCountsA[1].Sigma, gap)
}

func TestPhase3c_CountsConsistentDirection(t *testing.T) {
	// Les directions de mise à jour doivent rester cohérentes :
	// - Beaucoup de kills → μ monte plus
	// - Beaucoup de deaths → μ descend plus
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}

	// Variante 1 : A0 nombreux kills, peu deaths
	cHigh := &CountInputs{
		TeamA: []PlayerCounts{{Kills: pf64(30), Deaths: pf64(2)}},
		TeamB: []PlayerCounts{{Kills: pf64(5), Deaths: pf64(15)}},
	}
	high, _, errH := UpdateTwoTeamWithCountsEP(m, cHigh, p)
	if errH != nil {
		t.Fatalf("variante high : %v", errH)
	}

	// Variante 2 : A0 même outcome mais counts inversés (peu kills, beaucoup deaths)
	cLow := &CountInputs{
		TeamA: []PlayerCounts{{Kills: pf64(2), Deaths: pf64(15)}},
		TeamB: []PlayerCounts{{Kills: pf64(5), Deaths: pf64(15)}},
	}
	low, _, errL := UpdateTwoTeamWithCountsEP(m, cLow, p)
	if errL != nil {
		t.Fatalf("variante low : %v", errL)
	}

	if high[0].Mu <= low[0].Mu {
		t.Errorf("A0 avec gros stats (μ=%v) devrait être > A0 avec faibles stats (μ=%v)",
			high[0].Mu, low[0].Mu)
	}
}

func TestPhase3c_CountsNilEquivalentToNoCount(t *testing.T) {
	// counts=nil dans WithCountsEP doit produire EXACTEMENT le même résultat
	// que UpdateTwoTeamEP (pas de count factors ajoutés au graph).
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}

	plainA, plainB, _ := UpdateTwoTeamEP(m, p)
	withNilA, withNilB, _ := UpdateTwoTeamWithCountsEP(m, nil, p)

	for i := range plainA {
		if math.Abs(plainA[i].Mu-withNilA[i].Mu) > 1e-9 {
			t.Errorf("plain[%d].Mu = %v, withNil[%d].Mu = %v", i, plainA[i].Mu, i, withNilA[i].Mu)
		}
		if math.Abs(plainA[i].Sigma-withNilA[i].Sigma) > 1e-9 {
			t.Errorf("plain[%d].Sigma = %v, withNil[%d].Sigma = %v", i, plainA[i].Sigma, i, withNilA[i].Sigma)
		}
	}
	for i := range plainB {
		if math.Abs(plainB[i].Mu-withNilB[i].Mu) > 1e-9 {
			t.Errorf("teamB plain[%d] vs withNil divergent", i)
		}
	}
}

func TestPhase3c_CountsMismatchedLengths_Error(t *testing.T) {
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}

	// counts.TeamA len ≠ m.TeamA len
	bad := &CountInputs{
		TeamA: []PlayerCounts{{Kills: pf64(10)}}, // 1 au lieu de 2
		TeamB: []PlayerCounts{{Kills: pf64(5)}},
	}
	if _, _, err := UpdateTwoTeamWithCountsEP(m, bad, p); err == nil {
		t.Error("expected error on mismatched counts.TeamA length")
	}
}

func TestPhase3c_PartialCounts_OnlyTrackedPlayers(t *testing.T) {
	// Cas réaliste : sur 4 joueurs par équipe, on n'a les counts que pour 1
	// joueur tracké par équipe (l'owner du sync). Les autres ont counts=nil.
	p := DefaultPriors()
	a := []Gaussian{p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState()}
	b := []Gaussian{p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState(), p.NewPlayerState()}
	m := TwoTeamMatch{TeamA: a, TeamB: b, ResultA: TeamWin}

	// Seul A0 et B0 ont des counts ; A1-3 et B1-3 sont nil.
	counts := &CountInputs{
		TeamA: []PlayerCounts{
			{Kills: pf64(25), Deaths: pf64(5)}, // A0 carry connu
			{}, {}, {},                         // A1-3 pas de signal individuel
		},
		TeamB: []PlayerCounts{
			{Kills: pf64(8), Deaths: pf64(15)},
			{}, {}, {},
		},
	}
	postA, _, err := UpdateTwoTeamWithCountsEP(m, counts, p)
	if err != nil {
		t.Fatalf("UpdateTwoTeamWithCountsEP: %v", err)
	}

	// A0 avec stats explicites doit avoir μ > A1-3 (qui ont juste l'info win).
	for i := 1; i < 4; i++ {
		if postA[0].Mu <= postA[i].Mu {
			t.Errorf("A0 (counts complets) μ=%v devrait > A%d (counts nil) μ=%v",
				postA[0].Mu, i, postA[i].Mu)
		}
	}
	// A1-3 doivent être égaux entre eux (même priors, même outcome win,
	// pas d'obs individuelle).
	for i := 2; i < 4; i++ {
		if math.Abs(postA[1].Mu-postA[i].Mu) > 1e-6 {
			t.Errorf("A1 μ=%v, A%d μ=%v devraient être identiques (pas d'obs)",
				postA[1].Mu, i, postA[i].Mu)
		}
	}
}
