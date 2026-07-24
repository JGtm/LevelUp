package ops

import (
	"strings"
	"testing"
)

// seed_citation_mappings_test.go — garde-rails sur l'état DÉCIDÉ (décisions
// utilisateur I7, 2026-07-24) de trois citations : player_vs_everything (victoires
// Firefight), road_trip (médaille Splatter), flag_defender (désactivée). Pure data
// (defaultCitationMappings, sans DB) → hors build cgo. Fige la décision pour qu'un
// futur remap accidentel casse le test.

// citationByNorm retourne la CitationMapping seedée pour un norm donné, ou échoue.
func citationByNorm(t *testing.T, norm string) CitationMapping {
	t.Helper()
	for _, m := range defaultCitationMappings() {
		if m.Norm == norm {
			return m
		}
	}
	t.Fatalf("citation %q absente de defaultCitationMappings()", norm)
	return CitationMapping{}
}

// TestPlayerVsEverything_CountsFirefightWins : « Éliminations Firefight » compte les
// VICTOIRES en Baptême du feu via la fonction custom compute_wins_firefight (décision
// I7). L'ancien stat_name total_enemy_kills (kills) doit avoir disparu.
func TestPlayerVsEverything_CountsFirefightWins(t *testing.T) {
	m := citationByNorm(t, "player_vs_everything")
	if m.MappingType != mappingTypeCustom {
		t.Errorf("MappingType = %q, want %q", m.MappingType, mappingTypeCustom)
	}
	if m.CustomFunction != "compute_wins_firefight" {
		t.Errorf("CustomFunction = %q, want compute_wins_firefight", m.CustomFunction)
	}
	if m.StatName != "" {
		t.Errorf("StatName = %q, want vide (plus de total_enemy_kills)", m.StatName)
	}
	if !m.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if m.TierTargets != tierTargets5_10_15_25_50 {
		t.Errorf("TierTargets = %q, want %q (paliers de victoires)", m.TierTargets, tierTargets5_10_15_25_50)
	}
}

// TestRoadTrip_MappedToSplatterMedal : « Virée sur la route » remappée sur la médaille
// d'écrasement Splatter (221693153), présente au catalogue Infinite (décision I7). La
// cible doit rester exactement la médaille de la citation `splatter` (cohérence).
func TestRoadTrip_MappedToSplatterMedal(t *testing.T) {
	const splatterMedalID = 221693153
	m := citationByNorm(t, "road_trip")
	if m.MappingType != mappingTypeMedal {
		t.Errorf("MappingType = %q, want %q", m.MappingType, mappingTypeMedal)
	}
	if m.MedalID != splatterMedalID {
		t.Errorf("MedalID = %d, want %d (Splatter)", m.MedalID, splatterMedalID)
	}
	if !m.Enabled {
		t.Errorf("Enabled = false, want true")
	}
	if sp := citationByNorm(t, "splatter"); sp.MedalID != splatterMedalID {
		t.Errorf("splatter.MedalID = %d, want %d — road_trip doit cibler la même médaille Splatter", sp.MedalID, splatterMedalID)
	}
}

// TestFlagDefender_Disabled : « Défenseur du drapeau » désactivée — aucun award
// d'ingestion ne mesure la défense de son propre drapeau (décision I7). Vérifie aussi
// qu'elle n'est enfant d'aucun composite (sa désactivation ne rend rien inatteignable).
func TestFlagDefender_Disabled(t *testing.T) {
	m := citationByNorm(t, "flag_defender")
	if m.Enabled {
		t.Errorf("flag_defender.Enabled = true, want false (désactivée)")
	}
	for _, c := range defaultCitationMappings() {
		if c.MappingType != mappingTypeComposite {
			continue
		}
		if strings.Contains(c.CompositeChildren, `"flag_defender"`) {
			t.Errorf("composite %q référence flag_defender désactivée → potentiellement inatteignable", c.Norm)
		}
	}
}
