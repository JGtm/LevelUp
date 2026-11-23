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
Contrat d'exécution : skill `plan-execution`, complément `levelup-plan`. Statuts : `[x]` fait, `[~]` couvert
ailleurs (référence), `[!]` non traité (motif écrit). Aucune case vide à la clôture.
Plans liés : `.ai/PLAN_DECODEUR_2026-11-18.md` (v1.5.0, dont ce plan reprend la forme et les règles),
`.ai/PLAN_REJEU_3D_2026-11-15.md` (décision D-18), `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`.

Abréviations : `DEC` = dépôt `JGtm/halo-infinite-film-decoder` (PUBLIC), lu pour l'instruction dans
`C:/Users/Guillaume/Downloads/Scripts/dec-v150` (branche `feat/v1.5.0`, même arbre que le tag `v1.5.0`,
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
- **R-4 — une piste, une branche, des écarts attribués.** Clone neuf `C:/Users/Guillaume/Downloads/Scripts/dec-v160`
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
  worktrees et sorties sous `C:/Users/Guillaume/Downloads/Scripts/`. Une branche de travail est poussée dès
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
  l'item annonce (un `-run` sans correspondance répond `ok` : ce n'est pas un vert). Référence : 299 SKIP
  attendus sur la suite complète à blanc au tag v1.5.0 ; tout nombre supérieur s'explique avant de conclure.
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

Référence de V1, V3 et V4 : la v1.5.0 (références du clone, jamais refigées avant la fin de la phase).
Référence de V2, qui s'intègre après les pistes du backlog : le MÊME commit au niveau 1.

1. G-equiv-ref aux niveaux 1, 2, 4 et `N`, puis sous `GOMAXPROCS=1` et `GOMAXPROCS=2` au niveau `N`.
   Pour V2 : au niveau 1, contre les références de la branche (aucun écart admis) ; aux niveaux 2, 4 et `N`,
   empreintes par étape (`-out-dir`) égales à celles du niveau 1 du même commit.
2. G-equiv-2p au niveau `N`.
3. Trois passes complètes des 20 films au niveau `N` (G-equiv-ref trois fois) : trois fois 20 identiques.
4. Relevé A (V0) sur les 13 films au niveau 1 et au niveau `N` : empreintes du document et des faits égales
   entre elles et égales à celles du relevé de référence (v1.5.0 pour V1, V3, V4 ; niveau 1 du même commit
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
| 2 Vitesse 1 | V1.1 puis V1.2-3 (B) ; conceptions V4.1, V3.1, V2.1 (S) ; corrections A.2, A.5, A.6 (B, un agent, à la suite) ; mesures A.3, A.4, C.2, C.3 (L) | V1.4 (M) ; preuve de V1 (L) | moyen |
| 3 Vitesse 2 | V4.2-3 (B) ; C.1c et C.2c si prouvées (B) ; D.1 (B) ; F.1, F.3, F.4, F.7 (Haiku) ; mesures B.3, B.7, D.2, E.1 (L) | V4.4 (M) ; preuve de V4 (L) | moyen |
| 4 Vitesse 3 | V3.2, puis V3.3, V3.4 (B) ; B.2, B.4, B.6 (B, un agent, à la suite) ; D.3, D.4 (B, S) ; E.3, E.6 (B) ; mesures E.4, F.5, F.6 (L) | V3.5 (M) ; preuve de V3 (L) | moyen |
| 5 Intégration « vitesse » | V2.2 (S) ; E.2 après A (B) ; E.5, C.4 (S, Haiku) ; F.2 (Haiku) | I.1 à I.3 : preuve complète (L), gate du corpus, relevés (M) | élevé (machine) |
| 6 Intégrations du backlog | V2.3-4 (B, Opus) ; B.5 si sa condition est tenue (B) ; D.5, D.6 côté LevelUp (B, S) | A, C, B, D, E, F intégrées une à une (L), schéma 127 à la première | élevé |
| 7 V2 | — | V2.5 (M), preuve de V2 (L), intégration | moyen |
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
| V1.1 | Sonnet, bas | S | conception au plan | V0.5 |
| V1.2, V1.3 | Sonnet, moyen | B | `internal/source/`, `filmcache/`, `decfilm/` | V1.1, Gate V0 |
| V1.4 | pilote | M | relevés | V1.3 |
| V4.1 | Sonnet, moyen | S | conception au plan | V0.5, 0.7 |
| V4.2, V4.3 | Sonnet, élevé (plusieurs fichiers) | B | `replay/film_scan*.go`, `build_from_film.go`, `movement_states.go` | V4.1, V1 intégrable |
| V4.4 | pilote | M | relevés | V4.3 |
| V4.5 | Haiku, bas | S | fragments de crédit | — |
| V3.1 | Sonnet, moyen | S | candidats au plan | V0.5 |
| V3.2 | Sonnet, élevé (trois lecteurs, une recherche) | B | `marche_images_cles*.go`, `keyframe_etat_complet_*.go`, `keyframe_loadout.go` | V3.1, V4 |
| V3.3 | Sonnet, bas | B | sonde de calibration | V3.2 |
| V3.4 | Sonnet, moyen, un agent par candidat | B | selon V3.1 | V3.2 |
| V3.5 | pilote | M | relevés | V3.4 |
| V2.1 | Sonnet, élevé (instruction, sans code) | S | `v2_conception.md` | V0.5 |
| V2.2 | Opus, moyen (motif : conception d'une lecture concurrente sur un état partagé) | S | conception au plan | V2.1 |
| V2.3, V2.4 | Opus, élevé (même motif : l'erreur est silencieuse) | B | `marche_trames*.go`, `distribuer*.go`, `world.go`, `keyframe_liaison.go` | V2.2, V4, V3 |
| V2.5 | pilote | M | relevés | V2.4 |
| I.1 à I.3 | pilote ; Haiku, bas pour refiger les empreintes sous sa conduite | L, M | intégration | Gates V1, V4, V3 |
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

Dépendances entre pistes de vitesse, dites : V1, V2 et V4 lisent toutes la structure d'options de V0.6 et
aucune ne l'écrit ; V2 et V4 touchent toutes deux la distribution aux canaux (V4 les inscriptions, V2 le
distributeur) : V2 passe après V4 ; V3 passe après V4. Les pistes de vitesse passent donc une à une.
Le travail du contributeur croise V0.6, V1, V3, V4, C, D et tous les fichiers à écrivain unique : c'est
pourquoi il entre APRÈS le gel, sur une base qui ne bouge plus (piste P).

Fichiers à écrivain unique, le pilote, à l'intégration seulement : `replay/document_schema.go`,
`document_chronicle.go`, `layers.go`, `coverage.go`, `filmfacts*.go`, tous les goldens, les fichiers de
révision de couche, les références d'équivalence, les fixtures de contrat, et les fichiers communs de R-15.
Aucun agent ne monte le schéma ni ne régénère un golden : il rend la liste exacte de ce qu'il fait rougir.

## Étape 0 — témoins, références, instruments (pilote ; un agent Sonnet)

- [ ] 0.1 Clone neuf `Scripts/dec-v160` à la tête de `main` ; vérifier `main` = `v1.5.0` (`dc967c2`) ;
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

- [ ] V0.1 Décision du 2026-11-23 : le jeu ne reprend que 2 des 10 témoins de durée, les autres étant absents de l'archive ; pas de film à 33 joueurs.
      Jeu de films fixe, 13 films, critères écrits dans `bench/README.md` : les dix témoins de durée du
      2026-11-10 (builds `HI_1_12_0` et `HI_1_13_0` ; Slayer, Drapeau, Bases, Roi de la colline, Balle,
      Grande bataille à 33 joueurs ; 25 à 67 chunks) ; un film `HI_1_11_0` de Grande bataille et un film de
      version 39 ou 40, pris dans les 20 de l'équivalence ; le film de plus fort pic de mémoire du parc
      (0.3). `bench/films.tsv` : nom neutre, build, mode, module de carte, joueurs, chunks, octets
      compressés. Aucun fichier du jeu, aucun chunk, aucun identifiant de joueur ni de match dans le dépôt
      public. Second jeu, « public » : les mini-bobines du dépôt, pour que la commande tourne chez quiconque
      et en CI ; sans catalogue de bornes dans le dépôt, il ne mesure que le chargement (à établir ici).
- [ ] V0.2 **Relevé A, dans le décodeur.** Commande `cmd/film-bench`, bâtie sur `research/duree`, par les
      SEULES entrées publiques, options MINIMALES (carte seule) :
      `GOWORK=off go run ./cmd/film-bench -films bench/films.tsv -correspondance <fichier hors dépôt>
      -racine <film_chunks> -catalogue <map_quant_bounds.json> -passes 5 -fils 1 -sortie bench/releves/<version>.tsv`.
      Phases, lues par l'observateur existant : **chargement** (lecture, décompression et découpage,
      ensemble), **décodage** (du film aux faits, jusqu'à l'étape `clockOrigin`), **publication** (des faits
      au document), **total**, **pic de mémoire**. La **décompression** est chronométrée à part, par
      `decfilm.Inflate` sur les octets de chaque chunk lus avant le chronomètre : c'est une colonne à côté,
      pas une part du chargement. Écrit dans `bench/README.md` : le total du relevé A N'EST PAS le temps
      d'une génération de rejeu (ni source de dégât, ni tableau, ni objectifs, ni zones).
- [ ] V0.3 Protocole de mesure, écrit dans `bench/README.md` : fichiers lus avant le chronomètre ; un
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
- [ ] V0.5 Relevés de RÉFÉRENCE de la v1.5.0, par le pilote, en fenêtre de mesure. Relevé A : l'outil est
      copié, sans être commité, dans un worktree posé sur le tag `v1.5.0` (il n'emploie que des entrées
      publiques de cette version) ; rangé `bench/releves/v1.5.0.tsv`. Relevé B : branche de montée au
      `require` v1.5.0. Profil du temps et des allocations sur 3 films (`-cpuprofile`, `-memprofile` de
      `replay-build`) : part de chaque phase et des dix premières fonctions, dans
      `.ai/annexes_decodeur_v160/profil.md`. C'est l'entrée des conceptions V1.1 à V4.1.
- [ ] V0.6 Socle des options (R-16), un seul écrivain : une structure d'options AJOUTÉE (nombre de fils,
      choix des canaux), valeur zéro = comportement actuel, sans aucun effet à ce stade ; plafond du nombre
      de fils dans la brique (R-12). Intégrée dans `feat/v1.6.0` avant toute piste V.
- [ ] V0.7 Côté LevelUp, sur la branche de montée (pseudo-version de V0.6) : niveau de fils réglable par
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

- [ ] V1.1 Conception écrite ICI avant le code, après lecture de la méthode du parseur voisin
      (`src/theater/parser/chunk.rs`) et du profil V0.5 : chunks rangés par indice ; erreurs rendues dans
      l'ordre des chunks (la première par indice, pas la première arrivée) ; tolérances de `inflate`
      inchangées (entrée claire, flux tronqué, zstd abîmé) ; au plus autant de chunks en cours de
      décompression que de fils.
- [ ] V1.2 Code : `source.Load`, `decfilm.Load` et `decfilm.LoadDir` gardent signature et comportement ; une
      entrée de chargement AJOUTÉE lit les options de V0.6. Le découpage en paquets reste dans l'ordre.
- [ ] V1.3 Tests neufs sur les mini-bobines (noms contenant `Fils`) : octets clairs et paquets égaux à un
      fil et à N fils ; erreur d'un chunk du milieu égale dans les deux cas ; ordre de fin perturbé par un
      délai aléatoire ; sous `-race`.
- [ ] V1.4 Mesure par le relevé A aux niveaux 1, 2, 4 et `N` : chargement, décompression, total, pic ; par
      le relevé B au niveau retenu.
- **Gate V1** : preuve d'identité (points 1 à 6) ; G-clone sur `./internal/source/ ./filmcache/ ./decfilm/
  -run Fils -race -count=3` (au moins 3 tests `PASS`) ; G-rev (rouge attendu sur la couche `source` et, par
  sa valeur amont, nulle part ailleurs : liste rendue au pilote) ; G-compat ; au niveau 1, pic et total dans
  le bruit du relevé v1.5.0.

## Piste V4 — lecture au choix des canaux (Sonnet)

- [ ] V4.1 Conception écrite ICI avant le code : le choix des canaux de V0.6 (valeur zéro = tous les
      calques = comportement actuel) ; table des dépendances entre calques, tirée de `layers.go` et des
      étapes de `observe.go` (un calque demandé entraîne ceux dont il se nourrit : identité, pont, fil des
      morts) ; ce que la marche peut ne pas distribuer quand aucun calque demandé n'y a d'intérêt. Les
      appelants à servir sont ceux de 0.7.
- [ ] V4.2 Code : clôture des dépendances, canaux non demandés non inscrits à la marche. Le document partiel
      porte les MÊMES octets pour les calques demandés ; les calques non demandés sont absents, et `layers`
      le dit par ce qui existe déjà (aucun champ neuf).
- [ ] V4.3 Garde-rail : un test échoue si un calque est ajouté à `layers.go` sans ligne dans la table des
      dépendances.
- [ ] V4.4 Mesure par le relevé A : document complet (inchangé) ; puis positions seules, tirs seuls, fil des
      morts seul : décodage, publication, total, pic.
- [ ] V4.5 Crédit à nuzzles, fragment rendu pour les pistes V (R-15) : `README.md` (section « Speed »,
      méthodes reprises, lien vers son dépôt), `NOTICE.md` (remerciement, licence de sa source, révision
      lue), une ligne en tête de chaque fichier qui porte une méthode reprise, notes de la release, et côté
      LevelUp les notes de version et la page des crédits (L.5). `FORMAT.md` n'est pas touché : le format ne
      change pas.
- **Gate V4** : preuve d'identité pour le document complet (points 1 à 3, 6) ; pour chaque sélection
  mesurée, les empreintes par étape de `replay-equiv -out-dir` des calques demandés sont égales à celles du
  document complet sur les 20 films ; G-clone sur `./replay/ -run 'Canaux|Calques'` (au moins 3 tests
  `PASS`, dont le garde-rail V4.3) ; G-rev ; G-compat.

## Piste V3 — lectures refaites et état à la demande, réduit par DD-7 (Sonnet)

- [ ] V3.1 Instruction sans code : liste fermée des états construits d'office pour des lecteurs qui ne les
      demandent pas, et des lectures refaites, avec leur coût dans le profil V0.5 et leurs lecteurs ;
      méthode du parseur voisin relue (`src/theater/runtime/mod.rs`) ; motif d'abandon de sa forme littérale
      écrit (DD-7). Sortie : candidats écrits ici.
- [ ] V3.2 Images-clés : une seule recherche pour les trois lecteurs (état complet du bipède, dotations de
      naissance, équipes).
- [ ] V3.3 Sonde de calibration et récupération d'équipement passées derrière la marche, seulement si le
      profil V0.5 les montre au-dessus de 1 % du total ; sinon `[!]` avec le chiffre.
- [ ] V3.4 Les candidats de V3.1 retenus par le pilote, un item écrit ici par candidat avant son code ; un
      candidat qui toucherait l'état du monde de la marche est refusé ici (il appartient à V2).
- [ ] V3.5 Mesure par les relevés A et B : décodage, publication, total, pic.
- **Gate V3** : preuve d'identité (points 1 à 4, 6 ; un seul niveau si la piste ne lance aucun fil) ;
  G-clone sur `./internal/grammar/ ./replay/ -run 'ImagesCles|Keyframe'` (nombre de `PASS` au moins égal à
  celui de la tête avant la piste, relevé par l'agent) ; G-rev ; G-compat.

## Intégration « vitesse » (pilote)

- [ ] I.1 V1, V4 puis V3 fusionnées dans `feat/v1.6.0`, une à une ; G-equiv-ref après chacune : aucun écart,
      références non refigées ; empreintes de révision refigées par leur porte, valeurs inchangées (R-13) ;
      G-façade.
- [ ] I.2 Preuve d'identité complète, points 1 à 7 (gate du corpus au niveau `N`, pseudo-version désignée
      par `chore/830-decodeur-v160`, `GOWORK=off`).
- [ ] I.3 Relevés intermédiaires A (`bench/releves/v1.6.0-vitesse.tsv`, quatre niveaux) et B ; tableau avant
      et après présenté au user, avec le pic par niveau.
- **Gate I** : G-clone sur `./...` vert ; G-rev vert ; 20 identiques à chaque niveau et en double passe ;
  gate du corpus sans gain, perte ni décision.

## Piste V2 — lecture de plusieurs chunks à la fois (Opus ; s'intègre APRÈS les pistes A à F)

- [ ] V2.1 Instruction sans code, dans `.ai/annexes_decodeur_v160/v2_conception.md` (vague 2) : ce qu'un
      chunk hérite du précédent dans notre marche (table d'entités du monde, liaisons des images-clés, table
      anticipée, neufs suspendus, profil de balayage, compteurs de replis, crochets des canaux, marche
      commune de la source de dégât) ; à quelle condition un chunk peut partir d'une copie de l'état ;
      comment on RECONNAÎT qu'il dépendait du précédent ; méthode du parseur voisin relue
      (`parser/v41/chunks.rs`) ; part de la marche des trames dans le profil V0.5. Si cette part est sous
      20 % du total, ou si la dépendance ne se reconnaît pas par un fait écrit, la piste s'arrête : `[!]`
      avec la mesure (abandon écrit et motivé, R-3).
- [ ] V2.2 Conception relue par le pilote puis écrite ICI, datée, avant le code : forme de la copie d'état,
      règle de relecture en séquence, ordre de remise aux canaux (les crochets publient pendant la marche :
      leur ordre d'appel reste celui d'un fil), borne de mémoire (copies d'état en vol au plus égales au
      nombre de fils).
- [ ] V2.3 Code derrière les options de V0.6 (zéro ou un fil = la marche actuelle, chemin inchangé).
- [ ] V2.4 Tests neufs (noms contenant `Parallele`) : empreinte de la structure de lecture égale à un fil et
      à N fils sur les deux mini-bobines à trames (`marche_determinisme_test.go` étendu) ; chunk qui dépend
      du précédent forcé par construction ; ordre de fin perturbé par un délai aléatoire ; `-race -count=3`.
- [ ] V2.5 Mesure par les relevés A et B aux quatre niveaux : décodage, total, pic, relectures en séquence
      par film. Aucun film du jeu fixe ne dépasse 2 Gio à aucun niveau ; un niveau qui dépasse est retiré
      des niveaux permis.
- **Gate V2** : preuve d'identité dans sa forme V2 (points 1 à 7) ; empreinte des FAITS égale à celle du
  niveau 1 sur les 20 films ; aucun compteur de repli du document ne bouge ; G-clone sur
  `./internal/grammar/ -run 'Parallele|Determinis' -race -count=3` (au moins 4 tests `PASS`) ; G-rev ;
  G-compat ; relecture du diff par le pilote avant intégration (état partagé, ordre des crochets).

## Piste A — score et tableau (Sonnet ; Opus pour A.3)

- [ ] A.1 Mesure sans code, sur les films du parc par le harnais : les cinq Grandes batailles restées
      « incomplètes » sans rang non confirmé ni manque ; la Grande bataille « complète » à tort (points
      publiés au-dessus du tableau) ; le film dont le tableau cesse de monter dans les dernières secondes.
      Cause par film, ou `[!]`.
- [ ] A.2 Selon A.1 : correction du verdict, écrite ici avant son code.
- [ ] A.3 Recherche (une session) : arbre des actions décalé sur le build qui précède `HI_1_11_0` (31
      lectures concordantes, égalité non testable sur des films à trames arrêtées) et les 34 Grandes
      batailles à drapeau sans groupe de scripts. Sortie : décalage PROUVÉ par le tableau du film, ou `[!]`.
- [ ] A.4 Assaut : points de bombe hors de l'arbre des scripts (4 films). Recherche bornée ; publication si
      le film les nomme.
- [ ] A.5 Courbe de score qui recule : la pénalité de suicide ou de trahison (−100) publiée comme telle, au
      lieu de la plus longue suite non décroissante. Mesure attendue : accord au score personnel de l'API en
      hausse, aucun joueur exact avant ne devient inexact.
- [ ] A.6 Révision de couche pour la table des réponses de jeu : une ligne ajoutée à la table rend un rejeu
      « à redécoder ».
- [ ] A.7 (ajouté le 2026-11-23, issu de 0.4) `joinSegments` (`replay/score_timeline_players.go`) : le comparateur
      devient total, instant de début PUIS slot ; test unitaire à deux segments de même début, ordre d'insertion
      inversé ; effet attendu sur les 20 films : aucun ; change le cas d'égalité, aujourd'hui indéterminé.
      Sonnet, effort bas, classe B.
- **Gate A** : équivalence attribuée (G-equiv-ref, chaque écart rattaché à un item) ; accord au score
  personnel de l'API (export 0.8) sans baisse pour aucun build ; G-clone sur `./reponsesdejeu/ ./replay/
  ./internal/facts/objectives/` ; G-rev ; G-compat.

## Piste B — drapeau et zones (Sonnet ; Opus pour B.1)

- [ ] B.1 Recherche (une session) sur les 16 portages à l'état « au socle » faux : porteur précédent hors
      des 8 premières places ou sur une place sans humain nommé (7) ; variante à un seul drapeau publiée
      avec deux drapeaux d'équipe (4) ; résidu (5). Une cause prouvée par famille, ou `[!]`.
- [ ] B.2 Corrections prescrites par B.1, écrites ici avant leur code.
- [ ] B.3 Équipe du porteur sur une carte à 6 socles (25 portages sur 25 d'un film) : lecture de l'équipe
      par le film, ou `[!]`.
- [ ] B.4 Compteurs de zones sur d'anciens builds (octet signé qui avance de 32) : borne par build, lue dans
      les définitions du film.
- [ ] B.5 Ouverture de la bande au-delà de 8 joueurs, SEULEMENT si, après B.2 et B.4, le temps de porteur
      est à 2 s de l'API pour au moins autant de joueurs au-delà de 8 qu'en deçà (91,7 % en v1.5.0) ; sinon
      `[!]` avec la mesure.
- [ ] B.6 Lettre des zones comme contrôle : deux compteurs de couverture (lettres lues, lettres en accord
      avec le rang publié) ; lecture du composant du marqueur qui arrête la lecture aux images-clés.
- [ ] B.7 Zone au propriétaire de nom inconnu (un film) : cause, puis lecture ou `[!]`.
- **Gate B** : équivalence attribuée ; temps de porteur contre l'API sur les 186 films à drapeau : aucun
  porteur juste avant ne devient faux ; compteurs de zones contre l'API sans baisse ; G-clone sur
  `./replay/ ./internal/grammar/` ; G-rev ; G-compat.

## Piste C — gestes (Sonnet)

- [ ] C.1 Accroupissement et glissade sur les versions récentes : le bit de valeur 1 de l'entrée de contrôle,
      recoupé avec la vitesse et la hauteur sur 20 films. Vérité : les 7 rejeux qui les publient déjà par
      l'autre voie. Publication si l'accord atteint 90 %.
- [ ] C.2 Bouton de saut sur les builds 1.9 à 1.11 (accord de 50 à 61 %) : disposition de la vue de contrôle
      de ces builds ; publication du bouton seulement à 90 % d'accord sur tous les builds, sinon `[!]`.
- [ ] C.3 Sens des quatre booléens d'un rechargement : mesure contre les munitions lues ; nommés s'ils sont
      prouvés, sinon laissés tels quels.
- [ ] C.4 Recoupement avec les canaux du contributeur (manche, postures de contrôle, gestes), sans code :
      table « son champ, notre champ, lequel reste », versée à l'inventaire P.1 et au dossier P.2.
- **Gate C** : équivalence attribuée ; poids ajouté au document sous 1 % sur 20 films ; G-clone sur
  `./replay/ ./internal/grammar/` ; G-rev ; G-compat.

## Piste D — répliques des Spartans (Sonnet ; dans la v1.6.0, DD-4)

- [ ] D.1 Lecture par la grammaire des genres 75 et 76 de la vue A (locuteur, identifiant du son, priorité,
      indice de réplique ; annonceur : identifiant du son, destinataires), aujourd'hui traversés sans valeur.
      Contrôle : mêmes comptes que le témoin sur 3 films.
- [ ] D.2 Nom court de la réplique, DANS le décodeur (cadrage du user) : recherche bornée de la
      correspondance entre l'indice du film et l'entrée des tables de vocalisation (5 noms résolus à ce
      jour). Un nom est publié quand il est prouvé ; une réplique dont le nom n'est pas prouvé est publiée
      sans nom, et comptée. La recherche suit R-3. Aucun nom inventé.
- [ ] D.3 Publication : calque `chatter[]` (image, joueur, son, genre, nom court s'il existe) et sa
      couverture (lus, publiés, sans joueur, hors du match) ; les annonces de l'annonceur dans un calque à
      part si D.1 les lit. Le film ne dit pas quelle variante audio a joué : le document ne le dit pas non
      plus.
- [ ] D.4 Fragments de documentation (R-15) : `FORMAT.md`, `GLOSSARY.md` ; crédit à Den (méthode et table de
      textes, libfilmshell, MIT) pour `NOTICE.md`.
- [ ] D.5 HORS DÉCODEUR (DD-4) : côté LevelUp, à la montée, commande qui bâtit
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

- [ ] E.1 Éliminations dont un bot est l'auteur : mesure d'abord (combien `eliminations[]` en porte sur les
      films à bots, combien `scoreActions` en nomme) ; puis publication avec un auteur sans identifiant et
      sa place. Vérité : le tableau du film.
- [ ] E.2 VIP : temps en VIP, plus long temps en VIP, meilleure série en VIP, publiés en courbes par joueur
      comme les compteurs du tableau. L'hôte dit que le match est un VIP par la constante déjà exportée
      (`decfilm.ObjectiveTypeVip`). Après la piste A (mêmes fichiers).
- [ ] E.3 Nom des options de variante : le document publie le nom à côté du hachage quand le FILM ou les
      fichiers du JEU le donnent (chaîne lue, contrôlée par son hachage). La table du parseur voisin n'est
      ni copiée ni consultée comme source. Une option sans nom lu reste un hachage, comptée.
- [ ] E.4 Apparence enregistrée des Spartans (armure, revêtement) : recherche bornée (une session), sur 3
      films ; publication si la lecture est prouvée contre la personnalisation publique de l'API ; sinon
      `[!]`. À remettre à la session du rejeu 3D dans les deux cas.
- [ ] E.5 Zoom et coup de mêlée sans mort : contrôle sur pièces de ce qui est déjà publié (visée) et de ce
      que les gestes du contributeur couvrent (coup) ; sort écrit, versé à l'inventaire P.1.
- [ ] E.6 Victime d'une élimination appariée au plus proche : victime absente et compteur d'ambiguïté quand
      une action a plusieurs candidats.
- **Gate E** : équivalence attribuée ; éliminations et statistiques VIP contre l'API sans baisse ; G-clone
  sur `./replay/ ./internal/grammar/ ./internal/facts/objectives/` ; G-rev ; G-compat.

## Piste F — contrôles et petites entrées (Haiku ; Sonnet)

- [ ] F.1 `scripts/briques-workspace.sh` (LevelUp) : sous Windows, l'entrée `use` du clone s'écrit avec des
      barres inverses ; témoin : `go test` d'un sous-paquet du clone passe avec le `GOWORK` engendré.
- [ ] F.2 Les sept décisions devenues sans effet de `config/replay_corpus.toml` (2026-11-16) retirées
      (DD-8) ; leur liste est écrite au journal et montrée dans le rapport d'exécution. Précision du
      2026-11-23 : le retrait ne porte que sur les 7 décisions du 2026-11-16 : `coverage.hits.{noShooter,
      stoppedFrames, vitalsUnread}`, `coverage.flagCarries.{seatsBeyondBand, gaugePoints}`,
      `flagCarries.spans.returnProgress/{n, total}`. Les 21 décisions du 2026-11-22 restent nécessaires tant
      que la montée du décodeur n'est pas fusionnée dans la branche de version (la base du gate final sera
      alors le décodeur v1.4.1).
- [ ] F.3 Repli du voisinage des pièces montées : compté sur les films à véhicules par le harnais ; à zéro
      partout, l'entrée du registre est retirée et sa constante gardée, marquée dépréciée.
- [ ] F.4 Temps forts « mode » sans action de score (3 410 en Drapeau, 691 en Total Control, 303 en
      Extraction) : recroisés avec les huit noms d'action neufs de la v1.5.0. Verdict écrit ; aucun code si
      rien ne se prouve.
- [ ] F.5 Décalage de 35 s entre le film et le temps de jeu de l'API : une session de mesure (ce que l'API
      compte avant le coup d'envoi et après la fin). Cause, ou `[!]`.
- [ ] F.6 Lâchers 2 à 4 images après une mort (4 cas) : un fait du film qui nomme le propriétaire du lâcher,
      ou `[!]` (aucune tolérance de temps).
- [ ] F.7 Cinq rejeux sans build : pourquoi aucun compteur nommé n'y est lisible ; verdict écrit.
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

- [ ] P.1 (2026-11-23 : inventaire rendu dans `.ai/annexes_decodeur_v160/inventaire_contributeur.md` ; non coché : l'ordre des tags des trois briques et la table par canal, exigés ci-dessous, ne figurent pas encore dans ce rendu) Inventaire en lecture seule de ses copies, en deux parts confiées à deux agents (P.1a le
      décodeur, P.1b les deux autres briques), pour PRÉPARER l'accueil (aucun code, aucune PR), sur la
      tête de son dépôt au jour de l'item : pour le décodeur, diff de `third_party/film-decoder` contre
      v1.3.6 puis contre v1.5.0 ; pour les deux autres briques, diff à trois voies contre nos v1.4.0 et
      v1.2.0. Par canal : fichiers, schéma qu'il prend chez lui, recouvrement avec la v1.5.0 et avec nos
      pistes (tables de C.4 et E.5), règle du dépôt en jeu, ajout d'une autre brique dont il dépend, sort du
      codec `REPLAYINPUTS30`, dépendance neuve (`woozymasta/bcn`, licence à relire), fichiers du jeu omis.
      Rendu : `.ai/annexes_decodeur_v160/inventaire_contributeur.md`, avec l'ORDRE DES TAGS des trois
      briques.
- [ ] P.2 Dossier remis au contributeur, en anglais, `.ai/annexes_decodeur_v160/accueil_contributeur.md`,
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
- [ ] P.3 Points de synchronisation, annoncés au contributeur par le user ou par une issue du dépôt : fin de
      la phase vitesse (les options et le balayage du film ont leur forme finale) ; première intégration au
      schéma 127 ; gel. À chacun : le commit, la liste des fichiers de la liste des collisions qui ont bougé.
- [ ] P.4 Gel : toutes nos pistes intégrées ou statuées, `feat/v1.6.0` verte (G-clone sur `./...`, G-rev,
      G-compat, G-equiv-ref à 20 identiques sur références refigées) ; commit de gel annoncé ; plus aucune
      écriture de notre côté hors R-17.
- [ ] P.5 Vérification à l'arrivée, PAR PR du décodeur, dans cet ordre, sur une branche de contrôle (jamais
      sur `feat/v1.6.0`) : contrôle des identifiants et des fichiers du jeu (R-5) ; relecture par un agent
      Sonnet qui n'a pas écrit le code, axes : heuristique ou dérivation à la place d'une lecture, repli
      sans nom, sortie changée non déclarée, recouvrement avec un champ existant ; G-clone sur `./...` ;
      G-rev ; G-compat ; équivalence ATTRIBUÉE (chaque écart rattaché à ce que la PR déclare) ; G-equiv-2p ;
      garde de la précision R-14 pour tout canal qui touche tirs, touches ou dégâts ; poids ajouté au
      document mesuré ; relevé A à un fil (total dans les 10 % du relevé d'avant la PR, sinon retour au
      contributeur avec la mesure) ; si V2 est intégrée, empreintes du niveau `N` égales à celles du
      niveau 1. Un contrôle rouge : la PR lui revient avec la pièce ; nous ne corrigeons pas son code à sa
      place sans son accord.
- [ ] P.6 Réception des deux autres briques, dans l'ordre de P.1 : `go test -count=1 -v ./...` de la brique,
      zéro `SKIP` inattendu ; G-compat contre son dernier tag ; aucun fichier extrait du jeu ; licence de
      toute dépendance neuve relue ; `make go-api-test-gamefiles` une fois, par le pilote, seul sur la
      machine. Tags mineurs posés par le pilote. Ordre : lecteur de fichiers, puis brique des cartes (elle
      en dépend), puis montée des `require` du décodeur si un canal l'exige (suivie de G-clone sur `./...`
      et de G-equiv-ref sans écart), puis seulement le tag du décodeur. La montée de ces deux `require` dans
      LevelUp se fait sur la branche du plan 3D, pas sur celle de ce plan.
- [ ] P.7 Intégration de ses PR du décodeur dans `feat/v1.6.0`, une à une, sous le SEUL schéma 127 : le
      pilote fond les entrées de chronique, refige goldens, révisions (rang neuf, R-13), fixtures de contrat
      et références, tient le registre des changements de sortie (une ligne par canal, avec son crédit), et
      assemble ses fragments de documentation (R-15). G-façade après chaque intégration.
- [ ] P.8 Après la dernière intégration : G-corpus au niveau `N` ; son application compile et joue un match
      contre `feat/v1.6.0` (contrôle fait par lui) ; la session du rejeu 3D est prévenue des champs ajoutés
      (R-10) ; crédit du contributeur dans `NOTICE.md` et les notes de version.
- **Gate P** : chaque PR a passé P.5 ; les trois briques sont taguées dans l'ordre de P.1, sauf le décodeur
  (tag en L.3) ; aucun compte de tirs, de touches ni de dégâts n'a bougé ; le document n'a qu'UNE entrée de
  chronique 127 ; G-corpus sans perte non décidée par le user.

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

- 2026-11-23 — L'en-tête de `LU apps/go-api/cmd/killsource/main.go` (lignes 3 à 8) affirme qu'aucun chemin de l'application n'utilise le décodeur killsource : faux (appels de production dans `internal/replaybuild/kills.go:57` et `internal/sync/killcollector/collector_run.go:215`).
- 2026-11-23 — Un commentaire du test de surface de la façade, `LU apps/go-api/internal/archlint/film_facade_surface_test.go` (ligne 166), dit que `NomPrecisionParArmePasseSautee` et `NomDistancesDeToucheDesactivees` ont quitté la façade en v1.5.0 ; ils y restent, marqués `Deprecated` (`decfilm/replis_obsoletes.go`). Incohérence à trancher.
- 2026-11-23 — 113 parcours de `map` classés sans effet par un outil, non relus un par un (table sous `Scripts/dec-v160-recup/etape0/maprange_classes3.tsv`) : à relire avant l'intégration de la lecture de plusieurs chunks à la fois.
