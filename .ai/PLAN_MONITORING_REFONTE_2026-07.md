# PLAN — Refonte de fond du monitoring admin (2026-07)

> Statut : EN COURS (démarré 2026-07-10, branche `feat/monitoring-refonte-2026-07`).
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

- [ ] A2.1 Endpoint `GET /admin/monitoring/detections` (query : `status`, `level`,
      `module`, `title`, `limit`) → liste `_latest` (fingerprint, counts cumulés,
      first/last_seen, statut, note).
- [ ] A2.2 Endpoint `PATCH /admin/monitoring/detections/{fingerprint}` (body :
      `{status, note?}`) → append `detection_status_events`.
- [ ] A2.3 ErrorCollector : conservé comme tampon mémoire, mais la page lit le store
      persisté (le panneau « erreurs récurrentes » top-12 mémoire est REMPLACÉ —
      supprimer le code débranché, règle 0 code mort).
- [ ] A2.4 UI : section « Détections » — table TanStack (tri par last_seen/count,
      filtres statut/niveau/module), actions par ligne : Reconnaître / Mettre en
      sourdine / Résoudre (+ note). Query keys dans `lib/query/keys.ts`, strings FR+EN
      dans le manifest `admin.toml`. (Elle devient l'onglet « Détections » en A3 ;
      livrer d'abord sous l'onglet Logs actuel si A3 pas encore passée.)
- [ ] A2.5 Badges d'onglets : compter uniquement les détections `open` (le bruit `muted`
      ne colore plus la nav).
- [ ] A2.6 Tests : handler httptest (list + patch + réouverture), vitest sur la table
      et le mapping statuts.

**Gate A2** : `go test ./internal/api/handlers/...` + `make check-types` + `make test-web`
verts ; redémarrer le serveur local et vérifier que les détections et statuts survivent
(procédure : `Start-Process` détaché, port 8000).

### A3 — Architecture de l'information : onglets par question opérateur (effort : moyen)

Applique DC-8 et DC-9. Réorganisation de coquille : on déplace des sections existantes,
on n'en crée pas (le nouveau contenu arrive en A4-A7).

- [ ] A3.1 Routes file-based : `routes/admin/` recomposé vers les 6 onglets DC-8
      (jamais toucher `routeTree.gen.ts`) ; redirections des anciennes URLs
      (`/admin/convergence`, `/admin/data-quality`, `/admin/logs`, `/admin/access`,
      `/admin/titles`, `/admin/lab`) vers leur nouvelle destination.
- [ ] A3.2 Onglet « Données » : fusion Qualité données + Convergence + Invariants en
      sections d'une même page (les composants existants sont déplacés, pas réécrits).
      `InvariantsSection` quitte Système ; le KPI invariants de l'overview devient un
      verdict agrégé pointant vers Données.
- [ ] A3.3 Onglet « Sync » : absorbe `TokenHealthSection` (quitte Système) et la santé
      pool ; tokens n'apparaissent plus qu'ici (État n'affiche que le verdict).
- [ ] A3.4 Onglet « Gestion » : Access (users/invites) + Titres (avec son diagnostic)
      regroupés ; nav séparée visuellement du bloc observation (ordre : observation
      d'abord, gestion à droite).
- [ ] A3.5 Retrait du Lab (DC-9), inventaire sur pièces puis suppression complète :
      front `features/admin/lab/` (AdminLabPage, WaypointExplorerPanel + tests) et
      `features/lab/` (ResourcesPanel, LabHelp, i18n, queries), route `routes/admin/lab.tsx`,
      clés i18n `admin.lab.*` + manifests lab, query keys lab ; back `handlers/lab.go`
      + `lab_test.go`, `service.LabService`, routes `/lab/resources`, `/lab/contracts`
      (déjà 0 caller front — confirmer), `/lab/diagnostics`, `/lab/waypoint`, wiring.
      Vérifier grep zéro référence résiduelle avant suppression ET après.
- [ ] A3.6 Runbook `docs/RUNBOOK_ADD_TITLE.md` (EN-only — politique docs) : parcours
      « ajouter un titre » avec les CLI existantes (probe, metadata-fetch,
      populate-assets, config/titles TOML, onglet Gestion→Titres pour le diagnostic).
- [ ] A3.7 `tabBadges.ts` remappé sur les 6 onglets (source inchangée : overview seul) ;
      `AdminLayout` mis à jour ; strings FR+EN.
- [ ] A3.8 Tests : vitest nav/badges/redirects ; `go test ./internal/api/...` (routes
      lab supprimées) ; test grep garde-rail « zéro import features/lab ».

**Gate A3** : `make check-types && make test-web` + `go build ./... && go test ./...`
verts ; `grep -rn "features/lab\|/lab/" apps/web/src apps/go-api/internal --include="*.ts*" --include="*.go"`
→ 0 résultat applicatif ; les 6 onglets naviguent en local, anciennes URLs redirigent.

### A4 — Fraîcheur des données (effort : moyen) — atterrit dans « État »

- [ ] A4.1 Service `ops.ComputeDataFreshness(title)` : par joueur suivi — dernier match
      persisté (`match_registry` via timestamp canonique COALESCE), dernier cycle sync
      OK, âge, statut ok/warn/critical (seuils DC-3). Multi-titre : itère les slugs
      actifs du registry (capabilities, jamais `slug ==`) — couvre le trou de
      visibilité H5 (scheduler V2 = halo_infinite only, H5 passe par liveRunner).
- [ ] A4.2 Âge du dernier backup : lecture marker/dernier succès (source : `cron_runs`
      A1 pour backup-once ; à défaut mtime du dernier log de succès — décision au
      moment de l'implémentation, consigner).
- [ ] A4.3 Endpoint `GET /admin/monitoring/freshness?title=` + panneau onglet État
      (KPI accent + drill-down liste joueurs) ; badge si critical.
- [ ] A4.4 Tests : analysis/ops purs sur dataset hétérogène (joueur à jour, périmé,
      jamais synchronisé, titre sans capability).

**Gate A4** : tests Go verts ; l'onglet État local affiche la fraîcheur pour
halo_infinite ET halo_5.

### A5 — Ressources machine & process (effort : rapide/moyen) — « État » + détail « Système »

- [ ] A5.1 `GET /admin/monitoring/resources` : RSS/heap/goroutines, tailles des DB
      (shared, metadata, pve, social, players agrégés, monitoring) + WAL présents,
      disque libre du volume data, `duckdb_budgets` + pool stats (relecture expvar
      existants — enfin surfacés), uptime + compteur de restarts (marqueurs
      server.crash.log → `cron_runs` ou table dédiée).
- [ ] A5.2 Verdict compact sur État ; panneau détaillé sur Système.
- [ ] A5.3 Seuils visuels : disque < 2 Go = warn, < 500 Mo = critical (VPS 2 Go RAM /
      disque serré — pièges connus BuildKit/restic).

**Gate A5** : tests + vérif visuelle locale ; aucune valeur en dur non nommée.

### A6 — Statut unifié des crons + feature liveness (effort : moyen) — « État »

- [ ] A6.1 Registre `CronStatus` central (DC-5) branché sur : auto_sync, HealthScheduler,
      world_leaderboard_cron, catalog_refresh_cron, asset_name_sweep, spartan_cron,
      backup. Persisté dans `cron_runs` (A1).
- [ ] A6.2 Heartbeats features (DC-5, liste fermée) : `prestige_hook`,
      `notifications_push`, `watcher_rta`, `media_pipeline` — timestamp au passage réel
      dans le code (le cas « hook câblé mais jamais invoqué » devient visible).
- [ ] A6.3 Endpoint + panneau État « Crons & features » : chaque ligne = dernier
      succès, échecs consécutifs, heartbeat age ; accent destructive si
      consecutive_failures >= 3 ou heartbeat jamais vu.
- [ ] A6.4 Tests : registre pur + httptest.

**Gate A6** : tests verts ; arrêter un cron en local (config) → la ligne passe warn.

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
- 2026-07-10 (cartographie front A3, à intégrer à A3.5) : `features/lab/` N'EST PAS
  entièrement supprimable. `DiagnosticsPanel`, `getLabText`/`normalizeLabLocale`,
  `useLabDiagnostics` y sont réutilisés par `AdminDataQualityPage` (→ onglet Données), et
  `ChartsShowcasePage` par le bac à sable dev `/lab/charts` (route hors admin). A3.5 doit
  donc supprimer l'ONGLET Lab (`routes/admin/lab.tsx`, `features/admin/lab/*`,
  `admin.lab.*`, `admin.nav.lab`, entrée TABS) et le back Lab (handlers/lab.go, LabService,
  routes `/lab/*`), MAIS conserver les briques `features/lab/` encore consommées — sinon
  casse Données + charts. Le grep de garde-rail A3.8 doit viser `features/lab/queries`
  (endpoints back supprimés) et non `features/lab` en bloc.
