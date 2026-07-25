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

// findFragRole projette (role -> kills) des rôles d'une classe.
func findFragRole(c domain.FragClassEntry, role string) (int, bool) {
	for _, r := range c.Roles {
		if r.Role == role {
			return r.Kills, true
		}
	}
	return 0, false
}

// TestBuild_GrenadeSubLevelByType (V72-15.2) : le niveau 2 de la classe Grenade ventile par
// TYPE (famille registre) ; le total de la classe reste le compteur API autoritatif et le
// résidu « autre grenade » absorbe l'écart (Σ types == kills de la classe, invariant b).
func TestBuild_GrenadeSubLevelByType(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 50, Kills: 3, Class: "grenade", Role: "grenade", Family: "frag_grenade"},
		{WeaponID: 51, Kills: 2, Class: "grenade", Role: "grenade", Family: "plasma_grenade"},
	}
	// total API grenade = 6 ; typé = 5 (frag 3 + plasma 2) → résidu « autre grenade » = 1.
	counts := domain.FragKillTypeCounts{Grenade: 6, Total: 20}
	fd := Build(rows, counts, false)
	assertFragInvariants(t, fd)
	g, ok := findFragClass(fd, domain.FragClassGrenade)
	if !ok || g.Kills != 6 || !g.Authoritative {
		t.Fatalf("grenade = %+v, want kills=6 Authoritative (total API)", g)
	}
	if len(g.Roles) != 3 {
		t.Fatalf("grenade.Roles = %+v, want 3 types (frag/plasma/autre)", g.Roles)
	}
	if g.Roles[0].Role != domain.FragRoleGrenadeFrag || g.Roles[0].Kills != 3 {
		t.Errorf("grenade.Roles[0] = %+v, want frag(3) en tête (kills desc)", g.Roles[0])
	}
	if k, ok := findFragRole(g, domain.FragRoleGrenadePlasma); !ok || k != 2 {
		t.Errorf("type plasma = %d (present=%v), want 2", k, ok)
	}
	if k, ok := findFragRole(g, domain.FragRoleGrenadeOther); !ok || k != 1 {
		t.Errorf("résidu autre grenade = %d (present=%v), want 1", k, ok)
	}
}

// TestBuild_GrenadeLeafWhenNoTypedRows : sans row grenade typée (donnée absente), la classe
// Grenade reste une FEUILLE (dégradation data-driven, ni Infinite ni H5 en dur).
func TestBuild_GrenadeLeafWhenNoTypedRows(t *testing.T) {
	fd := Build(nil, domain.FragKillTypeCounts{Grenade: 4, Total: 10}, false)
	assertFragInvariants(t, fd)
	g, ok := findFragClass(fd, domain.FragClassGrenade)
	if !ok || g.Kills != 4 || g.Roles != nil {
		t.Fatalf("grenade = %+v, want kills=4 FEUILLE (Roles nil) sans row typée", g)
	}
}

// TestBuild_GrenadeLeafWhenOverAttributed : si le registre sur-attribue (Σ types > total API),
// la classe Grenade reste feuille (pas de ventilation trompeuse) — le total garde le compteur
// API et l'invariant b tient trivialement.
func TestBuild_GrenadeLeafWhenOverAttributed(t *testing.T) {
	rows := []port.WeaponKillRow{
		{WeaponID: 50, Kills: 5, Class: "grenade", Role: "grenade", Family: "frag_grenade"},
	}
	fd := Build(rows, domain.FragKillTypeCounts{Grenade: 4, Total: 10}, false)
	assertFragInvariants(t, fd)
	g, ok := findFragClass(fd, domain.FragClassGrenade)
	if !ok || g.Kills != 4 || g.Roles != nil {
		t.Fatalf("grenade = %+v, want kills=4 FEUILLE (sur-attribution registre)", g)
	}
}

// TestBuild_MechanicKillsExcludedFromGunClasses (V72-15.3) : sur H5 une mêlée/un assassinat
// attribué à l'arme TENUE (MechanicKills) est RETIRÉ de la classe gun — sans ce retrait il
// double-compterait avec le compteur natif Mêlée et ferait Σ classes > total. Dataset choisi
// pour que le non-retrait DÉBORDE (Σ > total).
func TestBuild_MechanicKillsExcludedFromGunClasses(t *testing.T) {
	// 10 frags au total = 7 BR (arme) + 3 mêlées, dont 2 mêlées attribuées à l'arme tenue
	// (BR) dans weapon_kills (MechanicKills=2). Compteur natif Mêlée = 3.
	rows := []port.WeaponKillRow{
		{WeaponID: 60, Kills: 9, Class: "shoulder", Role: "automatic", MechanicKills: 2},
	}
	counts := domain.FragKillTypeCounts{Melee: 3, Total: 10}
	fd := Build(rows, counts, true)
	assertFragInvariants(t, fd) // Σ classes == 10 (sinon le non-retrait ferait 12 > 10)
	sh, ok := findFragClass(fd, domain.FragClassShoulder)
	if !ok || sh.Kills != 7 {
		t.Fatalf("shoulder = %+v, want kills=7 (9 - 2 mécaniques natives)", sh)
	}
	if len(sh.Roles) != 1 || sh.Roles[0].Kills != 7 {
		t.Errorf("shoulder.Roles = %+v, want [automatic(7)] (rôle porte les kills d'arme seuls)", sh.Roles)
	}
	melee, ok := findFragClass(fd, domain.FragClassMelee)
	if !ok || melee.Kills != 3 {
		t.Errorf("melee = %+v, want kills=3 (compteur natif, sert les 2 mêlées attribuées à l'arme)", melee)
	}
	if _, ok := findFragClass(fd, domain.FragClassUnattributed); ok {
		t.Error("attribution exacte après retrait : aucune classe unattributed attendue")
	}
}
