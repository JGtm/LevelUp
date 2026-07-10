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
	"levelup/go-api/internal/platform/auth"
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
			// Étape 0 attribution : détenteur identifié « réseau sous writer »
			// (PersistPerMatchRatings = 1 fetch carnage par nouveau match).
			runCtx := ctxkeys.WithDBWriterLabel(ctxkeys.WithTitleSlug(ctx, halo5.TitleSlug), "h5_livesync_postscore")
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
			if _, lerr := syncpkg.RunLUSRV2ShadowOwnerOnly(runCtx, playerDB.SQLDb(), syncpkg.NewPinnedSharedAccess(shared), xuid); lerr != nil {
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
		// Filet de convergence à 0 insert : sonde « reste-t-il des matchs en shared
		// SANS enrichment par-joueur ? » (parité avec hasConvergenceBacklog Infinite).
		// Réutilise le helper title-agnostic CountSharedMatchesMissingEnrichment, via
		// les mêmes acquisitions que loadKnownMatchIDs (handle player + shared coordonné
		// B-swap). Consultée UNIQUEMENT à 0 insert (cf. shouldRunPostScore). Erreur
		// d'ouverture → false : on ne force pas un PostScore sur une DB inaccessible.
		HasEnrichmentBacklog: func(ctx context.Context) bool {
			return h5HasEnrichmentBacklog(ctx, sharedProvider, playerPath, sharedPath, xuid)
		},
		// Achievements Xbox post-sync (parité Infinite). Gaté sur la capability du
		// titre AU CÂBLAGE (hook nil si absente → jamais appelé). Réutilise le chemin
		// title-aware SyncEngineForTitle.RunAchievementsOnly (lease player DB + gate
		// capability interne + provider MSAL). Best-effort côté runner.
		RunAchievements: buildAchievementsHook(cfg, gamertag, xuid),
	}, logger)
}

// buildAchievementsHook construit le hook de sync des achievements Xbox du titre
// Halo 5, ou nil si le titre ne déclare PAS la capability achievements (gate
// title-agnostic AU CÂBLAGE, jamais slug==literal — archlint). Réutilise le chemin
// title-aware existant (SyncEngineForTitle.RunAchievementsOnly), qui gère le lease
// player DB, le gate capability interne et l'ouverture metadata. RunAchievementsOnly
// retourne false sur erreur (déjà loggée) — on la remonte en err best-effort pour
// que le runner la trace dans le SyncResult.
func buildAchievementsHook(cfg *config.AppConfig, gamertag, xuid string) func(ctx context.Context) error {
	desc := titlePkg.DefaultRegistry().Get(halo5.TitleSlug)
	if desc == nil || !desc.HasCapability(titlePkg.CapAchievements) {
		return nil
	}
	return func(ctx context.Context) error {
		// Provider MSAL config-only (zéro dépendance injectée), comme le CLI
		// sync-achievements (cmd_sync_achievements.go). NewSyncEngineForTitle résout
		// tous les chemins DB via PathResolver + titleSlug.
		engine := syncpkg.NewSyncEngineForTitle(cfg.RepoRoot, halo5.TitleSlug, gamertag, xuid, nil, auth.NewMSALProvider())
		if !engine.RunAchievementsOnly(ctx) {
			return fmt.Errorf("RunAchievementsOnly a échoué (voir logs)")
		}
		return nil
	}
}

// h5HasEnrichmentBacklog ouvre player+shared (mêmes helpers que loadKnownMatchIDs /
// PostScore — handle player RW ref-compté + shared via le provider B-swap) et compte
// les matchs présents en shared.match_participants pour le xuid mais SANS row
// player_match_enrichment (CountSharedMatchesMissingEnrichment, title-agnostic).
// > 0 → un coéquipier a inséré des matchs du titre que ce joueur n'a jamais enrichis.
// Erreur d'ouverture (player DB pas encore créée au tout 1er run, shared verrouillé)
// → false : on ne force pas un reconcile sur une DB inaccessible (best-effort). Sonde
// consultée UNIQUEMENT à 0 insert (cf. shouldRunPostScore), donc hors-cycle nominal.
func h5HasEnrichmentBacklog(ctx context.Context, sharedProvider sharedprovider.Provider, playerPath, sharedPath, xuid string) bool {
	playerDB, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		return false
	}
	defer playerDB.Close()
	shared, release, err := syncpkg.AcquireSharedWriterStandalone(ctxkeys.WithDBWriterLabel(ctx, "h5_livesync_backlog_probe"), sharedProvider, sharedPath)
	if err != nil {
		return false
	}
	defer release()
	return syncpkg.CountSharedMatchesMissingEnrichment(ctx, playerDB.SQLDb(), shared, xuid) > 0
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
