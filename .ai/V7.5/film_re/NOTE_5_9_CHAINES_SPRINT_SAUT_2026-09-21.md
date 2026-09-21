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

### Ce qui restait, et qui est RESOLU au 5.9.5

**QUELLE FENTE PORTE LE SPRINT ?** La reponse etait dans l image apres tout, et le 5.9.5 la lit :
`FUN_1407e9ce4` aiguille sur le GROUPE DE TAG et appelle trois desenregistreurs paralleles dont
chacun teste l index actif contre SA fente — `'saev'` fente 0, **`'sasp'` fente 1**, `'sagh'`
fente 2. Voir la section 5.9.5 ci-dessous.

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

### Complement au maillon 5 : d ou vient le manifeste de contacts

Lecture de `FUN_1408b2f90` (le seul appelant). Le manifeste passe par deux appels, et le
premier incrementeur est de surcroit GARDE PAR LE TYPE D OBJET :

```
uVar9 = FUN_1405839d0(unite+0x2c, 0x6f626a65)     tag 'obje' de l objet
iVar5 = FUN_1408b44fc(uVar9)
if (iVar5 != 5)  ->  FUN_1408b19cc(idx, 0)         chute libre : aucun contact
...
uVar9 = FUN_1408b14c0(unite)                       la structure de SUPPORT de l unite
uVar4 = FUN_140d988c8(<position>, idx)             une INTERROGATION du monde de collision
if (FUN_1408e2f8c(uVar9, uVar4))  ->  FUN_1408b19cc(idx, uVar9)
```

La chaine est donc **non trouvee a `FUN_140d988c8`** : ce qui reste a lire est si cette
interrogation a une entree REPLIQUEE autre que la position de l objet. Si elle n en a pas,
l etat aerien se recalcule a partir de la position — ce que le rejeu possede deja par `i0`, et
c est exactement ce que le port derive du 5.9.4 exploite.

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

---

## 5.9.3 — LA MESURE DE VERIFICATION

Instrument : `mouvement_5_9_research_test.go` (paquet `grammar`, `//go:build research`). Il
appelle **la fonction de production** `ScanMovementStates` — pas une copie — et confronte ce
qu elle derive a l ORACLE PHYSIQUE du lot 5.7.5. Un film a la fois, porte propre (5.7.4),
largeurs d axe de la carte installees, aucune base DuckDB.

### L oracle de contenu tient sur les deux films (il precede toute conclusion)

| film | carte | records `ti=35` | desyncs | `i0` | `i1` | `i21` | `i25` |
|---|---|---:|---:|---:|---:|---:|---:|
| `bfecd02b` | snowbound | 97 345 | 6 (0,01 %) | 85,5 % | 77,5 % | **65,3 %** | 97,1 % |
| `4f77afc1` | flood gulch | 321 335 | 56 (0,02 %) | 72,6 % | 63,8 % | **69,6 %** | 98,2 % |

### (A) LE SAUT DERIVE, TEL QUE LA PRODUCTION LE PUBLIE

| film | lectures `i1` retenues | montees FERMEES | sauts RETENUS | part | transitions publiees | vies |
|---|---:|---:|---:|---:|---:|---:|
| `bfecd02b` | 75 086 | 1 159 | **198** | 17,1 % | 396 | 52 |
| `4f77afc1` | 201 340 | 5 183 | **966** | 18,6 % | 1 932 | 176 |

La SELECTIVITE est le chiffre qui compte : **une montee sur six** passe la fenetre de hauteur.
Les cinq autres sont des rampes, des chutes, des canons a homme et des oscillations de marche —
la hauteur les separe, et c est tout l interet de la methode.

Les quatre genres publies, cote a cote :

| film | `crouch` | `slide` | `mobility` | `jumpDerived` |
|---|---:|---:|---:|---:|
| `bfecd02b` | 52 lectures / 24 vies | 47 / 26 | 301 / 29 | **396 / 52** |
| `4f77afc1` | 191 / 41 | 222 / 38 | 1 017 / 73 | **1 932 / 176** |

### L ACCORD AVEC L ORACLE — LE RAPPEL EST TOTAL, ET L ECART A UNE CAUSE NOMMEE

Appariement a 40 000 us (deux ticks de 60 Hz), par vie.

| film | sauts publies | apparies | precision | episodes de l oracle dans H | couverts | **rappel** |
|---|---:|---:|---:|---:|---:|---:|
| `bfecd02b` | 198 | 136 | 68,7 % | 136 | 136 | **100,0 %** |
| `4f77afc1` | 966 | 829 | 85,8 % | 830 | 829 | **99,9 %** |

**LA PRECISION N EST PAS UN TAUX D ERREUR, C EST UNE DIFFERENCE DE POPULATION**, et elle se
mesure : la passe de recherche filtre la vitesse sur les slots LIES AU BIPEDE et en jette une
part, la porte de production non.

| film | lectures `i1` vues par la production | vues par l oracle apres filtre | ecart |
|---|---:|---:|---:|
| `bfecd02b` | 75 086 | 55 044 | **20 042** |
| `4f77afc1` | 201 340 | 169 981 | **31 359** |

Les 62 et 137 sauts « sans episode en face » vivent dans cet ecart : ce sont des sauts de vies
que l instrument de recherche ecarte, pas des sauts inventes. **Le seul episode de l oracle non
couvert** sur les deux films est `4f77afc1` slot 646 a 7 797 284 348 us — une lecture de vitesse
que la porte de production ne retient pas. Un sur 966.

### (B) `i57` — LA DENSITE DU CHAMP DU SPRINT

| film | etiquette `-1` | `0` | `1` | `2` | total | part des records |
|---|---:|---:|---:|---:|---:|---:|
| `bfecd02b` | 318 | 7 | 297 | 2 | 624 | **0,64 %** |
| `4f77afc1` | 1 457 | 61 | 1 330 | 32 | 2 880 | **0,90 %** |

**LA DISTRIBUTION EST QUASI BINAIRE**, et c est exactement ce que la chaine du 5.9.1 predit :
l etiquette est un INDEX DE FENTE, et sur ces deux matchs une seule fente travaille. `-1`
(aucune capacite active) et `1` (la fente 1 active) s equilibrent — 318 contre 297, 1 457 contre
1 330 : **une pose pour une levee**, la signature d un interrupteur. Les fentes `0` et `2` sont
quasi inutilisees (7 et 2 ; 61 et 32).

**UNE CORROBORATION DE LA GRAMMAIRE, TOMBEE DE LA MESURE** : le nombre d observations
`i57.reference` EGALE exactement le nombre d etiquettes `0` — 7 = 7 sur `bfecd02b`, 61 = 61 sur
`4f77afc1`. C est ce que `FUN_142f268c4` ecrit : la charge supplementaire `FUN_142f25d78(bloc+0xc)`
n est lue que lorsque l index vaut `0`. La grammaire portee et l ecrivain disent la meme chose.

**CE QUE CELA NE DIT PAS** : laquelle des trois fentes porte `'sasp'`. 297 et 1 330 activations
suffiraient a DATER les usages d une capacite ; il manque le NOM. Deux voies restent ouvertes, et
aucune n est une lecture de plus dans l image : un releve Theater sur un match dont l equipement
est connu, ou un film ou une seule capacite est distribuee.

---

## 5.9.4 — LE PORT : LE SAUT, PUBLIE ET DIT DERIVE

Decision de l utilisateur du 2026-09-21. **Schema 65 -> 66**, `grammar-2026-09-21.5`,
`killsource-2026-09-21.5`.

| ce qui est porte | ou |
|---|---|
| le genre `jumpDerived` dans `stances[].kind` | `types/grammar_mouvement.go` |
| la constante `SpartanJumpHeightM = 0.85` (+ tolerance, seuil de montee, borne de tenue), chacune avec sa mesure datee en commentaire | `types/grammar_mouvement.go` |
| la derivation elle-meme (integration de la vitesse verticale tenue, segmentation, fenetre de hauteur) | `grammar/movement_states_jump.go` (fichier neuf) |
| la capture d `EtatVitesse`, que le balayage jetait | `grammar/movement_states.go` |
| `coverage.stances.jumpEpisodes` et `.jumpsDerived` | `replay/document_stances.go` + jumeau servi `domain/replaydoc` |
| les libelles « Saut (derive) » / « Jump (derived) », et la priorite d affichage (le saut passe devant) | `apps/web/.../i18n.ts`, `stanceLogic.ts` |
| l entree de chronique v66, qui DIT que c est derive et ce qui le remplacera | `replay/document_chronicle.go` |

**DEUX REFUS ECRITS DANS LE CODE, chacun avec son test** : un episode encore OUVERT a la fin de
la marche n est pas publie (pas d instant de fin mesure) ; un silence de replication de plus de
250 ms n est pas une vitesse tenue. `movement_states_jump_test.go` les tient, avec un RATCHET sur
le nom du genre — le renommer en `jump` ferait passer un calcul pour une lecture.

**CE QUI N EST PAS PORTE** : le sprint. Sa chaine est complete, sa densite est mesuree, la fente
n est pas nommee, et aucune derive n a ete autorisee pour lui.

**CE QUI RESTE AU PILOTE** : le re-figeage des ENTREES (`inputs_*.bin.gz`), qui demande un
decodage. Les goldens d assemblage et les huit fixtures de contrat ont ete refiges depuis les
entrees EXISTANTES : leur seul ecart est `schema 65 -> 66`, et ils ne porteront de `jumpDerived`
qu apres ce re-figeage. C est attendu, c est le piege 6 de la passation 5.7, et c est dit ici
pour que personne ne le lise comme une derivation muette.

---

## 5.9.5 — LA FENTE DU SPRINT EST NOMMEE, ET LE SPRINT EST PUBLIE **LU**

### (b) LE MAILLON QUI MANQUAIT, ET IL SE LIT CHEZ L ECRIVAIN

La recherche du tag `'sasp'` (`0x73617370`) dans l image rend **24 sites**. L un d eux n est pas
un consommateur mais un AIGUILLAGE : `FUN_1407e9ce4` prend une definition de capacite, lit son
GROUPE DE TAG (`FUN_1405a602c`) et appelle, pour chacun des trois groupes, un desenregistreur
different sur le composant de capacite de l unite (`FUN_14049d444`) :

| groupe | 4CC | desenregistreur | fente | offset dans le composant | index actif teste |
|---|---|---|---|---:|---:|
| `0x73616576` | **`saev`** esquive | `FUN_14319d0ac` | **0** | `comp+0x1c` | `comp+0x10 == 0` |
| `0x73617370` | **`sasp` SPRINT** | `FUN_14319d1ec` | **1** | `comp+0x20` | `comp+0x10 == 1` |
| `0x73616768` | **`sagh`** grappin | `FUN_14319d14c` | **2** | `comp+0x24` | `comp+0x10 == 2` |

Les trois fonctions sont RIGOUREUSEMENT PARALLELES — meme corps, meme forme, seuls changent
l offset de fente et l index compare. Chacune ne desactive proprement que si **l index actif est
le sien** : c est cette comparaison qui LIE le groupe de tag a l index, et elle ne laisse aucune
ambiguite.

`FUN_1408decb0` confirme par un second aiguillage sur les memes trois groupes
(`FUN_14319cdbc` / `FUN_14319ce7c` / `FUN_140a0f3c4`).

**Consequence, et c est la reponse du lot** : `comp+0x10` est l index actif, `FUN_14319db80` le
pose depuis `bloc+3`, et `FUN_142f268c4` ecrit `bloc+3 = R(2) - 1`. Donc **le brut `2` d `i57`
signifie « le sprint est actif »**, le brut `3` le grappin, le brut `1` l esquive, le brut `0`
aucune capacite.

### (a) `i48` NE POUVAIT PAS REPONDRE, ET LA LECTURE LE DIT

`FUN_1406d0ff0` lit `R(3)` compteur + porte `R(1)` + `R(6)` rang — **un seul** rang dans la
palette `sofd` du match, pas trois fentes. `i48` designe la capacite d ARMURE equipee ; il ne
dit rien de l ordre des fentes du composant. La voie (a) est donc close, par lecture.

### (c) LA VERIFICATION DE CONTENU — CE QU ELLE PROUVE ET CE QU ELLE NE PEUT PAS PROUVER

Instrument `mouvement_5_9_sprint_research_test.go`, porte propre, population retenue.

**Repartition des lectures d `i57` (slots lies au bipede)** :

| film | brut 0 aucune | brut 1 `saev` | brut 2 **`sasp`** | brut 3 `sagh` | total |
|---|---:|---:|---:|---:|---:|
| `bfecd02b` | 209 | 1 | **205** | 1 | 416 |
| `4f77afc1` | 1 152 | 12 | **1 088** | 49 | 2 301 |

Une pose pour une levee, sur les deux films : c est un interrupteur, et c est ce qu un index de
fente active doit etre.

**LE CONTROLE GRATUIT DU GRAPPIN, ET IL EST FRANC.** Vitesse au sol pendant les intervalles de la
fente 2, sur `4f77afc1` (41 intervalles, 15 vies) :

| | p10 | mediane | p90 |
|---|---:|---:|---:|
| DANS les intervalles de la fente 2 | 1,06 | 2,22 | **5,84 m/s** |
| HORS | 0,95 | 2,28 | 2,88 m/s |

**Deux fois le plateau de course au p90.** C est la TRACTION du grappin, et aucune autre capacite
ne fait cela. Si la fente 2 est bien le grappin, **la lecture de l index de fente est juste** —
donc la fente 1 est bien `'sasp'`. C est la preuve croisee la plus forte disponible, et elle
porte sur le MEME champ, le MEME pliage, le MEME instrument.

**LE SPRINT, LUI, NE PEUT PAS ETRE PROUVE PAR SA VITESSE — ET LE DEPOT LE SAVAIT DEJA.**

| film | intervalles fente 1 | vies | duree mediane | jugeables | precision vs plateau | rappel |
|---|---:|---:|---:|---:|---:|---:|
| `bfecd02b` | 177 | 34 | 0,82 s | 160 | **37,5 %** | 60,8 % |
| `4f77afc1` | 895 | 143 | 1,00 s | 727 | **42,0 %** | 40,5 % |

Seuil LU sur la mesure, pas choisi : `Vm` 2,31 / `Vs` 2,85 -> 2,58 m/s (`bfecd02b`) ; `Vm` 2,55 /
`Vs` 2,84 -> 2,69 m/s (`4f77afc1`). **L ecart entre marche et sprint est de 0,29 a 0,54 m/s**, et
la dispersion intra-classe le couvre. C est exactement le negatif du lot 5.3.5 (« la distribution
de vitesse au sol n a QU UN SEUL mode ») : **demander au plateau de valider le sprint, c est
demander a un instrument deja mesure comme non discriminant de trancher.** Le score bas ne dit pas
que la fente est mal nommee — il redit que la vitesse ne separe pas.

Ce qui CONVERGE quand meme, et sur les deux films :

| film | vitesse MAX dans l intervalle (p10 / mediane) | temoin apparie (p10 / mediane) | victoires |
|---|---|---|---:|
| `bfecd02b` | **2,67** / 2,96 m/s | 2,00 / 2,76 m/s | 63,8 % |
| `4f77afc1` | **2,73** / 2,92 m/s | 1,91 / 2,79 m/s | 62,1 % |

Le temoin est apparie : meme vie, meme duree, cinq secondes plus tot. La mediane de la vitesse max
DANS l intervalle vaut 2,92-2,96 m/s — le plateau haut. Et le p10 dit que **neuf intervalles sur
dix atteignent au moins 2,67 m/s**, contre 1,91-2,00 pour les temoins.

**Deux controles qui n ont PAS conclu, et il faut le dire** : la duree brute des intervalles monte
a 166,7 s et 869,4 s (une fente reste armee pendant un silence de replication) — la borne de tenue
de 250 ms ramene le maximum a 6,4 s et 13,1 s, mais **ne change pas le score**, donc la duree
n etait pas la cause. Et l exclusion « on ne sprinte pas accroupi » porte sur 0 et 10 poses
d accroupi : population trop mince, et de toute facon sprint + accroupi = GLISSADE dans ce jeu.

### CE QUI EST PORTE

`stances[].kind` gagne **`sprint`**, LU — dans la MEME montee de schema 66 que `jumpDerived`, et
la chronique v66 porte les deux en les SEPARANT par nature. Le decalage `+1` vit en un seul point
(`sprintAbilitySlotRaw`), avec son ratchet. Libelle « Sprint » / « Sprint ».

**Mesure de la production** (`ScanMovementStates`, `bfecd02b`) : **616 lectures `sprint` sur
65 vies**, contre 52 `crouch`, 47 `slide`, 301 `mobility` et 396 `jumpDerived`. C est le canal LU
le plus dense du calque.

### LE REPORT QUI RESTE, ECRIT AVEC SON ADRESSE

`abilityActive[]` — publier la fente active comme un CANAL A PART, avec ses trois noms — n est
PAS fait. Les fentes 0 et 2 sont nommees par l image au meme titre que la 1, mais leur population
est mince (1 et 1 sur `bfecd02b`, 12 et 49 sur `4f77afc1`) et le lot n a pas mesure l esquive.
Le croisement du grappin avec les `grappleLines[]` du lot 3.7 (41 intervalles contre les paires
d `i59` etiquette 3) reste a faire : c est le controle qui fermerait la fente 2 nommement.
