# NOTE 5.9 — LES CHAINES DE DONNEES DU SPRINT ET DU SAUT

> Lot 5.9 (post-chantier, suite du 5.7). Branche `feat/decfilm-57`, worktree
> `LevelUp-wt-decfilm-57`, base `c3f01dd4b` (passation 5.7) puis fusion de
> `feat/recherche-decodeur-film` (`875ead234`).
>
> Methode imposee : lecture de FLUX DE DONNEES A REBOURS, du CONSOMMATEUR vers le champ du
> film. Ghidra en lecture seule sur `127.0.0.1:8089`. Aucune base DuckDB. Un film a la fois.

---

## 5.9.1 — LE SPRINT : LA CHAINE EST COMPLETE, DU GRAPHE D ANIMATION AU CHAMP DU FILM

### Ce que la passation 5.7 laissait ouvert

« QUI POSE LE BIT 45 de `obj+0x8b8` ? », avec quinze assignateurs du mot entier a lire un par
un. **La question etait la bonne, la requete ne l etait pas.** Chercher « qui ASSIGNE le mot »
rate un `OR` de lecture-modification-ecriture dont le masque arrive par registre. La requete
juste est **« qui MATERIALISE `1<<45` »** :

| requete | resultat |
|---|---:|
| instructions chargeant l immediat `0x200000000000` dans un registre | **30** sites |
| ... dont le registre est ensuite combine par `OR` avec `[x+0x8b8]` | **1 seul** |

Les quinze assignateurs de la passation n ont pas eu a etre lus un par un : **aucun** d eux
n est l ecrivain. L ecrivain est `FUN_1431a2474`, que la liste de la passation ne contenait
pas. Les deux fonctions « a commencer par » ont tout de meme ete lues en entier : `FUN_1409aac4c`
ne touche que les bits 37 et 15 de ce mot, `FUN_140775a24` que les bits 2, 3, 7 et 8 — et elle
TESTE le bit 45 (`140775e09`) sans jamais le poser.

### La chaine, maillon par maillon

| # | maillon | adresse | ce qui est lu / pose |
|---|---|---|---|
| 1 | condition d animation `transition_conditions_is_sprinting_tlg` | chaine `143bc3e80` | consommateur de depart |
| 2 | fonction de graphe `SpartanAbilityIsSprinting` | chaine `1436f7170`, enregistree par `FUN_140fe6664` | |
| 3 | accesseur | `FUN_142a0c70c` | `obj = FUN_140477618(&poignee,1)` ; rend `(*(u64*)(obj+0x8b8) >> 0x2d) & 1` |
| 4 | **ECRIVAIN DU BIT 45** | **`FUN_1431a2474`** | `1431a263a` : `& 0xffffdfffffffffff` ; `1431a2646` : `| 0x200000000000` ; `1431a2658` : `*(u64*)(obj+0x8b8) = uVar6` |
| 5 | la CONDITION de ce bit | meme fonction | `*(float*)(this+0x5c) > DAT_143cd8370` — **la fraction de sprint** |
| 6 | l ecrivain de la fraction | `FUN_1431a2c94(cur = this+0x5c, cible = DAT_143cd8374, taux = this+0x7c)` | rampe bornee au pas de temps, sous la garde `FUN_1431a1e90(this, unite, sasp+0x10)` |
| 7 | la CLASSE de `this` | `FUN_140583a94(this+8, 0x73617370)` = tag **'sasp'** | type reflechi **`Sprint::SynchronizedTimePointFloatInstance`** (chaine `143e2b358`, enregistrement `FUN_1431a28c8`, descripteur `DAT_144814ea0`, taille **4**, code de type `0xa000001`) |
| 8 | qui ACTIVE cette instance | `FUN_1406c9b1c` — l applicateur d etat replique | sous le bit `0x2000000` du masque : `FUN_14319db80(capacites, replique+0x12e4, 0)` |
| 9 | l activateur | `FUN_14319db80` | `param_2[3]` est un **INDEX DE FENTE** dans la table a trois fentes `capacites+0x1c[0..2]` ; different de la fente courante (`capacites+0x10`) : `FUN_14319d5ec` DESACTIVE puis `FUN_14319e278` ACTIVE ; `-1` = aucune |
| 10 | **LE CHAMP DU FILM** | `ti=35 i57 biped-spartan-ability-component`, `FUN_142f02810` puis `FUN_142f268c4(biped+0x12e4, flux)` | `R(2)` puis `*(char*)(bloc+3) = valeur - 1` |

### Ce que cela etablit, et qui ferme un report ouvert du registre

**L ETIQUETTE D `i57` EST L INDEX DE LA FENTE DE CAPACITE ACTIVE.** Valeurs `-1, 0, 1, 2` :
`-1` = aucune capacite active, `0..2` = la fente active de la table a trois entrees du bipede.
Le report « `i57` : quelle valeur signifie actif ? » (registre, lot R7-b) a donc une reponse :
**ce n est pas un booleen, c est un index de fente**, et « actif » se lit `>= 0`.

Deux corroborations independantes de la cardinalite **3** :

- `FUN_140f8f300` (l autre applicateur du meme bloc, sous le bit `0x1000000`) boucle
  exactement `while (uVar6 < 3)` sur `capacites+0x1c` ;
- `i56 biped-spartan-ability-energy-component` lit un masque **`R(3)`** — un bit par fente.

Et les deux charges d `i57` sont attachees a des FENTES, pas a des etats : `FUN_142f268c4`
appelle `FUN_142f25d78(bloc+0xc)` quand l index vaut `0` et `FUN_142f262d4(bloc+0x1c)` quand il
vaut `2`.

### L octet RUNTIME de l etiquette 3, relu comme demande

`FUN_142f262d4` est le corps de l index `2`. Sa garde est `param_1[2] & 1` puis
`param_1[2] & 0x10`, soit `biped+0x1302`. **Recherche d instructions sur `+0x1302` : ZERO
site dans l image.** L octet n est donc jamais adresse par deplacement constant depuis le
bipede : il est ecrit par l API du bloc (constructeur `FUN_140f03db8`, `FUN_140f03dfc`), que
Ghidra ne replie pas en offset. Le maillon reste **non trouve a `FUN_140f03db8`** ; ce qui
reste a lire est le corps de cette famille de constructeurs.

### Ce qui reste, et c est UNE question

**QUELLE FENTE PORTE LE SPRINT ?** La chaine dit que la fente active est dans le film ; elle ne
dit pas laquelle des trois est `'sasp'`. La table `capacites+0x1c[i]` porte des index de
definition resolus au chargement du match : la reponse est MESURABLE sur film (5.9.3), pas
lisible dans l image.

---

## 5.9.2 — LE SAUT ET L ETAT AERIEN

### La chaine depuis le consommateur

| # | maillon | adresse | ce qui est lu |
|---|---|---|---|
| 1 | fonction de graphe `IsAirborne` | chaine `143757660`, enregistree par `FUN_140dd208c` | |
| 2 | pont | `FUN_142c66744` | `FUN_140477618(idx, 0x1003)` puis `FUN_140769cb4(unite)` |
| 3 | **accesseur** | `FUN_140769cb4` | aerien = `*(char*)(u+0x89b) != 0` **OU** `(*(u16*)(u+0x898) & 0x1000) != 0` |
| 4 | **seul ecrivain de `+0x89b`** | `FUN_1408b19cc` | `INC byte [u+0x89b]` borne a `0xff` ; remis a **0** des qu un contact apparait dans la liste `param_2+0xb10` (pas de `0x6c`) ; range `+0x89c` (ticks aeriens a l atterrissage), incremente `+0x89f` (ticks au sol), efface le bit 12 de `+0x898` |
| 5 | ses SEULS appelants | `FUN_1408b2f90` a `1408b321b` et `1408b32fd` | la resolution de contact du controleur de personnage |

**`+0x89b` EST UN COMPTEUR DE TICKS SANS CONTACT**, pas un drapeau replique. La chaine **n est
pas trouvee a `FUN_1408b2f90`** : ce qui reste a lire est l origine de son `param_2`, le
manifeste de contacts. Si ce manifeste sort du solveur de collision local, l etat aerien est
DERIVE de la position — que le rejeu possede deja par `i0`.

### Le mode de physique de personnage, et une valeur NOMMEE PAR LE JEU

`IsClambering` (chaine `143756f00`) donne `FUN_142c66808` : `FUN_1406b8244(idx) == 2`.
`FUN_1406b8244` rend `*(u32*)(u + 0x2dc + *(u16*)(u+0x2de))` — un **mode de physique de
personnage** indexe. La chaine **`CharacterPhysicsModeClambering`** (`143df73d0`) nomme la
valeur **2**. Le meme mode est teste par l applicateur de posture aux valeurs **6** et **9**.

### `c_biped_airborne_state` : l enregistrement ne porte aucun nom de champ

`FUN_1432226c0` enregistre la chaine `143e2bfc0` et ecrit le descripteur `DAT_144815240`.
**Ses seules references croisees sont les siennes** (idem pour `c_biped_ground_state` /
`FUN_1431be7cc` et pour le descripteur `DAT_144815260` du vehicule) : la reflexion les atteint
indirectement. L enregistrement ne declare AUCUN nom de champ. L instanciation reste **non
trouvee a `FUN_1432226c0`**.

### CE QUE `i55` APPLIQUE REELLEMENT — et ce n est pas une posture

`FUN_1406c9b1c` applique le bloc replique `biped+0x12b4` par `FUN_142f23b20`, qui appelle
`FUN_141fd9b88(octet de genre, bloc, unite)` :

| genre | applicateur | ce qu il fait |
|---|---|---|
| 0 | `FUN_142f23a04` | sous `u+0x8f9 == 4` ou mode `== 9` : poste l EVENEMENT D OBJET **`0xd`** (corps court) ou `FUN_142f28400(bloc)` (corps long, six mots) ; sinon, si mode `== 6`, pose `u[0x2e8 + u16@0x2de] = 1`, sinon `FUN_14320cbb8(u)` ou le drapeau **`0x2b`** |
| 1 | `FUN_142f238d4` | poste l evenement **`0x1d`** (valeur 1) ou **`0x1c`** (valeur 2) |
| 2 | `FUN_142f23978` | poste l evenement **`0x2b`** |
| 3 | en ligne dans `FUN_141fd9b88` | poste l evenement **`0xc`** |

**Les quatre charges d `i55` ne sont donc pas quatre postures : ce sont quatre EVENEMENTS
D OBJET TYPES**, postes par `FUN_14080b870(index d objet, &evenement)`. Cela explique pourquoi
aucune chaine ne s attache a l octet de genre et pourquoi la glose « debout / accroupi / en
l air » ne s est jamais validee : l octet de genre choisit un CANAL D EVENEMENT, pas un etat.

Le nommage des types `0xc`, `0xd`, `0x1c`, `0x1d`, `0x2b` reste a faire. Deux autres membres de
la meme enumeration sont croises ailleurs dans ce lot : `FUN_140eb2eb0(idx, 0x3a)` dans
`Sprint::Update` et `FUN_1406c88a8(u, 0x39)` / `(u, 0x3b)` dans l activation de capacite.

### La verite terrain physique reste la meme

**H = 0,85 m**, montee **0,467 s** (`bfecd02b`) et **0,466 s** (`4f77afc1`), pic **x 10,7** et
**x 3,9** au-dessus des voisins (lot 5.7.5). C est elle qui autorise la publication DERIVEE
decidee par l utilisateur le 2026-09-21 (cf. 5.9.4).
