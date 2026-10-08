# PLAN — Représentation intermédiaire du film, étape 2 (2026-10-03)

> **Statut : EN COURS — validé et lancé par l'utilisateur le 2026-10-03** (« Oui tu as mon
> accord », puis « je t'ai dit que je te le donnais » : l'accord est le GO) — dernier item de clôture
> de l'étape 1 (`.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE1_2026-10-02.md`), rédigé le 2026-10-03
> par la session qui a exécuté l'étape 1. L'étape 1 est fusionnée dans `feat/v75` (`67c379fc1`).
> Contrat : skill `plan-execution` (ce plan fait foi en cas de divergence). On ne revient vers
> l'utilisateur que pour une fusion dans `feat/v75` ou une décision produit nouvelle ; les pushes de
> la branche de travail pour la CI se font sans redemander.
>
> **Pour qui** : l'agent qui exécutera l'étape 2, et la session de la campagne de grammaire, qui
> travaille sur des fichiers voisins (§1.3).
>
> **Sources (à lire avant le premier lot)** :
> - `docs/adr/0037-film-intermediate-representation.md` — IR-4 (états et verdicts), IR-6 (couche de
>   récupération), IR-7 (paramètres hors flux), IR-10 (ordre de migration), IR-11 (tests T1 à T7) ;
> - `.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` — §3.3 (lots 2.1 à 3.2) ;
> - `.ai/V7.5/film_re/RAPPORT_IR_CARTOGRAPHIE_GO_2026-10-01.md` — §1.3 (table des traversées de la
>   cuisson), §1.6 (doublons), §3.3 (API du marcheur et `Distribuer`), §3.6 (couche de
>   récupération), §3.7 (ratchets), §3.8 (migration, fichiers et risques par lot) ;
> - `.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE1_2026-10-02.md` — §6 (découvertes) et §7 (journal :
>   pièges d'exploitation) ;
> - `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` — §3 (décisions), §6.1 et §6.2 (lots des vagues 1 et
>   2), pour la coordination du §1.3 ;
> - `CLAUDE.md` (règles, dont la règle 17 sur les commentaires).
>
> Les numéros de ligne des sources datent du 2026-10-01 : l'étape 1 et la campagne ont déplacé du
> code. Rouvrir chaque fichier avant d'agir (règle 4 du contrat).

---

## 0. Objet, critères de succès, périmètre

**Objet.** Faire LIRE LA STRUCTURE aux faits au lieu de leur faire parcourir le film : les canaux
(états de mouvement, tir continu, canaux d'image-clé, canaux de tête de vue A) deviennent des
consommateurs d'UNE marche (`grammar.Distribuer`) ; la récupération heuristique devient une couche
mutualisée dont chaque record est marqué « récupéré », nommé et compté (ADR 0037 IR-6) ; le
statborg descend de `facts/objectives` dans la grammaire ; puis les changements de comportement
DÉCLARÉS (morts d'objet sur le marcheur unique, canaux delta lus par la marche là où elle couvre
mieux, killsource en dernier). L'étape 3 (lots 3.1 et 3.2) retire les marcheurs redondants, unifie
le contexte d'une cuisson et mesure.

**Critères de succès (mesurables).**
1. **Différence nulle par lot** (T4) : `replay-equiv` identique sur le corpus et killsource
   identique à l'octet sur les témoins, SAUF ce que le lot déclare d'avance : des comptes de replis
   (2.2, 2.4, ADR 0034 D-10 amendé) ou un changement de comportement (2.7, prouvé au
   `replay-corpus-gate` et au banc de vérité `internal/replayverite`).
2. **T3 étendu** (IR-11) : à la fin de l'étape 2, tout fait publié cite une étendue qui existe dans
   la structure (record, composant, tour de vue C) ou un record marqué « récupéré » avec sa méthode.
3. **Un seul `IDLowBits`** et des paramètres hors flux avec leur provenance (IR-7).
4. **Pas de régression de plus de 10 %** de durée ni de pic mémoire d'une cuisson (trois témoins et
   un BTB, machine calme, binaires alternés), mesurée avant 2.4 et après chaque lot qui change le
   nombre de parcours (2.4, 2.5, 2.7, 3.1).
5. **Traversées** : le nombre de parcours du film par une cuisson (rapport §1.5 : ≈ 37 parcours delta,
   ≈ 14 parcours d'images-clés, 3 lectures du chunk de temps forts) est re-compté à la fin de chaque
   lot et ne remonte jamais ; cible de l'étape 3 : une marche, une récupération, un contexte.

**Hors périmètre** (explicite) :
- toute correction de grammaire (lecteurs de composants, localisateurs, naissances, fermeture) :
  c'est la campagne (§1.3) ;
- toute cuisson ou recuisson du parc, tout backfill : un lot qui fait monter `grammar.Rev` ou
  `killsource.Rev` (2.7) s'arrête à la preuve ; la recuisson et le backlog du collecteur se
  déclenchent sur signal de l'utilisateur ;
- Halo 5, la persistance de la structure (IR-8).

## 1. Base, branche, coordination

### 1.1 Branche et worktree
- Branche **`feat/ri-etape2`**, créée depuis `origin/feat/v75` APRÈS la fusion de l'étape 1.
- Worktree de l'étape 1 réutilisé (`LevelUp-wt-ri`, un seul exécutant), jonctions `apps/web/node_modules`,
  `data/cache/film_chunks` et `data/cache/film_manifests` vers le checkout principal ; toutes retirées
  par `(Get-Item <jonction>).Delete()` AVANT tout `git worktree remove` (jamais `--force`).
- Cache de compilation Go DÉDIÉ (`GOCACHE=%LOCALAPPDATA%\go-build-ri`) : la campagne compile en
  parallèle sur la même machine.

### 1.2 Fusions
- Au début de chaque lot : `git merge origin/feat/v75` (en conflit, `feat/v75` a raison), passe de
  référence sur la tête fusionnée (`replay-equiv -update` et killsource `json` des témoins), commit des
  références si elles bougent.
- La branche se fusionne dans `feat/v75` lot par lot ou par groupe de lots, chaque fois après CI verte
  au niveau job, `make gate-push` et accord de l'utilisateur ; la campagne est prévenue avant.

### 1.3 Coordination avec la campagne de grammaire

État au 2026-10-03 : vague 1 en cours (lecteurs `ti=3`, dispositifs `ti=43`, moteur, véhicules
`ti=40` en delta ; la marche d'image-clé de toutes les générations n'est PAS retenue). Vague 2
annoncée : unification des localisateurs jumeaux (LU, sans différence), signature du localisateur
sur tous les emplacements de l'archétype (LS), désaveu des déclarations que la table de datums dit
mortes (LP, avec peut-être une variante de la marche d'image-clé qui lit le bloc de type 1), et les
naissances lues par la grammaire des messages de la vue A.

| Lot de ce plan | Fichiers de la campagne qu'il touche ou qu'il lit | Règle |
|---|---|---|
| 2.0, 2.6 | aucun | libres dès la fusion de l'étape 1 |
| 2.1 | `movement_states.go`, `tir_continu.go` et `lecteur.go` (L4a de la vague 1 : renommage de `VehicleTypePhysicsAssumed`, champ `etatComplet`) | ces trois fichiers après la fusion de la vague 1 ; les fichiers neufs et ceux de la marche avant (convenu le 2026-10-03 ; la ligne d'origine disait « aucun ») |
| 2.2 (canaux d'image-clé) | `keyframe_anticipe.go`, `keyframe_liaison.go`, `keyframe_datums.go`, `keyframe_world*.go` (LP, variante de la marche d'image-clé) | avant que LP ne démarre, ou après sa fusion — CONVENU le 2026-10-03 : LP n'a pas démarré, 2.2 part ; LP se construira sur la structure |
| 2.3 (tête de vue A) | `frame_vue_messages.go` et les naissances par la vue A | après la fusion du lot des naissances, ou avec lui — CONVENU le 2026-10-03 : les naissances par la vue A sont un chantier de recherche sans lot écrit, 2.3 n'a pas à l'attendre |
| 2.4, 2.5 (récupération) | lecteurs de composants des vagues 1 et 2 (`dispatch_*.go`, `components_*.go`) en LECTURE seulement ; créations (`equipment_creation*.go`, `vehicle_creation.go`) | après la fusion de la vague 1 ; les lecteurs sont des briques, inchangées |
| 2.7 (a) morts d'objet, 3.1 localisateurs | `object_deaths_march.go`, `facts/killsource/walk.go` (LU, LS) | APRÈS la fusion de LU ; le retrait des localisateurs jumeaux est LU, pas ce plan |
| 2.7 (b) canaux delta par la marche | fermeture des trames (toute la campagne) | seulement là où la marche couvre AU MOINS autant que la recherche d'ancres, canal par canal, mesuré sur le corpus |
| 2.7 (c) killsource | `facts/killsource/*` (LU, LS) | en DERNIER, après LU ; LS, retiré par la campagne le 2026-10-04, n'imposait qu'un ordre d'écriture des mêmes fichiers (journal) |

Chaque session prévient l'autre quand elle fusionne dans `feat/v75` ; une découverte qui touche un
fichier de l'autre se signale, ne se corrige pas.

Ordre de fusion convenu avec la campagne le 2026-10-05 (chaque fusion reste soumise à l'accord de
l'utilisateur, §1.2) :
1. le lot « vue A » V1 de la campagne ;
2. 2.7.a et 2.7.a0 (ce plan), après fusion de `feat/v75` dans la branche et remesure ;
3. le lot LR de la campagne, rebasé sur cette tête : sa règle de lecture du jeu (compte ≥ 5 →
   échec) ne tient que sous le découpage déclaré ; son gate 2 officiel se joue sous `-mpp-declare` ;
4. le lot « vue A » V2 (il touche aussi `debut_de_liste.go`).
Le DELTA à génération contredite (découverte 15) va dans le lot de la campagne qui suit LR ; la
fermeture d'un épisode par le même objet occupant est déjà dans 2.7.a.

## 2. Décisions

**Utilisateur.**
- 2026-10-02 : la représentation intermédiaire est menée dans une autre conversation, en parallèle de
  la campagne de grammaire.
- 2026-10-03 : la mémoire de la marche des images-clés n'est pas partagée entre la cuisson et
  killsource à l'étape 1 ; elle se reprend à l'étape 2 (lot 2.7 (c) puis 3.1 : killsource devient un
  canal de la même marche, contexte unique par cuisson).
- GO de l'étape 2 : DONNÉ le 2026-10-03 (statut ci-dessus) ; l'ordre des lots qui attendent la
  campagne est réglé au §3 (report par le plan), la mutualisation par DT2-5.
- 2026-10-04 : « option A ». Les lectures heuristiques qui décident aujourd'hui DEVANT une lecture de
  la grammaire — les fenêtres de bits des images-clés (armes portées, marque de portage,
  inventaire) et l'ancrage d'en-tête bipède (positions et huit balayages) — restent HORS du registre
  des replis jusqu'aux lots de comportement de 2.7, qui les ordonneront derrière la grammaire ; elles
  s'y inscriront alors « après la lecture », comptées, leurs records marqués récupérés (item
  2.7.d). Pas de montée du cliquet `NbDevantLaLecture`.
- 2026-10-05 : « Oui je valide cette piste. Mais à noter comme potentielle optimisation avec Cheat
  Engine dans le BACKLOG.md ». Réponse à la proposition suivante : lire le bloc MPP des anciens
  films d'après la taille d'état de création que chaque film déclare (`n1`, découverte 13), au lieu
  de la largeur calibrée film par film. Item 2.7.a0 ; l'observation dynamique avec Cheat Engine est
  inscrite au backlog (`.ai/BACKLOG.md`).
- 2026-10-05, en réponse à l'instruction du gate de corpus (journal du même jour) : « Oui,
  admises » — les baisses instruites des huit films (postures, changements d'arme, tir continu,
  dotations de naissance, trajets) sont admises ; « Par joueur » — la primauté de la lecture
  nomme les occupants par joueur, plus par corps (découverte 16).
- 2026-10-06, en réponse à l'instruction du gate de 2.7.b (item 2.7.b) : « Oui, admis » — le
  repli neuf signalé par construction, les 24 lectures de capacité faites à la création d'un corps
  avant son premier mouvement et la remise à zéro d'équipement de fin de manche lue « utilisée »
  sont admis ; les deux derniers restent notés (découvertes 22 et 24). Même jour : « Oui, fusionne
  au vert » — le lot lint du décodeur (`feat/lint-decodeur`) se fusionne dans `feat/v75` juste après
  la vue A de la campagne, au vert.
- 2026-10-07 : « tu pourras fusionner si la CI est verte » — accord de fusion de 2.7.b dans
  `feat/v75`, sous condition d'une CI verte sur la tête fusionnée.
- 2026-10-07, en réponse à la mesure de 2.7.c (décision 4 de 2.7.c0) : « Lire sans la partie
  optionnelle (Recommandé) » — la vue A unique lit le message de kill (genre 85) sans sa queue ;
  killsource y prend ses kill-events ; sa recherche bit à bit ne reste qu'en rattrapage, sur les
  films dont la vue A ne se lit pas, comptée. Changement de grammaire, vérifié sur tout le corpus
  avant fusion (item 2.7.c4).
- 2026-10-07, en réponse à la vérification de 2.7.c4 sur 28 films (433 kills réels perdus sur quatre
  films anciens, 200 morts publiées sans assistant ni parts de dégâts, quand la recherche ne suivait
  qu'un arrêt de la lecture) : « Chercher aussi là (Recommandé) » — le rattrapage cherche dans toute
  trame dont la lecture de la vue A n'est pas établie (la vue B ne commence pas à sa fin : lecture
  arrêtée, ou terminateur que la marche ne retient pas), entre le premier genre de la vue A et le
  début de la vue B (la fin du payload quand la liste n'est pas localisée), sans rendre un message
  que la vue A a lu.

**Techniques** (une objection de l'utilisateur les rouvre) :
- **DT2-1 — Un distributeur.** `grammar.Distribuer(fc, canaux ...Canal)` marche UNE fois les deux
  phases et donne chaque paquet aux canaux ; un `Canal` déclare ses intérêts (archétype, index de
  composant, vues voulues) et ne lit jamais un octet (rapport §3.3). L'union des intérêts décide
  quels crochets de l'`Observation` la marche pose : l'interprétation reste pendant la marche (IR-8),
  donc la différence nulle.
- **DT2-2 — « Interprété » devient exact.** Une occurrence dont l'archétype et l'index sont dans
  l'union des intérêts et qui est traversée est marquée `EtatInterprete` ; l'état ne dépend plus de
  la seule capture de la trace (précision d'IR-4 du 2026-10-03).
- **DT2-3 — L'en-tête de la marche.** Les paramètres hors flux (`IDLowBits`, découpage MPP,
  disposition i0, `gate15`, octet `+0x818`) deviennent un en-tête résolu AVANT la marche, chacun avec
  sa provenance (lu, calibré, supposé) ; types dans `grammar/lecture` (sans logique, ratchet de
  l'étape 1), résolution dans la grammaire.
- **DT2-4 — La récupération est une couche.** Un record trouvé par une méthode heuristique est rendu
  avec `PreuveRecupere`, le nom de sa méthode et son compte au registre `facts/fallback` (ADR 0034
  D-10 amendé) ; il n'est jamais mêlé à ce que la marche a lu. Un compte publié qui dépend du nombre
  de passes (`repli_bande_bipede_comblee`, sommé par relevé) change quand les passes sont
  mutualisées : changement DÉCLARÉ au lot, pas une régression.
- **DT2-5 — Mesurer avant de mutualiser.** Le gain de 2.4 et 2.5 (20 à 40 % du temps de décodage,
  estimation de l'analyse) est mesuré AVANT d'écrire ces lots (étapes par `replay/observe.go`, pic par
  la sentinelle `filmproc`). Règle, sauf objection de l'utilisateur au GO : si les balayages ancrés et
  les balayages d'objets du monde pèsent ensemble plus de 10 % de la durée d'une cuisson, 2.4 et 2.5
  mutualisent ; sinon ils se réduisent au marquage des records récupérés (DT2-4), sans mutualisation.
- **DT2-7 — Où vivent les canaux.** Les canaux et le distributeur vivent dans la grammaire : `replay`
  et `decfilm` n'importent pas `grammar/lecture` (ratchet de l'étape 1, IR-9), et la façade `decfilm`
  ne grandit pas — une entrée de façade qui distribue remplace celles qu'elle couvre (plafond de
  `archlint/film_facade_surface_test.go` : 179 au 2026-10-03, à re-mesurer à l'entrée de chaque lot).
- **DT2-8 — Ce qui se compte se dit en données.** Les comptes de la couche de récupération vont dans
  les comptes de replis de la grammaire, versés par `replay` dans `coverage.fallbacks` (J8.7) ; les
  constats passent par `constat` (ADR 0034 D-4) ; aucune ligne de journal dans la grammaire.
- **DT2-6 — Les changements de comportement viennent en dernier**, chacun déclaré, prouvé au corpus
  et au banc de vérité, avec montée de `grammar.Rev` (et `killsource.Rev` pour 2.7 (c)) ; la
  recuisson est un geste de l'utilisateur.

## 3. Lots

Tous les lots hors 2.7 : différence nulle, ou comptes de replis déclarés d'avance. Les lots qui
changent la forme du paquet `grammar` régénèrent l'empreinte à RÉVISION CONSTANTE après la preuve
(`LEVELUP_UPDATE_GRAMMAR_REV=1 go test ./internal/games/halo_infinite/film/internal/grammar/ -run TestGrammarRevSuitLaGrammaire -update-grammar-rev`).

**Ordre.** 2.1 (en-tête compris), 2.6, puis 2.2 et 2.3 selon la coordination du §1.3, la mesure M, 2.4, 2.5,
2.7, 3.1, 3.2. Un lot qui attend la campagne (§1.3) est DIFFÉRÉ PAR CE PLAN (exception de la règle 1
du contrat) : le lot indépendant suivant passe devant, et le report s'écrit au journal avec sa
dépendance. Aucun autre réordonnancement.

### Lot 2.0 — En-tête de la marche — FUSIONNÉ DANS 2.1
*Décision d'exécution du 2026-10-03* : seul, l'en-tête n'a pas de consommateur de production à
différence nulle (règle 7 : un accesseur lu par les seuls tests est du code mort) ; son
consommateur est le distributeur, qui prend l'en-tête comme ENTRÉE de la marche (2.1.0), et son
unification est 2.7.c. Relu sur pièces le 2026-10-03 : `IDLowBits` = 13 n'est pas « en dur » mais la
valeur de l'image statique (`DAT_144706100` = 0x1FFF, catégorie 7 de `FUN_1406d3140`,
`varwidth.go`), que deux écrivains du jeu réécrivent au runtime — provenance PRÉSUMÉE ; la marche
des morts d'objet la CALIBRE de 10 à 15 (`object_deaths_calibrate.go`). `gate15` est un choix de
killsource (`pickGate15`) : il entre à 2.7.c ; l'octet `+0x818` reste un repli nommé du registre.

### Lot 2.1 — L'en-tête et le distributeur ; canaux des états de mouvement et du tir continu (taille M) — CLOS le 2026-10-03
*EN DEUX TEMPS, convenu avec la campagne le 2026-10-03 (même formule que le lot 1.3 de l'étape 1)* :
2.1.0, 2.1.1 et la marque « interprété » de 2.1.3 d'abord, en fichiers neufs (`grammar/lecture`,
`grammar/entete.go`, `grammar/distribuer*.go`) et dans les fichiers de la marche
(`marche_trames*.go`) ; 2.1.2 et le resserrage de T3 (2.1.3) après la fusion de la vague 1 dans
`feat/v75`, que la campagne signale (L4a touche `movement_states.go`, `tir_continu.go` et
`lecteur.go`). Les canaux déclarent leurs intérêts par PAIRES (archétype, composant), résolues dans le
registre du film : jamais une liste figée (la vague 1 ajoute des lecteurs), jamais un nom de composant
seul (« high-frequency » désigne deux tables de composant aux grammaires différentes, ti=3 et ti=4 :
règle D-89 de la campagne, « routage par archétype ou par table de composant, jamais par nom », gardée
par `ecs_dispatch_table_guard_test.go` qui arrive avec L8). Le lot 2.6, indépendant, est passé
devant ; les lots 2.2 et 2.3, qui consomment le distributeur, passent devant la fin de 2.1 dès que
2.1.1 est fait (règle d'ordre du §3 : le reste de 2.1 attend une dépendance du plan).
- [x] 2.1.0 L'en-tête (ex-2.0) : `lecture.Provenance` (relue, mesurée, présumée — la table du profil
      —, calibrée sur le film, imposée à la construction) et `lecture.Parametre[T]`, sans logique ;
      `grammar.EnTete` (`IDLowBits`, découpage MPP du format, découpage d'i0) résolu par le contexte ;
      la marche est construite DEPUIS l'en-tête, et le distributeur le rend aux canaux.
      *Fait* (`lecture/entete.go`, `grammar/entete.go`) : largeur de l'identifiant bas présumée (13,
      image statique), MPP du format présumé (la table, dérivation unique `mppDuFormat` que
      `InstallFilmFormatMPP` et `PreuveDImageCle` partagent), i0 imposé par l'appelant ou présumé par
      le catalogue — non résolu sinon : l'auto-détection reste à la demande, sans compter son repli
      avant qu'un balayage la demande. La marche des trames lit l'identifiant bas sous l'en-tête, la
      phase des images-clés pose son MPP ; découverte 6.
- [x] 2.1.1 `Canal`, `Distribuer` (DT2-1), dans des fichiers neufs `grammar/distribuer*.go`.
      *Fait* : une marche des deux phases (images-clés puis trames), chaque paquet à chaque canal ;
      intérêts par paires (archétype, composant) ; crochets posés par canal puis fondus dans
      l'observation de la marche des trames (un crochet posé deux fois est refusé) ; la phase des
      images-clés reste sous l'observation du contexte, comme aujourd'hui. Pas de consommateur de
      production avant le second temps (2.1.2) ; tests : phases égales à leurs itérateurs, crochets.
*Décisions d'exécution du second temps (2026-10-03, après la fusion de la vague 1 dans `feat/v75`,
`2393d7db7`, refusionnée en `cc395eb7f`)* — relu sur pièces : le balayage des états de mouvement
pilote lui-même la marche (`nouveauMarcheurDesTrames`, `parcourir`) et lit, outre la structure, trois
choses internes à la marche : le verdict de la vue C (`trameLue.lecture.verdict`, pris seulement
pour une trame marchée par classes de vue), la trace des records (`lecturesDeComposant` sur
`FrameRecord.Trace`, compte `VehicleTypePhysicsByWriterLaw` de la vague 1) et l'arène de la marche
pour ses crochets :
1. *Deux canaux des trames, une distribution.* Le balayage des états de mouvement et le collecteur
   du tir continu deviennent deux `CanalDesTrames` ; `ScanMarcheDesTrames` les distribue
   (`Distribuer`) et assemble `MarcheDesTrames` depuis les deux canaux et le bilan de la marche
   (liaisons, NEW refusés, replis d'anticipation et de début de liste). `movement_states.go` ne
   pilote plus la marche.
2. *Le tir continu prend son verdict au crochet* (`VueControleHook`), que la marche par classes de
   vue publie une fois par trame marchée et jamais ailleurs — exactement les trames dont le verdict
   est pris aujourd'hui. Le crochet joue pendant la marche, la trame arrive après : le collecteur
   garde le verdict reçu jusqu'à la clôture de la trame, qui l'efface.
3. *Les comptes du balayage se lisent dans la structure* : paquets et listes d'événements (`Debut`),
   records et désynchronisations du bipède (`Records`) ; `VehicleTypePhysicsByWriterLaw` par paire
   (archétype du record, `compVehicleTypePhysics`) : un record compte quand une de ses occurrences,
   traversée ou infranchissable, porte ce nom dans l'archétype du registre — la règle de
   `lecturesDeComposant`, qui perd son seul appelant de production et est retirée, avec la ligne
   MUTATION de son test désignant le nouveau site (accord de la campagne du 2026-10-03, à trois
   conditions : par paire et par la constante, mêmes occurrences prouvées par la passe, mutation
   rejouée rouge sur le nouveau site).
4. *Intérêts* : le canal des états interprète, sur ti=35 dans les trames, l'accroupi, la glissade,
   l'action de mobilité, la capacité active et la vitesse de translation (deux orthographes
   chacun) — ce que son crochet garde ; la posture et le contrôle d'unité, publiés par le même
   désérialiseur, ne sont pas interprétés. Un film dont l'archétype bipède ne déclare aucun des
   quatre états : ni crochet, ni intérêt (comme aujourd'hui, aucune lecture). Le tir continu
   n'interprète aucun composant (la vue C n'en est pas un).
5. *T3 resserré* (2.1.3) : le test de provenance distribue le canal de production, dont il double le
   crochet ; chaque lecture que le canal interprète cite une occurrence `EtatInterprete`, la posture
   et le contrôle d'unité une occurrence délimitée.
- [x] 2.1.2 États de mouvement (`movement_states*.go`) et tir continu (`tir_continu.go`) deviennent
      deux canaux ; `ScanMarcheDesTrames` les distribue ; `replay/film_scan_mouvement.go` inchangé
      dans ce qu'il publie.
      *Fait* : `movementStateScanner` et `collecteurTirContinu` sont deux `CanalDesTrames` ;
      `ScanMarcheDesTrames` les distribue (`Distribuer`) et ne pilote plus la marche (`marcher`
      retiré) ; les comptes de la marche (liaisons, NEW refusés, replis) arrivent par le bilan. Le
      tir continu prend le verdict au crochet et le clôt à la trame (`ouvrir` retiré, la clôture du
      canal finit et trie les rafales). `VehicleTypePhysicsByWriterLaw` se compte dans la structure,
      par paire (archétype du record, `compVehicleTypePhysics`, masque résolu au registre :
      `occurrencesDuComposant`) ; `lecturesDeComposant` retirée avec l'accord de la campagne, la
      ligne MUTATION de son test désigne le nouveau site, mutation rejouée rouge.
      `replay/film_scan_mouvement.go` inchangé. Le commentaire de garde d'absence disait « trois »
      états pour quatre testés : corrigé (règle 17).
- [x] 2.1.3 « Interprété » exact (DT2-2) ; T3 de l'étape 1 resserré : chaque lecture d'état cite
      l'occurrence marquée interprétée.
      *Premier temps fait* : la marque — interprétée quand un canal l'interprète et qu'elle est
      traversée ; la règle « valeur capturée par la trace » est retirée (aucun lecteur de cet état ;
      sans canal, rien n'est interprété) ; mutation jouée rouge. *Second temps fait* : T3 distribue
      les deux canaux de production doublés de témoins ; chaque lecture que le crochet garde (un
      état autre que la posture et le contrôle d'unité, sur un bipède) cite une occurrence
      `EtatInterprete`, les autres une occurrence délimitée — 23 456 lectures dont 23 427
      interprétées, 11 751 entrées de contrôle citées. Deux mutations jouées rouges (la vitesse
      retirée des intérêts du canal ; la posture ajoutée) ; l'accroupi n'est pas une cible de
      mutation, il n'est jamais traversé dans les trames de ces bobines (lot 2.2, décision 2).
- Gate : T4 (étapes `movementStates` et `continuousFire` de `replay-equiv`), killsource identique,
  T1/T3/T6 verts.
  *Passé* (passe `ri31` contre la référence fusionnée `ri30`) : `replay-equiv` 20/20 identiques
  et digests identiques, tous décodés depuis le film ; faits 20/20 (dont
  `VehicleTypePhysicsByWriterLaw`, condition de la campagne) et killsource 19/19 identiques à
  l'octet. G-film et archlint (T1, T3, T6 compris), `go vet` (avec et sans `research`),
  `golangci-lint` (0 problème) verts ; empreinte de la grammaire régénérée à révision constante.
  Parcours de la cuisson : inchangés (une phase des images-clés pour les préliminaires, une marche
  des trames). CI verte au niveau job sur la fusion (`2f8346d42`, run `37145047635`) et sur le
  commit du lot (`04208d803`, run `37146393269`).

### Lot 2.2 — Canaux d'image-clé (taille M) — coordination §1.3 — CLOS le 2026-10-03 (2.2.2 couvert par 2.7.d, décision de l'utilisateur du 2026-10-04)
*Décisions d'exécution du 2026-10-03* (relu sur pièces : chaque consommateur pilote aujourd'hui sa
propre boucle chunks -> paquets d'image-clé -> mémoire d'ancres ; seul `ScanPlayerTeams` parcourt
des corps, ceux de ti=9 ; coût mesuré de la phase complète des images-clés, mémoire chaude : 18 à
65 ms par film, contre 14 à 99 s de décodage, cinq films) :
1. *Un canal d'image-clé ne fait pas marcher les trames.* `Canal` = intérêts, image-clé, clôture ;
   `CanalDesTrames` y ajoute les crochets et la trame. Une distribution dont aucun canal ne lit les
   trames ne les marche pas.
2. *Les intérêts ont une phase.* `Interet.Phase` (trames par défaut, ou images-clés) : la phase des
   images-clés ne marque interprétées que les occurrences qu'un canal y lit. Correction du premier
   temps de 2.1, qui y marquait les intérêts d'un canal des trames alors qu'aucun crochet n'y reçoit
   de valeur (IR-4 : interprétée = un canal l'interprète).
3. *Un corps n'est parcouru que s'il est lu.* Dans une distribution, la phase des images-clés
   parcourt l'état complet des records des archétypes qu'un canal y interprète ; les autres records
   gardent leur identité, leur ancre et leur liaison (chaînée ou élue), sans composant, marqués
   `lecture.CorpsNonParcouru`. L'itérateur `ImagesCles` garde la marche complète (son consommateur,
   `KeyframeClosure`, mesure tous les corps). Le coût de chaque consommateur ne change pas : ceux qui
   lisent des identités ne parcourent aucun corps (comme aujourd'hui), les équipes parcourent ti=9.
4. *Sans registre, les ancres.* La marche d'ancres n'a pas besoin du registre : une phase des
   images-clés sans corps à parcourir rend ses ancres sur un film sans `chunk_00`, comme les
   consommateurs le font aujourd'hui (bobine historique `minifilm_000d5950`, goldens du rejeu). Le
   registre est exigé par un corps à parcourir et par la phase des trames.
5. *La marche d'ancres du paquet est exposée* (`MarcheDistribuee.Ancres` : ancres dans l'ordre de la
   marche, écartés, décisions) : la couverture des armes portées et la liaison la lisent sans remarcher.
6. *Pas de mutualisation entre consommateurs dans ce lot* : chaque point d'entrée public garde sa
   signature et devient une distribution à un canal ; le nombre de parcours d'images-clés de la
   cuisson ne change pas (critère 5), sauf dans la marche des trames, dont les deux préliminaires
   (table anticipée, liaison) partagent désormais une phase des images-clés au lieu de deux
   parcours. Mettre les canaux d'une cuisson dans une seule distribution est le lot 3.1 (un contexte
   par cuisson).
7. *2.2.2* : prévu — la fenêtre de 32 bits (armes portées, marque de portage) et les motifs des
   emprises d'inventaire deviennent des méthodes de récupération nommées au registre et comptées
   par film. RELU SUR PIÈCES AVANT D'ÉCRIRE (découverte 8) : la grammaire atteint ces composants
   dans l'état complet du bipède, et ces fenêtres décident donc devant une lecture disponible ;
   2.2.2 est statué `[!]`, décision de l'utilisateur demandée. La recherche exhaustive de l'en-tête
   exact de ti=9 (`player_entities_entetes.go`) n'est pas une fenêtre de valeur : elle prouve une
   absence, et ses doutes sont déjà comptés et publiés (`coverage.seats`).
- [x] 2.2.1 Armes portées (`keyframe_loadout.go`), inventaire (`inventory_decode.go`), marques de
      portage (`keyframe_carrier_mark.go`), équipes (`player_teams.go`), recensements et bandes
      (`world_object_census.go`, `slot_band_*.go`, `offline_biped_band.go`), générations vivantes
      (partie image-clé, `generations_vivantes.go`), table anticipée (`keyframe_anticipe.go`) et
      liaison (`keyframe_liaison.go`) consomment la phase `ImagesCles`.
      *Fait* : chaque lecteur est un canal de la phase des images-clés et son point d'entrée public
      une distribution à un canal (signature inchangée) : armes portées (`canalDesArmesPortees`, la
      couverture de la marche lue dans `MarcheDistribuee.Ancres`), inventaire (`canalDInventaire`),
      marques de portage (`canalDesMarquesDePortage`), équipes (`canalDesEquipes` : il interprète i0
      de ti=9, la phase parcourt les corps de ti=9 et eux seuls ; l'index se relit à l'étendue d'i0,
      sans seconde traversée), recensement (`canalDuRecensement`), bandes (`releveDesSlots`, partagé
      par la règle comblée, la règle observée et le recensement ; `releveDeLaBandeBipede` sur la phase
      restreinte aux chunks demandés), générations vivantes (`releveDesViesBipedes`), table anticipée
      (`canalDeLaTableAnticipee`). La marche des trames lit ses deux préliminaires (table anticipée,
      liaison avec la table de datums de chaque image-clé) dans UNE phase des images-clés
      (`marche_trames_preliminaires.go`) et pose la liaison chunk par chunk
      (`lierLesImagesClesDuChunk`). Formes instrument (`lierLeChunkAuMonde`,
      `TableAnticipee.AjouterChunk`) passées dans les tests ; garde-rail neuf
      `marche_images_cles_unique_test.go` (un seul pilotage des images-clés, sites restants nommés :
      découverte 7 ; mutation jouée rouge). Distributeur : décisions 1 à 5, tests neufs (corps lu
      seulement s'il est lu, phase des intérêts, film sans registre ; trois mutations jouées rouges).
- [~] 2.2.2 Les fenêtres lues bit à bit À L'INTÉRIEUR des images-clés (armes : fenêtre de 32 bits,
      inventaire : emprises) deviennent des méthodes de la couche de récupération (DT2-4), marquées.
      *Couvert par 2.7.d* (décision de l'utilisateur du 2026-10-04, option A ; découverte 8) : la
      grammaire atteint ces composants dans l'état complet du bipède ; les inscrire maintenant comme
      replis les déclarerait `devant_la_lecture`, que le cliquet `NbDevantLaLecture` interdit
      d'augmenter. 2.7.d les fait lire par la grammaire d'abord, et inscrit ce qui reste de
      l'heuristique « après la lecture ».
- Gate : T4 ; comptes de replis déclarés si la bande n'est plus relevée deux fois (rapport §1.6).
  *Passé* (passe `ri22a` contre la référence `ri21a`) : `replay-equiv` 20/20 identiques, tous
  décodés depuis le film (`depuis_les_faits=false` ×20) ; faits 20/20 et killsource 19/19 identiques
  à l'octet. La bande reste relevée deux fois (cache du contexte et positions) : aucun compte de
  repli ne change. G-film, archlint, G-race, `go vet` (avec et sans `research`), `golangci-lint`
  (0 problème) verts ; empreinte de la grammaire régénérée à révision constante. Parcours
  d'images-clés de la cuisson : un de moins (les deux préliminaires de la marche des trames
  partagent une phase), aucun corps parcouru de plus.

### Lot 2.3 — Canaux de tête de vue A (taille M) — coordination §1.3 — CLOS le 2026-10-03
*Décisions d'exécution du 2026-10-03* (relu sur pièces : six balayages de tête parcourent chacun
tous les paquets delta et relisent le préambule de 9 bits, `readPacketHead`, avant de décoder le
corps de leur événement ; la marche des trames lit déjà ce préambule, `PacketHeadEventType`, pour
décider de localiser la liste d'événements, sans le ranger dans la structure) :
1. *Les phases d'une distribution sont celles que ses canaux lisent.* `Canal` = intérêts et
   clôture ; `CanalDImageCle` (images-clés), `CanalDesTrames` (crochets et trames), `CanalDesTetes`
   (tête de chaque trame). La phase des images-clés tourne quand un canal la lit ou quand la marche
   des trames en a besoin (ses préliminaires) ; la marche des trames quand un canal des trames est
   là ; sinon, une PASSE DES TÊTES (aucun record marché, aucun registre) pour les canaux de tête.
2. *La tête se range dans la vue A* : la continuation puis, quand elle annonce un message, son genre
   (`consumeVueA`), vue arrêtée au premier corps — la forme que la marche par rangs donne déjà à une
   vue A lue depuis la tête. La marche complète la range aussi pour les paquets à liste localisée
   ou non localisée (vue A « non lue » aujourd'hui, alors qu'elle a lu la tête pour en décider).
3. *Les décodeurs partent de la tête rangée* : chaque canal de tête prend la continuation et le
   genre dans la structure ; une tête qui ne tient pas dans le payload (paquet d'un octet) garde la
   lecture tolérante d'aujourd'hui (`readPacketHead`, zéros au-delà du payload) : c'est 2.3.2.
4. *Pas de mutualisation entre canaux dans ce lot* (même règle que 2.2) : chaque point d'entrée
   public devient une distribution à un canal de tête. `ScanFireEvents`, `ScanZoomEvents` et
   `ScanTranslocatorTeleports` prennent le contexte du film au lieu du film, pour distribuer dans
   le contexte de la cuisson au lieu d'en ouvrir un second (cible de l'étape 3 : un contexte par
   cuisson).
- [x] 2.3.1 Tirs (36), translocations (117), lunette, ramassages, apparitions (103), événements de
      véhicule : la marche lit la tête de chaque paquet UNE fois (`event_list.go`) et la donne aux
      canaux (`fire_events.go`, `transloc_events.go`, `zoom_events.go`, `biped_pickups.go`,
      `equipment_spawn_events.go` ; les événements de véhicule vivent dans `event_list.go`).
      *Fait* : `CanalDesTetes` et la passe des têtes (`distribuer_tetes.go` : `rangerLaTete` range
      la continuation et le genre dans la vue A, `teteDe` la rend aux canaux) ; six canaux
      (`canalDesTirs`, `canalDesTeleportations`, `canalDeLaLunette`, `canalDesRamassages`,
      `canalDesApparitions`, `canalDesEvenementsDeVehicule`), chaque point d'entrée une distribution
      à un canal ; les décodeurs de corps partent du bit qui suit la tête (`eventPayloadStartBit`).
      La marche complète range la tête de chaque trame avant de la marcher et décide de localiser la
      liste sur elle (plus de seconde lecture du préambule). `PacketHeadEventType` reste la forme
      des instruments (fabrique de mini-films du rejeu, comptes par type). Tests : formes de la
      tête, passe des têtes égale à la tête de la marche complète, aucune image-clé ni record marché
      par la passe ; deux mutations jouées rouges.
- [x] 2.3.2 Les paquets dont la tête n'est pas lisible gardent la règle d'aujourd'hui.
      *Fait* : une tête qui ne tient pas dans le payload (paquet d'un octet : la vue A s'arrête sans
      genre) garde la lecture tolérante d'avant (`teteDuPayload`, zéros au-delà du payload) — c'est
      ce que comptaient les apparitions et les événements de véhicule sur ces paquets.
- Gate : T4.
  *Passé* (passe `ri23a` contre `ri22a`) : `replay-equiv` 20/20 identiques, tous décodés depuis le
  film ; faits 20/20 et killsource 19/19 identiques à l'octet. G-film, archlint, vet (avec et sans
  `research`), `golangci-lint` (0 problème) verts ; empreinte de la grammaire régénérée à révision
  constante. Parcours de la cuisson : inchangés (chaque lecteur de tête garde son parcours des
  trames, désormais sans relire la tête ni la marche) ; leur mise en commun est le lot 3.1. CI
  verte au niveau job sur `2aae5c7e1` (run `37137855861`).

### Mesure avant 2.4 et 2.5 (DT2-5) — taille S — CLOSE le 2026-10-03
- [x] M.1 Durées par étape et pic mémoire de la cuisson, trois témoins et un BTB, machine calme,
      binaires alternés ; part des balayages ancrés et des balayages d'objets du monde.
      *Fait* (machine calme, signal de la campagne) : binaire de la base de l'étape (`67c379fc1`)
      contre celui du lot 2.3 (`2aae5c7e1`), deux tours alternés sur les quatre films
      (`replay-equiv`, journal des étapes), trois tours de plus sur le BTB, puis une cuisson de
      chaque binaire sous profil de tas et une sous trace du ramasse-miettes (`replay-build`).
      **Durées** (moyennes) : `084a804d` 108,98 → 107,99 s (−0,9 %), `e5adf7b2` 43,70 → 43,84 s
      (+0,3 %), `60ae07c4` 31,54 → 31,31 s (−0,7 %), `11de8353` 37,91 → 37,76 s (−0,4 %).
      **Pic** : témoins 0,49 → 0,52, 0,49 → 0,50 et 0,48 → 0,44 Gio (les deux sens) ; BTB 1,01 →
      1,10 Gio sur les cinq paires alternées (+9 %), 1,05 → 1,09 sous profil de tas, 1,03 → 0,90
      sous trace (−13 %). La trace explique l'écart (découverte 9) : le pic se forme dans la
      dernière seconde de la cuisson et dépend du calage du cycle du ramasse-miettes ; pendant le
      décodage, le tas vivant des deux binaires est le même à ±5 Mo par fenêtre de cinq secondes,
      et le profil n'attribue aux lots que +267 Mo d'allocations sur 41,9 Go (+0,6 %). Aucune
      structure retenue : critère 4 tenu. **Étapes** (BTB, `replay-build`, sans le hachage du
      harnais) : cuisson 98,6 s, dont killsource 23,3 s et décodage 73,2 s ; balayages ancrés
      24,2 s (pont d'identité, positions par l'ancrage des bipèdes, porté par l'étape
      `translocations` : 8,9 s ; les huit passes du marcheur ancré : 15,3 s, ≈ 1,75 s chacune) ;
      balayages d'objets du monde 38,1 s (placements 6,0, pads 12,0, véhicules 16,8, projectiles
      3,3) ; ensemble 62,3 s, **63 % de la cuisson**. Témoins (`replay-equiv`) : objets du monde
      seuls 35 à 42 % de la cuisson ; avec les balayages ancrés, 68 à 76 % (borne haute : le
      harnais hache les positions dans l'étape qui les suit).
- [x] M.2 Proposition chiffrée à l'utilisateur : mutualiser (2.4, 2.5 complets) ou marquer seulement.
      *Fait* : les deux familles pèsent bien plus que le seuil de 10 % → **2.4 et 2.5 mutualisent**
      (règle de DT2-5 ; aucune objection de l'utilisateur au GO). À gagner sur le BTB : les huit
      passes du marcheur ancré refont chacune le même parcours (15,3 s), les quatre balayages
      d'objets du monde parcourent chacun le film (38,1 s). Proposition présentée à l'utilisateur
      au point d'étape du 2026-10-03 ; une objection la rouvre. 2.4 et 2.5 attendent la fusion de
      la vague 1 (§1.3).

### Lot 2.4 — Récupération ancrée mutualisée (taille L) — CLOS le 2026-10-04 (la marque et le registre couverts par 2.7.d, décision de l'utilisateur du 2026-10-04)
*Décisions d'exécution du 2026-10-03* — relu sur pièces et mesuré avant le code (instrument
`grammar/ancrage_partage_research_test.go`) : la cuisson ancre les records bipèdes NEUF fois avec
les mêmes paramètres (positions, puis changements d'arme, deltas d'inventaire, rangs de capacité,
changements d'équipement, camouflage, grappin, impulsions et charges de capacité). L'ancrage
(curseur bit à bit) coûte 0,6 à 1,8 s par passe sur quatre films ; au-delà de l'ancrage, chaque
balayage coûte 0 à 140 ms (les changements d'équipement 0,17 à 0,95 s, leur récupération gatée) ;
la marche de TOUS les corps ancrés jusqu'au bout de leur masque coûte 31 à 159 ms (240 000 à
488 000 records par film). Le gain est donc dans l'ancrage, pas dans les corps.
1. *Un ancrage par film, mémorisé dans le contexte.* Les records bipèdes ancrés sous les paramètres
   du contexte (`ChunkNumbers`, `BipedSlots`, `I0Layout`, générations vivantes datées par paquet) sont
   relevés une fois, rangés compacts (par paquet : chunk et rang ; par record : bit d'i0, slot,
   génération, masque d'au plus sept index), et les neuf lecteurs les parcourent dans l'ordre du flux.
   `walkDeltaBipedRecords` reste la primitive d'ancrage et n'a plus que la mémoire pour appelant ;
   le garde-rail « pas de dixième site » reste vert. La visée sans position (`offline_aim_only.go`,
   masque qui ne commence pas à i0) et la récupération gatée des équipements
   (`equipment_recovery.go`, fenêtres de saut) ancrent sous d'AUTRES prédicats : autres méthodes,
   autres records, une passe chacune — elles ne partagent pas cet ancrage. La bande `ti=40` des
   véhicules n'a qu'un lecteur : rien à mettre en commun.
2. *Les positions bipèdes lisent cet ancrage* quand la cuisson ne force ni chunks, ni découpage, ni
   générations (c'est le cas de la cuisson) : paramètres et suites ancrées prouvés égaux paquet par
   paquet sur les vingt films du corpus (2.4.2). Les positions des véhicules (bande `ti=40`, toutes
   générations) et les instruments qui forcent leurs options gardent l'ancrage du payload
   (`ScanBipedRecords`). La bande bipède n'est plus relevée deux fois : `repli_bande_bipede_comblee`
   compte une fois par film — changement DÉCLARÉ (DT2-4, ADR 0037 D-10), la seule différence
   attendue à la preuve (coverage et rapport de replis des faits).
3. *La marque « récupéré » et le compte au registre attendaient une décision de l'utilisateur —
   donnée le 2026-10-04 (option A) : ils passent à 2.7.d*
   (2.4.1, partie statuée `[!]`, découverte 10) : l'ancrage décide aujourd'hui DEVANT la lecture de la
   marche pour les records qu'elle lit — même question que 2.2.2.
4. *Les étendues de composants des records ancrés ne sont pas rangées* : aucun lecteur ne les lit —
   chaque balayage marche son record jusqu'à son composant, mesuré bon marché ci-dessus (règle 7 :
   une donnée sans lecteur est du code mort). Elles se rangeront avec leur premier lecteur.
- [~] 2.4.1 UN ancrage bipède par film (positions et les huit passes du marcheur ancré, véhicules
      compris), mémorisé ; records `PreuveRecupere`, méthode nommée, étendues de composants jusqu'au
      premier infranchissable (`delta_biped_walk.go`, `offline_biped*.go`, `ability_*.go`,
      `camo_state.go`, `grapple_state.go`, `held_weapon_changes.go`, `inventory_delta.go`,
      `equipment_changes.go`, `equipment_recovery.go`, `offline_aim_only.go`, `replay/film_scan.go`).
      Le garde-rail `delta_biped_walk_guard_test.go` (pas de dixième site d'ancrage) reste vert.
      *Fait* : l'ancrage unique mémorisé (`ancres_bipedes.go` : `FilmContext.ancresBipedes`,
      `parcourirLesAncresBipedes`), lu par les huit balayages et par les positions de la cuisson
      (`ancrageDuContexte`, `balayerLesPositions`, `positionsDesAncres`, `lireLaPosition` sortie de
      `ScanBipedRecords`) ; véhicules, visée sans position et récupération gatée : décision 1 ;
      `replay/film_scan.go` inchangé (mêmes points d'entrée). Garde-rail vert. *Couvert par 2.7.d* :
      la marque `PreuveRecupere`, la méthode nommée et le compte au registre (décision de
      l'utilisateur du 2026-10-04, option A ; découverte 10). Les étendues de composants ne sont pas
      rangées, faute de lecteur (décision 4).
- [x] 2.4.2 Égalité de bande (`fc.BipedSlots()` contre `bipedSlotBand` recalculée) et de découpage
      prouvée film par film AVANT la bascule.
      *Fait* (`ancrage_partage_research_test.go`, avant la bascule) : sur les vingt films du corpus
      d'équivalence, chunks, bande (36 à 256 slots), découpage et générations des positions égaux à
      ceux du contexte, et suites ancrées identiques paquet par paquet (11 130 à 79 550 paquets
      delta par film, aucun différent).
- [x] 2.4.3 Le harnais `FuzzFilmRecordReaders` couvre la couche de récupération (aucune panique,
      records récupérés bornés par les bits du payload), comme il couvre la marche depuis l'étape 1.
      *Fait* (`ancres_bipedes_fuzz_test.go`) : l'ancrage d'un payload quelconque sous la bande de
      tous les slots, le rangement compact de chaque record et sa relecture (égalité exigée), la
      lecture de sa position avec directions et vitalité ; bornes : i0 dans le payload, masque de
      deux à sept index, records sans chevauchement. Campagne de 45 s : 2,5 millions d'exécutions,
      aucune panique.
- Gate : T4 sur les données ; `coverage.fallbacks` déclaré ; banc `b.Loop` ; critère 4.
  *Passé* (passe `ri24a` contre `ri31`) : digests `replay-equiv` identiques sur toutes les étapes
  de données des vingt films, tous décodés depuis le film ; seule l'étape `artifact` diffère, sur
  seize films, et la seule différence de l'artefact est le compte déclaré de
  `repli_bande_bipede_comblee` (vérifié en entier sur deux films : 2 → 1, 4 → 2) ; les faits
  diffèrent sur les mêmes seize films par leur rapport de replis ; killsource 19/19 identique à
  l'octet. Banc `BenchmarkRecuperationAncree` (`b.Loop`, mini-bobine, dix paires alternées) :
  médiane 904 → 316 ms (−65 %). Critère 4 (machine calme, binaires alternés, deux tours) : durées
  de cuisson −13 à −16 % (BTB 112,9 → 98,2 s ; `e5adf7b2` 45,4 → 39,0 ; `60ae07c4` 32,2 → 27,0 ;
  `11de8353` 39,1 → 34,1), pics dans la dispersion (BTB 0,92 à 1,09 Gio des deux côtés) ; trace
  du ramasse-miettes : le tas vivant du décodage monte de 15 à 30 Mo (l'ancrage mémorisé, ≈ 11 Mo
  rangés sur le BTB), sans effet sur le pic de fin de cuisson. G-film, archlint, `go vet` (avec et
  sans `research`), `golangci-lint` (0 problème) verts ; empreinte de la grammaire régénérée à
  révision CONSTANTE : aucune donnée décodée ne change, seul le compte déclaré. Parcours de la
  cuisson : huit balayages bit à bit des trames delta de moins (neuf ancrages bipèdes → un), et un
  relevé de la bande bipède de moins. CI verte au niveau job sur `057c0cffd` (run `37150564223`).

### Lot 2.5 — Récupération des objets du monde (taille L) — fait, CI verte, fusionné dans `feat/v75` ; clôture à la mesure de durée à machine calme
*Décisions d'exécution du 2026-10-04* — relu sur pièces et mesuré avant le code (profil CPU d'une
cuisson du BTB et sonde temporaire des appels, retirée) : sur le BTB, les poses, les socles et les
projectiles font QUATRE balayages bit à bit de pistes d'objets du monde (`ScanWorldObjectsForBand`) :
équipement `ti=37` aux poses (3,9 s) PUIS aux socles (3,4 s, même bande, mêmes bornes, mêmes
largeurs), armes au sol `ti=42` (3,6 s), projectiles (3,8 s) ; et QUATRE marches de création
(`runCreationWalk`, environ 2 s chacune) : équipement aux poses PUIS aux socles, armes au sol,
véhicules `ti=40`. Chaque balayage teste chaque bit de chaque payload delta ; le préfixe, l'en-tête
et l'appartenance du slot à la bande font l'essentiel du coût.
1. *Les pistes des objets du monde se relèvent en UNE passe sur l'union des bandes, mémorisée dans
   le contexte.* Au premier balayage de pistes, le contexte relève celles des bandes à pistes de la
   cuisson (équipement, armes au sol, projectiles) et de la bande demandée : chaque payload n'est
   parcouru qu'une fois, chaque bande gardant SON curseur (un record accepté n'avance que le curseur
   de sa bande, comme le balayage d'une bande seule), d'où des pistes identiques par construction.
   Une demande suivante sur la même bande, aux mêmes bornes et aux mêmes largeurs, rend une copie
   de ce qui est relevé.
2. *Les créations multi-archétypes en UNE passe, mémorisées par leurs entrées.* À la première marche
   de création, le contexte marche ensemble les archétypes de création de la cuisson (équipement,
   armes au sol, véhicules), chacun avec SON curseur et sous le profil de balayage du moment
   (largeurs MPP comprises) ; une demande suivante ne réutilise une marche que si l'archétype, la
   bande, les bornes et le profil sont les mêmes — c'est le cas des socles et des véhicules, qui
   installent les largeurs que les poses ont lues ou calibrées. La calibration MPP reste un
   préliminaire des poses, jouée avant.
3. Aucun fichier de `replay` ne change : les points d'entrée restent ceux d'aujourd'hui.
4. *Marque « récupéré » et registre* : comme pour l'ancrage bipède, couverts par 2.7.d (décision de
   l'utilisateur du 2026-10-04) — ces balayages lisent des records que la marche des trames lit
   aussi quand elle les atteint.
- [x] 2.5.1 Créations multi-archétypes en une passe ; pistes sur l'union des bandes
      (`equipment_creation*.go`, `vehicle_creation.go`, `ground_weapon_creation.go`, `projectiles.go`,
      `equipment_placements.go`, `replay/build_ground_weapons.go`, `replay/build_vehicles.go`) ;
      calibration MPP gardée en préliminaire.
      *Fait* : `pistes_du_monde.go` (une passe sur l'union des bandes à pistes, un curseur par
      bande, la reconnaissance d'un record jugée une fois pour toutes les bandes qui portent son
      slot : `enteteDObjetDuMonde`, `masqueDObjetDuMonde`, `echantillonDuRecord`, que le balayage
      d'une bande seule partage) ; `creations_du_monde.go` (une passe des créations, un curseur par
      archétype, le début de l'en-tête NEW lu une fois : `archetypeDeLEnTeteNEW`, première étape de
      `matchWorldObjectNewHeaderIn`, qui reste la seule reconnaissance) ; mémoire dans le contexte
      (`recuperations.go` : ancrage bipède, pistes, créations), rendue en copie, les créations
      réutilisées sous la même clé seulement (archétype, bande, bornes, profil). La marche d'un
      archétype seul (`runCreationWalk`, `scanPayload`) est retirée : la passe la remplace pour
      tous ; la calibration MPP des poses reste jouée avant. `equipment_placements.go` et les
      fichiers de `replay` n'ont pas eu à changer (mêmes points d'entrée). Tests : pistes d'une
      passe égales au balayage de chaque bande seule (12 381 échantillons, trois bandes),
      créations d'une passe égales à l'oracle d'un archétype seul (66 créations), copies et clé de
      réutilisation ; harnais de fuzz étendu aux deux passes ; trois mutations jouées rouges.
      Garde-rail neuf `grammar/recuperations_ratchet_test.go` (relevé par la liste de livraison
      avant la fusion, règle 6 de CLAUDE.md) : les primitives de relevé de la couche de
      récupération (ancrage bipède, pistes, créations) n'ont que des appelants de production
      nommés, avec leur raison ; un balayage neuf qui referait le parcours au lieu de lire la
      mémoire du contexte rougit, sans quoi le nombre de parcours remonterait sans qu'aucun test
      de données ne le voie (critère 5). Il tient aussi l'ancrage de 2.4, dont le garde-rail
      existant ne tient que les copies de la boucle d'ancrage ; deux mutations jouées rouges.
- Gate : T4 ; critère 4.
  *T4 passé* (passe `ri25b`, code final, contre la référence du lot 2.4 `ri24a` ; une première passe
  `ri25a` avant le refactor des signatures demandé par le lint rendait déjà la même chose) :
  digests `replay-equiv` identiques sur les vingt films, tous décodés depuis le film ; faits 20/20
  et killsource 19/19 identiques à l'octet. Références d'équivalence re-figées sur ces sorties (le
  compte déclaré de 2.4 y entre, seize films, ligne `artifact` seule). Banc `BenchmarkObjetsDuMonde`
  (`b.Loop`, mini-bobine, dix paires alternées, machine chargée) : médiane 1 113 → 325 ms
  (−71 %). G-film, archlint, `go vet` (avec et sans `research`), `golangci-lint` (0 problème)
  verts ; empreinte de la grammaire régénérée à révision constante. Parcours de la cuisson : quatre
  balayages bit à bit de pistes → un, quatre marches de création → une.
  *Critère 4 : EN ATTENTE d'une machine calme* — la vague 2 de la campagne compile et décode par
  intermittence pendant plusieurs heures ; elle signalera la fin. Sous charge, non conclusif : une
  cuisson du BTB par binaire (sans alternance) donne 76,0 → 60,2 s ; trace du ramasse-miettes : tas
  vivant du décodage en médiane 226 → 236 Mo (la mémoire des pistes et des créations), pic de fin de
  cuisson dans la dispersion connue (découverte 9).
  CI verte au niveau job sur `1b94fad1b` (run `37193642578`, garde-rail compris) ; fusionné dans
  `feat/v75` avec les lots 2.1 à 2.6 le 2026-10-04 (journal).

### Lot 2.6 — Le statborg descend dans la grammaire (taille M) — CLOS le 2026-10-03
- [x] 2.6.1 La lecture du statborg (`facts/objectives/statborg.go`, `film.go`, `extract.go`) devient
      une méthode de récupération de la grammaire ; `objectives` consomme ; les faits ne lisent plus
      d'octet (ADR 0034 D-2 amendé).
      *Fait* : le statborg, le pied de film et les rafales de capture sont lus par
      `grammar/signaux` (`LireLeStatborg`, `FooterEvents`, `CaptureBurstTimes`), une FEUILLE de
      l'arbre de la grammaire (découverte 1) ; `objectives` garde ses points d'entrée, porte les deux
      replis du statborg dans `ComptesDesReplis` et rend ses constats sous les mêmes codes ; `film.go`
      supprimé (sa doc de paquet passe dans `doc.go`). Ce que la méthode rend est un type à part
      (`types.StatRecord`), jamais mêlé à la structure : la marque `PreuveRecupere` n'a pas d'objet
      tant que ces records n'y entrent pas, et les deux replis restent nommés et comptés au registre
      (sites déplacés). Garde-rail neuf `archlint/film_faits_sans_octets_test.go` (exception datée :
      `killsource`, retrait en 2.7.c ; trois mutations jouées rouges). Conséquence de révision écrite
      (ADR 0037 D-6, SYNC_GUIDE EN et FR) : `objectives` entre `grammar` par sa valeur.
- Gate : T4 ; killsource inchangé ; `archlint` (extraction de bits hors `source`).
  *Passé* : `replay-equiv` 20/20 identiques, tous décodés depuis le film
  (`depuis_les_faits=false` ×20) ; faits 20/20 et killsource 19/19 identiques à l'octet à la
  passe de référence (même code que la base) ; archlint, G-film, `go vet` (avec et sans
  `research`), `golangci-lint` (0 problème) verts ; empreintes de `grammar` et d'`objectives`
  régénérées à révision constante.

### Lot 2.7 — Changements de comportement déclarés (taille L) — coordination §1.3
*Mesure avant 2.7.a, 2026-10-04* (instrument `grammar/morts_marche_unique_research_test.go`, huit
films du corpus à véhicules, contexte posé comme la cuisson : profil calibré de killsource —
génération stricte — et largeurs MPP des véhicules). Une première mesure sans le profil de la
cuisson était FAUSSE (générations lues contre celles du recensement : 1 mort sur 56 dans une vie) ;
refaite sous le bon profil, toutes les morts confirmées le sont à moins d'une minute de la fin de
leur vie au recensement des images-clés. Morts de véhicule (`ti=40`), records : marche à huit vues
138 ; marche des trames de la cuisson 131 (communes 118 ; les 20 propres à la marche à huit vues
sont dans des listes d'événements que la marche des trames ne localise pas, 19 confirmées) ; la
même sous les largeurs MPP des véhicules 150 ; et avec la récupération des listes non localisées
(localisateur unique, ordre « signature puis largeur libre ») 165 — deux seulement manquent (toutes
deux confirmées), vingt-neuf de plus (vingt-cinq confirmées). Occupation : 543 → 688 lectures.
`IDLowBits` : la calibration de la marche des morts rend 13 sur les huit films, et 13 sur les 48
films à véhicules du parc local (faits de cuisson), jamais le cadre par défaut.
*Décisions d'exécution du 2026-10-04* :
1. *Les morts d'objet et l'occupation deviennent un canal de la marche des trames* ([Distribuer],
   `ScanMarcheDesTrames`) : même récolte que la marche des morts (règle d'acceptation, dénominateurs,
   dédoublonnage), sur les records de la vue B de chaque trame. `ScanObjectDeaths` devient une
   projection de cette marche (instruments), comme `ScanMovementStates`.
2. *Les listes d'événements que la marche des trames ne localise pas sont récupérées pour ce seul
   canal* : début par le localisateur unique dans l'ordre des marches qui lisent les morts
   (signature, puis largeur libre), vue B lue sous le monde de la marche, rendu intact ; les autres
   canaux ne voient pas ces records. Compté au repli `repli_localisation_largeur_libre`, déjà inscrit
   « après la lecture ».
3. *RETIRÉE le 2026-10-04 (avant fusion), sur signalement de la campagne.* Elle faisait marcher les
   trames de la cuisson sous les largeurs MPP que les véhicules calibrent sur les poses des formats
   sans largeur relue (8/3) : c'est le lot LM de la campagne, MIS DE CÔTÉ par l'utilisateur le
   2026-10-02 (« corrections d'abord, uniquement générales lues dans le jeu » ; l'exception D6 est
   suspendue, plan de la campagne §3). La passe `ri27b` montrait pourtant la vue C fermée deux fois
   et demie à cinq fois plus souvent sur six de ces films (`084a804d` 4 837 → 23 642 paquets) : le fait est consigné
   (découverte 11), la décision reste celle de l'utilisateur. La marche des trames garde les
   largeurs du contexte, et les morts de véhicule s'y lisent, alors que la marche à huit vues les
   lisait sous les largeurs calibrées.
4. *`IDLowBits`* : la marche des trames garde l'en-tête (13, présumé, valeur statique de
   l'exécutable) ; la calibration de la marche des morts disparaît avec elle (13 partout où elle a
   tourné) et le repli `repli_cadre_de_marche_par_defaut_conserve` est retiré (aucun déclenchement) :
   un seul `IDLowBits` (IR-7).
5. *La marche à huit vues est retirée dans ce lot* (chronologie, calibration, déroulage à huit vues,
   site du localisateur) : sans appelant de production, elle serait du code mort (règle 7) ; 3.1.1
   garde la timeline de killsource. Les fichiers de la campagne qui la citent (le localisateur
   unique, son test, des instruments de recherche) sont mis à jour après l'avoir prévenue.
- [x] 2.7.a0 **Découpage MPP déclaré par le film** — prérequis de 2.7.a. Décision de
      l'utilisateur du 2026-10-05 (découverte 13). La campagne l'a confié à ce plan le même jour,
      sous trois conditions : son gate 2, la provenance présumée par mesure, sa double preuve comme
      oracle. Périmètre fermé :
      1. *profile* : la table des tailles d'état de création que l'exécutable courant déclare
         (`vtable+0x20`, getters relus le 2026-10-05) pour les archétypes dont l'état de création
         lit le bloc MPP : 35 → 0x98, 36 → 0x60, 37, 38 et 39 → 0x68, 40 → 0xb0, 41 → 0xd4,
         42 → 0xa8, 43 → 0x60. Règle sur la taille `n1` déclarée :
         - égale à la taille courante → découpage relu 9/5 ;
         - taille courante − 4 → 8/3, PRÉSUMÉ PAR MESURE (provenance écrite, la clé `n1` lue dans
           le film citée, liste gelée par un test) ;
         - autre valeur → inconnu.
      2. *grammar* : `n1` est lu dans la première image-clé du film, au premier record d'un
         archétype à bloc MPP, avant son état de création, et le résultat est mémorisé sur le
         contexte. `EnTete().MPP` et `MPPWidthsForFilm` rendent le découpage déclaré pour un format
         sans largeur relue. Un film dont le `n1` est inconnu garde le chemin actuel (calibration
         sur les poses) ; ce cas est compté.
      3. *Cuisson* :
         - le découpage du film est posé sur le contexte pour TOUTE la cuisson, après le profil
           calibré et la carte ;
         - les socles, les véhicules et les poses d'équipement le prennent, et la calibration
           redevient un contrôle ;
         - `repli_largeurs_mpp_calibrees_sur_le_film` ne se déclenche plus que pour un `n1`
           inconnu (le registre suit).
      4. *Contrôle par record* : tous les records de la clé de l'image-clé qui décide doivent
         déclarer le même découpage. Un seul discordant, et le film ne déclare rien : chemin
         actuel, avertissement par film. *Précisé à l'écriture* : le contrôle porte sur cette
         image-clé, pas sur chaque marche d'image-clé. Les records d'un film viennent d'un seul
         écrivain, et un contrôle dans les marches toucherait l'en-tête que la preuve des ancres
         et killsource lisent.
      5. *killsource inchangé* : son profil garde `MPPParDefaut`. L'écart est déclaré et
         l'alignement se fait en 2.7.c. La déclaration n'entre ni dans l'en-tête de la marche
         ni dans la preuve des ancres d'image-clé.
      6. *Outil du gate 2* : drapeau `-mpp-declare` dans `cmd_fermeture` (outil de recherche),
         ajouté à la demande de la campagne. Il pose `ResolutionMPP` sur chaque film et
         journalise le découpage et sa provenance (`mpp_declare.tsv`).
      *Écrit* (`3ee8e7bf2`, local) :
      - `profile/mpp_declare.go` et son test ; la table du profil et son catalogue commis (deux
        lignes sous la clé `format=20,21,24,25`) ; `profile-2026-10-05` ;
      - `grammar/mpp_declare.go` (`DeclarationMPP`, `ResolutionMPP`) et son test sur les sept
        bobines : 8/3 déclaré sans discordance sur les cinq anciennes (84 à 247 records de la clé),
        9/5 sur les deux du format 27 ;
      - `Relue` → `Decide` ; `grammar-2026-10-05` ;
      - cuisson : `replay.poserLeDecoupageMPPDuFilm`, et `gwWidthsForFilm` et les poses par
        `ResolutionMPP` ;
      - registre du repli calibré, ADR 0037 IR-7 ;
      - killsource et objectives : empreintes recopiées à révision constante ;
      - fixtures de contrat : seule la télémétrie des révisions change.
      Gates :
      - tests de la règle et sur les sept bobines (8/3 sur les cinq anciennes, 9/5 sur les deux
        récentes, aucun record contredit) ;
      - `go test` des paquets touchés ;
      - preuve d'équivalence : format 27 identique, formats anciens classés ;
      - gate de corpus contre `feat/v75` : aucun film en baisse ;
      - gate 2 de la carte v2 de la campagne (`cmd_fermeture -mode v2 -denominateur-fixe`) : aucun
        film en baisse, gagnés et perdus, part de factices ;
      - killsource à l'octet (`KILLSOURCE_FIXTURES`) ;
      - `grammar.Rev`, ADR 0037 IR-7 amendé.
      *Clos le 2026-10-05.* Gates joués (journal du même jour) ; baisses du gate de corpus admises
      par l'utilisateur. Double preuve de la campagne (oracle, 114 films anciens du parc) :
      identités lues contre le catalogue des tags installés, 1 480 366 records d'image-clé sur
      1 480 374 connus sous 8/3, aucun sous 9/5, miroir exact sur le format 27 ; fermeture des
      formats 24 et 25, 8/3 meilleure largeur sur 106 films sur 107, paquets sains 19-26 % →
      67-68 % ; formats 20 et 21 muets (aucune largeur ne ferme) : leur champ d'index repose sur
      `n1` seul. Réserves du vérificateur :
      1. `b429a7d3` (format 24, parc) : `n1` déclare 8/3 (232 records, aucun discordant) ; paquets
         sains 5 560 → 5 558, records utiles sains 50 → 50. Les quatre perdus (18:606, 19:1046,
         23:606, 33:1058) étaient de faux sains de 9/5 : chacun un seul NEW d'archétype MPP qui
         ouvre la liste par la recherche « fermeture », identité inconnue du catalogue installé
         (`10830ea5`, `82ebff11`, `3182036d`, `be63fce4`), champ de tête à 0x1c0 ou 0x19b (bits 7
         et 8, jamais écrits) ; sous 8/3 la liste n'est plus localisée. Instruit.
      2. Part contredite des fermés au bit, 2,4 % contre 0,6 % au témoin (« masque au-delà de
         l'archétype ») : famille du lot LR de la campagne, qui ne tient que sous le découpage
         déclaré. Hors de ce plan.
      3. Ratchet de fermeture d'image-clé : il mesure les bobines sous le découpage du format (9/5
         pour les anciennes), donc ne bouge pas. Rejoué sous le découpage déclaré : cinq archétypes
         objet 932 → 1 039 sur 22 319 ; baisses par ligne sur `ti=42` (`a521164d` 24 → 12,
         `60ae07c4` 5 → 3, `e5adf7b2` 13 → 1), `ti=38` de `a521164d` (122 → 121), `ti=37` de
         `e5adf7b2` (1 → 0). Aligné en 2.7.c (accord de la campagne).
      4. killsource lit `MPPParDefaut` (D-108 de la campagne), deux découpages sur les formats
         anciens : 2.7.c, déjà prévu.
- [x] 2.7.a Morts d'objet sur le marcheur unique (huit vues → trois, monde unifié) — après LU.
      *Écrit* (décisions 1 à 5) : `grammar/canal_des_morts.go` (canal des trames et d'image-clé :
      récolte des records de la vue B, récupération des listes non localisées par [debutRecupere],
      monde rendu intact, sans observation) ; `ScanMarcheDesTramesAvec` le distribue avec les deux
      autres quand on lui demande les morts (la cuisson : calque des véhicules balayé, un film sans
      véhicule ne paie pas cette lecture), `ScanObjectDeaths` en est une projection ;
      `MarcheDistribuee` expose la marche aux
      canaux de la grammaire ; `object_deaths_march.go` et `object_deaths_calibrate.go` retirés,
      `ObjectDeathStats` sans les champs de la calibration ; cuisson : marche des trames sous les
      largeurs MPP du contexte (décision 3 retirée), morts et occupation posées sur le calque des
      véhicules depuis elle
      (`mortsDeVehicule`), étapes observées `vehicleDeaths` et `vehicleDeaths.stats` ; repli
      `repli_cadre_de_marche_par_defaut_conserve` retiré, sites de `repli_localisation_largeur_libre`
      déplacés ; localisateur unique (en-tête, site, affirmation fausse sur la génération stricte
      corrigée) et son test portés ; sondes portées (`c2_decor_ti40`, `ti40_marche_desync`,
      `m4b_monture`, `ti40_morts_alignement`) ou retirées (`campagne_bis3`, accord de la campagne ;
      l'instrument de la mesure) ; `grammar.Rev` = `grammar-2026-10-04` (goldens de `killsource` et
      d'`objectives` régénérés à révision constante) ; ADR 0037 amendé (IR-2, IR-6, IR-7).
      *Correctif de l'instruction* (`953401feb`) : un épisode lu se ferme sur la lecture suivante
      du MÊME OBJET occupant (slot ET génération), plus du slot seul — sur `4f77afc1`, une lecture
      fausse de la marche des trames (slot 737 en génération 3, attachée au bipède 618) fermait à
      8410 le trajet du conducteur 737, dont l'arme tire jusqu'à 9101. Test
      `TestUneLectureDUneAutreGenerationNeFermePasLEpisode`, rouge sans le correctif.
      *Primauté par joueur* (`e127e90fb`, décision de l'utilisateur du 2026-10-05,
      découverte 16) : un épisode de repli n'est plus écarté parce que le film lit le même
      joueur dans le même véhicule pendant une autre de ses vies. Test
      `TestLaPrimauteNommeLeJoueurPasLeCorps`, rouge sans la règle.
      *Clos le 2026-10-05, après le lot « vue A » V1 de la campagne* (option A de l'utilisateur) :
      fusion de `feat/v75` (`5bc1fd938`, puis `65c99b669`) dans la branche, canal des morts sur
      `listeAnnoncee`. Gate de corpus contre `5bc1fd938` (schéma 79) : les mêmes 315 lignes en
      baisse que le gate admis, valeurs comprises ; banc `ok` sur les 19 témoins ; aucun FAUX.
      killsource sur films réels identique à `feat/v75`. La part (a) de la décision du 2026-10-05
      (listes que la marche ne localise pas) attend le lot « vue A » V2, qui localisera la vue B
      par la fin de la vue A.
- [x] 2.7.b Canaux delta lus par la marche là où elle couvre au moins autant que la recherche
      d'ancres, canal par canal, mesuré sur le corpus.
      *Mesure du 2026-10-06* (instrument `grammar/ri27b_canaux_marche_research_test.go`, les 20
      films du corpus d'équivalence, contexte de la cuisson : génération stricte, carte, découpage
      MPP résolu ; références re-figées sur `feat/v75` `fed1efed2`, `0e0143148`). Pour les neuf
      lecteurs ancrés, record par record (paquet, slot) et crochet par crochet, ce que l'ancrage lit
      contre ce que la marche des trames lit (ses records relus à l'étendue de leur trace, sous son
      cadre) :
      - quand les deux lisent le même record, la valeur est la même : 7 divergences sur 4,6 millions
        de positions communes, aucune sur les onze crochets ;
      - records bipèdes delta : ancrage 5 436 632, marche 5 088 890 (et 101 148 de plus par la
        récupération des listes non localisées du canal des morts), communs 4 615 828 ;
      - l'ancrage seul (820 804 records) ne porte presque jamais un slot que la marche a lu dans le
        même paquet (50) : ce sont d'autres records. Dans les trames fermées, 35 678, tous en deçà
        de la fin de la lecture de la marche sauf 68. *Lecture corrigée à l'instruction du gate
        (même jour)* : « en deçà de la fin » comptait aussi ce qui PRÉCÈDE un début de vue B
        localisé ; 35 256 de ces records précèdent le début qu'a choisi la fermeture
        (`DebutParFermeture`), 185 seulement sont dans l'étendue lue — ce ne sont pas de fausses
        ancres (décision 9).
        Ailleurs : listes non localisées 313 606 (la récupération en relit 162 770 avec la marche,
        85 276 communs) ; trames refusées 376 550, dont 269 899 au-delà du dernier record de la
        marche ; queues opaques 17 458 ;
      - la marche seule (574 210 records) : dans les trames fermées 258 629 (records sans i0, que
        l'ancrage ne trouve pas), dans les trames refusées 270 813 ;
      - par crochet, la marche seule lit plus que l'ancrage l'arme portée (7 665 contre 1 855), les
        cartouches, les charges, l'équipement, les grenades et le rang ; moins le camouflage (49 301
        contre 53 538), les deux capacités, les munitions (74 658 contre 92 706) et les positions
        (4 619 747 contre 5 436 632, 85 %). L'écart est dans les listes non localisées et au-delà de
        la fin de la marche des trames refusées ;
      - fidélité : quand la trame part de sa tête, les crochets que la marche publie elle-même sont
        ceux de ses records ; quand le localisateur a cherché le début de la liste, ses essais en
        publient 5 à 40 fois plus (découverte 19).
      *Décisions d'exécution du 2026-10-06* :
      1. Les huit lecteurs à crochets (charges, impulsions, rangs, camouflage, grappin, arme portée,
         deltas d'inventaire, équipement) deviennent des canaux de la marche des trames. L'ancrage
         passe derrière elle (option A de l'utilisateur ; la part « avec 2.7.b » de 2.7.d) : il ne
         lit plus que les records dont la marche n'a lu aucun record du même slot dans le paquet,
         dans une trame qu'elle n'a pas fermée ; ses lectures se comptent au registre des replis,
         ordre « après la lecture ». Par lecteur, la couverture devient : communs, plus la marche
         seule, plus l'ancrage récupéré — au moins celle d'aujourd'hui, moins les fausses ancres des
         trames fermées.
      2. Les positions restent à l'ancrage dans ce lot : la marche seule en lit 85 %, elle
         n'accumule pas les positions (découverte 21) et leur lecteur a sa propre grammaire d'i0.
         Elles passent derrière la grammaire avec 2.7.d.
      3. Les essais de la marche ne publient plus aucun crochet de canal : la porte unique des états
         de mouvement couvre tous les crochets que les canaux de la marche posent, garde-rail
         compris. Préalable des canaux (découverte 19).
      4. Les lectures des records NEW bipèdes ne sont pas prises (découverte 20) : les lecteurs
         gardent leur sens, des deltas.
      5. Les lectures des deux sources se rangent dans l'ordre du flux (paquet, puis bit du record)
         avant le traitement de chaque lecteur, inchangé : la source change, pas la logique.
      *Décisions d'exécution ajoutées à l'instruction du gate (2026-10-06)* :
      6. Un corps mort n'agit plus : le record qui porte le dead-state d'une vie et ceux du même
         corps qui le suivent, jusqu'au record NEW qui recrée la génération, ne vont à aucun lecteur,
         d'une source ou de l'autre (ils décrivent le cadavre : emplacements vidés, équipement
         retiré). Ils se comptent.
      7. Une annonce n'est pas un changement : une émission d'arme portée qui répète la famille
         précédente de l'emplacement, ou qui annonce un emplacement vide sans occupant connu, est
         `Restated`, pas une prise ni un lâcher.
      8. La garde des générations vivantes datées (lot R2-bis), que l'ancrage applique à chaque
         en-tête, s'applique aussi aux records de la marche : aucun record de trame fermée ne la
         rate sur les 20 films ; 15 records de trames refusées ou à queue opaque, aux masques
         manifestement faux, sont écartés et comptés.
      9. Une trame fermée ne prouve sa liste qu'à partir de son début de vue B quand ce début a été
         LOCALISÉ (signature, chaîne de NEW de tête, fermeture) : le premier candidat d'où la marche
         ferme le paquet peut être au milieu de la liste. L'ancrage rend ce qui précède ce début,
         pour un slot que la marche n'a pas lu. Un début LU (la tête ; la fin de la vue A quand le
         lot V2 de la campagne arrive) prouve tout le paquet.
      *Écrit* (`a5232e3e5`, `4f4049ebd`, puis les décisions 8 et 9) : canal des lectures bipèdes
      (`grammar/canal_des_lectures_bipedes.go`, `grammar/lectures_bipedes.go`) — il recueille les
      publications des onze crochets des huit lecteurs, datées de la position du lecteur de la marche,
      les attribue trame par trame au composant du record bipède delta retenu dont l'étendue les
      porte, puis fait passer l'ancrage derrière (records marqués récupérés, repli
      `repli_ancrage_bipede_apres_la_marche` au registre, ordre « après la lecture ») et range tout
      dans l'ordre du flux ; les huit lecteurs (charges, impulsions, rangs, camouflage, grappin, arme
      portée, deltas d'inventaire, équipement et équipement d'unité) rejouent ces publications sur
      leurs crochets sans relire un bit ; porte unique des essais étendue aux douze crochets de canal
      (`grammar/porte_des_essais.go`, garde-rail) ; la cuisson distribue le canal avec les états de
      mouvement, et ses étapes depuis le film se réordonnent (monde, états de mouvement, portage,
      capacités, pont) ; ADR 0037 IR-6 et IR-8 amendés.
      *Gate de corpus contre `fed1efed2`, après les décisions 6 à 9* (19 témoins) : aucun oracle ne
      bouge (kills, morts, assistances, score personnel, équipes, vies, V-1, V-2, V-4 à V-8) ; aucune
      lecture d'un lecteur ne baisse contre la base, sauf les lâchers d'une arme inconnue (48 → 0,
      décision 7) ; les lectures montent partout (prises et échanges d'arme, rangs `i48`, charges,
      impulsions, grenades, équipement), les prises d'arme se lient à leur objet au sol (fins
      « vues » devenues « ramassées »), les récupérations gatées d'équipement deviennent inutiles
      (la marche lit ces records : mêmes valeurs), un portage de bombe se ferme à l'armement au lieu
      de rester ouvert jusqu'à la fin du film (`c75f33b8`), les épisodes de camouflage finissent à
      leur première lecture. FAUX du banc, instruits : (a) R-1 du repli neuf sur les 19 témoins, par
      construction (mécanisme D-L0-5 de la campagne) ; (b) R-1 de deux replis existants vus pour la
      première fois (`repli_lien_prise_arme_abandonne`, plus de prises dont certaines sans position
      d'acteur à ±250 ms ; `repli_rang_capacite_vie_elargie`, plus d'impulsions et de charges en bord
      de vie) ; (c) V-3 sur trois témoins : 24 lectures du rang de capacité 13 à 15 ms après la
      création du corps, avant son premier mouvement — la piste d'une vie ne part que du premier
      mouvement (découverte 22) ; (d) V-3 sur `51ebbc0f` : l'équipement retiré par le jeu en fin de
      manche, lu comme dépensé (découverte 24). Admis par l'utilisateur le 2026-10-06 (§2).
      *Reprise de `feat/v75` après la vue A V2 et V3 de la campagne* (`c16708f2c`, fusion de
      `2707fdb31`) : `grammar.Rev` = `grammar-2026-10-06.5` (la vue A prend `.4`) ; le canal des
      états de mouvement ne compte plus un paquet à début lu dans la vue A (`DebutParVueA`) comme
      localisé ni comme ouvert par un NEW de tête (constat P1 de la relecture de V2 et V3, correction
      convenue avec la campagne : décompte seul, format des faits inchangé,
      `TestUnDebutLuNEstNiLocaliseNiOuvertParUnNeuf`) ; un début lu dans la vue A prouve tout le paquet
      pour l'ancrage derrière la marche (`TestCeQueLaFermetureProuve`). Population des huit lecteurs
      sur les 20 films, même code avec et sans V2 et V3 : +806 records, −51 (32 annonces
      d'emplacement vide, quelques ancres démenties par une trame désormais fermée, une quinzaine de
      records perdus là où la marche depuis la fin de la vue A bute ; instruits au handoff de la
      campagne `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md`, §2.2). Gate de corpus contre
      `2707fdb31` (19 témoins) : les familles admises, à l'identique (mêmes V-3, mêmes replis vus pour
      la première fois, toutes les lectures en hausse hors des lâchers d'une arme inconnue) ; une
      lecture de capacité de plus non publiée faute de piste (`111fa685`, télémétrie).
      *Clos le 2026-10-07, sur `feat/v75` `b5c9489ef`* (lot lint, clôture de la campagne, lot
      « équipes source » de levelup-dc : SchemaDesFaits 6, schéma 80) : gate de corpus contre
      `b5c9489ef` (19 témoins) : les mêmes FAUX et les mêmes familles en baisse que le gate admis, à
      l'identique ; `KILLSOURCE_FIXTURES` vert ; passe de référence b6 (20 films décodés depuis le
      film, `depuis_les_faits=false`, faits mis de côté `film_facts_b6`) : références d'équivalence
      re-figées, elles prennent la vue A V2 et V3, le lot des équipes et 2.7.b ; killsource json sur
      les 19 témoins identique à la passe b1 sauf un compteur de diagnostic de calibration
      (`c75f33b8`, lectures par les deux voies 5 → 6) ; erreurs de passe d'avant ce lot seulement
      (vies sans identité de `a349fea8` et `50247b26`, mêmes comptes qu'en b1 ; équipe de bot non
      lue de `bcb6d393`, lot des équipes). `replay.SchemaVersion` reste 80. Accord de fusion de
      l'utilisateur du 2026-10-07 (CI verte).
- [x] 2.7.c killsource EN DERNIER : `runWalk`, timeline, calibration deviennent des canaux et des
      préliminaires de la même marche ; contexte partagé avec la cuisson (décision de l'utilisateur
      du 2026-10-03) ; `IDLowBits` unifié (IR-7). Le découpage MPP déclaré par le film (2.7.a0)
      gagne killsource et le ratchet de fermeture d'image-clé : golden régénéré, chaque baisse
      de `ti=42` justifiée record par record par la preuve 2 de la campagne (identité inconnue
      du catalogue installé sous 9/5, connue sous le découpage déclaré) ; une baisse qui ne se
      justifie pas ainsi s'instruit (condition de la campagne du 2026-10-05). La lecture en chaîne
      des événements de killsource (`facts/killsource/eventchain.go`, son propre portage de
      `FUN_14076a1c4`, qui lit la vue A message par message depuis chaque kill-event candidat)
      devient une lecture de la vue A unique de la grammaire (lot « vue A » de la campagne,
      relecture du 2026-10-05).
      *Relu sur pièces le 2026-10-07, à l'ouverture* (`feat/v75` `94ac8fd68` fusionné : `e470a2e3e`).
      killsource lit le film lui-même en sept endroits :
      - `world.go` : la timeline — le registre de `chunk_00`, les images-clés dans l'ordre du temps, le
        pré-chargement de la première déclaration de chaque slot, le balayage des ancres `ti=35` que la
        marche d'image-clé saute, les fenêtres de vie des slots bipèdes non déclarés ;
      - `walk.go` : sa marche des records. Le monde est restauré après chaque paquet (aucune liaison
        NEW portée), sur huit vues, sous `DefaultFrameConfig` (`IDLowBits` 13). Le début de la vue B
        vient de `DebutDeLaVueB` : fin de la vue A, puis signature, puis largeur libre. Seuls les
        records entièrement portés gardent leur dead-state, filtrés ensuite par crédibilité (bande
        bipède dérivée des images-clés, indices dans le roster, catégorie dans l'énumération) ;
      - `calibrate.go` : l'oracle de largeur d'axe et la décision de la largeur du mot de poignée
        (400 paquets sans événement, sous la timeline ; « non discriminée » sur les 19 témoins) ;
      - `eventchain.go` et `assist.go` : les kill-events (genre 85), cherchés à chaque bit et validés
        par une chaîne de trois messages, et le choix de `gate15` ;
      - `scan.go` et la sonde relâchée de `health.go` : le gabarit du dead-state cherché à chaque bit
        des paquets à événements ;
      - `feed.go` et `film_table.go` : les chunks HIGHLIGHT et `chunk_00`, par des fonctions de la
        grammaire ;
      - `botmeta.go` et `botmeta_equipe.go` : les paquets BOT_METADATA (seconde copie de la grammaire
        de la fiche joueur, signalée par levelup-dc), par le champ interne `payload`, que le
        garde-rail ne compte pas.
      Le critère de retrait de l'exception du garde-rail de la couche des faits
      (`archlint/film_faits_sans_octets_test.go` : « killsource devient un canal de la marche
      unique ; la ligne devient périmée ») demande donc que TOUT ce que killsource lit descende dans
      la grammaire. Le périmètre de 2.7.c est plus large que les trois éléments que l'item nomme
      (marche, timeline, calibration) : signalé à l'utilisateur.
      Lecture retenue de la décision du 2026-10-03 (« lot 2.7 (c) puis 3.1 ») : en 2.7.c, killsource
      ouvre son propre contexte de film, posé comme celui de la cuisson, et lit par la marche unique
      (changement de comportement, prouvé). En 3.1.1, la cuisson et killsource partagent un seul
      contexte et une seule marche (sans différence).
      Sous-items, dans l'ordre :
      - [x] 2.7.c0 *Mesure*, avant tout code de production branché. Corpus : les 20 films du corpus
            d'équivalence et les quatre films de référence de killsource (`KILLSOURCE_FIXTURES`).
            Contexte : celui de la cuisson (profil calibré, génération stricte, carte, découpage MPP
            résolu). Mesures :
            1. *dead-states bipèdes* : la marche de killsource contre la récolte de la marche des
               trames (canal des morts, listes non localisées récupérées comme le fait killsource),
               record par record (paquet, slot). On compte les communs (valeurs égales ou non), les
               propres à chaque côté par classe de trame (verdict, début de vue B), les crédibles des
               deux côtés, et on compare la bande bipède au filtre par archétype ;
            2. *effet sur le résultat* : le `Result` de killsource avec les candidats de la marche des
               trames à la place des siens (lignes publiées gagnées et perdues, par temps de
               l'hybride ; santé ; couverture) ;
            3. *calibration* : la décision du mot de poignée et l'oracle d'axe, sous la timeline et
               sous le monde des préliminaires de la marche (liaison des images-clés de chaque chunk) ;
            4. *découpage MPP déclaré* : killsource sous `ResolutionMPP` contre `MPPParDefaut`, sur
               les films anciens (dead-states, lignes publiées) ;
            5. *kill-events* : la chaîne de killsource contre la vue A unique (`lireLaVueA`). On
               compte les paquets dont la vue A est portée jusqu'à son terminateur, les genres 85
               lus, ce que la variante de partie du film décide de leur queue
               (`queueDuKillPossible`), et `gate15` contre la règle de la grammaire pour le genre 15.
            Gate : chiffres au journal, puis décisions d'exécution écrites ici avant le code.
            *Mesuré le 2026-10-07* (instruments `grammar/ri27c_killsource_research_test.go` et
            `facts/killsource/ri27c_research_test.go` ; sorties `scratchpad/ri/ri27c`), 23 films :
            1. *Dead-states* (records entièrement portés, décodage du format) : 3 030 communs à
               valeur égale, 2 à valeur différente, 172 propres à killsource (77 dans des trames
               que la marche lit, 72 dans des paquets sans événement, 20 récupérés, 3 non
               localisés), 635 propres à la marche des trames (dont 166 sur le BTB `1c4c63c2`).
               La bande de killsource (images-clés plus balayage des ancres `ti=35`) est polluée
               sur 9 films (bornes 128, 256, 4352, 7808, 8064 : de fausses ancres) ; celle de la
               phase des images-cles de la marche ne l'est sur aucun.
            2. *Résultat* : 3 453 lignes publiées, 3 440 couvertes, à l'identique dans les quatre
               variantes de la marche des trames (filtre de bande ou d'archétype, découpage du
               format ou déclaré, queues rompues acceptées ou non). Seule la voie technique change
               (`Read.Path`) : 272 lignes passent du balayage à la marche sous le filtre de bande,
               273 sous le filtre d'archétype, 519 sous le découpage déclaré (lignes de la marche
               2 506 → 2 937, du balayage 893 → 462).
            3. *Calibration* : la décision du mot de poignée est aveugle sur les 23 films, sous la
               timeline comme sous le monde des préliminaires (trois scores égaux) : l'invariant
               partout. L'oracle d'axe est plat sur 19 films, en désaccord sur 2 (oracle seul).
            4. *Découpage déclaré* : la marche et la calibration de killsource sous le découpage
               déclaré rendent le même résultat que sous le format sur les 22 films qui en
               déclarent un ; la marche des trames y lit plus (point 2).
            5. *Kill-events* : killsource en lit 3 552. La vue A unique atteint 2 280 messages de
               genre 85 et les refuse tous : la variante du film laisse la queue possible (19
               films), ou la vue A ne se lit pas au-delà de sa tête (4 films anciens). Des 3 552,
               2 163 sont le premier 85 que la vue A atteint, même bit, six champs égaux ; 1 280
               sont au-delà de son arrêt (un 85 plus tôt dans le paquet : 964 ; vue A illisible :
               316) ; 109 ailleurs. La vue A atteint 117 messages 85 que killsource n'a pas (sa
               chaîne de trois messages les refuse). Lue SANS queue, la vue A va jusqu'à son
               terminateur sur 2 267 des 2 280 paquets ; là où un localisateur indépendant
               (signature, chaîne, fermeture) a trouvé le début de la vue B d'un paquet fermé
               (1 529), la fin de la vue A tombe dessus au bit près 1 415 fois, avant lui 108 fois
               (32 rejoints par une chaîne de records lisibles), au-delà 3 fois. Ghidra : les deux
               réglages qui posent la queue (`kill_playback_enabled`, `play_of_the_game_enabled`,
               `FUN_140373a60` et `FUN_140373b40`) s'enregistrent à faux par défaut
               (`FUN_140ad2d08(…, 0)`), et le code ne fait que les lire.
            *Décisions d'exécution du 2026-10-07* :
            1. Les dead-states de killsource viennent de la marche des trames : la récolte du canal
               des morts, avec la trame et la position, sous la règle de qualité de killsource
               (records entièrement portés). Le filtre d'archétype (record lié comme bipède)
               remplace la bande ; la timeline (pré-chargement, balayage des ancres, fenêtres de
               vie) est retirée.
            2. La calibration devient un préliminaire du contexte du film, même règle, sous le
               monde des préliminaires ; aveugle partout, elle laisse l'invariant (aucune sortie ne
               change).
            3. Le contexte de killsource prend le découpage MPP déclaré, comme la cuisson.
            4. Les kill-events : décision de l'utilisateur du 2026-10-07 (§2) — la vue A unique lit
               le 85 sans queue, killsource y prend ses kill-events, sa recherche bit à bit ne
               reste qu'en rattrapage compté là où la vue A ne se lit pas (item c4).
            5. Le contenu publié ne change pas sur les 23 films ; le changement de voie technique se
               déclare, avec la montée de `killsource.Rev` en c2.
      - [x] 2.7.c1 *Lectures déplacées, sans différence* : `feed.go`, `film_table.go`,
            BOT_METADATA (repliées sur le lecteur de la grammaire de la fiche joueur si
            l'équivalence tient, sinon déplacées telles quelles), le gabarit du dead-state et sa
            sonde relâchée descendent dans la grammaire ; killsource consomme leurs résultats.
            Gate : killsource identique à l'octet (json des 19 témoins, `KILLSOURCE_FIXTURES`),
            `replay-equiv` identique.
            *Écrit le 2026-10-07* :
            - la table des joueurs : killsource prend celle de la grammaire
              (`grammar.ScanFilmPlayerTable`, la lecture de la cuisson) ; ses causes de refus
              deviennent celles de la grammaire, et sa copie de la traduction erreur → cause est
              retirée. Un tampon vide se lit « tronqué » (convention de la grammaire), plus « sans
              registre » : cas synthétique seulement, test mis à jour ;
            - le fil des kills : `grammar.FilDesKills` (même règle : le chunk qui porte le plus de
              kills) ; l'ancre du repli `repli_chunk_du_pied_par_argmax` suit au registre ;
            - le motif des xuid : `grammar.LecturesDuMotifDesXUID`, la boucle que
              `grammar.ScanPlayerIndices` partage désormais (`lecturesDuMotif`) ;
            - BOT_METADATA : `grammar.PaquetsBotMetadata` (`grammar/bot_metadata.go`) — l'instant
              de chaque paquet, son nombre de bots, le balayage des noms, et la grammaire de
              l'écrivain repliée sur le corps commun des fiches de joueur, dont la lecture du bloc
              de 44 octets rend désormais l'équipe et son jumeau (`lireLeBloc44`) ; killsource n'en
              garde que l'agrégat et le relevé des équipes ;
            - le gabarit du dead-state et sa sonde relâchée : `grammar.BalayerLesEtatsDeMort`
              (`grammar/etats_de_mort_balayes.go`), rangé par killsource dans l'ordre total de
              `f.t0` ; l'exception d'extraction datée `byteAtBit` est retirée (portée sur
              `source.OctetAuBit`, la même lecture).
            Révisions constantes (`grammar-2026-10-06.5`, `killsource-2026-09-27`) : empreintes
            recopiées, aucune sortie ne change par construction. `KILLSOURCE_FIXTURES` vert sur les
            quatre films de référence ; suite du film verte (`G-film`). Il reste à killsource, pour
            c2 à c4 : `chunks.go` (paquets et charge utile), `world.go`, `walk.go`, `calibrate.go`,
            `eventchain.go` et `assist.go`. Preuve d'équivalence (`replay-equiv` et killsource
            json) interrompue le 2026-10-07 à 11 h 10 pour laisser la machine à la recuisson du
            parc de levelup-d0 ; elle se rejoue sur `feat/v75` fusionnée (`48fce6602`).
            *Clos le 2026-10-07*, sur `c11f30159` (fusion de `feat/v75` `48fce6602`) : passe de référence
            c0 (binaires de `origin/feat/v75`, références d'équivalence re-figées : aucune ne bouge) puis
            passe c1 (binaires de la branche) — `replay-equiv` 20 films identiques
            (`depuis_les_faits=false`), faits identiques à l'octet sur les 20 films, killsource json
            identique sur les 19 témoins ; `KILLSOURCE_FIXTURES` vert ; suite du film et garde-rails
            verts.
      - [x] 2.7.c2 *La marche de killsource devient un canal de la marche des trames* : dead-states
            bipèdes avec leur trame et leur position, rendus par la grammaire ; timeline retirée ;
            calibration en préliminaire du contexte. Règles (filtre de bande ou d'archétype,
            qualité des records) fixées par les décisions de c0.
            *Écrit le 2026-10-07* :
            - les morts : `grammar.LireLesMortsDeLaMarche` (`grammar/morts_de_la_marche.go`)
              distribue le seul canal des morts sur la marche des trames du contexte de killsource ;
              sur demande, le canal garde chaque dead-state `Mort` avec sa trame (position du chunk,
              rang, horodatage), le premier bit de son composant, la qualité du record (entièrement
              porté ou non) et sa récupération. killsource (`walk.go`) garde les records propres
              d'archétype bipède, puis son filtre de crédibilité (indices dans le roster, catégorie
              dans l'énumération) ;
            - la timeline (`world.go` : registre, pré-chargement, balayage des ancres `ti=35`,
              fenêtres de vie) et la bande bipède disparaissent, avec le repli
              `repli_deadstate_hors_bande_bipede` (registre, nom, versement, `ReplisDuDecodage`) et
              la dernière exception de `archlint/no_recomputed_film_context_test.go` ;
            - la calibration : `grammar.FilmContext.ScoresDeCalibration`
              (`grammar/calibration_de_la_marche.go`) compte le critère de killsource sous le monde
              des préliminaires de la marche (`parcourirLesPreliminaires`, `marche_trames.go` : table
              anticipée puis liaison des images-clés de chaque chunk), le monde restauré après chaque
              essai ; la décision reste à killsource (`calibrate.go`) ;
            - les entrées de l'ancienne marche dans la grammaire (`DebutDeLaVueB`, `VueADuFilm`,
              `VueADuFilmSousCarte`) et le champ `calibration.VueA` sont retirés ; leurs tests passent
              sur la marche des trames (`debut_par_vue_a_test.go`, `vue_a_majeure_test.go`), le
              garde-rail de la lecture unique de la vue A n'a plus que deux appelants permis ;
            - les instruments de killsource bâtis sur l'ancienne marche sont retirés
              (`world_precision_test.go`, `vehicules_v10_deadstate_test.go`,
              `i0_poignee_score_research_test.go`, `ri27c_research_test.go`,
              `ri27c_calibration_research_test.go`) ; tests neufs de la grammaire
              (`morts_de_la_marche_test.go` : chaque mort se relit à sa position ; le critère de la
              calibration, rejoué, rend les mêmes scores) ;
            - `killsource.Rev` monte (`killsource-2026-10-07`, chronique ; le rang
              `killsource-2026-09-21` passe à l'archive) ; `grammar.Rev` reste `.6` (empreinte
              régénérée, complément de chronique) ; registre des reports : la partie killsource de la
              bande bipède est close.
      - [x] 2.7.c3 *Découpage MPP déclaré et `IDLowBits`* : le contexte de killsource prend le
            découpage du film comme la cuisson, et l'en-tête de la marche ; ratchet de fermeture
            d'image-clé régénéré sous le découpage déclaré, baisses de `ti=42` justifiées record par
            record (preuve 2 de la campagne) ou instruites.
            *Écrit le 2026-10-07* :
            - `killsource.poserLeProfil` (`decode.go`) pose, après le profil, le découpage que la
              grammaire résout pour le film (`ResolutionMPP`), comme `replay.poserLeDecoupageMPPDuFilm` :
              la marche ET les cadres de la calibration (`cadresDeCalibration`, sous le profil du
              contexte) lisent 8/3 sur les formats anciens ; un découpage non résolu se dit
              (`killsource.decoupage_mpp_non_resolu`, avertissement). Tests :
              `facts/killsource/decoupage_mpp_test.go` (bobine `e5adf7b2` : 8/3 posé ; mutation rouge) ;
            - l'en-tête de la marche : killsource le prend par la marche des trames (c2) ; les cadres
              de la calibration lisent l'identifiant bas de l'en-tête (`ScoresDeCalibration`) et non
              plus celui du cadre par défaut — même valeur (13, présumé), une seule provenance ;
            - le ratchet de fermeture d'image-clé mesure les bobines sous le même découpage
              (`bobines_memo_test.go`) : 7 lignes montent (+135), 5 descendent (−28), aucun total ne
              bouge. Preuve 2, record par record (instrument
              `grammar/ri27c3_fermeture_mpp_research_test.go`, catalogue des tags du jeu installé,
              260 414 tags) : 269 fermetures perdues et 376 gagnées (49 et 44 slots) ; sur CHACUNE,
              le mot de 32 bits du bloc MPP lu sous 9/5 est inconnu du catalogue, et lu sous 8/3 c'est
              un tag du groupe que l'archétype attend (`weap` pour `ti=42`, `eqip` pour 37, `bloc` ou
              `scen` pour 38). Sur les cinq bobines anciennes, aucun record de `ti=35` à 43 ne lit un
              tag attendu sous 9/5, 17 341 sur 17 379 sous 8/3. Aucune baisse à instruire ; golden
              régénéré avec son historique (découvertes 32 et 33) ;
            - commentaires devenus faux corrigés (`grammar/mpp_declare.go`,
              `replay/build_from_film.go`).
            *c2 et c3 clos le 2026-10-07* (preuve commune ; passe c23, binaires de l'arbre de travail,
            faits mis de côté, `depuis_les_faits=false`, contre la passe c1) :
            - killsource json sur les 19 témoins : contenu des 2 747 morts publiées identique (tout
              sauf la voie technique) ; couverture, publication et catalogue identiques ; 411 morts
              changent de voie technique (359 du balayage à la marche, 52 de la marche au balayage :
              marche 2 198 → 2 505, balayage 549 → 242) ; santé : verdicts inchangés, aucune alerte
              ni dégradation neuve, part inexpliquée en baisse sur deux films (`111fa685` 15,9 →
              12,4 %, `fb1a1a72` 7,9 → 7,3 %) ; calibration : la décision du mot de poignée reste
              l'invariant partout, l'oracle d'axe (diagnostic) reste plat, sa seule contradiction
              (`a521164d`) disparaît ;
            - `replay-equiv` sur les 20 films : 61 ou 62 étapes sur 64 identiques ; ne bougent que
              `killsource` (résultat, voies et calibration compris), `killRefs` sur 13 films (le
              décompte des voies que le document porte) et l'artefact. L'artefact, cuit aux deux
              codes et comparé par `replay-diff` sur `000d5950` et `64e8adfa`, ne bouge que dans sa
              couverture (4 et 8 mesures sur 942 et 1 008) : décompte des voies, ligne du repli
              retiré (et `repli_deadstate_indice_hors_roster` 1 → 2 sur `000d5950`), révision de
              killsource. Faits : différents sur les 20 films (killsource y est persisté, révision
              montée) ;
            - `KILLSOURCE_FIXTURES` (quatre films de référence) : goldens régénérés, ne bougent que
              des voies (`9b191a7f` 01:00 et `78919882` 09:18 passent du balayage à la marche, même
              ligne publiée), les décomptes par voie, les paquets à événements localisés (5 025 →
              5 027 et 5 713 → 5 714 : tous) et les scores de l'oracle d'axe ; la mini-bobine
              (`000d5950`) ne bouge que par le score de la poignée (68 → 63, décision inchangée) ;
            - suite du film, garde-rails, `go vet` (avec et sans `research`), `golangci-lint`
              (0 problème) : voir le journal.
      - [x] 2.7.c4 *Kill-events par la vue A unique* (décision de l'utilisateur du 2026-10-07) :
            la lecture de la vue A lit le genre 85 sans queue — la garde de l'écrivain tient ses deux
            réglages à leur défaut de l'exécutable, faux — et range ses messages de kill (position,
            champs) dans la structure ; killsource les y prend. Sa recherche bit à bit descend dans
            la grammaire comme rattrapage : seulement dans les trames dont la vue A ne se lit pas
            jusqu'à son terminateur, et seulement au-delà du bit où la lecture s'est arrêtée (règle
            amendée par la décision de l'utilisateur du 2026-10-07 après la vérification, §2 : toute
            trame dont la lecture de la vue A n'est pas établie, avant la vue B) ; elle se compte au
            registre des replis (ordre « après la lecture »). `grammar.Rev` monte.
            *Écrit le 2026-10-07* :
            - `chargeJoueurTue` lit le 85, partie fixe seule, quelle que soit la variante de partie
              (doc : les deux réglages enregistrés à faux par `FUN_140ad2d08(…, 0)`, valeur
              présumée) ; la variante ne décide plus que du 116 (`queueDuKillPossible` retiré) ;
            - la vue A range ses messages de kill : `lecture.VueA.Kills`, `lecture.MessageDeKill`
              (position du message, victime, tueur, assistant, deux parts, drapeau ; seize octets,
              gelés), recueillis par la lecture unique (`Lecteur.killLu`) ;
            - le rattrapage (`grammar/kills_rattrapes.go`) : la recherche bit à bit et la chaîne
              d'évènements de killsource descendent dans la grammaire (`chaine_d_evenements.go`,
              `chaine_d_evenements_corps.go` ; tables en fonctions, aucune variable de paquet de
              plus). Écrit d'abord borné aux trames dont la vue A s'arrête, à partir du genre du
              message qui l'arrête ; la vérification sur 28 films (ci-dessous) a montré 433 kills
              réels perdus sur quatre films anciens, derrière des terminateurs que la marche ne
              retient pas. Règle finale (décision de l'utilisateur du 2026-10-07, §2) : toute trame
              dont la vue B ne commence pas à la fin de la vue A lue, entre le premier genre de la
              vue A et le début de la vue B (la fin du payload sans liste localisée), sans rendre un
              message que la vue A a lu (`fenetreDuRattrapage`, le canal des kills lit la trame
              après sa marche). Test `TestLeRattrapageCherchePartoutOuLaLectureNEstPasEtablie`
              (cinq cas) ; mutations jouées, rouges : le rattrapage d'une trame dont la vue B commence
              à la fin de la vue A, la recherche au-delà du début de la vue B, un message de la vue A
              rendu deux fois. `gate15` tranché par film, par la même règle, à la première trame qui
              en a besoin. Repli neuf `repli_kill_rattrape_hors_vue_a` (registre, nom, versement,
              `ReplisDuDecodage.KillsRattrapes`) ; sites de `repli_chaine_evenement_code_non_modelise`
              déplacés. L'oracle de la chaîne et la preuve d'équivalence du curseur descendent avec
              elle, identiques à l'octet ;
            - killsource prend morts et kill-events dans une seule marche
              (`grammar.LireLaMarcheDeKillsource`, canal des kills, `kill_event.go`) ; les couples se
              résolvent après elle. `eventchain.go`, `eventbody.go` et la lecture des octets des
              paquets (`packet.payload`, `hasEvents`) sortent de killsource ;
            - killsource ne parcourt plus de paquet : `chunks.go` ne garde que l'origine des instants
              (le plus petit horodatage des paquets de réplication, la valeur du premier de l'ancien
              tri) et la version du kill-feed ; `packet`, le tri des paquets de réplication
              (`trierPaquetsT0` et son test) et `film.ms` sortent (code mort, `ms` signalé par le
              lint) ; les tests prennent les types de paquet de la grammaire ; l'en-tête de `chunks.go`
              ne cite plus `world.go` (retiré en c2). L'exception de killsource dans
              `archlint/film_faits_sans_octets_test.go` devient périmée, donc rouge : retirée ici (le
              gate de c4 l'exige ; part de 2.7.c5 avancée), la liste reste vide en cliquet. Elle ne
              tenait plus que par un faux positif : le champ de position `Chunk` d'un paquet. Mutation
              jouée (`src.Chunk(0)` dans `chunks.go`) : rouge ;
            - instrument `grammar/ri27c4_kills_research_test.go` : confronte, trame par trame, les
              kills de la vue A et du rattrapage à la recherche bit à bit d'avant (communs ; d'avant
              seuls, rangés par leur place dans la vue A ; neufs, avec la longueur de chaîne que la
              recherche d'avant exigeait) ;
            - `grammar.Rev` → `grammar-2026-10-06.7` (rang convenu avec levelup-5c, qui tient la
              série du 2026-10-07) ; `killsource-2026-10-07` garde son rang non publié (complément) ;
              `objectives` à révision constante (complément) ; fixtures de contrat : seule la révision
              de grammaire bouge sur les huit bobines.
            *Clos le 2026-10-07* (preuve : passe c4b, binaires de l'arbre de travail, faits mis de
            côté, contre la passe c23 ; une première passe sous la règle d'avant la décision a mesuré
            la perte qui l'a motivée) :
            - killsource json sur les 19 témoins : contenu des 2 747 morts publiées identique sauf
              deux qui gagnent un kill-event attaché (une sans assistant, une avec un assistant
              nommé, parts 64/35) ; 56 morts passent du balayage à la marche, 20 de la marche au
              balayage (marche 2 505 → 2 541) ; santé : seuls les décomptes par voie bougent,
              verdicts, couverture, publication et catalogue identiques. Sous la règle d'avant :
              200 morts perdaient leur kill-event sur quatre films ;
            - `replay-equiv` sur les 20 films : `killsource` et l'artefact bougent partout ;
              `killRefs` sur 10 films ; les étapes de la marche (états de mouvement sur 9 films,
              inventaire et tir continu sur 5, camouflage sur 3, morts de véhicule sur 2, armes
              tenues sur 1, statistiques de ces étapes sur 14 à 16) bougent avec la vue B des
              trames à kill, que la fin de la vue A fixe désormais. Contre la passe c4 (règle d'avant),
              seuls `killsource`, `killRefs` (4 films) et l'artefact diffèrent. Replis : la chaîne
              arrêtée sur un code non modélisé tombe de 9 697 à 1 110 sur `11de8353` (ordre de
              grandeur partout), `repli_kill_rattrape_hors_vue_a` compte de 0 à 339 par film,
              `repli_appariement_par_fenetre_temporelle` et `repli_couple_recolle_sur_le_voisin`
              reviennent aux comptes d'avant (ils montaient à 130 et 222 sous la règle d'avant) ou
              baissent ;
            - marche (instrument de trames, 28 films, contexte de killsource) : fermées 674 171 →
              698 688, refusées 156 170 → 131 798, records +133 000 ; découverte 34 pour les trames
              qui régressent ;
            - kill-events (`ri27c4_kills_research_test.go`, 28 films) : la vue A lit 3 047 messages
              (2 828 que la recherche d'avant trouvait, 219 qu'elle écartait : découverte 35) ; le
              rattrapage en retient 867, tous trouvés aussi par la recherche d'avant ; celle-ci en
              trouvait 563 de plus : 549 de bruit (chaîne courte : 361 derrière le terminateur d'une
              vue A retenue, 148 au-delà du début de la vue B, 40 dans une vue A retenue) et 14 réels
              dans une vue A retenue mais mal lue (découvertes 30 et 34) ;
            - artefacts cuits (`000d5950`, `64e8adfa`, `replay-diff`) : contre c23, couverture (19 et
              31 écarts sur 942 et 1 008 mesures : trames fermées +197 et +2 387, voies des morts,
              révision de grammaire) et une posture de sprint perdue sur `64e8adfa` ; contre c4 :
              identiques ;
            - `KILLSOURCE_FIXTURES` : goldens régénérés — `000d5950` une mort de la marche au
              balayage, `78919882` 28 (découverte 34), décomptes par voie et cumul ; la mini-bobine
              une mort de la marche au balayage. Comptes d'assistance (`assist_test.go`, mesures
              réécrites avec leur raison) : kill-events −1 à −7 par film (bruit), `fccc61cd` +1 mort
              attachée avec un assistant nommé (16 → 17, l'écart à l'API passe de 2 à 1) ;
            - suite du film (19 paquets), `archlint`, `go vet` (avec et sans `research`),
              `golangci-lint` (0 problème) : verts.
            Reporté à 2.7.c5, par dépendance d'ordre des fusions : `SchemaVersion` et
            `SchemaDesFaits` montent à la fusion de `feat/v75`, aux rangs libres à ce moment-là
            (85 et 10 annoncés) ; `grammar.Rev` s'y renumérote après les rangs des autres sessions.
      - [x] 2.7.c5 *Clôture* : exception du garde-rail retirée (périmée, donc rouge),
            `killsource.Rev` et `grammar.Rev`, ADR 0037 amendé, registre des replis
            (`repli_largeur_mot_de_poignee_inferee`, `repli_localisation_largeur_libre`,
            `repli_record_desynchronise_jete` : sites déplacés), doc de killsource. Gate de l'item.
            *Ajouté le 2026-10-07* : à la fusion de `feat/v75`, renumérotation des rangs pris
            entre-temps (levelup-d0 : `grammar-2026-10-07`, `SchemaDesFaits` 9, `SchemaVersion`
            84 ; levelup-5c : la série du 2026-10-07) — `grammar.Rev` au premier rang libre après
            eux ; `SchemaVersion` 85 avec son entrée de chronique (le contenu cuit change : la
            marche lit au-delà des messages de kill, 2.7.c4) ; `SchemaDesFaits` 10 (la section des
            kills des faits change de contenu : 2.7.c2 retire un compte de repli, 2.7.c4 en
            ajoute un). Les rangs dépendent de l'ordre des fusions : ils se posent à la fusion.
            *Écrit le 2026-10-07* :
            - exception du garde-rail des faits : `[~]` retirée en 2.7.c4 (le gate de c4 l'exigeait) ;
            - fusion de `feat/v75` `fad38a03c` (`c4116f88c`) : conflits des chroniques et goldens de
              révision, des formes des types et des fixtures de contrat résolus côté `feat/v75`, puis
              renumérotation convenue avec levelup-d0 et levelup-5c (qui fusionnera après) :
              `grammar.Rev` `grammar-2026-10-07.2` (écrit `.6.7` sur la branche, entrée de chronique
              renumérotée, les compléments c1-c2 regroupés devant elle) ; `SchemaVersion` 85 (entrée
              v85, justification de `TestStructureIsOptionalInDocument`, plafonds des deux fichiers de
              chronique montés de leur entrée, exception écrite) ; `SchemaDesFaits` 10 (section des
              kills) ; 83 et 8, réservés au lot, restent sans emploi. `killsource-2026-10-07` garde son
              rang ; sa chronique, condensée (une entrée pour c2 à c4), tient sous 500 lignes ;
              `objectives` à révision constante (complément, étape identique sur les 20 films) ;
              goldens de révision, d'assemblage, de forme et fixtures de contrat régénérés ;
            - ADR 0037 amendé : IR-6 (killsource canal de la marche, message de kill sans sa queue,
              règle du rattrapage et les deux décisions de l'utilisateur, la fin de vue A fausse
              retenue de la découverte 34), IR-7 (killsource et le ratchet sous le découpage déclaré),
              D-2 (la marche de killsource repliée, exception retirée, liste vide en cliquet) ;
            - registre des replis : les sites de `repli_largeur_mot_de_poignee_inferee`,
              `repli_localisation_largeur_libre` et `repli_record_desynchronise_jete` pointent déjà
              sur le code en place (`calibrate.go`, `walk.go`, `localisateur.go`,
              `canal_des_morts.go` ; `TestToutSiteDuRegistreExiste` vert) : rien à déplacer ;
            - doc de killsource (`doc.go`) : ce que le paquet lit — rien, la grammaire lit pour lui.
            *Gate de l'item, le 2026-10-07* :
            - `replay-corpus-gate` contre `feat/v75` `fad38a03c` (19 témoins, banc de vérité
              compris, 14 min) : 352 gains, 93 pertes, toutes déclarées (entrée v85) — voies des
              morts redistribuées entre marche et balayage (contenu publié identique) ; trous du tir
              continu fragmentés (séries en hausse, trous en baisse) ; postures aberrantes coupées
              (`084a804d` : une escalade de 168 s ; `11de8353` : sprints coupés à leur vraie fin) ;
              compteurs de la marche (impulsions −1 à −6, refus +1 à +2, liaisons oubliées +12) ;
              deux trajets de véhicule raccourcis : `e5adf7b2` finit 7 s avant la mort de son
              occupant, sans trace à pied entre les deux (8 tirs de véhicule détachés) — régression
              probable, découverte 34 suspecte, non instruite trame par trame ; `4f77afc1`
              (passager, 10 s), sens non instruit. Banc de vérité : `repli_kill_rattrape_hors_vue_a`
              nouveau (le rattrapage compté que la décision du 2026-10-07 a créé) et
              `repli_deadstate_indice_hors_roster` sur trois films (0 → 1 ou 2 : morts que la marche
              lit désormais, d'indice hors du roster, non publiées). Fusion accordée par
              l'utilisateur le 2026-10-07 (« Ok tu pourras fusionner »), donnée après le point d'étape
              qui annonçait le défaut rare des fins de liste fausses ;
            - `KILLSOURCE_FIXTURES` après la fusion : vert ; suite du film et `archlint` : verts ;
            - `make gate-push` sur `27d38aa0d` (après la refusion de `feat/v75` `0a9a327d7`, Tactique
              v2, sans conflit) : vert en 1 703 s (lint Go 0 problème, typecheck et lint du web sans
              erreur, baseline : tous les tests présents, aucun échec) ; vitest du web vert (fixtures
              de contrat) ;
            - troisième refusion, `feat/v75` `cdd642061` : levelup-d0 y a fusionné `SchemaVersion` 86
              (pousseur et colline par le nom, publication seule) avant ce lot ; le lot passe donc à
              `SchemaVersion` 87 (entrée v87 de la chronique, justification et plafonds ; 83 et 85,
              réservés au lot, restent sans emploi), goldens d'assemblage et de forme et fixtures de
              contrat régénérés à 87 ; suite du film et `archlint` verts ;
            - CI de `ff2b1c969` : tout vert sauf le lint (`goconst` : la fusion réunit quatre
              `DatePose` « 2026-10-07 », le rattrapage des kills et trois replis du rejeu des zones
              par le nom) ; constante `date1007` posée dans `registre.go` comme `date0927`
              (`0e65358bf`), aucune valeur ne change ;
            - quatrième fusion de `feat/v75` (`92c6a5d6a`, levelup-d0 : désignateur de colline et
              premier contact par le nom, sans révision ni schéma, `SchemaVersion` 87 tient) : seul
              conflit, les `DatePose` du registre des replis des objectifs ; la constante
              `dateNomsDeProprietes` de levelup-d0 et `date1007` deviennent la seule `date1007`
              (convenu entre les deux sessions), cinq entrées la portent ; vet du film et
              d'`archlint`, tests du registre, d'`archlint`, du rejeu et des armes verts ; CI de la
              tête fusionnée verte.
- [ ] 2.7.d Les lectures heuristiques qui décident devant la lecture de la grammaire passent derrière
      elle (décision de l'utilisateur du 2026-10-04, option A ; découvertes 8 et 10) : les fenêtres de
      bits des images-clés (armes portées, marque de portage, inventaire) cèdent la place à la
      lecture de l'état complet du bipède par la grammaire (intérêts de la phase des images-clés) ;
      l'ancrage d'en-tête bipède ne lit plus que les records que la marche ne lit pas (avec 2.7.b) ;
      même principe pour les pistes et les créations d'objets du monde (2.5, décision 4). Ce qui
      reste de chaque heuristique s'inscrit au registre « après la lecture », compté, ses records
      marqués `PreuveRecupere` avec leur méthode (DT2-4) ; le cliquet `NbDevantLaLecture` ne monte
      pas.
      *Ouvert le 2026-10-07 (soir), sur `feat/v75` `312073cd3`* (accord de l'utilisateur : « vas-y tu
      peux le faire »). *État au 2026-10-08 : d0, d2, d3, d4 clos ; d1 `[!]`, décision de
      l'utilisateur demandée — l'item reste ouvert par elle.* Sous-items, dans l'ordre :
      - [x] 2.7.d0 *Mesure*, avant tout code de production. Corpus : les 28 films de la mesure de
            2.7.c4 (20 d'équivalence et 8 de killsource), contexte de la cuisson (profil calibré,
            génération stricte, carte, découpage MPP déclaré). Trois instruments, aucun fichier de
            production touché : `grammar/ri27d0_images_cles_research_test.go`,
            `grammar/ri27d0_positions_research_test.go`, `grammar/ri27d0_monde_research_test.go`
            (sorties `scratchpad/ri/d0`).
            1. *Images-clés* : pour chaque record bipède, les composants i22, i30 à i48 relus par la
               grammaire à l'étendue de leur occurrence, contre les fenêtres de la production dans
               l'emprise du même record. 10 710 records bipèdes : la traversée va au bout dans 9 849,
               mais 410 seulement se ferment sur le record suivant (3,8 %, preuve de la phase) ; les
               armes portées s'accordent (mêmes noms) dans 386 records ; la grammaire lit moins
               d'armes connues que la fenêtre dans 8 180 (emplacements vides `ffffffff` ou
               identifiants inconnus là où la fenêtre trouve des paires plausibles, MA40 et BR75) ;
               le compteur de grenades vaut autre chose que 4 dans 9 204 records, alors que
               l'écrivain l'écrit toujours à 4 ; l'écart entre l'i43 de la traversée et la première
               famille de la fenêtre se disperse (0 pour 334 records, puis -92, 128, -20, 94…), comme
               l'écart entre la fin de la traversée et le record suivant. Même dans les records
               fermés, 123 sur 410 rendent un compteur de grenades faux. La lecture de l'état complet
               du bipède aux images-clés n'est donc PAS disponible : découverte 36.
            2. *Positions* : la marche d'abord, l'ancrage derrière selon la règle du canal des
               lectures bipèdes de 2.7.b (i0 ajouté à ses composants), contre l'ancrage seul.
               6 987 554 positions communes, égales (90 % lues par la marche, 10 % par l'ancrage
               derrière), 12 différentes ; 4 307 de l'ancrage seul : 2 244 sur un slot que la marche
               lit comme un autre objet ou une autre vie (un objet `ti=20` sur un slot de la bande
               bipède, positions constantes de paquet en paquet : découverte 38), 1 687 d'un corps
               mort (règle 6 de 2.7.b), 375 dans l'étendue que la fermeture prouve (fausses ancres),
               1 sans i0 absolu ; 6 740 de la marche seule (5 323 dans des trames fermées, 1 417
               dans des trames non prouvées).
            3. *Objets du monde* : les records que la marche lit dans les bandes de la cuisson (DELTA
               à i0 déquantifiable, NEW des archétypes de création) contre la passe des pistes et la
               passe des créations, record par record. Pistes : communes au même bit 868 700
               (`ti=37`), 574 567 (`ti=42`), 358 749 (`ti=41`), 14 à bit différent ; de la passe
               seule, dans un paquet où la marche lit le slot, 173 698 (`ti=37`) et 459 960 (`ti=42`)
               — la marche y lit le MÊME record sous l'autre archétype : la passe donne un record à
               toutes les bandes qui portent son slot (découverte 37) ; dans l'étendue que la
               fermeture prouve, sans record de la marche pour ce slot, 118 918, 114 306 et 10 308
               (en-têtes fortuits : découverte 39) ; ailleurs (trames non prouvées, listes non
               localisées), 520 470, 505 513 et 112 870, que l'ancrage derrière la marche
               garderait ; de la marche seule, 1 750, 1 239 et 95. Créations : communes 10 064
               (`ti=37`), 10 188 (`ti=42`), 911 (`ti=40`) ; de la passe seule dans l'étendue prouvée
               7 565, 6 244 et 208 (découverte 39), ailleurs 7 688, 6 414 et 601 ; de la marche
               seule 5 337, 664 et 68 (4 129 de l'équipement dans des trames fermées).
            Gate : chiffres ci-dessus ; décisions d'exécution ci-dessous.
            *Décisions d'exécution du 2026-10-07* :
            1. *La marche désigne les records, les lecteurs existants les lisent à leur position.*
               Positions ([lireLaPosition]), échantillons de pistes ([echantillonDuRecord]) et
               créations ([equipCreationWalk.creationA]) gardent leurs lecteurs ; seule la source
               des positions de record change : les records de la marche d'abord, avec l'archétype
               que sa table d'entités leur donne (un record n'est jamais rendu à la bande d'un autre
               archétype), puis l'ancrage ou la passe derrière elle, selon la règle de 2.7.b : un
               slot que la marche n'a pas lu dans le paquet, hors de l'étendue que la fermeture de
               la trame prouve. Mêmes valeurs aux records communs, par construction.
            2. *Les positions bipèdes suivent les règles du canal des lectures bipèdes* : corps mort
               et générations refusées écartés (règles 6 et 8 de 2.7.b) ; une position ne se lit que
               d'un i0 absolu dans la région jouée (la grammaire d'i0 de l'ancrage). Le compte
               `repli_ancrage_bipede_apres_la_marche` compte désormais aussi les records qui ne
               portent qu'une position (changement déclaré).
            3. *Ce qui reste de la passe des pistes et de celle des créations s'inscrit au registre
               « après la lecture »*, compté par film, ses échantillons et créations marqués
               récupérés.
      - [!] 2.7.d1 *Images-clés : armes portées, marque de portage, inventaire.* Non exécutable tel
            qu'écrit : la grammaire ne lit pas l'état complet du bipède aux images-clés (mesure 1 de
            2.7.d0, découverte 36) ; retirer les fenêtres viderait les armes et les grenades des
            fiches. Décision de l'utilisateur demandée le 2026-10-07 (recommandation : un lot de
            grammaire d'abord, la lecture de l'état complet du bipède aux images-clés ; 2.7.d1 en
            dépend). Décision de l'utilisateur du 2026-10-08 (« Pour le point 2 met un workflow
            ultracode dessus ») : traité maintenant, lot de grammaire d'abord (lecture des
            images-clés sous la portée de l'état complet, lot LK de la campagne), puis 2.7.d1 ; plan
            dédié en préparation (journal du 2026-10-08, après-midi). Reste `[!]` jusqu'à sa clôture.
      - [x] 2.7.d2 *Positions derrière la marche* (décisions 1 et 2).
            *Décision d'exécution 4 (2026-10-07, relue sur pièces à l'écriture)* : la cuisson lisait
            les positions (étage du pont d'identité) AVANT la marche des trames, jouée par les états
            de mouvement après le monde. Lire les positions derrière la marche la ferait jouer deux
            fois (une marche des lectures bipèdes au pont, une seconde aux états de mouvement). La
            cuisson joue donc sa marche UNE fois, en tête (`filmScan.lireLaMarcheDesTrames`), et les
            états de mouvement prennent son résultat. Les morts d'objet s'y lisent quand les
            images-clés portent des slots de véhicule (`LecturesDeLaMarche.MortsSiVehicules`) : la
            première condition du calque des véhicules, qui ne se balaie qu'après ; leur coût mesuré
            sur quatre films (deux tours) est dans le bruit (0,68 contre 0,68 s ; 1,51 contre
            1,50 s ; 3,49 contre 3,53 s). Le collecteur de la synchronisation (pont d'identité, son
            propre contexte) joue désormais la marche des lectures bipèdes : coût déclaré, rendu par
            3.1.1 (un contexte par passe du collecteur).
            *Écrit* : `grammar/positions_lues.go` (records retenus par le canal des lectures bipèdes,
            i0 ajouté à ses intérêts ; fusion des deux sources dans l'ordre du flux ; lecture par
            `lireLaPosition` au bit d'i0 ; sans registre, l'ancrage seul), `i0AbsoluDeLaRegion` (la
            grammaire d'i0 de l'ancrage, partagée), le compte de
            `repli_ancrage_bipede_apres_la_marche` étendu aux records rendus pour leur seule position
            (registre mis à jour) ; `replay` joue la marche en tête. Tests : fusion dans l'ordre du
            flux, grammaire d'i0, positions de la mini-bobine (chaque position d'un record retenu,
            quanta égaux à ceux de l'ancrage aux records communs, le joueur du slot 529 que la bande
            des images-clés manque au chunk 5 lu par la marche ; mutation « ancrage seul » jouée
            rouge). Golden des familles de la mini-bobine : `bipedPositions` 28 004 → 29 000 (les 996
            positions du slot 529). Reste : preuve d'équivalence (seules les positions changent),
            puis 2.7.d3.
            *Instruction du premier gate de corpus (d2 et d3 contre `312073cd3`, 2026-10-08)* :
            deux règles corrigées dans ce sous-item, chacune née d'un FAUX du banc.
            1. *Un dead-state qui ne dit pas la mort ne tue pas le corps* (découverte 41) : la règle
               « un corps mort n'agit plus » de 2.7.b tuait le corps au premier dead-state, même quand
               il ne dit pas la mort (`Mort` à faux, lu dans une trame refusée : deux sur `a349fea8`,
               un sur `111fa685`). Sur `a349fea8`, le slot 590 perdait ses positions et ses lectures
               jusqu'à la fin du film (1 573 positions ; sept ramassages hors vie au banc, V-3 5 → 12).
               Elle exige désormais `Mort`, comme le canal des morts ; les huit lecteurs de 2.7.b en
               profitent aussi (changement déclaré).
            2. *La bande bipède reste la population des positions* (découverte 42) : la marche lit des
               corps qu'aucune image-clé ne porte — les corps posés en fin de match pour la scène des
               vainqueurs (`bfecd02b` slot 602, `c75f33b8` slot 592, `f75e7053` slot 601, en hauteur,
               dans la dernière minute), un joueur né au dernier chunk d'une bobine (slot 529 de la
               mini-bobine). Publiés en traces, ils donnaient des fins de vie sans identité (O-V2, +1
               sur cinq films) et des replis de génération inconnue. Les positions de la marche ne se
               retiennent que pour un slot de la bande bipède du contexte, la population de l'ancrage ;
               le golden de la mini-bobine revient à 28 004 et son test tient la population.
            3. *Dans une trame que sa fermeture ne prouve pas, l'archétype que la marche donne au slot
               ne l'est pas non plus* (découverte 45, instruite sur une perte de filet du gate contre
               `acfe4851a` : un trou de 5,6 s dans la trace d'un joueur de `bf15f7ab`, quatre tirs
               sans tireur). La marche lisait le joueur du slot 553 sous l'archétype de l'objet qui
               occupait son slot, dans des trames non fermées, et la règle de 2.7.b (un slot lu écarte
               l'en-tête ancré) écartait ses vraies positions. Dans une telle trame, seul un record
               de la marche du même archétype écarte désormais l'en-tête ancré ou l'enregistrement de
               la passe ([rendParLAncrage], deux cas ajoutés au test de la règle, deux mutations
               jouées rouges) ; la règle vaut aussi pour les huit lecteurs de 2.7.b (changement
               déclaré). Mini-bobine : positions inchangées, pistes de la bande des armes au sol
               51 → 54 (des fragments d'équipement dans des trames non prouvées, découverte 44).
      - [x] 2.7.d3 *Pistes et créations des objets du monde derrière la marche* (décisions 1 et 3).
            *Écrit* : `grammar/objets_du_monde_lus.go` — un canal de la marche des trames retient les
            records DELTA à i0 des archétypes à pistes et les records NEW des archétypes de création,
            avec l'archétype que la marche leur donne (même condition de trame que le canal des
            lectures bipèdes : lue et marchée par classes de vue) ; les pistes d'un archétype se lisent
            au bit d'i0 de ses records (`decodeWorldObjectPos`), ses créations à leur en-tête
            (`creationA`), puis la passe rend derrière la marche ce qu'elle n'a pas lu (règle de
            2.7.b, les trames du canal des lectures bipèdes, désormais gardées dans le contexte).
            `ScanWorldObjectsForBand` prend l'archétype de la bande (`ArchetypeDeBandeInconnu` : la
            passe seule, pour les enveloppes d'instrument qui balaient des bandes arbitraires) ;
            deux replis neufs au registre, ordre « après la lecture » :
            `repli_pistes_du_monde_apres_la_marche`, `repli_creations_du_monde_apres_la_marche`.
            Les comptes de création décrivent désormais les créations rendues (lues par la marche,
            ou rendues par la passe). Ratchet des appelants nommés mis à jour (le lecteur de
            création lu aux records que la marche désigne n'est pas un parcours bit à bit). Tests :
            le slot 1596 de la mini-bobine, dans les bandes des armes au sol et de l'équipement, lu
            comme un équipement par la marche, n'est plus rendu qu'à l'équipement ; la création
            fortuite du slot 1476 (paquet 4:576, trame prouvée depuis son début) ne se rend plus ;
            créations communes égales ; deux mutations jouées rouges. Golden des familles de la
            mini-bobine : projectiles 53 → 52 (une piste de cinq points, dont quatre dans des trames
            prouvées où la marche ne lit pas ce slot), armes au sol 46 → 42 (des équipements),
            créations d'équipement 38 → 36 (deux créations fortuites). Reste : preuve, puis 2.7.d4.
            *Instruction du premier gate de corpus (2026-10-08), lectures sur pièces des documents
            de base et de tête* : les objets que la tête ne publie plus sont des fantômes. Armes au
            sol perdues sur `4f77afc1` : sept objets « apparus » à z = −500 à −944 m, aux bords x de
            la carte ; projectiles perdus sur `fb1a1a72` : trente pistes immobiles à x = −230,77 m,
            souvent par paires au même instant ; véhicules : deux vies au châssis `00155903` (mot
            d'identité d'une création fortuite, sans famille ni trajectoire) disparaissent sur
            `084a804d`, et la vie du slot 829 y prend sa vraie naissance (le NEW que la marche lit au
            chunk 18, châssis `5b80c406`, un Ghost), si bien que le relais vers le slot 835 (même
            châssis, même point) se fusionne en un seul véhicule ; les bornes de la carte se
            resserrent quand un point aberrant disparaît (`084a804d` maxX 202,97 → 43,81 ;
            `50247b26` maxZ 77,13 → 30,82, un point isolé du slot 607). Sur `50247b26`, ce
            resserrement fait sortir des bornes un véhicule volant (slot 802, 236 images au-dessus
            de tous les joueurs) : le banc le compte en V-2 ; les bornes viennent des seules traces
            de joueurs (`boundsOf`), défaut de construction préexistant, hors de ce lot.
            *Après la fusion de `feat/v75` `acfe4851a` (arrêts de la vue B, 2026-10-08)* : plus de
            trames se ferment, donc plus d'en-têtes fortuits tombent dans une étendue prouvée.
            Mini-bobine : projectiles 52 → 51, pistes de la bande des armes au sol 42 → 51, puis 54
            avec la règle de la découverte 45 (découverte 44 : des vies d'équipement y gardent des
            fragments, qu'aucun objet publié ne lit ; le test du slot 1596 ne lui interdit plus que
            les trames prouvées), créations d'équipement 36 → 35 (slot 1465, chunk 2 paquet 1882), créations d'armes
            au sol 28 → 27 (slot 1529 génération 2, chunk 3 paquet 378, avant la vraie création de
            génération 1 du même slot au paquet 540) ; le test du point 6 de `ti=42` passe à 27
            créations et 21 références, mutation de la porte rejouée rouge. Le premier gate contre
            `acfe4851a` a révélé la découverte 43 (fin du film des socles), corrigée ici : la fin
            du film des armes au sol est strictement après la dernière image-clé, comme celle des
            poses (`gwFilmEndUS`, test `TestGwFilmEndApresLaDerniereImageCle`, mutation jouée
            rouge).
      - [x] 2.7.d4 *Clôture* : registre des replis, montée de `grammar.Rev`, ADR 0037 amendé, doc,
            gate de l'item.
            *Fait (2026-10-08, sur `feat/v75` `acfe4851a` fusionnée, rangs alignés avec le lot des
            arrêts de la vue B, fusionné avant)* : registre des replis (deux entrées neuves, ordre
            « après la lecture », et le texte de `repli_ancrage_bipede_apres_la_marche` étendu aux
            positions ; doc du registre des positions) ; `grammar.Rev` `grammar-2026-10-07.8` →
            `grammar-2026-10-08` (chronique, empreinte) ; `killsource.Rev` et `objectives.Rev`
            constantes (sorties identiques, complément de chronique, goldens régénérés) ;
            `replay.SchemaVersion` 88 → 89 (chronique v89, `structure_test`, plafonds du ratchet de
            taille), `SchemaDesFaits` 10 constant (aucune section ne change de forme) ; fixtures de
            contrat, forme du document, goldens d'assemblage, formes des types, références
            d'équivalence re-figées ; ADR 0037 IR-6 amendé (positions, objets du monde, population,
            `Mort`).
            *Gate de l'item* :
            - `replay-equiv`, binaires de `acfe4851a` (références re-figées par lui) contre ceux du
              lot : 20 films sur 20 divergent, par les étapes attendues — positions, socles,
              poses (orientation du poseur, qui suit les positions) et l'artefact partout,
              projectiles sur 17 films, véhicules sur 10 ; les huit lecteurs bipèdes sur quatre
              films (`a349fea8`, `e5adf7b2` : le dead-state qui ne dit pas la mort ; `111fa685`,
              `1c4c63c2` : la règle des trames non prouvées, 1 à 10 lectures rendues par lecteur) ;
              `killsource` et `objectives` identiques. Références re-figées par le binaire final.
            - `cmd/killsource json` sur les 19 témoins, binaire de `acfe4851a` contre binaire du
              lot : sorties identiques à l'octet (`killsource.Rev` reste).
            - `replay-corpus-gate` contre `acfe4851a` (19 témoins, banc de vérité compris), après
              les correctifs des découvertes 43 et 45 : FAUX partout par les deux replis neufs (R-1,
              nouveaux par construction) ; gains V-2 sur six témoins (`fb1a1a72` 4 → 0,
              `d9781168` 1 → 0, `c75f33b8` 2 → 0, `084a804d` 10 → 9, `4f77afc1` 10 → 3,
              `f75e7053` 2 → 1), `084a804d` O-V1 FP 1 → 0 et V-4 1 → 0, fins de vie sans identité
              en baisse sur trois témoins. FAUX instruits : `0797ce72` V-3 10 → 17 (sept
              changements d'arme des slots 595 et 611, 66 s avant leurs vies, lus dans des trames
              refusées : un dead-state sans mort les coupait, la règle de `Mort` les rend) et
              `repli_lien_prise_arme_abandonne` 0 → 4 ; `e5adf7b2` V-3 13 → 14 et un mort de
              moins par les vies pour un joueur (12 → 11) ; `a349fea8`
              `repli_impulsion_fusionnee_dans_le_geste` 0 → 1 (lectures du slot 590 rendues) ;
              `396cfc92` une fin de vie sans identité de plus et
              `repli_vie_coupee_au_trou_de_replication` 0 → 1 (une vraie position précoce, slot 520
              au chunk 1, trame fermée, avance l'origine du rejeu de 28 006 à 17 578 ms) ;
              `50247b26` V-2 899 → 1 135 (2.7.d3, bornes). Filet (sans oracle) : projectiles −1 à
              −30, armes au sol −1 à −7, véhicules −1 à −5 (châssis de créations fortuites ;
              naissances moins écartées, l'emprise jouée se mesurant sur des positions sans
              fantômes : 69 → 4 sur `4f77afc1`, 47 → 11 sur `084a804d` ; d'où des relais au même
              point que la règle des relais fond ; `4f77afc1` : 15 tirs de véhicule non placés), des occupations de
              socle sans ramasseur daté de plus là où une création fortuite coupait une vie
              (`4f77afc1` +4, `d9781168` +1, `084a804d` +1 ; `c75f33b8` une occupation non couverte
              de plus), un socle de Disrupteur reconnu sur `f75e7053` (sa première arme était lue à
              z = −755 m).
            - Tests du film, d'`archlint`, de `replaybuild`, vet (avec `research`), golangci-lint :
              verts ; `KILLSOURCE_FIXTURES` et `make gate-push` : ci-dessous.
- Gate : `replay-corpus-gate` et banc de vérité ; `KILLSOURCE_FIXTURES` en local ; montée de
  `grammar.Rev` (et `killsource.Rev` pour 2.7.c) ; recuisson et backlog sur signal de l'utilisateur.

### Lot 3.1 — Retrait des marcheurs redondants, un contexte par cuisson (taille M)
- [ ] 3.1.1 Retrait de `marchRecordsOf`, de la timeline de killsource et des recopies de pilotage qui
      restent ; un contexte par cuisson (killsource et rejeu) et par passe du collecteur (rapport
      §1.6, marche d'ancres d'image-clé recalculée deux fois par cuisson). Les localisateurs jumeaux
      sont retirés par LU (campagne), pas ici.
- [ ] 3.1.2 `DecodeFrameViews` sans appelant de production depuis l'étape 1 (découverte 2 de
      l'étape 1) : ses quatre fichiers de test non étiquetés passent sur la structure, puis il est
      retiré avec ce qu'il ne sert plus.
- Gate : banc ; T4 ; critère 4.

### Lot 3.2 — Mesure (taille S)
- [ ] 3.2.1 Durées par étape, pic mémoire, nombre de parcours du film ; rapport publié, comparé à la
      mesure M.1. Y compris la lecture des porteurs au sync des matchs à bombe
      (`replay.PortagesAuSync`) : depuis 2.7.b, ses changements d'arme tenue marchent les trames du
      film au lieu du seul ancrage.

## 4. Contrat d'exécution

Identique à l'étape 1 (§4 de son plan) : skill `plan-execution`, ordre strict, une étape commencée
est terminée, aucun report d'une action exécutable maintenant, statut de chaque item, clôture d'un
lot = gate + items statués + plan à jour + entrée `.ai/thought_log.md` + point à l'utilisateur en
langage clair sans codes de lots ; commits préfixés `ri2(<lot>): …`, chemins explicites, jamais
`git stash` ; demander avant tout push et toute fusion ; aucune commande `go` concurrente sur un même
cache ; jamais de Python ; commentaires = contrat (règle 17).

**Gates** (depuis `apps/go-api`) :

| Nom | Commande |
|---|---|
| G-unit | `go test <paquets du lot> -count=1` |
| G-arch | `go test ./internal/archlint/ -count=1` |
| G-vet | `go vet ./...` (CGO) et `go vet -tags=research ./...` |
| G-film | `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1` |
| G-race | `go test -race -count=1 -run TestDeuxFilmsEnParallele ./internal/games/halo_infinite/film/internal/grammar/` |
| G-equiv | `go run ./cmd/replay-equiv -repo-root <worktree>` → zéro divergence, chaque film décodé (faits mis de côté) |
| killsource | `go run ./cmd/killsource json <film> -carte <carte> -cache <worktree>/data/cache` sur les témoins de `config/replay_corpus.toml`, avant/après → identique à l'octet |
| G-corpus (2.7) | `go run ./cmd/replay-corpus-gate` (mode `base` contre `origin/feat/v75`, banc de vérité `cmd/replay-verite` compris) → aucune perte non déclarée |
| G-perf (critère 4) | binaires de base et du lot ALTERNÉS, deux tours, trois témoins et un BTB, machine calme, faits effacés avant chaque cuisson ; écart par paire et dispersion publiés |
| G-CI | `gh run list --branch feat/ri-etape2 --limit 3` → jobs verts |
| G-push | `make gate-push` avant toute fusion dans `feat/v75` |

**Pièges d'exploitation connus (étape 1)** :
- la fraîcheur des faits ne regarde que les révisions déclarées : avant toute passe de preuve à
  révision constante, renommer `data/cache/film_facts` du worktree (dossier réel, jamais une
  jonction), vérifier `depuis_les_faits=false` dans le journal, et comparer les faits neufs à
  l'octet ; même précaution avant chaque cuisson mesurée ;
- sous la charge de la campagne, la durée ET le pic mémoire varient jusqu'à 15 % pour un même
  binaire : mesurer machine calme, binaires alternés, publier l'écart par paire et sa dispersion ;
- un PC qui se met en veille gèle une passe de décodage sans la tuer : la relancer n'est pas
  nécessaire, elle reprend au réveil ;
- `replay-equiv` se lance avec `-repo-root <worktree>`, jamais `LEVELUP_REPO_ROOT` ; une passe longue
  se lance détachée et se surveille, jamais en arrière-plan d'un shell qui la tuerait ;
- une édition par numéro de ligne (`awk`, `sed` sur `NR`) après un premier ajout dans le même fichier
  a écrasé une ligne de code à l'étape 1 : éditer par motif exact.

## 5. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier (§1.3, §2, §7) ; reprendre à la première case non
statuée ; `git -C <worktree> log --oneline -10` ; vérifier sur `origin/feat/v75` si la campagne a
fusionné un lot depuis la dernière reprise (et refusionner, re-figer).

## 6. Découvertes (consignées, non traitées)

Les découvertes encore ouvertes de l'étape 1 restent dans son plan, §6 ; celles qui relèvent de ce
plan y sont reprises comme items (3.1.2).

1. *(lot 2.6)* **Deux instruments de `grammar` importent `facts/objectives`** pour leurs oracles
   (`sonde_registre_verdicts_test.go`, `zone_census_report_test.go`, non étiquetés, gardés par
   l'environnement) : tout paquet des faits qui consomme la grammaire fermerait un cycle d'imports
   dans le binaire de test de `grammar`. Le lot 2.6 l'a contourné par une FEUILLE de l'arbre de la
   grammaire (`grammar/signaux`, qui n'importe pas `grammar`). Le contournement tient tant que
   `objectives` ne consomme que des feuilles ; à rouvrir le jour où il devrait consommer `grammar`
   lui-même (déplacer ces deux instruments, ou leurs oracles).
2. *(lot 2.6)* **Deux lecteurs du même pied de film** vivent dans l'arbre de la grammaire :
   `signaux.FooterEvents` (blocs th=10, équipe à l'octet 37) et `grammar.ParseHighlightEvents`
   (tous les types, gamertag par version). Même balayage (xuid au bit, marqueur de fin, bloc de 60
   octets), bornes de xuid recopiées. Les unifier demanderait une preuve d'équivalence ; non traité.
3. *(lot 2.6)* **L'horloge du statborg est recopiée** : base = premier paquet delta du chunk,
   `start_ms` du manifeste ; `navpoint_radial_scan.go` et plusieurs instruments de `grammar` disent
   « la MÊME base que `objectives.StatRecords` » et la recalculent. `signaux` pourrait la porter pour
   tous ; non traité.
4. *(lot 2.6)* **Le statut de l'ADR 0037 était périmé** depuis la fusion de l'étape 1 (« Step 1 is
   being executed ») : corrigé dans le lot (affirmation fausse rencontrée, CLAUDE.md règle 17).
5. *(lot 2.6)* **Deux enveloppes de production lues par les seuls tests**, déplacées telles quelles
   pour garder le déplacement vérifiable : `signaux.scanFrameForRecords` et
   `signaux.decodeComponents` (l'une rend `scanFrameAvecReplis` sans compte, l'autre
   `decodeComponentsAvecArret` sans le drapeau d'arrêt). Dette antérieure (CLAUDE.md règle 7) ; non
   traitée.
6. *(lot 2.1)* **Les deux phases ne lisent pas sous le même découpage MPP** : la phase des
   images-clés pose celui du FORMAT du film (`EnTete.MPP`, comme `InstallFilmFormatMPP` avant), la
   marche des trames lit sous celui du CONTEXTE — l'invariant (9/5), sauf installation en cours.
   Les records NEW d'objets du monde que la marche des trames traverse le lisent donc aux largeurs
   par défaut sur les formats qui en ont d'autres. Gardé tel quel (différence nulle) ; à mesurer
   avant de l'unifier (une montée de `grammar.Rev`).
7. *(lot 2.2)* **Des parcours d'images-clés restent hors de la phase**, hors de la liste fermée de
   2.2.1 : la marche des naissances (`birth_loadouts.go` : sa marche d'ancres et `LierTableDeDatums`
   chunk par chunk), l'anneau de la bombe (`navpoint_radial_scan.go`, état complet de ti=12), les
   objectifs (`objective_scan.go`, état complet de ti=11), les morts d'objet (`object_deaths.go`,
   `marchPacketsOf`, lot LU de la campagne), killsource (`facts/killsource/world.go`, 2.7.c),
   `roster_type8.go`. À porter par 3.1 (une distribution par cuisson) ; non traité.
8. *(lot 2.2)* **Les fenêtres de bits des images-clés décident DEVANT une lecture que la grammaire
   atteint.** Mesure du 2026-10-03 sur trois bobines par build (phase complète des images-clés) : la
   traversée de l'état complet du bipède (ti=35) atteint les compteurs de grenades (i22), le bloc
   des munitions (i30 à i42), les quatre `weapon-state-type-info` (i43 à i46, l'identifiant d'arme
   en clair), les ensembles de grenades et de capacité (i47, i48) dans 188 records sur 237
   (`e5adf7b2`), 80 sur 80 (`fb1a1a72`) et 188 sur 209 (`a521164d`) ; elle s'arrête ensuite à i59 ou
   i60 (`simulation-state-component`, non porté), donc aucun record n'est prouvé fermé. Les inscrire
   au registre comme méthodes de récupération (2.2.2 tel qu'écrit) les déclarerait
   `devant_la_lecture` — la violation de D14 (b) que le cliquet `NbDevantLaLecture` interdit de
   faire monter. La forme juste est de LIRE ces valeurs par la grammaire (intérêts de la phase des
   images-clés sur ti=35, lecture à l'étendue de l'occurrence) et de retirer les fenêtres :
   changement de comportement (une fenêtre retient toute famille connue où qu'elle tombe dans
   l'emprise du record, alias compris ; la grammaire lirait les emplacements que l'écrivain écrit),
   donc un lot de 2.7, prouvé au corpus et au banc de vérité. Décision de l'utilisateur du
   2026-10-04 : option A — item 2.7.d (2.2.2 statué `[~]`).
9. *(mesure M)* **Le pic d'une cuisson n'est pas dans le décodage.** Trace du ramasse-miettes sur
   le BTB (`084a804d`, les deux binaires) : le tas vivant reste sous 260 Mo pendant tout le
   décodage, puis monte à 450-525 Mo dans la dernière seconde (fin du décodage, assemblage,
   écriture des faits et de l'artefact) ; l'empreinte que la sentinelle mesure y approche le double
   (la cible de croissance du ramasse-miettes) et varie de 0,90 à 1,14 Gio d'une cuisson à l'autre,
   selon le calage du cycle sur ce pic d'allocations. Deux conséquences : un gain de la
   représentation intermédiaire sur le décodage ne se verra pas dans ce pic, et le seuil de 10 % du
   critère 4 ne se décide pas sur lui avec quelques paires. Les mesures du critère 4 (après 2.4,
   2.5, 2.7, 3.1) et 3.2 gagneraient à lire aussi le tas vivant par phase (trace du ramasse-miettes,
   comme à M.1). Non traité.
10. *(lot 2.4)* **L'ancrage d'en-tête bipède décide devant la lecture de la marche.** Les neuf
   lecteurs ancrés (positions et huit canaux) lisent tous les records qu'ils ancrent, y compris ceux
   que la marche des trames lit (97 447 records `ti=35` marchés contre 162 444 ancrés sur
   `bfecd02b`, en-tête de `movement_states.go`). Inscrire l'ancrage au registre comme méthode de
   récupération (DT2-4 ; ADR 0037 D-10 : nommée, ordonnée après la lecture, comptée, retirée) le
   déclarerait `devant_la_lecture`, ce que le cliquet `NbDevantLaLecture` interdit de faire monter.
   Même question que 2.2.2 (découverte 8) ; la forme juste est 2.7.b (les canaux lus par la marche
   là où elle couvre au moins autant, l'ancrage seulement après). Décision de l'utilisateur du
   2026-10-04 : option A — item 2.7.d.
11. *(lot 2.7.a)* **Sous les largeurs MPP calibrées sur les poses (8/3), la marche des trames ferme
   la vue C deux fois et demie à cinq fois plus souvent sur six des neuf films des formats sans
   largeur relue, à l'identique sur les trois autres** (suite de la
   découverte 6). Passe `ri27b` (marche des trames de la cuisson sous `gwWidthsForFilm`) contre
   `v75w2`, paquets à vue C fermée : `084a804d` 4 837 → 23 642, `1c4c63c2` 13 389 → 33 999,
   `60ae07c4` 13 948 → 34 578, `11de8353` 5 631 → 13 839, `111fa685` 4 026 → 11 932, `e5adf7b2`
   4 149 → 12 267 ; `a349fea8` 424 → 428, `a521164d` 692 → 693, `50247b26` 139 → 138. Non retenu :
   c'est le lot LM, mis de côté par l'utilisateur le 2026-10-02 (largeur mesurée sur des builds
   sans exécutable, non lue dans le jeu ; exception D6 suspendue). Signalé à la campagne ; la
   décision reste celle de l'utilisateur.
12. *(lot 2.7.a, recherche Ghidra du 2026-10-05, demandée par l'utilisateur : « le jeu dans sa
   version actuelle sait lire tous les films ; les films contiennent eux-mêmes leur index de
   décodage »)* **Aucune donnée du film trouvée qui fasse lire le bloc MPP autrement entre les
   formats 25 et 27.** Lecture seule, HTTP 127.0.0.1:8089. Le lecteur d'état par défaut des
   véhicules est `FUN_1410a5a74` (vtable `0x143736fd8` +0x60). Les descripteurs par type sont
   enregistrés statiquement (`FUN_140e453b4`), la table `DAT_144e61d88` n'a qu'un écrivain
   (`FUN_14054d014`). `FUN_14080cfe8` lit des largeurs littérales ; `FUN_141fd72c0` (R(9)) et
   `FUN_14080d4d0` n'ont que lui pour appelant. Le préfixe R(1)+R(8) des lecteurs d'état par défaut
   est lu puis jeté, sauf dans `FUN_140f44c38`. Versions par type sur les sept bobines :
   `11de8353`, `111fa685` et `e5adf7b2` (8/3) identiques au format 27 (9/5) sur les 25 index qui
   varient, y compris l'index 0x28 que `FUN_140ff8d70` consulte (absent de la liste de 1.9.1
   ter). Registre des archétypes 36 à 43 : `111fa685` et `e5adf7b2` identiques au format 27.
   Lecteurs de la version de format : seuils 3, 7, 11, 13/14 et 15, aucun entre 25 et 27. L'autre
   écart de trois bits connu, celui du tir à composantes (campagne, R1 §3), n'a PAS la même
   frontière : formats 24 seulement (`e5adf7b2`, format 25, lit 16 comme le 27). Ses largeurs sont
   elles aussi fixées par l'exécutable : R(3) par `FUN_1406d310c(6)`, R(16) littéral, base des axes
   12 ou 4. Un élément commun aux deux chemins est donc peu probable. Reste ouvert : l'écart est
   soit dans notre propre lecture ailleurs (le 8/3 le masquerait), soit piloté par du code non
   encore lu.
13. *(lot 2.7.a, agent d'enquête du 2026-10-05, demandé par l'utilisateur)* **Les deux écarts : condition
   « non trouvée » dans l'exécutable courant, et notre lecture en amont n'est pas fautive ; la
   différence est chez l'écrivain des anciens films.**
   - *Bloc MPP (LU)* : chaîne entièrement littérale (R(9) `141fd72de`, R(2) `14080d18b`, R(5)
     `14080d1cf`, R(3) `14080d20f`) ; le 3e argument de `FUN_14080cfe8` est écrasé en `14080d077` ;
     un seul lecteur du bloc ; l'écrivain `FUN_142f1bc2c` écrit 9 et 5.
   - *Bloc MPP (MESURÉ)* : oracle `n2` modal à 0,974-1,000 en 8/3 sur les cinq bobines anciennes
     pour ti=35, 37, 38, 42 et 43 (0,02-0,54 en 9/5), l'inverse sur le format 27 ; mêmes
     identifiants de 32 bits un bit plus tôt (ti=38, préfixe V = porte 1 + octet 3 sur 100 % des
     records) ; les deux autres bits tombent dans une plage de zéros (R(2), index, compte),
     inséparables et sans effet sur les valeurs lues.
   - *LE FILM DIT LA TAILLE* : `n1` (mot de tête des records d'image-clé) est la taille de la
     structure d'état de création que l'écrivain a rangée (`FUN_142e2d08c` y met `vtable+0x20`).
     Sur les formats 21, 24 et 25 elle vaut 4 octets de moins pour chaque archétype à bloc MPP
     (ti=37 : 100 contre 104 ; ti=35 : 148 contre 152 ; ti=42 : 164 contre 168 ; ti=43 : 92 contre
     96 ; ti=40 : 172, taille courante 176), inchangée pour les autres. Le jeu ne lit `n1` que comme
     garde (> 0). Clé « taille déclarée − 4 ⇔ 8/3 » : vraie sur 7 bobines × 7 archétypes, mesurée.
   - *Tir à composantes* : boucle littérale (R(16) `14080c74c`), écrivain `FUN_142f193e4`
     identique, répartiteur `FUN_14080a9d4` sans version. Frontière = format 24, HI_1_8_0 compris
     (`60ae07c4` 914/918 à 13 bits) ; `e5adf7b2` (format 25) 397/422 à 16. Seule clé disponible :
     format 24 / cardinal 121, mesurée.
   - *Fait lu, sans conclusion de l'agent* : `FUN_1428e219c` (appelée par `FUN_140ba23e4`)
     n'installe la lecture d'un film que si le premier mot de chunk_00 (version majeure) vaut 41 ;
     sinon, comme sur ses autres échecs, elle poste l'événement 0x1e (`FUN_142988e98`).
     Relu le 2026-10-05.
   - Piste restante classée première par l'agent : observation dynamique d'un film ancien dans le
     jeu (points d'arrêt, MCP Cheat Engine), sur autorisation de l'utilisateur.
14. *(lot 2.7.a, instruction du gate de corpus, 2026-10-05)* **Une posture dont le début est
   désormais lu mais pas la fin court jusqu'à la fin de la vie.** Postures invraisemblables
   (escalade > 2 s, saut > 3 s, glissade > 3 s) sur les huit films en baisse : 118 épisodes
   (3 473 s) → 94 (2 690 s) ; 27 retirés, 3 nouveaux : glissade de 29 s du slot 619 de `e5adf7b2`
   (3177-3464, aucune fin de `i62` lue), saut de 13 s du slot 742 de `084a804d` (6197-6328, il
   traverse la montée à bord lue à 6206 : la piste s'arrête, rien ne ferme le saut), escalade de
   52 s du slot 629 de `11de8353` (3289-3812). Le bâtisseur des postures ne ferme un épisode que
   sur une lecture levée ou la fin de la vie ; l'état complet de l'image-clé le fermerait.
   Couvert par 2.7.d (lecture de l'état complet du bipède aux images-clés). Non traité.
15. *(lot 2.7.a, même instruction)* **Un DELTA dont la génération contredit l'entité du monde au
   même slot est lié quand même.** Sur `4f77afc1`, le record du slot 737 en génération 3 (le monde
   y tient la génération 1, vivante jusqu'à 8943 au moins aux images-clés) est lu propre
   (`DesyncAt = -1`, liaison d'image-clé, liste localisée par la marche, pas une liste récupérée)
   et publie une lecture d'occupation attachée au bipède 618. La conséquence côté rejeu est
   corrigée en 2.7.a (fermeture par le même objet) ; la lecture elle-même relève de la famille
   « lecture contredite » de la campagne (lot LR), à qui elle est signalée. Non traité ici.
16. *(lot 2.7.a, même instruction)* **La primauté de la lecture nomme les occupants par CORPS, pas
   par JOUEUR.** Un épisode de repli est écarté si le film a lu, pour la même vie de véhicule, des
   montées à bord dont son slot ne fait pas partie (`vehicleFilmRides.contredit`). Sur `084a804d`,
   le film lit le joueur `…447` (slot 574) au volant du 879 à 8960 ; son trajet de repli de la vie
   précédente (slot 745, 6697-8656, trou de position) est écarté, et dix tirs de l'arme du 879
   perdent leur tireur. Les trois tirs du 916 perdus (slot 608) relèvent de l'autre forme
   (chevauchement de cinq images avec le trajet lu du 599). Règle arbitrée le 2026-09-21.
   Décision de l'utilisateur du 2026-10-05 : par joueur (`e127e90fb`, item 2.7.a) ; le
   chevauchement contredit toujours, un occupant sans identité reste désigné par son slot.
17. *(relecture du lot « vue A » V1 de la campagne, 2026-10-05)* **Les seuils de la règle 5 ne sont
   plus vérifiés par le lint sur le code NEUF du décodeur.** `.golangci.yml:223` exempte de
   `gocyclo`, `funlen` et `lll` tout `film/(replay|internal/(facts|source|grammar|profile))/`,
   exemption écrite pour du code DÉPLACÉ (lots E.2 et 2.5). Le code neuf de ces couches (celui de
   ce plan compris) n'est donc tenu aux seuils que par la relecture : la relecture de V1 y a trouvé
   deux aiguillages à complexité 20 et 22 sans justification. Non traité ici ; à porter à
   l'utilisateur (resserrer l'exemption au code d'avant une date, comme `only-new-issues`).
   Décision de l'utilisateur du 2026-10-06 : « Option 1 » — `gocyclo` et `funlen` sortent de
   l'exemption (`lll` y reste), les fonctions qui dépassent (16 mesurées, toutes de complexité)
   sont simplifiées quand c'est simple et couvert, sinon exemptées avec une raison écrite ; lot
   séparé `feat/lint-decodeur`, mené par un agent en parallèle de ce plan.
18. *(fusion de `feat/v75` dans la branche, 2026-10-05)* **Le banc killsource sur films réels
   (`TestGoldenFilms`, sauté sans `KILLSOURCE_FIXTURES`, donc absent de la CI) était rouge sur
   `feat/v75` depuis au moins `87cdfa761`** : trois films, la seule ligne de diagnostic de
   calibration (scores de l'oracle de profilage ; décision et morts identiques). Sortie de la
   branche fusionnée identique à `feat/v75` seule. Signalé à la campagne, qui a régénéré les
   goldens (`65c99b669`, même famille que sa découverte D23). Un banc qui ne tourne qu'en local
   peut rougir sans que personne le voie : à rejouer à chaque fusion qui touche le décodeur.
19. *(lot 2.7.b, mesure du 2026-10-06)* **Les essais du localisateur de liste publient les crochets
   de composant.** La porte unique des essais ne couvre que les états de mouvement : quand le
   localisateur cherche le début d'une liste, ses lectures d'essai déposent, sur les 20 films,
   243 949 appels d'arme portée pour 16 032 lectures retenues, 258 086 de munitions pour 61 998 —
   tous les crochets des huit lecteurs ancrés, 5 à 40 fois les lectures retenues. Dans les trames
   parties de leur tête, les crochets de la marche sont exactement ceux de ses records, sauf 135
   appels de capacité non prédite (à rapporter à la réparation d'un composant non porté). Sans
   incidence aujourd'hui (aucun de ces crochets n'est posé sur la marche) ; préalable de 2.7.b.
20. *(lot 2.7.b, même mesure)* **Les records NEW bipèdes portent les composants des lecteurs
   ancrés** : 9 447 lectures d'arme portée, 4 413 de cartouches, 4 398 de munitions, 2 345 de
   grenades sur les 20 films. L'ancrage ne les voit jamais. Matière possible des dotations de
   naissance ; hors de ce plan.
21. *(lot 2.7.b, même mesure)* **La marche n'accumule pas les positions** : l'accumulateur de la
   capture de position (`captureDePosition.accum`) n'a pas d'écrivain de production, le décodeur
   de positions accumule de son côté. Une position lue par la marche n'est qu'un delta quantifié
   tant que la marche ne tient pas un monde de positions.
22. *(lot 2.7.b, instruction du gate du 2026-10-06)* **La piste d'une vie part de son premier
   mouvement, pas de sa création.** Le record NEW d'un bipède porte sa position de naissance (i0),
   que le décodeur de positions ne lit pas ; un corps immobile (gel d'avant-match, quelques
   dixièmes de seconde après une réapparition) n'a pas de delta de position. La marche lit le rang
   de capacité que le jeu transmet 13 à 15 ms après la création (`0797ce72` slot 524 : NEW à
   3642,645 s, rang à 3642,658 s, première position à 3643,426 s) : le banc le juge « hors vie ».
   À traiter avec les positions derrière la grammaire (2.7.d) : la position du NEW ouvrirait la
   piste.
23. *(même instruction)* **Un début de liste choisi par la fermeture ne prouve que la suite de la
   liste.** `debutParFermetureRangee` prend le PREMIER candidat (un NEW) d'où la marche ferme le
   paquet ; il peut être au milieu de la liste. Sur les 20 films, 2 449 paquets fermés à début
   `DebutParFermeture` portent 35 256 records ancrés avant ce début (2 152 avec un composant des
   huit lecteurs). La campagne avait mesuré ~600 paquets où deux débuts incompatibles ferment ;
   son lot V2 (début lu à la fin de la vue A) en tranchera une part. Liste des paquets remise à la
   campagne (`scratchpad/ri/ri27b/paquets_par_fermeture.tsv`). Traité dans 2.7.b pour les lecteurs
   (décision 9) ; la grammaire elle-même garde son premier rang (campagne).
24. *(même instruction)* **En fin de manche, le jeu retire l'équipement des corps.** La marche lit ce
   retrait (records sans position, après le dernier mouvement) et le lecteur d'équipement le publie
   « dépensé » : `51ebbc0f`, deux à la fin de la manche 1 (un hors de la piste, V-3). Ce n'est pas
   un geste. Un signal de fin de manche manque à la grammaire ; le rejeu connaît les bornes de
   manche. Hors de ce lot.
25. *(même instruction)* **La garde des générations vivantes datées ne valait que pour l'ancrage.**
   Le lot R2-bis l'a posée sur « tous les lecteurs de records delta bipèdes » ; le canal de 2.7.b
   l'avait perdue pour les records de la marche (décision 8). Un record de trame non prouvée peut
   être lu au-delà d'une largeur fausse : génération 0 sur un slot vivant en 1, masque à trente
   composants.
26. *(relecture de la vue A V2 et V3 de la campagne, 2026-10-06, constat P2)* **Un essai de début de
   liste peut armer le diagnostic de la liaison par anticipation.** `lectureDEssai` restaure la
   table d'entités (`World.Snapshot`/`Restore` : `slots` seulement) ; un DELTA lu pendant l'essai sur
   un slot non lié qu'une image-clé ultérieure déclare laisse `anticipationDite` vrai, et le
   diagnostic « repli actif » part avec le slot d'un essai jeté, que le compte de la liaison (sur
   l'observation neuve de l'essai) ne voit pas. Préexistant par `debutParFermetureRangee` ; la vue A
   V2 l'étend à chaque paquet d'un film préfixe. Signalé à la campagne.
27. *(reprise de la vue A V2 et V3, 2026-10-06)* **La marche partie de la fin de la vue A bute sur les
   composants que le décodeur ne porte pas** (`ti=43 i19`, `ti=12 i16`, `ti=45 i0`, `ti=10 i2`…) ;
   ce qui les suit dans le paquet n'est plus lu par aucun canal, alors que la signature du slot 123
   le faisait lire. Pour les huit lecteurs, une quinzaine de records sur 20 films (mini-bobine :
   paquet 2:712). Décision de l'utilisateur du 2026-10-06, rapportée par la campagne : « Non,
   grammaire d'abord » — pas de reprise en queue ; les composants bloquants sont au registre des
   reports, avec le handoff `.ai/HANDOFF_COMPOSANTS_BLOQUANTS_VUE_B_2026-10-06.md` pour un agent frais
   (lancement au choix de l'utilisateur ; recommandation : après la fusion de 2.7.b).
28. *(clôture de 2.7.b, 2026-10-07)* **Un test de chronométrage rougit la CI et le gate local sous
   charge.** `TestRosterDesFilms_AnnuaireContreJointure` (`sync/killcollector/backfill_cout_integration_test.go`)
   exige que la jointure par match coûte au moins dix fois l'annuaire de passe : 7,7 mesuré sur la
   CI de `9d37645b3`, 9,1 sur le gate local du lot lint ; vert sur les autres exécutions. Hors de ce
   plan : un rapport de chronos sur une machine partagée ne se fige pas à un seuil fixe. Le paquet
   `platform/duckdb` tourne en 250 s seul sur ce poste, pour un plafond de 300 s : sous charge, il
   sort en dépassement.
29. *(recuisson unique du parc par levelup-dc, 2026-10-07)* **Le document des porteurs au sync
   journalise « roster SANS ÉQUIPE » en erreur sur les modes à crâne, bombe et VIP.**
   `replay.PortagesAuSync` ne lit les équipes du film que pour le drapeau ; son document (jamais
   publié) passe quand même par `journaliserLesPlaces` (`replay/sieges.go`) : 82 matchs journalisés
   pendant le rattrapage, dont 73 sans aucune équipe. Le placement des vies, lui, se range par camp
   depuis la feuille du match (`placement_des_vies.go`). Journal en trop ; correction proposée à
   levelup-dc (son journal) ; à reprendre avec 2.7.c, qui passe ce chemin sur la marche unique. Le
   même rattrapage laisse 105 matchs que le `--dry-run` sélectionne et que la passe réelle ne prend
   pas (sélection contre `killcollector.PlacementRev`), non instruit.
30. *(mesure de 2.7.c, 2026-10-07)* **109 kill-events de killsource tombent dans la vue A lue, hors
   du début d'un message.** La vue A unique est passée par leur bit sans y lire de genre 85 : un
   faux positif du balayage bit à bit (la chaîne de trois messages tient par hasard) ou une vue A
   mal lue avant eux. À instruire avec les kill-events (2.7.c4).
   *Instruit en 2.7.c4 (2026-10-07)* : les deux. Sur 28 films, la recherche d'avant trouve dans une
   vue A lue, hors d'un début de message, 14 messages à chaîne longue (douze évènements) dans des
   trames dont la fin de vue A est retenue — une vue A mal lue, découverte 34 — et 40 à chaîne courte
   (bruit). Là où la fin n'est pas retenue, le rattrapage les reprend.
31. *(même mesure)* **Deux règles trouvent le chunk des temps forts.** La cuisson le désigne par le
   type du manifeste (repli : le dernier numéro, `repli_temps_forts_dernier_numero`), killsource
   par son contenu (le chunk qui porte le plus de kills). 2.7.c1 déplace la règle de killsource
   telle quelle ; les réunir changerait un comportement. Non traité.
32. *(2.7.c3, 2026-10-07)* **La preuve des ancres d'image-clé juge encore les films anciens sous le
   découpage MPP du format (9/5).** Depuis 2.7.c3, la cuisson, killsource et le ratchet de fermeture
   lisent le corps des records sous le découpage déclaré (8/3) ; la preuve des ancres
   (`PreuveDImageCle`, `keyframe_world_preuve.go`) lit l'en-tête de la marche, où la résolution
   n'entre pas (`mpp_declare.go`). Sur ces films, un record d'objet n'y est donc prouvé qu'à
   l'identité mal lue. Y poser la résolution changerait les ancres de toutes les lectures : à
   mesurer avant d'être décidé. Non traité.
33. *(même lot)* **269 records d'image-clé des cinq bobines anciennes ne ferment plus sous le
   découpage déclaré**, sur 49 slots (`ti=38` surtout, `ti=42`, un `ti=37`). Aucun n'est arrêté par
   un composant sans lecteur : la traversée finit avant ou après la frontière, un composant plus loin
   lit faux. Leur fermeture sous 9/5 était une coïncidence (identité inconnue du catalogue, preuve 2) ;
   le composant fautif n'est pas instruit.
34. *(vérification de 2.7.c4, 2026-10-07)* **Une fin de vue A fausse, après un message de dégâts
   (genre 0), décide parfois du début de la vue B.** Sur une table des genres égale, la fin lue de la
   vue A vaut lecture. Lue au-delà des messages de kill, elle décide sur bien plus de trames, et juste
   presque partout (28 films, contexte de killsource : trames fermées 674 171 → 698 688, refusées
   156 170 → 131 798, records lus +133 000 ; 24 839 trames deviennent fermées, 322 cessent de l'être).
   Pas toujours. Sur les quatre films de référence de killsource, parmi les 262 trames à kill que la
   marche d'avant localisait et fermait, la fin lue sans la queue du 85 tombe au début localisé dans
   236 ; avec la queue, dans aucune (la queue est bien absente). Les 26 autres portent toutes, après
   le 85 et ses genres 82, un message de dégâts (genre 0, `damage_aftermath`) ; la lecture finit sur
   lui ou sur un ou deux messages qui le suivent (genre 1, genres 82), par un terminateur lu des
   centaines à des milliers de bits trop tôt (`78919882` : 22 trames sur 83 ; `fccc61cd` : 3 ;
   `9b191a7f` : 1 ; hors de ce compte, `000d5950`, chunk 3, rang 804 : fin au bit 2 787 contre 3 180,
   où la marche lit 18 records dont une mort créditée). Effet : sur ces trames la marche part trop tôt
   et refuse le paquet ; killsource passe leurs morts au balayage (`78919882` : 28 morts, contenu
   publié identique ; 20 sur les 19 témoins, 56 font le chemin inverse), et la cuisson y perd ses
   records — le même film gagne par ailleurs 4 400 trames fermées. Suspect : une branche de la lecture du genre 0 (`chargeDegatsApres`,
   `FUN_1407f15a4`, déjà noté pour sa porte à polarité discutée, D-LN-2). À instruire avec la
   grammaire de la vue A ; non traité.
35. *(même vérification)* **Des messages de kill que la recherche d'avant écartait par construction
   sont lus par la vue A** : 42 suicides (tueur = victime), 85 sans tueur, 13 sans victime, 79 dont la
   chaîne d'évènements qui suit ne se lisait pas (28 films). Ils entrent dans les kill-events de
   killsource. Sur les 19 témoins, deux morts publiées gagnent un kill-event attaché (l une sans
   assistant, l autre avec un assistant nommé) ; leur origine n est pas instruite au-delà.
36. *(mesure de 2.7.d0, 2026-10-07)* **La grammaire ne lit pas l'état complet du bipède aux
   images-clés.** La traversée de `ti=35` va au bout (9 849 records sur 10 710, carte posée), mais
   410 seulement se ferment sur le record suivant (3,8 %) et les valeurs relues à l'étendue des
   composants sont fausses : compteur de grenades différent de 4 dans 9 204 records (l'écrivain écrit
   4), emplacements d'arme vides ou inconnus là où la fenêtre trouve des paires plausibles ; même
   parmi les 410 records fermés, 123 compteurs faux (fermetures fortuites). L'écart entre l'i43 de la
   traversée et la première famille trouvée par fenêtre n'est pas constant : un composant de largeur
   variable est mal lu avant i43 dans la plupart des records. C'est le constat du golden de fermeture
   d'image-clé (`ti=35` : 0 à 3 records fermés par film, sans carte), que la découverte 8 avait lu
   comme « la grammaire atteint ces composants ». Non traité : il touche la grammaire des composants
   (campagne) ; 2.7.d1 en dépend.
37. *(même mesure)* **La passe des pistes donne un record à toutes les bandes qui portent son slot.**
   Les bandes de l'équipement et des armes au sol se recouvrent : sur quatre films, 108 386
   échantillons de la bande des armes au sol sont des records que la marche lit comme de l'équipement
   (`ti=37`), et 31 284 l'inverse. Une arme au sol et un équipement partagent alors un même échantillon
   de piste. 2.7.d3 (décision 1) le corrige par construction.
38. *(même mesure)* **L'ancrage bipède lit la position d'objets qui ne sont pas des joueurs.** Un slot
   de la bande bipède occupé par un autre archétype (un objet `ti=20` sur `111fa685`, slot 554, position
   constante de paquet en paquet) passe la grammaire d'en-tête de l'ancrage : 2 244 positions sur 28
   films. 2.7.d2 (décision 1) le corrige par construction.
39. *(même mesure)* **Des en-têtes d'objet du monde fortuits, dans des trames que la fermeture
   prouve.** La passe des pistes rend 243 532 échantillons, et celle des créations 14 017 créations,
   à des positions où la marche, qui a lu la trame jusqu'à son terminateur, ne lit aucun record de ce
   slot : ce sont des motifs à l'intérieur d'autres records. 2.7.d3 les écarte (règle de 2.7.b) ; le
   gate de l'item dira ce qu'ils publiaient.
40. *(2.7.d3, 2026-10-07)* **Le nuage de positions des véhicules ancre encore sous la grammaire
   d'en-tête bipède.** Le calque des véhicules lit ses positions par `ScanBipedPositionsForBand` sur la
   bande `ti=40` (un ancrage de payload, hors de l'ancrage du contexte : décision 1 du lot 2.4), sans
   la marche devant lui. Le même principe s'y appliquerait (la marche lit les records `ti=40` et leur
   i0). Non traité : hors de la liste de 2.7.d (positions bipèdes, huit lecteurs, pistes et créations
   des objets du monde).
41. *(gate de 2.7.d, 2026-10-08)* **La règle « un corps mort n'agit plus » de 2.7.b tuait le corps sur
   un dead-state qui ne dit pas la mort.** Elle testait la présence du composant, pas son drapeau
   `Mort` ; un dead-state à `Mort` faux, lu dans une trame refusée, écartait toutes les lectures du
   corps jusqu'à la fin du film (`a349fea8` slot 590 ; `111fa685` slot 545). Corrigé en 2.7.d2 : il
   bloquait le gate (V-3).
42. *(même gate)* **La marche lit des corps bipèdes qu'aucune image-clé ne porte** : les corps posés
   en fin de match pour la scène des vainqueurs, et un joueur né au dernier chunk d'une bobine. La
   bande des images-clés reste la population des positions (2.7.d2). Les huit lecteurs de 2.7.b ne
   filtrent pas leurs records par la bande ; ce qu'ils publient de ces corps n'est pas instruit.
43. *(gate de 2.7.d après la fusion de `feat/v75` `acfe4851a`, 2026-10-08)* **La fin du film des armes
   au sol tenait à des créations fortuites.** `gwFilmEndUS` prenait la plus tardive de la dernière
   image-clé, des positions et des créations ; sur `bcb6d393`, une création fortuite 2,6 s après la
   dernière image-clé la portait au-delà. Retirée (trame prouvée), la fin du film tombait SUR la
   dernière image-clé, que la fenêtre de recensement (exclusive à droite) retranchait alors de la vie
   des objets encore recensés : huit socles sortaient pris entre les deux dernières images-clés au
   lieu de jamais pris, treize armes « vues » au lieu d'« ouvertes ». La règle des poses
   (`equipment_placement_ends.go` : fin du film = dernière image-clé + 1) vaut désormais pour les
   socles. Corrigée en 2.7.d3 : elle bloquait le gate (régression publiée que le lot révélait).
44. *(même gate)* **Des vies d'équipement gardent des fragments sous la bande des armes au sol.**
   Derrière la marche, dans les trames qu'elle ne prouve pas, la passe des pistes rend encore à la
   bande des armes au sol les échantillons d'un équipement dont le slot tombe dans les deux bandes :
   sur la mini-bobine, sept vies d'équipement s'y découpent en 22 pistes (les pistes de la bande
   passent de 46 à 51), dix autres disparaissent. Aucun objet publié ne les lit : les armes au sol
   naissent des créations de leur archétype et ne prennent que les pistes de leurs vies. Règle
   candidate, non traitée : une vie (slot, génération) que la marche lit sous un archétype ne se
   rend pas à la bande d'un autre.
45. *(gate de 2.7.d après la fusion de `acfe4851a`, 2026-10-08)* **La table d'entités de la marche
   peut lire un joueur sous l'archétype de l'objet qui occupait son slot avant lui.** `bf15f7ab`,
   slot 553 : la marche lit le NEW du bipède (`ti=35`, génération 1, chunk 14 paquet 1094), puis
   ses records delta sous `ti=20` pendant 11 s, dans des trames qu'elle ne ferme pas, jusqu'à ce
   que la table redonne `ti=35`. La règle de l'ancrage de 2.7.b (un slot que la marche a lu
   écarte l'en-tête ancré) écartait alors les vraies positions du joueur : 605 positions perdues,
   un trou de 5,6 s dans sa trace, quatre tirs sans tireur (et, avant 2.7.d, les lectures des
   huit lecteurs du même corps). Corrigé en 2.7.d2 : dans une trame que la fermeture ne prouve
   pas, seul un record de la marche du même archétype écarte l'en-tête ancré ou l'enregistrement
   de la passe. La cause dans la table (un NEW lu dans une trame non fermée qui ne reste pas
   dans la table ?) n'est pas instruite : elle relève de la marche, pour la campagne de grammaire.

## 7. Journal

- 2026-10-03 : plan écrit par la session de l'étape 1, à partir de l'ADR 0037, de l'analyse du
  2026-10-01 (§3.3), du rapport de cartographie (§1.3, §1.6, §3.3, §3.6, §3.8), de l'état de la
  campagne au 2026-10-03 et des décisions de l'utilisateur ; à valider par l'utilisateur.
- 2026-10-03 : plan validé par l'utilisateur (« Oui tu as mon accord », en réponse à la présentation
  du plan et à la demande d'accord de fusion de l'étape 1) ; le GO d'exécution daté se confirme après
  la fusion de l'étape 1.
- 2026-10-03 : GO de l'utilisateur (« je t'ai dit que je te le donnais ») ; branche `feat/ri-etape2`
  créée depuis `origin/feat/v75` = `67c379fc1` (amont désactivé), dans le worktree de l'étape 1.
  Ouverture du lot 2.0.
- 2026-10-03 : lot 2.0 fusionné dans 2.1 (décision d'exécution, consignée au lot : sans le
  distributeur, l'en-tête n'aurait pas de consommateur de production) ; `IDLowBits` relu sur pièces
  (`varwidth.go`) : 13 est la valeur présumée de l'image statique, la marche des morts d'objet la
  calibre. Ouverture du lot 2.1.
- 2026-10-03 : lot 2.1 DIFFÉRÉ PAR LE PLAN : le lot L4a de la vague 1 de la campagne touche
  `movement_states.go` (renommage d'une constante dans `trame()`) et `lecteur.go` (champ
  `etatComplet`) ; la ligne du §1.3 qui disait « aucun » est corrigée. Reprise à la fusion de la vague
  1, que la campagne signale. Ouverture du lot 2.6, indépendant.
- 2026-10-03 : lot 2.6 CLOS (cf. le lot). Premier essai dans `grammar` même : refusé à la compilation
  des tests (cycle d'imports, découverte 1) ; sous-paquet feuille `grammar/signaux`. Preuve à
  différence nulle passée sur le binaire du lot contre la passe de référence (`replay-equiv` 20/20,
  faits 20/20 et killsource 19/19 à l'octet). Baseline des tests de la CI : 9 lignes relocalisées
  de `objectives` vers `grammar/signaux` (mêmes noms), datées dans `scripts/check_test_baseline.sh`
  — oubliées au premier commit du lot, la CI les aurait refusées. La campagne confirme que LP et les naissances par la
  vue A n'ont pas démarré (la vague 2 attend la fusion de la vague 1, une recuisson et le GO de
  l'utilisateur) : les lots 2.2 et 2.3 peuvent partir ; LU et LS prendront `marchLocateStrict`,
  `facts/killsource/walk.go`, `object_deaths_march.go` et la signature du slot 123, LP la lecture du
  bloc de type 1 et `keyframe_world*.go` — à signaler si un lot de ce plan doit y toucher.
- 2026-10-03 : lot 2.1, PREMIER TEMPS fait (2.1.0, 2.1.1, marque de 2.1.3), convenu avec la campagne :
  fichiers neufs et fichiers de la marche seulement. Différence nulle sur le binaire du lot contre la
  référence du lot 2.6 (`replay-equiv` 20/20 décodés depuis le film, faits 20/20 et killsource 19/19
  à l'octet) ; G-film, archlint, G-race, vet (avec et sans `research`), lint verts ; empreinte de la
  grammaire régénérée à révision constante. Les lots 2.2 et 2.3 passent devant le second temps
  (règle d'ordre du §3 : le reste de 2.1 attend la fusion de la vague 1).
- 2026-10-03 : CI verte au niveau job sur `3be61faf3` (premier temps de 2.1, run `37127379956`).
- 2026-10-03 : signal « machine calme » de la campagne (vague 1 décodée) : la passe de preuve de 2.2
  est arrêtée pour jouer la mesure de performance de l'étape 1 (critère 4), qui l'attendait : aucune
  régression (détail au plan de l'étape 1, qui est CLOS) ; campagne prévenue de la fin de la mesure.
- 2026-10-03 : lot 2.2 CLOS — 2.2.1 fait et prouvé (cf. le lot), CI verte au niveau job sur `6063f13b5`
  (run `37133432247`) ;
  2.2.2 statué `[!]` (découverte 8),
  décision de l'utilisateur demandée. Décisions d'exécution 1 à 7 écrites au lot avant le code ;
  la décision 2 corrige le premier temps de 2.1 (un test de 2.1 ne passait que par des marques de la
  phase des images-clés : l'accroupissement n'est jamais traversé dans les trames de la bobine
  `000d5950` ; le test prend désormais le bouclier). Différence nulle prouvée. ADR 0037 amendé
  (IR-3 : distribution de la phase des images-clés, corps lus seulement par un canal, film sans
  registre ; IR-4 : intérêts par phase). Suite : lot 2.3 (canaux de tête de vue A).
- 2026-10-03 : lot 2.3 fait et prouvé (cf. le lot), clôture à la CI verte du commit du lot. Décisions
  d'exécution 1 à 4 écrites au lot avant le code : les phases d'une distribution sont celles que ses
  canaux lisent (`CanalDImageCle`, `CanalDesTrames`, `CanalDesTetes`), la tête se range dans la vue
  A (la marche complète la range aussi pour les listes localisées), les décodeurs partent de la tête
  rangée, chaque point d'entrée est une distribution à un canal. Deux instruments de recherche anciens
  (retours rejeu : `m4b_compteur_research_test.go`, `p4_entites_ti9_research_test.go`) ajustés
  mécaniquement au contexte de `ScanFireEvents` ; campagne prévenue. Suite : la mesure M (machine
  calme, signal de la campagne) ; le second temps de 2.1 à la fusion de la vague 1.
- 2026-10-03 : lot 2.3 CLOS (CI verte au niveau job sur `2aae5c7e1`, run `37137855861`). Mesure M
  CLOSE (cf. M.1 et M.2) : aucune régression (durées de −0,9 % à +0,3 % ; l'écart de pic du BTB est
  le calage du ramasse-miettes, découverte 9) ; balayages ancrés et d'objets du monde = 63 % d'une
  cuisson du BTB, donc 2.4 et 2.5 mutualisent (DT2-5). TOUS LES LOTS RESTANTS ATTENDENT LA CAMPAGNE
  (report par le plan, §3) : le second temps de 2.1, 2.4 et 2.5 attendent la fusion de la vague 1 ;
  2.7 ne peut pas se finir avant LU (2.7.a) et LS (2.7.c), et un lot commencé se finit (règle 2 du
  contrat) ; 3.1 suit 2.7.c, 3.2 clôt. 2.2.2 attend la décision de l'utilisateur. Campagne
  prévenue de la fin de la mesure ; elle annonce la fusion de la vague 1 dans l'heure (après la
  relance d'un job de CI). Reprise à son signal : `feat/v75` dans cette branche, passe de référence
  re-figée, puis le second temps de 2.1.
- 2026-10-03 : vague 1 de la campagne fusionnée dans `feat/v75` (`2393d7db7`, grammar-2026-10-03.2),
  refusionnée ici (`cc395eb7f`) : conflits sur les deux empreintes seulement, prises de `feat/v75`
  puis régénérées à révision constante ; G-film et archlint verts. PREUVE DE LA FUSION : passe du
  binaire de la campagne (`2393d7db7`) contre celle du binaire fusionné, machine laissée par la
  campagne — digests `replay-equiv` 20/20, faits 20/20 et killsource 19/19 identiques à l'octet,
  tous décodés depuis le film : les lots 2.6, 2.1 (premier temps), 2.2 et 2.3 restent à différence
  nulle par-dessus la vague 1. Références d'équivalence re-figées sur la tête fusionnée (passe
  `ri30`, références de la vague 1). Ouverture du second temps de 2.1 (décisions 1 à 5 au lot,
  écrites avant le code ; le retrait de `lecturesDeComposant`, fichier de la campagne, convenu avec
  elle à trois conditions).
- 2026-10-03 : lot 2.1 fait et prouvé (cf. le lot), clôture à la CI verte du commit du lot ; les
  trois conditions de la campagne sont tenues (paire et constante, faits identiques par film,
  mutation rejouée rouge sur le nouveau site) et elle en est prévenue. ADR 0037 amendé (IR-2 : les
  états de mouvement et le tir continu sont deux canaux du distributeur). Suite : lot 2.4
  (récupération ancrée mutualisée), que la fusion de la vague 1 débloque.
- 2026-10-03 : lot 2.1 CLOS (CI verte au niveau job sur la fusion et sur le commit du lot). Lot 2.4
  ouvert sur la même base (`feat/v75` n'a pas bougé, référence : passe `ri31`) ; décisions 1 à 4
  écrites au lot avant le code, après la mesure (l'ancrage domine les neuf lecteurs ancrés, la
  marche des corps est bon marché) et la preuve d'égalité des paramètres sur les vingt films
  (2.4.2). Lot fait et prouvé (cf. le lot) : un ancrage par film au lieu de neuf, cuisson plus
  courte de 13 à 16 %, la seule différence publiée étant le compte déclaré de
  `repli_bande_bipede_comblee`. Partie de 2.4.1 en attente : la marque « récupéré » et le registre
  (découverte 10, même question que 2.2.2). ADR 0037 amendé (IR-6). Campagne prévenue avant la
  passe et la mesure, et à leur fin.
- 2026-10-04 : lot 2.4 CLOS (CI verte au niveau job sur `057c0cffd`, run `37150564223`) ; la
  question de la marque « récupéré » est posée à l'utilisateur (options : rester hors du registre
  jusqu'aux lots de comportement de 2.7, qui ordonneront ces lectures après la grammaire — recommandé
  —, ou les inscrire « devant la lecture » en relevant le cliquet). Ouverture du lot 2.5 sur la même
  base (`feat/v75` n'a pas bougé, référence : passe `ri24a`), par la mesure.
- 2026-10-04 : DÉCISION DE L'UTILISATEUR (« option A », réponse à la question du point d'étape) : les
  lectures heuristiques qui décident devant une lecture de la grammaire restent hors du registre
  jusqu'à 2.7 ; item 2.7.d ajouté, 2.2.2 et la partie « marque et registre » de 2.4.1 statués `[~]`
  vers lui (découvertes 8 et 10 closes par la décision). La campagne lance la recuisson du parc
  local (binaire de `feat/v75`), puis la vague 2 (LU, LS, LP, naissances par la vue A) sur GO de
  l'utilisateur : elle préviendra avant de toucher un fichier ; les fichiers de 2.5 (créations,
  pistes, poses) ne sont pas dans sa liste.
- 2026-10-04 : lot 2.5 fait et prouvé (cf. le lot) : décisions 1 à 4 écrites au lot après la
  mesure (profil CPU d'une cuisson du BTB et sonde temporaire, retirée) ; pistes en une passe sur
  l'union des bandes et créations en une passe, mémorisées ; différence nulle contre la passe du
  lot 2.4 ; références d'équivalence re-figées (le compte déclaré de 2.4 y entre). La mesure de
  durée du critère 4 attend une machine calme : la vague 2 de la campagne décode par intermittence
  pendant plusieurs heures et signalera sa fin. Après 2.5, tous les lots restants attendent la
  campagne (2.7.a et 3.1 : LU ; 2.7.c : LU et LS ; 2.7.b et 2.7.d avec eux ; 3.2 clôt).
- 2026-10-04 : avant la fusion des lots 2.1 à 2.6 dans `feat/v75`, la liste de livraison relève que
  la mise en commun de 2.4 et 2.5 n'a pas de garde-rail qui l'empêche de se défaire (règle 6) :
  ratchet des appelants des primitives de relevé ajouté à 2.5.1 (cf. le lot). Porte locale
  (`make gate-push`) verte sur `b55533d24` ; le run de CI de `e722585ae` n'avait échoué que sur le
  test d'empreinte, régénérée par `b55533d24`.
- 2026-10-04 : FUSION des lots 2.1 à 2.6 dans `feat/v75`, en avance rapide jusqu'au commit qui porte
  cette entrée, sur accord de l'utilisateur (« Tu as mon accord »). CI verte au niveau job sur
  `b55533d24` (run `37191934414`) et sur `1b94fad1b` (run `37193642578`) ; `make gate-push` vert sur
  `b55533d24`, le commit suivant n'ajoute qu'un test, passé avec `go vet` et `golangci-lint` du
  paquet. La campagne a été prévenue avant (son intégrateur ne démarre pas avant plusieurs heures) et
  le sera après ; sa branche `feat/rejeu-vies-bots` (schéma 78) se reprendra sur cette tête : si elle
  change les digests, les références d'équivalence se régénèrent sur la tête fusionnée (les deux
  changements s'additionnent, aucun côté n'est juste seul). Restent ouverts : le
  critère 4 de 2.5 (machine calme), puis 2.7, 3.1 et 3.2 (attente de LU et LS).
- 2026-10-04 : CI de `feat/v75` verte au niveau job sur la fusion (`083e1a4bc`, runs `37203461920`
  en push et `37203466501` en PR). La campagne a fusionné par-dessus la correction des vies de bots
  (`feat/v75` = `6fa631df0`, schéma du document 77 → 78, aucune révision de couche ne monte) ;
  `feat/ri-etape2` est avancée sur cette tête. Les références d'équivalence ne sont PAS re-figées :
  leur ligne `artifact` diffère par le schéma ; elles se re-figent à l'ouverture du prochain lot,
  par la passe de référence du §1.2 sur la tête fusionnée du moment (une passe maintenant serait à
  refaire, et la republication du parc par la campagne occupe la machine).
- 2026-10-04 : la vague 2 de la campagne est assemblée sur `6fa631df0` (`feat/campagne-grammaire`
  `bd9193d16`, fusion dans `feat/v75` sur accord de l'utilisateur) : LU retenu (localisateur unique,
  différence nulle, `grammar/localisateur.go`), LT retenu (règle de tête de liste, `grammar.Rev` =
  `grammar-2026-10-03.5`), LS RETIRÉ (ses ordres par site sont mesurés, pas lus dans le jeu), LP et LN
  non retenus ; `killsource.Rev` et le schéma du document inchangés. Conséquence pour ce plan : 2.7.c
  n'attendait LS que pour ne pas écrire `facts/killsource/*` en même temps que lui (aucune dépendance
  de contenu) ; LS retiré, tous les lots restants n'attendent plus que la fusion de LU (§1.3 mis à
  jour). Si la recherche de la campagne sur la localisation haute fréquence devient un lot qui touche
  `grammar/localisateur.go` ou `facts/killsource/*`, elle préviendra avant. Suite, LU fusionné : fusion
  de `feat/v75`, passe de référence (références d'équivalence re-figées : schéma 78, LT), puis 2.7.a.
- 2026-10-04 : OUVERTURE DU LOT 2.7 — LU et LT fusionnés dans `feat/v75` (`87cdfa761`). Fusion de
  `feat/v75` dans `feat/ri-etape2` (`45184faf4`, sans conflit) ; passe de référence `v75w2` sur la
  tête fusionnée : vingt films décodés depuis le film, aucun échec ; écarts tous venus de `feat/v75`
  (`artifact` sur les vingt : schéma 78 et LT ; `movementStates.stats` sur dix-sept et
  `continuousFire.stats` sur treize, `continuousFire` lui-même sur deux : localisation des listes par
  LT) ; références re-figées (`0355c46ab`), killsource de référence `ks_v75w2` (dix-neuf témoins, code
  0). Campagne prévenue avant et après. Elle annonce un lot « lire la vue A jusqu'au bout »
  (décision de l'utilisateur du 2026-10-04) qui se posera sur la structure : 2.7 et 3.1 ne touchent
  ni la lecture de la vue A ni `distribuer_tetes.go`, et l'ordre de 2.7.c se calera avec ce lot pour
  que killsource ne change qu'une fois. 2.7.a commence par la mesure (instrument
  `grammar/morts_marche_unique_research_test.go`, les deux marches sur les huit films du corpus qui
  portent des véhicules).
- 2026-10-05 : 2.7.a ÉCRIT (`b093ef10b`, local), GATE DE CORPUS MIXTE, DÉCISION POSÉE À
  L'UTILISATEUR. Équivalence (`ri27c`) propre : mouvement et tir identiques, killsource identique à
  l'octet, morts de véhicule 142 → 148. Gate (19 témoins contre `87cdfa761`) : sortie 1 — FAUX sur
  `084a804d` et `e5adf7b2` (une action hors vie de plus chacun), PERTE sur `60ae07c4`, `a349fea8`,
  `a521164d`, `11de8353`, `4f77afc1` (épisodes à bord, tirs posés sur un véhicule, deux rafales,
  morts lues), gains au banc (`4f77afc1` 112 → 95 et `11de8353` 8 → 3 actions hors vie). La
  variante sans la récupération des listes perd davantage : la récupération reste. Les onze morts
  de `4f77afc1` que le recensement n'apparie pas ne viennent pas de la récupération : véhicules nés
  entre deux images-clés, liés par NEW dans le monde de la marche, absents du recensement par
  images-clés. Cause des pertes sur les formats anciens : la marche à huit vues de production lit
  sous la largeur MPP mesurée sur le film (`build_vehicles.go`, `gwWidthsForFilm`), la marche des
  trames sous celle du contexte (règle de l'utilisateur du 2026-10-02). Deux parts : les listes que
  la marche ne localise pas (le lot de la campagne « lire la vue A jusqu'au bout » devrait les
  rendre) et les records de la vue B qui déraillent derrière un véhicule mal découpé (aucune lecture
  générale connue ne les rend sur ces versions ; découverte 11). Options posées : garder la lecture
  actuelle et reprendre après ce lot de la campagne (recommandée), basculer avec les pertes, ou
  admettre la largeur mesurée pour la marche sur ces versions (exception à la règle du 2026-10-02).
  La campagne lance ce lot depuis `87cdfa761` (`feat/cg3-vue-a`) et y garde verte la marche à huit
  vues de production ; 2.7.a n'entre pas dans `feat/v75` avant lui. Elle mesure en parallèle, sans
  code de production, le découpage 8/3 des formats anciens par double preuve (fermeture au bit,
  châssis du jeu installé).
- 2026-10-05 : l'utilisateur réaffirme que « le jeu dans sa version actuelle sait lire tous les
  films » et que « les films contiennent eux-mêmes leur index de décodage ». L'argument tiré des
  notes de mise à jour de Halo Support (films « invalidés ») est retiré. À sa demande, recherche
  Ghidra de la donnée du film qui piloterait le bloc MPP ; la campagne a arrêté la sienne pour ne
  pas doubler. Résultat (découverte 12) : rien trouvé entre les formats 25 et 27. 2.7.a reste hors
  de `feat/v75`, avec la lecture actuelle des morts gardée.
- 2026-10-05 : à la demande de l'utilisateur, un agent d'enquête (worktree dédié, Ghidra en
  lecture seule, sans passe de corpus, sans commit) a lu les deux blocs : découverte 13. Worktree
  et branche de l'agent retirés (aucune jonction, `git status` vide). Résultat transmis à la
  campagne. 2.7.a reste hors de `feat/v75` ; la suite dépend de l'utilisateur.
- 2026-10-05 : DÉCISION DE L'UTILISATEUR (« Oui je valide cette piste ») : le bloc MPP des
  anciens films se lit d'après la taille d'état de création que le film déclare. L'observation
  dynamique avec Cheat Engine est inscrite au backlog. Item 2.7.a0 ajouté ; la campagne le confie
  à ce plan sous trois conditions (son gate 2, la provenance présumée par mesure, sa double preuve
  comme oracle). Tailles courantes relues dans Ghidra (`vtable+0x20` des neuf descripteurs).
  Ouverture de 2.7.a0.
- 2026-10-05 : 2.7.a0 ÉCRIT (`3ee8e7bf2`, `a06ecfd06`, local) ; GATES JOUÉS.
  - *Preuve `ri27d` contre `ri27c`* :
    - killsource identique à l'octet sur les 19 témoins ;
    - format 27 et `50247b26` (non déclaré) : seule l'étape `artifact` change, par la télémétrie
      des révisions ;
    - films déclarés 8/3 : `birthLoadouts` ×4 à ×20 (`084a804d` 16 → 365, `1c4c63c2`
      27 → 571), `movementStates` +0,3 à +14 %, `continuousFire` ×1,5 à ×4 ;
    - `60ae07c4` : `birthLoadouts` 6 → 5.
  - *`50247b26`* (format 20, majeure 31) : DISCORDANT (173 records de la clé, dont 53 ti=38 à
    n1=92, soit taille courante − 12). Chemin calibré gardé et averti.
  - *Gate 2 de la campagne* (carte v2, sans puis avec `-mpp-declare`, 20 films) :
    - aucun film en baisse nette ;
    - sains 313 542 → 397 824 (+84 282), utiles sains +1 855 974 ;
    - 428 sains perdus en brut, dont 347 sur `1c4c63c2` ;
    - factices des gains au bit : 3,1 %, dont 8,5 % sur `1c4c63c2`.
  - *Gate de corpus contre `87cdfa761`* (cumul 2.7.a et 2.7.a0) : sortie 1.
    - FAUX sur `084a804d` (V-3 92 → 96) et `e5adf7b2` (V-3 10 → 11) ;
    - PERTE sur les autres films anciens et sur `4f77afc1` (ce dernier vient de 2.7.a) ;
    - gains P-1 (paquets fermés ×2,5 à ×5), V-6 de `084a804d` 6 → 1, V-3 de `4f77afc1`
      112 → 95.
  - *Suite* : instruction par un agent d'enquête (placement des deux bits de fin, pertes du
    gate 2 par famille, écarts du rejeu), condition de clôture de la campagne. Non
    fusionnable en l'état.
- 2026-10-05 : INSTRUCTION DES PERTES (agent d'enquête ; pièces sous `scratchpad/agentA/` de la
  session).
  - *8/3 tient.* Dix-huit placements des deux bits ont été testés, et aucun record authentique ne
    les départage : sur 93 096 records d'image-clé et 7 512 NEW à identifiant connu, R(2), le
    compte, d4d0 et G3 valent 0, l'index 0 ou 1. Les 3 125 NEW à champs non nuls ont tous un
    identifiant inconnu. Retirer une porte est réfuté par `n2`.
  - Le champ de tête est un champ de drapeaux ; la taille de structure (0x60 octets) désigne
    « R(2) absent, index R(5) », équivalent à 8/3 en lecture (provenance ajoutée à
    `profile/mpp_declare.go`).
  - *Les 428 pertes du gate 2* :
    - 418 étaient de faux sains de 9/5 : liste ouverte par la recherche « fermeture » sur un
      faux NEW d'archétype MPP ;
    - les 10 autres viennent d'un monde empoisonné : une lecture contredite qui lie un faux NEW
      ou délie un faux DEL ;
    - aucune ne vient d'un composant non porté ni du placement des bits.
    La campagne prend en lot « LR » les deux corrections qui en découlent : la règle de lecture
    du jeu « compte ≥ 5 → échec », et un repli de début de liste qui ne modifie plus le monde.
  - *Côté rejeu, ce qui vient de 2.7.a0 est une correction* : birthLoadouts de `60ae07c4`,
    weaponChanges `taken` → `swapped` de `084a804d`, postures de `11de8353`.
  - Les V-3 du banc sont des FAUX POSITIFS : tirs d'arme de véhicule par un occupant dont la
    piste s'arrête à la montée.
  - Perte probable de 2.7.a, non établie : le trajet du slot 608 de `084a804d` est retiré au lieu
    d'être tronqué à la mort, et ses 3 tirs perdent leur slot.
- 2026-10-05 : BANC DE VÉRITÉ, V-3 CORRIGÉ (accord de la campagne, qui l'utilise aussi comme
  juge) : un trajet du slot à bord d'un véhicule fait partie de sa vie, rattaché par slot quel que
  soit le siège (règle de l'utilisateur du 2026-09-21). Test
  `TestHorsVie_UnTrajetDuSlotFaitPartieDeSaVie`, rouge sans la correction.
- 2026-10-05 : PUSH de `feat/ri-etape2` (`ced770753`, la campagne y prend `-mpp-declare` pour LR) ;
  CI rouge au lint (`cmd_fermeture/mpp_declare.go` sans l'étiquette `research`), corrigé
  (`46b51efe7`), lint vert. GATE DE CORPUS, BANC CORRIGÉ (19 témoins contre `87cdfa761`) : les
  FAUX V-3 de `084a804d` et `e5adf7b2` disparaissent ; `4f77afc1` passe FAUX (V-3 4 → 5 :
  capacités d'image-clé du slot 737 hors de son trajet raccourci). Seconde passe restreinte aux
  huit films en baisse, artefacts gardés, et lectures d'occupation relevées par un instrument
  jetable (non versionné) : cause trouvée (découverte 15), corrigée côté rejeu (`953401feb`,
  item 2.7.a). Gate complet après le correctif : sortie 1, AUCUN FAUX ; onze témoins sans écart,
  huit en baisse, banc `ok` partout (`4f77afc1` V-3 4 → 2).
- 2026-10-05 : INSTRUCTION DES 334 LIGNES EN BAISSE des huit films (gate `27g`), par famille :
  - *Postures* (169 lignes, six films) : CORRECTIONS. Les durées baissent parce que des
    transitions sont désormais lues (fin de sprint, fin d'escalade) ; ex. le sprint du slot 513 de
    `084a804d` 7606-7714 devient 7606-7609 puis 7698-7714, l'accroupi de 59 s du slot 685 (en
    sprintant) disparaît. Invraisemblables 118 → 94 ; trois nouvelles (découverte 14).
  - *Changements d'arme* (12 lignes) : CORRECTIONS. 43 « prises » deviennent des « échanges »
    (l'arme de départ est lue) ; deux « lâchers » disparaissent, dont celui de `084a804d` (slot
    521 à 9319), relu : l'emplacement passe de rien à rien, reclassé en ré-annonce.
  - *Tir continu et rafales* (39 lignes) : AMÉLIORATIONS que le gate compte à rebours. Moins de
    trous (`084a804d` : 27 620 → 8 815 paquets non lus, temps tenu en trou 899 → 400 s), trois fois
    plus de rafales lues ; un compteur de trous qui baisse est une baisse de manque.
  - *Dotations de naissance* (5 lignes) : AMÉLIORATIONS. Lues 0 → 5 à 210 par film, publiées 0 →
    5 à 204 ; les compteurs « non affichable » montent avec ce qui est lu.
  - *Compteurs des postures* (19 lignes) : les refus croissent avec les records lus (`111fa685` :
    230 131 → 251 150 records, refus 25 → 50).
  - *Véhicules* (78 lignes) : trajets lus 269 → 326 sur les sept films à véhicules (58 nouveaux,
    dont un trajet de 163 s sur `111fa685` borné par la montée et la destruction lues ; 4 lectures
    de la base non relues) ; trajets de repli 196 → 156, la plupart remplacés par la lecture aux
    mêmes bornes ; faux passagers retirés (`084a804d` V-6 6 → 1) ; fins de véhicule mieux connues
    (`084a804d` détruits 35 → 48) ; morts lues sans vie recensée (`4f77afc1` 0 → 11, véhicules nés
    entre deux images-clés). PERTE RÉELLE RESTANTE : 13 tirs d'arme de véhicule de `084a804d`
    perdent leur tireur par la primauté de la lecture (découverte 16), 7 en gagnent.
  Décision demandée à l'utilisateur : admettre ces pertes (gate en base), et la forme de la
  primauté (découverte 16). La fusion de 2.7.a attend toujours le lot « vue A » de la campagne.
- 2026-10-05 : DÉCISIONS DE L'UTILISATEUR (§2) : baisses ADMISES ; primauté PAR JOUEUR, écrite
  (`e127e90fb`). Gate complet `27h` (19 témoins contre `87cdfa761`) : sortie 1, aucun FAUX, banc
  `ok` partout ; onze témoins sans écart ; `084a804d` 44 → 25 lignes en baisse, toutes dans les
  familles admises (le trajet de repli du slot 745 revient, V-6 reste à 1) ; tirs de véhicule
  rattachés 194 → 198, seuls les 3 du 916 (slot 608, chevauchement) restent sans tireur ; les
  sept autres films inchangés depuis `27g`. Campagne prévenue avant et après chaque passe. Reste
  pour clore 2.7.a : fusion de `feat/v75` après le lot « vue A » de la campagne, remesure, gate
  de corpus, `make gate-push`, accord de fusion.
- 2026-10-05 : DOUBLE PREUVE DE LA CAMPAGNE reçue (oracle de 2.7.a0, pièces dans son scratchpad
  `mpp-dp/`), ses quatre réserves instruites ou rangées (item 2.7.a0) : `b429a7d3` déclare 8/3 et
  ne perd que quatre faux sains de 9/5 sans record utile ; le ratchet de fermeture d'image-clé,
  rejoué sous le découpage déclaré (+107 sur les cinq archétypes objet, baisses par ligne sur
  `ti=42`), s'aligne en 2.7.c avec killsource. Deux mesures d'un film et deux instruments jetables
  (non versionnés, retirés). ADR 0037 IR-7 complété. 2.7.a0 CLOS. Ordre de fusion convenu avec la
  campagne (§1.3) : V1, puis 2.7.a et 2.7.a0, puis LR (qui ne tient que sous le découpage
  déclaré), puis V2.
- 2026-10-05 : RELECTURE DU LOT « VUE A » V1 DE LA CAMPAGNE avant sa fusion, à sa demande (diff
  `87cdfa761..3bacfadeb`, 90 fichiers) : deux relecteurs à contexte frais en parallèle (règles du
  projet ; équivalence de la lecture unique et couverture des tests), lecture seule, plus mes
  vérifications de la structure (`lecture.Paquet.VueA` sans logique, façade et `replay` intacts,
  garde-fou de lecture unique, ADR 0037). Aucun P0, sortie identique confirmée par les deux.
  Constats recevables : 3 du premier, 12 du second (dont le champ mort `finVueA`, trouvé par les
  deux) ; transmis triés à la campagne : quatre familles à corriger avant fusion (recopie de
  `FUN_140c1e9d4`, commentaires faux, nombres magiques, aiguillages sans justification), trois à
  trancher (troisième portage de `damage_aftermath` dont une copie diverge, D-LN-2 de la campagne ;
  deux gardes recopiées), une pour V2. La campagne corrige avant de fusionner. Découverte 17. À la
  fusion de V1 dans cette branche : `canal_des_morts.go` doit prendre `listeAnnoncee(&p.VueA)` au
  lieu de `p.VueA.Etat == lecture.VueArretee` (une vue A lue jusqu'au bout est terminée même avec
  des messages).
- 2026-10-05 : RONDE 2 de la relecture de V1, sur les seules corrections (`3bacfadeb..020d0e9cc`),
  un relecteur à contexte frais : huit corrections sur dix tiennent (23 conditions vérifiées). Reste
  un P1 de commentaire (la phrase « seule lecture complète de la vue A » ignore la lecture en
  chaîne de killsource, `eventchain.go`) et un P2 (helper de balayage d'archlint sans garde-rail).
  Bornes de la relecture : pas de ronde 3, le P1 restant est porté à l'utilisateur et relu par moi
  à son correctif (commentaire seul). Item 2.7.c complété : la chaîne de killsource rejoindra la
  vue A unique.
- 2026-10-05 : FUSION DE `feat/v75` DANS LA BRANCHE, après le lot « vue A » V1 de la campagne
  (`5bc1fd938`, avec le correctif de rejeu du schéma 79, puis `65c99b669`, goldens killsource) :
  `de99f7fbf` et `a87c90904`. Conflits : révisions (rangs de fusion `grammar-2026-10-06.2` et
  `profile-2026-10-06.2`, chroniques réunies), empreintes régénérées (killsource et objectives à
  révision constante), fixtures de contrat du schéma 79 régénérées, identiques à `feat/v75` hors
  chaînes de révision. Le canal des morts reconnaît un paquet à événements par `listeAnnoncee`.
  `go vet`, tests du décodeur, d'archlint et des gates, lint : verts. Gate de corpus contre
  `5bc1fd938` : les mêmes 315 lignes en baisse que le gate admis, aucun FAUX, banc `ok` partout.
  Banc killsource sur films réels : rouge sur `feat/v75` avant la fusion (découverte 18), vert après
  la régénération de la campagne. 2.7.a CLOS. Reste avant la fusion dans `feat/v75` : CI de la
  branche, `make gate-push`, accord de l'utilisateur.
- 2026-10-06 : FUSION DE 2.7.a ET 2.7.a0 DANS `feat/v75` = `8dfadd07e` (avance rapide depuis
  `65c99b669`, accord de l'utilisateur du 2026-10-05 « fusionne au vert ») : CI au niveau job verte
  sur `8dfadd07e` après relance (la première exécution avait été annulée par un arrêt du runner
  GitHub, sans échec réel), gitleaks et pré-contrôle de déploiement verts, `make gate-push` vert
  (suite rejouée : la première avait été faussée par la mise en veille du poste). Campagne prévenue
  avant et après ; elle rebase LR sur cette tête. Suite du plan : 2.7.b.
- 2026-10-06 : FUSION DE `feat/v75` (`fed1efed2` : LR de la campagne, page Tendances) dans la
  branche (`db44b91e5`), puis passe de référence v75w3 : `replay-equiv -update` sur les 20 films,
  tous décodés depuis le film, et killsource json sur les 19 témoins. Références d'équivalence
  re-figées (`0e0143148`) : elles dataient du 2026-10-04 et prennent 2.7.a, 2.7.a0, la vue A V1 et
  LR. killsource identique à la passe d'avant LR, sauf la ligne de diagnostic de calibration
  (7 témoins sur 19). Faits de la passe mis de côté (`film_facts_v75w3`). Mesure de 2.7.b sur les
  20 films en trois exécutions (la dernière avec la récupération des listes, la répartition de
  l'ancrage seul et la fidélité selon le localisateur) : décisions d'exécution 1 à 5, découvertes
  19 à 21. Campagne prévenue avant et après la passe et la mesure.
- 2026-10-06 : 2.7.b ÉCRIT, GATE INSTRUIT. Premier gate de corpus complet (19 témoins, contre
  `fed1efed2`) : 220 actions hors vie (V-3) dues aux records de cadavres et aux annonces
  d'emplacements vides → décisions 6 et 7 (`4f4049ebd`). Deuxième gate : V-3 résiduels instruits
  témoin par témoin (lectures de naissance, une remise à zéro de fin de manche, un record déchet
  de génération 0 dans une queue opaque), une lecture de grenade perdue sur `c75f33b8` (paquet
  22:376 : la marche part d'un NEW du milieu de la liste) → mesures sur les 20 films (garde des
  générations, ancres seules par verdict et par début de vue B) → décisions 8 et 9. Troisième
  gate : plus aucune lecture perdue contre la base hors des lâchers d'arme inconnue ; FAUX
  restants instruits (item 2.7.b), à faire admettre. Découvertes 22 à 25 ; liste des paquets à
  début choisi par la fermeture remise à la campagne pour son lot V2. Campagne prévenue avant et
  après chaque passe.
- 2026-10-06 (soir) : relecture de la vue A V2 et V3 de la campagne rendue (un relecteur Opus en
  contexte frais : aucun P0, un P1 dans `movement_states.go`, un P2 consigné, découverte 26) ;
  campagne fusionnée dans `feat/v75` (`2707fdb31`), reprise dans la branche (`c16708f2c`, rang
  `.5`, P1 corrigé), CI de la branche corrigée (paramètre inutile de `maskHas`, golden des formes).
  Population des huit lecteurs avec et sans V2 (+806 / −51), chiffres versés au handoff de la
  campagne ; gate de corpus contre `2707fdb31` : familles admises, à l'identique. Lot lint : fusion de
  `feat/v75` dans sa branche (`9321554d6`), CI et `make gate-push` en cours avant son avance rapide.
- 2026-10-06 (soir) : FUSION DU LOT LINT DU DÉCODEUR DANS `feat/v75` = `9321554d6` (avance rapide depuis
  `2707fdb31`, accord de l'utilisateur du même jour « Oui, fusionne au vert », découverte 17) : fusion
  de `feat/v75` dans sa branche sans conflit hors du journal, `golangci-lint` sans remarque, CI verte
  au niveau job, `make gate-push` vert hors de deux dépassements dus à la charge (`platform/duckdb`
  au plafond de 300 s, test de rapport de coût de `killcollector`), verts rejoués seuls. Campagne et
  levelup-dc prévenus avant et après. Ordre convenu ensuite : clôture de la campagne (doc), lot
  « équipes source » de levelup-dc (SchemaDesFaits 6), puis 2.7.b.
- 2026-10-07 (nuit) : 2.7.b CLOS. Reprise de `feat/v75` `b5c9489ef` (`95f28c138` : fixtures du web
  passées au schéma 80 puis régénérées, golden d'empreinte de killsource re-figé ; un épisode de
  disque plein pendant un `go build ./...`, refait proprement), CI verte au niveau job sur
  `95f28c138`, gate de corpus contre `b5c9489ef` identique au gate admis, `KILLSOURCE_FIXTURES`
  vert, passe de référence b6 et références re-figées. Accord de fusion de l'utilisateur (« tu
  pourras fusionner si la CI est verte »). Reste : CI verte sur la tête finale, `make gate-push`,
  avance rapide de `feat/v75`, signal à levelup-dc pour la recuisson unique du parc (avec le
  rattrapage du placement des vies : `killcollector.PlacementRev` monte). Suite du plan : 2.7.c.
- 2026-10-07 (3 h) : FUSION DE 2.7.b DANS `feat/v75` = `4f112add5` (avance rapide depuis `2668848b1`,
  lot Sessions intégré au passage sans conflit ni fichier du décodeur), accord de l'utilisateur
  « tu pourras fusionner si la CI est verte » : CI verte au niveau job sur `4f112add5` (et sur
  `9d37645b3` après relance du seul job de couverture, rougi par le test de chronométrage de la
  découverte 28) ; `make gate-push` vert hors du dépassement de 300 s de `platform/duckdb`, vert
  rejoué seul (242 s). levelup-dc (phase D des équipes, schéma 81) fusionne derrière, puis lance la
  recuisson unique du parc avec le rattrapage du placement des vies. Suite du plan : 2.7.c.
- 2026-10-07 : OUVERTURE DE 2.7.c (« ok oui fait l'étape suivante du plan stp »). Fusion de `feat/v75`
  `94ac8fd68` (`e470a2e3e`). Relecture sur pièces : killsource lit le film en sept endroits, et le
  critère de retrait de l'exception du garde-rail de la couche des faits demande qu'ils descendent
  tous dans la grammaire ; sous-items c0 à c5 écrits. Mesure c0 sur 23 films (les 20 du corpus et
  trois films de référence de killsource) : la marche des trames lit ce que la marche de killsource
  lit (3 030 dead-states communs, contenu publié identique dans toutes les variantes, seule la voie
  technique change) ; la calibration du mot de poignée est aveugle partout ; la vue A unique
  s'arrête au premier kill de chaque paquet, faute de savoir si le 85 porte sa queue — lue sans
  queue, sa fin tombe 1 415 fois sur 1 529 au bit près sur le début de vue B localisé
  indépendamment, et l'exécutable enregistre les deux réglages de la queue à faux par défaut.
  Décisions d'exécution 1 à 3 et 5 écrites ; la décision 4 (kill-events) est demandée à
  l'utilisateur. Découvertes 30 et 31.
- 2026-10-07 : 2.7.c1 CLOS. Décision de l'utilisateur sur les kill-events (« Lire sans la partie
  optionnelle ») consignée (§2, décision 4 de c0, item c4). Lectures de killsource hors de sa
  marche descendues dans la grammaire (`b00ba0989`) ; preuve interrompue à 11 h 10 pour la recuisson
  du parc de levelup-d0 (passes arrêtées, fichiers partiels retirés), `feat/v75` `48fce6602`
  re-fusionnée (`c11f30159`, conflits des empreintes de révision : `feat/v75` a raison, empreintes
  recopiées), puis passes c0 et c1 rejouées après la recuisson (11 h 33 – 11 h 54) : identiques à
  l'octet. c2 et c3 écrits dans l'arbre de travail pendant l'attente, non commités.
- 2026-10-07 : 2.7.c2 ET 2.7.c3 CLOS (un commit). Les morts et la calibration de killsource passent
  sur la marche des trames, sous le découpage MPP déclaré et l'identifiant bas de l'en-tête ; la
  timeline, la bande bipède et son repli sont retirés, avec les entrées de l'ancienne marche dans la
  grammaire. Ratchet de fermeture d'image-clé régénéré sous le découpage déclaré : chaque baisse
  justifiée par la preuve 2 (instrument `ri27c3`). Preuve commune : contenu publié identique sur les
  19 témoins, seules les voies techniques, la couverture de l'artefact et les diagnostics de
  calibration bougent. Le commit de clôture de c1 (`5ee99df38`) emportait par erreur les suppressions
  de fichiers de c2 ; refait sans elles avant tout push (`8aa5c597a`). Découvertes 32 et 33.
- 2026-10-07 : 2.7.c4 CLOS. La vue A lit le message de kill sans sa partie optionnelle et range
  ses messages ; killsource y prend ses kill-events dans la marche de ses morts ; la recherche bit à
  bit descend dans la grammaire en rattrapage compté. La vérification sur 28 films sous la règle
  décidée d'abord (rattrapage après un arrêt seulement) a montré 433 kills réels perdus sur quatre
  films anciens derrière des terminateurs que la marche ne retient pas : seconde décision de
  l'utilisateur (§2), le rattrapage couvre toute trame dont la lecture de la vue A n'est pas établie,
  avant la vue B. Preuve c4b : publication de killsource identique hors deux kill-events gagnés,
  marche nettement meilleure (fermées +24 500 sur 28 films), régression localisée sur les trames à
  kill dont la fin de vue A, fausse après un message de dégâts, est retenue (découverte 34). killsource
  ne lit plus aucun octet : son exception au garde-rail des faits est retirée. Coordination :
  levelup-d0 a fusionné `grammar-2026-10-07`, `SchemaDesFaits` 9 et `SchemaVersion` 84 dans
  `feat/v75` (`292ef56a5`) ; renumérotation à la fusion (2.7.c5). Découvertes 34 et 35 ; 30 instruite.
- 2026-10-07 (soir) : 2.7.c5 CLOS, donc 2.7.c. Quatre refusions de `feat/v75` (`fad38a03c`,
  `0a9a327d7`, `cdd642061`, `92c6a5d6a`) avec renumérotation négociée : `grammar-2026-10-07.2`,
  `SchemaVersion` 87 (levelup-d0 a fusionné 84 puis 86 avant ; 83 et 85 restent sans emploi),
  `SchemaDesFaits` 10. ADR 0037 amendé (IR-6, IR-7, D-2), doc de killsource. Gate de corpus contre
  `feat/v75` : 352 gains, 93 pertes déclarées, banc de vérité expliqué (repli du rattrapage nouveau
  par décision, un repli existant sur trois films) ; une régression probable notée (un trajet de
  véhicule raccourci, découverte 34). `make gate-push` vert, CI verte sur la tête fusionnée. FUSION
  DANS `feat/v75` par avance rapide, accord de l'utilisateur du 2026-10-07 (« Ok tu pourras
  fusionner »). Recuisson du parc et backlog killsource : sur signal de l'utilisateur. Suite du
  plan : 2.7.d.
- 2026-10-07 (nuit) : ouverture de 2.7.d (« vas-y tu peux le faire »). 2.7.d0 CLOS : trois mesures
  sur 28 films. Les positions et les objets du monde peuvent passer derrière la marche (décisions 1 à
  3), avec des corrections par construction (découvertes 37 à 39). Les images-clés NON : la grammaire
  ne lit pas l'état complet du bipède (découverte 36) ; 2.7.d1 statué `[!]`, décision de l'utilisateur
  demandée (recommandation : lot de grammaire d'abord). Suite : 2.7.d2 puis 2.7.d3.
- 2026-10-08 : 2.7.d2, 2.7.d3 et 2.7.d4 CLOS sur `feat/v75` `acfe4851a` fusionnée (arrêts de la vue
  B, fusionnés avant ce lot ; rangs alignés avec levelup-5c : `grammar-2026-10-08`,
  `SchemaVersion` 89 ; `killsource.Rev`, `objectives.Rev` et `SchemaDesFaits` constants). Les gates
  ont fait corriger quatre règles dans le lot : le dead-state qui ne dit pas la mort (découverte
  41), la population des positions (42), la fin du film des socles (43), l'archétype non prouvé des
  trames non fermées (45). 2.7.d reste ouvert par 2.7.d1 `[!]` (décision de l'utilisateur). Suite :
  `make gate-push`, CI, accord de fusion ; puis 3.1 si l'utilisateur tranche 2.7.d1 en faveur d'un
  lot de grammaire d'abord.
- 2026-10-08 (matin) : FUSION DANS `feat/v75` par avance rapide (`2c8b16a47`), accord de l'utilisateur
  (« Ok pour la fusion et le recalcul »), CI verte sur la tête ; checkout principal avancé, pairs
  prévenus (levelup-5c ×2, levelup-d0). RECUISSON DU PARC FAITE : 172 artefacts au schéma 89 (lot des
  arrêts de la vue B compris), 0 erreur, 18 min ; projections rejouées (usage, niveaux d'armes,
  véhicules, Assaut, drapeaux), 0 échec ; serveur local relancé (binaire reconstruit). Backfill
  killsource (`killsource-2026-10-07.2`, lot de la vue B) : décision de l'utilisateur demandée.
- 2026-10-08 (après-midi) : BACKFILL KILLSOURCE FAIT, accord de l'utilisateur (« Ok pour 1 ») :
  serveur local coupé et pairs prévenus avant, `levelup backfill-killsource` sur le parc, 66 min,
  code 0. 1 809 matchs collectés, 189 164 morts écrites, 0 erreur de décodage, 0 erreur d'écriture,
  43 passes non publiables, 26 matchs sans fil des morts ; 48 matchs écartés faute de carte résolue
  (ils restent candidats). Signalements connus : équipe de bots non lue dans `BOT_METADATA` sur 66
  matchs (`HI_1_12_0`, découverte D1 du lot killsource), vies publiées sans identité sur 5 matchs, un
  film à la séquence de chunks trouée. Serveur local relancé. 2.7.d1 rouvert par l'utilisateur
  (« Pour le point 2 met un workflow ultracode dessus ») : première passe à sept agents (comprendre,
  mesurer, concevoir, deux contre-vérifications). Mesure sur 28 films, lecture des images-clés sous
  la portée de l'état complet et forme d'`i0` du jeu : records bipèdes d'image-clé fermés 410 ->
  5 399 sur 10 710, compteur de grenades différent de 4 : 9 204 -> 380, armes égales à la fenêtre :
  386 -> 6 955, aucun autre archétype ne bouge. Les contre-vérifications valident la lecture du jeu
  (huit fonctions posent la portée, pas six ; le `R(2)` final de la queue d'`i0` est lu, seule la
  borne `N` reste d'exécution) et corrigent le plan : deux baisses non adjugées (un film où une
  image-clé ne ferme plus, un format où quelques grenades se dégradent), un gate vide (le golden des
  bobines est joué sans carte et ne voit pas la portée), le critère écrit de la bascule à rejouer
  AVANT tout retrait, la décision du gate objectives à chaque montée de `grammar.Rev`, une mesure de
  coût, la règle d'admission à écrire comme un amendement de 2.7.b, le résidu sous la portée (88
  arrêts, pas 861). Seconde passe lancée (cinq agents : Ghidra, deux mesures, plan réécrit, une
  contre-vérification), aucun code de production ; plan dédié à suivre.
- 2026-10-08 (fin d'après-midi) : seconde passe de 2.7.d1 finie (cinq agents, 74 min, aucun code de
  production). Ghidra : largeur du handle de la queue d'`i0` LUE (13 bits, déjà dans le Go, non
  câblée : le chemin actuel la lit sur 1 bit) ; `i42` porte l'emplacement DÉSIRÉ des deux mains et
  l'identifiant de la demande, pas l'emplacement dégainé ; la marque de portage est une configuration
  des champs de fin d'`i11`, pas un identifiant ; l'état par défaut de `ti=9` n'a aucun lecteur de
  position sous la portée. Mesures : sous la portée, ancres, élections, preuves et équipes
  identiques, carte des trames delta identique à l'octet, seule la fermeture des records bouge (+4 999,
  −10 : fermetures de hasard de la base, adjugées) ; critère écrit de la bascule : la portée seule
  8/599, la portée avec la forme d'`i0` du jeu 369/599 (61,6 %) avec et sans bouchons, l'instrument
  que la doc nomme aveugle depuis le lot 2.3 (0/599 dans tous les cas) ; forme de l'écrivain en delta :
  0 trame saine perdue, +2 au mieux ; ratchet de fermeture en contexte de cuisson sur les sept bobines
  53 -> 666 / 1 368 ; admission : dans les formats 24 à 27, les records non fermés à `n(i22) = 4` sont
  aussi justes que les fermés, les formats 20 et 21 sont faux après `i22` ; la « Dynamo » de la
  fenêtre est le M41 SPNKr lu un bit trop tôt (100 %). Plan dédié réécrit (700 lignes), contre-vérifié
  « partiel » (commandes de gate à corriger, décisions à rendre à l'utilisateur). Demandé à
  l'utilisateur : amendement du critère écrit, une ou deux fusions, règle d'admission, marque de
  portage, valeurs publiées qui changent, coût de la mise en œuvre.
