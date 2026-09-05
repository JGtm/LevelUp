# RAPPORT — lot V3 item A : DATER LA DESTRUCTION D'UN VEHICULE PAR LA MORT DE SON CONDUCTEUR

> Execute le 2026-09-02 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB, AUCUN fichier de production modifie
> (deux `*_test.go` neufs). Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v3`), donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data`, lecture seule). `internal/himap/`,
> `cmd/vs-measure/`, `cmd/vehicle-sprite/`, `sprites_v4/` : NON touches.

## 0. LE RESULTAT EN CINQ LIGNES

1. **LE GATE ECHOUE. La destruction d'un vehicule n'est PAS datable par la mort de son
   conducteur** avec les primitives disponibles. Sur 460 vies de vehicule (12 films, 3 regimes),
   **0 destruction datee**. Les huit gates ecrits avant mesure sont verdis un par un au § 3.
2. **LE POSTULAT DE DEPART EST REFUTE SUR PIECES, et c'est le fait le plus utile du lot** : la
   FIN DE TRAJECTOIRE d'un vehicule n'est PAS le moment ou il cesse d'etre conduit. Le flux de
   position du vehicule continue **12 a 36 s (mediane, par lot)** APRES que son occupant l'a
   quitte. Un vehicule replique encore une fois abandonne, et peut etre repris (la vie
   `slot=771` de `0d76e8f1` porte **trois** occupants successifs).
3. **LE CONDUCTEUR NE MEURT QUASIMENT JAMAIS A BORD** : sur 80 trous d'embarquement, **3 (3,8 %)**
   contiennent une mort de LEUR occupant, contre **17 (21,3 %)** pour le TEMOIN a occupant
   decale sur le MEME intervalle. Le signal est **ANTI-correle (-17,5 pts)**, pas absent au
   hasard : tout embarquement se termine par une SORTIE, et l'occupant survit.
4. **CE QUI EST, LUI, CONFIRME A L'ECHELLE DU CORPUS** : **69 / 80 trous d'embarquement (86,3 %)**
   portent un evenement de sortie de leur occupant, et cette sortie ferme le trou a +/-2 s dans
   **69 / 69 = 100 %** des cas. C'est le recoupement V2b (10/10 sur un film) reproduit sur
   12 films — la chaine « trou de position <-> occupant <-> evenement de sortie » est SAINE.
   Le **temps a bord** en decoule directement : mediane **11,2 a 12,5 s** selon le lot.
5. **CONSEQUENCE SUR LE LIVRABLE** : l'item A3 (point d'entree de PRODUCTION) etait conditionne
   au passage du gate. Le gate ne passe pas : **A3 n'est pas ecrit** — publier une « cause de fin
   de vie = destruction » sur un signal refute serait exactement le « dead code museum » que la
   revue interdit. Ce qui MERITE une production est autre chose (§ 6), et se decide au niveau
   superviseur.

---

## 1. LE GATE, ECRIT AVANT LA MESURE

Il vit dans l'en-tete de `apps/go-api/internal/analysis/replay/vehicules_v3_destruction_test.go`
(etage 1) et de `vehicules_v3_trous_test.go` (etages 2 et 3), en constantes nommees — aucun
nombre magique. Rappel integral.

### 1.1 Etage 1 — la definition d'une DESTRUCTION DATEE

Une vie de vehicule V (recensement des images-cles, cle `(slot, gen)`, fenetre bornee
`[premiere image-cle .. premiere image-cle APRES la derniere]`) est DETRUITE ET DATEE a `t_mort`
si les quatre conditions tiennent :

1. **FIN SERREE** — V porte une trajectoire dans sa fenetre ; `t_fin` = dernier echantillon de
   position du slot dans cette fenetre (~0,5 s, contre +/-20 s pour le recensement seul).
2. **OCCUPANT COURANT** — il existe un bipede O dont un TROU du flux de position s'OUVRE a moins
   de `attBordRayonM` (1,5 m) de V pendant la fenetre (primitive V1a.4 / V1c rejouee telle
   quelle), dont le trou COUVRE `t_fin`, et qui n'a AUCUN evenement de sortie entre l'ouverture
   du trou et `t_fin`.
3. **MORT DE L'OCCUPANT** — O meurt a `t_mort` avec `|t_mort - t_fin| <= v3dFenetreMS` (3 s),
   horloge calee par le pont de production (`buildOwners` -> `DeathOffsetMS`).
4. **COHERENCE SPATIALE** — derniere position de V a moins de `v3dRayonMortM` (12 m) de la
   position de O a `t_mort`, OU O n'a aucun echantillon a +/-2 s de sa mort (signal attendu d'un
   occupant EMBARQUE, V1a.4). Distance par `dist3` (geometry.go) via l'adaptateur `v2dDist` —
   jamais de formule recopiee (garde-rail `TestUneSeuleFormuleDeDistance3D`).

| gate | enonce | seuil |
|---|---|---|
| **1** | part DETRUITE (vies a occupant courant) au-dessus du TEMOIN a occupant decale | `> +10 pts` |
| **2** | mediane `\|t_mort - t_fin\|` | `<= 5 000 ms` |
| **3** | part des destructions spatialement coherentes | `>= 90 %` |

**TEMOIN (etage 1)** : le meme test, mais avec les morts d'un AUTRE joueur — rotation
deterministe de 3 crans dans le roster trie. Il conserve la densite de morts d'un joueur reel et
ne change QUE l'identite : c'est exactement l'hypothese « au hasard » du lot V2 item 4b.

### 1.2 Etage 2 — le trou d'embarquement (gate ecrit apres le negatif de l'etage 1, avant sa mesure)

L'etage 1 a rendu **0 occupant courant** : le flux du VEHICULE survit systematiquement a la
sortie de son occupant. L'etage 2 retourne la question : **l'occupant meurt-il PENDANT qu'il est
a bord ?** Le trou `[gs, ge]` est l'intervalle d'embarquement.

| gate | enonce | seuil |
|---|---|---|
| **4** | part des trous contenant une mort de LEUR occupant, au-dessus du temoin (occupant decale, MEME intervalle) | `> +10 pts` |
| **5** | mediane `\|t_mort - t_fin du vehicule\|` sur les morts a bord | `<= 5 000 ms` |
| **6** | *controle* V2b a l'echelle du corpus : part des trous fermes par une sortie de leur occupant (+/-2 s) | publie, non bloquant |

### 1.3 Etage 3 — la sortie est-elle une mort ? (gate ecrit apres le negatif de l'etage 2)

L'etage 2 a montre que la mort a bord est ANTI-correlee, et que 86-100 % des trous se ferment sur
une sortie. Hypothese : quand l'occupant est TUE, le moteur l'ejecte, la sortie est emise, le flux
reprend — la mort tombe donc AU BORD du trou, pas dedans.

| gate | enonce | seuil |
|---|---|---|
| **7** | part des trous dont l'occupant meurt a `+/-3 s` de la FERMETURE, au-dessus du temoin | `> +10 pts` |
| **8** | mediane `\|t_mort - t_fin du vehicule\|` sur ces cas | `<= 5 000 ms` |

---

## 2. LE CORPUS ET LA MESURE

12 films du corpus V1a, trois regimes, mesures en trois lots de 4 (le decodage coute 100-170 s par
film ; l'agregat est publie par lot).

| lot | films | regime |
|---|---|---|
| A | `0d76e8f1` beh., `fccc61cd` LS, `4898d586` beh., `e1bdb97f` beh. | Super Fiesta |
| B | `32a37698` beh., `e3b10d4b` beh., `51d3ab9f` LS, `d99e5dbd` LS | Super Fiesta |
| C | `e232ffce` beh., `b232e02d` beh., `d332c3a9` beh., `c6250266` beh. | Slayer / Team Slayer / Fiesta (NON-SF) |

### 2.1 Couverture — etage 1

| lot | vies recensees | avec trajectoire | avec candidat occupant | **occupant COURANT** |
|---|---|---|---|---|
| A | 166 | 55 | 26 | **0** |
| B | 200 | 63 | 21 | **0** |
| C | 94 | 38 | 17 | **0** |
| **corpus** | **460** | **156 (33,9 %)** | **64 (13,9 %)** | **0 (0 %)** |

Aucune vie n'a d'occupant encore a bord a sa fin serree. Les 64 vies a candidat sont donc toutes
classees SORTIE ; les 396 autres INCONNUE (304 sans trajectoire du tout, 92 avec trajectoire mais
sans candidat — la primitive V1c n'attribue que 15-21 % des vies, limite publiee au rapport V1).

### 2.2 Etage 2 — les 80 trous d'embarquement

| lot | trous | fermes | duree du trou (mediane) | fermeture -> fin serree du vehicule (mediane) |
|---|---|---|---|---|
| A | 30 | 30 | **12 095 ms** | **+25 659 ms** |
| B | 25 | 25 | **12 463 ms** | **+12 762 ms** |
| C | 25 | 25 | **11 181 ms** | **+35 752 ms** |
| **corpus** | **80** | **80** | ~12 s | **+13 a +36 s** |

**C'est la refutation du postulat** : la mediane de l'ecart « l'occupant descend » -> « le
vehicule cesse d'emettre » vaut 13 a 36 s selon le lot, jamais zero.

| lot | MORT A BORD | TEMOIN (occupant decale, meme intervalle) | ecart |
|---|---|---|---|
| A | 0 / 30 = 0,0 % | 6 / 30 = 20,0 % | **-20,0 pts** |
| B | 2 / 25 = 8,0 % | 5 / 25 = 20,0 % | **-12,0 pts** |
| C | 1 / 25 = 4,0 % | 6 / 25 = 24,0 % | **-20,0 pts** |
| **corpus** | **3 / 80 = 3,8 %** | **17 / 80 = 21,3 %** | **-17,5 pts** |

Le controle V2b (gate 6) tient partout :

| lot | trous avec sortie de leur occupant | dont la sortie ferme le trou (+/-2 s) |
|---|---|---|
| A | 27 / 30 = 90,0 % | **27 / 27 = 100 %** |
| B | 25 / 25 = 100 % | **25 / 25 = 100 %** |
| C | 17 / 25 = 68,0 % | **17 / 17 = 100 %** |
| **corpus** | **69 / 80 = 86,3 %** | **69 / 69 = 100 %** |

### 2.3 Etage 3 — la mort au BORD du trou

| fenetre | reel (corpus) | TEMOIN | ecart |
|---|---|---|---|
| +/-1 000 ms | 6 / 80 = 7,5 % | 6 / 80 = 7,5 % | **0,0 pt** |
| **+/-3 000 ms (gate 7)** | **17 / 80 = 21,3 %** | **11 / 80 = 13,8 %** | **+7,5 pts** |
| +/-10 000 ms | 33 / 80 = 41,3 % | 29 / 80 = 36,3 % | +5,0 pts |
| +/-20 000 ms | 41 / 80 = 51,3 % | 51 / 80 = 63,8 % | -12,5 pts |

Detail par lot a +/-3 s : A `5/30 = 16,7 %` vs temoin `4/30 = 13,3 %` (**+3,3**) ; B `7/25 = 28,0 %`
vs `3/25 = 12,0 %` (**+16,0**) ; C `5/25 = 20,0 %` vs `4/25 = 16,0 %` (**+4,0**). **Un seul lot sur
trois franchit le seuil** — l'ecart corpus (+7,5) reste sous 10 pts, et la dispersion inter-lot
(3,3 / 16,0 / 4,0) est du bruit d'echantillon, pas un signal.

Ecart SIGNE mort - fermeture du trou, sur les cas retenus : mediane **+990 ms** (A, n=5),
**+1 671 ms** (B, n=7), **+2 307 ms** (C, n=5) — toujours POSITIF : quand une mort tombe la, elle
SUIT la fermeture. Coherent avec « le joueur descend puis se fait tuer a pied », pas avec « il
meurt dans le vehicule ».

Gate 8 (`|t_mort - fin serree du vehicule|`) : mediane **78 911 ms** (A, n=5), **14 012 ms**
(B, n=7), **3 464 ms** (C, n=5). Un lot sur trois sous 5 s, sur 5 echantillons : rien de datable.

---

## 3. VERDICT DES HUIT GATES

| gate | enonce | mesure corpus | verdict |
|---|---|---|---|
| 1 | destruction datee > temoin +10 pts | **0 vie a occupant courant** — indeterminable | **ECHOUE** |
| 2 | mediane `\|t_mort - t_fin\|` <= 5 s | 0 datation | **ECHOUE** |
| 3 | coherence spatiale >= 90 % | 0 datation | **ECHOUE** |
| 4 | mort a bord > temoin +10 pts | **-17,5 pts** (3,8 % vs 21,3 %) | **ECHOUE (anti-correle)** |
| 5 | mediane `\|t_mort - t_fin\|` (morts a bord) | 70 s / 267 s sur n=2 et n=1 | **ECHOUE** |
| 6 | *controle* la sortie ferme le trou | **69 / 69 = 100 %** (86,3 % des trous portent une sortie) | **PASSE** |
| 7 | sortie par la mort > temoin +10 pts | **+7,5 pts** (21,3 % vs 13,8 %) | **ECHOUE** |
| 8 | la mort date la fin du vehicule | mediane 3,5 / 14 / 79 s selon le lot, n<=7 | **ECHOUE** |

**VERDICT GLOBAL : la voie « destruction datee par la mort du conducteur » est REFUTEE sur ce
corpus.** Le seul gate qui passe est le CONTROLE (6), et il valide la chaine d'observation, pas
l'hypothese.

---

## 4. POURQUOI, MECANIQUEMENT — ce que la mesure a appris

1. **Un vehicule replique sa position bien apres avoir ete quitte** (mediane +13 a +36 s). La fin
   de trajectoire est une MISE AU REPOS, pas une disparition — exactement le piege deja documente
   pour l'equipement (`EquipmentLifeSpan.T1US`, commentaire de `equipment_creation_width.go`).
   Toute datation appuyee sur « le flux s'arrete » est donc fausse pour les vehicules.
2. **Un slot de vehicule porte plusieurs occupants successifs** dans une meme vie recensee
   (`0d76e8f1` `slot=771` : 512 puis 514 puis 515). « L'occupant d'une vie » n'existe pas ; il
   faut raisonner par EPISODE d'occupation (le trou), et c'est ce que l'etage 2 fait.
3. **Les joueurs sortent vivants** : 86,3 % des episodes se terminent par un evenement de sortie
   qui ferme le trou a la milliseconde, et l'occupant ne meurt ni dedans (3,8 %, sous le temoin)
   ni juste apres au-dela du hasard (+7,5 pts). En arene Halo Infinite, le vehicule sert de
   moyen de transport plus que de tombeau — la regle metier « si le vehicule est detruit, le
   joueur meurt aussi » n'est pas fausse, mais l'evenement « vehicule detruit AVEC occupant » est
   trop rare dans ce corpus pour porter une datation.
4. **La non-attribution reste le plafond** : 64 vies a candidat sur 460 (13,9 %). Meme si la
   correlation existait, elle ne couvrirait qu'un septieme des vehicules.

---

## 5. CE QUI EST LIVRE, CE QUI NE L'EST PAS

| item | statut | justification |
|---|---|---|
| A1 gate ecrit AVANT mesure | `[x]` | en-tete des deux instruments, constantes nommees, temoin defini avant. Trois etages, huit gates, chacun ecrit avant sa propre mesure. |
| A2 instrument garde + mesure corpus + classement | `[x]` | `V3_DESTR_FILMS` / `V3_DESTR_ROOT` (+ `V3_DESTR_DUMP`, `V3_DESTR_BRUT`), skip propre sans env. 12 films, 3 regimes, 460 vies, 80 episodes d'occupation. Classement publie (§ 2). |
| A3 point d'entree de PRODUCTION | `[!]` **NON FAIT** | **Le plan le conditionne explicitement au passage du gate** (« Si le gate passe : brancher un point d'entree de PRODUCTION »). Le gate echoue sur 7 de ses 8 branches. Ecrire `ScanFilmVehicleLives` avec un champ « cause = destruction » sur un signal refute produirait du code mort porteur d'une fausse promesse (anti-pattern n°1 de la grille de revue). Ce qui merite une production est different, et se decide au niveau superviseur : § 6. |
| A4 rapport + thought_log + tests verts | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` ; `go test ./internal/analysis/replay/ ./internal/analysis/filmdec/` EXIT=0, `grep -c '^--- FAIL:'` = **0**. |

---

## 6. CE QUI MERITERAIT UNE PRODUCTION (a trancher par le superviseur, non ecrit ici)

La mesure a valide, a l'echelle de 12 films, un objet qui n'est PAS la destruction :
**l'EPISODE D'OCCUPATION**.

```
type VehicleRide struct {
    OccupantSlot uint32   // bipede, pont slot->xuid deja detenu par le rejeu
    VehicleSlot  uint32   // vehicule le plus proche a l'ouverture du trou (< 1,5 m)
    BoardUS      uint64   // ouverture du trou de position (~0,5 s)
    ExitUS       uint64   // evenement de sortie (A LA MILLISECONDE, 100 % de recoupement)
    Seat         uint32   // R(6) de l'evenement de sortie (0 = conducteur, 93,8 % V2b)
}
```

Faits qui le portent : 86,3 % des episodes ont leur sortie, 100 % de ces sorties ferment le trou
a +/-2 s, temps a bord median ~12 s. Il donne au rejeu 2D : qui conduit quoi, quand il monte,
quand il descend, et la COULEUR D'EQUIPE du vehicule pendant l'episode. **Il ne donne pas la
destruction** — et le champ « cause de fin de vie » ne doit PAS etre invente.

Ce qu'il faudrait pour dater la destruction, et qui reste ouvert :

1. **La grammaire de bits d'i2 et i3 pour `ti=40`** (Ghidra), sans quoi i4 (integrite) reste du
   bruit — condition de reprise deja posee par V2b § 2.5. C'est LA voie directe.
2. **L'evenement de degats/mort d'objet dans la liste d'evenements** : l'histogramme de tete
   (`PORT_LISTE_EVENEMENTS` § 2) montre un **type 0 « degats » a 1 313 occurrences** sur
   `0d76e8f1`, jamais decode. Si sa charge porte la cible et la letalite, elle daterait la
   destruction a la milliseconde, sans passer par le conducteur. Voie non exploree, moins chere
   que (1) car le cadrage de la liste est deja porte et prouve bit-exact.

---

## 7. INSTRUMENTS ECRITS (neufs, aucun code de production modifie)

| fichier | lignes | test | gardes |
|---|---|---|---|
| `apps/go-api/internal/analysis/replay/vehicules_v3_destruction_test.go` | 494 | `TestV3DestructionDatee` (etage 1 : le gate ecrit avant mesure) | `V3_DESTR_FILMS`, `V3_DESTR_ROOT`, `V3_DESTR_DUMP`, `V3_DESTR_BRUT` |
| `apps/go-api/internal/analysis/replay/vehicules_v3_trous_test.go` | 411 | etages 2, 3 et 4 du meme test, plus le rapport de l'etage 1 | idem |

L'ETAGE 4 (« le devenir d'un embarquement », gate B5) a ete ajoute apres coup, dans le cadre de
l'item B : il mesure si l'occupant d'un EMBARQUEMENT date meurt pendant son trou. Il vit ici parce
que seule cette couche detient le pont slot -> xuid et le calage d'horloge du fil des morts. Ses
chiffres sont publies dans `V3_EMBARQUEMENT_2026-09-02.md`.

Reutilisation stricte, aucune grammaire ni aucun calage recopie : `v0Corpus`/`v0Bornes`,
`v1aBandeVehicule`/`v1aOptions`/`v1aPistes`, `v1cVies`/`v1cGapStartsNearVehicles`/`v1cAttribue`,
`attVehiculeLePlusProche`/`attEcartUS`/`attPart`, `indexBySlot`/`slotTrack.at`, `v2dTightEnd`,
`v2dDist` (adaptateur `dist3`), `ScanFilmDeaths`/`ScanFilmPlayerIndices`/`injectiveOrEmpty`/
`buildOwners`, `filmdec.ScanFilmVehicleEvents`, `filmdec.ScanFilmWorldObjectKeyframes`,
`filmdec.ScanFilmBipedPositions(ForBand)`.

Un piege corrige en cours de route, note car il fausse toute mesure de trou : la primitive V1c ne
compte les trous que sur les echantillons `HasWorld`, alors que `indexBySlot` garde TOUT. Lire la
fermeture d'un trou sur le flux complet la fait tomber immediatement apres l'ouverture et rend
« occupant deja parti » partout. `v3dMondeSeul` aligne les deux flux.

Commandes de rejeu (avant-plan, GOCACHE isole, `CGO_ENABLED=0`) :

```
V3_DESTR_ROOT=<data>/cache \
V3_DESTR_FILMS="0d76e8f1:behemoth,fccc61cd:launch site,4898d586:behemoth,e1bdb97f:behemoth" \
  go test ./internal/analysis/replay/ -run TestV3DestructionDatee -v -timeout 60m

# diagnostic par vie (pourquoi un candidat n'est pas l'occupant courant)
V3_DESTR_DUMP=1 V3_DESTR_ROOT=... V3_DESTR_FILMS="0d76e8f1:behemoth" go test ...

# nuage vehicule en FLUX BRUT (post-filtres desarmes) : mesure de sensibilite, +1 vie sur 45
V3_DESTR_BRUT=1 V3_DESTR_ROOT=... V3_DESTR_FILMS="0d76e8f1:behemoth" go test ...
```

Suite sans environnement (tout skippe) :

```
go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
ok  levelup/go-api/internal/analysis/filmdec   1.559s
ok  levelup/go-api/internal/analysis/replay   28.942s
EXIT=0 · grep -c '^--- FAIL:' = 0 · gofmt -l vide · go vet propre
```
