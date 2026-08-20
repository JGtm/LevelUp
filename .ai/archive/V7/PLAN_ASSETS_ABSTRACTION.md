# V7 — Couche d'abstraction Assets (local-first → API)

> Document de plan détaillé. Statut : à implémenter.
> Branche cible : à créer (`feat/v7-assets-abstraction`).

---

## TL;DR

Aujourd'hui 4 domaines (médailles, maps, défis, battle pass) ré-implémentent
chacun le pattern « cherche local → fallback API → persiste ». Le code est
éparpillé entre :

- `apps/go-api/internal/api/handlers/assets.go` (médailles, maps, BP image)
- `apps/go-api/internal/platform/halo/challenges_details.go` (badges en data-URL base64)
- `apps/go-api/internal/platform/halo/battlepass_details.go` (track defs + précache fire-and-forget)
- `apps/go-api/internal/platform/duckdb/{medal,map,asset}_cache_repo.go`
- `apps/go-api/cmd/{populate-assets,migrate-static-maps}/main.go`

On crée **un seul package `apps/go-api/internal/assets/`** qui expose un
`Resolver` typé par *kind* (`medal-image`, `map-image`, `challenge-badge`,
`bp-track-image`, `bp-background`, `medal-meta`, `challenge-def`, `track-def`,
`asset-translation`).

Toute la logique « lookup local → fetch remote → persist → renvoyer » vit
derrière une seule interface. Les handlers, services et le `HaloProvider`
n'appellent **plus jamais** directement le filesystem, DuckDB ou GameCMS pour
un asset : ils passent par `assets.Resolver`.

### Phases livrables et indépendamment vérifiables

- **P0 — Fondation** : interfaces, types, registry, contrat URL unifié.
- **P1 — Sources locales** : `LocalFSStore` (binaire) + `DuckDBIndexStore` (index).
- **P2 — Source distante** : `GameCMSFetcher` + `DiscoveryUGCFetcher` avec singleflight.
- **P3 — Resolver, WriteQueue, Reconcile** : orchestration, observabilité, concurrence DuckDB.
- **P4 — Câblage images** : médailles, maps, défis (badges), battle pass (track + bg).
- **P5 — Câblage data** : challenge definitions, track definitions, asset_translations, medal metadata.
- **P6 — Câblage exhaustif & nettoyage** : suppression des chemins legacy.

Chaque phase : code + tests unitaires + tests d'intégration + logs `slog`
structurés + métriques Prom + entrée `.ai/thought_log.md`.

---

## Décisions actées

| Décision | Choix |
|----------|-------|
| Contrat sortie binaires | `GET /api/v1/assets/{kind}/{title}/{id}[?variant=&lang=]`, redirect 302 ou contenu direct. **Plus de data-URL base64.** |
| Périmètre | Images **et** données (track defs, challenge defs, asset_translations, medal metadata). |
| Layout | Nouveau package `internal/assets/`, isolé de `platform/halo`, `platform/duckdb`, `api/handlers`. |
| Frontend | Aucun changement de logique : il consomme déjà des URLs. Seul changement visible : les défis ne reçoivent plus de `data:image/png;base64`. |
| Stockage | **Binaire = FS** (toujours dispo), **DuckDB = index seul** (URL résolue, hash, fetched_at, JSON brut, traductions). |
| Concurrence DuckDB | Toutes les écritures DuckDB du process passent par un `WriteQueue` (1 goroutine writer par DB path). |
| Bricolage | Aucune phase intermédiaire avec double chemin : chaque phase supprime immédiatement le legacy qu'elle remplace. |

---

## Phase 0 — Fondation (`internal/assets/`)

### Fichiers à créer

| Fichier | Contenu |
|---------|---------|
| `internal/assets/doc.go` | Doc package. |
| `internal/assets/kinds.go` | Enum `Kind` (constantes typées). |
| `internal/assets/ref.go` | `Ref{Kind, TitleID, ID, Variant, Lang}` + `String()` (clé singleflight + logging). |
| `internal/assets/payload.go` | `Payload` discriminé : `BinaryPayload`, `URLPayload`, `JSONPayload`. `Source` enum. |
| `internal/assets/errors.go` | `ErrNotFound`, `ErrUpstreamUnavailable`, `ErrUnsupportedKind`, `ErrPersistFailed`. |
| `internal/assets/resolver.go` | Interface publique `Resolver`. |

### Types clés

```go
type Kind string

const (
    KindMedalImage             Kind = "medal-image"
    KindMapImage               Kind = "map-image"
    KindChallengeBadge         Kind = "challenge-badge"
    KindBPTrackImage           Kind = "bp-track-image"
    KindBPBackground           Kind = "bp-background"
    KindMedalMetadata          Kind = "medal-meta"
    KindChallengeDefinition    Kind = "challenge-def"
    KindRewardTrackDefinition  Kind = "track-def"
    KindAssetTranslation       Kind = "asset-translation"
)

type Ref struct {
    Kind    Kind
    TitleID string
    ID      string  // medal_id, map_id, challenge path slug, etc.
    Variant string  // "spritesheet", "background", asset_type, …
    Lang    string  // BCP-47 pour KindAssetTranslation
}

type Source uint8

const (
    SourceLocalFile Source = iota + 1
    SourceLocalDB
    SourceRemote
)

type Resolved struct {
    Payload   Payload
    Source    Source
    FetchedAt time.Time
    ETag      string
}

type Resolver interface {
    Get(ctx context.Context, ref Ref) (Resolved, error)
    Refresh(ctx context.Context, ref Ref) (Resolved, error)        // force le fetch remote
    Warm(ctx context.Context, refs ...Ref)                          // pré-cache async
    RegisterLocalFile(ctx context.Context, ref Ref, path string) error
    Close(ctx context.Context) error
}
```

### Contrat URL unifié

Tout asset binaire est exposé sous :

```
GET /api/v1/assets/{kind}/{title}/{id}[?variant=&lang=]
```

Réponses possibles :
- `302 Found` + `Location` (cas le plus fréquent : redirection vers FS local ou CDN).
- `200 OK` + `Content-Type` + bytes (si on doit servir directement).
- `404 Not Found` (référence invalide).
- `502 Bad Gateway` (cache miss + upstream injoignable).

Les payloads JSON sont retournés via les endpoints métier existants
(`/home`, `/match-view`, etc.) mais **peuplés par des services qui appellent
`Resolver.Get`**, jamais en lisant DuckDB directement.

### Tests P0

- `internal/assets/ref_test.go` — `String()`/parsing inverse, égalité.
- `internal/assets/payload_test.go` — discrimination des variantes.

---

## Phase 1 — Sources locales (séparation binaire / index)

### Principe directeur

Le **binaire est sur FS**, **DuckDB ne stocke que l'index**.
Conséquence cruciale : un échec d'écriture DuckDB ne perd **jamais** le binaire ;
au pire on re-paie un fetch des métadonnées la prochaine fois.

### Interfaces

```go
type BinaryStore interface {
    LookupBinary(ctx context.Context, ref Ref) (*BinaryPayload, error)  // (nil, nil) si miss
    PersistBinary(ctx context.Context, ref Ref, p BinaryPayload) error  // atomique : tmp+rename
    Path(ref Ref) string
}

type IndexStore interface {
    LookupIndex(ctx context.Context, ref Ref) (*IndexEntry, error)
    PersistIndex(ctx context.Context, ref Ref, e IndexEntry) error      // peut échouer sur lock
    Available(ctx context.Context) bool                                 // false si DB lockée/inaccessible
}
```

### Implémentations

#### `internal/assets/store_localfs.go` — `LocalFSStore` (BinaryStore)

- Lecture/écriture sous `data/cache/<kind>/<title>/<id>[.<variant>]`.
- Écriture **atomique** : `os.WriteFile` sur `<path>.tmp` puis `os.Rename`.
- Détection du content-type via signature (PNG/JPEG/JSON).
- Override `RootOverrides map[Kind]string` pour pointer `static/maps/`,
  `static/medals/`, etc. (legacy statique).
- **Aucune dépendance DuckDB** — toujours disponible.

#### `internal/assets/store_duckdb.go` — `DuckDBIndexStore` (IndexStore)

- Consolide les 3 repos existants (`medal_cache_repo`, `map_cache_repo`,
  `asset_cache_repo`) + tables `*_definitions` / `*_translations`.
- `Available()` ping la DB ; si erreur de lock → retourne `false`, le resolver
  bascule en mode FS-only sans bruit.
- `PersistIndex` n'écrit **jamais** directement : pousse dans le `WriteQueue`
  (cf. P3) ; le retour est immédiat.

### Tests P1

- `store_localfs_test.go` — round-trip Persist→Lookup, content-type, miss,
  écriture atomique (kill au milieu → pas de corruption).
- `store_duckdb_test.go` — fixtures pour chaque `Kind`, comportement
  `Available()` quand la DB est lockée.

---

## Phase 2 — Source distante (GameCMS + Discovery UGC)

### Fichiers

| Fichier | Rôle |
|---------|------|
| `internal/assets/fetcher.go` | Interface `Fetcher.Fetch(ctx, Ref) (Payload, error)`. |
| `internal/assets/fetcher_gamecms.go` | Implémente Fetch pour les kinds binaires GameCMS + JSON definitions. |
| `internal/assets/fetcher_discoveryugc.go` | `KindMapImage` et `KindAssetTranslation`. |
| `internal/assets/fetcher_chain.go` | Fallback médaille (URL individuelle 404 → spritesheet). |

### `GameCMSFetcher`

```go
type GameCMSFetcher struct {
    httpClient *http.Client
    tokens     TokenProvider          // anciennement BPTokenProvider
    baseURL    string                 // défaut: https://gamecms-hacs.svc.halowaypoint.com
}
```

Centralise **toutes les URLs aujourd'hui en dur** dans `assets.go`,
`challenges_details.go`, `battlepass_details.go`, `medal_provider.go`.

### `DiscoveryUGCFetcher`

Réutilise les méthodes existantes du `HaloProvider` via une **petite interface**
(pas d'import circulaire) :

```go
type discoveryClient interface {
    FetchAssetTranslation(ctx context.Context, t AssetType, id, lang string) (json.RawMessage, error)
    FetchMapImageURL(ctx context.Context, mapID string) (string, error)
}
```

### Tests P2

- `fetcher_gamecms_test.go` — table-driven par `Kind`, mock HTTP via `httptest`.
- `fetcher_discoveryugc_test.go` — idem.
- Cas 404, 5xx, body invalide → erreur typée `ErrUpstreamUnavailable`.

---

## Phase 3 — Resolver, WriteQueue, Reconcile, observabilité

### `DefaultResolver` — flux d'exécution

```
1. Lookup binaire FS                    → hit ? return SourceLocalFile.
2. Lookup index DuckDB (best-effort)    → si Available()=false ou erreur lock,
                                          on log Debug et on traite comme miss.
3. singleflight.Do(ref.String())        → fetcher.Fetch.
4. Persist FS (synchrone, atomique)     → échec = ErrPersistFailed (espace disque).
5. Enqueue write index (asynchrone)     → le retour HTTP n'attend jamais la DB.
6. return SourceRemote.
```

### `WriteQueue` — sérialisation des écritures DuckDB

```go
type WriteQueue struct {
    ch       chan writeJob   // capacité bornée (ex: 256)
    idxStore IndexStore
    backoff  retryBackoff    // 50ms → 2s, max 5 tentatives
}
```

- **Une goroutine writer par DB path** (typiquement `metadata.duckdb`).
- L'unique connexion RW (pool=1, déjà imposée par `db.go`) garantit qu'aucune
  concurrence interne ne se produit.
- Sur **erreur de lock** (DuckDB renvoie `Could not set lock` ou `database is
  locked`) → backoff exponentiel + retry. Au-delà du max → log Warn + métrique
  `assets_index_write_dropped_total{kind=…}`. Le binaire reste sur FS, donc le
  prochain Lookup re-tentera l'écriture index via `ReconcileWorker`.
- Si `WriteQueue.ch` est **plein** (saturation) → drop + log Warn + métrique
  `assets_index_write_overflow_total`. Aucune perte fonctionnelle.

### `ReconcileWorker`

- Périodique (configurable, défaut 5 min).
- Scanne le filesystem (`data/cache/<kind>/...`) et compare avec
  `idxStore.LookupIndex` ; pour chaque binaire sans entrée index → ré-enqueue
  un `writeJob` (recompute hash si besoin).
- Permet aussi à un CLI (`populate-assets`, `migrate-static-maps`) qui a écrit
  FS pendant que le serveur était down de voir ses entrées indexées au
  démarrage suivant.

### Politique de concurrence (résumé strict)

| Acteur | Lecture FS | Lecture DuckDB | Écriture FS | Écriture DuckDB |
|--------|------------|----------------|-------------|-----------------|
| Serveur HTTP runtime | directe (parallèle) | pool RO/RW partagé | directe (atomique tmp+rename) | **uniquement** via `WriteQueue` |
| CLI `populate-assets` | directe | directe | directe | **uniquement** via `WriteQueue` (même process) |
| CLI `migrate-static-maps` | directe | directe | directe | idem |
| Sync engine | directe | directe | directe | idem |

**Règle absolue** : jamais d'écriture DuckDB en dehors du `WriteQueue` du
process courant.

**Inter-process** (CLI vs serveur en même temps) : DuckDB rejette naturellement
le 2ᵉ ouvreur RW au démarrage. Détection : si `OpenReadWrite` échoue, le store
retourne `Available()=false`, le resolver sert depuis FS et le re-fetch met à
jour FS sans erreur ; à la fin de l'autre process, `ReconcileWorker` complète
l'index.

### Métriques

```go
type Metrics interface {
    IncHit(Kind, Source)
    IncMiss(Kind)
    IncFetchError(Kind)
    IncIndexUnavailable()
    IncIndexWriteDropped(Kind)
    IncIndexWriteOverflow()
    ObserveLatency(Kind, Source, time.Duration)
}
```

Implémentation Prom-friendly réutilisant les compteurs déjà en place dans
`internal/ops`.

### Logs structurés (obligatoires partout)

| Event | Niveau | Attrs |
|-------|--------|-------|
| `assets: lookup` | Debug | `kind`, `id` |
| `assets: cache_hit_fs` | Debug | `kind`, `latency_ms` |
| `assets: cache_hit_index` | Debug | `kind`, `latency_ms` |
| `assets: cache_miss` | Info | `kind`, `id` |
| `assets: index_unavailable` | Warn (throttlé 1/min) | `path`, `err` |
| `assets: fetch_ok` | Info | `kind`, `id`, `bytes`, `latency_ms`, `singleflight_shared` |
| `assets: fetch_error` | Warn | `kind`, `id`, `err` |
| `assets: persist_fs_failed` | Error (bloquant) | `kind`, `id`, `path`, `err` |
| `assets: persist_index_enqueued` | Debug | `kind`, `id` |
| `assets: persist_index_retry` | Warn | `attempt`, `err` |
| `assets: persist_index_dropped` | Warn (non bloquant) | `kind`, `id` |
| `assets: reconcile_run` | Info | `scanned`, `enqueued`, `duration_ms` |

### Wire-up

```go
// internal/assets/wire.go
func New(cfg AssetConfig) (*DefaultResolver, error)
```

Démarre le `WriteQueue` worker et le `ReconcileWorker`. `Close(ctx)` flush la
queue avec timeout.

### Tests P3

- `resolver_default_test.go` — fakes `BinaryStore`/`IndexStore`/`Fetcher` :
  - hit FS pur → pas d'appel fetcher ni index.
  - hit index sans FS → fetch + persist.
  - `idxStore.Available()=false` → fetch normal, persist enqueued, métrique
    `index_unavailable` incrémentée.
  - 100 goroutines concurrentes même `Ref` → fetcher appelé 1 seule fois.
  - `WriteQueue` plein → réponse OK, drop loggé, métrique overflow incrémentée.
- `write_queue_test.go` :
  - Sérialisation : 50 jobs concurrents → fake `IndexStore` assert
    `atomic.LoadInt32(&inFlight) <= 1`.
  - Lock conflict : fake renvoie 3 fois `errLock`, 4ᵉ OK → succès après retry.
  - Lock conflict permanent → drop après N tentatives, métrique incrémentée.
- `reconcile_test.go` — fixtures FS avec 3 fichiers, IndexStore vide → 3 enqueues.
- `resolver_metrics_test.go` — chaque chemin incrémente la bonne métrique.

---

## Phase 4 — Câblage des images (médailles, maps, défis, BP)

### Fichiers modifiés

#### `apps/go-api/internal/api/handlers/assets.go`

- Supprime `MedalImageRepo`, `MapImageRepo`, `medalSFGroup`, `mapSFGroup`,
  `bpSFGroup`, `fetchMedalImageURL`, `fetchMapImageURL`.
- `AssetHandler{resolver assets.Resolver}` unique.
- `GetMedalImage`, `GetMapImage`, `GetBattlePassImage` → 3 trampolines de ~15
  lignes : parse `chi.URLParam`, construit `assets.Ref`, appelle
  `resolver.Get`, puis `http.Redirect` ou `http.ServeContent` selon `Payload`.
- Nouvel endpoint générique `GET /api/v1/assets/{kind}/{title}/{id}` pour les
  nouveaux types (challenge badge, BP background) — remplace les data URLs.

#### `apps/go-api/internal/platform/halo/challenges_details.go`

- `fetchChallengeBadgeDataURL` → **supprimée**.
- `buildChallengeItem` reçoit un `assets.Resolver` injecté dans `HaloProvider` ;
  construit `item.ImageURL = "/api/v1/assets/challenge-badge/halo_infinite/" + slug`.
- Le `Resolver.Get` est appelé en amont (warm-up optionnel) ou à la première
  requête HTTP — **plus de base64 inline dans le JSON**.

#### `apps/go-api/internal/platform/halo/battlepass_details.go`

- `preCacheBPTrackImages` → supprimé, remplacé par `resolver.Warm(ctx, refs...)`.
- URLs des images dans la réponse JSON pointent sur
  `/api/v1/assets/bp-track-image/...`.

#### `apps/go-api/internal/platform/halo/provider.go`

- `HaloProvider` reçoit un `assets.Resolver` via `WithAssetResolver(...)`.
- Suppression de `battlepassMetaPath` et `challengeBadgeCacheDir` au profit du
  resolver.

### Tests P4

- Réécriture de `assets_test.go` autour d'un fake `Resolver` (pas de stub repo).
- `challenges_details_test.go` : assert `ImageURL` est désormais une URL
  relative `/api/v1/assets/challenge-badge/...`, plus jamais
  `data:image/png;base64,...`.
- `battlepass_details_test.go` : idem pour `BattlePassImage` et
  `BackgroundImage`.
- `apps/web/e2e/assets-flow.spec.ts` : page Home charge, les `<img>` de défis
  et de BP pointent sur `/api/v1/assets/...`, le réseau retourne 200/302, pas
  de 404.

---

## Phase 5 — Câblage des données (definitions, translations, medal meta)

### Fichiers modifiés

#### `apps/go-api/internal/platform/halo/challenges_details.go`

- `loadChallengeDefinitionFromMetadata` + `storeChallengeDefinitionInMetadata`
  + `fetchChallengeDefinition` → **fusionnés** en un appel
  `resolver.Get(KindChallengeDefinition)` qui renvoie un
  `JSONPayload{TypedValue: *challengeDefinitionRaw}`.

#### `apps/go-api/internal/platform/halo/battlepass_details.go`

- `loadTrackDefinitionFromMetadata` + `storeTrackDefinitionInMetadata` +
  `fetchRewardTrackDefinition` → idem (`KindRewardTrackDefinition`).

#### `apps/go-api/internal/platform/halo/medal_provider.go`

- `FetchMedalsMetadata` reste l'implémentation côté `Fetcher`, mais les
  consommateurs (`cmd/refresh-metadata`) passent par
  `resolver.Get(KindMedalMetadata)`.

#### `apps/go-api/cmd/populate-assets/main.go`

- `fetchAllLangs` n'appelle plus `provider.FetchAssetTranslation` directement ;
  construit un `assets.Ref{Kind: KindAssetTranslation, ID: assetID, Variant:
  assetType, Lang: lang}` et délègue.
- Le check de fraîcheur `--freshness` devient une option du `Resolver`
  (`Refresh(ctx, Ref)` qui force le fetch).

#### `apps/go-api/cmd/migrate-static-maps/main.go`

- `metaRepo.UpsertMapImageRegistry` → `resolver.RegisterLocalFile(ctx, Ref,
  localPath)`.
- Le scan filesystem reste, mais l'écriture passe par l'abstraction.

### Tests P5

- `resolver_json_test.go` : un kind JSON déserialise vers le bon type via un
  `Decoder` enregistré par `Kind`.
- Tests CLI `populate-assets` : vérifient que `--dry-run` n'écrit ni dans
  DuckDB ni sur disque, et que la sortie passe par le resolver (mock vérifie
  les appels).

---

## Phase 6 — Câblage exhaustif et nettoyage

### Suppressions strictes (zéro fallback parallèle)

- `internal/platform/duckdb/medal_cache_repo.go`, `map_cache_repo.go`,
  `asset_cache_repo.go` : déplacés intégralement derrière `DuckDBIndexStore`.
  Les méthodes `MetadataRepo.GetMedalImageCache/UpsertMedalImageCache` etc.
  sont **supprimées de l'API publique** du repo (ou marquées via un type
  non-exporté).
- `medalSFGroup`, `mapSFGroup`, `bpSFGroup` dans `handlers/assets.go` :
  supprimés (le singleflight est désormais centralisé dans `DefaultResolver`).
- `data/cache/challenge_badges/` et `data/cache/battlepass_assets/` :
  conservés comme stockage, mais le **chemin** n'est plus codé en dur dans
  `halo/*_details.go` — il vient de `AssetConfig.LocalRoot`.
- `WithBattlePassCache`, `BPTokenProvider` au niveau handler : remplacés par
  l'injection unique dans `assets.New(...)`.

### Audit exhaustif (commandes de vérification)

```bash
rg -n "data/cache/(challenge_badges|battlepass_assets)" apps/go-api
rg -n "fetchChallenge|fetchReward|fetchMedalImage|fetchMapImage" apps/go-api
rg -n "GetMedalImageCache|GetMapImageCache|UpsertMapImageCache" apps/go-api
rg -n "data:image/.*;base64" apps/go-api apps/web
```

Chaque commande doit renvoyer **0 résultats hors de `internal/assets/`** avant
la fin de P6.

### Tests P6

- `tests/contracttest/assets_contract_test.go` (nouveau) : tape sur un serveur
  HTTP réel monté avec un `Resolver` factice, valide tous les endpoints
  `/api/v1/assets/{kind}/...`.
- `tests/contracttest/no_legacy_paths_test.go` : utilise `go/parser` pour
  s'assurer qu'aucun fichier hors `internal/assets/` n'importe
  `medal_cache_repo`, `map_cache_repo` ou n'appelle
  `http.Get("https://gamecms-hacs.svc.halowaypoint.com/...")`.
- Couverture min package `internal/assets/` ≥ 85 % (cible ajoutée au
  `apps/go-api/Makefile`).
- `apps/web/e2e/assets-flow.spec.ts` étendu : 4 domaines (medal popup, match
  card map, home challenges, home BP) — toutes les images chargent en < 500 ms
  au second hit (cache local).

---

## Fichiers concernés (récap)

### Backend

- `apps/go-api/internal/assets/**` — **nouveau package complet**.
- `apps/go-api/internal/api/handlers/assets.go` — réécriture P4.
- `apps/go-api/internal/api/handlers/assets_test.go` — réécriture P4.
- `apps/go-api/internal/platform/halo/challenges_details.go` — P4 & P5
  (suppression `fetchChallengeBadgeDataURL`, fusion des fns de cache).
- `apps/go-api/internal/platform/halo/battlepass_details.go` — P4 & P5
  (suppression `preCacheBPTrackImages`, fusion `loadTrack…/storeTrack…`).
- `apps/go-api/internal/platform/halo/medal_provider.go` — P5 (utilisé comme
  Fetcher derrière le Resolver).
- `apps/go-api/internal/platform/halo/provider.go` — injection du resolver.
- `apps/go-api/internal/platform/duckdb/{medal,map,asset}_cache_repo.go` —
  encapsulés P6.
- `apps/go-api/internal/service/home_service.go` — supprime les logs ad hoc
  « cache miss → live » au profit du resolver.
- `apps/go-api/cmd/populate-assets/main.go` — P5.
- `apps/go-api/cmd/migrate-static-maps/main.go` — P5.
- `apps/go-api/cmd/server/main.go` — wire-up P3 (`assets.New(cfg)`).

### Frontend

- `apps/web/src/components/ui/match-card.tsx`
- `apps/web/src/features/home/HomeChallengesList.tsx`
- `apps/web/src/features/home/HomeBattlePassPanel.tsx`
- `apps/web/src/features/palmares/SeasonPassPage.tsx`

→ Consomment toujours `image_url`, mais ces URLs sont désormais toutes
servies par `/api/v1/assets/...`. **Aucune logique frontend à changer**, juste
vérifier en e2e.

---

## Vérification finale

1. `cd apps/go-api && go test ./internal/assets/... -race -cover` → tous verts,
   couverture ≥ 85 %.
2. `go test ./...` complet vert (handlers, services, contracttest).
3. `go vet ./...` + `golangci-lint run` propres sur `internal/assets/`.
4. Les 4 commandes `rg` du bloc « audit exhaustif » de la phase 6 retournent
   **0** résultat hors `internal/assets/`.
5. `cd apps/web && pnpm test` (Vitest) et `pnpm e2e -- --grep assets-flow`
   verts.
6. **Healthcheck manuel** : démarrer le serveur, charger Home + une vue match,
   inspecter `slog` :
   - `assets: cache_hit_fs` au 2ᵉ chargement.
   - `assets: fetch_ok` au 1ᵉʳ.
   - `singleflight_shared=true` lors d'un rafraîchissement parallèle.
7. **Smoke prod-like** :
   ```bash
   curl -I http://localhost:8080/api/v1/assets/medal-image/halo_infinite/3233952928
   ```
   → 302 ; second appel sub-50 ms.

---

## Hors périmètre

- Refonte du modèle DuckDB metadata (les tables existantes sont réutilisées).
- Migration des assets statiques `static/medals/`, `static/ranks/`,
  `static/weapons-assets/` qui ne suivent **pas** déjà le pattern
  local-first/API-fallback (à traiter dans un sprint dédié si besoin).
- Authentification / refresh des tokens Halo — déjà géré, le resolver reçoit
  un `TokenProvider`.

---

## Stratégie Git

- Branche unique : `feat/v7-assets-abstraction`.
- 1 commit par phase :
  - `feat(assets): P0 fondation (Kind, Ref, Payload, Resolver iface)`
  - `feat(assets): P1 stores locaux (FS binaire + DuckDB index)`
  - `feat(assets): P2 fetchers GameCMS + Discovery UGC`
  - `feat(assets): P3 resolver + WriteQueue + Reconcile + observabilité`
  - `refactor(assets): P4 câblage handlers et HaloProvider (images)`
  - `refactor(assets): P5 câblage definitions, translations, medal meta`
  - `refactor(assets): P6 nettoyage exhaustif des chemins legacy`
- Entrée `.ai/thought_log.md` à chaque commit.
