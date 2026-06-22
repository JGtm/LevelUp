package mapper

import (
	"fmt"
	"time"

	"levelup/go-api/internal/platform/halo/duration"
)

// Ce fichier ne contient plus qu'une fine couche d'adaptation : le parsing des
// durées ISO-8601 « flavoured Halo » est centralisé dans le leaf neutre
// internal/platform/halo/duration (source unique, regex canonique gérant aussi
// les jours "P1D" — corrige un bug latent de divergence inter-packages).
//
// On conserve les noms ParseISO8601Duration / DurationSeconds /
// DurationSecondsFloat exportés ici pour ne pas casser les appelants, et on
// ré-enveloppe l'erreur du leaf avec ErrInvalidDuration (errors.go) afin de
// préserver les vérifications errors.Is(err, ErrInvalidDuration) existantes.

// ParseISO8601Duration parse une durée ISO 8601 flavoured Halo en time.Duration.
// Entrée vide → (0, nil). Erreur enveloppée dans ErrInvalidDuration si invalide.
func ParseISO8601Duration(s string) (time.Duration, error) {
	d, err := duration.ParseISO8601Duration(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidDuration, err)
	}
	return d, nil
}

// DurationSeconds renvoie la durée en secondes entières (troncature), adapté aux
// colonnes INTEGER du schéma v6. Erreur enveloppée dans ErrInvalidDuration.
func DurationSeconds(s string) (int, error) {
	n, err := duration.Seconds(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidDuration, err)
	}
	return n, nil
}

// DurationSecondsFloat renvoie la durée en secondes fractionnaires, adapté aux
// colonnes FLOAT (avg_life_seconds). Erreur enveloppée dans ErrInvalidDuration.
func DurationSecondsFloat(s string) (float64, error) {
	f, err := duration.SecondsFloat(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidDuration, err)
	}
	return f, nil
}
