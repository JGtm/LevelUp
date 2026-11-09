package service

import (
	"testing"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
)

// TestMaxRankNameInEveryLanguage : une langue sans libellé propre prend celui de la langue de repli,
// sinon celui de la langue par défaut ; aucune langue connue n'est laissée vide.
func TestMaxRankNameInEveryLanguage(t *testing.T) {
	onlyDefault := maxRankNameInEveryLanguage(i18n.Text{i18n.Default: "SR 152"})
	for _, l := range i18n.Locales() {
		if onlyDefault[l] != "SR 152" {
			t.Errorf("source à la seule langue par défaut : %s = %q", l, onlyDefault[l])
		}
	}
	withFallback := maxRankNameInEveryLanguage(i18n.Text{i18n.Default: "Héros", i18n.Fallback: "Hero"})
	if withFallback[i18n.Default] != "Héros" || withFallback[i18n.Fallback] != "Hero" {
		t.Errorf("langues propres conservées : %v", withFallback)
	}
	for _, l := range i18n.Locales() {
		if withFallback[l] == "" {
			t.Errorf("langue %s vide : %v", l, withFallback)
		}
	}
}

// TestRankLabelTexts : le libellé d'un rang est rendu dans chaque langue connue, numéro de sous-rang
// en chiffres romains, sans langue codée.
func TestRankLabelTexts(t *testing.T) {
	e := mappings.RankEntry{
		ID:    1,
		Title: map[string]string{string(i18n.Fallback): "Platinum", string(i18n.Default): "Platine"},
		Tier:  mappings.SameInEveryLocale("2"),
	}
	got := rankLabelTexts(e)
	if got[i18n.Default] != "Platine II" || got[i18n.Fallback] != "Platinum II" {
		t.Errorf("libellés = %v", got)
	}
	if empty := rankLabelTexts(mappings.RankEntry{ID: 2}); len(empty) != 0 {
		t.Errorf("entrée vide : %v", empty)
	}
}
