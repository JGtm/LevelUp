# PLAN — Réapparitions prévisibles : Tactique puis rejeu 2D (2026-11-15)

Statut : **en exécution depuis le 2026-11-15** (décisions du user du 2026-11-15 : « 1) on garde v8.2.0 ;
2) à 6) ok ; 7) oui on verra »).
Contrat d'exécution : skill `plan-execution` (ordre strict, une étape close et vérifiée avant la suivante,
aucune case vide à la clôture, zéro fix hors périmètre, découvertes en section 8). Grille relue : skill
`plan-review`.

Constats relus sur pièces le 2026-11-15 sur `d94048a830` (dossier principal). L'exécutant les RE-VÉRIFIE
sur la base retenue en D-1 avant de coder : la 8.2.0 a pu déplacer ces fichiers.

---

## 0. Objectif et critère de succès

Idée retenue par le user le 2026-11-15 (« j'aime bien l'idée, pour le point 3 c'est très juste, il faudra
vraiment travailler le rendu UI ») : le jeu choisit le point de réapparition selon la position des joueurs,
et ce choix se prévoit.

**Mesure de départ** (annexe `annexes_reapparitions/mesure_assassin_4c4_2026-11-15.txt`, outil jetable copié
à côté) : Assassin 4c4 hors Fiesta et BTB, 45 cartes, 43 966 réapparitions, validation croisée par match.

| À moins de 8 m du point réel | 1er choix | 3 premiers |
|---|---|---|
| Hasard | 14,1 % | — |
| Fréquence seule | 15,6 % | 38,4 % |
| Modèle (6 variables, logit conditionnel) | **44,9 %** | **62,5 %** |

**Corrigé le 2026-11-15 (revue du lot A)** : l'outil jetable comptait comme candidats les points vus dans le match
testé, départageait les égalités en faveur du point réel et publiait pour « hasard » la chance d'UN seul tirage.
Chiffres corrigés, même jeu (annexe `mesures_lot_A.md`) : hasard à trois tirages **36,2 %** (pas 14,1 %), fréquence seule
38,3 % aux trois premiers, modèle **44,9 %** au premier choix et **62,8 %** aux trois premiers. Le modèle est inchangé à
0,3 point près ; c'est le hasard de référence qui était faux (le gain du modèle sur le hasard à trois tirages est de
26 points, pas de 48). Au premier choix, le hasard de 14,1 % tient.

Poids : distance à l'ennemi le plus proche +1,98 (dominant), ennemis à moins de 20 m -0,65, distance au
coéquipier le plus proche -1,08 (la proximité favorise), distance au barycentre ennemi -0,93, fréquence
+0,45, lieu de la mort +0,15. Pas de ligne de vue.

| Lot | Ce que le user verra | Critère de succès |
|---|---|---|
| A | Rien à l'écran | Le modèle vit dans le dépôt (paquet pur + CLI testée), la CLI reproduit la mesure de départ à 1 point près, le fichier de poids est versionné, les mesures M-1 à M-4 sont écrites en annexe |
| B | Tactique : grappes de TOUTES les réapparitions du joueur, et la sortie de spawn d'une vie rattachée à la grappe de CETTE vie | Filtre de grappe sur « toutes les réapparitions » ; une route n'est comptée que si la vie part de la grappe choisie ; tests verts |
| C | Tactique : où l'adversaire réapparaît selon la zone tenue par l'équipe | Lecture servie avec la référence sans condition, planchers appliqués, état vide conforme |
| D | Rejeu 2D : à chaque mort, la probabilité de réapparition par zone, puis le point réel | Maquette validée par le user AVANT tout code ; calque conforme à la maquette ; règle Seul / Masquer respectée |

## 1. Décisions à trancher AVANT exécution

- **D-1 — base et version.** **Tranché (user, 2026-11-15, « on garde v8.2.0 ») : `release/v8.2.0`.** Règles
  de la release (pilote levelup-dd) : pas de PR, chaque lot se pousse sur `release/v8.2.0` après ses portes,
  rebasé juste avant l'envoi, SHA transmis à levelup-dd. Un worktree et une sous-branche par agent
  (`feat/reapp-a` -> `LevelUp-wt-reapp-a`, etc.).
- **D-2 — modes couverts à la première livraison.** Proposition : un mode est couvert si M-3 (lot A) y
  donne au moins 50 % dans les 3 premières zones, sinon il ne l'est pas. Dans un mode non couvert, le calque
  du rejeu et la lecture C n'apparaissent pas : afficher une action impossible est exclu (règle des états
  vides du 2026-10-17). Hors Arène (BTB), l'exclusion est d'emblée : 4c4 seulement.
  **Tranché (user, 2026-11-15) : ok sur la proposition.**
- **D-3 — échelle de la prévision.** Proposition : le texte parle en ZONES nommées (les callouts déjà
  servis à Tactique et au rejeu) ; le dessin montre les points. Une zone reçoit la somme des probabilités
  de ses points.
  **Tranché (user, 2026-11-15) : ok sur la proposition.**
- **D-4 — ligne de vue.** Proposition : elle n'entre au modèle que si M-2 (lot A) montre un gain d'au moins
  3 points sur « 3 premières zones », toutes cartes confondues. Sinon elle est écartée et on n'y revient pas
  dans ce plan.
  **Tranché (user, 2026-11-15) : ok sur la proposition.**
- **D-5 — « zone tenue par l'équipe » (lot C).** Proposition : à l'instant où l'adversaire réapparaît, au
  moins deux joueurs vivants de l'équipe du joueur actif sont dans la zone choisie. Alternative : le
  barycentre de l'équipe dans la zone.
  **Tranché (user, 2026-11-15) : ok sur la proposition.**
- **D-6 — maquettes (lots C et D).** Proposition : une seule page de maquettes (artefact claude.ai à onglets,
  comme les maquettes du rejeu du 2026-11-09), avec deux ou trois variantes par écran. Le user choisit,
  l'implémentation suit la variante choisie, et rien n'est codé côté web pour C et D avant ce choix.
  **Tranché (user, 2026-11-15) : ok sur la proposition.**
- **D-7 — prod.** Le lot A change le format des fichiers tactiques par match ; ils se régénèrent depuis
  les rejeux déjà sur disque (`levelup tactical-rasters --backfill`, aucune relecture de film, aucune base).
  Proposition : en local, exécuté par l'agent ; sur le serveur de prod, commande donnée au user, qui décide
  du moment.
  **Tranché (user, 2026-11-15, « oui on verra ») : ok, le moment se décide plus tard.**

## 2. Ce qui existe (relu sur pièces)

- Grappes de spawn : `internal/analysis/tactical/spawn.go` (`GrappesDeSpawn`, grille 0,5 m en 2D, plancher
  3 matchs distincts PAR GRAPPE, identifiant = barycentre + nombre de cellules). Seul le spawn de départ y
  entre : `service/tactical_service_lectures.go` `spawnsDeDepart`, helper unique tenu par
  `archlint/no_local_spawn_depart_test.go`.
- Filtre `spawn` : `api/handlers/tactical.go` (`scopeAvecSpawn`) ; il restreint l'UNIVERS aux matchs dont la
  première vie part de la grappe (`perimetreDuSpawn`), pour toutes les lectures.
- Lecture « routes » : `domain/tactical_raster.go` `TacticalQuestionRoutes` ; 15 premières secondes de
  CHAQUE vie (`analysis/tactical/vies.go`, `Route.DebutFrame`).
- Fichier tactique par match (sidecar) : `domain/tactical_raster.go`, `TacticalRasterSchemaVersion = 6` ;
  par joueur nommé : cellules, `Spawns` (frame, x, y, premiere_vie), premières entrées, routes. NI z, NI
  équipe, NI position des autres joueurs. Construit à la cuisson (`sync/replayartifacts/raster.go`) depuis
  `analysis/tactical/tracks.go` : `Spawns` = premier point de CHAQUE piste.
- PIÈGE : une piste n'est pas toujours une vie. Un trou de réplication coupe une vie en deux pistes (registre
  des reports, ligne « vie coupée par un trou de réplication »). La mesure de départ n'a gardé que les
  pistes séparées de la précédente par 4 à 15 s.
- Artefact de rejeu : `tracks[]` (`xuid`, `team`, `endFrame`, `points[]` de `t`, x, y, z) ; `structure[]`
  (emprises au sol `x0,y0,x1,y1`, faces haute `z` et basse `zb`), matière possible de la ligne de vue.
- Rattrapage : `cmd/levelup/cmd_tactical_rasters.go` (`--backfill`, `--dry-run`), lit les artefacts
  seuls.
- Routes HTTP : `/tactical/{map_id}/raster|cellule|sequence|background`, `/matches/{match_id}/replay`,
  `/replay/background`, `/replay/callouts`.
- Capacité : `CapFilmReplayArtifact` garde déjà les lectures d'artefact.

## 3. Lot A — le modèle dans le dépôt (Go, effort M)

Périmètre fermé :

- [x] A.1 Sidecar v7 (voie de repli cablee dans la 8.2.0 ; voies `film` et `fin de manche` cablees le 2026-11-15 sur la montee du decodeur v1.3.9, branche `feat/reapp-film-v139`, livraison 8.2.1 (le pilote de la 8.2.0 relaie le 15/11 que la 8.2.0 sort sans le nouveau decodeur)) : `TacticalRasterSchemaVersion` 6 -> 7 ; par joueur, `Equipe` ; par spawn, `Z`,
      `Reapparition bool`, sa `Source` (`film` ou `repli_ecart`), `DebutDeManche`, la `Mort` qui
      précède et l'`Instantane` (xuid, équipe, x, y, z de chaque joueur vivant à cette frame, apparus dans
      la même seconde exclus) ; compte par source dans la couverture du sidecar (`reapparitions`).
      Construction dans `analysis/tactical` (pur : `reapparitions.go`), tests unitaires.
      **Règle (grammaire du film d'abord)** : (1) une piste est close par une MORT si `endReason ==
      "released"` avec `releaseFrame` dans [-2, +6] frames de la fin de piste, OU si un compte à rebours du
      même joueur COMMENCE dans [-2, +6] frames de cette fin ; la piste suivante est alors une
      réapparition (source `film`). (2) Une piste close par `deleted` est une FIN DE MANCHE : la suivante
      est un spawn de début de manche, ni une réapparition ni une entrée du modèle. (3) Sans aucune de ces
      lectures dans le document : REPLI sur l'écart de 4 à 15 s entre deux pistes, nommé et compté
      (`repli_ecart`) ; dans un document qui porte au moins une lecture, une piste sans preuve n'est pas
      une réapparition (jamais de repli). Tolérances et bornes en constantes nommées, en millisecondes converties avec le pas du document (le garde-rail `no_seuil_en_images_go_test` interdit un seuil en images) : 66 ms avant et 198 ms après, soit les 2 et 6 images de la preuve du rejeu web à 33 ms. Entrée pure : par
      piste, fin lue facultative (raison + frame) ; par joueur, la liste des débuts de compte ;
      `JoindreDebutsDeCompte` (index film -> xuid par `identity.players[].filmIndex`, deuxième copie de la
      jointure du rejeu web).
      **Dépendance explicite** : le câblage depuis le document (`endReason`, `releaseFrame`,
      `respawnCountdowns[]`, `identity.players[]`) arrive avec la montée du décodeur v1.3.9 (schéma 123 ;
      `respawnCountdowns[]` dès le schéma 120) sur `release/v8.2.0`, hors de ce lot. Les champs n'existent
      pas sur la base de ce lot (décodeur v1.3.7) : `EntreeOccupationDeLArtefact` ne câble que la voie de
      repli, et les trois autres voies sont prêtes et testées côté pur.
      Fait (2026-11-15, 8.2.1) : `EntreeOccupationDeLArtefact` traduit `endReason` / `releaseFrame` en `Piste.Fin`
      (`finDeLaPiste` ; `deleted` sans frame publiee prend la fin de la piste) et `respawnCountdowns[].from` en
      `DebutsDeCompte` par `JoindreDebutsDeCompte` sur `identity.players[]` ; les comptes sans xuid sont comptes
      (`EntreeOccupation.ComptesSansJoueur`). Tests `raster_reapparitions_film_test.go`. Cinq films cuits a la v1.3.9 :
      lacher et debut de compte a 0 a 99 ms de la fin de piste (tolerance 198 ms tenue), annexe
      `annexes_reapparitions/mesures_lecture_film.md`.
- [x] A.2 `analysis/tactical/reapparition.go` (pur, sans I/O) : points candidats d'une carte (points
      observés dans au moins 2 AUTRES matchs que celui de l'événement évalué, clé à 0,5 m et 1 m de hauteur), les six variables de la mesure,
      score, probabilités (softmax), ajustement (montée de gradient), évaluation (point exact, rayon 8 m,
      zone nommée ; 1er choix et 3 premiers). Tests : cas construits à la main, déterminisme, plancher.
      Revue du 2026-11-15 : candidats hors du match évalué (`Candidat.CandidatPour`), hasard exact du top 3 au rayon
      (`partTouchee`), égalités du rayon partagées sans favoriser le réel (`toucheRayon`), tests d'exclusion du pli, de signe
      du gradient et de repli d'équipe.
      Fait : `reapparition.go` (candidats, sept variables, softmax), `reapparition_ajustement.go` (echantillons, montee de gradient, validation croisee a 5 plis par match, plis ajustes en parallele), `reapparition_evaluation.go` (point exact, rayon de 8 m, zone nommee), `ligne_de_vue.go` (septieme variable, mesure M-2) ; tests construits a la main, determinisme.
- [x] A.3 CLI `levelup spawn-model` : `evaluate` (imprime le tableau de la mesure, par mode et par
      carte, validation croisée par match) et `fit` (écrit le fichier de poids). Lecture des sidecars un
      match à la fois (pas de chargement du parc en mémoire). Logs `slog` structurés.
      Fait : `cmd_spawn_model*.go`. La commande lit les artefacts un a un et le registre par `OpenReadForQuery` (une base tenue en ecriture ailleurs fait echouer l'ouverture avec un message qui le dit). Sections `m1` a `m4`, `poids`, `--famille`, `--matchs-min-carte`, `--par-carte`, `--variantes`. Gate de capacite : `film.spawn_forecast`.
      Revue du 2026-11-15 : les familles de modes viennent de `config/titles/halo_infinite/mappings/spawn_families.toml`
      (`mappings.SpawnFamilySet`, plus aucun nom de mode dans la CLI) ; `fit` refuse d'ecraser le fichier de reference
      avec `--famille` ou `--limit` (`--sortie` ecrit ailleurs) ; seuil de cartes par defaut 8 pour `evaluate`, 3 pour `fit`
      (commande exacte dans `docs/COMMANDS.md`) ; `--variantes` sort sans erreur pour `fit` ; effectifs comptes sur les
      reapparitions retenues.
- [x] A.4 Fichier de poids versionné : chemin par `PathResolver` (nouvelle méthode, sous
      `data/titles/{slug}/reference/`), contenu = révision, date, poids par famille de modes, effectifs et
      scores de l'évaluation. Lecture par le service, avec repli journalisé et compté si le fichier manque.
      Fait : `PathResolver.SpawnModelPath`, `domain.PoidsReapparition`, `service.ChargerPoidsReapparition` (repli journalise et compte `spawn_model_poids_repli_total`), ecrit par `spawn-model fit` (poids par famille de modes plus la famille `tous`, effectifs et scores de validation croisee), relu par `spawn-model evaluate --sections poids`. Fichier : `data/titles/halo_infinite/reference/spawn_model.json`.
- [x] A.5 Rattrapage `levelup tactical-rasters --backfill` (sidecars v7) prouvé sur une COPIE d'une vingtaine
      d'artefacts dans un dossier temporaire, compte avant/après. JAMAIS sur `data/` du dossier principal :
      l'app de vérification de la 8.2.0 (port 8000, binaire antérieur) y lit les sidecars v6 et perdrait ses
      lectures tactiques. Le rattrapage complet se fait au déploiement de la 8.2.0 (geste ajouté à la liste
      des gestes de prod). La CLI `spawn-model` lit donc les ARTEFACTS (même fonction pure que le sidecar),
      pas les sidecars.
      Fait sur une copie de 20 artefacts : 20 sidecars v6 -> 20 v7, seconde passe 0 ecriture, duree par artefact 151 ms (v6) contre 167 ms (v7), +7 % d'octets ; detail et commande de controle (`grep -l '"schema_version":7' ... | wc -l`) dans l'annexe. Rien n'a ete ecrit dans le dossier de donnees principal.
- [x] A.6 Mesures en annexe `annexes_reapparitions/mesures_lot_A.md` :
      M-1 témoin : la CLI reproduit la mesure de départ à 1 point près (sinon expliquer l'écart AVANT de
      continuer) ; M-2 ligne de vue approchée par `structure[]`, gain mesuré (alimente D-4) ;
      M-3 par mode (Assassin, Fiesta, Drapeau, Bases, Roi de la colline, Oddball ; alimente D-2) ;
      M-4 prévision à l'instant de la MORT contre l'instant de la réapparition (le rejeu affiche la
      prévision pendant le compte à rebours : il faut savoir ce qu'elle vaut 5 s avant).
      Fait : M-1 a 0,1 point du temoin ; M-2 (+0,2 a +0,6 point, mesurable sur 3 cartes seulement) ; M-3 ; M-4 (la prevision perd 9 a 30 points a l'instant de la mort). Voie de decision : le repli (parc au schema 116). Rejouees le 2026-11-15 avec la methode corrigee (annexe, tableaux AVANT / APRES) : D-2 et D-4 tiennent ; a l'instant de la MORT, Bases (49,7 %) et Roi de la colline (48,3 %) passent sous 50 % en zone nommee. Conclusions de la regle : D-2 couvre Assassin, Fiesta, Drapeau, Bases et Roi de la colline (Oddball non mesurable) ; D-4 ecarte la ligne de vue. Le user tranche.
- [x] A.7 Capacité fine `film.spawn_forecast` dans `config/titles/*/mappings/capabilities.toml`
      (vraie pour `halo_infinite`, fausse pour `halo_5`) ; `ErrCapabilityNotSupported` sinon.
      Fait : constante `games.CapFilmSpawnForecast`, `AllCapabilityKeys`, `supported` pour `halo_infinite` (TOML et adaptateur), absente pour `halo_5`, `not_exposed` pour le titre synthetique de test ; la CLI `spawn-model` s'arrete proprement sur un titre qui ne la declare pas. Le 503 `ErrCapabilityNotSupported` des lectures servies arrive avec la route du lot D (aucun service ne la consomme avant).
      Revue du 2026-11-15 : la production du sidecar consulte aussi la capacite (`capabilityPrevisionArmee`) : sans
      `film.spawn_forecast`, le sidecar ne porte ni reapparition, ni source, ni debut de manche, ni mort, ni instantane, ni
      couverture ; le sidecar de `halo_infinite` est identique octet pour octet (20 rejeux, avant / apres).
- [ ] A.8 Mesure d'accord entre les deux definitions d'une reapparition (lecture du film contre repli
      sur l'ecart de 4 a 15 s), sur le parc regenere a la v1.3.9 : meme `spawn-model evaluate`, compteurs
      de source `film` et `repli_ecart`, part des reapparitions sur lesquelles les deux definitions
      s'accordent, et M-1 a M-4 rejouees sur la voie du film. **Dependance** : montee du decodeur v1.3.9
      (schema 123, `respawnCountdowns[]` des le schema 120, `endReason` / `releaseFrame` par piste) sur
      `release/v8.2.0`, cablage de `EntreeOccupationDeLArtefact` sur ces champs, regeneration du parc.
      Non coche tant que ces trois conditions ne sont pas reunies ; les mesures M-1 a M-4 du lot A
      tournent sur le repli (annexe).
      Avance (2026-11-15) : la mesure d'accord est dans `spawn-model evaluate` (bilan de lecture : documents lus au film,
      accord, film seul, repli seul, dont debut de manche, comptes sans joueur ; `cmd_spawn_model_accord.go`, teste).
      Premiere lecture sur cinq films cuits a la v1.3.9 (annexe `mesures_lecture_film.md`) : 645 reapparitions au film,
      le repli en retrouve 641 et en ajoute 7, toutes des debuts de manche. Reste : la regeneration du parc (cuisson en
      lot : accord du user requis avant de la lancer), puis la mesure du parc et M-1 a M-4 rejouees sur la voie du film.

Gate A : `cd apps/go-api && go test ./internal/analysis/tactical/... ./internal/domain/... ./cmd/levelup/...
./internal/archlint/...` vert ; `make go-api-lint` sans hausse de la baseline ; annexe M-1 à M-4 écrite ;
décisions D-2 et D-4 confirmées par le user sur ces chiffres.

## 4. Lot B — Tactique : toutes les réapparitions du joueur (Go + web, effort S-M)

- [x] B.1 Helper unique `reapparitionsDuJoueur` à côté de `spawnsDeDepart` (même garde-rail étendu).
      Fait : `service/tactical_service_lectures.go`. Une seule boucle de lecture (`spawnsDuJoueur`), deux
      prédicats (`spawnsDeDepart` : `PremiereVie` ; `reapparitionsDuJoueur` : `PremiereVie || Reapparition ||
      DebutDeManche`) et le choix `spawnsSelonVies`. Le garde-rail `archlint/no_local_spawn_depart_test.go`
      tient les trois drapeaux (self-check positif sur chacun). **Décision écrite : « toutes les vies » =
      toutes les apparitions du joueur** : première vie, réapparitions après mort, débuts de manche (un début
      de manche n'est pas une réapparition mais c'est une apparition). Une piste sans drapeau (vie coupée par
      un trou de réplication, repli hors des 4 à 15 s) n'est pas une apparition et n'entre dans aucune grappe.
      `tactical.PointSpawn` gagne `Frame` (clé de jointure B.3 ; les grappes l'ignorent).
- [x] B.2 Grappes : paramètre `vies` (`depart` par défaut, `toutes`) sur les routes `raster` et
      `cellule` ; identifiant de grappe inchangé dans sa forme.
      Fait : `vies` est un CHAMP DU CORPS des deux POST (comme `spawn` : ces routes n'ont pas de query string),
      `domain.TacticalScope.Vies`, constantes et `NormaliserVies` dans `domain/tactical_vies.go`, valeur
      inconnue = `ErrTacticalViesInconnue` -> 400 `tactical_lives_unknown` (handler sans logique, refus du
      service). Les grappes de la lecture d'artefact suivent `vies` ; l'identifiant garde sa forme (barycentre
      + nombre de cellules) : un identifiant n'existe que sous sa valeur de `vies`, le web remet donc le filtre
      de grappe à « Toutes » quand la valeur change. Sidecar v6 : refusé à la lecture par
      `domain.SidecarRasterCourant` (match « non mesuré », WARN avec le remède `levelup tactical-rasters
      --backfill`), jamais relu ni recalculé à la volée : sur le parc local (v6) la page ne montre rien tant
      que le rattrapage v7 (geste G7) n'est pas passé.
- [x] B.3 Routes : sous un filtre de grappe avec `vies=toutes`, seules comptent les routes dont la vie part
      de la grappe (jointure `Route.DebutFrame` = `Spawn.Frame`, à vérifier sur pièces) ; avec `depart`,
      comportement actuel inchangé (témoin avant/après sur trois cartes).
      Jointure VÉRIFIÉE sur pièces : spawn (`NouvelIndexVies`, `Spawns()`) et route (`enrichirVies`) sont le
      premier point retenu de la MÊME piste par `pointsDansLaFenetre` ; test `analysis/tactical`
      `TestOccupation_LaRouteDUneVieSeJointAuSpawnParSonInstant` (points hors fenêtre et désordonnés). Sous
      `toutes` et un filtre de grappe, `perimetreDuSpawn` rend des COPIES des sidecars dont les routes du
      joueur sont réduites aux vies de la grappe (`sidecarsAuxVies`) ; les lectures SQL et « temps » gardent
      la règle d'univers (matchs ayant au moins une vie partie de la grappe). Le détail d'une cellule (`Cellule`)
      applique le même filtre de grappe, sous `toutes` comme sous `depart` (`tactical_service_vies.go`,
      `appliquerLeFiltreDeGrappe`).
      **Décision du user, 2026-11-15 (« ok avec ta reco »)** : le détail d'une cellule applique AUSSI le filtre `spawn`
      sous `vies=depart` ; il le copiait sans le lire (défaut d'avant le lot), la liste d'une zone cliquée ne suivait donc
      pas la grappe choisie. Sous `depart` le détail se réduit aux matchs dont la première vie part de la grappe (comme
      la cellule agrégée, qui compte toutes les routes de ces matchs) ; sous `toutes`, aux routes des vies parties de
      la grappe. Test : `TestVies_CelluleDepartSousFiltre` (6 contributions sans le correctif, 3 avec). Témoin avant/après : voir le journal
      du lot (annexe `annexes_reapparitions/temoin_lot_B.md`).
- [x] B.4 Web : bascule « Départ / Toutes les vies » dans la barre de la lecture des grappes ; query keys
      dans `lib/query/keys.ts` ; textes FR et EN ; `make generate-types`.
      Fait : pilule « Vies » (`TacticalPlanVies.tsx`, `vies.logic.ts`, `pilules.ts`), proposée dès qu'une lecture
      peut servir des grappes ; `vies` entre dans les paramètres de lecture (non posté sous « Départ » : corps et
      clé de cache d'avant inchangés). Query keys : aucune clé neuve, `tacticalRaster` / `tacticalCellule`
      portent déjà l'empreinte du corps (`hashFiltre`), où `vies` entre. Textes FR et EN dans
      `manifests/tactical.toml`. `make openapi-gen` puis `generate-types` : `vies` ajouté aux deux corps.

Gate B : tests Go du service et des handlers verts, témoin avant/après égal en mode `depart`,
`make check-types`, `make test-web`, contrôle à l'écran ajouté à l'artefact des vérifications.

Gate B passé le 2026-11-15 (lot B CLOS ; le contrôle à l'écran reste au user) :
- `go test ./internal/service/... ./internal/api/handlers/... ./internal/analysis/tactical/...` vert ;
  `go test ./internal/archlint/...` vert (2 661 s sous la charge de la machine, `-timeout 60m` : le délai par défaut de
  10 minutes ne suffit pas quand d'autres sessions compilent) ;
- `golangci-lint run --new-from-merge-base=origin/main` : 0 issue (une première passe avait relevé 2 `lll` sur la doc
  du champ `vies`, corrigés et regénérés ; `--allow-parallel-runners` quand une autre session lint) ;
- `tsc -b` (`make check-types`) propre, `eslint src/features/tactical` propre, `node tools/knip-ratchet.mjs` 0 / 0 / 0,
  `check-generated-types-fresh` OK ; vitest : une passe complète sous charge a rendu 44 fichiers en échec, tous par
  délai de 5 s des gardes qui balaient l'arbre (aucun du lot) ; les 44 relancés avec `--testTimeout=240000` : 44 / 44 et
  104 / 104 tests verts ; les tests du lot (`vies.logic.test.ts`, `TacticalAnalysisView.vies.test.tsx`) verts ;
- témoin avant/après en mode `depart` sur trois cartes : identique octet pour octet (annexe `temoin_lot_B.md`).

Ronde de revue du lot B (2026-11-15, cinq P2, un seul commit) : tous `[x]`.
- [x] P2-1 `vies=toutes` exige `film.spawn_forecast` : `viesAcceptees` (`tactical_service_vies.go`) refuse `toutes` avec
      `ErrCapabilityNotSupported` (503) pour la lecture et le détail quand la capability manque ; tests
      `TestVies_ToutesExigeSpawnForecast` (capabilities RÉELLES du titre synthétique : refus ; `depart` servi ;
      halo_infinite accepte) et `TestTacticalHandler_ToutesRefuseSurUnTitreSansSpawnForecast` (vrai service,
      capabilities du titre synthétique, dépôt qui panique s'il est lu : 503 sur les deux routes).
- [x] P2-2 isolation PAR MATCH de la jointure : `TestVies_LaJointureEstParMatch` (trois matchs aux instants de
      réapparition différents, chacun avec une piste sans drapeau dont l'instant est celui d'un AUTRE match) et
      `TestVies_DeuxViesDUnMemeMatchDansLaMemeGrappe` (les deux routes comptent, le match une fois). Mutation :
      `viesDesSpawns` rendant l'union des instants de tous les matchs fait rougir le premier (la route de la
      piste sans drapeau passe le filtre) ; code remis.
- [x] P2-3 la bascule « Vies » reste offerte dès que le titre déclare la capability, sous « Départ » et sans grappe
      servie, y compris sur la lecture « morts » ; elle n'est jamais offerte sans. Le bootstrap ne sert au web que les
      capabilities de TITRE : ajout de la capability de titre `spawn_forecast` (`title.CapSpawnForecast`, déclarée par
      halo_infinite, miroir TS `TITLE_CAPABILITIES`, lue par `useCapability`, jamais par le slug), pendant de la fine
      `film.spawn_forecast`, égalité tenue par `TestSpawnForecastCoarseFineMirror` (trois titres). Sans la
      capability, `vies` n'est jamais posté (`viesEffectives`). Tests : trois cas dans
      `TacticalAnalysisView.vies.test.tsx`.
- [x] P2-4 « Vies » ajouté au `it.each` « revient à la zone la plus chaude » de `TacticalAnalysisView.test.tsx` ;
      mutation : sans `:${vies}` dans la clé de `useZone`, le cas rougit.
- [x] P2-5 `ReglagesEnPilules` : 94 -> 79 lignes (`PilulesDeReapparition` extrait dans `TacticalPlanVies.tsx`).

## 5. Lot C — Tactique : réapparitions adverses selon la zone tenue (Go + web, effort M)

- [ ] C.1 Maquette (page unique de D-6, onglet « Tactique »), choix du user consigné ici avec sa citation.
- [ ] C.2 Lecture `reapparitions_adverses` : pour la zone choisie, répartition des réapparitions adverses
      par zone quand l'équipe tient la zone (règle D-5), à côté de la répartition sans condition.
      Planchers : 3 matchs distincts et 20 réapparitions, sinon état vide.
- [ ] C.3 Handler sans logique, service, domaine ; tests mock du port et `httptest`.
- [ ] C.4 Web selon la maquette choisie : tokens de couleur uniquement, titres factuels sans possessif,
      états vides (bloc + pointillé), FR et EN.

Gate C : tests verts, `make check-types`, `make test-web`, contrôle à l'écran par le user.

## 6. Lot D — rejeu 2D : prévision de réapparition (Go + web, effort M-L)

- [ ] D.1 Maquette (même page, onglet « Rejeu ») : variantes du calque pendant le compte à rebours, du point
      réel à l'apparition, de la légende et du réglage dans le bouton « Vue ». Choix du user consigné ici.
      **Gate : aucun code de D.3 avant ce choix.**
- [ ] D.2 Route `GET /matches/{match_id}/replay/reapparitions` : pour chaque mort suivie d'une
      réapparition, probabilités par zone échantillonnées chaque seconde du compte à rebours, et point réel.
      Les candidats excluent le match affiché (une prévision ne voit pas la réponse). Service par le port
      de rejeu existant, calcul dans `analysis/tactical`.
- [ ] D.3 Calque web : `model/respawnForecast.logic.ts` (pur, testé) + calque dans `layers/` ; il suit
      Seul / Masquer (la prévision appartient au joueur mort) ; contrat des calques en delta respecté.
- [ ] D.4 Vérification EN LECTURE (pas seulement à l'arrêt), sur trois rejeux choisis par le user.

Gate D : tests verts, garde-rails du rejeu verts, contrôle à l'écran du user en lecture.

## 7. Exécution

- Ordre : A, puis B, puis C et D (maquette commune avant tout code web de C et D). Un lot est clos quand
  toutes ses cases ont un statut (`[x]` fait, `[~]` couvert ailleurs avec référence, `[!]` non traité avec
  justification écrite) et que son gate est passé.
- Agents Sonnet, un worktree par agent, aucune jonction vers `data/` (incident du 2026-11-13). Les
  commandes qui lisent le parc se lancent depuis le dossier principal ou avec sa racine de données passée
  en option (à vérifier sur pièces au lot A).
- Témoins avant/après pour toute lecture existante touchée (B.3).
- Reprise de session : cocher les cases ici, une entrée au `thought_log.md` à chaque lot clos.
- Notes de version et changelog FR/EN dans le même envoi que chaque lot visible (B, C, D).

- Dépendances datées du 2026-11-15, acceptées par la revue du lot A (aucune n'est un report) :
  (a) `JoindreDebutsDeCompte`, `CompteBrut`, `Piste.Fin` et `DebutsDeCompte` n'ont aucun appelant de production
  sur la base de ce lot : ils sont câblés dans la 8.2.0 avec la montée du décodeur v1.3.9 (A.1, A.8) ;
  (b) `PoidsDeFamille` et `ChargerPoidsReapparition` n'ont aucun appelant serveur : ils sont branchés au lot C
  (service de lecture). Le choix de la famille d'un match (`spawn_families.toml`, `mappings.SpawnFamilySet`) et le
  repli sur « tous » (`tactical.PoidsPourFamille`) sont codés et testés ; la CLI `spawn-model` s'en sert.

- Lot A envoyé sur `release/v8.2.0` le 2026-11-15 AVANT la montée du décodeur v1.3.9, à la demande du
  user (« tu peux continuer ? ») : la voie de repli suffit à ouvrir le lot B ; le câblage de la lecture du
  film (A.1, A.8) se fait dans un envoi à part dès la montée poussée. Le rattrapage v7 de la prod ne se lance
  qu'après ce câblage et la régénération (geste G7).

- Décision du user transmise par le pilote de la release (2026-11-15) : la 8.2.0 porte les lots A et B ; les lots C
  et D passent dans la version suivante (leur branche release, recalés sur elle). Le câblage de la lecture du
  film (A.1, A.8) reste dans la 8.2.0, au SHA de la montée du décodeur v1.3.9.

## 7 bis. Orchestration des lots C et D (2026-11-16, pilote levelup-dd)

Go du user le 2026-11-16 (« Tu as mon go pour "Réapparitions, deux étapes suivantes", pas pour le reste »).
Base : `release/v8.3.0`. Agents sans commit ni push ; le pilote relit sur pièces, commite et pousse.

| Piste | Items | Modèle | Worktree / branche | Attend |
|---|---|---|---|---|
| P1 | A.8 (mesure d'accord sur le parc régénéré au schéma 124, M-1 à M-4 sur la voie du film) | Sonnet | `C:\t\reapp-a8`, détaché, lecture seule du parc | rien |
| P2 | C.1 + D.1 (page unique de maquettes, onglets « Tactique » et « Rejeu ») | Opus (rendu UI soigné demandé par le user) | aucun (fichier HTML hors dépôt) | rien |
| P3 | C.2 + C.3 (lecture `reapparitions_adverses`, service, handler, tests) | Sonnet | `C:\t\reapp-c`, `feat/reapp-c` | rien |
| P4 | D.2 (route des prévisions de réapparition, calcul dans `analysis/tactical`) | Sonnet | `C:\t\reapp-d`, `feat/reapp-d` | rien |
| P5 | C.4 + D.3 (web) | Sonnet | worktrees de P3 et P4 | choix du user sur les maquettes (D-6) |
| P6 | Revue finale du diff complet (une ronde) | Fable | lecture seule | P3 à P5 |
| P7 | D.4 et contrôles à l'écran | user | — | P5, régénération de la 8.3.0 pour la prod |

Bornes : quatre agents d'écriture à la fois, une seule tâche lourde en mémoire (P1 lit le parc un artefact à la
fois), `GOCACHE=%LOCALAPPDATA%\go-build-<piste>` par piste. P3 et P4 écrivent dans des fichiers distincts de
`analysis/tactical` et du service ; un fichier partagé (routes, OpenAPI, `generated.ts`) n'a qu'un écrivain à la
fois : P3 d'abord, P4 se recale au moment de l'intégration.

## 8. Découvertes (hors périmètre, consignées sans être traitées)

- Échelle du point exact (`rangsExacts`) : pour trois ex aequo ou plus, elle ne donne pas l'espérance
  exacte d'un départage au hasard (4 ex aequo : 0 % au lieu de 25 % au premier choix) ; exacte pour deux.
  Négligeable avec des variables continues ; à reprendre si le service classe des candidats ex aequo.
- Le fit écrit sous la racine de ressources du dépôt (`data/titles/.../reference`) : `LEVELUP_DATA_DIR`
  ne la détourne pas.
- Retirer la jointure Go `JoindreDebutsDeCompte` (index film -> xuid par `identity.players[].filmIndex`)
  quand le xuid sera toujours publie par le decodeur sur le compte a rebours (version apres la v1.3.9) ;
  c'est la deuxieme copie de cette jointure (la premiere est la preuve de fin de vie du rejeu web) : une
  troisieme impose de la centraliser.
- `structure[]` n'est portee que par 3 cartes sur 60 lues du parc (schema 116) : la ligne de vue ne se
  mesure que la, et un artefact sans structure a une septieme variable nulle.
- Les sidecars pesent environ 680 Ko par match en schema 6 (727 Ko en schema 7) : environ 1 Go pour les
  1 455 artefacts du parc.
- `mode_category` vaut `Other` pour certaines parties de Roi de la colline en Doubles ; le classement par
  famille de `spawn-model` s'appuie donc sur le nom de la variante d'abord.
- Oddball : 7 matchs sur le parc, non mesurable.
- La projection des zones du port vers le type pur de `tactical` existe en deux copies (service Tactique,
  `cmd_spawn_model_zones.go`) ; une troisieme impose de la centraliser.
- `golangci-lint run --new-from-merge-base=origin/main` demande plus de 20 minutes sur ce poste (4 Go de
  memoire) : le delai par defaut du Makefile (5 min) ne suffit pas ; a lancer avec un delai large.
- (Lot B) Les sidecars du parc local sont en schéma 6 : refusés à la lecture, la page Tactique n'y montre aucune
  lecture d'artefact ni grappe tant que le rattrapage v7 (geste G7) n'est pas passé.
- (Lot B) `domain/tactical.go` est à 500 lignes (limite) : toute nouvelle déclaration du périmètre Tactique va dans
  un fichier voisin (`tactical_vies.go`).
- (Lecture du film) `endReason: deleted` couvre la fin de manche, mais aussi la remise en place, le depart et le slot
  repris par un autre joueur (`coverage.tracks.lifeEnds.retaken`) ; la regle du lot A lit toute suppression suivie
  d'une vie comme un debut de manche. Sur les cinq films mesures, 3 suppressions en Grande bataille ne sont suivies
  d'aucune vie du meme joueur (sans effet) ; a verifier sur le parc regenere (A.8) avant d'afficher un debut de manche.
- (Lecture du film) Le decodeur ne publie pas la frame d'une suppression : la fin retenue est celle de la piste, ce qui
  rend la tolerance sans objet pour `deleted`.
