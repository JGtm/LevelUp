# Lot C — phase 0 (RECENSER) : journal des mesures, items C.0.1 et C.0.3

> Plan maitre : `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`, section « Lot C ».
> Perimetre de cette phase : C.0.1 et C.0.3 SEULEMENT. C.0.2 (lecture des 32 drapeaux de
> `boundary-visibility` par le hook `SetManagedObjectHook`) attend la fusion du lot 0 et n'est
> PAS traite ici. Lecture seule : aucun code de production modifie, aucune ecriture DuckDB,
> aucune retro-ingenierie, aucun composant non porte decode.
>
> Worktree `LevelUp-wt-zones-film`, branche `wt/zones-film`, base `3fdbb8030`.
> Mesures du 2026-08-17. Verdicts des gates : `LOTC_gates.log`. Sorties : `lotC/*.tsv`.

## 1. L'instrument

| fichier | role |
|---|---|
| `apps/go-api/internal/analysis/filmdec/zone_census_test.go` | garde d'environnement, recensement des tables d'image-cle, construction des bandes de slots et des trois temoins |
| `apps/go-api/internal/analysis/filmdec/zone_census_scan_test.go` | le balayage des paquets delta (une seule passe), l'horloge de match, les fenetres, le plancher de bruit |
| `apps/go-api/internal/analysis/filmdec/zone_census_report_test.go` | entree du test, oracle des evenements nommes, journalisation, ecriture des TSV |

Garde : `ZONE_FILM` (repertoire des chunks d'UN film, chemin absolu) + `ZONE_OBJTYPE`
(`zone` | `flag` | `none`). Sans `ZONE_FILM` le test SKIP proprement (verifie, `EXIT_TEST_SKIP=0`).

Trois choix de methode, tous imposes par des faits du depot :

1. **Ancrage par bande de slots** (`matchWorldObjectRecord` + slots lus dans les tables
   d'image-cle), et non traversee canonique. La traversee s'arrete au premier composant
   PRESENT non porte ; ti=10 n'a qu'un composant porte sur 30 et ti=12 un sur 28, donc la
   voie canonique perdrait la suite de chaque paquet des le premier record un peu bavard.
2. **Une seule passe pour toutes les bandes.** Une position de bit ne rend qu'une valeur de
   slot, donc au plus une bande : une table `slot -> classe` rend exactement la meme mesure
   que sept balayages, pour un septieme du cout (D17 — la machine de l'utilisateur paie).
3. **Bande OBSERVEE et non comblee.** `worldObjectSlotBand` comble la plage parce qu'un
   projectile vit moins d'une seconde et n'apparait presque jamais en image-cle. Un objet de
   mode est l'exact contraire : il vit toute la partie, donc il est present a CHAQUE
   image-cle et l'observe est deja complet (meme raisonnement que le lot R4 pour ti=11).

## 2. Le temoin fantome — ce que l'ancrage reconnait vraiment

**C'est le resultat le plus important de la phase, et il conditionne la lecture de tout le
reste.** L'en-tete « objet du monde » ne contraint que 21 bits, dont la bande de slots ne
fournit qu'une poignee. Sur plusieurs centaines de millions de positions de bit par film, il
tombe juste PAR HASARD un tres grand nombre de fois. Trois temoins de meme cardinalite que la
bande reelle le mesurent :

| temoin | definition | 7344d24f (Strongholds) | 000d5950 (Slayer) |
|---|---|---|---|
| VIDE | slots du haut de l'espace, jamais vus en image-cle | 26 899 records · 203,8/slot | 7 498 records · 374,9/slot |
| INCONNU | slots absents de toute image-cle (portent aussi les entites fugaces) | 23 725 records · 179,7/slot | 9 707 records · 485,4/slot |
| OCCUPE | slots vus porter un archetype hors perimetre (temoin POSITIF) | 208 172 records · 1 577,1/slot | 33 093 records · 1 654,7/slot |

Le temoin POSITIF rend beaucoup de records : le balayeur voit bien les vrais records. Mais le
temoin VIDE n'est PAS a zero — il rend 58 a 375 records par slot selon le film. **Le nombre
de records d'une bande n'est donc pas une mesure exploitable en soi** : sur `64e8adfa`, ti=10
rend 271,8 records/slot contre 224,4 pour le temoin VIDE (1,2x) ; sur `24dbb67d`, ti=12 rend
88,4/slot contre 186,7 (0,5x, SOUS le bruit).

**La grandeur qui separe le signal du bruit est le PLANCHER DE BRUIT PAR INDEX** : la mediane
des annonces sur les 64 index de composant possibles, calculee DANS la bande elle-meme. Un
tirage au hasard repartit l'index de composant a peu pres uniformement sur 0..63 ; un
composant reellement annonce s'en detache d'un facteur. Seuil ecrit avant la mesure :
**exces >= 3x le plancher**. Second controle, independant : la part de records dont le masque
porte un index HORS de la grammaire de l'archetype (un record de ti=10 ne peut pas annoncer
i40) — elle mesure directement la contamination de la bande.

| film | ti=10 hors grammaire | ti=12 hors grammaire | ti=47 hors grammaire |
|---|---|---|---|
| 7344d24f | 22,56 % | 13,98 % | 4,71 % |
| 696a9d7c | 17,90 % | **3,40 %** | 6,40 % |
| 64e8adfa | 37,27 % | 30,23 % | 15,17 % |
| 53ce4390 | 38,81 % | 35,38 % | 23,39 % |
| 000d5950 (Slayer) | — (bande vide) | 61,11 % | 85,23 % |

## 3. C.0.1 (1) — slots par archetype dans le World d'image-cle

`WalkKeyframeWorld` sur tous les paquets d'image-cle. Denominateur : records d'image-cle du
film. TSV : `lotC/<short8>_kf_slots.tsv` (avec la liste complete des slots).

| film | mode | chunks | tables KF | records KF | ti=10 | ti=12 | ti=23 | ti=11 | ti=13 | ti=47 | ti=4 |
|---|---|---|---|---|---|---|---|---|---|---|---|
| 7344d24f | Strongholds | 32 | 31 | 14 014 | 81 | 16 | **0** | **0** | 26 | 8 | 1 |
| 696a9d7c | Strongholds | 30 | 29 | 12 979 | 77 | 10 | **0** | **0** | 26 | 8 | 1 |
| 0a247154 | KOTH | 41 | 40 | 18 411 | 54 | 58 | **0** | 1 | 20 | 8 | 1 |
| 01e1f945 | KOTH | 29 | 28 | 8 591 | 57 | 15 | **0** | 1 | 20 | 9 | 1 |
| 606d9844 | KOTH | 14 | 13 | 4 171 | 30 | 10 | **0** | 1 | 20 | 9 | 1 |
| 8076f97f | KOTH | 20 | 19 | 7 082 | 38 | 7 | **0** | 1 | 20 | 10 | 1 |
| 64e8adfa | CTF | 44 | 43 | 13 654 | 157 | 34 | **0** | 10 | 30 | 8 | 2 |
| 530820e5 | CTF | 26 | 25 | 7 750 | 77 | 19 | **0** | 5 | 19 | 8 | 1 |
| 53ce4390 | CTF | 40 | 39 | 14 587 | 118 | 16 | **0** | 5 | 19 | 8 | 1 |
| 24dbb67d | Oddball | 28 | 27 | 8 396 | 48 | 17 | **0** | 10 | 24 | 8 | 2 |
| 000d5950 | Slayer (temoin) | 27 | 26 | 7 825 | **0** | 3 | **0** | **0** | 8 | 8 | 1 |
| **total** | | **331** | **320** | **117 460** | | | **0** | | | | |

(Les colonnes donnent le nombre de SLOTS DISTINCTS ; les records d'image-cle par archetype
sont dans les TSV. La bande retenue est identique au nombre de slots observes sur tous les
films sauf `0a247154` ti=12, ou 1 slot sur 58 est ecarte comme ambigu.)

**Le temoin Slayer fait son travail** : ti=10 y est ABSENT (0 slot), alors qu'il est present
sur les dix films a objectif. ti=12, ti=13, ti=47 et ti=4 existent aussi en Slayer — ce ne
sont donc pas des archetypes propres aux modes a objectif.

### Stabilite des numeros de slot entre films

Recouvrement des NUMEROS de slot, mesure par paires :

| paire | ti=10 | ti=12 | ti=13 | ti=47 | ti=4 |
|---|---|---|---|---|---|
| 7344d24f / 696a9d7c (Strongholds) | 10 / 81 | 5 / 16 | **26 / 26** | **8 / 8** | **1 / 1** |
| 0a247154 / 01e1f945 (KOTH) | 2 / 54 | 0 / 58 | 0 / 20 | 1 / 8 | **1 / 1** |
| 606d9844 / 8076f97f (KOTH) | 1 / 30 | 0 / 10 | 0 / 20 | 0 / 9 | **1 / 1** |
| 64e8adfa / 530820e5 (CTF) | 9 / 157 | 7 / 34 | 18 / 30 | **8 / 8** | **1 / 1** |
| 64e8adfa / 53ce4390 (CTF) | 16 / 157 | 0 / 34 | 0 / 30 | 0 / 8 | **1 / 1** |

Lecture : **les slots BAS, alloues tot, sont stables ; les slots hauts derivent.** ti=4 tient
le slot 123 sur les onze films. ti=47 tient 1298..1312 (pairs) sur les deux Strongholds et
sur deux des trois CTF. ti=13 tient 1520..1547 sur les deux Strongholds. Pour ti=10 et ti=12
le recouvrement global est faible, mais les PREMIERS slots sont identiques : les deux
Strongholds partagent 1544, 1549, 1613, 1620, 1621, 1622, 1627 (ti=10) et 1543, 1548, 1623,
1624, 1625 (ti=12). **Consequence pour la phase 1 : aucun numero de slot ne doit etre cable.**
La bande doit etre relue dans les images-cle du film, comme l'instrument le fait.

## 4. C.0.1 (2) — records d'IMAGE-CLE par le walker deterministe

`WalkKeyframeRecords` sur les memes payloads. Resultat identique sur les onze films :
**il parse UN SEUL record par table, puis s'arrete sur `en-tete-invalide`.**

| film | tables d'image-cle | records parses | cause d'arret |
|---|---|---|---|
| tous (11 films) | 8 a 44 selon le film | 1 par table | `en-tete-invalide` (100 %) |

Ce n'est pas une surprise, c'est une CONFIRMATION de ce que le depot avait deja mesure et
ecrit : `keyframe_record_walk.go:259-273` etablit qu'aucun decalage ne rend une marche
bit-exacte et que « le corps d'un record de la table d'image-cle n'est PAS le corps d'un
record NEW ». Le walker deterministe ne peut donc pas servir de recensement par archetype :
**le recensement d'image-cle exploitable est celui de la section 3** (`WalkKeyframeWorld`, qui
balaie les en-tetes au lieu de parser les corps). Item statue en consequence.

## 5. C.0.1 (3) — records DELTA et annonces au masque

Une passe par film sur tous les paquets delta. TSV par film :
`lotC/<short8>_delta_masques.tsv` (ti, i, composant, niveau, statut, annonces, % des records
de l'archetype, plancher de bruit, exces, et les trois temoins par index).

| film | paquets delta | ti=10 records | ti=12 records | ti=47 records | ti=11 records | ti=23 records |
|---|---|---|---|---|---|---|
| 7344d24f | 36 350 | 38 525 | 22 471 | 16 591 | **0** | **0** |
| 696a9d7c | 34 276 | 35 121 | 18 663 | 14 946 | **0** | **0** |
| 0a247154 | 47 854 | 16 412 | 9 410 | 11 274 | 127 | **0** |
| 01e1f945 | 33 102 | 26 707 | 8 811 | 8 004 | 73 | **0** |
| 606d9844 | 14 791 | 4 998 | 2 331 | 3 253 | 23 | **0** |
| 8076f97f | 21 495 | 8 953 | 4 199 | 5 556 | 83 | **0** |
| 64e8adfa | 50 956 | 42 666 | 15 476 | 14 308 | 4 160 | **0** |
| 530820e5 | 29 148 | 14 149 | 4 100 | 6 967 | 677 | **0** |
| 53ce4390 | 45 856 | 46 754 | 5 070 | 10 719 | 871 | **0** |
| 24dbb67d | 31 834 | 8 748 | 1 503 | 6 073 | 612 | **0** |
| 000d5950 | 30 418 | **0** | 342 | 2 410 | **0** | **0** |
| **total** | **376 080** | | | | | **0** |

Ces comptes de records incluent le bruit d'ancrage (section 2) : ils situent l'ordre de
grandeur, ils ne prouvent rien a eux seuls. L'ordre de grandeur concorde avec le releve LIVE
du 31/07 (`RELEVE_TERRAIN_CAPTURES_2026-07-31.md:68-70`), qui comptait 26 006 dispatches
ti=10 et 18 903 ti=12 sur `7344d24f` la ou l'ancrage en compte 38 525 et 22 471 — l'ecart est
du meme ordre que le taux hors grammaire (22,6 % et 14,0 %).

**Forme du masque** : les records de ces archetypes sont COURTS. Sur `696a9d7c`, ti=12 :
17 905 records a 1 composant sur 18 663 (95,9 %) ; ti=10 : 29 542 sur 35 121 (84,1 %). L'en-tete
« objet du monde » (branche eparse, selecteur de base nul) leur convient donc bien : c'est ce
que la faible part hors grammaire de ti=12 sur ce film (3,40 %) confirme le plus nettement.

### Top 15 des composants NON PORTES qui se detachent du bruit

Cumul sur les onze films, avec le maximum par film et l'exces maximal observe
(table complete triee : `lotC/LOTC_table_C03.tsv`).

| # | ti | i | composant | annonces (cumul 11 films) | films | max / film | exces max |
|---|---|---|---|---|---|---|---|
| 1 | 10 | 26 | `managed-object-rtpc-component` | 67 765 | 10 | 13 964 (53ce4390) | 113,8x |
| 2 | 12 | 14 | `managed-navpoint-radial-progress` | 52 387 | 11 | 17 369 (696a9d7c) | **868,5x** |
| 3 | 10 | 27 | `managed-object-rtpc-component` | 49 082 | 10 | 18 395 (696a9d7c) | 98,1x |
| 4 | 47 | 2 | `personal-ai-data-component` | 47 906 | 11 | 13 361 (7344d24f) | **1 214,6x** |
| 5 | 13 | 1 | `managed-object-property-component` | 16 607 | 11 | 4 086 (696a9d7c) | 88,9x |
| 6 | 10 | 1 | `managed-object-boundary-color-component` | 10 347 | 10 | 3 261 (64e8adfa) | 6,5x |
| 7 | 10 | 4 | `managed-object-navpoint-component` | 6 817 | 10 | 1 711 | 7,7x |
| 8 | 12 | 1 | `managed-navpoint-flags-component` | 6 086 | 11 | 3 613 (64e8adfa) | 21,8x |
| 9 | 10 | 2 | `managed-object-navpoint-component` | 5 682 | 10 | 2 302 | 3,9x |
| 10 | 13 | 13 | `managed-object-player-masked-property-component` | 5 236 | 11 | 3 347 | 37,4x |
| 11 | 13 | 21 | `managed-object-player-masked-property-component` | 5 094 | 11 | 3 354 | 37,5x |
| 12 | 10 | 17 | `managed-object-navpoint-component` | 4 891 | 10 | 2 369 | 16,0x |
| 13 | 10 | 24 | `managed-object-looping-sound-component` | 4 789 | 10 | 938 | 3,8x |
| 14 | 10 | 8 | `managed-object-navpoint-component` | 4 529 | 10 | 1 360 | 2,6x |
| 15 | 10 | 7 | `managed-object-navpoint-component` | 4 217 | 10 | 1 655 | 11,0x |

**Ce que ce classement dit.** Quatre canaux se detachent d'un ou deux ordres de grandeur au-dessus
du bruit, et ils portent chacun l'essentiel des records de leur archetype :
`ti=12 i14 radial-progress` (74,7 a 93,1 % des records ti=12 sur les Strongholds et les KOTH),
`ti=10 i26`/`i27 rtpc` (17 a 53 %), `ti=47 i2 personal-ai-data` (74 a 81 % sur les modes a
zones et KOTH). **Les candidats attendus par le plan se comportent tres differemment :**
`ti=12 i13 top-progress` et `i15 bottom-progress` restent au plancher (exces 1,1x sur
7344d24f, 134 et 135 annonces) ; `ti=12 i10 timers` 1,7x ; `ti=12 i9 formatted-text` 1,7x ;
`ti=10 i22 interaction-filter` 0,9x ; `ti=10 i23 flags` 0,6x ; seul `ti=10 i1 boundary-color`
depasse legerement, et sur les dix films ou il existe (3,8x et 3,7x sur les deux Strongholds ;
maximum 6,5x sur `0a247154`, minimum 1,9x sur `606d9844`). **Sur les trois progressions de
ti=12, une seule parle : la RADIALE.**

(Dans la table ci-dessus, « max / film » et « exces max » peuvent tomber sur des films
differents : le volume d'annonces suit la taille du film, l'exces suit la purete de la bande.)

## 6. Verdict ti=23 (`selectable-zone-data`)

**ti=23 est ABSENT, en image-cle ET en delta, sur les ONZE films du corpus.**

- image-cle : 0 record et 0 slot sur 11 films, denominateur cumule **117 460 records
  d'image-cle** repartis sur **320 tables d'image-cle** (detail par film en section 3) ;
- delta : 0 record reconnu sur 11 films — mecaniquement, puisque aucun slot n'a jamais porte
  ti=23 dans une image-cle, il n'existe aucune bande a balayer. Le verdict delta est donc une
  CONSEQUENCE du verdict image-cle, et non une mesure independante : c'est dit ici pour qu'on
  ne le lise pas comme deux preuves ;
- denominateur delta parcouru : **376 080 paquets delta**, tous modes confondus.

Cela ETEND a 11 films (5 modes, dont Strongholds x2, KOTH x4, CTF x3, Oddball, Slayer) le
constat que le releve du 31/07 avait pose sur 2 films. L'ecart n°2 du plan (§1.3) est
confirme : **le lot C n'a rien a attendre de ti=23**, et son deserialiseur
(`components_world.go:108`, largeurs figees 6/6/6, aucun appelant) reste sans emploi.

## 7. C.0.3 — verdict ti=47 (`splash-message`) et densite autour des captures

L'oracle est `objectiveevents.NamedEvents(src, objectiveType)` sur la source disque canonique
(`filmcache.Open`). L'horloge des annonces est CELLE de `StatRecords` :
`tMS = manifeste.start_ms[chunk] + (us_du_paquet - us_du_premier_paquet_delta_du_chunk)/1000`.

**Deux precautions prises AVANT de lire les chiffres, toutes deux imposees par des mesures :**

1. **La famille d'objectif est FOURNIE, jamais devinee** (`ZONE_OBJTYPE`). La table `flag`
   appliquee au film KOTH `606d9844` rend 267 « evenements de drapeau » dont 199
   `flag_returns` — sur un match qui n'a jamais vu un drapeau. `namedStatSlots` n'a de table
   que pour `zone` et `flag` (named.go:83-105) : **KOTH, Oddball et Slayer n'ont AUCUN oracle
   nomme**, et la densite y est declaree non mesurable (5 films sur 11).
2. **Les fenetres ne retiennent que les evenements d'OBJECTIF.** La meme table porte aussi
   `kills` et `assists` : sur `7344d24f` ils ajoutent 175 evenements aux 71 evenements de zone,
   et la fenetre +/- 3 s passait alors a 462,9 s sur 608,2 s de match (76 %) — une comparaison
   de densite vide de sens. Le gate parle des « captures » : les evenements de combat sont
   ecartes des fenetres et le nombre ecarte est publie.

Denominateurs retenus (fenetre du gate, +/- 3 s) :

| film | mode | evenements d'objectif retenus | combat ecartes | duree de match | secondes en fenetre | secondes hors |
|---|---|---|---|---|---|---|
| 7344d24f | zone | 71 (59 captures + 12 securisations) | 175 (117 frags + 58 assist.) | 608,2 s | 249,0 | 359,2 |
| 696a9d7c | zone | 77 (61 captures + 16 securisations) | 147 (102 + 45) | 573,2 s | 251,8 | 321,5 |
| 64e8adfa | flag | 116 (65 prises, 17 vols, 13 porteurs tues, 10 retours, 7 assist., 4 captures) | 150 (109 + 41) | 851,0 s | 350,0 | 500,9 |
| 530820e5 | flag | 56 (23, 10, 10, 7, 3, 3) | 127 (92 + 35) | 486,8 s | 215,7 | 271,1 |
| 53ce4390 | flag | 49 (21, 13, 10, 3, 1, 1) | 150 (115 + 35) | 765,6 s | 218,2 | 547,4 |

### ti=47 : ses deux composants PORTES ne sont pas les messages « zone capturee »

| film | mode | i0 static (porte) | i1 dynamic (porte) | i2 personal-ai (non porte) |
|---|---|---|---|---|
| 7344d24f | zone | 84 ann · 0,76x | 2 482 ann · **0,01x** | 13 361 ann · 1,61x |
| 696a9d7c | zone | 86 ann · 0,92x | 2 460 ann · **0,01x** | 11 543 ann · 1,60x |
| 64e8adfa | flag | 119 ann · 1,19x | 12 068 ann · 1,97x | 60 ann · 0,52x |
| 530820e5 | flag | 74 ann · 2,79x | 5 410 ann · **3,27x** | 38 ann · 0,51x |
| 53ce4390 | flag | 188 ann · 2,30x | 8 065 ann · 2,70x | 184 ann · 0,24x |

(« ann » = annonces sur le film ; le facteur est densite en fenetre / densite hors fenetre.)

**Verdict.** Le comportement de ti=47 est MODE-DEPENDANT, et de facon tranchee :

- **En modes a zones, le splash dynamique porte (i1) est ANTI-correle aux captures : 0,01x**
  — 12 annonces en fenetre contre 2 470 hors fenetre sur `7344d24f`. Ce n'est pas du bruit
  (l'exces sur plancher est de 225,6x, le canal est massivement reel) : c'est un canal reel
  dont les emissions tombent presque exclusivement AILLEURS que pres des captures. Reponse a
  la question de C.0.3 : **non, les deux composants portes de ti=47 ne sont pas les messages
  « zone capturee »**.
- **En CTF, i1 SUIT les evenements de drapeau** : 1,97x / 3,27x / 2,70x sur les trois films.
  Le canal existe donc bien comme message d'evenement d'objectif, mais pour la famille
  drapeau, pas pour les zones.
- **Le canal dominant de ti=47 en modes a zones est le composant NON PORTE i2
  `personal-ai-data-component`** (77 a 81 % des records de l'archetype, exces jusqu'a
  1 214,6x), a 1,60-1,61x autour des captures. Il est quasi absent en CTF (0,4 a 1,7 %).
- Reserve honnete : la bande ti=47 du temoin Slayer porte 85,23 % de records hors grammaire,
  donc l'ancrage y est tres contamine. Sur les films a objectif il tient beaucoup mieux
  (4,7 a 23,4 %), mais les chiffres ti=47 doivent etre lus avec cette marge.

## 8. GATE 0 — verdict

Enonce du plan, repris mot pour mot et **non modifie** : « au moins un composant NON PORTE de
ti=10 ou ti=12 est annonce >= 100 fois par film Strongholds ET ses annonces se concentrent
autour des captures (fenetre +/- 3 s : densite >= 3x la densite hors fenetre) ».

**Clause 1 — TENUE, tres largement.**

| film Strongholds | composants non portes de ti=10/ti=12 a >= 100 annonces | les trois premiers |
|---|---|---|
| 7344d24f | **53** | ti=12 i14 : 16 788 · ti=10 i27 : 16 755 · ti=10 i26 : 6 522 |
| 696a9d7c | **34** | ti=10 i27 : 18 395 · ti=12 i14 : 17 369 · ti=10 i26 : 6 948 |

**Clause 2 — NON TENUE.** Aucun composant ne reunit les deux clauses sur les deux films.

| composant | 7344d24f | 696a9d7c |
|---|---|---|
| ti=12 i14 `radial-progress` | 16 788 ann · **2,35x** (serre 1 s : 1,75x) | 17 369 ann · **2,18x** (1,68x) |
| ti=10 i27 `rtpc` | 16 755 ann · **2,37x** (1,73x) | 18 395 ann · **2,10x** (1,65x) |
| ti=10 i26 `rtpc` | 6 522 ann · 1,25x | 6 948 ann · 1,18x |
| ti=10 i1 `boundary-color` | 1 194 ann · 1,13x | 697 ann · 1,37x |
| ti=12 i22 `visual-state-groups-2` | 360 ann · 1,80x | 284 ann · **3,20x** (serre 1 s : 3,58x) |
| ti=12 i21 `visual-state-groups-1` | 247 ann · 1,92x | 198 ann · 2,61x (3,81x) |
| ti=10 i22 `interaction-filter` | 272 ann · 1,38x | 116 ann · 0,97x |
| ti=10 i23 `flags` | 188 ann · 1,32x | 102 ann · 1,68x |

Le meilleur des canaux a fort volume plafonne a **2,37x** ; le seul composant a franchir 3x
(`ti=12 i22`, 3,20x sur `696a9d7c`) retombe a 1,80x sur l'autre film Strongholds. Le
diagnostic a fenetre serree (+/- 1 s, ecrit comme diagnostic et ne jugeant rien) ne renverse
rien pour les canaux a fort volume : il les fait BAISSER (1,73x et 1,75x), ce qui exclut que
le negatif vienne d'une fenetre trop large.

> **GATE 0 : NON TENU.** Seuil non rebaisse, non contourne, non reinterprete.

**Ce que le negatif dit — et ce qu'il ne dit pas.** L'enonce de repli prevu par le plan
(« les zones ne parlent pas en delta par ces archetypes ») serait FAUX tel quel, et le dire
importe plus que de cocher une case : `ti=12 i14` porte 74,7 % et 93,1 % de tous les records
de ti=12 sur les deux films Strongholds, a 140,5x et 868,5x le plancher de bruit. **Le canal
existe, il est massif, il est mesure.** Ce qui echoue, c'est le CRITERE DE CONCENTRATION
TEMPORELLE — et il y a une raison mecanique de s'y attendre : `radial-progress` est une
PROGRESSION, une valeur continue reemise pendant tout le remplissage d'une zone (plusieurs
dizaines de secondes), pas un marqueur d'instant. Un canal continu ne peut pas etre 3x plus
dense dans +/- 3 s autour de l'instant de capture ; le test tel qu'ecrit mesure la
concentration d'un EVENEMENT, alors que les deux canaux dominants sont des ETATS.

Consequence par le contrat (`plan-execution`, regle 9) : **le lot s'arrete a la fin de la
phase 0** et la phase 2 du plan item 4 (`PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`) garde sa
forme actuelle. La suite est un ARBITRAGE DU SUPERVISEUR, pas une decision d'executeur : soit
le negatif est acte tel quel, soit le gate 0 est reformule pour un canal d'etat (par exemple :
« la valeur du canal change de facon monotone dans une fenetre encadrant chaque capture »,
qui exigerait de LIRE la valeur, donc la phase 1). Les deux options sont ecrites ; aucune
n'est appliquee ici.

## 9. Cout machine (D17)

Instrument compile UNE fois (`go test -c`), puis **un film par processus, en avant-plan**,
sous surveillance memoire (`Start-Process -PassThru`, echantillonnage 150 ms de
`PeakWorkingSet64`, plafond 3 072 Mo, kill au-dela). Cout mesure sur 3 films (`7344d24f`,
`530820e5`, `0a247154`) AVANT de lancer les 8 autres.

| grandeur | valeur |
|---|---|
| duree par film | 3,7 s (606d9844) a 15,0 s (53ce4390) |
| duree cumulee, 11 films | **112 s** |
| pic memoire par film | **15 a 18 Mo** |
| plafond atteint / processus tue | **jamais** (18 Mo sur un plafond de 3 072 Mo) |

Le risque machine qui a motive D17 ne se materialise pas sur cet instrument : il lit un chunk
a la fois et ne retient que des compteurs. Le seul poste couteux en memoire,
`objectiveevents.StatRecords`, n'est appele que sur les 5 films dotes d'un oracle et reste
sous 20 Mo. Verdicts et couts : `LOTC_gates.log`.

## 10. Statut des items

- [x] **C.0.1** — sections 3 (slots et stabilite), 4 (records d'image-cle), 5 (records delta
  et annonces au masque, temoin fantome), 6 (verdict ti=23). Les 11 films du corpus, un par
  processus. Reserve ecrite : le walker deterministe d'image-cle ne parse qu'un record par
  table (section 4), fait deja etabli par le depot ; le recensement d'image-cle repose donc
  sur `WalkKeyframeWorld`.
- [~] **C.0.2** — hors perimetre de cette execution : attend le hook `SetManagedObjectHook`
  de l'item 0.6 (lot 0, non fusionne).
- [x] **C.0.3** — table de sortie `lotC/LOTC_table_C03.tsv` (385 lignes, triee par annonces
  decroissantes, une colonne par film), top 15 en section 5, verdict ti=23 en section 6,
  verdict ti=47 et densites en section 7.
- [!] **Gate 0** — NON TENU (section 8). Clause 1 tenue (53 et 34 composants a >= 100
  annonces) ; clause 2 non tenue (maximum 2,37x pour un seuil de 3x). Arbitrage superviseur
  requis avant toute phase 1.

## 11. Decouvertes (hors perimetre — notees, NON traitees)

1. **L'ancrage « objet du monde » a un plancher de bruit tres eleve sur les bandes larges.**
   Le temoin VIDE rend 58 a 375 records par slot. Sur `64e8adfa`, ti=10 (157 slots) n'est qu'a
   1,2x ce plancher ; sur `24dbb67d`, ti=12 est SOUS le plancher. Les scanners de production
   qui utilisent `matchWorldObjectRecord` (ti=37/41/42, `projectiles.go`, `equipment_state.go`)
   travaillent sur des bandes du meme ordre : **le taux de faux positifs de ces balayages
   n'est chiffre nulle part**. Piste : `equipment_state.go:246` exige `rec.Idx[0] == 0`, ce
   qui divise le bruit par ~64 ; les balayages qui ne l'exigent pas ne l'ont pas.
2. **`FrameConfig.IDLowBits` est une valeur de RUNTIME qui varie d'un film a l'autre**
   (`frame_records.go:39-44` : 11 sur `000d5950`, 14 sur le film de la capture live), or
   `matchWorldObjectRecord` cable 13 bits (`projectiles.go:305`, `equipment_creation.go:88`).
   Si la largeur reelle differe de 13 sur certains films, l'ancrage y lit un mauvais numero de
   slot. C'est un candidat direct pour expliquer les 22 a 39 % de records hors grammaire, et
   ca merite d'etre chiffre avant d'investir dans la phase 1.
3. **`ti=13 managed-object-property` est un archetype vivant que le plan n'a pas inventorie** :
   35 687 records sur `7344d24f`, `i1 managed-object-property-component` a 40,6x le plancher
   (4 086 annonces sur `696a9d7c`, 52,8 % des records ti=13 sur `01e1f945`), et trois
   `player-masked-property` (i13, i17, i21) entre 22x et 37x. Ses slots sont STABLES entre les
   deux Strongholds (26/26). Une « propriete d'objet, masquee par joueur » est exactement la
   forme d'un etat de zone par equipe — piste au moins aussi bonne que ti=10.
4. **`ti=47 i2 personal-ai-data-component`** (non porte) est le canal le plus concentre du
   corpus (jusqu'a 1 214,6x le plancher, 80 % des records de l'archetype) et il est
   MODE-DEPENDANT (present en zones/KOTH, absent en CTF). Aucune ligne de registre ne le
   mentionne.
5. **KOTH et Oddball n'ont aucun oracle d'evenement nomme** (`namedStatSlots` n'a que `zone`
   et `flag`), donc aucune mesure de densite n'est possible sur 5 films du corpus. Un oracle
   de repli existe pourtant : `objectiveevents.ScoreCurve` donne les increments de score de
   mode a la ms. Non employe ici (le plan prescrit `NamedEvents`), mais c'est ce qui
   permettrait de tester le gate 0 sur KOTH — le mode ou la colline BOUGE et ou l'etat vivant
   aurait le plus de valeur.
6. **`000d5950` (temoin Slayer) ne porte que 27 chunks utiles** la ou le plan §1.2 en annonce
   49. Le corpus local est peut-etre partiel pour ce film ; sans consequence pour cette phase
   (il ne sert que de temoin negatif), mais a verifier avant de s'en servir comme temoin de
   re-cuisson (D10).
7. **La table `ecs_table.tsv` est exacte** sur les sept archetypes recenses : nombre de
   composants, noms, niveaux et statuts de portage concordent avec le registre des onze films
   (le statut est demande au dispatch reel, `consumeByName`, pas a la table). Aucune ligne a
   corriger.
