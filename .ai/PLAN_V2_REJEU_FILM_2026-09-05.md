# Plan v2 du rejeu et du film — 2026-09-05

> Plan d'exécution issu du registre `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md` (constats P0/P1
> vérifiés, décisions utilisateur du 2026-09-05). Contrat d'exécution : skill `plan-execution`.
> Chaque lot est exécuté par un agent dans un worktree dédié, relu par une revue adversariale
> (skill `adversarial-review`) en fin de tâche, puis intégré dans `feat/v75` par le superviseur.
> Source de vérité de l'avancement : ce fichier (mis à jour par le superviseur, dans le
> checkout principal uniquement) + un journal par lot `.ai/V7.5/v2/LOT_<X>.md` (tenu par
> l'exécuteur, dans son worktree).

## Décisions utilisateur (fermes, 2026-09-05)

| # | Sujet | Décision |
|---|---|---|
| 1 | `match_player_positions` (carte de chaleur) | Projection de l'artefact par `persist` après chaque cuisson, patron `bomb_stats_persister` ; le mode `-write` du diagnostic disparaît pour cette table |
| 2 | Document stocké / document servi | Séparer MAINTENANT, avant le tag v7.5.0 (forme de fil inchangée dans un premier temps : `generated.ts` sans diff) |
| 3 | Capability du rejeu | `film.replay_artifact` gouverne la PRODUCTION (pas de cuisson sans clé), comme les cinq clés `film.*` ; l'affichage suit (pas de fichier, pas de page) + capability title-level `replay` pour la route web |
| 4 | Seuil R5 côté web | Compté en lignes de CODE (`max-lines` eslint avec `skipComments`), seuil 500 inchangé ; le cliquet qui compte les lignes brutes disparaît |
| 5 | Déplacement de `analysis/filmdec` + `analysis/replay` sous `games/halo_infinite/film/` | APRÈS le merge de v7.5.0, commit de déplacement pur ; inscrit dans Notion sous « Séquence à dérouler à la release, dans l'ordre : » |
| 6 | Vocabulaire | « heatmap » banni partout (« carte de chaleur »), ajouté au garde anti-anglicismes et les manifestes existants corrigés ; « lobby » gardé comme mot assimilé, documenté dans la doctrine du garde |

## Règles d'environnement (à respecter par tout exécuteur)

- Worktree dédié `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-v2-<lot>`, branche `feat/v2-<lot>`
  (préfixe `feat/**` = la CI se déclenche au push). Checkout principal
  `C:\Users\Guillaume\Downloads\Scripts\LevelUp-go-migration` : lecture seule (données `data/`,
  registre, annexes), jamais d'écriture.
- Commandes `go` en SÉRIE (jamais deux à la fois), avec un cache propre au lot :
  `GOCACHE=C:\Users\Guillaume\AppData\Local\go-build-v2-<lot>`, `CGO_ENABLED=1`, depuis
  `apps/go-api`. Tests d'intégration : `-tags=integration -p 1`.
- Gates en AVANT-PLAN uniquement (aucun run d'arrière-plan, aucun outil d'attente).
- Web : `npm --prefix apps/web run typecheck`, `npm --prefix apps/web run lint`, vitest via
  `cd apps/web && node_modules/.bin/vitest run --pool=forks <filtre>` hors sandbox ;
  `git checkout -- apps/web/src/routeTree.gen.ts` avant tout staging (régénéré par l'outillage).
- `git add` NOMMÉ (jamais `-A` ni `.`) ; un commit par étape, préfixe `v2(<lot>.<n>)` ; jamais
  `git stash` ; push de la branche du lot en fin de tâche, puis `gh run list --branch <b>` et
  `gh run watch <id> --exit-status` en avant-plan ; tout job rouge se répare, même préexistant.
- Tests sur films réels (variables `REPLAY_FILM_DIR`, `KILLSOURCE_FIXTURES`, `ECS_TABLE_FILM`,
  `DELTA_WITNESS_FILM`) : réservés au lot E, un film à la fois, `-p 1`. Aucune cuisson
  d'artefacts en lot.
- `.ai/` : l'exécuteur n'écrit que `.ai/V7.5/v2/LOT_<X>.md` et une entrée en FIN de
  `.ai/thought_log.md` par tâche close. Découvertes hors périmètre : consignées, pas traitées.
- Doctrine : CLAUDE.md (règles 1-16, ART, anti-patterns), skills `arch-rules`,
  `frontend-patterns`, `color-tokens`, `db-schema`. Aucun emoji dans les fichiers versionnés.

## Protocole de revue et d'intégration

1. Fin de tâche : l'exécuteur commit, pousse, surveille la CI, rend un rapport (items statués,
   gates joués avec sorties, commits, découvertes, questions).
2. Le superviseur vérifie le rapport sur pièces, puis lance une revue adversariale (contexte
   frais, contrat du lot, lentilles L1-L6 selon le diff) : ronde 1, tri P0/P1/P2, corrections
   par l'exécuteur, ronde 2 sur les corrections. Deux rondes maximum.
3. Intégration dans `feat/v75` par le superviseur (merge sans fast-forward, un merge par lot),
   CI verte au niveau job, journal.
4. Ordre d'intégration prévu : C (capabilities) → A (faits) → B (document servi) → F (tests, CI)
   → G (outils) → E (décodeur) → D (modèle web, qui se rebase sur `feat/v75` avant sa dernière
   étape).

## Lots

Statuts : `[ ]` à faire · `[>]` en cours · `[x]` fait et vérifié · `[~]` couvert ailleurs (réf.)
· `[!]` non traité (justification au journal).

### Lot A — Faits du film (Go) — modèle Opus — `feat/v2-faits`

Tâche A-I (petite, à clore et relire d'abord)
- [ ] A.1 (P0-1) Bumper `KillSourceDecoderRev` (`sync/killcollector/collector.go:65`) à une valeur
      datée 2026-09-05 ; ajouter un garde-rail d'empreinte : test qui calcule une empreinte des
      sources non-test de `internal/games/halo_infinite/film/killsource/` et la compare à une
      empreinte figée à côté de la constante ; si l'empreinte change sans bump de la révision,
      le test échoue avec un message qui dit quoi faire. Vérifier que les familles « tirs » et
      « positions » (`match_weapon_shots`, `kill_positions`) partagent ou non cette révision ;
      documenter le résultat dans le journal (pas de nouvelle révision dans ce lot).
- [ ] A.2 (G4) Enrôler `kill_positions` et `match_weapon_hit_distance` dans
      `sync/no_art_patterns_test.go` (`tablesProtegees`) et `append_only_state_guard_test.go`
      après avoir vérifié qu'elles sont append-only avec vue `_latest` ; le gate doit passer sans
      aucune allowlist nouvelle.
- [ ] A.3 (A0) Le runtime n'écrit plus `data/titles/halo_infinite/reference/map_weapon_pads.json`
      (suivi par git) : `mvar_rattrapage.go:175-236` → `mapcatalog.AddEntry` écrit dans un overlay
      NON versionné résolu par `PathResolver` (sous `reference/generated/`, ignoré par git), fusionné
      en lecture par le chargeur du catalogue (la version versionnée prime en cas de doublon) ;
      garde-rail : test qui interdit toute écriture runtime du chemin versionné.
- Gate A-I : `go test ./internal/sync/... ./internal/analysis/replay/... ./internal/domain/...`
  + `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` +
  `golangci-lint run` sur les paquets touchés.

Tâche A-II
- [ ] A.4 (A1) Les dérivations post-cuisson (usage de session, statistiques d'Assaut, T0 film)
      se déclenchent sur « un artefact vient d'être rangé », quel que soit le rangeur
      (`StoreBuildArtifact` côté ouvrier, `buildAll` local) : un point d'entrée unique
      `derivations` appelé des deux chemins ; test qui prouve l'appel sur le chemin ouvrier.
- [ ] A.5 (A2) Un composant de rattrapage unique pour les passes dérivées : prédicat de
      fraîcheur = digest (« artefact absent OU périmé OU sans dérivés ») ; l'état « dérivés
      faits » est enregistré dans l'index des artefacts (digest des dérivations), pas déduit de
      la seule présence du fichier.
- [ ] A.6 (décision 1) `match_player_positions` devient une projection de l'artefact :
      persister INSERT-only sur le patron exact de `persist/bomb_stats_persister.go`, appelé par
      les dérivations ; `player_positions_repo.WriteMatch` (DELETE+INSERT) et le mode `-write`
      de `cmd/diag_weapons_v3` pour cette table sont supprimés ; lecture inchangée pour la carte
      de chaleur. `match_objective_events` (captures CTF du diagnostic) est HORS périmètre :
      consigner en découverte.
- Gate A-II : idem A-I + `go test -tags=integration -p 1 ./internal/sync/replayartifacts/...
  ./internal/persist/...` + test de non-régression ART.

### Lot B — Document servi (Go) — modèle Opus — `feat/v2-docservi`

- [ ] B.1 Créer `internal/domain/replaydoc` : les types du document SERVI, miroir exact de la
      forme de fil actuelle (mêmes noms de types, mêmes tags JSON, mêmes `omitempty`), sans
      aucun import de `analysis/replay`. Constante `ContractVersion` initialisée à la valeur
      courante de `analysis/replay.SchemaVersion`, documentée : « ne bouge que si la forme
      servie change ».
- [ ] B.2 Convertisseur exhaustif `analysis/replay.ReplayDocument` → `replaydoc.ReplayDocument`
      dans `internal/service/` (fonction pure), avec un test de parité par réflexion : tout champ
      exporté du document stocké a une décision (copié, transformé, ou explicitement non servi,
      dans une liste datée) ; réconcilier avec `TestReplayDocumentFieldCountIsFrozen`.
- [ ] B.3 `handlers/replay.go` (et toute route qui expose un type `analysis/replay`) ne sert
      plus que `replaydoc` ; `analysis/replay.ReplayDocument` n'est plus un schéma OpenAPI.
- [ ] B.4 Ratchet `archlint` : aucun type de `internal/analysis/` en corps de requête ou de
      réponse Huma (allowlist datée pour les exceptions restantes, avec date de retrait).
- Gate B : `make generate-types` puis `git diff --exit-code apps/web/src/lib/api/generated.ts`
  (vide) et `git diff --exit-code apps/go-api/openapi.yaml` (vide ou uniquement réordonnancement
  prouvé équivalent) ; `go test ./internal/api/... ./internal/service/... ./internal/domain/...
  ./internal/archlint/...` ; `make go-api-test` ; typecheck web.

### Lot C — Capabilities et vocabulaire (Go + web) — modèle Opus — `feat/v2-capabilities`

- [ ] C.1 Clé fine `film.replay_artifact` dans `config/titles/*/mappings/capabilities.toml`
      (`supported` pour halo_infinite, `not_exposed` pour halo_5), déclarée dans le registre Go
      des `CapabilityKey` ; capability title-level `replay` (`CapReplay`) dans
      `domain/title/registry.go` + `config/titles/halo_infinite/title.toml`, servie au web par le
      bootstrap (`lib/capabilities/capabilities.ts`, ratchet de parité existant étendu) ;
      `synthetic_title_b` (tests) déclare les clés `film.*` en `not_exposed` + un cas `supported`.
- [ ] C.2 Portes Go (décision 3) : `replayartifacts.Run` refuse en tête (avant `enqueueAll` et le
      rattrapage des cartes) quand le titre n'a pas `film.replay_artifact` ; les routes `/replay*`
      et les deux repos `WithObjectiveEventsRepo` / `WithPlayerPositionsRepo` répondent 503 via
      `ErrCapabilityNotSupported` quand la clé manque (la doc inversée de
      `api/wire/registry_pages.go:113-117` devient vraie) ; test 503 sur halo_5.
- [ ] C.3 Web : client de `GET /api/v1/titles/{slug}/capabilities` (query key dans
      `lib/query/keys.ts`), hook `useDataCapability(key)` jumeau de `useCapability`, un seul
      `FeatureGate` ; règle des deux portes appliquée à `MatchKillDistanceSection` (clé
      `film.kill_positions`, sinon rien), `SessionUsageSection` (`unsupported` → masqué), filtre
      « Avec rejeu / Sans rejeu » et colonnes rejeu de l'Explorer (capability `replay`). La route
      `replay.tsx` n'est PAS touchée (lot D).
- [ ] C.4 (décision 6) `heatmap` ajouté à `FORBIDDEN_PATTERNS` du garde anti-anglicismes ; toutes
      les chaînes FR « Heatmap … » corrigées (`explorer.toml:54`, `timeseries.toml:194`,
      `MatchPositionsHeatmap.tsx:42`, et tout autre `grep -i heatmap` côté FR) ; « lobby »
      inscrit dans la doctrine du garde comme mot assimilé (à côté de badge/playlist).
- Gate C : `go test ./internal/domain/title/... ./internal/games/... ./internal/api/...
  ./internal/sync/replayartifacts/... ./internal/contracttest/...` ; `make generate-types` ;
  typecheck + lint + vitest ciblé (capabilities, match-view, session-detail, explorer, garde
  anti-anglicismes).

### Lot D — Modèle web du rejeu — modèle Opus — `feat/v2-web-modele`

Tâche D-I (horloges)
- [ ] D.1 (P0-7) `apps/web/src/lib/replay/matchClock.ts` : conversions entre `event_time_ms`
      (horloge gameplay recalée côté Go), `t0_ms`, `originMs`, `t0FilmMs`, `frame ×
      frameIntervalMs`, avec tests à oracle écrit à la main ; `MatchKDCumulChart` et la courbe
      de score (`_scoreCurve.ts`) de l'onglet Chronologie partagent UN axe temporel ; le test qui
      verrouille la prémisse « borné au coup d'envoi » est corrigé.
- [ ] D.2 (P0-5) `features/match-replay/model/replayClock.ts` : `replayClock(doc, header)` rend
      une horloge établie ou `null`, un seul verdict pour la page ; les cinq sites
      (`replayWindow.ts`, `killFeedLogic.ts`, `replayMediaLogic.ts`, `presenceFeed.ts`,
      `seatLogic.ts`) la consomment ; garde-rail interdisant `originMs` hors de ce module.
- [ ] D.3 (J2) `killFx.ts` et `replaySound.ts` consomment `feedEntries` déjà recalé, sans
      refaire `alignFeed` ; (résidu P0-6) un seul formateur d'horloge visible dans le rejeu.
- [ ] D.4 (J3, J5) Supprimer `replaySchemaLogic.ts` et son garde ; supprimer `roundAtFrame` et
      ses six assertions.
- [ ] D.5 (N1, N2, N3) `_kdCumul.ts` et `_cadence.ts` extraits sur le patron de `_scoreCurve.ts`
      avec tests à oracle ; `MatchCombatCtfOverlay.test.tsx` réécrit avec l'option entière et
      des abscisses attendues ; `formatDurationMShort` aux quatre sites `MmSSs` + garde-rail.
- Gate D-I : typecheck, lint, vitest complet du dossier `features/match-replay` +
  `features/match-view` + `lib/replay` ; aucun changement visuel voulu (les captures avant/après
  des specs de rasterisation existantes doivent être identiques).

Tâche D-II (modèle, calques, arborescence)
- [ ] D.6 `features/match-replay/model/useReplayModel(doc, matchView, settings)` : le seul lieu
      de jointure (`clock`, `feed` recalé une fois, `score`, `media`, `identity`, `players`) ;
      remplace les 12 `useMemo` de `routes/.../replay.tsx` ; logique testable sans React.
- [ ] D.7 Contrat de calque unique `paint(ctx, frame, dpr)` pour les dix fonctions libres ;
      `draw` devient une boucle dans `replayCompose.ts`, testée au contexte enregistreur (ordre
      des calques, bascules) (M3).
- [ ] D.8 `playbackStore` hors de l'arbre React (le canvas écrit, tout le monde lit) ;
      `ReplayTransport` et `ReplaySettingsDrawer` remontent frères ; `useReplayDrawer` disparaît.
- [ ] D.9 Cinq canoniques avec garde-rail le même jour : `replayView.ts` (`CanvasView` +
      `projectTo`, K3), `buildLivesBySlot`/`lifeOfSlotAt` (K1), `useMatchSides(scoreboard)` (K2),
      `carriedGlyphPulse.ts` (K4), `covers`/`spansAt` (K5) ; les copies migrent toutes.
- [ ] D.10 Les 11 faux hooks redeviennent des fonctions pures testées ; garde de porteur (M4) et
      liste `InkVar` (M5) dérivés de la source.
- [ ] D.11 (décision 4) `max-lines` eslint 500 avec `skipComments` sur `apps/web/src` ; le
      cliquet de `placementFamily.guard.test.ts` qui compte les lignes brutes disparaît ;
      arborescence par responsabilité : `layers/`, `ui/`, `model/`, `hooks/`, `sound/`, `export/`,
      `settings/`, `i18n/` (déplacements purs, imports mis à jour).
- [ ] D.12 (M6) `tools/lint-no-hardcoded-colors.mjs` joué dans le job `frontend` de la CI
      (script `lint:colors`) ; les 9 copies partielles de tests supprimées.
- [ ] D.13 (N4) Les modules partagés que l'allowlist croisée dément descendent dans
      `lib/replay/` ; `tools/lint-cross-feature-imports.mjs` porte sur des modules, pas sur une
      paire de features.
- [ ] D.14 Dernière étape : `git merge feat/v75` (lot C intégré) puis la route `replay.tsx`
      gatée par la capability title-level `replay` (état « titre sans rejeu » propre).
- Gate D-II : typecheck, lint (dont `max-lines`), `lint:colors`, vitest complet web, specs de
  rasterisation identiques avant/après.

### Lot E — Décodeur (Go) — modèle Opus — `feat/v2-decodeur`

Tâche E-I (refonte à comportement identique)
- [ ] E.1 Avant tout changement : jouer et archiver les digests de référence (goldens
      inconditionnels `TestGoldenMiniBobine`, `equivalence_minifilm_test.go`,
      `zero_disque_test.go`, et l'équivalence locale sur films réels avec les variables
      d'environnement des tests, films lus dans `data/cache/` du checkout principal, un film à
      la fois, `-p 1`). Ces digests sont le gate de chaque étape suivante.
- [ ] E.2 (E1, E2, E6) Code mort : supprimer `filmdec/entity.go` + `entity_quant.go` sauf
      `bitLen`, `readQuantStat`, `quantStatDefaultWidth` ; les 22 `Set*` sans appelant ; la
      branche `if useLegacyAngularVel` et son drapeau ; le drapeau `useBipedDefaultStateDeser` et
      ses branches inatteignables (`traverse.go:1093`, `probe_export.go:101`) ; `dynPrecHook` et
      `repTraceHook` avec leurs 8 blocs de sauvegarde ; `probe_export.go` (9 exports sans
      appelant) et `frame_debug.go`. `consumeObjectAngularVelocity` RESTE (vivant pour `ti=40`).
      Redescendre le ratchet `archlint/filmdec_package_vars_test.go` à la valeur mesurée.
- [ ] E.3 (E3) Préambule d'événement : les six lectures passent par `PacketHeadEventType`
      (`event_list.go`) ; une seule table de domaines avec `dom3 = 7` (valeur mesurée) ; test qui
      confronte les tables recopiées (`weapon_hits_decode.go:17`, `zoom_events.go:68`) à la table
      canonique ; la justification périmée de `biped_pickups.go:207-209` est remplacée.
- [ ] E.4 (E4) `walkDeltaBipedRecords(fc, visit)` remplace les neuf triples boucles ; garde-rail
      interdisant le littéral `bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt` hors du marcheur.
- [ ] E.5 (E5) Le garde-rail du verrou (`replay/world_object_precision_guard_test.go:107-112`)
      généralisé en ratchet AST : tout appelant de production de `Scan*` détient
      `LockProcessDecode` ; `killcollector/positions.go` mis en conformité.
- Gate E-I : digests de E.1 identiques ; `go test ./internal/analysis/filmdec/...
  ./internal/games/halo_infinite/film/... ./internal/analysis/replay/... ./internal/sync/killcollector/...
  ./internal/archlint/...` ; `golangci-lint run` sur `filmdec`.

Tâche E-II (preuves en CI)
- [ ] E.6 (F3, résidu F1) Golden inconditionnel sur la mini-bobine versionnée pour les ~26
      familles `Scan*` non couvertes, en appelant les points d'entrée de famille (pas les
      enveloppes) : comptes et digests par famille figés dans un golden ; assertions nommées sur
      les indices d'archétype (`i0`/`i4`/`i11`/`i43-46`) et sur l'empreinte du registre réel ;
      `registry_test.go` repointé sur la mini-bobine versionnée avec `t.Fatal` (plus de chemin
      absolu, plus de `t.Skipf`).
- [ ] E.7 (F4) Contrôle code ↔ `ecs_table.tsv` sur les largeurs entières (`bits_typ`), 179
      composants, zéro fixture, joué en CI.
- Gate E-II : suite `filmdec` + `killsource` verte sans variable d'environnement ; les nouveaux
  goldens échouent quand on inverse une largeur (preuve écrite au journal : mutation manuelle
  puis retour).

### Lot F — Tests, garde-rails et CI (Go + CI) — modèle Opus — `feat/v2-tests-ci`

- [ ] F.1 (G1) Assertions de VALEUR sur le document obtenu par
      `wire/build_queue_worker_binary_integration_test.go` (kills, score, `originMs`, roster
      attendus écrits à la main depuis la fixture).
- [ ] F.2 (G3) Baseline de présence des tests (`scripts/check_test_baseline.sh`) : entrées pour
      `analysis/replay`, `replaybuild`, `replayartifacts`, `killcollector`, `objectiveevents` ; le
      contrôle « par paquet » réel ; le script invoqué par la CI (job `go-coverage`) si ce n'est
      pas déjà le cas — vérifier sur pièces avant.
- [ ] F.3 (G7) Tests de `ReplayPurgeCron.RunOnce` (garde `months <= 0`, sélection par âge,
      aucune suppression hors du dossier des artefacts).
- [ ] F.4 (I1) Les deux tests qui ouvrent le jeu passent sous `//go:build gamefiles`
      (`cmd/mapstruct-build/equivalence_gamefiles_test.go`, `cmd/mapfond-build/reglages_test.go`) ;
      `himap/corpus_tag_test.go` promu en ratchet `archlint` sur tout le module.
- [ ] F.5 (H7) `.golangci.yml` : l'exemption `^cmd/` ne couvre plus les binaires de production
      (`path-except` sur `cmd/(server|levelup|replay-worker|replay-build|replay-equiv)/`) ;
      l'exemption fine morte (`:211`, `:246`) supprimée ; le ratchet reste vert sur la branche.
- [ ] F.6 (M1) Les deux specs de rasterisation (`e2e/replay-explosion-raster.spec.ts`,
      `replay-muzzle-raster.spec.ts`) jouées dans le job `frontend` de `ci.yml` (sans serveur) ;
      preuve : un run CI vert qui les exécute.
- Gate F : `go test ./internal/archlint/... ./internal/service/... ./internal/api/wire/...` ;
  `bash scripts/check_test_baseline.sh` ; `golangci-lint run ./...` ; CI verte sur la branche.

### Lot G — Outils et catalogues (Go, docs) — modèle Sonnet — `feat/v2-outils`

- [ ] G.1 (H5) `cmd/levelup/backfill_memlimit.go` et `cmd/replay-worker/memlimit.go` passent sur
      `internal/filmproc` (`memguard.go`, callback) ; `sentinelleTokens` réduit au seul helper ;
      le garde `archlint/no_unbounded_film_loop_test.go:168-172` mis à jour (il entérinait la
      justification périmée).
- [ ] G.2 (I2) `internal/himap/heightfield.go` : 175 lignes mortes et leur test supprimés.
- [ ] G.3 (H2) Porter `C:\Users\Guillaume\Downloads\Halo Infinite - Sons armes\_outils\livraison.py`
      en Go comme mode `livrer` de `cmd/weapon-sounds` (sortie identique : `weaponSoundVariations.ts`
      et les assets de `static/sounds/`, prouvée par comparaison octet à octet sur la sortie
      actuelle) ; la recette `.ai/V7.5/RECETTE_SONS_ARMES.md` mise à jour.
- [ ] G.4 (H3) Section « Chaînes de fabrication des assets versionnés » dans `docs/COMMANDS.md`
      ET `docs/FR/COMMANDS.md` (règle 15) : les onze chaînes, sortie versionnée, prérequis, quand
      rejouer (liste dans l'annexe G9 du registre).
- Gate G : `go build ./...` ; `go test ./internal/himap/... ./internal/archlint/... ./cmd/levelup/...
  ./cmd/replay-worker/... ./cmd/weapon-sounds/...` ; `golangci-lint run` sur les paquets touchés.

### Après la release (Notion, décision 5)

- [ ] R.1 Déplacement pur de `internal/analysis/filmdec` et `internal/analysis/replay` sous
      `internal/games/halo_infinite/film/` (ADR 0012), commit de déplacement seul, ratchet
      « `analysis/` n'importe pas `games/{slug}` » posé avant.
- [ ] R.2 `cmd/` en trois familles (`cmd/tools/` pour les onze chaînes de fabrication) ; sondes
      sans référence supprimées ou `levelup diag <x>`.

## Constats P2 hors lots (backlog, entrent au lot qui touche leur fichier)

Voir registre, sections « requalifiés P2 » et « P2 des auditeurs ». Non planifiés ici : C1, C2,
C3 (ports et ratchets `archlint` de couches), B5 (`KillPairToleranceMS`), D4 (`games.Resolver`
aux deux sites Halo), G5 (`FilmCacheDir` par titre), L8 (`soundUrlOf` avec slug), J2 résidus.

## Journal

- 2026-09-05 : plan écrit depuis le registre ; 7 worktrees créés (`LevelUp-wt-v2-*`), jonctions
  `node_modules` pour docservi, capabilities, web-modele, tests-ci ; lancement des lots A, B, C,
  D, E, F, G en parallèle.

## Découvertes (hors périmètre, à ne pas traiter dans les lots)

- (vide)
