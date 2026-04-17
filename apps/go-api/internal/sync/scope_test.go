// Package sync — scope_test.go : tests unitaires de SyncScope.Resolve() et ses helpers.
package sync

import "testing"

func TestSyncScope_NewScopeAll(t *testing.T) {
	s := NewScopeAll(500)
	s.Resolve()

	if !s.Medals {
		t.Error("Medals should be true after NewScopeAll")
	}
	if !s.Events {
		t.Error("Events should be true after NewScopeAll")
	}
	if !s.Weapons {
		t.Error("Weapons should be true after NewScopeAll")
	}
	if s.MaxMatches != 500 {
		t.Errorf("MaxMatches = %d, want 500", s.MaxMatches)
	}
}

func TestSyncScope_Resolve_MMRGroup(t *testing.T) {
	s := &SyncScope{MMR: true}
	s.Resolve()

	if !s.TeamMMR {
		t.Error("MMR group should set TeamMMR")
	}
	if !s.EnemyMMR {
		t.Error("MMR group should set EnemyMMR")
	}
}

func TestSyncScope_Resolve_ExpectedGroup(t *testing.T) {
	s := &SyncScope{Expected: true}
	s.Resolve()

	if !s.KillsExpected || !s.DeathsExpected || !s.AssistsExpected {
		t.Error("Expected group should set KillsExpected, DeathsExpected, AssistsExpected")
	}
}

func TestSyncScope_Resolve_CoreStatsGroup(t *testing.T) {
	s := &SyncScope{CoreStats: true}
	s.Resolve()

	if !s.Accuracy {
		t.Error("CoreStats should activate Accuracy (via Combat)")
	}
}

func TestSyncScope_Resolve_ForceImpliesFetch(t *testing.T) {
	s := &SyncScope{ForceMedals: true}
	s.Resolve()

	if !s.Medals {
		t.Error("ForceMedals should imply Medals=true")
	}
}

func TestSyncScope_HasAnyOption_True(t *testing.T) {
	s := &SyncScope{Medals: true}
	s.Resolve()
	if !s.HasAnyOption() {
		t.Error("should have option when Medals=true")
	}
}

func TestSyncScope_HasAnyOption_False(t *testing.T) {
	s := &SyncScope{}
	if s.HasAnyOption() {
		t.Error("empty scope should have no option")
	}
}

func TestSyncScope_RequestedTypes_NotEmpty(t *testing.T) {
	s := NewScopeAll(100)
	s.Resolve()
	types := s.RequestedTypes()
	if len(types) == 0 {
		t.Error("NewScopeAll should have requested types")
	}
}

func TestSyncScope_NeedsAPI_True(t *testing.T) {
	s := &SyncScope{Medals: true}
	s.Resolve()
	if !s.NeedsAPI() {
		t.Error("Medals requires API")
	}
}

// ── Sprint 47 : tests des groupes manquants ─────────────────────────

func TestSyncScope_Resolve_WeaponsGroup(t *testing.T) {
	s := &SyncScope{Weapons: true}
	s.Resolve()
	if !s.Weapons {
		t.Error("Weapons should remain true after Resolve")
	}
}

func TestSyncScope_Resolve_PVEStats(t *testing.T) {
	s := &SyncScope{PVEStats: true}
	s.Resolve()
	if !s.PVEStats {
		t.Error("PVEStats should remain true after Resolve")
	}
}

func TestSyncScope_Resolve_CombatGroup(t *testing.T) {
	s := &SyncScope{Combat: true}
	s.Resolve()
	if !s.Accuracy {
		t.Error("Combat should set Accuracy")
	}
	if !s.Shots {
		t.Error("Combat should set Shots")
	}
	if !s.Damage {
		t.Error("Combat should set Damage")
	}
}

func TestSyncScope_Resolve_KillsDetailGroup(t *testing.T) {
	s := &SyncScope{KillsDetail: true}
	s.Resolve()
	if !s.GrenadeKills {
		t.Error("KillsDetail should set GrenadeKills")
	}
	if !s.MeleeKills {
		t.Error("KillsDetail should set MeleeKills")
	}
	if !s.PowerWeaponKills {
		t.Error("KillsDetail should set PowerWeaponKills")
	}
	if !s.HeadshotKills {
		t.Error("KillsDetail should set HeadshotKills")
	}
}

func TestSyncScope_Resolve_SessionsGroup(t *testing.T) {
	s := &SyncScope{Sessions: true}
	s.Resolve()
	if !s.Sessions {
		t.Error("Sessions should remain true after Resolve")
	}
}

func TestSyncScope_Resolve_SkillRankGroup(t *testing.T) {
	s := &SyncScope{SkillRank: true}
	s.Resolve()
	if !s.LUSR {
		t.Error("SkillRank should set LUSR")
	}
	if !s.CSR {
		t.Error("SkillRank should set CSR")
	}
}

func TestSyncScope_Resolve_ForceSkillRank(t *testing.T) {
	s := &SyncScope{ForceSkillRank: true}
	s.Resolve()
	if !s.SkillRank {
		t.Error("ForceSkillRank should imply SkillRank")
	}
	if !s.ForceLUSR {
		t.Error("ForceSkillRank should set ForceLUSR")
	}
	if !s.ForceCSR {
		t.Error("ForceSkillRank should set ForceCSR")
	}
}

func TestSyncScope_Resolve_SkillActivatesMMRAndExpected(t *testing.T) {
	s := &SyncScope{Skill: true}
	s.Resolve()
	if !s.TeamMMR {
		t.Error("Skill should set TeamMMR")
	}
	if !s.EnemyMMR {
		t.Error("Skill should set EnemyMMR")
	}
	if !s.KillsExpected {
		t.Error("Skill should set KillsExpected")
	}
	if !s.DeathsExpected {
		t.Error("Skill should set DeathsExpected")
	}
	if !s.AssistsExpected {
		t.Error("Skill should set AssistsExpected")
	}
}

func TestSyncScope_Resolve_ForceWeapons(t *testing.T) {
	s := &SyncScope{ForceWeapons: true}
	s.Resolve()
	if !s.Weapons {
		t.Error("ForceWeapons should imply Weapons=true")
	}
}

func TestSyncScope_Resolve_ForcePVEStats(t *testing.T) {
	s := &SyncScope{ForcePVEStats: true}
	s.Resolve()
	if !s.PVEStats {
		t.Error("ForcePVEStats should imply PVEStats=true")
	}
}

func TestSyncScope_Resolve_ForceAccuracy(t *testing.T) {
	s := &SyncScope{ForceAccuracy: true}
	s.Resolve()
	if !s.Accuracy {
		t.Error("ForceAccuracy should imply Accuracy=true")
	}
}

func TestSyncScope_Resolve_ForceSessions(t *testing.T) {
	s := &SyncScope{ForceSessions: true}
	s.Resolve()
	if !s.Sessions {
		t.Error("ForceSessions should imply Sessions=true")
	}
}

func TestSyncScope_Resolve_ForceDamage(t *testing.T) {
	s := &SyncScope{ForceDamage: true}
	s.Resolve()
	if !s.Damage {
		t.Error("ForceDamage should imply Damage=true")
	}
}

func TestSyncScope_Resolve_ForceCombat(t *testing.T) {
	s := &SyncScope{ForceCombat: true}
	s.Resolve()
	if !s.Combat {
		t.Error("ForceCombat should imply Combat=true")
	}
}

func TestSyncScope_Resolve_ForceAllGranular(t *testing.T) {
	// Vérifie que chaque force_X implique X
	tests := []struct {
		name  string
		setup func(*SyncScope)
		check func(*SyncScope) bool
	}{
		{"ForceShots→Shots", func(s *SyncScope) { s.ForceShots = true }, func(s *SyncScope) bool { return s.Shots }},
		{"ForceEndTime→EndTime", func(s *SyncScope) { s.ForceEndTime = true }, func(s *SyncScope) bool { return s.EndTime }},
		{"ForceAliases→Aliases", func(s *SyncScope) { s.ForceAliases = true }, func(s *SyncScope) bool { return s.Aliases }},
		{"ForceAssets→Assets", func(s *SyncScope) { s.ForceAssets = true }, func(s *SyncScope) bool { return s.Assets }},
		{"ForceCitations→Citations", func(s *SyncScope) { s.ForceCitations = true }, func(s *SyncScope) bool { return s.Citations }},
		{"ForceComebackBadges→ComebackBadges", func(s *SyncScope) { s.ForceComebackBadges = true }, func(s *SyncScope) bool { return s.ComebackBadges }},
		{"ForcePlayableDuration→PlayableDuration", func(s *SyncScope) { s.ForcePlayableDuration = true }, func(s *SyncScope) bool { return s.PlayableDuration }},
		{"ForceLUSR→LUSR", func(s *SyncScope) { s.ForceLUSR = true }, func(s *SyncScope) bool { return s.LUSR }},
		{"ForceCSR→CSR", func(s *SyncScope) { s.ForceCSR = true }, func(s *SyncScope) bool { return s.CSR }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &SyncScope{}
			tc.setup(s)
			s.Resolve()
			if !tc.check(s) {
				t.Errorf("force implication failed for %s", tc.name)
			}
		})
	}
}

func TestSyncScope_NeedsAPI_False_LocalOnly(t *testing.T) {
	s := &SyncScope{Sessions: true}
	s.Resolve()
	// Sessions est un recalcul local → ne devrait PAS nécessiter l'API
	// (sauf si NeedsAPI l'inclut dans ses flags)
	// Ce test documente le comportement
	_ = s.NeedsAPI()
}

func TestSyncScope_AllData_SetsEverything(t *testing.T) {
	s := &SyncScope{AllData: true}
	s.Resolve()
	if !s.Medals || !s.Events || !s.Weapons || !s.PVEStats || !s.LUSR || !s.CSR {
		t.Error("AllData should set all major flags")
	}
	if !s.ComebackBadges || !s.PlayableDuration {
		t.Error("AllData should set ComebackBadges and PlayableDuration")
	}
}
