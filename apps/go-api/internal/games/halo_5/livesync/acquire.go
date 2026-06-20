package livesync

import (
	"context"
	"fmt"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/auth/pool"
)

// AcquireRunner construit le runner live d'un titre (Halo 5+) prêt à RunDelta,
// avec l'auth résolue depuis le pool : pinne un token sur le joueur, le pose dans
// le ctx (l'adapter du titre lit ctxkeys.HaloTokens), et instancie le runner
// registry-driven (RunnerForTitle). C'est la SOURCE UNIQUE de cette séquence,
// partagée par les deux déclencheurs serveur :
//   - le scheduler périodique (scheduler.acquireLiveTitleRunner) ;
//   - le watcher temps-réel (sync.Trigger via factory câblée dans main.go).
//
// release libère le lease pool — à DIFFÉRER par le caller : le SpartanToken doit
// rester valide pendant TOUT le RunDelta (fetch cryptum complet). En cas d'erreur,
// release est un no-op (le lease éventuel est déjà libéré) et le *Runner est nil.
//
// Erreur si : pool nil, Acquire échoue (joueur absent du pool / annulation), ou
// slug non câblé live (RunnerForTitle == nil).
func AcquireRunner(ctx context.Context, tokenPool pool.Pool, cfg *config.AppConfig, slug, gamertag, xuid string) (*Runner, context.Context, func(), error) {
	noop := func() {}
	if tokenPool == nil {
		return nil, ctx, noop, fmt.Errorf("pool absent (auth %s indisponible)", slug)
	}
	lease, err := tokenPool.Acquire(ctx, pool.PolicyPinnedPlayer, gamertag)
	if err != nil {
		return nil, ctx, noop, fmt.Errorf("acquire token pool (%s): %w", gamertag, err)
	}
	r := RunnerForTitle(slug, cfg, gamertag, xuid)
	if r == nil {
		lease.Release()
		return nil, ctx, noop, fmt.Errorf("RunnerForTitle(%s) = nil (titre non câblé live)", slug)
	}
	return r, ctxkeys.WithHaloAuth(ctx, lease.Tokens, xuid), lease.Release, nil
}
