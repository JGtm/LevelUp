package title

import "testing"

// TestXboxTitleIDFor_RegistryDriven (PMT-6) : XboxTitleIDFor lit l'XboxTitleID du
// descripteur (source unique) — valeur Halo inchangée (parité), titre synthétique
// distinct via la méthode, slug inconnu → "".
func TestXboxTitleIDFor_RegistryDriven(t *testing.T) {
	// Parité : le helper package-level rend toujours la valeur Halo historique.
	if got := XboxTitleIDFor(DefaultSlug); got != "2043073184" {
		t.Errorf("XboxTitleIDFor(%q) = %q, want 2043073184 (parité Halo)", DefaultSlug, got)
	}
	if got := XboxTitleIDFor("no_such_title"); got != "" {
		t.Errorf("XboxTitleIDFor(inconnu) = %q, want \"\"", got)
	}

	// Méthode registry-driven : un titre synthétique au descripteur distinct rend
	// SON XboxTitleID, pas celui d'Halo (preuve que la vérité vient du descripteur).
	r := NewRegistry()
	r.Register(&TitleDescriptor{
		Slug:        "synthetic_test_title",
		Name:        "Synthetic",
		Status:      StatusActive,
		XboxTitleID: "9999999999",
	})
	if got := r.XboxTitleIDFor("synthetic_test_title"); got != "9999999999" {
		t.Errorf("r.XboxTitleIDFor(synthetic) = %q, want 9999999999", got)
	}
	if got := r.XboxTitleIDFor(DefaultSlug); got != "2043073184" {
		t.Errorf("r.XboxTitleIDFor(%q) = %q, want 2043073184", DefaultSlug, got)
	}
	if got := r.XboxTitleIDFor("unknown"); got != "" {
		t.Errorf("r.XboxTitleIDFor(unknown) = %q, want \"\"", got)
	}
}
