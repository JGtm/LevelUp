package sync

import (
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

func TestMergeSkillIntoParticipants(t *testing.T) {
	tm, em := 1500.0, 1450.0
	ke, kd := 12.5, 2.1
	skill := map[string]*MatchSkillData{
		"player1": {
			XUID:           "player1",
			TeamMMR:        &tm,
			EnemyMMR:       &em,
			KillsExpected:  &ke,
			KillsStdDev:    &kd,
			DeathsExpected: nil,
		},
	}
	parts := []ParticipantRow{
		{XUID: "player1", MatchID: "m1"},
		{XUID: "player2", MatchID: "m1"}, // pas de skill data
	}
	out := MergeSkillIntoParticipants(parts, skill)
	if out[0].TeamMMR == nil || *out[0].TeamMMR != tm {
		t.Errorf("p1 team_mmr: want %.0f, got %v", tm, out[0].TeamMMR)
	}
	if out[0].EnemyMMR == nil || *out[0].EnemyMMR != em {
		t.Errorf("p1 enemy_mmr: want %.0f, got %v", em, out[0].EnemyMMR)
	}
	if out[0].KillsExpected == nil || *out[0].KillsExpected != ke {
		t.Errorf("p1 kills_expected: want %.1f, got %v", ke, out[0].KillsExpected)
	}
	if out[1].TeamMMR != nil {
		t.Errorf("p2 should have nil team_mmr (no skill data), got %v", *out[1].TeamMMR)
	}
}

func TestParticipantXUIDs_FiltersBots(t *testing.T) {
	rows := []ParticipantRow{
		{XUID: "2533274858283686"},
		{XUID: "bid(3.0)"},
		{XUID: "2533274823110022"},
		{XUID: ""},
	}
	got := ParticipantXUIDs(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 humans, got %d: %v", len(got), got)
	}
}
