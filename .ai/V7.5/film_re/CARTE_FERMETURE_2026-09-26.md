# Carte de fermeture — mesure de référence (J4.0.5, 2026-09-26)

> Plan : `.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, lot J4.0.5. Instrument :
> `film/research/cmd_fermeture` (tag `research`), bâti sur la branche `feat/suite-audit-decodeur`
> à `371008278` (J1, J2, J3, J4.0). Base des deltas de J4.6, J5, J6, J10 et J11.3.

## Protocole

- 18 témoins de `config/replay_corpus.toml` (tous sauf `f75e7053`, absent du cache de ce poste),
  au moins un film par build, BTB compris (décodés le même jour par le G-equiv à 1,2 Gio au plus).
- Films lus un à un, en place, depuis `data/cache/film_chunks` du checkout principal (lecture
  seule), sentinelle `filmproc.Arm` à 4 Gio par film ; rapport écrit hors de `data/`.
- Contexte d'instrument : largeurs d'axe lues dans le film, profil par défaut, sans la
  calibration killsource de la cuisson.
- Commande (depuis `apps/go-api`) :
  `cmd_fermeture -racine <principal>/data/cache/film_chunks -films bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,0797ce72,60ae07c4,bfecd02b,396cfc92,111fa685,e5adf7b2,11de8353,a521164d,a349fea8,50247b26,4f77afc1,084a804d -sortie <hors data> -top 40`
- Résultat : 18 films mesurés, 0 échec, pics mémoire 55 à 215 Mio, 15 à 43 s par film.

## Lecture

- **Aucun build n'atteint le déclencheur** de la représentation intermédiaire (>= 95 % des records
  utiles fermés). Le meilleur est HI_1_13_0 (9 films) : 65,5 % des paquets, 79,8 % des records utiles.
- **Les anciennes versions ne ferment presque rien** : version-31 (1,0 %), version-33 (1,6 %),
  HI_1_4_1 (6,3 %), avec 0,0 à 0,6 % des records utiles.
- **La première cause d'arrêt, de très loin, n'est pas un composant** : « vue C : terminateur hors
  cadre » arrête 214 536 paquets (borne supérieure du gain : 2 171 370 records utiles), sur tous les
  builds. Viennent ensuite « liste d'événements non localisée » (35 422 paquets, gain nul) et un
  composant d'animation non porté sans usage produit (`ti=43 device-animation-layer-state`,
  12 113 paquets, HI_1_12_0 et HI_1_8_0). Les composants non portés suivants pèsent chacun moins de
  5 000 paquets.
- Conséquence pour la suite : le levier n° 1 de la fermeture est la fin de la vue de contrôle
  (vue C), pas le portage de composants ; c'est l'entrée naturelle d'un jalon de grammaire (J6/J10)
  ou du chantier de représentation intermédiaire, à statuer à leur entrée — rien n'est traité ici.

## Résumé brut de l'instrument

Films mesures : 18 ; echecs : 0. Table ECS : `internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv`. Contexte d instrument (largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).

## Par build

| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |
|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 2 | 9385/49819 (18,8 %) | 6370/24719 (25,8 %) | 9385/44511 (21,1 %) | 9385/43709 (21,5 %) | 112673/693285 (16,3 %) | 146918 | non atteint | 215 Mio |
| HI_1_11_0 | 1 | 4285/16824 (25,5 %) | 2959/9073 (32,6 %) | 4285/15097 (28,4 %) | 4285/14831 (28,9 %) | 65967/246477 (26,8 %) | 76224 | non atteint | 96 Mio |
| HI_1_12_0 | 1 | 5835/21864 (26,7 %) | 5201/19134 (27,2 %) | 5835/21581 (27,0 %) | 5835/9156 (63,7 %) | 31993/117732 (27,2 %) | 30137 | non atteint | 55 Mio |
| HI_1_13_0 | 9 | 201394/307458 (65,5 %) | 170201/246824 (69,0 %) | 201394/291845 (69,0 %) | 201394/285780 (70,5 %) | 1601587/2007507 (79,8 %) | 1479612 | non atteint | 168 Mio |
| HI_1_4_1 | 1 | 701/11130 (6,3 %) | 656/6174 (10,6 %) | 701/9839 (7,1 %) | 701/9497 (7,4 %) | 1/141879 (0,0 %) | 5 | non atteint | 81 Mio |
| HI_1_8_0 | 1 | 13969/49696 (28,1 %) | 13054/43640 (29,9 %) | 13969/46436 (30,1 %) | 13969/45153 (30,9 %) | 78247/257296 (30,4 %) | 79294 | non atteint | 106 Mio |
| HI_1_9_0 | 1 | 5738/17629 (32,5 %) | 4404/10304 (42,7 %) | 5738/15987 (35,9 %) | 5738/15738 (36,5 %) | 57875/225121 (25,7 %) | 75327 | non atteint | 90 Mio |
| version-31 | 1 | 153/17919 (0,9 %) | 130/10140 (1,3 %) | 153/15662 (1,0 %) | 153/15382 (1,0 %) | 10/246018 (0,0 %) | 3 | non atteint | 160 Mio |
| version-33 | 1 | 463/28751 (1,6 %) | 399/14259 (2,8 %) | 463/24710 (1,9 %) | 463/23756 (1,9 %) | 2302/378095 (0,6 %) | 6 | non atteint | 176 Mio |

## Causes d arret (premiere cause de chaque paquet non ferme)

Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne sont pas lus du tout).

| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | Gain potentiel | Builds |
|---|---|---|---|---|---|---|---|---|
| 1 | vue C : terminateur hors cadre | - |  | - | - | 214536 | 2171370 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 2 | liste d evenements non localisee | - |  | - | - | 35422 | 0 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 3 | ti=43 device-animation-layer-state | 43 | i35 | non_porte | aucun | 12113 | 67207 | HI_1_12_0, HI_1_8_0 |
| 4 | vue C : kind non porte | - |  | - | - | 4849 | 53055 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 5 | ti=2 managed-engine-timers | 2 | i15 | non_porte | aucun | 3635 | 27 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 6 | vue C : bloc 0xbc non porte | - |  | - | - | 1549 | 17685 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 7 | ti=3 low-frequency | 3 | i0 | non_porte | aucun | 1187 | 129 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 8 | ti=40 vehicle-auto-turret-aiming-vector | 40 | i31 | non_porte | aucun | 807 | 4796 | HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 9 | ti=43 device-position-group | 43 | i21 | non_porte | aucun | 599 | 8447 | HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 10 | vue B : fin de payload | - |  | - | - | 595 | 2568 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 11 | ti=40 vehicle-weapon-set | 40 | i37, i38 | non_porte | aucun | 539 | 9789 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 12 | ti=12 managed-navpoint-visual-state-groups-component-1 | 12 | i21 | non_porte | aucun | 321 | 3636 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 13 | ti=12 managed-navpoint-visual-state-groups-component-2 | 12 | i22 | non_porte | aucun | 320 | 3924 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-31, version-33 |
| 14 | ti=10 managed-object-navpoint | 10 | i10, i11, i12, i15, i16, i17, i2, i3, i4, i5, i6, i7, i8 | non_porte | aucun | 307 | 3282 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 15 | ti=56 archetype hors registre | 56 |  | - | - | 231 | 1023 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 16 | ti=35 biped-spartan-ability-non-predicted-state | 35 | i58, i59 | partiel | rejeu : grappleLines[] (schema 8) — la ligne blanche du joueur vers son ancre, fenetre [t0,t1] par vie | 215 | 1807 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 17 | vue C : debordement | - |  | - | - | 145 | 237 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 18 | ti=11 managed-objective-interaction-filter | 11 | i4 | non_porte | aucun | 83 | 550 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0 |
| 19 | ti=40 vehicle-type-state | 40 | i32, i33 | non_porte | aucun | 83 | 1506 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_9_0, version-31 |
| 20 | ti=2 matchflow-isplaying-flags | 2 | i17 | non_porte | aucun | 70 | 13 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 21 | ti=2 game-engine-soft-ceilings | 2 | i11 | non_porte | aucun | 66 | 376 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 22 | ti=12 managed-navpoint-override-flags | 12 | i16 | non_porte | aucun | 64 | 318 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 23 | ti=18 effect-state-data | 18 | i0, i1, i10, i12, i13, i2, i22, i23, i28, i3, i31, i4, i6, i7, i8, i9 | partiel | aucun | 64 | 184 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31 |
| 24 | ti=43 device-machine-flags | 43 | i39 | non_porte | aucun | 63 | 743 | HI_1_12_0, HI_1_13_0, HI_1_4_1, version-31, version-33 |
| 25 | ti=12 managed-navpoint-visual-state-groups-component-0 | 12 | i20 | non_porte | aucun | 61 | 606 | HI_1_13_0, HI_1_4_1, version-33 |
| 26 | ti=40 air-drop-flight | 40 | i44, i45 | non_porte | aucun | 59 | 1142 | HI_1_11_0, HI_1_4_1, version-31, version-33 |
| 27 | ti=51 archetype hors registre | 51 |  | - | - | 56 | 420 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 28 | ti=10 managed-object-interaction-filter | 10 | i22 | non_porte | aucun | 55 | 645 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-33 |
| 29 | ti=59 archetype hors registre | 59 |  | - | - | 52 | 325 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 30 | ti=40 vehicle-low-frequency | 40 | i46, i47 | non_porte | aucun | 48 | 889 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-31, version-33 |
| 31 | ti=2 GameEngineComposerLetterboxComponent | 2 | i14 | non_porte | aucun | 42 | 429 | HI_1_10_0, HI_1_12_0, HI_1_8_0, HI_1_9_0 |
| 32 | ti=50 archetype hors registre | 50 |  | - | - | 41 | 161 | HI_1_10_0, HI_1_11_0, HI_1_12_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, version-33 |
| 33 | ti=19 sound-placement-state-data | 19 | i0, i1, i13, i18, i2, i3, i31, i4, i6, i8, i9 | non_porte | aucun | 40 | 214 | HI_1_10_0, HI_1_12_0, HI_1_13_0, HI_1_8_0, HI_1_9_0 |
| 34 | ti=43 device-position-animation-name | 43 | i19 | non_porte | aucun | 37 | 333 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
| 35 | ti=40 warp | 40 | i45, i46 | non_porte | aucun | 36 | 679 | HI_1_11_0, HI_1_4_1, version-31, version-33 |
| 36 | ti=0 game-engine-soft-ceilings | 0 | i11 | non_porte | aucun | 34 | 216 | HI_1_10_0, HI_1_13_0, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 37 | ti=52 archetype hors registre | 52 |  | - | - | 34 | 227 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0, version-31, version-33 |
| 38 | ti=55 archetype hors registre | 55 |  | - | - | 29 | 203 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_8_0, HI_1_9_0 |
| 39 | ti=0 game-engine-disabled-kill-volume-flags | 0 | i13 | non_porte | aucun | 27 | 161 | HI_1_10_0, HI_1_13_0, HI_1_8_0, version-33 |
| 40 | ti=40 vehicle-auto-turret-triggers | 40 | i30 | non_porte | aucun | 27 | 37 | HI_1_10_0, HI_1_11_0, HI_1_13_0, HI_1_4_1, HI_1_9_0, version-33 |
