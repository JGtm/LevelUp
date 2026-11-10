# Divergence de lecture contre le temoin : dix films

## Films

| film | build | compares | fermes nous | fermes temoin | fermes les deux | en desaccord | nous seuls | temoin seul | appariement KO | verdicts differents |
|---|---|---|---|---|---|---|---|---|---|---|
| 03b7bbb6 | HI_1_13_0 | 26339 | 25859 | 26339 | 25859 | 48 | 0 | 480 | 0 | 0 |
| 221dd082 | HI_1_12_0 | 33244 | 30507 | 33241 | 30505 | 48 | 2 | 2736 | 0 | 0 |
| 090139ed | HI_1_13_0 | 31266 | 31196 | 31266 | 31196 | 8 | 0 | 70 | 0 | 0 |
| 77e41773 | HI_1_12_0 | 34584 | 33844 | 34584 | 33844 | 29 | 0 | 740 | 0 | 0 |
| 59680177 | HI_1_13_0 | 37735 | 33354 | 37734 | 33353 | 93 | 1 | 4381 | 0 | 0 |
| 6bbad606 | HI_1_13_0 | 45213 | 45000 | 45213 | 45000 | 38 | 0 | 213 | 0 | 0 |
| 29a3ef4c | HI_1_13_0 | 42017 | 37692 | 42016 | 37692 | 101 | 0 | 4324 | 0 | 0 |
| 3f0a1fcb | HI_1_13_0 | 40978 | 40540 | 40615 | 40539 | 12 | 1 | 76 | 0 | 0 |
| 672641d0 | HI_1_13_0 | 77395 | 72292 | 77374 | 72279 | 107 | 13 | 5095 | 0 | 0 |
| 8b1f15cb | HI_1_13_0 | 45941 | 38625 | 45467 | 38379 | 278 | 246 | 7088 | 0 | 0 |

## Decalages de frontiere entre composants rattrapes au record suivant (tous les paquets)

- 134 : B : ti=40 composant 0 (update)
- 90 : B : ti=35 composant 57 (update)

## Controle de l'instrument

- echantillon : 100 paquets tires au hasard (graine 20261110) parmi les 388569 fermes par les deux lectures sans reprise du temoin : 0 desaccords de position
- population entiere de ces paquets : 685 desaccords sur 388569

## Tous builds : premiers desaccords par lecteur en cause

### Cause d'arret : vue B : sortie par rejet

Total : 18385 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 9077 | 49.4 % | B | B : sortie par rejet (hors datum) |
| 2 | 6953 | 37.8 % | B | B : ti=40 composant 0 (update) |
| 3 | 810 | 4.4 % | B | B : type de l'entite (update) |
| 4 | 534 | 2.9 % | B | B : ti=35 composant 57 (update) |
| 5 | 186 | 1.0 % | B | B : ti=21 composant 16 (update) / nous rejetons un en-tete ou le temoin lit le terminateur |
| 6 | 152 | 0.8 % | B | B : ti=21 composant 16 (update) |
| 7 | 127 | 0.7 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 8 | 106 | 0.6 % | B | B : ti=41 composant 0 (update) |
| 9 | 77 | 0.4 % | B | B : le temoin s'arrete ou reprend (arret aucun) |
| 10 | 68 | 0.4 % | A | A : largeur du message genre 5 |
| 11 | 58 | 0.3 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 12 | 48 | 0.3 % | A | A : largeur du message genre 82 |
| 13 | 45 | 0.2 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 14 | 30 | 0.2 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |
| 15 | 19 | 0.1 % | B | B : ti=21 composant 2 (create) |

### Cause d'arret : vue C : terminateur hors cadre

Total : 3580 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 2852 | 79.7 % | B | B : ti=40 composant 0 (update) |
| 2 | 433 | 12.1 % | B | B : type de l'entite (update) |
| 3 | 66 | 1.8 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 4 | 60 | 1.7 % | B | B : ti=21 composant 16 (update) |
| 5 | 46 | 1.3 % | B | B : ti=35 composant 57 (update) |
| 6 | 29 | 0.8 % | B | B : ti=21 composant 16 (update) / sortie 1 |
| 7 | 22 | 0.6 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 8 | 15 | 0.4 % | B | B : ti=13 etat de create / sortie 1 |
| 9 | 11 | 0.3 % | B | B : ti=41 composant 0 (update) |
| 10 | 9 | 0.3 % | A | A : largeur du message genre 5 |
| 11 | 8 | 0.2 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 12 | 6 | 0.2 % | B | B : ti=37 composant 0 (update) |
| 13 | 5 | 0.1 % | B | B : ti=47 etat de create / sortie 1 |
| 14 | 4 | 0.1 % | A | A : largeur du message genre 82 |
| 15 | 3 | 0.1 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |

### Cause d'arret : liste d evenements non localisee

Total : 838 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 838 | 100.0 % | A | A : liste d'evenements non localisee |

### Cause d'arret : vue C : kind non porte

Total : 1036 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 633 | 61.1 % | B | B : ti=40 composant 0 (update) |
| 2 | 254 | 24.5 % | B | B : type de l'entite (update) |
| 3 | 65 | 6.3 % | B | B : ti=21 composant 16 (update) |
| 4 | 38 | 3.7 % | B | B : ti=21 composant 16 (update) / sortie 1 |
| 5 | 15 | 1.4 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 6 | 10 | 1.0 % | B | B : ti=35 composant 57 (update) |
| 7 | 6 | 0.6 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 8 | 4 | 0.4 % | B | B : ti=41 composant 0 (update) |
| 9 | 3 | 0.3 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 10 | 2 | 0.2 % | A | A : largeur du message genre 5 |
| 11 | 2 | 0.2 % | B | B : ti=47 etat de create / sortie 1 |
| 12 | 1 | 0.1 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |
| 13 | 1 | 0.1 % | A | A : le temoin ne termine pas la vue des evenements |
| 14 | 1 | 0.1 % | B | B : le temoin s'arrete ou reprend (arret aucun) |
| 15 | 1 | 0.1 % | B | B : ti=37 composant 0 (update) |

### Autres causes d'arret

- 698 : vue B : fin de payload
- 153 : vue C : bloc 0xbc (desalignement)
- 108 : ti=56 archetype hors registre
- 56 : ecrivain : ordre de la vue B
- 54 : ti=59 archetype hors registre
- 50 : ecrivain : masque au-dela de l archetype
- 47 : ti=0 i18 forge-engine-player-roles-component
- 45 : ti=51 archetype hors registre
- 36 : ti=48 i0 forge-player-data-edited-objects-ids-component
- 34 : ti=50 archetype hors registre
- 33 : ti=52 archetype hors registre
- 32 : ti=55 archetype hors registre
- 25 : ti=53 archetype hors registre
- 25 : ti=54 archetype hors registre
- 24 : ti=61 archetype hors registre
- 21 : ti=24 i0 state-checksum-component
- 19 : ti=57 archetype hors registre
- 19 : vue C : debordement
- 18 : ti=0 i20 forge-engine-map-properties-component
- 18 : ti=63 archetype hors registre
- 16 : ti=49 i0 managed-radialmenu-title-component
- 16 : ti=58 archetype hors registre
- 16 : ti=60 archetype hors registre
- 15 : ti=0 i19 forge-engine-total-budget-component
- 15 : ti=62 archetype hors registre
- 14 : ti=0 i21 forge-engine-forge-mode-component
- 11 : ti=25 i0 powerframe-player-selection-data-component
- 10 : ti=24 i1 persistence-state-bucket-component
- 10 : ti=36 archetype hors registre
- 10 : ti=39 archetype hors registre
- 10 : ti=42 archetype hors registre
- 9 : ti=18 i2 effect-state-data-component
- 9 : ti=40 archetype hors registre
- 8 : ti=26 i0 supply-lines-blocked-status-component
- 8 : ti=43 archetype hors registre
- 8 : ti=7 i0 forge-sim-generic-component
- 7 : ti=0 i22 forge-engine-monitor-noclip-component
- 7 : ti=14 i1 crew-marked-objects-component
- 7 : ti=15 i2 nav-cutscene-flag-component
- 7 : ti=34 i10 tacmap-mapdismissallock
- 7 : ti=38 archetype hors registre
- 6 : ti=34 i15 tacmap-tooltip-state
- 6 : ti=41 archetype hors registre
- 6 : ti=46 i0 state-broker-state-changed-data-component
- 5 : ti=15 i1 nav-cutscene-flag-component
- 5 : ti=15 i3 nav-cutscene-flag-component
- 5 : ti=23 i1 selectable-zone-data-component
- 5 : ti=27 i1 supply-lines-item-unlocked-component
- 5 : ti=5 i10 player-primary-respawn-object-component
- 5 : ti=5 i7 player-desired-respawn-seat-component
- 4 : ti=18 i0 effect-state-data-component
- 4 : ti=19 i0 sound-placement-state-data-component
- 4 : ti=19 i3 sound-placement-state-data-component
- 4 : ti=23 i0 selectable-zone-data-component
- 4 : ti=24 i3 persistence-state-bucket-component
- 4 : ti=31 i0 tacmap-category
- 4 : ti=31 i1 tacmap-mission
- 4 : ti=34 i14 tacmap-playercontrol
- 4 : ti=46 i1 state-broker-state-changed-data-component
- 4 : ti=46 i2 state-broker-state-changed-data-component
- 4 : ti=49 i2 managed-radialmenu-title-component
- 3 : ti=12 i19 managed-navpoint-visual-states-component
- 3 : ti=18 i15 effect-state-data-component
- 3 : ti=24 i2 persistence-state-bucket-component
- 3 : ti=26 i1 supply-lines-blocked-status-component
- 3 : ti=26 i2 supply-lines-blocked-status-component
- 3 : ti=27 i0 supply-lines-item-unlocked-component
- 3 : ti=34 i12 tacmap-backmenulocked-state
- 3 : ti=37 archetype hors registre
- 3 : ti=46 i4 state-broker-state-changed-data-component
- 3 : ti=49 i1 managed-radialmenu-title-component
- 3 : ti=49 i3 managed-radialmenu-title-component
- 3 : ti=7 i1 forge-sim-generic-component
- 3 : ti=7 i3 forge-sim-generic-component
- 2 : ecrivain : masque epars non croissant
- 2 : ti=0 i24 forge-engine-baking-data-component
- 2 : ti=0 i25 forge-engine-baking-data-component
- 2 : ti=18 i1 effect-state-data-component
- 2 : ti=18 i10 effect-state-data-component
- 2 : ti=18 i5 effect-state-data-component
- 2 : ti=20 i2 spawn-filter-filters-component
- 2 : ti=23 i3 selectable-zone-data-component
- 2 : ti=24 i10 persistence-state-bucket-component
- 2 : ti=24 i4 persistence-state-bucket-component
- 2 : ti=24 i8 persistence-state-bucket-component
- 2 : ti=26 i3 supply-lines-blocked-status-component
- 2 : ti=27 i10 supply-lines-item-unlocked-component
- 2 : ti=28 i0 narrative-moment-beat-activation-component
- 2 : ti=31 i2 tacmap-visiblity
- 2 : ti=31 i4 tacmap-missionstate
- 2 : ti=46 i10 state-broker-state-changed-data-component
- 2 : ti=46 i6 state-broker-state-changed-data-component
- 2 : ti=49 i24 managed-radialmenu-enabled-component
- 2 : ti=49 i7 managed-radialmenu-title-component
- 2 : ti=49 i9 managed-radialmenu-description-component
- 2 : ti=7 i2 forge-sim-generic-component
- 2 : ti=7 i4 forge-sim-generic-component
- 2 : ti=7 i9 forge-sim-generic-component
- 1 : ecrivain : vue C en-tete cdc04 pose
- 1 : ecrivain : vue C index non croissants
- 1 : ti=0 i26 forge-engine-baking-data-component
- 1 : ti=12 i2 managed-navpoint-visibility-distance-filters-component
- 1 : ti=18 i12 effect-state-data-component
- 1 : ti=18 i14 effect-state-data-component
- 1 : ti=18 i16 effect-state-data-component
- 1 : ti=18 i20 effect-state-data-component
- 1 : ti=19 i1 sound-placement-state-data-component
- 1 : ti=19 i14 sound-placement-state-data-component
- 1 : ti=19 i2 sound-placement-state-data-component
- 1 : ti=19 i5 sound-placement-state-data-component
- 1 : ti=23 i19 selectable-zone-data-component
- 1 : ti=23 i2 selectable-zone-data-component
- 1 : ti=23 i4 selectable-zone-data-component
- 1 : ti=23 i5 selectable-zone-data-component
- 1 : ti=23 i8 selectable-zone-data-component
- 1 : ti=24 i11 persistence-state-bucket-component
- 1 : ti=24 i15 persistence-state-bucket-component
- 1 : ti=24 i6 persistence-state-bucket-component
- 1 : ti=26 i12 supply-lines-blocked-status-component
- 1 : ti=26 i4 supply-lines-blocked-status-component
- 1 : ti=26 i5 supply-lines-blocked-status-component
- 1 : ti=27 i18 supply-lines-item-unlocked-component
- 1 : ti=27 i19 supply-lines-item-unlocked-component
- 1 : ti=27 i2 supply-lines-item-unlocked-component
- 1 : ti=27 i22 supply-lines-item-unlocked-component
- 1 : ti=27 i34 supply-lines-item-unlocked-component
- 1 : ti=27 i46 supply-lines-item-unlocked-component
- 1 : ti=27 i5 supply-lines-item-unlocked-component
- 1 : ti=27 i7 supply-lines-item-unlocked-component
- 1 : ti=31 i10 tacmap-missioninfo
- 1 : ti=31 i3 tacmap-rawlocation
- 1 : ti=31 i5 tacmap-description
- 1 : ti=31 i9 tacmap-collectibles
- 1 : ti=46 i14 state-broker-state-changed-data-component
- 1 : ti=46 i29 state-broker-state-changed-data-component
- 1 : ti=46 i3 state-broker-state-changed-data-component
- 1 : ti=46 i49 state-broker-state-changed-data-component
- 1 : ti=46 i5 state-broker-state-changed-data-component
- 1 : ti=48 i1 forge-player-data-editing-graphs-component
- 1 : ti=49 i11 managed-radialmenu-description-component
- 1 : ti=49 i12 managed-radialmenu-description-component
- 1 : ti=49 i14 managed-radialmenu-description-component
- 1 : ti=49 i17 managed-radialmenu-iconindex-component
- 1 : ti=49 i21 managed-radialmenu-iconindex-component
- 1 : ti=49 i34 managed-radialmenu-identifier-component
- 1 : ti=49 i6 managed-radialmenu-title-component
- 1 : ti=7 i25 forge-sim-generic-component
- 1 : ti=7 i32 forge-sim-generic-component

## Build HI_1_12_0 : premiers desaccords par lecteur en cause

### Cause d'arret : vue B : sortie par rejet

Total : 3375 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 3264 | 96.7 % | B | B : sortie par rejet (hors datum) |
| 2 | 46 | 1.4 % | A | A : largeur du message genre 82 |
| 3 | 18 | 0.5 % | A | A : largeur du message genre 36 |
| 4 | 11 | 0.3 % | A | A : largeur du message genre 0 |
| 5 | 9 | 0.3 % | B | B : ti=35 composant 57 (update) |
| 6 | 8 | 0.2 % | A | A : largeur du message genre 5 |
| 7 | 5 | 0.1 % | A | A : largeur du message genre 21 |
| 8 | 5 | 0.1 % | A | A : largeur du message genre 9 |
| 9 | 3 | 0.1 % | A | A : largeur du message genre 15 |
| 10 | 2 | 0.1 % | A | A : largeur du message genre 6 |
| 11 | 2 | 0.1 % | B | B : ti=22 composant 0 (create) |
| 12 | 1 | 0.0 % | A | A : largeur du message genre 39 |
| 13 | 1 | 0.0 % | A | A : le temoin ne termine pas la vue des evenements |

### Cause d'arret : vue C : terminateur hors cadre

Total : 0 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|

### Cause d'arret : liste d evenements non localisee

Total : 89 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 89 | 100.0 % | A | A : liste d'evenements non localisee |

### Cause d'arret : vue C : kind non porte

Total : 0 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|

### Autres causes d'arret

- 9 : ecrivain : masque au-dela de l archetype
- 2 : ecrivain : ordre de la vue B
- 2 : vue C : debordement

## Build HI_1_13_0 : premiers desaccords par lecteur en cause

### Cause d'arret : vue B : sortie par rejet

Total : 15010 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 6953 | 46.3 % | B | B : ti=40 composant 0 (update) |
| 2 | 5813 | 38.7 % | B | B : sortie par rejet (hors datum) |
| 3 | 810 | 5.4 % | B | B : type de l'entite (update) |
| 4 | 525 | 3.5 % | B | B : ti=35 composant 57 (update) |
| 5 | 186 | 1.2 % | B | B : ti=21 composant 16 (update) / nous rejetons un en-tete ou le temoin lit le terminateur |
| 6 | 152 | 1.0 % | B | B : ti=21 composant 16 (update) |
| 7 | 127 | 0.8 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 8 | 106 | 0.7 % | B | B : ti=41 composant 0 (update) |
| 9 | 77 | 0.5 % | B | B : le temoin s'arrete ou reprend (arret aucun) |
| 10 | 60 | 0.4 % | A | A : largeur du message genre 5 |
| 11 | 58 | 0.4 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 12 | 45 | 0.3 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 13 | 30 | 0.2 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |
| 14 | 19 | 0.1 % | B | B : ti=21 composant 2 (create) |
| 15 | 12 | 0.1 % | B | B : ti=37 composant 0 (update) |

### Cause d'arret : vue C : terminateur hors cadre

Total : 3580 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 2852 | 79.7 % | B | B : ti=40 composant 0 (update) |
| 2 | 433 | 12.1 % | B | B : type de l'entite (update) |
| 3 | 66 | 1.8 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 4 | 60 | 1.7 % | B | B : ti=21 composant 16 (update) |
| 5 | 46 | 1.3 % | B | B : ti=35 composant 57 (update) |
| 6 | 29 | 0.8 % | B | B : ti=21 composant 16 (update) / sortie 1 |
| 7 | 22 | 0.6 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 8 | 15 | 0.4 % | B | B : ti=13 etat de create / sortie 1 |
| 9 | 11 | 0.3 % | B | B : ti=41 composant 0 (update) |
| 10 | 9 | 0.3 % | A | A : largeur du message genre 5 |
| 11 | 8 | 0.2 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 12 | 6 | 0.2 % | B | B : ti=37 composant 0 (update) |
| 13 | 5 | 0.1 % | B | B : ti=47 etat de create / sortie 1 |
| 14 | 4 | 0.1 % | A | A : largeur du message genre 82 |
| 15 | 3 | 0.1 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |

### Cause d'arret : liste d evenements non localisee

Total : 749 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 749 | 100.0 % | A | A : liste d'evenements non localisee |

### Cause d'arret : vue C : kind non porte

Total : 1036 paquets

| rang | paquets | part | vue | lecteur en cause |
|---|---|---|---|---|
| 1 | 633 | 61.1 % | B | B : ti=40 composant 0 (update) |
| 2 | 254 | 24.5 % | B | B : type de l'entite (update) |
| 3 | 65 | 6.3 % | B | B : ti=21 composant 16 (update) |
| 4 | 38 | 3.7 % | B | B : ti=21 composant 16 (update) / sortie 1 |
| 5 | 15 | 1.4 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (Component(35, 0)) |
| 6 | 10 | 1.0 % | B | B : ti=35 composant 57 (update) |
| 7 | 6 | 0.6 % | B | B : le temoin s'arrete ou reprend (arret Component(35, 0)) |
| 8 | 4 | 0.4 % | B | B : ti=41 composant 0 (update) |
| 9 | 3 | 0.3 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (UnknownEntity) |
| 10 | 2 | 0.2 % | A | A : largeur du message genre 5 |
| 11 | 2 | 0.2 % | B | B : ti=47 etat de create / sortie 1 |
| 12 | 1 | 0.1 % | - | aucun desaccord de position, le temoin ne ferme pas non plus (BadHeader) |
| 13 | 1 | 0.1 % | A | A : le temoin ne termine pas la vue des evenements |
| 14 | 1 | 0.1 % | B | B : le temoin s'arrete ou reprend (arret aucun) |
| 15 | 1 | 0.1 % | B | B : ti=37 composant 0 (update) |

### Autres causes d'arret

- 698 : vue B : fin de payload
- 153 : vue C : bloc 0xbc (desalignement)
- 108 : ti=56 archetype hors registre
- 54 : ecrivain : ordre de la vue B
- 54 : ti=59 archetype hors registre
- 47 : ti=0 i18 forge-engine-player-roles-component
- 45 : ti=51 archetype hors registre
- 41 : ecrivain : masque au-dela de l archetype
- 36 : ti=48 i0 forge-player-data-edited-objects-ids-component
- 34 : ti=50 archetype hors registre
- 33 : ti=52 archetype hors registre
- 32 : ti=55 archetype hors registre
- 25 : ti=53 archetype hors registre
- 25 : ti=54 archetype hors registre
- 24 : ti=61 archetype hors registre
- 21 : ti=24 i0 state-checksum-component
- 19 : ti=57 archetype hors registre
- 18 : ti=0 i20 forge-engine-map-properties-component
- 18 : ti=63 archetype hors registre
- 17 : vue C : debordement
- 16 : ti=49 i0 managed-radialmenu-title-component
- 16 : ti=58 archetype hors registre
- 16 : ti=60 archetype hors registre
- 15 : ti=0 i19 forge-engine-total-budget-component
- 15 : ti=62 archetype hors registre
- 14 : ti=0 i21 forge-engine-forge-mode-component
- 11 : ti=25 i0 powerframe-player-selection-data-component
- 10 : ti=24 i1 persistence-state-bucket-component
- 10 : ti=36 archetype hors registre
- 10 : ti=39 archetype hors registre
- 10 : ti=42 archetype hors registre
- 9 : ti=18 i2 effect-state-data-component
- 9 : ti=40 archetype hors registre
- 8 : ti=26 i0 supply-lines-blocked-status-component
- 8 : ti=43 archetype hors registre
- 8 : ti=7 i0 forge-sim-generic-component
- 7 : ti=0 i22 forge-engine-monitor-noclip-component
- 7 : ti=14 i1 crew-marked-objects-component
- 7 : ti=15 i2 nav-cutscene-flag-component
- 7 : ti=34 i10 tacmap-mapdismissallock
- 7 : ti=38 archetype hors registre
- 6 : ti=34 i15 tacmap-tooltip-state
- 6 : ti=41 archetype hors registre
- 6 : ti=46 i0 state-broker-state-changed-data-component
- 5 : ti=15 i1 nav-cutscene-flag-component
- 5 : ti=15 i3 nav-cutscene-flag-component
- 5 : ti=23 i1 selectable-zone-data-component
- 5 : ti=27 i1 supply-lines-item-unlocked-component
- 5 : ti=5 i10 player-primary-respawn-object-component
- 5 : ti=5 i7 player-desired-respawn-seat-component
- 4 : ti=18 i0 effect-state-data-component
- 4 : ti=19 i0 sound-placement-state-data-component
- 4 : ti=19 i3 sound-placement-state-data-component
- 4 : ti=23 i0 selectable-zone-data-component
- 4 : ti=24 i3 persistence-state-bucket-component
- 4 : ti=31 i0 tacmap-category
- 4 : ti=31 i1 tacmap-mission
- 4 : ti=34 i14 tacmap-playercontrol
- 4 : ti=46 i1 state-broker-state-changed-data-component
- 4 : ti=46 i2 state-broker-state-changed-data-component
- 4 : ti=49 i2 managed-radialmenu-title-component
- 3 : ti=12 i19 managed-navpoint-visual-states-component
- 3 : ti=18 i15 effect-state-data-component
- 3 : ti=24 i2 persistence-state-bucket-component
- 3 : ti=26 i1 supply-lines-blocked-status-component
- 3 : ti=26 i2 supply-lines-blocked-status-component
- 3 : ti=27 i0 supply-lines-item-unlocked-component
- 3 : ti=34 i12 tacmap-backmenulocked-state
- 3 : ti=37 archetype hors registre
- 3 : ti=46 i4 state-broker-state-changed-data-component
- 3 : ti=49 i1 managed-radialmenu-title-component
- 3 : ti=49 i3 managed-radialmenu-title-component
- 3 : ti=7 i1 forge-sim-generic-component
- 3 : ti=7 i3 forge-sim-generic-component
- 2 : ecrivain : masque epars non croissant
- 2 : ti=0 i24 forge-engine-baking-data-component
- 2 : ti=0 i25 forge-engine-baking-data-component
- 2 : ti=18 i1 effect-state-data-component
- 2 : ti=18 i10 effect-state-data-component
- 2 : ti=18 i5 effect-state-data-component
- 2 : ti=20 i2 spawn-filter-filters-component
- 2 : ti=23 i3 selectable-zone-data-component
- 2 : ti=24 i10 persistence-state-bucket-component
- 2 : ti=24 i4 persistence-state-bucket-component
- 2 : ti=24 i8 persistence-state-bucket-component
- 2 : ti=26 i3 supply-lines-blocked-status-component
- 2 : ti=27 i10 supply-lines-item-unlocked-component
- 2 : ti=28 i0 narrative-moment-beat-activation-component
- 2 : ti=31 i2 tacmap-visiblity
- 2 : ti=31 i4 tacmap-missionstate
- 2 : ti=46 i10 state-broker-state-changed-data-component
- 2 : ti=46 i6 state-broker-state-changed-data-component
- 2 : ti=49 i24 managed-radialmenu-enabled-component
- 2 : ti=49 i7 managed-radialmenu-title-component
- 2 : ti=49 i9 managed-radialmenu-description-component
- 2 : ti=7 i2 forge-sim-generic-component
- 2 : ti=7 i4 forge-sim-generic-component
- 2 : ti=7 i9 forge-sim-generic-component
- 1 : ecrivain : vue C en-tete cdc04 pose
- 1 : ecrivain : vue C index non croissants
- 1 : ti=0 i26 forge-engine-baking-data-component
- 1 : ti=12 i2 managed-navpoint-visibility-distance-filters-component
- 1 : ti=18 i12 effect-state-data-component
- 1 : ti=18 i14 effect-state-data-component
- 1 : ti=18 i16 effect-state-data-component
- 1 : ti=18 i20 effect-state-data-component
- 1 : ti=19 i1 sound-placement-state-data-component
- 1 : ti=19 i14 sound-placement-state-data-component
- 1 : ti=19 i2 sound-placement-state-data-component
- 1 : ti=19 i5 sound-placement-state-data-component
- 1 : ti=23 i19 selectable-zone-data-component
- 1 : ti=23 i2 selectable-zone-data-component
- 1 : ti=23 i4 selectable-zone-data-component
- 1 : ti=23 i5 selectable-zone-data-component
- 1 : ti=23 i8 selectable-zone-data-component
- 1 : ti=24 i11 persistence-state-bucket-component
- 1 : ti=24 i15 persistence-state-bucket-component
- 1 : ti=24 i6 persistence-state-bucket-component
- 1 : ti=26 i12 supply-lines-blocked-status-component
- 1 : ti=26 i4 supply-lines-blocked-status-component
- 1 : ti=26 i5 supply-lines-blocked-status-component
- 1 : ti=27 i18 supply-lines-item-unlocked-component
- 1 : ti=27 i19 supply-lines-item-unlocked-component
- 1 : ti=27 i2 supply-lines-item-unlocked-component
- 1 : ti=27 i22 supply-lines-item-unlocked-component
- 1 : ti=27 i34 supply-lines-item-unlocked-component
- 1 : ti=27 i46 supply-lines-item-unlocked-component
- 1 : ti=27 i5 supply-lines-item-unlocked-component
- 1 : ti=27 i7 supply-lines-item-unlocked-component
- 1 : ti=31 i10 tacmap-missioninfo
- 1 : ti=31 i3 tacmap-rawlocation
- 1 : ti=31 i5 tacmap-description
- 1 : ti=31 i9 tacmap-collectibles
- 1 : ti=46 i14 state-broker-state-changed-data-component
- 1 : ti=46 i29 state-broker-state-changed-data-component
- 1 : ti=46 i3 state-broker-state-changed-data-component
- 1 : ti=46 i49 state-broker-state-changed-data-component
- 1 : ti=46 i5 state-broker-state-changed-data-component
- 1 : ti=48 i1 forge-player-data-editing-graphs-component
- 1 : ti=49 i11 managed-radialmenu-description-component
- 1 : ti=49 i12 managed-radialmenu-description-component
- 1 : ti=49 i14 managed-radialmenu-description-component
- 1 : ti=49 i17 managed-radialmenu-iconindex-component
- 1 : ti=49 i21 managed-radialmenu-iconindex-component
- 1 : ti=49 i34 managed-radialmenu-identifier-component
- 1 : ti=49 i6 managed-radialmenu-title-component
- 1 : ti=7 i25 forge-sim-generic-component
- 1 : ti=7 i32 forge-sim-generic-component
