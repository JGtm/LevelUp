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

func TestRankCatalog_MaxRank(t *testing.T) {
	t.Parallel()

	// Catalog non trié : MaxRank retourne l'entrée du rank_id le plus élevé.
	cat := NewRankCatalog("halo_infinite", []RankEntry{
		{ID: 100, Title: map[string]string{"en": "Colonel", "fr": "Colonel"}},
		{ID: 272, Title: map[string]string{"en": "Hero", "fr": "Héros"}},
		{ID: 1, Title: map[string]string{"en": "Recruit", "fr": "Recrue"}},
	})
	e, ok := cat.MaxRank()
	if !ok || e.ID != 272 {
		t.Fatalf("MaxRank = (%+v, %v), want ID 272", e, ok)
	}
	if fr, _ := e.FullLabel("fr"); fr != "Héros" {
		t.Errorf("FullLabel(fr) = %q, want Héros", fr)
	}

	// Catalog vide / nil → (zero, false).
	if _, ok := NewRankCatalog("x", nil).MaxRank(); ok {
		t.Error("MaxRank sur catalog vide devrait retourner ok=false")
	}
	var nilCat *RankCatalog
	if _, ok := nilCat.MaxRank(); ok {
		t.Error("MaxRank sur catalog nil devrait retourner ok=false")
	}
}

const expectedFullLabel = "Bronze I BRONZE"

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
	if label != expectedFullLabel || fallback {
		t.Errorf("EN = (%q, %v)", label, fallback)
	}

	// Locale FR seulement pour Title → fallback EN sur Subtitle/Tier.
	label, fallback = entry.FullLabel("fr")
	if label != expectedFullLabel || !fallback {
		t.Errorf("FR partial = (%q, %v) — want fallback=true", label, fallback)
	}

	// Locale inconnue → tout EN.
	label, fallback = entry.FullLabel("de")
	if label != expectedFullLabel || !fallback {
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
	if label, ok := c.FullLabel(1, "en"); !ok || label != expectedFullLabel {
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

// Un *RankCatalog nil doit se comporter comme un catalog vide : le chargement
// est best-effort (metadata absente en mode démo) et buildTitleRuntime logge
// ranks_count via Len() sans re-vérifier le pointeur (panic boot E2E 2026-07-10).
func TestRankCatalog_NilReceiverBehavesAsEmpty(t *testing.T) {
	t.Parallel()
	var c *RankCatalog

	if got := c.Len(); got != 0 {
		t.Errorf("Len() sur nil = %d, want 0", got)
	}
	if got := c.TitleSlug(); got != "" {
		t.Errorf("TitleSlug() sur nil = %q, want \"\"", got)
	}
	if _, ok := c.Get(1); ok {
		t.Error("Get(1) sur nil : ok = true, want false")
	}
	if _, ok := c.Next(1); ok {
		t.Error("Next(1) sur nil : ok = true, want false")
	}
	if got := c.CumulativeXPRequired(10); got != 0 {
		t.Errorf("CumulativeXPRequired(10) sur nil = %d, want 0", got)
	}
	if _, ok := c.FullLabel(1, "fr"); ok {
		t.Error("FullLabel(1) sur nil : ok = true, want false")
	}
	if got := c.IDs(); got != nil {
		t.Errorf("IDs() sur nil = %v, want nil", got)
	}
}
