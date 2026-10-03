# Lot `positions_limbe` — véhicules qui « partent tout seuls », joueur qui « vole » depuis hors carte

Enquête en lecture seule, commit `43a01721e`, 2026-09-23. MESURÉ = sortie chiffrée ou fichier:ligne ; DÉDUIT = conséquence
directe de mesures ; HYPOTHÈSE = non tranché, avec la sonde qui tranche.

Pièces (toutes dans `SP/sondes/positions_limbe/`) : `sweep.mjs`/`sweep_out.txt` (balayage des 111 documents), `bits.mjs`/
`bits_out.txt` (chaînes de bits des valeurs aberrantes), `temoin_bits.mjs` (témoin), `census.mjs`/`census_out.txt`
(recensement final + durées d'affichage), `probe1.txt`/`probe2.txt` (sonde Go sur 3 fichiers de faits : 81c02726,
ab526724, 879a4dba), `maps.tsv` (carte par match, copie DuckDB). Sonde Go : worktree
`C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-inv-positions_limbe` (détaché, `43a01721e`), fichier non commité
`apps/go-api/internal/games/halo_infinite/film/replay/limbe_positions_research_test.go` — à retirer par le superviseur.

---

## 1. Constat reproduit sur pièces

### 1.1 Les deux cas de l'utilisateur (81c02726, Strongholds sur Isolation, décalage 22 700 ms)

**Mongoose 770** (châssis `af31ab1a`, jamais occupé) — MESURÉ (`vsamples.mjs 81c02726 770`) : échantillons immobiles au
spawn (-42.07, -21.42, 118.35) jusqu'à t=279 (affiché 0:05.2), puis UN échantillon t=983 (1:15.6) en
(-201.16, 88.24, z=-235.46), puis retour au spawn à t=1035 (1:20.8) et immobile ensuite.
Le client interpole 279→983 (70,4 s, 193 m à 2,7 m/s) : le Mongoose part vers la gauche/le haut dès 0:05.2. Il sort du
cadre (≈ 0:09 à 0:15 selon la taille de la toile) et `edgeMarkFor` le PLAQUE à la marge
(`layers/vehiclesPaint.ts:512-513`) : il glisse le long du bord puis reste bloqué dans le coin. C'est le
« s'immobilise hors zone jouable ». Il y reste jusqu'à 1:15.6, puis revient au spawn en 5,2 s.
Mongoose 768 : même chose, t=812 (0:58.5, vue 11 s) et t=983 (1:15.6, interpolation 914→1691 = **77,7 s**).

**Madina97294** (slot 523) — MESURÉ : la vie publiée commence à t=645 (0:41.8) en (-78.6, 46.38, **z=-325.4**) ;
le point suivant arrive à t=692 (0:46.5) au vrai spawn (-1.72, -22.1, 118.8), à 456 m en 4,66 s. Le client affiche
d'abord la flèche hors cadre (nom + distance), puis interpole sur 4,7 s jusqu'au spawn : le joueur « vole ».

### 1.2 Ce ne sont pas des positions : la même suite de bits sur des cartes et des axes différents

MESURÉ (`bits.mjs`, catalogue `map_quant_bounds.json`). Isolation et Starboard partagent la boîte du canevas Forge
(59 cartes Forge : X[-231.00, 231.64] Y[-227.08, 226.35] Z[-946.22, 242.33], largeurs 15/15/17). Les valeurs
aberrantes ne sont ni aux bornes ni au quantum 0 ou maximal : (-201.16, 88.24) ↔ q=(2113, 22787) ;
(217.3, -200.53) ↔ q=(31752, 1918). La table par défaut (±20000, 22 bits) est exclue : le balayage rejette `index == -1`.

Une fois ces valeurs réécrites en bits (X|Y|Z aux largeurs de chaque carte), elles se répètent **à l'identique d'une
carte à l'autre, même quand les largeurs et les bornes diffèrent** :

- **Motif A** : `000100001000001101100100000011…`, les 30 à 35 premiers bits identiques, au même décalage.
  - Isolation (15/15/17) : Mongooses 768 et 770 (-201.16, 88.24).
  - Fortitude (15/15/17) : Mongoose 774 (-201.16, 88.26).
  - Behemoth (**17/17/15**) : Warthog 771, Mongoose 768 d'un autre film, et le bipède 532 (-548.95, 297.4).
  - Mêmes bits, donc coordonnées « monde » différentes : ce ne sont pas des positions.
- **Motif B** : période de 30 bits `011111000001000000011101111110`. Le champ Z (17 bits) vaut exactement les 17 premiers
  bits de la période suivante.
  - Cartes Forge : Empyrean, Rat's Nest ×2, Fortitude, Curfew, Isolation, Critical Dewpoint, Starboard (Madina,
    ab526724), Snowbound. Les valeurs y sont récurrentes : -6.79, 217.3, y=-200.53, z=-370.22.
  - Même motif sur Launch Site (17/17/15, Wraith 784 : champ **X** = 63520 = la valeur du champ **Z** des cartes
    Forge), Behemoth et Recharge.
- **Témoin** (`temoin_bits.mjs`) : 320 échantillons NORMAUX des mêmes documents.
  - Normaux : 1 paire sur 49 600 partage ≥ 24 bits au même décalage.
  - Aberrants : 62 paires sur 1 926 (×1 600) ; 22 des 64 échantillons aberrants appartiennent à un motif transverse.

### 1.3 Recensement sur le parc (111 documents, `census.mjs`)

Garde utilisée : la règle de `boundsOf` (p1..p99 ± 12 étendues centrales, `replay/geometry.go:219-246`).
Durée fautive calculée selon la mécanique du client :
- trace : interpolation sans limite (`lib/replay/replayLogic.ts:29-48`) ;
- véhicule : maintien du dernier échantillon jusqu'à `t1max`, repli sur `spawn` avant le premier échantillon
  (`model/vehiclesLayer.ts:391-411`) ;
- vie close : flèche de mort pendant 2,5 s (`layers/replayMarkers.ts:289-304`).

| Catégorie | Cas | Documents | Hors garde | Affichage fautif |
|---|---:|---:|---:|---|
| T1 premier point d'une vie de bipède aberrant (> 25 m et > 30 m/s du 2e) | 9 | 9 | 7/9 | 0,5 à 4,7 s (23 s au total) |
| T2 vie d'un seul point, hors garde | 22 (sur 51 vies d'un point) | 21 | 22/22 | 2,6 s chacune (pion d'une image + flèche de mort) |
| T3 excursion à l'intérieur d'une vie | 1 (3 points) | 1 | 1/1 | 8,6 s |
| T4 dernier point aberrant | 3 | 3 | 1/3 | 3,8 à 16,8 s |
| V1 échantillon de véhicule en aller-retour (voisins d'accord, lui à > 25 m) | 11 (10 vies) | 7 | 10/11 | **1 344 s**, médiane 75,6 s, max 419 s |
| V2 vie de véhicule fantôme (tous les échantillons hors garde) | 5 | 4 | 5/5 | **1 948 s**, 270 à 561 s chacune |
| V3 naissance (`spawn`) hors garde | 11 (6 sans aucun échantillon) | 3 | 11/11 | **3 684 s** d'affichage au faux spawn |

**Documents touchés : 33 sur 111.** Dénominateurs : 11 443 vies, 3 508 787 points ; 514 vies de véhicule,
127 255 échantillons. Ce ne sont pas des chutes réelles : celles de Behemoth (771/800/804, z → -45) sont continues,
dans la garde, et le recensement ne les compte pas.

---

## 2. Cause racine (prouvée)

**Trois balayages ancrés bit à bit publient des « records » dont la position n'en est pas une**
(`ScanBipedPositionsForBand` pour les bipèdes et la bande `ti=40`, `ScanVehicleCreationsForBand` pour les naissances).
Les filtres statistiques en aval les laissent passer, puis le client les interpole sur des silences de plusieurs
dizaines de secondes.

1. **Le balayage.** Le lecteur essaie l'en-tête bipède à CHAQUE position de bit du paquet delta
   (`internal/grammar/delta_biped_walk.go:72-91`). Ses critères : préfixe, slot dans la bande, 2 bits nuls, compte de
   masque, index croissants, porte i0 de 5 bits nuls (`offline_biped.go:268-314`). Aucune validation du record entier.
   Le dépôt le dit lui-même : « sur des millions de positions de bit un motif conforme finit par apparaître »
   (`offline_filters.go:9-18`). §1.2 prouve que les bits lus ne sont pas une position quantifiée par la carte.
2. **Les filtres le laissent passer** (`offline_biped_band.go:97-101`) :
   - `DropIsolated` garde tout point dont le voisin le plus proche est à ≤ 15 s. Le Mongoose 770 à t=983 a un vrai
     voisin 5,2 s après ; Madina à t=645 a le sien 4,66 s après.
   - `DropTeleportsExcept` accepte TOUJOURS le premier point d'un slot (`offline_filters.go:128-164`, condition `a.ok` ligne 145).
     C'est le cas de Madina : premier point du slot 523 dans le film.
   - Après un long silence, la vitesse calculée est faible : 193 m / 70 s = 2,7 m/s. Au retour, 37 m/s < 100 m/s.
     Madina : 456 m en 4,66 s = 97,8 m/s, juste sous le seuil de 100.
3. **Le client amplifie.**
   - `positionAt` interpole linéairement quel que soit le trou (`replayLogic.ts:29-48`).
   - `Point.g` est publié : « la piste ne doit donc pas être interpolée au travers », `replay/document_aim.go:51-65`.
     Or le client ne le lit **nulle part** (grep `\.g\b` sur `apps/web/src` : aucun lecteur).
   - Le véhicule est plaqué à la marge (`vehiclesPaint.ts:512`) et tenu jusqu'à `t1max`. Sans échantillon, il est dessiné
     au `spawn`.

**Ce que le film dit (grammaire), mesuré sur les faits persistés** (`probe1.txt`, `probe2.txt`) :

- **Pas de delta avant la création.** Les ouvertures de vie qui PRÉCÈDENT le record de création de leur slot : 1 sur 53
  (81c02726), 1 sur 144 (ab526724), 0 sur 219 (879a4dba). Ce sont exactement les deux points de Madina, 4,65 s et 9,49 s
  avant la création de son corps.
  - La création coïncide avec le vrai placement. Délai création → première position valide : médiane 10 à 30 ms,
    max 0,02 s (81c02726) et 0,98 s (879a4dba).
  - Avant-match : 22 à 31 s (lobby). En match, un seul cas à 8,9 s (ab526724, slot 620), et le film n'y écrit aucune
    position entre les deux.
  - Le mot « limbes » ne s'applique donc pas : **aucun corps n'existe quand le point aberrant est écrit.**
  - La règle était déjà mesurée au lot E2 : « AUCUN record de création ne tombe dans l'intervalle d'une vie »
    (`replay/identity_registry_creation.go:178-184`). Elle n'est simplement pas appliquée : `viesOuvertes` accepte une
    création qui tombe DANS la vie (celle de Madina, 645→741, est ouverte par la création de t≈692).
- **Parc (identité publiée).**
  - Les 31 cas T1+T2 sont tous dans une vacance du slot :
    - 9 vies ouvertes par une création postérieure à leur premier point ;
    - 12 vies d'un point avant la première création du slot ;
    - 10 vies d'un point après la fin de la vie précédente, nommées par propagation.
  - 20 vies du parc sont entièrement antérieures à la première création de leur slot. **Toutes font 1 ou 2 points** :
    aucune vraie vie n'est dans ce cas.
- **Véhicules (échantillons en aller-retour, V1).**
  - Aucun record de création du slot à ± 200 ms de l'échantillon. Le recensement déclare le véhicule vivant sur les deux
    images-clés qui l'encadrent (770 : 894,4 et 1094,5).
  - 5 cas sur 5 ne portent AUCUN des composants capturés : vitesse (i1), orientation (i2), vitalité.
  - Leurs voisins portent i2. Mais cette signature vide existe aussi sur 0,7 à 1,7 % des échantillons normaux :
    nécessaire, pas suffisante.
- **Aucun véhicule ne se déplace pendant un silence.** Écarts > 5 s entre deux échantillons :
  - 565 au total ; 21 avec un déplacement > 2 m ;
  - **les 21 touchent tous un échantillon aberrant** (20 allers ou retours d'excursion, 1 dernier échantillon aberrant :
    879a4dba, v821) ;
  - aucun déplacement réel pendant un silence de plus de 5 s sur le parc.

**Origine exacte des bits** : HYPOTHÈSE, non nécessaire pour corriger. Candidats, par ordre de plausibilité :

- (a) Faux en-tête dans la vue C (contrôle) ou dans la liste d'événements. Ce sont des flux d'une autre grammaire,
  balayés avec celle du gestionnaire d'entités (`.ai/V7.5/film_re/NOTE_5_13_VUES_ET_RANGS_2026-09-22.md` §4 ;
  `NOTE_5_14_CLASSES_DE_VUE` §3).
- (b) Faux en-tête dans le corps d'un record authentique d'un autre slot. Indice : 8 des 10 instants V1 examinés
  tombent à ± 15 images de la fin d'une vie de joueur.
- (c) Record de création mal cadré. Le balayage des naissances accepte aussi des ancres fausses :
  - générations nulles, impossibles (`774 gen=0`, `832 gen=0`, `823 gen=0` sur 879a4dba) ;
  - une « création » au motif A (`789 gen=2` en (-201.16, 88.26, -390.99), châssis `0x2e8c8128` inconnu).

**Sonde qui tranche** (non lancée, coût : un décodage de 81c02726, 17 Mo de chunks, < 2 min, sous verrou) :
- Pour chaque ancre acceptée, noter son bit, son paquet, son tag et les 2 bits « fini ».
- Pour les paquets des instants 645,4 / 812,7 / 983,6 / 983,9, et pour un témoin d'ancres normales :
  - relever le départ de boucle (`marchStartOf`) et les records de la marche (`DecodeFrameRecords`) ;
  - classer chaque ancre : début d'un record marché du même slot, intérieur d'un autre record, liste d'événements,
    vue C, ou au-delà d'une désynchronisation ;
  - tester `TryDeltaAt` à l'ancre, avec un record suivant valide à sa fin (`frame_records.go:223`).

Ghidra n'est pas nécessaire.

---

## 3. Historique — ce qui a été vu, et pourquoi ça n'a pas suffi

- **2026-09-08, bornes aberrantes** (`thought_log.md:5868-5917`).
  - Le point exact de Madina (81c02726, slot 523, image 645, z=-325.4) est qualifié d'« artefact de décodage ».
  - Le rejet par centiles n'a été appliqué qu'au **cadre** (`boundsOf`) : le point reste publié et le client le dessine.
- **2026-09-12, lot B-bis** (`.ai/V7.5/RAPPORT_LOT_BBIS_BIT_PROJECTILE_2026-09-12.md:259-268`).
  - Sur les cartes Forge, 612 pas impossibles d'objets du monde : « la signature d'un **faux positif du balayage par
    position de bit** ».
  - Consigné (« à instruire par la sélectivité du balayage »), non traité.
- **2026-09-14, lot 1.6.5** (`tracks_publication.go:21-40`).
  - `DefaultMinPoints` passe de 2 à 1 sur décision utilisateur : « si le film le dit, on publie ».
  - Or 22 des 51 vies d'un point sont nos faux positifs hors garde, et 20 vies (1-2 points) précèdent toute création.
    Ce n'est pas le film qui le dit, c'est le balayage.
- **Lot 1.9.13.** `Point.g` est publié « pour que le client ne trace aucun segment au travers », mais aucun lecteur
  n'a été câblé côté web.
  - Mesuré : 974 lacunes dans 97 documents, 21 472 s au total.
  - 222 d'entre elles déplacent le pion de plus de 10 m à travers la carte (6 807 s).
- **Filtres hors ligne**, calibrés sur 000d5950 : « aberrations séparées de 66 à 320 s » (`offline_filters.go:13-24`).
  Les nôtres ont un vrai voisin à moins de 15 s, et le premier point d'un slot est accepté d'office.
- **`vehicleSpawnsByLife`** retient le record de création le PLUS PRÉCOCE (`vehicle_tracks.go:199-212`). Une fausse
  création antérieure l'emporte donc sur la vraie, et donne du même coup une famille de châssis inconnue.
  - Flood Gulch : `906`, `999`, `908`, `960` sont dessinés en losange neutre à leur faux spawn, jusqu'à 621 s.

---

## 4. Solution proposée

### Option 1 (RECOMMANDÉE) — une publication qui applique la grammaire de la vie, des replis nommés, et un client qui respecte les silences

Tout se fait à l'**assemblage** (`BuildFromPositions`), donc la republication part des faits persistés : verdict
« republier », **aucun décodage de film**.

**Portes grammaticales (bipèdes)** — dans `replay/`, avant le calcul de `a.origin` (`build.go:141-147`), en s'appuyant
sur `a.opt.BipedCreations` :
- **R-B1 : une vie ouverte par un record de création commence à ce record.**
  - Les points de la vie antérieurs au record sont écartés et comptés.
  - Règle E2 mesurée : « aucun record de création dans l'intervalle d'une vie ».
  - Effet : sur 81c02726 la vie de Madina démarre au vrai spawn (t=692) ; sur ab526724 sa vie d'un point (t=269) relève de R-B2. Probablement les 9 cas T1 (vérifié 1 sur 1
    sur les faits, les 8 autres à mesurer).
- **R-B2 : aucune vie avant la première création de son slot**, quand le slot en porte au moins une.
  - Parc : 20 vies, toutes de 1 ou 2 points. Aucune vraie vie ne serait perdue (négatif mesuré).
  - Emplacement : `identity_registry_creation.go` / `lives_decoupe.go`, avec un compteur de couverture.

**Replis nommés et comptés** (registre `facts/fallback`, date + critère de retrait = l'option 2 livrée et mesurée à 0) :
- **F-1 : position hors de l'emprise jouée du film.**
  - Même règle et même constante que `boundsOf` (`boundsRejectSpreads = 12`, calibrée sur le parc : écart
    artefacts ≥ 17,7 contre légitime ≤ 9,5).
  - Appliquée aux points de trace, aux échantillons de véhicule et aux `spawn`.
  - Troisième usage de la garde : **centraliser `guardOf/axisGuard`** et poser un garde-rail (test grep qui interdit
    une réécriture de la constante).
  - Couvre 51 des 56 cas et les 11 naissances.
- **F-2 : échantillon de véhicule atteint ou quitté à travers un silence de plus de `lifeGapUS` avec un déplacement.**
  - Mesuré : sur 565 silences de plus de 5 s, les 21 avec un déplacement sont tous aberrants ; les 544 autres n'ont aucun déplacement.
  - Couvre le cas V1 dans la garde (Behemoth 798).
  - Une vie de véhicule sans position restante n'est plus publiée (`NoPosition++`), et son `spawn` hors emprise non plus.

**Client** (`apps/web`) :
- **C-1** : `positionAt`, `trailAt` et `altitudeAt` respectent `Point.g` : ni position interpolée ni segment au
  travers d'une lacune (fichier `lib/replay/replayLogic.ts`). Le rendu pendant la lacune est la question Q1.
- **C-2** : publier `g` aussi sur `VehicleSample` (même sémantique, côté Go `vehicle_tracks.go:397-427`), et que
  `vehiclePositionAt` **maintienne** la dernière position au travers au lieu d'interpoler. Sur les données saines le
  rendu ne change pas (0 déplacement réel mesuré pendant un silence).
  - Un résidu non filtré ne serait plus visible que pendant ses propres images, au lieu de 70 à 419 s de dérive.
  - Fichier : `model/vehiclesLayer.ts:407-411`.

**Montée de schéma : OUI**, 68 → 69, avec l'entrée de chronique :
- nouveaux compteurs : `coverage.tracks.{avantCreation, viesAvantPremiereCreation, horsEmprise}` et
  `coverage.vehicles.{echantillonsHorsEmprise, echantillonsAuTraversDUnSilence, spawnsHorsEmprise}` ;
- nouveau champ `VehicleSample.g` ;
- régénération `make generate-types`.

**Republication : OUI**, des 111 artefacts depuis les faits (sans décodage) — **décision utilisateur**, serveur arrêté.

**Tests (rouges avant, verts après)** :
- Go :
  - vie dont le premier point précède sa création → démarre à la création, compteur à 1 ;
  - vie entièrement avant la première création → écartée ;
  - point, échantillon et spawn hors emprise → écartés et comptés ;
  - aller-retour au travers d'un silence → écarté ;
  - témoins négatifs : chute continue au-delà du cadre (Behemoth 771, t=3371-3384) conservée ; véhicule garé
    100 s conservé.
- Régression sur faits réels (test `research`, ≤ 3 fichiers) :
  - 81c02726 : 770 n'a plus d'échantillon à t=983, Madina démarre à t=692 ;
  - ab526724 : 769/770 non publiés ;
  - 879a4dba : excursions 770 et 774 retirées.
- Web (vitest) : `positionAt` et `trailAt` au travers d'un point `g`, maintien véhicule au travers de `g`.

**Gate sur documents réels** (après republication, `census.mjs` + `bits.mjs` relancés) :
- T1 à T4 hors garde = 0, V1 = V2 = V3 = 0 ;
- paires de motifs transverses ≥ 24 bits = 0 ;
- chutes réelles de Behemoth présentes à l'identique ;
- points retirés = somme des nouveaux compteurs de couverture (attendu ≈ 50 points de trace, dont les 20 vies de 1-2 points,
  ≈ 30 échantillons de véhicule, 11 naissances) ;
- plus aucun échantillon de véhicule déplacé pendant un silence de plus de 5 s (21 → 0).

**Risques** :
- L'origine du cadre temporel est le premier point trié : filtrer AVANT `ouvrir`, sinon `originMs` bouge sur tout le
  document.
- R-B1/R-B2 dépendent du balayage des créations bipèdes. Sa signature de 43 bits a donné 0 lecture sur le témoin
  fantôme (E2). Une création manquée désarme la règle, dans le sens prudent.
- F-1 est une heuristique mesurée, pas une grammaire : elle peut écarter une position réelle sur une carte à zone jouée
  très étendue et peu échantillonnée. Il faut compter ce qu'elle écarte, jamais le taire.
- Diffs de goldens attendus : identité (vies propagées), couverture, `frameCount` éventuel.

**Taille : M.**

### Option 2 — porte grammaticale au décodage

- Valider chaque ancre de position par la grammaire du record entier : `TryDeltaAt` à l'ancre, plus un record suivant
  valide à sa fin, génération cohérente avec le monde (`World.GenerationMatches`), refus de `gen=0`.
- Pour les créations de véhicule : cohérence du mot de châssis entre les records d'une même vie.
- Doctrinalement plus pure, et c'est elle qui permettrait de retirer F-1 et F-2. Mais :
  - il faut d'abord la sonde du §2 (1 film) ;
  - les composants `ti=40` i30-i42 ne sont pas portés, donc la fin des records n'est pas calculable pour certains ;
  - montée de `GrammarRev`, donc verdict « redécoder » sur tout le parc (lourd, RAM).
- **Taille : L.**
- Recommandation : lancer la sonde en lot de mesure après l'option 1, puis décider.

---

## 5. Questions pour l'utilisateur (une ligne chacune)

1. **Pendant un silence de réplication d'un joueur** (lacune `g`, 974 dans le parc), le pion doit-il rester à sa dernière
   position en pâlissant (proposé), ou disparaître jusqu'au point suivant ?
2. **« Si le film le dit, on publie » (2026-09-14)** : 20 vies de 1-2 points précèdent la création du corps et 22 vies
   d'un point sont hors de la carte (flèches de mort fantômes). Ce n'est pas le film qui les dit mais le balayage :
   d'accord pour ne plus les publier (comptées en couverture) ?
3. **Mécanique** : en multijoueur Halo Infinite, un véhicule inoccupé peut-il changer de place sans nouvelle vie
   (script Forge, téléporteur) ? Si c'est « jamais », la règle F-2 devient une règle de jeu et non un repli.
4. **Republication** des 111 artefacts depuis les faits (schéma 69, sans décodage) : feu vert, serveur arrêté ?

---

## 6. Hors périmètre découvert (noté, non traité)

- **Starboard** (ab526724, f0220a96) : 7 véhicules RÉELS (Scorpion, Wasps, Warthogs, châssis connus) sont recensés
  vivants tout le match à y≈-130, hors de l'arène d'un CTF sans véhicule, et affichés 10 min. Il faut une décision de
  rendu : afficher ou non les véhicules réels hors de la zone jouée.
- Le balayage des créations de véhicule accepte des handles de génération 0 (impossibles) et des mots de châssis
  inconnus à des positions hors carte (879a4dba : 7 cas).
- Châssis inconnus récurrents : `1a043c29`, `f4c45d71`, `233c877d`, `001b33fc`, `dd7f9102`, `bcfb852f`, `b857fb95`.
  Ils vont par paires au même point (Flood Gulch, Fortitude) et sont affichés en losange neutre : table
  `vehicleFamilyByChassis` à compléter.
- Même famille de faux positifs sur le balayage des objets du monde (projectiles Forge) : lot B-bis, non traité.
- Les 222 lacunes de bipède avec déplacement de plus de 10 m (6 807 s) sont couvertes par C-1. Il reste à mesurer
  combien tombent pendant un trajet en véhicule que le prédicat « embarqué » ne couvre pas.
