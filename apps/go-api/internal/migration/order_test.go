package migration

// order_test.go — garde-fous de l'ordre explicite des migrations (Phase 1.5.0).
// Verrouille canonicalOrder (order.go) : complétude, pas de stale/doublon, et
// surtout PREUVE que rendre l'ordre explicite NE CHANGE PAS l'ordre actuel.

import (
	"strings"
	"testing"
)

// TestCanonicalOrderCompleteness : toute migration enregistrée est dans
// canonicalOrder, et canonicalOrder ne contient ni doublon ni entrée morte.
// → ajouter une migration sans la lister ici fait échouer le boot-order audit.
func TestCanonicalOrderCompleteness(t *testing.T) {
	registered := make(map[string]bool)
	for _, m := range All() {
		// Les migrations "test_*" sont enregistrées dynamiquement par les tests
		// d'intégration (helpers_extra_test.go) — elles ne font pas partie du
		// registre de production et ne doivent pas figurer dans canonicalOrder.
		if strings.HasPrefix(m.Name, "test_") {
			continue
		}
		registered[m.Name] = true
		if _, ok := canonicalIndex[m.Name]; !ok {
			t.Errorf("migration %q absente de canonicalOrder (order.go) — l'ajouter à la bonne position", m.Name)
		}
	}
	// Pas de doublon dans canonicalOrder.
	// NB : le check inverse « canonicalOrder ⊆ steps enregistrés » (entrée morte)
	// est désormais dans halo_infinite/migrations/order_audit_test.go, car les
	// steps title-owned sont dans canonicalOrder mais PAS dans le registre global
	// All() (Phase 1.5.1 B). Ce test ne voit que le registre global.
	seen := make(map[string]bool, len(canonicalOrder))
	for _, n := range canonicalOrder {
		if seen[n] {
			t.Errorf("doublon dans canonicalOrder: %q", n)
		}
		seen[n] = true
	}
}

// TestSortByCanonicalIsNoOpOnCurrentRegistry : la bascule vers l'ordre explicite
// reproduit EXACTEMENT l'ordre d'enregistrement courant. C'est la garantie que
// Phase 1.5.0 ne réordonne aucune migration (donc ne casse aucun boot).
func TestSortByCanonicalIsNoOpOnCurrentRegistry(t *testing.T) {
	cur := All()
	before := make([]string, len(cur))
	for i, m := range cur {
		before[i] = m.Name
	}
	sorted := make([]Migration, len(cur))
	copy(sorted, cur)
	sortByCanonicalOrder(sorted)
	for i := range sorted {
		if sorted[i].Name != before[i] {
			t.Fatalf("sortByCanonicalOrder change l'ordre au rang %d: %q -> %q (devrait être un no-op)",
				i, before[i], sorted[i].Name)
		}
	}
}
