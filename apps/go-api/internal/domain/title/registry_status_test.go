package title

import "testing"

func TestRegistry_IsActiveAndActive(t *testing.T) {
	r := NewRegistry()

	// halo_infinite est actif par défaut.
	if !r.IsActive(DefaultSlug) {
		t.Fatalf("%s devrait être actif", DefaultSlug)
	}
	if r.IsActive("inconnu") {
		t.Errorf("titre inconnu ne doit pas être actif")
	}

	// Un titre coming_soon existe mais n'est PAS actif.
	r.Register(&TitleDescriptor{Slug: "futur_titre", Name: "Futur", Status: StatusComingSoon})
	if r.IsActive("futur_titre") {
		t.Errorf("titre coming_soon ne doit pas être actif")
	}
	if !r.Exists("futur_titre") {
		t.Errorf("titre coming_soon doit exister dans le registre")
	}

	// Active() exclut le coming_soon et inclut halo.
	active := r.Active()
	var foundDefault, foundComingSoon bool
	for _, td := range active {
		if td.Status != StatusActive {
			t.Errorf("Active() retourne un titre non-actif: %s (%s)", td.Slug, td.Status)
		}
		switch td.Slug {
		case DefaultSlug:
			foundDefault = true
		case "futur_titre":
			foundComingSoon = true
		}
	}
	if !foundDefault {
		t.Errorf("Active() doit contenir %s", DefaultSlug)
	}
	if foundComingSoon {
		t.Errorf("Active() ne doit pas contenir le titre coming_soon")
	}
}
