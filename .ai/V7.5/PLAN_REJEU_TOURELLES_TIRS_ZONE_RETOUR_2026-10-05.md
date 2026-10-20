# PLAN — Rejeu 2D : tourelles grises, tirs depuis l'avant du véhicule, cercle de retour du drapeau

> Demande utilisateur du 2026-10-05 (points 18, 19, 20 de sa liste), mot pour mot :
>
> 18. « Label sous les "Tourelles automatiques bannies" sur la carte Blizzard à virer, et comme les
>     tourelles sont non jouables il faut les mettre en gris »
> 19. « Les tirs venant des véhicules partent du pion du joueur, pas de la pointe avant du véhicule,
>     c'est à corriger »
> 20. « Quand le drapeau est hors de son socle, s'il est au sol, le drapeau au sol, il faut faire
>     apparaitre le cercle dans lequel doivent se mettre les joueurs pour le retourner plus vite.
>     Demande exprimée 1000 fois »
>
> Précision de l'utilisateur sur le point 20, le même jour : « En mode CTF si un joueur prend le
> drapeau ennemi et pour quelque raison que ce soit, mort ou lâché volontaire, le drapeau se retrouve
> au sol, il doit afficher un cercle en partant de sa base. En jeu, dans Halo Infinite, ce cercle
> (normalement déjà trouvé dans le code) indique aux joueurs à qui appartient ce drapeau qu'ils
> peuvent se positionner dans ce cercle pour renvoyer leur drapeau sur son socle. »
>
> Branche : `claude/turrets-vehicle-fire-flag-b2ba4b` (worktree dédié créé par l'app), base
> `feat/v75` `87cdfa761`. Exécution sous le contrat `plan-execution`. Les exécutants ne committent
> pas et n'écrivent ni ce plan ni `.ai/thought_log.md` : le superviseur le fait.

---

## 1. Diagnostic sur pièces (2026-10-05)

### Point 20 — le cercle existe, et il ne s'affiche pas sur la moitié des parties de CTF

- Le cercle est codé (`flagReturnZone.ts`, règle `doc.flagReturnZone` : rayon 1,3, minuterie 30 s,
  3,1 s à un défenseur, lus dans le script du jeu). Il ne se dessine que pour un drapeau d'ÉQUIPE
  (`if (now.team >= 0)`, règle du drapeau neutre).
- Sur une carte ABSENTE du catalogue de socles (`data/titles/halo_infinite/reference/map_objectives.json`),
  `buildFlagCarries` publie TOUS les portages sur UN drapeau d'équipe -1 (repli nommé
  `repli_index_drapeau_zero_pour_tous`), sans état `home` : le client le prend pour un drapeau
  neutre et ne dessine aucun cercle ; les deux drapeaux sont fusionnés en un.
- Mesure `match_registry` (lecture seule) : **760 matchs de CTF sur 1 512 sont sur une carte sans
  socle d'équipe au catalogue**, dont TOUTES les cartes « Ranked:CTF 3 Captures » récentes de
  l'utilisateur (`70dd38c5` 112 matchs, `5c215079` 108, `78677080` 72, `a54808fb` 54, `74b8c681` 12…).
  Cache des rejeux : `fd247c3f`, `f7b74a65`, `fa70c437`, `92c950ee`, `f1db4a07`, `798d1ff4`
  (`spawns 0`, `carrierTeamUnknown` = tous les portages, un drapeau d'équipe -1). Huit socles du
  catalogue portent en plus `team_index = -1` sans être neutres (Cliffside, Solitude…) : même effet
  sur ces drapeaux (`a32ee8d2`).
- **Le film suffit à retrouver base et camp** (relevé sur `fd247c3f`, `f7b74a65`, `92c950ee`,
  `798d1ff4`, témoin catalogue `eba1e63f`) : les VOLS (`flag_steals`) d'un camp se font tous au même
  point, à moins de 0,7 m, et c'est la base du camp ADVERSE ; les captures d'un camp tombent sur le
  point où l'autre camp vole ; sur le témoin, les deux points sont les socles du catalogue. L'équipe
  du porteur est dans le film (`scan.TeamOf`, lot 1.7) ; la vie libre de l'objet renaît AU POINT de
  son socle (0,006 m au plus sur le parc, `flag_neutral.go`).
- Sur les cartes du catalogue, le cercle est dessiné mais peu lisible : disque à 10 % d'opacité,
  anneau de 1,5 px à 45 %, et l'arc de la jauge posé SUR l'anneau dès que le rayon dépasse 20 px —
  l'œil lit une minuterie, pas une zone. Plancher de 4 px sur les grandes cartes (caché par le
  glyphe du drapeau, 19 px de haut).

### Point 19 — l'éclair part du centre du véhicule

- `vehicle_weapons.toml` : 8 armes sur 15 n'ont AUCUN `mount` — canons du Ghost, canons et bombe de
  la Banshee, canons du Chopper (tir vers l'avant) ; LAAG du Warthog, mitrailleuse du Falcon,
  mitrailleuse du Scorpion, tourelle du Wraith (tourelles). Sans montage, `vehicleShotPlacement`
  pose l'éclair au CENTRE du sprite, là où se trouve le cône du conducteur : c'est « le pion ».
- Les tourelles déclarées posent l'éclair sur leur PIVOT (canon du Scorpion : `ay = -0,05`, le centre
  du char) : le canon n'a pas de longueur.
- Le « ! » du tireur (`fireMark.ts`) se pose sur la position INTERPOLÉE du bipède même quand le
  joueur est embarqué (le bipède ne réplique plus : sur `81c02726`, 1 échantillon en 77 s de Ghost),
  alors que son pion est masqué : un « ! » flotte loin du véhicule à chaque tir.
- Mesures des sprites (boîte du corps, repère nez en haut) : avant du Ghost à `ay = -0,464`, canons
  jumelés à `ax ≈ ±0,15` ; Banshee `-0,477` ; Chopper `-0,475` (canons à `±0,10`) ; le canon du
  Scorpion atteint `ay = -0,49`.

### Point 18 — les tourelles automatiques sont bleues et nommées

- `vehiclesPaint.drawVehiclesLayer` : un élément de carte (`kind = map_element`, famille
  `tourelle_auto_bannie`, Snowbound `bfecd02b`) prend `vehicleColorAt(...) ?? neutralInk` ;
  `neutralInk` = `divergent-neutral`, BLEU dans la palette par défaut (le bleu de l'allié). Et son
  libellé (« Tourelle automatique bannie ») s'écrit sous le pictogramme dès que le calque des noms
  est allumé.

## 2. Décisions tranchées

- D1 (18) : un élément de carte NON JOUABLE (`kind = map_element`) ne porte plus de libellé et se
  peint au gris « aucun camp » `zone-neutral` (`zoneInk.fill`, achromatique dans les 5 palettes,
  déjà l'encre des zones et socles sans camp). La tourelle FIXE (`fixed_turret`, jouable, occupée)
  n'est pas touchée (hors demande).
- D2 (20, serveur) : quand le catalogue ne donne pas de socle d'ÉQUIPE à un drapeau, le rejeu lit sa
  base et son camp DANS LE FILM : base = point des vols de l'équipe adverse (affiné par la renaissance
  de l'objet au même point quand elle existe), camp = l'équipe qui n'y vole pas. Le catalogue reste
  la source quand il nomme le camp ; le film le contrôle (accord / contradiction comptés).
- D3 (20, web) : le cercle se lit comme une ZONE AU SOL : disque plus franc, anneau net à l'encre du
  camp propriétaire, épaissi quand un défenseur est dedans ; la jauge devient un arc DISTINCT posé
  à l'extérieur de l'anneau. Le rayon reste celui du jeu (1,3), centré sur le pied du drapeau.
- D4 (19) : chaque arme de véhicule a un montage. Arme fixe : à l'AVANT du sprite (bouche mesurée
  sur le sprite). Tourelle : pivot + LONGUEUR DE CANON (`reach`, fraction de la longueur du sprite)
  dans la direction de la visée du tireur ; sans visée lue, le pivot (bouffée ronde, comme avant).
- D5 (19) : le « ! » d'un tireur EMBARQUÉ ne se dessine pas — même porte que son pion
  (`embarkedAtSlot`) : pas de pion, pas de « ! ».

## 3. Lots

### L1 — Base et camp de chaque drapeau lus dans le film (Go, `film/replay`) — exécutant opus-high

- [x] 1.1 Mesure (`film/replay/drapeau_bases_film_research_test.go`, seuils écrits avant : rayon 3 m, part du groupe principal ≥ 0,75, les deux camps volent, séparation ≥ 10 m, renaissance = ≥ 2 naissances à ≤ 0,10 m). 8 films hors catalogue → deux bases (dispersion des vols 0,37-1,36 m, séparation 24-45 m, renaissance à 0,05-0,75 m du centre des vols) ; 6 témoins catalogue → deux bases, centre des vols à 0,05-0,75 m du socle adverse, renaissance à ≤ 0,008 m du socle ; 4 neutres → « neutre » (centres à 0,27-0,76 m) ; `73c1df0b`, `be758198` : aucun vol, aucune base.
- [x] 1.2 `flag_film_bases.go` (lecture + résolution des socles, catalogue d abord, film en contrôle ou en source quand le catalogue ne nomme pas le camp) ; couverture `filmBases`, `spawnsFromFilm`, `filmBaseAgree`, `filmBaseContradict` ; replis `repli_index_drapeau_zero_pour_tous` / `repli_nombre_drapeaux_hors_catalogue_sans_passage` resserrés à « ni catalogue ni film » ; repli nommé neuf `repli_socle_du_film_au_centre_des_vols` (base sans renaissance à portée) ; docs inversées corrigées (`replaybuild/flagspawns.go`, `flag_*.go`, `placement_des_vies.go`…).
- [x] 1.3 `SchemaVersion` 78 → 79 (entrée v79 de la chronique), goldens d assemblage et de forme, 8 fixtures de contrat 79, `openapi.yaml`, `generated.ts`, table de polarité ; aucune révision de couche de décodage montée → verdict `republier` depuis des faits à jour.
- [x] 1.4 Gate (codes de sortie 0) : tests `film/...`, `replaybuild` + `service` + `domain`, `contracttest`, `archlint`, `replaydiff`, `killcollector` + `api`, `go vet ./...`, fraîcheur des types générés. Rejeu depuis les faits (`replaybuild/drapeau_bases_film_parc_research_test.go`, par `BuildFromFacts` car les faits du cache sont en grammaire `2026-09-27.3`, périmés pour la branche) : les 8 films hors catalogue passent de 1 drapeau d équipe -1 à 2 drapeaux d équipe, 2 à 9 `home` chacun, `carrierTeamUnknown` 0 ; témoins à deux socles et neutres identiques à l octet ; `a32ee8d2`, `61614156`, `81cc9952` perdent leur drapeau -1 immobile (`carrierTeamUnknown` → 0).

### L2 — Cercle lisible, tourelles grises sans libellé (web) — exécutant sonnet-medium

- [x] 2.1 `flagReturnZone.ts` : rendu D3 (+ tests). Disque 0,2 (0,3 occupé), anneau 0,9 / 2 px (3 px occupé), jauge sur son propre rayon `gaugeRadiusPx = max(zone + 5, 20)`, plancher de zone 8 px ; 7 tests de rendu (dont « jauge et anneau jamais au même rayon »).
- [x] 2.2 D1 : règle `vehiclePaintColor` / `vehicleIsMapElement` dans `model/vehiclesLayer.ts`, champ `VehicleStyle.mapElementInk` (= `zoneInk.fill`, posé sur une ligne existante de `ReplayCanvas.tsx`), plus aucun libellé pour un `map_element`. `labelOfFamily` RESTE (la tourelle fixe vide le lit). Tests : `vehiclesPaint.mapElement.test.ts`, `model/vehiclePaintColor.test.ts` (sortis pour tenir sous 500 lignes).
- [x] 2.3 Gate : typecheck (cache purgé) exit 0 ; lint exit 0 (26 avertissements préexistants) ; vitest `match-replay` exit 0, 3 225 tests.

### L3 — Les tirs partent de l'avant du véhicule (TOML + Go + web) — exécutant opus-medium

- [x] 3.1 `vehicle_weapons.toml` : les 15 armes ont un montage. Tir vers l avant à la bouche mesurée (Ghost `ay -0,464`, Banshee canons et bombe `-0,477`, Chopper `-0,475`, `ax 0` entre les canons jumelés) ; tourelles = pivot + `reach` (canon du Scorpion 0,44, Rockethog et LAAG 0,12 au pivot `0,26`, Falcon 0,04, mitrailleuse du Scorpion `(0,08 ; -0,07)` 0,02, tourelle du Wraith `ay -0,34` 0,10). À confirmer à l écran : mitrailleuse du Scorpion, tourelle du Wraith, `reach` du Falcon (aucun canon dessiné).
- [x] 3.2 `reach` : chargeur (0..1, tourelle seulement) + garde-rail « toute arme a un montage, toute tourelle un reach > 0 », types publiés, conversion, service, `openapi.yaml`, `generated.ts`, entrée v79 complétée (table posée à la requête : aucun artefact périmé).
- [x] 3.3 `tourelleBarrelOffset` (`vehicleWeaponMounts.ts`) : bouche = pivot tourné par le cap du châssis + `reach × longueur du sprite` le long de la visée ; sans visée, le pivot. `fireMark.ts` : `embarkedAtSlot`, câblé sur la ligne existante de `ReplayCanvas.tsx`. Tests ajoutés (bouche devant / de côté / sans visée, « ! » embarqué).
- [x] 3.4 Gate : tests Go `mappings` + `service` + `domain`, `film/replay`, `contracttest` + `archlint`, `go vet`, fraîcheur des types : 0 ; typecheck, lint, vitest `match-replay` (3 233 tests) : 0.

### L4 — Clôture (superviseur)

- [x] 4.1 `delivery-checklist` ; `go test ./...` (194 paquets ok ; `internal/watcher` `TestRESTPoller_BackoffOnRateLimit` instable sous charge, hors diff, vert 4 fois isolé) ; `go test -tags=integration -p 1` sur `sync`, `persist`, `replaybuild` : 0 (13 paquets) ; `go vet` 0 ; `golangci-lint --new-from-merge-base=origin/main` : 0 issue ; typecheck 0 ; lint 0 ; vitest complet 8 846 ok (2 délais de garde-rails sous charge, verts isolés) puis `match-replay` 3 236 ok après les corrections.
- [x] 4.2 Thought log et plan à jour ; commit `28c542b33` (accord de l utilisateur, après la revue adversariale qu il a demandée) ; `feat/v75` avancé en avance rapide (`87cdfa761..28c542b33`, contrôles d envoi verts) ; checkout principal mis à jour ; CI `37351843479` verte au niveau job (10 jobs, E2E sauté) et Deploy Pre-Check vert.
- [x] 4.3 Vérifications à l écran remises à l utilisateur (après recuisson des matchs témoins : schéma 79, et grammaire de la vague 2 déjà périmée pour le cache) : cercle de retour sur une partie classée CTF 3 Captures hors catalogue (`f1db4a07`, `92c950ee`, `fd247c3f`) et sur Origin (`d6918972`) ; tourelles grises sans libellé sur Snowbound (`bfecd02b`) ; tirs du Ghost (`81c02726`), Chopper / Scorpion / Wasp (`8a485699`), Warthog / Falcon / Wraith (`4f77afc1`).

## 4. Journal

- 2026-10-05 — Diagnostic et plan (superviseur). L1 et L2 lancés en parallèle (Go / web, fichiers
  disjoints) ; L3 après L1 (deux lots Go ne compilent jamais en même temps).
- 2026-10-05 — Corrections de la ronde 1 (opus-high) : socle concordant préféré, séparation 6 m (= 2 × rayon), socle sans camp promu neutre, 5 branches testées, `PlacementRev` → `placement-2026-10-05-v1`, phrase fausse du TOML corrigée ; 8 mutations rouges ; gates verts.
- 2026-10-05 — Revue adversariale COMPLÈTE demandée par l utilisateur avant fusion : 3 relecteurs aveugles en parallèle (serveur L3 + contrat ; web + armes L5/L2 + géométrie ; tests L6), sans commande go. 0 P0, 0 P1 ; 10 P2 : commentaire faux de `FlagSpawn.Neutral`, instrument resté à 10 m, 4e copie de `b2i` (règle 6), commentaire faux de `MIN_RADIUS_PX`, câblage du canevas non testé (porte du « ! », encre grise), naissance isolée, repli neutre sans catalogue, recomptes de naissances. Tous corrigés (opus-medium) : garde-rail de source `ui/vehicleWiring.guard.test.ts`, `b2i` supprimé (if explicites), instrument branché sur les constantes de production, 3 tests Go ; 9 mutations rouges puis restaurées. P0+P1 à 0 : pas de ronde 2.
- 2026-10-05 — L3 rendu et relu (géométrie de la bouche). Clos.
- 2026-10-05 — Revue adversariale de L1 (contexte frais, sans commande go) : 0 P0, 0 P1, 4 P2 (socles jumeaux d équipes opposées, seuil de 10 m qui exclut Isolation et commentaire faux, base neutre lue sur un socle -1 du catalogue, `PlacementRev` non monté) + 5 branches non testées. Tous traités dans un lot de corrections (exécutant opus-high), avec la phrase fausse de l en-tête de `vehicle_weapons.toml`.
- 2026-10-05 — Gate de clôture partiel (superviseur) : `go test ./...` 194 paquets ok, 1 échec `internal/watcher` `TestRESTPoller_BackoffOnRateLimit` (minutage sous charge, hors diff, vert 4 fois isolé) ; `go vet` 0 ; typecheck 0 ; lint 0 ; vitest complet 8 846 ok, 2 délais dépassés sur des garde-rails de balayage sous charge, verts isolés.
- 2026-10-05 — L1 rendu par l exécutant, diff relu (logique `flag_film_bases.go`, plafonds de taille relevés avec exception datée comme au schéma 78). Clos. L3 lancé.
- 2026-10-05 — L2 rendu par l exécutant, diff relu par le superviseur (règle d encre dans `model/`, 0 ligne ajoutée au canvas, gates rejoués par l exécutant avec codes de sortie). Clos.

## 5. Découvertes (notées, non traitées)

- Placement des vies au sync : traité dans le lot de corrections (`PlacementRev` monté — la revue a montré que des cartes AU catalogue changent aussi).
- `084a804d`, `390b1de5` (6 socles au catalogue) : aucune base lue dans le film, comportement inchangé ; cause non vérifiée.
- `73c1df0b` : 419 prises pour 2 vols au signal — verdict CTF peut-être faux positif.
- Faits du cache en grammaire `2026-09-27.3` alors que `feat/v75` porte `grammar-2026-10-03.5` (campagne de grammaire, vague 2, déjà fusionnée) : le parc local sortira `redecoder` pour cette raison, indépendamment de ce plan.

- `7206e05b` (Origin - Ranked, CTF) : artefact de 53 Ko sans aucun objectif ni drapeau
  (`flagFilm: false`, `objectives` vide) — film non lu à l'époque de la cuisson ; à revoir hors de
  ce plan.
