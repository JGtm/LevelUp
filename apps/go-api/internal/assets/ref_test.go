package assets

import (
	"testing"
)

func TestRef_String_AllFields(t *testing.T) {
	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42", Variant: "spritesheet", Lang: "fr-FR"}
	got := ref.String()
	want := "medal-image/halo_infinite/42/spritesheet/fr-FR"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_String_NoVariantNoLang(t *testing.T) {
	ref := Ref{Kind: KindChallengeBadge, TitleID: "halo_infinite", ID: "weekly-heroic"}
	got := ref.String()
	want := "challenge-badge/halo_infinite/weekly-heroic"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_String_VariantOnly(t *testing.T) {
	ref := Ref{Kind: KindBPTrackImage, TitleID: "hi", ID: "tracks/s6.png", Variant: "tracks"}
	got := ref.String()
	want := "bp-track-image/hi/tracks/s6.png/tracks"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRef_CacheKey_EqualsString(t *testing.T) {
	ref := Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: "LiveFire"}
	if ref.CacheKey() != ref.String() {
		t.Error("CacheKey() doit être égal à String()")
	}
}

func TestRef_LogAttrs_MinimalFields(t *testing.T) {
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "LiveFire"}
	attrs := ref.LogAttrs()
	// Doit contenir kind, title, id — soit 6 éléments (3 paires)
	if len(attrs) != 6 {
		t.Errorf("longueur inattendue: got %d, want 6 — %v", len(attrs), attrs)
	}
}

func TestRef_LogAttrs_WithVariantAndLang(t *testing.T) {
	ref := Ref{Kind: KindAssetTranslation, TitleID: "hi", ID: "asset-1", Variant: "v", Lang: "fr-FR"}
	attrs := ref.LogAttrs()
	// kind, title, id, variant, lang → 10 éléments
	if len(attrs) != 10 {
		t.Errorf("longueur inattendue: got %d, want 10 — %v", len(attrs), attrs)
	}
}
