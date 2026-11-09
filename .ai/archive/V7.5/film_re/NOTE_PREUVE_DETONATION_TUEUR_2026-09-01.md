# Preuve terrain : attribution DÉTONATION→tireur des touches explosives confrontée au VRAI TUEUR

Date : 2026-09-01 · Statut : Complété (résultat NUANCÉ, honnête) · Branche : `wt/trame-preuve`

## Ce qu'on cherchait à prouver

Un workflow précédent avait montré que l'attribution DÉTONATION→tireur (viser le point de
détonation d'un projectile explosif, pas la victime) est INTERNE-cohérente et discriminante :
alignement visée→détonation 2-8° pour la thèse contre 61-85° pour un témoin décalé, gain BTB
réel. Mais il ne l'avait JAMAIS confrontée à la vérité terrain — le vrai tueur d'un kill
explosif — parce qu'il récoltait les morts par `geoCollectDamageKills`, qui rend 0 à 5 morts par
film et 0 kill explosif : preuve « PLAUSIBLE non PROUVÉE » (commit `01d98f878`).

## La cause de la famine, et le correctif

`geoCollectDamageKills` ne décode la boucle de records QUE pour les paquets SANS liste
d'événements (`pay[0]&0x40 == 0`). Or les dead-states (les morts, oracle du tueur EnumB) sont
TOUS dans les paquets À ÉVENTS (mesure killsource : 93/93). Ce collecteur sautait exactement les
paquets porteurs de kills.

Le VRAI scan de kills robuste vit dans `internal/games/halo_infinite/film/killsource` (gate (b)
98,2 %, part localisée 95,8-97,6 % — c'est le « 97,6 % » du dead-state, cf.
`filmdec/killhealth.go`). Il importe `filmdec` ET `internal/analysis` : impossible de l'importer
depuis un test `filmdec` (cycle). On a donc PORTÉ son algorithme avec les primitives exportées de
`filmdec`, dans `deto_preuve_robuste_helpers_test.go` (`robustCollectKills`). Quatre éléments que
`geoCollectDamageKills` n'a pas :

1. Timeline chronologique (killsource/world.go) : monde unique, preload de la première
   déclaration de chaque slot, keyframes appliqués DANS L'ORDRE DU TEMPS.
2. Localisateur d'events (killsource/walk.go) : dans un paquet à events la boucle de records ne
   commence pas au bit 2 — signature stricte slot 123 / 35 bits, puis repli largeur libre.
   C'EST le correctif qui débloque les morts.
3. Huit vues de réplication par paquet (Views=8).
4. Snapshot/Restore + filtre `DesyncAt==-1` (records propres seulement) + crédibilité (plage
   bipède dérivée, indices dans le lobby réel, catégorie ≤ 9) + déduplication (victime, instant).

Effet mesuré : le harvest passe de 1-5 morts à 10-26 morts par film, et débloque 4 à 14 kills
EXPLOSIFS appariables par film (morts à détonation source dans un rayon de 8 unités).

## Livrables

- `apps/go-api/internal/analysis/filmdec/deto_preuve_robuste_helpers_test.go` — le scan robuste
  porté (`robustCollectKills`).
- `apps/go-api/internal/analysis/filmdec/deto_preuve_robuste_test.go` — `TestDetoPreuveRobuste` :
  rejoue la mesure M5 (verité terrain) en ne changeant QUE la source des morts, + témoin tireur
  aléatoire.
- `apps/go-api/internal/analysis/filmdec/deto_preuve_finder_test.go` — `TestDetoPreuveFindRockets` :
  balaie `LOT1_CORPUS` et classe les films par tirs de lanceur explosif (moissonner les kills
  explosifs, rares).

## Résultats (28 chunks/film, borne RAM ; identité roster↔FilmIndex apprise des victimes)

Accord au VRAI TUEUR (dead-state EnumB → FilmIndex) des kills explosifs reliés. `these` =
DÉTONATION→tireur géométrique ; `vict` = VICTIME→tireur (la voie réfutée pour le splash) ;
`hasard` = tireur aléatoire (plancher ≈ 1/lobby).

| film | carte | kills robustes (vs grossier) | expl. reliés | injective | these | vict | hasard |
|---|---|---|---|---|---|---|---|
| 000d5950 | cliffhanger | 20 (vs 5) | 7 | oui | 4/5 (80 %) | 4/4 | 14 % |
| 00502e52 | bazaar | 15 (vs 2) | 6 | oui | 4/5 (80 %) | 3/5 | 13 % |
| 01db4132 | forest | 22 (vs 3) | 10 | oui | 3/9 (33 %) | 2/7 | 13 % |
| 3e9967f6 | aquarius | 10 (vs 7) | 7 | oui | 3/4 (75 %) | 4/5 | 19 % |
| 29397c98 | cliffhanger | 11 (vs 4) | 4 | oui | 0/2 (0 %) | 1/3 | 18 % |
| 542e129d | aquarius | 14 (vs 8) | 7 | oui | 0/5 (0 %) | 1/4 | 13 % |
| 08964aeb | aquarius | 21 (vs 7) | 12 | NON (roster 4/8) | 0/2 | 0/2 | 22 % |
| 282ced4c | aquarius | 14 (vs 15) | 2 | NON (roster 4/8) | 0/1 | 0/1 | 18 % |
| 02d39fa0 | forbidden | 21 (vs 4) | 14 | NON (roster 4/8) | 0/3 | 1/3 | 27 % |
| 34dac77d | aquarius | 24 (vs 4) | 9 | NON (roster 6/8) | 0/4 | 0/4 | 15 % |
| 4f77afc1 | BTB flood gulch | 10 (vs 1) | 1 | NON | — (0 évaluable) | — | — |

### Cumuls

- Films à identité INJECTIVE (les 6 premiers) : thèse **14/30 = 47 %**, voie victime 15/28 =
  54 %, hasard ≈ 15 %.
- Films à géométrie/identité FORTE (victime nettement > hasard : 000d5950, 00502e52, 01db4132,
  3e9967f6) : thèse **14/23 = 61 %**, voie victime 13/21 = 62 %, hasard ≈ 15 %.
- Films aquarius/forbidden à identité DÉGRADÉE (injective false, roster 4/8) : TOUTES les voies
  géométriques s'effondrent à ~0 %, y compris la voie victime et près du hasard — signe d'un
  cadre de coordonnées / d'une bijection roster invalide sur ces cartes, PAS d'une réfutation de
  la thèse (le témoin échoue autant qu'elle).

## Verdict

**Forme faible — CONFIRMÉE.** L'attribution DÉTONATION→tireur porte un signal RÉEL sur le tueur :
sur les films à cadre valide elle tombe juste ≈ 47-61 %, soit 3 à 4× le plancher de hasard
(≈ 15 %). Le scan robuste a bien débloqué la vérité terrain que le workflow précédent n'avait
jamais pu confronter (0 → 4-14 kills explosifs par film).

**Forme forte — NON PROUVÉE / réfutée.** Deux résultats interdisent de la survendre :

1. L'accord terrain (~50 %) ne reproduit PAS la cohérence interne (98 %) : confrontée au dead-state,
   l'attribution géométrique est modeste, et elle DÉGRADE quand la scène est dense en roquettes
   (forest : 33 % sur 9 kills — détonations simultanées, mauvais appariement détonation↔mort).
2. La thèse ne bat PAS la voie VICTIME→tireur au niveau des KILLS (47 % vs 54 % ; 61 % vs 62 %
   sur les films forts) : un kill explosif est majoritairement un tir DIRECT où la détonation
   COÏNCIDE avec la victime, donc viser la détonation ≈ viser la victime. La supériorité de la
   thèse mesurée précédemment (2-8° vs 61-85°) porte sur le SPLASH (touches non fatales, M1/M4),
   pas sur le crédit du kill.

Note technique (piste, non un artefact) : l'oracle NAISSANCE (tireur du tir dont la naissance est
la plus proche de la détonation) concorde avec le vrai tueur 6/7 là où il est évaluable — la
détonation EST bien causée par le vrai tueur ; c'est l'étape d'attribution de la détonation à un
tireur PAR LA GÉOMÉTRIE DE VISÉE qui est l'étape lossy (~50 %), pas l'appariement détonation↔kill.

## Reproduire

```
export LOT1_CORPUS=".../data/cache/film_chunks"
# classer les films riches en roquettes
LOT1_CORPUS=$CORP go test ./internal/analysis/filmdec/ -run TestDetoPreuveFindRockets -v
# la preuve sur un film (arène auto-détectable ; forge : LOT1_SONDE_MAP="flood gulch")
LOT1_TRAME_FILM="$CORP/000d5950" LOT1_MAXCHUNKS=28 \
  go test ./internal/analysis/filmdec/ -run TestDetoPreuveRobuste -v
```

## Limites / pistes non traitées (registre)

- Cartes aquarius [13 12 11] / forbidden [13 12 14] : bijection roster incomplète (4/8) et
  géométrie effondrée — à diagnostiquer (précision de layout ? détection de base ?) avant d'y
  fonder une mesure. NON traité ici (hors périmètre preuve).
- BTB (4f77afc1) : 28/63 chunks → identité famélique, 0 kill explosif évaluable. Le BTB exige tout
  le film (RAM) et une bijection roster à marge nulle (killsource : lignes justes en agrégat,
  fausses individuellement) — non exploitable pour cette preuve terrain.
- Densité de roquettes : l'appariement « détonation source la plus proche de la victime » se
  trompe quand plusieurs détonations coexistent ; une contrainte temporelle plus serrée autour de
  l'instant de mort pourrait aider (non tenté — le pic M4 est à -1 s, fenêtre volontairement large).
