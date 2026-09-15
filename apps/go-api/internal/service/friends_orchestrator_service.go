// Package service — friends_orchestrator_service.go : orchestration multi-DB
// du recompute is_with_friends.
//
// §4 du plan Squad/Sessions overhaul. Itère tous les couples (titleSlug,
// gamertag) configurés via cfg.LoadPlayers() et invoque sync.RecomputeIsWithFriends
// pour chaque player DB. Erreurs per-DB ne stoppent pas les autres ; agrégat
// {processed, failed, totalPromoted} retourné.
//
// Les amis sont résolus PAR JOUEUR (data/global/player_friends.json) : chaque
// player DB est recalculée avec la liste de SON propriétaire.
//
// Modes d'invocation :
//   - Bootstrap initial : RecomputeAll (tous les joueurs, chacun avec ses amis).
//   - Incrémental : RecomputeForPlayer après un PUT de la liste d'un joueur
//     (idempotent via la garde is_with_friends=FALSE).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
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

// FriendsGamertagsLoader retourne la liste courante des amis D'UN JOUEUR
// (clé : xuid). Implémenté typiquement par friendstore.FriendStore.Get.
type FriendsGamertagsLoader func(xuid string) ([]string, error)

// FriendsOrchestratorService orchestre le recompute is_with_friends sur toutes
// les player DBs configurées (multi-titres).
type FriendsOrchestratorService struct {
	cfg         *config.AppConfig
	loadFriends FriendsGamertagsLoader
	notifierFor FriendsNotifierFactory // optionnel — §6 notif friend_sync_completed
}

// NewFriendsOrchestratorService crée un orchestrator. cfg fournit LoadPlayers
// + chemins DB ; loadFriends résout la liste d'amis d'un joueur à la demande.
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

	// Énumération multi-titres : LoadPlayers() sans filtre = tous les titres.
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		return res, fmt.Errorf("RecomputeAll LoadPlayers: %w", err)
	}
	// Recompute is_with_friends sur les joueurs ACTIFS uniquement (skip pauses).
	players = domain.SyncablePlayers(players)

	for _, p := range players {
		if p.IsDemo {
			continue // demo profile, pas de DB réelle
		}
		// Amis DU joueur traité : une liste par profil, plus une pour l'instance.
		friends, ferr := s.loadFriends(p.XUID)
		if ferr != nil {
			res.Failed++
			res.PerPlayerErrors[p.PlayerSlug] = ferr.Error()
			slog.ErrorContext(ctx, "friends orchestrator: lecture des amis échouée",
				"player_slug", p.PlayerSlug, "err", ferr)
			continue
		}
		if len(friends) == 0 {
			continue // aucun ami déclaré pour ce joueur : rien à promouvoir
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
		// PMT-11 : libellés Discord du titre courant (ctx) ; failsafe Halo.
		notifyCfg.Labels = notify.LabelsForSlug(ctxkeys.TitleSlug(ctx))
		go notify.NotifyFriendSyncCompleted(notifyCfg, slug, promoted)
	}
}

// RecomputeForPlayer relance le recompute is_with_friends sur les DBs DU SEUL
// joueur donné (tous ses titres), après une écriture de SA liste d'amis. La
// garde FALSE rend l'opération idempotente ; la sémantique reste additive (un
// ami retiré ne démote pas les anciens matchs, cf. friends_recompute.go).
//
// Retourne le nombre de matchs promus, pour que l'appelant décide d'émettre ou
// non la notification friend_sync_completed.
func (s *FriendsOrchestratorService) RecomputeForPlayer(ctx context.Context, xuid string) (int64, error) {
	if xuid == "" {
		return 0, fmt.Errorf("RecomputeForPlayer: xuid requis")
	}
	friends, err := s.loadFriends(xuid)
	if err != nil {
		return 0, fmt.Errorf("RecomputeForPlayer loadFriends: %w", err)
	}
	if len(friends) == 0 {
		return 0, nil
	}
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		return 0, fmt.Errorf("RecomputeForPlayer LoadPlayers: %w", err)
	}

	var promoted int64
	for _, p := range domain.SyncablePlayers(players) {
		if p.IsDemo || p.XUID != xuid {
			continue
		}
		playerDBPath := config.PlayerDBPath(s.cfg, p.TitleSlug, p.Gamertag)
		sharedDBPath := config.SharedDBPath(s.cfg, p.TitleSlug)
		r, rerr := sync.RecomputeIsWithFriends(ctx, s.cfg.SharedProvider, playerDBPath, sharedDBPath, p.XUID, friends)
		if rerr != nil {
			slog.ErrorContext(ctx, "friends orchestrator: recompute joueur échoué",
				"player_slug", p.PlayerSlug, "title_slug", p.TitleSlug, "err", rerr)
			return promoted, rerr
		}
		promoted += r.MatchesPromoted
		if r.MatchesPromoted > 0 {
			s.emitFriendSyncCompleted(ctx, p.PlayerSlug, r.MatchesPromoted)
		}
	}
	return promoted, nil
}
