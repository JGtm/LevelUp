package canonical

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

func TestAssetReferenceText(t *testing.T) {
	cases := []struct {
		name string
		ref  *AssetReference
		want i18n.Text
	}{
		{"nil", nil, nil},
		{"defaut seul: range sous la langue de repli", &AssetReference{DefaultLabel: "Bazaar"}, i18n.Text{i18n.Fallback: "Bazaar"}},
		{"deux langues", &AssetReference{DefaultLabel: "Bazaar", Labels: map[string]string{"fr": "Bazar", "en": "The Bazaar"}},
			i18n.Text{i18n.LocaleFR: "Bazar", i18n.LocaleEN: "The Bazaar"}},
		{"le libelle de la langue de repli prime sur le defaut", &AssetReference{DefaultLabel: "Bazaar (defaut)", Labels: map[string]string{"en": "Bazaar"}},
			i18n.Text{i18n.LocaleEN: "Bazaar"}},
		{"valeur vide et cle hors liste ignorees", &AssetReference{DefaultLabel: "Bazaar", Labels: map[string]string{"fr": "", "zz": "Zzz", "EN-us": "x"}},
			i18n.Text{i18n.Fallback: "Bazaar"}},
		{"rien", &AssetReference{}, i18n.Text{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ref.Text()
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
