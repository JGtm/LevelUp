# Plan — Catalogue global Playlists / Pairs / Maps

> Analyse réalisée le 2026-04-30. Conception d'un référentiel global Halo Infinite
> pour accélérer la cascade de filtres et écrémer l'UI aux options réellement utiles.
> Ce document couvre uniquement le design data + sync — pas l'UI React (cible d'un sprint suivant).

---

## 1. Contexte

Le système de filtres actuel ([FilterOmnibar.tsx](apps/web/src/components/shell/FilterOmnibar.tsx) → [filters_service.go](apps/go-api/internal/service/filters_service.go)) recharge **tous les matchs du joueur** à chaque toggle de checkbox via [filters_repo.go](apps/go-api/internal/platform/duckdb/filters_repo.go) `LoadMatchesForFilters()`, puis recalcule la cascade Expérience → Playlists → Modes → Maps en mémoire Go. Aucun cache de la hiérarchie n'existe — le `staleTime: 5min` côté TanStack Query ne suffit pas car chaque interaction utilisateur invalide la requête.

Sur un gros historique, le scan de `match_participants` jointé à `match_registry` à chaque toggle devient le principal goulot perçu côté UI. La latence n'est pas dramatique (200-500ms typique), mais elle empêche le sentiment d'instantanéité attendu sur un widget de filtre.

### Insight clé

Les playlists Halo Infinite sont **au niveau du jeu**, pas au niveau du joueur. Sources :

- [Halo Waypoint — Multiplayer Playlists](https://support.halowaypoint.com/hc/en-us/articles/17920041655188-Halo-Infinite-Multiplayer-Playlists) : liste des ~25-30 playlists permanentes (Ranked Arena, Quick Play, BTB, Firefight, Action Sack, etc.) + rotations hebdomadaires Ranked
- [den.dev — Halo Infinite Playlist Weights](https://den.dev/blog/halo-infinite-playlist-weights/) : structure API `discovery-infiniteugc.svc.halowaypoint.com/hi/playlists/{asset_id}/versions/{version_id}` avec `CustomData.PlaylistEntries[].MapModePairAssetId` + `Metadata.Weight`

La hiérarchie est donc **stable, énumérable, partagée entre tous les joueurs**. Elle évolue à la cadence des saisons (~3 mois) plus quelques rotations Ranked hebdomadaires. C'est exactement le profil d'un référentiel — pas d'une mv par joueur.

---

## 2. Audit de l'existant

| Composant | État | Référence |
|---|---|---|
| `shared.match_registry` | Stocke déjà `playlist_id`, `pair_id`, `map_id`, `game_variant_id` (UUID Asset) + noms PublicName + i18n FR | [schema.go:97-129](apps/go-api/internal/sync/schema.go#L97-L129) |
| `version_id` par asset | **Absent** — ni dans `match_registry`, ni extrait par les transforms | [transforms.go:114-122](apps/go-api/internal/sync/transforms.go#L114-L122) |
| Provider DiscoveryUGC Go | Existe avec retry exponentiel (4 tentatives, base 800ms × 2) | [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) |
| Types `AssetTypePlaylist` / `AssetTypePair` | Définis et fonctionnels, jamais appelés depuis `internal/sync/` | [discovery_types.go](apps/go-api/internal/platform/halo/discovery_types.go) |
| Cache générique d'assets bruts | `metadata.waypoint_assets_raw(title_id, asset_id, version_id, raw_json, content_hash)` | [steps_metadata.go:170-189](apps/go-api/internal/migration/steps_metadata.go#L170-L189) |
| Traductions FR/EN d'assets | `metadata.asset_translations(asset_id, asset_type, lang, name)` | [steps_metadata.go:20-28](apps/go-api/internal/migration/steps_metadata.go#L20-L28) |
| Notion `experience` | Dérivée heuristiquement à chaque requête depuis `is_ranked` + `is_firefight` (3 buckets : `PVP non classé`, `PVP classé`, `PVE`) | [filters_service.go:18-19](apps/go-api/internal/service/filters_service.go#L18-L19) |
| Tables dédiées catalogue | **Aucune** — pas de `playlists_catalog`, `playlist_pair_links`, ni `map_mode_pair_definitions` | — |

### Conclusion d'audit

80% du terrain est préparé : asset IDs déjà stockés en match, provider DiscoveryUGC déjà opérationnel, cache générique disponible, point d'injection sync clairement identifiable. Il reste à modéliser la **relation** entre playlists et pairs, persister `version_id`, et ajouter une stratégie de refresh.

---

## 3. Stratégie cible

### 3.1 Catalogue global dans `metadata.duckdb`

Source de vérité du catalogue Halo, **partagée entre tous les joueurs**. Quatre tables (cf. §4 pour le schéma SQL).

### 3.2 Stratégie de refresh — pas de worker par sync

Les playlists changent rarement. Trois mécanismes complémentaires, du plus paresseux au plus actif :

1. **Lazy detection au sync (zéro fetch immédiat)** — Lors de l'ingestion d'un match, si `playlist_id` ou `pair_id` est absent du catalogue, on enqueue dans `catalog_fetch_queue` sans bloquer l'ingestion ni déclencher d'appel HTTP. Coût : un INSERT OR IGNORE.
2. **Bootstrap initial (one-shot CLI)** — Commande `populate-playlists-catalog` qui (a) seed la queue depuis `SELECT DISTINCT playlist_id, pair_id FROM shared.match_registry`, (b) drain la queue en appelant DiscoveryUGC, (c) persiste dans le catalogue. Couvre 100% de ce que les joueurs ont déjà vu sans liste manuelle à maintenir.
3. **Refresh mensuel (cron / job planifié)** — Une fois par mois, drain la queue accumulée par la détection lazy + re-fetch les `playlists_catalog` où `is_active = true` pour détecter les changements de `version_id` (rotations Ranked, mises à jour de weights). Marquer `is_active = false` les playlists non vues depuis N matchs / X mois (à calibrer).

**Pourquoi pas un worker à chaque sync ?**
- Les rotations Ranked changent au pire toutes les semaines, pas plus
- La très grande majorité des sync delta ne rencontrent **aucun nouvel asset_id** (le joueur rejoue les mêmes playlists)
- L'API DiscoveryUGC a un rate limiter — multiplier les appels par sync est un mauvais investissement
- La latence d'enqueue dans une table DuckDB est ~µs ; la latence d'un round-trip HTTP est ~100ms. Pas de débat.

### 3.3 Écrémage UX — réponse à « garder l'utile »

Avec le catalogue global en place, la requête de filtres devient un `LEFT JOIN catalogue ↔ matchs_du_joueur` qui permet trois modes d'affichage :

| Mode | Comportement | Cas d'usage |
|---|---|---|
| **Joué** (défaut) | `WHERE match_count > 0` — ne montre que les playlists/maps/modes touchés | 95% des interactions ; menu réduit de ~80 à ~10-15 options sur un joueur typique |
| **Tous** (toggle) | Catalogue complet, options grisées avec `match_count = 0` | Découverte, comparaison entre joueurs |
| **Compteurs visibles** | `Ranked Slayer (24)` à côté de chaque option | Guide visuel pour prioriser les filtres riches en données |

C'est le vrai gain UX, plus important que la perf brute.

---

## 4. Schéma proposé (`metadata.duckdb`)

```sql
-- Référentiel des playlists Halo Infinite
CREATE TABLE playlists_catalog (
  playlist_asset_id   UUID PRIMARY KEY,
  current_version_id  UUID,
  name_en             VARCHAR,
  name_fr             VARCHAR,
  experience          VARCHAR,        -- 'ranked' | 'social' | 'btb' | 'firefight' | 'action_sack' | 'limited_time' | 'custom_browser' | 'unknown'
  is_ranked           BOOLEAN,
  is_active           BOOLEAN DEFAULT TRUE,
  first_seen_at       TIMESTAMP,
  last_seen_at        TIMESTAMP,
  last_fetched_at     TIMESTAMP
);

-- Définitions canoniques des paires map+mode
CREATE TABLE map_mode_pair_definitions (
  pair_asset_id          UUID PRIMARY KEY,
  current_version_id     UUID,
  name_en                VARCHAR,
  name_fr                VARCHAR,
  map_asset_id           UUID,
  map_name               VARCHAR,
  game_variant_asset_id  UUID,
  mode_canonical         VARCHAR,     -- 'slayer' | 'ctf' | 'oddball' | 'koth' | 'strongholds' | 'extraction' | 'firefight_kotr' | 'fiesta' | ...
  last_fetched_at        TIMESTAMP
);

-- Relation N-N playlist ↔ pair, avec poids de tirage
CREATE TABLE playlist_pair_links (
  playlist_asset_id  UUID,
  pair_asset_id      UUID,
  weight             DOUBLE,           -- depuis CustomData.PlaylistEntries[].Metadata.Weight
  PRIMARY KEY (playlist_asset_id, pair_asset_id)
);

-- File d'attente du fetcher (pattern Kinds, drain mensuel)
CREATE TABLE catalog_fetch_queue (
  asset_type    VARCHAR,             -- 'playlist' | 'pair' | 'map'
  asset_id      UUID,
  version_id    UUID,                -- nullable si on ne connaît pas encore
  enqueued_at   TIMESTAMP,
  attempts      INTEGER DEFAULT 0,
  last_error    VARCHAR,
  PRIMARY KEY (asset_type, asset_id)
);
```

### Décisions de modélisation

- **Pas de table `map_canonical_definitions` séparée** : `map_asset_id` + `map_name` dans `map_mode_pair_definitions` suffisent. Une map n'a pas de hiérarchie propre indépendante du pair.
- **`experience` en VARCHAR + enum applicatif** : pas de table de référence séparée. La liste reste petite et stable, et l'enum vit dans le code Go pour bénéficier de la validation au compile-time.
- **`weight` est conservé** : pas critique pour le filtre, mais ouvre la porte à une future feature « probabilité de tomber sur cette map dans Quick Play ».
- **`current_version_id` au lieu d'une table d'historique** : on ne garde que la version active. Si un audit historique des versions est jamais demandé, `waypoint_assets_raw` garde déjà les snapshots bruts datés.

---

## 5. Plan d'implémentation par phases

### Phase A — Migration schéma metadata (1 commit)

- Ajouter une migration dans [steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) qui crée les 4 tables.
- Tests : migration appliquée deux fois sans erreur, schéma matches.

### Phase B — Extraction `version_id` au sync (1 commit)

- Étendre [transforms.go:89-164](apps/go-api/internal/sync/transforms.go#L89-L164) `ExtractRegistry()` pour extraire `Playlist.VersionId`, `PlaylistMapModePair.VersionId` depuis le payload Halo.
- Ajouter colonnes `playlist_version_id`, `pair_version_id` dans `match_registry` ([schema.go:97-129](apps/go-api/internal/sync/schema.go#L97-L129)).
- Migration de backfill (NULL acceptés, hydratés au prochain sync).
- Tests : `transforms_test.go` couvre extraction sur fixtures réelles.

### Phase C — Détection lazy + enqueue (1 commit)

- Hook après `ExtractRegistry()`, avant `InsertRegistryIfNotExists()` ([writes.go:22-65](apps/go-api/internal/sync/writes.go#L22-L65)) : vérifier si `playlist_id` et `pair_id` existent dans `playlists_catalog` / `map_mode_pair_definitions`. Si absents → INSERT OR IGNORE dans `catalog_fetch_queue`.
- Tests : un sync delta avec asset inconnu enqueue une ligne ; un sync delta avec asset connu n'enqueue rien ; pas de blocage de l'ingestion en cas d'erreur DB sur la queue.

### Phase D — Drain de la queue (1 commit)

- Service `CatalogFetcherService` qui :
  1. SELECT les lignes `catalog_fetch_queue` triées par `attempts` ASC, `enqueued_at` ASC
  2. Pour chaque playlist : appel [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) `FetchAsset(AssetTypePlaylist, ...)` → parse `CustomData.PlaylistEntries` → upsert `playlists_catalog` + `playlist_pair_links` + enqueue les pairs si inconnus
  3. Pour chaque pair : `FetchAsset(AssetTypePair, ...)` → upsert `map_mode_pair_definitions`
  4. Sur succès → DELETE de la queue ; sur erreur → `attempts++` + `last_error`
- Pas de worker auto à ce stade — exposé via CLI `drain-catalog-queue`.
- Tests : drain sur fixtures DiscoveryUGC mockées ; gestion 404 (asset disparu) → marquer `is_active = false` ; gestion erreur transitoire → réessayable.

### Phase E — CLI bootstrap one-shot (1 commit)

- Commande Go `populate-playlists-catalog` qui :
  1. Seed la queue depuis `SELECT DISTINCT playlist_id, pair_id FROM shared.match_registry WHERE playlist_id IS NOT NULL`
  2. Lance le drain
  3. Loggue les stats finales (X playlists, Y pairs, Z erreurs)
- Tests : sur une DB de test avec 5 matchs distincts, peuple correctement le catalogue.

### Phase F — Mapping `experience` (1 commit)

- Définir l'enum `Experience` dans `internal/games/canonical/` ou `internal/sync/`.
- Heuristique de classification depuis le nom de playlist + flags `is_ranked` / `is_firefight` (réutiliser la logique existante de [filters_service.go](apps/go-api/internal/service/filters_service.go) mais l'appliquer **au moment du fetch catalogue**, pas à chaque requête).
- Stocker la valeur calculée dans `playlists_catalog.experience`.
- Tests : 25-30 playlists permanentes mappées correctement (snapshot des noms officiels).

### Phase G — Migration `FiltersService` vers le catalogue (1 commit)

- Réécrire [filters_service.go](apps/go-api/internal/service/filters_service.go) `Resolve()` pour requêter `playlists_catalog` ⨝ `match_registry` au lieu de scanner `match_participants`.
- Ajouter le toggle `mode_only_played` (défaut `true`) dans la signature de l'endpoint.
- Tests : parité de comportement avec l'ancien service sur fixtures réelles ; perf (benchmark Go) pour mesurer le gain.

### Phase H — Refresh mensuel (1 commit, optionnel selon cadence)

- Soit cron OS (instructions doc dans `docs/`), soit goroutine dans [server.go](apps/go-api/internal/api/server.go) avec `time.Ticker(30*24*time.Hour)`.
- Drain de la queue + re-fetch des `is_active = true`.

---

## 6. Décisions ouvertes

| # | Question | Options |
|---|---|---|
| 1 | Critère de désactivation `is_active = false` | (a) jamais (manuel) ; (b) pas vue depuis 3 mois ; (c) pas vue depuis 6 mois |
| 2 | Mécanisme du refresh mensuel | (a) cron OS documenté ; (b) goroutine Go avec ticker ; (c) endpoint admin déclenchable |
| 3 | Faut-il exposer `weight` dans l'API React | Pas pour le filtre, mais pour une future page « stats par carte/mode » oui |
| 4 | Mapping `experience` : enum Go vs table SQL | Enum Go préféré (validation compile-time, peu d'entrées) — à confirmer |
| 5 | Faut-il garder un historique des `version_id` par playlist | Non au début (`waypoint_assets_raw` couvre l'audit forensique si besoin) |

---

## 7. Tests à prévoir

- **Unitaires Go** : extraction `version_id`, classification `experience`, parsing `CustomData.PlaylistEntries`
- **Intégration DuckDB** (`//go:build integration`) : migration applicable, INSERT/UPSERT, JOIN catalogue ↔ registry
- **Fixtures DiscoveryUGC** : capturer 2-3 réponses réelles (Quick Play, Ranked Arena, Firefight) dans `testdata/`
- **Vitest React** (Phase G+) : `useFiltersResolve` consomme bien le toggle `mode_only_played`
- **Benchmark perf** : `go test -bench=BenchmarkFiltersResolve` avant/après pour quantifier le gain

---

## 8. Hors scope (à reporter)

- Refonte UI du `FilterOmnibar` au-delà de l'ajout du toggle « Joué / Tous »
- Page de configuration admin du catalogue (édition manuelle d'`is_active`, etc.)
- Statistiques agrégées par playlist (matchs joués totaux, durée moyenne, etc.) — possible suite naturelle
- Intégration AMQP `lobby-hi.svc.halowaypoint.com` mentionnée dans l'article den.dev pour la liste autoritative des playlists actives — overkill pour le besoin actuel

---

## 9. Estimation ordre de grandeur

- Phases A à E : ~2-3 jours de dev focalisé (catalogue + bootstrap fonctionnels)
- Phases F à G : ~1-2 jours (migration FiltersService + tests parité)
- Phase H : ~0,5 jour
- Total raisonnable : **3-5 jours-homme** pour un sprint dédié

---

## 10. Références

- [FilterOmnibar.tsx](apps/web/src/components/shell/FilterOmnibar.tsx) — UI actuelle des filtres
- [globalFilterStore.ts](apps/web/src/stores/globalFilterStore.ts) — Zustand store
- [filters_service.go](apps/go-api/internal/service/filters_service.go) — Logique cascade actuelle
- [filters_repo.go](apps/go-api/internal/platform/duckdb/filters_repo.go) — Accès données filtres
- [discovery_client.go](apps/go-api/internal/platform/halo/discovery_client.go) — Provider Halo DiscoveryUGC
- [discovery_types.go](apps/go-api/internal/platform/halo/discovery_types.go) — Types AssetType
- [transforms.go](apps/go-api/internal/sync/transforms.go) — Extraction registry au sync
- [writes.go](apps/go-api/internal/sync/writes.go) — Persistance match_registry
- [schema.go](apps/go-api/internal/sync/schema.go) — Schéma `shared.match_registry`
- [steps_metadata.go](apps/go-api/internal/migration/steps_metadata.go) — Migrations metadata
- Pattern Kinds (référence d'inspiration) : [resolver_default.go](apps/go-api/internal/assets/resolver_default.go), [kinds.go](apps/go-api/internal/assets/kinds.go), [fetcher_chain.go](apps/go-api/internal/assets/fetcher_chain.go)
- [Halo Waypoint — Multiplayer Playlists](https://support.halowaypoint.com/hc/en-us/articles/17920041655188-Halo-Infinite-Multiplayer-Playlists)
- [den.dev — Halo Infinite Playlist Weights](https://den.dev/blog/halo-infinite-playlist-weights/)
