package livesync

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/worldenrich"
)

// runnerBuilders mappe le slug d'un titre servi par un runner live-sync DÉDIÉ vers
// son constructeur. Clé = const de package du titre (JAMAIS slug==literal — archlint),
// idiome additionalTitleRegistrars. Halo Infinite N'EST PAS ici (il passe par le
// SyncEngine). Un 3e titre live-only s'ajoute par une entrée + son constructeur.
var runnerBuilders = map[string]func(cfg *config.AppConfig, gamertag, xuid string) *Runner{
	halo5.TitleSlug: newHalo5Runner,
}

// HandlesTitle indique si un titre est servi par un runner live-sync dédié (vs le
// SyncEngine Infinite). Utilisé par les entonnoirs (scheduler/HTTP) pour brancher,
// et par la précondition player-DB (un titre live-only n'a pas de player DB → ne
// doit pas être skippé sur son absence).
func HandlesTitle(slug string) bool {
	_, ok := runnerBuilders[slug]
	return ok
}

// RunnerForTitle construit le runner live-sync d'un titre, ou nil si le titre passe
// par le chemin Infinite (SyncEngine). Pattern caller :
//
//	if r := livesync.RunnerForTitle(slug, cfg, gt, xuid); r != nil { return r }
//	return <SyncEngine / newEngineFor>
func RunnerForTitle(slug string, cfg *config.AppConfig, gamertag, xuid string) *Runner {
	if build, ok := runnerBuilders[slug]; ok {
		return build(cfg, gamertag, xuid)
	}
	return nil
}

// newHalo5Runner câble le runner live Halo 5. Réseau DIFFÉRÉ : source + resolver
// construits à RunDelta (pas ici, qui est sur le hot path de dispatch). viewer.XUID
// = xuid db_profiles (self autoritatif, pas de PeopleHub) ; persist/known-set sur le
// shared h5 (provider nil — legacy, sûr).
func newHalo5Runner(cfg *config.AppConfig, gamertag, xuid string) *Runner {
	sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
	logger := slog.Default()
	return NewRunner(Deps{
		NewSource: halo5.NewCaptureSource,
		Viewer:    canonical.PlayerIdentity{Gamertag: gamertag, XUID: xuid},
		Resolver:  halo5ResolverFactory(cfg, gamertag, xuid, logger),
		LoadKnown: func(ctx context.Context) (map[string]bool, error) {
			return loadKnownMatchIDs(ctx, sharedPath)
		},
		PersistAll: func(ctx context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
			return persistBatches(ctx, sharedPath, batches)
		},
	}, logger)
}

// halo5ResolverFactory retourne un factory LAZY : construit (à RunDelta) un
// CachingResolver PeopleHub résolvant les gamertags du roster → xuid Xbox RÉELS
// (title-agnostic), seedé avec le self (xuid db_profiles → toujours résolu).
// Échec d'auth → nil (xuid roster "" ; le câblage participants resolve-or-skip).
func halo5ResolverFactory(cfg *config.AppConfig, viewerGamertag, viewerXUID string, logger *slog.Logger) func(ctx context.Context) XUIDResolver {
	return func(ctx context.Context) XUIDResolver {
		base, err := worldenrich.BuildResolver(cfg, viewerGamertag)
		if err != nil {
			logger.WarnContext(ctx, "h5 sync: resolver PeopleHub indisponible (xuid roster non résolus)",
				"viewer", viewerGamertag, "err", err)
			return nil
		}
		return worldenrich.NewCachingResolver(
			[]*auth.PeopleHubResolver{base},
			map[string]string{viewerGamertag: viewerXUID}, // self autoritatif
			nil,
		)
	}
}
