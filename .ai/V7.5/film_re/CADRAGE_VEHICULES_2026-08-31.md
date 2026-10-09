# CADRAGE — les véhicules dans les films Theater (lot V0)

> Établi le 2026-08-31 dans le worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`.
> Reconnaissance et mesures légères. Aucun code de production écrit, aucun commit.
> Chaque affirmation cite sa pièce : fichier:ligne, requête SQL, ou sortie d'instrument.
> Les instruments jetables employés sont listés au § 7 ; ils sont sous garde d'environnement.

## 0. LE RÉSULTAT EN CINQ LIGNES

1. **L'archétype véhicule est `ti=40`, 48 composants, et il est STABLE** — vérifié sur le
   registre de 11 films couvrant **6 empreintes de registre distinctes** (donc plusieurs builds
   du jeu). Ce que le décodeur lit déjà et ce qui lui manque est chiffré au § 1.
2. **LA VOIE EMPLOYÉE JUSQU'ICI SUR `ti=40` DÉCODE LA POSITION AVEC LA GRAMMAIRE D'UN AUTRE
   ARCHÉTYPE.** `ti=40` porte à `i0` la forme *dynamic-precision* du BIPÈDE (porte de 5 bits),
   pas la forme *object-position* des objets du monde (porte de 3 bits). Mesuré : la grammaire
   bipède rend **99,4 à 100 %** de pas de trajectoire physiquement plausibles, la grammaire
   objet du monde **21,2 à 41,8 %**. C'est le fait central de ce cadrage (§ 2).
3. **La position, le cap et la vélocité des véhicules se lisent AVEC LE CODE EXISTANT**, à un
   point d'entrée près : `ScanBipedRecords` est déjà paramétré par une bande de slots, il suffit
   de lui donner celle de `ti=40`. Mesuré : 92,3 à 95,5 % des échantillons portent le cap `i2`.
4. **L'identité du châssis passe par le record de CRÉATION**, dont le déserialiseur de
   default-state (`FUN_1410A5A74`) n'est pas porté. La piste courte est le mot `MPPWord32`, et
   elle est étayée : `FUN_1410a5a74` figure parmi les six appelants DIRECTS du bloc MPP (§ 3).
   La piste `vehicle-type-state` (i33) est **réfutée comme signal delta** (§ 3.2).
5. **AUCUNE ALERTE CORPUS.** 26 films Super Fiesta sur Behemoth (18) et Launch Site (8) sont en
   cache et tous portent des véhicules ; 348 des 951 films du cache en portent (§ 4).

---

## 1. Q1 — L'ARCHÉTYPE VÉHICULE AUJOURD'HUI

### 1.1 Ce que dit la table de référence, et ce que dit le film

`apps/go-api/internal/analysis/filmdec/testdata/ecs_table.tsv` donne `ti=40` avec
**48 composants** (`i0` à `i47`). Le registre `chunk_00` **du film** rend exactement la même
liste, dans le même ordre (instrument `TestV0ComposantsRegistre`, § 7.2).

Comptes de la table, recomptés :

| grandeur | valeur mesurée sur `ecs_table.tsv` |
|---|---|
| couples `ti >= 0` | **1 067** |
| archétypes porteurs (`ti >= 0`, au moins un composant) | **49** (`ti=8` est vide) |
| indices d'archétype présents | 0 à 49, soit **50 blocs** |
| composants de `ti=40` | **48** |

Ceci **réconcilie l'anomalie laissée ouverte** par `VEHICULES_ARCHETYPE_40.md` § « UNE ANOMALIE
À TRANCHER » (174 contre 118) : les deux chiffres décrivent des choses différentes. Le fichier
`chunk_00` porte **118 blocs** de 64 slots (c'est le compte brut, celui que journalise
`registry_fingerprint.go:107-119`), mais seuls les **50 premiers** sont des archétypes d'objet —
borne sémantique `kfArchMax = 50` tirée de `DAT_144e61d88 COUNT=0x32`
(`keyframe_world.go:20-24`). Les blocs 50 à 117 ne portent aucun slot non vide : 118 blocs et
1 067 slots non vides sont **le même registre**, et le « 50 blocs / 49 porteurs / 1 067 couples »
du plan est la lecture sémantique du même objet. Le **174** reste sans pièce ; il est à
abandonner.

### 1.2 Statut de chaque composant de `ti=40`

D'après la colonne `status` d'`ecs_table.tsv` (`porte` = déserialiseur bit-exact câblé,
`non_porte` = aucun, `partiel` = incomplet) :

**PORTÉS — 30 composants, `i0` à `i29`** (tout le tronc objet + unité) :

```
i0  object-position-dynamic-precision   i15 object-low-frequency
i1  object-translational-velocity-dyn.  i16 object-physics-flags
i2  object-forward-and-up-dyn.-prec.    i17 object-frame-configuration
i3  object-angular-velocity-dyn.-prec.  i18 unit-control
i4  object-body-vitality                i19 unit-actor-control
i5  object-shield-vitality              i20 unit-actor-state
i6  object-region-state                 i21 unit-desired-aiming-vector
i7  object-damage-sections              i22 unit-grenade-counts
i8  object-constraint                   i23 unit-malleable-property
i9  object-multiplayer-properties       i24 unit-low-frequency
i10 object-parent-state                 i25 unit-command-tick
i11 object-dead-state (forme LOURDE,    i26 unit-equipment
    191 bits, la même que le bipède)    i27 unit-stun
i12 object-scale                        i28 unit-active-camo-state
i13 object-maximum-vitalities           i29 unit-crouch
i14 object-dissolver
```

Plus `i43 simulation-state` (**partiel**, `FUN_142ED6D88`) et `i44 simulation-state-playback`
(porté, `FUN_142ed6d20`).

**NON PORTÉS — 16 composants** : `i30` à `i42`, `i45 air-drop-flight`, `i46 warp`,
`i47 vehicle-low-frequency`.

### 1.3 Correction à `VEHICULES_ARCHETYPE_40.md` (2026-07-27)

Le document de l'état de l'art liste la partie propre au véhicule comme **`i30`-`i37`, huit
composants**. C'est **incomplet** : la partie véhicule compte **quatorze** composants.

| oublié par le document du 27/07 | nom |
|---|---|
| `i38` | `vehicle-weapon-set-component` |
| `i39` | `vehicle-auto-turret-component` |
| `i40` | `vehicle-equipment-turret-parent-component` |
| `i41` | `vehicle-seats-override-pitch-component` |
| `i42` | `vehicle-seats-override-yaw-component` |
| `i47` | `vehicle-low-frequency-component` |

En revanche la liste des composants COMMUNS du même document (`i0/i1/i2/i3/i4/i5/i11/i18/i21/
i22/i26/i28`) est **juste**, et sa conclusion « la position, la vitesse, l'orientation, la vie
et le bouclier d'un véhicule se lisent déjà » est **confirmée par la mesure** (§ 2) — mais pas
par la voie que le dépôt employait, ce qui change tout (§ 2.1).

`i41`/`i42` (`seats-override-pitch` / `seats-override-yaw`) sont notables : ce sont les deux
seuls composants du registre entier dont le nom cite un SIÈGE. Ils ne portent pas l'occupant,
mais ils confirment que la notion de siège vit dans cet archétype.

### 1.4 Ce que le FLUX porte réellement — et le plancher de faux positifs

La table dit ce que le décodeur sait lire ; elle ne dit pas ce qui arrive. Mesure sur les
records delta `ti=40` acceptés par le filtre de production (instrument `TestV0ComposantsFlux`,
§ 7.2) :

| composant | `8a049c50` Behemoth SF | `fccc61cd` Launch Site SF | `0d76e8f1` Behemoth SF 8 j. |
|---|---|---|---|
| records acceptés | 7 760 | 5 454 | 32 328 |
| `i0` position | 100,0 % | 100,0 % | 100,0 % |
| `i1` vélocité | 97,9 % | 97,5 % | 98,3 % |
| `i2` cap (forward-and-up) | 93,4 % | 91,9 % | 95,2 % |
| `i3` vitesse angulaire | 97,0 % | 92,9 % | 95,7 % |
| `i25` unit-command-tick | 99,7 % | 99,4 % | 99,7 % |
| `i34` vehicle-type-physics | **0 %** | 18,1 % | 31,1 % |
| `i37` vehicle-emp-timer | 5,3 % | 4,1 % | 2,6 % |
| `i4` body-vitality | 0,9 % | 3,7 % | 3,9 % |
| `i7` damage-sections | 0,3 % | 0,3 % | 0,2 % |

**LE PLANCHER DE FAUX POSITIFS SE MESURE DANS LA MÊME SORTIE, ET C'EST CE QUI REND LE TABLEAU
LISIBLE.** Le balayage rend aussi des index `i49`, `i50`, `i55`, `i56`, `i57`, `i58`, `i60` —
qui **ne peuvent pas exister** dans un archétype de 48 composants. Ce sont donc des ancres
fausses, et elles pèsent au plus **13 records sur 7 760, soit 0,17 %**. Conclusion à retenir
comme règle de lecture : **toute ligne au-dessous de ~0,3 % est indiscernable du bruit d'ancre
et ne doit pas être interprétée.**

Conséquences immédiates, toutes du bon côté ou du mauvais côté de ce plancher :

- Le masque nominal d'un véhicule est **`{i0, i1, i2, i3, i25}`**, plus `i34` et `i37` par
  intermittence. Cinq composants, tous **portés**.
- `i33 vehicle-type-state` : 2 records sur 5 454, 10 sur 32 328 — **sous le plancher**. Ce n'est
  pas un signal du flux delta (§ 3.2).
- `i11 object-dead-state` : 1 record sur 5 454, 0 ailleurs — **sous le plancher**. La
  destruction d'un véhicule **ne se lit pas** dans le flux delta par `i11` (§ 2.4).
- `i34 vehicle-type-physics` est absent de Behemoth et présent à 18-31 % sur Launch Site et sur
  le Behemoth à 8 joueurs : c'est un composant **conditionnel**, pas universel. À ne pas mettre
  sur le chemin critique.

### 1.5 Le registre n'est pas le même d'un build à l'autre — mais `ti=40` ne bouge pas

`registry_fingerprint.go:40-57` pose une empreinte de référence
(`0x61e492dd4de7fd4e`, 118 blocs / 1 067 slots) et alerte quand un film en diffère. Mesure sur
11 films du corpus (instrument `TestV0ComposantsRegistre`) :

| empreinte | slots non vides | films |
|---|---|---|
| `0x61e492dd4de7fd4e` (référence) | 1 067 | `000d5950`, `4fbae6c3`, `51d3ab9f`, `c1e8d359`, `941f7f8c` |
| `0x1d319f2489a2e4f9` | 1 070 | `8a049c50`, `e3b10d4b` |
| `0xa63f21e3484bc17d` | 1 067 | `0d76e8f1` |
| `0xb4555fa7df7266ae` | 1 068 | `fccc61cd` |
| `0xf4b0ed2f5e5acba` | 1 033 (116 blocs) | `084a804d` |
| `0x548f764a7b27c32b` | 1 033 (117 blocs) | `a349fea8` |

**Six grammaires distinctes dans le corpus** — et pourtant, sur les onze films, `ti=40` porte
**48 composants, avec `i33 = vehicle-type-state` et `i47 = vehicle-low-frequency`, sans
exception**. L'archétype véhicule est donc stable là où le registre global ne l'est pas.

> **Ce que ça impose au lot V1** : ne jamais câbler un index de composant en dur sans le relire
> dans le registre DU FILM traité. Le dépôt a déjà l'outil (`ParseRegistryChunk` +
> `reg.Archetype(ti)`), et `ground_weapon_creation.go:93-108` en donne le patron exact.

---

## 2. Q2 — LE SUIVI D'ENTITÉS VÉHICULE

### 2.1 LA FAUTE DE GRAMMAIRE — la découverte de ce lot

Le dépôt ne connaît qu'une voie pour les positions de `ti=40` :
`filmdec.ScanFilmWorldObjects(dir, wr, 40)` (`projectiles.go:110-121`). C'est celle qu'emploie
la sonde du 18/08 (`attachement_phase0_bord_test.go:102`), et c'est la seule employée depuis.

**Elle décode `i0` avec la grammaire d'un autre archétype.** Le registre le dit noir sur blanc :

```
ti=36  i0 = object-position-component                       <- objets du monde
ti=37  i0 = object-position-component
ti=38  i0 = object-position-component
ti=39  i0 = object-position-component
ti=41  i0 = object-position-component
ti=42  i0 = object-position-component
ti=43  i0 = object-position-component

ti=35  i0 = object-position-dynamic-precision-component     <- BIPÈDE
ti=40  i0 = object-position-dynamic-precision-component     <- VÉHICULE
```

`ti=40` et `ti=35` sont **les deux seuls** archétypes à porter la forme *dynamic-precision*.
Les deux grammaires diffèrent par leur PORTE, et le dépôt les documente lui-même :

| chemin | porte | total | référence |
|---|---|---|---|
| objet du monde (`object-position-component`) | **3 bits** — precHigh + index-sel + index de région | 45 | `traverse.go:143-171` ; `decodeWorldObjectPos`, `projectiles.go:362-379` |
| bipède (`…-dynamic-precision-…`) | **5 bits** — 3 spine + 1 useDefault + 1 index de région | 47 | `traverse.go:122-141` ; `i0SpineBits=3`, `i0UseDefaultBits=1` (`i0_layout.go:51-52`) |

Lire un véhicule avec la porte à 3 bits, c'est attaquer les trois axes **deux bits trop tôt**.
Comme la porte nominale est faite de zéros dans les deux cas, aucun contrôle de format ne le
détecte : le test `PeekBits(pay, at, 3) != 0` passe, les axes sortent, la valeur est fausse.

### 2.2 La mesure qui tranche, et son critère écrit avant

Une lecture de 17 bits redonne toujours une coordonnée **dans** l'emprise de la carte :
« être dans la carte » ne discrimine rien. Ce qui discrimine, c'est que **une trajectoire réelle
est continue**. Critère posé avant la mesure : le véhicule le plus rapide de Halo Infinite reste
largement sous **35 m/s** (le dépôt retient déjà cet ordre de grandeur,
`offline_biped.go:137-140`) ; un pas au-delà n'est pas un déplacement, c'est une lecture
désalignée. Pas mesuré = deux échantillons consécutifs du même slot séparés de 0 < Δt ≤ 2 s.

Sortie de `TestV0CadrageGrammaireI0` (§ 7.1), **même film, même bande de slots `ti=40`** :

| film | carte | slots | objet du monde (3 bits) | bipède dyn.-préc. (5 bits) | témoin fantôme (grammaire bipède) |
|---|---|---|---|---|---|
| `8a049c50` | Behemoth | 28 | 7 585 pas, **41,8 %** < 35 m/s | 7 696 pas, **99,4 %** | 3 pas, 33,3 % |
| `fccc61cd` | Launch Site | 21 | 5 346 pas, **37,4 %** | 5 395 pas, **100,0 %** | 0 pas |
| `51d3ab9f` | Launch Site | 39 | 15 170 pas, **33,5 %** | 15 284 pas, **99,8 %** | — |
| `0d76e8f1` | Behemoth | 47 | 32 020 pas, **21,2 %** | 32 149 pas, **99,9 %** | 3 pas, 33,3 % |

Le **témoin fantôme** — même décodeur, même grammaire gagnante, sur une bande de MÊME
cardinalité faite de slots jamais vus porter le moindre archétype — rend **0 à 3 pas** contre
5 000 à 32 000. Le critère ne se satisfait donc pas tout seul : c'est bien le signal qui le
satisfait, pas la mesure qui l'accorde d'office.

**Verdict : la grammaire d'`i0` de `ti=40` est celle du bipède. Tout chiffre de position de
véhicule produit avant ce cadrage — sonde du 18/08 comprise — est à considérer comme non
mesuré.** Cela n'invalide pas la conclusion « gate 0 négatif » de cette sonde (elle portait sur
le champ `i10`, pas sur la position), mais l'oracle géométrique qui la soutenait travaillait sur
des coordonnées fausses : il est à rejouer.

### 2.3 Ce qui se lit DÉJÀ, et le morceau exact qui manque

`ScanBipedRecords` (`offline_biped.go:264-300`) est **déjà paramétré par une bande de slots**.
Rien dedans n'est propre au bipède, à trois réglages près, tous accessibles :

- `opt.RequireTag1` : à **désarmer** (le tag de 2 bits est la génération du handle et les objets
  du monde en emploient les quatre — règle établie par `matchWorldObjectRecord`,
  `projectiles.go:311-316`) ;
- `bipedMinMaskCnt = 2` (`offline_biped.go:52`) : convient, le masque véhicule en porte cinq ;
- `ascendingFromZero` exige `i0` en tête (`offline_biped.go:357-369`) : c'est vrai à 100 %.

**LE SEUL MORCEAU MANQUANT EST UN POINT D'ENTRÉE.** `ScanFilmBipedPositions`
(`offline_biped.go:159-213`) relève lui-même sa bande via `bipedSlotBand`
(`offline_biped.go:220-241`), qui filtre en dur `r.TI == BipedTypeIndex`. Il faut une entrée
`ScanFilmBipedPositionsForBand(dir, band, opt)` — la bande venant de
`ScanFilmWorldObjectKeyframes(dir, 40).Band` (`world_object_census.go:63-87`), déjà exportée.
Le cadrage l'a fait tenir en **une boucle de vingt lignes** dans l'instrument
(`v0ScanBipedeSurBande`) sans toucher au décodeur : c'est un refactor d'exposition, pas un
décodage nouveau.

**ORIENTATION ET VITALITÉ VIENNENT AVEC, DANS LE MÊME RECORD.** `opt.CaptureDirs` poursuit le
curseur après `i0` sur `i1`/`i2`/`i3`/`i4`/`i5` (`scanRecordDirs`, `offline_aim.go:158-186`) et
s'arrête proprement au premier composant non modélisé — soit `i25`, qui vient après. Mesuré :

| film | échantillons | `i2` cap | `i1` vélocité | `i4` vitalité |
|---|---|---|---|---|
| `0d76e8f1` Behemoth | 32 246 | 30 784 (**95,5 %**) | 29 038 (90,1 %) | 1 249 (3,9 %) |
| `fccc61cd` Launch Site | 5 431 | 5 011 (**92,3 %**) | 4 570 (84,1 %) | 201 (3,7 %) |

> **RÉSERVE À NE PAS SAUTER.** Ces taux disent que le CURSEUR ARRIVE, pas que les valeurs sont
> justes. Deux noms diffèrent entre `ti=35` et `ti=40` : `i2` est
> `object-forward-and-up-component` chez le bipède et `…-dynamic-precision-…` chez le véhicule ;
> `i3` de même. `consumeByName` route déjà les deux orthographes vers le même déserialiseur
> (`traverse.go:214` pour `i3`, `traverse.go:302-304` pour `i2`, avec le commentaire « ti=38 i2
> (reuse biped i2 deser) ») — mais c'est une réutilisation héritée de `ti=38`, **jamais mesurée
> sur `ti=40`**. Le cap publié doit être confronté à la direction de déplacement avant d'être
> dessiné : c'est un gate du lot V1, pas un acquis.

### 2.4 Création (NEW) et destruction — les deux vrais trous

**CRÉATION : bloquée par un déserialiseur non porté.** Le record NEW est ce qui dit l'archétype
et porte l'état initial (`equipment_creation.go:18-25`). Sa lecture exige le déserialiseur du
default-state de l'archétype. Or :

- `KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md:30` donne `ti=40 → vtable[0x60] = 0x1410A5A74`,
  classé **REAL** — donc **pas** le stub à zéro bit ;
- `default_state_arch.go:44-65` : la table `defaultStateDeserByTI` **n'a pas de clé `40`** ;
- `default_state_arch.go:23-24` : `ti=40` **ne figure pas** dans la liste des stubs mesurés.

Donc `ti=40` consomme un nombre de bits inconnu et non nul, et le record NEW est illisible. **Il
n'y a aucun mystère sur la marche à suivre** : `equipCreationWalk`
(`equipment_creation.go:249-280`) est **déjà paramétrée par `(ti, deser)`**, et
`ground_weapon_creation.go:72-75` montre l'appel complet pour `ti=42`. Le lot V1 a donc
exactement **deux artefacts à produire** : un `consumeDefaultStateTI40` porté depuis
`FUN_1410A5A74` (`default_state_ti42.go` est le modèle d'écriture, feuille par feuille) et une
ligne de plus dans `defaultStateDeserByTI`.

**DESTRUCTION : `i11` est porté, mais il n'arrive pas.** `ecs_table.tsv` donne `ti=40 i11` comme
**porté**, forme LOURDE de 191 bits, la même que le bipède — la mécanique de capture existe
(`EntityTrace.Dead`, `traverse.go:36`). Mais la mesure du § 1.4 le trouve **0 à 1 fois** dans
tout un film, sous le plancher de faux positifs. La destruction d'un véhicule **ne se lira pas**
dans le flux delta par `i11`.

Ce qui la BORNE, en revanche, existe et fonctionne : le recensement des images-clés
(`ScanFilmWorldObjectKeyframes`, `world_object_census.go:63-87`) donne, par vie `(slot, gen)`,
les instants où elle est recensée. La dernière image-clé qui recense une vie et la première qui
ne la recense plus **encadrent** sa disparition — c'est exactement l'acquis établi pour `ti=37`
(`world_object_census.go:15-25`). Mesuré (instrument `TestV0CadrageRecensement`, § 7.1) :

| film | carte / mode | images-clés | bande | vies `ti=40` | dont vues 1 fois | durée médiane |
|---|---|---|---|---|---|---|
| `8a049c50` | Behemoth SF | 25 | 28 | **27** | 7 | 100 s |
| `e3b10d4b` | Behemoth SF | 30 | 67 | **62** | 31 | 60 s |
| `fccc61cd` | Launch Site SF | 29 | 21 | **21** | 2 | 160 s |
| `4fbae6c3` | Launch Site SF | 26 | 16 | **16** | 3 | 120 s |
| `c1e8d359` | Behemoth Team Slayer | 37 | 13 | **13** | 3 | **420 s** |
| `941f7f8c` | Bazaar SF (témoin) | 25 | **0** | **0** | 0 | — |

Deux lectures qui comptent :

- **Aucune vie n'est recensée sur toute la durée du match**, sur aucun film. Les véhicules
  naissent et meurent : la matière du lot V2 est là, mesurée.
- La **confirmation du plan** sur le mode : le Team Slayer classique rend 13 vies de durée
  médiane **420 s** contre 27 à 62 vies de 60 à 100 s en Super Fiesta. « Moins de véhicules, non
  aléatoires » n'est plus une hypothèse.

### 2.5 Comparaison aux précédents `ti=37` / `ti=41` / `ti=42`

| étape | équipement `ti=37` | arme au sol `ti=42` | projectile `ti=41` | véhicule `ti=40` |
|---|---|---|---|---|
| grammaire `i0` | objet du monde | objet du monde | objet du monde | **bipède dyn.-préc.** |
| position delta | oui | **RÉFUTÉE** (`keyframe_ground_weapons.go:118-155`) | oui | **oui, mesurée ici** |
| default-state porté | oui (`FUN_1407f105c`) | oui (`FUN_1407f0c68`) | non câblé (`0x1408EFB58`) | **NON (`0x1410A5A74`)** |
| record NEW lisible | oui | oui | — | **non** |
| identité | `MPPWord32` = GlobalID `eqip`, 11/11 | `MPPWord32`, hypothèse `weap` | — | **`MPPWord32`, hypothèse `vehi`** |
| recensement images-clés | oui | oui | — | **oui, mesuré ici** |

**La différence décisive avec `ti=42`** : la position des armes au sol a été réfutée parce
qu'une arme posée **ne bouge pas** — mesurer sa dispersion n'avait pas de sens, et le témoin
fantôme rendait 493 slots contre 661. Ici le témoin fantôme rend **1 vie contre 84** et
**0 à 3 pas contre 32 000**. Le véhicule est l'inverse du cas arme au sol : c'est un objet dont
le mouvement EST le signal.

---

## 3. Q3 — L'IDENTITÉ DU CHÂSSIS

### 3.1 Voie A — `MPPWord32` du record de création (la plus courte)

**La pièce qui la fonde.** `.ai/V7.5/killweapon/KILLFEED_STATE.md:166-168` énumère les six
appelants DIRECTS de `FUN_14080cfe8`, le déserialiseur du bloc
`object-multiplayer-properties` :

> « Le NEW deser `FUN_14080cfe8` a 6 callers DIRECTS (`FUN_1407f2224`, `FUN_1408efb58`,
> `FUN_1408f0b48`, `FUN_140f44c38`, `FUN_140fe7630`, `FUN_1410a5a74`) »

`FUN_1410a5a74` **est** le default-state de `ti=40`
(`KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md:30`). Autrement dit : **le record de création d'un
véhicule contient un bloc MPP**, comme celui d'un équipement et celui d'une arme au sol. Or
`consumeMultiplayerPropertiesBlock` (`default_state.go:331-365`) publie
`MPPWord32` — le mot de 32 bits **inconditionnel**, présent sur tous les records.

Le précédent est établi deux fois : pour `ti=37`, ce mot est le **GlobalID du tag `eqip`** de
l'objet, résolu **11 valeurs sur 11** dans les 105 tags `eqip` du jeu
(`equipment_creation.go:27-34`) ; pour `ti=42`, l'hypothèse est le tag `weap`
(`ground_weapon_creation.go:18-21`). **Pour `ti=40`, l'hypothèse est le tag `vehi`.**

**Preuve attendue, à écrire avant la mesure du lot V1** — trois critères, tous falsifiables :

1. **Constance par vie** : `MPPWord32` doit être identique pour tous les records de création
   d'une même vie `(slot, gen)`, et le nombre de valeurs distinctes par film doit être de
   l'ordre du nombre de types de châssis de la carte (quelques unités), pas de l'ordre du
   nombre de vies (des dizaines).
2. **Résolution en `vehi`** : chaque valeur doit se résoudre dans les modules du jeu comme un
   tag de groupe `vehi`. L'outillage existe (`himap.ModuleIndex.Lookup`,
   `moduleindex.go:75-100`) et la chaîne `vehi → weap` est déjà parcourue par le chantier
   killweapon (règle R-VÉHICULE : 46 `weap` référencés par un `vehi` DIRECT).
3. **Oracle de position**, celui qui a validé `ti=42` : un déserialiseur juste fait atterrir le
   curseur exactement au début d'`i0` — 282 atterrissages sur 289 pour `ti=42`, contre **0 sur
   289** pour trois déserialiseurs faux passés par le même code
   (`keyframe_ground_weapons.go:131-141`). C'est le gate de portage de `consumeDefaultStateTI40`.

**Coût** : une session Ghidra sur `FUN_1410A5A74` (feuille par feuille, modèle
`default_state_ti42.go`), puis deux lignes de câblage. **Risque** : moyen — le déserialiseur
peut appeler des sous-fonctions non encore portées ; `default_state_arch.go:30-32` pose la règle
(« un archétype dont UNE largeur de feuille n'est pas établie statiquement n'est PAS inscrit
dans la table »).

### 3.2 Voie B — `vehicle-type-state` (i33) : RÉFUTÉE comme signal delta

Le nom promettait. La mesure ne suit pas : `i33` apparaît dans **2 records sur 5 454**
(`fccc61cd`) et **10 sur 32 328** (`0d76e8f1`), soit 0,0 % dans les deux cas — **sous le
plancher de faux positifs de 0,17 %** établi au § 1.4. `i33` n'est pas un signal du flux delta.

Cela ne le condamne pas partout : il peut vivre dans l'état complet du record de CRÉATION ou de
l'image-clé, où tous les composants sont émis. Mais il n'y sera lisible qu'après le
default-state — c'est-à-dire **après** la voie A. Il ne raccourcit rien.

### 3.3 Voie C — nommer le châssis : la moitié déjà dans le dépôt, avec ses limites

`internal/games/halo_infinite/film/damagetag/data/labels.tsv` porte **89 entrées de classe
`VEHICULE`** (recompté ce jour), dont le champ `Detail` cite les banques Wwise du châssis.
**24 racines distinctes** y sont citées (recomptées ce jour ; `GUIDE_KILLSOURCE.md` § 6ter en
annonce 19 — l'écart n'est pas arbitré ici, il est signalé). Par fréquence, châssis d'abord :

```
CHASSIS (11)   veh_un_wasp 10 · veh_cv_banshee 10 · veh_cv_ghost 8 · veh_un_scorpion 6
               veh_bt_chopper 6 · veh_un_falcongrenadelauncher 3 · veh_un_rockethog 2
               veh_un_pelican 2 · veh_cv_wraith 2 · veh_cv_phantom 2 · veh_un_falconlmgturret 1
TOURELLES (6)  tur_un_gausscannon 15 · tur_un_machinegun 14 · tur_cv_shadeturret 12
               tur_un_rocketturret 10 · tur_cv_plasmacannon 4 · tur_bt_gatlingmortar 2
AUTRES (7)     lvl_moments_ge_shared_autoturret_banished(_fire) 3+3 · chm_ge_weaanim_plasmaturret 3
               prototype_weapon_lightrifle 4 · wea_fr_heatwave 3 · cfx_un_spartan 2
               wea_cv_plasmapistol 1
```

**TROIS LIMITES, toutes documentées, aucune contournable** :

1. Ce sont les tags de l'**ARMEMENT**, pas du châssis. Un véhicule non armé (Mongoose,
   Razorback) n'y figure pas — et de fait aucune racine `veh_un_warthog`, `veh_un_mongoose` ou
   `veh_un_gungoose` n'apparaît ci-dessus.
2. Il ne nomme qu'un véhicule **qui a tué**. Un véhicule présent et jamais meurtrier est muet.
3. **La racine de banque n'est pas une identité de châssis** : seuls 53 tags sur 89 (59,6 %)
   citent une racine unique ; `0000d4ff` en cite cinq (Scorpion, Chopper, Banshee, Wasp, Shade).
   La règle anti-invention de `GUIDE_KILLSOURCE.md` § 6ter est **non négociable** : publier la
   disjonction entière, jamais choisir.

**Usage juste de cette voie** : elle fournit le **vocabulaire de châssis** attendu (11 racines
`veh_*`, 6 racines `tur_*`) et servira de **table de traduction** une fois la voie A ouverte,
ainsi qu'au lot S1 (sons). Elle ne remplace pas l'identité lue dans le film.

### 3.4 Recommandation

**Voie A, avec la voie C en table de noms.** Ordre du lot V1 : porter
`consumeDefaultStateTI40` → valider par l'oracle de position → lire `MPPWord32` → résoudre en
`vehi` par `himap` → traduire en nom lisible via le vocabulaire de la voie C. Chaque étape a un
précédent exécuté dans ce dépôt et un gate chiffré.

---

## 4. Q4 — LE CORPUS

### 4.1 Comment le cache s'indexe

Par les **8 caractères hexadécimaux qui précèdent le premier tiret du `match_id`**
(`internal/domain/title/film_id.go:28-36`) :
`data/cache/film_chunks/<short8>/chunk_NN.bin` et `film_manifests/<short8>.json`
(`filmcache.go:5-6`). La jointure SQL est donc `substr(match_id,1,8) = short8`.

**951 répertoires de chunks** en cache, **954 manifests**. Les 951 s'apparient **tous** à une
ligne de `match_registry` — aucun film orphelin.

### 4.2 Behemoth et Launch Site — PAS D'ALERTE CORPUS

Requête sur `shared_matches_v2.duckdb` (attaché `READ_ONLY`) croisée avec la liste des
répertoires du cache :

| carte | mode (`pair_name`) | films en cache | dernier |
|---|---|---|---|
| Behemoth | `Super Fiesta:Slayer on Behemoth - Forge` | **18** | 2026-03-22 |
| Behemoth | `Arena:Team Slayer on Behemoth - Forge` | 4 | 2026-03-24 |
| Behemoth | `Arena:Slayer on Behemoth - Forge` | 3 | 2026-03-19 |
| Behemoth | `Arena:CTF on Behemoth` | 2 | 2026-02-03 |
| Behemoth | `Arena:Neutral Flag CTF on Behemoth` | 2 | 2026-01-08 |
| Behemoth | `Fiesta:Slayer on Behemoth - Forge` | 1 | 2025-12-22 |
| Behemoth | `Tactical:Slayer on Behemoth - Forge` | 1 | 2025-12-06 |
| Behemoth | `Arena:King of the Hill on Behemoth` | 1 | 2025-11-01 |
| Launch Site | `Super Fiesta:Slayer on Launch Site - Forge` | **8** | 2026-07-24 |

**26 films Super Fiesta sur les deux cartes porteuses. Les 26 portent des slots `ti=40`.** Les
deux cartes sont par ailleurs au catalogue de bornes
(`data/titles/halo_infinite/reference/map_quant_bounds.json` : `behemoth` module `va_behemoth`,
`launch site` module `va_launchsite`, toutes deux `axisWidths [17,17,15]`) — donc **qualifiables
au sens du chemin de position**, ce qui n'allait pas de soi.

### 4.3 Le bon critère de qualification n'est pas le nombre de slots

Le balayage du cache entier (§ 4.4) trouve `ti=40` sur des cartes d'arène sans véhicule
conduisible : Illusion 38/38 films (7 slots), Live Fire 31/31 (2 slots), Goliath 13/13 (1 slot).
Mesure décisive sur `49e6248f` (Illusion Super Fiesta, 7 slots en image-clé) :
**17 records delta acceptés sur tout le film**, aucun `i1`/`i2`/`i3`/`i25` — contre **32 328
records, `i1`/`i2`/`i3` > 95 %** pour `0d76e8f1` (Behemoth SF).

Ces entités `ti=40` d'arène sont statiques : tourelles montées, objets de classe véhicule, ou
liaisons du marcheur d'images-clés. **Ce ne sont pas des véhicules conduits.**

> **CRITÈRE DE QUALIFICATION D'UN FILM, à retenir pour V1** : le **nombre de records delta
> `ti=40` acceptés**, pas le nombre de slots en image-clé. Seuil proposé, à confirmer :
> **≥ 1 000 records** (les quatre films mesurés sont à 5 454 / 7 760 / 15 000 / 32 328 ;
> l'Illusion à 17).

### 4.4 Le corpus total, mesuré

Balayage des **951** films du cache, trois premiers chunks chacun
(instrument `TestV0CadrageBalayageCache`, § 7.1) : **348 films (36,6 %)** portent au moins un
slot `ti=40`. Répartition par carte (extrait, colonne `max_slots` = maximum sur les trois
premiers chunks) :

| carte | films | avec `ti=40` | max slots | lecture |
|---|---|---|---|---|
| Fragmentation Heavies | 7 | 7 | **40** | BTB lourd — le plus dense |
| Thunderhead Heavies | 4 | 4 | 32 | BTB lourd |
| Highpower | 6 | 5 | 30 | BTB |
| Fortitude Heavies | 4 | 4 | 30 | BTB lourd |
| Threshold | 5 | 5 | 28 | BTB |
| Deadlock / Fragmentation | 9 / 8 | 8 / 8 | 22 | BTB |
| Obituary / Insolence | 6 / 6 | 6 / 5 | 20 | BTB |
| Oasis | 6 | 6 | 17 | BTB |
| Breaker | 5 | 5 | 15 | BTB |
| **Behemoth** | 32 | 31 | **12** | arène à véhicules |
| Command | 10 | 10 | 13 | — |
| Starboard | 14 | 12 | 11 | — |
| Snowbound | 20 | 20 | 10 | — |
| **Launch Site** | 8 | 8 | **9** | arène à véhicules |
| High Ground | 15 | 13 | 9 | — |
| Illusion | 38 | 38 | 7 | **statique — pas de véhicule conduit** |
| Live Fire / The Pit / Empyrean | 31 / 17 / 15 | 31 / 16 / 15 | 2 | **statique** |
| Goliath / Cliffside | 13 / 12 | 13 / 11 | 1 | **statique** |

Ordre de grandeur : les cartes BTB (Heavies inclus) offrent **~100 films** avec deux à quatre
fois plus d'entités que Behemoth. Elles sont le corpus de **volume** ; Behemoth et Launch Site
restent le corpus de **référence** (peu de châssis, mode connu de l'utilisateur, carte au
catalogue de bornes).

### 4.5 Les meilleurs candidats

Classement des 26 films Super Fiesta par slots `ti=40` (trois premiers chunks), avec date, durée
et nombre de joueurs suivis :

| rang | short8 | carte | slots | date | durée | joueurs | note |
|---|---|---|---|---|---|---|---|
| 1 | **`0d76e8f1`** | Behemoth | 10 | 2025-10-13 | 658 s | **8** | **LE MEILLEUR** : 47 slots sur tout le film, **32 328 records delta**, `i34` à 31 % — le plus riche mesuré |
| 2 | **`ef058ab4`** | Behemoth | **12** | 2026-01-22 | 474 s | 1 | densité maximale au recensement |
| 3 | **`8a049c50`** | Behemoth | 10 | 2026-03-22 | 473 s | 1 | mesuré de bout en bout ici (27 vies, 7 760 records, 99,4 %) |
| 4 | `4898d586` | Behemoth | 11 | 2026-02-19 | 658 s | 3 | récent, long, 3 joueurs suivis |
| 5 | `e624c2a4` | Behemoth | 11 | 2026-02-18 | 514 s | 3 | — |
| 6 | **`fccc61cd`** | Launch Site | 7 | **2026-07-24** | 567 s | 0 | **le plus récent du cache** ; mesuré ici (21 vies, 5 454 records, 100,0 %) |
| 7 | `d99e5dbd` | Launch Site | 9 | 2026-02-15 | 564 s | 3 | meilleur Launch Site par densité |
| 8 | `51d3ab9f` | Launch Site | 7 | 2026-01-21 | **719 s** | 1 | le plus long ; mesuré (39 slots, 15 284 pas, 99,8 %) |
| 9 | `e3b10d4b` | Behemoth | 7 | 2026-03-17 | 586 s | 1 | 67 slots / 62 vies sur tout le film — fort renouvellement |
| 10 | `f82203b7` | Launch Site | 6 | 2026-02-09 | 619 s | 2 | — |

**Témoins négatifs, déjà mesurés à zéro** — même mode (Super Fiesta), cartes sans véhicule :
`5fead0af` et `0d6f6eaa` (Streets), `941f7f8c` (Bazaar), `22495b4d` (Aquarius). Les quatre
rendent **0 slot `ti=40`**, et `941f7f8c` rend **une bande vide** au recensement complet. Ajouter
`f0680b37` (Behemoth **Tactical** Slayer, 0 slot) : c'est le témoin le plus fort du lot, car il
isole le MODE à carte constante.

**Témoins positifs BTB** conservés du lot du 18/08 : `084a804d` (Fortitude Heavies, 26 slots),
`a349fea8` (Fragmentation Heavies, 40 slots).

### 4.6 Une pièce manquante côté outillage

Le fixture film → carte des tests de la phase 0 (`attCartes`,
`attachement_phase0_cartes_test.go`) **ne couvre aucun film du corpus véhicules**. Les
instruments de ce cadrage contournent en recevant le nom de carte par l'environnement. Le lot V1
doit prendre le nom de carte de `match_registry.map_name` (jointure du § 4.1) plutôt que
d'allonger un fixture à la main.

---

## 5. Q5 — LE PLAN DES LOTS V1 ET V2

### 5.1 V1 — détection, identité, position, état

Cinq étapes ordonnées. **Chaque seuil est écrit ici, avant toute mesure.** Chaque étape est un
gate : on n'entame pas la suivante avant de l'avoir passée.

#### V1.1 — Qualifier le corpus (pas de décodage nouveau)

Publier, pour les 26 films Super Fiesta + les 6 Behemoth non-SF + les 2 BTB témoins + les
5 témoins négatifs, le nombre de records delta `ti=40` acceptés et le nombre de vies recensées.

- **Gate** : les films retenus ont **≥ 1 000 records** ; **les 5 témoins négatifs rendent une
  bande vide** (0 slot, 0 record). Un témoin négatif non nul arrête le lot.
- **Coût** : faible (l'instrument existe, § 7). **Risque** : nul.

#### V1.2 — Exposer le balayage bipède sur bande arbitraire

Ajouter `ScanFilmBipedPositionsForBand(dir, band, opt)` à côté de `ScanFilmBipedPositions`, la
seconde devenant un appel de la première avec `bipedSlotBand`. Aucun changement de grammaire.

- **Gate de non-régression, non négociable** : sur `000d5950`, `ScanFilmBipedPositions` rend
  **exactement le même nombre d'échantillons et les mêmes quanta `Q`** qu'avant le refactor.
  Le pipeline arme-du-kill (59/59 en Theater) doit rester à 59/59.
- **Gate de fonction** : sur les 3 films mesurés, la part de pas sous 35 m/s reste **≥ 99 %**.
- **Coût** : très faible. **Risque** : faible, borné par le gate de non-régression.

#### V1.3 — Valider la grammaire d'`i2` et `i3` sur `ti=40`

Le cap est lu à 92-95 % (§ 2.3) mais la réutilisation du déserialiseur bipède pour les
orthographes *dynamic-precision* n'a jamais été mesurée sur `ti=40`.

- **Oracle** : pour les échantillons dont la vélocité `i1` dépasse 5 m/s, l'écart circulaire
  entre le cap `i2` et le cap de déplacement (calculé sur les positions consécutives) doit avoir
  une **moyenne circulaire sous 15°** et une **médiane d'écart absolu sous 30°**. C'est le patron
  de validation déjà employé pour le cap de visée du bipède (`offline_aim.go:68-78`, écart
  moyen nul à moins de 2°).
- **Témoin** : le même calcul avec les caps **permutés** entre échantillons doit rendre une
  distribution uniforme (médiane d'écart ≈ 90°).
- **Si l'oracle échoue** : ne pas publier le cap. La position seule suffit au lot V1 ; le cap
  passe au registre des reports. **Ne pas bricoler une correction d'angle sans mesure.**
- **Coût** : faible. **Risque** : moyen — c'est le point le plus susceptible de tomber.

#### V1.4 — Porter le default-state de `ti=40` et ouvrir le record NEW

Décompiler `FUN_1410A5A74` feuille par feuille (modèle d'écriture : `default_state_ti42.go`),
écrire `consumeDefaultStateTI40`, l'inscrire dans `defaultStateDeserByTI`, brancher
`equipCreationWalk{ti: 40, deser: consumeDefaultStateTI40}`.

- **Gate d'oracle de position**, celui qui a validé `ti=42` : **≥ 95 % des records de création
  reconnus atterrissent exactement au début d'`i0`** (référence `ti=42` : 282/289 = 97,6 %).
- **Témoin obligatoire** : trois déserialiseurs FAUX (largeurs voisines) passés par le même code
  doivent rendre **0 %**. Sans ce témoin, l'oracle ne prouve rien.
- **Témoin fantôme** : bande de même cardinalité, slots jamais vus porter `ti=40` → **0 record
  de création accepté**.
- **Coût** : élevé (session Ghidra). **Risque** : moyen-élevé — grammaire potentiellement longue
  (celle de `ti=42` fait 80 bits minimum). C'est le chemin critique du lot.

#### V1.5 — L'identité du châssis

Lire `MPPWord32` de chaque record de création `ti=40`, croiser avec `himap.ModuleIndex`.

- **Gate 1, constance** : `MPPWord32` **constant à 100 %** sur les records d'une même vie
  `(slot, gen)`.
- **Gate 2, cardinalité** : **≤ 8 valeurs distinctes par film** sur Behemoth / Launch Site
  (une poignée de châssis), contre 16 à 62 vies recensées. Si le nombre de valeurs suit le
  nombre de vies, ce n'est pas une identité.
- **Gate 3, résolution** : **≥ 90 %** des valeurs distinctes se résolvent en un tag de groupe
  `vehi` dans les modules du jeu (référence `ti=37` : 11/11).
- **Gate 4, cohérence croisée** : sur un film où un véhicule a tué, le châssis nommé par
  `MPPWord32` doit être **compatible** avec la disjonction de banques Wwise du `damagetag`
  correspondant (§ 3.3). Compatible, pas égal — la règle anti-invention interdit de choisir.
- **Coût** : moyen. **Risque** : moyen. **Dépend entièrement de V1.4.**

#### Livrable de V1 et gate global

Sur `0d76e8f1` (Behemoth SF, 8 joueurs) **et** `fccc61cd` (Launch Site SF), publier : la liste
des véhicules du match avec châssis nommé (ou la disjonction), leur trajectoire, leur cap si
V1.3 a passé, leur intervalle de vie borné par le recensement. **Sur les 5 témoins négatifs :
zéro véhicule, zéro trajectoire.**

### 5.2 V2 — emplacements, spawns, cooldowns

**Prérequis** : V1.4 (le record de création porte la position de naissance) et V1.5 (le châssis).

1. **Emplacements de spawn** : agréger les positions de naissance par châssis sur les 18 films
   Behemoth SF. **Gate** : les naissances d'un même châssis se groupent en **≤ 4 amas de rayon
   ≤ 2 m**, et le nombre d'amas est stable d'un film à l'autre (± 1).
2. **Confrontation `.mvar`** : croiser ces amas avec les emplacements du `map.mvar` de la carte.
   Précédent au centimètre : les socles de power-up
   (`project_carte_oracle_positions_bornage`). **Gate** : **≥ 80 % des amas** à moins de
   **1 m** d'un emplacement déclaré. **PIÈGE DOCUMENTÉ, à ne pas réapprendre** : une carte Forge
   est un **canevas + rack** — Behemoth et Launch Site sont toutes deux étiquetées « - Forge »
   dans `pair_name`, donc **ce piège s'applique ici**, il n'est pas théorique.
3. **Cooldowns** : par emplacement, mesurer l'écart entre la fin bornée d'une vie et la
   naissance suivante au même emplacement. **Gate** : la distribution doit être **unimodale et
   resserrée** (écart interquartile ≤ 25 % de la médiane) — un cooldown est une constante de
   jeu, pas une loi étalée. **Limite structurelle à écrire dans le livrable** : les images-clés
   sont espacées de ~20 s, donc **la fin de vie est bornée à ±20 s, jamais datée**. Un cooldown
   de 30 s n'est pas mesurable à cette résolution ; il faudra alors un signal de destruction
   dans le flux (piste : `i14 object-dissolver`, porté, 0,1 % — à instruire, pas à supposer).
4. **État (détruit ou non)** : la borne du recensement donne « disparu entre T1 et T2 ». Ne
   **jamais** écrire « détruit à T » tant qu'un signal daté n'existe pas.

### 5.3 ÉVALUATION — réimplémenter le walker d'événements (board / exit) ?

**Contexte, et il est contraignant.** Le modèle de paquet
`[1 bit config][liste d'événements][trame de records ECS]` et l'arithmétique
`octet0 = 0xC0 | (type >> 1)` viennent du chantier trame mené **hors de ce dépôt**. Les mesures
citées (`biped_board_vehicle` 374 sur 154 films, `unit_exit_vehicle` 5 600 sur 279 films,
`unit_enter_vehicle` type 53 = 0 en arène, occupant + siège lisibles) sont réelles ; **le code ne
l'est pas ici**. `PLAN_VEHICULES_TOURELLES.md:46-51` l'interdit sans décision explicite.

**Ce que ça apporterait** : l'OCCUPANT et le SIÈGE — donc la couleur d'équipe du conducteur, que
le lot V4 (sprites teintables) réclame explicitement.

**Ce que ça coûterait** : la grammaire est connue au niveau du cadre, pas au niveau de la charge
utile. Il faudrait porter le décodage des trois références gardées et de la charge par type
d'événement, puis re-mesurer sur notre corpus. Ordre de grandeur : comparable à V1.4 (une
session de rétro-ingénierie + un lot de mesures), soit **le double du chemin critique de V1**.

**Ce que ça risquerait** : une divergence à deux décodeurs du même fait — exactement la faute
que le dépôt a déjà payée avec le parser statborg, supprimé pour cette raison
(`filmdec/doc.go:8-16`). Et le **piège explicitement daté** du 30/08 : « tout NOM ou NUMÉRO de
type d'événement antérieur au 30/08 est sans valeur ». Un portage fait ici, sans les notes
`NOTE_*_2026-08-30/31`, repartirait précisément du matériel déclaré suspect.

**RECOMMANDATION — NE PAS RÉIMPLÉMENTER, et ce n'est pas un report par prudence** : V1 et V2
tels que planifiés **n'en ont pas besoin**. Détection, identité, position, cap, état, spawns et
cooldowns se lisent tous dans la TRAME ECS, présente ici. L'occupation par événements est
nécessaire au lot **V4** seulement.

**Trois issues, à trancher par le superviseur ou l'utilisateur** :

- **(a) attendre l'atterrissage du code externe** — coût nul ici, calendrier subi. *Recommandée.*
- **(b) demander l'export du walker** (un fichier, une grammaire, ses tests) — coût faible,
  dépendance humaine, pas de double décodeur.
- **(c) réimplémenter** — coût élevé, risque de double décodeur divergent, matériel de départ
  déclaré suspect. **À n'ouvrir que si V4 est déclaré bloquant ET que (a) et (b) sont fermées.**

**Repli mesurable si l'occupation reste fermée** : l'oracle géométrique de coïncidence prolongée
(`attachement_phase0_bord_test.go:31-46`, seuils 1,5 m / 3 s) **doit être rejoué** — il tournait
sur des positions de véhicule décodées avec la mauvaise grammaire (§ 2.2). Rejoué sur des
positions justes, il pourrait suffire à attribuer un conducteur. **C'est un candidat sérieux, et
il ne coûte qu'un lancement.**

### 5.4 Ordre recommandé et dépendances

```
V1.1 qualification corpus      (aucune dépendance)          faible / nul
V1.2 balayage sur bande        (aucune dépendance)          faible / faible
V1.3 validation cap i2/i3      (V1.2)                       faible / moyen
V1.4 default-state ti=40       (aucune — Ghidra)            eleve  / moyen-eleve   <- chemin critique
V1.5 identite MPPWord32        (V1.4)                       moyen  / moyen
V2   spawns / cooldowns        (V1.4 + V1.5)                moyen  / moyen
--   rejeu oracle geometrique  (V1.2)                       faible / faible        <- a lancer tot
```

V1.1 à V1.3 sont indépendantes de V1.4 : elles livrent déjà la **détection + position + état
borné** sans aucune rétro-ingénierie. **Si le budget est contraint, V1.1-V1.3 + le rejeu de
l'oracle géométrique constituent un lot livrable à eux seuls.**

---

## 6. DÉCOUVERTES HORS PÉRIMÈTRE — notées, non traitées

1. **`ScanFilmWorldObjects` accepte n'importe quel `typeIndex`** sans vérifier que l'archétype
   porte bien `object-position-component`. C'est ce qui a permis l'erreur du § 2.1 de passer
   inaperçue pendant deux semaines. Un garde-fou (lire le nom d'`i0` dans le registre et refuser
   les archétypes *dynamic-precision*) tient en cinq lignes. **Non traité** : hors périmètre V0.
2. **Six empreintes de registre distinctes dans le cache** (§ 1.5), là où
   `registry_fingerprint.go:50-56` n'en documente que deux (`000d5950`/`64e8adfa` et
   `06dfe6d9`). L'alerte se déclenche à bon droit ; le commentaire est à jour dans son principe
   mais sous-estime l'ampleur. **Non traité.**
3. **`attCartes` est un fixture manuel** qui ne couvre pas le corpus véhicules alors que
   `match_registry.map_name` porte l'information. Dette d'outillage de test. **Non traité.**
4. **`ti=40` sur des cartes sans véhicule** (Illusion, Live Fire, Goliath, Cliffside, The Pit,
   Empyrean, Snowbound, Starboard, Command) : entités de classe véhicule statiques — tourelles
   montées ? liaisons du marcheur ? La question mérite une mesure propre ; elle intéresse
   directement le volet **tourelles** du chantier. **Non traitée ici.**
5. **`i34 vehicle-type-physics` absent de `8a049c50` et présent à 18-31 % ailleurs** (§ 1.4) :
   composant conditionnel non expliqué. **Non traité.**
6. **89 entrées `VEHICULE` du `damagetag` toutes sans `Name`**, donc 100 % affichées en
   « Autres » côté produit. `GUIDE_KILLSOURCE.md` § 6ter le qualifie de « seul livrable immédiat
   du sujet véhicules » et l'estime à « une table de 89 lignes, pas de la rétro-ingénierie ».
   **Hors périmètre de ce cadrage, mais c'est un gain produit à coût quasi nul.**

---

## 7. INSTRUMENTS ÉCRITS — jetables, sous garde d'environnement

Deux fichiers de test, **aucune ligne de code de production modifiée**, aucun commit. Les deux
sont en LECTURE SEULE (aucune base ouverte ; seul le balayage du cache écrit un CSV, et
uniquement dans le chemin qu'on lui donne). **À supprimer à la clôture du lot V0.**

### 7.1 `apps/go-api/internal/analysis/replay/vehicules_v0_cadrage_test.go`

| test | garde | ce qu'il mesure |
|---|---|---|
| `TestV0CadrageBalayageCache` | `V0_BALAYAGE` + `V0_SORTIE` | présence de `ti=40` sur les 951 films du cache, relevé CSV |
| `TestV0CadrageRecensement` | `V0_FILMS` | vies `ti=40` recensées aux images-clés, durées bornées |
| `TestV0CadrageNuageDelta` | `V0_FILMS` + `V0_DELTA` | vies décodées par le chemin objet du monde + témoin fantôme |
| `TestV0CadrageGrammaireI0` | `V0_FILMS` | **la comparaison des deux grammaires d'`i0`** + cap/vélocité/vitalité + témoin fantôme |

Garde commune : `ATT_FILM` (racine du cache film). `V0_FILMS` = `short8:carte,...`, le nom de
carte étant celui de `map_quant_bounds.json`.

```
CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
  V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
  go test ./internal/analysis/replay/ -run TestV0Cadrage -v -timeout 60m
```

### 7.2 `apps/go-api/internal/analysis/filmdec/vehicules_v0_composants_test.go`

| test | garde | ce qu'il mesure |
|---|---|---|
| `TestV0ComposantsRegistre` | `V0_CHUNK_DIRS` (+ `V0_DETAIL`) | empreinte du registre du film et liste des composants de `ti=40` |
| `TestV0ComposantsFlux` | `V0_CHUNK_DIRS` | histogramme des composants présents dans les records delta `ti=40` |

```
CGO_ENABLED=0 V0_CHUNK_DIRS=<cache>/film_chunks/0d76e8f1 \
  go test ./internal/analysis/filmdec/ -run TestV0Composants -v -timeout 60m
```

### 7.3 Fichiers temporaires (scratchpad, hors dépôt)

- `films_cache.csv` — les 951 identifiants courts du cache, pour la jointure DuckDB.
- `ti40_cache.csv` — le relevé `short8, slots40, slots_bipede, archetypes` des 951 films.

Toutes les requêtes DuckDB ont été passées en `ATTACH ... (READ_ONLY)`. Aucune base n'a été
ouverte en écriture.
