# Plan d'implémentation : CLI populate-assets en Go

## Contexte

Remplacement du script Python `populate_asset_translations.py` (417 lignes) par un CLI Go réutilisant l'infrastructure existante du projet.

**Objectif** : Peupler `asset_translations` dans `metadata.duckdb` avec les noms localisés des assets (maps, playlists, pairs, game_variants) en 14 langues BCP-47.

## Architecture cible

```
apps/go-api/
  cmd/
    populate-assets/           # Nouveau CLI
      main.go                  # Entry point
  internal/
    platform/
      halo/
        discovery_client.go    # Nouveau : client API Discovery UGC
        discovery_types.go     # Types Asset, AssetType constants
    domain/
      asset_langs.go           # Constantes 14 langues BCP-47
```

## Tâches d'implémentation

### T1 : Constantes langues (domain/asset_langs.go)

```go
package domain

// BCP-47 language codes for Halo Infinite multilingual assets
var TargetLanguages = []string{
    "en-US", "fr-FR", "de-DE", "es-ES", "es-MX",
    "it-IT", "ja-JP", "ko-KR", "pt-BR", "zh-CN",
    "zh-TW", "nl-NL", "pl-PL", "ru-RU",
}

const DefaultLanguage = "en-US"
```

### T2 : Client Discovery UGC (platform/halo/discovery_client.go)

**Endpoint pattern** :
```
GET https://gamecms-hacs.svc.halowaypoint.com/hi/multiplayer/file/{asset_type}/{title_id}/{asset_id}/{version_id}
Header: Accept-Language: {lang}
```

**Types d'assets** :
- Maps → `map-variants`
- Playlists → `playlists`
- PlaylistMapModePairs → `playlist-map-mode-pairs`
- GameVariants → `ugc-game-variants`

**Méthodes à créer** :
```go
// FetchAsset récupère un asset avec Accept-Language header
func (p *HaloProvider) FetchAsset(
    ctx context.Context,
    assetType string,
    titleID string,
    assetID string,
    versionID string,
    lang string,
) (*DiscoveryAsset, error)

// FetchMatchStats récupère les stats d'un match (pour extraire version_ids)
func (p *HaloProvider) FetchMatchStats(
    ctx context.Context,
    matchID string,
) (map[string]interface{}, error)
```

**Structures** :
```go
type DiscoveryAsset struct {
    AssetID     string `json:"AssetId"`
    VersionID   string `json:"VersionId"`
    PublicName  string `json:"PublicName"`
    Description string `json:"Description,omitempty"`
}

type MatchInfoAsset struct {
    AssetID   string `json:"AssetId"`
    VersionID string `json:"VersionId"`
}
```

### T3 : Repository métadonnées (ajout dans metadata_repo.go)

**Méthodes à ajouter** :
```go
// GetDistinctAssetIDs retourne les asset_ids distincts depuis match_registry
func (r *MetadataRepo) GetDistinctAssetIDs(
    ctx context.Context,
    assetType string,
    sharedDB *DB,
) ([]string, error)

// GetExistingTranslations retourne les (asset_id, lang) déjà présents et frais
func (r *MetadataRepo) GetExistingTranslations(
    ctx context.Context,
    assetType string,
    lang string,
    freshnessDays int,
) (map[string]bool, error)

// UpsertAssetTranslation insère ou met à jour une traduction
func (r *MetadataRepo) UpsertAssetTranslation(
    ctx context.Context,
    assetID string,
    assetType string,
    lang string,
    name string,
    description string,
) error
```

### T4 : CLI main.go

**Structure** :
```go
package main

import (
    "context"
    "flag"
    "fmt"
    "sync"

    "levelup/go-api/internal/config"
    "levelup/go-api/internal/domain"
    "levelup/go-api/internal/platform/duckdb"
    "levelup/go-api/internal/platform/halo"
)

func main() {
    // Flags : --types, --langs, --dry-run, --force, --concurrency
    // 1. Ouvrir metadata.duckdb + shared_matches_v2.duckdb
    // 2. Pour chaque asset_type : récupérer asset_ids distincts
    // 3. Construire cache version_id (via match_stats API)
    // 4. Pour chaque langue : fetch en parallèle (goroutines + semaphore)
    // 5. Upsert dans asset_translations
    // 6. Rapport final
}
```

**Pipeline de traitement** :
1. **Scan** : `GetDistinctAssetIDs` pour chaque type
2. **Version cache** : Fetch représentatif par asset via `FetchMatchStats`
3. **Parallélisme** : Goroutines avec semaphore (concurrency=10 par langue)
4. **Upsert** : Transaction par batch (ex: 50 rows) pour éviter locks
5. **Progress** : Log toutes les 100 assets fetchées

### T5 : Tests unitaires

**Tests à créer** :
- `discovery_client_test.go` : Mock HTTP responses
- `metadata_repo_asset_test.go` : Tests GetDistinctAssetIDs, UpsertAssetTranslation
- `populate_assets_integration_test.go` : Test end-to-end avec DB mémoire

## Dépendances externes

**Aucune** : Réutilise 100% de l'infrastructure existante :
- `duckdb.DB`, `duckdb.MetadataRepo`
- `halo.HaloProvider` (rate limiter, retry, HTTP client)
- `config.AppConfig` (RepoRoot, paths)
- `domain` types

## Avantages vs Python

1. **Typage fort** : Détection erreurs à la compilation
2. **Performance** : Goroutines natives (vs asyncio overhead)
3. **Maintenance** : Code Go unifié, pas de dépendance Python
4. **Tests** : Mocking HTTP facile via `httptest`
5. **Déploiement** : Binaire standalone (pas de venv)

## Estimation

| Tâche | Lignes Go | Temps |
|-------|-----------|-------|
| T1: Constantes langues | ~20L | 5min |
| T2: Discovery client | ~150L | 1h |
| T3: Repo métadonnées | ~100L | 45min |
| T4: CLI main | ~200L | 1h30 |
| T5: Tests | ~150L | 1h |
| **Total** | **~620L** | **4h15** |

## Ordre d'exécution

1. T1 (constantes) → T2 (client API) → T3 (repo) → T4 (CLI) → T5 (tests)
2. Tester avec `--dry-run --types map --langs fr-FR` (scope réduit)
3. Exécuter full : `populate-assets --types map,playlist,pair,game_variant`
4. Vérifier dans metadata.duckdb : `SELECT COUNT(*), lang FROM asset_translations GROUP BY lang`

## Notes techniques

### Rate limiting
- Réutiliser `HaloProvider.rateLimiter` (60 req/min)
- Pas de burst : wait systématique avant chaque requête

### Version IDs
- Cache global `map[string]map[string]string` : `asset_type → asset_id → version_id`
- Construire en 1 passe via batch de `FetchMatchStats` (dédupliqué)
- Match stats JSON : `MatchInfo.{MapVariant, Playlist, PlaylistMapModePair, UgcGameVariant}.VersionId`

### Erreurs API
- 404 sur asset : log warning + skip (asset supprimé par 343)
- 5xx : retry 3× avec backoff exponentiel (déjà géré par HaloProvider)
- Network timeout : configurable via flag `--timeout` (défaut 15s)

### Parallélisme
- Semaphore global : `--concurrency 10` (défaut)
- Goroutines par langue ET par asset (crossproduct)
- Channel pour collecter résultats → writer thread upsert DB

### Dry-run
- DB en mémoire (`:memory:`)
- Afficher samples : "Would upsert: map {asset_id} [fr-FR] → {PublicName}"
- Compter total sans écrire

## Commandes finales

```bash
# Compilation
cd apps/go-api
go build -o ../../bin/populate-assets ./cmd/populate-assets

# Dry-run (test)
./bin/populate-assets --dry-run --types map --langs fr-FR

# Exécution maps uniquement (phase initiale)
./bin/populate-assets --types map

# Full (tous assets, toutes langues)
./bin/populate-assets

# Force re-fetch (ignorer fraîcheur)
./bin/populate-assets --force

# Concurrency réduite (si rate limit)
./bin/populate-assets --concurrency 5
```

## Intégration avec Phase 3

**Après T4 (CLI fonctionnel)** :
- Task 6 : Exécuter `populate-assets --types map` → 14×N traductions
- Task 7 : Créer `migrate_static_maps_to_registry.go` (résolution name→id via asset_translations)
- Task 8+ : Enrichir builders Go avec `map_image_url`
