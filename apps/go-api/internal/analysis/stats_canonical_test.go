// Package analysis — stats_canonical_test.go : tests roundtrip pour le
// converter `StatsMatchRowFromCanonical` (P4.3c, ADR 0011).
package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// TestStatsMatchRowFromCanonical_RoundtripFields garantit que les champs clés
// (K/D/A, KDA, Accuracy, Outcome, IsRanked, Playlist) survivent à la conversion.
func TestStatsMatchRowFromCanonical_RoundtripFields(t *testing.T) {
	startTime := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)
	kills := 17
	deaths := 8
	assists := 4
	kda := 2.625
	accuracy := 0.45
	timePlayed := 720
	personalScore := 1500
	rank := 2
	teamID := 0
	dmgDealt := 3500
	dmgTaken := 2800
	teamMMR := 1250.0
	enemyMMR := 1180.0
	perfScore := 88.5
	killsExpected := 12.5
	deathsExpected := 9.0

	canonicalRow := canonical.PlayerMatchRow{
		Summary: canonical.MatchSummary{
			MatchID:      "m-stats-1",
			StartedAtUTC: startTime,
			IsRanked:     boolPtr(true),
			Outcome:      canonical.OutcomeWin,
			Playlist:     &canonical.AssetReference{ID: "ranked-arena", DefaultLabel: "Ranked Arena"},
		},
		Self: canonical.MatchParticipant{
			Kills:          &kills,
			Deaths:         &deaths,
			Assists:        &assists,
			KDA:            &kda,
			Accuracy:       &accuracy,
			TimePlayed:     &timePlayed,
			PersonalScore:  &personalScore,
			RankInMatch:    &rank,
			TeamID:         &teamID,
			DamageDealt:    &dmgDealt,
			DamageTaken:    &dmgTaken,
			KillsExpected:  &killsExpected,
			DeathsExpected: &deathsExpected,
			Outcome:        canonical.OutcomeWin,
		},
		Enrichment: canonical.PlayerMatchEnrichment{
			TeamMMR:          &teamMMR,
			EnemyMMR:         &enemyMMR,
			PerformanceScore: &perfScore,
			SkillSnapshot:    &canonical.SkillSnapshot{},
		},
	}

	out := StatsMatchRowFromCanonical(canonicalRow, 225)

	if out.MatchID != "m-stats-1" {
		t.Errorf("MatchID: got %q", out.MatchID)
	}
	if out.Kills != kills || out.Deaths != deaths || out.Assists != assists {
		t.Errorf("K/D/A: got %d/%d/%d, want %d/%d/%d", out.Kills, out.Deaths, out.Assists, kills, deaths, assists)
	}
	if out.Outcome == nil || *out.Outcome != domain.OutcomeWin {
		t.Errorf("Outcome: got %v, want %d", out.Outcome, domain.OutcomeWin)
	}
	if !out.IsRanked {
		t.Errorf("IsRanked: got false, want true")
	}
	if out.PlaylistName != "Ranked Arena" {
		t.Errorf("PlaylistName: got %q", out.PlaylistName)
	}
	if out.KDA == nil || *out.KDA != kda {
		t.Errorf("KDA: got %v, want %v", out.KDA, kda)
	}
	if out.KillsExpected == nil || *out.KillsExpected != killsExpected {
		t.Errorf("KillsExpected: got %v", out.KillsExpected)
	}
	if out.DamageDealt == nil || *out.DamageDealt != float64(dmgDealt) {
		t.Errorf("DamageDealt: got %v", out.DamageDealt)
	}
}

// TestStatsMatchRowFromCanonical_CombatYieldDerived vérifie que OC/DR (rendement
// offensif / résistance défensive) sont DÉRIVÉS depuis dégâts + K/D/A. Avant ce fix
// ils restaient nil → KPI Rendement/Résistance et nuage OC/DR de la page session vides.
func TestStatsMatchRowFromCanonical_CombatYieldDerived(t *testing.T) {
	kills, assists, deaths := 10, 6, 4
	dd, dt := 2000, 1800
	row := canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{
			Kills: &kills, Assists: &assists, Deaths: &deaths,
			DamageDealt: &dd, DamageTaken: &dt,
		},
	}
	out := StatsMatchRowFromCanonical(row, 225)
	// OC = 225*(10 + 6/3)/2000 = 225*12/2000 = 1.35
	if out.OffensiveConversion == nil || *out.OffensiveConversion < 1.349 || *out.OffensiveConversion > 1.351 {
		t.Fatalf("OffensiveConversion: want ~1.35, got %v", out.OffensiveConversion)
	}
	// DR = 1800/(225*4) = 2.0
	if out.DefensiveResistance == nil || *out.DefensiveResistance != 2.0 {
		t.Fatalf("DefensiveResistance: want 2.0, got %v", out.DefensiveResistance)
	}
}

// TestStatsMatchRowFromCanonical_NoDamageNoCombatYield : sans dégâts, OC/DR nil (pas de 0 trompeur).
func TestStatsMatchRowFromCanonical_NoDamageNoCombatYield(t *testing.T) {
	kills, deaths := 5, 3
	row := canonical.PlayerMatchRow{
		Self: canonical.MatchParticipant{Kills: &kills, Deaths: &deaths},
	}
	out := StatsMatchRowFromCanonical(row, 225)
	if out.OffensiveConversion != nil || out.DefensiveResistance != nil {
		t.Fatalf("expected nil OC/DR without damage, got oc=%v dr=%v", out.OffensiveConversion, out.DefensiveResistance)
	}
}

// TestStatsMatchRowFromCanonical_Outcomes vérifie le mapping de tous les outcomes.
func TestStatsMatchRowFromCanonical_Outcomes(t *testing.T) {
	cases := []struct {
		canonical canonical.Outcome
		domain    int
	}{
		{canonical.OutcomeWin, domain.OutcomeWin},
		{canonical.OutcomeLoss, domain.OutcomeLoss},
		{canonical.OutcomeTie, domain.OutcomeDraw},
		{canonical.OutcomeDNF, domain.OutcomeDNF},
	}
	for _, c := range cases {
		row := canonical.PlayerMatchRow{
			Self: canonical.MatchParticipant{Outcome: c.canonical},
		}
		out := StatsMatchRowFromCanonical(row, 225)
		if out.Outcome == nil {
			t.Errorf("Outcome %v: got nil, want %d", c.canonical, c.domain)
			continue
		}
		if *out.Outcome != c.domain {
			t.Errorf("Outcome %v: got %d, want %d", c.canonical, *out.Outcome, c.domain)
		}
	}
}
