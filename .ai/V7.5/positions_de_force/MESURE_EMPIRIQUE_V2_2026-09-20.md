# Mesure empirique v2 des positions de force — 2026-09-20

> Item 2bis.B du plan `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md` (décisions D10a, D10b,
> D11, D12). Branche `wt/power-positions`. Outil : `apps/go-api/cmd/mappower-build
> --mesure --reglage v2`, paquet pur `apps/go-api/internal/analysis/powerpos`
> (`ReglageV2`, `angles.go`, `ponderation.go`, `morphologie.go`, `diagnostic.go`).
> Données lues en SEULE lecture sur `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
> (`duckdb.OpenReadForQuery`, vues `_latest` uniquement, aucune écriture).
> **`ReglageV1` est intact** (preuve datée du verdict v1, D12) ; la v2 est un réglage NEUF.
>
> Commandes exactes :
>
> ```
> cd apps/go-api
> # calibrage (trois cartes, sans l'oracle) — rapports conservés sous mesures_v2_2026-09-20/_calibrage_v{1,2}/
> go run ./cmd/mappower-build --mesure --reglage v1 --data-root <racine> \
>   --sortie .ai/V7.5/positions_de_force/mesures_v2_2026-09-20/_calibrage_v1 --cartes "recharge,aquarius,streets"
> go run ./cmd/mappower-build --mesure --reglage v2 --data-root <racine> \
>   --sortie .ai/V7.5/positions_de_force/mesures_v2_2026-09-20/_calibrage_v2 --cartes "recharge,aquarius,streets"
> # passe complète (12 cartes de la v1)
> go run ./cmd/mappower-build --mesure --reglage v2 --data-root <racine> \
>   --sortie .ai/V7.5/positions_de_force/mesures_v2_2026-09-20 \
>   --cartes "live fire,recharge,streets,aquarius,forbidden,origin,lattice,solitude,fortress,empyrean,illusion,bazaar"
> ```
>
> `--exclure-variantes` vaut « Live Fire - Ranked » par défaut (voir « Live Fire » ci-dessous).

## Ce que la v1 n'avait pas, et pourquoi (rappel du verdict)

Le verdict v1 (`VERDICT_ORACLE_2026-09-20.md`, §6-§7) a manqué 13 zones fortes : 6 par
**fragmentation** (des cellules passaient le seuil mais le plus grand amas 4-connexe faisait 2
à 10 cellules pour 12 exigées), 6 par **score sous le seuil** (dont deux à moins de 0,01 du p90
de leur carte), 1 non scorable. Et les positions retenues tombaient dans les bases et les
points de défense : le rapport de duel mesure OÙ l'on gagne, pas COMMENT le lieu est fait.

La v2 ajoute trois choses de nature différente : le **rang du tueur** (D10a), deux **signaux
angulaires** tirés des positions tueur/victime (D10b), et une **sélection par hystérésis +
fermeture morphologique** à la place du seuil unique + 4-connexité + 12 cellules.

## 1. Le rang des joueurs — où il vit, et sa couverture

Recensement sur pièces (migrations `internal/games/halo_infinite/migrations/`, skill
`db-schema`) : **une seule source par match ET par joueur**, la vue `match_csrs_latest` de la
base PARTAGÉE (clé `(match_id, xuid)`, colonne `rating_value`). Les autres candidates ne
répondent pas à « quel était le rang de CE tueur ce jour-là » :

| Source | Pourquoi non |
|---|---|
| `match_skill_rank_latest` (base joueur) | pas de colonne `xuid` : c'est le rang du propriétaire de la base, jamais celui d'un adversaire |
| `player_csr_snapshots_latest` | état par playlist et saison, sans `match_id` |
| `player_skill_state_v2_latest` (partagée) | état COURANT par (xuid, groupe de playlist) : le rang d'aujourd'hui, pas celui du match |
| `match_participants` | aucun CSR : `rank` = classement de fin de partie, `team_mmr` / `enemy_mmr` = moyennes d'équipe |

`match_csrs` n'est alimentée que par les matchs CLASSÉS : en playlist sociale personne n'a de
rang. **La couverture est donc structurellement partielle**, mesurée sur les trois cartes de
calibrage (`_calibrage_v1/_rapport.md`) :

| Carte | Kills lus | Kills à rang connu | part | Matchs de la carte | avec ≥ 1 rang |
|---|---|---|---|---|---|
| recharge | 7 344 | 3 442 | **47 %** | 888 | 686 (77 %) |
| streets | 6 950 | 2 666 | **38 %** | 730 | 542 (74 %) |
| aquarius | 4 945 | 926 | **19 %** | 579 | 406 (70 %) |

Rampe mesurée sur les **28 937 rangs de tout le titre : p10 1 251, médiane 1 442, p90 1 590**
(échelle CSR). Pendant le calibrage, l'assiette des quantiles était restreinte aux matchs
des cartes demandées (12 051 rangs sur 1 634 matchs : 1 241 / 1 437 / 1 583) — ce qui
faisait dépendre le poids d'un kill de la liste `--cartes`, donc de la passe. Corrigé le
2026-09-20 14:20, AVANT tout verdict, en gardant le réglage figé tel quel (la rampe n'en
fait pas partie : elle se mesure « sur le corpus », et le corpus est désormais le titre).
Effet mesuré sur les trois cartes de calibrage : identique sur streets et aquarius ; sur
recharge, une composante de 9 cellules passe à 12 et devient une 4e position (7,8 m²) —
c'est la sensibilité de la taille minimale, notée plus bas. Sur les 12 cartes, voir la
section « Passe v2 » ci-dessous.

**Ce qu'on en fait (D10a)** : `KillSample.RangTueur *float64` (nil = inconnu), et une
**rampe** de poids (`powerpos.Ponderation`) : 0,5 au p10 du corpus, 1,5 au p90, linéaire
entre, bornée au-delà ; **rang inconnu = poids 1,0**. Le poids neutre n'est pas un défaut par
dépit : un poids bas effacerait les playlists sociales, un poids haut les privilégierait.
Symétrique autour de 1,0, la rampe penche le corpus sans le déplacer en moyenne. Le
retrécissement de l'avantage reste compté en engagements BRUTS (un poids ne crée pas une
observation). La variante « tueurs au-dessus de la médiane » (`kills_tueur_fort`,
`morts_tueur_fort`) est COMPTÉE au CSV, mesurée à part, pas mélangée au score.

## 2. Les signaux angulaires — ce qu'ils montrent sur les trois cartes de calibrage

Par cellule, deux sommes vectorielles de directions unitaires (`SommeAngulaire{Cos, Sin, N}`,
additives donc lissables sur le même disque de 2 m) :

- **exposition** = dispersion des directions cellule → tueur, pour les morts subies dans la
  cellule ; **abri = 1 − exposition** ;
- **couverture** = dispersion des directions cellule → victime, pour les kills partis de la
  cellule.

Dispersion = 1 − résultante moyenne, **corrigée du biais de petit échantillon** :
`sqrt(max(0, (n·R² − 1) / (n − 1)))` — sous l'isotropie, R² brut vaut 1/n en espérance, ce
qui ferait passer une cellule à 10 morts pour un abri (R ≈ 0,32). Tests purs :
`angles_test.go` (uniforme, concentré, opposés, correction du biais sur 400 tirages, vide,
additivité).

Mesuré par DISQUE scorable (`_calibrage_v1/_rapport.md`, section « Signaux angulaires ») :

| Carte | Disques | N sortant p10/p50/p90 | N entrant p10/p50/p90 | couverture p10/p50/p90 | abri p10/p50/p90 | (couv + abri)/2 p10/p50/p90 | corr(couv, abri) | corr(couv, avantage) | corr(abri, avantage) | corr(asym, avantage) |
|---|---|---|---|---|---|---|---|---|---|---|
| recharge | 1 915 | 68 / 121 / 216 | 71 / 120 / 202 | 0,53 / 0,71 / 0,91 | 0,07 / 0,26 / 0,47 | 0,42 / 0,49 / 0,56 | **−0,70** | −0,06 | −0,13 | −0,24 |
| streets | 1 842 | 63 / 121 / 192 | 66 / 119 / 185 | 0,53 / 0,72 / 0,91 | 0,06 / 0,24 / 0,47 | 0,41 / 0,49 / 0,55 | **−0,73** | −0,12 | −0,12 | −0,32 |
| aquarius | 1 456 | 47 / 82 / 127 | 48 / 82 / 121 | 0,58 / 0,82 / 1,00 | 0,00 / 0,22 / 0,43 | 0,42 / 0,51 / 0,60 | **−0,54** | −0,05 | −0,15 | −0,21 |

Trois faits, qui commandent le réglage :

1. **Couverture et abri sont fortement ANTICORRÉLÉS** (−0,54 à −0,73). Un lieu ouvert tue de
   partout ET meurt de partout ; un couloir ni l'un ni l'autre. Pris séparément, les deux axes
   mesurent surtout l'OUVERTURE du lieu — les planches `_exposition.png` et `_couverture.png`
   se ressemblent trait pour trait. **À poids égaux, cette composante commune s'annule** et il
   ne reste que ce que le lieu a d'ASYMÉTRIQUE : il voit beaucoup et n'est vu que de peu. C'est
   la propriété que les guides décrivent. L'étalement de l'asymétrie (0,14) est le tiers de
   celui de chaque axe (0,40) : la moitié de la variance était de l'ouverture.
2. **Les axes angulaires sont indépendants de l'avantage** (−0,05 à −0,15) et l'asymétrie lui
   est légèrement opposée (−0,21 à −0,32) : c'est une information NOUVELLE, pas une redite du
   rapport de duel. Un lieu d'où l'on gagne ses duels n'est pas, en général, un lieu
   asymétrique — cohérent avec le verdict v1 (l'avantage retrouve les bases).
3. **Le nombre de directions par disque est grand** (p10 : 47 à 71 ; le plancher de 40
   engagements le garantit) : un minimum de 20 directions par axe ne mord presque jamais.

Étalement (p90 − p10) de CHAQUE axe par disque, réglage v1, moyenne des trois cartes
(`_calibrage_v1/_rapport.md`, section « Axes du score ») — c'est ce qui décide des poids :

| Axe | recharge | streets | aquarius | moyenne |
|---|---|---|---|---|
| avantage | 0,19 | 0,17 | 0,15 | **0,17** |
| intensité | 0,21 | 0,20 | 0,20 | **0,20** |
| hauteur | 0,25 | 0,23 | 0,19 | **0,22** |
| portée | 0,95 | 1,00 | 0,81 | **0,92** |
| couverture | 0,38 | 0,39 | 0,42 | 0,40 |
| abri | 0,40 | 0,41 | 0,43 | 0,41 |
| (couv + abri)/2 | 0,14 | 0,14 | 0,18 | **0,14** |

**Découverte au passage** : en v1, la portée (poids 0,10, étalement 0,92 parce que sa
normalisation entre p50 et p90 de la carte SATURE à 0 et 1 sur la moitié des disques) pesait
dans le score autant que l'avantage (poids 0,50, étalement 0,17 à cause du rétrécissement).
Un poids nominal ne dit rien sans l'étalement de son axe.

---

## Réglage v2 figé — 2026-09-20 14:06

> Ce réglage est **FIGÉ**, choisi sur les distributions de TROIS cartes (recharge, aquarius,
> streets) SANS regarder l'oracle. Les neuf autres cartes servent de validation à l'étape
> 2bis.D. Il ne se retouche pas après le verdict v2 (même règle que la v1, D12).
> Implémentation : `powerpos.ReglageV2()` (`reglage_v2.go`).

### La formule

Mêmes conditions de mesure que la v1 (disque de 2,0 m, cellule centrale vue dans ≥ 3 matchs,
≥ 40 engagements dans le disque, prior de 40 engagements, dénivelé de référence 1,5 m).

```
avantage   = comme en v1, mais sur les comptes PONDÉRÉS par le rang du tueur
             (rétrécissement proportionné à l'échantillon BRUT)
intensite  = comme en v1
hauteur    = comme en v1
portee     = comme en v1
couverture = dispersion corrigée des directions de tir du disque      (0,5 si < 20 directions)
abri       = 1 − dispersion corrigée des directions d'où l'on y meurt  (0,5 si < 20 directions)

SCORE = 0,26 × couverture + 0,26 × abri + 0,21 × avantage + 0,16 × hauteur
      + 0,09 × intensite + 0,02 × portee
```

### Justification des poids — une règle, pas un goût

**Poids nominal = part voulue / étalement mesuré de l'axe**, puis normalisation à 1. Parts :
asymétrie (couverture + abri, à poids ÉGAUX pour annuler l'ouverture) **2**, avantage **1**,
hauteur **1**, intensité **0,5**, portée **0,5**.

| Axe | Part | Étalement | Part / étalement | Poids normalisé |
|---|---|---|---|---|
| couverture + abri | 2 (soit 1 par axe sur l'asymétrie de 0,14, donc 1 / 0,14 chacun) | 0,14 | 7,14 + 7,14 | **0,26 + 0,26** |
| avantage | 1 | 0,17 | 5,88 | **0,21** |
| hauteur | 1 | 0,22 | 4,55 | **0,16** |
| intensité | 0,5 | 0,20 | 2,50 | **0,09** |
| portée | 0,5 | 0,92 | 0,54 | **0,02** |

Pourquoi ces parts : les deux signaux qui disent COMMENT LE LIEU EST FAIT (asymétrie
angulaire, hauteur) reçoivent trois parts sur cinq ; ceux qui disent CE QUI S'Y PASSE
(avantage, intensité, portée) deux parts. L'avantage passe d'une part sur deux (v1) à une sur
cinq : le verdict v1 a montré qu'il retrouve les bases. L'intensité à une demi-part : le
plancher de 40 engagements en fait déjà une condition d'entrée. La portée à une demi-part :
un poste de sniper n'est pas forcément un lieu qu'on TIENT, et son axe sature (voir ci-dessus).

Contributions effectives (poids × étalement) : asymétrie 0,073, portée 0,018, avantage 0,036,
hauteur 0,035, intensité 0,018. **Le score v2 s'étale de 0,08 à 0,09 entre p10 et p90** (v1 :
0,15 à 0,18) — c'est une note de classement, et les seuils sont posés sur ses quantiles par
carte, pas sur sa valeur absolue.

### La règle de sélection

1. **Scorabilité** — inchangée (≥ 3 matchs distincts sur la cellule centrale, ≥ 40
   engagements dans le disque).
2. **Hystérésis** — une cellule est une AMORCE si son score ≥ p95 de la carte (et ≥ 0,57,
   plancher absolu) ; elle est retenue si son score ≥ p90 de la carte ET si elle est reliée en
   4-connexité à une amorce. Le seuil de Canny, celui de tous les détecteurs de contour : un
   lieu commence à un maximum franc et s'étend tant que le score reste dans le décile
   supérieur. Une cellule à 0,001 sous le p90 n'entre pas ; une cellule à 0,001 sous le p95
   entre si elle touche un maximum. C'est la distinction que le seuil unique v1 ne savait pas
   faire (Orange Pipes 0,678 contre 0,680).
3. **Fermeture morphologique de rayon 1 cellule** (dilatation puis érosion par un carré 3×3) :
   un trou d'une cellule n'est pas un mur. La fermeture ne retire jamais une cellule retenue ;
   les cellules qu'elle ajoute font partie du polygone, **jamais des moyennes ni des comptes**
   (`NbCellulesMesurees` au JSON, test `TestFermetureNEntrePasDansLesMoyennes`).
4. **Composantes en 8-connexité APRÈS fermeture**, taille minimale **10 cellules** (2,5 m²).
   La 8-connexité que la v1 refusait à raison sur un semis clairsemé est légitime sur un
   ensemble déjà refermé.
5. **8 positions au maximum par carte**, enveloppe convexe dilatée d'une demi-cellule
   (inchangé).

Justifications chiffrées (calibrage, `_calibrage_v2/_rapport.md`, section « Sélection ») :

| Carte | Seuil amorce (p95) | Seuil croissance (p90) | Cellules amorce | Retenues | Après fermeture | Composantes avant fermeture (tailles) | Après (tailles) | Positions |
|---|---|---|---|---|---|---|---|---|
| recharge | 0,585 | 0,573 | 96 | 152 | 203 | 16 : 56 36 27 14 3 3 3 2 | 5 : 85 69 34 12 3 | 4 |
| streets | 0,582 | 0,573 | 93 | 149 | 174 | 15 : 51 28 17 11 9 9 7 4 | 7 : 88 49 22 11 2 1 1 | 4 |
| aquarius | 0,600 | 0,591 | 73 | 117 | 138 | 31 : 31 14 8 8 6 5 5 4 | 19 : 36 20 14 11 9 8 8 6 | 4 |

(Rapport `_calibrage_v2/_rapport.md`, regénéré avec la rampe du titre ; avec l'assiette
restreinte du calibrage initial, recharge rendait `5 : 85 69 34 9 3` et 3 positions.)

- **Plancher absolu 0,57** : juste sous l'amorce la plus basse mesurée (0,582, streets) — un
  filet qui ne mord sur aucune carte de calibrage, même rôle que le 0,65 de la v1.
- **Fermeture + 8-connexité** : sur aquarius, 31 amas avant fermeture (le plus grand 31, puis
  14, 8, 8, 6…) deviennent 19 (36, 20, 14, 11, 9…) — les fragments de 5 à 8 se soudent, ce qui
  est précisément le défaut diagnostiqué au verdict v1 (amas de 2 à 10 pour 12 exigés).
- **Taille minimale 10** : après fermeture, sur recharge et streets, la plus petite composante
  retenue fait 11 à 12 cellules et la suivante 2 à 3 : la coupure à 10 tombe dans ce creux.
  Dix cellules = 2,5 m², un pas de tir. **Sensibilité connue** : la composante de recharge
  qui fait 12 cellules avec la rampe du titre en faisait 9 avec la rampe du calibrage — une
  position de 7,8 m² tient à trois cellules. La v1 avait la même fragilité (deux zones à
  0,01 du seuil) ; l'hystérésis la réduit sans la supprimer.
- **Croissance au p90 et non au p80** (essayé, écarté, voir ci-dessous).

### Ce qui a été essayé et écarté

| Essayé | Verdict | Pourquoi (chiffres) |
|---|---|---|
| Croissance au **p80** (amorce p95) | **Écarté** | des lieux de 154 cellules (67 m², recharge) et 241 cellules (150 m², streets) : une salle entière, pas une position. Au p90 : 85 et 88 cellules au plus (42 et 59 m²). |
| Plancher absolu **0,55** | **Corrigé avant gel** | posé avant la mesure ; le p75 du score v2 vaut 0,552-0,574, un plancher à 0,55 aurait été sous la croissance de toutes les cartes. Relevé à 0,57, sous l'amorce la plus basse (0,581). |
| Poids « à la main » (0,30 / 0,20 / 0,15 / 0,10 / 0,15 / 0,10) | **Remplacés avant gel** | posés avant de mesurer les étalements ; la portée y pesait autant que l'asymétrie. Remplacés par la règle part / étalement. |
| Couverture et abri à poids DIFFÉRENTS | **Écarté** | toute différence de poids réintroduit la composante « ouverture » (corr −0,70), qui n'est pas ce qu'on cherche. |
| Restreindre le corpus aux tueurs au-dessus d'un rang | **Écarté (mesuré à part)** | avec 19 à 47 % de kills à rang connu, le corpus tomberait sous les 40 engagements par disque sur la plupart des cellules. La rampe garde tout et penche ; la variante « tueurs forts » reste comptée au CSV. |
| Corriger les positions de « Live Fire - Ranked » à la lecture | **Écarté au profit de l'exclusion** | la formule inverse ne saurait pas distinguer une ligne déjà recuite d'une ligne périmée et la doublerait ; un chantier voisin recuit la base. Exclusion par `--exclure-variantes`, comptée (`kills_exclus`). |

## Live Fire — exclusion de la variante classée

Les positions de kill de « Live Fire - Ranked » sont fausses en base (découverte n°1 du plan :
x_faux ≈ x_juste / 2 + 23,26 m, lignes décodées avant le 2026-09-15). La passe v2 **exclut
cette variante** (`--exclure-variantes "Live Fire - Ranked"`, valeur par défaut datée dans
`main.go` avec son critère de retrait). Live Fire est donc mesurée sur sa seule variante de
base ; le contrôle de cohérence des variantes n'a plus rien à comparer sur cette carte.
Comme en v1, Live Fire ne compte pas comme preuve au verdict.

## Passe v2 sur les 12 cartes de la v1 — résultats (2026-09-20, après gel)

Couverture du rang, carte par carte (`_rapport.md`, section « Couverture du rang ») :

| Carte | Kills lus | Kills exclus (variante) | Kills à rang connu | part | Matchs de la carte avec ≥ 1 rang |
|---|---|---|---|---|---|
| live fire | 4 007 | **3 854** | 0 | 0 % | 704 / 915 (77 %) |
| recharge | 7 344 | 0 | 3 442 | 47 % | 686 / 888 (77 %) |
| streets | 6 950 | 0 | 2 666 | 38 % | 542 / 730 (74 %) |
| aquarius | 4 945 | 0 | 926 | 19 % | 406 / 579 (70 %) |
| illusion | 4 213 | 0 | 0 | **0 %** | 0 / 62 |
| bazaar | 4 002 | 0 | 0 | **0 %** | 63 / 214 |
| forbidden | 3 345 | 0 | 0 | **0 %** | 77 / 132 |
| origin | 3 151 | 0 | 1 142 | 36 % | 106 / 151 |
| lattice | 3 716 | 0 | 3 622 | **97 %** | 126 / 143 |
| solitude | 2 519 | 0 | 1 259 | 50 % | 278 / 342 |
| fortress | 1 213 | 0 | 0 | **0 %** | 27 / 49 |
| empyrean | 1 390 | 0 | 0 | **0 %** | 160 / 243 |
| **total** | 46 795 | 3 854 | 13 057 | **28 %** | |

Lecture : le rang n'existe que pour les matchs classés, et les matchs PORTEURS DE POSITIONS
DE KILL de cinq cartes sont tous sociaux (illusion, bazaar, forbidden, fortress, empyrean :
0 %), alors que les matchs de ces cartes ont pour partie un rang (bazaar 63 / 214) — les
films décodés et les matchs classés ne sont pas les mêmes matchs. Sur Live Fire, exclure la
variante classée exclut de fait tous les tueurs à rang connu. La pondération par rang est
donc un signal qui ne parle que sur la moitié des cartes ; sur les autres, le score v2 vaut
un score sans rang (poids neutre). Consigné au plan (découverte 10).

Signaux angulaires sur les 12 cartes : corr(couverture, abri) de **−0,34 (forbidden) à
−0,73 (streets)**, partout négative ; asymétrie p50 de 0,47 à 0,51, étalement 0,14 à 0,24 —
les cartes de calibrage ne sont pas des cas particuliers.

Positions retenues (`positions_v2.json`, `_rapport.md`) :

| Carte | Scorables | Amorce (p95) | Croissance (p90) | Composantes après fermeture (tailles) | Positions | Cellules (min / méd / max) | Aire m² (min / max) |
|---|---|---|---|---|---|---|---|
| live fire | 1 059 | 0,602 | 0,587 | 14 : 31 19 17 16 10 3 2 1 | **5** | 10 / 17 / 31 | 6,0 / 14,2 |
| recharge | 1 915 | 0,585 | 0,573 | 5 : 85 69 34 12 3 | **4** | 12 / 52 / 85 | 7,8 / 41,6 |
| streets | 1 842 | 0,582 | 0,573 | 7 : 88 49 22 11 2 1 1 | **4** | 11 / 36 / 88 | 5,5 / 58,6 |
| aquarius | 1 456 | 0,600 | 0,591 | 19 : 36 20 14 11 9 8 8 6 | **4** | 11 / 17 / 36 | 7,4 / 14,2 |
| illusion (témoin) | 1 181 | 0,628 | 0,611 | 9 : 40 30 11 6 4 3 2 1 | **3** | 11 / 30 / 40 | 6,8 / 21,2 |
| bazaar (témoin) | 1 052 | 0,611 | 0,601 | 14 : 30 26 19 9 8 2 1 1 | **3** | 19 / 26 / 30 | 11,4 / 27,0 |
| forbidden | 741 | 0,605 | 0,588 | 8 : 53 11 6 4 3 1 1 1 | **2** | 11 / 32 / 53 | 7,0 / 30,0 |
| origin | 759 | 0,615 | 0,597 | 7 : 33 13 12 11 5 1 1 | **4** | 11 / 12 / 33 | 5,4 / 20,4 |
| lattice | 933 | 0,609 | 0,595 | 13 : 26 23 12 8 6 5 4 1 | **3** | 12 / 23 / 26 | 7,2 / 13,1 |
| solitude | 614 | 0,629 | 0,595 | 6 : 22 15 10 4 1 1 | **3** | 10 / 15 / 22 | 5,0 / 10,5 |
| fortress | 102 | 0,645 | 0,606 | 3 : 4 2 2 | **0** | — | — |
| empyrean | 128 | 0,630 | 0,610 | 6 : 2 2 1 1 1 1 | **0** | — | — |

**35 positions** (v1 : 31), de **5,0 à 58,6 m²** (v1 : 6 à 31). Fortress et empyrean rendent
toujours zéro (3 % de cellules scorables : la scorabilité, inchangée, décide — D8). Les deux
plus grandes positions (88 cellules sur streets, 85 sur recharge) sont des couloirs entiers
retenus d'un bloc par la croissance au p90 ; la v1 n'en produisait pas d'aussi grandes, et
c'est au verdict 2bis.D de dire si un couloir de 59 m² est une position ou une salle.
**Aucune de ces positions n'a été confrontée à l'oracle** : la v2 sera jugée sur les neuf
cartes hors calibrage (D11), avec le témoin géographique.

## Fichiers produits

- `mesures_v2_2026-09-20/{carte}__arene.csv` — une ligne par cellule peuplée, 25 colonnes :
  les 13 de la v1 + `dir_sortante_{cos,sin,n}`, `dir_entrante_{cos,sin,n}`,
  `kills_ponderes`, `morts_ponderees`, `kills_rang_connu`, `morts_rang_connu`,
  `kills_tueur_fort`, `morts_tueur_fort` (sommes brutes, aucune dérivée).
- `mesures_v2_2026-09-20/{carte}__arene_{duel,occupation,score,exposition,couverture}.png`
  — cinq planches par carte dont le fond est publié (exposition et couverture lues sur le
  disque, cellules à ≥ 10 directions ; score étalé entre p10 et p99 de la carte).
- `mesures_v2_2026-09-20/positions_v2.json` — même forme que `positions.json` v1
  (`schema_version` 1) + `nb_cellules_mesurees` par position, réglage v2 sérialisé.
- `mesures_v2_2026-09-20/_rapport.md` — toutes les tables, dont les nouvelles sections
  (couverture du rang, signaux angulaires, axes, sélection).
- `mesures_v2_2026-09-20/_calibrage_v1/_rapport.md` et `_calibrage_v2/_rapport.md` — les
  rapports de calibrage sur les trois cartes (les CSV et PNG de calibrage, identiques à ceux
  de la passe complète, ne sont pas conservés).
