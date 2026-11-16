# Mesure empirique des positions de force — 2026-09-20

> Étape 1 du plan `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md`. Branche `wt/power-positions`.
> Outil : `apps/go-api/cmd/mappower-build`, paquet pur `apps/go-api/internal/analysis/powerpos`.
> Données lues en SEULE lecture sur `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
> (`duckdb.OpenReadForQuery`, vues `_latest` uniquement, aucune écriture).
>
> Commandes exactes de la passe :
>
> ```
> cd apps/go-api
> go run ./cmd/mappower-build --recensement \
>   --data-root <racine contenant data/> \
>   --sortie .ai/V7.5/positions_de_force/recensement_2026-09-20.md
> go run ./cmd/mappower-build --mesure \
>   --data-root <racine contenant data/> \
>   --sortie .ai/V7.5/positions_de_force/mesures_2026-09-20 \
>   --cartes "live fire,recharge,streets,aquarius,forbidden,origin,lattice,solitude,fortress,empyrean,illusion,bazaar"
> ```

## Corpus

Recensement complet : `recensement_2026-09-20.md` (89 lignes, toutes cartes du registre qui
portent au moins une position de kill ou un artefact de rejeu). Registre : **9 110 matchs**
retenus, 34 écartés faute de nom de carte, 0 artefact de rejeu orphelin.

**La clé d'agrégation rabote les suffixes de variante de playlist** (`decfilm.NormalizeMapName`,
qui retire « - Ranked » et « Heavies ») : la variante classée d'une carte est le même niveau
moteur, donc le même repère monde. Sans ce rabotage, le corpus de Live Fire serait 50 + 30
matchs au lieu de 80, celui de Recharge 47 + 31 au lieu de 78. L'**axe** sépare l'arène du BTB
(douze contre douze, par `halo_infinite.InferModeCategoryFromPairName`) ; le PvE est exclu de
la mesure (ses kills visent des vagues d'IA). La colonne `mode_category` du registre n'est PAS
utilisable : mesurée le 2026-09-20, elle vaut « Other » sur 126 matchs de
« Ranked:Strongholds on Live Fire ».

Les quarante cartes les mieux fournies, triées par matchs porteurs de positions de kills :

| # | Carte | Axe | Matchs avec positions | Kills | Matchs joués | Artefacts de rejeu | Catégories |
|---|---|---|---|---|---|---|---|
| 1 | live fire | arene | 80 | 7861 | 915 | 6 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 2 | recharge | arene | 78 | 7344 | 888 | 6 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 3 | streets | arene | 75 | 6950 | 730 | 5 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 4 | aquarius | arene | 57 | 4945 | 579 | 5 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 5 | illusion | arene | 55 | 4213 | 62 | 3 | Assassin, Other, Super Fiesta |
| 6 | bazaar | arene | 52 | 4002 | 214 | 3 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 7 | cliffhanger | arene | 48 | 3692 | 121 | 1 | Assassin, Fiesta, Other, Super Fiesta |
| 8 | prism | arene | 48 | 3620 | 57 | 2 | Assassin, Other, Super Fiesta |
| 9 | behemoth | arene | 46 | 3129 | 65 | 3 | Assassin, Fiesta, Other, Super Fiesta |
| 10 | catalyst | arene | 45 | 3505 | 190 | 3 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 11 | forest | arene | 45 | 3591 | 111 | 3 | Assassin, Fiesta, Other, Ranked, Super Fiesta |
| 12 | forbidden | arene | 44 | 3345 | 132 | 0 | Assassin, Other, Ranked, Super Fiesta |
| 13 | chasm | arene | 43 | 3151 | 117 | 2 | Assassin, Other, Ranked, Super Fiesta |
| 14 | origin | arene | 37 | 3151 | 151 | 3 | Assassin, Other, Ranked |
| 15 | lattice | arene | 28 | 3716 | 143 | 3 | Other, Ranked |
| 16 | solitude | arene | 26 | 2519 | 342 | 1 | Assassin, Fiesta, Other, Ranked |
| 17 | snowbound | arene | 25 | 1418 | 25 | 2 | Assassin |
| 18 | starboard | arene | 24 | 1844 | 29 | 1 | Assassin, Other, Ranked |
| 19 | launch site | arene | 23 | 1508 | 23 | 3 | Super Fiesta |
| 20 | absolution | arene | 22 | 1612 | 22 | 1 | Assassin |
| 21 | the pit | arene | 21 | 1562 | 21 | 1 | Assassin |
| 22 | dynasty | arene | 20 | 1482 | 22 | 1 | Assassin, Other, Super Fiesta |
| 23 | curfew | arene | 19 | 1572 | 22 | 1 | Assassin, Other |
| 24 | domicile | arene | 19 | 1205 | 24 | 2 | Assassin, Other |
| 25 | cliffside | arene | 18 | 1373 | 20 | 0 | Assassin, Other |
| 26 | fortress | arene | 18 | 1213 | 49 | 2 | Assassin, Ranked |
| 27 | nemesis | arene | 18 | 1155 | 18 | 1 | Assassin |
| 28 | empyrean | arene | 17 | 1390 | 243 | 1 | Assassin, Other, Ranked, Super Fiesta |
| 29 | goliath | arene | 17 | 1285 | 17 | 1 | Assassin |
| 30 | shiro | arene | 17 | 1426 | 23 | 0 | Assassin, Other, Super Fiesta |
| 31 | dredge | arene | 16 | 1179 | 60 | 1 | Assassin, Other, Ranked |
| 32 | high ground | arene | 16 | 1078 | 16 | 1 | Assassin |
| 33 | isolation | arene | 16 | 998 | 22 | 3 | Assassin, Other |
| 34 | houseki | arene | 15 | 1261 | 17 | 0 | Assassin, Super Fiesta |
| 35 | takamanohara | arene | 15 | 1149 | 15 | 2 | Assassin |
| 36 | vagabond | arene | 15 | 1217 | 23 | 0 | Assassin |
| 37 | banished narrows | arene | 14 | 1057 | 33 | 4 | Assassin, Other, Ranked |
| 38 | fragmentation | btb | 14 | 1598 | 134 | 0 | BTB |
| 39 | elevation | arene | 12 | 843 | 22 | 0 | Assassin, Other |
| 40 | command | btb | 11 | 1482 | 18 | 0 | BTB |

Ce que ce tableau dit, et qui commande la suite :

1. **Le corpus des positions de kill est FIN.** La carte la mieux servie compte 80 matchs et
   7 861 kills. Les quatre premières (Live Fire, Recharge, Streets, Aquarius) sont seules
   au-dessus de 50 matchs.
2. **Dix cartes du circuit HCS sont présentes** — live fire (80), recharge (78), streets (75),
   aquarius (57), forbidden (44), origin (37), lattice (28), solitude (26), fortress (18),
   empyrean (17). **Argyle est ABSENTE du corpus** (aucun match).
3. **Le corpus d'artefacts de rejeu est DÉRISOIRE** : 92 artefacts pour tout le titre, soit
   0 à 6 par carte. C'est ce chiffre, et non la qualité du signal, qui décide du sort de
   l'occupation par équipe (cf. « Score figé »).
4. Les cartes BTB plafonnent à 14 matchs (Fragmentation). Aucune n'atteint le niveau de
   corpus des cartes d'arène ; l'axe BTB n'est pas mesuré ici.

## Cartes mesurées

Douze cartes : les dix cartes HCS du corpus, plus deux témoins hors HCS (**illusion**, 55
matchs et **bazaar**, 52 matchs — les deux cartes les mieux fournies qui ne sont pas au
circuit). Tableaux complets dans `mesures_2026-09-20/_rapport.md` ; un CSV par carte (une
ligne par cellule peuplée, treize colonnes brutes, aucune dérivée) et trois PNG de contrôle
(duel, occupation, score) par carte dont le fond est publié.

| Carte | Matchs | Kills | Cellules peuplées | Artefacts | Cellules scorables | Positions retenues |
|---|---|---|---|---|---|---|
| live fire | 80 | 7 861 | 4 516 | 6 | 2 103 (47 %) | 3 |
| recharge | 78 | 7 344 | 2 992 | 6 | 1 915 (64 %) | 4 |
| streets | 75 | 6 950 | 2 947 | 5 | 1 842 (63 %) | 4 |
| aquarius | 57 | 4 945 | 3 184 | 5 | 1 456 (46 %) | 5 |
| illusion (témoin) | 55 | 4 213 | 3 496 | 3 | 1 181 (34 %) | 4 |
| bazaar (témoin) | 52 | 4 002 | 3 897 | 3 | 1 052 (27 %) | 2 |
| forbidden | 44 | 3 345 | 2 814 | 0 | 741 (26 %) | 1 |
| origin | 37 | 3 151 | 3 797 | 3 | 759 (20 %) | 3 |
| lattice | 28 | 3 716 | 3 645 | 2 | 933 (26 %) | 4 |
| solitude | 26 | 2 519 | 2 916 | 1 | 614 (21 %) | 1 |
| fortress | 18 | 1 213 | 3 540 | 2 | 102 (3 %) | **0** |
| empyrean | 17 | 1 390 | 3 834 | 1 | 128 (3 %) | **0** |

`lattice` n'a pas de fond de carte publié : ses CSV existent, ses PNG non.

### Plancher de matchs distincts — le rayon du nuage

Mesure reprise de `cmd/mappos-build` (2026-08-30), qui l'avait faite sur les positions de
PASSAGE : sans filtre le nuage s'étendait à 268 m du centre, à deux matchs il retombait à
27 m, à trois à 19,4 m. **Sur les positions de KILL, le résultat est tout autre** : p99 de la
distance au barycentre, par plancher (entre parenthèses, les cellules retenues).

| Carte | ≥ 1 match | ≥ 2 | ≥ 3 | ≥ 5 |
|---|---|---|---|---|
| live fire | 24,3 (4 027) | 23,7 (2 980) | 23,4 (2 110) | 23,4 (983) |
| recharge | 19,1 (2 694) | 18,7 (2 289) | 18,2 (1 916) | 16,4 (1 142) |
| streets | 19,7 (2 641) | 19,4 (2 250) | 19,0 (1 848) | 16,6 (1 153) |
| aquarius | 21,6 (2 649) | 20,4 (2 041) | 19,3 (1 460) | 18,4 (654) |
| illusion | 22,7 (2 751) | 22,1 (1 831) | 21,9 (1 198) | 21,7 (431) |
| bazaar | 24,6 (2 795) | 22,2 (1 759) | 21,0 (1 064) | 18,0 (376) |
| forbidden | 28,4 (2 814) | 27,0 (1 421) | 25,6 (777) | 21,9 (281) |
| origin | 20,7 (2 587) | 19,6 (1 435) | 19,7 (769) | 19,5 (169) |
| lattice | 20,6 (2 805) | 20,1 (1 689) | 19,6 (951) | 18,0 (301) |
| solitude | 19,1 (2 134) | 18,0 (1 227) | 17,4 (626) | 16,9 (128) |
| fortress | 19,7 (1 493) | 18,9 (483) | 19,4 (145) | 15,2 (9) |
| empyrean | 24,1 (1 705) | 23,1 (553) | 22,3 (170) | 20,1 (14) |

**Le nuage des kills est DÉJÀ compact** : de 1 à 3 matchs, le rayon ne bouge que de 4 à 10 %
(live fire : 24,3 → 23,4 m). Le plancher ne sert donc PAS ici à rogner des bras hors de
l'arène, comme il le faisait pour les positions de passage — une position de kill est
forcément sur du terrain joué. Il sert à autre chose : garantir qu'une cellule a été vue dans
plusieurs parties, donc qu'elle n'est pas l'anecdote d'un match. Son coût est en revanche
réel : à 3 matchs il ne reste que 145 cellules sur fortress et 170 sur empyrean.

### La taille d'échantillon par cellule, et ce qu'elle interdit

| Carte | engagements p50 | p75 | p90 | p99 | max | cellules ≥ 6 | ≥ 12 | ≥ 25 |
|---|---|---|---|---|---|---|---|---|
| live fire | 3 | 5 | 8 | 16 | 22 | 898 | 141 | 0 |
| recharge | 4 | 7 | 10 | 22 | 41 | 1 013 | 212 | 25 |
| streets | 4 | 7 | 10 | 17 | 28 | 1 008 | 193 | 2 |
| aquarius | 2 | 5 | 7 | 13 | 19 | 536 | 46 | 0 |
| illusion | 2 | 3 | 6 | 11 | 22 | 367 | 35 | 0 |
| bazaar | 1 | 3 | 5 | 11 | 24 | 310 | 27 | 0 |
| forbidden | 2 | 3 | 5 | 9 | 14 | 220 | 9 | 0 |
| origin | 1 | 2 | 4 | 9 | 26 | 165 | 15 | 1 |
| lattice | 1 | 3 | 5 | 10 | 24 | 258 | 22 | 0 |
| solitude | 1 | 3 | 4 | 8 | 14 | 114 | 2 | 0 |
| fortress | 0 | 1 | 2 | 4 | 11 | 14 | 0 | 0 |
| empyrean | 0 | 1 | 2 | 5 | 7 | 19 | 0 | 0 |

### LA MESURE QUI A TOUT DÉCIDÉ : le rapport de duel par cellule est du bruit

Dispersion de `kills_depuis / engagements`, sur les cellules à ≥ 6 engagements :

| Carte | n | p10 | p25 | p50 | p75 | p90 |
|---|---|---|---|---|---|---|
| live fire | 898 | 0,29 | 0,38 | 0,50 | 0,62 | 0,73 |
| recharge | 1 013 | 0,30 | 0,38 | 0,50 | 0,60 | 0,71 |
| streets | 1 008 | 0,29 | 0,38 | 0,50 | 0,62 | 0,71 |
| aquarius | 536 | 0,29 | 0,38 | 0,50 | 0,62 | 0,71 |
| illusion | 367 | 0,29 | 0,38 | 0,50 | 0,62 | 0,67 |
| bazaar | 310 | 0,29 | 0,38 | 0,50 | 0,62 | 0,71 |
| forbidden | 220 | 0,29 | 0,33 | 0,50 | 0,62 | 0,67 |
| origin | 165 | 0,33 | 0,43 | 0,54 | 0,67 | 0,75 |
| lattice | 258 | 0,25 | 0,40 | 0,50 | 0,67 | 0,75 |
| solitude | 114 | 0,29 | 0,33 | 0,50 | 0,67 | 0,79 |
| fortress | 14 | 0,33 | 0,37 | 0,48 | 0,67 | 0,69 |
| empyrean | 19 | 0,32 | 0,38 | 0,50 | 0,62 | 0,71 |

**Douze cartes, douze fois la même courbe, au centième.** Une mesure de terrain ne se
comporte pas ainsi : Aquarius et Lattice ne sont pas la même carte. Ce que cette table montre
est la dispersion d'une PIÈCE ÉQUILIBRÉE tirée six à dix fois — le nombre médian
d'engagements par cellule vaut 1 à 4. Poser un seuil sur cette grandeur sélectionnerait des
cellules chanceuses, et en sélectionnerait tout autant sur une carte tirée au hasard.

**Conséquence, et c'est la décision structurante de l'étape : le score ne se calcule pas sur
la cellule, mais sur son VOISINAGE.** La cellule reste l'unité d'adressage et de publication
(décision D1 du plan) ; elle cesse d'être l'unité de comptage.

### Portée et dénivelé

| Carte | portée médiane p50 (m) | p90 | dénivelé p10 (m) | p50 | p90 |
|---|---|---|---|---|---|
| live fire | 4,4 | 8,0 | −0,4 | 0,0 | 0,8 |
| recharge | 4,5 | 9,1 | −0,5 | 0,0 | 0,8 |
| streets | 4,7 | 9,2 | −0,6 | 0,0 | 0,6 |
| aquarius | 4,1 | 8,2 | −0,5 | 0,0 | 0,8 |
| illusion | 4,0 | 11,7 | −0,5 | 0,0 | 1,1 |
| bazaar | 4,0 | 11,6 | −0,3 | 0,0 | 0,6 |
| forbidden | 4,9 | 11,6 | −0,4 | 0,0 | 0,8 |
| origin | 6,1 | 10,8 | −0,6 | 0,0 | 1,1 |
| lattice | 7,2 | 12,0 | −0,5 | 0,0 | 1,6 |
| solitude | 5,3 | 10,5 | −0,5 | −0,0 | 1,3 |
| fortress | 3,6 | 5,9 | −0,3 | 0,0 | 0,2 |
| empyrean | 3,8 | 8,3 | −0,3 | 0,0 | 0,2 |

Ces deux-là, contrairement au rapport de duel, DIFFÈRENT d'une carte à l'autre : la portée
médiane va de 3,6 m (fortress) à 7,2 m (lattice), le p90 du dénivelé de 0,2 m (fortress,
empyrean) à 1,6 m (lattice). Ils portent donc de l'information — à condition d'être
normalisés PAR CARTE, sans quoi Lattice écraserait Fortress sur les deux axes.

### Occupation par équipe

| Carte | Artefacts | Cellules avec présence | Cellules à ≥ 3 matchs | écart p10 | p50 | p90 |
|---|---|---|---|---|---|---|
| live fire | 6 | 3 466 | 3 118 | −0,38 | +0,03 | +0,45 |
| recharge | 6 | 2 954 | 2 673 | −0,36 | +0,06 | +0,46 |
| streets | 5 | 2 918 | 2 593 | −0,30 | +0,20 | +0,62 |
| aquarius | 5 | 3 163 | 2 903 | −0,42 | +0,11 | +0,62 |
| illusion | 3 | 3 337 | 1 301 | −0,62 | +0,11 | +0,71 |
| bazaar | 3 | 3 802 | 2 252 | −0,38 | +0,20 | +0,71 |
| forbidden | 0 | 0 | 0 | — | — | — |
| origin | 3 | 3 738 | 2 246 | −0,64 | +0,05 | +0,64 |
| lattice | 2 | 3 374 | **0** | — | — | — |
| solitude | 1 | 2 659 | **0** | — | — | — |
| fortress | 2 | 3 486 | **0** | — | — | — |
| empyrean | 1 | 3 665 | **0** | — | — | — |

Le signal est spatialement très cohérent à l'œil (planches `*_occupation.png` : de larges
plages rouges et bleues, aucun poivre-et-sel) — mais c'est précisément ce qui doit rendre
méfiant. Avec un à six matchs, ce que ces plages dessinent est le CÔTÉ DE DÉPART des équipes
gagnantes de ces matchs-là, pas une propriété de la carte. Et la médiane de l'écart est
POSITIVE partout (+0,03 à +0,20) : elle mesure d'abord que les vainqueurs vivent plus
longtemps, donc occupent plus, partout.

---

## Score figé — 2026-09-20

> Ce réglage est **FIGÉ**. Il ne sera plus retouché après le verdict contre l'oracle
> (étape 2) : une retouche de seuil a posteriori invalide le verdict et se traite comme une
> nouvelle étape 2. Implémentation : `powerpos.ReglageV1()`, `powerpos.Score`,
> `powerpos.Selectionne`.

### La formule

Pour chaque cellule de 0,5 m qui passe le plancher, on somme les accumulateurs de toutes les
cellules d'un **disque de 2,0 m** autour d'elle (49 cellules). Soient `K` les kills partis du
disque, `M` les morts subies dedans, `N = K + M`.

```
avantage  = borne01( 0,5 + 2 × ( (K + 0,5 × 40) / (N + 40) − 0,5 ) )
intensite = borne01( ln(1 + K) / ln(1 + K_p95_carte) )
hauteur   = borne01( 0,5 + denivele_pondere / (2 × 1,5) )
portee    = borne01( 0,5 + (portee_ponderee − portee_p50_carte) / (2 × (portee_p90_carte − portee_p50_carte)) )

SCORE = 0,50 × avantage + 0,25 × intensite + 0,15 × hauteur + 0,10 × portee
```

`denivele_pondere` et `portee_ponderee` sont les moyennes, pondérées par les kills, des
médianes par cellule du disque. `K_p95`, `portee_p50` et `portee_p90` sont mesurés **par
carte** sur ses disques scorables.

### Justification, signal par signal

| Signal | Poids | Pourquoi ce poids |
|---|---|---|
| **avantage** | 0,50 | C'est la définition même : un lieu d'où l'on gagne ses duels. Rétréci vers 0,5 avec une force de 40 engagements virtuels, ce qui ramène un disque à 40 engagements à mi-chemin de la neutralité et laisse un disque à 400 s'exprimer. Sans rétrécissement, la table de dispersion ci-dessus montre exactement ce qui se passerait : les petits échantillons trustent les extrêmes. |
| **intensite** | 0,25 | Sans elle, un recoin où trois duels ont bien tourné bat l'angle d'où partent cent kills. Une position de force est un lieu où il se passe quelque chose. En logarithme parce que les volumes par disque s'étalent sur deux ordres de grandeur, et rapportée au p95 de la carte parce que Live Fire (7 861 kills) et Fortress (1 213) n'ont pas les mêmes volumes. |
| **hauteur** | 0,15 | Seule mesure empirique de « tenir la hauteur ». Poids modeste parce que l'échelle est étroite : le p90 du dénivelé médian par cellule vaut 0,2 m (fortress) à 1,6 m (lattice). Normalisée sur 1,5 m, ce qui place le p90 de la carte la plus verticale près du maximum de l'axe. |
| **portee** | 0,10 | « Voir loin » discrimine (3,6 à 7,2 m de médiane selon la carte), mais un poste de sniper n'est pas forcément une position qu'on TIENT — d'où le poids le plus faible. Normalisée entre le p50 et le p90 **de la carte**. |
| **occupation** | **0,00** | Écartée. Voir ci-dessous. |

### La règle de sélection

1. **Scorabilité** — la cellule centrale doit avoir été vue dans **≥ 3 matchs distincts**
   (plancher repris de `tactical.PlancherMatchsParCellule`), et son disque porter
   **≥ 40 engagements**. Sous ces deux conditions la cellule n'a pas de score du tout.
2. **Seuil double** — score ≥ **p90 des cellules scorables de la carte**, ET score ≥ **0,65**
   (plancher absolu). Le quantile seul retiendrait toujours 10 % des cellules, y compris là
   où rien ne ressort ; le plancher seul serait un étalonnage global sur des cartes qui n'ont
   ni les mêmes volumes ni les mêmes distances.
3. **Composantes connexes en 4-connexité**, taille minimale **12 cellules** (3 m² de
   cellules). En 8-connexité, deux positions distinctes qui se frôlent par un coin
   fusionneraient en une enveloppe recouvrant le vide entre elles.
4. **8 positions au maximum par carte**, les mieux notées. Une carte d'arène en compte trois
   à six dans les guides ; en publier trente serait colorier la carte, pas la lire.
5. **Enveloppe** : enveloppe convexe des coins des cellules, dilatés d'une demi-cellule.
   Écart assumé de la v1 : une composante concave est recouverte par son plus petit convexe
   (test `TestEnveloppeRecouvreLeConcave`). Le remplacement futur ne change pas le type
   publié, seulement la fonction qui le remplit.

Sur le plancher absolu à 0,65 : **il ne mord sur aucune des douze cartes mesurées**, dont le
p90 du score va de 0,662 (fortress) à 0,726 (lattice). C'est son rôle — un filet qui ne se
déclenche que sur une carte dont le décile supérieur serait plus plat que tout ce qui a été
mesuré. Ce qui écarte réellement une carte pauvre, c'est la scorabilité : fortress et
empyrean n'ont que 3 % de cellules scorables, leurs composantes n'atteignent pas 12 cellules,
et **elles rendent zéro position**. C'est le comportement voulu par la décision D8 du plan :
une carte sous le plancher est ABSENTE, pas vide.

### Ce qui a été essayé et écarté

| Essayé | Verdict | Pourquoi |
|---|---|---|
| Rapport de duel **par cellule** (sans voisinage) | **Écarté** | Bruit binomial pur : douze cartes rendent la même dispersion au centième (0,29 / 0,38 / 0,50 / 0,62 / 0,71). Mesure ci-dessus. |
| Rapport de duel **sans rétrécissement** | **Écarté** | Un disque à 12 engagements pèserait autant qu'un disque à 300 ; c'est la même faute une échelle plus haut. |
| **Occupation par équipe** dans le score | **Écartée (poids 0)** | (a) six cartes sur douze n'ont AUCUNE cellule au plancher de 3 matchs de présence — le signal manque là où il faudrait trancher ; (b) sa médiane est positive partout (+0,03 à +0,20) : elle mesure que les vainqueurs vivent plus longtemps, un biais global et non un lieu ; (c) à 1–6 matchs par carte, les plages lues sont le côté de départ des équipes gagnantes de ces matchs-là. Les colonnes restent au CSV et la planche de contrôle reste produite : le signal se réévaluera quand le parc d'artefacts se comptera en centaines. |
| **8-connexité** pour les composantes (comme `tactical.GrappesDeSpawn`) | **Écartée** | Deux positions qui se touchent par un coin fusionneraient. Un pont qui ne tient que par un coin est un artefact de discrétisation, pas un lieu. Coût assumé : une position en diagonale se scinde — la taille minimale s'en charge. |
| **Plancher de rareté à 5 matchs** | **Écarté** | Ne gagne presque rien sur le rayon du nuage (16–23 m contre 17–26 m à 3) et coûte la moitié des cellules : 9 cellules restantes sur fortress, 14 sur empyrean. |
| **Plancher absolu du score à 0,25** | **Corrigé avant gel** | Première valeur posée avant la mesure ; le score médian observé vaut 0,59, un plancher à 0,25 n'aurait jamais rien filtré. Relevé à 0,65, juste sous le p90 le plus bas mesuré. |
| Compter une **mort sans tueur** en « morts ici » | **Écarté** | 6,5 % des lignes de `kill_positions_latest` n'ont pas de position de tueur (chute, suicide). Les compter peindrait les fosses en positions faibles alors que personne ne les tient. L'échantillon est écarté en entier et compté. |
| Compter les **points de piste** plutôt que leur durée | **Écarté** | Les points d'une piste ne sont pas équidistants (t = 0, 13, 26, 49, 51, 60…) ; compter les points ferait peser une zone où le décodeur a échantillonné serré autant qu'une zone tenue. |

### Ce que le score figé rend, carte par carte

| Carte | Scorables | score p50 | p90 | max | seuil appliqué | Positions | Cellules (min/méd/max) | Aire m² (min/max) |
|---|---|---|---|---|---|---|---|---|
| live fire | 2 103 | 0,594 | 0,667 | 0,746 | 0,667 | 3 | 36 / 42 / 65 | 20,0 / 31,2 |
| recharge | 1 915 | 0,593 | 0,680 | 0,793 | 0,680 | 4 | 14 / 46 / 52 | 6,0 / 22,5 |
| streets | 1 842 | 0,589 | 0,691 | 0,786 | 0,691 | 4 | 19 / 39 / 47 | 9,8 / 25,5 |
| aquarius | 1 456 | 0,611 | 0,687 | 0,756 | 0,687 | 5 | 18 / 21 / 29 | 8,1 / 13,8 |
| illusion | 1 181 | 0,592 | 0,707 | 0,848 | 0,707 | 4 | 12 / 18 / 30 | 6,9 / 15,1 |
| bazaar | 1 052 | 0,592 | 0,679 | 0,777 | 0,679 | 2 | 30 / 33 / 36 | 17,4 / 20,4 |
| forbidden | 741 | 0,608 | 0,678 | 0,747 | 0,678 | 1 | 17 / 17 / 17 | 7,9 / 7,9 |
| origin | 759 | 0,613 | 0,689 | 0,844 | 0,689 | 3 | 15 / 16 / 23 | 7,9 / 14,8 |
| lattice | 933 | 0,592 | 0,726 | 0,794 | 0,726 | 4 | 17 / 19 / 23 | 8,2 / 11,0 |
| solitude | 614 | 0,596 | 0,714 | 0,809 | 0,714 | 1 | 19 / 19 / 19 | 9,9 / 9,9 |
| fortress | 102 | 0,589 | 0,662 | 0,757 | 0,662 | **0** | — | — |
| empyrean | 128 | 0,595 | 0,663 | 0,753 | 0,663 | **0** | — | — |

Les positions font **6 à 31 m²**, soit un pas de tir et sa marge — l'ordre de grandeur d'un
lieu qu'on tient, et non d'une salle entière. Aucune n'est sous 1 m² ni au-dessus de 30 % de
la carte (invariant de relecture de l'étape 3).

**Aucune de ces positions n'a été confrontée à un guide pro.** C'est voulu : l'oracle est
écrit séparément (étape 0), et le verdict se rend à l'étape 2 contre des seuils écrits
d'avance. Le score ci-dessus a été choisi sur les distributions, sans jamais regarder la
réponse attendue.

## Réserve sur Live Fire — contamination mesurée

Le contrôle de cohérence des variantes (`mappower-build`, `controle_variantes.go`) déclare
**live fire INCOHÉRENTE** : les barycentres des nuages de kills de « Live Fire » et de
« Live Fire - Ranked » sont distants de **9,88 m**, alors que les dix autres cartes à deux
variantes tiennent sous 2,6 m et que huit d'entre elles sont sous 1,2 m.

Détail mesuré : x médian 8,2 m pour la carte de base contre 21,3 m pour la variante classée,
avec un y médian identique (32,7 m des deux côtés) et un x maximal de 27,3 m contre 36,9 m.
**13 des 30 matchs « - Ranked » débordent** au-delà de l'emprise de la carte de base. Les
ARTEFACTS DE REJEU des deux variantes, eux, s'accordent (x maximal 27,3 et 27,4 m), et les
deux variantes partagent un seul fond publié (`sgh_interlock.json`, qui déclare les deux
noms). C'est donc un défaut de décodage des POSITIONS DE KILL, pas une différence de
géométrie.

Ce chantier ne le corrige pas (hors périmètre — consigné dans la section « Découvertes » du
plan). Conséquence pour l'étape 2 : **les chiffres de Live Fire ne comptent pas comme preuve**,
dans un sens comme dans l'autre. Les neuf autres cartes HCS mesurées suffisent au critère de
quatre cartes du plan.

## Fichiers produits

- `recensement_2026-09-20.md` — recensement complet du corpus (89 lignes).
- `mesures_2026-09-20/{carte}__arene.csv` — 12 fichiers, une ligne par cellule peuplée,
  colonnes brutes : `col, lig, centre_x, centre_y, kills_depuis, morts_dedans,
  portee_mediane_m, denivele_median_m, matchs_kills, matchs_presence, matchs_distincts,
  occupation_gagnants_ms, occupation_perdants_ms`.
- `mesures_2026-09-20/{carte}__arene_{duel,occupation,score}.png` — 33 planches de contrôle
  (11 cartes ; lattice n'a pas de fond publié).
- `mesures_2026-09-20/_rapport.md` — toutes les tables ci-dessus, regénérées à chaque passe.
