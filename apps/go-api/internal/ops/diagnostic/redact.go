package diagnostic

import (
	"regexp"
	"strings"
)

// redacted remplace une valeur retirée.
const redacted = "[retiré]"

// forbiddenKey : un nom de clé qui ne sort jamais (comparaison insensible à la casse, sur le nom entier).
var forbiddenKey = regexp.MustCompile(`(?i)(token|secret|password|passwd|webhook|cookie|credential|authorization|api[_-]?key|private[_-]?key|refresh|bearer|cipher|session[_-]?key|signature|^(?:(?:auth|authorization|oauth|device|user)[_-]?)?code$)`)

// ForbiddenKey dit si un nom de clé désigne une valeur qui ne doit pas figurer dans le rapport.
func ForbiddenKey(key string) bool { return forbiddenKey.MatchString(key) }

// RedactValue retire d'un document décodé (map, tableau, scalaire) toute valeur portée par une clé interdite,
// récursivement, et passe les chaînes restantes par [Scrub]. secrets : valeurs exactes à effacer partout.
func RedactValue(v any, secrets []string) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if ForbiddenKey(k) {
				out[k] = redacted
				continue
			}
			out[k] = RedactValue(val, secrets)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = RedactValue(val, secrets)
		}
		return out
	case string:
		return Scrub(t, secrets)
	}
	return v
}

var (
	// bearerPattern : un en-tête d'autorisation.
	bearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{8,}`)
	// jwtPattern : un jeton à trois segments (JWT : l'en-tête commence par « eyJ »).
	jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{4,}`)
	// msTokenPattern : un jeton de rafraîchissement Microsoft (« M.C5… », « 0.A… ») ou un long blob opaque.
	msTokenPattern = regexp.MustCompile(`\b(?:M\.[A-Z]\d+_[A-Za-z0-9._!*$-]{20,}|0\.A[A-Za-z0-9._-]{40,})`)
	// pairPattern : « clé=valeur », « clé: valeur » ou « "clé":"valeur" » dont la clé est sensible. Un code d'autorisation
	// (clé « code », « auth_code », « device_code »…) est un secret ; un « status_code » ou un « error_code » n'en est pas un.
	pairPattern = regexp.MustCompile(`(?i)("?\b(?:[A-Za-z0-9_.-]*(?:token|secret|password|passwd|webhook|cookie|credential|authorization|api[_-]?key|refresh)[A-Za-z0-9_.-]*|(?:(?:auth|authorization|oauth|device|user)[_-]?)?code)"?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&]+)`)
	// urlSecretPattern : un paramètre d'adresse sensible (?secret=…, &code=…, &state=…).
	urlSecretPattern = regexp.MustCompile(`(?i)([?&](?:secret|code|state|token|access_token|refresh_token|id_token)=)[^&\s"']+`)
)

// Scrub efface d'un texte de journal ce qui ressemble à un secret : les valeurs exactes de secrets, les
// en-têtes d'autorisation, les jetons de forme connue, les paires clé=valeur sensibles et les paramètres
// d'adresse sensibles. Une ligne n'est jamais supprimée : seule la valeur est remplacée, le journal reste lisible.
func Scrub(s string, secrets []string) string {
	for _, sec := range secrets {
		if len(sec) >= 6 {
			s = strings.ReplaceAll(s, sec, redacted)
		}
	}
	s = bearerPattern.ReplaceAllString(s, "Bearer "+redacted)
	s = jwtPattern.ReplaceAllString(s, redacted)
	s = msTokenPattern.ReplaceAllString(s, redacted)
	s = urlSecretPattern.ReplaceAllString(s, "${1}"+redacted)
	s = pairPattern.ReplaceAllString(s, "${1}"+redacted)
	return s
}
