package analysis

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

// Le retrait du préfixe de catégorie et les overrides visent le nom de la langue par défaut ; pour une
// autre langue le nom est rendu tel quel (rogné).
func TestPlaylistLabelConfig_DisplayIn(t *testing.T) {
	cfg := PlaylistLabelConfig{StripCategory: true}
	if got := cfg.DisplayIn(i18n.Default, "Arène delta : Héritage"); got != "Delta : Héritage" {
		t.Errorf("langue par défaut : %q", got)
	}
	h5 := PlaylistLabelConfig{Overrides: map[string]string{"Super Fiesta Fête": "Super Fiesta"}}
	if got := h5.DisplayIn(i18n.Default, "Super Fiesta Fête"); got != "Super Fiesta" {
		t.Errorf("langue par défaut, override : %q", got)
	}
	if got := h5.DisplayIn(i18n.Fallback, "Super Fiesta Fête"); got != "Super Fiesta Fête" {
		t.Errorf("autre langue, override non appliqué : %q", got)
	}
	if got := cfg.DisplayIn(i18n.Fallback, " Arena Delta: Legacy "); got != "Arena Delta: Legacy" {
		t.Errorf("autre langue : %q, attendu le nom tel quel, rogné", got)
	}
	if got := cfg.DisplayIn(i18n.Fallback, ""); got != "" {
		t.Errorf("nom vide : %q", got)
	}
}

// Le mode affiché suit la langue ; ResolveModeUIWithVariant (clé) lit la langue par défaut.
func TestResolveModeUIWithVariantIn_SuitLaLangue(t *testing.T) {
	pair := i18n.Text{i18n.Default: "Assassin", i18n.Fallback: "Arena:Slayer"}
	if got := ResolveModeUIWithVariantIn(i18n.Fallback, pair, nil); got == nil || *got != "Slayer" {
		t.Errorf("en : %v, attendu Slayer", got)
	}
	if got := ResolveModeUIWithVariantIn(i18n.Default, pair, nil); got == nil || *got != "Assassin" {
		t.Errorf("fr : %v, attendu Assassin", got)
	}
	if got := ResolveModeUIWithVariant(pair, nil); got == nil || *got != "Assassin" {
		t.Errorf("clé : %v, attendu Assassin", got)
	}
	// Titre sans pair : le nom de la variante prend le relais, dans la langue demandée.
	variant := i18n.Text{i18n.Default: "Assassin", i18n.Fallback: "Slayer"}
	if got := ResolveModeUIWithVariantIn(i18n.Fallback, nil, variant); got == nil || *got != "Slayer" {
		t.Errorf("variante en : %v", got)
	}
}
