package ops

import (
	"context"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// TestTitleDataChecks_PerTitleLabeling prouve le cœur de MT-10 : titleDataChecks
// produit des contrôles dépendant du titre dont le NOM est préfixé par le slug
// quand plusieurs titres coexistent (labelTitle=true), et NON préfixé en mono-titre
// (labelTitle=false → sortie byte-identique au comportement historique).
//
// On n'asserte pas le statut OK (le TempDir est vide → tout KO) mais les NOMS :
// c'est la dimension titre qui est testée, indépendamment de la présence des DBs.
func TestTitleDataChecks_PerTitleLabeling(t *testing.T) {
	pr := titlePkg.NewPathResolver(t.TempDir())
	ctx := context.Background()

	single := titleDataChecks(ctx, pr, titlePkg.DefaultSlug, false)
	if len(single) == 0 {
		t.Fatal("titleDataChecks mono-titre : aucun contrôle produit")
	}
	for _, c := range single {
		if strings.Contains(c.Name, "/") {
			t.Errorf("mono-titre : nom préfixé inattendu %q (doit rester byte-identique)", c.Name)
		}
	}
	// Repères : les contrôles de base attendus en mono-titre.
	if !hasCheck(single, "shared_matches_v2") || !hasCheck(single, "metadata") {
		t.Errorf("mono-titre : noms de base manquants dans %v", names(single))
	}

	multi := titleDataChecks(ctx, pr, "synthetic_title_b", true)
	if len(multi) != len(single) {
		t.Fatalf("multi vs mono : %d vs %d contrôles (même structure attendue)", len(multi), len(single))
	}
	for _, c := range multi {
		if !strings.HasPrefix(c.Name, "synthetic_title_b/") {
			t.Errorf("multi-titre : nom non préfixé %q (attendu slug/...)", c.Name)
		}
	}
	if !hasCheck(multi, "synthetic_title_b/shared_matches_v2") {
		t.Errorf("multi-titre : contrôle shared_matches_v2 du 2e titre absent dans %v", names(multi))
	}
}

func hasCheck(cs []HealthCheck, name string) bool {
	for _, c := range cs {
		if c.Name == name {
			return true
		}
	}
	return false
}

func names(cs []HealthCheck) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name
	}
	return out
}
