// Package ops — media_filename.go : validation du fuseau des noms de capture et petits outils
// de nom de fichier. Les motifs d'heure de capture et leur sens (début ou fin) vivent dans la
// table unique des sources, media_capture_source.go.
package ops

import (
	"regexp"
	"strconv"
)

func SanitizeMediaTimezone(tz string) string {
	for _, c := range tz {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '/' || c == '_' || c == '-' || c == '+':
		default:
			return ""
		}
	}
	return tz
}

// thumbHashSuffixRe matche le suffixe hash ajouté aux miniatures GIF.
// Ex: "Halo Infinite 2025-12-18 17-40-46_9430c6551833" → base "Halo Infinite 2025-12-18 17-40-46"
var thumbHashSuffixRe = regexp.MustCompile(`_[0-9a-fA-F]{6,}$`)

// mustAtoi convertit une string en int, retourne 0 en cas d'erreur.
func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
