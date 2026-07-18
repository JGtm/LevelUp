package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// activeRankedPlaylists retourne les playlists classées à interroger pour compléter
// les CSR du joueur : les playlists réellement ACTIVES découvertes par le cron
// classement (dernier batch de world_csr_leaderboard_snapshots), au lieu de la liste
// statique rankedplaylists.Active() (historiquement 4). Fallback sur Active() si le
// provider shared est absent, la table vide, ou en cas d'erreur — jamais MOINS que le
// comportement historique. Season-agnostic (le format de saison Waypoint diffère de
// e.csrSeasonID) : on lit le dernier scrape, qui porte les actives courantes.
func (e *SyncEngine) activeRankedPlaylists(ctx context.Context) []rankedplaylists.Playlist {
	fallback := rankedplaylists.Active()
	if e.sharedProvider == nil {
		return fallback
	}
	db, release, err := e.sharedProvider.Get(ctx)
	if err != nil {
		slog.DebugContext(ctx, "activeRankedPlaylists: shared reader indispo — fallback statique", "err", err)
		return fallback
	}
	defer release()
	// LB1 (revue 2026-07) : partition PAR playlist, pas MAX(fetched_at) GLOBAL. Le
	// scraper estampille fetched_at PAR playlist (chacune scrapée à un instant distinct
	// du batch) ; un MAX global ne matchait que la DERNIÈRE playlist scrapée → 1 seule
	// des ~7 actives retournée, fallback jamais déclenché. On lit la vue
	// world_csr_leaderboard_latest (source unique du « dernier par (titre, saison,
	// playlist) ») → toutes les playlists actives, chacune à son propre dernier fetch.
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest
		WHERE title_slug = ? AND playlist_id <> ''`, e.titleSlug)
	if err != nil {
		slog.DebugContext(ctx, "activeRankedPlaylists: requête échouée — fallback statique", "err", err)
		return fallback
	}
	defer rows.Close()
	var out []rankedplaylists.Playlist
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fallback
		}
		if pl, ok := rankedplaylists.Lookup(id); ok {
			out = append(out, pl)
			continue
		}
		// Playlist active hors référence statique : on l'interroge quand même (nom vide,
		// la lecture catalogue-first complètera le libellé).
		out = append(out, rankedplaylists.Playlist{AssetID: id})
	}
	if err := rows.Err(); err != nil || len(out) == 0 {
		return fallback
	}
	return out
}

func (e *SyncEngine) runCSRSnapshotSync(ctx context.Context, playerDB *sql.DB, client HaloClient) ([]PlayerPlaylistCSR, error) {
	if strings.TrimSpace(e.csrSeasonID) == "" {
		// Visibilité explicite : sans cette config, player_csr_snapshots reste vide
		// éternellement et la home affiche "Aucun classement". Bug racine difficile
		// à diagnostiquer côté UI ; un WARN rend le silence visible aux ops.
		slog.WarnContext(ctx,
			"post-sync: CSR snapshot sync SKIPPED — csr_season_id non configuré "+
				"(ajouter le champ \"csr_season_id\" dans app_settings.json, ex. \"CsrSeason13-1\", "+
				"ou définir l'env var LEVELUP_CSR_SEASON_ID)",
			"gamertag", e.gamertag,
		)
		return nil, nil
	}
	slog.DebugContext(ctx, "post-sync: sync CSR snapshots", "gamertag", e.gamertag, "season", e.csrSeasonID)
	activePlaylists := e.activeRankedPlaylists(ctx)
	csrs, err := syncPlayerCSRs(ctx, client, playerDB, e.xuid, e.csrSeasonID, activePlaylists)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: CSR snapshots échoué", "gamertag", e.gamertag, "err", err)
		return nil, err
	}
	slog.DebugContext(ctx, "post-sync: CSR snapshots sauvegardés", "gamertag", e.gamertag, "count", len(csrs))
	return csrs, nil
}

// achievementsOutcome distingue les issues du sync achievements. Motivation : le hook
// runner Halo 5 (buildAchievementsHook) ne doit remonter une erreur (« sync terminée
// avec erreurs partielles ») QUE sur un échec RÉEL — pas sur un skip bénin (pas de
// provider/capability/token), attendu et récurrent pour H5 (les tokens sync_meta sont
// souvent indisponibles un cycle donné ; ils se resynchronisent ensuite). Sans cette
// distinction, chaque cycle sans token émettait un WARN + un status=failure trompeurs.
type achievementsOutcome int

const (
	achievementsSynced  achievementsOutcome = iota // succès du fetch + persist Xbox
	achievementsSkipped                            // skip bénin : provider/capability/token absent
	achievementsFailed                             // échec réel : XSTS, metadata, HTTP, DB
)

// runAchievementsSync récupère les achievements Xbox pour le joueur et les persiste.
// Retourne l'issue (synced / skipped / failed). Nécessite e.provider non nil ; skippé
// (achievementsSkipped) sinon.
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) achievementsOutcome {
	if e.provider == nil {
		slog.DebugContext(ctx, "achievements: provider nil — sync ignorée", "gamertag", e.gamertag)
		return achievementsSkipped
	}
	// Gate title-agnostic (skill arch-rules) : ne tenter le fetch Xbox que si le
	// titre déclare la capability achievements. Un futur titre sans succès Xbox
	// (xbox_title_id absent → fetch voué à l'échec) est skippé proprement, sans
	// brancher sur slug==literal. HINF + Halo 5 la déclarent → comportement inchangé.
	if desc := titlePkg.DefaultRegistry().Get(e.titleSlug); desc != nil && !desc.HasCapability(titlePkg.CapAchievements) {
		slog.DebugContext(ctx, "achievements: capability absente — sync ignorée",
			"gamertag", e.gamertag, "title_slug", e.titleSlug)
		return achievementsSkipped
	}

	// Résoudre l'access_token Xbox Live : store watcher_tokens d'abord (ADR 0023),
	// puis résidus legacy sync_meta / env var.
	accessToken, err := e.resolveAchievementsAccessToken(ctx, playerDB)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec résolution access_token",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}
	if accessToken == "" {
		// Skip bénin, attendu et récurrent (H5) : Debug uniquement, pas d'erreur
		// remontée — le token se resynchronisera à un cycle ultérieur.
		slog.DebugContext(ctx, "achievements: aucun access_token disponible — sync ignorée",
			"gamertag", e.gamertag)
		return achievementsSkipped
	}

	// Obtenir un XSTS token pour Xbox Live.
	xstsResult, err := auth.AcquireXSTSForRTA(ctx, accessToken)
	if err != nil {
		slog.WarnContext(ctx, "achievements: échec acquisition XSTS",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}

	// Phase 2 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : cache duckdbpkg pour
	// l'ecriture RW de metadata (achievements upsert). Aligne le DSN avec
	// engine.go:249 et citations_backfill.go via OpenReadWriteShared
	// (cle "rw:"+path partagee).
	//
	// Fix race 2026-05-27 : `xbox_achievement_definitions` est globale (1 row
	// par achievement_id, 144 communs aux 4 joueurs). 2 SyncEngine en
	// parallèle (cf. Coordinator parallel_slots:2 ou scheduler errgroup) qui
	// faisaient un upsert (INSERT-OR-UPDATE via conflict clause) sur la même
	// row déclenchaient "TransactionContext Error: Conflict on update!" côté
	// DuckDB (vu dans logs/sync.log 23:58:11 Madina97294 et précédents).
	// Sérialisation applicative via dblease.AcquireWriterCtx(KindMetadata)
	// — bloque le 2ème caller jusqu'au Release du 1er, sans contention DuckDB.
	metadataLease, err := dblease.AcquireWriterCtx(ctx, nil, e.metadataDBPath, dblease.KindMetadata)
	if err != nil {
		slog.WarnContext(ctx, "achievements: acquisition lease metadata échouée",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}
	defer metadataLease.Release()

	metadataHandle, err := duckdbpkg.OpenReadWriteShared(e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "achievements: ouverture metadata DB échouée",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}
	defer metadataHandle.Close()
	metadataDB := metadataHandle.SQLDb()

	client := NewXboxHTTPClient(xstsResult, titlePkg.XboxTitleIDFor(e.titleSlug))
	if err := SyncAchievements(ctx, client, e.resolver, metadataDB, playerDB, e.xuid, e.titleSlug); err != nil {
		slog.WarnContext(ctx, "achievements: sync échouée",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}

	slog.InfoContext(ctx, "achievements: sync terminée avec succès", "gamertag", e.gamertag)
	return achievementsSynced
}

// RunAchievementsOnly synchronise uniquement les achievements Xbox du joueur,
// indépendamment du sync des matchs. Utilisé par le CLI sync-achievements pour
// le backfill admin one-shot. Retourne true UNIQUEMENT si la sync a réussi (un skip
// bénin — pas de token ce cycle — rend false, cohérent avec l'usage CLI historique).
func (e *SyncEngine) RunAchievementsOnly(ctx context.Context) bool {
	return e.runAchievementsOnlyOutcome(ctx) == achievementsSynced
}

// RunAchievementsHook exécute le sync achievements pour un hook post-sync (runner
// live, ex. Halo 5) et retourne une erreur UNIQUEMENT sur un échec RÉEL (XSTS /
// metadata / HTTP / DB). Un skip bénin (pas de provider/capability/token ce cycle)
// → nil : le runner ne doit PAS marquer le cycle en « erreurs partielles » pour un
// cas attendu et récurrent (les tokens sync_meta H5 se resynchronisent au fil des
// cycles). Supprime le bruit d'erreur prod à chaque sync H5.
func (e *SyncEngine) RunAchievementsHook(ctx context.Context) error {
	if e.runAchievementsOnlyOutcome(ctx) == achievementsFailed {
		return fmt.Errorf("sync achievements échoué (voir logs)")
	}
	return nil
}

// runAchievementsOnlyOutcome acquiert le dblease sur la player DB (évite les collisions
// avec un sync concurrent), ouvre la player DB, puis délègue à runAchievementsSync.
// Provider nil / échec lease / échec ouverture → skip ou échec typé (voir godoc du
// type achievementsOutcome).
func (e *SyncEngine) runAchievementsOnlyOutcome(ctx context.Context) achievementsOutcome {
	if e.provider == nil {
		slog.WarnContext(ctx, "achievements: provider nil — sync ignorée",
			"gamertag", e.gamertag)
		return achievementsSkipped
	}

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, e.playerDBPath, dblease.KindPlayer)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: lease player DB échoué",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(e.playerDBPath)
	if err != nil {
		slog.ErrorContext(ctx, "achievements: ouverture player DB échouée",
			"gamertag", e.gamertag, "err", err)
		return achievementsFailed
	}
	defer playerHandle.Close() //nolint:errcheck

	return e.runAchievementsSync(ctx, playerHandle.SQLDb())
}

// selectMatchesMissingDominanceFlags retourne les match_ids dont le dominance_flag
// est NULL (jamais calculé). La valeur 0 = "aucune dominance détectée" est un
// résultat valide et non re-traité.
func selectMatchesMissingDominanceFlags(ctx context.Context, playerDB *sql.DB) ([]string, error) {
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment_latest WHERE dominance_flag IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// mergeUniqMatchIDs fusionne deux slices de match_ids en dédupliquant, a en tête.
func mergeUniqMatchIDs(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, id := range a {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	for _, id := range b {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// seedCatalogFromCSRs ouvre metadata.duckdb en RW et appelle seedPlaylistsCatalog.
// Best-effort : log WARN si metadata inaccessible, n'interrompt jamais le pipeline.
func (e *SyncEngine) seedCatalogFromCSRs(ctx context.Context, csrs []PlayerPlaylistCSR) {
	if e.metadataDBPath == "" || len(csrs) == 0 {
		return
	}
	// META-1 (revue 2026-06-01) : sérialisation applicative via le lease KindMetadata,
	// EXACTEMENT comme runAchievementsSync. metadata.duckdb est un fichier PARTAGÉ écrit
	// concurremment par le post-sync de N joueurs (Coordinator parallel_slots>1). Sans
	// ce lease, deux seeds concurrents font un UPDATE-then-INSERT sur la même row
	// playlists_catalog → "TransactionContext Error: Conflict on update!" (même classe
	// d'incident que xbox_achievement_definitions). Ordre de lock cohérent avec
	// runAchievementsSync : KindPlayer (déjà tenu par le run) puis KindMetadata.
	metadataLease, err := dblease.AcquireWriterCtx(ctx, nil, e.metadataDBPath, dblease.KindMetadata)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: catalog seed — acquisition lease metadata échouée",
			"gamertag", e.gamertag, "err", err)
		return
	}
	defer metadataLease.Release()

	mh, err := duckdbpkg.OpenReadWriteShared(e.metadataDBPath)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: catalog seed désactivé (metadata inaccessible)",
			"gamertag", e.gamertag, "err", err)
		return
	}
	defer mh.Close()
	seedPlaylistsCatalog(ctx, mh.SQLDb(), csrs, e.titleSlug)
}

// resolveAchievementsAccessToken résout l'access_token Xbox Live (achievements)
// selon la priorité ADR 0023 : store watcher_tokens d'abord, puis les résidus
// legacy sync_meta / env var. Délègue à auth.ResolveMSAccessTokenStoreFirst
// (source UNIQUE de l'ordre de résolution, partagée avec world-enrich).
//
// Avant ce câblage, ce chemin lisait EXCLUSIVEMENT sync_meta et n'a jamais
// consulté le store → il servait toujours un RT legacy et comptait la télémétrie
// de dépréciation duckdb_oauth à chaque post-sync des 4 joueurs (incident prod
// 2026-07-12), alors que le store watcher_tokens couvrait ces joueurs. Store-first,
// la télémétrie ne se déclenche plus qu'en vraie absence de RT store.
//
// Retourne ("", nil) si aucun token n'est disponible (non fatal — skip achievements).
func (e *SyncEngine) resolveAchievementsAccessToken(ctx context.Context, playerDB *sql.DB) (string, error) {
	legacy := e.readLegacyAuthInputs(ctx, playerDB)
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(e.repoRoot).WatcherTokensDir())
	return auth.ResolveMSAccessTokenStoreFirst(ctx, e.provider, store, e.xuid, e.gamertag, legacy)
}

// readLegacyAuthInputs lit les résidus legacy depuis sync_meta (DB déjà ouverte)
// + le fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GT>. Best-effort (champs vides
// si absents). Ces valeurs ne servent QUE si le store watcher_tokens ne couvre pas
// le joueur (cf. ResolveMSAccessTokenStoreFirst) → à supprimer en Phase 5 (D2).
func (e *SyncEngine) readLegacyAuthInputs(ctx context.Context, playerDB *sql.DB) auth.LegacyAuthInputs {
	var cacheJSON, refreshToken string
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'msal_token_cache'").Scan(&cacheJSON); err != nil {
		slog.DebugContext(ctx, "achievements: msal_token_cache absent", "gamertag", e.gamertag)
	}
	if err := playerDB.QueryRowContext(ctx,
		"SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'").Scan(&refreshToken); err != nil {
		slog.DebugContext(ctx, "achievements: oauth_refresh_token absent", "gamertag", e.gamertag)
	}

	// Fallback env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> (résidu dev/transition).
	fromEnv := false
	if refreshToken == "" && e.gamertag != "" {
		key := strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(e.gamertag))
		if v := os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key); v != "" {
			refreshToken = v
			fromEnv = true
		}
	}

	return auth.LegacyAuthInputs{
		OAuthRT:        refreshToken,
		MSALCache:      cacheJSON,
		Source:         "player_db.sync_meta",
		OAuthRTFromEnv: fromEnv,
	}
}
