# Portage des ajouts du contributeur au décodeur de film : inventaire par thème et plan d'items

Date de la lecture : 2026-11-23. Lecture seule : aucun code écrit, aucun test lancé, aucun film décodé.
Ce document est destiné à un dépôt public. Le contributeur se dit « le contributeur » ; aucun pseudo de joueur, aucun prénom, aucun chemin du poste.

## 0. Mode d'emploi

### Légende

- **(V)** vérifié : ligne relue dans le fichier, ou commande rejouée (comparaison d'empreintes git, `grep`, `diff`).
- **(D)** déduit : raisonné sans preuve. La phrase dit ce qui permettrait de le vérifier.
- `[C] chemin:ligne` : chez le contributeur, dépôt public `llampwall/installation-08`, révision `a66b791`, sous `third_party/film-decoder/`.
- `[T] chemin:ligne` : dépôt d'origine, branche `feat/v1.6.0`, commit `ea8156c` au moment de la lecture (la branche bouge : les lignes sont à relire à l'exécution).
- `[B] chemin` : dépôt d'origine au tag `v1.3.6`, point de départ de sa copie.
- « Tête » = `[T]`. « Pistes » = les pistes de la version v1.6.0 en cours (V0.6, V2 à V4, A à F, K, P), d'après la table des fichiers par piste du plan de la version.

### Suppositions

- Je suppose que « tous ces ajouts » couvre ce qui est du décodeur et du paquet `replay` ; les deux autres briques ne sont traitées que par la section 7.
- Je suppose que le mainteneur écrit tout lui-même, sur la branche de version, et que l'autre agent traite les gestes (thème T4) avec le même rythme.
- Je n'ai pas lu le plan des gestes de l'autre agent : il n'existait pas encore dans le dossier au moment de la lecture. Les fichiers partagés sont déduits de la lecture du code (section 2, thème T4).

### Méthode

1. Comparaison par empreinte git de chaque fichier de sa copie contre le tag `v1.3.6` : **73 fichiers ajoutés, 249 modifiés, 0 supprimé** (V ; mêmes comptes que l'inventaire déjà établi). Les 249 « modifiés » se décomposent ainsi (V) :
   - 44 fichiers de test, et 47 fichiers de production ou de référence qui portent un vrai changement (dont 9 fichiers de référence, les goldens et fichiers `testdata`) ;
   - 158 fichiers de production dont l'en-tête dit « log and error strings translated to English » : ils ne portent que la traduction, sauf une quinzaine qui portent aussi un vrai changement (ils sont cités dans les fiches des thèmes).
2. Pour chaque thème : lecture des en-têtes et des diffs contre `v1.3.6`, puis recherche du même fait à la tête.
3. Lecteurs dans son application : `grep` des champs dans `web/src` et `cmd/`, `internal/` de son dépôt.

### Ce qui change la lecture de tout le reste

- Sa chronique de document (`[C] replay/document_chronicle.go`, entrées v116 à v130) montre que ses pas de schéma sont des pas de **thème** : v116 actions du film, v117 gestes, v118 pistes au débit du film, v119 direction des tirs, v120 postures de la commande, v121 tireur de la place, v122 visées sans position, v123 manche de déplacement, v124 munitions, v125 décalage du tir, v126 hauteur des projectiles, v127 projectiles au débit du film, v128 projectiles collés, v129 détonation d'un lancer, v130 candidats de socle (V). Ses numéros ne sont pas ceux de la tête (126 actuel, 127 prévu) : la tête a son propre v116 « directions de tir » (V, `[T] replay/document_chronicle.go:4055`).
- **Une partie de son travail est déjà à la tête** : la direction des tirs par la visée i21 (`shots_visee.go`) y est identique à la sienne à deux commentaires près (V : `diff` des deux fichiers). Un reste de ses ajouts n'a pas d'équivalent.
- Son application embarque le décodeur comme bibliothèque et lui passe les entrées de l'hôte, comme LevelUp (V : `[C]` `internal/build/options.go:20-75`). Une partie de ce qu'il a ajouté est donc une **entrée que l'hôte fournit** (causes des morts, table des scores cibles) et non une lecture du film.
- Les fixtures et commentaires du contributeur citent des pseudos de joueurs réels (V : un commentaire de `[C] replay/places_de_controle.go`, de `grenades_detonation.go` et de la chronique v129 ; un test de `grenades_detonation_test.go`). **Tout est à purger au portage** (règle du dépôt public).

## 1. Vue d'ensemble

### Les 25 éléments portables et leur classe

Classes : (a) tel quel ou presque ; (b) à réécrire selon les règles ; (c) déjà couvert, rien à porter ; (d) à ne pas reprendre ; (e) indécidable sans mesure.
Les gestes (T4) ne sont pas comptés : ils sont traités ailleurs.

| N° | Élément | Thème | Classe |
|---|---|---|---|
| E1 | Pistes au débit du film (positions, visée) | T1 | (a) |
| E2 | Visées sans position fusionnées à ces pistes | T1 | (a) |
| E3 | Décalage du tir dans sa frame (`shots[].o`) | T2 | (a) |
| E4 | Tri des tirs par (frame, décalage) | T2 | (d) |
| E5 | Lecture des visées sans position sur tout film | T2 | (e) |
| E6 | Direction des tirs par la visée i21, couverture, repli nommé | T2 | (c) |
| E7 | Tireur par place de contrôle lue dans les tirs | T3 | (e), bloqué par DD-2 |
| E8 | Arme tirée en main (`drawnWeapons[]`) | T5 | (a) |
| E9 | Dégâts par entité (`damages[]`) | T6 | (b) |
| E10 | Causes des morts (`killCauses[]`) | T7 | (d) |
| E11 | Munitions portées (`ammoReads[]`) | T8 | (b) |
| E12 | Projectiles : hauteur et pleine cadence | T9 | (a) |
| E13 | Projectiles collés (`projectiles[].attach`) | T9 | (e) |
| E14 | Détonation d'un lancer sans vol (`grenades[].deto`) | T9 | (a) |
| E15 | Socles d'équipement | T10 | (e) |
| E16 | Candidats de socle (`padCandidates`) | T10 | (b) |
| E17 | Axes et échelle d'un socle (DTO du calque de carte) | T10 | (a) |
| E18 | Seconde famille du Stalker Rifle | T10 | (e) |
| E19 | Table des scores cibles par variante | T11 | (d) |
| E20 | `filmcache.ParseManifest` et `filmcache.NewSource` | T12 | (a) |
| E21 | `decfilm.LoadMapQuantCatalogBytes` | T12 | (a) |
| E22 | Codec de faits `REPLAYINPUTS30`, schémas de faits 23 à 25 | T12 | (c) pour le codec ; les sections neuves sont dans les items |
| E23 | Traduction des journaux et des erreurs (158 fichiers) | T13 | (d) |
| E24 | Noms anglais des replis dans les journaux | T13 | (d) |
| E25 | Instruments de recherche et sondes (17 tests `research`, sonde de score) | T13 | (d) |

Décompte : **(a) 9, (b) 3, (c) 2, (d) 6, (e) 5** = 25.

### Volume

- 21 items de code (19 sûrs, 2 conditionnels), 8 mesures (2 conditionnelles), 2 items du pilote. Détail en section 4.
- Lourds : T1.1, T9.2, et T3.1 si DD-2 est levée.
- Quantité de code à reprendre : environ 1 500 lignes de production et une vingtaine de tests unitaires à porter (D : somme des tailles de fichiers lues ; les en-têtes de commentaire font une bonne part de ces lignes).

## 2. Les thèmes

Pour chaque thème : (1) fichiers et publication, qui le lit chez lui ; (2) lecture ou déduction, et règles heurtées ; (3) ce qui existe à la tête et conflits avec les pistes ; (4) classe.

### T1. Pistes au débit du film (positions et visée à pleine cadence)

**(1) Fichiers et publication.**
- Ajoutés : `[C] replay/tracks_plein_debit.go` (130 lignes ; `FullRateTrack` l. 61, unité en milliseconde l. 54, `pleinDebit.ajouter` l. 92), `replay/tracks_visees_seules.go` (115 lignes ; `repartirLesViseesSeules` l. 33, `pleinDeLInstant` l. 47, `fusionnerLesVisees` l. 72). (V)
- Modifiés : `replay/tracks_publication.go` (`accDeSlot` l. 155, `publierLesVies` l. 169 extraite de `decimateTracks`, compteurs `FullRateSamples` l. 199), `replay/document_tracks.go:179` (`Track.FullRate`), `replay/build_vehicles.go:123` (voir T2). (V)
- Publie `tracks[].fullRate` : colonnes différentielles d'entiers `t, x, y, z, gaps, aimT, h, p`, au rythme du film (60 ou 30 par seconde) à côté des points de la grille ; `h` et `p` sont les quanta bruts d'i21. Il y ajoute `stickT, sx, sy` (manche : thème T4). Couverture : `coverage.tracks.fullRateSamples`, `fullRateAimSamples`, `noWorld`, `fullRateAimOnlySamples`, `aimOnlyNoLife`. (V)
- Lecteurs chez lui : `web/src/lib/replay/replayFullRate.ts` (seul lecteur du codage), `replayNormalize.ts`, `features/match-replay/scene3d/tracks3d.ts`, `fineTracks3d.ts`, `export/replayVideoEncoder.ts`. (V : `grep fullRate`). Les colonnes `stickT`, `sx`, `sy` n'ont **aucun lecteur** dans son application web (V : seul `generated.ts` les cite).
- **Indispensable à son lecteur 3D** : l'en-tête dit que la visée interpolée entre deux points distants de 100 ms est en retard dès que le joueur tourne (V, `tracks_plein_debit.go` l. 9-17).

**(2) Lecture et déduction.**
- Lecture pure : positions et visées telles que le film les écrit, chacune à son instant ; aucun rééchantillonnage ; arrondi au centième déjà en usage pour `Point`. (V)
- Aucun seuil neuf. L'unité de temps (`fullRateTimeUnitUS`, la milliseconde) est nommée et motivée dans le fichier ; la lacune reprend `lifeGapUS` (règle de `Point.G`). (V)
- Règles heurtées : **poids**. L'en-tête mesure 1,86 Mo compressé à 0,1 ms et 1,64 Mo à 1 ms, pour un document qui en fait 0,83 sans pleine cadence, sur un film (V, l. 46-52) ; sa chronique v123 mesure +16 % de plus pour le manche sur un autre film. Le document de LevelUp est servi au navigateur de la 2D : décision D1 (section 8).
- Un seul fil : les accumulateurs sont par vie et par slot, rien de partagé entre fils (D ; à prouver par la preuve « même sortie à N fils »).

**(3) Tête et conflits.**
- Aucun `FullRate` à la tête (V : `grep` sur le dépôt, hors tests). `tracks[].points` reste un point par frame (`[T] replay/document_tracks.go:159`).
- Fichiers communs avec les pistes : `replay/tracks_publication.go` (non touché depuis la v1.5.0), `replay/build_pistes.go` et `replay/build_vehicles.go` (touchés depuis la v1.5.0 par les pistes K). (V : `git diff --name-only v1.5.0 HEAD`)
- Fichier partagé avec le portage des gestes : `replay/tracks_plein_debit.go` (colonnes du manche).

**(4) Classe.** E1 (a), E2 (a), sous réserve de la décision D1 sur le poids.

### T2. Orientation et visée des tirs

**(1) Fichiers et publication.**
- `[C] replay/shots_visee.go` (116 lignes) : **identique à la tête** (V, `diff` : deux paragraphes de commentaire). Publie `shots[].h`, `shots[].p`, `coverage.shots.aim {track, sameTick, record, none}`, repli nommé `repli_visee_du_tir_lue_dans_le_record`.
- `[C] replay/shots.go:158-162` et `replay/vehicle_shots.go` : `shots[].o`, le décalage du tir dans sa frame, en millisecondes entières (reste de la division de l'horodatage par le pas de grille ; absent à 0). `[C] replay/document_tracks.go:65`. (V)
- `[C] replay/tri_total.go:88` et `replay/vehicle_shots.go` : tri des tirs par (frame, décalage) au lieu du seul ordre du film. (V)
- `[C] replay/build_vehicles.go:113-123` : les visées sans position (`VehicleScan.Aims`) se lisent sur **tout film**, pas seulement les films à véhicules ; son commentaire dit que cela change le contenu des directions de tir sans changer le format. (V)
- Lecteurs : `shots[].o` est lu par `scene3d/weaponIdleEffects3d.ts:281` et normalisé par `lib/replay/replayNormalize.ts:296` (V). Utile à la 3D pour dater un tir à la milliseconde.

**(2) Lecture et déduction.**
- `o` est un reste de division : lecture, aucun seuil. (V)
- La visée tenue est bornée par `viseeTenueMaxUS = lifeGapUS`, constante nommée déjà à la tête. (V)
- E4 : le tri par `o` change l'**ordre** de `shots[]` dans tous les films dont une frame porte au moins deux tirs, dans un calque existant. Les tirs d'une frame sortent aujourd'hui dans l'ordre du film (`[T] replay/tri_total.go:86`) ; le flux est déjà ordonné en temps, donc le tri ne change l'ordre que si une fusion (tirs de véhicule) le décale (D ; vérifiable en comparant les deux ordres sur les 20 films). Règle heurtée : rejeu identique à l'octet hors calque neuf. E4 : ne pas reprendre ; le client trie par (`t`, `o`) s'il le veut.
- E5 : la lecture sur tout film change la valeur de `shots[].h` pour les films sans véhicule (sa mesure : 5 690 visées lues dont 4 000 posées sur un film ; chronique v122, V) ; c'est un changement de contenu d'un calque existant, et touche de près la « justesse des tirs » (DD-2). Mesure M2.

**(3) Tête et conflits.**
- Déjà publié : `shots[].h/p` (`[T] replay/document_tracks.go:57-80`, `replay/shots_visee.go:33` `ShotAimCoverage`, `:82` `viser`), repli nommé (`[T] internal/facts/fallback/noms.go:196`), chronique v116. (V)
- Absent : `Shot.O`. Les visées sans position se lisent seulement sur le chemin des véhicules (`[T] replay/build_vehicles.go:115`). (V)
- Conflits : `replay/shots.go` (touché par la piste K depuis la v1.5.0), `replay/build_vehicles.go` (idem). (V)

**(4) Classe.** E3 (a), E4 (d), E5 (e), E6 (c).

### T3. Tireur d'un tir par place de contrôle

**(1) Fichiers et publication.**
- Ajouté : `[C] replay/places_de_controle.go` (247 lignes ; `lirePlacesDeControle` l. 73, `tenuesDeLaPlace` l. 152, `ControlPlaceCoverage` l. 48). Modifiés : `replay/tirs_par_place.go` (`tireursDesPlaces` l. 95), `replay/fire_bursts.go`, `replay/build_pistes.go:168,181`, `internal/facts/fallback/registre_replay_tirs.go` (repli `repli_tireur_par_le_siege_de_la_place`). (V)
- Publie : aucun champ neuf sur `shots[]` ; elle **change le joueur** auquel un tir ou une rafale est rendu, et ajoute `coverage.controlPlaces {read, bindings, players, multiPlace, overlaps, shotsByPlace, shotsBySeat, burstsByPlace, burstsBySeat}`. (V)
- Lecteur : aucun lecteur direct ; elle sert l'attribution des tirs et les postures de la commande (T4). (D)

**(2) Lecture et déduction.**
- Le fait lu est réel : l'index de tireur d'un record de tir est la place de contrôle, et le même record nomme l'unité qui tire (V, en-tête l. 5-22).
- Le reste est de la déduction : « cœur » d'une tenue (de la première à la dernière liaison), « étendue » (la présence du joueur), part « tenue par personne » d'un écart entre deux cœurs (V, l. 110-205). Aucune de ces règles n'a de seuil mesuré ni de nom de règle du dépôt.
- Sa mesure (chronique v121, V) : sur un film à arrivées en cours, le siège rendait 388 tirs à la place elle-même, la lecture 0 ; validation croisée 0 faux ; tirs rattachés 7 195 à 7 227, rafales 175 à 186.
- Règles heurtées : DD-2 (« tirs sans place » est cité comme hors périmètre) ; garde R-14 (la cible du plan : aucun compte de tirs ne bouge). Cette lecture **change des comptes** par construction.

**(3) Tête et conflits.**
- La tête a une réponse différente au même problème : `[T] replay/tirs_par_place.go:37-90` rend le tireur par l'occupant du siège du roster à l'instant, sinon la place (v126, « les tirs se lisent à tout rang », chronique `document_chronicle.go:4566`). (V)
- Conflits : `replay/build_pistes.go`, `replay/shots.go`.

**(4) Classe.** E7 (e), bloqué tant que le mainteneur ne lève pas DD-2. Mesure M3 : accord entre place lue et siège sur 20 films, y compris les films à arrivées en cours ; validation par le tueur du fil de mort.

### T4. États du bipède et gestes : **traité ailleurs**

Thème de l'autre agent (accroupi, glissade, saut, mêlée, lancer, rechargement, manche de déplacement). Je ne le refais pas. Fichiers de ma lecture qu'il partage avec les thèmes de ce document (D, déduit de la lecture du code) :

| Fichier (chez lui) | Pourquoi il est partagé |
|---|---|
| `replay/tracks_plein_debit.go` | colonnes `stickT/sx/sy` dans `FullRateTrack` (mon T1) |
| `replay/places_de_controle.go` | sert à la fois l'attribution des tirs (T3) et les postures |
| `replay/document_actions_joueur.go` | contient `DrawnWeapon`, `Damage`, `KillCause` (mes T5, T6, T7) et `PlayerAction` (les gestes) |
| `replay/build_inventaire.go` : `poserActionsDuJoueur` (l. 192) | une seule fonction pour les quatre calques |
| `types/grammar_actions.go` | `DrawnWeaponRead` (T5) et `FilmActionMessage` genres 0, 38, 39, 40 (dégâts et gestes) |
| `internal/grammar/vue_a_actions.go`, `actions_du_film.go`, `lecture/actions.go` | lecture des messages de la vue A : genre 0 dégâts (T6), 38 à 40 gestes |
| `internal/grammar/movement_states.go` | `armes` (T5) et `boutons` (gestes) ajoutés à la même marche |
| `replay/film_scan_mouvement.go`, `film_inputs.go`, `options.go`, `observe.go` | champs d'entrée `DrawnWeapons` (T5), `Actions`, `ControlInputs` (gestes) |
| `internal/grammar/tir_continu.go` | crochet `boutons` (gestes) |
| `replay/document_stances.go`, `build_postures.go` | postures de la commande |
| `replay/coverage.go`, `layers.go`, `document.go`, `filmfacts*.go` | écrivain unique : le pilote |

Constats utiles à l'autre agent, relevés en passant :
- les colonnes du manche n'ont aucun lecteur dans son application (V, ci-dessus) ;
- sa web lit `stances[]` pour `clamber`, `sprint`, `slide` (V : `scene3d/filmActions3d.ts:64`, `firstPersonEvents3d.ts:20-24`) ;
- le bit de saut diffère de celui que la tête décrit (déjà noté dans le recoupement C.4 des pistes).

### T5. Arme tirée en main (`drawnWeapons[]`)

**(1) Fichiers et publication.**
- Ajoutés : `[C] internal/grammar/jeu_darmes_desire.go` (81 lignes ; `canalDuJeuDArmes` l. 21, `recevoir` l. 67), `types/grammar_actions.go:10` (`DrawnWeaponRead`, `DrawnWeaponStats`), `replay/document_actions_joueur.go:19,104` (`DrawnWeapon`, `buildDrawnWeapons`). (V)
- Modifiés : `internal/grammar/observateur.go:272` (le crochet reçoit désormais le slot : `func(slot, JeuDArmes)`), `unit_weaponstate.go:378`, `keyframe_etat_complet_bipede.go:201`, `porte_des_essais.go:44-64`, `movement_states.go:204,235` ; côté `replay` : `film_inputs.go:108`, `options.go`, `film_scan_mouvement.go`, `build_inventaire.go:195`, `layers.go:156`, `coverage.go:202` (`PlayerActions`), `document.go:208`, sections de faits dans `filmfacts_actions.go` (V pour la liste, D pour l'emplacement exact du codage dans les faits).
- Publie `drawnWeapons[] {t, slot, d?, d2?}` : l'emplacement d'arme désiré en main principale et en seconde main, **à chaque changement**, alors que `inventory[].d` ne se lit qu'aux images-clés (toutes les ~20 s). (V)
- Lecteurs : `model/weaponInHand.ts`, `scene3d/firstPersonEvents3d.ts`, `sound/reloadSound.ts` (V : `grep drawnWeapons`). **Indispensable** à la vue subjective (arme en main).

**(2) Lecture et déduction.** Lecture pure : le composant i42 est déjà décodé, le canal ne fait que recueillir ses trois champs avec le slot (V, en-tête). Aucun seuil. Dédoublonnage par (slot, instant) : une seconde traversée du même record est comptée (`Duplicates`). Un slot non lié au bipède est écarté et compté (`SlotUnbound`). Rien ne heurte une règle.

**(3) Tête et conflits.**
- La tête a le crochet sans slot (`[T] internal/grammar/observateur.go:272`, `unit_weaponstate.go:377`). Elle publie `weaponChanges[]` : la **famille** d'arme par emplacement (`[T] replay/document_weapon_changes.go:65`) ; ce n'est pas l'emplacement tiré. Les deux faits se complètent, ils ne se recouvrent pas. (V)
- Les prérequis de la marche existent (`MarcheDistribuee.Entites`, `[T] internal/grammar/distribuer.go:114`). (V)
- Conflits : `movement_states.go` (V4), `keyframe_etat_complet_bipede.go` (V3), `film_scan_mouvement.go` (V4), `options.go` (V0.6). (V : table du plan)

**(4) Classe.** E8 (a).

### T6. Dégâts par entité (`damages[]`)

**(1) Fichiers et publication.** `[C] replay/document_actions_joueur.go:35` (`Damage {t, slot, by?, src?}`), `:137` (`buildDamages`) ; lecture dans `internal/grammar/actions_du_film.go` / `vue_a_actions.go` (genre 0 de la vue A) ; `build_inventaire.go:196`. Publie un message `damage_aftermath` par ligne, par **slot de piste** de la victime et de l'auteur. (V) Lecteurs : `model/damageArcs.ts`, `model/hud3d_logic.ts:42`, `scene3d/shieldImpacts3d.ts:162`, `shockBurst3d.ts`, `shockBurstTargets3d.ts:129`, `bodyFx3d.ts` (V : `grep damages`) ; elles s'appuient sur `d.slot`, `d.by`, `d.t`.

**(2) Lecture et déduction.** Lecture pure. Il ne publie **aucune quantité** : le champ « magnitude » n'est pas un dégât (V, chronique v116 de sa copie). Pas de seuil.

**(3) Tête et conflits.**
- La tête lit les mêmes messages (`[T] internal/grammar/degats_de_joueur.go:18`, `DegatLu`) et publie `damage[]` (`[T] replay/document_damage.go:30`) : instant, auteur et victime **en xuid**, effet (`jpt!`), arme, points de bouclier et de corps. (V)
- Différences qui comptent pour la 3D : (i) la victime et l'auteur sont des xuid, donc **absents pour un bot** (V, `document_damage.go` en-tête) alors que son lecteur indexe par slot de piste ; (ii) la tête ne publie que les messages qui retirent des points (`poserLesDegats`, `[T] replay/document_damage.go:70` : `Bouclier+Corps == 0` est écarté). Son lecteur apparie des dégâts à des tirs (80 % de concordance d'après son en-tête, V, `shieldImpacts3d.ts:14`) : ce que les messages à zéro point changent à cet appariement n'est pas connu (mesure M7).
- Collision de nom : `replay.Damage` existe déjà à la tête. (V)

**(4) Classe.** E9 (b) : ne pas porter son calque ; **compléter `damage[]`** par `slot` (victime) et `by` (auteur), en `omitempty`, après la mesure M7 (messages à zéro point, victimes sans xuid). Aucun compte existant ne bouge (garde R-14 inchangée).

### T7. Causes des morts (`killCauses[]`)

**(1) Fichiers et publication.** `[C] replay/document_actions_joueur.go:51` (`KillCause {feedMs, killer, victim?, class, category?}`), `options.go:87` (`Options.KillCauses`, **fourni par l'hôte** depuis killsource), `build_inventaire.go:198`, `layers.go:191`. Chez lui l'hôte les remplit (`[C] internal/build/options.go:195-225`). Lecteurs : `lib/replay/killsFromDocument.ts`, `model/replayScoreboardLogic.ts` ; son en-tête dit que son serveur n'a **pas** la route des éliminations et que `killCauses` y supplée (V : `killsFromDocument.ts:6-24`).

**(2) Lecture et déduction.** Aucune lecture du décodeur : une recopie de ce que killsource a déjà résolu, comme `neutralDeaths`.

**(3) Tête et conflits.** La tête a `Options.NeutralDeaths` (`[T] replay/options.go:390`) pour le même patron, et `eliminations[]` (`[T] replay/document_eliminations.go:43`) avec `headshot` et `perfect`. LevelUp a son propre modèle d'éliminations (V : `apps/web/src/features/match-replay/model/killFeedLogic.ts` existe, et l'en-tête de `killsFromDocument.ts` dit que sa copie reprend ce modèle) ; que LevelUp serve l'arme de chaque élimination par sa route est (D : l'en-tête cite le `weaponKey` de ce modèle ; à vérifier en lisant la route des éliminations de LevelUp). La catégorie `Headshot` de killsource **double** `eliminations[].headshot` : « publier chaque fait une fois ».

**(4) Classe.** E10 (d) : à ne pas reprendre. Décision D4 si le mainteneur veut un document autosuffisant : alors l'item T7.1 et la mesure M8.

### T8. Munitions portées (`ammoReads[]`)

**(1) Fichiers et publication.** `[C] replay/ammo_reads.go` (94 lignes ; `AmmoRead` l. 10, `buildAmmoReads` l. 27), `build_inventaire.go:59-61`, `document.go:134`, `coverage.go:262`, `layers.go:150`, facts : `filmfacts_encode.go:174-183` et `filmfacts_decode.go:216-228` (les munitions de chaque delta d'inventaire entrent dans les faits). Publie `ammoReads[] {t, ms, slot, w, mag?, res?, gauge?, src}`, images-clés et deltas fusionnés, une ligne par changement de valeur. (V) Lecteurs : `model/hud3d_logic.ts:78-81`, `model/weaponInHand.ts:28,81`, `reloadMagazine.ts`, `pickupAmmo.ts`, `spawnAmmo.ts`, `scene3d/firstPersonEvents3d.ts`, `weaponIdleEffects3d.ts` (V). **Indispensable** (HUD de munitions, chargeur par tir).

**(2) Lecture et déduction.** Lecture : le chargeur, la réserve et la fraction R(12) du composant de munitions sont déjà décodés. La jauge est `fraction / 4095` (V, l. 55-58), même convention que `AmmoSlot.Gauge` (`[T] replay/inventory.go:46,213`). Le dédoublonnage (valeur égale à la précédente) est une compression, pas une déduction. Règles heurtées : sa **couverture réutilise `GrenadeReadCoverage`** (V, `ammo_reads.go:86-93`) : un calque neuf a sa couverture propre.

**(3) Tête et conflits.**
- La grammaire lit déjà ces munitions : `types.InventoryDeltaAmmo` (`[T] types/grammar_bipede.go:188`) et `Options.InventoryDeltaAmmoRefused` (`[T] replay/film_inputs.go:83`).
- **Mais les faits ne les gardent pas** : l'encodage des deltas écrit seulement `Grenades`, `Sel`, `Mask` (`[T] replay/filmfacts_encode.go:162-173`). Le calque exige donc une section de faits neuve, d'où un pas de schéma des faits (écrivain unique : le pilote). (V)
- Conflits : `build_inventaire.go`, `filmfacts_encode.go`, `filmfacts_decode.go` (écrivain unique), `coverage.go`.

**(4) Classe.** E11 (b).

### T9. Projectiles et lancers

**(1) Fichiers et publication.**
- Hauteur et pleine cadence : `[C] replay/projectiles.go:51` (`Projectile.Z`), `:60` (`FullRate *FullRateFlight`), `:103` (`FullRateFlight`). Publie `projectiles[].z` (parallèle à `p`) et `projectiles[].fullRate {t,x,y,z}` (colonnes différentielles, même codage que T1). (V)
- Collés : `[C] internal/grammar/projectile_attaches.go` (127 lignes ; `ProjectileAttach` l. 38, `ScanProjectileAttaches` l. 120), `replay/projectiles_attache.go` (105 lignes ; `attachePoseRange` l. 29, `attachePose` l. 45, `poserLesAttaches` l. 78), `replay/filmfacts_attaches.go`, `movement_states.go:217`, `recuperations.go`, `film_scan_pont.go`, `options.go:79`. Publie `projectiles[].attach {fromFrame, parentSlot, node, pose}`. (V)
- Détonation d'un lancer sans vol : `[C] replay/grenades_detonation.go` (79 lignes ; `GrenadeDetonation` l. 29, `grenadeDetonationWindowUS` l. 43, `detonationOfThrow` l. 47), `internal/grammar/grenade_events.go:173,416` (`GrenadeThrow.Life/HasLife`), `replay/grenades.go:89,106`, `replay/filmfacts_lancers.go`. Publie `grenades[].deto {t,x,y,z}` pour un lancer sans `proj`. (V)
- Lecteurs : `scene3d/flights3d.ts` (hauteur, pleine cadence, `attach`, `deto`), `scene3d/stuckGrenades3d.ts`, `model/grenadeFx.ts`, `scene3d/shots3d.ts` (V). Indispensable (arcs et rebonds des grenades, grenade collée au joueur).

**(2) Lecture et déduction.**
- Z et pleine cadence : lecture pure d'échantillons que la piste brute portait déjà. Aucun seuil. (V)
- Détonation : le lien est le **handle** du projectile, une lecture. La fenêtre de recherche est de 6 s, constante nommée, motivée par « fusée la plus longue plus marge » (V). À rattacher à une durée de fusée lue si le film la donne (D : à chercher dans les lectures de durée d'effet déjà à la tête), sinon à garder nommée. Mesure citée : 3 791 des 3 793 lancers liés, 153 des 153 vols (V, en-tête).
- **Attache : la déquantification de la pose est mesurée, pas lue** (V, `projectiles_attache.go` en-tête l. 13-23) : trois R(16) lus comme des demi-plages de ±2 unités, retenues comme « la meilleure plage de 1 à 8 unités » contre l'animation reconstruite de son propre client sur 36 grenades. L'en-tête le dit lui-même : « not a constant found in the binary ». C'est une constante tirée d'un client : règle « la grammaire prime sur les heuristiques » et « aucun seuil sans raison » heurtée. La lecture, elle (parent, nœud, trois quanta bruts, lus dans i10), est solide. Mesure M4, décision D3.

**(3) Tête et conflits.**
- Absents à la tête : `Projectile.Z`, `FullRate`, `Attach`, `Grenade.Deto`, `GrenadeThrow.Life` (V : `[T] replay/projectiles.go:35-64`, `replay/grenades.go:51`). Prérequis existants : `grammar.LireHandle` (`[T] internal/grammar/handle.go:54`), motif d'en-tête NEW (`grenade_events.go:346`). (V)
- La tête lit déjà l'état parent i10 pour d'autres objets (`[T] internal/grammar/components_object_state.go:59`, `canal_des_morts.go`), pas pour les projectiles. (V)
- Conflits : `replay/grenades.go` (touché depuis la v1.5.0), `film_scan_pont.go`, `recuperations.go`, `vue_a_charges.go`, `lecture/paquet.go` (touchés), `options.go` (V0.6). (V)
- Nom : `replay.GrenadeDetonation`, `FullRateFlight`, `ProjectileAttach` : libres à la tête. (V)

**(4) Classe.** E12 (a), E14 (a), E13 (e).

### T10. Objets du monde

**(1) Fichiers et publication.**
- Socles d'équipement : `[C] replay/powerup_pads.go:75` (règle élargie), `:91` (`padItemFamily`), `:97` (`padGrenadePrefix`), `replay/catalog.go:126` (`LabelCatalog.DeployedEquipment`), `replay/build_ground_weapons.go` (transmis aux socles). Une famille d'équipement porté (Repulsor, Drop Wall, Grapple, Thruster, Translocator) donne un socle publié dans `weaponPads[].weapon`. Aucun champ neuf. (V)
- Candidats de socle : `[C] replay/ground_weapon_pads.go:207,287` (`gwPadCandidates`), `document.go:287` (`PadCandidates`), `layers.go:170`. Publie `padCandidates[]` : les grappes à une seule apparition que la règle « un socle est une récurrence » écarte. Son serveur les promeut avec la carte de la variante (`cmd/infinite-theater-3d/serve_weapons_doc.go:120`, `serve_objectives.go:204-232`) puis vide le champ. Aucun lecteur dans l'application web. (V)
- Axes et échelle d'un socle : `[C] replay/map_weapon_pads.go:85-88` (`Forward`, `Up`, `Scale` en `omitempty`), remplis par l'hôte depuis `map.mvar`. Calque de requête : ni empreinte, ni schéma. (V)
- Seconde famille du Stalker Rifle : `[C] filmshell/catalogue.go:145` (identifiant `0xadb78225`, nom `Stalker Rifle (standard)`, cadences 900/300, fusion vers `Stalker Rifle`). (V)
- Lecteurs : `scene3d/equipment3d.ts` (D : cité par `grep`, non relu).

**(2) Lecture et déduction.**
- Socles d'équipement : la règle devient « toute famille du manifeste sauf `grenade_*` et les formes déployées ». C'est une exclusion par **préfixe de nom** et un marqueur `deployed` du manifeste de l'hôte. Sa mesure : sur une carte, trois ramassages de Repulsor à moins de 0,8 m d'un point d'apparition d'équipement (V, en-tête). Une carte ne suffit pas. Mesure M5.
- Candidats : lecture et regroupement existants ; le seuil `gwPadMinHits` existe déjà. **Pas de couverture propre** (« celles des seuls socles publiés »), à ajouter. (V)
- Axes et échelle : champs de DTO, aucune lecture.
- Stalker : une entrée de table d'identité comme celle du Shock Rifle `0x1a22fee6` (`[T] filmshell/catalogue.go:139`). Elle dit venir d'une correction du contributeur. À vérifier. Mesure M6.

**(3) Tête et conflits.** Aucun de ces fichiers n'a bougé depuis la v1.3.6 à la tête, sauf deux lignes de `catalog.go` (V : `git diff v1.3.6 HEAD --stat`). `replay/build_ground_weapons.go` est touché par la piste K. Côté hôte : `config/titles/halo_infinite/mappings/replay_labels.toml` marque déjà des équipements `kind = "deployed"` ; le catalogue Go de LevelUp (`replaylabels/catalog.go`) ne remplit pas `DeployedEquipment`. (V)

**(4) Classe.** E16 (b), E17 (a), E15 (e), E18 (e).

### T11. Table des scores cibles par variante

`[C] replay/score_target_table.go` (64 lignes ; `scoreTargetByVariant` l. 9, `ScoreTarget` l. 61) : recopie de la table `[score_target]` de `regulation.toml` de LevelUp, 49 variantes. Lecteur : son hôte (`[C] internal/build/options.go:35`). (V)
- Règle heurtée : **table par variante** quand le film donne le fait. La tête lit déjà les règles de la variante dans le film (`rules`, `[T] replay/document_regles.go:20`) ; le score cible n'y est pas, mais `decfilm.ReadMatchInfo` porte le score de victoire de la variante (V, `[T] replay/document_score.go:172-181`). L'API existante accepte déjà l'entrée de l'hôte (`ScoreInput.TargetScore`, `[T] replay/score_timeline.go:41`).
- **Classe E19 (d).** Rien à porter : sa table reste dans son application, qui continue de passer `TargetScore`.

### T12. Options, entrées publiques, cache des films, codec de faits

**(1) Fichiers.**
- `[C] filmcache/source_from_file.go` (42 lignes ; `ParseManifest` l. 23, `NewSource` l. 40) et `filmcache/filmcache.go` (champ `direct`, `dirDesChunks`). Servent son outil `decode --manifest <fichier>` hors disposition du cache. (V)
- `[C] decfilm/catalog_bytes.go:13` (`LoadMapQuantCatalogBytes`) : catalogue de bornes de carte lu depuis des octets embarqués. (V)
- Champs d'entrée publics d'`Options` et de `FilmInputs` : `ProjectileAttaches`, `Actions`, `DrawnWeapons`, `DrawnWeaponStats`, `ControlInputs`, `ControlInputStats`, `KillCauses`. Ils accompagnent chacun des thèmes T5 à T9 et T4 : pas d'item à part. (V)
- Codec de faits : `[C] replay/filmfacts.go:254` `REPLAYINPUTS30`, `SchemaDesFaits = 25` (`filmfacts_fichier.go:267`). Sections neuves dans `filmfacts_actions.go`, `filmfacts_boutons.go`, `filmfacts_attaches.go`, `filmfacts_lancers.go`, `filmfacts_complement.go`. (V)
- Lecteurs : `[C] internal/build/build.go:216-221` (`ParseManifest`, `NewSource`), `internal/build/resolve.go:136-137`, `internal/build/options.go` (V).

**(2) Lecture et déduction.** Pas de déduction. Un défaut possible : `NewSource` met `tailles` à `nil` (V, l. 41-42) alors que `ParseManifest` lit le même `manifestJSON` qui porte `size_bytes` (`[T] filmcache/filmcache.go:72-78`) ; si une méthode de la source indexe `tailles[i]`, elle sort des bornes. À éprouver par un test (D : non rejoué). Un manifeste lu par chemin n'est donc pas contrôlé contre la troncature, ce que `Open` fait.

**(3) Tête et conflits.**
- `filmcache` et `decfilm/catalog*` : aucun changement à la tête depuis la v1.3.6 sur ces fichiers (V : `git diff --stat v1.3.6 HEAD -- filmcache`). Conflit : `decfilm/` touché par V0.6 (« seule écrivaine de la structure d'options ») : seulement pour d'autres fichiers du dossier.
- **Collision de nom et de contenu** : la tête a déjà `REPLAYINPUTS30` et `SchemaDesFaits = 30` (`[T] replay/filmfacts.go:255`, `filmfacts_fichier.go:263`). Sa copie emploie la même magie `REPLAYINPUTS30` avec un contenu différent et le schéma de faits 25. Un fichier de faits écrit par sa copie n'est refusé que par le numéro de schéma des faits : à surveiller dans la note de migration. (V)
- Même nom de fichier des deux côtés : `replay/options_defauts.go`, `replay/document_schema.go`, `replay/shots_visee.go`, `internal/facts/fallback/registre_replay_tirs.go` (V) : à fusionner à la main, jamais à copier.

**(4) Classe.** E20 (a), E21 (a), E22 (c).

### T13. Traductions, noms anglais, instruments

- 162 fichiers de son arbre portent le marqueur « log and error strings translated to English », dont 158 fichiers de production modifiés contre `v1.3.6` (V). Messages de journal, clés d'attribut, textes de `diag.Signaler`. Ils changent aussi les `Message` et `Attrs` de diagnostics, et trois lignes d'un fichier de référence (`golden_minibobine_familles.tsv`). Hors périmètre : « pas de traduction des fichiers existants » (plan P.2). **E23 (d).**
- `[C] internal/facts/fallback/english_names.go` (155 lignes) et son test : un nom anglais par repli pour les journaux seulement, utilisé par `compteur.go` et `fallbacks_publication.go`. Le nom publié `coverage.fallbacks[]` ne change pas. **E24 (d).**
- 17 tests sous balise `research` ajoutés (V : liste lue) et la sonde `score_keyframe_probe_test.go`. Instruments de mesure du contributeur : les verdicts utiles sont dans ses en-têtes et ses chroniques, déjà cités ici. **E25 (d)** ; les tests **unitaires** (non `research`) accompagnent chaque item.
- Autres fichiers modifiés par la seule traduction ou par des références : `revision/*`, `killicon`, `damagetag`, `internal/facts/killsource`, `internal/facts/objectives`, `internal/source`, `internal/profile` : rien à reprendre (V pour `revision/*` : zéro ligne hors traduction).

### Véhicules et sièges

Aucun ajout propre. `[C] replay/build_vehicles.go` ne fait que découper une fonction et lire les visées sans position hors véhicules (T2) ; `vehicle_shots.go` ne reçoit que `o` et le viseur partagé (V : `diff` de ces fichiers contre `v1.3.6`).

## 3. Ce qui existe déjà à la tête, fait par fait

| Fait | Sa publication | À la tête | Verdict |
|---|---|---|---|
| Direction des tirs par la visée i21 | `shots[].h/p` | identique, `shots_visee.go` | déjà là (E6) |
| Décalage du tir | `shots[].o` | absent | à porter (E3) |
| Rechargement | `actions[]` genre `reload` | `reloads[]`, `[T] replay/document_reloads.go:24` | gestes : autre agent |
| Manche de déplacement | `stickT, sx, sy` | `stances[]` `moveCommand`, `document_stances_commandes.go` | gestes : autre agent |
| Dégâts encaissés | `damages[]` par slot | `damage[]` par xuid, `document_damage.go:30` | compléter (E9) |
| Éliminations avec cause | `killCauses[]` | `eliminations[]`, `document_eliminations.go:43` (compteur) | non repris (E10) |
| Score cible | table `ScoreTarget` | `ScoreInput.TargetScore`, fourni par l'hôte | non repris (E19) |
| Tireur par place | lecture des places | siège du roster, `tirs_par_place.go:37-90` | mesure (E7) |
| Codec de faits | `REPLAYINPUTS30`, schéma 25 | `REPLAYINPUTS30`, schéma 30 | collision (E22) |

## 4. Le plan : items, ordre, volume

### Règles communes à tous les items

- **Preuve de base** : tests unitaires du paquet (ceux que le contributeur a écrits, réécrits selon les règles, plus la purge des pseudos) ; `G-clone` sur `./...` ; `G-rev` ; `G-compat` ; équivalence **attribuée** sur les 20 films (chaque écart rattaché à ce que l'item déclare) ; `G-equiv-2p` ; poids du document mesuré ; relevé à un fil dans les 10 % (P.5 du plan).
- **Même sortie à N fils** : un item qui ajoute une lecture à la marche des trames branche son canal dans le mécanisme de la tête (`[T] internal/grammar/movement_states.go:196`, `marcherPourLaCuisson`) et fournit l'empreinte du niveau N égale à celle du niveau 1.
- **Garde R-14** : tout item qui touche tirs, touches ou dégâts prouve que les empreintes des étapes de tirs, de touches, de dégâts et de statistiques de tir ne bougent pas.
- **Écrivain unique** : le schéma du document (127), le schéma des faits (31), la chronique, les goldens, les références d'équivalence, les fixtures de contrat, les révisions de couche et `coverage.go` / `layers.go` / `document.go` / `filmfacts*.go` sont posés par le pilote à l'intégration (R-15, P.7). Un item **déclare** ce qu'il change dans ces fichiers, il ne les écrit pas.
- Chaque item : fichiers de ≤ 500 lignes, fonctions de ≤ 80, ≤ 5 paramètres, commentaires en français qui disent le contrat, crédit dans `NOTICE.md` (item du pilote), aucun pseudo ni xuid réel.

### Mesures (M), à exécuter sur le corpus de 20 films, sans modifier le dépôt

| N° | Mesure | Décide | Sortie attendue |
|---|---|---|---|
| M1 | Poids compressé du document avec et sans `fullRate`, sur 5 films de tailles différentes | D1 | octets avec et sans, ratio |
| M2 | Direction des tirs des films sans véhicule : part tirée d'une visée sans position, et accord avec le cap vers l'ennemi le plus proche (la méthode du v116 de la tête, avec témoin décalé) | E5 | accord avant et après |
| M3 | Place lue contre siège du roster : nombre de tirs et de rafales rendus à un autre joueur par chacune, sur les films à arrivées en cours ; validation par le tueur des morts du fil | E7 (si DD-2 levée) | table de désaccords |
| M4 | Pose des projectiles collés : minimum et maximum des trois R(16) sur toutes les grenades collées du corpus ; saturation aux bornes 0 et 65 535 ; lecture de la largeur dans le jeu si disponible | E13 | plage observée |
| M5 | Socles d'équipement : nombre de socles publiés avec la règle élargie sur le corpus, nombre de faux socles (lieu sans point d'apparition à moins d'un mètre), sur au moins 5 cartes | E15 | faux socles par carte |
| M6 | Identifiant `0xadb78225` : nom de l'arme dans les étiquettes de LevelUp et sur les films où il apparaît dans un tir | E18 | nom confirmé ou infirmé |
| M7 | Messages de dégât à zéro point et victimes sans xuid : comptes sur le corpus (`coverage.damage` de la tête) | E9 | parts de messages perdus |
| M8 | Accord de `eliminations[].headshot` avec la catégorie `Headshot` de killsource, sur le corpus | E10 (si D4 le demande) | accord, désaccords |

M3 et M8 sont conditionnelles.

### Items de code

Taille : S = moins d'une demi-séance ; M = une séance ; L = une séance et sa relecture. « Parallèle » = peut se coder en même temps qu'un autre item de la même vague.

| Item | Thème | Fichiers de la tête à créer (C) ou modifier (M) | Ce qui change dans le document publié | Preuve | Dépend de | Taille |
|---|---|---|---|---|---|---|
| T12.1 | cache | M `filmcache/filmcache.go`, C `filmcache/source_from_file.go` (en remplissant `tailles` depuis `size_bytes`) | rien (API) | tests de `Chunk` sur une source par chemin, avec et sans taille ; `G-compat` | rien | S |
| T12.2 | catalogue | C `decfilm/catalog_bytes.go` | rien (API) | test de schéma refusé ; `G-compat` | rien | S |
| T10.3 | DTO socle | M `replay/map_weapon_pads.go:61` (`MapWeaponPadDTO` : `Forward`, `Up`, `Scale` en `omitempty`) | forme de la couche de requête ; ni empreinte ni schéma | `G-compat`, test de forme | rien | S |
| T10.4 | Stalker | M `filmshell/catalogue.go:139` (entrée, cadences, fusion) | libellés d'arme du fil de mort | test de la fusion ; `G-equiv` sur un film à Stalker | M6 | S |
| T2.1 | décalage du tir | M `replay/shots.go:170`, `replay/vehicle_shots.go`, `replay/document_tracks.go:57` (`Shot.O`) ; **sans** toucher `tri_total.go` | `shots[].o` | `shots_visee_test` + test du reste de division ; R-14 | rien | S |
| T6.1 | dégâts par piste | M `replay/document_damage.go:30,70` (`Damage.Slot`, `By` en `omitempty`), `replay/document_hits.go` si besoin | `damage[].slot`, `damage[].by` | test de projection ; R-14 : mêmes comptes | M7 | S |
| T10.2 | candidats | M `replay/ground_weapon_pads.go:194`, `replay/build_ground_weapons.go:177` (appel), `replay/document.go` (`PadCandidates`) ; couverture neuve `coverage.groundWeapons.candidates` | `padCandidates[]` | tests des grappes à une apparition ; `buildWeaponPads` inchangé (octet identique) | rien | S |
| T9.1 | hauteur et pleine cadence des projectiles | M `replay/projectiles.go:35` (`Z`, `FullRate`, `FullRateFlight`, `pleinDebitDeVol`) | `projectiles[].z`, `projectiles[].fullRate` ; couverture du nombre d'échantillons | `projectiles_plein_debit_test` réécrit ; octet identique de `p` | T1.1 (réutilise `fullRateTimeUnitUS` et `centiemes`) | M |
| T5.1 | arme tirée, lecture | C `internal/grammar/jeu_darmes_desire.go`, M `observateur.go:272`, `unit_weaponstate.go:377`, `keyframe_etat_complet_bipede.go:201`, `porte_des_essais.go`, `movement_states.go:196`, C types `DrawnWeaponRead`, `DrawnWeaponStats` | rien encore | test du canal ; équivalence : la lecture seule ne change aucun octet | rien | M |
| T5.2 | arme tirée, publication | C `replay/document_arme_tiree.go` (la partie `DrawnWeapon` détachée de `document_actions_joueur.go`), M `build_inventaire.go`, `film_inputs.go`, `options.go`, `film_scan_mouvement.go` ; déclarer coverage, layer, faits | `drawnWeapons[]`, `coverage.drawnWeapons` | `document_actions_joueur_test` (partie arme) | T5.1 | M |
| T8.1 | faits : munitions | M `replay/filmfacts_encode.go:162`, `filmfacts_decode.go` (`InventoryDelta.Ammo`) ; déclarer le pas de faits | rien (faits) | aller-retour de faits ; `G-rev` | rien | S |
| T8.2 | munitions publiées | C `replay/ammo_reads.go` (avec une couverture propre `AmmoReadCoverage`), M `build_inventaire.go:55` | `ammoReads[]`, `coverage.ammoReads` | `ammo_reads_test` ; vérifier la convention de `gauge` contre `AmmoSlot.Gauge` | T8.1 | M |
| T1.1 | pleine cadence | C `replay/tracks_plein_debit.go` **sans** les colonnes du manche, M `replay/tracks_publication.go:96` (`accDeSlot`, `publierLesVies`), `replay/document_tracks.go:116` (`Track.FullRate`), compteurs de `TrackCoverage:313` | `tracks[].fullRate`, `coverage.tracks.fullRateSamples/fullRateAimSamples/noWorld` | `tracks_plein_debit_test` réécrit ; `points` identique à l'octet ; poids (M1) ; `G-equiv-2p` | D1, M1 | L |
| T2.2 | visées sans position sur tout film | M `replay/build_vehicles.go:115` (lire `Aims` hors chemin véhicule) ; couverture | valeurs de `shots[].h` sur les films sans véhicule ; `coverage.vehicles.aimReads` | M2 ; R-14 : aucun compte de tir ne bouge | M2, DD-2 | M |
| T1.2 | visées seules au débit du film | C `replay/tracks_visees_seules.go`, M `tracks_publication.go` | `fullRate.aimT/h/p` plus complets ; `coverage.tracks.fullRateAimOnlySamples/aimOnlyNoLife` | `tracks_visees_seules_test` | T1.1, T2.2 | M |
| T9.2 | collés, lecture | C `internal/grammar/projectile_attaches.go`, M `recuperations.go`, `movement_states.go`, `film_scan_pont.go`, `options.go`, `film_inputs.go`, `types` ; faits : section des attaches (`filmfacts_attaches.go`) | rien encore | `projectile_attaches_test` (deux enregistrements capturés, sans pseudo) ; marche : même sortie à N fils | rien | L |
| T9.3 | collés, publication | C `replay/projectiles_attache.go` ; la pose publiée en **quanta bruts** ou avec une plage tirée de M4 | `projectiles[].attach` | `projectiles_attache_test` | T9.2, T9.1, M4, D3 | M |
| T9.4 | détonation d'un lancer | M `internal/grammar/grenade_events.go:346` (`Life`, `HasLife`), C `replay/grenades_detonation.go` (fenêtre nommée), M `replay/grenades.go:92` ; faits : `filmfacts_lancers.go` | `grenades[].deto` | `grenades_detonation_test`, `grenade_life_test` | rien (mais conflit sur `grenades.go`) | M |
| T10.1 | socles d'équipement | M `replay/powerup_pads.go:64`, `replay/catalog.go:42`, `replay/build_ground_weapons.go:177` ; hôte : `DeployedEquipment` | `weaponPads[]` plus complet pour les équipements | `powerup_pads_test` ; M5 | M5 | M |
| T3.1 | places lues | C `replay/places_de_controle.go` réécrit (règles nommées), M `replay/tirs_par_place.go`, `fire_bursts.go`, `build_pistes.go`, `internal/facts/fallback/registre_replay_tirs.go` | joueur d'un tir ou d'une rafale ; `coverage.controlPlaces` | M3 ; **R-14 ne passe pas** : exception du user | DD-2, M3 | L |
| T7.1 | causes des morts | C `replay/document_causes.go`, M `options.go` (`Options.KillCauses`) | `killCauses[]` | M8 ; fournisseur côté hôte | D4, M8 | S |
| PL.1 | pilote : écrivains uniques | schéma du document 127, schéma des faits 31, chronique, goldens, références, révisions de couche (rang neuf), fixtures, registre des changements de sortie | une entrée de chronique 127 | gates finals | tous | — |
| PL.2 | pilote : crédit et notes | `NOTICE.md` (crédit), fragments de `FORMAT.md` et `GLOSSARY.md`, note de migration (section 6), notes de version | — | relecture | tous | — |

### Ordre proposé

1. **Vague 0 (parallèle, sans conflit)** : M1, M4, M5, M6, M7 ; T12.1, T12.2, T10.3, T10.4 (après M6).
2. **Vague 1 (parallèle)** : T2.1, T6.1 (après M7), T10.2, T8.1, T5.1.
3. **Vague 2** : T1.1 (après D1 et M1), T8.2 (après T8.1), T5.2 (après T5.1).
4. **Vague 3** : T9.1 (après T1.1) ; T1.2 et T2.2 (après T1.1, M2 et la décision D2) ; T9.2 puis T9.3 (après M4 et D3) ; T9.4 ; T10.1 (après M5).
5. **Après le gel de la branche** : tout item qui écrit `options.go`, `film_scan_*.go`, `movement_states.go`, `keyframe_etat_complet_bipede.go`, `distribuer_tetes.go`, `vue_a_lecture.go` passe après les pistes V0.6, V2, V3, V4 et D, comme le dit le plan de la version (« les pistes d'abord, gel, puis les ports »).
6. **Conditionnels** : T3.1 (si DD-2 est levée), T7.1 (si D4 le demande).
7. **Pilote** : PL.1 et PL.2 à chaque fin de vague, une fois le schéma 127 et les faits 31 figés.

Items dont les fichiers de production ne sont touchés par **aucune** piste en cours (peuvent partir avant le gel, au rythme du plan) : T12.1, T12.2, T10.3, T10.4, T8.1, T6.1 (`document_damage.go`, non touché depuis la v1.5.0). Les autres passent par au moins un fichier touché par les pistes : T2.1 (`shots.go`), T10.2 (`build_ground_weapons.go`), T9.4 (`grenades.go`), T2.2 (`build_vehicles.go`), T5.x et T9.2 (marche, options). (V : `git diff --name-only v1.5.0 HEAD` rapproché de la liste des fichiers touchés.) Les fichiers à écrivain unique (faits, schéma, goldens) restent au pilote pour tous.

### Volume honnête

- 21 items de code : 9 de taille S (T12.1, T12.2, T10.3, T10.4, T2.1, T6.1, T10.2, T8.1, et T7.1 conditionnel) ; 9 M (T9.1, T5.1, T5.2, T8.2, T1.2, T2.2, T9.3, T9.4, T10.1) ; 3 L (T1.1, T9.2, et T3.1 conditionnel).
- Estimation (D) : environ 17 séances pour les 19 items sûrs, plus 2 à 3 si T3.1 et T7.1 sont retenus. Les mesures (8) tiennent en 4 à 5 séances, car elles se regroupent sur un seul harnais.
- Le point dur est l'écrivain unique (faits, goldens, références) : le pilote les assemble en une passe par vague, pas en une par item.

## 5. Côté LevelUp (hors plan du décodeur)

Ce qui accompagne le décodeur chez l'hôte, déduit de la lecture des chemins (D ; vérifiable en relisant ces fichiers) :
- `apps/go-api/internal/domain/replaydoc/document.go` et `service/replayview/convert_document.go` : le document est recopié dans un type de LevelUp ; chaque champ publié (`fullRate`, `drawnWeapons`, `ammoReads`, `shots[].o`, `projectiles[].z/fullRate/attach`, `grenades[].deto`, `padCandidates`, `damage[].slot`) y a une ligne.
- `apps/go-api/internal/replaybuild/options.go` : fournir `DeployedEquipment` (depuis `replay_labels.toml`, entrées `kind = "deployed"`) et, si D4, `KillCauses`.
- `replaylabels/catalog.go` : remplir `LabelCatalog.DeployedEquipment`.
- Reprise de `promotePadCandidates` (promotion des candidats par la carte de la variante) : côté serveur, dépendante du lecteur de fichiers (`mapvar`).
- `make generate-types` pour le web.

## 6. Canevas de la note de migration pour le contributeur

Principe : son module a le même chemin (`github.com/JGtm/halo-infinite-film-decoder`). Il remplace `third_party/film-decoder` par un `require` sur la version v1.6.0 et retire la ligne `replace` du décodeur. Les deux autres `replace` restent tant que les briques ne sont pas publiées.

| Thème | Tu lisais chez toi | Dans le dépôt d'origine | Tu retires de ta copie |
|---|---|---|---|
| T1 | `tracks[].fullRate {t,x,y,z,gaps,aimT,h,p}` | même champ, même codage (D1 peut en faire une option : à confirmer) | `replay/tracks_plein_debit.go`, `tracks_visees_seules.go`, champs de `coverage.tracks` |
| T1 (manche) | `fullRate.stickT/sx/sy` | selon l'autre note (gestes) | idem |
| T2 | `shots[].h`, `shots[].p`, `coverage.shots.aim` | **déjà là**, mêmes noms, schéma 116 de la tête | ta version de `shots_visee.go` |
| T2 | `shots[].o` | `shots[].o` | ton tri par (`t`, `o`) dans `tri_total.go` : le dépôt d'origine garde l'ordre du film ; trie chez toi si tu en as besoin |
| T2 | visées sans position sur tout film | selon M2 | — |
| T3 | `coverage.controlPlaces`, tireur par place lue | **non repris dans la v1.6.0** (hors périmètre « précision ») ; le dépôt d'origine rend le tireur par le siège du roster | `places_de_controle.go` reste chez toi, hors de la copie du décodeur, ou en patch |
| T5 | `drawnWeapons[]` | `drawnWeapons[]` | `jeu_darmes_desire.go`, crochet à deux paramètres |
| T6 | `damages[] {t,slot,by,src}` | `damage[]` : mêmes lignes, auteur et victime en xuid **et** `slot`/`by` ; `src` devient `effect` (hexadécimal sur huit chiffres) ; le type s'appelle `replay.Damage` du dépôt d'origine, pas le tien | ton type `Damage` et `buildDamages` |
| T7 | `killCauses[]`, `Options.KillCauses` | **non repris** : tu continues à nourrir ta propre liste, ou tu lis `eliminations[]` pour `headshot` | `KillCause`, `keepKillCausesOfPublishedTracks` |
| T8 | `ammoReads[]` | `ammoReads[]` (couverture `coverage.ammoReads`, type propre) | `ammo_reads.go` |
| T9 | `projectiles[].z`, `.fullRate`, `.attach`, `grenades[].deto` | mêmes champs ; `attach.pose` selon D3 (quanta bruts ou plage) | `projectiles_attache.go`, `grenades_detonation.go` |
| T10 | `weaponPads[]` pour les équipements ; `padCandidates` ; `Forward/Up/Scale` | mêmes noms ; `DeployedEquipment` rempli par ton hôte comme avant | tes patches de `powerup_pads.go`, `ground_weapon_pads.go`, `map_weapon_pads.go`, `catalog.go` |
| T10 | Stalker `0xadb78225` | selon M6 | ton entrée de `filmshell/catalogue.go` |
| T11 | `replay.ScoreTarget(variante)` | **n'existe pas** : garde la table chez toi et passe `ScoreInput.TargetScore` | `score_target_table.go` |
| T12 | `filmcache.ParseManifest`, `filmcache.NewSource`, `decfilm.LoadMapQuantCatalogBytes` | mêmes noms, mêmes signatures | `source_from_file.go`, `catalog_bytes.go` |
| T13 | journaux traduits, `english_names.go` | **non repris** : les messages et clés du dépôt d'origine restent en français | tes 158 fichiers traduits ; tes outils qui lisent les clés anglaises du journal sont à reprendre |

Avertissements à écrire dans la note :
- **Numéros de schéma** : tes documents portent `schemaVersion` 126 à 130 et les faits 23 à 25. La v1.6.0 sera au document 127 et aux faits 31 : un de tes documents au numéro 127 serait pris pour un document courant alors qu'il n'a pas le même contenu. Régénère tous tes documents et tous tes faits, ne réutilise rien.
- **Magie de faits** : ton codec `REPLAYINPUTS30` et celui du dépôt d'origine ont le même nom et des contenus différents ; seul le numéro de schéma des faits les distingue.
- **Noms de fichiers en double** : si tu gardes une copie locale, ne mélange pas tes `options_defauts.go`, `document_schema.go`, `shots_visee.go`, `registre_replay_tirs.go` avec ceux du dépôt d'origine.
- **Pseudos** : tes commentaires et tests citent des pseudos de joueurs réels ; le dépôt d'origine est public, il n'en reprend aucun. Fais de même.

## 7. Les deux autres briques (images de carte, lecteur de fichiers du jeu)

Hors plan. L'inventaire déjà établi dit :
- cartes : départ `v1.0.1`, **195 fichiers ajoutés et 27 modifiés** contre `v1.0.1` ; le dépôt d'origine est à `v1.4.0` ;
- lecteur de fichiers : départ `v1.0.1`, **6 ajoutés** (surtout des tests), **20 modifiés** ; le dépôt d'origine est à `v1.2.0`.

Ce que le décodeur en **dépend** :
- Les nouveaux fichiers de production du décodeur n'importent aucun symbole neuf des deux briques : le code de production de sa copie n'importe que `mapvar` du lecteur, à travers six fichiers de `replay/` (`map_objectives.go`, `map_weapon_pads.go`, `map_weapon_pads_catalog.go`, `objectives_catalog.go`, `socles_de_drapeau.go`, `zone_attribution.go`), et ces fichiers ne portent que la traduction, hors l'ajout de champs à `map_weapon_pads.go` (V : `grep` des imports et `diff`).
- Les champs `Forward`, `Up`, `Scale` du DTO d'un socle sont remplis **par l'hôte** avec ce que le lecteur de fichiers lit dans `map.mvar` (axes des socles). Le décodeur ne les lit pas : l'item T10.3 n'attend donc aucun tag. Le **remplissage** côté LevelUp attend les axes dans `mapvar`, donc un tag du lecteur de fichiers (D : le contributeur les a ajoutés dans ses six fichiers `mapvar`).
- Aucune dépendance du décodeur vers la brique des cartes pour ces ajouts (D : ses tests importent `himap`, pas ses ajouts).
- Dépendance neuve de sa brique des cartes : `github.com/woozymasta/bcn` v0.7.0 (V : `[C] go.mod:15`), licence à relire par le pilote (plan P.6).

## 8. Ce que je n'ai pas pu établir, et décisions du mainteneur

### Non établi

1. **Le contenu du plan de l'autre agent** : fichiers partagés déduits (T4).
2. **Les mesures du contributeur** (nombre de tirs, accords, poids) sont lues dans ses en-têtes et ses chroniques, **non rejouées** : aucun film n'a été décodé. Les 8 mesures ci-dessus servent à les refaire.
3. **Pas de compilation** : je n'ai pas vérifié que ses fichiers s'insèrent dans la tête sans changement d'API. J'ai vérifié par lecture que les interfaces du distributeur de canaux (`Canal`, `CanalDesTrames`, `[T] internal/grammar/distribuer.go:63-87`) sont identiques aux siennes, et que `LireHandle`, `accumSlot` et `Entites` existent.
4. **Lecture ligne à ligne** : faite pour les fichiers de production des thèmes T1, T2, T5, T8, T9, T10, T11, T12 (en-têtes et corps lus) ; pour T3, seul le haut du fichier et la fin sont lus en entier ; pour les 158 fichiers de traduction, j'ai filtré les lignes qui ne sont pas des messages et lu les résidus des 11 plus gros (V) ; les 44 tests modifiés et les goldens ne sont pas lus (sauf l'effet de `trierTirs` et de `minifilm.tsv`, V).
5. **Les lecteurs** dans son application sont établis par `grep` (V pour la présence), pas par relecture complète du rendu : « indispensable » est (D) pour `equipment3d.ts`.
6. **Défaut possible de `NewSource`** (taille nulle) : (D), à éprouver par un test.
7. **Détails de l'ordre des tirs** : l'affirmation que le tri par `o` ne change l'ordre qu'après une fusion de véhicule est (D) ; à vérifier en comparant les deux ordres sur les 20 films.
8. **Plage de la pose collée** : la valeur ±2 est une mesure sur son client ; je n'ai pas pu la confirmer contre le code du jeu (le serveur de décompilation de la session ne répondait pas).
9. **Volume en lignes** : somme approximative des tailles de fichiers lues.
10. **Date de la tête** : les lignes `[T]` sont celles de `feat/v1.6.0` au commit `ea8156c` ; les pistes en cours déplacent les lignes, et peuvent aussi livrer un des faits ci-dessus avant le portage (en particulier `shots.go`, `grenades.go`, `build_vehicles.go`).

### Décisions qui reviennent au mainteneur

- **D1. Pleine cadence dans le document principal ou non.** Elle double environ le poids compressé des pistes sur le film mesuré (0,83 à 1,64 Mo). Options : toujours publiée ; option d'`Options` choisie par l'hôte ; document à part pour la 3D. Recommandation (D) : option d'`Options`, puisque la tête a déjà un porteur unique d'options (R-16).
- **D2. Tireur par place (E7) et visées sans position sur tout film (E5).** Les deux touchent la justesse des tirs (DD-2, garde R-14). Garder hors périmètre, ou lever DD-2 pour ces deux seuls canaux après M2 et M3.
- **D3. Pose des projectiles collés (E13).** Publier les trois quanta **bruts** (un fait de lecture) et laisser la plage au client ; ou garder une plage mesurée sur l'animation de son client (heuristique). Recommandation (D) : quanta bruts, plage restituée par M4 ou par le code du jeu.
- **D4. Causes des morts (E10).** Ne pas reprendre (LevelUp a déjà ses éliminations et leur arme, et `eliminations[].headshot` existe) ; ou le reprendre pour qu'un document de rejeu se suffise à lui-même.
- **D5. Dégâts (E9).** Étendre `damage[]` par `slot` et `by` plutôt que reprendre `damages[]` (recommandation) ; ou publier les deux.
- **D6. Socles d'équipement (E15).** Accepter la règle « toute famille du manifeste sauf `grenade_*` et les formes déployées » si M5 montre peu de faux socles ; sinon une liste d'équipements nommée par le manifeste de l'hôte.
- **D7. Ordre des tirs (E4).** Garder l'ordre du film (recommandation) ou prendre son tri (changement de l'ordre d'un calque existant).
- Hésitations porter / ne pas reprendre, thème par thème : E5, E7, E10 (porter ou non), E13, E15, E18.

### Ce qui est sûr

- Neuf éléments sont à reprendre presque tels quels (E1, E2, E3, E8, E12, E14, E17, E20, E21).
- Trois sont à réécrire (E9, E11, E16).
- Deux sont déjà couverts (E6 : la direction des tirs ; E22 : le codec).
- Six ne sont pas à reprendre (E4, E10, E19, E23, E24, E25).
