// Package sync — assetnames_wiring.go : câblage de la résolution autonome des
// noms d'assets (internal/assetnames) au moteur de sync.
//
// Pré-pass primary-write : pour les assets neufs vus dans un cycle, on peuple
// metadata.asset_translations (le dictionnaire lu par tout l'affichage end-user
// ET par EnrichRegistryFromMetadata) AVANT l'écriture registry, de sorte que les
// noms soient résolus dès le premier passage — sans heal, sans backfill de masse,
// sans action admin. Cf. plan « Résolution autonome des noms d'assets Halo »
// (2026-06-13).
//
// Écriture via ops.UpsertAssetTranslation (SELECT-then-write ART-safe, JAMAIS
// d'ON CONFLICT) sur le handle metadata RW PARTAGÉ (e.metaDB obtenu via
// OpenReadForQuery dans engine.run — il réutilise le handle RW de main.go au
// lieu d'ouvrir un 2e handle). Le Fetcher (halo.NewAssetNameFetcher) est
// token-free (API publique GameCMS) et injecté par le wiring.
package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"levelup/go-api/internal/assetnames"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/ops"
)

// WithAssetNameResolution active la résolution autonome des noms d'assets au
// sync (pré-pass primary-write). fetcher = source des noms (token-free). nil →
// résolution désactivée (parité legacy). Le gating effectif (flag
// LEVELUP_SYNC_RESOLVE_ASSETS + capability titre) est décidé par le caller : il
// ne câble cette option que si la résolution doit tourner.
func (e *SyncEngine) WithAssetNameResolution(fetcher assetnames.Fetcher) *SyncEngine {
	e.assetFetcher = fetcher
	return e
}

// resolveCycleAssets résout les noms des assets neufs du cycle vers
// asset_translations, AVANT la phase d'insert (donc avant EnrichRegistryFromMetadata
// dans submitOrInsertMatch). Best-effort, convergent : un échec laisse l'asset
// absent → repris au prochain cycle. No-op si la résolution n'est pas câblée ou
// si le handle metadata n'est pas disponible.
func (e *SyncEngine) resolveCycleAssets(ctx context.Context, fetched []*fetchedMatch) {
	if e.assetFetcher == nil || e.metaDB == nil {
		return
	}
	var refs []assetnames.AssetRef
	for _, fm := range fetched {
		if fm == nil || fm.Registry == nil {
			continue
		}
		refs = append(refs, collectAssetRefsFromRegistry(fm.Registry)...)
	}
	resolveAssetRefs(ctx, e.assetFetcher, e.metaDB, e.titleSlug, e.gamertag, refs)
}

// ResolveAssetsFromStats résout les noms des assets neufs d'un lot de matchs
// (raw Stats JSON déjà fetchés par le pipeline V2) vers asset_translations,
// AVANT la construction/persistance des batches (primary write V2). Réutilise la
// même logique que le pré-pass V1 (resolveCycleAssets). Best-effort, convergent.
// metaDB = handle metadata RW partagé ; fetcher nil ou metaDB nil → no-op.
func ResolveAssetsFromStats(
	ctx context.Context,
	fetcher assetnames.Fetcher,
	metaDB *sql.DB,
	titleSlug string,
	statsList []map[string]any,
) {
	if fetcher == nil || metaDB == nil || len(statsList) == 0 {
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
	resolveAssetRefs(ctx, fetcher, metaDB, titleSlug, "v2_cycle", refs)
}

// resolveAssetRefs résout un lot de refs et logge le résultat. Cœur partagé
// V1 (resolveCycleAssets) / V2 (ResolveAssetsFromStats). logCtx identifie la
// source dans les logs (gamertag V1, "v2_cycle" V2).
func resolveAssetRefs(
	ctx context.Context,
	fetcher assetnames.Fetcher,
	metaDB *sql.DB,
	titleSlug, logCtx string,
	refs []assetnames.AssetRef,
) {
	if fetcher == nil || metaDB == nil || len(refs) == 0 {
		return
	}
	// titleID DiscoveryUGC = slug du titre (convention de cmd/populate-assets, le
	// populateur historique de asset_translations).
	res, _ := assetnames.Resolve(ctx, fetcher, opsAssetStore{db: metaDB}, refs,
		assetnames.Config{TitleID: titleSlug})
	if res.Resolved > 0 || res.Errors > 0 || res.Capped > 0 {
		slog.InfoContext(ctx, "sync: résolution noms d'assets terminée",
			"source", logCtx, "title_slug", titleSlug,
			"requested", res.Requested, "resolved", res.Resolved,
			"skipped", res.Skipped, "capped", res.Capped, "errors", res.Errors)
	}
	if res.Capped > 0 {
		slog.WarnContext(ctx, "sync: résolution noms d'assets — cap atteint (restants repris au prochain cycle)",
			"source", logCtx, "capped", res.Capped)
	}
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
func (s opsAssetStore) Upsert(ctx context.Context, assetType, assetID, lang, name string) error {
	if s.db == nil {
		return nil
	}
	_, err := ops.UpsertAssetTranslation(ctx, s.db, assetType, assetID, lang, name)
	return err
}

// isMissingTableErr reconnaît l'absence de table (fixtures DuckDB :memory:).
func isMissingTableErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "Table with name")
}
