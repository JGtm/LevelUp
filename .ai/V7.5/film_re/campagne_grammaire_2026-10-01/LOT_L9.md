# Lot L9 — marche d'image-clé de toutes les générations (2026-10-03)

> Plan : `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.2 « L9 ». Worktree `LevelUp-wt-cg-l9`,
> branche `feat/cg-l9`, base `af6e93e23` (L0 fusionné). Films lus en place (lecture seule) depuis
> `data/cache/film_chunks` du checkout principal ; 20 films du corpus (19 témoins de
> `config/replay_corpus.toml` + `1c4c63c2`). Ghidra en lecture seule (HTTP 127.0.0.1:8089,
> `decompile_function`, `get_xrefs_to`). Aucune cuisson, aucune base, aucun backfill.
>
> Convention : **lu** = lu dans Ghidra ; **mesuré** = compté sur les films ; **estimé** = déduit
> d'une mesure ou d'une lecture par un raisonnement écrit ; **hypothèse** = ni lu ni mesuré.
> « Sain » = paquet fermé (définition L0 : reste nul et aucune règle de l'écrivain contredite).
> `scratchpad/` = `C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/`
> (non versionné : TSV, surcouches, binaires et décompilations du lot).

## 0. Statut : [!] NON RETENU

Le lot tel qu'il est écrit — lever la garde de génération 0 de `kfAnchorFromID` et ouvrir le voisin,
le recalage et le saut de largeur à toutes les générations — est **lu dans le jeu** (le jeu ne teste
jamais la génération d'un identifiant, §1), mais il **échoue au gate 2 sur 18 films sur 20**
(**−102 902** paquets sains, **−1 283 357** records utiles sains, mesuré, §3). Aucune de ses
parties ne le tient seule (§3.2). La garde de génération était, dans le balayeur d'ancres du Go
(qui n'existe pas dans le jeu), le seul filtre contre les fausses ancres prises dans l'état d'un
record : sans elle, un petit entier suivi d'un mot nul devient un « voisin » de génération 0.

Une variante qui porte AUSSI ce que le jeu fait de la génération à la lecture — l'entrée
d'image-clé du slot `s` n'est consommée qu'appariée à l'entrée `s` de la table de datums, sous sa
génération (§1, `FUN_1408f1730`) — donne **+20 958 sains, +447 733 utiles sains**, déclare 100 %
des entrées vivantes de toutes les générations et aucune déclaration contraire au bloc (mesuré,
variante V3e, §4). Elle n'est pas livrée : (1) elle **échoue encore au gate 2** sur trois films
(`a349fea8` −1, `c75f33b8` −3 / −18, `fb1a1a72` −2 / −38 ; §4.3) ; (2) elle **lit le bloc de type 1
en production**, ce qui la met sous le gate 4 (durée et pic de cuisson) et dans le périmètre de LP
(vague 2) ; (3) sa prémisse « une entrée d'image-clé a toujours une entrée allouée de même
génération au bloc qui la précède » est contredite par 850 records de la marche de référence, non
instruits (§4.4). C'est une décision de l'utilisateur (D-L9-3).

Commit : ce document et la sonde `l9_generations_research_test.go` (tag `research`) seulement.
Aucun fichier de production ne change ; `grammar.Rev` inchangée (`grammar-2026-10-02`).

## 1. Ce qui est lu dans le jeu

| Fonction | Ce qu'elle fait de la génération (lu) |
|---|---|
| `FUN_142f2e598` (pose d'un slot) | `gen = (gen + 1) & 3` dans l'octet `+0x01` de l'entrée de datum, `drapeaux |= 4`, puis `eid = gen << 30 \| slot`. Un slot jamais alloué porte 0 ; ses allocations successives portent 1, 2, 3, puis **0 à la quatrième**. La génération 0 est une génération du jeu. |
| `FUN_142e2bfd0` (lecteur de la table d'image-clé, appelé par `FUN_1428e2a9c` ← `FUN_1428e2a04`) | `R(32)` d'identifiant rangé tel quel en `entrée+0x00`, `R(32)` d'archétype, `R(32)`, `R(4)` (`FUN_142e29cf8`), `R(8)`. **Aucun test sur la génération** ; la seule valeur spéciale est l'archétype `0xffffffff` (pas de corps). La boucle remplit un vecteur PRÉ-DIMENSIONNÉ d'entrées de 200 octets, une entrée après l'autre (`puVar12 += 0x32`). |
| `FUN_142e2aab4` (chargement d'un état d'image-clé) | restaure la table de datums (`FUN_142f22be8(monde + 0x120, …)`, entrées de 0x18 octets) et le vecteur des entités (`FUN_141f850dc(monde + 0x42f0, …)`, entrées de 200 octets), puis appelle `FUN_1408f1730`. |
| `FUN_1408f1730` | parcourt la table de datums ; pour chaque entrée `drapeaux & 1`, `& 4`, ni `& 2` ni `& 0x20`, forme `eid = octet_gen << 30 \| index` et lit l'archétype en `index * 200 + 4` du vecteur des entités : **l'entrée d'image-clé est appariée par SLOT à l'entrée de datum, sous la génération de la table de datums**. |
| `FUN_1408f1618`, `FUN_142f2f73c` (cas 2) | posent ou reforment `eid` depuis l'octet de génération de la table : aucune génération n'est filtrée. |
| `FUN_142f2e174`, `FUN_142f30610`, `FUN_142f2c754` (écrivain de vue) | l'identifiant écrit est le mot `+8` de l'entrée de vue, sans test de génération. |

Le seul identifiant nul du jeu est la sentinelle `0xffffffff` ; « génération 0 = handle nul »
(commentaire de `keyframe_world.go`) est **réfuté par l'allocateur**. Décompilations :
`scratchpad/L9/ghidra/` (20 fonctions, dont les dix extraites à la première exécution du lot).

Estimé (lu en partie) : la table d'image-clé est DENSE par slot (vecteur pré-dimensionné rempli
séquentiellement, relu par index de slot par `FUN_1408f1730`) ; les traînées de sentinelles que le
balayeur Go traite comme « fin de table » seraient les entrées des slots vides. Non vérifié (D-L9-9).

## 2. Ce qui a été essayé dans le code (variante A, le lot tel qu'écrit)

Patch : `scratchpad/L9/l9_variante_A.patch` (retiré de l'arbre, non commité).

- `kfAnchorFromID` : la garde `gen == 0` retirée ; `readKeyframeHeader` : idem.
- `suivante` : voisin `s == prev+1` et recalage `ti == 35` de toute génération ;
  `marcherLaTable` : saut de largeur de toute génération.
- `motifDAncre` : la contrainte « bits 63-62 non nuls » retirée (le motif reste un sur-ensemble).
- Élection (`kfCand.betterThan`) et règle de coïncidence de `TableDeDatums` : « génération basse »
  remplacée par le rang d'allocation `(gen + 3) & 3` (1, 2, 3, puis 0), seul ordre que
  `FUN_142f2e598` justifie.
- Oracle du test différentiel `keyframe_world_motif_test.go` et commentaire de `kfEcrireRecord`
  mis à jour. `go build` du lot vert ; tests non joués (lot arrêté au gate 2).

Révision prévue, non appliquée : le format du dépôt n'admet pas `grammar-2026-10-03.l9`
(`revision/chronique.go:45`, `^AAAA-MM-JJ(\.N)?$`, et `Rang.Suit` exige des rangs sans trou) ; la
seule valeur admise sur cette branche était `grammar-2026-10-03` (L2 et L3a portent chacun
`grammar-2026-10-02.2` : la renumérotation se fait à l'intégration).

## 3. Gate 2 de la variante A, et de ses parties (carte v2, 20 films)

Carte : `cmd_fermeture -mode v2 -denominateur-fixe denominateurs.tsv -paquets` (recette L0,
`scratchpad/L9/carte.sh`). Avant = binaire construit sur `af6e93e23` : ses dix TSV sont identiques à
ceux de « L0 après » (`l0_tsv/carte_apres/`, hors pic et durée), 276 327 sains. Comparaison paquet
par paquet : `net_sains.awk` de L0. Durée d'une carte : 77 s.

### 3.1 Variante A, par film (mesuré)

| Film | Build | Sains avant | Sains après | Net | Sains perdus (devenus contredits / non fermés) | Gagnés | Utiles sains avant | après | Net utiles |
|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | HI_1_13_0 | 19152 | 6735 | −12417 | 12491 (18 / 12473) | 74 | 159920 | 43065 | −116855 |
| 084a804d | HI_1_10_0 | 4773 | 4520 | −253 | 1643 (8 / 1635) | 1390 | 88036 | 64220 | −23816 |
| 111fa685 | HI_1_10_0 | 4012 | 3364 | −648 | 1068 (2 / 1066) | 420 | 42601 | 21930 | −20671 |
| 11de8353 | HI_1_9_0 | 5612 | 5146 | −466 | 806 (11 / 795) | 340 | 65230 | 49843 | −15387 |
| 1c4c63c2 | HI_1_10_0 | 13333 | 11705 | −1628 | 4306 (163 / 4143) | 2678 | 165404 | 102958 | −62446 |
| 396cfc92 | HI_1_13_0 | 22819 | 18316 | −4503 | 4801 (14 / 4787) | 298 | 166984 | 123528 | −43456 |
| 4f77afc1 | HI_1_13_0 | 22260 | 10358 | −11902 | 12476 (170 / 12306) | 574 | 550289 | 193041 | −357248 |
| 50247b26 | version-31 | 139 | 149 | +10 | 0 | 10 | 274 | 275 | +1 |
| 51ebbc0f | HI_1_13_0 | 9759 | 6632 | −3127 | 3302 (12 / 3290) | 175 | 58585 | 30348 | −28237 |
| 60ae07c4 | HI_1_8_0 | 13802 | 11970 | −1832 | 2194 (6 / 2188) | 362 | 82730 | 69121 | −13609 |
| a349fea8 | version-33 | 420 | 431 | +11 | 0 | 11 | 3595 | 3597 | +2 |
| a521164d | HI_1_4_1 | 692 | 696 | +4 | 1 (1 / 0) | 5 | 78 | 74 | −4 |
| bcb6d393 | HI_1_12_0 | 5830 | 3470 | −2360 | 2377 (2 / 2375) | 17 | 35143 | 15026 | −20117 |
| bf15f7ab | HI_1_13_0 | 28465 | 15469 | −12996 | 13295 (15 / 13280) | 299 | 214974 | 99680 | −115294 |
| bfecd02b | HI_1_13_0 | 26380 | 13767 | −12613 | 12654 (3 / 12651) | 41 | 226529 | 102321 | −124208 |
| c75f33b8 | HI_1_13_0 | 22854 | 15588 | −7266 | 7364 (10 / 7354) | 98 | 144430 | 79277 | −65153 |
| d9781168 | HI_1_13_0 | 26210 | 14838 | −11372 | 11536 (19 / 11517) | 164 | 180052 | 89071 | −90981 |
| e5adf7b2 | HI_1_11_0 | 4123 | 3333 | −790 | 1123 (6 / 1117) | 333 | 79457 | 53023 | −26434 |
| f75e7053 | HI_1_13_0 | 23416 | 13105 | −10311 | 10342 (1 / 10341) | 31 | 160042 | 79806 | −80236 |
| fb1a1a72 | HI_1_13_0 | 22276 | 13833 | −8443 | 8507 (5 / 8502) | 64 | 161566 | 82358 | −79208 |
| **corpus** | | 276327 | 173425 | **−102902** | 110286 (466 / 109820) | 7384 | 2585919 | 1302562 | **−1283357** |

Juge sur gagnés et perdus (mesuré, `factices.awk`) : 11 157 paquets gagnés au bit près, dont
**4 678 factices (41,9 %)** ; 111 729 perdus au bit près, dont 1 909 factices dans la référence.

**Mécanisme (mesuré, sonde §5)** : la variante A déclare 5 368 records de génération 0, **tous
contraires au bloc de type 1** (3 489 sur un slot vivant sous une autre génération, 1 879 sur un
slot non vivant), et **0 des 191** entités vivantes de génération 0 ; la génération 1 déclarée
tombe de 372 871 à 333 500. Exemple `bcb6d393` chunk 1 : le vrai record « slot 8, génération 1,
ti 6 » au bit 16 278 est remplacé par « slot 8, génération 0, ti 0 » au bit 14 431, 1 847 bits plus
tôt, dans le corps du record précédent : la décision « voisin » rend le premier candidat
consécutif dans l'ordre des bits, et `0x00000008` suivi d'un mot nul y suffit désormais.

### 3.2 Les parties de A, une à une (mesuré, surcouche `-overlay`, `scratchpad/L9/var_*`)

| Variante | Contenu | Net sains | Net utiles sains | Films en baisse (sains ou utiles) |
|---|---|---|---|---|
| V1 | génération 0 toujours refusée ; voisin, recalage, saut ouverts aux générations 2-3 | −4 174 | −8 863 | 10 (`bfecd02b` −2 156, `f75e7053` −1 262, `bcb6d393` −642, `4f77afc1` −574…) |
| V2 | A, mais la table de datums (`TableDeDatums`) et `readKeyframeHeader` refusent la génération 0 | −101 686 | −1 255 040 | 18 |
| V5a | `readKeyframeHeader` seul accepte la génération 0 (lecteur à position exacte : preuve, marche déterministe, balayages navpoint/objectif) | **0** | **0** | 0 (aucun paquet ne change) |
| V5b | `TableDeDatums` seule accepte la génération 0 (+ rang d'allocation) | −1 558 | −42 231 | 4 (`4f77afc1` −1 066, `1c4c63c2` −717, `51ebbc0f` −197, `fb1a1a72` −2) |

Lecture : la perte vient de la génération 0 dans le balayeur (A et V2 identiques à 1,2 % près) ; les
générations 2-3 au voisin et au recalage perdent aussi (V1, ce que le commentaire du recalage
disait déjà des 14 faux d'`a0c36016`). Seul V5a, le lecteur d'en-tête à position exacte, est
neutre : correction juste et sans gain mesuré, non livrée seule (D-L9-8).

## 4. La variante « témoin de la table de datums » (recherche, non livrée)

### 4.1 La règle (lue en partie, §1)

Sources : `scratchpad/L9/var_V3/` et `l9_variante_V3e_*.diff`.

- Toutes les générations sont des ancres (A).
- Un candidat n'est une ancre que si le bloc de type 1 qui PRÉCÈDE le paquet d'image-clé porte, au
  même slot, une entrée **allouée** (drapeaux ≠ 0) **de la même génération** — c'est l'appariement
  par slot de `FUN_1408f1730` et l'écriture conjointe de `FUN_142f2e598`. Appliqué au balayeur
  (voisin, recalage, saut, élection, écartés) et à `TableDeDatums`.
- L'élection perd sa clé de génération (consécutif, puis slot bas, puis bit bas) : la génération
  est témoignée par la table, le rang d'allocation n'a plus d'objet.
- `MarcheDImageCle.Records(pay)` garde sa forme : le contexte du film indexe, une fois, la table de
  chaque bloc par identité du payload d'image-clé qui le suit (même clé que la mémoire des marches).
  Sans bloc, aucun témoin (comportement à définir pour une livraison : D-L9-3).

Variantes mesurées : V3a (même génération seulement) −14 950 sains, 14 films en baisse ; V3b
(+ entrée allouée) +7 044, 4 films ; V3c (V3b + `TableDeDatums` sous témoin) +7 294, 3 films ; V3d
(V3b, `TableDeDatums` sans génération 0) +6 715, 2 films ; **V3e (V3c + élection sans
génération) +20 958, 3 films**.

### 4.2 V3e par film (mesuré)

| Film | Build | Sains avant | après | Net | Perdus (contredits / non fermés) | Gagnés | Utiles sains avant | après | Net utiles |
|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | HI_1_13_0 | 19152 | 20617 | +1465 | 1 (0 / 1) | 1466 | 159920 | 176155 | +16235 |
| 084a804d | HI_1_10_0 | 4773 | 6662 | +1889 | 0 | 1889 | 88036 | 139637 | +51601 |
| 111fa685 | HI_1_10_0 | 4012 | 4993 | +981 | 0 | 981 | 42601 | 69044 | +26443 |
| 11de8353 | HI_1_9_0 | 5612 | 6188 | +576 | 0 | 576 | 65230 | 78560 | +13330 |
| 1c4c63c2 | HI_1_10_0 | 13333 | 18027 | +4694 | 1434 (224 / 1210) | 6128 | 165404 | 305651 | +140247 |
| 396cfc92 | HI_1_13_0 | 22819 | 24424 | +1605 | 0 | 1605 | 166984 | 182235 | +15251 |
| 4f77afc1 | HI_1_13_0 | 22260 | 25609 | +3349 | 186 (43 / 143) | 3535 | 550289 | 656209 | +105920 |
| 50247b26 | version-31 | 139 | 139 | 0 | 0 | 0 | 274 | 274 | 0 |
| 51ebbc0f | HI_1_13_0 | 9759 | 10003 | +244 | 12 (0 / 12) | 256 | 58585 | 60966 | +2381 |
| 60ae07c4 | HI_1_8_0 | 13802 | 14814 | +1012 | 0 | 1012 | 82730 | 92174 | +9444 |
| a349fea8 | version-33 | 420 | 419 | **−1** | 1 (0 / 1) | 0 | 3595 | 3595 | 0 |
| a521164d | HI_1_4_1 | 692 | 692 | 0 | 0 | 0 | 78 | 78 | 0 |
| bcb6d393 | HI_1_12_0 | 5830 | 5830 | 0 | 0 | 0 | 35143 | 35143 | 0 |
| bf15f7ab | HI_1_13_0 | 28465 | 29071 | +606 | 0 | 606 | 214974 | 220624 | +5650 |
| bfecd02b | HI_1_13_0 | 26380 | 28051 | +1671 | 0 | 1671 | 226529 | 245594 | +19065 |
| c75f33b8 | HI_1_13_0 | 22854 | 22851 | **−3** | 3 (0 / 3) | 0 | 144430 | 144412 | **−18** |
| d9781168 | HI_1_13_0 | 26210 | 28280 | +2070 | 0 | 2070 | 180052 | 199037 | +18985 |
| e5adf7b2 | HI_1_11_0 | 4123 | 4925 | +802 | 0 | 802 | 79457 | 102694 | +23237 |
| f75e7053 | HI_1_13_0 | 23416 | 23416 | 0 | 0 | 0 | 160042 | 160042 | 0 |
| fb1a1a72 | HI_1_13_0 | 22276 | 22274 | **−2** | 2 (0 / 2) | 0 | 161566 | 161528 | **−38** |
| **corpus** | | 276327 | 297285 | **+20958** | 1639 (267 / 1372) | 22597 | 2585919 | 3033652 | **+447733** |

- Juge sur gagnés et perdus : 22 769 gagnés au bit près dont **273 factices (1,2 %)** ; 4 895
  perdus au bit près dont 3 523 factices dans la référence (exception D2) ; fermetures factices
  du corpus 8 758 → 5 674.
- Indicateur (D1, utiles sains / fixe consolidé de R-COMB-2, 20 films : 7 758 290) : 33,3 % → 39,1 %.
  Aucun film ne dépasse son fixe.
- Sonde (§5) : toutes les entrées vivantes déclarées, génération 0 **191 / 191**, 1 373 612 / 373 612,
  2 3 158 / 3 158, 3 418 / 418 (référence : 0, 372 871, 305, 0) ; **aucune déclaration contraire au
  bloc** ; 2 400 records d'entités libérées (drapeaux `0x7`) sous leur génération.
- Killsource (information, binaire construit sous la même surcouche, 19 témoins) : 15 sorties
  identiques à l'octet ; `111fa685` (2 morts) et `11de8353` (1 mort) passent du `scan` à la marche,
  avec leurs compteurs d'accord ; `51ebbc0f` et `bf15f7ab` : la chaîne de diagnostic `calibration`
  seule. Aucune valeur de mort ne change.

### 4.3 Les six paquets perdus (instruits en partie)

| Film | Paquets | Avant → après | Ce qui change (mesuré) |
|---|---|---|---|
| `c75f33b8` | 11:712, 11:716, 16:1148 | localisée, fermé → liste non localisée | aucun record de la marche ne change dans le chunk ni dans le précédent ; la table de datums perd le slot 254 → `ti 8`, lu sur un en-tête de génération 2 alors que le bloc dit le slot jamais alloué (drapeaux 0, génération 0) |
| `a349fea8` | 10:932 | localisée, fermé → liste non localisée | idem, slot 256 → `ti 43`, génération 2, bloc : jamais alloué |
| `fb1a1a72` | 28:750, 28:752 | fermés → sortie de vue B par rejet / ordre et masque contredits | les slots 152-155 sont désormais liés `ti 3`, génération 0 (bloc : vivants, génération 0) ; la référence liait 152 à `ti 10` par un en-tête de génération 2 que le bloc contredit. Estimé : la liaison juste expose la grammaire `ti=3` non portée (L8) |

Les pertes de `c75f33b8` et `a349fea8` existent aussi en V3d (table sans témoin) : elles passent
alors par les candidats ÉCARTÉS de la marche, que la liaison du chunk relit (estimé, non tracé
paquet par paquet). Aucune n'est une fermeture factice retirée : l'exception D2 ne s'applique pas.

### 4.4 Ce qui contredit la prémisse (mesuré, non instruit)

Marche de RÉFÉRENCE, 20 films : 850 records sur des slots que le bloc qui précède dit non alloués
(drapeaux 0) : 825 sous une autre génération que celle du bloc (814 de génération 1, 10 de
génération 2, 1 de génération 3), 25 de génération 1 sous la même génération. Exemple : `084a804d` chunks 37-39, slots
256 à 265, `ti 17`, enchaînés en voisins. Ou ce sont de fausses ancres (V3e les retire, sans perte
sur ce film : +1 889), ou le bloc n'est pas le témoin complet de l'image-clé. Tant que ce n'est pas
tranché, la règle de V3e n'est que « lue en partie ».

## 5. Le témoin d'en-tête (« 87 sur 87 »)

Sonde `l9_generations_research_test.go` (`TestL9Generations`), sur la première image-clé de chaque
chunk, toutes les entrées du bloc qui la précède (recherche exhaustive, même critère que
`b3EnTeteDImageCle`) :

| Classe d'entrée du bloc | Entrées | En-tête exact présent | Déclarées : référence / A / V3e |
|---|---|---|---|
| vivante, génération 0 | 191 | 191 (100 %) | 0 / 0 / 191 |
| vivante, génération 1 | 373 612 | 373 612 | 372 871 / 333 500 / 373 612 |
| vivante, génération 2 | 3 158 | 3 158 | 305 / 1 563 / 3 158 |
| vivante, génération 3 | 418 | 418 | 0 / 179 / 418 |
| **témoin négatif** : jamais allouée (drapeaux 0, génération 0), identifiant de génération 0 | 4 016 555 | 251 635 (6,3 %) | 0 / 1 029 / 0 |

Le témoin négatif par tranche de slot : 26,6 % sous 1 024, 7,35 % de 1 024 à 2 047, 4,0 % de 2 048
à 4 095, 3,0 % au-delà. Les 87 eid de P1 (BIS_3 §2 : 57 de génération 0, 30 de génération 2) sont
des entrées vivantes de ces classes ; leur liste exacte n'a pas été recalculée (elle exige le
diagnostic de la marche des deltas de BIS_3, 2 573 s). La présence à 100 % des 191 entrées vivantes
de génération 0 (171 entre 1 024 et 2 047, contre 7,35 % par hasard) dit que leurs records sont
bien dans l'image-clé ; l'argument de BIS_3 « le hasard d'un motif de 58 bits est négligeable » est,
lui, faux pour la génération 0 (D-L9-2).

## 6. Gates

Joués sur l'arbre commité (production inchangée, sonde ajoutée), depuis `apps/go-api`,
`GOCACHE=…/go-build-cg-l9`, une commande `go` à la fois :

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go test ./internal/archlint/ -count=1` | `ok levelup/go-api/internal/archlint 78.782s` |
| sonde sur la tête (`bcb6d393`) | `ok … grammar 0.538s` ; records identiques à ceux du binaire « avant » |
| gate 2 (carte v2, 20 films) | §3 et §4 : **rouge** pour A, V1, V2, V5b, V3a à V3e ; neutre pour V5a |

Non joués, parce que le lot s'arrête au gate 2 et que le commit ne change aucune sortie : G-film,
`keyframe_closure.golden`, killsource du lot (joué pour V3e à titre d'information, §4.2),
`replay-equiv`, mutations, montée de révision. Les binaires « avant » de la première exécution ne
portaient aucune marque VCS : ils ont été reconstruits depuis l'arbre propre à `af6e93e23`.

Sorties : `scratchpad/L9/` — `carte_avant/`, `carte_{A,V1,V2,V5a,V5b,V3_gen,V3_gen_drapeaux,V3c,V3d,V3e}/`,
`*_net.tsv`, `*_perdus.txt`, `sonde20_{avant,A,V3c,V3e}/` (`l9_records.tsv`, `l9_temoin.tsv`,
`l9_gen0.tsv`), `slots*/l9_slots.tsv`, `ks_avant/`, `ks_V3e/`, `ghidra/`.

## 7. Écarts

- Statut [!] : le lot ne livre aucune correction de production.
- La révision `grammar-2026-10-03.l9` demandée n'est pas admise par le format du dépôt (§2).
- Le témoin « 87 sur 87 » est remplacé par le témoin exhaustif de toutes les entrées vivantes
  plus un témoin négatif (§5) ; les 87 eid eux-mêmes ne sont pas recalculés.
- Revue adversariale de fin de lot (plan §6.0, lots à risque) non jouée : le lot n'est pas retenu.
- D'autres processus `go` (sessions L8) tournaient sur la machine pendant le lot ; aucun dans ce
  worktree (arbre propre vérifié à l'entrée).

## 8. Découvertes

- **D-L9-1** — La garde de génération du balayeur d'ancres (`kfAnchorFromID`, voisin, recalage,
  saut) n'a aucun fondement dans le jeu, mais elle est le seul filtre du balayeur contre les fausses
  ancres de l'état : la lever seule perd 102 902 paquets sains (§3). Toute marche « toutes
  générations » exige un autre témoin de l'identifiant.
- **D-L9-2** — Pour la génération 0, la présence d'un en-tête exact n'est pas probante à elle seule
  (6,3 % de présence par hasard, 26,6 % sous le slot 1 024, §5) ; elle l'est par contraste (191 / 191).
  Les témoins de BIS_3 qui raisonnent sur « 58 bits » sont à relire pour la génération 0.
- **D-L9-3** — Décision demandée : une marche « toutes générations sous le témoin de la table de
  datums » (V3e, §4) donne +20 958 sains / +447 733 utiles sains et 100 % des entrées vivantes
  déclarées, mais lit le bloc de type 1 en production (gate 4, voisin de LP), perd 6 paquets sur
  3 films et repose sur une prémisse contredite par 850 records (§4.4). À instruire avant tout lot :
  ces 850 records, les 6 pertes, le comportement sans bloc, la durée et le pic de cuisson.
- **D-L9-4** — 1 492 records de la marche de référence (2 400 sous V3e) sont des entités libérées
  (drapeaux `0x7`) sous leur génération : l'image-clé porte des entités détruites dont le `DEL` n'est
  pas soldé. LP (« désavouer les déclarations non vivantes ») les retirerait : à peser dans LP.
- **D-L9-5** — `fb1a1a72` 28:750 et 28:752 : une liaison juste (slots 152-155, `ti 3`, génération 0,
  vivants au bloc) fait perdre deux paquets que la référence fermait sur une liaison fausse
  (`ti 10`, génération 2) ; interaction estimée avec la grammaire `ti=3` non portée (L8).
- **D-L9-6** — L'ordre « génération basse » de l'élection, même réécrit en rang d'allocation, empêche
  de lire les records de génération 0 du pool 3 (`ti 41`, slots 1 024-1 279) : après le dernier
  véhicule, l'élection préfère le slot 1 282 de génération 1 au slot 1 129 de génération 0
  (`0797ce72` chunk 11) et saute tout le pool (V3c 0 / 191, V3e 191 / 191 ; `l9_gen0.tsv`).
- **D-L9-7** — `readKeyframeHeader` qui accepte la génération 0 est une correction juste (lecteur à
  position exacte) et neutre sur la carte (V5a : 0 paquet changé) ; non livrée seule, faute de gain.
- **D-L9-8** — `keyframe_world.go` affirme « gen==0 = handle null » et `keyframe_datums.go`
  « génération non nulle » comme garde de l'écrivain : affirmations fausses au regard de
  `FUN_142f2e598` (règle 17, à corriger avec le lot qui changera ces gardes).
- **D-L9-9** — Estimé : la table d'image-clé est dense par slot (§1) ; si c'est confirmé, la marche
  déterministe (`WalkKeyframeRecords`) pourrait compter les entrées au lieu de chercher des ancres.
  Non vérifié (écrivain de la table d'image-clé non localisé ; `DAT_144dbfc90` est une borne
  générique, 30 lecteurs).
