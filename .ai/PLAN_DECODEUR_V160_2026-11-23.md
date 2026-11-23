# PLAN — Décodeur de film v1.6.0 : vitesse, backlog, canaux du lecteur 3D (2026-11-23)

**État : décisions tranchées et GO de l'utilisateur donnés le 2026-11-23 ; le coût de chaque vague est
annoncé à son lancement.** (User, 2026-11-23 : « Tu as mon feu vert d'emblée. C'est bien de découper les
morceaux en plus petit, ça permet de paralléliser et d'utiliser des modèles et des efforts moins coûteux en
général. »)

Statut : **écrit le 2026-11-23 ; relu le même jour contre la grille `plan-review` par un second agent
(« à reprendre sur quelques points avant le go ») ; corrections B1 à B4 et I1 à I13 appliquées le 2026-11-23.
Commandé par le user le 2026-11-22 (« La v1.5.0 est publiée. Le user te confie la v1.6.0 du décodeur. Avant
tout code, écris un plan propre et fais-le relire contre la grille plan-review »). Décisions DD-1 à DD-8
TOUTES TRANCHÉES par le user le 2026-11-23 (section 2). Le contributeur reporte lui-même ses ajouts ; tout
se fait dans la v1.6.0, sans échéance. GO D'EXÉCUTION donné le 2026-11-23.**

**État au 2026-11-23 (soir) : vitesse intégrée (parallélisme compris), pistes A, B, E et F rendues ou statuées, gestes, répliques (lecture) et canaux du contributeur PORTÉS PAR NOUS et intégrés en six intégrations (tête `7bda46f`, document au schéma 127, faits au schéma 31). Restent : D.2 en cours, D.4 à D.6, E.4, R.*, L.* ; V4 et V2 abandonnées.**
Contrat d'exécution : skill `plan-execution`, complément `levelup-plan`. Statuts : `[x]` fait, `[~]` couvert
ailleurs (référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture.
Plans liés : `.ai/PLAN_DECODEUR_2026-11-18.md` (v1.5.0, dont ce plan reprend la forme et les règles),
`.ai/PLAN_REJEU_3D_2026-11-15.md` (décision D-18), `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`.

Abréviations : `DEC` = dépôt `JGtm/halo-infinite-film-decoder` (PUBLIC), lu pour l'instruction dans
`Scripts/dec-v150` (branche `feat/v1.5.0`, même arbre que le tag `v1.5.0`,
`dc967c2`) ; `LU` = ce dépôt ; `HOTE` = le dossier principal de LevelUp (films, bases : LECTURE SEULE) ;
« le contributeur » = l'auteur du lecteur 3D, dépôt public `llampwall/installation-08` (AGPL-3.0) ;
« le parseur voisin » = `halo_api` de nuzzles (Rust, MIT ou Apache-2.0) ; « témoin » = libfilmshell de Den ;
« niveau de fils » = le nombre de fils que le décodeur a le droit de lancer (1 = comportement actuel).

## 0. Objectif et critère de succès

Objectif : publier UNE version v1.6.0 du décodeur qui (1) se mesure en vitesse par une commande et un relevé
rangé avec la version, (2) lit plus vite par les méthodes reprises du parseur voisin sans changer un octet
du rejeu, (3) traite chaque entrée du backlog qui concerne le décodeur, (4) accueille les canaux du lecteur
3D, que le contributeur reporte lui-même sur la v1.5.0 ; puis UNE montée dans LevelUp et UNE régénération du
parc, avant le premier démarrage de LevelUp 8.3.0. Aucune échéance : la version attend le contributeur, et
rien n'est renvoyé à une version suivante par manque de temps.

Critère de succès :
- un relevé de vitesse v1.5.0 et un relevé v1.6.0 existent, produits par la même commande sur le même jeu de
  films, rangés dans le dépôt et dans les notes de version ;
- pour chaque piste de vitesse, les 20 films de l'équivalence sont identiques, à chaque niveau de fils, à la
  référence définie pour elle (section 3, « Preuve d'identité ») ; le gate du corpus (19 témoins), lancé au
  niveau de fils maximal, ne montre ni gain, ni perte, ni décision neuve ;
- chaque entrée de la section 1.4 a un sort écrit (fait, couvert, écarté, ou abandon motivé) ;
- aucun symbole public retiré, aucune signature changée (prouvé par commande) ; aucun repli neuf ;
- v1.6.0 taguée après relecture adversariale, PUIS équivalence et gate du corpus finals ; LevelUp monté, CI
  verte ; parc régénéré une fois.

| Piste | Ce que le user verra |
|---|---|
| V0 | Un tableau de vitesse par version du décodeur (chargement, décompression, décodage, publication, total, pic de mémoire) |
| V1, V2, V3, V4 | Des rejeux générés plus vite, à sortie identique ; le gain se lit dans le tableau |
| A | Grande bataille et Assaut : le détail daté des points n'est plus dit « complet » à tort ; le recul d'une pénalité apparaît sur la courbe |
| B | Drapeau : moins d'états « au socle » faux ; porteurs au-delà de 8 joueurs si la condition est tenue |
| C | Fiche du joueur : accroupissements, glissades et sauts au bouton si la lecture est prouvée |
| D | Les répliques des Spartans existent dans le rejeu (qui parle, quand, quelle réplique) ; une maquette des trois idées d'affichage |
| E | Éliminations dont un bot est l'auteur ; temps et série en tant que VIP ; nom des options de variante |
| F | Rien (contrôles, ménage) |
| P | Rien dans la 2D ; le lecteur 3D reçoit ses canaux (pleine cadence, visée, arme en main, munitions, gestes, hauteur des projectiles), écrits par le contributeur, vérifiés et intégrés par nous |

Version : v1.6.0 (ajouts seulement : version mineure). Schéma du document : 126 vers 127, UNE montée pour
tout ce qui change la sortie, nos pistes A à F et les canaux du contributeur. Les pistes V ne touchent pas
au schéma.

Hors périmètre : tout ce qui touche à la justesse des tirs, des touches et de la précision par arme (DD-2,
tranchée par le user le 2026-11-23) ; la question de licence posée par le parseur voisin (backlog,
« Licence » : décision du user, aucun geste ici) ; le paquet web et le module Go des constructeurs 3D du
contributeur (plan du rejeu 3D) ; tout code d'affichage des répliques dans LevelUp.

## 1. Instruction : ce que les pièces disent (2026-11-23)

Lu sur pièces par le rédacteur du plan, en lecture seule : `DEC` au tag v1.5.0 ; le backlog, le registre des
reports, le plan et les annexes de la v1.5.0 ; le plan du rejeu 3D ; les commandes `replay-equiv`,
`replay-build` et `replay-corpus-gate` de LevelUp ; le dépôt public du contributeur par `gh` (tête
`a66b791460`, 2026-11-22 : `third_party/README.md`, `third_party/REBASE-v1.3.6.md`) ; le dépôt public du
parseur voisin par `gh` (tête `03291129df`, 2026-11-22 : arborescence et notice de son banc). Les références
`fichier:ligne` de `DEC` sont celles du tag v1.5.0 : l'exécutant les rouvre à l'étape 0.

### 1.1 Vitesse : l'existant

| Constat | Pièce |
|---|---|
| Le film se charge chunk après chunk, sur un seul fil : lecture du fichier, décompression, découpage en paquets, dans un seul appel | `DEC/internal/source/film.go:103-128` (`Load`) ; façade `decfilm.Load`, `decfilm.LoadDir` (`decfilm/decfilm.go:467,470`) |
| Décompression : bibliothèque standard pour zlib (`film.go:188`, `Decompresser` ; le choix zstd ou zlib se fait dans `inflate`, `film.go:217-246`), `klauspost/compress` pour zstd (`internal/source/zstd.go`). `decfilm.Inflate` est public (`decfilm.go:466`) : la décompression d'un chunk se chronomètre à part, pas à l'intérieur d'un chargement | idem |
| Aucun fil lancé en production, hors la sentinelle de mémoire | `DEC/memguard/memguard.go:175` (seul `go` hors tests) |
| La marche des trames porte UN état du monde d'un bout à l'autre du film : un chunk dépend de ceux qui le précèdent (table d'entités, liaisons des images-clés, table anticipée). Ce monde sert à LIRE les trames | `DEC/internal/grammar/marche_trames.go:109,132` (`marcheurDesTramesDepuis`, `parcourir`), `DEC/internal/grammar/world.go:94,120` (`NewWorld`, `PoserChunkCourant`) |
| Une marche, N canaux : chaque canal déclare ses intérêts, la marche distribue. Le choix des canaux par l'appelant n'existe pas | `DEC/internal/grammar/distribuer.go:142` (`Distribuer` ; en-tête lignes 1-70) ; `DEC/replay/options.go:27` (`Options` sans sélection) |
| La marche commune de la source de dégât et de la cuisson est DÉJÀ en production | `DEC/internal/facts/killsource/decode.go:360` (`LireLaMarcheCommune`) ; `DEC/replay/build_from_film.go:98` (`PrendrePourLaCuisson`) |
| Trois entrées publiques de l'assemblage : film vers document, film vers document et faits, faits vers document. Un observateur date chaque étape (`Options.Observe`) : la dernière étape du balayage, `clockOrigin`, sépare le décodage de la publication | `DEC/replay/build_from_film.go:51,63`, `build_from_facts.go:52`, `options.go:461-464`, `observe.go:92` (`clockOrigin` ; liste fermée lignes 47-92 ; émission dans `replay/film_scan_pont.go:175`) |
| Un instrument de durée existe, jetable, sous l'étiquette `research` : chargement, décodage, publication, pic de mémoire ; options MINIMALES (carte seule : ni source de dégât, ni tableau, ni objectifs, ni zones) | `DEC/research/duree/duree_research_test.go` |
| Six bancs `Benchmark*` dans quatre fichiers, tous de micro-lecture ; aucun de bout en bout | `DEC/internal/grammar/bench_*_test.go` |
| Le déterminisme de la marche est déjà testé (deux marches, même empreinte ; deux films en parallèle sous `-race`) | `DEC/internal/grammar/marche_determinisme_test.go` (`TestLaStructureEstDeterministe`, ligne 75) ; `DEC/internal/grammar/deux_films_parallele_test.go:51` (`TestDeuxFilmsEnParallele`, le test `-race`) ; CI `film-race`, `.github/workflows/ci.yml:66-78` |
| La révision d'une couche est tenue par une empreinte des JETONS de ses sources (commentaires et mise en page écartés) : toucher `internal/source` ou `internal/grammar` fait rougir le test de révision, même sans changer la lecture. La v1.5.0 a déjà refigé une empreinte sans monter la valeur | `DEC/internal/grammar/rev_test.go:1-45` ; journal v1.5.0, seconde intégration |
| Les faits décodés se gardent sous `<racine>/data/cache/film_facts/<titre>/` et sont réemployés quand leurs révisions sont celles du code : deux branches aux mêmes révisions se servent le même cache (incident de la v1.5.0) | `LU apps/go-api/internal/domain/title/registry_film_facts.go:52-77` ; journal v1.5.0, 2026-11-22 |
| `replay-equiv` sans `-deux-passes` compare aux références du clone ; avec `-deux-passes`, il compare deux passes du MÊME commit (décodage, puis rejeu depuis les faits) et ne lit AUCUNE référence. Ni lui, ni `replay-build`, ni le gate du corpus n'ont de réglage du niveau de fils | `LU apps/go-api/cmd/replay-equiv/main.go:136-163`, `cmd/replay-build/main.go:59-72`, `cmd/replay-corpus-gate/main.go:117-132` |
| Le dépôt du décodeur n'a aucun dossier `cmd/` ; aucun catalogue de bornes de carte n'y a été trouvé | arborescence de `DEC` |
| Plafond de mémoire par film : 3 Gio en production, 2 Gio pour une mesure sur le poste, 4 Gio dans le gate | `DEC/memguard/memguard.go` ; `LU apps/go-api/internal/replaychild/replaychild.go:68-121` ; `cmd/replay-corpus-gate/bake.go:72` |
| Dix témoins de durée existent (2026-11-10) : builds 1.12 et 1.13, six modes, 25 à 67 chunks, dont une Grande bataille à 33 joueurs. AUCUN n'est dans le corpus d'équivalence : leurs identifiants courts ne sont pas dans le dépôt public | `.ai/annexes_marche_complete/temoins_duree.txt` ; `DEC/replay/testdata/equivalence/` |
| Deux reports de vitesse mesurés attendent : trois lecteurs d'images-clés qui cherchent chacun (environ 30 % du décodage seul) ; sonde de calibration hors de la marche (1,5 %) | `.ai/REGISTRE_REPORTS.md:362-363` |
| Bruit connu : `replay-equiv -deux-passes` signale « 1 écart à classer : kill » dès la v1.4.1 ; un soupçon de non-déterminisme sur `joinSegments` n'a pas été observé | plan v1.5.0, Découvertes ; `DEC/replay/score_timeline_players.go:181-188` |
| Outils de compatibilité : ni `gorelease` ni `apidiff` ne sont installés sur le poste (`benchstat` l'est) | `which`, `go env GOPATH`/bin, 2026-11-23 |

### 1.2 Vitesse : ce que fait le parseur voisin

Source : `.ai/BACKLOG.md`, entrée « Comparaison avec les parseurs voisins » (lecture de ses sources à la
révision `6e93b68`, version 0.6.1), et la notice de son banc relue le 2026-11-23
(`benchmarks/001_film_parse/README.md`). Aucune copie locale ; ses sources de méthode
(`src/theater/parser/chunk.rs`, `parser/v41/chunks.rs`, `src/theater/runtime/mod.rs`) n'ont PAS été relues
pour ce plan : chaque piste V les lit avant d'écrire sa conception.

- Quatre méthodes : décompression des chunks répartie sur plusieurs fils ; lecture de plusieurs chunks à la
  fois, chaque fil partant d'une copie de l'état, la lecture refaite dans l'ordre quand le chunk dépendait du
  précédent ; état du monde à un instant construit à la demande (`FilmRuntime`) ; lectures partielles
  (réplication seule, résumé seul).
- Son banc : sept films fixes, 25 passes par film, médianes, fichiers lus avant le chronomètre, ordre des
  passes tourné, un mode à un fil et un mode à plusieurs fils, une empreinte de la sortie relevée hors
  chronomètre. C'est le modèle du point 2.
- Ses chiffres (Apple M1 Max) ne mesurent pas notre travail : il rend des objets typés, nous rendons le
  document de rejeu fini. Aucun objectif chiffré n'en est tiré (décision du user du 2026-11-22).

### 1.3 Copies du contributeur du lecteur 3D

Accessibles par `gh` (dépôt public), pas en local. Lu le 2026-11-23 à `a66b791460`, par sa propre liste de
changements, PAS par son code. Sa copie a bougé depuis la note du 2026-11-22 : le document y est au schéma
130 et les faits au schéma 25 (la note disait 127).

| Brique | Sa copie | Publiée chez nous | Ce qu'elle porte en plus (sa liste, `third_party/README.md`) |
|---|---|---|---|
| Décodeur | v1.3.6 portée | v1.5.0 (document 126, faits 30) | pistes à pleine cadence ; visées seules ; direction d'un tir par la visée ; décalage d'un tir dans l'image ; tireur par place de contrôle ; lectures du manche de déplacement ; postures de la vue de contrôle (accroupi, glissade, saut tiré de la vitesse verticale) ; arme en main par place ; dégâts par place ; causes des morts ; gestes (rechargement, lancer, coup) ; munitions portées ; hauteur des projectiles (son 126) ; projectiles à pleine cadence (son 127) ; projectiles collés (son 128) ; détonation d'un lancer sans vol publié (son 129) ; candidats de socle (son 130) ; socles d'équipement ; axes d'un socle ; seconde famille du Stalker Rifle ; table des scores cibles par variante ; `filmcache.ParseManifest` et `NewSource` ; `decfilm.LoadMapQuantCatalogBytes` ; codec `REPLAYINPUTS30` ; messages et clés de journal traduits en anglais |
| Cartes | v1.0.1 modifiée | v1.4.0 | export du monde en GLB pour la 3D (117 fichiers neufs, 15 modifiés, dépendance `woozymasta/bcn`) ; décodeur d'images `bitmapdecode` ; matériaux, émissifs, styles Forge, terrain, décor, armes, armures, bras, effets de particules ; messages traduits |
| Lecteur de fichiers | v1.0.1 modifiée | v1.2.0 | lecture des compagnons `.module_hd1` ; dix fichiers pour le maillage du monde ; `mapvar` : mode Forge, axes des socles, éclairage, styles ; messages traduits |

Le décodeur DÉPEND des deux autres briques : `go.mod` exige `halo-infinite-files-reader v1.0.1` et
`halo-infinite-map-images v1.0.1` ; son code de production importe `files-reader/mapvar` (paquet `replay`),
ses tests importent `map-images/himap`. Un canal du contributeur qui s'appuie sur un ajout de `mapvar` (les
axes d'un socle, par exemple) exige donc d'abord un tag du lecteur de fichiers, puis une montée du `require`
du décodeur. L'ordre des tags se lit dans l'inventaire (P.1).

Recouvrements déjà visibles avec la v1.5.0, à lui signaler canal par canal (items P.1, P.2) : `reloads[]` contre ses
gestes ; `moveCommand` contre son manche ; `coverage.seats.tirsParPlace` contre son tireur par place ;
`damage[]` contre ses dégâts par place ; `shots_visee.go` existe des deux côtés. Deux de ses apports
heurtent les règles du dépôt : le saut « tiré de la vitesse verticale » est une dérivation, pas une lecture
(règle R-2) ; la table des scores cibles par variante est une table par variante (R-2). Ses traductions de
messages ne remontent pas (la note du 2026-11-22 le dit déjà).

État au 2026-11-23 (user : « Oui [le contributeur] s'en occupe seul ») : le contributeur reporte LUI-MÊME ses ajouts
sur la v1.5.0, pour le décodeur et pour ses copies de la brique des cartes et du lecteur de fichiers. Son
retour est attendu « dans tous les cas » (user, même message) ; son accord sur le découpage D-15 (paquet
web, module Go des constructeurs) relève du plan du rejeu 3D. Fichiers que sa copie du décodeur touche,
d'après son inventaire de rebase (`third_party/REBASE-v1.3.6.md`, lu le 2026-11-23) et que nos pistes
touchent aussi : `replay/options.go`, `observe.go`, `film_scan_mouvement.go`, `film_scan_pont.go`,
`layers.go`, `coverage.go`, `document.go`, `document_schema.go`, `document_chronicle.go`,
`document_stances.go`, `shots.go`, `shots_visee.go`, `build.go`, `build_calques.go`, `filmfacts*.go` ;
`internal/grammar/vue_a_lecture.go`, `distribuer_tetes.go`, `movement_states.go`,
`keyframe_etat_complet_bipede.go`, les fichiers de révision ; `filmcache/filmcache.go` ; `decfilm/` ; tous
les goldens. C'est la liste des collisions de la piste P.

**Inventaire du 2026-11-23** (`.ai/annexes_decodeur_v160/inventaire_contributeur.md`, lecture seule par `gh api`, tête `a66b791460` lue, arbres non tronqués). Ce qui est marqué « supposé » se déduit des noms de fichiers et des en-têtes, sans vérification ligne à ligne.

- Versions de départ : décodeur v1.3.6 (`5d63c67`), cartes v1.0.1 (`95547a3`), lecteur v1.0.1 (`01ec0fd`). Schémas de la copie : document 130, faits 25 (le README de la copie annonce encore 125 et 23 : périmé).
- Décodeur (copie 2 607 fichiers ; v1.3.6 : 2 534 ; v1.5.0 : 2 629). Contre v1.3.6 : 73 fichiers ajoutés et 249 modifiés (comparaison par empreinte). Contre v1.5.0 : 186 fichiers présents seulement dans la copie (68 ajouts du contributeur, 118 retirés ou renommés par l'amont, surtout des tests de recherche), 208 présents seulement en v1.5.0, 420 de même chemin au contenu différent (mélange non exploitable). Sur les 186, 33 sont du code de production.
- Cartes (copie 320 fichiers). Contre v1.0.1 : 195 ajoutés, 27 modifiés. Contre v1.4.0 : 57 fichiers absents de la copie. Le README annonce 117 fichiers neufs et 15 modifiés : chiffre non retenu, à trancher.
- Lecteur de fichiers (copie 92 fichiers). Contre v1.0.1 : 6 ajoutés (surtout des tests), 20 modifiés, 5 absents (fixtures de jeu omises).
- Canaux neufs probables (supposé, à confirmer sur tout l'arbre de la v1.5.0) : pistes à pleine cadence (`FullRate`), projectiles collés et détonations, munitions portées, gestes du joueur, codec `REPLAYINPUTS30`, table des scores cibles par variante (table par variante, refusée par R-2 sauf décision).
- Recouvrements (lus sur pièces) : `reloads[]` de la v1.5.0 contre les gestes de la copie (`replay/document_gestes.go`, genre 38) : recouvrement direct. `moveCommand` (`replay/document_stances_commandes.go`, intervalles tenus) contre `replay/stick_de_deplacement.go` (valeurs brutes par vie). Fichiers communs modifiés des deux côtés, à comparer : `replay/tirs_par_place.go`, `replay/shots_visee.go`, `internal/grammar/tir_continu.go`, `replay/match_clock.go` (contenu non examiné).
- Non vérifié : le contenu des 425 fichiers modifiés (décodeur, cartes et lecteur), les traductions de messages annoncées par le README, et l'ordre des tags des trois briques (exigé par P.1).

### 1.4 Entrées du backlog qui concernent le décodeur, et leur sort

| Entrée (backlog, sauf mention) | Constat sur pièces | Sort |
|---|---|---|
| Comparaison avec les parseurs voisins : vitesse | section 1.1 et 1.2 | Pistes V0 à V4 |
| Idem : nom des options de variante (un hachage chez nous) | lecture des réglages présente (`film_variante_de_partie.go`) | E.3, source = le jeu ou le film |
| Idem : apparence enregistrée des Spartans (armure, revêtement) | absente chez nous | E.4 (recherche bornée) |
| Idem : événements de zoom et de coup de mêlée sans mort | la visée est déjà publiée ; le coup figure dans les gestes du contributeur | E.5 (contrôle), P |
| Idem : licence | décision du user | Hors périmètre |
| Éliminations dont un bot est l'auteur absentes du fil | le film nomme ses éliminations (`scoreActions`) ; le calque `eliminations[]` est neuf en v1.5.0 | E.1 |
| Durées et meilleure série en tant que VIP | écarté de la v1.5.0 avec l'accord du user | E.2 |
| Libellés des médailles sans jumeau | champ déclaré en v1.5.0 (J.2) ; reste la liste des champs « servis seuls » côté LevelUp | `[~]` montée v1.5.0 (L.4), contrôlé en L.4 |
| Chantier décodeur à ouvrir (D-6) | entrée chapeau, vidée par la v1.5.0 | `[~]` close à la clôture |
| Verdict « lecture complète » du score | traité en v1.5.0 (B.1, B.3) ; restes : Assaut, cinq Grandes batailles « incomplètes », une « complète » à tort, arbre décalé du build précédent, 34 films de Grande bataille à drapeau sans groupe de scripts, courbe qui ne recule pas | Piste A |
| Tirs, touches et dégâts : lecture suffisante | justesse des tirs | Écarté : hors périmètre (DD-2, user, 2026-11-23) |
| Section « Chantier précision » : tirs du BR75 sur un build ancien et baisse acceptée du BR75 ; précision des armes à énergie (créateur d'un objet projectile) ; copies des messages de tir de rang 1 et plus ; parts d'échec en hausse (tirs sans visée, projectile au tir ambigu, tirs sans place, tirs embarqués sans trajet) ; table `match_weapon_hit_distance` sans écrivain ni lecteur | justesse des tirs, des touches, de la précision par arme | Écarté : hors périmètre (DD-2, user, 2026-11-23) |
| Coup d'envoi estimé depuis l'API | réglé côté LevelUp (ADR 0050) ; rien dans le décodeur | Écarté : pas le décodeur |
| Ordre de rotation des collines | demande d'affichage, aucun travail du décodeur | Écarté : pas le décodeur |
| Nommer les états des marqueurs | la lettre se lit (G.1 de la v1.5.0) ; reste la lettre comme contrôle et le composant qui arrête la lecture aux images-clés | B.6 |
| Commandes et gestes du joueur | rechargements et commande publiés en v1.5.0 ; bouton de saut à 63 % d'accord ; accroupi et glissade jamais annoncés sur les versions récentes | Piste C |
| Répliques des Spartans | texte retrouvé dans le jeu le 2026-11-22 (191 sons sur 201) ; cadrage du user du même jour | Piste D (DD-4 : dans la v1.6.0) |
| Temps forts de type « mode » | rien à publier de plus ; 3 410 événements en Drapeau, 691 en Total Control, 303 en Extraction sans action de score | F.4 |
| Temps forts « mode » en double en base | base de LevelUp | Écarté : pas le décodeur |
| Touches et précision par l'ancien appariement | retrait fait à la montée v1.5.0 | `[~]` pour le retrait ; table écartée : hors périmètre (DD-2) |
| Porteurs du drapeau faux en Grande bataille | faux porteur corrigé (E.1, E.2) ; 16 portages gardent un « au socle » faux ; bande au-delà de 8 non ouverte | Piste B |
| Auteur d'un dégât sur l'index brut | fait en v1.5.0 (D.1) | `[~]` |
| Écarts restants du gate à la montée v1.3.9 | compteurs nommés faits en v1.5.0 (piste C) | `[~]`, remesuré en L.7 |
| Trajet de Warthog ; pièces montées à plus de 15 m | faits en v1.5.0 (J.7, J.8) ; reste le repli du voisinage, plus utilisé | F.3 |
| Deux films aux faits d'un ancien schéma | dépend de la régénération | L.6 |
| `replay-equiv` ne compare pas le tableau | fait en v1.5.0 (J.4) | `[~]` |
| Découvertes v1.5.0, vague 2 : décalage de 35 s entre film et temps de jeu de l'API | inexpliqué | F.5 |
| Idem : lâchers 2 à 4 images après une mort (4 cas) | une tolérance de temps serait une règle neuve | F.6 |
| Idem : accroupi, glissade, bouton de saut, booléens du rechargement | voir Commandes et gestes | Piste C |
| Découvertes v1.5.0, recherches : zone au propriétaire de nom inconnu | un film sur Recharge | B.7 |
| Idem : compteurs de zones sur d'anciens builds | à borner par build avant d'ouvrir au-delà de 8 joueurs | B.4 |
| Idem : victime d'une élimination appariée au plus proche | correction proposée : victime absente et compteur quand plusieurs candidats | E.6 |
| Idem : compteur des émissions jetées gonflé hors Drapeau | corrigé avant le tag (`e703301`) | `[~]`, contrôlé en 0.5 |
| Idem : table des réponses de jeu sans révision de couche | une ligne ajoutée ne rend aucun rejeu « à redécoder » | A.6 |
| Idem : coup d'envoi 0,9 s plus tôt sur trois anciens builds | sous la tolérance, non vérifiable à l'API | `[!]` d'avance : aucune vérité disponible, noté à la clôture |
| Idem : tableau « Impact des coéquipiers » | LevelUp | Écarté : pas le décodeur |
| Plan v1.5.0, Découvertes : sous Windows, le script d'espace de travail écrit le chemin du clone avec des barres obliques ; `go test` d'un sous-paquet du clone échoue alors ; avec des barres inverses tout passe | `LU scripts/briques-workspace.sh:38` (`pwd -W`) ; marche de publication, geste 1 | F.1 |
| Idem : « 1 écart à classer : kill » en double passe | bruit de base à lever AVANT la phase vitesse | 0.4 |
| Idem : 5 rejeux sans build sans compteur nommé | non instruit | F.7 |
| Idem : équipe du porteur inconnue sur une carte à 6 socles | 25 portages sur 25 d'un film | B.3 |
| Idem : sept décisions devenues sans effet du manifeste du gate | avertissements du gate depuis le 2026-11-18 | F.2 (DD-8 : retirées) |
| Registre des reports : images-clés, trois lecteurs ; sonde de calibration | mesurés le 2026-11-13 | V3.2, V3.3 |
| Registre des reports : une seule marche pour la source de dégât et la cuisson | déjà en production (section 1.1) | `[~]`, registre à mettre à jour à la clôture |

## 2. Décisions

Toutes tranchées. Message du user du 2026-11-23, cité tel quel : « Oui [le contributeur] s'en occupe seul. Je suis ok
avec tes recommandations pour tout sauf sur le fait d'attendre une 1.7.0 pour certains trucs. On doit
attendre le retour de [le contributeur] dans tous les cas donc on peut faire le tout patiemment et à notre rythme. Et
tout faire pour la 1.6 je veux dire. » ([le contributeur] = le contributeur du lecteur 3D.)

### Tranché par le user

- **DD-1 — le point 1 : on attend le contributeur, une seule version, une seule régénération.** Tranchée le
  2026-11-23 (« On doit attendre le retour de [le contributeur] dans tous les cas […] tout faire pour la 1.6 »). Aucune
  v1.7.0 : la v1.6.0 n'est pas taguée sans ses canaux. Le contributeur reporte lui-même ses ajouts sur la
  v1.5.0, pour les trois briques (« Oui [le contributeur] s'en occupe seul ») : nos agents n'écrivent pas ce code.
  Notre part est la piste P : préparer, remettre, synchroniser, vérifier, intégrer.
- **DD-2 — chantier « précision » : HORS PÉRIMÈTRE de la v1.6.0.** Tranchée le 2026-11-23 (« En effet les
  items qui touchent à la précision sont hors périmètre »). Rien de la section « Chantier précision » du
  backlog n'entre ici, ni aucune autre entrée qui touche à la justesse des tirs, des touches ou de la
  précision par arme : lecture suffisante des tirs, tirs sans visée, projectiles au tir ambigu, tirs sans
  place, tirs embarqués sans trajet, copies des messages de tir, créateur des projectiles, BR75, table
  `match_weapon_hit_distance`. Garde mécanique sur le travail du contributeur : règle R-14.
- **DD-3 — parallélisme par défaut : un fil.** Tranchée le 2026-11-23 (« ok avec tes recommandations pour
  tout »). La bibliothèque reste à UN fil par défaut. LevelUp règle le niveau de fils par installation : un
  fil sur le serveur tant qu'une mesure sur le serveur ne dit pas mieux ; sur le bureau et en développement,
  la valeur que le relevé et le pic de mémoire mesuré permettent, et un fil faute de mesure. Le plafond de
  mémoire par film ne bouge pas.
- **DD-4 — répliques des Spartans : DANS la v1.6.0.** Tranchée le 2026-11-23 (même phrase, et « tout faire
  pour la 1.6 »). Le décodeur publie l'événement et le nom court. Côté LevelUp : la table des textes et UNE
  maquette interactive des trois idées d'affichage. L'affichage réel se fait dans un plan à part, après le
  choix du user sur la maquette.
- **DD-5 — enchaînement avec la montée v1.5.0.** Tranchée le 2026-11-23 (même phrase). La montée v1.5.0 n'est
  PAS fusionnée dans `release/v8.3.0` maintenant. La montée v1.6.0 part de sa tête (aujourd'hui `93a4744e7b`
  sur `chore/830-decodeur-v150-miroirs`, que le pilote avancera sur `chore/830-decodeur-v150`), reçoit
  `release/v8.3.0` par fusion régulière, et tout entre dans la release en une fois, autour de la
  régénération unique.
- **DD-6 — copies des briques des cartes et du lecteur de fichiers.** Tranchée le 2026-11-23 (« Oui [le contributeur]
  s'en occupe seul », et même phrase). Le contributeur les reporte lui-même, dans leurs dépôts ; elles sont
  reçues par la piste P. Le décodeur dépend de ces deux briques : ce qu'un de ses canaux exige du lecteur de
  fichiers ou de la brique des cartes est tagué AVANT ce canal.
- **DD-7 — l'axe « état du monde à la demande ».** Tranchée le 2026-11-23 (même phrase). Chez nous le monde
  sert à lire les trames : il ne peut pas attendre. L'axe est réduit à ce qui peut réellement attendre ou ne
  pas être refait (piste V3) ; la forme littérale du parseur voisin est abandonnée, motif écrit ; un
  candidat que la mesure ne montre pas en gain est abandonné par écrit.
- **DD-8 — sept décisions devenues sans effet dans le manifeste du gate du corpus.** Tranchée le 2026-11-23
  (même phrase) : retirées (F.2) ; leur liste est montrée dans le rapport d'exécution. Précision du
  2026-11-23 : le retrait ne porte que sur les 7 décisions du 2026-11-16 listées à F.2 ; les 21 décisions du
  2026-11-22 restent nécessaires jusqu'à la fusion de la montée du décodeur dans la branche de version.
- **Nom des options de variante.** Tranchée le 2026-11-23 (même phrase) : les noms viennent du jeu ; crédit
  à nuzzles pour la méthode seulement.
- **Régénération du parc.** Go donné d'avance par le user le 2026-11-18 (« Pour les trucs plus tard je te
  donne mon go d'avance »), rappelé le 2026-11-22 : une seule fois, après la v1.6.0, avant le premier
  démarrage de LevelUp 8.3.0. Le user est prévenu AU LANCEMENT (serveur arrêté, environ 2 h 30, pas pendant
  qu'il joue).
- **Décisions du 2026-11-23 (soir).** Aucune option dans le décodeur : tout est publié, pleine cadence comprise. Dégâts à zéro point publiés. Série de santé et de bouclier par vie publiée. Purge de l'historique git abandonnée. Gestes (accroupi, saut) publiés seulement pour la version actuelle du jeu. Lecture au choix des canaux (V4) abandonnée. Les ajouts du contributeur sont repris par nous (piste P, « Portage par nous ») ; note de migration remise à la fin.
- **Rythme.** 2026-11-23 : « on peut faire le tout patiemment et à notre rythme ». Aucune échéance ; des
  vagues petites et sûres ; aucun item renvoyé à une autre version par manque de temps.

### Tranché par le pilote (conventions, modifiables par le user)

- Tags et fusions des dépôts de briques : par le pilote (délégation des 2026-11-11 et 2026-11-13) ; `main`
  de LevelUp reste un geste du user.
- Une piste de vitesse qui n'apporte aucun gain mesuré au-delà du bruit du relevé n'est pas fusionnée : son
  item passe `[!]` avec ses chiffres. Aucune complexité sans gain.
- Les pistes V n'ajoutent AUCUN champ au document ni aux faits (un compteur de relectures vit dans le relevé,
  jamais dans `coverage`).
- La commande du relevé vit dans `DEC/cmd/film-bench`, sans étiquette `research`.
- Les films du relevé portent dans le dépôt public un nom neutre (`f01` à `f13`) ; la correspondance avec
  les matchs vit dans `.ai/annexes_decodeur_v160/films_du_releve.tsv`.
- Méthodes du parseur voisin : reprises, jamais son code ni ses tables.
- Ce qui est demandé au contributeur de ne pas remonter : ses traductions de messages ; la table des scores
  cibles par variante (LevelUp l'injecte déjà) ; le saut tiré de la vitesse verticale tant qu'une lecture du
  film ne le remplace pas. Le codec `REPLAYINPUTS30` ne remonte que si un de ses canaux l'exige. Un
  désaccord de sa part sur l'un de ces points revient au user.

## 3. Règles d'exécution

Les règles R-1 à R-10 du plan v1.5.0 s'appliquent, avec ces précisions.

- **R-1 — pas de correction sans mesure.** Pistes A à F et P : mesure AVANT, vérité indépendante, mesure
  APRÈS, compte des cas qui changent. Une correction qui améliore un compte et en dégrade un autre s'arrête.
- **R-2 — la grammaire prime.** Aucune heuristique, aucun repli neuf (le gate du corpus juge FAUX tout repli
  parti de zéro), aucune table par film ou par carte, aucune liste de composants écrite à la main : une
  règle se lit dans ce que le film déclare (leçon de la v1.5.0 : la règle du drapeau écrite sur une liste de
  composants dégradait les zones ; lue dans la table des noms du mode, elle était juste).
- **R-3 — une recherche est bornée par session, pas par la version.** Une session d'agent rend : une cause
  prouvée et un item de correction écrit ICI, daté, avant son code ; ou un abandon ÉCRIT et motivé (`[!]`),
  avec ce qui a été écarté. Une recherche abandonnée peut reprendre dans une session suivante tant qu'une
  piste NOUVELLE existe, écrite au journal avant la reprise ; sans piste nouvelle, le `[!]` reste. Aucun
  item n'est renvoyé à une autre version par manque de temps (user, 2026-11-23) ; jamais de correction
  « plausible ». Une recherche se découpe : d'abord un item de MESURE sans code de production, peu coûteux ;
  puis, seulement si la cause est prouvée, un item de CORRECTION (suffixe « c ») écrit dans sa piste avant
  son code, confié à un autre agent.
- **R-4 — une piste, une branche, des écarts attribués.** Clone neuf `Scripts/dec-v160`
  (tête de `main` = `v1.5.0`), un worktree par agent qui écrit (`.../Scripts/dec-v160-<piste>`), branche
  d'intégration `feat/v1.6.0`. Une piste n'entre qu'après une passe d'équivalence dont CHAQUE écart est
  rattaché à un de ses items. Pour une piste V, le seul résultat admis est : aucun écart.
- **R-5 — dépôt public.** Aucun identifiant de joueur réel ni pseudo dans une fixture, un golden, un test,
  un relevé ou un message de commit (contrôle `25(33|35)[0-9]{12}` avant chaque commit, binaires
  décompressés compris). Les agents ne commitent ni ne poussent ; le pilote relit, commite, pousse. Jamais
  `--no-verify`, `git stash`, `git add -A`, jonction ni lien symbolique. Plafond de 500 lignes par fichier.
- **R-6 — registre des changements de sortie.** `.ai/annexes_decodeur_v160/changements_de_sortie.md`, tenu
  par le pilote à chaque intégration ; il devient le corps des notes de version. Les pistes V y ont une
  ligne : « aucun changement de sortie », avec la preuve. Après CHAQUE mise à jour de ce plan ou du
  registre, le pilote relit la ligne écrite dans le fichier (une écriture a échoué en silence en v1.5.0).
- **R-7 — un seul schéma, une seule régénération.** 126 vers 127 une fois, à la première intégration d'une
  piste qui change la sortie ; aucune régénération partielle du parc pendant le plan.
- **R-8 — machine.** Un seul décodage de lot à la fois ; mémoire libre d'au moins 4 Go vérifiée avant chaque
  passe lourde ; suites complètes, passes d'équivalence, gate du corpus et relevés de vitesse au pilote. Un
  relevé de vitesse se prend dans une FENÊTRE DE MESURE : aucun agent en cours (`ListAgents` vide), aucun
  processus `go` ni de test, machine au repos ; le pilote ouvre et ferme ces fenêtres et les écrit au
  journal.
- **R-9 — lecture seule des données.** Rien n'est écrit dans `HOTE` ; les films se lisent en place ou par
  copie (jamais par jonction) ; la vérité de l'API et les faits des matchs viennent d'une COPIE de la base.
- **R-10 — lecteurs d'autres plans.** Aucune piste ne change le sens, l'unité ni l'emplacement d'un champ
  existant du document. La session du rejeu 3D est prévenue avant le tag de tout champ que le contributeur ajoute
  (P.8).
- **R-11 — dossiers, caches, poussée.** Aucun dossier de travail sous `C:\t` (vidé après 72 h) : clones,
  worktrees et sorties sous `Scripts/`. Une branche de travail est poussée dès
  son premier commit relu. À la CRÉATION du worktree de LevelUp : `npm ci` dans `apps/web` (le crochet de
  poussée en a besoin) et espace de travail engendré. Au moment de pousser LevelUp : `GOWORK=off` (le
  crochet joue ce que joue la CI). Caches de compilation : `GOCACHE=%LOCALAPPDATA%\go-build-dec160-<piste>`,
  retirés à la clôture de la piste (`scripts/disk-hygiene.ps1 -Lot dec160-<piste> -Apply`).
- **R-12 — déterminisme (pistes V).** Le non-déterminisme est le risque numéro un. Règles de code : un
  résultat produit par plusieurs fils se range par indice de chunk, jamais par ordre d'arrivée ; aucun fil
  n'écrit une structure partagée ; aucun parcours de `map` ne décide d'un ordre de sortie ; aucune horloge
  ni adresse ne sert de clé. Le nombre de fils effectif est le minimum de l'option, de `GOMAXPROCS` et d'une
  borne écrite dans la brique (8). Preuves : « Preuve d'identité », plus bas.
- **R-13 — révisions de couche et cache de faits.**
  - Une piste V prouve une sortie identique : elle ne monte AUCUNE valeur de révision. Le test de révision
    rougit quand même (il hache les jetons de la couche) : l'agent rend la liste des tests de révision
    rouges ; le pilote refige l'empreinte par la porte nommée de la couche (marche de publication, geste 3),
    à l'intégration, après la preuve d'identité, valeur inchangée.
  - Chaque intégration d'une piste qui change la grammaire ou les objectifs (A à F, et chaque PR du
    contributeur) pose un RANG de
    révision neuf pour la couche touchée. À défaut, le cache de faits (`<racine>/data/cache/film_facts/`)
    est purgé avant toute passe.
  - Le gate de toute piste inclut les tests de révision (commande G-rev, plus bas).
  - Un cache de faits et un espace de travail PAR PISTE qui décode, de noms distincts, jamais partagés :
    worktree `Scripts/LevelUp-wt-dec160-<piste>` (films du corpus copiés sous son `data/cache`, sans jonction)
    et fichier `Scripts/dec-v160-recup/<piste>.work` à barres inverses. Quatre au plus à la fois, retirés à la
    clôture de la piste, après recherche de jonction. Établi le 2026-11-23 : `replay-equiv -repo-root` n'accepte une racine distincte que si elle porte
    aussi `data/titles/halo_infinite/reference`, `config/` et `apps/go-api/cmd/replay-equiv/testdata`, et il y
    écrit son cache, son verrou et `logs/general.log` ; donc UN WORKTREE LevelUp PAR PISTE qui décode, plutôt
    qu'une racine de données partielle.
- **R-14 — garde mécanique de la précision (piste P).** Pour tout canal du contributeur qui touche les tirs,
  les touches ou les dégâts, les empreintes des étapes de tirs, de touches, de dégâts et de statistiques de
  tir de `replay-equiv -out-dir` sont IDENTIQUES avant et après sur les 20 films (les noms exacts de ces
  étapes sont relevés en 0.6). Une empreinte qui bouge : la PR revient au contributeur avec la pièce ; le
  canal n'entre que sous une forme qui laisse ces empreintes intactes.
- **R-15 — fichiers communs.** `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md`, `README.md` et le schéma du
  document ne sont écrits par aucun agent. Chaque piste rend un fragment dans un fichier neuf
  (`.ai/annexes_decodeur_v160/fragments/<piste>_<fichier>.md`) ; le pilote assemble à l'intégration.
- **R-16 — porteur unique des options.** Le nombre de fils et le choix des canaux vivent dans UNE structure
  d'options ajoutée, posée par un seul écrivain avant toute piste de vitesse (V0.6). Aucune piste V ne crée
  son propre réglage.
- **R-17 — une correction après un gate le fait repasser.** Aucun gate n'est tenu pour acquis sur un commit
  antérieur à une correction.
- Marche de publication : `.ai/annexes_decodeur_1118/marche_de_publication.md` (19 gestes), chemins
  remplacés par ceux de R-4 ; le gate du corpus ne juge une branche que si `go.mod` de `LU` la désigne par
  une pseudo-version, avec `GOWORK=off`.
- Reprise de session : ce fichier (cases, journal), le registre des changements de sortie, `git log` de
  `feat/v1.6.0`, `git worktree list` des dépôts, `ListAgents`, puis `.ai/thought_log.md`.

### Commandes de gate communes

`RACINE` = la racine de données de la piste (R-13). `N` = le niveau de fils maximal permis.

- **G-clone** (tout test du clone) : `cd <worktree> && GOWORK=off
  DECODEUR_FILM_RACINE_DONNEES="<HOTE>" go test -count=1 -v <paquets> [-run <motif>] -p 1`. Lecture du
  résultat : zéro `FAIL` ; zéro `SKIP` hors de la liste attendue (tests `...Regenerate`, `Fidelite`,
  `ReecrireIdentites`) ; quand la commande filtre par `-run`, le nombre de `--- PASS` est au moins celui que
  l'item annonce (un `-run` sans correspondance répond `ok` : ce n'est pas un vert). Référence : 301 SKIP
  attendus sur la suite complète à blanc (référence du 2026-11-23, qui remplace 299 : même liste de noms qu'à l'étape 0, l'écart venait du comptage) ; tout nombre supérieur s'explique avant de conclure.
- **G-rev** : marche de publication, geste 3, en mode vérification :
  `GOWORK=off go test -count=1 -v ./internal/profile/ ./internal/source/ ./internal/grammar/
  ./internal/facts/killsource/ ./internal/facts/objectives/ -run 'RevSuit|ChroniqueDe|ChroniqueCouvre|ChroniqueDObjectives'`
  (10 tests `PASS` attendus) et `go test -count=1 -v ./revision/ ./types/`.
- **G-compat** (surface publique, à chaque gate de piste) : `gorelease -base=v1.5.0` lancé par
  `go run golang.org/x/exp/cmd/gorelease@v0.0.0-20261121134650-e1094eeca1b5` (version épinglée le 2026-11-23) depuis le worktree, `GOWORK=off` ; verdict attendu :
  aucun changement incompatible. Si l'outil ne tourne pas sur le poste (à établir en 0.6), repli écrit :
  `go doc -all` de chacun des 19 paquets publics, comparé à la capture faite sur `v1.5.0` ; toute ligne
  retirée ou changée est un échec, une ligne ajoutée est admise.
- **G-façade** (à chaque intégration, pas seulement à la montée) : `cd LU/apps/go-api && go test -count=1 -v
  ./internal/archlint/ -run 'TestSurfaceDeLaFacadeDuDecodeur|TestSurfaceDeReplayCiteeHorsDuDecodeur'`, la
  branche de montée désignant la pseudo-version intégrée ; un plafond dépassé se relève là, daté.
- **G-equiv-ref** (contre des références) : `cd LU/apps/go-api && GOWORK=<espace de la piste> go run
  ./cmd/replay-equiv -repo-root <RACINE> -originaux <HOTE>/data/film_identites/xuids_originaux.txt -fils <n>` :
  `20 identique(s), 0 different(s)`. Compare aux références du clone. Commande exacte, passée à blanc le
  2026-11-23 sans `-update` (5 min 27 s, 20 identiques, 0 différent) : depuis `Scripts/LevelUp-wt-dec160/apps/go-api`,
  `GOWORK=Scripts/dec-v160-recup/dec160.work go run ./cmd/replay-equiv -repo-root <LevelUp-wt-dec160>
  -originaux <HOTE>/data/film_identites/xuids_originaux.txt -out-dir <sortie>`.
- **G-equiv-2p** (double passe, même commit) : la même commande avec `-deux-passes` : artefact identique à
  l'octet entre le décodage et le rejeu depuis les faits, aucun écart à classer. Ne lit aucune référence :
  elle ne prouve rien contre la v1.5.0. Une double passe est verte quand les 20 artefacts sont identiques à
  l'octet et que ses seuls écarts à classer sont ceux de l'écart attendu : « killsource, 20 sur 20, jamais dans
  l'artefact » (cause prouvée le 2026-11-23 : le champ non exporté `paquet` de `killsource.Kill`,
  `internal/facts/killsource/kill.go:69-72`, entre dans l'empreinte de l'étape sans être écrit au fichier de
  faits).
- **G-corpus** : marche de publication, geste 9, avec `--fils <N>` : 19 témoins, ni gain, ni perte, aucune
  décision ajoutée au manifeste. La base se cuit comme elle se cuisait ; la tête au niveau `N`.
  Commande de référence (passée à blanc le 2026-11-23, comme test d'infrastructure) : depuis
  `Scripts/LevelUp-wt-dec160/apps/go-api`, `GOWORK=off`, compiler `./cmd/replay-corpus-gate` (`CGO_ENABLED=0`),
  puis `replay-corpus-gate.exe --reference=base --base <sha> --strict --source-root <LevelUp-wt-dec160>
  --parc-root <gate>/parc --lock-root <gate>/lock --work-root <gate>/workN --json <gate>/<nom>.json`,
  montage sous `Scripts/dec-v160-recup/gate/`.

Le drapeau `-fils` de `replay-equiv` et de `replay-build`, et `--fils` du gate, n'existent pas aujourd'hui :
ils sont posés par V0.7, AVANT la première piste de vitesse.

### Preuve d'identité à l'octet (pistes V)

Référence de V3 et V4 (V1 abandonnée le 2026-11-23 ; la piste G passe en premier) : la v1.5.0 (références du clone, jamais refigées avant la fin de la phase).
Référence de V2, qui s'intègre après les pistes du backlog : le MÊME commit au niveau 1.

1. G-equiv-ref aux niveaux 1, 2, 4 et `N`, puis sous `GOMAXPROCS=1` et `GOMAXPROCS=2` au niveau `N`.
   Pour V2 : au niveau 1, contre les références de la branche (aucun écart admis) ; aux niveaux 2, 4 et `N`,
   empreintes par étape (`-out-dir`) égales à celles du niveau 1 du même commit.
2. G-equiv-2p au niveau `N`.
3. Trois passes complètes des 20 films au niveau `N` (G-equiv-ref trois fois) : trois fois 20 identiques.
4. Relevé A (V0) sur les 13 films au niveau 1 et au niveau `N` : empreintes du document et des faits égales
   entre elles et égales à celles du relevé de référence (v1.5.0 pour V3, V4 ; niveau 1 du même commit
   pour V2).
5. Dix passes du relevé A au niveau `N` sur trois films, dont la Grande bataille à 33 joueurs : empreintes
   constantes.
6. Tests neufs de la piste, par G-clone avec `-race -count=3` sur les mini-bobines du dépôt, dont UN test
   qui perturbe l'ordre de fin des chunks par un délai aléatoire à graine écrite et exige la même empreinte.
   Le job CI `film-race` reçoit ces tests.
7. À l'intégration : G-corpus au niveau `N`.

| Piste | Items | Modèle | Effort | Risque |
|---|---|---|---|---|
| 0 témoins, références, instruments | 8 | pilote (Fable) ; un agent Sonnet | moyen | aucun |
| V0 relevé de vitesse, socle des options | 7 | Sonnet | moyen | faible |
| V1 décompression sur plusieurs fils | 4 | Sonnet | rapide | faible |
| V4 lecture au choix des canaux | 5 | Sonnet | moyen | moyen |
| V3 lectures refaites, état à la demande | 5 | Sonnet | moyen | moyen |
| V2 plusieurs chunks à la fois | 5 | Opus (état partagé entre fils, relecture en séquence : l'erreur y est silencieuse) | lourd | élevé |
| A score et tableau | 6 | Sonnet ; Opus pour A.3 (grammaire d'anciens builds) | moyen | moyen |
| B drapeau et zones | 7 | Sonnet ; Opus pour B.1 (trois causes dans la grammaire) | lourd | élevé |
| C gestes | 4 | Sonnet | moyen | moyen |
| D répliques | 6 | Sonnet | lourd | moyen |
| E canaux neufs du backlog | 6 | Sonnet ; Haiku pour E.5 | moyen | moyen |
| F contrôles et petites entrées | 7 | Haiku (F.1, F.3, F.7) ; Sonnet | rapide | faible |
| P accueil du travail du contributeur | 8 | pilote ; Sonnet pour l'inventaire et la relecture de chaque PR | lourd | élevé |
| R relecture | 2 | Fable ; une ronde de corrections Sonnet | moyen | aucun |
| L publication, montée, régénération | 9 | pilote ; Sonnet pour les miroirs | lourd | élevé |

La table ci-dessus résume les pistes ; le modèle et l'effort de CHAQUE item sont dans la « Table
d'exécution », qui prime. Estimation, comptée sur cette table, items conditionnels compris : environ
20 sessions Haiku (effort bas), 14 Sonnet à effort bas, 30 Sonnet à effort moyen (dont une relecture par PR
du contributeur, une dizaine), 6 Sonnet à effort élevé, 2 Opus (conception et code de V2), 2 Fable
(relecture finale, diff de la montée). Une reprise de recherche (R-3) ajoute une session, Opus seulement
sur motif écrit. Passes du pilote, recomptées :
- preuve d'identité d'une piste V : 9 passes d'équivalence, environ 1 h, plus deux relevés A d'environ
  25 minutes : 1 h 50 par piste, 7 h 20 pour quatre ;
- étape 0 : marche à blanc (35 minutes dont le gate), trois doubles passes (40 minutes), relevés de
  référence (1 h) : 2 h 15 ;
- intégration « vitesse » : preuve complète et gate du corpus, 2 h 15 ;
- pistes A à F : une passe et deux reprises d'attribution par piste, 20 minutes chacune, 2 h ; suite
  complète du clone à chaque intégration (5 minutes, une douzaine de fois), 1 h ;
- accueil du contributeur : une vérification à l'arrivée par PR (équivalence attribuée, double passe, garde
  de la précision, suite du clone), 30 minutes, pour une dizaine de PR sur les trois briques ; un test du
  jeu installé pour les deux autres briques ; un gate du corpus de plus après son intégration : 6 h ;
- fin : équivalence refigée et double passe (30 minutes), gate du corpus compté TROIS fois (la v1.5.0 en a
  demandé quatre), 1 h 15, relevés finals (1 h), gates de LevelUp et filet de poussée (1 h 30) : 4 h 15 ;
- régénération : 2 h 30, dérivés 25 minutes.
Total : environ 28 heures de machine hors régénération, dont 8 heures de relevés en fenêtre de mesure,
pendant lesquelles aucun agent ne tourne. Aucune échéance : la fin de la version dépend du retour du
contributeur. Notre part tient en 9 à 11 jours de sessions, étalés à volonté. Le coût est annoncé au user
avant chaque vague.

### Vagues et parallélisme

User, 2026-11-23 : « C'est bien de découper les morceaux en plus petit, ça permet de paralléliser et
d'utiliser des modèles et des efforts moins coûteux en général. » Chaque item de la table d'exécution est
petit et autonome : UN agent, une session courte, le modèle et l'effort les moins coûteux qui suffisent.

Quatre classes de charge règlent ce qui tourne en même temps :

| Classe | Ce que fait l'item | Combien à la fois |
|---|---|---|
| S | Sans film : lecture de code ou de pièces, conception, rédaction, inventaire | autant que la borne d'agents |
| B | Code et tests ciblés sur les mini-bobines du dépôt | jusqu'à 4, sans fichier commun |
| L | Décode un lot de films de l'hôte (mesure sur le parc, équivalence, suite complète, gate) | UN seul, file tenue par le pilote |
| M | Fenêtre de mesure de vitesse (relevés A et B, profil) | seul sur la machine : rien d'autre ne tourne |

Borne : 4 agents d'écriture à la fois, toutes sessions confondues ; les items S en lecture seule s'y
ajoutent. Deux items qui touchent un même fichier passent l'un après l'autre. Une intégration à la fois,
par le pilote.

Chemin critique : 0, V0, la phase vitesse, le schéma 127, le gel de la base, l'accueil du contributeur.
Les pistes du backlog ne touchent aucun fichier des pistes de vitesse (table des fichiers) : leur code
s'écrit EN MÊME TEMPS, dans leurs worktrees ; seule leur intégration attend la phase vitesse.

| Vague | Items en parallèle (classe) | File L et fenêtres M, au pilote | Coût relatif |
|---|---|---|---|
| 0 Préparation | 0.3, 0.5, 0.6, 0.7, P.1a, P.1b (S, Haiku) ; 0.8 (Sonnet bas) ; P.2 après P.1 (S) | 0.1, 0.2, 0.4 (L) | faible |
| 1 Socle | V0.1, V4.5 (S, Haiku) ; V0.2-3 et V0.6 (B, deux agents) ; V0.7 après V0.6 (B) ; mesures A.1, B.1, C.1 (L, une à la fois) | V0.4, V0.5 (M) ; Gate V0 | moyen |
| 1 bis — gains du profil | G.2, G.6, G.7 en parallèle (B, trois agents), puis G.1 (B) après G.2 | fenêtre de mesure et preuve au pilote : relevé complet après chaque intégration (M), preuve d'identité (L) | moyen |
| 2 Vitesse 1 | conceptions V4.1, V3.1 (S) ; corrections A.2, A.5, A.6 (B, un agent, à la suite) ; mesures A.3, A.4, C.2, C.3 (L) | — | moyen |
| 3 Vitesse 2 | V4.2-3 (B) ; C.1c et C.2c si prouvées (B) ; D.1 (B) ; F.1, F.3, F.4, F.7 (Haiku) ; mesures B.3, B.7, D.2, E.1 (L) | V4.4 (M) ; preuve de V4 (L) | moyen |
| 4 Vitesse 3 | V3.2, puis V3.3, V3.4 (B) ; B.2, B.4, B.6 (B, un agent, à la suite) ; D.3, D.4 (B, S) ; E.3, E.6 (B) ; mesures E.4, F.5, F.6 (L) | V3.5 (M) ; preuve de V3 (L) | moyen |
| 5 Intégration « vitesse » | E.2 après A (B) ; E.5, C.4 (S, Haiku) ; F.2 (Haiku) | I.1 à I.3 : preuve complète (L), gate du corpus, relevés (M) | élevé (machine) |
| 6 Intégrations du backlog | B.5 si sa condition est tenue (B) ; D.5, D.6 côté LevelUp (B, S) | A, C, B, D, E, F intégrées une à une (L), schéma 127 à la première | élevé |
| 7 V2 | abandonnée le 2026-11-23 (voir la piste V2) | — | — |
| 8 Gel et accueil | relectures P.5 par PR (S, Sonnet), une par PR | P.4 gel ; vérifications P.5 (L) ; P.6 ; P.7 ; P.8 | élevé, étalé sur le retour du contributeur |
| 9 Fin | R.1 (Fable), R.2 (Sonnet) ; miroirs L.4 (B) ; notes de version (Haiku) | L.1 à L.9 | élevé |

Une recherche qui n'a pas abouti suit R-3 : elle reprend si une piste nouvelle existe, sans retarder les
vagues ; son item de correction s'intègre avant le gel. Si le contributeur est prêt avant la vague 8, ses
PR sont relues dès leur arrivée (P.5) et attendent le gel pour entrer. S'il tarde, les vagues 0 à 7 se
terminent et la version attend, branche gelée. Le coût de chaque vague est annoncé à son lancement.

### Table d'exécution : un item, un agent

Elle prime sur les mentions de modèle des titres de piste. « Effort » : bas, moyen, élevé. Une recherche
est découpée : un item de MESURE sans code (suffixe absent ou « m »), puis, seulement si la cause est
prouvée, un item de CORRECTION (suffixe « c ») écrit dans sa piste avant son code. Les items « c » ci-dessous
sont conditionnels. Opus n'apparaît qu'avec son motif.

| Item | Modèle, effort | Classe | Fichiers ou rendu | Dépend de |
|---|---|---|---|---|
| 0.1, 0.2 | pilote | L | clone, worktree, marche à blanc | — |
| 0.3 | Haiku, bas ; le pic par le pilote | S, L | `films_du_releve.tsv` | 0.1 |
| 0.4 | Sonnet, moyen (instruction de l'écart « kill ») ; passes au pilote | S, L | constat au journal | 0.2 |
| 0.5, 0.6, 0.7 | Haiku, bas (trois agents) | S | constats au plan ; liste des appelants | 0.1 |
| 0.8 | Sonnet, bas | L | export hors dépôt, `avant.md` | 0.1 |
| V0.1 | Haiku, bas | S | `bench/films.tsv`, `bench/README.md` | 0.3 |
| V0.2, V0.3 | Sonnet, moyen | B | `cmd/film-bench/`, `bench/` | Gate 0 |
| V0.4 | Haiku, bas (script) ; mesure au pilote | S, M | `releve_bout_en_bout.md` | V0.7, 0.8 |
| V0.5 | pilote (mesure) ; Haiku, bas (mise en forme du profil) | M, S | `bench/releves/v1.5.0.tsv`, `profil.md` | V0.2 |
| V0.6 | Sonnet, bas | B | `replay/options.go`, `decfilm/` | Gate 0 |
| V0.7 | Sonnet, moyen | B | `LU cmd/replay-equiv`, `replay-build`, `replay-corpus-gate` | V0.6 intégrée |
| V1.1 (abandonné) | Sonnet, bas | S | conception au plan | V0.5 |
| V1.2, V1.3 (abandonnés) | Sonnet, moyen | B | `internal/source/`, `filmcache/`, `decfilm/` | V1.1, Gate V0 |
| V1.4 (abandonné) | pilote | M | relevés | V1.3 |
| V4.1 | Sonnet, moyen | S | conception au plan | V0.5, 0.7 |
| V4.2, V4.3 | Sonnet, élevé (plusieurs fichiers) | B | `replay/film_scan*.go`, `build_from_film.go`, `movement_states.go` | V4.1 (V1 abandonnée le 2026-11-23) |
| V4.4 | pilote | M | relevés | V4.3 |
| V4.5 | Haiku, bas | S | fragments de crédit | — |
| V3.1 | Sonnet, moyen | S | candidats au plan | V0.5 |
| V3.2 (abandonné) | Sonnet, élevé (trois lecteurs, une recherche) | B | `marche_images_cles*.go`, `keyframe_etat_complet_*.go`, `keyframe_loadout.go` | V3.1, V4 |
| V3.3 | Sonnet, bas | B | sonde de calibration | V3.2 |
| V3.4 | Sonnet, moyen, un agent par candidat | B | selon V3.1 | V3.2 |
| V3.5 | pilote | M | relevés | V3.4 |
| V2.1 (abandonné) | Sonnet, élevé (instruction, sans code) | S | `v2_conception.md` | V0.5 |
| V2.2 (abandonné) | Opus, moyen (motif : conception d'une lecture concurrente sur un état partagé) | S | conception au plan | V2.1 |
| V2.3, V2.4 (abandonnés) | Opus, élevé (même motif : l'erreur est silencieuse) | B | `marche_trames*.go`, `distribuer*.go`, `world.go`, `keyframe_liaison.go` | V2.2, V4, V3 |
| V2.5 (abandonné) | pilote | M | relevés | V2.4 |
| G.2 | Sonnet, moyen | B | `canal_des_morts.go`, `canal_des_vitalites.go` (chemins du profil) | V0.5 (fait) |
| G.6 | Sonnet, moyen | B | `offline_filters.go`, `offline_biped_band.go`, `positions_lues.go` | V0.5 (fait) |
| G.7 | Sonnet, moyen | B | `equipment_recovery.go` | V0.5 (fait) |
| G.1 | Sonnet, moyen | B | `localisateur.go`, `frame_records.go` | G.2 |
| G.4 | Sonnet, élevé (lancé le 2026-11-23) | B | `delta_biped_walk.go`, `offline_biped.go`, `offline_aim_only.go` | profil 2 (fait) |
| G.8 (abandonné, remplacé par G.8') | — | B | `keyframe_loadout.go` | profil 2 (fait) |
| G.5 (fait) | non consigné | B | `world.go`, `debut_de_liste.go` | profil 2 (fait) |
| H.1 (fait) | non consigné | B | `offline_biped.go` (`matchBipedHeader`) | profil 2 (fait) |
| G.8' (fait) | non consigné | B | `keyframe_loadout.go` (`motsParRecord`) | profil 2 (fait) |
| G.10 | à décider | B | `replay/shots.go`, `replay/vehicle_tracks.go` (`indexBySlot`, `vehiclePositionsBySlot`) | profil 2 (fait) |
| G.11 | pilote | S | note à l'hôte (`-plafond-gio`) | profil 2 (fait) |
| G.3 | à décider (gardé en dernier, profil 2 section 8) | B | `replay/film_scan.go` | nouveau profil |
| I.1 à I.3 | pilote ; Haiku, bas pour refiger les empreintes sous sa conduite | L, M | intégration | Gates V4, V3 (V1 abandonnée le 2026-11-23) |
| A.1 | Sonnet, bas | L | mesure, cause par film | 0.8 |
| A.2 | Sonnet, moyen | B | `document_score_actions*.go` | A.1 |
| A.3 | Sonnet, moyen ; reprise Opus seulement si la mesure désigne une lecture de grammaire non décodée (motif au journal) | L | mesure | 0.8 |
| A.3c | Sonnet, moyen | B | `reponsesdejeu/` | A.3 prouvée |
| A.4 | Sonnet, moyen | L | mesure | 0.8 |
| A.4c | Sonnet, moyen | B | `reponsesdejeu/`, `document_score_actions*.go` | A.4 prouvée, A.2 |
| A.5 | Sonnet, moyen | B | `score_timeline*.go` | A.2 |
| A.6 | Sonnet, bas | B | `reponsesdejeu/`, révision | A.5 |
| A.7 | Sonnet, bas | B | `replay/score_timeline_players.go` | Gate 0 |
| B.1 | Sonnet, élevé ; reprise Opus sur la même condition que A.3 | L | mesure, trois familles | 0.8 |
| B.2 | Sonnet, moyen, un item par cause prouvée | B | `flag_*.go`, `porteurs_*.go` | B.1 |
| B.3, B.7 | Sonnet, bas (deux mesures) | L | mesures | 0.8 |
| B.3c, B.7c | Sonnet, moyen | B | `flag_assign.go` ; `zone_states_owner*.go` | mesure prouvée, B.2 |
| B.4 | Sonnet, moyen | B | `internal/facts/objectives/named*.go` | B.2 |
| B.5 | Sonnet, moyen ; la mesure de sa condition au pilote | L, B | `porteurs_entrees.go` | B.2, B.4 |
| B.6 | Sonnet, moyen | B | `components_navpoint*.go`, `zone_states_lettres.go` | B.2 |
| C.1, C.2, C.3 | Sonnet, bas (trois mesures) | L | mesures d'accord | Gate 0 |
| C.1c, C.2c, C.3c | Sonnet, moyen | B | `frame_vue_controle.go`, `document_stances*.go` | mesure à 90 % |
| C.4 | Haiku, bas | S | table des recouvrements | P.1a |
| D.1 | Sonnet, moyen | B | `vue_a_genres.go`, `vue_a_lecture.go` | Gate 0 |
| D.2 | Sonnet, moyen (mesure sur les tables du jeu installé) | L | correspondance indice vers nom | D.1 |
| D.3 | Sonnet, moyen | B | `replay/document_repliques*.go` (neufs) | D.1 |
| D.4 | Haiku, bas | S | fragments | D.3 |
| D.5 | Sonnet, moyen | B | `LU cmd/` (table des répliques) | D.3 |
| D.6 | Sonnet, moyen | S | maquette interactive | D.5 |
| E.1 | Sonnet, bas (mesure) | L | compte des éliminations d'auteur bot | 0.8 |
| E.1c | Sonnet, moyen | B | `document_eliminations.go`, `eliminations.go` | E.1 |
| E.2 | Sonnet, moyen | B | `internal/facts/objectives/named*.go`, `score_timeline*.go` | A intégrée |
| E.3 | Sonnet, moyen | B | `film_variante_de_partie.go` | Gate 0 |
| E.4 | Sonnet, moyen (mesure) | L | lecture de l'apparence sur 3 films | 0.8 |
| E.4c | Sonnet, moyen | B | fichiers neufs | E.4 prouvée |
| E.5 | Haiku, bas | S | sort écrit | P.1a |
| E.6 | Sonnet, bas | B | `eliminations.go` | E.1c |
| F.1 | Haiku, bas | B | `LU scripts/briques-workspace.sh` | — |
| F.2 | Haiku, bas | S | `LU config/replay_corpus.toml` | — |
| F.3 | Haiku, bas ; le compte sur films au pilote | L, B | registre des replis | Gate 0 |
| F.4 | Haiku, bas | S | verdict écrit (base copiée, sans décodage) | 0.8 |
| F.5, F.6 | Sonnet, bas (deux mesures) | L | cause ou `[!]` | 0.8 |
| F.7 | Haiku, bas | L | verdict écrit | Gate 0 |
| P.1a, P.1b | Haiku, bas (décodeur ; deux autres briques) | S | `inventaire_contributeur.md` | — |
| P.2 | Sonnet, moyen | S | `accueil_contributeur.md` (anglais) | P.1 |
| P.3, P.4 | pilote | S, L | annonces, gel | vagues 1 à 7 |
| P.5 | Sonnet, moyen (relecture d'une PR) ; contrôles au pilote | S, L | une relecture par PR | P.4, PR reçue |
| P.6, P.7, P.8 | pilote ; Haiku, bas pour les refigeages guidés | L | intégration, tags | P.5 |
| R.1 | Fable | S | relecture finale | P.8 |
| R.2 | Sonnet, moyen | B | corrections P0, P1 | R.1 |
| L.1 à L.3, L.6 à L.9 | pilote ; Haiku, bas pour l'assemblage des fragments et les notes | L, M | publication | R.2 |
| L.4 | Sonnet, élevé (miroirs, plusieurs fichiers) ; Fable pour relire le diff de la montée | B | `LU` | L.3 |
| L.5 | pilote ; Haiku, bas pour les notes de version | L | gates de LevelUp | L.4 |

Ce qui ne tourne JAMAIS ensemble : deux items L ; un item M et quoi que ce soit ; deux items d'une même
piste de code (mêmes fichiers) ; une intégration et une autre. Ce qui tourne ensemble sans réserve : les
items S ; un item B de chaque piste parmi V, A, B, C, D, E, F, dans la limite de 4.

Fichiers attendus par piste (à confirmer en 0.6 ; deux pistes sur un même fichier = l'une après l'autre,
écrit au journal) :

| Piste | Écrit dans `DEC` (et `LU` quand indiqué) | Ne touche pas |
|---|---|---|
| V0 | `cmd/film-bench/` (neuf), `bench/` (neuf) ; V0.6 : `replay/options.go` (structure d'options ajoutée), `decfilm/` (V0.6 seul) ; lu : `internal/depottest/` (gate V0, à confirmer si l'on y écrit) ; LU : `cmd/replay-equiv`, `cmd/replay-build`, `cmd/replay-corpus-gate` (V0.7) | le reste |
| V1 | `internal/source/film.go`, `internal/source/source.go`, `internal/source/zstd.go`, `filmcache/` (lecture des chunks), `decfilm/` (une entrée de chargement ajoutée, `LoadDir` inchangée) ; lu : `replay/options.go` | `internal/grammar/`, `replay/` |
| V4 | `replay/film_scan*.go` sauf `replay/film_scan_images_cles.go` (V3), `replay/build_from_film.go`, `internal/grammar/movement_states.go`, appelants de `Distribuer` : `internal/grammar/equipement_porte.go`, `internal/grammar/lectures_bipedes.go`, `internal/grammar/objets_du_monde_lus.go`, `internal/grammar/marche_de_killsource.go` ; lus : `replay/observe.go`, `replay/layers.go` (écrivain unique : le pilote) ; V4.5 : fragments de `README.md` et `NOTICE.md` | `internal/source/`, `internal/grammar/distribuer*.go` (le distributeur lui-même) |
| V3 | `internal/grammar/marche_images_cles*.go`, `internal/grammar/keyframe_etat_complet_*.go`, `internal/grammar/keyframe_loadout.go`, `replay/film_scan_images_cles.go` (après V4) | `internal/grammar/marche_trames*.go`, `internal/grammar/keyframe_liaison.go`, `internal/source/` |
| V2 | `internal/grammar/marche_trames*.go`, `internal/grammar/marche_commune.go`, `internal/grammar/marche_de_killsource.go` (après V4), `internal/grammar/distribuer*.go` (glob : `distribuer_tetes.go` et `distribuer_crochets.go` inclus), `internal/grammar/world.go`, `internal/grammar/keyframe_liaison.go`, `internal/grammar/keyframe_anticipe.go` ; lu, écrit si l'ordre des étapes change : `replay/observe.go` | `internal/source/`, `replay/` (sauf `observe.go`) |
| A | `reponsesdejeu/`, `replay/document_score_actions*.go`, `replay/score_timeline*.go`, `replay/score_timeline_players.go` (`joinSegments`, A.5), `internal/facts/objectives/score.go` | drapeau, zones |
| B | `replay/flag_*.go` (dont `replay/flag_assign.go`), `replay/porteurs_*.go` (dont `replay/porteurs_entrees.go`, B.5), `replay/zone_states_*.go` (dont `replay/zone_states_lettres.go`), `internal/grammar/components_navpoint*.go`, `internal/facts/objectives/named*.go` (B.4, avant E.2) ; `replay/coverage.go` : fragment seulement, le pilote assemble (C15) | score |
| C | `internal/grammar/frame_vue_controle.go`, `replay/document_stances*.go` | vue A |
| D | `internal/grammar/vue_a_genres.go`, `internal/grammar/vue_a_lecture.go`, fichiers neufs `replay/document_repliques*.go` ; fragments de `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md` (R-15) ; LU : `cmd/` des répliques (neuf, à côté de `cmd/weapon-sounds`, D.5) | vue de contrôle |
| E | `replay/document_eliminations.go`, `replay/eliminations.go`, `internal/grammar/film_variante_de_partie.go`, `internal/facts/objectives/named*.go` (E.2, après A et B.4), `replay/score_timeline*.go` (E.2, après A.5) ; lus : `internal/facts/objectives/extract.go`, `decfilm/decfilm.go` (`ObjectiveTypeVip`) | drapeau |
| F | `internal/facts/fallback/` (`registre.go`, `compteur.go`, `noms.go`, `registre_filmdec.go`) ; LU : `scripts/briques-workspace.sh` (F.1), `config/replay_corpus.toml` (F.2) | — |
| Contributeur (copie publique, reçue par la piste P) | DEC, ajoutés (73) : `replay` 47, `internal/grammar` 17, `internal/facts/fallback` 4, `types` 2, `decfilm` 1, `filmcache` 1, `internal/grammar/lecture` 1. DEC, modifiés (249, contre v1.3.6) : `replay` 138, `internal/grammar` 61, `revision` 8, `filmcache` 4, `internal/facts/fallback` 3, `internal/facts/killsource` 3, `internal/facts/objectives` 3, `internal/source` 3, autres 26. LU cartes : ajoutés 195 (`himap` 191, `bitmapdecode` 4), modifiés 27 (`himap` 24, racine 3). LU lecteur : ajoutés 6 (`mapvar` 4, `himodule` 1, `ooz` LICENSE 1), modifiés 20 (`hinavmesh` 7, `mapvar` 6, `himodule` 5, `ooz` 2) | — |

**Conflits relevés le 2026-11-23** (contrôle 0.5 et 0.6, section 2.2). Règle commune : un fichier revendiqué par deux pistes passe l'une après l'autre, dans l'ordre des vagues, et l'ordre est écrit au journal.

- `replay/film_scan_images_cles.go` : V4 (motif `replay/film_scan*.go`) et V3. V4 d'abord (code en vague 3), V3 ensuite (vague 4) ; la ligne de V4 exclut ce fichier.
- `decfilm/` : V0.6 et V1. V0.6 d'abord (seul écrivain de la structure d'options) ; V1 n'y ajoute qu'une entrée.
- `internal/facts/objectives/named*.go` : B.4 et E.2. B.4 d'abord (vague 4), E.2 ensuite (vague 5).
- `replay/score_timeline*.go` : A.5 et E.2. A.5 d'abord, E.2 ensuite.
- `internal/grammar/marche_de_killsource.go` : V4 (appelant de `Distribuer`, ligne 104) et V2. V4 d'abord, V2 ensuite (V2.3, vague 6) ; la phrase de V4 « ne touche pas `marche_*.go` » est retirée.
- `FORMAT.md`, `GLOSSARY.md`, `NOTICE.md`, `README.md` : D (D.4) et V4 (V4.5). Aucun agent n'écrit ces fichiers communs (R-15) : chaque piste rend un fragment, le pilote assemble. La phrase de V4.5 « `FORMAT.md` n'est pas touché » reste vraie : seul le fragment de D.4 concerne `FORMAT.md`.
- `replay/coverage.go` : B.6 et l'écrivain unique (pilote). B.6 rend un fragment que le pilote assemble.

Conflits avec le contributeur (après le gel, piste P) : `internal/grammar/movement_states.go` (V4) ; `replay/film_scan*.go` (V4) ; `replay/document_stances*.go` (C) ; `internal/grammar/vue_a_lecture.go` (D) ; `internal/grammar/keyframe_etat_complet_bipede.go` (V3, par le glob) ; `internal/grammar/distribuer_tetes.go` (V2, par le glob) et `distribuer_crochets.go` (V2) ; `replay/options.go` (V0.6). Ordre : les pistes d'abord (vagues 1 à 7), gel (P.4), puis ses PR (P.7).

Dépendances entre pistes de vitesse, dites (V1 abandonnée le 2026-11-23 ; la piste G passe en premier) : V2 et V4 lisent la structure d'options de V0.6 et
aucune ne l'écrit ; V2 et V4 touchent toutes deux la distribution aux canaux (V4 les inscriptions, V2 le
distributeur) : V2 passe après V4 ; V3 passe après V4. Les pistes de vitesse passent donc une à une.
Le travail du contributeur croise V0.6, V1, V3, V4, C, D et tous les fichiers à écrivain unique : c'est
pourquoi il entre APRÈS le gel, sur une base qui ne bouge plus (piste P).

Fichiers à écrivain unique, le pilote, à l'intégration seulement : `replay/document_schema.go`,
`document_chronicle.go`, `layers.go`, `coverage.go`, `filmfacts*.go`, tous les goldens, les fichiers de
révision de couche, les références d'équivalence, les fixtures de contrat, et les fichiers communs de R-15.
Aucun agent ne monte le schéma ni ne régénère un golden : il rend la liste exacte de ce qu'il fait rougir.

## Étape 0 — témoins, références, instruments (pilote ; un agent Sonnet)

- [x] 0.1 (2026-11-23 : fait ; `main` = `v1.5.0` (`dc967c2`), branche `feat/v1.6.0` poussée, tête `7bda46f` ; worktree `Scripts/LevelUp-wt-dec160` sur `chore/830-decodeur-v160`, `npm ci` fait) Clone neuf `Scripts/dec-v160` à la tête de `main` ; vérifier `main` = `v1.5.0` (`dc967c2`) ;
      branche `feat/v1.6.0`, poussée. Worktree `Scripts/LevelUp-wt-dec160` sur `chore/830-decodeur-v160`,
      créée de la tête de la montée v1.5.0 (DD-5), sans jonction ; `npm ci` et espace de travail engendré
      (R-11) ; racines de données par piste préparées à la demande (R-13).
- [x] 0.2 (2026-11-23 : suite complète, tests de révision et équivalence à 20 identiques faits et verts ; gate du corpus lancé comme test d'infrastructure : code 0, 19 témoins ok, zéro gain, zéro perte, 464 s ; schéma 126 des deux côtés ; registre des replis lu, 127 entrées ; montage sous `Scripts/dec-v160-recup/gate/`) Marche de publication rejouée à blanc sur le clone SANS changement : G-clone sur `./...` (33
      paquets), G-rev, G-equiv-ref à 20 identiques. Le gate du corpus est lancé une fois, comme TEST
      D'INFRASTRUCTURE seulement (parc de travail, verrou, manifeste lus) : sans changement de `go.mod` il
      compare la v1.5.0 à elle-même et ne prouve rien sur les sorties.
- [x] 0.3 (fait le 2026-11-23 : le jeu se compose de films présents, 11 du corpus d'équivalence et les 2 témoins restants ; il couvre 7 familles de modes, 7 versions du film, de 13 à 69 morceaux, 5 films à plus de 24 joueurs ; aucun film à 33 joueurs ; table dans `.ai/annexes_decodeur_v160/films_du_releve.tsv`, note voisine `films_du_releve_note.md` ; le film de plus fort pic de mémoire reste à désigner par mesure, à la première fenêtre de mesure (candidat : le plus lourd, 69 morceaux)) Jeu de films fixe du relevé arrêté (V0.1) : présence de chaque film dans `HOTE` vérifiée ; le film
      de plus fort pic de mémoire du parc désigné par mesure ; correspondance écrite dans
      `.ai/annexes_decodeur_v160/films_du_releve.tsv`.
- [x] 0.4 (fait le 2026-11-23 :
      F. double passe trois fois : 20 artefacts identiques à l'octet à chaque passe ; 20 écarts « à classer »,
      toujours les mêmes, un par film, tous à l'étape `killsource` : écart ATTENDU et fixe, cause prouvée
      (champ `paquet` non écrit au fichier de faits), « killsource, 20 sur 20, jamais dans l'artefact » ;
      G. parcours de `map` : 412 relevés, 113 classés sans effet sans relecture un par un, 1 SUSPECT :
      `joinSegments` (`replay/score_timeline_players.go:188`, tri sur le seul instant de début) ;
      un item de correction est écrit en piste A, voir A.7.)
      Bruit de base du déterminisme, AVANT toute piste V : G-equiv-2p sur les 20 films, trois fois.
      L'écart « à classer : kill » est instruit jusqu'à sa cause ; `joinSegments` et tout parcours de `map`
      qui ordonne une sortie sont relevés, fichier et ligne. Sortie : zéro écart à classer en double passe
      sur la v1.5.0, ou un item de correction écrit ici (il changerait la sortie : il rejoint alors la
      vague 6, celle des intégrations du backlog, et la phase vitesse se prouve avec ce bruit nommé et borné à son étape).
- [x] 0.5 (2026-11-23 : références de la section 1.1 contrôlées et corrigées ; le compteur des émissions jetées au tag est contrôlé par le gate du corpus de la v1.5.0 (quatrième passage, 19 témoins ok) et par le passage à blanc du même jour) Contrôles sur pièces : le compteur des émissions jetées est juste hors Drapeau au tag
      (`e703301`) ; chaque référence `fichier:ligne` de la section 1.1 rouverte sur la tête du clone.
- [~] 0.6 (2026-11-23 : table des fichiers confirmée et corrigée ; étapes observées des tirs `fire`, `continuousFire`, des dégâts `damages`, `damages.stats`, des touches `gameEvents`, `gameEvents.stats` (lien des touches à confirmer) ; `gorelease` et `apidiff` absents du poste, `benchstat` présent ; essais faits : D, `replay-equiv -repo-root` sur une racine distincte ne marche que si elle porte aussi les trois dossiers de LevelUp (voir R-13) ; E, `gorelease` épinglé à `v0.0.0-20261121134650-e1094eeca1b5`, sortie à blanc « Suggested version: v1.5.1 », aucune incompatibilité ; reste à vérifier qu'il couvre les 19 paquets publics) Table des fichiers par piste confirmée sur la tête du clone, conflits écrits au journal ; noms des
      étapes de tirs, de touches et de dégâts de `replay-equiv` relevés (R-14) ; outil de G-compat essayé et
      sa version épinglée, ou repli retenu et capture `go doc -all` de `v1.5.0` rangée hors dépôt ;
      `replay-equiv -repo-root` essayé sur une racine de données distincte (R-13).
- [x] 0.7 (fait le 2026-11-23 : 21 sites de production, 5 lisent tout, 16 une partie ; table dans `.ai/annexes_decodeur_v160/appelants_partiels_0_7.md` ; à tracer par V4.1 avant la conception : que la génération complète consomme bien tous les canaux) Liste fermée des appelants de LevelUp qui ne lisent qu'une partie du film (candidats : rattrapage
      de la source de dégât, coup d'envoi du film, rasters tactiques), avec fichier et ligne : elle fixe ce
      que V4 doit servir et ce que L.4 branche. Écrite ici avant la vague 1.
- [x] 0.8 (fait le 2026-11-23 : copie de la base du 2026-11-22 (2 045 matchs), faits des 13 films exportés hors dépôt, mesures avant par piste dans `.ai/annexes_decodeur_v160/avant.md` ; les valeurs « parc » y décrivent le décodeur v1.4.1, le parc n'étant pas régénéré, l'état v1.5.0 par joueur est à reprendre en début de piste ; aucune vérité d'apparence des Spartans dans la copie, pour E.4) Vérité de l'API et faits des matchs : export unique en lecture sur une COPIE de la base partagée,
      hors dépôt, pour les pistes A, B, E et pour le relevé B (`levelup replay-facts-export` sur les 13
      films). Mesures AVANT par piste dans `.ai/annexes_decodeur_v160/avant.md`.
- **Gate 0** : 0.2 vert (G-clone, G-rev, 20 identiques) ; 0.4 statué ; 0.6 et 0.7 écrits ; `avant.md`
  complet. Aucun relevé de vitesse n'est exigé ici.

Gate 0 PASSÉ le 2026-11-23, avec une réserve : la couverture de tous les paquets publics par l'outil de compatibilité (0.6) reste à vérifier, au premier gate de piste.

## Piste V0 — relevé de vitesse par version, socle des options (Sonnet)

Deux relevés, parce que la génération d'un rejeu dépend de la base de LevelUp, absente du dépôt public.

- [x] V0.1 (2026-11-23 : fait et intégré dans `feat/v1.6.0` ; commande `cmd/film-bench`, `bench/films.tsv` à noms neutres) Décision du 2026-11-23 : le jeu ne reprend que 2 des 10 témoins de durée, les autres étant absents de l'archive ; pas de film à 33 joueurs.
      Jeu de films fixe, 13 films, critères écrits dans `bench/README.md` : les dix témoins de durée du
      2026-11-10 (builds `HI_1_12_0` et `HI_1_13_0` ; Slayer, Drapeau, Bases, Roi de la colline, Balle,
      Grande bataille à 33 joueurs ; 25 à 67 chunks) ; un film `HI_1_11_0` de Grande bataille et un film de
      version 39 ou 40, pris dans les 20 de l'équivalence ; le film de plus fort pic de mémoire du parc
      (0.3). `bench/films.tsv` : nom neutre, build, mode, module de carte, joueurs, chunks, octets
      compressés. Aucun fichier du jeu, aucun chunk, aucun identifiant de joueur ni de match dans le dépôt
      public. Second jeu, « public » : les mini-bobines du dépôt, pour que la commande tourne chez quiconque
      et en CI ; sans catalogue de bornes dans le dépôt, il ne mesure que le chargement (à établir ici).
- [x] V0.2 (2026-11-23 : fait et intégré dans `feat/v1.6.0` ; commande `cmd/film-bench`) **Relevé A, dans le décodeur.** Commande `cmd/film-bench`, bâtie sur `research/duree`, par les
      SEULES entrées publiques, options MINIMALES (carte seule) :
      `GOWORK=off go run ./cmd/film-bench -films bench/films.tsv -correspondance <fichier hors dépôt>
      -racine <film_chunks> -catalogue <map_quant_bounds.json> -passes 5 -fils 1 -sortie bench/releves/<version>.tsv`.
      Phases, lues par l'observateur existant : **chargement** (lecture, décompression et découpage,
      ensemble), **décodage** (du film aux faits, jusqu'à l'étape `clockOrigin`), **publication** (des faits
      au document), **total**, **pic de mémoire**. La **décompression** est chronométrée à part, par
      `decfilm.Inflate` sur les octets de chaque chunk lus avant le chronomètre : c'est une colonne à côté,
      pas une part du chargement. Écrit dans `bench/README.md` : le total du relevé A N'EST PAS le temps
      d'une génération de rejeu (ni source de dégât, ni tableau, ni objectifs, ni zones).
- [x] V0.3 (2026-11-23 : fait et intégré dans `feat/v1.6.0`, avec l'option de profilage) Protocole de mesure, écrit dans `bench/README.md` : fichiers lus avant le chronomètre ; un
      PROCESSUS NEUF par film pour le pic de mémoire (sentinelle de `memguard`, plafond de mesure de 2 Gio) ;
      une passe d'échauffement, puis 5 passes ; ordre des films tourné d'une passe à l'autre ; empreintes
      SHA-256 du document et des faits calculées HORS chronomètre ; si deux relevés consécutifs diffèrent de
      plus de 5 % sur le total médian d'un film, 9 passes, puis arrêt : la dispersion est écrite telle
      quelle. Format : TSV, une ligne par film et par niveau de fils (médiane et minimum de chaque phase,
      pic, empreintes, compte de relectures en séquence), en-tête commenté (version, commit, version de Go,
      système, processeur, cœurs, mémoire, `GOMAXPROCS`, date).
- [ ] V0.4 **Relevé B, de bout en bout, dans LevelUp.** `replay-build` sur les 13 films, faits de 0.8,
      `-fils 1` : durée par film (5 passes, médiane) et pic publié par la commande ; rangé
      `.ai/annexes_decodeur_v160/releve_bout_en_bout.md`. C'est le chiffre d'une génération de rejeu
      (référence du 2026-11-13 : 3,5 s en médiane, 10,1 s au pire sur dix témoins).
- [x] V0.5 (2026-11-23 : fait ; relevé de référence de la v1.5.0 dans `bench/releves/v1.5.0.tsv` (13 films, 5 passes) ; profil sur cinq films dans `.ai/annexes_decodeur_v160/profil.md` : décodage plus de 90 %, publication 5 à 10 %, décompression moins de 0,5 %, chargement moins de 1 %) Relevés de RÉFÉRENCE de la v1.5.0, par le pilote, en fenêtre de mesure. Relevé A : l'outil est
      copié, sans être commité, dans un worktree posé sur le tag `v1.5.0` (il n'emploie que des entrées
      publiques de cette version) ; rangé `bench/releves/v1.5.0.tsv`. Relevé B : branche de montée au
      `require` v1.5.0. Profil du temps et des allocations sur 3 films (`-cpuprofile`, `-memprofile` de
      `replay-build`) : part de chaque phase et des dix premières fonctions, dans
      `.ai/annexes_decodeur_v160/profil.md`. C'est l'entrée des conceptions V1.1 à V4.1.
- [x] V0.6 (2026-11-23 : fait et intégré dans `feat/v1.6.0` : porteur `OptionsVitesse`, un fil par défaut, borne `MaxFils` = 8 ; `gorelease` : ajouts compatibles, version suggérée v1.6.0) Socle des options (R-16), un seul écrivain : une structure d'options AJOUTÉE (nombre de fils,
      choix des canaux), valeur zéro = comportement actuel, sans aucun effet à ce stade ; plafond du nombre
      de fils dans la brique (R-12). Intégrée dans `feat/v1.6.0` avant toute piste V. Réexport par la façade retiré le 2026-11-23 (cycle d'import sous l'étiquette de recherche) : le porteur vit dans `replay` seul.
- [ ] V0.7 (décalé le 2026-11-23 : le niveau de fils dans les outils de LevelUp n'a de sens que lorsqu'une entrée du décodeur accepte les options ; il se fera avec la première piste qui en lit une) Côté LevelUp, sur la branche de montée (pseudo-version de V0.6) : niveau de fils réglable par
      `-fils` dans `replay-equiv` (transmis à ses enfants) et `replay-build`, par `--fils` dans le gate du
      corpus (cuisson de la tête seulement) ; défaut 1 ; tests des trois lignes de commande.
- Rangement « avec chaque version » : `bench/releves/vX.Y.Z.tsv` commité dans la PR de la version ; tableau
  des médianes par version dans `bench/README.md` ; même tableau, avec le relevé B, dans les notes de la
  release GitHub. Geste 11 bis ajouté à la marche de publication.
- **Gate V0** : G-clone sur `./cmd/film-bench/ ./internal/depottest/ ./replay/ ./decfilm/` (au moins un
  test `PASS` par fichier de test neuf, comptés dans le rendu) ; G-rev ; G-compat ; G-façade ; la commande
  tourne sur le jeu public sans donnée de l'hôte ; deux relevés A consécutifs de la v1.5.0 tiennent le
  protocole V0.3, empreintes identiques ; G-equiv-ref avec `-fils 1` : 20 identiques ; `cd LU/apps/go-api &&
  go test -count=1 -v ./cmd/replay-equiv/ ./cmd/replay-build/ ./cmd/replay-corpus-gate/`.

## Piste V1 — décompression sur plusieurs fils (Sonnet)

ABANDONNÉE le 2026-11-23 sur mesure : gain mesuré de 0,1 à 0,8 % du total (profil, section 4, axe 1) ; même à coût nul, le gain reste plafonné à ce chiffre.

- [!] V1.1 (abandonné, mesure du 2026-11-23 : décompression de 0,1 à 0,8 % du total) Conception écrite ICI avant le code, après lecture de la méthode du parseur voisin
      (`src/theater/parser/chunk.rs`) et du profil V0.5 : chunks rangés par indice ; erreurs rendues dans
      l'ordre des chunks (la première par indice, pas la première arrivée) ; tolérances de `inflate`
      inchangées (entrée claire, flux tronqué, zstd abîmé) ; au plus autant de chunks en cours de
      décompression que de fils.
- [!] V1.2 (abandonné, mesure du 2026-11-23) Code : `source.Load`, `decfilm.Load` et `decfilm.LoadDir` gardent signature et comportement ; une
      entrée de chargement AJOUTÉE lit les options de V0.6. Le découpage en paquets reste dans l'ordre.
- [!] V1.3 (abandonné, mesure du 2026-11-23) Tests neufs sur les mini-bobines (noms contenant `Fils`) : octets clairs et paquets égaux à un
      fil et à N fils ; erreur d'un chunk du milieu égale dans les deux cas ; ordre de fin perturbé par un
      délai aléatoire ; sous `-race`.
- [!] V1.4 (abandonné, mesure du 2026-11-23) Mesure par le relevé A aux niveaux 1, 2, 4 et `N` : chargement, décompression, total, pic ; par
      le relevé B au niveau retenu.
- **Gate V1** : preuve d'identité (points 1 à 6) ; G-clone sur `./internal/source/ ./filmcache/ ./decfilm/
  -run Fils -race -count=3` (au moins 3 tests `PASS`) ; G-rev (rouge attendu sur la couche `source` et, par
  sa valeur amont, nulle part ailleurs : liste rendue au pilote) ; G-compat ; au niveau 1, pic et total dans
  le bruit du relevé v1.5.0.

## Piste V4 — lecture au choix des canaux (Sonnet)

- [x] V4.1 (2026-11-23 : conception rendue ; ABANDON de la piste décidé par le user le 2026-11-23, « ok on abandonne le choix des canaux » ; détail : `.ai/annexes_decodeur_v160/v4_conception.md` ; elle déconseille le chantier pour la v1.6.0 : environ 73 % du temps est incompressible quel que soit le choix, le gain d'une sélection est de 22 à 30 %, aucun appelant de LevelUp ne choisirait de sélection, et le chantier touche les mêmes fichiers que le parallélisme ; décision demandée au user, V4.2 à V4.4 suspendus) Conception écrite ICI avant le code : le choix des canaux de V0.6 (valeur zéro = tous les
      calques = comportement actuel) ; table des dépendances entre calques, tirée de `layers.go` et des
      étapes de `observe.go` (un calque demandé entraîne ceux dont il se nourrit : identité, pont, fil des
      morts) ; ce que la marche peut ne pas distribuer quand aucun calque demandé n'y a d'intérêt. Les
      appelants à servir sont ceux de 0.7.
- [!] V4.2 (abandonné le 2026-11-23, décision du user sur la conception V4.1 : gain de 22 à 30 % pour une sélection que nul appelant ne demande, conflit de fichiers avec le parallélisme) Code : clôture des dépendances, canaux non demandés non inscrits à la marche. Le document partiel
      porte les MÊMES octets pour les calques demandés ; les calques non demandés sont absents, et `layers`
      le dit par ce qui existe déjà (aucun champ neuf).
- [!] V4.3 (abandonné le 2026-11-23, avec V4.2) Garde-rail : un test échoue si un calque est ajouté à `layers.go` sans ligne dans la table des
      dépendances.
- [!] V4.4 (abandonné le 2026-11-23, avec V4.2) Mesure par le relevé A : document complet (inchangé) ; puis positions seules, tirs seuls, fil des
      morts seul : décodage, publication, total, pic.
- [~] V4.5 (2026-11-23 : V4 abandonnée ; le crédit à nuzzles est porté par L.3 (notes de la release) et L.5 (crédits de LevelUp)) Crédit à nuzzles, fragment rendu pour les pistes V (R-15) : `README.md` (section « Speed »,
      méthodes reprises, lien vers son dépôt), `NOTICE.md` (remerciement, licence de sa source, révision
      lue), une ligne en tête de chaque fichier qui porte une méthode reprise, notes de la release, et côté
      LevelUp les notes de version et la page des crédits (L.5). `FORMAT.md` n'est pas touché : le format ne
      change pas.
- **Gate V4** : preuve d'identité pour le document complet (points 1 à 3, 6) ; pour chaque sélection
  mesurée, les empreintes par étape de `replay-equiv -out-dir` des calques demandés sont égales à celles du
  document complet sur les 20 films ; G-clone sur `./replay/ -run 'Canaux|Calques'` (au moins 3 tests
  `PASS`, dont le garde-rail V4.3) ; G-rev ; G-compat.

## Piste V3 — lectures refaites et état à la demande, réduit par DD-7 (Sonnet)

- [~] V3.1 (2026-11-23 : couvert par les trois profils `profil.md`, `profil2.md`, `profil_films_recents.md` et par la piste K ; aucun candidat hors de ces pièces) Instruction sans code : liste fermée des états construits d'office pour des lecteurs qui ne les
      demandent pas, et des lectures refaites, avec leur coût dans le profil V0.5 et leurs lecteurs ;
      méthode du parseur voisin relue (`src/theater/runtime/mod.rs`) ; motif d'abandon de sa forme littérale
      écrit (DD-7). Sortie : candidats écrits ici.
- [!] V3.2 (abandonné le 2026-11-23 : déjà mémoïsé dans le code, 29 recherches pour 556 appels sur un film ; le chiffre de 30 % du registre des reports est périmé) Images-clés : une seule recherche pour les trois lecteurs (état complet du bipède, dotations de
      naissance, équipes).
- [!] V3.3 (2026-11-23 : la partie équipement est gardée, reprise par G.7 ; la sonde de calibration n'est pas retenue : 1,5 % sur film01 seulement, moins de 1 % ailleurs, décision du pilote ; `[!]` à poser avec ce chiffre) Sonde de calibration et récupération d'équipement passées derrière la marche, seulement si le
      profil V0.5 les montre au-dessus de 1 % du total ; sinon `[!]` avec le chiffre.
- [~] V3.4 (2026-11-23 : couvert par les pistes G et K) Les candidats de V3.1 retenus par le pilote, un item écrit ici par candidat avant son code ; un
      candidat qui toucherait l'état du monde de la marche est refusé ici (il appartient à V2).
- [~] V3.5 (2026-11-23 : couvert par K.I5 et K.3) Mesure par les relevés A et B : décodage, publication, total, pic.
- **Gate V3** : preuve d'identité (points 1 à 4, 6 ; un seul niveau si la piste ne lance aucun fil) ;
  G-clone sur `./internal/grammar/ ./replay/ -run 'ImagesCles|Keyframe'` (nombre de `PASS` au moins égal à
  celui de la tête avant la piste, relevé par l'agent) ; G-rev ; G-compat.

## Piste G — gains du profil (Sonnet, effort moyen ; ajoutée le 2026-11-23)

Origine : le profil de vitesse (`.ai/annexes_decodeur_v160/profil.md`, sections 3 à 5) montre que, sur les films dont la vue A n'est pas portée (anciennes versions du film), la localisation de la liste d'événements passe par un balayage bit à bit avec décodage d'essai, refait trois fois par trame (marche, canal des morts, canal des vitalités) : 73,5 millions de décodages d'essai sur un film ancien, contre 175 sur un film récent de même taille, qui est 8 fois plus rapide. Aucun item G ne change la sortie : chacun se prouve par ses empreintes.

Ordre de lancement : G.2, G.6 et G.7 en parallèle (lancés le 2026-11-23 : agents Sonnet à effort moyen, branches `piste/g2`, `piste/g6`, `piste/g7`, un worktree par agent) ; G.1 après G.2, même zone de code. G.4, G.8, G.5 et G.3 : à décider sur un nouveau profil.

Preuve commune à chaque item : empreintes de `film-bench` identiques au relevé de la v1.5.0 sur cinq films, puis preuve d'intégration (G-equiv-ref). Les chemins de fichiers viennent du profil : à rouvrir à l'exécution.

- [x] G.2 (2026-11-23 : intégré dans `feat/v1.6.0`, empreintes du document et des faits identiques sur les 13 films) Localiser la liste une seule fois par trame non localisée, pour les deux canaux (morts, vitalités), au lieu d'une fois par canal. (lancé le 2026-11-23, branche `piste/g2`)
      Gain attendu : 9 à 12 % sur les films anciens (film06 2,4 s, film09 1,35 s, film04 2,4 s, film12 0,46 s). Risque : très faible (même résultat ; mémoire par paquet). Fichiers : `canal_des_morts.go` (`recupererLaListe`, appels lignes 85 et 160-177), `canal_des_vitalites.go` (ligne 84). Preuve : celle de la piste.
- [x] G.1 (2026-11-23 : intégré dans `feat/v1.6.0` ; preuve : test différentiel, 15 564 acceptations sur 10 310 payloads et 29 049 trames de deux films réels ; relevé des 13 films après les cinq gains, voir le bilan) Ne décoder un delta d'essai que si son identifiant peut convenir (slot 123 ou génération du monde), lu dans l'en-tête avant le décodage. Après G.2 (même zone de code).
      Gain attendu : 14 à 24 % (film06 -17 à -24 %, film09 -18 à -23 %, film04 -14 à -18 %, film12 -17 à -20 % ; film01 : 0). Risque : faible à moyen (même booléen ; à prouver par comparaison des positions rendues sur le corpus). Fichiers : `localisateur.go` (`marchSignature123`, `marchLocateFallback`), `frame_records.go` (`TryDeltaAt`). Preuve : celle de la piste, puis comparaison des positions rendues sur le corpus.
- [x] G.6 (2026-11-23 : intégré dans `feat/v1.6.0`, empreintes du document et des faits identiques sur les 13 films) Ne pas tripler la suite des positions au plus haut : filtrage en place dans une variante interne ; les fonctions exportées gardent leur contrat. (lancé le 2026-11-23, branche `piste/g6`)
      Gain attendu : 0,4 à 0,6 Gio de pic en moins sur le plus gros film (film04) ; temps : environ 1 %. Risque : faible. Fichiers : `offline_filters.go` (`DropIsolated`), `offline_biped_band.go` (lignes 147-151), `positions_lues.go` (`lireLesPositions`, pré-dimensionnement). Preuve : celle de la piste.
- [!] G.7 (2026-11-23 : retiré de l'intégration sur mesure, motif mesuré : −4 à −5 % sur le plus gros film seulement, +2 % reproductible sur un film ancien ; branche `piste/g7` gardée) Tester slot et génération avant de lire la poignée complète à chaque bit, dans la récupération d'équipement. (lancé le 2026-11-23, branche `piste/g7`)
      Gain attendu : 5 à 7 % sur film04 (`LireHandleDelta` : 1,15 s sur 2,57 s). Risque : faible. Fichiers : `equipment_recovery.go` (`scanEquipRecoveryPacket`, lignes 210-260). Preuve : celle de la piste.
- [x] H.1 (2026-11-23 : intégré dans `feat/v1.6.0`) Calculer la largeur d'en-tête d'un bipède après les rejets de tête, et non avant. Preuve : test différentiel contre l'ancienne forme gardée en référence, plus de 900 000 occurrences ; empreintes des 13 films identiques à la v1.5.0. Gain : voir G.8'. Fichiers : `offline_biped.go` (`matchBipedHeader`).
- [x] G.8' (2026-11-23 : intégré dans `feat/v1.6.0`, remplace G.8) Préfiltre exact des mots de 32 bits d'une image-clé, à la place d'une recherche en table à chaque bit. Preuve : test différentiel, plus de 900 000 occurrences ; empreintes identiques. Gain mesuré avec H.1 : −4,7 % sur cinq films, dont −5 à −8 % sur les films récents. Fichiers : `keyframe_loadout.go` (`motsParRecord`).
- [x] G.4 (2026-11-23 : intégré dans `feat/v1.6.0` ; preuve : test différentiel contre la forme bit à bit, empreintes du document et des faits identiques sur les 13 films) Avance par mot (saut des bits nuls) dans les balayages bit à bit. Risque moyen : à prouver par empreintes identiques.
      Gain attendu : -5 à -9 % sur film06 et film09. Risque : moyen (la grammaire avance d'un bit : l'équivalence se prouve par les mêmes sorties). Fichiers : `delta_biped_walk.go` (lignes 91-107), `offline_biped.go` (`matchBipedHeader`, ligne 318), `offline_aim_only.go` (ligne 110). Preuve : celle de la piste.
- [!] G.8 Index de slots par tableau plutôt que par `map[uint32]`. **Abandonné, remplacé par G.8'** (profil 2 : le coût était dans `motsParRecord` et dans la copie de la table des slots, traités par G.8' et G.5).
      Gain attendu : -4 à -6 % sur film01 et film12. Risque : faible ; effort moyen. Fichiers : `keyframe_loadout.go` (`motsParRecord`, ligne 164) et les accès `mapaccess1_fast32` (à localiser). Preuve : celle de la piste.
- [x] G.5 (2026-11-23 : intégré dans `feat/v1.6.0`) Ne plus copier la table des slots à chaque essai : un essai se défait par un journal d'annulation ; toute écriture de la table passe par deux primitives, et une garde lit les sources. Preuve : test différentiel contre l'ancienne forme gardée en référence, 1 520 803 essais de deux films réels dont 609 422 modifiant le monde, état complet comparé ; empreintes identiques. Gain mesuré : −10 % sur le plus gros film, −29,5 % d'octets alloués. Fichiers : `world.go` (`cloneSlots`, `Snapshot`, `Restore`), `debut_de_liste.go` (`lectureDEssai`).
- [~] G.3 (2026-11-23 : repris par la piste K, chantier 2) Faire tourner en parallèle, entre eux et d'après la marche, les balayages hors marche ; `decodeFilmVehicleScan` avec `decodeFilmPlacements` dans `balayerMonde`. Sans effet attendu à un fil (DD-3). **À décider sur un nouveau profil.** Note (profil 2, section 8, 2026-11-23) : possible mais PAS tel quel (contexte de film muté, mémos sans verrou) ; gardé en dernier, à décider.
      Gain attendu : -8 à -20 % de durée murale à plusieurs fils (film06 : de 38 % à environ 19 %, borné par `decodeFilmVehicleScan` ; film01 : environ 20 %). Risque : moyen (état partagé du `FilmContext`, caches et verrous à vérifier ; les crochets publient). Fichiers : `replay/film_scan.go` (`balayerPositions`, `balayerMonde`, `balayerPont`, `balayerCapacites`). Preuve : celle de la piste.

- [x] G.10 (2026-11-23 : intégré, `da3bd2c` ; positions par rangs, empreintes des 13 films identiques ; pic du plus gros film −294 Mio, contre −0,4 à −0,5 Gio visés) mémoire : `indexBySlot` sans copie des positions (index de positions de 4 octets au lieu d'une copie de 176 octets par position ; 37 usages de `.pts` dans 11 fichiers de `replay/`). Gain attendu : pic −0,4 à −0,5 Gio sur le plus gros film, alloué −4,8 Gio. Risque moyen ; preuve : empreintes des 13 films. Source : `.ai/annexes_decodeur_v160/profil2.md`, section 6.2 (M1).
- [ ] G.11 : note à l'hôte sur le plafond souple de mémoire (`-plafond-gio`) : à 1 Gio, le pic du plus gros film tombe à 1 161 Mio pour +1,3 % de temps (mesuré sur ce seul film). Réglage de l'hôte, pas du décodeur ; à consigner dans la mise en production. Source : `.ai/annexes_decodeur_v160/profil2.md`, section 6.2 (M0).

Gate de la piste G : G-clone sur les paquets touchés (`-race` pour G.3) ; G-rev, empreinte refigée par le pilote à valeur constante (R-13) ; G-compat ; G-equiv-ref : 20 identiques contre les références de la v1.5.0 ; G-equiv-2p ; relevé complet de `film-bench` sur la machine au repos (fenêtre de mesure, R-8), comparé à `bench/releves/v1.5.0.tsv`.

**Bilan au 2026-11-23 (cinq gains intégrés dans `feat/v1.6.0`, tête `4618ce3`).** Preuve de l'état intégré : suite complète, code 0, 34 paquets, 301 tests sautés, aucun échec ; `go vet` avec et sans l'étiquette de recherche ; lint 0 issue ; équivalence : 20 identiques contre la v1.5.0 ; double passe : 20 artefacts identiques à l'octet, seuls écarts à classer les 20 de l'étape `killsource` ; relevé des 13 films, 5 passes, machine seule : 26 empreintes sur 26 identiques à la v1.5.0 ; empreinte de révision de la grammaire refigée à valeur constante (`grammar-2026-11-22.2`). Gain, tête contre le relevé de référence de la v1.5.0 (pris sous charge) : décodage cumulé 68,2 s → 44,6 s (−34,5 %), total cumulé 71,6 s → 48,3 s (−32,5 %) ; plus gros film 19,6 s → 12,2 s ; film ancien 18,1 s → 10,6 s ; Total Control ancien 9,7 s → 5,9 s ; crâne 4,0 s → 2,5 s ; une Grande bataille récente 2,3 s → 2,2 s. Pic du plus gros film : 2 198 → 1 969 Mio. Relevé rangé dans `.ai/annexes_decodeur_v160/releve_apres_cinq_gains.tsv`.

**Bilan au 2026-11-23 (six gains intégrés, tête `785c38e`).** Mêmes preuves que le bilan précédent, rejouées : suite complète code 0, 34 paquets ; équivalence 20 identiques et double passe identique ; 130 empreintes sur 130 identiques à la v1.5.0 (13 films, 5 passes, machine seule). Décodage cumulé 68,2 s → 34,7 s (−49 %) ; total 71,6 s → 38,3 s ; film le plus ancien 18,1 s → 5,8 s ; Total Control ancien 9,7 s → 3,3 s ; plus gros film 19,6 s → 10,7 s (pic 2 198 → 2 053 Mio) ; Slayer récent 1 123 → 1 000 ms. Relevé : `.ai/annexes_decodeur_v160/releve_apres_six_gains.tsv`.

## Piste K — films récents : parallélisme (ajoutée le 2026-11-23)

Origine : le user a fixé la priorité le 2026-11-23 : la vitesse compte surtout sur les matchs neufs, donc sur la dernière version du film ; un gain majeur sur un fil se prend, sinon le parallélisme. Seuil du pilote : au moins 15 % du total sur au moins trois films récents sur cinq.

Mesure (`.ai/annexes_decodeur_v160/profil_films_recents.md`) : sur un film récent, 99,996 % des trames sont lues jusqu'au bout et aucune ne part en recherche de repli ; les leviers des films anciens valent donc 0. Le meilleur gain sur un fil vaut 4 à 7 % ; additionnés, sept petits gains feraient 12 à 20 %. **Pas de gain majeur sans parallélisme.** Quatre parcours des images-clés, fonctions du payload, pèsent environ 40 % du temps ; les balayages d'après la marche, 36 à 45 %, équilibrés. Le parallélisme par morceau de la marche a été mesuré à blanc : il passe (70 à 100 % des morceaux) mais ne vise que 22 à 25 % du total, plafonne à −9 à −17 %, et porte le plus de risque : la piste V2 reste abandonnée, cette fois sur mesure.

Ordre : K.0 et K.C ensemble ; puis les items de K.1 et K.2 tels que la conception relue les découpe, écrits ICI avant leur code.

- [x] K.0 (2026-11-23 : intégré dans `feat/v1.6.0`, tête `daac986` ; preuve : test différentiel contre la forme bit à bit sur 823 millions de positions de payloads réels, cas construits aux frontières, empreintes du document et des faits identiques à la v1.5.0 sur les 13 films ; empreinte de la grammaire refigée à révision constante) Rejet par mot de 64 bits dans la table des datums et dans les en-têtes de joueur. Durée à mesurer machine seule (K.3).
- [x] K.C (2026-11-23 : rendue, relue par le pilote, rangée dans `.ai/annexes_decodeur_v160/parallelisme_conception.md`) Conception. Elle fait foi pour le détail des items ci-dessous (sa section 5) ; elle a été écrite sur la tête `785c38e`, les agents partent de `daac986` (K.0 a sorti `slotsDEntetesExacts` de la liste des balayages nommés).

Constats de la conception : les quatre parcours des images-clés sont des fonctions pures ; deux seulement ont déjà un mémo ; les largeurs posées puis restaurées sur le contexte ne gênent pas le chantier 1 ; le pré-passage part au premier accès aux images-clés, là où la preuve des ancres se fige aujourd'hui, et range ses résultats depuis le fil appelant, sans verrou sur un chemin de lecture. Entrée publique : `FilmContext.PoserFils(n)`, nourrie par un champ `Vitesse` de `replay.Options`. À un fil : le chemin d'aujourd'hui, aucun fil lancé, garanti par un test. Le chantier 2 se scinde : 2a, les lectures qui ne dépendent que du film, tournent pendant la marche (gain estimé −8 à −10 %, risque faible) ; 2b, les voies sur un contexte dérivé, se décide après la mesure du chantier 1.

Décisions du pilote (2026-11-23) : chantier 1 et chantier 2a lancés ; chantier 2b (J3 à J10) décidé sur la mesure I5, avec le user (il vise deux hôtes : un serveur à peu de cœurs et l'app de bureau) ; le nombre de fils est toujours choisi par l'hôte, jamais par le décodeur ; `killsource` ne reçoit pas de champ propre tant que l'hôte partage le contexte (à vérifier dans LevelUp avec V0.7).

Fondations, sans changement de comportement (un agent Sonnet, à la suite, branche `piste/k-fondations`). Preuve de l'état intégré avec K.J1 (tête `2f22fb2`) : `go vet` avec et sans l'étiquette de recherche, tests des paquets touchés code 0, compatibilité publique contre la v1.5.0 (ajouts seulement : `Options.Vitesse`, `OptionsVitesse`, `MaxFils`), empreintes du document et des faits des 13 films identiques à la v1.5.0 à 1 et à 4 fils demandés (rien ne lance encore de fil). Écart à la conception : l'appel passe par un assistant `poserLesFils`, testé à part. Limite connue de `-etapes` : trois étapes par film ne s'écrivent pas en JSON, leur ordre est prouvé, leur contenu ne l'est que par l'empreinte du document :
- [x] K.F1 (2026-11-23 : intégré dans `feat/v1.6.0`) Assistant de fils (`internal/grammar/fils.go`) : exécute dans l'appelant à un fil, relaie une panique, seule source de goroutines du module (garde-rail).
- [x] K.F2 (2026-11-23 : intégré dans `feat/v1.6.0`) Porteur sur le contexte : `PoserFils`, `Fils`, défaut 1.
- [x] K.F3 (2026-11-23 : intégré dans `feat/v1.6.0`) Champ `Vitesse` de `replay.Options`, posé avant le profil et la carte.
- [x] K.F4 (2026-11-23 : intégré dans `feat/v1.6.0`) `film-bench -fils N` et `-etapes` (suite ordonnée des étapes et de leurs empreintes : preuve de l'ordre publié).

Chantier 1, pré-décodage des images-clés (après les fondations). Preuve de l'état intégré avec K.J2 (tête `bf69129`) : `go vet` avec et sans l'étiquette de recherche ; tests des paquets de la grammaire, du rejeu, des faits, de la façade, de l'outil et des gardes du dépôt, code 0 ; détection de courses sur les tests neufs, trois passes, code 0 ; empreintes du document et des faits des 13 films identiques à la v1.5.0 à 1, 2, 4 et 8 fils, plus trois passes à 8 fils ; suite ordonnée des étapes identique à 1, 2, 4 et 8 fils ; compatibilité publique : ajouts seulement. Durées non mesurées (K.I5) :
- [x] K.I1 (2026-11-23 : intégré ; le pré-passage part au premier accès aux images-clés, range depuis le fil appelant ; la liste des payloads vit dans `marche_images_cles.go`, seul fichier que le garde-rail autorise à choisir les paquets d'image-clé) Cadre du pré-passage et ancres.
- [x] K.I2 (2026-11-23 : intégré ; même clé que le lecteur existant) Table des datums.
- [x] K.I3 (2026-11-23 : intégré ; mémo neuf, consulté par le canal des équipes, jamais écrit hors du pré-passage) En-têtes exacts de joueur (mémo neuf).
- [x] K.I4 (2026-11-23 : intégré ; lemme des records muets prouvé par test ; calcul fait pour tout payload à bipède, le travail en trop reste à mesurer en K.I5) Mots par record, au début du balayage des états (mémo dans le canal) ; lemme des records muets prouvé par test.
- [x] K.I5 (2026-11-23 : mesuré, 13 films, 5 passes, machine au repos, aucun test d'une autre session pendant les passes ; relevés `.ai/annexes_decodeur_v160/releve_parallelisme_fils{1,2,4,8}.tsv` ; 52 empreintes identiques à la v1.5.0) Mesure à 1, 2, 4 et 8 fils et pic de mémoire.
      Total cumulé des 13 films : v1.5.0 71,6 s ; un fil 34,8 s (−51 %) ; deux fils 29,0 s (−59 %) ; quatre fils 27,0 s (−62 %) ; huit fils 26,2 s (−63 %).
      Films récents, total par film, v1.5.0 puis 1 / 2 / 4 / 8 fils : Slayer 1 123 ms → 826 / 622 / 570 / 550 ; Bases 1 586 → 1 191 / 891 / 803 / 766 ; Colline 1 371 → 1 048 / 776 / 705 / 675 ; Drapeau d'arène 525 → 399 / 271 / 247 / 236 ; Bombe 1 197 → 903 / 648 / 579 / 547. Soit, sur un film récent, environ −26 % à un fil (effet de K.0), −45 % à deux fils, −50 % à quatre.
      Films anciens : le plus ancien 18,1 s → 5,5 s à un fil, 4,6 s à quatre ; le plus gros 19,6 s → 10,1 s à un fil, 8,8 s à quatre.
      Pic de mémoire : inchangé d'un niveau de fils à l'autre (plus gros film 1 965 à 2 022 Mio contre 2 198 en v1.5.0 ; film récent 235 à 256 Mio). Le gain plafonne après quatre fils. La publication ne bouge pas (environ +4 % contre le relevé de référence, à contrôler au relevé final).
- [x] K.I6 (2026-11-23 : fait, `41389c4` ; le job de courses couvre `^TestFils|^TestVoie|^TestExecuterEnParallele`, `bench/README.md` dit que les durées par étape ne sont pas significatives à plus d'un fil) Job de CI de détection de courses élargi aux tests neufs (préfixes `TestFils`, `TestVoie`, `ExecuterEnParallele`) ; `bench/README.md` : les durées par étape ne sont pas significatives à plus d'un fil. Haiku, effort bas.

Chantier 2a, lectures qui ne dépendent que du film :
- [x] K.J1 (2026-11-23 : intégré dans `feat/v1.6.0` ; `LireLesFaitsDuFilmSeul` prend le film et une fonction de roster, le pont verse les replis au même endroit ; test d'égalité avec le pont seul) Extraire du pont la lecture « film seul » (temps forts, médailles, index des joueurs, origine d'horloge), sorties identiques.
- [x] K.J2 (2026-11-23 : intégré ; la voie lit les faits du film seul, les scripts et la table des joueurs aux listages ; `rosterOf` vérifiée pure ; une panique est relayée une fois ; seule instruction `go` de `replay`) Tâche de fond de `replay` pendant la marche (après K.F3 et K.J1).

Chantier 2b, voies sur contexte dérivé (J3 à J10 de la conception) :
- [x] K.J4 (2026-11-23 : intégré ; toutes les suppositions de la conception tiennent ; seule l'étape des poses écrit des mémos froids) Gardes de caractérisation à un fil (tests sur le corpus) : à faire tôt, elles transforment les « déduit » de la conception en « vérifié ».
- [x] K.2b (2026-11-23 : FAIT sur décision du user, « on continue les chantiers sur la 1.6.0 » ; J3 contexte de voie, J6a à J6c lire et publier séparés, J7 à J9 orchestrateur et voies du monde et des sièges intégrés, tête `4ee6876` ; J5 sans objet : chaque voie copie la table des datums ; preuve : tests et détection de courses code 0, empreintes des 13 films identiques à la v1.5.0 à 1, 2, 4 et 8 fils, ordre des étapes identique. Mesure machine au calme, 5 passes, relevés `.ai/annexes_decodeur_v160/releve_voies_fils{1,2,4,8}.tsv` : total des 13 films 33,5 s à un fil, 25,8 s à deux, 24,2 s à quatre, 23,5 s à huit, contre 34,8 / 29,0 / 27,0 / 26,2 avant les voies ; film récent Slayer 798 / 534 / 479 / 457 ms contre 1 123 ms en v1.5.0. Limites : le détecteur de courses ne couvre que les mini-bobines ; à deux fils demandés, quatre fils sont actifs ; une voie perd les crochets d'observation du parent.) J3, J5, J6a à J6c, J7 à J10 : à décider après K.I5 ; items écrits ici s'ils sont retenus. Avis du pilote après la mesure (2026-11-23) : ne pas les faire dans la v1.6.0. Sur un film récent à quatre fils il reste environ 570 ms, dont 90 de publication ; les voies viseraient au mieux 10 % de plus, pour huit items dont un lourd et le risque le plus élevé de la piste. Décision demandée au user.

- [ ] K.3 Relevé final machine seule sur les 13 films, quatre niveaux de fils ; V0.7 (niveau de fils dans les outils de LevelUp) suit.

Gate de la piste K : à tout niveau de fils, empreintes du document et des faits identiques à un fil sur les 13 films et 20 identiques contre les références ; à un fil, aucun fil lancé ; tests neufs sous `-race -count=3` ; G-rev ; G-compat (ajouts seulement) ; aucun film au-dessus de 3 Gio à aucun niveau.

À savoir avant de choisir le niveau par défaut de l'hôte : le nombre de cœurs du serveur de production (question posée au user le 2026-11-23).

## Intégration « vitesse » (pilote)

- [~] I.1 (2026-11-23 : V4 abandonnée par le user, V3 couverte par les pistes G et K (V3.4) ; les pistes G et K se sont intégrées une à une dans `feat/v1.6.0`, empreintes des 13 films identiques à chaque intégration) V4 puis V3 fusionnées dans `feat/v1.6.0`, une à une (V1 abandonnée le 2026-11-23 ; la piste G passe en premier) ; G-equiv-ref après chacune : aucun écart,
      références non refigées ; empreintes de révision refigées par leur porte, valeurs inchangées (R-13) ;
      G-façade.
- [ ] I.2 Preuve d'identité complète, points 1 à 7 (gate du corpus au niveau `N`, pseudo-version désignée
      par `chore/830-decodeur-v160`, `GOWORK=off`).
- [ ] I.3 Relevés intermédiaires A (`bench/releves/v1.6.0-vitesse.tsv`, quatre niveaux) et B ; tableau avant
      et après présenté au user, avec le pic par niveau.
- **Gate I** : G-clone sur `./...` vert ; G-rev vert ; 20 identiques à chaque niveau et en double passe ;
  gate du corpus sans gain, perte ni décision.

## Piste V2 — lecture de plusieurs chunks à la fois (Opus ; s'intègre APRÈS les pistes A à F)

ABANDONNÉE le 2026-11-23 sur analyse : le monde est unique pour tout le film, l'image-clé d'un morceau ne rétablit qu'une partie de l'état, et une liaison survivante sort dans les records publiés ; gain plafonné à 10-15 %. Détail : `.ai/annexes_decodeur_v160/profil2.md`, section 7.

- [!] V2.1 (voir en tête de piste) Instruction sans code, dans `.ai/annexes_decodeur_v160/v2_conception.md` (vague 2) : ce qu'un
      chunk hérite du précédent dans notre marche (table d'entités du monde, liaisons des images-clés, table
      anticipée, neufs suspendus, profil de balayage, compteurs de replis, crochets des canaux, marche
      commune de la source de dégât) ; à quelle condition un chunk peut partir d'une copie de l'état ;
      comment on RECONNAÎT qu'il dépendait du précédent ; méthode du parseur voisin relue
      (`parser/v41/chunks.rs`) ; part de la marche des trames dans le profil V0.5. Si cette part est sous
      20 % du total, ou si la dépendance ne se reconnaît pas par un fait écrit, la piste s'arrête : `[!]`
      avec la mesure (abandon écrit et motivé, R-3).
- [!] V2.2 (voir en tête de piste) Conception relue par le pilote puis écrite ICI, datée, avant le code : forme de la copie d'état,
      règle de relecture en séquence, ordre de remise aux canaux (les crochets publient pendant la marche :
      leur ordre d'appel reste celui d'un fil), borne de mémoire (copies d'état en vol au plus égales au
      nombre de fils).
- [!] V2.3 (voir en tête de piste) Code derrière les options de V0.6 (zéro ou un fil = la marche actuelle, chemin inchangé).
- [!] V2.4 (voir en tête de piste) Tests neufs (noms contenant `Parallele`) : empreinte de la structure de lecture égale à un fil et
      à N fils sur les deux mini-bobines à trames (`marche_determinisme_test.go` étendu) ; chunk qui dépend
      du précédent forcé par construction ; ordre de fin perturbé par un délai aléatoire ; `-race -count=3`.
- [!] V2.5 (voir en tête de piste) Mesure par les relevés A et B aux quatre niveaux : décodage, total, pic, relectures en séquence
      par film. Aucun film du jeu fixe ne dépasse 2 Gio à aucun niveau ; un niveau qui dépasse est retiré
      des niveaux permis.
- **Gate V2** : preuve d'identité dans sa forme V2 (points 1 à 7) ; empreinte des FAITS égale à celle du
  niveau 1 sur les 20 films ; aucun compteur de repli du document ne bouge ; G-clone sur
  `./internal/grammar/ -run 'Parallele|Determinis' -race -count=3` (au moins 4 tests `PASS`) ; G-rev ;
  G-compat ; relecture du diff par le pilote avant intégration (état partagé, ordre des crochets).

## Piste A — score et tableau (Sonnet ; Opus pour A.3)

- [x] A.1 (2026-11-23 : trois causes prouvées, rapport `.ai/annexes_decodeur_v160/a1_verdicts_du_score.md`) Mesure sans code, sur les films du parc par le harnais : les cinq Grandes batailles restées
      « incomplètes » sans rang non confirmé ni manque ; la Grande bataille « complète » à tort (points
      publiés au-dessus du tableau) ; le film dont le tableau cesse de monter dans les dernières secondes.
      Cause par film, ou `[!]`.
- [x] A.2 (2026-11-23 : A.2a, A.2b et A.2c intégrés) Selon A.1 : corrections, écrites ici avant leur code (2026-11-23). Le verdict lui-même est juste : ce sont trois lectures qui se corrigent.
  - [x] A.2a (2026-11-23 : codé et prouvé sur films, `piste/a2a`, `d96be0c` ; genres 13, 27 et 117 lus, genre 93 lu dans son seul cas inhibé ; 143 trames arrêtées sur 39 films ramenées à 0, les 1 258 632 trames se ferment ; les cinq films deviennent « complets » sans changer un point, 28 films sur 39 au total ; sur 461 joueurs aucun ne se dégrade ; quelques tirs, touches et dégâts en plus dans les listes ; intégré le 2026-11-23, fusion `e3d422b`, révision de la grammaire montée à `grammar-2026-11-23` ; l'attribution des écarts d'équivalence est portée par L.1) Lecteurs de charge des genres 13, 27, 93 et 117 de la vue A, qui arrêtent aujourd'hui la lecture d'une trame (cinq Grandes batailles « incomplètes » alors que leurs points égalent l'API pour 128 joueurs ; environ 180 rejeux du parc, estimation). Largeurs à établir : témoin, puis preuve par le film (la trame se poursuit jusqu'à sa fin sur toutes les occurrences). Un genre non prouvé reste arrêtant. Sortie changée sur les films concernés : écart à attribuer à l'intégration.
  - [x] A.2b (2026-11-23 : codé et prouvé sur films, `piste/a2b`, `1d5f91a` ; sur le film, les éliminations du joueur concerné passent de 21 à 20 et sa somme de 2 325 à 2 225, aucun autre total ne bouge, 35 messages comptés comme copies ; trois films du corpus : aucune trame écartée ; intégré le 2026-11-23, fusion `303b7d7`, même montée de révision ; l'attribution des écarts d'équivalence est portée par L.1) Trame renvoyée en entier par le jeu (un film : une élimination comptée deux fois, +100 points) : écarter une trame dont TOUS les messages, au moins deux, répètent à clé identique (rang, charge, destinataires, numéro de séquence) ceux d'une trame de moins de 3 s ; la fenêtre d'une seconde reste pour un message isolé. Fichier : `internal/grammar/evenements_de_jeu.go`. Preuve : test avec les deux trames du film ; aucun changement sur les autres films relus.
  - [x] A.2c (2026-11-23 : codé et prouvé sur films, `piste/a2c`, `a2cfcdd` ; critère factuel ajouté, seuil intact ; le film passe de 17 680 à 18 805 points, égal à l'API pour ses 8 joueurs ; aucune autre manche changée sur 27 films ; les 4 films de la découverte restent inchangés à juste titre : leur manche 1 n'existe pas dans le film ; intégré le 2026-11-23, fusion `0242b1f`, révision des objectifs montée à `objectives-2026-11-23` ; l'attribution des écarts d'équivalence est portée par L.1) Dernière manche courte refusée (un film : 1 125 points absents du tableau) : la manche est écrite dans le film avec la remise à zéro simultanée de tous les joueurs présents. Décision du pilote : admettre une manche par ce FAIT (remise à zéro de tous les joueurs présents dans la même seconde) plutôt qu'abaisser le seuil de part d'enregistrements de 10 à 7 % ; le seuil n'est abaissé que si le fait ne se lit pas. Fichiers : `internal/facts/objectives/statborg.go`, `rounds_decision.go`. Preuve : le film réglé, aucun autre changement sur l'échantillon de 220 films, les 4 films à manche manquante de la section Découvertes examinés.
- [!] A.3 (2026-11-23 : verdict « non prouvable », `a_score_verdicts_a3_a4_a5_a6.md` ; sur 9 Grandes batailles de `HI_1_10_0`, 2 990 trames restent arrêtées sur une soixantaine de genres, la somme des actions n'égale l'API que pour 33 joueurs sur 232 sans le décalage et 38 avec : l'égalité n'est pas testable ; 14 lectures directes du tableau concordent avec la table décalée, 0 contraire ; pour les 34 Grandes batailles à drapeau, aucun groupe ni décalage ne se prouve (meilleur accord 13,5 %). Manque pour reprendre : la table des genres de la vue A de `HI_1_10_0`, et la source du groupe de scripts du drapeau dans les fichiers du jeu) Recherche (une session) : arbre des actions décalé sur le build qui précède `HI_1_11_0` (31
      lectures concordantes, égalité non testable sur des films à trames arrêtées) et les 34 Grandes
      batailles à drapeau sans groupe de scripts. Sortie : décalage PROUVÉ par le tableau du film, ou `[!]`.
- [!] A.4 (2026-11-23 : reporté au backlog de LevelUp, entrée « Points de bombe en Assaut » de `.ai/BACKLOG.md` ; l'explosion (+200) se reconnaît 22 fois sur 22 par le compteur, aucune source ne nomme l'action ; rien publié) Assaut : points de bombe hors de l'arbre des scripts (4 films). Recherche bornée ; publication si
      le film les nomme.
- [x] A.5 (2026-11-23 : fait, `436c09c` ; cause prouvée sur 1 595 films : 1 282 pénalités dans 635 films, la baisse de exactement 100 porte avec elle la montée du compteur de suicides ou de trahisons ; sur 67 films dont un total change, exacts 864 sur 1 030 avant, 928 sur 1 030 après, aucun exact devenu inexact ; sur les 20 films du corpus d'équivalence aucun total ne change) Courbe de score qui recule : la pénalité de suicide ou de trahison (−100) publiée comme telle, au
      lieu de la plus longue suite non décroissante. Mesure attendue : accord au score personnel de l'API en
      hausse, aucun joueur exact avant ne devient inexact.
- [x] A.6 (2026-11-23 : fait, `436c09c` ; la table des réponses de jeu est associée à la révision de la grammaire (`Couche.Associes`), une ligne de plus change l'empreinte ; 4 tests neufs) Révision de couche pour la table des réponses de jeu : une ligne ajoutée à la table rend un rejeu
      « à redécoder ».
- [ ] A.7 (2026-11-23 : codé sur `piste/a7`, `7701999`, comparateur instant de début puis slot, test neuf ; intégré le 2026-11-23, fusion `03f5144` ; non coché : la preuve d'équivalence sur les 20 films attend L.1, les références n'étant pas refigées) (ajouté le 2026-11-23, issu de 0.4) `joinSegments` (`replay/score_timeline_players.go`) : le comparateur
      devient total, instant de début PUIS slot ; test unitaire à deux segments de même début, ordre d'insertion
      inversé ; effet attendu sur les 20 films : aucun ; change le cas d'égalité, aujourd'hui indéterminé.
      Sonnet, effort bas, classe B.
- **Gate A** : équivalence attribuée (G-equiv-ref, chaque écart rattaché à un item) ; accord au score
  personnel de l'API (export 0.8) sans baisse pour aucun build ; G-clone sur `./reponsesdejeu/ ./replay/
  ./internal/facts/objectives/` ; G-rev ; G-compat.

## Piste B — drapeau et zones (Sonnet ; Opus pour B.1)

- [x] B.1 (2026-11-23 : causes prouvées, `b1_causes_des_portages_faux.md` ; famille 1 : lecture limitée aux 8 premiers sièges ; famille 2 : variante à un seul drapeau lue avec deux ; famille 3 : suite d'un siège qui change d'occupant, et position d'un corps immobile) Recherche (une session) sur les 16 portages à l'état « au socle » faux : porteur précédent hors
      des 8 premières places ou sur une place sans humain nommé (7) ; variante à un seul drapeau publiée
      avec deux drapeaux d'équipe (4) ; résidu (5). Une cause prouvée par famille, ou `[!]`.
- [x] B.2 (2026-11-23 : intégré ; B.2a tous les sièges du film lus pour le drapeau (`3f6eda6`), B.2b une base neutre lue par le film fait un seul drapeau (`32eb357`), B.2d la suite d'un siège se coupe au changement d'occupant (`2734cea`), B.2c un porteur immobile garde sa position à travers un trou de lecture (`f49affe`) ; sur 218 joueurs de 13 films, justes à 2 s de l'API : 161 avant, 197 avec B.2a et B.2b, 203 avec B.2d, aucun juste devenu faux) Corrections prescrites par B.1, écrites ici avant leur code.
- [!] B.3 (2026-11-23 : verdict rendu sans code, `b_zones_verdicts_b3_b4_b6_b7.md` ; l'équipe est lue pour 25 joueurs sur 25, ce que la règle ne sait pas nommer est le DRAPEAU (3 par camp) ; conséquence mesurable nulle : temps de porteur à 2 s de l'API pour 24 joueurs sur 25 ; le compteur `carrierTeamUnknown` mesure l'abstention de la règle, nom trompeur, entrée du backlog) Équipe du porteur sur une carte à 6 socles (25 portages sur 25 d'un film) : lecture de l'équipe
      par le film, ou `[!]`.
- [!] B.4 (2026-11-23 : verdict rendu sans code ; le composant 21 côté A n'est pas un compteur sur ces builds (valeurs congrues à 8 modulo 32), la définition des emplacements du tableau n'existe dans le film que pour `HI_1_12_0` et `HI_1_13_0` ; sans effet aujourd'hui (`objectives[]` vide) ; manque pour reprendre : un lecteur Go de ces définitions) Compteurs de zones sur d'anciens builds (octet signé qui avance de 32) : borne par build, lue dans
      les définitions du film.
- [~] B.5 (2026-11-23 : couvert par B.2a, qui lit tous les sièges ; justes à 2 s de l'API 161 -> 203 sur 218, aucun juste devenu faux ; le rapport exact au-delà de 8 joueurs contre en deçà n'a pas été calculé séparément) Ouverture de la bande au-delà de 8 joueurs, SEULEMENT si, après B.2 et B.4, le temps de porteur
      est à 2 s de l'API pour au moins autant de joueurs au-delà de 8 qu'en deçà (91,7 % en v1.5.0) ; sinon
      `[!]` avec la mesure.
- [!] B.6 (2026-11-23 : calcul fait et mesuré, 231 lettres justes sur 231 (77 films de Bases, `Unmatched` 0), mais NON intégré : il reste sur la branche `piste/bl-zones`, sans appelant ; manquent la source (balayage qui publie état et équipe des marqueurs) et la lecture du composant qui arrête les images-clés) Lettre des zones comme contrôle : deux compteurs de couverture (lettres lues, lettres en accord
      avec le rang publié) ; lecture du composant du marqueur qui arrête la lecture aux images-clés.
- [!] B.7 (2026-11-23 : non reproduit ; sur le film et sur un second film de Bases, les trois zones sont nommées et aucun repli ne joue ; la cause était la sonde de la recherche, pas le rejeu) Zone au propriétaire de nom inconnu (un film) : cause, puis lecture ou `[!]`.
- **Gate B** : équivalence attribuée ; temps de porteur contre l'API sur les 186 films à drapeau : aucun
  porteur juste avant ne devient faux ; compteurs de zones contre l'API sans baisse ; G-clone sur
  `./replay/ ./internal/grammar/` ; G-rev ; G-compat.

## Piste C — gestes (Sonnet)

- [~] C.1 (2026-11-23 : repris par le portage des gestes du contributeur : accroupi publié comme commande tenue, `de99cbc`, seulement sur les films à pleine cadence, donc pour la version actuelle du jeu (décision du user) ; la glissade n'est pas publiée : 74 % seulement des séries rapides atteignent le pic, aucune vérité indépendante, `c123_gestes_mesures.md`) Accroupissement et glissade sur les versions récentes : le bit de valeur 1 de l'entrée de contrôle,
      recoupé avec la vitesse et la hauteur sur 20 films. Vérité : les 7 rejeux qui les publient déjà par
      l'autre voie. Publication si l'accord atteint 90 %.
- [~] C.2 (2026-11-23 : repris par le portage des gestes : saut publié comme commande tenue sur les films à pleine cadence seulement, `de99cbc` ; les builds 1.9 à 1.11 sont à 30 Hz et ne publient pas le saut) Bouton de saut sur les builds 1.9 à 1.11 (accord de 50 à 61 %) : disposition de la vue de contrôle
      de ces builds ; publication du bouton seulement à 90 % d'accord sur tous les builds, sinon `[!]`.
- [~] C.3 (2026-11-23 : repris par le portage des gestes : les rechargements gagnent l'instant et le compte des copies (`276b1e1`) ; aucun des quatre booléens n'est nommé, laissés tels quels) Sens des quatre booléens d'un rechargement : mesure contre les munitions lues ; nommés s'ils sont
      prouvés, sinon laissés tels quels.
- [x] C.4 (2026-11-23 : table rendue, `.ai/annexes_decodeur_v160/c4_e5_recoupements.md` ; le contributeur apporte coup de mêlée, lancer, rechargement et déplacement brut ; désaccord sur le bit du saut à trancher par un film récent, avec C.2) Recoupement avec les canaux du contributeur (manche, postures de contrôle, gestes), sans code :
      table « son champ, notre champ, lequel reste », versée à l'inventaire P.1 et au dossier P.2.
- **Gate C** : équivalence attribuée ; poids ajouté au document sous 1 % sur 20 films ; G-clone sur
  `./replay/ ./internal/grammar/` ; G-rev ; G-compat.

## Piste D — répliques des Spartans (Sonnet ; dans la v1.6.0, DD-4)

- [ ] D.1 (2026-11-23 : intégré dans `feat/v1.6.0`, tête `cd8f560`, empreintes des 13 films identiques ; genre 75 = réplique d'un Spartan, genre 76 = annonce ; lus seulement quand un canal le demande, rien de publié ; essai sur un film : 365 répliques, 951 annonces, aucune trame arrêtée ; non coché : contrôle des comptes contre le témoin sur 3 films, à la file des mesures) Lecture par la grammaire des genres 75 et 76 de la vue A (locuteur, identifiant du son, priorité,
      indice de réplique ; annonceur : identifiant du son, destinataires), aujourd'hui traversés sans valeur.
      Contrôle : mêmes comptes que le témoin sur 3 films.
- [ ] D.2 (2026-11-23 : EN COURS, agent lancé le 2026-11-23, branche `piste/d2`) Nom court de la réplique, DANS le décodeur (cadrage du user) : recherche bornée de la
      correspondance entre l'indice du film et l'entrée des tables de vocalisation (5 noms résolus à ce
      jour). Un nom est publié quand il est prouvé ; une réplique dont le nom n'est pas prouvé est publiée
      sans nom, et comptée. La recherche suit R-3. Aucun nom inventé.
- [x] D.3 (2026-11-23 : intégré, `b4958dc` puis fusion `cc0de33` ; calques `chatter[]` et `announcements[]` publiés depuis un canal de la marche, couverture, schéma 127) Publication : calque `chatter[]` (image, joueur, son, genre, nom court s'il existe) et sa
      couverture (lus, publiés, sans joueur, hors du match) ; les annonces de l'annonceur dans un calque à
      part si D.1 les lit. Le film ne dit pas quelle variante audio a joué : le document ne le dit pas non
      plus.
- [ ] D.4 (2026-11-23 : `FORMAT.md`, `GLOSSARY.md` et `NOTICE.md` posés pour les répliques, les annonces et les gestes, `3faf1b4` ; reste le nom court de D.2) Fragments de documentation (R-15) : `FORMAT.md`, `GLOSSARY.md` ; crédit à Den (méthode et table de
      textes, libfilmshell, MIT) pour `NOTICE.md`.
- [ ] D.5 (décision du user du 2026-11-23 : textes ET voix, en français et en anglais, versés au dépôt comme les sons d'armes ; estimation donnée au user : textes environ 40 Ko, voix 12 à 30 Mo pour les deux langues, à MESURER avant de verser ; deux inconnues à lever : nombre de voix de Spartan enregistrées par réplique, et la voix anglaise n'est pas installée sur le poste) HORS DÉCODEUR (DD-4) : côté LevelUp, à la montée, commande qui bâtit
      la table des répliques depuis le jeu installé (identifiant du son vers textes français et anglais et
      fichiers de voix), faite comme celle des sons d'armes (`cmd/weapon-sounds`), derrière le tag
      `gamefiles` ; aucun fichier du jeu au dépôt.
- [ ] D.6 HORS DÉCODEUR (DD-4) : UNE maquette interactive des trois idées du user (icône de son et
      texte sur le joueur ; icône et son joué ; une variante au hasard dans le fil avec un filtre
      « Répliques »), sur un rejeu réel, inscrite à l'artefact des vérifications. Les zones de lieu de
      LevelUp servent telles quelles. L'affichage réel se fait dans un plan à part, après le choix du user
      sur la maquette : aucun code d'affichage ici.
- **Gate D** : équivalence attribuée, plus G-equiv-2p (canal de faits neuf) ; poids ajouté sous 1 % ;
  budget des fixtures de contrat relevé et daté ; G-clone sur `./replay/ ./internal/grammar/` ; G-rev ;
  G-compat.

## Piste E — canaux neufs du backlog (Sonnet ; Haiku pour E.5)

- [x] E.1 (2026-11-23 : intégré, `b3aca22` ; éliminations et actions de score d'un bot publiées par place, sans identifiant ; mesure `e1_eliminations_de_bots.md` : 0 élimination de bot dans `eliminations[]` avant, sur 5 films à bots, écart de 1 à 12,6 % contre le tableau du film) Éliminations dont un bot est l'auteur : mesure d'abord (combien `eliminations[]` en porte sur les
      films à bots, combien `scoreActions` en nomme) ; puis publication avec un auteur sans identifiant et
      sa place. Vérité : le tableau du film.
- [x] E.2 (2026-11-23 : intégré, `caaa8ca` ; courbes par joueur du temps en VIP, du plus long temps et de la meilleure série ; temps 24 sur 24 exacts, série 24 sur 24, plus long temps 22 sur 24 plus deux à 0,1 s ; constat vérifié le 2026-11-23 : sur un des trois films VIP, 23 courbes pour 8 joueurs au lieu de 24, la courbe absente est la meilleure série en VIP d'un joueur dont la valeur est 0 dans le film ET dans l'API ; une courbe absente se lit zéro, le chiffre est juste, laissé en l'état ; poids +3,1 à +4,1 % sur ces films, documents identiques à 1 et 4 fils) VIP : temps en VIP, plus long temps en VIP, meilleure série en VIP, publiés en courbes par joueur
      comme les compteurs du tableau. L'hôte dit que le match est un VIP par la constante déjà exportée
      (`decfilm.ObjectiveTypeVip`). Après la piste A (mêmes fichiers).
- [!] E.3 (2026-11-23 : source des noms d'options non établie, motif ci-dessous ; instruit sans code, `dec-v160-recup/pistes/e3/instruction.md` hors dépôt ; le film ne porte que le hachage, vérifié sur deux films à tous les décalages de bit ; les archives lisibles du jeu ne rendent aucun des 30 hachages testés ; les noms sont très probablement dans l'exécutable, non lu ; aucun code tant que la source n'est pas établie) Nom des options de variante : le document publie le nom à côté du hachage quand le FILM ou les
      fichiers du JEU le donnent (chaîne lue, contrôlée par son hachage). La table du parseur voisin n'est
      ni copiée ni consultée comme source. Une option sans nom lu reste un hachage, comptée.
- [ ] E.4 Apparence enregistrée des Spartans (armure, revêtement) : recherche bornée (une session), sur 3
      films ; publication si la lecture est prouvée contre la personnalisation publique de l'API ; sinon
      `[!]`. À remettre à la session du rejeu 3D dans les deux cas.
- [x] E.5 (2026-11-23 : zoom déjà publié chez nous, sorties de lunette sous-comptées ; coup de mêlée sans mort couvert par le contributeur seulement ; même annexe que C.4) Zoom et coup de mêlée sans mort : contrôle sur pièces de ce qui est déjà publié (visée) et de ce
      que les gestes du contributeur couvrent (coup) ; sort écrit, versé à l'inventaire P.1.
- [x] E.6 (2026-11-23 : intégré, `caaa8ca` ; champ `victimAmbiguous`, victime absente quand plusieurs candidates restent ; effet nul sur 27 films, 332 joueurs sur 332 égaux à l'API) Victime d'une élimination appariée au plus proche : victime absente et compteur d'ambiguïté quand
      une action a plusieurs candidats.
- **Gate E** : équivalence attribuée ; éliminations et statistiques VIP contre l'API sans baisse ; G-clone
  sur `./replay/ ./internal/grammar/ ./internal/facts/objectives/` ; G-rev ; G-compat.

## Piste F — contrôles et petites entrées (Haiku ; Sonnet)

- [x] F.1 (2026-11-23 : fait sur `chore/830-decodeur-v150`, `aab869c44e` ; témoin joué : `go test` d'un sous-paquet du clone passe avec l'espace de travail engendré) `scripts/briques-workspace.sh` (LevelUp) : sous Windows, l'entrée `use` du clone s'écrit avec des
      barres inverses ; témoin : `go test` d'un sous-paquet du clone passe avec le `GOWORK` engendré.
- [x] F.2 (2026-11-23 : fait, même commit ; sept blocs retirés, liste dans `dec-v160-recup/pistes/f2_retraits.md` ; restent 20 décisions du 2026-11-22 et `coverage.hits.noVictim` du 2026-11-16, hors liste ; test du manifeste vert) Les sept décisions devenues sans effet de `config/replay_corpus.toml` (2026-11-16) retirées
      (DD-8) ; leur liste est écrite au journal et montrée dans le rapport d'exécution. Précision du
      2026-11-23 : le retrait ne porte que sur les 7 décisions du 2026-11-16 : `coverage.hits.{noShooter,
      stoppedFrames, vitalsUnread}`, `coverage.flagCarries.{seatsBeyondBand, gaugePoints}`,
      `flagCarries.spans.returnProgress/{n, total}`. Les 21 décisions du 2026-11-22 restent nécessaires tant
      que la montée du décodeur n'est pas fusionnée dans la branche de version (la base du gate final sera
      alors le décodeur v1.4.1).
- [!] F.3 (2026-11-23 : repli GARDÉ, `e_f_verdicts_e2_e6_f3_f6.md` ; zéro sur 13 films à véhicules, mais le film de format 31, seul cas visé par le repli, n'a aucune pièce montée ; à reprendre sur le parc régénéré) Repli du voisinage des pièces montées : compté sur les films à véhicules par le harnais ; à zéro
      partout, l'entrée du registre est retirée et sa constante gardée, marquée dépréciée.
- [x] F.4 (2026-11-23 : verdict rendu, aucun code ; 409 temps forts « mode » sur 410 ont un objectif ET une action de score du même joueur à 2 s ; le reste vient de films d'anciens builds où rien ne se publie, refusé par le roster) Temps forts « mode » sans action de score (3 410 en Drapeau, 691 en Total Control, 303 en
      Extraction) : recroisés avec les huit noms d'action neufs de la v1.5.0. Verdict écrit ; aucun code si
      rien ne se prouve.
- [x] F.5 (2026-11-23 : cause trouvée sur 7 films, aucun code ; la durée de l'API compte l'avant coup d'envoi, 26 à 41 s, égal à l'instant de coup d'envoi du film à 0,4 s près, et s'arrête environ 6,5 s avant la fin du film) Décalage de 35 s entre le film et le temps de jeu de l'API : une session de mesure (ce que l'API
      compte avant le coup d'envoi et après la fin). Cause, ou `[!]`.
- [!] F.6 (2026-11-23 : aucun défaut ; dans les 4 cas la vie « repreneuse » est du MÊME joueur que la vie achevée, rien à créditer à un autre ; aucune tolérance posée) Lâchers 2 à 4 images après une mort (4 cas) : un fait du film qui nomme le propriétaire du lâcher,
      ou `[!]` (aucune tolérance de temps).
- [x] F.7 (2026-11-23 : verdict rendu, `f7_rejeux_sans_build.md` ; ces films n'ont pas de section d'identification (format 20), donc ni table de joueurs ni compteur nommé) Cinq rejeux sans build : pourquoi aucun compteur nommé n'y est lisible ; verdict écrit.
- [x] F.8 (fait le 2026-11-23, `41389c4` : l'aide de test échoue en nommant les candidates quand deux cartes partagent la signature) (ajouté le 2026-11-23, issu de la preuve d'A.2b) Harnais de test : `mapEntryFromCatalog` (`replay/ability_impulses_film_test.go`) choisit une carte par l'ordre de parcours d'une `map` quand deux cartes du catalogue portent la même signature d'axes ; le document produit par les tests et les sondes qui l'appellent varie alors d'un lancement à l'autre (mesuré : dix lancers, deux empreintes). Aucun appelant de production : l'hôte donne toujours la carte. Correction : échouer en nommant les candidates quand la signature est partagée, ou prendre la carte donnée par l'appelant ; test : même erreur sur cent appels. Haiku, effort bas, classe B.
- **Gate F** : G-clone sur `./internal/facts/fallback/ ./decfilm/` ; équivalence attribuée pour F.3 ;
  G-compat (la constante de F.3 est toujours là).

## Piste P — accueil du travail du contributeur (pilote ; Sonnet pour l'inventaire et les relectures)

Le contributeur écrit lui-même le report de ses ajouts sur la v1.5.0, dans les trois briques (DD-1, DD-6).
Aucun agent de ce plan n'écrit ce code. Notre part : préparer, remettre, synchroniser, vérifier, intégrer.

**Risques de collision, dits.** Son travail et nos pistes touchent les mêmes fichiers (liste de la section
1.3) : les options (`replay/options.go` : V0.6, V4), le balayage du film (`film_scan_*.go` : V4, V3), la vue
A (`vue_a_lecture.go` : D), les postures (`document_stances.go`, vue de contrôle : C), le chargement
(`filmcache/` : V1), la façade `decfilm`, et tout ce qui n'a qu'un écrivain : schéma, chronique, faits,
goldens, révisions, références. Trois recouvrements de FOND s'y ajoutent : ses gestes contre nos
`reloads[]`, son manche contre notre `moveCommand`, ses postures de contrôle contre la piste C. Un rebase
fait pendant que nos pistes bougent serait à refaire à chaque intégration, et deux écritures du schéma se
marcheraient dessus.

**Règle de gel.** Nos pistes s'intègrent d'abord (vagues 1 à 9). Le pilote pose alors un commit de GEL sur
`feat/v1.6.0`, annoncé au contributeur : c'est SA base. À partir du gel, plus aucune piste de ce plan
n'écrit sur la branche ; seules y entrent ses PR, une à une, puis les corrections de la relecture. Un
correctif de notre côté après le gel lui est annoncé avant d'être poussé. Avant le gel, il peut travailler
sur le tag `v1.5.0` ; il sait qu'un seul rebase l'attend, sur le commit de gel.

- [~] P.1 (2026-11-23 : remplacé par le portage par nous, voir « Portage par nous » ; `portage_complet_contributeur.md` porte l'inventaire par thème et la table par canal ; inventaire rendu dans `.ai/annexes_decodeur_v160/inventaire_contributeur.md` ; non coché : l'ordre des tags des trois briques et la table par canal, exigés ci-dessous, ne figurent pas encore dans ce rendu) Inventaire en lecture seule de ses copies, en deux parts confiées à deux agents (P.1a le
      décodeur, P.1b les deux autres briques), pour PRÉPARER l'accueil (aucun code, aucune PR), sur la
      tête de son dépôt au jour de l'item : pour le décodeur, diff de `third_party/film-decoder` contre
      v1.3.6 puis contre v1.5.0 ; pour les deux autres briques, diff à trois voies contre nos v1.4.0 et
      v1.2.0. Par canal : fichiers, schéma qu'il prend chez lui, recouvrement avec la v1.5.0 et avec nos
      pistes (tables de C.4 et E.5), règle du dépôt en jeu, ajout d'une autre brique dont il dépend, sort du
      codec `REPLAYINPUTS30`, dépendance neuve (`woozymasta/bcn`, licence à relire), fichiers du jeu omis.
      Rendu : `.ai/annexes_decodeur_v160/inventaire_contributeur.md`, avec l'ORDRE DES TAGS des trois
      briques.
- [~] P.2 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Dossier remis au contributeur, en anglais, `.ai/annexes_decodeur_v160/accueil_contributeur.md`,
      envoyé par le user. Contenu fermé :
      (a) l'état de départ : tag `v1.5.0` du décodeur (`dc967c2`, document au schéma 126, faits au schéma
      30), brique des cartes v1.4.0, lecteur de fichiers v1.2.0 ; ce que la v1.5.0 a ajouté et qui recouvre
      ses canaux (`reloads[]`, `moveCommand`, `tirsParPlace`, `damage[]`, `eliminations[]`, coup d'envoi par
      l'horloge), d'après P.1 ;
      (b) les règles du dépôt public : aucun identifiant de joueur réel, aucun fichier extrait du jeu, test
      du jeu installé nommé `*_gamefiles_test.go` et étiqueté `gamefiles`, 500 lignes par fichier, aucun
      repli sans nom ni compteur, la grammaire prime sur toute dérivation, passage par la façade `decfilm`,
      ajouts seulement (aucun symbole public retiré, aucune signature changée), langue et nommage de chaque
      dépôt gardés, pas de traduction des fichiers existants ;
      (c) les numéros de schéma : UN seul passage pour toute la version, document 127 et faits 31 ; sur sa
      branche il prend ces deux numéros pour l'ensemble de ses canaux, en une entrée de chronique ; le
      pilote fond son entrée avec la nôtre à l'intégration et refige lui-même goldens, révisions, références
      et fixtures : il n'a pas à les tenir verts contre notre branche ;
      (d) la marche de publication (`marche_de_publication.md`, gestes 1 à 12), les commandes de gate de ce
      plan qu'il peut jouer chez lui (G-clone, G-rev, G-compat), et le fait que les tags sont posés par le
      pilote ;
      (e) ce que nos pistes changent pendant qu'il travaille : la table des fichiers par piste et la liste
      des collisions ;
      (f) la synchronisation : base `v1.5.0` d'ici le gel ; un seul rebase, sur le commit de gel de
      `feat/v1.6.0`, que nous lui annonçons ; une PR par canal ou petit groupe, chacune disant ce qui change
      dans la sortie ; PR des deux autres briques d'abord quand un canal en dépend ;
      (g) ce que nous lui demandons de ne pas remonter (section 2), et la garde de la précision (R-14) :
      un canal qui touche tirs, touches ou dégâts ajoute des champs sans changer un compte existant ;
      (h) ce qui est vérifié à l'arrivée (P.5), pour qu'il le sache d'avance.
- [~] P.3 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Points de synchronisation, annoncés au contributeur par le user ou par une issue du dépôt : fin de
      la phase vitesse (les options et le balayage du film ont leur forme finale) ; première intégration au
      schéma 127 ; gel. À chacun : le commit, la liste des fichiers de la liste des collisions qui ont bougé.
- [~] P.4 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Gel : toutes nos pistes intégrées ou statuées, `feat/v1.6.0` verte (G-clone sur `./...`, G-rev,
      G-compat, G-equiv-ref à 20 identiques sur références refigées) ; commit de gel annoncé ; plus aucune
      écriture de notre côté hors R-17.
- [~] P.5 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Vérification à l'arrivée, PAR PR du décodeur, dans cet ordre, sur une branche de contrôle (jamais
      sur `feat/v1.6.0`) : contrôle des identifiants et des fichiers du jeu (R-5) ; relecture par un agent
      Sonnet qui n'a pas écrit le code, axes : heuristique ou dérivation à la place d'une lecture, repli
      sans nom, sortie changée non déclarée, recouvrement avec un champ existant ; G-clone sur `./...` ;
      G-rev ; G-compat ; équivalence ATTRIBUÉE (chaque écart rattaché à ce que la PR déclare) ; G-equiv-2p ;
      garde de la précision R-14 pour tout canal qui touche tirs, touches ou dégâts ; poids ajouté au
      document mesuré ; relevé A à un fil (total dans les 10 % du relevé d'avant la PR, sinon retour au
      contributeur avec la mesure) ; si V2 est intégrée, empreintes du niveau `N` égales à celles du
      niveau 1. Un contrôle rouge : la PR lui revient avec la pièce ; nous ne corrigeons pas son code à sa
      place sans son accord.
- [~] P.6 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Réception des deux autres briques, dans l'ordre de P.1 : `go test -count=1 -v ./...` de la brique,
      zéro `SKIP` inattendu ; G-compat contre son dernier tag ; aucun fichier extrait du jeu ; licence de
      toute dépendance neuve relue ; `make go-api-test-gamefiles` une fois, par le pilote, seul sur la
      machine. Tags mineurs posés par le pilote. Ordre : lecteur de fichiers, puis brique des cartes (elle
      en dépend), puis montée des `require` du décodeur si un canal l'exige (suivie de G-clone sur `./...`
      et de G-equiv-ref sans écart), puis seulement le tag du décodeur. La montée de ces deux `require` dans
      LevelUp se fait sur la branche du plan 3D, pas sur celle de ce plan.
- [~] P.7 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Intégration de ses PR du décodeur dans `feat/v1.6.0`, une à une, sous le SEUL schéma 127 : le
      pilote fond les entrées de chronique, refige goldens, révisions (rang neuf, R-13), fixtures de contrat
      et références, tient le registre des changements de sortie (une ligne par canal, avec son crédit), et
      assemble ses fragments de documentation (R-15). G-façade après chaque intégration.
- [~] P.8 (2026-11-23 : remplacé par le portage par nous, décision du user du 2026-11-23, voir « Portage par nous ») Après la dernière intégration : G-corpus au niveau `N` ; son application compile et joue un match
      contre `feat/v1.6.0` (contrôle fait par lui) ; la session du rejeu 3D est prévenue des champs ajoutés
      (R-10) ; crédit du contributeur dans `NOTICE.md` et les notes de version.
- **Gate P** : chaque PR a passé P.5 ; les trois briques sont taguées dans l'ordre de P.1, sauf le décodeur
  (tag en L.3) ; aucun compte de tirs, de touches ni de dégâts n'a bougé ; le document n'a qu'UNE entrée de
  chronique 127 ; G-corpus sans perte non décidée par le user.

### Portage par nous (2026-11-23)

Décision du user du 2026-11-23 : NOUS reprenons les ajouts du contributeur dans notre dépôt, un thème à la fois, sans attendre
ses demandes de fusion ; une note de migration lui est remise à la fin (P.9). Pièces : `portage_complet_contributeur.md`,
`portage_des_gestes_contributeur.md`, `comparaison_tirs_contributeur.md`, `vitalites_sante_bouclier.md`,
`journal_des_integrations.md`. Les lectures sont reprises sous nos règles (la grammaire prime, aucun seuil sans raison,
replis nommés et comptés), pas copiées.

Porté et intégré dans `feat/v1.6.0` (six intégrations) :
- gestes : coup de mêlée et lancer amorcé (`276b1e1`) ; rechargements avec leur instant et le compte des copies ; accroupi et saut comme commandes tenues, publiés seulement quand le pas des commandes dit 60 Hz, donc pour la version actuelle du jeu (`de99cbc`) ;
- tireur d'un tir par la place de contrôle lue dans le film (`f6e3441`, correctif `1c62962`) ;
- décalage du tir dans son image, dégâts désignés par piste, dégâts à zéro point publiés (`be66d06`) ;
- visées sans position lues sur tout film, versées dans la pleine cadence (`6995a51`) ;
- arme tirée en main et munitions portées, munitions gardées dans les faits (`a5ebfcc`, `9277e4b`) ;
- positions, visée et projectiles à pleine cadence (`3bcd38e`), puis pleine cadence TOUJOURS publiée, l'option est retirée (`56fa59c`) ;
- série de santé et de bouclier par vie, à chaque changement (`c7aef20`) ;
- projectiles collés, avec un calque pour ceux qu'aucun vol ne porte (`b927896`, `2af640a`) ;
- détonation d'un lancer sans vol, candidats de socle, socles d'équipement (`2a563d3`) ; source de film par chemin, catalogue de bornes en octets, axes du socle d'arme (`e97771d`).

Non porté, et pourquoi :
- causes des morts (`killCauses`) : LevelUp a ses éliminations et leur arme, et `eliminations[].headshot` existe ;
- table des scores cibles par variante : reste chez l'hôte, qui passe déjà `TargetScore` ;
- traductions des journaux et noms anglais des replis, instruments et sondes de recherche : hors règles du dépôt ;
- tri des tirs par décalage : nous gardons l'ordre du film ;
- glissade : aucune vérité indépendante (74 % des séries rapides au pic, sous les 90 % exigés) ;
- bouton de saut des builds à 30 Hz et saut dérivé de la vitesse verticale : non publiés ;
- booléens d'un rechargement : aucun nom prouvé ;
- seconde famille du Stalker Rifle : absente du catalogue de la tête, non statuée par les pièces d'intégration ;
- plage de la pose d'un projectile collé : les trois valeurs brutes sont publiées, aucune plage figée (mesure sur 4 944 grenades).

Décisions du user du 2026-11-23 qui s'y rattachent : aucune option dans le décodeur (tout est publié, pleine cadence comprise) ; dégâts à zéro point publiés ; série de santé et de bouclier par vie publiée ; purge de l'historique git abandonnée.

- [ ] P.9 Note de migration remise au contributeur (en anglais), à la clôture.
- [ ] P.10 Crédit au contributeur sous son pseudo GitHub llampwall : remerciements du décodeur à la clôture, crédits de LevelUp à la montée (page des crédits, fichier de crédits), pour le décodeur et surtout le rejeu 3D.

## Étape R — relecture, AVANT les gates finals

- [ ] R.1 Relecture adversariale (skill `adversarial-review`, Fable) du diff du décodeur
      `v1.5.0..feat/v1.6.0`, toutes pistes intégrées. Axes imposés : non-déterminisme (ordre des fils,
      parcours de `map`), état partagé entre fils, heuristique déguisée, repli neuf, identifiant réel,
      symbole public retiré ou signature changée, sortie changée sans ligne au registre, champ ajouté par
      une piste V, révision de couche non montée pour un changement de lecture.
- [ ] R.2 P0 et P1 corrigés en une ronde (Sonnet) ; un constat qui touche un compteur publié n'est jamais
      classé mineur (leçon du gate de la v1.5.0) ; P2 au backlog.

## Étape L — gates finals, publication, montée, régénération (pilote)

Ordre imposé : R.1, R.2, puis L.1, L.2, L.3. Toute correction après L.1 fait repasser L.1 (R-17).

- [ ] L.1 Sur le commit corrigé : références d'équivalence refigées avec chaque écart attribué, puis
      G-equiv-ref à 20 identiques, G-equiv-2p, G-clone sur `./...`, G-rev, G-compat, G-façade ; G-corpus au
      niveau `N` ; tout écart à inscrire au manifeste est présenté au user, qui décide ; registre des replis
      égal à `FORMAT.md`.
- [ ] L.2 Relevés finals A (`bench/releves/v1.6.0.tsv`) et B, en fenêtre de mesure ; fichiers communs
      assemblés des fragments (R-15).
- [ ] L.3 PR du décodeur, CI verte ; tag v1.6.0 sur le commit qui a passé L.1 ; release GitHub : registre
      des changements de sortie, tableau de vitesse, crédits (nuzzles, Den, le contributeur).
- [ ] L.4 Montée dans LevelUp sur `chore/830-decodeur-v160` : `require` sur le tag ; miroirs Go et web
      SEULEMENT pour ce que LevelUp lit ; contrat de l'API, fixtures ; plafonds de la façade ; niveau de
      fils réglé par installation dans `internal/config` (DD-3) : 1 sur le serveur, et sur le bureau et en
      développement la valeur que le relevé B et le pic mesuré permettent sous le plafond inchangé, 1 faute
      de mesure ; appelants de 0.7 branchés sur le choix des canaux, CHACUN avec ses
      témoins avant et après (sortie de la source de dégât, coup d'envoi du film, rasters tactiques :
      capturée sur 5 films avant le branchement, identique après) ; liste des champs « servis seuls » vidée
      si elle ne l'est pas ; relecture adversariale du diff de la montée (Fable), P0 et P1 corrigés.
- [ ] L.5 Gates de LevelUp avec `GOWORK=off` : `go build`, `go vet`, `go test ./...`,
      `go test -tags=integration -p 1` sur `replaybuild`, `sync/filmfetch`, `sync/killcollector`,
      `sync/replayartifacts`, `internal/api/wire` ; `make replay-equiv` ; suite complète du web une fois ;
      `make gate-push` ; CI lue après chaque poussée. Notes de version dans les deux langues, crédits,
      journal, backlog, registre des reports. Fusion dans `release/v8.3.0` (DD-5).
- [ ] L.6 Régénération UNIQUE du parc (go d'avance du 2026-11-18, rappelé le 2026-11-22), le user PRÉVENU AU
      LANCEMENT : serveur arrêté, environ 2 h 30, un film à la fois, pas pendant qu'il joue ; AVANT le
      premier démarrage de LevelUp 8.3.0. Puis les dérivés (`backfill-usage-summary`,
      `backfill-pad-tiers --force`, `backfill-killsource --workers 1`, `backfill-vehicle-takes --force`,
      `backfill-flag-grabs-net --force`, `tactical-rasters --backfill`) et le rattrapage du coup d'envoi
      (`levelup backfill-kickoff`). La v1.6.0 porte les canaux du contributeur (DD-1) : il n'y a qu'une
      régénération.
- [ ] L.7 Mesures APRÈS sur le parc régénéré, comparées à `avant.md` de ce plan ET aux mesures reportées du
      plan v1.5.0 (A.3, C.4, E.4, L.7) ; durée de la régénération comparée au précédent (1 455 films en
      2 h 26) ; vérifications à l'écran inscrites à l'artefact des vérifications.
- [ ] L.8 Gestes de production inscrits à la liste de la release : dossier d'attente, bascule ; niveau de
      fils du serveur à 1.
- [ ] L.9 Clôture : tous les items statués ; recherches closes ou `[!]` avec leurs pièces et leur motif ;
      backlog mis à jour entrée par entrée (section 1.4) ; registre des reports (dont le report de
      la marche commune, déjà fait) ; items L.6, L.7 et R.3 du plan v1.5.0 statués `[~]` vers ce plan ;
      entrée de `.ai/thought_log.md` ; worktrees retirés un à un après recherche de jonction, dans un tour à
      part ; racines de données et caches dédiés supprimés.
- **Gate L** : chaque ligne du registre des changements de sortie retrouvée dans les mesures APRÈS ; CI de
  LevelUp verte sur `release/v8.3.0`.

## Journal d'exécution

- 2026-11-23 (soir) — FAITS CHIFFRÉS ÉTABLIS SUR PIÈCES : document au schéma 127 (126 en v1.5.0), faits au schéma 31 (30 en v1.5.0). Poids d'un rejeu compressé : +45 à +46 % depuis la v1.5.0 (trois films : 329 993 -> 482 325, 840 362 -> 1 225 879, 7 833 786 -> 11 368 190 octets ; pleine cadence et santé et bouclier par vie en sont la cause). Vitesse d'un film récent (Slayer, machine seule, total) : 1,12 s en v1.5.0, 0,80 s à un fil, 0,53 s à deux, 0,48 s à quatre, 0,46 s à huit (`releve_voies_fils1.tsv` à `releve_voies_fils8.tsv`). Pic de mémoire du plus gros film, un fil : 1 624 Mio à la cinquième intégration (2 198 Mio en v1.5.0). Tests de la suite complète : 4 538 passés, 301 sautés, seuls 4 tests de fixtures de contrat rouges (fixtures de l'hôte à recopier). Annexes versées : `journal_des_integrations.md` et les rapports de portage, de score, de drapeau et de verdicts E et F.

- 2026-11-23 (soir) — INTÉGRATION 6, tête `7bda46f` : `bl-score` puis `bl-divers`. Entrent : pénalité de suicide ou de trahison publiée (A.5), table des réponses de jeu associée à la révision de la grammaire (A.6), statistiques VIP en courbes par joueur (E.2), victime absente quand elle est ambiguë (E.6). Fusions sans conflit ; `Couche.Associes` devient une chaîne pour garder la structure comparable. Contrôles : 4 538 tests passés, 301 sautés, 4 rouges de contrat ; courses `replay` et `grammar` sans signalement ; `gorelease` : seules les deux constantes de schéma changent de valeur ; empreintes de 13 films égales à 1 et 4 fils ; trois films VIP égaux à 1 et 4 fils. Un des trois films VIP a 23 courbes VIP pour 8 joueurs (la courbe absente vaut zéro dans le film et dans l'API).

- 2026-11-23 (soir) — INTÉGRATION 5, tête `f13aacd` : `p-colles`. Entrent les projectiles collés, lus par la marche et publiés sur leur vol, plus un calque pour ceux qu'aucun vol ne porte. Fusion sans conflit (`movement_states.go` à exactement 500 lignes). Contrôles : 4 516 tests passés, 301 sautés, 4 rouges de contrat ; courses sans signalement ; `gorelease` inchangé ; empreintes à 1 fil = 4 fils pour les 13 films ; poids final 482 325 / 1 225 879 / 11 368 190 octets, pic du plus gros film 1 624 Mio.

- 2026-11-23 (soir) — INTÉGRATION 4, tête `46be3cf` : `p-place` puis `p-visee`. Entrent le tireur d'un tir par la place de contrôle lue dans le film, et les visées sans position lues sur tout film. Fusions automatiques sans conflit. Fixtures de contrat : 11 461 910 octets, sous le plafond de 11 Mio. Trois films d'entrées changent. Contrôles : 4 494 tests passés, 301 sautés, 4 rouges de contrat ; courses sans signalement ; `gorelease` inchangé ; empreintes à 1 fil = 4 fils pour les 13 films.

- 2026-11-23 (soir) — INTÉGRATION 3, tête `2a49ff4` : `p-cadence` et `vitalites`. Entrent la pleine cadence toujours publiée (l'option est retirée) et la série de santé et de bouclier par vie. Plafond des fixtures de contrat relevé à 11 Mio (mesure 11 354 146 octets). Poids gzip des trois films : 481 783 / 1 223 306 / 11 231 739 octets (v1.5.0 : 329 993 / 840 362 / 7 833 786). Contrôles : 4 466 tests passés, 301 sautés, 4 rouges de contrat ; courses sans signalement ; `gorelease` inchangé ; empreintes à 1 fil = 4 fils pour les 13 films ; pic du plus gros film 1 726 Mio.

- 2026-11-23 (soir) — INTÉGRATION 2, tête `ddf99f9` : `p-tirs`, `p-arme`, `p-socles`. Entrent le décalage du tir dans son image, les dégâts désignés par piste et à zéro point, l'arme tirée en main, les munitions portées, les candidats de socle, les socles d'équipement et la détonation d'un lancer sans vol. Fusions sans conflit. Plafond des fixtures de contrat relevé à 5 Mio (mesure 5 226 571 octets), fixtures d'entrées pseudonymisées et contrôlées (aucun identifiant d'origine). Contrôles : 4 438 tests passés, 301 sautés, 4 rouges de contrat ; courses sans signalement ; `gorelease` : un type de socle d'arme devenu non comparable, corrigé (`2ac0a7e`) ; empreintes à 1 fil = 4 fils pour les 13 films.

- 2026-11-23 (soir) — INTÉGRATION 1, tête `2ac0a7e` : `a7`, `a2a`, `a2b`, `a2c`, `b2d` (B.2a, B.2b, B.2d), `b2c`, `e1c`, `d3`, puis les deux files des gestes (`gestes-a`, `gestes-b`, avec des conflits d'union résolus en gardant les deux canaux, répliques puis gestes) et `portage-0` (source de film par chemin, catalogue en octets, axes du socle d'arme, munitions dans les faits). Poses de l'intégrateur : calques `chatter` et `announcements`, porte de cadence des gestes, schéma 127 et faits 31, révisions `grammar-2026-11-23` et `objectives-2026-11-23`. Contrôles : 4 393 tests passés, 301 sautés, seuls 4 tests de fixtures de contrat rouges (verts sur la racine bac à sable ; à recopier dans l'hôte : 8 fichiers 126 supprimés, 8 fichiers 127 et le manifeste ajoutés) ; courses `replay` et `grammar` sans signalement ; `gorelease` : retraits signalés (deux champs `SeatsBeyondBand`, trois compteurs de `ControlHoldStats`), conservés et dépréciés ensuite (`0cc5366`) ; empreintes à 1 fil = 4 fils pour les 13 films, toutes différentes de la v1.5.0 comme attendu.

- 2026-11-23 (après-midi) — Reprise après une coupure de courant : une référence git et un fichier en cours d'écriture abîmés, réparés ; aucun travail intégré perdu. Seconde partie du chantier 2 faite et mesurée (K.2b). G.10 intégré (pic du plus gros film −294 Mio). Sur leurs branches, prouvés, en attente de l'intégration commune du contenu : A.7, A.2a à A.2c (score), B.2a à B.2d (drapeau : 203 joueurs sur 218 à 2 s de l'API contre 161, aucun juste devenu faux), E.1c (éliminations de bots, par place), D.3 (répliques publiées), gestes (accroupi, saut, mêlée, lancer, rechargements). Décisions du user : le chantier 2b se fait dans la v1.6.0 ; les gestes ne se publient que pour la version actuelle du jeu (condition retenue : la cadence des commandes mesurée dans le film, deux films de même version ayant deux cadences) ; NOUS reprenons les ajouts du contributeur du lecteur 3D dans notre dépôt, avec une note de migration pour lui à la fin (la piste P devient un portage ; plans : `dec-v160-recup/pistes/gestes/portage.md` et `pistes/portage/portage_complet.md`, à verser en annexe après purge des pseudos cités).

- 2026-11-23 — Mesure du parallélisme faite machine au repos : un film récent passe de 1,1 s (v1.5.0) à 0,83 s à un fil, 0,62 s à deux, 0,57 s à quatre ; total des 13 films de 71,6 s à 34,8 / 29,0 / 27,0 / 26,2 s ; mémoire inchangée. L'attente d'une machine au repos par l'indicateur de charge de Windows était fausse (36 % affichés pour une machine inoccupée) : remplacée par le relevé des processus de test avant et après chaque niveau.

- 2026-11-23 — Chantier 1 du parallélisme (K.I1 à K.I4) et tâche de fond (K.J2) intégrés, tête `bf69129` : sorties identiques à tous les niveaux de fils, ordre des étapes identique. Un conflit de fusion résolu (la même méthode de liste des payloads écrite par deux agents). A.2a et A.2c prouvés sur films, sur leurs branches ; A.2b en preuve. Reste K.I5 : la mesure machine seule, à lancer quand plus aucun test d'une autre session ne tourne.

- 2026-11-23 — Décisions du user : la lecture au choix des canaux est abandonnée pour la v1.6.0 ; les répliques auront leurs textes et leurs voix en français et en anglais au dépôt (D.5), le poids étant bien plus faible qu'il ne le pensait. K.I1 et K.J2 codés, intégration en cours.

- 2026-11-23 — Fondations du parallélisme, K.J1 et D.1 intégrés (tête `cd8f560`), empreintes des 13 films identiques à chaque intégration. A.1 rend trois causes prouvées ; A.2 se découpe en trois corrections de lecture, A.2a lancée. K.I1, K.I4 et K.J2 lancés ensemble. F.1 et F.2 faits côté LevelUp. C.4 et E.5 rendus. V4 : conception rendue, chantier déconseillé, décision demandée au user. Piste V3 statuée : couverte par les profils et les pistes G et K.

- 2026-11-23 — K.0 intégré (tête `daac986`). Conception du parallélisme rendue et relue : chantier 1 sans refonte préalable, chantier 2 scindé en 2a (pendant la marche, risque faible) et 2b (à décider sur mesure). Fondations et K.J1 lancés. Le user précise sa cible : le serveur a peu de cœurs, l'app de bureau en aura davantage, les deux comptent. Pistes du backlog démarrées en parallèle, comme le plan le prévoit : A.7 fait sur `piste/a7` (à intégrer après preuve d'équivalence), D.1 fait sur `piste/d1` (lecture des répliques, rien de publié), A.1 en mesure, E.3 instruit : ni le film ni les fichiers lisibles du jeu ne donnent le nom des options ; source non établie, aucun code.

- 2026-11-23 — Sixième gain (G.4) intégré, tête `785c38e`, relevé à six gains pris machine seule. Profil des films récents rendu : pas de gain majeur sur un fil, la piste K (parallélisme) s'ouvre selon la priorité fixée par le user ; K.0 (rejet par mot, un fil) et K.C (conception, sans code) lancés ensemble. Le parallélisme par morceau, mesuré à blanc, reste écarté.

- 2026-11-23 — CINQ GAINS INTÉGRÉS : décodage −34,5 % sur le jeu de mesure, rejeux identiques
  Intégrés dans `feat/v1.6.0` (tête `4618ce3`) : G.2, G.6, G.1, H.1 (largeur d'en-tête d'un bipède calculée après les rejets de tête), G.8' (préfiltre exact des mots de 32 bits d'une image-clé, à la place d'une recherche en table à chaque bit) et G.5 (un essai se défait par un journal d'annulation, au lieu de copier la table des slots). G.7 reste retiré (−4 à −5 % sur le plus gros film seulement, +2 % reproductible sur un film ancien).
  Preuve de chaque gain : test différentiel contre l'ancienne forme gardée en référence de test (G.1 : 15 564 acceptations ; G.5 : 1 520 803 essais de deux films réels, dont 609 422 modifiant le monde, état complet comparé ; H.1 et G.8' : plus de 900 000 occurrences) ; mutants détectés.
  Preuve de l'état intégré : suite complète code 0, 34 paquets, 301 tests sautés, aucun échec ; `go vet` avec et sans l'étiquette de recherche ; lint 0 issue ; équivalence 20 identiques contre la v1.5.0 ; double passe : 20 artefacts identiques à l'octet, seuls écarts à classer les 20 de `killsource` ; relevé des 13 films, 5 passes, machine seule : 26 empreintes sur 26 identiques à la v1.5.0 ; empreinte de révision de la grammaire refigée à valeur constante (`grammar-2026-11-22.2`).
  Gain, tête contre le relevé de référence de la v1.5.0 (pris sous charge) : décodage cumulé des 13 films 68,2 s → 44,6 s (−34,5 %) ; total cumulé 71,6 s → 48,3 s (−32,5 %) ; plus gros film 19,6 s → 12,2 s ; film ancien 18,1 s → 10,6 s ; crâne 4,0 s → 2,5 s. Pic du plus gros film 2 198 → 1 969 Mio. Relevé : `.ai/annexes_decodeur_v160/releve_apres_cinq_gains.tsv`.
  G.4 lancé (agent Sonnet, effort élevé, branche `piste/g4`) ; V2 abandonnée sur analyse (`profil2.md`) ; G.3 gardé en dernier, à décider.

- 2026-11-23 — PISTE G : deux gains intégrés, un retiré
  Intégrés dans `feat/v1.6.0` : G.2 (la liste d'une trame non localisée est récupérée une fois) et G.6 (filtres de positions en place). Retiré : G.7 (candidats d'équipement écartés sur leur tête), −4 à −5 % sur le plus gros film seulement, +2 % reproductible sur un film ancien ; sa branche `piste/g7` est gardée. Le réexport des options par la façade est retiré (cycle d'import sous l'étiquette de recherche) : le porteur vit dans `replay` seul.
  Preuve sur l'état retenu : suite complète code 0, 34 paquets, 301 tests sautés (référence 301, et non 299 : même liste de noms, l'écart venait du comptage) ; lint 0 issue ; ajouts seulement, version suggérée v1.6.0 ; équivalence 20 identiques contre la v1.5.0 ; double passe : 20 artefacts identiques à l'octet, les seuls écarts à classer étant les 20 de l'étape `killsource`.
  Relevé apparié des 13 films, avec G.7 avant son retrait : décodage cumulé 70,0 s → 57,8 s (−17,5 %), total 73,7 s → 61,5 s (−16,5 %) ; films anciens et très gros −20 à −22 %, films récents −1 à −4 % (bruit) ; pic du plus gros film 2 192 → 2 051 Mio.
  Réserve : une autre session faisait tourner des tests sur la machine pendant les fenêtres de mesure ; le relevé de la version finale se refera machine au repos.
  G.1 est lancé (agent Sonnet, worktree `Scripts/dec-v160-g1`).

- 2026-11-23 — RELEVÉ DE RÉFÉRENCE ET PROFIL : les pistes de vitesse sont revues (aucun commit)
  Relevé de référence de la v1.5.0 rangé (`bench/releves/v1.5.0.tsv`, 13 films, 5 passes) : médianes de 0,5 s (13 morceaux) à 19,6 s (69 morceaux, pic 2 198 Mio pour un plafond de 3 Gio). D'autres processus tournaient pendant la fenêtre : ordres de grandeur sûrs, valeurs à quelques pour cent près non.
  Profil sur cinq films (`.ai/annexes_decodeur_v160/profil.md`) : décodage plus de 90 % du total, publication 5 à 10 %, décompression moins de 0,5 %, chargement moins de 1 %.
  Anomalie expliquée : sur les films dont la vue A n'est pas portée, la localisation de la liste d'événements refait un balayage bit à bit avec décodage d'essai, trois fois par trame : 73,5 millions de décodages d'essai contre 175 sur un film récent de même taille (ce dernier est 8 fois plus rapide).
  Faits : V0.1, V0.2, V0.3 et V0.6 intégrés dans `feat/v1.6.0` (V0.6 : un fil par défaut, borne 8) ; V0.5 fait ; V0.7 décalé.
  Décisions du pilote : V1 abandonnée (gain de 0,1 à 0,8 %) ; V3.2 abandonnée (déjà mémoïsée) ; V3.3 garde la récupération d'équipement ; V4 gardée ; V2 reportée ; piste G ajoutée.
  Lancés le 2026-11-23 : G.2, G.6, G.7 (Sonnet, effort moyen, branches `piste/g2`, `piste/g6`, `piste/g7`) ; G.1 après G.2.

- 2026-11-23 — GATE 0 PASSÉ, vague 1 lancée (aucun commit).
  Gate du corpus lancé à blanc comme test d'infrastructure (base `93a4744e7b`) : code 0, 19 témoins ok, zéro gain, zéro perte, 464 s ; registre des replis lu (127 entrées).
  Jeu fixe du relevé : 13 films (11 de l'équivalence, 2 témoins), pas de film à 33 joueurs ; table dans `films_du_releve.tsv`.
  Export de vérité fait (copie de la base du 2026-11-22, 2 045 matchs) ; mesures avant dans `avant.md`. Compteur des émissions jetées contrôlé (gate v1.5.0 et passage à blanc).
  Passage à blanc : 28 décisions « sans effet » signalées, attendu (il compare la v1.5.0 à elle-même) ; 7 retirées (F.2), 21 nécessaires jusqu'à la fusion de la montée.
  Vague 1 lancée (coût moyen annoncé) : relevé de vitesse V0.1 à V0.3 (Sonnet, `dec-v160-v0`, branche `piste/v0`) ; socle des options V0.6 (Sonnet, effort bas, `dec-v160-opt`, branche `piste/v0-options`).

- 2026-11-23 — Résumé du jour (aucun commit) :
  A et B : suite complète du clone à blanc (code 0, 33 paquets, 3 187 PASS, 299 SKIP, 0 échec ; 299 = référence) et tests de révision (13 tests, 18 avec sous-tests, verts, aucun sauté).
  C : équivalence contre les références, sans `-update` : 20 identiques, 0 différent (5 min 27 s).
  D et E : racine distincte utilisable seulement avec ses trois dossiers (un worktree LevelUp par piste) ; `gorelease` épinglé, « Suggested version: v1.5.1 », aucune incompatibilité ; couverture des paquets publics à vérifier.
  F : double passe trois fois, 20 artefacts identiques à chaque passe ; 20 écarts `killsource` fixes, cause prouvée. G : parcours de `map`, 1 suspect (`joinSegments`, item A.7).
  Reste pour le Gate 0 : gate du corpus comme test d'infrastructure, item 0.3 (jeu de films du relevé), item 0.8 (export de vérité et mesures avant).
- 2026-11-23 — GO du user. Clone `Scripts/dec-v160` sur `feat/v1.6.0` (tag `v1.5.0`, `dc967c2`), poussé ; worktree `Scripts/LevelUp-wt-dec160` sur `chore/830-decodeur-v160` depuis `93a4744e7b`, `npm ci` fait.
  Trois contrôles en lecture seule rendus par des agents Haiku : 0.5 et 0.6 (`controle_0_5_0_6.md`), 0.7 (`appelants_partiels_0_7.md`), inventaire du contributeur (`inventaire_contributeur.md`).
  Sept collisions de fichiers entre pistes (et avec le pilote) relevées et inscrites sous la table « Fichiers attendus par piste » ; sept autres avec le contributeur, arbitrées après le gel.
  Dossier d'accueil du contributeur (P.2) en cours d'écriture.
  Les items qui décodent des films (0.2, 0.4, 0.8) attendent la fin des gates de la v1.5.0.

- 2026-11-23 — plan écrit, puis relu par un second agent contre la grille `plan-review` ; corrections
  appliquées le même jour : preuve d'identité en commandes distinctes et niveau de fils réglable dans les
  outils de LevelUp ; règle des révisions de couche et des caches par piste ; deux relevés de vitesse ;
  relecture avant les gates finals ; V2 après les pistes du backlog ; DD-2 tranchée par le user ; DD-3
  tranchée par le pilote ; DD-7 et DD-8 ajoutées. Aucun code, aucun commit.

- 2026-11-23 — DÉCISIONS DU USER, plan mis à jour sur place, aucun code : le contributeur reporte lui-même
  ses ajouts sur la v1.5.0 dans les trois briques (« Oui [le contributeur] s'en occupe seul ») ; tout se fait dans la
  v1.6.0, sans échéance, aucune v1.7.0 ; DD-1 à DD-8 tranchées (section 2). Les pistes P et M deviennent la
  piste P « accueil du travail du contributeur », avec une règle de gel ; R-3 ne renvoie plus rien à une
  autre version ; vagues réduites à une intégration à la fois.
- 2026-11-23 — GO DU USER (« Tu as mon feu vert d'emblée »), avec la consigne de découper en morceaux plus
  petits : table d'exécution ajoutée (un item, un agent, le modèle et l'effort les moins coûteux qui
  suffisent, fichiers, dépendances, classe de charge) ; recherches découpées en mesure puis correction ;
  table des vagues réécrite avec ce qui tourne en même temps. Aucun item n'est lancé par cette mise à jour.

## Découvertes (consignées, non traitées)

- 2026-11-23 (intégrations) — 33 collages de projectile se posent sur une entité de l'archétype des véhicules que `vehicles[]` ne publie pas sous ce slot : ils restent sans cible (`unresolved`), aucune résolution forcée (déduit). Entrée du backlog.
- 2026-11-23 (intégrations) — Deux joueurs (API 483 et 514 tirs) masquaient un manque réel d'environ 200 tirs non écrits, que la lecture à tout rang fait apparaître ; renvoyé au chantier précision du backlog (DD-2).
- 2026-11-23 (B.3) — Le compteur `carrierTeamUnknown` porte un nom trompeur : il mesure l'abstention de la règle qui nomme le DRAPEAU pris (32 sur 32 portages d'un film à 6 socles), pas une équipe inconnue (l'équipe est lue pour 25 joueurs sur 25). Entrée du backlog.
- 2026-11-23 (montée de LevelUp) — Adaptations attendues côté LevelUp à la montée : ignorer les lignes de dégâts à zéro point ; accepter une élimination ou une action de score par `place` sans identifiant ; une courbe de score peut reculer (pénalité de suicide ou de trahison) ; les rejeux pèsent 45 à 46 % de plus, leur chargement dans le navigateur est à vérifier ; les 8 fichiers de fixtures de contrat 126 sont à remplacer par les 8 du schéma 127.
- 2026-11-23 (B.6) — Le calcul de la lettre des zones comme contrôle existe sur la branche `piste/bl-zones` (231 sur 231), sans appelant : volontairement non intégré, la source manque.
- 2026-11-23 (E.2) — Sur un des trois films VIP, 23 courbes pour 8 joueurs au lieu de 24 : la courbe absente est la meilleure série en VIP d'un joueur dont la valeur est 0 dans le film et dans l'API ; une courbe absente se lit zéro.
- 2026-11-23 (A.4) — Sur un film d'Assaut dont le groupe de scripts est mal résolu, des points publiés en trop (−2 400 à +450 d'écart) ; et un événement de détonation porte le joueur d'un autre siège que celui dont le score monte de 200 : à reprendre avec l'entrée du backlog sur les points de bombe.

- 2026-11-23 (A.2b) — Un comptage variable d'un lancement à l'autre sur un film a été élucidé : il vient du harnais de test, qui devine la carte au hasard entre deux candidates de même signature ; le décodeur publié n'est pas en cause (carte imposée : dix lancers identiques). Devenu l'item F.8.

- 2026-11-23 (A.1) — Quatre films sur 220 ont une manche écrite non retenue (29 à 38 enregistrements de joueurs, 8,7 à 13,1 %) et les manches 0 et 2 sans la 1 : manche réelle perdue ou numérotation, à examiner avec A.2c.
- 2026-11-23 (A.1) — Les builds `HI_1_10_0` et `HI_1_8_0` ont des centaines à des milliers de trames arrêtées sur de nombreux genres : table des genres non lisible sur ces builds.
- 2026-11-23 (A.1) — Le verdict ne distingue pas un tableau en retard par la pénalité de −100 d'un tableau en retard par une action de trop.
- 2026-11-23 (K.F4) — `film-bench -etapes` : trois étapes par film (`placements.stats`, `pads`, `vehicles`) ne s'écrivent pas en JSON ; leur empreinte est celle du texte de l'erreur.

- 2026-11-23 — L'en-tête de `LU apps/go-api/cmd/killsource/main.go` (lignes 3 à 8) affirme qu'aucun chemin de l'application n'utilise le décodeur killsource : faux (appels de production dans `internal/replaybuild/kills.go:57` et `internal/sync/killcollector/collector_run.go:215`).
- 2026-11-23 — Un commentaire du test de surface de la façade, `LU apps/go-api/internal/archlint/film_facade_surface_test.go` (ligne 166), dit que `NomPrecisionParArmePasseSautee` et `NomDistancesDeToucheDesactivees` ont quitté la façade en v1.5.0 ; ils y restent, marqués `Deprecated` (`decfilm/replis_obsoletes.go`). Incohérence à trancher.
- 2026-11-23 — 113 parcours de `map` classés sans effet par un outil, non relus un par un (table sous `Scripts/dec-v160-recup/etape0/maprange_classes3.tsv`) : à relire avant l'intégration de la lecture de plusieurs chunks à la fois.
- 2026-11-23 — Publication mesurée 3 % plus lente qu'en v1.5.0 (12 à 23 ms sur le plus gros film), sans cause identifiée au profil ; écart accepté par le pilote (moins de 0,3 % d'un rejeu), à remesurer sur la version finale. Le profil du même jour (`.ai/annexes_decodeur_v160/profil2.md`, section 1.2) chiffrait la même régression à +6 à +12 % sur quatre films en A/B : deux mesures d'écart différent, à rapprocher sur la version finale.
- 2026-11-23 — Des durées prises pendant que d'autres agents mesurent sont inexploitables (une ronde entière faussée) ; les empreintes restent fiables. Les durées se mesurent à part, machine seule.
