// Package sync — backfill_flags_extra_test.go : tests ComputeParticipantBitsFromData.
//
// La fonction calcule le bitmask backfill à partir des colonnes présentes.
package sync

import (
	"testing"
)

func TestComputeParticipantBitsFromData_Empty(t *testing.T) {
	data := map[string]interface{}{}
	got := ComputeParticipantBitsFromData(data)
	if got != 0 {
		t.Errorf("empty data: got %d, want 0", got)
	}
}

func TestComputeParticipantBitsFromData_WithShotsFired(t *testing.T) {
	data := map[string]interface{}{
		"shots_fired": float64(100),
	}
	got := ComputeParticipantBitsFromData(data)
	if got&PBitShots == 0 {
		t.Errorf("shots_fired present: bit PBitShots should be set, got %b", got)
	}
}

func TestComputeParticipantBitsFromData_WithDamage(t *testing.T) {
	data := map[string]interface{}{
		"damage_dealt": float64(3000),
	}
	got := ComputeParticipantBitsFromData(data)
	if got&PBitDamage == 0 {
		t.Errorf("damage_dealt present: bit PBitDamage should be set, got %b", got)
	}
}

func TestComputeParticipantBitsFromData_WithMMR(t *testing.T) {
	data := map[string]interface{}{
		"team_mmr":  float64(1500),
		"enemy_mmr": float64(1400),
	}
	got := ComputeParticipantBitsFromData(data)
	if got&PBitTeamMMR == 0 {
		t.Errorf("team_mmr present: bit PBitTeamMMR should be set, got %b", got)
	}
	if got&PBitEnemyMMR == 0 {
		t.Errorf("enemy_mmr present: bit PBitEnemyMMR should be set, got %b", got)
	}
	if got&PBitMMR != PBitMMR {
		t.Errorf("both MMR present: PBitMMR group should be fully set, got %b", got)
	}
}

func TestComputeParticipantBitsFromData_AllBits(t *testing.T) {
	data := map[string]interface{}{
		"team_mmr":            float64(1500),
		"enemy_mmr":           float64(1400),
		"kills_expected":      float64(10),
		"deaths_expected":     float64(8),
		"accuracy":            float64(45),
		"shots_fired":         float64(200),
		"damage_dealt":        float64(3000),
		"avg_life_seconds":    float64(30),
		"grenade_kills":       float64(2),
		"melee_kills":         float64(1),
		"power_weapon_kills":  float64(3),
		"personal_score":      float64(2500),
		"headshot_kills":      float64(5),
		"max_killing_spree":   float64(7),
		"kda":                 float64(2.0),
		"time_played_seconds": float64(600),
	}
	got := ComputeParticipantBitsFromData(data)
	expected := PBitAllStats &^ (PBitMedals | PBitKillerVictim) // all stat bits minus medals and KV (not in participant data)
	if got != expected {
		t.Errorf("all bits: got %b, want %b", got, expected)
	}
}

func TestComputeParticipantBitsFromData_NilValue(t *testing.T) {
	data := map[string]interface{}{
		"shots_fired": nil, // nil should NOT set the bit
	}
	got := ComputeParticipantBitsFromData(data)
	if got&PBitShots != 0 {
		t.Errorf("nil value: bit should not be set, got %b", got)
	}
}

func TestComputeParticipantBitsFromData_PartialExpected(t *testing.T) {
	data := map[string]interface{}{
		"kills_expected": float64(10),
		// deaths_expected missing
	}
	got := ComputeParticipantBitsFromData(data)
	if got&PBitKillsExp == 0 {
		t.Errorf("kills_expected present: PBitKillsExp should be set")
	}
	if got&PBitDeathsExp != 0 {
		t.Errorf("deaths_expected missing: PBitDeathsExp should not be set")
	}
}
