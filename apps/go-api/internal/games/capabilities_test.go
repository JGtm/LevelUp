package games

import (
	"testing"

	"levelup/go-api/internal/games/mappings"
)

func TestAllCapabilityKeys_Count(t *testing.T) {
	// Garde-fou : si une CapabilityKey est ajoutée sans mettre à jour
	// AllCapabilityKeys(), ce compteur le signale.
	if got := len(AllCapabilityKeys()); got != 24 {
		t.Errorf("AllCapabilityKeys() = %d clés, want 24 (mettre à jour si ajout de capability)", got)
	}
}

func TestCapabilityMapFromMappings_OK(t *testing.T) {
	set := mappings.NewCapabilityMappingSet("halo_infinite", 1, map[string]string{
		string(CapMatchHistory):       mappings.CapStatusSupported,
		string(CapTimeseries):         mappings.CapStatusNotExposed,
		string(CapMatchSkillSnapshot): mappings.CapStatusDegraded,
	})
	cm, err := CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings: %v", err)
	}
	if cm[CapMatchHistory] != CapSupported {
		t.Errorf("match.history = %q, want supported", cm[CapMatchHistory])
	}
	if cm.Has(CapTimeseries) {
		t.Errorf("timeseries not_exposed ne doit pas Has()")
	}
	if !cm.Has(CapMatchSkillSnapshot) {
		t.Errorf("skill.snapshot degraded doit Has()")
	}
}

func TestCapabilityMapFromMappings_UnknownKey(t *testing.T) {
	set := mappings.NewCapabilityMappingSet("x", 1, map[string]string{
		"made.up.capability": mappings.CapStatusSupported,
	})
	if _, err := CapabilityMapFromMappings(set); err == nil {
		t.Fatalf("attendu une erreur pour clé de capability inconnue")
	}
}

func TestCapabilityMapFromMappings_InvalidStatus(t *testing.T) {
	set := mappings.NewCapabilityMappingSet("x", 1, map[string]string{
		string(CapMatchHistory): "bogus",
	})
	if _, err := CapabilityMapFromMappings(set); err == nil {
		t.Fatalf("attendu une erreur pour statut invalide")
	}
}

func TestCapabilityMapFromMappings_Nil(t *testing.T) {
	if _, err := CapabilityMapFromMappings(nil); err == nil {
		t.Fatalf("attendu une erreur pour set nil")
	}
}
