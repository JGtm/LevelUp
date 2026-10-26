# Mesures bis 1 — campagne de grammaire, réponse à la critique de complétude (2026-10-01)

> Répond aux points **10, 11, 12, 13, 21, 22, 23 et 29** de `CRITIQUE_COMPLETUDE_1.md` (item 1.3 du
> plan, lot L1). Worktree `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, rien de
> commité. Aucun fichier de production touché, `grammar.Rev` inchangé (seuls des `*_research_test.go`
> sous tag `research`). Aucune base, aucune cuisson. Films en lecture seule, 20 films (les 19 témoins de
> `config/replay_corpus.toml` + `1c4c63c2`), 9 builds.
>
> Convention : **mesuré** = compté par la sonde sur les 20 films ; **estimé** = dérivé d'une mesure par
> une hypothèse écrite ; **sain** = paquet fermé qui ne contredit aucun des trois invariants de
> l'écrivain instrumentés (ordre NEW*/DELTA*/DEL* et slots croissants, masque écrivable, vue C
> possible : `campagne_invariants_research_test.go`) ; **factice** = paquet fermé qui en contredit au
> moins un. « Sain » n'est pas « juste » : c'est l'absence de contradiction détectable.
>
> **Corrections du 2026-10-02** (critique de complétude n° 2, points N15 et N19), faites en place et
> marquées « corrigé le 2026-10-02 » : colonne « Estimateur » du §1 et sa phrase d'égalité ; valeur
> « sous l'oracle » des entrées au §1 ; écart 598 / 623 eid au §9.

## 0. Protocole et contrôles

**Sonde** : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/campagne_bis1_research_test.go`
(neuf), plus trois ajouts minimes aux sondes de l'étape 4 : `campagne_naissances_research_test.go`
(deux champs du collecteur : liaisons d'oracle par région, eid rejetés rangés),
`campagne_m1_research_test.go` (deux appels qui remplissent ces champs), `campagne_marche_research_test.go`
(le paquet garde ses records utiles fermés). Aucune sortie de `TestCampagneMesuresCiblees` ne change.

Commande (depuis `apps/go-api`, `GOCACHE` dédié, CGO) :
`CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_FILMS=<20 ids> CAMPAGNE_SORTIE=<scratchpad>
go test -tags=research -count=1 -timeout 180m -run '^TestCampagneMesuresBis1$' ./internal/games/halo_infinite/film/internal/grammar/`.
Un film à la fois, sentinelle `filmproc` 4 Gio, pics de 94 à 333 Mio, 14 marches par film, 1 309 s.
Deux passes : la seconde ajoute le statut de référence de chaque paquet perdu ; `mb_variantes.tsv`,
`mb_entrees.tsv` et `mb_rejets.tsv` sont **identiques à l'octet** entre les deux.

**Contrôles (mesurés, tous tenus)** : la marche `reference` rend exactement la carte v2 et MESURES §2 —
284 704 paquets fermés, 2 588 167 / 5 961 028 records utiles, 264 757 hors cadre ; 8 388 fermés
contredits (MESURES §T2-4) ; 2 301 194 entrées fermées dont 2 301 082 utiles (CARTE §5). `oracle-NEW`
+29 769 / −2 158 ; `tete-bloc` +19 105 / −7 751 ; `oracle-bloc` +2 090 / −3 ;
`oracle-bloc+tete-bloc` +20 809 / −7 535 (les chiffres TSV cités par la critique) ; 5 170 liaisons.

**Les marches** (`mb_variantes.tsv`) : `reference` ; `oracle-NEW` ; l'oracle restreint à UNE région
de M1 : `oracle-(i)`, `oracle-(ii)`, `oracle-(iii')-rejet`, `oracle-(iii')-ouverte`, `oracle-(iii)` ;
`oracle-(i)+(ii)` ; `oracle-bloc` ; `tete-bloc` ; `oracle-bloc+tete-bloc` ; et trois localisateurs
filtrés par les invariants de l'écrivain (§2) : `bande+inv`, `tete-bloc+inv`, `tete-bloc+inv+paquet`.
Chaque paquet fermé de chaque marche passe aux invariants (juge, `mb_juge.tsv`).

Sorties : `mesures_bis_tsv/` — brutes `mb_variantes.tsv`, `mb_juge.tsv`, `mb_entrees.tsv`,
`mb_rejets.tsv` (une ligne par film et par clé) ; agrégats `mb_agg_variantes_par_build.tsv`,
`mb_agg_juge_par_build.tsv`, `mb_agg_fermes_sains_par_build.tsv`, `mb_agg_l1a_par_film.tsv`,
`mb_agg_entrees_par_build.txt`.

---

## 1. Item 1.3 — dénominateur indépendant des entrées de contrôle (points 22, 38)

**Règle** (T5, établie chez l'écrivain `FUN_142f2c3b0` / `FUN_14076b0e8`) : la vue C d'un film ne
porte que des entrées kind 0, **au plus une par joueur** (index de contrôle = place du joueur),
index croissants, au plus 32. Elle donne une **borne haute** du nombre d'entrées écrites par paquet :
le nombre de joueurs. Trois dénominateurs indépendants de la fermeture, sommés sur TOUS les paquets
delta marchés (629 142 ; listes non localisées comprises) :

- `D sièges` = paquets × sièges occupés de la table du film (`ScanFilmPlayerTable`) ; indisponible sur
  `50247b26` (version-31) et `a349fea8` (version-33), films sans section d'identification ;
- `D sièges ∪ vus` = paquets × |sièges ∪ index vus dans une vue C fermée du film| (couvre les joueurs
  arrivés en cours de partie : 2 567 entrées fermées hors sièges, dont 2 496 sur HI_1_11_0) ;
- `D ti9` = Σ paquets × entités `managed-player` (`ti=9`) liées au monde au début de leur chunk.

Numérateur : les entrées utiles (portant le bloc 0x68) des vues C fermées.

| Build | Entrées utiles fermées | D sièges | D sièges ∪ vus | D ti9 | Estimateur de la CARTE §5, appliqué au build agrégé | Part des paquets fermés |
|---|---|---|---|---|---|---|
| HI_1_13_0 | 1 619 501 | 49,7 % | **45,0 %** | 49,3 % | 66,9 % | 66,9 % |
| HI_1_12_0 | 30 136 | 17,2 % | 17,2 % | 17,2 % | 26,7 % | 26,7 % |
| HI_1_11_0 | 76 222 | 19,7 % | 18,9 % | 19,1 % | 25,5 % | 25,5 % |
| HI_1_10_0 | 420 612 | 13,5 % | 11,0 % | 13,6 % | 22,2 % | 22,2 % |
| HI_1_9_0 | 75 327 | 17,8 % | 17,8 % | 17,5 % | 32,6 % | 32,6 % |
| HI_1_8_0 | 79 279 | 19,9 % | 13,3 % | 19,9 % | 28,1 % | 28,1 % |
| HI_1_4_1, version-31, version-33 | 5 | ≈ 0 % | ≈ 0 % | ≈ 0 % | version-33 1,6 %, version-31 0,9 %, HI_1_4_1 « - » (aucune entrée utile) | version-33 1,6 %, version-31 0,9 %, HI_1_4_1 6,3 % |
| **corpus** | 2 301 082 | 28,7 % | **24,2 %** | 25,1 % | 45,3 % | 45,3 % |

**Vérification de la règle sur les paquets fermés (mesuré)** : aucune vue C fermée ne porte plus de 32
entrées ; le nombre d'entrées dépasse le nombre d'entités `ti=9` du chunk dans 2 paquets fermés
seulement (HI_1_8_0, +1) et le nombre de sièges dans 11 (joueurs arrivés en cours, HI_1_8_0 et
HI_1_11_0). La borne tient.

**Mais la borne n'est pas atteinte (mesuré)** : sur HI_1_13_0, seuls 43 061 des 224 818 paquets fermés
(19 %) portent une entrée par siège ; 50 394 en ont une de moins, 52 942 deux de moins, 78 421 plus
de deux de moins. Sur HI_1_9_0, 12 paquets fermés sur 5 739 ont une entrée par siège. Un joueur
n'a pas d'entrée dans chaque paquet. Le dénominateur « une par joueur » surestime donc le nombre
d'entrées écrites, et les parts du tableau sont des **bornes basses** (mesurées) de la vraie part.

**L'estimateur actuel est-il tautologique ? Oui, prouvé et mesuré.** CARTE §5 pose, par film,
D = U + (U / F) × N (U entrées utiles fermées, F paquets fermés, N paquets non fermés), d'où
U / D = F / (F + N) : la part estimée **est** la part des paquets fermés. Les deux dernières colonnes
du tableau sont égales build par build (HI_1_13_0 : 224 818 / 335 960 = 66,9 %), **sauf HI_1_4_1**
(corrigé le 2026-10-02) : sans entrée utile fermée (U = 0), l'estimateur y est indéfini, alors que
6,3 % de ses paquets ferment.

**Écart avec les valeurs publiées par la CARTE §5 (corrigé le 2026-10-02)** : cette colonne applique
l'estimateur au build (ou au corpus) AGRÉGÉ (`mb_agg_entrees_par_build.txt`). La CARTE §5 l'applique
FILM PAR FILM, puis somme numérateurs et dénominateurs : le résultat pondère les films et s'écarte de
la part agrégée des paquets fermés quand un build compte plusieurs films de densités d'entrées
différentes. D'où HI_1_10_0 21,9 % à la CARTE contre 22,2 % ici (trois films), et le corpus 43,6 %
contre 45,3 %. HI_1_13_0 (dix films) donne 66,9 % des deux façons. Les deux calculs sont
tautologiques : film par film, la part estimée égale la part des paquets fermés. Pondérer par le nombre
de joueurs n'y change rien : ce nombre est constant dans un film, il se simplifie. L'estimateur ne
mesure donc rien des entrées ; il suppose que les paquets non fermés portent autant d'entrées que les
fermés, ce qu'aucune mesure ne soutient.

**Statut de l'item 1.3** : la moitié « entrées » du déclencheur se calcule désormais avec un
dénominateur indépendant, mais seulement en **borne basse** : HI_1_13_0 ≥ 45,0 % (sièges ∪ vus) à
≥ 49,7 % (sièges) ; corpus ≥ 24,2 %. Aucune borne haute indépendante n'existe sans lire les paquets non
fermés (la vraie part est dans [borne basse, 100 %]). Même sous l'oracle des naissances (dénominateur
fixe, §7), HI_1_13_0 ≥ 48,8 % sous l'oracle (i)+(ii) (≥ 48,5 % sous l'oracle-NEW ; corrigé le
2026-10-02) : la moitié « entrées » **ne peut pas être déclarée atteinte**. Le
dénominateur exact demanderait la règle qui décide qu'un joueur a une entrée dans un paquet (état
`t+0x20` ∈ {1, 2} et tampon non vide chez `FUN_142f2c3b0` : quand l'écrivain d'entrée est-il appelé
pour un joueur ?), non lue. Proposition de statut : `[~]` dénominateur borne basse livré ici,
dénominateur exact `[!]` (lecture Ghidra de l'appelant de `FUN_14076b0e8` à faire).

---

## 2. L1a mesuré — « bande OU bloc » + filtre des invariants de l'écrivain (point 23)

**Ce qui est mesuré** (`cmTeteInv`) : candidats NEW de tête admis par la bande de production OU par
l'allocation de leur eid au bloc de type 1 (comme `tete-bloc`), puis :

1. un candidat n'est gardé que si son NEW a un masque écrivable pour son `R(6)` (aucun bit au-delà de
   l'archétype, dense > 7 bits, épars croissant) ;
2. chemin « chaîne » (localisateur strict trouvé) : la chaîne doit toujours finir au bit près sur le
   début strict, ET respecter l'ordre de l'écrivain (NEW* puis DELTA*, slots croissants, aucun DEL),
   ET chacun de ses records doit avoir un masque écrivable ; sinon on essaie le candidat suivant ;
3. chemin « fermeture » (localisateur strict muet) : le paquet doit fermer ET le paquet fermé ne doit
   contredire aucun invariant ;
4. variante `+paquet` : sur le chemin chaîne aussi, le paquet décodé depuis le début retenu ne doit
   contredire aucun invariant, sinon le début strict est gardé.

`bande+inv` applique les mêmes filtres aux seuls candidats de la bande de production : il isole
l'effet du filtre.

**Résultat brut par build (mesuré, paquets fermés ; gagnés / perdus contre la référence)** :

| Build | tete-bloc | tete-bloc+inv | tete-bloc+inv+paquet | bande+inv |
|---|---|---|---|---|
| HI_1_13_0 | +15 248 / −3 055 | +14 627 / −1 692 | +14 381 / −2 615 | +603 / −1 291 |
| HI_1_12_0 | +695 / 0 | +684 / −4 | +541 / −4 | 0 / −4 |
| HI_1_11_0 | +221 / −336 | +77 / −222 | +77 / −222 | 0 / −179 |
| HI_1_10_0 | +2 231 / −2 717 | +756 / −6 350 | +756 / −6 350 | +24 / −5 593 |
| HI_1_9_0 | +337 / −264 | +54 / −193 | +54 / −193 | 0 / −123 |
| HI_1_8_0 | +319 / −1 379 | +187 / −414 | +187 / −414 | 0 / −103 |
| corpus | +19 105 / −7 751 | +16 407 / −8 925 | +16 018 / −9 848 | +627 / −7 343 |

Le compte brut mélange deux choses : le filtre RETIRE des fermetures factices de la référence (le
gate D2 l'admet), et il en perd de saines. Ventilation des **perdus** par leur état en référence
(mesuré, corpus) : `tete-bloc+inv` perd 7 004 paquets factices (10 960 utiles) et **1 921 paquets
sains (38 685 utiles)** ; `bande+inv` perd 7 314 factices et 29 sains ; `tete-bloc` perd 125 factices et
7 626 sains (139 294 utiles).

**Bilan en fermetures saines (mesuré, `mb_agg_fermes_sains_par_build.tsv`)**, delta contre la
référence (paquets fermés sains ; records utiles fermés dans des paquets sains) :

| Build | tete-bloc | tete-bloc+inv | bande+inv |
|---|---|---|---|
| HI_1_13_0 | +10 891 ; +84 825 | **+14 380 ; +153 699** | +763 ; +20 289 |
| HI_1_12_0 | +682 ; +6 155 | +684 ; +6 165 | +1 ; +10 |
| HI_1_11_0 | −270 ; −7 837 | +17 ; **−1 904** | −17 ; −652 |
| HI_1_10_0 | −2 050 ; −58 006 | +753 ; **−21 094** | +835 ; +711 |
| HI_1_9_0 | −100 ; −2 406 | −13 ; **−1 404** | +3 ; +39 |
| HI_1_8_0 | −1 219 ; −8 806 | −129 ; **−1 953** | +3 ; +18 |
| corpus | +7 955 ; +13 927 | +15 714 ; +133 511 | +1 588 ; +20 415 |

**Réponse : non, le filtre n'annule pas les pertes sur HI_1_8_0 à HI_1_11_0.** Il les réduit (records
utiles sains perdus : HI_1_10_0 −58 006 → −21 094, HI_1_8_0 −8 806 → −1 953, HI_1_11_0 −7 837 →
−1 904, HI_1_9_0 −2 406 → −1 404), mais les quatre builds restent en baisse d'utiles sains. Par film
(`mb_agg_l1a_par_film.tsv`), les six films de HI_1_8_0 à HI_1_11_0 baissent tous ; le plus touché est
`1c4c63c2` (HI_1_10_0) : 683 paquets sains perdus, −13 616 utiles sains. Le niveau `+paquet` ne fait
pas mieux (identique sur ces builds, moins bon sur HI_1_13_0 et HI_1_12_0). Sur HI_1_12_0 et HI_1_13_0,
le filtre fait mieux que la variante sans filtre (HI_1_13_0 : +153 699 utiles sains contre +84 825) ;
sur HI_1_13_0, trois films perdent des paquets sains (`4f77afc1` : 307, `396cfc92` : 114, `51ebbc0f` :
46 ; les sept autres 0), et les dix gagnent en utiles sains.

Conséquence pour L1a (estimé à partir de ces mesures) : le gate « aucune baisse sur aucun build, sauf
fermeture factice retirée » **n'est pas tenu** par `tete-bloc+inv` (perte saine sur quatre builds). Le
filtre d'invariants ne suffit pas : l'élargissement « OU bloc » reste nuisible avant HI_1_12_0, où la
table de l'allocateur ne prédit rien (MESURES §T1-3). Une restriction de l'élargissement aux builds
dont l'allocateur est prédictif serait une branche sur le build : à proscrire ; une condition
mesurable par film (prédiction de l'allocateur vérifiée sur les NEW lus du chunk) est la piste.

**Effet propre du filtre (mesuré, `bande+inv`)** : des 8 388 fermetures factices de la référence,
7 314 ne ferment plus, 990 restent fermées et deviennent saines (le monde a changé en amont), 84
restent contredites ; il ne perd que 29 paquets sains et en gagne 627. C'est l'effet attendu du lot L0 appliqué au localisateur.

---

## 3. Les paquets gagnés passés aux invariants de l'écrivain (point 21)

Part des paquets GAGNÉS (fermés dans la variante, non fermés en référence) qui contredisent un
invariant (mesuré, `mb_agg_variantes_par_build.tsv`, colonne `gagnes_contredits`) :

| Variante | Gagnés | dont factices | Part | Perdus (dont sains en référence) |
|---|---|---|---|---|
| oracle-NEW | 29 769 | 461 | **1,5 %** | 2 158 (1 901) |
| oracle-(i) | 16 974 | 84 | 0,5 % | 591 (534) |
| oracle-(ii) | 12 805 | 200 | 1,6 % | 1 265 (1 182) |
| oracle-(i)+(ii) | 29 755 | 280 | 0,9 % | 1 587 (1 454) |
| oracle-(iii')-rejet | 363 | 147 | **40,5 %** | 885 (758) |
| oracle-(iii')-ouverte | 21 | 6 | 28,6 % | 2 (1) |
| oracle-(iii) | 25 | 25 | **100 %** | 90 (89) |
| oracle-bloc | 2 090 | 26 | 1,2 % | 3 (0) |
| tete-bloc | 19 105 | 3 320 | **17,4 %** | 7 751 (7 626) |
| oracle-bloc+tete-bloc | 20 809 | 3 356 | 16,1 % | 7 535 (7 416) |
| tete-bloc+inv | 16 407 | 56 | 0,3 % | 8 925 (1 921) |
| tete-bloc+inv+paquet | 16 018 | 56 | 0,3 % | 9 848 (2 840) |
| bande+inv | 627 | 0 | 0 % | 7 343 (29) |

Par build pour `oracle-NEW` : HI_1_13_0 172 / 16 918 (1,0 %), HI_1_10_0 214 / 3 942 (5,4 %), HI_1_8_0
49 / 6 219 (0,8 %), HI_1_9_0 15 / 1 291, HI_1_11_0 10 / 983, HI_1_12_0 1 / 412. Pour `tete-bloc` :
HI_1_10_0 1 563 / 2 231 (**70 %**), HI_1_11_0 153 / 221 (69 %), HI_1_9_0 167 / 337 (50 %), HI_1_8_0
156 / 319 (49 %), HI_1_13_0 1 235 / 15 248 (8,1 %).

Nature des contradictions dans les gains (mesuré, corpus) : `oracle-NEW` masque 221, ordre + masque
163, ordre 61, vue C 10, autres 6 ; `tete-bloc` masque 2 668, ordre + masque 500, ordre 61, masque +
vue C 62, vue C 25.

Lecture : les gains de l'oracle sur les régions (i) et (ii) sont presque tous sains ; ceux des régions
(iii) et (iii') sont majoritairement factices et leurs pertes dominent (§4). La perte nette de
`tete-bloc` sur les vieux builds vient de chaînes fausses : 70 % de ses gains sur HI_1_10_0 sont
factices.

---

## 4. La borne de L1 ventilée par région et par sous-lot ; gain propre de L1a (point 29)

Oracle restreint à une seule région de M1 (mesuré ; corpus ; paquets gagnés / perdus, net ; records
utiles fermés ; fermetures saines nettes) :

| Marche | Liaisons | Gagnés / perdus | Net | Utiles fermés | Paquets sains nets ; utiles sains |
|---|---|---|---|---|---|
| oracle-(i) — tête des paquets à événements (**L1a**) | 1 371 | +16 974 / −591 | **+16 383** | +196 804 | +16 370 ; +195 456 |
| oracle-(ii) — paquets à événements non localisés (**L1b**) | 1 910 | +12 805 / −1 265 | **+11 540** | +162 964 | +11 418 ; +161 019 |
| oracle-(iii') après rejet (R-L1 a) | 1 289 | +363 / −885 | **−522** | −13 731 | −530 ; −14 494 |
| oracle-(iii') après « ouverte » | 89 | +21 / −2 | +19 | +132 | +14 ; +93 |
| oracle-(iii) NEW lu désynchronisé (aucun lot) | 511 | +25 / −90 | **−65** | −724 | −89 ; −847 |
| somme des cinq marches séparées | 5 170 | | +27 355 | +345 445 | |
| **oracle-(i)+(ii)** | 3 281 | +29 755 / −1 587 | **+28 168** | **+360 182** | +28 027 ; +357 028 |
| oracle-NEW (les cinq régions) | 5 170 | +29 769 / −2 158 | +27 611 | +344 129 | +27 435 ; +340 168 |

HI_1_13_0 : (i) +10 328 net, +105 107 utiles ; (ii) +6 815, +95 674 ; (iii') rejet −76 ; (iii) −65 ;
(i)+(ii) +17 034, +193 047 utiles, **80,6 → 86,4 %** d'utiles fermés (dénominateur variable) ;
oracle-NEW 86,2 %.

Lecture (mesuré) :
- **La borne de L1 est portée par (i) et (ii) seulement.** (iii') rejet et (iii) sont NÉGATIVES : les lier
  ferme moins (pertes saines 758 et 89 paquets) et leurs gains sont factices à 40,5 % et 100 %. Sans
  elles la borne est meilleure : **+28 168 paquets nets, +360 182 utiles**, contre +27 611 / +344 129.
- Les régions interagissent peu : somme des marches séparées +27 355 contre +27 611 pour l'oracle
  complet (+256), +27 923 pour (i) et (ii) séparées contre +28 168 ensemble (+245).
- Par sous-lot : **L1a** (région (i)) borne +16 383 paquets nets, +196 804 utiles ; **L1b** (région
  (ii), dépend de R-L1 (c)) +11 540 / +162 964 ; **L1c** (récupération par recherche d'en-tête, toutes
  régions) n'apporte, au-delà de L1a + L1b, que (iii') et (iii) : **négatif** (−568 paquets nets en
  marches séparées : −522 + 19 − 65 = −568). (iii') « ouverte » est négligeable (+19).
- **Gain propre de L1a seul** : borne d'oracle **+16 383 paquets nets, +196 804 utiles** (+16 370 paquets
  sains). Le localisateur réel mesuré au §2 (`tete-bloc+inv`) fait +7 482 paquets nets bruts, +15 714
  paquets sains nets, +133 511 utiles sains sur le corpus, mais avec des pertes saines sur quatre builds.
  Sur HI_1_13_0 il dépasse la borne d'oracle de (i) en utiles sains (+153 699 contre +104 901) :
  les deux ne mesurent pas le même mécanisme (le localisateur relit les records de la chaîne et lie
  les naissances AVANT le paquet, l'oracle lie après) — voir §7.

Proposition de correction du PLAN §6.1 (non appliquée ici, ce document ne modifie pas le plan) :
remplacer « +27 611 » par la ventilation ci-dessus ; retirer (iii) de la borne de L1 ; marquer (iii')
rejet comme borne négative tant que R-L1 (a) n'est pas tranché.

---

## 5. La variante `oracle-bloc+tete-bloc` (point 11)

Mesuré (`mb_agg_variantes_par_build.tsv`) ; utiles fermés sur dénominateur variable :

| Build | Paquets fermés | Utiles fermés | Gagnés / perdus | Net | Gains factices |
|---|---|---|---|---|---|
| HI_1_13_0 | 237 358 | 83,4 % | +15 595 / −3 055 | +12 540 | 1 236 (7,9 %) |
| HI_1_12_0 | 6 530 | 33,1 % | +695 / 0 | +695 | 13 |
| HI_1_11_0 | 4 227 | 26,4 % | +278 / −336 | −58 | 153 (55 %) |
| HI_1_10_0 | 29 406 | 18,4 % | +3 183 / −2 518 | +665 | 1 597 (50 %) |
| HI_1_9_0 | 5 891 | 26,8 % | +416 / −264 | +152 | 167 (40 %) |
| HI_1_8_0 | 13 195 | 29,8 % | +588 / −1 362 | −774 | 157 (27 %) |
| HI_1_4_1, version-31, version-33 | 711 ; 165 ; 495 | ≈ 0 % | +54 / 0 | +54 | 33 |
| **corpus** | 297 978 | 44,8 % | **+20 809 / −7 535** | **+13 274** | 3 356 (16,1 %) |

Lecture : `oracle-bloc` seul ajoute +2 090 / −3 ; combiné à `tete-bloc`, il ajoute au corpus +1 704
gagnés et retire 216 pertes par rapport à `tete-bloc` seul (+13 274 net contre +11 354). En
fermetures saines, la combinaison fait +9 837 paquets et +51 942 utiles (contre +7 955 / +13 927 pour
`tete-bloc`) ; elle reste en perte saine sur HI_1_8_0 à HI_1_11_0 (HI_1_10_0 −33 440 utiles sains).

---

## 6. Réconciliation de la région (iii), des liaisons et des témoins (points 12, 13)

**Liaisons (mesuré, `mb_rejets.tsv`, eid liés par l'oracle)** : (i) 1 371 + (ii) 1 910 + (iii') après
rejet 1 289 + (iii') après « ouverte » **89** + (iii) 511 = **5 170**. Le RAPPORT §1 omet la ligne
(iii') « ouverte » : 5 081 + 89 = 5 170. Une (iii') « après terminateur » existe aussi (26 eid à
≤ 3 paquets) mais n'est pas une région non lue (une vue B terminée n'a rien après elle) : elle n'est
ni liée ni comptée.

**Témoins à ≤ 3 paquets en région non lue** (mesuré, `mc_m1_region_x_distance.tsv`) : (i) 41 + (ii) 95
+ (iii') rejet 90 + (iii') ouverte **13** + (iii) 0 = **239** (1,4 % de 16 882). Les 63 autres témoins
trouvés à ≤ 3 paquets tombent dans une vue B terminée (7) ou dans un record lu (56) : ce ne sont pas
des régions non lues, ils n'entrent pas dans le taux.

**Région (iii) contre `ti=3 i0`** — ce ne sont pas les mêmes populations (mesuré) :

- (iii) = eid dont la MEILLEURE occurrence retenue par M1 est un NEW lu désynchronisé, à ≤ 3 paquets :
  **511 eid, 13 978 paquets hors cadre, tous sur HI_1_13_0** (`fb1a1a72` 340, `51ebbc0f` 167,
  `c75f33b8` 4). Sur les autres builds, (iii) n'existe qu'à plus de 3 paquets (5 eid) : non liée. Le
  « corpus = HI_1_13_0 » de la critique est donc exact et attendu. La phrase de MESURES §T1-5 « les
  autres builds suivent la même hiérarchie » ne vaut que pour (i) et (ii).
- `ti=3 i0` (MESURES §T7-5) = eid dont un NEW du même eid, plus tôt dans le chunk, s'est désynchronisé
  sur `ti=3 i0 low-frequency`, quelle que soit l'occurrence que M1 retient : **1 134 eid, 27 763
  paquets**, tous sur les trois mêmes films de HI_1_13_0.
- Croisement (mesuré) des 1 134 eid `ti=3 i0` par région M1 retenue : (iii) 536 (dont 511 liés,
  25 à plus de 3 paquets) — 14 532 paquets ; (iii') après rejet 465 (192 liés) — 10 148 ; (iii')
  après « ouverte » 28 (5 liés) — 740 ; (ii) 70 (36 liés) — 1 403 ; (i) 35 (1 lié) — 940. Total
  1 134 eid, 27 763 paquets.
- Réciproque : **les 511 eid de (iii) liés sont tous des `ti=3 i0`** (12 autres (iii) « désynchronisé
  sur un autre composant » existent, tous à plus de 3 paquets, non liés). « Presque tout `ti=3 i0` »
  (RAPPORT l.95) est exactement « tout `ti=3 i0` » pour la partie liée.
- Pourquoi `ti=3 i0` > (iii) : 623 de ces eid ont aussi une occurrence en région (iii'), (ii) ou (i)
  que M1 préfère (traversée propre), ou n'ont leur NEW désynchronisé qu'à plus de 3 paquets.

Ce que la mesure ajoute : lier (iii) est **nuisible** (§3, §4 : 25 gains tous factices, 90 pertes) —
le `R(6)` du NEW désynchronisé de `ti=3` lie un archétype faux. Le chiffre « 13 978 paquets » ne doit
plus figurer dans une borne.

---

## 7. Déclencheur : dénominateur fixe contre dénominateur variable (point 10)

Records utiles fermés, trois dénominateurs (mesuré) : **variable** = records utiles lus par la marche
elle-même ; **fixe réf.** = records utiles lus par la référence ; **fixe max** = par film, le maximum
des records utiles lus sur les 14 marches (le même pour toutes les variantes). Dernière colonne :
records utiles fermés dans des paquets SAINS, sur le dénominateur fixe max.

| Build | Marche | Variable | Fixe réf. | Fixe max | Sains / fixe max |
|---|---|---|---|---|---|
| HI_1_13_0 | reference | 80,6 % | 80,6 % | 78,4 % | 78,1 % |
| | oracle-NEW | 86,2 % | 87,8 % | 85,4 % | 85,1 % |
| | oracle-(i)+(ii) | 86,4 % | 88,3 % | 85,9 % | 85,6 % |
| | tete-bloc | 83,2 % | 84,3 % | 82,0 % | 81,4 % |
| | tete-bloc+inv | 84,7 % | 86,5 % | 84,1 % | 84,1 % |
| HI_1_12_0 | reference | 28,9 % | 28,9 % | 28,1 % | 28,1 % |
| | oracle-NEW | 31,0 % | 31,6 % | 30,7 % | 30,7 % |
| | tete-bloc | 33,1 % | 34,0 % | 33,1 % | 33,1 % |
| HI_1_10_0 | reference | 19,4 % | 19,4 % | 18,7 % | 18,3 % |
| | oracle-NEW | 25,9 % | 23,3 % | 22,4 % | 21,8 % |
| | tete-bloc | 17,7 % | 15,9 % | 15,3 % | 14,7 % |
| HI_1_8_0 | oracle-NEW | 45,7 % | 48,7 % | 45,6 % | 45,4 % |
| corpus | reference | 43,4 % | 43,4 % | 42,0 % | 41,8 % |
| | oracle-NEW | 49,6 % | 49,2 % | 47,5 % | 47,2 % |
| | oracle-(i)+(ii) | 49,6 % | 49,5 % | 47,8 % | 47,5 % |
| | tete-bloc+inv | 45,8 % | 45,4 % | 43,9 % | 43,9 % |

Tous les builds : `mb_agg_variantes_par_build.tsv` (colonnes `pct_var`, `pct_ref`, `pct_max`).

Lecture (mesuré) :
- Le dénominateur variable bouge dans les deux sens : sur HI_1_10_0 l'oracle lit MOINS (1 401 238
  utiles lus contre 1 559 051) et son pourcentage variable (25,9 %) surestime ; sur HI_1_13_0 et
  HI_1_8_0 il lit PLUS et le variable sous-estime (86,2 % contre 87,8 % sur dénominateur fixe réf.).
- Sur HI_1_12_0, `tete-bloc` dépasse l'oracle-NEW sur TOUS les dénominateurs (33,1 % contre 30,7 % en
  fixe max, sans gain factice notable : 13 sur 695). Ce n'est pas un artefact du dénominateur : l'oracle
  n'est pas une borne au sens « gain maximal » ; c'est la borne d'UN mécanisme (lier une naissance
  retrouvée après son paquet), pas de tout correctif de L1.

**Lequel est honnête** : le dénominateur **fixe max** (le même pour toutes les marches d'un film, et au
moins égal à ce que chacune lit), et le numérateur en **paquets sains**. Le variable récompense une
marche qui lit moins ; le fixe réf. peut être dépassé par une marche qui lit plus loin que la référence
(il surestime alors). Ni l'un ni l'autre n'est le nombre de records écrits par le jeu, inconnu sans
lecture complète ; le fixe max en est la meilleure borne basse mesurée. Avec lui, HI_1_13_0 est à
78,1 % (référence) et **85,6 %** sous l'oracle (i)+(ii) : le déclencheur (95 %) reste hors d'atteinte,
la conclusion du RAPPORT §3 tient ; ses pourcentages « sous l'oracle » sont à remplacer par la colonne
fixe max.

Pour les entrées (§1), le dénominateur indépendant est fixe par construction (il ne dépend pas de la
marche) : HI_1_13_0, entrées utiles fermées / D sièges ∪ vus = 45,0 % (référence), 48,5 % (oracle-NEW),
48,8 % (oracle-(i)+(ii)), 47,6 % (`tete-bloc+inv`) ; corpus 24,2 % → 27,0 %.

---

## 8. Limites

- « Sain » ne prouve pas une lecture juste : trois invariants seulement (ordre, masque, vue C).
  Un paquet sain peut être faux ; un factice est faux au moins une fois.
- L'oracle par région lie avec le `R(6)` trouvé par M1 ; la réserve D-7 (`R(6)` contredit le pont 144
  fois sur 266) et les ≈ 4,7 % de liaisons fausses estimées restent valables, région par région.
- Le dénominateur des entrées est une borne haute (règle « au plus une par joueur ») : les parts sont
  des bornes basses. La table du film est celle du DÉBUT du film ; les entités `ti=9` sont celles du
  monde au début du chunk.
- `tete-bloc+inv` est une variante de MESURE (sonde) : elle ne préjuge pas de l'écriture du lot L1a.

## 9. Synthèse par point de la critique

| Point | Réponse mesurée |
|---|---|
| 10 | Dénominateur variable instable (HI_1_10_0 surestime, HI_1_13_0 sous-estime) ; honnête = fixe max + paquets sains : HI_1_13_0 78,1 % → 85,6 % sous (i)+(ii). L'oracle n'est pas une borne de L1 : `tete-bloc` le dépasse sur HI_1_12_0 sur tous les dénominateurs. |
| 11 | `oracle-bloc+tete-bloc` : +20 809 / −7 535, net +13 274 ; HI_1_13_0 +15 595 / −3 055 ; 16,1 % des gains factices ; perte saine sur HI_1_8_0 à HI_1_11_0. |
| 12 | (iii) liée = 511 eid / 13 978 paquets, tous HI_1_13_0, tous `ti=3 i0` ; `ti=3 i0` = 1 134 eid / 27 763 paquets, population plus large (régions (iii'), (ii), (i) retenues pour 598 eid). Lier (iii) est nuisible (−65 net, 100 % de gains factices). Les 623 eid du §6 (1 134 − 511) = ces 598 + les 25 de la région (iii) trouvés à plus de 3 paquets, non liés (corrigé le 2026-10-02). |
| 13 | 5 081 + 89 ((iii') « ouverte ») = 5 170 liaisons ; témoins 41 + 95 + 90 + 13 = 239. |
| 21 | Gains factices : oracle-NEW 461 / 29 769 (1,5 %), (i) 0,5 %, (ii) 1,6 %, (iii') rejet 40,5 %, (iii) 100 %, `tete-bloc` 17,4 % (70 % sur HI_1_10_0), `tete-bloc+inv` 0,3 %. |
| 22 | Dénominateur indépendant livré en borne basse : HI_1_13_0 ≥ 45,0 % (sièges ∪ vus), corpus ≥ 24,2 % ; l'estimateur CARTE §5 est tautologique (= part des paquets fermés, 66,9 %). |
| 23 | `tete-bloc+inv` : corpus +16 407 / −8 925 dont 7 004 factices retirées ; **ne compense pas** les pertes saines de HI_1_8_0 à HI_1_11_0 (utiles sains −1 953, −1 404, −21 094, −1 904) ; gagne +153 699 utiles sains sur HI_1_13_0. |
| 29 | Borne L1a (i) +16 383 nets / +196 804 utiles ; L1b (ii) +11 540 / +162 964 ; (iii') rejet −522 ; (iii') ouverte +19 ; (iii) −65 ; (i)+(ii) +28 168 / +360 182 (> oracle complet +27 611) ; L1c au-delà de L1a+L1b : négatif. |

## 10. Gate de cette étape (sorties relevées le 2026-10-01)

- `go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/` : sans sortie (OK) ;
  `gofmt -l` sur le paquet : vide.
- `go test -tags=research -count=1 -timeout 180m -run '^TestCampagneMesuresBis1$'` sur les 20 films :
  `--- PASS: TestCampagneMesuresBis1 (1259.53s)` (passe 1) et `ok ... 1308.903s` (passe 2).
- Sans variables d'environnement, `TestCampagneMesuresBis1` et `TestCampagneMesuresCiblees` : `SKIP`,
  `ok` (les deux compilent ensemble).
- `TestGrammarRevSuitLaGrammaire`, `TestChroniqueCouvreLaRevisionCourante` (sans tag) : `PASS` —
  `grammar.Rev` et son empreinte inchangés (aucun fichier non-test modifié par cette étape).
