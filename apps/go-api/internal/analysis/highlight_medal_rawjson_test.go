package analysis

import (
	"encoding/json"
	"testing"
)

// TestMedalRawJSON_FormeLue — le document doit rendre EXACTEMENT le champ que le
// lecteur extrait (platform/duckdb.medalNameFromRawJSON lit `medal_name`).
func TestMedalRawJSON_FormeLue(t *testing.T) {
	raw, err := MedalRawJSON("Odin's Raven")
	if err != nil {
		t.Fatalf("MedalRawJSON: %v", err)
	}
	var relu struct {
		MedalName string `json:"medal_name"`
	}
	if err := json.Unmarshal([]byte(raw), &relu); err != nil {
		t.Fatalf("document illisible (%q): %v", raw, err)
	}
	if relu.MedalName != "Odin's Raven" {
		t.Errorf("medal_name = %q, attendu %q", relu.MedalName, "Odin's Raven")
	}
}

// TestMedalRawJSON_EchappeLesCaracteresSpeciaux — les noms mesures portent des
// apostrophes et des esperluettes (« Tag & Bag », « Flyin' High »). Une
// concatenation de chaines les casserait ; l encodeur, non.
func TestMedalRawJSON_EchappeLesCaracteresSpeciaux(t *testing.T) {
	for _, nom := range []string{"Tag & Bag", "Flyin' High", "Fire & Forget", `guillemet " interne`} {
		raw, err := MedalRawJSON(nom)
		if err != nil {
			t.Fatalf("MedalRawJSON(%q): %v", nom, err)
		}
		var relu struct {
			MedalName string `json:"medal_name"`
		}
		if err := json.Unmarshal([]byte(raw), &relu); err != nil {
			t.Fatalf("%q -> document illisible (%q): %v", nom, raw, err)
		}
		if relu.MedalName != nom {
			t.Errorf("%q relu en %q", nom, relu.MedalName)
		}
	}
}

// TestMedalRawJSON_NomVideEstUneErreur — pas de document « renseigne » qui ne
// porte rien : l appelant sans nom laisse raw_json a NULL.
func TestMedalRawJSON_NomVideEstUneErreur(t *testing.T) {
	for _, nom := range []string{"", "   "} {
		if raw, err := MedalRawJSON(nom); err == nil {
			t.Errorf("MedalRawJSON(%q) = %q, attendu une erreur", nom, raw)
		}
	}
}
