package games_test

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// citations_capability_mirror_test.go — cohérence de la capability de titre `citations`
// (title.CapCitations : gouverne la route /pages/citations, la page web et la bascule
// Citations/Commendations du front) avec la capability de donnée `citations.engine`
// (capabilities.toml : ce que l'adapter produit).
//
// Contrairement à l'engagement (où la capability coarse peut précéder la fine), les deux
// sont ici LA MÊME question — « ce titre sert-il le moteur de citations calculées ? » :
//   - citations.engine servi sans `citations` ⇒ la route est fermée alors que le moteur
//     produit : la page est inatteignable ;
//   - `citations` sans citations.engine servi ⇒ la page s'ouvre sur un moteur absent :
//     page vide au lieu des commendations natives.
//
// L'égalité est donc exigée dans les deux sens, pour tous les titres du dépôt.
func TestCitationsCoarseFineMirror(t *testing.T) {
	root := mirrorRepoRoot(t)

	for _, slug := range []string{title.DefaultSlug, "halo_5"} {
		t.Run(slug, func(t *testing.T) {
			set, err := mappings.LoadCapabilitiesFromFile(
				filepath.Join(root, "config", "titles", slug, "mappings", "capabilities.toml"))
			if err != nil {
				t.Fatalf("LoadCapabilitiesFromFile(%s): %v", slug, err)
			}
			cm, err := games.CapabilityMapFromMappings(set)
			if err != nil {
				t.Fatalf("CapabilityMapFromMappings(%s): %v", slug, err)
			}
			fine := cm[games.CapCitationsEngine]
			served := fine == games.CapSupported || fine == games.CapDegraded

			coarse := loadCoarseCapability(t, root, slug, title.CapCitations)
			if served != coarse {
				t.Errorf("titre %q : citations.engine=%q (servi=%v) mais title.CapCitations déclarée=%v — "+
					"les deux doivent concorder (déclarer/retirer \"citations\" dans title.toml ou registry.go).",
					slug, fine, served, coarse)
			}
		})
	}
}

// loadCoarseCapability résout une capability coarse : halo_infinite est built-in
// (NewRegistry), les autres titres sont chargés depuis title.toml.
func loadCoarseCapability(t *testing.T, root, slug string, cap title.Capability) bool {
	t.Helper()
	if slug == title.DefaultSlug {
		desc := title.NewRegistry().Get(title.DefaultSlug)
		if desc == nil {
			t.Fatalf("descripteur built-in %q introuvable", slug)
		}
		return desc.HasCapability(cap)
	}
	desc, err := title.LoadTitleManifest(root, slug)
	if err != nil {
		t.Fatalf("LoadTitleManifest(%s): %v", slug, err)
	}
	return desc.HasCapability(cap)
}
