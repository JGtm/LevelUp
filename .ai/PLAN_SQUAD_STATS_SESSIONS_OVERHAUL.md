# Plan — Améliorations pages Escouade + Stats solo

**Branche cible** : `feat/foundations-axes-1-3-4` (branche courante au moment du plan, conformément à la stratégie 1 tâche = 1 branche, N commits). Tout le lot est livré sur cette branche, en plusieurs commits ordonnés (backend DTO/service → repo Q29 → recompute → frontend Squad → frontend Stats).

## Context

La page Escouade (`/players/$playerSlug/squad/`) souffre de plusieurs incohérences UX/fonctionnelles identifiées en revue :

1. La liste déroulante de sessions inclut les sessions solo (rôle dévolu à la page Stats).
2. Pas de multi-sélection de sessions, alors qu'un gros corpus rend la sélection unique pénalisante.
3. Tri non garanti chronologique décroissant.
4. Le label « parmi 50 coéquipiers » est trompeur : Q29 inclut tous les coéquipiers vus en match `is_with_friends=TRUE`, y compris des randoms ; ces derniers n'ont ni DB ni `performance_score`.
5. Périmètre des filtres NavL2 vs filtre session ambigu : on ne sait pas si la période filtre les sessions du dropdown, les matchs analysés, ou les deux.

But : que la page Escouade ne montre que des sessions squad, avec une sélection multi-sessions confortable même sur gros corpus, et un dropdown coéquipiers limité aux **amis** (joueurs ayant une DB). Pour un coéquipier hors top 50, proposer un flux explicite "Ajouter comme ami + sync". La page Stats solo bénéficie en miroir des mêmes améliorations (multi-select sessions, tri chrono, filtre interne) et du recompute rétroactif `is_with_friends`.

---

## Plan d'implémentation

### 1. Backend — sessions squad-only + multi-sélection

Fichiers :
- [apps/go-api/internal/domain/teammates.go](../apps/go-api/internal/domain/teammates.go)
- [apps/go-api/internal/service/teammates_service.go](../apps/go-api/internal/service/teammates_service.go)
- [apps/go-api/internal/platform/duckdb/queries_squad.go](../apps/go-api/internal/platform/duckdb/queries_squad.go)

Changements :
- `TeammatesQueryRequest.PickedSquadSession *string` → `PickedSquadSessions []string`. Idem `PickedSoloSession` (cohérence). DTO TS aligné dans `apps/web/src/lib/api/types.ts`.
- `filterSynthesisBySession()` (teammates_service.go:123-150) : remplacer match-label-unique par `slices.Contains(picked, label)`.
- Q29 (queries_squad.go:7-31) : restreindre aux **amis** uniquement. Joindre sur `app_settings.friend_gamertags` (résolus via `xuid_aliases`) pour ne lister dans le top que des joueurs ayant une DB. Renvoyer aussi le compte total d'amis dans la réponse pour le label UI.
- `SessionLabelsList` : trier `squad[]` par `started_at DESC` côté repo (vérifier la requête sessions). Ajouter `started_at` + `ended_at` dans la struct retournée pour permettre le mini-filtre période côté client.

Tests : `apps/go-api/internal/service/teammates_service_test.go` existant — étendre avec cas multi-labels et cas friends-only.

### 2. Frontend — dropdown sessions multi-select avec mini-filtre période

Fichiers :
- [apps/web/src/features/squad/SquadLayout.tsx](../apps/web/src/features/squad/SquadLayout.tsx) (lignes 269-289 surtout)
- Réutiliser `apps/web/src/components/ui/GamertagCombobox.tsx` comme base de pattern (multi-select + fuzzy search + max limit + couleurs).

Changements :
- Créer `apps/web/src/components/ui/SessionMultiSelect.tsx` (composant **partagé** Squad + Stats) — dérivé du pattern Combobox :
  - Input texte (filtre fuzzy sur le label session)
  - Mini date-range picker intégré (n'affecte QUE le filtrage interne du dropdown, pas le `globalFilterStore`)
  - Liste virtuelle de checkboxes (sessions filtrées par texte + période interne)
  - Tri par `started_at DESC` par défaut
  - Bouton "Valider" qui propage la sélection au parent (deferred validation)
- SquadLayout : remplacer `pickedSquadSessionLabel` (string) par `pickedSquadSessionLabels` (string[]) ; alimenter le composant à partir de `SessionLabelsList.squad` uniquement (les solo restent à la page Stats).
- Persistance localStorage : `squad-sessions-{playerSlug}` (multi).

### 3. Frontend — top 50 → "amis" + flux d'ajout

Fichiers :
- [apps/web/src/features/squad/SquadLayout.tsx](../apps/web/src/features/squad/SquadLayout.tsx)
- [apps/go-api/internal/api/handlers/setup.go](../apps/go-api/internal/api/handlers/setup.go)
- [apps/go-api/internal/service/profile_service.go](../apps/go-api/internal/service/profile_service.go)
- [apps/web/src/features/settings/SettingsPage.tsx](../apps/web/src/features/settings/SettingsPage.tsx) (réutiliser `useUpdateSettings`)

Changements :
- Label dropdown : "parmi N amis" (au lieu de "50 coéquipiers"), avec N = `friend_gamertags.length` retourné par le backend.
- Quand l'utilisateur tape un gamertag absent du top : proposer une carte "Ajouter X comme ami et lancer la sync" → ouvre une **modale de confirmation** indiquant que le sync initial peut prendre du temps selon le corpus du joueur (texte i18n FR + EN). Sur confirmation :
  - Appel POST `/setup/players` (`CreatePlayerProfileRequest`) — déjà en place.
  - Settings PATCH pour ajouter à `friend_gamertags`.
  - Lancement du sync initial (handler existant).
  - Toast progress + invalidation des queries Squad.
- Pas de hook auto-create dans Settings PATCH (option B retenue) — l'ajout reste explicite et tracé.

#### 3bis. Mutualisation du flux d'ajout sur la page Settings → Utilisateurs

Fichier : [apps/web/src/features/settings/SettingsPage.tsx](../apps/web/src/features/settings/SettingsPage.tsx) (section `FriendGamertagesSection` lignes 822-848).

Aujourd'hui cette section permet d'ajouter un gamertag dans `friend_gamertags` via PATCH settings, mais ne crée **pas** de profil DuckDB et ne déclenche pas de sync. Il faut aligner ce flux sur celui créé en §3 :

- Extraire le composant créé en §3 (`AddFriendModal` + carte CTA "Ajouter X comme ami et lancer la sync") dans `apps/web/src/features/friends/AddFriendFlow.tsx` (composant **partagé** Squad + Settings).
- Sur Settings : remplacer la mutation directe `friend_gamertags` par l'appel au flux mutualisé. Même UX (modale de confirmation + message de durée + toast progress + invalidation queries).
- Le backend ne change pas : un seul handler dorsal (`POST /friends/add` ou équivalent introduit en §3) sert les deux pages.

Bénéfice : un seul code path, un seul ensemble de tests UI, comportement identique côté backend (recompute §4 + notifications §7 déclenchés indépendamment du point d'entrée UI).

### 4. Backend — recompute `is_with_friends` (initial + incrémental sur ajout d'ami)

**Constat issu de l'audit (à acter)** :
- `is_with_friends` est aujourd'hui **non écrit par le sync Go** ([apps/go-api/internal/sync/writes.go](../apps/go-api/internal/sync/writes.go) `UpsertPlayerEnrichment` ne touche pas la colonne ; défaut schema = FALSE dans [sync/schema.go:40](../apps/go-api/internal/sync/schema.go#L40) et [migration/steps_player.go:26](../apps/go-api/internal/migration/steps_player.go#L26)). Les données existantes proviennent du sync Python historique.
- La vue matérialisée `mv_player_matches` exporte `is_with_friends` ([sync/aggregates.go:30](../apps/go-api/internal/sync/aggregates.go#L30)) → tout UPDATE rend la vue stale.
- L'enregistrement de plusieurs joueurs/titres se fait via `db_profiles.json` (title-scoped) alors que `friend_gamertags` est player-global → itérer toutes les player DBs (toutes (title, gamertag) combos).
- Lease lock concurrentiel disponible via `AcquireLeaseCtx(ctx, dbPath)` ([sync/engine.go:113](../apps/go-api/internal/sync/engine.go#L113)).

**Conséquence** : la couche n'est pas `service/` + `platform/duckdb/` mais bien `internal/sync/`, qui contient déjà `AcquireLeaseCtx`, `refreshAggregates` (unexported) et le pattern `session_recalc.go`. On suit ce pattern à la lettre.

**Fichiers** :
- Nouveau : `apps/go-api/internal/sync/friends_recompute.go` — fonction principale `RecomputeIsWithFriends(ctx, playerDBPath, sharedDBPath, friendXUIDs []string) (RecomputeResult, error)`. Pattern miroir de `session_recalc.go`.
- Nouveau : `apps/go-api/internal/service/friends_orchestrator_service.go` — orchestration multi-DB : énumère via `cfg.LoadPlayers()` ([cmd/levelup/cmd_sync.go:103](../apps/go-api/cmd/levelup/cmd_sync.go#L103)) + `PathResolver.PlayerDBPath(slug, gamertag)`, appelle `sync.RecomputeIsWithFriends` pour chaque DB. Retourne un agrégat `{processed, failed, totalRowsUpdated}`.
- Port : `apps/go-api/internal/port/services.go` — interface `FriendsOrchestratorService`.
- Modifié : `apps/go-api/internal/api/handlers/settings.go` PATCH handler — détecter le diff de `friend_gamertags`, appeler `FriendsOrchestratorService.OnFriendsChanged(ctx, addedXUIDs)` en **async** (pattern existant : voir `PostRecalculateSessions` dans le même fichier).
- Modifié : `apps/go-api/internal/sync/aggregates.go` — exporter `RefreshAggregates` (capital R) ou exposer un wrapper public ; documenter l'expansion d'API.

**Logique SQL par player DB** (résolution XUID amont, en mémoire, via une seule requête `shared.xuid_aliases`) :
```sql
UPDATE player_match_enrichment
SET is_with_friends = TRUE
WHERE is_with_friends = FALSE
  AND match_id IN (
    SELECT match_id FROM shared.match_participants
    WHERE xuid IN (?, ?, ...)  -- liste des XUIDs amis pré-résolus
  );
```

**Étapes ordonnées par player DB** :
1. `relPlayer, err := sync.AcquireLeaseCtx(ctx, playerDBPath)` — bloquer le sync concurrent. `defer relPlayer()`.
2. `relShared, err := sync.AcquireLeaseCtx(ctx, sharedDBPath)` (READ-ONLY suffit côté shared : on n'écrit que dans player DB ; le lease shared est défensif si un sync est en cours d'écriture).
3. ATTACH `shared_matches_v2.duckdb` AS `shared` (read_only).
4. Résoudre XUIDs amis (un seul SELECT in shared.xuid_aliases) ; logger les gamertags non résolus en Warn (pas une erreur).
5. `BEGIN TRANSACTION` côté player DB.
6. UPDATE atomique avec liste de XUIDs.
7. `COMMIT`. Capturer `rowsAffected`.
8. Si `rowsAffected > 0` : `sync.RefreshAggregates(playerDB)` pour rebuild `mv_player_matches`.
9. DETACH shared. Release leases.

**Modes d'invocation** :
- **Bootstrap initial** (one-shot) : nouvelle commande CLI `cmd/levelup/cmd_recompute_friends.go` qui appelle `FriendsOrchestratorService.RecomputeAll(ctx)` avec **tous** les `friend_gamertags` actuels. À exécuter une fois au déploiement, ou idempotemment au démarrage du serveur si flag de version détecté. Documenté dans `.ai/thought_log.md` au moment de la livraison.
- **Incrémental** (PATCH /settings) : sur diff des `friend_gamertags`, déclencher `OnFriendsChanged(ctx, addedXUIDs)` async. La garde `is_with_friends = FALSE` rend l'opération idempotente — relancer 2× est sans effet.

**Rétro-compat sync Go** : ouvrir un follow-up (hors scope ce lot) pour que `UpsertPlayerEnrichment` ([sync/writes.go:156](../apps/go-api/internal/sync/writes.go#L156)) calcule `is_with_friends` à l'écriture initiale (intersection xuid match ↔ friend_gamertags). Sans ça, tout nouveau match ingéré reste FALSE jusqu'au prochain trigger. Une note `// TODO(friends-recompute): compute is_with_friends inline` est ajoutée dans `writes.go` pour traçabilité.

**Multi-titres** :
- `cfg.LoadPlayers()` retourne déjà des entrées title-aware via `db_profiles.json`. Aucun branchement par slug ; on itère ce que la config liste.
- `PathResolver.PlayerDBPath(slug, gamertag)` résout les chemins (pas de `filepath.Join` direct). Conformité PathResolver respectée.
- Aucune capability check requise : `is_with_friends` est défini sur tous les titres (pas de TitleDataAdapter dépendant).

**Gestion des erreurs / atomicité** :
- Per-DB transaction : `BEGIN/COMMIT` par player DB. Une DB qui échoue ne bloque pas les autres ; l'orchestrateur logue l'échec et continue.
- Idempotence : la garde `is_with_friends = FALSE` rend tout retry safe.
- Réponse HTTP : 202 Accepted avec un job ID si async (à coupler au pattern `PostRecalculateSessions` déjà en place).

**Couches respectées (arch-rules)** :
- `internal/sync/` : SQL + lease + refresh aggregates (cohérent avec engine, session_recalc, backfill_weapons).
- `internal/service/` : orchestration multi-DB + exposition au port, sans SQL direct.
- `internal/api/handlers/` : déclenchement, pas de logique métier.
- `internal/port/` : interface `FriendsOrchestratorService`.

### 5. Frontend — page Stats solo : symétrie + nettoyage

Fichiers (à confirmer pendant l'implémentation, route Stats à identifier — probablement `apps/web/src/features/stats/` ou `routes/players/$playerSlug/stats/`).

Changements miroir Squad :
- Dropdown sessions alimenté **uniquement** par `SessionLabelsList.solo` (les sessions squad disparaissent → cohérent avec le rôle "Stats = solo, Squad = squad").
- Réutiliser le composant `SessionMultiSelect` créé en §2 (multi-sélection + filtre fuzzy + mini date-range + tri chrono décroissant + persistance localStorage `stats-sessions-{playerSlug}`).
- Côté backend : si la page Stats consomme un endpoint analogue à teammates avec un `PickedSoloSession *string`, le passer en `PickedSoloSessions []string` symétrique. Sinon (filtrage côté client à partir de `SessionLabelsList.solo`), juste exploiter le tableau.

Bénéfice automatique du recompute (§4) : les sessions improprement classées "solo" (ami non détecté à l'époque) basculent en `squad[]` et sortent de la page Stats sans action supplémentaire.

Hors scope Stats : pas de top coéquipiers, pas de flux ajout ami (Stats = solo).

### 6. Notifications — émission lors de l'ajout d'ami et de la fin de sync

L'infrastructure de notifications existe déjà ([apps/go-api/internal/notifications/types.go](../apps/go-api/internal/notifications/types.go)) : catégories typées, severities (info/success/warn/error), delivery channels (`toast`/`inapp`/`both`/`off`), préférences par utilisateur seedées via [migration/steps_player_notifications.go](../apps/go-api/internal/migration/steps_player_notifications.go), service + emitter ([notifications/service.go](../apps/go-api/internal/notifications/service.go), [notifications/emitter.go](../apps/go-api/internal/notifications/emitter.go)). On l'**étend**, on ne réinvente rien.

**2 nouvelles catégories** à ajouter dans `notifications/types.go` :
- `CategoryFriendAdded` (`"friend_added"`) — émise quand le PATCH friend_gamertags se termine et que le profil DuckDB est créé. Severity = `success`. Title i18n : "Ami ajouté" / "Friend added".
- `CategoryFriendSyncCompleted` (`"friend_sync_completed"`) — émise quand le sync initial du nouvel ami **et** le recompute `is_with_friends` ont terminé sur toutes les player DBs. Severity = `success`. Title i18n : "Sync de {gamertag} terminée" / "{gamertag} sync completed".

**Émission** :
- `friends_orchestrator_service.go` (§4) appelle `notifications.Emitter.Emit(ctx, CategoryFriendAdded, payload)` après création de profil + PATCH settings réussi.
- À la fin du sync initial du nouvel ami (callback dans `sync.Engine` ou hook équivalent), émettre `CategoryFriendSyncCompleted` avec en payload `{gamertag, matches_synced, recompute_rows_affected}`.

**Frontend** :
- Ajouter les 2 clés i18n FR + EN dans [apps/web/src/features/notifications/i18n.ts](../apps/web/src/features/notifications/i18n.ts) : `notif.friend_added.title`, `notif.friend_added.body`, `notif.friend_sync_completed.title`, `notif.friend_sync_completed.body`.
- **Toggles toast/in-app** : aucune modification UI nécessaire. La page [NotificationsSettingsTab.tsx](../apps/web/src/features/notifications/NotificationsSettingsTab.tsx) énumère dynamiquement toutes les catégories retournées par `AllCategories()` côté Go ; les 2 nouvelles apparaissent automatiquement avec leur sélecteur delivery (toast / inapp / both / off) et leur severity. Defaults : delivery=`both`, severity=`success`.
- **Seed des préférences** : étendre [migration/steps_player_notifications.go](../apps/go-api/internal/migration/steps_player_notifications.go) pour insérer les 2 nouvelles catégories dans la table de préférences au prochain démarrage. Migration idempotente (`INSERT OR IGNORE` ou équivalent existant).
- Drawer in-app et toast bridge prennent en charge sans modif (rendu générique via `format.ts` + `icons.tsx`). Vérifier qu'une icône par défaut est définie pour les 2 catégories sinon ajouter dans `icons.tsx`.

**Discord — inclus dans le scope (infra existante)** :

L'infrastructure Discord existe dans [apps/go-api/internal/notify/](../apps/go-api/internal/notify/) (à ne pas confondre avec `internal/notifications/` user-facing) : `NotifyConfig` + `LoadNotifyConfig(settingsPath)` lisent `discord_webhook_url` depuis `app_settings.json` ([notify/discord.go:60-112](../apps/go-api/internal/notify/discord.go#L60-L112)) avec flags par catégorie. Pattern : `NotifyXxx(cfg, ...)` failsafe (recover panic, jamais d'erreur propagée), short-circuit si webhook vide ou flag off ([notify/notifiers.go](../apps/go-api/internal/notify/notifiers.go)).

**Ajouts** :
- Dans `notify/discord.go` `NotifyConfig` : 2 flags `NotifyFriendAdded bool`, `NotifyFriendSyncCompleted bool`. Chargés depuis `app_settings.json` via les clés `discord_notify_friend_added` (défaut true), `discord_notify_friend_sync_completed` (défaut true).
- Dans `notify/notifiers.go` : 2 nouvelles fonctions failsafe :
  - `NotifyFriendAdded(cfg NotifyConfig, gamertag string, addedBy string)` — embed succinct (titre, gamertag, timestamp).
  - `NotifyFriendSyncCompleted(cfg NotifyConfig, gamertag string, matchesSynced int, rowsAffected int64, duration time.Duration)` — embed avec stats.
- Dans `notify/embeds.go` : 2 builders `BuildFriendAddedEmbed` + `BuildFriendSyncCompletedEmbed` (i18n FR/EN via `cfg.Lang`).
- Frontend Settings : ajouter 2 `ToggleRow` dans la section Discord existante de [SettingsPage.tsx](../apps/web/src/features/settings/SettingsPage.tsx) (lignes 240-242, à côté de `discord_notify_sync`/`discord_notify_backfill`/`discord_notify_new_media`) :
  - `<ToggleRow label={t.discordNotifyFriendAdded} value={merged.discord_notify_friend_added ?? true} ... disabled={!merged.discord_notifications_enabled} />`
  - `<ToggleRow label={t.discordNotifyFriendSyncCompleted} value={merged.discord_notify_friend_sync_completed ?? true} ... disabled={!merged.discord_notifications_enabled} />`
  - Master switch `discord_notifications_enabled` continue de désactiver tout. Cohérence avec le pattern existant.
- 2 nouvelles clés i18n FR + EN dans le fichier i18n des settings : `discordNotifyFriendAdded`, `discordNotifyFriendSyncCompleted`.

**Émission** :
- `friends_orchestrator_service.go` (§4) appelle `notify.NotifyFriendAdded(cfg, gt, "")` après création de profil + PATCH OK, **en parallèle** de `notifications.Emitter.Emit(CategoryFriendAdded)`. Les deux systèmes coexistent : `notifications/*` = toast/in-app user-facing ; `notify/*` = alerting Discord externe.
- À la fin du sync initial du nouvel ami : `notify.NotifyFriendSyncCompleted(cfg, gt, matchesSynced, rowsAffected, dur)` **et** `notifications.Emitter.Emit(CategoryFriendSyncCompleted, payload)`.

**Préférences user** : double opt-in (toast in-app + Discord webhook). Defaults sensibles : toast=on, Discord=on si webhook configuré. Désactivation séparée par canal.

### 7. Documentation UX — clarifier scope filtres

Fichier : [apps/web/src/features/squad/SquadLayout.tsx](../apps/web/src/features/squad/SquadLayout.tsx) + i18n.

- Ajouter une ligne d'aide sous le bloc filtres : "La période et les modes (NavL2) filtrent les **matchs analysés**. Le sélecteur de sessions ci-dessous restreint en plus à des sessions précises."
- Le mini-filtre période interne au dropdown sessions, lui, ne filtre QUE le contenu du dropdown (libellé visuel : "Filtrer la liste").

---

## Tests

### Go (apps/go-api)

| Couche | Fichier | Cas couverts |
|---|---|---|
| `service/teammates_service_test.go` | existant, à étendre | (a) multi-labels squad → union des matchs ; (b) labels solo ignorés côté squad ; (c) liste vide = pas de filtre session ; (d) Q29 friends-only : un random co-joueur n'apparaît pas dans le top |
| `platform/duckdb/squad_repo_test.go` | existant, à étendre | Q29 modifiée : seed avec `friend_gamertags=[A,B]` + un random C → C absent du résultat ; ordre `started_at DESC` sur `SessionLabelsList.squad` |
| `sync/friends_recompute_test.go` | nouveau | (a) ajout ami → matchs antérieurs basculent à `is_with_friends=TRUE` ; (b) idempotence : 2e appel → 0 rows affected ; (c) gamertag inconnu (absent xuid_aliases) → log Warn + 0 rows, pas d'erreur ; (d) plusieurs XUIDs amis en un seul UPDATE ; (e) **`mv_player_matches` rafraîchie après UPDATE** (vérifier que la vue retourne les nouvelles lignes squad) ; (f) **lease lock effectivement acquis** (test concurrent : 2e appel doit attendre) ; (g) cas `rowsAffected=0` n'appelle pas `RefreshAggregates` (skip optim) |
| `service/friends_orchestrator_service_test.go` | nouveau | (a) itère bien tous les (title, gamertag) de `db_profiles.json` ; (b) une DB échoue → autres DBs continuent + agrégat erreur retourné ; (c) `OnFriendsChanged` appelé sans nouveau friend → no-op |
| `api/handlers/settings_test.go` | existant, à étendre | PATCH avec diff `friend_gamertags` → invoque l'orchestrateur (mock) en async ; PATCH sans diff → no-op |
| `api/handlers/teammates_test.go` | existant, à étendre | DTO multi-labels accepté + rétro-compat (champ singulier mappé en slice de 1) |
| `cmd/levelup/cmd_recompute_friends_test.go` | nouveau | bootstrap CLI : exécution sur fixture multi-titres → toutes les DBs basculées correctement |
| `notifications/service_test.go` | existant, à étendre | émission de `CategoryFriendAdded` et `CategoryFriendSyncCompleted` avec payload correct ; respect des préférences delivery (off → pas d'émission) |
| `notifications/emitter_test.go` | existant, à étendre | persistance + récupération des 2 nouvelles catégories |
| `notify/notifiers_extra_test.go` | existant, à étendre | `NotifyFriendAdded` et `NotifyFriendSyncCompleted` : webhook vide → no-op, flag désactivé → no-op, panic du builder → recovered |
| `notify/embeds_test.go` | existant, à étendre | builders `BuildFriendAddedEmbed` + `BuildFriendSyncCompletedEmbed` : champs FR et EN, format timestamp, stats |

Stratégie de mocks : mock `port.TeammatesRepository` + `port.FriendsRecomputeRepository` via interfaces ; DuckDB `:memory:` pour les tests d'intégration repo.

### Frontend (apps/web)

| Composant | Fichier | Cas |
|---|---|---|
| `SessionMultiSelect` | `apps/web/src/components/ui/SessionMultiSelect.test.tsx` (composant partagé) | filtre fuzzy, mini date-range, multi-checkboxes, validation différée, persistance localStorage, tri chrono décroissant |
| `SquadLayout` | existant | propagation des sessions multi vers la query, label "parmi N amis", dropdown alimenté par `squad[]` uniquement |
| `StatsLayout` (page Stats) | existant ou nouveau | dropdown alimenté par `solo[]` uniquement, intégration `SessionMultiSelect` |
| Flux ajout ami | `apps/web/src/features/squad/AddFriendModal.test.tsx` | modale de confirmation affichée, message sur durée du sync, mutations chaînées |

### Commandes

```bash
# Go
cd apps/go-api && go test ./... -race && go vet ./...
cd apps/go-api && go test ./internal/service/... -v -run "TestTeammates|TestFriendsRecompute"

# Frontend
cd apps/web && npm run typecheck && npm run lint && npm run test
```

## Logging (slog, clés structurées projet)

Conformément aux conventions arch-rules (`"err"`, `"player"`, `"match_id"`, `"duration"`, `"titleSlug"`).

| Endroit | Niveau | Message + clés |
|---|---|---|
| `teammates_service.GetPage` | Debug | `"squad page request"` + `picked_squad_count`, `selected_gamertags_count`, `friends_count` |
| `teammates_service.filterSynthesisBySession` (cas N labels) | Debug | `"filter sessions"` + `labels_count`, `matched_count` |
| `squad_repo.LoadTopTeammates` (Q29) | Info | `"top teammates loaded"` + `friends_count`, `rows`, `duration` |
| `sync.RecomputeIsWithFriends` (entrée/fin) | Info | `"friend recompute start"` + `player`, `titleSlug`, `friend_xuids_count` ; `"friend recompute done"` + `player`, `titleSlug`, `rows_affected`, `aggregates_refreshed`, `duration` |
| `sync.RecomputeIsWithFriends` (erreur) | Error | `"friend recompute failed"` + `err`, `player`, `titleSlug`, `friend_xuids_count` |
| `sync.RecomputeIsWithFriends` (xuid non résolu) | Warn | `"friend xuid not in aliases"` + `player`, `gamertag` |
| `service.FriendsOrchestratorService.OnFriendsChanged` | Info | `"friends changed orchestration"` + `added_count`, `removed_count`, `db_count` |
| `service.FriendsOrchestratorService` (agrégat) | Info | `"friends recompute summary"` + `processed`, `failed`, `total_rows`, `duration` |
| `setup handler / add friend` | Info | `"add friend requested"` + `friend_gamertag`, `existing_friends_count` |
| Settings PATCH avec diff friend_gamertags | Info | `"friend gamertags changed"` + `added`, `removed`, `total_friends` |
| `cmd_recompute_friends` (bootstrap) | Info (start/end) | `"bootstrap recompute start"`, `"bootstrap recompute done"` + counts |
| Émission notif `friend_added` | Info | `"notif emitted"` + `category=friend_added`, `friend_gamertag` |
| Émission notif `friend_sync_completed` | Info | `"notif emitted"` + `category=friend_sync_completed`, `friend_gamertag`, `matches_synced`, `rows_affected` |
| `notify.NotifyFriendAdded` (envoi OK) | Info | `[Discord:friend_added]` + `gamertag`, `webhook_status` (existing log style du package notify) |
| `notify.NotifyFriendSyncCompleted` (envoi OK) | Info | `[Discord:friend_sync_completed]` + `gamertag`, `matches_synced`, `duration` |
| `notify.Notify*` (webhook vide / flag off) | Debug | `[Discord:...]` + `"skip: webhook empty"` ou `"skip: flag disabled"` |

Aucun `fmt.Println` ni `log.Printf`. Toutes les erreurs DB remontées avec `slog.ErrorContext(ctx, msg, "err", err, ...)`.

## Risques & mitigations (recompute `is_with_friends`)

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| **Race avec sync concurrent** sur la même player DB → écriture concurrente, perte de données | Moyenne | Élevé | `AcquireLeaseCtx(ctx, playerDBPath)` + `AcquireLeaseCtx(ctx, sharedDBPath)` avant tout UPDATE. Couvert par test `(f) lease lock effectivement acquis`. |
| **`mv_player_matches` stale** après UPDATE → page Squad lit des lignes obsolètes | Élevée | Élevé | `sync.RefreshAggregates(playerDB)` appelé après chaque UPDATE non vide. Vérifié par test `(e)`. |
| **Bootstrap manqué sur déploiement** → données orphelines (toutes FALSE) sur installation neuve | Élevée (1ère install) | Élevé | Commande CLI `cmd_recompute_friends` documentée dans la procédure de release + entrée thought_log. |
| **Nouveau match ingéré reste à FALSE** car sync Go ne calcule pas `is_with_friends` | Élevée (en continu) | Moyen | TODO inline dans `writes.go`. Mitigation provisoire : trigger automatique du recompute en fin de chaque sync delta (ajout dans `engine.go` → 1 ligne) — à inclure dans ce lot pour fermer la boucle. |
| **Atomicité multi-DB** : N-1 DBs OK, 1 DB échoue | Moyenne | Faible | Pas de rollback inter-DB (par design). Échec loggé, retry idempotent via la garde `is_with_friends = FALSE`. Agrégat `{processed, failed}` retourné. |
| **Friend gamertag introuvable dans `xuid_aliases`** (jamais croisé en match) | Élevée (joueurs neufs) | Faible | Log Warn, skip silencieux côté UPDATE. Pas d'erreur 500. Test `(c)` couvre ce cas. |
| **DETACH oublié sur erreur** entre ATTACH et UPDATE | Faible | Faible | `defer` immédiat après ATTACH ; pattern miroir de `backfill_weapons.go`. |
| **Multi-titres : itération incomplète** (oubli d'un titre) | Faible | Moyen | `cfg.LoadPlayers()` est la seule source de vérité — pas de path codé en dur. Test `(a)` orchestrateur seed plusieurs titres. |
| **PATCH /settings async non terminé avant retour UI** → user voit l'ancien état | Moyenne | Faible | UI : invalider les queries Squad après PATCH ; mutation chainée affiche un toast "recalcul en cours" si le recompute renvoie un job ID 202. Pattern PostRecalculateSessions existant. |
| **Charge sur gros corpus** (joueur avec 50k matchs × N amis) | Faible | Moyen | UPDATE indexé sur `match_id` + filtre `IN (...)` — DuckDB bench typique <500ms pour 100k. Mesurer en log `duration` ; ajouter un seuil d'alerte si > 5s. |

## Vérification end-to-end

Manuel (browser) :
1. Ouvrir `/players/{slug}/squad/synergies` — vérifier que les sessions listées sont uniquement `is_with_friends=TRUE`.
2. Vérifier le tri chronologique décroissant dans le dropdown.
3. Sélectionner 2-3 sessions, valider, vérifier que la table coéquipiers se recalcule sur l'union.
4. Tester le mini-filtre date interne au dropdown — vérifier qu'il NE modifie PAS les matchs analysés (les KPIs globaux ne bougent pas tant que la sélection n'est pas validée).
5. Taper un gamertag inconnu → modale de confirmation → confirmer → vérifier création profil + sync + apparition dans les amis.
6. Vérifier que le label affiche "parmi N amis" avec N = nombre d'amis configurés.
7. **Recompute rétroactif** : ajouter un ami avec qui des matchs anciens existent → vérifier qu'au retour sur la page Squad, ces matchs apparaissent dans une session squad et que l'ami apparaît dans le top (sans avoir relancé de sync complet). Vérifier que `mv_player_matches` reflète bien les nouvelles lignes squad (requête DuckDB CLI : `SELECT COUNT(*) FROM mv_player_matches WHERE is_with_friends=TRUE` avant/après).
7a. **Bootstrap initial** : sur fixture neuve où `is_with_friends` est FALSE partout, exécuter `levelup recompute-friends` (nouveau cmd) → vérifier que la commande termine sans erreur sur tous les titres + agrégats rafraîchis + page Squad peuplée.
7b. **Concurrence sync** : lancer un sync delta sur un joueur en parallèle d'un PATCH friend_gamertags → vérifier qu'aucune erreur de DB locked, que les deux opérations terminent dans l'ordre, et que l'état final est cohérent (lease lock fonctionnel).
7c. **Notifications** : confirmer qu'un toast `friend_added` apparaît immédiatement après confirmation de la modale (côté Squad **et** côté Settings), puis qu'un toast `friend_sync_completed` apparaît à la fin du sync initial. Vérifier que les notifications sont aussi visibles dans le drawer in-app. **Discord** : avec `DISCORD_WEBHOOK_URL` configuré dans `.env.local` et `discord_notifications_enabled=true`, vérifier que les 2 embeds Discord arrivent dans le canal (un à l'ajout, un à la fin du sync). Désactiver `discord_notify_friend_added` puis re-tester → seul l'embed `sync_completed` doit arriver.
7d. **Settings → Utilisateurs** : tester le même flux d'ajout depuis la page Settings → résultat identique au flux Squad (modale, recompute, notifications).
7e. **Toggles notifications** : dans `NotificationsSettingsTab`, vérifier que `friend_added` et `friend_sync_completed` apparaissent avec sélecteur delivery. Mettre delivery=`off` sur `friend_added` → confirmer l'absence de toast/in-app à l'ajout suivant. Dans la section Discord de Settings, désactiver `discord_notify_friend_added` → confirmer l'absence d'embed Discord (mais toast in-app toujours présent si activé). Master switch `discord_notifications_enabled` à false → aucun embed quel que soit l'état des sous-flags.
8. **Page Stats solo** : vérifier que le dropdown ne contient plus que des sessions `is_with_friends=FALSE`. Après le recompute du cas 7, vérifier que les sessions concernées disparaissent de la page Stats.
9. **`SessionMultiSelect` partagé** : confirmer que le même composant est instancié sur Squad et Stats sans divergence visuelle.

## Thought log

À la fin, ajouter une entrée dans `.ai/thought_log.md` (date 2026-04-27, statut Complété, décisions principales : DTO multi-sessions, Q29 friends-only, recompute incrémental `is_with_friends`, mini-filtre période interne au dropdown, flux ajout ami explicite, mutualisation `SessionMultiSelect` Squad + Stats).
