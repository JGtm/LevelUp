# Lot E — les mesures de reference AVANT changement (item E.1)

> Mesures jouees le 2026-09-05 dans le worktree `LevelUp-wt-v2-decodeur`, branche
> `feat/v2-decodeur`, HEAD `a21fd77f4`, arbre PROPRE (aucune modification du lot).
> Prefixe commun de toute commande : depuis `apps/go-api`,
> `GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-decodeur CGO_ENABLED=1`, `-p 1 -parallel 1
> -count=1`. Films lus en LECTURE SEULE sous
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/`.
>
> CE DOCUMENT EST LE GATE DES ITEMS E.2 a E.5 : apres chaque item, ces memes commandes sont
> rejouees et confrontees ligne a ligne. Un chiffre qui bouge = le refacto n'est pas a
> comportement identique.

## 0. Ce que « l'equivalence locale » recouvre reellement (verification demandee)

Le registre parle de « 49 etapes d'equivalence en local ». VERIFIE : ces 49 etapes sont celles de
`cmd/replay-equiv` (`internal/analysis/replay/testdata/equivalence/<film>.tsv` : 1 ligne de
grammaire + **49 lignes d'etape** sur les 13 films figes). Ce harnais **cuit un artefact par
film** (`replaybuild.Builder.BuildBytes`, `cmd/replay-equiv/child.go:121-127`). Le mandat du lot
E l'interdit explicitement (« Aucune cuisson d'artefacts »), et la doctrine du depot exige de
DEMANDER avant toute cuisson. Il n'a donc PAS ete joue.

Ce qui a ete joue a sa place, et qui est ce qui existe REELLEMENT sous garde de variable
d'environnement : les goldens sur films reels de `killsource` (4 films, sortie figee ligne a
ligne), le temoin de marche delta (`DELTA_WITNESS_FILM`, 3 films figes), l'empreinte du registre,
le controle G2 de la table ECS (`ECS_TABLE_FILM`) et les tests d'integration de `killcollector`
(`KILLSOURCE_FIXTURES`). `REPLAY_FILM_DIR` n'est PAS une garde de verification : c'est la porte
de REGENERATION des goldens (`golden_inputs_test.go:1199-1208`, `minifilm_test.go:75-81`) — la
jouer reecrirait la reference. Elle est donc volontairement laissee vide.

## 1. Etage inconditionnel — aucune variable d'environnement

### 1.1 Le gate de paquets complet

```
go test ./internal/analysis/filmdec/... -p 1 -parallel 1 -count=1
  -> ok  levelup/go-api/internal/analysis/filmdec  21.720s

go test ./internal/games/halo_infinite/film/... ./internal/analysis/replay/... -p 1 -parallel 1 -count=1
  -> ok  levelup/go-api/internal/games/halo_infinite/film/damagetag   0.335s
  -> ok  levelup/go-api/internal/games/halo_infinite/film/filmcache   0.338s
  -> ok  levelup/go-api/internal/games/halo_infinite/film/killicon    0.384s
  -> ok  levelup/go-api/internal/games/halo_infinite/film/killsource  1.297s
  -> ok  levelup/go-api/internal/games/halo_infinite/film/medalname   0.255s
  -> ok  levelup/go-api/internal/analysis/replay                     12.841s
  -> ok  levelup/go-api/internal/analysis/replay/mapvar               0.395s

go test ./internal/sync/killcollector/... ./internal/archlint/... -p 1 -parallel 1 -count=1
  -> ok  levelup/go-api/internal/sync/killcollector   0.147s
  -> ok  levelup/go-api/internal/archlint            19.422s

go build ./...                     -> exit 0
golangci-lint run ./internal/analysis/filmdec/...   -> 6 issues (goconst 4, unparam 2)
```

CORRECTION DU 2026-09-06 — la premiere mesure de lint de ce fichier disait « 0 issues. » ET
ELLE ETAIT FAUSSE. `golangci-lint` tient un cache de RESULTATS **global a la machine**
(`%LocalAppData%\golangci-lint`, independant de `GOCACHE`), indexe par fichier ; il servait
donc des verdicts calcules dans un AUTRE jeu de fichiers du meme paquet. Rejoue avec
`GOLANGCI_LINT_CACHE` propre au lot, l'etat de base rend **6 issues** :
4 `goconst` (`traverse.go` : `biped-spartan-ability`, `biped-spartan-ability-component`,
`biped-desired-ability-set`, `biped-spartan-ability-energy`) et 2 `unparam`
(`components_biped_ability.go:314` `consume140c1e9d4 - w always receives 12`,
`components_movement.go:66` `consumeDynPrecVec3 - mag always receives 19`).
C'est de la dette ANTERIEURE, gelee par le ratchet CI (`--new-from-merge-base=origin/main`).
LECON, valable pour tous les lots : une mesure de lint de reference exige un
`GOLANGCI_LINT_CACHE` propre au lot, sans quoi elle ment.

### 1.2 Les goldens nommes, un par un (verdict PASS de chacun)

`go test ./internal/games/halo_infinite/film/killsource/ -run 'TestGoldenMiniBobine|TestGoldenPhrasesJustes' -v`

| test | verdict | duree |
|---|---|---|
| `TestGoldenPhrasesJustes` (+ 5 sous-tests) | PASS | 0.02 s |
| `TestGoldenMiniBobine` | PASS | 0.34 s |

`go test ./internal/analysis/replay/ -run 'TestEquivalenceMiniFilm|TestZeroDisque|TestGoldenInputs|TestGoldenAssembly|TestMiniFilmDecodes|TestShotsCoverage|TestBridgeNames|TestSeventyGrenade|TestProjectilesAndInventory|TestGoldenDocument|TestCatalogueDuTitre' -v`

| test | verdict | duree |
|---|---|---|
| `TestEquivalenceMiniFilm` | PASS | 2.33 s |
| `TestGoldenAssembly` | PASS | 0.88 s |
| `TestShotsCoverageIsFourEightyThreeOfFiveNineteen` | PASS | 0.77 s |
| `TestBridgeNamesNinetyLivesOfHundredFive` | PASS | 0.73 s |
| `TestSeventyGrenadeThrowsAreAllPlaced` | PASS | 0.91 s |
| `TestProjectilesAndInventoryCounts` | PASS | 0.90 s |
| `TestGoldenDocumentPublishesItsOwnUncertainty` | PASS | 0.75 s |
| `TestCatalogueDuTitreNommeLesArmesDuFilmDeReference` | PASS | 0.00 s |
| `TestGoldenAssemblyPhrasesJustes` / `PorteSesDenominateurs` / `FigeLesChiffresDuChantier` | PASS | 0.00 s |
| `TestGoldenInputsRoundTrip` | PASS | 1.12 s |
| `TestGoldenInputsVersionGuard` | PASS | 0.06 s |
| `TestGoldenInputsRegenerate` | SKIP (porte de regeneration, VOULU) | 0.00 s |
| `TestMiniFilmDecodesTheFireEvents` | PASS | 0.01 s |
| `TestMiniFilmDecodesTheGrenadeThrows` | PASS | 0.02 s |
| `TestMiniFilmDecodesProjectileFlights` | PASS | 0.71 s |
| `TestMiniFilmDecodesTheDeathThread` | PASS | 0.03 s |
| `TestMiniFilmDecodesTheKeyframes` | PASS | 1.39 s |
| `TestZeroDisqueBuildFromFilm` | PASS | 0.02 s |
| `TestZeroDisqueBalayagesSupportes` | PASS | 1.80 s |

### 1.3 Les 7 digests de la mini-bobine (`testdata/equivalence/minifilm.tsv`)

`TestEquivalenceMiniFilm` PASSE : les valeurs ci-dessous sont donc CELLES MESUREES par le
decodeur d'aujourd'hui, pas seulement le contenu d'un fichier.

```
# digest-grammar: 2
fire           519  f4923e8292847ad0c87e5e868bc68cdc22f273b57d20417be8a12d6220aebaa6
grenades        70  813221b5232887513a543e1dcf41de9ba2730aa4373cbf9dccff667905326511
loadouts       150  e5dadc04d5adb2d1f51441f9e20ef7224d5af564058e133eacb9a02d0f67a79e
inventory      184  a2b99e97ff46d483948a23640a4d2a1959ee08ddc1b30621d6eb13a4ef8df936
deaths          93  66ade0850df0fc0ecad199aee4fe9f3d6c511e489cb85c3204ab32ee4d1aa92e
playerIndices    1  b5cd498aee9520d1bac673f947ce773ddf6b05d447c4673265b22a1a38be4f2a
projectiles     22  f5c1780009d4d81369ebe809af0900be2ae7ec605d0caa1a97eabce3d1a2eb61
```

### 1.4 Le ratchet des variables de paquet

`archlint/filmdec_package_vars_test.go` : `filmdecVarsGeles = 118`, test VERT (le compte mesure
vaut donc 118 ou moins ; le test n'echoue qu'au-dessus).

## 2. Etage films reels — sous garde de variable d'environnement

`KILLSOURCE_FIXTURES=/c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

### 2.1 killsource, les 4 films de reference

`go test ./internal/games/halo_infinite/film/killsource/ -run 'TestGoldenFilms|TestReferenceFilms|TestLigneDiscriminanteEstServieParLaMarche' -v -p 1 -parallel 1 -count=1`

Verdict de reference : **`FAIL` — et cet echec est ANTERIEUR au lot** (arbre propre, aucune
modification). Il est reproduit tel quel ci-dessous ; le gate des items suivants est de le
retrouver A L'IDENTIQUE, ni plus ni moins.

| sous-test | verdict avant | duree |
|---|---|---|
| `TestLigneDiscriminanteEstServieParLaMarche` | PASS | 6.36 s |
| `TestGoldenFilms/000d5950` | PASS | 2.46 s |
| `TestGoldenFilms/9b191a7f` | PASS | 4.31 s |
| `TestGoldenFilms/78919882` | PASS | 8.61 s |
| `TestGoldenFilms/fccc61cd` | **FAIL** (1 ligne) | 4.22 s |
| `TestReferenceFilms/{000d5950,9b191a7f,78919882,fccc61cd}` | PASS | 15.64 s au total |

L'ECART, unique et stable — `testdata/fccc61cd.golden` ligne 53 :

```
fige   : marche, source appartenant a la victime : 3 propose(s), 2 publiee(s)
obtenu : marche, source appartenant a la victime : 2 propose(s), 2 publiee(s)
(74 lignes figees, 74 obtenues)
```

Le compte PUBLIE est identique (2) : aucune donnee servie ne change ; seule la ventilation des
CANDIDATS PROPOSES a bouge. `cumul.golden` n'est alors pas compare (le test refuse de figer un
etat casse).

Comptes publies par `TestReferenceFilms` (references chiffrees a retrouver a l'identique) :

| film | calibration | couverture | gate (b) | concordance |
|---|---|---|---|---|
| 000d5950 | axisW=13 indexW=2 [score 347, mediane 95] recordStateParam=4 [x1.001] | 93/93 reels, 93 reconstruits, 93 morts du feed, 0 de BOT | MARCHE 84/86 = 97.7 %, SCAN 8/10 = 80.0 %, marge 36 | redondants 87, ACCORD 84, DESACCORD 0, paquets localises 3423/3508 |
| 9b191a7f | axisW=16 indexW=3 [score 170, mediane 57] recordStateParam=4 [x1.158] | 84/84 reels, 85 reconstruits, 87 morts du feed, 3 de BOT | MARCHE 73/75 = 97.3 %, SCAN 9/11 = 81.8 %, marge 22 | redondants 83, ACCORD 73, DESACCORD 0, paquets localises 5525/5714 |
| 78919882 | axisW=15 indexW=3 [score 433, mediane 100] recordStateParam=2 [x1.000] | 99/99 reels, 99 reconstruits, 99 morts du feed, 0 de BOT | MARCHE 92/93 = 98.9 %, SCAN 5/8 = 62.5 %, marge 35 | redondants 94, ACCORD 92, DESACCORD 0, paquets localises 4816/5027 |
| fccc61cd | axisW=16 indexW=2 [score 1111, mediane 316] recordStateParam=4 [x1.188] | 95/95 reels, 95 reconstruits, 96 morts du feed, 2 de BOT | MARCHE 85/86 = 98.8 %, SCAN 7/8 = 87.5 %, marge 22 | redondants 90, ACCORD 85, DESACCORD 0, paquets localises 4686/4839 |

### 2.2 Temoin de marche delta et empreinte du registre — UN film par processus

`DELTA_WITNESS_FILM=<...>/film_chunks/<film> go test ./internal/analysis/filmdec/ -run 'TestDeltaWalkWitness|TestRegistryFingerprintOnFilm' -v -p 1 -parallel 1 -count=1`

Verdict de reference : **`FAIL` sur les TROIS films** — echec ANTERIEUR au lot, lui aussi.
`TestRegistryFingerprintOnFilm` PASSE sur les trois.

| film | paquets delta | records MESURES (figes) | aboutis MESURES (figes) | verdict |
|---|---|---|---|---|
| 000d5950 | 14350 | **38883** (38878) | **30089** (30080) — 77.383 % | FAIL |
| 06dfe6d9 | 6606 | **10610** (10613) | **8489** (8502) — 80.009 % | FAIL |
| 64e8adfa | 14357 | **39818** (39806) | **31990** (31973) — 80.341 % | FAIL |

Empreintes de registre (toutes PASS) :

| film | chunk_00 compresse -> inflate | blocs / slots / porteurs | empreinte mesuree | concordance avec `KnownRegistryFingerprint` |
|---|---|---|---|---|
| 000d5950 | 435 425 -> 1 973 120 (x4.5) | 50 / 1067 / 49 | `0x61e492dd4de7fd4e` | true |
| 06dfe6d9 | 525 722 -> 1 944 960 (x3.7) | 49 / 1031 / 48 | `0x5827362c37d2adb3` | false (film hors empreinte connue) |
| 64e8adfa | 510 435 -> 1 973 120 (x3.9) | 50 / 1067 / 49 | `0x61e492dd4de7fd4e` | true |

### 2.3 Table ECS confrontee au registre d'un film reel

`ECS_TABLE_FILM=<...>/film_chunks/000d5950 go test ./internal/analysis/filmdec/ -run 'TestG1TableSuitLeCode|TestG2TableSuitLeRegistreDuFilm|TestG3TableSuitLeDocument' -v`
-> **`ok ... 0.733s`**, les trois PASS.

```
G1 : PASS (0.07s)
G2 [000d5950] : 50 blocs, 49 porteurs, 1067 lignes de registre confrontees a 1067 lignes de table (+14 alias)
G3 : 27 references de champ verifiees contre 1479 champs du paquet ../replay
```

### 2.4 killcollector, chemin d'integration sur film reel

`KILLSOURCE_FIXTURES=<...>/film_chunks go test -tags=integration ./internal/sync/killcollector/ -v -p 1 -parallel 1 -count=1`
-> **`ok ... 52.181s`** — 67 PASS, 2 SKIP, 0 FAIL.

| test | verdict | duree |
|---|---|---|
| `TestKillSourceCollecteFilmReelEtRelitParLaVue` | PASS | 9.92 s |
| `TestTirsParArmeSuiventLeurPropreCapability` | PASS | 13.40 s |
| `TestKillSourceRosterResoutLesXuid` | PASS | 7.78 s |
| `TestPorteDePrecisionSansFixture` | PASS | 2.17 s |
| `TestPorteDesTirsSansFixture` | PASS | 1.90 s |
| `TestPrecisionSurFilmReel` | SKIP (`KILLSOURCE_HITS_FIXTURE_DIR` absent) | 0.00 s |
| `TestKillSourcePositionsFilmReelEtRelitParLaVue` | SKIP (film 9b191a7f : 90 morts, **0 ligne de position**) | 3.78 s |

Sans garde (`go test ./internal/sync/killcollector/`) : 0 SKIP, tout PASS en 0.140 s.

## 3. Ce que ces mesures disent, et qui n'est PAS a traiter dans ce lot

1. **Deux jeux de references figees sont deja perimes sur l'arbre d'arrivee** — le golden
   `fccc61cd` (1 ligne) et les trois comptes du temoin delta. C'est la preuve mesuree du constat
   P0-1 du registre (« `KillSourceDecoderRev` fige alors que le decodeur a change »), qui est
   traite par le **lot A**. Le lot E ne regenere AUCUNE de ces references : il les garde comme
   temoin d'invariance.
2. `TestKillSourcePositionsFilmReelEtRelitParLaVue` — le seul test de bout en bout de
   `buildPositionRows` sur film reel — se SKIPPE parce que son film code en dur (`9b191a7f`) est
   joue sur une carte absente du catalogue de bornes. Le fichier est hors du perimetre ferme du
   lot E (seul `positions.go` y entre). Consigne, non traite.
