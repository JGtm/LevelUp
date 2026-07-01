package games

import "testing"

// Réutilise stubCapResolver / resolverSansCapability définis dans
// career_progression_source_test.go (même package).

func TestProvidesBattlePass_TitleAvecCapability(t *testing.T) {
	res := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"halo_infinite": {CapBattlePass: CapSupported},
		},
	}
	if !ProvidesBattlePassFromResolver(res, "halo_infinite") {
		t.Fatal("titre avec battlepass.progression supported doit fournir la surface Battle Pass")
	}
}

func TestProvidesBattlePass_TitleSansCapability(t *testing.T) {
	res := &stubCapResolver{
		caps: map[string]CapabilityMap{
			// Halo 5 : capabilities déclarées mais BP/Challenges not_exposed.
			"halo_5": {CapBattlePass: CapNotExposed, CapChallenges: CapNotExposed},
		},
	}
	if ProvidesBattlePassFromResolver(res, "halo_5") {
		t.Fatal("titre sans battlepass.progression NE doit PAS fournir la surface Battle Pass")
	}
	if ProvidesChallengesFromResolver(res, "halo_5") {
		t.Fatal("titre sans challenges.surface NE doit PAS fournir la surface Challenges")
	}
}

func TestProvidesChallenges_TitleAvecCapability(t *testing.T) {
	res := &stubCapResolver{
		caps: map[string]CapabilityMap{
			"halo_infinite": {CapChallenges: CapSupported},
		},
	}
	if !ProvidesChallengesFromResolver(res, "halo_infinite") {
		t.Fatal("titre avec challenges.surface supported doit fournir la surface Challenges")
	}
}

func TestProvidesLiveService_TitleSansCapabilitiesDeclarees_DefautTrue(t *testing.T) {
	res := &stubCapResolver{
		caps:  map[string]CapabilityMap{},
		found: map[string]bool{"halo_5": false},
	}
	if !ProvidesBattlePassFromResolver(res, "halo_5") {
		t.Fatal("titre sans capabilities déclarées doit retomber sur le défaut true (BP)")
	}
	if !ProvidesChallengesFromResolver(res, "halo_5") {
		t.Fatal("titre sans capabilities déclarées doit retomber sur le défaut true (Challenges)")
	}
}

func TestProvidesLiveService_ResolverSansExtension_DefautTrue(t *testing.T) {
	if !ProvidesBattlePassFromResolver(resolverSansCapability{}, "halo_5") {
		t.Fatal("resolver sans CapabilityResolver doit retomber sur le défaut true (BP)")
	}
	if !ProvidesChallengesFromResolver(resolverSansCapability{}, "halo_5") {
		t.Fatal("resolver sans CapabilityResolver doit retomber sur le défaut true (Challenges)")
	}
}

func TestProvidesLiveService_ResolverNil_DefautTrue(t *testing.T) {
	if !ProvidesBattlePassFromResolver(nil, "halo_5") {
		t.Fatal("resolver nil doit retomber sur le défaut true (byte-identique Infinite, BP)")
	}
	if !ProvidesChallengesFromResolver(nil, "halo_5") {
		t.Fatal("resolver nil doit retomber sur le défaut true (byte-identique Infinite, Challenges)")
	}
}
