# Plan — suite du chantier Tactique (trois restes consignes) — 2026-09-07

Perimetre FERME : trois items, rien d'autre. Toute decouverte va dans
`.ai/DECOUVERTES_TACTIQUE_2026-09-07.md`, jamais dans le lot. Executeur : Sonnet, brief ferme,
pas de sous-agent, gates au premier plan ; une relecture ciblee par item, sur pieces, par le
superviseur (pas de relecteur adverse : lots d'interface et de contrat, faible risque).
Contrat d'execution : skill `plan-execution`. Branche : `feat/tactique` (ou une branche
`feat/tactique-suite` issue de `feat/v75` apres la fusion du chantier, au choix de l'utilisateur).

> AMENDEMENT 2026-09-07 (soir, `.ai/PLAN_ORCHESTRATION_2026-09-07.md`) : P.1 est FAIT (fusion
> `f2c8ddce1`). S.2 : le peintre partage n'est PAS dans `lib/replay/` mais dans
> `features/match-replay/layers/heatmapLayer.ts` — S.2 DEPLACE le noyau dans `lib/replay/` puis
> supprime `features/tactical/heatPaint.ts` (lot Q7 du plan d'orchestration). S.3 est GELE
> derriere l'inventaire P1 du plan v2 (axes ROSTER et TEMPS) : faire S.3 avant P1 fabriquerait
> une deduction que le paradigme condamne (decision D10). S.1 = lot M1 (vague 2). Branche :
> une branche `feat/` neuve par lot, issue de `feat/v75`.

## Pre-requis commun — reprendre `feat/v75` (lot D present) — FAIT (`f2c8ddce1`)

`feat/v2-web-modele` (lot D de l'audit : lecture du rejeu pilotee par `?frame=` via le
`playbackStore`, peintre de chaleur dans `lib/replay/`) est FUSIONNE dans `feat/v75` ; `feat/tactique`
ne l'a pas encore.
- [ ] P.1 `git merge origin/feat/v75` dans `feat/tactique` ; conflits attendus : aucun sur
      `features/tactical/` (dossier propre au chantier) ; possibles sur `features/match-replay/`
      (le chantier n'y touche pas : prendre `feat/v75`), `lib/query/keys.ts`, `lib/api/generated.ts`
      (regenerer : `go run ./cmd/openapi-gen` puis `make generate-types`), `.ai/thought_log.md`
      (garder les deux).
- **Gate** : typecheck, vitest complet, `go test ./...`, CI verte.

## S.1 — Lien « voir dans le rejeu » depuis une cellule (item 5.5 + moitie de 5.6)

Ce que le contrat porte deja : par cellule, la valeur et le nombre de matchs contributeurs ; pour
`temps`/`routes` le sidecar connait l'instant contributeur (`frame` de premiere entree /
debut de vie) ; pour `morts`/`kills`/`gagne` le journal porte `time_ms`.
- [x] S.1.1 Contrat : `POST .../tactical/{map_id}/cellule` (ou champ optionnel de la reponse
      raster sur demande `cellule: {col,row}`) qui rend, pour UNE cellule, la liste des
      contributions `{match_id, instant_ms, xuid}` **filtree par l'ownership XUID** (ADR 0029 :
      seuls les matchs ouvrables par l'appelant) + `matchs_non_ouvrables` (compte). Service :
      lecture depuis le journal (`kill_positions_latest` / `match_kill_events_latest`) pour les
      questions de base, depuis les sidecars pour `temps`/`routes` (le sidecar porte `frame` par
      spawn/route ; pour `temps`, l'instant de premiere entree est dans `Raster` — verifier ce
      que le schema 6 publie, sinon consigner « temps sans instant »).
      PREUVE (lot M1, branche `feat/tactique-lien-rejeu`) : endpoint
      `POST /players/{player_slug}/tactical/{map_id}/cellule` (`api/openapi.yaml`,
      schemas `TacticalCelluleBody`/`TacticalCelluleReponse`/`TacticalContribution`) ;
      domaine `internal/domain/tactical_cellule.go` ; port
      `TacticalRepository.MatchsOuvrables` (`internal/port/tactical.go`) implemente par
      `internal/platform/duckdb/tactical_repo_ownership.go` (EXISTS sur
      `match_participants`, meme garde que la Couche B ADR 0029) ; service
      `internal/service/tactical_service_cellule.go` (dispatch morts/kills/gagne ->
      `KillPositions`, isole -> `MortsAvecContexte`, temps/routes -> sidecars, `TimeMs`
      toujours present au schema 6 courant — aucun cas de « temps sans instant »
      rencontre, note dans le fichier). Filtrage d'ownership APRES lecture (defense en
      profondeur), tri date desc puis instant croissant.
- [x] S.1.2 Web : `TacticalCellCard` liste les contributions (match, date, instant) avec un lien
      vers la route du rejeu et `?frame=` (modele D : lire comment `replay.tsx` consomme le
      parametre apres le lot D — `playbackStore`), `footer` = « N matchs comptes non ouvrables ».
      PREUVE : `apps/web/src/features/tactical/TacticalCellCard.tsx` (liste de contributions,
      lien `?frame=` via `router.buildLocation` + `instantToFrame`, footer conditionnel
      `matchsNonOuvrables > 0`) ; hook `useTacticalCellule` (`queries.ts`, requete seulement
      si une cellule est selectionnee) ; cle `queryKeys.tacticalCellule` (`lib/query/keys.ts`,
      title-scopee, garde `keys.title-slug.guard.test.ts` mis a jour) ; wiring dans
      `TacticalAnalysisView.tsx` ; i18n FR/EN complete (`tactical.toml` +
      `generated/tactical.ts`).
- [x] S.1.3 Tests : service (ownership : un match d'un autre joueur n'apparait pas mais compte),
      handler, logique pure web (instant -> frame), rendu de la carte.
      PREUVE : Go — `internal/service/tactical_service_cellule_test.go` (ownership,
      filtre de cellule, trois faces morts/kills/gagne, isole, temps, routes, tri, refus
      carte/question/capability) ; `internal/platform/duckdb/tactical_repo_ownership_test.go`
      (`:memory:`, MatchsOuvrables mien/tiers, date canonique, doublon, listes vides) ;
      `internal/api/handlers/tactical_cellule_test.go` + `tactical_mapid_test.go`
      (nominal, defauts, 404 carte inconnue, 503 capability, formes hostiles de map_id).
      Web — COMPLETE dans cette reprise (les tests Go et le contrat existaient deja, la
      logique pure et le rendu de carte manquaient) : `instantToFrame` teste dans
      `tacticalView.logic.test.ts` (pas par defaut, pas explicite, arrondi, bornes 0/negatif) ;
      rendu ajoute dans `TacticalCellCard.test.tsx` (placeholder, chargement, vide, lien
      `?frame=` construit, ordre des liens, footer conditionnel 0 vs >0).
- **Gate** : `?frame=` positionne le rejeu (test de la route, deja couvert par le lot D —
  `replay.gate.test.tsx`) ; filtrage d'acces teste ; typecheck + vitest ; `go test`
  service/handler ; contrat regenere. TOUS VERTS le 2026-09-08 (voir thought_log).

## S.2 — Un seul peintre de chaleur (item 5.1, fusion avec D.13) — FAIT (lot Q7, 2026-09-07,
branche `feat/peintre-chaleur-unique`)

- [x] S.2.1 Deplace (pas juste remplace, cf. AMENDEMENT en tete de fichier) le noyau
      `features/match-replay/layers/heatmapLayer.ts` (buildHeatmap/drawHeatmapLayer/heatRamp)
      vers `lib/replay/heatPaint.ts`, y ajoute l'entree « cellules pre-agregees »
      (buildTacticalGrid/drawTacticalHeatmap, ex-`features/tactical/heatPaint.ts`) ; les DEUX
      fichiers copies sont SUPPRIMES (le rejeu et le tactique importent desormais le noyau
      commun ; `drawHeatmap` interne partage la fusion de plages + alignement pixel, parametre
      par `HeatSource`/`HeatGeometry` pour que le Y-flip du rejeu et son absence cote tactique
      restent chacun dans leur adaptateur).
- [x] S.2.2 Garde-rail `lib/replay/heatPaint.guard.test.ts` : grep (`import.meta.glob`) sur
      les cinq noms (`buildHeatmap`, `drawHeatmapLayer`, `drawTacticalHeatmap`,
      `buildTacticalGrid`, `heatRamp`) — rouge si definis hors de `heatPaint.ts`, et
      self-check positif qu'ils y sont bien. Mutation jouee (copie de
      `drawTacticalHeatmap` dans `features/tactical/` -> rouge -> retiree -> vert).
- **Gate** : rendu identique — `tacticalGridFromRaster` (4 tests ajoutes, aucun test
  existant n'en avait avant Q7) + snapshot leger d'intensites pour les deux entrees ;
  `heatmapLayer.test.ts` migre tel quel vers `lib/replay/heatPaint.test.ts` (memes
  assertions, `view` de `drawHeatmapLayer` recalculee via les memes primitives
  `worldToCanvas`/`canvasScale` puisque le noyau n'importe plus `CanvasView`) ; typecheck
  (`tsc -b`) vert ; vitest complet 653 fichiers / 6979 tests verts (18 skip pre-existants) ;
  `crossFeatureBoundary.guard` vert ; garde-rail Q7 vert. Decouvertes : voir
  `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md` (section « Lot Q7 »).

## S.3 — Arrivees et departs dans le contexte de mort (premiere entree du registre)

En V1, `match_death_context` ne lit pas la participation (`teammates_left` = 0) parce que ses
instants sont sur l'horloge de l'API (`start_time`) et le journal sur celle du film (frame 0),
avec un T0 de 17 a 92 s selon les matchs.
- [ ] S.3.1 Calage : reutiliser l'appareil existant (`match_registry.real_start_time`,
      `t0_quality`, `cmd/backfill_t0_film`, `t0_film.go`) — verifier sur pieces lequel est
      disponible AU SYNC dans le collecteur ; ramener `first_joined_time`/`last_leave_time` sur
      l'horloge du film ; sans calage fiable pour un match : participation NON appliquee et
      compteur `killsource_isolement_matchs_sans_calage`.
- [ ] S.3.2 `death_context.go` : present = arrive <= t < parti ; absent hors de tout etat et du
      total ; `teammates_left` reel. Passe `decoder_rev` montee -> le rattrapage
      `backfill-killsource` reecrit les contextes.
- [ ] S.3.3 Tests : present des le debut (first_joined = start + T0) et mort a 25 s film ->
      present ; arrivee en cours calee ; match sans calage -> personne exclu + compteur ;
      mutation « comparer sans calage » tombe.
- **Gate** : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` ;
  `no_art_patterns_test` ; 2 relecteurs si `internal/sync` est touche (regle du skill).

## Ordre et clôture

P -> S.1 -> S.2 -> S.3 (S.3 est independant et peut passer avant S.2 si le sync est prioritaire).
Cloture : thought_log, CI verte, `make gate-push`, fusion par l'utilisateur.
