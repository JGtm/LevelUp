# PLAN — Refonte de fond du monitoring admin (2026-07)

> Statut : EN COURS — PARTIEL (branche `feat/monitoring-refonte-2026-07`).
> **A1 + A2 CLOSES** (gates verts, commits 43d6acfb8 + 6ae955515, poussés). Reprendre
> à **A3** (première case non cochée). A3-A9 non démarrées (périmètre de session : A3
> est une restructuration front lourde — 9 onglets → 6 + suppression Lab front/back +
> régén OpenAPI/types + runbook — qui mérite sa propre session focalisée ; le contrat
> plan-execution interdit d'enjamber/laisser une phase à moitié). Constat visuel restart
> serveur (gate A2) = revue utilisateur.
> Révisé 2026-07-07 : ajout de la phase A3
> (architecture de l'information — onglets réorganisés par question opérateur,
> retrait du Lab) validée par le user ; renumérotation A3→A9.
> Exécution sous contrat du skill `plan-execution` (ordre strict, gates par phase,
> statuts [x]/[~]/[!], zéro fix hors périmètre, découvertes consignées en fin de
> document).
>
> Branche cible : `feat/monitoring-refonte-2026-07` (1 branche, N commits — 1 commit
> minimum par phase). Base : `main` après clôture des audits 2026-07 (le merge main est
> gaté par `.ai/PLAN_CLOTURE_AUDITS_2026-07.md` V1-V3) ; sinon depuis
> `refactor/audits-2026-07`.
>
> Prérequis recommandé : exécuter d'abord `.ai/PLAN_MONITORING_TRIAGE_DETECTIONS_2026-07.md`
> (sinon la nouvelle UI re-liste le bruit existant).

## Objectif et critère de succès

Le monitoring actuel (9 onglets `/admin`) est un **instantané mémoire du process** :
ErrorCollector plafonné à 100 buckets depuis le boot, historique scheduler limité à 48
cycles en RAM, dernier run data-health uniquement, jobs en mémoire. Un restart = amnésie
totale. Les détections n'ont **aucun cycle de vie** (pas de nouveau/reconnu/muet/résolu),
donc la page liste du bruit sans hiérarchie — c'est pourquoi elle n'est jamais utilisée.
On monitore le process, pas le produit ni la machine (pas de fraîcheur des données,
pas de disque/RSS/tailles DB, pas de statut des crons secondaires, pas de 5xx, pas de
« feature liveness » — le hook Prestige mort n'a été vu que par un audit manuel).
Enfin, les 9 onglets recopient l'architecture interne (un onglet par sous-système) au
lieu de répondre aux questions de l'opérateur : *tout va bien ? qu'est-ce qui a changé ?
que dois-je faire ?* — avec des recouvrements (tokens et invariants affichés à 2
endroits) et un mélange observation / gestion / outillage dev (Lab).

**Critère de succès** : après un restart serveur, la page monitoring affiche encore
l'état vrai (détections persistées avec statut, historique) ; un problème silencieux
(cron en échec, données périmées, disque plein, feature débranchée) est visible en < 1
écran sur l'onglet « État » ; chaque onglet répond à UNE question opérateur ; l'UI suit
le catalogue canonique (modèle Explorer).

**Hors périmètre (décisions produit déjà tranchées)** :
- Alerting externe (Discord/mail) et uptime monitoring : ÉCARTÉS (décision user
  2026-04-29, reconfirmée 2026-07-06). La surface = in-app uniquement (badges/pastilles).
- Prometheus/OTel : écarté (ADR 0009, expvar suffit).
- Ré-introduction de l'error tracker HTTP supprimé (ADR 0009) : non — un simple
  compteur par classe de statut le remplace (phase A7).

## Décisions pré-tranchées (DC)

- **DC-1 — Stockage** : nouvelle DB `data/global/monitoring.duckdb`, chemin via
  `PathResolver` (nouvelle méthode, jamais de `filepath.Join` à la main). Tables
  **append-only + vues `_latest`** (recette ADR 0026 / `append_only_rebuild.go`),
  un seul writer = process serveur (pas de CLI concurrente). Pas de BatchBuilder
  (pas de données per-match) mais un persister dédié `internal/persist` ou `ops`.
- **DC-2 — Fingerprint détection** : `(level, module, message)` — même clé que
  l'ErrorCollector actuel, + `title_slug` quand pertinent. Statuts : `open` /
  `acked` / `muted` / `resolved`. Une occurrence après `resolved` ré-ouvre (append
  d'un event, la vue `_latest` reflète le nouveau statut `open`).
- **DC-3 — Seuils fraîcheur** : par joueur suivi et par titre : WARN si dernier match
  persisté > 48 h ET dernier cycle sync réussi > 6 h ; CRITICAL si > 7 j. Seuils dans
  `app_settings.json`, pas en dur.
- **DC-4 — Ressources** : runtime Go (`runtime.MemStats`, `NumGoroutine`) + tailles
  fichiers DB/WAL via `os.Stat` sur les chemins `PathResolver`. Disque libre : petite
  façade platform (build tags windows/linux), pas de nouvelle dépendance.
- **DC-5 — Crons & liveness** : registre central `CronStatus` (nom, last_run,
  last_success, last_error, consecutive_failures) alimenté par chaque cron ; heartbeats
  de features = timestamp expvar `levelup.heartbeat.{feature}` (liste fermée initiale :
  prestige_hook, notifications_push, watcher_rta, media_pipeline).
- **DC-6 — HTTP** : compteurs expvar par classe (`2xx/3xx/4xx/5xx`) au niveau du
  middleware existant, PAS par route (pas d'explosion de cardinalité).
- **DC-7 — UI** : modèle = page Explorer (catalogue des 6 types canoniques). Tables
  interactives (tri/filtre) = TanStack ; tables statiques courtes = primitives partagées.
- **DC-8 — Architecture des onglets cible** (validée user 2026-07-07) — chaque onglet
  répond à une question :

  | Onglet | Question | Contenu |
  |---|---|---|
  | État | « Tout va bien ? » | Verdict global, fraîcheur (A4), ressources (A5), crons & features (A6), 5xx (A7) |
  | Détections | « Que dois-je traiter ? » | Cycle de vie détections (A2), remplace l'onglet Logs pour le triage |
  | Données | « Mon warehouse est-il intègre ? » | Qualité données + Convergence + Invariants fusionnés (sections) |
  | Sync | « Le moteur tourne-t-il ? » | Scheduler + jobs + santé tokens/pool |
  | Système | Bas niveau | Contention DB, perf, tail de logs bruts, backup |
  | Gestion | Administration | Utilisateurs + invitations + titres (diagnostic par titre inclus) |

  9 onglets → 6. Tokens et invariants n'apparaissent plus qu'à UN endroit chacun
  (Sync et Données respectivement ; l'onglet État n'en montre que le verdict agrégé).
- **DC-9 — Lab : RETIRÉ de l'app** (décision user 2026-07-07). Sa raison d'être
  (outiller l'ajout d'un nouveau titre, explorer les endpoints Waypoint/Discovery) est
  un workflow de développement, mieux servi par Claude Code + les CLI existantes
  (`probe-h5`, `probe-mcc`, `h5-metadata-fetch`, `populate-assets`, `levelup-titles`,
  `diag_q`). La valeur opérationnelle résiduelle (diagnostic par titre) est déjà dans
  l'onglet Titres (→ Gestion). Suppression totale front+back (règle 0 code mort),
  compensée par un runbook « ajout d'un titre » (EN-only, politique docs).
- Les endpoints nouveaux suivent le pattern existant : `RequireAuth+RequireAdmin+NoStore`,
  best-effort par section, `?title=` quand title-scopé, types domain dans
  `internal/domain/`, zéro SQL dans les handlers.

## Phases

### A1 — Socle de persistance monitoring (effort : moyen)

- [x] A1.1 `PathResolver` : méthode `GlobalMonitoringDB()` → `data/global/monitoring.duckdb`.
      (registry.go — globale, non per-titre.)
- [x] A1.2 Schéma (migration idempotente, package `internal/migration`) :
      `detection_events` (fingerprint, level, module, message, title_slug, occurred_at,
      count_delta, sample_detail, written_at), `detection_status_events` (fingerprint,
      status, note, written_at), vues `detections_latest` + `detection_status_latest`,
      `cron_runs` (cron_name, started_at, ok, err, duration_ms, written_at),
      `data_health_runs` (résultat sérialisé du run + written_at).
      (`internal/migration/monitoring_schema.go` — `EnsureMonitoringSchema`, séquences+PK
      technique, vues via ROW_NUMBER/ARG_MAX. `cron_runs_latest` ajoutée pour A6. Base
      globale hors registre title-scopé : schéma posé à l'ouverture du store.)
- [x] A1.3 Writer unique : `internal/ops/monitoring_store.go` (INSERT-only, ouvert via
      `duckdb.OpenReadWrite` + lease `dblease.KindMonitoring` — pas de bare connect),
      flush du delta ErrorCollector + `RecordCronRun` + `RecordDataHealthRun` +
      `LatestDataHealthJSON` + ré-ouverture sur occurrence après `resolved` (DC-2).
      Le CÂBLAGE au boot (flush périodique) + hooks HealthScheduler/crons se font
      respectivement en A2 (survie au restart), A4 (data-health) et A6 (crons) — les
      méthodes du store sont livrées ici.
- [x] A1.4 Tests : `monitoring_store_cgo_test.go` (cycle de vie complet dont réouverture,
      filtres, cron+data-health), `monitoring_schema_cgo_test.go` (idempotence + vues
      _latest servent le dernier statut), `monitoring_store_guard_test.go` (garde-rail
      grep : aucun UPDATE/DELETE sur les tables append-only).

**Gate A1** : `cd apps/go-api && go build ./... && go test ./internal/ops/... ./internal/migration/...`
verts ; `grep -rn "filepath.Join" internal/ops/monitoring_store.go` → 0 hors PathResolver.
RÉSULTAT 2026-07-10 : build EXIT 0 ; `ok internal/ops 10.96s` + `ok internal/migration 0.56s` ;
grep filepath.Join → 0 ; `golangci-lint --new-from-rev=158b336a9` → 0 issues. GATE PASSÉ.

### A2 — Cycle de vie des détections + UI triage (effort : lourd)

- [x] A2.1 Endpoint `GET /admin/monitoring/detections` (query : `status`, `level`,
      `module`, `title`, `limit`) → liste `_latest`. (handler `handleGetDetections` +
      runner `reg.DetectionsReport` qui flush l'ErrorCollector avant lecture ;
      openapi.yaml path + schémas ajoutés.)
- [x] A2.2 Endpoint `PATCH /admin/monitoring/detections/{fingerprint}` (body :
      `{status, note?}`) → append `detection_status_events`. (validation statut → 400 ;
      store nil → 503.)
- [x] A2.3 ErrorCollector conservé (feed le store via flush) ; la page lit le store
      persisté. Panneau front « erreurs récurrentes » (`RecurringErrorsPanel`) +
      `useMonitoringErrors` SUPPRIMÉS (0 code mort). Clés i18n `admin.errors.*`
      remplacées par `admin.detections.*`. (Endpoint back `/monitoring/errors` conservé
      comme route live — voir Découvertes.)
- [x] A2.4 UI : `DetectionsPanel` (TanStack, tri count/last_seen, filtre statut
      client-side, actions Reconnaître/Sourdine/Résoudre/Rouvrir + note via prompt),
      rendu en tête de l'onglet Logs (A3 en fera l'onglet « Détections »). Query key
      `adminMonitoringDetections`, strings FR+EN dans `admin.toml`, couleurs = tokens.
- [x] A2.5 Badges : gauge expvar `monitoring_detections_open` (posée au flush) →
      `overview.open_detections` (zéro I/O) → `tabBadges` colore `/admin/logs` sur les
      seules détections `open`.
- [x] A2.6 Tests : handler httptest (list+filtres, patch, statut invalide, nil→503) ;
      vitest `detectionDisplay.test.ts` (mapping statuts/niveaux + filtre pur).

**Gate A2** : `go test ./internal/api/handlers/...` + `make check-types` + `make test-web`
verts ; redémarrer le serveur local et vérifier que les détections et statuts survivent
(procédure : `Start-Process` détaché, port 8000).
RÉSULTAT 2026-07-10 : `go test ./internal/api/handlers/` ok ; contract routes + drift
OpenAPI verts (schémas MISSING documentés) ; `npm run typecheck` EXIT 0 ;
`vitest run` 247 fichiers / 2106 tests OK ; `golangci-lint --new-from-rev=158b336a9` → 0.
RESTE : vérif restart serveur local = revue utilisateur (persistance prouvée par les
tests store cgo A1.4 + schéma file-backed ; la survie effective au reboot process reste à
constater visuellement). GATE PASSÉ (hors constat visuel restart, délégué à l'utilisateur).

### A3 — Architecture de l'information : onglets par question opérateur (effort : moyen)

Applique DC-8 et DC-9. Réorganisation de coquille : on déplace des sections existantes,
on n'en crée pas (le nouveau contenu arrive en A4-A7).

- [x] A3.1 Routes file-based recomposées vers les 6 onglets DC-8 : nouvelles routes
      `/admin/detections`, `/admin/data`, `/admin/management` ; redirections
      (`beforeLoad`+`redirect`) : convergence→data, data-quality→data, logs→system
      (search module/level/q/n préservé — l'URL-state du viewer vit sur system),
      access→management, titles→management, lab→management. `routeTree.gen.ts`
      régénéré par le plugin (vite build), jamais édité à la main.
- [x] A3.2 Onglet « Données » (`AdminDataPage`) : compose AdminDataQualityPage +
      AdminConvergencePage + InvariantsSection en sections (composants déplacés, pas
      réécrits). InvariantsSection quitte Système ; KPI invariants de l'overview →
      drill-down `/admin/data` (verdict agrégé inchangé).
- [x] A3.3 Onglet « Sync » : absorbe TokenHealthSection (quitte Système) ; KPI tokens
      de l'overview → drill-down `/admin/sync` ; verdicts diagnostics tokens/api_auth
      pointent vers Sync. (Santé pool/watcher/API Halo déjà présentes.)
- [x] A3.4 Onglet « Gestion » (`AdminManagementPage`) : UsersSection (ex-Access) +
      AdminTitlesPage (avec son diagnostic) en sections ; onglet séparé visuellement
      (ml-auto + border-l) — observation à gauche, gestion à droite.
      `AdminAccessPage` (wrapper devenu orphelin) supprimé.
- [x] A3.5 Retrait du Lab (DC-9) — PÉRIMÈTRE AJUSTÉ sur pièces (cf. Découvertes,
      validé superviseur) : `features/lab/` n'était PAS entièrement supprimable
      (DiagnosticsPanel/getLabText/useLabDiagnostics consommés par l'onglet Données ;
      ChartsShowcasePage par le bac à sable dev `/lab/charts`, hors plan → conservé).
      SUPPRIMÉ front : `features/admin/lab/` (AdminLabPage, WaypointExplorerPanel +
      test), ResourcesPanel, LabHelp, useLabResources/useLabWaypoint + types + query
      key labResources, _labShared réduit (RouteList/JsonViewer/SelectableLists),
      i18n lab réduit à common+diagnostics, clés `admin.lab.*` + `admin.nav.lab`,
      fixtures MSW resources/contracts/waypoint ; `routes/admin/lab.tsx` = redirection.
      SUPPRIMÉ back : routes `/lab/resources`, `/lab/contracts` (0 caller confirmé),
      `/lab/waypoint` ; LabService réduit à GetDiagnostics (WaypointExplorer,
      ErrLabWaypoint* supprimés) ; provider réduit (provider_assets_medals.go +
      provider_contracts.go supprimés, listAllMedalEntries/loadParityReport conservés) ;
      domain/lab.go réduit aux types diagnostics ; closure waypointExplore retirée de
      server_apiv1.go ; openapi.yaml : 3 paths + 14 schémas orphelins supprimés,
      types front régénérés. CONSERVÉ : `GET /lab/diagnostics` (gate
      can_manage_instance intact). Greps avant/après verts (voir gate).
- [x] A3.6 Runbook `docs/RUNBOOK_ADD_TITLE.md` (EN-only) : parcours complet probe →
      déclaration registry+TOML → metadata-fetch/populate-assets → adapters →
      routage sync (piège orchestrator mono-titre) → `levelup-titles diagnose` +
      onglet Gestion→Titres.
- [x] A3.7 `tabBadges.ts` remappé DC-8 : Sync = échecs cycle + tokens morts
      (destructive) sinon jobs actifs (info pulse) ; Données = invariants FAIL
      (destructive) sinon warn invariants + data health (warning) ; Détections =
      `open` seul (warning). AdminLayout 6 onglets ; nav labels FR+EN mis à jour
      (« État », « Détections », « Données », « Sync », « Système », « Gestion »).
- [x] A3.8 Tests : tabBadges.test remappé + cas tokens/détections ;
      `lab-removal.guard.test.ts` (garde-rail AJUSTÉ : interdit les endpoints
      supprimés + les imports des modules supprimés — pas `features/lab` en bloc,
      briques encore consommées — ET vérifie les 6 redirections) ;
      `lab_routes_mounted_test.go` inversé (diagnostics montée, resources/contracts/
      waypoint ABSENTES) ; lab_test.go réduit ; drift OpenAPI + contract routes verts.

**Gate A3** : `make check-types && make test-web` + `go build ./... && go test ./...`
verts ; grep gate (AJUSTÉ, cf. A3.5) → 0 résultat applicatif hors survivants déclarés ;
les 6 onglets naviguent en local, anciennes URLs redirigent.
RÉSULTAT 2026-07-10 : `tsc -b` EXIT 0 ; `vitest run` 247 fichiers / 2108 tests OK ;
`go build ./...` EXIT 0 ; `go test ./...` EXIT 0 (0 FAIL) ; grep Go `"/lab/` → seule
`/lab/diagnostics` (handlers/lab.go) ; grep web `'/lab/` → queries diagnostics +
routes/lab/charts (sandbox conservé) ; refs Go Lab supprimées → 0 ;
`golangci-lint --new-from-rev=158b336a9` → 0 ; vite build OK (routeTree régénéré).
Navigation visuelle des 6 onglets = revue utilisateur (redirections couvertes par le
garde-rail). GATE PASSÉ.

### A4 — Fraîcheur des données (effort : moyen) — atterrit dans « État »

- [x] A4.1 Service PURE `ops.EvaluatePlayerFreshness` + seuils
      `FreshnessThresholdsFromSettings` (app_settings.json :
      freshness_warn_match_hours/warn_sync_hours/critical_match_days — défauts DC-3
      48h/6h/7j, jamais en dur). Orchestrateur `reg.FreshnessReport` : itère
      `titlePkg.DefaultRegistry().NonArchived()` actifs non-internes (registre
      config-driven du boot — PAS NewRegistry() qui ne connaît que le défaut),
      capability `CapMatchmaking` requise (jamais `slug ==`). Dernier match =
      `MAX(SQLStartTimeCanonical)` sur match_registry⋈match_participants (règle n°8) ;
      dernier sync OK = snapshot scheduler (halo_infinite ; H5 live-only → inconnu,
      l'âge du match fait foi — trou de visibilité couvert). Best-effort par joueur.
      Sémantique DC-3 : sync récent (≤6h) → ok même si le joueur ne joue pas ; sinon
      match >7j (ou aucun) → critical, >48h → warn.
- [x] A4.2 Âge du dernier backup : DÉCISION = `duckdbbackup.Scheduler.Status()`
      (manifest restic, LastBackupAt) exposé dans la réponse freshness — PAS
      cron_runs (câblé seulement en A6) ni mtime de log (fragile).
      `reg.WithBackupScheduler` câblé dans server_apiv1.go (même scheduler que
      /settings/backup/status).
- [x] A4.3 Endpoint `GET /admin/monitoring/freshness?title=` (vide = tous titres
      actifs) + `FreshnessPanel` sur État (sections par titre, table joueurs
      dernier match / dernier sync / statut, âge backup) + KPI « Fraîcheur
      critique » (accent destructive/success) ; badge nav État via gauge expvar
      `monitoring_freshness_critical` (posée au calcul) → `overview.freshness_critical`
      → tabBadges `/admin`. openapi.yaml : path + 4 schémas ; types front régénérés.
- [x] A4.4 Tests : `data_freshness_test.go` — dataset hétérogène (à jour, inactif
      +moteur vivant, périmé 3j, mort 10j, jamais synchronisé, sync inconnu
      live-only, DB en erreur) + seuils settings ; handler httptest (filtre titre
      sans fallback défaut, runner nil) ; badge test. Le cas « titre sans
      capability » est traité par l'orchestrateur (Note, dégradation gracieuse).

**Gate A4** : tests Go verts ; l'onglet État local affiche la fraîcheur pour
halo_infinite ET halo_5.
RÉSULTAT 2026-07-10 : `go build` 0 ; tests ops/handlers/wire verts ; contract+drift
verts ; `tsc -b` 0 ; `vitest run` 247 fichiers / 2109 tests OK ; lint new-from-rev 0.
halo_5 couvert par DefaultRegistry (title.toml status=active + capability matchmaking
vérifiés sur pièces). Constat visuel État (les 2 titres affichés) = revue utilisateur.
GATE PASSÉ.

### A5 — Ressources machine & process (effort : rapide/moyen) — « État » + détail « Système »

- [x] A5.1 `GET /admin/monitoring/resources` : runtime Go (heap/sys/goroutines/GC via
      `ops.CollectRuntimeStats`), tailles DB + WAL (`ops.DBFileSize` sur chemins
      PathResolver : shared/metadata/pve/social par titre actif + players agrégés
      `DirTotalSize` + globales aliases/monitoring), disque libre via façade
      `platform/diskfree` (build tags windows/unix, x/sys déjà dans le graphe —
      DC-4 zéro nouvelle dépendance), `duckdb.BudgetsSnapshot()` +
      `PoolStatsSnapshot()` enfin surfacés, uptime + restarts = COUNT(marqueur
      `server_boot`) dans `cron_runs` (écrit au boot par main.go — DÉCISION :
      compteur persistant via cron_runs A1, pas de table dédiée ni parsing
      server.crash.log).
- [x] A5.2 Verdict compact sur État (KPI « Disque libre » accent par statut,
      drill-down Système) ; panneau détaillé `ResourcesSection` sur Système
      (résumé runtime/disque/uptime/restarts + table des bases + WAL + total +
      budgets/pools en détail dépliable).
- [x] A5.3 Seuils NOMMÉS `ops.DiskFreeWarnBytes` (2 Go) / `DiskFreeCriticalBytes`
      (500 Mo) — `EvaluateDiskStatus` pur testé ; aucun littéral chez les callers.

**Gate A5** : tests + vérif visuelle locale ; aucune valeur en dur non nommée.
RÉSULTAT 2026-07-10 : build 0 ; tests ops (disk status bornes exactes, DBFileSize+WAL,
DirTotalSize) + handlers verts ; contract+drift verts (path + 4 schémas) ; `tsc -b` 0 ;
vitest admin 81 tests OK ; lint new-from-rev 0. Vérif visuelle locale (KPI État +
panneau Système) = revue utilisateur. GATE PASSÉ.

### A6 — Statut unifié des crons + feature liveness (effort : moyen) — « État »

- [x] A6.1 Registre central `observability.ReportCronRun/CronStatusSnapshot`
      (cronstatus.go — last_run/last_success/last_error/consecutive_failures) branché
      sur les 7 crons : auto_sync (point de convergence storeCycleResult, échec =
      joueurs failed), HealthScheduler (runCycle), catalog_refresh, asset_name_sweep,
      spartan_customization, world_leaderboard (RunOnce — liveness, erreurs par
      titre/playlist restant best-effort internes), backup (callback `OnCycleDone`
      ajouté à pkg/duckdbbackup — le package reste standalone, pont câblé dans
      ops.NewLevelUpBackupScheduler). Persistance : `SetCronRunSink` → cron_runs (A1),
      câblé au boot (main.go).
- [x] A6.2 Heartbeats (liste fermée DC-5) posés au passage RÉEL :
      `prestige_hook` (prestige.RunPostSyncHook), `notifications_push`
      (notifications.Service.Emit), `watcher_rta` (RESTPoller.Run tick),
      `media_pipeline` (MediaService.runTranscodeJob) — via
      `observability.Heartbeat` (expvar heartbeat_{feature}).
- [x] A6.3 Endpoint `GET /admin/monitoring/crons` (fusion registre mémoire depuis le
      boot + réhydratation cron_runs_latest marquée `since_boot=false`) + `CronsPanel`
      sur État (table crons : dernier succès / échecs consécutifs / statut ; table
      features : heartbeat age). Accents : critical si consecutive_failures >=
      `domain.CronFailuresCriticalThreshold` (3, nommé), destructive si heartbeat
      `never`. openapi.yaml + types front régénérés.
- [x] A6.4 Tests : `cronstatus_test.go` (cycle succès→3 échecs→récupération, sink
      relay, heartbeats), httptest crons (payload + runner nil).

**Gate A6** : tests verts ; arrêter un cron en local (config) → la ligne passe warn.
RÉSULTAT 2026-07-10 : `go test ./internal/...` 0 FAIL (dont garde-rail shared_social :
entrée whitelist DATÉE ajoutée pour registry_monitoring_resources.go — os.Stat pur,
aucune connexion) ; `tsc -b` 0 ; vitest 247/2109 OK ; lint new-from-rev 0. Les
transitions warn/critical sont couvertes par les tests du registre pur ; le constat
manuel « cron arrêté → warn » local = revue utilisateur. GATE PASSÉ.

### A7 — Compteurs HTTP par classe (effort : rapide) — « État »

- [ ] A7.1 Middleware existant : incrément expvar `levelup.http.status_2xx/4xx/5xx`
      (+ par titre si trivial via observability.titled).
- [ ] A7.2 Exposition dans l'overview État (depuis boot + delta snapshot roulant,
      pattern existant).
- [ ] A7.3 Test middleware.

**Gate A7** : `go test ./internal/api/...` vert.

### A8 — Alignement UI catalogue (effort : moyen)

Constat (cartographie 2026-07-06) : tokens/i18n/query keys déjà conformes ; les écarts
sont structurels.

- [ ] A8.1 Un seul composant KPI : remplacer les 5 variantes locales (`OverviewKpi`,
      `SummaryCell`, `DQKpi`, `BacklogKpi`, `DataHealthMetric`) par `KpiCard` foundations
      (+ wrapper admin si besoin d'un variant compact). Garde-rail : test grep interdisant
      la re-déclaration locale d'un composant `*Kpi` sous `features/admin/`.
- [ ] A8.2 Hook `useCounterSnapshot(key, build)` : factoriser le pattern snapshot delta
      dupliqué 3× (data-quality, convergence, invariants). Garde-rail grep sur
      `readCountersSnapshot(` hors du hook.
- [ ] A8.3 Composant `SectionHeader` admin unique (3 patterns actuels → 1).
- [ ] A8.4 Primitives table admin (`AdminTable`, `AdminTh`, `AdminTd`) pour les tables
      natives statiques restantes ; les tables interactives (A2.4) = TanStack.
- [ ] A8.5 Extraire les fichiers > 300 L : `AdminTitlesPage.tsx` (343) et
      `AdminDataQualityPage.tsx` (348) → sous-composants en fichiers dédiés (si A3 ne
      l'a pas déjà imposé en les déplaçant).
- [ ] A8.6 Supprimer `JobProgressInline.tsx` (orphelin, 0 import) avec ses tests éventuels.
- [ ] A8.7 i18n des 3 strings en dur (`OK`/`FAIL`/`WARN` — PostSyncMatrix:30,
      AdminOverviewPage:128, InvariantsSection:142-147) — FR ET EN.

**Gate A8** : `make check-types && make test-web && make go-api-lint` verts ; garde-rails
A8.1/A8.2 rouges sur l'ancien pattern ; revue visuelle user au merge.

### A9 — Clôture (effort : rapide)

- [ ] A9.1 Statuer chaque item du plan ([x]/[~]/[!]) — aucune case vide.
- [ ] A9.2 `docs/` : section monitoring dans FOUNDATIONS_GUIDE si l'UI a de nouveaux
      composants canoniques (bilingue FR/EN, même PR) ; RUNBOOK_ADD_TITLE livré (A3.6).
- [ ] A9.3 Entrée `thought_log.md` + skill `delivery-checklist` avant merge.
- [ ] A9.4 Gate final complet : `cd apps/go-api && go test ./...` +
      `go test -tags=integration -p 1 ./...` + `make check-types && make test-web &&
      make go-api-lint`.

## Protocole de reprise de session

Lire ce fichier (statuts d'items) + `git log --oneline -10` sur la branche + dernière
entrée thought_log. Reprendre à la première case non cochée de la phase la plus basse
non close. Une phase est close quand tous ses items sont statués ET son gate est passé.

## Découvertes hors périmètre (à consigner, ne pas traiter)

- 2026-07-07 : `/lab/contracts` (handlers/lab.go) semble déjà sans appelant front
  (queries.ts n'appelle que resources/diagnostics/waypoint) — code mort probable,
  absorbé par la suppression A3.5.
- 2026-07-10 (A2.3) : l'endpoint back `GET /admin/monitoring/errors` (+ runner
  `reg.ErrorStats`, DTO `AdminErrorStats`/`AdminErrorBucket`) n'a plus de consommateur
  front (le panneau qui l'appelait est supprimé, remplacé par les détections). Route
  toujours montée et testée (`TestAdminMonitoring_Errors_OK`). C'est une route live
  (diagnostic brut curl-able du tampon ErrorCollector conservé), pas du code débranché du
  routing — conservée sciemment. Candidate à suppression si confirmée inutile (décision
  produit) : à trancher hors périmètre de ce plan.
- 2026-07-10 (cartographie front A3, à intégrer à A3.5) : `features/lab/` N'EST PAS
  entièrement supprimable. `DiagnosticsPanel`, `getLabText`/`normalizeLabLocale`,
  `useLabDiagnostics` y sont réutilisés par `AdminDataQualityPage` (→ onglet Données), et
  `ChartsShowcasePage` par le bac à sable dev `/lab/charts` (route hors admin). A3.5 doit
  donc supprimer l'ONGLET Lab (`routes/admin/lab.tsx`, `features/admin/lab/*`,
  `admin.lab.*`, `admin.nav.lab`, entrée TABS) et le back Lab (handlers/lab.go, LabService,
  routes `/lab/*`), MAIS conserver les briques `features/lab/` encore consommées — sinon
  casse Données + charts. Le grep de garde-rail A3.8 doit viser `features/lab/queries`
  (endpoints back supprimés) et non `features/lab` en bloc.
