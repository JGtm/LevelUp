package games

import "testing"

// stubCapResolver implémente EndpointResolver + CapabilityResolver pour piloter
// le gating sans charger de TOML réel.
type stubCapResolver struct {
	caps  map[string]CapabilityMap
	found map[string]bool
}

func (s *stubCapResolver) HostFor(string, EndpointKey) (string, bool) { return "", false }

func (s *stubCapResolver) CapabilitiesFor(slug string) (CapabilityMap, bool) {
	if s.found != nil && !s.found[slug] {
		return nil, false
	}
	c, ok := s.caps[slug]
	return c, ok
}

// resolverSansCapability implémente seulement EndpointResolver (pas l'extension)
// pour vérifier le défaut byte-identique Infinite.
type resolverSansCapability struct{}

func (resolverSansCapability) HostFor(string, EndpointKey) (string, bool) { return "", false }

func TestProvidesLiveCareerProgression_TitleAvecCatalogue(t *testing.T) {
	res := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"halo_infinite": {CapCareerRankCatalog: CapSupported},
		},
	}
	if !ProvidesLiveCareerProgressionFromResolver(res, "halo_infinite") {
		t.Fatal("titre avec career.rank_catalog supported doit fournir le live careerranks")
	}
}

func TestProvidesLiveCareerProgression_TitleSansCatalogue(t *testing.T) {
	// Halo 5 : déclare ses capabilities mais SANS career.rank_catalog → l'endpoint
	// economy careerranks ne le concerne pas → le live doit être court-circuité.
	res := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"halo_5": {CapCareerProgression: CapSupported}, // pas de CapCareerRankCatalog
		},
	}
	if ProvidesLiveCareerProgressionFromResolver(res, "halo_5") {
		t.Fatal("titre sans career.rank_catalog NE doit PAS fournir le live careerranks")
	}
}

func TestProvidesLiveCareerProgression_TitleInconnu_DefautTrue(t *testing.T) {
	// Titre absent du resolver (found=false) → défaut sûr = supposer Infinite.
	res := &stubCapResolver{
		caps:  map[string]CapabilityMap{},
		found: map[string]bool{"halo_5": false},
	}
	if !ProvidesLiveCareerProgressionFromResolver(res, "halo_5") {
		t.Fatal("titre sans capabilities déclarées doit retomber sur le défaut true")
	}
}

func TestProvidesLiveCareerProgression_ResolverSansExtension_DefautTrue(t *testing.T) {
	if !ProvidesLiveCareerProgressionFromResolver(resolverSansCapability{}, "halo_5") {
		t.Fatal("resolver sans CapabilityResolver doit retomber sur le défaut true")
	}
}

func TestProvidesLiveCareerProgression_ResolverNil_DefautTrue(t *testing.T) {
	if !ProvidesLiveCareerProgressionFromResolver(nil, "halo_5") {
		t.Fatal("resolver nil doit retomber sur le défaut true (byte-identique Infinite)")
	}
}
