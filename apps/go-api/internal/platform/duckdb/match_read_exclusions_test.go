package duckdb

import (
	"strings"
	"testing"
)

// TestExcludedVariantClause : la clause de masquage read-side est produite UNIQUEMENT
// pour les titres ayant des game_variant exclus (Halo 5 = Campagne), avec le bon alias
// et le bon nombre d'args ; vide (no-op) pour les autres titres (ex. Infinite).
func TestExcludedVariantClause(t *testing.T) {
	// Titre sans mode masqué → no-op (aucune clause, aucun arg).
	if clause, args := ExcludedVariantClause("halo_infinite", "r"); clause != "" || args != nil {
		t.Errorf("halo_infinite: attendu ('', nil), obtenu (%q, %v)", clause, args)
	}
	if clause, args := ExcludedVariantClause("", "r"); clause != "" || args != nil {
		t.Errorf("titre vide: attendu ('', nil), obtenu (%q, %v)", clause, args)
	}

	// Halo 5 → clause NOT IN + 2 args (les GUID Campagne) + alias injecté.
	clause, args := ExcludedVariantClause("halo_5", "r")
	if !strings.Contains(clause, "r.game_variant_id") {
		t.Errorf("alias non injecté dans la clause: %q", clause)
	}
	if !strings.Contains(clause, "NOT IN (?,?)") {
		t.Errorf("clause NOT IN attendue avec 2 placeholders: %q", clause)
	}
	if len(args) != 2 {
		t.Fatalf("args = %d, want 2 (GUID Campagne)", len(args))
	}
	// Les args correspondent au set déclaré, dans l'ordre.
	want := readExcludedGameVariantIDs["halo_5"]
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %v, want %v", i, a, want[i])
		}
	}

	// Alias paramétrable (autre table).
	if c2, _ := ExcludedVariantClause("halo_5", "reg"); !strings.Contains(c2, "reg.game_variant_id") {
		t.Errorf("alias 'reg' non injecté: %q", c2)
	}
}
