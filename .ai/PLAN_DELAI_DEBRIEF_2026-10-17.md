# Plan — Délai du débrief (2026-10-17)

Branche `feat/delai-debrief` (depuis `origin/feat/v75` @ `c7decf35b`), worktree
`../LevelUp-wt-delai-debrief`. Contrat d'exécution : skill `plan-execution` (le plan fait foi en
cas de divergence). La release v8.0.0 (`.ai/PLAN_RELEASE_V8_2026-10-17.md`, item 0.1b) attend la
fusion de ce lot dans `feat/v75`.

## Objectif et critère de succès

Toutes les données d'un match 4v4 (stats, source des kills, rejeu 2D, dérivations du rejeu,
citations du film) disponibles **moins de 5 min après la fin du match**, publication du film par
Halo comprise, **prouvé par la mesure de l'étape 1** sur un vrai match, en mode local ET en mode
ouvrier.

Décisions de l'utilisateur (2026-10-17), fermes : indicateur « Analyse en cours » inchangé, aucun
masquage de match ; hors périmètre : délai de publication du film par Halo et lecture des morts
(killsource) sur le VPS.

## Constats vérifiés sur pièces (2026-10-17, `c7decf35b`)

| # | Constat de départ | Verdict | Pièces |
|---|---|---|---|
| 1 | File de l'ouvrier sans priorité | **Confirmé** en mode ouvrier : `ClaimBuildJob` trie `priority DESC, enqueued_at ASC` (`ops/build_queue.go:142-145`), `EnqueueReplayBuild` ne pose jamais `Priority` (`wire/registry_build_queue.go:89-94`) : un match frais passe derrière le rattrapage enfilé aux cycles précédents. **Infirmé** pour l'ORDRE en local (`selectionnerLeTravail` met les insérés d'abord, `artifacts.go:398-413`), mais un défaut voisin y existe : `Deriver` ne tourne qu'après TOUTE la cuisson du cycle (`artifacts.go:372-381`), donc les dérivations du match frais attendent jusqu'à 4 cuissons de rattrapage (budget 5 min, `backlog.go:72`), et les citations (1.6) attendent la fin de 1.58. | idem |
| 2 | Rien n'est déclenché à la livraison | **Partiellement faux.** Les DÉRIVATIONS sont déjà déclenchées à la livraison dans les deux modes : dépôt d'ouvrier `StoreBuildArtifact` → `deriverArtefactRange` → `replayartifacts.Deriver` (`wire/registry_build_queue.go:244`, synchrone, writer borné à 8 s) ; local `buildAll` → `Deriver` (`artifacts.go:378`). **Confirmé pour les CITATIONS** : seules les recalcule l'étape 1.6 du post-sync de CHAQUE joueur (`citations_backfill.go:421-470`, `SelectMatchesFilmPlusRecent`), donc au cycle suivant de chaque joueur suivi (15 min), y compris pour les coéquipiers suivis d'un match cuit par le cycle d'un autre. | idem |
| 3 | Quel chemin enfile en prod | **Cartographié.** Le hook s'installe TOUJOURS (`scheduler/auto_sync_engine.go:151`, `cmd/server/sync_v2_wiring.go:342`, `handlers/sync_handler.go:209`) et décide à chaque cycle par `replaybuild.DecidePlacement` : en prod, `local` est refusé, vide = `worker` si un jeton d'ouvrier est configuré, sinon `off`. Mode ouvrier : 1.58 enfile (`enqueueAll`, `cuisson.go:37`) via `ServiceRegistry.EnqueueReplayBuildJob` (résolution du manifeste + URL pré-signées) ; l'ouvrier `cmd/replay-worker` interroge toutes les 5 s, cuit, dépose (`POST /internal/build-queue/artifact`). Mode local : 1.58 cuit dans un enfant borné (`SpawnBuildOne`), au plus 5 matchs par cycle. Le commentaire « À N'APPELER QU'EN LOCAL » de `engine_options.go:180` est périmé (il décrit l'avant-hook) : corrigé dans ce lot (règle 17, item 1.9). | idem |
| 4 | Verrou de décodage tenu par la recherche | Propre au poste. Gate de l'étape 5 : aucun détenteur de `data/cache/film_decode.lock.json` avant de mesurer. | — |

Détection : le watcher (`internal/watcher/daemon.go:603`) soumet un sync au coordinateur dès la
fin de match détectée (sondage 30 s) ; le cycle planifié de 15 min n'est qu'un filet.

## Décisions de conception (soumises à l'utilisateur avec ce plan)

- **DD-1 Mesure** : une table append-only `match_step_events` dans la base monitoring globale
  (même writer, même lease `KindMonitoring` que la file de construction ; aucune base per-match
  touchée). Les instants sont CAPTÉS au moment de l'événement par un collecteur en mémoire borné
  (`observability.ReportMatchStep`, patron `SetCronRunSink`), puis écrits par la boucle de flush
  monitoring existante (60 s) et à chaque lecture du rapport. Le CLI n'enregistre rien (aucun
  collecteur branché hors serveur : la base monitoring a un seul écrivain).
- **DD-2 Étapes mesurées** (liste fermée) : `match_end` (fin de match API), `detected` (sortie de
  la file du watcher, ou début du cycle qui insère), `stats_written`, `killsource_written`,
  `replay_enqueued` (ouvrier) / `replay_build_started` (local), `replay_claimed` (ouvrier),
  `artifact_stored`, `derivations_written`, `citations_written` (une par joueur suivi, xuid en
  détail). Délai total d'un match = dernière de (`derivations_written`, dernière
  `citations_written` postérieure aux dérivations) − `match_end`. Objectif : ≤ 5 min, constante
  nommée.
- **DD-3 Priorité** : trois niveaux, `domain.BuildPriorityInsere` > `BuildPriorityRecent` >
  `BuildPriorityRattrapage` (0). Récent = match terminé depuis moins de
  `analysis.AnalysisPendingWindow` (24 h, la fenêtre de l'indicateur : ce que l'utilisateur voit
  « en cours » passe devant). Une remise en file d'un job déjà en file à priorité plus basse la
  RELÈVE (événement append-only, `enqueued_at` conservé) ; jamais d'abaissement.
- **DD-4 Local** : quand le cycle a des matchs insérés à cuire, le rattrapage ne cuit pas ce
  cycle-là (il reprend aux cycles sans insertion) — les dérivations et les citations du match frais
  ne passent plus derrière jusqu'à 4 vieux films. La jauge de retard reste publiée.
- **DD-5 Citations à la livraison** : `replayartifacts.Deriver` notifie les matchs dont les
  dérivations sont écrites (marqués) ; un service `internal/service/citationsrelance` les reçoit
  (file bornée, dédoublonnée, une goroutine sur `schedulerCtx`/`schedulerWG`, drainée avant
  `duckdb.CloseAll`) et recalcule, pour chaque joueur SUIVI participant au match, ses citations de
  ce match sous son lease writer joueur — jamais dans le handler HTTP. Prédicat de déclenchement =
  celui de `SelectMatchesFilmPlusRecent` (dérivation plus récente que les citations), factorisé
  (une copie). Joueur occupé (lease non obtenu dans la borne) : journalisé + compteur, son
  prochain post-sync rattrape par le même prédicat ; pas de nouvelle tentative.

## Risques identifiés (hors périmètre, à décider par l'utilisateur SI la mesure les montre)

- R1 : en post-sync, la source des kills (1.57, budget 5 min) passe AVANT la mise en file (1.58).
  Si `killsource_written` → `replay_enqueued` dépasse ~1 min sur la preuve : arrêt, mesure
  rapportée, décision utilisateur.
- R2 : film non publié au premier sync déclenché par le watcher → prochaine tentative au cycle
  suivant (15 min). Même règle.

## Étapes

Statuts : `[x]` fait et vérifié · `[~]` couvert ailleurs (référence) · `[!]` non traité
(justification au journal). Aucune case vide à la clôture d'une étape. Ordre strict : l'étape N
est close (gate passé, items statués, plan + thought_log à jour, point à l'utilisateur) avant
N+1. Une découverte hors périmètre va dans « Découvertes », pas dans le code. Commit par étape,
préfixé `E<n>`, sur accord de l'utilisateur. Caches dédiés :
`GOCACHE=%LOCALAPPDATA%\go-build-delai-debrief`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-delai-debrief`.
Jamais `go build ./...`, jamais `git add -A`, jamais `git stash`.

### Étape 1 — Mesure de bout en bout (A)

- [x] 1.1 `internal/domain/match_steps.go` : type `MatchStep` + les 10 constantes de DD-2,
      `MatchStepEvent{TitleSlug, MatchID, Step, At, Detail}`, `DebriefObjective = 5 * time.Minute`.
- [x] 1.2 `internal/observability/matchsteps.go` : `ReportMatchStep`, `SetMatchStepSink`,
      `DrainMatchSteps` ; tampon borné (4 096), dépassement compté
      (`match_steps_dropped_total`) et journalisé une fois par flush ; sink nil = no-op.
- [x] 1.3 `internal/ops` : table `match_step_events` (création au schéma monitoring, INSERT pur,
      index `(match_id)`), `RecordMatchSteps(ctx, []MatchStepEvent)` (une acquisition de lease
      par lot), `MatchDelayReport(ctx, limit)` ; rétention ajoutée à `monitoring_retention.go`
      (cap + justification).
- [x] 1.4 `internal/analysis/debrief_delay.go` : fonction pure `DebriefDelay(events)` → délai par
      étape depuis `match_end`, délai total (règle DD-2), objectif tenu ou non, étapes absentes
      signalées ; tests unitaires (match complet, ouvrier, local, citations antérieures aux
      dérivations ignorées, `match_end` absent).
- [x] 1.5 Points d'émission (liste fermée) : watcher `daemon.go` (dequeue → `detected`) ;
      `runPostSyncPipeline` début (`stats_written` + `match_end` lu dans le segment de lecture
      shared déjà ouvert, `detected` = début du cycle pour les insérés) ;
      `killcollector.RunPostSync` (passe film écrite → `killsource_written`) ;
      `EnqueueReplayBuild` (créé → `replay_enqueued`) ; `ServiceRegistry.ClaimBuildJob`
      (`replay_claimed`) ; `StoreBuildArtifact` et `cuireUnMatch` (`artifact_stored`) ;
      `buildAll` (`replay_build_started`) ; `marquerDerivations` (`derivations_written`) ;
      `runPostSyncCitations` (`citations_written`, par match calculé).
- [x] 1.6 `internal/api/wire` : la boucle `RunDetectionFlushLoop` vide le collecteur dans le
      store ; `BuildQueueReport` vide puis joint `match_delays` (50 derniers matchs) à
      `domain.AdminBuildQueueResponse` ; sink branché au boot, débranché dans
      `closeMonitoringStore` (comme le sink cron).
- [x] 1.7 OpenAPI régénéré (`make openapi-gen`, `make generate-types`), miroir de types web si
      présent.
- [x] 1.8 Web `features/admin/sync/BuildQueueSection.tsx` : tableau « Délai du débrief »
      (TanStack Table ; colonnes : match, fin, une colonne par étape en « +m:ss », total,
      objectif tenu) ; chaînes FR + EN dans `i18n.ts` ; tokens de couleur uniquement ; état vide
      affiché ; test vitest du formatage (logique dans un `*.logic.ts`).
- [x] 1.9 Commentaire périmé de `engine_options.go:180` (« À N'APPELER QU'EN LOCAL ») réécrit
      selon le contrat réel (hook toujours installé, placement décidé par cycle).

**Gate E1** : `go test ./internal/domain/ ./internal/observability/ ./internal/ops/
./internal/analysis/ ./internal/api/... ./internal/watcher/ ./internal/sync/
./internal/sync/replayartifacts/ ./internal/sync/killcollector/` ;
`go test -tags=integration -p 1` sur `./internal/ops/ ./internal/sync/ ./internal/sync/replayartifacts/`
; `make go-api-lint` 0 issue ; `make check-types` ; `make test-web`.

### Étape 2 — Priorité du match frais (B)

- [x] 2.1 `domain` : constantes `BuildPriorityInsere`, `BuildPriorityRecent`,
      `BuildPriorityRattrapage` (DD-3).
- [x] 2.2 `replayartifacts.EnqueueFunc` et `ServiceRegistry.EnqueueReplayBuild(Job)` prennent la
      priorité ; l'action admin passe `BuildPriorityRattrapage` (comportement inchangé, cf.
      Découvertes D-1).
- [x] 2.3 `buildWork.priorite` posé par `selectionnerLeTravail` : insérés → Insere ; rattrapage
      → Recent si fin de match dans `AnalysisPendingWindow`, sinon Rattrapage (heure canonique
      lue par `lireQueueRecente`, fragment `SQLStartTimeCanonical`, pas `start_time` brut).
- [x] 2.4 `ops.EnqueueBuildJob` : job existant `queued` à priorité inférieure → événement
      `queued` à la priorité demandée, `enqueued_at` d'origine conservé ; `running` intact ;
      jamais d'abaissement.
- [x] 2.5 Local (DD-4) : `selectionnerLeTravail` ne retient pas le rattrapage quand des insérés
      cuisables existent ; jauge de retard toujours mesurée.
- [x] 2.6 Tests : `ops` (prise dans l'ordre Insere > Recent > Rattrapage puis ancienneté ;
      relèvement ; non-abaissement) ; `replayartifacts` (priorités posées ; local sans rattrapage
      quand insérés ; jauge publiée).

**Gate E2** : mêmes commandes que E1 limitées à `./internal/ops/ ./internal/api/...
./internal/sync/replayartifacts/ ./internal/domain/` (unitaire + intégration `-p 1`) ;
`make go-api-lint`.

### Étape 3 — Citations du film à la livraison (C)

- [~] 3.1 Dérivations à la livraison : `[~]` déjà en place (constat 2) — vérifier que les tests
      existants le tiennent (`build_queue` dépôt → `Deriver`, cuisson locale → `Deriver`) ; en
      ajouter un s'il manque.
- [x] 3.2 `DerivationsDeps.ApresDerivation func(ctx, titleSlug string, matchIDs []string)` :
      appelé par `marquerDerivations` avec les seuls matchs marqués ; nil = rien. Branché aux
      trois appelants de `Deriver` (post-sync local, dépôt d'ouvrier, rattrapage des dérivés).
- [x] 3.3 Vérifier sur pièces avant de coder : mode d'ouverture de la base joueur par le
      post-sync (lease tenu tout le pipeline ou non) ; `BackfillMatchCitations` n'écrit pas la
      base partagée (sinon lecture seule impossible → writer borné) ; options du moteur
      (`citationWeaponSourceDuMoteur`) reproduites à l'identique.
- [x] 3.4 Factoriser le prédicat « dérivation plus récente que les citations » de
      `citationsfilm.SelectMatchesFilmPlusRecent` en une fonction par match réutilisée par le
      post-sync et le service (une seule écriture SQL).
- [x] 3.5 `internal/service/citationsrelance` (port dans `internal/port/`) : file bornée
      dédoublonnée, une goroutine ; par match : participants (lecture shared par le provider du
      titre) ∩ profils suivis du titre (`PlayerDirectory`/`ProfileGate`) ; par joueur :
      prédicat 3.4, lease writer joueur borné (constante nommée), `BackfillMatchCitations`,
      `ReportMatchStep(citations_written)` ; échecs journalisés (`slog.WarnContext` + `"err"`) et
      comptés ; jamais de panic.
- [x] 3.6 Câblage au boot (`cmd/server/main.go`) sur `schedulerCtx`/`schedulerWG` ; aucun appel
      depuis un handler (le dépôt ne fait que POSER dans la file, non bloquant).
- [x] 3.7 Multi-titre : le service n'est sollicité que par `Deriver`, déjà gardé par
      `film.replay_artifact` ; chemins par `PathResolver` ; aucun `slug ==`.
- [x] 3.8 Tests : unitaires du service (mocks des ports : joueur non suivi ignoré, prédicat faux
      → rien, lease refusé → compteur, dédoublonnage, arrêt propre) ; intégration
      (`-tags=integration`) : artefact déposé → dérivation → nouvelle génération dans
      `match_citations_latest` pour deux joueurs suivis d'un même match, passe crédit absente
      sans effet.

**Gate E3** : `go test` (unitaire + intégration `-p 1`) sur `./internal/service/citationsrelance/
./internal/sync/ ./internal/sync/replayartifacts/ ./internal/sync/citationsfilm/
./internal/api/wire/ ./internal/archlint/` ; `make go-api-lint` ;
`go test ./internal/sync/ -run NoArt` (garde-rail anti-ART, allowlist inchangée).

### Étape 4 — Gates complets et revue adversariale

- [x] 4.1 `cd apps/go-api && go test ./...` vert.
- [x] 4.2 `go test -tags=integration -p 1` sur tous les paquets touchés.
- [x] 4.3 `make go-api-lint`, `make check-types`, `make test-web`, `make gate-push` (exit 0).
- [x] 4.4 Revue adversariale (skill `adversarial-review`, contexte frais, Opus : lot à risque
      persist/sync) ; constats recevables corrigés ou statués par écrit ici.
- [x] 4.5 Commit(s) sur accord, push `feat/delai-debrief`, CI verte au niveau job.

### Étape 5 — Preuve sur vrais matchs (D)

Dépendance utilisateur : jouer un match 4v4 par mode. Une cuisson par match, par le pipeline
normal (aucune cuisson en lot ni en boucle).

- [x] 5.1 Préalables : aucun détenteur de `data/cache/film_decode.lock.json`, aucun traitement de
      recherche en cours ; serveur local relancé sur la branche (air, port 8000 vérifié).
- [x] 5.2 Mode local (`replay_build_location=local`) : un match 4v4 joué ; rapport « Délai du
      débrief » relevé (capture des valeurs dans le journal du plan) ; total ≤ 5 min.
- [x] 5.3 Mode ouvrier : jeton d'ouvrier local, `go run ./cmd/replay-worker` sur le poste,
      réglage `worker` ; un match 4v4 joué ; rapport relevé ; total ≤ 5 min.
- [x] 5.4 Si un total dépasse 5 min : arrêt, ventilation par étape rapportée à l'utilisateur
      (R1/R2 ou autre), aucune adaptation silencieuse.

### Étape 6 — Livraison

- [x] 6.1 Entrée `.ai/thought_log.md` ; ce plan à jour (cases, journal, découvertes) ;
      `.ai/REGISTRE_REPORTS.md` alimenté des découvertes non traitées.
- [x] 6.2 Fusion dans `feat/v75` sur accord explicite de l'utilisateur ; CI verte au niveau job
      sur `feat/v75`.
- [x] 6.3 Session de release prévenue (`ListAgents` puis `SendMessage`).
- [x] 6.4 `powershell -File scripts/disk-hygiene.ps1 -Lot delai-debrief -Apply` ; worktree
      retiré sans `--force`.

## Reprise de session

Relire `plan-execution`, puis ce fichier : reprendre à la première case non statuée de l'étape
courante. `git -C ../LevelUp-wt-delai-debrief log --oneline -10` pour l'état des commits.

## Découvertes (notées, non traitées)

- D-1 L'action admin « construire » n'a jamais posé de priorité alors que le commentaire de
  `ClaimBuildJob` (`ops/build_queue.go:121-122`) promet qu'« un rattrapage massif ne double jamais
  un match qu'un admin vient de demander ». Hors périmètre : elle reste au niveau Rattrapage (même
  service qu'avant pour elle).

- D-2 (revue ronde 2, P2) : depuis que le post-sync V2 tient le lease joueur, un sync du watcher
  du MÊME joueur qui tombe pendant ce post-sync voit son lot asynchrone attendre le lease : le
  worker unique du `CombinedPersister` (`internal/persist/combined_persister.go:171-174`) épuise
  ses 3 × 30 s (~93 s, tous joueurs en attente derrière lui), puis le lot repasse par le WAL
  (reprise périodique, jusqu'à 10 min). Même classe que l'existant (un post-sync V1 qui tient
  le lease bloque de même un lot V2) ; le cycle V2 ne réclame pas le `SyncGate` par joueur
  (`internal/scheduler/auto_sync.go` `runOnceV2`). Non traité : la mesure de l'étape 5 dira
  si le cas se produit.
- D-3 (revue ronde 2) : `replayartifacts/backlog.go:220` garde une construction de placeholders
  à la main (code antérieur au lot).
- D-4 : `internal/platform/duckdb` (non touché par le lot) frôle le budget de 300 s du test de
  baseline de `make gate-push` : 290 s seul en intégration, 300,3 s sous charge (un traitement de
  recherche tournait) — le paquet a été compté « absent » ce jour-là. Rejoué seul : vert.

## Journal

- 2026-10-17 : constats vérifiés sur pièces, plan écrit, soumis à l'utilisateur. Aucun code.
- 2026-10-18 : plan validé par l'utilisateur (DD-1 à DD-5 ; preuve par deux vrais matchs joués,
  pas de suppression de match ni de simulation comme preuve). Étape 1 exécutée. Écarts au texte
  des items, tranchés sur pièces :
  - 1.2 : pas de « sink » branchable, un collecteur ARMÉ par `WithMonitoringStore` et désarmé par
    `closeMonitoringStore` (`EnableMatchSteps`) ; remise en tête (`RequeueMatchSteps`) d'un lot
    dont l'écriture échoue. Le CLI ne l'arme jamais.
  - 1.3 : colonne `step_at` (`at` est un mot réservé DuckDB) ; AUCUN index secondaire (la table
    est purgée par DELETE de rétention, un index ART de plus serait exposé au bug #23645) ; lecture
    nommée `MatchDelays`.
  - 1.5 : `detected` vient du watcher seul (un cycle planifié n'a pas d'instant de détection
    distinct de l'écriture : l'étape est alors absente plutôt qu'inventée) ; `stats_written` et
    `match_end` par le sous-package `internal/sync/debrief` (racine de `internal/sync/` gelée) ;
    `killsource_written` dans `marquerFilm`, seul point traversé par les deux boucles du
    collecteur ; `citations_written` dans la boucle de `BackfillMatchCitations`, APRÈS chaque
    écriture réussie (couvre le post-sync et le futur service de l'étape 3, jamais un match
    sauté) ; titre lu dans le contexte (`ctxkeys.TitleSlug`).
  - 1.8 : tableau statique `AdminTable`, comme ses voisins de la section (aucune interaction : la
    règle TanStack vise les tableaux interactifs) ; chaînes dans `manifests/admin.toml` (source du
    i18n admin), générées par `scripts/build_i18n_manifests.mjs`.
  - 1.9 : la même affirmation périmée existait en trois copies (`engine_options.go`, `engine.go`,
    en-tête de `replayartifacts/artifacts.go`) : les trois corrigées.
- 2026-10-18 : étape 2 exécutée. Écarts tranchés sur pièces :
  - 2.3 : « récent » se juge sur l'âge de SYNCHRO (`first_sync_at`, en Go contre
    `analysis.AnalysisPendingWindow`), la règle même de l'indicateur « Analyse en cours », et non
    sur la fin du match ; lu par une requête courte sur les seuls matchs retenus
    (`prioriserLeRattrapage`) plutôt qu'en ajoutant une colonne à `requeteQueueRecente`, partagée
    avec deux autres lecteurs. Une première écriture en SQL (`first_sync_at >= ?`) a été refusée
    par le garde-rail `no_retention_window_inline_test.go` : l'âge se juge en Go.
  - 2.2 : `EnqueueReplayBuild` (action admin) garde sa signature et la priorité de rattrapage ;
    `EnqueueReplayBuildJob` (fil de l'eau) porte la priorité de la sélection.
  - Seuil de 500 lignes : `registry_build_queue.go` (500 après 2.2) a été découpé —
    dérivations du dépôt dans `registry_build_queue_derivations.go` ; `build_queue.go` (509) —
    relèvement dans `build_queue_priority.go`. Garde-rails suivis : entrée d'allowlist
    `no_shared_writer_on_request_path_test.go` déplacée sur le nouveau fichier (même prise de
    writer), motif `registry_build_queue*.go` dans `no_replay_firefight_literal_test.go`, lecture
    de source de `build_queue_read_segment_test.go`.
  - Commentaire de `ClaimBuildJob` corrigé : il promettait que l'action admin passait devant le
    rattrapage, ce qu'aucun appelant ne faisait (D-1).
- 2026-10-18 : étape 3 exécutée. Constats sur pièces (3.3) et écarts :
  - 3.3 : le post-sync tient le lease writer de la base joueur (`dblease.KindPlayer`) pendant
    TOUT le pipeline (`engine.go`, post-drain) ; `BackfillMatchCitations` n'écrit que la base
    joueur (aucune écriture shared) — la relance lit la base partagée par le provider du titre,
    sans writer ; options du moteur reproduites par `citationWeaponSourceDuTitre` (une seule
    écriture, utilisée par le moteur et par `sync.RecalculerCitations`).
  - 3.2 : pas de champ `ApresDerivation` dans `DerivationsDeps` mais un notifieur unique du
    processus (`replayartifacts.SetNotifieurDerivations`), appelé par `marquerDerivations`
    avec les seuls matchs écrits : les trois appelants de `Deriver` en héritent sans trois
    chaînes d'injection (scheduler, V2, handler), et un CLI ne le branche jamais. Branché et
    débranché par la tâche de fond `citationsRelanceTask` (`cmd/server/background_tasks.go`,
    coupée en démo : elle écrit les bases joueurs).
  - 3.5 : la relance ne recalcule QUE si le joueur a déjà des citations du match et que le film
    a été dérivé après elles (`citationsfilm.FilmPlusRecentPourLeMatch`) : un premier calcul
    sans les récompenses du joueur (étape 1.56 de SON post-sync) serait faux et ne serait plus
    jamais repris. Joueur occupé : borne `citationsfilm.AttenteBaseJoueur` (20 s). Travail
    base dans `internal/sync/citationsfilm/relance.go` (lease, ouverture, prédicat, calcul
    injecté) ; file et boucle dans `internal/service/citationsrelance` ; câblage dans
    `internal/api/wire/registry_citations_relance.go` (port `port.FilmCitationsRecomputer`).
  - 3.8 : la chaîne est tenue par morceaux plutôt que par un seul test de bout en bout : dépôt →
    `Deriver` (test existant `TestStoreBuildArtifact_DeclencheLesDerivations`), `Deriver` →
    notifieur (`notifieur_test.go`), service multi-joueurs (`service_test.go`, deux suivis + en
    pause + auth seule + absent), recalcul réel sur une base joueur fichier avec lease
    (`citations_relance_integration_test.go`), lecture des participants par le provider
    (`registry_citations_relance_cgo_test.go`). La preuve de bout en bout est l'étape 5.
  - 3.1 `[~]` : déjà en place, tenu par `TestStoreBuildArtifact_DeclencheLesDerivations` (dépôt)
    et `TestDeriver_LesAppelantsTransmettentLeSegmentDeLecture` (cuisson locale, rattrapage).
  - Ajout après le premier gate : PRÉ-VÉRIFICATION EN LECTURE (`duckdb.OpenReadForQuery`, sans
    lease) avant d'attendre la base joueur — une relance sans objet ne patiente plus 20 s
    derrière un post-sync, et ne laisse pas de WARN « dblease ctx cancelled » dans les détections.
- 2026-10-18 : étape 4, gates 4.1 (`go test ./...`) et 4.2 (intégration `-p 1` des 14 paquets
  touchés) VERTS. Revue adversariale ronde 1 : deux relecteurs Opus aveugles (L1 DuckDB /
  concurrence ; L6 + L3 tests et règles). Constats recevables :
  - R1 (trouvé par LES DEUX, P1 retenu) : le post-sync du cycle V2 (`cmd/server/sync_v2_wiring.go`
    `playerDBOpenerRW`, `internal/sync/v2/post_sync_runner.go`) ouvre la base joueur en écriture
    SANS le lease `KindPlayer` que tient le post-sync V1. La garde « base tenue » de la relance ne
    protège donc pas d'un post-sync V2 : deux recalculs du même match, et la ligne de base
    cumulée (`loadCumulExcluding`) du post-sync peut lire une génération à moitié écrite par la
    relance. Défaut antérieur de V2 (V1 et V2 se chevauchent déjà), que la relance rend
    atteignable par un nouvel écrivain. DÉCISION DEMANDÉE À L'UTILISATEUR (modèle de verrou du
    moteur V2).
  - R2 (P1, corrigé) : le rapport triait les matchs sans fin mesurée par leur dernier instant ;
    un rattrapage ou un recalcul de vieux matchs chassait les matchs frais du tableau. Le rapport
    ne retient plus que les matchs dont la fin est mesurée (test étendu).
  - R3 (P2, corrigé, code du lot) : « morts lues » était capté pour une lecture à refaire
    (`OutcomeSansKillFeedARelire`). Retiré ; test `killcollector/debrief_steps_test.go` ajouté.
  - R4 (P2, corrigé) : l'ajout dans `runPostSyncPipeline` ramené à une ligne
    (`reportDebriefInserted`) ; « stats écrites » n'est plus conditionné à la lecture partagée.
  - R5 (P2, corrigé) : deux copies à la main de la liste de `?` remplacées par
    `duckdbpkg.Placeholders` / `ToAnySlice`.
  Conditions vérifiées qui tiennent : 17 (L1) et 26 (L6+L3).
  - R1 tranché par l'utilisateur le 2026-10-18 (« V2 prend le verrou ») : `postSyncPlayerOpener`
    (`cmd/server/sync_v2_wiring.go`) ouvre la base joueur du post-sync V2 sous le lease
    `KindPlayer`, tenu jusqu'à la fermeture, comme V1 ; test `sync_v2_wiring_lease_test.go`.
    Gates rejoués verts (intégration sync/v2, scheduler, cmd/server, ops, sync, debrief,
    killcollector, replayartifacts, api/wire, archlint ; lint 0 issue).
- 2026-10-18 : revue ronde 2 (un relecteur Opus aveugle, sur le seul commit de corrections) :
  0 P0/P1 (contre 2 en ronde 1 : convergence), 1 P2 consigné en D-2, 14 conditions vérifiées
  qui tiennent dont l'absence d'interblocage (aucune réacquisition du lease joueur dans le
  pipeline ni dans les hooks, ordre joueur puis shared cohérent avec V1). Arrêt des rondes (§8).
- 2026-10-18 : 4.3 — `make go-api-lint` 0 issue ; `make check-types` et eslint (0 erreur) dans
  `make gate-push` ; `make test-web` 962 fichiers / 10 020 tests. `make gate-push` rouge sur sa
  seule étape baseline : `internal/platform/duckdb` hors budget sous charge (D-4), rejoué seul en
  intégration (`-p 1 -timeout=300s`) : vert en 290 s. La CI fait foi (4.5).
- 2026-10-18 : étape 5 en attente, deux dépendances : (1) l'utilisateur ne peut pas jouer les
  deux matchs aujourd'hui (report valide : action que seul l'utilisateur peut faire) ; (2) un
  traitement de recherche d'une autre session (`replay-parc-vies`) tient le verrou de décodage du
  poste — session levelup-50 sollicitée pour en connaître la fin. CI de la branche en cours.
- 2026-10-18 : DÉCISION UTILISATEUR (message du 2026-10-18) : fusionner dans `feat/v75` dès la CI
  verte, la preuve de l'étape 5 se fera ensuite (en local quand l'utilisateur pourra jouer, ou en
  prod après le déploiement, par le tableau « Délai du débrief »). Le traitement de recherche de
  levelup-50 finit vers 19 h 45 ; une autre passe longue sera annoncée avant d'être lancée.
- 2026-10-18 : 4.5 CI de la branche verte (run 39128534319). 5.1 à 5.4 `[!]` : preuve reportée par
  décision de l'utilisateur, consignée au registre des reports avec sa condition de reprise.
  6.1-6.2 : fusion `1b01444a1` dans `feat/v75` ; CI de `feat/v75` suivie au niveau job.
- 2026-10-18 : CI de `feat/v75` verte sur la fusion (run 39131709680, tous les jobs ; E2E réservé
  aux PR vers main). 6.3 : sessions pairs prévenues (levelup-b5, levelup-34, levelup-22 ; la
  session de release n'étant pas identifiable par son nom). 6.4 : caches dédiés du lot purgés,
  worktree retiré. Reste ouvert : l'étape 5 (preuve), au registre des reports — l'utilisateur la
  fera quand le traitement de recherche de levelup-50 sera fini. Le serveur local tourne déjà sur
  le code fusionné (relancé à 16 h 02).
- 2026-10-19 : ÉTAPE 5, PREUVE FAITE sur deux vrais matchs 4v4 joués par l'utilisateur, verrou de
  décodage libre, serveur local sur `feat/v75` (lot inclus). Instants lus dans une copie de la
  base monitoring (serveur arrêté une minute, puis relancé). Écart à la fin du match :
  | Étape | Match 1, local (`53ca795f`) | Match 2, ouvrier (`680e0b32`) |
  |---|---|---|
  | détecté (watcher) | +0:17 | +0:07 |
  | stats écrites | +0:20 | +0:41 |
  | morts lues | +1:14 | +1:27 |
  | rejeu lancé (local) / mis en file (ouvrier) | +2:40 | +2:31 |
  | pris par l'ouvrier | — | +2:32 |
  | rejeu rangé | +2:49 | +2:38 |
  | dérivations écrites | +2:49 | +2:41 |
  | citations après les dérivations | +2:49 | +2:45 (relance à la livraison) |
  | **total** | **2 min 49 s, tenu** | **2 min 45 s, tenu** |
  Le job du match 2 est entré en file à la priorité 100 (inséré). Les citations du match 2 ont
  été écrites deux fois : par le post-sync (+2:31, avant le film) puis par la relance dès les
  dérivations (+2:45) — c'est la relance qui livre les citations du film sans attendre le cycle
  suivant. Plus gros poste dans les deux cas : de « morts lues » à « rejeu lancé » (86 s et 64 s),
  l'étape de lecture des morts traitant son retard (7 vieux matchs) avant de passer la main au
  rejeu (risque R1, hors périmètre, sans effet sur l'objectif). Mode ouvrier monté pour la mesure
  (jeton temporaire dans `.env.local`, réglage `worker`, ouvrier local) puis RETIRÉ : `.env.local`
  et `app_settings.json` restaurés à l'octet, ouvrier arrêté, serveur relancé en local.
