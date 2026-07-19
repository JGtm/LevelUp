package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ptrInt (helper partagé, match_view_canonical_test.go) → pointeur int.

// fragClassKills projette (class -> kills) pour des assertions concises.
func fragClassKills(fd *domain.FragDistribution) map[string]int {
	out := map[string]int{}
	if fd == nil {
		return out
	}
	for _, c := range fd.Classes {
		out[c.Class] = c.Kills
	}
	return out
}

// TestBuildViewerFragDistribution_NilOrEmpty : l'adapter Match view rend nil quand le
// viewer est absent (pas de ligne is_me) ou n'a aucun frag (total <= 0) — le front rend
// alors null (pas de sunburst vide).
func TestBuildViewerFragDistribution_NilOrEmpty(t *testing.T) {
	if fd := buildViewerFragDistribution(nil, nil, false); fd != nil {
		t.Errorf("viewer absent : want nil, got %+v", fd)
	}
	me := &domain.MatchScoreboardRow{XUID: "x1", IsMe: true, Kills: ptrInt(0)}
	if fd := buildViewerFragDistribution(me, nil, false); fd != nil {
		t.Errorf("0 frag : want nil, got %+v", fd)
	}
}

// TestBuildViewerFragDistribution_InfiniteCapOff : sur Infinite (hasMechanics=false) les
// compteurs natifs de la ligne is_me (melee/grenade) alimentent les classes API, les bulk
// weapon kills du viewer (filtrés sur son xuid) alimentent les classes gun, le reste
// retombe dans « Non attribué ». Pas de classe spartan, Mêlée FEUILLE (invariant d).
func TestBuildViewerFragDistribution_InfiniteCapOff(t *testing.T) {
	me := &domain.MatchScoreboardRow{
		XUID: "me", IsMe: true, Kills: ptrInt(20),
		MeleeKills: ptrInt(3), GrenadeKills: ptrInt(2),
	}
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "me", WeaponID: 1, Kills: 8, Class: "shoulder", Role: "precision"},
		{XUID: "me", WeaponID: 2, Kills: 2, Class: "sidearm", Role: "sidearm"},
		{XUID: "other", WeaponID: 1, Kills: 99, Class: "shoulder", Role: "precision"}, // ignoré (pas is_me)
	}
	fd := buildViewerFragDistribution(me, bulk, false)
	if fd == nil {
		t.Fatal("want distribution, got nil")
	}
	if fd.TotalKills != 20 {
		t.Errorf("TotalKills = %d, want 20", fd.TotalKills)
	}
	got := fragClassKills(fd)
	// gun: shoulder=8, sidearm=2 ; API: melee=3, grenade=2 ; unattr = 20-15 = 5.
	if got["shoulder"] != 8 || got["sidearm"] != 2 || got["melee"] != 3 || got["grenade"] != 2 {
		t.Errorf("classes = %+v, want shoulder8/sidearm2/melee3/grenade2", got)
	}
	if got["unattributed"] != 5 {
		t.Errorf("unattributed = %d, want 5", got["unattributed"])
	}
	if _, ok := got["spartan_ability"]; ok {
		t.Error("cap off : aucune classe spartan_ability attendue (invariant d)")
	}
	// Σ classes == total (invariant a).
	sum := 0
	for _, k := range got {
		sum += k
	}
	if sum != fd.TotalKills {
		t.Errorf("Σ classes = %d != total %d (invariant a)", sum, fd.TotalKills)
	}
}

// TestBuildViewerFragDistribution_H5CapOn : sur H5 (hasMechanics=true) les mécaniques
// natives de la ligne is_me alimentent la classe Capacités spartanes et le niveau 2 de
// Mêlée (assassinat + corps-à-corps direct, disjoints).
func TestBuildViewerFragDistribution_H5CapOn(t *testing.T) {
	me := &domain.MatchScoreboardRow{
		XUID: "me", IsMe: true, Kills: ptrInt(30),
		MeleeKills: ptrInt(4), GrenadeKills: ptrInt(3),
		AssassinationKills: ptrInt(2), GroundPoundKills: ptrInt(1), ShoulderBashKills: ptrInt(1),
	}
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "me", WeaponID: 1, Kills: 10, Class: "shoulder", Role: "automatic"},
	}
	fd := buildViewerFragDistribution(me, bulk, true)
	if fd == nil {
		t.Fatal("want distribution, got nil")
	}
	got := fragClassKills(fd)
	// melee classe = melee(4)+assass(2)=6 ; spartan = ground(1)+shoulder(1)=2.
	if got["melee"] != 6 {
		t.Errorf("melee = %d, want 6 (4 mêlée + 2 assassinats disjoints)", got["melee"])
	}
	if got["spartan_ability"] != 2 {
		t.Errorf("spartan_ability = %d, want 2", got["spartan_ability"])
	}
	// Vérifie le niveau 2 de Mêlée (2 rôles dont l'Assassinat, gated par la mécanique
	// native H5) et la présence de la classe Capacités spartanes → dépend des compteurs
	// gp/sb/assass portés par la ligne scoreboard (FIX D : chargés depuis Q12).
	var meleeRoles []domain.FragRoleEntry
	hasSpartan := false
	for _, c := range fd.Classes {
		switch c.Class {
		case domain.FragClassMelee:
			meleeRoles = c.Roles
		case domain.FragClassSpartanAbility:
			hasSpartan = true
		}
	}
	if !hasSpartan {
		t.Error("classe Capacités spartanes absente (attendue quand gp/sb portés par le scoreboard)")
	}
	if len(meleeRoles) != 2 {
		t.Fatalf("melee.Roles = %+v, want 2 rôles (assassination+direct_melee)", meleeRoles)
	}
	foundAssassination := false
	for _, r := range meleeRoles {
		if r.Role == domain.FragRoleAssassination {
			foundAssassination = true
			if r.Kills != 2 {
				t.Errorf("rôle Assassinat kills = %d, want 2", r.Kills)
			}
		}
	}
	if !foundAssassination {
		t.Errorf("rôle Assassinat absent de melee.Roles = %+v", meleeRoles)
	}
}

// TestBuildCombatTabFull_ExcludesNonCombatFromWeaponBreakdown : le breakdown par-ARME
// (combat_tab.weapon_kills) ne contient QUE des armes réelles — les buckets non-combat
// H5 (véhicule/tourelle/environnement/non attribué) sont écartés (FIX 2). Ces kills
// restent comptés dans les classes du sunburst.
func TestBuildCombatTabFull_ExcludesNonCombatFromWeaponBreakdown(t *testing.T) {
	const myXUID = "me"
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "me", WeaponID: 1, WeaponLabel: "BR75", Kills: 5, Class: "shoulder"},
		{XUID: "me", WeaponID: 2, WeaponLabel: "Spartan", Kills: 9, Class: "unattributed"},
		{XUID: "me", WeaponID: 3, WeaponLabel: "Warthog", Kills: 2, Class: "vehicle"},
		{XUID: "me", WeaponID: 4, WeaponLabel: "Turret", Kills: 1, Class: "turret"},
		{XUID: "other", WeaponID: 1, WeaponLabel: "BR75", Kills: 99, Class: "shoulder"}, // pas is_me
	}
	tab := buildCombatTabFull("m1", bulk, nil, nil, nil, nil, myXUID, 60000)
	if len(tab.WeaponKills) != 1 {
		t.Fatalf("WeaponKills = %d, want 1 (armes réelles seulement) : %+v", len(tab.WeaponKills), tab.WeaponKills)
	}
	if tab.WeaponKills[0].WeaponLabel != "BR75" || tab.WeaponKills[0].Class != "shoulder" {
		t.Errorf("WeaponKills[0] = %+v, want BR75/shoulder", tab.WeaponKills[0])
	}
}
