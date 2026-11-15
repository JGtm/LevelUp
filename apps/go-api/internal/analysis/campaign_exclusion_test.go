package analysis

import (
	"testing"
)

// TestCampaignExcludedVariantIDs : accès à la source unique, copie défensive.
func TestCampaignExcludedVariantIDs(t *testing.T) {
	ids := CampaignExcludedVariantIDs("halo_5")
	if len(ids) != 2 {
		t.Fatalf("halo_5: attendu 2 GUID Campagne, obtenu %d", len(ids))
	}
	if CampaignExcludedVariantIDs("halo_infinite") != nil {
		t.Errorf("halo_infinite: attendu nil (aucun mode masqué)")
	}
	// Mutation du retour ne doit pas corrompre la source.
	ids[0] = "MUTATED"
	if CampaignExcludedVariantIDs("halo_5")[0] == "MUTATED" {
		t.Errorf("CampaignExcludedVariantIDs doit retourner une copie défensive")
	}
}
