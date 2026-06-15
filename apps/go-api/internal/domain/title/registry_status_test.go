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

// TestTitleDescriptor_IsActive — prédicat domaine pur (PMT-8), par statut.
func TestTitleDescriptor_IsActive(t *testing.T) {
	cases := []struct {
		status Status
		want   bool
	}{
		{StatusActive, true},
		{StatusComingSoon, false},
		{StatusArchived, false},
		{Status(""), false},
	}
	for _, tc := range cases {
		td := &TitleDescriptor{Slug: "x", Status: tc.status}
		if got := td.IsActive(); got != tc.want {
			t.Errorf("IsActive() pour status %q : attendu %v, got %v", tc.status, tc.want, got)
		}
	}
}

// TestRegistry_NonArchived — le switcher garde active + coming_soon, exclut archived.
func TestRegistry_NonArchived(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{Slug: "soon", Status: StatusComingSoon})
	r.Register(&TitleDescriptor{Slug: "old", Status: StatusArchived})

	var hasDefault, hasSoon, hasOld bool
	for _, td := range r.NonArchived() {
		switch td.Slug {
		case DefaultSlug:
			hasDefault = true
		case "soon":
			hasSoon = true
		case "old":
			hasOld = true
		}
	}
	if !hasDefault || !hasSoon {
		t.Errorf("NonArchived() doit inclure active (%v) + coming_soon (%v)", hasDefault, hasSoon)
	}
	if hasOld {
		t.Error("NonArchived() ne doit pas inclure archived")
	}
}
