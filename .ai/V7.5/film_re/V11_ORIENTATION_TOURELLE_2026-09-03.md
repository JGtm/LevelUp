# RAPPORT — lot V11 : ORIENTATION DE LA TOURELLE, VISEE DE L'ARTILLEUR ET DU PASSAGER

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`. Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v11`), films du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks`, LECTURE SEULE).
> `apps/web/`, `replay/vehicle_rides*.go`, `filmdec/event_list.go` : **NON TOUCHES**.
>
> **CONCURRENCE.** D'autres sessions travaillent dans le meme worktree (lots V8 / V9 / V10). Ce
> lot n'a modifie qu'UN fichier de production preexistant (`filmdec/offline_aim.go`, extraction
> de deux formules d'angle, zero bit change) et n'en a cree qu'un (`filmdec/offline_aim_only.go`).

---

## 0. LE RESULTAT EN CINQ LIGNES

1. **PISTE A — `i2` REJOUE SOUS LA GRAMMAIRE CORRIGEE : TOUJOURS REFUTE.** Avec les variantes
   `-dynamic-precision-` du lot V9, l'ecart median de `i2` au cap de deplacement passe de
   91,0 / 40,3 / 137,3 / 71,7 deg a **47,9 / 122,7 / 99,9 / 75,7 deg** (4 films) — au-dessus du
   seuil de 30 deg partout. Le controle `i1` reste a 1,7-2,9 deg dans les deux regimes : le
   curseur est bon, la faute est bien dans `i2`. Un signal apparait sur le plus gros film
   (R 0,376 contre 0,032 au temoin), pas sur les trois autres. **Rien publie** (§ 2).
2. **PISTE B — LA TOURELLE NE REPLIQUE RIEN : REFUTEE, ET AVEC UN TEMOIN.** Un balayage
   d'en-tetes SANS exigence d'`i0` rend, par slot, **139,6 touches** sur les slots tourelle
   contre **86,3** sur une bande FANTOME (`0d76e8f1`) et **85,5 contre 194,2** (`fccc61cd`) —
   soit sous le bruit sur un film et 1,6x dessus sur l'autre, avec un histogramme de formes de
   masque PLAT dans les deux cas. Les slots MUETS non-tourelle rendent le meme chiffre
   (152,9 / 108,0) : rien n'est propre a la tourelle (§ 3).
3. **PISTE C — `i21` EST ABSENT DE `ti=40`, MAIS IL EST PARTOUT SUR LE BIPEDE, DANS DES RECORDS
   QUE LE DEPOT NE SAVAIT PAS LIRE.** C'est le resultat du lot. `ScanBipedRecords` exige un `i0`
   ABSOLU et un masque commencant par 0 ; or la forme de masque la plus frequente de la bande
   bipede APRES ce filtre est `i21,i25` — **un record de visee SANS position**. 22 963 lectures
   sur `0d76e8f1` (222,9 par slot) contre **0,9 par slot** sur la bande FANTOME (§ 4).
4. **CES LECTURES SONT JUSTES ET ELLES SURVIVENT AU TROU.** Appariees a moins de 200 ms a la
   lecture `i21` AVEC position du meme slot : ecart median **0,2 a 0,5 deg**, R 0,979 a 0,989 ;
   temoin par melange **75,7 a 93,7 deg**, R 0,011 a 0,134. Et sur les **35 episodes
   d'occupation attestes** de 5 films, **35 / 35 (100 %)** portent au moins une visee A BORD, a
   5 a 46 lectures par seconde, quand le meme episode porte **0 ou 1** lecture `i21` avec
   position (§ 5).
5. **REPONSE PAR ROLE, ET ELLE EST POSITIVE POUR LES TROIS.** CONDUCTEUR, ARTILLEUR et PASSAGER
   ont chacun leur propre slot bipede, donc chacun sa visee lisible pendant qu'il est a bord.
   Ce n'est PAS l'orientation de la tourelle en tant qu'objet — c'est la visee de l'homme qui la
   tient, ce qui est la meme chose pour un cone de visee. **Porte dans `filmdec`
   (`ScanFilmBipedAimOnly`) ; NON porte dans le document** : son seul emplacement correct est
   l'episode d'occupation (`replay/vehicle_rides*.go`), fichier explicitement hors perimetre
   (§ 7).

---

## 1. CE QUI EST MESURE, ET PAR QUOI

| instrument | fichier | garde | ce qu'il mesure |
|---|---|---|---|
| oracle du cap `i2`, deux grammaires | `replay/vehicules_v11_orientation_test.go` (NEUF, 114 L) | `ATT_FILM` + `V0_FILMS` | piste A |
| marche sequentielle des records `ti=40` | `filmdec/vehicules_v11_tourelle_test.go` (NEUF, 350 L) | `V11_ROOT` + `V11_FILMS` | piste B, voie 1 |
| balayage d'en-tetes SANS `i0`, 5 classes de slots | `filmdec/vehicules_v11_scan_test.go` (NEUF, 445 L) | idem | pistes B et C, + gate (c) |
| visee sans position : presence, justesse, trous, occupation | `filmdec/vehicules_v11_visee_test.go` (NEUF, 406 L) | idem | piste C |
| garde-rail de grammaire SANS environnement | `filmdec/offline_aim_only_test.go` (NEUF, 123 L) | aucune | le decodeur livre |

---

## 2. PISTE A — `i2` DU CHASSIS, RELU AVEC LA GRAMMAIRE ETABLIE PAR V9

Oracle INCHANGE depuis V1a (c'est la condition de comparabilite) : l'echantillon porte `i2` et
`i1`, la norme de sa velocite depasse 5 m/s, un echantillon suivant du meme slot existe a moins
de 2 s. Seuils ecrits avant mesure : moyenne circulaire < 15 deg, mediane des ecarts absolus
< 30 deg. Flux BRUT (aucun post-filtre). Temoin par MELANGE deterministe.

### 2.1 `i2` contre le cap de DEPLACEMENT

| film | regime | paires | moyenne circ. | mediane abs. | R | temoin (mediane / R) | verdict |
|---|---|---|---|---|---|---|---|
| `0d76e8f1` | AVANT (bipede) | 18 440 | +155,0 | **91,0** | 0,029 | 94,2 / 0,057 | ECHOUE |
| `0d76e8f1` | **APRES (dyn.-prec.)** | 5 129 | -32,8 | **47,9** | **0,376** | 89,5 / 0,032 | **ECHOUE** |
| `fccc61cd` | AVANT | 1 061 | +48,0 | **40,3** | 0,555 | 46,5 / 0,559 | ECHOUE |
| `fccc61cd` | **APRES** | 76 | +127,1 | **122,7** | 0,532 | 124,7 / 0,424 | **ECHOUE** |
| `8a049c50` | AVANT | 2 559 | -162,1 | **137,3** | 0,409 | 124,2 / 0,462 | ECHOUE |
| `8a049c50` | **APRES** | 996 | +167,2 | **99,9** | 0,198 | 109,2 / 0,204 | **ECHOUE** |
| `51d3ab9f` | AVANT | 3 695 | +63,9 | **71,7** | 0,273 | 70,8 / 0,155 | ECHOUE |
| `51d3ab9f` | **APRES** | 1 078 | +18,9 | **75,7** | 0,207 | 93,7 / 0,097 | **ECHOUE** |

### 2.2 Les deux controles, et ce qu'ils disent

**CONTROLE DE CURSEUR — `i1` contre le DEPLACEMENT, sous les deux regimes :**

| film | AVANT (mediane / R) | APRES (mediane / R) |
|---|---|---|
| `0d76e8f1` | 1,7 / 0,997 | **2,3 / 0,994** |
| `fccc61cd` | 1,9 / 0,996 | **7,0 / 0,979** |
| `8a049c50` | 1,7 / 0,992 | **2,0 / 0,980** |
| `51d3ab9f` | 2,1 / 0,995 | **2,9 / 0,988** |

`i1` est lu AVANT `i2` : il PASSE dans les deux regimes, donc le curseur arrive bon jusqu'a `i2`
et l'echec est bien dans `i2` — la conclusion de V1a § 3.3 tient sous la grammaire corrigee.

**COUVERTURE — la grammaire corrigee DIVISE par deux a trois le nombre d'`i2` porteurs d'une
direction**, et c'est coherent avec le desassemblage : les chemins `mode 2` (192 bits bruts) et
`delta` a quartets n'ecrivent AUCUNE direction absolue.

| film | i2 direction presente, AVANT | APRES |
|---|---|---|
| `0d76e8f1` | 30 784 / 32 246 (95,5 %) | **12 257 (38,0 %)** |
| `fccc61cd` | 5 011 / 5 431 (92,3 %) | **2 415 (44,5 %)** |
| `8a049c50` | 7 248 / 7 737 (93,7 %) | **4 675 (60,4 %)** |
| `51d3ab9f` | 14 238 / 15 351 (92,7 %) | **6 100 (39,7 %)** |

**CE QUE JE NE MAQUILLE PAS.** Sur `0d76e8f1` — le plus gros corpus, 5 129 paires — le regime
corrige rend R = **0,376 contre 0,032 au temoin** et une moyenne circulaire de -32,8 deg : il y
a une ASSOCIATION reelle entre `i2` et le cap de deplacement, la ou la grammaire du bipede n'en
rendait aucune (0,029 contre 0,057). Elle ne suffit pas : la mediane reste a 47,9 deg, et les
trois autres films ne la reproduisent pas (0,198 / 0,532 / 0,207 contre des temoins de meme
ordre). Deux lectures possibles, non departagees ici : soit la direction 19 bits des deux
chemins porteurs (`FUN_140c5fa84` et la branche `g1=0,g2=0` de `FUN_14076e744`) n'a pas le meme
encodage, soit le vecteur lu est le HAUT et non l'AVANT. **Non instruit, non bricole, rien
publie.**

---

## 3. PISTE B — L'ENTITE TOURELLE

### 3.1 Voie 1 : la marche sequentielle des records — INEXPLOITABLE, et il faut le dire

`DecodeFrameViews` sur tout `0d76e8f1` : 40 189 paquets, 148 146 records, 35 549 desync,
**323 records `ti=40`** — quand `ScanBipedRecords` en accepte **32 246** sur le meme film. La
marche en recupere donc **1,0 %**, et ce qu'elle recupere est majoritairement post-desync : les
masques imprimes portent des index **i48 a i63**, impossibles pour un archetype de 48
composants. Deux formes seulement sont credibles (`i0,i1,i2,i3,i25` : 17 records ; masque vide :
14). **Cette voie ne repond pas a la question**, et c'est un fait mesure, pas une opinion.

### 3.2 Voie 2 : le balayage d'en-tetes SANS exigence d'`i0` — LA REPONSE

Grammaire d'en-tete inchangee (prefixe, slot 13 bits, tag, couple de zeros, compteur, index
strictement croissants et < 48), **moins la contrainte « premier index = 0 »**. Quatre classes de
slots, dont un PLANCHER DE BRUIT mesure (bande FANTOME : slots jamais vus porter le moindre
archetype, meme cardinalite que la classe TOURELLE).

`0d76e8f1` :

| classe | slots | touches / slot | dont SANS i0 / slot | formes sans i0 (top) |
|---|---|---|---|---|
| CHASSIS | 27 | **1 555,6** | 400,8 | `i1,i3,i25` (3011) · `i1,i2,i3,i25` (1267) · `i4` (761) · `i37` (641) |
| **TOURELLE** | 10 | **139,6** | 131,3 | `i12` (75) · `i8` (70) · `i4` (67) · `i40` (54) — **PLAT** |
| MUET (autres) | 8 | 152,9 | 140,8 | `i13,i33,i44` (103) · `i12` (83) — PLAT |
| FANTOME | 10 | **86,3** | 82,5 | `i25` (46) · `i4` (40) · `i26` (39) — PLAT |

`fccc61cd` :

| classe | slots | touches / slot | dont SANS i0 / slot |
|---|---|---|---|
| CHASSIS | 13 | **872,2** | 439,1 |
| **TOURELLE** | 4 | **85,5** | 82,2 |
| MUET | 4 | 108,0 | 94,5 |
| FANTOME | 4 | **194,2** | 167,2 |

**VERDICT : REFUTE.** Sur `fccc61cd` la classe TOURELLE est **SOUS** le plancher de bruit
(85,5 contre 194,2). Sur `0d76e8f1` elle est 1,6x dessus — mais la classe MUET non-tourelle
rend 152,9, donc l'excedent n'est pas propre a la tourelle ; il s'explique par la proximite de
bande (un slot voisin d'un chassis qui emet 1 555 en-tetes par slot capte des appariements
decales). Surtout, **l'histogramme des formes de masque est PLAT** dans les trois classes non
chassis, alors qu'il est massivement concentre pour le chassis. Une entite qui replique quelque
chose produit des formes recurrentes ; la tourelle n'en produit aucune.

Le detail par slot le confirme (`0d76e8f1`, classe TOURELLE) : `770:136 774:105 785:251 790:89
797:128 799:118 801:173 804:174 806:142 813:80` — a comparer au FANTOME `2821:65 2822:75
2823:108 2824:215 2825:75 2826:92 2827:68 2828:59 2829:50 2830:56`. Meme dispersion, meme ordre
de grandeur.

**CONCLUSION.** La tourelle d'artilleur n'est pas seulement muette en POSITION (acquis V8) :
elle est muette **tout court** dans le flux delta. Aucun cone d'artilleur ne peut venir d'elle.
Corollaire : les composants `i31 vehicle-auto-turret-aiming-vector`, `i41 seats-override-pitch`
et `i42 seats-override-yaw` de `ti=40` sont bien NON PORTES, mais surtout ils ne sont **jamais
emis** — leur taux dans les masques du chassis (5,9 / 5,7 / 2,7 par slot sans i0) est du meme
ordre que le bruit fantome (1,1 / 1,1 / 2,2). Les porter ne servirait a rien.

---

## 4. PISTE C — `i21`, ET LE POINT AVEUGLE DU DECODEUR

### 4.1 Le fait qui retourne la question

`i21` reste **absent du flux `ti=40`** : le lot V1 l'avait mesure a 0,00 % du masque sur 81 540
records, et le balayage sans `i0` de ce lot n'y change rien (2,8 par slot sur le chassis, contre
2,1 sur la bande fantome — bruit). **Le vehicule ne vise pas.**

Mais la meme mesure, appliquee a la bande BIPEDE, rend ceci (`0d76e8f1`, records SANS `i0`) :

```
formes : i21,i25 (17411) · i5,i21,i25 (4887) · i5 (3875) · i25 (1178) · i1 (1133) · i33 (911)
index  : i21 = 228,8 par slot   contre FANTOME 2,1 par slot
```

**Le bipede replique sa VISEE dans des records qui ne portent AUCUNE position**, et
`ScanBipedRecords` ne peut pas les voir : il exige un `i0` absolu ET `idx[0] == 0`
(`ascendingFromZero`). C'est une exigence du DETECTEUR, pas du format.

### 4.2 Le decodeur livre, et son plancher de bruit

`ScanFilmBipedAimOnly` / `ScanBipedAimRecords` (`filmdec/offline_aim_only.go`, NEUF, 219 L) :
meme en-tete que `matchBipedHeaderRaw` **tag == 1 compris**, moins la contrainte `idx[0] == 0` ;
les composants qui precedent `i21` sont consommes par leurs detenteurs existants ; `i21` par
`readAimingVectorComponent`. Aucune grammaire reecrite.

| film | lectures | par slot | slots | TEMOIN FANTOME (par slot) | rapport |
|---|---|---|---|---|---|
| `0d76e8f1` | 22 963 | **222,9** | 103 | **0,9** | **x261** |
| `fccc61cd` | 4 832 | **46,5** | 104 | **0,3** | **x155** |
| `4898d586` | 24 050 | **231,2** | 104 | **0,2** | **x925** |
| `51d3ab9f` | 19 460 | **196,6** | 99 | **0,2** | **x811** |
| `8a049c50` | 8 148 | **86,7** | 94 | **0,6** | **x151** |

---

## 5. LES GATES DE LA VISEE SANS POSITION

### 5.1 Gate (a) — ORACLE : accord avec une direction CONNUE

Chaque lecture SANS position est appariee a la lecture `i21` AVEC position du MEME slot la plus
proche dans le temps, a moins de 200 ms. La reference est le champ deja publie (`Point.H`),
valide en production par l'oracle du kill.

| film | paires | mediane \|ecart de cap\| | R | TEMOIN par melange (mediane / R) |
|---|---|---|---|---|
| `0d76e8f1` | 2 607 | **0,4 deg** | 0,979 | 86,1 / 0,058 |
| `fccc61cd` | 1 948 | **0,4 deg** | 0,985 | 83,8 / 0,063 |
| `4898d586` | 2 535 | **0,5 deg** | 0,989 | 93,7 / 0,011 |
| `51d3ab9f` | 4 557 | **0,2 deg** | 0,989 | 86,4 / 0,031 |
| `8a049c50` | 2 917 | **0,4 deg** | 0,980 | 75,7 / 0,134 |

**PASSE.** 0,2 a 0,5 deg contre 75,7 a 93,7 deg au temoin.

### 5.2 Gate (b) — LA VISEE SURVIT-ELLE AU TROU DE POSITION ?

| film | trous >= 3 s | lectures DANS un trou | part |
|---|---|---|---|
| `0d76e8f1` | 33 | 11 295 | **49,2 %** |
| `fccc61cd` | 8 | 679 | 14,1 % |
| `4898d586` | 35 | 8 544 | 35,5 % |
| `51d3ab9f` | 30 | 7 755 | 39,9 % |
| `8a049c50` | 15 | 592 | 7,3 % |

### 5.3 Gate décisif — LA VISEE PENDANT UN EPISODE D'OCCUPATION ATTESTE

Episodes ATTESTES par la SORTIE (`ScanFilmVehicleEvents`, 105/105 en bande au lot V8) : le debut
est le dernier point de position qui la precede — la primitive du trou de V1a.4. L'EMBARQUEMENT
n'est pas utilise : il est rarissime (1 pour 10 sorties sur `0d76e8f1`).

| film | evenements | episodes attestes | episodes avec >= 1 visee A BORD |
|---|---|---|---|
| `0d76e8f1` | 11 | 8 | **8 / 8 (100 %)** |
| `fccc61cd` | 3 | 2 | **2 / 2 (100 %)** |
| `4898d586` | 18 | 15 | **15 / 15 (100 %)** |
| `51d3ab9f` | 10 | 9 | **9 / 9 (100 %)** |
| `8a049c50` | 1 | 1 | **1 / 1 (100 %)** |
| **TOTAL** | 43 | **35** | **35 / 35 (100,0 %)** |

Le detail episode par episode (`0d76e8f1`) — la colonne de droite est ce que le rejeu a
aujourd'hui :

```
slot=514 siege=0 duree=  8,1 s · visees SANS i0 =  370 (45,4 /s) · lectures i21 AVEC i0 = 0
slot=514 siege=0 duree= 17,9 s · visees SANS i0 =  518 (28,9 /s) · lectures i21 AVEC i0 = 0
slot=515 siege=0 duree= 18,8 s · visees SANS i0 =  397 (21,1 /s) · lectures i21 AVEC i0 = 1
slot=531 siege=0 duree=  5,8 s · visees SANS i0 =  168 (28,9 /s) · lectures i21 AVEC i0 = 1
slot=554 siege=0 duree= 15,0 s · visees SANS i0 =  389 (25,9 /s) · lectures i21 AVEC i0 = 0
slot=551 siege=0 duree= 21,1 s · visees SANS i0 =  107 ( 5,1 /s) · lectures i21 AVEC i0 = 0
slot=559 siege=0 duree=108,6 s · visees SANS i0 = 2241 (20,6 /s) · lectures i21 AVEC i0 = 0
slot=602 siege=0 duree= 44,2 s · visees SANS i0 = 1588 (35,9 /s) · lectures i21 AVEC i0 = 0
```

**5 a 46 lectures par seconde contre 0 ou 1 sur tout l'episode.** C'est exactement l'information
qui manquait au cone de visee d'un occupant.

### 5.4 Gate (c) — CETTE VISEE EST-ELLE DISTINCTE DU CAP DU CHASSIS ?

Cap du chassis pris sur `i1` (direction de velocite, la seule orientation de vehicule VALIDEE :
1,7-2,1 deg au deplacement), vehicule NOMME par la sortie (`VehicleSlot`), fenetre de 20 s avant
la sortie, appariement a 500 ms.

| film | sorties exploitables | paires | mediane | q1 | q3 | part sous 30 deg |
|---|---|---|---|---|---|---|
| `0d76e8f1` | 9 | 3 226 | **21,8 deg** | 9,8 | 46,3 | 62,9 % |
| `fccc61cd` | 2 | 806 | **17,8 deg** | 5,7 | 52,9 | 58,8 % |
| `4898d586` | 15 | 3 879 | **15,7 deg** | 7,3 | 39,6 | 66,1 % |
| `51d3ab9f` | 8 | 2 882 | **19,8 deg** | 7,0 | 42,2 | 64,6 % |

**PASSE, avec sa nuance ecrite.** La visee de l'occupant N'EST PAS le cap du chassis : mediane
15,7 a 21,8 deg, quartile superieur 39,6 a 52,9 deg, et un tiers des instants a plus de 30 deg
d'ecart. Elle en est CORRELEE, ce qui est attendu — le corpus d'episodes attestes est fait de
sieges 0 (des conducteurs), et un conducteur regarde surtout ou il roule. Un artilleur
donnerait un ecart bien plus grand ; **aucune sortie de ce corpus ne porte un siege > 0**, donc
ce cas n'est PAS mesure ici, et je ne l'extrapole pas.

---

## 6. LA REPONSE, PAR ROLE (la question de l'utilisateur)

| role | ce que le rejeu a aujourd'hui | ce que V11 etablit |
|---|---|---|
| **CONDUCTEUR** | un cone oriente par le CAP DU VEHICULE (`vehiclesLayer.vehicleAimAngle`, approximation assumee) | sa VRAIE visee est lisible, a 20-46 lectures/s, et elle differe du cap du chassis de 15,7 a 21,8 deg en mediane (q3 40-53 deg) |
| **ARTILLEUR** | rien | **rien du cote de l'entite tourelle** (§ 3, refute avec temoin) — mais sa visee PERSONNELLE est lisible par le meme canal que celle du conducteur, puisqu'elle vit sur SON slot bipede |
| **PASSAGER** | rien | idem : un passager a son propre slot bipede, donc sa propre visee, pendant tout l'episode |

**Le cone d'artilleur ne viendra donc pas de la tourelle : il viendra de l'homme.** C'est une
bonne nouvelle de conception — la tourelle n'a pas de sprite ni de trajectoire (V8), alors que
l'occupant est deja rattache a son vehicule par le calque.

**Sur la piste 5 (« un passager a toujours une arme »).** Elle n'a pas eu a servir de repli :
la visee continue est disponible, elle est plus riche qu'un cone ephemere au moment du tir, et
elle couvre 100 % des episodes attestes contre les seuls instants de tir. Elle reste utile
comme ORACLE INDEPENDANT (croiser la visee lue a l'instant d'un tir en vehicule avec la
direction tireur -> victime) : **non fait dans ce lot**, faute de temps, et inscrit au registre
des reports (§ 8). L'oracle deja obtenu — 0,2 a 0,5 deg d'accord avec le champ `Point.H` publie,
lui-meme valide par l'oracle du kill — est du meme ordre de force.

---

## 7. CE QUI EST LIVRE, ET CE QUI NE L'EST PAS

### 7.1 Livre (production, additif, contrat INCHANGE)

| fichier | etat | contenu |
|---|---|---|
| `filmdec/offline_aim_only.go` | **NEUF** (219 L) | `BipedAim` + `ScanFilmBipedAimOnly` (film) + `ScanBipedAimRecords` (pur) + `matchAimOnlyRecord` / `ascendingMask` / `aimOnlyCursorToI21`. Toute la chaine de preuve en en-tete. |
| `filmdec/offline_aim_only_test.go` | **NEUF** (123 L) | garde-rail de grammaire sur payload synthetique, **sans environnement** : lecture juste, egalite des conventions d'angle avec `BipedPosition`, masque `i5,i21` (composant anterieur consomme), et **six temoins** (tag != 1 · zeros casses · slot hors bande · masque declarant i0 · masque sans i21 · composant non modelise avant i21) qui doivent tous rendre zero lecture. |
| `filmdec/offline_aim.go` | MODIFIE (+14 L) | extraction de `aimHeadingDegFromRaw` / `aimPitchDegFromRaw` : la convention d'angle avait DEUX appelants a naitre, elle a desormais UN detenteur. **Zero bit change**, `BipedPosition.AimHeadingDeg` / `AimPitchDeg` rendent exactement la meme valeur (verifie par le garde-rail). |

**`SchemaVersion` inchangee (30). Aucun champ publie, aucun `openapi.yaml`, aucun golden, aucun
type web, AUCUN artefact reconstruit** — le document ne change pas, donc l'artefact non plus.

### 7.2 NON livre, et pourquoi — CONFLIT DE PERIMETRE SIGNALE, NON RESOLU

La visee d'un occupant n'a qu'un emplacement correct dans le document : **l'EPISODE
D'OCCUPATION**, c'est-a-dire `replay/vehicle_rides.go` / `vehicle_rides_events.go`. C'est le
seul objet qui connait a la fois l'occupant, son vehicule et sa fenetre ; le point de
trajectoire (`Point.H`) ne convient pas, puisque par definition l'occupant n'a PAS de position
pendant l'episode. Or ces fichiers sont **explicitement hors perimetre** (un autre agent y
travaille au moment de ce lot). `VehicleTrack.Samples` a ete ecarte pour deux raisons : la
visee est celle de l'OCCUPANT et non du vehicule (un vehicule a plusieurs occupants, donc
plusieurs cones), et `vehicle_tracks.go` est a 533 L de dette gelee.

**Je signale plutot que de resoudre a l'aveugle**, comme le mandat le demande. Le pas suivant
est ecrit au § 8.1.

---

## 8. CONDITIONS DE REPRISE (registre des reports)

1. **PUBLIER LA VISEE D'OCCUPANT SUR L'EPISODE D'OCCUPATION.** Forme proposee : sur le `Ride`
   publie, une serie `aim: [{t, h, p}]` decimee au pas de frame (100 ms), alimentee par
   `filmdec.ScanFilmBipedAimOnly` filtre a `[debut, fin]` de l'episode et au slot de l'occupant.
   Cout : un bump de `SchemaVersion`, `openapi.yaml`, les golden, et un rendu web (le cone
   existe deja pour le bipede a pied : c'est le meme). **A faire par le detenteur de
   `vehicle_rides*.go`, ou apres liberation du fichier.**
2. **LE CONE DU CONDUCTEUR DEVRAIT CESSER D'ETRE LE CAP DU VEHICULE.** L'approximation actuelle
   se trompe de 15,7 a 21,8 deg en mediane et de plus de 40 deg au quartile superieur. Le
   remplacement est disponible ; il depend du point 1.
3. **`i2` DU CHASSIS RESTE OUVERT** (§ 2). Deux hypotheses testables et non departagees : les
   deux chemins porteurs de direction n'ont pas le meme encodage ; ou le vecteur 19 bits est le
   HAUT et non l'AVANT. Le signal de `0d76e8f1` (R 0,376 contre 0,032) merite d'etre poursuivi
   — c'est la seule orientation qui vaudrait AUSSI A L'ARRET, ce que `i1` ne donne pas.
4. **L'ORACLE DU TIR (piste 5) N'A PAS ETE POSE.** Croiser la visee lue a l'instant d'un tir en
   vehicule avec la direction tireur -> victime validerait la visee A BORD par une source
   totalement independante. Les 23 tirs en vehicule de `0d76e8f1` en sont le corpus.
5. **AUCUNE SORTIE DU CORPUS NE PORTE UN SIEGE > 0.** Le cas ARTILLEUR / PASSAGER est donc
   etabli par CONSTRUCTION (chaque occupant a son slot bipede, donc sa visee) mais n'est pas
   mesure sur un episode d'artilleur atteste. Un film ou l'evenement de sortie nomme la tourelle
   (les 6 desaccords du lot V8) serait le corpus qu'il faut.
6. **LE BALAYAGE SANS `i0` OUVRE D'AUTRES PORTES, NON EXPLOREES.** Les formes `i1,i3,i25` (3 011
   sur le chassis) et `i5` (3 875 sur le bipede) sont des records reels que le decodeur ignore :
   de la velocite de vehicule et du bouclier de joueur perdus a chaque film. Hors perimetre.

---

## 9. GATES D'EXECUTION

```
gofmt -l internal/                                                  -> sortie VIDE
CGO_ENABLED=0 go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/   -> exit 0
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
    ok  levelup/go-api/internal/analysis/filmdec   1,4 s
    ok  levelup/go-api/internal/analysis/replay   29,1 s
    grep -c '^--- FAIL:'  =  0
```

Fichiers : tous <= 445 L (seuil 500). Fonctions neuves : la plus longue fait 45 L (seuil 80).
`offline_aim.go` passe de 387 a 401 L.

Rejeu des mesures :

```
# piste A — l'oracle du cap i2, deux grammaires (~50 s pour 4 films)
CGO_ENABLED=0 ATT_FILM=<cache> V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site,8a049c50:behemoth,51d3ab9f:launch site" \
  go test ./internal/analysis/replay/ -run TestV11OrientationChassis -v -timeout 120m

# pistes B et C — balayage sans i0, 5 classes (~35 s / film) ; gate (c) inclus
CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS="0d76e8f1,fccc61cd" \
  go test ./internal/analysis/filmdec/ -run "TestV11ScanSansI0|TestV11ConeDistinct" -v -timeout 180m

# la marche sequentielle (piste B, voie 1) — pour reproduire son inexploitabilite
CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS="0d76e8f1" \
  go test ./internal/analysis/filmdec/ -run TestV11TourelleRecords -v -timeout 120m

# piste C — presence, justesse, trous, occupation (~50 s / film)
CGO_ENABLED=0 V11_ROOT=<cache> V11_FILMS="0d76e8f1,fccc61cd,4898d586,51d3ab9f,8a049c50" \
  go test ./internal/analysis/filmdec/ -run TestV11Visee -v -timeout 180m
```

---

## 10. STATUT DES ITEMS DU MANDAT

| item | statut | justification |
|---|---|---|
| A. rejouer l'oracle geometrique d'`i2` sous la grammaire V9 | `[x]` mesure, **ECHOUE** | § 2 — 47,9 / 122,7 / 99,9 / 75,7 deg de mediane, seuil 30. Controle `i1` vert. Rien publie |
| B. `i2` de l'ENTITE TOURELLE | `[x]` mesure, **REFUTE** | § 3 — la tourelle n'emet AUCUN record : 139,6 / 85,5 touches par slot contre 86,3 / 194,2 au FANTOME, formes de masque plates. Gate « decorrelee du chassis » sans objet |
| B-bis. oracle par les TIRS de tourelle | `[!]` non traite | § 8.4 — la piste C ayant repondu, l'oracle du tir n'etait plus le seul disponible ; inscrit au registre |
| C. `i21` sur `ti=40` et sur l'entite tourelle, grammaire corrigee | `[x]` mesure, **ABSENT de ti=40** (2,8 par slot contre 2,1 au bruit) — mais **PRESENT et decisif sur le BIPEDE** | §§ 4-5 |
| D. Ghidra si les mesures hesitent | `[~]` non necessaire | les mesures n'ont pas hesite ; la grammaire d'`i21` etait deja resolue et validee en production |
| GATE (a) oracle geometrique | `[x]` **PASSE** | 0,2 a 0,5 deg d'accord avec le champ `Point.H` publie, 5 films |
| GATE (b) temoin par permutation / decalage | `[x]` **PASSE** | melange : 75,7 a 93,7 deg, R 0,011-0,134 · bande FANTOME : 0,2 a 0,9 lecture par slot contre 46,5 a 231,2 |
| GATE (c) distincte du cap du chassis | `[x]` **PASSE avec nuance** | § 5.4 — mediane 15,7-21,8 deg, q3 39,6-52,9 deg ; correlee mais distincte. Cas artilleur non mesure (aucun siege > 0 au corpus) |
| publication `filmdec` | `[x]` | `offline_aim_only.go` + garde-rail sans environnement |
| publication DOCUMENT + bump + artefacts | `[!]` **NON FAIT — conflit de perimetre signale** | § 7.2 — l'unique emplacement correct est `replay/vehicle_rides*.go`, explicitement hors perimetre. Rien invente ailleurs |
| rapport + entree de thought_log en tete | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |

---

## 11. CR HONNETE

- **Ce que le lot prouve** : le film contient la visee de CHAQUE occupant de vehicule, en
  continu, a 5-46 lectures par seconde, sur 35 episodes attestes sur 35 ; elle est juste a
  0,2-0,5 deg contre la reference deja publiee, et son plancher de faux positifs est de 0,2 a
  0,9 lecture par slot contre 46 a 231 pour le signal.
- **Ce que le lot refute, avec temoin** : la tourelle d'artilleur ne replique rien. Elle est
  muette au sens fort, pas seulement en position.
- **Ce que le lot ne prouve pas** : que le rejeu affiche quoi que ce soit de nouveau. Rien n'est
  publie dans le document, et c'est un choix de perimetre assume, pas un oubli — le fichier
  d'accueil appartient a une autre session.
- **La decouverte de methode, et elle depasse ce lot** : `ScanBipedRecords` exige un `i0`
  absolu et un masque commencant par 0. Cette exigence n'est pas dans le format ; elle rendait
  invisible **22 963 records de visee sur un seul film**. Tout ce que le depot a conclu d'une
  « absence » dans le flux delta bipede est a relire a cette lumiere.
- **Ce qui a failli passer inapercu** : la premiere voie ouverte pour la tourelle (la marche
  sequentielle) recuperait 1 % des records et rendait des masques post-desynchronisation
  portant des index impossibles. Un lot presse y aurait lu « la tourelle porte i41/i42 ». C'est
  le temoin FANTOME du second balayage, et lui seul, qui a evite la conclusion inverse.
