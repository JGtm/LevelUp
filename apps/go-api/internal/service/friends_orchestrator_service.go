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
	"levelup/go-api/internal/sync"
)

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
}

// NewFriendsOrchestratorService crée un orchestrator. cfg fournit LoadPlayers
// + chemins DB ; loadFriends résout settings.friend_gamertags à la demande.
func NewFriendsOrchestratorService(
	cfg *config.AppConfig,
	loadFriends FriendsGamertagsLoader,
) *FriendsOrchestratorService {
	return &FriendsOrchestratorService{cfg: cfg, loadFriends: loadFriends}
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

		r, err := sync.RecomputeIsWithFriends(ctx, playerDBPath, sharedDBPath, p.XUID, friends)
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
