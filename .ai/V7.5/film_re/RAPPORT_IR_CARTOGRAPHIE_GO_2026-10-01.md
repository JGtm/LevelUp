# Rapport — Représentation intermédiaire du film : inventaire, germes, conception Go, migration

> Analyse en LECTURE SEULE, 2026-10-01. Dépôt `C:\Users\Guillaume\Downloads\Scripts\LevelUp`,
> branche `feat/v75`, tête `8b894a677`. Aucune compilation, aucun test, aucune base, aucune cuisson.
> Les chemins sont relatifs à `apps/go-api/internal/games/halo_infinite/film/` sauf mention ; les
> numéros de ligne sont ceux de la tête (re-mesurés), les symboles font foi.
>
> Conventions : **[É]** établi sur pièce (fichier:ligne ou sortie de commande citée) ;
> **[S]** supposé / estimation (raisonnement explicite, à mesurer).
>
> La branche de J12 n'est pas visible (`git branch -a` : seule `remotes/origin/feat/suite-audit-decodeur`
> existe ; `git worktree list` : aucun worktree J12). Le §5 raisonne donc sur la description de J12
> dans le plan (`.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md:1132-1162` et journal `:1658-1687`)
> croisée avec l'arbre de la tête.

---

## 0. Synthèse

1. **~40 traversées : confirmé à la tête [É]**. Une cuisson « depuis le film » fait **37 parcours
   des paquets delta** (22 balayages bit à bit, 3 marches grammaticales dont 1 de killsource, 6
   lectures de tête de vue A, 6 parcours de `replaybuild`/killsource sur les paquets à événements
   ou les trames du statborg), **+3** sous garde de mode, **~16 parcours d'images-clés** (la marche
   d'ancres est mémorisée par contexte, mais **deux contextes** la recalculent : killsource et
   replay), **3 lectures du chunk de temps forts**, **2 recherches de motif xuid**. Le marcheur
   ancré bipède tourne **10 fois** par cuisson (positions, 8 passes de canal dont `i48` deux fois,
   véhicules).
2. **J4 S1 n'a pas réduit les traversées [É]** : il a déplacé cinq lecteurs de `replay` vers
   `grammar` et créé `grammar.ScanPontDIdentite` (`internal/grammar/pont_identite.go:82`), qui
   ENCHAÎNE six lectures, chacune sa propre traversée, appelées par la cuisson ET le collecteur. C'est
   une unification d'IMPLÉMENTATION (fin de la seconde séquence divergente), pas de passe. La
   faiblesse 6 était hors périmètre du plan (§1.4, `:114`).
3. **Le germe central existe** : `ScanMarcheDesTrames` (`internal/grammar/movement_states.go:138`)
   est déjà une marche grammaticale complète (bit de configuration, vue A, vue B, vue C, verdict de
   fermeture) ; `FrameClosure` (`frame_closure.go:165`) en recopie le pilotage. **Il y a trois
   marcheurs grammaticaux de trames delta incompatibles** (3 vues par classes ; 8 vues de
   `object_deaths_march.go` ; 8 vues de `killsource/walk.go`, jumeau déclaré), trois modèles de
   monde et deux sources d'`IDLowBits` (13 en dur contre 10..15 calibré).
4. **Trois corrections au modèle de la spec, sur pièces** : (a) la NOTE 5.16 rend caduc le modèle
   « trois tables d'entités par vue » (`frame_infer.go:104-117`) : c'est UNE table de datums indexée
   par l'eid, avec la vue en attribut (le commentaire de `world.go:32` est resté à l'ancien modèle) ;
   (b) un parcours « en flux, une fois » est incompatible avec trois lectures en avant déjà en
   production (table anticipée, bande de slots « chunk d'après », générations vivantes datées par des
   créations postérieures) : la cible réaliste est **2 phases** (images-clés puis trames) plus des
   préliminaires bornés ; (c) « terminateur hors cadre » n'est pas une queue opaque localisable :
   c'est une fermeture refusée dont la position fautive est inconnue.
5. **Vue C [É]** : la cause n° 1 ne désigne pas une lacune de la vue C (ses lacunes ont leurs propres
   causes : kind 1/2, bloc 0xbc, débordement). C'est le verdict résiduel « tout a été lu jusqu'aux
   terminateurs, mais le paquet ne ferme pas » (`frame_closure_classement.go:94-108`), que le code
   attribue « le plus souvent [à] une fin de vue B fausse » (`frame_vue_controle.go:290-293`).
   Elle pèse **91,9 %** des records utiles non fermés du corpus de référence, **92,5 %** sur
   HI_1_13_0 : atteindre 95 % y exige d'en résorber **≥ 81 %** ; fermer TOUTES les autres causes ne
   porterait le build qu'à 81,3 %. Le sujet touche le « résidu film dense », clos par décision
   utilisateur du 23/09 (mémoire), donc sa reprise est une décision utilisateur.
6. **La mesure du déclencheur sous-compte l'utile [É]** : la colonne `product_use` de
   `ecs_table.tsv` marque `aucun` des composants que le produit lit (états de mouvement i29/i54/i57/i62,
   vitesse i1 du saut, équipe ti=9 i0, véhicules ti=40, zones ti=13) ; 36 lignes à usage produit à la
   tête contre 44 annoncées par la spec.
7. **Ordre de migration corrigé** : créations et positions (premiers chez la spec) ne peuvent PAS
   migrer vers la marche à différence nulle — le chercheur d'ancres trouve 162 444 records `ti=35`
   sur `bfecd02b` contre 97 447 pour la marche (`movement_states.go:16-26`). Ils passent d'abord
   dans la couche de récupération, telle quelle. Ordre sûr : mouvement + tir continu, canaux
   d'image-clé, canaux de tête de vue A, récupération ancrée mutualisée, puis seulement les
   changements de comportement (morts d'objet, canaux delta, killsource en dernier).
8. **J12 impose l'ordre** : J12.1/J12.3/J12.4/J12.5 touchent les mêmes fichiers (`film_scan*.go`,
   `world.go`, `film_context.go`, `killsource/decode.go`, goldens d'empreinte, références
   d'équivalence qui bougent sous J12). La représentation intermédiaire démarre APRÈS la fusion de
   J12 et le re-figeage des références.
9. **Le « 7,9 Go » de la spec est mal attribué [É]** : c'est `51101d1d` (CTF Fortress, pas un BTB),
   causé par le déroulage `NamedEventsFrom/incrementTimes` du statborg, corrigé le 2026-09-03
   (4,9 s, 0,08 Gio). Pics mesurés depuis : 0,08 à 0,68 Gio (bombes), 0,11 à 1,19 Gio (G-equiv J2,
   BTB compris).

---

## 1. Inventaire des traversées en production

### 1.1 Les primitives de parcours

| Primitive | Fichier:ligne | Nature | Ce qu'elle parcourt |
|---|---|---|---|
| `source.Load` → `appendPackets` | `internal/source/film.go:103`, `:261` | découpe | Décompresse chaque chunk UNE fois, découpe tous les paquets (`Payload` = sous-tranche, `types/source.go:23-32`). Le film entier reste résident. |
| `FilmContext.ChunkAt` / `FilmChunkAt` | `internal/grammar/film_chunks.go:140`, `:156` | accès | Rend octets + paquets ; `filmPacketsOf` (`:211`) **ré-alloue** un `[]FilmPacket` à CHAQUE appel. |
| `DecodeFrameViewsCurseur` → `decodeFrameParRangs` | `frame_harvest.go:162`, `:315` | **M** marche grammaticale | Bit de config, vue A (`consumeVueA`, `frame_vue_messages.go:75`), vue B (`decodeInferLoop`, `frame_infer.go:253`), vue C (`consumeVueC`, `frame_vue_controle.go:122`), oracle `vueCFermee` (`:294`). |
| `DecodeFrameRecords` | `frame_records.go:250` | **M** | Boucle de records d'UNE vue ; appelée en boucle de 8 vues par `marchRecordsOf` (`object_deaths_march.go:181`) et `killsource.walkFrom` (`facts/killsource/walk.go:62`). |
| `walkDeltaBipedPayload` / `walkDeltaBipedRecords` | `delta_biped_walk.go:81`, `:106` | **H** balayage bit à bit | Ancre les en-têtes bipèdes (`matchBipedHeader`, `offline_biped.go:273`) à chaque bit ; échec = `p++` (`delta_biped_walk.go:90`). |
| `runCreationWalk` / `matchWorldObjectNewHeader(In)` | `vehicle_creation.go:78` ; boucles `equipment_creation.go:272`, `biped_creation.go:200` | **H** | En-têtes de record NEW d'un archétype, à chaque bit. |
| `ScanWorldObjectsForBand` | `projectiles.go:161` (boucle `:354`) | **H** | Pistes d'objets du monde d'une bande de slots. |
| `MarcheDImageCle.Marcher` | `keyframe_world_marche.go:84` (mémo `:55`) | **K** ancres + élection (repli) | Chaque payload d'image-clé, une fois **par contexte de film** ; preuve par fermeture d'état complet (`keyframe_world_preuve.go:78-107`). |
| `WalkKeyframeFullState` | `keyframe_fullstate_loop.go:73` | **M** | Corps d'état complet d'un record d'image-clé. |
| `WalkKeyframeRecords` | `keyframe_record_walk.go:183` | **M** (chaîne de l'écrivain) | **Aucun appelant de production** (grep : seuls les tests). |
| `TableDeDatums` / `candidatsDeDatum` | `keyframe_datums.go:83`, `:120` | **H** sur image-clé | En-têtes candidats à CHAQUE bit + plus longue suite croissante. |
| `readPacketHead` / `PacketHeadEventType` | `event_list.go:86`, `:100` | **M-tête** | Préambule de 9 bits + type du PREMIER message de la vue A. |
| `marchLocateStrict/Fallback`, `locateStrict/Fallback` | `object_deaths_march.go:123`, `:139` ; `killsource/walk.go:87-125` | **H** | Début de la vue B d'un paquet à événements (signature slot 123, 35 bits). |
| `debutDeLaListe` / `candidatsDeTete` | `debut_de_liste.go:41`, `:51` | **H** + preuve | Records NEW de tête de liste (chaîne ou fermeture). |
| `weaponv3.ResolveXuidToPI` ; `chercherMotifs` | `player_index.go:83` ; `sync/killcollector/shots.go:128-138` | **O** motif d'octets | Motif du xuid dans chaque chunk de réplication. |
| `ParseHighlightEvents` | `highlight_events.go` | **T** | Chunk de type 3 (temps forts). |
| `scanFrameAvecReplis` | `facts/objectives/statborg.go:231` (boucle `:235`) | **H** dans `facts` | Records du statborg dans toutes les trames delta. |

### 1.2 Graphe d'appel d'une cuisson (branche « décoder »)

```
replaybuild.(*Builder).BuildBytes                     replaybuild/replaybuild.go:237 (ctx = context.Background(), :243)
├─ entreesDeLaCuisson                                  replaybuild/filmfacts_cuisson.go:97
│  ├─ chargerLeFilmDeLaCuisson → chargerFilm → decfilm.LoadDir     filmload.go:70-71        [source.Load]
│  ├─ lireMorts → decfilm.ScanDeaths                   filmload.go:107-108                    [T]
│  ├─ statborgDuFilm → StatRecordsAvecReplis + CaptureBurstTimes   matchfacts.go:102-108      [H trames delta, octets]
│  └─ decodeKillSource → decfilm.Decode                kills.go:34-48 (context.Background)
│     └─ killsource.Decode → decodeCtx.prepare         facts/killsource/decode.go:92, :190-265
│        loadFilm · loadKillFeed (tous les chunks) · readFilmTable · lireIndexParMotif ·
│        loadBotMeta · scanKillEvents (pickGate15 ×2 + 1) · newTimeline (NOUVEAU FilmContext) ·
│        calibrate (échantillon) · runWalk (8 vues) · scanFilm (dead-state bit à bit)
└─ documentDeLaCuisson → replay.BuildFromFilmAvecFaits filmfacts_cuisson.go:199 ; replay/build_from_film.go:61
   ├─ grammar.NewFilmContextForMap (SECOND contexte, mémo neuf)     build_from_film.go:95
   └─ scanFilmInputs                                  build_from_film.go:213-268
      balayerPositions (film_scan.go:28) → lirePontDIdentite → grammar.ScanPontDIdentite
                                       → ScanFireEvents → ScanKeyframeLoadoutsMarche → balayerNaissances
      balayerPortage (:129) → ScanHeldWeaponChanges, ScanBipedPickups, ScanKeyframeInventory, ScanInventoryDeltas
      balayerCapacites (:220) → ScanAbilityRanks, ScanEquipmentChanges, ScanCamoStates, ScanGrappleReads,
                                ScanAbilityImpulses, ScanAbilityCharges
      balayerMonde (:332) → ScanZoomEvents, ScanEquipmentPlacements, ScanEquipmentSpawnEvents,
                            pads ×2, véhicules, balayerCalquesGardes (marques, ti=13, bombe)
      balayerEtatsDeMouvement (film_scan_mouvement.go:30) → ScanMarcheDesTrames
      balayerPont (:393) → ScanGrenadeThrows, ScanProjectiles, ScanFilmPlayerTable, ScanPlayerTeams
```

La liste fermée des étapes observées est `replay/observe.go:45` (`BuildFromFilmSteps`) : **46
étapes**, dont 34 hors `.stats` (comptage `awk` de la liste) ; `replaybuild` y ajoute ses étapes
(`score`, `objectives`, `vip`, `skull`, `bomb`, artefact) — d'où les « 61 étapes » du journal du plan.

### 1.3 Table des traversées de la cuisson

Légende des natures : **H** balayage bit à bit · **M** marche grammaticale · **M-tête** tête de
vue A seule (coût O(paquets), pas O(bits)) · **K** parcours d'images-clés par la marche mémorisée ·
**T** temps forts · **O** motif d'octets · **R** chunk_00. « Δ » = paquets delta (type 0).

| # | Étape | Fonction (fichier:ligne) | Parcourt | Garde (sortie) | Options / filtres | Nature |
|---|---|---|---|---|---|---|
| 0 | chargement | `source.Load` (`internal/source/film.go:103`) | tous les chunks | film décompressé + paquets | — | découpe |
| 1 | (replaybuild) | `ScanDeaths` (`grammar/deaths_source.go:115`, `scanDeaths :123`) | chunk de temps forts (numéro du manifeste) | morts | — | T |
| 2 | (replaybuild) | `objectives.StatRecordsAvecReplis` (`facts/objectives/statborg.go:186`) | toutes les trames Δ des chunks datables | `StatRecord` | plafond `statMaxRecordsPerFilm` | **H** (`:235`) |
| 3 | (replaybuild) | `objectives.CaptureBurstTimes` (`facts/objectives/extract.go:324` → `film.go:346`) | trames Δ des chunks de jeu | instants de rafales | `countDistinctTiers` | O |
| 4 | killsource | `loadKillFeed` (`facts/killsource/feed.go:96`) | **tous** les chunks, `ParseHighlightEvents` sur chacun | kill-feed (argmax de kills) | repli `repli_chunk_du_pied_par_argmax` | T ×n |
| 5 | killsource | `readFilmTable` (`film_table.go:113`) | chunk_00 | table des 32 sièges | — | R |
| 6 | killsource | `lireIndexParMotif` (`index_motif.go:76`) | chunks 1..n-1 | index ↔ nom | `weaponv3.ResolveXuidToPI` | O |
| 7 | killsource | `loadBotMeta` (`botmeta.go:113`, `paquetsBotMeta :176`) | paquets de métadonnées de bot | bots | — | partiel |
| 8 | killsource | `scanKillEvents` (`assist.go:158`) + `pickGate15` (`:187`, 2 passes) | paquets Δ à événements | kill-events 85 | ancre bit à bit `killEventsAvecArrets` (`eventchain.go:366-389`) | **H ×3** |
| 9 | killsource | `newTimeline` (`facts/killsource/world.go:55`) | images-clés : `grammar.NewFilmContext(f.src).MarcheDImageCle()` (`:66`, **contexte neuf**) + `sweepKeyframe` (`:254`) | monde par images-clés | `ParseRegistryChunk` refait (`:59`) | **K** + **H** |
| 10 | killsource | `calibrate` (`calibrate.go:172`) | échantillon (`calibSample :357`) × hypothèses | profil calibré (mot de poignée…) | — | M partiel |
| 11 | killsource | `runWalk` (`walk.go:141`) | tous les paquets type 0 | dead-states | 8 vues, `locateRecordsAvecVerdict` (bit à bit), snapshot/restore du monde | **M** + H |
| 12 | killsource | `scanFilm` (`scan.go:114`) | paquets Δ à événements | candidats dead-state | gabarit 58 bits à chaque bit (`scan.go:61-64`) | **H** |
| 13 | translocations | `ScanTranslocatorTeleports` (`transloc_events.go:119`) | Δ | événements 117 | `readPacketHead` (`:159`) | M-tête |
| 14 | bipedCreations | `fc.CreationsDeBipede` → `ScanBipedCreationsForBand` (`biped_creation.go:145`) | Δ | créations ti=35 | bande `fc.BipedSlots()` (K mémo), signature 43 bits | **H** (`:200`) |
| 15 | (génér. vivantes) | `fc.GenerationsVivantes` → `viesConnuesDuFilm` (`generations_vivantes.go:287`) | images-clés | vies (slot, gen) | — | K |
| 16 | positions | `ScanBipedPositions` (`offline_biped.go:207`) → `scanBipedChunks` (`offline_biped_band.go:147`) | Δ | positions | `bipedSlotBand` **recalculée** (`offline_biped.go:213`, `offline_biped_band.go:179`), filtre de génération daté par paquet (`offline_biped_band.go:164`), exemptions, vitesse | **H** |
| 17 | deaths | `scanDeaths` dans le pont (`pont_identite.go:111`) | temps forts (**2e lecture**) | morts | — | T |
| 18 | playerIndices | `scanPlayerIndices` (`pont_identite.go:116`, `player_index.go:56`) | chunks de réplication | table d'index | `ResolveXuidToPI` (**2e recherche**) | O |
| 19 | clockOrigin | `ScanClockOrigin` (`origin.go:30`) | 2 en-têtes du chunk 1 | origine | — | trivial |
| 20 | fire | `ScanFireEvents` (`fire_events.go:131`) | Δ | tirs 36 | tête de liste | M-tête |
| 21 | loadouts | `ScanKeyframeLoadoutsMarche` (`keyframe_loadout.go:89`) | images-clés (K mémo) + fenêtre de 32 bits à chaque bit (`:201`) | armes portées | catalogue `known` | K + **H** |
| 22 | birthLoadouts | `ScanBirthLoadouts` (`birth_loadouts.go:64`) | paquets Δ portant une création + images-clés + datums | dotations de naissance | `lierLeChunk` (`:158-170`, `TableDeDatums`) | partiel M + H |
| 23 | heldWeaponChanges | `ScanHeldWeaponChanges` (`held_weapon_changes.go:86`, marcheur `:104`) | Δ | prises/lâchers | marcheur ancré | **H** |
| 24 | pickups | `ScanBipedPickups` (`biped_pickups.go:121`) | Δ | ramassages 8/9 | tête de liste (`:173`) | M-tête |
| 25 | inventory | `ScanKeyframeInventory` (`inventory_decode.go:140`) | images-clés (K) + bits dans les emprises (`:281`, `:307`) | inventaires | — | K + H |
| 26 | inventoryDeltas | `ScanInventoryDeltas` (`inventory_delta.go:88`, `:95`) | Δ | i22/i47 | marcheur ancré | **H** |
| 27 | abilityRanks | `ScanAbilityRanks` → `walkAbilityEmissionsWith` (`ability_rank.go:167`, `:182`) | Δ | i48 | marcheur ancré | **H** |
| 28 | equipmentChanges | `ScanEquipmentChanges` (`equipment_changes.go:89`) → **même** `walkAbilityEmissionsWith` (`:97`) + `scanEquipmentRecovery` (`equipment_recovery.go:164`, `:237`) | Δ (+ fenêtres) | changements d'équipement | i48 **relu une 2e fois** | **H** + H partiel |
| 29 | camoStates | `ScanCamoStates` (`camo_state.go:82`, `:123`) | Δ | i28 | marcheur ancré | **H** |
| 30 | grappleReads | `ScanGrappleReads` (`grapple_state.go:73`, `:110`) | Δ | i59 tag 3 | marcheur ancré | **H** |
| 31 | abilityImpulses | `ScanAbilityImpulses` (`ability_impulses.go:78`, `:106`) | Δ | i57/i59 tag 1 | marcheur ancré | **H** |
| 32 | abilityCharges | `ScanAbilityCharges` (`ability_charges.go:80`, `:104`) | Δ | i56 | marcheur ancré | **H** |
| 33 | zoomEvents | `ScanZoomEvents` (`zoom_events.go:110`) | Δ | lunette | tête (`:147`) | M-tête |
| 34 | placements | `ScanEquipmentPlacements` (`equipment_placements.go:132`) : `worldObjectSlotBand` (K), `ScanWorldObjectsForBand` (H), `CalibrateMPPWidthsOf` (`equipment_creation_width.go:238`, 63 découpages candidats, arrêt anticipé), `ScanEquipmentCreationsForBand` (H) | Δ + images-clés | poses ti=37 | calibration toujours jouée (avant le `switch`) | K + **H ×3** |
| 35 | spawnEvents | `ScanEquipmentSpawnEvents` (`equipment_spawn_events.go:70`) | Δ | événements 103 | tête (`:121`) | M-tête |
| 36 | pads | `decodeFilmPadScans` (`replay/build_ground_weapons.go:67`) ×2 archétypes : `ScanWorldObjectKeyframes` (K), créations (H), pistes (H) | Δ + images-clés | socles ti=42 et ti=37 | largeurs MPP de 34 | 2K + **H ×4** |
| 37 | vehicles | `decodeFilmVehicleScan` (`replay/build_vehicles.go:104`) : K (`:108`), créations (`:114`, H), `ScanBipedPositionsForBand` bande ti=40 (`:120`, H), `ScanVehicleEvents` (`:224`, M-tête), `ScanBipedAimOnly` (`:210`, H, `offline_aim_only.go:133-143`), `ScanMarchFacts` (`:155`) | Δ + images-clés | véhicules, morts écrites, occupation | si la bande ti=40 est non vide | K + **H ×3** + M-tête + **M** |
| 37a | (dans 37) | `ScanMarchFacts` (`object_deaths.go:127`) : `marchPacketsOf` (K), `calibrateFrameConfig` (`object_deaths_calibrate.go:77`, IDLowBits 10..15 sur ≤ 3 000 paquets), puis tous les Δ en 8 vues | Δ + images-clés | morts d'objet, occupation | localisateur bit à bit | **M** + M partiel ×6 |
| 38 | carrierMarks | `ScanCarrierMarks` (`keyframe_carrier_mark.go:80`) | images-clés | marques de portage | **garde Drapeau** | K |
| 39 | zoneReads/flagGauge | `ScanManagedProperties` (`zone_state_scan.go:159`, boucle `:251`) | Δ | propriétés ti=13 | **gardes Zones/Drapeau** (une lecture partagée) | **H** |
| 40 | bombReads | `ScanNavpointRadial` (`navpoint_radial_scan.go:116`, boucle `:263`, état complet `:354`) | Δ + images-clés (état complet) | anneau ti=12 | **garde Bombe** | **H** + K |
| 41 | movementStates / continuousFire | `ScanMarcheDesTrames` (`movement_states.go:138`) : `ConstruireTableAnticipee` (K, `keyframe_anticipe.go:116`), `lierLeChunkAuMonde` par chunk (K + `TableDeDatums` H, `keyframe_liaison.go:87-122`), tous les Δ par `DecodeFrameViews` 3 vues (`:300`), `debutDeLaListe` (`:287`) | Δ + images-clés | états de mouvement, tir continu (vue C) | `ClassesDeVue`, `TablesParVue` | **M** + K + H |
| 42 | grenades | `ScanGrenadeThrows` (`grenade_events.go:208`, boucle `:305`) | Δ | lancers | motif d'amorce par build | **H** |
| 43 | projectiles | `ScanProjectiles` → `ScanWorldObjects(ti=41)` (`projectiles.go:130`) | Δ + images-clés | trajectoires | bande ti=41 | K + **H** |
| 44 | filmTable | `ScanFilmPlayerTable` (`film_player_table.go:134`) | chunk_00 | sièges | — | R |
| 45 | playerTeams | `ScanPlayerTeams` (`player_teams.go:142`) | images-clés + état complet (`:255`) | équipes, entités joueur | — | K + M |

### 1.4 Le collecteur killsource (par match)

```
KillSourceCollector.collect                           sync/killcollector/collector_run.go:213
├─ decodeFilmForMatch → bridge.go:131 decfilm.Load (chunks en mémoire)
├─ decoderSousLaCarte → decfilm.Decode                 collector_run.go:187      [n° 4 à 12 ci-dessus]
├─ collectShots → BuildWeaponShotsBatch                shots.go:78
│     resolvePlayerIndices → chercherMotifs (lecteur PROPRE, hors décodeur, exception datée 8.17)   [O, 3e recherche]
│     decfilm.ScanFireEventsB5 par chunk brut (weaponscan)                                          [H]
├─ collectPositions → buildPositionRows → lireLePontDuCollecteur
│     decfilm.NewFilmContextForMap (contexte neuf)    positions.go:317
│     decfilm.ScanPontDIdentite                       positions.go:318          [n° 13 à 19]
│     ContextesDesMorts / PlacementDesVies (sans relecture, isolation_facts.go:337, placement_des_vies.go:100)
│     PortagesAuSync (replay/porteurs_au_sync.go:103), selon le mode :
│        drapeau : StatRecordsCtx + CaptureBurstTimes + ScanPlayerTeams + decodeFilmPlacements + pads ×2
│        bombe   : ScanKeyframeLoadoutsMarche + ScanBirthLoadouts + ScanHeldWeaponChanges
└─ collectHits                                         hits.go:89
      INACTIF pour Halo Infinite : `match.weapon.accuracy = "not_exposed"`
      (config/titles/halo_infinite/mappings/capabilities.toml:80) ; s'il tournait, il RELIRAIT le film
      depuis le disque trois fois (ScanFilmWeaponShots/Damages, BuildBipedTracks → ScanFilmBipedPositions(dir)).
```

### 1.5 Recompte à la tête

| Catégorie | Cuisson | Collecteur (base / mode drapeau ou bombe) |
|---|---|---|
| Δ, balayage bit à bit complet (H) | **22** dans `BuildFromFilm` (créations, positions, 8 passes du marcheur ancré, poses ×3 dont la calibration MPP à arrêt anticipé, socles ×4, véhicules ×3, grenades, projectiles) + statborg (1) | créations, positions, motif propre + `ScanFireEventsB5` / + statborg, poses ×3, socles ×4, ou arme tenue |
| Δ à événements seulement (H) | killsource : kill-events ×3, dead-state ×1 | idem (killsource) |
| Δ, marche grammaticale (M) | **3** : `ScanMarcheDesTrames`, `ScanMarchFacts`, `runWalk` (+ calibrations sur échantillons) | 1 (`runWalk`) |
| Δ, tête de vue A (M-tête) | **6** (translocations, tirs, ramassages, lunette, apparitions 103, événements de véhicule) | 1 (translocations) |
| Δ, octets (statborg rafales) | 1 | 0 / 1 |
| **Total Δ** | **≈ 37**, **+3** sous garde (ti=13, ti=12, et la garde Drapeau ajoute des images-clés) | **≈ 9** / jusqu'à ≈ 20 |
| Images-clés (K) | ≈ 14 parcours de la mémo (`BuildFromFilm`) + 1 marche complète dans le contexte de killsource + `sweepKeyframe` + `TableDeDatums` ×2 (mouvement, naissances) | killsource + pont (2 contextes) |
| Temps forts (T) | **3** (`lireMorts`, pont, `loadKillFeed` sur tous les chunks) | 2 |
| Motif xuid (O) | **2** | **3** (dont un lecteur propre à `sync`) |

Conclusion [É] : l'ordre de grandeur « ~40 traversées » de l'audit (faiblesse 6,
`.ai/AUDIT_DECODEUR_FILM_2026-09-24.md:310-311`) tient à la tête (37 parcours Δ + gardes), et le
« marcheur rappelé par 9 sites » vaut aujourd'hui **7 sites de production exécutés 8 fois** (i48 lu
deux fois : `ability_rank.go:182` est appelé par `ScanAbilityRanks` ET `ScanEquipmentChanges:97`) ;
`unit_equipment_scan.go:85` est un 8e site **sans appelant de production** (grep `ScanUnitEquipment`).
Avec les positions (`offline_biped.go:241`) et les véhicules, le moteur d'ancrage bipède tourne **10 fois**.

### 1.6 Lectures jumelles et doublons constatés

| Doublon | Pièces | Effet |
|---|---|---|
| Trois marcheurs grammaticaux de trames | `movement_states.go:184-217` (3 vues par classes, datums, anticipation) ; `object_deaths_march.go:44` + `:181` (8 vues, timeline chronologique) ; `killsource/walk.go:141` (8 vues, timeline + fenêtres de trou). Le jumelage 2/3 est DÉCLARÉ (`object_deaths_march.go:27-30`). | Trois mondes, trois localisateurs, deux nombres de vues. |
| `IDLowBits` | Défaut 13 (`frame_records.go:143-146`) dans la marche du mouvement et dans `runWalk` (`walk.go:143-144`) ; calibré 10..15 par `ScanMarchFacts` (`object_deaths_calibrate.go:53-54`, `:77-106`). | Paramètre hors flux (P4) à deux provenances. |
| Marche d'ancres d'image-clé | Mémo par contexte (`keyframe_world_marche.go:39-59`) ; killsource ouvre son propre contexte (`facts/killsource/world.go:66`), la cuisson le sien (`build_from_film.go:95`), le collecteur le sien (`positions.go:317`). | La primitive la plus coûteuse (découverte 8.25 : 77 % du temps de test de `grammar`) recalculée deux fois par cuisson. |
| Temps forts | `scanDeaths` par numéro de manifeste (`deaths_source.go:123-150`) ; `loadKillFeed` par argmax sur tous les chunks (`feed.go:96-120`). | Deux localisations du chunk de type 3 ; la seconde est un repli compté 19/19 (`MESURES_CLOTURE_J11:115`). |
| Index par motif xuid | `index_motif.go:76`, `player_index.go:83`, `killcollector/shots.go:128` | Trois lectures, dont un lecteur de bits hors décodeur. |
| Bande bipède | `fc.BipedSlots()` (`film_context.go:370`) puis `bipedSlotBand` recalculée par `ScanBipedPositions` (`offline_biped.go:213`) | Le repli `repli_bande_bipede_comblee` est **sommé par relevé** (`offline_biped_band.go:202`) : son compte publié dépend du nombre de traversées. |

### 1.7 Ce qu'a réellement fait J4 S1

[É] D'après les commits et le code :
- **J4.2** (`dd0d4dfa3`) : déplacement pur de `deaths_source.go`, `player_index.go`, `origin.go`,
  `film_player_table.go`, `inventory_decode.go` de `replay` vers `grammar` ; quatre types vers
  `film/types` ; prédicat de finalisation dans la feuille `film/finalise` ; G-equiv 20/20.
- **J4.3** (`ba7ad4322`) : `grammar.ScanPontDIdentite` (`pont_identite.go:82-129`), « l'étage unique des
  lectures du pont d'identité » — translocations, créations, positions avec exemptions, morts, index,
  origine — appelé par la cuisson (`replay/film_scan_pont.go:36`) ET le collecteur
  (`killcollector/positions.go:318`) ; les politiques (roster, injectivité, fatalité) restent chez
  chaque appelant ; ratchet `archlint/film_pont_identite_test.go`.
- **J4.4** (`bc982cdf8`) : `IsolationDecoderRev` monté (le collecteur gagne les exemptions).
- **J4.5** (`a15bfc126`) : façade 173 → 165.
- **J4.6** (`c69d0c097`) : les sept lecteurs artisanaux passent par `source` ; ratchet structurel
  d'extraction de bits, allowlist de 9 exceptions datées (découverte 8.17).

**Ce que S1 n'a pas fait** : `lire` (`pont_identite.go:95-123`) appelle six fonctions qui parcourent
chacune le film ; la cuisson et le collecteur font chacun ces six traversées. « Étage unique » =
**une seule séquence d'appels, une seule implémentation**, pas une seule passe. C'est conforme au plan
(DU-3 = S, §1.4 « Faiblesse 6 … hors périmètre »).

---

## 2. Germes existants de la représentation intermédiaire

| Germe | Partie du modèle §4 produite | Comment | Ce qui manque |
|---|---|---|---|
| `internal/source/film.go` (`Load`, `Packets`, `AllPackets`, `Meta`) | **Chunks[]**, **Paquets[]** (chunk, index, type, horodatage, payload sans copie — P9) | Une décompression, une découpe, sous-tranches (`:261-293`) | Étendue en octets du chunk et taille annoncée hors `Meta` ; `FilmPacket` reconverti à chaque `ChunkAt` (`film_chunks.go:211`). |
| `frame_records.go` | **Record** : genre par code préfixe, eid (bas + génération), NEW / DEL / DELTA, `FrameRecord{Type, ID, Slot, TypeIndex, Trace, DesyncAt, HeaderBit}` (`:149-161`) | Port de `FUN_1406cd128` | `HeaderBit` **non posé** par la boucle de production (`frame_infer.go:290`), donc pas d'étendue de record ; pas de vue ; `IDLowBits` non calibré (P4). |
| `traverse.go` (`EntityTrace`, `CompResult`, `traverseComponentLoopFrom`) | **Composant** {index, nom, `StartBit`, porté ou non} et `DesyncAt` = premier infranchissable ; masque, état par défaut, `EndBit` (`:19-41`, `:245-291`) | Chaque composant présent est ajouté avec son bit de début ; un non porté arrête (`:283-285`) | Fin de composant implicite ; provenance de largeur non typée (`largeurCalibree`/`largeurBouchon`, `:260-282`, vides en production — `profil_balayage.go:300-304`) ; `Payload any` alloue ; ~56 o par composant. |
| `frame_vue_messages.go` | **Vue A** {vide, genres, portée} (`:47-57`) | Lit le premier genre `R(7)` puis s'arrête (`:75-89`) | Les 123 grammaires de message ne sont pas portées : une vue A non vide est **infranchissable** ; la vue B est alors trouvée par recherche (`marchLocateStrict`, `debutDeLaListe`). Les décodeurs de tête (`event_list.go`, tirs, 117, 103, ramassages, lunette, véhicules) et la grammaire partielle d'événement de killsource (`eventchain.go:1-40`) vivent à côté. |
| `frame_vue_controle.go` | **Vue C** {kinds, entrées `kind 0`, arrêt typé `ArretVueC`, fermée} (`:38-58`, `:93-105`, `:309-319`) ; oracle de fermeture du paquet (`vueCFermee :294`) | `consumeVueC` + `lireEntreeDeControle` ; verdict publié au crochet `VueControleHook` | Étendue par entrée ; kinds 1/2 et bloc 0xbc non portés. |
| `frame_harvest.go` (`DecodeFrameViewsCurseur`, `decodeFrameParRangs`) | **Trame delta** dans l'ordre : config, A, B, C ; curseur final ; rangs lus (`:162-190`, `:315-355`) | La marche de production par classes de vue | Étendues de vue non rendues (seulement le curseur final) ; `ScanFrameTargets`/`DecodeFrameResync` (récupération) cohabitent dans le même fichier sans appelant de cuisson. |
| `frame_infer.go` (`decodeInferLoop`, `rejetDeVue`, `corpsDeRecordNeuf`) | Boucle de la vue B avec la garde de la table de datums, liaison par anticipation (repli compté), refus des NEW contradictoires (`:152-170`, `:178-206`, `:253-345`) | Sortie `hitEnd=true` sur terminateur OU sur rejet (`:281-293`) | La sortie par rejet n'est pas distinguée de la sortie par terminateur dans ce que la marche rend (seulement comptée dans l'observateur, `observateur.go:291-303`). |
| `observateur.go` (`Observation`, ~33 crochets) | Mécanisme **interprété** : le désérialiseur publie la valeur pendant la traversée (`:38-275`) | Crochets posés par balayage | Le crochet ne reçoit ni l'étendue du composant ni l'identité complète du record (seul le slot de capture). |
| `lecteur.go`, `lecteur_position.go`, `lecteur_position_exceptions.go` | Lecteur unique (= `source.Bits` + profil + capture + observateur) ; portage unique de `FUN_14076e524` ; **13 sites en exception datée** | `Lecteur` décore `source.Bits` (`lecteur.go:20-40`) | Les exceptions datées sont des « hypothèses » au sens de la spec §4, non exposées par composant. |
| `delta_biped_walk.go` | Records bipèdes **récupérés** par ancrage (`deltaBipedRecord`, `:44-61`) avec `LifeKey` (`:132`) | Marcheur ancré, avance `i0 + largeur` | Pas de mémo (10 exécutions), pas de marque « récupéré », relecture par canal (`walkRecordTo`). |
| `frame_closure.go`, `frame_closure_classement.go` (J4.0) | **Fermeture** par paquet, par vue, par archétype, records utiles, causes d'arrêt (`FrameClosureReport :133-151`) | Rejoue la marche de production en recopiant son pilotage (`:159-200`) | Pas d'étendue par record ; « hors cadre » non ventilé par sortie de vue B ; dénominateur « entrées de contrôle utiles » absent (`MESURES_CLOTURE_J11:94-96`). |
| `keyframe_closure.go` | Fermeture des records d'image-clé par archétype (`:127-166`) | Bornes = ancres suivantes de la marche | Fermeture par record non rendue. |
| `keyframe_record_walk.go` | **Image-clé** déterministe : en-tête de 108 bits, `n1`/`n2`, état complet, arrêts typés (`KeyframeWalkStop :115`) | Chaîne de l'écrivain `FUN_142e2bfd0` (`:8-35`) | **Pas utilisée en production** : la production marche par ancres et élection (`keyframe_world*.go`). |
| `keyframe_world_marche.go`, `keyframe_world_preuve.go` | Phase « images-clés » mémorisée : une marche par payload et par contexte, avec rejet d'un élu contredit par un record prouvé | `memoireDesMarches` (`:39-59`), `PreuveDImageCle` | La mémo n'est pas partagée entre contextes ; l'élection est un repli (46 394 déclenchements sur 19 témoins, `MESURES_CLOTURE_J11:112`). |
| `keyframe_record_spans.go` | Étendue d'un record d'image-clé (par l'ancre suivante) | `KeyframeRecordSpan` (`:31-53`) | Sans appelant de production ; étendue « délimitée par heuristique », pas par traversée. |
| `world.go`, `keyframe_datums.go`, `keyframe_liaison.go`, `keyframe_anticipe.go` | **État de marche** : table slot → (archétype, eid complet, liaison souple/dure, génération inconnue, vue) (`world.go:23-80`) ; table de datums ; table anticipée | `BindFull/BindImageCle/BindDatum/BindSoft/BindWildcard` | Pas d'exposition en lecture seule ; la provenance de la liaison n'est qu'en drapeaux (`Soft`, `GenAny`) ; la table de datums est lue à **position libre** (`candidatsDeDatum`, bit à bit). |
| `generations_vivantes.go` + `types.LifeKey` (J5) | Identité (slot, génération) (P6) ; générations vivantes datées | Créations + images-clés (`:291-312`) | Repose sur la création **heuristique** (n° 14). |
| `revision/` (J3) | Révision par fermeture des imports, empreinte insensible aux commentaires | `perimetre.go`, `couches.go:20-28` | Rien : la RI y entre d'office si elle vit dans l'arbre de `grammar` (§3.7). |
| `facts/fallback` + `grammar/replis_du_film.go` | Registre nommé des replis (119 entrées) ; comptes de `grammar` en données | `Repli{Nom, Condition, Sites, CompteurBranche…}` (`repli.go:206-242`) ; `ComptesDesReplis` versé par `replay` | Infrastructure prête pour la couche P8. |
| `replay/observe.go` + `cmd/replay-equiv` | Preuve T4 par étape (46 étapes de `BuildFromFilm`) | Digest par étape | — |
| `pont_identite.go` (J4.3) | Orchestration partagée cuisson / collecteur | Séquence unique de 6 lectures | Six traversées. |

**Corrections du modèle de la spec issues des germes**

1. **P5 « trois tables d'entités, une par vue » est caduc [É]** : `frame_infer.go:104-117` (lot 5.16) —
   « LE MODELE N ETAIT PAS « LES TROIS TABLES D ENTITES PAR VUE » … une seule table, indexée par l'eid
   ENTIER », la table de datums du décodeur partagé, dont l'image-clé est le dump. `world.go:32` dit
   encore l'inverse (commentaire périmé, à signaler à J12.5). La structure doit exposer UNE table
   (slot → eid complet, archétype, vue, provenance de la liaison).
2. **Pas de marche en un seul passage avant [É]** : la table anticipée exige les images-clés
   ULTÉRIEURES (`movement_states.go:194-199`, `keyframe_anticipe.go:182`) ; la bande bipède lit
   l'image-clé du chunk d'après (`offline_biped_band.go:182`) ; le filtre de génération daté refuse
   un record AVANT une création postérieure (`generations_vivantes.go:42-58`). D'où au minimum : phase
   K (toutes les images-clés), créations, puis phase Δ.
3. **« Terminateur hors cadre » n'est pas une queue opaque** : la position fautive est inconnue
   (`frame_closure_classement.go:14-15`). La structure doit porter une **fermeture refusée** par paquet
   (records « non prouvés ») distincte d'une **queue opaque** à position connue.
4. **De la récupération vit DANS la marche [É]** : localisation de la vue B par signature
   (`marchLocateStrict`), records NEW de tête (`debutDeLaListe`), table de datums à position libre,
   liaison par anticipation, élection d'ancre d'image-clé. P8 exige de les marquer, sans quoi T1 mesure
   un mélange.
5. **L'interprétation est parfois à état** : la position delta d'i0 s'accumule sur la précédente
   (`world.go:66-75`, `SetPos`). « Décoder à la demande » doit donc se faire en repli ordonné par canal,
   pas en accès aléatoire.

---

## 3. Conception Go

### 3.1 Emplacement des types — recommandation : `film/internal/grammar/lecture`

[É] `film/types` est haché **par ses octets** dans les périmètres de `source`, `grammar`, `killsource`
et `objectives` (goldens `internal/*/testdata/*_perimetre.golden`). Une couche se reconnaît par
préfixe de racine (`revision/perimetre.go:171-173`, `couches.go:20-28`) : un sous-paquet de
`internal/grammar/` appartient à la couche `grammar` et entre chez `killsource`/`objectives` **par la
valeur de `grammar.Rev`**, pas par ses octets.

Recommandation [S, à trancher à l'ADR — question ouverte 2 de la spec] : un paquet feuille
`film/internal/grammar/lecture` (types seuls, n'importe que `film/types`), au lieu de `film/types` :
- la structure EST la sortie de `grammar` : sa forme doit faire monter `grammar.Rev`, et elle seule ;
- `source` et `profile` ne voient pas un changement de forme de la structure ;
- coût : la règle « Go `internal` » laisse `film/replay` l'importer. Ajouter à
  `archlint/film_layers_deps_test.go` la ligne `grammar/lecture` = `coucheGrammar` et une règle
  « ni `replay` ni `decfilm` n'importent `grammar/lecture` » (la spec : `replay` ne voit pas la
  structure, la façade ne grandit pas).

Effet de bord à décider : si `objectives` consomme la structure (statborg, §3.8 L2.6), son périmètre
gagne `amont grammar` (aujourd'hui `amont source` seul, `objectives_perimetre.golden`) : chaque montée de
`grammar.Rev` ferait monter `objectives.Rev` et recuire les calques d'objectifs.

### 3.2 Types proposés (ordre de grandeur mémoire)

```go
package lecture // film/internal/grammar/lecture — feuille, aucune logique

// Etendue : l'unite de provenance (P7). 16 octets ; materialisee seulement dans les faits.
type Etendue struct {
	Chunk  uint16 // numero de chunk
	Paquet uint32 // rang du paquet dans le chunk
	Debut  uint32 // bit de debut dans le payload
	Bits   uint32 // longueur en bits (un record d'image-cle depasse 125 000 bits)
}

type Etat uint8 // P2
const (
	Interprete Etat = iota + 1 // valeur typee capturee pendant la marche
	Delimite                   // etendue connue, sens non publie
	Infranchissable            // largeur inconnue : queue opaque a partir d'ici
)

type ProvenanceLargeur uint8 // P4 au niveau du composant
const (
	Ecrivain        ProvenanceLargeur = iota + 1 // port Ghidra (D-3 regle 1)
	Presumee                                     // largeur par fermeture (DU-9), liste gelee
	ExceptionDatee                               // lecture du jeu ecartee (lecteur_position_exceptions.go)
	Calibree                                     // largeurCalibree (harnais)
	Bouchon                                      // largeurBouchon (harnais, jamais en production)
)

type Composant struct { // 12 octets
	Index uint8             // index d'iteration dans l'archetype (< 64)
	Etat  Etat
	Prov  ProvenanceLargeur
	Val   uint8             // genre de valeur capturee, 0 = aucune
	Debut uint32            // bit de debut dans le payload
	Bits  uint32            // 0 si infranchissable
}

type Genre uint8 // Neuf, Delta, Suppression, Fin
type Liaison uint8 // LueNeuf, ChaineImageCle, Datum, Anticipation (repli), Joker (generation inconnue)
type Preuve uint8 // Ferme (paquet ferme), NonProuve (fermeture refusee), Recupere (couche P8)

type Record struct { // ~40 octets
	Genre    Genre
	Vue      uint8   // rang 0..2 ; image-cle : rang lu dans l'eid
	Liaison  Liaison
	Preuve   Preuve
	Vie      types.LifeKey // (slot, generation), 8 octets
	TI       int16   // archetype, -1 non resolu
	Desync   int16   // index du composant infranchissable, -1
	Debut    uint32  // bit de l'en-tete
	Bits     uint32
	Masque   uint64
	Comps    [2]uint32 // [debut, fin) dans Paquet.Comps (pas de tranche par record)
}

type CauseQueue uint8 // ComposantNonPorte, MessageVueANonPorte, ListeNonLocalisee,
                      // FinDePayloadVueB, ArretVueC(kind, 0xbc, debordement, plafond)
type QueueOpaque struct{ Debut, Bits uint32; Cause CauseQueue; Record int32; Composant int16 }

type Fermeture struct {
	Consommes, Longueur uint32
	Ferme        bool  // oracle vueCFermee
	SortieVueB   uint8 // Terminateur | RejetHorsDatum | RejetDeVue (observateur.go:291-303)
	VueCVide     bool
}

type Paquet struct { // reutilise d'un tour a l'autre (arene) : valide pendant le tour seulement
	Chunk, Index int32
	Type         uint16
	TS           uint64
	Payload      []byte // sous-tranche du chunk (P9 : aucune copie)
	VueA         Vue    // {Debut, Bits, Etat, Genres [ ]uint8}
	Records      []Record
	Comps        []Composant
	VueC         VueC   // entrees de controle avec etendue
	Fermeture    Fermeture
	Queue        *QueueOpaque
}
```

Mémoire [É pour l'existant, S pour la proposition] :

| Forme | Taille | Pièce |
|---|---|---|
| `types.Packet` | 56 o | `types/source.go:23-32` (3 int, 1 uint64, 1 tranche) |
| `grammar.FilmPacket` | ~40 o, réalloué à chaque `ChunkAt` | `film_chunks.go:211-227` |
| `FrameRecord` + `EntityTrace` | ~128 o + tranche de `CompResult` par record | `frame_records.go:149-161`, `traverse.go:32-41` |
| `CompResult` | ~56 o (dont `Name string` 16 o, `Payload any` 16 o) | `traverse.go:19-29` |
| Record delta actuel (4 composants) | **~350 o**, alloué par record | calcul [S] |
| Record proposé (4 composants) | **40 + 4 × 12 ≈ 90 o**, dans une arène par paquet | calcul [S] |

Volumes [É, spec §8] : 54 760 records sur `000d5950`, 97 447 records bipèdes sur `bfecd02b`, 315 251 sur
`4f77afc1`. Matérialiser ~1 M de records Δ d'un BTB coûterait ~90 Mo [S], plus les images-clés (état
complet : jusqu'à 64 composants par entité, ~0,8 Ko par entité [S]). En flux, l'arène est bornée par le
plus gros paquet : la plus grosse image-clé citée fait 1 028 032 bits (`keyframe_datums.go:43-46`),
~1 300 entités, ~1 Mo [S].

### 3.3 API du marcheur

```go
package grammar

// Parametres hors flux (P4) : resolus AVANT la marche, avec leur provenance.
type EnTete struct {
	Profil      profile.Profile        // cle de build (lu)
	Balayage    ProfilDeBalayage       // calibre par killsource (provenance Calibree)
	IDLowBits   ValeurProvenance[int]  // UNE valeur : aujourd'hui 13 en dur ou 10..15 calibre
	MPP         ValeurProvenance[profile.MPPWidths] // relue au profil ou calibree
	I0          ValeurProvenance[profile.I0Layout]  // catalogue ou auto-detection (repli)
}

// Phase K : chaque image-cle, une fois par film, dans l'ordre du flux.
func (c *FilmContext) ImagesCles() iter.Seq2[*lecture.Paquet, error]
// Phase D : chaque trame delta, dans l'ordre du flux, sous l'etat de marche courant.
func (c *FilmContext) Trames(interets lecture.Interets) iter.Seq2[*lecture.Paquet, error]
// Etat de marche en lecture seule, valide pendant le tour (P5).
func (c *FilmContext) Etat() lecture.Entites

// Le distributeur : UNE marche, N canaux. Les canaux ne lisent jamais un octet.
type Canal interface {
	Interets() lecture.Interets         // (ti, index) a interpreter, vues voulues
	ImageCle(p *lecture.Paquet, e lecture.Entites)
	Trame(p *lecture.Paquet, e lecture.Entites)
	Clore()
}
func Distribuer(c *FilmContext, canaux ...Canal) (lecture.Diagnostics, error)
```

Choix et raisons :
- **`iter.Seq2` comme primitive** (Go 1.26, `go.mod`; `iter` déjà employé dans
  `internal/openspartan/iterator.go`) ; **le distributeur** au-dessus, parce que l'interprétation se fait
  PENDANT la traversée : les valeurs sont publiées par les crochets de l'`Observation`
  (`observateur.go:38-275`). L'union des `Interets` décide quels crochets la marche pose — c'est la forme
  paresseuse de P9 qui garde la sémantique actuelle des crochets (différence nulle). Une ré-interprétation
  ultérieure reste possible sur l'étendue (`consumeByNameCapturing` rejoué à `Debut`) pour les outils.
- **Deux phases, pas une** (correction 2 du §2) : K puis Δ ; les créations (aujourd'hui heuristiques)
  avant les records ancrés (le filtre daté change l'ANCRAGE : un en-tête refusé ne consomme pas de bits,
  `delta_biped_walk.go:87-96`, donc filtrer après coup ne rend pas le même ancrage).
- **Durée de vie** : `*Paquet` valide pendant le tour ; `Clone()` pour un test ou un outil. La
  matérialisation complète (`lecture.Film`) est réservée aux tests (P9).
- **Concurrence** : un contexte par goroutine (D-5, `film_context.go:89-93`) ; T6 par
  `deux_films_parallele_test.go` sous `-race` (job `film-race` de J12.6).

### 3.4 Tables d'entités

Le `World` (`world.go:23-80`) est l'implémentation ; la structure en expose une **vue en lecture
seule** `lecture.Entites` : `Archetype(slot) (ti, eid uint32, vue int8, liaison Liaison, ok bool)`. Les
liaisons existantes se projettent : `BindFull` → `LueNeuf` ; `BindImageCle` → `ChaineImageCle` ;
`BindDatum` → `Datum` (lu à position libre : marqué récupéré) ; `BindSoft` par anticipation →
`Anticipation` (repli `repli_liaison_par_anticipation`, 6 587 déclenchements, `MESURES_CLOTURE_J11:118`) ;
`BindWildcard` → `Joker` (`GenAny`). UNE table (correction 1 du §2), la vue en attribut.

### 3.5 Trois états ↔ statuts existants

| Spec (P2) | `ecs_table.tsv` (statique, colonne `status`) | À l'exécution (`CompResult` / marche) |
|---|---|---|
| **interprété** | `porte` (544) ou `partiel` (51) **et** composant demandé par un canal | `Ported=true` et crochet ou `Payload` (`capture.go`) |
| **délimité** | `porte` sans demande ; largeur présumée (DU-9) ; exception datée | `Ported=true` ; `largeurCalibree`/`largeurBouchon` (harnais, vides en production) |
| **infranchissable** | `non_porte` (440), `deser_non_cable` (32), ou branche non portée d'un `partiel` | `Ported=false`, `DesyncAt = i` (`traverse.go:283-285`) |
| (alias) | `alias` (14) | orthographe acceptée par le dispatch |

[É] Comptes : `awk` sur la colonne 6 de `internal/grammar/testdata/ecs_table.tsv`. Le statut est une
CAPACITÉ statique ; l'état est par OCCURRENCE (un `partiel` est délimité ou infranchissable selon la
branche lue). Les vues ont leurs propres infranchissables : vue A non vide (`frame_vue_messages.go:70-74`),
vue C `ArretVueC` (`frame_vue_controle.go:38-58`).

### 3.6 Couche de récupération (P8) ↔ balayages existants ↔ registre

Principe : la récupération lit les paquets ou portions que la marche n'a pas fermés, rend des records
`Preuve = Recupere` avec leur méthode, et compte. Comptes en données côté `grammar` (`ComptesDesReplis`,
`replis_du_film.go:29-95`), versés par `replay` (J8.7). Une entrée par méthode au registre `facts/fallback`.

| Méthode de récupération | Balayage existant | Remarque |
|---|---|---|
| Ancrage d'en-tête bipède | `walkDeltaBipedPayload` (positions + 8 passes + véhicules) | Mutualiser : UN ancrage par film, mémorisé, records avec étendues de composants jusqu'au premier infranchissable. |
| En-tête de création | `matchWorldObjectNewHeader(In)` (ti 35, 37, 40, 42) | Une passe multi-archétypes. |
| Pistes d'objets du monde | `ScanWorldObjectsForBand` (37, 41, 42 et la bande ti=40 des positions) | Une passe sur l'union des bandes. |
| Début de vue B | `marchLocateStrict/Fallback`, `debutDeLaListe` | Aujourd'hui DANS la marche ; repli `repli_localisation_largeur_libre` (28 748). |
| Ancre d'image-clé | élection (`kfCand.betterThan`), `sweepKeyframe` (killsource) | `repli_ancre_d_image_cle_par_election`. |
| Datums à position libre | `candidatsDeDatum` | Liaison `Datum`. |
| Motifs | grenade (`grenade_events.go:305`), xuid (`ResolveXuidToPI`, `chercherMotifs`), armes dans l'image-clé (`keyframe_loadout.go:201`) | — |
| Statborg | `scanFrameAvecReplis` (dans `facts`) | Le descendre en `grammar` (les faits ne lisent plus d'octet). |
| Dead-state, kill-events (killsource) | `scanFilm`, `killEventsAvecArrets` | Déjà nommés et comptés par killsource. |

Point de vigilance [É] : certains comptes publiés dépendent du NOMBRE de traversées (bande bipède
sommée par relevé, §1.6) : mutualiser change `coverage.fallbacks` → changement DÉCLARÉ.

### 3.7 Révisions et ratchets

**Révisions** : les types en `grammar/lecture` entrent dans `grammar_perimetre.golden` (une ligne
`paquet`) ; tant qu'aucune sortie ne change, la règle du plan s'applique : empreinte régénérée à
révision CONSTANTE après G-equiv 0 (précédent J4.0, journal `:1436-1442`). `grammar.Rev`
(`grammar-2026-09-27.3`, `rev.go:110`) ne monte qu'aux lots de comportement ; toute montée fait monter
`killsource.Rev` (chaîne, `rev.go:1-30`) et rouvre le backlog du collecteur — à éviter jusqu'au lot
killsource.

**Ratchets touchés**

| Ratchet | Impact |
|---|---|
| `archlint/film_facade_surface_test.go` (`plafondSurfaceFacade = 187`, `:155` ; `plafondSurfaceReplay = 297`, `:361`) | Aucune hausse visée : le collecteur continue de recevoir des faits par `decfilm` (substitution de `ScanPontDIdentite`, pas d'ajout). |
| `archlint/filmsource_leaf_test.go`, `film_types_leaf_test.go` | Inchangés si la RI n'est pas dans `film/types`. |
| `archlint/film_layers_deps_test.go` | Ajouter `grammar/lecture` ; nouvelle règle d'interdiction pour `replay`/`decfilm`. |
| `no_raw_film_bytes_outside_source_test.go`, `no_raw_film_bytes_extraction_test.go` | La marche lit par `Lecteur` ; le déplacement du statborg retire une extraction de `facts`. |
| `no_recomputed_film_context_test.go`, `no_film_reread_test.go` | Respectés (un contexte par film). |
| `film_file_size_test.go`, `film_function_length_test.go` | `replay/film_scan.go` = 490 lignes, `film_context.go` = 484, `frame_infer.go` = 463, `observateur.go` = 451, `keyframe_world.go` = 459 (wc) : tout ajout en fichier neuf. |
| `film_tri_total_test.go` | Aucun nouveau `sort.Slice*` ; après J12.1, comparateurs totaux. |
| `grammar/delta_biped_walk_guard_test.go` | Interdit un 10e site d'ancrage : la récupération mutualisée réutilise `walkDeltaBipedPayload`. |
| `grammar/generations_vivantes_datees_ratchet_test.go` | Accès atemporel limité à deux appelants nommés. |
| `archlint/keyframe_walk_proof_test.go` | La phase K passe par `MarcheDImageCle` (avec preuve). |
| `archlint/film_pont_identite_test.go` | À réécrire quand le pont devient un canal. |
| `replay/observe_test.go` (+ `BuildFromFilmSteps`) | Liste fermée des 46 étapes : à garder stable pour T4. |
| `frame_closure_ratchet_test.go`, `keyframe_closure_ratchet_test.go` | Base de T1 (aucune baisse). Attention : les huit mini-bobines n'ont **aucune trame delta** ; le golden de trames utilise deux bobines killsource (journal `:1437-1440`). |

### 3.8 Migration pas à pas

**Ordre des canaux : la proposition de la spec est à CORRIGER [É].** La marche grammaticale couvre
moins de records bipèdes que le chercheur d'ancres (162 444 contre 97 447 sur `bfecd02b`,
`movement_states.go:16-26`) ; créations et positions ne peuvent donc pas « migrer vers la structure » à
différence nulle tant que la fermeture n'a pas atteint le déclencheur. Ordre sûr :

| Lot | Contenu | Fichiers touchés (principaux) | Preuve | Taille | Risques |
|---|---|---|---|---|---|
| **0** | Préalables : J12 fusionné, références `replay-equiv` re-figées après J12, ADR de la RI (amende 0034 D-1, D-2, D-6, D-7, D-10), instrument de ventilation de la vue C (§4.6) | `research/cmd_fermeture` | — | S | — |
| **1.1** | Paquet `grammar/lecture` (types seuls) + ratchet de couche | neufs : `grammar/lecture/*.go` ; `archlint/film_layers_deps_test.go` ; `grammar_perimetre.golden` | G-arch, empreinte à révision constante | S | Placement (§3.1). |
| **1.2** | **Marcheur phase Δ = refactor de la marche de production** : le pilotage de `movementStateScanner.marcher` (`movement_states.go:184-217`) devient `FilmContext.Trames` ; `ScanMarcheDesTrames` et `FrameClosure` deviennent ses deux premiers consommateurs (fin de la copie de pilotage, `frame_closure.go:159-164`) ; `HeaderBit` posé dans `decodeInferLoop` ; étendues de vues ; sortie de vue B typée | `movement_states.go`, `frame_closure.go`, `frame_harvest.go`, `frame_infer.go`, neufs `grammar/marche_trames*.go` | `replay-equiv` 0 (étapes `movementStates`, `continuousFire`), `frame_closure.golden` identique | M | Correction de la spec : plutôt qu'un marcheur « non consommé » parallèle (contraire à P1), le marcheur EST la marche de production. |
| **1.3** | Phase K : `FilmContext.ImagesCles` sur la mémo existante + étendues d'état complet ; `KeyframeClosure` consommateur | `keyframe_world_marche.go`, `keyframe_closure.go`, neufs | `keyframe_closure.golden` identique, G-equiv 0 | M | Aucun (mémo inchangée). |
| **1.4** | Tests : T1 sur la structure, T3 (toute lecture de mouvement et de tir continu cite une étendue existante), T5 (`fuzz_records_test.go` étendu au marcheur), T6 (`deux_films_parallele_test.go`) | tests | verts | S | Fixtures delta : bobines killsource. |
| **2.1** | Canaux mouvement + tir continu formalisés (`Canal`) | `movement_states.go`, `tir_continu.go`, `replay/film_scan_mouvement.go` | `replay-equiv` 0 | S | — |
| **2.2** | Canaux d'image-clé : armes portées, inventaire, marques, équipes, recensements et bandes, générations vivantes (partie K), table anticipée, liaison | `keyframe_loadout.go`, `inventory_decode.go`, `keyframe_carrier_mark.go`, `player_teams.go`, `world_object_census.go`, `slot_band_*.go`, `generations_vivantes.go`, `keyframe_anticipe.go`, `keyframe_liaison.go`, `offline_biped_band.go` | `replay-equiv` 0 ; comptes de replis : déclarés si la bande n'est plus relevée deux fois | M | Comptes « par relevé » (§1.6). |
| **2.3** | Canaux de tête de vue A (tirs 36, 117, lunette, ramassages, 103, véhicules) : la marche lit la tête une fois par paquet | `fire_events.go`, `transloc_events.go`, `zoom_events.go`, `biped_pickups.go`, `equipment_spawn_events.go`, `event_list.go` | `replay-equiv` 0 | M | Paquets sans tête lisible : même règle qu'aujourd'hui. |
| **2.4** | **Récupération ancrée mutualisée** : un ancrage bipède par film (positions + 8 passes), un par bande ti=40 ; records marqués `Recupere`, nommés, comptés | `delta_biped_walk.go`, `offline_biped*.go`, `ability_*.go`, `camo_state.go`, `grapple_state.go`, `held_weapon_changes.go`, `inventory_delta.go`, `equipment_changes.go`, `equipment_recovery.go`, `offline_aim_only.go`, `replay/film_scan.go` | `replay-equiv` 0 sur les données ; `coverage.fallbacks` déclaré ; banc `b.Loop` | L | Égalité de bande (`fc.BipedSlots()` contre `bipedSlotBand(chunks)`) et de découpage à prouver film par film. |
| **2.5** | Récupération des objets du monde (créations multi-archétypes, pistes sur l'union des bandes) ; calibration MPP gardée en préliminaire | `equipment_creation*.go`, `vehicle_creation.go`, `ground_weapon_creation.go`, `projectiles.go`, `equipment_placements.go`, `replay/build_ground_weapons.go`, `replay/build_vehicles.go` | `replay-equiv` 0 | L | Les largeurs MPP posées entre deux balayages (`gwInstallMPPWidths`). |
| **2.6** | Statborg descendu en `grammar` (récupération), `objectives` consomme | `facts/objectives/statborg.go` (683 lignes), `film.go`, `extract.go`, `grammar` neufs | `replay-equiv` 0, `killsource` inchangé | M | Périmètre d'`objectives` (§3.1). |
| **2.7** | **Changements de comportement déclarés** : (a) morts d'objet sur le marcheur unique (8 → 3 vues, monde unifié) ; (b) canaux delta lus par la marche là où elle couvre mieux (records sans i0 absolu) ; (c) killsource en DERNIER (`runWalk`, timeline, calibration ; contexte partagé avec la cuisson) | `object_deaths*.go`, `facts/killsource/{walk,world,chunks,calibrate,decode}.go`, `replaybuild/kills.go`, `killcollector` | `replay-corpus-gate` + banc de vérité `internal/replayverite` ; `KILLSOURCE_FIXTURES` en local ; `grammar.Rev`, `killsource.Rev` → backlog | L | Ancres Theater et goldens killsource ; 8 vues « lisent au-delà de la trame » (`movement_states.go:78-86`). |
| **3.1** | Retrait des marcheurs redondants (`marchRecordsOf`, localisateurs jumeaux, timeline de killsource, recopies de pilotage) ; contexte unique par cuisson (killsource + replay) et par passe du collecteur | idem | banc, `replay-equiv` 0 | M | — |
| **3.2** | Mesure : durées par étape (`replay/observe.go:90-95`, `LEVELUP_LOG_LEVEL=debug`), pic par `filmproc.Arm` (`filmproc/memguard.go:139`), 3 témoins + 1 BTB | — | rapport | S | — |

---

## 4. La vue C : « terminateur hors cadre »

### 4.1 Définition dans le code [É]

`bloquantDuPaquet` (`frame_closure_classement.go:94-109`) rend, dans l'ordre : vue A non portée → vue B
non terminée (composant ou slot du dernier record, ou fin de payload) → `ArretVueC` ≠ aucun (débordement,
kind 1/2, bloc 0xbc, plafond) → **sinon** `causeTerminateurHorsCadre` (`:26`, `:108`). Il n'est donc
atteint que si : la vue B s'est « terminée » (`hitEnd = true` : terminateur `recEnd`, OU rejet de la
garde de datums ou de vue, `frame_infer.go:281-293`), la vue C a lu jusqu'à son terminateur, et
`vueCFermee` est faux (reste > 7 bits ou non nul, `frame_vue_controle.go:294-305`). L'en-tête du fichier
le dit : « une largeur est fausse QUELQUE PART devant, sans dire où » (`:14-15`).

### 4.2 Lacune de la vue C ou symptôme amont ? Symptôme amont, majoritairement [É + S]

- Les lacunes PROPRES de la vue C ont leurs causes distinctes : kind non porté (4 849 paquets),
  bloc 0xbc (1 549), débordement (145) — `CARTE_FERMETURE_2026-09-26.md:62-75`. Hors cadre = 214 536.
- Le code : « Une vue qui se lit jusqu'à un terminateur sans fermer le paquet a été lue à une position
  fausse — le plus souvent une fin de vue B fausse » (`frame_vue_controle.go:290-293`).
- La sonde P1-S3 (`.ai/V7.5/retours_rejeu_2026-09-23/SONDE_P1S3_tir_continu_vue_controle.md:245-246`) :
  sur `81c02726`, « ~8 900 paquets lisent une vue C vide qui ne clôt pas = fin de vue B fausse ». Une vraie
  vue C est rarement vide (43 612 entrées sur 6 865 paquets clos, ~6,4 par paquet, `:146`).
- NOTE 5.15 (`.ai/V7.5/film_re/NOTE_5_15_RANG_1_FILM_DENSE_2026-09-22.md:40-56`) : sur `bfecd02b`, la vue
  B sortait par REJET 23 452 fois contre 400 terminateurs, 21 988 rejets sur des slots jamais liés.
  Le lot 5.16 a porté la table de datums ; le résidu de `bfecd02b` est « des lectures prises à une
  position FAUSSE » (`frame_infer.go:119-125`).
- NOTE 5.26 (`NOTE_5_26_NAISSANCES_NON_DECLAREES_2026-09-23.md:3-10`, `:138-148`) : les trames
  « s'abandonnent au premier en-tête qui désigne une entité NÉE ET MORTE entre deux images-clés » (801
  eid sur `bfecd02b`) ; les deux dernières pistes sont fermées par l'écrivain ; reste 35,4 % / 53,7 % des
  ticks perdus. Le lot M4b (`debut_de_liste.go:5-20`) a mesuré la cascade : un bipède né en milieu de
  chunk dont le NEW est sauté fait rejeter 438 paquets sur `81c02726`.

Mécanisme [S, cohérent avec ces pièces] : une naissance non lue (NEW sauté en tête de liste, ou porté
par un paquet que la marche n'a pas traversé) laisse l'entité hors de la table ; chaque delta suivant de
cette entité fait sortir la vue B par rejet ; la vue C est alors lue au milieu de records de vue B ; un
`R(1) = 0` y passe pour un terminateur ; le paquet ne ferme pas. S'y ajoutent les largeurs fausses non
détectées (une traversée qui « tombe juste ») et, plus marginalement, une vue C lue elle-même de travers
(forme courte de `DAT_145121140`, `frame_vue_controle.go:275-280`). **Ce n'est pas un trou de grammaire
de la vue C** ; c'est le défaut de modèle d'entités et de traversée de la vue A / vue B.

### 4.3 Part du déclencheur qui en dépend [É]

Calcul sur les TSV de référence (`.ai/V7.5/film_re/carte_fermeture_2026-09-26/`, colonnes `utiles`,
`utiles_fermes`, et `utiles_en_jeu` des bloquants) :

| Build | Records utiles non fermés | Dont « hors cadre » | Part |
|---|---|---|---|
| HI_1_13_0 (9 films) | 405 920 | 375 607 | **92,5 %** |
| HI_1_10_0 | 580 612 | 571 066 | 98,4 % |
| HI_1_11_0 | 180 510 | 174 522 | 96,7 % |
| HI_1_12_0 | 85 739 | 18 226 | 21,3 % (premier : `ti=43 i35`, 67 207) |
| HI_1_4_1 | 141 878 | 133 969 | 94,4 % |
| HI_1_8_0 | 179 049 | 171 637 | 95,9 % |
| HI_1_9_0 | 167 246 | 162 025 | 96,9 % |
| version-31 | 246 008 | 214 062 | 87,0 % |
| version-33 | 375 793 | 350 256 | 93,2 % |
| **Total** | **2 362 755** | **2 171 370** | **91,9 %** |

Sur HI_1_13_0 (2 007 507 utiles, 1 601 587 fermés à la référence) : 95 % exige 1 907 132 fermés, soit
**+305 545**, c'est-à-dire **81,3 %** des records en jeu derrière « hors cadre ». Fermer toutes les
AUTRES causes (30 313) ne mène qu'à 81,3 %. Le déclencheur dépend donc **entièrement** de cette cause.

Deux biais, à ne pas arrondir en faveur du déclencheur :
1. **Dénominateur** : seuls les records LUS comptent (`MESURES_CLOTURE_J11:91-93`) ; corriger une fausse
   sortie de vue B fera LIRE des records aujourd'hui invisibles.
2. **« Utile » sous-compté [É]** : `cmd_fermeture/table.go:84-87` (`product_use` non vide et ne
   commençant pas par `aucun`) ; à la tête, 36 lignes qualifient (35 `porte`, 1 `partiel`), et sont
   marqués `aucun` : `ti=35 i29/i54/i57/i62` (états de mouvement publiés depuis le schéma 65), `ti=35 i1`
   (saut dérivé), `ti=9 i0` (« l'équipe du rejeu vient de la base », alors que `film_scan.go:423-425`
   dit que le film est la SEULE source d'équipe), `ti=40 i0..i2` (véhicules), `ti=13 i0..i3` (zones),
   `ti=42 i0`. La spec annonce 44 composants utiles.

### 4.4 Recommandation

- **Instrument d'abord** (recherche, sans production) : ventiler « hors cadre » dans `FrameClosure` par
  sortie de vue B (terminateur / rejet hors datum / rejet de vue — compteurs déjà dans
  `observateur.go:291-303`), par vue C vide ou non, et par existence d'une naissance non lue de l'eid
  rejeté. Ajouter le dénominateur des entrées de contrôle utiles. Corriger `product_use`.
- **Décision utilisateur requise** : le chantier « vue B / naissances non déclarées » est le résidu film
  dense, CLOS le 23/09 (« NE PAS ROUVRIR sans décision user », mémoire du projet), et la cause commune des
  14 exceptions datées de J6 est renvoyée à une campagne de recherche (découverte 8.27).

---

## 5. Conflits avec J12 et ordre imposé

| Item J12 (plan `:1134-1159`, journal `:1683-1687`) | Ce qu'il touche | Recouvrement avec la RI |
|---|---|---|
| J12.1 `go fix` + 188 tris convertis | Tous les fichiers du cliquet `archlint/film_tri_total_test.go` (189 appels nommés « fichier:fonction ») et toute boucle à trois clauses | Fichiers de la RI portant des tris : `movement_states.go`, `tir_continu.go`, `object_deaths.go` (2), `object_deaths_march.go`, `offline_biped_band.go`, `biped_creation.go`, `killsource/world.go`, `objectives/statborg.go` (2) ; boucles : `frame_harvest.go`, `frame_vue_controle.go`, `debut_de_liste.go`, `offline_biped.go`, `killsource/walk.go`, `statborg.go`. |
| J12.3 contexte + `slog…Context` ; plus de `slog` dans `grammar`/`facts` (diagnostics typés remontés) | `replay/film_scan.go` (34 appels `slog`), `film_scan_mouvement.go` (3), `film_scan_pont.go`, `build_from_film.go`, `replaybuild` (`BuildBytes` crée son `context.Background()`, `replaybuild.go:243` ; `kills.go:48`) ; `grammar/film_context.go`, `world.go`, `grenade_events.go`, `registry_fingerprint.go` ; `killsource/decode.go` (6) ; `objectives/statborg.go`, `film.go`, `named_*.go` | **Cœur de la RI** : l'orchestration (`film_scan*.go`), la table d'entités (`world.go`), le contexte (`film_context.go`), killsource. Les « diagnostics typés » de J12.3 sont la section Diagnostics de la structure (§4 de la spec) : la RI doit les reprendre, pas en créer une seconde forme. |
| J12.4 22 variables de paquet dé-exportées | ex. `replay.BuildFromFilmSteps` (`observe.go:45`), `grammar.GrenadeTypeIDsByRank`, `WeaponHitDistanceEdges` | La liste fermée des étapes, outil de T4. |
| J12.5 docs (13 affirmations fausses, chemins `.ai/`, ADR 0034 réorganisé avec annexe) | commentaires de `grammar`, ADR 0034 | L'ADR de la RI amende 0034 : à écrire sur la version réorganisée. `world.go:32` (trois tables) est une affirmation fausse à corriger. |
| J12.6 job CI `film-race` | `deux_films_parallele_test.go` | Prérequis de T6. |
| J12.7 tag `research` sur 218 fichiers | tests `*_research_test.go` de `grammar` | Les tests de la RI suivent la règle du tag. |
| J12.1 à J12.7 : goldens | `grammar_rev.golden`, `*_perimetre.golden`, références `replay-equiv` (J12 change quatre digests d'étape sans effet publié : killsource, drapeau, crâne, socles — journal `:1685-1687`) | Toute preuve « différence nulle » de la RI se fait contre les références POST-J12. |

**Ordre imposé** : (1) J11 restant (J11.4 vague locale, J11.5 vérification visuelle — cases `[ ]`,
plan `:1121-1129`) et fusion de J12 dans `feat/v75` ; (2) re-figeage des références `replay-equiv` à la
tête post-J12 ; (3) ADR de la RI sur l'ADR 0034 réorganisé ; (4) lot 1.1. Démarrer avant J12 garantit
des conflits sur `film_scan*.go`, `world.go`, `film_context.go`, `killsource/decode.go` et sur chaque
golden d'empreinte.

---

## 6. Mémoire et performance

### 6.1 Chiffres mesurés trouvés dans le dépôt [É]

| Mesure | Valeur | Source |
|---|---|---|
| Référence avant refacto (2026-09-02) | 2 min 24 à 2 min 49 par film ; `playerIndices` 35-40 s ; socles 12,7-15,2 s ; poses 11,8-14 s ; chaque scanner Δ 6,6-7,9 s ; killsource 7,9-9,4 s ; pics 0,18-0,20 Gio ; BTB `1c4c63c2` tué à 4,09 Gio | `.ai/V7.5/MESURES_CUISSON_PERF.md:15-31` |
| Après contexte partagé et boucles chaudes (2026-09-03) | 15,7 / 18,6 / 18,2 s (−89 %) ; BTB `084a804d` 19 min 54 → 1 min 40 ; pics 0,17-0,43 Gio | `:170-191` |
| Bombes bornées (lot 4b) | `51101d1d` 4,9 s / 0,08 Gio ; `a349fea8` 1 min 48 / 0,48 Gio ; `1c4c63c2` 1 min 54 / 0,68 Gio ; `60ae07c4` 25 s / 0,34 Gio | `:193-215` |
| « 7,9 Go » | `51101d1d` le 2026-08-24 : 7,9 Go en 2,6 s, dû à `NamedEventsFrom`/`incrementTimes` (2 163 333 677 événements, ~26 Gio sans garde), corrigé le 2026-09-03 | `.ai/V7.5/REGISTRE_REPORTS.md:492`, `filmproc/doc.go:9`, `replay/testdata/equivalence/BOMBES.txt:16` |
| Décodage contre rejeu depuis les faits (S8, 2026-09-18) | 12,7 s à 2 min 37 contre 122 à 355 ms : **95× à 442×** ; faits de 2,7 à 11,9 Mo | `docs/adr/0034-film-decoder-profile-and-layers.md:806-815` |
| G-equiv J2 (2026-09-26) | pics 0,11 à 1,19 Gio, BTB compris | plan, journal `:1430-1434` |
| Carte de fermeture seule (J4.0.5) | 15 à 43 s par film, pics 55 à 215 Mio | `CARTE_FERMETURE_2026-09-26.md:17` |
| Instrument J11.3 (fermeture + gb1) | 20 films en 3 min 19, pics 55 Mio à 1,3 Gio | `MESURES_CLOTURE_J11_2026-10-01.md:8` |
| Banc lecteur de bits | `BitReaderReadBits` ~122 µs/op (~530 Mo/s) ; `TraverseEntity` ~350 ms/op | `internal/grammar/testdata/bench_baseline.txt` |
| Sentinelle mémoire | `filmproc.Arm(tool, giB, …)` pose les plafonds d'un décodage | `filmproc/memguard.go:139` |

### 6.2 Ce qu'une marche unique économiserait, sur pièces

- **Socle incompressible [É]** : `source.Film` garde TOUS les chunks décompressés (`film.go:42-47`).
  La lecture en flux de la RI ne baisse pas ce plancher : les lectures en avant (§2, correction 2)
  interdisent de libérer un chunk après sa marche. Le gain mémoire de P9 porte sur les records (§3.2 :
  ~350 o alloués par record aujourd'hui contre ~90 o en arène), pas sur le film.
- **Où sont les doublons exacts [É]** : marcheur ancré bipède ×10 (même bande, même découpage, même filtre
  daté) ; `i48` relu ×2 ; marche d'ancres d'image-clé ×2 contextes (+ killsource : `ParseRegistryChunk`
  refait, `sweepKeyframe`) ; `TableDeDatums` ×2 ; pistes d'objets du monde sur 5 bandes ; créations sur 4
  archétypes ; temps forts ×3 ; motif xuid ×2 (×3 au collecteur) ; `FilmPacket` réalloué à chaque `ChunkAt`.
- **Ce que la marche grammaticale seule n'économise pas [S]** : les marches grammaticales ne sont que 3 ;
  les 22 balayages bit à bit restent dans la couche de récupération tant que la fermeture n'a pas atteint
  le déclencheur. Le gain de l'étape 3 vient de la **mutualisation de la récupération** (§3.8 L2.4/L2.5) et
  du **contexte partagé** killsource + replay (L3.1), pas de la seule marche.
- **Estimation [S]** : à la référence du 2026-09-02, les scanners Δ pesaient 6,6-7,9 s chacun, dont
  l'essentiel était la bande et le découpage recalculés (supprimés au lot 2) ; après le lot 4 la cuisson
  entière pèse ~15-19 s sur un film de 30 chunks. Passer de ~22 balayages Δ complets à ~5 (un ancrage
  bipède, un ancrage ti=40, une passe de créations, une passe de pistes, les motifs) et de 2 marches
  d'ancres d'image-clé à 1 : de l'ordre de **20 à 40 % du temps de décodage**, davantage sur un BTB où les
  passes bit à bit dominent. À MESURER avant d'engager (L3.2) : durée par étape par `observe.go`, et banc
  `b.Loop` sur les deux bobines à trames delta.

---

## 7. Points à soumettre (questions ouvertes de la spec, réponses proposées)

1. **Nom et lieu** : `film/internal/grammar/lecture` (types) et `FilmContext.ImagesCles/Trames` +
   `grammar.Distribuer` (marcheur) — en français, comme `ScanPontDIdentite` et `MarcheDImageCle`.
2. **Table des entités** : état de `grammar` exposé en lecture seule (`lecture.Entites`), UNE table
   (NOTE 5.16), la vue et la provenance de liaison en attributs.
3. **Seuil du déclencheur** : par build, mais sur une colonne `product_use` CORRIGÉE et avec le
   dénominateur des entrées de contrôle ; la mesure actuelle sous-compte.
4. **Persistance de la structure** : non ; le cache de faits rejoue déjà 95× à 442× plus vite.
5. **Ordre** : J12, références re-figées, ADR, puis lots 1.1 → 3.2 du §3.8 ; la reprise de la vue B /
   naissances (§4) est une décision utilisateur distincte (résidu film dense clos le 23/09).
6. **Changements déclarés prévisibles** : `coverage.fallbacks` (comptes sommés par relevé), morts d'objet
   (8 → 3 vues), killsource (backlog du parc).
