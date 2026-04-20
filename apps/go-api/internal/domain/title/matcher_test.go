package title

import "testing"

func TestRegistry_MatchByXboxTitleID(t *testing.T) {
	r := NewRegistry()

	got := r.MatchByXboxTitleID("1144039928")
	if got == nil {
		t.Fatal("expected match for Halo Infinite Xbox Title ID")
	}
	if got.Slug != DefaultSlug {
		t.Errorf("slug = %q, want %q", got.Slug, DefaultSlug)
	}
}

func TestRegistry_MatchByXboxTitleID_Unknown(t *testing.T) {
	r := NewRegistry()
	if r.MatchByXboxTitleID("9999999") != nil {
		t.Error("expected nil for unknown title ID")
	}
}

func TestRegistry_MatchBySteamAppID(t *testing.T) {
	r := NewRegistry()

	got := r.MatchBySteamAppID("1336960")
	if got == nil {
		t.Fatal("expected match for Halo Infinite Steam App ID")
	}
	if got.Slug != DefaultSlug {
		t.Errorf("slug = %q, want %q", got.Slug, DefaultSlug)
	}
}

func TestRegistry_MatchBySteamAppID_Unknown(t *testing.T) {
	r := NewRegistry()
	if r.MatchBySteamAppID("730") != nil {
		t.Error("expected nil for unknown Steam App ID (CS2)")
	}
}

func TestRegistry_MatchPresence(t *testing.T) {
	r := NewRegistry()

	got := r.MatchPresence("1144039928")
	if got == nil || got.Slug != DefaultSlug {
		t.Errorf("MatchPresence = %v", got)
	}
	if r.MatchPresence("0000000") != nil {
		t.Error("expected nil for unknown")
	}
}

func TestRegistry_MatchByXboxTitleID_MultiTitle(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{
		Slug:        "halo_mcc",
		Name:        "Halo MCC",
		Provider:    "halo_mcc",
		Status:      StatusActive,
		XboxTitleID: "1144234394",
	})

	hi := r.MatchByXboxTitleID("1144039928")
	mcc := r.MatchByXboxTitleID("1144234394")

	if hi == nil || hi.Slug != "halo_infinite" {
		t.Errorf("Halo Infinite not matched")
	}
	if mcc == nil || mcc.Slug != "halo_mcc" {
		t.Errorf("Halo MCC not matched")
	}
}
