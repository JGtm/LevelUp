// Package port — sync_jobs.go : les jobs de synchronisation lancés depuis l'API (sync initiale,
// sync delta d'un joueur ou de tous, rattrapage, recalcul des sessions).
//
// Un handler HTTP décode la requête, appelle ce port et encode la réponse ; la résolution du profil
// joueur, les garde-fous (cooldown, déduplication entre sources, jobs actifs), la construction du
// moteur et le lancement du job en arrière-plan vivent dans l'implémentation
// (`service.SyncJobService`). Le port ne mentionne ni le moteur de synchronisation ni le suivi en
// direct : un changement de moteur ne touche pas la couche HTTP.
package port

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// SyncCaller est ce que la couche HTTP sait de l'appelant et que les jobs de sync consultent : la
// session n'est pas un type du service.
type SyncCaller struct {
	// Authenticated : une session existe (sans elle, un sync manuel est refusé).
	Authenticated bool
	// HaloTokens : jetons Halo de la session ; nil = aucun. Requis par la sync initiale seulement
	// (le sync manuel delta s'authentifie par le pool de jetons).
	HaloTokens *domain.HaloTokens
}

// InitialSyncService lance la sync initiale d'un joueur.
type InitialSyncService interface {
	// CheckInitialSyncEnabled refuse (SyncErrInitialDisabled) quand l'instance interdit la sync
	// initiale. Appelé AVANT la lecture du corps de la requête : un refus prime sur un corps invalide.
	CheckInitialSyncEnabled(ctx context.Context) error
	// StartInitialSync crée le job de sync initiale et le lance en arrière-plan ; rend le job tel qu'il
	// était à sa création.
	StartInitialSync(ctx context.Context, caller SyncCaller, req domain.InitialSyncStartRequest) (*domain.AsyncJobStatus, error)
}

// ManualSyncService lance les syncs delta à la demande.
type ManualSyncService interface {
	// StartDeltaSync lance la sync delta d'un joueur (tous titres confondus pour la recherche du profil).
	StartDeltaSync(ctx context.Context, caller SyncCaller, playerSlug string) (*domain.AsyncJobStatus, error)
	// StartSyncAll lance la sync delta de tous les joueurs configurés, l'un après l'autre.
	StartSyncAll(ctx context.Context, caller SyncCaller) (*domain.AsyncJobStatus, error)
}

// BackfillService lance le rattrapage des données manquantes d'un joueur.
type BackfillService interface {
	// StartBackfill valide le joueur (suivi dans le titre demandé), crée le job de rattrapage et le
	// lance en arrière-plan.
	StartBackfill(ctx context.Context, caller SyncCaller, req domain.BackfillStartRequest) (*domain.AsyncJobStatus, error)
}

// SessionsRecalcService lance le recalcul des sessions de tous les joueurs suivis.
type SessionsRecalcService interface {
	// StartSessionsRecalc crée le job de recalcul et le lance en arrière-plan.
	StartSessionsRecalc(ctx context.Context) (*domain.AsyncJobStatus, error)
}

// SyncService regroupe les services de la sync d'un joueur ou de tous : ce que consomme le handler des
// routes /sync.
type SyncService interface {
	InitialSyncService
	ManualSyncService
}

// SyncJobService regroupe les quatre services : une seule implémentation les porte.
type SyncJobService interface {
	SyncService
	BackfillService
	SessionsRecalcService
}

// SyncJobErrorKind classe un refus d'un job de sync. La couche HTTP en tire le statut, le code et le
// message ; le service ne porte aucun texte destiné à l'écran.
type SyncJobErrorKind string

// Refus des jobs de sync.
const (
	// SyncErrSettingsLoad : la configuration de l'instance est illisible.
	SyncErrSettingsLoad SyncJobErrorKind = "settings_load"
	// SyncErrInitialDisabled : l'instance interdit la sync initiale.
	SyncErrInitialDisabled SyncJobErrorKind = "initial_disabled"
	// SyncErrInvalidMaxMatches : max_matches hors de 1..2000.
	SyncErrInvalidMaxMatches SyncJobErrorKind = "invalid_max_matches"
	// SyncErrInitialAlreadyActive : un job de sync initiale est déjà actif pour ce joueur.
	SyncErrInitialAlreadyActive SyncJobErrorKind = "initial_already_active"
	// SyncErrDeltaAlreadyActive : un job de sync est déjà actif pour ce joueur (sync delta).
	SyncErrDeltaAlreadyActive SyncJobErrorKind = "delta_already_active"
	// SyncErrAlreadyRunning : une autre source (watcher, auto-sync) synchronise déjà ce joueur.
	SyncErrAlreadyRunning SyncJobErrorKind = "already_running"
	// SyncErrBackfillAlreadyActive : un rattrapage est déjà actif pour ce joueur.
	SyncErrBackfillAlreadyActive SyncJobErrorKind = "backfill_already_active"
	// SyncErrAuthRequired : aucune session.
	SyncErrAuthRequired SyncJobErrorKind = "auth_required"
	// SyncErrHaloTokensMissing : session sans jetons Halo (sync initiale).
	SyncErrHaloTokensMissing SyncJobErrorKind = "halo_tokens_missing"
	// SyncErrCooldown : sync manuel déclenché trop tôt ; RetryAfter dit dans combien de temps réessayer.
	SyncErrCooldown SyncJobErrorKind = "cooldown"
	// SyncErrProfilesLoad : db_profiles.json est illisible.
	SyncErrProfilesLoad SyncJobErrorKind = "profiles_load"
	// SyncErrPlayersLoad : la liste des joueurs est illisible (recalcul des sessions).
	SyncErrPlayersLoad SyncJobErrorKind = "players_load"
	// SyncErrPlayerNotFound : joueur absent des profils (tous titres confondus).
	SyncErrPlayerNotFound SyncJobErrorKind = "player_not_found"
	// SyncErrPlayerNotFoundForTitle : joueur absent des profils du titre TitleSlug.
	SyncErrPlayerNotFoundForTitle SyncJobErrorKind = "player_not_found_for_title"
	// SyncErrNoPlayers : aucun joueur configuré.
	SyncErrNoPlayers SyncJobErrorKind = "no_players"
)

// SyncJobError est le refus typé d'un job de sync (`errors.As`).
type SyncJobError struct {
	Kind SyncJobErrorKind
	// PlayerSlug et TitleSlug : le joueur et le titre concernés, quand le refus les nomme.
	PlayerSlug string
	TitleSlug  string
	// RetryAfter : délai avant une nouvelle tentative (SyncErrCooldown).
	RetryAfter time.Duration
	// Err : cause interne, journalisée par le service ; jamais rendue au client.
	Err error
}

// Error décrit le refus pour le journal.
func (e *SyncJobError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("sync job: %s: %v", e.Kind, e.Err)
	}
	return "sync job: " + string(e.Kind)
}

// Unwrap rend la cause interne.
func (e *SyncJobError) Unwrap() error { return e.Err }
