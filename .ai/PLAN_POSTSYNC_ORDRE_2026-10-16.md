# Plan — Post-sync : un match d'escouade ne s'affiche plus « Solo » sans LUSR pendant des minutes (2026-10-16)

> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, aucun report,
> statut de chaque item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture,
> zéro correctif hors périmètre : les trouvailles vont en section « Découvertes »).
> Branche : `feat/postsync-ordre` depuis `feat/v75`, worktree dédié `../LevelUp-wt-postsync`.
> Exécutant : Sonnet ; relecture adversariale : Opus (lot sync = à risque).
> Reprise de session : lire ce fichier (cases cochées), `git log --oneline -15` de la branche,
> dernière entrée du `.ai/thought_log.md`.

## Objectif et critère de succès

Symptôme (user, 16/10) : un match d'escouade fraîchement synchronisé s'affiche « Solo », sans
LUSR, pendant quelques minutes, puis se complète seul.

Causes établies sur pièces (16/10) :
- C1. `is_with_friends` est écrit FAUX par défaut au premier écrit (`sync/collect.go:216-222`,
  colonne `DEFAULT FALSE`, vue `_latest` en `COALESCE(..., FALSE)`) : « pas encore calculé » et
  « solo » sont indiscernables. La vraie valeur n'est posée qu'à l'étape 3.5 du post-sync.
- C2. Le post-sync est séquentiel (`sync/engine_postsync.go:267-477`) et les étapes film lentes
  (1.57 source du kill : 8 films / 5 min ; 1.58 rejeu : 5 min) passent AVANT LUSR (2),
  CSR (3.1), escouade (3.5) et agrégats (4).
- C3. Le cache de lecture joueur n'est vidé qu'en fin de post-sync (`engine_postsync.go:170`) ;
  `HomeMatchesCache` (45 s) n'est jamais vidé par le cycle automatique V2.

Critère de succès : sur un vrai match d'escouade synchronisé, la tuile affiche « Escouade » dès
la première lecture, le LUSR est présent avant la lecture des films ; aucun résultat final ne
change (citations, domination, préparation du match, source du kill, rejeu).

## Décisions tranchées (pas de décision en cours de route)

- D1. Ordre cible du post-sync (contraintes de dépendance vérifiées le 16/10) :
  `-2 lignes d'enrichissement` → `scoring` → `1.54 événements` → `1.56 PSA` → `3.5 escouade` →
  `1.7 domination` → `2 LUSR` → `3.1 CSR` → `4 agrégats` → **vidage des caches** →
  `1.57 source du kill` → `1.58 rejeu` → `1.6 citations` → `4.5 médias` → `5 succès` →
  `6 préparation` → (vidage final existant, `defer`).
  Interdits : 1.54 après la domination ou les citations (domination figée à 0) ; citations avant
  1.57 (citations par arme perdues, calcul unique) ; préparation avant 1.57 ou les citations ;
  PSA après les citations ou l'escouade ; 1.58 avant 1.57.
- D2. `is_with_friends` est calculé dès le premier écrit, avec le MÊME prédicat que le recalcul
  (même équipe, xuid dans les amis, xuid différent du joueur). Le recalcul de l'étape 3.5 reste
  en place, il reste l'autorité convergente (ami ajouté/retiré). Liste d'amis inconnue ou vide →
  champ non posé (comportement actuel).
- D3. Aucun changement de schéma, de vue `_latest`, ni de sémantique de colonne. L'écriture passe
  par le champ pointeur existant `EnrichmentRow.IsWithFriends` (chemin INSERT-only documenté,
  `persist/player_persister.go:218`).
- D4. Hors périmètre : indicateur « analyse en cours » côté web (film, rejeu), CSR (aucun défaut
  constaté, voir « Constat CSR »), Halo 5 (`livesync` n'a pas d'étape film).
- D5. (utilisateur, message du 16/10, relayé par le coordinateur) Écart accepté : la pénalité de départ du
  LUSR v2 est lue sur la passe crédit-seul avant 1.57. Mesuré sur la copie de la base partagée : 1 match sur
  2 394 (0,04 %) change de contexte de départ (un bot, égalité 28-28 contre 28-29). Justification : l'ancien
  ordre lisait déjà la passe crédit chaque fois que 1.57 n'atteignait pas un match dans son cycle (8 films
  par cycle, file globale) ; la source du LUSR dépendait donc du cycle. D1 la rend constante.

## Étape 0 — Préparation (rapide)

- [x] 0.1 `git worktree list` ; worktree `../LevelUp-wt-postsync`, branche `feat/postsync-ordre`
  depuis `feat/v75` ; noter le commit de base.
- [x] 0.2 Base verte : `go test ./internal/sync/... ./internal/persist/...` et
  `go test -tags=integration ./internal/sync/... ./internal/persist/...` (`-p 1`) ; noter les
  rouges préexistants éventuels (ils restent à réparer, règle CI).
- [x] 0.3 Inventaire écrit dans ce fichier (section « Inventaire ») : colonnes de
  `mv_player_matches` (`sync/aggregates.go`) et étape productrice de chacune. Si une colonne
  dépend de 1.57, 1.58 ou des citations, un second rafraîchissement des agrégats est ajouté après
  1.6 (item 1.4) ; sinon 1.4 est statué `[~]`.

Gate 0 : tests de base exécutés, inventaire rempli.

## Étape 1 — Ordre du post-sync (moyen)

- [x] 1.1 Réordonner `runPostSyncPipeline` selon D1 (déplacement de blocs, aucun changement de
  logique d'étape). Construire `postSyncFilmSteps` une seule fois et le réutiliser.
- [x] 1.2 Vidage du cache de lecture joueur après les agrégats
  (`duckdbpkg.InvalidatePlayerReadCaches`), le `defer` final reste (ratchet
  `archlint/player_read_cache_invalidation_test.go`).
- [x] 1.3 Vidage de `HomeMatchesCache` au même point, pour V1 ET V2, via un rappel du moteur
  (modèle `mediaHook` / `prestigeHook` ; `sync` n'importe pas `service`). Brancher le rappel dans
  le câblage du serveur pour les deux chemins.
- [~] 1.4 Second rafraîchissement des agrégats après 1.6 — seulement si l'inventaire 0.3 le
  justifie.
- [x] 1.5 `sync/postsync_clock.go` : la liste des étapes affichées suit l'ordre D1 et contient
  `kill_source` et `replay_artifacts` (aujourd'hui absentes, `weapon_kills` périmé) — nécessaire
  pour lire l'ordre réel au tableau de bord en étape 4.
- [x] 1.6 Test qui épingle l'ordre : `StepTimings` d'un post-sync contient `friends`,
  `skill_rating`, `aggregates` AVANT `kill_source`, et `citations` APRÈS `replay_artifacts`,
  `snapshot_readiness` en dernier. Mutation : remettre l'ancien ordre → rouge.
- [x] 1.7 Test d'intégration : match inséré par un coéquipier (`events_loaded=false`) → après un
  post-sync, domination NON figée à 0 et citations avec événements (preuve que 1.54 reste avant).
- [x] 1.8 Seuils : `runPostSyncPipeline` ≤ 80 L ou exemption existante inchangée ; aucun nouveau
  fichier à la racine de `internal/sync` (ratchet `sync_root_freeze_test.go`).

Gate 1 : `go test ./internal/sync/...`, `go test -tags=integration -p 1 ./internal/sync/...
./internal/persist/...`, `go test ./internal/archlint/...`, `golangci-lint` 0 nouveau.

## Étape 2 — « Escouade » dès le premier écrit (moyen)

- [x] 2.1 Prédicat pur unique (dans un fichier existant de `internal/sync`) :
  `withFriends(participants, playerXUID, friendXUIDs) bool` = même équipe, xuid ami, xuid ≠
  joueur, bots exclus comme dans le SQL.
- [x] 2.2 `fetchedMatch` porte les xuids amis résolus ; `buildBatchFromFetchedMatchCtx` pose
  `EnrichmentRow.IsWithFriends` quand la liste est connue (signature ≤ 5 paramètres).
- [x] 2.3 Résolution des amis UNE fois par passe : V1 (`engine_batch_path.go`, via
  `friendsLoader` + `LookupFriendXUIDs`) et V2 (`engine_v2bridge.go`, même chose côté cycle).
  Échec de résolution → log WARN, champ non posé, pas d'échec du sync.
- [x] 2.4 Test d'équivalence : sur un même jeu de participants (même équipe, équipe adverse, joueur
  présent dans sa propre liste d'amis, bot, liste vide), le prédicat pur et
  `loadMatchesWithFriends` (SQL) rendent le même verdict.
- [x] 2.5 Test d'intégration persist : lot d'un match d'escouade → `player_match_enrichment_latest`
  porte `is_with_friends = TRUE` AVANT tout post-sync ; le recalcul 3.5 qui suit promeut 0 et
  démote 0.
- [x] 2.6 Garde-rail ART : `go test ./internal/sync/ -run NoArt` vert sans allowlist ajoutée.

Gate 2 : mêmes commandes que le gate 1 + `go test ./internal/persist/...`.

## Étape 2bis — Simulation de bout en bout sur le faux serveur Halo (moyen)

Harnais existant (plan de stratégie des tests de sync, 24/05) : `internal/sync/halotest/fake_server.go`
sert le match réel figé `internal/sync/testdata/jgtm_full_match/` (stats, skill, film, historique) ;
`internal/sync/engine_e2e_test.go` l exerce déjà.

- [x] 2b.1 Test de bout en bout (dans un fichier de test existant ou un nouveau `*_test.go` sous
  `internal/sync/`) : faux serveur + base vierge + liste d amis contenant un coéquipier réel du
  match figé → après le PREMIER écrit (avant le post-sync), la ligne d enrichissement lue par
  `player_match_enrichment_latest` vaut `is_with_friends = TRUE` ; après le post-sync complet :
  LUSR présent, domination non NULL, citations présentes, ordre des `StepTimings` conforme à D1.
- [x] 2b.2 Même test sans liste d amis → `is_with_friends = FALSE`, post-sync identique par
  ailleurs.
- [x] 2b.3 Comparaison avant/après : le même test exécuté sur le commit de base (ancien ordre) et
  sur la branche donne les MÊMES valeurs finales (citations, domination, LUSR, préparation,
  escouade) ; seule la chronologie change. Écart = arrêt et analyse.
  Contrôle ajouté (coordinateur, 16/10) : la pénalité de départ du LUSR v2 donne la MÊME valeur avant et
  après réordonnancement ; si le match figé n'a pas de joueur parti, test d'intégration ciblé : match
  avec un partant, passe crédit-seul seule présente (aucune passe film) -> `loadQuitTimeline` rend
  `available=true` et le delta suit la timeline, pas le repli. Si le LUSR diffère entre passe crédit et
  passe film (couverture différente) : arrêt et compte rendu chiffré.

Gate 2bis : 2b.1-2b.3 cochés, tableau avant/après dans le journal.

## Étape 3 — Vérification complète (lourd)

- [ ] 3.1 (après le gate 2bis) `cd apps/go-api && go test ./...` et `go test -tags=integration ./...` verts.
- [ ] 3.2 `make go-api-lint` ; `make gate-push`.
- [ ] 3.3 Relecture adversariale (skill `adversarial-review`, Opus, contexte frais) du diff
  complet ; chaque constat P1/P2 corrigé ou statué par écrit.
- [ ] 3.4 Push de la branche, CI verte (tous jobs).

Gate 3 : 3.1-3.4 cochés, sorties consignées dans le journal.

## Étape 4 — Validation sur données réelles (moyen, nécessite un vrai match)

- [ ] 4.1 Mesure AVANT sur l'historique local : pour les matchs déjà présents, délai entre
  `match_registry.first_sync_at` et le `written_at` de la ligne d'enrichissement « friends » et de
  la ligne LUSR (requête sur une COPIE de la base, `cmd/diag_q`).
- [ ] 4.2 Serveur local relancé sur le code de la branche (après accord des sessions pairs,
  `ListAgents`), un cycle de synchro avec au moins un nouveau match d'escouade. S'il n'y en a
  aucun, demander au user d'en jouer un (blocage légitime).
- [ ] 4.3 Sur ce match : tuile « Escouade » à la première lecture (navigateur, capture), LUSR
  présent avant la fin de 1.57, ordre des étapes lu au tableau de bord admin, citations et
  domination présentes, préparation atteinte, source du kill traitée au cycle normal.
- [ ] 4.4 Mesure APRÈS (même requête qu'en 4.1) et comparaison écrite dans le journal.
- [ ] 4.5 Recalcul de l'escouade relancé à blanc sur les 4 joueurs : 0 promotion, 0 démotion.

Gate 4 : 4.1-4.5 cochés, capture et chiffres dans le journal.

## Étape 5 — Clôture (rapide)

- [ ] 5.1 Documents : ordre du post-sync dans `docs/SYNC_GUIDE.md` ET `docs/FR/` (bilingue) s'il
  y est décrit ; commentaires d'en-tête de `engine_postsync.go` alignés.
- [ ] 5.2 `.ai/thought_log.md`, `.ai/REGISTRE_REPORTS.md` (découvertes non traitées).
- [ ] 5.3 Skill `delivery-checklist`.
- [ ] 5.4 Fusion dans `feat/v75` : SUR ACCORD du user.

## Constat CSR (16/10, sur pièces, aucune action)

- Sur 691 matchs en matchmaking synchronisés moins de 5 min après leur fin (base locale + base de
  prod copiée), 688 ont leurs données de classement ; les 3 autres sont des matchs avortés de
  moins de 2 min. Le retard de publication du classement par Halo n'apparaît pas dans les données.
- Les 58 matchs « Ranked » locaux sans classement durent tous 32 à 76 s : matchs avortés, Halo ne
  publie pas de classement pour eux. Pas un défaut.
- La correction de la semaine passée (le classement décidé par la réponse de Halo) est dans
  `feat/v75` mais PAS dans `main` : elle arrive en prod avec la version 7.5. Dernière écriture de
  CSR par match en prod : 26/05 ; aucun match classé de plus de 2 min n'y a été synchronisé depuis
  le 17/09, donc rien à observer là-bas d'ici le déploiement.

## Inventaire (étape 0.3)

`mv_player_matches` (`sync/aggregates.go`, `playerMaterializedViews`) = 5 colonnes lues dans
`player_match_enrichment_latest` (filtre `performance_score IS NOT NULL`) :

| Colonne | Étape productrice | Dépend de 1.57 / 1.58 / citations ? |
|---|---|---|
| `match_id` | ligne d'enrichissement (`-2` ou collecte) | non |
| `performance_score` | `scoring` (`batchComputePerformanceScores`, lit `match_participants` seul) | non |
| `session_id`, `session_label` | `scoring` (`recalculateSessionsInline`) | non |
| `is_with_friends` | `3.5 friends` (et, dès l'étape 2 de ce plan, le premier écrit) | non |

Conclusion : aucune colonne ne dépend de 1.57, 1.58 ni des citations ; un second rafraîchissement
des agrégats après 1.6 n'apporterait rien. Item 1.4 : `[~]` couvert par cet inventaire.

## Découvertes

- (16/10, exécutant) Arrêt initial retiré : la pénalité de départ du LUSR v2 (`loadQuitTimeline`)
  lit `match_kill_events_latest`, mais la passe crédit-seul écrite au premier écrit (voir « Journal
  d'exécution ») la porte déjà ; D1 reste valide. Le constat est conservé comme contrôle de 2b.3.
- (16/10, exécutant) Le bloc engagement du scoring (`loadSyntheticKillEventsForMatch`, `engagement.go`)
  lit `match_kill_events_latest` avant 1.54 dans l'ordre actuel comme dans D1 ; non traité.
- (16/10, exécutant, étape 2bis) Le faux serveur HTTP (`halotest`) ne tourne pas dans ce dépôt : les fichiers
  du match figé (`chunks/`, `api_*.json`) sont exclus par `.gitignore` (seuls `README.md` et `manifest_raw.json`
  sont suivis) et aucun test hors `halotest` ne l'utilise (`engine_e2e_test.go` ne l'exerce pas, contrairement
  au texte du plan). La simulation de 2bis utilise donc `mockHaloClient` + le chunk de film v41 versionné
  (275 événements, 101 morts) : elle tourne en CI. Non traité : brancher le vrai client HTTP sur le faux serveur.
- (16/10, exécutant, relecture 3.3 ronde 2) `LookupFriendXUIDs` (`sync/session_recalc.go`) avale l'erreur de chaque
  requête par gamertag : une base fermée en cours de boucle rend une liste partielle sans WARN, pour le recalcul
  comme pour le premier écrit. Un emprunt RW du `SharedBorrower` n'étant pas compté, `ResolveFriendXUIDs` ne
  relit pas comme le fait `KnownLoader`. Non traité (fonction partagée, fenêtre étroite, le recalcul converge).
- (16/10, exécutant, relecture 3.3) `engine.go` (767 L), `api/handlers/sync_handler.go` (702 L) et
  `scheduler/auto_sync.go` (531 L) dépassent 500 L et ont grossi de 10 à 12 lignes avec le rappel de vidage du
  cache Home ; aucun gate ne mesure la taille de fichier (`funlen` seulement). Découpage non traité.
- (16/10, exécutant, étape 1) `apps/web/src/features/admin/convergence/timeline.ts` (`STEP_ORDER`, mapping
  couleur du tableau de bord) ne connaît pas `kill_source`, `replay_artifacts`, `snapshot_readiness` et
  garde l'ancien ordre ; ces étapes retombent sur la couleur de repli. Cosmétique, hors lot Go : non traité.
- (16/10, superviseur) Le worktree `feat/citations-film` (plan `.ai/PLAN_CITATIONS_FILM_2026-10-16.md`,
  non implémenté à ce jour) touche aussi l étape des citations. Ce plan place les citations APRÈS
  les films, ce qui va dans son sens (son étape E4 suppose déjà les citations après le Deriver).
  À signaler à la fusion, pas à traiter ici.

## Journal d'exécution

- 16/10 Étape 0 CLOSE. 0.1 : worktree `LevelUp-wt-postsync`, branche `feat/postsync-ordre`, base `27947da10`.
  0.2 : `go test ./internal/sync/... ./internal/persist/...` vert (sync 138 s, persist 4 s) ;
  `go test -tags=integration -p 1` des mêmes paquets vert (sync 223 s, persist 59 s) ; aucun rouge
  préexistant. 0.3 : inventaire rempli, 1.4 statué `[~]` (aucune colonne de `mv_player_matches` ne dépend
  de 1.57/1.58/citations).
- 16/10 Arrêt levé par le coordinateur : D1 maintenue, prémisse de l'arrêt fausse (vérifié sur pièces :
  `persist/shared_persister.go:380-388`, `persistKillerVictim` écrit AUSSI une passe crédit-seul via
  `persistCreditKillEvents(CreditKillEventsFromPairs(...))` dans la transaction du premier écrit ;
  `persist/kill_events_credit.go:143` pose `FeedKillerXUID` ; `time_ms` repris des couples ; préséance
  inversée le 2026-08-03 : le crédit est la base, le film enrichit ; le chunk highlight est récupéré au
  premier écrit en V1 et en V2 Phase 3, `v2/fetchers.go:198`, sinon par 1.54 qui reste avant le LUSR).
  Options a/b/c : SANS OBJET. Contrôle ajouté à 2b.3 : la pénalité de départ du LUSR v2 doit donner la
  même valeur avant/après ; test ciblé crédit-seul (match avec partant, aucune passe film) :
  `loadQuitTimeline` rend `available=true`. Si le LUSR diffère entre passe crédit et passe film : arrêt.
- 16/10 Étape 1 CLOSE. Ordre D1 en place (`runPostSyncPipeline` 318 -> 232 lignes : PSA, catalogue,
  escouade, domination et citations extraites en méthodes dans `engine_postsync_scoring.go`, aucun
  nouveau fichier racine ; `runPostSyncPipeline` garde l'exemption funlen existante, le lint ne voit
  aucune issue nouvelle). 1.2 : `flushReadCaches` après les agrégats + `defer` final conservé
  (ratchet archlint vert). 1.3 : `SyncEngine.WithHomeCacheFlush` + `ServiceRegistry.InvalidateHomeMatches`,
  branché aux trois sites de construction du moteur (`scheduler.BuildEngine`, `SyncHandler.newEngineFor`,
  `buildSyncEngineFactoryParityComplete` -> V2), chacun gardé par un test de câblage (`HasHomeCacheFlush`).
  1.5 : `postSyncStepNames` = ordre d'exécution, `weapon_kills` retiré, `kill_source` et `replay_artifacts` ajoutés.
  1.6 : `TestRunPostSyncPipeline_StepOrder` (ordre littéral) et `..._FlushesHomeCacheBeforeFilmSteps`
  (vidage vu avant la source du kill, compteur 0 puis 1) ; mutation vérifiée : l'ancien ordre rend le test
  rouge. 1.7 : `TestRunPostSyncPipeline_EventsBeforeDominance` (intégration) : événements arrivant à 1.54,
  `dominance_flag` = valeur attendue (non nulle) ; mutation (domination avant 1.54) : rouge (`flag = 0`).
  Réserve 1.7 : le volet « citations avec événements » n'est pas exécutable dans ce harnais (les citations
  exigent la base metadata) ; la contrainte « 1.54 avant les citations » est tenue par le test d'ordre 1.6.
  Gate 1 : `go test` sync/persist/archlint/scheduler/wire/handlers/cmd/server verts ; `go test -tags=integration -p 1`
  sync + persist vert (sync 241 s) ; `golangci-lint --new-from-rev=27947da10` sur les paquets touchés : 0 issue.
- 16/10 Étape 2 CLOSE. 2.1 `withFriends` (friends_recompute.go, jumeau du SQL de `loadMatchesWithFriends`).
  2.2 `fetchedMatch.FriendXUIDs` + `applyIsWithFriends` dans `buildBatchFromFetchedMatchCtx` (signature inchangée, 5 paramètres).
  2.3 résolution UNE fois par passe : V1 `run()` -> `historyPaginationInputs.friendXUIDs` -> chaque match (`engine.go`,
  `engine_fetch.go`) ; V2 `cycleBatchPersisterV1Bridge.resolveCycleFriends` (un appel par joueur du cycle, avant le
  premier Submit) + `ApplyFriendsToBatch`, source branchée par `syncv2.WithFriendsSource(persister, deps.Friends.Get)`
  (constructeur inchangé). Échec/liste vide : WARN, champ non posé. 2.4 `TestWithFriends_Cases` +
  `TestWithFriends_MatchesSQLVerdict` (11 cas, même verdict que la requête du recalcul). 2.5
  `TestFirstWrite_PosesIsWithFriends_RecomputeConfirms` (dans `internal/sync`, qui seul peut appeler le recalcul) :
  TRUE avant tout post-sync, recalcul 0 promotion / 0 démotion. 2.6 `TestNoARTPatternsOnProtectedTables` etc. verts,
  aucune allowlist touchée. Gate 2 : tests sync/persist/archlint/cmd-server verts, intégration sync+persist verte,
  lint (`--new-from-rev` du commit E1) 0 issue.
- 16/10 Étape 2bis (tests) : `postsync_e2e_test.go` (+ `postsync_e2e_firstwrite_test.go`) : match d'escouade
  (Player0 + ami Player2, 8 joueurs, film v41, médaille mappée) joué par le chemin de production
  (`fetchAndPersistMatches` puis `runPostSyncPipeline`). 2b.1 : après le premier écrit `is_with_friends` = TRUE
  AVANT le post-sync ; après : LUSR présent (`LUSR=1503.1/Or IV`), domination calculée (0), citation `e2e_medal`=2,
  événements chargés, ordre D1. 2b.2 : sans liste d'amis FALSE/FALSE, le reste identique au cas escouade.
  2b.3 : même épreuve sur le commit de base `27947da10` (`git archive` en dossier temporaire, aucun worktree touché) :
  valeurs finales IDENTIQUES (voir tableau au journal), seuls changent `is_with_friends` après le premier écrit
  (FALSE -> TRUE) et l'ordre des étapes. Test ciblé du contrôle coordinateur :
  `sync/skill/skill_v2_quit_credit_pass_test.go` (passe crédit-seule seule en base -> `available=true`, delta
  suit la timeline et diffère du repli) : vert.
  ARRÊT (consigne 3 du coordinateur) : mesure sur la COPIE de la base partagée (`diag_q`, lecture seule) de la
  passe crédit-seul contre la passe fusionnée (`match_kill_events_latest`) : sur 2 394 matchs comparables, 2 383
  identiques, 11 différents (9 avec plus de frags côté fusionné, 2 avec moins ; 295 659 frags crédit contre
  295 704 fusionnés, +0,015 %), tous avec des partants. Sur les 43 partants de ces 11 matchs, le contexte du
  départ (en retard / pas en retard) change pour UN seul : match `760fb768-55c9-48c9-9b2d-353a6846c696`
  (CTF Empyrean, joueurs suivis : Madina97294, JGtm, Chocoboflor, tous côté 0, victoire), partant = un bot
  (`bid(56...`), égalité 28-28 (crédit) contre 28-29 (fusionné) à 372 541 ms : delta « sans lien » contre « lié ».
  Les bots sont dans les rosters du LUSR v2 (`buildTwoTeamRosters` ne filtre que xuid vide). Soit 1 match sur
  2 394 (0,04 %) où la pénalité de départ d'un partant, donc le LUSR, peut différer selon que la passe lue est
  la passe crédit ou la passe fusionnée. Gate 2bis NON passé : décision du coordinateur attendue.
- 16/10 Arrêt levé par le coordinateur (D5 : écart accepté par l'utilisateur, voir « Décisions »). 2b.3 statué
  `[x]` en citant D5 ; gate 2bis PASSÉ : 2b.1, 2b.2, 2b.3 cochés, tableau avant/après au journal (thought_log).
- 16/10 Étape 3.1 : `go test ./...` vert (201 paquets ok, 7 min 41 s, avant les corrections de la relecture).
- 16/10 Étape 3.3, ronde 1 de la relecture adversariale (2 relecteurs Opus, contexte frais, lentilles
  L1 anti-ART/concurrence/exactitude de l'ordre et L3+L6 anti-patterns/tests ; rien n'a été dit de l'auteur).
  Relecteur A : 1 constat recevable (P2), 22 conditions vérifiées qui tiennent. Relecteur B : 6 constats,
  22 conditions vérifiées. Triage :
  - B1 P1 (premier écrit V2 sans test, `persist_v1bridge.go`, câblage `sync_v2_wiring.go`) : CORRIGÉ.
    `TestPersistCycle_PosesIsWithFriendsOnFirstWrite` (ami coéquipier -> TRUE, adverse -> FALSE, sans source ->
    non posé ; mutation : retirer `ApplyFriendsToBatch` -> rouge) et `TestBuildCyclePersister_WiresFriendsSource`
    (extraction de `buildCyclePersister` + `CycleBatchPersisterHasFriendsSource`).
  - A1 P2 (`resolveCycleFriends` lisait `LookupCachedDB` sans emprunt : connexion fermable sous la lecture par
    une bascule RO/RW) : CORRIGÉ. `WithFriendsSource` prend l'emprunt `SharedBorrower` de la découverte
    (`knownSetSharedBorrower`), un seul emprunt par cycle ; base illisible -> WARN, champ non posé.
  - B2 P2 (branchement V1 `engine.go` sans test) : CORRIGÉ par un garde textuel `TestRunHistoryInputsCarryFriends`
    (`run()` n'est pas exécutable sans moteur complet) ; la boucle `engine_fetch.go` est couverte par les tests e2e.
  - B3 P2 (vidage `InvalidatePlayerReadCaches` à mi-pipeline sans test) : CORRIGÉ en ajoutant `flushReadCaches` aux
    points du ratchet `archlint/player_read_cache_invalidation_test.go` (renforcement, pas une allowlist) ; son
    appel avant 1.57 est tenu par `..._FlushesHomeCacheBeforeFilmSteps`.
  - B4 P2 (commentaires faux : doc de `runPostSyncPipeline`, trois renvois à `engine_postsync_films.go`
    inexistant, « le premier écrit l'a déjà posé ») : CORRIGÉS dans le même diff (règle 17).
  - B5 P2 (histoire dans les en-têtes de tests ; `TestPostSyncE2E_Dump` = harnais de mesure ponctuelle) : CORRIGÉ.
    Dump supprimé (la comparaison est consignée dans ce plan et le journal), en-têtes réécrits.
  - B6 P2 (engine.go 757 -> 767, sync_handler.go 690 -> 702, auto_sync.go 520 -> 531 : fichiers déjà au-delà de
    500 L qui grossissent) : STATUÉ, non corrigé : +10 à +12 lignes de champ, d'option et de commentaire par
    site, inévitables pour brancher un rappel sur trois constructeurs de moteur ; `engine_postsync.go`, le fichier
    visé par le lot, passe de 527 à 442 lignes. Dette de découpage de ces trois fichiers : Découvertes.
- 16/10 Étape 3.3, ronde 2 (1 relecteur Opus, contexte frais, sur les seules corrections `3f3f9b67d..b9350bc68`) :
  0 P0, 0 P1 (1 en ronde 1 : décroissance respectée), 2 constats P2, 14 conditions vérifiées. Triage :
  - C1 P2 (le test V2 ne distinguait pas l'emprunt du getter en cache ni ne vérifiait le release) : CORRIGÉ. Le getter
    rend une base vide, l'emprunt est compté ; le test exige 2 emprunts et 2 libérations. Mutations vérifiées :
    revenir à `p.sharedDB()` -> rouge ; retirer `defer release()` -> rouge.
  - C2 P2 (état RW du provider : l'emprunt rend la connexion de l'écrivain sans la compter ; libérée pendant la
    boucle, la liste d'amis ressort partielle, sans WARN car `LookupFriendXUIDs` avale l'erreur par gamertag) :
    STATUÉ. Fenêtre étroite, aucune corruption, le recalcul du post-sync converge ; le commentaire de
    `WithFriendsSource` dit désormais exactement ce que le code tient. Relire/réessayer la résolution (contrat de
    `SharedBorrower`) et remonter l'erreur de `LookupFriendXUIDs` : Découvertes (hors lot, `LookupFriendXUIDs` est
    partagé avec le recalcul).
