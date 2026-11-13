// Package service — sync_jobs_engine.go : construction du moteur d'un job de sync.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	settings_platform "levelup/go-api/internal/platform/settings"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
)

// newEngineFor instancie le moteur de sync d'un joueur pour la sync initiale, avec le chargeur des
// amis DU joueur (le recalcul is_with_friends du post-sync delta reste actif), les options de
// session, la saison CSR, le provider shared du titre (nil = mode hérité), l'enfilage des rejeux, le
// scan des médias et le hook Prestige. Le moteur écrit dans les bases du titre demandé.
func (s *SyncJobService) newEngineFor(titleSlug, gamertag, xuid string, tokens *domain.HaloTokens) *go_sync.SyncEngine {
	var loader go_sync.FriendsLoader
	if s.friendStore != nil && xuid != "" {
		loader = func() ([]string, error) {
			return s.friendStore.Get(xuid)
		}
	}
	engine := go_sync.NewSyncEngineForTitle(s.engineRoot(), titleSlug, gamertag, xuid, tokens, s.provider).
		WithFriendsLoader(loader).
		WithSessionOptionsLoader(settings_platform.SessionOptionsLoader(s.settingsStore, s.cfg.TitleSettingsPath(titleSlug))).
		WithCSRSeasonID(s.cfg.CurrentCSRSeasonID)
	if p := s.cfg.SharedProviderForTitle(titleSlug); p != nil {
		engine = engine.WithSharedProvider(p)
	}
	// Le hook des artefacts de rejeu s'installe toujours : c'est lui qui décide (réglage relu à chaque
	// cycle). Même câblage que le moteur de l'auto-sync.
	if s.settingsStore != nil {
		engine = engine.WithReplayArtifacts(replayartifacts.NewHook(s.cfg, s.settingsStore, s.replayEnqueue))
		engine = engine.WithMediaScanHook(NewLiveMediaScanHook(s.cfg, s.settingsStore, gamertag))
	}
	// Le hook Prestige tourne à la fin du pipeline post-sync, pendant que le bail est encore tenu.
	if s.prestigeHook != nil {
		engine = engine.WithPrestigeHook(func(hookCtx context.Context, playerSlug, ts string) {
			s.prestigeHook(hookCtx, playerSlug, ts)
		})
	}
	return engine
}

// defaultRunnerFor choisit le runner d'une sync initiale d'après le registre des titres, jamais d'après
// le slug : un titre en suivi live seul (Halo 5) reçoit le runner dédié que rend le `LiveRunnerBuilder`
// et un contexte portant le jeton Spartan (l'adapter le lit du contexte) ; les autres reçoivent le
// moteur de sync, le contexte inchangé.
func (s *SyncJobService) defaultRunnerFor(ctx context.Context, titleSlug, gamertag, xuid string, tokens *domain.HaloTokens) (DeltaRunner, context.Context) {
	if s.liveRunner != nil {
		if r := s.liveRunner(titleSlug, gamertag, xuid); r != nil {
			return r, ctxkeys.WithHaloAuth(ctx, tokens, xuid)
		}
	}
	return s.newEngineFor(titleSlug, gamertag, xuid, tokens), ctx
}

// newPooledEngine construit le moteur du sync manuel DELTA : celui de l'auto-sync (jetons du pool
// partagé, post-sync, file d'écriture), l'authentification venant du pool et non de la session. Sans
// constructeur injecté, repli sur le moteur de la sync initiale avec des jetons vides : le câblage
// manque, l'authentification échouera probablement.
//
// La sync initiale reste sur les jetons de la session : elle cible un joueur fraîchement connecté,
// pas encore dans le pool.
func (s *SyncJobService) newPooledEngine(ctx context.Context, gamertag, xuid string) *go_sync.SyncEngine {
	if s.engineBuilder != nil {
		slog.DebugContext(ctx, "sync_jobs: moteur sync manuel construit via le pool partagé (aligné auto-sync)",
			"gamertag", gamertag, "xuid", xuid)
		return s.engineBuilder(ctx, gamertag, xuid)
	}
	slog.WarnContext(ctx, "sync_jobs: EngineBuilder absent, repli sur le moteur de la sync initiale (jetons vides, auth probablement en échec)",
		"gamertag", gamertag, "xuid", xuid)
	return s.newEngineFor(ctxkeys.TitleSlug(ctx), gamertag, xuid, &domain.HaloTokens{})
}
