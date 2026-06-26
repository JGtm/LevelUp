package halo_5

import (
	"testing"

	"levelup/go-api/internal/games"
)

// TestCapabilities_UsesInjected vérifie que WithCapabilities prime sur le fallback
// codé (chemin nominal : la map TOML est la source). Symétrie avec l'équivalent
// Halo Infinite (adapter_capabilities_test.go) : un pivot multi-titre doit couvrir
// WithCapabilities sur CHAQUE titre, pas seulement Infinite.
func TestCapabilities_UsesInjected(t *testing.T) {
	injected := games.CapabilityMap{
		games.CapMatchHistory: games.CapSupported,
		games.CapTimeseries:   games.CapSupported, // diffère du fallback (not_exposed)
	}
	// Factory non-nil → la map STATIQUE injectée est rendue verbatim (pas le fallback,
	// pas de rétrogradation runtime).
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil).WithCapabilities(injected)

	caps := a.Capabilities()
	if len(caps) != 2 {
		t.Fatalf("Capabilities() = %d entrées, want 2 (la map injectée, pas le fallback %d entrées)",
			len(caps), len(fallbackCapabilities()))
	}
	if !caps.Has(games.CapTimeseries) {
		t.Errorf("timeseries doit être supported (injecté), got %q", caps[games.CapTimeseries])
	}
	if !caps.Has(games.CapMatchHistory) {
		t.Errorf("match.history doit être supported (injecté), got %q", caps[games.CapMatchHistory])
	}
}

// TestCapabilities_InjectedDegradedWhenNoFactory vérifie l'INTERACTION entre la map
// injectée et la dégradation runtime propre à Halo 5 : sans source-factory câblée,
// même une map injectée "supported" est rétrogradée à not_exposed (on ne ment jamais
// sur ce qui est réellement servable live). C'est la divergence h5 vs HI (HI ne
// rétrograde que career.progression ; h5 rétrograde TOUT sans factory).
func TestCapabilities_InjectedDegradedWhenNoFactory(t *testing.T) {
	injected := games.CapabilityMap{
		games.CapMatchHistory: games.CapSupported,
		games.CapTimeseries:   games.CapSupported,
	}
	// newSource == nil → toutes les capabilities rétrogradées, y compris l'injectée.
	caps := NewDataAdapter(nil, nil).WithCapabilities(injected).Capabilities()
	if len(caps) != 2 {
		t.Fatalf("Capabilities() = %d entrées, want 2 (clés de la map injectée préservées)", len(caps))
	}
	for k, v := range caps {
		if v != games.CapNotExposed {
			t.Errorf("factory nil : capability injectée %q = %q, want not_exposed", k, v)
		}
	}
}

// TestWithCapabilities_Chainable vérifie que WithCapabilities retourne le récepteur
// (builder chaînable) — invariant du pattern adapter partagé par les deux titres.
func TestWithCapabilities_Chainable(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil)
	if got := a.WithCapabilities(games.CapabilityMap{}); got != a {
		t.Errorf("WithCapabilities doit retourner le récepteur (chaînable), got %p want %p", got, a)
	}
}
