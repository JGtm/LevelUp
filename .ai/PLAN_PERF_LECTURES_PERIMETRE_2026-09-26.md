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
- DA.10 (superviseur, 2026-09-26, après le premier journal P2) : parité exacte gardée, mais la jambe
  kill-feed « toute la base » ne relit plus la fenêtre `_latest` entière. Deux pas : (1) LOCALISER
  les matchs candidats dans la table BRUTE `match_kill_events` (et `killer_victim_pairs`), filtrés par
  les xuids restants (tueur OU victime) avec un gamertag non vide — lecture qui ne rend que des
  `match_id`, jamais une valeur (toute version d'une ligne qui a porté le xuid désigne son match :
  sur-ensemble des matchs où `_latest` le montre) ; (2) LIRE la jambe kill-feed de portée « matchs
  de la lecture » existante sur ces seuls candidats. Un seul gabarit, aucune copie de la cascade.
  Garde-rail de lecture brute : une entrée d'allowlist datée pour ce seul site (ou le dire s'il ne
  voit pas le fichier) ; test d'intégration « version ancienne ≠ `_latest` », rouge si le pas 2 lit
  la brute ; ADR 0036 : paragraphe « locating read » avec renvoi à l'ADR 0026. Si le pas 1 dépasse
  500 ms sur Relations Nuzzles : essayer une liste liée en un seul paramètre, sinon arrêt propre.

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
- [x] A.1 mode « portée base » de l'annuaire (DA.3) + tests unitaires et d'intégration contre la
      VRAIE vue (un niveau de la cascade par xuid, dont un xuid nommé hors de la lecture) —
      `lectureANommer.porteeBase` + `nommerLignesPorteeBase` (`squad_repo_annuaire.go`, même
      `nommerLignesSelon` que `nommerLignes`, Escouade / Carrière / Comparer inchangés) ; gabarits
      `AnnuaireNomsBaseSQL` / `AnnuaireKillFeedBaseSQL` et `AnnuaireGamertags.Nomme`
      (`analysis/identity_annuaire.go`, un gabarit par niveau, DDL de la vue intact) ; tests
      `TestAnnuaireGamertags_Nomme`, `TestAnnuaireSQL_PorteeBase` et `match_view_repo_annuaire_test.go`
      (x_ailleurs : participant nommé sur un autre match ; x_kfailleurs : nommé par le seul kill-feed
      d'un autre match)
- [x] A.2 vue match : Q12, Q21, Q23, Q23b sans la vue, nommées par l'annuaire du match ; parité
      (DA.4) sur TOUS les matchs de la copie pour Q12 / Q21, sur ≥ 200 matchs pour Q23 / Q23b ;
      chrono avant / après (DA.6) — Q12 et Q23 nommées en Go (`match_view_repo_scoreboard.go`,
      `match_view_repo_noms.go`, extrait de `match_view_repo_extras.go` gelé), Q21 aussi (events à
      xuid seulement), Q23b sans colonne de nom (sa jointure servait une colonne jamais rendue) ;
      Q12 en ordre total (`p.xuid`) ; parité et chrono au journal (budgets JGtm tenus pour
      l'annuaire ; Q23 dépasse 100 ms sur 1 à 8 mesures sur 42 sous charge, par sa propre requête ;
      coût du repli quand il part : journal)
- [x] A.3 `ResolveGamertags` avec le match (DA.5) : port, service, doubles, parité sur les
      événements de ≥ 200 matchs de la copie, chrono — `ResolveGamertags(ctx, matchID, xuids)`,
      `enrichGamertags` passe `tl.MatchID`, double `fakeGTResolver` + `TestGetMatchEvents_ResolveurRecoitLeMatch` ;
      la carte ne porte que les xuids que `Nomme` accepte ; libellé du front vérifié
      (`maskedPlayerLabel` = « Joueur » + 4 derniers caractères = `analysis.MaskedXuidLabel`)
- [x] A.4 Relations : Q28, Q28 scopé, heatmap Q29 sans la vue ; parité sur les cinq joueurs suivis
      (tous les couples joueur / croisé, pas seulement les lignes servies) ; chrono — nommées sur le
      périmètre scopé ou l'historique (`matchsDuPerimetre`, `relations_repo.go`) ; la heatmap vit dans
      `relations_moments_repo.go` (lecteur réel de Q29, hors de la liste : écart consigné)
- [x] A.5 sections de durée (DA.7) + ratchet et en-têtes (DA.8) — sections `match_scoreboard`,
      `match_events`, `match_encounters` (+ `_annuaire` chacune), `match_encounter_stats`,
      `resolve_gamertags_annuaire`, `relations`, `relations_annuaire`, `relations_heatmap`,
      `relations_heatmap_annuaire` (`TestMatchView_Annuaire_SectionsDeDuree`) ; ratchet : les cinq
      fichiers sortent de la table (3, 2, 1, 2, 1 -> 0) ; en-têtes de `squad_repo_annuaire.go` et des
      fichiers de requêtes mis à jour
- [x] A.6 mutations jouées (au moins : jointure réintroduite -> ratchet rouge ; repli portée base
      retiré -> test du xuid nommé hors lecture rouge ; match non passé à `ResolveGamertags` ->
      rouge ; une section de durée retirée -> rouge), chacune rouge puis restaurée — cinq jouées,
      cinq rouges, restaurées au `cmp` (journal)

Gate P2 (depuis `apps/go-api`, une commande à la fois) : `gofmt -l ./internal ./cmd` vide ;
`go build ./...` ; `go vet ./...` ; `go test ./internal/service/... ./internal/platform/duckdb/...
./internal/analysis/... ./internal/api/... ./internal/archlint/... ./internal/port/...` ;
`go test -tags=integration -p 1 ./internal/platform/duckdb/...` ; les garde-rails de
`internal/sync` (`go test ./internal/sync/ -run 'NoArt|Legacy|Sentinel'`) ;
`golangci-lint run --new-from-rev=ea5682373 ./...` 0 issue (avec et sans `--build-tags=integration`) ;
`cmd/perimetre_probe_tmp/` absent ; `git status` sans artefact sous `data/`.

Journal P2 (2026-09-26, branche `feat/perf-perimetre` depuis 948523a36 ; commit du code f576df10e
A.1 à A.5, puis ce journal ; reprise d'un exécuteur interrompu : aucun commit de sa part, sa sonde
et ses sorties « avant » Q12 / Q21 réutilisées après vérification, cf. écart (8)) :

- Mesure : COPIE de `shared_matches_v2.duckdb` (1,3 Go, base du 2026-09-23 19:56, pas de `.wal`)
  dans le scratchpad, sonde temporaire `cmd/perimetre_probe_tmp/` (jamais commitée, supprimée)
  appelant les VRAIS repos, `access_mode=read_only`, 2 threads, 512 Mo ; binaire « avant » construit
  sur l'arbre exporté de 948523a36 (`diff -r` de `apps/go-api` contre le worktree : identique),
  « après » sur le worktree, « sans repli » = le worktree avec le repli toute la base neutralisé le
  temps du build (restauré au `cmp`). MACHINE PARTAGÉE ET CHARGÉE (une autre session tournait des
  tests de fuzz) : `SELECT count(*) FROM v_gamertag_lookup` a coûté 3,0 à 8,6 s selon les passes
  (1,7-1,8 s au lot L7) ; les chiffres absolus sont bruités, les ordres de grandeur non.
- Chrono vue match, avant (JGtm, dernier match + 10 au hasard, 1 tour) : Q12 médiane 3,48 s
  (2,96-7,40), Q21 3,41 s, Q23 3,56 s, Q23b 5,65 s (dont fenêtre du kill-feed seule 1,68 s), RG
  3,31 s. Nuzzles (6 matchs) : Q12 4,22 s, Q21 4,25 s, Q23 4,67 s, Q23b 6,74 s (fenêtre 2,32 s).
- Chrono vue match, après (JGtm, dernier match + 20 au hasard, trois passes de deux tours, 42
  mesures par lecture et par passe) : Q12 médiane 16-33 ms, max 36-71 ms ; Q21 médiane 7-21 ms, max
  27-53 ms ; RG médiane 5-13 ms, max 12-41 ms ; Q23 médiane 46-89 ms, max 65-156 ms ; l'annuaire du
  match ≤ 9 ms sur TOUTES les mesures (aucun repli toute la base dans l'échantillon). BUDGET DA.6 :
  tenu pour Q12, Q21, RG ; Q23 dépasse 100 ms sur 1, 1 puis 8 mesures sur 42 (machine chargée), par
  sa PROPRE requête (historique commun, 40-99 ms), pas par l'annuaire. Q23b (plus aucun nom) : médiane
  1,73-4,11 s contre 1,65-4,56 s pour sa fenêtre du kill-feed seule, mesurée juste après ; écart moyen
  par match +182 ms dans la passe la plus chargée, dans le bruit de deux mesures de plusieurs
  secondes — non concluant au-delà ; le reste de Q23b est la fenêtre (hors lot).
- LE REPLI « TOUTE LA BASE » (DA.3), quand il part : la jambe kill-feed de la vue sans borne de match
  évalue la fenêtre `_latest` du journal canonique entière (3,95 M lignes brutes) : 3 à 7 s par
  lecture (5,3 s au CLI, 12,2 s une fois sous charge, et Q23 a alors dépassé son délai de 20 s :
  « context deadline exceeded », section rencontres perdue pour ce match). Il part pour tout match dont
  un participant n'a ni alias, ni nom de participant dans TOUTE la base, ni nom au kill-feed du match :
  23 des 1 160 matchs de JGtm, 15 / 587 Chocoboflor, 14 / 1 275 Madina97294, 1 / 39 XxDaemonGamerxX,
  4 688 / 7 190 Nuzzles (31 426 couples, 25 064 xuids sur la base ; seuls 11 de ces couples sont
  nommés ailleurs). Nuzzles, après, 6 matchs : 4 déclenchent le repli, Q12 2,9-7,2 s et Q23 3,4-5,8 s
  (l'annuaire), Q21 et RG ≤ 40 ms. Sans le repli : Nuzzles Q12 médiane 31-44 ms (max 51-107), Q23
  38-46 ms ; JGtm Q12 22-34 ms, Q21 11-23 ms, Q23 72-86 ms, RG 7-10 ms.
- Chrono Relations, après avec repli (deux tours) : JGtm Q28 1,97-2,32 s dont Q28 seule 1,72-2,08 s
  et annuaire 241-244 ms (budget +300 ms tenu), Q28 scopé 30 matchs 132-134 ms (annuaire 4-6 ms), Q29
  78-80 ms (annuaire 30-31 ms, budget 300 ms tenu), Q29 scopé 21-22 ms. Autres joueurs : annuaire Q28
  Madina97294 282-311 ms, Chocoboflor 246-347 ms, XxDaemonGamerxX 8-11 ms (le premier passage
  kill-feed sur la liste liée des matchs de l'historique) ; aucun repli. Nuzzles : annuaire Q28
  3,8-4,9 s, Q29 4,7-5,9 s (repli : 2 973 récurrents sans aucun nom), scopés 7-10 ms. Sans le repli :
  Nuzzles annuaire Q28 684-920 ms, Q29 223-231 ms. Avant (1 tour, passe la plus chargée) : JGtm Q28
  14,1 s, scopé 14,9 s, Q29 20,4 s ; Nuzzles 7,4 / 5,2 / 5,2 / 5,5 s (passe de l'exécuteur précédent,
  moins chargée : JGtm Q28 8,2 s, Q29 5,0 s).
- Parité (DA.4), au code final, sorties comparées triées (DA.9) puis dans l'ordre : Q12 93 000 lignes
  (tous les matchs de la copie ayant une ligne servie, 9 088 / 9 170), Q21 1 311 594 events (tous les
  matchs), Q23 et Q23b 2 625 lignes chacune (279 couples match / joueur, 233 matchs distincts, 60
  matchs au hasard par joueur suivi), RG 1 853 xuids (les mêmes 279 appels ; 165 matchs portent des
  events), Relations 12 289 lignes (cinq joueurs, période entière et 30 matchs : TOUS les récurrents,
  Q28 n'a pas de LIMIT) et heatmap 1 265 lignes : ZÉRO écart de nom ou de compteur avec le repli.
  Écarts (i), (ii), (iii) : 0, 0, 0 sur la copie (aucun xuid d'event inconnu de toutes les sources,
  aucun bot hors sources, aucun nom de kill-feed variable touché) ; (i) épinglé par un test dès P2, (iii) seulement en A.12
  (le premier journal disait à tort « épinglés par les tests »), (ii) par le test Escouade. Aucun nom
  perdu. Sans le repli : exactement les écarts de L7 — 11 couples Q12 (10 matchs, 8 xuids, ex.
  « Feelgood Joker » -> « Joueur 6576 » sur trois matchs) et 2 lignes Relations de Nuzzles, des ROUGES
  au sens de DA.4 ; Q21, Q23, RG, heatmap sans écart.
- Ordre : Q12 avait un ordre des ex aequo (équipe, rang) stable d'une exécution à l'autre ; sans la
  jointure il variait (689 matchs entre deux passes) -> départage `p.xuid` (conséquence de l'ADR 0036 :
  un ordre dont une réponse dépend est total) ; l'ordre des ex aequo diffère donc de l'avant sur 5 805
  matchs (contenu identique). Q21 : l'ordre des events de même milliseconde diffère de l'avant sur
  3 883 matchs ; il est désormais stable d'une passe à l'autre et suit l'ordre d'insertion (`id`) à 278
  lignes près, quand l'avant s'en écartait sur 260 036 lignes ; laissé non total (découverte (3)). Q23
  (ex aequo de `count_together`, découverte (5) de L7) : ordre différent, contenu identique.
- DA.5 vérifié sur pièces : `maskedPlayerLabel` (`apps/web/src/lib/players/displayName.ts`) rend
  « Joueur » + les 4 derniers caractères, comme `analysis.MaskedXuidLabel` : un xuid absent de la carte
  s'affiche comme la vue le rendait. Aucun lecteur web de `GET /matches/{match_id}/events` trouvé
  (découverte (6)).
- Mutations jouées (toutes rouges puis restaurées, `cmp` à l'appui) : jointure réintroduite dans Q12
  -> `TestLecturesDeLaVueDesNoms_Ratchet` rouge ; repli toute la base neutralisé -> cinq tests rouges
  (Q12, Q21, Q23, ResolveGamertags, Relations : x_kfailleurs) ; `ResolveGamertags(ctx, "", …)` dans le
  service -> `TestGetMatchEvents_ResolveurRecoitLeMatch` rouge ; section `relations_annuaire` retirée
  -> `TestMatchView_Annuaire_SectionsDeDuree` rouge ; Q12 nommée en portée de la lecture au lieu de la
  portée base -> `TestMatchView_Annuaire_Q12MemeNomsQueLaVue` rouge (x_ailleurs, x_kfailleurs).
- Écarts à la lettre, et pourquoi : (1) `match_view_repo_noms.go` (nouveau, non test) : Q21 et Q23
  extraites de `match_view_repo_extras.go`, gelé au-delà de 500 L (578 -> 490) ; (2)
  `relations_moments_repo.go`, hors de la liste : c'est le lecteur réel de Q29 ; (3) fixtures de tests
  existants adaptées au schéma réel (`match_participants.gamertag`, tables de l'annuaire) :
  `match_view_repo_meta_test.go` (attente de l'orphelin : NULL -> « Joueur 9999 », écart (i) ; nom du
  test gardé pour ne pas toucher la baseline, commentaire « nom historique »), `relations_repo_test.go`,
  `queries_relations_moments_timezone_test.go`, `gamertag_resolve_test.go` ; aucun test renommé ni
  supprimé : baseline JSONL inchangée ; (4) Q12 en ordre total (cf. Ordre) ; (5) un échec de Q21 est
  désormais journalisé (WARN) avant la dégradation en liste vide, qui était muette ; (6) un seul commit
  de code pour A.1 à A.5 : le ratchet et le fichier de tests d'intégration couvrent les cinq items à la
  fois, un commit intermédiaire aurait été rouge ; (7) le filtre du gate `NoArt|Legacy|Sentinel` ne voit
  pas `TestNoARTPatterns…` (casse) : rejoué aussi avec `NoART|NoRaw|Allowlist|Bulk|Interpolated|Legacy`,
  13 tests verts ; (8) parité « avant » Q12 / Q21 : sorties de l'exécuteur interrompu (même sonde, arbre
  vérifié), rejouées une fois (contenu et ordre identiques) ; Q23 / Q23b / RG / Relations « avant »
  refaites à 60 matchs par joueur.
- Dette : aucun fichier gelé ne grossit (`queries_match.go` 622 = 622, `queries_career_encounters.go`
  568 -> 565, `match_view_repo_extras.go` 578 -> 490) ; les autres sous 500 L ; aucune fonction
  nouvelle au-delà de 80 L (`scanScoreboardRow` 73 -> 72) ; golangci 0 issue.
- Gate (code f576df10e) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ;
  `go test ./internal/service/... ./internal/platform/duckdb/... ./internal/analysis/...
  ./internal/api/... ./internal/archlint/... ./internal/port/...` 0 (34 paquets ok, aucun `FAIL`) ;
  `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 0 (5 paquets ok) ; garde-rails
  `internal/sync` verts ; `golangci-lint run --new-from-rev=ea5682373 ./...` 0 issue, idem avec
  `--build-tags=integration` ; `cmd/perimetre_probe_tmp/` absent ; l'artefact
  `data/titles/halo_5/warehouse/metadata.duckdb` recréé par `./internal/api/...` retiré, `git status`
  sans artefact sous `data/`.
- DÉCISION À PRENDRE (superviseur) : DA.3 tient la parité (zéro nom perdu) et les budgets de
  l'échantillon JGtm ; hors échantillon, le repli toute la base coûte autant ou plus que la vue qu'il
  remplace, pour 2 % des matchs de JGtm et 65 % de ceux de Nuzzles (et ses Relations). Chiffres avec
  et sans le repli ci-dessus ; piste mesurée à titre d'information : découverte (1) de P2.
  -> TRANCHÉE : DA.10, ci-dessous.

- [x] A.10 (DA.10, 2026-09-26) repli toute la base en deux pas — `lireKillFeedBase` =
  `localiserKillFeed` (gabarit `analysis.AnnuaireKillFeedLocaliserSQL` : `WITH cherches(xuid) AS
  (VALUES …)`, quatre `SELECT match_id` en UNION sur `match_kill_events` et `killer_victim_pairs`,
  xuids liés UNE fois) puis `lireKillFeed` (la jambe de la vue, inchangée, `_latest` bornée par
  `match_id`) ; `AnnuaireKillFeedBaseSQL` supprimé (plus d'appelant) ; test
  `TestAnnuairePorteeBase_RepliLitLaDerniereVersion` ; ADR 0036 : paragraphe « Directory scope base
  and the locating read » sous I1.

Journal A.10 (2026-09-26, code sur f576df10e ; même copie, même protocole, sonde temporaire
supprimée) :

- Garde-rail de lecture brute : `TestNoRawAppendOnlyReads` (`internal/platform/duckdb/
  no_raw_rating_reads_test.go`, défaut ; il scanne `platform/duckdb`, `api`, `service`, `analysis`
  et connaît `match_kill_events` depuis le 2026-09-07) — absent de mon grep `internal/archlint` /
  `internal/sync`, trouvé ROUGE par le gate au premier passage (« internal\analysis\
  identity_annuaire.go »), ce qui vaut mutation « entrée d'allowlist retirée ». UNE entrée ajoutée,
  ligne 70 : `"identity_annuaire.go"`, datée du 2026-09-26, « lecture de localisation, sur-ensemble de
  matchs, aucune valeur lue ; valeurs par `_latest` » (allowlist 4 -> 5, commentaire daté). Limite :
  l'allowlist est par NOM DE FICHIER — une seconde lecture brute dans `identity_annuaire.go` passerait
  (découverte (10)). Le site est aussi épinglé par `TestAnnuaireSQL_PorteeBase` (la localisation ne
  projette que `match_id`) et le test de version ci-dessous. (Les garde-rails de `internal/sync` ne
  lisent que les mutations de `match_kill_events` : `TestNoMutationOnAppendOnlyStateTables`.)
- Test de version : sur mw1, une passe ancienne nomme x_version « ZzAncienNom » et x_disparu
  « ZzDisparu », la dernière passe nomme x_version « AaNouveauNom » et ne porte plus x_disparu ;
  Q12 et ResolveGamertags sur mv1 rendent « AaNouveauNom » et le libellé masqué / l'absence (les noms
  choisis pour que le MAX de la brute diffère de celui de `_latest`). Journal versionné du test :
  table brute à passes + vue `_latest` à la règle de la migration (`decode_pass` de la ligne la plus
  récente par match). Le harnais commun (`player_repos_test.go`, gelé au-delà de 500 L) n'est pas
  touché : la brute simulée (= la `_latest` à une version) est posée par `simulerJournalBrut` (appelée par `seedPorteeBase` et par
  `TestGamertagRepo_ResolveGamertags`, dont un xuid inconnu déclenche le repli).
- Mutations (rouges puis restaurées, `cmp`) : pas 2 lisant la table brute (même gabarit, `_latest`
  remplacée par la brute) -> `TestAnnuairePorteeBase_RepliLitLaDerniereVersion` rouge (Q12 :
  « ZzAncienNom », « ZzDisparu » ; ResolveGamertags idem) ; localisation neutralisée (aucun
  candidat) -> six tests rouges (Q12, Q21, Q23, ResolveGamertags, Relations, version).
- Chrono (2 threads, 512 Mo, machine toujours partagée ; vue seule 4,2 s) :
  vue match Nuzzles, les 6 mêmes matchs, deux tours : Q12 médiane 184-211 ms (max 341), Q23 184-186
  ms (max 336), Q21 1 ms (max 12), RG 0 ms (max 8) ; l'annuaire des 4 matchs à repli 151-325 ms
  (ancien repli : 2,8-7,2 s ; avant le lot : Q12 4,2 s, Q23 4,7 s de médiane) ; les 2 autres ≤ 8 ms.
  JGtm, ses 23 matchs à repli, deux tours (46 mesures par lecture) : Q12 médiane 18-28 ms (max
  212), Q21 11-14 ms (max 177), RG 5-9 ms (max 186), Q23 177-281 ms (max 439 ; l'annuaire dépasse
  100 ms sur 45 mesures sur 46, max 360 ms) ; ancien repli, un tour : Q23 médiane 4,19 s (max
  15,2 s), Q12 max 6,2 s, Q21 max 4,7 s, et ResolveGamertags en ERREUR une fois (délai de 10 s
  dépassé). Le repli ne part pas pour toutes les lectures d'un même match : Q12 écarte les lignes
  toutes nulles, Q21 et RG ne voient que les xuids des events.
  Relations, deux tours : JGtm Q28 2,21-2,48 s dont annuaire 275-277 ms (budget +300 ms tenu), Q29
  90-94 ms, scopés 25-150 ms ; Madina97294 annuaire Q28 316-350 ms, Chocoboflor 237 ms, Xx 8-10 ms ;
  Nuzzles Q28 2,31-2,46 s dont annuaire 868-871 ms (ancien repli 3,8-4,9 s ; sans repli 684-920 ms),
  Q29 387-409 ms dont annuaire 327-345 ms (ancien 4,7-5,9 s), scopés 16-114 ms.
  Pas 1 seul (la localisation exacte, 2 973 xuids récurrents sans nom de Nuzzles, CLI, 3 exécutions) :
  215-219 ms -> 2 matchs candidats. Sous le seuil de 500 ms : la variante « un seul paramètre
  tableau » n'a pas été essayée (la liste VALUES est déjà liée une fois ; 2 973 et non 25 000
  xuids : 25 064 est le compte de toute la base, aucune lecture ne les cherche tous).
- BUDGET DA.6 : inchangé sur l'échantillon JGtm (aucun repli) ; quand le repli part, l'annuaire coûte
  150 à 360 ms, soit au-dessus des 100 ms d'une lecture de la vue match — la table brute (3,95 M
  lignes) est parcourue à chaque localisation (découverte (9)).
- Parité (au code final, même couverture que le premier balayage ; tables `_latest` et brute
  matérialisées en tables temporaires dans la sonde pour tenir le temps — même contenu) : Q12 93 000,
  Q21 1 311 594, Q23 / Q23b 2 625, RG 1 853, Relations 12 289, heatmap 1 265 : ZÉRO écart trié avec
  l'avant ; Q12 et Q21 identiques octet pour octet au balayage de f576df10e (ordre compris). Les 11
  couples Q12 (« Feelgood Joker », « madWasabii », « shake1179 », « BlockedChart3 »,
  « Symbolicdeth », « FUGMO », « Viridianvoid », « WrathfulAsp3213 ») et les 2 lignes Relations de
  Nuzzles gardent leur nom. Aucun nom perdu.
- Gate (code A.10, rejoué en entier) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ;
  `go vet ./...` 0 ; `go test ./internal/service/... ./internal/platform/duckdb/...
  ./internal/analysis/... ./internal/api/... ./internal/archlint/... ./internal/port/...` 0 (34
  paquets ok) ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 0 (5 paquets ok ;
  un premier passage rouge — `TestNoRawAppendOnlyReads` et `TestGamertagRepo_ResolveGamertags` — a
  conduit à l'entrée d'allowlist et à `simulerJournalBrut`) ; garde-rails `internal/sync`
  (`NoArt|NoART|NoRaw|Allowlist|Bulk|Interpolated|Legacy|Sentinel`, 14 tests) verts ;
  `golangci-lint run --new-from-rev=ea5682373 ./...` 0 issue, idem `--build-tags=integration` ;
  `cmd/perimetre_probe_tmp/` absent ; artefact `data/titles/halo_5/warehouse/metadata.duckdb`
  retiré. Dette : `squad_repo_annuaire.go` 378 L, `identity_annuaire.go` 176 L, aucune fonction
  au-delà de 80 L, `player_repos_test.go` (gelé) non touché.

- [x] A.11 (2026-09-26) correctifs de la revue adversariale du lot A — (1) couverture des deux
  branches VICTIME de la localisation (`AnnuaireKillFeedLocaliserSQL`, victime de
  `match_kill_events` et victime de `killer_victim_pairs`) : test
  `TestAnnuairePorteeBase_RepliNommeLesVictimes` ; (2) ADR 0036, section « Exceptions » mise à jour
  (découverte (11) de P2).

Journal A.11 (2026-09-26, sur 5ab3c13c7) :

- Test : sur mv2, x_vicmke et x_vickvp ne sont nommés par rien (ni alias, ni participant, ni
  kill-feed de mv2) ; hors de la lecture, x_vicmke n'est nommé QUE comme victime du journal canonique
  (mb3, « NomVicMKE ») et x_vickvp QUE comme victime de `killer_victim_pairs` (mb4, « NomVicKVP »).
  Q12 (vue match) et `ResolveGamertags` doivent rendre ces deux noms, et Q12 égale la vue.
- Mutations (rouges puis restaurées, `cmp`) : branche victime de `match_kill_events`
  (`identity_annuaire.go:168`, `victim_xuid` -> `feed_killer_xuid`) -> test rouge (x_vicmke
  « Joueur cmke », absent de la carte) ; branche victime de `killer_victim_pairs` (`:174`,
  `victim_xuid` -> `killer_xuid`) -> test rouge (x_vickvp « Joueur ckvp », absent) ; en plus, filtre
  de nom de chacune des deux branches cassé (`:169`, `:175`, `victim_gamertag IS NULL`) -> rouge
  deux fois.
- ADR 0036, « Exceptions (state on 2026-09-26) » : introduction au passé pour le lot A ; tableau I1
  réduit aux quatre lectures qui restent (Explorer, Médias, classement mondial, killcollector) ; une
  phrase liste les lectures retirées le 2026-09-26 et leur coût après (chiffres du journal P2 / A.10) ;
  le paragraphe « Lot A must keep the names » réécrit au passé avec le mécanisme retenu (portée base,
  localisation) et la parité chiffrée ; tableau I2 : Q28 requalifiée (vue retirée) avec sa mesure du
  lot A, fenêtre du kill-feed de Q23b ajoutée (environ 0,9 s au repos, 1,6 à 4,6 s de médiane sous
  charge, non attribuée), et le repli de localisation quand il part (150 à 360 ms, au-dessus du budget
  de 100 ms, deux passes sur la brute de 3,95 M lignes, piste « une passe unique » non mesurée).
- Gate (consigne A.11) : `gofmt -l` des fichiers touchés vide ; `go vet ./internal/analysis/
  ./internal/platform/duckdb/` 0 ; `go test ./internal/analysis/...` 0 ; `go test -tags=integration -p 1
  ./internal/platform/duckdb/...` 0 (5 paquets ok) ; `golangci-lint run --new-from-rev=ea5682373 ./...`
  0 issue, idem avec `--build-tags=integration`.

- [x] A.12 (2026-09-26) correctifs de la revue fraîche de la branche — trois constats P2, aucun
  défaut de comportement : (1) l'ADR 0036 (tableau Guardrails, ligne I1) citait `ResolveGamertags`
  comme lecteur « encore dans la table » du ratchet, que ce lot en a retiré ; (2) l'écart (iii) de
  DA.4 n'était épinglé par aucun test, alors que l'en-tête de `match_view_repo_annuaire_test.go` et le
  journal P2 l'affirmaient ; (3) commentaires rendus faux par le lot (découverte (4) de P2).

Journal A.12 (2026-09-26, sur 35e9766ff) :

- (1) ADR 0036 l. 347 : l'exemple devient le `ResolveXUIDByGamertag` de l'Explorer (vérifié au
  ratchet : `platform/duckdb/explorer_repo.go`, 1 occurrence permise) ; rien d'autre dans la phrase.
- (2) Test `TestAnnuairePorteeBase_EcartNomme_NomDeKillFeedVariable` : x_varie, sans alias ni nom de
  participant, nommé « NomA » au kill-feed `_latest` du match lu (mv3) et « NomB » à celui d'un autre
  match (mb5) ; la vue rend « NomB » (vérifié dans le test), Q12 et `ResolveGamertags` sur mv3 rendent
  « NomA » — l'écart (iii) admis. En-tête du fichier de tests corrigé : (i) et (iii) nommés avec leur
  test, (ii) renvoyé au test Escouade (`TestSquadRepo_Annuaire_BotHorsDeToutesLesSources`, même
  cascade) ; phrase du journal P2 corrigée. Mutation : étape « kill-feed des matchs de la lecture »
  sautée (`lireKillFeedDeLaLecture` appelée sans matchs de lecture, tout passe par le repli base) ->
  test rouge (Q12 et ResolveGamertags rendent « NomB ») ; restaurée, `cmp` 0.
- (3) Commentaires seuls : `domain/match_view_raw.go` (`EventRaw.Gamertag`) et `domain/match_view.go`
  (`ActorGamertag`) décrivent l'annuaire du match en portée base (`match_view_repo_noms.go`), le
  libellé masqué pour un xuid inconnu et le nom absent pour un event sans xuid ;
  `platform/duckdb/explorer_repo.go` : la phrase portait sur la clause des frags parfaits et restait
  vraie ; précisée (`perfectKillMedalInClause`, même jeton que Q12 et Q30). Fichiers gelés au-delà de
  500 L non grossis (650, 878, 572 lignes).
- Gate : `gofmt -l` des fichiers touchés vide ; `go vet ./internal/domain/ ./internal/platform/duckdb/`
  0 ; `go test ./internal/domain/...` 0 ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...`
  0 ; golangci `--new-from-rev=ea5682373` 0 issue avec et sans `--build-tags=integration` ; artefacts
  sous `data/` retirés.

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
- [x] P3.1 changelog EN + FR (D3.1) — `6606fd3c9` (entrée « Loading performance » /
      « Performance des chargements » en Changed, « Five ADRs » avec 0036, trois lignes Ops de
      réglage : `LEVELUP_SLOW_REQUEST_MS` 1000, `LEVELUP_DUCKDB_THREADS` 2,
      `LEVELUP_DUCKDB_MEMORY_LIMIT` 512MB) ; phrase du lot A complétée par le superviseur à la
      clôture avec les chiffres du journal P2 / A.10
- [x] P3.2 notes de version EN + FR (D3.2) — `6606fd3c9`, une puce « chargements » dans la
      section Escouade, sessions et progression ; bloc v7.5 68 -> 69 puces EN et FR

Gate P3 : parité des puces EN / FR du bloc v7.5 (compte au parseur ou au test vitest du parseur
s'il existe) ; chaque chiffre cité retrouvé dans le handoff ou le journal P2 ; `git diff --stat`
= périmètre.

Journal P3 (2026-09-26, exécuteur Opus effort moyen, lancé EN PARALLÈLE de P2 sur décision du
superviseur — écart à l'ordre strict du §1 : fichiers disjoints, aucune commande go, aucun
chiffre du lot A écrit avant sa clôture, marqueur `<!-- lot-A -->` remplacé ensuite) : puces
comptées selon la règle de `apps/web/src/features/help/parseReleaseNotes.ts` (ligne commençant
par `- ` entre deux lignes de version ; une ligne de suite est ignorée, d'où une puce sur une
ligne) : 68 -> 69 EN et FR. Chaque chiffre de l'entrée vient du §2 / §3 du handoff ; défauts Ops
lus dans `internal/api/middleware/slog_logger.go` et `internal/platform/duckdb/db.go:42-43`.
Chiffres du lot A (superviseur) : lectures de la vue match 3,3-3,6 s -> 5-89 ms en médiane
(journal P2, JGtm), heatmap Relations 5,0-20,4 s -> 78-94 ms (P2 et A.10), liste Relations
8,2-14,1 s -> environ 2 s (A.6 : 1,97-2,32 s au total ; A.10 : annuaire 275-277 ms), sur copie,
machine chargée — dit tel quel dans l'entrée.

## 5. Clôture (superviseur)

- [x] C.1 gates rejoués par le superviseur sur la tête de la branche (P2 complet + P1 / P3) — sur
      `5ab3c13c7` (PowerShell, GOCACHE privé, une commande à la fois) : gofmt vide, build 0, vet 0,
      tests service / duckdb / analysis / api / archlint / port 0, intégration `-p 1`
      platform/duckdb 0, garde-rails sync `NoART|Legacy|Sentinel` 0 (40 paquets ok) ; sur
      `35e9766ff` (après A.11) : `go test ./...` complet, 188 paquets ok, 2 rouges de seuil de temps hors du lot, verts rejoués seuls (journal, découverte (13))
- [x] C.2 revue adversariale du diff `feat/v75..feat/perf-perimetre` (lentilles : parité des noms
      et contrat du port ; SQL / périmètre sous les fenêtres) — skill `adversarial-review` — une
      ronde, deux relecteurs Opus effort élevé aveugles : relecteur A (parité des noms, contrat du
      port, anti-ART et lecture de localisation) 0 constat, 15 conditions vérifiées ; relecteur B
      (SQL et périmètre, couverture des tests) 0 défaut de comportement, 16 conditions vérifiées,
      1 constat P2 de couverture (branches « victime » de la localisation sans test) retenu et
      corrigé en A.11 (`TestAnnuairePorteeBase_RepliNommeLesVictimes`, 4 mutations rouges). Pas de
      ronde 2 : aucun P0 / P1.
- [!] C.3 push de `feat/perf-perimetre`, CI verte au niveau job — en attente de l'utilisateur
      (consigne du 2026-09-26 : rien vers `feat/v75` tant qu'il ne l'a pas dit ; le push de la
      branche lui est demandé)
- [!] C.4 mesure en réel après fusion : l'utilisateur ouvre une vue match et la page Relations ;
      durées lues dans `logs/http.log` et `http_timings` (le superviseur n'ouvre pas de navigateur)
      — dépend de C.5
- [!] C.5 go utilisateur, fusion `--no-ff` dans `feat/v75`, push, retrait du worktree — consigne
      utilisateur du 2026-09-26 : « on va pas fusionner sur feat/v75, je dirais quand je serais
      prêt ». Préalable signalé par la session du plan de suite de l'audit du décodeur : son lot
      J1 (`feat/suite-audit-decodeur`) doit être fusionné dans `feat/v75` AVANT la fusion v7.5 ->
      main.
- [x] C.6 handoff mis à jour (§7 décisions : ADR et lot A faits) ; entrée thought_log

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
- (P2, 2026-09-26) (1) [TRAITÉE par DA.10 / A.10 : localisation dans la brute puis lecture par
  `_latest` bornée] Le repli « toute la base » de DA.3 évalue la fenêtre `_latest` du journal
  canonique entière (3 à 7 s, 12 s sous charge) pour tout match ou périmètre qui contient un xuid que
  rien ne nomme ailleurs (2 % des matchs de JGtm, 65 % de ceux de Nuzzles, ses Relations) ; il ne
  trouve un nom que pour 11 couples sur 31 426. Piste mesurée à titre d'information, non traitée : les
  matchs candidats lus sur la table brute `match_kill_events` (`DISTINCT match_id WHERE
  feed_killer_xuid IN (…) OR victim_xuid IN (…)`, 0,31 s sur la copie, 2 threads) sont un sur-ensemble
  des partitions `_latest` de ces xuids ; lier la fenêtre à cette liste donnerait le même nom — mais
  c'est une lecture brute d'une table append-only (règle ART n° 2 : lecture par `_latest` seulement),
  décision hors de ce lot. (2) I4 : la vue match charge l'annuaire du même match trois fois par
  ouverture (Q12, Q21, Q23 ; plus ResolveGamertags sur l'endpoint des events) ; quand le repli part, il
  est payé deux fois en parallèle (Q12, Q23). (3) Q21 n'a pas d'ordre total (`time_ms` seul) : l'ordre
  des events de même milliseconde suit aujourd'hui l'insertion ; le départager par `he.id` demanderait
  la colonne `id` dans les schémas de test qui créent `highlight_events` (21 fixtures). (4) [TRAITÉE en
  A.12 ; le commentaire d'`explorer_repo.go` était en fait encore vrai — il parle de la clause des
  frags parfaits, pas de la vue — et a seulement été précisé] Commentaires
  hors périmètre devenus faux : `domain/match_view_raw.go` (`EventRaw.Gamertag` « résolu via
  v_gamertag_lookup… nil si orphelin, le service affichera le XUID brut »), `domain/match_view.go:306`
  (`ActorGamertag`), `platform/duckdb/explorer_repo.go:194` (« même approche que Q12MatchScoreboard »).
  (5) Q23 (historique commun du joueur avec chaque participant, sans nom) coûte seule 40-99 ms sous
  charge : le budget de 100 ms n'a pas de marge. (6) Aucun lecteur web de `GET
  /matches/{match_id}/events` (seul le type `MatchEventTimeline` est exporté, `lib/api/types.ts`) :
  ResolveGamertags sert un endpoint sans consommateur dans `apps/web`. (7) Gate P2 : le filtre
  `NoArt|Legacy|Sentinel` ne sélectionne pas `TestNoARTPatternsOnProtectedTables` (casse ; `NoART`).
  (8) Le test `TestGetMatchEvents_ResolvesGamertagViaView` garde un nom historique (la vue n'est plus
  lue) pour ne pas toucher la baseline.
- (A.10, 2026-09-26) (9) La localisation parcourt la table brute `match_kill_events` (3,95 M lignes,
  deux passes pour tueur et victime) à chaque repli : 150-360 ms par lecture de la vue match qui le
  déclenche, au-dessus du budget de 100 ms ; une passe unique (`feed_killer_xuid IN … OR victim_xuid
  IN …`) ou la compaction de la brute (lot C) sont les pistes, non mesurées. (10) L'allowlist de
  `TestNoRawAppendOnlyReads` est indexée par nom de fichier (`filepath.Base`) : l'entrée
  `identity_annuaire.go` couvre tout le fichier, pas le seul gabarit de localisation ; et le
  garde-rail ne scanne pas `internal/sync` (killcollector). (11) La table « Exceptions » de l'ADR 0036 (I1) liste encore les lectures
  retirées par le lot A (vue match, ResolveGamertags, Relations) et leur compte au ratchet : à retirer
  avec elles (« An exception leaves this list together with its read ») [TRAITÉE en A.11]. (12) Le harnais `player_repos_test.go` n'a pas de table brute
  `match_kill_events` : toute lecture en portée base d'un test qui l'utilise et déclenche le repli
  échoue sans `simulerJournalBrut` (`match_view_repo_annuaire_test.go`, appelée par
  `seedPorteeBase` et `TestGamertagRepo_ResolveGamertags`).
- (clôture, superviseur, 2026-09-26) (13) Deux tests à seuil de temps rougissent quand la machine
  est chargée (suite complète `go test ./...` pendant que d'autres sessions compilent) : `sync/skill`
  `TestLUSRV2Shadow_RafalesBornees_300Candidats` (seuil strict de 2 s dépassé de 5 à 86 ms) et
  `watcher` `TestPlayerWatcher_PostExitGrace_IdempotentInactive`. Verts rejoués seuls ; risque de
  faux rouge sur un runner CI chargé.

## 7. Journal (superviseur)

- 2026-09-26 : plan écrit ; worktree `LevelUp-wt-perf-perimetre` et branche `feat/perf-perimetre`
  créés depuis `feat/v75` `ea5682373` (local en avance d'un commit sur `origin/feat/v75`
  `378509f5e` : suppression de `INVOUT.csv`, non poussée).
- 2026-09-26 : P1 close (`948523a36`, exécuteur Opus). Consigne utilisateur en cours de route :
  un agent à la fois pour l'implémentation, Opus au plus, effort ajusté (types d'agents à effort
  fixé chargés après un rechargement de l'éditeur), puis jusqu'à 2, puis 4 agents en parallèle.
  Premier exécuteur P2 interrompu par le rechargement (rien commité) ; relancé en effort élevé,
  reprise de ses restes (copie de base, arbre de base). P3 lancé en parallèle (effort moyen,
  `6606fd3c9`).
- 2026-09-26 : P2 (`f576df10e`, `80184e83d`) : parité exacte, mais le repli kill-feed « toute la
  base » coûtait 3 à 7 s quand il partait (2 % des matchs de JGtm, 65 % de ceux de Nuzzles, Relations
  de Nuzzles). Décision DA.10 du superviseur : localiser les matchs candidats dans la table brute
  (lecture de localisation seule, allowlistée et datée), lire les valeurs par `_latest` bornée —
  `5ab3c13c7` : repli 150-360 ms, parité exacte, 0 nom perdu. Revue adversariale (C.2) puis A.11
  (`35e9766ff`). Changelog complété par le superviseur. Branche ni poussée ni fusionnée : attente
  de l'utilisateur (C.3 à C.5).
- 2026-09-26 (C.1, suite complète) : `go test ./...` sur `35e9766ff` : 188 paquets ok, 2 rouges
  hors du lot, tous deux sur des seuils de TEMPS pendant que d'autres sessions chargeaient la
  machine — `sync/skill` `TestLUSRV2Shadow_RafalesBornees_300Candidats` (3 rafales tenues
  2,006 / 2,032 / 2,086 s pour un seuil strict de 2 s) et `watcher`
  `TestPlayerWatcher_PostExitGrace_IdempotentInactive` (état `Watching` au lieu de `Idle` après
  l'expiration d'un délai de grâce). Rejoués seuls : verts (le second trois fois de suite). Le lot
  n'a touché ces paquets que par une ligne de commentaire (en-têtes P1.3 de deux tests
  `sync/skill`). Consigné en découverte (13), non traité.
- 2026-09-26 (revue fraîche demandée par l'utilisateur) : un relecteur Opus effort élevé, aveugle,
  sur tout le diff `ea5682373..ecbb918a1`, périmètre fermé : 0 défaut de comportement, 24
  conditions vérifiées, 3 constats P2 de véracité (exemple périmé du tableau Guardrails de l'ADR,
  écart (iii) annoncé épinglé sans test, commentaires du domaine décrivant l'ancienne jointure).
  Corrigés en A.12 (`220e0fbda`, test `TestAnnuairePorteeBase_EcartNomme_NomDeKillFeedVariable`,
  mutation rouge) ; vérifiés sur pièces par le superviseur (diff relu, tests rejoués : duckdb
  intégration ciblée et domain verts). Pas de seconde ronde : aucun P0 / P1.
