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
	// WithObjectiveStats active l'extraction des stats objectifs par joueur (CTF/Zones/Oddball)
	// depuis GetMatchStats vers shared.match_objective_stats. Activé par défaut (données déjà
	// présentes dans le payload participants, aucun appel réseau supplémentaire).
	WithObjectiveStats bool
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
		WithObjectiveStats:  true,
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

// PostSyncStepTiming chronomètre une étape du pipeline post-sync (dashboard
// monitoring P4 — timeline + détection des goulots). Items = volume traité
// par l'étape (0 si non significatif).
type PostSyncStepTiming struct {
	Step       string `json:"step"`
	DurationMs int64  `json:"duration_ms"`
	Items      int    `json:"items"`
}

// PostSyncResult agrège les compteurs du pipeline post-sync.
// Tags JSON snake_case : la struct est exposée telle quelle par le dashboard
// monitoring admin (champ PostSync de scheduler.PlayerOutcomeDetail).
type PostSyncResult struct {
	PerfScoresComputed       int   `json:"perf_scores_computed"`
	LUSRUpdated              int   `json:"lusr_updated"`
	CareerSynced             bool  `json:"career_synced"`
	ViewsRefreshed           int   `json:"views_refreshed"`
	AchievementsSynced       bool  `json:"achievements_synced"`
	MatchesPromotedFriends   int64 `json:"matches_promoted_friends"`   // §7 hook auto-recompute is_with_friends post-sync
	EngagementScoresComputed int   `json:"engagement_scores_computed"` // Phase 3 plan engagement
	EngagementCoefsUpdated   int   `json:"engagement_coefs_updated"`   // Phase recompute coefs : nb modes recomputes (0..2)
	SessionsAssigned         int   `json:"sessions_assigned"`          // recalcul session_id post-sync (auto)
	// WeaponKillsProcessed / WeaponKillsNoFilm RETIRÉS le 2026-09-01 : l'étape 1.55
	// qui les alimentait est supprimée avec son producteur (lot arme-source-unique).
	// Le compte de films traités et de films absents est désormais celui de l'étape
	// 1.57, publié en expvar (`killsource_matchs_collectes`, `killsource_films_absents`).
	CitationsComputed      int `json:"citations_computed"`       // matchs traités par le pipeline post-sync (étape 1.6 citations)
	DominanceFlagsComputed int `json:"dominance_flags_computed"` // matchs traités par le pipeline post-sync (étape 1.7 dominance_flag)
	ConvergedEvents        int `json:"converged_events"`         // matchs rattrapés par la convergence events (étape 1.54)
	ConvergedPSA           int `json:"converged_psa"`            // matchs rattrapés par la convergence PSA (étape 1.56)
	SnapshotReadyMarked    int `json:"snapshot_ready_marked"`    // matchs marqués snapshot_ready_at (étape 6 readiness, Phase 2)

	// Chronométrage du pipeline (dashboard monitoring P4) : durée totale +
	// durée par étape (timeline + détection des goulots). Renseigné par
	// runPostSyncPipeline via postSyncClock ; vide sur le path léger
	// CSR+achievements (DurationMs seul).
	DurationMs  int64                `json:"duration_ms"`
	StepTimings []PostSyncStepTiming `json:"step_timings,omitempty"`

	// FatalErrors collecte les erreurs FATAL DuckDB (IsInvalidatedError)
	// rencontrées dans le post-sync — chaque entrée = "<step>: <err>".
	// Propagée vers SyncResult.Errors par l'appelant (engine.go) pour que
	// Status() renvoie "partial_success" au lieu de mentir avec "success"
	// alors qu'une étape critique a invalidé une DB.
	// Cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md Phase 5 "Status sync honnête".
	FatalErrors []string `json:"fatal_errors,omitempty"`
}
