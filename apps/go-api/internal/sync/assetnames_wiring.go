// Package sync — assetnames_wiring.go : câblage de la résolution autonome des
// noms d'assets (internal/assetnames) au moteur de sync.
//
// Pré-pass primary-write : pour les assets neufs vus dans un cycle, on peuple
// metadata.asset_translations (le dictionnaire lu par tout l'affichage end-user
// ET par EnrichRegistryFromMetadata) AVANT l'écriture registry, de sorte que les
// noms soient résolus dès le premier passage — sans heal, sans backfill de masse,
// sans action admin.
//
// Auth : GameCMS EXIGE un token Spartan (401 sans). Les tokens proviennent du
// POOL UNIFIÉ (auth/pool) — la même source que TOUS les syncs — via l'unique
// helper resolveRefsWithPool. Pas de chemin store (haloTokensForDrain) : il
// échoue sur RT périmés. Écriture via ops.UpsertAssetTranslation (ART-safe).
// Cf. plan « Résolution autonome des noms d'assets Halo » (2026-06-13/14).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/platform/halo"
)

// assetSweepMaxAssets : cap du balayage périodique. Large car la traîne d'assets
// distincts est bornée (centaines) ; ExistsFresh rend les balayages suivants ~no-op.
const assetSweepMaxAssets = 500

// WithAssetNameResolution branche le POOL unifié de tokens (auth/pool — la même
// source que tous les syncs) pour la résolution autonome des noms d'assets au
// sync (pré-pass primary-write). p nil → résolution désactivée (parité legacy).
// Le kill-switch LEVELUP_SYNC_RESOLVE_ASSETS=0 désactive aussi (cf. resolveRefsWithPool).
func (e *SyncEngine) WithAssetNameResolution(p pool.Pool) *SyncEngine {
	e.assetPool = p
	return e
}

// resolveCycleAssets résout les noms des assets neufs du cycle vers
// asset_translations, AVANT la phase d'insert (donc avant EnrichRegistryFromMetadata
// dans persistFetchedMatch). Best-effort, convergent. No-op si le pool n'est pas
// câblé ou si le handle metadata n'est pas disponible.
func (e *SyncEngine) resolveCycleAssets(ctx context.Context, fetched []*fetchedMatch) {
	if e.assetPool == nil || e.metaDB == nil {
		return
	}
	var refs []assetnames.AssetRef
	for _, fm := range fetched {
		if fm == nil || fm.Registry == nil {
			continue
		}
		refs = append(refs, collectAssetRefsFromRegistry(fm.Registry)...)
	}
	resolveRefsWithPool(ctx, e.assetPool, e.metaDB, e.titleSlug, e.gamertag, refs, 0)
}

// ResolveAssetsFromStats résout les noms des assets neufs d'un lot de matchs
// (raw Stats JSON déjà fetchés par le pipeline V2) vers asset_translations,
// AVANT la construction/persistance des batches (primary write V2). Réutilise la
// même logique que le pré-pass V1. Best-effort. p/metaDB nil → no-op.
func ResolveAssetsFromStats(ctx context.Context, p pool.Pool, metaDB *sql.DB, titleSlug string, statsList []map[string]any) {
	if p == nil || metaDB == nil || len(statsList) == 0 {
		return
	}
	var refs []assetnames.AssetRef
	for _, stats := range statsList {
		if stats == nil {
			continue
		}
		// gamertag inutile pour l'extraction des assets (on ne lit que les champs
		// playlist/map/pair/game_variant + versions).
		reg, err := ExtractRegistry(stats, "")
		if err != nil || reg == nil {
			continue
		}
		refs = append(refs, collectAssetRefsFromRegistry(reg)...)
	}
	resolveRefsWithPool(ctx, p, metaDB, titleSlug, "v2_cycle", refs, 0)
}

// ResolveUnresolvedAssetNames balaye match_registry pour les assets restés en UUID
// (name == id) et résout leurs noms localisés → asset_translations. FILET de
// convergence pour la traîne (asset jamais résolu ET jamais rejoué → jamais
// re-tenté in-sync). À appeler en BASSE fréquence (cron catalogue). Best-effort ;
// ExistsFresh rend les balayages suivants ~no-op. p/metaDB/sharedDB nil → no-op.
func ResolveUnresolvedAssetNames(ctx context.Context, p pool.Pool, metaDB, sharedDB *sql.DB, titleSlug string) assetnames.Result {
	var zero assetnames.Result
	if p == nil || metaDB == nil || sharedDB == nil {
		return zero
	}
	var refs []assetnames.AssetRef
	for _, c := range unresolvedAssetColumns {
		rs, err := collectUnresolvedRefs(ctx, sharedDB, c.kind, c.idCol, c.nameCol, c.verCol)
		if err != nil {
			slog.WarnContext(ctx, "asset sweep: collecte échouée", "asset_type", c.kind, "err", err)
			continue
		}
		refs = append(refs, rs...)
	}
	return resolveRefsWithPool(ctx, p, metaDB, titleSlug, "sweep", refs, assetSweepMaxAssets)
}

// resolveRefsWithPool est l'UNIQUE point d'acquisition de token + résolution.
// Acquiert un token du POOL unifié (PolicyAnyPublic, comme le watcher/drain),
// construit le fetcher authentifié, résout, libère le lease. Gaté par le
// kill-switch LEVELUP_SYNC_RESOLVE_ASSETS. logCtx identifie la source dans les
// logs (gamertag V1, "v2_cycle", "sweep"). maxAssets 0 → défaut du résolveur.
func resolveRefsWithPool(
	ctx context.Context,
	p pool.Pool,
	metaDB *sql.DB,
	titleSlug, logCtx string,
	refs []assetnames.AssetRef,
	maxAssets int,
) assetnames.Result {
	var res assetnames.Result
	if p == nil || metaDB == nil || len(refs) == 0 || !halo.AssetNameResolutionEnabled() {
		return res
	}
	lease, err := p.Acquire(ctx, pool.PolicyAnyPublic, "")
	if err != nil || lease == nil || lease.Tokens == nil {
		slog.WarnContext(ctx, "sync: résolution noms d'assets — token pool indisponible",
			"source", logCtx, "err", err)
		if lease != nil {
			lease.Release()
		}
		return res
	}
	defer lease.Release()
	return resolveRefs(ctx, halo.NewAssetNameFetcher(halo.AssetNameResolveRateLimit, lease.Tokens),
		metaDB, titleSlug, logCtx, refs, maxAssets)
}

// resolveRefs est le cœur TESTABLE de la résolution (fetcher injecté) :
// assetnames.Resolve (dédup + cap + skip-fresh + écriture asset_translations) +
// logging. Séparé de resolveRefsWithPool pour permettre les tests avec un fetcher
// factice (sans pool ni réseau).
func resolveRefs(
	ctx context.Context,
	fetcher assetnames.Fetcher,
	metaDB *sql.DB,
	titleSlug, logCtx string,
	refs []assetnames.AssetRef,
	maxAssets int,
) assetnames.Result {
	var res assetnames.Result
	if fetcher == nil || metaDB == nil || len(refs) == 0 {
		return res
	}
	// discovery-infiniteugc EXIGE version_id. Les refs sans version (typiquement des
	// paires dont match_registry ne porte pas la version) ne sont pas fetchables ici :
	// elles nécessitent l'expansion playlist (version via RotationEntries). On les
	// écarte proprement — ce n'est pas une erreur réseau — en le signalant en agrégat
	// (règle no-silent-caps).
	var withVersion []assetnames.AssetRef
	noVersion := 0
	for _, r := range refs {
		if strings.TrimSpace(r.VersionID) == "" {
			noVersion++
			continue
		}
		withVersion = append(withVersion, r)
	}
	if noVersion > 0 {
		slog.InfoContext(ctx, "sync: refs d'assets sans version_id ignorées (non fetchables — expansion playlist requise)",
			"source", logCtx, "title_slug", titleSlug, "count", noVersion)
	}
	if len(withVersion) == 0 {
		return res
	}
	// TitleID est ignoré par le fetcher discovery-infiniteugc (préfixe jeu "hi" fixe
	// dans l'URL) ; on passe titleSlug par cohérence (logging / multi-titre futur).
	res, _ = assetnames.Resolve(ctx, fetcher, opsAssetStore{db: metaDB}, withVersion,
		assetnames.Config{TitleID: titleSlug, MaxAssets: maxAssets})
	if res.Requested > 0 {
		slog.InfoContext(ctx, "sync: résolution noms d'assets terminée",
			"source", logCtx, "title_slug", titleSlug,
			"requested", res.Requested, "resolved", res.Resolved,
			"skipped", res.Skipped, "capped", res.Capped, "errors", res.Errors)
	}
	if res.Capped > 0 {
		slog.WarnContext(ctx, "sync: résolution noms d'assets — cap atteint (restants repris au prochain cycle)",
			"source", logCtx, "capped", res.Capped)
	}
	return res
}

// collectAssetRefsFromRegistry extrait les tuples (type, id, version) d'une row
// registry, limités aux assets dont le nom n'est PAS encore résolu (nil, vide ou
// == id). Pur, zéro DB.
func collectAssetRefsFromRegistry(reg *MatchRegistryRow) []assetnames.AssetRef {
	if reg == nil {
		return nil
	}
	var out []assetnames.AssetRef
	add := func(kind string, id, name, version *string) {
		if id == nil {
			return
		}
		assetID := strings.TrimSpace(*id)
		if assetID == "" || !needsRegistryNameOverride(name, assetID) {
			return
		}
		ref := assetnames.AssetRef{AssetType: kind, AssetID: assetID}
		if version != nil {
			ref.VersionID = strings.TrimSpace(*version)
		}
		out = append(out, ref)
	}
	add(games.AssetKindPlaylist, reg.PlaylistID, reg.PlaylistName, reg.PlaylistVersionID)
	add(games.AssetKindMap, reg.MapID, reg.MapName, reg.MapVersionID)
	add(games.AssetKindPair, reg.PairID, reg.PairName, reg.PairVersionID)
	add(games.AssetKindGameVariant, reg.GameVariantID, reg.GameVariantName, reg.GameVariantVersionID)
	return out
}

// unresolvedAssetColumns : colonnes (id, name, version) de match_registry par type
// d'asset, pour le balayage des assets restés en UUID (name == id).
var unresolvedAssetColumns = []struct{ kind, idCol, nameCol, verCol string }{
	{games.AssetKindPlaylist, "playlist_id", "playlist_name", "playlist_version_id"},
	{games.AssetKindMap, "map_id", "map_name", "map_version_id"},
	{games.AssetKindPair, "pair_id", "pair_name", "pair_version_id"},
	{games.AssetKindGameVariant, "game_variant_id", "game_variant_name", "game_variant_version_id"},
}

// collectUnresolvedRefs liste les (asset_id, version_id) DISTINCTS d'un type dont
// le nom est resté en UUID (name == id ou NULL). Fallback sans version_id si la
// colonne n'existe pas (schéma ancien). Pur (lecture seule).
func collectUnresolvedRefs(ctx context.Context, sharedDB *sql.DB, kind, idCol, nameCol, verCol string) ([]assetnames.AssetRef, error) {
	q := fmt.Sprintf(
		`SELECT DISTINCT %s, COALESCE(%s, '') FROM match_registry
		 WHERE %s IS NOT NULL AND %s <> '' AND (%s IS NULL OR %s = %s)`,
		idCol, verCol, idCol, idCol, nameCol, nameCol, idCol)
	rows, err := sharedDB.QueryContext(ctx, q)
	if err != nil {
		// Fallback sans la colonne version_id.
		q2 := fmt.Sprintf(
			`SELECT DISTINCT %s, '' FROM match_registry
			 WHERE %s IS NOT NULL AND %s <> '' AND (%s IS NULL OR %s = %s)`,
			idCol, idCol, idCol, nameCol, nameCol, idCol)
		rows, err = sharedDB.QueryContext(ctx, q2)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	var out []assetnames.AssetRef
	for rows.Next() {
		var id, ver string
		if scanErr := rows.Scan(&id, &ver); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, assetnames.AssetRef{AssetType: kind, AssetID: strings.TrimSpace(id), VersionID: strings.TrimSpace(ver)})
	}
	return out, rows.Err()
}

// opsAssetStore adapte le dictionnaire asset_translations à assetnames.Store.
// Écrit via ops.UpsertAssetTranslation (ART-safe) ; lit la présence via un
// SELECT pur. Le handle est le metadata RW partagé (cf. engine.run).
type opsAssetStore struct{ db *sql.DB }

// ExistsFresh indique qu'une traduction utilisable (nom non vide) existe déjà.
// Table absente (fixtures de test) → traitée comme « pas présente » sans erreur.
func (s opsAssetStore) ExistsFresh(ctx context.Context, assetType, assetID, lang string) (bool, error) {
	if s.db == nil {
		return false, nil
	}
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM asset_translations
		WHERE asset_type = ? AND asset_id = ? AND lang = ?
		  AND name IS NOT NULL AND TRIM(name) <> ''
		LIMIT 1`, assetType, assetID, lang).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		if isMissingTableErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Upsert écrit asset_translations via le helper ART-safe SELECT-then-write.
// Logge chaque résolution réussie en DEBUG (diagnostic), routé vers logs/sync.log.
func (s opsAssetStore) Upsert(ctx context.Context, assetType, assetID, lang, name string) error {
	if s.db == nil {
		return nil
	}
	if _, err := ops.UpsertAssetTranslation(ctx, s.db, assetType, assetID, lang, name); err != nil {
		return err
	}
	slog.DebugContext(ctx, "asset name résolu",
		"asset_type", assetType, "asset_id", assetID, "lang", lang, "name", name)
	return nil
}

// isMissingTableErr reconnaît l'absence de table (fixtures DuckDB :memory:).
func isMissingTableErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "Table with name")
}
