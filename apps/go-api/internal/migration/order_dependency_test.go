package migration

import "testing"

// order_dependency_test.go — garde-fou de DIRECTION de dépendance (Phase 1.5 / MT-23).
//
// Issu de la revue adversariale ultracode 2026-06-15 : les tests de complétude
// (TestCanonicalOrderCompleteness, order_audit_test.go) prouvent que chaque step
// est PRÉSENT dans canonicalOrder, mais sont AVEUGLES à la DIRECTION — un step
// dépendant (ApplySchema nil, qui lit/écrit une table créée par un AUTRE step)
// peut être placé AVANT son créateur. Sur une DB fraîche, le dépendant tape alors
// une table absente ; l'erreur backfill étant non-fatale (registry.go), ça
// converge sur 2 boots — latent, jamais un crash, mais incorrect.
//
// Ce test attrape toute FUTURE inversion de ce type lors des relocations Phase 1.5.

// stepDependencies déclare les couples (dépendant → créateur requis) où le
// dépendant ne crée PAS sa table. canonicalOrder DOIT placer le créateur AVANT.
// Étendre dès qu'une nouvelle step ApplySchema-nil est relocalisée.
var stepDependencies = map[string]string{
	// (à remplir au fil des relocations Phase 1.5)
}

// knownPreExistingInversions : inversions DÉJÀ présentes dans canonicalOrder à la
// découverte du garde-fou (2026-06-15), tolérées car PROD-SAFE — sur les DB déjà
// migrées c'est un no-op (schema_migrations name-keyed, les 2 steps sont done) ;
// sur une DB fraîche c'est une convergence sur 2 boots via le backfill non-fatal.
// Les CORRIGER = réordonner canonicalOrder, ce qui rippe
// TestSortByCanonicalIsNoOpOnCurrentRegistry (canonicalOrder == ordre de
// registration) → décision/sign-off dédiée. NE PAS ajouter de NOUVELLE inversion
// ici : toute nouvelle dépendance va dans stepDependencies (→ échec CI).
var knownPreExistingInversions = map[string]string{
	// shared_seed_tier_boundaries_v2 (title-owned, ApplySchema nil) INSERT dans
	// lusr_hyperparams_v2, table créée par shared_create_skill_v2_tables (global,
	// canon pos +1). Cf. .ai/thought_log.md (escalade reorder).
	"shared_seed_tier_boundaries_v2": "shared_create_skill_v2_tables",
}

func TestCanonicalOrder_DependencyDirection(t *testing.T) {
	last := len(canonicalOrder)

	// (1) Toute dépendance déclarée DOIT respecter créateur-avant-dépendant.
	for dependent, creator := range stepDependencies {
		cr, dr := canonicalRank(creator), canonicalRank(dependent)
		if cr >= last {
			t.Errorf("créateur %q absent de canonicalOrder", creator)
		}
		if dr >= last {
			t.Errorf("dépendant %q absent de canonicalOrder", dependent)
		}
		if cr >= dr {
			t.Errorf("INVERSION de dépendance : créateur %q (pos %d) doit précéder le dépendant %q (pos %d)",
				creator, cr, dependent, dr)
		}
	}

	// (2) Les inversions pré-existantes connues sont documentées, pas masquées :
	//     si l'une est CORRIGÉE (par le reorder sous sign-off), le test le signale
	//     pour qu'on la retire de la liste (garde le filet honnête).
	for dependent, creator := range knownPreExistingInversions {
		cr, dr := canonicalRank(creator), canonicalRank(dependent)
		if cr < dr {
			t.Errorf("l'inversion connue %q→%q est désormais CORRIGÉE (créateur pos %d < dépendant pos %d) — la retirer de knownPreExistingInversions",
				creator, dependent, cr, dr)
		} else {
			t.Logf("inversion pré-existante TOLÉRÉE (prod-safe ; fix = reorder sous sign-off) : dépendant %q (pos %d) avant son créateur %q (pos %d)",
				dependent, dr, creator, cr)
		}
	}
}
