# Mesures du lot A : la prevision de reapparition (2026-11-15)

Plan : `PLAN_REAPPARITIONS_2026-11-15.md`, lot A (A.6). Les tableaux ci-dessous sont les sorties de
`levelup spawn-model evaluate`, recopiees sans retouche.

## Conditions

- Branche `feat/reapp-a` (partie de `origin/release/v8.2.0`), binaire compile depuis cette branche.
- Parc : dossier de donnees du dossier principal, passe par `LEVELUP_DATA_DIR` ; aucune ecriture dans ce dossier,
  aucun lien. Les artefacts sont au schema 116, a 33 ms par frame, 1 456 artefacts presents.
- **Voie de decision « reapparition » : le REPLI sur l'ecart de 4 a 15 s, partout.** Le parc local est au schema 116 :
  il ne porte ni compte a rebours de reapparition, ni fin de vie lue (le decodeur v1.3.7 de cette base ne les publie
  pas). Compteur de source sur tout le parc lu : 0 par le film, 111 088 par le repli (avant les filtres des mesures),
  0 debut de manche reconnu. Les voies `film` et `fin de manche` sont ecrites et testees cote pur (A.1) ; leur accord
  avec le repli se mesurera apres la regeneration du parc a la v1.3.9 (case A.8 du plan).
- Base partagee ouverte en lecture par `OpenReadForQuery` (aucun autre processus ne la tenait : pas de serveur sur le
  port 8000 pendant les mesures).
- Famille d'un match : `game_variant_name` + `mode_category` du registre (`cmd_spawn_model_modes.go`). BTB et rotations
  Husky Raid exclus ; Fiesta (dont Super Fiesta) prime sur les modes d'objectif. `levelup spawn-model evaluate
  --variantes` imprime la table variante -> famille du parc.
- Validation croisee a 5 plis, le pli d'un evenement etant celui de son match (hachage FNV du match_id).
- Une carte n'est mesuree que si elle est vue dans au moins 8 matchs de la famille (seuil de la mesure de depart) ;
  les familles clairsemees sont aussi mesurees avec un seuil de 3 matchs. Les reapparitions d'un joueur dont l'equipe
  est inconnue (211 sur le parc lu) sont ecartees ; un joueur d'equipe inconnue n'est compte ni
  coequipier ni adversaire.

### Commandes exactes

```
export LEVELUP_DATA_DIR=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data
cd apps/go-api && go build -o levelup.exe ./cmd/levelup/        # puis, depuis la racine du worktree :
levelup.exe spawn-model evaluate --sections m1,m2,m3,m4 --par-carte               # seuil de 8 matchs par carte
levelup.exe spawn-model evaluate --sections m2,m3,m4 --matchs-min-carte 3         # familles clairsemees
levelup.exe spawn-model evaluate --variantes                                       # table des variantes
```

Duree d'une passe : lecture des 1 306 artefacts retenus en 3 min 25 s a 3 min 56 s, puis environ 5 min d'ajustements.

## Bilan de lecture

| artefacts | lus | sans registre | hors familles | illisibles |
|---|---|---|---|---|
| 1456 | 1306 | 0 | 150 | 0 |

| reapparitions (source film) | reapparitions (repli ecart 4-15 s) | debuts de manche | sans equipe | non finies |
|---|---|---|---|---|
| 0 | 111088 | 0 | 211 | 0 |

- assassin : 550 matchs lus
- bases : 77 matchs lus
- colline : 48 matchs lus
- drapeau : 161 matchs lus
- fiesta : 463 matchs lus
- oddball : 7 matchs lus

## M-1 temoin

43920 reapparitions sur 45 cartes (annexe : 43966 sur 45), hors candidats : 151

| mesure | depot | annexe | ecart |
|---|---|---|---|
| hasard, a moins de 8 m | 14.1 % | 14.1 % | +0.0 |
| frequence seule, 1er, a moins de 8 m | 15.5 % | 15.6 % | -0.1 |
| frequence seule, 3 premiers, a moins de 8 m | 38.4 % | 38.4 % | +0.0 |
| modele, 1er, a moins de 8 m | 45.0 % | 44.9 % | +0.1 |
| modele, 3 premiers, a moins de 8 m | 62.6 % | 62.5 % | +0.1 |
| frequence seule, point exact, 1er | 4.4 % | 4.5 % | -0.1 |
| frequence seule, point exact, 3 premiers | 12.8 % | 12.8 % | -0.0 |
| modele, point exact, 1er | 10.9 % | 11.0 % | -0.1 |
| modele, point exact, 3 premiers | 27.7 % | 27.7 % | +0.0 |

| variable | poids |
|---|---|
| ennemi_le_plus_proche_10m | +1.99 |
| ennemis_a_moins_de_20m | -0.66 |
| equipier_le_plus_proche_10m | -1.08 |
| distance_au_lieu_de_la_mort_10m | +0.15 |
| frequence_log | +0.45 |
| distance_au_barycentre_ennemi_10m | -0.94 |

| carte | n | top1 freq | top1 modele | top3 freq | top3 modele | candidats |
|---|---|---|---|---|---|---|
| Kaiketsu | 886 | 13.7 | 33.5 | 39.2 | 67.6 | 17 |
| Solution | 809 | 14.8 | 31.6 | 39.3 | 65.4 | 19 |
| Dredge | 1097 | 11.4 | 26.3 | 30.7 | 56.0 | 35 |
| Houseki | 972 | 9.7 | 19.5 | 22.5 | 47.6 | 29 |
| Perilous | 711 | 12.1 | 23.6 | 29.4 | 53.6 | 33 |
| Vagabond | 852 | 2.9 | 13.5 | 12.6 | 35.9 | 53 |
| Chasm | 776 | 3.1 | 10.6 | 12.8 | 35.3 | 45 |
| Sylvanus | 921 | 6.4 | 16.6 | 23.3 | 44.8 | 32 |
| Solitude | 719 | 2.0 | 11.4 | 11.0 | 32.3 | 58 |
| Nemesis | 1030 | 2.3 | 12.5 | 10.8 | 31.0 | 51 |
| Dynasty | 1320 | 2.7 | 13.8 | 11.6 | 31.5 | 55 |
| Kiken'na | 1074 | 14.0 | 22.1 | 28.7 | 47.5 | 35 |
| Curfew | 1174 | 4.6 | 12.9 | 15.4 | 34.0 | 52 |
| Prism | 755 | 3.1 | 10.9 | 9.5 | 27.9 | 40 |
| Argyle | 1114 | 3.9 | 13.6 | 18.8 | 36.0 | 35 |
| Goliath | 1034 | 8.3 | 15.4 | 19.3 | 36.1 | 52 |
| Absolution | 1220 | 3.4 | 8.2 | 8.4 | 25.1 | 63 |
| Recharge | 1009 | 3.8 | 10.7 | 12.9 | 29.4 | 58 |
| Detachment | 1321 | 2.3 | 8.3 | 10.4 | 26.0 | 59 |
| Live Fire | 768 | 5.6 | 9.8 | 13.4 | 28.6 | 46 |
| The Pit | 1661 | 2.2 | 6.6 | 5.4 | 20.5 | 81 |
| Starboard | 1348 | 5.1 | 10.3 | 10.6 | 25.7 | 52 |
| Cliffside | 1002 | 4.2 | 8.9 | 11.4 | 26.2 | 91 |
| Salvation | 778 | 4.0 | 11.7 | 18.9 | 32.5 | 41 |
| Catalyst | 1092 | 3.9 | 7.8 | 9.9 | 23.1 | 96 |
| Illusion | 933 | 3.6 | 9.1 | 6.8 | 19.8 | 78 |
| Cliffhanger | 1126 | 2.9 | 6.9 | 5.3 | 17.8 | 86 |
| Shogun | 678 | 7.7 | 11.7 | 21.5 | 33.9 | 41 |
| Empyrean | 945 | 1.3 | 7.3 | 6.7 | 18.5 | 67 |
| Elevation | 778 | 3.3 | 7.2 | 10.7 | 22.0 | 67 |
| Origin | 1117 | 2.5 | 4.7 | 3.8 | 15.0 | 101 |
| Takamanohara | 884 | 2.4 | 8.0 | 9.3 | 20.5 | 68 |
| Streets | 1134 | 1.9 | 6.0 | 7.5 | 18.5 | 76 |
| High Ground | 924 | 2.7 | 7.6 | 7.4 | 18.1 | 71 |
| Domicile | 937 | 4.3 | 6.3 | 7.6 | 17.8 | 81 |
| Bazaar | 1114 | 3.1 | 6.3 | 6.7 | 15.7 | 97 |
| Shiro | 1088 | 1.9 | 7.2 | 8.8 | 17.7 | 58 |
| Isolation | 748 | 4.3 | 8.3 | 10.4 | 18.6 | 64 |
| Snowbound | 1020 | 2.3 | 5.6 | 6.2 | 14.3 | 91 |
| Behemoth | 859 | 0.7 | 4.1 | 3.4 | 11.2 | 133 |
| Fortress | 786 | 2.4 | 5.1 | 7.9 | 15.6 | 87 |
| Forbidden | 1028 | 0.4 | 2.9 | 1.1 | 8.1 | 136 |
| Aquarius | 774 | 1.3 | 3.1 | 3.5 | 9.0 | 91 |
| Banished Narrows | 837 | 3.5 | 6.6 | 13.1 | 18.4 | 67 |
| Smallhalla | 767 | 2.6 | 4.0 | 7.4 | 11.5 | 105 |

**Lecture.** Tous les ecarts au temoin sont de 0,1 point ou moins (seuil du plan : 1 point). Le depot retrouve 43 920
reapparitions sur 45 cartes (annexe de depart : 43 966) : l'ecart de 46 evenements vient de deux choix de portage, sans
effet mesurable : l'ecart de 4 a 15 s se lit en millisecondes avec la cadence du document (33 ms par frame : 4 s valent
121 frames, la mesure jetable comptait 120), et les vies dont l'equipe est inconnue ne sont plus comptees parmi les
adversaires. Les poids different de 0,01 au plus.

## M-2 ligne de vue approchee

Methode : segment des yeux de l'ennemi (pied + 1,6 m) au torse du point candidat (pied + 1,0 m) contre les boites
`structure[]` de l'artefact (emprise x0, y0, x1, y1 ; faces `z` haute et `zb` basse) ; les boites qui contiennent une
extremite sont ecartees (emprise alignee d'une piece tournee). Septieme variable : nombre d'ennemis a moins de 40 m qui
voient le point. Le point candidat est vise a 1,0 m et non a la hauteur des yeux : ecart mineur a la formulation du plan
(« z + 1,6 m environ, vers le point candidat »), pose pour viser le torse d'un joueur qui reapparait.

**Limite de la mesure, a lire avant le gain.** `structure[]` n'existe que pour les cartes dont un fichier de structure
fige a ete extrait : 4 artefacts sur 60 dans un echantillon tire au hasard ; sur le parc lu, **3 cartes sur 60** en
portent (Streets, Streets - Ranked, Cliffhanger). Le gain ne se mesure donc que sur ces cartes, pas « toutes cartes
confondues ».

Cartes dont les artefacts portent la structure : 3 sur 60 lues.
- Streets - Ranked (10323 boites)
- Cliffhanger (10067 boites)
- Streets (10323 boites)


| mode | reapparitions | a 8 m, 3 premiers : 6 variables | 7 variables | gain | zone nommee, 3 premiers : 6 variables | 7 variables | gain | a 8 m, 1er : gain |
|---|---|---|---|---|---|---|---|---|
| assassin | 2260 | 60.3 | 60.5 | +0.2 | 75.8 | 76.4 | +0.5 | +0.3 |
| fiesta | 6476 | 58.6 | 59.2 | +0.6 | 76.9 | 76.9 | +0.0 | +0.4 |
| toutes familles (moyenne ponderee) | 8736 | | | +0.5 | | | +0.2 | +0.4 |

## M-3 par mode

Seuil de 8 matchs par carte (celui de la mesure de depart) :

Zone a 8 m : un des candidats retenus est a moins de 8 m du point reel. Zone nommee : la zone du catalogue (une zone recoit la somme des probabilites de ses candidats). Couverture : part des reapparitions dont le point reel a une zone.

| mode | matchs | reapparitions | cartes | candidats/carte | a 8 m, 3 premiers : hasard | freq | modele | a 8 m, 1er : freq | modele | zone nommee, 3 premiers : hasard | freq | modele | couverture des zones | exact 1er : modele | exact 3 : modele |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| assassin | 550 | 43920 | 45 | 65 | 14.1 | 38.4 | 62.6 | 15.5 | 45.0 | 23.1 | 48.6 | 75.1 | 97 % | 10.9 | 27.7 |
| fiesta | 463 | 39178 | 14 | 82 | 11.8 | 24.2 | 56.3 | 12.8 | 41.7 | 23.0 | 48.0 | 76.4 | 98 % | 7.8 | 21.5 |
| drapeau | 161 | 4359 | 5 | 66 | 13.2 | 39.5 | 54.6 | 19.8 | 38.4 | 25.6 | 47.5 | 72.3 | 96 % | 8.4 | 23.2 |
| bases | 77 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| colline | 48 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| oddball | 7 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |

Seuil de 3 matchs par carte (rend mesurables Bases, Roi de la colline et Drapeau sur davantage de cartes) :

Zone a 8 m : un des candidats retenus est a moins de 8 m du point reel. Zone nommee : la zone du catalogue (une zone recoit la somme des probabilites de ses candidats). Couverture : part des reapparitions dont le point reel a une zone.

| mode | matchs | reapparitions | cartes | candidats/carte | a 8 m, 3 premiers : hasard | freq | modele | a 8 m, 1er : freq | modele | zone nommee, 3 premiers : hasard | freq | modele | couverture des zones | exact 1er : modele | exact 3 : modele |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| assassin | 550 | 46007 | 50 | 64 | 14.3 | 38.8 | 62.8 | 15.8 | 45.3 | 23.1 | 48.3 | 75.0 | 97 % | 10.8 | 27.5 |
| fiesta | 463 | 39178 | 14 | 82 | 11.8 | 24.2 | 56.3 | 12.8 | 41.7 | 23.0 | 48.0 | 76.4 | 98 % | 7.8 | 21.5 |
| drapeau | 161 | 12948 | 27 | 59 | 14.4 | 42.6 | 56.9 | 19.0 | 39.3 | 27.4 | 49.5 | 71.5 | 98 % | 8.5 | 23.1 |
| bases | 77 | 4058 | 15 | 43 | 16.2 | 44.3 | 64.9 | 19.1 | 48.6 | 22.7 | 43.6 | 71.4 | 99 % | 12.8 | 31.3 |
| colline | 48 | 1307 | 5 | 46 | 14.9 | 40.4 | 66.6 | 16.3 | 49.4 | 19.1 | 41.9 | 69.5 | 100 % | 9.0 | 25.2 |
| oddball | 7 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |

**Oddball** : 7 matchs lus, 720 reapparitions, mais aucune carte n'a assez de matchs pour des candidats (au moins deux
matchs par point, au moins deux candidats) : non mesurable sur ce parc, quel que soit le seuil.

La metrique « zone nommee » prend les zones du catalogue de callouts (meme cascade que l'onglet Tactique, zone resolue
par `tactical.NommerZone` au point et a la hauteur du point reel) ; une zone recoit la somme des probabilites de ses
candidats. La metrique « zone a 8 m » compte un succes quand un des candidats retenus est a moins de 8 m du point reel.

## M-4 prevision a l'instant de la mort contre l'instant de la reapparition

Memes evenements et memes candidats dans les deux colonnes (evenements dont les deux instantanes ont au moins un
adversaire) ; chaque modele est ajuste sur son propre instant. Seuil de 8 matchs par carte :

Memes evenements (les deux instantanes ont au moins un adversaire), memes candidats ; chaque modele est ajuste sur son propre instant.

| mode | reapparitions | a 8 m, 3 premiers : reapparition | mort | ecart | a 8 m, 1er : reapparition | mort | ecart | zone nommee, 3 premiers : reapparition | mort | ecart |
|---|---|---|---|---|---|---|---|---|---|---|
| assassin | 43901 | 62.6 | 46.0 | -16.6 | 45.0 | 29.2 | -15.8 | 75.0 | 59.8 | -15.2 |
| fiesta | 39149 | 56.3 | 37.7 | -18.5 | 41.7 | 24.7 | -17.0 | 76.4 | 59.2 | -17.2 |
| drapeau | 4356 | 54.4 | 45.3 | -9.2 | 38.4 | 29.6 | -8.7 | 72.2 | 61.3 | -10.9 |
| bases | 6024 | (populations non comparables : 0 / 0) | | | | | | | | |
| colline | 3404 | (populations non comparables : 0 / 0) | | | | | | | | |
| oddball | 720 | (populations non comparables : 0 / 0) | | | | | | | | |

Seuil de 3 matchs par carte :

Memes evenements (les deux instantanes ont au moins un adversaire), memes candidats ; chaque modele est ajuste sur son propre instant.

| mode | reapparitions | a 8 m, 3 premiers : reapparition | mort | ecart | a 8 m, 1er : reapparition | mort | ecart | zone nommee, 3 premiers : reapparition | mort | ecart |
|---|---|---|---|---|---|---|---|---|---|---|
| assassin | 45987 | 62.8 | 46.1 | -16.7 | 45.3 | 29.5 | -15.8 | 75.0 | 59.7 | -15.3 |
| fiesta | 39149 | 56.3 | 37.7 | -18.5 | 41.7 | 24.7 | -17.0 | 76.4 | 59.2 | -17.2 |
| drapeau | 12942 | 56.9 | 46.9 | -10.0 | 39.2 | 30.2 | -9.0 | 71.6 | 61.9 | -9.7 |
| bases | 4055 | 64.9 | 43.7 | -21.2 | 48.7 | 28.3 | -20.4 | 71.5 | 51.3 | -20.2 |
| colline | 1305 | 66.4 | 36.5 | -30.0 | 49.6 | 23.8 | -25.8 | 69.3 | 49.7 | -19.6 |
| oddball | 720 | (populations non comparables : 0 / 0) | | | | | | | | |

Les lignes « populations non comparables » sont des familles sans carte mesurable au seuil retenu (cf. M-3).

## Conclusions que la regle du plan en tire (le user tranche D-2 et D-4)

**D-2 (mode couvert si au moins 50 % dans les 3 premieres zones).** Modele, 3 premieres zones nommees : Assassin
75,1 %, Fiesta 76,4 %, Drapeau 72,3 % (71,5 % a 3 matchs par carte), Bases 71,4 %, Roi de la colline 69,5 % (seuil de
3 matchs pour les deux derniers). Sur la metrique « zone a 8 m dans les 3 premiers » : 62,6 / 56,3 / 54,6 / 64,9 / 66,6 %.
La regle couvre donc **Assassin, Fiesta, Drapeau, Bases et Roi de la colline**, sur les deux metriques. **Oddball** n'est
pas mesurable (7 matchs) : non couvert faute de mesure. BTB est exclu d'emblee. Reserves : Bases et Colline reposent sur
15 et 5 cartes a 3 matchs minimum seulement, et tout est mesure sur le REPLI, pas sur la lecture du film.

**D-4 (ligne de vue gardee si gain d'au moins 3 points sur « 3 premieres zones », toutes cartes confondues).** Gain
mesure : +0,2 a +0,6 point (zone a 8 m, 3 premiers), +0,0 a +0,5 point (zone nommee, 3 premiers), moyenne ponderee +0,5 et
+0,2, sur les 3 cartes qui portent la structure. La regle conclut a **ecarter la ligne de vue**. Reserve : la mesure
« toutes cartes confondues » est impossible (structure absente de 57 cartes sur 60) et la geometrie est une approximation
par boites ; la conclusion vaut pour cette approximation et ces 3 cartes.

**M-4 (ce que vaut la prevision 5 s avant).** A l'instant de la mort, la prevision perd 9 a 30 points sur « zone a 8 m
dans les 3 premiers » (Assassin -16,6, Fiesta -18,5, Drapeau -9,2 ; -10,0 a 3 matchs, Bases -21,2, Colline -30,0) et 10 a
20 points sur la zone nommee. Elle reste tres au-dessus du hasard (zone nommee, 3 premieres : Assassin 59,8 %, Fiesta
59,2 %, Drapeau 61,3 %, contre un hasard de 23 a 27 %), mais Bases (51,3 %) et Colline (49,7 %) tombent a la limite du
seuil de 50 % de D-2.

## A.5 rattrapage `levelup tactical-rasters --backfill`, sur une copie

Copie de 20 artefacts (23 Mo) dans un dossier temporaire du scratchpad, racine passee par `LEVELUP_DATA_DIR` ; rien n'est
ecrit dans le dossier de donnees principal. Le binaire « schema 6 » est celui de la base de la branche (`git archive`
de `HEAD`, compile a part).

| passe | sidecars lus / ecrits / deja a jour | duree | par artefact | taille des 20 sidecars |
|---|---|---|---|---|
| binaire de la release (schema 6), sidecars absents | 20 / 20 / 0 | 3 019 ms | 151 ms | 13 579 949 octets |
| ce lot (schema 7) sur les 20 sidecars v6 | 20 / 20 / 0 | 3 341 ms | 167 ms | 14 538 426 octets |
| ce lot, seconde passe immediate | 20 / 0 / 20 | 374 ms | 19 ms | inchange |
| ce lot, sidecars absents | 20 / 20 / 0 | 2 990 ms | 150 ms | 14 538 426 octets |

Avant : 20 sidecars en schema 6, 0 en schema 7. Apres : 0 en schema 6, 20 en schema 7. Le cout du schema 7 est de l'ordre
de +10 % de temps (rattrapage sur v6) et +7 % d'octets ; a 150-170 ms par artefact, les 1 455 artefacts du parc se
regenerent en environ 4 minutes. Contenu v7 sur ces 20 : 1 880 spawns dont 1 689 reapparitions (toutes `repli_ecart`) et
10 952 positions d'instantane.

Commande de controle (compte les sidecars en schema 7 d'un dossier de donnees) :

```
grep -l '"schema_version":7' <racine_donnees>/cache/replays/halo_infinite/rasters/*.json | wc -l
```

`grep -L '"schema_version":7' ...` liste ceux qui restent a regenerer.
