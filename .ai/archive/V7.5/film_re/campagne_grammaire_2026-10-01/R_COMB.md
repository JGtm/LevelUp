# R-COMB : les leviers de la phase 2 mesurés ensemble (2026-10-02)

> Recherche R-COMB du PLAN §6.0 (« Mesures ouvertes ») et §6.1 ; critique n° 2, point N10.
> Worktree temporaire `LevelUp-wt-cg-comb`, HEAD détachée sur `fe18bf67c`. Rien n'est commité.
> Aucun fichier suivi n'est modifié : trois sondes neuves `r_comb_*_research_test.go`, une copie de
> la surcouche, des TSV. `grammar.Rev` reste `grammar-2026-09-27.3`. Aucune base, aucune cuisson,
> aucun backfill. Films lus en lecture seule, un à la fois, sentinelle `filmproc` à 4 Gio.
>
> Conventions (celles de `MESURES_BIS_1.md`) :
> - **mesuré** : compté par une sonde sur les films ;
> - **estimé** : dérivé d'une mesure par une hypothèse écrite ;
> - **sain** : paquet fermé qui ne contredit aucun des trois invariants de l'écrivain instrumentés
>   (ordre de la vue B, masque écrivable, vue C possible ; `cmContredit`, le juge de
>   `campagne_bis1_juge_research_test.go`). C'est la définition D2 : les fermetures factices sont
>   exclues. « Sain » ne veut pas dire « juste ».
> - **brut** : tous les paquets fermés, factices compris.

## Corrections du 2026-10-02 (ajoutées après coup ; le texte d'origine ci-dessous n'est pas réécrit)

Sources : verdict adverse du chantier comb (`VERIFICATIONS_ADVERSES_R.md`, « Chantier comb »),
`CRITIQUE_COMPLETUDE_R.md` (points 10, 11, 21), `SURCOUCHE_UNIQUE.md` §4.5 et `R_COMB_2.md`.

1. **Phrase « la phase 2 seule n'atteindra pas 95 % » (§7)** — NON CONFIRMÉE pour sa partie estimée
   (vérificateur) : « au plus ~55 700 records, ~90 % » n'est pas une borne (5,95 records par paquet
   est une densité MOYENNE ; les paquets fermés en portent 9,09, soit ~85 100 records, ~91 % ; seule
   borne stricte : 99,5 %) ; L6a hors Live Fire était oublié. Puis **réfutée par la mesure**
   (R-COMB-2) : sous onze leviers (C11), HI_1_13_0 atteint 93,8 % (92,3 % sans l'oracle L9) et
   HI_1_8_0 95,2 %, sur le fixe consolidé.
2. « L1 vaut +24 916 paquets bruts dans la combinaison » ne vaut que pour l'ordre D-RI ; avec L1
   dérivé sans L9 : +26 620 (vérificateur).
3. Le mécanisme du recouvrement L1 × L2 (naissances ratées parce que `ti=43` n'est pas porté) est
   ESTIMÉ ; seul le recouvrement des gains est mesuré (vérificateur). R-COMB-2 mesure l'inverse avec
   un L1a réel : interactions positives (§5.2 de `R_COMB_2.md`).
4. Gate 2 : le critère du plan est NET par film, et la combinaison le tient ; « aucune perte saine »
   (pertes brutes) est une information, pas le critère (vérificateur).
5. §4, L6b : « le gain vient surtout de HI_1_11_0 (+322) » est trompeur ; +322 est la marginale,
   l'interaction de HI_1_11_0 n'est que de +29 ; le surplus vient de HI_1_8_0 (+218) et de HI_1_13_0
   (+176) (vérificateur).
6. `rcaDerivation` exclut du maximum les marches de diagnostic « L1(communes)+L9 » (+24 utiles sur
   `4f77afc1`, effet < 0,001 point) ; « la marche L9 + liaisons communes rend EXACTEMENT L1 dérivé
   sans L9 + L9 » est faux sur les utiles lus (+24) et le hors cadre (+8) (vérificateur).
7. RC-6 « une seule liaison d'oracle de plus » (410 contre 409) est déduit des comptes ; les oracles
   ne sont pas comparés liaison par liaison (vérificateur).
8. **Dénominateur** : le fixe de cette note (HI_1_13_0 2 880 403) est remplacé par le fixe consolidé
   recalculé par R-COMB-2 selon sa règle (HI_1_13_0 3 073 267, corpus 7 758 290). Sur ce fixe, sous
   les six leviers de cette note, seuls `f75e7053` (98,3 %) et `81c02726` (97,5 %) dépassent 95 % ;
   `c75f33b8` tombe à 87,7 % (RC-4, critique point 11).
9. **Vocabulaire** : les « bornes » de cette note (oracle L1, L9) sont des gains d'oracle dans le
   monde de la référence, pas des majorants (critique point 21, `R_COMB_2.md` §8).
10. **Rejoué après J12** : `TestRComb` (24 configurations, 21 films, 506 lignes) rend
    `r_comb_configs.tsv` à l'octet sous la surcouche unique post-J12 ; `TestRCombDerivation` n'est
    pas rejoué (supposé identique). La surcouche `r_comb_overlay/` de cette note est remplacée par
    `surcouche_unique_postj12/` (commande : `SURCOUCHE_UNIQUE.md` §7).
11. Gate de la note : `archlint` était rouge à `fe18bf67c` (`TestNoExpiredTODO`, échéance hors
    campagne), soldé depuis par `feat/v75`.

## 0. Réponse

**L'indicateur de la spec** : records utiles fermés dans des paquets sains, sur le dénominateur
FIXE (§1.4), par build, sous la combinaison complète L1 + L8 + L2 + L9 + L6a + L6b. Il est mesuré
sur les 20 films, et `81c02726` est publié à part.

La combinaison complète existe en deux variantes, qui diffèrent par la marche d'où l'oracle L1 est
dérivé (§5) :
- **ordre D-RI** : L1 est dérivé dans un monde où L9 est déjà posé ;
- **L1 dérivé sans L9** : L1 est dérivé avant L9, puis les deux sont posés ensemble.

| Build | Référence | Combinaison complète, ordre D-RI | Combinaison complète, L1 dérivé sans L9 |
|---|---|---|---|
| HI_1_13_0 (10 films) | 69,8 % | **86,0 %** | **88,0 %** |
| HI_1_12_0 (1 film) | 24,1 % | **88,8 %** | **88,8 %** |
| HI_1_8_0 | 28,5 % | 51,3 % | 50,7 % |
| HI_1_9_0 | 25,5 % | 37,0 % | 36,7 % |
| HI_1_11_0 | 26,0 % | 35,4 % | 35,3 % |
| HI_1_10_0 (3 films) | 17,8 % | 22,2 % | 21,8 % |
| version-33, HI_1_4_1, version-31 | ≤ 0,7 % | ≤ 0,7 % | ≤ 0,7 % |
| **corpus (20 films)** | 39,1 % | 50,6 % | 51,3 % |
| hors corpus : `81c02726` (HI_1_13_0) | 77,1 % | 98,2 % | 98,2 % |

Source : `r_comb_tsv/r_comb_par_build.tsv`, colonne `pct_fixe_sains`.

**La phrase « la phase 2 seule n'atteindra pas 95 % »** :
- **Mesuré.** Sous les six leviers mesurés ensemble, aucun build n'atteint 95 %, quel que soit
  l'ordre de dérivation. Le plus proche est HI_1_12_0, à 88,8 % (un seul film). Vient ensuite
  HI_1_13_0, à 86,0 % ou 88,0 %. L1 et L9 y sont des oracles, donc des bornes de leur mécanisme : un
  lot réel fera au plus autant.
- **Pour HI_1_13_0 : la phrase tient (estimé, §7).** Il manque 200 319 records utiles sains, soit
  7,0 points. Les lots non mesurés ici (L3, L4, L7, R-P6) sont bornés à 9 361 paquets au plus sur
  tout le corpus. À la densité mesurée des paquets non fermés, cela fait 55 700 records au plus.
- **Pour HI_1_12_0 : non décidable sans mesure (ouvert).** Il manque 9 078 records sains, soit
  55,5 % des records lus dans ses paquets non fermés. Aucun lot non mesuré n'a de borne sur ce film.
- **Pour les builds anciens : la phrase tient largement (estimé).** Ils sont à 51 % au plus.
- **Dépendance au dénominateur.** Sur l'ancien dénominateur fixe du RAPPORT §3 (maximum des 14
  marches de bis 1), la même combinaison afficherait 96,2 % ou 98,5 % sur HI_1_13_0, et 103,7 % sur
  HI_1_12_0. Ce dénominateur est périmé : il monte de 11,9 % sur HI_1_13_0 dès qu'on y verse les
  marches mesurées depuis. La phrase ne tient donc que sur le dénominateur recalculé, comme la règle
  de la §6.0 le prescrit.

**Ce que la mesure ensemble apprend en plus** (§4 et §5) :
- les leviers ne s'additionnent pas. Sur HI_1_13_0, la somme des gains seuls vaut +547 609 utiles
  sains, la combinaison +465 529 (ordre D-RI) ou +525 771 (L1 dérivé sans L9) ;
- **L1 et L2 se recouvrent.** Une partie de la borne de L1 rattrape des naissances qu'on rate
  parce que `ti=43` n'est pas porté. Sur `81c02726`, L1 seul vaut +2 884 paquets sains, et son gain
  marginal dans la combinaison vaut 0 ;
- **dans l'ordre D-RI, l'oracle L1 dérivé après L9 fait perdre 2 381 paquets à `4f77afc1`.** La
  cause est 19 liaisons de plus, toutes dans les chunks 27 à 54 (§5). La marginale de L9 devient
  alors négative sur HI_1_13_0 (−29 360 utiles sains). Dérivé sans L9, L9 reste positif (+30 882).

---

## 1. Protocole

### 1.1 Les leviers

Chaque levier est la MÊME copie de recherche que sa mesure séparée. Les contrôles du §2 le
vérifient.

| Levier | Copie de recherche | Mesure séparée d'origine |
|---|---|---|
| L1 | oracle des naissances, régions (i) et (ii) de la sonde M1 (`cmCollecteur.parRegion`) | BIS_1 §4, `oracle-(i)+(ii)` |
| L9 | oracle P1-pont : eid de l'image-clé incomplète liés AU DÉBUT de leur chunk, avec l'archétype du pont du bloc (`b3Diag.imageCle`) | BIS_3 §2, `oracle-P1-pont` |
| L8 | `ti=3 i0` low-frequency porté et `ti=3 i1` lu sur 26 bits (`b3CrochetTi3(true, true)`) | BIS_3 §6, `ti3-i0+i1-26` |
| L2 | grammaire `ti=43` de T7 (`b2vLireTi43`) | BIS_2 §5.4, `grammaire-ti43-T7` |
| L6a | largeurs de la ligne de l'index de plage lu (`bis2C3.parIndex`), avec les bornes des plages 0, 2 et 3 de Live Fire (`mb2_regions_live_fire.tsv`) | BIS_2 §3.3, `reference+lecture-par-index` |
| L6b | sites `flock-position` et `tacmap-displayasset` lus comme le jeu (`bis2SitesJeu`) | BIS_2 §3.1, BIS_4 §3 |

Portée de L6a : il n'agit que là où les bornes des plages sont connues, donc sur les deux films de
Live Fire (`0797ce72`, `60ae07c4`). C'est la portée que lui donne le PLAN §6.1 (« ailleurs nul ou
non mesurable »). Sur les 18 autres films, l'interrupteur ne change rien, par construction : sans
borne, `bis2LargeursDeLIndex` rend la lecture de production. Les configurations qui ne diffèrent
que par L6a n'y sont pas rejouées ; elles sont recopiées et marquées
`rejoue = non (identique par construction)`.

### 1.2 Contexte

- 18 films sous le contexte des instruments (`cmOuvrir`), le même que la carte v2.
- Les deux films de Live Fire sous le contexte de production (`b2pOuvrirProduction` + entrée
  `Live Fire` ou `Live Fire - Ranked` de `map_quant_bounds.json`). Ce contexte est requis par L6a.
  La référence en contexte des instruments y est rejouée comme contrôle (§2).

### 1.3 Les oracles se dérivent sous les mêmes composants

Un oracle se déduit d'une marche : les rejets qu'il répare dépendent des composants lus.
- L'oracle L1 vient de la sonde M1 (`nouveauCollecteur`).
- L'oracle L9 vient du diagnostic des populations (`b3NouveauDiag`).

Pour un jeu de leviers de composant K, la chaîne compte trois marches :
- **A** : K seul, avec M1 et le diagnostic branchés. Elle donne l'oracle L9(K) et l'oracle
  L1(K, sans L9) ;
- **B** : K + L9(K), avec M1 branché. Elle donne l'oracle L1(K, avec L9) ;
- **C** : K + L9(K) + L1(K, avec L9).

Chaque marche est aussi une configuration mesurée.
- `TestRComb` dérive L1 dans l'ordre D-RI : B avant C, la marche d'image-clé avant les naissances.
- `TestRCombDerivation` rejoue la paire L1 + L9 et la combinaison complète avec L1(K, sans L9), et
  compare les deux oracles liaison par liaison.

Configurations par film : 24 (20 hors Live Fire, plus 4 recopiées).
- référence ;
- chaque levier seul ;
- L1 + L9 ;
- la combinaison complète `full` ;
- pour chaque levier X, `full-X` ;
- pour chaque levier de composant X, `full-X-L1` et `full-X-L1-L9`, qui sont les marches de
  dérivation ;
- `full-L1-L9`, les composants seuls.

La contribution marginale d'un levier vaut `full − (full-X)`. Son interaction vaut
`marginal − (X seul − référence)`.

### 1.4 Dénominateurs (PLAN §6.0 « Pourcentages »)

- **variable** : les records utiles lus par la marche elle-même.
- **fixe** : par film, le maximum des records utiles lus sur l'ensemble des marches de la
  campagne :
  - les configurations R-COMB (les deux tests) ;
  - les 14 marches de bis 1 (`mb_variantes.tsv`) ;
  - les marches de bis 3, sauf `largeur-*` (`mb3_variantes.tsv`) ;
  - les marches `ti=3`, sauf le témoin `ti=4` (`mb3_ti3.tsv`) ;
  - les marches `ti=43` (`mb2_ti43.tsv`) ;
  - les A/B de position, instruments et production (`mb2_positions*.tsv`).

  Les témoins NÉGATIFS (largeur d'identifiant fausse, `ti=4` à 26 bits) sont exclus : ce ne sont ni
  des lots ni des oracles. Le maximum mélange les contextes sur Live Fire (instruments et
  production). Source de chaque maximum : `r_comb_denominateurs.tsv`. HI_1_13_0 : **2 880 403**.
  Corpus : 6 585 067.
- **fixe14** : le maximum des 14 marches de bis 1 seules, dénominateur du RAPPORT §3. Il est
  publié pour la continuité seulement. HI_1_13_0 : 2 574 513.

### 1.5 Sondes et commandes

| Fichier (`apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) | Tags | Rôle |
|---|---|---|
| `r_comb_research_test.go` (382 l.) | `research && campagne_overlay` | `TestRComb` : les 24 configurations, le juge sur chaque paquet fermé, la comparaison paquet par paquet avec la référence et avec `full` |
| `r_comb_derivation_research_test.go` (153 l.) | `research && campagne_overlay` | `TestRCombDerivation` : interaction L1 × L9 |
| `r_comb_agregats_research_test.go` (343 l.) | `research` | `TestRCombAgregats` : dénominateurs, tables par build, par film, marginaux. Ne lit que des TSV. |

Surcouche : `r_comb_overlay/`. Ses fichiers `capture.go`, `lecteur_position.go` et
`lecteur_position_exceptions.go` sont identiques à l'octet à ceux de `mesures_bis2_overlay/`
(vérifié par `cmp`). `overlay.json` est réécrit sur ce worktree.

Commandes, depuis `apps/go-api`. `GOCACHE` et `GOTMPDIR` dédiés dans le scratchpad, CGO avec
`C:/msys64/ucrt64/bin`, `-count=1` :

```
CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks
CAMPAGNE_FILMS=bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2,81c02726
CAMPAGNE_SORTIE=<scratchpad>
CAMPAGNE_CATALOGUE=<LevelUp>/data/titles/halo_infinite/reference/map_quant_bounds.json
CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked"
CAMPAGNE_BORNES_sgh_interlock="0:0.266705,51.033180,-9.330649,55.209633,106.171257,78.573174;2:27.834414,23.467054,-9.330649,101.837700,106.171265,78.573174;3:-3500.449951,-3412.616455,-9.330649,4389.524902,4362.876953,880.165344"
go test -tags=research,campagne_overlay -overlay=<worktree>/.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_comb_overlay/overlay.json \
  -count=1 -timeout 240m -run '^TestRComb$' ./internal/games/halo_infinite/film/internal/grammar/
# même chose avec -run '^TestRCombDerivation$'
CAMPAGNE_RCOMB=r_comb_tsv/r_comb_configs.tsv CAMPAGNE_RCOMB_DERIVATION=r_comb_tsv/r_comb_derivation.tsv \
CAMPAGNE_TSV=<dossier de la campagne> CAMPAGNE_HORS_CORPUS=81c02726 CAMPAGNE_SORTIE=r_comb_tsv \
  go test -tags=research -count=1 -run '^TestRCombAgregats$' ./internal/games/halo_infinite/film/internal/grammar/
```

Durées et mémoire :
- `TestRComb` : 2 298 s ; pics de 124 à 468 Mio ; `1c4c63c2` le plus long, 11 min 55 s.
- `TestRCombDerivation` : 1 109 s ; pics de 122 à 381 Mio.
- Deux autres chantiers (`r_loc`, `r_nais`) faisaient tourner leurs propres tests en même temps : les
  durées sont gonflées. Les comptes n'en dépendent pas : chaque process lit ses films seul, dans son
  propre `GOCACHE`.

Sorties (`r_comb_tsv/`) :
- brutes : `r_comb_configs.tsv` (une ligne par film et par configuration), `r_comb_derivation.tsv`,
  `r_comb_derivation_liaisons.tsv`, `r_comb_derivation_detail.tsv` (chaque liaison qui diffère) ;
- agrégats : `r_comb_denominateurs.tsv`, `r_comb_par_build.tsv` (toutes les configurations, par
  build, corpus, et `81c02726` à part), `r_comb_marginaux.tsv`, `r_comb_par_film.tsv`.

## 2. Contrôles (mesurés, tous tenus)

- **Référence = carte v2.** Contexte des instruments sur les 20 films (la ligne
  `controle:reference-instruments` sur Live Fire) : 629 142 paquets, **284 704 fermés**,
  2 588 167 / 5 961 028 records utiles fermés, 264 757 hors cadre. Ce sont les nombres de la carte
  v2 et de MESURES_BIS_1 §0.
- **Référence R-COMB** (Live Fire en production) : 284 619 fermés et 8 383 fermés contredits.
  L'écart est de −81 sur `0797ce72` et −4 sur `60ae07c4`, comme dans BIS_2 §1.3.
- **Chaque levier seul** redonne sa mesure séparée (corpus, net des paquets fermés bruts) :

| Levier seul | R-COMB | Mesure séparée |
|---|---|---|
| L8 | +30 608 | BIS_3 : +30 618 / −10 = +30 608 |
| L2 | +18 093 | BIS_2 : +18 105 / −12 = +18 093 |
| L9 | +1 783 | BIS_3 : +1 784 / −1 = +1 783 |
| L6a | +5 023 (`0797ce72` +3 269 / −3, `60ae07c4` +1 806 / −49) | BIS_2 §3.3 : identique |
| L6b | +467 | BIS_2 §3.1 : +406 (`flock-position`) + 61 (`tacmap-displayasset`) |
| L1 | +28 166 | BIS_1 : +28 168. L'écart de 2 vient du contexte de production sur Live Fire. |

- **Déterminisme.** `TestRCombDerivation` rejoue à l'identique, sur les 21 films, `full` et
  `L1+L9` de `TestRComb` : même nombre de fermés, de sains et d'utiles sains.

## 3. L'indicateur par build (mesuré, `r_comb_par_build.tsv`)

Records utiles fermés en paquets sains. Brut = factices compris.

| Build | Configuration | Variable, sains | Fixe, brut | **Fixe, sains** | Fixe14, sains |
|---|---|---|---|---|---|
| HI_1_13_0 | référence | 80,3 % | 70,0 % | 69,8 % | 78,1 % |
| | full (ordre D-RI) | 86,9 % | 86,2 % | **86,0 %** | 96,2 % |
| | full (L1 dérivé sans L9) | 88,3 % | 88,3 % | **88,0 %** | 98,5 % |
| HI_1_12_0 | référence | 28,9 % | 24,1 % | 24,1 % | 28,1 % |
| | full (les deux) | 88,8 % | 88,8 % | **88,8 %** | 103,7 % |
| HI_1_8_0 | référence → full (D-RI) → full (sans L9) | 30,9 → 51,3 → 50,7 % | | 28,5 → **51,3** → 50,7 % | |
| HI_1_9_0 | idem | 26,3 → 37,0 → 36,8 % | | 25,5 → **37,0** → 36,7 % | |
| HI_1_11_0 | idem | 27,4 → 35,4 → 35,3 % | | 26,0 → **35,4** → 35,3 % | |
| HI_1_10_0 | idem | 19,0 → 24,9 → 24,8 % | | 17,8 → **22,2** → 21,8 % | |
| version-33, HI_1_4_1, version-31 | idem | ≤ 0,8 % | | ≤ 0,7 % | |
| corpus (20 films) | idem | 43,2 → 52,3 → 53,0 % | 39,3 → 50,8 → 51,6 % | 39,1 → **50,6** → 51,3 % | 41,7 → 54,0 → 54,8 % |
| `81c02726` (hors corpus) | idem | 79,9 → 98,2 → 98,2 % | | 77,1 → **98,2** → 98,2 % | |

En paquets (corpus) :
- référence : 284 619 fermés, dont 276 236 sains ;
- full, ordre D-RI : 366 219 fermés (+81 600), dont 357 291 sains (+81 055) ;
- full, L1 dérivé sans L9 : 367 923 fermés, dont 358 979 sains (+82 743).

Sur HI_1_13_0 :
- référence : 224 737 fermés sur 335 960 paquets, 86 968 hors cadre ;
- full, L1 dérivé sans L9 : 280 630 fermés, 32 144 hors cadre.

**Les entrées de contrôle** (l'autre moitié de la spec, item 1.3 `[!]` : seule une borne basse
existe). Le dénominateur « D sièges » de BIS_1 §1 ne dépend pas de la marche :

| Build | Référence | full, ordre D-RI |
|---|---|---|
| HI_1_13_0 | 1 603 762 / 3 255 664 = 49,3 % | 1 955 563 / 3 255 664 = **≥ 60,1 %** |
| HI_1_12_0 | 30 125 / 174 912 | 108 266 / 174 912 = ≥ 61,9 % |
| corpus | 27,5 % | ≥ 35,0 % |

`r_comb_derivation.tsv` ne compte pas les entrées : la variante « sans L9 » n'a pas cette ligne.
Cette moitié reste une borne basse et ne peut pas être déclarée atteinte.

**Films au-dessus de 95 %** (fixe, sains, `r_comb_par_film.tsv`) :
- `f75e7053` 98,3 %, `c75f33b8` 97,2 %, et `81c02726` 98,2 % (hors corpus), tous HI_1_13_0 ;
- `bf15f7ab` est à 94,6 %.

## 4. Contributions marginales et interactions (mesuré, `r_comb_marginaux.tsv`, combinaison `full`, ordre D-RI)

« Seul » = X − référence ; « marginal » = full − (full-X) ; interaction = marginal − seul.
Unités : paquets sains / records utiles fermés sains.

**HI_1_13_0** (dénominateur fixe 2 880 403) :

| Levier | Seul | Marginal dans full | Interaction | Points de l'indicateur (marginal) |
|---|---|---|---|---|
| L8 `ti=3` | +30 597 / +234 374 | **+32 108 / +246 704** | +1 511 / +12 330 | 8,56 |
| L1 oracle (i)+(ii) | +16 980 / +192 158 | +11 915 / +107 616 | **−5 065 / −84 542** | 3,74 |
| L2 `ti=43` | +6 332 / +62 435 | +2 341 / +28 177 | **−3 991 / −34 258** | 0,98 |
| L6a Live Fire | +3 280 / +26 763 | +3 545 / +27 649 | +265 / +886 | 0,96 |
| L6b deux sites | +65 / +1 851 | +241 / +1 892 | +176 / +41 | 0,07 |
| L9 image-clé | +1 048 / +30 028 | **−1 112 / −29 360** | −2 160 / −59 388 | −1,02 |
| somme des seuls | +58 302 / +547 609 | | | |
| combinaison (full − référence) | +53 644 / +465 529 | | −4 658 / −82 080 | |

Avec L1 dérivé sans L9 (seuls L1 et L9 ont leur `full-X` mesuré dans cette variante) :
- L1 : marginal +14 105 / +167 858, interaction −2 875 / −24 300 ;
- L9 : marginal +1 078 / +30 882, interaction +30 / +854 ;
- combinaison : +55 834 / +525 771.

**Corpus (20 films)** :

| Levier | Seul (sains / utiles sains) | Marginal dans full | Interaction |
|---|---|---|---|
| L8 | +30 596 / +234 374 | +32 035 / +244 132 | +1 439 / +9 758 |
| L1 | +28 025 / +357 010 | +24 844 / +288 727 | −3 181 / −68 283 |
| L2 | +17 598 / +142 399 | +14 876 / +119 220 | −2 722 / −23 179 |
| L6a | +5 070 / +38 063 | +6 113 / +44 124 | +1 043 / +6 061 |
| L9 | +1 774 / +45 226 | −271 / −14 096 (+1 417 / +37 071 si L1 dérivé sans L9) | −2 045 / −59 322 |
| L6b | +460 / +5 144 | +971 / +6 610 | +511 / +1 466 |
| combinaison | +81 055 / +757 140 (somme des seuls +83 523 / +822 216) | | |

Lecture (mesuré, sauf mention) :
- **L8 domine dans la combinaison** sur HI_1_13_0. Il se renforce un peu avec les autres
  (+1 511). Il se concentre sur `fb1a1a72` (+20 378 sains marginaux) et `51ebbc0f` (+11 459).
- **L1 et L2 se recouvrent.** Leurs deux interactions sont négatives sur HI_1_13_0, et sur
  `81c02726` la marginale de L1 tombe à 0 sous L2 (L1 seul : +2 884). L'oracle L1 lie des
  naissances dont le NEW est perdu parce qu'un composant `ti=43` désynchronise la lecture. Une fois
  `ti=43` porté, ces naissances se lisent : il n'y a plus rien à lier. Conséquence (estimé) : la
  borne de L1 (+28 168 paquets nets, PLAN §6.1) n'est pas un gain indépendant de L2. Dans la
  combinaison, L1 vaut +24 916 paquets bruts sur le corpus et +11 937 sur HI_1_13_0.
- **Interactions positives sur les autres builds.**
  - HI_1_12_0 : L1 +1 188 sains et L2 +1 276. L1 seul y vaut +401, mais +1 589 dans la
    combinaison : sous `ti=43`, les paquets libérés exposent des naissances que l'oracle lie.
  - HI_1_8_0 : L6a +778 et L9 +206.
- **L8 a une marginale négative sur HI_1_10_0** (−74 sains, −2 594 utiles, entièrement sur
  `084a804d`), alors que L8 seul y est nul. L'oracle L1 dérivé avec ou sans L8 diffère d'une seule
  liaison (410 contre 409), et cela suffit à changer 65 paquets. C'est la sensibilité de l'oracle M1,
  pas un effet de la grammaire `ti=3` : sans L1, L8 n'y change rien (`full-L8-L1` contre `full-L1`).
- **L6b** gagne plus dans la combinaison que seul (+971 contre +460 sains sur le corpus). Le gain
  vient surtout de HI_1_11_0 (+322) et de HI_1_8_0 (+218, alors que L6b seul y est nul).

## 5. L'interaction L1 × L9 : un artefact de dérivation de l'oracle (mesuré, `r_comb_derivation*.tsv`)

Sur `4f77afc1` (HI_1_13_0), sans composants :
- **L1 + L9 avec L1 dérivé après L9 : +3 221 / −2 381 paquets** contre la référence, dont
  **2 386 paquets sains perdus** ;
- **L1 + L9 avec L1 dérivé sans L9 : +2 980 / −5.**

La différence entre les deux oracles, liaison par liaison :
- 125 liaisons communes ;
- 18 liaisons propres à la dérivation « avec L9 » (19 sous les composants), sur des eid
  `ti=41`, `ti=42`, `ti=35`, `ti=40` et `ti=0`, dans les chunks 27, 47, 51 à 54 ;
- 2 liaisons propres à la dérivation « sans L9 » ;
- aucune liaison où la même clé change d'archétype.

La marche « L9 + liaisons communes seulement » rend EXACTEMENT la marche « L1 dérivé sans L9 + L9 » :
- sans composants, 25 885 fermés et 24 959 sains ;
- sous les composants, 26 405 fermés, 25 455 sains, +3 501 / −6.

Les 2 liaisons propres à « sans L9 » ne changent donc rien. Les pertes viennent des liaisons que
M1 trouve SEULEMENT quand L9 est posé. Liste complète : `r_comb_derivation_detail.tsv`.

Sur les autres films, l'écart entre les deux dérivations est petit et de signe variable :

| Film | Liaisons en plus (avec L9) / en moins | Effet |
|---|---|---|
| `1c4c63c2` | 28 à 29 / 8 | « avec L9 » meilleur de +101 sains |
| `111fa685` | 11 / 3 | « avec L9 » meilleur de +41 |
| `a349fea8` | 10 / 0 | aucun |
| `60ae07c4`, `084a804d`, `11de8353`, `e5adf7b2` | 2 à 6 / 0 ou 1 | « avec L9 » meilleur : +206, +121, +23, +10 sains |
| les autres | 0 / 0 | identiques |

Par build, « avec L9 » gagne sur HI_1_10_0, HI_1_8_0, HI_1_9_0 et HI_1_11_0 (+0,1 à +0,6 point). Il
ne perd que sur HI_1_13_0, de 2,0 points, entièrement sur `4f77afc1`.

Lecture :
- **Mesuré** : l'oracle L1 n'est pas une borne stable en combinaison. Son contenu dépend du monde
  où M1 cherche, et une poignée de liaisons fausses suffit à perdre des milliers de paquets. C'est
  déjà la réserve D-7 de BIS_1 §8 (`R(6)` contredit le pont), vue ici en combinaison.
- **Estimé** : un lot L1 réel qui lit les naissances dans le monde de L9 rencontrera ces mêmes eid.
  Le gate « aucun film en baisse » de L1 doit se jouer sur la tête qui porte déjà L9, et
  `4f77afc1` doit être nommé comme témoin. Ce n'est pas une branche par film : c'est le film où
  l'interaction se voit.

## 6. Par film : le critère du gate 2 (mesuré, `r_comb_par_film.tsv`)

**En net** (critère du gate 2, paquets sains et utiles sains contre la référence) : **aucun film ne
baisse** sous la combinaison complète, dans les deux variantes. Les moins bons :
- `50247b26` (version-31) +1 / +24 ;
- `a521164d` (HI_1_4_1) +69 / 0 ;
- `1c4c63c2` (HI_1_10_0) +766 à +867 sains.

**En brut, des paquets sains sont perdus** sur 9 films dans l'ordre D-RI, et 9 films avec L1 dérivé
sans L9 :

| Film | Ordre D-RI | L1 dérivé sans L9 |
|---|---|---|
| `1c4c63c2` | 1 216 | 1 216 |
| `4f77afc1` | 2 389 | 11 |
| `084a804d` | 214 | 244 |
| `111fa685` | 59 | 59 |
| `60ae07c4` | 21 | 21 |
| `396cfc92` | 11 | 11 |
| `e5adf7b2` | 9 | 9 |
| `11de8353` | 3 | 3 |
| `fb1a1a72` | 1 | 1 |

Chaque perte se rattache à un levier seul (colonne `ps_ref` de `r_comb_configs.tsv`) :
- **L1 (oracle)** : `1c4c63c2` 1 160, `084a804d` 237, `111fa685` 59, `396cfc92` 11,
  `e5adf7b2` 9, `51ebbc0f` 8, `4f77afc1` 6 ;
- **L6a** : `60ae07c4` 33 ;
- **L2** : `1c4c63c2` 47, `084a804d` 4 ;
- **L6b** : `60ae07c4` 8.

Les composants seuls (`full-L1-L9`) perdent 91 paquets sains sur tout le corpus, dont 49 sur
`1c4c63c2` et 33 sur `60ae07c4`. Presque toutes les pertes saines de la combinaison viennent donc
de l'oracle L1, déjà connues séparément (BIS_1 §3 : 1 454 sains perdus), plus l'interaction du §5.

## 7. La phrase « la phase 2 seule n'atteindra pas 95 % »

**Mesuré.** Sous la combinaison des six leviers, aucun build n'atteint 95 % sur le dénominateur
fixe, dans aucun des deux ordres de dérivation :
- HI_1_12_0 : 88,8 % ;
- HI_1_13_0 : 86,0 % ou 88,0 % ;
- les autres builds : ≤ 51,3 %.

Le dénominateur variable ne change pas ce verdict : HI_1_13_0 y est au plus à 88,3 %. Deux réserves :
- L1 et L9 y sont des oracles : un lot réel fera au plus autant que leur mécanisme ;
- L6a ne porte que sur Live Fire.

**HI_1_13_0 : la phrase tient (estimé).**
1. Mesuré : il faut 0,95 × 2 880 403 = 2 736 383 records utiles sains. La meilleure variante en a
   2 536 064 : il en manque **200 319**, soit 7,0 points.
2. Mesuré : il reste 55 330 paquets non fermés, qui portent 329 270 records utiles lus et non
   fermés, soit 5,95 par paquet.
3. Les lots non mesurés ici ont des bornes publiées, en paquets, sur TOUT le corpus (PLAN §6.1) :
   - L3 : ≤ 4 331, estimé ;
   - L4 : ≤ 1 666, estimé ;
   - L7 : ≤ 137, mesuré ;
   - R-P6 : +3 227 bruts pour la meilleure borne mesurée ;
   - LS : son effet sur la fermeture n'est pas mesuré, et c'est une part de la région (ii), déjà
     dans l'oracle L1.

   Soit ≤ 9 361 paquets. Même tous sur HI_1_13_0, sans recouvrement avec L1, à 5,95 records par
   paquet, cela fait ≤ 55 700 records, soit +1,9 point : **≤ 90 % environ**.
4. De plus, le dénominateur fixe monte encore quand un lot lit plus loin.

**HI_1_12_0 : la phrase n'est pas décidable sans mesure (ouvert).**
- Mesuré : il manque 9 078 records sains, soit 6,2 points, sur un seul film, `bcb6d393`.
- Mesuré : il reste 3 031 paquets non fermés (dont 2 436 hors cadre), qui portent 16 343 records
  utiles lus et non fermés. Il faudrait en fermer 55,5 %.
- Aucun lot non mesuré n'a de borne PAR FILM sur `bcb6d393`.

**Builds HI_1_8_0 à HI_1_11_0 et versions anciennes : la phrase tient (estimé).**
- Ils sont à 51,3 % au plus. Les écarts (≥ 43 points) dépassent de loin toute borne publiée.
- Sur les trois builds les plus anciens, presque rien n'est lu : ≤ 0,7 %.

**Réserve de dénominateur (mesuré).** L'ancien dénominateur fixe (14 marches de bis 1, RAPPORT §3)
afficherait 96,2 % ou 98,5 % sur HI_1_13_0 et 103,7 % sur HI_1_12_0. Un pourcentage au-dessus de
100 % prouve que ce dénominateur est périmé : la marche combinée lit plus de records utiles que les
14 marches. Le PLAN §6.0 le prévoit (« recalculé à chaque vague »). Les pourcentages « sous
l'oracle » du RAPPORT §3 sont à lire sur le nouveau maximum : sur ce dénominateur, la référence
HI_1_13_0 vaut 69,8 % et non plus 78,1 %.

**Proposition de correction du RAPPORT §3** (non appliquée : ce document ne modifie ni le RAPPORT
ni le PLAN) :

> « la phase 2 seule ne fera pas atteindre le déclencheur sur HI_1_13_0 ni sur les builds
> anciens : mesuré sous les six leviers combinés (86,0 à 88,0 %, R-COMB), estimé pour les lots non
> mesurés (≤ 90 %) ; non décidé sur HI_1_12_0 (88,8 % mesuré, un film) ».

## 8. Limites

- **Oracles.** L1 et L9 sont des oracles : la borne d'UN mécanisme, pas un lot. Leur combinaison est
  sensible à l'ordre de dérivation (§5). Les liaisons de L1 portent le `R(6)` de M1 (réserve D-7).
- **L9** est dérivé sans L1, puis seulement L1 sans ou avec L9. La dérivation croisée complète
  (L9 dérivé dans le monde de L1) n'est pas jouée : c'est un effet de second ordre, chunk par chunk.
- **Surcouche** prise sur `69564ef7d` (pré-J12) : à re-synchroniser à la fusion de J12, comme les
  copies de `mesures_bis2_overlay/` (PLAN §6.0).
- **L6a** ne porte que sur Live Fire. Les cartes à deux sbsp restent hors mesure, faute de l'ordre
  des plages (R-L6).
- **Juge** : trois invariants seulement. « Sain » ne prouve pas une lecture juste.
- **Dénominateur fixe** : maximum de marches dont certaines lisent faux (oracles, A/B de position).
  C'est la règle du PLAN §6.0 ; il ne compte pas les records écrits par le jeu.
- **Entrées** : dénominateur en borne haute (sièges), donc parts en bornes basses. Elles ne sont pas
  comptées sous la variante « sans L9 ».
- **`81c02726`** n'est pas dans le corpus des 20 films. Il est publié à part.

## 9. Découvertes (consignées, non traitées)

- **RC-1** — L'oracle L1 dérivé dans le monde de L9 perd 2 381 paquets (2 386 sains) sur `4f77afc1`,
  par 18 à 19 liaisons que M1 ne trouve qu'avec L9 posé (§5). Pour le gate de L1 : le jouer sur une
  tête qui porte L9, et nommer `4f77afc1`.
- **RC-2** — L1 et L2 se recouvrent : la borne de L1 compte des naissances perdues à cause de
  `ti=43`. Sur `81c02726`, la marginale de L1 sous L2 vaut 0. Le classement « par gain mesuré » du
  PLAN §6.1 additionne des bornes qui se recouvrent : sous la combinaison, L1 vaut +24 916 paquets
  bruts sur le corpus, et non +28 168.
- **RC-3** — Le dénominateur fixe14 du RAPPORT §3 est périmé : on atteint 103,7 % sur HI_1_12_0. Le
  nouveau maximum est publié dans `r_comb_denominateurs.tsv`. HI_1_13_0 : 2 880 403, soit +11,9 %.
- **RC-4** — Trois films HI_1_13_0 dépassent 95 % sous la combinaison : `f75e7053`, `c75f33b8` et
  `81c02726` (hors corpus).
- **RC-5** — `go test ./internal/archlint/` est ROUGE sur ce HEAD, pour une seule cause :
  `TestNoExpiredTODO`. Un `TODO(expiry:2026-10-01)` de
  `internal/api/handlers/json_huma_coverage_test.go:34` est échu depuis le 2026-10-02. L'échec est
  indépendant de cette recherche : identique avec et sans les trois sondes. Tous les autres tests du
  paquet passent (§10).
- **RC-6** — L8 a une marginale négative sur `084a804d` (−73 sains), à travers une seule liaison
  d'oracle de plus (410 contre 409). C'est une autre manifestation de RC-1 : un oracle qui change
  avec le monde.

## 10. Gate (sorties relevées le 2026-10-02, depuis `apps/go-api` de ce worktree)

- `gofmt -l` sur `r_comb_research_test.go`, `r_comb_derivation_research_test.go` et
  `r_comb_agregats_research_test.go` : **vide**.
- `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/` : sans sortie (OK).
- `go vet -tags=research,campagne_overlay -overlay=<r_comb_overlay/overlay.json> ./internal/games/halo_infinite/film/internal/grammar/` :
  sans sortie (OK).
- `go test -count=1 ./internal/archlint/` : **FAIL**, `--- FAIL: TestNoExpiredTODO` seul.
  - C'est le même échec avec les trois sondes retirées du dossier : elles ont été déplacées dans le
    scratchpad, le test joué, puis elles ont été remises.
  - La cause est RC-5 : une échéance de date dans un fichier hors périmètre, que cette mission ne
    peut pas modifier.
  - `go test -count=1 -skip '^TestNoExpiredTODO$' ./internal/archlint/` : **ok** (29 s). Il couvre
    le ratchet de taille (sondes de 153 à 382 lignes), le tag `research` en ligne 1 et les suffixes
    de garde.
- Tests : `TestRComb` PASS (2 298 s), `TestRCombDerivation` PASS (1 109 s), `TestRCombAgregats`
  PASS. Sans variables d'environnement, les trois tests rendent SKIP.
- `grammar.Rev` est inchangé : aucun fichier non-test n'a été modifié, et `git status` ne montre que
  des fichiers neufs.
