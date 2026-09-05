# PLAN — Intégration des branches actives dans l'architecture cuisson-perf, puis merge et push de feat/v75

> Date : 2026-09-05. Branche `wt/cuisson-perf` (worktree `LevelUp-wt-cuisson-perf`), HEAD de départ
> `31b754363`. Exécution sous le contrat du skill `plan-execution` (ordre strict, aucun report
> d'un item exécutable, statut par item, vérification sur pièces). Pilote : cette session ;
> exécution et revue par agents Opus ; le pilote commite (jamais de stash, jamais de hook
> contourné).

## §0 Mandat (utilisateur, 2026-09-05)

« Le sujet optimisation doit reprendre le travail des véhicules et du translocateur/répulseur et
des nouvelles stats pour l'intégrer proprement à son refacto (branches actives non mergées avec
feat/v75). Quand ce sera fait tu mergeras dans feat/v75 [...] et quand tu auras fini, tu push et
vérifieras que tout est vert, même si ce n'est pas de ta faute. » Contrainte ajoutée : « propre,
solide et pérenne, si jamais tu dois faire des ajustements ».

## §1 État des lieux (mesuré le 2026-09-05, 14 h 30)

| Branche | Avance / retard sur feat/v75 | Base | Schéma | Périmètre | Décision |
|---|---|---|---|---|---|
| `feat/v75` (origin `7fb4b60a1`) | — (65 commits depuis notre dernière réconciliation `ceabaad67`) | — | **38** | translocateur, propulseur, charges d'équipement (3 balayages ANCIENNE forme appelés dans `build.go:308/457/473`), vue match, himodule mmap, CI | **Étape A** — réconcilier d'abord |
| `origin/feat/v75-vehicules-sons` (28) ∪ `wt/vehicule-deadstate` (+1 : `0b5141b8a`, tests + notes) | 28 / **339** | `14a115bb1` | **31** (bump propre à la branche) | 65 fichiers filmdec + 15 replay non-test, **8 balayages ancienne forme** (5 appelés en production via `build_vehicles.go`), `build.go` (+Vehicles, attachVehicles, attachVehicleShots, buildShots à 3 retours), `document.go` +97, web match-replay ×50, sons ×66, assets ×20, 3 outils cmd | **Étape B** — le gros morceau |
| `wt/session-usage` (10) | 10 / 22 | `da616828f` | 38 (inchangé) | paquet `sessionusage`, `replay/usage_summary*.go` (projection du DOCUMENT, aucun balayage), persister INSERT-only + 2 tables append-only neuves + vues `_latest`, migration, capability `film.usage_summary`, crochet dans `replayartifacts/{cuisson,artifacts,journal}.go` + `usage.go`, CLI `backfill-usage-summary`, killcollector (classifier, postsync), web session-detail, openapi +307 | **Étape C** |
| `wt/game-changers` (4) | 4 / 15 | `ca55f0ed7` | — | web uniquement (match-replay ×12, ui ×3) | **Étape D** |
| `wt/assaut-stats` (3) | 3 / 34 | `146f1d92e` | 38 | `replay/bomb_stats.go`, `bomb_arms.go` + tests de recherche ; **aucun appelant dans tout go-api** ; worktree avec 6 fichiers NON commités (recherche du 04/09) | **[!] non intégrée** (D5) |
| `wt/r9-repulseur`, `wt/r12-repulseur`, `wt/son-propulseur`, `wt/lint-filmdec`, `wt/himap-mmap`, `wt/trame-film` | 0 | — | — | déjà dans feat/v75 | arrivent par l'étape A |
| `main` | 9 commits hors feat/v75 (hotfix 7.3.1 classement mondial) | — | — | — | hors mandat — signalé à l'utilisateur |

Corpus du harnais : 13 films (`internal/analysis/replay/testdata/equivalence/CORPUS.txt`), 45
étapes observées, marqueur `# digest-grammar: 2`.

## §2 Décisions

- **D1 — Ordre** : A (amont) → B (véhicules) → C (usage de session) → D (game-changers) → E
  (gates + revue de branche) → F (merge, push, CI). Chaque étape se ferme par un commit du pilote ;
  aucune étape ne commence tant que la précédente n'a pas son gate.
- **D2 — Rejouer, jamais réintroduire** : tout balayage en ancienne forme (`ScanFilm*(dir)`)
  arrivant d'une branche se réécrit en `Scan*(film, ...)` lisant `film.Packets(i)` et le
  `FilmContext` (bande de slots, layout catalogue, registre) ; une enveloppe `(dir)` n'existe que
  pour les tests (liste fermée d'archlint). Chaque nouveau balayage de production reçoit son étape
  observée (`BuildFromFilmSteps` + `opt.observe`). Les garde-rails (observe_test,
  no_film_reread, filmsource_leaf, no_recomputed_film_context, ratchet des vars, liste des
  enveloppes) sont la checklist : un rouge = un oubli, on corrige dans le sens de l'intention de la
  branche, jamais en élargissant une allowlist sans justification datée.
- **D3 — Un seul bump de schéma** : l'étape A absorbe le 38 amont tel quel ; l'étape B porte le
  **39** (champs véhicules posés SUR le 38, le 31 de la branche est abandonné) ; C et D ne
  bumpent pas (le résumé d'usage est une projection hors artefact ; game-changers est web).
- **D4 — Preuve par le harnais à chaque étape** : passe complète SANS `-update` d'abord ; chaque
  écart doit être **corrélé** à l'intention de la branche (quels films, quelle étape, pourquoi ces
  films-là et pas les autres) — un écart inexpliqué = arrêt et rapport ; puis `-update`, puis
  passe de vérification 13/13 ; diff des comptes au §5. Une étape sans changement attendu (C, D)
  doit rendre 13/13 identiques SANS `-update`.
- **D5 — `wt/assaut-stats` n'est pas intégrée** : `BuildBombStats`/`BuildBombArms` n'ont aucun
  appelant hors de leurs tests (bibliothèque non câblée : table `match_bomb_stats`, capability
  `film.bomb_stats` et écriture au sync restent à faire), elles ne dépendent que de
  `objectiveevents.IdentifiedEvent` (post-décodage : **aucune dépendance au refacto**, rien à
  rejouer), et le worktree porte de la recherche non commitée. L'intégrer aujourd'hui poserait
  du code mort dans feat/v75 (anti-pattern n°1) sans rien lui faire gagner ; la branche mergera
  sans friction à la clôture de son chantier. Ajustement signalé à l'utilisateur.
- **D6 — Champ `structure` de l'artefact : hors périmètre** — contrairement au rapport de
  l'agent amont, deux lecteurs web subsistent (`features/match-replay/replayLogic.ts:274` —
  repli des bornes, `replayNormalize.ts:407`). Son retrait est une décision produit (bornes sans
  structure) + web + schéma : au registre des reports, condition de reprise = décision produit.
- **D7 — mmap** : le passage à la projection mémoire est dans `internal/himodule/projection_*.go`
  (lecteur des modules de cartes), pas dans la cuisson des films. `filmsource` lit des chunks zlib
  de quelques dizaines de Ko puis les décompresse : aucun gain à attendre d'une projection. Pas
  d'action ; consigné.
- **D8 — Merge final et CI** : fetch frais ; si `feat/v75` a bougé depuis A, mini-réconciliation
  (même protocole) AVANT le merge ; merge dans le worktree partagé uniquement s'il est propre
  (sinon attendre / signaler) ; `git push origin feat/v75` (jamais `main`) ; CI verte **au niveau
  job** (`Frontend`, `Go Build + Test` matrice, `Go Lint`, `Lease Enforcement`, `Coverage +
  Baseline`, `OpenAPI Lint`, `Contract Test`) — un rouge préexistant se corrige aussi (mandat).
  **Aucune re-cuisson de masse sans accord explicite** (le 39 rend tous les artefacts à recuire :
  proposition faite à la clôture, jamais lancée d'office).
- **D9 — Union des deux branches véhicules** : `origin/feat/v75-vehicules-sons` (4 commits de CI
  propres à la branche) et `wt/vehicule-deadstate` (`0b5141b8a`, dead-state v13 : tests + notes)
  ont divergé ; on intègre l'union (merge de l'une puis de l'autre), rien n'est perdu.

## §3 Étapes

### Étape A — Réconciliation amont (`feat/v75` `7fb4b60a1` → `wt/cuisson-perf`)
- [x] A.1 `git fetch origin` + `git merge --no-ff --no-commit origin/feat/v75` — cible **`7fb4b60a1`**
  (celle du §1, inchangée ; 65 commits amont, 15 à nous, base de merge `ceabaad67`), **324 fichiers**,
  **5 conflits** résolus « sémantique amont dans notre architecture », aucun commit.
  PREUVE : `git diff --name-only --diff-filter=U` rend VIDE, `git grep '^<<<<<<< HEAD$'` rend VIDE.
  Détail fichier par fichier au §6. Les fichiers croisés annoncés `offline_biped.go`,
  `golden_inputs_test.go`, `no_rewritten_slot_band_test.go` et les deux `COMMANDS.md` ont fusionné
  SEULS (vérifiés : +31/+33 lignes amont, notre section « outillage de construction » intacte).
- [x] A.2 Les 3 balayages amont passent en nouvelle forme : `ScanTranslocatorTeleports(film, entry)`,
  `ScanAbilityImpulses(fc)`, `ScanAbilityCharges(fc)` ; les 3 enveloppes `(dir)` survivent (des tests
  les appellent) et entrent dans la liste fermée d'archlint (**40 → 43**) ; 3 étapes observées
  (`BuildFromFilmSteps` **31 → 34**). PREUVE : `go test ./internal/analysis/replay/ -run
  TestObserveEtapesBuildFromFilm` et `./internal/archlint/` verts ; le TSV d'un film porte 48 étapes
  au lieu de 45.
- [x] A.3 Gates — `gofmt -l .` VIDE · `go build ./...` code 0 · `go vet ./...` code 0 ·
  suites `./internal/analysis/... ./internal/replaybuild/ ./internal/games/halo_infinite/film/...
  ./internal/sync/... ./internal/archlint/ ./cmd/...` **EXIT_TESTS=0** (66 paquets `ok`) ·
  `go test -tags=integration -p 1 ./internal/sync/...` **EXIT_INTEG=0** · `golangci-lint run` :
  **281 issues** au total, **0 sur un fichier touché par la résolution** (croisement mesuré :
  seuls `filmdec/components_biped_ability.go` et `filmdec/traverse.go` en portent dans les deux
  paquets concernés, aucun des deux n'a été modifié).
- [x] A.4 Harnais — passe SANS `-update` : 0 identique / 13 différents ; écarts CORRÉLÉS et mesurés
  (§5) ; `-update` (13/13 écrits) ; passe de vérification **13 identiques, 0 différent, 0 écarté,
  0 échec, 0 illisible**. Témoins : `01e1f945` **17,9 s**, `7344d24f` **21,0 s**, `696a9d7c`
  **20,5 s** (cible < 100 s). Aucun écart inexpliqué.
- [ ] A.5 Commit du pilote.

### Étape B — Véhicules (union D9) — schéma 39
- [ ] B.1 Merge `origin/feat/v75-vehicules-sons` puis `wt/vehicule-deadstate` (`--no-commit`) ;
  conflits (base à 339 commits : `build.go`, `document.go`, `registry.go`, `traverse.go`,
  `offline_biped.go`, `event_list.go`, `frame_records.go`, `keyframe_*`, `equipment_creation.go`,
  `unit_weaponstate.go`, tests, web match-replay, `.ai`) résolus « sémantique véhicules dans notre
  architecture ET sur le schéma 38 amont ».
- [ ] B.2 Les 8 balayages en nouvelle forme : `ScanVehicleEvents(film)`,
  `ScanKeyframeRecordSpans(film)`, `ScanBipedAimOnly(film)`, `ScanBipedPositionsForBand(film, band, opt)`,
  `ScanVehicleCreations(ForBand)(film, ...)`, `ScanVehicleOccupancy(film)` ;
  `decodeFilmVehicleScan(filmDir, ...)` devient une consommation du film chargé et du
  `FilmContext` ; enveloppes `(dir)` hors production ; étapes observées (`vehicles`,
  `vehicleShots`, …) ; gardes.
- [ ] B.3 `SchemaVersion = 39` (une fois, commentée : véhicules sur le 38) ; document : champs
  véhicules ; web : types régénérés (`make generate-types`) si l'OpenAPI bouge.
- [ ] B.4 Gates (comme A.3) + web : typecheck, lint, vitest sur match-replay.
- [ ] B.5 Harnais : écarts corrélés (films AVEC véhicules bougent sur `vehicles`/`shots`/`coverage`/
  `artifact` ; films sans véhicule : `artifact` seul par le numéro de schéma) ; `-update` ;
  13/13 ; diff des comptes au §5. Témoin obligatoire : un BTB du corpus.
- [ ] B.6 Commit du pilote.

### Étape C — Usage de session (`wt/session-usage`)
- [ ] C.1 Merge `--no-commit` ; conflits `replayartifacts/{cuisson,artifacts,journal}.go` : le
  crochet de projection se rejoue dans `cuireUnMatch`/le bilan du lot 5 (projection APRÈS
  rangement, lecture du fichier rangé, writer court APRÈS toute cuisson — doctrine de `usage.go`) ;
  `killcollector`, `no_art_patterns_test.go` (entrées datées existantes), migration, openapi.
- [ ] C.2 Vérifier sur pièces : persister INSERT-only, tables append-only avec vues `_latest`,
  capability et non `slug ==`, aucune lecture brute des tables append-only.
- [ ] C.3 Gates + intégration `-p 1` (dont `usage_integration_test.go`) + web session-detail.
- [ ] C.4 Harnais : **13/13 identiques attendus sans `-update`** (projection hors artefact) ; si
  `document_pickups.go`/`pad_pickup_dating.go` font bouger un compte : écart nommé, corrélé,
  re-figé.
- [ ] C.5 Commit du pilote.

### Étape D — Game-changers (`wt/game-changers`, web)
- [x] D.0 `git fetch origin` — **0 commit** entre `eb80a4f0a` et `origin/feat/v75` (amont
  toujours gelé, mesuré). `git merge --no-ff --no-commit wt/game-changers` — base de merge
  **`ca55f0ed7`** (celle du §1), **4 commits**, **16 fichiers** (`.ai/thought_log.md`,
  `.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md` neuf, 3 `components/ui/collapsed-items-toggle*`
  neufs, 12 `features/match-replay`). Détail au §6.
- [x] D.1 Merge automatique SANS AUCUN CONFLIT (`git status` : « All conflicts fixed » —
  aucun marqueur `<<<<<<<`, `git grep` VIDE) : les croisements annoncés avec B/C
  (`i18n.ts`, `i18nContract.ts`, `MatchEquipmentUsageSection.tsx`) ne se sont pas manifestés
  ici, B et C n'étant pas encore mergées dans cette branche — laissés au pilote à la fusion
  finale, comme prescrit. `.ai/thought_log.md` fusionné seul par git en UNION (nos entrées en
  tête à la ligne 1, les leurs à leur ancre d'origine plus bas, aucune perdue des deux côtés).
- [x] D.2 Gates web D12 complets, tous verts : `npm run typecheck` **exit 0** ·
  `npm run lint` **exit 0** (23 warnings préexistants, 0 erreur, aucun sur un fichier touché) ·
  `npm run lint:fields` **exit 0** (220 labels FR+EN, 1643 fichiers scannés, 0 violation) ·
  `npm run test` **exit 0** (577 fichiers / 6008 tests passés, 14 skipped, 0 échec) ·
  `npm run build` **exit 0** (avertissements de taille de chunk préexistants) ·
  `node tools/knip-ratchet.mjs` **exit 0** (files 0/0, exports 0/0, types 0/0 — le nouveau
  `collapsed-items-toggle.tsx` est consommé par `equipmentUsageColumns.ts` /
  `MatchEquipmentUsageSection.tsx` / `MatchPadControlSection.tsx`, aucun export mort neuf).
  PREUVE harnais [~] : `git diff --stat eb80a4f0a -- apps/go-api` **VIDE** — rien côté Go,
  harnais non rejoué (étape déclarée sans changement attendu, prouvé sur pièces).
- [x] D.3 Commit du pilote : merge commit sur `wt/integ-gamechangers`, SHA au §6.

### Étape E — Filet complet et revue de branche
- [ ] E.1 `make gate-push` vert (lint Go de branche, baseline de tests, web) ; intégration `-p 1`
  complète sur sync.
- [ ] E.2 Revue adversariale du diff d'intégration (`31b754363..HEAD`, contexte frais) ;
  corrections ; seconde lecture.
- [ ] E.3 Journal, registre des reports (D5, D6, D7), thought_log ; commit.

### Étape F — Merge, push, CI
- [ ] F.1 `git fetch` ; si `origin/feat/v75` ≠ base de A : mini-réconciliation (protocole A) +
  harnais + gates ; commit.
- [ ] F.2 Worktree partagé propre → `git merge --no-ff wt/cuisson-perf` dans `feat/v75` ; sinon
  attendre/signaler.
- [ ] F.3 `git push origin feat/v75` ; suivi `gh run watch` ; tout job rouge (même préexistant)
  se diagnostique et se corrige ; re-push jusqu'au vert au niveau job.
- [ ] F.4 Rapport à l'utilisateur : ce qui est intégré, les écarts nommés, D5/D6/D7, la
  proposition de re-cuisson de masse (jamais lancée d'office), la note sur `main`.

## §4 Hors périmètre / découvertes
- N-1 `wt/assaut-stats` : D5.
- N-2 champ `structure` : D6 (lecteurs web `replayLogic.ts:274`, `replayNormalize.ts:407`).
- N-3 mmap himodule : D7.
- N-4 `main` a 9 commits hors `feat/v75` (hotfix 7.3.1) — à ramener dans feat/v75 un jour ; hors
  mandat.
- N-5 les trois outils `cmd/{weapon-sounds,vs-measure,vehicle-sprite}` de la branche véhicules
  arrivent tels quels ; à statuer par leur auteur (outillage de recherche vs production).

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
  `vehicles`/`vehicleShots`, devra refaire le diff par NOM ; (b) l'étape `translocations` est
  observée AVANT `positions` parce que le balayage arme l'exemption du filtre — si un lot futur
  déplaçait ce balayage, l'ordre de `BuildFromFilmSteps` devrait suivre.

- 2026-09-05 — **ÉTAPE D exécutée et commitée** (worktree dédié `LevelUp-wt-integ-gamechangers`,
  branche `wt/integ-gamechangers`, HEAD de départ `eb80a4f0a`). D.0 : `git fetch origin` — 0 commit
  d'écart `eb80a4f0a..origin/feat/v75` (amont toujours gelé). `git merge --no-ff --no-commit
  wt/game-changers` (base `ca55f0ed7`, 4 commits, 16 fichiers) : **merge automatique SANS AUCUN
  CONFLIT** — `git status` rend « All conflicts fixed », `git grep '^<<<<<<< HEAD$'` VIDE. Les
  croisements redoutés avec les étapes B/C (`i18n.ts`, `i18nContract.ts`,
  `MatchEquipmentUsageSection.tsx`) ne se sont pas matérialisés : cette branche ne porte encore
  aucune des deux, rien à arbitrer ici — reste au pilote à la fusion finale. `.ai/thought_log.md`
  a fusionné seul en union (nos entrées en tête, celles de `wt/game-changers` à leur ancre
  d'origine, aucune perdue).

  **Fichiers apportés** : `.ai/PLAN_REPLI_GAME_CHANGERS_2026-09-05.md` (neuf, plan de repli
  « game changers »), 3 `apps/web/src/components/ui/collapsed-items-toggle.{tsx,test.tsx,
  guard.test.ts}` (extraction du contrôle « Voir plus (N) », 3e usage → règle n°6), 12 fichiers
  `apps/web/src/features/match-replay/` dont `gameChangers.ts`/`.test.ts` (nouveaux — la liste
  tranchée par vote utilisateur des familles d'équipement/armes « game changer ») et les
  adaptations de `equipmentUsageColumns.ts`, `MatchEquipmentUsageSection.tsx`,
  `MatchPadControlSection.tsx`, `padControlLogic.ts`, `i18n.ts`, `i18nContract.ts`.

  **Gates D.2 — tous verts, un code de sortie par ligne** : `npm run typecheck` exit 0 ·
  `npm run lint` exit 0 (23 warnings préexistants, 0 erreur, aucun sur un fichier touché par ce
  merge) · `npm run lint:fields` exit 0 (220 labels FR+EN, 1643 fichiers scannés, 0 violation) ·
  `npm run test` exit 0 (**577 fichiers, 6008 tests passés, 14 skipped, 0 échec**) ·
  `npm run build` exit 0 (avertissements de taille de chunk préexistants, non nouveaux) ·
  `node tools/knip-ratchet.mjs` (depuis la racine) exit 0 — **files 0/0, exports 0/0, types
  0/0** : le nouveau `collapsed-items-toggle.tsx` est bien câblé (consommé par
  `equipmentUsageColumns.ts`, `MatchEquipmentUsageSection.tsx`, `MatchPadControlSection.tsx`),
  aucun export mort neuf, plafond jamais relevé.

  **Preuve harnais [~]** : `git diff --stat eb80a4f0a -- apps/go-api` rend VIDE — zéro octet
  touché côté Go par ce merge (branche 100% web) ; le harnais d'équivalence (13 films) reste
  valide sans rejeu, comme prévu pour une étape sans changement attendu côté artefact.

  **Douteux / hors périmètre signalé, non traité ici (règle n°5 zéro fix opportuniste)** : le
  journal `wt/game-changers` (commit `c972b8be0`) documente un lot I abandonné sur clarification
  utilisateur (exposition `weapon_key` pour répliquer le repli aux graphes de performance) — décision
  déjà actée par cette branche avant le merge, rien à statuer côté intégration.

  Commit du pilote : merge de fusion sur `wt/integ-gamechangers`, message français, voir
  `git log --oneline -1` pour le SHA (rapporté à l'utilisateur en clôture de tâche).
