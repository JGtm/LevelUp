package mappings

// loader_impact_test.go — LE FAIT « LE PROJECTILE EXPLOSE A L'IMPACT » (2026-10-25), declare par
// arme dans les deux registres du titre : `[shot_impact]` de replay_labels.toml (armes a pied) et
// `impact` de vehicle_weapons.toml (armes de vehicule).

import (
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

func TestShotImpact_ArmesAPiedDeclarees(t *testing.T) {
	racine := vwRepoRoot(t)
	set, err := LoadReplayLabelsFromFile(filepath.Join(racine, "config", "titles", "halo_infinite", "mappings", "replay_labels.toml"))
	if err != nil {
		t.Fatalf("replay_labels.toml : %v", err)
	}
	var got []string
	for k := range set.ShotImpacts() {
		got = append(got, k)
	}
	sort.Strings(got)
	want := []string{"hinf_cindershot", "hinf_fuel_rod_spnkr", "hinf_hydra", "hinf_m41_spnkr"}
	if !slices.Equal(got, want) {
		t.Errorf("armes a pied qui explosent = %v, attendu %v", got, want)
	}
	// Le plasma n'explose pas (decision du 2026-10-25), quelle que soit sa forme de rendu.
	for _, k := range []string{"hinf_ravager", "hinf_plasma_pistol"} {
		if set.ShotImpacts()[k] {
			t.Errorf("%s ne doit pas exploser", k)
		}
	}
}

func TestShotImpact_CleHorsShotEffectsRefusee(t *testing.T) {
	const toml = "[meta]\ntitle_slug = \"t\"\nschema_version = 1\n[shot_effects]\nhinf_a = \"ballistic\"\n[shot_impact]\nhinf_b = true\n"
	if _, err := LoadReplayLabelsFromBytes("t.toml", []byte(toml)); err == nil {
		t.Error("cle absente de [shot_effects] acceptee, attendue refusee")
	}
	const ok = "[meta]\ntitle_slug = \"t\"\nschema_version = 1\n[shot_effects]\nhinf_a = \"explosive\"\n[shot_impact]\nhinf_a = true\n"
	set, err := LoadReplayLabelsFromBytes("t.toml", []byte(ok))
	if err != nil || !set.ShotImpacts()["hinf_a"] {
		t.Errorf("entree valide : err=%v impacts=%v", err, set.ShotImpacts())
	}
}

func TestRegistreArmesVehicule_Impact(t *testing.T) {
	set := vwRegistre(t)
	// Mortier du Wraith, Rockethog, missiles du Wasp, bombe de la Banshee : explosent. Le canon du
	// Scorpion (aucun projectile publie) et le lance-grenades du Falcon n'ont pas ete retenus.
	for tag, want := range map[string]bool{
		"121B4009": true, "C7D50912": true, "11725DC4": true, "850902EF": true,
		"49E40D17": false, "0BB6976B": false, "0042678E": false,
	} {
		w, ok := set.Weapon(tag)
		if !ok {
			t.Fatalf("%s absent du registre", tag)
		}
		if w.Impact != want {
			t.Errorf("%s (%s) : impact = %v, attendu %v", tag, w.Vehicle, w.Impact, want)
		}
	}
}

func TestRegistreArmesVehicule_ImpactExigeUnTirAuCoup(t *testing.T) {
	const continu = `[[weapons]]
tag = "00015CD3"
vehicle = "falcon"
label = { en = "a", fr = "b" }
fire = "continuous"
fx = "ballistic"
tint = "kinetic"
sound = "s"
proof = "p"
impact = true
`
	if _, err := LoadVehicleWeaponsFromBytes("t.toml", []byte(continu)); err == nil {
		t.Error("impact sur un tir continu accepte, attendu refuse")
	}
}
