# NOTE 5.3 — LES ETATS DE MOUVEMENT DU SPARTAN DANS LE FILM

> Lot 5.3 (post-chantier, RECHERCHE puis port si prouve). Branche `feat/decfilm-53`, base
> `6e86db356`. Question de l'utilisateur (2026-09-19) : « on a les evenements de joueurs comme
> les slide, crouch, sprint et saut ? ».
>
> **ETAT : point 1 (L'ECRIVAIN) FAIT. Point 2 (PREUVE SUR FILM) EN COURS — sa conclusion de
> cadence a ete RETIREE le 2026-09-21 (§ 2 quater), l'instrument n'etant pas etalonne.** L'ecrivain aux § 1 a 2.9,
> D1 sur mini-bobines au § 2.4 bis, **la mesure sur `bfecd02b` et `4f77afc1` au § 2 ter**.
> Le port (5.3.3) N'EST PAS LANCE : il attend le retour du pilote. Aucune base DuckDB n'a ete
> ouverte a aucun moment ; les films ont ete lus UN A LA FOIS.

---

## 0. LA REPONSE COURTE, TELLE QUE L'ECRIVAIN LA DONNE

| Geste | Le film l'ecrit-il ? | Ou | Forme |
|---|---|---|---|
| **Accroupi** | **OUI** | `ti=35 i29` | booleen + fraction 0..1. **MESURE : uniquement a l'IMAGE-CLE** (99 % des records d'image-cle, 0 % des records delta) — donc un etat tous les ~18 s, pas par image (§ 2ter.1) |
| **Glissade** | **GRAMMAIRE ACQUISE, DONNEE INACCESSIBLE** | `ti=35 i62` | booleen + direction/intensite + 2 fractions. **MESURE : lue 0 fois sur les deux films** — `i62` est DERRIERE le bloquant `i60 simulation-state-component`. Porter `i60` est le pre-requis chiffre (§ 2ter.3) |
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

## 2 ter. LA PREUVE SUR FILM (5.3.2) — DEUX FILMS, UN A LA FOIS, AUCUNE BASE OUVERTE

> Voie libre du pilote le 2026-09-21. `bfecd02b` (Snowbound, Team Slayer) puis `4f77afc1`
> (Flood Gulch). Instrument : `grammar/mouvement_5_3_2*_research_test.go`, tag `research`.
> Aucune base DuckDB, aucun artefact, aucun corpus.

### 2ter.1 LE FAIT STRUCTURANT : DEUX CANAUX, DEUX CADENCES

> **CONCLUSION RETIREE LE 2026-09-21 — VOIR LE § 2 QUATER.** Les chiffres de ce paragraphe
> restent (ils ont ete mesures), mais leur LECTURE — « les etats ne voyagent qu'a
> l'image-cle » — est FAUSSE : le depot documente en trois endroits un etat bipede PAR TICK
> dans les deltas de type 0, accroupissement compris. Le « 0,0 % » ci-dessous est un defaut
> d'instrument, et le § 2quater.5 nomme le suspect.

La marche de production ne lit pas les composants de mouvement sur le chemin delta — et ce
n'est pas un choix, c'est une limite : `scanRecordDirs` ne modelise que `i1`, `i2`, `i3`, `i4`,
`i5` et `i21`, et s'arrete au premier composant hors de cette liste (**D8**). L'instrument
rejoue donc la VRAIE boucle de composants (`traverseComponentLoopFrom`) sur les records delta,
et la meme sur les records d'image-cle.

| | `bfecd02b` delta | `bfecd02b` image-cle | `4f77afc1` delta | `4f77afc1` image-cle |
|---|---|---|---|---|
| records `ti=35` | **162 444** (90 slots) | **207** | **378 661** (253 slots) | **995** |
| `i1` vitesse | 90,1 % | 24,2 % | 86,3 % | 26,4 % |
| `i18` unit-control | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i29` unit-crouch | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i54` mobility-action | **0,3 %** (464) | 99,0 % | **0,5 %** (1 867) | 98,8 % |
| `i55` posture-physics | **0,0 %** | **99,0 %** | **0,0 %** | **98,8 %** |
| `i62` biped-slide | 0,0 % (2 au masque) | 0,0 % | 0,0 % (2 au masque) | 0,0 % |

**CE QUE CELA CHANGE POUR LE PRODUIT.** L'accroupissement, la posture et le mot de controle
sont **echantillonnes a l'image-cle**, soit une fois toutes les ~18 s par bipede — pas par
image. Des « intervalles d'etat par joueur » a la cadence de l'image sont donc **impossibles**
pour ces champs : ce que le film permet, c'est un ETAT AU MOMENT DE L'IMAGE-CLE. Seule
l'action de mobilite est datee finement, parce qu'elle voyage en delta.

### 2ter.2 D1 TRANCHEE SUR FILM — ET SANS CONSEQUENCE MESURABLE

| | chemin delta | image-cle | tags | NON NULS |
|---|---|---|---|---|
| `bfecd02b` | `i55` sur **0** des 162 444 records | 205/207 | 0:138 1:15 2:23 3:29 | **67 (32,7 %)** |
| `4f77afc1` | `i55` sur **0** des 378 661 records | 983/995 | 0:730 1:70 2:89 3:94 | **253 (25,7 %)** |

Les deux films confirment la mesure des mini-bobines (23,5 %) : **le `Skip(2)` saute bien, une
fois sur quatre, une charge que le jeu lit** — et il ne le fait QUE sur le chemin d'image-cle,
jamais en delta.

**ET POURTANT RIEN NE CASSE, POUR UNE RAISON MESUREE** : la marche d'image-cle du bipede
s'arrete de toute facon avant d'avoir quoi que ce soit a verifier. Les bloquants, comptes :

```
bfecd02b : i60 simulation-state-component 171 · i57 biped-spartan-ability 18 · i59 …-non-predicted-state 17   (marche complete 1/207)
4f77afc1 : i60 simulation-state-component 859 · i59 …-non-predicted-state 68 · i57 biped-spartan-ability 58   (marche complete 10/995)
```

Aucun oracle de fermeture ne s'exerce au-dela d'`i55`. **D1 est donc reelle, documentee, et
inoffensive tant que la marche s'arrete a `i60`** — elle deviendra bloquante le jour ou `i60`
sera porte. C'est une dette datee, pas un incident.

### 2ter.3 LA GLISSADE EST INACCESSIBLE — ET C'EST UN PRE-REQUIS CHIFFRE, PAS UNE IMPASSE

`i62 biped-slide` est lu **0 fois** sur les deux films. La raison est mesuree, et ce n'est PAS
que le film n'en parle pas : **`i62` est DERRIERE le bloquant**. `i60` arrete la marche, et
`i62` vient apres. Sur le chemin delta, `i62` n'apparait au masque que 2 fois par film.

**Pour publier la glissade, il faut d'abord porter `i60 simulation-state-component`** (et,
accessoirement, `i57` et `i59`). Le cout est chiffre : 171 + 859 records bloques sur les deux
films. Tant que ce n'est pas fait, la glissade n'est pas publiable — et dire « le film ne la
porte pas » serait faux.

### 2ter.4 LE MOT DE 32 BITS N'EST PAS UN CHAMP DE BOUTONS — D7 EST REFUTEE

`i18 +0x544` ne voyage qu'a l'image-cle : **35 records sur 205** (`bfecd02b`) et **180 sur
983** (`4f77afc1`). Ventilation bit a bit, chaque bit contre la composante verticale de la
vitesse au record suivant du meme slot :

| film | bits allumes | frequence par bit | montee qui suit | plancher |
|---|---|---|---|---|
| `bfecd02b` | **32 / 32** | 2,9 % a 14,3 % | 25 % a 100 % (n <= 5) | 5,8 % |
| `4f77afc1` | **32 / 32** | 7,2 % a 20,6 % | 21 % a 62 % | 8,9 % |

**AUCUN BIT MORT, AUCUN BIT DOMINANT.** Un champ de bits de commande aurait la signature
inverse : la plupart des bits jamais allumes (le jeu n'a pas 32 actions), et un ou deux bits
tres frequents (avancer, tirer). Ici les 32 bits sont allumes dans la meme fourchette etroite
et aucun ne predit la montee mieux que ses voisins. **C'est la signature d'un mot OPAQUE a
forte entropie** — jeton, horodatage, hachage — pas d'un champ de boutons.

**D7 est donc REFUTEE, et avec elle la derniere piste « entree repliquee » du saut.** Il ne
reste aucun candidat de forme « bouton » sur l'archetype bipede. Le § 2.9.4 tenait : Theater
rejoue un ETAT, pas des entrees.

> Reserve ecrite : la cadence d'image-cle (~18 s) rend de toute facon impossible d'apparier un
> bit a un saut, qui dure environ une seconde. Meme si un bit de saut existait dans ce mot, ce
> canal ne permettrait pas de le dater. Le negatif ci-dessus ne repose pas sur cette reserve —
> il repose sur l'absence de bit mort — mais elle le double.

### 2ter.5 L'ACTION DE MOBILITE, MESUREE — ET L'ORACLE DE VITESSE QUI REFUTE LE SPRINT

| | `bfecd02b` | `4f77afc1` |
|---|---|---|
| initiations (`flag1`) | **453** sur 9 slots | **1 792** sur 63 slots |
| `flag2` | 0 | 3 |
| identifiant 10 bits present | **0** | **0** |
| `+0x9c` R(2) | 0:342 · **2:111** | 0:1 623 · **2:169** |
| `+0x98` R(7) | 0:342 · 1:74 · 3:37 | 0:1 535 · 1:73 · 3:56 · 5:53 · 8:21 · 10:18 · 13:19 · 15:17 |

**TROIS RESULTATS NETS.**

1. **L'identifiant de 10 bits n'est JAMAIS transmis** — 0 fois sur 2 245 initiations. Le champ
   `+0x08` reste a sa sentinelle : **il ne porte pas l'identite de l'action**, contrairement a
   ce que le § 2.8 donnait pour le candidat le plus probable. Rayé.
2. **`+0x9c` ne prend que DEUX valeurs, 0 et 2** — jamais 1 ni 3. Ce n'est donc pas un enumere
   a quatre actions ; c'est un drapeau a deux etats loge dans deux bits. L'hypothese
   « Sprint / Thruster / Clamber / Slide sur 2 bits » (§ 2.8) est **refutee par les valeurs**.
3. **`+0x98` porte 3 valeurs sur un film et 8 sur l'autre**, avec 0 tres dominant (75 % et
   86 %). C'est le seul champ qui se comporte comme un discriminant d'action — et son domaine
   depend du film, donc du contenu de la partie.

**L'ORACLE DE VITESSE REFUTE LE SPRINT.** La magnitude quantifiee (monotone en vitesse) au
moment de l'initiation, contre les records ordinaires :

| classe | `bfecd02b` p10 / median / p90 | `4f77afc1` p10 / median / p90 |
|---|---|---|
| action de mobilite | 55 / **125** / 164 (n = 449) | 94 / **130** / 189 (n = 1 725) |
| debout | 136 / **211** / 245 (n = 145 875) | 129 / **217** / 250 (n = 324 920) |

**L'action de mobilite se produit PLUS LENTEMENT que la marche ordinaire, sur les deux films.**
Un sprint irait plus vite. Ces initiations ne sont donc pas des sprints : le profil est celui
d'un geste ou l'on RALENTIT — s'agripper a un rebord (escalade), ou amorcer une poussee.
**C'est la premiere mesure du lot qui dit ce que `i54` n'est PAS, et elle est franche.**

### 2ter.6 LE SPRINT — NON TRANCHE, ET DIT COMME TEL

La distribution de la magnitude debout est **resserree** (p10 136, mediane 211, p90 245 sur
`bfecd02b`) : **aucune seconde bosse** ne s'y detache. Le sprint ne se lit donc pas comme une
classe separee de cette seule distribution. Deux raisons possibles, non departagees ici : la
quantification (log/exp) ecrase le haut de la plage, ou la vitesse de sprint n'est pas assez
distante de la marche pour se voir sans dequantification. **Conclusion : non tranche.** Ce qui
le trancherait : dequantifier la magnitude en m/s et comparer aux vitesses connues du jeu.

### 2ter.7 LES CINQ INSTANTS PAR ETAT, POUR L'OEIL DE L'UTILISATEUR

Sur `bfecd02b` (Snowbound, Team Slayer), en **TEMPS DE BARRE THEATER** = temps film depuis le
debut, `mm:ss`. Le `slot` est l'entite, c'est-a-dire UNE VIE — l'attribution vie -> joueur est
le travail de l'index de `replaybuild` et n'est PAS faite ici (le document n'a pas ete cuit :
aucune base ouverte). Les instants sont espaces d'au moins dix secondes pour ne pas donner cinq
fois le meme geste.

| etat | 5 instants (temps de barre Theater) |
|---|---|
| **ACCROUPI** | `02:40` (slot 518) · `03:00` (542) · `03:40` (547) · `04:20` (552) · `04:40` (559) |
| **GLISSADE** | **AUCUN** — inaccessible, voir § 2ter.3 |
| **ACTION DE MOBILITE** | `00:39` (516) · `01:00` (513) · `01:20` (518) · `01:37` (523) · `01:48` (515) |
| **MONTEE (candidat saut, `dirZ > 0,60`)** | `00:19` (512) · `00:36` (518) · `00:46` (517) · `00:57` (515) · `01:08` (521) |

> Les instants « accroupi » viennent des images-cles (cadence ~18 s) ; les « action de
> mobilite » et les « montee » viennent du chemin delta, donc de l'image exacte.

### 2ter.8 CE QUE L'INSTRUMENT A APPRIS SUR LUI-MEME

Deux defauts de harnais ont ete trouves par des ECARTS DE COMPTE, et ils sont consignes parce
que le prochain instrument les referait :

1. **`traverseComponentLoopFrom` ne pose pas `EndBit`** — c'est `TraverseEntity` qui le fait
   apres elle. Sans cette ligne, la fin du DERNIER composant d'un record vaut 0 et ce composant
   n'est jamais lu : `i54`, plus haut index de 459 records, etait vu **5 fois au lieu de 487**.
2. **Le registre d'un film peut nommer un composant SANS le suffixe `-component`** (le dispatch
   accepte les deux orthographes). Comparer au seul nom long fait manquer des composants.

Dans les deux cas, c'est la mesure separee du MASQUE — le denominateur — qui a revele l'ecart.
Un instrument qui ne publie que ses trouvailles ne peut pas se corriger lui-meme.

## 2 quater. CORRECTION DE CAP (2026-09-21) — LA CONCLUSION « IMAGE-CLE SEULEMENT » EST RETIREE

> L'utilisateur, qui fait autorite sur le film, a corrige : ces etats sont connus A L'INSTANT
> PRECIS, « on sait a la milliseconde ce que le joueur fait, on sait meme quand il zoome ». Il a
> raison, et le depot le documentait deja en trois endroits. **La conclusion du § 2ter.1 est
> RETIREE.** Ce paragraphe dit ce qui la remplace, et ce qui reste a faire.

### 2quater.1 CE QUE LE DEPOT DISAIT DEJA, ET QUE LE § 2 TER A CONTREDIT

| source | ce qu'elle dit |
|---|---|
| `RECAP_STATS_EXPLOITABLES.md` § TIER 3 | « ETAT BIPED PAR FRAME (sante, bouclier, velocite, **CROUCH**, position, aim, munitions) — source : **deltas type-0 (~60 fps)** + snapshots keyframe type-2 (~18-20 s) » |
| `HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md` l. 29 | « **per-tick** biped state (health, shield, velocity, **crouch**, position, aim, ammo) » |
| `RECETTE_DECODAGE_FILM_CHUNKS.md` | la table des deserialiseurs : `i18 unit-control FUN_141017084`, `i29 unit-crouch FUN_142ed42a8`, `i21 aiming FUN_14076df7c` |

**L'accroupissement est par tick dans les deltas de type 0.** Le « `i29` a 0,0 % des records
delta » du § 2ter.1 est donc un **DEFAUT D'INSTRUMENT**, pas un fait du film, et il est retire.

### 2quater.2 LE MODELE D'UN ETAT PAR INSTANT QUI EXISTE DEJA : LE ZOOM

`zoom_events.go` lit l'etat de lunette **a l'instant**, et il le fait dans la LISTE
D'EVENEMENTS en tete des paquets delta — pas dans la trame de composants :

```
[1 bit config] [ ( 1 [R(7) type] [3 references gardees] [charge] )* 0 ] [trame de records]
```

`unit_zoom` (type 21) : ~400 000 occurrences sur 1 367 films ; charge `R(2)` = le palier de
lunette + 1 ; pont vers le joueur par la premiere reference (domaine 4) **+ 512 = le slot du
bipede** (63 index sur 64 tombent sur un slot reel, contre 0 sur 64 pour toute autre base) ;
valide contre Theater (6 entrees en lunette sur 6, a moins de 1,2 s).

**ET SA LECON, QUI EST EXACTEMENT LA MIENNE** : « Sept campagnes de mesure ont conclu "aucun
evenement de zoom dans la bobine" parce qu'elles lisaient le type a `payload[0] & 0x7F` — elles
ignoraient le bit de configuration et decalaient donc TOUT d'un bit. » **Quand une mesure rend
zero, le suspect numero un est l'instrument.**

### 2quater.3 CE QUE LE RECENSEMENT DES EVENEMENTS DONNE SUR `bfecd02b` — MESURE ETALONNEE

Par `PacketHeadEventType`, la porte du depot (jamais une arithmetique refaite a la main) :
**31 232 paquets delta, 5 274 portent un evenement en tete (16,9 %), 20 types distincts.**

| type | nom | en tete | part |
|---|---|---|---|
| 36 | `action_weapon_fire` | 2 611 | 49,5 % |
| 82 | `PlayerGameEventSmall` | 578 | 11,0 % |
| 15 | `Script` | 378 | 7,2 % |
| **21** | **`unit_zoom`** | **320** | **6,1 %** |
| 0 | `damage_aftermath` | 277 | 5,3 % |
| 38 | `weapon_reload` | 229 | 4,3 % |
| 9 | `biped_pickup` | 142 | 2,7 % |
| 39 | `biped_throw_initiate` | 76 | 1,4 % |

**Le temoin passe** : `unit_zoom` est bien la, 320 fois, sur ce film. Le canal des instants est
vivant et lisible. Les types 42, 43, 72 et 78 sont a **0 en tete** — et c'est un PLANCHER, pas
un negatif : ce scanner ne lit que le PREMIER evenement de chaque liste.

**`PlayerGameEventSmall` (type 82, 578 occurrences) est le candidat a instruire** : un
evenement de joueur generique, frequent, dont la charge n'est pas portee.

### 2quater.4 L'ETALONNAGE N'A PAS PU SE FAIRE PAR LA PORTE DE PRODUCTION — MESURE A CONSIGNER

Le balayage bipede de production rend **ZERO record** sur `bfecd02b`, `DropSaturated` a vrai
comme a faux, profil MPP du build installe ou non, alors que la marche directe en lit 162 444
avec la meme bande de slots et le meme decoupage d'`i0`. `ScanBipedPositions` a donc une
condition d'entree que ce film ne remplit pas — vraisemblablement les bornes de carte, que
`QuantaOnly` dispense de FOURNIR mais dont le filtre de saturation depend encore.

Deux controles ont malgre tout ete faits :

1. **Le filtre `RequireTag1` n'est PAS la cause.** Sans lui : 162 487 records au lieu de
   162 444, et `i29` passe de 0 a **1**. Le crible d'ancre n'ecarte donc pas la population
   cherchee.
2. **Le gradient du masque est coherent** — `i0` 100 %, `i25` 100 %, `i1` 90,3 %, `i21` 62,4 %,
   `i5` 32,2 % — et le dispatch de production est alle AU BOUT de 162 482 records sur 162 487.

### 2quater.5 CE QUI RESTE, ET C'EST PRECIS

**LE SUSPECT NOMME** : `walkDeltaBipedPayload` est un **CHERCHEUR D'ANCRES** — il balaie le
payload bit a bit et retient ce qui RESSEMBLE a un en-tete de bipede. Ce n'est pas le decodeur
de trame. Le depot en a un vrai : **`DecodeFrameRecords` (`frame_records.go`)**, qui consomme
le preambule du paquet puis lit les records DANS L'ORDRE. `event_list.go` precise sa limite :
il saute la liste d'evenements, donc il fonctionne sur les paquets a liste vide (octet de tete
`0x80..0xBF`) et rate ceux qui portent un evenement — soit, sur `bfecd02b`, **83,1 % des
paquets** (25 958 sur 31 232).

**LA PROCHAINE MESURE, ET ELLE N'EST PAS FAITE ICI** : re-ventiler `ti=35` avec
`DecodeFrameRecords` sur ces 83,1 % de paquets, etalonner sur `i21`/`i0`/`i1`, puis rendre pour
`i29` la cadence reelle et les intervalles d'accroupissement par slot. **Aucune conclusion de
cadence ne doit etre tiree avant cet etalonnage** — celle du § 2ter.1 ne l'a pas ete, et elle
etait fausse.

**A LIRE AVANT**, sur demande de l'utilisateur : `RE_EXE_GHIDRA_FINDINGS.md`,
`PLAN_FILM_ECS_DECODER.md`, et le `thought_log` des 2026-06-03/04/05 (archive Q2).

## 2 quinquies. LE VRAI DECODEUR DE TRAME, ET L ETALON QUI TRANCHE (2026-09-21)

### 2quin.1 LA MESURE PAR `DecodeFrameRecords`

Troisieme instrument (`mouvement_5_3_2d`), sur le decodeur de trame du depot : preambule de
paquet, puis les records **dans l ordre**, contre un `World` amorce par les images-cles du
chunk. Domaine : les paquets a liste d evenements VIDE (`pay[0]&0x40 == 0`), soit **25 958 des
31 232 paquets delta** de `bfecd02b`.

| | valeur |
|---|---|
| paquets cadres | 25 958 |
| trames decodees **sans erreur** | **9 786 (37,7 %)** |
| records de trame | 24 940, dont **10 060 de `ti=35`** |
| etalon | `i0` 62,5 % · `i1` 55,5 % · **`i21` 64,5 %** · **`i25` 95,2 %** |
| `i18` / `i29` / `i54` / `i55` / `i62` | **4 · 1 · 8 · 3 · 1** (soit 0,0 % a 0,1 %) |
| intervalles d accroupi | **0** sur 37 slots |

**DEUX DECODEURS INDEPENDANTS CONVERGENT.** Le chercheur d ancres (162 444 records) et le
decodeur de trame (10 060 records de `ti=35`) donnent la meme reponse : dans la trame de
composants des paquets delta, l accroupissement n est pas la.

**ET LA MESURE PORTE SA PROPRE RESERVE** : 37,7 % de trames decodees sans erreur, ce n est pas
un cadrage sain. Prise seule, elle ne suffirait pas.

### 2quin.2 L ETALON QUI TRANCHE — ET IL N EST NI L UN NI L AUTRE DE MES INSTRUMENTS

Le depot porte une VERITE TERRAIN, obtenue par capture live (Cheat Engine) et consignee dans
`components_position_i0.go` :

> « sur les **15 529 records du masque `{i0,i1,i21,i25}`** dont l oracle de POSITION Rosette
> donne la longueur vraie (**113 bits**), la seule largeur de `i0` pour laquelle les desers
> PORTES de `i1` et de `i21` consomment exactement leurs largeurs vraies (**31 et 25 bits**)
> est **47 bits** : `i1` tombe juste sur 100,0 % des records et `i21` sur 100,0 %. »

Et `unit_weaponstate.go` : « l oracle de position Rosette donne **unit-command-tick = 10 bits
constants** ».

**LE RECORD BIPEDE DELTA DOMINANT EST DONC `{i0, i1, i21, i25}` — 47 + 31 + 25 + 10 = 113 bits.
Position, velocite, visee, tick de commande. L ACCROUPISSEMENT N Y EST PAS.**

Ce n est pas mon instrument qui le dit : c est une capture live du jeu, sur 15 529 records, et
mes deux lecteurs independants retrouvent exactement ce masque (`i0` 100 %, `i25` 100 %, `i1`
90,3 %, `i21` 62,4 % au chercheur d ancres).

### 2quin.3 CE QUE CELA VEUT DIRE, ET CE QUE CELA NE VEUT PAS DIRE

> **AMENDE LE 2026-09-21 PAR LE § 2 SEXIES.** Le record delta DOMINANT porte bien quatre
> composants (l oracle Rosette reste vrai), mais le record RARE en porte plus — et c est lui
> qui porte l accroupissement. La conclusion « l accroupissement n est pas dans la trame » est
> RETIREE : il y est, derriere les largeurs fausses de `ti=0 i0` et des trois `partiel` du
> bipede.

**CE QUE CELA VEUT DIRE** : la phrase « per-tick biped state (health, shield, velocity,
**crouch**, position, aim, ammo) » de `RECAP_STATS_EXPLOITABLES` et du handoff decrit le
VOCABULAIRE de l archetype bipede — ce que le film PEUT porter — et non ce que le record delta
porte a chaque tick. Le record delta dominant porte quatre composants, pas sept.

**CE QUE CELA NE VEUT PAS DIRE** : que le film ignore l accroupissement a l instant.
L utilisateur a raison sur le fond — Theater le rejoue — et le depot montre DEJA ou un etat par
instant se loge quand il n est pas dans la trame : **le canal d evenements**. `unit_zoom` en est
la preuve vivante (~400 000 occurrences, pont vers le slot par domaine 4 + 512, valide contre
Theater 6/6). L accroupissement par instant est donc a chercher **la**, pas dans les masques.

### 2quin.4 `PlayerGameEventSmall` — LE CANDIDAT, ET CE QUE SON ECRIVAIN DIT

578 occurrences en tete sur `bfecd02b` (11,0 % des paquets a evenement), deuxieme type le plus
frequent apres le tir. Son lecteur (`vtable+0x68` = `FUN_14080add8`) :

```c
FUN_14080b30c(param_3);              // initialisation, 0 bit
*(undefined4 *)(param_3 + 0xa0) = 0; // remise a zero d un champ
FUN_14080ae70(param_3, param_4);     // R(32) -> param_3[0]
FUN_14080ae28(param_4);              // une seconde lecture
```

La charge commence donc par **un mot de 32 bits**, suivi d une seconde lecture non encore
relevee. **Aucun sous-type n est nomme a ce stade** : le nommer demande de relever
`FUN_14080ae28` et de ventiler le mot sur le film. C est la prochaine mesure, et elle n est pas
faite ici.

### 2quin.5 LES CINQ INSTANTS, ET POURQUOI ILS NE SONT PAS RE-CALCULES

Les instants d accroupi du § 2ter.7 viennent des images-cles, et ils restent valides pour ce
qu ils sont : les seuls instants d accroupissement que le depot sait dater aujourd hui. Le
decodeur de trame n en produit **aucun** (0 intervalle sur 37 slots), et il serait malhonnete de
publier une liste vide comme un progres. Des instants d accroupi A LA CADENCE DU JEU viendront
du canal d evenements, quand il sera lu.

## 2 sexies. L HYPOTHESE DU PILOTE EST CONFIRMEE : L ACCROUPI PAR INSTANT EST DANS LA TRAME, DERRIERE UNE LARGEUR FAUSSE

> Mesure du 2026-09-21 sur `bfecd02b`, toujours par `DecodeFrameRecords`. Le pilote a pose la
> bonne question : **un delta ne porte `i29` QUE quand l accroupi CHANGE**, donc ces records
> sont rares — et si ce sont AUSSI ceux qui echouent, l accroupi est bien la.

### 2sex.1 LA POPULATION EN ECHEC, ET CE QU ELLE PORTE

**16 172 records desynchronises**, dont seulement **44 de `ti=35`** (0,3 %). La comparaison qui
decide n est pas un compte mais un RAPPORT — la part des masques portant chaque composant parmi
les records EN ECHEC, contre cette meme part parmi les records SAINS :

| composant | part des ECHECS `ti=35` | part des SAINS | facteur |
|---|---|---|---|
| `i18 unit-control` | **18,18 %** (8) | 0,05 % (5) | **x 364** |
| `i29 unit-crouch` | **9,09 %** (4) | 0,04 % (4) | **x 227** |
| `i54 biped-mobility-action` | **22,73 %** (10) | 0,10 % (10) | **x 227** |
| `i55 biped-posture-physics` | **13,64 %** (6) | 0,07 % (7) | **x 195** |
| `i62 biped-slide` | **18,18 %** (8) | 0,03 % (3) | **x 606** |

**LES CINQ COMPOSANTS DE MOUVEMENT SONT SUR-REPRESENTES D UN FACTEUR 200 A 600 DANS LES
ECHECS.** Le « `i29` a 0,0 % » des mesures precedentes etait donc une mesure de la population
SAINE — et la population saine est, par construction, la population PAUVRE : les records
`{i0,i1,i21,i25}` de 113 bits que l oracle Rosette decrit. Les records RICHES, ceux qui portent
un changement d etat, sont precisement ceux qui cassent.

**L hypothese du pilote est confirmee : l accroupissement par instant EST dans la trame.**

### 2sex.2 LE COMPOSANT FAUTIF, NOMME, PAR ARCHETYPE

> **CE TABLEAU EST FAUX ET REMPLACE PAR LE § 2 SEPTIES (2026-09-21).** Les 13 463 « `ti=0 i0` »
> etaient des REJETS DE GENERATION, pas des composants fautifs : sur le chemin delta,
> `DesyncAt = 0` est une sentinelle de rejet et `TypeIndex` n y est jamais pose. La
> sur-representation du § 2sex.1, elle, tient et se renforce.

| archetype | echecs | composant fautif dominant |
|---|---|---|
| **`ti=0`** (game-engine) | **13 686** (84,6 %) | **`i0` : 13 463** — le verrou principal |
| `ti=2` | 424 | `i15` : 328 |
| `ti=5` | 422 | `i22` : 367 |
| `ti=10` | 315 | — |
| **`ti=35`** (bipede) | **44** | **`i59` : 18 · `i60` : 16 · `i57` : 10** |

Cote bipede, les trois fautifs sont exactement les trois composants que `ecs_table.tsv` declare
**`partiel`** : `i57 biped-spartan-ability`, `i59 biped-spartan-ability-non-predicted-state`,
`i60 simulation-state`. Un record qui porte l accroupissement porte aussi ces composants-la, et
la marche casse sur eux — pas sur `i29`.

**LA CHAINE CAUSALE, ECRITE** : `ti=0 i0` casse dans 13 463 paquets → le reste du paquet n est
jamais atteint → les records bipedes RICHES, qui viennent apres dans la trame, sont perdus →
`i29` parait absent. Ce n est pas le film qui se tait, c est la marche qui n arrive pas jusqu a
lui.

### 2sex.3 CE QUE CELA FAIT DU LOT

**L accroupissement par instant n est PAS un item de recherche : c est un ITEM DE GRAMMAIRE**,
et sa liste de travaux est nommee, dans l ordre du rendement :

1. **`ti=0 i0`** — 13 463 echecs a lui seul, 83 % de tous les echecs du film. Tant qu il n est
   pas bit-exact, aucune trame riche ne se lit.
2. **`ti=35 i57`, `i59`, `i60`** — les trois `partiel` du bipede, 44 echecs sur ce film, mais ce
   sont eux qui gardent l acces a `i29`, `i18`, `i54`, `i55` et `i62`.
3. `ti=2 i15` (328) et `ti=5 i22` (367), secondaires.

**CE QUI EST DONC RETIRE** : la conclusion du § 2quin.3 selon laquelle « le record delta
dominant porte quatre composants, donc l accroupissement n y est pas ». Le record DOMINANT en
porte quatre — l oracle Rosette reste vrai — mais le record RARE en porte plus, et c est lui qui
compte pour l accroupissement. Les deux enonces ne se contredisent pas ; le second manquait.

### 2sex.4 CE QUI N A PAS ETE FAIT, ET POURQUOI

Le pilote demandait aussi (2) l ecrivain du masque delta — comment le jeu decide d inclure `i29`
— et (3) la ventilation du `R(32)` de `PlayerGameEventSmall`. **Ni l un ni l autre n est fait.**
Le resultat de (1) les deprioritise : il n y a plus de mystere sur l endroit ou vit
l accroupissement par instant, donc plus besoin de l ecrivain du masque pour le trouver, ni du
canal d evenements comme piste de remplacement. Ce qui reste est un travail de largeurs, et il
commence par `ti=0 i0`.

## 2 septies. CORRECTION DU § 2 SEXIES — `ti=0 i0` N ETAIT PAS LE VERROU, ET VOICI LA VRAIE LISTE

> Ouverture du lot de grammaire 5.3.3. Premiere action : re-verifier le classement des echecs
> AVANT de toucher une largeur. Elle a trouve une erreur de lecture, et la liste change.

### 2sept.1 L ERREUR, ET SA CAUSE EXACTE

Le § 2sexies donnait `ti=0 i0 game-engine-team-mapping` a **13 463 echecs**, « le verrou
principal ». **C EST FAUX.** Sur le chemin delta, `DecodeFrameRecords` pose, quand le test de
generation echoue :

```go
rec.Trace = EntityTrace{DesyncAt: 0, EndBit: br.BitPos()}
rec.DesyncAt = 0
break   // aucun composant n est lu, et rec.TypeIndex n est JAMAIS pose
```

`DesyncAt = 0` y est une **SENTINELLE de rejet**, pas un index de composant ; et `TypeIndex`
reste a sa valeur nulle, c est-a-dire **0**. Mon histogramme lisait donc « archetype 0,
composant 0 » la ou le decodeur disait « je n ai meme pas regarde ce record ».

**LE DISCRIMINANT QUI MANQUAIT** : un rejet de generation ne franchit AUCUN composant
(`len(Trace.Comps) == 0`). Avec ce test, la ventilation se separe proprement.

### 2sept.2 LA VENTILATION CORRIGEE, ET LA CORRECTION DE HARNAIS QU ELLE A ENTRAINEE

Seconde erreur trouvee dans la foulee : le monde (`World`) etait remis a neuf **a chaque
chunk**, donc les liaisons slot -> archetype posees par les chunks precedents etaient perdues.
Le monde persiste desormais sur tout le film.

| | avant correction | apres |
|---|---|---|
| trames decodees sans erreur | 9 786 (37,7 %) | **10 512 (40,5 %)** |
| records `ti=35` | 10 060 | **11 150** |
| **rejets de GENERATION** (monde, pas grammaire) | 13 463 | **12 316** |
| **desynchronisations REELLES de grammaire** | 2 709 | **3 130** |

### 2sept.3 LA VRAIE LISTE DES COMPOSANTS FAUTIFS, PAR RENDEMENT

| archetype | echecs reels | composant fautif dominant |
|---|---|---|
| `ti=2` (game-engine) | 442 | **`i15 managed-engine-timers-component` : 331** |
| `ti=5` (joueur) | 429 | **`i22 player-aim-assist-component` : 371** |
| `ti=10` (managed-object) | 354 | **`i5 managed-object-navpoint-component` : 228** |
| `ti=0` | 265 | (disperse) |
| `ti=18` | 145 | — |
| **`ti=35` (bipede)** | **69** | **`i60 simulation-state` : 38 · `i59` : 19 · `i57` : 12** |

**`game-engine-team-mapping` n apparait plus.** Les trois `partiel` du bipede, eux, tiennent :
ce sont bien `i57`, `i59` et `i60` qui gardent l acces aux composants de mouvement.

### 2sept.4 CE QUI NE BOUGE PAS : LA SUR-REPRESENTATION

Elle se RENFORCE apres correction, ce qui est le meilleur signe qu elle est reelle :

| composant | part des ECHECS `ti=35` | part des SAINS | facteur |
|---|---|---|---|
| `i18 unit-control` | **15,94 %** | 0,13 % | **x 123** |
| `i29 unit-crouch` | **15,94 %** | 0,13 % | **x 123** |
| `i55 biped-posture-physics` | **24,64 %** | 0,13 % | **x 190** |

La conclusion du § 2sexies tient donc **entierement sur ce point** : l accroupissement par
instant est dans la trame, dans les records riches, et les records riches echouent. Seule la
LISTE DES COMPOSANTS A CORRIGER etait fausse.

### 2sept.5 CE QUI RESTE LE PREMIER OBSTACLE, ET CE N EST PAS UNE GRAMMAIRE

**12 316 rejets de generation contre 3 130 desynchronisations reelles** : les quatre cinquiemes
des trames perdues le sont parce que le monde ne connait pas encore la liaison slot ->
archetype, pas parce qu une largeur est fausse. Le monde s amorce aux images-cles ET aux
records NEW des deltas — mais un record NEW n est lie que si sa marche est PROPRE, et une trame
qui casse tot n en lie aucun. C est un amorcage circulaire.

**AUCUNE LARGEUR N A ETE TOUCHEE.** Corriger une grammaire avant d avoir leve cet amorcage
reviendrait a mesurer le gain sur une population de trames que le monde mutile encore.

### 2sept.6 L ORDRE DE TRAVAIL QUE CETTE MESURE IMPOSE

1. **L AMORCAGE DU MONDE** — 12 316 trames, quatre fois le total des desyncs de grammaire.
   Question a instruire : d ou la production tire-t-elle ses liaisons (les `world_dump` que
   `frame_records.go` mentionne), et peut-on les charger avant la premiere trame ?
2. `ti=2 i15 managed-engine-timers` (331), `ti=5 i22 player-aim-assist` (371),
   `ti=10 i5 managed-object-navpoint` (228).
3. `ti=35 i60`, `i59`, `i57` (69 au total) — les gardiens des composants de mouvement, et la
   cible finale du lot 5.3.

## 2 octies. ETAPE (1) — L AMORCAGE DU MONDE N EST PAS LA CAUSE, ET C EST MESURE

> Lot de grammaire, etape (1). Deux corrections tentees, et une mesure qui les refute toutes
> les deux. **Aucune largeur touchee, aucune perte.**

### 2oct.1 LES DEUX CORRECTIONS TENTEES

1. **Le monde persiste sur tout le film** (deja fait au 5.3.3.0) : 37,7 % -> 40,5 % de trames
   saines.
2. **Liaison par SLOT, generation neutralisee** : `BindWildcard` au lieu de `BindFull` pour
   chaque entite d image-cle — la porte que `world.go` prevoit pour « une liaison slot ->
   archetype dont la GENERATION est INCONNUE ». C est exactement la forme demandee (« la
   liaison ne doit pas dependre d une marche propre »).

**RESULTAT : AUCUN CHANGEMENT. Pas un chiffre ne bouge.** 25 958 paquets cadres, 10 512 trames
saines (40,5 %), 12 316 rejets, 3 130 desyncs reelles, memes fautifs, meme etalon.

### 2oct.2 CE QUE LA MESURE DE COUVERTURE DIT, ET POURQUOI ELLE TRANCHE

Si la generation etait en cause, les slots rejetes seraient les memes que ceux vus dans les
records sains. Mesure :

| | valeur |
|---|---|
| slots DISTINCTS rejetes | **4 568** |
| dont vus AUSSI dans un record sain | **28 (0,6 %)** |
| slots les plus rejetes | 137 (102), 1041 (82), 2616 (63), 3933 (58), 5268 (54) — **tous inconnus** |

**99,4 % des slots rejetes n apparaissent JAMAIS dans un record sain.** Ce ne sont donc ni des
generations qui avancent, ni des entites que l amorcage aurait oubliees : **4 568 slots
distincts, etales de 137 a 5 268, c est plus d entites que n en porte un match.** Ce sont des
identifiants LUS DANS DU BRUIT.

### 2oct.3 LA CONCLUSION, ET ELLE DEPLACE LE PROBLEME

`DecodeFrameRecords` rend la main au PREMIER record en echec. Les 12 316 rejets sont donc
12 316 paquets dont le **PREMIER** record echoue deja son test de generation — avec un slot qui
n existe pas. **Le cadrage de ces paquets est faux des le premier bit de trame**, et tout ce
qui suit est du bruit.

**L amorcage du monde n est donc PAS la cause, et la cible « rejets < 1 000 » n est pas
atteignable par lui.** Le vrai sujet est le CADRAGE : pourquoi, sur 60 % des paquets a liste
d evenements vide, la trame ne commence-t-elle pas ou le preambule le dit ?

Pistes que la mesure designe, aucune instruite ici :

- le preambule vaut `DefaultPacketPreambleBits = 2` pour tous ces paquets ; `event_list.go`
  dit que ces 2 bits sont `[config][continuation=0]` — le filtre `pay[0]&0x40 == 0` teste bien
  le bit de continuation, mais rien ne garantit que le preambule soit de 2 bits pour TOUS ;
- `bpkCalibre` (`biped_pickup_research_test.go`) BALAYE deja `IDLowBits` sur les paquets a
  liste vide pour trouver la largeur qui maximise le taux de trames exactes : **le depot a
  donc deja un instrument de calibration de cadrage**, et il faudrait le rejouer sur ce film
  avant toute grammaire.

### 2oct.4 AUCUNE PERTE, ET RIEN N EST TOUCHE

Le seul octet modifie est dans un `_test.go` sous tag `research` (`BindFull` -> `BindWildcard`,
plus la mesure de couverture). `grammar.Rev`, `facts.Rev`, le ratchet 0.A.3 et les fixtures
sont inchanges par construction. Gates sans decodage verts.

## 2 nonies. LE CADRAGE : `bpkCalibre` DIT 9, L ECRIVAIN DIT 13 — ET C EST L ECRIVAIN QUI A RAISON

> Etape (1) du cadrage. La calibration a ete jouee, puis ARBITREE par l ecrivain, et
> l arbitrage a evite un correctif qui aurait detruit le decodage des bipedes en rendant tous
> les gates verts.

### 2non.1 LA CALIBRATION DU DEPOT, SUR `bfecd02b`

`TestBipedPickupCalibration` (`bpkCalibre`), balayage d `IDLowBits` sur les paquets a liste
vide, 3 000 paquets :

| `IDLowBits` | trames EXACTES | profondeur (record/paquet) |
|---|---|---|
| **9** | **97,5 %** | **1,02** |
| 10 | 5,3 % | 1,46 |
| 11 | 85,2 % | 1,04 |
| 12 | 11,2 % | 2,08 |
| **13 (defaut)** | **5,1 %** | 1,63 |
| 14 | 8,2 % | 1,28 |
| 15 | 28,3 % | 1,05 |
| 16 | 17,2 % | 1,15 |

Temoins de decalage au cadrage retenu : +0 bit **79,0 %**, +1 **0,0 %**, +2 **0,1 %**, +3
**0,0 %**. **Le cadrage au bit 2 est donc juste** — ce n est pas la position de la trame qui est
en cause.

### 2non.2 CE QUE `IDLowBits = 9` FAIT VRAIMENT, MESURE

| | defaut 13 | calibre 9 |
|---|---|---|
| trames decodees sans erreur | 10 512 (40,5 %) | **25 182 (97,0 %)** |
| rejets de generation | 12 316 | **405** |
| desyncs reelles | 3 130 | **371** |
| **records `ti=35`** | **11 150** | **1** |
| etalon `i21` | 64,3 % | **0,0 %** |

**97 % de trames « saines » et UN SEUL record de bipede.** L etalon `i21` a REFUSE de publier —
la garde posee au paragraphe 2 quinquies a fait exactement son travail. Un correctif adopte sur
le seul taux de trames aurait rendu tous les gates verts en detruisant le decodage des joueurs.

### 2non.3 L ARBITRAGE : L ECRIVAIN, ET IL EST DEJA DANS LE DEPOT

`readRecordID` porte `FUN_1406d3140(_, _, 7, _)` — **categorie 7** de la table de plages du jeu.
Et `varwidth.go` ecrit, apres relecture de `FUN_140d10bb0` :

> « `W` ne vaut 13 que pour les categories 0, 1, **7** et 8 — celles dont la plage derive de
> `DAT_144706100` (0x1FFF, et `bitLen(0x1DFF) == bitLen(0x1FFF) == 13`). »

`varWidthRange(7)` retombe sur `varWidthDefaultRange = 0x1FFF`, donc `varWidthBits(7) = 13`.
**LA LARGEUR D ID BAS EST 13, ET ELLE VIENT DE L ECRIVAIN.** Ce n est ni une constante « qui
marche » ni une ligne de profil : c est une categorie de la table du jeu.

### 2non.4 POURQUOI `bpkCalibre` REND 9, ET CE QUE CELA APPREND DE L ORACLE

Son critere d exactitude est « la trame consomme le payload a moins d un octet pres ». **Une
largeur TROP PETITE le satisfait de facon degeneree** : la profondeur tombe a **1,02
record/paquet** — un record par paquet, la ou un paquet delta a 60 Hz en porte plusieurs
dizaines. La trame « ferme » parce qu elle lit un seul gros record et s arrete, pas parce
qu elle est juste.

**LECON, ET ELLE VAUT POUR TOUT LE CHANTIER** : un oracle de FERMETURE ne prouve pas une
largeur ; il lui faut un oracle de CONTENU (ici : le compte de records par paquet, et la
presence d `i21`). `bpkCalibre` reste valide pour ce qu il fait — departager des cadrages a
largeur EGALE — mais il ne peut pas arbitrer une largeur.

### 2non.5 CONSEQUENCE POUR L ETAPE (2)

**IL N Y A RIEN A CORRIGER DANS `frame_records.go`.** La largeur d ID y est deja celle de
l ecrivain. L etape (2) telle qu elle etait prevue — « corriger le cadrage » — **n a pas
d objet**, et aucun octet de production n a ete touche.

**LA CAUSE DES 12 316 REJETS RESTE DONC OUVERTE**, et deux hypotheses sont eliminees pour de
bon : ce n est pas l amorcage du monde (paragraphe 2 octies), ce n est pas la largeur d ID.
Ce qui reste a instruire, dans l ordre : le PREAMBULE (les 2 bits valent-ils 2 bits pour TOUS
les paquets a liste vide ?), et surtout **`IDBase`** — `FUN_1406d3140` rend
`(queue << 30) | (base + valeur)` avec une base de 0x200 / 0x300 / 0x400 **selon la
categorie**, et `varwidth.go` dit explicitement que cette base **n est pas portee** et que la
changer « se juge au gate de decodage ». Une base fausse decale TOUS les identifiants — c est
exactement le symptome mesure : des slots qui n existent pas.

## 2 decies. `IDBase` — LA TABLE DU JEU, LUE EN ENTIER, ET UN NEGATIF QUI FERME LA PISTE

> Troisieme hypothese du cadrage, instruite chez l ecrivain. **Negatif : le port etait deja
> exact. Aucun octet de production touche.** Et la lecture rend au passage une table complete
> que le depot n avait pas, plus une validation croisee du zoom.

### 2dec.1 L ECRIVAIN DE LA TABLE, LU EN ENTIER

`FUN_1406d3140` lit sa base et sa plage dans une table indexee par categorie :

```c
uVar7 = DAT_144706100;                            // plage par defaut
if (DAT_144706104 != '\0') {
    uVar8 = (&DAT_1451f98d0)[param_3 * 2];        // BASE de la categorie
    uVar7 = (&DAT_1451f98d4)[param_3 * 2];        // PLAGE de la categorie
}
```

Et `FUN_140d10bb0` REMPLIT cette table, categorie par categorie (`piVar2` = la plage,
`piVar2[-1]` = la base, boucle `iVar3` de 0 a 8) :

| categorie | BASE | PLAGE | largeur `bitLen(plage)` |
|---|---|---|---|
| 0 | `0x200` | `0x1DFF` | 13 |
| 1 | `0x200` | `0x1DFF` | 13 |
| 2 | `0x200` | `0x100` | 8 |
| 3 | **`0x300`** | `0x100` | 8 |
| 4 | `0x200` | `0x200` | 9 |
| 5 | **`0x400`** | `0x100` | 8 |
| 6 | `0` | `0x200` | 9 |
| **7** | **`0`** | **`0x1FFF`** | **13** |
| 8 | `0` | `0x1FFF` | 13 |

Les PLAGES concordent exactement avec `varWidthRange` du depot. **Les BASES, que `varwidth.go`
declarait explicitement NON PORTEES, sont desormais lues** — et elles valent bien 0x200 / 0x300
/ 0x400, mais **pas pour toutes les categories** : les categories 6, 7 et 8 ont une base NULLE.

### 2dec.2 LE NEGATIF : LE PORT ETAIT DEJA EXACT

`readRecordID` porte la categorie **7**. Sa base est **0**, sa largeur **13**. Or
`DefaultFrameConfig()` rend deja `IDLowBits: 13, IDBase: 0`.

**LES DEUX VALEURS DU PORT SONT CELLES DE L ECRIVAIN.** Il n y a rien a corriger, aucune mesure
a refaire, aucun gate a jouer : changer `IDBase` reviendrait a s ecarter de l ecrivain. La piste
est FERMEE, et c est un negatif ecrit, pas un abandon.

**TROIS HYPOTHESES SONT DESORMAIS ELIMINEES** pour les 12 316 rejets : l amorcage du monde
(§ 2 octies), la largeur d ID (§ 2 nonies), la base d ID (ici). Toutes trois par l ecrivain ou
par la mesure, aucune par lassitude.

### 2dec.3 VALIDATION CROISEE GRATUITE : LA BASE DU ZOOM ETAIT UNE MESURE, ELLE EST MAINTENANT UNE GRAMMAIRE

`zoom_events.go` porte `zoomSlotBase = 512`, obtenue par FORCE BRUTE : « base 512 : 63 index
sur 64 tombent sur un slot bipede reellement vu dans le film (98 %) ; bases 0, 256, 768, 1024 :
0 sur 64 ». La premiere reference d `unit_zoom` est de **domaine 4**.

**Et la categorie 4 de la table du jeu a pour base `0x200` = 512.**

La constante que sept campagnes avaient cherchee empiriquement est donc la base de sa categorie,
lue chez l ecrivain. Ce n est pas une coincidence a 1 chance sur 4 : c est la confirmation que
le « domaine » d une reference d evenement EST la categorie de `FUN_1406d3140`, et que la table
ci-dessus vaut pour les references d evenements comme pour les identifiants de record.

**CE QUE CELA OUVRE, ET QUI N EST PAS DE CE LOT** : les domaines 7 et 8 d `unit_zoom` ont une
base de 0 et une largeur de 13 ; les domaines 3 et 5, des bases 0x300 et 0x400. Un lot
d evenements pourrait porter ces bases au lieu de les mesurer.

### 2dec.4 SUITE, DITE COMME CONVENU

La cause des 12 316 rejets reste ouverte, et les trois suspects nommes sont tombes. **Je passe
donc a `ti=35 i60`, `i59`, `i57` sur la population SAINE actuelle** (11 150 records de bipede,
etalon `i21` a 64,3 %), en le disant : ces trois composants gardent l acces a `i29`, `i18`,
`i54`, `i55` et `i62`, et ils sont mesures fautifs 38, 19 et 12 fois.

## 2 undecies. LE PREAMBULE NE DISTINGUE RIEN — ET `ti=35 i60` A UN PREDICAT ENTIEREMENT LISIBLE

### 2und.1 LE PREAMBULE : QUATRIEME HYPOTHESE, QUATRIEME NEGATIF

Mesure sans hypothese sur `bfecd02b` : la distribution de l octet de TETE et de la TAILLE des
paquets, **10 512 sains contre 12 316 rejetes**.

| octet de tete | sains | rejetes |
|---|---|---|
| `0x80` | 139 (1,32 %) | **0** |
| `0x88` | 1 | 1 |
| `0x89` | 28 (0,27 %) | 20 (0,16 %) |
| `0x8A` | 7 | 5 |
| **`0xA0`** | **10 337 (98,34 %)** | **12 290 (99,79 %)** |

**AUCUNE valeur n est propre aux rejetes.** `0xA0` ecrase les deux populations ; la seule
valeur exclusive (`0x80`, 139 paquets) est propre aux SAINS. Les tailles ne separent pas
davantage : les rejetes sont seulement un peu plus GROS (129 paquets de moins de 64 octets
contre 1 964 chez les sains), ce qui s explique sans hypothese — un gros paquet porte plus de
records, donc plus d occasions de casser.

**LA PISTE DU PREAMBULE EST FERMEE.** Quatre hypotheses eliminees pour les 12 316 rejets :
amorcage du monde, largeur d ID, base d ID, preambule. Aucune par lassitude.

### 2und.2 `ti=35 i60 simulation-state` — L ECRIVAIN, ET LA QUEUE N EST PAS UN MYSTERE

`ecs_table.tsv` le declare `partiel` : « structure connue ; **la queue depend d un predicat sur
les vecteurs decodes** ». Lecture de `FUN_142ED6D88` :

```c
FUN_140c1e79c(param_2);                               // (R1[R19] + R8)
cVar1 = FUN_140501798(param_1 + 0xb, param_1 + 0xe);  // LE PREDICAT
if (cVar1 != '\0') {
    FUN_14076e494(param_2, param_1 + 0x11, 0x10, 0, 0, 0);   // la QUEUE : position, axe 16 bits
    ...
}
```

**LE PREDICAT, LU EN ENTIER** (`FUN_140501798`), sur les deux vec3 deja decodes :

```
orthonormes(v1, v2) :=
      | ‖v1‖² − 1.0 | < 0.001   et fini
  et  | ‖v2‖² − 1.0 | < 0.001   et fini
  et  | v1·v2 − 0.0 | < 0.001   et fini
```

Constantes relues dans l image, aucune devinee :

| adresse | valeur | role |
|---|---|---|
| `DAT_143cd8370` | **0.0f** | le produit scalaire vise |
| `DAT_143cd8374` | **1.0f** | la norme visee (la meme constante qu au § 2.1) |
| `DAT_143cd8380` | **`0x7FFFFFFF`** | masque de valeur absolue |
| `DAT_143cd84bc` | **0.001f** | l epsilon |

C est un **test d orthonormalite a 10⁻³** : la queue n est lue que si les deux vecteurs forment
une base orthonormee valide.

**CE QUE CELA CHANGE, ET C EST DECISIF** : le predicat porte sur des valeurs **DEJA DECODEES DU
FLUX**, pas sur un octet d etat RAM. Il est donc **entierement calculable par le decodeur**, et
`i60` peut devenir bit-exact — contrairement a `i57`, dont `ecs_table` dit que son etiquette 3
est « gardee par des octets d etat RUNTIME : desync PROPRE ».

**CHEMIN DE PORT, ECRIT** : decoder les deux vec3 (ils le sont deja : `4 x R(16)` puis
`4 x R(16)` du corps), evaluer `orthonormes`, et ne lire la queue `FUN_14076e494(..., 0x10)`
que si le predicat tient. Aucune constante « qui marche » : les quatre valeurs viennent de
l image.

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
  **REFUTEE LE 2026-09-21 (§ 2ter.4)** : les 32 bits sont TOUS allumes, dans une fourchette
  etroite (7,2 a 20,6 % sur `4f77afc1`), et aucun ne predit la montee mieux que ses voisins. Un
  champ de boutons aurait des bits MORTS et un ou deux bits dominants. C'est un mot opaque a
  forte entropie. **Il ne reste aucun candidat de forme « entree » sur le bipede.**
- **D8 (5.3)** — **LE BALAYAGE BIPEDE DE PRODUCTION NE LIT QUE SIX COMPOSANTS, ET S'ARRETE AU
  PREMIER AUTRE.** `scanRecordDirs` (`offline_aim.go`) modelise `i1`, `i2`, `i3`, `i4`, `i5` et
  `i21`, puis rend la main (« composant non modelise -> curseur non fiable »). Tout ce qui est
  au-dela d'`i21` — donc TOUS les composants de mouvement — n'est JAMAIS lu sur le chemin delta
  en production. Ce n'est pas un defaut : ce balayage ne cherche que les positions et les
  directions. Mais cela veut dire qu'un port qui voudrait publier l'action de mobilite devra
  faire marcher la boucle COMPLETE sur les records delta, ce que 5.3.2 a prouve faisable
  (100 % de marches completes sur 541 105 records). **NON TRAITEE** : c'est le premier item de
  chiffrage de 5.3.3.
- **D9 (5.3)** — **`i54` PORTE DEUX CHAMPS DONT LE DOMAINE MESURE CONTREDIT L'HYPOTHESE DE
  L'ECRIVAIN.** L'identifiant de 10 bits n'est transmis **0 fois sur 2 245 initiations**, et
  `+0x9c` ne prend que **deux** valeurs (0 et 2), jamais 1 ni 3. L'hypothese « enumere a quatre
  actions » du § 2.8 est refutee par les valeurs ; seul `+0x98` se comporte en discriminant, et
  son domaine varie d'un film a l'autre (3 valeurs contre 8). **NON TRAITEE** : nommer les
  classes demande de croiser `+0x98` avec la carte et le geste, ce qui est un lot en soi.
