# Mesures du lot A : la prevision de reapparition (2026-11-15)

Plan : `PLAN_REAPPARITIONS_2026-11-15.md`, lot A (A.6). Les tableaux ci-dessous sont les sorties de
`levelup spawn-model evaluate`, recopiees sans retouche.

## Correction du 2026-11-15 (revue du lot A) : ce qui a changé, AVANT / APRÈS

La revue du lot A a relevé trois défauts de méthode, hérités de l'outil jetable de la mesure de départ
(`outil_mesure_2026-11-15.go.txt`) et reportés tels quels dans la CLI. Les mesures ont été rejouées avec le code
corrigé, avec les mêmes commandes, sur le même parc (1 306 artefacts lus, schéma 116, repli sur l'écart de 4 à 15 s).

| Défaut | Avant | Après |
|---|---|---|
| Hasard de l'échelle « à 8 m, 3 premiers » | la chance d'UN tirage (part des candidats à moins de 8 m), publiée comme hasard du top 3 | l'espérance exacte de trois tirages sans remise : `1 - C(n-k, 3) / C(n, 3)` par événement (k candidats à moins de 8 m, n candidats), moyennée ; le hasard d'un tirage reste publié, sous le nom « 1er tirage » |
| Candidats | les points vus dans au moins 2 matchs, le match évalué compris : son point réel devenait candidat dès qu'UN autre match le voyait | les points vus dans au moins 2 AUTRES matchs que celui de l'événement (comme le fera le service, qui ne voit jamais le match à prévoir) ; la fréquence (`frequence_log`) était déjà comptée hors du match évalué (test ajouté) |
| Égalités de score au rayon de 8 m | départagées par l'ordre des candidats (le total des réapparitions, match évalué compris) : favorisaient le point réel | les ex aequo se partagent à parts égales (espérance d'un départ au hasard entre eux) |

L'outil jetable avait les trois mêmes biais (candidats tirés de tous les événements, match testé compris ; départage
par `sort.SliceStable` sur une liste triée par total ; hasard = `in / len(pos)`, un seul tirage). Seule la fréquence
y était déjà hors du match testé (`c.total - c.byMatch[e.match]`). Le chiffre corrigé de la mesure de départ est donc
la colonne « APRÈS » du M-1 ci-dessous.

### M-1 témoin, Assassin 4c4 hors Fiesta et BTB, 45 cartes, seuil de 8 matchs par carte

| mesure | départ (outil jetable) | AVANT (CLI du lot A) | APRÈS (corrigé) |
|---|---|---|---|
| réapparitions retenues | 43 966 | 43 920 | 43 920 |
| hors candidats | 149 | 151 | 512 |
| hasard, à moins de 8 m, 1er tirage | 14,1 % | 14,1 % | 14,1 % |
| hasard, à moins de 8 m, 3 tirages | (jamais mesuré) | 14,1 % (le hasard d'UN tirage, publié à tort) | **36,2 %** |
| fréquence seule, 1er, à moins de 8 m | 15,6 % | 15,5 % | 15,4 % |
| fréquence seule, 3 premiers, à moins de 8 m | 38,4 % | 38,4 % | 38,3 % |
| modèle, 1er, à moins de 8 m | **44,9 %** | 45,0 % | 44,9 % |
| modèle, 3 premiers, à moins de 8 m | **62,5 %** | 62,6 % | 62,8 % |
| fréquence seule, point exact, 1er / 3 premiers | 4,5 / 12,8 % | 4,4 / 12,8 % | 4,4 / 12,8 % |
| modèle, point exact, 1er / 3 premiers | 11,0 / 27,7 % | 10,9 / 27,7 % | 10,9 / 27,8 % |

Poids (tout le jeu), départ / AVANT / APRÈS : ennemi le plus proche +1,98 / +1,99 / +1,99 ; ennemis à moins de 20 m
-0,65 / -0,66 / -0,65 ; équipier le plus proche -1,08 / -1,08 / -1,08 ; distance à la mort +0,15 / +0,15 / +0,15 ;
fréquence +0,45 / +0,45 / +0,49 ; barycentre ennemi -0,93 / -0,94 / -0,93.

**Ce que les chiffres corrigés disent.** Le modèle ne bouge presque pas : les candidats hors du match évalué et les
égalités n'avaient presque aucun effet sur lui (+0,3 point au plus). Ce qui change est la lecture du « 62 % » :
le hasard à 8 m sur trois tirages est de 36 %, pas de 14 %, et la fréquence seule (38 %) ne le dépasse que de 2 points.
Le gain du modèle est de 26 points sur le hasard à trois tirages (62,8 contre 36,2), et non de 48. Au premier choix
(14 % de hasard, 45 % pour le modèle), la lecture d'origine tenait.

### Les mêmes tableaux, AVANT / APRÈS (M-3, zone à 8 m puis zone nommée, 3 premiers)

Seuil de 8 matchs par carte :

| mode | hasard 8 m, 3 tirages | fréquence | modèle | zone nommée, modèle |
|---|---|---|---|---|
| assassin | 14,1 -> **36,2** | 38,4 -> 38,3 | 62,6 -> 62,8 | 75,1 -> 75,0 |
| fiesta | 11,8 -> **31,1** | 24,2 -> 24,1 | 56,3 -> 56,3 | 76,4 -> 76,4 |
| drapeau | 13,2 -> **34,7** | 39,5 -> 38,8 | 54,6 -> 55,1 | 72,3 -> 72,1 |

Seuil de 3 matchs par carte :

| mode | hasard 8 m, 3 tirages | fréquence | modèle | zone nommée, modèle |
|---|---|---|---|---|
| assassin | 14,3 -> **36,5** | 38,8 -> 38,7 | 62,8 -> 63,0 | 75,0 -> 74,9 |
| fiesta | 11,8 -> **31,1** | 24,2 -> 24,1 | 56,3 -> 56,3 | 76,4 -> 76,4 |
| drapeau | 14,4 -> **37,5** | 42,6 -> 41,5 | 56,9 -> 57,7 | 71,5 -> 71,2 |
| bases | 16,2 -> **41,3** | 44,3 -> 44,3 | 64,9 -> 67,6 | 71,4 -> 69,5 |
| colline | 14,9 -> **37,7** | 40,4 -> 38,1 | 66,6 -> 64,8 | 69,5 -> 67,1 |

Sur Bases, la fréquence seule (44,3) dépasse à peine le hasard à trois tirages (41,3) : la prévision y vient du modèle.
La règle D-2 (au moins 50 % des 3 premières zones nommées) tient encore sur les cinq familles à l'instant de la
réapparition. Oddball reste non mesurable (7 matchs).

### M-2, M-4 : écarts AVANT / APRÈS

| mesure | AVANT | APRÈS |
|---|---|---|
| M-2, gain de la 7e variable, 3 premiers à 8 m (Assassin / Fiesta / moyenne pondérée) | +0,2 / +0,6 / +0,5 | +0,3 / +0,6 / +0,5 |
| M-2, gain, zone nommée (Assassin / Fiesta / moyenne) | +0,5 / 0,0 / +0,2 | +0,5 / 0,0 / +0,2 |
| M-2, gain, 1er à 8 m (Assassin / Fiesta / moyenne) | +0,3 / +0,4 / +0,4 | +0,5 / +0,4 / +0,5 |
| M-4, écart mort - réapparition, 3 premiers à 8 m, seuil 8 : Assassin / Fiesta / Drapeau | -16,6 / -18,5 / -9,2 | -16,6 / -18,5 / -9,3 |
| M-4, même écart, seuil 3 : Drapeau / Bases / Colline | -10,0 / -21,2 / -30,0 | -9,5 / -20,9 / -26,4 |
| M-4, zone nommée à l'instant de la mort, seuil 3 : Drapeau / Bases / Colline | 61,9 / 51,3 / 49,7 | 61,6 / 49,7 / 48,3 |

D-4 (ligne de vue écartée : gain sous 3 points) : inchangé. **À l'instant de la mort, Bases (49,7 %) et Roi de la colline
(48,3 %) passent sous le seuil de 50 % de D-2** (ils le tenaient de peu : 51,3 et 49,7). La prévision affichée pendant le
compte à rebours ne les couvrirait donc pas sur ce parc ; Assassin, Fiesta et Drapeau restent à 59 à 62 %.

### Le fichier de poids (`levelup spawn-model fit --matchs-min-carte 3`), AVANT / APRÈS

Les mêmes réapparitions retenues qu'avant (le seuil de 3 matchs par carte du fichier livré est maintenant celui de `fit`
par défaut, et la commande est écrite dans `docs/COMMANDS.md`). Les effectifs `cartes` et `matchs` comptent désormais les
réapparitions retenues, comme `reapparitions` : avant, ils comptaient le jeu lu avant tout filtre.

| famille | réapparitions | cartes AVANT -> APRÈS | matchs AVANT -> APRÈS | hasard 8 m, 3 tirages | modèle à 8 m, 3 premiers | exact, 3 premiers |
|---|---|---|---|---|---|---|
| assassin | 46 007 | 55 -> 50 | 547 -> 542 | 14,3 -> **36,5** | 62,8 -> 63,0 | 27,5 -> 27,6 |
| fiesta | 39 178 | 24 -> 14 | 463 -> 452 | 11,8 -> **31,1** | 56,3 -> 56,3 | 21,5 -> 21,5 |
| drapeau | 12 948 | 35 -> 27 | 161 -> 150 | 14,4 -> **37,5** | 56,9 -> 57,7 | 23,1 -> 22,9 |
| bases | 4 058 | 30 -> 15 | 77 -> 55 | 16,2 -> **41,3** | 64,9 -> 67,6 | 31,3 -> 30,0 |
| colline | 1 307 | 28 -> 5 | 48 -> 18 | 14,9 -> **37,7** | 66,6 -> 64,8 | 25,2 -> 20,7 |
| tous | 109 454 | 60 -> 51 | 1 303 -> 1 293 | 13,1 -> **34,0** | 58,5 -> 58,6 | 24,4 -> 24,4 |

Le champ `definition` du fichier dit maintenant ce que comptent les effectifs, le seuil de cartes et la règle de
candidats ; le fichier porte aussi `rayon8m_top1_hasard` (le hasard d'un tirage), à côté de `rayon8m_top3_hasard`.
Oddball (720 réapparitions, aucune carte mesurable) n'a pas de poids propres : une réapparition d'Oddball retombe sur
la famille « tous » (`tactical.PoidsPourFamille`).

Les tableaux complets APRÈS (sorties de `levelup spawn-model evaluate`, recopiées) suivent ; les tableaux
d'origine (AVANT) sont conservés plus bas, tels que le lot A les avait écrits.

### Commandes de la passe corrigée

```
export LEVELUP_DATA_DIR=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data
levelup.exe spawn-model evaluate --sections m1,m2,m3,m4 --par-carte               # seuil de 8 matchs par carte
levelup.exe spawn-model evaluate --sections m2,m3,m4 --matchs-min-carte 3         # familles clairsemees
levelup.exe spawn-model fit --matchs-min-carte 3                                  # ecrit reference/spawn_model.json
```

Les familles de modes viennent désormais de `config/titles/halo_infinite/mappings/spawn_families.toml` (ordre d'essai :
Fiesta, Drapeau, Bases, Colline, Oddball, Assassin) ; aucune affectation ne change : le test du fichier reprend la table
de cas de l'ancien test. Les tableaux suivent dans l'ordre du fichier (Fiesta d'abord), non plus celui du lot A.

### APRÈS, sorties complètes

#### Seuil de 8 matchs par carte (`--sections m1,m2,m3,m4 --par-carte`)

##### Bilan de lecture

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

##### M-1 temoin : Assassin hors Fiesta et BTB

43920 reapparitions sur 45 cartes (annexe : 43966 sur 45), hors candidats : 512

| mesure | depot | annexe | ecart |
|---|---|---|---|
| hasard, 1er tirage, a moins de 8 m | 14.1 % | 14.1 % | +0.0 |
| hasard, 3 tirages, a moins de 8 m (esperance exacte) | 36.2 % | | |
| frequence seule, 1er, a moins de 8 m | 15.4 % | 15.6 % | -0.2 |
| frequence seule, 3 premiers, a moins de 8 m | 38.3 % | 38.4 % | -0.1 |
| modele, 1er, a moins de 8 m | 44.9 % | 44.9 % | -0.0 |
| modele, 3 premiers, a moins de 8 m | 62.8 % | 62.5 % | +0.3 |
| frequence seule, point exact, 1er | 4.4 % | 4.5 % | -0.1 |
| frequence seule, point exact, 3 premiers | 12.8 % | 12.8 % | -0.0 |
| modele, point exact, 1er | 10.9 % | 11.0 % | -0.1 |
| modele, point exact, 3 premiers | 27.8 % | 27.7 % | +0.1 |

| variable | poids |
|---|---|
| ennemi_le_plus_proche_10m | +1.99 |
| ennemis_a_moins_de_20m | -0.65 |
| equipier_le_plus_proche_10m | -1.08 |
| distance_au_lieu_de_la_mort_10m | +0.15 |
| frequence_log | +0.49 |
| distance_au_barycentre_ennemi_10m | -0.93 |

| carte | n | top1 freq | top1 modele | top3 freq | top3 modele | candidats |
|---|---|---|---|---|---|---|
| Kaiketsu | 886 | 13.7 | 33.6 | 39.2 | 68.2 | 17 |
| Solution | 809 | 14.8 | 31.3 | 39.3 | 65.1 | 19 |
| Dredge | 1097 | 11.4 | 26.7 | 30.7 | 56.5 | 35 |
| Houseki | 972 | 9.7 | 19.5 | 22.5 | 47.9 | 29 |
| Perilous | 711 | 12.1 | 23.3 | 29.4 | 53.7 | 33 |
| Vagabond | 852 | 2.9 | 13.5 | 12.6 | 35.8 | 53 |
| Chasm | 776 | 3.1 | 10.7 | 12.8 | 35.6 | 45 |
| Solitude | 719 | 2.0 | 11.4 | 11.0 | 32.7 | 56 |
| Sylvanus | 921 | 6.4 | 16.5 | 23.3 | 45.0 | 32 |
| Nemesis | 1030 | 2.3 | 12.4 | 10.8 | 30.8 | 51 |
| Dynasty | 1320 | 2.7 | 13.8 | 11.6 | 31.5 | 55 |
| Kiken'na | 1074 | 14.0 | 22.2 | 28.7 | 47.9 | 35 |
| Prism | 755 | 3.1 | 10.7 | 9.5 | 28.5 | 40 |
| Curfew | 1174 | 4.6 | 12.8 | 15.4 | 33.7 | 52 |
| Argyle | 1114 | 3.9 | 14.3 | 18.8 | 36.4 | 35 |
| Goliath | 1034 | 8.3 | 15.2 | 19.3 | 36.4 | 51 |
| Absolution | 1220 | 3.4 | 8.1 | 8.4 | 25.2 | 63 |
| Recharge | 1009 | 3.8 | 10.7 | 12.9 | 29.3 | 57 |
| Detachment | 1321 | 2.3 | 8.3 | 10.4 | 26.3 | 59 |
| The Pit | 1661 | 2.2 | 6.9 | 5.4 | 20.7 | 81 |
| Cliffside | 1002 | 4.2 | 8.8 | 11.4 | 26.4 | 89 |
| Starboard | 1348 | 5.1 | 10.3 | 10.6 | 25.6 | 52 |
| Live Fire | 768 | 5.6 | 9.5 | 13.4 | 28.1 | 46 |
| Salvation | 778 | 4.0 | 11.4 | 18.9 | 32.4 | 41 |
| Catalyst | 1092 | 3.9 | 7.6 | 9.9 | 23.3 | 95 |
| Illusion | 933 | 3.6 | 9.0 | 6.8 | 19.9 | 77 |
| Cliffhanger | 1126 | 2.9 | 7.2 | 5.3 | 17.9 | 86 |
| Shogun | 678 | 7.7 | 11.8 | 21.5 | 33.5 | 41 |
| Empyrean | 945 | 1.3 | 7.3 | 6.7 | 18.6 | 67 |
| Origin | 1117 | 2.5 | 4.7 | 3.8 | 15.3 | 100 |
| Takamanohara | 884 | 2.4 | 7.9 | 9.3 | 20.7 | 68 |
| Elevation | 778 | 3.3 | 6.9 | 10.7 | 21.9 | 65 |
| Streets | 1134 | 1.9 | 5.9 | 7.5 | 18.3 | 76 |
| High Ground | 924 | 2.7 | 7.6 | 7.4 | 18.1 | 71 |
| Domicile | 937 | 4.3 | 6.5 | 7.6 | 17.6 | 80 |
| Bazaar | 1114 | 3.1 | 6.6 | 6.7 | 15.7 | 96 |
| Shiro | 1088 | 1.9 | 7.1 | 8.8 | 17.6 | 58 |
| Isolation | 748 | 4.3 | 8.2 | 10.4 | 18.7 | 63 |
| Snowbound | 1020 | 2.3 | 5.4 | 6.2 | 14.3 | 90 |
| Behemoth | 859 | 0.7 | 4.2 | 3.4 | 11.2 | 129 |
| Fortress | 786 | 2.4 | 5.2 | 7.9 | 15.6 | 85 |
| Forbidden | 1028 | 0.4 | 3.0 | 1.1 | 7.9 | 135 |
| Banished Narrows | 837 | 3.5 | 6.7 | 13.1 | 19.2 | 67 |
| Aquarius | 774 | 1.3 | 3.1 | 3.5 | 8.7 | 91 |
| Smallhalla | 767 | 2.6 | 4.0 | 7.4 | 11.6 | 101 |

##### M-3 par mode (validation croisee par match, 5 plis, cartes vues dans au moins 8 matchs)

Zone a 8 m : un des candidats retenus est a moins de 8 m du point reel. Zone nommee : la zone du catalogue (une zone recoit la somme des probabilites de ses candidats). Couverture : part des reapparitions dont le point reel a une zone.

| mode | matchs | reapparitions | cartes | candidats/carte | a 8 m, 3 premiers : hasard | freq | modele | a 8 m, 1er : freq | modele | zone nommee, 3 premiers : hasard | freq | modele | couverture des zones | exact 1er : modele | exact 3 : modele |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 463 | 39178 | 14 | 82 | 31.1 | 24.1 | 56.3 | 12.6 | 41.7 | 23.0 | 48.0 | 76.4 | 98 % | 7.8 | 21.5 |
| drapeau | 161 | 4359 | 5 | 65 | 34.7 | 38.8 | 55.1 | 19.7 | 38.6 | 25.7 | 47.5 | 72.1 | 96 % | 8.4 | 23.3 |
| bases | 77 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| colline | 48 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| oddball | 7 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| assassin | 550 | 43920 | 45 | 64 | 36.2 | 38.3 | 62.8 | 15.4 | 44.9 | 23.2 | 48.6 | 75.0 | 97 % | 10.9 | 27.8 |

##### M-2 gain de la ligne de vue (7e variable) sur les 6 variables

Cartes dont les artefacts portent la structure : 3 sur 60 lues.
- Streets - Ranked (10323 boites)
- Cliffhanger (10067 boites)
- Streets (10323 boites)


| mode | reapparitions | a 8 m, 3 premiers : 6 variables | 7 variables | gain | zone nommee, 3 premiers : 6 variables | 7 variables | gain | a 8 m, 1er : gain |
|---|---|---|---|---|---|---|---|---|
| fiesta | 6476 | 58.6 | 59.2 | +0.6 | 76.9 | 76.9 | +0.0 | +0.4 |
| assassin | 2260 | 60.4 | 60.7 | +0.2 | 75.7 | 76.2 | +0.5 | +0.5 |
| toutes familles (moyenne ponderee) | 8736 | | | +0.5 | | | +0.2 | +0.5 |

##### M-4 prevision a l'instant de la mort, contre l'instant de la reapparition

Memes evenements (les deux instantanes ont au moins un adversaire), memes candidats ; chaque modele est ajuste sur son propre instant.

| mode | reapparitions | a 8 m, 3 premiers : reapparition | mort | ecart | a 8 m, 1er : reapparition | mort | ecart | zone nommee, 3 premiers : reapparition | mort | ecart |
|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 39149 | 56.3 | 37.7 | -18.5 | 41.7 | 24.7 | -17.0 | 76.4 | 59.2 | -17.2 |
| drapeau | 4356 | 55.0 | 45.7 | -9.3 | 38.5 | 29.8 | -8.7 | 72.1 | 61.0 | -11.1 |
| bases | 6024 | (populations non comparables : 0 / 0) | | | | | | | | |
| colline | 3404 | (populations non comparables : 0 / 0) | | | | | | | | |
| oddball | 720 | (populations non comparables : 0 / 0) | | | | | | | | |
| assassin | 43901 | 62.7 | 46.2 | -16.6 | 44.9 | 29.1 | -15.7 | 75.0 | 59.8 | -15.2 |


#### Seuil de 3 matchs par carte (`--sections m2,m3,m4 --matchs-min-carte 3`)

##### Bilan de lecture

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

##### M-3 par mode (validation croisee par match, 5 plis, cartes vues dans au moins 3 matchs)

Zone a 8 m : un des candidats retenus est a moins de 8 m du point reel. Zone nommee : la zone du catalogue (une zone recoit la somme des probabilites de ses candidats). Couverture : part des reapparitions dont le point reel a une zone.

| mode | matchs | reapparitions | cartes | candidats/carte | a 8 m, 3 premiers : hasard | freq | modele | a 8 m, 1er : freq | modele | zone nommee, 3 premiers : hasard | freq | modele | couverture des zones | exact 1er : modele | exact 3 : modele |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 463 | 39178 | 14 | 82 | 31.1 | 24.1 | 56.3 | 12.6 | 41.7 | 23.0 | 48.0 | 76.4 | 98 % | 7.8 | 21.5 |
| drapeau | 161 | 12948 | 27 | 53 | 37.5 | 41.5 | 57.7 | 18.7 | 39.4 | 28.8 | 48.9 | 71.2 | 98 % | 8.7 | 22.9 |
| bases | 77 | 4058 | 15 | 34 | 41.3 | 44.3 | 67.6 | 19.1 | 48.3 | 26.8 | 41.9 | 69.5 | 99 % | 12.5 | 30.0 |
| colline | 48 | 1307 | 5 | 37 | 37.7 | 38.1 | 64.8 | 16.8 | 46.4 | 21.8 | 39.7 | 67.1 | 100 % | 7.6 | 20.7 |
| oddball | 7 | 0 | 0 | 0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0.0 | 0 % | 0.0 | 0.0 |
| assassin | 550 | 46007 | 50 | 64 | 36.5 | 38.7 | 63.0 | 15.7 | 45.3 | 23.2 | 48.3 | 74.9 | 97 % | 10.8 | 27.6 |

##### M-2 gain de la ligne de vue (7e variable) sur les 6 variables

Cartes dont les artefacts portent la structure : 3 sur 60 lues.
- Cliffhanger (10067 boites)
- Streets (10323 boites)
- Streets - Ranked (10323 boites)


| mode | reapparitions | a 8 m, 3 premiers : 6 variables | 7 variables | gain | zone nommee, 3 premiers : 6 variables | 7 variables | gain | a 8 m, 1er : gain |
|---|---|---|---|---|---|---|---|---|
| fiesta | 6476 | 58.6 | 59.2 | +0.6 | 76.9 | 76.9 | +0.0 | +0.4 |
| assassin | 2260 | 60.4 | 60.7 | +0.2 | 75.7 | 76.2 | +0.5 | +0.5 |
| toutes familles (moyenne ponderee) | 8736 | | | +0.5 | | | +0.2 | +0.5 |

##### M-4 prevision a l'instant de la mort, contre l'instant de la reapparition

Memes evenements (les deux instantanes ont au moins un adversaire), memes candidats ; chaque modele est ajuste sur son propre instant.

| mode | reapparitions | a 8 m, 3 premiers : reapparition | mort | ecart | a 8 m, 1er : reapparition | mort | ecart | zone nommee, 3 premiers : reapparition | mort | ecart |
|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 39149 | 56.3 | 37.7 | -18.5 | 41.7 | 24.7 | -17.0 | 76.4 | 59.2 | -17.2 |
| drapeau | 12942 | 57.7 | 48.2 | -9.5 | 39.4 | 30.3 | -9.1 | 71.2 | 61.6 | -9.6 |
| bases | 4055 | 67.5 | 46.6 | -20.9 | 48.3 | 27.9 | -20.4 | 69.4 | 49.7 | -19.7 |
| colline | 1305 | 64.4 | 38.0 | -26.4 | 46.1 | 23.2 | -22.9 | 67.0 | 48.3 | -18.8 |
| oddball | 720 | (populations non comparables : 0 / 0) | | | | | | | | |
| assassin | 45987 | 63.0 | 46.4 | -16.6 | 45.3 | 29.5 | -15.9 | 74.9 | 59.7 | -15.2 |


---

# Mesures d'origine du lot A (AVANT correction, conservées pour comparaison)

## Conditions (mesures d'origine)

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
