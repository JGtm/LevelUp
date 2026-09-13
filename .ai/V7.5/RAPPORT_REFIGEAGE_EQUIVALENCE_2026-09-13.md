# Rapport — re-figeage du corpus d'équivalence et classification des écarts

> Lot 0.A.1c du `PLAN_DECODEUR_FILM_2026-09-13.md`, exécuté le 2026-09-13 sur la branche
> `feat/decfilm-0A` (worktree `LevelUp-wt-decfilm-0A`). Références re-figées au commit
> d'intégration `cbfdc269d`, schéma 54. Les anciennes références (lot 4b, `179bd7401`,
> schéma 34) restent lisibles par `git show cbfdc269d:<chemin du .tsv>`.
>
> **Objet** : distinguer, écart par écart, les DIVERGENCES VOULUES (contenu cuit changé par
> une montée de schéma documentée) des CONSTATS DE RÉGRESSION (une perte que rien ne
> documente). Ce rapport ne corrige rien.

## 0. Verdict

**Aucun constat de régression.** Les 110 écarts mesurés (sur 650 couples film × étape) se
rattachent tous à une entrée documentée. Les seules BAISSES de compte sont :

| Baisse | Films | Entrée qui l'explique |
|---|---|---|
| `objectives` → 0 | `084a804d`, `1c4c63c2`, `51101d1d`, `9f57c612` | Garde d'effectif, commit `ebd012e3b` (lot 6.7-B1 item 5) |
| `projectiles` −11 | `60ae07c4` (Live Fire) | Chronique **v53** (porte d'i0, `regionIndexBits` = 2 sur Live Fire) |
| `artifact` (octets) | `1c4c63c2`, `51101d1d`, `7344d24f`, `a349fea8` | Grandeur DÉRIVÉE (longueur du document), conséquence des deux lignes ci-dessus et des changements de contenu amont — aucune couche n'a perdu d'entrée sur ces films |

Preuve de complétude : le tableau de la §1 porte les 50 étapes × 13 films ; toute cellule
n'est ni `.` (identique) ni `hausse`/`contenu`/`NEW` que dans les trois cas ci-dessus.

## 1. Tableau complet — 13 films anciens × 50 étapes

Légende : `.` = digest identique · `contenu` = compte égal, sha différent · `hausse` = compte
en hausse · `BAISSE` = compte en baisse (candidat perte) · `NEW` = étape neuve.

| etape | 000d5950 | 01e1f945 | 64e8adfa | 7344d24f | 696a9d7c | 084a804d | 1c4c63c2 | 53ce4390 | d9781168 | 9f57c612 | 60ae07c4 | 51101d1d | a349fea8 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `score` | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu |
| `objectives` | . | . | hausse | . | . | BAISSE | BAISSE | hausse | . | BAISSE | . | BAISSE | contenu |
| `vip` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `skull` | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu |
| `bomb` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `flag` | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu |
| `zones` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `zoneRoles` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `killsource` | . | . | . | . | . | contenu | contenu | . | . | . | . | . | contenu |
| `spawnPoints` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `spawnPointsState` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `neutralDeaths` | . | . | . | . | . | hausse | . | . | . | . | . | . | . |
| `killRefs` | . | . | . | . | . | contenu | contenu | . | . | . | . | . | . |
| `translocations` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `positions` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `bipedCreations` | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW | NEW |
| `fire` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `loadouts` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `heldWeaponChanges` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `heldWeaponChanges.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `pickups` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `pickups.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `inventory` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `inventory.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `inventoryDeltas` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `inventoryDeltas.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `abilityRanks` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `abilityRanks.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `equipmentChanges` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `equipmentChanges.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `camoStates` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `camoStates.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `grappleReads` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `grappleReads.stats` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `abilityImpulses` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `abilityCharges` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `zoomEvents` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `placements` | . | . | . | . | . | . | . | . | . | . | hausse | . | . |
| `placements.stats` | . | . | . | . | . | . | . | . | . | . | contenu | . | . |
| `pads` | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu |
| `vehicles` | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu | contenu |
| `carrierMarks` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `zoneReads` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `bombReads` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `grenades` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `projectiles` | . | . | . | . | . | . | . | . | . | . | BAISSE | . | . |
| `deaths` | . | . | . | . | . | contenu | contenu | . | . | . | . | . | . |
| `playerIndices` | . | . | . | . | . | . | . | . | . | . | . | contenu | . |
| `clockOrigin` | . | . | . | . | . | . | . | . | . | . | . | . | . |
| `artifact` | hausse | hausse | hausse | BAISSE | hausse | hausse | BAISSE | hausse | hausse | hausse | hausse | BAISSE | BAISSE |

Synthèse : **540 couples identiques, 110 différents. 34 étapes sur 50 ne bougent sur aucun
film** — dont les couches les plus volumineuses du document : `positions`, `fire`,
`grenades`, `inventory`, `pickups`, `loadouts`, `equipmentChanges`, `zones`, `spawnPoints`,
`vip`, `bomb`, `zoneReads`, `carrierMarks`, `clockOrigin`. Les 16 étapes qui bougent sont
celles qu'énumèrent les §2 et §3.

## 2. Les baisses, une par une, et l'entrée qui les explique

### 2.1 `objectives` vers 0 sur quatre films — GARDE D'EFFECTIF, voulue

| Film | Ancien | Nouveau | Sièges au coup d'envoi | Slots du statborg |
|---|---|---|---|---|
| `084a804d` | 210 | 0 | 26 | 8 |
| `1c4c63c2` | 26 343 | 0 | 24 | 8 |
| `51101d1d` | 49 | 0 | 10 | 8 |
| `9f57c612` | 2 | 0 | 10 | 8 |

Entrée : commit `ebd012e3b` — « fix(6.7-B1) : GARDE D'EFFECTIF du calque des actions
d'objectif (item 5) ». Phrase citée :

> « Le statborg ne connait que HUIT slots d'entite de joueur (`statSlotMin` a `statSlotMax`,
> pairs). C'est une contrainte de FORMAT : au-dela, les compteurs lus sur ces huit slots ne
> correspondent a personne de facon reproductible. [...] `replaybuild.identifiedEvents` refuse
> le film ENTIER — rien dans le film ne dit LESQUELLES des huit series correspondent a un
> joueur reel — compte le refus (`coverage.objectives.refusedByRoster`) et emet un
> `slog.WarnContext`. Un silence documente vaut mieux qu'un calque faux. »

Code : `internal/replaybuild/matchfacts.go:336`, garde
`objectiveevents.RosterFitsStatborg` (`internal/analysis/objectiveevents/rosterfit.go:72`,
`n <= StatPlayerSlots`).

Preuve à l'exécution — la passe `-update` a émis le refus pour ces quatre films, et pour eux
seuls parmi les treize anciens :

```
WARN replaybuild: actions d'objectif REFUSEES — effectif hors du format du statborg
  match_id=9f57c612-...  nommees=4    sieges=10  lignes=12  slots=8
  match_id=51101d1d-...  nommees=71   sieges=10  lignes=13  slots=8
  match_id=084a804d-...  nommees=216  sieges=26  lignes=26  slots=8
  match_id=1c4c63c2-...  nommees=242  sieges=24  lignes=24  slots=8
```

Le commit chiffre lui-même le gain de justesse : sur `4f77afc1` le calque publiait « 65 prises
de drapeau » là où « l'oracle API en compte QUATRE pour les 36 lignes de sa feuille ». La
baisse est le RETRAIT d'un calque faux, pas la perte d'un calque juste.

### 2.2 `projectiles` moins 11 sur `60ae07c4` — PORTE D'i0 SUR LIVE FIRE, voulue

`60ae07c4` est le seul film Live Fire du corpus, et `CORPUS.txt` le désigne depuis le
2026-09-02 comme « carte a plus de 2 regions — ECART ATTENDU ».

Entrée : chronique **v53** (2026-09-12, lot B-bis), `document_chronicle.go:1121` :

> « La PORTE d'`object-position-component` (i0 des objets du monde : projectiles ti=41,
> équipement ti=37, armes au sol ti=42) était écrite EN DUR à 3 bits [...] La largeur de cet
> index est une CONSTANTE PAR CARTE (`regionIndexBits` du catalogue de bornes). Elle vaut 1
> sur 78 cartes du catalogue et DEUX sur la 79e, Live Fire (`sgh_interlock`, quatre régions
> déclarées, arène en région 1). »

> « Un lancer se perd sur `0797ce72` (103 -> 102) — le garde d'auteur de v52 refuse une
> fenêtre ambiguë, comme il le faisait déjà sur Cliffhanger. »

Le même correctif explique sur le même film la HAUSSE de `placements` (57 vers 190) :
« les POSES D'ÉQUIPEMENT passent de 49 à 227 sur `0797ce72` ». Le solde du film est
largement positif (artefact plus 31 260 octets).

### 2.3 `artifact` en baisse sur quatre films — GRANDEUR DÉRIVÉE

`artifact` est la dernière étape : son « compte » est la longueur en octets du document cuit,
pas un nombre d'entités.

| Film | Ancien | Nouveau | Delta | Part | Couche ayant perdu des entrées |
|---|---|---|---|---|---|
| `1c4c63c2` | 8 301 577 | 6 864 012 | −1 437 565 | −17,3 % | `objectives` (§2.1) |
| `51101d1d` | 650 627 | 628 208 | −22 419 | −3,4 % | `objectives` (§2.1) |
| `7344d24f` | 2 354 586 | 2 352 860 | −1 726 | −0,07 % | **aucune** |
| `a349fea8` | 8 709 326 | 8 702 481 | −6 845 | −0,08 % | **aucune** |

Les deux premiers sont la conséquence arithmétique du retrait des actions d'objectif.

Pour `7344d24f` et `a349fea8`, **aucune des 49 autres étapes n'est en baisse** (cf. §1 : leurs
seules cellules non-`.` sont `contenu` ou `NEW`). Aucune couche publiée n'a donc perdu
d'entrée ; la variation de 0,07-0,08 % porte sur le CONTENU des entrées (`score`, `skull`,
`flag`, `pads`, `vehicles`, `killsource` ont un sha différent à compte égal), c'est-à-dire les
corrections d'identité et de manche v44 à v50. Ce n'est pas une perte.

## 3. Les hausses et les changements de contenu, rattachés

| Étape | Films | Entrée qui l'explique |
|---|---|---|
| `score` | 13/13, compte égal | Domaine des compteurs au niveau de l'enregistrement, `6066e462c` (6.11 item 3) ; bornes de manche `round_bounds.go` (lot 6.R) ; `statborg.go` a bougé de 82 lignes depuis `179bd7401`. Forme de `ScoreInput` ET de `StatRecord` INCHANGÉE (vérifié champ par champ) : c'est bien le contenu qui change, pas la structure |
| `flag` | 13/13, compte égal | `FlagInput` gagne `TeamOf`, `Spawns`, `Neutral`, `RadiusM`, `ResetSeconds`, `SoloSeconds` — chronique v42 (« le drapeau a enfin des porteurs ») et v46 (« un joueur ne porte plus son propre drapeau ») |
| `skull`, `pads`, `vehicles` | 13/13, compte égal | v41 (« trois calques rattrapent une track = une vie »), v47, v52 |
| `bipedCreations` | 13/13, ÉTAPE NEUVE | Les anciennes références portaient 49 étapes, les neuves 50 : étape ajoutée depuis le figeage. Addition, pas écart |
| `deaths`, `killRefs` | `084a804d` et `1c4c63c2` — les DEUX films v39 du corpus, comptes inchangés (356, 495) | Chronique **v54** : « le bloc d'event de 60 octets du chunk HIGHLIGHT porte le gamertag a `b[0:32]` sur les FilmMajorVersion <= 38 et >= 41, et a `b[12:44]` sur les versions 39-40. `ScanDeaths` passait 0 en dur au parseur ». **Prédiction vérifiée** : l'entrée désigne les versions 39-40, et l'écart tombe sur ces deux films-là et eux seuls |
| `killsource` | `084a804d`, `1c4c63c2`, `a349fea8` | v54 (même cause) ; `killsource/decode.go` modifié depuis le figeage |
| `neutralDeaths` | `084a804d` 0 vers 2 | v54 |
| `objectives` en hausse | `64e8adfa` 250 vers 340, `53ce4390` 200 vers 226 | v44 (manche déclarée confrontée au temps), v47 et v48 (pont d'identité), `20f138e30` (complétion d'identité par résidu de manche) |
| `placements`, `placements.stats` | `60ae07c4` | v53 (cf. §2.2) |
| `playerIndices` | `51101d1d`, compte égal | v50 (registre d'identité des joueurs) |
| `objectives` sha seul | `a349fea8` 0 vers 0 | Le calque rend `nil` là où il rendait une tranche vide : représentation, aucun contenu |
| `artifact` en hausse | 9 films | Somme des gains ci-dessus |

## 4. Constats de régression à instruire

**AUCUN.**

Justification par le tableau de la §1 : les seules cellules `BAISSE` y sont `objectives`
(4 films, §2.1), `projectiles` (1 film, §2.2) et `artifact` (4 films, §2.3, grandeur dérivée
dont deux occurrences sont la conséquence arithmétique de §2.1 et deux ne s'accompagnent
d'aucune baisse de couche). Chacune est rattachée à un commit ou à une entrée de chronique
nommée ; deux le sont par une preuve indépendante — une trace d'exécution pour §2.1, une
prédiction vérifiée sur les films exacts que l'entrée désigne pour v54.

Ce que ce rapport NE prouve PAS, et qu'il n'a pas mandat de prouver : que chacune des vingt
montées de schéma était juste. Il prouve que tout écart mesuré entre le schéma 34 et le
schéma 54 est rattaché à une décision écrite, et qu'aucun écart n'est orphelin.

## 5. Rejouer

Depuis `apps/go-api` du worktree, jonctions `data/cache/film_chunks` et
`data/cache/film_manifests` en place vers le checkout principal, `GOCACHE` privé :

```bash
# 1. relire une ancienne reference (lot 4b, schema 34)
git show cbfdc269d:apps/go-api/internal/games/halo_infinite/film/replay/testdata/equivalence/51101d1d.tsv

# 2. re-figer (fait le 2026-09-13 ; sous-ensembles pour tenir sous 10 min par commande)
go run ./cmd/replay-equiv -repo-root <worktree> -films 000d5950,01e1f945,64e8adfa,7344d24f,696a9d7c -update
go run ./cmd/replay-equiv -repo-root <worktree> -films 53ce4390,d9781168,9f57c612,60ae07c4,51101d1d -update
go run ./cmd/replay-equiv -repo-root <worktree> -films 084a804d -update
go run ./cmd/replay-equiv -repo-root <worktree> -films 1c4c63c2 -update
go run ./cmd/replay-equiv -repo-root <worktree> -films a349fea8 -update
go run ./cmd/replay-equiv -repo-root <worktree> -films bcb6d393,a521164d,111fa685 -update
go run ./cmd/replay-equiv -repo-root <worktree> -films 11de8353,e5adf7b2,fb1a1a72,50247b26 -update

# 3. verifier le determinisme (aucune reference ecrite)
go run ./cmd/replay-equiv -repo-root <worktree> -films <les memes sous-ensembles>
```

La classification elle-même ne décode rien : elle compare les colonnes `compte` et `sha` des
`.tsv` anciens (extraits de `cbfdc269d`) et neufs, étape par étape.

## 6. Détail des 110 couples différents

| Film | Étape | Ancien | Nouveau | Delta | Classe |
|---|---|---|---|---|---|
| `000d5950` | `score` | 1 | 1 | 0 | contenu |
| `000d5950` | `skull` | 1 | 1 | 0 | contenu |
| `000d5950` | `flag` | 1 | 1 | 0 | contenu |
| `000d5950` | `bipedCreations` | (absente) | 99 | +NEW | etape neuve |
| `000d5950` | `pads` | 1 | 1 | 0 | contenu |
| `000d5950` | `vehicles` | 1 | 1 | 0 | contenu |
| `000d5950` | `artifact` | 2501122 | 2523402 | 22280 | hausse |
| `01e1f945` | `score` | 1 | 1 | 0 | contenu |
| `01e1f945` | `skull` | 1 | 1 | 0 | contenu |
| `01e1f945` | `flag` | 1 | 1 | 0 | contenu |
| `01e1f945` | `bipedCreations` | (absente) | 113 | +NEW | etape neuve |
| `01e1f945` | `pads` | 1 | 1 | 0 | contenu |
| `01e1f945` | `vehicles` | 1 | 1 | 0 | contenu |
| `01e1f945` | `artifact` | 1940276 | 1948773 | 8497 | hausse |
| `64e8adfa` | `score` | 1 | 1 | 0 | contenu |
| `64e8adfa` | `objectives` | 250 | 340 | 90 | hausse |
| `64e8adfa` | `skull` | 1 | 1 | 0 | contenu |
| `64e8adfa` | `flag` | 1 | 1 | 0 | contenu |
| `64e8adfa` | `bipedCreations` | (absente) | 138 | +NEW | etape neuve |
| `64e8adfa` | `pads` | 1 | 1 | 0 | contenu |
| `64e8adfa` | `vehicles` | 1 | 1 | 0 | contenu |
| `64e8adfa` | `artifact` | 3082162 | 3118191 | 36029 | hausse |
| `7344d24f` | `score` | 1 | 1 | 0 | contenu |
| `7344d24f` | `skull` | 1 | 1 | 0 | contenu |
| `7344d24f` | `flag` | 1 | 1 | 0 | contenu |
| `7344d24f` | `bipedCreations` | (absente) | 122 | +NEW | etape neuve |
| `7344d24f` | `pads` | 1 | 1 | 0 | contenu |
| `7344d24f` | `vehicles` | 1 | 1 | 0 | contenu |
| `7344d24f` | `artifact` | 2354586 | 2352860 | -1726 | **BAISSE** |
| `696a9d7c` | `score` | 1 | 1 | 0 | contenu |
| `696a9d7c` | `skull` | 1 | 1 | 0 | contenu |
| `696a9d7c` | `flag` | 1 | 1 | 0 | contenu |
| `696a9d7c` | `bipedCreations` | (absente) | 110 | +NEW | etape neuve |
| `696a9d7c` | `pads` | 1 | 1 | 0 | contenu |
| `696a9d7c` | `vehicles` | 1 | 1 | 0 | contenu |
| `696a9d7c` | `artifact` | 2220334 | 2223087 | 2753 | hausse |
| `084a804d` | `score` | 1 | 1 | 0 | contenu |
| `084a804d` | `objectives` | 210 | 0 | -210 | **BAISSE** |
| `084a804d` | `skull` | 1 | 1 | 0 | contenu |
| `084a804d` | `flag` | 1 | 1 | 0 | contenu |
| `084a804d` | `killsource` | 1 | 1 | 0 | contenu |
| `084a804d` | `neutralDeaths` | 0 | 2 | 2 | hausse |
| `084a804d` | `killRefs` | 1 | 1 | 0 | contenu |
| `084a804d` | `bipedCreations` | (absente) | 379 | +NEW | etape neuve |
| `084a804d` | `pads` | 1 | 1 | 0 | contenu |
| `084a804d` | `vehicles` | 1 | 1 | 0 | contenu |
| `084a804d` | `deaths` | 356 | 356 | 0 | contenu |
| `084a804d` | `artifact` | 9256015 | 9330766 | 74751 | hausse |
| `1c4c63c2` | `score` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `objectives` | 26343 | 0 | -26343 | **BAISSE** |
| `1c4c63c2` | `skull` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `flag` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `killsource` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `killRefs` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `bipedCreations` | (absente) | 586 | +NEW | etape neuve |
| `1c4c63c2` | `pads` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `vehicles` | 1 | 1 | 0 | contenu |
| `1c4c63c2` | `deaths` | 495 | 495 | 0 | contenu |
| `1c4c63c2` | `artifact` | 8301577 | 6864012 | -1437565 | **BAISSE** |
| `53ce4390` | `score` | 1 | 1 | 0 | contenu |
| `53ce4390` | `objectives` | 200 | 226 | 26 | hausse |
| `53ce4390` | `skull` | 1 | 1 | 0 | contenu |
| `53ce4390` | `flag` | 1 | 1 | 0 | contenu |
| `53ce4390` | `bipedCreations` | (absente) | 122 | +NEW | etape neuve |
| `53ce4390` | `pads` | 1 | 1 | 0 | contenu |
| `53ce4390` | `vehicles` | 1 | 1 | 0 | contenu |
| `53ce4390` | `artifact` | 2998023 | 3028577 | 30554 | hausse |
| `d9781168` | `score` | 1 | 1 | 0 | contenu |
| `d9781168` | `skull` | 1 | 1 | 0 | contenu |
| `d9781168` | `flag` | 1 | 1 | 0 | contenu |
| `d9781168` | `bipedCreations` | (absente) | 160 | +NEW | etape neuve |
| `d9781168` | `pads` | 1 | 1 | 0 | contenu |
| `d9781168` | `vehicles` | 1 | 1 | 0 | contenu |
| `d9781168` | `artifact` | 2586160 | 2641104 | 54944 | hausse |
| `9f57c612` | `score` | 1 | 1 | 0 | contenu |
| `9f57c612` | `objectives` | 2 | 0 | -2 | **BAISSE** |
| `9f57c612` | `skull` | 1 | 1 | 0 | contenu |
| `9f57c612` | `flag` | 1 | 1 | 0 | contenu |
| `9f57c612` | `bipedCreations` | (absente) | 93 | +NEW | etape neuve |
| `9f57c612` | `pads` | 1 | 1 | 0 | contenu |
| `9f57c612` | `vehicles` | 1 | 1 | 0 | contenu |
| `9f57c612` | `artifact` | 1584146 | 1600852 | 16706 | hausse |
| `60ae07c4` | `score` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `skull` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `flag` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `bipedCreations` | (absente) | 171 | +NEW | etape neuve |
| `60ae07c4` | `placements` | 57 | 190 | 133 | hausse |
| `60ae07c4` | `placements.stats` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `pads` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `vehicles` | 1 | 1 | 0 | contenu |
| `60ae07c4` | `projectiles` | 658 | 647 | -11 | **BAISSE** |
| `60ae07c4` | `artifact` | 3036129 | 3067389 | 31260 | hausse |
| `51101d1d` | `score` | 1 | 1 | 0 | contenu |
| `51101d1d` | `objectives` | 49 | 0 | -49 | **BAISSE** |
| `51101d1d` | `skull` | 1 | 1 | 0 | contenu |
| `51101d1d` | `flag` | 1 | 1 | 0 | contenu |
| `51101d1d` | `bipedCreations` | (absente) | 36 | +NEW | etape neuve |
| `51101d1d` | `pads` | 1 | 1 | 0 | contenu |
| `51101d1d` | `vehicles` | 1 | 1 | 0 | contenu |
| `51101d1d` | `playerIndices` | 1 | 1 | 0 | contenu |
| `51101d1d` | `artifact` | 650627 | 628208 | -22419 | **BAISSE** |
| `a349fea8` | `score` | 1 | 1 | 0 | contenu |
| `a349fea8` | `objectives` | 0 | 0 | 0 | contenu |
| `a349fea8` | `skull` | 1 | 1 | 0 | contenu |
| `a349fea8` | `flag` | 1 | 1 | 0 | contenu |
| `a349fea8` | `killsource` | 1 | 1 | 0 | contenu |
| `a349fea8` | `bipedCreations` | (absente) | 316 | +NEW | etape neuve |
| `a349fea8` | `pads` | 1 | 1 | 0 | contenu |
| `a349fea8` | `vehicles` | 1 | 1 | 0 | contenu |
| `a349fea8` | `artifact` | 8709326 | 8702481 | -6845 | **BAISSE** |
