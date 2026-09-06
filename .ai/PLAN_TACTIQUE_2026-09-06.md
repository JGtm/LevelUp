# PLAN — Onglet « Tactique » + derives de coordination — V1

> Redige le 2026-09-05 depuis `feat/v75`. **Revision 4 (2026-09-06)** : execution demarree.
> Le chantier coexiste avec l'audit v2 du rejeu et du film (`.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`,
> lots A-G sur `feat/v2-*`). Croisement des diffs fait sur pieces : les phases 1-3 ne
> croisent AUCUN lot ; les phases 4-7 croisent C (capabilities), B (document servi) et D
> (modele web) et sont GELEES jusqu'a leur integration dans `feat/v75`. Deux concepts
> supprimes parce que le lot C les apporte sous un autre nom : `CapTactical` -> `CapReplay`
> (title-level) ; `film.replay_tracks` -> `film.replay_artifact` (data-level) ;
> `useDataCapability(cle fine)` cote web.
>
> Contrat d'execution : skill `plan-execution` (ordre strict, aucun report, statut par item,
> verification sur pieces, zero fix hors perimetre). Orchestration : superviseur = ce contexte
> (revues adversariales, commits d'integration, CI) ; executeurs = agents Opus (algos, Go,
> web complexe) ou Sonnet (docs, catalogues, deplacements purs).

## Regles d'environnement (memes que l'audit v2 — un seul poste, un seul cache par lot)

- Worktree DEDIE `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-tactique`, branche
  `feat/tactique` depuis `feat/v75`. Checkout principal : lecture seule, jamais d'ecriture.
- Commandes `go` en SERIE (jamais deux a la fois), cache propre au lot :
  `GOCACHE=C:\Users\Guillaume\AppData\Local\go-build-tactique`, `CGO_ENABLED=1`, gcc msys64
  ucrt64 dans le PATH, depuis `apps/go-api`. Integration : `-tags=integration -p 1`.
- Gates en AVANT-PLAN uniquement (aucun run d'arriere-plan, aucun outil d'attente).
- Web : `npm --prefix apps/web run typecheck`, `npm --prefix apps/web run lint`, vitest via
  `cd apps/web && node_modules/.bin/vitest run --pool=forks <filtre>` hors sandbox ;
  `git checkout -- apps/web/src/routeTree.gen.ts` avant tout staging.
- `git add` NOMME (jamais `-A` ni `.`) ; un commit par etape, prefixe `tactique(<phase>.<n>)` ;
  jamais `git stash` ; l'executeur ne POUSSE pas — le superviseur pousse apres revue.
- Aucune cuisson d'artefacts. Aucun film reel. Aucun serveur laisse tourner.
- `.ai/` : l'executeur n'ecrit que ce plan (`.ai/PLAN_TACTIQUE_2026-09-06.md`, cases +
  journal) et une entree en FIN de `.ai/thought_log.md` par phase close. Decouvertes hors
  perimetre : consignees en §7, pas traitees. Aucun emoji dans les fichiers versionnes.

## Protocole de revue et d'integration

1. Fin de phase : l'executeur commit (sans pousser), rend un rapport (items statues, gates
   joues avec sorties, commits, decouvertes, questions).
2. Le superviseur verifie le rapport sur pieces, rejoue les gates, puis lance la revue
   adversariale (skill `adversarial-review` : sous-agent au contexte frais, contrat en 6
   lignes, lentilles selon le diff, recevabilite fichier:ligne + declencheur + consequence) :
   ronde 1, tri P0/P1/P2, corrections par l'executeur, ronde 2 sur les corrections. Deux
   rondes maximum.
3. Le superviseur pousse `feat/tactique`, surveille la CI en avant-plan, journal.
4. Integration dans `feat/v75` par l'utilisateur ou sur son go, apres le lot C de l'audit au
   minimum (phases 4-7 : apres C, B et D).

## 0. Objectif et critere de succes

**Objectif** — un onglet `Tactique` sous Ascension qui repond, carte par carte et sur
l'ensemble des matchs filtres, a six questions de placement : ou je passe mon temps, ou je
meurs, ou je tue, ou je meurs isole, ou je gagne, et quelles routes je prends en sortant du
spawn. Plus le derive non spatial que ce chantier rend possible : l'ECHANGE, servi en KPI ici
et en graphes sur la page Escouade — **livre en premier** (phases 1-3), parce qu'il ne depend
d'aucun artefact et ne croise aucun lot de l'audit.

**Critere de succes V1** — sur une carte jouee au moins 10 fois : les six lectures
s'affichent en moins d'une seconde des le premier affichage, chaque cellule chaude se rouvre
sur la liste des matchs qui l'ont alimentee, un clic ouvre le rejeu 2D a l'instant
contributeur, et la page Escouade montre qui echange pour qui. Aucune re-cuisson.

**Hors perimetre V1** — portee « Tout le monde » (ossature posee, valeur non exposee) ;
comparaison de deux jeux de filtres ; export d'image ; feu croise ; Halo 5 (degradation
propre uniquement) ; toute re-cuisson d'artefacts.

**Effort** — phase 1 : petit · phase 2 : moyen · phase 3 : moyen · phases 4-6 : un lot
chacune · phase 7 : lourd (autant que 1 a 6) · phase 8 : petit.

## 1. Decisions produit — TRANCHEES avant execution

| Sujet | Decision |
|---|---|
| Nom de l'onglet | **Tactique** (FR) / **Tactics** (EN), rang 5 apres Realisations — utilisateur, 2026-09-05 |
| Emplacement | `/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique` |
| Ecran d'entree | Grille des cartes JOUEES, triees par nombre de matchs |
| Unite de lecture | Une carte a la fois. Jamais d'agregat inter-cartes. |
| Grille | 0,5 m, bornes = union des bornes des matchs retenus |
| Echelle des densites | Quantile p50 vers p95 des cellules alimentees, valeur **par match** |
| Echelle de « Ou je gagne » | **Symetrique autour de zero**, bornee par le p95 de la valeur absolue. Plancher **par cote : au moins 3 victoires ET 3 defaites** dans la cellule. Libelle = correlation, jamais une cause. |
| Cellule jamais atteinte | VIDE, jamais peinte en froid |
| Plancher par cellule | 3 matchs distincts (calibration mesuree de `mappos-build`) |
| Plancher par carte | 10 matchs ; en dessous, desaturee et non ouvrable |
| Axe QUI | `Moi / Escouade / Adversaires`. **Escouade = LA COMPOSITION CHOISIE** dans le selecteur de coequipiers de la barre L2 (le meme que la page Escouade) — arrete par l utilisateur le 2026-09-06, REMPLACE « mes coequipiers du match » (phase 2). `Adversaires` = l autre equipe du match. **Sans composition, l axe `escouade` est REFUSE** (400 type `tactical_squad_axis_without_composition`) et le client ne le propose pas : retomber sur « mes coequipiers du match » repondrait a une AUTRE question sous le meme nom. Le KPI d echange « sur cette carte » reste sur MON CAMP entier (coherent avec la page Escouade : KPI = camp, matrice = roster) — deux perimetres voisins, deux predicats. |
| Spawn : depart vs reapparition | Filtre « spawn de depart » = **premiere vie** seulement. Lecture « routes » = 15 premieres secondes de **toutes** les vies. |
| Algorithme de grappes | Densite des premiers points sur la grille de 0,5 m ; composantes connexes (8-voisinage) au-dessus du plancher de 3 matchs distincts ; nommage par le callout le plus proche du barycentre. Aucun catalogue manuel. |
| Filtres | **Barre L2 de l onglet = un MIX de l Explorateur et de l Escouade, sans rien inventer** (utilisateur, 2026-09-06) : periode/saison + playlists + modes par `features/_shared/useLocalFilterBar` (perimetre LOCAL a la page, jamais le store global) ; sessions par `SessionMultiSelect` (elles suivent la composition, comme `SquadLayout`) ; solo/escouade/mixte comme l Explorateur ; **selecteur de composition** `GamertagCombobox` comme l Escouade. **RESOLUTION COTE CLIENT, PUIS LISTE BLANCHE** (arbitrage superviseur, phase 4 bis) : la barre produit un `FilterContextInput`, le client le fait resoudre par le endpoint de resolution existant (`POST /filters/match-ids` -> `service.FilteredMatchIDs`, base JOUEUR — la seule qui porte les sessions), et **les deux endpoints tactiques passent en POST** avec `{ match_ids, coequipiers?, question?, qui? }`. Les parametres plats de filtre (playlist, mode, from, to, outcome...) DISPARAISSENT du contrat tactique : la resolution les a deja appliques. Liste vide = AUCUN match, jamais « tous » ; composition BORNEE a 3 xuids (`^\d{1,32}$`), au-dela ou mal formee = 400 typee. `matchs_filtres` = les matchs du perimetre joues SUR CETTE CARTE (dont `matchs_retenus`, les mesures, est un sous-ensemble). Le filtre de session MARCHE (utilisateur, 2026-09-06). Etat de la barre dans l URL via `usePageScope` (`tacticalScope.ts`). |
| Controles | `select` pour la question (forme `MediaToolbar`) ; segmentes pour les axes courts. **La question courante est reprise dans le titre de la carte.** |
| Stockage des rasters | **Un raster par match, calcule UNE FOIS a la cuisson** (etape post-sync `replayartifacts`, precedent `usage.go`), sidecar via `PathResolver`. La page en somme N. Aucun cache d'agregat, aucune invalidation. |
| Rattrapage | CLI `levelup tactical-rasters --backfill` : LIT les artefacts existants, ne cuit RIEN. |
| **Capabilities** | **Title-level `replay` (`CapReplay`, lot C)** gate l'onglet, les routes et les sections web. **Data-level** : `film.kill_positions` (morts/kills/gagne), `film.kill_source` (echange), **`film.replay_artifact` (lot C)** pour tout ce qui vient des pistes. Cote web, clé fine par `useDataCapability` (lot C). Ce plan ne cree AUCUNE capability. |
| Jamais un taux seul | Tout taux de couverture est servi AVEC son compte brut ET une quantite PAR MATCH (doctrine `SquadAssistPairsTable`). |
| Plancher d'echantillon | 30 morts pour un taux par joueur sans reserve ; en dessous « echantillon faible » (`explorer.briefing.low_sample`). **Aucun classement de joueurs.** |
| Maille du nuage | Isolement x couverture : par SESSION sous les reperes joueur (taille = morts). Session sous 5 morts exclue. |
| Fenetre d'echange | **5 s** — utilisateur, 2026-09-05. |
| Cas limites de l'echange | Un tueur qui abat DEUX coequipiers puis meurt dans la fenetre **echange les deux morts**. Un tueur mort de l'environnement, d'une grenade perdue ou de lui-meme **n'echange rien** : seul un kill de coequipier compte. Un kill de coequipier est un evenement dont `feed_killer_xuid` est un coequipier de la victime initiale et dont la victime est le tueur initial. |
| Rayon d'isolement | **Portee du radar** : 18 m en Arene, 24 m en BTB — utilisateur, 2026-09-05. Table mesuree dans `config/titles/{slug}/mappings/` (doctrine `regulation.toml`) ; variante non listee = pas de lecture ; rayon **par match**. |
| Cas limite de l'isolement | Tous les coequipiers deja morts -> mort **exclue du denominateur**. |
| Deux taux d'echange | Tactique : par carte, libelle « sur cette carte ». Escouade : par session et composition. |
| **Substrat de « Ou je gagne » (V1)** | **Les ENGAGEMENTS du joueur** : la position de ses kills ET de ses morts (les deux faces, sinon la lecture se confond avec « ou je tue »). Arrete par l utilisateur le 2026-09-06. A re-decider en phase 7 quand l occupation existera ; le libelle UI dit ce qu il mesure. |
| **Perimetre du KPI d echange par carte** | **Les morts de MON CAMP** (moi + mes coequipiers du match), jamais mes seules morts : meme definition que la page Escouade, denominateur le moins biaise. Arrete par l utilisateur le 2026-09-06. La lecture par joueur vit sur la page Escouade (matrice, nuage). |
| Instant contributeur | Morts / kills : l'horodatage. Occupation : premiere entree dans la cellule. Routes : debut de la vie. |
| Seuil du « Cap du moment » | Rendu seulement si au moins **30 morts d'equipe** ET (ecart d'au moins **5 points** au taux d'echange habituel OU part de morts isolees d'au moins **50 %**). Sinon non rendu. |
| Couleurs V/D | `outcome-win` / `outcome-loss` — jamais `compare-a/b`. |
| Anglicismes | `heatmap` entre au garde anti-anglicismes (lot C.4) : aucune chaine FR de ce chantier ne le contient. |

## 2. Ce qui existe deja — verifie sur pieces, a NE PAS reecrire

- **`kill_positions`** (`killer_x/y/z`, `victim_x/y/z`, `match_id`, `killer_xuid`,
  `time_ms`), vue `kill_positions_latest` ; jointure vers la victime :
  `platform/duckdb/kill_distance_repo.go:122` (`kill_positions_latest x match_kill_events_latest`).
- **`match_kill_events_latest`** : `victim_xuid`, `feed_killer_xuid`, `feed_present`, `time_ms`.
- **`service.replayService`** (`replay_service.go`) : `GetReplay`, `AvailableSet` — LE
  lecteur d'artefacts (`archlint/no_second_artifact_sink_test.go`). Refondu par le lot B.
- **`sync/replayartifacts/usage.go`** (`projeterResumeUsage`) : precedent de projection
  post-cuisson. Modifie par le lot C (porte de capability).
- **`cmd/mappos-build`** : art anterieur (grille 0,5 m, matchs DISTINCTS par cellule, filtre
  de rarete). `map_positions_jouees.json` : une carte, preuve, pas une source.
- `PathResolver.MapBackgroundPath` + sidecar ; `map_callouts.json`.
- **Types du document stocke** : `analysis/replay/document.go` — `Track{Slot, Team(-1), XUID,
  Points []Point{T,X,Y,Z}, StartFrame, EndFrame}`.
- **Coordination Escouade existante** (ne pas dupliquer) : `SquadAssistPairsTable`,
  `squad_impact.go`, `SquadEngagementGapChart`, `SquadSynergyHistoryTable`,
  `FirstBloodLanes`. Aucune ne lit une position ni un appariement dans le temps.
- **Port** : `port/repository_data.go` (`KillDistanceRepository` a cote duquel ajouter).
- **Gabarits UI** : `section-card.tsx` (+ `footer`), `KPIStrip.tsx`, `KpiCard.tsx`,
  `empty-state.tsx`, `page-unavailable.tsx`, `select.tsx`, `info-tooltip.tsx`, `FeatureGate`,
  `feedback/NarrativeBadge.tsx`, gabarit `CoachFocusCard`, `formatSignedPoints` /
  `isFullHistoryScope` (briefing Explorateur). Charts : `Heatmap2DChart`, `HistogramChart`,
  `ScatterChart`. Aucun nouveau wrapper. Palette joueurs `squad-player-1..4` validee.
- **Escouade, mecanique de donnees** : `SquadContext` distribue un `pageData` unique aux
  onglets (Contributions, Dynamique) — aucune query key propre par onglet.

## 3. Architecture — couches

```
apps/go-api/internal/
  analysis/tactical/              (NOUVEAU — pur, aucun I/O)
    grid.go        Grille(bornes, pasM) -> index cellule <-> monde
    raster.go      Rasterise(points, grille) -> Raster{cellules, comptes, matchsDistincts}
    merge.go       Somme(rasters par match) -> Raster agrege + planchers (3 distincts ; 3V et 3D)
    quantile.go    Echelle(raster) -> p50/p95 ; EchelleSymetrique(raster signe)
    spawn.go       (phase 7) GrappesDeSpawn -> composantes connexes
    tracks.go      (phase 7) Occupation(pistes, pasMs=250)
  analysis/coordination/          (NOUVEAU — pur, aucun I/O, PARTAGE Tactique + Escouade)
    measure.go     Couverture{Taux, Brut, ParMatch, N, EchantillonFaible} — jamais un taux nu
    trade.go       Echanges(kills, morts, equipes, fenetre) -> paires + delais
    isolation.go   (phase 7) ; dispersion.go (phase 7)
  domain/
    tactical.go  coordination.go
  port/
    repository_data.go            + TacticalRepository
  platform/duckdb/
    tactical_repo.go
  service/
    tactical_service.go           (phase 2) ; squad : consomme analysis/coordination (phase 3)
  api/handlers/
    tactical.go
  sync/replayartifacts/raster.go  (phase 6)
  cmd/levelup/cmd_tactical_rasters.go (phase 6)
apps/web/src/
  features/squad/                 (phase 3) sections Synergies + Dynamique, Cap du moment
  features/tactical/              (phase 4 ; phase 5 gelee jusqu'au lot D)
  components/charts/heatPaint.ts  (phase 5, GELEE — extrait de heatmapLayer.ts apres le lot D)
```

`analysis/coordination/` est PUR et PARTAGE. Regles : aucun SQL hors `platform/duckdb` ;
aucune logique dans le handler ; types dans `domain/` ; chemins par `PathResolver` ;
artefacts lus par `ReplayService` uniquement ; branchement par capability jamais par slug.

## 4. Phases

### Phase 1 — Socle pur (aucune I/O, aucun reseau) — EXECUTABLE MAINTENANT
- [x] 1.1 `analysis/tactical/grid.go` : monde <-> cellule, bornes en union, pas 0,5 m
- [x] 1.2 `analysis/tactical/raster.go` : comptes + matchs distincts par cellule, a partir de
      points `(matchID, x, y)` ; une cellule jamais atteinte n'existe pas dans le raster
- [x] 1.3 `analysis/tactical/merge.go` : somme de rasters par match ; planchers (3 matchs
      distincts ; **3 V et 3 D** pour un raster signe) ; valeur **par match**
- [x] 1.4 `analysis/tactical/quantile.go` : p50/p95 sur cellules alimentees ; **echelle
      symetrique** par p95 de |valeur| pour un champ signe
- [x] 1.5 `analysis/coordination/measure.go` : type `Couverture` (taux, brut, par match, N,
      echantillon faible sous 30) — le seul type de retour d'un taux
- [x] 1.6 `analysis/coordination/trade.go` : fenetre 5 s en constante nommee (date + autorite
      en commentaire) ; **les deux cas limites** de §1 ; sortie = paires (vengeur, venge) avec
      delai, + par mort : vengee ou non
- [~] 1.7 `domain/tactical.go`, `domain/coordination.go` : types de resultat — couvert par
      1.1, 1.5 et 1.6 : un type ne se livre pas apres le code qui le compile.
      `domain/tactical.go` (PositionSample, BornesMonde, CelluleTactique, EchelleTactique)
      est dans le commit 1.1 ; `domain/coordination.go` porte Couverture (commit 1.5) puis
      KillEvent, EquipesParMatch, MortSuivie, PaireEchange, BilanEchanges (commit 1.6).
- **Gate PASSE le 2026-09-06** (avant-plan, `GOCACHE=...\go-build-tactique`, `CGO_ENABLED=1`) :
  `go vet` sur les trois arbres, propre (0,8 s) ; `go test` idem, 7 paquets verts en 13,1 s
  (tactical 0,278 s ; coordination 0,306 s ; domain 0,313 s) ; `golangci-lint run` sur les
  deux nouveaux paquets : 0 issue. Fichiers de 30 a 315 L, fonctions <= 33 L, <= 3 parametres.
  Chaque condition du gate a ete verifiee PAR INVERSION (le test echoue si on retourne la
  condition qu'il garde) : floor -> troncature, plancher par cote -> plancher global, valeur
  par match -> par matchs de la cellule, ecart de taux -> difference brute, p95 sur |valeur|
  -> p95 signe, fenetre 5 s -> 15 s, vengeur unique -> appariement un-pour-un, tueur sans
  auteur -> vengeur valide, equipes inconnues/coequipier -> vengeable, plancher 30 -> 8,
  garde-rail du taux nu (fonction float64 ajoutee).
- **Gate** : `cd apps/go-api && go test ./internal/analysis/tactical/...
  ./internal/analysis/coordination/... ./internal/domain/...` vert ; tests a cellules posees
  a la main (comptes EXACTS) ; somme de deux rasters ; plancher par cote (3 V / 0 D -> vide) ;
  echange : mort a t, kill du tueur a t+3 s compte, a t+9 s non ; un tueur de deux
  coequipiers mort a t+4 s -> DEUX echanges ; tueur mort de l'environnement -> zero ;
  `Couverture` avec 8 morts -> echantillon faible ; `go vet` propre ; fichiers <= 500 L,
  fonctions <= 80 L.

### Phase 2 — Port, repo, service, handler (aucun artefact, aucune capability creee) — CLOSE 2026-09-06, revue ronde 1 SOLDEE
- [x] 2.1 `port/tactical.go:33-73` : `TacticalRepository` (`MapsPlayed`,
      `KillPositions`, `KillEvents`) a cote de `KillDistanceRepository`. Les deux lectures
      spatiales rendent L'UNIVERS AVEC les points (`domain.TacticalPositions` /
      `TacticalKillEvents`) : un match retenu sans position doit compter au denominateur, et
      le deduire des points l'effacerait (defaut P0 de la phase 1). Types dans
      `domain/tactical.go`, a cote du contrat du service. Deplace de
      `repository_data.go` par T12 : ce fichier depassait deja 500 L, la phase 2
      l'avait porte a 651, et `port/tactical.go` existait deja pour cette raison.
- [x] 2.2 `platform/duckdb/tactical_repo.go` (397 L) + `_test.go` (475 L, SANS tag de build —
      le gate ne pose pas `-tags=integration`, et un test derriere un tag que le gate ne pose
      pas ne garde rien ; vraies migrations shared sur `:memory:`, aucune DDL recopiee).
      Filtre par `analysis.BuildNeighborsWhereClause` : le fragment timezone canonique en
      vient, aucun litteral recopie. `publishable` exige des deux cotes (attribution PAR
      LIGNE) ; garde d'ambiguite `HAVING count(*) = 1` sur le double kill ; codes
      d'issue en parametres LIES. **R2** : les constantes prennent le prefixe `Q`
      (le garde-rail `campaign_exclusion_guard_test` ne balaye QUE les `Q*` — sous
      leurs anciens noms mes lecteurs per-player passaient sous son radar) et
      portent le token d'exclusion Campagne, resolu au call site.
      **R3** : le nom FR d'une carte vient de `metadata.asset_translations` (helper
      de paquet `mapNameFRFromAssetTranslations`, promu depuis
      `EngagementScoreRepo.resolveMapNameFR`), jamais de `match_registry.map_name_fr`
      qui est systematiquement NULLE. Tests scindes en `tactical_repo_test.go` (470 L)
      et `tactical_repo_cartes_test.go` (242 L).
- [x] 2.3 `service/tactical_service.go` + `_test.go` / `_echange_test.go` (mock du port).
      Trois questions, trois axes (`moi` / `escouade` = meme `team_id` DU MATCH moi
      exclu / `adv`). Test de reference : 12 V dont 6 muettes / 8 D dont 4 muettes,
      cellule 6 V-4 D -> 0,00.
      **R1 — LES PORTES SONT DES CLES DE LECTURE, plus des cles d'ecriture** :
      `positionsDeKillLisibles` = `film.kill_positions` (capture Infinite) OU
      `match.events.spatial` (natif Halo 5) ; `journalDesMortsFiable` =
      `film.kill_source` OU `match.killfeed.per_kill == supported` STRICTEMENT
      (`Has` accepte `degraded`, et c'est justement ce que declare Infinite pour ce
      kill-feed). Sans cela un joueur Halo 5 recevait un 503 sur une jointure qui
      marche integralement. **T9** : la couverture de localisation
      (`evenements_journal` / `evenements_localises`) est servie pour TOUS les titres —
      c'est une propriete de la mesure, pas un KPI.
- [x] 2.4 `api/handlers/tactical.go` (2 routes Huma) + `_test.go` httptest ; cablage
      `api/wire/registry_pages.go:165` (`ServiceRegistry.Tactical`) et
      `api/server_apiv1.go:715` — un seul endroit de construction ; `openapi.yaml` et
      `generated.ts` regeneres. **R4** : tags JSON snake_case sur `BornesMonde`,
      `CelluleTactique`, `EchelleTactique` et `Couverture` — le contrat melangeait
      `map_id` et `MinX`/`Brut`/`EchantillonFaible`. **T10** : `matchs_victoire` /
      `matchs_defaite` publies au niveau du raster signe. **T11** : `session` RETIRE
      du contrat (jamais applique par `BuildNeighborsWhereClause` — on n'accepte pas
      ce qu'on n'honore pas).
      **AMENDE PAR G2 (revue ronde 1 de la PHASE 3, 2026-09-06)** : l'univers passe aux
      `Rasterise*` est desormais celui des matchs MESURES, et le contrat publie
      `matchs_filtres` (tous les matchs du filtre) A COTE de `matchs_retenus` (les
      mesures). Un match dont le film n'a jamais ete decode ne peut alimenter aucune
      cellule : le garder au denominateur faisait varier l'intensite avec la COUVERTURE DE
      FILM au lieu du jeu. L'exemple de reference 12 V / 8 D reste VALIDE — ses 20 matchs
      sont mesures, 10 sont seulement muets DE MOI (le zero LEGITIME de la phase 1, a ne
      pas confondre avec l'ILLISIBLE de G2).
- [x] 2.5 `slog.InfoContext` sur chaque calcul : `tactical_service.go:93` (cartes),
      `:133` (`player`, `titleSlug`, `map_id`, `question`, `qui`, `matchs_retenus`,
      `cellules`, `points_ignores`, `duration`), `:275` (echange).
      `ErrorContext(..., "err", err)` sur toute erreur (service, repo `degrader` /
      `scanRows`, handler `mapTacticalError`). Aucune erreur avalee : une ligne SQL
      illisible est signalee PUIS propagee, et un filtre ecarte est journalise.
- **Gate PASSE le 2026-09-06** (avant-plan, une commande `go` a la fois,
  `GOCACHE=...\go-build-tactique`, `CGO_ENABLED=1`, depuis `apps/go-api`) : `go vet` propre
  sur les 7 arbres (1,9 s) ; `go test -count=1` vert sur 22 paquets en 42,3 s (duckdb
  39,1 s ; api 22,7 s ; service 10,1 s ; handlers 9,2 s ; archlint 6,9 s ; contracttest
  0,4 s) ; `golangci-lint run --new-from-merge-base=origin/main` (LE ratchet de la CI) :
  **0 issue** ; `go run ./cmd/openapi-gen -check` : a jour. Ratchets verts :
  `no_slug_comparison_test`, `no_data_path_join_test`, `no_raw_start_time_literal_test`,
  `no_inline_objective_latest_view_test`, `no_raw_outcome_literal_test`,
  `tactical_pure_test`, et `no_raw_kill_scope_literal_test` — celui-ci a MORDU (`read_path`
  ecrit en clair dans la fixture, corrige en `killscope.ReadPathFilmWalk`). Seuils : fichiers
  neufs <= 475 L, fonctions <= 45 L, <= 5 parametres. `internal/sync/` et `internal/persist/`
  NON touches : pas de passe `-tags=integration`.
- **Gate REJOUE apres la revue ronde 1, le 2026-09-06** (avant-plan, en serie) :
  `go vet` propre sur les 9 arbres (2,7 s) ; `go test -count=1` vert sur 24 paquets en
  43,5 s (duckdb 40,0 s ; api 22,5 s ; service 10,0 s ; handlers 10,0 s ; archlint 6,8 s ;
  tactical 0,23 s ; coordination 0,24 s ; contracttest 0,46 s) ;
  `golangci-lint run --new-from-merge-base=origin/main` : **0 issue** ;
  `openapi-gen -check` : a jour. Seuils reverifies APRES ajouts : les deux fichiers de
  test avaient franchi 500 L, scindes comme `merge_test.go` en phase 1
  (`tactical_repo_test.go` 470 / `tactical_repo_cartes_test.go` 242 ;
  `tactical_service_test.go` 423 / `tactical_service_echange_test.go` 267).
  Quatre INVERSIONS jouees : R1 (predicat reduit a `film.kill_positions` -> le test des
  positions natives tombe), R2 (token retire de `QTacticalUnivers` -> le garde-rail
  structurel tombe en le nommant), T1 (`prendVictime := question == Morts` -> la face
  victime de « ou je gagne » disparait), T5 (garde de composition retiree -> un joueur
  inconnu tombe dans `adv`).
- **Gate** : `go test ./internal/platform/duckdb/... ./internal/service/... ./internal/api/...`
  vert ; repo en DuckDB `:memory:` ; handler `httptest` ; titre sans `film.kill_positions`
  -> 503 ; ratchets `no_slug_comparison_test`, `no_data_path_join_test` verts ;
  `make generate-types` puis `git diff --exit-code apps/web/src/lib/api/generated.ts`
  ne montre que les nouveaux types.

### Phase 3 — L'ECHANGE SUR LA PAGE ESCOUADE — CLOSE 2026-09-06, revues rondes 1 ET 2 SOLDEES (tous les items statues)
- [x] 3.1 Service Escouade : `service/teammates/teammates_squad_echange.go` (279 L) +
      `domain/squad_echange.go` (122 L, tags snake_case), branche dans
      `teammates_service.go` (`WithEchange`) et `GetPage`. L'echange entre dans le
      `pageData` de `SquadContext` — AUCUNE query key propre. Mesure par
      `coordination.Echanges` / `Ripostes` / `Mesurer` UNIQUEMENT ; aucun appel du
      service Tactique (le PORT est lu directement, comme lui).
      **KPI = MON CAMP** (decision utilisateur 2026-09-06) ; **matrice = ROSTER**
      (un vengeur de passage compte au taux, pas a la matrice : la page ne sait pas
      le nommer — doctrine SquadAssistPairsTable). **HABITUEL** = la meme mesure sur
      `allSquadRowsForTimeline` (historique complet de la composition), meme
      mecanique que `buildBriefingBaseline` de l'Explorateur.
      **MapID devient OPTIONNEL** (`domain/tactical.go:TacticalQuery`,
      `QTacticalUnivers` : predicat de carte neutralise par parametre) — UN seul SQL
      pour les deux pages, la constante reste sous le radar de
      `campaign_exclusion_guard_test` ; `KillPositions` exige toujours une carte.
      **Porte partagee** : `journalDesMortsFiable` quitte le service Tactique pour
      `games.JournalDesMortsFiable` (`internal/games/kill_journal_gate.go`) — deux
      lecteurs, une seule definition. Porte fermee -> section OMISE, jamais des zeros.
      **`coordination.Ripostes`** (nouveau, `riposte.go`) rend les memes morts SANS
      borne de temps : c'est la seule source des deux barres hors fenetre. Noyau
      commun `suivreMorts(kills, equipes, fenetreMs)` — pas de seconde recherche de
      vengeur. `domain.MortSuivie` ajoute a la liste blanche du garde-rail du taux nu,
      justification datee sur place. Contrat : `openapi-gen` + `generated.ts` (+40 L,
      additions pures).
      **REVUE R1 — G2 (touche AUSSI la phase 2)** : le denominateur « par match » ne
      compte plus que les matchs MESURES (journal des morts lisible). Le numerateur ne
      peut venir que d'eux, et les films Theater EXPIRENT : diviser par tous les matchs du
      filtre faisait varier la grandeur avec la COUVERTURE DE FILM au lieu du jeu
      (20 matchs sur 20 decodes contre 2 sur 20 -> 0,20 et 0,02 pour le meme jeu).
      Escouade : `Couverture.ParMatch` et les cases se divisent par `matchs_mesures`,
      `matchs_total` reste publie. Le lecteur rend le drapeau par match (EXISTS sur
      `match_kill_events_latest`, `publishable` exige comme dans les deux lectures,
      parametre neutre et ratchet Campagne conserves).
      **REVUE R1 — G1** : `bucketDelai` PARCOURT `bornesDelaiMs` au lieu de recopier le
      decoupage en arithmetique ; intervalles semi-ouverts sauf celui qui ferme la fenetre
      (5 000 ms reste un echange). Test de PROPRIETE sur 13 delais poses a la main.
- [x] 3.2 Onglet **Synergies** : `SquadEchangeMatrixCard.tsx` — `SectionCard` « Qui
      echange pour qui » + `Heatmap2DChart` (ligne = vengeur, colonne = venge,
      orientation `SquadAssistPairsTable`), palette `frequency` mono-teinte, diagonale
      absente, cases a zero emises. Bandeau de couverture AU-DESSUS du graphe + ligne
      narrative. Deux `NarrativeBadge` « le plus / le moins couvert », rendus seulement
      a >= 2 joueurs, plancher de 30 morts atteint ET >= 3 vengeances d'ecart.
      `Heatmap2DChart` gagne `paletteMode='frequency'` + `formatTooltip` /
      `formatTooltip` (son libelle par defaut parle de taux de victoire) — appelants
      existants inchanges.
      **REVUE R1 — W1 (P0)** : le wrapper deduit ses axes de l'ORDRE D'APPARITION des
      points ; sauter la diagonale decalait l'axe X d'un cran (roster [A,B,C,D] ->
      colonnes B,C,D,A), et sur un duo les deux axes sortaient inverses. `matriceSeries`
      emet desormais TOUTES les cases dans l'ordre du roster, diagonale comprise avec une
      valeur VIDE (`value: number | null`). **W4** : la prop morte `formatLabel` est
      supprimee. **W5** : les branches neuves du wrapper sont testees.
- [x] 3.3 Onglet **Dynamique** : `SquadEchangeDelaiCard.tsx` — `SectionCard` « Delai
      d'echange » + `HistogramChart`, 5 barres dans la fenetre en couleur de serie,
      2 hors fenetre en `divergent-neutral` (la palette n'a pas de token `muted`).
      Fenetre marquee dans le pied de carte ET en suffixe d'etiquette (`markLine` non
      exposee par le wrapper). Pied = definition en une phrase + couverture.
      `HistogramChart` gagne `binAttenuated` (il ne peint que `series[0]` : deux
      series auraient ete ignorees en silence).
      **REVUE R1 — W2** : `divergent-neutral` vaut #60A5FA (blue-400) dans la palette PAR
      DEFAUT — plus soutenu que la serie —, et AUCUN token semantique du depot n'est
      achromatique dans les quatre palettes (mesure sur pieces). L'attenuation passe donc
      par la COULEUR DE SERIE en opacite reduite + liseré tireté, sans dependance de
      palette. La doc qui disait « le gris neutre de la maison » etait inversee.
- [x] 3.4 KPI « Taux d'echange » : `SquadEchangeKpi.tsx` (`KPIStrip`, monte dans
      `SquadLayout.tsx`) — valeur + « N vengees sur M » + ecart en points
      (`formatSignedPoints`) + fleche `KPITrend`, masque si `isFullHistoryScope`.
      **REVUE R1 — W3** : l'ecart, son arrondi et le masquage plein-historique quittent le
      composant pour `squadEchange.logic.ecartEchange` (pure, consommee aussi par
      `capDuMoment`) ; `SquadEchangeKpi.test.tsx` cree — la tuile n'avait AUCUN test, et
      supprimer le masquage ou inverser le signe passait.
      **Les deux helpers ont ETE DEPLACES** dans `apps/web/src/lib/baseline.ts` (avec
      leurs tests) et l'Explorateur pointe dessus : `tools/lint-cross-feature-imports.mjs`
      est a son plafond (7/7), un `squad -> explorer` de plus l'aurait franchi, et une
      copie aurait donne deux definitions du meme ecart. Les deux commentaires qui
      citaient l'ancienne adresse sont corriges dans le meme commit.
- [x] 3.5 « Cap du moment » : `SquadEchangeCapCard.tsx` en tete de Synergies, gabarit
      `KpiCard` accent `info` dans LES DEUX SENS (jamais rouge). Regle de seuil de §1
      appliquee dans `squadEchange.logic.capDuMoment` : rendue SEULEMENT a >= 30 morts
      d'equipe ET >= 5 points d'ecart ; sinon PAS rendue (aucun etat vide). Troisieme
      refus ajoute et documente : sans reference mesuree (habituel a N=0) il n'y a pas
      d'ecart.
      **REVUE R1 — W6** : la phrase porte deja la direction (« de moins » / « de plus ») ;
      le nombre est donc une MAGNITUDE (`lib/baseline.formatPoints`). `formatSignedPoints`
      sur une valeur absolue forcait un « + » dans une phrase qui disait « de moins ».
      **W9** : la borne exacte des badges (`n === 30`) est posee a cote de son voisin a 29.
- [x] 3.6 Toutes les chaines dans `manifests/squad.toml` (34 cles `squad.echange.*`,
      FR + EN), regen `build_i18n_manifests.mjs`, acces par `squadEchangeStrings.ts`
      (modele `squadFocusStrings`). Zero chaine en dur dans les composants, zero hex,
      zero classe Tailwind couleur (`lint-no-hardcoded-colors` : 0 violation).
      Garde-rail dedie `squadEchange.i18n.test.ts` : FR et EN non vides, aucune cle
      orpheline, aucune cle non resolue, et aucun anglicisme de ce chantier en FR.
      **REVUE R1 — W8** : le garde s'arretait a l'ACCESSEUR et annoncait « aucune
      orpheline » a tort (`squad.echange.empty_description` etait declaree, exposee, et
      affichee par personne). La cle est supprimee (FR + EN) et le garde va desormais
      jusqu'au COMPOSANT, inversion jouee. **W7** : les fixtures dupliquees — et deja
      divergentes — du test de logique sont supprimees au profit de
      `squadEchange.fixtures`. **W9** : la borne exacte des badges (`n === 30`) est
      posee a cote de son voisin a 29.
- [~] 3.7 **Gate web des sections par `useDataCapability('film.kill_source')` — COUVERT
      PAR LA DEGRADATION GO, et le lot C tranche CONTRE ce gate.** Statut revu de `[!]`
      a `[~]` le 2026-09-06, apres l'integration du lot C de l'audit v2 (merge
      `741e1731f`) qui apporte le hook : `apps/web/src/lib/capabilities/dataCapabilities.ts`.
      Le hook EXISTE desormais — mais sa doctrine, lue sur pieces, refuse ce branchement
      pour DEUX raisons, chacune suffisante :
      **(a) N'ENTRE DANS `DATA_CAPABILITIES` QU'UNE CLE EFFECTIVEMENT GATEE COTE UI.**
      C'est un sous-ensemble VOLONTAIRE du vocabulaire Go ; une cle listee « pour plus
      tard » serait du vocabulaire mort (CLAUDE.md n 7). `film.kill_source` n'y figure
      pas, et n'a aucune raison d'y entrer pour une section qui ne se gate pas ici.
      **(b) UNE PORTE QUI ARRIVE DEJA DANS LE PAYLOAD NE SE RELIT PAS.** Le fichier
      documente exactement ce cas avec `film.usage_summary`, ABSENT de la liste parce que
      sa porte de titre arrive deja en clair dans la reponse : « la lire une seconde fois
      ici ferait deux sources de verite pour une seule question, plus une requete
      inutile ». Notre section est dans la meme situation, en plus tranche : elle est
      OMISE du `pageData` par le Go quand `games.JournalDesMortsFiable` est fermee
      (`internal/games/kill_journal_gate.go`), et l'absence EST la reponse.
      **ET IL SERAIT FAUX POUR HALO 5.** La porte Go est un OU sur DEUX provenances :
      `film.kill_source` (Infinite, decodage du film) OU `match.killfeed.per_kill`
      a `supported` STRICTEMENT (Halo 5, kill-feed natif). Un
      `useDataCapability('film.kill_source')` cote web fermerait la section pour un
      joueur Halo 5 dont le serveur SERT pourtant la mesure — le defaut exact que la
      correction R1 de la phase 2 avait deja elimine cote Go.
      **CE QUI TIENT LIEU DE GATE, ET IL EST TESTE** : porte fermee -> section absente du
      `pageData` -> les trois composants ne sont pas montes (`SquadSynergiesPage`,
      `SquadDynamiquePage`, `SquadEchangeKpi` : `if (!echange) return null`) ; contrat
      present mais vide -> `EmptyState`. Verifie par
      `SquadEchangeConstatCard.test.tsx` (« ne rend RIEN quand la section est absente du
      contrat ») et `SquadEchangeMatrixCard.test.tsx` (etat vide sans paire).
- **Gate PASSE le 2026-09-06** (avant-plan, une commande `go` a la fois,
  `GOCACHE=...\go-build-tactique`, `CGO_ENABLED=1`). Go : `go vet` propre sur les
  9 arbres, `go test -count=1` vert, `golangci-lint run --new-from-merge-base=origin/main`
  a 0 issue, `openapi-gen -check` a jour. Web : `typecheck` propre, `lint` 0 erreur,
  vitest `squad` + `lib/baseline` + garde anti-anglicismes verts,
  `lint-no-hardcoded-colors` 0 violation, `lint-cross-feature-imports` a 7/7
  (INCHANGE — c'est ce qui a impose le deplacement de 3.4), manifests regeneres sans
  diff residuel.
- **Gate** : typecheck + lint + vitest cible (squad) ; matrice sur un jeu pose a la main ;
  etat vide -> `EmptyState` ; **Cap non rendu sous le seuil** ; **anti-biais** (8 morts et
  100 % -> echantillon faible, classe devant personne) ; `node tools/lint-no-hardcoded-colors.mjs` ;
  parite FR/EN ; garde anti-anglicismes.
- **Livrable** : push par le superviseur + CI verte au niveau job.

### Phases 4-7 — etat au 2026-09-06 apres integration des lots C, B et F dans `feat/v75` (fusionnes dans `feat/tactique`, 741e1731f)
> **Phases 4 et 6 : EXECUTABLES.** La 6 ne dependait que de B (`ReplayService` /
> `domain/replaydoc`) et de C (porte `film.replay_artifact` dans `replayartifacts`) — tous
> deux integres. La 4 ne touche que des routes, des manifestes et des cles de requete.
> Ordre : **4 puis 6**.
> **Phase 5 : GELEE jusqu'au lot D** (modele web) : elle touche `replay.tsx` (refondu par
> D.6, D.8) et extrait le peintre de `heatmapLayer.ts` (arbre reorganise par D.11, D.13).
> La faire avant = la faire deux fois. A la reprise : refusionner `feat/v75`, relire D.13
> (`lib/replay/`) pour y loger le peintre.
> **Phase 7 : apres la 6** (elle en consomme les sidecars) ; sa surface Escouade (nuage
> isolement x couverture) ne depend pas de D.
> Ce que C apporte et que ce plan consomme : `CapReplay` (gate title-level de l'onglet,
> phase 4), `film.replay_artifact` (porte data-level des sidecars, phase 6),
> `useDataCapability` (cote web, regle des deux portes — voir la case 3.7).

### Phase 4 — Grille des cartes — CLOSE 2026-09-06, revues rondes 1 ET 2 SOLDEES (11 constats)
> ORDRE D'EXECUTION : 4.4, puis 4.2, puis 4.3+4.1 — l'ordre des DEPENDANCES, pas celui de
> la liste. La vignette de 4.3 consomme l'endpoint de 4.4 et les cles de 4.2 ; la route de
> 4.1 importe la page de 4.3 ET type l'ecriture de `?carte=` que cette page fait. Chaque
> commit compile ; l'inverse aurait impose des commits morts.
- [x] 4.1 Route file-based `routes/.../players/$playerSlug/ascension/tactique.tsx`
      (`validateSearch` : `carte`) ; 5e onglet « Tactique » / « Tactics » dans
      `AscensionLayout.tsx:128-138`, libelles `tabTactical` dans le dictionnaire
      d'Ascension (`features/ascension/i18n.ts:20,194,429`) comme les quatre autres.
      DEUX PORTES : `FeatureGate capability="replay"` masque l'onglet ;
      `RouteCapabilityGate` (via `features/tactical/TacticalTab.tsx`) rend l'etat
      « indisponible pour ce titre » A LA PLACE de la page — qui n'est donc pas montee et
      n'emet aucune requete. Une URL s'ouvre sans passer par la barre d'onglets : une seule
      porte n'aurait garde que le chemin nominal.
      **ECART ASSUME AU BRIEF, sur pieces** : le brief demandait `PageUnavailable`. Ce
      gabarit-la est celui d'ADR 0029 (ressource EXISTANTE mais inaccessible au joueur :
      match non participe, acces refuse) et exige des libelles et des actions. Le gabarit du
      depot pour « capability absente au niveau route » est `RouteCapabilityGate` ->
      `FeatureUnavailable` -> `EmptyStateCard`, deja utilise par la route Ascension parente
      (`ascension.tsx`, capability `lusr`) et porteur du libelle FR/EN de `replay`. Prendre
      `PageUnavailable` aurait recopie ce libelle et fabrique un second gabarit pour le meme
      etat. Aucun chrome maison : `EmptyStateCard` est un gabarit de la liste.
      **Livre dans le meme commit que 4.3** (`tactique(4.3+4.1)`) : dependance mutuelle
      (la route importe la page ; le schema de recherche de la route type l'ecriture de
      `?carte=` par la page), les separer donnait un commit qui ne compile pas.
      `routeTree.gen.ts` REGENERE par le build (`npm run build`), jamais edite : 26 lignes
      ajoutees, toutes relatives a cette route — verifie ligne a ligne.
- [x] 4.2 `lib/i18n/manifests/tactical.toml` (14 cles, FR + EN, vocabulaire arrete :
      carte / matchs / victoires / defaites / plancher) + regen
      (`lib/i18n/generated/tactical.ts`) ; accesseurs types dans
      `features/tactical/i18n.ts`. Le manifeste entre dans
      `lib/i18n/no-anglicisms.guard.test.ts:253` DES SA CREATION — c'est celui ou
      « heatmap » serait le plus tentant. Query keys `lib/query/keys.ts:171-180`
      (`tacticalMaps(playerSlug, titleSlug, filterHash)` et `tacticalMapBackground(...)`,
      `titleSlug` en 2e segment), declarees title-scopees dans
      `keys.title-slug.guard.test.ts:78-81` — le volet COMPLETUDE reste vert.
- [x] 4.3 `features/tactical/` : `tacticalLogic.ts` (PUR — tri par nombre de matchs, depart
      stable par le nom, lecture du verdict de plancher, barre V/D, couverture, traduction
      du filtre global), `queries.ts`, `i18n.ts`, `TacticalPage.tsx` (une `SectionCard`
      « Cartes jouees », `footer` = couverture + phrase du plancher, `EmptyState` sans
      carte), `TacticalMapTile.tsx` (fond + nom FR du contrat + « N matchs » + barre
      `outcome-win` / `outcome-loss`).
      **LE PLANCHER N'EST PAS RECALCULE COTE CLIENT** : le serveur publie `sous_plancher`
      par carte et `plancher_matchs` pour la page ; le client LIT le verdict et NOMME le
      seuil. Carte sous le plancher : affichee (le joueur doit voir qu'il y a joue),
      desaturee par des utilitaires SANS couleur (`opacity-60 grayscale`), bouton
      DESACTIVE (`aria-disabled`), raison en clair « N matchs sur 10 requis ».
      **DECISION — LE CLIC SELECTIONNE, IL N'OUVRE PAS ENCORE, ET IL LE DIT.** La vue par
      carte est la phase 5, gelee jusqu'au lot D : sa route n'existe pas. Un `Link` vers
      elle ne compilerait pas ; un `navigate` vers une chaine construite serait un lien
      MORT (404). Le bouton ecrit donc `?carte=<map_id>` dans l'URL — nom accessible
      « Selectionner <carte> », etat `aria-pressed`, selection visible, URL partageable —
      et c'est exactement l'etat que la phase 5 consommera.
      **La SESSION est retiree du filtre envoye** (coherence avec T11 : aucune requete
      shared de cet onglet ne l'honore ; transmettre un filtre sans effet ferait croire a
      l'appelant qu'il a filtre).
- [x] 4.4 Endpoint image par CARTE : `GET /players/{slug}/tactical/{map_id}/background.png`
      (route chi nue) + son CALAGE `.../background` (Huma), `api/handlers/tactical.go:130-200`.
      AUCUNE SECONDE RESOLUTION : la cascade de cles quitte `resolveBackgroundKey` pour
      `resolveBackgroundKeyDepuis` (`service/replay_map_background.go:150`), la lecture du
      PNG pour `readBackgroundImage` (`:82`), la cascade des noms candidats pour
      `assemblerIdentites` (`platform/duckdb/replay_map_repo.go:130`) — les DEUX entrees
      (par match, par carte) les consomment. `PathResolver.MapBackgroundPath` inchange.
      Le handler porte DEUX factories (Tactical + Replay) : faire transiter le fond par le
      service Tactique aurait ete un service qui en appelle un autre.
      **LE FOND N'EST PAS SOUS `LocalOnlyReplay`** : ce garde protege les trajectoires
      decodees du film ; une image de carte est une donnee de REFERENCE versionnee, et sous
      le garde la grille aurait ete vide en production sans que rien ne soit protege
      (teste : appel depuis une adresse non locale -> 200). Contrat : fragment manuel pour
      la route binaire (comme le PNG par match), allowlist chi-brut du garde-rail du
      document partage, `openapi.yaml` et `generated.ts` regeneres.
- **Gate PASSE le 2026-09-06** (avant-plan, une commande `go` a la fois,
  `GOCACHE=...go-build-tactique`, `CGO_ENABLED=1`). Go : `go vet` propre sur 8 arbres
  (1,1 s) ; `go test -count=1` vert sur 25 paquets (duckdb 42,6 s ; api 22,7 s ; service
  10,2 s ; handlers 10,0 s ; archlint 7,5 s ; contracttest 0,5 s) ;
  `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` : **0 issue** ;
  `openapi-gen -check` : a jour. Web (depuis `apps/web`, `node_modules/.tmp` purge) :
  `typecheck` propre ; `lint` **0 erreur** (30 warnings, tous preexistants — le seul que ce
  lot avait introduit, un `setState` dans un effet pour l'URL d'objet du fond, a ete
  supprime en faisant de l'URL l'entree de cache elle-meme) ;
  `vitest run --pool=forks tactical ascension keys capabilities` : 21 fichiers, 171 tests
  verts ; garde anti-anglicismes vert ; `lint-no-hardcoded-colors` 0 violation ;
  `lint-cross-feature-imports` a **7/7 INCHANGE** (la feature n'importe que `@/lib` et
  `@/components`) ; manifestes regeneres sans diff residuel.
  **SIX INVERSIONS JOUEES** : branche « cle map_id d'abord » neutralisee ->
  `TestMapBackgroundForMap_ParMapID` tombe ; garde de session retiree du filtre -> le test
  T11 tombe ; `disabled` et desaturation retires de la vignette -> le test du plancher
  tombe ; `FeatureGate` retire de la barre d'onglets -> le test « pas d'onglet sans
  replay » tombe ; `RouteCapabilityGate` retire de `TacticalTab` -> les deux tests de la
  seconde porte tombent.
- [x] 4.5 **Revue adversariale ronde 1 — 8 constats, tous corriges** en 2 commits
      `tactique(4.5)`. Aucun report.
      **G1 (P1, P0 sur un hote Windows expose) — LE map_id N'ETAIT JAMAIS VALIDE AVANT LE
      SYSTEME DE FICHIERS.** C'est la premiere cle de fond entierement controlee par
      l'appelant (sur le chemin par match elle venait de `match_registry`), et elle
      atteignait `filepath.Join(map_backgrounds, cle + ".json")` avec pour seul controle un
      `TrimSpace`. Sous Windows, `..\..\x` traverse chi comme UN SEUL segment et
      `filepath.Join` resout l'antislash comme separateur : `os.Stat` et `os.ReadFile`
      sortaient du repertoire des fonds. DEUX PORTES desormais :
      `handlers.MapIDValide` (`api/handlers/tactical.go:252`, liste blanche
      `^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`, appelee AVANT toute resolution de service) et
      `service.cleDeFondSure` (`service/replay_map_background.go:216`), dans
      `resolveBackgroundKeyDepuis` — dernier point avant `PathResolver`, gardant les DEUX
      branches (cle map_id et cle rendue par l'index des noms : c'est la CLE qui doit etre
      sure, quelle que soit sa provenance) ; une cle VIDE reste legitime (carte native).
      **AUCUNE NORMALISATION** : le motif s'applique a la valeur BRUTE — un `TrimSpace`
      prealable faisait de `carte%20` un `carte` valide, defaut trouve par le test lui-meme.
      **AMENDEMENT AU BRIEF, au service de sa propre raison** : le brief tranchait
      `map_background_not_available` sur les trois routes ; la lecture de placement rend
      `tactical_map_unknown`, qui est SON code d'absence. Un code propre a la validation
      dirait a l'appelant qu'il a franchi le routeur mais pas le filtre — l'oracle que le
      brief voulait justement eviter en refusant le 400. Un test l'exige : map_id hostile et
      carte jamais jouee rendent des reponses OCTET POUR OCTET identiques, ce qui a impose
      de factoriser le message d'absence du fond (`messageSansFond`).
      Tests : `api/handlers/tactical_mapid_test.go` (predicat aux deux bords, les trois
      routes, indiscernabilite) et `service/replay_map_background_traversee_test.go`
      (les deux branches + la PREUVE PAR LE CHEMIN : les cles de la table font reellement
      sortir `filepath.Clean(MapBackgroundMetaPath(...))` du repertoire des fonds, avec
      sentinelle anti-vacuite, et toute cle acceptee y reste).
      **G2 (P1) — LE TEST DU GARDE LOCAL NE POUVAIT PAS ECHOUER.** Il ne montait aucun
      middleware et partait deja d'une adresse non locale (defaut du defaut de `httptest`,
      deja documente dans `replay_test.go`). Le MONTAGE est desormais garde par un ratchet
      sur le site reel : `archlint/tactical_background_local_gate_test.go`, sur le modele et
      avec l'extracteur de `replay_routes_capability_gate_test.go` — generalise par
      constructeur (`sitesDeMontage(fichier, ctor)`) plutot que recopie. **RATCHET A DEUX
      FACES** : les routes tactiques ne sont pas sous `LocalOnlyReplay`, ET le rejeu l'est
      toujours — sans la seconde, la premiere passerait aussi si le garde disparaissait du
      depot. Le test HTTP est renomme et reecrit pour ce qu'il couvre reellement : le
      handler ne branche sur aucune adresse d'appel.
      **G3 (P2)** — les deux branches `missing_map_id` (400) supprimees, remplacees par la
      validation G1. `openapi.yaml` INCHANGE (690129 octets) : ce 400 n'etait declare nulle
      part au contrat, donc `generated.ts` ne bouge pas non plus. `replay.go` non touche
      (meme forme morte, hors perimetre — consignee au §7).
      **W1 (P1)** — le passage du filtre a la requete n'etait asserte nulle part (`api.get`
      double, personne ne regardait le chemin) : figer la cle ou retirer le suffixe d'URL
      laissait 15 tests verts pendant qu'a l'ecran un changement de periode resservait la
      grille precedente. Trois tests dans `TacticalPage.test.tsx`.
      **W2 (P1)** — le chemin NOMINAL du fond n'etait jamais joue (`getBlob` toujours
      double en rejet) : `<img src={fond ?? ''}>` serait passe, soit une icone cassee sur
      chaque vignette sans fond. Deux tests, image resolue et image refusee.
      **W3 (P1)** — `useMatchRoute` double a « toujours faux » : deux onglets pouvaient
      briller ensemble et retirer `&& !isTactical` passait. Le double suit desormais la
      route courante ; deux tests exigent EXACTEMENT UN `aria-selected`.
      **W4 (P2)** — le fond etait cache PAR JOUEUR alors que c'est une donnee de reference
      du TITRE : N images retenues par joueur consulte, pour le meme contenu, et la borne
      ecrite en commentaire etait fausse. Cle passee a
      `tacticalMapBackground(titleSlug, mapId)` (`lib/query/keys.ts:185`), classee dans la
      categorie « title-scopee par le 1er argument » qui existait deja au garde-rail
      (`assetMaps`, `presence`). L'URL de fetch garde le joueur — la route reste derriere
      l'ownership. Borne reecrite.
      **W5 (P2)** — pointeur de doc corrige vers `TacticalTab.test.tsx`.
- **Gate REJOUE apres la revue ronde 1, le 2026-09-06** (avant-plan, en serie). Go :
  `go vet` propre sur 7 arbres ; `go test -count=1` vert sur 24 paquets (duckdb 44,4 s ;
  api 23,1 s ; service 11,2 s ; handlers 9,5 s ; archlint 8,9 s ; title 7,5 s ;
  contracttest 0,5 s) ; `golangci-lint run --new-from-merge-base=origin/main` : **0 issue** ;
  `openapi-gen -check` a jour et `generated.ts` INCHANGE. Web : `typecheck` propre ; `lint`
  **0 erreur** (30 warnings preexistants) ; `vitest tactical ascension keys capabilities` :
  21 fichiers, **178 tests** verts ; `lint-no-hardcoded-colors` 0 violation ;
  `lint-cross-feature-imports` **7/7 inchange** ; manifestes sans diff residuel.
  **SIX INVERSIONS/MUTATIONS JOUEES** : motif de validation neutralise -> 3 tests de
  handler tombent ; `cleDeFondSure` neutralisee -> le test de traversee tombe en montrant
  `err = <nil>` (la resolution ABOUTISSAIT sur une cle `..`) ; ligne de montage tactique
  deplacee dans le groupe `LocalOnlyReplay` -> le ratchet tombe en nommant
  `server_apiv1.go:713` ; cle de cache figee + suffixe d'URL retire -> le test du filtre
  tombe ; `<img>` rendue inconditionnelle -> les deux tests du fond tombent ;
  `&& !isTactical` retire -> le test « un seul onglet selectionne » tombe.
- **Gate** : typecheck + test-web ; couleurs ; parite FR/EN.
### Phase 4 bis — Barre L2 et perimetre de matchs (items 4.5 et 4.6) — CLOSE 2026-09-06, revues rondes 1 ET 2 SOLDEES (20 constats)
> Arbitrages utilisateur du 2026-09-06 : le filtre de session doit MARCHER ; la barre L2 est un
> mix Explorateur + Escouade ; « Escouade » = la composition choisie.
> Architecture tranchee par le superviseur : **resolution du perimetre COTE CLIENT** par le
> endpoint de resolution existant, puis **POST avec liste blanche de match_id** — les deux
> endpoints tactiques perdent leurs parametres plats de filtre.
- [x] 4.5 **Perimetre de matchs par liste blanche** — `domain.ListeBlancheMatchs`
      (`domain/tactical.go:114-141`, type NOMME plutot qu'un `[]string` : le zero-value est
      « aucune restriction », et TOUTE liste — vide comprise — vient de `RestreindreAux`,
      parce qu'avec un slice nu « liste vide » et « pas de liste » sont le meme `len() == 0`
      et qu'un appelant qui oublie la sienne servirait l'historique ENTIER en silence).
      `TacticalQuery` perd `Filtre`, gagne `Matchs` + `Coequipiers` ;
      `TacticalScope` / `TacticalRasterRequest` cotes service ;
      `ErrTacticalEscouadeSansComposition`.
      **Repo** : `platform/duckdb/tactical_repo_univers.go` (NOUVEAU, 175 L — le fichier
      d'origine etait a 486 et le perimetre l'aurait pousse au-dela de 500) porte
      `QTacticalUnivers`, `clausePerimetre` et `chargerUnivers`. La liste blanche est liee
      par `Placeholders`/`ToAnySlice` (les helpers canoniques du paquet, pas une 5e copie du
      motif) ; la composition est UN `EXISTS` PAR XUID sur `match_participants` avec
      `c.team_id = mp.team_id` — donc un ET, et un coequipier qui jouait EN FACE ce jour-la
      exclut le match. Liste vide -> `AND FALSE`. Ratchet Campagne, parametre neutre de
      carte et drapeau `Mesure` (G2) conserves. Taxonomie de modes RETIREE du lecteur (il
      ne classe plus rien) : `WithModeTaxonomy` et `logIgnoredFilters` supprimes avec leurs
      deux call sites (`registry_pages.go:167`, `registry_pages_home.go:225`).
      **Service** : `service/tactical_service_perimetre.go` (NOUVEAU, 149 L) porte
      `validerLecture`, `requeteDuScope`, `compositionNettoyee` et les predicats d'axe.
      `escouade` = LA COMPOSITION CHOISIE (appartenance a la liste de xuids), refusee sans
      elle ; `adv` inchange ; **le KPI d'echange garde `campDuMatch`** — mon camp ENTIER,
      qui ne doit PAS retrecir parce qu'on a nomme deux coequipiers dans la barre.
      `MatchsFiltres` = les matchs du perimetre joues SUR CETTE CARTE, dont
      `MatchsRetenus` (les mesures) est un SOUS-ENSEMBLE — corrige en ronde 1 (G4) : la
      premiere version publiait la taille de la liste blanche, toutes cartes confondues.
      **API** : les deux lectures passent en POST ; corps `{ match_ids, coequipiers?,
      question?, qui? }` ; `MapIDValide` reste EN TETE de chemin ; nouveau code
      `tactical_squad_axis_without_composition` (400) — distinct de l'axe inconnu, parce que
      l'axe EXISTE et que c'est la composition qui manque. **`/tactical/` entre dans
      `middleware.readOnlyPostPrefixes`** : sans cette entree la garde d'ecriture du groupe
      joueur refusait en 401 une lecture que la meme personne obtenait en GET la veille
      (verifie par inversion). **Huma ne met PAS a plat une struct EMBARQUEE dans un corps**
      (il en fait une propriete a part, et le corps aplati part en 422 — piege mesure ici) :
      les deux champs de perimetre sont ecrits dans chacun des deux corps, la conversion
      vers le domaine restant unique (`scopeDepuis`). Contrat + `generated.ts` regeneres.
- [x] 4.6 **Barre L2 de l onglet** — `features/tactical/TacticalFilterBar.tsx` (152 L) +
      `tacticalScope.ts` (129 L) via `usePageScope`. AUCUN nouveau composant de filtre :
      `useLocalFilterBar` (periode/saison, experience, playlists, modes, `viewLabels` pour
      solo/escouade/mixte), `SessionMultiSelect` et `GamertagCombobox` — les trois existants,
      assembles dans UNE seule ligne collante (une seconde ligne aurait donne deux zones de
      filtres pour un seul scope).
      **`useLocalFilterBar` ETENDU plutot que recopie** (option `committed`) : l'etat
      committed est LEVE chez l'appelant quand il le demande, le pending restant interne.
      Le pending est un CALQUE sur le committed, pas une copie — donc un retour navigateur
      qui change l'URL remet aussi les pills, sans effet de synchronisation ; une copie
      initialisee au montage serait restee figee. Le hook expose aussi `sessionOptions` (la
      reponse de `/filters/resolve` qu'il interroge DEJA pour ses counts : deux requetes
      auraient donne deux comptes de sessions sur la meme page) et un `extras` en FONCTION
      (les controles ajoutes ont besoin de ces sessions, donc elles ne peuvent pas etre
      capturees avant l'appel). Les TROIS consommateurs existants (Citations, SessionDetail,
      SessionCompare/Synthesis) sont inchanges — 18 fichiers de tests verts.
      **Perimetre** : `contexteFiltre(scope)` (pur) -> `POST /filters/match-ids` ->
      `match_ids` postes a la grille. `filter_mode` suit la SELECTION (`sessions` des qu'une
      session est epinglee), comme `splitTemporalFiltered` cote Go.
      **La composition voyage en GAMERTAGS dans l'URL** (vocabulaire du selecteur, de la page
      Escouade et le seul lisible dans un lien) et se traduit en XUIDS a la requete, par la
      liste de coequipiers (`/pages/career/encounters` — celle-la inclut les amis,
      contrairement a `top-encounters` qui les exclut par construction ; MEME cle de cache
      que la page Carriere, donc jamais deux requetes pour une seule liste). **Un nom non
      traduisible ARRETE la lecture** au lieu d'etre ignore : l'ignorer ELARGIRAIT le
      perimetre et rendrait une grille plus fournie que demandee sans rien dire.
      `validateSearch` de la route declare tout le scope (un parametre non declare serait
      efface par le routeur au premier `navigate`). 14 cles i18n FR+EN de plus.
- [~] 4.6bis **« Sans composition, l'axe Escouade n'est pas propose » — la moitie CLIENT
      releve de la phase 5, GELEE.** Il n'existe aujourd'hui AUCUN selecteur d'axe a
      l'ecran : les axes sont ceux des RASTERS (`?question=` / `?qui=`), c'est-a-dire la vue
      d'analyse par carte, item 5.2. Poser ici un helper « axes disponibles » que personne
      n'appelle serait du code mort (CLAUDE.md n 7). La regle EST appliquee et testee cote
      Go — refus type `ErrTacticalEscouadeSansComposition`, 400 propre, aucune lecture de
      base declenchee (`service/tactical_service_perimetre_test.go`,
      `api/handlers/tactical_test.go`) — et la phase 5 lira ce refus pour griser l'axe.
- **REVUE ADVERSARIALE RONDE 1 — 17 constats, TOUS corriges** (2026-09-06, 3 commits
  `tactique(4.7)`). Aucun acces indu, aucune injection, aucun resultat faux cote Go ; le
  fil commun des constats retenus est ailleurs : **une frontiere juste peut rester fausse
  a l'USAGE.**
  - **W1 (P1) — « AUCUNE CARTE JOUEE » ETAIT AFFICHE PENDANT L'ATTENTE.** La grille est
    suspendue tant que le perimetre n'est pas resolu, et en TanStack v5 `isLoading` vaut
    `isPending && isFetching` — donc FAUX sur une requete desactivee. La page rendait donc
    l'etat VIDE au premier montage, a chaque clic sur Analyser (nouvelle cle, aucune
    donnee en cache) et DEFINITIVEMENT si `/filters/match-ids` echouait, l'echec etant en
    plus avale. La page lit desormais `isPending`/`isError` DES DEUX requetes, dans
    l'ordre composition impossible -> echec -> attente -> etat vide ; l'echec de
    resolution est journalise puis propage. `TacticalPage.tsx:79-100`, `queries.ts:52-72`.
  - **W2 (P1) — LE SELECTEUR PROPOSAIT CE QU'IL ALLAIT REFUSER.** Le combobox offre
    quatre sources (joueurs configures, coequipiers frequents, annuaire, repli Xbox) plus
    la saisie libre ; la page ne sait traduire en XUID que les coequipiers frequents.
    Choisir un nom sous « Autres joueurs » bloquait la grille sur « Coequipier
    introuvable ». `GamertagCombobox` gagne une prop `sources` (retro-compatible : omise,
    comportement complet — `SquadLayout`, Compare, Settings, Admin inchanges) et l'onglet
    n'offre que `['frequent']`, saisie libre coupee. Accepter puis refuser est toujours
    pire que ne pas proposer.
  - **W3 (P1) — LE LABEL DE SESSION ZOMBIE.** Un label porte son compte de matchs
    (« … (13) »), le backend filtre par EGALITE STRICTE, et le scope le persistait brut
    (URL + miroir) : deux matchs de plus a la synchronisation suivante et le lien partage
    rendait une grille vide sans rien dire. `reconcileSquadSessionLabels` /
    `stripSessionCountSuffix` DESCENDENT de `features/squad/squadPending.ts` vers
    `lib/sessions/sessionLabels.ts` — deux consommateurs, une definition — et la barre
    remappe le label des que la liste arrive, retire ce qui ne se remappe pas, et le DIT.
  - **W4 (P1) — « REINITIALISER LES FILTRES » N'EN REINITIALISAIT QUE LA MOITIE.** En
    mode controle, `onCommit` ne patche que les 5 champs du hook : sessions et composition
    survivaient, et `hasActiveFilters` les ignorait — une session seule et le bouton ↺
    n'etait meme pas rendu, pendant que la barre annoncait « aucun filtre actif » sur une
    lecture restreinte. `useLocalFilterBar` gagne `extrasActifs` et `onResetExtras`. La
    CARTE, elle, n'est PAS effacee : c'est une selection, pas un filtre.
  - **W5, W6 (P1) — DEUX INVARIANTS SANS TEST** : « resolution vide -> la requete PART
    quand meme » (la mutation `length ? ids : null` restait verte) et « la periode reglee
    arrive au scope » (intervertir debut/fin restait vert). Testes.
  - **G1 (P1, doc inversee)** — `server_apiv1.go:719` annoncait « GET » pour deux routes
    passees en POST, au SEUL endroit ou l'on voit la pile de middlewares ; il nomme
    desormais l'entree `readOnlyPostPrefixes` et dit ce qui reste applique (ownership,
    CSRF).
  - **G3 (P2 -> pris) — LA COMPOSITION N'ETAIT BORNEE PAR RIEN** : 5 000 chaines
    arbitraires -> 5 000 `EXISTS` correles -> 30 s puis 500, et le motif `^\d{1,32}$` de
    l'ancien contrat avait disparu avec les parametres plats. `domain.MaxCoequipiers` (3)
    + `domain.ValiderComposition` (motif xuid, meme que `handlers.xuidPattern`), refus
    TYPE en 400 `tactical_composition_invalid`, applique AVANT toute lecture et sur LES
    DEUX entrees. La validation REFUSE, elle ne nettoie pas.
  - **G4 (decision superviseur) — `matchs_filtres` REDEVIENT PAR CARTE** : les matchs du
    perimetre joues sur cette carte, dont `matchs_retenus` est un sous-ensemble. La
    version de 4.5 publiait la taille de la liste blanche, toutes cartes confondues : deux
    grandeurs sans denominateur commun sous des noms qui invitent a en faire un rapport.
    La ligne « Filtres » du §1 et l'item 4.5 disent desormais la bonne definition.
  - **P2 pris** : W7 (les deux tests d'etat vide etaient vacants), W8 (`getMatchCount`
    manquant — les sessions a 0 match sous le filtre etaient proposees avec leur compte
    fige), W9 (corps POST types par le contrat genere), W10 (`setCalque(null)` avant que
    le commit ne soit applique -> un cadre intermediaire sur l'ancien scope), W11
    (`TACTICAL_URL_KEYS` n'etait compare a rien : une cle manquante casse le partage
    d'URL en silence), W12 (assertion sur l'objet ENTIER), W13 (liste de consommateurs
    fausse en tete de `useLocalFilterBar`), G2 (commentaire d'inversion citant une
    fonction supprimee).
- **REVUE ADVERSARIALE RONDE 2 — 3 constats, TOUS corriges** (2026-09-06, commit
  `tactique(4.8)`). Derniere salve : W1, W2, W4-W6, W8-W11, G1, G2 et G4 de la ronde 1
  tiennent, et les composants partages ont ete verifies consommateur par consommateur
  (combobox bit-a-bit identique sans `sources`, helpers de session byte-identiques, hook
  inchange en mode non controle).
  - **P1 — LA COPIE AVAIT PERDU LA GARDE DE L'ORIGINAL, ET C'EST UN ELARGISSEMENT
    SILENCIEUX.** Session SOLO epinglee, puis un coequipier ajoute : `avecComposition`
    passe a vrai, la liste proposee bascule sur les sessions d'ESCOUADE, la reconciliation
    ne retrouve plus le label solo et rendait `[]` — l'effet ecrivait alors
    `setScope({ sessions: [] })`, `filter_mode` retombait en `period` SANS DATES, et la
    lecture passait d'une soiree a l'HISTORIQUE ENTIER pour seul signal un `console.warn`.
    `SquadLayout.tsx:452` porte exactement la garde qui manquait
    (`if (reconciled.length === 0) return`, « si TOUS les labels sont des zombies pour la
    composition courante, on ne fait rien ») : elle est reprise telle quelle. Le remappage
    `(4)` -> `(6)` et le retrait d'un zombie INDIVIDUEL restent.
    **ET LA SITUATION SE VOIT** : `sessionsHorsListe` (pur, meme notion d'identite que la
    reconciliation) alimente une note sous la barre — gabarit du depot (`role="status"`,
    tint `warning`, `text-warning`), cle i18n FR+EN — qui dit ce qui est vrai : ces
    sessions ne sont pas dans la liste courante, ET le filtre reste applique.
  - **P2 — LE MOTIF XUID EXISTAIT EN DEUX EXEMPLAIRES.** `domain.motifXUID` et
    `handlers.xuidPattern` gardaient deux frontieres (composition tactique,
    `with_player` de l'Explorateur) sous deux definitions, alors que le contrat de la
    ronde 1 disait REUTILISER. Source unique `domain.XUIDValide` ; `handlers.xuidPattern`
    SUPPRIMEE, son appelant migre ; garde-rail
    `archlint/no_local_xuid_pattern_test.go` (patron des `no_local_*`) interdisant toute
    autre compilation du motif dans `internal/`, y compris ecrite `[0-9]{1,32}`.
    Le ratchet porte un SELF-CHECK POSITIF — il doit reconnaitre la source unique
    elle-meme — et c'est lui qui a fait tomber la premiere version du garde-rail, dont le
    motif de detection etait faux : elle ne detectait plus rien, y compris ce qu'elle
    devait interdire.
  - **P2 — CE QU'ON NE PROPOSE PAS, ON NE L'INTERROGEAIT QUAND MEME PAS.** Le filtrage
    par `sources` s'appliquait au RESULTAT : avec `sources={['frequent']}`, chaque frappe
    de deux caracteres partait vers `/directory/gamertags/search` pour une reponse jetee.
    `useGamertagSuggestions` gagne `rechercheDistante` (defaut : vrai — les cinq
    consommateurs existants sont bit-a-bit identiques), desarmant la requete ET son repli
    Xbox. Le test PROUVE UN NON-EVENEMENT en montant les deux comboboxes cote a cote :
    l'arrivee de la requete du combobox libre etablit que la fenetre de debounce est
    passee pour les deux.
- **Gate REJOUE apres la ronde 2, le 2026-09-06** (avant-plan, en serie). Go : `gofmt`
  propre ; `go vet` propre ; `go test -count=1 ./internal/domain/... ./internal/api/...
  ./internal/archlint/... ./internal/service/...` : **15 paquets `ok`, aucun `FAIL`**
  (api 23,9 s ; service 10,6 s ; handlers 9,7 s ; archlint 9,9 s ; title 11,2 s) ;
  `golangci-lint run --new-from-merge-base=origin/main` : **0 issue** ;
  `openapi-gen -check` : a jour. Web : `typecheck` propre ; `lint` **0 erreur** (30
  warnings preexistants) ; **suite vitest COMPLETE : 606 fichiers, 6405 tests passes,
  14 skip, 0 fail** ; couleurs 0 violation ; imports croises **7/7 inchange** ; garde
  anti-anglicismes + fraicheur des types generes verts ; manifestes regeneres.
  **TROIS INVERSIONS** : garde `length === 0` retiree -> le test de l'elargissement
  silencieux tombe ; seconde definition du motif reintroduite dans `handlers` -> le
  garde-rail tombe en la nommant (`api/handlers/match_view.go:32`) ; requete distante
  reactivee -> le test du non-evenement tombe.
- **Gate REJOUE apres la ronde 1, le 2026-09-06** (avant-plan, en serie). Go : `gofmt`
  propre, `go vet` propre sur les 8 arbres, `go test -count=1` **sans aucun `FAIL`**,
  `golangci-lint run --new-from-merge-base=origin/main` : **0 issue**, `openapi-gen -check`
  a jour et `generated.ts` INCHANGE (les 400 ne sont declares nulle part au contrat).
  Web : `typecheck` propre ; `lint` **0 erreur** (30 warnings preexistants) ; **suite
  vitest COMPLETE** : **606 fichiers, 6402 tests passes, 14 skip, 0 fail** (93,4 s) ;
  couleurs 0 violation ; imports croises **7/7 inchange** ; garde anti-anglicismes verte ;
  manifestes sans diff residuel.
  **SEPT INVERSIONS JOUEES** : `isLoading` remis a la place d'`isPending` -> 5 tests de la
  page tombent ; reconciliation neutralisee -> les 2 tests de label zombie tombent ;
  `onResetExtras`/`extrasActifs` neutralises -> les 2 tests de ↺ tombent ; periode
  intervertie -> le test W6 tombe ; borne de composition retiree -> le test des 4
  coequipiers tombe ; motif xuid retire -> les 4 formes hostiles passent ;
  `matchs_filtres` remis a la liste blanche -> 20 au lieu de 8.
- **Gate PASSE le 2026-09-06** (avant-plan, une commande `go` a la fois,
  `GOCACHE=...\go-build-tactique`, `CGO_ENABLED=1`). Go : `go vet` propre sur les 8 arbres
  (10,4 s) ; `go test -count=1` sur `./internal/service/... ./internal/api/...
  ./internal/domain/... ./internal/port/... ./internal/platform/duckdb/...
  ./internal/archlint/... ./contracttest/... ./cmd/...` : **aucun `FAIL`**, tous paquets
  `ok` (duckdb 43,4 s ; api 23,3 s ; service 9,4 s ; handlers 8,5 s ; cmd/server 7,6 s ;
  contracttest 0,6 s) ; `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` :
  **0 issue** (13 `gofmt` corriges avant) ; `openapi-gen -check` : a jour.
  Web (depuis `apps/web`, `node_modules/.tmp` purge) : `typecheck` propre ; `lint` **0
  erreur** (30 warnings preexistants) ; **suite vitest COMPLETE** `vitest run --pool=forks` :
  **606 fichiers, 6390 tests passes, 14 skip, 0 fail** (86,5 s) ;
  `lint-no-hardcoded-colors` 0 violation ; garde anti-anglicismes vert ;
  `lint-cross-feature-imports` **7/7 INCHANGE** (la feature n'importe que `@/lib`,
  `@/components` et `features/_shared`) ; manifestes regeneres sans diff residuel ;
  `routeTree.gen.ts` NON touche (aucune route ajoutee).
  **HUIT INVERSIONS JOUEES.** Go : liste vide traitee comme « aucune restriction » -> le
  test des trois lectures tombe (2 matchs / 2 points au lieu de 0) ; `EXISTS` sans
  `c.team_id = mp.team_id` -> le match ou l'ami jouait EN FACE entre dans l'univers ; axe
  escouade rendu a « meme equipe que moi » -> les deux coequipiers hors composition se
  peignent ; `/tactical/` retire de `readOnlyPostPrefixes` -> les quatre cas de la garde
  d'ecriture tombent. Web : `filter_mode` fige a `period` -> le test de session tombe (page
  ET logique pure) ; `sessionsProposees` sans predicat -> les quatre tests de composition
  tombent (barre ET logique) ; nom introuvable IGNORE -> les trois tests de resolution
  tombent ; plafond de composition et `vue=all` ecrits dans l'URL -> les deux tests de
  scope tombent.
- **Gate** : Go vet + test (service, api, duckdb, archlint, contracttest) ; openapi-gen -check ;
  typecheck, lint, vitest `tactical`, couleurs, imports croises (plafond 7/7 : les helpers
  d Explorateur/Escouade necessaires descendent dans `lib/` ou `features/_shared/`), manifestes.

### Phase 5 — Vue d'analyse, lectures SQL, drilldown — GELEE JUSQU'AU LOT D
- [ ] 5.1 Peintre partage dans `lib/replay/` (selon D.13), extrait de `heatmapLayer.ts`,
      garde-rail grep contre une seconde implementation du noyau
- [ ] 5.2 `tacticalScope.ts` (usePageScope) ; barre d'outils ; titre « Plan de <carte> — <question> »
- [ ] 5.3 `KPIStrip` : matchs retenus, couverture, morts isolees, echange « sur cette carte »
- [ ] 5.4 `SectionCard` « Plan » : canvas + legende ; `footer` = unite, planchers, source
- [ ] 5.5 `?frame=` sur la route du rejeu, selon le modele D.6/D.8 (playbackStore)
- [ ] 5.6 `SectionCard` « Cellule selectionnee » : contributeurs, lien `?frame=` ; **liste
      filtree par l'ownership XUID** (ADR 0029) ; `footer` = matchs comptes non ouvrables
- **Gate** : typecheck + vitest ; test pur de `tacticalLogic.ts` ; smoke canvas ; garde-rail
  du peintre ; filtrage d'acces ; `?frame=` positionne le rejeu ; couleurs.

### Phase 6 — Rasters par match a la cuisson + rattrapage — CLOSE 2026-09-06, revue ronde 1 SOLDEE (14 constats)
- [x] 6.0 `analysis/tactical/tracks.go` (293 L, PUR — avance de la phase 7) :
      `Occupation(g, EntreeOccupation, pasMs)` reechantillonne les pistes a pas FIXE de
      250 ms en tenant la derniere position connue, sur une fenetre DEMI-OUVERTE.
      **POURQUOI PAS LES POINTS BRUTS** : le film ne replique une position que lorsqu'elle
      change assez — un joueur immobile derriere un mur produit deux points en quinze
      secondes pendant qu'un joueur qui court en produit un par frame. Compter les points
      mesurerait le MOUVEMENT, pas le temps passe. **DEMI-OUVERTE** : 2 s de presence font
      huit quarts de seconde, pas neuf ; compter la borne haute ajouterait un echantillon
      par vie, soit une demi-minute inventee sur un match a cent vies.
      Rendu par joueur NOMME : echantillons (`[]domain.PositionSample`), premier point de
      chaque vie (le spawn), premiere entree dans chaque cellule (l'instant contributeur).
      Vie sans xuid ECARTEE — la rattacher par son slot prendrait un ordre pour une
      identite. `IntervalleFrameMs`/`pasMs` <= 0 : sortie vide, jamais une cadence devinee
      (l'appelant resout `replay.DefaultFrameIntervalMS`). Onze tests a trajectoires posees
      a la main, comptes EXACTS.
- [x] 6.1 `PathResolver.TacticalRasterDir` / `TacticalRasterPath`
      (`domain/title/registry.go:763-793`) : `data/cache/replays/{slug}/rasters/{short}.json`.
      **LE SOUS-DOSSIER N'EST PAS UNE PREFERENCE DE RANGEMENT** : les deux parcours du
      dossier d'artefacts (`service.replayService.AvailableSet`,
      `scheduler.purgeReplayArtifactsForTitle`) ne comptent QUE les `.json` de premier
      niveau et sautent les repertoires. A plat, un sidecar aurait ete lu comme l'artefact
      d'un match inexistant par le premier, et supprime comme un artefact indatable par le
      second. Meme cle courte que l'artefact (`FilmShortMatchID`).
- [x] 6.2 `sync/replayartifacts/raster.go` (252 L) : QUATRIEME projection post-cuisson,
      apres le T0 du film, le resume d'usage et les stats d'Assaut — memes artefacts
      ranges, meme place (apres toute cuisson), meme regime best-effort. **LA SEULE QUI
      N'ECRIT AUCUNE BASE** : son resultat est un fichier a cote de son artefact, donc
      aucun writer, aucun lease, aucune regle ART. Ecriture par `platform/atomicfile`,
      compteurs `postsync_replay_rasters_{ecrits,echecs}_total`.
      Contrat du sidecar : `domain/tactical_raster.go` (129 L, `schema_version` propre,
      tags snake_case, `pas_m` / `frame_interval_ms` / `pas_echantillon_ms` — cette
      derniere est L'UNITE de `echantillons`, verifiee a la lecture).
      **`Raster.CellulesBrutes()`** (`analysis/tactical/raster.go`) : la forme qu'on
      STOCKE. `Cellules()` applique le plancher de rarete (trois matchs distincts), qui sur
      un raster d'UN match vide tout par construction — LE PLANCHER APPARTIENT A L'AGREGAT,
      JAMAIS AU MATCH.
      **LE GATE EST CELUI DE L'ETAPE** (`film.replay_artifact`, deja franchi en tete de
      `Run`) : sans la cle rien n'est cuit, donc rien n'est projete. Le relire ferait deux
      sources de verite pour une question, plus un TOML par cycle — contrairement aux gates
      jumeaux d'`usage.go` et `bombstats.go`, qui portent des cles DIFFERENTES de celle de
      l'etape. Sept tests sans base ; artefact de schema 20 projete ; artefact sans piste
      nommee -> sidecar VIDE mais PRESENT, serialise `[]` et jamais `null`.
- [x] 6.3 `cmd/levelup/cmd_tactical_rasters.go` (235 L) : `levelup tactical-rasters
      --backfill [--dry-run] [--title] [--limit]`. **ELLE N'OUVRE AUCUNE BASE, PAS MEME EN
      LECTURE** — le sidecar est par match et anonyme, il n'y a rien a demander a DuckDB.
      Enumeration par LE lecteur d'artefacts du depot (`ReplayService.AvailableSet`), qui
      ne compte que les `.json` de premier niveau : le sous-dossier qu'elle remplit est
      exclu de sa propre enumeration, par construction (teste).
      Idempotente, cle de fraicheur DOUBLE (format du sidecar + schema de l'artefact) ;
      raccourci qui evite d'ouvrir l'artefact quand le sidecar declare deja
      `replay.SchemaVersion` — un artefact ne peut pas porter un schema superieur a ce que
      le binaire courant sait produire. Pas de mode par defaut : sans `--backfill`, refus.
      Gate par capability. **RATCHET `archlint/no_cuisson_depuis_tactique_test.go`** : ni
      ce rattrapage ni `service/tactical*` ne peuvent nommer `replaybuild`,
      `BuildFromFilm` ou `filmcache` — self-check PAR CIBLE (un simple compte aurait masque
      la disparition d'une cible, `internal/service/tactical` en matchant plusieurs).
      Sept tests sur repertoire temporaire. `docs/COMMANDS.md` + `docs/FR/COMMANDS.md`
      (bilingue, meme commit — CLAUDE.md n 15).
- [x] 6.4 `service/tactical_service_rasters.go` (245 L) : la question `temps` sur la meme
      route `POST /tactical/{map_id}/raster`, portee data-level `film.replay_artifact`.
      **PORT `TacticalRepository.Univers`** (`port/tactical.go`,
      `platform/duckdb/tactical_repo_univers.go`) : l'occupation n'a besoin de la base que
      pour son univers — passer par `KillPositions` aurait scanne les positions de toute la
      carte pour jeter le resultat. C'est le MEME `chargerUnivers`, expose : deux
      definitions auraient mesure sur deux populations sous le meme nom de filtre (teste
      par egalite profonde entre les deux lectures).
      **PORT `TacticalRasterStore`** + `service.NewTacticalRasterStore` (la source est un
      FICHIER, pas une base : le service reste testable sans disque), cable au seul endroit
      de construction (`registry_pages.go`). **UN SIDECAR ABSENT EST UN MATCH NON MESURE**
      — il compte dans `matchs_filtres`, pas dans `matchs_retenus` : meme regle que le
      drapeau `Mesure` (correction G2), appliquee a l'autre substrat.
      **`tactical.RasteriseComptes`** somme des comptes DEJA agreges plutot que de
      re-fabriquer un point par echantillon (des centaines de milliers de structures par
      affichage) ; **`tactical.EnSecondes`** convertit la VALEUR (secondes par match) en
      laissant le BRUT en echantillons — c'est la mesure, la seconde n'en est que l'unite.
      L'echelle est calculee APRES la conversion, sur les valeurs qui seront peintes.
      **`facesDeLaQuestion` gagne le cas `temps` -> AUCUNE face** : sans lui la question
      serait tombee dans la branche par defaut et aurait compte les deux faces comme
      « ou je gagne », publiant un denominateur de couverture qui ne decrit pas la mesure
      affichee. Onze tests service (mock du port sidecar), deux tests handler, contrat +
      `generated.ts` regeneres. `tactical_service_test.go` avait franchi 500 L : le double
      du port descend dans `tactical_mock_test.go` (meme discipline que les scissions des
      phases 1, 2 et 4 bis).
- [x] 6.5 **Revue adversariale ronde 1 — 5 constats P1, 8 P2, 1 decision produit : TOUS
      corriges** en 4 commits `tactique(6.5.<n>)`. Aucun report.
      **C1 (P1) — LE TEMPS EN VEHICULE ETAIT PERDU EN SILENCE SUR UN MATCH « MESURE ».**
      La cuisson coupe une piste en nouvelle vie des qu'un trou depasse 5 s
      (`replay.lifeGapUS`) ; or un occupant embarque CESSE de repliquer son bipede — ce sont
      ses trous qui portent les episodes — et ces episodes durent 13 a 36 s en mediane.
      `vie()` ne tenait donc jamais une position d'un morceau au suivant. L'en-tete
      justifiait en plus le reechantillonnage par « un joueur immobile quinze secondes »,
      cas INATTEIGNABLE par construction : doc inversee.
      Lien verifie sur pieces AVANT de coder : `VehicleRide` est IMBRIQUE dans
      `VehicleTrack.Rides`, a cote de `VehicleTrack.Samples` — aucun appariement a faire.
      `EntreeOccupation.Embarquements` (pur) est rempli par l'adaptateur unique de
      `replayartifacts/raster.go`. Regles : un episode ne cree JAMAIS de spawn ; un
      echantillon dans la fenetre prend la position du vehicule, le point de bipede est
      ignore ; un episode sans point de vehicule n'attribue RIEN ; deux episodes
      chevauchants d'un MEME xuid (ambiguite conducteur/passager que le document publie) ne
      sont pas sommes. `TacticalRasterSchemaVersion` 1 -> 2 (la formule change). La LACUNE
      residuelle est ECRITE dans `domain/tactical_raster.go` : la primitive n'attribue que
      15,6 a 21,1 % des vies de vehicule.
      **C2 (P1) — LE RATCHET ANTI-CUISSON NE FERMAIT PAS LA PORTE QU'IL CROYAIT FERMER.**
      La CLI importe deja `replayartifacts` (legitimement, pour deux projections pures), et
      ce paquet exporte `SpawnBuildOne` et `Run` : remplacer une branche d'echec par
      `replayartifacts.SpawnBuildOne(...)` laissait le ratchet VERT en faisant de la passe
      une cuisson EN LOT. Quatre motifs ajoutes + SELF-CHECK PAR MOTIF (un vocabulaire mort
      n'interdit rien).
      **C3 (P1) — LE REMEDE PRESCRIT ETAIT UN NO-OP SUR LES SIDECARS QU'IL VISAIT.** Le
      service ecarte un sidecar aux mauvaises unites en prescrivant `--backfill` ; la CLI ne
      regardait que le schema. `domain.SidecarRasterCourant` est LE predicat des deux, plus
      un garde-rail BORNE aux fichiers qui nomment `TacticalRasterSidecar` (le motif nu
      remontait cinq comparaisons de schema legitimes) et un test de non-divergence des
      unites chez le paquet PROPRIETAIRE des constantes.
      **C4 (P1)** — l'echange sous `temps` etait une decision jamais prouvee (les huit cas
      montaient un titre sans `film.kill_source`) : teste, avec sa moitie « couverture
      d'evenements a zero ».
      **C5 (P1)** — la ligne de cablage de `Run` n'etait traversee par rien : test
      `-tags=integration` qui pilote `Run` de bout en bout via le seam `Deps.BuildOne`,
      sans decoder un film ni ecrire hors d'un repertoire temporaire.
      **C6** logs sur les deux erreurs jetees de `lireSidecarRaster` · **C7** troisieme
      copie du motif `os.ReadFile`+`Unmarshal(&ReplayDocument)` -> `lireDocumentRange` +
      ratchet · **C8** `Raster()` 83 -> 25 L (`rasterDeKills` extraite ; `funlen` ne le
      voyait pas, la position d'une fonction n'est pas une ligne ajoutee) et
      `registry.go` ramene de 987 a ses 957 L par `registry_tactical.go` · **C9**
      `points_ignores` transporte par le sidecar · **C10** quatre tests de mutations qui
      passaient · **C11** pointeur de doc corrige (`no_second_artifact_sink_test` compte le
      puits d'ECRITURE, pas l'enumeration).
      **C12 (decision produit du superviseur)** — un artefact purge EMPORTE son sidecar, et
      le court-circuit « dossier vide » de la purge compte desormais les ARTEFACTS et non
      les entrees (avec `rasters/` present, chaque tick ouvrait la shared pour rien).
- **Gate REJOUE apres la ronde 1, le 2026-09-06** (avant-plan, en serie, code de sortie
  verifie). Go : `gofmt` propre ; `go vet` propre sur les 11 arbres ; `go test -count=1`
  sur les dix arbres dont `./cmd/...` : **aucun `FAIL`, code 0** ;
  **`go test -count=1 -tags=integration -p 1 ./internal/sync/replayartifacts/...
  ./internal/persist/... ./internal/scheduler/...` : `ok`/`ok`/`ok`, CODE 0** (12,4 s /
  47,7 s / 39,0 s) ; `golangci-lint run --new-from-merge-base=origin/main` : **0 issue** ;
  `golangci-lint --enable funlen` sur les six arbres touches : **aucun fichier du lot
  signale** (la dette remontee est preexistante et gelee) ; `openapi-gen -check` a jour.
  Web : `typecheck` propre ; **suite vitest COMPLETE 606 fichiers / 6405 tests / 14 skip /
  0 fail** (106,8 s). `git status` : 0 fichier sous `data/`.
  **VINGT MUTATIONS JOUEES, toutes mordantes.** C1 : voie des embarquements supprimee -> 4
  tests tombent. C10 : `frame := p.T` -> la frame d'entree tombe ; vie a un point ecartee ->
  le spawn disparait. C3 : cle de fraicheur ramenee au schema -> les deux cas d'unites
  tombent. C2 : `SpawnBuildOne` reintroduit -> nomme a la ligne ; motif renomme -> le
  self-check tombe. C6 : sidecar jamais relu -> l'idempotence tombe. C4 : `lireLeJournal`
  supprime -> l'echange disparait. C9 : transport retire -> 0 au lieu de 8. C10 : sidecar
  vide traite en non mesure -> 3 au lieu de 4. C5 : ligne de cablage retiree -> le sidecar
  ne nait pas. C7 : quatrieme copie -> nommee a la ligne. C12 : sidecar non supprime ->
  orphelin ; court-circuit sur les entrees -> la base est ouverte pour rien ; suppression
  etendue aux artefacts gardes -> le test le nomme.
- **Gate PASSE le 2026-09-06** (avant-plan, une commande `go` a la fois,
  `GOCACHE=...go-build-tactique`, `CGO_ENABLED=1`). Go : `gofmt` propre ; `go vet` propre
  sur les 10 arbres (3,1 s) ; `go test -count=1` sur `./internal/analysis/tactical/...
  ./internal/sync/replayartifacts/... ./internal/service/... ./internal/api/...
  ./internal/domain/... ./internal/port/... ./internal/platform/... ./internal/archlint/...
  ./contracttest/... ./cmd/...` : **aucun `FAIL`**, code de sortie 0 (duckdb 63,9 s ;
  api 24,4 s ; service 14,9 s ; handlers 14,0 s ; archlint 12,6 s ; contracttest 0,7 s) ;
  **`go test -tags=integration -p 1` sur `./internal/sync/replayartifacts/...
  ./internal/persist/...` : `ok` / `ok`, CODE DE SORTIE 0** (14,1 s et 40,4 s — le diff
  touche `internal/sync/`, cette passe est obligatoire) ;
  `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` : **0 issue** ;
  `openapi-gen -check` : a jour ; `generated.ts` regenere (2 lignes, additions pures).
  Web (`node_modules/.tmp` purge) : `typecheck` propre ; `lint` **0 erreur** (30 warnings
  preexistants) ; **suite vitest COMPLETE** `vitest run --pool=forks` : **606 fichiers,
  6405 tests passes, 14 skip, 0 fail** (134,3 s) ; `routeTree.gen.ts` NON touche.
  Seuils : fichiers neufs <= 367 L, fonctions <= 45 L, <= 5 parametres ;
  `tactical_service_test.go` ramene de 510 a 471 L par extraction du mock.
  **ONZE INVERSIONS JOUEES.** 6.0/6.2 : `Cellules()` a la place de `CellulesBrutes` -> le
  sidecar est vide ; fenetre fermee (`tMs <= finMs`) -> 9 echantillons au lieu de 8 ;
  `Joueurs` en `nil` -> serialise `null` ; echelle de temps par defaut retiree -> le test
  du schema 20 tombe. 6.3 : cle de fraicheur inversee -> le sidecar perime est saute ;
  sidecar jamais relu -> l'idempotence tombe (2 ecritures a la seconde passe) ; cuisson
  reintroduite -> le ratchet la nomme en `cmd_tactical_rasters.go:108` ; cible du ratchet
  renommee -> le self-check tombe. 6.4 : denominateur mis a tous les matchs filtres -> 4
  au lieu de 3 ; conversion en secondes retiree -> 6 s au lieu de 1,5 ; garde d'unites
  retiree -> un sidecar a 1000 ms de pas est somme ; `facesDeLaQuestion` sans le cas
  `temps` -> la couverture publie 0/1 au lieu de 0/0.
- **Gate** : projection sur fixture (comptes exacts) ; schema ancien (v20) projete ;
  idempotence ; aucun chemin de la page n'ecrit ni ne cuit ; `no_second_artifact_sink_test`.

### Phase 7 — Occupation, spawns, routes, isolement — APRES LA 6 (le lot lourd)
- [ ] 7.1 `tracks.go` (250 ms, position tenue) ; 7.2 `spawn.go` (composantes connexes,
      premieres vies, callouts) ; 7.3 table du rayon de radar + chargeur ; 7.4 `isolation.go`
      (rayon en parametre, tous-morts exclu) ; 7.5 `dispersion.go` ; 7.6 filtre spawn de
      depart, lectures routes / isole / temps / gagne ; 7.7 Escouade Synergies : nuage
      isolement x couverture (quadrants nommes, sessions, taille = morts)
- **Gate** : trajectoires posees a la main ; rayon par match (18/24 dans le meme filtre) ;
  variante absente -> pas de lecture ; tous-morts exclu ; premiere vie seule ; session < 5
  morts exclue ; aucune cuisson.

### Phase 8 — Cloture
- [ ] 8.1 `.ai/thought_log.md` ; 8.2 `REGISTRE_REPORTS.md` si report ; 8.3 `make gate-push`
      puis CI verte au niveau JOB ; 8.4 revue adversariale du diff integral avant merge

## 5. Prepare pour l'ouverture au public (sans rien exposer en V1)
Raster anonyme ; drilldown = frontiere (ownership XUID) ; sidecars par match, pas par joueur
(« Tout le monde » = sommer plus de sidecars) ; plancher par cellule deja la.

## 6. Journal
- 2026-09-06 : **revue adversariale ronde 1 de la phase 6 — 14 constats, TOUS corriges** en 4 commits `tactique(6.5.<n>)`. Le P1 qui compte est un defaut de MESURE, pas de code : **le temps passe en vehicule n'entrait nulle part, sur des matchs comptes comme MESURES**. La cuisson coupe une vie des qu'un trou depasse 5 s ; un occupant embarque cesse de repliquer son bipede, et ces episodes durent 13 a 36 s en mediane — le reechantillonnage ne pouvait donc structurellement pas les voir, et l'en-tete justifiait le mecanisme par un cas (« immobile quinze secondes ») que la coupe des vies rend INATTEIGNABLE : doc inversee sur la lacune meme qu'elle masquait. Corrige en ATTRIBUANT sans inventer — pendant un episode l'occupant est a la position du vehicule, un episode sans point de vehicule n'attribue rien, et un embarquement ne cree jamais de spawn. Schema du sidecar 1 -> 2, lacune residuelle (15,6 a 21,1 % des vies attribuees) ECRITE dans le contrat. Les quatre autres P1 disent la meme chose autrement : **un garde qui ne garde pas ce qu'il croit**. Le ratchet anti-cuisson ignorait `SpawnBuildOne`, que la CLI pouvait deja appeler ; le remede que le service PRESCRIT dans son avertissement etait un no-op sur exactement les sidecars qu'il ecartait ; l'echange sous `temps` etait une decision jamais prouvee ; et la ligne de cablage qui fait naitre les sidecars a la cuisson n'etait traversee par aucun test. Une mesure a corrige une croyance au passage : `points_ignores` est structurellement NUL aujourd'hui, `replay.Point` etant en float32 et JSON ne pouvant exprimer aucune valeur non finie — le fait est fige par un test plutot que redecouvert comme un bug. Decision produit prise : un artefact purge emporte son sidecar. Gate complet rejoue (integration `-p 1` sur trois arbres, code 0 ; vitest complet 606/6405/0), VINGT mutations jouees.
- 2026-09-06 : **phase 6 CLOSE — les rasters d'occupation sont cuits UNE FOIS, la page n'en somme que des fichiers.** Un sidecar par match (`rasters/{short}.json`), depose par la quatrieme projection post-cuisson et rattrape hors ligne par `levelup tactical-rasters --backfill`. Quatre proprietes portent tout le lot. (1) **UN ECHANTILLON = 250 ms DE PRESENCE, PAS UN POINT DE FILM** : le film ne replique une position que lorsqu'elle change assez, donc compter les points bruts aurait mesure le mouvement et non le temps passe ; la fenetre est demi-ouverte, 2 s font huit quarts de seconde. (2) **LE PLANCHER DE RARETE APPARTIENT A L'AGREGAT** : ecrit avec `Cellules()`, un sidecar de match aurait ete vide par construction (une cellule d'un match compte un match distinct) — d'ou `CellulesBrutes()`, la forme qu'on stocke. (3) **UN SIDECAR ABSENT EST UN MATCH NON MESURE**, pas un match a zero : meme regle que le drapeau `Mesure` de la correction G2, appliquee a l'autre substrat, sans quoi l'intensite aurait varie avec la couverture de film au lieu du jeu. (4) **RIEN NE CUIT, ET C'EST GARDE PAR RATCHET** : ni la page ni le rattrapage ne peuvent nommer `replaybuild`/`BuildFromFilm`/`filmcache`, avec self-check par cible. Deux pieges rencontres et fermes sur place : le sous-dossier `rasters/` n'est pas un choix de rangement mais la condition de cohabitation (les deux parcours du dossier d'artefacts ne comptent que les `.json` de premier niveau — a plat, un sidecar aurait ete lu comme un match par `AvailableSet` et supprime par la purge) ; et deux tests existants employaient « temps » comme exemple de question INCONNUE, ce que l'ouverture du vocabulaire a invalide. Gate complet vert, integration `-p 1` comprise (code de sortie 0), suite vitest complete (606 fichiers / 6405 tests / 0 fail), ONZE inversions jouees.
- 2026-09-06 : **revue adversariale ronde 2 de la phase 4 bis — 3 constats, TOUS corriges** en 1 commit `tactique(4.8)` ; pas de ronde 3. Le P1 est le meme defaut que la ronde 1 avait deja nomme, une couche plus bas : **une copie qui perd la garde de l'original**. La reconciliation des labels de session avait ete reprise de `SquadLayout` SANS son `if (reconciled.length === 0) return` — et cette garde-la ne protege pas d'un zombie, elle protege d'un CHANGEMENT DE CONTEXTE : ajouter un coequipier bascule la liste proposee de « solo » a « escouade », la session epinglee n'y figure plus, et l'ecrire a vide faisait passer la lecture d'une soiree a l'historique entier, en silence. Garde reprise, et la situation est desormais AFFICHEE plutot que muette. Les deux P2 ferment ce que la ronde 1 avait laisse a moitie : une seule definition du motif XUID (+ garde-rail archlint, dont le self-check positif a rattrape un ratchet faux) et une requete d'annuaire qui n'est plus emise quand sa source n'est pas proposee. Gate complet rejoue, trois inversions.
- 2026-09-06 : **revue adversariale ronde 1 de la phase 4 bis — 17 constats, TOUS corriges** en 3 commits `tactique(4.7)`. Aucun acces indu, aucune injection, aucun resultat faux cote Go : les defauts etaient d'USAGE. Le plus couteux (W1) : « Aucune carte jouee » s'affichait PENDANT l'attente, parce qu'en TanStack v5 `isLoading` est faux sur une requete desactivee — au premier montage, a chaque clic sur Analyser, et definitivement si la resolution echouait. Les trois autres P1 web disent la meme chose autrement : le selecteur proposait des noms qu'il allait refuser (W2), le label de session persiste devenait un zombie a la synchronisation suivante (W3), et « Reinitialiser les filtres » n'en reinitialisait que la moitie (W4). Cote Go : composition non bornee (5 000 `EXISTS` possibles) desormais bornee et typee, `matchs_filtres` redevenu PAR CARTE sur decision du superviseur, et un commentaire de montage qui annoncait « GET » sur des routes en POST. Gate rejoue en entier, sept inversions.
- 2026-09-06 : **phase 4 bis CLOSE (items 4.5 et 4.6) — le filtre de session MARCHE.** Le perimetre de l'onglet est desormais une LISTE BLANCHE de match_id, resolue cote client par le endpoint de filtres (base JOUEUR, la seule qui porte les sessions) et postee aux deux endpoints tactiques, qui perdent leurs parametres plats de filtre. Barre L2 assemblee de trois controles existants, etat dans l'URL. Gate complet vert (Go : 0 issue lint, contrat a jour ; web : suite vitest COMPLETE, 606 fichiers / 6390 tests / 0 fail), HUIT inversions jouees. Deux pieges rencontres et documentes sur place : Huma ne met pas a plat une struct embarquee dans un corps (422 silencieux), et un POST de lecture sous /players/ doit entrer dans `readOnlyPostPrefixes` sous peine de 401 pour un visiteur anonyme.
- 2026-09-06 : **second rouge du meme run CI, cote Go cette fois (job « Go Coverage + Baseline », qui joue `./...` COMPLET, donc aussi `cmd/**`).** Le garde-rail de source `TestInventaireEtProductionResolventPareil` (`apps/go-api/cmd/mapfond-inventaire/resolution_test.go`) lisait le corps de `resolveBackgroundKey` — devenue une simple enveloppe (`replay_map_background.go:126`) apres la phase 4, qui a deplace la cascade et ses deux `return <ident>, nil` dans `resolveBackgroundKeyDepuis` (`:151`). Le garde comptait donc 0 chemin de resolution dans une fonction vide au lieu de 2. Corrige en faisant lire au garde `resolveBackgroundKeyDepuis` (la fonction qui PORTE la cascade), commentaires mis a jour pour nommer les deux enveloppes qui l'appellent ; commentaire ajoute a `resoutFond` (`main.go:169`) sur `cleDeFondSure` (sans objet ici : les map_id de l'inventaire viennent de `match_registry`, jamais d'une entree hostile). Inversion jouee : remise temporaire de `resolveBackgroundKey(` dans le garde -> memes deux echecs qu'en CI (0 chemin, -1/-1) ; restaure. Gate vert : `./cmd/mapfond-inventaire/`, `./cmd/... ./internal/service/...` (aucun `--- FAIL`), `go vet`. Lecon : **le job coverage joue `./...` dont `cmd/`** — un gate local scope a `internal/` ne l'aurait pas revele.
- 2026-09-06 : **CI Frontend rouge (run 34031528843) : titre de route manquant, garde-rail
  `pageTitle.test.ts`.** La 5e route Ascension `/ascension/tactique` (commit `tactique(4.6)`)
  n'avait pas d'entree dans `PLAYER_SUFFIX_OVERRIDES` (`apps/web/src/lib/pageTitle.ts`) :
  le garde-rail I18 exige un titre FR+EN non-fallback pour CHAQUE route reelle balayee sous
  `src/routes/**`. Ajoute `{ pattern: '/ascension/tactique', title: { fr: 'Ascension —
  Tactique', en: 'Ascension — Tactics' } }` sur le meme patron que les 3 voisins
  (objectifs/coaching/realisations). Lecon : **la CI joue TOUTE la suite vitest** — un
  `vitest run` filtre sur un seul fichier de test ne l'aurait pas revele si le filtre avait
  exclu `pageTitle.test.ts`. Gate complet rejoue en avant-plan apres correctif : typecheck,
  lint (0 erreur, 30 warnings preexistants), `vitest run --pool=forks` (604 fichiers, 6369
  tests, 14 skip, 0 fail), `lint-no-hardcoded-colors` (0 violation), manifestes i18n a jour
  (aucun regenere : titre hors TOML). Investigation associee (avertissement React « Hooks
  order changed » sur `SquadSynergiesPage` vu dans le meme run) : § 7:2026-09-06,
  NON CORRIGE (non reproduit).
- 2026-09-06 : **revue adversariale ronde 2 de la phase 4 — DERNIERE salve, 3 constats, tous
  corriges** en 1 commit `tactique(4.6)`. G2, G3 et W1-W5 de la ronde 1 tiennent ; G1 etait
  PARTIEL. Le fil commun des trois : **une frontiere fermee par le CODE reste ouverte par le
  LIBELLE, par l'ORDRE, ou par un test qui la contourne.**
  - **P1 — L'ORACLE AVAIT SURVECU DANS LE MESSAGE.** La ronde 1 avait rendu le CODE du
    refus identique a celui d'une carte inconnue. Restait le corps : sur
    `/tactical/zzz/raster`, le service enrobait sa sentinelle avec la carte demandee
    (`fmt.Errorf("%w (%q)", ...)`) et `mapTacticalError` publiait `err.Error()` — le corps
    CITAIT l'identifiant. Sur un map_id hostile, le handler publiait la sentinelle nue, donc
    SANS l'identifiant. La presence des parentheses suffisait a dire laquelle des deux
    frontieres avait ete heurtee. Le test de la ronde 1 ne le voyait pas : son double rendait
    la sentinelle NUE, forme que le service reel ne produisait jamais — **un double infidele
    valide le handler contre une realite qui n'existe pas.**
    Corrige AUX DEUX COUCHES, chacune avec son propre test et sa propre inversion : le
    service rend la sentinelle nue et met le detail au JOURNAL
    (`service/tactical_service.go:129`, `:185` ; test
    `service/tactical_service_carte_test.go`) ; le handler publie le message CANONIQUE quoi
    que le service lui donne (`api/handlers/tactical.go:305-318` ; ceinture eprouvee par
    `api/handlers/tactical_oracle_test.go`, avec un double qui ENROBE). Les deux 400
    (`question`, `qui`) continuent, EUX, de nommer la valeur refusee : parametres de requete
    a validation unique, aucune seconde frontiere dont les rendre indiscernables, et nommer
    la valeur est ce qui rend le 400 utile. La comparaison octet pour octet porte desormais
    aussi sur les EN-TETES, et couvre les trois routes (raster, image, calage).
  - **P2 — L'ORDRE.** `h.newSvc` (qui resout le joueur et OUVRE sa base) s'executait avant
    `MapIDValide` sur la route raster, alors que les deux routes du fond validaient d'abord
    et que l'en-tete du fichier de tests affirmait le contraire. Validation remontee
    (`tactical.go:132-143`) ; le double COMPTE desormais les appels a la fabrique, pas
    seulement ceux au service — avec sa sentinelle anti-vacuite (sur une entree legitime, la
    fabrique EST appelee).
  - **P2 — UN TEST VERT POUR LA MAUVAISE RAISON, ET SEULEMENT SOUS WINDOWS.** Le test de la
    garde d'index posait sa cle hostile en la passant a `MapBackgroundMetaPath`, dont le
    `filepath.Join` NETTOYAIT le `..\` : le sidecar atterrissait hors de `map_backgrounds/`,
    qui n'etait donc jamais cree, `MapBackgroundIndexFor` echouait sur `os.ReadDir`, et la
    fonction sortait AVANT la garde. Retirer la garde laissait le test vert sur ce poste (il
    mordait sous Linux). Reecrit : le repertoire EXISTE, deux sidecars y sont poses par des
    noms de fichier choisis (`os.WriteFile`, jamais `filepath.Join` de la cle), dont
    `..evade.json` — un nom parfaitement legal dont le STEM porte `..`. Le decor est
    verifie AVANT l'assertion (l'index resout bien vers la cle hostile, et son sidecar est
    lisible), de sorte que seule la garde puisse expliquer le refus. Inversion rejouee ICI,
    sous Windows : garde retiree -> `err = <nil>`, la resolution ABOUTISSAIT.
- 2026-09-06 : arbitrages utilisateur — filtre de session a faire MARCHER (liste blanche `FilteredMatchIDs`), barre L2 = mix Explorateur + Escouade, « Escouade » = composition choisie. Items 4.5 / 4.6 ajoutes (phase 4 bis, avant la 6).
- 2026-09-06 : **revue adversariale ronde 1 de la phase 4 — 8 constats, TOUS corriges** en
  2 commits `tactique(4.5)`. Gate rejoue integralement, six inversions/mutations jouees.
  - **G1 (P1, P0 sur un hote Windows expose) — UNE TRAVERSEE DE REPERTOIRE.** Le `map_id`
    est la premiere cle de fond entierement controlee par l'appelant, et elle atteignait
    `filepath.Join` sans validation : `..\..\x` passe chi comme UN SEUL segment, et sous
    Windows l'antislash EST un separateur. Ce qui protegeait le depot n'etait pas une
    verification mais trois accidents de plate-forme. Deux portes posees (liste blanche au
    handler, garde de chemin au service, juste avant `PathResolver`), aucune normalisation
    (le `TrimSpace` prealable rendait `carte%20` valide — trouve par le test), et un refus
    OCTET POUR OCTET identique a une carte inconnue, pour ne pas remplacer un oracle par un
    autre.
  - **G2 (P1)** — le test qui affirmait que le fond n'est pas sous le garde local du rejeu
    ne montait aucun middleware : la mutation qu'il pretendait attraper le laissait vert.
    Remplace par un ratchet sur le SITE DE MONTAGE, a DEUX FACES (les routes tactiques
    dehors, le rejeu dedans) — sans la seconde face, la premiere passerait aussi si le
    garde disparaissait.
  - **G3, W1-W5** — un 400 mort retire du handler ; trois doubles de test trop complaisants
    (le chemin de requete jamais asserte, le fond jamais servi, `useMatchRoute` toujours
    faux) ; et une cle de cache par joueur pour une donnee de reference du titre.
  - **FIL COMMUN DES SIX DEFAUTS DE TEST** : un double qui repond toujours la meme chose ne
    teste rien. `api.get` sans assertion sur le chemin, `getBlob` toujours en rejet,
    `useMatchRoute` toujours faux, `httptest` toujours distant — chacun rendait vert le
    scenario qu'il pretendait couvrir. C'est le meme piege que la ronde 2 de la phase 3
    (« un garde-fou qui promet plus qu'il ne tient »), vu du cote des doubles.
- 2026-09-06 : **phase 4 CLOSE — l'onglet Tactique apparait sous Ascension**, en 3 commits
  (`tactique(4.4)`, `(4.2)`, `(4.3+4.1)`). Items 4.1-4.4 `[x]`, gate joue en avant-plan des
  deux cotes, six inversions jouees. Non pousse : revue du superviseur.
  - **ORDRE DES DEPENDANCES, PAS DE LA LISTE** : 4.4 (le contrat) precede 4.2 (les cles et
    le vocabulaire), qui precede 4.3+4.1 (la page et sa route). Chaque commit compile.
  - **4.3 ET 4.1 SONT UN SEUL COMMIT** : la route importe la page, et le schema de
    recherche de la route TYPE l'ecriture de `?carte=` que la page fait. Les separer donnait
    un commit qui ne compile pas, dans un sens comme dans l'autre.
  - **DECISION PRODUIT — le clic SELECTIONNE.** La route de la phase 5 n'existe pas (gelee
    jusqu'au lot D). Un `Link` vers elle ne compile pas ; un `navigate` vers une chaine
    construite est un lien mort. Le bouton ecrit `?carte=<map_id>` : le nom accessible dit
    « Selectionner <carte> », l'etat est `aria-pressed`, la selection se voit et se partage,
    et c'est l'etat que la phase 5 consommera. Aucun bouton qui ne fait rien sans le dire.
  - **ECART ASSUME AU BRIEF** : `PageUnavailable` (ADR 0029, ressource inaccessible)
    remplace par `RouteCapabilityGate` -> `FeatureUnavailable` -> `EmptyStateCard`, le
    gabarit que le depot reserve deja a « capability absente au niveau route » et qui porte
    le libelle FR/EN de `replay`. Justification detaillee dans la case 4.1.
  - **LE FOND DE CARTE SORT DU GARDE LOCAL DU REJEU**, et c'est le seul choix qui rendait la
    grille utilisable en production : ce garde protege les trajectoires decodees du film,
    pas une image de carte versionnee. Teste depuis une adresse non locale.
- 2026-09-06 : **revue adversariale ronde 2 de la phase 3 — DERNIERE salve, 3 constats P2,
  tous corriges** en 1 commit `tactique(3.9)`. Aucun P0, aucun P1. Trois garde-fous qui
  PROMETTAIENT plus qu'ils ne tenaient — c'est le fil commun des trois.
  - **P2-1 — le garde accesseur -> composant etait troue par SOUS-CHAINE.**
    `composants.includes('t.' + a)` acceptait `coverage` grace a `t.coverageHint`,
    `lowSample` grace a `t.lowSampleHint`, `delayBin` grace a `t.delayBinOpen`,
    `delayNarrative` grace a `t.delayNarrativeEmpty`. Le PREFIXE d'un accesseur n'est pas
    cet accesseur. Passe a une regex a FRONTIERE DE MOT (`t.<accesseur>`).
    DEUX inversions jouees, et la seconde est la vraie preuve : (A) `t.coverage(` retire de
    ses DEUX consommateurs, garde corrige -> tombe en nommant `coverage` ; (B) MEME
    suppression, garde revenu a `includes` -> **6 tests au vert**. Le bandeau de couverture
    — qui n'est pas decoratif, les films expirent — pouvait donc disparaitre des deux cartes
    sans que rien ne le dise. NOTE : l'inversion telle que scriptee par le superviseur
    (retirer `t.coverage(` de la SEULE matrice) ne tombe pas, et c'est normal —
    `coverage` a DEUX consommateurs, la matrice et la carte des delais.
  - **P2-2 — deuxieme definition de « match mesure ».** `matchsMesures` comptait les
    match_id distincts presents dans les evenements, alors que le drapeau `Mesure`
    (EXISTS + publishable) de G2 est calcule par `chargerUnivers` POUR LES DEUX SURFACES.
    Les deux coincidaient par accident (tous deux exigent `publishable`), rien ne les liait
    — et la fixture de test le prouvait, qui declarait `m1` `Mesure: false` en lui donnant
    deux kill-events sans qu'aucune assertion ne bronche. Une seule definition desormais :
    le drapeau du lecteur. Fixture rendue coherente ; deux tests ajoutes (un match LISIBLE
    et MUET compte au denominateur — zero legitime ; un match ILLISIBLE ne compte pas, meme
    si un evenement le mentionne) ; le sous-test « aucun match mesure » renomme « aucun
    match lisible » et pose desormais `Mesure: false` au lieu d'une absence d'evenements.
    INVERSION jouee : comptage par evenements restaure -> les deux nouveaux tests tombent.
  - **P2-3 — le lisere tirete n'etait pas un second indice.** `borderColor` valait la
    couleur de REMPLISSAGE, sous la meme opacite globale de 0,35 appliquee a l'element
    entier : a l'ecran, seule l'opacite se voyait. La doc de la prop, le commentaire de la
    carte et le test `borderType === 'dashed'` promettaient pourtant deux indices — et le
    test CADENASSAIT une promesse que l'ecran ne tenait pas. Le lisere est RETIRE plutot que
    maquille ; l'atténuation est un seul indice graphique, l'opacite, et le SECOND indice
    est le MOT : suffixe « hors fenetre » sur l'etiquette d'axe et pied de carte. Les trois
    docs sont reecrites en consequence. Retro-compat intacte : sans la prop, `data` reste
    `[3, 5, 2]` (test conserve).
- 2026-09-06 : lots C, B, F de l audit integres dans `feat/v75` et FUSIONNES dans `feat/tactique` (741e1731f, seul conflit : thought_log, union). Phases 4 et 6 degelees, 5 attend D, 7 suit 6. Item 3.7 statue `[~]` (porte deja dans le payload, doctrine `dataCapabilities.ts`). « Cap du moment » renomme « Constat du moment » (decision utilisateur).
- 2026-09-06 : **cloture de la phase 3 — deux retouches** (`tactique(3.8)`), sur le HEAD
  d'apres la fusion de `feat/v75` (lots C, B, F ; merge `741e1731f`). **(1) RENOMMAGE**
  decide par l'utilisateur : notre carte « Cap du moment » devient « **Constat du moment** »
  (EN « Current reading »). La page portait DEJA un « Cap d'escouade » (`SquadFocusStrip`,
  prospectif) et l'onglet Entrainement un « Cap du moment » (`CoachFocusCard`,
  `profile.coach.title`) : trois « Cap » a quelques centimetres, dont deux qui regardent
  devant et un qui regarde les matchs affiches. Cles de manifeste, accesseurs, symboles de
  logique, fichiers du composant et `data-testid` renommes d'un bloc — le seul « Cap » qui
  survit chez nous est celui des voisins, qui le meritent. **(2) ITEM 3.7 passe de `[!]` a
  `[~]`** : le lot C apporte `useDataCapability`, et sa doctrine tranche CONTRE un gate web
  ici (une cle qui n'est pas gatee cote UI n'entre pas dans `DATA_CAPABILITIES` ; une porte
  deja presente dans le payload ne se relit pas — precedent `film.usage_summary`). Il serait
  en outre FAUX pour Halo 5, servi par son kill-feed natif sans `film.kill_source`. La
  degradation Go (section omise) tient lieu de gate, et elle est testee. Aucun code pour ce
  volet.
- 2026-09-06 : **revue adversariale ronde 1 de la phase 3 — 12 constats, TOUS corriges** en
  5 commits `tactique(3.7)`. Gate rejoue integralement, cinq inversions jouees.
  - **W1 (P0) — LA MATRICE SE LISAIT DE TRAVERS.** `Heatmap2DChart` deduit ses categories
    d'axe de l'ORDRE D'APPARITION des points ; `matriceSeries` sautait la diagonale, donc
    la premiere COLONNE rencontree etait le DEUXIEME joueur : roster [A,B,C,D] -> lignes
    A,B,C,D, colonnes B,C,D,A ; sur un duo, l'axe X sortait exactement inverse. Chaque case
    designait le mauvais couple. Correction : TOUTES les cases roster x roster, dans
    l'ordre, diagonale COMPRISE avec une valeur VIDE (`value: number | null` -> `'-'`
    ECharts, hors visualMap, sans etiquette ni infobulle ; les vides sont ecartes AVANT le
    calcul de l'echelle). INVERSION : diagonale re-sautee -> 3 tests tombent, dont un qui
    rend litteralement `['B','C','D','A']`.
  - **W2 (P1) — les barres « attenuees » etaient BLEUES, et la doc disait le contraire.**
    `divergent-neutral` vaut #60A5FA (blue-400) dans la palette PAR DEFAUT, plus soutenu
    que la serie (blue-300) ; il n'est gris que sous okabe-ito / cividis / tol-bright.
    MESURE SUR PIECES sur les 4 palettes : AUCUN token semantique n'est achromatique
    partout (seuls 3 tokens de TEXTE de badge le sont, noirs ou blancs). Il n'y avait donc
    pas de « token gris » a prendre : l'attenuation passe par la COULEUR DE SERIE en
    opacite reduite + liseré tireté — les « barres hachurees » du plan d'origine, sans
    aucune dependance de palette. La prop du wrapper devient `binAttenuated` (predicat).
  - **W3 (P1) — le KPI n'avait aucun test et calculait.** L'ecart, son arrondi et le
    masquage plein-historique etaient inlines : supprimer le masquage ou inverser le signe
    passait. Sortis dans `squadEchange.logic.ecartEchange`, pure, consommee AUSSI par
    `capDuMoment`. `SquadEchangeKpi.test.tsx` cree (7 cas).
  - **W4 (P1)** — `formatLabel` etait une prop MORTE du wrapper partage : supprimee
    (regle 7). **W5 (P1)** — les branches neuves des deux wrappers sont testees, inversions
    jouees (sans `formatTooltip`, la matrice annonce « Win Rate: 400.0% » pour 4
    vengeances ; sans `binAttenuated`, `data` reste `[3, 5, 2]`).
  - **W6 (P1)** — « +20 pts de moins que d'habitude » se contredisait tout seul :
    `formatSignedPoints(Math.abs(...))` forcait le `+` dans une phrase qui portait deja la
    direction. `lib/baseline.formatPoints` (magnitude non signee) ajoutee a cote de la
    signee. **W7 (P2)** — fixtures dupliquees et deja divergentes, supprimees au profit de
    `squadEchange.fixtures`. **W8 (P2)** — cle i18n orpheline `empty_description`
    supprimee, et le garde va desormais jusqu'au COMPOSANT (il annoncait « aucune
    orpheline » a tort ; INVERSION jouee). **W9 (P2)** — borne exacte des badges
    (`n === 30`) posee a cote de son voisin a 29.
  - **G1 (P2)** — `bucketDelai` recopiait le decoupage en arithmetique (`delai / 1000`,
    bornage a 4) a cote de la tranche `bornesDelaiMs` qui le definit : deux definitions,
    dont une invisible. Il PARCOURT desormais la tranche. Consequence assumee : les
    intervalles sont SEMI-OUVERTS partout sauf celui qui ferme la fenetre (5 000 ms reste
    un echange). Test de PROPRIETE sur 13 delais. INVERSION : arithmetique restauree +
    borne deplacee -> le test nomme les deux divergences.
  - **G2 (P1, requalifie) — LE DENOMINATEUR « PAR MATCH » COMPTAIT DES MATCHS ILLISIBLES.**
    Le numerateur ne peut venir que des matchs dont le journal a ete lu ; le denominateur
    comptait tous les matchs du filtre. Les films Theater EXPIRENT : deux filtres a 20/20 et
    a 2/20 matchs decodes rendaient 0,20 et 0,02 pour EXACTEMENT le meme jeu — la grandeur
    que le contrat annonce « comparable d'un filtre a l'autre » ne l'etait pas. C'est le
    pendant du P0 de la phase 1 : le zero LEGITIME compte, l'ILLISIBLE est compte A PART.
    Corrige sur LES DEUX SURFACES. Escouade : `Couverture.ParMatch` et les cases de la
    matrice se divisent par `matchs_mesures`, `matchs_total` reste publie (le bandeau dit
    « N mesures sur M »). Tactique : l'univers passe aux `Rasterise*` est celui des matchs
    MESURES, et le contrat publie `matchs_filtres` (tous) a cote de `matchs_retenus`
    (mesures) ; l'exemple de reference 12 V / 8 D reste valide — ses 20 matchs sont mesures,
    10 sont seulement muets DE MOI. Le lecteur rend le drapeau par match (EXISTS sur
    `match_kill_events_latest`, `publishable` exige comme dans les deux lectures, parametre
    neutre et ratchet Campagne conserves). INVERSIONS jouees des DEUX cotes (Escouade :
    0,05 au lieu de 0,50 ; Tactique : `MatchsRetenus` 10 au lieu de 3 et valeur 0,30 au lieu
    de 1,00). Docs de `domain/tactical.go` et `domain/squad_echange.go` mises a jour dans le
    meme commit.
  - **Sans suite** : le dernier intervalle ouvert (> 7 s agrege 7,1 s et 400 s) est un choix
    produit remonte au decideur, pas un defaut.
- 2026-09-06 : **phase 3 CLOSE** — l'echange est mesure, servi et affiche, en 6 commits
  `tactique(3.1)` a `(3.6)`. Items 3.1-3.6 `[x]`, 3.7 `[!]` (le hook `useDataCapability`
  n'existe pas dans le depot : il arrive avec le lot C de l'audit v2 ; aucune gate de
  substitution inventee — la degradation est cote Go, section OMISE du `pageData`, et le
  comportement des composants est verifie par test). Gates joues en avant-plan, Go et web.
  TROIS decisions prises faute de tranche du plan, consignees au §7 : la matrice s'arrete
  au ROSTER quand le KPI porte sur le CAMP ; le « cap du moment » se tait aussi quand
  l'habituel n'a aucune mort mesuree ; les deux barres hors fenetre exigeaient une lecture
  SANS borne de temps, ajoutee au socle partage (`coordination.Ripostes`, noyau commun avec
  `Echanges`) plutot que recopiee dans le service. Non pousse : revue du superviseur.
- 2026-09-06 : les deux decisions produit ouvertes en phase 2 (substrat de « ou je gagne » = engagements ; KPI d echange par carte = mon camp) sont ARRETEES par l utilisateur — inscrites au §1.
- 2026-09-06 : **revue adversariale ronde 1 de la phase 2 — 16 constats, TOUS corriges** en
  6 commits `tactique(2.5)`. Gate rejoue integralement, quatre inversions jouees.
  - **T12** — `TacticalRepository` quitte `repository_data.go` (deja > 500 L, porte a 651
    par la phase 2) pour `port/tactical.go`, cree pour cette raison exacte cote service.
  - **R1 (P1 multi-titre)** — la porte de lecture etait une cle d'ECRITURE.
    `film.kill_positions` gouverne la CAPTURE (sa propre doc le dit) ; Halo 5 remplit la
    MEME table nativement (`match.events.spatial`) et recevait un 503. Deux predicats
    nommes, chacun a deux provenances ; le kill-feed natif est exige `supported`
    STRICTEMENT parce que `Has` accepte `degraded` — ce que declare Infinite, et c'est
    exactement le defaut qui fabriquerait de faux echanges.
  - **R2 (P1)** — les ~287 matchs de Campagne d'un joueur Halo 5 entraient dans la grille
    et dans l'univers. Token d'exclusion + prefixe `Q` sur les constantes : le garde-rail
    structurel existait, il ne voyait pas mes lecteurs faute du prefixe.
  - **R3 (P1)** — `match_registry.map_name_fr` est systematiquement NULLE ; toutes les
    cartes sortaient avec un nom FR vide, et la fixture semait une valeur qui n'existe pas
    en prod. Resolution par `asset_translations`, en REUTILISANT une des deux copies du
    paquet (promue en helper), jamais une troisieme.
  - **R4 + T9 + T10 + T11** — contrat : tags snake_case sur les quatre types nus, couverture
    de localisation publiee (`evenements_journal` / `evenements_localises`), les deux
    denominateurs de la lecture signee publies, `session` retire (jamais applique).
  - **T1-T8** — huit tests manquants, dont la face VICTIME de « ou je gagne » (P0 de test),
    `MapsPlayed` sous filtre aux trois couches, la borne `to`, la victime bot, le joueur hors
    composition, la mort non vengeable, la position de TUEUR partielle, l'echelle non
    symetrique. Deux docs inversees corrigees sur pieces (identites vides, `MatchsRetenus`).
- 2026-09-06 : phase 2 CLOSE — port, lecteur DuckDB, service et handler livres en 4 commits
  (`tactique(2.1)` a `(2.4)`), items 2.1-2.5 `[x]`, gate joue en avant-plan (vet propre,
  22 paquets verts en 42,3 s, ratchet de lint de la CI a 0 issue, contrat OpenAPI a jour,
  `generated.ts` en additions pures). DEUX DECISIONS PRODUIT prises faute de tranche du plan,
  consignees en §7 et A CONFIRMER : « ou je gagne » se lit sur les ENGAGEMENTS (mes kills ET
  mes morts), et le KPI d'echange d'une carte porte sur les morts de MON CAMP. Non pousse :
  revue du superviseur.
- 2026-09-06 : branche renommee `wt/tactique` -> `feat/tactique` par le superviseur : le filtre `push.branches` de `ci.yml` ne couvre pas `wt/**` (seulement `feat/**`, `fix/**`, ...), aucun run ne se declenchait. Meme convention que les lots `feat/v2-*` de l audit.
- 2026-09-06 : revision 4, worktree `LevelUp-wt-tactique` cree, phase 1 lancee.
- 2026-09-06 : **revue adversariale ronde 2 (derniere salve) — 3 constats, tous corriges** en
  2 commits `tactique(1.9)`. **R2-1 (P1, doc inversee)** : depuis la correction C, la phrase
  « les bornes d'une lecture agregee sont l'UNION des bornes des rasters sommes » etait fausse
  (un raster par match n'a aucune cellule a 3 matchs distincts, ses bornes sont vides par
  construction) — `analysis/tactical/doc.go:32-42` dit desormais que les bornes se lisent sur
  L'AGREGAT, et la regle est epinglee par `raster_test.go:229`. **R2-2 (P1)** : le garde-rail
  du taux nu exemptait tout type d'un autre paquet (`type TauxEchange float64` dans `domain`
  passait) ; logique inversee en LISTE BLANCHE datee des types de retour
  (`no_naked_rate_test.go:12-46` : error, bool, int, int64, string, `domain.Couverture`,
  `domain.BilanEchanges`, plus les conteneurs dont chaque composant CLE COMPRISE y figure —
  `MortSuivie` et `PaireEchange` ecartees, verification sur pieces : aucune fonction exportee
  ne les rend), verificateur `:147`. **R2-3 (P2)** : sentinelle anti-vacuite retablie
  (`no_naked_rate_test.go:76` : compte des fonctions exportees inspectees, echec a zero).
  Gate rejoue : `go vet` propre, `go test -count=1` vert sur 8 paquets en 10,7 s,
  `golangci-lint` a 0 issue.
- 2026-09-06 : **revue adversariale ronde 1 — 13 constats retenus, TOUS corriges** en 5
  commits `tactique(1.8)`. Gate elargi a `./internal/archlint/...` (le ratchet d'imports y
  entre) : `go vet` propre, `go test -count=1` vert sur 8 paquets en 9,1 s, `golangci-lint`
  a 0 issue, seuils reverifies.
  - **A (P0)** — le denominateur « par match » excluait les matchs retenus SANS point (12 V /
    8 D dont 2 victoires muettes : +0,10 lu au lieu de 0,00). L'univers devient une ENTREE
    explicite (`Rasterise(g, matchs, points)`, `RasteriseAvecResultats(g, resultats, points)`,
    tous deux `(*Raster, error)`), un point hors univers rend `ErrMatchHorsUnivers`. Tests :
    `raster_test.go:121` (match muet au denominateur), `:148` (hors univers), `:100` (univers
    declare vs points illisibles), `:164` (cotes sur l'univers),
    `merge_cellules_test.go:143` (le cas P0 chiffre : 12 V / 8 D, valeur 0,00).
  - **B** — `OutcomeUnknown` valait contradiction dans `Somme` ; il vaut desormais absence
    d'information (la valeur connue l'emporte), et les univers doivent etre identiques
    (`ErrUniversIncompatible`). Tests : `merge_test.go:114` et `:134`.
  - **C** — `Bornes()` englobait les cellules sous le plancher (cadre 14x trop large sur
    Dredge) ; il cadre les cellules LISIBLES. Test : `raster_test.go:196`.
  - **D** — l'invariant de purete n'etait garde par rien : ratchet
    `internal/archlint/tactical_pure_test.go:57` (ImportsOnly, refuse `analysis/replay`,
    `platform/duckdb`, `database/sql` dans les deux paquets).
  - **E** — le garde-rail du taux nu laissait passer `[]float64`, `map[string]float64`,
    `*float64` et un type nomme : deballage recursif + interdiction des types struct exportes
    (`no_naked_rate_test.go:33`, helpers `:104` et `:127`).
  - **F** — cinq tests manquants de `merge.go` : `merge_cellules_test.go:193` (cellule
    incomplete, cotes asymetriques 4 V / 3 D), `:96` (plancher decidant des deux cotes),
    `merge_test.go:181` (pas de 0,25 m et points illisibles cumules),
    `merge_cellules_test.go:299` (tri signe).
  - **G** — le tri des paires d'echange n'avait jamais deux vengeurs : `trade_test.go:317`
    (quatre vengeurs, deux paires partageant le meme venge, cinq executions).
  - Seuil : `merge_test.go` passe a 523 L par ces ajouts, scinde en `merge_test.go` (Somme +
    helpers) et `merge_cellules_test.go` (lectures agregees).
- 2026-09-06 : phase 1 CLOSE — socle pur livre en 6 commits (`tactique(1.1)` a `(1.6)`),
  items 1.1-1.6 `[x]`, 1.7 `[~]` (couvert par 1.1/1.5/1.6), gate vert et rejoue en
  avant-plan, `golangci-lint` a 0 issue, zero I/O et zero import de `analysis/replay` ou de
  `platform/duckdb` dans les deux nouveaux paquets. Non pousse : revue du superviseur.

## 7. Decouvertes (a remplir pendant l'execution — ne rien corriger hors perimetre)
- 2026-09-06 (phase 6, revue) — **LE DOCUMENT EST RELU TROIS A QUATRE FOIS PAR ARTEFACT ET
  PAR CYCLE.** Les quatre projections post-cuisson (T0 du film, resume d'usage, stats
  d'Assaut, raster tactique) appellent chacune `lireDocumentRange` sur le MEME fichier —
  soit, sur un document de ~2 Mo, une dizaine de megaoctets lus et desserialises par match.
  La factorisation de la lecture est faite (C7) ; le passage a UNE lecture par artefact, le
  document circulant entre les projections, ne l'est PAS : il change la forme des quatre
  `persister*`/`projeter*`, c'est-a-dire une refonte de flux, pas une factorisation
  mecanique. NON TRAITE — melanger les deux dans le meme diff aurait rendu la revue de
  chacune impossible. Petit lot a part, mesurable au compteur
  `replay_artifact_digest_reads_total`.
- 2026-09-06 (phase 6, revue) — **UN FAUX SPAWN A LA SORTIE D'UN VEHICULE.** Un occupant qui
  descend rouvre une vie de bipede, et cette vie est ouverte par un trou > 5 s — donc son
  premier point est publie comme un SPAWN alors que ce n'est pas une reapparition mais une
  descente. Consequence : les grappes de reapparition de la phase 7 verraient des amas aux
  points de descente des vehicules, la ou personne ne reapparait jamais. La matiere pour le
  trancher existe deja dans le sidecar : un spawn dont la frame suit de peu le `T1` d'un
  embarquement du meme joueur n'en est pas un. NON TRAITE : c'est une decision de la phase 7
  (grappes de spawn), et la corriger a l'aveugle ici aurait fige une heuristique avant
  d'avoir la lecture qui la valide.
- 2026-09-06 (phase 6, revue) — **LA COUVERTURE DES EPISODES D'EMBARQUEMENT EST PARTIELLE
  ET C'EST UNE LIMITE AMONT.** La primitive d'attribution n'apparie que 15,6 a 21,1 % des
  vies de vehicule (mesure publiee dans `analysis/replay/document_vehicles.go`). Le temps
  embarque non apparie n'est donc mesure par personne : la lecture « ou je passe mon temps »
  SOUS-ESTIME le vehicule, structurellement. C'est ecrit dans `domain/tactical_raster.go`
  pour qu'aucun lecteur ne prenne le silence pour une absence de deplacement. NON TRAITE :
  se corrigerait en amont, dans la primitive, pas dans ce chantier.
- 2026-09-06 (phase 6) — **LA PURGE RECURRENTE DES ARTEFACTS NE CONNAIT PAS LES SIDECARS :
  ils deviennent ORPHELINS.** `scheduler.purgeReplayArtifactsForTitle` supprime les
  artefacts sortis de la fenetre `replay_retention_months` et saute les repertoires
  (`e.IsDir()`) — ce qui protege bien `rasters/` d'une suppression accidentelle, mais laisse
  aussi sur le disque le sidecar d'un artefact purge. Consequence : aucune erreur, aucune
  lecture fausse (un match hors fenetre est hors perimetre de toute facon), seulement de
  l'espace disque qui ne se libere jamais — quelques kilooctets par match, contre quelques
  megaoctets pour l'artefact. NON TRAITE : le fichier lu par le cron est
  `internal/scheduler/replay_purge_cron.go`, hors perimetre de la phase 6, et le choix
  « purger aussi le sidecar » vs « les garder, ils sont minuscules et re-servent si la
  fenetre s'elargit » est une decision produit.
- 2026-09-06 (phase 6) — **`domain/tactical.go` EST A 504 LIGNES, deja au-dela du seuil.**
  La phase 4 bis l'y a porte (486 -> 504). La phase 6 ne l'a donc PAS grossi : la constante
  `TacticalQuestionTemps` vit dans `domain/tactical_raster.go`, a cote du substrat dont elle
  tire sa mesure, avec la justification ecrite sur place. Consequence assumee : le
  vocabulaire complet des questions se lit en DEUX endroits. NON TRAITE (la dette de seuil
  est gelee par la baseline, CLAUDE.md n 5) ; une scission de ce fichier — l'evidente est
  « types de lecture » vs « ce que la page publie » — serait un petit lot a part.
- 2026-09-06 (phase 6) — **DEUX TESTS EXISTANTS PRENAIENT « temps » POUR UNE QUESTION
  INCONNUE.** `TestRaster_QuestionEtAxeNommentLaValeurRefusee` et
  `TestTacticalService_VocabulaireRefuse` employaient cette chaine comme exemple de valeur
  hors vocabulaire ; l'ouverture de la quatrieme question les a fait tomber. Corriges DANS
  LE PERIMETRE (leur intention est intacte : la fixture prend « tout-sauf-ca », qui n'a
  aucune chance d'entrer au vocabulaire, et le commentaire dit pourquoi). Lecon generale :
  **une fixture de valeur INVALIDE ne doit pas emprunter au champ semantique de la
  feature** — elle finit par devenir valide.
- 2026-09-06 (phase 4 bis) — **LES SESSIONS PROPOSEES NE SONT PAS CELLES DE LA COMPOSITION
  EXACTE, et l'ecart est dans le sens de la prudence.** La barre de l'Escouade propose
  `composition_sessions`, que seule la requete de page teammates sait calculer (intersection
  serveur sur la composition exacte). L'onglet Tactique, lui, prend la liste de
  `/filters/resolve` et la coupe sur `is_squad` — donc les sessions d'ESCOUADE du joueur, pas
  celles de la composition. Consequence : une session proposee peut ne porter AUCUN match de
  la composition (le serveur resserre ensuite, la grille est alors vide) ; l'inverse — masquer
  une session que la composition a jouee — ne peut pas se produire. Aller chercher la liste
  exacte imposerait la requete de page Escouade (lourde) et un import croise
  `tactical => squad` au plafond de 7/7. NON TRAITE, a revoir si la V1 montre le cas.
- 2026-09-06 (phase 4 bis) — **`matchs_filtres` A CHANGE DE SENS, et la phase 5 doit le
  savoir.** Il valait « les matchs du filtre SUR CETTE CARTE » ; il vaut desormais « la taille
  de la liste blanche recue », toutes cartes confondues (decision superviseur). Le pied de
  carte de la phase 5 ne doit donc PAS en faire un rapport (« N sur M ») : les deux grandeurs
  n'ont pas le meme denominateur. Le champ le dit dans sa doc, en toutes lettres.
- 2026-09-06 (phase 4 bis) — **LE MOTIF `strings.TrimRight(strings.Repeat("?,", n), ",")` A
  AU MOINS SIX COPIES dans `platform/duckdb`** (`career_repo_encounters.go:28`,
  `citations_repo.go:415`, `explorer_repo.go:116/191/456/572`,
  `highlight_events_repo.go:126/186`, ...) alors que le helper canonique `Placeholders(n)`
  existe depuis la migration sharedprovider, dans le meme paquet, avec sa doc d'usage. Le
  lecteur tactique consomme le helper ; les copies preexistantes ne sont PAS touchees (hors
  perimetre). Une passe de centralisation + garde-rail grep serait un petit lot a part.
- 2026-09-06 (fix CI hors phase, hors perimetre Tactique) — **AVERTISSEMENT REACT « Hooks
  order changed » sur `SquadSynergiesPage`, vu dans le run CI 34031528843, NON CORRIGE :
  non reproduit.** Le run affichait, pendant le test `SquadSynergiesPage.test.tsx >
  capability expected_stats presente...`, un avertissement « An update to SquadSynergiesPage
  inside a test was not wrapped in act(...) » suivi de « React has detected a change in the
  order of Hooks called by SquadSynergiesPage » (position 1 : `useCallback` -> `useContext`).
  Verifie sur pieces : `SquadSynergiesPage.tsx:36-49` (les 5 hooks propres —
  `useSquadContext`/`useContext`, `useFieldMappings`, `useAppShellStore`, `useCapability`,
  `useMemo` — sont TOUS inconditionnels, avant les deux retours anticipes `!hasSelection` /
  `!hasRows`, lignes ~52 et ~63) ; `SquadEchangeConstatCard.tsx` (son unique retour anticipe
  `if (!cap) return null` vient APRES `useAppShellStore` et `useMemo`) ; et
  `SquadEchangeMatrixCard.tsx` (aucun retour anticipe, tous les hooks inconditionnels) —
  les trois composants nommes dans la consigne sont propres. Le mismatch rapporte
  (`useCallback` en position 1) ne correspond a AUCUN hook reellement declare dans
  `SquadSynergiesPage` : dans ce test, `useSquadContext` est integralement mocke
  (`vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue(...)`, aucun hook interne
  reel appele) — incompatible avec une violation structurelle de ce fichier, compatible avec
  une mise a jour hors `act()` d'un composant tiers de l'arbre (react-query/zustand) qui
  brouille l'attribution du fiber en fin de test. NON REPRODUIT : `vitest run
  SquadSynergiesPage.test.tsx` isole (7/7 verts, aucun avertissement) et `vitest run
  --pool=forks` suite complete x2 (604 fichiers, 6369 tests, 0 avertissement « Hooks »).
  Aucune correction appliquee (rien a corriger sur preuves ; un correctif sans repro serait
  un fix a l'aveugle, interdit par la regle de verification sur pieces). A surveiller au
  prochain run CI rouge sur ce fichier.
- 2026-09-06 (phase 4, revue R1) — **LA MEME BRANCHE MORTE SURVIT DANS `handlers/replay.go`.**
  Les deux `if in.MatchID == "" { 400 missing_match_id }` (`replay.go:79-81`, `:105-107`,
  et le pendant chi `:157-160`) sont inatteignables pour la meme raison que les
  `missing_map_id` retires ici : chi ne route pas un segment vide, et la liaison de
  parametre de Huma refuse la chaine vide avant le handler. Le 400 n'est declare nulle part
  au contrat. Hors perimetre de la phase 4 (fichier du rejeu, lot B/D de l'audit). NON
  TRAITE.
- 2026-09-06 (phase 4, revue R1) — **AUCUNE AUTRE ROUTE DU DEPOT NE FABRIQUE UN CHEMIN DE
  FICHIER A PARTIR D'UN PARAMETRE D'URL** : verifie sur pieces pendant G1. Les autres
  lectures de fichiers passent par un identifiant deja resolu en base (match_id ->
  `match_registry`) ou par un catalogue versionne. La surface fermee par `MapIDValide` etait
  la seule de cette forme ; si une seconde apparait, elle doit reutiliser ce helper plutot
  qu'en recopier le motif (la regle des <= 2 copies s'appliquera des la deuxieme).
- 2026-09-06 (arbitrage utilisateur) — **LOT « MINIATURES DE CARTES » ACCEPTE**, a ouvrir apres la V1 de Tactique : la vignette de la grille charge le PNG pleine resolution (jusqu a 1,4 Mo par carte, une quinzaine de cartes) faute de pipeline de miniatures dans le depot. Perimetre pressenti : generation d une miniature par fond (PathResolver, meme sidecar), endpoint `.../background/thumb.png`, la grille la consomme ; le rejeu garde la pleine resolution. Hors de ce plan.
- 2026-09-06 (phase 4) — **LE CALAGE PAR CARTE N'A AUCUN CONSOMMATEUR WEB EN PHASE 4.**
  `GET /players/{slug}/tactical/{map_id}/background` (Huma) est servi parce que le contrat
  PAR MATCH en sert un et que le depot a pose la regle « l'image et son calage vont toujours
  ensemble » (`handlers/replay.go`, `MapBackgroundImage` refuse de servir une image dont le
  sidecar est illisible). La vignette de la grille n'a besoin que de l'image ; c'est la
  phase 5 (le plan de la carte, en coordonnees monde) qui lira le calage. Surface publiee un
  lot en avance, ASSUMEE et signalee ici : si la phase 5 devait etre abandonnee, cette route
  serait a retirer avec elle.
- 2026-09-06 (phase 4) — **LE FILTRE DE SESSION DE L'OMNIBAR EST IGNORE PAR L'ONGLET, EN
  SILENCE.** Le contrat de l'onglet n'accepte pas `session` (retrait T11 : aucune requete
  shared ne l'honore), et le client le retire donc avant d'envoyer. Consequence produit : un
  joueur qui epingle une session dans l'omnibar voit une grille qui porte sur la periode
  entiere, sans que rien ne le lui dise. Trois issues possibles (joindre la base joueur,
  faire descendre des `match_id`, ou afficher une reserve dans le pied de carte) ; aucune ne
  releve de la phase 4. NON TRAITE.
- 2026-09-06 (phase 4) — **LA VIGNETTE CHARGE L'IMAGE PLEINE RESOLUTION.** Un fond de carte
  pese jusqu'a 1,4 Mio (chiffre de `match-replay/queries.ts`) et la vignette l'affiche dans
  un cadre de quelques centaines de pixels. Une grille de trente cartes telecharge donc
  plusieurs dizaines de Mio pour un rendu de miniatures. Il n'existe aucun pipeline de
  miniatures dans le depot ; en creer un (cuisson d'une variante reduite a cote du fond,
  meme sidecar) est un lot a part. NON TRAITE.
- 2026-09-06 (phase 4) — **LA CONSIGNE `git checkout -- routeTree.gen.ts` NE VAUT QUE POUR
  UN LOT QUI N'AJOUTE PAS DE ROUTE.** Le fichier est VERSIONNE et les commits qui ajoutent
  une route l'incluent (precedent : `4cc6e4b97`, la route du rejeu 2D). La regle du depot est
  « ne jamais l'EDITER a la main » — il se regenere par le plugin de routeur, ici via
  `npm run build`. Le diff a ete verifie ligne a ligne (26 lignes, toutes relatives a
  `ascension/tactique`) avant d'etre stage.
- 2026-09-06 (phase 4) — **`react-hooks/set-state-in-effect` interdit le motif « URL d'objet
  creee au montage, revoquee au demontage ».** La premiere version du chargeur de fond
  posait l'URL dans un `useState` depuis un `useEffect` : lint (warning) et cascade de
  rendus. Retenu : l'URL d'objet EST l'entree de cache (`staleTime` et `gcTime` infinis),
  bornee par le nombre de cartes du titre. Contrepartie assumee : aucune revocation avant la
  fermeture de l'onglet. `match-replay` evite le probleme autrement (il decode vers un
  `HTMLImageElement` et revoque aussitot), mais ce motif ne se rend pas dans un `<img src>`.
- 2026-09-06 (phase 3) — **DECISION PRODUIT : la MATRICE s'arrete au ROSTER quand le KPI
  porte sur le CAMP.** Le plan tranche le perimetre du taux (mon camp entier, decision
  utilisateur du 2026-09-06) mais pas celui de la matrice. Retenu : les axes de la matrice
  sont le joueur principal et les coequipiers SELECTIONNES ; un allie de passage qui venge
  un membre du camp compte au TAUX (il est bien de mon camp) et n'a AUCUNE ligne dans la
  matrice, parce que la page ne sait pas le nommer — afficher un xuid nu vaut moins que
  l'ecarter (doctrine SquadAssistPairsTable, qui ecarte pour la meme raison). Les deux
  perimetres sont documentes cote a cote dans `domain/squad_echange.go` et testes
  (`TestBuildSquadEchange_MatriceEcarteHorsRoster`). CONSEQUENCE ASSUMEE : la somme des
  cases de la matrice peut etre inferieure au `brut` du KPI.
- 2026-09-06 (phase 3) — **DECISION PRODUIT : le « cap du moment » se tait aussi sans
  REFERENCE MESUREE.** Le plan nomme deux seuils (30 morts d'equipe, 5 points d'ecart). Un
  troisieme refus a ete ajoute : si l'habituel porte sur ZERO mort vengeable, son taux vaut
  0 par construction et l'« ecart » affiche ne serait que l'absence de la reference — la
  carte n'est pas rendue (`squadEchange.logic.capDuMoment`, teste). Le cas se produit sur
  une composition dont AUCUN match historique n'a de journal des morts.
- 2026-09-06 (phase 3) — **LES DEUX BARRES HORS FENETRE N'EXISTAIENT PAS DANS LE SOCLE.**
  `coordination.Echanges` ne rend un delai que pour les morts vengees DANS la fenetre de
  5 s : la distribution demandee par le plan (5-7 s, au-dela) n'avait aucune source. Trois
  issues etaient possibles ; la retenue est `coordination.Ripostes` (meme noyau
  `suivreMorts`, borne desactivee), parce que recopier la recherche de vengeur dans le
  service aurait donne deux definitions de « qui venge qui » au premier ajustement, et que
  laisser les deux barres a zero aurait affiche une distribution muette. Le type de retour
  `[]domain.MortSuivie` a impose une ligne datee dans la liste blanche de
  `no_naked_rate_test.go` (le type ne porte aucun quotient). Invariant teste : sous la
  fenetre, `Ripostes` et `Echanges` rendent le MEME vengeur et le MEME delai.
- 2026-09-06 (phase 3) — ~~`useDataCapability` N'EXISTE PAS~~ **SOLDE par le lot C
  (integre le 2026-09-06, merge `741e1731f`) : le hook existe, et sa doctrine tranche CONTRE
  ce gate chez nous.** `dataCapabilities.ts` n'accepte dans `DATA_CAPABILITIES` qu'une cle
  EFFECTIVEMENT gatee cote UI, et refuse de relire une porte deja presente dans le payload
  (precedent `film.usage_summary`, documente sur place) — or notre section est OMISE du
  `pageData` quand la porte est fermee. Un `useDataCapability('film.kill_source')` serait
  la seconde source de verite ET faux pour Halo 5, servi par son kill-feed natif sans cette
  cle. Item 3.7 statue `[~]`. RIEN A FAIRE.
- 2026-09-06 (phase 3) — **DEUX WRAPPERS DE CHART ETAIENT TROP ETROITS POUR UN SECOND
  USAGE, et c'etait invisible.** `Heatmap2DChart` cablait en dur un tooltip « Win Rate /
  Matchs » (donc faux pour toute autre donnee) ; `HistogramChart` ne peint que `series[0]`
  et IGNORE EN SILENCE toute serie supplementaire. Les deux ont recu une option optionnelle
  plutot qu'un jumeau (aucun appelant existant modifie). Le silence de `HistogramChart` sur
  `series[1..]` reste un piege pour le prochain appelant : il merite un avertissement de
  developpement ou un test. NON TRAITE (hors perimetre).
- 2026-09-06 (phase 3) — **`tools/lint-cross-feature-imports.mjs` est A SON PLAFOND (7/7).**
  Tout nouvel import croise non declare fait echouer le gate. Ce lot l'a contourne
  proprement (deplacement vers `lib/`), mais les 7 violations residuelles restent, dont les
  4 importeurs de `explorerMatchesClientSort` deja notes comme « demenagement du a faire
  dans un lot dedie ». NON TRAITE.
- 2026-09-06 (phase 2) — **DECISION PRODUIT A CONFIRMER : « ou je gagne » se lit sur les
  ENGAGEMENTS.** Le plan tranche la FORME (raster signe, echelle symetrique, plancher par
  cote) mais pas le SUBSTRAT. Retenu : mes kills ET mes morts
  (`domain.TacticalQuestionGagne`, `service/tactical_service.go:165`). Motif : la question
  demande ou ma presence correle avec la victoire ; la seule presence mesurable avant les
  rasters d'occupation (phases 6-7) est le combat, et il a deux faces — ne garder que les
  kills confondrait « ou je gagne » avec « ou je tue », qui est deja une question a part.
  Substitution prevue par l'occupation quand elle existera.
- 2026-09-06 (phase 2) — **DECISION PRODUIT A CONFIRMER : le KPI d'echange d'une carte porte
  sur les morts de MON CAMP** (mes coequipiers ET moi), pas sur les miennes seules ni sur
  toutes les morts du match (`tactical_service.go:245`). Le plan dit « par carte, libelle
  sur cette carte » sans nommer le perimetre.
- 2026-09-06 (phase 2) — ~~`port/repository_data.go` passe de 614 a 651 lignes~~ **SOLDE
  par T12** (revue ronde 1) : l'interface est allee dans `port/tactical.go`, le fichier est
  revenu a 614 L.
- 2026-09-06 (phase 2) — **le vocabulaire de filtre de l'Explorateur a maintenant DEUX
  lecteurs Go** : `handlers.parseNeighborsFilterSpec` (sur `*http.Request`) et
  `handlers.TacticalFilterQuery.spec` (sur une entree typee Huma). Les PREDICATS sont
  partages (`playlistOrSessionPattern`, `xuidPattern`, `parseCsvFilterParam`,
  `analysis.IsValidOutcomeLabel`), seul l'assemblage est double. La regle des <= 2 copies
  tient ; un TROISIEME lecteur exigera un helper canonique + garde-rail. NON TRAITE.
- 2026-09-06 (phase 2) — **PIEGE HUMA : un champ EMBARQUE de type non exporte n'est pas lie.**
  `tacticalFilterInput` embarque dans l'entree rendait TOUS les filtres vides, sans erreur ni
  log — la reflexion de Huma ne peut pas assigner un champ non exporte. Type renomme
  `TacticalFilterQuery` (exporte) ; le piege est garde par
  `TestTacticalHandler_FiltreExplorateur`. Aucun autre handler du depot n'embarque de struct
  d'entree : rien d'autre a verifier.
- 2026-09-06 (phase 2) — **`make generate-types` ne regenere PAS `openapi.yaml`.** La cible
  n'appelle qu'`openapi-typescript` sur le yaml existant ; c'est `make openapi-gen` qui ecrit
  `apps/go-api/api/openapi.yaml` (et non `apps/go-api/openapi.yaml`). Les deux ont ete
  jouees, dans cet ordre. Par ailleurs ce worktree n'a PAS de `apps/web/node_modules` :
  `make generate-types` echoue, et la generation a ete faite avec le binaire
  `openapi-typescript` 7.13.0 du checkout principal (meme version que `package.json`), lu en
  seule lecture. `tools/check-generated-types-fresh.mjs` passe en SILENCE (exit 0) quand le
  CLI manque — un garde-rail qui ne peut pas mordre en local ; la CI, elle, l'exercera.
- 2026-09-06 (revue R1) — **une cle FINE de LECTURE manque au vocabulaire des
  capabilities.** La porte des rasters est aujourd'hui un OU sur deux cles qui repondent
  chacune a une AUTRE question : `film.kill_positions` dit « ce titre CAPTURE des positions
  en decodant son film », `match.events.spatial` dit « ce titre expose des positions
  NATIVES ». Aucune des deux ne dit ce que le lecteur veut savoir — « la table
  `kill_positions` est-elle lisible pour ce titre ? ». Une cle du genre
  `match.kill_positions.readable` le dirait d'un mot et retirerait le OU. Elle releve du
  VOCABULAIRE DE CAPABILITIES, chantier du lot C de l'audit v2 — pas de ce lot, qui a pour
  consigne de n'en creer aucune. NON TRAITE.
- 2026-09-06 (revue R3) — **la resolution du nom FR d'une carte existe en DEUX copies dans
  `platform/duckdb`** : `mapNameFRFromAssetTranslations` (par `map_id`, promue en helper de
  paquet pour ce lot) et `FiltersRepo.applyMapFRTranslations` (par NOM EN, faute de map_id
  sous la main — elle resout d'abord nom -> id). Le lecteur tactique est le DEUXIEME
  consommateur de la premiere, la regle des <= 2 copies tient donc encore ; un troisieme
  consommateur exigera un helper canonique unique + garde-rail. NON TRAITE.
- 2026-09-06 (revue T11) — **le filtre de SESSION n'est applicable par aucune requete
  shared.** `MatchFilterSpec.SessionID` existe, mais `BuildNeighborsWhereClause` le range
  dans `IgnoredFilters` : les sessions vivent dans `player_match_enrichment` (base JOUEUR),
  que ces requetes ne joignent pas. Le parametre a ete retire du contrat de l'onglet plutot
  que laisse en decor. S'il devient necessaire, il faudra soit joindre la base joueur, soit
  faire descendre des `match_id` depuis la page. NON TRAITE.
- 2026-09-06 (phase 2) — **`internal/contracttest` n'existe pas** : le paquet est a
  `apps/go-api/contracttest`. Le gate a ete joue sur `./contracttest/...`.
- 2026-09-06 (phase 1) — **TROIS implementations de quantile dans le depot, deux conventions
  differentes.** `analysis/replay.gwPadsQuantile` tronque l'index (`s[int(q*(n-1))]`),
  `analysis/temporal.quantileSorted` interpole lineairement ; `analysis/tactical.quantile`
  est la troisieme et suit la convention de `temporal`. Deux mesures du depot qui disent
  « p95 » ne disent donc pas toutes la meme chose. La regle des <= 2 copies demanderait un
  helper canonique + garde-rail, ce qui exigerait de toucher `replay` et `temporal` : hors
  perimetre de la phase 1, NON TRAITE. Candidat pour un lot de dette dedie.
- 2026-09-06 (phase 1) — **le hook `go-vet` de lefthook noie son verdict.** Chaque commit
  affiche une soixantaine de lignes « build constraints exclude all Go files » (les `cmd/*`
  et `internal/ooz` derriere un tag ou du CGO) avant son `OK`. Un vrai echec de vet s'y
  perdrait a la lecture. NON TRAITE (hors perimetre).

## 8. Reprise de session
Avancement = les cases de ce fichier dans le worktree. Reprendre a la premiere case non
statuee de la phase la plus basse non close. `[x]` fait, `[~]` couvert ailleurs (reference),
`[!]` non traite (justification). Aucune case vide a la cloture.
