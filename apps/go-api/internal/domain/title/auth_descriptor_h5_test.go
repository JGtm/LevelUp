package title

import (
	"path/filepath"
	"runtime"
	"testing"
)

// TestLoadAuthDescriptor_Halo5_NoClearance verrouille le contrat « clearance
// optionnel » (review Phase 1a, finding #5) : le VRAI auth.toml halo_5, qui
// déclare clearance_url="" (Halo 5 n'utilise pas le 343-clearance — ClearanceAware
// :false confirmé par la sonde), doit charger SANS erreur. Garde-fou avant le
// cutover status=active (sinon l'onboarding/refresh Halo 5 casserait).
func TestLoadAuthDescriptor_Halo5_NoClearance(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")

	desc, err := LoadAuthDescriptor(repoRoot, "halo_5")
	if err != nil {
		t.Fatalf("LoadAuthDescriptor(halo_5) doit réussir avec clearance_url vide: %v", err)
	}
	if desc.ClearanceURL != "" {
		t.Errorf("ClearanceURL = %q, want vide (Halo 5 sans clearance)", desc.ClearanceURL)
	}
	// Les audiences (mirroir Infinite, confirmé sonde) doivent être présentes.
	if desc.SpartanAudience == "" || desc.SpartanTokenURL == "" || desc.XSTSAudience == "" {
		t.Errorf("descripteur halo_5 incomplet: %+v", desc)
	}
}
