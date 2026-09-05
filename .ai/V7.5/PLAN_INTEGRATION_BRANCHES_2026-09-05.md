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
- [x] B.0 D8bis — `git fetch origin` : **`origin/feat/v75` inchangé à `7fb4b60a1`**, ZÉRO commit
  amont depuis l'étape A (`git log --oneline 7fb4b60a1..origin/feat/v75` rend vide). Aucune
  mini-réconciliation. Merge lancé sur cette base.
- [x] B.1 Merge `origin/feat/v75-vehicules-sons` (`1e3d459d1`, 28 commits, base `14a115bb1`) puis
  ajout du second parent `wt/vehicule-deadstate` (`0b5141b8a`) — **387 fichiers, 26 conflits**,
  résolus « sémantique véhicules dans NOTRE architecture ET sur le schéma 38 amont ». Détail
  fichier par fichier au §6. PREUVE : `git diff --name-only --diff-filter=U` VIDE,
  `git grep '^<<<<<<< '` VIDE.
- [x] B.2 Les balayages en nouvelle forme, les CINQ sites de production de `build_vehicles.go`
  compris ; trois enveloppes SANS APPELANT supprimées (N-6) ; quatre enveloppes ajoutées à la
  liste fermée d'archlint (**43 → 47**) ; ratchet des vars de `filmdec` **116 → 118** (deux vars
  de la branche, chacune justifiée) ; UNE étape observée `vehicles` (`BuildFromFilmSteps`
  **34 → 35**). Tableau des migrations au §6. Gardes vertes : `./internal/archlint/` **ok**.
- [x] B.3 `SchemaVersion = 39` (une fois, `document.go`, commentaire : les trois montées 29/30/31
  du chantier fondues en une, posées sur le 38 — décision D3) ; `make openapi-gen` +
  `make generate-types` + `make openapi-check` **exit 0** ; `contracttest` **ok**
  (`wantReplayDocumentFields` 52 → 54).
- [x] B.4 Gates — `gofmt -l .` VIDE · `go build ./...` **EXIT_BUILD=0** · `go vet ./...`
  **EXIT_VET=0** · suites `./internal/analysis/... ./internal/replaybuild/
  ./internal/games/halo_infinite/film/... ./internal/sync/... ./internal/archlint/
  ./internal/api/... ./cmd/...` **EXIT_TESTS=0** (71 paquets `ok`). Web : détail au §6.
- [x] B.5 Harnais — passe 1 SANS `-update` : 0 identique / 13 différents ; **diff PAR NOM** (la
  comparaison est POSITIONNELLE) : `vehicles` NEUVE 13/13, `artifact` bouge 13/13, `killsource`
  bouge **4/13** — les 4 films dont la bande `ti=40` est peuplée, et eux seuls : c'est la
  correction déclarée du lot V9 de la branche (N-8), pas un effet de l'intégration ; AUCUNE autre
  étape ne bouge, aucune ne disparaît. Corrélation MESURÉE : `artifact` dépasse son plancher de
  +450 octets sur EXACTEMENT les 5 films à `vehicles` non vide. Puis `-update` (13 écrits) et
  passe de vérification **13 identiques, 0 différent, 0 écarté, 0 échec, 0 illisible**. Gate de
  lignes : 49 étapes sur les 13 TSV ; `minifilm.tsv` INTACT. Témoin BTB obligatoire : `084a804d`
  (180 vies, 109 478 échantillons). Tableau complet au §5.
- [x] B.6 Commit du pilote — un seul commit de merge à DEUX parents.

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
- [ ] D.1 Merge ; conflits web match-replay (avec B) résolus dans le sens des deux intentions.
- [ ] D.2 typecheck, lint, vitest ; harnais 13/13 sans `-update` (rien côté Go).
- [ ] D.3 Commit du pilote.

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
- N-6 **Trois entrées de balayage du chantier véhicules arrivaient SANS AUCUN APPELANT** (tests
  compris) : `ScanFilmVehicleCreationsForBand`, `ScanFilmKeyframeRecordSpans` et
  `ScanFilmVehicleOccupancy` — cette dernière avec sa forme film, dont elle était l'unique
  consommateur. Elles ont été **supprimées** à l'étape B (règle D2 « une enveloppe sans appelant
  est supprimée » + règle 0 du dépôt), et NON inscrites à la liste fermée d'archlint. Rien de la
  MESURE n'est perdu : les moitiés pures et testées restent (`KeyframeRecordSpans`,
  `VehicleKeyframeStates`, `FindKeyframeBlockInsertion`), et l'en-tête de chaque fichier dit ce
  qui a été retiré et pourquoi. À signaler à l'auteur du chantier.
- N-7 `apps/web/.../ReplayCanvas.tsx` est **exactement à son plafond** (664 lignes pour 665) après
  le câblage véhicules. La prochaine addition devra être payée par une extraction, comme les dix-sept
  précédentes.
- N-8 **L'ÉCART `killsource` de l'étape B est une CORRECTION DÉCLARÉE DE LA BRANCHE, pas une
  régression de l'intégration** : `filmdec/traverse.go` routait les deux composants
  `-dynamic-precision-` d'orientation de `ti=38/39/40/43` vers les désérialiseurs du BIPÈDE ; le
  lot V9 du chantier (Ghidra, validé 6/6 contre la table ECS live) les remet sur les leurs. Le
  digest `killsource` bouge donc sur les 4 films du corpus dont la bande `ti=40` est peuplée, et
  sur eux seuls. Chiffres et corrélation au §5. **À signaler à l'utilisateur** : le kill-feed des
  matchs à véhicules change avec cette livraison, dans le sens de la correction.

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


### Étape B — véhicules (union D9), schéma 39 (2026-09-05)

**LE DIFF EXPLOITABLE EST CELUI PAR NOM D'ÉTAPE** (la comparaison du harnais est POSITIONNELLE, et
`vehicles` s'insère APRÈS `pads`, donc au milieu du fichier) : copie des 13 TSV prise avant
refigeage, comparée nom par nom aux TSV refigés. **AUCUNE étape ne disparaît**, et seuls **TROIS
noms** bougent :

| étape | films touchés | nature | cause |
|---|---|---|---|
| `vehicles` | **13/13** | NOUVELLE | le calque `ti=40` du chantier. Contenu VIDE (sha `f73d0cef…`, `VehicleScan{}`) sur 8 films sans aucun slot `ti=40` aux images-clés ; contenu propre sur 5 : `084a804d` (180 vies, 109 478 échantillons, 78 événements, 64 297 visées), `a349fea8`, `1c4c63c2` (67 vies), `53ce4390` (26 vies) et `60ae07c4` (4 slots recensés, **0 échantillon** — bande présente, entités muettes) |
| `artifact` | **13/13** | CHANGE (octets) | schéma **38 → 39** + le calque véhicules. **Plancher mesuré : +450 octets**, et il tombe sur les 8 films à `vehicles` VIDE (`000d5950`, `01e1f945`, `51101d1d`, `64e8adfa`, `696a9d7c`, `7344d24f`, `9f57c612`, `d9781168`) — le coût fixe du numéro de schéma et du calque vide. `60ae07c4` fait **+452** : deux octets de plus, exactement le bloc de couverture « balayé, rien trouvé ». Les 4 films à véhicules montent de **+301 553** (`53ce4390`) à **+1 980 061** (`084a804d`) |
| `killsource` | **4/13** | CHANGE | **ÉCART HORS DE L'ATTENDU, MESURÉ ET EXPLIQUÉ** — voir ci-dessous |

**CORRÉLATION MESURÉE, PAS DÉDUITE** : `artifact` dépasse le plancher sur **EXACTEMENT** les
5 films dont `vehicles` n'est pas vide, et sur **AUCUN** des 8 autres. 13/13, zéro exception. Le
sha de `vehicles` est LE MÊME (`f73d0cef…`) sur les 8 films vides et DISTINCT sur les 5 autres :
le zéro est un zéro MESURÉ (le balayage a tourné et n'a rien trouvé), pas un balayage qui n'aurait
pas tourné.

**L'ÉCART `killsource`, ET POURQUOI IL N'EST PAS UNE RÉGRESSION DE L'INTÉGRATION.** Il tombe sur
`084a804d`, `1c4c63c2`, `53ce4390` et `a349fea8` — les **QUATRE films dont la bande `ti=40` est
peuplée**, et sur aucun autre (`60ae07c4`, dont la bande existe mais dont les entités ne
répliquent pas, ne bouge pas). Sa cause est une **CORRECTION DÉCLARÉE DE LA BRANCHE**, mesurée par
elle et arrivée avec elle (`filmdec/traverse.go`, lot V9 du 2026-09-03) : les deux composants
`-dynamic-precision-` d'orientation de `ti=38/39/40/43` étaient routés vers les désérialiseurs du
BIPÈDE, ce qui amputait `i2` de ses bits de tête et `i3` de son gate externe. Les vrais
désérialiseurs (`FUN_140c5f7ec` / `FUN_140d87740`) ont été résolus statiquement sous Ghidra et
validés 6/6 contre la table ECS live (`components_dynprec_orientation.go`). Le marcheur de records
que `killsource` emprunte (`consumeByName`) traverse donc ces deux composants autrement — et
seulement là où des entités `ti=38/39/40/43` en émettent. Les deux autres fichiers du chemin
`killsource` touchés par la branche (`frame_records.go`, `components_object.go`) sont **neutres au
bit près** (plomberie de sonde et extraction d'un décodeur : vérifié sur pièces).

**LES QUATRE FILMS À VÉHICULES, CHIFFRÉS** :

| film | variante | vies `ti=40` | échantillons | `artifact` | `killsource` |
|---|---|---|---|---|---|
| `084a804d` | BTB Heavies : CTF | 180 | 109 478 | 7 275 954 → **9 256 015** (+1 980 061) | bouge |
| `a349fea8` | BTB Heavies : Total Control | — | — | 6 897 326 → **8 709 326** (+1 812 000) | bouge |
| `1c4c63c2` | BTB : One Flag CTF | 67 | 111 822 | 7 433 969 → **8 301 577** (+867 608) | bouge |
| `53ce4390` | CTF : Arena (Behemoth) | 26 | 28 072 | 2 696 470 → **2 998 023** (+301 553) | bouge |
| `60ae07c4` | Ranked : Oddball (Live Fire) | 4 recensées, **0** échantillon | 0 | 3 035 677 → **3 036 129** (+452) | IDENTIQUE |

**AUCUN AUTRE DIGEST NE BOUGE** : `score`, `objectives`, `vip`, `skull`, `bomb`, `flag`, `zones`,
`zoneRoles`, `spawnPoints`, `spawnPointsState`, `neutralDeaths`, `killRefs`, `translocations`,
`positions`, `fire`, `loadouts`, `heldWeaponChanges`, `pickups`, `inventory`, `inventoryDeltas`,
`abilityRanks`, `equipmentChanges`, `camoStates`, `grappleReads`, `abilityImpulses`,
`abilityCharges`, `zoomEvents`, `placements`, `pads`, `carrierMarks`, `zoneReads`, `bombReads`,
`grenades`, `projectiles`, `deaths`, `playerIndices`, `clockOrigin` et les six canaux `.stats` sont
**IDENTIQUES AU BIT PRÈS** sur les 13 films. En particulier `positions` ne bouge pas : la
correction `i2`/`i3` dyn.-préc. ne touche que le marcheur ECS de `killsource`, jamais le nuage
bipède (qui lit `i0` AVANT `i2`).

**GATE DE LIGNES : les 13 TSV portent 49 étapes, tous** (48 + `vehicles`), et `minifilm.tsv` ses 7.

**MINI-BOBINE (étage CI) : AUCUN REFIGEAGE.** `TestEquivalenceMiniFilm` ne couvre que sept étapes
(`fire`, `grenades`, `loadouts`, `inventory`, `deaths`, `playerIndices`, `projectiles`) : aucune ne
bouge.

**PASSES ET DURÉES.** Passe 1 SANS `-update` : **0 identique / 13 différents / 0 écarté / 0 échec /
0 illisible** ; passe `-update` : 13 écrits, BILAN **13 identiques** ; passe de VÉRIFICATION sans
`-update` : **13 identiques, 0 différent, 0 écarté, 0 échec, 0 illisible** (exit 0). Témoins de la
passe de vérification : `01e1f945` **18,2 s**, `7344d24f` **22,2 s**, `696a9d7c` **22,9 s** —
cible < 100 s tenue (le gain du chantier survit au merge). Pics 0,08 à 0,67 Gio ; le plus long est
`084a804d` à 2 min 35,6 s, le BTB à 180 véhicules.

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

- 2026-09-05 — **ÉTAPE B exécutée (B.0 à B.5), un seul commit de merge** (union D9 : deux parents,
  `origin/feat/v75-vehicules-sons` `1e3d459d1` et `wt/vehicule-deadstate` `0b5141b8a`, sur
  `eb80a4f0a`). D8bis : `origin/feat/v75` MESURÉ inchangé à `7fb4b60a1` — zéro commit amont.
  387 fichiers, 26 conflits.

  **LES 26 CONFLITS ET LEUR RÉSOLUTION** (règle : la SÉMANTIQUE véhicules est préservée
  intégralement, dans NOTRE architecture, et POSÉE SUR LE 38 amont) :

  | conflit | ce que la branche voulait | résolution |
  |---|---|---|
  | `replay/build.go` (2 blocs) | ajouter `Vehicles VehicleScan` à `Options`, un appel de décodage à `BuildFromFilm`, et — dans `BuildFromPositions` — `buildShots` à 3 retours, `attachVehicles`, `attachVehicleShots`, le journal sur `doc.Coverage.Shots` | Le premier bloc EST celui que le lot 1 a sorti de `build.go`. Diff bloc-à-bloc base `14a115bb1` contre la branche : elle n'y touche QUE ces deux choses. Bloc supprimé ; le champ REJOUÉ dans `options.go` au rang de `Pads`, l'appel dans `build_from_film.go` juste après `pads`. Les hunks de `BuildFromPositions` avaient fusionné SEULS (vérifié : `shotOrphans`, `attachVehicles`, `attachVehicleShots`, `shotsPub` en place) ; le second conflit est l'UNION du bloc « impulsions + charges » (à nous) et de la ligne `shotsPub := doc.Coverage.Shots` (à elle) |
  | `replay/document.go` | trois blocs de doc de version (29 véhicules, 30 tirs embarqués, 31 visée d'occupant) et `SchemaVersion = 31` | Notre bloc 29→38 est conservé INTÉGRALEMENT ; les trois blocs de la branche FONDENT EN UN, « CE QUE LA VERSION 39 PORTE… EN TROIS TEMPS », avec les renvois de numéro corrigés (« un artefact 38 doit se lire à re-cuire ») ; `const SchemaVersion = 39`. Les champs `Vehicles`, `VehicleLabels` et `Shot.Vehicle` avaient fusionné seuls |
  | `replay/structure_test.go` | les trois mêmes blocs + `SchemaVersion != 31` | Même geste : nos blocs jusqu'à 38, puis les trois de la branche renumérotés `v39 (1)/(2)/(3)` sous une introduction qui dit la fusion, puis `!= 39` |
  | `contracttest/replay_contract_test.go` | `wantReplayDocumentFields` 44 → 46 (`vehicles`, `vehicleLabels`) | Notre 52 + les deux champs = **54**, avec l'entrée de chronique « 52 → 54 » et « Les vingt et une fois » |
  | `replay/testdata/assembly_000d5950.golden` | `schema 31` | `schema 39` |
  | `filmdec/offline_biped.go` (4 hunks) | scinder `ScanFilmBipedPositions` en `bipedScanChunks` / `bipedI0Layout` / `scanBipedChunks` pour ouvrir `ScanFilmBipedPositionsForBand(dir, band, opt)`, et ajouter l'option `DynPrecOrientation` | Structure de la branche reprise TELLE QUELLE, en forme FILM : `ScanBipedPositions(film, opt)` délègue à `ScanBipedPositionsForBand(film, band, opt)` ; les trois helpers prennent `*filmsource.Film` ; la bande est une `SlotBand` dense (le type du lot 2) et non une map. `DynPrecOrientation` conservé mot pour mot, sa grammaire RÉSOLUE UNE FOIS PAR PAYLOAD (hors de la boucle de records) au lieu d'une fois par record |
  | `filmdec/offline_aim.go` | `scanRecordDirs(pay, …, g dirsGrammar)` + deux lecteurs dyn.-préc. qui allouent chacun leur `BitReader` | Corps de la branche pris INTÉGRALEMENT ; la tête garde NOTRE lecteur partagé : `scanRecordDirs(br, …, g)`, et `readForwardComponentDynPrec` / `readAngularVelocityComponentDynPrec` reçoivent `br` au lieu d'allouer — deux allocations de moins par record, exactement la raison d'être du lecteur partagé |
  | `filmdec/offline_aim_test.go` | deux appels à `scanRecordDirs` | Signature à cinq paramètres, lecteur partagé |
  | `.ai/thought_log.md` (2 blocs) | deux blocs insérés aux mêmes ancres | UNION : nos entrées en tête, puis celles de la branche, ligne de séparation rétablie ; l'entrée V13 du dead-state (second parent) insérée en tête de la section du chantier |
  | `.ai/V7.5/REGISTRE_REPORTS.md` | une ligne de report (cap i2 réfuté) | UNION du tableau |
  | `.gitignore` | un commentaire sur `.gocache-*/` | UNION (le motif était déjà chez nous, le commentaire est repris) |
  | `api/openapi.yaml`, `web/lib/api/generated.ts` | contrat régénéré au schéma 31 | D11 : `--ours` puis **régénération** (`make openapi-gen && make generate-types && make openapi-check`), JAMAIS à la main |
  | 13 fichiers `web/features/match-replay/` | le calque véhicules dans le tiroir, le contrat, la normalisation, les marqueurs, le cône | UNION partout. Trois résolutions non triviales : (a) `ReplaySettingsLayers.tsx` — la branche pose une liste PLATE, nous des `LayerGroup` ; la bascule `vehicles` entre dans le groupe TERRAIN (l'intention écrite de la branche : « meubles du terrain, pas l'enjeu du mode ») ; (b) `replayAimCone.ts` — la branche extrait `drawAimSector` pour que le calque véhicules réutilise le cône, nous avions ajouté la LUNETTE (ouverture variable + gain d'opacité) ; le secteur extrait reçoit deux options de plus, `halfAngle` et `alphaBoost`, et le cône du pion les lui passe : une seule géométrie, les deux mécaniques intactes ; (c) `ReplayCanvas.tsx` — `showNames` n'existe plus (le calque des noms a quitté le tiroir le 02/09, toujours allumé) : le calque véhicules reçoit `showNames: true` |

  **DEUX RUPTURES DE COMPILATION LAISSÉES PAR L'AUTO-MERGE**, toutes deux dans des tests de la
  branche appelant des symboles disparus au lot 1 — intention rejouée, jamais contournée :
  `filmdec/event_list_test.go` redéclarait `itoa` (le nôtre vit dans
  `lot1_visee_calib_research_test.go` : le doublon est retiré, la mesure est identique) ; et
  `killsource/vehicules_v10_deadstate_test.go` appelait `DirChunks(dir)`, supprimé au lot 1 —
  remplacé par `filmsource.LoadDir(dir, nil)`, la source que `loadFilm` prend désormais.

  **LES BALAYAGES MIGRÉS (D2)**, avec l'enveloppe conservée et son unique raison d'être :

  | ancien (site) | nouveau | enveloppe `(dir)` gardée pour |
  |---|---|---|
  | `ScanFilmWorldObjectKeyframes(dir, 40)` — `build_vehicles.go:84` | `ScanWorldObjectKeyframes(film, VehicleTypeIndex)` — existait déjà (`world_object_census.go:79`) | déjà à la liste depuis le lot 1 |
  | `ScanFilmVehicleCreationsForBand(dir, wr, band)` — `build_vehicles.go:90` | `ScanVehicleCreationsForBand(fc, wr, band)` — `vehicle_creation.go` | AUCUNE : l'enveloppe n'avait pas d'appelant, elle est SUPPRIMÉE (N-6) |
  | `ScanFilmBipedPositionsForBand(dir, band, opt)` — `build_vehicles.go:96` | `ScanBipedPositionsForBand(film, band, opt)` — `offline_biped.go` | `filmdec/offline_biped_test.go`, `vehicle_creation_test.go`, huit instruments `vehicules_v*` de `filmdec` et sept de `replay` |
  | `ScanFilmBipedAimOnly(dir)` — `build_vehicles.go:125` | `ScanBipedAimOnly(fc)` — `offline_aim_only.go` (bande bipède du CONTEXTE) | `filmdec/vehicules_v11_scan_test.go` |
  | `ScanFilmVehicleEvents(dir)` — `build_vehicles.go:139` | `ScanVehicleEvents(fc)` — `event_list.go` (bande bipède du CONTEXTE) | `filmdec/event_list_test.go`, `event_list_board_test.go`, `replay/vehicules_v2b_occupant_test.go`, `vehicules_v3_destruction_test.go` |
  | `ScanFilmVehicleCreations(dir, wr)` | `ScanVehicleCreations(fc, wr)` | `replay/vehicules_v2b_cooldown_test.go` |
  | `ScanFilmKeyframeRecordSpans(dir)` | `ScanKeyframeRecordSpans(film)`, puis SUPPRIMÉ (N-6) | — |
  | `ScanFilmVehicleOccupancy(dir)` | `ScanVehicleOccupancy(film)`, puis SUPPRIMÉ (N-6) | — |

  `decodeFilmVehicleScan` ne touche plus le disque : elle prend le `*FilmContext` et fait dessus
  ses CINQ lectures (recensement, créations, nuage, événements, visées) ; le découpage d'i0 du
  nuage `ti=40` vient désormais du CONTEXTE (`ImposedLayout`) et non de l'auto-détection — la même
  règle du catalogue que les positions bipèdes du même film (lot 3). Le nom garde le préfixe
  `decodeFilm*` de ses cinq sœurs (`decodeFilmPadScans`, `decodeFilmPlacements`, …), qui est aussi
  ce que lit `observe_test.go`.

  **UNE FACTORISATION, PARCE QUE C'ÉTAIT LA TROISIÈME COPIE** : la boucle de marche des records de
  création existait pour l'équipement et pour l'arme au sol ; le véhicule en aurait été la
  troisième. `runCreationWalk(fc, w, &st)` est devenu le point de passage UNIQUE des trois, chacun
  gardant ses refus en propre (règle n°6 du dépôt).

  **GARDES TOUCHÉES** :
  - `archlint/no_film_reread_test.go` — `enveloppesInterditesEnProduction` **43 → 47** :
    `ScanFilmBipedPositionsForBand`, `ScanFilmBipedAimOnly`, `ScanFilmVehicleEvents`,
    `ScanFilmVehicleCreations`, avec la justification datée et la vérification d'homonymie exigée
    par le fichier (aucune méthode ne porte ces noms — vérifié). Les TROIS enveloppes sans appelant
    ne sont PAS entrées à la liste : elles ont été supprimées (N-6).
  - `archlint/filmdec_package_vars_test.go` — ratchet **116 → 118**, avec les deux variables
    nommées et justifiées : `unitRefHook` (une SONDE, nil en production, le patron déjà compté du
    paquet) et `vehicleMediaFrameBits` (une TABLE DE GRAMMAIRE mesurée). L'intégration n'en ajoute
    AUCUNE de son fait — la seule erreur sentinelle qu'elle allait poser est restée locale.
  - `archlint/no_recomputed_film_context_test.go` — l'entrée
    `offline_biped.go/ScanBipedPositions -> DetectI0LayoutOf` devient
    `offline_biped.go/bipedI0Layout -> DetectI0LayoutOf` : MÊME appel, MÊME valeur, le site a
    seulement été extrait pour que les deux entrées le partagent. Aucune allowlist élargie.
  - `archlint/no_rewritten_slot_band_test.go` — une entrée d'allowlist de plus,
    `vehicle_creation_test.go`, pour sa bande FANTÔME : c'est le NÉGATIF d'une bande (le témoin
    d'ancrage), pas une quatrième règle — exactement le cas déjà admis pour
    `equipment_creation_test.go`.
  - `replay/observe.go` — `BuildFromFilmSteps` **34 → 35** : `vehicles` entre APRÈS `pads`, à
    l'endroit exact où le balayage tourne. **Pas d'étape pour les tirs en véhicule ni pour la
    couverture** : ils vivent dans `BuildFromPositions`, que l'observateur ne couvre pas, et le
    digest `artifact` les porte.
  - `web/placementFamily.guard.test.ts` — plafond de `ReplayCanvas.tsx` **INCHANGÉ à 665** ; le
    fichier en fait 664 après le câblage véhicules (N-7).
  - `no_unbounded_film_loop_test.go`, `filmsource_leaf_test.go`, `no_rewritten_slot_band_test.go`
    (règle 2), `no_art_patterns_test.go` : verts sans modification.
