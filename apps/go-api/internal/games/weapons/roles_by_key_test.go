package weapons

// roles_by_key_test.go — GARDE-RAIL de `RolesByKey` (lot armes au sol, 2026-09-10).
//
// PUREMENT EN MÉMOIRE (aucun `go:build cgo`, contrairement aux tests du registre qui ouvrent
// une DuckDB : `RolesByKey` ne lit que le slice Go seedé au build) — c'est exactement ce qui
// permet au service `replay_weapon_labels.go` de l'appeler à la requête sans coût.

import "testing"

func TestRolesByKey_ConnaitLesTroisRolesDuFiltreArmesSpeciales(t *testing.T) {
	roles := RolesByKey()
	cas := map[string]string{
		"hinf_s7_sniper": roleSniper,  // sniper : ce qu'un socle distribue et qu'on ramasse.
		"hinf_m41_spnkr": rolePower,   // power : lance-roquettes.
		"hinf_needler":   roleSpecial, // special : arme hors les deux autres rôles distinctifs.
		"hinf_ma40_ar":   roleAuto,    // témoin négatif : une arme de départ, pas « spéciale ».
	}
	for key, want := range cas {
		if got := roles[key]; got != want {
			t.Errorf("role[%q] = %q, attendu %q", key, got, want)
		}
	}
}

func TestRolesByKey_ToutesLesArmesDuRegistreSontCouvertes(t *testing.T) {
	roles := RolesByKey()
	if len(roles) != len(weaponRegistryWeapons) {
		t.Fatalf("%d rôles pour %d armes du registre — une clé du registre s'est dupliquée ou "+
			"perdue", len(roles), len(weaponRegistryWeapons))
	}
	for _, w := range weaponRegistryWeapons {
		if roles[w.key] != w.role {
			t.Errorf("role[%q] = %q, attendu %q (registre)", w.key, roles[w.key], w.role)
		}
	}
}
