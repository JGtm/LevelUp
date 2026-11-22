# Plan : correctif 8.2.2, rattrapage de l'historique par la synchronisation automatique

Date : 2026-11-22. Contrat d'exécution : skill `plan-execution` (avec `levelup-plan`).
Branche d'intégration : `release/v8.2.2`, créée depuis `origin/main` (fee334a039, v8.2.1).
Worktree d'intégration : `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-822`.
Origine : issue GitHub #213 (joueur sur l'application Windows 8.2.1, ~7 000 matchs, historique bloqué).

## Objectif

Toute synchronisation automatique (cycle périodique, lancement, fermeture du jeu, « Synchroniser
maintenant » du menu de la zone de notification) rattrape l'historique d'un joueur jusqu'à son premier
match, Halo Infinite comme Halo 5, puis redevient un delta. Critère observable : un test de bout en bout
sur un faux historique de 1 000 matchs montre que des cycles successifs du VRAI chemin automatique
rangent les 1 000 matchs, rangent la fin d'historique, puis que le cycle suivant ne parcourt plus qu'une
page.

## Constat (vérifié sur pièces le 2026-11-22)

- Le rattrapage livré en 8.1.2 (`sync_meta.history_complete_at`, `internal/sync/engine.go:486`,
  `internal/sync/writes.go:85-128`) ne vit que dans le moteur V1 (`SyncEngine.run`).
- Le cycle automatique passe par le pipeline V2 (`internal/scheduler/auto_sync.go` `runOnceV2`,
  ADR 0027). Sa découverte `matchListProviderV2.ListUnknownMatches`
  (`internal/sync/v2/fetchers.go:91`) s'arrête au premier match connu, au plus 20 pages, et ignore
  l'état de l'historique. Le tick manuel du menu (`cmd/server/boot_shell.go:170`,
  `RunOnceTrigger`) et le rattrapage au lancement (`internal/wakeup`) prennent le même chemin.
- Halo 5 : `livesync.Runner.RunDelta` (`internal/games/halo_5/livesync/runner.go:113`) s'arrête au
  premier connu ; le parcours complet (`livesync/backfill.go`) n'existe qu'en commande.
- Les tests du rattrapage 8.1.2 ne passaient que par le moteur V1 : aucun test ne jouait le chemin
  que l'ordonnanceur appelle vraiment. C'est le trou que ce plan ferme (étape 5).

## Décisions

- **D-1** Version 8.2.2 (correctif), depuis `main`. Après le squash de la 8.2.2 par l'utilisateur,
  `main` est fusionnée dans `release/v8.3.0` (geste du pilote, sur accord).
- **D-2** Une seule règle, une seule source : un paquet `internal/sync/historystate` porte la clé
  `history_complete_at` (nom inchangé : les joueurs déjà marqués par V1 restent marqués), sa lecture
  (`Complete(ctx, db) bool`, lecture en échec = complet, journalisé, comme aujourd'hui), son écriture
  (`MarkComplete`), le budget par passe et la borne de sûreté. V1, V2 et Halo 5 l'appellent ; plus
  aucune copie de la clé ni de la logique ailleurs (garde-rail grep, D-9).
- **D-3** Budget par passe de rattrapage : 200 matchs inconnus, `domain.DefaultSyncOptions().MaxMatches`
  (valeur unique, pas de nouvelle constante). Borne de sûreté de profondeur : 50 000 matchs (constante
  `maxHistoryMatchesPerPass` de V1, déplacée dans `historystate`).
- **D-4** Halo Infinite, V2 : quand l'historique d'un joueur n'est pas complet, la découverte ne
  s'arrête plus au premier connu : elle saute les connus, garde les inconnus dans l'ordre de l'API
  jusqu'au budget, et continue jusqu'à la page vide ou partielle (fin d'historique vue) ou jusqu'au
  budget atteint (fin non vue). Historique complet : comportement actuel inchangé (arrêt au premier
  connu, 20 pages).
- **D-5** La fin d'historique se range seulement si, pour CE joueur : la découverte a vu la fin, aucune
  erreur de découverte, la persistance du cycle a réussi, et chaque match inconnu listé pour lui a été
  persisté ou rendu « introuvable » par l'API (404/410, compté et journalisé). Toute autre erreur de
  récupération laisse la clé absente : la passe suivante reprend (les connus sont sautés). L'écriture de
  `sync_meta` se fait sous le bail d'écriture du joueur (`AcquirePlayerWriterTimeout`, règle ART n°4).
- **D-6** Halo 5 : `Runner.RunDelta`, historique incomplet = passe de rattrapage par la mécanique de
  `livesync/backfill.go` (connus sautés, persistance page par page), bornée au budget D-3 en matchs
  insérés, fin rangée à la page vide sans erreur (règle D-5). La commande `h5-backfill` garde son
  comportement (sans budget).
- **D-7** Cadence : tant qu'au moins un joueur du cycle a un historique incomplet, l'ordonnanceur
  replanifie le cycle suivant après 10 minutes (constante nommée `catchUpCycleInterval`) au lieu de
  l'intervalle réglé ; dès que tous sont complets, retour à l'intervalle réglé. Tous les profils de
  déploiement. Le résultat de cycle porte l'information par joueur (`HistoryPending`). Journal :
  une ligne par cycle qui dit combien de joueurs rattrapent et l'intervalle choisi.
- **D-8** Le tick du suivi en direct (watcher, 25 matchs) reste tel quel : il passe par V1, qui suit
  déjà la règle (via `historystate` après migration).
- **D-9** Garde-rail : un test `archlint` interdit le littéral `"history_complete_at"` hors de
  `internal/sync/historystate`.
- **D-10** (révisée le 22/11 par l'utilisateur) Bouton « Synchroniser maintenant » dans Réglages ›
  Données, visible par tout joueur, DANS la 8.2.2 : il lance le même cycle que l'automatique
  (`RunOnceTrigger`), donc le rattrapage compris (étape 4bis). L'état de l'historique à l'écran :
  décision de l'utilisateur en attente (D-15). L'aide est corrigée (étape 6) et renvoie à ce bouton.
- **D-12** (utilisateur, 22/11) Tout l'historique, sans limite de date ni de nombre. Le choix d'une
  limite est une ligne du backlog.
- **D-13** (utilisateur, 22/11) Marque-page au lieu du rebalayage : `historystate` range, par joueur,
  la position où la passe de rattrapage s'est arrêtée (décalage dans la liste de l'API et identifiant du
  dernier match traité, clé `sync_meta` à côté de `history_complete_at`). La passe suivante : (1) delta
  du dessus jusqu'au premier connu, qui compte N nouveaux matchs ; (2) reprise au marque-page + N moins
  une page de recouvrement ; (3) la page de recouvrement doit contenir l'identifiant rangé, sinon
  reprise depuis le début (lente, sûre), journalisée et comptée. Un match en échec ramène le marque-page
  avant lui. Fin vue = les deux clés : fin rangée, marque-page effacé. Halo 5 suit la même règle si sa
  liste se pagine par décalage ; sinon écart écrit dans le rapport, rebalayage gardé pour Halo 5 seul.
- **D-14** Avant de figer le budget (200) et la cadence (10 min) : mesure d'UNE passe de rattrapage de
  200 matchs sur une copie de base (durée, appels API, pic mémoire, disque des films), étape 5bis. Si
  la passe dépasse 5 minutes ou 1,5 Go de pic, le budget ou la cadence se règlent sur la mesure (le
  pilote propose, l'utilisateur tranche).
- **D-15** (utilisateur, 22/11) État de l'historique à l'écran, version simple, élégante, aux normes UI
  de l'app (étape 4ter) : dans l'onglet « Données & Médias », à côté du bouton, par titre suivi : état
  (complet / rattrapage en cours), matchs enregistrés, date du plus ancien, dernière passe (heure,
  matchs ajoutés, erreur éventuelle), prochaine passe. Texte seul, pas de graphique. Mise à jour
  automatique tant qu'un rattrapage tourne. Lecture seule côté serveur.
- **D-16** (pilote, 22/11) Un match qui échoue toujours ne garde pas le joueur en rattrapage à vie. Le marque-page
  (`history_catchup_cursor`) porte aussi la liste ordonnée des matchs en échec de rattrapage avec leur nombre de
  passes (`failed: [{id, attempts}]`, du plus ancien au plus récent, au plus 200 entrées, les plus anciennes sortent ;
  une liste ordonnée plutôt qu'un objet, pour savoir qui est le plus ancien). Un match en échec à `maxCatchUpAttempts`
  (3) passes est abandonné : il n'est plus retenté par le rattrapage, ne compte plus dans le budget, ne retient ni la
  fin ni le marque-page ; journalisé (`slog.Warn`, identifiant et dernière erreur) et compté
  (`sync_v2_history_abandoned_total`) au moment de l'abandon, dans la passe du dernier échec. La fin se range quand la
  fin est vue et que les seuls inconnus restants sont abandonnés ; le journal de fin dit combien. Un 404/410 reste
  « introuvable » : jamais compté comme échec. Un match abandonné qui sort de la liste (borne de 200) redevient un inconnu
  ordinaire.
- **D-11** Rien ne change dans la base : aucune migration, aucune table. Clé `sync_meta` existante.
- Réservé à l'utilisateur : squash dans `main`, tag `v8.2.2`, publication de la release et de la mise à
  jour de l'application, réponse publique sur #213 (texte validé par lui le 22/11).

## Pistes et modèles

Chemin critique : étape 1 (paquet partagé), puis pistes A, B, C en parallèle, puis intégration.

| Étape | Piste | Modèle | Worktree / branche | Effort |
|---|---|---|---|---|
| 1 | socle | Sonnet | `LevelUp-wt-822-socle` / `fix/822-socle` | rapide |
| 2 | A Infinite V2 | Sonnet | `LevelUp-wt-822-a` / `fix/822-a` | moyen |
| 3 | B Halo 5 | Sonnet | `LevelUp-wt-822-b` / `fix/822-b` | moyen |
| 4 | C cadence | Sonnet | `LevelUp-wt-822-c` / `fix/822-c` | rapide |
| 4bis + 4ter | E bouton et état de l'historique | Sonnet | `LevelUp-wt-822-e` / `fix/822-e` | moyen |
| 5bis | mesure d'une passe | Sonnet | copie de base dans le scratchpad | rapide |
| 5 | convergence bout en bout | Sonnet | `LevelUp-wt-822` (intégration) | moyen |
| 6 | aide, version, notes | Haiku | `LevelUp-wt-822-d` / `fix/822-d` | rapide |
| 7 | revue adversariale du diff cumulé | Fable | lecture seule | moyen |
| 8 | gates complets, journal, PR | pilote | `LevelUp-wt-822` | moyen |

Caches : chaque agent pose `GOCACHE=%LOCALAPPDATA%\go-build-822<piste>` et
`GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-822<piste>` ; clôture par
`powershell -File scripts/disk-hygiene.ps1 -Lot 822<piste> -Apply`.

## Étapes

### Étape 1 — paquet `internal/sync/historystate` (socle)

- [x] 1.1 Paquet `historystate` : clé, `Complete`, `MarkComplete` (sous bail joueur tenu par l'appelant,
  documenté dans le paquet), `PassBudget()` (= D-3), `MaxDepth` (= D-3). `MarkComplete` rend un booléen
  (clé rangée ou non). Lecture et écriture de `sync_meta` déplacées dans le paquet `internal/sync/syncmeta`
  (évite le cycle `sync` -> `historystate` -> `sync` ; `sync.GetSyncMeta` / `SetSyncMeta` supprimées, appelants migrés).
- [x] 1.2 V1 migré : `engine.go` et `writes.go` appellent `historystate` ; anciennes fonctions
  `historyComplete` / `markHistoryComplete` / `SyncMetaHistoryCompleteAt` supprimées avec leurs usages.
- [x] 1.3 Champ `HistoryPending bool` sur `domain.SyncResult` (V1 et H5) et sur l'issue par joueur du
  cycle V2 (`PlayerOutcome`), sans encore le remplir hors V1 ; V1 le remplit (incomplet en fin de passe).
- [x] 1.4 Garde-rail D-9 dans `internal/archlint/` (`history_key_single_source_test.go`, littéraux lus dans l'AST).
- [x] 1.5 Tests unitaires `historystate` sur DuckDB `:memory:` ; tests V1 existants du rattrapage verts
  sans modification de leur intention.
- Gate : `cd apps/go-api && go build ./... && go vet ./internal/sync/... ./internal/archlint/... &&
  go test ./internal/sync/... ./internal/archlint/... ./internal/domain/...`.

### Étape 2 — piste A : découverte V2 (Halo Infinite)

- [x] 2.0 Marque-page (D-13) ajouté à `historystate` (`LoadCursor`, `SaveCursor`, `ClearCursor`, type `Cursor`, clé `history_catchup_cursor`), tests. Commit `beb37a3eb8` (fix/822-a), à fusionner tôt : la piste Halo 5 en dépend.
- [x] 2.1 `KnownLoader.LoadKnown` rend `KnownState` (ensemble connu + `HistoryState{Incomplete, Cursor}`, lus sur la base joueur déjà ouverte).
- [x] 2.2 `ListUnknownMatches` : deux modes (D-4, D-13), rend `Listing{Unknown, EndSeen, Proposed}` (`v2/fetchers_history.go`).
- [x] 2.3 `cycle.go` + `v2/history.go` : après une persistance réussie, range la fin ou le marque-page pour chaque
  joueur qui remplit D-5 ; remplit `HistoryPending` (sorties anticipées comprises) ; introuvables reconnus par
  `sync.IsNotFound` (verdict typé de la brique halo-api, 404/410) ; « persisté » prouvé par relecture de l'ensemble connu.
- [x] 2.4 Câblage `cmd/server/sync_v2_wiring.go` : `WithHistoryWriter(playerDBOpenerRW)`, l'ouverture sous bail joueur du post-sync
  (`postSyncPlayerOpener`) ; ratchet `sync_v2_wiring_history_test.go`.
- [x] 2.6 D-16 : échecs de rattrapage comptés dans le marque-page, abandon à 3 passes (`v2/history_settle.go`, `historystate.LoadCatchUp` / `SaveFailures`), tests (abandon à la 3e passe puis fin rangée, échec puis succès, borne de 200, 404 jamais un échec).
- [x] 2.5 Tests : découverte (incomplet : saute les connus, budget, fin vue, page partielle, borne de
  sûreté ; complet : inchangé) ; cycle (fin rangée seulement sous D-5 : cas erreur de découverte, erreur
  de récupération, introuvable, persistance en échec).
- Gate : `go test ./internal/sync/v2/... ./cmd/server/...` puis
  `go test -tags=integration ./internal/sync/... ./internal/persist/...`.

### Étape 3 — piste B : Halo 5

- [x] 3.1 `Runner.RunDelta` : historique incomplet = passe de rattrapage par `RunBackfill` (option
  `BackfillDeps.Budget`, 0 = sans borne : `h5-backfill` inchangé ; boucle non dupliquée), budget =
  `historystate.PassBudget()` ramené au `MaxMatches` de l'appelant s'il est plus serré. Nouveau fichier
  `livesync/catchup.go`. Connus lus UNE fois par `loadKnownMatchIDs` (aucune nouvelle lecture du connu).
  Base joueur h5 : `sync.OpenPlayerDB` (handle RW ref-compté, comme les autres hooks du runner).
- [x] 3.2 Fin rangée sous D-5 par `historystate.MarkComplete`, sous le bail
  `dblease.AcquireWriterCtx(KindPlayer)` (attente bornée à `PlayerLeaseTimeout`) ; `HistoryPending` et
  `HistoryExhausted` remplis (pending vrai si l'état est illisible ou la source échoue avant).
  `halo5.CaptureStats.HistoryEnd` ajouté : vrai seulement sur page vide ou incomplète, faux sur
  l'anti-boucle (page qui reprend des matchs vus).
- [x] 3.3 Tests : `catchup_test.go` (passe bornée, trois passes jusqu'à la fin, erreur de page, erreur
  de persistance, timeline manquante = pas une erreur, anti-boucle, écriture de la fin échouée, complet =
  delta, état illisible, source KO, known-set KO, sans hooks, budget optionnel de `RunBackfill`),
  `capture_test.go` (`HistoryEnd`), `catchup_integration_test.go` (base joueur temporaire : hooks, bail
  tenu, trois passes).
- [x] 3.4 Marque-page Halo 5 (D-13) : delta du dessus, reprise à marque-page + N moins une page, contrôle
  de l'identifiant sur la page de recouvrement, reprise depuis 0 journalisée et comptée ; abandon d'une page
  après 3 passes en échec (D-16 adapté, `historystate.Streak`) ; `backfillMaxPages` remplacé par
  `historystate.MaxDepth`. Tests : reprise avec N neufs, recouvrement raté, page durable abandonnée à la 3e
  passe puis fin rangée, page en échec une fois puis réussie (compte remis à zéro), erreur de persistance
  durable abandonnée.
- Gate : `go test ./internal/games/halo_5/...`.

### Étape 4 — piste C : cadence de l'ordonnanceur

- [x] 4.1 `catchUpCycleInterval` = 10 min ; après chaque cycle (V2 et filet `syncPlayer`), le ticker
  est recalé sur 10 min si un joueur a `HistoryPending`, sinon sur l'intervalle réglé (D-7). Fait
  (`scheduler/auto_sync_loop.go`). Un joueur dont la passe a échoué avant de savoir (issue `failed`,
  absente de l'issue du cycle, erreur globale du cycle, `syncPlayer` en échec) est lu par le port
  `port.HistoryCompletionReader` (`platform/duckdb/historyrepo`, `OpenReadForQuery`, sans écriture) :
  sans clé de fin il compte comme en rattrapage. Décision : si l'intervalle réglé est plus court que
  10 min, il garde la main (le rattrapage ne ralentit jamais). Le compte à rebours part de la fin du
  cycle (le ticker est recalé après chaque cycle). `NextCycleAt()` exposé (sert à 4ter).
- [x] 4.2 Ligne de journal par cycle (`auto_sync: cycle suivant planifié` : joueurs en rattrapage,
  intervalle choisi) ; auto-sync désactivée = aucun cycle, comme aujourd'hui.
- [x] 4.3 Tests avec horloge et ticker injectés (`scheduler/auto_sync_cadence_test.go`) : bascule
  6 h -> 10 min -> 6 h, joueur Halo 5, issue en échec et erreur de cycle selon la clé de fin,
  désactivé, intervalle court, intervalle réglé qui change, `Run` avec ticker injecté.
- Gate : `go test ./internal/scheduler/...`.

### Étape 4bis — bouton « Synchroniser maintenant » (Réglages › Données)

- [x] 4b.1 Route serveur qui lance le cycle de l'ordonnanceur (`RunOnceTrigger`, raison `manual`),
  refus propre si un cycle tourne déjà (409 + message i18n), auto-sync désactivée = le bouton lance
  quand même un cycle (décision : un clic explicite passe outre le réglage). Pas de logique dans le
  handler (port vers l'ordonnanceur). Fait : `POST /api/v1/sync/now` (202 `SyncNowResponse`, 409
  `sync_cycle_busy` avec message du catalogue `sync.now.busy`, 503 `scheduler_unavailable`) et
  `GET /api/v1/sync/cycle` (200 `SyncCycleState{running}`), `RequireAuth` sans `RequireAdmin` (tout
  joueur connecté ; la démo est refusée par `RequireAuth`). Aucune route existante ne convenait :
  `POST /sync/all` et `POST /admin/actions/auto-sync/run` sont réservées à l'administrateur,
  `POST /players/{slug}/sync` lance le moteur V1 d'un seul joueur. Le port `port.SyncNowService` est
  porté par l'ordonnanceur (`StartSyncNow`, `CycleRunning`) ; `RunOnceTrigger` refuse désormais un
  second cycle simultané (résultat vide `busy`), et un cycle hors tick recale la cadence de `Run`
  (`replanFromCycle`). Cycle lancé sous le contexte et le `WaitGroup` du serveur (`BindManualCycles`).
- [x] 4b.2 Bouton dans l'onglet `data` des réglages, visible pour tout joueur et tout profil de
  déploiement sauf démo ; textes FR/EN ; état « en cours » tant que le cycle tourne. Fait
  (`features/settings/SyncNowCard.tsx`, gardé par la capacité `can_run_sync` du bootstrap, jamais par
  le profil) ; « en cours » lu sur `GET /sync/cycle`, relu toutes les 3 s tant qu'il tourne.
- [x] 4b.3 Tests : handler `httptest` (202, 409 FR/EN, 503, démo 403, joueur connecté / visiteur),
  ordonnanceur (jamais deux cycles, auto-sync désactivée, cadence recalée), composant vitest.
- Gate : `go test ./internal/api/... ./internal/scheduler/...`, `make generate-types` sans diff
  résiduel, `npx vitest run src/features/settings --maxWorkers=2`, `npx tsc --noEmit -p .`.

### Étape 4ter — état de l'historique (D-15)

- [x] 4t.1 Port + lecture `platform/duckdb` : nombre de matchs et date du plus ancien par joueur et
  titre (fuseau canonique, fragment partagé `sqlfrag`), état et marque-page par `historystate`. Fait :
  `port.HistoryProgressReader`, `platform/duckdb/historyrepo` (`ReadHistoryProgress` : base partagée par
  le fournisseur du titre, `SQLStartTimeCanonical`, Campagne retirée par `SQLExcludeCampaignVariants` comme
  dans les autres lectures d'historique ; clé de fin par `OpenReadForQuery`, base absente = incomplet).
  Le marque-page (D-13, étape 2.0) n'existe pas encore sur cette branche : l'état lu est la seule clé de
  fin ; l'intégration pourra ajouter la position du marque-page à `HistoryProgress` sans changer la route.
- [x] 4t.2 Dernière passe et prochaine passe lues de l'ordonnanceur (issue par joueur déjà tenue,
  heure du prochain cycle), derrière un port. Fait : `port.SyncActivity` (`LastPass`, `NextCycleAt`,
  `AutoSyncEnabled`). Écart au plan : `PlayerOutcomeDetail` ne porte pas les joueurs du pipeline V2 (le
  diag et la notification Discord les compteraient deux fois s'ils y entraient) ; la dernière passe vit
  donc dans une table à part (`scheduler/auto_sync_passes.go`), alimentée par l'issue du cycle V2 et par
  `syncPlayer`. Mémoire seule : après un redémarrage, l'écran dit « Aucune passe depuis le démarrage ».
  L'erreur d'une passe n'est pas servie en texte (le texte d'une erreur ne part pas au client) : l'écran
  dit l'issue (réussie, partielle, échec, différée) et le nombre de matchs ajoutés.
- [x] 4t.3 Route de lecture (handler sans logique), réponse localisée sans paire `_fr`/`_en`,
  `make generate-types`. Fait : `GET /api/v1/players/{player_slug}/sync/history` (200
  `HistorySyncStatus`), sous le groupe `/players/{player_slug}` (garde de propriété héritée), service
  `service.HistorySyncService`, libellés d'état et d'issue du catalogue `sync.history.*`
  (`internal/i18n/messages/sync.toml`).
- [x] 4t.4 Encadré dans l'onglet « Données & Médias », avec le bouton de l'étape 4bis : skills
  `frontend-patterns`, `color-tokens`, `foundations-usage` ; query key dans `lib/query/keys.ts` ;
  rafraîchissement périodique seulement pendant un rattrapage ; état vide et état d'erreur ; FR/EN.
  Fait (`SyncNowCard.tsx`, `HistoryState.tsx`, `syncNowQueries.ts`, `syncNowLogic.ts`,
  `syncNowText.ts`) ; clé `historySyncStatus(playerSlug, locale)` ; relecture toutes les 5 s pendant un
  cycle, toutes les 30 s pendant un rattrapage, jamais au repos, une fois à la fin de chaque cycle.
- [x] 4t.5 Tests : repo sur DuckDB `:memory:`, handler `httptest`, composant vitest. Fait (dépôt,
  service, handler, ordonnanceur, logique pure et composant : complet / rattrapage / échec de passe / cycle
  en cours / désactivé / EN / vide / erreur / chargement).
- Gate : celui de l'étape 4bis.

### Étape 5bis — mesure d'une passe (D-14)

- [ ] 5b.1 Sur une COPIE de la base locale (règle : neutraliser les chemins des réglages avant de lancer
  un serveur sur copie), clé `history_complete_at` retirée pour un joueur de plus de 1 000 matchs, une
  passe : durée, appels API par type, pic mémoire, octets écrits sous l'archive des films. Résultat
  écrit dans ce plan.

### Étape 5 — convergence de bout en bout (le test qui manquait)

- [x] 5.1 Fait. `cmd/server/sync_v2_convergence_env_integration_test.go` (banc) et `sync_v2_convergence_integration_test.go` (scénarios) : l orchestrateur est construit par `buildSyncV2Orchestrator` ; la file de persistance et son worker par `openPersistQueue` ; les bases par les migrations du démarrage (`runMigrations`, `RunPlayerMigrations`) ; les cycles sont joués PAR L ordonnanceur (`RunOnceTrigger`, boucle `Run` vivante, cadence lue sur `NextCycleAt`). Le faux client Halo est un faux `http.DefaultTransport` (le client de production `PooledHaloClient` -> brique `halo-api` fait ses vraies requêtes HTTP, qui répondent en mémoire : aucune connexion). 1 000 matchs, pages de 25, 2 arrivées de matchs neufs en haut : +200 par cycle, fin rangée au 6e cycle, appels `GetMatchHistory` par cycle [8 11 11 11 11 4], cycle suivant = 1 page, cadence 10 min puis 6 h. Variantes : match en 500 durable (demandé aux cycles 2 à 4, abandonné à la 3e passe, fin rangée au cycle 4), match 404 (ne coûte aucune passe). Écarts à la fabrique : `Settings`, `Friends`, `PostSyncRunner`, `PrestigeHook`, `ReplayEnqueue` nuls (médias, rejeux, notifications, défis ; hors du contrat de découverte, de persistance et de fin).
- [x] 5.2 Fait. `internal/games/halo_5/livesync/catchup_convergence_integration_test.go` : `Runner.RunDelta` avec capture de pages simulée (`historyWorld` du paquet) et vraie base joueur DuckDB temporaire pour l état (clé de fin, marque-page, compte d échecs) ; 1 000 matchs, matchs neufs en haut, fin rangée au cycle 6, pages lues par cycle [8 10 10 10 10 3], delta d une page ensuite ; variante page en échec durable abandonnée à la 3e passe. La persistance des matchs est simulée (écart assumé : la base partagée Halo 5 n a pas de rôle dans le contrat de fin).
- [x] 5.3 Fait. `internal/sync/engine_convergence_integration_test.go` : `SyncEngine.RunDelta` avec le `MaxMatches` de 25 du suivi en direct, vraies bases sous le fournisseur B-swap ; 1 000 matchs, 3 arrivées en haut, fin rangée puis delta d une page. Défaut trouvé et gardé sauté (`engine_convergence_defaut_integration_test.go`) : voir Découvertes.
- Gate : `go test -tags=integration -run Convergence ./internal/...`.

### Étape 6 — aide, version, notes (Haiku)

- [x] 6.1 Aide FR/EN (`features/help/i18n/{fr,en}.ts`) : le texte qui annonce un bouton de
  synchronisation manuelle dans les réglages décrit le comportement réel (synchronisation automatique,
  historique rattrapé de 200 matchs par passe, toutes les 10 minutes tant qu'il n'est pas complet ;
  parties personnalisées non synchronisées). Toutes les langues de `config/locales.toml`.
  Constat : le bouton existe (`settings/SyncTab.tsx`, « Synchroniser tous les joueurs ») ; ce qui
  était faux, c'était « une synchronisation complète peut être forcée depuis les Paramètres ». Réécrit
  dans les deux langues (entrée « Sync » / « Synchronisation », puce manuelle en FR).
- [x] 6.2 Version 8.2.2 : `apps/desktop/src-tauri/{Cargo.toml,Cargo.lock,tauri.conf.json}` (trois
  lignes faites). `apps/web/package.json` et `package-lock.json` NON modifiés : leur version est
  `0.0.0`, la 8.2.1 ne les a jamais touchés (`git show fee334a039` ne liste que les trois fichiers desktop).
- [x] 6.3 `docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md`, `docs/RELEASE_NOTES.md`,
  `docs/FR/RELEASE_NOTES.md` : section 8.2.2 (sans date, « Unreleased » / « Non publié » comme
  le fichier le fait ailleurs). Issue #213 citée dans les deux CHANGELOG (les sections précédentes en citent).
- Gate : `cd apps/web && npx vitest run src/features/help --maxWorkers=2 && npx tsc --noEmit -p .`.

### Étape 7 — revue adversariale (Fable)

- [ ] 7.1 Skills `adversarial-review` + `levelup-revue` sur `git diff origin/main...release/v8.2.2`,
  une ronde ; P0/P1 corrigés, P2 au backlog.

### Étape 8 — gates complets, journal, PR

- [ ] 8.1 `go test ./...`, `go test -tags=integration ./internal/sync/... ./internal/persist/...
  ./internal/games/...`, `make go-api-lint`, web : `npx vitest run` complet une fois, `make check-types`.
- [ ] 8.2 ADR 0027 : amendement court (EN) « history catch-up applies to every sync path ».
- [ ] 8.3 Entrée `.ai/thought_log.md`.
- [ ] 8.4 Poussée de `release/v8.2.2`, CI verte (`gh run list --branch release/v8.2.2 --workflow CI
  --limit 1`), PR vers `main` (squash par l'utilisateur).
- [ ] 8.5 Texte de réponse à #213 prêt (validé par l'utilisateur le 22/11), publié après la PR.

## Statuts, ordre, reprise

`[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification). Aucune case vide à la
clôture. Une étape est close quand son gate passe et que son commit est sur sa branche. Reprise : lire ce
fichier, `git log --oneline -10` dans chaque worktree, `ListAgents`.

## Découvertes

(Les correctifs hors périmètre sont interdits : les noter ici.)

- Étape 1 : `sync/match_recomputer.go` lit `sync_meta` par sa propre requête (clé `scoring_revision`),
  copie de `syncmeta.Get` ; à aligner plus tard, hors périmètre.
- Étape 1 : en mode file asynchrone, si la réouverture de la base joueur échoue après le drain, V1 ne
  peut pas ranger la fin (base `nil`) : `HistoryPending` reste vrai, la passe suivante rattrape.
- Étape 1 : une passe V1 qui sort en erreur avant la fin (chargement des connus, bail) rend
  `HistoryPending` faux (valeur par défaut) ; l'ordonnanceur (étape 4) lit aussi l'erreur du cycle.
- Étape 2 : `GetMatchStats` signale un 404/410 par un verdict TYPÉ (`infinite.IsNotFound`, `errors.Is(err, ErrFilmGone)`
  sur `*HTTPError`) ; ajouté `haloadapter.IsNotFound` et son ré-export `sync.IsNotFound` (même motif qu'`IsAuthError`).
  Les erreurs de récupération de V2 conservent la chaîne `%w`, donc le verdict traverse `RunFetchShared`.
- Étape 2 : le risque d'un match qui échoue toujours (parse, persistance, 500 permanent) est traité par D-16 :
  abandon à la 3e passe en échec, fin rangée ensuite. Les échecs se comptent par passe où le match était listé ;
  un orphelin de l'ensemble connu abandonné reste redemandé par la rotation des orphelins (`Set.Recover`) sans
  jamais bloquer.
- Étape 2 : l'entrée « introuvable » du dessus de liste (404 jamais persisté) est recomptée comme nouveau match à chaque
  passe ; elle décale la reprise du marque-page de ce nombre ; au-delà d'une page (25) la page de recouvrement ne
  contient plus le dernier match rangé et la lecture repart du début (journalisé, compté).
- Étape 2 : la page de recouvrement est lue deux fois (vérification puis lecture ordinaire) : un appel d'API de plus par
  passe, laissé tel quel.
- Étape 2 : joueur dont `LoadKnown` échoue (état de l'historique inconnu) : `HistoryPending` reste faux ; l'ordonnanceur
  (étape 4) lit l'erreur du cycle. Sorties anticipées du cycle : seules les issues par joueur déjà créées portent
  `HistoryPending`.
- Étape 2 : `ClearCursor` range une valeur vide (aucun DELETE sur `sync_meta`, règle anti-ART) ; le type s'appelle
  `Cursor` et la lecture `LoadCursor` (un type et une fonction ne peuvent pas porter le même nom dans un paquet).
- Étape 3 (suite, marque-page) : la liste Halo 5 se pagine par décalage (`start` de `GetPlayerMatches`).
  Le marque-page est branché (D-13) : `catchup.go` (`topDelta` : delta du dessus, N, décalage du
  marque-page ; reprise à marque-page + N moins une page) et `backfill_walk.go` (page de recouvrement,
  contrôle de l'identifiant par le `seen` de la page, reprise depuis 0 sinon). N compte aussi les matchs
  écartés (Warzone, campagne), jamais connus : la marge d'une page de recouvrement absorbe ce surcompte
  (au-delà de 24 entrées écartées de suite en haut de liste : reprise depuis 0, lente et sûre). Le marque-page
  se range après une page pleine propre qui a inséré des matchs, et une dernière fois à la sortie ; une
  page en erreur le laisse avant elle.
- Étape 3 (suite, abandon, D-16 adapté) : une page qui échoue 3 passes de suite (`maxCatchUpAttempts`) est
  abandonnée à la 3e (Warn : décalage, tentatives, dernière erreur ; compteur expvar
  `h5_history_abandoned_pages_total`), la passe continue et la fin peut se ranger. Le compte vit dans
  `historystate.Streak` (clé `history_failure_streak`, fonctions neutres `LoadStreak` / `SaveStreak` /
  `ClearStreak`, ajoutées à `historystate` dans `streak.go` ; `cursor.go` et le format du marque-page ne
  sont pas touchés ; le garde-rail archlint nomme la nouvelle clé). Le point d'échec est l'identifiant du
  marque-page. Compteur de reprises depuis 0 : `h5_history_cursor_missed_total`.
- Étape 3 (suite) : un match du dessus resté en échec de persistance pendant le delta n'est pas déclaré traité
  (le marque-page n'est pas décalé) : la passe suivante, ne retrouvant plus le marque-page sur sa page de
  recouvrement, repart de 0. Un tel match reste, comme avant, derrière un delta qui s'arrête au premier connu
  tant que le rattrapage n'est pas fini.
- Étape 3 (suite) : `backfillMaxPages` supprimé, la boucle s'arrête à `historystate.MaxDepth` (décalage) :
  une seule source pour la profondeur.
- Étape 3 : un match rangé sans timeline ou sans carnage (`EventsFailed`, `CarnageFailed`) est connu
  ensuite : le rebalayage ne le retente jamais (le commentaire de `captureMatchTimeline` dit le contraire).
  Hors périmètre.
- Étape 3 : le `PostScore` du runner Halo 5 récupère un carnage par match inséré
  (`PersistPerMatchRatings`) : une passe de rattrapage de 200 matchs coûte 200 appels de plus ; à mesurer à
  l'étape 5bis. Le tick du suivi en direct (watcher) passe aussi par `RunDelta` pour Halo 5 : il rattrape
  donc lui aussi, avec son `MaxMatches` serré comme budget.
- Étape 3 : une passe au budget atteint pile sur une page pleine ne voit la fin qu'à la passe suivante
  (une page vide, zéro match inséré) : comportement voulu, testé.
- Étape 5 : le moteur V1 (suivi en direct) verse l échec de récupération d un match dans les avertissements, pas dans les erreurs : la fin d historique se range avec un match perdu pour toujours (contraire à D-5), alors que D-8 le disait conforme. Test gardé sauté : `TestConvergence_V1_SuiviEnDirect_MatchEnEchecNeDoitPasRangerLaFin`. La correction demande l abandon (D-16) côté V1.
- Étape 5 : le moteur V1 reparcourt la liste depuis le haut à chaque passe de rattrapage (pas de marque-page, D-13 ne le couvre pas) : avec 25 matchs par passe, 1 000 matchs coûtent 41 passes et jusqu à 40 pages lues par passe. Hors périmètre.
- Étape 4 : `POST /admin/actions/auto-sync/run` (et la commande `sync` du menu de la zone de notification)
  appellent `RunOnceTrigger` ; depuis le garde « jamais deux cycles à la fois », un appel pendant un cycle
  rend un résultat vide (`busy`) que le job admin rapporte comme réussi avec zéro joueur : un 409 serait
  plus juste (non traité, hors périmètre).
- Étape 4bis : « Synchroniser maintenant » est ouvert à tout joueur connecté et lance un cycle pour TOUS les
  joueurs suivis, sans délai minimal entre deux cycles (le sync manuel d'un seul joueur, lui, a un
  `cooldown_retry_after_s`) : sur une instance hébergée, un compte peut enchaîner les cycles. À arbitrer
  par l'utilisateur (un délai minimal serait une ligne dans l'ordonnanceur).
- Étape 4bis : le gate de l'étape (api, scheduler) ne joue pas `archlint` ; le garde-rail
  `TestDeploymentSingleResolver_LecturesDesTroisReglages` (plafond des lectures de `.DemoMode` /
  `.AuthMode` dans `server_apiv1.go`) a refusé le second `RequireAuth(cfg.DemoMode, cfg.AuthMode)` du
  commit 4bis. Corrigé dans le commit 4ter : « Synchroniser maintenant » est monté dans le groupe
  `RequireAuth` existant, le groupe administrateur (`RequireAdmin`) y est imbriqué, sans nouvelle lecture.
- Étape 4ter : le compte « matchs enregistrés » exclut la Campagne des titres qui la masquent
  (`SQLExcludeCampaignVariants`), comme les autres lectures de l'historique du joueur ; il peut donc être
  inférieur au nombre de lignes de `match_participants`.
- Étape 4ter : `fix/822-e` n'a pas les étapes 2 et 3 : `PlayerOutcome.HistoryPending` n'est pas encore
  rempli par le pipeline V2 ni par le runner Halo 5 ; la cadence de rattrapage ne s'active, tant que ce
  n'est pas intégré, que par un joueur en échec (clé de fin absente) ou par le moteur V1 (filet de boot).
