package mappings

import (
	"reflect"
	"testing"
)

func TestNormalizeLang(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"fr-FR": "fr",
		"de-DE": "de",
		"en":    "en",
		"EN":    "en",
		"":      "",
		"PT-BR": "pt",
	}
	for in, want := range cases {
		if got := NormalizeLang(in); got != want {
			t.Errorf("NormalizeLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRankEntry_FullLabel(t *testing.T) {
	t.Parallel()
	entry := RankEntry{
		ID:       1,
		Title:    map[string]string{"en": "Bronze", "fr": "Bronze"},
		Subtitle: map[string]string{"en": "I"},
		Tier:     map[string]string{"en": "BRONZE"},
	}

	// Locale présente partout.
	label, fallback := entry.FullLabel("en")
	if label != "Bronze I BRONZE" || fallback {
		t.Errorf("EN = (%q, %v)", label, fallback)
	}

	// Locale FR seulement pour Title → fallback EN sur Subtitle/Tier.
	label, fallback = entry.FullLabel("fr")
	if label != "Bronze I BRONZE" || !fallback {
		t.Errorf("FR partial = (%q, %v) — want fallback=true", label, fallback)
	}

	// Locale inconnue → tout EN.
	label, fallback = entry.FullLabel("de")
	if label != "Bronze I BRONZE" || !fallback {
		t.Errorf("DE inconnue = (%q, %v) — want fallback=true", label, fallback)
	}

	// Entrée vide.
	empty := RankEntry{ID: 0}
	label, fallback = empty.FullLabel("en")
	if label != "" {
		t.Errorf("empty = %q, want vide", label)
	}
	_ = fallback
}

func TestRankCatalog_Lifecycle(t *testing.T) {
	t.Parallel()
	entries := []RankEntry{
		{ID: 1, Title: map[string]string{"en": "Bronze", "fr": "Bronze"}, Subtitle: map[string]string{"en": "I"}, Tier: map[string]string{"en": "BRONZE"}},
		{ID: 2, Title: map[string]string{"en": "Silver", "fr": "Argent"}, Subtitle: map[string]string{"en": "I"}, Tier: map[string]string{"en": "SILVER"}},
		{ID: 5, Title: map[string]string{"en": "Diamond", "fr": "Diamant"}, Subtitle: map[string]string{"en": "III"}, Tier: map[string]string{"en": "DIAMOND"}},
	}
	c := NewRankCatalog("halo_infinite", entries)

	if c.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q", c.TitleSlug())
	}
	if c.Len() != 3 {
		t.Errorf("Len = %d, want 3", c.Len())
	}

	if e, ok := c.Get(1); !ok || e.ID != 1 {
		t.Errorf("Get(1) = (%+v, %v)", e, ok)
	}
	if _, ok := c.Get(99); ok {
		t.Errorf("Get(99) devrait être absent")
	}

	// Next(1) = Bronze → Silver (ID=2 dans le catalog).
	if e, ok := c.Next(1); !ok || e.ID != 2 {
		t.Errorf("Next(1) = (%+v, %v), want ID=2", e, ok)
	}
	// Next(2) = pas d'ID=3 dans le catalog → false.
	if _, ok := c.Next(2); ok {
		t.Errorf("Next(2) sans ID=3 devrait être absent")
	}

	// FullLabel ID connu.
	if label, ok := c.FullLabel(1, "en"); !ok || label != "Bronze I BRONZE" {
		t.Errorf("FullLabel(1, en) = (%q, %v)", label, ok)
	}
	// FullLabel ID inconnu.
	if _, ok := c.FullLabel(99, "en"); ok {
		t.Errorf("FullLabel(99) devrait être absent")
	}

	// IDs trié.
	ids := c.IDs()
	if !reflect.DeepEqual(ids, []int{1, 2, 5}) {
		t.Errorf("IDs = %v, want [1,2,5]", ids)
	}
}

func TestRankCatalog_EmptyConstructor(t *testing.T) {
	t.Parallel()
	c := NewRankCatalog("title_x", nil)
	if c.Len() != 0 {
		t.Errorf("Len = %d, want 0", c.Len())
	}
	if len(c.IDs()) != 0 {
		t.Errorf("IDs() devrait être vide")
	}
}
