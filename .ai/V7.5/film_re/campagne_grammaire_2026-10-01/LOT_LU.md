# Lot LU — un seul localisateur de la boucle de records, zéro différence (2026-10-04)

> Lot LU du plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 LU, premier lot de la vague 2,
> GO de l'utilisateur du 2026-10-04), sous le contrat `plan-execution`, dans le cadre de la décision
> du 2026-10-02 : **corrections d'abord, uniquement générales, lues dans le jeu**.
>
> Worktree `LevelUp-wt-cg2-lu`, branche `feat/cg2-lu`, base `2393d7db7` (vague 1 fusionnée,
> `grammar-2026-10-03.2`, `killsource-2026-09-27`). Films lus en place
> (`LevelUp/data/cache/film_chunks`, lecture seule), aucune cuisson du parc. Mesures et scripts :
> `scratchpad/v2-LU/` (session `f46f71fc`). GOCACHE `go-build-cg2-lu`, une commande `go` à la fois.
> Convention : **mesuré** = compté par un outil sur les films ; **établi** = lu dans le jeu ;
> **supposé** = hypothèse écrite.

## 0. Statut

| Item | Statut | En une ligne |
|---|---|---|
| LU.1 un localisateur, appelé par tous les sites | [x] | `grammar/localisateur.go` : `LocaliserBoucleDeRecords(pay, w, cfg, ordre)` ; la copie de `killsource/walk.go` (`signature123`, `locateStrict`, `locateFallback`, `locateRecords`, `locateRecordsAvecVerdict`) est supprimée |
| LU.2 paramètre d'ordre (D-73) | [x] | `SignatureStricte` (cuisson) et `SignaturePuisLargeurLibre` (marche des morts d'objet, killsource) ; les deux ordres diffèrent sur DEUX points, pas un (D-LU-1) |
| LU.3 prédicat d'archétype `high-frequency` par table | [~] | aucun prédicat d'archétype n'existe à sortie constante : le localisateur ne reconnaît que le slot 123 (§2.3) ; il revient à LS, sur la clé de L8 (`archetypeHauteFrequence`, D-LU-4) |
| LU.4 garde-rail règle 6 | [x] | `archlint/film_localisateur_unique_test.go`, lu dans l'arbre syntaxique, rouge sur la base (deux copies vues) |
| LU.5 révisions à révision constante | [x] | `grammar-2026-10-03.2` et `killsource-2026-09-27` inchangées ; empreintes régénérées |
| Gates 1, 2, 3 et ceux du lot | [x] | sortie identique partout (§4) |

Verdict : **[x] retenu**.

## 1. Ce qui est lu dans le jeu (établi)

LU n'ajoute aucune règle de lecture : il déplace et réunit un localisateur existant. Une seule
partie de sa signature est fondée dans le jeu, relue pour l'en-tête du fichier unique (Ghidra,
`HaloInfinite.exe`, HTTP direct 127.0.0.1:8089, lecture seule) :

- `FUN_142f2c3b0` (enregistreur par tick), écrivain 0 : `CALL 0x142f2c050` (vue A, liste
  d'événements) en `142f2c578`, puis `XOR R8D,R8D` en `142f2c57d` et `CALL 0x1406d49c4` en
  `142f2c583` ;
- `FUN_1406d49c4` écrit UN bit, la valeur de `param_3` (R8B) : le terminateur `0` de la vue A ;
- `FUN_14076a1c4` (lecteur de la vue A) : `R(1)` ; `0` → fin ; sinon un message (`FUN_14080a9d4`).
  Déjà extrait par R-LOC (`r_loc_ghidra/FUN_14076a1c4.c`) ;
- R-LOC §4.1 (établi) : la vue B commence au bit qui suit ce terminateur, et la longueur de la vue A
  n'est écrite nulle part.

D'où le « bit nul qui précède la position » du localisateur. Le slot 123, la largeur 35 et le
composant unique restent **mesurés** (690 / 690 contre une vérité de position indépendante), comme
avant le lot ; le repli à largeur libre reste un repli nommé (`repli_localisation_largeur_libre`).
Aucun ordre n'est fondé dans le jeu : les deux ordres sont ceux que le dépôt avait déjà, conservés
à l'identique (écart déclaré §6).

## 2. Ce qui change

### 2.1 Code

- **Neuf `grammar/localisateur.go`** : `marchSignatureSlot`, `marchSignatureBits`,
  `marchSignature123`, `marchLocateStrict`, `marchLocateFallback` (déplacés tels quels de
  `object_deaths_march.go`), la constante `resteMinimalALargeurLibre` (16, l'ancien littéral de la
  boucle du repli), le type `OrdreDeLocalisation` et `LocaliserBoucleDeRecords`, qui remplace
  `marchLocalise`. Les trois fonctions qui essaient des positions déclarent la neutralisation des
  états de mouvement (garde-rail `etats_mouvement_porte_guard_test.go`, pointé sur le nouveau
  fichier).
- `grammar/object_deaths_march.go` : n'a plus de localisateur ; `marchDebut` appelle
  `LocaliserBoucleDeRecords(..., SignaturePuisLargeurLibre)`. En-tête réécrit (règle 17).
- `grammar/debut_de_liste.go` : `localiserLaListe` appelle `LocaliserBoucleDeRecords(...,
  SignatureStricte)` au lieu de `marchLocateStrict`.
- `facts/killsource/walk.go` : copie supprimée (−58 lignes, import `source` retiré) ; `runWalk`
  appelle `grammar.LocaliserBoucleDeRecords(..., grammar.SignaturePuisLargeurLibre)`. En-tête du
  localisateur réécrit. Deux instruments de test (`vehicules_v10_deadstate_test.go`,
  `world_precision_test.go`) appellent le localisateur unique.
- `facts/fallback/registre_killsource.go` : `repli_localisation_largeur_libre` cite le site
  `grammar/localisateur.go` (une ancre au lieu de deux) ; `noms.go`, `grammar/replis_du_film.go` :
  chemins cités corrigés.
- Règle 17, affirmations rendues fausses par le lot ou trouvées fausses en le faisant :
  `grammar/observateur.go` (citait `marchLocate`, enveloppe de recherche, comme localisateur de
  production) ; `deto_preuve_robuste_helpers_test.go` (citait `killsource/walk.go:signature123`) ;
  `march_locate_research_test.go` (enveloppe réécrite sur le localisateur unique).
- Tests neufs `grammar/localisateur_test.go`, sur la bobine contiguë
  `facts/killsource/testdata/minibobine_000d5950` :
  - `TestLocaliserBoucleDeRecordsSuitLOrdreDuSite` : ordre de la cuisson = première signature
    stricte telle quelle ; ordre des marches = signature à la génération du monde, sinon repli et
    verdict. La bobine porte les deux cas (sinon `Fatal`) ;
  - `TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches` : monde lié à une AUTRE génération
    sur le slot 123 → la cuisson garde la signature, les marches l'écartent ;
  - `TestLaCuissonNeDemarreQueSurLaSignatureStricte` : un début de liste attribué à la signature
    (`lecture.DebutParSignature`) n'est jamais une position du repli.
- Garde-rail `archlint/film_localisateur_unique_test.go` : dans la production de `film/**` (hors
  `film/research/` et fichiers `research`), deux formes interdites hors de l'hôte : l'**essai de
  position** (`BitAt(_, s-1)` dont la position `s` est essayée par `TryDeltaAt(_, s, …)` dans le même
  fichier) et la **comparaison d'un champ `Slot` au littéral 123** ; l'hôte doit porter exactement
  deux essais. Vecteurs : la copie retirée de killsource (2 essais, 2 slots), le test de bit de
  `killsource/eventchain.go` (0), un essai sur une autre position (0).

### 2.2 Équivalence, par lecture du code

- Killsource : `cfg = grammar.DefaultFrameConfig()`, `cfg.Obs` nul ; la neutralisation des états de
  mouvement que le localisateur unique pose est donc sans effet pour lui
  (`neutraliserEtatsDeMouvement` rend une fonction vide sur un observateur nul). La copie supprimée
  et le localisateur unique avaient le même corps, littéraux compris (`s+35 < nb`, `s+16 < nb`,
  `Slot == 123`, contrôle de génération aux deux étages).
- Cuisson : `SignatureStricte` rend `marchLocateStrict` sans autre essai ; la neutralisation de plus
  s'emboîte (sauvegarde nulle, restauration nulle, puis restauration de l'extérieure).

### 2.3 Le prédicat d'archétype par table (LU.3, [~])

Le plan (§6.2 LU, critique R point 7, `R_COMB_2.md` §7) demande que le prédicat qui reconnaît
l'archétype `high-frequency` s'écrive par la table de son composant, jamais par le nom. À sortie
constante, le localisateur ne reconnaît AUCUN archétype : il exige le slot 123, 35 bits et un
composant unique. Écrire le prédicat ici serait soit du code mort (aucun appelant), soit un
changement de sortie (une conjonction de plus). Il revient à LS, qui en a l'usage (« tout slot lié à
l'archétype `high-frequency` »), sur la clé que L8 a posée : `archetypeHauteFrequence = 4`
(`grammar/components_frequences.go`, `FUN_140e462d8` écrit `+0x4754 = 4`, table `0x143d06a60`,
lecteur `FUN_14076d034`), une seule source de vérité.

## 3. Carte de fermeture v2, avant / après, par film

`cmd_fermeture -mode v2 -denominateur-fixe integ2/denominateurs.tsv -paquets`, binaires de la base
(`git archive 2393d7db7`) et du lot, table ECS identique à l'octet. Juge de L0 (colonne `ferme`).
Comparaison par `integ2/gate2.awk`.

| Film | Sains avant | Sains après | Utiles sains avant | Utiles sains après | Perdus (contredits / non fermés) | Gagnés |
|---|---|---|---|---|---|---|
| bcb6d393 | 5 879 | 5 879 | 35 492 | 35 492 | 0 (0 / 0) | 0 |
| fb1a1a72 | 43 450 | 43 450 | 325 749 | 325 749 | 0 (0 / 0) | 0 |
| d9781168 | 26 317 | 26 317 | 180 854 | 180 854 | 0 (0 / 0) | 0 |
| c75f33b8 | 23 872 | 23 872 | 153 997 | 153 997 | 0 (0 / 0) | 0 |
| bf15f7ab | 28 603 | 28 603 | 216 104 | 216 104 | 0 (0 / 0) | 0 |
| 51ebbc0f | 20 443 | 20 443 | 140 617 | 140 617 | 0 (0 / 0) | 0 |
| 084a804d | 4 832 | 4 832 | 89 616 | 89 616 | 0 (0 / 0) | 0 |
| 0797ce72 | 19 156 | 19 156 | 159 929 | 159 929 | 0 (0 / 0) | 0 |
| 111fa685 | 4 026 | 4 026 | 42 896 | 42 896 | 0 (0 / 0) | 0 |
| e5adf7b2 | 4 147 | 4 147 | 80 067 | 80 067 | 0 (0 / 0) | 0 |
| 60ae07c4 | 13 946 | 13 946 | 83 669 | 83 669 | 0 (0 / 0) | 0 |
| a349fea8 | 424 | 424 | 3 654 | 3 654 | 0 (0 / 0) | 0 |
| a521164d | 692 | 692 | 78 | 78 | 0 (0 / 0) | 0 |
| 11de8353 | 5 629 | 5 629 | 65 621 | 65 621 | 0 (0 / 0) | 0 |
| 50247b26 | 139 | 139 | 274 | 274 | 0 (0 / 0) | 0 |
| bfecd02b | 27 666 | 27 666 | 239 045 | 239 045 | 0 (0 / 0) | 0 |
| 4f77afc1 | 24 082 | 24 082 | 607 244 | 607 244 | 0 (0 / 0) | 0 |
| 396cfc92 | 23 131 | 23 131 | 169 502 | 169 502 | 0 (0 / 0) | 0 |
| f75e7053 | 23 673 | 23 673 | 162 137 | 162 137 | 0 (0 / 0) | 0 |
| 1c4c63c2 | 13 388 | 13 388 | 165 965 | 165 965 | 0 (0 / 0) | 0 |
| **corpus** | **313 495** | **313 495** | **2 922 510** | **2 922 510** | **0** | **0** |

- `fermeture_paquets.tsv` (629 142 paquets) **identique à l'octet** ; 12 autres TSV identiques à
  l'octet ; `fermeture_films.tsv` identique hors `pic_octets` et `duree_ms` ;
  `fermeture_resume.md` identique hors colonne « Pic mémoire » et chemin de la table ECS.
- Listes non localisées 47 854 avant et après (colonne `liste`, même fichier à l'octet).
- Juge des invariants sur gagnés et perdus : 0 et 0 ; part de factices : sans objet (aucun gain).
- Gate 2 : aucun film en baisse, 0 sain perdu au sens strict (les deux comptes bruts nuls).

## 4. Gates (sorties exactes, depuis `apps/go-api`, GOCACHE du lot)

- `gofmt -l ./internal ./cmd` : vide.
- `go vet ./...` : rc 0. `go vet -tags=research ./internal/games/halo_infinite/film/...` : rc 0.
- `go test ./internal/archlint/` : `ok levelup/go-api/internal/archlint 35.783s` (premier passage
  rouge : `TestToutReplinNommeEstAuRegistre` prenait la constante `largeurMinimaleDuRepli` pour un
  repli non inscrit ; renommée `resteMinimalALargeurLibre`, ce n'est pas un repli mais une borne).
- G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) sur
  l'arbre final : 20 paquets `ok`, rc 0.
- `golangci-lint run --new-from-rev 2393d7db7` (grammar, facts, archlint) : `0 issues.`
- **Révisions** : `grammar.Rev` = `grammar-2026-10-03.2` inchangée, empreinte régénérée à révision
  constante (`1bfd06f0…` → `2ed2fd57…`) ; `killsource.Rev` = `killsource-2026-09-27` inchangée,
  empreinte régénérée (`eb0fe3c1…` → `2604bc68…`) ; `objectives` inchangé (empreinte verte sans
  régénération) ; aucune entrée de `rev_chronique.go` (aucune sortie ne change, règle commune).
- **`frame_closure.golden`** : régénéré par `-update-frame-closure`, `git diff` vide : identique.
- **Killsource (gate 3)** : `killsource json <film> -carte … -cache … -catalogue …` sur les 19 témoins
  de `config/replay_corpus.toml` (`integ2/cartes.tsv`), base contre lot : 19 / 19 rc 0, **19 / 19 JSON
  identiques à l'octet**, stderr identiques hors horodatage. Aucune mort, valeur ni voie
  (`read_path`) changée. `1c4c63c2` non joué : killsource refuse un film sans carte (« la carte du
  match est obligatoire »), comme à R-COMB-2 et L0 (D22) ; ce film est couvert par la carte v2.
- **`replay-equiv`** (racine factice `scratchpad/v2-LU/repo`, copie de `integ3/repo` sans faits ni
  artefacts ; `config/` et références `film/replay/testdata/equivalence` identiques à l'octet à celles
  du worktree, vérifié par `diff -rq`) : les deux côtés rendent `BILAN : 0 identique(s), 20
  different(s)` contre les références de `67c379fc1`, qui n'ont pas été re-figées après la vague 1
  (§6) ; **les 20 TSV de digests (61 étapes) de la base et du lot sont identiques à l'octet**.
- **Compteur `repli_debut_de_liste_ferme_au_bit`** (journal du rejeu, « repli declenche ») : identique
  film par film (522 lignes de déclenchements de replis identiques sur 20 films). Valeurs :
  000d5950 2, 01e1f945 9, 084a804d 472, 111fa685 141, 11de8353 128, 1c4c63c2 6 428, 50247b26 14,
  51101d1d 1, 53ce4390 19, 60ae07c4 104, 64e8adfa 60, 696a9d7c 17, 7344d24f 27, 9f57c612 112,
  a349fea8 44, a521164d 5, bcb6d393 5, d9781168 261, e5adf7b2 159, fb1a1a72 39.
  `repli_localisation_largeur_libre` (le repli du localisateur) identique aussi : de 96 (`51101d1d`)
  à 5 290 (`084a804d`).
- Durées et pics de `replay-equiv` : pics égaux à ± 0,07 Gio ; durées du lot plus courtes sur
  plusieurs films (`1c4c63c2` 5 min 50 → 3 min 47), la base ayant tourné pendant la suite G-film
  (machine chargée, mesure non comparable ; LU ne relève pas du gate 4).

### 4.1 Mutations (chacune prouvée rouge)

Surcouche `-overlay` sur `localisateur.go`, `debut_de_liste.go` ou `walk.go` (fichiers
`scratchpad/v2-LU/mut/`), sauf M4 et M7 (le garde-rail lit les sources sur disque, la surcouche ne
l'atteint pas).

| # | Mutation | Rouge |
|---|---|---|
| M1 | l'ordre de la cuisson reçoit le contrôle de génération et le repli | `TestLocaliserBoucleDeRecordsSuitLOrdreDuSite` (« ordre de la cuisson : (377, true), attendu (-1, false) ») |
| M2 | l'ordre des marches garde une signature d'une autre génération | `TestLocaliserBoucleDeRecordsControleLaGenerationDesMarches` (« la signature 89 d une autre generation est gardee ») |
| M3 | l'ordre des marches perd le repli | `TestLocaliserBoucleDeRecordsSuitLOrdreDuSite` et `killsource.TestGoldenMiniBobine` |
| M4 | une copie du localisateur dans un fichier temporaire de `killsource` | `TestLocalisateurDeBoucleUnique` (« 1 essai(s) de position et 1 comparaison(s) de Slot au litteral 123 ») ; fichier retiré, arbre vérifié |
| M5 | killsource câblé sur `SignatureStricte` | `killsource.TestGoldenMiniBobine` |
| M6 | la cuisson câblée sur `SignaturePuisLargeurLibre` | `TestLaCuissonNeDemarreQueSurLaSignatureStricte` (« debut 377 attribue a la signature, signature stricte -1 ») |
| M7 | le garde-rail joué sur l'arbre de la base | rouge : `killsource/walk.go` 2 essais + 2 slots, `grammar/object_deaths_march.go` 2 essais, hôte absent |

M1, M2 et M6 étaient VERTES contre la suite existante (grammar, killsource, replay, replaybuild) avant
les trois tests neufs : rien ne tenait l'ordre de chaque site (D-LU-3).

## 5. Pertes instruites

Aucune : 0 paquet sain perdu, 0 record utile sain perdu, 0 mort killsource changée, 0 digest
`replay-equiv` changé.

## 6. Écarts

- **Critère « chaque ordre porte la fonction du jeu qui le fonde »** : les deux ordres
  (`SignatureStricte`, `SignaturePuisLargeurLibre`) ne sont pas lus dans le jeu ; ce sont ceux que le
  dépôt avait (mesurés, R-LS), conservés à l'identique parce que le lot exige zéro différence. Le
  slot 123, la largeur 35 et le composant unique de la signature sont mesurés, pas lus ; seul le
  bit nul qui précède est établi (§1). Aucune règle neuve n'est introduite.
- **Prédicat d'archétype par table** : non écrit (§2.3, [~] vers LS).
- **`replay-equiv` contre les références** : 20 / 20 différents des deux côtés, parce que les
  références de `67c379fc1` n'ont pas été re-figées après la vague 1 (§6.0 point 6 du plan le
  demande avant le lot suivant ; hors du lot). Le gate « 0 divergence » est joué base contre lot,
  digests à l'octet.
- **Non joués** : `replay-corpus-gate` (§6.0 point 6, hors de la liste des gates du lot ; il
  rejouerait les mêmes sorties que `replay-equiv`, identiques à l'octet) ; banc de vérité ; revue
  adversariale de fin de lot (§6.0 : LU est un lot à risque) — laissée au pilote, comme la revue de
  la vague 1.
- Killsource sur 19 films et non 20 (`1c4c63c2` sans carte, D22).

## 7. Fichiers de la représentation intermédiaire touchés

Aucun. Fichiers touchés : `grammar/localisateur.go` (neuf), `grammar/localisateur_test.go` (neuf),
`grammar/object_deaths_march.go`, `grammar/debut_de_liste.go`, `grammar/observateur.go`,
`grammar/replis_du_film.go`, `grammar/etats_mouvement_porte_guard_test.go`,
`grammar/march_locate_research_test.go`, `grammar/deto_preuve_robuste_helpers_test.go`,
`grammar/testdata/grammar_rev.golden`, `facts/killsource/walk.go`,
`facts/killsource/vehicules_v10_deadstate_test.go`, `facts/killsource/world_precision_test.go`,
`facts/killsource/testdata/killsource_rev.golden`, `facts/fallback/noms.go`,
`facts/fallback/registre_killsource.go`, `archlint/film_localisateur_unique_test.go` (neuf).
`git diff 2393d7db7...feat/ri-etape2` (lecture seule, tête `0280e41bc`) : parmi ces fichiers, la
représentation intermédiaire ne touche que `facts/fallback/noms.go`, dans un autre bloc (l. 218,
replis d'`objectives` ; LU l. 179) ; elle n'ajoute aucun appelant de `marchLocalise`,
`marchLocateStrict`, `locateRecords` ni `locateRecordsAvecVerdict`. Conflit textuel non attendu
(supposé, aucun essai de fusion joué).

## 8. Découvertes (consignées, non traitées)

- **D-LU-1 — Les deux ordres diffèrent sur deux points, pas un.** D-73 disait : la cuisson n'a pas de
  repli à largeur libre. Elle n'a pas non plus le contrôle de génération de la signature stricte :
  `localiserLaListe` prenait `marchLocateStrict` tel quel, quand `marchLocalise` et killsource
  écartent une signature dont la génération n'est pas celle du monde (puis essaient le repli). Le
  paramètre d'ordre porte les deux différences ; leur justesse côté cuisson n'est pas instruite
  (sous `GenerationStricte` faux, le contrôle est neutre ; la cuisson ne lève pas ce drapeau, supposé
  d'après `profil_balayage.go` l. 263-271, non mesuré).
- **D-LU-2 — La boucle de records reste jumelle.** `grammar.marchRecordsOf` et `killsource.walkFrom`
  déroulent la même boucle ; seul le premier pose le cadre du lecteur (`br.poserCadre(cfg)` : profil
  et observateur). Effet non instruit ; hors du lot (LU = localisateur).
- **D-LU-3 — Rien ne tenait l'ordre de chaque site.** M1, M2 et M6 passaient la suite existante :
  l'écart de D-73 et D-LU-1 n'était gardé par aucun test, seulement par les mesures sur films. Tenu
  désormais par `localisateur_test.go`.
- **D-LU-4 — Prédicat de LS.** La reconnaissance de l'archétype `high-frequency` s'écrira
  `TypeIndex == archetypeHauteFrequence` (ou sur la liaison du monde, `ArchetypeForSlot`), la
  constante de L8 ; c'est l'ordre `SignatureStricte` / `SignaturePuisLargeurLibre` qui recevra, par
  site, l'étage « signature sur un autre slot `high-frequency` » (D13).
- **D-LU-5 — Copies de test du localisateur.** `deto_preuve_robuste_helpers_test.go`
  (`rbSignature123`, `rbLocateStrict`, test ordinaire) et `r_loc_ls_research_test.go` (variantes de
  R-LS) recopient la signature ; le garde-rail exclut les tests par construction (instruments).
  Non traité.
- **D-LU-6 — Le localisateur a trois sites, pas deux.** Le plan parle des « deux marches » ; la
  cuisson (`debutDeLaListe`, appelée par `movement_states.go`, `frame_closure.go`,
  `frame_closure_detail.go`) en est un troisième, et c'est elle qui porte l'ordre différent.
