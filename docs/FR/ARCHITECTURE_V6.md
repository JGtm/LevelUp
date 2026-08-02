# Architecture LevelUp v6 — DuckDB Shared Matches + Assets i18n

> Version anglaise : [../ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md)

> **Version** : 6.3.0 — **Mise à jour** : 2026-04-12

LevelUp utilise une architecture DuckDB v6 fondée sur les **matchs partagés** et une **i18n centralisée via `asset_translations`** :

- `data/warehouse/shared_matches_v2.duckdb` — toutes les données de matchs, partagées entre tous les joueurs
- `data/warehouse/metadata.duckdb` — référentiels : noms d'assets (14 langues), armes, rangs de carrière, citations
- `data/players/{gamertag}/stats.duckdb` — enrichissements par joueur uniquement

## Bases de données

```text
data/
  warehouse/
    metadata.duckdb
    shared_matches_v2.duckdb
    shared_pve.duckdb
  players/
    {gamertag}/
      stats.duckdb
```

## Tables clés (vue d'ensemble)

### metadata.duckdb

- `asset_translations` : noms localisés pour les maps, playlists, paires et variantes de jeu — 14 langues BCP-47 (`en-US`, `fr-FR`, …) — **ajouté en v6.3** — peuplé par les migrations Go de metadata (`internal/games/halo_infinite/migrations/`) lors du seeding de la metadata
- `challenge_definitions` : définitions versionnées des défis Halo (`challenge_path` + `content_hash`) avec catégorie, difficulté, seuil et récompenses XP
- `challenge_translations` : titres et descriptions de défis localisés dans toutes les langues exposées par le CMS (BCP-47, fallback `en-US`)
- `weapon_labels` : weapon_id (filmshell UBIGINT) → `name_en`, `name_fr` — ajouté en v5.4
- `career_ranks` : définitions des paliers de rang
- `citation_mappings` : mappings médaille → citation
- `mode_name_tr` / `mode_*` : traductions des noms de modes de jeu (surcharges legacy, supplantées par `asset_translations` pour les noms de map/playlist/paire/variante)

### shared_matches_v2.duckdb

Tables principales :
- `match_registry` : une ligne par match
- `match_participants` : stats par joueur pour tous les matchs

Vues SQL (`ensure_resolution_views()`) :
- `v_match_full` : `match_registry` enrichi des noms i18n issus de `meta.asset_translations` — 8 LEFT JOIN (en-US + fr-FR × map/playlist/paire/variante). Colonnes : `map_name`, `map_name_fr`, `game_variant_name`, `game_variant_name_fr`, etc.
- `v_gamertag_lookup` : XUID → gamertag courant (FULL OUTER JOIN `xuid_aliases` + `match_participants` + `match_kill_events_latest`)

> `v_killer_victim_full` a été **supprimée le 2026-08-02** et n'est plus une vue garantie v6.
> Ses deux LEFT JOIN re-joignaient `v_gamertag_lookup` pour produire des colonnes qui portaient
> déjà ces noms-là, et elle ne « marchait » que par le renommage silencieux des homonymes par
> DuckDB. Son unique lecteur (match-view Q20) lit désormais la table directement et rend les
> mêmes six colonnes. Les paires tueur → victime vivent dans `killer_victim_pairs` (historique,
> toujours lue) et dans `match_kill_events` — cette dernière se lit UNIQUEMENT par sa vue
> `match_kill_events_latest` (ADR 0026).

> **Important** : `v_match_full` requiert que `metadata.duckdb` soit ATTACHée comme `meta` dans la même connexion pour que les JOIN i18n fonctionnent. `DuckDBRepository` s'en charge automatiquement.

### stats.duckdb (par joueur)

- `player_match_enrichment` : performance_score, session_id, etc.
- `challenge_snapshots` : historique append-only de l'état des défis par joueur (actif/complété/à venir, progression, XP, expiration), dédupliqué au changement d'état

## Import OpenSpartan

LevelUp accepte un upload SQLite ponctuel depuis [OpenSpartan Workshop](https://github.com/OpenSpartan) (tracker Halo communautaire de Den Delimarsky, crédité dans `docs/ACKNOWLEDGMENTS.md`), pour que les joueurs qui suivaient déjà leurs matchs là-bas avant de passer à LevelUp puissent rapatrier cet historique. L'import parse le fichier et écrit les matchs dans `shared_matches_v2.duckdb` via le même chemin `persist.SharedPersister` que la sync live — pas de SQL ad hoc. Code : `internal/openspartan/` (lecteur) + `internal/openspartan/mapper/` (mapping des lignes) + `internal/service/openspartan_import_service.go` (orchestration) + `internal/api/handlers/openspartan_import.go` (endpoint d'upload protégé par auth, désactivé en mode démo) + `OpenSpartanImportCard.tsx` (UI d'onboarding). Le nom vient du projet tiers dont l'app lit les données, pas d'un choix de nommage LevelUp.

---

## Architecture multi-titres (Sprint 44)

LevelUp prend en charge plusieurs titres de jeu via un **agencement de données title-aware**. Chaque titre dispose de son propre arbre de données isolé :

```text
data/
  titles/
    halo_infinite/          # titre par défaut
      warehouse/
        metadata.duckdb
        shared_matches_v2.duckdb
      players/
        {gamertag}/
          stats.duckdb
    halo_mcc/               # second titre (exemple)
      warehouse/
        metadata.duckdb
        shared_matches_v2.duckdb
      players/
        {gamertag}/
          stats.duckdb
  warehouse/                # agencement plat legacy (compatibilité ascendante)
  players/                  # agencement plat legacy (compatibilité ascendante)
```

### Composants clés

| Composant | Rôle |
|-----------|------|
| `TitleRegistry` | Registre en mémoire des titres connus (slug, nom, statut, capabilities) |
| `PathResolver` | Résout tous les chemins de fichiers relatifs à un slug de titre (`TitleDataDir`, `SharedDBPath`, `PlayerDBPath`, etc.) |
| Middleware `TitleExtractor` | Lit l'en-tête `X-LevelUp-Title` / la session / le fallback → injecte `title_slug` dans le contexte de requête |
| `db_profiles.json` v3 | Profils joueur scopés par titre : `{ "version": "3.0", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }` |

### Stratégie de routage

L'API utilise une sélection de titre **par en-tête** (`X-LevelUp-Title`). Les URLs restent inchangées (`/api/v1/players/{slug}/...`). Le middleware injecte le titre dans le contexte de requête, et tous les services en aval (PlayerResolver, ProfileService, etc.) l'utilisent pour scoper l'accès aux données.

### Frontend

Le `appShellStore` suit `currentTitleSlug` et fournit `switchTitle()` qui :
1. POST `/session/context` avec le nouveau titre
2. Ré-amorce l'application (re-bootstrap)
3. Réinitialise les caches scopés par joueur

Le client API envoie l'en-tête `X-LevelUp-Title` pour les titres non-par-défaut.

### Compatibilité ascendante

- `PathResolver` fournit des méthodes `Legacy*` (`LegacySharedDBPath`, `LegacyPlayerDir`, etc.) pour l'agencement plat `data/warehouse/`
- Les fichiers `db_profiles.json` v2.1 sont auto-détectés et lus comme des profils `halo_infinite` implicites
- `LoadPlayers()` sans filtre de titre retourne les joueurs de tous les titres

---

## Schéma de services canonique + adapters sémantiques (plan multi-titres Phases A–E)

Au-dessus de l'agencement de stockage title-aware, LevelUp expose un schéma de services canonique et deux adapters de titre par titre. Cela découple les services produit du schéma DuckDB par titre et des labels/unités par titre.

```text
HTTP handler → product service → games.Resolver
                                    ├─ Data(slug)     → games.TitleDataAdapter
                                    └─ Semantic(slug) → games.TitleSemanticAdapter
```

### Packages

| Package | Rôle |
|---------|------|
| `internal/games/canonical/` | Enum `FieldKey` (59 clés), enums (`Outcome`, `MatchType`, `RatingType`, `Bucket`, `GroupBy`), scopes (`StatsScope`, `TimeseriesQuery`, `CareerOptions`), types match/career/timeseries — tous stables, agnostiques, utilisés par les services |
| `internal/games/mappings/` | Loader TOML strict (`go-toml/v2`), validation (locales, formats, collisions de `display_order`, conversions d'unités), `FieldMappingSet`, registre |
| `internal/games/halo_infinite/` | Implémentation HI : `DataAdapter` (encapsule les repos existants), `SemanticAdapter` (encapsule `FieldMappingSet`), `AssetURLAdapter` (compose les URLs `/static/...`) |
| `internal/games/synthetic_title_b/` | Corpus de test synthétique, tests cross-titres isolés uniquement — jamais référencé par le code de production |
| `internal/games/{adapter,resolver}.go` | Interfaces `TitleDataAdapter` + `TitleSemanticAdapter` + `TitleAssetURLAdapter`, `StaticResolver` |
| `internal/assets/static/` | Composition pure d'URL/chemin pour `/static/{folder}/{titleSlug}/{id}{ext}` — aucune connaissance du titre, aucune I/O, tests table-driven |

### Mappings TOML (versionnés dans Git)

Trois fichiers TOML par titre sous `config/titles/{slug}/mappings/` :

```text
config/
  titles/
    halo_infinite/
      mappings/
        fields.toml           # 59 FieldKey × labels EN/FR + format + group + display_order
        assets.toml           # modes / challenge_tier / cadence / challenge_status / medal_tier / prestige_level
        outcomes.toml         # win / loss / tie / dnf — labels + color_token (design system)
    synthetic_title_b/
      mappings/
        fields.toml           # corpus de test synthétique
        assets.toml           # labels divergents pour les tests d'isolation cross-titres
        outcomes.toml         # labels divergents (Triomphe / Défaite / Match nul / Forfait)
```

`fields.toml` est obligatoire ; `assets.toml` et `outcomes.toml` sont optionnels (leur absence est silencieuse). Chaque TOML porte un `[meta].schema_version` (cf. `tools/mappings/CHANGELOG.md`).

Le boot `Registry.LoadFromConfigDir()` charge les trois fichiers par titre. Un échec sur l'un des fichiers émet `mappings_validation_failed` (Error) et l'agrège dans la slice d'erreurs retournée — mais un titre en échec ne bloque pas les autres.

#### Décision : pas de hot-reload (dev ou prod)

Le plan §7.3 réservait un mode `GAMES_HOT_RELOAD=true` pour le rechargement TOML à chaud en dev. Nous l'avons **délibérément non implémenté** (vérifié le 2026-04-26) :

- Prod : le hot-reload est interdit par §7.3 — la couche sémantique est un contrat versionné qui ne change que via PR + golden parity. Coût = un redéploiement par changement de label, acceptable pour quelques dizaines de FieldKey.
- Dev : une édition TOML avec le setup actuel signifie `Ctrl+C` + `air` à nouveau (~3-5s de rebuild + reboot). À notre fréquence d'édition (~1 édition TOML par sprint en dehors de l'onboarding d'un nouveau titre), le gain (~5s/édition, aucun impact production) ne justifie pas le coût (watcher fsnotify compatible Windows + méthode `Registry.Reload()` + invalidation ETag dans le handler `/field-mappings` + tests des conditions de course).

Conséquence : l'événement de log `mappings_hot_reloaded` du plan §8.1 est intentionnellement absent (8/9 événements émis, celui-ci est le 9e). À reconsidérer si/quand :
- L'onboarding d'un second vrai titre nécessite une itération TOML intensive, ou
- Le volume du catalogue croît (ex. médailles, familles d'armes) et le tuning de labels à chaud devient utile.

### API HTTP (derrière `MULTI_TITLE_API_ENABLED=true`)

- `GET /api/v1/titles/{slug}/field-mappings?locale=fr` — expose le `FieldMappingSet` d'un titre avec ETag + `Cache-Control: max-age=300`.

> Note : l'ancienne route proof-of-concept `GET /api/v1/titles/{slug}/preview/career` a été supprimée (orpheline côté frontend, cf. `server.go` + `multi_title_smoke_test.go`).

### Hooks frontend (Phase D + Phase finition)

```ts
import {
  useFieldLabel,    // FieldKey  → 'Éliminations' (kills FR) / 'Kills' (EN) / 'kills' (fallback)
  useAssetLabel,    // (kind,id) → 'Classé' (mode.Ranked FR) / 'Ranked' (EN) / 'Ranked' (fallback id)
  useOutcomeLabel,  // outcome key → 'Victoire' (win FR) / 'Win' (EN) / 'win' (fallback key)
  useAssetMapping,  // DTO complet (label + color_token + icon + display_order)
  useOutcomeMapping,
} from '@/lib/i18n/fieldMappings'
```

Tous les hooks lisent `currentTitleSlug` et `locale` depuis `appShellStore`, partagent un cache TanStack Query unique (`staleTime: Infinity` — versionné dans Git, pas de hot-reload), et retombent gracieusement sur la clé/l'id brut si l'endpoint est absent (flag off, 404, erreur réseau).

Les composants consomment ces hooks au lieu de coder les labels en dur. La frontière TOML vs i18n React (cf. plan §6.9) est imposée par `tools/lint-no-hardcoded-fields.mjs`, qui scanne 277 fichiers et rejette tout littéral correspondant à un label déclaré dans `fields.toml`, `assets.toml` ou `outcomes.toml` (whitelist pour les dictionnaires de fallback sous `features/*/fallback.i18n.ts`).

Pages migrées au 2026-04-26 : Career (encounters), Home (barre KPI, liste des défis, identité spartan), Match View (scoreboard), Synthesis (top weeks), Compare (delta cards), Media (catégories de modes), Objectifs (paliers de défis, cadences, niveaux de prestige), Communauté (palier de leaderboard), Session Detail (outcomes). Les dictionnaires `kpi.i18n.ts`, `highlights.i18n.ts`, `compare/i18n.ts` conservent leurs labels FR/EN comme fallback pour `MULTI_TITLE_API_ENABLED=false`.

### Dégradation capability-aware

Chaque `TitleDataAdapter` expose une `Capabilities() games.CapabilityMap` reflétant le support par titre des capabilities produit. Un appel `Load*` sur une capability marquée `not_exposed` retourne `games.ErrCapabilityNotSupported`, que les services en aval traduisent en un champ explicite `not_supported_reason` plutôt qu'un payload vide silencieux.

Voir [`.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`](../../.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md) pour le rationnel de conception et [`tools/mappings/CHANGELOG.md`](../../tools/mappings/CHANGELOG.md) pour l'historique de versioning du schéma TOML.
