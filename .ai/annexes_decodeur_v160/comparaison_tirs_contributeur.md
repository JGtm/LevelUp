# Les tirs : la copie du contributeur lit-elle mieux que le dépôt d'origine ?

Lecture seule, le 2026-11-23. Aucune mesure sur film. « vérifié » = ligne relue ; « déduit » = raisonné sans preuve.
Chemins : `contrib/` = `contributeur/repo/third_party/film-decoder/` ; `tête/` = dépôt d'origine à la tête de `feat/v1.6.0` ; `v1.3.6` = le point de départ de la copie.

## Conclusion

La copie du contributeur lit MOINS de tirs que la tête d'origine, et pas davantage que `v1.3.6`. Elle n'a changé ni la source ni le genre de message des tirs : seulement leur direction, leur horodatage et leur attribution (vérifié). La tête, elle, a élargi la lecture depuis `v1.3.6` : tout message de tir de la liste est lu, à tout rang (v1.5.0, schéma 126). Aucun des deux côtés n'est à 100 % de couverture ; la cause connue est surtout le rattachement du tir à une vie nommée (voir section 5).

## 1. D'où viennent les tirs, et ce que la copie a changé

Source unique (vérifié) : le record `action_weapon_fire`, type 36 de la liste d'événements d'un paquet delta, lu par `internal/grammar/fire_events.go` (`TypeTirArme` = 36, ligne 59 environ ; `canalDesTirs`). Les tirs sont publiés par `replay/shots.go` (à pied), `replay/vehicle_shots.go` (en véhicule, tirs orphelins retrouvés dans un trajet), puis `replay/fire_bursts.go` (rafales : armes à tir continu, lues dans la vue de commande par `internal/grammar/tir_continu.go`, pas dans un record 36).

Comparaison de la copie du contributeur à `v1.3.6` (octet par octet, vérifié) :

| Fichier | Verdict | Détail |
|---|---|---|
| `internal/grammar/fire_events.go` | identique à `v1.3.6` | Aucune lecture nouvelle. Le commentaire « les records 36 hors tête de liste ne sont pas lus » (ligne 49) reste vrai chez lui. |
| `internal/grammar/fire_aim_modal.go` | changé : lecture de la visée | Lit deux drapeaux avant la visée et renonce si l'un est posé (« le record ne porte pas de visée lisible »). Touche la visée, pas la présence du tir. |
| `internal/grammar/tir_continu.go` | changé : branchement | Reçoit un collecteur de boutons (gestes, postures). Les rafales lues sont les mêmes ; seul le décompte du temps de trou est refactoré (`tenirLeTrou`). |
| `replay/shots.go:158-168` | changé | `O` = décalage du tir dans sa frame, en ms ; la direction passe par `viseur.viser` (`replay/shots_visee.go`, identique à la tête). Le tir est posé de la même façon. |
| `replay/vehicle_shots.go:225` et tri | changé | Même `O` et même visée ; tri par (frame, décalage). Les tirs de véhicule sont retrouvés comme avant. |
| `replay/build_vehicles.go:123` | changé | Les visées sans position se lisent sur tout film, plus seulement sur les films à véhicules. Change la direction de certains tirs, pas leur nombre. |
| `replay/tirs_par_place.go`, `fire_bursts.go`, `places_de_controle.go` (neuf) | changé : attribution | Le tireur d'un tir ou d'une rafale est rendu par la place de contrôle lue (repli : le siège). Change le joueur, pas le nombre de tirs lus. |
| Tirs de véhicule, armes à rayon, tirs sans position | aucun changement de lecture | Mêmes sources et mêmes écartements qu'à `v1.3.6` (déduit des diffs ci-dessus : aucun fichier de lecture touché). |

Liste des changements réels de LECTURE des tirs : aucun. Les changements portent sur la visée (`fire_aim_modal.go`, `shots_visee.go`, `build_vehicles.go:123`), l'horodatage (`shots.go:162`, tri) et l'attribution (`places_de_controle.go`, `tirs_par_place.go`). Les autres écarts de ces fichiers sont des journaux traduits en anglais et des signatures refactorées.

## 2. Compteurs de couverture

Côté tête (vérifié, `tête/replay/coverage_layer.go:19-59`) : `LayerCoverage` du calque des tirs compte `available` (événements de tir lus), `attached` (rattachés à un slot), `noSlot` (aucune vie connue du joueur à l'instant), `ambiguous` (vies qui se recouvrent), `outOfWindow` (aucune position assez proche), `unpublished` (piste trop courte), `byUnit` et `unitOtherIndex`, `aim` (origine de la direction). Les tirs de véhicule ont leurs propres compteurs (`ShotsAmbiguous`, `ShotsUnplaced` = sans position, `ShotsNoRide` = hors trajet), journalisés par `vehicle_shots.go`. Les rafales ont un bilan équilibré : lues = publiées + autre bit + sans joueur + ambiguës + monture sans arme + sans piste + arme inconnue + arme non continue + vides (`tête/replay/document_fire_bursts.go:119-122`).

Côté contributeur (vérifié) : les mêmes compteurs (hérités de `v1.3.6`), plus `coverage.controlPlaces {read, bindings, players, multiPlace, overlaps, shotsByPlace, shotsBySeat, burstsByPlace, burstsBySeat}` (attribution, pas couverture) et `coverage.shots.aim` (origine de la direction). Aucun compteur d'écartement de tir de plus.

Chiffres de couverture écrits dans ses documents ou ses tests : je n'en trouve aucun du genre « N tirs sur M » pour la lecture des tirs (vérifié par recherche dans `docs/` et `third_party/README.md` : aucun taux de lecture). Il écrit des chiffres de DIRECTION, pas de lecture :
- `web/src/features/match-replay/model/shotFx.ts`, en-tête : le champ de visée du tir est présent sur 90 tirs sur 483 du film témoin (18,6 %), 16,1 % sur 23 artefacts locaux ; en relisant la visée dans les trajectoires, 483/483 (100 %) sur le témoin et 99,1 % sur son corpus ; sur 2 866 tirs portant les deux, écart médian 0,3 degré. C'est une couverture de la direction, calculée côté application.
- Sa chronique de schéma 122 : 5 690 visées sans position lues, dont 4 000 posées sur un film (cité dans `portage_complet.md` T2, vérifié par ce document).
Nos mesures M2 et M3 (`mesures/mesures.md`) : lire les visées sans position change la direction de 2,5 % des tirs des films sans véhicule ; la place lue contre le siège change le joueur de 3,9 % des tirs. Ni l'une ni l'autre ne change le nombre de tirs lus.

Côté tête, chiffres de couverture écrits (`tête/replay/document_chronicle.go:4775-4790`, vérifié) sur le corpus d'équivalence de 20 films : événements de tir 44 837 -> 59 249 sur 16 films avec la lecture à tout rang ; tirs publiés 52 348 -> 66 406 ; `coverage.shots.available` 54 006 -> 68 418. Soit 96,9 % publiés avant, 97,1 % après (déduit : publiés / `available`). Attention : `available` est lui-même le dénominateur « tirs lus », donc ce taux ne mesure pas les tirs absents du film lu.

## 3. Comment son lecteur 3D se sert des tirs

Vérifié (`web/src/features/match-replay/model/shotFx.ts:142`, `scene3d/weaponIdleEffects3d.ts:278-281`, `model/fireBursts.ts`) :
- Il lit `doc.shots` : `t` (frame), `o` (décalage en ms, pour dater le tir à la milliseconde et pour le chargeur), `slot`, `x`, `y`, `w` (arme), `v` (véhicule), `h` (cap).
- Il ne dessine pas seulement les tirs publiés : `const tirs = [...doc.shots, ...fireBurstShots(doc)]` (`shotFx.ts:142`). Les coups des rafales sont SIMULÉS à la cadence de l'arme (un coup par frame au plus par rafale), parce que le film n'écrit pas l'instant de chaque coup d'une arme à tir continu.
- La direction ne dépend pas du champ `h` du tir : il relit le cap du regard dans les trajectoires à l'instant du tir (couverture 100 % sur son témoin, 99,1 % sur son corpus d'après son en-tête).
- Les tirs servent aussi de repères au chargeur (`magazines`) et à l'animation d'arme au repos : il combine les tirs avec les lectures de munitions (`doc.ammoReads`) et les rechargements.
- Il n'exige donc pas tous les tirs : il comble les armes continues par simulation et la direction par la trajectoire. Un tir de plus ou de moins déplace un effet (éclair de bouche, chargeur), pas la structure du rejeu (déduit : je n'ai pas lancé son application).
- Les sons : son lecteur joue des sons de rechargement et d'environnement (`docs/KNOWN-ISSUES.md`, section Sounds) ; je n'ai pas trouvé de son de tir qui comble un tir manquant (déduit, non vérifié ligne à ligne).

## 4. Ce que chaque côté sait des tirs que l'autre ne sait pas

La tête a, et pas la copie (vérifié) :
- Tirs à tout rang (v1.5.0, `tête/internal/grammar/fire_events.go:132,158,161` ; entrée v126 de la chronique, `document_chronicle.go:4775`). La copie a zéro occurrence de `tirsApresLaTete` ni de lecture de `VueA.Tirs`. Gain mesuré par la tête : +32 % d'événements de tir sur 16 films du corpus (44 837 -> 59 249). Un tir de rang 1 et plus n'a pas de visée (`HasAim` faux).
- Un tir lu à tout rang améliore aussi les touches (`coverage.hits.noShooter` 800 -> 36), les liens projectile-tir (1 567 -> 2 052) et les tirs par place (1 306 -> 1 866).
- La règle du numéro de tir en retard (`shotStats[].shots`) et le tireur d'un tir par l'occupant du siège du roster à l'instant (`tête/replay/tirs_par_place.go:37-90`).

La copie a, et pas la tête (vérifié) : `shots[].o` ; la direction sur tout film (`build_vehicles.go:123`) ; la lecture de la place de contrôle pour le tireur (`places_de_controle.go`) ; le tri par (frame, décalage). Ce sont des améliorations d'exactitude et de datation, mesurées petites (M2 : 2,5 % des tirs de films sans véhicule ; M3 : environ 4 % des tirs enregistrés, 6,5 % des joueurs). La direction des tirs par la visée i21 (`shots_visee.go`) est identique des deux côtés.

## 5. Pourquoi la couverture n'est pas de 100 %, et cause connue

D'après les commentaires des deux dépôts (vérifié, sauf mention) :
1. Tirs hors tête de liste : jusqu'à `v1.3.6` et dans la copie du contributeur, seul le record de tir en tête de liste d'une trame est lu (`fire_events.go:49` : sauter les événements qui précèdent demande la grammaire de charge de chacun). La tête a levé cette limite en v1.5.0 ; son commentaire de la ligne 49 n'a pas été retiré et dit encore le contraire (incohérence à corriger, déduit du rapprochement avec les lignes 132-161).
2. Tirs rattachés à une vie non nommée : « 63 à 92 % des tirs perdus tombent dans une vie non nommée » (`tête/replay/closures.go:17`, mesure du 2026-08-08 sur sept films, avant le pont refermé). Concerne les deux côtés ; la tête a refermé le pont depuis.
3. Tirs de véhicule sans position, ambigus ou hors trajet : comptés (`ShotsUnplaced`, `ShotsAmbiguous`, `ShotsNoRide`), non publiés.
4. Tirs en rafale : les armes à tir continu n'écrivent pas chaque coup ; les trous de lecture (`holes`) ne portent aucun tir, par décision explicite (`fireBursts.ts`, décision de l'utilisateur du 2026-09-24 citée par le contributeur). Le total des coups de rafale est un contrôle, pas une lecture.
5. Index de tireur qui n'est pas la place (build ancienne, `tirs_index_fiable.go`) : seule la référence 0 pose les tirs.

Les chiffres « 96,9 % -> 97,1 % publiés » de la tête mesurent les publiés contre les lus, pas contre les tirs du film. Le taux de tirs réellement écrits dans le film mais jamais lus n'est mesuré dans aucun des deux dépôts (déduit ; à vérifier en comparant le numéro de tir de chaque tireur, qui est écrit à chaque tir, au nombre de tirs lus).
