# Plan — Sync des Achievements Xbox/Halo Infinite

> Branche : `copilot/fetch-halo-infinite-achievements`  
> Statut : **En cours de planification**  
> Mis à jour : 2026-04-24 (rev. assets Kind + multilinguisme + contrat API + pagination)

---

## 1. Objectif

Intégrer la récupération et la persistance des **achievements Xbox Live** (succès) d'un joueur Halo
Infinite dans le pipeline de synchronisation Go existant.

**Décision de fréquence (tranchée)** : les achievements sont synchronisés **à chaque cycle de sync**,
quelle que soit son origine :
- Sync **manuelle** (`POST /api/v1/players/{slug}/sync`)
- Sync **automatique** via `AutoSyncScheduler.syncPlayer()`
- Sync **scheduled** via `AutoSyncScheduler.Run()` (tick périodique)

Il n'y a **aucune garde temporelle** (pas de TTL 7 jours, pas de `fetched_at` comparé à `now()`).
La sync achievements est inconditionnelle, systématique, idempotente.

---

## 2. Contexte technique

### 2.1 API Xbox Live utilisée

| Endpoint | Description |
|----------|-------------|
| `GET https://achievements.xboxlive.com/users/xuid({xuid})/achievements?titleId=1144039928` | Progression joueur (achievements débloqués) — **paginé** |
| `GET https://achievements.xboxlive.com/users/xuid({xuid})/achievements/{scid}/{achievementId}` | Détail d'un achievement unique |
| `GET https://titlehub.xboxlive.com/titles/titleid/1144039928/decoration/Achievement` | Définitions statiques du titre (Phase 2) |

- **Title ID Halo Infinite (Xbox)** : `1144039928`
- **SCID Halo Infinite** : à récupérer depuis la réponse API (`service_config_id` dans chaque achievement)
- **Auth** : XSTS token avec `RelyingParty = http://xboxlive.com` (déjà géré par `internal/platform/auth/xsts.go`)
- Headers requis :
  - `Authorization: XBL3.0 x=<userhash>;<xsts_token>`
  - `x-xbl-contract-version: 2` (API v2 Xbox One/Halo Infinite — **obligatoire**, sans ce header l'API retourne les schémas Xbox 360 v1)
  - `Accept-Language: fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7` (voir §2.4)

### Structure JSON de réponse (API v2)

```json
{
  "achievements": [
    {
      "id": "1",
      "serviceConfigId": "...",
      "name": "Soldat de fer",
      "description": "Terminez une partie Ranked sans mourir.",
      "lockedDescription": "Accomplissez quelque chose d'exceptionnel.",
      "isSecret": false,
      "progressState": "Achieved",
      "progression": {
        "requirements": [{"id": "1", "current": "1", "target": "1", ...}],
        "timeUnlocked": "2024-01-15T18:30:00Z"
      },
      "mediaAssets": [
        {
          "name": "default",
          "type": "Icon",
          "url": "https://achievements.xboxlive.com/images/..."
        }
      ],
      "rewards": [{"type": "Gamerscore", "value": "10", ...}],
      "achievementType": "Persistent",
      "participationType": "Individual"
    }
  ],
  "pagingInfo": {
    "continuationToken": "3",
    "totalRecords": 87
  }
}
```

**Points clés** :
- `name`, `description`, `lockedDescription` → **localisés** selon `Accept-Language`
- `mediaAssets[].url` → URL **absolue** directe (ne passe pas par GameCMS)
- `pagingInfo.continuationToken` → non nul si d'autres pages existent (query param `?skipItems=N`)

### 2.2 Points d'intégration dans le code existant

```
internal/sync/engine.go
  └── run()
      └── runConditionalPostSync()        ← dispatcher selon matchesInserted
          ├── matchesInserted > 0
          │   └── runPostSyncPipeline()
          │       ├── batchComputePerformanceScores()
          │       ├── batchComputeLUSR()
          │       ├── runCareerSync()
          │       ├── refreshAggregates()
          │       └── [NOUVEAU] runAchievementsSync()   ← chemin "avec matchs"
          └── matchesInserted == 0
              ├── runCareerSync()          ← déjà présent
              └── [NOUVEAU] runAchievementsSync()       ← chemin "sans match"
                  ↑ refresh access_token via e.provider (voir §3)

internal/scheduler/auto_sync.go
  └── syncPlayer()                        ← appelle engine.RunDelta() — aucune modif nécessaire

internal/api/handlers/sync_handler.go
  └── handler sync manuelle               ← appelle RunDelta() — aucune modif nécessaire
```

> **Aucune modification** dans `syncPlayer()` ni dans le handler HTTP : la logique est intégrée
> dans `runConditionalPostSync()`, qui est le seul dispatcher depuis `run()`.

> **Cas "0 nouveau match"** : `runConditionalPostSync()` n'appelle **pas** `runPostSyncPipeline()`
> quand `matchesInserted == 0` — il retourne directement un `PostSyncResult` avec `runCareerSync()`.
> Les achievements sont un système indépendant des matchs (comme la carrière) : ils doivent être
> synchronisés dans **les deux branches**, d'où l'ajout de `runAchievementsSync()` aussi dans la
> branche `else`.

### 2.4 Stratégie multilingue

L'API Xbox renvoie `name`, `description` et `lockedDescription` dans la langue demandée via
`Accept-Language`. Le projet supporte `fr-FR` et `en-US` (pattern existant dans `career_ranks` :
colonnes `title_en` / `title_fr`).

**Décision** : appeler `GetPlayerAchievements` **deux fois** par sync (une fois par langue), et
stockqer les textes dans des colonnes suffixées dans `xbox_achievement_definitions` :

```sql
name_en        VARCHAR,   -- Accept-Language: en-US
name_fr        VARCHAR,   -- Accept-Language: fr-FR
description_en VARCHAR,
description_fr VARCHAR,
locked_desc_en VARCHAR,
locked_desc_fr VARCHAR,
```

Alternative écartée : table séparée de traductions (surcharge inutile pour 2 langues fixes et ~100 achievements).

**`image_url`** : l'URL de l'icône ne dépend pas de la langue — elle est identique quelle que soit
la langue de la requête. Un seul appel (en-US) suffit pour la récupérer.

### 2.3 Constats vérifiés dans le code actuel

- Les routes de sync **manuelles** lisent aujourd'hui `sess.HaloTokens` depuis la session HTTP.
    Cette session ne transporte actuellement que `SpartanToken` et `ClearanceToken`.
- `AutoSyncScheduler` et `ServiceRegistry` savent déjà reconstituer un `access_token` Microsoft
    frais depuis `sync_meta` / refresh token avant d'appeler `provider.Exchange(accessToken)`.
- `SyncEngine.run()` n'attache aujourd'hui `result.PostSync` que si
    `matchesInserted > 0 || postResult.CareerSynced`.
- `runCareerSync()` ouvre `metadata.duckdb` en **lecture seule** pour enrichir la carrière.
    Le moteur de sync ne prend actuellement des write leases que sur `playerDB` et `sharedDB`.

---

## 3. Auth — Décision tranchée : `TokenProvider` injecté dans `SyncEngine`

### Problème

`HaloTokens` ne contient que `SpartanToken` et `ClearanceToken` — intentionnellement : ce sont
des tokens Halo-only. L'`accessToken` Microsoft est consommé dans `provider.Exchange()` puis jeté.
Or `runAchievementsSync()` a besoin d'appeler `AcquireXSTSForRTA(ctx, accessToken)` pour obtenir
un token XSTS avec `RelyingParty = http://xboxlive.com`.

### Décision : Option B — refresh on-demand via `TokenProvider`

`HaloTokens` reste inchangé. `SyncEngine` reçoit un `auth.TokenProvider` injecté, et
`runAchievementsSync()` obtient un `access_token` frais en relisant `sync_meta` depuis
`playerDB` au moment de l'appel.

**Avantages :**
- Zéro diff sur `domain.HaloTokens`, `Attempt`, `auth.go` handler, session HTTP
- Token Microsoft toujours frais (pas de risque d'expiration en cache)
- Séparation de couches respectée : la couche sync ne transporte pas de token Microsoft au-delà
  de son besoin immédiat

### Diff minimal

**`internal/sync/engine.go`** — nouveau champ + mise à jour constructeur :

```go
type SyncEngine struct {
    gamertag       string
    xuid           string
    playerDBPath   string
    sharedDBPath   string
    metadataDBPath string
    tokens         *domain.HaloTokens
    provider       auth.TokenProvider // ← nouveau
}

func NewSyncEngine(
    repoRoot, gamertag, xuid string,
    tokens *domain.HaloTokens,
    provider auth.TokenProvider, // ← nouveau
) *SyncEngine {
    ...
    return &SyncEngine{ ..., provider: provider }
}
```

**`internal/scheduler/auto_sync.go`** — `defaultEngineFactory` et signature `EngineFactory` :

```go
// EngineFactory — signature étendue
type EngineFactory func(repoRoot, gamertag, xuid string, tokens *domain.HaloTokens, provider auth.TokenProvider) DeltaRunner

// defaultEngineFactory
func defaultEngineFactory(repoRoot, gamertag, xuid string, tokens *domain.HaloTokens, provider auth.TokenProvider) DeltaRunner {
    return sync.NewSyncEngine(repoRoot, gamertag, xuid, tokens, provider)
}

// syncPlayer — passer s.provider
runner := s.EngineFactory(s.cfg.RepoRoot, p.Gamertag, p.XUID, result.Tokens, s.provider)
```

**`internal/api/handlers/sync_handler.go`** — idem, passer le provider injecté au handler.

### 3.1 Implémentation de `runAchievementsSync`

`runAchievementsSync()` lit `msal_token_cache` et `oauth_refresh_token` depuis `playerDB`
(déjà ouvert), puis appelle les méthodes existantes du provider :

```go
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) bool {
    slog.DebugContext(ctx, "post-sync: sync achievements", "gamertag", e.gamertag)

    accessToken, err := resolveAccessToken(ctx, playerDB, e.gamertag, e.provider)
    if err != nil || accessToken == "" {
        slog.WarnContext(ctx, "post-sync: access_token introuvable pour achievements",
            "gamertag", e.gamertag, "err", err)
        return false
    }

    xstsResult, err := authpkg.AcquireXSTSForRTA(ctx, accessToken)
    if err != nil {
        slog.WarnContext(ctx, "post-sync: XSTS RTA échoué", "gamertag", e.gamertag, "err", err)
        return false
    }
    xboxClient := newXboxHTTPClient(xstsResult)

    if err := SyncAchievements(ctx, xboxClient, playerDB, e.xuid); err != nil {
        slog.WarnContext(ctx, "post-sync: achievements échoué", "gamertag", e.gamertag, "err", err)
        return false
    }
    return true
}
```

`resolveAccessToken` est un helper interne au package `sync` (non exporté) qui reproduit
la séquence `TrySilentRefresh → TryOAuthRefresh`.

**Garde-fou structurel — `resolveAccessToken` ne prend PAS `*sql.DB`.**

La signature est :
```go
func resolveAccessToken(ctx context.Context, msalCacheJSON, oauthRefreshToken, gamertag string, provider auth.TokenProvider) (string, error)
```

Les deux lectures `sync_meta` sont faites dans `runAchievementsSync` **avant** l'appel
à `resolveAccessToken`, via `QueryRowContext` sans transaction :

```go
func (e *SyncEngine) runAchievementsSync(ctx context.Context, playerDB *sql.DB) bool {
    // Lire les credentials AVANT tout appel réseau.
    // À ce stade, toutes les étapes du pipeline ont retourné → aucune tx active.
    var msalCacheJSON, oauthRefreshToken string
    _ = playerDB.QueryRowContext(ctx,
        "SELECT value FROM sync_meta WHERE key = 'msal_token_cache'",
    ).Scan(&msalCacheJSON)
    _ = playerDB.QueryRowContext(ctx,
        "SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'",
    ).Scan(&oauthRefreshToken)

    accessToken, err := resolveAccessToken(ctx, msalCacheJSON, oauthRefreshToken, e.gamertag, e.provider)
    // ...
}
```

**Pourquoi cette séparation est le vrai garde-fou :**

`playerDB` est `OpenReadWrite` → `MaxOpenConns(1)`. Si un futur développeur ajoute un
`BeginTx` dans une étape du pipeline sans refermer la transaction avant que
`runAchievementsSync` soit appelé, les deux `QueryRowContext` échoueront avec une erreur
explicite (`context deadline exceeded` ou driver error) plutôt qu'un deadlock silencieux.

De plus, `resolveAccessToken` n'ayant aucune dépendance vers `*sql.DB`, elle est
entièrement testable sans base de données — un test unitaire passe les deux strings
directement et mock `provider`.

### 3.2 Question de périmètre d'autorisation

`POST /api/v1/sync/all` réutilise les tokens de la session courante pour potentiellement
cibler d'autres joueurs. Avec Option B, chaque joueur rechargera son propre cache MSAL depuis
sa `playerDB` — le token est donc toujours celui du propriétaire du compte, pas celui de
la session appelante. Ce point est résolu nativement.

---

## 4. Schéma DuckDB

### 4.1 `metadata.duckdb` — définitions (référentiel)

> **Phase 1 : table créée par migration Go uniquement.** Elle n'est pas alimentée par le
> hot path de sync joueur (voir §5.3 bis). La migration crée la structure ; le peuplement
> initial est délégué à un endpoint ou script séparé (Phase 2).

```sql
CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
    achievement_id   VARCHAR PRIMARY KEY,
    -- Textes localisés (2 langues, pattern identique à career_ranks)
    name_en          VARCHAR NOT NULL DEFAULT '',
    name_fr          VARCHAR NOT NULL DEFAULT '',
    description_en   VARCHAR,
    description_fr   VARCHAR,
    locked_desc_en   VARCHAR,
    locked_desc_fr   VARCHAR,
    gamerscore       INTEGER NOT NULL DEFAULT 0,
    -- URL absolue de l'icône (mediaAssets[type=Icon].url depuis l'API Xbox)
    -- Invariante selon la langue — récupérée depuis la réponse en-US
    image_url        VARCHAR,
    is_secret        BOOLEAN NOT NULL DEFAULT false,
    rarity_category  VARCHAR,   -- "Common" | "Rare" (2 valeurs Xbox)
    rarity_percent   FLOAT,
    fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

> **Abandon de `name VARCHAR` / `description VARCHAR` mono-langue** : les textes Xbox sont
> localisés via `Accept-Language` — une seule colonne stockerait silencieusement la mauvaise
> langue selon l'ordre des appels. Colonnes `_en` / `_fr` explicites, comme `career_ranks`.

### 4.2 `stats.duckdb` (par joueur) — progression

```sql
CREATE TABLE IF NOT EXISTS player_achievements (
    achievement_id   VARCHAR PRIMARY KEY,
    unlocked         BOOLEAN NOT NULL DEFAULT false,
    unlocked_at      TIMESTAMP,
    current_progress INTEGER,
    target_progress  INTEGER,
    fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Upsert inconditionnel à chaque sync (`ON CONFLICT (achievement_id) DO UPDATE SET ...`).

> **Note** : `INSERT OR REPLACE` n'existe pas en DuckDB — utiliser exclusivement
> `INSERT INTO ... ON CONFLICT (...) DO UPDATE SET`.

**Convention de nommage fixée** : `snake_case` pour toutes les colonnes, `TIMESTAMP` (sans TZ)
pour tous les champs temporels. Applicable uniformément aux migrations, structs Go et tests.

---

## 5. Nouveaux fichiers Go à créer

```
internal/sync/
├── xbox_client.go           ← interface XboxAchievementsClient + implémentation HTTP (auth XSTS)
├── achievements.go          ← structs JSON, SyncAchievements(), upserts DuckDB
└── achievements_test.go     ← tests unitaires (mock XboxClient)
```

### 5.1 `xbox_client.go` — client HTTP Xbox

```
XboxAchievementsClient        interface (mockable)
  └── GetPlayerAchievements(ctx, xuid, lang string) ([]PlayerAchievement, error)
        ↑ endpoint achievements.xboxlive.com — progression par joueur (Phase 1)
        ↑ lang = "en-US" ou "fr-FR" → Accept-Language header
        ↑ gère la pagination interne (loop sur continuationToken)

  // Phase 2 uniquement :
  // GetAchievementDefinitions(ctx) ([]AchievementDef, error)
  //   ↑ endpoint titlehub — global au titre, pas de xuid

xboxHTTPClient               implémentation concrète
    authHeader string          ← XSTSResult.AuthHeader() = "XBL3.0 x=<userhash>;<xsts_token>"
```

> Le client stocke `authHeader string` (résultat de `xstsResult.AuthHeader()`), pas le token brut.
> `XSTSResult.AuthHeader()` est déjà implémenté dans `internal/platform/auth/xsts.go`.

**Headers envoyés par `xboxHTTPClient`** :
```
Authorization:           XBL3.0 x=<userhash>;<xsts_token>
x-xbl-contract-version:  2
Accept-Language:         <lang>  (ex: "en-US" ou "fr-FR")
Accept:                  application/json
```

**Pagination** : `GetPlayerAchievements` boucle jusqu'à `pagingInfo.continuationToken == ""`
en ajoutant `?skipItems=N` (ou `&continuationToken=...`) à chaque page.

### 5.1 bis `KindAchievementImage` — intégration au resolver assets

Chaque achievement expose une ou plusieurs entrées `mediaAssets[]` — en pratique une icône
(`type: "Icon"`) avec une URL absolue (ex: `https://achievements.xboxlive.com/images/{path}.png`).
Cette URL **ne passe pas par GameCMS** — elle est directement servable.

**Ce que le plan original oubliait** : ces images ne sont pas cachées. Sans `Kind`, elles ne
passent ni par `LocalFSStore` ni par `DuckDBIndexStore`. À chaque affichage le client ferait
une requête directe vers Xbox, ou pire, le frontend ne les afficherait pas du tout.

**Décision** : ajouter `KindAchievementImage` dans `internal/assets/kinds.go` et un fetcher
HTTP simple (URL absolue, authentification XSTS non requise pour les images publiques) :

```go
// kinds.go
// KindAchievementImage est l'icône d'un achievement Xbox (PNG).
// Source : URL absolue contenue dans mediaAssets[type=Icon].url (API Xbox v2).
// L'URL est stockée dans xbox_achievement_definitions.image_url.
KindAchievementImage Kind = "achievement-image"
```

`Ref` pour une image d'achievement :
```go
assets.Ref{
    Kind:    assets.KindAchievementImage,
    TitleID: "halo_infinite",
    ID:      achievement_id,   // ex: "1"
    // Variant: vide (une seule icône par achievement)
}
```

L'URL absolue est persistée dans `asset_index` (colonne `url`) via le resolver standard —
aucun fetcher GameCMS nécessaire. Un `XboxImageFetcher` (ou réutilisation du `GenericURLFetcher`
existant si présent) effectue un `GET` direct sur `image_url` depuis `xbox_achievement_definitions`.

> **Note** : le caching des images est Phase 1, mais peut être implémenté dans le même lot que
> `SyncAchievements` puisque l'URL est déjà connue après l'appel API. Warm optionnel (fire-and-forget).

### 5.2 `achievements.go` — responsabilités

```
PlayerAchievement            struct (mapping JSON → DuckDB player)
  achievement_id, name_en, name_fr, description_en, description_fr,
  locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
  rarity_category, rarity_percent,
  unlocked, unlocked_at, current_progress, target_progress

// AchievementDef réservé Phase 2 (endpoint titlehub)

SyncAchievements(ctx, client, playerDB, xuid) error
  1. GetPlayerAchievements(xuid, "en-US") → slice EN
  2. GetPlayerAchievements(xuid, "fr-FR") → slice FR
  3. Merger EN + FR (clé = achievement_id)
  4. Upsert player_achievements dans stats.duckdb
     (ON CONFLICT (achievement_id) DO UPDATE SET ...)
  5. [Optionnel Phase 1] Warm des images via resolver.Warm(KindAchievementImage refs)
```

> Les deux appels API (EN + FR) sont faits séquentiellement — même jeu de progression,
> seuls `name`/`description`/`lockedDescription` diffèrent. `image_url` et `progressState`
> sont lus depuis la réponse EN-US (premier appel).

Phase 1 ne touche pas `metadata.duckdb` — aucun write lock concurrentiel.

### 5.3 Intégration dans `engine.go`

`runAchievementsSync()` est une méthode du `SyncEngine`, appelée depuis **deux endroits** :
- Dans `runPostSyncPipeline()` (chemin `matchesInserted > 0`)
- Dans la branche `else` de `runConditionalPostSync()` (chemin `matchesInserted == 0`), aux côtés de `runCareerSync()`

```go
// Dans runConditionalPostSync() — branche 0 match :
return domain.PostSyncResult{
    CareerSynced:       e.runCareerSync(ctx, playerDB, client),
    AchievementsSynced: e.runAchievementsSync(ctx, playerDB),
}

// Dans runPostSyncPipeline() — chemin > 0 match :
// 5. Achievements Xbox
r.AchievementsSynced = e.runAchievementsSync(ctx, playerDB)
return r
```

Le corps de `runAchievementsSync()` est documenté au §3.1.

### 5.3 bis Écriture dans `metadata.duckdb` — décision tranchée

**Phase 1 : les définitions globales sont découplées du hot path joueur.**

Raison principale : DuckDB n'autorise qu'un seul writer simultané par fichier. `POST /api/v1/sync/all`
déclenche des syncs en parallèle ; si deux goroutines tentent d'ouvrir `metadata.duckdb` en
read-write simultanément, la seconde échoue avec une erreur de lock. Ce risque est éliminé en
n'écrivant jamais dans `metadata.duckdb` depuis `runAchievementsSync()`.

- **Phase 1** : `SyncAchievements()` n'écrit que dans `stats.duckdb` du joueur (`player_achievements`)
- `xbox_achievement_definitions` dans `metadata.duckdb` est créée par la migration Go mais
  son peuplement est réservé à la Phase 2 (endpoint ou commande dédiée)
- `runCareerSync()` continue d'ouvrir `metadata.duckdb` en **lecture seule** — inchangé

### 5.4 Mise à jour `domain/sync.go`

Ajouter dans `PostSyncResult` — Phase 1 : booléen uniquement, pas de compteur :

```go
AchievementsSynced bool
```

`AchievementsCount` n'est pas ajouté en Phase 1 : le booléen suffit pour les logs et la
condition d'attachement. Phase 2 pourra l'ajouter avec une sémantique précise si nécessaire.

### 5.5 Publication du résultat post-sync — décision tranchée

La condition d'attachement de `result.PostSync` est étendue à :

```go
if result.MatchesInserted > 0 || postResult.CareerSynced || postResult.AchievementsSynced {
    result.PostSync = &postResult
}
```

Si les achievements réussissent seuls sur un cycle à `0 match` + carrière non modifiée, le
résultat est correctement attaché et loggé.

---

## 6. Migrations de schéma Go

Les migrations Go sont des blocs `init()` dans les fichiers du package
`apps/go-api/internal/migration/`. Les fichiers Python dans `src/data/migration/steps/`
ne sont **pas** exécutés par le backend Go (`migration.RunForDB()`).

### `steps_metadata.go` — ajouter un bloc `Register()`

```go
Register(Migration{
    Name:        "add_xbox_achievement_definitions",
    TargetDB:    TargetMetadata,
    Description: "Table xbox_achievement_definitions (référentiel achievements Halo Infinite)",
    ApplySchema: func(db *sql.DB) error {
        return execScript(db, `
            CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
                achievement_id   VARCHAR PRIMARY KEY,
                name             VARCHAR NOT NULL,
                description      VARCHAR,
                locked_desc      VARCHAR,
                gamerscore       INTEGER NOT NULL DEFAULT 0,
                image_url        VARCHAR,
                is_secret        BOOLEAN NOT NULL DEFAULT false,
                rarity_category  VARCHAR,
                rarity_percent   FLOAT,
                fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        `)
    },
})
```

### `steps_metadata.go` — `xbox_achievement_definitions` (colonnes multilingues)

> Le schéma reflète la structure multilingue décrite en §2.4 et §4.1.

```go
Register(Migration{
    Name:        "add_xbox_achievement_definitions",
    TargetDB:    TargetMetadata,
    Description: "Table xbox_achievement_definitions (référentiel achievements Halo Infinite — bilingue EN/FR)",
    ApplySchema: func(db *sql.DB) error {
        return execScript(db, `
            CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
                achievement_id   VARCHAR PRIMARY KEY,
                name_en          VARCHAR NOT NULL DEFAULT '',
                name_fr          VARCHAR NOT NULL DEFAULT '',
                description_en   VARCHAR,
                description_fr   VARCHAR,
                locked_desc_en   VARCHAR,
                locked_desc_fr   VARCHAR,
                gamerscore       INTEGER NOT NULL DEFAULT 0,
                image_url        VARCHAR,
                is_secret        BOOLEAN NOT NULL DEFAULT false,
                rarity_category  VARCHAR,
                rarity_percent   FLOAT,
                fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        `)
    },
})
```

### `steps_player.go` — ajouter un bloc `Register()`

```go
Register(Migration{
    Name:        "add_player_achievements",
    TargetDB:    TargetPlayer,
    Description: "Table player_achievements (progression achievements par joueur)",
    ApplySchema: func(db *sql.DB) error {
        return execScript(db, `
            CREATE TABLE IF NOT EXISTS player_achievements (
                achievement_id   VARCHAR PRIMARY KEY,
                unlocked         BOOLEAN NOT NULL DEFAULT false,
                unlocked_at      TIMESTAMP,
                current_progress INTEGER,
                target_progress  INTEGER,
                fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        `)
    },
})
```

---

## 7. Endpoints API (optionnel / Phase 2)

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/v1/players/{slug}/achievements` | Progression achievements du joueur |
| `GET` | `/api/v1/achievements` | Toutes les définitions (metadata) |

Ces endpoints sont **Phase 2** — la sync est le livrable Phase 1.

---

## 8. Plan d'implémentation (ordre)

- [ ] **Étape 1** — Migrations Go : `steps_metadata.go` (`xbox_achievement_definitions`) +
    `steps_player.go` (`player_achievements`)
- [ ] **Étape 2** — `domain/sync.go` : ajouter `AchievementsSynced bool` à `PostSyncResult` +
    étendre la condition d'attachement (`|| postResult.AchievementsSynced`)
- [ ] **Étape 3** — `kinds.go` : ajouter `KindAchievementImage` + `allKinds` + `IsBinary()`
    + extension de fichier `.png` dans `store_localfs.go`
- [ ] **Étape 4** — `xbox_client.go` : interface `XboxAchievementsClient` (Phase 1 :
    `GetPlayerAchievements(ctx, xuid, lang string)` avec pagination interne et headers
    `x-xbl-contract-version: 2` + `Accept-Language`) + implémentation `xboxHTTPClient`
- [ ] **Étape 5** — `achievements.go` : struct `PlayerAchievement` (champs bilingues) +
    `SyncAchievements(ctx, client, resolver, playerDB, xuid)` — 2 appels EN+FR, merge,
    upsert `ON CONFLICT DO UPDATE` + `resolver.Warm(KindAchievementImage)` fire-and-forget
- [ ] **Étape 6** — `achievements_test.go` : tests unitaires avec mock `XboxAchievementsClient`
    (couvre pagination, merge EN+FR, upsert, warm assets)
- [ ] **Étape 7** — `engine.go` :
    - Ajouter champ `provider auth.TokenProvider` à `SyncEngine` + mettre à jour `NewSyncEngine`
    - Ajouter helper non-exporté `resolveAccessToken(ctx, playerDB, gamertag, provider) (string, error)`
    - Ajouter méthode `runAchievementsSync(ctx, playerDB)` (voir §3.1) — passe le resolver
    - Brancher dans `runPostSyncPipeline()` et dans la branche `0 match` de `runConditionalPostSync()`
- [ ] **Étape 8** — `auto_sync.go` : mettre à jour `EngineFactory` + `defaultEngineFactory` +
    `syncPlayer()` pour passer `s.provider`
- [ ] **Étape 9** — Handler sync manuelle : passer le provider à `EngineFactory`
- [ ] **Étape 10** — Tests d'intégration : 3 chemins (manuel, auto, scheduled), cas `0 match`,
    cas `career=false` / `achievements=true`, cas pagination (>1 page)
- [ ] **Étape 11** *(Phase 2)* — `GetAchievementDefinitions` + peuplement `metadata.duckdb` +
    endpoints API `GET /achievements` et `GET /players/{slug}/achievements`

---

## 9. Contraintes et décisions

| Sujet | Décision |
|-------|----------|
| Fréquence | Chaque sync, sans condition temporelle |
| Cas 0 match | `runAchievementsSync()` appelé dans **les deux** branches de `runConditionalPostSync()`, comme `runCareerSync()` |
| Erreur API Xbox | `slog.Warn` + continuation (non bloquant, même pattern que `runCareerSync`) |
| Auth | **Option B tranchée** : `TokenProvider` injecté dans `SyncEngine` ; refresh on-demand depuis `sync_meta` via `TrySilentRefresh → TryOAuthRefresh`. `HaloTokens`, `Attempt`, session HTTP : inchangés |
| Idempotence | `ON CONFLICT (achievement_id) DO UPDATE SET ...` — safe à rejouer N fois |
| Migrations | Blocs `Register()` Go dans `steps_metadata.go` (table vide Phase 1) / `steps_player.go` — pas de fichiers Python |
| Définitions (Phase 1) | Table `xbox_achievement_definitions` créée par migration mais **non alimentée** par le hot path sync — Phase 2 uniquement |
| Progression | Upsert à chaque sync dans `stats.duckdb` du joueur (`player_achievements`) |
| Write lock metadata | Aucun write sur `metadata.duckdb` depuis le hot path joueur — risque de lock DuckDB éliminé |
| Header Xbox | `xboxHTTPClient` stocke `authHeader string` = `xstsResult.AuthHeader()` |
| **Header contrat** | `x-xbl-contract-version: 2` **obligatoire** — sans lui, l'API retourne le schéma Xbox 360 v1 (champs différents, `image_id int` au lieu de `mediaAssets[]`) |
| **`Accept-Language`** | Deux appels par sync : `en-US` puis `fr-FR` — textes mergés avant upsert. `image_url` invariant, lu depuis la réponse EN-US |
| **Multilinguisme** | Colonnes `name_en` / `name_fr` / `description_en` / `description_fr` / `locked_desc_en` / `locked_desc_fr` — pattern `career_ranks` |
| **Pagination** | `GetPlayerAchievements` boucle sur `pagingInfo.continuationToken` — géré en interne dans le client, opaque pour `SyncAchievements` |
| **`KindAchievementImage`** | Nouveau Kind dans `internal/assets/kinds.go` — images cachées via `LocalFSStore` + `DuckDBIndexStore` comme tous les autres assets visuels. Warm fire-and-forget après upsert |
| Publication PostSync | Condition étendue : `matchesInserted > 0 \|\| CareerSynced \|\| AchievementsSynced` |
| `AchievementsCount` | **Supprimé en Phase 1** — booléen suffisant |
| Cross-player achievements | Résolu nativement par Option B : chaque joueur utilise son propre token depuis sa `playerDB` |
| Convention schéma | `snake_case` colonnes + `TIMESTAMP` (sans TZ) partout |
| pandas / SQLite | Aucun — DuckDB + `database/sql` Go uniquement |
