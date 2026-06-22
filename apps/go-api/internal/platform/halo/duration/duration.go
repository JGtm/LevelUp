// Package duration — source UNIQUE du parsing des durées ISO-8601 « flavoured Halo ».
//
// Pourquoi ce leaf neutre : trois implémentations divergentes coexistaient
// (internal/games/halo_5, internal/platform/halo, internal/openspartan/mapper),
// avec des REGEX différentes (bug latent, pas qu'une violation DRY) :
//
//   - halo_5            : `^P(?:(\d+)D)?T(?:…)?$`            → `T` OBLIGATOIRE → "P1D" REJETÉ.
//   - platform/halo     : `^P(?:(\d+)D)?(?:T(?:…)?)?$`        → `T` optionnel    → "P1D" = 86400.
//   - openspartan/mapper: `^PT(?:…)?$`                        → pas de `D` du tout → "P1D" REJETÉ.
//
// Conséquence : une durée "P1D" (un jour, sans composante de temps) parsait d'un
// côté et échouait des deux autres → résultat incohérent selon le chemin de code.
//
// Ce package vit volontairement comme LEAF stdlib-only (zéro import interne) afin
// d'être importable sans cycle par les trois consommateurs, dont les mappers PURS
// halo_5 / openspartan (qui ne doivent surtout pas dépendre du package
// platform/halo, lourd en I/O DuckDB).
//
// Grammaire canonique acceptée : P[nD][T[nH][nM][n[.frac]S]]
//   - le marqueur T est OPTIONNEL (corrige le bug "P1D") ;
//   - jours / heures / minutes / secondes sont tous optionnels ;
//   - les fractions de secondes sont préservées (ex. "PT16.48S", "AvgLifeTime") ;
//   - AU MOINS une composante est requise : "P", "PT", "" sont invalides ;
//   - les valeurs négatives sont impossibles (la classe \d+ ne les capture pas) ;
//   - semaines/mois/années (P1Y, P1W…) NE sont PAS supportés (jamais émis par
//     l'API Halo pour un temps de jeu/durée de match).
package duration

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidDuration est renvoyée par ParseISO8601Duration quand l'entrée ne
// correspond pas à la grammaire ou ne porte aucune composante de temps.
var ErrInvalidDuration = errors.New("duration: invalid ISO 8601 duration")

// MaxPlausibleSeconds borne une durée de match/temps-de-jeu plausible (24 h).
// Au-delà = donnée corrompue / overflow regex. Utilisé par les variantes bornées
// (h5) pour distinguer « indisponible » d'une valeur absurde.
const MaxPlausibleSeconds = 24 * 3600

// iso8601DurationRe est la regex CANONIQUE : P[nD][T[nH][nM][n[.frac]S]].
// Le groupe T(...) est entièrement optionnel → "P1D" matche (corrige le bug).
var iso8601DurationRe = regexp.MustCompile(
	`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

// ParseISO8601Duration convertit une durée ISO-8601 flavoured Halo en
// time.Duration. Entrée vide → (0, nil) : de nombreux payloads omettent les
// champs de durée optionnels (dégradation gracieuse côté appelant).
//
// Renvoie ErrInvalidDuration si la chaîne ne matche pas la grammaire OU ne porte
// aucune composante de temps (ex. "P", "PT").
func ParseISO8601Duration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	m := iso8601DurationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, s)
	}
	if m[1] == "" && m[2] == "" && m[3] == "" && m[4] == "" {
		return 0, fmt.Errorf("%w: %q n'a aucune composante de temps", ErrInvalidDuration, s)
	}
	var total time.Duration
	if m[1] != "" { // jours
		d, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, fmt.Errorf("%w: jours %q: %v", ErrInvalidDuration, m[1], err)
		}
		total += time.Duration(d) * 24 * time.Hour
	}
	if m[2] != "" { // heures
		h, err := strconv.Atoi(m[2])
		if err != nil {
			return 0, fmt.Errorf("%w: heures %q: %v", ErrInvalidDuration, m[2], err)
		}
		total += time.Duration(h) * time.Hour
	}
	if m[3] != "" { // minutes
		mi, err := strconv.Atoi(m[3])
		if err != nil {
			return 0, fmt.Errorf("%w: minutes %q: %v", ErrInvalidDuration, m[3], err)
		}
		total += time.Duration(mi) * time.Minute
	}
	if m[4] != "" { // secondes (potentiellement fractionnaires)
		sec, err := strconv.ParseFloat(m[4], 64)
		if err != nil {
			return 0, fmt.Errorf("%w: secondes %q: %v", ErrInvalidDuration, m[4], err)
		}
		total += time.Duration(sec * float64(time.Second))
	}
	return total, nil
}

// Seconds renvoie la durée en secondes ENTIÈRES TRONQUÉES (vers le bas), adapté
// aux colonnes INTEGER du schéma v6 (duration_seconds, time_played_seconds, …).
// Propage l'erreur de ParseISO8601Duration (openspartan/mapper attend ce contrat).
func Seconds(s string) (int, error) {
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0, err
	}
	return int(d / time.Second), nil
}

// SecondsFloat renvoie la durée en secondes FRACTIONNAIRES (précision préservée),
// adapté aux colonnes FLOAT (avg_life_seconds). Propage l'erreur de parsing.
func SecondsFloat(s string) (float64, error) {
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0, err
	}
	return float64(d) / float64(time.Second), nil
}

// SecondsInt64 renvoie la durée en secondes ENTIÈRES TRONQUÉES, avec dégradation
// gracieuse à 0 (au lieu d'une erreur) pour entrée vide ou non parsable. Forme
// historiquement consommée par le chemin Compare (service record Waypoint).
func SecondsInt64(s string) int64 {
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0
	}
	return int64(d / time.Second)
}

// SecondsRoundedBoundedPtr renvoie la durée en secondes ENTIÈRES ARRONDIES
// (au plus proche), bornée à [0, MaxPlausibleSeconds]. nil si vide, non parsable,
// sans composante, ou hors borne — le canonique distingue « indisponible » de 0.
// Forme historiquement consommée par les mappers Halo 5 (durée de match).
func SecondsRoundedBoundedPtr(s string) *int {
	if strings.TrimSpace(s) == "" {
		return nil // vide = « indisponible » (≠ 0), contrat h5
	}
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return nil
	}
	secs := d.Seconds()
	if secs < 0 || secs > MaxPlausibleSeconds {
		return nil
	}
	out := int(math.Round(secs))
	return &out
}

// MillisBounded renvoie la durée en MILLISECONDES (précision fractionnaire
// préservée), bornée à [0, MaxPlausibleSeconds]. ok=false si vide, non parsable,
// sans composante, ou hors borne. Forme consommée par les events Halo 5
// (TimeSinceStart) et la durée de vie moyenne fractionnaire.
func MillisBounded(s string) (int, bool) {
	if strings.TrimSpace(s) == "" {
		return 0, false // vide = « indisponible », contrat h5 events
	}
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0, false
	}
	secs := d.Seconds()
	if secs < 0 || secs > MaxPlausibleSeconds {
		return 0, false
	}
	return int(math.Round(secs * 1000)), true
}

// SecondsFloatBoundedPtr renvoie la durée en secondes FRACTIONNAIRES, bornée à
// [0, MaxPlausibleSeconds]. nil si invalide ou hors borne. Forme consommée par
// AvgLifeTimeOfPlayer Halo 5 ("PT16.48S" → 16.48). Conserve exactement la même
// arithmétique que l'ancien chemin (millis bornés ÷ 1000).
func SecondsFloatBoundedPtr(s string) *float64 {
	ms, ok := MillisBounded(s)
	if !ok {
		return nil
	}
	f := float64(ms) / 1000.0
	return &f
}
