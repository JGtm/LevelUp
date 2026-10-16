// Package handlers — session_context_validate.go : validation de forme du corps de
// POST /session/context.
//
// Cette validation borne ce qu'une session anonyme peut écrire sur disque ; elle ne
// décide PAS de l'accès au joueur désigné (autorisation, traitée ailleurs).
package handlers

import (
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

const (
	// maxPlayerSlugRunes borne le slug de joueur accepté en session. Un gamertag Xbox
	// tient en 15 caractères (12 + suffixe pour les gamertags modernes) ; la marge
	// couvre les clés de profil historiques.
	maxPlayerSlugRunes = 64

	// slugExtraRunes : séparateurs admis en plus des lettres, chiffres et marques
	// Unicode (espace, soulignement, tiret, point, dièse du suffixe de gamertag).
	slugExtraRunes = " _-.#"
)

// sessionLocales est la liste fermée des langues acceptées en session ; elle est
// celle du front (KNOWN_LOCALES de lib/i18n/locale.ts).
var sessionLocales = map[string]bool{"fr": true, "en": true}

// validateSessionContextRequest vérifie la forme des champs renseignés de la requête.
// Elle retourne nil si tout est acceptable, sinon une erreur 400 au contrat d'API.
func validateSessionContextRequest(req *domain.SessionContextRequest) error {
	if req.Locale != nil && !sessionLocales[*req.Locale] {
		return humacore.NewError(http.StatusBadRequest, "invalid_locale", "locale must be one of: fr, en")
	}
	if req.PlayerSlug != nil && !isValidPlayerSlugShape(*req.PlayerSlug) {
		return humacore.NewError(http.StatusBadRequest, "invalid_player_slug", "player_slug has an invalid format")
	}
	return nil
}

// isValidPlayerSlugShape : 1 à maxPlayerSlugRunes caractères de l'alphabet de slug,
// sans espace en tête ou en queue et sans slug fait uniquement de points (« . »,
// « .. » désigneraient un répertoire une fois le slug utilisé comme segment de
// chemin).
func isValidPlayerSlugShape(slug string) bool {
	n := utf8.RuneCountInString(slug)
	if n == 0 || n > maxPlayerSlugRunes || slug != strings.TrimSpace(slug) {
		return false
	}
	if strings.Trim(slug, ".") == "" {
		return false
	}
	for _, r := range slug {
		if r == utf8.RuneError || !isSlugRune(r) {
			return false
		}
	}
	return true
}

func isSlugRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) ||
		strings.ContainsRune(slugExtraRunes, r)
}
