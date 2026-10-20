package config

// config_auth.go — modes d'authentification et d'inscription : lecture normalisée,
// refus au démarrage d'une valeur inconnue (registre sécurité C5-5). Les comparaisons
// du code (`authMode == "xbox"`, authz.Enforced…) sont exactes : une valeur mal
// orthographiée ou mal cassée éteignait la propriété des joueurs sans un mot.

import (
	"fmt"
	"slices"
	"strings"
)

// Modes reconnus ; les défauts valent quand la variable est absente ou vide.
const (
	defaultAuthMode         = "none"
	defaultRegistrationMode = "invite"
	authModePassword        = "password"
	authModeXbox            = "xbox"
)

var (
	// knownAuthModes : valeurs admises de LEVELUP_AUTH_MODE (défaut defaultAuthMode).
	knownAuthModes = []string{defaultAuthMode, authModePassword, authModeXbox}
	// knownRegistrationModes : valeurs admises de LEVELUP_REGISTRATION (défaut
	// defaultRegistrationMode).
	knownRegistrationModes = []string{defaultRegistrationMode, "open", "closed"}
)

// normalizeMode lit une valeur de mode : minuscules, espaces de bord retirés ; vide
// (ou blanc seul) donne def.
func normalizeMode(raw, def string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return def
	}
	return v
}

// validateModes refuse un mode d'authentification ou d'inscription inconnu. Appelée
// par Validate dans TOUS les environnements (démo et développement compris). Un champ
// vide (configuration construite à la main) vaut le défaut, comme à la lecture.
func (c *AppConfig) validateModes() error {
	if !slices.Contains(knownAuthModes, normalizeMode(c.AuthMode, defaultAuthMode)) {
		return fmt.Errorf("LEVELUP_AUTH_MODE=%q inconnu (valeurs admises : %s)", c.AuthMode, strings.Join(knownAuthModes, ", "))
	}
	if !slices.Contains(knownRegistrationModes, normalizeMode(c.RegistrationMode, defaultRegistrationMode)) {
		return fmt.Errorf("LEVELUP_REGISTRATION=%q inconnu (valeurs admises : %s)",
			c.RegistrationMode, strings.Join(knownRegistrationModes, ", "))
	}
	return nil
}
