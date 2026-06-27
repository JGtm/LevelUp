package synthetic_title_b

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// TestSkeleton_TitleManifestComplete : le squelette 2e titre porte un title.toml
// valide, découvrable par le registre piloté par config, en coming_soon avec les
// 13 capabilities (« même niveau d'information que Halo » : 11 produit + team_mmr
// + damage_taken, miroir des flags scalaires Provides*).
func TestSkeleton_TitleManifestComplete(t *testing.T) {
	t.Parallel()
	desc, err := title.LoadTitleManifest(repoRoot(t), TitleSlug)
	if err != nil {
		t.Fatalf("LoadTitleManifest(%s): %v", TitleSlug, err)
	}
	if desc.Slug != TitleSlug {
		t.Errorf("slug = %q", desc.Slug)
	}
	if desc.Status != title.StatusComingSoon {
		t.Errorf("status = %q, want coming_soon (squelette non servi en prod)", desc.Status)
	}
	if got := len(desc.Capabilities); got != 13 {
		t.Errorf("capabilities = %d, want 13 (11 produit + team_mmr + damage_taken)", got)
	}
	if !desc.HasCapability(title.CapLUSR) || !desc.HasCapability(title.CapWorldLeaderboard) {
		t.Errorf("capabilities incomplètes: %v", desc.Capabilities)
	}
	// Miroir title-level des flags scalaires (squelette = défaut true comme Halo).
	if !desc.HasCapability(title.CapTeamMMR) || !desc.HasCapability(title.CapDamageTaken) {
		t.Errorf("team_mmr/damage_taken manquantes: %v", desc.Capabilities)
	}
	if desc.IsDefault {
		t.Error("le squelette ne doit pas être is_default")
	}
}

// TestSkeleton_RegistryDiscoversAsComingSoon : le registre piloté par config
// découvre le squelette (built-in Halo + synthetic_title_b coming_soon).
func TestSkeleton_RegistryDiscoversAsComingSoon(t *testing.T) {
	reg := title.NewRegistry()
	errs := title.LoadTitlesIntoRegistry(reg, repoRoot(t), nil)
	if len(errs) != 0 {
		t.Fatalf("LoadTitlesIntoRegistry: %v", errs)
	}
	if !reg.Exists(TitleSlug) {
		t.Fatal("synthetic_title_b non découvert depuis sa config")
	}
	// coming_soon : listé dans le switcher (NonArchived) mais PAS actif (pas servi
	// ni provisionné).
	if reg.IsActive(TitleSlug) {
		t.Error("le squelette ne doit PAS être actif (coming_soon)")
	}
	var inNonArchived bool
	for _, td := range reg.NonArchived() {
		if td.Slug == TitleSlug {
			inNonArchived = true
		}
	}
	if !inNonArchived {
		t.Error("le squelette devrait apparaître dans le switcher (NonArchived)")
	}
}

// TestSkeleton_CapabilitiesTOMLAllSupported : capabilities.toml fines déclarées
// (même niveau qu'Halo) chargent proprement.
func TestSkeleton_CapabilitiesTOMLAllSupported(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "config", "titles", TitleSlug, "mappings", "capabilities.toml")
	set, err := mappings.LoadCapabilitiesFromFile(path)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile: %v", err)
	}
	if len(set.Keys()) == 0 {
		t.Error("aucune capability fine chargée")
	}
}

// TestSkeleton_EndpointsTOMLRoutesDistinct : constants.toml [endpoints] route vers
// des hosts DISTINCTS d'Halo (oracle b : le seam route vraiment).
func TestSkeleton_EndpointsTOMLRoutesDistinct(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), "config", "titles", TitleSlug, "constants.toml")
	set, err := mappings.LoadEndpointsFromFile(path)
	if err != nil {
		t.Fatalf("LoadEndpointsFromFile: %v", err)
	}
	host, ok := set.Host(mappings.EndpointStats)
	if !ok {
		t.Fatal("endpoint stats absent")
	}
	if host == "https://halostats.svc.halowaypoint.com:443" {
		t.Error("le squelette doit router vers SON host (example.test), pas celui d'Halo")
	}
}

// TestSkeleton_AuthDescriptorDistinct : auth.toml route un descripteur auth distinct
// d'Halo (audiences différentes).
func TestSkeleton_AuthDescriptorDistinct(t *testing.T) {
	t.Parallel()
	desc, err := title.LoadAuthDescriptor(repoRoot(t), TitleSlug)
	if err != nil {
		t.Fatalf("LoadAuthDescriptor: %v", err)
	}
	halo := title.DefaultHaloAuthDescriptor()
	if desc.XSTSAudience == halo.XSTSAudience || desc.SpartanAudience == halo.SpartanAudience {
		t.Error("le descripteur auth du squelette doit diverger d'Halo (audiences)")
	}
}
