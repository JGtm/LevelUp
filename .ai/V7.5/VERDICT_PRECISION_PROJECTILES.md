# VERDICT — précision par arme pour les armes à projectile (piste E, session 1/2)

> Session de mesure du 2026-08-08, branche `research/v75-precision`, worktree
> `LevelUp-wt-precision`. Timebox décision #6 du master plan : 2 sessions, verdict écrit
> quel que soit le résultat. **Ceci est l'état à la fin de la session 1/2** — il conclut sur
> trois hypothèses et laisse une voie ouverte, avec son plan de test.
>
> Corpus : les 951 films du cache du dépôt principal, lus en LECTURE SEULE. Aucun fichier de
> production touché. Instrument : `apps/go-api/cmd/tmp_pjcnt/` (binaire de recherche).
> Référence externe : `shared.match_participants.shots_fired / shots_hit`, jamais dans un
> critère de décodage.

---

## 0. LES CINQ LIGNES

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

## 5. VERDICT DE FAISABILITÉ AU 2026-08-08

> **La précision par arme des armes à projectile n'est PAS dérivable à un niveau publiable
> dans l'état actuel du décodage.** Le meilleur estimateur produit cette session (déconvolution
> normalisée) est stable hors échantillon mais rate son contrôle positif de 0,02 à 0,095, soit
> 3 fois la tolérance qui rend quatre armes publiables aujourd'hui.

**Ce qui a changé, et qui n'est pas rien :**

- le blocage est **relocalisé** : dénominateur disponible, numérateur absent — le dossier le
  plaçait ailleurs ;
- une piste nommée dans le handoff (le compteur comme source de tirs) est **fermée par
  mesure** ;
- une lecture de l'état de l'art (« le record de projectile est une touche ») est
  **contredite** par deux mesures indépendantes, ce qui évite à la session 2 de partir sur une
  fausse prémisse ;
- une voie **neuve** existe, elle est chiffrée, et ses deux défauts sont nommés.

**Ce qui n'a PAS changé** : le plafond de validation du handoff tient intégralement. Même une
estimation juste ne se **valide** pas par une population à arme dominante — le Needler ne
compte que 2 observations à >= 80 % de pureté. La validation passe par le **contraste
intra-joueur**, comme écrit.

---

## 6. SESSION 2/2 — LE PLAN, DANS L'ORDRE

Trois étapes, chacune avec son critère d'arrêt. **Si l'étape A échoue, le verdict devient
définitivement « non faisable » et la session s'arrête là.**

**A. Réparer le contrôle positif de la déconvolution (le go/no-go).**
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
     qu'elle prétend et le verdict est clos.

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

**Budget** : 1 session. Si A n'est pas tranchée à mi-session, écrire le verdict négatif —
le timebox prime.

---

## 7. CE QU'IL NE FAUT PAS REFAIRE (fermé par cette session)

| piste | pourquoi elle est fermée |
|---|---|
| **Le compteur 7 bits comme source des tirs de projectile** | pas moyen plat entre familles d'armes : Needler 1,3383 contre BR75 1,3545 sur les mêmes matchs. Il mesure la complétude du scan |
| **Traiter le record de projectile comme une touche** | taux de porteur 0,0067 et cadence 83,4 ms (q90 100 ms) : c'est un TIR. Compter les records du Needler comme ses touches surestime sa précision d'un facteur ~4 |
| **La déconvolution au grain MATCH sans normalisation de visibilité** | coefficients inintelligibles (MA40 0,53, BR75 0,61, Sniper 5,04) : la fraction de tirs visible varie de 0,31 à 0,92 selon le mode et entre dans les coefficients |
| **Chercher le dénominateur** | il n'a jamais manqué. Chercher les TOUCHES |

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
