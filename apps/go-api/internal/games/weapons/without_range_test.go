package weapons

// without_range_test.go — GARDE-RAIL de l'attribut « sans portée ». Purement en mémoire.

import "testing"

func TestWithoutRange_ClesConnuesDuRegistre(t *testing.T) {
	connues := make(map[string]bool, len(weaponRegistryWeapons))
	for _, w := range weaponRegistryWeapons {
		connues[w.key] = true
	}
	for key := range weaponKeysWithoutRange {
		if !connues[key] {
			t.Errorf("%q est déclarée sans portée mais n'existe pas dans le registre d'armes", key)
		}
	}
}

func TestWithoutRange_ArmesDeContactEtEnvironnement(t *testing.T) {
	for _, key := range []string{keyHinfEnergySword, keyHinfGravityHammer, keyHinfUnarmed, keyHinfEnvironment} {
		if !IsWithoutRange(key) {
			t.Errorf("IsWithoutRange(%q) = false, attendu true", key)
		}
	}
	// Témoins négatifs : une arme à distance, un lance-roquettes du même rôle que l'épée,
	// une clé inconnue.
	for _, key := range []string{"hinf_br75", "hinf_m41_spnkr", "inconnue"} {
		if IsWithoutRange(key) {
			t.Errorf("IsWithoutRange(%q) = true, attendu false", key)
		}
	}
}

// L'attribut ne reclasse rien : l'épée et le marteau d'Infinite restent des armes lourdes de
// rôle `power` pour la répartition des frags, le Face-à-face et l'Explorer.
func TestWithoutRange_NeChangeNiClasseNiRole(t *testing.T) {
	roles, classes := RolesByKey(), ClassesByKey()
	for _, key := range []string{keyHinfEnergySword, keyHinfGravityHammer} {
		if roles[key] != rolePower || classes[key] != clsHeavy {
			t.Errorf("%q : rôle %q / classe %q, attendu %q / %q", key, roles[key], classes[key], rolePower, clsHeavy)
		}
	}
}
