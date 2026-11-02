// system_locale.go — fuseau horaire et langue d'interface tirés du système au premier lancement
// (capacité AdoptSystemLocale, vraie sous le profil desktop). Dans un fichier dédié : config.go
// est gelé en taille (config_size_ratchet_test.go).
package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/platform/syslocale"
)

// SystemLocale : le fuseau et la langue du système, tels qu'ils servent de défaut aux clés
// `user_timezone` et `lang` absentes d'app_settings.json. Valeur zéro hors profil desktop
// (rien n'est lu) ; Lang vide quand le système ne désigne aucune langue de l'application.
type SystemLocale struct {
	// Timezone : nom IANA chargeable (jamais vide sous la capacité : repli sur defaultUserTimezone).
	Timezone string
	// Lang : code de langue de config/locales.toml, "" si aucune langue d'interface du système n'en fait partie.
	Lang string
}

// DeclaredString rend la valeur d'une clé chaîne d'app_settings.json quand elle est ÉCRITE. Règle
// UNIQUE de la configuration, du magasin de réglages et de /bootstrap : une clé absente, vide ou non
// textuelle n'est pas écrite, et c'est alors le défaut (celui du système sous la capacité
// AdoptSystemLocale) qui sert.
func DeclaredString(v any) (string, bool) {
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// LangOr rend la langue du système, ou fallback quand le système n'en désigne aucune de l'application
// (ou hors profil desktop : valeur zéro). Défaut d'une clé `lang` non écrite, pour le magasin de
// réglages comme pour /bootstrap.
func (sys SystemLocale) LangOr(fallback string) string {
	if sys.Lang != "" {
		return sys.Lang
	}
	return fallback
}

// TimezoneOr rend le fuseau du système, ou fallback hors profil desktop (valeur zéro). Défaut d'une
// clé `user_timezone` non écrite, pour le magasin de réglages comme pour /bootstrap.
func (sys SystemLocale) TimezoneOr(fallback string) string {
	if sys.Timezone != "" {
		return sys.Timezone
	}
	return fallback
}

// readSystemLocale lit le système ; substituée par les tests.
var readSystemLocale = syslocale.ReadRaw

// adoptSystemLocale pose le fuseau et la langue du système sous la capacité AdoptSystemLocale.
// Le fuseau devient le fuseau de l'utilisateur quand app_settings.json n'en porte pas ; une valeur
// déjà écrite n'est jamais remplacée. Hors capacité, rien n'est lu.
func adoptSystemLocale(cfg *AppConfig) {
	if !cfg.Capabilities.AdoptSystemLocale {
		return
	}
	sys := resolveSystemLocale(readSystemLocale())
	cfg.SystemLocale = sys
	if _, declared := readUserTimezone(cfg.AppSettingsPath); !declared {
		cfg.UserTimezone = sys.Timezone
	}
}

// resolveSystemLocale interprète les valeurs brutes du système : fuseau IANA validé (chargeable
// par time.LoadLocation, sinon repli journalisé sur defaultUserTimezone) et première langue
// d'interface reconnue par i18n.Lookup.
func resolveSystemLocale(raw syslocale.Raw) SystemLocale {
	if raw.ReadErr != nil {
		slog.Warn("config: lecture du fuseau ou de la langue du système incomplète", "err", raw.ReadErr)
	}
	sys := SystemLocale{Timezone: systemTimezone(raw), Lang: systemLang(raw)}
	slog.Info("config: fuseau et langue du système adoptés par défaut",
		"windows_zone", raw.WindowsZone, "timezone", sys.Timezone, "ui_languages", raw.UILanguages,
		"lang", sys.Lang)
	return sys
}

// systemTimezone rend le nom IANA du fuseau du système, ou defaultUserTimezone (avec un
// avertissement) quand le système n'en désigne aucun que la table sache traduire ou que la base
// de fuseaux ne le charge pas.
func systemTimezone(raw syslocale.Raw) string {
	iana, ok := raw.Timezone()
	if !ok {
		slog.Warn("config: fuseau du système non reconnu, repli sur le fuseau par défaut",
			"windows_zone", raw.WindowsZone, "iana_zone", raw.IANAZone, "repli", defaultUserTimezone)
		return defaultUserTimezone
	}
	if _, err := time.LoadLocation(iana); err != nil {
		slog.Warn("config: fuseau du système absent de la base de fuseaux, repli sur le fuseau par défaut",
			"timezone", iana, "err", err, "repli", defaultUserTimezone)
		return defaultUserTimezone
	}
	return iana
}

// systemLang rend la première langue d'interface du système que config/locales.toml connaît ;
// "" (avec un avertissement) si aucune n'est connue : le défaut d'app_settings.json s'applique.
func systemLang(raw syslocale.Raw) string {
	for _, tag := range raw.UILanguages {
		if l, ok := i18n.Lookup(tag); ok {
			return string(l)
		}
	}
	if len(raw.UILanguages) > 0 {
		slog.Warn("config: aucune langue d'interface du système parmi celles de l'application",
			"ui_languages", raw.UILanguages)
	}
	return ""
}

// readUserTimezone lit user_timezone dans app_settings.json. declared dit si le fichier porte un
// fuseau non vide ; sinon tz vaut defaultUserTimezone (fichier absent, illisible ou muet).
func readUserTimezone(settingsPath string) (tz string, declared bool) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return defaultUserTimezone, false
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return defaultUserTimezone, false
	}
	if s, ok := DeclaredString(m["user_timezone"]); ok {
		return s, true
	}
	return defaultUserTimezone, false
}
