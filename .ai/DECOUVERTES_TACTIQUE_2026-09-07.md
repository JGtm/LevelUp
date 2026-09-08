# Decouvertes du chantier Tactique — registre a part (decision utilisateur 2026-09-07)

Regle : on CONSIGNE, on ne traite pas. Chaque entree : date ; fichier:ligne ; fait ; condition de reprise.
Le §7 du plan `.ai/PLAN_TACTIQUE_2026-09-06.md` reste la source des decouvertes des phases 1 a 7A ;
ce fichier prend le relais a partir de 7C et recoit toute nouvelle decouverte.

## 7C — faits d'isolement au sync (branche `feat/tactique`, HEAD `c894e0282`)

- 2026-09-07 ; `internal/sync/killcollector/roster.go:157` + `analysis/replay/death_context.go:194` ;
  **FAIT FAUX POSSIBLE** : `ArriveeMS`/`DepartMS` (horloge API, depuis `start_time`) sont compares a
  `time_ms` du journal (horloge FILM, frame 0) ; ecart T0 de 17 a 92 s selon les matchs ; pour les
  morts du debut de partie un coequipier present est sorti du total -> `teammates_visible = 0`,
  `nearest NULL` = « isolee ». NON CORRIGE, NON PUSHE EN PROD (branche de feature). Reprise : soit
  retirer l'usage de la participation en V1 (`teammates_left` = 0), soit caler avec
  `match_registry.real_start_time` / `t0_quality` (appareil existant, `cmd/backfill_t0_film`).
  **A traiter AVANT tout merge de `feat/tactique`.**
- 2026-09-07 ; `analysis/replay/lives_export_test.go:180-193` ; test d'orthogonalite tautologique
  (trois litteraux) ; l'orthogonalite `end_cause`/`named_by` n'est pas prouvee sur un vrai pont.
  Reprise : fixture `ResolveSlotXUID` avec fermeture par reapparition (`closeByRespawn`, `fire = nil`).
  **TRAITÉ le 2026-09-07 (Q8)** : `TestViesNommees_LesDeuxAxesSontOrthogonaux` rejoue desormais
  un vrai pont (deux vies de 111 calibrent la fenetre de reapparition, une troisieme vie
  anonyme est nommee par `closeByRespawn` et sort `film_end`+`closure`) ; vert.
- 2026-09-07 ; `sync/killcollector/isolation_facts_integration_test.go` ; le test film reel ne peut
  jamais passer (roster `fakeRoster{}` vide -> `Equipes` vide -> skip). Reprise : roster construit
  depuis `replay.ScanDeaths(film)`, exiger vies > 0 ET contextes > 0, `t.Fatal` sinon ; commande
  `KILLSOURCE_FIXTURES=<racine>/data/cache/film_chunks go test -count=1 -tags=integration -p 1 -run FaitsDIsolementFilmReel ./internal/sync/killcollector/`.
  **TRAITÉ le 2026-09-07 (Q8)** : roster `filmRoster` construit depuis `replay.ScanDeaths(film)`
  (gamertag+xuid reels du film), garde `positions==0 || vies==0 || contextes==0` par film puis
  `t.Fatal` (plus de `t.Skipf` maquillant une couverture nulle) ; compile sous `-tags=integration`
  (verifie), **pas encore rejoue sur fixtures reelles** — a faire par le superviseur, commande
  ci-dessus depuis `apps/go-api`.
- 2026-09-07 ; `sync/killcollector/isolation_facts.go:176` ; un pont non publiable
  (`IndexDisagreements > 0`) fait tomber toutes les morts dans `killsource_isolement_morts_sans_lieu`.
  Reprise : compteur dedie.
  **TRAITÉ le 2026-09-07 (Q8)** : `metricIsolationPontNonPublicable` (`killsource_isolement_pont_
  non_publiable`) ajoute, `toDeathContextRows` l'alimente AVANT tout appel a `ContextesDesMorts`
  quand `!replay.PontPubliable(mat.report)` ; test `TestToDeathContextRows_PontNonPublicable_
  CompteDedie` vert.
- 2026-09-07 ; `sync/killcollector/roster.go` ; `LEFT JOIN match_registry` sans test. Reprise : test
  `IdentitiesForMatch` sur base `:memory:` migree, participants sans registre.
  **FERME CADUQUE le 2026-09-07 (Q8)** : verifie sur pieces, `roster.go` (module `IdentitiesForMatch`)
  ne porte plus aucun `LEFT JOIN match_registry` — le JOIN a disparu avec 7C.9 (retrait
  `ArriveeMS`/`DepartMS` et de la requete `match_registry` qui les lisait). Rien a tester.
- 2026-09-07 ; `platform/duckdb/no_raw_rating_reads_test.go:44` ; `match_kill_events`,
  `kill_positions`, `match_bomb_stats` absentes de la regex de lecture brute. Reprise : les ajouter.
  **TRAITÉ le 2026-09-07 (Q8)** : les trois tables ajoutees a `rawRe` ; mutation testee (un
  literal `FROM kill_positions` hors vue fait rougir `TestNoRawAppendOnlyReads`) ; **P0 = 0** :
  tous les lecteurs existants (`platform/duckdb`, `api`, `analysis`) passaient deja par `_latest`.
- 2026-09-07 ; `sync/killcollector/positions.go:103` ; le refus des positions au sync etait
  journalise en Debug seulement — une table neuve vide ne se remarque pas. Reprise : WARN + compteur.
  **TRAITÉ le 2026-09-07 (Q8)** : le refus « collecteur non cable » (capability presente,
  `WithPositionCapture` absent — la vraie regression de cablage, distincte du refus « titre sans
  la capability » qui reste Debug) passe en `slog.WarnContext` + `metricPositionsNotWired`
  (`killsource_positions_non_cablees`) ; test mis a jour, vert.
- 2026-09-07 ; `sync/killcollector/capture.go` ; la capture de positions est desormais branchee au
  sync de PROD (elle ne l'etait que dans le backfill) : scan complet des bipedes par match synchronise.
  Cout a observer au premier deploiement (duree de l'etape post-sync, `PostSyncBudget`). NON TRAITE
  (point de vigilance PROD, hors perimetre Q8 — cf. §1.2 du plan d'orchestration).
- 2026-09-07 ; `api/wire/registry.go` + `cmd/server/main.go:1421` + `api/server_apiv1.go` ;
  `settingsStore.Load()` en erreur se degrade en silence chez deux appelants (dont le cron de purge :
  settings illisible = retention illimitee = purge desactivee sans un mot). Reprise : WARN.
  **TRAITÉ le 2026-09-07 (Q8)**, verifie sur pieces : `api/wire/registry.go` (`retentionMoisRejeu`)
  loggait DEJA en WARN avant degradation (regle n°3 deja appliquee, rien a faire) — les deux
  appelants reellement silencieux etaient `cmd/server/main.go` (fermeture du cron de purge
  `replayPurgeCron`, ~ligne 1420) et `internal/api/server_apiv1.go` (`instanceLockedFn`, ligne
  314) ; les deux loggent desormais en WARN avant de degrader.

## Fusion `origin/feat/v75` -> `feat/tactique` (worktree `LevelUp-wt-tactique`)

- 2026-09-07 ; `apps/go-api/internal/sync/replayartifacts/raster.go:301` (`projeterRastersTactiques`) ;
  v75 a remplace toute la chaine de bursts (`b.usage []artefactCuit`) par le pipeline `Deriver`
  (`derivations.go`, `artefactLu` avec `doc *replay.ReplayDocument` deja parse) — la fusion a adapte
  la signature (`[]artefactCuit` -> `[]artefactLu`) et cable l'appel dans `Deriver` apres
  `persisterStatsBombe`, mais `ProjeterRasterTactique` continue de relire le fichier via
  `lireDocumentRange(path)` au lieu de reutiliser `a.doc` deja en memoire : DOUBLE LECTURE/PARSE de
  chaque artefact par cycle (une fois par `Deriver.lireArtefacts`, une fois par la projection des
  rasters). NON TRAITE (fusion = brancher, pas optimiser) — a regrouper avec la note deja au registre
  §7 du plan tactique sur le document qui devrait circuler entre projections.

- 2026-09-07 ; `apps/go-api/internal/platform/duckdb/tactical_repo.go` (`QTacticalPositions`) et
  `tactical_repo_isolement.go` (`QTacticalIsolement`) ; v75 a pose (lot 3, 2026-09-06) le
  garde-rail `kill_measured_guard_test.go` qui interdit tout littéral `kill_positions(_latest)`
  hors de `kill_measured.go` — les deux requêtes tactique nommaient la table en dur et faisaient
  rougir `TestJointureMesureeUneSeuleFois` apres la fusion. ADAPTE (pas une decouverte a
  consigner sans agir — la ratchet de v75 l'exige) : les deux templates SQL prennent desormais
  `%s` a la place du littéral, et les deux sites d'appel passent la constante non exportee
  `positionsAtKill` de `kill_measured.go` (meme paquet `duckdb`). AUCUNE reutilisation de
  `measuredKillsQuery` : ces deux lectures n'ont ni classificateur d'arme ni garde d'unanimite
  sur `source_tag` (documente dans l'en-tete de tactical_repo.go, « CE QUI DIFFERE DE
  KillDistanceRepo ») — seul le NOM de table est partage, pas la jointure entiere.

## Phase 5 — vue d'analyse (branche `feat/tactique`, worktree dedie, HEAD `a395fa78f`)

- 2026-09-07 ; `apps/web/src/lib/api/types.ts` (schema `TacticalRaster`) ; `matchs_sans_rayon`
  et `morts_equipe_a_terre` sont publies par le contrat mais non consommes par la vue (hors
  perimetre du brief 5.2-5.6). `matchs_sans_rayon` ressemble a la meme reserve que
  `Couverture.echantillon_faible` (matchs dont la variante n'a pas de rayon connu, ecartes de
  la mesure d'isolement) — pourrait justifier une note de couverture sur la tuile KPI
  « Morts en isolement ». **`matchs_sans_rayon` TRAITÉ le 2026-09-07 (Q8)** : note
  `tactical.kpi.no_radius_note` rendue sur la tuile Isolement quand `> 0`, testee. `morts_
  equipe_a_terre` reste NON TRAITE (hors perimetre Q8, aucun item ne le cite).
- 2026-09-07 ; `apps/web/src/features/tactical/TacticalPlanCard.tsx` ; le canvas est peint a
  `k = 1` (pas de `devicePixelRatio`), plus simple que `heatmapLayer.ts` (`k = dpr`, cf.
  `useReplayHeatmap.ts`) — bords de cellule moins nets sur ecran haute densite. Choix de
  portee assume pour cette passe, pas un defaut fonctionnel. NON TRAITE.
- 2026-09-07 ; `apps/web/src/features/tactical/TacticalPage.test.tsx:183` ; le test « la carte
  selectionnee dans l'URL est marquee comme telle » attendait encore l'ANCIEN comportement
  (grille + vignette « pressee ») alors que le placeholder de la phase 5 (commit `a395fa78f`)
  avait deja pose le retour anticipe qui affiche `TacticalAnalysisView` a la place de la
  grille — cassE des ce commit, non detecte a l'epoque. CORRIGE dans cette passe (bloquait le
  gate de la feature qu'elle teste).
- 2026-09-07 ; `apps/web/src/lib/query/keys.title-slug.guard.test.ts` ; la fabrique
  `tacticalRaster` (ajoutee par le placeholder, commit `a395fa78f`) n'etait classee ni
  title-scopee ni agnostique — completude du garde-rail rompue des ce commit, non detectee a
  l'epoque. CORRIGEE dans cette passe (bloquait le gate).
- 2026-09-07 ; `apps/go-api/internal/platform/duckdb/tactical_repo_isolement.go`
  (`QTacticalIsolement`/`MortsAvecContexte`, posee en 7C) ; aucun test `:memory:` DEDIE a
  cette requete — elle n'est exercee que par ses consommateurs (service Tactique existant,
  et desormais le nuage Escouade du lot 7.7/7B) via mock du port. NON TRAITE (hors perimetre
  de 7.7 ; le lot 7.7 n'ajoute aucune SQL, il reutilise la lecture telle quelle).

## Lot Q8 — cloture reelle de la phase 8 (branche `feat/tactique-cloture`)

- 2026-09-07 ; `cmd/server/main.go:722` ; un TROISIEME appelant silencieux de
  `settingsStore.Load()` : `if s, lerr := settingsStore.Load(); lerr == nil { analysis.
  SetExcludeAssistsFromYield(s.RendementExcludeAssists) ... }` degrade en silence (aucun WARN
  sur `lerr != nil`) — hors perimetre du brief Q8 (qui ne citait que `registry.go`,
  `main.go:~1421` et `server_apiv1.go`). NON TRAITE, decouvert en verifiant sur pieces les
  autres appels de `settingsStore.Load()` dans `cmd/server/main.go`.

## Lot Q7 — fusion du peintre de chaleur (branche `feat/peintre-chaleur-unique`)

- 2026-09-07 ; `apps/web/src/features/match-replay/layers/heatmapLayer.guard.test.ts` ;
  son en-tete documente encore « planté dans `heatmapLayer.ts` comme dans
  `ReplayHeatmapLegend.tsx` » — `heatmapLayer.ts` a disparu avec Q7 (noyau deplace vers
  `lib/replay/heatPaint.ts`), mais ce garde ne teste QUE `ReplayHeatmapLegend.tsx` (aucun
  import du fichier disparu) : il reste vert et pertinent, seul le commentaire est perime.
  NON TRAITE (renommage/reecriture du commentaire hors perimetre Q7 — fix cosmetique sans
  gate a risque). Reprise : prochain lot qui touche ce fichier, ou toilettage documentaire.

## Lot M1 — lien « voir dans le rejeu » depuis une cellule (branche `feat/tactique-lien-rejeu`)

- 2026-09-07/08 ; `apps/go-api/internal/domain/tactical_cellule.go` (doc de
  `TacticalContribution.InstantMs`) + `apps/web/src/features/tactical/tacticalView.logic.ts`
  (doc de `instantToFrame`) ; pour les questions `morts`/`kills`/`gagne`/`isole`,
  `instant_ms` est sur l'horloge du MATCH (`match_kill_events.time_ms` /
  `match_death_context.time_ms`), DISTINCTE de l'horloge du FILM que consomme `?frame=`
  (`analysis/replay/lives_export.go` etablit `horlogeFilm = horlogeMatch + DeathOffsetMS`,
  decalage PAR MATCH mesure de 3,6 a 50,8 s, non publie dans le contrat). Seules `temps` et
  `routes` (instant tire du sidecar de raster, deja en horloge film) donnent une frame
  EXACTE. Decision du lot (assumee, pas un defaut a corriger dans ce lot) : le lien est
  construit pour LES SIX questions plutot que d'en priver quatre sur six, avec cette reserve
  documentee. Condition de reprise : publier `DeathOffsetMS` par match dans le contrat (ou
  une frame deja convertie cote Go) pour que `morts`/`kills`/`gagne`/`isole` ouvrent le rejeu
  a l'instant exact.

  **TRAITE le 2026-09-08 (lot M1b, decision utilisateur ferme « corriger le decalage »).**
  `coverage.bridge.deathOffsetMs` publie cote Go (pointeur, additif, bump SchemaVersion
  48->49 pour la reprise du backfill) ; le contrat `TacticalContribution` publie `clock`
  (`"match"`/`"film"`) par question au lieu de laisser le web deviner ; le lien devient
  `?t=<instant_ms>&clock=<clock>` (le service ne convertit jamais lui-meme, il n'a pas
  toujours l'artefact) ; la route du rejeu attend le document, convertit
  (`lib/replay/replayLogic.resolveTacticalReplayInstant` + `resolveTacticalOpenAtFrame`,
  fonctions pures testees) et positionne `ReplayCanvas` via `openAtFrame` — offset inconnu
  (artefact < 49, ou pont qui n'a apparie aucune mort) : le rejeu s'ouvre au debut et affiche
  un avis FR/EN, jamais un saut approximatif presente comme exact. `instantToFrame` retire
  (mort). Decouverte au passage : `?frame=` de M1 n'etait PAS cable cote route (`replay.tsx`
  n'avait aucun `validateSearch` ni lecture de la query) — le lien de M1 n'ouvrait donc jamais
  le rejeu au bon instant, meme pour `temps`/`routes` ; corrige dans ce meme lot
  (`openAtFrame` sur `ReplayCanvas`/`useReplayPlayback`). Detail :
  `.ai/V7.5/v2/CHRONIQUE_49_2026-09-08.md`. Reste ouvert : le parc d'artefacts anterieur au
  schema 49 n'a pas ce calage — recuisson necessaire (`.ai/V7.5/REGISTRE_REPORTS.md`).
