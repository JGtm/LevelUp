# Carte de fermeture v2 — mesure de la campagne de grammaire (2026-10-01)

> **Corrections du 2026-10-02** (critique de complétude n° 2, N13). Le corps de ce document est
> conservé tel qu'écrit le 2026-10-01 ; là où il diverge de ce bandeau, **ce bandeau fait foi**. Les
> corrections du code de la carte relèvent du lot L0 (PLAN §6.2), non fait.
> - **§5, estimateur des entrées de contrôle utiles : tautologique** (prouvé et mesuré,
>   `campagne_grammaire_2026-10-01/MESURES_BIS_1.md` §1). Il redonne, film par film, la part des
>   paquets fermés ; il ne mesure rien des entrées. Les bornes basses indépendantes sont dans BIS_1
>   §1 (HI_1_13_0 ≥ 45,0 %, corpus ≥ 24,2 %). Ses parts par build (21,9 % sur HI_1_10_0, 43,6 % sur
>   le corpus) somment les films ; BIS_1 donne la valeur agrégée (22,2 %, 45,3 %).
> - **§4, classe « réalloué sous une autre génération » : erreur D-43.** 12 854 de ses 12 906
>   paquets (99,6 %) sont des naissances de génération 0 déjà libérées au bloc suivant : l'entrée
>   « génération 0, drapeaux 0 » y passe pour vide (`MESURES_BIS_3_POPULATIONS.md` §3.1). Ce sont
>   des naissances non lues.
> - **§4, classe « naissance non lue » : erreur D-44.** Un NEW lu mais désynchronisé y est rangé
>   comme une naissance non lue : 3 560 paquets sur `81c02726`, film hors corpus
>   (`MESURES_BIS_2.md` §5.1 et §7). La part de cette erreur dans les 213 033 paquets (84,4 %) du corpus
>   n'est pas mesurée ; la région (iii) en montre au moins 13 978 sur HI_1_13_0. Le 84,4 % repris
>   par le RAPPORT §1 porte ces deux erreurs.
> - Voir aussi la section « Corrections » de `MESURES_CIBLEES.md` et le PLAN §5 (D-42 à D-44,
>   D-63, D-65).

> Plan : `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, étape 1 (items 1.1 à 1.7). Worktree
> `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, base `feat/v75` = `69564ef7d`,
> modifications NON commitées (liste au §9). Films lus en place, en lecture seule, depuis
> `LevelUp/data/cache/film_chunks` (manifestes : `LevelUp/data/cache/film_manifests`). Aucune base,
> aucune cuisson, aucun backfill. Sorties brutes : `carte_fermeture_v2_2026-10-01/` (résumé + 10 TSV).
>
> Référence : carte J4.0.5 (`CARTE_FERMETURE_2026-09-26.md`) et sa reprise à la tête J11
> (`MESURES_CLOTURE_J11_2026-10-01.md` §2).

## 1. Protocole

- Instrument : `film/research/cmd_fermeture` (tag `research`), nouveau mode `-mode v2`. La carte v2 passe
  par `grammar.FrameClosureDetaillee` (nouveau, `frame_closure_detail.go`) : même classement que
  `grammar.FrameClosure` (inchangé), plus le détail de chaque paquet. La marche recopie le PILOTAGE de
  `decodeFrameParRangs` (bit de configuration, `consumeVueA`, `decodeInferLoop`, `consumeVueC`,
  `vueCFermee`) et appelle les mêmes lecteurs ; la sortie de vue B se lit aux compteurs
  `RejetsHorsDatum` / `RejetsDeVue` de l'observateur, avant et après la boucle de records.
- Corpus : les 19 témoins de `config/replay_corpus.toml` + `1c4c63c2` (20 films, 9 builds), un film à
  la fois, sentinelle `filmproc` à 4 Gio par film.
- Commande (depuis `apps/go-api`, binaire construit par `go build -tags=research`) :
  `cmd_fermeture -racine C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks -films bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2 -sortie <scratchpad>/carte_v2 -plafond-gib 4 -top 40 -mode v2`
- Résultat : 20 films, 0 échec, 1 min 18 s, pics de 69 Mio (`bcb6d393`) à 321 Mio (`1c4c63c2`). Le
  pic inclut le second chargement du film AVEC son manifeste pour l'item 1.5.
- Deux passes de contrôle, mêmes 20 films : (a) l'outil de la tête NON modifié (`-mode fermeture`,
  table de la tête) ; (b) l'outil v2 avec la table de la tête (`-table` copie de `ecs_table.tsv`
  d'avant l'item 1.4). Elles servent au §2.

## 2. Les colonnes anciennes sont identiques à la référence

| Comparaison | Résultat |
|---|---|
| (a) outil de la tête contre `MESURES_CLOTURE_J11` §2, colonne « tête » (paquets fermés, records utiles fermés / lus, entrées de contrôle fermées) | **identique sur les 20 films** |
| (b) outil v2, table de la tête, contre (a) : `fermeture_archetypes.tsv`, `fermeture_bloquants.tsv` | **identiques à l'octet** |
| (b) contre (a) : `fermeture_films.tsv` | identique, hors `pic_octets` et `duree_ms` (mesures de machine) |
| (b) contre (a) : sections « Par build » et « Causes d arret » du résumé | identiques, hors la colonne de pic mémoire |

Par rapport à la référence J4.0.5 elle-même (18 films), les seuls écarts sont ceux que J11 §2 a déjà
expliqués : +1 à +11 records utiles sur `084a804d`, `0797ce72`, `4f77afc1`, `11de8353`, et deux films
absents de J4.0.5 (`f75e7053`, `1c4c63c2`). « Hors cadre » vaut 264 757 paquets à la tête (214 536 sur
les 18 films de J4.0.5).

**Avec la table corrigée (item 1.4), seuls les comptes UTILES changent** (paquets, vues, archétypes,
bloquants inchangés). Par build, records utiles fermés / lus :

| Build | Table de la tête | Table corrigée |
|---|---|---|
| HI_1_13_0 (10 films) | 1 754 834 / 2 187 577 (80,2 %) | **2 017 679 / 2 504 223 (80,6 %)** |
| HI_1_10_0 (3) | 258 618 / 1 397 862 (18,5 %) | 302 662 / 1 559 051 (19,4 %) |
| HI_1_8_0 | 78 247 / 257 296 (30,4 %) | 83 225 / 268 352 (31,0 %) |
| HI_1_12_0 | 31 993 / 117 732 (27,2 %) | 35 158 / 121 628 (28,9 %) |
| HI_1_11_0 | 65 967 / 246 477 (26,8 %) | 80 032 / 289 378 (27,7 %) |
| HI_1_9_0 | 57 876 / 225 122 (25,7 %) | 65 452 / 248 285 (26,4 %) |
| version-33 | 2 302 / 378 095 (0,6 %) | 3 602 / 469 451 (0,8 %) |
| HI_1_4_1 | 1 / 141 879 (0,0 %) | 80 / 181 607 (0,0 %) |
| version-31 | 10 / 246 018 (0,0 %) | 277 / 319 053 (0,1 %) |
| corpus | 2 249 848 / 5 198 058 | 2 588 167 / 5 961 028 |

Le déclencheur (>= 95 % par build) reste non atteint partout. « Hors cadre » porte 3 142 830 records
utiles en jeu (2 745 875 sous l'ancienne table) ; sur HI_1_13_0, 450 803 des 486 544 records utiles non
fermés, soit **92,7 %** (92,5 % sous l'ancienne table).

## 3. « Vue C : terminateur hors cadre » ventilé (item 1.1)

Sortie de la vue B (comment la boucle de records s'est arrêtée), vue C (vide = son seul terminateur,
sinon nombre d'entrées), reste du payload derrière le terminateur de la vue C.

| Build | Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | Vue C vide | Reste 0-7 (non nuls) | 8-63 | >= 64 | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|---|---|
| HI_1_10_0 | 79 964 | 1,0 % | 99,0 % | 0 | 0 | 99,4 % | 0 | 0,1 % | 99,9 % | 1 235 464 |
| HI_1_11_0 | 10 198 | 2,8 % | 97,2 % | 0 | 0 | 98,2 % | 0 | 0,0 % | 100,0 % | 202 898 |
| HI_1_12_0 | 3 249 | 1,5 % | 98,5 % | 0 | 0 | 98,3 % | 0 | 0 | 100,0 % | 18 529 |
| HI_1_13_0 | 86 921 | 2,4 % | 97,6 % | 0 | 0 | 98,7 % | 1 paquet | 0,2 % | 99,8 % | 450 803 |
| HI_1_4_1 | 8 364 | 23,9 % | 76,1 % | 0 | 0 | 76,4 % | 1 paquet | 0,0 % | 100,0 % | 171 027 |
| HI_1_8_0 | 30 283 | 3,3 % | 96,7 % | 0 | 0 | 98,3 % | 0 | 0,7 % | 99,3 % | 176 241 |
| HI_1_9_0 | 9 792 | 2,3 % | 97,7 % | 0 | 0 | 99,0 % | 0 | 0 | 100,0 % | 177 455 |
| version-31 | 13 780 | 22,5 % | 77,5 % | 0 | 0 | 78,2 % | 0 | 0 | 100,0 % | 276 937 |
| version-33 | 22 206 | 13,3 % | 86,7 % | 0 | 0 | 86,8 % | 0 | 0 | 100,0 % | 433 476 |
| **corpus** | **264 757** | **4,7 %** | **95,3 %** | **0** | **0** | **96,1 %** | 2 paquets | 0,2 % | **99,8 %** | 3 142 830 |

Par film (pourcentages des paquets hors cadre ; « naissance non lue » et « vivant au bloc » : part
des REJETS des paquets hors cadre, cf. §4) :

| Film | Build | Hors cadre | Rejet hors datum | Vue C vide | Reste >= 64 | Naissance non lue | Vivant au bloc | Utiles en jeu |
|---|---|---|---|---|---|---|---|---|
| `084a804d` | HI_1_10_0 | 23 082 | 99,4 % | 99,9 % | 100,0 % | 89,5 % | 4,8 % | 439 828 |
| `111fa685` | HI_1_10_0 | 10 944 | 96,3 % | 96,3 % | 100,0 % | 74,5 % | 3,3 % | 199 075 |
| `1c4c63c2` | HI_1_10_0 | 45 938 | 99,5 % | 99,8 % | 99,9 % | 77,8 % | 20,2 % | 596 561 |
| `e5adf7b2` | HI_1_11_0 | 10 198 | 97,2 % | 98,2 % | 100,0 % | 85,3 % | 0,6 % | 202 898 |
| `bcb6d393` | HI_1_12_0 | 3 249 | 98,5 % | 98,3 % | 100,0 % | 95,8 % | 0,0 % | 18 529 |
| `0797ce72` | HI_1_13_0 | 5 510 | 82,5 % | 87,9 % | 99,3 % | 43,7 % | 5,1 % | 35 108 |
| `396cfc92` | HI_1_13_0 | 8 362 | 99,9 % | 100,0 % | 98,5 % | 92,8 % | 0,0 % | 56 886 |
| `4f77afc1` | HI_1_13_0 | 9 535 | 99,5 % | 99,9 % | 100,0 % | 70,6 % | 19,8 % | 179 588 |
| `51ebbc0f` | HI_1_13_0 | 16 929 | 98,0 % | 98,9 % | 100,0 % | 96,0 % | 0,0 % | 29 399 |
| `bf15f7ab` | HI_1_13_0 | 1 658 | 97,8 % | 92,8 % | 100,0 % | 67,9 % | 0,0 % | 9 831 |
| `bfecd02b` | HI_1_13_0 | 3 154 | 99,7 % | 100,0 % | 100,0 % | 99,9 % | 0,0 % | 23 320 |
| `c75f33b8` | HI_1_13_0 | 2 296 | 99,4 % | 99,9 % | 100,0 % | 99,9 % | 0,0 % | 10 245 |
| `d9781168` | HI_1_13_0 | 11 249 | 98,3 % | 99,9 % | 100,0 % | 98,3 % | 0,0 % | 64 305 |
| `f75e7053` | HI_1_13_0 | 4 287 | 100,0 % | 100,0 % | 100,0 % | 85,0 % | 0,0 % | 26 439 |
| `fb1a1a72` | HI_1_13_0 | 23 941 | 98,0 % | 99,3 % | 100,0 % | 76,2 % | 0,6 % | 15 682 |
| `a521164d` | HI_1_4_1 | 8 364 | 76,1 % | 76,4 % | 100,0 % | 91,3 % | 0,0 % | 171 027 |
| `60ae07c4` | HI_1_8_0 | 30 283 | 96,7 % | 98,3 % | 99,3 % | 88,1 % | 0,7 % | 176 241 |
| `11de8353` | HI_1_9_0 | 9 792 | 97,7 % | 99,0 % | 100,0 % | 80,2 % | 1,4 % | 177 455 |
| `50247b26` | version-31 | 13 780 | 77,5 % | 78,2 % | 100,0 % | 95,0 % | 0,0 % | 276 937 |
| `a349fea8` | version-33 | 22 206 | 86,7 % | 86,8 % | 100,0 % | 87,6 % | 7,6 % | 433 476 |

Croisement corpus (top) : `rejet hors datum · vue C vide · reste >= 64` = **249 931 paquets** (94,4 %
des hors cadre, 2 978 203 utiles en jeu) ; puis `terminateur · 1 entrée · >= 64` 7 543 ; `terminateur ·
vide · >= 64` 3 995. Les autres combinaisons pèsent moins de 1 000 paquets chacune
(`fermeture_hors_cadre.tsv`).

Dernier record lu avant la fin de la vue B, dans les paquets hors cadre sortis par rejet (corpus,
top) : DELTA `ti=35 i25 unit-command-tick` 79 041 paquets ; DELTA `ti=40 i25 unit-command-tick`
55 996 ; DELTA `ti=4 i0 high-frequency` 43 715 ; DELTA `ti=37 i3 object-angular-velocity` 17 019 ;
DELTA `ti=10 i26 managed-object-rtpc` 10 025 ; DELTA `ti=42 i3 object-angular-velocity` 7 171 ; aucun
record (le premier en-tête de la vue B est rejeté) 1 410. Table complète :
`fermeture_hors_cadre_dernier.tsv` (1 973 lignes).

Lecture, sans interprétation au-delà des chiffres : la sortie « rejet de vue » ne se produit JAMAIS
(0 sur 259 692 rejets) ; « autre » vaut 0 (la combinaison de compteurs est cohérente sur tout le
corpus) ; quand la vue B sort par rejet, le curseur est remis à la fin de l'en-tête rejeté
(`frame_infer.go`, `decodeInferLoop`) et la vue C lue à partir de là est presque toujours « vide »
(son premier bit vaut 0), avec 64 bits ou plus de payload derrière elle.

Sorties de la vue B sur TOUS les paquets qui l'atteignent (corpus) : terminateur 296 457 (94,5 %
fermés, 4,2 % hors cadre) ; rejet hors datum 259 692 (**1,8 % fermés**, 97,1 % hors cadre) ; ouverte
24 273 (désynchronisation, jamais fermés). Sur HI_1_13_0 : terminateur 226 930 (98,8 % fermés), rejet
hors datum 86 562 (0,7 % fermés).

## 4. Les rejets contre le bloc de type 1 (item 1.2)

Pour chaque paquet dont la vue B sort sur un rejet, l'eid COMPLET de l'en-tête rejeté est relu
(`IDLowBits + 2` bits avant la fin de la vue B) et confronté au bloc de type 1 (`LireBlocDeDatums`) de
SON chunk (état du slot), puis à celui du chunk SUIVANT (naissance). Classes d'état : celles de
`TestBloc521Rejets` (lot 5.21). « Naissance non lue » : le bloc suivant porte une allocation de ce
slot sous la génération de l'eid (vivante ou déjà libérée) que le bloc du chunk ne portait pas, et
aucun NEW de ce slot n'a été lu dans le chunk.

Tous les blocs de type 1 consultés se lisent (0 illisible sur les 20 films, tous builds) ; 14 chunks
consultés (le chunk d un rejet, et son suivant) n en portent aucun, non identifiés un à un ; avec le
dernier chunk (sans suivant), ils forment la classe « non mesurable ».

Rejets des paquets HORS CADRE (252 262), corpus :

| État au bloc du chunk | Paquets | Part |
|---|---|---|
| vide | 127 533 | 50,6 % |
| trace (génération ou drapeau posés) | 109 919 | 43,6 % |
| vivant | 14 809 | 5,9 % |
| absent (slot au-delà de la table) | 1 | 0,0 % |

| Naissance | Paquets | Part | Utiles en jeu |
|---|---|---|---|
| **naissance non lue** | **213 033** | **84,4 %** | **2 742 690** |
| vivant au bloc du chunk | 14 723 | 5,8 % | 36 054 |
| réalloué sous une autre génération | 12 906 | 5,1 % | 150 131 |
| aucune allocation | 6 451 | 2,6 % | 40 542 |
| non mesurable | 4 920 | 2,0 % | 19 290 |
| libéré avant le chunk (même génération) | 173 | 0,1 % | 1 303 |
| NEW lu dans le chunk | 56 | 0,0 % | 84 |

Par build (rejets des paquets hors cadre) :

| Build | Rejets hors cadre | Naissance non lue | Utiles en jeu de ces paquets | Part des utiles en jeu des rejets hors cadre |
|---|---|---|---|---|
| HI_1_13_0 | 84 825 | 71 292 (84,0 %) | 389 845 | 88,0 % |
| HI_1_10_0 | 79 180 | 63 926 (80,7 %) | 1 136 685 | 92,3 % |
| HI_1_8_0 | 29 289 | 25 813 (88,1 %) | 151 359 | 88,5 % |
| version-33 | 19 245 | 16 860 (87,6 %) | 361 517 | 95,1 % |
| version-31 | 10 684 | 10 145 (95,0 %) | 231 119 | 97,9 % |
| HI_1_11_0 | 9 910 | 8 452 (85,3 %) | 177 371 | 87,9 % |
| HI_1_9_0 | 9 567 | 7 674 (80,2 %) | 147 100 | 83,8 % |
| HI_1_4_1 | 6 362 | 5 807 (91,3 %) | 129 172 | 97,4 % |
| HI_1_12_0 | 3 200 | 3 064 (95,8 %) | 18 522 | 100,0 % |
| **corpus** | **252 262** | **213 033 (84,4 %)** | **2 742 690** | **91,7 %** |

Rapporté aux records utiles non fermés de HI_1_13_0 (486 544) : les paquets « rejet hors datum +
naissance non lue » en portent 389 845, soit **80,1 %**.

Ce que cela dit de l'hypothèse de travail (« une naissance d'entité non lue fait sortir la vue B par
rejet, la vue C est lue au mauvais endroit ») : sur le corpus, 95,3 % des paquets hors cadre sortent de
la vue B par un rejet hors datum, et pour 84,4 % de ces rejets le bloc de type 1 du chunk suivant
montre une allocation de CET eid née pendant le chunk, sans NEW lu. La mesure est COMPATIBLE avec
l'hypothèse et la rend majoritaire ; elle ne la prouve pas : elle ne dit pas comment l'écrivain émet
la naissance (piste T1 de l'étape 2, Ghidra), ni que la lecture reprendrait juste si le record de
naissance était lu. Précédent cohérent : sur `bfecd02b`, 0 slot rejeté vivant au bloc du chunk (lot
5.21 : 0 sur 23 325) ; la v2 y trouve 3 143 naissances non lues sur 3 154 rejets.

Deux populations à part :
- « vivant au bloc du chunk » (14 723 rejets, dont 10 700 sur HI_1_10_0 et 1 457 sur version-33 ;
  20,2 % des rejets hors cadre de `1c4c63c2`, 19,8 % de `4f77afc1`) : l'eid était VIVANT sous cette
  génération au début du chunk, et pourtant non lié au monde hors ligne ; ce n'est pas une naissance
  non lue (découverte consignée, non traitée).
- « aucune allocation » sur `0797ce72` (1 809 des 4 544 rejets hors cadre) : ni le bloc du chunk ni
  le suivant ne montrent cet eid. Limite écrite : une entité de génération 0 née et morte entre deux
  blocs y tombe aussi (entrée `drapeaux 0, génération 0`, indiscernable d'un slot jamais alloué).

Rejets dans des paquets FERMÉS : 4 598 (1,8 % des rejets), presque tous « vide / aucune allocation »
(3 045) ou « trace / aucune allocation » (1 261) : des fins de vue B par en-tête rejeté qui ferment
le paquet au bit près. Matière pour la piste T3 (une sortie par rejet est-elle jamais légitime).

## 5. Dénominateur des entrées de contrôle utiles (item 1.3)

Définition, par le canal de production : la seule lecture de production de la vue C est le tir continu,
`grammar/movement_states.go:190` (`obs.VueControleHook = sc.tir.recevoir`), publié par
`replay/film_scan_mouvement.go:53`. Le collecteur (`grammar/tir_continu.go:121-130`) compte toutes les
entrées d'une vue C fermée, mais n'en LIT que celles qui portent le bloc de 0x68 octets
(`if !e.Bloc { continue }` puis `c.entree(e)`) : une entrée de contrôle est UTILE quand `Bloc` est vrai.

| Build | Paquets | Vues C fermées | Entrées fermées | Utiles fermées | Moyenne par vue C fermée | Dénominateur estimé | Part estimée | Utiles lues hors fermeture (non prouvées) |
|---|---|---|---|---|---|---|---|---|
| HI_1_13_0 | 335 960 | 224 818 (66,9 %) | 1 619 514 | 1 619 501 | 7,20 | 2 420 887 | 66,9 % | 207 |
| HI_1_10_0 | 129 369 | 28 741 (22,2 %) | 420 684 | 420 612 | 14,63 | 1 924 866 | 21,9 % | 68 |
| HI_1_8_0 | 49 696 | 13 969 (28,1 %) | 79 294 | 79 279 | 5,68 | 282 042 | 28,1 % | 104 |
| HI_1_12_0 | 21 864 | 5 835 (26,7 %) | 30 137 | 30 136 | 5,16 | 112 921 | 26,7 % | 10 |
| HI_1_11_0 | 16 824 | 4 285 (25,5 %) | 76 224 | 76 222 | 17,79 | 299 267 | 25,5 % | 51 |
| HI_1_9_0 | 17 629 | 5 739 (32,6 %) | 75 327 | 75 327 | 13,13 | 231 389 | 32,6 % | 10 |
| version-33 | 28 751 | 463 (1,6 %) | 6 | 3 | 0,01 | 186 | 1,6 % | 3 621 |
| HI_1_4_1 | 11 130 | 701 (6,3 %) | 5 | 0 | 0 | 0 | - | 2 015 |
| version-31 | 17 919 | 153 (0,9 %) | 3 | 2 | 0,01 | 234 | 0,9 % | 4 180 |
| corpus | 629 142 | 284 704 (45,3 %) | 2 301 194 | 2 301 082 | 8,08 | 5 271 792 | 43,6 % | 10 266 |

Le dénominateur ESTIMÉ ajoute, par film, la moyenne d'entrées utiles d'une vue C fermée multipliée par
les paquets non fermés (listes non localisées comprises). **Par construction, la part estimée d'un film
égale sa part de vues C fermées** (U / (U + U/F × N) = F / (F + N)) : avec cet estimateur, la moitié
« entrées » du déclencheur se confond avec la part des paquets fermés (66,9 % sur HI_1_13_0, non
atteinte). Un dénominateur indépendant demanderait le nombre d'entrées qu'écrit le jeu par paquet
(lecture de l'écrivain, hors de cette étape). 99,995 % des entrées fermées portent le bloc (2 301 082
sur 2 301 194).

## 6. Compte déclaré du chunk des temps forts (item 1.5)

Le chunk de type 3 est choisi par son type au manifeste (`finalise.EstTempsForts`, film rechargé par
`filmcache.LoadFilmDir`) ; son paquet de type 9 commence par un u32 big-endian, le nombre d'événements
déclarés ; les événements trouvés sont ceux de `grammar.ParseHighlightEvents` sur le même chunk.

**20 films sur 20 : déclarés = trouvés, écart 0** (de 136 à 1 305 événements ; un seul paquet de
type 9 par film). Le fil des morts (`grammar.ScanDeaths`) rend exactement les événements `death` sur les
20 films. Par film : `fermeture_chunk3.tsv` ; `0797ce72` 220/220 et `4f77afc1` 844/844 (les deux films
mesurés par le port Rust, mêmes valeurs).

## 7. Mode borné (item 1.6)

Lectures de la vue B qui finissent APRÈS le dernier bit du payload (le lecteur rend des zéros au-delà) :

| Build | Records lus | Records débordants | Composants débordants | NEW propres débordants | Paquets | Terminateurs de vue B lus au-delà |
|---|---|---|---|---|---|---|
| HI_1_13_0 | 3 355 181 | 270 | 1 630 | 68 | 270 | 11 |
| HI_1_8_0 | 338 252 | 337 | 1 312 | 79 | 337 | 12 |
| HI_1_10_0 | 1 799 156 | 15 | 62 | 8 | 15 | 3 |
| autres builds | 2 090 370 | 36 | 177 | 14 | 36 | 6 |
| corpus | 7 582 959 | 658 (0,009 %) | 3 181 | 169 | 658 | 32 |

Un mode qui refuserait de lire au-delà du payload toucherait 658 records et 169 liaisons NEW sur le
corpus : un effet négligeable sur la fermeture. « NEW propres débordants » = traversés sans
désynchronisation sur un corps qui déborde, donc liés au monde (sauf refus contre une entité vivante).

## 8. Colonne `product_use` corrigée (item 1.4)

36 -> 73 lignes à usage produit dans `grammar/testdata/ecs_table.tsv` ; chaque ligne modifiée cite son
canal de production (`fichier:ligne`). Seule la colonne 12 change (contrôle : colonnes 1-11 et 13-16
identiques à la tête).

| Lignes | Avant | Après (canal de production) |
|---|---|---|
| `ti=35 i1` | aucun (« oracle interne ») | `movementStates` jumpDerived — `grammar/movement_states.go:350`, publié `replay/film_scan_mouvement.go:52` |
| `ti=35 i29`, `i62`, `i54` | aucun | `movementStates` accroupi / glissade / escalade — `movement_states.go:325`, `:331`, `:336` |
| `ti=35 i57` | aucun | `movementStates` sprint (`movement_states.go:341`) et impulsions de capacité (`grammar/ability_impulses.go:100`, `replay/film_scan.go:300`) |
| `ti=35 i10` | aucun | occupation des véhicules — `grammar/object_deaths.go:233` (ScanMarchFacts), `replay/build_vehicles.go:155`, `replay/vehicle_tracks.go:133` |
| `ti=9 i0` | aucun (« l'équipe vient de la base ») | équipe des joueurs — `grammar/player_teams.go:142`, `replay/film_scan.go:429` |
| `ti=40 i0`, `i1`, `i2` | aucun | trajectoires et cap des véhicules — `replay/build_vehicles.go:120`, `:257`, `:258`, `replay/vehicle_heading.go:57`, `:70` |
| `ti=40 i11` | aucun | morts écrites des véhicules — `grammar/object_deaths.go:242`, `replay/build_vehicles.go:163` |
| `ti=13 i1`, `i2..i33` | aucun | état des zones et jauge de retour du drapeau — `grammar/zone_state_scan.go:231`, `replay/build_zones.go:59`, `:81`, `:92` |
| `ti=12 i14` | aucun | armement de la bombe — `grammar/navpoint_radial_scan.go:203-204`, `replay/bomb_armings.go:162`, `:209` |
| `ti=42 i0` | aucun (« refusé à la publication ») | datation des ramassages aux socles — `replay/build_ground_weapons.go:112`, `replay/ground_weapon_objects.go:173`, `:330` |
| `ti=42 i20` | aucun | munitions à la naissance — `grammar/equipment_creation.go:383`, `replay/ground_weapon_objects.go:186`, `replay/document_ground_weapon_items.go:225` |
| `ti=11 i0`, `i3`, `i5`, `i12`, `i13`, `i14` | « publie par hook » | **aucun** — `ScanObjectives` (`grammar/objective_scan.go:179`) n'a aucun appelant de production (`grammar/keyframe_scan_counters.go:58-62`) |
| `ti=37 i20`, `i21`, `i23`, `i24` | « publie par hook » | **aucun** — `ScanEquipmentState` (`grammar/equipment_state.go:242`) n'a aucun appelant de production |
| `ti=13 i0`, `ti=35 i18`, `ti=35 i55` | aucun | aucun, avec la pièce (`zone_state_scan.go:29-35` ; `movement_states.go:355`) |

Restent `aucun`, vérifiés sans appelant de production : `ti=35 i26` (`ScanUnitEquipment`), `ti=35 i42`
et `i51` (hooks jamais posés hors des tests), `ti=12 i11` / `i12` (publiés au hook, filtrés par
`navpoint_radial_scan.go:204`).

Consommateurs mis à jour : `cmd_fermeture` lit la colonne telle quelle (`table.go`) ; le golden
`grammar/testdata/frame_closure.golden` est régénéré par sa porte (`-update-frame-closure`) avec une
ligne d'historique : seuls les comptes utiles bougent (`ks_000d5950` 5 889/30 428 -> 6 608/31 200,
`ks_e5adf7b2` 0/1 -> 0/23), aucun compte de paquet ni de record, aucune ligne `fermes` en baisse.

## 9. Code et garanties

Fichiers : `grammar/frame_closure_detail.go`, `frame_closure_detail_records.go`,
`frame_closure_detail_test.go` (neufs) ; `grammar/frame_closure_ratchet_test.go` (ligne d'historique),
`testdata/frame_closure.golden`, `testdata/ecs_table.tsv`, `testdata/grammar_rev.golden` ;
`research/cmd_fermeture/` : `v2.go`, `v2_datums.go`, `v2_chunk3.go`, `v2_rapport.go`, `v2_resume.go`,
`v2_research_test.go` (neufs), `main.go`, `rapport.go`, `resume.go`, `gb1_research_test.go`.

- `grammar.FrameClosure` et tous les fichiers lus par une cuisson sont inchangés ; le seul code de
  production ajouté est un instrument (`frame_closure_detail*.go`) qu'aucun chemin de cuisson n'appelle.
- `grammar.Rev` inchangé (`grammar-2026-09-27.3`) ; empreinte régénérée à révision constante
  (`4bc8e05a…` -> `3cbd64ac…`, 207 fichiers) par `LEVELUP_UPDATE_GRAMMAR_REV=1 … -update-grammar-rev`.
- Garde-fou de la recopie du pilotage : `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` (carte
  détaillée = carte de `FrameClosure`, champ à champ, sur `ks_000d5950` et `ks_e5adf7b2`, et le détail
  se recoupe : un détail par paquet, mêmes paquets fermés, mêmes hors cadre et utiles en jeu, aucune
  sortie « autre ») ; mutation jouée (eid relu un bit trop tôt) : rouge.

## 10. Limites

- `utiles` ne compte que les records LUS : corriger la sortie de vue B fera lire des records
  aujourd'hui invisibles (la borne de MESURES_CLOTURE_J11 §2 tient toujours).
- « Naissance non lue » s'appuie sur deux blocs de type 1 distants d'un chunk : une allocation née ET
  libérée puis réallouée sous une autre génération dans le même chunk est classée « réalloué » ; une
  entité de génération 0 née et morte entre deux blocs est invisible.
- Le dénominateur des entrées est une estimation, qui se confond avec la part des paquets (§5).
- Aucune lecture de Ghidra n'appuie cette étape : ce sont des mesures. Les constats de l'étape 2
  (T1, T3, T8) les interpréteront.
