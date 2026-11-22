# PLAN — Décodeur de film v1.6.0 : vitesse, backlog, canaux du lecteur 3D (2026-11-23)

Statut : **écrit le 2026-11-23 ; relu le même jour contre la grille `plan-review` par un second agent
(« à reprendre sur quelques points avant le go ») ; corrections B1 à B4 et I1 à I13 appliquées le 2026-11-23.
Commandé par le user le 2026-11-22 (« La v1.5.0 est publiée. Le user te confie la v1.6.0 du décodeur. Avant
tout code, écris un plan propre et fais-le relire contre la grille plan-review »). DD-2 TRANCHÉE par le user
le 2026-11-23. DD-1, DD-4, DD-5, DD-6, DD-7 et DD-8 À TRANCHER par le user. AUCUN GO D'EXÉCUTION. Le pilote
versionne ce fichier sur la branche de version avant le go.**
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
du rejeu, (3) traite chaque entrée du backlog qui concerne le décodeur, (4) reçoit les canaux du lecteur 3D
dès que le contributeur a donné son accord ; puis UNE montée dans LevelUp et UNE régénération du parc, avant
le premier démarrage de LevelUp 8.3.0.

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
| D | Les répliques des Spartans existent dans le rejeu (qui parle, quand, quelle réplique), selon DD-4 |
| E | Éliminations dont un bot est l'auteur ; temps et série en tant que VIP ; nom des options de variante |
| F | Rien (contrôles, ménage) |
| P | Rien dans la 2D ; le lecteur 3D reçoit ses canaux (pleine cadence, visée, arme en main, munitions, gestes, hauteur des projectiles) |
| M | Rien dans LevelUp tant que le plan 3D ne les consomme pas |

Version : v1.6.0 (ajouts seulement : version mineure). Schéma du document : 126 vers 127, UNE montée pour
toutes les pistes qui changent la sortie (A à F, P). Les pistes V ne touchent pas au schéma.

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
| Décompression : bibliothèque standard pour zlib (`film.go:217-246`), `klauspost/compress` pour zstd (`internal/source/zstd.go`). `decfilm.Inflate` est public (`decfilm.go:466`) : la décompression d'un chunk se chronomètre à part, pas à l'intérieur d'un chargement | idem |
| Aucun fil lancé en production, hors la sentinelle de mémoire | `DEC/memguard/memguard.go:175` (seul `go` hors tests) |
| La marche des trames porte UN état du monde d'un bout à l'autre du film : un chunk dépend de ceux qui le précèdent (table d'entités, liaisons des images-clés, table anticipée). Ce monde sert à LIRE les trames | `DEC/internal/grammar/marche_trames.go:109-150` (`NewWorld`, `parcourir`, `PoserChunkCourant`) |
| Une marche, N canaux : chaque canal déclare ses intérêts, la marche distribue. Le choix des canaux par l'appelant n'existe pas | `DEC/internal/grammar/distribuer.go:1-70` ; `DEC/replay/options.go:27` (`Options` sans sélection) |
| La marche commune de la source de dégât et de la cuisson est DÉJÀ en production | `DEC/internal/facts/killsource/decode.go:360` (`LireLaMarcheCommune`) ; `DEC/replay/build_from_film.go:98` (`PrendrePourLaCuisson`) |
| Trois entrées publiques de l'assemblage : film vers document, film vers document et faits, faits vers document. Un observateur date chaque étape (`Options.Observe`) : la dernière étape du balayage, `clockOrigin`, sépare le décodage de la publication | `DEC/replay/build_from_film.go:51,63`, `build_from_facts.go:52`, `options.go:461-464`, `observe.go:97` |
| Un instrument de durée existe, jetable, sous l'étiquette `research` : chargement, décodage, publication, pic de mémoire ; options MINIMALES (carte seule : ni source de dégât, ni tableau, ni objectifs, ni zones) | `DEC/research/duree/duree_research_test.go` |
| Six bancs `Benchmark*` dans quatre fichiers, tous de micro-lecture ; aucun de bout en bout | `DEC/internal/grammar/bench_*_test.go` |
| Le déterminisme de la marche est déjà testé (deux marches, même empreinte ; deux films en parallèle sous `-race`) | `DEC/internal/grammar/marche_determinisme_test.go` ; CI `film-race`, `.github/workflows/ci.yml:66-78` |
| La révision d'une couche est tenue par une empreinte des JETONS de ses sources (commentaires et mise en page écartés) : toucher `internal/source` ou `internal/grammar` fait rougir le test de révision, même sans changer la lecture. La v1.5.0 a déjà refigé une empreinte sans monter la valeur | `DEC/internal/grammar/rev_test.go:1-45` ; journal v1.5.0, seconde intégration |
| Les faits décodés se gardent sous `<racine>/data/cache/film_facts/<titre>/` et sont réemployés quand leurs révisions sont celles du code : deux branches aux mêmes révisions se servent le même cache (incident de la v1.5.0) | `LU apps/go-api/internal/domain/title/registry_film_facts.go:52-77` ; journal v1.5.0, 2026-11-22 |
| `replay-equiv` sans `-deux-passes` compare aux références du clone ; avec `-deux-passes`, il compare deux passes du MÊME commit (décodage, puis rejeu depuis les faits) et ne lit AUCUNE référence. Ni lui, ni `replay-build`, ni le gate du corpus n'ont de réglage du niveau de fils | `LU apps/go-api/cmd/replay-equiv/main.go:136-163`, `cmd/replay-build/main.go:59-72`, `cmd/replay-corpus-gate/main.go:117-132` |
| Le dépôt du décodeur n'a aucun dossier `cmd/` ; aucun catalogue de bornes de carte n'y a été trouvé | arborescence de `DEC` |
| Plafond de mémoire par film : 3 Gio en production, 2 Gio pour une mesure sur le poste, 4 Gio dans le gate | `DEC/memguard/memguard.go` ; `LU .../replaychild/replaychild.go:68-121` ; `cmd/replay-corpus-gate/bake.go:72` |
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
du décodeur. L'ordre des tags se lit dans l'inventaire (P.0, M.0).

Recouvrements déjà visibles avec la v1.5.0, à trancher canal par canal (item P.0) : `reloads[]` contre ses
gestes ; `moveCommand` contre son manche ; `coverage.seats.tirsParPlace` contre son tireur par place ;
`damage[]` contre ses dégâts par place ; `shots_visee.go` existe des deux côtés. Deux de ses apports
heurtent les règles du dépôt : le saut « tiré de la vitesse verticale » est une dérivation, pas une lecture
(règle R-2) ; la table des scores cibles par variante est une table par variante (R-2). Ses traductions de
messages ne remontent pas (la note du 2026-11-22 le dit déjà).

Ce qu'on attend de lui, précisément (transmis le 2026-11-22, sans réponse connue à ce jour) : (a) son accord
sur le découpage D-15 (paquet web, module Go des constructeurs) ; (b) son accord pour que ses changements
des trois briques remontent ; (c) qui ouvre les PR : lui, ou nous avec sa relecture.

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
| Répliques des Spartans | texte retrouvé dans le jeu le 2026-11-22 (191 sons sur 201) ; cadrage du user du même jour | Piste D, selon DD-4 |
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
| Idem : huit décisions devenues sans effet du manifeste du gate | avertissements du gate depuis le 2026-11-18 | F.2, selon DD-8 |
| Registre des reports : images-clés, trois lecteurs ; sonde de calibration | mesurés le 2026-11-13 | V3.2, V3.3 |
| Registre des reports : une seule marche pour la source de dégât et la cuisson | déjà en production (section 1.1) | `[~]`, registre à mettre à jour à la clôture |

## 2. Décisions

### Tranché par le user

- **DD-2 — chantier « précision » : HORS PÉRIMÈTRE de la v1.6.0.** User, 2026-11-23 : « En effet les items
  qui touchent à la précision sont hors périmètre ». Rien de la section « Chantier précision » du backlog
  n'entre ici, ni aucune autre entrée qui touche à la justesse des tirs, des touches ou de la précision par
  arme : lecture suffisante des tirs, tirs sans visée, projectiles au tir ambigu, tirs sans place, tirs
  embarqués sans trajet, copies des messages de tir, créateur des projectiles, BR75, table
  `match_weapon_hit_distance`. Garde mécanique pour la piste P : règle R-14.
- **Régénération du parc.** Go donné d'avance par le user le 2026-11-18 (« Pour les trucs plus tard je te
  donne mon go d'avance »), rappelé le 2026-11-22 : une seule fois, après la v1.6.0, avant le premier
  démarrage de LevelUp 8.3.0. Le user est prévenu AU LANCEMENT (serveur arrêté, environ 2 h 30, pas pendant
  qu'il joue). Ce n'est pas une décision à reposer.

### À trancher par le user AVANT l'exécution (recommandation en tête)

- **DD-1 — le point 1 (canaux du contributeur) si son accord tarde. À trancher MAINTENANT, pas au tag.**
  Recommandé : (a) les pistes V et A à F ne l'attendent pas ; quand elles sont intégrées et relues sans que
  l'accord soit arrivé, la v1.6.0 est taguée SANS le point 1, qui part en v1.7.0 ; la 3D restant dans la
  8.3.0 (ta décision du 2026-11-22), la régénération unique et la sortie de la 8.3.0 attendent alors la
  v1.7.0. D'ici l'accord, UN seul geste sur le point 1 : l'inventaire en lecture seule de ses copies (P.0,
  M.0), sans code ni PR. Autres voies : (b) même tag, mais la 8.3.0 sort sans la 3D et le parc se régénère
  après la v1.6.0 ; (c) attendre l'accord pour taguer la v1.6.0 ; (d) ne rien lire de ses copies avant
  l'accord.
- **DD-4 — répliques des Spartans : dans la v1.6.0 ou en v1.7.0.** Recommandé : dans la v1.6.0 pour
  l'événement (qui parle, quand, identifiant du son, genre : items D.1, D.3, D.4), le nom court seulement
  s'il est prouvé à la fin de la première vague de recherche, sinon il part en v1.7.0. Coût : environ un
  jour et demi de sessions, une ligne de plus au schéma 127, et côté LevelUp la table des textes et UNE
  maquette interactive des trois idées d'affichage (items D.5 et D.6, hors décodeur, faits seulement si tu
  retiens cette voie). Aucun code d'affichage dans ce plan. Autre voie : toute la piste D en v1.7.0, la
  v1.6.0 gagne ce jour et demi.
- **DD-5 — enchaînement avec la montée v1.5.0.** Recommandé : la montée v1.5.0 ne fusionne PAS dans
  `release/v8.3.0` avant la régénération : un parc resté au schéma 125 y rendrait les rejeux « à
  régénérer » et gênerait tes contrôles à l'écran des autres plans. La montée v1.6.0 part de la tête de la
  montée v1.5.0 (aujourd'hui `93a4744e7b` sur `chore/830-decodeur-v150-miroirs`, que le pilote avancera sur
  `chore/830-decodeur-v150`), reçoit `release/v8.3.0` par fusion régulière, et entre dans la release en une
  fois, juste avant la régénération. Autre voie : fusionner la v1.5.0 maintenant et régénérer deux fois
  (contraire à ta consigne du 2026-11-22).
- **DD-6 — copies des briques des cartes et du lecteur de fichiers.** Recommandé : elles remontent dans ce
  plan (piste M), dans leurs dépôts, avec leurs numéros. Le décodeur dépend de ces deux briques : la part
  qu'un canal du contributeur exige (ajouts de `mapvar`, d'après l'inventaire) passe AVANT ce canal et
  bloque le tag du décodeur si le canal est retenu ; le reste (export du monde pour la 3D) ne bloque que le
  plan 3D. Autre voie : un plan frère pour le reste.
- **DD-7 — l'axe « état du monde à la demande ».** Chez nous le monde sert à lire les trames : il ne peut pas
  attendre. Recommandé : l'axe livre ce qui peut réellement attendre ou ne pas être refait : une seule
  recherche pour les trois lecteurs d'images-clés (30 % du décodage seul, mesuré), la sonde de calibration
  si le profil la montre, et les états que V3.1 trouvera construits d'office pour des lecteurs qui ne les
  demandent pas ; la méthode littérale du parseur voisin (un état du monde à un instant, servi à la demande)
  est abandonnée avec ce motif écrit. Autres voies : abandonner l'axe entier ; ou ouvrir en v1.7.0 une
  entrée publique « état à l'instant » bâtie depuis les faits (une fonction neuve, pas un gain de vitesse).
- **DD-8 — huit décisions devenues sans effet dans le manifeste du gate du corpus** (datées du 2026-11-16,
  signalées par le gate depuis le 2026-11-18). Recommandé : les retirer (F.2) ; la liste t'est montrée avant.
  Autre voie : les garder.

### Tranché par le pilote (conventions, modifiables par le user)

- **DD-3 — parallélisme et mémoire.** La bibliothèque reste à UN fil par défaut. Le plafond de mémoire par
  film ne bouge pas (3 Gio en production). LevelUp garde un fil partout à la montée ; toute hausse du niveau
  de fils sur un profil (bureau, développement, serveur) attend une mesure du pic sur ce profil et se décide
  à part.
- Tags et fusions des dépôts de briques : par le pilote (délégation des 2026-11-11 et 2026-11-13) ; `main`
  de LevelUp reste un geste du user.
- Une piste de vitesse qui n'apporte aucun gain mesuré au-delà du bruit du relevé n'est pas fusionnée : son
  item passe `[!]` avec ses chiffres. Aucune complexité sans gain.
- Les pistes V n'ajoutent AUCUN champ au document ni aux faits (un compteur de relectures vit dans le relevé,
  jamais dans `coverage`).
- La commande du relevé vit dans `DEC/cmd/film-bench`, sans étiquette `research`.
- Les films du relevé portent dans le dépôt public un nom neutre (`f01` à `f13`) ; la correspondance avec
  les matchs vit dans `.ai/annexes_decodeur_v160/films_du_releve.tsv`.
- Méthodes du parseur voisin : reprises, jamais son code ni ses tables. Le nom des options de variante vient
  du jeu ou du film, jamais de sa table (licence en question).
- Ne remontent pas du contributeur : ses traductions de messages ; la table des scores cibles par variante
  (LevelUp l'injecte déjà) ; le saut tiré de la vitesse verticale tant qu'une lecture du film ne le remplace
  pas. Le codec `REPLAYINPUTS30` est statué à l'inventaire (P.0) : repris seulement si un canal retenu
  l'exige.

## 3. Règles d'exécution

Les règles R-1 à R-10 du plan v1.5.0 s'appliquent, avec ces précisions.

- **R-1 — pas de correction sans mesure.** Pistes A à F et P : mesure AVANT, vérité indépendante, mesure
  APRÈS, compte des cas qui changent. Une correction qui améliore un compte et en dégrade un autre s'arrête.
- **R-2 — la grammaire prime.** Aucune heuristique, aucun repli neuf (le gate du corpus juge FAUX tout repli
  parti de zéro), aucune table par film ou par carte, aucune liste de composants écrite à la main : une
  règle se lit dans ce que le film déclare (leçon de la v1.5.0 : la règle du drapeau écrite sur une liste de
  composants dégradait les zones ; lue dans la table des noms du mode, elle était juste).
- **R-3 — une recherche est bornée, et la version a une taille.** Une session d'agent ; sortie : cause
  prouvée et item de correction écrit ICI, daté, avant son code ; ou `[!]` avec ce qui a été écarté.
  Critère de taille : à la fin de la première vague de recherche (vague 1), tout item de recherche qui n'a
  pas À LA FOIS sa cause prouvée et son item de correction écrit passe `[!]`, avec report en v1.7.0 inscrit
  au registre des reports. Aucune seconde session.
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
  existant du document. La session du rejeu 3D est prévenue avant le tag de tout champ que la piste P ajoute.
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
  - Chaque intégration d'une piste qui change la grammaire ou les objectifs (A à F, P) pose un RANG de
    révision neuf pour la couche touchée. À défaut, le cache de faits (`<racine>/data/cache/film_facts/`)
    est purgé avant toute passe.
  - Le gate de toute piste inclut les tests de révision (commande G-rev, plus bas).
  - Un cache de faits et un espace de travail PAR PISTE, de noms distincts, jamais partagés : racine de
    données `Scripts/dec160-racines/<piste>/` (copie des films du corpus, sans jonction) et fichier
    `Scripts/dec160-racines/<piste>.work`. Quatre racines au plus à la fois, supprimées à la clôture de la
    piste. L'item 0.6 vérifie que `replay-equiv -repo-root` accepte une racine de données distincte du
    dossier des sources ; sinon la racine est un worktree de LevelUp par piste.
- **R-14 — garde mécanique de la précision (piste P).** Pour tout canal du contributeur qui touche les tirs,
  les touches ou les dégâts, les empreintes des étapes de tirs, de touches, de dégâts et de statistiques de
  tir de `replay-equiv -out-dir` sont IDENTIQUES avant et après sur les 20 films (les noms exacts de ces
  étapes sont relevés en 0.6). Une empreinte qui bouge : le canal sort de la version.
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
  l'item annonce (un `-run` sans correspondance répond `ok` : ce n'est pas un vert).
- **G-rev** : marche de publication, geste 3, en mode vérification :
  `GOWORK=off go test -count=1 -v ./internal/profile/ ./internal/source/ ./internal/grammar/
  ./internal/facts/killsource/ ./internal/facts/objectives/ -run 'RevSuit|ChroniqueDe|ChroniqueCouvre|ChroniqueDObjectives'`
  (10 tests `PASS` attendus) et `go test -count=1 -v ./revision/ ./types/`.
- **G-compat** (surface publique, à chaque gate de piste) : `gorelease -base=v1.5.0` lancé par
  `go run golang.org/x/exp/cmd/gorelease@<version épinglée en 0.6>` depuis le worktree ; verdict attendu :
  aucun changement incompatible. Si l'outil ne tourne pas sur le poste (à établir en 0.6), repli écrit :
  `go doc -all` de chacun des 19 paquets publics, comparé à la capture faite sur `v1.5.0` ; toute ligne
  retirée ou changée est un échec, une ligne ajoutée est admise.
- **G-façade** (à chaque intégration, pas seulement à la montée) : `cd LU/apps/go-api && go test -count=1 -v
  ./internal/archlint/ -run 'TestSurfaceDeLaFacadeDuDecodeur|TestSurfaceDeReplayCiteeHorsDuDecodeur'`, la
  branche de montée désignant la pseudo-version intégrée ; un plafond dépassé se relève là, daté.
- **G-equiv-ref** (contre des références) : `cd LU/apps/go-api && GOWORK=<espace de la piste> go run
  ./cmd/replay-equiv -repo-root <RACINE> -originaux <HOTE>/data/film_identites/xuids_originaux.txt -fils <n>` :
  `20 identique(s), 0 different(s)`. Compare aux références du clone.
- **G-equiv-2p** (double passe, même commit) : la même commande avec `-deux-passes` : artefact identique à
  l'octet entre le décodage et le rejeu depuis les faits, aucun écart à classer. Ne lit aucune référence :
  elle ne prouve rien contre la v1.5.0.
- **G-corpus** : marche de publication, geste 9, avec `--fils <N>` : 19 témoins, ni gain, ni perte, aucune
  décision ajoutée au manifeste. La base se cuit comme elle se cuisait ; la tête au niveau `N`.

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
| D répliques (DD-4) | 6 | Sonnet | lourd | moyen |
| E canaux neufs du backlog | 6 | Sonnet ; Haiku pour E.5 | moyen | moyen |
| F contrôles et petites entrées | 7 | Haiku (F.1, F.3, F.7) ; Sonnet | rapide | faible |
| P canaux du contributeur | 9 | Sonnet | lourd | élevé |
| M briques des cartes et des fichiers | 4 | Sonnet | lourd | moyen |
| R relecture | 2 | Fable ; une ronde de corrections Sonnet | moyen | aucun |
| L publication, montée, régénération | 9 | pilote ; Sonnet pour les miroirs | lourd | élevé |

Estimation : environ 20 agents Sonnet, 3 à 4 Opus, 4 Haiku, 1 Fable. Passes du pilote, recomptées :
- preuve d'identité d'une piste V : 9 passes d'équivalence (6 de référence, 1 double, 3 répétées moins celle
  déjà comptée) soit environ 1 h, plus deux relevés A d'environ 25 minutes : 1 h 50 par piste, 7 h 20 pour
  quatre ;
- étape 0 : marche à blanc (35 minutes dont le gate), trois doubles passes (40 minutes), relevés de
  référence (1 h) : 2 h 15 ;
- intégration « vitesse » : preuve complète et gate du corpus, 2 h 15 ;
- pistes A à F : une passe et deux reprises d'attribution par piste, 20 minutes chacune, 2 h ; suite
  complète du clone à chaque intégration (5 minutes, une douzaine de fois), 1 h ;
- piste P : huit intégrations à 25 minutes, 3 h 20 ;
- fin : équivalence refigée et double passe (30 minutes), gate du corpus compté TROIS fois (la v1.5.0 en a
  demandé quatre), 1 h 15, relevés finals (1 h), gates de LevelUp et filet de poussée (1 h 30) : 4 h 15 ;
- régénération : 2 h 30, dérivés 25 minutes.
Total : environ 22 heures de machine hors régénération sans la piste P, dont 8 heures de relevés en fenêtre
de mesure, pendant lesquelles aucun agent ne tourne. Durée : 7 à 9 jours de sessions sans le point 1 ;
3 à 4 jours de plus pour P et M ; un jour et demi de moins si la piste D part en v1.7.0 (DD-4). Le coût est
annoncé au user avant chaque vague.

### Vagues et parallélisme

Chemin critique : 0, V0 (Gate V0), V1 et V4, intégration « vitesse », première intégration au schéma 127.
Borne : 4 agents d'écriture à la fois, toutes sessions confondues ; les recherches en lecture seule s'y
ajoutent, un film décodé à la fois ; tout s'arrête pendant une fenêtre de mesure.

- **Étape 0**, puis **V0**. Les pistes de vitesse démarrent après le Gate 0 ET le Gate V0.
- **Vague 1.** V1 et V4 ensemble (fichiers disjoints, à confirmer en 0.6 ; sinon l'une après l'autre) ; V3
  après V4 (toutes deux touchent le balayage du film dans `replay/film_scan*.go`). En même temps, sans code
  de production : les recherches A.1, A.3, A.4, B.1, C.1, C.2, D.2, E.4, F.5, F.6, la conception V2.1, et
  la piste F. Fin de vague : critère de taille de R-3.
- **Intégration « vitesse »** (pilote) : V1, V4, V3 ; preuve d'identité contre la v1.5.0.
- **Vague 2.** A, B, C, D (selon DD-4), E ensemble, sous réserve de la table ci-dessous ; intégrées une à
  une, schéma 127. V2 se code pendant cette vague dans son worktree et ne retarde aucune de ces pistes.
- **Intégration de V2**, APRÈS les pistes du backlog : rebasée sur la tête, prouvée par « niveau 1 = niveau
  N au même commit ». Si V2.1 conclut à l'arrêt ou si sa preuve échoue, la version part sans elle.
- **Vague 3 (après l'accord du contributeur).** P.0 et M.0 peuvent précéder (DD-1) ; P après C, D, E et V2 ;
  M selon l'inventaire.
- **R** (relecture, corrections), puis **L** (équivalence et gate finals, tag, montée, régénération).

Fichiers attendus par piste (à confirmer en 0.6 ; deux pistes sur un même fichier = l'une après l'autre,
écrit au journal) :

| Piste | Écrit dans `DEC` | Ne touche pas |
|---|---|---|
| V0 | `cmd/film-bench/` (neuf), `bench/` (neuf) ; V0.6 : la structure d'options ajoutée (`replay/options.go`, `decfilm/`) | tout le reste |
| V1 | `internal/source/film.go`, `source.go`, `zstd.go`, `filmcache/` (lecture des chunks), `decfilm/` (une entrée de chargement ajoutée, `LoadDir` inchangée) | `internal/grammar/`, `replay/` |
| V4 | `replay/film_scan*.go`, `build_from_film.go`, `internal/grammar/movement_states.go` et les appelants de `Distribuer` (inscription des canaux) ; lit `layers.go` | `internal/source/`, `distribuer*.go` (le distributeur lui-même), `marche_*.go` |
| V3 | `internal/grammar/marche_images_cles*.go`, `keyframe_etat_complet_*.go`, `keyframe_loadout.go`, `replay/film_scan_images_cles.go` (après V4) | `marche_trames*.go`, `keyframe_liaison.go`, `internal/source/` |
| V2 | `internal/grammar/marche_trames*.go`, `marche_commune.go`, `marche_de_killsource.go`, `distribuer*.go`, `world.go`, `keyframe_liaison.go`, `keyframe_anticipe.go` | `internal/source/`, `replay/` |
| A | `reponsesdejeu/`, `replay/document_score_actions*.go`, `score_timeline*.go`, `internal/facts/objectives/score.go` | drapeau, zones |
| B | `replay/flag_*.go`, `porteurs_*.go`, `zone_states_*.go`, `internal/grammar/components_navpoint*.go` | score |
| C | `internal/grammar/frame_vue_controle.go`, `replay/document_stances*.go` | vue A |
| D | `internal/grammar/vue_a_genres.go`, `vue_a_lecture.go`, fichiers neufs `replay/document_repliques*.go` | vue de contrôle |
| E | `replay/document_eliminations.go`, `eliminations.go`, `internal/grammar/film_variante_de_partie.go`, `internal/facts/objectives/named*.go` (E.2, après A) | drapeau |
| P | à écrire en P.0 ; touche `vue_a_lecture.go`, `distribuer_tetes.go`, `movement_states.go`, `replay/shots*.go`, `projectiles.go`, `filmcache/` | — |

Dépendances entre pistes de vitesse, dites : V1, V2 et V4 lisent toutes la structure d'options de V0.6 et
aucune ne l'écrit ; V2 et V4 touchent toutes deux la distribution aux canaux (V4 les inscriptions, V2 le
distributeur) : V2 passe après V4 ; V3 passe après V4. Seules V1 et V4 tournent ensemble.

Fichiers à écrivain unique, le pilote, à l'intégration seulement : `replay/document_schema.go`,
`document_chronicle.go`, `layers.go`, `coverage.go`, `filmfacts*.go`, tous les goldens, les fichiers de
révision de couche, les références d'équivalence, les fixtures de contrat, et les fichiers communs de R-15.
Aucun agent ne monte le schéma ni ne régénère un golden : il rend la liste exacte de ce qu'il fait rougir.

## Étape 0 — témoins, références, instruments (pilote ; un agent Sonnet)

- [ ] 0.1 Clone neuf `Scripts/dec-v160` à la tête de `main` ; vérifier `main` = `v1.5.0` (`dc967c2`) ;
      branche `feat/v1.6.0`, poussée. Worktree `Scripts/LevelUp-wt-dec160` sur `chore/830-decodeur-v160`,
      créée de la tête de la montée v1.5.0 (DD-5), sans jonction ; `npm ci` et espace de travail engendré
      (R-11) ; racines de données par piste préparées à la demande (R-13).
- [ ] 0.2 Marche de publication rejouée à blanc sur le clone SANS changement : G-clone sur `./...` (33
      paquets), G-rev, G-equiv-ref à 20 identiques. Le gate du corpus est lancé une fois, comme TEST
      D'INFRASTRUCTURE seulement (parc de travail, verrou, manifeste lus) : sans changement de `go.mod` il
      compare la v1.5.0 à elle-même et ne prouve rien sur les sorties.
- [ ] 0.3 Jeu de films fixe du relevé arrêté (V0.1) : présence de chaque film dans `HOTE` vérifiée ; le film
      de plus fort pic de mémoire du parc désigné par mesure ; correspondance écrite dans
      `.ai/annexes_decodeur_v160/films_du_releve.tsv`.
- [ ] 0.4 Bruit de base du déterminisme, AVANT toute piste V : G-equiv-2p sur les 20 films, trois fois.
      L'écart « à classer : kill » est instruit jusqu'à sa cause ; `joinSegments` et tout parcours de `map`
      qui ordonne une sortie sont relevés, fichier et ligne. Sortie : zéro écart à classer en double passe
      sur la v1.5.0, ou un item de correction écrit ici (il changerait la sortie : il rejoint alors la
      vague 2, et la phase vitesse se prouve avec ce bruit nommé et borné à son étape).
- [ ] 0.5 Contrôles sur pièces : le compteur des émissions jetées est juste hors Drapeau au tag
      (`e703301`) ; chaque référence `fichier:ligne` de la section 1.1 rouverte sur la tête du clone.
- [ ] 0.6 Table des fichiers par piste confirmée sur la tête du clone, conflits écrits au journal ; noms des
      étapes de tirs, de touches et de dégâts de `replay-equiv` relevés (R-14) ; outil de G-compat essayé et
      sa version épinglée, ou repli retenu et capture `go doc -all` de `v1.5.0` rangée hors dépôt ;
      `replay-equiv -repo-root` essayé sur une racine de données distincte (R-13).
- [ ] 0.7 Liste fermée des appelants de LevelUp qui ne lisent qu'une partie du film (candidats : rattrapage
      de la source de dégât, coup d'envoi du film, rasters tactiques), avec fichier et ligne : elle fixe ce
      que V4 doit servir et ce que L.4 branche. Écrite ici avant la vague 1.
- [ ] 0.8 Vérité de l'API et faits des matchs : export unique en lecture sur une COPIE de la base partagée,
      hors dépôt, pour les pistes A, B, E et pour le relevé B (`levelup replay-facts-export` sur les 13
      films). Mesures AVANT par piste dans `.ai/annexes_decodeur_v160/avant.md`.
- **Gate 0** : 0.2 vert (G-clone, G-rev, 20 identiques) ; 0.4 statué ; 0.6 et 0.7 écrits ; `avant.md`
  complet. Aucun relevé de vitesse n'est exigé ici.

## Piste V0 — relevé de vitesse par version, socle des options (Sonnet)

Deux relevés, parce que la génération d'un rejeu dépend de la base de LevelUp, absente du dépôt public.

- [ ] V0.1 Jeu de films fixe, 13 films, critères écrits dans `bench/README.md` : les dix témoins de durée du
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

## Piste V3 — lectures refaites et état à la demande, selon DD-7 (Sonnet)

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

- [ ] V2.1 Instruction sans code, dans `.ai/annexes_decodeur_v160/v2_conception.md` (vague 1) : ce qu'un
      chunk hérite du précédent dans notre marche (table d'entités du monde, liaisons des images-clés, table
      anticipée, neufs suspendus, profil de balayage, compteurs de replis, crochets des canaux, marche
      commune de la source de dégât) ; à quelle condition un chunk peut partir d'une copie de l'état ;
      comment on RECONNAÎT qu'il dépendait du précédent ; méthode du parseur voisin relue
      (`parser/v41/chunks.rs`) ; part de la marche des trames dans le profil V0.5. Si cette part est sous
      20 % du total, ou si la dépendance ne se reconnaît pas par un fait écrit, la piste s'arrête : `[!]`
      avec la mesure, report en v1.7.0 (R-3).
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
      table « son champ, notre champ, lequel reste », remise à P.0.
- **Gate C** : équivalence attribuée ; poids ajouté au document sous 1 % sur 20 films ; G-clone sur
  `./replay/ ./internal/grammar/` ; G-rev ; G-compat.

## Piste D — répliques des Spartans (Sonnet ; selon DD-4)

- [ ] D.1 Lecture par la grammaire des genres 75 et 76 de la vue A (locuteur, identifiant du son, priorité,
      indice de réplique ; annonceur : identifiant du son, destinataires), aujourd'hui traversés sans valeur.
      Contrôle : mêmes comptes que le témoin sur 3 films.
- [ ] D.2 Nom court de la réplique, DANS le décodeur (cadrage du user) : recherche bornée de la
      correspondance entre l'indice du film et l'entrée des tables de vocalisation (5 noms résolus à ce
      jour). Publié quand il est prouvé à la fin de la vague 1 ; sinon `[!]`, report en v1.7.0 (R-3).
      Aucun nom inventé.
- [ ] D.3 Publication : calque `chatter[]` (image, joueur, son, genre, nom court s'il existe) et sa
      couverture (lus, publiés, sans joueur, hors du match) ; les annonces de l'annonceur dans un calque à
      part si D.1 les lit. Le film ne dit pas quelle variante audio a joué : le document ne le dit pas non
      plus.
- [ ] D.4 Fragments de documentation (R-15) : `FORMAT.md`, `GLOSSARY.md` ; crédit à Den (méthode et table de
      textes, libfilmshell, MIT) pour `NOTICE.md`.
- [ ] D.5 HORS DÉCODEUR, seulement si DD-4 retient la v1.6.0 : côté LevelUp, à la montée, commande qui bâtit
      la table des répliques depuis le jeu installé (identifiant du son vers textes français et anglais et
      fichiers de voix), faite comme celle des sons d'armes (`cmd/weapon-sounds`), derrière le tag
      `gamefiles` ; aucun fichier du jeu au dépôt.
- [ ] D.6 HORS DÉCODEUR, même condition : UNE maquette interactive des trois idées du user (icône de son et
      texte sur le joueur ; icône et son joué ; une variante au hasard dans le fil avec un filtre
      « Répliques »), sur un rejeu réel, inscrite à l'artefact des vérifications. Les zones de lieu de
      LevelUp servent telles quelles. Aucun code d'affichage dans ce plan.
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
      que les gestes du contributeur couvrent (coup) ; sort écrit, remis à P.0.
- [ ] E.6 Victime d'une élimination appariée au plus proche : victime absente et compteur d'ambiguïté quand
      une action a plusieurs candidats.
- **Gate E** : équivalence attribuée ; éliminations et statistiques VIP contre l'API sans baisse ; G-clone
  sur `./replay/ ./internal/grammar/ ./internal/facts/objectives/` ; G-rev ; G-compat.

## Piste F — contrôles et petites entrées (Haiku ; Sonnet)

- [ ] F.1 `scripts/briques-workspace.sh` (LevelUp) : sous Windows, l'entrée `use` du clone s'écrit avec des
      barres inverses ; témoin : `go test` d'un sous-paquet du clone passe avec le `GOWORK` engendré.
- [ ] F.2 Selon DD-8 : les huit décisions devenues sans effet de `config/replay_corpus.toml` (2026-11-16)
      retirées, liste montrée au user avant.
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

## Piste P — canaux du contributeur (Sonnet ; ATTEND L'ACCORD, sauf P.0)

- [ ] P.0 Inventaire en lecture seule (DD-1), sur la tête de son dépôt au jour de l'item : diff de
      `third_party/film-decoder` contre v1.3.6, puis contre v1.5.0 ; pour chaque canal : fichiers, schéma
      qu'il prend chez lui, recouvrement avec la v1.5.0 (tables de C.4 et E.5), règle du dépôt en jeu, ajout
      du lecteur de fichiers ou de la brique des cartes dont il dépend (ordre des tags, DD-6), sort du codec
      `REPLAYINPUTS30`. Rendu : `.ai/annexes_decodeur_v160/inventaire_contributeur.md`. Les items P.1 à P.8
      sont réécrits ici, datés, d'après cet inventaire, AVANT tout code. Les numéros de schéma sont
      réattribués : un seul 127 pour toute la version.
- [ ] P.1 Pistes à pleine cadence et visées seules.
- [ ] P.2 Tirs : direction par la visée, décalage dans l'image, tireur par place de contrôle : champs
      ajoutés seulement, sous la garde R-14.
- [ ] P.3 Arme en main par place ; munitions portées.
- [ ] P.4 Dégâts par place et causes des morts, confrontés à `damage[]` et à la source de dégât, sous la
      garde R-14.
- [ ] P.5 Gestes (lancer, coup ; le rechargement existe) ; manche et postures de contrôle selon C.4.
- [ ] P.6 Projectiles : hauteur, pleine cadence, projectiles collés, détonation d'un lancer sans vol publié.
- [ ] P.7 Socles : socles d'équipement, candidats de socle, axes d'un socle (après le tag du lecteur de
      fichiers, M.1) ; seconde famille du Stalker Rifle.
- [ ] P.8 Façades : `filmcache.ParseManifest`, `filmcache.NewSource`, `decfilm.LoadMapQuantCatalogBytes`.
- Règles de la piste : une PR par canal ou petit groupe, chacune avec sa ligne au registre des changements de
  sortie ; son crédit rendu en fragment pour `NOTICE.md` et les notes de version ; un champ que LevelUp ne
  lit pas est légitime ; ses fixtures passent le contrôle des identifiants ; aucun repli sans nom ni
  compteur.
- **Gate P** (par canal) : équivalence attribuée, plus G-equiv-2p ; R-14 pour P.2 et P.4 ; si V2 est
  intégrée, empreintes du niveau `N` égales à celles du niveau 1 ; relevé A à un fil : total dans les 10 %
  du relevé « vitesse », sinon retour au pilote ; poids du document mesuré et présenté ; G-clone, G-rev,
  G-compat, G-façade ; son application compile contre la branche (contrôle fait par lui ou avec lui).

## Piste M — briques des cartes et du lecteur de fichiers (Sonnet ; ATTEND L'ACCORD, sauf M.0)

- [ ] M.0 Inventaire en lecture seule : diff à trois voies de ses copies (parties de v1.0.1) contre nos
      v1.4.0 et v1.2.0 ; ce qui est déjà chez nous ; ce qui est neuf ; ce que les canaux de P exigent (à
      taguer AVANT eux) et ce que seul le plan 3D attend ; fichiers du jeu omis ; dépendance neuve
      (`woozymasta/bcn`, licence à relire). Rendu : `inventaire_briques.md`. Items M.1 à M.3 réécrits
      d'après lui, avec l'ordre des tags.
- [ ] M.1 Lecteur de fichiers : d'abord ce que P exige (ajouts de `mapvar`), PR, CI, tag mineur, puis montée
      du `require` du décodeur ; ensuite compagnons `.module_hd1` et lecteurs du maillage.
- [ ] M.2 Cartes : `bitmapdecode`, export du monde et bibliothèques, par PR de taille relisible ; tests du
      jeu derrière `gamefiles` ; PR, CI, tag mineur.
- [ ] M.3 Montée des deux `require` dans LevelUp, sur la branche du plan 3D (pas sur celle de ce plan).
- **Gate M** : `go test -count=1 -v ./...` de chaque brique, zéro `SKIP` inattendu ; G-compat de chaque
  brique contre son dernier tag ; `make go-api-test-gamefiles` une fois, par le pilote, seul sur la machine ;
  aucun fichier extrait du jeu au dépôt ; après une montée du `require` du décodeur : G-clone sur `./...` et
  G-equiv-ref sans écart.

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
      fils laissé à 1 partout (DD-3) ; appelants de 0.7 branchés sur le choix des canaux, CHACUN avec ses
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
      (`levelup backfill-kickoff`). Selon DD-1, cette étape attend la v1.7.0 si le point 1 y est parti.
- [ ] L.7 Mesures APRÈS sur le parc régénéré, comparées à `avant.md` de ce plan ET aux mesures reportées du
      plan v1.5.0 (A.3, C.4, E.4, L.7) ; durée de la régénération comparée au précédent (1 455 films en
      2 h 26) ; vérifications à l'écran inscrites à l'artefact des vérifications.
- [ ] L.8 Gestes de production inscrits à la liste de la release : dossier d'attente, bascule ; niveau de
      fils du serveur à 1.
- [ ] L.9 Clôture : tous les items statués ; recherches closes ou `[!]` avec leurs pièces et leur report en
      v1.7.0 ; backlog mis à jour entrée par entrée (section 1.4) ; registre des reports (dont le report de
      la marche commune, déjà fait) ; items L.6, L.7 et R.3 du plan v1.5.0 statués `[~]` vers ce plan ;
      entrée de `.ai/thought_log.md` ; worktrees retirés un à un après recherche de jonction, dans un tour à
      part ; racines de données et caches dédiés supprimés.
- **Gate L** : chaque ligne du registre des changements de sortie retrouvée dans les mesures APRÈS ; CI de
  LevelUp verte sur `release/v8.3.0`.

## Journal d'exécution

- 2026-11-23 — plan écrit, puis relu par un second agent contre la grille `plan-review` ; corrections
  appliquées le même jour : preuve d'identité en commandes distinctes et niveau de fils réglable dans les
  outils de LevelUp ; règle des révisions de couche et des caches par piste ; deux relevés de vitesse ;
  relecture avant les gates finals ; V2 après les pistes du backlog ; DD-2 tranchée par le user ; DD-3
  tranchée par le pilote ; DD-7 et DD-8 ajoutées. Aucun code, aucun commit.

## Découvertes (consignées, non traitées)

(vide)
