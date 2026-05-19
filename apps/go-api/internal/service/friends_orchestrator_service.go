// Package service — friends_orchestrator_service.go : orchestration multi-DB
// du recompute is_with_friends.
//
// §4 du plan Squad/Sessions overhaul. Itère tous les couples (titleSlug,
// gamertag) configurés via cfg.LoadPlayers() et invoque sync.RecomputeIsWithFriends
// pour chaque player DB. Erreurs per-DB ne stoppent pas les autres ; agrégat
// {processed, failed, totalPromoted} retourné.
//
// Modes d'invocation :
//   - Bootstrap initial : RecomputeAll (tous les amis configurés, tous les joueurs).
//   - Incrémental : OnFriendsChanged (idempotent via la garde is_with_friends=FALSE).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/sync"
)

// FriendsNotifierFactory construit un emitter de notifications pour un slug
// (par-joueur). Optionnel : si nil sur le service, aucune notification n'est
// émise. Pattern miroir de api/handlers.NotificationsEmitterFactory.
type FriendsNotifierFactory func(ctx context.Context, slug string) (notifications.Emitter, error)

// FriendsOrchestratorResult agrège les résultats du recompute multi-DB.
type FriendsOrchestratorResult struct {
	Processed       int               // nombre de player DBs traitées avec succès
	Failed          int               // nombre de player DBs en échec (continuation, pas blocking)
	TotalPromoted   int64             // somme des MatchesPromoted sur toutes les DBs
	Duration        time.Duration     // durée totale de l'orchestration
	PerPlayerErrors map[string]string // map player_slug → erreur (si Failed)
}

// FriendsGamertagsLoader retourne la liste courante des amis configurés.
// Implémenté typiquement par settings.Store.Load().FriendGamertags.
type FriendsGamertagsLoader func() ([]string, error)

// FriendsOrchestratorService orchestre le recompute is_with_friends sur toutes
// les player DBs configurées (multi-titres).
type FriendsOrchestratorService struct {
	cfg         *config.AppConfig
	loadFriends FriendsGamertagsLoader
	notifierFor FriendsNotifierFactory // optionnel — §6 notif friend_sync_completed
}

// NewFriendsOrchestratorService crée un orchestrator. cfg fournit LoadPlayers
// + chemins DB ; loadFriends résout settings.friend_gamertags à la demande.
func NewFriendsOrchestratorService(
	cfg *config.AppConfig,
	loadFriends FriendsGamertagsLoader,
) *FriendsOrchestratorService {
	return &FriendsOrchestratorService{cfg: cfg, loadFriends: loadFriends}
}

// WithNotifier branche une factory de notifier pour émettre `friend_sync_completed`
// par-joueur quand un recompute promeut au moins 1 match. Best-effort : tout
// échec d'émission est loggé en Warn et n'arrête pas l'orchestration.
func (s *FriendsOrchestratorService) WithNotifier(f FriendsNotifierFactory) *FriendsOrchestratorService {
	s.notifierFor = f
	return s
}

// RecomputeAll lance le recompute sur toutes les (title, gamertag) configurées
// avec la liste actuelle des amis. Idempotent (la garde FALSE protège les
// retries). Mode bootstrap initial.
func (s *FriendsOrchestratorService) RecomputeAll(ctx context.Context) (FriendsOrchestratorResult, error) {
	start := time.Now()
	res := FriendsOrchestratorResult{PerPlayerErrors: map[string]string{}}

	friends, err := s.loadFriends()
	if err != nil {
		return res, fmt.Errorf("RecomputeAll loadFriends: %w", err)
	}
	if len(friends) == 0 {
		slog.InfoContext(ctx, "friends orchestrator: no friends configured, skip")
		return res, nil
	}

	// Énumération multi-titres : LoadPlayers() sans filtre = tous les titres.
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		return res, fmt.Errorf("RecomputeAll LoadPlayers: %w", err)
	}

	for _, p := range players {
		if p.IsDemo {
			continue // demo profile, pas de DB réelle
		}
		playerDBPath := config.PlayerDBPath(s.cfg, p.TitleSlug, p.Gamertag)
		sharedDBPath := config.SharedDBPath(s.cfg, p.TitleSlug)

		r, err := sync.RecomputeIsWithFriends(ctx, s.cfg.SharedProvider, playerDBPath, sharedDBPath, p.XUID, friends)
		if err != nil {
			res.Failed++
			res.PerPlayerErrors[p.PlayerSlug] = err.Error()
			slog.ErrorContext(ctx, "friends orchestrator: player failed",
				"player_slug", p.PlayerSlug,
				"title_slug", p.TitleSlug,
				"err", err,
			)
			continue
		}
		res.Processed++
		res.TotalPromoted += r.MatchesPromoted

		// §6 notif friend_sync_completed — émise uniquement quand au moins 1
		// match a effectivement été promu (sinon le recompute est un no-op et
		// notifier l'utilisateur n'apporte rien).
		if r.MatchesPromoted > 0 {
			s.emitFriendSyncCompleted(ctx, p.PlayerSlug, r.MatchesPromoted)
		}
	}

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "friends orchestrator done",
		"processed", res.Processed,
		"failed", res.Failed,
		"total_promoted", res.TotalPromoted,
		"duration_ms", res.Duration.Milliseconds(),
	)
	return res, nil
}

// emitFriendSyncCompleted émet une notif `friend_sync_completed` (in-app) +
// déclenche le webhook Discord si activé (§6.A + §6.B). Émis uniquement quand
// `promoted > 0`. Best-effort : warn log + continue, jamais d'erreur propagée.
func (s *FriendsOrchestratorService) emitFriendSyncCompleted(ctx context.Context, slug string, promoted int64) {
	if s.notifierFor == nil {
		return
	}
	em, err := s.notifierFor(ctx, slug)
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: friend_sync_completed factory failed",
			"player_slug", slug, "err", err)
		return
	}
	if err := em.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryFriendSyncCompleted,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.friend_sync_completed.title",
		BodyKey:  "notif.friend_sync_completed.body",
		Params:   map[string]any{"promoted": promoted, "slug": slug},
		Source:   "friends_orchestrator",
	}); err != nil {
		slog.WarnContext(ctx, "notifications: friend_sync_completed emit",
			"player_slug", slug, "err", err)
	}
	// §6.B Discord : webhook failsafe (no-op si webhook vide / NotifyFriends off).
	if s.cfg != nil {
		notifyCfg := notify.LoadNotifyConfig(s.cfg.AppSettingsPath)
		go notify.NotifyFriendSyncCompleted(notifyCfg, slug, promoted)
	}
}

// OnFriendsChanged est appelé après un PATCH /settings qui modifie
// friend_gamertags. Identique à RecomputeAll : la garde FALSE rend
// l'opération idempotente, donc relancer le recompute complet est sûr et
// permet de couvrir les ajouts ET les sessions historiques manquées.
//
// Implémente port.FriendsOrchestrator. Note : la sémantique additive ne
// démote PAS les anciens matchs si un ami est retiré (cf. friends_recompute.go).
func (s *FriendsOrchestratorService) OnFriendsChanged(ctx context.Context) error {
	_, err := s.RecomputeAll(ctx)
	return err
}
