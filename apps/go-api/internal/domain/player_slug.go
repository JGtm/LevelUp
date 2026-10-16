// Package domain — player_slug.go : forme d'un slug de profil joueur.
package domain

import "unicode"

// MaxPlayerSlugLength borne la longueur (en octets) d'un slug de profil accepté d'une
// requête. Un gamertag Xbox compte au plus 15 caractères (moderne : 12 plus un
// suffixe), soit moins de 64 octets en UTF-8.
const MaxPlayerSlugLength = 64

// IsValidPlayerSlug dit si slug a la forme d'un gamertag, clé d'un profil de
// db_profiles.json : 1 à MaxPlayerSlugLength octets de lettres, chiffres (Unicode),
// marques diacritiques, espaces, '_' ou '-', sans espace en tête ni en fin. Le slug
// devient un segment de chemin disque (dossier du joueur) : '.', '/', '\', ':' et tout
// autre signe sont refusés, ce qui exclut '.', '..' et toute traversée.
func IsValidPlayerSlug(slug string) bool {
	if slug == "" || len(slug) > MaxPlayerSlugLength {
		return false
	}
	if slug[0] == ' ' || slug[len(slug)-1] == ' ' {
		return false
	}
	for _, r := range slug {
		if !isPlayerSlugRune(r) {
			return false
		}
	}
	return true
}

func isPlayerSlugRune(r rune) bool {
	switch r {
	case ' ', '_', '-':
		return true
	case unicode.ReplacementChar:
		return false
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}
