package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// ptrInt (helper partagé, match_view_helpers_test.go) → pointeur int.

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
	if fd := buildViewerFragDistribution(nil, nil, nil, false); fd != nil {
		t.Errorf("viewer absent : want nil, got %+v", fd)
	}
	me := &domain.MatchScoreboardRow{XUID: "x1", IsMe: true, Kills: ptrInt(0)}
	if fd := buildViewerFragDistribution(me, nil, nil, false); fd != nil {
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
	fd := buildViewerFragDistribution(me, bulk, nil, false)
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
	fd := buildViewerFragDistribution(me, bulk, nil, true)
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

// TestBuildViewerFragDistribution_H5MechanicKillsNoDoubleCount (V72-15.3) : sur H5 les kills
// de mêlée/assassinat attribués à l'arme TENUE (MechanicKills sur la ligne bulk) sont retirés
// de la classe gun et servis par le compteur natif Mêlée → Σ classes == total (pas de
// double-comptage). Dataset choisi pour que le non-retrait DÉBORDERAIT (12 > 10).
func TestBuildViewerFragDistribution_H5MechanicKillsNoDoubleCount(t *testing.T) {
	me := &domain.MatchScoreboardRow{
		XUID: "me", IsMe: true, Kills: ptrInt(10),
		MeleeKills: ptrInt(3),
	}
	bulk := []domain.BulkWeaponKillRaw{
		// 9 kills BR dont 2 mêlées attribuées à l'arme tenue (MechanicKills=2).
		{XUID: "me", WeaponID: 1, Kills: 9, Class: "shoulder", Role: "automatic", MechanicKills: 2},
	}
	fd := buildViewerFragDistribution(me, bulk, nil, true)
	if fd == nil {
		t.Fatal("want distribution, got nil")
	}
	got := fragClassKills(fd)
	if got["shoulder"] != 7 {
		t.Errorf("shoulder = %d, want 7 (9 - 2 mécaniques natives)", got["shoulder"])
	}
	if got["melee"] != 3 {
		t.Errorf("melee = %d, want 3 (compteur natif)", got["melee"])
	}
	if _, ok := got["unattributed"]; ok {
		t.Errorf("Σ exacte après retrait : aucune classe unattributed attendue, got %+v", got)
	}
	sum := 0
	for _, k := range got {
		sum += k
	}
	if sum != fd.TotalKills {
		t.Errorf("Σ classes = %d != total %d (double-comptage non corrigé)", sum, fd.TotalKills)
	}
}

// TestBuildViewerFragDistribution_GrenadeSubLevel (V72-15.2) : les rows bulk class=grenade
// (avec Family) alimentent le niveau 2 de la classe Grenade par TYPE, le total restant le
// compteur natif ; elles ne fuient PAS dans les classes gun.
func TestBuildViewerFragDistribution_GrenadeSubLevel(t *testing.T) {
	me := &domain.MatchScoreboardRow{
		XUID: "me", IsMe: true, Kills: ptrInt(12),
		GrenadeKills: ptrInt(5),
	}
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "me", WeaponID: 1, Kills: 4, Class: "shoulder", Role: "automatic"},
		{XUID: "me", WeaponID: 20, Kills: 3, Class: "grenade", Role: "grenade", Family: "frag_grenade"},
		{XUID: "me", WeaponID: 21, Kills: 1, Class: "grenade", Role: "grenade", Family: "plasma_grenade"},
	}
	fd := buildViewerFragDistribution(me, bulk, nil, false)
	if fd == nil {
		t.Fatal("want distribution, got nil")
	}
	got := fragClassKills(fd)
	if got["grenade"] != 5 {
		t.Errorf("grenade = %d, want 5 (compteur natif, pas la somme des rows)", got["grenade"])
	}
	if got["shoulder"] != 4 {
		t.Errorf("shoulder = %d, want 4 (les rows grenade ne fuient pas dans les classes gun)", got["shoulder"])
	}
	var grenadeRoles []domain.FragRoleEntry
	for _, c := range fd.Classes {
		if c.Class == domain.FragClassGrenade {
			grenadeRoles = c.Roles
		}
	}
	// typé = 4 (frag 3 + plasma 1) ; total API = 5 → résidu « autre grenade » = 1.
	if len(grenadeRoles) != 3 {
		t.Fatalf("grenade.Roles = %+v, want 3 types (frag/plasma/autre)", grenadeRoles)
	}
	if grenadeRoles[0].Role != domain.FragRoleGrenadeFrag || grenadeRoles[0].Kills != 3 {
		t.Errorf("grenade.Roles[0] = %+v, want frag(3) en tête", grenadeRoles[0])
	}
}

// TestBuildCombatTabFull_ExcludesNonCombatFromWeaponBreakdown : le breakdown par-ARME
// (combat_tab.weapon_kills) ne contient que des OUTILS DE DESTRUCTION IDENTIFIABLES.
// Sont écartés les seuls buckets sans engin à nommer — non attribué, environnement,
// autre — dont les kills restent comptés dans les classes du sunburst.
//
// V73-3.2 : véhicule et tourelle ne sont PLUS écartés. Le registre les nomme par engin
// (« Warthog », « Tourelle Gauss »), ils ont donc leur place au breakdown comme au
// sunburst. Ce test figeait auparavant l'exclusion inverse (want 1).
func TestBuildCombatTabFull_ExcludesNonCombatFromWeaponBreakdown(t *testing.T) {
	const myXUID = "me"
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "me", WeaponID: 1, WeaponLabel: "BR75", Kills: 5, Class: "shoulder"},
		{XUID: "me", WeaponID: 2, WeaponLabel: "Spartan", Kills: 9, Class: "unattributed"},
		{XUID: "me", WeaponID: 3, WeaponLabel: "Warthog", Kills: 2, Class: "vehicle"},
		{XUID: "me", WeaponID: 4, WeaponLabel: "Turret", Kills: 1, Class: "turret"},
		{XUID: "me", WeaponID: 5, WeaponLabel: "Explosifs", Kills: 4, Class: "environmental"},
		{XUID: "me", WeaponID: 6, WeaponLabel: "Autre", Kills: 3, Class: "other"},
		{XUID: "other", WeaponID: 1, WeaponLabel: "BR75", Kills: 99, Class: "shoulder"}, // pas is_me
	}
	tab := buildCombatTabFull("m1", bulk, nil, nil, nil, nil, myXUID, 60000)

	got := make(map[string]string, len(tab.WeaponKills)) // label → classe
	for _, w := range tab.WeaponKills {
		got[w.WeaponLabel] = w.Class
	}
	want := map[string]string{"BR75": "shoulder", "Warthog": "vehicle", "Turret": "turret"}
	if len(got) != len(want) {
		t.Fatalf("WeaponKills = %d entrées, want %d (outils identifiables) : %+v",
			len(got), len(want), tab.WeaponKills)
	}
	for label, class := range want {
		if got[label] != class {
			t.Errorf("WeaponKills[%q] classe = %q, want %q", label, got[label], class)
		}
	}
	// Les buckets sans engin identifiable restent écartés (leur volume passe par le résidu).
	for _, label := range []string{"Spartan", "Explosifs", "Autre"} {
		if _, present := got[label]; present {
			t.Errorf("bucket non identifiable %q présent au breakdown par-arme", label)
		}
	}
}
