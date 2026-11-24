# Inventaire des copies du contributeur (lecture seule)

Date de la lecture : 2026-11-23. Méthode : `gh api` seul (arbres git récursifs, contenus décodés en base64). Aucun clone, aucune écriture hors de ce fichier, aucun commit, aucun push.

Légende : **[LU]** = vérifié sur le contenu ou l'arbre lus ce jour. **[SUPPOSÉ]** = déduit du nom d'un fichier, d'un README ou d'un en-tête, non vérifié ligne à ligne.

## 0. Sources et limites

| Élément | Valeur | Statut |
|---|---|---|
| Dépôt du contributeur | `llampwall/installation-08`, branche `main`, rendue publique | [LU] |
| Tête lue | `a66b791460d2f7567fb34d55622acbc027e689b2` (même tête que dans le plan) | [LU] |
| Arbres lus | non tronqués (`truncated = false`) | [LU] |
| Décodeur public, tag v1.5.0 | `dc967c27f1330de41d6b581e45603de1fcb244a1` | [LU] |
| Décodeur public, départ de la copie v1.3.6 | `5d63c673d476a2565e83c2eaccd04cff07341b94` | [LU] |
| Cartes public, dernier tag v1.4.0 | `9913d0b353b2be82bf720ad36063729ca316a6f7` | [LU] |
| Cartes public, départ de la copie v1.0.1 | `95547a36d894da59770ebf9f35ab038e299a3312` | [LU] |
| Lecteur public, dernier tag v1.2.0 | `a9b53a799c1bb13ffbbe185cf62488f5c705c1d4` | [LU] |
| Lecteur public, départ de la copie v1.0.1 | `01ec0fdcf6931774fa7fd25b2350dc3e34409078` | [LU] |

Méthode de comparaison : par chemin, puis par empreinte git de chaque fichier (SHA du blob). Deux fichiers de même taille mais de contenu différent sont comptés comme modifiés. Pour le décodeur, la taille seule en aurait manqué 7 sur 249.

Limites : le contenu des fichiers modifiés n'a pas été lu (sauf les trois documents et 15 en-têtes). Les noms de fichiers et les en-têtes donnent le sujet, pas la preuve du fonctionnement.

Hors périmètre, non inventoriés : `third_party/basisu` (114 fichiers), `meshoptimizer` (7), `webview_go` (16), `ww2ogg` (2).

## 1. Les trois documents de changements du contributeur

Tous les trois ont été lus en entier. [LU]

**`third_party/README.md`** (57 Ko, 621 lignes). Il décrit l'état des copies :
- Décodeur : « part de la v1.3.6 » ; liste de capacités ajoutées (pistes à pleine cadence, visées seules, direction des tirs par la visée, tireur par place de contrôle, manche de déplacement, postures de la vue de contrôle, arme en main, dégâts par place, causes des morts, gestes, munitions portées, codec `REPLAYINPUTS30`, socles d'équipement, seconde famille du Stalker Rifle, table des scores cibles par variante). Il ajoute aussi, à partir de l'étape 113, les projectiles (hauteur, pleine cadence, collés), la détonation d'un lancer sans vol publié, et les candidats de socle.
- Il indique que les messages de journal et les erreurs ont été traduits du français vers l'anglais. Cette traduction ne remonte pas dans notre dépôt (le plan le dit déjà).
- Cartes : « v1.0.1 modifiée », export du monde en GLB (117 fichiers neufs et 15 modifiés selon le README), décodeur d'images `bitmapdecode`, matériaux, émissifs, styles Forge, terrain, décor, armes, effets de particules.
- Lecteur : « v1.0.1 modifiée », lecture des compagnons `.module_hd1`, dix fichiers pour le maillage du monde, `mapvar` (mode Forge, axes des socles, éclairage, styles).
- Ajoute une dépendance `github.com/woozymasta/bcn` v0.7.0 (cartes).
- Incohérences du README relevées : son en-tête annonce le schéma document 125 et faits 23, alors que ses entrées plus récentes et la copie sont à 130 et 25. Son décompte « 117 neufs / 15 modifiés » pour les cartes ne correspond pas au décompte mesuré (voir section 3), probablement parce qu'il ne compte que la collecte du monde. [SUPPOSÉ]

**`third_party/REBASE-v1.3.6.md`** (6 Ko). Inventaire de rebase : trois familles de changements locaux (facades de cache, faits et gardes de révision, entrée des observations, types publics, capacités de rejeu). Pour chaque famille, la liste des fichiers et une colonne « l'amont a-t-il touché les mêmes fichiers ? ». Les fichiers touchés aussi par l'amont sont peu nombreux ; la plupart des fichiers locaux sont « non ». [LU]

**`third_party/REBASE-v1.3.6-verification.md`** (4 Ko). Vérification d'une reconstruction sur un film : l'ancien et le nouveau exécutable donnent les mêmes premiers éléments pour 30 tableaux. Il reste 3 810 écarts champ à champ (schéma 122 à 125, libellés de révision, quatre compteurs de visée, deux échantillons de visée en moins). L'auteur note qu'un seul film ne suffit pas à prouver que chaque échantillon retiré est invalide. [LU]

## 2. Décodeur (film-decoder)

### 2.1 Version de départ [LU]

- La copie part de la **v1.3.6** (le README et l'inventaire de rebase le disent).
- Constantes relevées sur les pièces :

| Constante | v1.3.6 (départ) | Copie du contributeur | v1.5.0 (public) |
|---|---|---|---|
| Schéma du document (`SchemaVersion`) | 115 (`replay/document.go`) | 130 (`replay/document_schema.go`) | 126 (`replay/document_schema.go`) |
| Schéma des faits (`SchemaDesFaits`) | 22 (`replay/filmfacts_fichier.go`) | 25 (`replay/filmfacts_fichier.go`) | 30 (`replay/filmfacts_fichier.go`) |

### 2.2 Compteurs de fichiers [LU]

Copie : 2 607 fichiers. Décodeur v1.3.6 : 2 534 fichiers ; v1.5.0 : 2 629 fichiers.

**Ce que le contributeur a changé lui-même (copie contre v1.3.6)** :
- Fichiers ajoutés : **73** (`replay` 47, `internal/grammar` 17, `internal/facts/fallback` 4, `types` 2, `decfilm` 1, `filmcache` 1, `internal/grammar/lecture` 1).
- Fichiers modifiés : **249** (taille différente pour 242). Par dossier, les plus touchés : `replay` 138, `internal/grammar` 61, `revision` 8, `filmcache` 4, `internal/facts/fallback` 3, `internal/facts/killsource` 3, `internal/facts/objectives` 3, `internal/source` 3.
- Fichiers supprimés par rapport à la v1.3.6 : 0.

**Copie contre la v1.5.0 (mélange des deux sources d'écart)** :
- Fichiers présents dans la copie et absents de la v1.5.0 : **186**. Sur ces 186 : **68** sont des ajouts du contributeur (ni en v1.3.6, ni en v1.5.0). Les **118** autres étaient en v1.3.6 et ont été retirés ou renommés par l'amont (surtout des tests de recherche, par exemple `internal/facts/killsource/*_research_test.go`, `e191c_*`, `ecart108_*`).
- Fichiers présents dans la v1.5.0 et absents de la copie : **208** (ajouts de l'amont depuis la v1.3.6 : `replay` 71, `replay/testdata` 56, `internal/grammar` 35, `internal/facts/objectives` 14, `internal/grammar/signaux` 5, `types` 5, etc.).
- Fichiers de même chemin mais de contenu différent : 420 (taille différente 410). Ce chiffre mélange l'évolution de l'amont et les changements du contributeur : il n'est pas exploitable seul.

Parmi les 186 ajoutés contre la v1.5.0 : **153 sont des fichiers de test** (dont 132 portent « research » dans leur nom) et **33 sont du code de production**.

Les 33 fichiers de production ajoutés contre la v1.5.0 [LU] :
`decfilm/catalog_bytes.go`, `filmcache/source_from_file.go`, `internal/facts/fallback/english_names.go`, `internal/facts/fallback/registre_replay_postures.go`, `internal/grammar/actions_du_film.go`, `internal/grammar/boutons_de_controle.go`, `internal/grammar/jeu_darmes_desire.go`, `internal/grammar/lecture/actions.go`, `internal/grammar/projectile_attaches.go`, `internal/grammar/type1_datums.go`, `internal/grammar/vue_a_actions.go`, `replay/ammo_reads.go`, `replay/build_postures.go`, `replay/document_actions_joueur.go`, `replay/document_gestes.go`, `replay/filmfacts_actions.go`, `replay/filmfacts_attaches.go`, `replay/filmfacts_boutons.go`, `replay/filmfacts_complement.go`, `replay/filmfacts_lancers.go`, `replay/filmfacts_schema23.go`, `replay/grenades_detonation.go`, `replay/places_de_controle.go`, `replay/postures_cinematique.go`, `replay/postures_de_controle.go`, `replay/postures_de_controle_vies.go`, `replay/projectiles_attache.go`, `replay/score_target_table.go`, `replay/stick_de_deplacement.go`, `replay/tracks_plein_debit.go`, `replay/tracks_visees_seules.go`, `types/grammar_actions.go`, `types/grammar_boutons.go`.

### 2.3 Les 15 plus gros fichiers ajoutés (contre la v1.5.0)

En-tête lu : 40 premières lignes. [LU]

| # | Fichier | Taille (o) | Ce qu'il apporte |
|---|---|---|---|
| 1 | `internal/grammar/f0_103_contexte_research_test.go` | 17 606 | Instrument de recherche (balise `research`). Mesure si le type 103 (objet d'équipement posé) désigne la vie de l'objet : déployé ou lâché. Ne publie rien. |
| 2 | `internal/grammar/e191c_prefixe_objet_research_test.go` | 16 635 | Recherche (balise `research`) sur la fermeture des cinq types d'objets du monde (37, 38, 41, 42, 43). Ne publie rien. |
| 3 | `internal/grammar/f0_103_verdicts_research_test.go` | 15 906 | Recherche (balise `research`) : verdicts sur les références du type 103. Ne publie rien. |
| 4 | `internal/grammar/mouvement_5_3_2b_research_test.go` | 14 792 | Recherche (balise `research`) : quel composant porte l'accroupissement image par image, et la vitesse du sprint en mètres par seconde. Ne publie rien. |
| 5 | `replay/projectiles_attaches_research_test.go` | 14 606 | Recherche (balise `research`) : calibration de la pose des projectiles collés. Verdict écrit dans l'en-tête : aucune correspondance simple, rien n'est livré. |
| 6 | `replay/score_koth_emissions_research_test.go` | 14 492 | Recherche (balise `research`) : émissions brutes du score du mode KOTH, par équipe. Lit les faits, ne décode aucun film. |
| 7 | `replay/ctf_research_test.go` | 14 469 | Recherche (balise `research`) : tirs perdus en CTF, ventilés en sous-causes. Ne modifie rien. |
| 8 | `internal/grammar/mouvement_5_3_2d_trame_research_test.go` | 14 411 | Recherche (balise `research`) : même question que le n° 4, avec le vrai décodeur de trame. |
| 9 | `internal/grammar/lot1_visee_nonmodale_research_test.go` | 14 236 | Recherche (balise `research`) : décodage de la visée des tirs qui touchent (cas non modal). L'en-tête dit que l'oracle de validation n'est pas concluant. |
| 10 | `internal/grammar/type1_datums.go` | 13 895 | **Production**. Lit le bloc de type 1 (table de datums, environ 343 Ko, placé avant chaque image-clé), que le décodeur ignorait. Interne au chargement de l'état ; l'en-tête ne dit pas qu'il publie un champ. |
| 11 | `internal/grammar/mouvement_5_9_sprint_research_test.go` | 13 628 | Recherche (balise `research`) : vérifie par le contenu que la fente 1 est le sprint. |
| 12 | `replay/directions_tir_research_test.go` | 13 603 | Recherche (balise `research`) : ce que représente la direction des tirs (cubemap), comparée à la visée du tireur. |
| 13 | `internal/grammar/ecart108_research_test.go` | 13 577 | Recherche (balise `research`) : écarts de 108 bits entre entrées d'image-clé. Ne publie rien. |
| 14 | `replay/h3_aquarius_reperes_research_test.go` | 13 461 | Recherche (balise `research`) : cause des écarts de repères sur deux films. Verdict écrit : l'oracle de mesure était en cause, pas le catalogue. |
| 15 | `internal/facts/fallback/english_names.go` | 13 385 | **Production**. Noms anglais des replis, pour les journaux seulement. Le nom publié (`coverage.fallbacks[]`) ne change pas. |

Sur ces 15, 13 portent bien la balise de compilation `research` (lu dans l'en-tête). Pour les 132 fichiers « research » du total, seul le nom a été vérifié, pas la balise.

### 2.4 Canaux neufs apportés par la copie [SUPPOSÉ sauf mention]

Ce que la copie publie et que la v1.5.0 ne publie pas sous le même nom :
- **Pistes à pleine cadence** (`tracks[].fullRate`, schéma 109 selon l'en-tête de `replay/document_tracks.go` de la copie). [LU] La v1.5.0 n'a aucun `FullRate` dans `replay/document.go`, `replay/tracks_publication.go` et `replay/document_chronicle.go` (les trois fichiers de publication lus). Canal neuf probable pour nous, à confirmer sur tout l'arbre de la v1.5.0.
- **Projectiles collés et détonations** : `replay/projectiles_attache.go` (champ `attach`), `replay/grenades_detonation.go` (champ `deto`). Aucun fichier de même nom côté v1.5.0. `Projectile.Z` et `Projectile.FullRate` sont ajoutés dans `replay/projectiles.go`, qui existe aussi côté v1.5.0 (fichier commun, modifié des deux côtés).
- **Gestes du joueur** (rechargement, lancer, coup de corps) : `replay/document_gestes.go`, liste d'événements avec un genre par geste. Voir le recouvrement ci-dessous.
- **Munitions portées** : `replay/ammo_reads.go`. Aucun fichier de même nom côté v1.5.0.
- **Table des scores cibles par variante** : `replay/score_target_table.go`. Le README dit lui-même que cette table est « par variante » (règle R-2 du dépôt). [LU]
- **Codec `REPLAYINPUTS30`** et **entrée de bloc de type 1** : `internal/grammar/type1_datums.go`, `replay/filmfacts_*.go`.
- **Mesure de la carte de quantification** : `decfilm/catalog_bytes.go` (lecture des bornes depuis des octets embarqués).

## 3. Cartes (`halo-infinite-map-images`)

### 3.1 Version de départ et compteurs [LU]

- Départ : **v1.0.1** (le README et l'arbre concordent). Copie : 320 fichiers.
- Contre v1.0.1 : **195 ajoutés** (`himap` 191, `bitmapdecode` 4) ; **27 modifiés** (`himap` 24, racine 3) ; **1 absent** (`himap/testdata/carte_validee_v1.png`, image de jeu exclue par la politique du dépôt, dite par le README).
- Contre le dernier tag v1.4.0 : **57 absents de la copie** (`himap` 56, `himap/testdata` 1) ; 33 modifiés ; 195 ajoutés. Les 57 absents sont des ajouts de l'amont entre v1.0.1 et v1.4.0.

### 3.2 Canaux neufs et recouvrements [SUPPOSÉ]

Ce que la copie ajoute et que la v1.4.0 n'a pas sous le même nom : export du monde en GLB (`glb_*.go`), décorateurs d'herbe (`decorateurs*.go`), émissifs et animations d'émissifs (`emissif*.go`), décodeur d'images `bitmapdecode`, effets de particules d'arme (`effet_arme.go`, `modele_particule.go`), normales des maillages (`geometry_normales.go`), lentille (`lens_flare.go`), styles Forge (`styles_forge.go`, `couches_style.go`, `forge_teintes.go`).

Recouvrements possibles avec la v1.4.0 : la v1.4.0 a `relief*.go` (fond Relief), `navmesh`, `zones_forge.go`, `cartes_forge.go`. Le sujet « Forge » est donc commun (styles de la copie contre zones et cartes Forge de l'amont). Non examiné ligne à ligne.

## 4. Lecteur de fichiers (`halo-infinite-files-reader`)

### 4.1 Version de départ et compteurs [LU]

- Départ : **v1.0.1**. Copie : 92 fichiers.
- Contre v1.0.1 : **6 ajoutés** (`mapvar/echelle_test.go`, `mapvar/forge_styles_test.go`, `mapvar/lighting_test.go`, `mapvar/representation_test.go`, `himodule/hd1_test.go`, `ooz/LICENSE`) ; **20 modifiés** (`hinavmesh` 7, `mapvar` 6, `himodule` 5, `ooz` 2, racine 0) ; **5 absents** (fixtures de jeu omises : `mapvar/testdata` 3, `hinavmesh/testdata` 2).
- Contre le dernier tag v1.2.0 : 21 modifiés, 13 absents (dont 6 fichiers `hinavmesh` ajoutés par l'amont entre v1.0.1 et v1.2.0).

### 4.2 Recouvrements [SUPPOSÉ]

Ajouts de la copie : éclairage des variantes (`lighting_test.go`), styles Forge (`forge_styles_test.go`), représentations (`representation_test.go`), échelle (`echelle_test.go`), lecture des compagnons `.module_hd1` (`hd1_test.go`). La v1.2.0 n'a aucun fichier de même nom pour ces sujets, mais le thème « Forge » recoupe les travaux de la brique des cartes (section 3.2).

## 5. Recouvrements avec la v1.5.0 (décodeur)

Ce que la v1.5.0 a déjà, par rapport à ce que la copie apporte. [LU] sur les fichiers cités.

| Sujet | Côté v1.5.0 (lu) | Côté copie | Lecture |
|---|---|---|---|
| Rechargements | `replay/document_reloads.go`, champ `reloads[]` (`replay/document.go`, l. 296), `coverage.reloads` | `replay/document_gestes.go` : le rechargement est un geste (genre 38, message `weapon_reload`) dans une liste d'événements. Pas de `document_reloads.go`, pas de champ `reloads`. | Même message source, forme différente. Recouvrement direct, à trancher. |
| Commande de déplacement | `replay/document_stances_commandes.go` : `stances[]` de genre `moveCommand` (commande tenue, en intervalles) | `replay/stick_de_deplacement.go` : deux scalaires bruts par vie, au débit du film. `replay/document_stances.go` existe des deux côtés. | Même sujet (le manche), forme différente : intervalles tenus contre valeurs brutes. |
| Éliminations | `replay/document_eliminations.go`, `replay/eliminations.go`, `coverage.eliminations` | Aucun fichier de production ajouté sur ce sujet. Les tests `killsource` ajoutés sont des instruments de recherche. | Pas de recouvrement de code de production. |
| Coup d'envoi par l'horloge | `replay/match_clock.go` (horloge match ↔ frame), `replay/document.go` (pointeur `originMs`) | `replay/match_clock.go` et `replay/document.go` existent aussi (commentaires identiques en tête). Contenu différent, non examiné. | Sujet présent des deux côtés, pas de fichier neuf du contributeur. |
| Tirs par place, « tirs à tout rang » | `replay/tirs_par_place.go`, `replay/tirs_index_fiable.go`, `internal/facts/fallback/registre_replay_places.go` | `replay/tirs_par_place.go` (commun, modifié), `replay/places_de_controle.go` (ajouté), `replay/tirs_par_place_test.go` (commun) | Fichier commun modifié des deux côtés. À comparer. |
| Visées seules, direction des tirs | `replay/document_aim.go`, `internal/grammar/offline_aim_only.go`, `visees_lues.go`, `replay/shots_visee.go` | `replay/tracks_visees_seules.go` (ajouté), `replay/shots_visee.go` (commun), directions de tir (recherche) | `shots_visee.go` existe des deux côtés : confirme le plan. |
| Tir continu | `replay/tir_continu_armes.go`, `replay/filmfacts_tir_continu.go`, `internal/grammar/tir_continu.go` | `internal/grammar/tir_continu.go` (commun, modifié) | Commun, à comparer. |
| Pistes à pleine cadence | absent de `document.go`, `tracks_publication.go`, `document_chronicle.go` | `FullRate` (`replay/document_tracks.go`) | Canal neuf probable (voir 2.4). |
| Projectiles collés, détonation | aucun fichier de même nom | `projectiles_attache.go`, `grenades_detonation.go` | Canal neuf d'après les noms. |

## 6. Ce qui reste à trancher ou à vérifier

- Les 425 fichiers modifiés (décodeur 249 contre v1.3.6, cartes 27, lecteur 20, et leurs recouvrements) ne sont pas lus contenu contre contenu. Les traductions de messages sont annoncées par le README, non vérifiées.
- L'ordre des tags entre les trois briques n'est pas traité ici (il relève de l'inventaire complet du plan).
- Les numéros de schéma du README sont périmés ; la copie est à 130 (document) et 25 (faits).
- Le total « 117 neufs / 15 modifiés » du README pour les cartes diffère du décompte mesuré (195 / 27 contre v1.0.1). Lequel est juste pour l'étendue réelle : non tranché.
