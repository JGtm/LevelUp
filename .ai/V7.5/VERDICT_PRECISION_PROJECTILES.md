# VERDICT — précision par arme pour les armes à projectile (piste E) — CLOS

> Timebox décision #6 du master plan : 2 sessions, verdict écrit quel que soit le résultat.
> **Les deux sessions ont été jouées le 2026-08-08**, branche `research/v75-precision`,
> worktree `LevelUp-wt-precision`. **Le verdict est NÉGATIF et le timebox est consommé.**
>
> Ordre de lecture si vous êtes pressé : **§0** (les cinq lignes) puis **§5** (le verdict) puis
> **§7** (ce qu'il ne faut pas refaire). Le reste est la démonstration.
>
> Ce qui a été fait : cinq hypothèses testées, deux voies de décodage fermées par mesure, une
> voie de décodage reformulée et renvoyée au décodeur, une voie d'inférence menée jusqu'à son
> critère d'arrêt — et raté.
>
> Corpus : les 951 films du cache du dépôt principal, lus en LECTURE SEULE. Aucun fichier de
> production touché. Instrument : `apps/go-api/cmd/tmp_pjcnt/` (binaire de recherche).
> Référence externe : `shared.match_participants.shots_fired / shots_hit`, jamais dans un
> critère de décodage.

---

## 0bis. LE VERDICT EN UNE PAGE  *(écrit à la clôture, il prime sur le §0 historique)*

**La précision par arme des armes à projectile n'est pas livrable.** Ni par décodage — deux des
trois voies sont fermées par mesure, la troisième dépend d'un composant du décodeur qui n'est
pas bit-exact — ni par inférence : le meilleur estimateur construit ici rate son contrôle
positif sur 2 des 4 armes dont on connaît la réponse.

```
CE QUI EST DISPONIBLE, ET NE L'ÉTAIT PAS AVANT CETTE SESSION
  les TIRS par arme, projectiles compris        le compte de records type 105 (cadence
                                                 mesurée = temps de cycle de l'arme)
  la référence API PAR ARME, 6 armes            reconstruite, effectifs publiés (§4quater.2)
  un appariement indice -> xuid rapide          898 films en 3 min, gate 576/576 (§4quater.1)

CE QUI MANQUE, ET C'EST TOUT CE QUI MANQUE
  les TOUCHES par arme sur les projectiles      absentes des records de dégât (82,7 % des
                                                 touches Fiesta n'ont aucun porteur),
                                                 absentes des codes 6/7 (ni arme ni tireur)

LA SEULE VOIE QUI RESTE, ET ELLE N'APPARTIENT PAS À CETTE PISTE
  attribuer l'ENTITÉ projectile à un joueur par sa TRAJECTOIRE — bloqué par
  `object-position-component` de ti=41, non bit-exact (45 bits contre 60). Travail de décodeur.
```

**Ce qui doit remonter au dossier, quel que soit l'avenir de la piste :** la correction d'offset
de `RoundsCorrected` (§4bis.2), l'identité `eventStart+106 == payload bit 110` (§1), l'unité du
record par arme (§3.3), et le fait que la précision par arme est une grandeur que **le moteur
calcule nativement** mais n'expédie qu'à sa télémétrie (§4bis.1).

---

## 0. LES CINQ LIGNES  *(rédaction de mi-parcours, conservée telle quelle)*

1. **Le blocage n'est pas là où le handoff le plaçait.** Le record de tir d'une arme à
   projectile est un **TIR**, pas une touche : le dénominateur par arme est **disponible**.
   C'est le NUMÉRATEUR (les touches par arme) qui manque, et lui seul.
2. **La piste du compteur 7 bits est morte** : son pas moyen ne distingue pas le Needler du
   BR75 (1,3383 contre 1,3545 sur les mêmes matchs). Il mesure la complétude de notre scan,
   pas les tirs invisibles.
3. **Une voie neuve a été ouverte et mesurée** — la déconvolution du taux de touche par arme
   contre le total de touches de l'API du match. Elle est **stable hors échantillon** sur les
   armes à fort volume (Needler 0,2964 ; moitiés 0,3156 / 0,2771) mais **rate son contrôle
   positif** de 0,02 à 0,095 sur les armes dont la précision est connue.
4. **Verdict au 2026-08-08 : NON PUBLIABLE en l'état** au niveau de qualité du dossier
   (± 0,03 hors échantillon). **Non fermée** pour autant : la voie 3 n'a pas encore été
   testée avec ses deux correctifs identifiés.
5. **Session 2/2** : instruire la voie 3 au grain JOUEUR avec contraste intra-joueur, ou
   conclure à l'impossibilité. Le plan précis est au §6.

---

## 1. LE PROTOCOLE, ET LE GATE DE REPRODUCTION QUI LE PRÉCÈDE

L'instrument de cette session **n'est pas** celui des lots précédents : il lit les records
par **balayage de PAQUETS** (`filmdec.WalkPackets`, `pay[0]>>1 == 105`), là où `mu` / `mu.ref`
balayaient les bits à la recherche d'un marqueur 11 bits. Deux instruments qui ne partagent
pas leur méthode de sélection : c'est ce qui donne de la valeur aux accords ci-dessous.

**Gate joué AVANT toute conclusion** — reproduction des chiffres publiés :

```
                                     cette session      publié            source
records / tirs API      Tactical        0,9232          0,9300            7ter.101
records / touches API   Tactical        3,4001          3,4655            7ter.101
records / tirs API      Fiesta          0,3060          0,3147            7ter.101
records / touches API   Fiesta          1,0983          1,1046            7ter.101
cadence inter-record    MA40 AR         83,4 ms         83,4 ms           7ter.80 (2)
cadence inter-record    Mk51 Sidekick  184,1 ms        183,8 ms           7ter.80 (6)
cadence inter-record    Needler         83,4 ms         83,4 ms           7ter.80 (2)
```

**Contrôle d'alignement du compteur 7 bits** (le discriminant : ± 1 bit doit tout détruire) :

```
position          paires    P(pas = +1)   P(pas = 0)
bit 25 (-1)       19 327      0,5452        0,4246
bit 26 (retenue)  19 327      0,8465        0,0002
bit 27 (+1)       19 327      0,0015        0,0002
```

Le profil est celui de 7ter.80 (0,9738 / 0,5037 / 0,0023). L'écart de niveau (0,8465 contre
0,9738) est **attendu et expliqué** : le balayage par paquets voit moins de records que le
balayage bit à bit (1 854 contre 2 501 par film en mode standard), donc plus de trous dans la
chaîne. Il n'affecte aucune conclusion ci-dessous, qui reposent toutes sur des **contrastes
internes** mesurés par le même instrument sur les mêmes matchs.

**Correspondance des repères de bits** (elle manquait au dossier et vaut d'être écrite) :
`eventStart` de l'ancien scanneur = **bit 4 du payload** de filmdec. Donc
`eventStart+22` (compteur 7 bits) = payload **bit 26**, et
`eventStart+106` (le « drapeau de touche » de 7ter.80/7ter.82) = payload **bit 110** —
c'est-à-dire le drapeau que `filmdec/fire_events.go` nomme déjà **« compteurs nuls »** dans
le layout du record type 105. **Même bit, deux noms, deux dossiers.**

---

## 2. HYPOTHÈSE 1 — le compteur de tir porte les tirs qui n'émettent aucun record : `[RÉFUTÉE]`

**L'idée.** Le compteur 7 bits est croissant et par joueur (7ter.80, `[ETABLI]`). S'il avance
sur TOUS les tirs, ses sauts donnent les tirs de projectile qui ne laissent pas de trace, donc
le dénominateur manquant.

**Ce que chaque hypothèse prédit** — c'est ce qui rend la mesure décisive :

| | pas moyen attendu, arme à trace instantanée | pas moyen attendu, arme à projectile |
|---|---|---|
| H1 vraie (le compteur voit tout) | ~1 | ~1/précision, soit **2,5 à 4** |
| H1 fausse | ~1 | ~1 |

**Mesure** — pas moyen du compteur entre deux records CONSÉCUTIFS DE LA MÊME ARME chez le
MÊME tireur (aucune référence API, aucun appariement indice → xuid) :

```
FIESTA, 30 films                         STANDARD, 30 films
arme            records  pas moyen       arme            records  pas moyen
Needler           5 233    1,3383        MA40 AR          28 952    1,2166
BR75              3 047    1,3545        BR75             15 779    1,1654
Pulse Carbine     2 592    1,2630        Mk51 Sidekick     9 890    1,1575
MA40 AR           2 328    1,3862        Needler           1 119    1,2017
Mk51 Sidekick     2 192    1,2520        Bandit Evo        2 153    1,1346
Cindershot          921    1,3018        Pulse Carbine       174    1,1296
M41 SPNKr           565    1,0944        M41 SPNKr            91    1,0278
```

**Le Needler et le BR75 sont indiscernables** (1,3383 contre 1,3545 en Fiesta ; 1,2017 contre
1,1654 en standard), sur 4 960 et 2 863 paires. Il n'existe aucun contraste entre les deux
familles d'armes.

**Ce que le pas mesure réellement** — et le contrôle positif est intégré : le niveau monte de
1,15 (standard) à 1,35 (Fiesta), c'est-à-dire là où le film montre trois fois moins de
records ; et à l'intérieur du mode standard, l'arme qui porte le pas le plus élevé est le
**MA40 AR** (1,2166) devant le BR75 (1,1654) — exactement l'ordre que 7ter.80 avait établi
pour la perte d'échantillonnage des automatiques. **Le pas du compteur mesure la complétude de
notre scan, pas les tirs invisibles.**

**Portée du refus.** Il ne dit pas que le compteur est inutile : la règle R3 de 7ter.80
(récupérer les tirs sautés des automatiques) reste valable, elle porte sur un autre usage.
Il dit que ce champ **ne rend pas les tirs de projectile perdus**, parce qu'il ne les a jamais
comptés.

---

## 3. HYPOTHÈSE 2 — le record d'une arme à projectile est une TOUCHE : `[CONTREDITE]`

C'est l'état de l'art (7ter.101, index §22) : *« l'unité du record dépend de l'arsenal — un TIR
sur arme à trace instantanée, une TOUCHE sur arsenal à projectiles »*. **Deux mesures de cette
session la contredisent au grain de l'ARME**, et le désaccord change le côté du blocage.

### 3.1 Le taux de porteur des armes à projectile est nul

Porteur = record dont le drapeau « compteurs nuls » (bit 110) vaut 0, c'est-à-dire qui applique
réellement du dégât. Si un record de projectile était une touche, **tous** ses records seraient
porteurs.

```
STANDARD, 30 films              FIESTA, 30 films
MA40 AR         0,4405          BR75            0,2898
BR75            0,4242          Mk51 Sidekick   0,2673
Bandit Evo      0,5044          VK78 Commando   0,2574
Mk51 Sidekick   0,3666          MA40 AR         0,2169
--------------------------      --------------------------
Needler         0,0080          Needler         0,0067
Pulse Carbine   0,0000          Pulse Carbine   0,0096
Ravager         0,0073          Ravager         0,0066
Fuel Rod SPNKr  0,0149          Cindershot      0,0033
                                M41 SPNKr       0,0053
```

Les armes à trace instantanée tombent sur la précision réelle (référence API tous modes
standard : **0,4293**) ; les armes à projectile tombent sur **zéro**. C'est la reproduction,
par un instrument indépendant et sur un bit nommé, du résultat de 7ter.80 (9) (Needler 0,0075,
Pulse Carbine 0,0075, Bulldog 0,0000).

### 3.2 La cadence inter-record d'une arme à projectile est son temps de cycle

```
arme            cadence médiane   q90        lecture
Needler            83,4 ms       100,1 ms    temps de cycle du Needler, flux PÉRIODIQUE
Pulse Carbine      66,8 ms       200,4 ms
Plasma Pistol      34,3 ms       216,4 ms
Ravager           350,7 ms      3 454 ms     arme à charge
Cindershot        936,7 ms      7 210 ms
M41 SPNKr       4 905,7 ms     19 804 ms     temps de rechargement d'un lance-roquettes
```

**Un flux de touches n'est pas périodique à 83,4 ms avec un q90 à 100 ms.** La même mesure
rend les cadences nominales connues des armes à trace instantanée (MA40 83,4 · BR75 67,0 ·
Sidekick 184,1 · Commando 150,2), dont deux sont publiées ailleurs au dixième de milliseconde
près : l'instrument est calibré sur des valeurs connues avant de servir sur des inconnues.

### 3.3 Ce que devient l'agrégat de 7ter.101

L'égalité `records ≈ shots_hit` mesurée en Fiesta (1,0983 ici, 1,1046 publié) **reste vraie
comme agrégat** — mais elle s'explique sans changer l'unité du record : le film ne montre que
**0,306** des tirs en Fiesta, et la précision de l'API y vaut **0,283**. Le rapport de ces deux
nombres vaut 1,08. **C'est une coïncidence numérique de famille de mode, et elle ne survit pas
à la ventilation par arme.**

> **STATUT.** `[MESURE]` — mesure avec deux contrôles (taux de porteur ET cadence, deux
> quantités indépendantes), non reproduite par un tiers. Elle **n'invalide pas** le taux de
> remplissage `porteurs/records` de §3quater sur arsenal à trace instantanée (mes chiffres le
> reproduisent : 0,4405 / 0,4242 contre une référence API à 0,4293). Elle **explique** au
> contraire pourquoi sa portée s'arrête en Fiesta : le quotient y est tiré vers zéro par des
> armes dont le record de tir ne porte jamais de dégât. À faire vérifier par un contexte frais
> avant d'amender l'index.

### 3.4 La conséquence, et c'est le résultat principal de la session

```
ce qu'il faut pour une précision par arme     état réel, mesuré cette session
DÉNOMINATEUR   tirs par arme et par joueur    DISPONIBLE — c'est le compte de records,
                                              projectiles compris (c'est déjà ce que
                                              `shared.match_weapon_shots` stocke)
NUMÉRATEUR     touches par arme et par joueur DISPONIBLE sur trace instantanée (porteurs),
                                              ABSENT sur projectile (taux de porteur ~0,005)
```

Le handoff du 2026-07-31 posait le problème comme un problème de touches **non rattachables à
un tireur** (les impacts codes 6 et 7). La formulation exacte est plus étroite : les touches de
projectile ne sont pas rattachables **ni à un tireur ni à une arme**, et le seul flux qui
nommerait les deux — le record de tir — est émis **avant** que le projectile ne touche.

---

## 4. HYPOTHÈSE 3 — déconvolution du taux de touche par arme : `[MESURE]`, insuffisante

**L'idée, et elle n'est dans aucun lot antérieur.** Le film donne, par match et par arme, un
compte de TIRS. L'API donne, par match, le total de TOUCHES toutes armes confondues. Donc :

```
touches_API(match)  =  somme sur les armes W de   p_W  x  tirs_W(match)
```

`tirs_W(match)` est connu, `p_W` est l'inconnue. Des centaines de matchs à mélanges d'armes
différents rendent le système surdéterminé : les `p_W` se résolvent aux moindres carrés.
**Aucun appariement indice → xuid n'est nécessaire** : la mesure vit au grain du match des deux
côtés. Et **le contrôle positif est intégré** — les armes à trace instantanée ont déjà un taux
de touche mesurable ligne à ligne, donc la méthode se juge sur des armes dont on connaît la
réponse.

**Correctif indispensable, découvert à la première passe** : sans lui la méthode ne vaut rien.
Le film ne montre pas la même FRACTION des tirs d'un match à l'autre (0,92 en Tactical, 0,31 en
Fiesta) ; cette fraction entre sinon dans les coefficients. Elle est retirée en normalisant les
tirs décodés de chaque match par son total de tirs API.

```
                      sans normalisation        avec normalisation
R2 hors echantillon    0,8782 / 0,8699           0,9666 / 0,9591
MA40 AR                0,5273                    0,3748    (référence API 0,4196)
BR75                   0,6147                    0,4110    (référence API ~0,43)
Mk51 Sidekick          0,4558                    0,5435    (référence API 0,4491)
Needler                0,1841                    0,2964
```

**Résultat, 891 matchs, 23 armes, moitiés A/B par parité de rang :**

```
arme                  tirs    taux porteur   coef total   coef A    coef B
--- contrôle positif : armes dont la précision est connue ---
MA40 AR          1 096 234      0,2731         0,3748     0,3887    0,3695   réf API 0,4196
BR75               747 806      0,2292         0,4110     0,4057    0,4192   réf non publiée
Mk51 Sidekick      316 669      0,2405         0,5435     0,5333    0,5431   réf API 0,4491
Bandit Evo          32 364      0,3468         0,4870     0,5105    0,4698   réf non publiée
--- armes à projectile : ce que la méthode prétend estimer ---
Needler            173 028      0,0025         0,2964     0,3156    0,2771
Pulse Carbine       70 175      0,0029         0,3956     0,3058    0,4049
Plasma Pistol       35 113      0,0076         0,2821     0,3170    0,2428
Cindershot          34 405      0,0010         0,1657     0,5353    0,1102
Ravager             49 204      0,0026        -0,0357    -0,0140   -0,1012
MLRS-2 Hydra        29 353      0,0011        -1,0576    -0,6423   -1,4978
Skewer              12 320      0,0289        -1,2993    -0,7195   -1,8714
Fuel Rod SPNKr      14 123      0,0024         1,5113     1,3274    1,9135
```

Nulle (touches API redistribuées entre matchs de la MÊME famille de mode, 5 tirages) : écart
moyen des coefficients **0,8637**, soit plus du double des coefficients eux-mêmes. La méthode
ne reproduit donc pas ses coefficients au hasard.

**CE QUI MARCHE.** Les armes à fort volume sortent des estimations **stables entre les deux
moitiés** : Needler 0,3156 / 0,2771 (écart 0,038), Plasma Pistol 0,3170 / 0,2428, MA40
0,3887 / 0,3695. La bande du Needler (0,28-0,32) tombe dans l'intervalle 0,24-0,39 que le
dossier lui prête. **Et l'ordre MA40 < Sidekick est retrouvé** (0,375 contre 0,544), là où le
calcul naïf de §3bis.0 l'inversait.

**CE QUI NE MARCHE PAS, et c'est disqualifiant en l'état.**

1. **Le contrôle positif échoue à la tolérance du dossier.** Les deux seules références API
   publiées telles quelles dans `.ai/GUIDE_WEAPON_SHOTS.md` §3bis.1 sont **MA40 0,4196** et
   **Mk51 Sidekick 0,4491** ; contre elles, les erreurs valent **-0,045** et **+0,095**. La
   liste publiable tient à **± 0,03 hors échantillon**. Une méthode qui manque de 0,095 une
   arme dont on connaît la réponse ne peut pas publier à 0,03 une arme dont on ne la connaît
   pas. *(Les références du BR75 et du Bandit Evo ne sont pas citées dans le dossier — seuls
   leurs biais de film le sont, -0,0108 et +0,0132 ; les reconstruire fait partie du plan de
   la session 2.)*
2. **Le système n'est pas identifiable pour les armes de faible volume** : Hydra -1,06,
   Skewer -1,30, Ravager -0,04, Mangler 1,19. Des taux de touche négatifs sont la signature
   d'une colinéarité non traitée, pas d'un bruit.
3. **La normalisation suppose que la visibilité du film est la MÊME pour toutes les armes d'un
   match.** Elle ne l'est probablement pas (7ter.80 a mesuré une perte spécifique aux
   automatiques). Ce biais entre directement dans les coefficients, et il n'est pas mesuré.

---

## 4bis. CE QUE LE BINAIRE DIT — lecture Ghidra du 2026-08-08 (cible 1 du timebox)

> Lecture statique, HaloInfinite.exe (311 104 fonctions analysées), via l'API HTTP du plugin
> GhidraMCP en lecture seule. **La question posée** : `shots_fired` tel que l'API le rapporte
> est-il une grandeur unique, ou une somme incluant la réconciliation par les munitions ?
> C'était la seule réserve `NON TESTE` de 7ter.81 (1f), et elle pesait directement sur le
> critère « ± 0,03 contre la référence API » de la session 2.

### 4bis.1 LE MOTEUR TIENT UN ENREGISTREMENT DE STATISTIQUES **PAR ARME**

Trois fonctions écrivent dans le **même** enregistrement — base commune, pas de 0xa8, index
résolu par le **même** `FUN_1408df3dc` :

```
  FUN_1408df45c   entry + 0x08  += 1                      TIRS      (ShotsFired)
  FUN_1408df6a4   entry + 0x0c  += 1                      TOUCHES   (ShotsLanded)
  FUN_1410af034   entry + 0x10  += (short)(dMag+dReserve) CORRECTION PAR LES MUNITIONS
```

**Le numérateur et le dénominateur de la précision par arme sont côte à côte dans un seul
enregistrement, indexés par la même arme.** La précision par arme n'est donc pas une grandeur
que nous essaierions de fabriquer : c'est une grandeur que le moteur calcule nativement.

Les deux compteurs ont chacun un chemin de repli identique (`FUN_1408df4f4`, « trouver ou créer
l'entrée »), qui écrit aux mêmes `+0x08` / `+0x0c`.

### 4bis.2 LA CORRECTION N'EST PAS SOMMÉE DANS LE COMPTEUR DE TIRS — et le dossier se corrige

**Correction à 7ter.81 (1f), qui écrit que `FUN_1410af034` ajoute à `entry+0x08` : il ajoute à
`entry+0x10`.** Huit octets d'écart, et ils changent le sens : en mémoire, la réconciliation par
les munitions **ne touche jamais** le compteur de tirs.

Et les trois compteurs sortent comme **trois champs nommés distincts du même événement de
télémétrie** (`FUN_140ad4e74`, événement Microsoft Xbox Telemetry CELL). Les trois chaînes sont
adjacentes en mémoire, espacées de 0x10, et référencées depuis cette seule fonction, dans
l'ordre :

```
  14369eec0  "RoundsCorrected"   <- 140ad4f55, 140ad4f7a
  14369eed0  "ShotsLanded"       <- 140ad4f87, 140ad4f8e
  14369eee0  "ShotsFired"        <- 140ad4f92, 140ad4f99
```

**Conséquence pour le critère de la session 2 : la réserve se lève dans le sens rassurant.**
Rien dans le chemin lu ne somme la correction au compteur de tirs — les deux sont rapportés
séparément. Le critère « ± 0,03 contre la référence API » mesure donc le film, pas un artefact
de la référence.

> **PORTÉE, et il faut la dire :** ce qui est établi porte sur les compteurs **par arme** et sur
> l'événement de télémétrie. Le producteur des noms **agrégés** `TotalShotsFired` /
> `TotalShotsLanded` (143ba7d28 / 143ba7d38, enregistrés chacun dans son propre global par
> `FUN_140181e60` et ses voisins) **n'a pas été tracé**. L'énoncé juste est *« aucune sommation
> dans le chemin lu »*, pas *« prouvé que l'API ne somme jamais »*.

### 4bis.3 POURQUOI LE FILM NE PORTERA JAMAIS CES COMPTEURS — l'argument de fermeture

Ces compteurs vivent dans une structure de **statistiques / télémétrie**, pas dans un composant
ECS répliqué. C'est exactement ce que 7ter.83 (2) avait mesuré par l'autre bout — énumération
exhaustive de **325 noms de composants** et **118 archétypes** du registre `chunk_00`, **zéro**
composant portant une statistique de tir, et côté arme seulement l'ÉTAT (identité par
emplacement, munitions, inventaire de chargeurs, surchauffe). **Les deux lectures se rejoignent :
le moteur calcule la précision par arme et l'expédie à la télémétrie ; il ne la réplique pas.**

Et le chemin de la touche confirme pourquoi elle est hors de portée hors ligne :
`FUN_1408df6a4` résout son arme en **déréférençant une chaîne de handles** — objet de dégât,
puis champ `+0x2f8` vers l'objet propriétaire, puis indice de joueur en `+0x340` ou `+0x2dc`, et
l'indice d'arme par `FUN_1408df3dc`. Le comptage n'a lieu que si cet indice de joueur **égale**
celui du bloc de statistiques (`param_1 + 4`). Autrement dit : au moment de l'impact le moteur
sait parfaitement qui a tiré et avec quoi — **il le sait par le graphe d'objets du serveur**,
celui-là même dont 7ter.88 (6) avait déjà noté qu'il n'est pas répliqué.

### 4bis.4 CORRECTION DE CE QUI PRÉCÈDE — la portée de l'argument était trop large

> **Cette sous-section corrige la première rédaction de §4bis, écrite le même jour.** Elle
> concluait « la voie du DÉCODAGE est fermée par construction du format ». **C'était un saut**,
> et le contre-exemple est dans ce chantier même : l'arme du kill a longtemps été déclarée
> absente du film, puis résolue à 97,6 % — parce que l'AGRÉGAT n'y était pas mais que
> l'information PAR ÉVÉNEMENT y était (le dead-state porte le tag de source de dégât).
> « Les compteurs ne sont pas répliqués » ne dit rien sur « l'événement est irrécupérable ».

Ce que §4bis.1 à §4bis.3 établissent réellement, et pas plus : **les compteurs AGRÉGÉS par arme
ne sont pas répliqués**, et le chemin de comptabilisation des statistiques résout l'arme par le
graphe d'objets du serveur. C'est une corroboration de 7ter.83 (2) par l'autre bout, rien de
plus.

**Le test qui manquait, et qui a été joué.** Une application de dégât produit un record type 105
à table non vide (un porteur). Sur arme à trace instantanée, c'est le record de tir lui-même ;
sur projectile le dégât arrive plus tard, donc le porteur d'impact — s'il existe — est un AUTRE
record, et rien ne garantit qu'il porte l'identifiant d'arme du suffixe commun. On compte donc
les porteurs **sans aucun filtre d'arme**, et on les confronte aux touches de l'API.

```
famille     films  porteurs (tir)  porteurs (TOUS)  touches API   TOUS/touches   gain hors filtre
TACTICAL       22           3 988            4 142        4 841       0,8556           x1,039
STANDARD       40          34 307           36 404       47 792       0,7617           x1,061
BTB            40          49 077           56 535      149 582       0,3780           x1,152
FIESTA         40           5 876            6 564       37 964       0,1729           x1,117
```

**Contrôle positif** : en Tactical (BR75 seul, arsenal hitscan pur) le film rend **0,8556** des
touches de l'API — le déficit de porteurs de ~15 % déjà documenté (§3quater réserve 2). C'est le
plafond de l'instrument, et il est atteint.

**Résultat** : en Fiesta le film n'en rend que **0,1729**. Lever mon filtre d'arme n'ajoute que
**12 %** de porteurs — il aurait fallu un facteur ~5. **31 400 touches sur 37 964 (82,7 %)
n'ont AUCUN porteur dans le flux des records de dégât.** Elles ne s'y cachent pas sous un autre
identifiant d'arme : elles n'y sont pas.

**Ce qui est donc fermé, et ce qui ne l'est PAS :**

```
FERMÉ (mesuré ici)      la voie des RECORDS DE DÉGÂT. Les touches de projectile n'y sont
                        pas, filtre d'arme levé compris.
FERMÉ (mesuré ailleurs) la voie des ÉVÉNEMENTS D'IMPACT codes 6 / 7 : leur corps ne porte ni
                        arme (168 380 observations, zéro exception, 7ter.91/7ter.94) ni
                        tireur (7ter.86 (5a)).
PAS FERMÉ               la voie de la RÉPLICATION. Le handle du code 7 est un SLOT identifié
                        (`objet + 0x114`) et l'enregistrement qui CRÉE ce slot n'a jamais été
                        cherché — `FUN_141fd8460` sérialise ce champ, la table de dispatch a
                        123 codes. C'est la piste nommée par le handoff du 2026-07-31, et ma
                        lecture Ghidra du jour ne l'a PAS touchée : j'ai lu le chemin des
                        STATISTIQUES, pas celui de la RÉPLICATION. Ce sont deux chemins de
                        code distincts, et c'est exactement la distinction qui avait fait
                        rater l'arme du kill.
```

**Ce que cela change pour la piste E : deux des trois voies de décodage sont fermées par
mesure ; la troisième — la réplication du projectile — reste ouverte et porte la même forme que
le précédent qui a résolu l'arme du kill.** L'inférence (§4) reste le repli, pas la seule
option.

---

## 4ter. LA VOIE DE LA RÉPLICATION — étape 0 de la session 2, et elle reformule le problème

> Lecture statique du 2026-08-08, seconde passe. **La question du handoff** : « le handle porté
> par le code 7 est un slot de réplication (`objet + 0x114`) — que porte l'enregistrement qui
> CRÉE ce slot ? » Réponse : la question était mal posée, et ce qu'on trouve à la place vaut
> mieux.

### 4ter.1 `objet + 0x114` EST BIEN LE SLOT DE RÉPLICATION — deux sites d'appel le confirment

`FUN_141fd8460` est le **convertisseur commun handle → slot** de l'émission d'événements : il
résout chaque handle d'entité (`FUN_140471c88`), lit `*(int *)(objet + 0x114)`, et **écarte
l'entité si ce champ vaut -1**. Son `param_2` indexe une table d'événements de pas 0x18
(`&DAT_144989fb0 + param_2 * 0x18`) : c'est bien le code d'événement. **Les codes 6 et 7
passent par là — c'est pour cela qu'ils portent des slots et non des handles.**

### 4ter.2 LE FAIT QUI CHANGE LE PROBLÈME — la touche n'est comptée QUE si le projectile est répliqué

Dans `FUN_142f1c44c` (impact de projectile), le comptage de la touche est gardé par :

```c
lVar9 = FUN_140477aa0(&param_1, 0x20);            // param_1 = handle du PROJECTILE
...
if ((lVar9 != 0) && (*(int *)(lVar9 + 0x114) != -1)) {   // <- le projectile EST REPLIQUE
    if (FUN_140f052cc(lVar10 + 0x180e0, param_1) == '\0') {   // porte de deduplication
        lVar9 = FUN_1404969f0(lVar9 + 0xe0);       // <- LE PROPRIETAIRE, par pointeur
        if (lVar9 != 0) { ... FUN_1408df6a4(...);   // <- le compteur de touches
```

**Conséquence, et elle est structurelle : tout projectile dont une touche est comptée est une
entité PRÉSENTE dans le flux de réplication, avec un slot.** Le projectile n'est pas un objet
fantôme du serveur — il est dans le film.

Et le propriétaire, lui, n'y est pas : il est lu **par déréférencement de pointeur**
(`projectile + 0xe0`), et il n'est **pas** transmis à l'émetteur d'événement. L'appel final
`FUN_140de8cb0(param_1, composant_proj, param_4, ..., cible, ...)` reçoit le **projectile** et
la **cible**, jamais le tireur. C'est la confirmation, côté code, du négatif déjà mesuré sur le
corps des codes 6/7.

L'ossature d'ownership existe pourtant côté moteur : `FUN_1406b312c` remonte de `objet - 0xc`
vers son ancêtre et s'arrête sur le premier qui porte un slot — « le plus proche ancêtre
répliqué ». Elle est appelée depuis le sous-système de dégât (`FUN_1404d6fb4`). Mais c'est une
marche dans le graphe d'objets du serveur, pas un champ répliqué.

### 4ter.3 CE QUE ÇA REFORMULE — on ne cherche plus un CHAMP, on cherche une ENTITÉ

```
  ANCIENNE FORMULATION  « quel champ de l'evenement d'impact nomme le tireur ? »
                        -> ferme, par mesure (corps des codes 6/7) ET par le code (4ter.2)

  FORMULATION JUSTE     « le projectile est une ENTITE REPLIQUEE du film, portant un slot.
                        Comment rattache-t-on cette entite a un joueur ? »
                        -> PAS ferme, et ce n'est meme pas la meme question
```

Et il existe une réponse candidate qui n'est **ni** un champ de propriétaire **ni** un
appariement d'horloge — donc qui échappe aux deux impasses du dossier : **la trajectoire**. Un
projectile part de son tireur. La première position répliquée de l'entité projectile, confrontée
aux positions des joueurs au même instant, attribue le projectile **géométriquement**. C'est
offline-pur, universel, et cela réutilise ce que le chantier rejeu 2D décode déjà.

**Le bloqueur est nommé, borné, et il n'est pas dans cette piste** : `object-position-component`
de l'archétype `ti=41` (i0, `FUN_14076e29c`) n'est **pas bit-exact** — 45 bits contre 60 — et i0
précède tous les autres composants (7ter.84 (5)(6)(7)). Tant qu'il ne l'est pas, aucune position
de projectile ne sort. **C'est un travail de décodeur, pas un travail de piste E.**

---

## 4quater. ÉTAPE A — la déconvolution au grain JOUEUR, et son critère d'arrêt

> Exécutée le 2026-08-08. **898 films exploitables, 8 562 observations (match x joueur),
> 23 armes.** Trois changements sur le §4 : grain joueur au lieu du match, coefficients
> **bornés dans [0,1]**, et la référence API par arme **reconstruite** au lieu d'être citée.

### 4quater.1 L'APPARIEMENT, ET LE GATE QUI L'A SAUVÉ

Le grain joueur exige l'appariement indice de film → xuid. Le résolveur du dépôt
(`weaponv3.ResolveXuidToPI`) relit 64 bits à chaque position de bit et pour chaque xuid : sur
60 films **il ne finit pas en 10 minutes**. Réécrit en recherche d'octets sur les 8 alignements
(`bytes.Index` sur le chunk décalé), il rend la même réponse en trois ordres de grandeur de
moins — **898 films en 3 min 10**.

**Et le gate a servi.** Première version : **277 accords contre 299 désaccords**. Cause : le
résolveur du dépôt retient la première occurrence **en position de bit**, la mienne retenait la
première de **chaque décalage** — deux occurrences différentes du même motif, donc deux
indices différents. Corrigé en prenant le minimum sur les huit décalages :
**576 accords, 0 désaccord.** *Un instrument réécrit pour la vitesse ne vaut que confronté à
celui qu'il remplace ; ici la confrontation a rattrapé une erreur qui aurait contaminé tout
l'aval en silence.*

Couverture : **8 562 joueurs appariés sur 10 296** (83,2 %). Les indices non rattachés au
roster sont **jetés**, jamais devinés.

### 4quater.2 LA RÉFÉRENCE API PAR ARME, RECONSTRUITE

Population à arme dominante >= 80 % des tirs décodés, effectif publié avec le chiffre :

```
arme              joueurs    tirs API   précision API
MA40 AR             1 006     433 621       0,3793
BR75                  750     253 866       0,4111
Mk51 Sidekick         226      42 693       0,4522
Bandit Evo             70      12 457       0,5124
M392 Bandit            37      14 765       0,3211
S7 Sniper              33       2 820       0,2759
```

Le dossier ne citait en clair que MA40 **0,4196** et Sidekick **0,4491** ; on retrouve 0,3793 et
0,4522 sur un corpus et une pureté qui ne sont pas exactement les siens. **Le contrôle positif
passe donc de deux points à six.**

### 4quater.3 LE RÉSULTAT, ET LE CRITÈRE N'EST PAS ATTEINT

```
CONTROLE POSITIF                coef      moitié A   moitié B   réf API    écart
BR75                           0,4155      0,4265     0,4051     0,4111   +0,0043   OK
Bandit Evo                     0,4878      0,4096     0,5778     0,5124   -0,0246   OK
MA40 AR                        0,3469      0,3425     0,3520     0,3793   -0,0324   limite
M392 Bandit                    0,2879      0,2536     0,3141     0,3211   -0,0332   limite
Mk51 Sidekick                  0,4984      0,5127     0,4796     0,4522   +0,0461   NON
S7 Sniper                      0,4390      0,4855     0,3762     0,2759   +0,1632   NON

ARMES A PROJECTILE (indicatif — aucune référence pour les valider)
Needler        0,2238  (0,2111 / 0,2316)      Mangler         0,3769  (0,3227 / 0,4627)
Ravager        0,2145  (0,2675 / 0,1598)      Fuel Rod SPNKr  0,2412  (0,2145 / 0,2487)
Plasma Pistol  0,1747  (0,1749 / 0,1632)      Cindershot      0,1608  (0,1653 / 0,1198)
Pulse Carbine  0,1474  (0,1185 / 0,1862)      CQS48 Bulldog   0,1153  (0,1835 / 0,0684)
M41 SPNKr      0,0687  (0,0510 / 0,2005)      MLRS-2 Hydra    0,0355  (0,0000 / 0,2412)
```

**CRITÈRE D'ARRÊT DU §6 : « MA40, BR75, Sidekick et Bandit Evo à ± 0,03 hors échantillon ».
DEUX SUR QUATRE.** BR75 (+0,0043) et Bandit Evo (-0,0246) passent ; MA40 rate de 0,0024 et le
Sidekick de 0,0161. **Le critère n'est pas atteint — le verdict est donc clos, et il est
négatif.**

**Ce qui a quand même changé, et il faut le dire aussi.** Le grain joueur borné divise l'erreur
du contrôle positif par ~2 à 3 par rapport au grain match (0,045 à 0,095 → 0,004 à 0,046), rend
les coefficients **stables entre moitiés** pour les armes à fort volume (Needler
0,2111 / 0,2316), et supprime les valeurs négatives. Le Needler passe de **0,007** (chiffre naïf
du dossier) à **0,2238**, dans l'ordre de grandeur attendu. **On est passé d'une réponse fausse
d'un facteur 30 à une réponse plausible mais non validable.**

**Trois réserves, et la première est disqualifiante à elle seule :**

1. **AUCUNE NULLE N'A ÉTÉ JOUÉE AU GRAIN JOUEUR.** Le critère ayant échoué avant, la nulle par
   permutation des touches entre joueurs du même match n'a pas été mesurée. **Les chiffres par
   arme à projectile ci-dessus sont donc INDICATIFS et rien de plus** — personne ne doit les
   publier ni les citer comme mesure.
2. **La référence est elle-même contaminée** : un joueur à 80 % de pureté porte 20 % de tirs
   d'autres armes, et §3bis.1 réserve 2 montre que ce biais existe (BR75). Une part du résidu
   est dans la référence, pas dans l'estimation — ce n'est pas une excuse, c'est une limite de
   la méthode de validation elle-même.
3. **Les armes rares restent non identifiables** : Hydra 0,0000 / 0,2412 entre moitiés, SPNKr
   0,0510 / 0,2005. Le grain joueur n'a pas résolu ce point, il l'a seulement borné.

---

## 5. VERDICT DE FAISABILITÉ AU 2026-08-08

> **La précision par arme des armes à projectile n'est PAS dérivable à un niveau publiable
> dans l'état actuel du décodage.** Le meilleur estimateur produit cette session (déconvolution
> normalisée) est stable hors échantillon mais rate son contrôle positif de 0,02 à 0,095, soit
> 3 fois la tolérance qui rend quatre armes publiables aujourd'hui.
>
> **ET LE BLOCAGE A UN NOM, UNE ADRESSE ET UN PROPRIÉTAIRE** (§4ter) : la donnée manquante est
> l'attribution d'une **entité projectile** à un joueur ; la voie candidate est **géométrique**
> (la trajectoire part du tireur) ; et elle est bloquée par un composant précis —
> `object-position-component` de `ti=41`, non bit-exact, 45 bits contre 60. **Ce n'est pas un
> travail de piste E, c'est un travail de décodeur.** La piste E ne peut pas se conclure
> positivement tant que ce composant n'est pas porté ; elle ne doit pas non plus être déclarée
> impossible à cause de lui.

**Ce qui a changé, et qui n'est pas rien :**

- le blocage est **relocalisé** : dénominateur disponible, numérateur absent — le dossier le
  plaçait ailleurs ;
- une piste nommée dans le handoff (le compteur comme source de tirs) est **fermée par
  mesure** ;
- une lecture de l'état de l'art (« le record de projectile est une touche ») est
  **contredite** par deux mesures indépendantes, ce qui évite à la session 2 de partir sur une
  fausse prémisse ;
- une voie **neuve** existe, elle est chiffrée, et ses deux défauts sont nommés.

- la lecture Ghidra (§4bis.1-3) montre que le moteur calcule bien la précision par arme, mais
  dans une structure de statistiques / télémétrie qu'il ne réplique pas ;
- **deux voies de décodage sur trois sont fermées par mesure** (§4bis.4) : les records de dégât
  ne portent pas les touches de projectile (0,1729 des touches API en Fiesta contre 0,8556 en
  Tactical, filtre d'arme levé), et les événements d'impact codes 6/7 ne nomment ni arme ni
  tireur. **La troisième — la réplication du projectile — reste ouverte**, et c'est la piste
  nommée par le handoff.

**Ce qui n'a PAS changé** : le plafond de validation du handoff tient intégralement. Même une
estimation juste ne se **valide** pas par une population à arme dominante — le Needler ne
compte que 2 observations à >= 80 % de pureté. La validation passe par le **contraste
intra-joueur**, comme écrit.

---

## 6. SESSION 2/2 — LE PLAN, DANS L'ORDRE

> **ORDRE RÉVISÉ le 2026-08-08 après §4bis.4.** La première rédaction envoyait la session 2
> directement sur la déconvolution. C'était l'ordre d'un verdict qui croyait le décodage fermé.
> Il ne l'est pas : **l'étape 0 passe devant**, parce qu'elle peut rendre la déconvolution
> inutile — et parce que c'est la seule voie qui porte la forme du précédent « arme du kill ».

**0. La voie de la réplication — FAITE le 2026-08-08, résultat en §4ter.** `[x]`
   Le slot `objet + 0x114` est confirmé, la touche n'est comptée que si le projectile est
   répliqué, et le propriétaire n'est jamais transmis à l'émetteur d'événement. **La question
   du handoff est close** ; elle est remplacée par « comment rattacher l'ENTITÉ projectile à un
   joueur », dont la réponse candidate est géométrique et dont le bloqueur est
   `object-position-component` de `ti=41`.

**0bis. Le report, et il est explicite.** Rendre i0 de `ti=41` bit-exact **n'appartient pas à
   la piste E** — c'est le décodeur (chantier rejeu 2D / filmdec, item déjà décrit en
   7ter.84 (5)(6)(7)). Le report est donc *valide* au sens du contrat d'exécution : dépendance
   externe explicite, pas un « je le ferai plus tard ». À porter au registre du décodeur, avec
   son critère : i0 de ti=41 bit-exact **et** une trajectoire de projectile qui sort.

**A. FAITE le 2026-08-08, critère NON ATTEINT — résultat en §4quater.** `[x]`
   Deux armes de contrôle sur quatre à ± 0,03 (BR75 +0,0043, Bandit Evo -0,0246 ; MA40 -0,0324,
   Sidekick +0,0461). **Le timebox est consommé et le verdict est clos.** L'étape B (contraste
   intra-joueur) n'est PAS jouée : elle était conditionnée à la réussite de A, et la jouer
   quand même reviendrait à chercher une confirmation après un critère raté.
   *Ce qui suit reste écrit comme la marche à suivre si la piste est un jour rouverte.*

**A (spécification d'origine, conservée pour une réouverture éventuelle).**
   - passer au grain **JOUEUR** (et non match) : il multiplie les observations par ~10 et
     décorrèle les mélanges d'armes, ce qui est exactement ce qui manque à l'identifiabilité.
     Coût : l'appariement indice → xuid, qui existe (`resolveFilmPlayerIndices`, mesuré à
     77,0 % en 4v4, 7ter.95) ;
   - contraindre les coefficients dans [0, 1] (moindres carrés **non négatifs**, borne haute)
     — les valeurs négatives actuelles sont de l'information jetée ;
   - restreindre aux armes portant >= 20 000 tirs, et publier la liste des armes écartées
     (jamais de troncature silencieuse) ;
   - reconstruire d'abord la **référence API par arme** des quatre armes publiables, par la
     population à arme dominante >= 80 % de §3bis.1 (le dossier ne cite en clair que le MA40
     et le Sidekick) : sans les quatre, le contrôle positif n'a que deux points ;
   - **CRITÈRE D'ARRÊT** : la méthode doit rendre MA40, BR75, Sidekick et Bandit Evo à
     **± 0,03 de leur référence API, hors échantillon**. En dessous, elle ne mesure pas ce
     qu'elle prétend et le verdict est clos. *(Ce critère est dé-risqué depuis §4bis : rien
     dans le binaire ne somme la réconciliation par les munitions au compteur de tirs, donc
     l'écart mesuré est bien celui du film.)*

**B. Si A passe — valider les armes à projectile par contraste intra-joueur.**
   - deux armes chez le MÊME joueur dans le MÊME match, l'une à trace instantanée (référence
     connue), l'autre à projectile ; nulle par permutation des étiquettes d'arme
     intra-joueur, 200 tirages ;
   - **CRITÈRE** : le contraste réel bat sa nulle 0/200 ET l'écart estimé reproduit celui des
     paires dont les deux armes sont connues.

**C. Écrire le verdict final et amender l'index.**
   - deux amendements sont dus quel que soit le résultat de A et B : la correction de §3.3
     ci-dessus (l'unité du record par arme) et l'identité `eventStart+106 == payload bit 110`
     du §1.

**Budget** : 1 session, partagée — étape 0 sur la première moitié, A (puis B si A passe) sur la
seconde. Si l'étape 0 déborde, elle s'arrête quand même à la mi-session : le timebox prime sur
la piste, et un négatif écrit vaut mieux qu'une piste laissée ouverte sans verdict.

---

## 7. CE QU'IL NE FAUT PAS REFAIRE (fermé par cette session)

| piste | pourquoi elle est fermée |
|---|---|
| **Le compteur 7 bits comme source des tirs de projectile** | pas moyen plat entre familles d'armes : Needler 1,3383 contre BR75 1,3545 sur les mêmes matchs. Il mesure la complétude du scan |
| **Traiter le record de projectile comme une touche** | taux de porteur 0,0067 et cadence 83,4 ms (q90 100 ms) : c'est un TIR. Compter les records du Needler comme ses touches surestime sa précision d'un facteur ~4 |
| **La déconvolution au grain MATCH sans normalisation de visibilité** | coefficients inintelligibles (MA40 0,53, BR75 0,61, Sniper 5,04) : la fraction de tirs visible varie de 0,31 à 0,92 selon le mode et entre dans les coefficients |
| **Chercher le dénominateur** | il n'a jamais manqué. Chercher les TOUCHES |
| **Chercher les touches de projectile dans les records de dégât** | filtre d'arme levé, le film ne rend que **0,1729** des touches API en Fiesta contre **0,8556** en Tactical : 31 400 touches sur 37 964 n'ont aucun porteur. Elles ne s'y cachent pas sous un autre identifiant |
| **Conclure « fermé » depuis l'absence d'un compteur AGRÉGÉ** | c'est le saut qui avait fait rater l'arme du kill : l'agrégat manquait, l'information par événement était là. Un négatif sur les compteurs n'est pas un négatif sur les événements |

---

## 8. OUTILLAGE ET REPRODUCTIBILITÉ

`apps/go-api/cmd/tmp_pjcnt/` — binaire de recherche, `CGO_ENABLED=0`, aucune dépendance DuckDB.
Ce chemin est **gitignoré** par convention du dépôt (`.gitignore:311`) : les sources sont
archivées sous **`.ai/V7.5/outillage/precision_projectiles/`**, avec leur README et les deux
requêtes d'export de référence. Cinq modes, tous sur un corpus plafonné par `-limit` (jamais de
balayage non borné : la bombe RAM du corpus est documentée).

```bash
# référence API (une fois) — duckdb CLI, READ_ONLY
#   match_registry x match_participants restreints aux films du cache -> ref.csv
#   metadata.weapon_labels -> weapons.csv

CGO_ENABLED=0 go build -o tmp_pjcnt.exe ./cmd/tmp_pjcnt/

tmp_pjcnt -films <cache> -ref ref.csv -famille STANDARD -limit 12 -align   # gate d'alignement
tmp_pjcnt -films <cache> -ref ref.csv -famille FIESTA   -limit 30 -hdr     # classes d'en-tête
tmp_pjcnt -films <cache> -ref ref.csv -armes weapons.csv -famille FIESTA -limit 30 -arme
tmp_pjcnt -films <cache> -ref ref.csv -armes weapons.csv -limit 900 -fit -norm -minshots 5000
tmp_pjcnt -films <cache> -ref ref.csv -famille TACTICAL -limit 22 -out m.csv   # bilan par match
```

Coût mesuré : **50 ms par film** en balayage de paquets (300 films en 15,0 s, chronométré) —
un ordre de grandeur sous le balayage bit à bit de 1,65 s/film mesuré en §4 du guide. Aucune
écriture hors du répertoire de sortie ; le cache de films n'est jamais ouvert en écriture.
