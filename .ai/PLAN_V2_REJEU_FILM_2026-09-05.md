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
- Gate de non-régression du rejeu (`make replay-corpus-gate`, `cmd/replay-corpus-gate`,
  manifeste `config/replay_corpus.toml`) : à lancer AVANT tout merge qui touche
  `analysis/replay`, `replaybuild`, `filmdec`, ou qui bumpe `SchemaVersion` — cf.
  docs/COMMANDS.md et `.ai/V7.5/v2/CORPUS_TEMOIN_2026-09-06.md`.

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
- 2026-09-06 : lot B rendu (4/4, CI verte, commits `40667b2c2`..`fa9fdae54`) ; tâche A-I rendue
  (3/3, CI verte, `2009fbfa7`..`26a488214`), A-II lancée. Directive utilisateur : plus de revue
  adversariale avant la fin de tous les lots (les quatre déjà lancées finissent). Verdict B-R2
  reçu : C1 P1 (la parité ne teste jamais la classe zéro/nil : `int` → `*int` ou `[]` contre
  `null` passent tous les gates ; réparation = jouer `comparer` sur un document zéro et à
  tranches vides), C2 P2 (un alias `type X = replay.ReplayDocument` contourne le ratchet
  `archlint`), C3 P2 (plancher documenté 104, mesuré 226). Aucune divergence live. Corrections
  à faire par l'exécuteur B en fin de chantier, avant merge.
- 2026-09-06 : verdict B-R1 reçu : équivalence de fil prouvée octet pour octet sur 106 artefacts,
  109 sidecars, 86 entrées de callouts et 1 000 documents de fuzz ; trois P2 (`ContractVersion`
  sans lecteur, `champsNonServis` incapable de couvrir un TYPE stocké neuf, prose du producteur
  recopiée dans `replaydoc/map.go`). Leçon de méthode : deux relecteurs dans le même worktree se
  polluent (mutations concurrentes) — un relecteur par worktree désormais. Quota : les deux
  relecteurs A-I arrêtés (revue A d'un bloc avec A-II en fin de chantier) ; lots D et E mis en
  pause à 01:05, reprise programmée 03:21 ; surveillance CI abandonnée pour tous les lots
  (vérification par le superviseur à l'intégration, demain).
- 2026-09-06 : lot C rendu (4/4, HEAD `f319393aa`, CI partiellement vue : 6 jobs verts, 2 en cours
  au dernier relevé, à revérifier). Écarts au plan constatés sur pièces : `title.toml` n'existe pas
  (descripteur built-in, `CapReplay` déclarée dans `NewRegistry()`) ; les manifestes « heatmap »
  vivent sous `apps/web/src/lib/i18n/manifests/` ; le chemin de test est `./contracttest/...` et
  le gate exige `-tags=integration`. Découverte à traiter en ronde de correction C :
  `MatchPositionsHeatmap` et ses deux hooks n'ont pas de garde `enabled` sur `film.*` (état vide sur
  un titre sans film) ; question ouverte : niveau INFO du refus de production contre DEBUG des
  gates jumeaux (à trancher en revue).
- 2026-09-06 : lot G rendu (4/4, HEAD `8614e2f2c`, 9 commits ; CI vue partiellement : Go Lint,
  build/test, frontend, lease, contrat, gitleaks, pre-check verts ; Go Coverage + Baseline non
  conclu, commit de clôture non surveillé). G.3 : mode `livrer` porté (dont un port bit à bit du
  générateur MT19937 de CPython), preuve octet à octet contre les scripts Python sur un jeu
  synthétique ; `[!]` comparaison avec l'artefact de production impossible, les `.wav` sources
  ont disparu du poste depuis le 2026-08-16. Découvertes : l'annexe G9 est imprécise sur
  `map_objects.csv`/`forge_object_types.csv` (import manuel, pas de producteur) et sur
  `mapnav-fetch` (sortie gitignorée) ; `vehicle-sprite` n'a pas de commande unique reproduisant
  les 18 véhicules ; un commit (`14dd46fd3`) porte un mojibake cosmétique dans son message ;
  un `git add` multi-chemins échoue en bloc sur un pathspec invalide. Coût : 828 k tokens pour
  un lot Sonnet (410 appels d'outils), plus que tout lot Opus rendu — Sonnet n'est pas moins cher
  sur un lot à quatre items hétérogènes.
- 2026-09-06 : lot F rendu (6/6, HEAD `1dd9b8ab4`, CI non surveillée). F.2(c) `[~]` : la CI
  invoquait déjà le script de baseline (`ci.yml:374-376`). Baseline 8 586 → 9 795 noms (+1 209
  entrées réelles pour les 5 paquets). Les deux specs de rasterisation étaient ROUGES au premier
  rejeu (harnais périmé par deux refactorings v7.5) : réparées, 3/3 vertes en local. F.5 :
  `path` + `path-except` sur une même règle est refusé par golangci-lint v2.12.2 ; `path-except`
  seul porte la condition ; dette visible 260 → 300, toute dans `cmd/server`/`cmd/levelup`.
  À FAIRE À L'INTÉGRATION (superviseur) : (1) `CLAUDE.md` section « Commandes utiles » cite
  `internal/himap/corpus_tag_test.go`, supprimé : pointer `internal/archlint/gamefiles_tag_test.go`
  dans le commit de merge ; (2) décision : les deux exemptions fines (`cmd/server/main.go`,
  `cmd/levelup/main.go`, funlen + gocyclo) restent SUPPRIMÉES — dette visible, ratchet vert, un
  hotfix dans `main()` ne fait pas remonter la ligne de déclaration ; (3) le lot E supprime des
  tests de `analysis/replay`, `killcollector`, `objectiveevents` désormais enrôlés dans la
  baseline : rejouer la baseline au merge de E. DÉCOUVERTE À INSTRUIRE AVANT LA RE-CUISSON DU
  PARC : sur le fixture `film_e2e/c0a82e88` (Husky Raid CTF) au schéma 39, `flagCarries` = 0 et
  `objectiveObjects` = 0 alors que le schéma 37 donnait « 92, famille flag » sur le même fixture
  et que l'API donne 3 captures ; piste `objectiveevents.ObjectiveTypeOf("Husky Raid:CTF")`.
  Autres découvertes : mode `coverage` du script de baseline invoqué par aucun gate ; `t.Skip`
  permanent compté « présent » ; fixture de purge dépendante du fuseau du runner ; 49 issues de
  lint visibles sur `cmd/server` (5) et `cmd/levelup` (44), dont `main` à 149 de complexité.
- 2026-09-06 : tâche A-II rendue (3/3, commits `69aed7e53`, `9b01446df`, `bddb608d1`, CI non
  surveillée). A.4 : `replayartifacts.Deriver` appelé des deux rangeurs (`Run` et
  `StoreBuildArtifact`), preuve de morsure ; A.5 : index `<artefact>.derived.json` + prédicat
  `DerivationsUpToDate` + rattrapage borné (horizon 64, plafond 5) sur les deux placements, 106
  artefacts locaux en ~21 cycles ; A.6 : `match_player_positions` CONVERTIE EN APPEND-ONLY
  (migration : `id` PK + `positions_pass` + vue `_latest` par passe), persister INSERT-only,
  `WriteMatch` + `write_conn.go` + `-write` positions du diagnostic supprimés ; grain de
  décimation `GrainPositionsMS = 20 000` (celui que le schéma déclarait, 215 positions par match
  contre 31 051 brutes) = seul paramètre produit tranché par l'exécuteur, à confirmer en revue.
  Risque d'intégration D-7 : A-I et C modifient tous deux `internal/domain/title/registry.go`
  (ajouts indépendants). `match_objective_events` non touchée (PK naturelle, pas de `_latest`).
- 2026-09-06 03:30 : tâche D-I rendue (5/5, HEAD `e78459c0f`, 7 commits, vitest web entier 6 287
  verts, build OK, CI non surveillée). Décisions produit de l'exécuteur à valider en revue :
  (1) ancre de l'axe commun = `header.t0_ms` (celui que le producteur retranche pour
  `event_time_ms`) ; (2) `_scoreEvents.ts` traité avec `_scoreCurve.ts` ; (3) sur un artefact
  sans `originMs`, le bloc « Score dans le temps » ne rend rien et la piste Médias se tait
  (0 carte perdue : les 5 artefacts concernés étaient déjà écartés par `filmClockTrusted`), le
  fil garde son repli mesuré ; (4) formateur du rejeu = celui qui tronque. Découvertes :
  `replay-explosion-raster.spec.ts` rouge AVANT le lot (harnais à 7 imports contre 6 réels ;
  réparé côté lot F, à réconcilier au merge) ; `SynthesisBipolaireChart` rendait « 1m60s »
  (corrigé par le helper canonique) ; `REGISTRE_REPORTS.md` l.449 à amender par le superviseur
  (condition exécutée par D.4). D-II lancée en deux temps : D.6-D.8 puis D.9-D.14.
- 2026-09-06 03:45 : tâche E-I rendue (5/5, 7 commits, 43 fichiers, +517/−1 317, CI non
  surveillée). Tous les témoins identiques à la référence archivée (`LOT_E_digests_avant.md`) :
  goldens inconditionnels, 4 goldens killsource sur films réels, 3 témoins de marche delta,
  table ECS, intégration killcollector ; ratchet des variables 118 → 111 ; trois garde-rails
  neufs validés par mutation. Découvertes : le golden `fccc61cd` et le témoin de marche delta
  étaient DÉJÀ rouges sur la base (preuve mesurée de P0-1, traité par A.1) ; 14 variables de
  paquet sans écrivain restent, dont des largeurs MESURÉES de chemins non nominaux (savoir de
  rétro-ingénierie) → E.8 ajouté à E-II : les convertir en constantes documentées (provenance,
  mesure), jamais les jeter ; `golangci-lint` a un cache GLOBAL indépendant de `GOCACHE` qui a
  menti (« 0 issues » sur un paquet qui en a 6) → toute mesure de lint isole
  `GOLANGCI_LINT_CACHE` ; `TestKillSourcePositionsFilmReelEtRelitParLaVue` se skippe (film codé
  en dur sur une carte hors catalogue), seul test de bout en bout de `buildPositionRows` → lot F
  ou ronde de correction ; les « 49 étapes d'équivalence » du registre sont `cmd/replay-equiv`,
  qui cuit un artefact par film (interdit dans les lots). E-II lancée (E.6, E.7, E.8).
- 2026-09-06 04:35 : tâche E-II rendue (3/3, commits `8a65d0969`, `25dbad2e6`, `0dd6e0fc8` +
  journal, HEAD `98df0b00c`, CI non surveillée). E.6 : golden inconditionnel sur la mini-bobine,
  30 familles (24 peuplées, 6 vides ou en refus propre expliquées : Fiesta d'arène, donc pas de
  véhicule ni d'objectif), `registry_test.go` sans chemin absolu ni skip ; deux pièges corrigés
  (un zéro de mauvais appel n'est pas une population vide : catalogue d'armes dérivé du film ;
  digest stable par réflexion, pas `%+v`). E.7 : 179 largeurs classées (114 fixes dont 111
  d'accord, 65 gardées), 3 écarts datés. E.8 : 10 constantes documentées, 5 suppressions,
  2 gardées, ratchet 111 → 96. Témoins E.1 toujours identiques. Mutations : `dom4RefWidth` rougit
  la seule famille qui lit ce domaine, `bipedIndexBits` en rougit 14. Découvertes (lot de
  révision de la table, pas à comportement identique) : `ti=43 i=0 object-position-component`
  table 15 bits contre 45/60 consommés ; `biped-map-editor-flag-component` table 1 contre R(8)
  confirmé au décompilé (table périmée) ; `accumWorld`/`accumSlot`/`inferResyncTargets` =
  interrupteurs de deux mécanismes entiers sans écrivain (décision produit) ; `weaponShots` /
  `weaponDamages` n'ont que la forme `dir`. LOT E TERMINÉ.
- 2026-09-06 05:20 : D-II premier temps rendu (D.6, D.7, D.8 magasin `[x]` ; HEAD `ef3c3be4a` ;
  route 395 → 316 L et 13 `useMemo` → 3 ; `draw` 222 → 22 L, 25 calques en donnée
  (`LAYER_ORDER`), composition testée au contexte enregistreur ; `playbackStore` par
  `useSyncExternalStore` ; vitest web 6 339 verts, rasterisation 3/3 identique). `useReplayDrawer`
  reste (un consommateur, 120 L que le canvas à 651/665 ne peut absorber). D.8 « Transport et
  tiroir frères du canvas » `[!]` DIFFÉRÉ par décision du superviseur : le tiroir est positionné
  en `absolute` par rapport à la racine du canvas, changer son parent change sa position à
  l'écran et aucune spec de rasterisation ne le couvre → item D.15, à faire APRÈS un gate visuel
  validé par l'utilisateur (témoins nommés par lui), pas dans ce chantier à UI constante.
  Découvertes : le témoin de rasterisation portait deux dérives (compte d'imports ET
  `drawGrenadeRestLayer` déplacé) — la version du lot D fait foi au merge, celle du lot F n'en
  corrige qu'une ; la position de lecture ne se remet pas à zéro d'un match à l'autre (avant
  comme après le lot, changement de comportement, hors contrainte) ; `buildScene` porte une
  exemption R5 assumée (table de liaison de 179 L). Second temps lancé (D.9 à D.14).
- 2026-09-06 06:40 : D.9 et D.10 (M4, M5) rendus (HEAD `2225147ac`, 6 commits ; K3 : 8
  redéclarations + 36 sites migrés ; K5 : 12 copies et non 10 ; gardes de K2/K5 resserrés sur la
  formule entière après 4 faux positifs ; vitest web 6 354 verts, rasterisation 3/3). D.10
  « faux hooks » requalifié `[~]` par le superviseur sur la mesure de l'exécuteur : ils sont 15,
  leur logique est déjà pure et testée à côté (ex. `useReplayVipCrown` → `drawVipCrown` testé),
  ils ne portent que mémoïsation et câblage ; les convertir déplacerait ~15 `useMemo` dans le
  canvas et risquerait une perte de mémoïsation à 60 im/s sans gate — constat W1 réfuté sur
  pièces pour ce qui est de la testabilité. Découverte : `MatchPadControlSection.tsx` lit
  `team_side` brut (troisième lecture du scoreboard, famille N4). L'exécuteur D s'arrête sur
  saturation de contexte (868 k tokens) ; exécuteur frais lancé sur D.11-D.14 depuis le journal.
- 2026-09-06 08:15 : D.11-D.14 rendus par l'exécuteur frais (HEAD `bb379d10b`, 14 commits ;
  `max-lines` 500 en error avec `skipComments`, cliquet de lignes brutes supprimé, 26 en-têtes
  repointés, arborescence en 8 dossiers par `git mv` purs + README + garde d'arborescence ;
  `lint:colors` en CI, 5 copies supprimées avec preuve 20/20 par mutation, 4 assertions de RENDU
  conservées à raison (la couleur peut venir de la donnée) ; 7 modules descendus dans
  `lib/replay/`, allowlist par modules nommés (2 paires → 8 modules), imports croisés hors tests
  9 → 4 ; merge de `feat/v2-capabilities` (1 conflit, thought_log) et route sous `matchmaking`
  puis `replay`, 4 cas de test ; vitest web 6 376 verts, rasterisation 3/3 après chacun des 12
  commits). Découvertes : `lib/ → features/` n'est contrôlé par aucun lint de frontière ; 7
  violations cross-feature restantes hors rejeu ; le témoin de rasterisation cherche désormais
  ses modules par nom (`e2e/_helpers/replaySource.ts`) — cette version fait foi au merge avec F.
  TOUS LES LOTS SONT RENDUS (A, B, C, D, E, F, G). Suite : vérification CI des 7 branches,
  revues adversariales en bloc, corrections, intégration dans l'ordre C → A → B → F → G → E → D.

## Découvertes (hors périmètre, à ne pas traiter dans les lots)

- (vide)

## Décisions utilisateur du 2026-09-06 (fermes)

| # | Sujet | Décision |
|---|---|---|
| 7 | Perte des ports de drapeau au schéma 39 (fixture Husky Raid CTF) | GRAVE, à instruire immédiatement : worktree `LevelUp-wt-v2-ctf`, branche `feat/v2-ctf-drapeaux`, bissection 37 → 39, correctif + test de non-régression, avis sur bump de schéma |
| 8 | `match_player_positions` append-only + grain 20 s (A.6) | Confirmés (recommandation superviseur : doctrine anti-corruption, grain déjà déclaré par le schéma et consommé en grille 20×20, constante réversible) |
| 9 | Transport et tiroir en frères du canvas (ex-D.15) | ON NE TOUCHE PAS : item abandonné (risque visuel sans valeur produit) |
| 10 | Table des largeurs `ecs_table.tsv` (`ti=43 i=0`, `biped-map-editor-flag-component`, 3 écarts datés) | À CORRIGER PRUDEMMENT : item E.9 après la revue du lot E — chaque entrée corrigée est adossée à une mesure (décompilé, film témoin), aucun changement de comportement du décodeur, le contrôle E.7 perd les allowlists correspondantes |
| 11 | Mécanismes de resynchronisation sans interrupteur (`accumWorld`, `accumSlot`, `inferResyncTargets`) | Gardés tels quels, commentaire daté au plus ; pas de retrait de fonctionnalité |

## Journal des revues adversariales (ronde 1, 2026-09-06)

| Lot | Verdict | P1 | P2 | Tiennent | Suite |
|---|---|---|---|---|---|
| B | B-R1 + B-R2 : équivalence octet pour octet prouvée (106 artefacts, 1 000 fuzz) | 1 (parité aveugle au zéro/nil) | 5 | 13 + 21 | corrigés (6/6, `49ab682af`), ronde 2 en cours |
| C | C-R1 : comportement livré exact, gardes qui ne mordent pas | 2 (garde des routes sur routeur reconstruit ; fixture synthétique sans test) | 4 (tests colonne, flash halo_5, filtre miroir, `FeatureGate` sans appelant) + obs. INFO | 21 | corrections en cours (8 points) |
| D | D-R1 : 25/25 calques identiques, jointure et replis équivalents, « 0 carte perdue » recompté vrai | 2 (câblage `t0Ms` sans test ; table de liaison peintre/calque sans test) | 7 (ordre tautologique, infobulle tronquée = exception acceptée, lint couleur aveugle à oklch/rgba et à `lib/`, garde horloge contournable, « 1m00s » = exception acceptée, exemption `max-lines` mal placée, 12 commentaires orphelins) | 24 | corrections en cours (11 points) |
| F | F-R1 : 25 conditions, 19 mutations | 1 (le « différentiel film ↔ API » est imposé par le pont d'identité : doc inversée) | 1 (ratchet `gamefiles` sur 2 racines) | 25 | corrections en cours (3 points) |
| G | G-R1 : fidélité du port prouvée sur jeu synthétique, MT19937 conforme | 3 (`livrer` exige jeu + cgo ; en-tête `GENERE PAR` non mis à jour ; aucun test octet à octet) | 7 (pic non ré-échantillonné, logs sentinelle modifiés, ratchet promis absent, vote vide avorte après effacement, `ntpath` partiel, `repr(float)`, erreur avalée) | 26 | exécuteur frais, corrections en cours (10 points) |
| A | A-R1 : 21 conditions, 12 mutations | 2 (marque posée sans écriture → exclusion à jamais ; rattrapage inatteignable quand rien à cuire) | 5 (verrou d'overlay ENOENT, migration non testée, `team = -1` partout + filtre mort, marques `.derived.json` polluent purge et backfill, 4 acquisitions du writer dans un handler à 30 s) | 21 | exécuteur frais, corrections en cours (7 points) |
| E | E-R1 : comportement identique confirmé (tous témoins), preuves neuves trouées | 2 (avance du marcheur non testée ; témoin delta ≠ oracle de E.4) | 4 (garde préambule contournable, ratchet verrou liste fermée + `rdata_weapon_scan` sans verrou, `-update` régénère le golden, en-tête du golden faux) | 21 | exécuteur frais, corrections + E.9 table ECS en cours |

Décisions superviseur : infobulle tronquée et « 1m00s » = exceptions documentées (5 et 6) ; `ContractVersion` supprimée ; `typesNonServis` ajouté ; hooks de mémoïsation gardés.

- 2026-09-06 (suite) : B-R3 (ronde 2) : les six corrections FERMÉES, un P3 neuf (garde trop
  strict sur exclusion imbriquée) — LOT B PRÊT À INTÉGRER (HEAD `49ab682af`). F corrections
  rendues (3/3, HEAD `62edaec30`). INSTRUCTION CTF CLOSE (`feat/v2-ctf-drapeaux`, `086a15f62`) :
  AUCUNE régression de production — la cuisson du film complet est identique avant et après le
  merge cuisson-perf, `flagCarries = 0` sur ce match à TOUS les schémas (3 prises sans pont
  d'identité), « 92 » était un compteur de journal ; la perte réelle était dans le FIXTURE E2E
  (morceaux 00 et 07 doublement compressés zlib, générés le 2026-08-25), rendu invisible puis
  actif par `c17f4941f` (retrait du repli zlib de `ParseRegistryChunk` → registre VIDE → 11
  calques tombés dans la seule cuisson réelle de la CI, verte faute d'assertion de valeur).
  Correctif : fixture pelé (octets du cache, sha256), `ParseRegistryChunk` REFUSE un tampon
  compressé (`ErrRegistryStillCompressed`), 3 tests dont l'intégrité du fixture, assertions de
  valeur E2E (captures = 3 = feuille de match). Pas de bump de schéma, parc non concerné.
  CONSÉQUENCE POUR F : `build_queue_worker_valeurs_integration_test.go` (F.1) a figé les
  valeurs du fixture CASSÉ (originMs 34 870, t0FilmMs 35 170, 22 vies…) → à re-mesurer sur le
  fixture corrigé au merge (intégrer CTF avant F). Disque : C: à 99 %, cache go-build principal
  69 Go, deux agents en « No space left on device » → purge (caches de relecteurs, `go-link-*`,
  `go clean -cache` principal).

- 2026-09-06 (ronde 2) : B fermé 6/6 (P3 résiduel) → PRÊT ; F fermé 3/3 + 2 retouches doc
  (`44067dfb6`) → PRÊT ; D fermé 10/11 + retouches N1/N4/N2/N3 (`8522e9a58`, vérifiées sur
  pièces : tolérance `/*` retirée, motif horloge qualifié) → PRÊT (intégration en dernier) ;
  C fermé 7/8, retouches N1 (porte sur un ANCÊTRE du montage, pas un frère) + N2 en cours ;
  E corrections 7/7 + E.9 (`caee98022`, `ti=35 i=50` 1 → 8 adossé au décompilé, notes pour
  `ti=43 i=0` et `ti=37 i=14`, contrôle G5) → ronde 2 en cours ; G corrections 10/10
  (`3efe72e71`, test octet à octet versionné, `ntpath` et `repr(float)` fidèles, publication
  tout ou rien) → ronde 2 en cours ; A corrections en cours (exécuteur frais). CTF : la revue a
  infirmé le volet « CI aveugle » (le téléchargeur de l'ouvrier pèle déjà une couche) mais
  révélé le fond : entre le schéma 20 (parc) et le HEAD, les actions `flag_captures` /
  `flag_steals` de `c0a82e88` ont disparu (17 → 12) et les 5 pontés sont échangés → bissection
  20 → 38 relancée. Proposition au user (feu vert attendu) : balayage du parc local (106
  artefacts, 9 schémas) re-cuits un à un au HEAD et comparés champ par champ, avant puis après
  intégration.

- 2026-09-06 11:00 : INTÉGRATION — B fusionné (`e1dfe6558`, `--no-ff`, conflit thought_log
  résolu par concaténation) ; F fusionné (`47740fc5e`) avec CLAUDE.md corrigé (garde-rail
  gamefiles → `internal/archlint/gamefiles_tag_test.go`). Conflit SÉMANTIQUE B/F attrapé par le
  hook pre-push `go-vet-cgo` : le test de valeurs F.1 typait le document `replay.ReplayDocument`
  alors que B fait servir `replaydoc.ReplayDocument` → test adapté aux types miroir (mêmes
  champs), `go vet` + intégration `api/wire` verts (`ok 18 s`). Décision user : feu vert au
  balayage du parc (un film par processus borné, jamais de bombe RAM) ; corpus élargi aux
  artefacts des autres worktrees et de la clé PNY (`E:\data\cache\replays\`, état du 31/07).
- 2026-09-06 11:20 : C fusionné (ronde 2 close par `b13f36fe9` : porte cherchée sur un ANCÊTRE
  du montage, mémo par titre prouvé) ; conflit thought_log concaténé. Balayage du parc lancé
  (worktree `LevelUp-wt-v2-balayage`, outil `cmd/replay-diff`, rapport
  `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md` attendu). Intégrés : B, F, C. Restent : A (corrections),
  G et E (ronde 2), D (prêt, en dernier), CTF (enquête rouverte).
- 2026-09-06 11:45 : ENQUÊTE CTF — VRAIE CAUSE ÉTABLIE (`feat/v2-ctf-drapeaux`, `91296072c` erratum
  + `eb5585109` correctif) : le commit `d173b1a8c` (2026-08-28, `replaybuild/matchfacts.go`,
  `identifiedEvents`) a REMPLACÉ le pont d'identité par triplet (`SlotIdentityFrom`) par le pont
  par morts (`ResolveRoundIdentity`, `deathInstantMin = 3`) alors que les deux couvertures sont
  complémentaires : les joueurs qui meurent moins de 3 fois (les meilleurs, donc les porteurs)
  sortaient du pont → actions d'objectif perdues (17 → 12 sur `c0a82e88`, captures et vols de
  SweatyYeti75 disparus). Régression, pas décision (« neutralité prouvée » contre le mauvais
  témoin). Correctif : `RoundIdentity.CompletedByLines` (mono-manche, compléter sans contredire,
  aucun xuid deux fois) → 7/7 pontés, 23 actions, chaque joueur = sa ligne de feuille de match.
  Impact : tout artefact cuit depuis le 2026-08-28 sur film mono-manche avec un joueur à < 3
  morts, toutes familles ; pas de bump de schéma (contenu enrichi) ; propagation par le
  `backfill-replay` de la release (inscrit au registre). Revue adverse CTF-R2 lancée. DÉCISION
  USER EN ATTENTE : `flagCarries` subit le même plafond mais son identité est construite dans
  `analysis/replay` sans faits de match (frontière délibérée) — recommandation superviseur :
  compléter dans `replaybuild` après cuisson (comme les actions), sans franchir la frontière.
- 2026-09-06 12:10 : E-R2 : six constats FERMÉS, E.9 jugé prudent (3 lignes du TSV, zéro code du
  décodeur, chaque note vérifiée), 3 P3 en retouche (résorption par la table non détectée, import
  aliasé invisible au ratchet du verrou, liste d'angles morts non exhaustive). G-R2 : dix constats
  FERMÉS, goldens reproduits indépendamment par le vrai Python (six `.wav` + `.ts` + console
  identiques), 2 P3 en retouche (préfixe UNC sensible à la casse, double `Peak()` sur une ligne de
  log). A-R2 en cours ; CTF-R2 (correctif du pont d'identité) en cours ; balayage en cours.
- 2026-09-06 12:30 : G retouches vérifiées sur pièces (`6effae3ac` : `EqualFold` sur le préfixe UNC,
  table à 32 entrées ; `Peak()` lu une fois) et G FUSIONNÉ dans `feat/v75`. Intégrés : B, F, C, G.
- 2026-09-06 12:45 : E retouches vérifiées (`60072d717`, 3 gardes, zéro code du décodeur, golden
  intact) ; aucun test supprimé par E ne figure dans la baseline (vérifié) ; E FUSIONNÉ dans
  `feat/v75`. Intégrés : B, F, C, G, E. Restent : A (ronde 2), D (dernier), CTF (revue R2).
- 2026-09-06 13:00 : CTF-R2 CONFIRME le diagnostic et le correctif du pont d'identité (0
  contradiction sur 8 films mono-manche ; 4 manches identique octet pour octet ; additif strict
  sur 11 films ; le triplet refuse toute ambiguïté). Constats : deux gardes (mono-manche,
  « jamais contredire ») sans test qui morde ; justification fausse sur le slot 12 ; décision
  « pas de bump » sûre aujourd'hui pour une raison non écrite (0 artefact au schéma 39 sur la
  machine, prod au schéma 2) mais fragile dès le premier artefact 39 cuit avant la release (le
  backfill le sauterait) ; fichier de test à 562 L. DÉCISION SUPERVISEUR : bump `SchemaVersion`
  39 → 40 avec chronique (la règle du dépôt : un artefact vN doit se voir comme à re-cuire) ; l'étape
  de release « re-cuisson du parc » passe à 40 (registre + Notion à mettre à jour). Corrections
  en cours chez l'enquêteur.
- 2026-09-06 13:15 : A-R2 : sept constats FERMÉS ; quatre défauts nés des corrections (N1 P2 :
  une lecture d'équipe en échec marquait le match « dérivé » — classe de C1 réintroduite ; N3
  jauge 0 sur cycle annulé ; N4 marque orpheline jamais ramassée ; N2 modes à > 2 camps, 4
  matchs sur 1 959, décision produit) + doc inversée dans la migration des positions → retouches
  en cours chez l'exécuteur A ; merge A ensuite (conflit attendu avec C sur
  `domain/title/registry.go` et `replayartifacts/artifacts.go`).
- 2026-09-06 13:40 : CTF corrections rendues (`7c85acf58` : deux gardes testées par mutation,
  justification du slot 12 vraie, SchemaVersion 40 + chronique, assertions déplacées) et CTF
  FUSIONNÉ dans `feat/v75`. Conflits sémantiques résolus par le superviseur : le test des calques
  d'objectif typé sur `replaydoc` (lot B), et les valeurs figées par F.1 remesurées au schéma 40
  (15 frags, 6 assistances, 1 capture, 1 vol = sommes des lignes des 7 pontés). Intégration wire,
  contrat et `generate-types` verts et sans diff. Intégrés : B, F, C, G, E, CTF.
- 2026-09-06 14:10 : A retouches rendues (`cb33d8ea8` : lecture d'équipe en échec = pas de marque,
  jauge non publiée sur horizon illisible, marques orphelines ramassées, docs inversées corrigées ;
  N2 modes > 2 camps consigné, décision produit pour le lot D) et A FUSIONNÉ dans `feat/v75`.
  Conflits résolus par le superviseur : `replayartifacts.Run` = porte de capability (C) PUIS
  compteur + `defer` rattrapage + `cuireLeCycle` (A) ; `racineDepot` en double (C dans
  `capability_test.go`, A dans `helpers_test.go`) → le fichier de A supprimé ; journal concaténé.
  Tests unitaires + intégration replayartifacts/persist/sync verts. Intégrés : B, F, C, G, E,
  CTF, A. Reste D.
- 2026-09-06 14:40 : D FUSIONNÉ dans `feat/v75` (seul conflit : journal ; le déplacement de
  `weaponSoundVariations.ts` sous `sound/` a absorbé l'en-tête du lot G automatiquement). Gates
  sur l'état fusionné : `tsc -b --force` 0, lint 0 erreur, `lint:colors` clean, frontière
  `7 <= plafond 7`, build OK, vitest 612 fichiers / 6 393 verts. LES SEPT LOTS ET LE CORRECTIF
  CTF SONT INTÉGRÉS : B, F, C, G, E, CTF, A, D. Reste : CI des derniers merges, balayage du parc
  (en cours), balayage « après » sur `feat/v75` intégré, décision `flagCarries`.
- 2026-09-06 15:20 : BALAYAGE « AVANT » RENDU (`feat/v2-balayage`, outil `cmd/replay-diff`, rapport
  `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`) : 119 matchs / 161 artefacts distincts / 19 schémas
  (1-38), 119/119 re-cuits à `f1c7b411f` (B+F), 0 échec, 36 min, pic 0,56 Gio, parc de référence
  intact. Régressions candidates : (1) actions d'objectif CTF non attribuées sur les 14 CTF du
  parc (297 actions, −20 captures) = le pont d'identité `d173b1a8c`, DÉJÀ CORRIGÉ (schéma 40) ;
  (2) grappin −10 à −40 % sur 16 matchs (coïncide avec les usages d'équipement du schéma 38,
  non tranché) ; (3) épisodes camo/surbouclier −1/−2 sur 11 matchs ; (4) un joueur perd toutes
  ses vies nommées sur 3 matchs. Tout le reste expliqué sur pièces (drapeau neutre schéma 35,
  véhicules fantômes fusionnés, identité des camps résolue à somme constante, bornes de scène
  corrigées d'un facteur 100). Piège de méthode : la première série de cuissons tournait SANS
  les faits (échappement bash `\$`), re-cuite après correction. Suite : balayage « APRÈS » sur
  `feat/v75` intégré (même outil) + instruction des candidates 2-4 (worktree
  `LevelUp-wt-v2-regressions`).
- 2026-09-06 15:40 : CI de `feat/v75` rouge sur les trois derniers merges, deux causes distinctes :
  (1) merges CTF et A : job Frontend, garde `replaySchemaLogic.guard.test.ts` (copie web de
  `SchemaVersion` à 39 contre 40) — résolu par le merge D qui supprime ce fichier (item D.4) ;
  (2) merges A et D : job Coverage + Baseline, 9 entrées de baseline pour `TestLireT0FilmArtefact*`
  (`sync/replayartifacts`) enrôlées par F et supprimées par A avec leur fonction (dérivation T0
  refondue en A.4) → entrées retirées de `.ai/baselines/tests_pre_migration.jsonl` (9 795 → 9 786),
  suppression volontaire documentée. Aucun test en échec dans la suite Go.
- 2026-09-06 13:50 : CI encore rouge sur `beeb6f3ee` : `TestEnTeteTSVersionneeSuitLeGabarit`
  (lot G) lisait `weaponSoundVariations.ts` à l'ancien emplacement, déplacé sous `sound/` par le
  lot D (D.11) — troisième conflit sémantique entre lots parallèles ; chemin de sortie du mode
  `livrer`, test d'en-tête et golden alignés sur `features/match-replay/sound/`.
- 2026-09-06 14:10 : BALAYAGE « APRÈS » RENDU (`feat/v2-balayage` `4c8de8a05`, fusionné) : 119/119
  re-cuits au schéma 40 (34,8 min, pic 0,54 Gio, parc intact). (a) cuisson `f1c7b411f` contre
  cuisson intégrée : 491 écarts, ZÉRO perte, ZÉRO disparition — `schemaVersion` 39 → 40 sur les
  119 et +438 actions d'objectif retrouvées sur 17 matchs (captures +23, vols +20), rien d'autre :
  100 matchs sur 119 strictement identiques hors numéro de schéma — preuve indépendante que le
  lot E est à comportement identique, que A ne touche pas le document et que D ne touche que le
  web. (b) référence contre intégré : candidate 1 RÉSOLUE (0 perte nouvelle, 300 pertes
  résorbées, neuf familles à zéro perte) ; candidates 2-4 inchangées (instruction en cours).
  Verdict : le parc peut être re-cuit au schéma 40 sans rien y perdre, sous réserve des
  candidates 2-4.
- 2026-09-06 15:00 : INSTRUCTION DES CANDIDATES 2-4 RENDUE (`feat/v2-regressions` `79bf2e6d2`) :
  UNE SEULE cause racine, différente de celle soupçonnée par le balayage — `48cf4905d`
  (2026-09-02, schéma 36, « une track = une vie ») a découpé les pistes par vie et trois
  consommateurs ont continué de supposer une piste par slot, ne gardant que la dernière : vies
  nommées (`owners.go`, désignation de la vie fermée jetée), grappin (`grapple_lines.go`,
  `byTrack[slot]`), fenêtres camo/surbouclier (`trackFrameWindows`). Régressions corrigées
  (index par piste, `closureReport.closedLife`), sept tests prouvés par mutation, SchemaVersion
  40 → 41, `closures.go` scindé ; `13d92593` reste à 0 à raison (épisode nul sur point aberrant
  supprimé). Impact parc : grappin 16 matchs / 54 tractions, épisodes 11 / 17, vies nommées 18
  matchs. Revue adverse REG-R1 lancée (additivité, quatrième consommateur, mutations) avant merge
  et balayage final au schéma 41.
- 2026-09-06 15:40 : REG-R1 : diagnostic confirmé (lignes fautives nommées dans `owners.go`,
  `grapple_lines.go`, `equipment_episodes.go`), correctif additif et sûr sur 4 films cuits (0 perte,
  0 déplacement, pont byte-identique), 7 mutations tiennent, 22 conditions. Cinq constats renvoyés :
  C1 moyen (une traction dont tir et accroche tombent hors de toute fenêtre est jetée alors que
  la base la publiait, même mono-vie → ne jamais publier moins que la base), C2 (`camoLives` /
  `overshieldLives` comptent des slots), C3 (« scission pure » fausse : `noteLife` ajouté dans
  le bloc déplacé), C4 (commentaire contradictoire sur `13d92593`), C5 (`usage_summary.go`
  attribue par slot « dernier gagnant » : quatrième consommateur, corrigé par piste).
  Corrections en cours ; ensuite ronde 2, merge, balayage final au schéma 41.
- 2026-09-06 16:00 : corrections REG rendues (`13c0336b6`) : C1 `lifeNearest` (jamais moins que la
  base, deux scénarios figés), C2 compteurs par vie, C3/C4 docs vraies, C5 `usageOwners.at(slot,
  frame)` par vie couvrante + `UsageSummaryRev` us1 → us2 (résumés d'usage à refaire au backfill ;
  divergence Go/web sur un slot à deux identités inscrite au registre). Cuisson de contrôle
  `879a4dba` identique à l'octet hors numéro de schéma. REG-R2 lancée ; puis merge, balayage
  final au schéma 41, Notion (40 → 41).
- 2026-09-06 16:30 : REG-R2 : C1-C4 FERMÉS, C5 PARTIEL (les poses d'équipement retombent sur le
  repli « dernier occupant » dans 32 à 95 % des cas, un lâcher à la mort crédité au joueur suivant
  sur un slot repris ; `avecVie` construit depuis `dernier` perd les lancers d'un joueur à vie
  unique sur slot repris — préexistant) + trois commentaires absolus et une condition de reprise
  fausse au registre. Cuissons : `879a4dba` identique à l'octet hors numéro, `4f77afc1` 28 → 31
  tractions sans perte. Dernières retouches en cours (N-3, N-4 par vie couvrante ou adjacente,
  docs) ; puis merge, balayage final au schéma 41, Notion.
- 2026-09-06 17:00 : retouches REG rendues (`5d9f70f92` : poses par `atOrJustBefore`, `avecVie` sur
  toutes les vies, commentaires exacts, registre corrigé) et REG FUSIONNÉ dans `feat/v75` sans
  conflit (SchemaVersion 41, UsageSummaryRev us2). Gates rejoués sur l'état fusionné. Suite :
  balayage final au schéma 41, Notion 40 → 41, nettoyage des worktrees.
- 2026-09-06 17:20 (correctif de journal) : la ligne de 17:00 annonçait le merge REG « sans
  conflit » avant qu'il ait eu lieu — la première tentative n'avait rien fusionné ; le merge réel
  est `b696c7b11` (conflit thought_log concaténé), SchemaVersion 41 vérifié sur pièces, build,
  tests unitaires et intégration verts, poussé. Notion : re-cuisson du parc portée à 41. Balayage
  FINAL au schéma 41 lancé (`apres3/`). Intégrés : B, F, C, G, E, CTF, A, D, balayage, REG.
- 2026-09-06 18:10 : BALAYAGE FINAL (schéma 41) RENDU ET FUSIONNÉ : 119/119, 37,9 min, pic 0,56 Gio.
  40 → 41 : zéro disparition, +52 tractions (16 matchs), +28 vies nommées (18), +15 épisodes (9),
  +12 intervalles de drapeau (2 matchs : `flagCarries` était un CINQUIÈME consommateur du défaut,
  bénéficiaire du correctif), douze axes strictement identiques. Référence → 41 : candidates 1,
  2, 4 à ZÉRO, candidate 3 à 2 (`13d92593` connu, `2cf24f30` s31 7 → 6 résidu réel). Trois
  passes : pertes distinctes 1 783 → 1 483 → 1 160, ZÉRO perte nouvelle (sous-ensembles stricts).
  Restent quinze faits, tous antérieurs au chantier, dont `d9781168` (s23) −6 portages de crâne
  d'Oddball (36 → 30), à instruire à part. VERDICT : le parc peut être re-cuit au schéma 41.
- 2026-09-06 17:45 : INTÉGRATION FLAGCARRIES : `feat/v2-flagcarries` (`9ab4436a9`) fusionné dans
  `feat/v75` sans conflit (merge `e9adb36f5`), SchemaVersion 42 vérifié sur pièces. Revue FLAG-R1 :
  aucun constat recevable, 11 conditions tiennent, trois cuissons témoins `changements = 0` (0
  portage perdu ni déplacé), accord joueur par joueur 33/33 avec le calque des actions. Gates
  rejoués sur l'état fusionné : build, tests unitaires (`analysis/replay`, `replaybuild`,
  `archlint`, `contracttest`, `sync/replayartifacts`), intégration `api/wire`, contrat OpenAPI
  régénéré identique — tous verts. Ouverts au registre : CTF multi-manche (`fb1a1a72`, le pont
  par manche ne tient pas) et calques VIP/crâne (même plafond, patron désormais posé). Suite :
  intégration `feat/v2-residus` (schéma 43) après revue RES-R1.
- 2026-09-06 18:24 : INTÉGRATION RESIDUS : `feat/v2-residus` (`cd5302ebf`) fusionné dans
  `feat/v75` (merge `dd8004e90`), conflits admis résolus (chronique des schémas 42→43 dans
  `document.go` et `structure_test.go`, golden `assembly_000d5950.golden` régénéré par
  `-update`, `thought_log.md` concaténé). SchemaVersion 43 vérifié sur pièces. Revue RES-R1 :
  17/17 conditions, trois constats mineurs corrigés (`cd5302ebf`). Une régression corrigée par
  abstention : le gate de présence des porteurs de crâne (`af89b091b`) traitait une vie
  anonyme comme une absence ; `carrierPresence` retient désormais les vies ANONYMES, le gate
  ne rejette plus que ce que les pistes publiées DÉMENTENT. Deux témoins Oddball (vérité
  terrain = temps de portage) : `d9781168` feuille 191 s / 196 s, artefact schéma 41
  60,1 s / 147,4 s, corrigé 172,5 s / 158,8 s (quasi la feuille) ; `51ebbc0f` 66 s → 225 s.
  Quatorze autres faits résiduels : anciens artefacts faux, reclassements ou normalisations.
  Gates rejoués sur l'état fusionné : build, tests unitaires (`analysis/replay`,
  `replaybuild`, `archlint`, `contracttest`, `sync/replayartifacts`), intégration `api/wire`,
  `golangci-lint --new-from-merge-base=origin/main` (0 issue), contrat OpenAPI régénéré
  identique — tous verts. Ouverts au registre : `51ebbc0f` découpe par manche des compteurs
  par joueur (63 assistances pour 5, déjà au `REGISTRE_REPORTS.md`) et l'angle mort du
  comparateur sur les intervalles rognés.
