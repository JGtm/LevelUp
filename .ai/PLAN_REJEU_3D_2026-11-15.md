# PLAN — Rejeu en 3D dans l'application de bureau (2026-11-15)

Statut : **rédigé le 2026-11-15 ; D-1 à D-9 et D-11 tranchés par le user le 2026-11-15 ;
D-10 (HUD) se tranche sur maquette ; D-12 retenu sous réserve du découpage du contributeur (0.2) ; D-13 (repère 3D des listes) attend sa validation ; le source du contributeur est à recevoir ; puis le go.
Exécution en pistes parallèles (section 1 bis) : Sonnet par défaut, Haiku, Opus si nécessaire, Fable pour l'audit 0.8 et la revue finale R.1.**
Contrat d'exécution : skill `plan-execution` (ordre strict dans chaque piste, un item clos et vérifié avant le suivant,
aucune case vide à la clôture, zéro fix hors périmètre, découvertes en section 8). Grille relue : skill
`plan-review`.

**Amendé le 2026-11-17** par le plan maître `.ai/PLAN_MAITRE_8_3_0_2026-11-17.md`, qui fixe l'ordre entre
plans, les fichiers partagés et la borne machine. Réponse du user du 2026-11-17 (plan maître, DM-2 : « Tout ce
que le plan permet ») : les items ouverts « dès le go, sans le source » démarrent. Restent attendus du user :
le source du contributeur, la validation de D-13, puis le go du reste du plan. Règles du plan maître qui
s'appliquent ici :
- rien de ce plan n'entre dans `release/v8.3.0` avant la Gate 0 (source reçu et mesuré), sauf D.2 ;
- D.2 part sur sa branche `refactor/rejeu-moteur-2d` et fusionne dans `release/v8.3.0` dès son gate, avant
  l'item E.1 de la refonte ;
- les items web qui se posent dans les cadres (C.5, C.8, D.3, E.5) attendent la fusion de la refonte ;
- A.4 attend l'intégration de la piste G du plan backlog (mêmes fichiers de capacités) ;
- un écart qui demanderait de changer le décodeur ou le document de rejeu (mesure M-4 comprise) revient au
  user avant tout code : une seule montée du décodeur et une seule régénération par version ;
- worktrees `C:/t/3d-<piste>`, sans aucune jonction ; borne de 4 agents d'écriture commune aux quatre plans.
Faits mis à jour le 2026-11-17 : `release/v8.3.0` porte le décodeur v1.4.1 et la brique des cartes v1.3.1 ;
`ui/ReplayCanvas.tsx` fait 810 lignes ; l'item 0.1 est `[~]` (le pilote de la 8.3.0 est la session du plan
maître).

Constats relus sur pièces le 2026-11-15 sur `origin/release/v8.2.0` (`0ccbed4e2c`) et sur le binaire
`infinite-theater-3d-0.1.0-test.8` du contributeur llampwall (lu, non lancé). L'exécutant les RE-VÉRIFIE
sur la tête de `release/v8.3.0` avant de coder (base déplacée le 2026-11-15, voir D-2).

---

## 0. Objectif et critère de succès

Demande du user (2026-11-15) :
- un rejeu en 3D, réservé à l'application de bureau ;
- au premier lancement de la version qui l'apporte, ou sur une installation neuve, l'app demande, avec
  l'accord du user, où est installé Halo Infinite (« Steam », « Xbox » ou « Parcourir ») ;
- un commutateur 2D / 3D dans le rejeu ;
- en 3D, une mini-carte 2D qui affiche la **vue Plan** (fonds au maillage du 13-14/11) et les autres joueurs
  comme la vue 2D, sur un périmètre plus petit.

Niveau visé (user, 2026-11-15) : **la qualité du lecteur du contributeur**. Cela couvre la carte texturée et
éclairée, et les Spartans avec armures, peintures, corps animé, armes en main et bras en 1re personne. Une
version plus légère a été envisagée (D-3).

| Lot | Ce que le user verra | Critère de succès |
|---|---|---|
| 0 | Rien à l'écran | D-1 exécutée (source du contributeur obtenu, ou repli écrit), mesures M-1 à M-4 écrites en annexe |
| A | Question « Où est installé Halo Infinite ? » à l'installation neuve, une fois après la mise à jour, et à l'activation de « Rejeu 3D » dans l'onglet Jeux | Aucun disque lu avant la réponse ; Steam et Xbox trouvés sur le poste du user ; « Parcourir » refuse un dossier qui n'est pas un `deploy` ; « Pas maintenant » laisse le réglage désactivé et activable plus tard dans Jeux |
| B | Progression « Préparation de la 3D » ; dans Jeux : place occupée, plafond, dossier, « Libérer l'espace » | Carte d'un match, armes, pièces d'armure, peintures et bras construits depuis le jeu installé, dans le dossier des fichiers 3D seulement ; reprise après redémarrage ; reconstruction après une mise à jour du jeu ; plafond et garde d'espace libre tenus |
| C | Commutateur 2D / 3D dans le menu « Vue » ; vue 3D au niveau du contributeur ; caméra libre, 3e personne, 1re personne | Mêmes horloge, joueur suivi, Seul / Masquer que la 2D ; 60 i/s visés sur le poste du user (M-3) ; aucun bouton 3D hors bureau ni sans jeu trouvé |
| D | Mini-carte en vue 3D : vue Plan, joueurs, périmètre resserré autour du joueur suivi | Maquette validée AVANT code ; mêmes marqueurs et règles Seul / Masquer que la 2D ; repli sur le Rendu pour une carte sans Plan |
| E | Repère « rejeu 3D » sur les tuiles, tableaux et page match ; filtre « Avec rejeu 3D » dans l'Explorer | Maquette validée AVANT code ; repère seulement quand le rejeu 2D existe et que la carte est prise en charge ; rien de changé hors bureau ou 3D inactive ; le lien ouvre le rejeu en 3D |

Hors périmètre de ce plan : sons extraits de l'installation (annonces, rechargements), export vidéo de la 3D.

## 1. Décisions

- **D-1 — partir du travail du contributeur.** **Tranché (user, 2026-11-15) : « il m'a donné son code exprès
  pour que je travaille l'intégration, j'ai sa validation ».** Le source reste à recevoir : les paquets
  test.8 et test.9 ne contiennent que l'exécutable (item 0.2).
  - Constat : son application lit déjà la géométrie, les lumières, les textures, les animations, les armures
    et les peintures depuis le jeu. Sa copie de `halo-infinite-map-images` (v1.0.1) porte des lecteurs que la
    nôtre (v1.3.0) n'a pas, et il a porté en Go une partie d'Ekur (GPL-3.0-or-later). Son code source n'est
    pas public (`llampwall/infinite-theater-3d` introuvable). L'AGPL donne droit au source à qui reçoit le
    binaire.
  - Viser sa qualité (D-3, D-4) revient en pratique à reprendre son code.
  - Proposition : le user lui demande le source et lui propose de contribuer. La lecture du jeu passe dans
    le dépôt `halo-infinite-map-images` ; le lecteur 3D s'adapte au document de rejeu de LevelUp. Ses crédits
    sont conservés, comme l'exige sa clause 7(b). Les mentions de copyright d'Ekur restent dans les fichiers
    qui le portent.
  - Si le source n'arrive pas, la qualité visée n'est plus tenable en 8.3.0, et le
    plan revient au user avant toute exécution.
- **D-2 — base.** **Tranché (user, 2026-11-15) : `release/v8.3.0`.** Le user visait d'abord la 8.2.0. Il a
  ensuite sorti la 3D et les temps forts de la 8.2.0 pour la 8.3.0 (décision relayée par le pilote
  levelup-dd le 2026-11-15). `release/v8.3.0` a été créée depuis `origin/main` (`5ecbcfe934`).
  - Règles de la release, celles du pilote de la 8.3.0, à relire au go : pas de PR ; chaque lot se pousse
    sur `release/v8.3.0` après ses portes, rebasé juste avant l'envoi ; SHA transmis au pilote.
  - Sous-branche `feat/rejeu-3d`, un worktree par agent (`LevelUp-wt-3d-<lot>`, SANS jonction vers `data/`).
  - Conséquence : la sortie de la 8.3.0 attend la clôture de ce plan. À annoncer au pilote avant le lot A.
- **D-3 — qualité de la carte.** **Tranché (user, 2026-11-15) : qualité du contributeur, « pas de version
  légère ».** Réglages de rendu à la volée dans le menu « Vue » (ombres, définition, distance d'affichage),
  sans reconstruction ni place en plus. La place sur le disque relève de D-11.
- **D-11 — stratégie anti-saturation du disque.** **Tranché (user, 2026-11-15, « ouais ok ») : proposition
  retenue, à affiner par l'audit 0.8.**
  - Constat : la place se compte **par carte, pas par match**. Une carte construite sert à tous les matchs
    joués dessus ; un match n'ajoute qu'un petit manifeste (quelques Ko). S'y ajoutent les bibliothèques
    communes (armes, armures, bras, peintures), construites une seule fois : rien que l'export des peintures
    fait 109 Mo en 52 000 fichiers, d'après le contributeur. Le reste est mesuré en M-1. Ordre de grandeur :
    20 à 40 cartes jouées font 12 à 36 Go à 600-900 Mo pièce. Le jeu compte environ 114 cartes, Forge
    comprises.
  - **Construction à la demande seulement.** Une carte ne se construit qu'à l'ouverture d'un rejeu en 3D sur
    cette carte. Jamais de construction de toutes les cartes des matchs en base au démarrage, contrairement
    au lecteur du contributeur.
  - **Plafond du cache 3D**, réglable dans Jeux, 20 Go par défaut. Au-delà, les cartes ouvertes il y a le
    plus longtemps sont supprimées. La carte du rejeu ouvert et les bibliothèques communes ne le sont jamais.
  - **Garde d'espace libre.** Avant chaque construction, le serveur vérifie qu'il restera au moins 10 Go
    libres sur le disque après la construction estimée. Sinon, rien n'est construit. Le message dit la place
    manquante et propose « Libérer l'espace » ou de changer de dossier ; la 2D reste disponible.
  - **Dossier des fichiers 3D** au choix, sur le modèle de l'archive des films (`FolderPickButton`, sur un
    autre disque par exemple). Le changement déplace les fichiers, comme `film archive-move`.
  - **Textures communes rangées une seule fois.** Si M-1 montre que des cartes partagent des textures, un
    dépôt commun par empreinte les range une fois pour toutes les cartes, comme le fait déjà la bibliothèque
    des bras du contributeur.
  - **Ménage automatique.** Les fichiers d'une construction échouée ou interrompue sont supprimés. Une carte
    reconstruite après une mise à jour du jeu remplace l'ancienne, supprimée une fois la nouvelle prête.
    Désactiver « Rejeu 3D » demande s'il faut supprimer les fichiers 3D.
  - Affichage dans Jeux : place occupée, plafond, espace libre du disque, « Libérer l'espace ».
- **D-4 — joueurs.** **Tranché (user, 2026-11-15) : niveau du contributeur.** Cela couvre le corps animé du
  Spartan, les pièces d'armure et les peintures du joueur (sa personnalisation publique), l'arme en main et
  les bras en 1re personne.
- **D-5 — moment de la question.** **Tranché (user, 2026-11-15).**
  - Le réglage « Rejeu 3D » vit dans l'onglet **Jeux** (`titles`, `TitlesTab`). Il est visible au bureau
    seulement, et porte trois états : `non_demande`, `refuse`, chemin.
  - La question s'affiche à trois moments : dans l'assistant d'installation neuve ; en carte d'accueil tant
    que l'état est `non_demande` (app mise à jour) ; à l'activation de « Rejeu 3D » dans l'onglet Jeux.
  - « Pas maintenant » écrit `refuse` : la carte ne revient plus, et l'activation reste possible dans Jeux.
- **D-6 — recherche.** **Tranché (user, 2026-11-15) : oui.** Seulement les emplacements de la boutique
  choisie. Pour Steam : registre `SteamPath`, puis les bibliothèques de `libraryfolders.vdf`. Pour Xbox :
  `<lecteur>:\XboxGames\Halo Infinite\Content\deploy` sur chaque lecteur fixe. Si rien n'est trouvé, proposer
  « Parcourir ». Aucune recherche avant le clic.
- **D-7 — mini-carte.** **Tranché (user, 2026-11-15) : oui, un peu écartée des bords.**
  - Coin bas gauche du lecteur 3D, carrée, environ 22 % de la largeur, écartée des bords (marge par token
    d'espacement, valeur fixée par la maquette).
  - Centrée sur le joueur suivi (sinon sur la caméra libre), rayon par défaut 30 m, avec les niveaux de zoom
    de la 2D.
  - **Verrouillée au nord (user, 2026-11-15) : elle ne tourne jamais avec la caméra.** Le « nord » est
    l'orientation par quarts de tour choisie pour la carte en 2D. Seul le cône de la caméra tourne.
    Le user a transmis ce point au contributeur : vérifier à l'intégration s'il l'a traité (item 0.7).
    (amendé le 2026-11-17, plan de refonte D-14) Le « nord » est l'orientation EFFECTIVE de la 2D, rotation
    automatique comprise.
  - Clic sur la mini-carte = recentrer la caméra libre.
- **D-8 — emplacement.** **Tranché (user, 2026-11-15) : ok.** La 3D occupe l'emplacement de `ReplayCanvas`,
  donc les cadres Standard / Cinéma / Plein cadre l'accueillent sans changement. (amendé le 2026-11-17, plan
  de refonte D-9 et F.3, validés par le user le 2026-11-15) Le commutateur 2D / 3D va dans l'emplacement en
  superposition à gauche du bouton « Vue », posé par l'item F.3 de la refonte, et non dans le panneau de
  `settings/ReplayViewMenu.tsx`.
- **D-9 — joueurs vus à travers les murs.** **Demandé par le user (2026-11-15), transmis au contributeur.**
  Un joueur caché par un mur reste visible par son seul contour (silhouette surlignée), à la couleur de son
  équipe (tokens). Un joueur visible n'a pas de contour. Seul / Masquer s'appliquent aussi aux contours.
  Vérifier à l'intégration si le contributeur l'a fait (item 0.7) ; sinon, c'est l'item C.7.
- **D-10 — HUD en vue 3D.** **À trancher sur maquette (item C.0).** Constat : le lecteur du contributeur
  affiche un HUD complet (vie, bouclier, arme, munitions, grenades, équipement, fil des éliminations, score).
  Sur la page de rejeu, le fil, les fiches joueurs et le bandeau de score le répètent. Proposition :
  - HUD réduit au joueur suivi : réticule, vie et bouclier, arme et munitions, grenades et équipement ;
  - ni fil des éliminations ni score dans la 3D, car la page les montre déjà ;
  - (amendé le 2026-11-17, plan de refonte D-2 et D-4) en Cinéma et en Plein cadre, les fiches et le fil
    restent à l'écran : le HUD ne reprend ni le fil ni le score, dans aucun cadre. La maquette C.0 se dessine
    DANS les cadres validés de la refonte (tableaux A, K et D du canevas de la refonte), jamais sur l'ancienne
    page.

- **D-12 — où vit le code.** **Tranché (user, 2026-11-15, « ok parfait, à voir comment le contributeur a pensé les choses ») : proposition retenue, confrontée en 0.2 au découpage du source du contributeur ; un écart est rapporté au user avant le lot B.**
  - **Une cinquième brique Go** pour l'export 3D depuis le jeu : cartes, armes, armures, corps et
    animations du Spartan, bras, peintures. Nom proposé : `JGtm/halo-infinite-3d-assets`, module
    `github.com/JGtm/halo-infinite-3d-assets`, AGPL-3.0. Les fichiers portés d'Ekur gardent leur en-tête
    GPL-3.0-or-later, et la clause 7(b) du contributeur s'applique à sa part (README et NOTICE).
    - Motif : c'est le premier critère de séparation de l'ADR 0043, un contributeur externe. llampwall peut y
      contribuer directement et son application autonome peut en dépendre au lieu d'en garder une copie.
      Ses tests `gamefiles` lourds ne pèsent pas sur la CI de LevelUp.
    - Rang : la brique 3D importe `halo-infinite-map-images` (lecture de la géométrie de carte déjà là) et
      `halo-infinite-files-reader`. Aucune brique n'importe la brique 3D.
    - Les lecteurs génériques de formats du jeu (textures, modèles, animations) qu'il a ajoutés à sa copie
      de `halo-infinite-map-images` vont dans la brique la plus basse qui en a besoin. Le rangement fichier
      par fichier se décide en 0.2, avec l'audit 0.8.
    - La détection de l'installation (A.1) reste dans `halo-infinite-map-images`, où vit déjà `DeployRoot`.
    - Aucun fichier extrait du jeu dans la brique (ADR 0041) : ses tests lisent l'installation derrière le
      tag `gamefiles`.
  - **Le lecteur 3D reste dans LevelUp** (`apps/web/src/features/match-replay/three/`). Il est branché sur le
    document de rejeu, l'horloge, Seul / Masquer, les tokens de couleur et l'i18n de LevelUp. Un paquet npm
    partagé avec l'application du contributeur n'est envisagé que si lui le demande, dans un plan à part.
  - **L'exploitation reste dans LevelUp**, selon le découpage de l'ADR 0042 : réglage, file de construction,
    cache et anti-saturation, routes, capacités.
  - Gestes : la création du dépôt et l'accès du contributeur reviennent au user. Les PR et les tags de la
    brique sont délégués à sa session. Amendement de l'ADR 0043 (cinquième brique) et ligne de la table
    `archlint/briques_imports_test.go`.

- **D-13 — repère « rejeu 3D » dans les listes.** **Demandé par le user (2026-11-15).** **Règle validée par le user le 2026-11-17 (« règle du repère ok ») ; dessin : variante C, couleur ambre.** Les surfaces qui
  disent ou filtrent « rejeu disponible » ont une variante « rejeu 3D disponible » : filtre de l'Explorer,
  tuiles de match, tableaux, page match. Proposition, **à valider** :
  - un match a un rejeu 3D quand son rejeu 2D existe ET que l'export sait construire sa carte depuis le jeu
    installé ;
  - le repère ne dépend pas de la présence de la carte dans le cache : une carte supprimée par le plafond se
    reconstruit à l'ouverture ;
  - hors bureau, ou si « Rejeu 3D » est inactif, rien ne change par rapport à aujourd'hui ;
  - le dessin du repère se choisit sur la maquette C.0 (lot E).

## 1 bis. Orchestration : pistes parallèles et modèles

Demande du user (2026-11-15) : un plan pensé pour la parallélisation, exécuté par **Sonnet** principalement,
**Haiku** dès que possible, **Opus** si nécessaire, **Fable** pour l'étape 0 et la revue finale.

Règles d'exécution (elles précisent le contrat `plan-execution` pour ce plan) :
- **Pistes.** Le plan se découpe en pistes indépendantes. Dans une piste, l'ordre est strict : un item est
  clos et vérifié avant le suivant. Des pistes différentes tournent en même temps.
- **Dépendances.** Un item ne démarre que si les items de sa colonne « Attend » sont clos. Seules les
  dépendances écrites dans la table ci-dessous comptent.
- **Un agent par worktree.** Worktree `LevelUp-wt-3d-<piste>`, sans jonction vers `data/`. Pour une brique,
  c'est un clone ou un worktree de son dépôt. Chaque agent reçoit un `GOCACHE` dédié.
- **Agents sans commit ni push.** Le pilote relit le compte rendu sur pièces, commite dans le worktree, puis
  intègre dans `feat/rejeu-3d` par rebase. Il lance la porte et pousse selon D-2. Dans une brique, le pilote
  pousse après relecture ; le tag est délégué à la session de la brique.
- **Bornes.**
  - Au plus 4 agents Sonnet ou Opus d'écriture en même temps ; les agents Haiku s'y ajoutent.
  - Une seule construction de carte ou un seul test `gamefiles` à la fois sur la machine (bombe RAM).
  - Un seul `make gate-push` à la fois.
- **Fichiers partagés.** Une seule piste écrit chacun des fichiers suivants à un instant donné ; la table
  ci-dessous dit laquelle : `capabilities.toml`, `config/deployment.go`, `domain/bootstrap.go`,
  `deploymentCapabilities.ts`, `ReplayViewMenu.tsx`, `replayPreferences.ts`, `TitlesTab`,
  `lib/query/keys.ts`. `openapi.yaml` et `generated.ts` sont régénérés par le pilote après chaque
  intégration (`make generate-types`), jamais fusionnés à la main.
- **Modèles.**
  - Sonnet par défaut.
  - Haiku pour un item mécanique, que le pilote contrôle sur pièces.
  - Opus seulement là où la table l'indique, avec son motif.
  - Fable pour l'audit 0.8 et la revue finale R.1.
  - Tout agent porte `model` explicite.

| Piste | Item | Modèle | Attend | Note |
|---|---|---|---|---|
| P0 Préalables | 0.1 pilote prévenu | pilote | go | message au pilote de la 8.3.0 |
| P1 Maquettes | C.0 page de maquettes | Sonnet | go | aucun code ; tranche D-7 (variante), D-10, D-13 (dessin) |
| P2 Installation | A.1 recherche dans `halo-infinite-map-images` | Sonnet | go | dépôt de la brique |
| P2 | A.4 capacité `can_use_game_install` | Haiku | go | trois endroits, modèle existant |
| P2 | A.2 + A.3 réglage et route | Sonnet | A.1 tagué, A.4 | |
| P2 | A.5 + A.6 web et tests | Sonnet | A.2 + A.3 intégrés | puis Gate A |
| P3 Moteur 2D | D.2 extraction de `ReplayCanvas` + témoins | Sonnet | go | refactor seul, aucun changement de rendu |
| P4 Règles du cache | B.4b règles pures anti-saturation | Sonnet | go | tailles estimées en paramètre, valeurs de M-1 posées ensuite |
| P5 Source et mesures | 0.2 inventaire de provenance (fichiers, licences, en-têtes) | Haiku | source reçu | |
| P5 | 0.2 comparaison du découpage à D-12 | Sonnet | inventaire 0.2 | écarts au user |
| P5 | 0.7 demandes D-7 / D-9 traitées ? | Haiku | source reçu | recherche dans le source, fichier et ligne |
| P5 | 0.6 M-4 écart des documents de rejeu | Sonnet | source reçu | |
| P5 | 0.3 M-1 tailles et durées | Sonnet | 0.2 | seul sur la machine pendant les constructions |
| P5 | 0.4 M-2 écart de `z` | Sonnet | 0.3 | |
| P5 | 0.5 M-3 images par seconde | Sonnet | 0.6 | l'agent prépare la mesure, le user la lance (aucun navigateur côté agent) |
| P5 | 0.8 audit d'optimisation | **Fable** | 0.2, 0.3 | lecture seule ; puis choix du user, Gate 0 |
| P6 Brique 3D | B.1 export de carte | Sonnet | Gate 0, dépôt créé par le user | |
| P6 | B.2 bibliothèques | Sonnet | Gate 0, dépôt créé | en parallèle de B.1 (autres paquets) ; tests `gamefiles` l'un après l'autre |
| P6 | E.1 `PrendEnCharge` | Sonnet | B.1 | |
| P7 Service 3D | B.3 + B.4 + B.5 + B.6 | Sonnet | B.4b, B.1 + B.2 tagués, Gate A | puis Gate B |
| P8 Lecteur 3D | C.1 dépendance `three`, chargement différé | Haiku | go | version exacte figée, aucun code de scène |
| P8 | C.2 + C.4 logique de scène et caméras | Sonnet ; **Opus** si M-4 relève un écart de structure (pose, animation, repère) | 0.6, C.1 | |
| P8 | C.3 horloge, joueur suivi, Seul / Masquer | Sonnet | C.2 | |
| P8 | C.7 contours à travers les murs | **Opus** (passe de rendu avec profondeur) ; sans objet si 0.7 les trouve faits | C.0 validé, C.2 | |
| P8 | C.5 + C.6 commutateur, progression | Sonnet | Gate B, C.3, refonte fusionnée (F.3) | seul écrivain de `ReplayViewMenu.tsx` et `replayPreferences.ts`, après les items C.3, F.2 et F.3 de la refonte |
| P8 | C.8 HUD | Sonnet | C.0 validé, C.3 | puis Gate C |
| P3 | D.1 + D.3 mini-carte | Sonnet | C.0 validé, D.2, C.3 | puis Gate D |
| P9 Listes | E.3 + E.4 Go (règle, champs, filtre) | Sonnet | D-13 validé | port `Replay3DAvailability` simulé jusqu'à E.2 |
| P9 | E.2 port réel | Sonnet | E.1 tagué, Gate B | |
| P9 | E.5 + E.6 web | Sonnet | C.0 validé, C.5, E.2 | puis Gate E |
| P10 Clôture | R.1 revue finale | **Fable** | Gates A à E | voir section 7 ter |
| P10 | R.2 corrections retenues | Sonnet ; Opus si l'écart est un algorithme | R.1 | une seule ronde |
| P10 | NOTICE, crédits, report des `[!]` | Haiku | R.2 | ADR 0049, note de version et journal : pilote |

Vagues qui en découlent, pour lecture rapide :
- **Dès le go, sans le source** : P0, P1, P2 (A.1, A.4), P3 (D.2), P4, P8 (C.1). Quatre agents Sonnet et
  deux Haiku.
- **Dès le source reçu** : P5 en parallèle des pistes encore ouvertes.
- **Après Gate 0** : P6 en parallèle de P8 (C.2, C.4, C.3, C.7).
- **Après Gate B** : P7 se clôt, puis C.5, C.6, E.2 ; P3 (D.3) et P9 se terminent.
- **Fin** : R.1, R.2, clôture.

## 2. Ce qui existe (relu sur pièces)

- **Détection du jeu.** `himap.DeployRoot()` (module `halo-infinite-map-images@v1.3.0`, `deploy_root.go`) lit
  l'env `LEVELUP_HALO_DEPLOY`, puis trois chemins fixes (Steam C:, `D:/SteamLibrary`, `C:/XboxGames`). Elle
  ne lit ni le registre, ni `libraryfolders.vdf`, ni les autres lecteurs. Appelants : seulement des outils
  `cmd/*` hors ligne. Aucun réglage, aucune route, aucune UI.
  - Garde-rails : `archlint/deploy_root_copie_unique_test.go` (un seul littéral du chemin du jeu) et
    `archlint/gamefiles_tag_test.go`.
  - **Piège :** la coquille retire `LEVELUP_*` de l'environnement du serveur (`sidecar.rs:25`,
    `STRIPPED_PREFIXES`). Le chemin doit venir du réglage, jamais de l'env.
- **Sélecteur de dossier.** Commande Tauri `pick_folder` (`folder_pick_tauri.rs:20`, ACL
  `capabilities/pick-folder.json`), côté web `lib/desktop/pickFolder.ts` et `components/ui/FolderPickButton.tsx`.
  Aucun changement Rust n'est nécessaire.
- **Onglet Jeux.** `features/settings/SettingsPage.tsx:107,142` (`id: 'titles'`, libellé « Games » / « Jeux »,
  `TitlesTab`).
- **Réglage modèle.** `film_archive_dir` :
  - Écriture : `PATCH /settings` (`handlers/settings.go:92`), validation dans `handlers/settings_film_archive.go`,
    écrivain unique `platform/settings.Store.Update`, fichier `app_settings.json`.
  - Lecture au démarrage : `config/config_roots.go:64-92` ; prise à chaud : `title.SetFilmArchiveRuntime`.
  - UI : `FilmArchivePanel.tsx`, aussi monté dans l'assistant (`features/setup/SetupPage.tsx:86`).
- **Capacités (ADR 0044).** `config/deployment.go` (`DeploymentCapabilities`, `capabilitiesOf`),
  `domain/bootstrap.go` (`CapabilityMap`), `service/bootstrap_service.go`, et côté web
  `lib/capabilities/deploymentCapabilities.ts` (`DEPLOYMENT_CAPABILITY_DEFAULTS`, `useDeploymentCapability`).
  Il n'existe aucune capacité « jeu installé » ni « 3D ».
- **Tâche de fond de plusieurs minutes.** Modèle à suivre : `internal/replayregen` (file, pause, reprise au
  redémarrage, respect du mode jeu `waitTurn`), avec `GET /desktop/replay-regen`. Pas de SSE : le web relit
  par `refetchInterval`.
- **Données dérivées par machine.** Sous `PathResolver.CacheRootDir()` (`<Data>/cache`, `registry.go:870`).
- **Rejeu web** (`apps/web/src/features/match-replay/`) :
  - `ReplayPage.tsx` (grille, carte à gauche) et `ui/ReplayCanvas.tsx` (canvas 2D monolithique, 809 lignes).
  - Calques : `layers/replayCompose.ts`, `buildScene` (ReplayCanvas l.466). Marqueurs :
    `layers/replayMarkers.ts` (`drawTracksLayer`). Vue : `model/replayView.ts` (`CanvasView`, `projectTo`,
    `orient`).
  - Horloge : `model/playbackStore.ts`, `hooks/useReplayPlayback.ts`, `hooks/useReplayClock.ts`.
  - Seul / Masquer : `hooks/useReplayPlayerFocus.ts`. Zoom : `hooks/useReplayZoom.ts` (`ZOOM_LEVELS`).
  - Aucune mini-carte. Aucune dépendance 3D dans `package.json`.
- **Vue Plan.** `?variante=plan` sur `GET .../replay/background.png`, calage commun `mapfondsidecar`, web
  `useReplayMapImage` et `lib/replay/mapBackgroundVariant.ts`. 111 cartes sur 114 ont un Plan ; Live Fire et
  deux variantes de Thunderhead n'en ont pas.
- **Document de rejeu.** `domain/replaydoc/document.go`, décodeur v1.3.7.
  - `Point {t,x,y,z,h,p,...}` en mètres monde, 33 ms par image. `h` = cap de visée en degrés, `p` = élévation.
  - Véhicules et équipement avec `z`. **Projectiles et tirs sans `z`.**
- **Personnalisation des joueurs.** À relire en lot 0 : ce que la synchro range déjà de l'apparence publique
  (armure, peintures) et ce que le lecteur du contributeur demande en plus.
- **Identité de carte.** Le sidecar et `map_quant_bounds.json` portent `module` : dossier de niveau pour une
  carte native, canevas pour une carte Forge. Pour une carte Forge, il faut en plus les objets de la variante
  (`.mvar`, déjà lus par `mapobj-build`).

## 3. Lot 0 — préalables (aucun code LevelUp)

- [~] 0.1 Le pilote de la 8.3.0 est
      prévenu que la release attend ce plan (D-2). Couvert le 2026-11-17 : le pilote de la 8.3.0 est la session
      du plan maître, qui porte cette attente (sa section « Sortie de la 8.3.0 »).
- [ ] 0.2 Source du contributeur reçu (dépôt privé partagé ou archive) ; son découpage en paquets comparé à D-12, écarts rapportés au user ; licence, clause 7(b) et fichiers portés d'Ekur relus. La liste des
      fichiers repris et leur destination (brique ou LevelUp) est rangée dans `.ai/annexes_rejeu_3d/provenance.md`.
- [ ] 0.3 M-1 : taille et durée de construction d'une carte native (Recharge) et
      d'une Forge ; idem pour la bibliothèque d'armes, d'armures et de bras. Part des textures partagées
      entre 5 cartes (gain du dépôt commun de D-11). Chiffres dans `annexes_rejeu_3d/mesures.md`.
- [ ] 0.4 M-2 : écart de `z` entre le document et le sol de la carte 3D sur 3 films. Il y a un défaut si la
      médiane dépasse 0,3 m.
- [ ] 0.5 M-3 : son lecteur branché sur un document de rejeu LevelUp dans la fenêtre WebView2 de l'app :
      images par seconde et mémoire GPU, réglages de rendu au plus haut et au plus bas. Seuil : 60 i/s sur le poste du user.
- [ ] 0.6 M-4 : écart entre son document de rejeu et le nôtre (champs qu'il lit et que le nôtre n'a pas).
      Chaque écart devient un item des lots B ou C, ou une découverte. (amendé le 2026-11-17) Un écart qui
      demanderait de changer le décodeur ou le document de rejeu revient au user avant tout code.
- [ ] 0.7 Les deux demandes transmises au contributeur sont-elles traitées dans le source reçu ? Il s'agit
      des contours à travers les murs (D-9) et de la mini-carte verrouillée au nord (D-7). Chacune est notée
      « faite par lui » (fichier et ligne) ou « à faire » (items C.7 et D.3).
- [ ] 0.8 Audit d'optimisation du code du contributeur par un agent **Fable** (`claude-fable-5-2`, demande
      du user du 2026-11-15), en lecture seule, après 0.2 et 0.3. Il couvre trois axes :
      - traitement : étapes de construction refaites inutilement, lectures du jeu répétées ;
      - mise en commun : textures, modèles et matériaux partagés entre cartes et entre bibliothèques ;
      - parallélisation : constructions ou étapes indépendantes, bornées en mémoire (règle de la bombe
        RAM), et stockage (format, compression, nombre de fichiers, petits fichiers des peintures).
      Chaque proposition porte un gain chiffré sur les mesures M-1, un coût et un risque. Une proposition
      sans mesure est écartée (pas de lot de performance sans mesure). Rendu :
      `annexes_rejeu_3d/audit_optimisations.md`. Le user retient ou écarte chaque proposition ; une
      proposition retenue devient un item des lots B ou C, avec son témoin avant / après.
- **Gate 0 :** annexes présentes, chiffres écrits, 0.7 statué, chaque proposition de 0.8 retenue ou écartée par le user.

## 4. Lot A — installation du jeu : question, recherche, réglage

Couches : la recherche vit dans la brique (`halo-infinite-map-images`), l'orchestration dans
`internal/service/gameinstall/`, le handler reste mince.

- [ ] A.1 Brique (dépôt dédié, PR et tag délégués à la session de la brique) :
      `himap.CandidatsInstallation(boutique)` (Steam : registre et `libraryfolders.vdf` ; Xbox : lecteurs fixes)
      et `himap.ValideDeploy(chemin)` (présence de `<racine>/<variante>/levels/multi`). Le littéral du chemin
      reste unique. Tests sur un faux arbre et un faux `libraryfolders.vdf`.
      Fait le 2026-11-17, hors tag : code écrit et relu sur pièces par le pilote (vet et tests rejoués, aucun
      chemin personnel), deux fonctions du lecteur `.vdf` découpées après un refus du lint, CI verte (tests
      Linux et Windows, lint, vulnérabilités), demande de fusion 5 fusionnée dans `main` de la brique. Le tag
      (version mineure) n'est PAS posé : il fige ces deux signatures sous la promesse de compatibilité ; il
      se pose quand A.2 en a besoin, après la comparaison au source du contributeur (0.2). D'ici là LevelUp
      peut viser une pseudo-version sur sa branche de travail (ADR 0043). Écarts retenus : `ValideDeploy` rend une erreur (nil = valide) qui dit ce qui manque ;
      une bibliothèque Steam sans le jeu reste candidate, c'est `ValideDeploy` qui trie ; l'ancien format de
      `libraryfolders.vdf` (sans clé `path`) n'est pas lu.
- [ ] A.2 Réglage `replay_3d` (`etat` : `non_demande` | `refuse` | `actif` ; `deploy` : chemin absolu ; `plafond_cache` : octets, 20 Go par défaut ; `dossier` : chemin absolu,
      `<Data>/cache/replay3d` par défaut) :
  - champ `AppSettings`, types `domain.Replay3DSettings` / `Replay3DUpdate` ;
  - validation dans `handlers/settings_replay_3d.go` (chemin absolu et `ValideDeploy`, 400 sinon) ;
  - lecture au démarrage dans `config_roots.go`, prise à chaud comme `SetFilmArchiveRuntime` ;
  - jamais par l'env (piège de `STRIPPED_PREFIXES`).
- [ ] A.3 Route `POST /desktop/game-install/search {store}` qui renvoie les candidats validés. Elle n'écrit
      rien : le choix du user passe par `PATCH /settings`. Journal
      `slog.InfoContext(ctx, "game install search", "store", s, "found", n)` ; une erreur de lecture du
      registre est journalisée avant le repli.
- [ ] A.4 Capacité `can_use_game_install` (bureau seul) aux trois endroits : struct, `CapabilityMap` avec son
      mapping, `DEPLOYMENT_CAPABILITY_DEFAULTS`. L'état « jeu trouvé » est une donnée servie par
      `GET /settings/replay-3d`, pas une capacité.
- [ ] A.5 Web :
  - composant `GameInstallQuestion` (Steam / Xbox / Parcourir avec `FolderPickButton` / Pas maintenant) ;
  - monté à trois endroits (D-5) : l'assistant (`SetupPage.tsx`, à côté de `FilmArchivePanel`), la carte
    d'accueil tant que l'état est `non_demande`, et la section « Rejeu 3D » de `TitlesTab` ;
  - dans `TitlesTab`, la section porte aussi l'interrupteur (son activation ouvre la question), le dossier
    du jeu, le dossier des fichiers 3D, le plafond, la place occupée, l'espace libre du disque et « Libérer
    l'espace » (D-11) ; la désactivation demande s'il faut supprimer les fichiers 3D ;
  - textes fr et en dans le dictionnaire typé de `features/settings/i18n/` ; clés de requête dans
    `lib/query/keys.ts`.
- [ ] A.6 Tests :
  - Go : handler httptest 200/400, service sur faux arbre, capacité fausse en hébergé ;
  - vitest : carte d'accueil masquée après réponse, activation dans Jeux qui ouvre la question, « Pas
    maintenant » qui laisse l'interrupteur éteint.
- **Gate A :** `cd apps/go-api && go test ./internal/service/gameinstall/... ./internal/api/handlers/... ./internal/config/... ./internal/archlint/...`,
  `make check-types`, `make test-web`. Puis vérification à l'écran par le user dans l'artefact `verifs` :
  installation neuve simulée (Data vide), app mise à jour, activation depuis Jeux.

## 5. Lot B — construction des ressources 3D depuis le jeu

Toute lecture du jeu vit dans la brique 3D (D-12), qui s'appuie sur `halo-infinite-map-images`. LevelUp orchestre, range et sert.

- [ ] B.1 Brique 3D : export de la carte en GLB (géométrie, instances, lumières, matériaux et textures en pleine
      qualité ; pour une carte Forge, le canevas et les objets du `.mvar`). Tests derrière le tag `gamefiles`
      (fichiers `*_gamefiles_test.go`), plus un test sans jeu sur un GLB minimal.
- [ ] B.2 Brique 3D : bibliothèques d'armes, de pièces d'armure, de bras en 1re personne et corps animé du
      Spartan ; export des tables de peintures (portage d'Ekur, mentions conservées). Mêmes règles de tests.
- [ ] B.3 `PathResolver.Replay3DDir(slug)` = `<dossier>/<slug>/` (`cartes/`, `textures/` si M-1 le justifie,
      `armes/`, `armures/`, `bras/`, `peintures/`). Le dossier vient du réglage, sinon `<Data>/cache/replay3d`.
      Chaque fichier porte une empreinte du jeu et de la révision de l'exporteur : une mise à jour du jeu ou
      de l'exporteur reconstruit le fichier, et l'ancien est supprimé une fois le nouveau prêt. Une
      construction écrit dans un fichier temporaire renommé à la fin ; un temporaire orphelin est supprimé au
      démarrage.
- [ ] B.4 Service `internal/service/replay3d/`. C'est une file à la manière de `replayregen` : une
      construction à la fois, attente pendant le mode jeu, reprise au redémarrage, journal de début, de fin
      (durée, octets) et d'échec. À l'activation : bibliothèques communes. À l'ouverture d'un rejeu en 3D : sa
      carte, et rien d'autre (aucune construction anticipée). La 2D n'est jamais bloquée.
- [ ] B.4b (partie pure faite le 2026-11-17, sur `feat/rejeu-3d` : `PlanEviction` et `CheckFreeSpace`,
      plafond par défaut de 20 Go et minimum libre de 10 Go comptés en multiples de 2^30 ; build, vet, tests,
      archlint, gofmt et golangci-lint verts, code relu et tests rejoués par le pilote. L'index des dernières
      ouvertures et le déplacement de dossier, qui touchent le disque, partent avec le service B.4 :
      `internal/analysis/` n'a pas droit au stockage. La case se coche avec B.4.)
      Anti-saturation (D-11), règles pures dans `internal/analysis/replay3dcache/`, testées sans
      disque :
  - choix des cartes à supprimer au-delà du plafond (dernière ouverture la plus ancienne d'abord ; jamais la
    carte ouverte ni une bibliothèque) ;
  - garde d'espace libre avant construction (taille estimée d'après M-1, 10 Go minimum restants). Un refus
    est journalisé (`slog.WarnContext`, place manquante) et rendu au web comme un état, pas comme une erreur
    500 ;
  - dernière ouverture par carte, tenue dans un petit index JSON du dossier, écrit sous verrou par le
    service seul ;
  - déplacement du dossier sur le modèle de `film archive-move` (copie, vérification, suppression de
    l'ancien).
- [ ] B.5 Routes, avec les handlers dans `handlers/replay3d.go` :
  - `GET /desktop/replay-3d/state` : file, pourcentage, place occupée ;
  - `POST /desktop/replay-3d/purge` : « Libérer l'espace » ;
  - `GET /players/{slug}/matches/{id}/replay/3d/scene` : manifeste de la carte et des pièces du match,
    personnalisation des joueurs ;
  - `GET .../replay/3d/files/{nom}` : lecture par plage, cache immuable par empreinte.

  Un titre sans capacité `replay_3d` (`capabilities.toml` ; Halo 5 : absente) renvoie
  `ErrCapabilityNotSupported`, donc un 404 propre.
- [ ] B.6 Tests :
  - règles : plafond, garde d'espace, carte ouverte protégée ;
  - service : file, reprise, empreinte périmée, temporaire orphelin, purge, déplacement de dossier ;
  - handler : 200, 404 titre, 404 hébergé, plage, refus faute de place.
- **Gate B :** `go test ./internal/analysis/replay3dcache/... ./internal/service/replay3d/... ./internal/api/handlers/...`,
  `go test -tags=gamefiles` sur la brique pour Recharge et une Forge, tailles dans les bornes de M-1.

## 6. Lot C — lecteur 3D

- [ ] C.0 Une seule page de maquettes (artefact claude.ai à onglets), validée par le user AVANT tout code
      qui dessine ce qu'elle montre (C.7, C.8, D.3, E.5) ; D.2 (refactor sans changement de rendu) et C.1
      (dépendance seule) n'attendent pas. Elle couvre : HUD de la 3D (D-10, deux ou trois variantes, dont le cas Plein cadre),
      mini-carte (D-7, deux ou trois variantes, verrouillée au nord), contours à travers les murs (D-9) et
      repère « rejeu 3D » des listes (D-13, deux ou trois variantes sur une tuile, une ligne de tableau et
      le bouton de la page match).
      Publiée le 2026-11-17 : https://claude.ai/artifact/V3Ybivkczabep8oU7U15m8 (source :
      `.ai/annexes_rejeu_3d/maquettes_c0.html`), dessinée dans les cadres validés de la refonte ; trois HUD,
      trois places de mini-carte (elles règlent la rencontre avec le fil posé en bas à gauche du lecteur),
      deux styles de contour, trois repères ; ligne posée dans l'artefact des vérifications. Fichier relu en
      entier par le pilote, rendu non vérifié à l'écran. Reste le choix du user.
      **Choix du user (2026-11-17)** : repère des listes = variante C (icône neuve, un cube dans la boucle), avec la
      couleur ambre proposée (« on distingue d'un coup d'œil ») : un token sémantique lui est donné en E.5 (skill
      `color-tokens`). HUD, mini-carte et joueurs à travers les murs : NON choisis, le user attend de voir le
      travail du contributeur (« ce sera du ThreeJS […] il bosse bien son sujet ») ; C.7, C.8, D.1 et D.3 attendent
      donc le source, et la rencontre du fil et de la mini-carte (plan maître, N21) se tranche à ce moment-là.
- [x] C.1 Dépendance `three` (MIT), version exacte figée. Le lecteur `features/match-replay/three/Replay3D.tsx`
      se charge en différé (import dynamique) : aucun poids ajouté à la 2D ni à l'hébergé.
      Fait le 2026-11-17 (`b1edd4d1bd` sur `feat/rejeu-3d`) : `three` et `@types/three` 0.187.0 exacts, point
      d'entrée unique `Replay3DLazy`, garde-rail `threeImportBoundary.guard.test.ts` ; typecheck, 6 tests,
      eslint, couleurs, imports entre features et knip verts, relus sur pièces par le pilote. À la réception
      du source du contributeur (0.2) : comparer sa version de `three` à celle-ci.
- [ ] C.2 Logique pure dans `features/match-replay/three/scene3dLogic.ts` (pose et animation d'un joueur à
      l'image fractionnaire, arme tenue, caméras), avec ses tests vitest. Le composant ne porte aucune
      logique métier. Code repris du contributeur avec crédit (D-1), adapté au document LevelUp (M-4).
- [ ] C.3 Même horloge (`playbackStore`, `frameRef`), même joueur suivi, mêmes `shownAtSlot` et `shown`
      (Seul et Masquer = toute la scène, règle du 15/11). Un joueur présent qui n'a pas quitté reste toujours
      affiché, même figé (règle du 08/11).
- [ ] C.4 Caméras : libre (souris et clavier), 3e personne, 1re personne avec bras et arme. Les tirs et
      projectiles sans `z` partent de la hauteur des yeux du tireur ; ce choix est écrit dans le code et dans
      la note de version.
- [ ] C.5 (amendé le 2026-11-17) Commutateur 2D / 3D dans l'emplacement en superposition posé par l'item F.3
      de la refonte (à gauche du bouton « Vue »), et non dans le panneau de `ReplayViewMenu.tsx` ; les réglages
      de rendu de D-3, eux, restent dans le panneau « Vue ». Il n'est visible que si `can_use_game_install` est
      vrai, que « Rejeu 3D » est actif et que le titre a `replay_3d`. Sinon le bloc affiche son état vide,
      jamais un bouton inactif. Le choix est mémorisé dans `replayPreferences.ts`.
- [ ] C.6 Progression « Préparation de la 3D » (lecture de `/desktop/replay-3d/state`) ; la 2D reste
      utilisable.
- [ ] C.7 Contours à travers les murs (D-9), sauf si 0.7 les trouve faits. La règle (caché ou non, couleur,
      Seul / Masquer) vit dans `scene3dLogic.ts`, avec ses tests. Le rendu, par passe de contour, vit dans le
      composant.
- [ ] C.8 HUD de la 3D selon la variante choisie en C.0 (D-10). Les textes sont en fr et en, et aucun
      élément ne répète ce que la page affiche déjà à côté.
- **Gate C :** `make check-types`, `make test-web`, vitest `scene3dLogic` (pose, caméras, contours). Puis vérification à l'écran par le
  user dans `verifs` : 3 rejeux choisis par lui, en lecture et pas seulement à l'arrêt.

## 7. Lot D — mini-carte

- [ ] D.1 Variante de la mini-carte choisie en C.0, recopiée ici avec la date du choix. Rien en D
      avant ce choix.
- [ ] D.2 Extraction du moteur 2D : `buildScene` et `draw` sortent de `ReplayCanvas.tsx` vers
      `layers/replayRenderer.ts`, sans changement de rendu. (amendé le 2026-11-17 : un agent n'ouvre pas de
      navigateur) Témoin avant / après : suites d'appels de dessin enregistrées par le contexte enregistreur
      existant (`features/match-replay/test/recordingContext`), 3 documents de test à 3 instants, capturées
      AVANT l'extraction et identiques APRÈS. Branche `refactor/rejeu-moteur-2d`, fusionnée dans
      `release/v8.3.0` dès son gate (plan maître, P-5).
- [ ] D.3 `ui/ReplayMiniMap.tsx` :
  - vue Plan forcée (`variante=plan`), repli sur le Rendu sans erreur ;
  - `CanvasView` centrée selon D-7 ;
  - calques réduits (fond, joueurs, morts, objectifs) ;
  - orientation de la carte, marge aux bords par token.
- **Gate D.2 (ajoutée le 2026-11-17, l'item fusionne seul) :** `make check-types`, `npx vitest run
  src/features/match-replay`, `node tools/lint-cross-feature-imports.mjs`, témoins identiques ;
  `ReplayCanvas.tsx` plus court qu'avant, aucun fichier neuf au-delà de 500 lignes.
- **Gate D :** `make check-types`, `make test-web`, témoins D.2 identiques, `make gate-push`. Puis
  vérification à l'écran par le user dans `verifs`, en lecture.

## 7 bis. Lot E — repère et filtre « rejeu 3D » dans les listes (D-13)

- [ ] E.1 Brique 3D : `PrendEnCharge(deploy, carte)`. Elle dit si l'export sait construire la carte : module
      natif présent dans l'installation, ou canevas Forge présent avec une variante lisible. Cela se décide
      sans rien construire. Tests sur un faux arbre.
- [ ] E.2 Port `port.Replay3DAvailability`, sur le modèle de `port.ReplayAvailability` : l'ensemble des
      cartes prises en charge, calculé une fois par installation et par révision de l'exporteur, puis gardé
      en mémoire. Il est invalidé au changement du dossier du jeu ou après une mise à jour du jeu. Vide quand
      « Rejeu 3D » est inactif ou hors bureau ; jamais un accès disque par ligne.
- [ ] E.3 Champ `has_replay_3d` (`omitempty`) à côté de `has_replay` partout où ce dernier est servi, soit
      les six producteurs relus sur pièces : `domain/home.go`, `domain/match_history.go`,
      `domain/explorer.go`, `domain/teammates.go`, `domain/match_view.go` (`replay_3d_available`) et
      `domain/tactical_cellule.go`. L'exécutant RE-VÉRIFIE la liste par
      `git grep -n -E 'HasReplay|ReplayAvailable' apps/go-api/internal/domain` avant de coder. Règle unique dans
      `internal/analysis/` : rejeu 2D présent ET carte prise en charge. Le garde-rail interdit de recalculer
      cette règle ailleurs (règle des trois copies).
- [ ] E.4 Filtre : `ReplayScope` passe à `'' | 'with' | 'with_3d' | 'without'`, en Go
      (`service/match_history_filter_replay.go`, `domain/explorer.go`, `domain/match_history.go`) et en web
      (`features/explorer/explorerScope.ts`). « Avec rejeu 3D » n'est proposé que si « Rejeu 3D » est actif
      au bureau ; sinon la valeur `with_3d` d'une URL retombe sur `with`, comme le nettoyage existant de
      `ExplorerPage.tsx`. Textes fr et en.
- [ ] E.5 Repère : `lib/match-nav/MatchReplayLink.tsx` reste le point unique de l'icône. Il gagne une
      présentation « 3D » selon la variante choisie en C.0, dans ses trois tailles (`icon`, `button`,
      `large`), plus le bouton de la page match (`MatchHeader.replayLink.tsx`). Le lien ouvre le rejeu
      directement en 3D (paramètre de recherche de la route, lu par le commutateur de C.5). Une carte non
      encore construite n'empêche pas le repère : la construction démarre à l'ouverture (B.4).
- [ ] E.6 Tests : règle E.3 (rejeu absent, carte non prise en charge, 3D inactive, hébergé), filtre E.4 (Go
      et vitest), présentations E.5 (vitest), nettoyage d'URL hors bureau.
- **Gate E :** `go test ./internal/analysis/... ./internal/service/... ./internal/api/handlers/... ./internal/archlint/...`,
  `make generate-types`, `make check-types`, `make test-web`. Puis vérification à l'écran par le user dans
  `verifs` : Explorer (filtre et colonne), tuiles de l'accueil, page match, Synergies escouade.

## 7 ter. Revue finale (piste P10)

- [ ] R.1 Revue finale par un agent **Fable** (`claude-fable-5-2`, demande du user du 2026-11-15), en
      lecture seule, suivant le skill `adversarial-review`. Elle porte sur le diff complet :
      `release/v8.3.0..feat/rejeu-3d`, plus les diffs des briques `halo-infinite-map-images` et
      `halo-infinite-3d-assets`. Axes de lecture :
      - suppressions de fichiers : plafond, purge, déplacement de dossier, temporaires ;
      - lectures du jeu bornées en mémoire ;
      - rien d'actif hors bureau ou sans « Rejeu 3D » ;
      - titre sans `replay_3d` ;
      - règles Seul / Masquer et joueur présent ;
      - licences, en-têtes Ekur et clause 7(b) ;
      - écarts aux mesures M-1 à M-3.
      Rendu : `annexes_rejeu_3d/revue_finale.md`. Chaque constat porte un fichier, une ligne et un scénario
      d'échec.
- [ ] R.2 Le pilote vérifie chaque constat sur pièces. Ceux retenus sont corrigés par Sonnet (Opus si le
      constat touche un algorithme), avec un test qui échoue avant le correctif. Une seule ronde : un constat
      neuf après R.2 va en découvertes.
- **Gate R :** chaque constat de R.1 retenu et corrigé, ou écarté avec motif écrit ; `make gate-push` vert.

Clôture :
- entrée `.ai/thought_log.md` ;
- report des items `[!]` dans `.ai/REGISTRE_REPORTS.md` ;
- ADR 0049 « rejeu 3D : fichiers du jeu lus sur la machine du user » (EN seul) : première lecture du jeu par
  le serveur en production, rangement sous Data, empreinte, capacité, provenance du code du contributeur ;
- `NOTICE` et crédits de LevelUp complétés (contributeur, Ekur) ;
- note de version 8.3.0.

## 8. Découvertes

(à remplir pendant l'exécution ; rien n'y est traité dans ce plan)

## 9. Reprise de session

Lire la section 1 (décisions), puis la table de la section 1 bis : pour chaque piste, première case vide
des sections 3 à 7 ter dont la colonne « Attend » est close. `git log --oneline -10` sur `feat/rejeu-3d` ;
`git worktree list` et `ListAgents` (agents encore en cours) ; `.ai/thought_log.md` (dernières entrées
« rejeu 3D »).
