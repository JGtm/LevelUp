# Mesures T.0 sur le parc des rejeux regeneres

Date : 2026-11-17. Parc : 1 455 documents de rejeu au schema 125 (`data/cache/replays/halo_infinite/`).

Methode : script Node (`fs` et `zlib` seuls), un document decompresse a la fois, compteurs seuls conserves, document libere avant le suivant ; lecture seule du parc, aucune base ouverte, aucune commande `levelup`. Le build est lu dans `coverage.decoder.build` (5 documents n'en portent pas : « (inconnu) », sans identite de joueur non plus). Les scripts et journaux sont hors du depot (`C:/t/t0-mesures/`).
Confidentialite : un match se designe par les 8 premiers caracteres de son nom de fichier, un joueur par sa position dans `shotStats` du document (« joueur N »).

Repartition du parc par build : HI_1_13_0 = 1 200 (82 %), HI_1_12_0 = 145, HI_1_11_0 = 57, HI_1_10_0 = 31, HI_1_8_0 = 13, HI_1_9_0 = 3, HI_1_4_1 = 1, sans build = 5. Les builds anterieurs a 1.12 pesent 108 documents : leurs pourcentages sont a lire avec cet effectif.

## Cles introuvables (a lire en premier)

- **Mode de jeu** : aucun document ne porte le mode. Seul proxy : `coverage.scoreActions.group` (le script de score, par exemple `slayerevents.lua`, `arenaflagevents.lua+flageventstubs.lua`, `bigteambattle.lua+btbslayerevents.lua`). Les 71 documents sans groupe (`(aucun)`) ne peuvent pas etre rattaches a un mode ; l'Assaut n'est donc pas isolable par nom (ces documents sont tous au verdict « incomplet », ils ne touchent pas le garde de T.3).
- **Totaux de fin de match par joueur (la feuille)** : introuvables dans le contrat (`ReplayDocument`) et dans tous les documents lus. `scoreTimeline.players[].stats` ne porte que des series datees (headshots, maxSpree, etc.), pas une feuille. La comparaison de la mesure 5 (calque de score a la derniere image contre la feuille) est donc IMPOSSIBLE sans la base ; un substitut interne, clairement nomme, est donne a la place.
- `grenadeLabels[].family` : absente dans tous les documents (voir mesure 4).
- Aucune autre cle attendue ne manque : `coverage.scoreActions.complete`, `shotStats[].sufficient`, `coverage.pickups.multiEvent`, `scoreTimeline.players`, `coverage.stances.{scanned,absent,jumpEpisodes}`, `medals[].name` existent.


## Mesure 1. Score par action declare complet

Cle : `coverage.scoreActions.complete`. Numerateur : documents au verdict vrai ; denominateur : tous les documents du build. Verification : pour chaque joueur de `scoreTimeline.players`, somme des `scoreActions[].points` du joueur contre la derniere valeur de `scoreTimeline.players[].score.total` (le score personnel publie dans le document). Une serie vide vaut zero (verifie : sur 79 joueurs a serie de score vide dans des documents complets, 76 ont une somme nulle ; 3 non). Un document est « egal » quand tous ses joueurs comparables le sont.

### M1a. Verdict de score complet, par build

| Build | Docs | Verdict complet | Part | Incomplet |
|---|---|---|---|---|
| (inconnu) | 5 | 0 | 0.0 % | 5 |
| HI_1_10_0 | 31 | 0 | 0.0 % | 31 |
| HI_1_11_0 | 57 | 12 | 21.1 % | 45 |
| HI_1_12_0 | 145 | 97 | 66.9 % | 48 |
| HI_1_13_0 | 1200 | 835 | 69.6 % | 365 |
| HI_1_4_1 | 1 | 0 | 0.0 % | 1 |
| HI_1_8_0 | 13 | 0 | 0.0 % | 13 |
| HI_1_9_0 | 3 | 0 | 0.0 % | 3 |
| TOTAL | 1455 | 944 | 64.9 % | 511 |

### M1b. Parmi les complets : somme des points par action = score personnel

| Build | Docs complets comparables | Docs tous joueurs egaux | Part docs | Joueurs compares | Joueurs egaux | Part joueurs |
|---|---|---|---|---|---|---|
| (inconnu) | 0 | 0 | n/a | 0 | 0 | n/a |
| HI_1_10_0 | 0 | 0 | n/a | 0 | 0 | n/a |
| HI_1_11_0 | 12 | 0 | 0.0 % | 313 | 208 | 66.5 % |
| HI_1_12_0 | 97 | 95 | 97.9 % | 868 | 858 | 98.8 % |
| HI_1_13_0 | 834 | 804 | 96.4 % | 6915 | 6864 | 99.3 % |
| HI_1_4_1 | 0 | 0 | n/a | 0 | 0 | n/a |
| HI_1_8_0 | 0 | 0 | n/a | 0 | 0 | n/a |
| HI_1_9_0 | 0 | 0 | n/a | 0 | 0 | n/a |
| TOTAL | 943 | 899 | 95.3 % | 8096 | 7930 | 97.9 % |

Lecture : le verdict complet ne sort que des builds 1.11 et suivants (aucun complet avant). En 1.12 et 1.13 la somme egale le score pour 98,8 % et 99,3 % des joueurs (96-98 % des documents). Le verdict est FAUX au build 1.11 : sur 12 documents complets, aucun n'est egal en entier, 33 % des joueurs ont un ecart (tous en Grande bataille : script `bigteambattle.lua+btbslayerevents.lua` ; somme superieure au score pour 74 joueurs, inferieure pour 29, hors series vides). Il n'est pas faux qu'en Grande bataille 1.11 : 51 joueurs sur 6 915 ont aussi un ecart en 1.13 (40 joueurs dans 23 documents de Melee avec une somme inferieure au score de 280 points en moyenne, 9 joueurs en Drapeau avec des ecarts de sens varie, 1 en Zones, 1 de plus par une serie de score vide ; en 1.12, 9 joueurs d'un document de Melee et 1 en Bases), donc le garde « somme = score de l'API » du plan reste necessaire sur tous les builds. L'Assaut n'est pas isolable (voir plus haut).

### M1c. Parmi les complets, par script de score (tous builds)

| Script de score | Complets | Docs egaux | Part docs | Joueurs | Egaux | Part joueurs |
|---|---|---|---|---|---|---|
| (aucun) | 0 | 0 | n/a | 0 | 0 | n/a |
| areacaptureeventstubs.lua+bastionevents.lua | 33 | 32 | 97.0 % | 266 | 265 | 99.6 % |
| areacaptureeventstubs.lua+sevents.lua | 57 | 56 | 98.2 % | 475 | 474 | 99.8 % |
| arenaflagevents.lua+flageventstubs.lua | 111 | 106 | 95.5 % | 925 | 916 | 99.0 % |
| bigteambattle.lua+btbslayerevents.lua | 16 | 4 | 25.0 % | 417 | 312 | 74.8 % |
| oddballevents.lua | 4 | 4 | 100.0 % | 32 | 32 | 100.0 % |
| slayerevents.lua | 722 | 697 | 96.5 % | 5981 | 5931 | 99.2 % |

### M1d. Complets, par build x script

| Build | Script | Complets | Docs egaux | Joueurs | Egaux | Part joueurs |
|---|---|---|---|---|---|---|
| HI_1_11_0 | bigteambattle.lua+btbslayerevents.lua | 12 | 0 | 313 | 208 | 66.5 % |
| HI_1_12_0 | areacaptureeventstubs.lua+bastionevents.lua | 12 | 11 | 98 | 97 | 99.0 % |
| HI_1_12_0 | areacaptureeventstubs.lua+sevents.lua | 8 | 8 | 66 | 66 | 100.0 % |
| HI_1_12_0 | arenaflagevents.lua+flageventstubs.lua | 15 | 15 | 123 | 123 | 100.0 % |
| HI_1_12_0 | bigteambattle.lua+btbslayerevents.lua | 4 | 4 | 104 | 104 | 100.0 % |
| HI_1_12_0 | oddballevents.lua | 1 | 1 | 8 | 8 | 100.0 % |
| HI_1_12_0 | slayerevents.lua | 57 | 56 | 469 | 460 | 98.1 % |
| HI_1_13_0 | areacaptureeventstubs.lua+bastionevents.lua | 21 | 21 | 168 | 168 | 100.0 % |
| HI_1_13_0 | areacaptureeventstubs.lua+sevents.lua | 49 | 48 | 409 | 408 | 99.8 % |
| HI_1_13_0 | arenaflagevents.lua+flageventstubs.lua | 96 | 91 | 802 | 793 | 98.9 % |
| HI_1_13_0 | oddballevents.lua | 3 | 3 | 24 | 24 | 100.0 % |
| HI_1_13_0 | slayerevents.lua | 665 | 641 | 5512 | 5471 | 99.3 % |

### M1e. Verdict incomplet : la somme egale-t-elle quand meme le score ? (faux negatifs)

| Build | Docs incomplets comparables | Docs tous egaux | Joueurs | Egaux | Part joueurs |
|---|---|---|---|---|---|
| (inconnu) | 1 | 1 | 1 | 1 | 100.0 % |
| HI_1_10_0 | 31 | 0 | 759 | 48 | 6.3 % |
| HI_1_11_0 | 45 | 0 | 1159 | 229 | 19.8 % |
| HI_1_12_0 | 48 | 36 | 552 | 356 | 64.5 % |
| HI_1_13_0 | 365 | 327 | 3173 | 2970 | 93.6 % |
| HI_1_4_1 | 0 | 0 | 0 | 0 | n/a |
| HI_1_8_0 | 13 | 0 | 227 | 10 | 4.4 % |
| HI_1_9_0 | 3 | 0 | 67 | 7 | 10.4 % |
| TOTAL | 506 | 364 | 5938 | 3621 | 61.0 % |

Lecture de M1e : un verdict « incomplet » est conservateur en 1.13 (93,6 % des joueurs y ont quand meme somme = score) ; avant 1.12 il est justifie (6 a 20 % d'egalite).

## Mesure 2. Lecture des tirs jugee suffisante

Cle : `shotStats[].sufficient`. Denominateur des joueurs : les entrees de `shotStats` (14 053 ; il manque 265 humains du roster, 1,8 %, qui n'ont pas d'entree et ne sont pas lisibles : sur les 14 318 humains, la part suffisante est de 45,2 %). Documents « tous suffisants » : toutes les entrees de `shotStats` vraies. Aucun document n'a de desaccord entre `coverage.hits.sufficient` et le decompte de `shotStats`.

### M2a. Lecture des tirs suffisante, par build

| Build | Docs avec shotStats | Joueurs | Suffisants | Part joueurs | Docs tous suffisants | Part docs | Docs partiels | Mediane coverage |
|---|---|---|---|---|---|---|---|---|
| (inconnu) | 0 | 0 | 0 | n/a | 0 | n/a | 0 | n/a |
| HI_1_10_0 | 31 | 760 | 0 | 0.0 % | 0 | 0.0 % | 0 | 0.49 |
| HI_1_11_0 | 57 | 1473 | 43 | 2.9 % | 0 | 0.0 % | 21 | 0.44 |
| HI_1_12_0 | 145 | 1424 | 837 | 58.8 % | 30 | 20.7 % | 107 | 0.82 |
| HI_1_13_0 | 1200 | 10100 | 5588 | 55.3 % | 154 | 12.8 % | 991 | 0.82 |
| HI_1_4_1 | 0 | 0 | 0 | n/a | 0 | n/a | 0 | n/a |
| HI_1_8_0 | 13 | 229 | 0 | 0.0 % | 0 | 0.0 % | 0 | 0.62 |
| HI_1_9_0 | 3 | 67 | 1 | 1.5 % | 0 | 0.0 % | 1 | 0.46 |
| TOTAL | 1449 | 14053 | 6469 | 46.0 % | 184 | 12.7 % | 1120 | 0.80 |

Lecture : suffisance quasi nulle avant 1.12 (0 a 3 %), autour de 56 % des joueurs en 1.12 et 1.13 (55,8 % sur les deux), mais seulement 13,7 % des documents ont TOUS leurs joueurs suffisants (184 sur 1 345). Parmi les documents qui ont au moins un joueur suffisant, la part mediane de joueurs suffisants est de 2 sur 3. Le Grande bataille est quasi toujours insuffisant (2 %).

### M2b. Par script de score (tous builds)

| Script | Docs | Joueurs | Part suffisants | Docs tous suffisants | Part docs |
|---|---|---|---|---|---|
| (aucun) | 71 | 1765 | 3.3 % | 0 | 0.0 % |
| areacaptureeventstubs.lua+bastionevents.lua | 48 | 390 | 75.6 % | 7 | 14.6 % |
| areacaptureeventstubs.lua+sevents.lua | 77 | 640 | 77.2 % | 17 | 22.1 % |
| arenaflagevents.lua+flageventstubs.lua | 179 | 1552 | 69.1 % | 28 | 15.6 % |
| bigteambattle.lua+btbslayerevents.lua | 44 | 1121 | 2.0 % | 0 | 0.0 % |
| oddballevents.lua | 7 | 56 | 78.6 % | 4 | 57.1 % |
| slayerevents.lua | 1023 | 8529 | 52.6 % | 128 | 12.5 % |

### M2c. Builds 1.12.0 et 1.13.0 par script

| Build | Script | Docs | Part joueurs suffisants | Docs tous suffisants |
|---|---|---|---|---|
| HI_1_12_0 | (aucun) | 8 | 4.5 % | 0 (0.0 %) |
| HI_1_12_0 | areacaptureeventstubs.lua+bastionevents.lua | 18 | 75.7 % | 2 (11.1 %) |
| HI_1_12_0 | areacaptureeventstubs.lua+sevents.lua | 10 | 72.3 % | 0 (0.0 %) |
| HI_1_12_0 | arenaflagevents.lua+flageventstubs.lua | 20 | 47.3 % | 2 (10.0 %) |
| HI_1_12_0 | bigteambattle.lua+btbslayerevents.lua | 4 | 1.9 % | 0 (0.0 %) |
| HI_1_12_0 | oddballevents.lua | 2 | 100.0 % | 2 (100.0 %) |
| HI_1_12_0 | slayerevents.lua | 83 | 79.7 % | 24 (28.9 %) |
| HI_1_13_0 | (aucun) | 8 | 14.9 % | 0 (0.0 %) |
| HI_1_13_0 | areacaptureeventstubs.lua+bastionevents.lua | 29 | 78.2 % | 5 (17.2 %) |
| HI_1_13_0 | areacaptureeventstubs.lua+sevents.lua | 66 | 79.1 % | 17 (25.8 %) |
| HI_1_13_0 | arenaflagevents.lua+flageventstubs.lua | 157 | 74.9 % | 26 (16.6 %) |
| HI_1_13_0 | bigteambattle.lua+btbslayerevents.lua | 1 | 8.3 % | 0 (0.0 %) |
| HI_1_13_0 | oddballevents.lua | 4 | 87.5 % | 2 (50.0 %) |
| HI_1_13_0 | slayerevents.lua | 935 | 50.5 % | 104 (11.1 %) |


Raisons d'insuffisance (seconde passe, `coverage` < `coverage.hits.minCoverage` ou `duplicates` > `maxDuplicates`), build 1.13 : 4 512 joueurs insuffisants, dont 48 % par couverture basse seule, 27 % par doublons seuls, 15 % par les deux, 10 % autre (images arretees ou signes vitaux non lus) ; 90 joueurs n'ont aucun tir.

## Mesure 3. Boucliers perdus contre degats subis

Definition (D-21) : par vie (`tracks[]` du joueur), passage de `points[].sh` de > 0 a <= 0 ; une valeur non publiee reprend la precedente ; une recharge puis une nouvelle chute comptent deux. Degats subis : `shotStats[].damageTaken` (tout le film). Morts : derniere valeur de `scoreTimeline.players[].deaths.total`. Perimetre : les 184 documents ou tous les joueurs sont suffisants (1 471 joueurs ; 154 en 1.13, 30 en 1.12), pour lesquels `sh` est publie sur 92 a 100 % des vies (mediane par joueur : 1.12 = 0,92 ; 1.13 = 1,00). Correlation de rang de Spearman, calculee par match puis mediane.

| Perimetre | Matchs | rho(boucliers, degats) mediane (Q1-Q3) | rho(boucliers, morts) | rho(degats, morts) | rho poolee boucliers/degats | Boucliers par mort (mediane) |
|---|---|---|---|---|---|---|
| Tous les tous-suffisants | 184 | 0,64 (0,44-0,81) | 0,49 | 0,71 | 0,83 | 1,00 (Q10 0,57 - Q90 1,42) |
| HI_1_12_0 | 30 | 0,71 (0,59-0,89) | 0,61 | 0,61 | 0,90 | 0,92 |
| HI_1_13_0 | 154 | 0,63 (0,41-0,81) | 0,48 | 0,72 | 0,82 | 1,00 |

Degats subis par passage a zero : mediane 325. Trois matchs de modes differents (build 1.13, 8 joueurs, choisis au milieu de la liste triee par nom de fichier de chaque script) :

| Match | Script | rho | Cas aberrants (ecart de rang >= 3 ou N/3) |
|---|---|---|---|
| 70b187b1 | slayerevents.lua | 0,49 | joueur 7 : 14 passages a zero pour 8 morts, 3 807 degats (rang 7,5 contre 3) |
| 9665b2b3 | arenaflagevents.lua+flageventstubs.lua | 0,64 | joueur 3 : 7 passages, 2 627 degats (rang 3,5 contre 7) ; joueur 4 : 10 passages, 2 316 degats (rang 8 contre 4) |
| 9c94aeab | areacaptureeventstubs.lua+sevents.lua | 0,80 | joueur 1 : 7 passages, 10 morts, 2 651 degats (rang 2 contre 5) |

Details du match 70b187b1 (joueur : passages a zero / vies avec zero sur vies / morts / degats subis) : 1 : 12 / 8 sur 9 / 8 / 3 616 ; 2 : 14 / 9 sur 10 / 9 / 4 422 ; 3 : 13 / 13 sur 16 / 16 / 4 550 ; 4 : 13 / 12 sur 17 / 16 / 4 619 ; 5 : 11 / 9 sur 12 / 11 / 3 904 ; 6 : 11 / 7 sur 8 / 8 / 3 072 ; 7 : 14 / 9 sur 9 / 8 / 3 807 ; 8 : 12 / 7 sur 11 / 11 / 4 264.
Match 9665b2b3 : 1 : 8 / 7 sur 8 / 7 / 2 620 ; 2 : 8 / 6 sur 8 / 7 / 2 418 ; 3 : 7 / 4 sur 7 / 7 / 2 627 ; 4 : 10 / 9 sur 10 / 10 / 2 316 ; 5 : 6 / 5 sur 6 / 5 / 1 979 ; 6 : 4 / 4 sur 5 / 4 / 1 703 ; 7 : 9 / 7 sur 9 / 8 / 2 660 ; 8 : 7 / 6 sur 9 / 8 / 2 277.
Match 9c94aeab : 1 : 7 / 7 sur 11 / 10 / 2 651 ; 2 : 7 / 5 sur 7 / 6 / 2 364 ; 3 : 9 / 7 sur 7 / 6 / 2 578 ; 4 : 8 / 6 sur 7 / 6 / 2 597 ; 5 : 11 / 10 sur 12 / 12 / 2 900 ; 6 : 9 / 7 sur 10 / 10 / 2 800 ; 7 : 13 / 10 sur 10 / 9 / 2 918 ; 8 : 7 / 3 sur 4 / 3 / 1 906.

Lecture : la regle est plausible. Le nombre de passages a zero suit les degats (rho poolee 0,83) et vaut en mediane un par mort, avec des cas ou il depasse le nombre de morts (14 passages pour 8 morts : recharges completes puis nouvelle chute, la regle « recharge puis nouvelle chute = deux » est exercee). Aucun cas extreme (beaucoup de boucliers pour tres peu de degats) dans les trois matchs ; les ecarts de rang relevent surtout de l'egalite des petits entiers (rangs ex aequo). Limite : seuls les matchs tous-suffisants ont ete mesures, comme demande ; la qualite de `sh` hors de ce perimetre n'est pas mesuree.

## Mesure 4. Ramassages natifs

Cles : `coverage.pickups.multiEvent` (le decodeur le definit comme le nombre de ramassages decodes suivis d'un autre evenement non decode : borne inferieure du rappel), `pickups[]` (`kind` = weapon, grenade ou equipment ; `xuid`), `grenadeLabels[].family`. Les ramassages par joueur sont comptes sur les humains du roster, zeros inclus (aucun ramassage sans joueur sauf 2 036, tous dans les 5 documents sans build).

### M4a. Ramassages natifs, borne inferieure (multiEvent > 0)

| Build | Docs avec coverage.pickups | multiEvent > 0 | Part | multiEvent cumule | Ramassages publies | Rapport multiEvent / publies |
|---|---|---|---|---|---|---|
| (inconnu) | 5 | 5 | 100.0 % | 1494 | 2036 | 73.4 % |
| HI_1_10_0 | 31 | 31 | 100.0 % | 8151 | 10870 | 75.0 % |
| HI_1_11_0 | 57 | 57 | 100.0 % | 14135 | 18578 | 76.1 % |
| HI_1_12_0 | 145 | 143 | 98.6 % | 12036 | 23810 | 50.6 % |
| HI_1_13_0 | 1200 | 1200 | 100.0 % | 82613 | 181328 | 45.6 % |
| HI_1_4_1 | 1 | 1 | 100.0 % | 142 | 190 | 74.7 % |
| HI_1_8_0 | 13 | 13 | 100.0 % | 2205 | 3207 | 68.8 % |
| HI_1_9_0 | 3 | 3 | 100.0 % | 232 | 436 | 53.2 % |
| TOTAL | 1455 | 1453 | 99.9 % | 121008 | 240455 | 50.3 % |

Lecture : le ramassage est une BORNE INFERIEURE dans 99,9 % des documents (1 453 sur 1 455 ; les deux exceptions sont en 1.12). Le `multiEvent` cumule vaut 46 % des ramassages publies en 1.13 et 51 % en 1.12 (73-76 % avant) : c'est un indicateur de l'etendue possible du manque, pas une mesure du manque (l'evenement suivant n'est pas necessairement un ramassage).

### M4b. Ramassages par joueur et par match (joueurs humains du roster, zeros comptes)

| Build | Paires joueur-match | Armes med (Q1-Q3) | Grenades med (Q1-Q3) | Equipements med (Q1-Q3) | Part avec 0 arme | Part avec 0 grenade |
|---|---|---|---|---|---|---|
| (inconnu) | 0 | n/a | n/a | n/a | n/a | n/a |
| HI_1_10_0 | 783 | 6 (2-12) | 2 (1-5) | 2 (1-4) | 9.5 % | 21.1 % |
| HI_1_11_0 | 1514 | 5 (2-10) | 2 (1-4) | 2 (1-4) | 9.0 % | 23.2 % |
| HI_1_12_0 | 1447 | 9 (3-14) | 4 (1-7) | 2 (0-4) | 7.7 % | 21.4 % |
| HI_1_13_0 | 10249 | 8 (4-14) | 5 (2-8) | 2 (1-4) | 5.9 % | 9.1 % |
| HI_1_4_1 | 28 | 2.5 (1-5) | 2 (1-3) | 1 (0-2) | 17.9 % | 17.9 % |
| HI_1_8_0 | 240 | 5 (2-11) | 2 (0-6) | 2 (1-4) | 15.0 % | 42.1 % |
| HI_1_9_0 | 85 | 2 (0-4) | 1 (0-2) | 0 (0-2) | 37.6 % | 49.4 % |
| TOTAL | 14346 | 8 (3-13) | 4 (2-7) | 2 (1-4) | 7.0 % | 13.3 % |

### M4c. Famille de grenade ramassee reliee a un libelle

| Build | Docs avec ramassage de grenade | grenadeLabels[].family rempli | Ramassages de grenade | Relies par family | Relies par nom (hors contrat) |
|---|---|---|---|---|---|
| (inconnu) | 5 | 0 | 533 | 0 | 533 (100.0 %) |
| HI_1_10_0 | 31 | 0 | 2491 | 0 | 2491 (100.0 %) |
| HI_1_11_0 | 57 | 0 | 4168 | 0 | 4168 (100.0 %) |
| HI_1_12_0 | 132 | 0 | 6947 | 0 | 6947 (100.0 %) |
| HI_1_13_0 | 1187 | 0 | 56947 | 0 | 56947 (100.0 %) |
| HI_1_4_1 | 1 | 0 | 62 | 0 | 62 (100.0 %) |
| HI_1_8_0 | 10 | 0 | 958 | 0 | 958 (100.0 %) |
| HI_1_9_0 | 3 | 0 | 102 | 0 | 102 (100.0 %) |
| TOTAL | 1426 | 0 | 72208 | 0 | 72208 (100.0 %) |


Lecture : `grenadeLabels[].family` n'est rempli dans aucun des 1 426 documents qui portent un ramassage de grenade ; le lien par la cle du contrat est donc impossible (0 %). La famille du ramassage (`grenade_frag`, `grenade_plasma`, `grenade_dynamo`, `grenade_spike`) correspond a un libelle par simple convention de nom (suffixe contre `en` en minuscules) pour 100 % des ramassages, mais cette convention n'est pas dans le contrat : ce n'est qu'une mesure de possibilite.

## Mesure 5. Fin de match contre feuille

Cle : `scoreTimeline.players` (frags, morts, assistances). La comparaison aux totaux de la feuille est IMPOSSIBLE : le document ne porte pas la feuille (voir « Cles introuvables »). Elle exige la base (ou la vue de match).

### M5a. scoreTimeline.players present

| Build | Docs | Avec scoreTimeline.players | Part | Humains du roster | Humains absents de la serie |
|---|---|---|---|---|---|
| (inconnu) | 5 | 1 | 20.0 % | 0 | 0 |
| HI_1_10_0 | 31 | 31 | 100.0 % | 783 | 24 |
| HI_1_11_0 | 57 | 57 | 100.0 % | 1514 | 42 |
| HI_1_12_0 | 145 | 145 | 100.0 % | 1447 | 27 |
| HI_1_13_0 | 1200 | 1199 | 99.9 % | 10241 | 153 |
| HI_1_4_1 | 1 | 0 | 0.0 % | 0 | 0 |
| HI_1_8_0 | 13 | 13 | 100.0 % | 240 | 13 |
| HI_1_9_0 | 3 | 3 | 100.0 % | 85 | 18 |
| TOTAL | 1455 | 1449 | 99.6 % | 14310 | 277 |

Lecture : 1 449 documents sur 1 455 portent la serie (99,6 %) ; les six autres sont le document 1.4.1, 4 documents sans build et 1 document 1.13. 277 humains du roster sur 14 310 (1,9 %) n'ont pas d'entree dans la serie ; les series vides valent zero (verifie ci-dessus).

Substitut interne (ce n'est PAS la mesure demandee) : sur les documents au verdict de score complet, derniere valeur des compteurs du calque contre le decompte des actions de score du meme document (`killPersonalScore`, `assistPersonalScore`). Il ne prouve rien sur la feuille ; il montre que le calque est coherent avec les actions.

### M5b. Substitut interne (docs au verdict de score complet seulement) : compteurs finaux du calque contre decompte des actions de score ; serie vide lue comme zero

| Build | Docs complets | Joueurs | Frags = nb de killPersonalScore | Part | Assistances = nb de assistPersonalScore | Part | Docs frags tous egaux | Docs assistances toutes egales |
|---|---|---|---|---|---|---|---|---|
| HI_1_11_0 | 12 | 313 | 312 | 99.7 % | 300 | 95.8 % | 11/12 | 5/12 |
| HI_1_12_0 | 97 | 868 | 868 | 100.0 % | 854 | 98.4 % | 97/97 | 87/97 |
| HI_1_13_0 | 834 | 6915 | 6911 | 99.9 % | 6864 | 99.3 % | 833/834 | 792/834 |
| TOTAL | 943 | 8096 | 8091 | 99.9 % | 8018 | 99.0 % | 941/943 | 884/943 |


Les assistances different plus souvent (autres actions d'assistance du jeu : marquage, detecteur, vehicule) : ce n'est pas un defaut du calque. Les ecarts de frags (5 joueurs sur 8 096) sont a examiner en fixation sur match reel.

## Mesure 6. Sprints et sauts

Cles : `coverage.stances.scanned` et `absent`, `coverage.stances.jumpEpisodes` ; denominateur de la derniere colonne : documents scanned.


Lecture : couverture des postures etablie dans 100 % des documents, tous builds, et au moins un episode de montee examine dans 100 % d'entre eux. Sur le parc, 2 499 582 episodes ont ete examines pour 656 151 sauts derives (26 %).

## Mesure 7. Medailles

Cles : `medals[]` (non vide), `medals[].name` (vide = sans nom), `coverage.medals.unknownName` (concorde : 0).


Lecture : une seule exception (1 document 1.13 sans medaille) ; 62 537 medailles, aucune sans nom.

## Ce qui reste a mesurer en base

- La part des frags qui recoivent une classe d'arme, par match avec rejeu (T.2b : `weapon_class` sur les evenements de frag) : elle exige la base partagee et le pont arme-par-frag, que ce script n'ouvre pas (base tenue en ecriture par un autre processus). La reserve du plan (un tiers des frags avec source au 2026-08-11) reste donc ouverte.
- La fixation de T.1 « fin de match = tableau de score » sur un match reel : il faut la feuille (frags, morts, assistances) de la base pour la comparer a la derniere valeur de `scoreTimeline.players`.
- Pour la mesure 1 : le score personnel « de l'API » du garde de T.3 vient de la vue de match ; ici il est remplace par la derniere valeur de `scoreTimeline.players[].score.total` du document (score affiche, `coverage.score.oracle` = displayed dans l'echantillon vu).


## Mesures en base (2026-11-18) — compteurs du rejeu contre le tableau de score

Methode. Un export unique de `match_participants` (29 107 lignes, 2 045 matchs, outil en lecture seule, base non verrouillee) vers `C:/t/t0-mesures/participants.tsv` ; puis un script Node (`base_vs_calque.js`), un document a la fois, sans base. Jointure : `matchId` complet du document = `match_id` ; joueur humain du roster (`roster[].xuid`, 16 chiffres nus, identique a `scoreTimeline.players[].xuid`, `scoreActions[].xuid` et a `match_participants.xuid`) ; bot = `roster[].bot` joint par `roster[].bid` (`bid(N.0)`), hors mesure. Valeurs lues comme la fiche : derniere valeur de `kills`, `deaths`, `assists` `.total` (serie vide = 0) ; somme de `scoreActions[].points` par destinataire, documents au verdict `coverage.scoreActions.complete` ; la fiche exige aussi `filmClockTrusted` (vrai dans les 944 documents complets). Les colonnes `score` et `personal_score` sont identiques sur les 28 954 lignes qui les portent : une seule comparaison les couvre. Rang = position dans `roster` du document. Journaux et scripts : `C:/t/t0-mesures/` (`base_vs_calque.js`, `pass_b.js`, `.json`, `.log`).

Taux de jointure : 14 303 humains du roster sur 14 346 (99,7 %) ont leur ligne en base, 99,4 a 100 % selon le build. Les 5 documents sans build (sans roster) ne peuvent pas etre joints : 133 lignes humaines de la base y correspondent.

### Frags, morts, assistances : derniere valeur du calque contre la base

Denominateur : humains joints dont le document porte une serie. Frags, morts et assistances sont aussi egaux tous les trois pour 14 015 joueurs sur 14 033 (99,87 %).

| Build | Docs | Humains du roster | Joints | Joints avec serie | Frags egaux | Morts egales | Assistances egales | Matchs compares | Matchs tous egaux |
|---|---|---|---|---|---|---|---|---|---|
| (inconnu) | 5 | 0 | 0 | 0 | n/a | n/a | n/a | 0 | n/a |
| HI_1_4_1 | 1 | 28 | 27 | 0 | n/a | n/a | n/a | 0 | n/a |
| HI_1_8_0 | 13 | 240 | 240 | 227 | 227 (100 %) | 227 (100 %) | 227 (100 %) | 13 | 13 (100 %) |
| HI_1_9_0 | 3 | 85 | 84 | 67 | 67 (100 %) | 67 (100 %) | 67 (100 %) | 3 | 3 (100 %) |
| HI_1_10_0 | 31 | 783 | 778 | 759 | 759 (100 %) | 759 (100 %) | 759 (100 %) | 31 | 31 (100 %) |
| HI_1_11_0 | 57 | 1514 | 1507 | 1472 | 1472 (100 %) | 1472 (100 %) | 1472 (100 %) | 57 | 57 (100 %) |
| HI_1_12_0 | 145 | 1447 | 1439 | 1420 | 1420 (100 %) | 1420 (100 %) | 1420 (100 %) | 145 | 145 (100 %) |
| HI_1_13_0 | 1200 | 10249 | 10228 | 10088 | 10082 (99,94 %) | 10078 (99,90 %) | 10078 (99,90 %) | 1199 | 1189 (99,17 %) |
| TOTAL | 1455 | 14346 | 14303 | 14033 | 14027 (99,96 %) | 14023 (99,93 %) | 14023 (99,93 %) | 1448 | 1438 (99,31 %) |

Tous les ecarts sont en 1.13 (26 ecarts sur 10 matchs : 6 frags, 10 morts, 10 assistances). Distribution de la difference (document moins base), les trois compteurs : -1 : 10 ; +1 : 3 ; de 2 a 3 en valeur absolue : 8 (+4, -4) ; de 4 a 10 : 5 (+4, -1) ; au-dela de 10 : 0. Huit des dix matchs n'ont qu'un joueur en ecart (en un a trois compteurs), un en a deux (`eba1e63f`) ; un match (`44e14331`) a ses 8 joueurs en ecart (12 ecarts, tous avec le document inferieur a la base, la plupart d'une unite), cause non etablie. Les 10 plus gros ecarts (match, rang, compteur, document -> base) :

| Match | Rang | Compteur | Document | Base | Ecart |
|---|---|---|---|---|---|
| d8cd155a | 3 | assistances | 9 | 1 | +8 |
| eba1e63f | 1 | assistances | 14 | 8 | +6 |
| 28109472 | 1 | morts | 11 | 6 | +5 |
| 400bb628 | 3 | morts | 16 | 12 | +4 |
| 78315425 | 1 | assistances | 0 | 4 | -4 |
| 040c5e7e | 6 | assistances | 6 | 3 | +3 |
| 28109472 | 1 | frags | 6 | 3 | +3 |
| 44e14331 | 5 | frags | 13 | 16 | -3 |
| e884797a | 5 | assistances | 4 | 1 | +3 |
| 400bb628 | 3 | assistances | 8 | 6 | +2 |

Humains joints sans serie dans le document : 270 (le calque n'a pas d'entree pour eux ; la fiche n'en tire pas de compteur). Pour 238 d'entre eux la base dit 0/0/0 (140 en 1.13, 19 en 1.12, 35 en 1.11, 19 en 1.10, 8 en 1.8.0, 17 en 1.9.0) : lire « sans serie » comme zero serait juste. Les 32 autres : 27 dans le seul document 1.4.1, qui n'a pas de calque du tout, et 5 en 1.8.0 avec une base non nulle.

### Score personnel : somme des points par action contre la base

Denominateur : humains joints des documents au verdict complet. La somme suit `personal_score` ET `score` a l'identique (colonnes egales).

| Build | Docs complets | Joueurs compares | Somme = base | Somme > base | Somme < base | Docs tous egaux |
|---|---|---|---|---|---|---|
| HI_1_11_0 | 12 | 326 | 226 (69,3 %) | 74 | 26 | 0/12 |
| HI_1_12_0 | 97 | 877 | 868 (99,0 %) | 0 | 9 | 96/97 (99,0 %) |
| HI_1_13_0 | 835 | 6991 | 6971 (99,7 %) | 0 | 20 | 832/835 (99,6 %) |
| TOTAL | 944 | 8194 | 8065 (98,4 %) | 74 | 55 | 928/944 (98,3 %) |

Les autres builds n'ont aucun document complet. En 1.12 et 1.13 reunies : 7 839 joueurs sur 7 868 (99,6 %). Les 29 ecarts de ces deux builds sont tous « somme inferieure » (actions manquantes) et tiennent dans 4 matchs : 1.13 `c75f33b8` (6 joueurs, ecart moyen -285), `df8fcbef` (7, -305), `9f57c612` (7, -171) ; 1.12 `35b75a31` (9, -189). Plus gros ecart : -720 (match `c75f33b8`, rang 5). En 1.11 (Grande bataille, voir M1b) l'ecart est dans les deux sens et touche 31 % des joueurs. Part de TOUS les joueurs joints qui verraient leur score avec le garde « verdict complet ET somme = base » : 8 065 sur 14 303 (56,4 %) ; 6 971 sur 10 228 (68,2 %) en 1.13, le reste etant des documents au verdict incomplet.

### Non joints

| Cote | Nombre | Ce que c'est |
|---|---|---|
| Humains du document absents de la base | 43 (41 matchs) | Entrees de roster sans aucune presence (`presence` vide), sans serie de score, sans tir : des places du film jamais occupees. Aucune ne porte de donnee. 21 en 1.13, 8 en 1.12, 7 en 1.11, 5 en 1.10, 1 en 1.9.0, 1 en 1.4.1 |
| Lignes humaines de la base sans joueur du document | 133 | Toutes dans les 5 documents sans build ni roster (impossibles a joindre) ; 0 ligne humaine orpheline dans les 1 450 autres documents |
| Serie de score hors roster | 1 | Dans un des 5 documents sans roster |
| Bots du document sans ligne en base | 126 sur 1 124 (998 joints) | Bots (`bot: true`) dont le `bid` n'est pas dans `match_participants` ; aucune serie de score |
| Lignes bot de la base sans bot du document | 55 sur 1 053 | Bots absents du roster du film (cause non verifiee) |

Conclusion. Les compteurs : le calque egale la base a la derniere image pour 99,96 % des frags, 99,93 % des morts et 99,93 % des assistances (99,87 % des joueurs sur les trois), 100 % avant 1.13 ; 10 matchs sur 1 448 (0,7 %) ont au moins un ecart, tous en 1.13, la plupart d'un joueur et d'une a trois unites, deux de plus de 5. Le score : sur les documents au verdict complet, la somme egale la base pour 98,4 % des joueurs (99,6 % en 1.12 et 1.13), mais 0 document 1.11 n'est juste en entier : le garde « somme = base » est necessaire et suffit a ecarter les 129 joueurs en ecart ; seuls 56,4 % des joueurs joints ont un verdict complet. Les builds les plus faibles : 1.13 pour les compteurs (seul build avec ecarts, 99,90 % en morts et assistances) et 1.11 pour le score (69,3 %, 12 documents).

## Mesures en base (2026-11-18) — source de dégât des morts et coup d'envoi

Lues par le pilote après les dérivés du parc, base partagée ouverte en lecture seule, serveur arrêté.

**Source de dégât des morts** (vue `match_kill_events_latest`) : 146 445 morts dans 1 462 matchs ; 97,7 % publiables ; 95,8 % avec une source de dégât connue et non contredite. Par match : médiane 98,7 %, premier décile 95,8 % ; 1 434 matchs à 90 % ou plus, 28 à 0 %. La classe d'arme d'un frag se déduit de cette source : le tableau « Éliminations par arme » de la fiche est donc rempli sur presque tout le parc. La mesure du 2026-08-11 (un tiers des frags avec une source) datait d'avant la régénération.

**Coup d'envoi lu dans le film** (`match_registry.t0_quality`, matchs avec rejeu) : 186 matchs sur 1 455 en `film_movement` (13 %), 1 266 en `ok` (estimation de l'API), 2 vides, 1 négatif. Inchangé par la régénération : le report du film vers le registre pour le parc déjà sur disque est la commande `cmd/backfill_t0_film`, qui n'a pas été lancée. Sa simulation (lecture seule) : 1 432 réparations, 8 refus (`burstTooSmall`), 15 inchangés ; coup d'envoi du film entre 15,2 s et 213,2 s, médiane 34,1 s ; plusieurs matchs portent aujourd'hui un coup d'envoi à 0 ou 1 ms. L'écriture attend l'accord du user.
