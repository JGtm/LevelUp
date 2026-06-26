package mappings

import (
	"reflect"
	"testing"
)

// TestNewCapabilityMappingSet_NilByKey : un byKey nil est normalisé en map vide
// (jamais un nil map déréférencé) — la lecture reste sûre.
func TestNewCapabilityMappingSet_NilByKey(t *testing.T) {
	t.Parallel()
	set := NewCapabilityMappingSet("title_x", 3, nil)
	if set.TitleSlug() != "title_x" || set.SchemaVersion() != 3 {
		t.Errorf("meta = (%q, %d), want (title_x, 3)", set.TitleSlug(), set.SchemaVersion())
	}
	// byKey nil → All() retourne une map vide (non nil), Keys() une slice vide.
	if got := set.All(); got == nil || len(got) != 0 {
		t.Errorf("All() sur byKey nil = %v, want map vide non-nil", got)
	}
	if got := set.Keys(); len(got) != 0 {
		t.Errorf("Keys() sur byKey nil = %v, want vide", got)
	}
	if _, ok := set.Status("anything"); ok {
		t.Errorf("Status sur set vide devrait être ok=false")
	}
}

// TestCapabilityMappingSet_All_DefensiveCopy : All() retourne une COPIE — muter la
// map retournée ne corrompt pas l'état interne du set (invariant d'immuabilité).
func TestCapabilityMappingSet_All_DefensiveCopy(t *testing.T) {
	t.Parallel()
	set := NewCapabilityMappingSet("t", 1, map[string]string{
		"match.history":        CapStatusSupported,
		"analytics.timeseries": CapStatusNotExposed,
	})

	got := set.All()
	want := map[string]string{
		"match.history":        CapStatusSupported,
		"analytics.timeseries": CapStatusNotExposed,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}

	// Mutation de la copie → l'état interne reste intact (copie défensive).
	got["match.history"] = "tampered"
	got["nouvelle.cle"] = "injectée"
	if st, _ := set.Status("match.history"); st != CapStatusSupported {
		t.Errorf("mutation de la copie a fui dans l'état interne: Status = %q", st)
	}
	if _, ok := set.Status("nouvelle.cle"); ok {
		t.Errorf("ajout dans la copie a fui dans l'état interne")
	}
	if len(set.All()) != 2 {
		t.Errorf("All() après mutation copie = %d entrées, want 2", len(set.All()))
	}
}

// TestCapabilityMappingSet_NilReceiver : toutes les lectures sur un *set nil sont
// sûres (pas de panique) — défense du pattern (un titre sans capabilities → set nil).
func TestCapabilityMappingSet_NilReceiver(t *testing.T) {
	t.Parallel()
	var set *CapabilityMappingSet
	if got, ok := set.Status("k"); ok || got != "" {
		t.Errorf("Status(nil receiver) = (%q, %v), want (\"\", false)", got, ok)
	}
	if got := set.All(); got != nil {
		t.Errorf("All(nil receiver) = %v, want nil", got)
	}
	if got := set.Keys(); got != nil {
		t.Errorf("Keys(nil receiver) = %v, want nil", got)
	}
}

// TestCapabilityMappingSet_Status_HitAndMiss : Status distingue une clé déclarée
// (hit) d'une clé absente (miss) — comportement, pas tautologie.
func TestCapabilityMappingSet_Status_HitAndMiss(t *testing.T) {
	t.Parallel()
	set := NewCapabilityMappingSet("t", 1, map[string]string{"match.history": CapStatusDegraded})
	if st, ok := set.Status("match.history"); !ok || st != CapStatusDegraded {
		t.Errorf("Status(hit) = (%q, %v), want (degraded, true)", st, ok)
	}
	if st, ok := set.Status("absente"); ok || st != "" {
		t.Errorf("Status(miss) = (%q, %v), want (\"\", false)", st, ok)
	}
}

// TestCapabilityMappingSet_Keys_Sorted : Keys() retourne les clés TRIÉES
// (déterminisme tests/diff), indépendamment de l'ordre d'insertion.
func TestCapabilityMappingSet_Keys_Sorted(t *testing.T) {
	t.Parallel()
	set := NewCapabilityMappingSet("t", 1, map[string]string{
		"zeta.cap":  CapStatusSupported,
		"alpha.cap": CapStatusSupported,
		"mid.cap":   CapStatusDegraded,
	})
	got := set.Keys()
	want := []string{"alpha.cap", "mid.cap", "zeta.cap"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() = %v, want %v (trié)", got, want)
	}
}
