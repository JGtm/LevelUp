package livesync

import (
	"context"
	"fmt"
	"log/slog"

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
	// Token PINNÉ sur le joueur d'abord (sync complet : inclut les hooks owner-
	// spécifiques comme les achievements Xbox, qui résolvent leur propre token MSAL).
	// Si le joueur n'a pas de token SAIN (RT Microsoft mort/révoqué → slot pool absent
	// ou malsain), on RETOMBE sur un token du POOL (PolicyAnyPublic) : les stats Halo 5
	// (matchs, carnage, service-record → CSR/SR) ne sont PAS gatées derrière le token du
	// joueur — n'importe quel compte sain les lit. Ainsi le rang/XP d'un joueur au RT
	// mort continue de se synchroniser (le hook achievements échoue best-effort, comme
	// aujourd'hui). Le SpartanToken vit dans le ctx (l'adapter le lit), tagué avec le
	// xuid du joueur CONSULTÉ (la collecte fetche bien ses stats, pas celles du compte pool).
	lease, err := tokenPool.Acquire(ctx, pool.PolicyPinnedPlayer, gamertag)
	if err != nil {
		fallback, ferr := tokenPool.Acquire(ctx, pool.PolicyAnyPublic, "")
		if ferr != nil {
			return nil, ctx, noop, fmt.Errorf("acquire token (%s): pinned=%v ; pool=%w", gamertag, err, ferr)
		}
		slog.WarnContext(ctx, "h5 sync: token joueur indisponible → fallback pool (stats publiques)",
			"slug", slug, "gamertag", gamertag, "xuid", xuid, "pinned_err", err)
		lease = fallback
	}
	r := RunnerForTitle(slug, cfg, gamertag, xuid)
	if r == nil {
		lease.Release()
		return nil, ctx, noop, fmt.Errorf("RunnerForTitle(%s) = nil (titre non câblé live)", slug)
	}
	return r, ctxkeys.WithHaloAuth(ctx, lease.Tokens, xuid), lease.Release, nil
}
