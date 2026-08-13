# Investigation TOITS — mesure et critere (2026-08-13)

> Lot 2 du chantier cartes : sur Illusion, Prism et Aquarius le fond rendait le PLAFOND
> (z-buffer « surface la plus haute », rendu.go). Piste du handoff : brancher
> `SurfaceReference`/« bande praticable la plus proche du sol de reference » (reference.go,
> ecrite pour la voie Volume morte) sur le rendu de production.

## 1. La sonde (`TestSondeToits`, sonde_toits_gamefiles_test.go)

Huit cartes cuites avec la voie de reference ARMEE mais non appliquee : les trois validees a
l'oeil (ridgeline, catalyst, va_behemoth) et les cinq jugees defectueuses ou non
satisfaisantes (ctf_illusion, sgh_crystalcaves, ctf_aquarius, ctf_forbidden, chasm). Mesure
par pixel : `zHaut` (surface retenue), `zProche` (surface la plus proche de la reference des
ancres), `ref`.

## 2. Ce que la mesure a REFUTE (ne pas rejouer)

1. **Seuil par pixel sur la hauteur du toit** : Cliffhanger garde 4,7 % de cellules
   « plafond » meme a S=20 m (T<=2) — ses rochers valides. Aucun couple (S, T) ne rend 0 %
   sur ridgeline/catalyst tout en mordant sur les defectueuses.
2. **Part d'ancres couvertes** : Catalyst, VALIDEE, est plus couverte (17/24 ancres sous une
   surface a >=4 m, surplomb median 6,3 m) qu'Aquarius, DEFECTUEUSE (9/22, 2,6 m). Les etages
   superieurs de Catalyst sont des sols joues, pas des toits.
3. (Deja refute au handoff, confirme :) l'ecart median aux ancres ne predit pas le defaut.

## 3. Ce qui SEPARE : la part de matiere qui cache un sol praticable

Definition d'une cellule « toit » : la surface haute domine d'au moins 2 m (`EcartPlafondMin`)
une surface situee a moins de 3 m de la reference (`TolSolReference`). Part sur la matiere du
cadre entier :

| carte | verdict utilisateur | part |
|---|---|---:|
| va_behemoth | nickel (arche comprise) | 19,2 % |
| ridgeline | VALIDEE (rochers = identite) | 25,5 % |
| catalyst | VALIDEE | 28,4 % |
| ctf_forbidden | partiellement touchee | 35,1 % |
| chasm | non satisfaisante | 37,2 % |
| ctf_illusion | defectueuse (toits) | 38,5 % |
| sgh_crystalcaves | defectueuse (toits) | 61,2 % |
| ctf_aquarius | defectueuse (toits) | 66,7 % |

Les deux populations sont disjointes ; `SeuilCarteCouverte = 1/3` passe dans la marge
(28,4 -> 35,1). NB : la part CALCULEE PRES DES ANCRES ne separe pas (catalyst 41,1 % >
forbidden 40,7 %) — c'est la part sur le cadre entier qui separe.

## 4. La regle de production (rendu_reference.go)

1. La cuisson arme la voie de reference : le z-buffer retient AUSSI, par pixel, la surface la
   plus proche du sol de reference interpole des ancres (`ArmeReference`).
2. `AppliqueReference` mesure la part de toits. Carte NON couverte (<= 1/3) : rien ne bouge,
   PNG identique au bit — ridgeline, catalyst, behemoth par construction.
3. Carte COUVERTE : dans la portee des ancres (`PorteeAncre`, 25 m), chaque pixel montre la
   surface la plus proche de la reference ; au-dela, le decor reste rendu par la surface la
   plus haute. Jamais de matiere creee ni supprimee : silhouette et positions jouees
   invariantes par construction (le banc rend les MEMES chiffres, verifie).

Aucun reglage par carte : trois constantes universelles, mesurees ici, documentees dans le
code avec ce tableau.

## 5. Gates joues (2026-08-13)

- Temoins unitaires `rendu_reference_test.go` : 4 verts (couverte -> sol ; non couverte ->
  intacte ; hors portee -> intact ; non armee -> no-op).
- `TestBancCliffhanger` : accord 64,7 % · positions 93,95 % — IDENTIQUES aux references.
- `TestBalayageCoquille` et re-cuisson complete : voir compte rendu du lot / thought_log.
