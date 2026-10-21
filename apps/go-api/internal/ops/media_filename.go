// Package ops — media_filename.go : heure de capture lue dans un nom de fichier. Les motifs
// reconnus et leur sens (début ou fin de capture) vivent dans la table unique des sources,
// media_capture_source.go.
package ops

import (
	"regexp"
	"strconv"
	"time"
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

// parseCaptureTimeFromFilename rend l'heure H écrite dans le nom par le logiciel de capture
// (heure locale loc, convertie en UTC), nil si aucun motif de captureTimeSources ne reconnaît
// le nom ou si loc est nil. H est un début OU une fin selon la source : qui doit placer la
// capture passe par captureInstantFromName, qui rend aussi la source.
func parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time {
	if inst := captureInstantFromName(name, loc); inst != nil {
		return &inst.At
	}
	return nil
}

// mustAtoi convertit une string en int, retourne 0 en cas d'erreur.
func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
