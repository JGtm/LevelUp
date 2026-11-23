# Contrôles 0.5 et 0.6 du plan v1.6.0 (lecture seule)

Date : 2026-11-23. Plan : `LevelUp-wt-int/.ai/PLAN_DECODEUR_V160_2026-11-23.md`.
Décodeur : `Scripts/dec-v160`, branche `feat/v1.6.0`, HEAD `dc967c2`, identique au tag `v1.5.0`
(tag annoté, pointe sur `dc967c2`, vérifié par `git log -1 v1.5.0`). Arbre de travail propre.
Abréviations : DEC = `dec-v160`, LU = `LevelUp-wt-int` (là où se trouve le plan ; les chemins LU
ci-dessous sont dans ce worktree sauf mention contraire), WT160 = `LevelUp-wt-dec160`.

Méthode : lecture des lignes citées (sed), recherches par Grep, vérification d'existence par
globbing shell. Aucun test lancé, aucun film lu, aucune modification du dépôt.

---

## 1. Item 0.5 : références `fichier:ligne` de la section 1.1

Légende : Juste = la ligne contient le symbole ; Décalé = le symbole est à une autre ligne ;
Fausse = le symbole n'est pas là ; Partielle = une partie juste, une partie fausse.

| Référence du plan | Verdict | Ce qu'on trouve (vérifié) |
|---|---|---|
| DEC `internal/source/film.go:103-128` (`Load`) | Juste | `func Load` à 103, fin à 128 |
| DEC `film.go:217-246` (zlib, bibliothèque standard) | Partielle | `compress/zlib` importé ligne 10 ; `zlib.NewReader` à 188 (`Decompresser`). 217-246 = `inflate`, qui choisit zstd ou zlib |
| DEC `decfilm/decfilm.go:466,467,470` (`Inflate`, `Load`, `LoadDir`) | Juste | 466 `Inflate`, 467 `Load`, 470 `LoadDir` |
| DEC `internal/source/zstd.go` (klauspost) | Juste | import `github.com/klauspost/compress/zstd` ligne 35 |
| DEC `memguard/memguard.go:175` (seul `go` hors tests) | Juste | grep des `go` hors `_test.go` : une seule occurrence, `go g.watch(...)` |
| DEC `internal/grammar/marche_trames.go:109-150` (`NewWorld`, `parcourir`, `PoserChunkCourant`) | Partielle | 109 = `marcheurDesTramesDepuis` ; `parcourir` = marche_trames.go:132 (méthode de `marcheurDesTrames`) ; `NewWorld` = `internal/grammar/world.go:94` ; `PoserChunkCourant` = `world.go:120` |
| DEC `internal/grammar/distribuer.go:1-70` (une marche, N canaux) | Juste pour l'en-tête | 1-70 = commentaire d'en-tête ; la fonction `Distribuer` est à `distribuer.go:142` |
| DEC `replay/options.go:27` (`Options`, sans sélection) | Juste | `type Options struct` à 27 ; struct de 468 lignes ; aucun champ de choix de canaux ni de fils (grep) |
| DEC `internal/facts/killsource/decode.go:360` (`LireLaMarcheCommune`) | Juste comme site d'appel | la ligne 360 l'appelle ; la définition est `internal/grammar/marche_commune.go:89` |
| DEC `replay/build_from_film.go:98` (`PrendrePourLaCuisson`) | Juste comme site d'appel | appel à la ligne 98 ; définition `internal/grammar/marche_commune.go:76` |
| DEC `replay/build_from_film.go:51,63` | Juste | 51 `BuildFromFilm`, 63 `BuildFromFilmAvecFaits` |
| DEC `replay/build_from_facts.go:52` | Juste | `BuildFromFacts` |
| DEC `replay/options.go:461-464` (`Observe`) | Juste | champ `Observe Observer` à 464 |
| DEC `replay/observe.go:97` (étape `clockOrigin`) | Décalé | 97 = `BuildFromFilmSteps` (copie de la liste). La liste est aux lignes 47-92 ; `clockOrigin` est à 92 ; l'émission est dans `replay/film_scan_pont.go:175` |
| DEC `research/duree/duree_research_test.go` | Juste | `//go:build research` ; le commentaire décrit bien `clockOrigin` |
| DEC `internal/grammar/bench_*_test.go` (six `Benchmark*`, quatre fichiers) | Juste | 6 fonctions dans 4 fichiers (`bench_balayage_bits`, `bench_decodeur` x3, `bench_objets_du_monde`, `bench_recuperation_ancree`) |
| DEC `internal/grammar/marche_determinisme_test.go` (déterminisme ; deux films en parallèle sous `-race`) | Partielle | le déterminisme est bien là : `TestLaStructureEstDeterministe` (75). Le test `-race` est `TestDeuxFilmsEnParallele`, dans `internal/grammar/deux_films_parallele_test.go:51`, pas dans ce fichier |
| DEC `.github/workflows/ci.yml:66-78` (job `film-race`) | Juste | job `film-race:` à la ligne 68, commande `go test -race ... -run TestDeuxFilmsEnParallele` vers la ligne 78 |
| DEC `internal/grammar/rev_test.go:1-45` (empreinte des jetons) | Juste | en-tête du gate ; la mention « JETONS » est à la ligne 38 |
| DEC `replay/score_timeline_players.go:181-188` (`joinSegments`) | Juste | appel à 181, définition `joinSegments` à 188 |
| LU `apps/go-api/internal/domain/title/registry_film_facts.go:52-77` | Juste | constante `SousDossierFilmFacts` à 52 ; `FilmFactsDir` à 67 ; `FilmFactsPath` à 76 |
| LU `apps/go-api/cmd/replay-equiv/main.go:136-163` | Juste | `lireDrapeaux` à 136 ; `-deux-passes` à 156 ; aucun réglage de fils |
| LU `apps/go-api/cmd/replay-build/main.go:59-72` | Juste | drapeaux de 59 à 72 ; aucun réglage de fils |
| LU `apps/go-api/cmd/replay-corpus-gate/main.go:117-132` | Juste | drapeaux de `main` à partir de 117 ; aucun réglage de fils |
| LU `.../replaychild/replaychild.go:68-121` | Juste, chemin à préciser | le fichier est `apps/go-api/internal/replaychild/replaychild.go` ; `memguard.ArmBytes` à 119, `memguard.Arm` à 121 |
| LU `apps/go-api/cmd/replay-corpus-gate/bake.go:72` (4 Gio) | Juste | `const plafondMemoireGate = 4` |
| Plafonds 3 Gio prod / 2 Gio mesure (DEC `memguard/memguard.go`) | Juste | `DefaultLimitGiB = 3` (ligne 39), `MeasureLimitGiB = 2` (ligne 45) |
| LU `.ai/REGISTRE_REPORTS.md:362-363` | Juste | 362 = phase des images-clés (trois lecteurs, environ 30 %) ; 363 = sonde de calibration (1,5 %) |
| LU `.ai/annexes_marche_complete/temoins_duree.txt` | Présent | 10 lignes de témoins hors commentaires (« dix témoins » confirmé) |
| DEC `replay/testdata/equivalence/` | Présent | 43 fichiers (non compté film par film) |
| DEC « aucun dossier `cmd/` » | Juste | pas de `cmd/` à la racine de DEC |
| DEC « aucun catalogue de bornes de carte » | Non vérifié | non cherché |

Bilan 0.5 : 0 référence fausse de bout en bout. Décalées ou partielles : `film.go:217-246`,
`marche_trames.go:109-150` (deux symboles sur trois sont ailleurs), `observe.go:97`,
`marche_determinisme_test.go` (le test `-race` est dans un autre fichier), `replaychild.go` (chemin
incomplet). Les autres sont justes.

---

## 2. Item 0.6 (première partie) : table des fichiers par piste

### 2.1 Existence des motifs

Absents, attendus car neufs : `cmd/film-bench/`, `bench/` (V0), `replay/document_repliques*.go` (D.3).

Absents, MAIS la table les donne à un mauvais chemin :

| Motif de la table | Réel | Piste |
|---|---|---|
| `layers.go` (lu par V4, « écrit par le pilote ») | `replay/layers.go` (pas dans `internal/grammar/`) | V4, pilote |
| `eliminations.go` (E.6 et ligne E) | `replay/eliminations.go` | E |
| `flag_assign.go` (ligne B, B.3c) | `replay/flag_assign.go` (pas dans `internal/grammar/`) | B |
| `zone_states_lettres.go` (B.6) | `replay/zone_states_lettres.go` | B |
| `projectiles.go` (contributeur) | `replay/projectiles.go` | contributeur |
| `distribuer_tetes.go` (contributeur) | `internal/grammar/distribuer_tetes.go` | contributeur |
| `porteurs_entrees.go` (B.5) | `replay/porteurs_entrees.go` (il n'existe pas dans `internal/`) | B |

Tous les autres motifs existent (ex. `internal/source/{film,source,zstd}.go`, `filmcache/`, `decfilm/`,
`replay/film_scan*.go` (7 fichiers), `internal/grammar/marche_images_cles*.go`,
`keyframe_etat_complet_*.go` (4), `keyframe_loadout.go`, `marche_trames*.go` (10),
`marche_commune.go`, `marche_de_killsource.go`, `distribuer*.go` (5), `world.go`, `keyframe_liaison.go`,
`keyframe_anticipe.go`, `reponsesdejeu/`, `replay/document_score_actions*.go`, `replay/score_timeline*.go`,
`internal/facts/objectives/score.go`, `replay/flag_*.go` (~45), `replay/porteurs_*.go`,
`replay/zone_states_*.go` (~45), `internal/grammar/components_navpoint*.go`,
`internal/grammar/frame_vue_controle.go`, `replay/document_stances*.go`, `internal/grammar/vue_a_genres.go`,
`vue_a_lecture.go`, `replay/document_eliminations.go`, `replay/eliminations.go`,
`internal/grammar/film_variante_de_partie.go`, `internal/facts/objectives/named*.go` (12),
`internal/facts/fallback/`, `internal/depottest/`, `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md`, `README.md`).

Côté LU (vérifié dans WT160 et LU, identique) : `cmd/weapon-sounds/`, `config/replay_corpus.toml`,
`scripts/briques-workspace.sh`.

### 2.2 Conflits (un fichier revendiqué par deux pistes, ou un motif qui en recouvre un autre)

| # | Fichier ou motif | Pistes | Nature |
|---|---|---|---|
| C1 | `replay/film_scan_images_cles.go` | V3 (écrit) et V4 (`replay/film_scan*.go`, écrit) | Le motif de V4 englobe le fichier de V3. V3 passe après V4 (plan), mais la ligne V4 doit exclure ce fichier. |
| C2 | `decfilm/` | V0 (structure d'options) et V1 (entrée de chargement) | Deux pistes écrivent le même dossier. V0.6 est seul avant toutes les autres, donc à écrire : V1 doit seulement ajouter. |
| C3 | `named*.go` (`internal/facts/objectives/`) | E.2 (ligne E) et B.4 (item B) | La ligne B ne le liste pas, mais B.4 l'écrit. E.2 et B.4 sont sur le même fichier : ordre à fixer. |
| C4 | `score_timeline*.go` | A.5 (ligne A) et E.2 (item E) | E.2 écrit le fichier, la ligne E ne le liste pas. E.2 « après A », donc ordre seulement. |
| C5 | `internal/grammar/marche_de_killsource.go` | V2 (ligne), V4 (appelant de `Distribuer` à la ligne 104) | Conflit réel. La ligne V4 dit « ne touche pas `marche_*.go` », ce qui est faux pour ce fichier. |
| C6 | Appelants de `Distribuer` hors liste | V4 doit les servir (0.7, V4.2) | `internal/grammar/equipement_porte.go:306`, `lectures_bipedes.go:171`, `objets_du_monde_lus.go:211` : absents de toute ligne. |
| C7 | `internal/grammar/movement_states.go` | V4 (écrit), contributeur (écrit) | Conflit avec le contributeur, qui entre après le gel (pas bloquant, à noter). |
| C8 | `distribuer_tetes.go`, `distribuer_crochets.go` | V2 (glob `distribuer*.go`), contributeur (`distribuer_tetes.go`) | V2 touche le motif entier, donc ces deux fichiers. |
| C9 | `replay/film_scan*.go` | V4 (écrit), contributeur (`film_scan_*.go`) | Conflit avec le contributeur, après le gel. |
| C10 | `replay/document_stances*.go` | C (écrit), contributeur (`document_stances.go`) | Idem, après le gel. |
| C11 | `internal/grammar/vue_a_lecture.go` | D (écrit), contributeur | Idem, après le gel. |
| C12 | `keyframe_etat_complet_bipede.go` | V3 (glob `keyframe_etat_complet_*.go`), contributeur | Idem, après le gel. |
| C13 | `replay/options.go` | V0.6 (écrit), contributeur (écrit) | V1, V2, V4 le lisent. Idem, après le gel. |
| C14 | `FORMAT.md`, `NOTICE.md` | D.4 (`FORMAT.md`, `GLOSSARY.md`, `NOTICE.md`) ; V4.5 (`README.md`, `NOTICE.md`, et « `FORMAT.md` n'est pas touché ») | Contradiction : V4.5 dit que FORMAT.md n'est pas touché, D.4 l'écrit. NOTICE.md est écrit par V4.5 et D.4. Les fragments de crédit de V4.5 et D.4 sont dans les mêmes fichiers. |
| C15 | `replay/coverage.go` | B.6 (deux compteurs de couverture) ; plan : « fichiers à écrivain unique : `coverage.go` » | B.6 écrit dans un fichier réservé au pilote. Ligne B absente. |

### 2.3 Fichiers qu'une piste touchera visiblement sans être dans sa ligne

| Piste | Fichier à ajouter | Raison (vérifiée par Grep) |
|---|---|---|
| V4 | `internal/grammar/equipement_porte.go`, `lectures_bipedes.go`, `objets_du_monde_lus.go` | Appels à `Distribuer` (C6) |
| V4 | `internal/grammar/marche_de_killsource.go` | Appel à `Distribuer` ligne 104 (C5) |
| V4 | `replay/observe.go` (lu) | Liste fermée des étapes, source de « clockOrigin » (V4.1) |
| V4 | `replay/layers.go` (lu, et pilote) | Table des calques (V4.1, V4.3) |
| V0 | `internal/depottest/` | Gate V0 le teste, à confirmer si l'on y écrit |
| V2 | `replay/observe.go` | Ordre des étapes (« crochets des canaux ») |
| B | `replay/coverage.go` (pilote seul) | B.6 (C15) |
| B | `replay/porteurs_entrees.go`, `replay/zone_states_*.go` | Déjà présents, mais B.5 et B.6 écrivent dans `replay/` : la ligne B les couvre, à confirmer |
| A | `replay/document_score_actions*.go` (déjà), `replay/score_timeline_players.go` | `joinSegments` (188) est dans `score_timeline_players.go`, touché par A.5 |
| E | `replay/score_timeline*.go` | E.2 (C4) |
| E | `internal/facts/objectives/extract.go`, `decfilm/decfilm.go` (lu) | `ObjectiveTypeVip` (E.2) : constante à lire, pas à écrire |
| D | `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md` | D.4 (C14) |
| D | `cmd/weapon-sounds` (LU) pour `cmd/` des répliques | D.5 : une commande neuve à côté (à confirmer) |
| F | (ligne absente) `internal/facts/fallback/` (`registre.go`, `compteur.go`, `noms.go`, `registre_filmdec.go`), LU `scripts/briques-workspace.sh`, LU `config/replay_corpus.toml` | F.1, F.2, F.3 : aucune ligne pour la piste F |
| P | Ligne « contributeur » : la référence « section 1.3 » ne contient aucune liste de fichiers (1.3 est un tableau des briques) | Liste à reconstituer ou à sourcer ailleurs |

### 2.4 Table corrigée (proposition)

| Piste | Écrit dans DEC | Écrit dans LU | Lu seulement | Ne touche pas |
|---|---|---|---|---|
| V0 | `cmd/film-bench/` (neuf), `bench/` (neuf), `replay/options.go` (V0.6), `decfilm/` (V0.6 seul) | `cmd/replay-equiv`, `cmd/replay-build`, `cmd/replay-corpus-gate` (V0.7) | `internal/depottest/` | le reste |
| V1 | `internal/source/{film,source,zstd}.go`, `filmcache/`, `decfilm/` (une entrée ajoutée) | — | `replay/options.go` | `internal/grammar/`, `replay/` |
| V4 | `replay/film_scan*.go` **sauf** `film_scan_images_cles.go` (C1), `replay/build_from_film.go`, `internal/grammar/movement_states.go`, les appelants de `Distribuer` (`equipement_porte.go`, `lectures_bipedes.go`, `objets_du_monde_lus.go`, `marche_de_killsource.go` : C5, C6) ; V4.5 : `README.md`, `NOTICE.md` (C14) | — | `replay/layers.go`, `replay/observe.go` | `internal/source/`, `distribuer*.go` (le distributeur), `marche_images_cles*.go`, `marche_trames*.go` (sauf ce qui appelle `Distribuer`) |
| V3 | `internal/grammar/marche_images_cles*.go`, `keyframe_etat_complet_*.go`, `keyframe_loadout.go`, `replay/film_scan_images_cles.go` (après V4) | — | — | `marche_trames*.go`, `keyframe_liaison.go`, `internal/source/` |
| V2 | `internal/grammar/marche_trames*.go`, `marche_commune.go`, `marche_de_killsource.go` (C5), `distribuer*.go` (C8), `world.go`, `keyframe_liaison.go`, `keyframe_anticipe.go` | — | `replay/observe.go` | `internal/source/`, `replay/` |
| A | `reponsesdejeu/`, `replay/document_score_actions*.go`, `replay/score_timeline*.go`, `internal/facts/objectives/score.go`, `replay/score_timeline_players.go` (`joinSegments`) | — | — | drapeau, zones |
| B | `replay/flag_*.go`, `replay/porteurs_*.go`, `replay/zone_states_*.go`, `internal/grammar/components_navpoint*.go`, `internal/facts/objectives/named*.go` (B.4, C3), `replay/zone_states_lettres.go`, `replay/flag_assign.go` | — | `replay/coverage.go` : pilote seul (C15) | score |
| C | `internal/grammar/frame_vue_controle.go`, `replay/document_stances*.go` | — | — | vue A |
| D | `internal/grammar/vue_a_genres.go`, `vue_a_lecture.go`, `replay/document_repliques*.go` (neufs), `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md` (C14) | `cmd/` des répliques (D.5, nouveau) | — | vue de contrôle |
| E | `replay/document_eliminations.go`, `replay/eliminations.go`, `internal/grammar/film_variante_de_partie.go`, `internal/facts/objectives/named*.go` (E.2, après A et après B.4 : C3), `replay/score_timeline*.go` (E.2 : C4) | — | `decfilm/decfilm.go` (`ObjectiveTypeVip`) | drapeau |
| F | `internal/facts/fallback/` | LU `scripts/briques-workspace.sh`, LU `config/replay_corpus.toml` | — | — |
| Contributeur | `replay/options.go`, `replay/film_scan_*.go`, `replay/document_stances.go`, `replay/shots*.go`, `replay/projectiles.go`, `internal/grammar/vue_a_lecture.go`, `internal/grammar/distribuer_tetes.go`, `internal/grammar/movement_states.go`, `internal/grammar/keyframe_etat_complet_bipede.go`, `filmcache/`, `decfilm/`, `replay/document_schema.go`, `replay/document_chronicle.go`, faits, goldens | — | — | — |

Fichiers à écrivain unique (pilote seul), à retirer des lignes de piste : `replay/document_schema.go`,
`replay/document_chronicle.go`, `replay/layers.go`, `replay/coverage.go`, `replay/filmfacts*.go`, les goldens,
les fichiers de révision de couche. Le plan les liste déjà, mais les pistes B et D les écrivent (C15).

Dépendances de pistes : V1, V2, V4 lisent les options de V0.6 ; V2 passe après V4 (distributeur et
appelants) ; V3 après V4 (`film_scan_images_cles.go`). À reporter sur la table.

---

## 3. Item 0.6 (troisième partie) : noms des étapes observées

Source : `replay/observe.go` (liste fermée, DEC, lignes 47 à 92) ; `internal/replaybuild/etapes_observees.go`
(WT160) ne porte que les étapes AVANT et APRÈS la marche de décodage. La liste du milieu n'est donc pas
dans ce fichier. Ordre réel : avant (13), milieu (`buildFromFilmSteps`), après (4).

**Étapes AVANT** (`etapes_observees.go`, `buildBytesStepsBefore`) :
score, objectives, vip, skull, bomb, flag, zones, zoneRoles, killsource, spawnPoints, spawnPointsState, neutralDeaths, killRefs

**Étapes du MILIEU** (`buildFromFilmSteps`, observe.go), dans l'ordre d'appel. En gras : tirs, touches, dégâts.

1. translocations
2. knockbacks
3. positions
4. bipedCreations
5. **fire** (tirs ; `film_scan.go:73`)
6. loadouts
7. birthLoadouts, birthLoadouts.stats
8. zoomEvents
9. placements, placements.stats
10. spawnEvents
11. pads, vehicles, carrierMarks, zoneReads, zoneKeyReads
12. flagGauge, flagGaugeWalk
13. bombReads
14. movementStates, movementStates.stats
15. **continuousFire**, continuousFire.stats (tir continu ; `film_scan_mouvement.go:78`)
16. controlHolds, controlHolds.stats
17. unitControls, unitControls.stats
18. bipedDeletions, bipedDeletions.stats
19. horloges
20. **gameEvents**, gameEvents.stats (messages de jeu ; `film_scan_mouvement.go:87`)
21. **damages**, damages.stats (dégâts ; `film_scan_mouvement.go:89`)
22. reloads, reloads.stats
23. filmScripts
24. vehicleDeaths, vehicleDeaths.stats
25. heldWeaponChanges, heldWeaponChanges.stats
26. pickups, pickups.stats
27. inventory, inventory.stats
28. inventoryDeltas, inventoryDeltas.stats
29. abilityRanks, abilityRanks.stats
30. equipmentChanges, equipmentChanges.stats, carriedEquipment
31. camoStates, camoStates.stats
32. grappleReads, grappleReads.stats
33. abilityImpulses, abilityCharges
34. grenades, projectiles
35. detonations
36. deaths, filmTable, playerTeams, rosterListings, rosterListings.stats, statborgEntries, statborgEntries.stats, playerIndices
37. clockOrigin (dernière du balayage)

**Étapes APRÈS** (`etapes_observees.go`, `buildBytesStepsAfter`) :
filmFactsRejoue (constante `EtapeRejeuDepuisLesFaits`, WT160 `internal/replaybuild/filmfacts_cuisson.go:77`), scoreTimeline, eliminations, artifact

Marquage et statut :
- **Tirs** : `fire`, `continuousFire`. Vérifié : les deux `observe` existent (lignes ci-dessus).
- **Dégâts** : `damages`, `damages.stats`. Vérifié : `observe` à `film_scan_mouvement.go:89`. La commit `591a121` (schéma 125) dit que `damages` et `damages.stats` sont des étapes nouvelles.
- **Touches** : `gameEvents`, `gameEvents.stats` (à confirmer). Supposé, non prouvé ligne à ligne : le calque « touches » (`replay/coverage.go:185`, « Hits ») dépend des messages de dégât lus, donc du flux `gameEvents`. Le lien exact est à établir par Grep dans la piste.
- Hors marquage, mais proches : `reloads` (rechargements), `projectiles`, `detonations`.

Le plan (0.6) dit « noms des étapes de tirs, de touches et de dégâts de `replay-equiv` ». `replay-equiv` ne contient
pas la liste : il la reçoit par `b.WithObserver(col.etape)` (`cmd/replay-equiv/child.go:157`) et compare les lignes
par nom. La liste ci-dessus est celle que la marche émet.

---

## 4. Outil de compatibilité publique (G-compat)

- `gorelease` : non installé. Absent du PATH (`which`) et de `le dossier des binaires Go du poste`.
- `apidiff` : non installé. Même constat.
- `GOBIN` : vide. `GOPATH` : le dossier Go du poste. `go version` : go1.26.11 windows/amd64.
- Présents dans `go\bin` : air, benchstat, gitleaks, goimports, golangci-lint, govulncheck, lefthook, oapi-codegen.
- Le cache de modules contient `golang.org/x/exp@v0.0.0-20260112195511-716be5621a96` et `golang.org/x/tools@...`. Non vérifié si l'un contient les sources de `gorelease` ou d'`apidiff` ; rien n'a été installé.
- Conclusion : G-compat exige une installation (ou le repli du plan, non décidé ici). Rien n'a été installé.

---

## 5. Ce qui reste non vérifié

- Le compteur d'« émissions jetées » juste hors Drapeau au tag `e703301` (0.5) : le commit existe (`git cat-file -t` = commit), le compteur n'a pas été recalculé.
- « Aucun catalogue de bornes » dans DEC : non cherché.
- La table de vocalisation « 5 noms résolus » (D.2) : introuvable dans DEC par Grep (`vocal|voice|chatter|speech|sayings`). Son emplacement est inconnu.
- Le « parseur voisin » (`src/theater/...`, `parser/v41/chunks.rs`) cité par V1.1, V2.1, V3.1 : dépôt externe, non ouvert.
- La liste exacte de ce que « touches » désigne dans `replay-equiv` : supposée, voir section 3.
- Aucun fichier du plan n'a été modifié. Aucun commit, aucun push, aucun test lancé.
