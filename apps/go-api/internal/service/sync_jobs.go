// Package service — sync_jobs.go : les jobs de synchronisation lancés depuis l'API (port.SyncJobService).
//
// Ce fichier porte le type, ses dépendances et leur câblage. Les opérations sont réparties par sujet :
// sync_jobs_initial.go (sync initiale), sync_jobs_manual.go (sync delta, cooldown, moteur du pool),
// sync_jobs_run.go (exécution d'un job et ses notifications), sync_jobs_engine.go (construction du
// moteur), sync_jobs_backfill.go (rattrapage), sync_jobs_recalc.go (recalcul des sessions),
// sync_jobs_profile.go (recherche du profil).
package service

import (
	"context"
	"sync"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/friendstore"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
)

// syncJobTimeout borne la durée d'un job de sync lancé depuis l'API. Sans borne, le contexte du job
// attendrait indéfiniment le bail du joueur si un autre écrivain (auto-sync, watcher) le tient ;
// 30 min couvrent une sync initiale volumineuse et son post-sync.
const syncJobTimeout = 30 * time.Minute

// defaultManualSyncCooldown borne la fréquence des syncs manuels delta (/sync/all et
// /players/{slug}/sync) : le sync manuel n'est qu'un déclencheur à la demande de l'auto-sync. Une
// deuxième demande dans la fenêtre est refusée (port.SyncErrCooldown). Ne touche pas la sync
// initiale (onboarding).
const defaultManualSyncCooldown = 5 * time.Minute

// SyncJobStore est le registre des jobs que les jobs de sync créent et font évoluer (*jobs.Store).
type SyncJobStore interface {
	Create(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus
	FindActiveJob(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus
	FindActiveInitialSync(playerSlug string) *domain.AsyncJobStatus
	Update(jobID string, fn func(*domain.AsyncJobStatus)) bool
	SetStatus(jobID string, status domain.JobStatus, step *string) bool
}

// SyncProfiles lit les profils suivis (db_profiles.json) ; titleFilter vide = tous les titres
// (*config.AppConfig).
type SyncProfiles interface {
	LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error)
}

// PrestigeHook ré-évalue les défis Prestige d'un joueur (identifié par son slug, soit le user_id des
// défis) après une ingestion.
type PrestigeHook func(ctx context.Context, playerSlug, titleSlug string)

// EngineBuilder construit un moteur de sync entièrement câblé pour (gamertag, xuid) ; en production,
// celui de l'auto-sync, de sorte que le sync manuel delta partage son pool de jetons.
type EngineBuilder func(ctx context.Context, gamertag, xuid string) *go_sync.SyncEngine

// syncDeltaRunner : la surface commune du moteur de sync et du runner de suivi en direct d'un titre.
type syncDeltaRunner interface {
	RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error)
}

// SyncJobService implémente port.SyncJobService.
type SyncJobService struct {
	cfg           *config.AppConfig
	profiles      SyncProfiles
	settingsStore *settings_platform.Store
	jobs          SyncJobStore
	provider      auth_platform.TokenProvider

	friendStore   *friendstore.FriendStore // nil : aucun chargeur d'amis sur les moteurs construits ici
	notifierFor   notifications.EmitterFactory
	postSync      port.PostSyncRunner
	prestigeHook  PrestigeHook
	replayEnqueue replayartifacts.EnqueueFunc
	engineBuilder EngineBuilder

	// syncGate déduplique les syncs entre sources : un job cède quand le joueur est déjà en vol
	// (watcher ou auto-sync).
	syncGate go_sync.SyncGate
	// serverCtx est le contexte de vie du serveur : les jobs en dérivent leur contexte, de sorte que
	// l'arrêt les annule et libère leurs baux avant la fermeture des bases.
	serverCtx context.Context

	cooldown     time.Duration
	cooldownMu   sync.Mutex
	lastManualAt map[string]time.Time

	// Coutures des tests : le runner d'une sync initiale et celui d'une sync delta.
	runnerFor       func(ctx context.Context, titleSlug, gamertag, xuid string, tokens *domain.HaloTokens) (syncDeltaRunner, context.Context)
	pooledRunnerFor func(ctx context.Context, gamertag, xuid string) syncDeltaRunner
}

var _ port.SyncJobService = (*SyncJobService)(nil)

// NewSyncJobService assemble le service. Sans WithSyncGate, aucune déduplication entre sources ;
// sans WithServerContext, les jobs ne sont pas annulés à l'arrêt.
func NewSyncJobService(
	cfg *config.AppConfig,
	settingsStore *settings_platform.Store,
	jobStore SyncJobStore,
	provider auth_platform.TokenProvider,
) *SyncJobService {
	s := &SyncJobService{
		cfg:           cfg,
		profiles:      cfg,
		settingsStore: settingsStore,
		jobs:          jobStore,
		provider:      provider,
		syncGate:      go_sync.NopSyncGate{},
		serverCtx:     context.Background(),
		cooldown:      defaultManualSyncCooldown,
		lastManualAt:  make(map[string]time.Time),
	}
	s.runnerFor = s.defaultRunnerFor
	s.pooledRunnerFor = func(ctx context.Context, gamertag, xuid string) syncDeltaRunner {
		return s.newPooledEngine(ctx, gamertag, xuid)
	}
	return s
}

// engineRoot rend la racine des ressources que reçoivent les constructeurs de moteur : leur seule
// lecture de RepoRoot dans ce service.
func (s *SyncJobService) engineRoot() string { return s.cfg.RepoRoot }

// WithProfiles remplace la source des profils (par défaut la configuration).
func (s *SyncJobService) WithProfiles(p SyncProfiles) *SyncJobService {
	s.profiles = p
	return s
}

// WithFriendStore branche le magasin des amis par joueur sur les moteurs construits ici : sans lui,
// les matchs synchronisés restent is_with_friends=FALSE jusqu'au prochain recalcul. Il sert aussi au
// recalcul des sessions (chaque joueur avec SA liste).
func (s *SyncJobService) WithFriendStore(store *friendstore.FriendStore) *SyncJobService {
	s.friendStore = store
	return s
}

// WithReplayEnqueuer branche la mise en file des rejeux sur le moteur construit pour la sync initiale.
func (s *SyncJobService) WithReplayEnqueuer(enqueue replayartifacts.EnqueueFunc) *SyncJobService {
	s.replayEnqueue = enqueue
	return s
}

// WithSyncGate branche la déduplication entre sources (le coordinateur partagé du watcher) ; nil est
// ignoré.
func (s *SyncJobService) WithSyncGate(g go_sync.SyncGate) *SyncJobService {
	if g != nil {
		s.syncGate = g
	}
	return s
}

// WithServerContext branche le contexte de vie du serveur ; nil est ignoré.
func (s *SyncJobService) WithServerContext(ctx context.Context) *SyncJobService {
	if ctx != nil {
		s.serverCtx = ctx
	}
	return s
}

// WithNotificationsEmitterFactory branche la fabrique d'émetteurs de notifications (match_synced,
// sync_error), best-effort : une erreur d'émission est journalisée et ne se propage pas.
func (s *SyncJobService) WithNotificationsEmitterFactory(f notifications.EmitterFactory) *SyncJobService {
	s.notifierFor = f
	return s
}

// WithPostSyncRunner branche le runner post-sync (deltas, notifications) : capturé avant la sync,
// finalisé après son succès.
func (s *SyncJobService) WithPostSyncRunner(r port.PostSyncRunner) *SyncJobService {
	s.postSync = r
	return s
}

// WithPrestigeHook branche le hook Prestige sur le moteur de la sync initiale ; nil est sans effet.
func (s *SyncJobService) WithPrestigeHook(hook PrestigeHook) *SyncJobService {
	s.prestigeHook = hook
	return s
}

// WithEngineBuilder branche le constructeur de moteur partagé avec l'auto-sync : le sync manuel delta
// passe alors par le pool de jetons plutôt que par les jetons de la session. Nil : repli sur le moteur
// de la sync initiale, avec des jetons vides.
func (s *SyncJobService) WithEngineBuilder(b EngineBuilder) *SyncJobService {
	s.engineBuilder = b
	return s
}

// WithManualSyncCooldown surcharge la fenêtre anti-spam du sync manuel delta ; 0 la désactive.
func (s *SyncJobService) WithManualSyncCooldown(d time.Duration) *SyncJobService {
	s.cooldown = d
	return s
}
