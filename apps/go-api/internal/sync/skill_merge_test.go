package sync

import "testing"

// skill_merge_test.go — tests du pont skill->ParticipantRow (MergeSkillIntoParticipants /
// ParticipantXUIDs), extraits de halo_skill_test.go lors de l'extraction du client
// vers haloclient (K3e) : ces tests utilisent ParticipantRow (type sync), pas les
// internes de parsing (restés dans haloclient).

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
