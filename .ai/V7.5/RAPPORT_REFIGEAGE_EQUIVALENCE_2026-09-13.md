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

> **MIS À JOUR LE 2026-09-13 APRÈS LE CROISEMENT PAR LE CORPUS GATE (§7).** Le verdict initial
> de cette section — « zéro constat orphelin » — valait pour l'oracle d'ÉQUIVALENCE, qui compare
> les sorties de balayage. Le corpus gate à polarité compare les AXES DU DOCUMENT PUBLIÉ, et il
> voit ce que l'équivalence ne pouvait pas voir. **Le verdict global devient : TROIS CONSTATS À
> INSTRUIRE** (§7.C) — `coverage.score.rounds 3 -> 1` sur `fb1a1a72`, l'effondrement du bloc
> monde/équipement sur `60ae07c4`, et des points de piste publiés en baisse sur trois témoins.
> Aucun n'est corrigé ici ; les trois sont recopiés en §4 du plan.
>
> Ce que le croisement NE remet PAS en cause : les 110 écarts de l'équivalence restent tous
> rattachés (§1 à §3), et 8 des 11 familles de pertes du gate sont expliquées (§7.A) ou sont des
> artefacts de polarité (§7.B). Les références re-figées restent utilisables comme ligne de base ;
> les trois constats portent sur le CONTENU publié, pas sur le figeage.

### 0.1 Verdict de l'oracle d'équivalence (inchangé)

**Aucun constat de régression sur cet oracle.** Les 110 écarts mesurés (sur 650 couples
film × étape) se rattachent tous à une entrée documentée. Les seules BAISSES de compte sont :

| Baisse | Films | Entrée qui l'explique |
|---|---|---|
| `objectives` → 0 | `084a804d`, `1c4c63c2`, `51101d1d`, `9f57c612` | Garde d'effectif, commit `ebd012e3b` (lot 6.7-B1 item 5) |
| `projectiles` −11 | `60ae07c4` (Live Fire) | Chronique **v53** (porte d'i0, `regionIndexBits` = 2 sur Live Fire) |
| `artifact` (octets) | `1c4c63c2`, `51101d1d`, `7344d24f`, `a349fea8` | Grandeur DÉRIVÉE (longueur du document), conséquence des deux lignes ci-dessus et des changements de contenu amont — aucune couche n'a perdu d'entrée sur ces films |

Preuve de complétude : le tableau de la §1 porte les 50 étapes × 13 films ; toute cellule
n'est ni `.` (identique) ni `hausse`/`contenu`/`NEW` que dans les trois cas ci-dessus.

### 0.2 Verdict du corpus gate (§7)

| Table | Contenu | Compte |
|---|---|---|
| §7.A | Divergences expliquées, entrée citée avec sa phrase | 6 familles |
| §7.B | Polarité douteuse (compteur d'échec hors liste fermée, axe neuf, ligne déjà acceptée au registre) | 8 familles |
| §7.C | **Constats de régression à instruire** | **3** |

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

## 7. Croisement par le corpus gate (schéma 34 -> 54, 5 témoins)

> Ajouté le 2026-09-13. Le corpus gate à polarité (`--base=179bd7401`) compare les AXES DU
> DOCUMENT PUBLIÉ (`replaydiff`), là où la §1 comparait les sorties de balayage. Il voit donc
> des pertes que l'équivalence ne pouvait pas voir. Bilan brut : gains 133 / 247 / 114 / 445 /
> 165 ; pertes 8 / 18 / 22 / 90 / 61.
>
> Avertissement de lecture, tiré de `internal/replaydiff/polarite.go` : la liste des compteurs
> d'ÉCHEC dont la baisse est un GAIN y est **fermée et nommée** (`unpublished`, `unnamedLives`,
> `noSlot`, `noTrack`, `outOfWindow`, `ambiguous`, `closedRefused`, `closedContested`,
> `indexDisagreements`, `slotCollisions`, `noBridge`, `ambiguousReturns`, `ambiguousSlot`,
> `shotsNoRide`, `periodsNoBridge`). « un compteur absent d'ici garde la lecture générique
> (plus = mieux) » — c'est la source de la moitié des « pertes » ci-dessous.

### 7.A — Divergences expliquées

| # | Ligne | Témoins | Entrée, avec la phrase |
|---|---|---|---|
| A1 | `geometry.*` et `geometryBounds/n` DISPARUS (382 -> —, yaw 353 -> —, bounds 4 -> —) | fb1a1a72, d9781168, 084a804d, 60ae07c4 — **pas bcb6d393** | Chronique **v52** point 4 (`document_chronicle.go:1084`), commit `70edf37b6` « les props Forge appartiennent a une carte, plus au titre » : « `geometry` devient les props de LA CARTE du match. Un répertoire unique les servait à tous les matchs : les artefacts d'une carte non extraite sortent désormais SANS props, **ce qui est la vérité**. » La même entrée mesure « PROPS : **382 props identiques sur les 76 artefacts, cartes confondues** » — 382 est exactement le chiffre du gate. Le seul témoin qui GARDE ses props est bcb6d393 (Cliffhanger) : « une seule extraction existe ». `geometryBounds` est dérivé (`build.go:39`, `geometryBounds(opt.Geometry)`), il tombe avec lui |
| A2 | `bounds.maxX 202.97 -> 43.03`, `maxY 152.73 -> 37.67`, `maxZ 205.81 -> 115.27` | 084a804d | **Changement de définition, pas perte.** À 34, `boundsOf` rendait les bornes BRUTES de tous les points (`git show 179bd7401:...geometry.go:161`, aucune garde, aucun compte d'écarté). Commit `490dc595e` « les bornes ignorent les echantillons aberrants » : « Un point sur 16 064 (z=-325 m sous un sol joue a 117 m) fixait a lui seul MinX, MaxY et MinZ de l'artefact 81c02726 [...] la scene se cadrait sur un point fantome. » Rejet par centiles p1..p99, seuil 12 étendues. Le catalogue confirme l'ordre de grandeur : `fortitude` est quantifiée sur X ∈ [−231,00 ; +231,64] (`map_quant_bounds.json`) — 202,97 était l'ENVELOPPE DE QUANTIFICATION atteinte par un artefact de décodage, pas l'aire jouée. Le commit le dit : « les artefacts deja cuits gardent leurs bornes fausses et doivent etre recuits » |
| A3 | `coverage.objectives.attached 187 -> 0` et les 44 axes `objectives.*` DISPARUS | 084a804d | **Garde d'effectif**, déjà classée en §2.1 : commit `ebd012e3b`, `matchfacts.go:336`, 26 sièges pour 8 slots, refus tracé à l'exécution (`nommees=216 sieges=26 slots=8`). Le gate voit ici la face DOCUMENT de ce que la §2.1 voyait au balayage |
| A4 | `joueur/2533274823110022/assists 69 -> 11` | d9781168 | **CORRECTION prouvée par la feuille de match.** `d9781168.facts.json` : `{"xuid":"2533274823110022","kills":18,"deaths":19,"assists":11}` — **la valeur 54 EST la feuille, à l'unité**. 69 assistances dans un Oddball était le déroulage non borné. Commit `f22474816` « BORNE DE DEROULAGE recalibree sur l'oracle, 100 000 -> 16 » : « 40 806 pulses pour 106 assistances reelles. On borne donc, et l'oracle dit de combien. » |
| A5 | `coverage.flagCarries.steals 78 -> 14` et `openings 46 -> 35` | fb1a1a72 | Même cause que A4 (`f22474816`). `Steals` n'est pas une richesse : c'est l'un des « trois signaux qui ont fondé ce verdict, publiés pour qu'il se vérifie » (`document_objectives_live.go:196`). `Openings` est « le nombre de PRISES de l'oracle (`flag_grabs` + `flag_steals`) une fois les émissions JUMELLES FUSIONNÉES » — un dénominateur d'oracle, que la borne et la fusion réduisent ensemble. 78 vols dans un CTF n'est pas un chiffre de match |
| A6 | `projectiles.p/n`, `projectiles/n`, `projectiles.t0/presents`, `projectiles.rest/presents` en baisse sur les 5 témoins | tous | Chronique **v52** point 3 : « UN VOL DE PROJECTILE S'ARRÊTE AU PREMIER PAS IMPOSSIBLE (> 10 m en 100 ms, `projectileMaxStepM`), et n'est pas recousu. `rest` tombe à false sur un vol coupé : il CERTIFIE une fin de vol. » Mesure de l'entrée : « 947 trajectoires sur 15 735 du parc (6,0 %) portaient au moins un pas impossible » ; golden `000d5950` « 439 -> 436 trajectoires, 2 732 -> 2 725 points ». Par témoin : bcb6d393 −20 pts (−4,9 %) · fb1a1a72 −277 pts / −25 vols (−9,3 %) · d9781168 −14 / −3 (−0,6 %) · 084a804d −107 / −25, `rest` 60 -> 52 (−1,1 %) · 60ae07c4 −14 / −4 (−0,3 %). La forme (vols ↓, points ↓, `rest` ↓) est celle que l'entrée décrit |

### 7.B — Polarité douteuse : la « perte » n'en est pas une

Trois motifs, tous vérifiés sur pièces : **(i)** compteur d'ÉCHEC absent de la liste fermée de
`polarite.go` — il garde la lecture générique « plus = mieux », donc sa baisse sort en perte
alors qu'elle est le gain cherché ; **(ii)** AXE NEUF — le compteur n'existait pas au schéma 34,
son apparition est un artefact de construction ; **(iii)** compteur de VOIE déjà inscrit au
registre des reports comme « ligne à ACCEPTER au gate ».

| # | Ligne | Témoins | Motif |
|---|---|---|---|
| B1 | `coverage.bridge.unnamedLives — -> 2` | bcb6d393 | **(ii) axe neuf** : `json:"unnamedLives"` est ABSENT à `179bd7401` (vérifié par `git grep`). Il naît avec v47 (« AUCUNE VIE PUBLIEE NE RESTE SANS NOM »). Il est de surcroît dans la liste fermée, donc son APPARITION est inversée en perte par `inverserSens(SensApparu)` — double artefact. Deux vies sans nom sur ce film sont une MESURE neuve, pas une perte |
| B2 | `coverage.vehicles.shotsNoRide — -> 2265` et `coverage.vehicles.ambiguous — -> 10` | 084a804d | **(ii) axe neuf** : `shotsNoRide` ABSENT à `179bd7401` (vérifié) ; tout le bloc `coverage.vehicles` naît au schéma **39** (chronique v39, « LES VÉHICULES »), donc après 34. Les deux sont dans la liste fermée : leur apparition est inversée en perte. Le plan les cite d'ailleurs comme des échecs qui BAISSENT (`shotsNoRide` 2768 -> 2265 au lot E2-bis) |
| B3 | `coverage.equipmentChanges.counterJumps` (4->2, 4->3, 11->7, 1->0), `missedEstimate` (4->2, 6->3, 17->8, 6->0), `livesFirstOffSpec` (1->0), `repeats` (2->0) | bcb6d393, fb1a1a72, 084a804d, 60ae07c4 | **(i)** Compteurs d'échec par construction — le type les décrit comme « ce qu'il a écarté, et — seul de tous les calques du rejeu — ce qu'il a MANQUÉ » (`document_equipment_changes.go:75`). `missedEstimate` est une ESTIMATION DE MANQUE : la voir baisser de 17 à 8 est le gain. Aucun n'est dans la liste fermée de `polarite.go` |
| B4 | `coverage.groundWeapons.rejected 211 -> 116`, `coverage.groundWeapons.unknown 36 -> 5` | 60ae07c4 | **(i)** `rejected` et `unknown` sont des échecs (`GroundWeaponCoverage`, `coverage_world.go:28`), hors liste fermée |
| B5 | `coverage.placements.unknown 57 -> 4` et les 5 `coverage.placements.byFamilyOrigin.<famille>/unknown` (dont `grenade_frag` 42 -> 4, et 4 DISPARUS) | 60ae07c4 | **(i)** « origine inconnue » est l'échec du calque des poses. Le mouvement est le GAIN mesuré par la chronique **v53** sur l'autre film Live Fire : « les POSES D'ÉQUIPEMENT passent de 49 à 227 sur `0797ce72` ». L'équivalence le confirme de son côté : `placements` 57 -> **190** en HAUSSE (§3) |
| B6 | `coverage.pickups.originUnknown 56 -> 40` | 60ae07c4 | **(i)** échec (`PickupCoverage`, « ce qu'il ne PEUT PAS voir »), hors liste fermée |
| B7 | `coverage.skullCarries.carrierAbsent 6 -> 0` et `2 -> 0` | d9781168, 60ae07c4 | **(i)** « porteur absent » est un échec d'attribution ; tomber à ZÉRO sur les deux témoins Oddball est le gain du pont d'identité (v42, v50). Hors liste fermée |
| B8 | `coverage.flagCarries.markerConfirmed` (6->5, 1->0) et `markerObserved` (6->5, 1->0) | bcb6d393, 084a804d | **(iii)** DÉJÀ INSCRIT au registre : `.ai/V7.5/REGISTRE_REPORTS.md` ligne 20 — « C'est une VOIE de retour au camp, pas une richesse [...] Generaliser la regle de polarite aux voies de `flagCarries` demande leur propre inventaire — hors perimetre du lot (regle 7) [...] **En attendant : ligne a ACCEPTER au gate** ». L'entrée nomme explicitement `markerConfirmed`/`markerObserved` parmi les compteurs de voie à faire entrer dans `prefixesMethode` |

### 7.C — Constats de régression à instruire

Trois constats. Aucun n'est démenti par une entrée ; aucun n'est expliqué par une entrée qui
le quantifie. Ils ne sont PAS corrigés ici (règle 7).

| # | Constat | Témoin | Ce qui est établi, et ce qui manque |
|---|---|---|---|
| **C1** | `coverage.score.rounds 3 -> 1` sur un CTF que le registre tient pour MULTI-MANCHE | fb1a1a72 | **Mécanisme établi.** `RealRounds` = `contiguousRounds(runs, materialRounds, presentRounds)` (`statborg.go:482`). `materialRounds` **IGNORE LES SLOTS D'ÉQUIPE** (`statborg.go:557`, `if IsTeamSlot(r.Slot) { continue }`). Or le registre dit de CE film, mesure du 2026-09-08 : « sur ce film les enregistrements de slot JOUEUR declarent TOUS la manche 0 [...] tandis que **les manches viennent des slots d EQUIPE** » (`REGISTRE_REPORTS.md` ligne 592). Les manches de ce film sont donc invisibles à `materialRounds` PAR CONSTRUCTION, et leur admission ne tient plus qu'à `runs[round] >= statMinRoundRun`, que le filtre de domaine du lot 6.11 (`6066e462c`) a resserré. Le garde de manche fantôme (`bb06cce5a`) ajoute la rupture `(vue && !present[round])`. **Deux lectures opposées, non tranchées** : (a) CORRECTION — les manches 1 et 2 sont des fantômes de la même famille que la manche 2 de `e60aaf06`, et deux oracles vont dans ce sens : la feuille donne `teamScores [0,1]` (une seule manche gagnée au total) et `regulation.toml [rounds_decide]` ne liste QUE les Oddball, donc CTF n'est pas un mode décidé aux manches ; (b) RÉGRESSION — le film est bien multi-manche (le registre l'appelle « CTF MULTI-MANCHE » depuis le 2026-09-06) et le document publie désormais UNE manche là où il en publiait trois. **Ce qui manque pour trancher** : relire, sur une cuisson de `fb1a1a72`, `RealRounds` slot par slot — combien de manches déclarent les slots d'ÉQUIPE, et `runs` passe-t-il `statMinRoundRun` sur les manches 1 et 2. Aucun décodage n'a été fait ici. À rapprocher du volet STATBORG de la ligne 592, **déclaré TOUJOURS OUVERT sur ce film exact** |
| **C2** | Le bloc MONDE / ÉQUIPEMENT s'effondre sur le seul témoin Live Fire : `groundWeapons.spawned 218 -> 35`, `accepted 429 -> 334`, `powerupAccepted 504 -> 399`, `skullCarries.grabs 39 -> 8`, `placements.lives 361 -> 327`, `equipmentChanges.decoded 38 -> 33` / `published 33 -> 29` / `lives 28 -> 26` / `spent 9 -> 6`, `groundWeaponItems.endSeen 183 -> 180`, `abilityLabels/n 4 -> 3`, `abilities/n 29 -> 27` | 60ae07c4 | **L'ordre de grandeur n'est PAS expliqué, et l'entrée la plus proche le CONTREDIT.** La chronique **v53** (porte d'i0, seul mécanisme documenté qui touche le décodage des objets du monde sur Live Fire) mesure explicitement, sur l'autre film Live Fire : « **Les armes au sol, les tirs et les ramassages ne bougent pas (217, 717, 108 des deux côtés)** : ils ne tiennent pas leur position de ce chemin. » Ici les armes au sol bougent beaucoup : `accepted` et `rejected` perdent **exactement 95 chacun** — 190 enregistrements quittent la classification —, et `spawned` perd 183. Le chiffre avancé de « 27 enregistrements sur 267 400 hors arène » ne couvre pas 190. Les trois commits fonctionnels du span sur ce calque (`838e9c7bb` ammo, `c49e787ed` pont aplati supprimé, `58da800a1` lecteurs au registre) ne le quantifient pas davantage. **Confondant à isoler** : 60ae07c4 est le seul témoin à la fois Live Fire (bit de région, v53) et Oddball (borne de déroulage, `f22474816` — voie plausible pour `skullCarries.grabs 39 -> 8`). **Ce qui manque** : cuire 60ae07c4 aux deux révisions encadrant v53 seule, et relever `coverage.groundWeapons` — si v53 explique 190, l'entrée v53 est fausse sur ce point et doit être amendée ; sinon la cause est ailleurs |
| **C3** | Points de piste publiés en baisse, et une capacité : `tracks.points/n` 36 581 -> 36 579 (−2), 111 956 -> 111 947 (−9), 45 137 -> 45 133 (−4) ; `tracks.points.hp/presents` 463 -> 461 ; `abilities/n` 166 -> 165 | d9781168, 084a804d, 60ae07c4 | **Richesse publiée, faible ampleur (0,005 à 0,008 %), aucune entrée qui la nomme.** Les points disparaissent en ENTIER (t, x, y, z baissent du même nombre). Ce n'est PAS la garde des bornes de A2 : `boundsOf` exclut des points du CALCUL des bornes, il n'en retire aucun du document (`geometry.go:233-248`, aucune mutation de `tracks`). Piste la plus plausible, non prouvée : la re-segmentation « une track = une vie » (v41, v43, v47) et le recalage du fil des morts (v48, `bestDeathOffset`) déplacent les frontières de vie, donc quelques points de bord à la décimation. **Ce qui manque** : identifier les 9 points de 084a804d (quelle piste, quel instant, bord de vie ou non). Sévérité faible, mais rien ne l'explique aujourd'hui |

### 7.D — Le web lit-il encore la géométrie ?

Oui, et il DÉGRADE PROPREMENT — aucun rendu ne casse :

- `apps/web/src/features/match-replay/ui/ReplayCanvas.tsx:481` :
  `drawGeometryLayer(ctx, doc.geometry ?? [], view, ...)` — le `?? []` absorbe l'absence ;
- même fichier `:442` : `floor: !!doc.geometry?.length` — la présence d'un sol est un booléen
  dérivé, pas une exigence ;
- `apps/web/src/lib/api/generated.ts:10234-10235` : `geometry?` et `geometryBounds?` sont
  déjà OPTIONNELS au contrat OpenAPI.

Conséquence produit, et elle est VOULUE : sur une carte sans extraction, le calque « fond de
carte (props Forge) » est désormais VIDE au lieu d'afficher les props D'UNE AUTRE CARTE. C'est
exactement ce que v52 appelle « ce qui est la vérité ». Un seul répertoire d'extraction existe
(« attribuée à `ridgeline` par son emprise »), donc 4 témoins sur 5 perdent le calque.
