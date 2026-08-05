# HANDOFF — décodage de l'inventaire vivant (grenades, capacités, armes)

> État au 2026-07-27, fin de session. Branche `feat/filmdec-continuation`.
> Film de référence : `9e8fb31b-ea96-4848-a3b0-03117171d01e` — Cliffhanger, Slayer:Arena Super
> Fiesta, 24/07/2026 19h21. Chunks en cache, capture Cheat Engine complète au dépôt.
>
> **À LIRE EN PREMIER, avant de reprendre quoi que ce soit.**

## 0. L'HORLOGE EST BONNE — c'est la DÉTECTION DES TIRS qui a un trou

Ce paragraphe a d'abord été écrit à l'envers, en concluant à une horloge fausse. La mesure
ci-dessous dit le contraire, et elle est décisive.

**Ce que le décodeur produit** (repère : premier paquet delta du film, `hz = 10`) :

    fil des morts (kill feed)   premiere a 31,6 s    puis 39,9 · 40,7 · 43,0 · 48,5
    pistes de joueur            premiere image 0     derniere 4876
    etats d inventaire          3,4 s .. 419,8 s
    tirs                        66,0 s .. 494,3 s
    lancers de grenade          73,1 s .. 490,9 s

**Ce que l'utilisateur observe en Theater** : le match commence à **00:25**, les premières morts
vers **00:30-00:40**.

**Le fil des morts tombe à 31,6 s, en plein dans la fenêtre annoncée.** Ce calque est décodé par
un chemin totalement distinct (paquets de type 3, événements de temps fort) de celui des tirs.
Il ancre donc l'horloge de façon indépendante :

- l'**identité du film** est confirmée une deuxième fois (déjà : 59 signatures sur 60 sur
  `9e8fb31b`, zéro sur les 948 autres) ;
- la **conversion image → temps est juste** — sinon le fil des morts serait décalé lui aussi ;
- le repère `originUS` est donc correct, et le début de match à 00:25 est cohérent avec un
  premier échange de tirs quelques secondes plus tard.

**LE VRAI DÉFAUT, isolé :** le calque des **tirs** ne produit rien avant 66,0 s, alors que des
morts sont journalisées dès 31,6 s. Il manque **34 secondes de tirs au début du film**, dont au
moins ceux qui ont causé les quatre premières morts. Ce n'est pas un décalage temporel — un
décalage aurait aussi déplacé le fil des morts — mais un **trou de rappel** : la détection des
tirs ne trouve pas ses ancres au début du film. Même symptôme, très probablement, que le rappel
de 23,7 % du gabarit rigide (§4).

**CONSÉQUENCE POUR LA VÉRITÉ TERRAIN : la confrontation EST possible.** L'horloge étant bonne,
la lecture de l'utilisateur à 00:25 correspond à l'image **250**, et le décodeur a des états
d'inventaire de part et d'autre. Ce qu'il ne faut pas faire, c'est ce que j'ai fait : comparer
mes états à 3,4 s (image 34, avant le début du match) à un relevé fait à 00:25.

**À FAIRE EN PREMIER** : reprendre les états d'inventaire dans une fenêtre autour de l'image 250
et les confronter au tableau du §3. C'est immédiat et ça ne demande aucun développement.

## 1. CE QUI EST ÉTABLI, AVEC SA PREUVE

Tout ce qui suit vient d'une **capture Cheat Engine du dispatch des composants** (site
`0x14076CD11`), 975 250 composants journalisés. Fichier : `.ai/V7.5/dumps/ce_run2_cliffhanger.bin.gz`.

### 1.1 Les deux identités qui fondent le reste

    position exacte = paquet.Start*8 + curseur_moteur

Établie par balayage de l'amorce sur 0..8 : seul `+0` produit un parse valide, et il en produit
**249 sur 249**. Vaut pour tout composant.

    largeur consommee = curseur(composant suivant) - curseur(composant courant)

Plus aucune largeur n'a besoin d'être portée depuis Ghidra.

### 1.2 Les grammaires mesurées

| composant | grammaire | preuve |
|---|---|---|
| `i22` comptes de grenades | `R(3)=4` puis **4 × R(8)**, valeurs 0/1/2 uniquement | 249 lectures relues aux positions exactes, compteur à 4 dans 100 % des cas, valeur max 2 |
| `i47` jeu de grenades | `[6 bits masque][3 bits sélection]` | 12 valeurs distinctes ; la sélection appartient **toujours** au masque (12/12) ; 2 bits hauts du masque toujours nuls |
| `i48` capacité équipée | 10 bits, index de palette | 218 lectures, 13 valeurs |
| `i57` capacité active | 2 bits, **bit 0 = interrupteur** | 990 lectures, bit 0 à 48 %, bit 1 à 4 % |
| `i42` sélecteur d'arme | 7 bits | 447 lectures, 7 valeurs, deux dominantes séparées d'un bit (99 et 97, 30 contre 29) |
| `i43`/`i44` armes | identifiant **64 bits à +1**, suffixe `0x42C9679F` à **+33** | 93/192 porteurs, **contrôle négatif à 0/192**, offset dominant 85 fois sur 93 |

### 1.3 L'arme en main est nommée

19 identifiants distincts, dont **16 sur 16 nommés** par `weapon_families.json` — table bâtie par
un chemin de décodage totalement indépendant (balayage des keyframes) :

    Mk51 Sidekick · Plasma Pistol · S7 Sniper · M41 SPNKr · Skewer · MA40 AR · Needler
    Ravager · Pulse Carbine · Shock Rifle · BR75 · MLRS-2 Hydra · Cindershot · Mangler
    Stalker Rifle · Sentinel Beam

Dispersion conforme à un Super Fiesta. Validation croisée entre deux décodages sans étape commune.

### 1.4 La structure du loadout d'armes

    i43 <-> i44 : 77 records ensemble          -> DEUX EMPLACEMENTS
    i43 <-> i45 : 0 ensemble                    -> i45 n est PAS un 3e emplacement
    i42 <-> i43 : 89 ensemble, i43 JAMAIS seul  -> le selecteur accompagne toujours

Sur 65 records portant deux identifiants lisibles : **zéro** où les deux emplacements portent la
même arme. C'est le test qui pouvait réfuter le modèle ; il ne le réfute pas.

### 1.5 i22 et i47 se valident mutuellement

    gren=[0 0 2 0] sel=3      gren=[0 2 0 0] sel=2
    gren=[0 0 0 2] sel=4      gren=[2 0 0 0] sel=1

Le compteur non nul et l'index sélectionné désignent **toujours le même** rang. Deux composants
décodés séparément qui concordent.

## 2. CE QUI N'EST PAS ÉTABLI

| question | état |
|---|---|
| **Quel compteur d'`i22` est quel type de grenade** | NON RÉSOLU. Le test prescrit (« le compte décroît après un lancer ») ne rend que **2 cas** exploitables, tous deux votant « compteur 1 → Spike ». Deux votes ne font pas une table. |
| **Quel index d'`i48` est quelle capacité** | NON RÉSOLU. Les noms n'existent pas dans les fichiers du jeu en build release (mesuré : 1 chaîne lisible sur 11 tags `eqip`, 0 libellé dans 238 tables `uslg`). |
| **Quel bit d'`i42` sélectionne, et dans quel sens** | NON RÉSOLU. Deux valeurs dominantes séparées d'un bit ; le sens reste à trancher. |
| **Les munitions** (`i30`/`i33`, chargeur `i31`/`i34`) | largeurs mesurées, grammaire NON décodée. |
| **Le compteur de réapparition** | pas commencé. |

## 3. LA VÉRITÉ TERRAIN DISPONIBLE

`.ai/V7.5/replay2d/VERITE_TERRAIN_INVENTAIRE_2026-07-27.md` — relevé par l'utilisateur en Theater, **au début
réel du match (00:25)**, pour huit joueurs : grenades, capacité et ses utilisations, arme en
main, chargeur et réserve.

C'est la seule source non circulaire pour nommer les capacités. **Elle est exploitable
immédiatement** : l'horloge est bonne (§0), 00:25 correspond à l'image 250.

## 4. LE VERROU DE PRODUCTION

Tout ce qui précède dépend de la **capture Cheat Engine**, donc du seul film capturé. Les 948
autres films exigent un scan hors ligne. État de ce scan :

| approche | candidats | vrais | précision | rappel |
|---|---|---|---|---|
| bornes du jeu seules | 5 085 | 249 | 4,9 % | 100 % |
| catalogue de valeurs | 3 202 042 | 179 | 0,01 % | 72 % |
| gabarit rigide (depuis le début du record) | 61 | 59 | **96,7 %** | 23,7 % |
| gabarit local (ancré sur i22) | 5 085 | 231 | 5,5 % | **97,9 %** |

Deux compromis opposés, dont l'union n'est pas la solution — il manque un test.

**LA PISTE LA PLUS PROMETTEUSE, non exploitée** : le suffixe `0x42C9679F` d'`i43` est un motif de
**32 bits à valeur imposée**, du même ordre de sélectivité que celui qui fait marcher le scan du
chantier armes (1 accident sur 9 000 000). Et `i43` co-occurre avec `i22` dans 93 % des records
de spawn. **Scanner ce suffixe devrait donner les records d'inventaire sans capture.** C'est
l'étape suivante évidente et elle n'a pas été faite.

## 5. CE QUI A ÉTÉ RÉFUTÉ — ne pas rouvrir

| piste | pourquoi elle est morte |
|---|---|
| Le catalogue de valeurs transposé du chantier armes | Leur sélectivité vient de la LARGEUR du champ (2³² pour 468 valeurs), pas du catalogue. Sur 9 bits (512 pour 8 valeurs) : 3,2 M de candidats. |
| La multiplicité de position comme discriminant | Distributions des vrais et des faux **identiques** (44,1 % contre 45,2 %). Zéro information. |
| `i0` comme cause du désordre | Corrigé de 23 à 47 bits, conforme à la mesure. **Aucun effet** (i22 : 11,971 % → 11,979 %), y compris à l'oracle. |
| Les gros records seraient des NEW | Réfuté : leur en-tête vaut **82 = 1+14+2+1+64**, ce sont des DELTA à masque DENSE. |
| L'appariement compteur → type par les lancers | 53 % de pureté contre 25 % par hasard, sur 15 cas. Signal réel, trop faible. |

## 6. OUTILS PRODUITS

    cmd/tmp_comptruth      LE JUGE : localise chaque lecture d un composant a l octet pres
    cmd/tmp_compsweep      la recette complete des composants, en une passe
    cmd/tmp_liveinv        l inventaire vivant par slot et par image (alimente le POC)
    cmd/tmp_i43probe       la sonde qui a trouve l identifiant d arme dans i43
    cmd/tmp_weaponpair     la correlation arme en main / loadout
    cmd/tmp_gabarit        le gabarit rigide et son balayage de validation
    cmd/tmp_gablocal       le gabarit local ancre sur le composant
    cmd/tmp_codist         distance entre deux composants d un meme record
    cmd/tmp_recgap         en-tete de record mesure, calibration d idLow (= 14)
    cmd/tmp_filmmatch      identifie le film par signature (59/60 contre 0/948)
    cmd/tmp_filmmanifest   manifeste FRAIS via /spectate (aucun chemin Go avant)
    cmd/tmp_findmatch      identifiant complet du match par carte + date
    tools/ce/filmdec_full_capture.lua   la capture large + signature d ancrage

## 7. L'ÉTAT DU POC

`.ai/ETAT_DU_POC.md` dit calque par calque ce qui est affiché et d'où viennent les données.
L'inventaire vivant y est câblé : quatre compteurs de grenade avec le rang sélectionné entouré,
et la capacité en cercle plein quand elle est active. **Sans nom de type ni icône**, faute de
mapping établi.

Artefact publié : https://claude.ai/code/artifact/eb7b8af2-94cb-47c6-9cdb-15af465b12ae

## 8. SI JE DEVAIS REPRENDRE, DANS CET ORDRE

1. **Confronter à l'image 250** (00:25) les états d'inventaire au tableau du §3. Rien à
   développer, l'horloge est bonne et les états existent. C'est ce qui donne d'un coup le
   mapping des grenades ET celui des capacités — les deux verrous du §2.
   Contrôle interne fourni par le relevé : trois joueurs au propulseur et trois au grappin ;
   tout index de capacité attribué doit respecter ce partitionnement.
2. **Calibrer les munitions** sur les huit couples chargeur/réserve du relevé (2/2, 6/6, 8/16,
   12/12, 25/75). Première vérité terrain jamais disponible pour `i30`/`i33`.
3. **Scanner le suffixe `0x42C9679F`** dans tout le film (§4), mesurer précision et rappel
   contre les 192 positions connues d'`i43`. C'est ce qui affranchit de la capture Cheat
   Engine, donc ce qui rend l'inventaire lisible sur les 948 autres films.
4. **Combler le trou de 34 secondes des tirs** (§0). Vérifier d'abord si les tirs manquants
   sont ceux des morts déjà journalisées à 31,6 / 39,9 / 40,7 / 43,0 s — le fil des morts sert
   d'oracle gratuit : chaque mort implique un tir peu avant.
