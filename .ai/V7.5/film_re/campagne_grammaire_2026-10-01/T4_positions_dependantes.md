# T4 — Grammaire de position « dépendante du build ou du contenu » : les 13 exceptions datées (2026-10-01)

> Campagne grammaire, phase 1, étape 2, piste T4. Plan : `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`.
> Code lu à la tête du worktree `feat/campagne-grammaire` (`69564ef7d`). Jeu lu dans Ghidra
> (`HaloInfinite.exe`, build `hi_1_13_0`, base `0x140000000`, serveur HTTP `127.0.0.1:8089`) en
> LECTURE SEULE : `decompile_function`, `disassemble_function`, `get_xrefs_to`,
> `get_function_callers`, `read_memory`, `search_strings`. Rien n'a été écrit dans Ghidra (pas de
> `disassemble_bytes`, qui crée des instructions). Aucune commande `go`, aucun film décodé, aucune base
> ouverte.
>
> Remarque de compte : la table `exceptionsDuPortage` (`grammar/lecteur_position_ratchet_test.go`)
> porte **13** clés, pas 14 (la 14e « exception » citée par le brief est sans doute la seconde
> branche de `unit-actor-state`, deux CALLs, une seule clé).

## 0. Verdict

1. **Dans l'exécutable disponible (HI_1_13_0), aucun des 13 sites n'a de grammaire qui dépende du
   build, du niveau du registre, d'un drapeau d'objet, du bit precHigh hors flux ou d'une version de
   format.** Le lecteur ne dépend que des bits du flux et des TABLES DE LA CARTE (`DAT_144632be0`, table
   par index) ; la table défaut est une constante du build (±20000). Les ÉCRIVAINS des 13 composants
   (case `+0x18` de leur vtable) ont été ouverts : ils sont SYMÉTRIQUES de leurs lecteurs, au même
   niveau `0x10`. Les paramètres `p4` et `p5` de `FUN_14076e494` (où arrivent le niveau du registre et
   l'octet `*(param_3+0x38)`) sont MORTS (désassemblage). **Le critère de retrait « la grammaire
   dépendante du build est établie » est donc inatteignable par Ghidra pour HI_1_13_0 : la grammaire
   y est établie, et c'est celle du portage unique.**
2. **Sept exceptions perdent des paquets sur un film HI_1_13_0** (`ti38-i18` fb1a1a72,
   `displayasset` et `areaofinterest` 51ebbc0f, `cooptetherarea` c75f33b8, `i0-bipede-prechigh`
   0797ce72, `unit-actor-state` 4f77afc1, `waypointstate` d9781168). Pour celles-là, aucune règle du
   jeu ne peut justifier l'ancien lecteur : les fermetures qu'il obtient sont des COMPENSATIONS ou des
   ATTRIBUTIONS fausses en amont (le décalage de masque est exclu par la note T2), pas une grammaire.
3. **Les fermetures de l'ancien lecteur ne sont reproduites par AUCUNE forme du lecteur du jeu**,
   à aucun niveau, sur les cartes concernées, dès que la porte vaut 0 (index 1 bit + `6+L` par axe ne
   correspond à aucune ligne de la table par index de Thunderhead, Fragmentation ou des canevas Forge).
   Sur les branches porte posée et precHigh = 1, l'ancien lecteur coïncide EXACTEMENT avec
   `FUN_14076e420(NIVEAU = niveau du registre)` (loi : `6+L` sur ±20000, **0 bit** sur ±100 pour
   L <= 2) : c'est l'origine probable de sa forme, mais elle ne tient que pour 3 des 6 cas chiffrés
   (§5). Une « grammaire dépendante du build » de cette famille n'est donc pas établie, et elle est
   réfutée pour `tacmap-poiicon` et `player-desired-respawn-location`.
4. Deux écarts RÉELS sortent de la lecture des écrivains : (a) `flock-destination` et
   `tacmap-waypointstate` écrivent leur queue (R(2), R(1)) SANS condition, les lecteurs la lisent si
   `param_4 > 1` (seule dépendance de contenu légitime : la colonne niveau du registre du film) ;
   (b) le Go lit les largeurs de la plage CATALOGUÉE pour TOUT index >= 0, le jeu lit la ligne de
   l'index lu (Live Fire : 4 plages déclarées).

## 1. Ce que lit le jeu — pièces

### 1.1 Les enveloppes et le lecteur nu (re-lus ce jour)

`FUN_14076e494` (désassemblage complet) :

```
14076e49a: MOV R9D,R8D        ; le NIVEAU (3e arg) devient le 4e arg de e524 : p4 (R9) est ÉCRASÉ
14076e4a0: CALL 0x14076f91c   ; garde, sans argument utile
14076e4b1: MOV R8,[RSP+0x68]  ; p6 (6e arg) seul lu sur la pile ; p5 ([RSP+0x60]) jamais lu
14076e4c0: CALL 0x14076e524   ; p6 == 0
14076e4dc: CALL 0x1411b259c   ; garde vraie : R(96)
14076e4e3: CALL 0x141f85880   ; p6 != 0 : bornes
```

Donc ni le niveau du registre (`param_4` des désérialiseurs, que certains sites passent en p4), ni
l'octet de contexte `*(param_3+0x38)` (que `FUN_142f036f0`, `FUN_142f03ec8`, `FUN_142ed9120` et
`FUN_14058c058` passent en p5) n'atteignent la lecture.

`FUN_14076f91c` : `return DAT_144e61ea0 != 0 || DAT_145121140 == 1` (0 bit).
- `DAT_144e61ea0` : écrit seulement par huit fonctions `142e2*`/`142e3*` qui le lèvent autour d'un
  appel de vtable (ex. `FUN_142e2d08c` : `DAT_144e61ea0 = 1; (*vtable[0x58])(...); DAT_144e61ea0 = 0`).
  Les lots R7-c/R7-e ont mesuré que le payload du film est écrit HORS de cette portée.
- `DAT_145121140` : mode de processus posé par `FUN_140a93ec8(mode)` (`1 -> FUN_142b5c658 -> 1`,
  `2 -> 2`, sinon `3`, `0 -> 0`), appelé à l'initialisation du moteur (`FUN_140a938b4`). Ce n'est pas
  une donnée du film.

`FUN_14076e524` (décompilation complète relue) : porte R(1) ; à 0, index sur `DAT_144632be0` bits ;
index != -1 -> bornes `DAT_14462cbe0 + idx*0x18`, largeurs `DAT_1445ccbe0 + (idx*0x20 + NIVEAU)*0xc` ;
sinon bornes `DAT_1445cc9c8`, largeurs `DAT_1445cc9e0 + NIVEAU*0xc` ; puis `FUN_140cc5128` (3 axes).
**Aucune autre branche.** Les lignes d'un index non déclaré ou non valide restent à ZÉRO (mise à zéro
par `FUN_140365eb0` / `FUN_140de9a1c`, remplissage par `FUN_140be9a14` des seules plages valides,
note 3.4).

### 1.2 Les écrivains (case `+0x18`) — symétriques

Table nom -> vtable obtenue par la méthode statique (nom ASCII -> xref DATA = getter -> xref au getter
= vtable+0x08). Case `+0x00` = `param_4` constant de l'exe ; `+0x18` écrit ; `+0x28` lit (sauf thunk
`FUN_14076ce9c` -> `+0x30`).

| Composant | vtable+8 | `+0x00` (param_4) | écrivain `+0x18` | lecteur |
|---|---|---|---|---|
| flock-position | `143d07e10` | `1405f0ac0` = `XOR EAX,EAX` (0) | `142eda580` : `JMP 1407eb600` (`R9D=0`, `R8D=0x10`) | `140ee7270` |
| flock-destination | `143c96c50` | `141179610` = `MOV EAX,2` | `142eda4b4` : bit, `1407eb600(0x10, R9D=0)`, **`142af29b0` = W(2) sans condition** | `140fb8af0` (R(2) si `param_4 > 1`) |
| player-desired-respawn-location | `143d0f2f8` | `14117b4a0` = 1 | `142f07f18` : bit ; si 1 : `1407eb600(0x10, R9D=0)` + `1407eb9bc` | `142f03ec8` |
| generic-rigid-body-transforms | `143d0c898` | 1 | `142f06958` : W(8) masque ; par bit `140c1c6d0` + `1407eb61c(pos, indice=*(short*)(obj+0x12), 0x10, 0)` | `142f036f0` |
| tacmap-areaofinterest | `143d07060` | 1 | `142ed9950` -> `142ede58c` : `141f860b0` | `142ed3c50` -> `142ed7764` |
| tacmap-displayasset | `143d07330` | 1 | `142ed9f0c` -> `142edeb40` : `141f860b0` | `142ed433c` -> `142ed7d38` |
| tacmap-cooptetherarea | `143d074c0` | 1 | `142ed9de4` -> `142ede980` : `141f860b0` | `142ed4198` |
| tacmap-waypointstate | `143d07380` | `141179610` = 2 | `142ede52c` : bit, W(32), `141f860b0`, **`JMP 1406d49c4` = W(1) sans condition** | `140f04d74` -> `140f04d88` (R(1) si `param_4 > 1`) |
| tacmap-poiicon | `143d06b50` | 1 | `142eda71c` -> `142edf1c4` : ..., `141f860b0`, ... | **`142ed4834`** -> `142ed8418` |
| crew-order | `143d08768` | 1 | `142ed9e3c` : `142b1d01c`, bit ; si 1 : `1407eb600(0x10, R9D=0)` | `142ed4274` -> `142ed9120` |
| unit-actor-state | `143d06dd8` | `140c85020` = `MOV EAX,4` | `142edde78` -> `1427e2e84` : `1407eb600(0x10, R9D=0)` x2 | `14058bcf4` -> `14058c058` |
| object-position (world-object) | `143d0bed8` | 1 | `1432067d4` | `+0x30` = `14076e29c` |
| object-position-dynamic-precision (i0 bipède) | `143e2bbd0` | 1 | `+0x18` `14320678c` (complet), `+0x20` `143206a88` (delta) | `+0x28` = `1406cfe44` (delta), `+0x30` = `14076e29c` (complet) |

- `141f860b0` = `FUN_1407eb61c(w, pos, -1, 0x10, 0)` (`MOV R9D,0x10`, `AND [RSP+0x20],0`).
- `1407eb600(w, pos, NIVEAU, bornes)` = `FUN_1407eb61c(w, pos, -1, NIVEAU, bornes)` (le 4e argument
  part en `[RSP+0x20]`) ; tous les sites listés posent `XOR R9D,R9D` : bornes nulles.
- `FUN_1407eb61c` : garde `f91c` -> W(96) (`1406d60f4`) ; sinon bornes nulles ->
  `FUN_140770640(pos, indice, NIVEAU)` puis `FUN_1407eb6a8` (W(1) = index == -1, index sur
  `DAT_144632be0` bits, axes) ; bornes non nulles -> `FUN_142e2d818` (loi sur les bornes, 3 axes).
- `FUN_140770640` : l'indice proposé n'est retenu que si sa plage est VALIDE (champ de bits
  `DAT_1445ccb60`) ET contient le point ; sinon `FUN_14077084c` cherche une plage valide qui le
  contient, ou rend -1. Largeurs : ligne NIVEAU de la table de l'index, ou de la table défaut.
- i0 bipède, branche absolue (`FUN_142e2d86c`) : W(1) precHigh puis
  `JMP 1407eb61c(..., 0x10, bornes = precHigh ? &DAT_143b8c6d0 : 0)` (`NEG R11B ; SBB ; AND`) :
  precHigh = 1 -> 3 x 14 = 42 bits sur ±100, symétrique de `FUN_141f85880`.

**Écriture et lecture ne divergent qu'en deux points, tous deux des QUEUES** : `flock-destination`
(W(2) toujours, R(2) si `param_4 > 1`) et `tacmap-waypointstate` (W(1) toujours, R(1) si
`param_4 > 1`). Avec le niveau 2 de l'exe, les deux côtés concordent ; la seule dépendance de
contenu possible est un film dont le registre (`entrée + 0x100`) porterait 0 ou 1 pour ces
composants.

### 1.3 world-object i0 (`FUN_14076e29c`, désassemblé)

```
14076e2c0: CALL 14076e420(lecteur, dst+4, 0x10)   ; precHigh R(1), puis e494(0x10, p6 = precHigh ? ±100 : 0)
14076e2c5: MOV R9B,AL                               ; precHigh -> 4e arg de la queue de poignée
14076e2ce: CALL 14076e3e4                           ; si precHigh : FUN_1408f0ac4 + R(1) [+ suite] ; sinon RIEN
14076e2d7: CALL 140492128                           ; position finie ? (exposant != 0xff sur x, y, z)
14076e2e7: CALL 14076e304                           ; R(2) seulement si finie
```

Le R(2) est donc conditionné par la FINITUDE de la position décodée : toujours vrai pour une
position quantifiée (bornes finies) ; seule la garde R(96) peut produire un NaN. Le Go le lit sans
condition — juste en film.

### 1.4 Aucun lecteur ne consulte une version de format

Relevé des appels et globales de `140ee7270`, `140fb8af0`, `142ed8418`, `142f03ec8`, `142ed3c50`,
`142ed7764`, `142ed4198`, `142ed4274`, `142ed9120`, `140f04d74`, `140f04d88`, `14076e29c` : seuls
`FUN_14076f91c`, `e524`/`e494`/`1424e0e38`, des lectures fixes (`1406cf008`, `14080dec4`,
`1406d84b4` à largeur fixe sur pile, `1424e268c` = R(2), `14076dc04` = R(19), `142b1cf3c`) et
`PTR_DAT_14474c308` (vecteur par défaut). Aucune référence à `DAT_145121140`, à `&DAT_144c23178`
(structure du film chargé) ni à un numéro de version.

## 2. Ce que fait le Go

- Portage unique (`lecteur_position.go`) : `lireE524Sur` lit la porte, l'index sur
  `tablesDuProfil().indexW` (= `DAT_144632be0` de la carte), puis `largeursDeLaLigne` : défaut
  `LargeursAxeParDefautDuBuild(L)` si idx < 0 ; **`t.axesCarte` pour TOUT idx >= 0 au niveau 0x10**.
  C'est le seul écart du portage au jeu (§4, constat C3).
- Exceptions (`lecteur_position_exceptions.go`) : trois familles de formes anciennes.
  - `lireVecteurAncienAuNiveauDuRegistre` (crew-order, poiicon, flock-destination, respawn) :
    precHigh R(1) -> 0 bit ; porte ; index FIGÉ à 1 bit ; axes `min(26, 6 + niveau du registre)`.
  - `lireCorpsDeTraverseeAncien` (tacmap x4, ti38-i18) : porte ; index et axes du descripteur de
    TRAVERSÉE (1 bit, 6/6/6) ; sans garde.
  - formes isolées : world-object (largeurs de carte quelle que soit la porte), flock-position (garde,
    porte, index 1 bit, `6+niveau`), i0 bipède precHigh = 1 (0 bit), unit-actor-state (R(16) plat),
    waypointstate (sans le R(1) de queue).
- `level` = `Archetype.Level(i)` = colonne niveau du registre du film (`component_param4.go`) ; égal à
  la constante `+0x00` de l'exe pour les composants contrôlés par `TestParam4RegistreParBuild`
  (seuls écarts par build connus : `ti=40 i2`, `biped-malleable-property`).

## 3. Les 13 exceptions, chiffres du dépôt, rangés par build

Builds : a521164d HI_1_4_1 ; 60ae07c4 HI_1_8_0 (Live Fire) ; 11de8353 HI_1_9_0 (Thunderhead, Forge) ;
084a804d HI_1_10_0 (Fortitude, Forge) ; 111fa685 HI_1_10_0 (Command, Forge) ; e5adf7b2 HI_1_11_0
(Fragmentation) ; bcb6d393 HI_1_12_0 (Cliffhanger) ; fb1a1a72, 51ebbc0f (Banished Narrows, Forge),
c75f33b8 (Curfew, Forge), d9781168 (Dredge, Forge), 4f77afc1 (Flood Gulch, Forge), 0797ce72 (Live
Fire), 000d5950 (Cliffhanger/Catalyst) : HI_1_13_0.

| Exception | Pertes (lecture du jeu) | Gains (lecture du jeu) | Perte sur HI_1_13_0 ? | Lecture |
|---|---|---|---|---|
| flock-position | e5adf7b2 (1_11) 1 paquet | 000d5950 (1_13) +24 paquets, ti=21 44->84, ti=35 3549->3638 | non | gain net massif ; perte au niveau du bruit |
| world-object-i0 | 11de8353 (1_9) 99->83 et 3->2 ; a521164d (1_4_1) 122->119 ; 60ae07c4 (1_8) ti=42 5->4 ; 111fa685 (1_10) 2->1 | bcb6d393 (1_12) 134->713, 5->70 ; fb1a1a72 (1_13) 317->349, 9->25 ; 60ae07c4 (1_8) ti=38 30->46 ; e5adf7b2 (1_11) 49->56 | non | non monotone en build (1_8 dans les DEUX sens) : pas une règle de build |
| ti38-i18 | fb1a1a72 (1_13) 317->245 ; 111fa685 (1_10) 72->30 ; 11de8353 (1_9) 99->19 | aucun | **oui** | contredit l'exe 1_13 (§1.2 symétrique) |
| flock-destination | 11de8353 (1_9) 2 listes | 000d5950 (1_13) 1 liste | non | candidat build <= 1_9 (1 film) |
| respawn-location | e5adf7b2 (1_11) 1 liste | e5adf7b2 (1_11) 1 liste ; 111fa685 (1_10) 1 liste | non | même film dans les deux sens : pas une règle de build |
| tacmap-poiicon | 11de8353 (1_9) 1 liste | aucun | non | 1 liste, 1 film |
| displayasset | 51ebbc0f (1_13) 10 paquets | 51ebbc0f (1_13) 14:42 ; 084a804d (1_10) ; 11de8353 (1_9) ; fb1a1a72 (1_13) ; 60ae07c4 (1_8) | **oui** | même film dans les deux sens ; contredit l'exe |
| areaofinterest | 51ebbc0f (1_13) 1 paquet | 11de8353 (1_9) ; fb1a1a72 (1_13) ; 60ae07c4 (1_8) | **oui** | contredit l'exe |
| cooptetherarea | c75f33b8 (1_13) 1 liste | aucun | **oui** | contredit l'exe |
| crew-order | 084a804d (1_10) 1 liste | e5adf7b2 (1_11) 1 liste | non | 1 point de chaque côté |
| i0-bipede-prechigh | 0797ce72 (1_13) 1 liste ; 084a804d (1_10) 2 listes | aucun | **oui** | contredit l'exe (écrivain `142e2d86c` : 42 bits) |
| unit-actor-state | 4f77afc1 (1_13) 4 listes + chaîne 12:1118..1128 | 084a804d (1_10) 6 ; e5adf7b2 (1_11) 2 ; 111fa685 ; 4f77afc1 (1_13) 48:776 ; d9781168 (1_13) | **oui** | même film dans les deux sens ; contredit l'exe |
| waypointstate | d9781168 (1_13) 1 liste | aucun | **oui** | contredit l'exe (écrivain : R(1) TOUJOURS écrit) |

## 4. Constats

### C1 — ÉTABLI (jeu + code), négatif : aucune grammaire de position dépendante du build dans HI_1_13_0

Preuves jeu : §1.1 à §1.4. Preuves Go : `lecteur_position.go` (`lireE524Sur`, `lireE494`,
`lireE420`, `lireF85880`) reproduit `e524`/`e494`/`e420`/`f85880` ; `lecteur_position_exceptions.go`
garde les anciennes formes.

Écart : le Go garde, pour 7 sites, une forme que l'exe du build de leurs films (HI_1_13_0) contredit
à l'écriture comme à la lecture.

Conséquence : le critère `critereDeRetraitDesExceptions` (« ... ou la grammaire dépendante du build
est établie ») ne peut pas tomber par Ghidra ; sa première moitié (« monter sans AUCUNE baisse »)
interdit toute migration tant qu'une seule liste fermée par compensation existe. Proposer à
l'utilisateur de le remplacer par un critère MESURÉ hors fermeture (§6) ; à défaut, statuer par build :
lecture du jeu imposée sur HI_1_13_0 (exe lu), exception conservée sur les seuls builds où la mesure
§6 la soutient.

### C2 — ÉTABLI : les queues de `flock-destination` et `tacmap-waypointstate` sont écrites sans condition

Jeu : `FUN_142eda4b4` se termine par `FUN_142af29b0` = W(2) (`MOV ECX,0x2` en `142af29c7`) ;
`FUN_142ede52c` se termine par `JMP 0x1406d49c4` (`142ede585`) = W(1). Lecteurs : `FUN_140fb8af0`
`if (1 < param_4) FUN_1424e268c` ; `FUN_140f04d88` `if (1 < param_4) R(1)`. Exe : `param_4` = 2
(`141179610` = `b8 02 00 00 00 c3`).

Go : `consumeTacmapWaypointState` (exception) ne lit PAS le R(1) ; `consumeFlockDestination` le lit si
`level > 1`. L'ancienne forme de waypointstate est donc fausse de 1 bit en plus de la position, sur
tout film dont le registre porte 2 (l'exe ne connaît que 2).

Effet attendu : sur d9781168 34:336 (liste perdue), il faut les DEUX écarts (+29 position, +1
queue) : l'ancien lecteur ne peut fermer cette liste que par compensation d'exactement 30 bits.

### C3 — ÉTABLI côté jeu et code, effet à mesurer : le Go lit les largeurs de la plage cataloguée pour TOUT index

Jeu : `e524` lit la ligne `(idx*0x20 + NIVEAU)` de l'index LU ; l'écrivain (`FUN_140770640`,
`FUN_14077084c`) n'émet qu'un index de plage VALIDE qui contient le point. Go :
`largeursDeLaLigne` rend `t.axesCarte` pour tout `idx >= 0` au niveau 0x10.

Écart : sur une carte à plusieurs plages valides (Live Fire : 4 déclarées, `regionIndexBits = 2`,
`region = 1`), une position d'une autre plage est lue aux largeurs de la plage 1 ; sur une carte à
une seule plage valide, un index != plage cataloguée est IMPOSSIBLE chez l'écrivain — sa lecture est
un témoin gratuit de cadrage faux (comme les bits de masque hors archétype de la note T2), que le Go
consomme aujourd'hui comme une position.

Correctif proposé (phase 2, change une sortie seulement sur Live Fire) : porter dans
`tablesDePosition` les bornes de chaque plage déclarée (donnée de carte, catalogue étendu) et lire
la ligne de l'index ; un index hors des plages valides connues -> arrêt typé « index de plage
impossible » (compteur), jamais une lecture aux largeurs de la carte.

### C4 — HYPOTHÈSE réfutée en partie : l'ancien lecteur est `FUN_14076e420(NIVEAU = niveau du registre)` sur deux branches seulement

Loi (`FUN_140be9b88`, `profile.LargeursAxeDuNiveau`) : table défaut (±20000) au niveau L <= 16 =
`6 + L` par axe ; bornes ±100 au niveau L <= 2 = **0 bit** par axe (12000 / 2^(16-L) < 1 casier).
L'ancien `lireVecteurAncienAuNiveauDuRegistre` (precHigh -> 0 bit ; porte posée -> `6 + L`) coïncide
donc EXACTEMENT avec `e420(L = param_4)` sur ces deux branches. Sur la branche porte à 0, il lit
index 1 bit + `6 + L` ; `e420(L)` y lirait la ligne L de la plage : pour les étendues des cartes en
cause (Forge 462/453/1188, Fragmentation 1680/1486/500), aucune ligne L ne vaut `(6+L)^3` (au
niveau qui donne 7 bits en x et y, z vaut 9 bits sur Forge et 5 sur Fragmentation).

Décomposition des cas chiffrés par l'ancien lecteur (bits rapportés par les exceptions) :

| Cas | Ancien | Jeu (exe) | Branche de l'ancien | `e420(param_4)` reproduit ? |
|---|---|---|---|---|
| flock-destination 11de8353 (L = 2) | 29 = 1+1+1+24+2 | 52 | porte posée | oui |
| flock-destination 11de8353 (L = 2) | 4 = 1+1+0+2 | 70 | precHigh = 1 | oui |
| crew-order 084a804d (L = 1) | 23 = 1+1+21 | 49 (+26) | porte posée | oui |
| tacmap-poiicon 11de8353 (L = 1), composant 239 -> 264 | 24 = 1+1+1+21 | 49 (+25) | porte à 0 | **non** |
| respawn e5adf7b2 (L = 1), composant 44 -> 71 | 1+1+1+1+21+19 = 44 | 71 | porte à 0 | **non** |
| tacmap (traversée 6/6/6, L = 1) | 1+[1]+18 | 67 / 49 | — | non (`e420(1)` donnerait 7/7/7) |

Conclusion : une règle « les anciens builds lisaient au niveau du registre » n'est pas établie (3
points sur un film chacun, 1_9 et 1_10) et elle est réfutée pour poiicon et respawn ; elle ne peut
de toute façon pas s'étendre à HI_1_13_0 (C1).

### C5 — ÉTABLI côté jeu : world-object i0 et flock-position — la lecture du jeu est la seule forme possible, les pertes sont au niveau du bruit

world-object : grammaire §1.3 ; les gains (+579, +65, +32, +16, +7 records d'image-clé) dominent les
pertes (-16, -3, -1, -1, -1), sans ordre de build (HI_1_8_0 gagne en ti=38 et perd en ti=42).
flock-position : +24 paquets sur 000d5950 contre -1 paquet sur e5adf7b2. Correctif proposé : retirer
ces deux exceptions (lecture `lireE420(0x10)` déjà portée, et `lireE494(0x10)`), en documentant les
pertes résiduelles comme compensations ; la queue R(2) de world-object est lue si la position est
finie (toujours vraie hors garde R(96)).

### C6 — PISTE non tenue par la position : ti38-i18 (perte sur HI_1_13_0, sans gain)

Jeu : `FUN_142f036f0` (R(8), puis par bit `FUN_140c1e79c` = R(1)[R(19)] + R(8) — largeur 8 relue en
`140c1e80f` — et `e494(0x10)`) ; écrivain `FUN_142f06958` strictement symétrique (W(8), par bit
`FUN_140c1c6d0` + `FUN_1407eb61c(pos, indice d'objet, 0x10, 0)`). Le Go « jeu » est donc la bonne
grammaire, et la baisse 317 -> 245 sur fb1a1a72 (HI_1_13_0) vient d'ailleurs : cadrage de
l'image-clé ti=38 (ordre des composants du corps, bloc d'en-tête, ou i0 de ti=38 qui précède i18 dans
le même record — l'essai « R(96) seul » de l'exception ne pouvait pas réussir sans i0 en R(96) aussi).
Hors périmètre T4 : à confier à la mesure §6.

## 5. Vecteurs de test construits d'après l'écrivain

Notation : W(n) = n bits écrits MSB d'abord (lecteur `1406cf008`/`ReadBits`). Carte Forge 1 plage :
`DAT_144632be0 = 1`, ligne 0x10 de la plage = 15/15/17 ; table défaut L16 = 22/22/22 ; ±100 L16 =
14/14/14.

1. **waypointstate** (`FUN_142ede52c`, registre L = 2) :
   `W(1)=1 | W(32)=0x0BADF00D | W(1)=0 (porte) | W(1)=0 (index) | W(15) W(15) W(17) | W(1)=1`
   = 83 bits. Attendu : le lecteur porté consomme 83 bits et rend la queue ; l'ancien en consomme 53.
   Variante porte posée : `W(1)=1 | W(66)` à la place de porte+index+axes = 101 bits.
2. **flock-destination** (`FUN_142eda4b4`, L = 2) :
   `W(1) | W(1)=1 (porte) | W(22) W(22) W(22) | W(2)` = 70 bits ; porte à 0 : 1+1+1+47+2 = 52 bits —
   les deux longueurs de l'exception. Contrôle de queue : avec un registre factice L = 1, le lecteur
   du jeu laisse 2 bits non lus ; l'instrument doit le signaler (C2).
3. **world-object i0** (`FUN_14076e29c` / écrivain `1432067d4`) :
   precHigh 0, porte posée : `W(1)=0 | W(1)=1 | W(22)x3 | W(2)` = 70 bits (pas de queue de poignée) ;
   precHigh 0, porte 0 : `W(1)=0 | W(1)=0 | W(1)=0 | W(15) W(15) W(17) | W(2)` = 52 bits ;
   precHigh 1 : `W(1)=1 | W(14)x3 | queue de poignée | W(2)`.
4. **i0 bipède, branche absolue precHigh = 1** (`FUN_142e2d86c`) : `W(1)=1 | W(14)x3` puis le R(2) de
   `LAB_1406cffd7` côté lecteur : 44 bits après le bit — l ancien n en lit aucun.
5. **index impossible** (C3) : carte à une plage, `W(1)=0 | W(1)=1 | ...` n'est jamais produit par
   `FUN_140770640` ; le lecteur doit rendre « index de plage impossible ».
6. **oracle de plage** (porte posée) : une position W(22)x3 en table défaut se déquantifie par
   `x = -20000 + q * 40000 / 2^22` ; l'écrivain ne pose la porte que si le point est HORS de toute plage
   valide (`FUN_14077084c`), et le borne à ±20000. Une lecture porte posée qui se déquantifie DANS la
   plage de la carte sur les trois axes est donc impossible chez l'écrivain : témoin de cadrage faux
   pour l'ancien comme pour le nouveau lecteur.

## 6. Mesures proposées (carte de fermeture v2, ou sonde `research`)

1. **Stabilité par entité** (oracle indépendant de la fermeture) : pour chaque site d'exception,
   sous les deux lecteurs, la valeur décodée d'une même entité (slot) d'un record à l'autre. Les
   zones tacmap, les points de passage, les nuées au repos et les corps rigides immobiles répètent
   la même position au bit près ; le lecteur sous lequel les positions d'une entité sont STABLES et
   DANS la carte est le bon, build par build. Chiffre à rendre : part des paires consécutives
   identiques par site, par build, par lecteur.
2. **Histogramme `IndexAbsolus` par site et par film** (déjà compté par `lireE524Sur`) : index
   hors {-1, plage cataloguée} sur une carte à une plage = cadrage faux (C3) ; sur Live Fire
   (0797ce72, 60ae07c4), part des index 0/2/3 = poids de l'écart C3.
3. **Registre par build des composants à queue** : colonne niveau (`entrée + 0x100`) de `ti=21 i2..i11`
   et `ti=34 i7` sur les mini-bobines des sept builds (même instrument que
   `TestParam4RegistreParBuild`) : toute valeur <= 1 est la seule dépendance de contenu légitime (C2).
4. **Ventilation de la carte v2** : pour chaque paquet perdu ou gagné par une exception, la sortie de
   vue B et le dernier composant lu : si la perte de l'ancien lecteur et le gain du nouveau
   s'accompagnent d'une liaison de slot différente (cas `unit-actor-state` 12:1118, slot 570 lié à ti=4),
   le paquet témoigne d'une ATTRIBUTION en amont, pas d'une largeur.

Effet attendu : sur HI_1_13_0, les sept sites passent au portage unique sans changer la fermeture de
plus de quelques listes (déjà chiffrées : -10 paquets 51ebbc0f displayasset, -1 à -4 listes ailleurs,
contre les gains déjà mesurés) ; les deux retraits C5 apportent le gain net déjà mesuré (image-clé
ti=38 bcb6d393 +579, flock 000d5950 +24 paquets).

## 7. Questions ouvertes

- Quel build a écrit les films HI_1_9_0 et HI_1_10_0 ? Un seul exécutable est disponible : toute
  règle « par build » antérieure à HI_1_13_0 ne peut se prouver que par mesure (§6.1).
- Live Fire : les plages 0, 2, 3 sont-elles valides (AABB non dégénérée) et quelles bornes ont-elles ?
  Donnée de carte (tag scenario), hors Ghidra.
- ti38-i18 : quelle partie du record d'image-clé ti=38 décroche réellement sur fb1a1a72 (C6) ?
- `FUN_14076e3e4` (queue de poignée de world-object, precHigh = 1) : sa longueur exacte
  (`FUN_1408f0ac4` + R(1) + suite) n'a pas été relue au bit près ; le R(59) mesuré de l'ancien lecteur
  vaut 42 + queue + 2 si la queue fait 15 bits.
