# Plan — Sync des Achievements Xbox/Halo Infinite

> Branche : `copilot/fetch-halo-infinite-achievements`  
> Statut : **En cours de planification**  
> Mis à jour : 2026-04-23 (rev. corrections bloquantes)

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
| `GET https://achievements.xboxlive.com/users/xuid({xuid})/achievements?titleId=1144039928` | Progression joueur (achievements débloqués) |
| `GET https://achievements.xboxlive.com/users/xuid({xuid})/achievements/{achievementId}` | Détail d'un achievement |
| `GET https://titlehub.xboxlive.com/titles/titleid/1144039928/decoration/Achievement` | Définitions statiques du titre |

- **Title ID Halo Infinite (Xbox)** : `1144039928`
- **Auth** : XSTS token avec `RelyingParty = http://xboxlive.com` (déjà géré par `internal/platform/auth/xsts.go`)
- Header requis : `Authorization: XBL3.0 x=<userhash>;<xsts_token>`

### 2.2 Points d'intégration dans le code existant

```
internal/sync/engine.go
  └── run()                              ← point d'intégration principal
      └── runPostSyncPipeline()          ← pipeline post-match, toujours exécuté
          ├── batchComputePerformanceScores()
          ├── batchComputeLUSR()
          ├── runCareerSync()            ← modèle à suivre
          ├── refreshAggregates()
          └── [NOUVEAU] runAchievementsSync()  ← à ajouter ici
              ↑ utilise e.tokens.MSAccessToken (voir §3 — option A)

internal/scheduler/auto_sync.go
  └── syncPlayer()                       ← appelle engine.RunDelta() — aucune modif nécessaire

internal/api/handlers/sync_handler.go
  └── handler sync manuelle              ← appelle RunDelta() — aucune modif nécessaire
```

> **Aucune modification** dans `syncPlayer()` ni dans le handler HTTP : la logique est intégrée
> dans `run()` via `runPostSyncPipeline()`, qui s'exécute dans tous les chemins de sync.

> **Cas "0 nouveau match"** : `runPostSyncPipeline()` est appelé inconditionnellement (voir
> `run()` dans `engine.go`). La sync achievements s'exécute donc même si aucun nouveau match
> n'a été inséré — comportement intentionnel et cohérent avec la décision §1.

---

## 3. Auth — Option A : `MSAccessToken` dans `HaloTokens`

### Problème

`HaloTokens` ne contient que `SpartanToken` et `ClearanceToken`. L'`accessToken` Microsoft est
consommé dans `provider.Exchange()` puis **jeté** — il n'est jamais transmis au `SyncEngine`.
Or `runAchievementsSync()` a besoin d'appeler `AcquireXSTSForRTA(ctx, accessToken)` pour obtenir
le token XSTS avec `RelyingParty = http://xboxlive.com`.

### Solution retenue

Ajouter `MSAccessToken string` à `domain.HaloTokens` et le stocker dans `ExchangeAccessToken()`.

```go
// apps/go-api/internal/domain/auth.go
type HaloTokens struct {
    SpartanToken   string
    ClearanceToken string
    MSAccessToken  string // token Microsoft — requis pour AcquireXSTSForRTA (achievements Xbox)
}
```

```go
// apps/go-api/internal/platform/auth/halo_exchange.go — ExchangeAccessToken()
return &ExchangeResult{
    Tokens: &domain.HaloTokens{
        SpartanToken:   spartanToken,
        ClearanceToken: clearanceToken,
        MSAccessToken:  accessToken, // ← stocké ici, transmis au SyncEngine via EngineFactory()
    },
    Gamertag: gamertag,
    XUID:     xuid,
}, nil
```

- **Zéro diff** dans le scheduler (`auto_sync.go`) et les handlers (`sync_handler.go`) :
  `result.Tokens` est déjà passé à `EngineFactory()` tel quel.
- **Zéro latence** ajoutée au flow d'auth principal : `AcquireXSTSForRTA` est appelé
  **lazily** dans `runAchievementsSync()`, uniquement quand on arrive à cette étape.

---

## 4. Schéma DuckDB

### 4.1 `metadata.duckdb` — définitions (référentiel)

```sql
CREATE TABLE IF NOT EXISTS xbox_achievement_definitions (
    achievement_id   VARCHAR PRIMARY KEY,
    name             VARCHAR NOT NULL,
    description      VARCHAR,
    locked_desc      VARCHAR,
    gamerscore       INTEGER NOT NULL DEFAULT 0,
    image_url        VARCHAR,
    is_secret        BOOLEAN NOT NULL DEFAULT false,
    rarityCategory   VARCHAR,   -- "Common" | "Uncommon" | "Rare" | "Ultra Rare"
    rarityPercent    FLOAT,
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 4.2 `stats.duckdb` (par joueur) — progression

```sql
CREATE TABLE IF NOT EXISTS player_achievements (
    achievement_id   VARCHAR NOT NULL,
    unlocked         BOOLEAN NOT NULL DEFAULT false,
    unlocked_at      TIMESTAMPTZ,
    current_progress INTEGER,
    target_progress  INTEGER,
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (achievement_id)
);
```

Upsert inconditionnel à chaque sync (`ON CONFLICT (achievement_id) DO UPDATE SET ...`).

> **Note** : `INSERT OR REPLACE` n'existe pas en DuckDB — utiliser exclusivement
> `INSERT INTO ... ON CONFLICT (...) DO UPDATE SET`.

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
  ├── GetAchievementDefinitions(ctx) ([]AchievementDef, error)
  │     ↑ endpoint titlehub — global au titre, pas par joueur (pas de xuid)
  └── GetPlayerAchievements(ctx, xuid)    ([]PlayerAchievement, error)

xboxHTTPClient               implémentation concrète
  ├── champ xstsToken string  ← obtenu via AcquireXSTSForRTA(ctx, e.tokens.MSAccessToken)
  └── méthodes HTTP correspondantes
```

### 5.2 `achievements.go` — responsabilités

```
AchievementDef               struct (mapping JSON → DuckDB metadata)
PlayerAchievement            struct (mapping JSON → DuckDB player)

SyncAchievements(ctx, client, metadataDB, playerDB, xuid) error
  1. Appel GetAchievementDefinitions() → upsert metadata.duckdb
  2. Appel GetPlayerAchievements(xuid) → upsert stats.duckdb
```

### 5.3 Intégration dans `engine.go`

```go
// Dans runPostSyncPipeline() :
// 5. Achievements Xbox
slog.DebugContext(ctx, "post-sync: sync achievements", "gamertag", e.gamertag)
xstsResult, err := authpkg.AcquireXSTSForRTA(ctx, e.tokens.MSAccessToken)
if err != nil {
    slog.WarnContext(ctx, "post-sync: XSTS RTA échoué", "gamertag", e.gamertag, "err", err)
} else {
    xboxClient := newXboxHTTPClient(xstsResult)
    if err := SyncAchievements(ctx, xboxClient, metadataDB, playerDB, e.xuid); err != nil {
        slog.WarnContext(ctx, "post-sync: achievements échoué", "gamertag", e.gamertag, "err", err)
    } else {
        r.AchievementsSynced = true
    }
}
```

### 5.4 Mise à jour `domain/sync.go`

Ajouter dans `PostSyncResult` :

```go
AchievementsSynced bool
AchievementsCount  int
```

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

- [ ] **Étape 1** — `domain/auth.go` : ajouter `MSAccessToken string` à `HaloTokens`
- [ ] **Étape 2** — `platform/auth/halo_exchange.go` : stocker `accessToken` dans `HaloTokens.MSAccessToken`
- [ ] **Étape 3** — Migrations Go : blocs `Register()` dans `steps_metadata.go` et `steps_player.go`
- [ ] **Étape 4** — `xbox_client.go` : interface `XboxAchievementsClient` + implémentation HTTP
- [ ] **Étape 5** — `achievements.go` : structs JSON + `SyncAchievements()` + upserts DuckDB
- [ ] **Étape 6** — `achievements_test.go` : tests unitaires avec mock client
- [ ] **Étape 7** — `domain/sync.go` : champs `AchievementsSynced` / `AchievementsCount` dans `PostSyncResult`
- [ ] **Étape 8** — `engine.go` : appel `runAchievementsSync()` dans `runPostSyncPipeline()`
- [ ] **Étape 9** — Tests d'intégration + vérification que la sync est bien déclenchée sur les 3 chemins (manuel, auto, scheduled)
- [ ] **Étape 10** *(Phase 2)* — Endpoints API `GET /achievements` et `GET /players/{slug}/achievements`

---

## 9. Contraintes et décisions

| Sujet | Décision |
|-------|----------|
| Fréquence | Chaque sync, sans condition temporelle |
| Cas 0 match | `runPostSyncPipeline()` s'exécute inconditionnellement — achievements synced même si 0 match inséré |
| Erreur API Xbox | `slog.Warn` + continuation (non bloquant, même pattern que `runCareerSync`) |
| Auth | Option A : `MSAccessToken` stocké dans `HaloTokens` dès `ExchangeAccessToken()`, passé au `SyncEngine` via `EngineFactory()` |
| Idempotence | `ON CONFLICT (...) DO UPDATE SET ...` — safe à rejouer N fois (`INSERT OR REPLACE` inexistant en DuckDB) |
| Migrations | Blocs `Register()` Go dans `steps_metadata.go` / `steps_player.go` — pas de fichiers Python |
| `GetAchievementDefinitions` | Pas de `xuid` — les définitions sont globales au titre (endpoint titlehub) |
| Définitions | Upsert à chaque sync dans `metadata.duckdb` (léger, ~100 entrées Halo Infinite) |
| Progression | Upsert à chaque sync dans `stats.duckdb` du joueur |
| pandas / SQLite | Aucun — DuckDB + `database/sql` Go uniquement |
