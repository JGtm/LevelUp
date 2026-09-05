# PLAN — Intégration des branches actives dans l'architecture cuisson-perf, puis merge et push de feat/v75

> Date : 2026-09-05. Branche `wt/cuisson-perf` (worktree `LevelUp-wt-cuisson-perf`), HEAD de départ
> `31b754363`. Exécution sous le contrat du skill `plan-execution` (ordre strict, aucun report
> d'un item exécutable, statut par item, vérification sur pièces — rappel opposable au §7).
> Pilote : cette session (orchestrateur-vérificateur) ; exécution et revue par agents Opus
> (conflits, balayages) ou Sonnet (tâches bornées) ; le pilote commite (jamais de stash, jamais de
> hook contourné). Relu par `plan-review` le 2026-09-05 (verdict : exécutable après corrections —
> toutes fondues ici ; l'addendum `*_REVUE.md` est supprimé à la clôture de B).

## §0 Mandat (utilisateur, 2026-09-05)

« Le sujet optimisation doit reprendre le travail des véhicules et du translocateur/répulseur et
des nouvelles stats pour l'intégrer proprement à son refacto (branches actives non mergées avec
feat/v75). Quand ce sera fait tu mergeras dans feat/v75 [...] et quand tu auras fini, tu push et
vérifieras que tout est vert, même si ce n'est pas de ta faute. » Contraintes ajoutées : « propre,
solide et pérenne, si jamais tu dois faire des ajustements » ; « si tu peux orchestrer plusieurs
étapes en parallèle, go » ; **`feat/v75` est GELÉ par l'utilisateur pour la durée de
l'intégration** (« C'est gelé, je touche plus à rien », 15 h).

## §1 État des lieux (mesuré le 2026-09-05, 14 h 30)

| Branche | Avance / retard sur feat/v75 | Base | Schéma | Périmètre | Décision |
|---|---|---|---|---|---|
| `feat/v75` (origin `7fb4b60a1`) | — (65 commits depuis notre dernière réconciliation `ceabaad67`) | — | **38** | translocateur, propulseur, charges d'équipement (3 balayages ANCIENNE forme appelés dans `build.go:308/457/473`), vue match, himodule mmap, CI | **Étape A** — FAIT (`eb80a4f0a`) |
| `origin/feat/v75-vehicules-sons` (28/339) ∪ `wt/vehicule-deadstate` (25/339 : 4 DERRIÈRE `sons` + 1 devant, `0b5141b8a`, tests + notes) | voir cellule | `14a115bb1` (base commune des deux têtes : `f4c3ed417`) | **31** (bump propre à la branche) | 65 fichiers filmdec + 14 replay non-test, **8 balayages ancienne forme** (5 sites de production dans `build_vehicles.go`), `build.go` (+Vehicles, attachVehicles, attachVehicleShots, buildShots à 3 retours), `document.go` +97, web match-replay ×50, sons ×66, assets ×20, 3 outils cmd, `openapi.yaml` +282 et `generated.ts` +132 (GÉNÉRÉS) | **Étape B** — le gros morceau |
| `wt/session-usage` (10) | 10 / 22 | `da616828f` | 38 (inchangé) | paquet `sessionusage`, `replay/usage_summary*.go` (projection du DOCUMENT, aucun balayage), persister INSERT-only + 2 tables append-only neuves + vues `_latest`, migration, capability `film.usage_summary`, crochet dans `replayartifacts/{cuisson,artifacts,journal}.go` + `usage.go`, CLI `backfill-usage-summary`, killcollector (classifier, postsync), web session-detail (~2 100 l.), `openapi.yaml` +307 et `generated.ts` +140 (GÉNÉRÉS) | **Étape C** |
| `wt/game-changers` (4) | 4 / 15 | `ca55f0ed7` | — | web uniquement (match-replay ×12, ui ×3) | **Étape D** |
| `wt/assaut-stats` (3) | 3 / 34 | `146f1d92e` | 38 | `replay/bomb_stats.go`, `bomb_arms.go` + tests de recherche ; **aucun appelant dans tout go-api** ; worktree avec 6 fichiers NON commités (recherche du 04/09) | **[!] non intégrée** (D5) |
| `wt/r9-repulseur`, `wt/r12-repulseur`, `wt/son-propulseur`, `wt/lint-filmdec`, `wt/himap-mmap`, `wt/trame-film` | 0 | — | — | déjà dans feat/v75 | arrivées par l'étape A |
| `main` | 9 commits hors feat/v75 (hotfix 7.3.1 classement mondial) | — | — | — | hors mandat — signalé à l'utilisateur |

Corpus du harnais : 13 films (`internal/analysis/replay/testdata/equivalence/CORPUS.txt`),
**48 étapes observées depuis A** (45 + `translocations`, `abilityImpulses`, `abilityCharges`),
marqueur `# digest-grammar: 2`. Intersections web mesurées : D ∩ B = `match-replay/i18n.ts`,
`i18nContract.ts` ; D ∩ C = `match-replay/MatchEquipmentUsageSection.tsx` ; B ∩ C = les deux
fichiers générés + `lib/api/types.ts` + `thought_log.md`.

## §2 Décisions

- **D1 — Ordre logique** : A (amont) → B (véhicules) → C (usage de session) → D (game-changers) →
  E (gates + revue de branche) → F (merge, push, CI). L'ordre est confirmé par la relecture (le
  recouvrement web B↔D tient en deux fichiers i18n, B↔C en fichiers générés). Voir D10 pour
  l'exécution PARALLÈLE de B, C, D.
- **D2 — Rejouer, jamais réintroduire** : tout balayage en ancienne forme (`ScanFilm*(dir)`)
  arrivant d'une branche se réécrit en `Scan*(film, ...)` lisant `film.Packets(i)` et le
  `FilmContext` (bande de slots, layout catalogue, registre) ; une enveloppe `(dir)` n'existe que
  pour les tests qui l'appellent (liste fermée d'archlint) et se supprime sans appelant. Chaque
  nouveau balayage de production reçoit son étape observée (`BuildFromFilmSteps` + `opt.observe`).
  Les garde-rails (observe_test, no_film_reread, filmsource_leaf, no_recomputed_film_context,
  ratchet des vars, liste des enveloppes) sont la checklist : un rouge = un oubli, on corrige dans
  le sens de l'intention de la branche, jamais en élargissant une allowlist sans justification datée.
- **D3 — Un seul bump de schéma, et le numéro n'est pas acquis** : A a absorbé le 38 amont ; B porte
  le **39** (champs véhicules SUR le 38 ; le 31 de la branche et ses trois blocs de doc de version
  29/30/31 fondus en un seul bloc 39) ; C et D ne bumpent pas. `feat/v75` a bumpé sept fois en trois
  jours et a subi une renumérotation par collision (`6ce0fcc2a`, 29 → 30) : (a) le 39 est ANNONCÉ à
  l'utilisateur à l'ouverture de B (fait, 15 h — et l'amont est gelé) ; (b) F.1 re-vérifie
  `const SchemaVersion` sur `origin/feat/v75` AVANT le merge ; si ≥ 39, notre bump devient N+1 et
  les 13 TSV sont re-figés puis re-vérifiés (le numéro entre dans le digest `artifact`).
  **Complément D13 (16 h 20)** : le travail non commité d'Assaut porte LUI AUSSI `38 → 39`
  (armements pausables au document). Les deux 39 se FONDENT en un seul à G.1 (aucun des deux
  n'est déployé) : un seul bloc de doc de version 39 = véhicules + armements/portage ; les TSV
  sont re-figés à G.1 en changement déclaré (films d'Assaut du corpus), le numéro ne bouge pas.
- **D4 — Preuve par le harnais à chaque étape** : passe complète SANS `-update` d'abord ; la
  comparaison du harnais est POSITIONNELLE (`parent.go`, ligne i contre ligne i) : quand une étape
  s'insère, le diff exploitable est **PAR NOM D'ÉTAPE** (copie des TSV avant / après). Chaque écart
  doit être **corrélé** à l'intention de la branche (quels films, quelle étape, pourquoi ces
  films-là) — un écart inexpliqué = arrêt et rapport ; puis `-update`, puis vérification 13/13 ;
  diff des comptes au §5. Une ligne AJOUTÉE n'est pas un écart, une ligne MANQUANTE l'est : gate de
  lignes (48 après A, **49 après B**, partout). Une étape sans changement attendu (C, D) rend 13/13
  identiques SANS `-update` — tout écart y est une ANOMALIE (le diff Go de C sur le document est un
  renommage pur `padFamilyKey` → `PadWeaponFamilyKey`).
- **D5 — `wt/assaut-stats` n'est pas intégrée** : `BuildBombStats`/`BuildBombArms` n'ont aucun
  appelant hors de leurs tests (imports `sort`, `strconv`, `objectiveevents` seulement : bibliothèque
  non câblée — table `match_bomb_stats`, capability `film.bomb_stats` et écriture au sync restent à
  faire), **aucune dépendance au refacto** (post-décodage, rien à rejouer), et le worktree porte de
  la recherche non commitée. L'intégrer poserait du code mort dans feat/v75 (anti-pattern n°1) ; la
  branche mergera à la clôture de son chantier (nuance : elle modifie aussi
  `bombe_portage_gate_test.go`, 12 l. — pas strictement additive). Ajustement signalé.
- **D6 — Champ `structure` de l'artefact : hors périmètre** — quatre sites web subsistent, dont un
  lecteur métier (`features/match-replay/replayLogic.ts:274` repli des bornes ;
  `replayNormalize.ts:131/250/407`). Retrait = décision produit + web + schéma : au registre des
  reports.
- **D7 — mmap** : la projection mémoire vit dans `internal/himodule/projection_*.go` (modules de
  cartes), pas dans la cuisson des films ; `filmsource` lit des chunks zlib de quelques dizaines de
  Ko : aucun gain à attendre. Pas d'action ; consigné.
- **D8 — Merge final et CI** : fetch frais ; contrôle D3(b) ; si `feat/v75` a bougé malgré le gel,
  mini-réconciliation (protocole A) AVANT le merge ; merge dans le worktree partagé uniquement s'il
  est propre ; `git push origin feat/v75` (jamais `main`) ; CI verte **au niveau JOB sur la liste
  MESURÉE au push** (`gh run view --json jobs`) : `Frontend`, `Go Build + Test` (matrice), `Go Lint`,
  `Go Lease Enforcement`, `Go Coverage + Baseline`, `OpenAPI Lint`, `Go Contract Test`, et — parce
  que C touche `internal/persist/**` et `internal/platform/duckdb/**` — `ADR 0021 Gate —
  shared_social invariants` (workflow séparé, déclenché par paths). `E2E React (Playwright)` ne
  tourne PAS sur un push (`if: pull_request`) : son absence n'est pas un rouge. Un rouge
  préexistant se corrige aussi (mandat).
- **D8bis — Re-mesure de l'amont au début de chaque étape** (même gelé) : `git fetch`, puis commits
  depuis A, `const SchemaVersion`, et `git diff eb80a4f0a..origin/feat/v75 | grep -oE
  'ScanFilm[A-Za-z]+\('`. Delta non nul = mini-réconciliation IMMÉDIATE. Mesure écrite au §6, même
  nulle.
- **D8ter — Ce que la prod sert entre le push et la re-cuisson, écrit et assumé** : le 39 périme tous
  les artefacts (`cmd_backfill_replay.go:79`) MAIS aucune re-cuisson de masse ne part seule — le
  rattrapage du fil de l'eau sélectionne par `os.Stat` (présence du fichier), pas par schéma
  (`replayartifacts/backlog.go`). Corpus MIXTE 38/39 : calque véhicules absent sur l'historique,
  présent sur les matchs cuits après déploiement ; le front lit `schemaVersion` comme un nombre
  (`replaySchemaLogic.ts`) — dégradation gracieuse. Assumé jusqu'à l'accord utilisateur sur
  `levelup backfill-replay --only-existing` (jamais lancé d'office).
- **D9 — Union des deux branches véhicules** : les deux têtes sont DISJOINTES hors `thought_log.md`
  (base commune `f4c3ed417` ; `0b5141b8a` = 2 tests neufs + 2 notes) : l'ordre ne change pas le
  contenu. `origin/feat/v75-vehicules-sons` D'ABORD parce que ses 4 commits de CI réparent le lint
  `unused` et deux gardes archlint (état intermédiaire vert), puis `wt/vehicule-deadstate` (trivial).
- **D10 — Exécution PARALLÈLE de B, C, D en worktrees dédiés** (demande utilisateur, 15 h 40) :
  trois worktrees dérivés du commit A `eb80a4f0a` — `LevelUp-wt-integ-vehicules`
  (`wt/integ-vehicules`), `LevelUp-wt-integ-usage` (`wt/integ-usage`), `LevelUp-wt-integ-gamechangers`
  (`wt/integ-gamechangers`) — jonctions `data/cache/{film_chunks,film_manifests,mvar,replays}` vers
  le cache principal, `LEVELUP_REPO_ROOT` pointé sur le worktree (pas de `db_profiles.json`),
  **`GOCACHE` propre à chaque worktree** (`<wt>/tmp/gocache` — des builds Go concurrents sur un
  cache partagé l'ont déjà corrompu), verrou solo de cuisson partagé par construction (le cache est
  commun : les cuissons se sérialisent, c'est voulu). Chaque agent commite dans SA branche
  d'intégration (commit final de l'étape, autorisé par le pilote pour ces branches jetables) ; le
  pilote FUSIONNE ensuite séquentiellement dans `wt/cuisson-perf` : B, puis C, puis D — conflits
  attendus et bornés (§1 intersections : fichiers générés → regénération D11 ; `i18n.ts` → union
  des clés ; `MatchEquipmentUsageSection.tsx` → les deux intentions ; `thought_log.md` → union) ;
  après chaque fusion : build, tests des paquets touchés, harnais SANS `-update` (attendu : les TSV
  de B) ; puis E sur le résultat fusionné. Les trois worktrees et branches d'intégration sont
  supprimés à la clôture de E.
- **D11 — Contrat OpenAPI : `openapi.yaml` et `generated.ts` sont GÉNÉRÉS** (`openapi.yaml` porte
  l'en-tête « NE PAS ÉDITER À LA MAIN — `make openapi-gen` — verrouillé par
  `TestOpenAPIYAMLIsUpToDate` »). B et C les modifient tous deux. Un conflit de merge sur l'un des
  deux ne se résout JAMAIS à la main : on prend `--ours` (indifférent), puis on REGÉNÈRE et on
  committe le résultat de la machine : `make openapi-gen && make generate-types && make
  openapi-check`. Gate (B, C, et chaque fusion D10) : `go test ./internal/api/ -run
  TestOpenAPIYAMLIsUpToDate -count=1` · `CGO_ENABLED=0 go test ./contracttest/... -run TestContract
  -count=1` · `node tools/check-generated-types-fresh.mjs` (aucun job CI ne vérifie la fraîcheur de
  `generated.ts`). Si `make openapi-gen` produit un diff APRÈS résolution, la résolution Go était
  incomplète : corriger le handler/DTO, jamais le YAML.
- **D12 — Gate web = ce que fait le job CI `Frontend`, pas moins** (B, C, D, chaque fusion, E) :
  `npm run typecheck && npm run lint && npm run lint:fields` · `npm run test` (vitest COMPLET) ·
  `npm run build` · `node tools/knip-ratchet.mjs` (ratchet code mort — il casse typiquement à la
  réconciliation de plusieurs branches ; un export neuf non consommé se câble ou se supprime,
  JAMAIS relever le plafond).

- **D13 — Stats d'Assaut : D5 est RENVERSÉE par l'utilisateur (16 h 10)** : « tu peux quand même
  traiter ce point ». État réel mesuré : la branche COMMITÉE ne porte que le noyau pur (E1/E2),
  mais le worktree porte 29 fichiers modifiés + 8 non suivis NON COMMITÉS (E2-bis, E2-ter, E3 :
  migration `match_bomb_stats`, `BombStatsPersister`, câblage `BatchBuilder`, durcissements ART —
  « Complétés » au journal du 04/09, gates passés selon lui, jamais commités : la session s'est
  arrêtée avant). Rien n'est écrit en base, rien n'est servi, rien n'est affiché : la
  fonctionnalité N'EST PAS opérationnelle aujourd'hui. Reprise en Étape G (après les fusions D10,
  avant E) : sécuriser le travail non commité, intégrer, puis exécuter E4 (crochet au sync), E5
  (API + web), E6 (backfill, clôture) du plan Assaut, qui fait foi pour les décisions tranchées.

## §3 Étapes

### Étape A — Réconciliation amont (`feat/v75` `7fb4b60a1` → `wt/cuisson-perf`) — CLOSE
- [x] A.1 Merge `origin/feat/v75` (`7fb4b60a1`, 65 commits, 324 fichiers, 5 conflits résolus
  « sémantique amont dans notre architecture »). PREUVE : `--diff-filter=U` vide, aucun marqueur.
  Détail au §6.
- [x] A.2 Les 3 balayages amont en nouvelle forme : `ScanTranslocatorTeleports(film, entry)`,
  `ScanAbilityImpulses(fc)`, `ScanAbilityCharges(fc)` ; enveloppes `(dir)` gardées pour leurs seuls
  tests, liste archlint **40 → 43** ; `BuildFromFilmSteps` **31 → 34**. Aucune allowlist élargie,
  ratchet des vars inchangé à 116.
- [x] A.3 Gates — gofmt vide · build 0 · vet 0 · 66 paquets ok · intégration `-p 1` exit 0 · lint
  281 issues, 0 sur un fichier touché.
- [x] A.4 Harnais — diff PAR NOM : 3 étapes neuves, `equipmentChanges` 13/13 (récupération gatée),
  `positions` sur EXACTEMENT les 2 films à translocations > 0, `artifact` 13/13 (37 → 38) ;
  `-update` ; vérification **13/13** ; **48 étapes par TSV**. Témoins 17,9 / 21,0 / 20,5 s.
- [x] A.5 Commit du pilote : **`eb80a4f0a`** (vérifications sur pièces : 0 conflit, 0 marqueur,
  0 `ScanFilm*(filmDir` dans `build_from_film.go`, 48 lignes × 13 TSV, build/vet/tests verts).

### Étape B — Véhicules (union D9) — schéma 39 — worktree `LevelUp-wt-integ-vehicules`
- [ ] B.0 D8bis (re-mesure amont) ; `git merge --no-ff --no-commit origin/feat/v75-vehicules-sons`
  puis, une fois résolu, `wt/vehicule-deadstate`.
- [ ] B.1 Conflits (base à 339 commits : `build.go` → nos `build_from_film.go`/`options.go`,
  `document.go` (bloc de version unique 39), `registry.go`, `traverse.go`, `offline_biped.go`,
  `event_list.go`, `frame_records.go`, `keyframe_*`, `equipment_creation.go`, `unit_weaponstate.go`,
  tests, web match-replay, `.ai`) résolus « sémantique véhicules dans notre architecture ET sur le
  schéma 38 amont ». Fichiers générés : D11.
- [ ] B.2 CINQ SITES DE PRODUCTION dans `build_vehicles.go`, un par case, aucun autre :
  - [ ] B.2.a `:84` `ScanFilmWorldObjectKeyframes(dir, 40)` → `ScanWorldObjectKeyframes(film,
    filmdec.VehicleTypeIndex)` (la forme neuve EXISTE : `filmdec/world_object_census.go:79` —
    remplacement d'appel)
  - [ ] B.2.b `:90` → `ScanVehicleCreationsForBand(film, wr, band)` (+ enveloppe `(dir)` hors prod)
  - [ ] B.2.c `:96` → `ScanBipedPositionsForBand(film, band, opt)` (+ enveloppe `(dir)` hors prod ;
    n'existe pas encore sur cuisson-perf)
  - [ ] B.2.d `:125` → `ScanBipedAimOnly(film)` (+ enveloppe `(dir)` hors prod)
  - [ ] B.2.e `:139` → `ScanVehicleEvents(film)` (+ enveloppe `(dir)` hors prod)
  - [ ] B.2.f `ScanKeyframeRecordSpans`, `ScanVehicleCreations`, `ScanVehicleOccupancy` : formes
    neuves + enveloppes `(dir)`, appelants NON production uniquement.
  - [ ] B.2.g `decodeFilmVehicleScan(filmDir, ...)` → `decodeVehicleScan(film, fc, ...)` : UN seul
    appel dans `BuildFromFilm`, UNE seule étape observée `vehicles` ; `attachVehicles`,
    `attachVehicleShots`, `buildShots` restent dans `BuildFromPositions` (pas d'étape : régime de
    `bots`/`successions`, couverts par `artifact`).
  - [ ] B.2.h ENVELOPPES OBLIGATOIRES (appelants mesurés dans les tests v13 de
    `wt/vehicule-deadstate`) : `ReadFilmChunk`, `CountFilmChunks`, `ScanFilmBipedPositions`,
    `ScanFilmBipedPositionsForBand`. Toute enveloppe sans appelant après merge se supprime.
- [ ] B.3 `SchemaVersion = 39` (une fois, commentée : véhicules sur le 38, décision D3) ; document :
  champs véhicules ; D11 (regénération, gates OpenAPI).
- [ ] B.4 Gates Go (comme A.3, `GOCACHE` propre) + D12 (web complet) + `go test ./internal/api/ -run
  TestOpenAPIYAMLIsUpToDate`.
- [ ] B.5 Harnais (D4, diff PAR NOM). ÉCARTS ATTENDUS SUR LA LISTE RÉELLE DES ÉTAPES : une étape
  NEUVE `vehicles` (seule greffe dans `BuildFromFilm`) ; `artifact` bouge sur 13/13 (numéro de
  schéma) et davantage sur les films à véhicule ; AUCUNE autre étape ne bouge. `shots` et
  `coverage` NE SONT PAS des étapes observées — ne pas leur en créer. Gate de lignes : **49
  partout**. Témoin obligatoire : un BTB du corpus (`084a804d`). Corrélation : les films dont
  `vehicles` est non vide sont exactement ceux dont l'artefact grossit au-delà du plancher.
- [ ] B.6 Commit de l'agent dans `wt/integ-vehicules` ; puis fusion D10 par le pilote dans
  `wt/cuisson-perf`.

### Étape C — Usage de session (`wt/session-usage`) — worktree `LevelUp-wt-integ-usage`
- [ ] C.0 D8bis ; `git merge --no-ff --no-commit wt/session-usage`.
- [ ] C.1 Conflits `replayartifacts/{cuisson,artifacts,journal}.go` : le crochet de projection se
  rejoue dans `cuireUnMatch`/le bilan du lot 5 (projection APRÈS rangement, lecture du fichier
  rangé, writer court APRÈS toute cuisson — doctrine de `usage.go`) ; `killcollector`,
  `no_art_patterns_test.go` (durcissement daté 2026-09-04 : garder), migration ; fichiers générés :
  D11.
- [ ] C.2 Vérifier sur pièces : persister INSERT-only, tables append-only avec vues `_latest` (ordre
  `films_latest` avant `players_latest`), capability et non `slug ==`, aucune lecture brute des
  tables append-only.
- [ ] C.2bis Recette ADR 0026 complète : statuer si `match_usage_players`/`match_usage_films`
  doivent entrer dans `internal/migration/append_only_rebuild.go` (la branche ne le fait pas) —
  l'ajouter, ou écrire pourquoi non au §4.
- [ ] C.3 Gates Go + intégration `-p 1` (dont `usage_integration_test.go`) + D11 + D12 (web complet
  — `usageLogic.ts` 486 l. et `SessionUsageSection.tsx` 417 l. passent ; ne pas « réparer » hors
  périmètre, noter au §4 si une résolution franchit 500).
- [ ] C.4 Harnais : **13/13 identiques SANS `-update`** — gate ferme, tout écart est une anomalie.
- [ ] C.5 Commit de l'agent dans `wt/integ-usage` ; fusion D10 par le pilote.

### Étape D — Game-changers (`wt/game-changers`, web) — worktree `LevelUp-wt-integ-gamechangers`
- [ ] D.0 D8bis ; `git merge --no-ff --no-commit wt/game-changers`.
- [ ] D.1 Conflits MESURÉS : avec l'amont A éventuellement ; `.ai/thought_log.md` en union. (Les
  croisements avec B — `i18n.ts`, `i18nContract.ts` — et avec C — `MatchEquipmentUsageSection.tsx` —
  se résolvent À LA FUSION D10, pas ici.)
- [ ] D.2 D12 (web complet). Harnais : `[~]` rien côté Go — vérifié par `git diff --stat -- apps/go-api`
  VIDE, sinon la case redevient `[ ]`.
- [ ] D.3 Commit de l'agent dans `wt/integ-gamechangers` ; fusion D10 par le pilote.

### Étape G — Stats d'Assaut : REPRISE du chantier `wt/assaut-stats` et mise en service (D13)
Exécutée APRÈS les trois fusions D10 (elle crochète la même zone que C : `replayartifacts`,
`persist`), AVANT E. Plan de référence : `.ai/V7.5/PLAN_ASSAUT_STATS_2026-09-04.md` (sur la
branche, il fait foi pour les décisions §2 : table dédiée `match_bomb_stats`, append-only +
`_latest`, INSERT-only via `BatchBuilder`, événements datés dans `match_objective_events`,
capability `film.bomb_stats`, désamorçage HORS LOT, aucune cuisson en lot).
- [ ] G.0 Sécuriser le travail NON COMMITÉ de l'autre session dans `LevelUp-wt-assaut-stats`
  (29 fichiers modifiés + 8 non suivis = lots E2-bis, E2-ter, E3 « Complétés » au journal mais
  jamais commités) : vérification sur pièces (build, vet, tests) puis commit SUR `wt/assaut-stats`
  au nom du lot, sans rien y changer.
- [ ] G.1 Merge `wt/assaut-stats` dans `wt/cuisson-perf` (fusionnée B+C+D) ; conflits attendus :
  `document.go` (champs armements/portage sur le 39), `persist/{batch,builder,combined_persister}.go`
  (C a ajouté le persister d'usage au même endroit), `no_art_patterns_test.go` et
  `append_only_state_guard_test.go` (deux durcissements datés : union), `migration/order.go`,
  `replaybuild/{matchfacts,zones}.go`, journaux. Harnais : écart déclaré si le document bouge
  (armements pausables, `assembly_000d5950.golden` a bougé sur la branche) — corrélé aux films
  d'Assaut du corpus, sinon anomalie.
- [ ] G.2 E4 — branchement au sync : dans `replayartifacts`, projeter les stats bombe du
  DOCUMENT RANGÉ (patron EXACT de `usage.go` arrivé par C : projection après rangement, lecture
  du fichier rangé, writer court après toute cuisson, via `persist`), gate par capability
  `film.bomb_stats` (TOML du titre), dégradation gracieuse (film absent, mode non-Assaut,
  capability absente → rien d'écrit, `slog.InfoContext` qui dit pourquoi), aucune erreur avalée,
  `maxPerCycle` et verrou inchangés. Test du crochet + intégration `-p 1`.
- [ ] G.3 E5 — lecture et API : repo `platform/duckdb` sur `match_bomb_stats_latest`, port, service
  (zéro SQL inline), handler Huma (zéro logique métier), capability → `ErrCapabilityNotSupported`
  en réponse partielle propre ; `make openapi-gen && make generate-types && make openapi-check`
  (D11).
- [ ] G.4 E5 — web : affichage dans la fiche de match sur le patron des autres modes à objectif
  (section objectifs, vues arrivées par l'amont), `useFieldLabel()`, i18n FR **et** EN, query key
  dans `lib/query/keys.ts`, tokens de couleur seulement ; D12 complet.
- [ ] G.5 E6 — backfill : `levelup backfill-replay` suffit-il (le crochet projette tout artefact
  cuit) ? sinon sous-commande dédiée sur le patron de `backfill-usage-summary` (C). JAMAIS lancé
  sur le parc. Registre des reports (désamorçage, condition « corpus portant un désamorçage
  avéré ») ; cases du plan Assaut statuées ; thought_log ; commit.

### Étape E — Filet complet et revue de branche (sur `wt/cuisson-perf` fusionnée)
- [ ] E.1 FILET COMPLET, pièce par pièce, un code de sortie consigné par ligne (EXIT_*=0 dans un log
  persistant, jamais un pipe) : `make gate-push` (son ratchet lint compare à `origin/main` : tout le
  delta v75 s'affiche, ce n'est pas une régression) · `gofmt -l .` vide · `go vet ./...` · `go build
  ./...` · `go test ./... -count=1` · `go test -tags=integration -p 1 ./internal/sync/... -count=1` ·
  `make openapi-check` · D12 complet · harnais 13/13 identiques SANS `-update` (49 lignes).
- [ ] E.2 Revue adversariale du diff d'intégration (`eb80a4f0a..HEAD`, contexte frais) ;
  corrections ; seconde lecture.
- [ ] E.3 Journal, registre des reports (D5, D6, D7), thought_log ; suppression des trois worktrees
  et branches d'intégration ; commit.

### Étape F — Merge, push, CI
- [ ] F.1 `git fetch` ; D3(b) numéro de schéma amont ; si `origin/feat/v75` ≠ `7fb4b60a1` :
  mini-réconciliation (protocole A) + harnais + gates ; commit.
- [ ] F.2 Worktree partagé propre → `git merge --no-ff wt/cuisson-perf` dans `feat/v75` ; sinon
  attendre/signaler.
- [ ] F.3 `git push origin feat/v75` ; suivi `gh run watch` ; tout job rouge (même préexistant)
  se diagnostique et se corrige ; re-push jusqu'au vert au niveau job sur la liste mesurée (D8).
- [ ] F.4 Rapport à l'utilisateur : ce qui est intégré, les écarts nommés, D5/D6/D7/D8ter, la
  proposition de re-cuisson de masse (jamais lancée d'office), la note sur `main`, le sort des
  branches absorbées (`origin/feat/v75-vehicules-sons` est publiée : proposer sa suppression ;
  `wt/cuisson-perf` poussée comme trace ou non — décision utilisateur).

## §4 Hors périmètre / découvertes
- N-1 `wt/assaut-stats` : D5 (+ nuance `bombe_portage_gate_test.go`).
- N-2 champ `structure` : D6 (quatre sites web).
- N-3 mmap himodule : D7.
- N-4 `main` a 9 commits hors `feat/v75` (hotfix 7.3.1) — à ramener dans feat/v75 un jour ; hors
  mandat.
- N-5 les trois outils `cmd/{weapon-sounds,vs-measure,vehicle-sprite}` de la branche véhicules
  arrivent tels quels ; à statuer par leur auteur (outillage de recherche vs production).
- N-6 la comparaison du harnais est positionnelle (§5 A) : une amélioration « diff par nom »
  de `cmd/replay-equiv` serait pérenne — hors périmètre de l'intégration, au registre.

## §5 Diffs des comptes (par étape)

### Étape A — réconciliation amont `7fb4b60a1` (2026-09-05)

**LA COMPARAISON DU HARNAIS EST POSITIONNELLE** (`cmd/replay-equiv/parent.go`, `comparer` : ligne
`i` contre ligne `i`), et l'étape `translocations` s'insère en **tête** de `BuildFromFilmSteps`
(le balayage tourne AVANT les positions, parce qu'il arme l'exemption du filtre de vitesse). La
passe 1 nomme donc « première étape en écart : `translocations` » sur les 13 films — un artefact
de décalage, pas un diagnostic. Le diff exploitable est celui **PAR NOM D'ÉTAPE**, entre la copie
des 13 TSV de référence prise avant refigeage et les TSV refigés. Il ne fait bouger que **six
noms**, et **aucune étape ne disparaît** :

| étape | films touchés | nature | cause amont |
|---|---|---|---|
| `translocations` | **13/13** | NOUVELLE | `filmdec/transloc_events.go` — l'événement 117 `EquipmentTranslocatorTeleportEffects`. Contenu NON VIDE sur 2 films seulement : `64e8adfa` (4 événements), `60ae07c4` (2) ; 0 sur les 11 autres |
| `abilityImpulses` | **13/13** | NOUVELLE | `filmdec/ability_impulses.go` — corps `tag==1` d'i57/i59 (usage du propulseur). Non vide sur 6 films : `000d5950` 86, `1c4c63c2` 42, `084a804d` 8, `64e8adfa` 7, `7344d24f` 2, `53ce4390` 1 |
| `abilityCharges` | **13/13** | NOUVELLE | `filmdec/ability_charges.go` — emplacements armés d'i56. Non vide sur 9 films (`084a804d` 148, `000d5950` 91, `1c4c63c2` 86, `53ce4390` 83, `a349fea8` 73, `64e8adfa` 35, `696a9d7c` 28, `7344d24f` 17, `01e1f945` 14) |
| `equipmentChanges` + `.stats` | **13/13** | CHANGE | RÉCUPÉRATION GATÉE des émissions i48 manquées (`filmdec/equipment_recovery.go`, neuf de l'amont) : `EquipmentChange` gagne `Recovered` et `Gap`, `EquipmentChangeStats` gagne `Recovered`. Le compte monte sur 8 films (`084a804d` et `a349fea8` +9, `000d5950` et `53ce4390` +2, quatre autres +1) et reste identique sur 5 — sur ces 5 le sha bouge quand même, **par les deux champs ajoutés au struct**, pas par une lecture qui change |
| `positions` | **2/13** | CHANGE | EXEMPTION DU FILTRE DE VITESSE par les téléportations (décision D2 amont) : `60ae07c4` 267 365 → 267 371 (+6), `64e8adfa` 293 811 → 293 823 (+12) |
| `artifact` | **13/13** | CHANGE (octets) | schéma **37 → 38** + les trois calques neufs et leurs couvertures. +364 à +7 769 octets |

**CORRÉLATION MESURÉE, PAS DÉDUITE** — c'est elle qui autorise le refigeage :

1. `positions` bouge sur **EXACTEMENT** les 2 films dont `translocations > 0`, et sur **AUCUN**
   des 11 films à `translocations == 0`. 13/13, zéro exception. Et le delta suit le nombre
   d'événements : 2 téléportations → +6 positions, 4 → +12 (3 enregistrements récupérés par
   téléportation, le même rapport sur les deux films). C'est l'exemption, et rien d'autre :
   sur un film sans tête 117 le filtre est bit à bit celui d'avant, ce que les 11 autres prouvent.
2. Les zéros des trois calques neufs sont des zéros **MESURÉS**, pas des balayages qui n'ont pas
   tourné : le journal des 13 films rend `composantAbsent=false` sur les 26 lignes
   (13 × impulsions + 13 × charges) — le composant est déclaré partout, il n'y a simplement eu ni
   impulsion ni charge sur ces matchs-là.
3. `artifact` : le plancher est **+364 octets**, et il tombe sur exactement les 3 films qui ont
   0 translocation, 0 impulsion ET 0 charge (`51101d1d`, `9f57c612`, `d9781168`) — c'est le coût
   fixe des calques vides, du numéro de schéma et des blocs de couverture. Les autres montent en
   proportion de ce qu'ils portent.

**AUCUN AUTRE DIGEST NE BOUGE** : `fire`, `loadouts`, `pickups`, `inventory`, `inventoryDeltas`,
`abilityRanks`, `camoStates`, `grappleReads`, `zoomEvents`, `placements`, `pads`, `carrierMarks`,
`zoneReads`, `bombReads`, `grenades`, `projectiles`, `deaths`, `playerIndices`, `clockOrigin`,
`score`, `objectives`, `flag`, `zones`, `killSource`, `neutralDeaths`, `spawnPoints` et les six
canaux delta sont **IDENTIQUES AU BIT PRÈS** sur les 13 films — la preuve que l'architecture du
chantier a traversé le merge sans dommage.

**GATE DE LIGNES (IMP-3 de la revue) : les 13 TSV portent 48 étapes, tous** (45 + 3), et
`minifilm.tsv` ses 7. Une ligne AJOUTÉE n'est pas un écart ; une ligne MANQUANTE le serait — et
`observe_test.go` rougirait avant le harnais.

**MINI-BOBINE (étage CI) : AUCUN REFIGEAGE.** `TestEquivalenceMiniFilm` ne couvre que sept étapes
(`fire`, `grenades`, `loadouts`, `inventory`, `deaths`, `playerIndices`, `projectiles`) : aucune ne
bouge, contrairement au 2026-09-03 où `fire` avait changé. Le test passe sur la référence existante.

**DURÉES ET PICS (passe de vérification)** : 5,1 s (`51101d1d`) à 1 min 58,7 s (`1c4c63c2`) ;
pics 0,10 à 0,66 Gio. Témoins : `01e1f945` **17,9 s**, `7344d24f` **21,0 s**, `696a9d7c`
**20,5 s** — le gain du chantier est intact après le merge (cible du plan : < 100 s).

### Étape B — (à remplir par l'agent : tableau par NOM d'étape, corrélation véhicules, 49 lignes)

### Étape C — (attendu : 13/13 identiques sans `-update`)

### Fusions D10 — (attendu : 13/13 identiques sans `-update` sur les TSV de B, après chaque fusion)

## §6 Journal
- 2026-09-05 14 h 40 — plan écrit après inventaire mesuré (§1) ; relecture `plan-review` lancée
  en parallèle de l'étape A (geste déjà rodé le 03/09).
- 2026-09-05 — **ÉTAPE A exécutée (A.1 à A.4), rien de commité** (mandat du pilote : l'arbre est
  résolu, l'index prêt). Cible `7fb4b60a1`, 324 fichiers, 5 conflits.

  **Les 5 conflits et leur résolution** (règle : la SÉMANTIQUE amont est préservée intégralement,
  dans NOTRE architecture) :

  | conflit | ce que l'amont voulait | résolution |
  |---|---|---|
  | `analysis/replay/build.go` | ajouter 5 champs à `Options` (`AbilityImpulses`/`Stats`, `AbilityCharges`/`Stats`, `Translocations`) et 3 balayages à `BuildFromFilm` (l.308/457/473) | Le bloc en conflit EST celui que le lot 1 a sorti de `build.go`. Vérifié sur pièces (diff bloc-à-bloc base `ceabaad67` contre amont) : l'amont n'y touche QUE ces deux choses. Bloc supprimé chez nous ; les 5 champs REJOUÉS dans `options.go` au même rang (commentaires amont inclus), les 3 balayages dans `build_from_film.go`. Les hunks amont de `BuildFromPositions` (calques `translocations`, `abilityImpulses`, `abilityCharges` + leurs couvertures gatées par `Scanned`) sont hors du bloc déplacé et ont fusionné seuls |
  | `filmdec/ability_rank.go` | extraire `abilityScanSetup` + `resolveAbilityScan(dir)` + `walkAbilityEmissionsWith(s, …)` pour que `ScanFilmEquipmentChanges` résolve le film UNE fois pour ses DEUX passes (strict, puis récupération gatée) ; scinder `walkComponentsAt` hors de `walkRecordComponents` | L'intention amont — résoudre une fois, partager — EST celle de notre `FilmContext`. La structure amont est reprise TELLE QUELLE, son champ `dir string` devenant `fc *FilmContext` et `slots map[uint32]bool` devenant `SlotBand` ; `resolveAbilityScan(fc)` lit `fc.ChunkNumbers/BipedSlots/I0Layout/bipedArchetype` (messages d'erreur du lot 1 conservés) ; `ReadFilmChunk(s.dir, c)` → `s.fc.ChunkAt(c)`. `walkComponentsAt` pris tel quel. La résolution unique amont est donc conservée ET élargie : le contexte la partage aussi avec les balayages hors de ce setup |
  | `filmdec/equipment_changes.go` | remplacer l'accumulation en ligne par DEUX PASSES (strict + récupération) et une fusion pure `assembleEquipmentChanges` | Corps amont pris INTÉGRALEMENT ; seule la tête change : `ScanEquipmentChanges(fc, bornAt)` fait `resolveAbilityScan(fc)`, et `ScanFilmEquipmentChanges(dir, …)` redevient l'enveloppe D2 (`LoadDir` + `NewFilmContext`). L'import `sort` amont fusionné avec notre import `filmsource` |
  | `.ai/thought_log.md` | deux blocs insérés aux mêmes ancres | UNION : nos entrées en tête, puis celles de l'amont — aucune perdue des deux côtés ; ligne de séparation rétablie |
  | `.ai/V7.5/REGISTRE_REPORTS.md` | une ligne de report neuve (`TestBancCliffhanger` rouge préexistant du corpus `gamefiles`) | UNION du tableau : notre ligne `hits.go` puis la sienne |

  **TROIS BALAYAGES MIGRÉS (D2)**, avec leur enveloppe conservée et son unique raison d'être :

  | ancien (production) | nouveau | enveloppe `(dir)` gardée pour |
  |---|---|---|
  | `ScanFilmTranslocatorTeleports(dir, entry)` — `build.go:308` | `ScanTranslocatorTeleports(film, entry)` — `transloc_events.go`, patron de `ScanZoomEvents` (`FilmChunkNumbers`/`FilmChunkAt`) | `replay/golden_inputs_test.go`, `filmdec/transloc_exemption_film_test.go`, `filmdec/transloc_positions_film_test.go` |
  | `ScanFilmAbilityImpulses(dir)` — `build.go:457` | `ScanAbilityImpulses(fc)` — `ability_impulses.go` | `replay/golden_inputs_test.go` |
  | `ScanFilmAbilityCharges(dir)` — `build.go:473` | `ScanAbilityCharges(fc)` — `ability_charges.go` | `replay/golden_inputs_test.go` |

  Le quatrième point d'entrée du même setup, `scanEquipmentRecovery`, ne lit plus le disque non
  plus : sa borne aux chunks des fenêtres est conservée (elle borne le TRAVAIL, marche bit à bit,
  pas seulement l'E/S) mais elle relit `s.fc.ChunkAt(c)`, en mémoire.

  **GARDES TOUCHÉES** :
  - `archlint/no_film_reread_test.go` (règle 3) — `enveloppesInterditesEnProduction` **40 → 43** :
    les trois enveloppes ci-dessus, avec la justification datée et la vérification d'homonymie
    exigée par le fichier (aucune méthode ne porte ces noms). L'allowlist
    `appelsDEnveloppeAutorises` (2 entrées `hits.go`, amont du 03/09) est **inchangée** et reste
    vivante — le test la vérifie dans les deux sens.
  - `replay/observe.go` — `BuildFromFilmSteps` **31 → 34** (`translocations` en tête,
    `abilityImpulses` et `abilityCharges` après `grappleReads`). Pas d'étape `.stats` pour les deux
    dernières : leurs témoins `Absent`/`Scanned` voyagent dans `Options` et sont couverts par le
    digest `artifact`, comme `pads`/`carrierMarks`/`zoneReads`/`bombReads`.
  - `filmdec/equipment_state.go` — commentaire de `EquipmentArchetypeDir` : « la seule des 40
    enveloppes » → « 40 au lot 6, 43 depuis le 2026-09-05 » (doc tenue à jour dans le commit qui
    la périme).
  - **AUCUNE allowlist élargie.** `archlint/filmdec_package_vars_test.go` : le ratchet reste à
    **116** — MESURÉ, l'amont n'ajoute AUCUNE variable de paquet à `filmdec` (le test passe sans
    Logf « a baissé », donc le compte est exactement 116). `no_recomputed_film_context_test.go`,
    `filmsource_leaf_test.go`, `no_rewritten_slot_band_test.go`, `no_unbounded_film_loop_test.go` :
    verts sans modification.

  **CINQ RUPTURES DE COMPILATION LAISSÉES PAR L'AUTO-MERGE**, toutes dans des tests NEUFS de
  l'amont appelant des symboles supprimés au lot 1 — intention amont rejouée, jamais contournée
  (même classe qu'`inflateChunk` et `worldObjectSlotBand` le 2026-09-03) :
  - `filmdec/i48_manques_research_test.go` : `bipedSlotBand(dir, …)` → `bipedSlotBandDir`,
    `bipedArchetype(dir)` → `bipedArchetypeDir`, `slots map[uint32]bool` → `SlotBand`
    (`len()` → `.Count()`, `s.slots[slot]` → `s.slots.Has(slot)`) ;
  - `filmdec/r8_i54_research_test.go` et `filmdec/r12_socle_research_test.go` : mêmes deux shims,
    même bascule de type (`r11_journal_research_test.go` en dépendait et se répare avec eux) ;
  - `filmdec/r9_i22_signal_research_test.go` : `worldObjectSlotBand(dir, n, ti)` →
    `worldObjectSlotBandDir`, `EquipmentArchetype(dir)` → `EquipmentArchetypeDir` ;
  - `filmdec/r9_ti37_identite_research_test.go` : `worldObjectSlotBand(dir, …)` → `…Dir` ;
  - `replay/ability_charges_film_test.go` et `replay/ability_impulses_film_test.go` : ils
    rejouaient la chaîne de production depuis un RÉPERTOIRE. Un shim `buildFromFilmDir` a été
    ajouté à `replay/film_shims_test.go` (fichier `_test.go` : le compilateur interdit qu'un
    chemin de production l'emprunte) ; il ÉCHOUE sur un répertoire illisible plutôt que de
    construire sur un film nil, parce que ces instruments comparent à des relevés Theater.
    Mesure inchangée dans les sept cas.

  **`replay.SchemaVersion` : aucune collision** — base 37, notre branche 37 (le chantier ne bumpe
  jamais, c'est un refacto), amont **38**. Le merge prend 38.

  **RESTE DOUTEUX / À SURVEILLER** : (a) la comparaison positionnelle du harnais rend son message
  « première étape en écart » inexploitable dès qu'une étape s'insère AILLEURS QU'À LA FIN — ce
  n'est pas un bug (le fichier de digests est une séquence figée) mais l'étape B, qui ajoutera
  `vehicles`, devra refaire le diff par NOM ; (b) l'étape `translocations` est observée AVANT
  `positions` parce que le balayage arme l'exemption du filtre — si un lot futur déplaçait ce
  balayage, l'ordre de `BuildFromFilmSteps` devrait suivre.
- 2026-09-05 15 h 39 — **A.5 : commit `eb80a4f0a`** après vérifications sur pièces du pilote.
- 2026-09-05 15 h 45 — Relecture `plan-review` fondue (BL-1 → D11, BL-2 → D3, BL-3 → D8, IMP-1/2/3
  → B.2/B.5/D4, IMP-4 → D12, IMP-5 → D8bis (+ gel utilisateur), IMP-6 → D.1/D10, IMP-7 → E.1,
  MIN-1..10 → D5/D6/D9/§1/D8ter/C.4/C.2bis/C.3/§7/F.4). D10 ajouté : B, C, D en PARALLÈLE dans trois
  worktrees dérivés de `eb80a4f0a` (créés, jonctions posées, `npm ci` lancé). Agents lancés :
  B (Opus), C (Opus), D (Sonnet).

## §7 Contrat d'exécution (rappel opposable)
Statuts : `[x]` fait · `[~]` couvert ailleurs (avec la référence) · `[!]` non traité (avec la
justification écrite). AUCUNE case vide à la clôture d'une étape. Ordre strict : l'étape N+1 ne
commence pas avant le gate de N (B, C, D sont parallèles entre elles par D10, mais chacune est
strictement ordonnée en interne, et E ne commence qu'après les trois fusions). Zéro fix hors
périmètre : la découverte va au §4, pas dans le diff. Jamais d'allowlist élargie sans
justification datée, jamais de plafond de ratchet relevé. REPRISE DE SESSION : lire §5 (dernier
diff de comptes consigné) puis §6 (journal) ; la première case non cochée de la première étape non
close est le point de reprise ; les worktrees D10 se listent par `git worktree list | grep integ`.
