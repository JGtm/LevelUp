package mappings

// loader_replay_labels_vehicles_test.go — la validation de [[vehicle_families]] (lot 1.9.9).

import "testing"

// entete : le minimum qu'un manifeste de rejeu doit porter pour être parsé.
const enteteVehFam = "[meta]\ntitle_slug = \"halo_infinite\"\nschema_version = 1\n"

func TestVehicleFamilies_EntreeValide(t *testing.T) {
	raw := enteteVehFam + `
[[vehicle_families]]
family = "tourelle_auto_bannie"
en     = "Banished auto-turret"
fr     = "Tourelle automatique bannie"
kind   = "map_element"
sprite = false
`
	set, err := LoadReplayLabelsFromBytes("t.toml", []byte(raw))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	got := set.VehicleFamilies()
	v, ok := got["tourelle_auto_bannie"]
	if !ok {
		t.Fatalf("famille absente du set : %+v", got)
	}
	if v.En != "Banished auto-turret" || v.Fr != "Tourelle automatique bannie" {
		t.Errorf("libellés = %+v", v)
	}
	if v.Kind != VehicleFamilyKindMapElement || v.Sprite {
		t.Errorf("nature/asset = %+v, attendu map_element sans sprite", v)
	}
}

// TestVehicleFamilies_TableAbsente : le régime NORMAL — aucun titre n'est obligé d'en déclarer.
func TestVehicleFamilies_TableAbsente(t *testing.T) {
	set, err := LoadReplayLabelsFromBytes("t.toml", []byte(enteteVehFam))
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	if got := set.VehicleFamilies(); got != nil {
		t.Errorf("familles = %+v, attendu nil : une table vide dirait « regarde, il n'y a rien »", got)
	}
}

// TestVehicleFamilies_RefusDesEntreesIncompletes : la validation est STRICTE et tout-ou-rien —
// une famille à moitié déclarée produirait un rendu neutre indistinguable d'une non-déclarée.
func TestVehicleFamilies_RefusDesEntreesIncompletes(t *testing.T) {
	cas := map[string]string{
		"famille vide":   "[[vehicle_families]]\nfamily = \"\"\nen = \"A\"\nfr = \"A\"\nkind = \"map_element\"\n",
		"libelle absent": "[[vehicle_families]]\nfamily = \"x\"\nen = \"A\"\nkind = \"map_element\"\n",
		"nature inventee": "[[vehicle_families]]\nfamily = \"x\"\nen = \"A\"\nfr = \"A\"\n" +
			"kind = \"decor\"\n",
		"nature absente": "[[vehicle_families]]\nfamily = \"x\"\nen = \"A\"\nfr = \"A\"\n",
		"famille en double": "[[vehicle_families]]\nfamily = \"x\"\nen = \"A\"\nfr = \"A\"\nkind = \"map_element\"\n" +
			"[[vehicle_families]]\nfamily = \"x\"\nen = \"B\"\nfr = \"B\"\nkind = \"map_element\"\n",
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, err := LoadReplayLabelsFromBytes("t.toml", []byte(enteteVehFam+corps)); err == nil {
				t.Errorf("chargement accepté alors que l'entrée est invalide (%s)", nom)
			}
		})
	}
}
