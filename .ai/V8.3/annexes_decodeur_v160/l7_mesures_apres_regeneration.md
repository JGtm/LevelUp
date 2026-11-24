# Mesures APRES la régénération du parc (étape L.7, plan décodeur v1.6.0)

Date : 2026-11-24. Aucun identifiant de joueur, pseudo ni chemin du poste dans ce fichier ; les films se désignent par
les 8 premiers caractères de leur identifiant de match. Les mentions « vérifié » et « déduit » sont posées à chaque
affirmation qui compte.

## 0. Ce qui a été mesuré, et comment

- Parc AVANT : les 1 455 rejeux de la sauvegarde (schéma 125, décodeur v1.4.1). Parc APRES : les 1 455 rejeux régénérés
  (schéma 127, décodeur v1.6.1). Mêmes 1 455 matchs des deux côtés (vérifié : 1 455 identifiants communs, durée et
  nombre d'images identiques pour tous).
- Une passe de lecture seule, un rejeu à la fois, sur les deux dossiers (aucune écriture, aucune base réelle ouverte, aucune
  commande de l'application). Le script d'extraction et les sorties intermédiaires sont dans un dossier de travail hors dépôt (
  les sorties contiennent des identifiants de joueurs).
- Vérité de l'API (score personnel, éliminations, tirs, temps de porteur, temps de jeu, catégorie de mode) : lue dans la
  COPIE de la base partagée d'avant, recopiée dans `l7` et ouverte en lecture seule par un petit programme Go (l'outil
  `duckdb` n'est pas installé). Ces colonnes de l'API ne dépendent pas de la régénération (déduit : la régénération ne
  réécrit que les rejeux et les tables dérivées ; vérifiable en rejouant la lecture sur la base réelle une fois
  l'application arrêtée). La base réelle n'a pas été ouverte : toute mesure qui lit une table dérivée après
  régénération n'est pas prise (section 6).
- Le rejeu ne porte pas le mode : la catégorie de mode vient du registre de la copie de base (vérifié : 1 455 films
  retrouvés au registre).
- Définitions : celles de `avant.md` du plan 2026-11-18 (somme des points publiés contre score personnel ; accord exact des
  compteurs nommés ; à 10 % près pour les tirs ; à 2 s près pour le temps de porteur). Les valeurs « avant » ont été
  RECALCULEES par le même script, pour que le dénominateur soit identique avant et après. Elles retrouvent celles de
  `avant.md` à quelques joueurs près (le script joint les joueurs du tableau du film, `avant.md` joignait aussi le
  roster : 98,41 % contre 98,42 %, 944 rejeux complets, 43 rejeux et 163 joueurs en écart, 22 rejeux à plus de 3 s de
  coup d'envoi, 186 rejeux à drapeau : tous retrouvés exactement).
- Durée de la régénération (vérifié, journal de l'opération) : 1 h 11 pour les 1 455 rejeux, 1 h 42 avec les passes
  dérivées, contre 2 h 26 la fois précédente. Aucun échec, aucune relance.

## 1. Tableau avant / après des mesures du plan

### Piste A : coup d'envoi (A.3)

| Mesure | Avant | Après |
|---|---|---|
| Rejeux datés par l'horloge de manche | 1 455 | 1 455 |
| Rejeux avec `t0FilmMs` | 1 447 (8 sans) | 1 455 (source : horloge pour tous, vérifié) |
| Écart après moins avant, par rejeu (n = 1 447) | sans objet | médiane -0,355 s ; p05 -0,52 s ; p95 -0,29 s ; min -1,47 s ; max +21,4 s |
| Rejeux à moins d'1 s de l'ancienne date | sans objet | 1 422 |
| Rejeux à plus de 3 s de l'ancienne date | 22 en désaccord mouvement/horloge | 22 rejeux changent de 16,2 à 21,4 s : les mêmes 22 (vérifié : même nombre, mêmes ordres de grandeur) |

Le plan annonçait « 0,3 à 0,9 s plus tôt » sur 19 témoins : retrouvé (médiane 0,36 s). Les 22 rejeux où le détecteur
de mouvement et l'horloge divergeaient de 16 à 21 s n'ont plus qu'une seule date : celle de l'horloge (vérifié).

### Piste B : score et tableau (A.3, A.4, A.5 du plan v1.6.0)

| Mesure | Avant | Après |
|---|---|---|
| Rejeux dont le verdict du score est « complet » | 944 | 1 332 |
| Dont HI_1_13_0 / HI_1_12_0 / HI_1_11_0 | 835 / 97 / 12 | 1 178 / 133 / 21 |
| Grande bataille complets | 17 sur 121 | 27 sur 121 |
| Rejeux complets où la somme d'un joueur diffère du tableau | 43 (163 joueurs sur 8 017) | 11 (14 joueurs sur 11 383) |
| Accord exact somme des points / score personnel, rejeux complets | 98,41 % (7 967 / 8 096) | 99,94 % (11 532 / 11 539) |
| Idem, Grande bataille complets | 77,32 % (341 / 441) | 100 % (695 / 695) |
| Idem, build HI_1_11_0 complets | 68,05 % (213 / 313) | 100 % (543 / 543) |
| Accord exact, tous les rejeux avec points publiés | 95,22 % (11 546 / 12 126) | 96,74 % (11 730 / 12 125) |
| Idem, HI_1_11_0 tous | 68,92 % (408 / 592) | 100 % (592 / 592) |
| Idem, HI_1_10_0 / HI_1_9_0 tous | 11,00 % (34 / 309) / 10,45 % (7 / 67) | inchangé : 34 / 309 et 7 / 67 (le verdict les dit incomplets) |
| Joueurs exacts avant devenus inexacts (règle de A.5) | sans objet | 0 (vérifié) ; 185 joueurs deviennent exacts |
| Lignes `scoreActions` ; dont sans identifiant (auteur bot) | 193 650 ; 0 | 194 571 ; 986 |
| Rejeux corrigés du décalage d'arbre (`treeShift`) | compteur absent | 23 rejeux, tous HI_1_11_0 ; 16 perdent 1 à 10 lignes (voir section 3) |
| Rejeux d'Assaut « complets » | 4 des 4 en écart | 0 : les 4 passent « incomplets » (A.4, vérifié) |

Lecture : le verdict « complet » progresse de 388 rejeux (+41 %) sans perte d'exactitude. Les 11 rejeux complets
encore en écart sont répartis sur des variantes variées (Slayer 6, KOTH 1, CTF 2, Bases 1, Grande bataille Fiesta 1),
sans groupe dominant (vérifié).

### Piste C : compteurs nommés (C.4 du plan v1.5.0, rejoué sur le parc)

Accord EXACT avec l'API, joueurs du tableau (14 034), un compteur absent du film comptant 0 (comme `avant.md`) :

| Compteur | Avant | Après |
|---|---|---|
| éliminations | 99,96 % | 99,99 % |
| morts | 99,93 % | 99,97 % |
| assistances | 99,93 % | 99,98 % |
| tirs à la tête | 99,90 % | 100,00 % |
| corps à corps | 99,84 % | 100,00 % |
| grenades | 99,75 % | 100,00 % |
| armes lourdes | 99,98 % | 99,99 % |
| série maximale | 99,77 % | 99,99 % |

Joueurs gagnés (inexact avant, exact après) : éliminations 4, morts 6, assistances 7, tête 14, corps à corps 23,
grenades 35, armes lourdes 2, série 30. **Joueurs perdus : 0 pour chaque compteur** (vérifié). Le plan v1.5.0 chiffrait,
sur 70 films, série +23, grenades +22, corps à corps +10, tête +3 : le parc entier donne plus, sans aucune perte.

Lecture par « exact sur lus » (définition d'`avant.md`, qui ne compte que les joueurs dont le film porte la valeur) :
éliminations 14 022 / 14 024, morts 14 020 / 14 024, assistances 12 502 / 12 505, tête 9 564 / 9 564, corps à corps
9 477 / 9 477, grenades 9 477 / 9 477, armes lourdes 8 563 / 8 564, série 14 031 / 14 032. DEUX LECTURES : le
dénominateur « lus » baisse (grenades 9 507 à 9 477, corps à corps 9 507 à 9 477, tête 9 585 à 9 564) parce que les
valeurs fausses abandonnées par la garde de C.3 deviennent « absentes » au lieu d'être lues ; comparer « exact sur lus »
sans le dire donnerait une impression de perte (24 joueurs pour les assistances, 26 pour le corps à corps)
qui disparaît quand l'absent compte 0 (aucun perdu).

Défauts mesurés :

| Défaut | Avant | Après |
|---|---|---|
| Couples joueur-manche où un compteur nommé dépasse les éliminations | 27 (26 rejeux) | 0 |
| Séries supérieures aux éliminations du joueur | 23 joueurs | 0 |
| Joueurs à exactement 16 grenades | 19 | 0 |

### Piste D : tirs et touches

| Mesure (n = 14 018 joueurs du bilan de tirs joints) | Avant | Après |
|---|---|---|
| Joueurs « suffisants » | 6 465 (46,12 %) | 8 078 (57,63 %) |
| Tirs à 10 % de l'API, tous | 59,73 % | 63,02 % |
| Touches à 10 %, tous | 58,41 % | 61,93 % |
| Dégâts à 10 %, tous | 64,80 % | 68,44 % |
| Tirs / touches / dégâts à 10 %, joueurs « suffisants » | 81,98 / 83,91 / 82,65 % | 79,76 / 80,30 / 78,45 % |
| Tirs / touches / dégâts à 10 %, non suffisants | 40,69 / 36,58 / 49,52 % | 40,25 / 36,94 / 54,83 % |
| Joueurs qui entrent / sortent de la tolérance de 10 % (tirs) | sans objet | 577 entrent, 116 sortent |
| Idem touches ; dégâts | sans objet | 501 / 8 ; 515 / 4 |
| Somme des tirs du bilan, parc | 4 371 764 | 4 045 551 (API sur les mêmes joueurs : 4 634 054) |
| Écart absolu cumulé aux tirs de l'API, par rejeu | 924 466 | 703 787 (812 rejeux plus proches, 151 plus loin de plus de 50) |

Deux lectures de « suffisants » : la part de joueurs suffisants monte de 11,5 points ; la ligne « suffisants » baisse
mécaniquement parce que 1 613 joueurs moins bien couverts entrent dans cette population. À population constante
(tous les joueurs), la précision monte (tableau ci-dessus).

Compteurs de couverture (parc entier) :

| Compteur | Avant | Après |
|---|---|---|
| Tirs lus (`coverage.shots.available`) | 2 673 085 | 3 256 907 (+21,8 %) |
| Tirs attribués (`coverage.shots.attached`, égal aux lignes de `shots`) | 2 619 279 | 3 190 246 (+21,8 %) ; part attribuée 97,99 % puis 97,95 % |
| Tirs sans emplacement | 53 806 | 66 661 |
| Touches publiées (`hits`) | 1 383 586 | 1 447 719 (+4,6 %) |
| Touches avec arme (lignes de `hits` portant `w`) | 1 197 454 (86,6 %) | 1 296 016 (89,5 %) |
| Touches sans tireur (`coverage.hits.noShooter`) | 79 457 | 1 419 |
| Lignes de dégâts (`damage`) | 1 748 892 | 2 453 017 (+40,3 %) |

### Piste E : drapeau (E.4 du plan v1.5.0)

Temps de porteur à 2 s de l'API, joueurs du tableau des rejeux à drapeau :

| Mesure | Avant | Après |
|---|---|---|
| Rejeux à drapeau | 186 | 190 (5 gagnés, 1 perdu : voir section 3) |
| Joueurs comparés | 1 716 | 1 824 |
| Accord à 2 s, tous | 86,07 % | 92,71 % |
| Accord, 8 joueurs ou moins | 89,81 % (736) | 91,71 % (736) |
| Accord, plus de 8 joueurs | 83,27 % (980) | 93,38 % (1 088) |
| Plus de 8 joueurs : film plus court que l'API de plus de 2 s / plus long | 120 / 44 | 36 / 36 |
| Joueurs devenus faux / devenus justes | sans objet | 0 / 143 (règle de Gate B tenue, vérifié) |
| Film `390b1de5` (faux porteur) : plus long temps de porteur ; joueurs à 2 s | 94,3 s ; 19 sur 25 | 46,3 s ; 24 sur 25 |

Définition : `avant.md` comptait 1 750 joueurs (88,11 %) en joignant aussi le roster ; ce tableau joint les joueurs du
tableau du film (1 716, 86,07 %) pour que le dénominateur soit le même des deux côtés. Les deux lectures ont le même
sens de variation. Le but E.4 (au-delà de 8 joueurs, accord au moins égal à celui des petites parties) est atteint :
93,38 % contre 91,71 %.

| Compteur de drapeau | Avant | Après |
|---|---|---|
| Portages publiés (intervalles « porté ») | 6 382 | 6 677 |
| Prises (`captures`) / vols (`steals`) | 2 148 / 2 182 | 2 317 / 2 486 |
| Jauges publiées : intervalles / points | 3 072 / 50 417 | 3 316 / 55 529 |
| Rejeux avec au moins une jauge | 171 | 182 |
| Portages d'équipe inconnue | 128 (12 rejeux) | 309 (17 rejeux) |

Réserve (B.3) : l'équipe du porteur reste inconnue pour 32 portages sur 32 du film `390b1de5` après (25 sur 25 avant) et les portages d'équipe inconnue augmentent (128 à 309) ; la lecture d'équipe du porteur n'est pas obtenue (déduit : le compteur `carrierTeamUnknown` compte ce que B.3 devait réduire ; à vérifier par B.3).

### Piste F : départs (présence)

Identique avant et après (vérifié) : 1 489 partants comparés, écart médian de la présence à l'API -17,15 s, 13,90 % à 5 s,
71,66 % à 30 s. La piste F n'a pas déplacé ces chiffres (le plan ne l'attendait pas : F.5 et F.6 restent des recherches).
Les valeurs d'`avant.md` (1 385 partants, -16,5 s) diffèrent par la convention de jointure, pas par le parc.

### Piste G : zones

| Mesure | Avant | Après |
|---|---|---|
| Rejeux avec `zoneStates` | 125 | 125 |
| `coverage.zones.letters`, somme | 231 (3 pour chacun des 77 rejeux de Bases) | 231 |
| Compteur « lettres en accord avec le rang » (B.6) | absent | toujours absent de `coverage.zones` (aucune clé neuve sous `zones.`) |

B.6 n'est donc pas visible dans le document (vérifié sur les clés de couverture ; si le compteur vit ailleurs, c'est
à indiquer par B.6).

### Piste H : gestes, rechargements, commandes

| Mesure | Avant | Après |
|---|---|---|
| Sauts dérivés / sprints | 655 702 / 1 029 888 | 655 702 / 1 029 891 |
| Accroupissements / glissades (`crouch`, `slide`) ; rejeux concernés | 1 129 / 1 039 ; 7 | 1 129 / 1 039 ; 7 (inchangé : C.1 n'a pas élargi ces deux genres) |
| Commande de déplacement (`moveCommand`) | 0 | 1 758 807 intervalles, 1 450 rejeux |
| Commande de saut (`jumpCommand`) | 0 | 685 139, 1 337 rejeux |
| Commande d'accroupi (`crouchCommand`) | 0 | 191 812, 1 337 rejeux |
| Rechargements publiés | 0 | 501 915 lignes, 1 436 rejeux |
| Gestes (`gestures`) | 0 | 196 056 lignes, 1 330 rejeux |
| Part fermée de la vue de contrôle, par build | 99,94 % (1.13), 99,97 % (1.12), 99,77 % (1.11), 96,13 % (1.10), 90,46 % (1.8) | identique à la décimale (vérifié) |

### Piste I : temps forts de type « mode »

Non mesurée : elle lit la table des temps forts de la base. La valeur « après » n'est pas prise (section 6).

### Piste J : poids et clés racine

| Mesure | Avant | Après |
|---|---|---|
| Poids compressé médian d'un rejeu | 1 105 992 o | 1 635 490 o (+47,9 %) |
| 90e centile / maximum | 1 664 945 / 8 579 753 o | 2 409 635 / 11 422 164 o |
| Total du parc (compressé) | 1,98 Go | 2,84 Go (+43,1 %) |
| Clés racine : médiane (min, max) ; union | 47 (32, 57) ; 69 | 56 (40, 67) ; 79 |

## 2. Agrégats des blocs `coverage`, avant contre après

### 2.1 Rejeux par schéma et par build (vérifié)

Schéma : 1 455 au 125 avant, 1 455 au 127 après. Builds (identiques avant et après) : HI_1_13_0 1 200 ; HI_1_12_0 145 ;
HI_1_11_0 57 ; HI_1_10_0 31 ; HI_1_8_0 13 ; sans build 5 ; HI_1_9_0 3 ; HI_1_4_1 1. Modes (registre) : Assassin 851, Super
Fiesta 449, Grande bataille 121, Husky Raid 15, Ranked 13, Fiesta 3, Other 3.

### 2.2 Replis (`coverage.fallbacks`)

84 noms de repli. Total des occurrences : 10 224 382 avant, 10 306 210 après (+0,8 %). Un seul repli nouveau, trois
disparus.

Dix plus fortes hausses (occurrences ; rejeux concernés avant / après) :

| Repli | Avant | Après | Écart | Rejeux |
|---|---|---|---|---|
| origine_au_sol_lachee_par_fenetre | 274 378 | 342 837 | +68 459 | 1 454 / 1 455 |
| enregistrement_statborg_abandonne | 2 897 546 | 2 921 737 | +24 191 | 1 455 / 1 455 |
| prise_de_socle_premiere_du_cycle | 8 583 | 12 352 | +3 769 | 893 / 922 |
| tireur_par_le_siege_de_la_place (NEUF) | 0 | 960 | +960 | 0 / 37 |
| identite_de_slot_par_residu_de_manche | 51 | 55 | +4 | 12 / 13 |

Les cinq lignes suivantes de la liste des dix ont un écart de 0 : il n'y a que cinq hausses (vérifié).

Dix plus fortes baisses :

| Repli | Avant | Après | Écart | Rejeux |
|---|---|---|---|---|
| identite_premier_occupant_du_siege | 69 850 | 65 156 | -4 694 | 1 449 / 1 449 |
| composants_statborg_arretes | 7 452 | 2 904 | -4 548 | 1 326 / 862 |
| emission_hors_domaine_jetee | 11 133 | 8 458 | -2 675 | 1 126 / 850 |
| tourelle_porteur_voisin_de_slot (DISPARU) | 2 588 | 0 | -2 588 | 191 / 0 |
| visee_du_tir_lue_dans_le_record | 7 909 | 7 546 | -363 | 543 / 496 |
| pistes_du_monde_apres_la_marche | 1 567 886 | 1 567 600 | -286 | 1 440 / 1 440 |
| ancrage_bipede_apres_la_marche | 836 569 | 836 320 | -249 | 152 / 128 |
| chaine_evenement_code_non_modelise | 104 661 | 104 613 | -48 | 97 / 63 |
| localisation_largeur_libre | 75 392 | 75 345 | -47 | 77 / 60 |
| nombre_drapeaux_hors_catalogue_sans_passage (DISPARU) | 19 | 0 | -19 | 1 / 0 |

Disparus : `index_drapeau_zero_pour_tous` (19, un rejeu), `nombre_drapeaux_hors_catalogue_sans_passage` (19, le même
rejeu), `tourelle_porteur_voisin_de_slot` (2 588 sur 191 rejeux). Les deux premiers sont liés à la seule régression
d'un rejeu à drapeau (section 3, point 1).

Coverage : 923 feuilles numériques ; 188 neuves, 4 disparues (`t0Film.tracks`, `t0Film.moving`, `t0Film.burst`, liées au retrait du détecteur de mouvement de A.1, et `flagCarries.filmBaseContradict`, 3 occurrences). Les plus fortes baisses relatives des feuilles communes sont des
compteurs de défauts : `flagCarries.seatsBeyondBand` 1 892 à 0, `scoreActions.noRecipient` 986 à 0,
`shots.unitOtherIndex` 25 902 à 2, `scoreActions.noTarget` 1 765 à 12, `hits.noShooter` 79 457 à 1 419,
`scoreActions.unconfirmedRank` 742 à 464.

### 2.3 Éliminations

| Compteur | Avant | Après |
|---|---|---|
| Éliminations du tableau (somme des éliminations des joueurs) | 143 639 | 143 646 |
| Calque `eliminations` : lignes ; rejeux | absent | 144 119 ; 1 448 |
| Dont avec victime nommée | sans objet | 122 519 |
| Dont d'auteur bot (ligne sans identifiant de joueur) | sans objet | 473 |
| Lignes de `scoreActions` sans identifiant (auteur bot) | 0 | 986 |
| Entrées de roster sans identifiant (bots) ; rejeux avec bots | 1 124 ; 682 | 1 124 ; 682 |

Le calque d'éliminations couvre 1 448 rejeux sur 1 455 : les 7 absents sont les 5 rejeux sans build, le seul HI_1_4_1,
et un HI_1_13_0 dont le tableau n'a aucune élimination (vérifié, `5da6fd30` : zéro dans les deux parcs).

### 2.4 Calques neufs (rejeux avec au moins une ligne ; total de lignes)

| Calque | Rejeux | Lignes |
|---|---|---|
| `chatter` (répliques) | 1 436 | 397 197 |
| `announcements` (annonces) | 1 436 | 256 249 |
| `gestures` | 1 330 | 196 056 |
| `vitals` (santé et bouclier) | 1 455 | 3 130 766 |
| `drawnWeapons` | 1 455 | 715 012 |
| `ammoReads` | 1 455 | 4 473 400 |
| `attachments` | 1 216 | 125 004 |
| `roster[].appearance` | 1 450 | 14 346 (une par joueur du roster) |
| Total des huit | | 9 308 030 |
| Aussi neufs : `eliminations` ; `reloads` ; `padCandidates` | 1 448 ; 1 436 ; 846 | 144 119 ; 501 915 ; 5 112 |

Listes qui grossissent : `stances` 1 753 162 à 4 388 923 (nouveaux genres de commande), `shots` +21,8 %, `damage` +40 %,
`padPickups` 50 927 à 68 394, `weaponPads` 11 521 à 15 302, `objectives` 47 878 à 52 010, `bursts` 26 649 à 27 180.
Listes au même nombre de lignes (vérifié) : pistes (qui gagnent des échantillons à pleine cadence), charges, équipements, armes au sol, poses, médailles, véhicules,
projectiles, grenades, lectures de grenades.

### 2.5 Répliques nommées par build

| Build | Rejeux | Avec répliques | Répliques lues | Nommées | Part nommée |
|---|---|---|---|---|---|
| HI_1_13_0 | 1 200 | 1 200 | 305 652 | 297 136 | 97,21 % |
| HI_1_12_0 | 145 | 145 | 38 811 | 34 127 | 87,93 % |
| HI_1_11_0 | 57 | 57 | 35 325 | 0 | 0 % |
| HI_1_10_0 | 31 | 31 | 16 207 | 0 | 0 % |
| HI_1_9_0 | 3 | 3 | 1 202 | 0 | 0 % |
| HI_1_8_0, HI_1_4_1, sans build | 19 | 0 | 0 | 0 | sans objet |

## 3. Régressions

Méthode : pour chaque rejeu et chaque compteur de lecture utile, avant moins après ; seuil « nette » = baisse de plus de
10 %. Toutes les listes du document ont aussi été balayées (nombre de rejeux en baisse de plus de 10 %, nombre qui
tombent à zéro).

Résultat d'ensemble (vérifié) : aucun rejeu ne perd de tirs lus, de tirs attribués, de touches, de touches avec arme,
d'éliminations, de points de piste, de pistes, de rafales, de médailles, de poses ni d'armes au sol (listes vides : 0
rejeu en baisse). Les tableaux « 20 pires » de ces compteurs sont donc vides. Le balayage de toutes les listes ne trouve
qu'un rejeu qui tombe à zéro (point 1).

### 3.1 Les quatre groupes qui baissent

**1. Un rejeu perd tout le calque de drapeau (le seul vrai recul).**

| Film | Build | Mode | Joueurs | Portages avant | Portages après |
|---|---|---|---|---|---|
| be758198 | HI_1_11_0 | Grande bataille (Fiesta CTF) | 24 | 19 | 0 |

Avant : calque de drapeau publié (19 portages, 48 vies d'objet, 237 lectures de jauge). Après : `flagFilm` faux, aucun
calque ; la couverture compte pourtant 5 prises et 8 vols (contre 1 et 1 avant). Les replis `index_drapeau_zero_pour_tous`
(19) et `nombre_drapeaux_hors_catalogue_sans_passage` (19) du rejeu avant ont disparu avec le calque. Contre l'API :
temps de porteur à 2 s pour 19 joueurs sur 24 avant, 17 après ; somme du film 34 s avant, 0 s après, API 142 s : le
calque d'avant était déjà très en deçà de l'API, et ce film n'a perdu que deux joueurs à 2 s. Hypothèse (déduit) : ce
film ne tenait son calque que par les deux replis maintenant retirés (peut-être le décodeur refuse-t-il désormais un
calque qui repose sur un indice de drapeau « zéro pour tous »). Pour le vérifier : rejouer ce seul film en lisant le détail de refus du
calque de drapeau ; décider si la perte est l'effet voulu du retrait des replis (alors à inscrire au registre des
changements de sortie) ou un oubli. Contre-poids : 5 autres rejeux gagnent un calque de drapeau (`3efe4592` Fiesta
1.12, `43880965`, `fd8ac070` Grande bataille 1.11, `9b3b9c1a`, `b4035a67` Grande bataille 1.10), avec 8, 29, 15, 37 et 26 portages (vérifié).

**2. Deux rejeux de CTF d'Assassin perdent un portage chacun.**

| Film | Build | Mode | Joueurs | Portages avant | Portages après |
|---|---|---|---|---|---|
| 6a711138 | HI_1_13_0 | Assassin | 10 | 9 | 8 |
| 70a1c6c6 | HI_1_13_0 | Assassin | 9 | 34 | 33 |

Dans les deux, les vols passent de 8 à 6 et de 13 à 10 et le nouveau compteur `gaugeReadsOffDrops` vaut 8 et 29.
Hypothèse (déduit) : la lecture de jauge écarte maintenant les lectures hors d'un lâcher connu, et un portage court
s'appuyait sur elles. Mesure contre l'API : à vérifier sur ces deux films (le temps de porteur des joueurs n'est pas
dégradé globalement : 0 joueur devenu faux sur les 1 824).

**3. Seize rejeux de Grande bataille HI_1_11_0 perdent de 1 à 10 lignes de `scoreActions` (de -0,4 % à -3,7 %).**

Exemples : `9dfd3c24` 273 à 263, `9ac5f43c` 316 à 307, `6b5de1d7` 264 à 256, `e5adf7b2` 268 à 261. Les 16 ont un
`treeShift` positif (de 98 à 213 lectures). Cause (vérifié) : c'est la correction du décalage de l'arbre de scripts
du build 1.11 (A.3) : des lignes mal attribuées sont retirées ; l'accord somme des points contre score personnel
passe de 12 / 25, 14 / 25, 15 / 27, 18 / 26, 15 / 25, 20 / 26 et 21 / 31 joueurs à tous les joueurs de ces films
(25 / 25, 25 / 25, 27 / 27, 26 / 26, 25 / 25, 26 / 26, 31 / 31). Ce n'est pas une régression de lecture, c'est le résultat voulu.

**4. Le bilan de tirs par joueur (`shotStats`) baisse sur 250 rejeux, pour l'essentiel en se rapprochant de l'API.**

La somme des tirs du bilan passe de 4,37 M à 4,05 M (API : 4,63 M) pendant que les tirs attribués montent de 21,8 %.
Vingt plus fortes baisses (tirs du bilan, somme des joueurs ; l'API sert de témoin) :

| Film | Build | Mode | Joueurs | Avant | Après | API |
|---|---|---|---|---|---|---|
| 01e1f945 | HI_1_13_0 | Assassin | 9 | 22 683 | 2 717 | 2 896 |
| e721b517 | HI_1_13_0 | Assassin | 12 | 22 689 | 4 774 | 3 339 |
| 05765a11 | HI_1_13_0 | Assassin | 8 | 12 739 | 4 037 | 2 980 |
| e6e4ba0b | HI_1_13_0 | Assassin | 9 | 12 067 | 4 135 | 4 043 |
| f3061ab7 | HI_1_13_0 | Assassin | 8 | 10 404 | 2 474 | 2 432 |
| 21ece4d8 | HI_1_13_0 | Assassin | 8 | 10 666 | 4 525 | 3 135 |
| 35f5cdea | HI_1_13_0 | Super Fiesta | 17 | 7 860 | 2 404 | 4 222 |
| 4db2574e | HI_1_12_0 | Grande bataille | 29 | 9 920 | 4 833 | 5 425 |
| bff0c899 | HI_1_12_0 | Grande bataille | 27 | 11 019 | 6 016 | 8 082 |
| 3abf0258 | HI_1_10_0 | Grande bataille | 32 | 18 053 | 13 183 | 15 036 |
| 47854796 | HI_1_13_0 | Assassin | 9 | 8 864 | 4 256 | 3 633 |
| 7064ffa7 | HI_1_13_0 | Assassin | 8 | 7 304 | 2 704 | 2 632 |
| 9d293550 | HI_1_11_0 | Grande bataille | 26 | 11 630 | 7 246 | 9 703 |
| d7d3c50b | HI_1_12_0 | Assassin | 15 | 4 964 | 614 | 596 |
| 760fb768 | HI_1_13_0 | Assassin | 11 | 7 292 | 3 204 | 3 214 |
| 8171b141 | HI_1_13_0 | Assassin | 8 | 6 190 | 2 607 | 3 140 |
| a517c2db | HI_1_13_0 | Assassin | 8 | 7 284 | 3 701 | 3 763 |
| 25392c3e | HI_1_13_0 | Super Fiesta | 10 | 5 096 | 1 514 | 2 260 |
| cfb85a58 | HI_1_13_0 | Assassin | 10 | 6 656 | 3 074 | 3 006 |
| a853258c | HI_1_13_0 | Assassin | 10 | 4 940 | 1 614 | 1 591 |

Lecture (vérifié sur le témoin de l'API) : dans la plupart des cas l'avant sur-estimait nettement (jusqu'à 8 fois l'API)
et l'après s'en rapproche. Mais pour sept films de la liste (35f5cdea, 4db2574e, bff0c899, 3abf0258, 9d293550, 25392c3e, 8171b141) l'après passe sous l'API : l'estimation est devenue prudente. À l'inverse, 151 rejeux s'éloignent de l'API de plus de 50 tirs ; les dix pires :
`084a804d` (Grande bataille 1.10 : 15 406 puis 17 999, API 15 342), `e0c325d9` (1.10 : 13 672 puis 15 614, API 13 871),
`c95535e9` (1.11 : 13 916 puis 12 417, API 14 440), `b71d39db` (Super Fiesta 1.13 : 3 337 puis 2 057, API 3 412),
`64e32547` (1.11 : 15 430 puis 14 157, API 17 152), `d22a5577` (1.11 : 10 760 puis 8 487, API 10 166), `e1bdb97f`
(Super Fiesta 1.13 : 2 865 puis 1 844, API 3 525), `c6250266` (Fiesta 1.13 : 4 313 puis 3 302, API 4 645),
`dd29957c` (1.10 : 8 149 puis 10 216, API 8 684), `be758198` (1.11 : 12 454 puis 11 474, API 13 478).
Hypothèse (déduit) : le bilan par joueur utilise une estimation par la couverture des messages ; elle change de signe selon
le build (trop haute en 1.10, trop basse en 1.11 et en Super Fiesta). Le rapport « tirs du bilan sur lignes `shots` » passe de
1,67 à 1,27. Pour le vérifier : comparer, sur ces films, `shotStats.coverage` à la part de messages non lus. Au niveau joueur : 19 joueurs seulement perdent plus de 10 %
de touches du bilan (exemple : 267 à 50 sur un joueur de `0d0bc019`, 55 à 9 sur `0f9550e5`).

### 3.2 Hypothèse « trames arrêtées sur les builds 1.9 et 1.10 » : vérifiée, mais ce n'est pas une régression

Les nouveaux genres (répliques, annonces, gestes, rechargements) héritent des trames arrêtées du balayage : 69 rejeux
ont au moins une trame arrêtée, 174 696 trames (0,39 % des 45,36 M de trames du parc ; part médiane 1,9 % par rejeu,
maximum 70,7 %). Répartition : les 31 rejeux 1.10, les 13 rejeux 1.8, les 3 rejeux 1.9, les 5 sans build, le 1.4.1,
12 des 57 rejeux 1.11, 3 des 145 rejeux 1.12 et 1 rejeu 1.13 (vérifié). Le total est quasi identique à celui d'avant (175 234 trames, mesuré par `scoreActions`), mais le nombre de rejeux touchés passe de 219 à 69. Pas de baisse de lecture due à
ces arrêts ; ils bornent seulement la couverture des calques neufs sur ces builds. Gestes : lus mais retenus par la règle
de validation (`gated`) sur 1.11 (16 529 lectures, 0 publiée), 1.9 (752, 0), 1.10 (7 887 lues retenues, 1 751 publiées sur 4 rejeux) : calque
absent des 57 rejeux 1.11 par décision de la règle, pas par perte.

## 4. Poids

Poids compressé d'un rejeu, par catégorie (octets ; médiane, 90e centile, maximum ; variation du total) :

| Catégorie | Rejeux | Médiane avant | Médiane après | P90 avant | P90 après | Max avant | Max après | Total |
|---|---|---|---|---|---|---|---|---|
| Tous | 1 455 | 1 105 992 | 1 635 490 | 1 664 945 | 2 409 635 | 8 579 753 | 11 422 164 | +43,1 % |
| Assassin | 851 | 1 121 373 | 1 661 802 | 1 472 787 | 2 180 778 | 3 219 796 | 4 598 861 | +48,1 % |
| Super Fiesta | 449 | 1 015 562 | 1 509 314 | 1 295 393 | 1 900 889 | 1 652 476 | 2 317 002 | +47,9 % |
| Grande bataille | 121 | 3 977 588 | 5 236 679 | 7 162 805 | 9 464 000 | 8 579 753 | 11 422 164 | +29,8 % |
| Husky Raid | 15 | 515 856 | 783 769 | 1 350 959 | 2 118 034 | 1 955 432 | 2 849 332 | +52,5 % |
| Ranked | 13 | 1 133 576 | 1 692 576 | 1 664 945 | 2 472 694 | 1 759 499 | 2 572 182 | +47,9 % |
| 8 joueurs ou moins | 647 | 1 096 959 | 1 625 224 | 1 450 383 | 2 123 405 | 5 909 807 | 7 492 480 | +47,7 % |
| 9 à 16 joueurs | 688 | 1 065 155 | 1 583 393 | 1 409 351 | 2 081 275 | 2 106 336 | 3 111 200 | +47,8 % |
| Plus de 16 joueurs | 120 | 3 974 660 | 5 235 711 | 7 162 805 | 9 464 000 | 8 579 753 | 11 422 164 | +30,1 % |

Rapport après sur avant, par rejeu : médiane 1,48 ; de 1,18 à 1,59 (vérifié). Fiesta (3) et Other (3) : +44 % et +48 %.
Les builds anciens (1.8 à 1.11) grossissent de 27 à 34 %, les builds récents de 43 à 48 %.

Origine (vérifié sur un échantillon de 49 rejeux, un sur trente, en JSON non compressé : 367 Mo avant, 543 Mo après) :
la liste `tracks` fait à elle seule 74 % de l'augmentation (déduit : les échantillons à pleine cadence, champ `fullRate`, sont neufs ; `tracks` passe de 286 Mo à 417 Mo sur l'échantillon),
puis `projectiles` 7,1 %, `ammoReads` 5,6 %, `vitals` 3,1 %, `stances` 3,0 %, `damage` 1,8 %, `shots` 1,4 %, le roster
(apparence) 0,8 %, `chatter` 0,8 %. Le dossier des rejeux sur disque (rasters compris) grossit de 39,8 % (journal de
l'opération). Gates de poids du plan (piste C et piste D : sous 1 % de plus ; piste H : sous 2 % de plus) : elles se
mesuraient sur 20 films et par lot ; le cumul sur le parc est +47,9 % de poids médian. À dire à l'utilisateur : le gain de poids est
un coût cumulé, pas une dérive d'un lot (déduit : les gates par lot ne voient pas l'addition).

## 5. Anomalies visibles dans les rejeux d'après

Toutes les anomalies ci-dessous existent à l'identique avant (vérifié, sauf mention), aucune n'est créée par la
régénération :

- **Rejeux sans roster : 5** (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`), tous sans build : les 5 rejeux sans build de F.7. Ils
  portent aussi toutes les « vies sans identité » : 1 466 vies sans nom sur 156 847 (0,93 %), toutes dans ces 5 rejeux (vérifié).
  Leur verdict de pont : « non publiable : aucun pont » (5 sur 5).
- **Rejeu sans piste : 0. Durée nulle : 0. Zéro image : 0** (vérifié).
- **Pistes sans identifiant de joueur : 4 071 pistes dans 506 rejeux** (bots et vies sans identité de joueur), identiques avant.
- **Rejeux sans tir : 1** (`3efe4592`, HI_1_12_0), sans touche : 20 (1.4.1, 1.8, et un HI_1_13_0), identiques avant.
- **Calque attendu absent** :
  - Répliques, annonces, rechargements, touches : absents des 19 rejeux HI_1_8_0, HI_1_4_1 et sans build ; les
    calques de vitalité, d'arme tirée et de munitions sont présents sur les 1 455 rejeux, l'apparence sur les 1 450 qui ont un roster (vérifié).
  - Éliminations : absentes de 7 rejeux (5 sans build, 1.4.1, et un 1.13 sans élimination au tableau).
  - Gestes : 1 330 rejeux sur 1 455 ; absents de tous les 1.11 (57) et 1.9 (3) par la règle de validation, de 27 des 31 rejeux
    1.10, de 13 des 145 rejeux 1.12 et de 6 des 1 200 rejeux 1.13, alors que la couverture a lu des gestes dans ces rejeux (105 rejeux lus mais non publiés,
    29 160 lectures retenues). À confirmer avec le plan si ces rejeux devaient en porter (déduit : la règle de validation les écarte).
  - Pièces jointes (`attachments`) : absentes de 197 rejeux 1.13 sur 1 200 et de 39 rejeux 1.12 sur 145 : probablement des
    matchs sans pièce attachée lue (déduit ; à vérifier sur la couverture `projectiles.attach`).
  - Noms des répliques : 0 % sur les builds 1.9 à 1.11 (52 734 répliques lues, aucune nommée), 88 % en 1.12, 97 % en 1.13 :
    le nom ne vient que des tables de réplique des deux builds récents.
- **Compteur de lettres de zones (B.6)** : pas de compteur d'accord dans le document (section 1, piste G).
- **Lectures de munitions : 275 rejeux** portent `ammoReads.deltaRefused` (delta refusé) ; 1 337 rejeux sont à cadence pleine
  pour les commandes.

## 6. Ce qui n'a pas pu être mesuré

- Toute mesure qui lit une table de la base APRES : temps forts « mode » (piste I, F.4), coup d'envoi et temps de jeu
  recalculés en base (le journal de l'opération dit : 1 466 matchs changent, écart médian -355 ms ; vérifié dans le
  journal, pas relu en base), journal des morts, vies et placement. Base réelle non ouverte, par consigne.
- Les mesures contre l'API utilisent la copie d'avant de la base : déduit identique pour les colonnes de l'API.
- Vérités absentes : apparence enregistrée contre la personnalisation publique (E.4 du plan v1.6.0 : aucune colonne d'apparence
  dans la copie ; les 14 346 apparences sont publiées pour 1 450 rejeux, sans comparaison possible), répliques contre le
  témoin (D.1, D.2), temps de VIP (E.2).
- Non repris : accord des tirs par arme dominante (le tableau d'`avant.md` par arme), les 16 portages « au socle » faux de
  B.1 (le compteur n'existe pas dans la couverture), la comparaison ligne à ligne du registre des changements de sortie (Gate L : le
  registre n'a pas été croisé ici), les vérifications à l'écran.
- Le poids par couche n'est mesuré que sur 49 rejeux (un sur trente) ; la catégorie « mode » vient du registre de la copie d'avant.
