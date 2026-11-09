# SONDE P1-S3 — Le tir continu est écrit dans la VUE DE CONTRÔLE (2026-09-23)

Plan : `../PLAN_RETOURS_REJEU_2026-09-23.md` §4.2 P1 (S3), §4.5 M4b. Suite de `SONDE_P1_tir_continu.md`
(branche `feat/rr-sondes`). Worktree `LevelUp-wt-rr-ghidra`, branche `feat/rr-ghidra`, base `740a09c7d`.
Ghidra (HaloInfinite.exe, base 0x140000000) en LECTURE SEULE par le pont HTTP, sous la voie `ghidra` ;
un seul film (81c02726), sous la voie `film`, aucune écriture dans `data/`.
Instruments : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/p1s3_vuec_*_research_test.go`
(tag `research`, commandes au §10).

Légende : **[M]** mesuré · **[D]** déduit · **[H]** hypothèse.

---

## 0. Verdict en sept lignes

1. **Le tir continu EST dans le film [M+D]** : pas dans un record d'événement ni dans un composant
   d'entité, mais dans la **VUE DE CONTRÔLE** (vue C, rang 2 de CHAQUE trame delta,
   `replication_control_view.cpp`), entrée `kind 0` du joueur (`FUN_1406d0388`), **bloc d'action**
   (`FUN_1406d025c`, +0x18 du bloc de 0x68 octets `FUN_1406cd860`) : **m0 = R(3)** (main 0, gâchettes
   d'entrée 0..2, MSB = entrée 0 = gâchette principale), m2 = R(3) (main 1), m4 / m5 = R(2) (barillets).
   Le tireur = l'**index de contrôle R(5)** = l'index de joueur du roster (G MONEY = 2).
2. **Pourquoi aucun record 36 [M, Ghidra]** : `action_weapon_fire` n'est émis qu'au bout du tir d'un
   barillet (`FUN_14202f3a0` → `FUN_141fd3b00` → `FUN_141fd8460(0x1ffffffff, 0x24, …)`) et seulement si
   `FUN_140de87fc` rend l'octet 1 = 1. Pour un barillet de **type de prédiction 1 ou 3**
   (`barillet+0x70`, short) c'est un **seau à jetons** (`état+0x40`, débit et capacité `barillet+0x74`,
   `+0x78` sur le serveur) : à débit nul, zéro record. Le **numéro de tir** (`FUN_141e2f590(joueur)+0x10`,
   8 bits) est incrémenté à CHAQUE tir (`0x14202ff65`) sans condition sur la décision d'émission
   (`CALL FUN_140de87fc` @0x14202f6a3), qui ne garde que l'appel d'émission (@0x1420331b8) : d'où ses sauts.
3. **Ce qui remplace le record [M, Ghidra]** : l'écrivain de la vue C (`FUN_1406d1134`) pose le bit
   (main, entrée) de m0 pour chaque gâchette TENUE dont le barillet est de type 1 **et dont le dernier tir
   n'a pas été émis** (`FUN_140eb2f20` = `état+0xd` bit 2), et le bit (main, barillet) de m4 pour chaque
   barillet de type 3 en tir. Canal complémentaire du record : l'un ou l'autre, jamais les deux.
4. **Theater [D, Ghidra]** rejoue ce bit comme une gâchette tenue : `FUN_14076b838` (code 0xd) →
   `FUN_1407ff8cc` → `FUN_1406dc5b8` (m0 bit 0 → drapeau de commande 31) → `FUN_1406db688` (unité+0x308) →
   `FUN_1406dba04` (bits 29..34 → arme+0x2fa, bits 2..5) → `FUN_1407fb92c` (gâchette « tenue ») →
   `FUN_1407fa928` fait tirer le barillet À SA CADENCE (tag) → `FUN_1408a397c` (effet, son) et
   `FUN_14202f3a0` (projectiles). **Les instants des tirs ne sont pas dans le film : Theater les simule
   depuis l'état de gâchette et la cadence de l'arme.**
5. **Sur 81c02726 [M]** : les 1 050 entrées qui tirent de tout le film sont de l'index 2 (G MONEY),
   **1 050 / 1 050 dans des paquets validés** par l'oracle de fermeture, toutes `m0 = 4` (gâchette
   principale) ; 994 dans ses deux épisodes de Ghost, 56 dans une troisième monture non publiée [D] ;
   aucun autre joueur. Gate : **6 / 6 frags** précédés d'entrées qui tirent (toutes les entrées lues
   des 2 s qui précèdent tirent, sauf à 2402 où la rafale commence à 2395), **0 au témoin −60 s**.
6. **Couverture [M]** : la vue C n'est atteinte que si la vue B clôt ; marche validée seule : 29 % et
   55 % des trames de ses épisodes (3 frags sur 6 sans entrée lue). Un balayage par la fin, étalonné
   (99,87 % juste quand il parle), porte la couverture à 83 % et 91 %.
7. **La réfutation de P1 (vue C) ne tient pas [D]** : le bloc est FERMÉ par construction aux tirs d'armes
   à coup (types 0/2/4, qui ont leur record 36) ; et la lecture de P1 refusait trois branches résolubles
   et lisait faux deux sous-lecteurs du bloc (§6).

---

## 1. L'émission d'`action_weapon_fire` (question 1)

Types d'événement (registre `FUN_140e453b4`, `mgr+0x210+type*8`) : **35 = `request_weapon_fire`**
(vtable `0x143d0add0`, objet `0x144724de0`), **36 = `action_weapon_fire`** (vtable `0x143d0aca0`, objet
`0x144724dd8`). Vtable du 36 : `+0x08` nom (`0x14080c1f0` → chaîne `0x143c97b48`), `+0x58` domaines
des réfs (`0x14080a048`), `+0x60` sérialiseur `FUN_142f193e4` (écrit `+0x01` par `FUN_141fd0890`),
`+0x68` désérialiseur `FUN_14080c1f8`, `+0x78` application `FUN_14080a18c` → `FUN_1407e00ac`.

Chemin [M, lecture du code] :

| étape | fonction | ce qui compte |
|---|---|---|
| tir d'un barillet | `FUN_14202f3a0` (appelé par `FUN_1420148e0` ← `FUN_14051a7f0` pour les armes à tir en attente, `arme+0x2f8` bit 0, posé par `FUN_1408a397c`) | numéro de tir : si `param_4 == NULL` (tir local, pas un rejeu d'événement), `b = FUN_141e2f590(joueur)+0x10 ; +0x10 = b+1` (8 bits) — À CHAQUE TIR |
| décision | `FUN_140de87fc(&out, arme, barillet, état_barillet, déf_barillet, unité, …)` | `déf+0x70` = type de prédiction ; 2 et 4 : émet ; 1 et 3 : émet ssi `état+0x40 <= (int)N − 1` **et** `clamp01(état+0x2c) < déf+0x7c`, N = `FUN_140de8a00` = `déf+0x74` (ou `déf+0x78` si mode serveur et ≥ 0), puis `état+0x40 += 1`, `état+0xd |= 4` ; 0 : émet sauf cas `bVar5` |
| seau | `FUN_1407fa928` | `état+0x40 -= (int)N × dt` (plancher 0) : débit N par seconde |
| gardes | `FUN_14202f3a0` fin | `cStack_11d56` = octet 1 de la décision, lu seulement si `FUN_140494ea0()` (partie en réseau) |
| émission | client (mode 2) : `FUN_141fd3d90` → `request_weapon_fire` (0x23) ; autorité (mode 3) : `FUN_141fd3b00` → `FUN_141fd8460(0x1ffffffff, 0x24)` | la charge (`FUN_141fd2b60`, 0x328 o) porte `+0x01` = le numéro de tir courant |

**Arme à son de COUP** (mortier du Wraith, canon du Scorpion, missiles du Wasp, Gungoose) : un record 36
par tir [M : P1, 23 tirs à coup sur 8a485699]. **Arme à son en BOUCLE** (Ghost, canons de la Banshee,
LMG…) : zéro record [M : P1] ; le type de prédiction 1/3 à débit nul est la cause [D : le seul chemin du
code qui supprime l'émission ; valeurs des tags non lues ici, cf. §7 M4b.0].

**Pourquoi le numéro avance sans record** [M] : l'incrément (`CALL FUN_141e2f590` @0x14202ff27, puis
`MOV byte ptr [R9+0x10],AL` @0x14202ff65) ne dépend pas de la décision (`CALL FUN_140de87fc`
@0x14202f6a3), qui ne garde que l'émission (`CALL FUN_141fd3b00` @0x1420331b8, `FUN_141fd3d90`
@0x14203326d) ; il est sauté seulement quand le tir rejoue un événement reçu (`param_4 != NULL`).
**Contrôle [D]** : sur 81c02726, rafales de l'épisode 1 (§4.4) = 40,6 s de gâchette tenue ; à la
cadence du Ghost (7,5 coups/s, V3F) ≈ 305 tirs ; le premier record de G MONEY porte `n = 56` =
312 mod 256 → **312 tirs, 7,7 /s** : le compteur a fait un tour ; P1 ne pouvait pas trancher 56 / 312,
la vue C tranche.

## 2. Ce que le film porte : grammaire complète de l'entrée (question 2, côté film)

Paquet delta = `[config R(1)][vue A][vue B][vue C]` (`FUN_142987460`, ordre prouvé au lot 5.14). Vue C
(`FUN_1406cf548`) : `{ R(1) ; kind R(2) }` ; kind 0 = `FUN_1406d0388` :

| champ | largeur | adresse / preuve | port de production |
|---|---|---|---|
| index de commande | R(1) [+ R(7)] | `FUN_1406cdc04` | lu |
| **index de contrôle = joueur** | **R(5)** | `FUN_1406d0388` | lu |
| bloc a présent | R(1) | | lu |
| +0x00 / +0x01 | R(1) [+ R(2)] | `FUN_1406cd860` | lu |
| couple analogique | R(6) R(6) | `FUN_1406d6ef4` (chemin quantifié) | lu |
| 3e champ | R(1) [+ **R(5)**] | `FUN_1406d84b4`, `[RSP+0x20]=5` @0x1422f6bdb | **refusé** |
| +0x10 | R(1) [+ **R(6)**] | `FUN_1406d84b4`, `[RSP+0x20]=6` @0x142265fcc | **refusé** |
| +0x14 drapeaux | R(1) [+ **R(5)**] | @0x1406cdb43 → 0x142265fe3 → **0x1406cd991 : le bloc d'action suit** | **refusé** |
| **bloc d'action** | voir ci-dessous | `FUN_1406d025c` | lu, deux sous-lecteurs faux |
| bloc b (0xbc) | R(1) [+ `FUN_141fdae44`] | | refusé (non porté ici non plus) |

Bloc d'action `FUN_1406d025c` (ordre des lectures relu) :
`g R(1)` ; si g : `R(1)` → **m0 R(3)** [(main 0, entrée 0) (0,1) (0,2), MSB d'abord] + **m2 R(3)** ;
`R(1)` → **m4 R(2)** + **m5 R(2)** ; `c R(1)` → [R(1) R(1), `FUN_1431a0bbc` R(1)+R(8), `FUN_1431a0abc`
R(1)+R(10), **`FUN_1431a0cbc` : mode R(2) ; 1 → R(19) (`FUN_14076dc04`, R9D=0x13 @0x1431a0cdf) ; 2, 3 →
rien ; 0 → vecteur quantifié à table d'exécution (`FUN_14076e524`, index R(bitLen(0x400)))**] ;
`R(3)` (+6) ; si m0|m4 : `FUN_1406d00ec` (+7, arme de la main 0) ; si m2|m5 : idem (+8) ;
**`FUN_140c9e4d8` : g R(1) ; genre R(2) ; genre 1 → `FUN_1406d3140` CATÉGORIE 1 (sonde comprise,
@0x140c9e9c7) + R(1)[R(6)] ; genre 2 → catégorie 2 (@0x1423e5a69)** ; R(1) ; [R(4) R(4) R(1)] ;
`FUN_140c9e738`.

La production (`consume1406d025c`, aussi appelée par `i19 unit-actor-control`) lit la catégorie 0 dans
les deux genres et modélise `FUN_1431a0cbc` comme `R(1)[R(1)[R(1)]]` : faux (§9).

## 3. Theater : d'où viennent l'animation et le son (question 2, côté lecteur) [D, Ghidra]

| étape | fonction | effet |
|---|---|---|
| réception | `FUN_14076b838` (vtable de la vue C) | code 0xd → copie dans la case du joueur `vue+0x25f0+i*0x68`, masque `+0x2554` |
| commande du joueur | `FUN_1406b09c0` → `FUN_14076b058` → `FUN_1407ff8cc` | bloc de 0x68 → commande (0xc0 o) |
| gâchettes | `FUN_1406dc5b8` | m0 bit 0 → drapeau 31, bits 1\|2 → 32 ; m4 → 33 / 34 ; m2 / m5 → 37..40 |
| unité | `FUN_14071c31c` → `FUN_1406db688` | `unité+0x308` = drapeaux de la commande |
| arme | `FUN_1406dba04` → `FUN_140613f98` | bits 29..34 (main 0) → `arme+0x2fa` bits 0..5 |
| gâchette | `FUN_1407fb92c` | entrée 0 « tenue » si bit 0 OU **bit 2** (l'état prédit) ; entrée 1/2 : bits 1 / 3 |
| barillet type 3 | `FUN_1407fa928` | `arme+0x2fa` bit `DAT_143d10650[b]` = 4 / 5 → `FUN_14060c5c0` (tir forcé) |
| tir | `FUN_1407fa928` → `FUN_1408a397c` → `FUN_14202f3a0` | cadence du barillet (tag), effets, son, projectiles (sans émission : pas l'autorité) |

L'écrivain (`FUN_14076a2f4` → `FUN_1406d175c`) prend l'arme de l'UNITÉ du joueur ; pour un pilote,
`FUN_1404998d0` rend l'arme du véhicule parent dont il tient les commandes (`parent+0x46c == unité`).

## 4. Mesure sur 81c02726 (question 3) [M]

Horloge : trame = (horodatage − 451 125 221 µs) / 100 ms. G MONEY = index 2 ; épisodes publiés
311-1088 (Ghost 769) et 2064-2869 (Ghost 771) ; frags au Ghost 395, 623, 650, 2159, 2228, 2402.

### 4.1 Oracle de cadrage (`TestP1S3VueCTirContinu`, 19 933 paquets delta)

| lecture de la vue C | lue jusqu'au terminateur | paquets CLOS (reste 0..7 bits nuls) |
|---|---:|---:|
| production (`consumeVueC`) | 13 226 | 3 902 |
| + entrée complète (3 branches) | 15 581 | 6 023 |
| + bloc d'action corrigé (cible, vecteur) | 15 791 | **6 865** (+2 963, 0 perdu) |

Vue B close sur 15 929 paquets, ouverte sur 3 951, liste non localisée 53. Arrêts restants : kinds 1/2
103, bloc de 0xbc 24, vecteur mode 0 1. Entrées : 43 612, dont index 0..7 validées à 99,7-100 % ;
index ≥ 8 : 146 entrées, 2 validées (bruit des paquets non clos).

### 4.2 Qui tire

| | entrées | validées | qui tirent | dans épisodes | hors |
|---|---:|---:|---:|---:|---:|
| index 2 (G MONEY) | 5 796 | 5 788 | **1 050 (1 050 validées)** | 994 | 56 |
| index 0,1,3,5,6,7 | 32 253 | 32 218 | 0 | — | — |
| index 4 | 5 417 | 5 408 | 1 (non validée) | | |

Valeurs : 1 050 × `m0=4 m2=0 m4=0 m5=0 r3=1` (gâchette principale, main 0) ; arme de la main 0 :
0 (990), 1 (56 = les 56 hors épisodes), −1 (4) ; cible typée genre 1 dans 847 entrées.

### 4.3 Gate (entrées qui tirent / entrées lues dans [f − 2 s, f])

| source | 395 | 623 | 650 | 2159 | 2228 | 2402 | témoin −60 s | hors épisodes |
|---|---|---|---|---|---|---|---|---|
| marche validée | **20/20** | 0/0 | 0/0 | **26/26** | 0/0 | **24/29** | 0 / 336 lues | 56 / 3 092 |
| marche + balayage (§5) | **65/65** | **47/47** | **38/38** | **62/62** | **46/46** | **27/60** | **0 / 430 lues** | 169 / 4 841 |

À 2402, les entrées 2393-2394 ne tirent pas, la rafale commence à 2395 (0,7 s avant le frag). Le témoin
+60 s tombe DANS un épisode pour 395, 2159, 2228 (58/58, 0/53, 92/92 : le pilote tire ou non selon
l'instant), hors épisode pour 623 et 650 (0/93, 0/81) et, pour 2402 (t 3002), dans la troisième
monture (47/57). Le gate de P1 (« absent à ±60 s ») supposait des épisodes plus courts que 60 s ; le
témoin qui vaut est −60 s et « hors épisodes ».

**Hors épisodes = une troisième monture non publiée [D]** : les 169 entrées sont dans deux rafales,
t 2984..3020 et 3104..3119 ; le document fait bouger le Ghost 771 à 2950-3094 (2 m/s et plus) alors
que sa monture publiée finit à 2869 et que la trace de G MONEY s'arrête à 2942 ; aucun autre joueur
n'a de gâchette continue.

### 4.4 Rafales (marche + balayage) et couverture

Épisode 1 : 362..463 (10,1 s) · 481..518 (3,7) · 551..658 (10,7) · 775..797 (2,2) · 820..830 (1,0) ·
932..1050 (11,8) · 1061..1072 (1,1). Épisode 2 : 2125..2163 · 2195..2235 · 2357..2379 · 2395..2411 ·
2461..2533 · 2562..2574 · 2622..2686 · 2772..2863. Chaque borne est encadrée par l'entrée lue sans tir la
plus proche (ex. 2395..2411 : sans tir à 2395 et à 2411 → bornes au tick près). Les 6 frags tombent
chacun dans une rafale.

Couverture des trames (100 ms) des épisodes : marche validée **29 % / 55 %** ; marche + balayage
**83 % / 91 %**. Cadence des ENTRÉES : un tick (16-17 ms) ou deux (33 ms) — c'est la cadence de
réplication de la commande, pas celle des tirs.

## 5. Balayage par la fin (`TestP1S3VueCBalayage`) — ce qu'un repli rendrait

Règle écrite avant la mesure : départ candidat = une vue C qui va au terminateur, clôt (0..7 bits nuls),
porte ≥ 1 entrée, index strictement croissants < 8. Tous les suffixes d'une vraie vue C sont candidats ;
la règle retient le **candidat le plus à gauche dont la chaîne contient tous les autres**, et se tait
sinon. Étalonnage sur les 6 281 paquets à départ vrai connu : **2 224 justes, 3 faux (0,13 %), 4 054
muets**. Sur les 13 652 paquets sans départ vrai : 5 054 retenus, 30 216 entrées. Candidat unique seul :
inutilisable (0 unique sur 6 281).

## 6. Pourquoi la vue C avait été réfutée par P1

- « Fermé aux 23 tirs de véhicule à coup (8a485699) » : **attendu** [D] — le bloc ne porte que les
  gâchettes de type 1 (et barillets de type 3) dont le tir n'est PAS émis ; une arme à coup a son record 36.
- « Fermé à la touche continue de la Banshee (t 2422) » : une touche peut suivre le lâcher (vol du
  projectile) ; un seul échantillon, lu par une grammaire qui refusait 3 branches et lisait faux la cible
  et le vecteur. **À mesurer** avec cet instrument (§10, 8a485699, index 7, 2334-2449) — pas fait ici
  (consigne : un seul film).
- « Ouvert en rafales hors épisodes chez six joueurs (`r3 = 6`) » : compatible avec des armes continues
  à pied (Rayon de Sentinelle, pris par 0, 1, 2, 4 — sauts +48, +181, +183, +210) [H, à mesurer].

## 7. Ce que M4b doit lire et faire (impactPlan)

- **M4b.0 (préalable, mesure)** : rejouer `TestP1S3VueCTirContinu` sur 8a485699 (Banshee index 7,
  Ghost index 4, Chopper index 7, Wasp index 0 : LMG attendue en m0 entre les missiles `11725DC4`) et lire
  dans les tags (`barillet+0x70/+0x74/+0x78/+0x7c`, `rounds per second`) les armes de véhicule, pour
  fixer la table type de prédiction / cadence par arme.
- **Grammaire (production)** : porter l'entrée complète de la vue C (3 branches du §2) et corriger
  `consume1406d025c` (catégorie 1/2 de `FUN_140c9e990`, vecteur `FUN_1431a0cbc`) ; `grammar.Rev` monte
  (le même bloc est lu par `i19` : effet sur la clôture de la vue B à mesurer au gate 5.16).
- **Faits** : par joueur et par tick, les bits m0/m2/m4/m5 (+ arme de main, cible typée) →
  **intervalles de tir continu [début, fin]** au tick ; `SchemaDesFaits` monte.
- **Publication** : une rafale par intervalle, posée sur le véhicule que le joueur monte (siège /
  monture ; l'entrée nomme le JOUEUR, pas le véhicule), arme = celle de la gâchette (entrée 0 = principale) ;
  **tirs aux instants début + k / cadence** (cadence du tag, comme Theater — Ghost 7,5 /s) ; son en
  boucle de début à fin ; les touches `damage_aftermath` gardent leurs instants et leurs victimes.
  Contrôle : nombre de tirs ≈ saut du numéro de tir du record 36.
- **Trous** : paquet dont la vue C n'est pas atteinte = trou NOMMÉ et compté en couverture, jamais un
  « pas de tir » ; le balayage du §5 est le repli nommé candidat (0,13 % d'erreur étalonnée).
- **Gate G1 redéfini** : 81c02726 — une entrée qui tire dans les 2 s avant chacun des 6 frags (6/6
  mesuré), 0 au témoin −60 s, 0 hors montures (après correction de la 3e monture, découverte §9).

## 8. Questions pour l'utilisateur

1. Theater ne reçoit pas les instants des tirs continus : il simule la rafale à la cadence de l'arme
   pendant que la gâchette est tenue. Le rejeu fait-il pareil (bornes lues dans le film, cadence de
   l'arme) ?
2. Les trous de couverture (vue B non close) : repli par le balayage étalonné (0,13 % d'erreur, compté),
   ou silence jusqu'à ce que la vue B close mieux ?

## 9. Découvertes hors périmètre (non traitées)

1. `consumeEntreeControle` refuse trois branches résolues (§2) ; `consume1406d025c` lit faux la cible
   typée et le vecteur — même bloc dans `i19 unit-actor-control` (vue B).
2. 81c02726 : 3e monture de G MONEY sur le Ghost 771 absente du document (≈ 2950-3119) ; trace de
   G MONEY arrêtée à 2942.
3. La vue C n'est atteinte que sur 34 % des paquets de 81c02726 (6 865 / 19 933) : ~8 900 paquets
   « lisent » une vue C vide qui ne clôt pas = fin de vue B fausse.
4. La cible typée du bloc (genre 1, catégorie 1) nommerait l'objet visé pendant la rafale [H].

## 10. Instruments et commandes

`p1s3_vuec_tir_continu_research_test.go` (lecture, oracle), `p1s3_vuec_publication_research_test.go`
(publication), `p1s3_vuec_balayage_research_test.go` (balayage étalonné). Durée : ~1 s et ~4 s.

    MOUV511_FILM=<depot>/data/cache/film_chunks/81c02726 \
    MOUV511_BORNES=<depot>/data/titles/halo_infinite/reference/map_quant_bounds.json \
    MOUV511_CARTE=isolation S3_ORIGINE_US=451125221 S3_INDEX=2 S3_JOUEURS=8 \
    S3_EPISODES=311-1088,2064-2869 S3_FRAGS=395,623,650,2159,2228,2402 \
      go test -tags=research -count=1 -v -timeout 60m -run '^TestP1S3VueC(TirContinu|Balayage)$' \
      ./internal/games/halo_infinite/film/internal/grammar/

    # 8a485699 (non joué ici) : MOUV511_CARTE="launch site" S3_ORIGINE_US=2331882869 S3_INDEX=7
    # S3_EPISODES=2334-2449,3099-3205 S3_FRAGS=... (Banshee et Chopper de KyleT1848)
