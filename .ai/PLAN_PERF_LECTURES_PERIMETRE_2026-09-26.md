# Plan : lectures par périmètre — ADR 0036 et lot A (vue match, Relations) — 2026-09-26

> Source : `.ai/HANDOFF_PERF_CHARGEMENTS_2026-09-24.md` (§4 ce qui reste, §6 recommandation,
> §7 décisions en attente) et le plan clos `.ai/PLAN_PERF_CHARGEMENTS_2026-09-23.md` (§9 ter,
> journal du lot L7, lectures consignées (a) à (f) et découverte (3) ; §11).
> Reprise demandée par l'utilisateur le 2026-09-26 (« reprends le handoff, worktree dédié »).
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun item sans statut, zéro fix
> hors périmètre, découvertes consignées ici). Le plan fait foi en cas de divergence.

## 0. Décisions tranchées par le superviseur (2026-09-26)

- **Recommandation §6 du handoff retenue.** L'ADR « lectures par périmètre » s'écrit tout de suite ;
  le lot A (vue match + Relations par le patron de l'annuaire) est le seul lot structurel
  acceptable AVANT la mesure prod (risque faible : lectures seules, aucune écriture, aucun
  invariant anti-ART touché). Les lots B (pages à historique complet, fenêtre du kill-feed) et C
  (compaction append-only) attendent la mesure prod : HORS de ce plan.
- **Hors plan, déjà réglé ou à l'utilisateur** : la branche `feat/perf-chargements` (locale et
  distante) n'existe plus (ménage de la bascule du 2026-09-25, branches 154 -> 4) ; la reconnexion
  SSO de Chocoboflor reste un geste de l'utilisateur (ADR 0023, jamais de re-capture) ; les
  découvertes du §11 du plan clos restent consignées, pas des lots.
- **Branche** `feat/perf-perimetre` depuis `feat/v75` `ea5682373` ; worktree
  `C:\Users\Guillaume\Projects\LevelUp-wt-perf-perimetre`. Fusion `--no-ff` dans `feat/v75` par le
  superviseur après CI verte au niveau job de la branche (go utilisateur demandé à ce moment-là).
- **Ajout du superviseur** : le changelog 7.5.0 (écrit le matin du 2026-09-23) ne mentionne ni la
  campagne perf (fusionnée le soir même) ni l'ADR 0036 ; il annonce « quatre ADR ». Étape P3.

## 1. Règles communes aux étapes

- Un exécuteur Opus par étape, dans le worktree, SÉQUENTIELLEMENT (P1 close avant P2, P2 avant P3).
- Périmètre FERMÉ par étape (liste de fichiers). Toute autre modification = découverte au §5.
- Aucun agent ne lance le serveur (`air`, `server.exe`), Vite, un navigateur, un backfill, ni
  n'ouvre en écriture une base sous `data/`. Chronométrage SQL : COPIE de
  `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\warehouse\shared_matches_v2.duckdb`
  (avec son `.wal` s'il existe) dans le scratchpad de l'agent, sonde Go temporaire sous
  `apps/go-api/cmd/perimetre_probe_tmp/` SUPPRIMÉE avant le commit, ouverture
  `access_mode=read_only`, `SET threads=2`, `SET memory_limit='512MB'` (réglages de la prod).
- Commandes `go` une à la fois, depuis `apps/go-api` du worktree, `CGO_ENABLED=1`,
  `GOCACHE=C:\Users\Guillaume\Projects\LevelUp-wt-perf-perimetre\.gocache-perimetre`, `CC` de
  l'environnement (gcc winlibs — ne pas le forcer), lint avec
  `GOLANGCI_LINT_CACHE=C:\Users\Guillaume\Projects\LevelUp-wt-perf-perimetre\.golangci-cache`.
- Interdits : push, `git add -A`, `git stash`, `--no-verify`, `--force`, commit sur une autre
  branche, écriture dans `.ai/thought_log.md` (le superviseur l'écrit depuis le rapport).
- Invariants intouchables : écritures per-match via `persist.BatchBuilder` ; tables append-only lues
  par leurs vues `_latest` ; `PathResolver` ; capabilities (jamais `slug == ...`) ; logs
  `slog.*Context` ; aucune string UI en dur ; fichiers ≤ 500 L et fonctions ≤ 80 L (dette gelée).
- Un test Go renommé ou supprimé : retirer sa paire (Package, Test) de
  `.ai/baselines/tests_pre_migration.jsonl` dans le même commit + paragraphe daté dans
  `scripts/check_test_baseline.sh` (leçon L5b).
- `go test ./internal/api/...` recrée `data/titles/halo_5/warehouse/metadata.duckdb` et des
  rasters sous `data/cache/replays` DANS le worktree : les supprimer avant chaque commit.
- Clôture d'étape = gate vert (commandes exactes) + items statués (`[x]` / `[~]` réf /
  `[!]` justifié) + section de l'étape mise à jour ici (items + journal) + commit(s) préfixé(s)
  `perf(perimetre-P<n>)` + rapport au superviseur (fait / non fait et pourquoi / découvertes /
  mutations jouées, avec leur résultat).
- Reprise de session : ce fichier fait foi ; reprendre à la première case non statuée de
  l'étape courante ; `git log --oneline feat/v75..feat/perf-perimetre`.

## 2. P1 — ADR 0036 « Page reads are scoped » (docs, EN-only)

Objectif : graver en invariants nommés ce que la campagne du 2026-09-23 a établi, pour que la
prochaine lecture de page ne rejoue pas l'historique entier. Les garde-rails existants DEVIENNENT
les invariants de l'ADR (chacun cité par chemin et nom de test, vérifiés sur pièces).

Décisions tranchées :
- D1.1 Fichier `docs/adr/0036-page-reads-are-scoped.md`, EN seul (politique : ADR = EN-only),
  statut Accepted (2026-09-26), gabarit des ADR 0033-0035 (Status, Branch, Relates to, Context,
  Decision, invariants numérotés, Consequences, Exceptions, Guardrails).
- D1.2 Invariants (un par règle, chacun avec son garde-rail) :
  I1 aucune lecture de page n'évalue `v_gamertag_lookup` : les noms viennent de l'annuaire de la
  lecture (`platform/duckdb/squad_repo_annuaire.go`, cascade `analysis.AnnuaireGamertags`) —
  `annuaire_ratchet_test.go` (`TestLecturesDeLaVueDesNoms_Ratchet`), table qui ne fait que
  descendre ; I2 une lecture d'une vue `_latest` dans le chemin d'une requête lie son périmètre
  SOUS la fenêtre (la fenêtre partitionne par `match_id` : seul `match_id` se pousse) —
  `tactical_repo_fenetres_test.go`, `weapon_range_repo_fenetres_test.go` (EXPLAIN ANALYZE) ;
  I3 une donnée relue d'une requête à l'autre passe par un cache invalidé au sync, jamais alimenté
  par un chargement dégradé — `archlint/player_read_cache_invalidation_test.go` + le test P0 de
  L9-go ; I4 un chargement par requête, jamais N (chargement partagé par requête de l'Escouade) —
  test(s) de L2 / L9-go à citer ; I5 le sync en régime stationnaire ne prend aucun écrivain quand
  rien n'est nouveau — `lusr_watermark_guardrail_test.go` ; I6 chaque lecture de page déclare ses
  sections de durée (`internal/observability/timing`, ligne `http_timings`,
  `LEVELUP_SLOW_REQUEST_MS`) ; I7 les bornes de ressources DuckDB se lisent à l'ouverture de chaque
  connexion (`TestResourceLimits_EnvSetAfterPackageInitIsHonored`).
- D1.3 Section Exceptions : chaque lecture encore hors invariant, avec son coût mesuré et ce qui
  la retire — les lectures de la vue figées par le ratchet (vue match, Relations : lot A de ce
  plan ; Explorer, Médias, classement mondial, killcollector : restent) ; la fenêtre du kill-feed
  sur tout l'historique (Carrière Q26 / Q27, Relations Q28 : lot B, après la mesure prod) ;
  `pages/home` et weapon_records (lot B). Chiffres repris de l'état des lieux §6 et du journal L7,
  jamais inventés.
- D1.4 `CLAUDE.md` : ligne `0036` ajoutée à la liste des ADR (même registre que les autres).
  En-têtes des garde-rails I1 à I7 : UNE ligne de commentaire « Invariant In de l'ADR 0036 »
  (commentaires seuls, aucun code).

Périmètre : `docs/adr/0036-page-reads-are-scoped.md` (créé), `CLAUDE.md`, les fichiers de test
garde-rails cités en D1.2 (commentaire d'en-tête seulement).

Items :
- [x] P1.1 ADR écrit (D1.1 à D1.3), chaque chemin et nom de test cité vérifié existant —
      `docs/adr/0036-page-reads-are-scoped.md` (328 L, EN seul) : Status / Branch / Relates to
      (0026, 0016, 0013, 0024, 0035) / Context (protocole, tableau avant-après §6 de l'état des
      lieux, chronos SQL, quatre formes de coût) / Decision (I1 à I7, chacun son garde-rail) /
      Consequences / Exceptions (état au 2026-09-26) / Guardrails (ce que chaque test bloque
      réellement et ce qu'il ne voit pas) / Alternatives rejetées par la mesure ; 47 chemins
      cités + 5 liens d'ADR, tous existants (hors routes et branches), 20 noms de test trouvés
      par `grep "func <Nom>("`, 40 autres identifiants trouvés ; aucun mot français hors `code`
- [x] P1.2 `CLAUDE.md` (D1.4) — entrée `0036` **lectures par périmètre** en fin de liste des
      ADR, trois lignes, même registre
- [x] P1.3 en-têtes des garde-rails (D1.4) — une ligne « Invariant In de l'ADR 0036
      (docs/adr/0036-page-reads-are-scoped.md). » à la fin du commentaire d'en-tête de 12
      fichiers (I1 : `annuaire_ratchet_test.go` ; I2 : `tactical_repo_fenetres_test.go`,
      `weapon_range_repo_fenetres_test.go` ; I3 : `archlint/player_read_cache_invalidation_test.go`,
      `player_read_cache_test.go` ; I4 : `teammates_service_loads_test.go`,
      `career_service_friends_test.go` ; I5 : `skill_v2_watermark_test.go`,
      `lusr_watermark_guardrail_test.go` ; I6 : `middleware/slog_logger_test.go`,
      `career_repo_annuaire_test.go` ; I7 : `db_resource_limits_env_test.go`) ; +1 ligne de
      commentaire par fichier, aucun code

Gate P1 : pour chaque chemin cité dans l'ADR, `test -e` vrai ; pour chaque nom de test cité,
`grep -rn "func <Nom>(" apps/go-api` trouve une ligne ; `gofmt -l` vide sur les fichiers Go
touchés ; `go vet` des paquets touchés ; `git diff --stat` = fichiers du périmètre seulement ;
aucun mot français dans l'ADR (relecture).

Journal P1 (2026-09-26, exécuteur Opus, worktree `LevelUp-wt-perf-perimetre`, base `28b972542`) :

- Garde-rails relus sur pièces (fichier ouvert, test lu), ce qu'ils bloquent RÉELLEMENT :
  I1 `TestLecturesDeLaVueDesNoms_Ratchet` compte l'identifiant nu `v_gamertag_lookup` dans les
  littéraux de chaîne de tout `internal/` hors tests, fichier par fichier (table qui ne fait que
  descendre) — il ne voit pas une nouvelle lecture de page qui APPELLE un lecteur resté dans la
  table (ex. `ResolveGamertags`) ; I2 les deux tests EXPLAIN ANALYZE ne couvrent que sept lectures
  (tactique x4, portée x3) ; I3 le ratchet archlint ne vérifie que trois points d'appel nommés
  (`runPostSyncPipeline`, `RecomputeIsWithFriends`, `SetExclusion`), le test P0 de L9-go est
  `TestPlayerReadCache_DegradedLoadNotStored` (+ ses deux voisins du même fichier) ; I7 relu tel
  que décrit (3 / 300MB posés après l'init).
- Écarts au plan, et pourquoi : (1) I5 — `lusr_watermark_guardrail_test.go` (cité par D1.2)
  interdit une COPIE du prédicat « déjà traité » et fige la frontière ≤, il ne fige PAS le zéro
  écrivain ; le garde-rail réel du zéro écrivain est `skill_v2_watermark_test.go`
  (`TestLUSRV2Shadow_Stationary_NoWriterAndOneInfoLine`, `..._NoCandidate_NoWriterAndOneInfoLine`,
  tag `cgo`) : les deux sont cités et portent l'en-tête I5 ; (2) I4 — tests trouvés : L2
  `teammates_service_loads_test.go` (`TestGetPage_LitLesEvenementsDImpactUneSeuleFois`,
  `TestGetPage_UnLoadForParMembre`, `TestGetPage_DeuxRequetesDeuxLectures`) et L9-go
  `career_service_friends_test.go` (`TestCareerService_ResolveFriendXUIDs_RegistreDAbordPuisUneLecture`) ;
  ils figent l'Escouade et les amis de la Carrière seulement (écrit dans l'ADR) ; (3) I6 — D1.2 ne
  nomme aucun test : cités `slog_logger_test.go` (`TestSlogLogger_TimingsLoggedForSlowRequest`,
  `TestSlogLogger_SlowSuccessLoggedAtInfo`) et `career_repo_annuaire_test.go`
  (`TestCareerRepo_Annuaire_SectionsDeDuree`, tag `integration`) ; l'OBLIGATION de déclarer ses
  sections n'est gardée par aucun test : écrit « not yet guarded » dans l'ADR (découverte (1)) ;
  (4) Exceptions : au-delà de la liste de D1.3, et pour tenir son « chaque lecture encore hors
  invariant », ajoutés sans chiffre inventé : `MortsParCarte` (découverte (5) de L5a, revérifiée
  dans `tactical_repo_morts_par_carte.go` : liste posée sur `mr` seulement ; non mesuré, non
  affecté), la part fenêtre du kill-feed de Q23b (~0,9 s, que le lot A laisse en place d'après
  DA.6 ; non affectée), les écrivains hors points d'invalidation (sync Halo 5, post-import
  OpenSpartan, processus CLI : découvertes (3) (4) et lecture (d) de L5b ; TTL 60 s seulement) ;
  (5) `TestCareerRepo_Annuaire_EcartNomme_NomHorsHistorique` est cité sous I1 comme modèle d'écart
  nommé, pas comme garde-rail (son fichier porte l'en-tête I6) ; le helper
  `exigerFenetresBornees` est cité par son nom, son fichier (`fenetres_perimetre_helpers_test.go`,
  outil, pas garde-rail) ne porte pas d'en-tête ; `sync/no_art_patterns_test.go` cité dans
  Consequences (invariants anti-ART intacts), sans en-tête ; (6) entrée `CLAUDE.md` en trois
  lignes (maximum de la consigne).
- Chiffres : tous repris de l'état des lieux (§1.3, §1.4, §2, §6), du handoff (§2, §4, §6) ou des
  journaux L2, L5a, L5b, L6, L7, L9-go et C.4 du plan clos ; format anglais (point décimal,
  séparateur de milliers « , ») sans arrondi ; « 10,7 s et 10,1 s » de la Carrière datés de la
  mesure intermédiaire (non captés le matin).
- Gate (depuis `apps/go-api`, `CGO_ENABLED=1`, GOCACHE privé, une commande `go` à la fois) :
  `gofmt -l` sur les 12 fichiers Go touchés : vide ; `go vet ./internal/platform/duckdb/
  ./internal/archlint/ ./internal/service/ ./internal/service/teammates/ ./internal/sync/skill/
  ./internal/api/middleware/` : 0 ; `go vet -tags=integration ./internal/platform/duckdb/` : 0
  (fichier `career_repo_annuaire_test.go` sous tag) ; chemins cités : tous présents (`test -e`,
  préfixe `apps/go-api/internal/` hors `docs/` et `.ai/`) ; 20 noms de test : chacun trouvé par
  `grep -rn "func <Nom>(" apps/go-api` ; relecture : aucun mot français hors `code` (grep des
  accents et d'une liste de mots outils sur le texte hors backticks : 0) ; `git diff --stat` = 14
  fichiers du périmètre (ADR, `CLAUDE.md`, 12 en-têtes) + ce plan. En plus, non exigé :
  `go test -count=1` des sept garde-rails du paquet duckdb cités (ratchet I1, fenêtres I2, cache
  I3 x3, bornes I7) : PASS.

## 3. P2 — Lot A : vue match et Relations sans la vue des noms (Go)

Objectif mesuré au lot L7 (copie, 2 threads / 512 Mo) : vue match Q12 1,7-2,2 s (3-6 ms sans la
jointure), Q21 2,2 s, Q23 2,2 s, Q23b 3,6 s (dont ~0,9 s de fenêtre du kill-feed), plus
`ResolveGamertags` 2,1 s : de l'ordre de 10 s de vue des noms par ouverture. Relations Q28
3,4-4,0 s (scopé 30 matchs : 2,4-2,7 s), heatmap Q29 2,3 s. Cible : plus aucune évaluation de la
vue dans ces lectures.

Décisions tranchées :
- DA.1 Lectures du lot (et seulement elles) : Q12 tableau de score et Q21 événements
  (`platform/duckdb/queries_match.go`), Q23 et Q23b (`queries_match_detail.go`) et leurs lecteurs
  (`match_view_repo_scoreboard.go`, `match_view_repo_extras.go`, et tout autre lecteur de ces
  gabarits trouvé par grep) ; `GamertagRepo.ResolveGamertags` (`gamertag_repo.go`, port
  `GamertagResolver` de `internal/port/services.go`, appelant
  `service/match_events_service.go`) ; Relations Q28 et Q28 scopé
  (`queries_career_encounters.go`) et la heatmap Q29 (`queries_relations_moments.go`) avec leurs
  lecteurs (`relations_repo.go`). Restent hors lot et dans le ratchet : Explorer, Médias,
  classement mondial, killcollector.
- DA.2 Les lignes se nomment par l'annuaire existant (`nommerLignes` / `annuaireDeLecture`,
  `squad_repo_annuaire.go`), UNE cascade (`analysis.AnnuaireGamertags.Resolve`), aucune copie.
  Matchs de la lecture : le match ouvert (vue match) ; l'historique du joueur
  (`QMatchsDuJoueurTpl`, comme L7) ou les matchs du périmètre scopé (Relations).
- DA.3 Parité des noms — le repli qui la tient. L7 a mesuré qu'un annuaire restreint aux matchs de
  la lecture perd des noms que la vue trouve AILLEURS : 11 couples (match, joueur) sur 93 636
  (vue match) passeraient à « Joueur #### », 2 lignes servies de Nuzzles (Relations). Mécanisme :
  un mode « portée base » de l'annuaire (option de `lectureANommer`, même cascade, activé par les
  lectures de CE lot seulement) : niveaux alias et participants lus sur TOUTE la base, filtrés par
  les xuids de la lecture (prédicats poussés sur les TABLES — c'est la sémantique exacte de la vue,
  MAX sur toute la base) ; niveau kill-feed d'abord sur les matchs de la lecture (la fenêtre
  `_latest` ne pousse que `match_id`), puis sur toute la base pour les SEULS xuids encore sans nom.
  Escouade et Carrière gardent leur mode actuel (parité mesurée et acceptée en L2 / L7 ; les
  basculer = hors périmètre, à consigner).
- DA.4 Écarts acceptables avec la vue, et seulement s'ils sont NOMMÉS et COMPTÉS sur la copie :
  (i) un xuid qu'aucune source ne connaît reçoit le libellé masqué là où la vue rendait NULL
  (Q21 affichait alors le xuid brut ; la vue promet « jamais de xuid brut ») ; (ii) un bot absent
  de toutes les sources reçoit son nom de bot ; (iii) un xuid que seul le kill-feed nomme et dont
  le nom varie reçoit le MAX des matchs de la lecture. Tout NOM remplacé par un libellé masqué ou
  un xuid brut, sur une ligne servie OU sur un couple de la copie, est un ROUGE.
- DA.5 `ResolveGamertags` reçoit le match : `ResolveGamertags(ctx, matchID string, xuids
  []string)` ; `MatchEventsService.enrichGamertags` passe `tl.MatchID` ; doubles de test mis à
  jour. La carte ne contient que les xuids que la cascade NOMME (bot, alias, participant,
  kill-feed) ; les autres restent absents (le front masque, contrat actuel du port). Un xuid connu
  d'une source mais sans nom non vide, que la vue rendait masqué côté serveur, devient absent :
  acceptable si le libellé affiché par le front (`displayPlayerName`) est identique à
  `analysis.MaskedXuidLabel` — le vérifier sur pièces ; sinon le rendre masqué côté serveur.
- DA.6 Budgets sur la copie (2 threads / 512 Mo, JGtm, son dernier match et 20 matchs au hasard) :
  Q12, Q21, Q23, `ResolveGamertags` ≤ 100 ms chacun, annuaire compris ; Q23b ≤ sa fenêtre du
  kill-feed + 100 ms ; Relations Q28 ≤ Q28 sans la jointure + 300 ms, Q29 ≤ 300 ms. Nuzzles
  (7 190 matchs, 2 973 joueurs récurrents sans aucun nom) mesuré et rapporté. Si un budget ne
  tient pas AVEC la parité de DA.3 / DA.4 : arrêt propre, les deux chiffres au rapport (avec et
  sans le repli), aucun choix silencieux.
- DA.7 Sections de durée (`internal/observability/timing`) sur chaque lecture et son annuaire,
  comme L7 (`<lecture>` et `<lecture>_annuaire`) ; test des sections.
- DA.8 Ratchet `annuaire_ratchet_test.go` : la table descend (queries_match.go 3 -> 0,
  queries_match_detail.go 2 -> 0, gamertag_repo.go 1 -> 0, queries_career_encounters.go 2 -> 0,
  queries_relations_moments.go 1 -> 0, sauf occurrence restante justifiée et datée) ; en-têtes de
  `squad_repo_annuaire.go` (lecteurs, mesures du lot A) et des fichiers de requêtes mis à jour.
- DA.9 L'ordre des ex aequo n'entre pas dans la parité (heatmap Q29 non déterministe : découverte
  (7) de L7, hors lot) : comparer les ensembles triés, noms et compteurs.

Périmètre : `internal/platform/duckdb/{queries_match.go, queries_match_detail.go,
match_view_repo_scoreboard.go, match_view_repo_extras.go, gamertag_repo.go,
queries_career_encounters.go, relations_repo.go, queries_relations_moments.go,
squad_repo_annuaire.go, annuaire_ratchet_test.go}` et leurs tests (nouveaux fichiers de test
autorisés, ex. `match_view_repo_annuaire_test.go`, `relations_repo_annuaire_test.go`) ;
`internal/analysis/identity_annuaire.go` (+ test) si le mode portée base exige un gabarit SQL ;
`internal/port/services.go` (signature du port) ; `internal/service/match_events_service.go` et
ses tests ; tout double de test de `GamertagResolver` trouvé par grep ; `.ai/baselines/
tests_pre_migration.jsonl` et `scripts/check_test_baseline.sh` seulement si un test est renommé
ou supprimé.

Items :
- [ ] A.1 mode « portée base » de l'annuaire (DA.3) + tests unitaires et d'intégration contre la
      VRAIE vue (un niveau de la cascade par xuid, dont un xuid nommé hors de la lecture)
- [ ] A.2 vue match : Q12, Q21, Q23, Q23b sans la vue, nommées par l'annuaire du match ; parité
      (DA.4) sur TOUS les matchs de la copie pour Q12 / Q21, sur ≥ 200 matchs pour Q23 / Q23b ;
      chrono avant / après (DA.6)
- [ ] A.3 `ResolveGamertags` avec le match (DA.5) : port, service, doubles, parité sur les
      événements de ≥ 200 matchs de la copie, chrono
- [ ] A.4 Relations : Q28, Q28 scopé, heatmap Q29 sans la vue ; parité sur les cinq joueurs suivis
      (tous les couples joueur / croisé, pas seulement les lignes servies) ; chrono
- [ ] A.5 sections de durée (DA.7) + ratchet et en-têtes (DA.8)
- [ ] A.6 mutations jouées (au moins : jointure réintroduite -> ratchet rouge ; repli portée base
      retiré -> test du xuid nommé hors lecture rouge ; match non passé à `ResolveGamertags` ->
      rouge ; une section de durée retirée -> rouge), chacune rouge puis restaurée

Gate P2 (depuis `apps/go-api`, une commande à la fois) : `gofmt -l ./internal ./cmd` vide ;
`go build ./...` ; `go vet ./...` ; `go test ./internal/service/... ./internal/platform/duckdb/...
./internal/analysis/... ./internal/api/... ./internal/archlint/... ./internal/port/...` ;
`go test -tags=integration -p 1 ./internal/platform/duckdb/...` ; les garde-rails de
`internal/sync` (`go test ./internal/sync/ -run 'NoArt|Legacy|Sentinel'`) ;
`golangci-lint run --new-from-rev=ea5682373 ./...` 0 issue (avec et sans `--build-tags=integration`) ;
`cmd/perimetre_probe_tmp/` absent ; `git status` sans artefact sous `data/`.

Journal P2 : (vide)

## 4. P3 — Docs de release 7.5.0 : campagne perf, lot A, ADR 0036 (EN + FR)

Décisions tranchées :
- D3.1 `docs/CHANGELOG.md` et `docs/FR/CHANGELOG.md`, bloc 7.5.0 : UNE entrée « Loading
  performance » (Changed) — chiffres avant / après du tableau §2 du handoff (dev local, même
  protocole) et du lot A (journal P2), mécanismes en une phrase chacun ; l'en-tête du bloc passe
  de « Four ADRs » à cinq avec la ligne 0036 ; une entrée Ops si l'ADR ou le lot en exige une
  (a priori aucune : ni migration ni variable nouvelle — `LEVELUP_SLOW_REQUEST_MS` et
  `LEVELUP_DUCKDB_THREADS` / `_MEMORY_LIMIT` à vérifier dans le bloc Ops existant, les y ajouter si
  absentes).
- D3.2 Notes de version in-app `docs/RELEASE_NOTES.md` et `docs/FR/RELEASE_NOTES.md` : UNE puce
  « chargements » dans le bloc v7.5, registre produit (sans nom de requête ni de table), FR sans
  anglicisme ; compte de puces vérifié au parseur (`parseReleaseNotes.ts`), parité EN / FR.

Périmètre : les quatre fichiers ci-dessus.

Items :
- [ ] P3.1 changelog EN + FR (D3.1)
- [ ] P3.2 notes de version EN + FR (D3.2)

Gate P3 : parité des puces EN / FR du bloc v7.5 (compte au parseur ou au test vitest du parseur
s'il existe) ; chaque chiffre cité retrouvé dans le handoff ou le journal P2 ; `git diff --stat`
= périmètre.

Journal P3 : (vide)

## 5. Clôture (superviseur)

- [ ] C.1 gates rejoués par le superviseur sur la tête de la branche (P2 complet + P1 / P3)
- [ ] C.2 revue adversariale du diff `feat/v75..feat/perf-perimetre` (lentilles : parité des noms
      et contrat du port ; SQL / périmètre sous les fenêtres) — skill `adversarial-review`
- [ ] C.3 push de `feat/perf-perimetre`, CI verte au niveau job
- [ ] C.4 mesure en réel après fusion : l'utilisateur ouvre une vue match et la page Relations ;
      durées lues dans `logs/http.log` et `http_timings` (le superviseur n'ouvre pas de navigateur)
- [ ] C.5 go utilisateur, fusion `--no-ff` dans `feat/v75`, push, retrait du worktree
- [ ] C.6 handoff mis à jour (§7 décisions : ADR et lot A faits) ; entrée thought_log

## 6. Découvertes (à consigner, pas à traiter)

- (P1, 2026-09-26) (1) I6 n'est pas gardé : aucun test n'échoue quand une nouvelle lecture de page
  ne déclare aucune section de durée (le middleware et quelques lectures seulement sont figés) —
  règle de revue tant qu'un ratchet n'existe pas. (2) I4 n'est figé que pour l'Escouade (Q32,
  `LoadFor`) et les amis de la Carrière ; rien de générique ne voit une lecture répétée dans une
  autre page. (3) Le ratchet de la vue (I1) compte des littéraux : une nouvelle lecture de page qui
  appelle un lecteur encore dans la table (`ResolveGamertags`, `loadMatchLobbies`,
  `GetStatLeaderboard`, `ResolveXUIDByGamertag`) passe sans le faire rougir. (4) Les tests de
  fenêtres (I2) ne couvrent que sept lectures ; `TacticalRepo.MortsParCarte` pose toujours sa liste
  sur `mr` seulement (découverte (5) de L5a, revérifiée ce jour), coût non mesuré. (5) Lien cassé
  dans l'ADR 0033 : `[ADR 0008](0008-title-path-isolation.md)` alors que le fichier est
  `0008-db-schema-multi-title-and-xuid-global.md`. (6) Ce plan, §1 : « Toute autre modification =
  découverte au §5 » alors que les découvertes sont au §6 (§5 = clôture).

## 7. Journal (superviseur)

- 2026-09-26 : plan écrit ; worktree `LevelUp-wt-perf-perimetre` et branche `feat/perf-perimetre`
  créés depuis `feat/v75` `ea5682373` (local en avance d'un commit sur `origin/feat/v75`
  `378509f5e` : suppression de `INVOUT.csv`, non poussée).
