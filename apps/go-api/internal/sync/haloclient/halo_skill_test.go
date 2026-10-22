package haloclient

import (
	"encoding/json"
	"testing"
)

func TestComputeEnemyMMR(t *testing.T) {
	cases := []struct {
		name       string
		selfTeam   int
		teamMMRs   map[string]float64
		wantNil    bool
		wantApprox float64
	}{
		{
			name:       "2 teams: returns other team",
			selfTeam:   0,
			teamMMRs:   map[string]float64{"0": 1500, "1": 1450},
			wantApprox: 1450,
		},
		{
			name:       "FFA 4 teams: returns avg of 3 others",
			selfTeam:   0,
			teamMMRs:   map[string]float64{"0": 1500, "1": 1400, "2": 1500, "3": 1600},
			wantApprox: 1500, // (1400 + 1500 + 1600) / 3
		},
		{
			name:     "single team: nil",
			selfTeam: 0,
			teamMMRs: map[string]float64{"0": 1500},
			wantNil:  true,
		},
		{
			name:     "empty: nil",
			selfTeam: 0,
			teamMMRs: nil,
			wantNil:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeEnemyMMR(tc.selfTeam, tc.teamMMRs)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil enemy_mmr")
			}
			if d := *got - tc.wantApprox; d > 0.5 || d < -0.5 {
				t.Errorf("expected ~%.1f, got %.1f", tc.wantApprox, *got)
			}
		})
	}
}

func TestFilterHumanXUIDs(t *testing.T) {
	in := []string{"2533274858283686", "bid(3.0)", "", "2533274823110022", "bid(0.0)"}
	got := filterHumanXUIDs(in)
	want := []string{"2533274858283686", "2533274823110022"}
	if len(got) != len(want) {
		t.Fatalf("want %d, got %d: %v", len(want), len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestUnwrapXUID(t *testing.T) {
	cases := map[string]string{
		"xuid(2533274858283686)": "2533274858283686",
		"2533274858283686":       "2533274858283686",
		"":                       "",
	}
	for in, want := range cases {
		if got := unwrapXUID(in); got != want {
			t.Errorf("unwrapXUID(%q) = %q, want %q", in, got, want)
		}
	}
}

//
// L'endpoint /hi/matches/{id}/skill expose pour chaque joueur classé un objet
// RankRecap qui contient PreMatchCsr et PostMatchCsr (Value/Tier/SubTier +
// MeasurementMatchesRemaining pour les matchs de placement). Ces 3 tests
// vérifient que le parser remplit correctement MatchSkillData.PreMatchCSR /
// PostMatchCSR pour les 3 cas réels : ranked stable, ranked en placement, et
// non-ranked (RankRecap absent → pointeurs nil).

func TestTransformMatchSkillResponse_RankedMatchWithRankRecap(t *testing.T) {
	raw := []byte(`{
		"Value": [
			{
				"Id": "xuid(2533274858283686)",
				"ResultCode": 0,
				"Result": {
					"TeamMmr": 1500.0,
					"TeamId": 0,
					"TeamMmrs": {"0": 1500.0, "1": 1480.0},
					"RankRecap": {
						"PreMatchCsr":  {"Value": 1247, "Tier": "Gold",     "SubTier": 4, "MeasurementMatchesRemaining": 0},
						"PostMatchCsr": {"Value": 1259, "Tier": "Gold",     "SubTier": 5, "MeasurementMatchesRemaining": 0}
					}
				}
			}
		]
	}`)
	var resp matchSkillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := transformMatchSkillResponse(resp)
	data := out["2533274858283686"]
	if data == nil {
		t.Fatal("expected data for xuid")
	}
	if data.PreMatchCSR == nil {
		t.Fatal("expected PreMatchCSR populated")
	}
	if data.PreMatchCSR.Value != 1247 || data.PreMatchCSR.Tier != "Gold" || data.PreMatchCSR.SubTier != 4 {
		t.Errorf("PreMatchCSR: got value=%v tier=%q sub=%d, want 1247/Gold/4",
			data.PreMatchCSR.Value, data.PreMatchCSR.Tier, data.PreMatchCSR.SubTier)
	}
	if data.PostMatchCSR == nil {
		t.Fatal("expected PostMatchCSR populated")
	}
	if data.PostMatchCSR.Value != 1259 || data.PostMatchCSR.Tier != "Gold" || data.PostMatchCSR.SubTier != 5 {
		t.Errorf("PostMatchCSR: got value=%v tier=%q sub=%d, want 1259/Gold/5",
			data.PostMatchCSR.Value, data.PostMatchCSR.Tier, data.PostMatchCSR.SubTier)
	}
	if data.PostMatchCSR.MeasurementMatchesRemaining != 0 {
		t.Errorf("PostMatchCSR.MeasurementMatchesRemaining: want 0 for stable rank, got %d",
			data.PostMatchCSR.MeasurementMatchesRemaining)
	}
	// Sanity : on n'a pas cassé le parsing MMR existant.
	if data.TeamMMR == nil || *data.TeamMMR != 1500.0 {
		t.Errorf("TeamMMR should remain populated, got %v", data.TeamMMR)
	}
}

func TestTransformMatchSkillResponse_PlacementMatch(t *testing.T) {
	// Match de placement : pas de Tier final, Value=0, MeasurementMatchesRemaining
	// décrémenté à chaque match joué (10 → 0).
	raw := []byte(`{
		"Value": [
			{
				"Id": "xuid(2533274858283686)",
				"ResultCode": 0,
				"Result": {
					"TeamMmr": 1450.0,
					"TeamId": 0,
					"TeamMmrs": {"0": 1450.0, "1": 1500.0},
					"RankRecap": {
						"PreMatchCsr":  {"Value": 0, "Tier": "", "SubTier": 0, "MeasurementMatchesRemaining": 5},
						"PostMatchCsr": {"Value": 0, "Tier": "", "SubTier": 0, "MeasurementMatchesRemaining": 4}
					}
				}
			}
		]
	}`)
	var resp matchSkillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := transformMatchSkillResponse(resp)
	data := out["2533274858283686"]
	if data == nil {
		t.Fatal("expected data for xuid")
	}
	if data.PostMatchCSR == nil {
		t.Fatal("expected PostMatchCSR populated even in placement")
	}
	if data.PostMatchCSR.MeasurementMatchesRemaining != 4 {
		t.Errorf("placement: want MeasurementMatchesRemaining=4, got %d",
			data.PostMatchCSR.MeasurementMatchesRemaining)
	}
	if data.PostMatchCSR.Value != 0 {
		t.Errorf("placement: want Value=0, got %v", data.PostMatchCSR.Value)
	}
	if data.PostMatchCSR.Tier != "" {
		t.Errorf("placement: want empty Tier, got %q", data.PostMatchCSR.Tier)
	}
}

func TestTransformMatchSkillResponse_NoRankRecap(t *testing.T) {
	// Match social/firefight/custom : pas de RankRecap dans le payload.
	// Les nouveaux champs PreMatchCSR/PostMatchCSR doivent rester nil, sans
	// affecter le parsing MMR/StatPerformances.
	raw := []byte(`{
		"Value": [
			{
				"Id": "xuid(2533274858283686)",
				"ResultCode": 0,
				"Result": {
					"TeamMmr": 1500.0,
					"TeamId": 0,
					"TeamMmrs": {"0": 1500.0, "1": 1450.0},
					"StatPerformances": {
						"Kills":  {"Count": 12, "Expected": 10.5, "StdDev": 2.3},
						"Deaths": {"Count":  8, "Expected":  9.1, "StdDev": 1.5}
					}
				}
			}
		]
	}`)
	var resp matchSkillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := transformMatchSkillResponse(resp)
	data := out["2533274858283686"]
	if data == nil {
		t.Fatal("expected data for xuid")
	}
	if data.PreMatchCSR != nil {
		t.Errorf("PreMatchCSR should be nil when RankRecap absent, got %+v", data.PreMatchCSR)
	}
	if data.PostMatchCSR != nil {
		t.Errorf("PostMatchCSR should be nil when RankRecap absent, got %+v", data.PostMatchCSR)
	}
	if data.KillsExpected == nil || *data.KillsExpected != 10.5 {
		t.Errorf("KillsExpected should remain parsed, got %v", data.KillsExpected)
	}
}

// Un match classé abandonné (tout le monde parti) revient de l'API avec un MMR à 0 pour toutes
// les équipes : un 0 est une valeur ABSENTE, jamais un niveau. Ni le MMR d'équipe ni le MMR
// adverse ne doivent sortir du décodage (sinon un écart « 0 - 0 » ou « 1500 - 0 » entre dans les
// statistiques), alors que le CSR du match reste lu tel quel.
func TestTransformMatchSkillResponse_RankedMatchWithZeroMMR(t *testing.T) {
	raw := []byte(`{
		"Value": [
			{
				"Id": "xuid(2533274858283686)",
				"ResultCode": 0,
				"Result": {
					"TeamMmr": 0.0,
					"TeamId": 0,
					"TeamMmrs": {"0": 0.0, "1": 0.0},
					"RankRecap": {
						"PreMatchCsr":  {"Value": 1247, "Tier": "Gold", "SubTier": 4, "InitialMeasurementMatches": 10},
						"PostMatchCsr": {"Value": 1247, "Tier": "Gold", "SubTier": 4, "InitialMeasurementMatches": 10}
					}
				}
			}
		]
	}`)
	var resp matchSkillResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data := transformMatchSkillResponse(resp)["2533274858283686"]
	if data == nil {
		t.Fatal("expected data for xuid")
	}
	if data.TeamMMR != nil {
		t.Errorf("TeamMMR: un MMR à 0 est absent, got %v", *data.TeamMMR)
	}
	if data.EnemyMMR != nil {
		t.Errorf("EnemyMMR: un MMR à 0 est absent, got %v", *data.EnemyMMR)
	}
	if data.PostMatchCSR == nil {
		t.Error("PostMatchCSR : le CSR du match reste lu tel quel")
	}
}

func TestComputeEnemyMMR_IgnoresZeroTeams(t *testing.T) {
	cases := []struct {
		name     string
		teamMMRs map[string]float64
		want     *float64
	}{
		{"2 equipes, adversaire a 0 : absent", map[string]float64{"0": 1500, "1": 0}, nil},
		{"2 equipes a 0 : absent", map[string]float64{"0": 0, "1": 0}, nil},
		{"FFA : la moyenne ignore l'equipe a 0", map[string]float64{"0": 1500, "1": 1400, "2": 0, "3": 1600}, f64(1500)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeEnemyMMR(0, tc.teamMMRs)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("attendu absent, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("attendu %v, got absent", *tc.want)
			}
			if *got != *tc.want {
				t.Fatalf("attendu %v, got %v", *tc.want, *got)
			}
		})
	}
}

func f64(v float64) *float64 { return &v }
