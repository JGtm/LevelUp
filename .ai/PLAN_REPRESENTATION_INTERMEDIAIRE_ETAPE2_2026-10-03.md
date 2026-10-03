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
> - `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` — §3 (décisions), §6.1 et §6.2 (lots des vagues 1 et
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
| 2.7 (c) killsource | `facts/killsource/*` (LU, LS) | en DERNIER, après LU et LS |

Chaque session prévient l'autre quand elle fusionne dans `feat/v75` ; une découverte qui touche un
fichier de l'autre se signale, ne se corrige pas.

## 2. Décisions

**Utilisateur.**
- 2026-10-02 : la représentation intermédiaire est menée dans une autre conversation, en parallèle de
  la campagne de grammaire.
- 2026-10-03 : la mémoire de la marche des images-clés n'est pas partagée entre la cuisson et
  killsource à l'étape 1 ; elle se reprend à l'étape 2 (lot 2.7 (c) puis 3.1 : killsource devient un
  canal de la même marche, contexte unique par cuisson).
- GO de l'étape 2 : DONNÉ le 2026-10-03 (statut ci-dessus) ; l'ordre des lots qui attendent la
  campagne est réglé au §3 (report par le plan), la mutualisation par DT2-5.

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

### Lot 2.1 — L'en-tête et le distributeur ; canaux des états de mouvement et du tir continu (taille M)
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
- [ ] 2.1.0 L'en-tête (ex-2.0) : `lecture.Provenance` (relue, mesurée, présumée — la table du profil
      —, calibrée sur le film, imposée à la construction) et `lecture.Parametre[T]`, sans logique ;
      `grammar.EnTete` (`IDLowBits`, découpage MPP du format, découpage d'i0) résolu par le contexte ;
      la marche est construite DEPUIS l'en-tête, et le distributeur le rend aux canaux.
- [ ] 2.1.1 `Canal`, `Distribuer` (DT2-1), dans des fichiers neufs `grammar/distribuer*.go`.
- [ ] 2.1.2 États de mouvement (`movement_states*.go`) et tir continu (`tir_continu.go`) deviennent
      deux canaux ; `ScanMarcheDesTrames` les distribue ; `replay/film_scan_mouvement.go` inchangé
      dans ce qu'il publie.
- [ ] 2.1.3 « Interprété » exact (DT2-2) ; T3 de l'étape 1 resserré : chaque lecture d'état cite
      l'occurrence marquée interprétée.
- Gate : T4 (étapes `movementStates` et `continuousFire` de `replay-equiv`), killsource identique,
  T1/T3/T6 verts.

### Lot 2.2 — Canaux d'image-clé (taille M) — coordination §1.3
- [ ] 2.2.1 Armes portées (`keyframe_loadout.go`), inventaire (`inventory_decode.go`), marques de
      portage (`keyframe_carrier_mark.go`), équipes (`player_teams.go`), recensements et bandes
      (`world_object_census.go`, `slot_band_*.go`, `offline_biped_band.go`), générations vivantes
      (partie image-clé, `generations_vivantes.go`), table anticipée (`keyframe_anticipe.go`) et
      liaison (`keyframe_liaison.go`) consomment la phase `ImagesCles`.
- [ ] 2.2.2 Les fenêtres lues bit à bit À L'INTÉRIEUR des images-clés (armes : fenêtre de 32 bits,
      inventaire : emprises) deviennent des méthodes de la couche de récupération (DT2-4), marquées.
- Gate : T4 ; comptes de replis déclarés si la bande n'est plus relevée deux fois (rapport §1.6).

### Lot 2.3 — Canaux de tête de vue A (taille M) — coordination §1.3
- [ ] 2.3.1 Tirs (36), translocations (117), lunette, ramassages, apparitions (103), événements de
      véhicule : la marche lit la tête de chaque paquet UNE fois (`event_list.go`) et la donne aux
      canaux (`fire_events.go`, `transloc_events.go`, `zoom_events.go`, `biped_pickups.go`,
      `equipment_spawn_events.go` ; les événements de véhicule vivent dans `event_list.go`).
- [ ] 2.3.2 Les paquets dont la tête n'est pas lisible gardent la règle d'aujourd'hui.
- Gate : T4.

### Mesure avant 2.4 et 2.5 (DT2-5) — taille S
- [ ] M.1 Durées par étape et pic mémoire de la cuisson, trois témoins et un BTB, machine calme,
      binaires alternés ; part des balayages ancrés et des balayages d'objets du monde.
- [ ] M.2 Proposition chiffrée à l'utilisateur : mutualiser (2.4, 2.5 complets) ou marquer seulement.

### Lot 2.4 — Récupération ancrée mutualisée (taille L)
- [ ] 2.4.1 UN ancrage bipède par film (positions et les huit passes du marcheur ancré, véhicules
      compris), mémorisé ; records `PreuveRecupere`, méthode nommée, étendues de composants jusqu'au
      premier infranchissable (`delta_biped_walk.go`, `offline_biped*.go`, `ability_*.go`,
      `camo_state.go`, `grapple_state.go`, `held_weapon_changes.go`, `inventory_delta.go`,
      `equipment_changes.go`, `equipment_recovery.go`, `offline_aim_only.go`, `replay/film_scan.go`).
      Le garde-rail `delta_biped_walk_guard_test.go` (pas de dixième site d'ancrage) reste vert.
- [ ] 2.4.2 Égalité de bande (`fc.BipedSlots()` contre `bipedSlotBand` recalculée) et de découpage
      prouvée film par film AVANT la bascule.
- [ ] 2.4.3 Le harnais `FuzzFilmRecordReaders` couvre la couche de récupération (aucune panique,
      records récupérés bornés par les bits du payload), comme il couvre la marche depuis l'étape 1.
- Gate : T4 sur les données ; `coverage.fallbacks` déclaré ; banc `b.Loop` ; critère 4.

### Lot 2.5 — Récupération des objets du monde (taille L)
- [ ] 2.5.1 Créations multi-archétypes en une passe ; pistes sur l'union des bandes
      (`equipment_creation*.go`, `vehicle_creation.go`, `ground_weapon_creation.go`, `projectiles.go`,
      `equipment_placements.go`, `replay/build_ground_weapons.go`, `replay/build_vehicles.go`) ;
      calibration MPP gardée en préliminaire.
- Gate : T4 ; critère 4.

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
- [ ] 2.7.a Morts d'objet sur le marcheur unique (huit vues → trois, monde unifié) — après LU.
- [ ] 2.7.b Canaux delta lus par la marche là où elle couvre au moins autant que la recherche
      d'ancres, canal par canal, mesuré sur le corpus.
- [ ] 2.7.c killsource EN DERNIER : `runWalk`, timeline, calibration deviennent des canaux et des
      préliminaires de la même marche ; contexte partagé avec la cuisson (décision de l'utilisateur
      du 2026-10-03) ; `IDLowBits` unifié (IR-7).
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
      mesure M.1.

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
