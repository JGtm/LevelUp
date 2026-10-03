# NOTE 5.23 — LA TABLE ANTICIPEE DES ARCHETYPES

Lot 5.23, branche `feat/decfilm-73`, base `f0bd32b0a` (lots 5.1 a 5.21, grammar-2026-09-22.10,
schema 67). Sur la mesure du lot 5.20.2 : **74,7 % des slots rejetes sont declares, avec leur
archetype, par l image-cle du chunk SUIVANT**.

Ce lot ne cherche pas l ecrivain de la naissance — sept lots l ont instruit (5.15 a 5.21) et la
question est close. Il construit un REPLI : une table `(slot, tete) -> archetype` batie sur les
images-cles de TOUT le film, consultee AU POINT DE REJET, qui lie une entite nee en milieu de
chunk sur la foi d une image-cle ULTERIEURE. **Le record de naissance reste non lu.** C est dit,
c est date, c est compte.

---

## 1. LA CLE, ET ELLE EST CELLE QUE LE JEU COMPARE

Une seule lecture de Ghidra dans tout le lot, et c est celle-la : `FUN_1406caad8`, la porte que
tout corps de delta franchit.

```
uVar21 = param_2 & 0x3fffffff                            ; le SLOT — 30 bits bas de l eid
si param_2 == 0xffffffff                     -> return 3 ; sentinelle
lVar19 = *(longlong *)(param_1 + 0x20)                   ; base de la table, pas 200
si (fin - base) / 200 <= uVar21              -> return 3 ; slot hors cardinal
si *(uint *)(uVar21 * 200 + lVar19) != param_2 -> return 3   ; <- LA CLE
puVar18 = uVar21 * 200 + lVar19 ; puVar18[1]             ; l ARCHETYPE, en +0x04
```

La table est INDEXEE par le slot et son entree porte l eid **ENTIER** : les deux bits de tete
comptent, et un delta dont la tete ne vaut pas celle de l entree ne rend AUCUN bit de corps.
**La cle est donc le mot de 32 bits lui-meme**, `(slot, tete)`.

**ET LES DEUX BITS DE TETE D UNE IMAGE-CLE SONT CEUX QU UN DELTA DOIT PRESENTER**, parce que
c est la MEME table : l entree de 200 octets que cette porte teste est celle que `FUN_142e2bfd0`
remplit depuis le payload d image-cle (`e[0x00] = R(32)` l eid, `e[0x04] = R(32)` l archetype —
lot 5.20.1, pas de 0xC8 = 200 confirme par `FUN_142e2bb9c`).

Le brief demandait de le DIRE si le tag de 2 bits d un en-tete de delta et les bits 30-31 d un
eid d image-cle n etaient pas le meme champ. **Au sens de la comparaison, ils le sont** — le jeu
les confronte mot a mot. Ce qu ils SIGNIFIENT reste ce que le lot 5.13.1 a etabli (le rang de la
vue chez `FUN_142f2e174`, la generation du datum chez `FUN_1408f1730`), et les deux films temoins
ne les departagent pas : une seule valeur, `1`, du cote des images-cles. La cle, elle, n est pas
ambigue, et ce lot cle dessus.

---

## 2. LA TABLE

`keyframe_anticipe.go` (207 lignes, couche `grammar`). Une passe sur les images-cles de TOUS les
chunks, par la lecture que le monde emprunte deja (`WalkKeyframeWorld` rend `Slot`, `TI`, `Gen` —
le mot de 32 bits decompose) ; aucune seconde lecture d image-cle n est ecrite.

Chaque cle porte la suite DATEE de ses declarations. `ArchetypeApres(id, chunk)` rend la PREMIERE
declaration STRICTEMENT POSTERIEURE au chunk du rejet : **anticiper, c est lire l avenir d un
slot, jamais son passe.** La passe entiere coute **1,8 s** sur `bfecd02b`, sans aucun decodage de
trame.

### Ce que la table pese (`TestTable523`, `bfecd02b`, carte `snowbound`)

| | valeur |
|---|---:|
| declarations d image-cle versees | **12 688** |
| cles `(slot, tete)` distinctes | **1 015** |
| cles portees par PLUS D UN archetype | **0** |
| tetes rencontrees cote image-cle | **`1` seule**, 12 688 fois |

**ZERO CONFLIT** : aucun slot n est reutilise sous la meme tete sur ce film. La datation par
chunk n arbitre donc rien ici — et elle reste, parce qu elle est la garde qui empechera un slot
recycle de rendre l archetype de son occupant PRECEDENT le jour ou un film en portera un.

### Ce que la table couvre

| ou le rejet trouve-t-il son archetype ? | rejets | part |
|---|---:|---:|
| **declare par une image-cle POSTERIEURE** | **17 432** | **74,7 %** |
| declare seulement par un chunk anterieur ou courant | 0 | 0,0 % |
| **aucune image-cle du film, jamais** | **5 893** | **25,3 %** |

Le chiffre reproduit celui du 5.20.2 au rejet pres. Declarant a **+1 chunk : 17 430** ; a +8 : 1 ;
a +13 : 1. Par archetype anticipe : `ti=35` **16 932** (les reapparitions de bipedes), `ti=42`
346, `ti=41` 85, `ti=40` 53, `ti=10` 9, `ti=37` 7.

### Et la tete discrimine

Les en-tetes rejetes portent la tete `1` 22 687 fois, mais aussi `0` (98), `2` (392) et `3`
(148) : **638 en-tetes presentent une tete qu AUCUNE image-cle du film n emploie**, donc un eid
que le jeu ne pourrait pas apparier. Une cle reduite au seul slot ne resoudrait que **19 rejets
de plus** et lierait ces 638-la. Le prix de la cle juste est de 19 liaisons ; ce qu elle ecarte
est 638 lectures prises a une position fausse.

---

## 3. LE REPLI, ET CE QU IL COUTE

`rejetDeVue` (`frame_infer.go`) recoit l eid COMPLET — la cle du jeu porte les deux bits de tete
— et consulte la table AVANT de compter un rejet hors datum. `World.LierParAnticipation`
(`world.go`) pose alors la liaison de la table de datums (`BindDatum` : `Soft`, `GenAny`, vue
INCONNUE, sans position), la compte par archetype (`Observation.LiaisonsParAnticipation`) et
journalise le premier usage du film. Sans table installee, pas un bit ne change.

### Le gate (`TestGate516`, carte `snowbound`, A/B `MOUV523_ANTICIPE=0`)

| mesure | `dad793c7` avant | apres | `bfecd02b` avant | apres |
|---|---:|---:|---:|---:|
| paquets a reste NUL | 5 354 / 5 365 | **5 355** | 2 884 / 30 387 | **3 919** |
| reste hors bourrage | 9 | 8 | 27 471 | 26 418 |
| debordements | 2 | 2 | 32 | **50** |
| records rendus | 5 641 | 5 649 | 176 786 | **240 488** |
| `ti=35` (desynchronises) | 75 (0) | 75 (0) | 129 572 (4) | **164 232 (4)** |
| fantomes | 1 | 1 | 31 | **49** |
| rejets hors datum | 2 | **1** | 23 769 | **16 129** |
| liaisons par anticipation | 0 | 8 | 0 | **254** |

Le film de calibration gagne un paquet et ne perd rien. Sur le film dense, **254 liaisons evitent
7 640 rejets** et rendent 63 702 records de plus : `ti=42` x3,59, `ti=40` (vehicule) x3,02,
`ti=10` x2,97, `ti=41` x2,43, `ti=32` x2,40, `ti=37` (equipement) x2,39, `ti=35` x1,27 — et
`ti=4`, le seul temoin propre du film (1,1 % de faute au 5.19.1), ne bouge pas.

### Les deux compteurs de faute qui montent, et leur cause

Debordements 32 -> 50, fantomes 31 -> 49. Le balayage par archetype (`MOUV523_TI`, un archetype
anticipe a la fois) l attribue : `ti=35` seul rend 48 et 47 ; `ti=42` seul rend **22 et 21**,
c est-a-dire DIX debordements de MOINS qu avant. Aucun autre archetype ne les deplace.

La raison est celle que le 5.16.2 avait deja ecrite : les slots rejetes se concentrent dans la
bande de bipedes 521-601 (27 slots, 72 % du volume — 5.19.2), que toutes les images-cles
ulterieures declarent. Un en-tete pris a une position FAUSSE y tombe donc facilement, et la ou il
s arretait il lit desormais un corps de bipede qui deborde. **Aucun paquet ne passe de FERME a
fautif** : les 18 quittent « reste hors bourrage » (27 471 -> 26 418, soit -1 053) pour
« debordement », et 1 035 le quittent pour « ferme ». L oracle de CONTENU ne bouge pas :
`ti=35` desynchronises 4 avant, 4 apres.

### Ce qui est nouvellement lu

**207 -> 222 etiquettes de composant** ; `i21 unit-desired-aiming-vector` 84 827 -> 106 108
(64,6 % des records `ti=35`, contre 65,5 %). Vingt-neuf etiquettes apparaissent — dont
`equipment-deployed-component` et `equipment-has-infinite-uses-component` (ti=37),
`crew-order-component`, `game-engine-current-state-component`, huit `tacmap-*`, deux
`statborg-*`, trois `forge-engine-*` — et quatorze disparaissent, treize `managed-navpoint-*`
et `projectile-deceleration-disabled-state` : les records `ti=12` passent de 33 a 37, mais leur
masque n est plus le meme et `managed-navpoint-visual-state-groups-component-0` cede la place
a `-3`.

Aucune de ces etiquettes n est un canal PUBLIE : `replay.SchemaVersion` reste **67**. La
couverture gagne un compteur, `Observation.LiaisonsParAnticipation`, qui n est PAS un champ du
contrat — `Observation` n est jamais publie, ni dans le document ni dans les faits persistes.

---

## 4. LA PRODUCTION — UN SEUL INSTALLATEUR, ET LA MESURE DIT POURQUOI

`ScanMovementStates` construit la table (`ConstruireTableAnticipee(fc)`, une passe, 1,8 s sur
29 chunks), la pose sur son monde et annonce le chunk courant. Aucun autre fichier de decodage
ne bouge — et c est la decouverte du point.

| marche de production | ce qu elle lie AVANT de marcher | anticipe ? |
|---|---|---|
| `killsource/world.go` `preload()` | la PREMIERE declaration de chaque slot de TOUTES les images-cles du film | **oui** |
| `object_deaths_march.go` `newMarchTimeline()` | le meme geste, mot pour mot | **oui** |
| `weapon_hits.go` | les images-cles DE SON CHUNK seulement | non, mais elle marche par `DecodeFrameRecords`, sans point de rejet |
| `ScanMovementStates` | les images-cles DEJA VUES, chunk par chunk | **non — c est elle qui gagne** |

Deux marches de production ANTICIPAIENT DEJA, plus largement que cette table (sans datation,
sans cle de tete), et depuis des lots anterieurs. C est la raison mesuree pour laquelle
**`facts.Rev` ne monte pas** — le monde de `killsource` connait deja ces slots, et
`LierParAnticipation` y rendrait `false` sur chacun — et pour laquelle les calques `vehicles`,
`rides` et `equipmentEpisodes` ne bougent pas d une unite. Aucun backlog killsource.

### Le rendu (`replay-build`, `bfecd02b`, carte snowbound, faits de film purges)

| calque | avant | apres |
|---|---:|---:|
| **`stances`** | **616** | **841** |
| dont sprint | 355 | **501** |
| dont saut derive | 252 | **327** |
| dont mobilite | 9 | **12** |
| dont accroupi | 0 | **1** |
| pistes · points | 90 · 27 703 | 90 · 27 703 |
| vehicules · embarquements · fins | 11 · 3 · 11 | 11 · 3 · 11 |
| `equipmentEpisodes` · `equipmentChanges` | 10 · 23 | 10 · 23 |
| `shots` · `pickups` · `padPickups` | 2 568 · 142 · 65 | 2 568 · 142 · 65 |
| artefact (octets) | 2 238 332 | 2 249 698 |

`replay-equiv -films bcb6d393` sans `-update` : **les SIX memes ecarts que les lots 5.14 a 5.21
et aucun autre**. Deux portent la mesure de ce lot — `movementStates` 1 737 -> **2 450**,
`artifact` 1 929 397 -> **1 938 579**. La reference est perimee depuis la fusion 5.10 ; le
re-figeage est un geste du pilote.

`replay.SchemaVersion` reste **67**.

---

## 5. LE RESTE, CHIFFRE

Sur les 23 325 rejets mesures avant le repli, la table en resout 17 432 (74,7 %). Les
**5 893 (25,3 %)** restants ne sont declares par aucune image-cle du film, et la CLE les coupe
en deux populations qui ne sont pas de meme nature :

| ce que le reste porte | rejets | part | ce que c est |
|---|---:|---:|---|
| tete `1`, slot jamais declare | **5 255** | 89,2 % | l entite est **nee ET morte entre deux images-cles** — la queue de cascade du 5.20.2 |
| tete `0` (98), `2` (392), `3` (148) | **638** | 10,8 % | un eid que **le jeu lui-meme ne pourrait pas apparier** : ce ne sont pas des naissances, ce sont des lectures prises a une position FAUSSE |

Dix-neuf des 638 seraient resolus par une cle qui ignorerait la tete ; on ne l ignore pas.

Apres le repli, le compte change de POPULATION et non de nature : 16 129 rejets hors datum au
lieu de 23 769, parce que 1 035 paquets de plus vont jusqu a leur bourrage et que la marche,
allant plus loin, rencontre des rejets qu elle n atteignait pas.

**L adresse de ce qui fermerait ce reste est connue, et ce lot ne l instruit pas** : le record de
NAISSANCE du flux de trame — D1 du 5.19, « le chunk 2 lit ZERO record `NEW` sur 1 196 paquets
delta alors qu au moins dix entites y naissent ». Les deux seules sources de la table de datums
sont lues et portees (5.20.3 (c)) ; ce qui manque est de voir le `NEW` la ou il est ecrit.

---

## 6. CE QUE LE LOT LAISSE OUVERT

Cinq decouvertes, toutes au §4 du plan et aucune traitee : trois anticipations du meme fait
coexistent desormais dans trois marches (D1) ; l anticipation du bipede est la moitie du gain et
la totalite du cout en debordements (D2) ; le film livre enfin `equipment-deployed-component`,
le signal « deploye / lache » que rien n exploite (D3) ; le masque des navpoints change (D4) ;
la table coute 1,8 s par balayage et n est pas partagee, faute de place dans `film_context.go`
(D5).
