// Package fragdist héberge le builder PUR de la « Répartition des frags » v2
// (sunburst hiérarchique classe→rôle). Extrait de package service (P6) pour être
// réutilisable SANS duplication par toutes les couches qui en ont besoin —
// Synthesis / Match view / Timeseries / Sessions (package service) ET la page
// Escouade (package service/teammates, qui ne peut pas importer son parent
// service). Cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md §2 + §6 (D-P6-1).
//
// Le builder reste PUR (aucune IO, aucun log — le câblage service loggue les
// compteurs via logFragDistribution). Les types (FragDistribution/FragClassEntry/
// FragRoleEntry/FragKillTypeCounts) vivent dans internal/domain (leaf partagé).
package fragdist

import (
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// canonicalFragClassOrder fixe l'ordre déterministe des classes du sunburst v2
// (§4 du plan). Toute sortie de Build suit cet ordre.
var canonicalFragClassOrder = []string{
	domain.FragClassShoulder, domain.FragClassSidearm, domain.FragClassHeavy,
	domain.FragClassMelee, domain.FragClassGrenade, domain.FragClassSpartanAbility,
	domain.FragClassUnattributed,
}

// gunFragClasses = classes dont la ventilation par arme vient du REGISTRE (estimé).
// Les autres classes (melee/grenade/spartan) sont servies par les totaux API, et
// les buckets non-combat H5 (vehicle/turret/…) retombent dans « Non attribué ».
var gunFragClasses = map[string]bool{
	domain.FragClassShoulder: true,
	domain.FragClassSidearm:  true,
	domain.FragClassHeavy:    true,
}

// Build assemble la répartition hiérarchique des frags (sunburst v2).
// Builder PUR (aucune IO, aucun log — le câblage service loggue les compteurs).
//
// Provenance (anti-double-source, §2) : classes gun shoulder/sidearm/heavy + rôles
// d'arme = registre (rows, Authoritative=false) ; classes melee/grenade/
// spartan_ability + total = compteurs kill-type API canoniques (counts,
// Authoritative=true) ; unattributed = counts.Total − Σ classes (résidu, ajouté si
// > 0). hasMechanics gate spartan_ability et le niveau 2 de Mêlée (invariant d — cap
// off ⇒ pas de spartan + Mêlée feuille).
//
// counts est un struct NEUTRE (domain.FragKillTypeCounts) : le builder est ainsi
// partagé tel quel par Synthesis (agrégat SynthesisDetailedStats), Match view
// (ligne scoreboard native du viewer) ET Escouade (agrégat par gamertag) — jamais
// dupliqué (règle ≤2 copies).
func Build(
	rows []port.WeaponKillRow,
	counts domain.FragKillTypeCounts,
	hasMechanics bool,
) domain.FragDistribution {
	byClass := make(map[string]domain.FragClassEntry, len(canonicalFragClassOrder))
	for _, e := range buildGunFragClasses(rows) {
		byClass[e.Class] = e
	}
	for _, e := range buildAPIFragClasses(counts, hasMechanics) {
		byClass[e.Class] = e
	}
	// Non attribué = résidu calculé (invariant a : Σ classes == total ; invariant c :
	// clamp >= 0). Un dépassement (Σ attribué > total) est une anomalie de données —
	// non ajouté ici, signalé par le câblage service (jamais avalé).
	sum := 0
	for _, e := range byClass {
		sum += e.Kills
	}
	if unattr := counts.Total - sum; unattr > 0 {
		byClass[domain.FragClassUnattributed] = domain.FragClassEntry{
			Class: domain.FragClassUnattributed, Kills: unattr, Authoritative: false,
		}
	}
	classes := make([]domain.FragClassEntry, 0, len(byClass))
	for _, c := range canonicalFragClassOrder {
		if e, ok := byClass[c]; ok {
			classes = append(classes, e)
		}
	}
	return domain.FragDistribution{TotalKills: counts.Total, Classes: classes}
}

// buildGunFragClasses agrège les classes gun (shoulder/sidearm/heavy) + leurs rôles
// depuis le registre (rows). Exclut les sentinels grenade/melee et les rows sans
// class/role résolu (dégradation propre, comme buildKillsByRole).
func buildGunFragClasses(rows []port.WeaponKillRow) []domain.FragClassEntry {
	type acc struct {
		kills  int
		byRole map[string]int
	}
	agg := make(map[string]*acc, len(gunFragClasses))
	for _, r := range rows {
		if r.IsGrenadeMelee || r.Class == "" || r.Role == "" || !gunFragClasses[r.Class] {
			continue
		}
		a := agg[r.Class]
		if a == nil {
			a = &acc{byRole: make(map[string]int)}
			agg[r.Class] = a
		}
		a.kills += r.Kills
		a.byRole[r.Role] += r.Kills
	}
	out := make([]domain.FragClassEntry, 0, len(agg))
	for class, a := range agg {
		if a.kills <= 0 {
			continue
		}
		out = append(out, domain.FragClassEntry{
			Class: class, Kills: a.kills, Authoritative: false,
			Roles: rolesFromMap(a.byRole, class),
		})
	}
	return out
}

// rolesFromMap trie les rôles (kills desc, tie-break alpha → ordre stable) et
// replie en FEUILLE une classe dont l'unique rôle porte son propre nom (ex. Poing/
// sidearm — D4 : pas de niveau 2 significatif).
func rolesFromMap(byRole map[string]int, class string) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, len(byRole))
	for role, kills := range byRole {
		if kills <= 0 {
			continue
		}
		roles = append(roles, domain.FragRoleEntry{Role: role, Kills: kills})
	}
	if len(roles) == 1 && roles[0].Role == class {
		return nil
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Kills != roles[j].Kills {
			return roles[i].Kills > roles[j].Kills
		}
		return roles[i].Role < roles[j].Role
	})
	return roles
}

// buildAPIFragClasses construit les classes servies par les compteurs kill-type API
// canoniques (Authoritative=true) : Mêlée, Grenade, et — si hasMechanics — Capacités
// spartanes.
func buildAPIFragClasses(counts domain.FragKillTypeCounts, hasMechanics bool) []domain.FragClassEntry {
	out := make([]domain.FragClassEntry, 0, 3)
	if mk := counts.Melee; mk > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassMelee, Kills: mk, Authoritative: true,
			Roles: meleeRoles(mk, counts.Assassination, hasMechanics),
		})
	}
	if gk := counts.Grenade; gk > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassGrenade, Kills: gk, Authoritative: true,
		})
	}
	if hasMechanics {
		if sk := counts.GroundPound + counts.ShoulderBash; sk > 0 {
			out = append(out, domain.FragClassEntry{
				Class: domain.FragClassSpartanAbility, Kills: sk, Authoritative: true,
				Roles: spartanRoles(counts.GroundPound, counts.ShoulderBash),
			})
		}
	}
	return out
}

// meleeRoles construit le niveau 2 de la classe Mêlée. FEUILLE sur Infinite (pas de
// mécanique native → invariant d). Sur H5 (hasMechanics) : Assassinat (total API) +
// Corps-à-corps direct.
//
// FORMULE PROVISOIRE : direct_melee = melee_total − assassination suppose
// assassination ⊆ melee. À CONFIRMER par la probe H5 (gate G2.3, phase P2) — si les
// deux compteurs sont disjoints, corriger ici. Cf. PLAN_FRAG_DISTRIBUTION_V2 §2.
// Clamp assassination ≤ melee pour garantir l'invariant (b) et un résidu ≥ 0.
func meleeRoles(meleeKills, assassinations int, hasMechanics bool) []domain.FragRoleEntry {
	if !hasMechanics {
		return nil
	}
	assass := assassinations
	if assass > meleeKills {
		assass = meleeKills
	}
	if assass < 0 {
		assass = 0
	}
	direct := meleeKills - assass
	roles := make([]domain.FragRoleEntry, 0, 2)
	if assass > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleAssassination, Kills: assass})
	}
	if direct > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleDirectMelee, Kills: direct})
	}
	return roles
}

// spartanRoles construit le niveau 2 des Capacités spartanes (ordre sémantique fixe :
// Frappe au sol puis Charge d'épaule). Σ rôles == kills de la classe (invariant b).
func spartanRoles(groundPound, shoulderBash int) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, 2)
	if groundPound > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleGroundPound, Kills: groundPound})
	}
	if shoulderBash > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleShoulderBash, Kills: shoulderBash})
	}
	return roles
}
