// Package ops — media_filename.go : parsing des datetimes depuis les noms de
// fichiers de capture (OBS/Xbox/ShadowPlay). Extrait de media.go (refactor god-file).
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

// xboxFilenameRe matche le pattern Xbox / NVIDIA ShadowPlay :
// "Halo Infinite 2024.11.15 - 21.30.45.01.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var xboxFilenameRe = regexp.MustCompile(
	`(\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})`)

// obsFilenameRe matche le pattern OBS Studio par défaut (%CCYY-%MM-%DD %hh-%mm-%ss) :
// "Replay 2026-04-19 17-10-54.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var obsFilenameRe = regexp.MustCompile(
	`(\d{4})-(\d{2})-(\d{2}) (\d{2})-(\d{2})-(\d{2})`)

// halo5FilenameRe matche le pattern Windows Game Bar des captures Halo 5 :
// "Halo_5_Guardians-2019-12-12_22h49.mp4" (séparateur "_", "h" entre heure et
// minute, PAS de secondes). Groupe 1=année 2=mois 3=jour 4=heure 5=min — la
// seconde est absente (parseCaptureTimeFromFilename la met à 0 quand m[6] manque).
var halo5FilenameRe = regexp.MustCompile(
	`(\d{4})-(\d{2})-(\d{2})_(\d{2})h(\d{2})`)

// captureTimeRegexes liste les patterns de noms de fichiers reconnus,
// du plus spécifique au plus générique. L'ordre importe peu en pratique
// (les patterns ne se chevauchent pas) mais OBS arrive en premier car
// c'est le format le plus fréquent dans nos captures.
var captureTimeRegexes = []*regexp.Regexp{obsFilenameRe, xboxFilenameRe, halo5FilenameRe}

// parseCaptureTimeFromFilename tente d'extraire la datetime depuis le nom de fichier.
// Formats supportés : OBS Studio, Xbox / NVIDIA ShadowPlay.
// Retourne nil si aucun pattern connu n'est trouvé ou si loc est nil.
// La datetime est interprétée comme heure locale (loc), puis convertie en UTC.
func parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time {
	if loc == nil {
		return nil
	}
	for _, re := range captureTimeRegexes {
		m := re.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		year := mustAtoi(m[1])
		month := mustAtoi(m[2])
		day := mustAtoi(m[3])
		hour := mustAtoi(m[4])
		min := mustAtoi(m[5])
		// La seconde est optionnelle : les patterns à 5 groupes (Halo 5
		// "..._22h49") n'ont pas de m[6] → seconde = 0.
		sec := 0
		if len(m) > 6 {
			sec = mustAtoi(m[6])
		}
		if year == 0 {
			continue
		}
		t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc).UTC()
		return &t
	}
	return nil
}

// mustAtoi convertit une string en int, retourne 0 en cas d'erreur.
func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
