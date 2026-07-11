package games_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// engagement_capability_mirror_test.go — garde-rail F7 (§4) : cohérence coarse↔fine
// de la capability engagement pour TOUS les titres (ferme le reliquat F15-12/L2-(3)
// pour cette capability).
//
// Invariant : un titre qui SERT l'engagement au niveau fin (engagement.score ∈
// {supported, degraded} → CapabilityMap.Has) DOIT déclarer la capability COARSE
// (title.CapEngagement), sinon le middleware RequireCapability bloque la route et
// le statut fin devient inatteignable (incohérence silencieuse). La réciproque
// (coarse sans fine servie) est tolérée : un titre peut déclarer l'intention coarse
// tout en gardant l'adaptateur en not_exposed (H5 avant E5).
func TestEngagementCoarseFineMirror(t *testing.T) {
	root := mirrorRepoRoot(t)

	for _, slug := range []string{title.DefaultSlug, "halo_5"} {
		t.Run(slug, func(t *testing.T) {
			fine := loadFineEngagement(t, root, slug)
			coarse := loadCoarseEngagement(t, root, slug)

			// Has() == true pour supported/degraded.
			served := fine == games.CapSupported || fine == games.CapDegraded
			if served && !coarse {
				t.Errorf("titre %q : engagement.score=%q (servi) mais capability COARSE title.CapEngagement ABSENTE — "+
					"la route serait bloquée par RequireCapability. Déclarer \"engagement\" dans title.toml (ou registry).", slug, fine)
			}
		})
	}
}

// loadFineEngagement lit le statut fin engagement.score depuis capabilities.toml.
func loadFineEngagement(t *testing.T, root, slug string) games.CapabilityStatus {
	t.Helper()
	path := filepath.Join(root, "config", "titles", slug, "mappings", "capabilities.toml")
	set, err := mappings.LoadCapabilitiesFromFile(path)
	if err != nil {
		t.Fatalf("LoadCapabilitiesFromFile(%s): %v", slug, err)
	}
	cm, err := games.CapabilityMapFromMappings(set)
	if err != nil {
		t.Fatalf("CapabilityMapFromMappings(%s): %v", slug, err)
	}
	return cm[games.CapEngagement]
}

// loadCoarseEngagement résout la capability coarse title.CapEngagement. halo_infinite
// est built-in (NewRegistry) ; les autres titres sont chargés depuis title.toml.
func loadCoarseEngagement(t *testing.T, root, slug string) bool {
	t.Helper()
	if slug == title.DefaultSlug {
		desc := title.NewRegistry().Get(title.DefaultSlug)
		if desc == nil {
			t.Fatalf("descripteur built-in %q introuvable", slug)
		}
		return desc.HasCapability(title.CapEngagement)
	}
	desc, err := title.LoadTitleManifest(root, slug)
	if err != nil {
		t.Fatalf("LoadTitleManifest(%s): %v", slug, err)
	}
	return desc.HasCapability(title.CapEngagement)
}

func mirrorRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// internal/games/<file> → remonter à la racine du repo (5 niveaux).
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
}
