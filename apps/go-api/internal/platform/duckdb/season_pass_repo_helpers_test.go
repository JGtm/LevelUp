// Package duckdb — season_pass_repo_helpers_test.go : résolution localisée des
// libellés Battle Pass (localizedText). Régression du bug « Rewards des battlepass
// pas traduits en ENG » : la locale de requête doit ordonner les clés de langue.
package duckdb

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
)

func TestLocalizedText_LocaleOrdering(t *testing.T) {
	multi := map[string]any{"fr": "Français", "en": "English", "default": "Default"}

	// FR (preferEN=false) : ordre fr,en,default — inchangé vs historique.
	if got := localizedText(multi, false); got != "Français" {
		t.Errorf("preferEN=false: got %q, want %q", got, "Français")
	}
	// EN (preferEN=true) : ordre en,default,fr.
	if got := localizedText(multi, true); got != "English" {
		t.Errorf("preferEN=true: got %q, want %q", got, "English")
	}

	// La clé "value" (déjà localisée côté serveur Halo) prime dans les deux sens.
	resolved := map[string]any{"value": "Resolved", "fr": "Français", "en": "English"}
	if got := localizedText(resolved, true); got != "Resolved" {
		t.Errorf("value prime (EN): got %q, want %q", got, "Resolved")
	}
	if got := localizedText(resolved, false); got != "Resolved" {
		t.Errorf("value prime (FR): got %q, want %q", got, "Resolved")
	}

	// EN demandé mais seul FR présent → fallback (default absent, fr dernier recours).
	frOnly := map[string]any{"fr": "SeulementFR"}
	if got := localizedText(frOnly, true); got != "SeulementFR" {
		t.Errorf("EN préféré, seul FR dispo: got %q, want %q", got, "SeulementFR")
	}

	// FR demandé mais seul EN présent → fallback en.
	enOnly := map[string]any{"en": "OnlyEN"}
	if got := localizedText(enOnly, false); got != "OnlyEN" {
		t.Errorf("FR préféré, seul EN dispo: got %q, want %q", got, "OnlyEN")
	}

	// String brute : trim, indépendant de la locale.
	if got := localizedText("  plain  ", true); got != "plain" {
		t.Errorf("string brute: got %q, want %q", got, "plain")
	}
}

// TestBPItemFieldCoalesce_ValueIsEnglishSource verrouille le fix « récompenses Battle
// Pass restent en FR en UI EN » : la chaîne anglaise des items vit dans `.value`
// (canonique en-US), PAS dans `.translations.en-US` (absent des payloads Waypoint).
// value doit donc être classée source ANGLAISE — précéder `.translations.fr-FR` en
// préférence EN, et rester après en préférence FR.
func TestBPItemFieldCoalesce_ValueIsEnglishSource(t *testing.T) {
	enSQL := bpItemFieldCoalesce(ctxkeys.WithLocale(context.Background(), "en"), "Title", "title")
	posVal := strings.Index(enSQL, "Title.value")
	posFR := strings.Index(enSQL, "Title.translations.fr-FR")
	if posVal < 0 || posFR < 0 {
		t.Fatalf("EN: expressions value/fr-FR absentes de %q", enSQL)
	}
	if posVal > posFR {
		t.Errorf("EN: value (anglais) doit précéder fr-FR dans le COALESCE, got %q", enSQL)
	}

	frSQL := bpItemFieldCoalesce(ctxkeys.WithLocale(context.Background(), "fr"), "Title", "title")
	posValFR := strings.Index(frSQL, "Title.value")
	posFRfr := strings.Index(frSQL, "Title.translations.fr-FR")
	if posValFR < 0 || posFRfr < 0 {
		t.Fatalf("FR: expressions value/fr-FR absentes de %q", frSQL)
	}
	if posFRfr > posValFR {
		t.Errorf("FR: fr-FR doit précéder value dans le COALESCE, got %q", frSQL)
	}
}
