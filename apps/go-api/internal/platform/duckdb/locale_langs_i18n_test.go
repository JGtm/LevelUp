package duckdb

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/i18n"
)

// L'ordre de préférence des étiquettes Halo reste celui d'avant l'étape 7 pour fr et en, et se dérive
// de config/locales.toml (étiquettes de la langue, puis langue de repli, puis langue par défaut).
func TestPreferredLangsForLocale_DeriveDeLaListeDesLangues(t *testing.T) {
	cases := map[string][]string{
		"fr":    {"fr-FR", "fr", "en-US", "en"},
		"en":    {"en-US", "en", "fr-FR", "fr"},
		"EN-GB": {"en-US", "en", "fr-FR", "fr"},
		"":      {"fr-FR", "fr", "en-US", "en"}, // vide : langue par défaut
		"de":    {"fr-FR", "fr", "en-US", "en"}, // inconnue : langue par défaut
	}
	for locale, want := range cases {
		if got := PreferredLangsForLocale(locale); !reflect.DeepEqual(got, want) {
			t.Errorf("PreferredLangsForLocale(%q) = %v, want %v", locale, got, want)
		}
	}
}

// La langue de lecture d'une médaille : la langue demandée si elle figure dans la liste (étiquette
// régionale ou casse comprises), sinon la langue de repli, jamais la langue par défaut.
func TestMedalReadLocale_LangueOuRepli(t *testing.T) {
	cases := map[string]i18n.Locale{"fr": i18n.LocaleFR, "en": i18n.LocaleEN, "FR-fr": i18n.LocaleFR, "": i18n.Fallback, "de": i18n.Fallback}
	for locale, want := range cases {
		if got := medalReadLocale(locale); got != want {
			t.Errorf("medalReadLocale(%q) = %q, want %q", locale, got, want)
		}
	}
}

// Une langue sans traduction officielle (ici hors liste) lit les noms de la langue de repli, puis ceux de
// la langue par défaut, puis l'identifiant : jamais un trou.
func TestPlaylistName_LangueSansTraduction(t *testing.T) {
	loc := i18n.Locale("zz")
	if got := playlistName("pl-1", loc, i18n.Text{i18n.LocaleFR: "Arène", i18n.LocaleEN: "Arena"}, "Canon"); got != "Canon" {
		t.Errorf("le nom canonique passe avant les autres langues : %q", got)
	}
	if got := playlistName("pl-1", loc, i18n.Text{i18n.LocaleFR: "Arène", i18n.LocaleEN: "Arena"}, ""); got != "Arena" {
		t.Errorf("langue de repli attendue : %q", got)
	}
	if got := playlistName("pl-1", loc, i18n.Text{i18n.LocaleFR: "Arène"}, ""); got != "Arène" {
		t.Errorf("langue par défaut attendue : %q", got)
	}
	if got := playlistName("pl-1", loc, nil, ""); got != "pl-1" {
		t.Errorf("identifiant attendu : %q", got)
	}
}
