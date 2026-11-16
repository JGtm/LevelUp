# Réapparitions lues dans le film : mesure sur le parc régénéré (2026-11-16)

Plan : `PLAN_REAPPARITIONS_2026-11-15.md`, item A.8. Suite de `mesures_lecture_film.md` (première lecture sur cinq films)
et de `mesures_lot_A.md` (M-1 à M-4 sur le repli, parc au schéma 116). Les tableaux ci-dessous sont les sorties de
`levelup spawn-model evaluate`, recopiées sans retouche (hors mise en forme du titre).

## Conditions

- Parc : `data/cache/replays/halo_infinite/` du dossier principal, régénéré au schéma 124 (décodeur v1.3.9) :
  1 455 artefacts `.json.gz` au schéma 124 et 1 artefact `.json` resté au schéma 107 (1 456 artefacts listés). Les six
  familles de modes de `spawn_families.toml` en couvrent 1 306 ; 150 artefacts sont hors famille (autres modes), non lus
  par `evaluate`.
- Binaire compilé depuis le worktree détaché `C:\t\reapp-a8` (base `release/v8.3.0`, commit `094693dbe5`) avec
  `GOCACHE=%LOCALAPPDATA%\go-build-reapp-a8`, CGO.
- Accès aux données, vérifié sur pièces : `config.ResolveRoots` (`internal/config/config_roots.go`) lit l'environnement ;
  `LEVELUP_REPO_ROOT` pose la racine de ressources (ici le worktree : `config/titles/`, `spawn_families.toml`,
  référentiel), `LEVELUP_DATA_DIR` pose la racine de données (ici le dossier `data` principal). `PathResolver` en tire
  `ReplayArtifactPath` et `SharedDBPath` sous la racine de données (la liste de 1 456 artefacts de la sortie le confirme),
  alors que `SpawnModelPath` (fichier de poids) reste sous la racine de ressources, donc dans le worktree. Le réglage
  `film_archive_dir` n'intervient pas : la commande ne lit que les artefacts, jamais un film.
- Seule base ouverte : `shared_matches_v2.duckdb`, par `duckdb.OpenReadForQuery` (`cmd_spawn_model.go`,
  `donneesDuParc`), pour un seul `SELECT` sur `match_registry` (carte, variante, catégorie). Aucun serveur ne la tenait
  (rien n'écoutait sur le port 8000) : l'ouverture s'est faite en lecture seule, le fichier n'a pas été modifié
  (dernière écriture vue avant les passes : 16 novembre 11:41). Aucun `fit`, aucune écriture, aucune cuisson.
- Un seul processus lourd à la fois. Mémoire de la commande : environ 100 à 650 Mo (maximum relevé 646 Mo), loin du
  plafond de 4 Go. Lecture du parc : 2 min 43 s ; mesures M-1 à M-4 : environ 10 min en tout par passe.

### Commandes exactes

```
set GOCACHE=%LOCALAPPDATA%\go-build-reapp-a8
cd C:\t\reapp-a8\apps\go-api
go build -o <dossier-temporaire>\levelup.exe ./cmd/levelup

set LEVELUP_REPO_ROOT=C:\t\reapp-a8
set LEVELUP_DATA_DIR=C:\Users\Guillaume\Downloads\Scripts\LevelUp\data
levelup.exe spawn-model evaluate --par-carte                                   # seuil de 8 matchs par carte (M-1 a M-4)
levelup.exe spawn-model evaluate --sections m2,m3,m4 --matchs-min-carte 3      # familles clairsemees (Bases, Colline)
```

Ce sont les deux passes de `mesures_lot_A.md` (mêmes sections, mêmes seuils), de sorte que les tableaux se comparent
ligne à ligne. Le bilan de lecture est identique dans les deux passes.

## Bilan de lecture

| artefacts | lus | sans registre | hors familles | illisibles |
|---|---|---|---|---|
| 1456 | 1306 | 0 | 150 | 0 |

| reapparitions (source film) | reapparitions (repli ecart 4-15 s) | debuts de manche | sans equipe | non finies |
|---|---|---|---|---|
| 111365 | 91 | 110 | 211 | 0 |

| documents lus au film | accord film et repli | film seul | repli seul | dont debut de manche au film | comptes a rebours sans joueur |
|---|---|---|---|---|---|
| 1305 | 110952 | 413 | 45 | 36 | 4 |

- assassin : 550 matchs lus ; bases : 77 ; colline : 48 ; drapeau : 161 ; fiesta : 463 ; oddball : 7.

### Lecture des chiffres

- Un document sur 1 306 ne porte aucune lecture du film : c'est l'artefact resté au schéma 107. Il décide au repli, ce
  qui donne les 91 réapparitions de source `repli_ecart` ; les 111 365 autres viennent du film.
- Le film lit 111 365 réapparitions. Le repli sur l'écart de 4 à 15 s en retrouve 110 952, soit 99,63 %, et n'en ajoute
  que 45 (0,04 %) ; l'accord sur l'union des deux définitions (111 410 pistes) est de 99,59 %.
- Film seul (413, 0,37 % des lectures du film) : réapparitions que le repli rejette. Un échantillon de 710 documents
  (voir plus bas) les situe presque toutes à plus de 15 s de la vie précédente (256 sur 270), le reste à moins de 4 s :
  la fenêtre de 4 à 15 s est trop étroite pour les attentes longues, pas trop large.
- Repli seul (45) : 36 sont des débuts de manche que le film lit comme des suppressions suivies d'une vie (80 %), 9 ne le
  sont pas (vies sans fin lue, le film n'a pas de preuve et n'en invente pas). Le repli n'ajoute donc presque que des
  débuts de manche, comme sur les cinq films.
- Comptes à rebours sans joueur : 4 sur tout le parc (0 sur les cinq films) ; ils sont écartés et comptés.
- Réapparitions sans équipe : 211, le même nombre qu'au repli (schéma 116) : la jointure d'équipe ne dépend pas de la
  source de décision.

### Comparaison à la première lecture sur cinq films

| | cinq films | parc (1 305 documents lus au film) |
|---|---|---|
| réapparitions lues au film | 645 | 111 365 |
| accord film / repli | 641 (99,4 % des lectures du film) | 110 952 (99,63 %) |
| film seul | 4 (0,6 %) | 413 (0,37 %) |
| repli seul | 7, toutes des débuts de manche | 45, dont 36 débuts de manche (80 %) |
| comptes à rebours sans joueur | 0 | 4 |

La première lecture se confirme à l'échelle : l'accord monte (99,4 % à 99,6 %), le repli ne retrouve pas la totalité du
film mais ne le contredit presque jamais, et son seul apport propre reste les débuts de manche.

### Comparaison au repli sur le parc au schéma 116 (lot A)

| | parc 116 (repli) | parc 124 (film) |
|---|---|---|
| artefacts lus (six familles) | 1 306 | 1 306 |
| réapparitions | 111 088 | 111 456 (111 365 film + 91 repli) |
| sans équipe | 211 | 211 |

## M-1 témoin : Assassin hors Fiesta et BTB (voie du film)

44006 réapparitions sur 45 cartes (annexe : 43966 sur 45), hors candidats : 510

| mesure | depot | annexe | ecart |
|---|---|---|---|
| hasard, 1er tirage, a moins de 8 m | 14.1 % | 14.1 % | +0.0 |
| hasard, 3 tirages, a moins de 8 m (esperance exacte) | 36.2 % | | |
| frequence seule, 1er, a moins de 8 m | 15.5 % | 15.6 % | -0.1 |
| frequence seule, 3 premiers, a moins de 8 m | 38.3 % | 38.4 % | -0.1 |
| modele, 1er, a moins de 8 m | 44.9 % | 44.9 % | +0.0 |
| modele, 3 premiers, a moins de 8 m | 62.7 % | 62.5 % | +0.2 |
| frequence seule, point exact, 1er | 4.4 % | 4.5 % | -0.1 |
| frequence seule, point exact, 3 premiers | 12.8 % | 12.8 % | -0.0 |
| modele, point exact, 1er | 11.0 % | 11.0 % | -0.0 |
| modele, point exact, 3 premiers | 27.8 % | 27.7 % | +0.1 |

| variable | poids |
|---|---|
| ennemi_le_plus_proche_10m | +1.98 |
| ennemis_a_moins_de_20m | -0.65 |
| equipier_le_plus_proche_10m | -1.08 |
| distance_au_lieu_de_la_mort_10m | +0.15 |
| frequence_log | +0.49 |
| distance_au_barycentre_ennemi_10m | -0.93 |

Le témoin tient à 0,2 point près (critère : 1 point), comme sur le repli (écart maximal 0,3). Les poids sont ceux du lot A à
0,01 près. Le tableau par carte (45 lignes, `--par-carte`) est dans la sortie complète ; comparé à celui du repli, le
modèle y bouge de 0,6 point au plus par carte (1er et 3 premiers) ; la fréquence seule bouge davantage sur Solitude
(1er : 2,0 -> 3,7), carte dont le nombre de réapparitions passe de 719 à 725.

## M-3 par mode, zone nommée, 3 premiers (voie du film)

Seuil de 8 matchs par carte (`--par-carte`) :

| mode | matchs | réapparitions | cartes | à 8 m, 3 premiers : hasard | fréquence | modèle | zone nommée : hasard | fréquence | modèle | couverture des zones |
|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 452 | 39413 | 14 | 31.1 | 23.9 | 56.1 | 23.0 | 47.9 | **76.2** | 98 % |
| drapeau | 46 | 4370 | 5 | 34.7 | 38.8 | 54.9 | 25.7 | 47.4 | **72.2** | 96 % |
| bases | 0 | 0 | 0 | | | | | | non mesurable | |
| colline | 0 | 0 | 0 | | | | | | non mesurable | |
| oddball | 0 | 0 | 0 | | | | | | non mesurable | |
| assassin | 517 | 44006 | 45 | 36.2 | 38.3 | 62.7 | 23.1 | 48.6 | **74.9** | 97 % |

Seuil de 3 matchs par carte (`--sections m2,m3,m4 --matchs-min-carte 3`), qui ouvre Bases et Colline :

| mode | matchs | réapparitions | cartes | à 8 m, 3 premiers : hasard | fréquence | modèle | zone nommée : hasard | fréquence | modèle | couverture des zones |
|---|---|---|---|---|---|---|---|---|---|---|
| fiesta | 452 | 39413 | 14 | 31.1 | 23.9 | 56.1 | 23.0 | 47.9 | **76.2** | 98 % |
| drapeau | 150 | 12966 | 27 | 37.5 | 41.3 | 57.7 | 28.8 | 48.9 | **71.2** | 98 % |
| bases | 55 | 4073 | 15 | 41.4 | 44.6 | 67.6 | 26.6 | 42.0 | **69.8** | 99 % |
| colline | 18 | 1308 | 5 | 37.7 | 38.1 | 64.8 | 21.8 | 39.7 | **67.0** | 100 % |
| oddball | 0 | 0 | 0 | | | | | | non mesurable | |
| assassin | 542 | 46098 | 50 | 36.5 | 38.8 | 63.0 | 23.1 | 48.4 | **74.9** | 97 % |

Sorties complètes (colonnes « à 8 m, 1er », « exact 1er » et « exact 3 ») : mêmes colonnes que `mesures_lot_A.md`.

### Écart au repli (zone nommée, modèle, 3 premiers, seuil de 3 matchs)

| mode | repli (parc 116) | film (parc 124) |
|---|---|---|
| assassin | 74,9 | 74,9 |
| fiesta | 76,4 | 76,2 |
| drapeau | 71,2 | 71,2 |
| bases | 69,5 | 69,8 |
| colline | 67,1 | 67,0 |

Les écarts sont d'au plus 0,3 point : la voie du film ne change pas la valeur du modèle, elle change l'origine de la
décision (preuve lue au lieu d'un écart de temps).

### Règle D-2 (au moins 50 % dans les 3 premières zones)

À l'instant de la réapparition, tous les modes mesurables passent largement (67,0 % à 76,2 %). **La liste des modes
couverts ne change pas** : Assassin, Fiesta, Drapeau, Bases, Roi de la colline ; Oddball (7 matchs, aucune carte
mesurable) reste hors mesure, donc non couvert. À l'instant de la MORT (M-4, ce que le rejeu afficherait pendant le
compte à rebours), les deux familles que le lot A avait relevées restent juste sous 50 % : Bases 49,8 % et Roi de la
colline 48,2 % (49,7 et 48,3 au repli). Assassin (59,7 %), Fiesta (59,1 %) et Drapeau (61,7 %) tiennent. Le choix de
retenir ou non Bases et Colline pour le calque du rejeu est une décision du user ; rien n'est changé ici.

## M-2 gain de la ligne de vue (7e variable)

Cartes dont les artefacts portent la structure : 3 sur 60 lues (Cliffhanger, Streets, Streets - Ranked ; 30 713 boîtes).

| mode | réapparitions | à 8 m, 3 premiers : 6 variables | 7 variables | gain | zone nommée : 6 variables | 7 variables | gain | à 8 m, 1er : gain |
|---|---|---|---|---|---|---|---|---|
| fiesta | 6524 | 58.3 | 59.0 | +0.7 | 76.5 | 76.4 | -0.1 | +0.2 |
| assassin | 2264 | 60.5 | 60.6 | +0.2 | 75.7 | 76.1 | +0.5 | +0.4 |
| toutes familles (moyenne pondérée) | 8788 | | | +0.6 | | | +0.0 | +0.2 |

Gain de 0,6 point au plus, loin des 3 points de D-4 : la ligne de vue reste écartée. La mesure ne porte toujours que sur
3 cartes.

## M-4 prévision à l'instant de la mort, contre l'instant de la réapparition

Mêmes événements (les deux instantanés ont au moins un adversaire), mêmes candidats ; chaque modèle est ajusté sur son
propre instant. Seuil de 8 matchs par carte pour Fiesta, Drapeau (premier jeu de lignes) et Assassin, seuil de 3 pour
Drapeau, Bases et Colline :

| mode | réapparitions | à 8 m, 3 premiers : réapparition | mort | écart | zone nommée, 3 premiers : réapparition | mort | écart |
|---|---|---|---|---|---|---|---|
| fiesta | 39384 | 56.2 | 37.7 | -18.4 | 76.2 | 59.1 | -17.1 |
| assassin | 43987 | 62.7 | 46.2 | -16.6 | 75.0 | 59.9 | -15.1 |
| drapeau (seuil 8) | 4368 | 55.0 | 45.8 | -9.2 | 72.1 | 61.1 | -11.0 |
| drapeau (seuil 3) | 12962 | 57.7 | 48.3 | -9.4 | 71.2 | 61.7 | -9.5 |
| bases (seuil 3) | 4070 | 67.5 | 46.5 | -21.1 | 69.8 | 49.8 | -20.0 |
| colline (seuil 3) | 1306 | 64.4 | 38.1 | -26.3 | 67.0 | 48.2 | -18.8 |

Au repli (`mesures_lot_A.md`) : Fiesta -18,5, Assassin -16,6, Drapeau -9,3 / -9,5, Bases -20,9, Colline -26,4. Aucune
valeur ne bouge de plus de 0,3 point. Oddball : populations non comparables (0 / 0).

## Les suppressions (`endReason: deleted`) non suivies d'une vie du même joueur (découverte du plan, section 8)

Méthode : sonde jetable (programme Go temporaire dans le worktree, lecture seule des artefacts, retiré ensuite, non
versionné), mêmes fonctions que la commande : `EntreeOccupationDeLArtefact` puis `NouvelIndexVies(...).Spawns()`. Elle
regroupe les pistes nommées par joueur, ordonne les vies par début, et pour chaque piste close par `deleted` cherche une
vie suivante du même joueur. Elle porte sur les 1 455 artefacts du schéma 124, tous modes confondus (pas seulement les six
familles). La grappe d'une suppression est le nombre de suppressions nommées du même document à moins de 30 images (environ
une seconde) d'elle.

| compte | valeur |
|---|---|
| pistes nommées | 154 242 : lâchées 147 239, supprimées 1 248, sans fin lue 5 755 |
| pistes sans xuid | 4 071, dont 36 supprimées |
| `coverage.tracks.lifeEnds` cumulés | lâchées 149 552, supprimées 1 245, reprises 39, non lues 1 649, ouvertes 357, fin du film 5 471, suppressions non prouvées 105 |
| suppressions nommées | 1 248 |
| suivies d'une vie du même joueur | 329 (26 %) |
| non suivies | 919 (74 %) : 887 en cours de match, sur 538 films, et 32 dans les 5 dernières secondes du document |
| débuts de manche posés par la règle | 251 ; aucun sans suppression préalable. L'écart avec 329 : 78 suppressions coïncident avec un début de compte à rebours dans la tolérance, la mort l'emporte |

Écart entre la suppression et la vie suivante (329 suppressions suivies) : moins de 1 s : 54 ; 1 à 4 s : 117 ; 4 à 15 s : 142 ;
15 à 60 s : 6 ; plus de 60 s : 10.

Taille de la grappe des 329 suppressions suivies : seule dans sa seconde : 112 (34 %) ; par deux : 30 ; par trois : 36 ;
quatre ou plus : 151 (46 %, jusqu'à 15 joueurs dans la même seconde).

Lecture :

- Les 919 suppressions sans vie suivante (dont 887 en cours de match) n'ont aucun effet sur la règle : sans vie suivante,
  rien n'est marqué. Elles sont fréquentes (538 films sur 1 455 en portent au moins une en cours de match), mais ce sont
  des départs, des corps remplacés ou des slots repris, pas des fins de manche. La mesure de la section 8 (3 sur cinq
  films) se confirme et s'élargit sans conséquence.
- Parmi les 329 suppressions suivies d'une vie, 151 sont collectives (quatre joueurs ou plus dans la même seconde : une
  vraie fin de manche), 66 sont des petits groupes (deux ou trois) et 112 sont isolées (remise en place d'un joueur, retour
  après départ). La règle les lit toutes comme un début de manche. **Avant d'afficher un « début de manche », le libellé
  n'est exact que pour les grappes collectives** (environ la moitié) ; pour les isolées il faudrait un autre nom ou ne rien
  dire. C'est un choix d'affichage à trancher, pas un défaut du modèle : ces vies ne sont de toute façon ni des
  réapparitions ni des entrées du modèle.
- Rien n'est modifié dans le code pour autant (hors périmètre).

### Échantillon partiel : écart des désaccords à la vie précédente

La même sonde, étendue pour mesurer l'accord film / repli par écart de temps, a lu 710 artefacts sur 1 455 (les
premiers par ordre de nom, tous modes) avant que le dossier de données ne disparaisse de la machine pendant la passe
(voir ci-dessous). Chiffres de l'échantillon, à lire comme indicatifs :

| | valeur |
|---|---|
| accord / film seul / repli seul | 68 232 / 270 / 54 |
| film seul : écart à la vie précédente | moins de 4 s : 14 ; plus de 15 s : 256 |
| repli seul : écart à la vie précédente | 4 à 15 s : 54 (dont 36 débuts de manche) |

Le repli seul est par construction dans la fenêtre 4 à 15 s ; le film seul est quasi tout hors de la fenêtre, du côté des
attentes longues.

## Incident constaté pendant la mesure

Les trois passes de la commande (bilan, M-1 à M-4 aux deux seuils) sont terminées avant l'incident ; seule la seconde
exécution de la sonde est partielle. Vers 17 h 33 le 16 novembre, pendant cette exécution, le contenu de
`C:\Users\Guillaume\Downloads\Scripts\LevelUp\data\` a disparu (dossier présent mais vide : plus de `cache`, `titles`,
`auth`, `desktop`, base partagée). Les lectures de la sonde ont alors échoué (745 artefacts « fichier introuvable »).
Cette session n'écrit nulle part sous `data` (lecture seule : artefacts, une base ouverte par `OpenReadForQuery`) ; au
moment du constat, deux commandes `git worktree remove --force` d'une autre session tournaient (sur `LevelUp-wt-deps`, puis
`LevelUp-wt-navmesh`), et le dossier principal avait été repositionné sur le commit `d6af3229ae` à 17 h 31. La cause n'a
pas été établie ici.
