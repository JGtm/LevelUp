// Package domain — sync.go : types pour le moteur de synchronisation (Sprint 18).
//
// Portage de src/data/sync/models_sync.py (Python).
// SyncOptions et SyncResult sont les seuls types domain publics du moteur sync.
// L'implémentation est dans internal/sync/.
package domain

import "time"

// SyncOptions paramètre un cycle de synchronisation.
// Portage de SyncOptions (Python models_sync.py).
type SyncOptions struct {
	// MatchType filtre les matchs récupérés ("all", "matchmaking", "custom", "local").
	MatchType string
	// MaxMatches limite le nombre de matchs récupérés (défaut 200, max 2000).
	MaxMatches int
	// WithParticipants active l'extraction de tous les participants du match (roster complet).
	WithParticipants bool
	// WithMedals active l'extraction des médailles de tous les joueurs.
	WithMedals bool
	// RequestsPerSecond contrôle le rate limiting vers l'API Halo.
	RequestsPerSecond int
}

// DefaultSyncOptions retourne les options par défaut (portage de SyncOptions() Python).
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        200,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 10,
	}
}

// SyncResult agrège les compteurs et erreurs d'un cycle de synchronisation.
// Portage de SyncResult (Python models_sync.py).
type SyncResult struct {
	MatchesInserted  int
	MatchesSkipped   int
	MedalsInserted   int
	ParticipantsDone int
	InsertedMatchIDs []string
	Errors           []string
	Warnings         []string
	StartedAt        time.Time
	FinishedAt       time.Time
	DurationSeconds  float64
}

// Status retourne "success", "partial_success" ou "failure".
func (r *SyncResult) Status() string {
	if len(r.Errors) == 0 {
		return "success"
	}
	if r.MatchesInserted > 0 {
		return "partial_success"
	}
	return "failure"
}

// AddError ajoute une erreur au résultat.
func (r *SyncResult) AddError(msg string) {
	r.Errors = append(r.Errors, msg)
}

// AddWarning ajoute un avertissement au résultat.
func (r *SyncResult) AddWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}
