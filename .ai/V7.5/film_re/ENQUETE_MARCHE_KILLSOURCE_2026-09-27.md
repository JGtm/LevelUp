# Enquête : bascule marche -> scan du killsource (2026-09-27)

Worktree : `C:/Users/Guillaume/Projects/LevelUp-wt-equiv-j51` (détaché). Aucun commit, aucune base
ouverte. Mesure : `TestGoldenFilms -update` puis lecture des lignes `marche` / `scan` des goldens ;
fixtures `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks` (lecture seule).

## Verdict en une phrase

La bascule vient du commit **`f3a2f00eb`** (3.4.1, 2026-09-17 : « la marche des morts calibrée
par la carte »). Elle ne touche que **le banc `TestGoldenFilms`**, qui appelle
`Decode(..., nil)` **sans la carte du match**. Le chemin de production (`replaybuild`,
`killcollector`) passe la carte depuis `6f6e86e6b`, le commit suivant, et n'est pas touché.
Avec la carte passée au banc, la marche **revient** : elle fait même mieux que la référence.

## 1. Bornes vérifiées

| rev | cumul marche | cumul scan | 000d5950 | 78919882 | 9b191a7f | fccc61cd |
|---|---|---|---|---|---|---|
| golden commis (`f14c7e496`) | 334/340 | 29/37 | – | – | 75 prop. | – |
| `f14c7e496` régénéré | **336/342 = 98.2 %** | 27/35 | 86/84 | 93/92 | 76/74 | 87/86 |
| `ea5682373` régénéré | **133/144 = 92.4 %** | 230/233 | 94/91 | 21/17 | 13/10 | 16/15 |

Les colonnes par film donnent « proposés / appariés » de la marche.

Réserve sur la borne basse : à `f14c7e496`, `TestGoldenFilms` n'est **pas strictement vert**.
Les goldens de ce commit sont un simple déplacement (le contenu de `cumul.golden` date de
`3f0ec70b3`, 2026-07-31, via `f2ab431c9`). La régénération montre :
- une dérive de 2 enregistrements (9b191a7f : 75 -> 76 pour la marche) ;
- « morts de BOT » passé de 17/13 à 0/3 ;
- la marge de bijection et une nouvelle section « PROVENANCE DES INDICES ».

Cette dérive est mineure et antérieure. Elle n'a pas été bisectée. Le critère retenu est donc le
**niveau de la marche** (~340 proposés contre ~140), comme le demandait le brief.

## 2. Bisection

**Passe 1 : `git bisect --first-parent`, de `f14c7e496` (bon) à `ea5682373` (mauvais).**

Journal : `scratchpad/bisect1.log`.

| commit | marche cumul | verdict |
|---|---|---|
| 64c5c3bae | 126/130 | mauvais |
| 4fc24674d | 126/130 | mauvais |
| e26047e7e | 336/342 | bon |
| a736579fe | 336/342 | bon |
| bd2819816 | 126/130 | mauvais |
| be694ec31 | 336/342 | bon |
| b3d306f61 | 126/130 | mauvais |
| **c1d3e94f5** (fusion, 2e parent `dcd18e87a`) | 126/130 | premier mauvais |

**Passe 2 : du côté fusionné, `be694ec31` (bon) à `dcd18e87a` (mauvais), sans first-parent.**

Journal : `scratchpad/bisect2.log`.

| commit | marche cumul | verdict |
|---|---|---|
| 248f49750 (base de fusion) | 336/342 | bon |
| 2ec222d30 | 126/130 | mauvais |
| 69796675a | 126/130 | mauvais |
| 181ad7ac7 | 126/130 | mauvais |
| ff71aa295 | 126/130 | mauvais |
| d687cb4d1 | 126/130 | mauvais |
| **f3a2f00eb** | 126/130 | **premier mauvais** (parent `d620a1a56`) |

La bascule se fait **d'un seul coup** : il n'y a qu'une marche.

## 3. Mécanisme

Avant `f3a2f00eb`, `killsource/calibrate.go` **inférait** par balayage, pour chaque film, la
largeur d'axe du chemin absolu de position et la largeur d'index (63 configurations). Il
**appliquait** ensuite le couple retenu, par exemple `axisW=16 indexW=3` sur 9b191a7f. Chaque
film était donc décodé à ses propres largeurs.

`f3a2f00eb` applique l'arbitrage V17 / M3-Q8 (« la valeur LUE prime sur la valeur mesurée ») :
- l'inférence devient un **oracle** qui n'écrit plus rien ;
- le profil appliqué devient `ProfilDeDepart()`, c'est-à-dire l'**invariant** : les largeurs de
  Cliffhanger, 13/13/14, indexW 1 ;
- `Movement.AbsoluteAxisW`, la largeur uniforme de 14, disparaît.

Dans ce commit, **aucune entrée de catalogue n'atteint encore `killsource`**. Les quatre films
sont donc décodés aux largeurs de Cliffhanger :
- **000d5950**, qui est Cliffhanger, n'est pas touché ;
- **78919882** (High Ground, 15/15/17), **9b191a7f** (Bazaar, 17/17/16) et **fccc61cd**
  (Launch Site, 17/17/15) lisent leurs records de bipède à la mauvaise largeur. La marche se
  désynchronise et perd ses records ; le scan, qui ne dépend pas de ces largeurs, les reprend.

`6f6e86e6b`, le commit suivant, ajoute `Options.Carte` et la branche dans `replaybuild` et
`killcollector`. Il laisse en revanche `TestGoldenFilms` / `decoderFixture` sur
`Decode(t.Context(), ..., nil)`.

Le code le prévoit : sans carte, repli nommé `repli_carte_absente_largeurs_par_defaut`, un
avertissement `slog.Warn` par film, et la calibration affiche « DEFAUT (carte absente) ». Le
golden régénéré l'aurait donc dit en clair. Mais les goldens sur film ne tournent pas en CI, et
le commit n'a mesuré que la mini-bobine (qui est un film Cliffhanger) : la chute n'a pas été vue.

## 4. Preuve expérimentale

Modification locale, annulée ensuite : dans `decoderFixture` (golden_test.go), on passe
`opts.Carte = carteDuCatalogue(t, nom)` avec la table film -> carte suivante :

| film | carte |
|---|---|
| 000d5950 | Cliffhanger |
| 9b191a7f | Bazaar |
| 78919882 | High Ground |
| fccc61cd | Launch Site |

Les noms de carte viennent du dépôt : `.ai/` et `oracle_lotB_overtime.tsv`.

**À `6f6e86e6b` (le commit de branchement), carte passée :**
- marche cumul **336/342 = 98.2 %**, scan 27/35 ;
- les quatre films sont **identiques à la référence** `f14c7e496`.
- La calibration affiche `LU [15 15 17] / [17 17 16] / [17 17 15] [CARTE]`.

**À la tête `ea5682373`, carte passée :**
- marche cumul **356/369 = 96.5 %**, scan **7/8**, désaccord entre voies 0, couples réels 371/371 ;
- 9b191a7f : marche 84/80, scan 2 (contre 13/10 et 73 sans carte). Les trois lignes publiées qui
  étaient servies par le scan repassent par la marche, avec **les mêmes tags**.
- Détail par film (marche proposés / appariés) :

| film | marche |
|---|---|
| 000d5950 | 94/91 |
| 78919882 | 98/94 |
| 9b191a7f | 84/80 |
| fccc61cd | 93/91 |

Conclusion : la bascule vient entièrement de la carte absente du banc. Rendre la carte à la
marche la fait revenir, et même au-delà de la référence en couverture.

## 5. Présence à la tête du plan (`feat/suite-audit-decodeur`, `e3672931e`)

- **Oui, dans le banc** : `golden_test.go` appelle toujours `Decode(t.Context(), filepath.Base(dir), src, nil)`.
- **Production non touchée** : `replaybuild/kills.go` pose `opts.Carte = &entry` (repli journalisé
  si la carte est hors catalogue), et `killcollector/collector_run.go` pose
  `opts.Carte = c.carteDuMatch(ctx, matchID)`. Le backfill passe par le collecteur.
- **CLI `cmd/killsource`** : `decfilm.Decode(context.Background(), name, src, nil)`, sans carte.
  Tout film hors Cliffhanger y est décodé aux mauvaises largeurs (repli déclaré au registre, avec
  pour critère de retrait « la CLI et les instruments la résolvent eux aussi »).

## 6. Constats annexes (non traités)

**a. Évolution de la marche après la bascule**, mesurée sur 000d5950, film insensible à la carte.
Journal : `scratchpad/bisect3.log`.

| étape | commit | marche (proposés / appariés) | scan |
|---|---|---|---|
| départ | – | 86/84 | 10 |
| 1 | `297a7c092` (fusion série 5, calibrate.go et decode.go touchés) | 87/85 | 9 |
| 2 | `c9eb3b6bd` (rr-m4b, « la vue B se ferme là où le tir se lit », dans la fusion `2f53b8f1c` puis `569932b42`) | **94/91** | 2 |

C'est une croissance de la marche (à suivre : 3 non-appariés sur 94). Elle explique l'écart entre
la tête avec carte (356/369) et la référence (336/342).

**b. `ProfilDeDepartPourCarte` signale à tort le repli pour Cliffhanger.** Le code fait :

```go
return p, p.LargeursObjetDuMonde() != avant
```

Pour la carte **Cliffhanger** (l'invariant), `CarteLue` vaut donc faux **même quand la carte est
passée**. Conséquences :
- avertissement « carte du match absente » à tort ;
- repli compté à tort ;
- « DEFAUT (carte absente) » affiché dans la calibration de 000d5950 avec carte.

Le défaut est présent à la tête du plan. Il est cosmétique pour le décodage (mêmes largeurs), mais
il fausse le compteur du repli.

**c. L'oracle ne voit plus rien à la tête.** Sur les trois cartes non invariantes, il répond
« PROFIL PLAT », contre des profils nets à `6f6e86e6b` (score 170 à 1131). Sur 000d5950, il dit 23
contre 13/13/14 (désaccords=1). Le juge interne censé détecter une entrée de catalogue fausse est
donc devenu aveugle. Non instruit.

## 7. Pistes de correction (non codées)

1. **Banc** : donner la carte à `decoderFixture`. Par exemple, un champ `carte` dans `references`
   (killsource_test.go) résolu par `carteDuCatalogue`, sur le modèle de
   `TestMotDePoigneeDeterministeSurFilm`. Ensuite, régénérer les goldens et relire le diff (la
   dérive annexe du §1 et les constats 6a et 6c y apparaîtront).
2. **Garde-rail** : faire échouer `TestGoldenFilms` si une référence décode avec
   `!Calibration.CarteLue`, pour qu'un banc sans carte ne passe pas en silence.
3. **CLI `cmd/killsource`** : résoudre la carte (catalogue + nom passé en argument ou lu en base),
   comme l'exige le critère de retrait du repli.
4. **`CarteLue`** : le définir par « une entrée de catalogue a été fournie et porte des largeurs »,
   et non par « les largeurs ont changé ».
5. **Décision de fond** (question au pilote, déjà ouverte dans le registre) : rendre la carte
   obligatoire à `Decode`, c'est-à-dire mettre de côté un film dont la carte est inconnue plutôt
   que le décoder aux largeurs d'une autre. Cela supprimerait la classe de défaut.

## 8. Risques

- **Production** : un match dont la carte n'est pas résolue (nom absent, carte hors catalogue)
  est décodé aux largeurs de Cliffhanger. La marche s'effondre et le scan prend le relais, ce qui
  est contraire à la règle « flux seul ». Le repli est journalisé mais **publie quand même**.
  L'ampleur en base n'a pas été mesurée (aucune base ouverte).
- **Fenêtre historique** : `f3a2f00eb` sans `6f6e86e6b` n'a existé que sur la branche
  `feat/decfilm-341`, pendant environ 40 minutes, et les deux sont entrés ensemble par
  `dcd18e87a`. Pas d'exposition en production attendue de ce fait.
- **Régénération** : régénérer les goldens sans corriger d'abord le banc figerait la régression
  comme référence.

## 9. État final

- `git bisect reset` fait.
- Checkout remis sur `ea5682373` détaché.
- `git status --short` : seul `?? .bin-audit/` apparaît, déjà présent avant l'enquête et hors de
  mon périmètre.
- Les deux modifications expérimentales de `golden_test.go` et les goldens régénérés ont été
  annulés (`git checkout -- .`).
