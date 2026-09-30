# Plan : Emprise — la ressource « véhicules » (lot L7 du plan de l'onglet) — 2026-09-28

> Sources, à lire avant tout lot, et qui FONT FOI pour le rendu :
> - maquette de l'onglet `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` (l. 621
>   « Contrôle des ressources », l. 688 fil de la session, l. 933-934 match par match, l. 946
>   « Frags obtenus avec… », l. 997 « Rendement », l. 1269-1270 pastilles et courbes) ;
> - `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md` (§0, §1 D2 / D9 / D10, §2 spécification
>   commune, L4 et L5 : une ressource = une entrée de liste, masquée si absente) ;
> - `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (à lire avant toute affirmation sur
>   l'équipement) ; `.ai/V7.5/HANDOFF_VEHICULES_2026-09-04.md` ; ADR 0034 (décodeur), 0026.
>
> Contrat d'exécution : skill `plan-execution`. Statuts `[x]` / `[~]` réf / `[!]` justifié ; aucune
> case vide à la clôture d'un lot.
>
> Statut : **GO utilisateur le 2026-09-30** (« attaque le plan des véhicules »). Exécution en cours
> sur `wt/emprise` (base `feat/v75`), conditions d'entrée amendées au §1.

## 0. Hors périmètre

- Ligne « pertes » des véhicules (voir D3).
- Sons, sprites, calque du rejeu 2D (chantier véhicules et tourelles, clos côté rejeu).
- Halo 5 (pas de film) : la ressource n'existe pas (règle D10 de l'onglet).
- Rattrapage prod : fait par l'utilisateur après le déploiement.

## 1. Conditions d'entrée (vérifiées par le superviseur avant L7.0 ; une seule manquante = pas de départ)

- **E1 (amendée le 2026-09-30 par le superviseur)** : `feat/suite-audit-decodeur` n'est PAS attendue.
  Elle est en plein travail dans une autre session (six sous-branches actives le 2026-09-30) et la
  fusionner de ce côté lui imposerait un rattrapage en cours de lots. À la place, une règle de
  frontière : L7 NE MODIFIE AUCUN des fichiers qu'elle réécrit (`film/replay/vehicle_rides*.go`,
  `build_vehicles.go`, `grammar/vehicle_occupancy*.go`, `film/replay/document_vehicles.go`) ; il LIT le
  calque véhicules du document de rejeu (`vehicles[].rides[]`, `family`, `part`, `carrier`) et accepte
  tout schéma ≥ 67 (71 sur `feat/v75`, 76 sur l'audit). Quand l'audit fusionnera, la dérivation se
  rejoue par sa commande de rattrapage ; aucun conflit de fichier attendu.
- **E2** Soirées témoins (relevé du superviseur du 2026-09-28 sur une copie de la base locale,
  frags dont la source est de classe véhicule, JGtm avec Madina97294 ou Chocoboflor dans le même
  camp) :
  - **principale : 24/07/2026**, Big Team Battle, 4 matchs (`fccc61cd` Launch Site, `879a4dba`
    Fortitude, `5676a9ba` Insolence, `4f77afc1` Flood Gulch — ce dernier est le film de référence du
    lot 5.10 de l'occupation), 131 frags de classe véhicule ;
  - **secondaire : 01/09/2026**, Quick Play, 4 matchs à véhicules (`f2966f08` et `7b0d89c4`
    Behemoth, `4ecdf3e7` High Ground, `bfecd02b` Snowbound), 30 frags de classe véhicule.
  Films en cache pour les huit ; artefacts au schéma 61 (sans occupation lue). Au total, la
  composition a 59 matchs à frags de véhicule en local (Snowbound, Behemoth, High Ground, Isolation
  en Quick Play surtout). L'utilisateur ne joue pas en général de modes à véhicules en escouade,
  mais d'autres utilisateurs de l'application le peuvent (2026-09-28) : la ressource est gardée.
- **E3 (amendée)** : les mesures de L7.0 construisent le document des huit témoins EN MÉMOIRE (comme
  le lot V0 du plan des vies) ; la recuisson réelle des huit artefacts (serveur arrêté, un film à la
  fois) n'est faite qu'à la clôture L7.5, pour la vérification sur données réelles.

## 2. Décisions tranchées

- **D1 — Lieu du calcul : dérivation de l'artefact** (nouvelle famille de `Deriver`,
  `sync/replayartifacts/derivations.go`), comme les deux autres ressources des mêmes cartes (bonus :
  famille d'usage ; armes spéciales et râteliers : `padtiers`). Une ressource calculée au sync
  pendant que ses voisines le sont à la cuisson donnerait deux populations de matchs à une même
  piste « Contrôle des ressources » et à une même case « sans film ». L'écart à la décision du
  2026-09-07 (« les données d'un match en base sont complètes au sync ») est donc celui de l'onglet
  entier, consigné au plan des vies (§7) et à statuer hors de ce plan. Précédent de forme :
  `match_flag_grabs_net` (famille, table append-only, capability fine, commande de rattrapage, ne
  redécode rien).
- **D2 — Une prise** = une vie de véhicule qui passe à un camp : le premier épisode (`rides[]`) d'un
  joueur de ce camp dans cette vie de véhicule, tout siège confondu ; un retour au même camp après
  un passage adverse est une nouvelle prise. Le joueur crédité est l'occupant de ce premier épisode
  (conducteur d'abord, siège 0, si deux épisodes commencent à la même image). Les épisodes publiés
  par le calque comptent tous (`src = film` et `src = proximity` : le calque a déjà écarté les
  replis contredits) ; la part `proximity` est publiée en couverture.
- **D3 — Pas de ligne « pertes ».** La maquette l. 1270 : « Les armes spéciales et les véhicules n'ont
  pas de perte mesurable par prise : leurs pastilles restent pleines ». La destruction n'est datée
  que sur 16 vies de véhicule sur 329 (relevé du 2026-09-28).
- **D4 — Temps à bord** = somme des épisodes (images × intervalle d'image), par camp et par joueur :
  barre fine « exposition » de « Frags obtenus avec les ressources » ; « Rendement face à
  l'adversaire » = frags par minute à bord.
- **D5 — Frags depuis un véhicule** = frags dont la SOURCE est de classe `vehicle` ou `turret`
  (`match_kill_events_latest`, classe lue par le registre qui sert déjà « Répartition des frags » et
  « Outils de destruction » : une seule définition dans l'app), tout le lobby, camp du tueur par
  `match_participants`. Écrasements (`CollisionDamage`, sans clé de registre) exclus, comme dans la
  Répartition des frags. C'est la lecture annoncée par le plan de l'onglet (L7 : « lecture de
  `match_kill_events_latest` sans filtre de tueur »).
- **D6 — Périmètre des véhicules** = toute vie de véhicule que le calque publie, tourelles fixes
  comprises ; le décor exclu par la MÊME règle que le rejeu (`vehicle_scenery.go`, appelée, jamais
  recopiée) ; une pièce montée (`part = turret`) appartient à son porteur. Grille « match par
  match » par famille (Warthog, Ghost, …) ; famille inconnue → « Véhicule inconnu ».
- **D7 — Couleur** : jeton `resource-vehicle` (déjà dans les quatre palettes depuis L0).
- **D8 — Un artefact sans occupation lue** (schéma < 67) : le match est « véhicules non mesurés »
  (même traitement que « sans film » pour cette ligne seulement), jamais zéro.
- **D9 — Rendement sans biais (superviseur, 2026-09-30, sur les réserves de L7.0)** : sur tout le
  lobby, seuls 62 % des frags de classe véhicule tombent dans un épisode publié (81 / 131 ; la part
  `proximity` pèse 52,8 % du temps à bord). Diviser TOUS les frags de classe véhicule par le seul
  temps à bord publié gonflerait le rendement. Donc : la barre « part des frags » garde D5 (tous les
  frags de classe véhicule, comme la Répartition des frags) ; le RENDEMENT (frags par minute à bord)
  ne compte au numérateur que les frags de classe véhicule tombés PENDANT un épisode publié de leur
  tueur, calculés à la dérivation par la même jonction d'horloge que les épisodes d'équipement
  (`replaybuild/equipment_episode_kills.go`, réutilisée, pas recopiée) ; la part appariée est
  publiée en couverture. Même principe que la correction R1 de l'onglet (numérateur et dénominateur
  sur la même population).
- **D10 — Épisodes sans xuid (bots)** : ni prise ni temps ; comptés en couverture.

## 3. Organisation

Celle du plan des vies (§3) : worktree `wt/emprise`, un exécuteur Opus par lot, lots séquentiels
(un lot ne commence qu'une fois le précédent clos et vérifié), gate commun identique (dont
`lefthook run pre-push` sur les lots web et la clôture), plus `go test -tags=integration -p 1` des
paquets touchés pour L7.2 et L7.3 ; commits `feat(emprise-vehicules/<lot>)`. Reprise de session :
même protocole que le plan des vies (§5).

## 4. Lots

### L7.0 — Témoin et mesures (superviseur puis exécuteur) · rapide

- [x] L7.0.1 Documents des huit témoins construits en mémoire (E3 amendée) ; frontière E1 tenue.
  Les huit documents sortent au schéma 71 (code actuel) ; aucun fichier de la frontière E1 touché
  (`git status` : deux tests de recherche neufs, rien d'autre).
- [x] L7.0.2 Sur les matchs du témoin : prises (D2), temps à bord (D4), part `proximity`, frags de
  classe véhicule par camp (D5). **Seuils écrits avant la mesure : ≥ 90 % des frags de classe
  véhicule d'un joueur de l'escouade tombent pendant un de ses épisodes publiés ; part `proximity`
  ≤ 50 % des épisodes.** Manqué : STOP, rapport à l'utilisateur. **Verdict : les deux seuils sont
  tenus à la lettre** (45,2 % ; 11/11), avec les réserves nommées ci-dessous.

**Instruments** (tests de recherche, sautés sans données, rien écrit dans `data/`) :
`platform/duckdb/emprise_vehicules_l70_frags_research_test.go` (`TestEmpriseL70Frags` : frags D5
par la requête, le classificateur `halo_infinite.NewKillSourceRegistry` et le résolveur de classe
`resolveWeaponKeyDimensions` du lecteur de la « Répartition des frags », sur copies de `shared` et
`metadata`) puis `replaybuild/emprise_vehicules_l70_research_test.go` (`TestEmpriseL70` : document
en mémoire par `v0Document`, horloge du lot V0.1 — film µs = origine + f × pas, match ms = film ms
− `coverage.bridge.deathOffsetMs`). Prises D2 : par vie (pièce montée repliée sur son porteur),
épisodes triés par `t0` puis siège (conducteur d'abord), une prise à chaque changement de camp ;
camp de l'occupant par `roster[].team` du film (0 discordance avec `match_participants.team_id`
sur les huit matchs).

**Prises (D2) et temps à bord (D4), par match** (camp 0 / camp 1 ; temps en s) :

| Match | Vies (sans épisode) | Épisodes (dont `proximity`) | Prises 0 / 1 | Temps à bord 0 / 1 |
|---|---|---|---|---|
| fccc61cd Launch Site | 15 (12) | 3 (3 = 100 %) | 2 / 1 | 15 / 21 |
| 879a4dba Fortitude | 44 (20) | 46 (14 = 30,4 %) | 16 / 12 | 523 / 258 |
| 5676a9ba Insolence | 60 (33) | 63 (41 = 65,1 %) | 16 / 18 | 543 / 944 |
| 4f77afc1 Flood Gulch | 146 (87) | 111 (44 = 39,6 %) | 32 / 29 | 1262 / 1343 |
| f2966f08 Behemoth | 12 (8) | 8 (3 = 37,5 %) | 4 / 2 | 148 / 45 |
| 7b0d89c4 Behemoth | 10 (6) | 5 (3 = 60,0 %) | 3 / 1 | 289 / 38 |
| 4ecdf3e7 High Ground | 9 (7) | 2 (1 = 50,0 %) | 0 / 2 | 0 / 14 |
| bfecd02b Snowbound | 11 (9) | 3 (0 = 0 %) | 1 / 1 | 158 / 119 |
| **Total** | 307 (182) | **241 (109 = 45,2 %)** | 74 / 66 | 2938 / 2782 |

Familles des prises (total des huit, 140) : warthog 42, mongoose 25, ghost 19, wraith 18, falcon 17,
banshee 9, chopper 4, famille inconnue 4 (879a4dba 2, 4f77afc1 2), shade 2. 3 épisodes sans
xuid (4f77afc1), 0 sans camp, 2 pièces montées orphelines (porteur absent du document).

**Frags de classe véhicule ou tourelle (D5), par camp du tueur** : 24/07 = 101 (fccc61cd 2, 879a4dba
13, 5676a9ba 35, 4f77afc1 51 dont 3 tourelle) ; 01/09 = 30 (f2966f08 4, 7b0d89c4 14, 4ecdf3e7 1,
bfecd02b 11). Par match (camp 0 / 1) : 2/0, 2/11, 15/20, 26/25, 4/0, 12/2, 1/0, 5/6.

**Seuil frags (escouade, tueur = JGtm, Madina97294 ou Chocoboflor)** : 11 frags, 11 pendant un de
leurs épisodes publiés, dès la lecture stricte (bornes incluses ; ±500 ms et ±1 s donnent le même
compte) — JGtm 1 (5676a9ba), 5 (4f77afc1), 5 (bfecd02b) ; Madina97294 et Chocoboflor 0.

**Réserves, à lire avant L7.1** :

1. **L'échantillon du seuil est petit : 11 frags**, tous de JGtm ; les 120 autres
   frags d engin des huit matchs sont ceux du reste du lobby.
2. **Tout le lobby, la couverture des épisodes tombe à 62 % (81/131)** : 879a4dba 9/13, 5676a9ba
   27/35, 4f77afc1 20/51 (39 %), f2966f08 1/4, 7b0d89c4 13/14, fccc61cd 0/2, 4ecdf3e7 0/1,
   bfecd02b 11/11. Cause connue : la primitive n'attribue qu'une part des vies de véhicule (limite
   publiée au calque). Conséquence pour D4 : un « rendement par minute à bord » par CAMP a un
   dénominateur (le temps à bord) plus bas que la réalité ; celui d'un joueur de l'escouade, qui
   tient ses frags dans ses épisodes, n'a pas ce biais. À statuer pour le bloc (L7.3), pas ici.
3. **`proximity` : 45,2 % des épisodes au total, mais 65,1 % à Insolence, 60 % à un Behemoth, 100 %
   à Launch Site.** En temps à bord, `proximity` pèse 3023 s sur 5720 s (52,8 %), au-dessus de 50 %.
   Le seuil du plan porte sur le COMPTE d'épisodes ; il est tenu sur l'ensemble des huit.
4. **Écart avec le relevé du 2026-09-28** : D5 ne compte que les sources à clé de registre (classe
   `vehicle` ou `turret` dans `metadata.weapons`) ; 30 frags de plus du 24/07 (13 à 879a4dba, 17 à
   4f77afc1) ont une source de classe « VEHICULE » SANS clé de registre (les écrasements, exclus
   par D5 comme dans la Répartition des frags) — 101 + 30 = 131, le chiffre du relevé.

Journal : [2026-09-30] L7.0 joué en avant-plan (une passe de 1188 s sur les huit films, pic mémoire
sous le plafond de 8 Gio) ; seuils tenus ; L7.1 autorisé.

### L7.1 — Projection pure (Go) · moyen

- [x] L7.1.1 Fonction pure depuis `ReplayDocument` (vies de véhicule, `rides`, famille, décor) :
  prises par camp et par joueur, temps à bord, par famille ; tests synthétiques (changement de camp,
  sièges simultanés, décor, pièce montée, famille inconnue, artefact sans occupation).

**Livré (2026-09-30)** : `film/replay/vehicle_takes.go` (hors fichiers de la frontière E1) et
`vehicle_takes_test.go` (17 tests). Lieu : `film/replay`, couche de publication du document, comme
les autres projections pures d'un `ReplayDocument` (`FlagTracksOf`, `PlacementDesVies`) ; pas
`analysis/` (il faudrait y importer le type du document ou en dupliquer la forme). Le coût est
connu : l'appelant du lot L7.2 cite de nouveaux identifiants `replay.X` hors de `film/`, donc le
plafond de `film_facade_surface_test.go` monte, avec sa justification datée, dans le commit L7.2.

API : `ProjectVehicleTakes(doc *ReplayDocument) VehicleTakesReport` ; `VehicleTakesReport{Measured,
Reason, Rows []VehicleUsageRow, Coverage VehicleTakesCoverage}` ; ligne = `(Camp, XUID, Family)` →
`Takes`, `AboardMS`, `Episodes`, `ProximityEpisodes` ; `VehicleFamilyUnknown = "unknown"` ;
raisons `VehicleTakesUnmeasured{Schema,NotScanned,NoInterval}` (D8 : schéma < 67, calque non
balayé ou pas d'image absent → `Measured = false`, aucune ligne ; un film balayé sans véhicule est
un zéro MESURÉ). Le camp est celui du film (`roster[].team`).

Mutations (copie, mutation, rouge constaté, restauration par copie, `cmp` identique) : M1 une seule
prise par vie (`camp != current` → `current == -1`) rouge sur 5 tests ; M2 prise à chaque épisode
(`|| true`) rouge sur 3 ; M3 ordre des sièges inversé ; M4 décor ignoré ; M5 pièce non rattachée ;
M6 doublon de pièce non écarté ; M7 famille vide gardée ; M8 épisodes `proximity` écartés ; M9
plancher de schéma abaissé ; M10 durée sans intervalle d'image ; M11 camp `-1` accepté ; M12 xuid
vide non compté ; M13 balayage ignoré ; M14 tri des lignes inversé ; M15 pièce orpheline gardée
pièce ; M16 constante de provenance dérivée ; M17 ordre du document au lieu de l'ordre de montée.
Dix-sept mutants tués. Deux tests renforcés en cours de route, parce qu'un mutant survivait ou que
l'ordre n'était pas épinglé : l'orphelin vérifie `PartRides == 0`, et un test liste les épisodes à
l'envers.

Non fait ici, et pourquoi : la projection n'est pas jouée sur les huit documents réels — le
rattrapage réel est prescrit au lot L7.5 (E3 amendée) et le test d'intégration de L7.2 lira des
artefacts ; l'instrument de recherche de L7.0 est hors d'`film/` et citerait de nouveaux
identifiants `replay.X` (plafond d'archlint) sans bénéfice avant L7.2.

Journal : [2026-09-30] L7.1 joué après les seuils de L7.0 ; gate verte (film/replay, vet, lint 0,
archlint, gofmt).

### L7.2 — Écriture (Go, persistance — lot sensible) · lourd

- [x] L7.2.1 Table append-only `match_vehicle_takes` (+ vue `_latest`), persister INSERT-only,
  inscriptions aux garde-rails (`no_art_patterns_test.go`, `append_only_state_guard_test.go`,
  regexp de `no_raw_rating_reads_test.go`, `compaction_registry.go` + e2e), ordre des migrations.
- [x] L7.2.2 Famille dans `Deriver` (une lecture de l'artefact, même marque par match), capability
  fine `film.vehicle_usage` dans `capabilities.toml` de Halo Infinite (et `not_exposed` dans la
  fixture du titre synthétique).
- [x] L7.2.3 Commande `levelup backfill-vehicle-takes` (modèle `backfill-pad-tiers` : reprenable,
  `--dry-run`, `--match`, serveur arrêté), documentée dans `docs/COMMANDS.md` et `docs/FR/COMMANDS.md`.
- [x] L7.2.4 Tests d'intégration (`-tags=integration -p 1`) : écriture, vue, famille dans `Deriver`,
  capability absente, D8, D9, D10, reprise de la commande.
- [x] D9 Appariement des frags de classe engin aux épisodes (lecture en base avant l'écriture,
  jonction d'horloge de `equipment_episode_kills.go` extraite en helper partagé, non recopiée).

**Schéma** (`match_vehicle_takes`, migration `shared_create_vehicle_takes`, une PK séquence, UN index
`match_id`) : `id`, `match_id`, `decode_pass`, `written_at`, `row_kind` (`take` | `match`), `camp`,
`xuid`, `family`, `takes`, `aboard_ms`, `episodes`, `proximity_episodes`, `frags`, puis les colonnes
de COUVERTURE recopiées sur chaque ligne de la passe : `measured`, `unmeasured_reason`, `doc_schema`
(D8), `episodes_read`, `episodes_unnamed` (D10), `episodes_no_camp`, `frags_read`, `frags_reason`,
`frags_total`, `frags_unmatched` (D9). Vue `match_vehicle_takes_latest` = dernière passe ENTIÈRE par
match. Justification : (1) UNE table plutôt que deux — la passe s'écrit en une transaction et la vue
retient une passe entière ; avec une table « prises » et une table « couverture », un match re-projeté
SANS prise servirait les prises de la passe précédente (arbitrage par clé, piège ADR 0026). (2) La
ligne `match` (camp -1, xuid et famille vides) existe TOUJOURS : elle porte « zéro mesuré » contre
« non mesuré » (D8) et fait retenir la passe. (3) Couverture recopiée sur chaque ligne, comme
`pads_confirmed` : une ligne se lit seule. (4) `episodes_unnamed` et non `…_no_xuid` : le semeur de
l'e2e de compaction sème tout nom en `xuid` comme une chaîne (colonne entière refusée).

**Fichiers** : `film/replay/vehicle_takes.go` (+ `Rides`, `Frags`), `vehicle_takes_frags.go` (+ test),
`equipment_episode_kills.go` (helpers `frameOfFilmMS` / `frameInWindow`, partagés) ;
`domain/frag_distribution.go` (`IsEngineFragClass`, une seule définition, reprise par
`service/fragdist`) ; `persist/vehicle_takes_persister.go` (+ tests), `batch.go`, `builder.go`,
`combined_persister.go` ; `migration/steps_shared_vehicle_takes.go`, `order.go`,
`compaction_registry.go` ; `games/adapter.go`, `capabilities.go`, `halo_infinite/adapter_data.go`,
`config/titles/{halo_infinite,synthetic_title_b}/mappings/capabilities.toml` ;
`sync/replayartifacts/vehicletakes.go` (+ tests), `derivations.go`, `journal.go` (compteurs ADR 0009) ;
`cmd/levelup/cmd_backfill_vehicle_takes.go` (+ test), `main.go` ; docs `COMMANDS.md` EN et FR ;
skill `db-schema` ; ratchet `archlint/film_facade_surface_test.go` 293 -> 298.

**Câblage** : `Deriver` lit les artefacts une fois, PRÉPARE la famille (porte `film.vehicle_usage`,
projection pure, lecture des frags par le segment de lecture `WithRead` — avant tout segment
d'écriture), puis l'écrit dans le segment unique après les niveaux d'armes. Frags : source mesurée de
`match_kill_events_latest` -> `KillSourceClassifier` -> `weapons.ClassesByKey` -> `IsEngineFragClass`
(même définition que la Répartition des frags), joints aux épisodes par l'horloge des épisodes
d'équipement (`frame = (time_ms - originMs) / pas`, bornes incluses). Lecture de base en échec = match
mesuré ni écrit ni marqué ; aucun événement de mort à source mesurée = prises écrites, frags « non lus »
(`no_kill_source`), jamais zéro. `DerivationsRev` NON montée, comme `padtiers` / `flaggrabsnet`
(ajoutées après `derivations-2026-09-06` sans la monter) : la monter rejouerait toutes les dérivations
(dont les positions) du parc pour une famille neuve ; le parc passe par `backfill-vehicle-takes`, qui ne
redécode rien. Capacité `film.vehicle_usage` déclarée pour Halo Infinite seulement ; aucun `slug ==`.

**Horloge vérifiée sur données réelles** (sonde jetable, supprimée) : sur 25 artefacts locaux à
équipement, `time_ms` de `match_kill_events_latest` joint par `originMs` retrouve 228 des 229 frags
comptés `K` par la cuisson (le manquant est un schéma 29) : la jonction est la bonne.

**Mutations** (copie, mutation, rouge constaté, restauration par copie, `cmp` identique) — appariement :
MP1 bornes exclusives, MP2 tueur ignoré, MP3 frag posé sur la mauvaise famille, MP4 dernier épisode au
lieu du premier, MP5 origine ignorée, MP6 lecture des événements ignorée ; persistance : MV1 doublon non
refusé, MV2 somme des frags non contrôlée, MV3 passe non mesurée avec lignes, MV4 sans ligne `match`, MV5
vue en ordre croissant, MV6 registre de compaction sur une mauvaise colonne ; famille : MD1 porte toujours
fermée, MD2 toujours ouverte, MD3 échec de lecture non inscrit au bilan, MD4 sans filtre de classe, MD5 D8
non honoré, MD6 `Deriver` sans écriture, MD7 D10 non compté, MD8 tueur bot compté, MD9 « sans événement »
lu comme zéro ; commande : MC1 reprise ignorée, MC2 `--force` ignoré, MC3 `--dry-run` écrit ; capability :
MK1 clé absente du TOML, MK2 fixture synthétique ouverte. Quatre mutants ont d'abord survécu ou rougi
pour la mauvaise raison et ont été refaits : MP3 et MP4 (tests renforcés : frag dans la seconde famille
triée, deux épisodes qui se chevauchent), MC2 (le test passait la clé de reprise vide : il passe
maintenant la clé fournie ET `--force`), et MD1/MD2/MD4 (import inutilisé : refaits compilables).

**Gate** (rejouée après la dernière modification) : `go test` par lots (games, migration, persist,
domain, service, analysis, replaybuild ; sync, platform, cmd ; le reste) vert — deux flakes de charge hors
périmètre rejoués seuls et verts (`sync/skill` rafales 2 s, `mapcatalog` overlay concurrent) ;
`go test -tags=integration -p 1` : `sync/replayartifacts`, `persist`, `migration`,
`games/halo_infinite/migrations`, `cmd/levelup` verts, et `sync` (découpé par initiale A-B, C, D-F, G-M, N-R, S,
T-Z : le paquet dépasse 600 s d'un bloc) vert ; `make go-api-lint` 0 (un gocyclo 20 sur la
validation a été scindé) ; `go test ./internal/archlint/...` vert ; `gofmt -l internal cmd` vide.

Journal : [2026-09-30] L7.2 joué en avant-plan ; schéma, famille, commande et tests livrés ; L7.3 autorisé.

### L7.3 — Bloc Emprise (Go) · moyen

- [x] L7.3.1 Ressource `vehicle` dans `analysis/squademprise` (`resourceOrder`, prises, temps,
  frags D5, rendement par minute à bord, grille par famille), lecture bornée (ADR 0036), contrat.

**Livré (2026-10-01)** : `analysis/squademprise/vehicles.go` (+ modifs `match.go`, `build.go`, `objects.go`, `habit.go`, `input.go`, `production.go`), `domain/squad_emprise_vehicles.go` (+ champs dans `squad_emprise.go`), `port.SquadVehicleRepository`, `platform/duckdb/squad_vehicle_repo.go`, `service/teammates/teammates_service_emprise_vehicles.go`, `service/squadagg/vehicle_labels.go`, câblage `registry_pages_home.go` sous `film.vehicle_usage`, contrat régénéré (`openapi.yaml`, `generated.ts`), ADR 0036 (garde-rail I2 ajouté).

**Forme JSON** (ressource `vehicle`, dans les listes existantes du bloc) : `resources[]` `{resource:"vehicle", taken:{us,them}, matches_measured}` (absente sans prise) ; `objects[]` et `matches[].resources[].objects[]` un objet par FAMILLE (`key` = famille du calque, `unknown` si inconnue ; `label` seulement pour les familles qualifiées par le manifeste du titre ; `taken`, `aboard_ms:{us,them}`, `squad[]` = un par joueur des fiches puis le reste du camp, avec `taken` et `aboard_ms`) ; `matches[].vehicles` = `measured`|`not_measured` (+ `vehicles_reason` : `no_pass`, `team_unknown`, ou la raison de l artefact) ; `production[]` `{resource:"vehicle", kills (D5, tous les frags de classe véhicule ou tourelle, par camp du tueur), exposure:{kind:"aboard_ms", value (temps à bord), kills (frags APPARIÉS)}, yield_us/yield_them (frags appariés par minute à bord), relative_gap}` ; `habit` : part des prises par soirée ; `vehicles` (bloc) = couverture `{unavailable?, matches_measured, matches_not_measured, episodes_read, episodes_unnamed (D10), episodes_no_camp, proximity_episodes, frags_matches, frags_total, frags_paired, paired_share}`. Ressource absente (titre sans capability) : pas de clé `vehicles`, `matches[].vehicles` vide ; lecture en échec : `vehicles.unavailable = "load_failed"`.

**Décisions d exécution** : (1) le périmètre commun frags / temps à bord / rendement = matchs mesurés dont la dérivation a apparié les frags ET dont les événements de mort ont été lus (ou sans frag d engin) ; les prises et les objets restent sur tous les matchs mesurés ; (2) les frags par camp (D5) se lisent au moment de la requête (`match_kill_events_latest`, classificateur + `resolveWeaponKeyDimensions`, les mêmes que la Répartition des frags) pour les seuls matchs dont la passe compte des frags d engin ; le total écrit (`frags_total`) sert de couverture ; (3) un tueur sans camp compte pour l adversaire (règle du paquet) ; (4) un registre d armes sans classe rend les matchs « non lus », jamais zéro ; (5) zéro mesuré = ressource sans entrée et `matches[].vehicles = measured` ; (6) ligne « match par match » `vehicle` masquée côté web quand aucune carte n en porte.

**Mutations** (copie, mutation, rouge constaté, restauration par copie, identité constatée) : pur MV1 camp inversé, MV2 frags non appariés gardés, MV3 événements ignorés, MV4 rendement sur tous les frags, MV5 tueur sans camp chez nous, MV6 absence de passe non distinguée, MV7 camp inconnu ignoré, MV8 véhicules non versés sans film, MV9 double compte avec film, MV10 épisodes sans xuid non comptés, MV11 temps à bord sans prise perdu, MV12 habitude sans véhicules, MV13 ordre des ressources, MV15 libellé de toutes les familles, MV16 temps par joueur perdu, MV17 échec de lecture tu, MV18 part appariée fausse, MV19 production absente ; dépôt MR1 fenêtre des prises non bornée, MR2 événements de tous les matchs, MR3 classe ignorée, MR4 tueur bot compté, MR5 registre sans classe lu comme zéro, MR6 classificateur absent non gardé, MR7 table absente en panne, MR8 ligne match prise pour une prise, MR9 tourelle hors engin, MR10 camp de tous les participants ; service MS1 non supporté lu comme panne, MS2 échec tu, MS3 lecture non appelée, MS4 joueur de la page perdu, MS5 périmètre vide ; câblage MW1 porte ouverte, MW2 sans classificateur ; libellés ML1 langue inversée. Les mutants MV6, MV7, MR3, MR9, MS1, MS2 ont été refaits compilables (import inutilisé ou variable inutilisée au premier jet).

**Gate** (après la dernière modification) : `go test` vert sur analysis, domain, port, service, api, platform, games, migration ; `go test -tags=integration -p 1` vert sur service/teammates, api/wire et platform/duckdb (découpé par initiale : le paquet dépasse 600 s d un bloc) ; `make go-api-lint` 0 issue ; `go test ./internal/archlint/...` vert (le garde de la Campagne a demandé l exclusion sur la lecture du camp : faite) ; `gofmt -l internal cmd` vide ; `make openapi-check` vert.

Journal : [2026-10-01] L7.3 joué en avant-plan ; L7.4 autorisé.

### L7.4 — Cartes (web) · moyen

- [ ] L7.4.1 `RESOURCE_ORDER`, `resourceColors.ts`, textes FR / EN dans un fichier de textes neuf
  (`empriseStrings.ts` à 490 lignes), lignes des sept cartes (maquette l. 621, 688, 933-934, 946,
  997, 1269-1270), pastilles pleines (D3).

### L7.5 — Clôture (superviseur)

- [ ] L7.5.1 Revue adversariale, rattrapage local, vérification sur le témoin, fusion, CI verte,
  gate visuel utilisateur après fusion ; prod par l'utilisateur.

## 5. Découvertes (à consigner ici, pas à traiter)

- Les compteurs de l'API `VehicleDestroys`, `DriverAssists`, `Hijacks` sont déclarés
  (`openspartan/halo_api_payload.go:134-136`) mais jamais persistés.
- `VehicleTransferDamage` (2 256 frags en local) : sens non établi.
- (L7.0, 2026-09-30) Le verdict de décor (`doc.VehicleScenery`) n'existe PAS dans l'artefact : il
  est posé à la requête par `service.decideVehicleScenery` (zone jouable de la carte), sur des
  vies qui n'ont aucun occupant par construction (`vehicleIsPosedOnly` : `len(Rides) == 0`). La
  dérivation de L7.2 lit l'artefact, donc sans verdict : le décor n'y porte jamais d'épisode, donc
  jamais de prise. La projection honore le verdict quand il est là (test) ; elle ne peut ni
  appeler ni recopier la règle, qui vit dans `service`.
- (L7.0) Les épisodes sans xuid (un bot, ou un slot que le pont n'a pas nommé : 3 à `4f77afc1`) ne
  font ni prise ni temps dans la projection : un bot à bord prend pourtant bien le véhicule à son
  camp. Non traité ; le camp d'un bot se lirait par le slot de sa piste (`Track.Team`), à statuer.
- (L7.0) La couverture des épisodes est de 62 % sur les frags d'engin de tout le lobby (11/11 pour
  l'escouade) : un rendement par minute à bord calculé par camp serait biaisé vers le haut.
- (L7.1) `film_facade_surface_test.go` : la première version du test de recherche L7.0 citait en
  commentaire `replay.VehicleRideSrcProximity` et a fait monter le plafond de 293 à 294 ; le nom a
  été retiré (constante recopiée en littéral, comme `v0SrcLue`).
- (L7.2) La classe d'une source de dégât (`KillSourceClassifier`) n'est armée que par la capability
  `film.kill_source` (`killcollector.ClassifierPourTitre`) : un titre qui déclarerait `film.vehicle_usage`
  sans elle écrirait des prises et des frags « non lus » (`no_classifier`). Halo Infinite déclare les deux.
- (L7.2) Une passe écrite avant l'arrivée des événements de mort (ou avant la recuisson d'un artefact de
  schéma < 67) garde « frags non lus » / « non mesuré » : la reprise de `backfill-vehicle-takes` se clé
  sur la PRÉSENCE en base, donc il faut `--force` après une recuisson (documenté aux deux COMMANDS).
  Le post-sync lit les frags après l'étape kill source (1.57 avant 1.58) et le dépôt d'ouvrier plus tard :
  le cas normal est « frags lus ».
- (L7.2) Le calque de décor (`VehicleScenery`) n'est jamais dans l'artefact (cf. L7.0) : la dérivation ne
  peut pas l'honorer, le décor sans occupant ne porte de toute façon aucune prise.
- (L7.2) Les tests `internal/sync/skill` (rafales bornées à 2 s) et `internal/mapcatalog` (overlay
  concurrent) rougissent parfois sous la charge d'un `go test ./...` complet et passent seuls. Non traité.
- (L7.2) L'e2e de compaction sème toute colonne dont le nom finit par `xuid` comme une chaîne : un
  compteur entier ne doit pas porter ce suffixe (piège, contourné par `episodes_unnamed`).
