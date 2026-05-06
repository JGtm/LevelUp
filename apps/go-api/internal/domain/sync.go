// Package domain — sync.go : types pour le moteur de synchronisation (Sprint 18).
//
// Portage de src/data/sync/models_sync.py (Python).
// SyncOptions et SyncResult sont les seuls types domain publics du moteur sync.
// L'implémentation est dans internal/sync/.
package domain

import (
	"fmt"
	"time"
)

// validSyncMatchTypes est l'ensemble des types de match valides pour la sync.
var validSyncMatchTypes = map[string]bool{
	"all":         true,
	"matchmaking": true,
	"custom":      true,
	"local":       true,
}

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
	// WithHighlightEvents active le parsing du chunk highlight events (kills, deaths, medals, mode).
	// Alimente highlight_events et killer_victim_pairs dans shared_matches_v2.
	// Activé par défaut (identique au comportement Python).
	WithHighlightEvents bool
	// RequestsPerSecond contrôle le rate limiting vers l'API Halo.
	RequestsPerSecond int
}

// Validate vérifie la cohérence des options de synchronisation.
// Appelé dans engine.go avant tout appel réseau (fail-fast B8).
func (o SyncOptions) Validate() error {
	if o.MatchType == "" {
		return fmt.Errorf("SyncOptions: MatchType vide (attendu : all|matchmaking|custom|local)")
	}
	if !validSyncMatchTypes[o.MatchType] {
		return fmt.Errorf("SyncOptions: MatchType invalide %q (attendu : all|matchmaking|custom|local)", o.MatchType)
	}
	if o.MaxMatches < 0 {
		return fmt.Errorf("SyncOptions: MaxMatches doit être ≥ 0 (reçu %d)", o.MaxMatches)
	}
	if o.RequestsPerSecond < 0 {
		return fmt.Errorf("SyncOptions: RequestsPerSecond doit être ≥ 0 (reçu %d)", o.RequestsPerSecond)
	}
	return nil
}

// DefaultSyncOptions retourne les options par défaut (portage de SyncOptions() Python).
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		MatchType:           "matchmaking",
		MaxMatches:          200,
		WithParticipants:    true,
		WithMedals:          true,
		WithHighlightEvents: true,
		RequestsPerSecond:   10,
	}
}

// SyncResult agrège les compteurs et erreurs d'un cycle de synchronisation.
// Portage de SyncResult (Python models_sync.py).
type SyncResult struct {
	MatchesInserted  int
	MatchesSkipped   int
	MedalsInserted   int
	ParticipantsDone int
	EventsInserted   int
	InsertedMatchIDs []string
	Errors           []string
	Warnings         []string
	StartedAt        time.Time
	FinishedAt       time.Time
	DurationSeconds  float64
	PostSync         *PostSyncResult
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

// PostSyncResult agrège les compteurs du pipeline post-sync.
type PostSyncResult struct {
	PerfScoresComputed       int
	LUSRUpdated              int
	CareerSynced             bool
	ViewsRefreshed           int
	AchievementsSynced       bool
	MatchesPromotedFriends   int64 // §7 hook auto-recompute is_with_friends post-sync
	EngagementScoresComputed int   // Phase 3 plan engagement
	EngagementCoefsUpdated   int   // Phase recompute coefs : nb modes recomputes (0..2)
	SessionsAssigned         int   // recalcul session_id post-sync (auto)
	WeaponKillsProcessed     int   // nouveaux matchs traités par le pipeline film/weapon kills
	WeaponKillsNoFilm        int   // matchs sans film (404/410, normal pour vieux matchs)
}
