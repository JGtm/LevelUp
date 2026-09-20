# NOTE 5.3 — LES ETATS DE MOUVEMENT DU SPARTAN DANS LE FILM

> Lot 5.3 (post-chantier, RECHERCHE puis port si prouve). Branche `feat/decfilm-53`, base
> `6e86db356`. Question de l'utilisateur (2026-09-19) : « on a les evenements de joueurs comme
> les slide, crouch, sprint et saut ? ».
>
> **ETAT : point 1 (L'ECRIVAIN) FAIT ; D1 MESUREE sur les sept mini-bobines (§ 2.4 bis).
> Point 2 (preuve sur film entier) EN ATTENTE DE VOIE LIBRE** — un backfill tient le parc ;
> aucun film du cache n'a ete lu, aucune base ouverte.

---

## 0. LA REPONSE COURTE, TELLE QUE L'ECRIVAIN LA DONNE

| Geste | Le film l'ecrit-il ? | Ou | Forme |
|---|---|---|---|
| **Accroupi** | **OUI, directement** | `ti=35 i29 unit-crouch-component` | ETAT par image : booleen + fraction 0..1 |
| **Glissade** | **OUI, directement** | `ti=35 i62 biped-slide-component` | ETAT par image : booleen + direction/intensite + 2 fractions 0..1 + un octet |
| **Sprint** | **OUI, mais sans son nom** | `ti=35 i54 biped-mobility-action-component` — l'ACTION DE MOBILITE, dont le corps est PARTAGE avec l'evenement de fil 43 `initiate_mobility_action` (§ 2.7) | EVENEMENT date : flag1 = « une action est transmise », puis un identifiant et une transformation. **Quelle** action reste a nommer (§ 2.8) ; repli mesurable par la vitesse `i1` |
| **Saut** | **PAS SOUS SON NOM — ni composant, ni evenement de joueur** | negatif mesure (§ 2.7) : seuls `ai_jump` (78) et `AILand` (72), prefixes AI. **Candidats vivants** : le mot de 32 bits optionnel d'`i18 unit-control +0x544` (§ 2.9.1, deja decode et jete), `i55` (§ 2.4), `i63`, et la composante verticale d'`i1` | Theater rejoue l'animation depuis un ETAT REPLIQUE, pas depuis les entrees (§ 2.9.4). Le bit de saut, s'il existe, est dans le mot de 32 bits — **mesurable en 5.3.2, non nommable avant** |

**LE NEGATIF EST MESURE, PAS SUPPOSE** (§ 3) : sur les **64 composants** de l'archetype bipede
(`ti=35`), **aucun** ne porte « sprint » ni « jump » dans son nom ; et sur le pool **complet**
des chaines de l'image, **aucun** nom de composant ne les porte non plus, alors que le moteur
emploie ces deux mots abondamment ailleurs (71 chaines « sprint », 94 « jump »).

---

## 1. LA METHODE, ET CE QUI LA REND PUBLIABLE

Chaine du descripteur du lot 3.7 (`NOTE_3_7_REAPPARITION_2026-09-17.md` § 7), rejouee telle
quelle : nom du composant -> accesseur `LEA reg,[rip+chaine] ; RET` -> slot unique de `.rdata`
= `descripteur + 0x18` -> ecrivain `descripteur + 0x40`. Chaque pas exige l'UNICITE.

**GARDE DE PUBLICATION REUTILISEE, PAS RECOPIEE** : `reapparition.Calibrer` (exporte par ce
lot) rejoue la chaine sur les six temoins du depot ; une seule concordance qui rate et la
passe ne publie rien.

```
== CALIBRATION (6 temoins) ==
  OK  managed-player-back-button-scoreboard-flair-component     ecrivain 0x142ed5af4
  OK  managed-navpoint-sub-type-component                       ecrivain 0x1410e0cac
  OK  managed-navpoint-radial-progress                          ecrivain 0x140fc8d14
  OK  device-position-animation-name-component                  ecrivain 0x1410156e4
  OK  managed-navpoint-manual-timer-initial-duration-component  ecrivain 0x142ed5194
  OK  managed-navpoint-manual-timer-current-duration-component  ecrivain 0x142ed512c
```

**6 / 6.** Les cinq cibles du lot sont ensuite resolues sans un echec, et les cinq adresses
d'ecrivain retombent **a l'octet** sur celles que le decodeur porte deja — la chaine lit bien
cette image-la.

| cible | descripteur | ECRIVAIN | compagnon `+0x28` |
|---|---|---|---|
| `ti=35 i29` unit-crouch | `143d062a8` | **`142ed42a8`** | `142ed9e90` |
| `ti=35 i62` biped-slide | `143d0ca68` | **`142f02978`** | `142f054b8` |
| `ti=35 i54` biped-mobility-action | `143d0c9c0` | **`1408f0264`** | `142f053f8` |
| `ti=35 i55` biped-posture-physics | `143d0cd98` | **`142f0293c`** | `142f05478` |
| `ti=35 i1` object-translational-velocity | `143d0c8d0` | **`14076d45c`** | `14320ca04` |

Ghidra etait joignable pendant ce lot (`127.0.0.1:8089`, `HaloInfinite.exe` analyse,
311 103 fonctions) et a servi **en lecture seule** pour les decompilations et les
desassemblages ci-dessous. Son index de chaines est vide : le balayage de vocabulaire du § 3
est fait par l'instrument Go, sur les octets des sections.

---

## 2. CE QUE LE JEU ECRIT, CHAMP PAR CHAMP

### 2.1 `ti=35 i29 unit-crouch-component` — ACCROUPI : un booleen ET une fraction

`FUN_142ed42a8`, 89 octets, lu en entier :

```
142ed42bc CALL 1406cf008                         R(1)   -> [unite + 0x7e8]   (octet)
142ed42c1 MOVSS XMM3,[143cd8374]                 borne haute
142ed42d1 XORPS XMM2,XMM2                        borne basse = 0.0
142ed42d9 MOV [RSP+0x20],0xa                     largeur = 10 bits
142ed42e7 CALL 1406d84b4                         dequantification
142ed42f3 MOVSS [RDI+0x7ec],XMM0                 -> [unite + 0x7ec]   (float32)
```

`DAT_143cd8374` lu dans l'image : `00 00 80 3f` = **1.0f**.

**VERDICT** : le film ecrit, A CHAQUE IMAGE ou le composant est porte, **un booleen « accroupi »
et une fraction quantifiee sur 10 bits dans [0.0, 1.0]** (1 024 paliers) — la PROGRESSION de
l'accroupissement, pas seulement son etat binaire. C'est un ETAT, pas un evenement : les
intervalles « accroupi de t0 a t1 » se reconstruisent par changement de front du booleen.

Le decodeur lit les deux et les JETTE (`consumeUnitCrouch`, `unit_weaponstate.go`).

### 2.2 `ti=35 i62 biped-slide-component` — GLISSADE : un booleen, une direction, deux fractions

`FUN_142f02978` (mince) -> `FUN_142f26ce8`, lu en entier :

| bits | -> | ce que c'est |
|---|---|---|
| `R(1)` | `[etat + 0x00]` octet | **« en glissade »**. A 0, le composant s'arrete la |
| `R(1)` puis `R(19)` + `R(10)` | `[etat + 0x04]` | direction (cubemap 19 bits) + magnitude (10 bits) : **le vecteur de la glissade** |
| `R(8)` dequant `[0.0, 1.0]` | `[etat + 0x10]` float | fraction |
| `R(8)` dequant `[0.0, 1.0]`, **seulement si `param_4 >= 1`** | `[etat + 0x14]` float | fraction ; **defaut 1.0** quand la porte est fermee (`XMM3` porte encore la borne haute) |
| `R(8)` | `[etat + 0x02]` (stocke en mot) | petit entier 0..255 |

La borne des deux dequantifications est le meme `143cd8374` = **1.0f**.

**VERDICT** : la glissade est un ETAT par image, avec sa direction et son intensite. Le
decodeur porte la grammaire **au bit pres** et jette toutes les valeurs.

**CORRECTION DE L'ETAT DES LIEUX DU BRIEF** : « biped-slide = nom present dans l'exe, alias non
rattache (ligne -1) » est FAUX. La ligne `-1` de `ecs_table.tsv` est l'ALIAS d'orthographe
accepte par le dispatch ; le composant reel est **`ti=35 i62`, statut `porte`**, ligne 751 de
la table.

### 2.3 `ti=35 i54 biped-mobility-action-component` — UNE INITIATION, PAS UN ETAT

`FUN_1408f0264` : `R(1)` flag1 -> `[0x1295]`, `R(1)` flag2 -> `[0x1296]`, puis si flag1 un
handle a largeur variable (`FUN_1408f0ac4`, categorie 0) et le corps `FUN_1408f02c8`.

Corps, champs relus un par un dans la decompilation :

| bits | -> champ | ce que c'est |
|---|---|---|
| `R(1)` ; si 1 `R(10)` sinon `0xFFFFFFFF` | `+0x08` | **identifiant optionnel sur 10 bits** (1 024 valeurs), sentinelle « aucun » |
| `R(1)` ; si 0 : position absolue + avant/haut | — | l'ancrage de l'action |
| `R(96)` brut | `+0x0c` | vecteur 3 flottants |
| position (porte de pleine precision) | `+0x3c` | |
| 3 x (3 x `R(12)`) | `+0x48` | |
| 2 x `R(24)` | `+0x6c`, `+0x78` | |
| 3 x `R(12)` | `+0x84` | |
| 2 x `R(10)` | `+0x90`, `+0x94` | deux flottants |
| `R(1)` | `+0xa1` | |
| `R(7)` | `+0x98` | entier 0..127 |
| `R(2)` | `+0x9c` | **enumere a 4 valeurs** |
| `R(1)` | `+0x9f` | |

Total mesure : **365 a 447 bits** selon les portes internes.

Le vocabulaire de l'image nomme ce composant : `initiate_mobility_action` (`143c97470`) et
`biped-initiate-mobility-action: relevance = %5.3f` (`143e0c500`). **C'est une INITIATION
d'action**, transmise avec la transformation du bipede au moment ou elle commence — donc un
EVENEMENT date, pas un niveau replique. Les candidats a l'identite de l'action sont `+0x08`
(10 bits), `+0x98` (7 bits) et `+0x9c` (2 bits) ; **aucun n'est nomme dans l'image**, leur sens
se tranche par la mesure (§ 4).

Le decodeur lit flag1 et flag2, les publie par `MobilityActionHook` — **sans aucun
consommateur** — et jette le reste.

### 2.4 `ti=35 i55 biped-posture-physics-component` — UN TAG DE 2 BITS QUI OUVRE QUATRE CHARGES

`FUN_142f0293c` resout `bipede + 0x12b4` puis appelle `FUN_142f1f630`, qui lit **`R(2)`** et
passe la valeur a `FUN_141fd997c`. Celle-ci ecrit un octet de discriminant en `+0x2c` de la
cible et appelle un lecteur DIFFERENT par valeur :

| tag | discriminant ecrit | lecteur | consomme des bits ? |
|---|---|---|---|
| 0 | — | `FUN_142f265dc` | **OUI** (`1406cf008`, `1406d310c`, ...) |
| 1 | `1` | `FUN_142f25a3c` | **OUI** (`1406d310c`, `14076dc04`, `14076e494`) |
| 2 | `2` | `FUN_142f263ac` | **OUI** (`14076dc04`, `14076e494`, `1408f0ac4`) |
| 3 | `3` | `FUN_142f264f4` | **OUI** (`14076dc04`, `14076e494`) |

Les charges ecrivent des vecteurs a 3 flottants et des handles a sentinelle `0xffff` /
`0xffffffff` : c'est un **ancrage physique** (« contre quoi le bipede se tient »), pas un
libelle de posture. Le vocabulaire de l'image le confirme du cote animation :
`biped_ground_transient_posture`, `biped_posture_animation`, `character_posture`.

> **DECOUVERTE D1 (§ 6)** : `consumeBipedPosturePhysics` fait `br.Skip(2)` — il lit le tag et
> **ne consomme AUCUNE des quatre charges**, alors que les quatre en consomment chez le jeu.
> **MESUREE le 2026-09-20 sur les sept mini-bobines : § 2.4 bis.**

### 2.4 bis LA MESURE DE D1 — SEPT MINI-BOBINES, AUCUN FILM DU CACHE

Instrument : `grammar/mouvement_i55_d1_research_test.go` (tag `research`, aucun octet de
production). Le tag est relu DIRECTEMENT dans le payload a `CompResult.StartBit`, la position
que la marche de production publie — donc sans crochet d'observation, donc sans toucher
`observateur.go` ni `components_probe.go` (qui feraient bouger `grammar.Rev`).

**CE QUE LES MINI-BOBINES PERMETTENT, ET CE QU'ELLES INTERDISENT.** Leur `PROVENANCE.txt` est
formel : chunk de REGISTRE + douze paquets d'IMAGE-CLE + PIED, et **aucun paquet de
replication**. Le chemin DELTA du bipede n'y est donc pas exercable — mesure a l'appui,
`ScanFilmBipedPositions` refuse les sept (« aucun slot biped (ti=35) dans les keyframes »). La
mesure se fait sur le chemin d'IMAGE-CLE, qui marche les memes composants par la meme boucle et
le meme `case`.

| bobine | records `ti=35` bornes | `i55` franchi | t0 | t1 | t2 | t3 |
|---|---|---|---|---|---|---|
| a521164d | 205 | 77 | 64 | 6 | 2 | 5 |
| 60ae07c4 | 236 | 50 | 40 | 6 | 2 | 2 |
| 11de8353 | 255 | 214 | 170 | 16 | 10 | 18 |
| 111fa685 | 214 | 158 | 119 | 12 | 12 | 15 |
| e5adf7b2 | 237 | 182 | 135 | 24 | 8 | 15 |
| bcb6d393 | 137 | 134 | 97 | 10 | 9 | 18 |
| fb1a1a72 | 80 | 80 | 60 | 5 | 9 | 6 |
| **TOTAL** | **1 364** | **895** | **685** | **79** | **52** | **79** |

**LE TAG EST NON NUL 210 FOIS SUR 895 (23,5 %).** Ce n'est donc pas un negatif : le `Skip(2)`
saute bel et bien, une fois sur quatre, une charge que le jeu lit.

**CONTROLE INTERNE (necessaire).** Si le curseur avait derive AVANT `i55`, les deux bits relus
seraient du bruit — et du bruit rend quatre valeurs equiprobables. L'ecart a l'uniforme vaut
**1 270** pour 895 lectures (un bruit en rendrait environ 3) : les deux bits sont un champ
structure, lu au bon endroit. Corroboration : `i29`, `i54` et `i55` sont declares par
**exactement les memes 895 records** (65,6 %), et `i62` par **aucun** — le masque d'un record
d'image-cle n'est pas du hasard.

**CE QUE CHAQUE TAG COUTE CHEZ LE JEU** (feuilles relevees au desassemblage ; les deux
occurrences d'un meme `ADD [reg+0x2c], n` sont les deux branches d'UNE lecture) :

| tag | lecteur | feuilles consommatrices | cout plancher |
|---|---|---|---|
| 0 | `FUN_142f265dc` | `1406cf008` x3 = 3 x R(1) ; une largeur variable via `1406d310c` | **≥ 3 bits + un champ a largeur variable** |
| 1 | `FUN_142f25a3c` | trois largeurs variables via `1406d310c` ; `14076e494` position ; `14076dc04(0x13)` = R(19) | **≥ 19 bits + position + 3 champs variables** |
| 2 | `FUN_142f263ac` | `14076e494` position ; `14076dc04` ; `1406cf008` = R(1) ; `1408f0ac4` handle ; une R(32) | **≥ 52 bits + position** |
| 3 | `FUN_142f264f4` | `1406cf008` x3 ; `14076e494` position ; `14076dc04(0x13)` = R(19) | **≥ 22 bits + position** |

`14076e494` est la position du port (`consumeE494Position`) : porte de pleine precision puis
`R(96)` brut, ou le corps quantifie. **Aucun des quatre tags ne coute zero bit.**

**« POURQUOI LA MARCHE RESTE-T-ELLE ALIGNEE ? » — SUR CES BOBINES, ELLE NE L'EST PAS, ET ELLE NE
L'A JAMAIS ETE.** L'oracle de fermeture le dit : sur les 1 364 records `ti=35` bornes, **6
ferment** (0,4 %), et **AUCUN de ces 6 n'a franchi `i55`**. La marche du bipede s'arrete plus
loin sur `i60 simulation-state-component`, non porte (c'est exactement ce que le golden
`keyframe_closure.golden` fige depuis le lot 0.A.3) — donc rien, sur le chemin d'image-cle,
n'a jamais verifie le curseur au-dela de `i55`. Il n'y a pas de paradoxe a expliquer ici : il
n'y avait pas d'alignement prouve.

**CE QUI RESTE OUVERT, ET C'EST 5.3.2.** La bit-exactitude du corpus concerne le chemin DELTA,
que les mini-bobines ne portent pas. Trois hypotheses, toutes mesurables sur un film entier :
(a) `i55` n'est pas declare dans les masques delta ; (b) il l'est, et son tag y est toujours 0
avec une charge de tag 0 nulle en pratique ; (c) il l'est avec des tags non nuls, et la marche
delta derive sans qu'aucun oracle actuel ne le voie. **Tant que ce compte n'est pas fait,
aucune conclusion de posture ne doit s'appuyer sur `i55`.**

### 2.5 `ti=35 i1 object-translational-velocity` — LA VITESSE, DEJA DECODEE ET JETEE

`FUN_14076d45c` : `R(1)` ; si pleine precision `R(96)` brut (3 flottants), sinon `R(19)`
direction + `R(10)` magnitude. C'est la voie du SAUT par derivation (composante verticale) et
du SPRINT par seuil de vitesse au sol.

### 2.6 `ti=35 i56 biped-spartan-ability-energy` — L'ENERGIE DE LA CAPACITE D'ARMURE, ET NON LE SPRINT

Le catalogue d'aout (`RECAP_STATS_EXPLOITABLES.md:229`, `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md:595`)
range `i56` sous « crouch / sprint / slide / mobilite ». **Le lecteur du depot dit autre chose, et
il a ete relu au desassemblage** (`ability_energy.go`, deser `FUN_140fc1410` -> `FUN_140fc147c`) :

```
R(3) masque ; puis, POUR CHAQUE BIT ARME, R(7)   -> [bipede + 0x12ea + i]
bit a 0 -> valeur par defaut 0x7F, AUCUN bit lu
cout total : 3 + 7 x popcount(masque) = 3 a 24 bits
```

C'est **la jauge des TROIS EMPLACEMENTS DE CHARGE de la capacite d'armure**, compagnon d'`i48`
`biped-desired-ability-set` (la capacite SELECTIONNEE). Le sprint de Halo Infinite ne consomme
aucune energie — le propulseur, le grappin, le repulseur, le mur, le camo et le surbouclier si.
**`i56` mesure donc l'usage d'EQUIPEMENT, pas le sprint** ; le classer sous « sprint » est une
erreur du catalogue d'aout, que la presente note corrige.

> Le fichier du depot portait deja la lecon : « le DECOMPILE MENT ICI » — Ghidra supprimait le
> bloc froid qui lit les 7 bits, et un portage anterieur en avait conclu « 3 bits, bit-exact ».
> Seul le desassemblage fait foi.

### 2.7 LE CANAL D'EVENEMENTS — UN SEUL CORPS POUR DEUX CANAUX, ET UN CANAL MESURE VIDE

Le film porte, a cote des composants, une liste d'EVENEMENTS. Deux types y touchent au
mouvement (`GRAMMAIRE_EVENTS_FILM_2026-08-30.md`, annexe A, table reconstruite depuis le
registrar `FUN_140e453b4`) :

| type de fil | nom | tampon | lecteur `vtable+0x68` |
|---|---|---|---|
| **43** | `initiate_mobility_action` | 164 octets | `0x142ef8f04` |
| **78** | `ai_jump` | 28 octets | `0x142ef8df0` |

> **CORRECTION DE LECTURE** : les nombres « 164 » et « 28 » sont la **TAILLE DU TAMPON DE
> RECEPTION** (`vtable+0x10`), pas un nombre d'occurrences — l'en-tete de l'annexe A le dit
> mot pour mot. Aucun comptage de corpus ne se lit dans cette table.

**LE FAIT DECISIF : L'EVENEMENT 43 ET LE COMPOSANT `i54` PARTAGENT LE MEME CORPS.**

```c
undefined1 FUN_142ef8f04(..., longlong param_3, undefined8 param_4) {
  uVar1 = FUN_1406cf008(param_4);          // R(1)  -> +0x9d   == flag1
  *(undefined1 *)(param_3 + 0x9d) = uVar1;
  *(undefined1 *)(param_3 + 0x9e) = 0;     //          +0x9e   == flag2, FORCE A 0
  FUN_1408f02c8(param_3, param_4);         // LE MEME CORPS QUE i54
  return 1;
}
```

`FUN_1408f02c8` n'a **que deux appelants** (xrefs Ghidra) : `FUN_1408f0264` (le deser du
composant `i54`) et `FUN_142ef8f04` (le lecteur de l'evenement 43). Meme structure, memes
champs, meme enum — **ce qui se decode une fois sert les deux canaux**.

**MAIS LE CANAL D'EVENEMENTS EST MESURE VIDE POUR CE TYPE.** Le lot R5 (2026-09-03) :
« Treize types suspects ont ZERO tete sur les 325 160 paquets » — dont 42 et 43. Le lot R7,
ecrit exactement pour lever le doute en marchant la LISTE ENTIERE de chaque paquet, conclut
(`RAPPORT_R7_TRAME_COMPLETE_2026-09-03.md`) :

| type | verdict R7 | mesure |
|---|---|---|
| 42 `biped_dodge` | **ABSENT du film** | 0 tete pour 30,3 attendues |
| 43 `initiate_mobility_action` | **ABSENT du film** | 0 tete pour 16,3 attendues |

**CONSEQUENCE POUR LE LOT** : la voie vivante de l'action de mobilite est le **COMPOSANT
`i54`**, pas l'evenement. C'est coherent avec ce que le depot a deja mesure sur `i54` — « le
temoin sans capacite porte quand meme 631 evenements » (`ecs_table.tsv`, ligne 743) : l'action
de mobilite circule, mais dans le flux d'etat.

> Confirmation croisee du champ-a-champ : le lot R7 portait DEJA la grammaire du type 43
> (`r7_charges_lot5_research_test.go:70`) et elle finit par `Skip(1 + 7 + 2 + 1)` — exactement
> les `+0xa1`, `+0x98`, `+0x9c`, `+0x9f` du § 2.3, releves independamment. Deux lectures, une
> grammaire.

**LE SAUT DU JOUEUR N'EST PAS UN EVENEMENT — NEGATIF MESURE SUR LA TABLE ENTIERE.** Sur les 123
types, les seuls au vocabulaire du saut sont **`ai_jump` (78)** et **`AILand` (72)**, tous deux
prefixes `AI`. Le vocabulaire de l'image qui les entoure est celui de la NAVIGATION DES BOTS :
`AI Jump Action: relevance = %5.3f`, `ai_clamber_from_jump`, `ai_clamber_max_jump_height`,
`Bot_EnablePathlessMeleeJump`, `BotTuning_JumpUpExceedsMaxHeightAddedCost`,
`HKAI_TRAVERSAL_TYPE_JUMP` (une categorie de traversee du moteur de navigation Havok). Aucun
type `biped_jump` ni `player_jump` n'existe. **Le saut du joueur reste donc a deriver de `i1`**
(composante verticale), et c'est ce que 5.3.2 mesure.

### 2.8 L'ENUM DE L'ACTION DE MOBILITE — LES TROIS CANDIDATS, ET CE QUI LES DEPARTAGERA

Le corps partage porte trois champs susceptibles de nommer l'action :

| champ | largeur | forme | lecture |
|---|---|---|---|
| `+0x08` | `R(1)` puis `R(10)` | sentinelle `0xFFFFFFFF` quand absent ; plage `FUN_1406d310c(0x400)` = 1 024 | **index de definition** d'action, le plus probable |
| `+0x98` | `R(7)` | entier nu 0..127 | |
| `+0x9c` | `R(2)` | entier nu 0..3, masque `& 0x03` | **candidat n°1 pour l'enum**, voir ci-dessous |

**CE QUI DESIGNE `+0x9c`** : la table d'actions d'entree du moteur (`143d03c40`, § 3) aligne
**exactement quatre** actions de mobilite consecutives — `Sprint` (`143d03c40`), `Thruster`
(`143d03c48`), `Clamber` (`143d03c58`), `Slide` (`143d03c60`) — et `+0x9c` a exactement quatre
valeurs. Ce n'est PAS une preuve : c'est une coincidence de cardinal, et la table d'entree est
une table de LIAISON DE COMMANDES, pas forcement l'enum reseau.

**AUCUN DES TROIS CHAMPS N'EST NOMME DANS L'IMAGE** : le balayage du pool complet des chaines
(§ 3) ne rend aucune etiquette attachee a ces offsets. Les nommer demande donc l'une des deux
voies suivantes, et la premiere est de loin la moins chere :

1. **LA VENTILATION SUR FILM (5.3.2)** — croiser, par joueur et par instant, la valeur de
   `+0x9c` (et de `+0x08`) avec la vitesse au sol d'`i1`, l'etat d'`i62` (glissade) et la
   variation d'altitude. Une classe qui coincide avec « vitesse > marche et pas de glissade »
   est le sprint ; une classe qui coincide avec `i62` actif est la glissade ; une classe qui
   coincide avec une montee franche en z contre un mur est l'escalade ; le reste est le
   propulseur, recoupable par `i56` (§ 2.6) et `i48`.
2. la chasse au CONSOMMATEUR du champ dans l'executable (quel code du jeu branche sur
   `bipede + 0x1294`), plus couteuse et sans garantie.

**LA VENTILATION DE L'ENUM PAR JOUEUR S'AJOUTE DONC AU TABLEAU DE 5.3.2.**

### 2.9 L'OBJECTION DE L'UTILISATEUR — THEATER REJOUE L'ESCALADE, DONC LE FILM PORTE DE QUOI LA DECLENCHER

**L'objection est juste, et elle vaut mieux que ma formulation du § 2.7.** Theater rejoue
l'animation de saut et d'escalade : le joueur s'agrippe au rebord. Quelque chose declenche cela.
L'hypothese proposee : les ENTREES repliquees (les bits d'action de la structure de controle
d'unite, comme dans les Halo precedents). Les quatre candidats ont ete lus CHEZ L'ECRIVAIN.

#### 2.9.1 `ti=35 i18 unit-control-component` — DEUX INDEX BORNES ET UN MOT DE 32 BITS

`FUN_141017084`, relu champ par champ (decompilation + desassemblage du site d'appel de queue) :

| bits | -> champ | forme |
|---|---|---|
| `R(1)` porte | — | a 0 : `+0x548` et `+0x726` recoivent la sentinelle `0xFFFF`, rien d'autre n'est lu |
| `R(5)` | `+0x548` (ushort) | **borne : `> 0x20` fait ECHOUER la lecture** (`return 0`) |
| `R(1)` puis `R(6)` | `+0x726` (ushort) | meme borne `<= 0x20`, meme sentinelle |
| `R(1)` puis `R(32)` | **`+0x544` (dword)** | optionnel ; defaut **0** quand la porte est fermee (`LEA R8,[RBP+0x544]`, `FUN_14080d69c`) |

**CE QUE CELA DIT.** `+0x548` et `+0x726` sont des **INDEX** : plage 0..32, sentinelle `0xFFFF`,
et une valeur hors plage arrete la lecture. Ce ne sont pas des drapeaux d'action. En revanche
**`+0x544` est un mot de 32 bits optionnel, de defaut 0** — c'est **exactement la forme d'un
champ de bits de commande**, et c'est le SEUL champ de cette forme sur tout l'archetype bipede.

**CE QUE CELA NE DIT PAS.** L'ecrivain ne le nomme pas : `FUN_14080d69c` est la feuille
generique « porte + valeur », aucune constante ni enumere ne l'accompagne, et le balayage du
pool des chaines (§ 3) ne rend aucune etiquette attachee a cet offset. Un mot de 32 bits a
defaut 0 est aussi bien un horodatage, un compteur, une graine ou un handle.

> Le glose d'`ecs_table.tsv` pour `i18` — « les ENTREES DE COMMANDE de l'unite (ce que le joueur
> appuie) » — porte deja la mention **« Non mesure »**. Cette note ne la confirme ni ne
> l'infirme : elle la REDUIT a un champ precis, `+0x544`, et la rend mesurable.

**LE DECODEUR LE LIT DEJA ET LE JETTE** (`consumeUnitControl` -> `consumeOpt32`). La ventilation
de ses 32 bits ne coute donc aucun octet de grammaire : c'est une sonde, pas un port.

#### 2.9.2 Les trois autres candidats, ecartes chez l'ecrivain

| composant | ce que l'ecrivain ecrit | verdict |
|---|---|---|
| `i19 unit-actor-control` | des handles resolus en RAM | **DEJA REFUTE par le depot** comme pont vers le joueur (`ecs_table.tsv`) |
| `i25 unit-command-tick` | `R(10)` = le NUMERO d'image d'entree | le film garde le TICK de commande **sans** le champ de boutons qui l'accompagnerait |
| `i49 biped-control-context` | `R(w)` avec **w = 4 si pleine precision, sinon 2** (`DAT_145121140`), puis `R(1)` -> `ctx+0xa33` / `+0xa36` | un CONTEXTE etroit, pas une pression de bouton. **Au passage : le glose « 3 bits (5 valeurs) » d'`ecs_table.tsv` est FAUX** — l'ecrivain lit 2 ou 4 bits selon un reglage de processus (D6) |

**Aucun champ de bits d'action replique n'a ete trouve sur le bipede**, hors le mot de 32 bits
d'`i18`.

#### 2.9.3 `i55` : les quatre tags sont-ils la machine d'etat de posture ?

L'hypothese de l'utilisateur (au sol / aerien / accroupi / escalade) a ete confrontee a l'image.

**L'ENUMERE DE POSTURE DU MOTEUR EXISTE, ET IL NE TIENT PAS SUR DEUX BITS.** Le cluster
`0x1437d6870` porte, dans l'ordre, l'enumere d'etat de personnage de Havok :

```
HK_CHARACTER_ON_GROUND · HK_CHARACTER_JUMPING · HK_CHARACTER_IN_AIR
HK_CHARACTER_CLIMBING  · HK_CHARACTER_FLYING  · HK_CHARACTER_USER_STATE_0..2
```

**CINQ etats au moins, donc trois bits au minimum.** Le tag d'`i55` en a deux : il ne PEUT pas
porter cet enumere. C'est un negatif de cardinal, pas une opinion.

**CE QUE LES QUATRE CHARGES DISENT VRAIMENT.** Elles ecrivent des vecteurs a trois flottants et
des handles a sentinelle (`0xffff` / `0xffffffff`), et le tag `2` lit en plus un HANDLE
(`FUN_1408f0ac4`) — c'est-a-dire une REFERENCE D'ENTITE. La forme est celle d'un **ANCRAGE** :
« contre quoi, et ou, le bipede se tient » — rien (t0), un ancrage simple (t1), **un ancrage a
une ENTITE** (t2), un ancrage a une POSITION du monde (t3). L'intuition « ca sent l'accrochage »
vise juste sur cette moitie-la : s'agripper a un rebord EST un ancrage. Mais l'etat aerien, lui,
n'y tient pas : il n'a pas de vecteur d'ancrage. **Nommer les quatre tags reste une mesure de
5.3.2** (§ 2.4 bis : le tag est non nul 23,5 % du temps).

#### 2.9.4 CE QUE LE GRAPHE D'ANIMATION DU JEU DIT DU DECLENCHEMENT

Le pool des chaines porte les noeuds du graphe d'animation, et ils tranchent la question du
MECANISME :

```
EntryNode_objects_animation_graphs_transition_conditions_is_sprinting_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_not_sprinting_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_airborne_tlg
EntryNode_objects_animation_graphs_transition_conditions_is_leap_airborne_tlg
EntryNode_objects_animation_graphs_character_sprint_ang
EntryNode_objects_animation_graphs_character_airborne_default_ang
```

Les transitions du graphe sont gardees par des **CONDITIONS D'ETAT** — `is_sprinting`,
`is_airborne` — et non par des pressions de bouton. Les accesseurs de script vont dans le meme
sens : `SpartanAbilityIsSprinting`, `SpartanAbilityGetSprintFraction`,
`SpartanAbilityIsClambering`, `IsAirborne`, `Unit_IsAirborne`.

**CONCLUSION DU § 2.9, ET CORRECTION DU § 2.7.** Theater ne rejoue pas des ENTREES : il rejoue
un ETAT REPLIQUE, et cet etat est precisement ce que le lot 5.3 a trouve — l'accroupissement et
sa progression (`i29`), la glissade et son vecteur (`i62`), l'ACTION DE MOBILITE avec sa
transformation d'ancrage (`i54`, 365 a 447 bits : c'est la pose de l'accrochage), la posture
physique et son ancrage (`i55`), l'action en cours (`i63`). **L'utilisateur a raison sur le
fond — le film porte de quoi rejouer l'escalade — et le mecanisme est l'etat, pas l'entree.**

La formulation a corriger est celle du saut : **« le saut n'a pas d'evenement » reste vrai et
mesure (§ 2.7), mais il ne faut pas en conclure que le film n'en sait rien.** Il reste UN
candidat de forme « entree » — le mot de 32 bits d'`i18 +0x544` — et trois candidats d'etat
(`i55`, `i63`, la composante verticale d'`i1`). Tous sont mesurables a la voie libre, aucun
n'est nommable avant.

---

## 3. LE NEGATIF, MESURE DEUX FOIS

**(a) Sur l'archetype.** Les 64 composants de `ti=35` (`ecs_table.tsv`) : aucun nom ne contient
« sprint », « jump », « clamber », « vault » ni « airborne ». Les seuls noms de geste sont
`i29 unit-crouch`, `i54 biped-mobility-action`, `i55 biped-posture-physics`, `i62 biped-slide`.

**(b) Sur l'image entiere.** Balayage du pool COMPLET des chaines des sections de donnees
(`reapparition.ChainesContenant`, instrument de ce lot) :

| mot | chaines dans l'image | dont noms de composant |
|---|---|---|
| sprint | 71 | **0** |
| jump | 94 | **0** |
| crouch | 66 | 1 (`unit-crouch-component`) |
| slide | 60 | 1 (`biped-slide-component`) |
| clamber | 34 | **0** |
| mantle | 0 | 0 |
| vault | 14 | **0** |
| thrust | 37 | **0** |
| posture | 14 | 1 (`biped-posture-physics-component`) |
| airborne | 68 | **0** |
| mobility | 10 | 1 (`biped-mobility-action-component`) |

Le moteur PARLE de sprint et de saut — `SpartanAbilityIsSprinting`,
`SpartanAbilityGetSprintFraction`, `SpartanAbilityIsClambering`, `IsAirborne`,
`Unit_IsAirborne`, `HK_CHARACTER_JUMPING`, `CharacterPhysicsModeClambering`, et une table
d'actions d'entree (`143d03c40`) qui aligne `Sprint`, `Thruster`, `Clamber`, `Slide` — **mais
aucun de ces mots ne nomme un composant replique**. Le sprint et le saut ne sont pas ecrits
sous leur nom dans le film ; s'ils y sont, ils y sont sous une autre forme (identifiant
d'action de `i54`, tag de `i55`, action de `i63`, ou derivation de la vitesse `i1`).

> Reserve ecrite : le balayage ne voit que l'ASCII a un octet. Un mot present uniquement en
> UTF-16 y echapperait. Le comptage « stance » (963) est domine par `instance` — sous-chaine,
> pas mot ; il n'est pas utilise comme preuve.

---

## 4. CE QUE LA PREUVE SUR FILM DOIT TRANCHER (point 2, en attente de voie libre)

1. `i29` : distribution du booleen et de la fraction ; les intervalles accroupis tiennent-ils ?
2. `i62` : le booleen de glissade donne-t-il des intervalles courts (< 2 s) coherents avec une
   vitesse elevee decroissante ?
3. `i54` : combien d'evenements par joueur et par match, et **LA VENTILATION DE L'ENUM** —
   `+0x9c` (2 bits) et `+0x08` (10 bits) croises avec la vitesse au sol d'`i1`, l'etat de
   glissade d'`i62` et la variation d'altitude, pour NOMMER les classes (sprint / escalade /
   glissade / propulseur). Voir § 2.8 : c'est la voie la moins chere pour nommer l'enum.
4. `i55` : le compte et la distribution du tag sont FAITS sur les mini-bobines (§ 2.4 bis) ; ce
   qui reste est le chemin DELTA — `i55` y est-il declare, avec quels tags, et la marche y
   ferme-t-elle apres l'avoir franchi ?
5. `i1` : la composante verticale signe-t-elle le saut (vz > 0 puis < 0) ? La vitesse au sol
   separe-t-elle sprint et marche par un seuil net ?
6. **`i18 unit-control +0x544`** : ventiler les **32 bits un par un** contre la
   composante verticale d'`i1`, l'etat d'`i29` (accroupi), celui d'`i62` (glissade) et les
   initiations d'`i54`. **Un bit de saut se signerait par lui-meme** : il s'allume une
   image AVANT que `vz` ne devienne positif. Le champ est deja decode et jete
   (`consumeOpt32`) : la sonde ne coute aucun octet de grammaire.

---

## 5. INSTRUMENTS, REJOUABLES

```bash
export GOCACHE=.../.gocache PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1
cd apps/go-api
MOUV_EXE="D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe" \
  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_mouvement
MOUV_EXE="..." go run -tags=research ./internal/games/halo_infinite/film/research/cmd_mouvement \
  -vocabulaire -plafond=25
```

```bash
# La mesure de D1, sur les sept mini-bobines — aucun film du cache, aucune base.
go test -tags=research -count=1 -v -run TestMouvementI55D1 \
  ./internal/games/halo_infinite/film/internal/grammar/
```

- `film/research/mouvement/` — cibles du lot, vocabulaire, rapport.
- `grammar/mouvement_i55_d1_research_test.go` — la mesure de D1 (`_test.go`, donc HORS de
  l'empreinte de la couche grammaire : `grammar.Rev` ne bouge pas).
- `film/research/reapparition/` — deux ajouts SEULEMENT : `Executable.ChainesContenant`
  (le pool complet des chaines, que l'en-tete de `univers.go` appelait deja) et l'export de
  `Calibrer` (la garde de publication, reutilisee au lieu d'etre recopiee).

---

## 6. DECOUVERTES HORS PERIMETRE (consignees, NON traitees)

- **D1 (5.3)** — `consumeBipedPosturePhysics` (`ti=35 i55`) saute les quatre charges du tag de
  2 bits, que le jeu lit toutes. **MESUREE sur les sept mini-bobines (§ 2.4 bis) : le tag est
  NON NUL 210 fois sur 895 franchissements (23,5 %)**, et aucun des quatre tags ne coute zero
  bit chez le jeu. La question « pourquoi la marche reste-t-elle alignee ? » ne se pose pas sur
  ces bobines — elle ne l'est pas : 6 fermetures sur 1 364 records, **0** parmi ceux qui
  franchissent `i55`. RESTE OUVERT pour le chemin DELTA, mesure de 5.3.2.
- **D2 (5.3)** — les commentaires de portage de `i54` et `i62` donnent comme « descripteur »
  l'adresse du SLOT de nom (`descripteur + 0x18`) : `143d0c9d8` au lieu de `143d0c9c0`,
  `143d0ca80` au lieu de `143d0ca68`. Aucune grammaire n'est fausse ; c'est la meme
  incoherence de convention que la decouverte de bord du lot 3.7 (§ 7.3).
- **D3 (5.3)** — `MobilityActionHook` (`observateur.go`) n'a toujours aucun consommateur. **Il prend son sens
  au § 2.7** : c'est le seul point d'ecoute existant de l'action de mobilite, et le canal
  d'evenements qui porterait la meme chose est mesure VIDE.
- **D4 (5.3)** — **LES DEUX TABLES DE TYPES D'EVENEMENT DU DEPOT NE SONT PAS INDEXEES PAREIL, ET
  C'ETAIT UNE QUESTION OUVERTE.** `event_types_catalogue_test.go` (types 50..127, base
  `0x144724A90`) porte son propre avertissement : « valable SEULEMENT si les deux espaces
  d'index coincident (non etabli) ». **Ils ne coincident pas, et l'arithmetique de la colonne
  « Objet » de l'annexe A le montre sans Ghidra** : l'objet du type de fil 43 est `0x144724d18`,
  soit `0x144724A90 + 81*8` — donc le SLOT 81 ; celui du type 78 est `0x144724d50`, soit le
  SLOT 88. Or `eventTypeNames[81]` vaut bien `initiate_mobility_action` et `eventTypeNames[88]`
  vaut `ai_jump`. **Le piege est reel** : nommer un type de FIL avec cette table rend
  « MusicTrigger » pour ce qui est `ai_jump`. Les instruments de recherche qui comptent
  (`lot1_tirs`, `r7_grammaire`, `r7_charges_lot5`) utilisent la bonne table, celle du
  registrar. **NON TRAITEE** : un lot d'outillage renommerait la carte en `eventSlotNames` et
  poserait le garde-rail.
- **D5 (5.3)** — **LE CATALOGUE D'AOUT RANGE `i56` SOUS « SPRINT », ET C'EST FAUX** (§ 2.6) :
  `biped-spartan-ability-energy` est la jauge des trois emplacements de charge de la capacite
  d'armure (masque `R(3)` + `R(7)` par charge armee, defaut `0x7F`), donc de l'EQUIPEMENT. Deux
  documents le propagent (`RECAP_STATS_EXPLOITABLES.md:229`,
  `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md:595`). **NON TRAITEE** : les corriger demande de
  toucher deux documents hors perimetre de ce lot ; la presente note fait foi en attendant.
- **D6 (5.3)** — **`ecs_table.tsv` DONNE A `i49 biped-control-context` « 3 bits (5 valeurs) » ;
  L'ECRIVAIN EN LIT 2 OU 4.** `FUN_14107166c` calcule sa largeur depuis `DAT_145121140` (le
  reglage de pleine precision du processus) : `w = 4` s'il vaut 1, `w = 2` sinon, puis `R(1)`.
  Le port du depot (`consumeBipedControlContext`) est juste ; c'est la TABLE qui ment. **NON
  TRAITEE** (regle 7) : une ligne de table, a corriger par le lot qui reprendra `ecs_table.tsv`
  avec son garde-rail (voir D2).
- **D7 (5.3)** — **UN MOT DE 32 BITS OPTIONNEL, DEJA DECODE ET JETE, N'A AUCUN SENS ETABLI.**
  `i18 unit-control +0x544` (§ 2.9.1) est le seul champ de tout l'archetype bipede ayant la forme
  d'un champ de bits de commande. `consumeUnitControl` le consomme par `consumeOpt32` et
  l'abandonne. **NON TRAITEE ICI, mais c'est un item de 5.3.2** : sa ventilation bit a bit est la
  mesure la moins chere du lot, et elle tranche la question du saut.
