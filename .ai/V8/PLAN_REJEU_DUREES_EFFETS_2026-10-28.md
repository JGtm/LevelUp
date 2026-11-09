# PLAN — Durées et positions des effets d'équipement lues dans le film (2026-10-28)

Branche : `feat/rejeu-durees-effets` (worktree `LevelUp-wt-durees-effets`), partie de `fix/rejeu-objets-engendres`
(`dad453334`, pas encore fusionnée dans `release/v8.1.0`). Contrat d'exécution : skill `plan-execution`
(ordre strict, aucune case vide à la clôture, zéro fix hors périmètre, découvertes notées et non traitées).

## Contexte (faits établis, ne pas re-démontrer)

- Le décodeur publie pour chaque objet engendré par un équipement porté (`carriedEquipment[].tracked[]`,
  pile i28) sa position de repos (trajectoire ti=41 de la même vie) — lot `fix/rejeu-objets-engendres`,
  `document_objets_engendres.go`.
- Le `t1` lu de cet objet (suppression écrite par le film, `CarriedTrackedEnd`) vaut la durée de
  l'effet pour le capteur (~15 s) et le champ de réparation (~11,2 s), mais pas pour l'écran occultant
  ni le traqueur (fin de la grenade, 0,1 à 0,4 s après le repos).
- SONDE DU SUPERVISEUR (f59c9332, Catalyst, 2026-10-28) : à l'impact de CHACUN des 11 écrans occultants
  naît une vie ti=38 que la pile i28 ne nomme pas, 150 à 300 ms AVANT la suppression de la grenade, qui
  vit 7,51 s (11/11 ; aucune aux instants témoins décalés de 37 s), avec une position à la création
  (`cfCanal.positionALaCreation`, posTI=38). Un écran sans trajectoire a sa vie ti=38 à 0,1 m de la main
  (lancé aux pieds). Des vies ti=12 de 5,0 s apparaissent aussi autour des lancers (non instruites).
- Le 103 (`ScanEquipmentSpawnEvents`) émis 33 à 67 ms après la pile nomme la GRENADE elle-même, pas la bulle.
- Décision du user (2026-10-28) : durée de l'écran = celle du film, 7,5 s (`SHROUD_DURATION_MS`,
  `apps/web/src/features/match-replay/layers/placementShapes.ts`). Il mesure 10 s à l'écran en jeu.
- Instruments existants à réutiliser (ne pas en réécrire) : `internal/games/halo_infinite/film/internal/grammar/`
  `champ_fin_research_test.go` (`cfCanal` : naissance/suppression de toute vie, position à la création
  pour un archétype), `equipporte_research_test.go` (`epContexte`, `EQUIPPORTE_CARTE`), le canal de
  production `nouveauCanalDeLEquipementPorte` (`prod.lu.lectures`).

## Objectif

Établir, PAR FAMILLE d'équipement, si le film porte un objet d'effet (né à l'usage ou à l'impact, distinct
de l'objet suivi par la pile) dont la naissance, la suppression et la position donnent la DURÉE et la
POSITION de l'effet, de façon fiable (constante ou explicable, sur plusieurs films et plusieurs builds).
Puis, sur décision du superviseur et du user, lire cet objet en production.

## Étape A — recherche (lecture seule, aucun code de production)

Films (tous locaux sous `C:\Users\Guillaume\Downloads\Scripts\LevelUp\data\cache\film_chunks\`) :

| id | carte (`EQUIPPORTE_CARTE`) | date | familles riches |
|---|---|---|---|
| 1ea71def | Deadlock | 2025-11-13 | capteur 15, mur 23, écran 10, camo 17, champ 23 |
| 52fc79ef | Corpo | 2025-12-20 | capteur 18, mur 19 |
| cba488f7 | Starboard | 2025-12-30 | écran 13 |
| c3af8191 | Forbidden | 2026-01-26 | traqueur 20 |
| 58d09c44 | Refuge | 2026-01-28 | capteur 18, mur 17, surbouclier 8, répulseur 21 |
| f59c9332 | Catalyst | 2026-04-17 | écran 11, translocateur 6 |
| bc0bdda3 | Forbidden | 2026-05-05 | traqueur 24 |
| 44e14331 | Shiro | 2026-07-03 | surbouclier 9 |
| 5676a9ba | Insolence | 2026-07-24 | champ 11, capteur 13, écran 7 |

Familles : capteur (`sensor`), traqueur (`threat_seeker`), écran (`shroud_screen`), champ de réparation
(`repair_field`), mur (`wall`), translocateur (`translocator_beacon`), et à titre de contrôle répulseur,
surbouclier et camouflage (familles sans objet attendu). La famille d'un usage se lit par le rang i48
(`ScanEquipmentChanges`, comme `TestChampParc`) rapproché des `abilityLabels` de l'artefact du match
(`C:\Users\Guillaume\Downloads\Scripts\LevelUp\data\cache\replays\halo_infinite\<id>.json`, champ `family`).

- [x] A.1 UN instrument de recherche `//go:build research` dans `internal/grammar/` (un fichier, réutilisant
  `cfCanal`, `epContexte`, `bornesDeLaCarte`, le canal de production). Pour chaque usage d'équipement
  (lecture `uses` du canal de production ou création i28), lister les vies NÉES dans
  [usage − 0,5 s, fin de l'objet suivi + 1,5 s] que la pile i28 ne nomme pas : archétype, décalage de
  naissance (par rapport à l'usage ET à la fin de l'objet suivi), durée de vie (suppression lue ou non),
  position à la création quand l'archétype est lisible, distance au repos de l'objet suivi. Et les mêmes
  comptes à des instants TÉMOINS (usage + 37 s) pour séparer le signal du bruit. Sortie TSV dans le
  scratchpad, une ligne par (usage, vie candidate).
- [x] A.2 Exécution sur les 9 films, UN PAR UN (jamais en parallèle, jamais en lot de processus), sous
  `CGO_ENABLED=0`, `GOCACHE=%LOCALAPPDATA%\go-build-durees-effets`. Rien n'est écrit hors du scratchpad.
  Noter la durée et le pic mémoire de chaque passe (le superviseur a mesuré 9 s sur f59c9332).
- [x] A.3 Tableau de synthèse par famille, dans ce plan (section « Résultats A ») : objet d'effet trouvé
  (archétype) ou non ; part des usages qui en ont un ; durée (min / médiane / max, et dispersion) ; part
  avec position ; distance médiane position d'effet ↔ repos de l'objet suivi ; témoins (combien de vies du
  même archétype et de même durée aux instants décalés). Verdict par famille : FIABLE (≥ 90 % des usages,
  durée constante à ±0,1 s ou explicable, ≤ 5 % aux témoins) / PARTIEL / ABSENT. Comparer aux durées
  déjà utilisées : capteur 15 s (`SENSOR_DURATION_MS`), mur (`WALL_DURATION_MS`), écran 7,5 s.
- [x] A.4 Les vies ti=12 de ~5,0 s : à quelle famille se rattachent-elles (usage, lancer, impact), ou bruit ?
  Une ligne de verdict.

Gate A : l'instrument compile sous `-tags research` (`go vet -tags research ./internal/games/halo_infinite/film/internal/grammar/`),
le tableau A.3 est rempli pour les 6 familles + 3 contrôles, chaque verdict cite ses comptes. ARRÊT et
compte rendu au superviseur : l'étape B ne commence pas sans son accord écrit.

## Résultats A (2026-10-28, exécuteur Sonnet)

**Instrument** : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/durees_effets_research_test.go`
(`//go:build research`, `TestDureesEffets`). Il réutilise `cfCanal`, `epContexte`, `bornesDeLaCarte`,
`nouveauCanalDeLEquipementPorte`, `ScanEquipementPorte`, `ScanEquipmentChanges`, `ScanProjectiles`,
`ScanClockOrigin` ; deux marches du film (1 : toutes les vies + lectures de l'équipement porté ;
2 : position à la création des seules vies voulues, via `cfCanal.positionALaCreation`). Sorties TSV par film
dans le scratchpad de session (`.../scratchpad/durees_effets/out/<id>.{usages,candidats,fenetres,ti_extra,archetypes}.tsv`
+ `run.txt`). Dépouillement : `analyse*.js`, `explore.js`, `ti12*.js` (même dossier, hors dépôt).

**Définitions.** Usage = lecture d'activation (i21) du canal de production, famille = rang i48 du paquet de la
prise joint à la palette `abilityLabels` de l'artefact ; recoupé : mes comptes d'usages par famille égalent
`carriedEquipment[].uses` de l'artefact sur 8 films sur 9 (sur `52fc79ef` l'instrument voit 44 murs / 49 capteurs
contre 19 / 18 publiés : porteurs sans piste publiée). Candidate = vie née dans [usage - 0,5 s, fin de l'objet suivi + 1,5 s]
(ou usage + 20 s sans objet suivi / sans fin lue) que AUCUNE pile i28 ne nomme. Témoins = mêmes fenêtres décalées de
+37 s et -37 s ; « propre » = aucun autre usage dans la fenêtre, « semi » = aucun autre usage de famille connue.
Part = usages ayant au moins une candidate de la grappe (archétype, durée à +-60 ms). Distance = effet (position à
la création) <-> repos de l'objet suivi (dernier point de sa piste ti=41 née à son nommage ; à défaut sa position
à la création pour un objet suivi non projectile). Les artefacts en cache ne publient AUCUNE position de repos
(`tracked[].x` absent des 9), le repos est donc relu dans le film.

| Famille (films / usages) | Objet d'effet distinct | Part des usages | Durée min / méd / max | Position | Dist. effet <-> repos | Témoins | Verdict |
|---|---|---|---|---|---|---|---|
| écran occultant `shroud_screen` (5 / 39) | **ti=38**, né à l'impact, 150 à 600 ms (méd 250) AVANT la suppression de la grenade, 150 à 1 300 ms (méd 372) après l'usage | 38/39 (97 %) ; par film 9/10, 4/4, 1/1, 13/13, 11/11 | 7 492 / 7 507 / 7 512 ms (constant +-10 ms, 2025-11 à 2026-07) | 38/38 | méd 0,03 m, max 0,65 m (n=34 ; 4 écrans sans piste de grenade ont quand même la position de l'effet) ; à la création de la grenade (lancer) méd 4,3 m | 5/74 (7 %) toutes fenêtres, 1/59 (2 %) propres, 1/63 semi | **FIABLE** (seul échec : `1ea71def` u177, grenade vivante 6,6 s, une ti=38 de 7 975 ms née 0,44 s AVANT l'usage) |
| mur `wall` (7 / 117) | **ti=38**, née 1,02 s (0,72 à 1,40) après l'usage ; FINIT avec l'objet suivi (ti=37) : écart de fin méd 0 ms [-66 ; +17] | 97/104 (93 %) des usages dont la fin du suivi est lue ; 97/117 (83 %) de tous (6 sans objet suivi, 7 sans fin lue) | naturelle 12 395 à 12 545 ms (5 films, méd 12 480), 16 650 à 16 720 ms (`52fc79ef`) ; tronquée de 1,3 à 11,5 s quand le mur est détruit ; l'objet suivi dure 0,57 à 0,90 s de plus | 97/97 | méd 1,40 m, p90 4,35 m (repos = position à la création du ti=37 suivi, pas une piste) | 0/28 propres, 0/59 semi (signature « finit avec le suivi ») ; la signature « née à +1 s » seule : 6/29 propres (21 %), bruit des autres murs | **FIABLE sous condition** (fin du suivi lue) ; PARTIEL (83 %) sur tous les usages |
| capteur `sensor` (4 / 96) | aucun (l'objet suivi ti=41 EST l'effet) ; ti=12 de 0,70 à 0,77 s, 31/96 (32 %), non ancrées, sans position : pings de détection | - (meilleure grappe ti=41 1,64 s : 65 % contre 19 % propres / 31 % semi, non ancrée, méd +15,7 s : bruit) | objet suivi : 15,2 à 16,8 s (n=33), 20,4 à 21,4 s (n=18, `52fc79ef`), fin avancée 0,6 à 11,6 s (n=32), 12,3 à 14,2 s (3), 18,5 à 18,6 s (2), 5 valeurs de 124 à 447 s (anormales) ; fin lue 93/96 | piste 88/95 | - | - | **ABSENT** comme objet distinct |
| champ de réparation `repair_field` (3 / 35) | aucun (suivi ti=41 = l'effet) | - (ti=41 0,42 s : 80 % contre 13 % propres / 38 % semi, non ancrée : bruit) | objet suivi : 11,2 à 12,2 s (25/34), fin avancée 1,7 à 9,7 s (8), 16,9 s (1) ; fin lue 34/35 | piste 30/35 | - | - | **ABSENT** comme objet distinct |
| traqueur `threat_seeker` (2 / 44) | ti=12 de 4 486 à 4 488 ms, née 0,8 à 2,3 s après l'usage, 0,3 s avant la fin de la grenade : marqueur de cible, pas l'effet | 10/44 (23 %) | 4 486 / 4 487 / 4 488 ms (objet suivi = grenade : 0,95 / 1,5 / 2,8 s) | 0/14 (navpoint sans position) | - | 0/66 propres, 0/68 semi | **PARTIEL** (marqueur seul ; aucun objet d'effet de durée ou de position) |
| translocateur `translocator_beacon` (1 / 6) | aucun d'utile : l'objet suivi (faille ti=38) donne déjà durée et position ; ti=10 née dans le même paquet que la faille (6/6), meurt 150 à 380 ms avant elle, sans position | ti=10 : 6/6 | faille : 10,1 / 30,2 / 30,2 s (10,1 ; 20,1 ; 30,2 x4) | faille 6/6 (à la création) ; ti=10 0/6 | - | ti=10 de 30 s aussi dans 2/2 fenêtres propres mais jamais née au nommage de la faille (celle du réel : à 0 à 2 ms) | **ABSENT** (redondant ; un seul film) |
| contrôle répulseur `repulsor` (5 / 43) | **inattendu : rafale de 32 vies ti=41** (min 32, méd 32, max 50) nées 0 à 83 ms (méd 37) après l'usage | 43/43 (100 %) | 299 / 415 / 2 305 ms (p5 300, p95 762) | 1 430/1 430 | - | >=30 vies : 1/81 (0/11 propres) | rafale FIABLE comme signature de l'usage ; effet instantané (poussée) : pas de durée d'intérêt |
| contrôle surbouclier `powerup_overshield` (4 / 24) | aucun ; ti=37 de 3,5 s : 10/24 mais non ancrée (offset -0,5 à +19,4 s) | - | - | - | - | 1/22 propres | **ABSENT** |
| contrôle camouflage `powerup_camo` (4 / 36) | aucun ; ti=37 de 2,75 s : 25/36 mais offsets plats (rythme 1,0 contre 0,6 par fenêtre témoin) | - | - | - | - | 0/8 propres, 2/22 semi | **ABSENT** |

Lecture. Pour l'écran occultant, la naissance de la ti=38 (impact) borne l'effet : début = naissance de la ti=38
(position de l'effet = position de repos de la grenade, à 0,03 m près), fin = naissance + 7,507 s. Pour le mur,
début = naissance de la ti=38 (environ 1 s après l'usage), fin = suppression de l'objet suivi (déjà publiée en
`t1`) : l'apport nouveau est le DÉBUT (l'objet suivi est nommé environ 0,2 s après l'usage, la barrière naît 0,8 s
plus tard) et l'archétype porte la position du mur. Capteur et champ de réparation : `t1 - t0` de l'objet suivi est déjà
la durée de l'effet (rien à lire de plus) ; elle VARIE selon le film (capteur 15,2 à 16,8 s (n=33) ou 20,4 à 21,4 s (n=18) ;
mur 12,9 à 14,6 s (n=52) ou 17,0 à 17,9 s (n=30) ; `52fc79ef`, Corpo 2025-12-20, est le film aux durées longues : variante de jeu
non identifiée) : la constante `WALL_DURATION_MS` (10 s) ne tient dans aucun film mesuré (barrière 12,4 s, objet
suivi 13,2 s) ; `SENSOR_DURATION_MS` (15 s) est proche de la durée lue sur 3 films sur 4 (15,2 à 16,8 s) et loin de celle de `52fc79ef` (20,4 à 21,4 s).

### A.4 — les vies ti=12 de ~5,0 s : BRUIT

ti=12 = archétype `managed-navpoint-*` (marqueur d'interface). Sur les 9 films : 1 141 vies, dont 482 (42 %) de
4 950 à 5 050 ms (la durée standard des marqueurs). Elles ne se rattachent ni aux lancers ni aux usages : 129/482
(27 %) naissent dans [-0,3 ; +0,7] s d'un lancer de grenade (`grenades` de l'artefact) contre 26 % attendu au hasard
(part du temps de film couverte par ces fenêtres) ; 91/482 (19 %) dans [-0,5 ; +3] s d'un usage d'équipement contre
18 % attendu ; par famille, aucun écart net (camouflage 14 contre 8,3 attendu, écran 14 contre 8,4, traqueur 7 contre
13,5). Seules durées distinctes liées à une famille : le marqueur de cible du traqueur (4,487 s, ci-dessus) et les pings
du capteur (0,73 s).

### Coût de l'instrument (par film, séquentiel ; pic mémoire = jeu de travail maximal du processus de test / tas vivant échantillonné)

| film | durée | pic mémoire | usages | vies | candidates |
|---|---|---|---|---|---|
| 1ea71def | 17,8 s | 457 Mo / 334 Mo | 191 | 5 359 | 56 864 |
| 52fc79ef | 10,6 s | 387 Mo / 257 Mo | 93 | 8 080 | 58 962 |
| cba488f7 | 5,1 s | 251 Mo / 185 Mo | 46 | 1 252 | 4 237 |
| c3af8191 | 5,1 s | 260 Mo / 209 Mo | 67 | 1 892 | 7 817 |
| 58d09c44 | 25,8 s | 767 Mo / 534 Mo | 175 | 8 259 | 107 355 |
| f59c9332 | 5,6 s | 307 Mo / 213 Mo | 51 | 1 508 | 5 470 |
| bc0bdda3 | 5,1 s | 271 Mo / 175 Mo | 80 | 1 385 | 6 981 |
| 44e14331 | 6,1 s | 342 Mo / 218 Mo | 33 | 2 752 | 5 793 |
| 5676a9ba | 10,1 s | 330 Mo / 246 Mo | 129 | 4 210 | 44 234 |

Incertitudes : (1) mur, la distance « au repos » est mesurée à la création de l'objet suivi ti=37, pas à une
piste ; (2) capteur / champ / traqueur / translocateur : 1 à 4 films, 6 à 96 usages ; (3) le « build » n'est pas lu,
seulement la date du film (2025-11-13 à 2026-07-24) ; (4) les fenêtres témoins « propres » sont peu nombreuses sur les
familles denses (8 à 36) : d'où la colonne « semi » ; (5) les usages de famille inconnue (rang absent, 15 à 110 par film)
ne sont pas classés ; (6) `52fc79ef` : durées longues (mur 17,2 s, capteur 20,6 s) sans cause établie.

## Étape B — fins et positions LUES pour les équipements qui durent (décision du user, 2026-10-28)

Décision du user (2026-10-28) : « c'est beaucoup mieux en lisant la durée de vie dans le film, fais pareil pour
les équipements qui durent ». Périmètre FERMÉ : mur de protection et écran occultant. Capteur et champ de
réparation lisent déjà leur fin (`endRead`, lot précédent) ; traqueur (impulsion), translocateur (faille déjà lue),
répulseur, surbouclier, camouflage : HORS périmètre. Base : fusionner d'abord `release/v8.1.0` (qui contient le lot
`fix/rejeu-objets-engendres`, schéma 105) dans la branche.

Invariants NON NÉGOCIABLES (exigence du user) : le DÉBUT et le TRACÉ du mur, la faille du translocateur et TOUS les
sons du rejeu restent identiques à l'octet (témoins avant/après, cf. B.0). Seule change la FIN d'affichage du mur et
de l'écran, et la position de l'écran quand sa grenade n'a pas de trajectoire. Une pose sans fin lue garde une
règle de repli (constante de famille).

- [x] B.0 Témoins AVANT (règle `feedback_temoins_avant_apres`) : sur 5 films (1ea71def, 52fc79ef, cba488f7,
  f59c9332, 5676a9ba), documents Go complets (T-Go) et sortie du script de caractérisation web du lot précédent
  (T-Web : faille, calendrier complet des sons, poses d'origine, fenêtres de toutes les poses), capturés depuis le
  code de la release AVANT toute modification, rangés dans le scratchpad `durees_effets/temoins/avant/`.
- [x] B.1 Mur, fin lue. Établir sur pièces le LIEN STRUCTUREL entre la pose de mur publiée (`equipmentPlacements`,
  origine `deployed`, issue du 103) et l'objet suivi du mur (`carriedEquipment[].tracked[]`, dont `t1` est la
  suppression écrite par le film) : clé de vie, référence du 103 (`Source` / `Spawned`), porteur. LA GRAMMAIRE PRIME :
  une jointure par proximité temporelle n'est admise que si aucun lien structurel n'existe, et alors bornée, comptée
  (couverture : liées / sans lien / ambiguës) et justifiée au journal. Publier côté Go la fin lue sur la pose de mur
  (champ dédié, schéma 106, calculé à la publication depuis les faits) ; web : la fenêtre du mur s'arrête à cette
  fin quand elle est lue. Repli sans fin lue : `WALL_DURATION_MS` passe de 10 s à la médiane mesurée de la
  barrière (12 480 ms, Résultats A), commentaire mis à jour (source = mesure du film, pas « une dizaine »).
- [x] B.2 Écran occultant, objet d'effet. Dans la grammaire de production (canal de l'équipement porté), lire la vie
  ti=38 née à l'impact de chaque objet suivi d'écran (naissance, suppression, position à la création). LIEN
  STRUCTUREL d'abord (composant du record de création de la ti=38 qui désigne la grenade, le porteur ou l'objet
  porté) ; à défaut règle d'association bornée (fenêtre et distance tirées des Résultats A), comptée et nommée
  (repli `fallback` si le projet l'exige pour un choix non lu). Révision de la couche concernée (grammaire et/ou
  faits, `SchemaDesFaits`), doc : `tracked[].effect {t0, t1, x, y, z}` (absents ensemble quand non lus), couverture
  et ligne slog. Web : l'écran se dessine de `effect.t0` à `effect.t1` à la position de l'effet ; sans effet, règle
  actuelle (repos de la grenade + `SHROUD_DURATION_MS`). Un écran sans trajectoire mais avec effet se dessine.
  Ne pas brancher le mur sur cet effet (B.1 suffit) sauf si B.1 le rend nécessaire, et alors le noter.
- [x] B.3 Tests : Go (lien ou association, bords de fenêtre, clé recyclée, absence), web (fenêtres mur et écran,
  repli), mutations vérifiées (inverser la condition fait rougir un test). Goldens et fixtures régénérés par les
  outils du projet (jamais à la main), `make generate-types` si l'OpenAPI bouge. Plafonds de taille : ne pas les
  relever sans justification datée.
- [x] B.4 Témoins APRÈS sur les 5 mêmes films : T-Go identique hors champs neufs et fins de mur ; T-Web : faille,
  sons, début et tracé du mur identiques à l'octet ; tableau des fins de mur et d'écran avant/après par film.
- [x] B.5 Gates : `go vet ./...`, `go test ./...` et `go test -tags=integration -p 1` sur les paquets du film
  touchés (depuis `apps/go-api`, `GOCACHE=%LOCALAPPDATA%\go-build-durees-effets`), `npx tsc -b --force`, eslint,
  vitest `src/features/match-replay src/lib/replay src/lib/api src/lib/i18n`, knip. Codes de sortie relevés.
- [x] B.6 Changelog et notes de version EN + FR (une ligne chacun), référence des canaux d'équipement
  (`.ai/archive/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`), entrée `.ai/thought_log.md`. NE PAS commiter :
  le superviseur relit, fait la revue adversariale et commite. Recuisson du parc : NE PAS lancer (le superviseur
  la demande au user).

- [x] B.7 Revue adversariale de l'étape B (superviseur, 2026-10-28) : 0 P0/P1, 5 P2 corrigés dans le lot (journal ci-dessous), sans décodage.

Gate B : B.0 à B.6 statués, gates verts, témoins conformes aux invariants. Compte rendu au superviseur.

## Découvertes (non traitées)

- (Étape B) `durees_effets_research_test.go` (étape A, commité) passait le ratchet de taille de `archlint` à 702 lignes : la
  justification écrite en tête n'est pas une exemption que le test reconnaît. Scindé par déplacement pur en deux fichiers
  (`durees_effets_ecriture_research_test.go`), la justification retirée (453 + 220 lignes).
- (Étape B) `film/types` est dans le périmètre de la couche `source` (fermeture des imports de `revision/`) : y ajouter les deux
  genres de lecture neufs aurait monté `source.Rev` (donc tout le décodeur et le backlog killsource). Les genres sont déclarés
  dans `grammar` (`CarriedEffectBirth`, `CarriedEffectEnd`, de type `types.CarriedReadKind`).
- (Étape B) Les quanta i0 d'un objet engendré sont rangés par clé de vie seule (`equipementPorteLu.quanta`) : une clé recyclée
  (2 bits de génération) écrase la position d'une naissance antérieure. Les candidates à l'effet ont leur table propre (vie + instant) ;
  la table des objets suivis n'est PAS corrigée (hors périmètre).
- (Étape B) Le repos publié d'une grenade (dernier point de grille à 100 ms de sa trajectoire) est à 0 à 1,96 m de la bulle (méd 0,6 m),
  pas à 0,03 m comme le mesurait l'étape A sur le dernier échantillon brut.
- (Étape B) Le décalage `originMs` du document (frames) et `ScanClockOrigin` (instruments) diffère de ~29,9 s sur `cba488f7` :
  les frames d'un instrument ne se lisent pas sur celles de l'artefact sans ce décalage.

- Le répulseur écrit une RAFALE de 32 vies ti=41 (0,3 à 0,76 s) à chacun de ses usages (43/43, témoins 1/81) : signature
  fiable de l'usage et de sa position, non instruite (hors périmètre de l'étape A).
- Les 9 artefacts en cache (schéma 104) ne portent aucune position de repos des objets engendrés (`tracked[].x` absent) :
  la couche du lot `fix/rejeu-objets-engendres` n'est pas encore cuite dans le parc.
- Capteur : 5 objets suivis dont la suppression est lue 124 à 447 s plus tard (2 sur `52fc79ef`, 3 sur `58d09c44`) : clé (slot, génération) réutilisée ou objet jamais supprimé ; la durée `t1 - t0` y est fausse.
- Capteur : 32 objets suivis sur 93 voient leur fin avant 11,6 s (capteur détruit ?) : `t1` est alors une destruction, pas une expiration.
- `52fc79ef` (Corpo, 2025-12-20) : durées longues (mur 17,2 s, capteur 20,6 s) contre 13,2 s / 15,5 s ailleurs : variante de jeu à identifier avant de figer une constante.
- Le traqueur n'a pas d'objet d'effet : ses seules traces distinctes sont des marqueurs ti=12 de 4,487 s sans position.
- `ti=12` : marqueurs d'interface (`managed-navpoint-*`), 482 de 1 141 durent 5,0 s ; sans lien avec les lancers ni les usages.
- Le translocateur (ti=10 jumelle de la faille, 150 à 380 ms plus courte) n'apporte rien de plus que l'objet suivi.

## Journal

### 2026-10-28 — Ouverture (superviseur)

- Plan rédigé ; worktree et branche créés depuis `dad453334`. Étape A confiée à un exécuteur Sonnet.

### 2026-10-28 — Étape A exécutée (exécuteur Sonnet)

- A.1 : instrument `durees_effets_research_test.go` (un fichier `research`, 701 lignes, aucune aide réécrite) ; `go vet -tags research ./internal/games/halo_infinite/film/internal/grammar/` code 0, gofmt propre.
- A.2 : 9 films, un processus à la fois (binaire de test compilé `go test -c -tags research`, `CGO_ENABLED=0`, `GOCACHE=%LOCALAPPDATA%\go-build-durees-effets`), tous code de sortie 0, 5 à 26 s et 251 à 767 Mo de jeu de travail par film (tableau « Coût » ci-dessus). Rien écrit hors du scratchpad.
- A.3 : tableau rempli ; FIABLE : écran occultant (ti=38, 7,507 s) et mur (ti=38, fin = fin du suivi, sous condition) ; ABSENT : capteur, champ de réparation, translocateur, surbouclier, camouflage ; PARTIEL : traqueur (marqueur de cible seul) ; contrôle répulseur : rafale de 32 ti=41 (découverte).
- A.4 : ti=12 de 5,0 s = bruit (marqueurs d'interface, 27 % près d'un lancer pour 26 % attendu).
- Reste incertain : mur « au repos » mesuré à la création du suivi ; familles à 1 ou 2 films ; `52fc79ef` à durées longues ; cause des 5 fins de capteur à 124 s et plus.
- Étape B NON commencée (accord écrit du superviseur requis). Rien commité.

### 2026-10-28 — Étape B exécutée (exécuteur Sonnet)

- Base : `release/v8.1.0` (`3d39a67c7`, lot `fix/rejeu-objets-engendres`, schéma 105) fusionnée dans la branche (`ee887a41c`).
- Jonctions posées sous le worktree, À RETIRER à la clôture (PowerShell `(Get-Item <chemin>).Delete()`, jamais `Remove-Item -Recurse`) :
  `data/cache/film_chunks` et `data/cache/film_manifests` (vers le dépôt principal, lecture seule) et `apps/web/node_modules`
  (vers le dépôt principal : `tsc -b`, vitest et knip y ont écrit leurs caches `.tmp` / `.vite`).
- **B.0** Binaire `replay-build` AVANT construit depuis `ee887a41c` avant toute modification ; 5 films, un par un, `LEVELUP_REPO_ROOT` = le worktree,
  sans `--facts` : 1ea71def Deadlock 36,6 s / 1,04 Go, 52fc79ef Corpo 14,6 s / 0,48 Go, cba488f7 Starboard 8,2 s / 0,41 Go,
  f59c9332 Catalyst 9 s / 0,54 Go, 5676a9ba Insolence 21,8 s / 0,58 Go (jeu de travail max). md5 AVANT 1ea71def `2fb9a4cc…`, 52fc79ef `8d2a558e…`,
  cba488f7 `d6a9f860…`, f59c9332 `06dc2024…`, 5676a9ba `445c14e3…`. T-Web AVANT (script local non versionné, supprimé : poses dessinées, début/tracé,
  fenêtres, faille, calendrier complet des sons, usage), deux passes identiques.
- **B.1 LIEN STRUCTUREL TROUVÉ** (instrument `durees_effets_lien_research_test.go`, 3 films à murs) : le panneau du mur publié (vie ti=37 que le balayage
  des poses rend et que le 103 désigne) est la MÊME vie (slot, génération) que celle que la pile i28 de l'objet porté nomme, 16 à 45 ms APRÈS sa
  création ; `Source` du 103 = l'objet porté (66 sur 66). Mesure : 67 murs suivis, 66 liés par la clé (le dernier n'a pas de 103). La
  génération ne tient que sur 2 bits : le lien exige la clé ET un nommage qui suit la création d'au plus 200 ms (borne du 103). Aucune jointure par
  proximité. Publié : `equipmentPlacements[].lifeEnd` ; web : la fenêtre s'arrête à cette fin ; repli `WALL_DURATION_MS` = 12 480 ms.
- **B.2 AUCUN LIEN STRUCTUREL** : le record de création de la ti=38 de la bulle ne porte que i0, i2, i14, i15 (pas de i9 ni i10) : il ne désigne ni la grenade
  ni le porteur. Association bornée, nommée, comptée (repli `repli_effet_d_ecran_par_la_fin_de_la_grenade`) : candidates = ti=38 non nommées nées 100 à 700 ms
  avant la suppression de la grenade (mesure A : 150 à 600 ms) ; retenue = à 3 m du repos publié de la grenade, ou seule de sa fenêtre sans repos ;
  plusieurs = ambigu, rien. Comptes sur les 4 films à écrans (35 grenades) : 34 effets lus (1ea71def 9/10, cba488f7 13/13, f59c9332 11/11, 5676a9ba 1/1),
  0 ambigu, 1 sans lien (la grenade restée 6,6 s), 34 suppressions lues. Distance bulle <-> repos publié 0 à 1,96 m (30 mesures), rivales à 9,5 et 30 m (étape A).
  Faits : `grammar.Rev` -> `grammar-2026-10-28` (deux genres de lecture neufs, codec inchangé, `SchemaDesFaits` 18 inchangé, `killsource`/`objectives` à révision constante).
  Schéma 105 -> 106.
- Trois erreurs de lecture corrigées en route (chacune a coûté une série de décodages APRÈS) : (1) l'exclusion « vie déjà nommée » ignorait que la clé
  revient (rendue datée) ; (2) les quanta de position étaient rangés par clé seule (table propre aux candidates) ; (3) le rayon de 1,5 m, mesuré en étape A sur le
  dernier échantillon brut, rejetait des bulles à 1,5 à 1,96 m du repos PUBLIÉ (grille de 100 ms) : 28 effets sur 35, puis 34 sur 35 à 3 m.
- **Décodages** (autorisés : 5 avant, 5 après, un par un) : AVANT 5. APRÈS : 5 séries de 5 (la première, une par correction ci-dessus, la série finale) = 25, plus 1 de contrôle
  (`cba488f7` avec le binaire final : même md5), plus ~10 passes d'instruments de recherche, toujours un processus à la fois, jamais plus de 1,1 Go, jamais pendant la recuisson. À dire au user.
- **B.3** Tests : Go grammaire (`equipement_porte_effet_test.go` : fenêtre aux bornes, exclusions, clé recyclée, fin rendue, génération recyclée) ; Go replay
  (`document_effets_test.go`, `equipment_placement_life_end_test.go`) ; web (fenêtre du mur : fin lue avant/après le repli, bornes, début inchangé ;
  écran : bulle lue, sans fin, sans trajectoire, sans position, autre famille). MUTATIONS vérifiées rouges puis restaurées : fenêtre 100 ms, nommage 2 s,
  borne de nommage du mur (200 ms), rayon <= 3 m, famille de l'écran, fin d'effet antérieure à la naissance, `lifeEnd` dans la fenêtre du mur, borne basse
  `max(lifeEnd, t0)`, `endRead` de la bulle, famille de la bulle côté web, repos de repli, `WALL_DURATION_MS`. Un mutant survit : retirer `v.nomme >= p.T0US` de
  `finsLuesDesPoses` ne change rien (le test `v.nomme-p.T0US <= 200 ms` en uint64 déborde pour un nommage antérieur : condition redondante, gardée par lisibilité).
  Goldens (assemblage x8, forme, formes des types, révisions grammar/killsource/objectives) et fixtures web `replay_schema_106_*` (x8) régénérés par les outils du
  projet ; `openapi.yaml` et `generated.ts` régénérés (`openapi-gen`, `npm run generate-types`). Plafonds `document_chronicle.go` 3633 -> 3659 et `structure_test.go`
  1486 -> 1491 (exceptions écrites du ratchet, montée de `SchemaVersion`).
- **B.4** T-Go : après retrait de `schemaVersion`, `layers`, `coverage.decoder.grammarRev`, du repli neuf, de `lifeEnd`, `effect` et des 7 compteurs neufs, les 5 documents sont IDENTIQUES
  (écarts : 0). T-Web : faille, calendrier complet des sons, usage d'équipement IDENTIQUES à l'octet sur 5 films ; début, tracé et position des poses hors écran identiques
  (hors indice : un écran de plus décale les poses fabriquées qui le suivent) ; début du mur (`t0`) et `t1` identiques sur 68 murs ; deux passes APRÈS identiques.
  Tableau murs : 1ea71def 24 murs, 22 à fin lue, fenêtre 10,0 s -> 5,4/13,3/14,6 s (min/méd/max) ; 52fc79ef 41 murs, 37 à fin lue, 10,0 -> 2,4/17,2/20,0 s ; 5676a9ba 3 murs, 3 lues, 10,0 -> 2,4/5,4/6,7 s.
  Tableau écrans : 1ea71def 8 -> 10 dessinés, 7,5 -> 7,5/7,5/7,6 s ; cba488f7 13 -> 13, 7,4/7,5/7,5 ; f59c9332 9 -> 11 (deux écrans sans trajectoire dessinés), 7,5 ; 5676a9ba 1 -> 1, 7,5 ;
  décalage de début médian 0 image, déplacement de position médian 0,15 à 1,07 m (max 1,93 m).
- **B.5** Gates : `go vet ./...` rc=0 (CGO=1, gcc msys64 15.2.0) ; `go test ./...` rc=0 (214 paquets ok, aucun FAIL, 297 s) ; `go test -tags=integration -p 1` sur `film/...`, `replaydoc`, `replayview`, `replaydiff` rc=0 (22 paquets ok) ; `go vet -tags research` sur `grammar` rc=0 ; `openapi-gen -check` rc=0 ; web : `npx tsc -b --force` rc=0, eslint (match-replay, lib/replay, lib/api) 0 erreur (8 avertissements, fichiers non touchés), vitest match-replay + lib/replay + lib/api + lib/i18n : 296 fichiers / 3 951 tests verts (4 fichiers / 7 tests ignorés préexistants), `npx knip` rc=0. Vérification finale : le binaire final redécode `cba488f7` (depuis_les_faits=false) au même md5 que la série APRÈS.
- **B.6** Changelog et notes de version EN + FR, référence des canaux d'équipement (section « Fin lue du mur et objet d'effet… »), entrée `.ai/thought_log.md`. Rien commité, aucune recuisson lancée.

### 2026-10-28 — Correction des 5 P2 de la revue adversariale (exécuteur Sonnet, sans décodage)

Chaque correction a un test qui rougit si on l'annule (mutation vérifiée puis restaurée).
1. **`SpawnedPositioned` dépassait son dénominateur** (`ScanEquipementPorte`) : une candidate à l'effet positionnée l'incrémentait alors que `SpawnedBirths` ne compte que les vies nommées. Un effet n'incrémente plus ce compte (pas de nouveau compteur : ajouter un champ aux statistiques changerait le codec des faits et `film/types`, dans le périmètre de la couche `source`). Test `TestScanEquipementPorteBrancheDeLEffet` (statistiques : seule la vie nommée se compte). Mutation : incrémenter pour tout genre de lecture -> rouge.
2. **Candidate réattribuée en silence** (`candidatesALEffet`) : **première attribution conservée** (l'écriture par slot n'écrase plus une candidate de même génération). Choix motivé : rendre la candidate ambiguë aurait effacé l'effet de l'écran sur la foi d'un mur voisin supprimé au même instant, alors que la publication ne lit une candidate que pour la famille de l'écran ; et la fin n'est plus rattachée à un objet : `finsDEffetLues` (replay) indexe les `effect-end` par la vie de l'effet, tous objets confondus (sans cela la fin rangée sous un objet absent du document était perdue). Tests `TestCandidateRetenueParDeuxObjetsGardeLaPremiere` (deux propriétaires) et `TestEffetFinRetrouveeSousUnAutreObjet`. Mutations : écriture inconditionnelle -> rouge ; fins rangées par objet (`_ = fins`) -> rouge.
3. **Clé de vie recyclée dans le même objet porté** (`situerLesEffets`) : la candidate porte désormais la frame de la suppression qui l'a rendue ; `deCetObjet` ne garde que celles dont la frame est celle de `T1` de CET objet suivi (même esprit que la garde de naissance du repos, `toleranceDeNaissance`). Un objet suivi sans suppression lue n'a aucune candidate. Test `TestEffetCleRecycleeDansLeMemeObjet` (bulle du second lancer à portée du repos du premier : sans garde, attribuée au premier ; deux candidates à portée : pas d'ambiguïté). Mutation : retirer `deCetObjet` -> rouge. Limite connue, non traitée : `o.suivis` est indexé par la clé seule, donc le second lancer d'une clé recyclée dans le même objet n'est pas publié comme second objet suivi (comportement du lot précédent).
4. **Quanta de la candidate non verrouillés** : `TestLireLesEffetsQuanta` (composant d'index 0 uniquement, bits non nuls, chemin dominant, région de la carte ; payload construit bit à bit) et `TestScanEquipementPorteBrancheDeLEffet` (position lue par vie ET instant : deux candidates de même clé ont deux positions, les quanta d'une vie nommée de même clé ne sont jamais les siens). Mutations : `quanta[n.Life]` à la place de `quantaEffets[...]` (lecture et publication) -> rouge ; `k.Index == 0` ou `k.Bits > 0` retirés -> rouge.
5. **Commentaire de `layers.go`** : `equipmentPlacements` lit aussi `FilmInputs.CarriedEquipment` (fin lue).

Gates rejoués : `go vet ./...` rc=0 ; `go test` de `film/...`, `replaydoc`, `replayview`, `replaydiff`, `archlint` rc=0 ; `go test -tags=integration -p 1` sur les mêmes paquets rc=0 ; `go vet -tags research` (grammar) rc=0 ; `openapi-gen -check` rc=0 ; golden de la grammaire regénéré (sources changées, `grammar.Rev` et `SchemaDesFaits` inchangés). Web non touché.
**Document publié** : aucun champ ni contrat ne change ; sur les 5 films des témoins, aucun cas des corrections 2 et 3 n'a été observé (34 effets sur 35, tous à `t1` lu, 0 ambigu), donc les documents sont attendus identiques, MAIS non prouvé sans décodage. Le compte `SpawnedPositioned` des faits persistés change (les films à écrans), jamais le document. Une série de témoins reste à la décision du superviseur.

## Reprise de session

Lire ce plan (cases et journal), reprendre à la première case non statuée de l'étape courante.
