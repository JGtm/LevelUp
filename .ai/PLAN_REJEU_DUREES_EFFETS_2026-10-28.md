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

- [ ] B.0 Témoins AVANT (règle `feedback_temoins_avant_apres`) : sur 5 films (1ea71def, 52fc79ef, cba488f7,
  f59c9332, 5676a9ba), documents Go complets (T-Go) et sortie du script de caractérisation web du lot précédent
  (T-Web : faille, calendrier complet des sons, poses d'origine, fenêtres de toutes les poses), capturés depuis le
  code de la release AVANT toute modification, rangés dans le scratchpad `durees_effets/temoins/avant/`.
- [ ] B.1 Mur, fin lue. Établir sur pièces le LIEN STRUCTUREL entre la pose de mur publiée (`equipmentPlacements`,
  origine `deployed`, issue du 103) et l'objet suivi du mur (`carriedEquipment[].tracked[]`, dont `t1` est la
  suppression écrite par le film) : clé de vie, référence du 103 (`Source` / `Spawned`), porteur. LA GRAMMAIRE PRIME :
  une jointure par proximité temporelle n'est admise que si aucun lien structurel n'existe, et alors bornée, comptée
  (couverture : liées / sans lien / ambiguës) et justifiée au journal. Publier côté Go la fin lue sur la pose de mur
  (champ dédié, schéma 106, calculé à la publication depuis les faits) ; web : la fenêtre du mur s'arrête à cette
  fin quand elle est lue. Repli sans fin lue : `WALL_DURATION_MS` passe de 10 s à la médiane mesurée de la
  barrière (12 480 ms, Résultats A), commentaire mis à jour (source = mesure du film, pas « une dizaine »).
- [ ] B.2 Écran occultant, objet d'effet. Dans la grammaire de production (canal de l'équipement porté), lire la vie
  ti=38 née à l'impact de chaque objet suivi d'écran (naissance, suppression, position à la création). LIEN
  STRUCTUREL d'abord (composant du record de création de la ti=38 qui désigne la grenade, le porteur ou l'objet
  porté) ; à défaut règle d'association bornée (fenêtre et distance tirées des Résultats A), comptée et nommée
  (repli `fallback` si le projet l'exige pour un choix non lu). Révision de la couche concernée (grammaire et/ou
  faits, `SchemaDesFaits`), doc : `tracked[].effect {t0, t1, x, y, z}` (absents ensemble quand non lus), couverture
  et ligne slog. Web : l'écran se dessine de `effect.t0` à `effect.t1` à la position de l'effet ; sans effet, règle
  actuelle (repos de la grenade + `SHROUD_DURATION_MS`). Un écran sans trajectoire mais avec effet se dessine.
  Ne pas brancher le mur sur cet effet (B.1 suffit) sauf si B.1 le rend nécessaire, et alors le noter.
- [ ] B.3 Tests : Go (lien ou association, bords de fenêtre, clé recyclée, absence), web (fenêtres mur et écran,
  repli), mutations vérifiées (inverser la condition fait rougir un test). Goldens et fixtures régénérés par les
  outils du projet (jamais à la main), `make generate-types` si l'OpenAPI bouge. Plafonds de taille : ne pas les
  relever sans justification datée.
- [ ] B.4 Témoins APRÈS sur les 5 mêmes films : T-Go identique hors champs neufs et fins de mur ; T-Web : faille,
  sons, début et tracé du mur identiques à l'octet ; tableau des fins de mur et d'écran avant/après par film.
- [ ] B.5 Gates : `go vet ./...`, `go test ./...` et `go test -tags=integration -p 1` sur les paquets du film
  touchés (depuis `apps/go-api`, `GOCACHE=%LOCALAPPDATA%\go-build-durees-effets`), `npx tsc -b --force`, eslint,
  vitest `src/features/match-replay src/lib/replay src/lib/api src/lib/i18n`, knip. Codes de sortie relevés.
- [ ] B.6 Changelog et notes de version EN + FR (une ligne chacun), référence des canaux d'équipement
  (`.ai/archive/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`), entrée `.ai/thought_log.md`. NE PAS commiter :
  le superviseur relit, fait la revue adversariale et commite. Recuisson du parc : NE PAS lancer (le superviseur
  la demande au user).

Gate B : B.0 à B.6 statués, gates verts, témoins conformes aux invariants. Compte rendu au superviseur.

## Découvertes (non traitées)

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

## Reprise de session

Lire ce plan (cases et journal), reprendre à la première case non statuée de l'étape courante.
