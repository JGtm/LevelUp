# L'image-clé sous la forme d'ÉTAT COMPLET : ce que le correctif gagne, archétype par archétype

Date : 2026-09-13, phase 5a. Suite de `NOTE_PROFIL_PAR_BUILD_2026-09-12.md` (phase 4), dont
elle CHIFFRE la découverte hors périmètre n°2 : « le modèle de record d'image-clé de la
PRODUCTION est faux ». Travail **hors ligne, recherche seulement, aucun code de production
modifié, aucun commit**. Le but de ce lot est de mesurer le gain AVANT qu'un lot de production
soit décidé — il ne corrige rien.

Instruments (tous sous garde d'environnement, sautés en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/imagecle_fermeture_research_test.go` | le tableau archétype × modèle : records fermés / total, sous la bonne forme, sous un témoin de hasard, et sous le modèle de la production |
| `apps/go-api/internal/analysis/filmdec/imagecle_oracle_n2_research_test.go` | `n2` comme oracle de largeur d'état par défaut, par build et par archétype |
| `apps/go-api/internal/analysis/filmdec/imagecle_production_research_test.go` | ce que la production déraille aujourd'hui, avec ses propres compteurs |

Corpus : 6 films, 3 builds — `000d5950`, `00162144`, `00502e52` (`HI_1_13_0`), `0014603f`,
`02784ce1` (`HI_1_12_0`), `00ba2e1c` (`HI_1_11_0`). **62 686 records d'image-clé bornés.**

---

## Résumé exécuté

1. **LA BONNE FORME FERME 8 796 RECORDS SUR 62 686 (14,0 %) ; LE MODÈLE DE LA PRODUCTION EN
   FERME 0 SUR 62 686 (0,0 %).** Mesuré sur les mêmes records, sous les mêmes bascules de
   process, à couverture de composants identique. Le témoin de hasard (le même lecteur, en-tête
   décalé d'UN bit) ferme 529 sur 62 686 (0,8 %) : le plancher est mesuré, pas calculé.
2. **CINQ ARCHÉTYPES PASSENT DE 0 % À UN TAUX QUASI PARFAIT, ET C'EST LE GAIN NET.** `ti=6`
   (statborg) **7 820 / 7 822 = 100,0 %**, `ti=15`, `ti=18`, `ti=22` **163 / 163 = 100,0 %**
   chacun, `ti=4` **112 / 157 = 71,3 %** — contre **0 / total** pour chacun sous la production
   ET sous le témoin de hasard.
3. **LE GAIN APPARENT DE `ti=38` EST DU BRUIT, ET LE TÉMOIN DE HASARD LE DIT.** 373 / 21 404
   (1,7 %) sous la bonne forme contre **364 / 21 404 (1,7 %) sous le témoin décalé d'un bit** :
   aucune conclusion ne se tire de cet archétype. Publier le chiffre sans son témoin l'aurait
   compté comme un acquis.
4. **AUCUN ARCHÉTYPE NE RÉGRESSE.** Sur les 33 archétypes rencontrés : la bonne forme gagne sur
   8, perd sur 0, égalité sur 25 (toutes à 0 / 0).
5. **LE TÉMOIN POSITIF `ti=9` NE FERME PAS, ET LA CAUSE EST NOMMÉE — CE N'EST PAS LE CADRE.**
   0 / 1 679, et les 1 679 marches s'arrêtent toutes au MÊME composant :
   `i4 managed-player-forge-weather-effect-overrides-component`, qui n'est pas porté par le
   décodeur. La dérivation de la phase 4 (le premier composant à 186) reste entière : la marche
   traverse i0 à i3 avant de buter. **C'est un manque de COUVERTURE, pas un défaut de forme** —
   et la distinction est mesurée, pas plaidée : `d` (désync) est compté à part de `s`
   (sous-lecture) et `u` (dépassement) dans tous les tableaux.
6. **CINQ LARGEURS D'ÉTAT PAR DÉFAUT MANQUANTES SONT MESURÉES ET CONFIRMÉES PAR LE
   DÉSASSEMBLAGE** (`ti=14` → 6 bits, `ti=17` → 8, `ti=21` → 18, `ti=29` → 1, `ti=47` → 6 ;
   toutes portées à 0 bit aujourd'hui). Les trois premières font fermer **10 541 records de
   plus** : `ti=14` 5 024/5 024 et `ti=17` 5 379/5 379 à 100 % sur les trois builds.
   **Projection : 19 337 / 62 686 = 30,8 %**, pour trois entrées dans `defaultStateDeserByTI`.
7. **LA PRODUCTION NE DÉRAILLE PAS PAR MANQUE DE COUVERTURE, ELLE DÉRAILLE PAR LE CADRE.**
   Sous son modèle, **seuls 3 477 records sur 62 686 (5,5 %) désynchronisent** : les 57 849
   autres marchent JUSQU'AU BOUT et atterrissent au mauvais bit. Le masque de présence fait
   croire que presque aucun composant n'est présent, la marche se termine trop tôt,
   proprement, et faux.
8. **IL N'EXISTE AUCUN COMPTEUR DE PRODUCTION « LA TABLE D'IMAGE-CLÉ A DÉRAILLÉ ».**
   `KillSourceHealth` compte des candidats d'attribution de mort, pas des records ;
   `WalkKeyframeRecords` n'a aucun appelant hors du paquet ; le seul chemin de production qui
   parse le corps d'un record d'image-clé est `ScanNavpointRadial` (ti=12), appelé par
   l'armement d'Assaut du rejeu. **Le correctif de forme ne répare aujourd'hui la sortie
   d'aucune page — il ouvre une voie.**

---

## A. Le tableau archétype × modèle

### A.1 Ce que « fermer » veut dire, et sur quel dénominateur

La frontière visée est l'ancre du record SUIVANT rendue par `WalkKeyframeWorld`. Sous la lecture
d'état complet, le filtre de ce balayeur n'est pas arbitraire : il exige que le mot de 32 bits à
`+32` vaille moins de 50, ce qui est EXACTEMENT le `typeIndex` que `FUN_142e2bfd0` teste
(`!= 0xffffffff`, borné par les 50 archétypes objet). Le balayeur est donc, sous cette lecture,
un filtre du format lui-même — et c'est une différence de nature avec le modèle de production,
où le même filtre signifie « les 26 bits de `field` sont nuls », une régularité sans justification.

Dénominateur unique : les records **bornés**, ceux dont le balayeur rend un voisin suivant. Un
record non borné n'a pas de frontière à atteindre et ne compte nulle part.

Quatre issues, exclusives, publiées séparément dans chaque cellule (`fermés/total  %  d.. s.. u..`) :

| code | sens |
|---|---|
| **fermé** | marche complète (aucun composant non porté) ET fin == frontière visée |
| `d` désync | marche arrêtée sur un composant non porté — **couverture du dispatch**, pas forcément le cadre |
| `s` sous-lecture | marche complète, fin AVANT la frontière |
| `u` dépassement | marche complète, fin APRÈS la frontière |

### A.2 LES TROIS MODÈLES COMPARÉS

| colonne | lecture | code qui la joue |
|---|---|---|
| **ÉTAT COMPLET** | en-tête 108 bits + `n1` (32) + état par défaut(ti) + `n2` (32) + composants dans l'ordre du registre, **sans masque** | `WalkKeyframeFullState` (`keyframe_fullstate_loop.go`, porté au lot R7-d, jamais branché) avec les déserialiseurs d'état par défaut existants |
| **TÉMOIN +1 bit** | la MÊME lecture, en-tête par entité à **109** bits | le même code, `HeaderBits` décalé |
| **PRODUCTION** | en-tête 64 bits + état par défaut + porte `R(1)` + **masque de présence** + composants présents | `readKeyframeHeader` puis `walkOneKeyframeRecord` — **le code de production lui-même**, celui que `WalkKeyframeRecords` enchaîne |

Le témoin de hasard n'est pas décoratif : la règle 4 de `METHODE_RETRO_INGENIERIE_FILM.md`
interdit de CALCULER un plancher de faux positifs sur ce flux. Il se mesure, et il vaut **0,8 %**.

### A.3 LE TABLEAU, tous films (62 686 records bornés)

| ti | ÉTAT COMPLET | TÉMOIN +1 bit | PRODUCTION |
|---|---|---|---|
| 0 | 0/6 — d6 | 0/6 | 0/6 |
| 1 | 0/1 — d1 | 0/1 | 0/1 |
| 2 | 0/163 — d163 | 0/163 | 0/163 |
| **4** | **112/157 = 71,3 %** (s45) | **0/157** | **0/157** |
| 5 | 0/5 216 — d5 216 | 0/5 216 | 0/5 216 |
| **6** | **7 820/7 822 = 100,0 %** | **0/7 822** | **0/7 822** |
| 9 | 0/1 679 — d1 679 | 0/1 679 | 0/1 679 |
| 10 | 0/1 364 — d1 364 | 0/1 364 | 0/1 364 |
| 11 | 0/24 — d24 | 0/24 | 0/24 |
| 12 | 0/366 — d366 | 0/366 | 0/366 |
| 13 | 0/1 240 (s1 085 u155) | 0/1 240 | 0/1 240 |
| 14 | 0/5 024 (u5 024) | 0/5 024 | 0/5 024 |
| **15** | **163/163 = 100,0 %** | **0/163** | **0/163** |
| 17 | 0/5 379 (u5 379) | 0/5 379 | 0/5 379 |
| **18** | **163/163 = 100,0 %** | **0/163** | **0/163** |
| 19 | 0/163 — d163 | 0/163 | 0/163 |
| 20 | 0/46 — d46 | 0/46 | 0/46 |
| 21 | 0/373 (s373) | 0/373 | 0/373 |
| **22** | **163/163 = 100,0 %** | **0/163** | **0/163** |
| 25, 26, 27 | 0/163 — d163 chacun | 0/163 | 0/163 |
| **29** | **0/157** | **138/157 = 87,9 %** | **0/157** |
| 34 | 0/163 — d163 | 0/163 | 0/163 |
| 35 (bipède) | 0/1 544 — d1 544 | 0/1 544 | 0/1 544 |
| 37 | 1/3 381 = 0,0 % | 26/3 381 = 0,8 % | 0/3 381 |
| 38 | 373/21 404 = **1,7 %** | 364/21 404 = **1,7 %** | 0/21 404 |
| 40 | 0/357 — d357 | 0/357 | 0/357 |
| 41 | 1/112 | 1/112 | 0/112 |
| 42 | 0/2 014 | 0/2 014 | 0/2 014 |
| 43 | 0/1 716 — d1 716 | 0/1 716 | 0/1 716 |
| 45 | 0/158 — d158 | 0/158 | 0/158 |
| 47 | 0/1 679 — d1 679 | 0/1 679 | 0/1 679 |
| **TOTAL** | **8 796 / 62 686 = 14,0 %** | **529 / 62 686 = 0,8 %** | **0 / 62 686 = 0,0 %** |

### A.4 LE MÊME TOTAL, PAR BUILD ET PAR FILM — la mesure ne tient pas à un film

| film | build | ÉTAT COMPLET | TÉMOIN +1 bit | PRODUCTION |
|---|---|---|---|---|
| `000d5950` | `HI_1_13_0` | 1 378/7 799 = **17,7 %** | 83/7 799 = 1,1 % | **0/7 799** |
| `00162144` | `HI_1_13_0` | 1 284/14 932 = **8,6 %** | 198/14 932 = 1,3 % | **0/14 932** |
| `00502e52` | `HI_1_13_0` | 1 539/8 020 = **19,2 %** | 56/8 020 = 0,7 % | **0/8 020** |
| `0014603f` | `HI_1_12_0` | 1 020/5 589 = **18,3 %** | 67/5 589 = 1,2 % | **0/5 589** |
| `02784ce1` | `HI_1_12_0` | 2 171/14 514 = **15,0 %** | 97/14 514 = 0,7 % | **0/14 514** |
| `00ba2e1c` | `HI_1_11_0` | 1 404/11 832 = **11,9 %** | 28/11 832 = 0,2 % | **0/11 832** |
| `HI_1_11_0` | 1 film | 1 404/11 832 = 11,9 % | 28/11 832 = 0,2 % | **0/11 832** |
| `HI_1_12_0` | 2 films | 3 191/20 103 = 15,9 % | 164/20 103 = 0,8 % | **0/20 103** |
| `HI_1_13_0` | 3 films | 4 201/30 751 = 13,7 % | 337/30 751 = 1,1 % | **0/30 751** |

**Zéro fermeture de la production sur les six films et les trois builds.** Ce n'est pas un taux
faible : c'est un compte nul sur 62 686 tentatives.

### A.5 CE QUE LE GAIN CONTIENT VRAIMENT — le témoin de hasard trie

Le gain net, une fois le plancher de hasard retiré, tient à **cinq archétypes** :

| ti | nom usuel | fermés (bonne forme) | témoin +1 bit | production |
|---|---|---|---|---|
| **6** | statborg | **7 820 / 7 822** | 0 | 0 |
| **4** | — | **112 / 157** | 0 | 0 |
| **15** | — | **163 / 163** | 0 | 0 |
| **18** | — | **163 / 163** | 0 | 0 |
| **22** | — (état par défaut STUB confirmé) | **163 / 163** | 0 | 0 |
| | **sous-total** | **8 421** | **0** | **0** |

Les 375 fermetures restantes (`ti=37`, `38`, `41`) sont **au plancher du hasard** (365 pour les
mêmes archétypes sous le témoin décalé) : **elles ne sont pas publiables comme un acquis.**

**Le témoin de hasard a trouvé autre chose que du bruit, et c'est une piste, pas un accident** :
`ti=29` ferme **138/157 (87,9 %) sous le témoin décalé d'un bit** et 0/157 à la largeur portée.
Un décalage d'un bit qui fait passer un archétype de 0 à 88 % n'est pas du hasard — c'est que
son état par défaut fait UN bit de plus que ce que le dépôt consomme. La section B le confirme
par deux chaînes et le chiffre.

`ti=4` est le seul des cinq à dépendre du build : **74/74 sur `HI_1_13_0`**, **38/57 sur
`HI_1_12_0`**, **0/26 sur `HI_1_11_0`** (sous-lecture pure). Son état par défaut est un STUB
(0 bit) dans le dépôt ; la sous-lecture dit qu'il a gagné de la largeur avant `HI_1_12_0` ou
qu'un composant a changé. Non tranché ici.

### A.6 POURQUOI LES 25 AUTRES ARCHÉTYPES NE FERMENT PAS — et ce n'est pas la même raison

La ventilation `d / s / u` sépare deux mondes, et c'est le résultat le plus utile pour un lot de
production :

**(1) COUVERTURE MANQUANTE — 15 134 records sur 62 686 (24,1 %) s'arrêtent sur un composant non
porté.** Ces archétypes ne peuvent PAS fermer tant que le composant n'est pas porté, quel que
soit le cadre. Un composant par archétype suffit à bloquer tout l'archétype :

| ti | composant qui arrête la marche | records |
|---|---|---|
| 9 | `i4 managed-player-forge-weather-effect-overrides-component` | 1 679 |
| 5 | `i22 player-aim-assist-component` | 5 216 |
| 47 | `i2 personal-ai-data-component` | 1 679 |
| 43 | `i19 device-position-animation-name-component` | 1 716 |
| 35 | `i60 simulation-state-component` (1 390) / `i59 biped-spartan-ability-non-predicted-state` (86) | 1 544 |
| 10 | `i2 managed-object-navpoint-component` | 1 364 |
| 40 | `i30 vehicle-auto-turret-triggers-component` | 357 |
| 12 | `i1 managed-navpoint-flags-component` | 366 |
| 0, 1, 2 | `i11 game-engine-soft-ceilings-component` | 170 |
| 19 | `i0 sound-placement-state-data-component` | 163 |
| 25 | `i0 powerframe-player-selection-data-component` | 163 |
| 26 | `i0 supply-lines-blocked-status-component` | 163 |
| 27 | `i0 supply-lines-item-unlocked-component` | 163 |
| 34 | `i10 tacmap-mapdismissallock` | 163 |
| 45 | `i0 matchflow-sequence-data-component` | 158 |
| 20 | `i2 spawn-filter-filters-component` | 46 |
| 11 | `i4 managed-objective-interaction-filter-component` | 24 |

**Neuf archétypes sont bloqués par leur PREMIER ou DEUXIÈME composant** (`i0` ou `i1` : ti=19,
25, 26, 27, 45, 12) : porter ces six déserialiseurs-là est le geste le moins cher du dossier.

**(2) LARGEUR FAUSSE — 38 756 records (61,8 %) marchent jusqu'au bout mais n'atterrissent pas.**
Ceux-là ont un cadre qui tient et une ou plusieurs largeurs de composant fausses. Le partage
sous-lecture / dépassement le dit : `ti=14` et `ti=17` dépassent SYSTÉMATIQUEMENT (5 024/5 024 et
5 379/5 379) — une largeur trop grande, pas un décrochage ; `ti=21` sous-lit systématiquement
(373/373).

### A.7 Le témoin positif `ti=9` : ce qu'il dit, et ce qu'il ne dit pas

Le contrôle écrit AVANT la mesure était « `ti=9` doit fermer à 100 % ». **Il ne ferme pas :
0/1 679.** Et le compte de désync dit pourquoi : **1 679 marches sur 1 679 s'arrêtent au MÊME
composant, `i4`**, dont le déserialiseur n'existe pas dans le dépôt. La marche a donc traversé
l'en-tête de 108 bits, `n1`, l'état par défaut de 14 bits, `n2`, puis les composants `i0` à `i3`
— exactement ce que la phase 4 avait dérivé — avant de buter sur un trou de couverture.

**Ce résultat ne réfute pas la phase 4 et ne la confirme pas non plus** : il est MUET sur le
cadre. Ce qui confirme le cadre, ce sont les cinq archétypes qui ferment à 100 % contre 0 % pour
les deux autres modèles.

---

## B. `n2` comme oracle de largeur d'état par défaut

### B.1 Les trois épreuves, écrites avant la mesure

`n1` et `n2` sont les deux tailles de tampon que le jeu alloue pour l'archétype
(`vtable[0x20]` et `vtable[0x10]` dans `FUN_1408f1aa4`) : constantes par archétype et par
build. `n1` se lit à une position FIXE (108) — sa constance teste l'en-tête. **`n2` se lit
APRÈS l'état par défaut — sa constance teste la LARGEUR de l'état par défaut.**

| épreuve | ce qu'elle fait |
|---|---|
| **A** la largeur portée | `n2` lu après le désérialiseur du dépôt. Constant => largeur indiscernable d'une juste |
| **B** le balayage | `n2` lu à `140 + w` pour w de 0 à 1 024 : les w qui rendent `n2` constant et crédible |
| **C** la fermeture | pour chaque candidat, le corps posé à `172 + w` atterrit-il sur le record suivant ? **Seconde chaîne, sans étape commune avec B** |

**Deux filtres sont indispensables, et chacun corrige une erreur de lecture mesurée :**

1. **Les ancres fortuites.** 3 à 15 records par groupe portent un `n1` différent du modal (en
   pratique 0) : ce ne sont pas des records de l'archétype mais des ancres retenues par hasard
   par le balayeur. Les garder rend tout balayage de largeur impossible. Retirés.
2. **Les zones de constance.** Une suite `w, w+1, w+2` dont `n2` DOUBLE (`v, 2v, 4v`) n'est pas
   une suite de candidats : c'est UNE zone de bits constante lue par une fenêtre glissante de
   32 bits. Sans ce repli, `ti=47` affichait 9 « candidats » là où il y en a 2.

### B.2 LES ÉTATS PAR DÉFAUT À CORRIGER — chacun confirmé par DEUX chaînes

Chaque ligne est mesurée sur le film ET relue dans l'exécutable (Ghidra, lecture seule,
2026-09-13). **Aucune n'est une largeur devinée.**

| ti | `vtable[0x60]` | porté aujourd'hui | largeur MESURÉE | grammaire relue dans l'exe | fermeture à cette largeur |
|---|---|---|---|---|---|
| **14** | `0x140FED6F4` | **0 bit** (absent de `defaultStateDeserByTI`) | **6 bits** (`n2 = 28`) | `FUN_1406cf008` = R(1) ; si 1 → R(8) ; puis **R(5)** | **864/864, 1 792/1 792, 2 368/2 368 = 100 % sur 3 builds** |
| **17** | `0x14101A0A4` | **0 bit** | **8 bits** (`n2 = 432`) | V ; puis **R(7)** | **891/891, 1 947/1 947, 2 541/2 541 = 100 % sur 3 builds** |
| **29** | `0x14116F514` | **0 bit** | **1 bit** | **V SEUL** (`R(1)` ; si 1 → R(8)), rien d'autre | **27/27, 56/56, 55/74 = 100 / 100 / 74,3 %** |
| **21** | `0x141133C24` | **0 bit** | **18 bits** (`n2 = 244`) | **PAS de préfixe de version** ; un seul **R(0x12) = R(18)** | 0/373 — la largeur est juste, un composant reste faux |
| **47** | `0x1410F44F8` | **0 bit** | **6 bits** (`n2 = 252`) | V ; puis **R(5)** | 0/1 679 — idem |

Les largeurs sont données avec le bit de version à 0, qui est sa valeur sur tout le corpus
(même constat que pour `ti=9` en phase 4). Les cinq désérialiseurs s'écrivent avec les
primitives DÉJÀ présentes dans `default_state_arch.go` :

```
14 : consumeVersionPrefix(br) ; br.ReadBits(5)
17 : consumeVersionPrefix(br) ; br.ReadBits(7)
21 : br.ReadBits(18)                              // PAS de prefixe de version
29 : consumeVersionPrefix(br)                     // meme forme que ti 11, 12, 20, 49
47 : consumeVersionPrefix(br) ; br.ReadBits(5)
```

### B.3 UNE CONTRADICTION DU DOSSIER, TRANCHÉE PAR LA MESURE

`default_state_arch.go` range `ti=14` parmi les STUBS (« + ti14 = FUN_140467a20, un `return;`
partagé »), alors que `KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md` lui donne
`vtable[0x60] = 0x140FED6F4`, classé REAL. **Les deux ne peuvent pas être vraies.** La mesure
tranche pour la table : `FUN_140FED6F4` consomme 6 bits, et poser ces 6 bits fait fermer
**5 024 records sur 5 024** sur trois builds. Le commentaire du code est à corriger dans le
même geste que le désérialiseur.

### B.4 CE QUE `n2` NE DIT PAS — et `ti=29` le montre

Sur `ti=29`, `n2` est CONSTANT à la largeur portée (0 bit, `n2 = 128`) — l'épreuve A passe.
Et pourtant la vraie largeur est **1 bit** : c'est la fermeture (épreuve C) qui l'a trouvée, et
le désassemblage qui l'a confirmée. **`n2` constant n'est donc pas une preuve que la largeur
est juste** : une zone de bits constante autour de la position rend plusieurs décalages
également « constants ». `n2` est un bon DÉTECTEUR (dispersé = faux à coup sûr) et un
MESUREUR seulement quand la fermeture le confirme. La note de phase 4 le présentait comme un
détecteur ; ce lot borne ce qu'il peut mesurer.

### B.5 Le bilan des 92 groupes (build × archétype)

| classe | groupes | ce que ça veut dire |
|---|---|---|
| `n2` constant à la largeur portée | **67** | largeur indiscernable d'une juste par cet oracle |
| ambigus (plusieurs zones) | **11** | la fermeture tranche pour 6 d'entre eux (`ti=14`, `ti=17` sur 3 builds) |
| muets (aucune largeur constante) | **14** | état par défaut de largeur VARIABLE, ou non résolu |

Les muets ne sont PAS tous des défauts : `ti=35`, `37`, `38`, `42`, `43` ont un `n2` constant
malgré une largeur portée variable (épreuve A passe), ce qui est le comportement attendu d'un
état par défaut à branches. **`ti=13` est le seul cas non résolu où la grammaire portée
correspond EXACTEMENT au décompilé** (V ; R(32) `propertyName` ; porte ; 1 ou 32 × R(4) — les
largeurs mesurées 46 et 170 le confirment au bit près) **et où `n2` reste dispersé.** Publié
comme ouvert, non traité.

---

## C. Ce que la production déraille aujourd'hui

### C.1 OÙ LA PRODUCTION PARSE-T-ELLE LE CORPS D'UN RECORD D'IMAGE-CLÉ ? PRESQUE NULLE PART

Relevé du 2026-09-13, et c'est le premier résultat de cet objectif :

| lecteur | statut | forme lue |
|---|---|---|
| `ScanNavpointRadial` (ti=12, `navpoint_radial_scan.go`) | **SEUL CHEMIN DE PRODUCTION** — appelé par `replay/bomb_armings.go:160`, la cuisson du rejeu (compte à rebours d'armement d'Assaut) | en-tête 64 bits + `TraverseEntity` |
| `ScanObjectives` (ti=11, `objective_scan.go`) | son propre en-tête le déclare « instrument de mesure, PAS une source de production » | idem |
| `WalkKeyframeRecords` (`keyframe_record_walk.go`) | **aucun appelant hors du paquet** | idem |
| `keyframe_loadout.go`, `keyframe_ground_weapons.go`, `keyframe_carrier_mark.go` | production | **ne parsent PAS le corps** : balayage de motifs d'octets dans l'emprise |
| `WalkKeyframeWorld`, `KeyframeRecordSpans` | production | **ancres et emprises seulement** |

**`KillSourceHealth` ne porte AUCUN compteur de record d'image-clé** : ses champs comptent des
candidats d'attribution de mort (`Candidates`, `UnexplainedPair/Self/BotIdx`) et sa
`CoverageRatio` vaut `DeathsCovered / DeathsReal`. **Il n'existe pas, dans le dépôt, de
compteur de production « la table d'image-clé a déraillé ».** Les seuls compteurs de cette
nature sont les `Key*` (`KeyRecords`, `KeyWalked`, `KeyBroken`, `KeyChained`) des deux
balayages ci-dessus. C'est un résultat, pas une lacune de la mesure.

### C.2 LES COMPTEURS DE PRODUCTION, TELS QUELS, SUR LES 6 FILMS

Instrument `imagecle_production_research_test.go`, qui APPELLE les deux balayages sans les
modifier :

| balayage | KeyRecords | KeyWalked | KeyBroken | KeyChained |
|---|---|---|---|---|
| `ScanFilmNavpointRadial` (ti=12, production) | **0** | 0 | 0 | 0 |
| `ScanFilmObjectives` (ti=11, instrument) | **27** | 27 | 0 | **0** |

Deux lectures, et il faut les deux :

1. **Le chemin de production ne s'engage pas sur ce corpus.** `ScanNavpointRadial` sort avant
   la boucle quand la bande de slots observés est vide (`len(band) == 0`) : aucun des six films
   n'est un Assaut, donc **0 record d'image-clé n'est parsé**. Les 366 records `ti=12` que
   porte la trame de ces films sont ce que ce chemin marcherait sur un film d'Assaut.
2. **Là où un balayage s'engage (`ti=11`), son propre témoin de santé vaut ZÉRO** : 27 marches
   « réussies », **0 chaînée**. Aucune ne retombe sur un en-tête valide. C'est le déraillement,
   dit par le compteur de production lui-même.

### C.3 LA MÊME POPULATION, LES MÊMES DÉFINITIONS, LES DEUX MODÈLES

Compteurs recalculés avec les définitions EXACTES de `scanKeyframe` (`KeyBroken` = désync ou
débordement ; `KeyChained` = la position d'arrivée porte un en-tête valide), plus la
**fermeture exacte**, que la production ne mesure pas :

| ti | modèle | records | marches | cassés | chaînés | **fermés** |
|---|---|---|---|---|---|---|
| 12 | **PRODUCTION** | 366 | 338 | 28 (7,7 %) | **7 (1,9 %)** | **0 (0,0 %)** |
| 12 | ÉTAT COMPLET | 366 | 0 | 366 (100 %) | 0 | 0 |
| 11 | **PRODUCTION** | 24 | 24 | 0 | **0 (0,0 %)** | **0 (0,0 %)** |
| 11 | ÉTAT COMPLET | 24 | 0 | 24 (100 %) | 0 | 0 |

**`KeyChained` est la définition FAIBLE de la santé** (un en-tête valide à l'arrivée) et la
production plafonne déjà à 1,9 % dessus. Sur la définition FORTE — atterrir au bon bit — elle
est à **0 sur 390**.

**ET LA BONNE FORME NE RÉCUPÈRE RIEN SUR CES DEUX ARCHÉTYPES-LÀ**, il faut le dire aussi net :
elle désynchronise à 100 % parce que leurs composants ne sont pas portés (`ti=12` bute sur
`i1 managed-navpoint-flags-component`, `ti=11` sur `i4 managed-objective-interaction-filter-component`).
**Le cadre juste ne remplace pas les déserialiseurs manquants.**

### C.4 L'ESTIMATION GLOBALE, ET CE QUE LE CORRECTIF RÉCUPÉRERAIT

| grandeur | valeur |
|---|---|
| records d'image-clé bornés, 6 films | **62 686** |
| que la production ferme aujourd'hui | **0 (0,0 %)** |
| que la bonne forme ferme, à couverture de composants INCHANGÉE | **8 796 (14,0 %)** |
| ce que le plancher de hasard donnerait | 529 (0,8 %) |
| gain net, hors plancher (`ti=4`, `6`, `15`, `18`, `22`) | **8 421** |
| **en ajoutant les trois largeurs d'état par défaut mesurées en B.2** (`ti=14` +5 024, `ti=17` +5 379, `ti=29` +138) | **19 337 / 62 686 = 30,8 %** |

Les 10 541 records supplémentaires ne coûtent que **trois entrées dans
`defaultStateDeserByTI`**, chacune écrite avec des primitives déjà présentes, chacune
confirmée par deux chaînes indépendantes.

---

## D. Prouvé / hypothèse / réfuté

### Prouvé (deux chaînes sans étape commune, ou témoin de hasard mesuré)

1. **Le modèle de record d'image-clé de la production ne ferme AUCUN record.** 0 sur 62 686,
   sur 6 films et 3 builds, avec le code de production lui-même (`readKeyframeHeader` +
   `walkOneKeyframeRecord`). Ce n'est pas un taux faible, c'est un compte nul.
2. **La forme d'ÉTAT COMPLET ferme 8 796 records sur 62 686 (14,0 %), au-dessus d'un plancher
   de hasard MESURÉ à 0,8 %** (le même lecteur, en-tête décalé d'un bit). Le plancher n'est pas
   calculé : il est mesuré sur le flux réel, comme la règle 4 de la méthode l'impose.
3. **Cinq archétypes ferment à un taux quasi parfait sous la bonne forme et à ZÉRO sous les
   deux autres modèles** : `ti=6` 7 820/7 822, `ti=15` 163/163, `ti=18` 163/163, `ti=22`
   163/163, `ti=4` 112/157. 8 421 records, plancher de hasard à 0.
4. **L'état par défaut de `ti=14` fait 6 bits.** Chaîne 1 : `n2` devient constant (= 28) à cette
   largeur, et la marche ferme 864/864, 1 792/1 792 et 2 368/2 368 sur trois builds. Chaîne 2 :
   `FUN_140FED6F4` décompilé — `FUN_1406cf008` = R(1), si 1 → R(8), puis R(5).
5. **L'état par défaut de `ti=17` fait 8 bits.** Chaîne 1 : `n2 = 432` constant, fermeture
   891/891, 1 947/1 947, 2 541/2 541. Chaîne 2 : `FUN_14101A0A4` = V puis R(7).
6. **L'état par défaut de `ti=29` fait 1 bit.** Chaîne 1 : fermeture 27/27, 56/56, 55/74.
   Chaîne 2 : `FUN_14116F514` ne contient QUE le préfixe de version.
7. **L'état par défaut de `ti=21` fait 18 bits et n'a PAS de préfixe de version.** Chaîne 1 :
   `n2 = 244` constant à w=18. Chaîne 2 : `FUN_141133C24` = un unique `R(0x12)`.
8. **L'état par défaut de `ti=47` fait 6 bits.** Chaîne 1 : `n2 = 252` constant à w=6.
   Chaîne 2 : `FUN_1410F44F8` = V puis R(5).
9. **`ti=14` est classé STUB à tort dans `default_state_arch.go`** : la table
   `KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md` a raison (`0x140FED6F4`, REAL), et la mesure
   comme le décompilé le confirment.
10. **Aucun archétype ne régresse sous la bonne forme** : 8 gagnent, 0 perdent, 25 sont à
    égalité (toutes à 0/0).
11. **Il n'existe aucun compteur de production « la table d'image-clé a déraillé » hors les
    `Key*` de deux balayages.** `KillSourceHealth` compte des candidats d'attribution de mort ;
    `WalkKeyframeRecords` n'a aucun appelant hors du paquet ; les autres lecteurs d'image-clé de
    la production balaient des motifs d'octets au lieu de parser le corps.

### Hypothèse (une seule chaîne, ou non tranché)

1. **`ti=4` dépend du build** : 74/74 sur `HI_1_13_0`, 38/57 sur `HI_1_12_0`, 0/26 sur
   `HI_1_11_0` (sous-lecture pure). Lecture la plus économique : son état par défaut ou un de
   ses composants a changé entre `HI_1_11_0` et `HI_1_12_0`. Non mesuré.
2. **`ti=38` et `ti=37` : rien.** 1,7 % contre 1,7 % au témoin décalé pour `ti=38` ; 0,0 %
   contre 0,8 % pour `ti=37`. Ces archétypes ne disent RIEN dans un sens ni dans l'autre.
3. **`ti=13` reste muet alors que sa grammaire portée correspond au décompilé au bit près**
   (largeurs 46 et 170 mesurées, exactement V(9) + 32 + 1 + 4 et V(9) + 32 + 1 + 128). Soit
   `FUN_14080dec4` ou `FUN_140CE59BC` a une largeur dépendante de la donnée, soit `n2` n'est pas
   à un décalage fixe pour cet archétype. Non tranché.
4. **Le reliquat de sous-lecture / dépassement (38 756 records, 61,8 %)** est attribué à des
   largeurs de composant fausses parce que la marche va au bout sans désync. C'est la lecture la
   plus économique, pas une mesure : aucun composant n'a été isolé.

### Réfuté

1. **« Le modèle de la production déraille surtout à cause des composants non portés. »**
   RÉFUTÉ : sous la production, **3 477 records sur 62 686 (5,5 %) seulement désynchronisent** —
   les 57 849 autres marchent JUSQU'AU BOUT et atterrissent au mauvais bit. Le problème est le
   CADRE, pas la couverture : le masque de présence fait croire que presque aucun composant
   n'est présent, la marche se termine trop tôt, proprement, et faux.
2. **« `ti=9` doit fermer à 100 % sous la bonne forme » (le témoin positif écrit avant la
   mesure).** RÉFUTÉ comme formulé : 0/1 679. Mais la cause est nommée et n'est pas le cadre —
   les 1 679 marches butent toutes sur `i4 managed-player-forge-weather-effect-overrides-component`,
   non porté. La dérivation de la phase 4 (premier composant à 186) n'est pas touchée.
3. **« `n2` constant prouve que la largeur d'état par défaut est juste. »** RÉFUTÉ par `ti=29` :
   `n2` y est constant à la largeur portée (0 bit) alors que la vraie largeur est 1 bit
   (fermeture + décompilé). `n2` dispersé prouve qu'une largeur est fausse ; `n2` constant ne
   prouve rien tout seul.
4. **« Corriger le cadre suffirait à débloquer les archétypes que la production lit
   réellement. »** RÉFUTÉ : `ti=11` et `ti=12` désynchronisent à 100 % sous la bonne forme,
   sur leur deuxième et cinquième composant. Le cadre juste ne remplace pas les déserialiseurs
   manquants.

---

## E. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautés, donc la CI
reste verte. Les chemins sont **en style Windows** (`C:/...`) : un chemin de style Git Bash
(`/c/...`) fait échouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"
F="000d5950 00162144 00502e52 0014603f 02784ce1 00ba2e1c"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"

# A : le tableau archetype x modele (3,1 s)
CGO_ENABLED=0 CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestImageCleFermetureParArchetype' -v -timeout 30m

# A, avec les temoins T+ / T- en ECHEC DUR plutot qu'en publication.
# ATTENDU AUJOURD'HUI : T- PASSE (la production ferme 0 contre 8 796) et T+ ECHOUE en nommant
# sa cause (ti=9 bute sur i4 managed-player-forge-weather-effect-overrides-component, 1 679 fois).
# C'est pour cela que le mode dur est OPT-IN : l'instrument PUBLIE par defaut.
CGO_ENABLED=0 CHUNK00_FILMS="$L" CHUNK00_TEMOINS_DURS=1 go test ./internal/analysis/filmdec/ \
  -run 'TestImageCleFermetureParArchetype' -v -timeout 30m

# B : n2 comme oracle de largeur d'etat par defaut (3,0 s)
CGO_ENABLED=0 CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestImageCleOracleN2' -v -timeout 30m

# C : les compteurs de production (21 s — les deux balayages rechargent les films)
CGO_ENABLED=0 CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestImageCleProduction' -v -timeout 30m
```

Lecture des cellules du tableau A : `fermes/total  %  dN sN uN` — `d` désync (composant non
porté), `s` sous-lecture, `u` dépassement.

Gates passés le 2026-09-13 : `gofmt -l internal/analysis/filmdec/` net,
`go vet ./internal/analysis/filmdec/` net, `go test ./internal/analysis/filmdec/` **ok** (sans
garde : les trois instruments sont sautés), `go test ./internal/archlint/` **ok** — le ratchet
des variables de paquet de `filmdec` n'est pas touché (aucune variable de niveau paquet
ajoutée : seulement des `const`, des types, des méthodes et des fonctions de test).

---

## F. Découvertes hors périmètre — notées, NON TRAITÉES (règle 7)

1. **Le décalage de 8 octets de `parseRegistry`** : connu depuis la phase 1, **NON TRANCHÉ**,
   pas touché par ce lot.
2. **Six archétypes sont bloqués par leur PREMIER ou DEUXIÈME composant** (`ti=19`, `25`, `26`,
   `27`, `45` sur `i0` ; `ti=12` sur `i1`), soit 1 176 records qui ne vont pas au-delà du
   deuxième composant. Écrire ces six déserialiseurs est le geste le moins cher du dossier —
   **sans garantie de fermeture pour autant** : rien ne dit que les composants suivants sont
   justes. Non traité.
3. **`ti=14` et `ti=17` dépassent SYSTÉMATIQUEMENT sous la largeur portée** (5 024/5 024 et
   5 379/5 379 en dépassement) : c'est la signature d'une largeur d'état par défaut manquante,
   et la section B l'a confirmée. Le même symptôme sur `ti=21` (373/373 en sous-lecture)
   pointe le même diagnostic mais la fermeture n'y suit pas — donc un composant reste faux.
4. **Le film `02784ce1` et `0014603f` déclenchent l'avertissement « empreinte du registre ECS du
   film INCONNUE »** au chargement (`ScanFilmObjectives`). Sans effet sur ces mesures (les
   noms de composant sont lus du film), mais l'empreinte connue du dépôt ne couvre pas
   `HI_1_12_0`. Non traité.
5. **La production ne parse le corps d'un record d'image-clé QUE pour l'armement d'Assaut**
   (`replay/bomb_armings.go` → `ScanNavpointRadial`). Tout le reste du rejeu lit les
   images-clés par balayage de motifs. C'est une information de cadrage pour un lot de
   production : le correctif de forme ne change aujourd'hui la sortie d'AUCUNE page — il ouvre
   une voie, il n'en répare pas une.

---

## Ce qui reste ouvert

1. **Le lot de production.** Trois entrées dans `defaultStateDeserByTI` (`ti=14`, `17`, `29`),
   deux de plus qui ne ferment pas encore mais dont la largeur est prouvée (`ti=21`, `47`), le
   commentaire STUB de `ti=14` à corriger, et le branchement de `WalkKeyframeFullState` à la
   place de `TraverseEntity` dans `navpoint_radial_scan.go` / `objective_scan.go`. **Ce lot n'a
   rien branché : il chiffre.**
2. **Ce que le correctif rendrait LISIBLE.** Le gain est mesuré en records fermés, pas en
   champs publiés. Aucune mesure de ce lot ne dit quelles valeurs deviennent exploitables — il
   faudrait, archétype par archétype, confronter les champs lus à un oracle (celui de la
   phase 3 pour l'équipe à 186 en est un).
3. **`ti=13`**, le seul archétype dont la grammaire portée est bit-exacte avec le décompilé et
   dont `n2` reste dispersé.
4. **Le reliquat de 61,8 %** (sous-lecture / dépassement) : aucun composant n'a été isolé comme
   coupable. La méthode existe (histogramme du composant pendant lequel la frontière est
   franchie, `kf35Break` du lot R7) ; elle n'a pas été appliquée par archétype ici.
5. **Les quatre builds non essayés** : `HI_1_4_1`, `HI_1_8_0`, `HI_1_9_0`, `HI_1_10_0`. La
   mesure tient sur trois builds consécutifs ; rien ne dit qu'elle tient sur les anciens, où la
   phase 4 a déjà montré que `n2` vaut 88 et non 136 pour `ti=9`.
