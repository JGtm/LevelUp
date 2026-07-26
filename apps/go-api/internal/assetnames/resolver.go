// Package assetnames — résolution ciblée des noms d'assets Halo dans le
// dictionnaire metadata.asset_translations, intégrée au sync réel.
//
// Problème adressé : asset_translations (lu par tout l'affichage end-user ET par
// EnrichRegistryFromMetadata au sync) n'était alimentée par AUCUN mécanisme
// runtime — seulement par le CLI manuel populate-assets, les seeds et l'action
// admin. Le contenu neuf n'avait donc jamais de nom localisé → UUID brut partout.
//
// Ce package résout une LISTE CIBLÉE d'assets (ceux vus dans un cycle de sync)
// vers asset_translations, best-effort et convergent. Il est découplé : les
// dépendances concrètes (Fetcher = halo.HaloProvider.FetchAsset, Store =
// ops.UpsertAssetTranslation) sont injectées via interfaces → testable sans
// réseau ni DuckDB. Le câblage concret vit dans le package sync
// (assetnames_wiring.go). Ne duplique pas cmd/populate-assets (batch full
// 14 langues) : c'est le chemin in-cycle minimal.
//
// Cf. plan « Résolution autonome des noms d'assets Halo » (2026-06-13).
package assetnames

import (
	"context"
	"strings"
)

// DefaultLangs : langues résolues par défaut. Seules fr-FR et en-US sont lues par
// le read-path (cf. PreferredLangsForLocale → [fr-FR,fr,en-US,en]) ; les résoudre
// suffit et borne le coût réseau hot-path.
var DefaultLangs = []string{"fr-FR", "en-US"}

// defaultMaxAssets : cap dur d'assets résolus par appel (garde-fou cold-start :
// 1er cycle après un déploiement où beaucoup d'assets sont neufs). Les assets
// écartés restent absents → repris au cycle suivant (convergent).
const defaultMaxAssets = 64

// AssetRef identifie un asset à résoudre. VersionID peut être "" → l'API prend
// la dernière version.
type AssetRef struct {
	AssetType string // games.AssetKind* (playlist|map|pair|game_variant)
	AssetID   string
	VersionID string
}

// Fetcher récupère le nom localisé d'un asset (1 appel réseau par langue).
// Implémenté par un adapter sur halo.HaloProvider.FetchAsset : API discovery-infiniteugc
// AUTHENTIFIÉE (Spartan token + 343-clearance, token du pool unifié) — ni publique,
// ni GameCMS. Sonde anonyme 2026-07-25 : 401. Cf. halo/asset_name_fetcher.go.
type Fetcher interface {
	FetchName(ctx context.Context, assetType, titleID, assetID, versionID, lang string) (string, error)
}

// Store lit la présence d'une traduction utilisable et écrit asset_translations.
// Implémenté par un adapter sur ops.UpsertAssetTranslation (ART-safe :
// SELECT-then-write, jamais d'ON CONFLICT).
type Store interface {
	// ExistsFresh indique qu'une traduction utilisable (nom non vide) existe déjà
	// pour (assetType, assetID, lang) → on évite l'appel réseau.
	ExistsFresh(ctx context.Context, assetType, assetID, lang string) (bool, error)
	Upsert(ctx context.Context, assetType, assetID, lang, name string) error
}

// Config paramètre une résolution.
type Config struct {
	TitleID   string   // titleID DiscoveryUGC (ex "hi" pour Halo Infinite) — fourni par le titre
	Langs     []string // défaut DefaultLangs
	MaxAssets int      // défaut defaultMaxAssets
}

// Result agrège les compteurs d'une résolution (logging + expvar).
type Result struct {
	Requested int // assets uniques soumis (après dedup, avant cap)
	Resolved  int // assets dont ≥1 langue a été fetchée puis upsertée
	Skipped   int // assets déjà présents (aucun fetch)
	Capped    int // assets écartés par le cap (repris au cycle suivant)
	Errors    int // erreurs fetch/upsert (niveau langue)
}

// Resolve résout une liste ciblée d'assets vers asset_translations. Best-effort,
// idempotent, convergent : un asset en échec (ou écarté par le cap) reste absent
// du dictionnaire → naturellement repris au cycle suivant. Ne retourne pas
// d'erreur fatale (la résolution des noms ne doit jamais casser le sync).
func Resolve(ctx context.Context, f Fetcher, s Store, refs []AssetRef, cfg Config) (Result, error) {
	var res Result
	if f == nil || s == nil || len(refs) == 0 {
		return res, nil
	}
	langs := cfg.Langs
	if len(langs) == 0 {
		langs = DefaultLangs
	}
	maxAssets := cfg.MaxAssets
	if maxAssets <= 0 {
		maxAssets = defaultMaxAssets
	}

	deduped := dedup(refs)
	res.Requested = len(deduped)
	kept, capped := applyCap(deduped, maxAssets)
	res.Capped = capped

	for _, ref := range kept {
		if ctx.Err() != nil {
			break
		}
		resolved, skipped, errCount := resolveOne(ctx, f, s, ref, cfg.TitleID, langs)
		res.Errors += errCount
		switch {
		case resolved:
			res.Resolved++
		case skipped:
			res.Skipped++
		}
	}
	return res, nil
}

// dedup réduit les refs à une par (assetType, assetID), gardant la première
// occurrence (donc son VersionID). Ignore les refs à type ou id vide.
func dedup(refs []AssetRef) []AssetRef {
	seen := make(map[string]struct{}, len(refs))
	out := make([]AssetRef, 0, len(refs))
	for _, r := range refs {
		at := strings.TrimSpace(r.AssetType)
		id := strings.TrimSpace(r.AssetID)
		if at == "" || id == "" {
			continue
		}
		key := at + "|" + id
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, AssetRef{AssetType: at, AssetID: id, VersionID: strings.TrimSpace(r.VersionID)})
	}
	return out
}

// applyCap borne le nombre d'assets traités. Retourne (gardés, nombre écarté).
func applyCap(refs []AssetRef, max int) ([]AssetRef, int) {
	if len(refs) <= max {
		return refs, 0
	}
	return refs[:max], len(refs) - max
}
