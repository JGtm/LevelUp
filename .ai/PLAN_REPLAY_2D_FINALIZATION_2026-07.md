# PLAN — Finalisation du Replay 2D (rejeu de match vue du dessus)

> Créé 2026-07-11. Branche `feat/filmdec-continuation` (worktree dédié).
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape close avant la
> suivante, aucun report d'action faisable, statut sur chaque item, gate + thought_log +
> point d'étape à chaque clôture).

## Objectif

Transformer la percée RE (trajectoires tous-joueurs décodées, carte 2D depuis `.module`,
arme-par-kill ~96%) en une **vraie feature de visualisation end-to-end** : remplacer le
stub `replay.tsx` par un rejeu 2D vue du dessus (carte + traînées joueurs + kills) piloté
par une timeline, servi par un endpoint, activé par disponibilité d'artefact par-match.

## Contrainte dure (actée, non maquillée)

Les trajectoires fidèles proviennent aujourd'hui d'une **capture CE live** — donnée
décodée disponible pour **1 match** (`000d5950`, Cliffhanger : `.ai/re_dump/
allbipeds_capture_sample.txt`, 8 slots, W=15, décodage VÉRIFIÉ le 2026-07-11 : 2000 rec,
17 desync 0.85%, trajectoires lisses pas moyen 0.08 max 0.23). Le décodage pur-offline
depuis les chunks téléchargés (match quelconque) reste le mur RE documenté (largeurs de
composants runtime = CE). => La feature s'ALLUME par-match quand un artefact existe ;
dégradation gracieuse sinon. Pipeline construit pour que tout futur artefact (nouvelle
capture ou point-5 cracké) traverse le même chemin. **Pas de flag global OFF « pour plus
tard »** (règle 11 projet) : gating = présence d'artefact.

## Décisions fermes (ne pas re-litiger)

- D1. Rendu front = **canvas 2D** (perf animation ; pas ECharts — ce n'est pas un chart au
  sens ADR 0001, c'est une scène spatiale animée). Wrapper isolé, logique hors composant.
- D2. Source trajectoires = artefact `replay.json` généré offline par un outil Go dédié
  (`cmd/replay-build`), extrayant la logique VALIDÉE de `tmp_kfmatch` (pas de throwaway).
- D3. Repère : positions dans le repère monde quantifié partagé (les 8 joueurs cohérents
  entre eux). Alignement carte = normalisation [0,1]² + meilleure des 8 orientations
  (approche `tmp_mapoverlay` déjà validée 84%). Map overlay = enrichissement (étape B), pas
  bloquant pour un premier rejeu.
- D4. Endpoint Huma `GET /players/{slug}/matches/{matchId}/replay` → JSON artefact ;
  handler mince, assemblage en analysis (arch-rules). Ownership : replay = donnée de match
  partagée, **pas** ownership-gated (cf `/careerranks`), mais 404 propre si pas d'artefact.
- D5. Gating front = le bouton/route « Rejeu 2D » n'apparaît que si l'API annonce un replay
  disponible (probe léger). Retrait du flag `REJEU_2D_ENABLED` et du commentaire « projet
  externe ».

## Étapes

### Étape A — Rejeu minimal end-to-end (trajectoires animées) [GATE: le rejeu s'anime en local]
- [x] A1. Package `internal/analysis/replay` : `ReplayDocument` (métadonnées + `[]Track` +
  bornes). Décodage (filmdec) séparé de la reconstruction PURE (absolu fixe / delta
  accumulé), tests `TestParseCapture` + `TestReconstruct` verts (go test replay = ok).
  Temps = index global du record (flux CE ordonné). Z écarté (plan X/Y).
- [x] A2. `cmd/replay-build <matchId> <capture> [filmDir]` : registry + capture →
  `ReplayArtifactPath` (PathResolver, R1). repoRoot via `LEVELUP_REPO_ROOT` (pas d'import
  `config`/CGO). VÉRIFIÉ 000d5950 : 8 tracks, 1992 pts, 2000 frames, bounds X[-67,70]
  Y[448,581], artefact 76 Ko cohérent avec le décodage tmp_kfmatch.
- [ ] A3. Endpoint `GET /players/{player_slug}/matches/{match_id}/replay` (404 si absent).
  CORRECTION archi (le code fait foi) : cette branche est ANTÉRIEURE à la migration Huma —
  routing = **chi + `writeJSON`/`writeError`** (pas Huma). Exemplaire cloné =
  `handlers/match_view_positions.go` (GetMatchPositions). Params **snake_case**
  (`{player_slug}`/`{match_id}`, l'ownership lit `chi.URLParam(r,"player_slug")`). Route
  montée sous le groupe `/players/{player_slug}` (server.go ~744, ownership transparent en
  mono-user, 404 sur artefact absent). Garde-rail réel = `contract_test.go` (chi↔openapi) →
  ajouter la route à `api/openapi.yaml`. `make generate-types` racine = STUB : la vraie
  gen TS = `cd apps/web && npm run generate-types` (openapi-typescript). CAVEAT MERGE :
  au merge vers main (qui a Huma), porter l'endpoint en Huma. Petit `ReplayService`
  (testable) lu via PathResolver.
  FAIT : `port.ReplayService` + `service.NewReplayService` + `handlers.ReplayHandler` +
  factory `reg.Replay` + route server.go:745 (hors sous-groupe capability) + path
  openapi.yaml (schéma inline). Tags JSON du document passés en camelCase (convention API).
  Tests handler verts (200/404 absent/404 joueur/500) + build CGO complet vert.
- [x] A4. Front : stub remplacé par `ReplayCanvas` (canvas vue du dessus, dots + traînées,
  play/pause + scrubber + vitesses). Query key `matchReplay`. i18n FR+EN. Type
  `ReplayDocument` hand-written dans `lib/api/types.ts` (convention `MatchPlayerPosition`,
  pas openapi-gen). Logique pure `replayLogic.ts` + 13 tests vitest verts ; tsc 0 erreur.
- [~] A5. Revue visuelle **PASSÉE en live** (worktree server+vite vs données réelles,
  JGtm/000d5950) : 8 joueurs animés, endpoint 200/404 OK. Reste (différé — user gate
  "avant activation") : retrait flag `REJEU_2D_ENABLED` (mort, 0 import) + point d'entrée
  discoverable depuis le match-view. À faire quand user valide "satisfaisant".
- GATE A : tsc + vitest + go build/tests verts ; **rejeu 000d5950 animé confirmé au
  navigateur** (3 captures). Reste flag+entrée pour clôturer A.

### Étape B — Overlay carte 2D [GATE: la géométrie Cliffhanger s'affiche sous les trajectoires]
- [ ] B1. Vérifier extraction géométrie Cliffhanger (`.module` local) via `himodule`/
  `tmp_geores` ; sinon statuer la source. Émettre polygones/points map dans `replay.json`.
- [ ] B2. Alignement trajectoires↔carte (normalisation + orientation) côté build (une fois),
  coords finales dans l'artefact.
- [ ] B3. Front : rendre la carte sous les trajectoires.
- GATE B : carte visible, trajectoires cohérentes dessus (revue visuelle).

### Étape C — Kill feed + arme [GATE: les kills apparaissent au bon instant avec l'arme]
- [ ] C1. Source kills : `chunk_27` (tueur/victime/temps) + arme warp (~96%). Vérifier
  l'outil/chemin existant (`README_KILLWEAPON_INDEX.md`) avant de coder (règle 14).
- [ ] C2. Émettre `[]KillEvent{t, killer, victim, weapon?}` dans l'artefact.
- [ ] C3. Front : flash/marqueur au temps du kill + libellé arme (asset FR).
- GATE C : kills positionnés dans le temps, arme affichée où dispo.

### Étape D — Identités & équipes [GATE: joueurs nommés/colorés par équipe si faisable]
- [ ] D1. Attribution slot→gamertag (record spawn xuid OU ordre roster `PLAYER_METADATA`) ;
  si infaisable proprement, rejeu anonyme + couleurs d'équipe inférées. Statuer.
- [ ] D2. Front : couleurs équipe + noms/tooltip.
- GATE D : cohérence identités OU anonymat assumé documenté.

### Étape E — Productionisation & livraison [GATE: delivery-checklist vert]
- [ ] E1. Nettoyage : pas de code mort, seuils fichier/fonction, logging slog, tests Go du
  package replay. Doc `internal/analysis/replay/README` si pattern réutilisable.
- [ ] E2. Bilingue docs si guide majeur touché ; ADR si décision d'archi (endpoint replay).
- [ ] E3. `delivery-checklist` complet ; thought_log ; point d'étape final. PAS de merge
  main sans feu vert user (deploy auto + revue visuelle au merge).

## Addendum revue de plan (corrections `plan-review` — CONTRAIGNANT)

Intégré 2026-07-11 après passage `plan-review`. Ces règles s'appliquent à toutes les étapes :

- R1. **PathResolver obligatoire.** L'artefact vit sous un chemin résolu par
  `title.PathResolver` (pas de `filepath.Join(..., "data", ...)` en dur). Ajouter la
  méthode de résolution `ReplayArtifact(matchID)` si absente. Idem lecture du cache de
  films (réutiliser le resolver existant des outils film).
- R2. **Gating title-agnostic.** Disponibilité = présence d'artefact + capability film
  (`HasCapability`, jamais `slug == "halo_infinite"`). H5/autres titres sans artefact =
  404 propre, aucune fuite d'un autre titre.
- R3. **Couleurs = tokens.** Le canvas a besoin de valeurs concrètes : résoudre les tokens
  sémantiques via `getComputedStyle` des variables CSS (équipe/joueur/kill), **aucun hex
  ni classe Tailwind couleur** en dur dans les composants (skill `color-tokens`). Un
  helper `resolveToken(name)` centralise.
- R4. **Tests par couche.** A3 inclut un test `httptest` (404 sans artefact / 200 avec).
  A1/E1 : tests unitaires du package `replay` (reconstruction absolu/delta, bornes).
  A4 : logique d'animation/scrubber extraite en `replay_logic.ts` + test vitest (pas de
  logique dans le composant — anti-pattern 7).
- R5. **Logging.** Code bibliothèque (`internal/analysis/replay`, endpoint) = `slog.*Context`
  structuré, jamais `fmt.Println`. Les `cmd/replay-build` (CLI) peuvent imprimer en stdout
  (sortie outil), mais loggent les erreurs via slog.
- R6. **Labels.** Libellés d'arme via les hooks de label existants + `asset_translations`
  (FR), pas de littéral. Réutiliser l'existant (`README_KILLWEAPON_INDEX.md`, skill
  `go-features`) avant d'écrire du neuf.
- R7. **Type de retour.** `ReplayDocument` = DTO d'artefact bespoke (assumé, pas canonical) ;
  il n'entre pas dans `canonical/`. Documenté dans le README du package (E1).
- R8. **Gates B/C/D** ajoutent aux critères visuels : `make check-types` + `go build ./...`
  verts (pas seulement « ça s'affiche »).
- R9. **Reprise de session** : source de vérité = ce fichier (cases + Journal). Reprendre à
  la 1re case non statuée de l'étape courante (contrat `plan-execution` règle 10).

Effort estimé : A moyen (nouveau front+endpoint+outil), B moyen (RE géométrie déjà faite,
reste assemblage+alignement), C moyen (arme existe, reste jointure temps), D léger-moyen
(attribution incertaine), E léger. Risque décroissant après A.

## Découvertes (hors périmètre — noter, ne pas traiter)

- [2026-07-11] Le helper de résolution du cache film pour tests (worktree→main tree via
  marqueur `/.claude/worktrees/`) est DÉJÀ dupliqué : `positions/positions_test.go` +
  `weaponv3/pi_resolver_test.go`. Une 3e copie déclencherait la règle 6 (centraliser +
  garde-rail). Évité en A1 (tests purs, validation données réelles via cmd A2). À
  centraliser (ex. `internal/analysis/filmtestutil`) si un 3e besoin apparaît — NON traité.

- [2026-07-11] DETTE CONTRAT PRÉ-EXISTANTE : `api/openapi.yaml` ne documente PAS les
  sous-routes film `/matches/{match_id}/{neighbors,objective-events,positions}` (grep=0),
  alors que `contract_test.go` (`TestContractRoutesDocumented`, seuil 0, cgo/integration)
  exige que TOUTE route chi soit documentée → le ratchet contrat est probablement DÉJÀ
  rouge sur cette branche. Non traité (hors périmètre, règle 7). Ma route `/replay` EST
  documentée. À réconcilier avec les autres à l'Étape E ou par le mainteneur.

## Journal d'exécution

- [2026-07-11] Plan créé. Décodage trajectoires re-vérifié sur pièces (tmp_kfmatch,
  000d5950) : OK. plan-review passé, addendum R1-R9 intégré. Prochaine action : Étape A1.
- [2026-07-11] Rappel user : « redessiner les maps depuis les objets/modules du jeu en
  local » = coeur de l'Étape B (extraction géométrie `.module`, déjà résolue offline,
  mémoire project_map_geometry_from_modules). Confirmé au périmètre, non différé.
- [2026-07-11] REVUE VISUELLE LIVE (basculement dev server sur le worktree vs données
  réelles du main checkout, autorisé user). Backend : GET .../000d5950/replay = 200 (8
  tracks) ; match absent = 404 replay_not_available. Front : 8 joueurs animés vue du
  dessus, couleurs tokens distinctes, traînées, timeline scrub + vitesses. FIX en cours de
  revue : `$matchId.tsx` n'avait pas d'`<Outlet/>` → route `/replay` masquée par
  MatchViewPage ; converti en **layout + index** (calqué sur `squad.tsx`/`squad/index.tsx`,
  MatchViewPage déplacé en `$matchId/index.tsx` inchangé, loader déplacé). Traînées
  renforcées (alpha 0.5, largeur 2.5, cap rond, fenêtre 260). User explore avant commit ;
  serveur worktree laissé actif (:5173/:8000). Décision rebase : PORTAGE EN AVANT (main
  +1231 commits + Huma ; filmdec/positions branch-only) — pas de rebase du labo RE.
- [2026-07-11] EN ATTENTE user : validation "satisfaisant" du replay → puis commit +
  activation (flag + point d'entrée). Restaurer `make dev` user sur demande.
