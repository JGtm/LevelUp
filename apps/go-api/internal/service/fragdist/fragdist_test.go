package fragdist

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// findFragClass renvoie l'entrée de classe (et sa présence) dans une distribution.
func findFragClass(fd domain.FragDistribution, class string) (domain.FragClassEntry, bool) {
	for _, c := range fd.Classes {
		if c.Class == class {
			return c, true
		}
	}
	return domain.FragClassEntry{}, false
}

// assertFragInvariants vérifie les invariants §2 (a)(b)(c) sur une distribution.
func assertFragInvariants(t *testing.T, fd domain.FragDistribution) {
	t.Helper()
	sum := 0
	for _, c := range fd.Classes {
		sum += c.Kills
		if len(c.Roles) > 0 { // (b) classe non-feuille : Σ rôles == kills de la classe
			rs := 0
			for _, r := range c.Roles {
				rs += r.Kills
			}
			if rs != c.Kills {
				t.Errorf("invariant (b) violé pour %q : Σ rôles=%d != kills=%d", c.Class, rs, c.Kills)
			}
		}
		if c.Class == domain.FragClassUnattributed && c.Kills < 0 { // (c)
			t.Errorf("invariant (c) violé : unattributed=%d < 0", c.Kills)
		}
	}
	if sum != fd.TotalKills { // (a) Σ classes == total
		t.Errorf("invariant (a) violé : Σ classes=%d != total=%d", sum, fd.TotalKills)
	}
}

// heterogeneousInfiniteRows — dataset gun réaliste multi-classes/multi-rôles (Infinite).
func heterogeneousInfiniteRows() []port.WeaponKillRow {
	return []port.WeaponKillRow{
		{WeaponID: 10, Kills: 10, Class: "shoulder", Role: "precision"}, // BR75
		{WeaponID: 11, Kills: 8, Class: "shoulder", Role: "automatic"},  // AR
		{WeaponID: 12, Kills: 5, Class: "heavy", Role: "sniper"},        // S7
		{WeaponID: 13, Kills: 3, Class: "heavy", Role: "power"},         // SPNKr
		{WeaponID: 14, Kills: 4, Class: "sidearm", Role: "sidearm"},     // Sidekick
		// Arme mêlée registre (épée) : class=melee → EXCLUE des classes gun (servie
		// par le total API melee) ; ne double-compte pas.
		{WeaponID: 15, Kills: 2, Class: "melee", Role: "melee"},
		// Row sans class/role résolu → ignorée (retombe dans le résidu).
		{WeaponID: 16, Kills: 7, Class: "", Role: ""},
	}
}

// TestBuild_InfiniteCapOff verrouille le cas Halo Infinite (cap native_kill_mechanics
// OFF) : gun classes registre + melee/grenade API, PAS de spartan et Mêlée FEUILLE
// (invariant d), résidu Non attribué calculé.
func TestBuild_InfiniteCapOff(t *testing.T) {
	counts := domain.FragKillTypeCounts{
		Melee:   6,
		Grenade: 4,
		Total:   50,
		// mécaniques natives absentes (Infinite)
	}
	// gun=30, melee=6, grenade=4 → attribué=40 ; total=50 → unattributed=10.
	fd := Build(heterogeneousInfiniteRows(), counts, false)
	assertFragInvariants(t, fd)

	if _, ok := findFragClass(fd, domain.FragClassSpartanAbility); ok {
		t.Error("cap off : aucune classe spartan_ability attendue (invariant d)")
	}
	melee, ok := findFragClass(fd, domain.FragClassMelee)
	if !ok || melee.Roles != nil {
		t.Errorf("cap off : Mêlée doit être une feuille (Roles nil), got %+v", melee)
	}
	if !melee.Authoritative {
		t.Error("Mêlée doit être Authoritative (total API)")
	}
	sidearm, ok := findFragClass(fd, domain.FragClassSidearm)
	if !ok || sidearm.Roles != nil {
		t.Errorf("Poing/sidearm doit être une feuille (rôle unique == classe), got %+v", sidearm)
	}
	shoulder, ok := findFragClass(fd, domain.FragClassShoulder)
	if !ok || shoulder.Kills != 18 || shoulder.Authoritative {
		t.Errorf("shoulder = %+v, want kills=18 estimé (Authoritative=false)", shoulder)
	}
	if len(shoulder.Roles) != 2 || shoulder.Roles[0].Role != "precision" || shoulder.Roles[0].Kills != 10 {
		t.Errorf("shoulder.Roles = %+v, want precision(10) en tête", shoulder.Roles)
	}
	unattr, ok := findFragClass(fd, domain.FragClassUnattributed)
	if !ok || unattr.Kills != 10 || unattr.Authoritative {
		t.Errorf("unattributed = %+v, want kills=10 (Authoritative=false)", unattr)
	}
}

// TestBuild_H5CapOn verrouille le cas Halo 5 (cap ON) : classe Capacités spartanes
// présente + Mêlée niveau 2 (assassination + direct_melee).
func TestBuild_H5CapOn(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 20, Kills: 12, Class: "shoulder", Role: "automatic"},
		{WeaponID: 21, Kills: 6, Class: "heavy", Role: "power"},
		{WeaponID: 22, Kills: 3, Class: "sidearm", Role: "sidearm"},
	}
	counts := domain.FragKillTypeCounts{
		Melee:         10,
		Assassination: 4,
		Grenade:       5,
		GroundPound:   3,
		ShoulderBash:  2,
		Total:         50,
	}
	// gun=21, melee=melee+assass=14, grenade=5, spartan=5 → attribué=45 ; total=50 →
	// unattributed=5. (assassinat ⊄ mêlée : compteurs API disjoints, cf. meleeRoles.)
	fd := Build(rows, counts, true)
	assertFragInvariants(t, fd)

	spartan, ok := findFragClass(fd, domain.FragClassSpartanAbility)
	if !ok || spartan.Kills != 5 || !spartan.Authoritative {
		t.Errorf("spartan_ability = %+v, want kills=5 Authoritative", spartan)
	}
	if len(spartan.Roles) != 2 ||
		spartan.Roles[0].Role != domain.FragRoleGroundPound || spartan.Roles[0].Kills != 3 ||
		spartan.Roles[1].Role != domain.FragRoleShoulderBash || spartan.Roles[1].Kills != 2 {
		t.Errorf("spartan.Roles = %+v, want ground_pound(3)+shoulder_bash(2)", spartan.Roles)
	}
	melee, ok := findFragClass(fd, domain.FragClassMelee)
	if !ok || len(melee.Roles) != 2 {
		t.Fatalf("melee = %+v, want 2 rôles (assassination+direct_melee)", melee)
	}
	// Ordre sémantique fixe (déterministe) : Assassinat puis Corps-à-corps direct.
	// direct_melee = melee (10) SANS soustraction ; assassination = assass (4) ;
	// total classe = 14 (disjoints).
	if melee.Roles[0].Role != domain.FragRoleAssassination || melee.Roles[0].Kills != 4 {
		t.Errorf("melee.Roles[0] = %+v, want assassination(4) en tête", melee.Roles[0])
	}
	if melee.Roles[1].Role != domain.FragRoleDirectMelee || melee.Roles[1].Kills != 10 {
		t.Errorf("melee.Roles[1] = %+v, want direct_melee(10)", melee.Roles[1])
	}
	if melee.Kills != 14 {
		t.Errorf("melee.Kills = %d, want 14 (melee 10 + assass 4, disjoints)", melee.Kills)
	}
}

// TestBuild_CanonicalOrder verrouille l'ordre déterministe des classes.
func TestBuild_CanonicalOrder(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 30, Kills: 5, Class: "heavy", Role: "power"},
		{WeaponID: 31, Kills: 5, Class: "shoulder", Role: "automatic"},
		{WeaponID: 32, Kills: 5, Class: "sidearm", Role: "sidearm"},
	}
	counts := domain.FragKillTypeCounts{
		Melee: 5, Grenade: 5,
		GroundPound: 5,
		Total:       40,
	}
	fd := Build(rows, counts, true) // 15 gun +5+5+5 spartan =30 ; total 40 → unattr 10
	assertFragInvariants(t, fd)
	want := []string{
		domain.FragClassShoulder, domain.FragClassSidearm, domain.FragClassHeavy,
		domain.FragClassMelee, domain.FragClassGrenade, domain.FragClassSpartanAbility,
		domain.FragClassUnattributed,
	}
	if len(fd.Classes) != len(want) {
		t.Fatalf("len classes = %d, want %d (%+v)", len(fd.Classes), len(want), fd.Classes)
	}
	for i, w := range want {
		if fd.Classes[i].Class != w {
			t.Errorf("classe[%d] = %q, want %q", i, fd.Classes[i].Class, w)
		}
	}
}

// TestBuild_AssassinationGreaterThanMelee : assassinat > mêlée est un cas VALIDE (les
// deux compteurs API sont DISJOINTS — cf. match 9bb09267… melee=2 < assass=4). Plus de
// clamp : la classe Mêlée = melee + assass, les deux rôles coexistent sans soustraction.
// C'est le cas qui « cassait » sous l'ancienne hypothèse d'inclusion.
func TestBuild_AssassinationGreaterThanMelee(t *testing.T) {
	counts := domain.FragKillTypeCounts{
		Melee:         2,
		Assassination: 4, // > melee : disjoint, aucun clamp
		Total:         6, // melee 2 + assass 4 (les assassinats SONT des kills du total)
	}
	fd := Build(nil, counts, true)
	assertFragInvariants(t, fd)
	melee, ok := findFragClass(fd, domain.FragClassMelee)
	if !ok || melee.Kills != 6 || len(melee.Roles) != 2 {
		t.Fatalf("melee = %+v, want kills=6 avec 2 rôles (assassination+direct_melee)", melee)
	}
	if melee.Roles[0].Role != domain.FragRoleAssassination || melee.Roles[0].Kills != 4 {
		t.Errorf("melee.Roles[0] = %+v, want assassination(4)", melee.Roles[0])
	}
	if melee.Roles[1].Role != domain.FragRoleDirectMelee || melee.Roles[1].Kills != 2 {
		t.Errorf("melee.Roles[1] = %+v, want direct_melee(2)", melee.Roles[1])
	}
}

// TestBuild_NoResidualWhenExact : aucune classe Non attribué quand l'attribution égale
// le total.
func TestBuild_NoResidualWhenExact(t *testing.T) {
	rows := []port.WeaponKillRow{{WeaponID: 40, Kills: 10, Class: "shoulder", Role: "automatic"}}
	fd := Build(rows, domain.FragKillTypeCounts{Total: 10}, false)
	assertFragInvariants(t, fd)
	if _, ok := findFragClass(fd, domain.FragClassUnattributed); ok {
		t.Error("attribution exacte : aucune classe unattributed attendue")
	}
}
