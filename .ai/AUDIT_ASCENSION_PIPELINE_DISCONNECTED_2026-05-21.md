# Audit — page Ascension vide malgré 700+ matchs ingérés

> Date du diagnostic : 2026-05-21
> Auteur : audit en pair avec l'utilisateur (branche `fix/media-paths-portable`)
> Statut : DIAGNOSTIQUÉ, NON CORRIGÉ — solutions documentées ci-dessous
> Réf ADR : [0014 — Progression Tracking V2 Ascension](../docs/adr/0014-progression-tracking-v2-ascension.md), [0015 — PlayerProfile Ascension V1](../docs/adr/0015-player-profile-ascension-v1.md)

---

## 1. Symptôme observé

Sur l'interface web, page `/players/<slug>/ascension` (composant [AscensionPage](../apps/web/src/features/ascension/AscensionPage.tsx)), pour les 4 joueurs configurés (JGtm, Chocoboflor, Madina97294, XxDaemonGamerxX) :

```
Ascension
Aucune streak en cours. Joue un match aujourd'hui pour démarrer une série !
Pas encore de record. Joue quelques matchs pour faire émerger tes meilleurs scores.
Aucun milestone configuré pour ce titre.
```

L'utilisateur a **769 matchs connus** (cf. log sync `match_ids connus chargés gamertag=JGtm known_count=769`) et le joueur a joué très récemment. La page devrait être peuplée. Elle ne l'est pas.

Les 3 messages correspondent aux 3 cas vides des composants frontend :
- [StreakDashboard.tsx:43](../apps/web/src/features/ascension/StreakDashboard.tsx#L43) — `items.length === 0`
- [RecordsTimeline.tsx](../apps/web/src/features/ascension/RecordsTimeline.tsx) — `personal_bests.length === 0 && history.length === 0`
- [MilestonesGrid.tsx](../apps/web/src/features/ascension/MilestonesGrid.tsx) — `items.length === 0`

---

## 2. Vérification live (serveur tournant, port 8000)

Les 3 endpoints HTTP renvoient des collections vides pour tous les joueurs :

```text
GET /api/v1/players/JGtm/milestones        → {"items":[]}
GET /api/v1/players/JGtm/streaks           → {"items":[]}
GET /api/v1/players/JGtm/records           → {"personal_bests":[],"history":[]}
GET /api/v1/players/Chocoboflor/milestones → {"items":[]}
GET /api/v1/players/Chocoboflor/streaks    → {"items":[]}
GET /api/v1/players/Chocoboflor/records    → {"personal_bests":[],"history":[]}
(idem Madina97294, XxDaemonGamerxX)
```

Les handlers ([handlers/progression.go](../apps/go-api/internal/api/handlers/progression.go)) sont en place, mountés correctement, retournent juste des tables vides. Le problème est en amont : **rien n'écrit dans les tables `streak`, `record_history`, `player_records`, `milestone_catalog`, `milestone_earned`**.

---

## 3. Architecture du pipeline V2 (rappel)

Le plan d'origine ([.ai/V7/PLAN_PROGRESSION_TRACKING_ASCENSION.md](.ai/V7/PLAN_PROGRESSION_TRACKING_ASCENSION.md)) prévoit deux choses :

### 3.1 Boot serveur (une fois)

- `migration.RunForDB(metaDB, TargetMetadata)` crée la table `milestone_catalog` (vide).
- **Étape manquante** : appeler `milestones.SyncCatalog(ctx, repo, "config/titles/halo_infinite/milestones/catalog.toml")` pour peupler la table avec les 13 milestones du TOML.

### 3.2 Hook post-sync (à chaque sync réussie)

Le pipeline V2 [EvaluateProgressionAfterSync](../apps/go-api/internal/api/post_sync_progression.go#L97) doit tourner après **chaque** sync :

1. Charge les matchs récents (120j) et le profil LUSR.
2. Évalue les streaks ([streaks.Evaluator.Evaluate](../apps/go-api/internal/progression/streaks/evaluator.go#L80)) → écrit dans `streak` (player DB).
3. Détecte les records ([records.Detector.Detect](../apps/go-api/internal/progression/records/detector.go#L92)) → écrit dans `player_records` (shared_social) et `record_history` (player DB).
4. Détecte les milestones franchis ([milestones.Detector.Detect](../apps/go-api/internal/progression/milestones/detector.go)) → écrit dans `milestone_earned` (player DB).
5. Génère les alerts du coach → émet des notifications (avec dédup 24h).

Au **premier passage**, le détecteur records crée systématiquement les PB (`current == nil → upsert`), donc une seule sync suffirait à peupler `player_records` et `record_history`.

---

## 4. Causes racines identifiées

### Cause A — `milestone_catalog` jamais peuplé au boot

**Preuve**

- [milestones.SyncCatalog()](../apps/go-api/internal/progression/milestones/catalog_loader.go#L92) existe et fonctionne (couvert par 9 tests unitaires).
- `grep -r "milestones.SyncCatalog" apps/go-api/` ne renvoie aucun call site applicatif — uniquement la définition et les tests.
- [cmd/server/main.go:549-633](../apps/go-api/cmd/server/main.go#L549-L633) applique la migration `create_milestone_catalog_metadata` mais ne charge **jamais** le TOML.
- Endpoint `/milestones` renvoie `items: []` pour tous les joueurs alors que [config/titles/halo_infinite/milestones/catalog.toml](../config/titles/halo_infinite/milestones/catalog.toml) contient 13 entrées valides.

**Effet** — Le frontend [MilestonesGrid](../apps/web/src/features/ascension/MilestonesGrid.tsx) reçoit `items: []` → affiche « Aucun milestone configuré pour ce titre ».

**Note** : le pattern correct existe déjà pour Prestige — [migration.RegisterPrestigeSeedMigration](../apps/go-api/internal/migration/steps_metadata_prestige_seed.go#L26) enregistre une migration backfill idempotente qui seed le catalogue depuis TOML. Il suffit de dupliquer ce pattern pour milestones (cf. Solution A ci-dessous).

---

### Cause B — Le hook progression n'est branché QUE sur le sync HTTP, jamais sur l'auto-sync

**Preuve**

1. Le hook [buildPostSyncDeltaHook](../apps/go-api/internal/api/post_sync_deltas.go#L33) **contient** l'appel à `EvaluateProgressionAfterSync`.
2. Ce hook est attaché **uniquement** au handler HTTP [SyncHandler](../apps/go-api/internal/api/handlers/sync_handler.go#L74) via [server.go:535](../apps/go-api/internal/api/server.go#L535) :
   ```go
   syncH = syncH.WithPostSyncDeltaHook(buildPostSyncDeltaHook(reg))
   ```
3. Le hook est invoqué seulement dans 2 endroits du `SyncHandler` :
   - [sync_handler.go:251](../apps/go-api/internal/api/handlers/sync_handler.go#L251) (endpoint `/sync`)
   - [sync_handler.go:334](../apps/go-api/internal/api/handlers/sync_handler.go#L334) (endpoint `/sync-delta`)
4. **L'auto-sync scheduler bypasse le handler HTTP** : [auto_sync.go:464](../apps/go-api/internal/scheduler/auto_sync.go#L464) appelle directement `runner.RunDelta(...)` sur le `SyncEngine` :
   ```go
   syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
   ```
5. Vérification par grep dans `internal/scheduler/` et `internal/sync/` :
   ```
   grep -rE "progression|EvaluateProgressionAfterSync|EmitPostSyncDeltas|SnapshotPlayerState" internal/{scheduler,sync}/
   → No matches found
   ```
6. Les logs `logs/sync.log` du serveur en cours montrent les étapes post-sync invoquées par l'auto-sync (`skill self-heal`, `events self-heal`, etc.) mais **aucune mention de `progression`, `streaks`, `records.Detect` ou `milestones`** depuis le boot.

**Effet** — Étant donné que `spnkr_auto_sync_enabled=true` avec `spnkr_auto_sync_interval_minutes=15` ([app_settings.json:35-37](../app_settings.json#L35)), 100 % des syncs en condition réelle passent par l'auto-sync. Le pipeline V2 n'a jamais l'occasion de tourner. Les tables `streak`, `player_records`, `record_history`, `milestone_earned` restent vides indéfiniment.

**Impact étendu (au-delà d'Ascension)**

Le hook `buildPostSyncDeltaHook` contient aussi [EmitPostSyncDeltas](../apps/go-api/internal/api/post_sync_deltas.go#L346), qui est responsable des notifications post-sync :
- `career_rank` (nouveau rang lifetime Halo)
- `skill_tier` (CSR/LUSR transition)
- `objective_completed`, `objective_assigned`
- `challenge_completed`, `challenge_added`
- `citation_tier`, `citation_mastery`
- `battlepass_completed`
- `threshold_crossed` (KD ratio, winrate)
- `personal_record` (best_kda)

**Aucune de ces notifications n'est émise depuis l'auto-sync.** Le bug Ascension est la pointe de l'iceberg — toute la couche notifications post-sync est désactivée en condition réelle.

---

### Cause C (atténuante) — Sémantique « bucket courant » des streaks (comportement attendu)

Même si la Cause B est corrigée, les **streaks ne se rétroactivent pas** sur l'historique. L'évaluateur [streaks/evaluator.go:100-112](../apps/go-api/internal/progression/streaks/evaluator.go#L100-L112) vérifie uniquement que le **bucket courant** (aujourd'hui pour `daily_*`, cette semaine pour `weekly_*`) contient au moins un match satisfaisant pour démarrer une streak. Cohérent avec le message UI « Joue un match aujourd'hui pour démarrer une série ».

**Conséquence pratique** : après application des fixes A + B, l'utilisateur verra :
- **milestones** : peuplés immédiatement (catalogue + détection earned à la prochaine sync).
- **records** : peuplés à la prochaine sync (premier passage initialise tous les PB).
- **streaks** : restent à 0 jusqu'à ce que l'utilisateur joue *et* déclenche une sync **dans le bucket courant**.

Ce point n'est pas un bug, c'est le design intentionnel. Le mentionner dans la PR pour éviter une re-création de ticket.

---

### Cause D (atténuante) — Seuil minimum pour records

[records.MinMatchesForRecord = 10](../apps/go-api/internal/progression/records/types.go#L41) sur une fenêtre de 120 jours ([ProgressionMatchHistoryDays](../apps/go-api/internal/api/post_sync_progression.go#L46)). Pour un compte avec <10 matchs récents (par ex. compte ami inactif), le détecteur retourne sans rien écrire — comportement attendu. Vérifier en prod que les comptes actifs ont ≥10 matchs sur 120j (a priori OK pour JGtm avec 769 matchs total).

---

## 5. Solutions recommandées

### Solution A — Peupler `milestone_catalog` au boot

**Approche recommandée : migration backfill idempotente** (pattern identique à Prestige).

Créer `apps/go-api/internal/migration/steps_metadata_milestones_seed.go` :

```go
package migration

import (
    "context"
    "database/sql"

    "levelup/go-api/internal/platform/duckdb"
    "levelup/go-api/internal/progression/milestones"
)

const milestonesSeedMigrationName = "seed_milestone_catalog_v1"

// RegisterMilestonesSeedMigration enregistre la migration backfill du
// catalogue milestones depuis TOML. Idempotente.
//
// catalogPath : chemin vers config/titles/{slug}/milestones/catalog.toml
// Doit être appelé avant RunForDB(_, TargetMetadata).
func RegisterMilestonesSeedMigration(catalogPath string) {
    for _, m := range registry {
        if m.Name == milestonesSeedMigrationName {
            return
        }
    }
    Register(Migration{
        Name:        milestonesSeedMigrationName,
        TargetDB:    TargetMetadata,
        Description: "Seed milestone_catalog depuis TOML config",
        ApplySchema: func(db *sql.DB) error { return nil },
        ApplyBackfill: func(db *sql.DB) error {
            // wrapper *sql.DB → duckdb.DB minimal pour appeler le repo
            wrappedDB := duckdb.NewFromSQLDB(db) // helper à créer ou adapter
            repo := duckdb.NewMilestoneCatalogRepo(wrappedDB)
            return milestones.SyncCatalog(context.Background(), repo, catalogPath)
        },
    })
}
```

**Câblage dans `cmd/server/main.go`** (après le block prestige seed, ligne ~556) :

```go
// Seed milestone catalog (idempotent, identique au pattern Prestige).
milestonesCatalogPath := filepath.Join(prestigeConfigDir, "milestones", "catalog.toml")
migration.RegisterMilestonesSeedMigration(milestonesCatalogPath)
```

**Variante simpler — appel direct hors migration** : appeler `milestones.SyncCatalog` directement après `migration.RunForDB(metaDB, TargetMetadata)`. Plus simple à coder mais non-idempotent au sens migration (réexécution à chaque boot). C'est OK car `SyncCatalog` utilise `UPSERT`, donc benigne — mais le pattern migration garde la trace dans `_migrations`.

**Recommandation** : pattern migration pour rester cohérent avec Prestige et bénéficier du logging migration.

**Tests à ajouter**

- `steps_metadata_milestones_seed_test.go` : TestSeedMilestonesCatalog_Idempotent (lance 2× → 13 lignes une seule fois).
- `steps_metadata_milestones_seed_test.go` : TestSeedMilestonesCatalog_MissingTOML_LogsWarn (TOML absent → pas de crash).

---

### Solution B — Brancher le pipeline V2 sur l'auto-sync

Trois options, par ordre de préférence :

#### Option B1 (recommandée) — Pipeline V2 dans `SyncEngine.runPostSyncPipeline`

Déplacer l'orchestration `EvaluateProgressionAfterSync` + `EmitPostSyncDeltas` **dans** le sync engine. Ainsi toute sync réussie (HTTP manuelle OU auto-sync) déclenche le pipeline.

**Étapes** :

1. Ajouter un nouveau champ à `SyncEngine` :
   ```go
   type SyncEngine struct {
       // ... existant
       postSyncProgression api.PostSyncProgressionRunner // ou interface équivalente
   }
   ```

2. Définir une interface dans `internal/sync/` ou un sous-package neutre :
   ```go
   type PostSyncProgressionRunner interface {
       Run(ctx context.Context, slug string) // capture before, lance after
   }
   ```

3. Adapter `buildPostSyncDeltaHook` pour qu'il implémente cette interface.

4. Wirer dans `server.go` au moment de la construction des `SyncEngine` (au lieu de `WithPostSyncDeltaHook` sur le handler) :
   ```go
   engine.WithPostSyncProgression(runner)
   ```

5. Dans `runPostSyncPipeline` (ou à la fin de `SyncEngine.run`), invoquer `e.postSyncProgression.Run(ctx, slug)` après le pipeline existant.

6. Retirer le câblage `WithPostSyncDeltaHook` du `SyncHandler` HTTP (les 2 sites deviennent inutiles).

**Avantages**

- Une seule source de vérité (le sync engine).
- Auto-sync, manual HTTP, scripts CLI — tous bénéficient.
- Élimine la duplication potentielle entre HTTP et scheduler.

**Inconvénients**

- Refactor non-trivial : `SyncEngine` n'a actuellement aucune dépendance vers `internal/api`. Il faut soit déplacer le pipeline (le sortir de `internal/api/post_sync_*`), soit créer une interface partagée dans un package neutre (ex: `internal/port` ou `internal/progression/orchestrator`).
- Touche le path critique de sync. Tests E2E à mettre à jour.

**Tests à ajouter**

- `engine_e2e_test.go` : nouveau test `TestRunDelta_TriggersProgressionPipeline` qui mocke `PostSyncProgressionRunner` et vérifie qu'il est appelé.
- `auto_sync_e2e_test.go` : vérifier qu'une sync auto déclenche bien le runner injecté.

---

#### Option B2 — Hook similaire sur `AutoSyncScheduler`

Exposer une méthode `WithPostSyncHook(...)` sur `AutoSyncScheduler` analogue à `SyncHandler.WithPostSyncDeltaHook`. Le scheduler appelle le hook après chaque `RunDelta` réussi.

**Étapes**

1. Ajouter au scheduler :
   ```go
   type postSyncHook = func(ctx context.Context, slug string) func(ctx context.Context)

   func (s *AutoSyncScheduler) WithPostSyncHook(h postSyncHook) *AutoSyncScheduler {
       s.postSync = h
       return s
   }
   ```

2. Dans `runSyncForPlayer` (autour de la ligne 464), appeler le hook avant/après `RunDelta` :
   ```go
   var after func(ctx context.Context)
   if s.postSync != nil {
       after = s.postSync(ctx, p.PlayerSlug) // capture before
   }
   syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
   if err == nil && after != nil {
       after(ctx) // évalue après
   }
   ```

3. Wirer dans `server.go` :
   ```go
   scheduler.WithPostSyncHook(buildPostSyncDeltaHook(reg))
   ```

**Avantages**

- Refactor minimal, ne touche pas au sync engine.
- Réutilise le hook existant.

**Inconvénients**

- Duplication : le câblage est en deux endroits (HTTP + scheduler). Si un 3e point d'entrée est ajouté (CLI `levelup sync-delta`), il faudra encore brancher.
- Risque d'oubli sur les futurs paths de sync.

**Tests à ajouter**

- `auto_sync_test.go` : `TestAutoSync_InvokesPostSyncHook` (mock du hook).

---

#### Option B3 — Brancher le hook sur le CLI aussi

[cmd/levelup/cmd_sync_delta.go](../apps/go-api/cmd/levelup/) — si l'utilisateur utilise `levelup sync-delta`, le hook devrait aussi tourner. Sinon on a 3 paths divergents (HTTP, auto-sync, CLI).

**Recommandation finale** — Combiner B1 + audit des autres entry points (CLI) pour s'assurer que **tous** les chemins de sync déclenchent le pipeline V2. B2 acceptable comme remédiation rapide si B1 est trop lourd.

---

## 6. Plan de remédiation suggéré

### Étape 1 (urgent, ~30min)

- Solution A : créer `RegisterMilestonesSeedMigration` + l'appeler dans `runMigrations()`.
- Tests : 2 unit (idempotence + TOML absent).
- Vérifier : redémarrer le serveur, `curl /milestones` → 13 items.

### Étape 2 (priorité haute, ~2-4h)

- Solution B2 (rapide) ou B1 (propre — préférer si on a le temps).
- Tests E2E : auto-sync déclenche bien le pipeline.
- Vérifier en condition réelle : attendre 1 cycle auto-sync (15min), `curl /records` → devrait avoir des PB initialisés.

### Étape 3 (validation)

- Logs : grep `progression: ` doit montrer les étapes du pipeline après chaque sync.
- Endpoint `/milestones` : 13 items, certains avec `earned: true` pour JGtm (qui a 769 matchs → centurion/vétéran/elite atteints).
- Endpoint `/records` : `personal_bests` non vide après 1 cycle.
- Endpoint `/streaks` : reste vide tant qu'aucun match n'est joué dans le bucket courant (comportement attendu, ne pas chercher à fixer).

### Étape 4 (suivi)

- Auditer les autres entry points de sync (CLI, scripts admin) → s'assurer qu'ils déclenchent aussi le pipeline si pertinent.
- Ajouter un test de garde : `TestAssertProgressionDeps_HookWiredOnAllSyncPaths` pour prévenir une régression future.
- Mettre à jour [docs/adr/0014-progression-tracking-v2-ascension.md](../docs/adr/0014-progression-tracking-v2-ascension.md) avec une note « Wiring : voir AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.md ».

---

## 7. Commandes utiles pour reprendre le diagnostic

### Vérifier l'état des endpoints

```bash
for slug in JGtm Chocoboflor Madina97294 XxDaemonGamerxX; do
  echo "=== $slug ==="
  curl -s http://localhost:8000/api/v1/players/$slug/milestones | head -c 200; echo
  curl -s http://localhost:8000/api/v1/players/$slug/streaks    | head -c 200; echo
  curl -s http://localhost:8000/api/v1/players/$slug/records    | head -c 300; echo
done
```

### Vérifier la table `milestone_catalog` (nécessite serveur arrêté ou copie de la DB)

```sql
ATTACH 'data/titles/halo_infinite/warehouse/metadata.duckdb' AS meta;
SELECT title_slug, COUNT(*) FROM meta.milestone_catalog GROUP BY title_slug;
-- Attendu post-fix A : title_slug='halo_infinite', count=13
-- Actuel : ligne vide / 0
```

### Vérifier qu'une sync déclenche bien le pipeline (post-fix B)

```bash
# Forcer une sync manuelle HTTP (déjà câblée, devrait déjà fonctionner)
curl -X POST http://localhost:8000/api/v1/players/JGtm/sync-delta

# Tail les logs côté progression
tail -f logs/sync.log logs/notifications.log | grep -iE 'progression|streaks|records|milestones'
```

### Diagnostic ad-hoc Go (template à recréer si besoin)

> Le script `cmd/diag_ascension/main.go` créé lors du diagnostic 2026-05-21 a été supprimé après usage. Il interrogeait les 3 tables (`milestone_catalog`, `streak`, `player_records`) pour chaque joueur. Le pattern (boilerplate DuckDB read-only multi-DB) est identique à [cmd/diag_lying_bits/main.go](../apps/go-api/cmd/diag_lying_bits/main.go). Recréer si besoin pour confirmation hors-ligne. **Limitation Windows** : impossible d'ouvrir les DB en RO tant que le serveur tient un lock RW → soit arrêter le serveur, soit faire le diag via les endpoints HTTP (comme ci-dessus).

---

## 8. Points d'attention pour le reviewer

- **Pas une nouvelle feature, c'est un bug de câblage**. Le code V2 (streaks/records/milestones/coach) est complet, testé, fonctionnel. Le seul problème est qu'il n'est jamais invoqué en condition réelle.
- **Régression silencieuse au-delà d'Ascension** : toutes les notifications post-sync (career_rank, skill_tier, citation_tier, threshold_crossed, etc.) sont aussi désactivées. Vérifier la table `notifications` dans `shared_social` — elle devrait montrer un trou depuis le déploiement V2 (sauf si l'utilisateur a déclenché des syncs manuelles via le bouton UI).
- **Multi-titres** : la solution A doit prendre le `prestigeConfigDir` global ou exposer un mapping par `title_slug`. Actuellement le projet est mono-titre (halo_infinite), mais l'ADR 0008 prévoit le multi-titres. Préférer la signature `RegisterMilestonesSeedMigration(configRoot string)` qui itère sur les titres connus, plutôt qu'une seule path en dur.
- **Idempotence** : `milestones.SyncCatalog` utilise `INSERT ... ON CONFLICT DO UPDATE` (cf. [milestones_catalog_repo.go:41](../apps/go-api/internal/platform/duckdb/milestones_catalog_repo.go#L41)). Sûr de réexécuter à chaque boot — pas de risque de doublons.

---

## 9. Annexe — extraits de code des principaux acteurs

### Le hook V2 (à câbler à toutes les paths de sync)

[apps/go-api/internal/api/post_sync_deltas.go:33-75](../apps/go-api/internal/api/post_sync_deltas.go#L33) :

```go
func buildPostSyncDeltaHook(reg *ServiceRegistry) handlers.PostSyncDeltaHook {
    return func(ctx context.Context, slug string) func(ctx context.Context) {
        pdb, err := reg.resolve(ctx, slug)
        // ... snapshot before
        return func(ctx context.Context) {
            // ... snapshot after, emit deltas, EvaluateProgressionAfterSync
        }
    }
}
```

### Le call site HTTP (déjà câblé)

[apps/go-api/internal/api/server.go:535](../apps/go-api/internal/api/server.go#L535) :

```go
syncH = syncH.WithPostSyncDeltaHook(buildPostSyncDeltaHook(reg))
```

### Le call site auto-sync (manquant)

[apps/go-api/internal/scheduler/auto_sync.go:464](../apps/go-api/internal/scheduler/auto_sync.go#L464) — appelle `runner.RunDelta` sans hook :

```go
syncResult, err := runner.RunDelta(ctx, domain.DefaultSyncOptions())
// → aucune invocation du pipeline V2 ici
```

### Le SyncCatalog manquant

[apps/go-api/internal/progression/milestones/catalog_loader.go:89-106](../apps/go-api/internal/progression/milestones/catalog_loader.go#L89) :

```go
func SyncCatalog(ctx context.Context, repo CatalogRepo, path string) error {
    entries, err := LoadCatalogFromFile(path)
    // ... upserts each entry
}
// Fonction existante, jamais appelée applicativement.
```

---

## 10. Glossaire rapide pour le reviewer

| Terme | Sens dans ce contexte |
|---|---|
| **Pipeline V2** | Orchestrateur `EvaluateProgressionAfterSync` qui enchaîne streaks → records → milestones → coach (cf. ADR 0014). |
| **Hook post-sync delta** | Closure `buildPostSyncDeltaHook` qui capture snapshot avant + après sync pour émettre les notifications de progression. Contient le pipeline V2. |
| **Auto-sync** | Scheduler interne ([AutoSyncScheduler](../apps/go-api/internal/scheduler/auto_sync.go)) qui tourne toutes les `spnkr_auto_sync_interval_minutes` (15 min par défaut). |
| **Bucket courant** | Tranche temporelle de la streak (jour ou semaine UTC). Une streak ne démarre que si le bucket courant contient un match satisfaisant. |
| **player_records** | Table dans `shared_social.duckdb`, stocke les PB par `(xuid, metric)`. |
| **milestone_catalog** | Table dans `metadata.duckdb`, référentiel des milestones par titre, chargée depuis TOML. |
| **streak** | Table dans `stats.duckdb` (player DB), stocke les streaks actives + historique par joueur. |

---

Fin du document. Pour toute question, voir la conversation Claude du 2026-05-21 ou l'entrée correspondante dans `.ai/thought_log.md`.
