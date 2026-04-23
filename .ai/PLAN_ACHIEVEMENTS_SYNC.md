# Plan — Sync des Achievements Xbox/Halo Infinite

> Branche : `copilot/fetch-halo-infinite-achievements`  
> Statut : **En cours de planification**  
> Mis à jour : 2026-04-23

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

internal/scheduler/auto_sync.go
  └── syncPlayer()                       ← appelle engine.RunDelta() — aucune modif nécessaire

internal/api/handlers/sync_handler.go
  └── handler sync manuelle              ← appelle RunDelta() — aucune modif nécessaire
```

> **Aucune modification** dans `syncPlayer()` ni dans le handler HTTP : la logique est intégrée
> dans `run()` via `runPostSyncPipeline()`, qui s'exécute dans tous les chemins de sync.

---

## 3. Schéma DuckDB

### 3.1 `metadata.duckdb` — définitions (référentiel)

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

### 3.2 `stats.duckdb` (par joueur) — progression

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

Upsert inconditionnel à chaque sync (`INSERT OR REPLACE` / `ON CONFLICT DO UPDATE`).

---

## 4. Nouveaux fichiers Go à créer

```
internal/sync/
├── achievements.go          ← client Xbox + extraction + upsert (logique principale)
└── achievements_test.go     ← tests unitaires (mock XboxClient)
```

### 4.1 `achievements.go` — responsabilités

```
XboxAchievementsClient        interface (mockable)
  ├── GetAchievementDefinitions(ctx, xuid) ([]AchievementDef, error)
  └── GetPlayerAchievements(ctx, xuid)    ([]PlayerAchievement, error)

AchievementDef               struct (mapping JSON → DuckDB)
PlayerAchievement            struct (mapping JSON → DuckDB)

SyncAchievements(ctx, client, metadataDB, playerDB, xuid) error
  1. Appel GetAchievementDefinitions → upsert metadata.duckdb
  2. Appel GetPlayerAchievements    → upsert stats.duckdb
```

### 4.2 Intégration dans `engine.go`

```go
// Dans runPostSyncPipeline() :
// 5. Achievements Xbox
slog.DebugContext(ctx, "post-sync: sync achievements", "gamertag", e.gamertag)
if err := SyncAchievements(ctx, xboxClient, metadataDB, playerDB, e.xuid); err != nil {
    slog.WarnContext(ctx, "post-sync: achievements échoué", "gamertag", e.gamertag, "err", err)
} else {
    r.AchievementsSynced = true
}
```

Le `xboxClient` utilise le XSTS token déjà présent dans `e.tokens` (le token Xbox Live
est obtenu via le même Device Code Flow, via `AcquireXSTSForRTA`).

### 4.3 Mise à jour `domain/sync.go`

Ajouter dans `PostSyncResult` :

```go
AchievementsSynced bool
AchievementsCount  int
```

---

## 5. Migrations de schéma

### `src/data/migration/steps/add_xbox_achievement_definitions.py`
→ `target_db = "metadata"`, `ensure_xbox_achievement_definitions()`

### `src/data/migration/steps/add_player_achievements.py`
→ `target_db = "player"`, `ensure_player_achievements()`

Import dans `__init__.py`.

---

## 6. Endpoints API (optionnel / Phase 2)

| Méthode | Route | Description |
|---------|-------|-------------|
| `GET` | `/api/v1/players/{slug}/achievements` | Progression achievements du joueur |
| `GET` | `/api/v1/achievements` | Toutes les définitions (metadata) |

Ces endpoints sont **Phase 2** — la sync est le livrable Phase 1.

---

## 7. Plan d'implémentation (ordre)

- [ ] **Étape 1** — Migrations DuckDB : créer `xbox_achievement_definitions` (metadata) et `player_achievements` (player)
- [ ] **Étape 2** — `achievements.go` : interface `XboxAchievementsClient` + structs JSON
- [ ] **Étape 3** — `achievements.go` : implémenter le client HTTP Xbox (XSTS auth)
- [ ] **Étape 4** — `achievements.go` : `SyncAchievements()` + upserts DuckDB
- [ ] **Étape 5** — `achievements_test.go` : tests unitaires avec mock client
- [ ] **Étape 6** — `engine.go` : appel `runAchievementsSync()` dans `runPostSyncPipeline()`
- [ ] **Étape 7** — `domain/sync.go` : champs `AchievementsSynced` / `AchievementsCount` dans `PostSyncResult`
- [ ] **Étape 8** — Tests d'intégration + vérification que la sync est bien déclenchée sur les 3 chemins (manuel, auto, scheduled)
- [ ] **Étape 9** *(Phase 2)* — Endpoints API `GET /achievements` et `GET /players/{slug}/achievements`

---

## 8. Contraintes et décisions

| Sujet | Décision |
|-------|----------|
| Fréquence | Chaque sync, sans condition temporelle |
| Erreur API Xbox | `slog.Warn` + continuation (non bloquant, même pattern que `runCareerSync`) |
| Auth | XSTS token déjà présent via `e.tokens` — pas de nouveau token à gérer |
| Idempotence | `ON CONFLICT DO UPDATE SET ...` — safe à rejouer N fois |
| Définitions | Upsert à chaque sync dans `metadata.duckdb` (léger, ~100 entrées Halo Infinite) |
| Progression | Upsert à chaque sync dans `stats.duckdb` du joueur |
| pandas / SQLite | Aucun — DuckDB + `database/sql` Go uniquement |
