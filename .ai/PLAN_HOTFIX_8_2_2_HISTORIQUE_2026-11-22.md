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
  (`Complete(ctx, db) (bool, error)` depuis D-25 : une lecture en échec rend l'erreur, jamais « complet » ;
  l'appelant fait un delta ordinaire, n'écrit rien de l'état, garde le joueur en attente et journalise), son écriture
  (`MarkComplete`), le budget par passe et la borne de sûreté. V2 et Halo 5 l'appellent (V1 seulement pour la borne de profondeur du RunFull : D-18) ; plus
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
- **D-8** (remplacée par D-18, 22/11) Le tick du suivi en direct (watcher, 25 matchs) passait par V1, supposé
  suivre déjà la règle : faux, voir D-18.
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
- **D-17** (utilisateur, 22/11) Un match synchronisé l'est en totalité, film traité compris. Le budget
  de temps de 5 minutes par cycle de chaque étape film (`killcollector.PostSyncBudget`,
  `replayartifacts.BudgetParCycle`) devient la seule limite effective ; les plafonds en nombre
  (`DefaultPostSyncPerCycle = 8`, `maxPerCycle = 5`) montent à une borne de sécurité large, pour que les
  gains du décodeur passent sans retoucher le code. Parallélisme de décodage inchangé, jeton d'écriture
  unique inchangé. L'état de l'historique réutilise l'indication existante des films en cours de
  traitement. Valeurs fixées après la mesure 5bis (temps par film, pic mémoire).
- **D-19** (utilisateur, 22/11, après la mesure 5bis) Le délai d'attente de la persistance du cycle V2
  (60 s fixes, `persist_v1bridge.go`) devient proportionnel au lot ; mesuré : ~1 s par match, 200
  matchs ≈ 3,7 min, d'où l'échec des passes. Le nombre de matchs persistés rapporté est juste même en
  cas d'échec (relecture). Budget de 200 par passe conservé.
  - [x] D-19.1 Délai = 60 s + 4 s par match soumis, plafond 20 min (`v2/persist_drain.go`, 200 matchs = 14 min ;
    mesure 1,1 s par match). File lente > ancien délai : cycle réussi (`TestPersistCycle_FileLente_DelaiProportionnelAuLot`).
  - [x] D-19.2 Expiration malgré tout : erreur typée `DrainTimeoutError`, le cycle relit l'ensemble connu
    (`v2/persist_pending.go`), compte et post-sync sur les matchs prouvés rangés, matchs en file ni échecs D-16 ni
    avancée du marque-page (`history_settle.go`), compteur `sync_v2_persist_drain_timeout_total`
    (`cycle_history_drain_test.go` : post-sync des rangés, aucun faux échec sur 5 passes, vrai 500 compté, tout rangé = réglage normal).
- **D-20** (utilisateur, 22/11) La cadence de 10 min tient tant que l'historique OU l'arriéré de films
  (source des frags, artefacts de rejeu) n'est pas vide ; l'état de l'historique réutilise l'indication
  existante des films en cours de traitement.
- **D-21** (utilisateur, 22/11) Ancre du marque-page : il retient aussi le match en tête de liste à la
  passe précédente ; le décalage des nouveaux matchs = position de cette ancienne tête dans la liste
  (exact, quels que soient les introuvables, Warzone ou Campagne). Reprise depuis 0 seulement si
  l'ancienne tête n'est pas retrouvée dans les premières pages. Infinite et Halo 5.
  - [x] D-21.1 API `historystate` (commit 0e04891af9) : `Cursor.HeadID` (JSON `head_match_id`, facultatif),
    `Cursor.HasAnchor`, `Cursor.ArrivedSince(top)`, `Cursor.ResumeOffset(arrived)`, `AnchorMaxPages` (8), `AnchorScanLimit(pageSize)`.
  - [x] D-21.2 Découverte V2 (`v2/fetchers_history_anchor.go`) : N = position de l'ancienne tête, reprise à marque-page + N contrôlée par
    l'identifiant rangé en première entrée de la page lue (plus de page de recouvrement) ; ancre perdue ou contrôle raté = reprise depuis 0,
    journalisée et comptée (`sync_v2_history_cursor_fallback_total`) ; marque-page sans ancre = comportement précédent.
    Tests : 30 introuvables en haut sur 3 passes (exact, zéro reprise depuis 0), N neufs, ancre perdue, contrôle raté, budget avant l'ancre, sans ancre.
  - [x] D-21.3 Test de convergence `cmd/server` mis à jour : appels `GetMatchHistory` par cycle [8 11 11 11 11 4] (56) -> [8 9 10 9 10 2] (48).
  - [ ] D-21.4 Halo 5 : à brancher par la piste Halo 5 sur l'API D-21.1 (`catchup.go` range `Cursor{Offset, LastMatchID}` sans `HeadID`).
- **D-22** (utilisateur, 22/11) Site hébergé : ancien comportement des films (8 frags et 5 rejeux par
  cycle, cadence non liée à l'arriéré de films), pour ménager le serveur de prod (2 vCPU, 2 Go). Le
  comportement D-17/D-20 ne vaut que sur le bureau, décidé par une capacité du profil de déploiement
  (ADR 0044), jamais par le profil lui-même. Le rattrapage des matchs (200 par passe, 10 min) vaut partout.
  - [x] D-22.1 Capacité `FilmCatchUp` de `config.DeploymentCapabilities` (vraie en desktop seulement ; fausse en hosted, dev et demo ;
    `TestDeployment_CapacitesParProfil`). Sans elle : 8 films de frags (`killcollector.FragsParCycleHeberge`), 5 rejeux
    (`replayartifacts.RejeuxParCycleHeberge`), retard lu sur 64 matchs (`BacklogHorizonHeberge`) ; avec elle : garde-fou 200, horizon 2000.
    Branchement : `replayartifacts.NewHook` lit la capacité (`Hook.PerCycle`, `Hook.Horizon`, copiés dans `Deps`), `SyncEngine.WithReplayArtifacts`
    la porte à la source des frags (`PostSyncHook.WithFilmCatchUp` ; un moteur non câblé part des bornes hébergées).
  - [x] D-22.2 Cadence et état à l'écran : `filmbacklog.ReaderFor(cfg.Capabilities.FilmCatchUp)` dans `cmd/server/boot_tasks.go` ; hors capacité, aucun
    lecteur d'arriéré, donc la cadence ne suit que l'historique et l'écran n'affiche pas « Analyse en cours » pour des films (décision prise avec D-22).
    Tests par profil : `TestEngine_BornesDesFilmsSuiventLeProfil` (hosted 8/5/64, dev idem, desktop 200/200/2000),
    `TestCadence_ArriereDeFilmsSuitLaCapaciteDuProfil` (hosted et dev : intervalle réglé ; desktop : 10 min), tests d'intégration des bornes hébergées.
- **D-23** (utilisateur, 22/11) Le delta des nouveaux matchs ne s'arrête plus au premier match connu mais
  après une page entière (25) de matchs connus d'affilée : un match rangé d'abord par un coéquipier
  (connu par ses participants) ne cache plus les matchs inconnus juste en dessous. Infinite (V2) et Halo 5.
  - [x] D-23.1 Constante unique `domain.DeltaKnownRun = 25` (commentaire de contrat : pourquoi une page de connus d'affilée),
    lue par tous les delta : V2 historique complet (`v2/fetchers.go` `listDelta`), V2 rattrapage sans ancre (`scanNew`, N =
    `topSpan` : toutes entrées jusqu'à la dernière inconnue, abandonnés exclus ; avec ancre `scanTop` ne s'arrête déjà pas
    au premier connu, il lit jusqu'à l'ancre : rien à changer, test ajouté), Halo 5 delta et delta du dessus
    (`halo_5/capture.go` `capturePage` : `CaptureStats.KnownRun` / `TopSpan`, entrées écartées neutres ; `catchup.go`
    `topDelta` : N sans ancre = `TopSpan`), moteur V1 (`engine.go` `paginateAndPersistHistory`, même règle, `allKnown` retiré).
    Bornes de pages inchangées (20 pages V2 en mode complet, `MaxDepth`).
  - [x] D-23.2 Tests : scénario A/B (V2 complet, V2 rattrapage avec et sans ancre, Halo 5 capture et RunDelta avec/sans ancre
    et complet, V1 intégration), 24 connus puis un inconnu, 25 connus d'affilée, entrée écartée neutre (Halo 5), appels d'API.
    Appels de pages d'historique, avant -> après : historique complet sans nouveau match 1 -> 1 ; avec un nouveau match 1 -> 2
    (V2, Halo 5, V1) ; V2 convergence `[8 9 10 9 10 2]` (48) inchangée, delta d'un match neuf après la fin 1 -> 2 pages ; Halo 5 convergence `[8 10 10 10 10 3]` (51)
    -> `[8 11 10 11 10 3]` (53) ; Halo 5 40 Warzone en tête `[10 11 10 10 10]` (51) -> `[10 12 12 12 12]` (58, les Warzone
    n'arrêtent ni ne remettent à zéro : 40 + 25 entrées = 3 pages au lieu de 2).
- **D-24** (pilote, 22/11) Les matchs introuvables et abandonnés ne sont plus demandés, ni comptés insérés. Un
  ensemble persistant par joueur des matchs à ne plus demander : introuvables (404/410) et abandonnés (D-16), rangé dans
  `sync_meta` sous la clé `history_skip_set` (paquet `historystate`, `skipset.go`) : JSON `{"skipped":[{"id","why"}],
  "saved_at"}`, `why` = `not_found` | `abandoned`, borné à 2 000 entrées (les plus anciennes sortent), valeur vide jamais
  de DELETE. Neutre pour le delta et la découverte (ne compte pas dans la suite de 25 connus et ne la remet pas à zéro,
  comme les Warzone de Halo 5), plus demandé, jamais compté inséré. `MarkComplete` ne l'efface pas. Le compte « inséré »
  d'un cycle = matchs PROUVÉS rangés (relecture de l'ensemble connu, mécanisme de D-5/D-19). Écritures sous le bail
  d'écriture déjà utilisé par V2 (`WithHistoryWriter`). Halo 5 : aucun match individuel n'y est introuvable (la liste
  porte les statistiques du match, un carnage ou une timeline en 404 donne un match collecté sans roster, jamais un
  match perdu) ; ses entrées écartées (Warzone, campagne) sont déjà neutres (D-23) et ses pages abandonnées relèvent de
  `historystate.Streak` (rattrapage seulement) : rien à ajouter.
  - [x] D-24.1 API `historystate` (`skipset.go` : `SkipSetKey`, `MaxSkipSet` 2000, `LoadSkipSet`, `AddSkips`) et garde-rail
    archlint des clés étendu (`history_key_single_source_test.go`) ; tests (aller-retour, doublons, borne, `MarkComplete`
    n'efface pas, valeur illisible, lecture en échec).
  - [x] D-24.2 Découverte V2 : `HistoryState.Skip`, lu par la capacité facultative `SkipSetLoader` (`known_loader_skip.go`,
    consultée par `RunDiscovery`) ; neutralité dans `listDelta`, `scanNew`, `scanFrom`, `scanTop` et `appendOrphans`.
  - [x] D-24.3 Cycle V2 : rangement après la persistance (`history_skip.go`, joueur complet comme en rattrapage, sous
    `WithHistoryWriter`, AVANT la fin d'historique : écriture en échec = fin non rangée) ; compte « inséré » sur preuve
    (`cycle_stored.go`, une relecture par joueur partagée avec le règlement du rattrapage).
  - [x] D-24.4 Tests : `cycle_history_skip_test.go` (1 000 matchs dont 30 introuvables : 970 annoncés, après la fin une
    page et zéro demande ; 30 en tête : trois pages, zéro demande ; delta pur ; abandonné non redemandé après la fin ;
    non rangé non compté) ; `cmd/server` : `TestDifficile_MatchIntrouvable_NEstPasCompteInsere` dé-sauté, comptes d'appels
    adaptés (voir 5t.7).
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
  ordinaire (précisé par D-24 : l'abandon est aussi rangé dans l'ensemble des matchs à ne plus demander, qu'il ne quitte
  qu'à la borne de 2 000).
- **D-18** (pilote, 22/11) UN SEUL moteur fait le rattrapage de l'historique : le cycle de l'ordonnanceur
  (pipeline V2 pour Halo Infinite, runner Halo 5). Le moteur V1 (`internal/sync/engine.go`), utilisé par
  le suivi en direct (`MaxMatches` 25), la synchronisation manuelle d'un joueur, la synchronisation initiale
  et le filet `syncPlayer` de l'ordonnanceur (orchestrateur V2 non câblé), redevient un delta pur : il ne
  lit plus ni n'écrit l'état de l'historique (`historystate`), ne bascule plus en mode full sur historique
  incomplet, ne range plus la fin (`MarkComplete`) et ne remplit plus `SyncResult.HistoryPending` (V2 et Halo 5
  le remplissent). `HistoryExhausted`, sans plus aucun lecteur de production, est supprimé du résultat. Un
  `RunFull` explicite garde son comportement de full (parcours jusqu'au plafond ou à la page vide, connus
  sautés) mais ne range plus la fin. Cause : le V1 rangeait la fin avec un match en échec (les échecs de
  récupération sont des avertissements, pas des erreurs, contraire à D-5) et rebalayait la liste depuis le
  haut à chaque passe (862 appels de liste pour 1 000 matchs par pas de 25). La première synchronisation
  (aucun match connu) garde son résultat : le delta récupère jusqu'à `MaxMatches` (200 ou
  `initial_max_matches`), comme le full. L'aide n'est pas concernée.
- **D-25** (pilote, 22/11, après 5t.8) Un cycle lancé juste après un redémarrage avec un journal d'écriture en attente
  n'est plus bloqué, et une lecture en échec de l'état de l'historique n'est plus « complet ».
  - [x] D-25.1 Cause racine : la découverte (`cmd/server/sync_v2_wiring.go`, `playerDBOpenerRO`) lisait la base joueur par
    `OpenReadForQuery`, qui, sans handle en cache, l'ouvre en lecture seule (clé `ro:`, `access_mode=read_only`) ; le worker
    de la file l'ouvre par `OpenReadWrite` (clé `rw:`, DSN nu) : refus « different configuration », hors de la liste
    transitoire du worker, lot laissé au journal jusqu'à la reprise périodique. Dans l'autre sens, un handle RW emprunté
    SANS référence se fermait sous la lecture quand le worker rendait le sien (« sql: database is closed »). Correction à la
    source : `duckdb.OpenPlayerReadForQuery` (`platform/duckdb/player_read_open.go`) prend une référence sur le handle en
    cache (RW, sinon RO) et, faute de handle, ouvre dans la configuration de l'écrivain ; branché sur la découverte et sur
    `historyrepo` (ordonnanceur et écran, qui lisent pendant que la file écrit). La seconde ouverture par cycle de D-24
    (`known_loader_skip.go`, une fenêtre de lecture seule de plus face au worker avant ce correctif) est repliée dans
    `LoadKnown` : une seule ouverture par joueur et par cycle ; un état illisible ne range pas non plus l'ensemble (`settleHistory`). Bases partagées exclues (`IsPlayerDBPath`) :
    leur bascule B-swap doit pouvoir fermer le handle RW. Le worker garde sa liste transitoire inchangée.
  - [x] D-25.2 Attente sans progrès : `BatchQueue.DrainWhileProgressing` (`persist/queue_drain_stall.go`, compteur
    d'acquittements de la file) rend `ErrDrainStalled` quand aucun lot n'est acquitté pendant `persistDrainStall` (60 s,
    `v2/persist_drain.go`) ; le cycle rend `DrainTimeoutError{Stalled: true}` (journalisé), traité comme D-19.2.
  - [x] D-25.3 `historystate.Complete(ctx, db) (bool, error)` (`ErrNoPlayerDB` sur base absente) ; V2 : état `Unreadable`
    (delta, aucune écriture de l'état, `HistoryPending` vrai) ; Halo 5 : `playerHistory.Complete` remonte l'erreur (RunDelta :
    delta en attente) ; `historyrepo` rend l'erreur (écran : « inconnu ») ; ordonnanceur : joueur compté en rattrapage.
  - [x] D-25.4 `TestDifficile_Interruption_CycleApresRedemarrageSansBlocage` dé-sauté et complété (journal vide, aucun
    doublon, fin rangée) : cycle après redémarrage 14 min 28 s -> 35 à 44 s (test entier 151 à 168 s, trois passes dont une après la fusion de D-24), zéro « different
    configuration », zéro « database is closed ». Tests : attente sans progrès (persist, v2), `Complete` sur erreur
    (historystate, V2 cycle et chargeur, Halo 5, historyrepo, ordonnanceur), partage du handle (platform/duckdb).
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
- [x] 3.5 Ancre du marque-page Halo 5 (D-21) : `HeadID` rangé à chaque marque-page (tête de la liste lue à la
  passe) ; N = `ArrivedSince` sur au plus `AnchorScanLimit(25)` = 200 entrées, toutes entrées comptées
  (écartées comprises), reprise à `ResumeOffset(N)` moins une page, `LastMatchID` gardé comme garde-fou ;
  marque-page sans ancre = règle « jusqu'au premier connu » ; ancre perdue = reprise depuis 0, journalisée et
  comptée (`h5_history_cursor_missed_total`). Code : `livesync/catchup_anchor.go`, `catchup.go` (`topDelta`),
  `backfill_walk.go` (`advance` range l'ancre), `halo_5/capture.go` (`CaptureStats.HeadID` / `TopIDs`,
  `CaptureOptions.TopLimit`, `ListMatchIDs`). Tests : `catchup_anchor_test.go` (30 Warzone en tête à deux
  passes, neufs mêlés Arena / Warzone, ancre perdue, connu au-dessus de l'ancre, lecture de liste en échec,
  marque-page sans ancre), `capture_test.go`, convergence
  `TestConvergence_Halo5_RunDelta_WarzoneEnTeteNeFaitJamaisRepartirDe0` : pages lues par cycle `[10 22 10 10 9]`
  (total 61, une reprise depuis 0 au cycle 2) avant, `[10 11 10 10 10]` (total 51, aucune reprise depuis 0)
  après ; le scénario tout Arena est inchangé (`[8 10 10 10 10 3]`).
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

### Étape 4f — suite de l'étape 4ter (décisions du pilote du 22/11)

- [x] 4f.1 Délai minimal : « Synchroniser maintenant » refuse un cycle s'il s'en est terminé un (toute
  origine) il y a moins de `manualCycleMinGap` (2 min, `scheduler/auto_sync_manual.go`) : 429
  `sync_cycle_cooldown`, `Retry-After` en secondes, message du catalogue `sync.now.cooldown` (FR/EN). Le tick
  et la commande du menu de la zone de notification ne sont pas concernés. `GET /sync/cycle` sert
  `last_cycle_at` et `manual_available_at` (absente si permis tout de suite). Écran : bouton désactivé
  « Disponible dans 1 min 40 » / « Available in 1 min 40 » avec compte à rebours, retour seul à l'action ;
  ligne visible sous le bouton, sans couleur d'alerte ; « en cours » : « Synchronisation en cours… » /
  « Sync in progress… » ; un 429 malgré tout = la même ligne, pas de toast.
- [x] 4f.2 Dernière passe persistée : la table de `auto_sync_passes.go` est rangée dans l'instantané
  post-sync (`persistence.go`, champ `last_passes`, version inchangée) et relue par `RehydrateFromDisk` ;
  instantané ancien sans le champ = table vide ; une entrée plus récente déjà en mémoire est gardée.
- [x] 4f.3 Vérifié sur la release intégrée : `PlayerOutcome.HistoryPending` est rempli par
  `sync/v2/history.go` (`historyTracker.apply`, clé `PlayerSlug`, y compris sur les sorties anticipées) et
  `SyncResult.HistoryPending` par `games/halo_5/livesync/runner.go` ; l'ordonnanceur les lit
  (`pendingHistoryAfterV2`, `syncPlayerWithHistory`). Un test par chemin, par la boucle de `Run` avec ticker
  injecté (`auto_sync_cadence_paths_test.go`) : en attente = 10 min, complet = intervalle réglé.
- [x] 4f.4 État de l'historique : `state` = `catching_up` quand la clé de fin manque, marque-page présent
  ou non (test `TestHistoryComplete_AvecMarquePage`) ; rien d'autre n'est servi, le marque-page non plus
  (règle 18).
- [x] 4f.5 `disk-hygiene.ps1 -Lot 822e -Apply` lancé (voir le rapport de l'étape).

### Étape 4g — films : le temps seul borne la passe, la cadence suit l'arriéré (D-17, D-20)

- [x] 4g.1 (D-17) Les deux étapes film ne sont plus bornées par un nombre mais par leur budget de temps
  (5 min chacune, inchangé). `DefaultPostSyncPerCycle` (8) et `maxPerCycle` (5) deviennent une même borne de
  sécurité `killcollector.GardeFouFilmsParCycle = 200` (`maxPerCycle` la reprend, une seule valeur). Parallélisme
  de décodage, jeton d'écriture unique, limite par film : inchangés. Adaptations vérifiées sur pièces :
  `replayartifacts.BacklogHorizon` 64 -> 2000 (le filtre « artefact présent » s'applique après la lecture du
  registre : à 64 la fenêtre n'aurait jamais dépassé les 64 matchs les plus récents, et le garde-fou n'aurait
  jamais mordu) ; `PostSyncBacklogHorizon` (64, taille d'une page) et `PostSyncBacklogPagesMax` (8 pages = 512
  candidats, pour 200 retenus) inchangés ; dérivations : le garde-fou vaut aussi 200, mais `Deriver` lit tous les
  documents de son lot en mémoire, donc `deriverParLots` les passe par lots de 5 et s'arrête entre deux lots sur
  le budget de temps. Tests : arrêt sur le budget de temps (killcollector et replayartifacts, base réelle ;
  `enLots`), tous les matchs traités sous un budget large (20 > 8 et > 5).
- [x] 4g.2 (D-20) La cadence de 10 min tient tant que l'historique OU l'arriéré de films n'est pas vide. Chemin de
  l'indicateur : chaque étape film publie à la fin de sa passe `filmbacklog.Publier(titre, étape, restant)`
  (`internal/sync/filmbacklog`, par titre : l'arriéré est celui du registre partagé du titre) ; l'ordonnanceur le
  lit par le port `port.FilmBacklogReader` (`WithFilmBacklogReader`, câblé dans `cmd/server/boot_tasks.go`, garde
  `boot_film_backlog_test.go`) via `filmBacklogOf` en fin de cycle, pour le cycle V2 ET pour le filet `syncPlayer`.
  L'arriéré est celui d'APRÈS la passe (non atteint faute de temps, reportés) ; il ne compte pas les matchs essayés
  et non résolus (échec, film mis de côté) ni les matchs sans carte, pour que la cadence courte ne tourne pas sans
  fin sur des films qui ne progressent pas. Halo 5 n'a aucune étape film : son titre ne publie jamais d'arriéré
  (rien à remonter). Journal `auto_sync: cycle suivant planifié` : `films_en_retard` (+ `films_frags`,
  `films_rejeux`). Tests : historique complet + arriéré -> 10 min, arriéré vide -> intervalle réglé, V2 et filet.
- [x] 4g.3 (D-15/D-20) État à l'écran : `GET /players/{slug}/sync/history` sert `films_pending` par titre
  (`port.SyncActivity.FilmsPending`) ; l'encadré réutilise la pastille des tuiles de match (mêmes textes du
  manifeste commun, « Analyse en cours » pendant un cycle, « Analyse en attente » sinon) au lieu de « Historique
  complet » tant que des films attendent ; relecture lente tant que c'est vrai. FR et EN (textes existants).
- Gate : `go test ./internal/sync/killcollector/... ./internal/sync/replayartifacts/... ./internal/sync/filmbacklog/...
  ./internal/scheduler/... ./internal/service/... ./cmd/server/...` (+ `-tags=integration`), `npx vitest run
  src/features/settings`, `npx tsc --noEmit -p .`.

### Étape 5bis — mesure d'une passe (D-14)

- [ ] 5b.1 Sur une COPIE de la base locale (règle : neutraliser les chemins des réglages avant de lancer
  un serveur sur copie), clé `history_complete_at` retirée pour un joueur de plus de 1 000 matchs, une
  passe : durée, appels API par type, pic mémoire, octets écrits sous l'archive des films. Résultat
  écrit dans ce plan.

### Étape 5 — convergence de bout en bout (le test qui manquait)

- [x] 5.1 Fait. `cmd/server/sync_v2_convergence_env_integration_test.go` (banc) et `sync_v2_convergence_integration_test.go` (scénarios) : l orchestrateur est construit par `buildSyncV2Orchestrator` ; la file de persistance et son worker par `openPersistQueue` ; les bases par les migrations du démarrage (`runMigrations`, `RunPlayerMigrations`) ; les cycles sont joués PAR L ordonnanceur (`RunOnceTrigger`, boucle `Run` vivante, cadence lue sur `NextCycleAt`). Le faux client Halo est un faux `http.DefaultTransport` (le client de production `PooledHaloClient` -> brique `halo-api` fait ses vraies requêtes HTTP, qui répondent en mémoire : aucune connexion). 1 000 matchs, pages de 25, 2 arrivées de matchs neufs en haut : +200 par cycle, fin rangée au 6e cycle, appels `GetMatchHistory` par cycle [8 11 11 11 11 4], cycle suivant = 1 page, cadence 10 min puis 6 h. Variantes : match en 500 durable (demandé aux cycles 2 à 4, abandonné à la 3e passe, fin rangée au cycle 4), match 404 (ne coûte aucune passe). Écarts à la fabrique : `Settings`, `Friends`, `PostSyncRunner`, `PrestigeHook`, `ReplayEnqueue` nuls (médias, rejeux, notifications, défis ; hors du contrat de découverte, de persistance et de fin).
- [x] 5.2 Fait. `internal/games/halo_5/livesync/catchup_convergence_integration_test.go` : `Runner.RunDelta` avec capture de pages simulée (`historyWorld` du paquet) et vraie base joueur DuckDB temporaire pour l état (clé de fin, marque-page, compte d échecs) ; 1 000 matchs, matchs neufs en haut, fin rangée au cycle 6, pages lues par cycle [8 10 10 10 10 3], delta d une page ensuite ; variante page en échec durable abandonnée à la 3e passe. La persistance des matchs est simulée (écart assumé : la base partagée Halo 5 n a pas de rôle dans le contrat de fin).
- [x] 5.3 Fait, puis révisé par D-18 (22/11). Le test de convergence V1 en 41 passes et le test du défaut (gardé sauté) sont remplacés, dans `internal/sync/engine_convergence_integration_test.go`, par le contrat de delta pur : sur un historique incomplet avec un match en échec durable, V1 ne touche à aucune clé de `historystate` et s'arrête au premier connu (une page lue par passe, matchs neufs du dessus pris) ; la première synchronisation (aucun connu) récupère jusqu'au plafond (200, 50) ; ni `RunDelta` ni `RunFull` ne rangent la fin. Supprimés avec le comportement : `engine_history_complete_test.go`, `engine_history_pending_integration_test.go`, `engine_convergence_defaut_integration_test.go`.
- Gate : `go test -tags=integration -run Convergence ./internal/...`.

### Étape 5ter — cas difficiles (tests seuls, aucune modification du code de production)

Banc étendu dans de nouveaux fichiers de `cmd/server` (`sync_v2_convergence_difficile_*_integration_test.go`,
`sync_v2_convergence_interruption_integration_test.go`) : plusieurs joueurs et une liste par joueur, démarrages
répétés sur les mêmes bases (orchestrateur, ordonnanceur, file de persistance et worker neufs), arrêt par la séquence
de production (`runShutdown`) ou brutal (journal d'écriture laissé sur disque), porte du worker de persistance.
Réutilisés du banc de base : faux client Halo (`fakeHalo`), `fakeTokenPool`, `recordingOrchestrator`, `waitCadence`.

- [x] 5t.1 Arrêt pendant la récupération (contexte annulé à la 100e demande de stats d'une passe, arrêt par `runShutdown`,
  démarrage neuf) : 1 000 matchs, aucun doublon, aucun trou, fin rangée, marque-page effacé, journal vide
  (`TestDifficile_Interruption_ArretPendantRecuperation`).
- [x] 5t.2 Arrêt pendant la persistance, trois fins : arrêt ordonné avec peu de reste, arrêt ordonné avec beaucoup de
  reste (le vidage de 5 s n'y suffit pas, le journal garde le reste), coupure brutale (dont un match écrit et non
  acquitté) (`TestDifficile_Interruption_ArretPendantPersistance`).
- [x] 5t.3 Arrêt entre deux passes : reprise au marque-page, page du haut puis lecture à la position rangée, aucun repli
  « depuis 0 » (`TestDifficile_Interruption_ArretEntreDeuxPasses`).
- [x] 5t.4 Historique complet : une page, ni marque-page ni clé de fin touchés, cadence = intervalle réglé
  (`TestDifficile_HistoriqueDejaComplet_CasNormalInchange`).
- [x] 5t.5 Deux joueurs dans le même cycle, l'un complet, l'autre en rattrapage (`TestDifficile_DeuxJoueurs_UnCompletUnEnRattrapage`).
- [x] 5t.6 Matchs neufs du dessus mêlés à 30 introuvables : rien de perdu, reprise exacte, zéro repli
  (`TestDifficile_MatchsNeufsEtIntrouvables_RepriseExacte`).
- [x] 5t.7 DÉFAUT corrigé par D-24 (test dé-sauté, `TestDifficile_MatchIntrouvable_NEstPasCompteInsere`, vert) ; constat d'origine :
  `populateOutcome` (`internal/sync/v2/cycle.go`) compte comme `MatchesInserted` tout match inconnu listé pour le joueur,
  qu'il ait été récupéré et rangé ou non. Un match introuvable (404) en haut de liste est donc compté inséré à CHAQUE
  cycle, pour toujours (et redemandé à chaque cycle) ; la notification de fin de cycle l'annonce comme nouveau match. Depuis D-23, les
  introuvables des 25 premières entrées sont recomptés et redemandés à chaque cycle (21 constatés, trois pages lues, après un
  rattrapage de 1 000 matchs dont 30 introuvables).
  Présent depuis D6.1, donc dans la 8.2.1. Corrigé (D-24) : le compte « inséré » se lit sur la relecture de l'ensemble connu ;
  les introuvables et les abandonnés entrent dans l'ensemble des matchs à ne plus demander, neutres pour le delta : après la fin du
  rattrapage, 20 introuvables dans les 40 plus récentes entrées = 2 pages et 0 demande de stats par cycle (avant : 3 pages, 21 demandes).
- [x] 5t.8 (corrigé par D-25, test dé-sauté) DÉFAUT constaté (`TestDifficile_Interruption_CycleApresRedemarrageSansBlocage`) : après un
  redémarrage avec un journal d'écriture en attente (coupure brutale en plein rattrapage), un cycle lancé aussitôt (a) reste
  bloqué 14 minutes 20 (le rejeu du journal échoue sur « same database file with a different configuration » parce que la
  découverte du cycle tient la base joueur en lecture ; erreur hors de la liste des erreurs transitoires du worker ; le
  vidage du cycle attend le journal entier : 60 s + 4 s par match soumis) ; (b) (constaté une fois sur deux exécutions) après la reprise périodique du journal, le
  cycle suivant a lu l'état de l'historique sur une poignée fermée (« sql: database is closed »), `historystate.Complete` l'a
  traduit en « complet » : 0 match inconnu listé, `HistoryPending` faux, cadence de 10 minutes perdue. Les deux sont le
  rejeu du journal contre la découverte, dans le même processus. Non corrigé ici (hors périmètre).
- Gate : `go test -tags=integration -run 'Difficile|Interruption' -v -timeout 40m ./cmd/server/`, `go vet -tags=integration`,
  golangci-lint `--new-from-merge-base=origin/main --build-tags=integration` sur `./cmd/server/...`.

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
  une ronde ; P0/P1 corrigés, P2 au backlog. Agent Fable (demande de l'utilisateur, 22/11), sur
  l'implémentation ET sur le concept : le rattrapage par cycle, le marque-page ancré, l'abandon,
  la cadence liée à l'arriéré de films, les alternatives (carte des identifiants, autre découpage).
  Il reçoit le code et les mesures, pas les conclusions du pilote.

### Étape 8 — gates complets, journal, PR

- [ ] 8.1 (révisé par l'utilisateur le 22/11 : contrôles sur le périmètre modifié seulement, la CI joue
  la suite complète) Une seule passe locale sur la branche intégrée : `go build ./...` ; `go test` et
  `go test -tags=integration` sur les seuls paquets touchés par `git diff --name-only origin/main...` ;
  `./internal/archlint/...` (gardes globales, une fois) ; golangci-lint `--new-from-merge-base` ; web :
  `npx vitest run` sur les dossiers touchés, `tsc`, eslint des fichiers touchés. Puis CI verte.
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
- D-19 : un match qui échoue au parsage dans `PersistCycle` n'est jamais soumis ; après une attente expirée il n'est pas distingué des matchs en file que par la liste `DrainTimeoutError.Submitted` (traité : seuls les soumis sont « en file »). Un match rangé en arrière-plan après l'expiration n'a pas les soins ciblés du post-sync de ce cycle (hors périmètre).
- D-21 : un marque-page sans ancre (rangé avant ce correctif) retombe sur la règle précédente une passe, puis reçoit son ancre.
- Test `TestSoak_V2_CycleStability` : un échec isolé lancé en parallèle d'un autre paquet, vert seul (3/3) et au rejeu complet (2/2) ; non relié.

(Les correctifs hors périmètre sont interdits : les noter ici.)

- Étape 4g : le filet `engine.run` (`runConditionalPostSync`) ne lance le post-sync complet, films compris, que si
  des matchs sont insérés ou si `hasConvergenceBacklog` (enrichissement, PSA, événements) est non vide : un arriéré
  de films seul ne le déclenche pas. Sans effet sur le cycle V2 (`RunPostSyncForV2Shared` lance toujours le pipeline
  complet) ; à traiter si le filet doit un jour résorber les films.
- Étape 4g : `BacklogHorizon` à 2000 fait lire jusqu'à 2000 identifiants et autant de `os.Stat` par cycle et par
  joueur pour le rattrapage des rejeux et des dérivés ; à mesurer sur la base réelle (étape 5bis).
  Sur le site hébergé il est ramené à 64 par la capacité `FilmCatchUp` (D-22) : le coût de 2000 lectures par cycle et par joueur ne s'y applique pas.
- D-22 : `killcollector/postsync.go` dépassait 500 lignes (506) depuis l'étape 4g et faisait échouer `TestTailleDesFichiersDuFilmNeCroitPas` (le gate de 4g ne lançait pas archlint) ; les bornes sont extraites dans `postsync_bornes.go` (474 lignes).

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
- Étape 3 (D-21) : le delta s'arrête au premier match connu ; un match connu arrivé en tête depuis la passe
  précédente (rangé par un coéquipier dans la base partagée) arrête le delta avant des matchs neufs inconnus
  plus bas, qui ne sont alors jamais pris par le delta. Comportement d'avant l'ancre, non traité ici. L'ancre
  compte exactement ces entrées dans N, et la tête de liste est relue (un appel par page, au plus 8) pour la
  retrouver au-delà du match connu.
- Étape 3 (D-21) : un marque-page rangé par une passe sans cursor dont la première page est abandonnée n'a pas
  d'ancre (la tête n'a pas été lue) ; la passe suivante retombe sur la règle sans ancre, puis les marque-pages
  suivants en portent une.
- Étape 3 : un match rangé sans timeline ou sans carnage (`EventsFailed`, `CarnageFailed`) est connu
  ensuite : le rebalayage ne le retente jamais (le commentaire de `captureMatchTimeline` dit le contraire).
  Hors périmètre.
- Étape 3 : le `PostScore` du runner Halo 5 récupère un carnage par match inséré
  (`PersistPerMatchRatings`) : une passe de rattrapage de 200 matchs coûte 200 appels de plus ; à mesurer à
  l'étape 5bis. Le tick du suivi en direct (watcher) passe aussi par `RunDelta` pour Halo 5 : il rattrape
  donc lui aussi, avec son `MaxMatches` serré comme budget.
- Étape 3 : une passe au budget atteint pile sur une page pleine ne voit la fin qu'à la passe suivante
  (une page vide, zéro match inséré) : comportement voulu, testé.
- Étape 5 (réglé par D-18) : le moteur V1 versait l'échec de récupération d'un match dans les avertissements, pas dans les erreurs, rangeait la fin avec un match perdu pour toujours (contraire à D-5) et rebalayait la liste depuis le haut à chaque passe (862 appels de liste pour 1 000 matchs). V1 ne rattrape plus : le rattrapage est fait par le seul cycle de l'ordonnanceur.
- D-18 : le filet `syncPlayer` de l'ordonnanceur (orchestrateur V2 non câblé) passe par V1 pour Halo Infinite : il ne rattrape donc plus l'historique de ce titre (delta pur, `HistoryPending` jamais rempli, pas de cadence de rattrapage). Le serveur câble toujours l'orchestrateur V2 : le filet ne sert qu'à un montage sans V2. Halo 5 garde son rattrapage dans le runner, même par le filet. Écrit dans le commentaire de contrat de `syncPlayer`.
- D-18 : le suivi en direct, la synchronisation manuelle d'un joueur et la synchronisation initiale (V1, delta pur) ne rattrapent plus l'historique : seul le cycle de l'ordonnanceur (et le bouton « Synchroniser maintenant », qui le lance) le fait. `SyncResult.HistoryExhausted` supprimé (aucun lecteur de production après la fin du rangement par V1) ; son écriture et ses assertions de test côté Halo 5 (`livesync/catchup.go`) sont retirées avec lui.
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
- Étape 5ter (corrigé par D-24) : DÉFAUT `populateOutcome` (`sync/v2/cycle.go`) compte inséré tout inconnu listé, récupéré ou non (match en 404 ou en 500 durable compris) : `MatchesInserted` et la notification « nouveaux matchs » sont faux ; un match introuvable en tête de liste est de plus redemandé à chaque cycle, sans fin (une demande par cycle). Test gardé sauté : `TestDifficile_MatchIntrouvable_NEstPasCompteInsere`.
- Étape 5ter : un cycle annulé en pleine récupération ne s'arrête pas avant la persistance : la phase de récupération rend les matchs déjà récupérés (les autres sont des erreurs de récupération), la persistance les soumet à la file puis le vidage échoue sur le contexte annulé ; les matchs soumis finissent en arrière-plan ou au journal d'écriture. Sans conséquence sur la convergence (marque-page non avancé, connus sautés), mais le travail soumis après l'annulation est inutile.
- Étape 5ter : la séquence d'arrêt vide la file de persistance dans une phase bornée à 5 s (`phaseBatchDrainCap`) ; une passe de rattrapage laisse jusqu'à 200 journaux d'écriture (environ 0,2 s par match sur ce banc, 1,1 s mesuré en production) : l'arrêt n'est alors pas « propre » et les bases se ferment sous le worker, le reste est rejoué au démarrage suivant (testé : aucun doublon, aucune perte). Comportement voulu du journal d'écriture, noté pour mémoire.
- Étape 5ter : lire l'état d'une base joueur (`OpenReadForQuery`) pendant qu'un worker de persistance l'écrit peut rendre `sql: database is closed` (poignée en cache fermée par le worker) ; `historystate.Complete` rend alors « complet ». Réglé par D-25 (découverte, `historyrepo` et lectures du banc par `OpenPlayerReadForQuery` ; `Complete` rend l'erreur).
- D-25 (gate) : `go test ./internal/archlint/` échoue sur la branche intégrée hors de ce lot : `TestRepoRootNEstPlusLuHorsConfigEtResolveur` compte 1 lecteur de RepoRoot dans `internal/sync/replayartifacts/artifacts.go` pour 2 à l'allowlist (fichier non touché ici, dernier commit 822-14) ; l'allowlist est à abaisser par la piste qui l'a migré. Gardes DuckDB, ouverture RW et clé d'historique vertes.
- D-25 : deux autres lecteurs d'une base joueur passent encore par `OpenReadForQuery` dans le serveur (instantané de fin de cycle `sync/snapshot/snapshot_cutter.go`, `sync/match_recomputer.go`) : sans handle en cache, ils ouvrent en lecture seule et peuvent faire refuser le worker (« different configuration ») le temps de leur lecture. Hors périmètre, noté.
- D-23 : un match inconnu proche du haut de liste qui échoue toujours (stats en 500) est maintenant retenté à chaque delta tant qu'il reste à moins de 25 entrées de connus au-dessus de lui (il remet la suite à zéro) ; en V1 et en V2 historique complet le delta lit alors deux pages au lieu d'une, jusqu'à ce que 25 matchs plus récents l'aient éloigné. Pas d'abandon hors rattrapage (D-16). Hors périmètre, noté.
- D-23 : sans ancre (marque-page rangé avant D-21), N compte les entrées jusqu'à la dernière inconnue ; un connu arrivé plus bas n'est pas compté (page de recouvrement ratée, reprise depuis 0 journalisée et comptée) : état transitoire d'une passe.
- D-24 : le moteur V1 (suivi en direct, synchronisation manuelle d'un joueur) ne lit ni n'écrit l'ensemble des matchs à ne plus demander : un match introuvable en haut de liste y est redemandé à chaque tick tant qu'il reste dans les premières entrées (une demande par tick). Hors périmètre (V1 est un delta pur, D-18), noté.
- D-24 : l'ensemble est lu par une seconde ouverture lecture seule de la base joueur par cycle et par joueur (`known_loader_skip.go`, capacité facultative `SkipSetLoader`) parce que `known_loader.go` était pris par une autre piste ; à replier dans `LoadKnown` (lecture sur la poignée déjà ouverte) quand cette piste sera intégrée. Replié par D-25 : `readHistoryState` lit l'ensemble sur l'ouverture de `LoadKnown` ; `known_loader_skip.go` et `SkipSetLoader` supprimés.
