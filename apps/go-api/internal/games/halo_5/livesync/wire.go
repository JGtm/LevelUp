package livesync

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
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
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(halo5.TitleSlug)
	playerPath := pr.PlayerDBPath(halo5.TitleSlug, gamertag)
	logger := slog.Default()
	// Provider per-titre du shared h5 : MÊME provider que la lecture (pool joueur
	// h5 via player_resolver.sharedReaderForTitle). En B-swap, lire ET écrire un
	// fichier par le même provider est OBLIGATOIRE — sinon RO (pool) + RW (ce
	// writer) non coordonnés sur le même fichier h5 → DuckDB "different
	// configuration". Le provider draine/ferme le handle RO avant d'ouvrir RW.
	// nil (Manager absent / kill-switch) → AcquireSharedWriterStandalone repasse
	// en legacy (dblease + OpenSharedDB direct) : sûr tant qu'aucun pool ne tient
	// le shared h5 en RO concurremment.
	sharedProvider := sharedProviderForPath(cfg, sharedPath)
	return NewRunner(Deps{
		NewSource: halo5.NewCaptureSource,
		Viewer:    canonical.PlayerIdentity{Gamertag: gamertag, XUID: xuid},
		Resolver:  halo5ResolverFactory(cfg, gamertag, xuid, logger),
		LoadKnown: func(ctx context.Context) (map[string]bool, error) {
			return loadKnownMatchIDs(ctx, sharedProvider, sharedPath)
		},
		PersistAll: func(ctx context.Context, batches []*persist.MatchBatch) ([]*persist.MatchBatch, []string) {
			return persistBatches(ctx, sharedProvider, sharedPath, batches)
		},
		// Hook CSR post-sync (G4 Phase 1) : persiste le CSR arena par playlist dans
		// la player DB h5 (créée à la volée). Réutilise la source live du runner.
		PersistCSR: func(ctx context.Context, src halo5.CaptureSource) (int, error) {
			return persistArenaCSR(ctx, src, playerPath, gamertag)
		},
		// Hook post-score : enrichment PAR JOUEUR (+ LUSR) INCRÉMENTAL des nouveaux
		// matchs, comme le post-sync Infinite. Rend le sync live h5 autonome (l'app
		// enrichit seule, sans cmd/h5-enrich manuel). Best-effort côté runner.
		// Tient le shared (writer coordonné) + la player DB RW le temps du recompute
		// incrémental (force=false → seuls les matchs neufs) : court.
		PostScore: func(ctx context.Context, src halo5.CaptureSource, inserted []string) error {
			runCtx := ctxkeys.WithTitleSlug(ctx, halo5.TitleSlug)
			playerDB, err := syncpkg.OpenPlayerDB(playerPath)
			if err != nil {
				return fmt.Errorf("open player DB: %w", err)
			}
			defer playerDB.Close()
			shared, release, err := syncpkg.AcquireSharedWriterStandalone(runCtx, sharedProvider, sharedPath)
			if err != nil {
				return fmt.Errorf("acquire shared: %w", err)
			}
			defer release()
			friends := otherPlayerGamertags(cfg, gamertag)
			if _, err := syncpkg.BackfillEnrichmentFromShared(runCtx, playerDB.SQLDb(), shared, xuid, friends, false); err != nil {
				return err
			}
			// LUSR incrémental owner-only (gated capability via registre boot + env
			// LUSR_V2 posés au boot serveur). Best-effort.
			if _, lerr := syncpkg.RunLUSRV2ShadowOwnerOnly(runCtx, playerDB.SQLDb(), shared, xuid); lerr != nil {
				slog.WarnContext(runCtx, "h5 post-score: LUSR incrémental échoué (non bloquant)",
					"gamertag", gamertag, "err", lerr)
			}
			// CSR par match (classés → match_skill_rank, priorité CSR>LUSR) + rang SR
			// (career_progression, title-agnostic) des nouveaux matchs, depuis 1 fetch
			// carnage chacun. src déjà authentifié (peu de matchs en delta).
			if csrN, srN := PersistPerMatchRatings(runCtx, src, playerDB.SQLDb(), shared, gamertag, xuid, inserted); csrN > 0 || srN > 0 {
				slog.InfoContext(runCtx, "h5 post-score: ratings par match écrits", "gamertag", gamertag, "csr", csrN, "sr", srN)
			}
			// Progression V2 (streaks/records/milestones/coach) title-agnostic, via le
			// hook injecté au boot (cfg.ProgressionAfterSync = api.BuildProgressionAfterSyncHook).
			// Best-effort. nil en CLI/tests → skip. playerSlug = gamertag pour h5.
			if cfg.ProgressionAfterSync != nil {
				cfg.ProgressionAfterSync(runCtx, halo5.TitleSlug, gamertag)
			}
			return nil
		},
		// Notif « titre prêt » (MT-19 / axe E) : délègue au notifier injecté au boot
		// (cfg.TitleReadyNotifier, = api.BuildTitleReadyNotifier). nil en CLI/tests →
		// no-op. Le titre (halo5.TitleSlug) et le joueur voyagent en arguments ; toute
		// la logique (watermark idempotent + Emit dans le flux du titre par défaut)
		// vit côté api, hors de la couche sync (zéro cycle d'import).
		NotifyFirstSync: func(ctx context.Context, inserted int) {
			if cfg.TitleReadyNotifier != nil {
				cfg.TitleReadyNotifier(ctx, halo5.TitleSlug, gamertag, xuid, inserted)
			}
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
		// Chaque résolveur est une chaîne PeopleHub→Profil Xbox (fallback universel
		// hors graphe social, fix #10) — câblé dans worldenrich, transparent ici.
		resolvers, _, err := worldenrich.BuildMultiResolver(cfg, allResolverGamertags(cfg, viewerGamertag))
		if err != nil || len(resolvers) == 0 {
			base, berr := worldenrich.BuildResolver(cfg, viewerGamertag)
			if berr != nil {
				logger.WarnContext(ctx, "h5 sync: resolver xuid indisponible (xuid roster non résolus)",
					"viewer", viewerGamertag, "err", berr)
				return nil
			}
			resolvers = []worldenrich.XUIDResolver{base}
		}
		// Graine = mapping déjà connu (shared.xuid_aliases, écrit par les runs
		// précédents) → on NE re-résout PAS les joueurs déjà vus via PeopleHub.
		// Le self (xuid db_profiles) prime. La persistance des NOUVELLES résolutions
		// passe par l'ingest (AddXUIDAliases → shared.xuid_aliases), d'où persist=nil.
		sharedPath := titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath(halo5.TitleSlug)
		seed := loadXUIDAliasesSeed(ctx, sharedProviderForPath(cfg, sharedPath), sharedPath)
		seed[viewerGamertag] = viewerXUID
		return worldenrich.NewCachingResolver(resolvers, seed, nil)
	}
}

// sharedProviderForPath résout le provider per-titre du shared d'un path via le
// Manager injecté dans cfg (B-swap). nil si le Manager est absent (mode legacy /
// kill-switch LEVELUP_USE_SHARED_PROVIDER=0) OU si l'ouverture échoue (fichier
// shared du titre pas encore créé) → AcquireSharedWriterStandalone repassera en
// legacy. Pour un titre dont le shared n'existe pas encore (premier run h5), le
// provider échoue ; le legacy crée le fichier au premier OpenSharedDB RW.
func sharedProviderForPath(cfg *config.AppConfig, sharedPath string) sharedprovider.Provider {
	if cfg.SharedManager == nil {
		return nil
	}
	p, err := cfg.SharedManager.For(sharedPath, cfg.UserTimezone)
	if err != nil {
		slog.Warn("livesync: provider per-titre indisponible, fallback legacy",
			"path", sharedPath, "err", err)
		return nil
	}
	return p
}

// otherPlayerGamertags retourne les gamertags des AUTRES joueurs déclarés (hors le
// viewer) — utilisés comme "amis" pour is_with_friends dans le hook post-score.
func otherPlayerGamertags(cfg *config.AppConfig, viewerGamertag string) []string {
	var out []string
	players, err := cfg.LoadPlayers("")
	if err != nil {
		return out
	}
	for i := range players {
		if gt := players[i].Gamertag; gt != "" && gt != viewerGamertag {
			out = append(out, gt)
		}
	}
	return out
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
