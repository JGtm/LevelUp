# RAPPORT R3 — Le filtre MaxSpeedMPS=100 : coût mesuré sur les téléportations, et remplacement proposé

Date : 2026-09-03. Lot R3 du plan `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md`.
Recherche pure, lecture seule : aucun fichier de production touché. Instrument (3 fichiers,
env-gatés `VITESSE_*`, skip par défaut, `CGO_ENABLED=0`, `LockProcessDecode`, fenêtres
bornées par chunks) :

- `apps/go-api/internal/analysis/filmdec/vitesse_filtre_research_test.go` (test + rapport)
- `apps/go-api/internal/analysis/filmdec/vitesse_filtre_outils_research_test.go` (simulateur, corroboration, catalogue)
- `apps/go-api/internal/analysis/filmdec/vitesse_filtre_artefact_research_test.go` (lecture de l'artefact publié)

`go vet ./internal/analysis/filmdec/` vert. Toutes les commandes rejouables en annexe.

## Verdict en trois phrases

**La prémisse du lot était à moitié fausse : le rejet en cascade « jusqu'à la fin de la vie »
n'existe pas — `DropTeleports` porte un réancrage aveugle après `maxRejectStreak=3` rejets
(offline_filters.go:89, offline_biped.go:144), et la production perd donc AU PLUS 3
échantillons bruts par téléportation (49 ms en médiane, 149 ms au pire), jamais l'arrivée
elle-même.** **Dans l'artefact publié, sur les 18 téléportations datées par l'événement 117 :
aucun saut perdu, arrivée publiée à +1 frame (15 cas) ou +0 (3 cas) de la frame de
l'événement — c'est la grille de 100 ms, pas le filtre — et le seul effet imputable au filtre
est un déplacement de 0,00 à 0,20 m du premier point d'arrivée publié.** **Pour R3.2, la
comparaison chiffrée donne l'option A (filtre levé à ±200 ms d'un événement 117 du même
slot) : couverture 51/51 rejets à tort, 0 fausse exemption, 0 changement sur les films
témoins (0 tête 117 sur 34 115 et 36 203 paquets delta) — l'option B (réancrage corroboré)
récupère autant mais MODIFIE les films sans translocateur (5 points sur `7344d24f`), ce que
le cahier des charges interdit.**

## 0. Méthode, et les pièges rencontrés (pour ne pas les refaire)

1. **Datation** : les 18 téléportations sont RE-DÉRIVÉES du film lui-même (recensement des
   têtes de type 117, lecture O(1) par paquet — même canal que R1 §4), pas recopiées à la
   main. Contrôle : les inventaires paquets delta / têtes 117 par film sont IDENTIQUES à la
   table R1 §4.1/4.3 (33 571/3, 46 534/4, 34 246/6, 37 598/2, 33 042/3).
2. **Fenêtres** : décodage sans filtre (`MaxSpeedMPS=0`, `IsolationGapMS=0`) des SEULS chunks
   couvrant [événement-5 s, événement+10 s] ; fenêtres qui se recouvrent ou partagent un
   chunk fusionnées en un seul décodage (sinon le chunk commun est décodé deux fois et les
   arrivées d'un groupe comptent comme bruit dans l'autre — bug vu et corrigé sur
   `1b2d9e08`, chunks 17-18/18-19).
3. **PIÈGE MAJEUR — le champ `bounds` de l'artefact est un CADRAGE D'AFFICHAGE** (étendue
   des pistes), PAS l'AABB de déquantification. Une première exécution l'a utilisé : tout
   était compressé ~10x (saut de 22,1 m rendu 2,05 m, calibrations 6,8-25,8 m). Les
   instruments R1 (`FAILLE_BOUNDS`) ont reçu ces mêmes bornes d'affichage — sans
   conséquence pour leurs verdicts (leurs critères spatiaux étaient exprimés dans le même
   repère des deux côtés), mais tout futur instrument qui veut des MÈTRES VRAIS doit
   déquantifier avec `map_quant_bounds.json` (l'entrée de la production, `MapQuantEntry`).
4. **Identification de carte** : les cartes des 4 films témoins n'étant documentées nulle
   part (et les DuckDB interdits), l'instrument identifie l'entrée de catalogue par
   CALIBRATION : parmi les entrées au même découpage i0 que le film, celle qui minimise
   l'écart médian 2D entre positions déquantifiées et piste publiée d'avant-saut. Écarts
   obtenus : 0,004-0,030 m sur 30-90 paires par film — la déquantification est au
   centimètre. (Les cartes Forge partagent un même canevas : à bornes identiques le NOM est
   indécidable — `fo13_frost`/`fo08_wetland` etc. — et sans effet sur la mesure ;
   `f2966f08` est identifiée uniquement : Behemoth, module `va_behemoth`.)
5. **Simulation** : la sémantique de `DropTeleports` est rejouée décision par décision
   (ancre par slot, rejet si vitesse > 100 m/s depuis la dernière position ACCEPTÉE,
   réancrage aveugle après 3 rejets), après `DropIsolated(15 s)` comme en production.
   Limite assumée : l'ancre d'un slot à l'entrée de fenêtre peut différer de celle de la
   production film-entier — les 5 s de trajectoire dense avant chaque événement l'égalisent.
6. **« Rejeté à tort »** = rejeté par la production ET corroboré : au moins 2 échantillons
   suivants du slot s'enchaînent sous 100 m/s depuis le point (une aberration du balayage
   est un point sans suite ; une vraie arrivée est suivie de toute la trajectoire).

## 1. Ce que la production fait vraiment (lu sur pièces, pas supposé)

`ScanFilmBipedPositions` applique `DropIsolated(15 000 ms)` puis `DropTeleports(100 m/s)`
(offline_biped.go:208-212). Deux faits que le lot devait établir :

- **La cascade est BORNÉE** : après 3 rejets consécutifs d'un slot, le 4e échantillon est
  accepté SANS condition (`maxRejectStreak=3`, « sinon une ancre elle-même fausse
  condamnerait tout le reste du slot »). Le scénario « la précédente reste l'ancienne
  position, donc les suivants sont rejetés jusqu'au bout » ne se produit jamais au-delà de
  3 échantillons.
- **Les aberrations ISOLÉES ne sont pas l'affaire de ce filtre** : les 7 aberrations
  historiques de `000d5950` (motivation du filtre, cf. en-tête offline_filters.go) sont
  isolées de 66 à 320 s — c'est `DropIsolated` qui les tue, en amont. `DropTeleports` ne
  protège que contre les aberrations IMBRIQUÉES dans une trajectoire vivante — qui ont,
  par construction, une profondeur de corroboration 0-1.

À la cadence d'émission réelle mesurée (17 ms médians partout, ~60 Hz), 100 m/s = un seuil
de 1,7 m PAR PAS : toute téléportation, même de 3 m, est rejetée (la plus petite datée,
3,09 m, part à 193 m/s).

## 2. R3.1 — la table des 18 téléportations

Colonnes : saut mesuré au film brut (pas consécutif, 3D) ; rejets de production dans la
zone d'arrivée [-0,5 s, +2 s] (tous corroborés, tous à ±200 ms de l'événement) ; retard du
1er point accepté par la production après l'arrivée réelle ; côté artefact : frame de
l'événement, retard de la première position publiée à <= 2 m de l'arrivée, pas maximal
publié, trous de frames à ±10, déplacement du 1er point publié vs l'arrivée réelle.

| # | film | évén. (film) | slot | saut film | v (m/s) | rejets | retard prod | frame | arrivée art. | pas art. | trous | dépl. |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 1b2d9e08 | 185 184 ms | 535 | 23,19 m/17 ms | 1372 | 3 | +149 ms | 1761 | +1 fr | 22,13 m @+1 | 1 [1766] | 0,00 m |
| 2 | 1b2d9e08 | 335 257 ms | 560 | 3,09 m/16 ms | 193 | 1 | +16 ms | 3261 | +1 fr | 3,24 m @+1 | 0 | 0,05 m |
| 3 | 1b2d9e08 | 350 989 ms | 560 | 8,10 m/17 ms | 470 | 3 | +50 ms | 3419 | +1 fr | 7,84 m @+1 | 0 | 0,09 m |
| 4 | a0c36016 | 72 799 ms | 520 | 9,12 m/17 ms | 540 | 3 | +49 ms | 690 | +1 fr | 8,88 m @+1 | 0 | 0,08 m |
| 5 | a0c36016 | 183 777 ms | 534 | 16,08 m/17 ms | 937 | 3 | +50 ms | 1800 | +1 fr | 16,25 m @+1 | 0 | 0,20 m |
| 6 | a0c36016 | 302 681 ms | 555 | 25,31 m/16 ms | 1540 | 3 | +50 ms | 2989 | +1 fr | 25,36 m @+1 | 0 | 0,09 m |
| 7 | a0c36016 | 659 247 ms | 623 | 21,01 m/17 ms | 1239 | 3 | +50 ms | 6554 | +1 fr | 20,91 m @+1 | 3 [6560-62] | 0,11 m |
| 8 | 4577fcc4 | 77 616 ms | 514 | 9,09 m/17 ms | 537 | 3 | +49 ms | 711 | +1 fr | 9,01 m @+1 | 0 | 0,09 m |
| 9 | 4577fcc4 | 80 925 ms | 514 | 8,66 m/17 ms | 513 | 3 | +49 ms | 744 | +1 fr | 8,39 m @+1 | 0 | 0,09 m |
| 10 | 4577fcc4 | 91 318 ms | 514 | 4,56 m/17 ms | 271 | 2 | +33 ms | 848 | +1 fr | 4,50 m @+1 | 0 | 0,09 m |
| 11 | 4577fcc4 | 517 735 ms | 591 | 8,37 m/17 ms | 502 | 3 | +50 ms | 5112 | +1 fr | 8,26 m @+1 | 0 | 0,11 m |
| 12 | 4577fcc4 | 523 658 ms | 591 | 8,49 m/17 ms | 508 | 3 | +49 ms | 5172 | +0 fr | 8,47 m @+0 | 0 | 0,16 m |
| 13 | 4577fcc4 | 528 413 ms | 591 | 14,76 m/17 ms | 874 | 3 | +49 ms | 5219 | +1 fr | 14,68 m @+1 | 0 | 0,12 m |
| 14 | f2966f08 | 304 847 ms | 555 | 12,25 m/17 ms | 721 | 3 | +49 ms | 2943 | +1 fr | 12,20 m @+1 | 0 | 0,10 m |
| 15 | f2966f08 | 614 570 ms | 610 | 7,72 m/17 ms | 444 | 3 | +49 ms | 6040 | +1 fr | 7,47 m @+1 | 0 | 0,17 m |
| 16 | faff9935 | 155 106 ms | 533 | 18,95 m/17 ms | 1126 | 3 | +49 ms | 1345 | +1 fr | 18,90 m @+1 | 0 | 0,01 m |
| 17 | faff9935 | 245 330 ms | 551 | 10,80 m/17 ms | 631 | 3 | +49 ms | 2248 | +0 fr | 9,81 m @+0 | 0 | 0,10 m |
| 18 | faff9935 | 252 623 ms | 551 | 17,99 m/20 ms | 915 | 3 | +47 ms | 2321 | +0 fr | 17,93 m @+0 | 0 | 0,14 m |

Notes de lecture :

- Les cas « +0 fr » (12, 17, 18) sont des événements tombés juste avant une frontière de
  frame : le premier échantillon ACCEPTÉ de l'intervalle est déjà l'arrivée. Sur ces
  lignes, la colonne « calibration » de l'instrument affiche la distance au point publié
  d'AVANT (déjà post-saut) — 8,45/9,96/17,95 m — ce n'est PAS une décalibration
  (l'identification de carte reste à 0,004-0,030 m).
- Ligne 1 : le retard de +149 ms combine 3 rejets (+0/+17/+33 ms) et une PAUSE D'ÉMISSION
  du slot (~100 ms) — le réancrage a eu lieu au premier échantillon disponible.
- « rejets slot hors zone » non nuls sur les lignes 8, 9, 11, 12 (3 chacun) sont les rejets
  des téléportations VOISINES du même slot dans la même fenêtre (zones distinctes) — ce ne
  sont pas des rejets supplémentaires.

## 3. Agrégats (18 téléportations, 5 films)

- **Échantillons bruts** : cadence médiane 17 ms sur les 18 fenêtres (380-899 échantillons
  du slot par fenêtre de 15 s).
- **Rejets à tort** : total **51** ; médiane **3**, pire **3** (la borne `maxRejectStreak`),
  minimum 1 (le saut de 3,09 m, sous le seuil dès le 2e pas). 51/51 à ±200 ms de
  l'événement. Aucun rejet non corroboré dans les zones d'arrivée.
- **Retard du 1er point de production** : médiane **49 ms**, pire **149 ms** — soit 3
  échantillons à 60 Hz ; jamais une arrivée perdue.
- **Artefact publié** : retard d'arrivée médian **+1 frame** (le minimum de la grille :
  `decimateTracks` publie « le premier observé par frame », l'arrivée ne peut apparaître
  avant la frame suivant le saut), pire +1 ; **aucun saut absent** ; déplacement du premier
  point d'arrivée publié **0,00-0,20 m** (l'effet net du filtre : le point de la frame
  d'arrivée est pris 33-149 ms plus tard que le premier échantillon réel, le long du chemin
  du joueur) ; **trous de frames** : 4 au total (frames 1766 et 6560-6562), tous à
  >= +500 ms du saut alors que les rejets s'épuisent à +50 ms — ce sont des pauses
  d'émission du film, PAS des effets du filtre.

**Conclusion R3.1 : le coût de production du filtre est réel mais borné — 1 à 3 échantillons
bruts (16-149 ms) par téléportation — et il est INVISIBLE dans l'artefact à l'exception d'un
déplacement <= 0,20 m du premier point d'arrivée.** Le retard « +1 frame » constaté est
celui de la grille de 100 ms, qu'il y ait filtre ou non.

## 4. Découverte latérale : les relocalisations de masse (débuts de manche)

Les rejets corroborés HORS zones d'arrivée ne sont PAS du bruit : ce sont des
relocalisations simultanées de tous les bipèdes (transitions de manche) — mêmes
mécaniques de rejet, même réancrage aveugle en <= 3 échantillons :

- `a0c36016` : 7 slots (513-521) rejetés au même instant @66 592 ms, profondeur 8/8 ;
- `f2966f08` : 5 slots (608-613) @600 139 ms, profondeur 8/8 ;
- `7344d24f` (témoin SANS translocateur) : 5 slots (532-537) @160 004 ms, profondeur 8/8.

Les VRAIES aberrations du balayage (profondeur 0-1) dans toutes les fenêtres mesurées :
**1 seule** sur ~224 000 échantillons décodés sans filtre (les gardes amont — bande de
slots, RequireTag1, DropSaturated, DropIsolated — font déjà l'essentiel du travail). Dans
ces fenêtres, le filtre a rejeté 68 positions réelles pour 1 aberration.

## 5. R3.2 — comparaison chiffrée des remplacements (PROPOSITION, décision D2 user)

### Option A — « le film fait foi » : filtre conservé SAUF à ±200 ms d'un événement 117 du même slot

| mesure | valeur |
|---|---|
| rejets à tort couverts | **51/51** (rejets zone restants : 0 sur 18/18) |
| fausses exemptions (points exemptés non corroborés) | **0** |
| films témoins (`696a9d7c`, `7344d24f`) | **0 tête 117** sur 34 115 / 36 203 paquets delta -> le filtre n'est JAMAIS levé : rien ne change, par construction |
| ce qu'elle ne couvre pas | (a) une téléportation dont l'événement n'est pas en TÊTE de liste (réserve R1 §4.3 : 3 `spent` sans événement, AUCUN avec téléportation mesurée — rien de perdu de mesurable) ; (b) les relocalisations de masse (autre mécanisme, pas d'événement 117) : elles restent filtrées comme aujourd'hui |
| coût de mise en œuvre | le lecteur d'événements 117 (brique `ScanFilmTranslocatorTeleports` déjà cadrée, R1 §6) + passer les instants au filtre |

### Option B — réancrage par corroboration (un rejet devient une ancre si les k suivants s'enchaînent depuis lui)

| mesure | valeur |
|---|---|
| k mesuré (pas choisi) | arrivées : profondeur de corroboration **8/8 (plafond) sur 18/18** ; l'unique aberration observée : profondeur 0 ; les aberrations imbriquées que le filtre vise sont par construction à profondeur 0-1 (la trajectoire continue au vrai endroit). **k=2** les sépare avec la marge maximale |
| rejets à tort couverts | 51/51 (dès k<=8), et le point d'arrivée LUI-MÊME est publié (déplacement résiduel 0 au lieu de 0,00-0,20 m) |
| risque de réancrage sur bruit | 0 réancrage sur les ~224 000 échantillons mesurés (1 seule aberration, profondeur 0) — mais le dénominateur d'aberrations est FAIBLE : la population qui a motivé le filtre (7 sur `000d5950`) est tuée en amont par DropIsolated, celle qui resterait n'a presque pas d'occurrences dans ces fenêtres |
| films sans translocateur | **B LES MODIFIE** : sur `7344d24f`, 5 points de relocalisation de manche @160 004 ms seraient publiés (positions réelles, mais changement quand même). L'exigence « ne rien changer » n'est PAS satisfaite |

### Conclusion proposée

**Option A.** Elle répond exactement à la doctrine du chantier (aucune heuristique :
l'exemption est déclenchée par un ENREGISTREMENT du film, l'événement 117), couvre 51/51
rejets mesurés avec 0 faux positif, et son invariance sur les films sans translocateur est
prouvée par dénominateur. L'option B est documentée comme alternative générale : elle
couvrirait AUSSI les relocalisations de masse et d'éventuelles téléportations sans
événement lisible, au prix d'un paramètre k (même mesuré, c'est un choix) et d'un
changement de comportement sur TOUS les films. Si un jour on veut publier les
relocalisations de manche, B (k=2) est la voie mesurée.

À dire pour peser la décision : le bénéfice VISIBLE de A dans l'artefact est minime
(déplacement <= 0,20 m du premier point d'arrivée, 0 frame de retard) ; son vrai bénéfice
est d'amont — des positions brutes exactes autour des sauts pour les consommateurs futurs
(datation/dessin des téléportations D4, appariements kill-position, marqueurs d'usage).

## 6. Statut des items du plan

- R3.1 : **fait** — 18/18 téléportations mesurées (artefact + film sans filtre), table §2,
  agrégats §3, imputabilité des trous §3.
- R3.2 : **fait — proposition seulement** (aucune application, décision D2 user) : §5,
  options A et B chiffrées, invariance témoin prouvée pour A, k mesuré pour B.

## Annexe — commandes exactes (depuis `apps/go-api` du worktree ; `<repo>` = LevelUp-go-migration)

Les 5 films de mesure (l'instrument identifie la carte par calibration contre l'artefact) :

```
CGO_ENABLED=0 VITESSE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
  VITESSE_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
  VITESSE_ARTEFACT=<repo>/data/cache/replays/halo_infinite/1b2d9e08.json \
  go test ./internal/analysis/filmdec/ -run '^TestVitesseFiltre$' -timeout 30m -v
```

(identique pour `a0c36016`, `4577fcc4`, `f2966f08`, `faff9935` — remplacer l'id dans
`VITESSE_FILM` et `VITESSE_ARTEFACT`.)

Les 2 témoins sans translocateur (recensement 117 film entier = preuve option A ; option B
mesurée sur 3 chunks, carte donnée — Strongholds Vagabond) :

```
CGO_ENABLED=0 VITESSE_FILM=<repo>/data/cache/film_chunks/696a9d7c \
  VITESSE_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
  VITESSE_CARTE=vagabond VITESSE_CHUNKS=8,9,10 \
  go test ./internal/analysis/filmdec/ -run '^TestVitesseFiltre$' -timeout 30m -v
```

(identique pour `7344d24f`.)

Gate : `CGO_ENABLED=0 go vet ./internal/analysis/filmdec/` — vert le 2026-09-03.
