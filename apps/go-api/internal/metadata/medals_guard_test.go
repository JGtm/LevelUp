package metadata

import (
	"testing"
)

// ── Cardinality Guard ─────────────────────────────────────────────────────────

func TestMedalImport_GuardCardinalityFail(t *testing.T) {
	// Local a 100 médailles, Waypoint en retourne 150 → écart 50% > seuil 10%
	r := CheckCardinalityGuard(150, 100, 10.0)
	if r.Passed {
		t.Errorf("expected cardinality guard to FAIL (150 vs 100), got Passed")
	}
}

func TestMedalImport_GuardCardinalityPass(t *testing.T) {
	// Local 100, Waypoint 105 → écart 5% < seuil 10%
	r := CheckCardinalityGuard(105, 100, 10.0)
	if !r.Passed {
		t.Errorf("expected cardinality guard to PASS (105 vs 100), got: %s", r.Reason)
	}
}

func TestMedalImport_GuardCardinalityFirstImport(t *testing.T) {
	// Local vide → premier import, toujours accepté
	r := CheckCardinalityGuard(200, 0, 10.0)
	if !r.Passed {
		t.Errorf("expected cardinality guard to PASS on first import, got: %s", r.Reason)
	}
}

func TestMedalImport_GuardCardinalityWaypointEmpty(t *testing.T) {
	// Waypoint retourne 0 médailles → erreur probable
	r := CheckCardinalityGuard(0, 100, 10.0)
	if r.Passed {
		t.Errorf("expected cardinality guard to FAIL when Waypoint returns 0")
	}
}

// ── Required Fields Guard ─────────────────────────────────────────────────────

func TestMedalImport_GuardMissingFields(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "Killing Spree", Category: "Spree", Rarity: "Legendary"},
		{MedalID: 2, Label: "", Category: "Multi", Rarity: "Rare"},      // label manquant
		{MedalID: 0, Label: "Ghost", Category: "Style", Rarity: "Epic"}, // medal_id=0
	}
	r := CheckRequiredFieldsGuard(entries)
	if r.Passed {
		t.Errorf("expected fields guard to FAIL (missing label + medal_id=0), got Passed")
	}
	if len(r.Details) != 2 {
		t.Errorf("expected 2 detail items, got %d: %v", len(r.Details), r.Details)
	}
}

func TestMedalImport_GuardAllFieldsPresent(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "Killing Spree", Category: "Spree", Rarity: "Legendary"},
		{MedalID: 2, Label: "Double Kill", Category: "Multi", Rarity: "Rare"},
	}
	r := CheckRequiredFieldsGuard(entries)
	if !r.Passed {
		t.Errorf("expected fields guard to PASS, got: %s", r.Reason)
	}
}

// ── Image Guard ───────────────────────────────────────────────────────────────

func TestMedalImport_GuardPartialImages(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "A", Category: "C", Rarity: "R", ImageURL: "https://img/1.png"},
		{MedalID: 2, Label: "B", Category: "C", Rarity: "R", ImageURL: ""},
		{MedalID: 3, Label: "C", Category: "C", Rarity: "R", ImageURL: "https://img/3.png"},
	}
	r := CheckImageGuard(entries)
	if r.Passed {
		t.Errorf("expected image guard to FAIL (2/3 images = partial), got Passed")
	}
}

func TestMedalImport_GuardAllImages(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "A", Category: "C", Rarity: "R", ImageURL: "https://img/1.png"},
		{MedalID: 2, Label: "B", Category: "C", Rarity: "R", ImageURL: "https://img/2.png"},
	}
	r := CheckImageGuard(entries)
	if !r.Passed {
		t.Errorf("expected image guard to PASS (all images present), got: %s", r.Reason)
	}
}

func TestMedalImport_GuardNoImages(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "A", Category: "C", Rarity: "R"},
		{MedalID: 2, Label: "B", Category: "C", Rarity: "R"},
	}
	r := CheckImageGuard(entries)
	if !r.Passed {
		t.Errorf("expected image guard to PASS (no images = accepted), got: %s", r.Reason)
	}
}

// ── Full Pass (RunAllGuards) ──────────────────────────────────────────────────

func TestMedalImport_FullPassPromotes(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "Killing Spree", Category: "Spree", Rarity: "Legendary", ImageURL: "https://img/1.png"},
		{MedalID: 2, Label: "Double Kill", Category: "Multi", Rarity: "Rare", ImageURL: "https://img/2.png"},
		{MedalID: 3, Label: "Overkill", Category: "Multi", Rarity: "Epic", ImageURL: "https://img/3.png"},
	}
	// localCount=3, waypointCount=3 → écart 0%, tous champs OK, toutes images OK
	r := RunAllGuards(entries, 3)
	if !r.Passed {
		t.Errorf("expected all guards to PASS for valid import, got: %s", r.Reason)
	}
}

func TestMedalImport_FullFailsOnCardinality(t *testing.T) {
	entries := make([]MedalEntry, 50)
	for i := range entries {
		entries[i] = MedalEntry{MedalID: int64(i + 1), Label: "M", Category: "C", Rarity: "R"}
	}
	// localCount=100, waypointCount=50 → écart 50% > 10%
	r := RunAllGuards(entries, 100)
	if r.Passed {
		t.Errorf("expected RunAllGuards to FAIL on cardinality, got Passed")
	}
}

func TestMedalImport_FullFailsOnFields(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "", Category: "C", Rarity: "R"}, // label vide
	}
	// localCount=0 (premier import) → cardinality OK, mais fields fail
	r := RunAllGuards(entries, 0)
	if r.Passed {
		t.Errorf("expected RunAllGuards to FAIL on missing fields, got Passed")
	}
}

func TestMedalImport_FullFailsOnPartialImages(t *testing.T) {
	entries := []MedalEntry{
		{MedalID: 1, Label: "A", Category: "C", Rarity: "R", ImageURL: "https://img/1.png"},
		{MedalID: 2, Label: "B", Category: "C", Rarity: "R", ImageURL: ""},
	}
	// localCount=0, fields OK, mais images partielles
	r := RunAllGuards(entries, 0)
	if r.Passed {
		t.Errorf("expected RunAllGuards to FAIL on partial images, got Passed")
	}
}
