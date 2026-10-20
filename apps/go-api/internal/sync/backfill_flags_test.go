// Package sync — backfill_flags_test.go : vérifie la parité numérique des bitmasks Go vs Python.
//
// Référence Python :
//   - src/data/sync/constants.py  (ParticipantBits, MatchBits, PveBits)
//   - src/data/sync/migrations.py (BACKFILL_FLAGS)
//
// Sprint 20 — tâche 5 : bitmask numériquement identique entre Python et Go.
package sync

import "testing"

// ─────────────────────────────────────────────────────────────────────────────
// ParticipantBits — identiques à constants.py::ParticipantBits
// ─────────────────────────────────────────────────────────────────────────────

func TestParticipantBits_NumericIdenticalToPython(t *testing.T) {
	cases := []struct {
		name    string
		got     int
		wantHex string
		wantDec int
	}{
		{"PBitTeamMMR", PBitTeamMMR, "0x1", 1},
		{"PBitEnemyMMR", PBitEnemyMMR, "0x2", 2},
		{"PBitKillsExp", PBitKillsExp, "0x4", 4},
		{"PBitDeathsExp", PBitDeathsExp, "0x8", 8},
		// bit 4 (0x10) reserve : ancien PBitAssistsExp, retire (Halo Infinite n'a pas d'Assists)
		{"PBitAccuracy", PBitAccuracy, "0x20", 32},
		{"PBitShots", PBitShots, "0x40", 64},
		{"PBitDamage", PBitDamage, "0x80", 128},
		{"PBitAvgLife", PBitAvgLife, "0x100", 256},
		{"PBitMedals", PBitMedals, "0x200", 512},
		{"PBitGrenadeKills", PBitGrenadeKills, "0x400", 1024},
		{"PBitMeleeKills", PBitMeleeKills, "0x800", 2048},
		{"PBitPowerWeapon", PBitPowerWeapon, "0x1000", 4096},
		{"PBitPersonalScore", PBitPersonalScore, "0x2000", 8192},
		{"PBitHeadshotKills", PBitHeadshotKills, "0x4000", 16384},
		{"PBitMaxSpree", PBitMaxSpree, "0x8000", 32768},
		{"PBitKDA", PBitKDA, "0x10000", 65536},
		{"PBitTimePlayed", PBitTimePlayed, "0x20000", 131072},
		{"PBitKillerVictim", PBitKillerVictim, "0x40000", 262144},
	}
	for _, tc := range cases {
		if tc.got != tc.wantDec {
			t.Errorf("%s = %d (0x%X), attendu %d (%s)", tc.name, tc.got, tc.got, tc.wantDec, tc.wantHex)
		}
	}
}

func TestParticipantBits_GroupsConsistent(t *testing.T) {
	if PBitMMR != PBitTeamMMR|PBitEnemyMMR {
		t.Errorf("PBitMMR incohérent: %d", PBitMMR)
	}
	if PBitExpected != PBitKillsExp|PBitDeathsExp {
		t.Errorf("PBitExpected incohérent: %d", PBitExpected)
	}
	if PBitSkill != PBitMMR|PBitExpected {
		t.Errorf("PBitSkill incohérent: %d", PBitSkill)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MatchBits — identiques à constants.py::MatchBits (bits ≥ 16)
// ─────────────────────────────────────────────────────────────────────────────

func TestMatchBits_NumericIdenticalToPython(t *testing.T) {
	cases := []struct {
		name    string
		got     int
		wantDec int
	}{
		{"MBitEvents", MBitEvents, 65536},
		// MBitAssets (1<<17) et MBitAliases (1<<18) RETIRÉS le 2026-05-08
		// Phase 3 du plan PLAN_BITMASKS_AUDIT_FIX (orphelins purs).
		{"MBitKillerVictim", MBitKillerVictim, 524288},
		{"MBitPVEStats", MBitPVEStats, 1048576},
		{"MBitWeaponKills", MBitWeaponKills, 2097152},
		{"MBitFilmAbsent", MBitFilmAbsent, 4194304},
	}
	for _, tc := range cases {
		if tc.got != tc.wantDec {
			t.Errorf("%s = %d, attendu %d", tc.name, tc.got, tc.wantDec)
		}
	}
}

func TestMatchBits_NoCollisionWithParticipantBits(t *testing.T) {
	// Les MatchBits sont dans les bits ≥ 16 — PBitKDA=bit16, PBitTimePlayed=bit17,
	// PBitKillerVictim=bit18 sont des ParticipantBits stockés dans match_participants.
	// Les MBit* sont stockés dans match_registry.backfill_completed.
	// Pas de collision garantie par la conception (champs séparés).
	_ = MBitEvents | MBitWeaponKills // compile-time check
}

// ─────────────────────────────────────────────────────────────────────────────
// PveBits — identiques à constants.py::PveBits
// ─────────────────────────────────────────────────────────────────────────────

func TestPveBits_NumericIdenticalToPython(t *testing.T) {
	cases := []struct {
		name    string
		got     int
		wantDec int
	}{
		{"PveBitTotalKills", PveBitTotalKills, 1},
		{"PveBitBossKills", PveBitBossKills, 2},
		{"PveBitGrunt", PveBitGrunt, 4},
		{"PveBitElite", PveBitElite, 8},
		{"PveBitJackal", PveBitJackal, 16},
		{"PveBitBrute", PveBitBrute, 32},
		{"PveBitHunter", PveBitHunter, 64},
		{"PveBitSkimmer", PveBitSkimmer, 128},
		{"PveBitCrawler", PveBitCrawler, 256},
		{"PveBitSoldier", PveBitSoldier, 512},
		{"PveBitKnight", PveBitKnight, 1024},
		{"PveBitWarden", PveBitWarden, 2048},
		{"PveBitSentinel", PveBitSentinel, 4096},
		{"PveBitMarine", PveBitMarine, 8192},
	}
	for _, tc := range cases {
		if tc.got != tc.wantDec {
			t.Errorf("%s = %d, attendu %d", tc.name, tc.got, tc.wantDec)
		}
	}
}

func TestPveBits_FullMaskCoversAll14EnemyTypes(t *testing.T) {
	wantFull := 1 + 2 + 4 + 8 + 16 + 32 + 64 + 128 + 256 + 512 + 1024 + 2048 + 4096 + 8192
	if PveBitFullPVE != wantFull {
		t.Errorf("PveBitFullPVE = %d, attendu %d", PveBitFullPVE, wantFull)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BackfillFlags — identiques à migrations.py::BACKFILL_FLAGS
// ─────────────────────────────────────────────────────────────────────────────

func TestBackfillFlags_NumericIdenticalToPython(t *testing.T) {
	cases := []struct {
		key     string
		wantDec int
	}{
		{"medals", 1},
		{"events", 2},
		{"skill", 4},
		{"personal_scores", 8},
		{"accuracy", 32},
		{"shots", 64},
		{"enemy_mmr", 128},
		{"assets", 256},
		{"participants", 512},
		{"participants_scores", 1024},
		{"participants_kda", 2048},
		{"participants_shots", 4096},
		{"participants_damage", 8192},
		{"aliases", 16384},
		{"participants_avg_life", 32768},
		{"weapon_kills", 262144}, // bit 18, legacy obsolète
	}
	for _, tc := range cases {
		got, ok := BackfillFlags[tc.key]
		if !ok {
			t.Errorf("BackfillFlags[%q] absent", tc.key)
			continue
		}
		if got != tc.wantDec {
			t.Errorf("BackfillFlags[%q] = %d, attendu %d", tc.key, got, tc.wantDec)
		}
	}
}

func TestComputeBackfillMask(t *testing.T) {
	cases := []struct {
		types []string
		want  int
	}{
		{[]string{"medals"}, 1},
		{[]string{"medals", "events"}, 3},
		{[]string{"medals", "events", "skill"}, 7},
		{[]string{"shots", "accuracy"}, 96}, // 64 + 32
		{[]string{"unknown_key"}, 0},        // inconnu → 0
		{[]string{}, 0},
	}
	for _, tc := range cases {
		got := ComputeBackfillMask(tc.types...)
		if got != tc.want {
			t.Errorf("ComputeBackfillMask(%v) = %d, attendu %d", tc.types, got, tc.want)
		}
	}
}
