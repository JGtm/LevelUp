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
		// MULTI-COMPTE (anti rate-limit) : la limite PeopleHub est PAR COMPTE, donc
		// répartir les résolutions en round-robin sur tous les comptes tokenisés
		// multiplie le quota effectif (le storm 429 venait d'un seul compte saturé).
		// BuildMultiResolver ignore les comptes sans token → fallback single si aucun.
		resolvers, _, err := worldenrich.BuildMultiResolver(cfg, allResolverGamertags(cfg, viewerGamertag))
		if err != nil || len(resolvers) == 0 {
			base, berr := worldenrich.BuildResolver(cfg, viewerGamertag)
			if berr != nil {
				logger.WarnContext(ctx, "h5 sync: resolver PeopleHub indisponible (xuid roster non résolus)",
					"viewer", viewerGamertag, "err", berr)
				return nil
			}
			resolvers = []*auth.PeopleHubResolver{base}
		}
		// Graine = mapping déjà connu (shared.xuid_aliases, écrit par les runs
		// précédents) → on NE re-résout PAS les joueurs déjà vus via PeopleHub.
		// Le self (xuid db_profiles) prime. La persistance des NOUVELLES résolutions
		// passe par l'ingest (AddXUIDAliases → shared.xuid_aliases), d'où persist=nil.
		sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
		seed := loadXUIDAliasesSeed(ctx, sharedPath)
		seed[viewerGamertag] = viewerXUID
		return worldenrich.NewCachingResolver(resolvers, seed, nil)
	}
}

// allResolverGamertags énumère les comptes (gamertags) utilisables pour la
// résolution xuid round-robin : le viewer + tous les joueurs déclarés (db_profiles).
// BuildMultiResolver écarte ceux sans token. Dédup, viewer en tête.
func allResolverGamertags(cfg *config.AppConfig, viewerGamertag string) []string {
	out := []string{viewerGamertag}
	seen := map[string]bool{viewerGamertag: true}
	if players, err := cfg.LoadPlayers(""); err == nil {
		for i := range players {
			gt := players[i].Gamertag
			if gt != "" && !seen[gt] {
				seen[gt] = true
				out = append(out, gt)
			}
		}
	}
	return out
}
