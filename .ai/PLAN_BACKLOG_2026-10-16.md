# Plan — backlog du 2026-10-16 (les 12 entrées actives de `.ai/BACKLOG.md`, périmètre fermé)

Origine : demande du user du 2026-10-16 (« 12 éléments en attente, analyser, vérifier, planifier ;
périmètre fermé ; la release ensuite »). Les 12 entrées sont celles de `.ai/BACKLOG.md` du commit
`aae0a6c6f`, de « [ci/durée] » à « [déploiement] ». Hors périmètre, inchangées : fins multi-équipes et
blocs Élimination/Infection (bloqués faute de donnée), `weapon_labels` H5 (relevé live), Tauri (V9).

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/backlog-2026-10-16`, worktree
`LevelUp-wt-backlog-1016`, partie de `origin/feat/v75`. Superviseur : session principale. Exécutants :
Sonnet (`model` explicite) ; Opus seulement pour E8.4 (identité des vies), annoncé au user avant.
Fusion dans `feat/v75` : geste du user. Push `main` : jamais.

Ordre strict E1 → E10. Une étape est close quand son gate est vert, chaque item statué (`[x]` /
`[~]` réf / `[!]` justifié), ce fichier à jour, une entrée au thought_log, `.ai/BACKLOG.md` à jour
(user, 2026-10-16 : au fil de l'eau, pas en fin de plan — l'entrée traitée est retirée ou réécrite
pour ce qui reste, et une ligne datée est ajoutée en tête du tableau « Récemment complété »), un
commit préfixé par l'étape (`b1016(E<n>): ...`). Découvertes hors périmètre : section 3, non traitées.

Environnement : `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`,
`GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-backlog1016`. Jamais `go build ./...`. Un seul `go` à
la fois. Aucune cuisson de rejeu en lot sans accord du user (bombe RAM). Clôture :
`scripts/disk-hygiene.ps1 -Lot backlog1016 -Apply`.

Reprise de session : lire la section 4 (journal), `git log --oneline -15` du worktree, reprendre au
premier item sans statut.

## 0. Vérification sur pièces (2026-10-16, superviseur + 4 lecteurs Sonnet)

| # | Entrée | Prémisse | Constat principal |
|---|---|---|---|
| 1 | CI 50 min | vraie | Run `38963177635` (feat/v75, vert) : 49 min. Go couverture 41 min + second passage `internal/sync` 7 min (en double), Vitest 11,5 min, tout le reste ≤ 5 min. Compilation sérialisée par `-p 1` : ~6,5 min. Le mémo de bobines du point 6 EXISTE déjà (`grammar/bobines_memo_test.go`, 2026-09-28) ; le paquet `grammar` reste à ~870 s sous couverture, dont `TestSuivanteEgaleLaBoucleDAvantSurTamponsSynthetiques` 233 s (tampons synthétiques, pas de bobine). Les paquets `film/...` n'ont aucune dépendance DuckDB, tests compris (`go list -test -deps`). Aucune vérification requise sur `main` (ruleset : suppression + non fast-forward seulement) ; `deploy.yml` attend la conclusion du run `ci.yml` entier. Couverture de référence : 81,0 %. |
| 2 | Tactique ~96 requêtes | vraie, mécanisme établi | 2 requêtes par carte ouvrable : `GET /tactical/{map}/background.png` (`TacticalMapTile.tsx:51`) et `GET /tactical/{map}/background` (`:134`), sans paresse ; + 4 fixes. `/tactical/maps` porte déjà les mini-plans. |
| 3a | Participation 503 | partiellement fausse | Pas de budget propre : 3 s d'attente de swap + 5 s dans `IsParticipant` (`match_view_repo.go:392`) ; aucun cache ; 503 fermé (`match_participation.go:56-60`). Pas monté sur Tactique. |
| 3b | Code Xbox redemandé | vraie (lecture) | L'effet de récupération `XboxLoginPage.tsx:235-260` relance `start` sur `attempt_not_found` sans tester le statut `authorized`/`provisioned` ; la rotation de session au succès (`auth.go:259-263`) produit ce 404 → jusqu'à 3 `start` → plafond 3/IP (`attempt_store.go:36`). |
| 4 | Arrêt > 10 s | vraie, sous-estimée | Pire cas borné 35 s (15+2+3+5+5+5, `main.go:1357-1430`) + étapes non bornées ; aucun `stop_grace_period` (`docker-compose.yml`, services `levelup` et `levelup-demo`). Commentaire `main.go:1351` périmé (cite `stop_timeout=20s` ; `.air.toml:46` : `kill_delay = "5s"`). |
| 5 | LUSR v1 à l'exclusion | vraie, plus grave | `match_recomputer.go:146` écrit en v1 (`rating_type='LUSR'`, INSERT) : ces lignes plus récentes GAGNENT dans `match_skill_rank_latest` et masquent la v2. La v2 ne filtre PAS `is_excluded` (seule la v1 appelle `LoadExcludedMatchIDs`). Les 8 composantes (`lusr_component_history`, lues par Prestige `campaign_repo.go:416`, profil `queries.go:431`, escouade) ne sont écrites QUE par la v1 (`AddLUSRComponents` sans appelant) : plus alimentées depuis la v2 canonique. |
| 6 | Vies sans identité | vraie, non mesurable sans cuisson | Compteur `coverage.bridge.unnamedLives` (`build_pistes.go:304`). Les 238 rejeux cuits du poste en ont 0 : les 43 films fautifs ne sont pas cuits localement. Règle du bot : `killsource/roster_releve.go:32-110`. |
| 7 | Position à la mort | imprécise | `match_death_context` ne stocke pas la position de la victime ; `ContextesDesMorts` (`death_context.go:178`) prend le dernier échantillon ≤ instant (≤ 1 s, décalé de `DeathOffsetMS`), `kill_positions` le plus proche à ± 120 ms (`killpos.go:263`). Seul `tactical_reel_research_test.go:248` compare (tolérance 0,5 m). |
| 8 | Score KOTH Lattice | vraie | Chaîne `score_timeline.go:91` → `objectives.SeriesTotal` (`score.go:180`, `longestRun` strict : une descente devrait y être filtrée, le défaut est donc ailleurs) ; seuil `hill_hold_threshold.go:99-181`. Films `26602661` et `7de0b91d` présents localement (chunks, faits, rejeu). Plan cité au bon chemin : `.ai/V7.5/PLAN_KOTH_REJEU_2026-10-09.md` D1. |
| 9 | Harvest | vraie | Absente de `map_quant_bounds.json`, de `mapModule` (`cmd/mapquant-build/main.go:175-262`) et de `map_objectives.json`. `map_id` et mvar inconnus localement. Recette recos-a : `24cf38183`, `5c7d2beff`. Témoin `e449a696` : chunks présents, ni faits ni rejeu. |
| 10 | Dette du 15/10 | vraie sauf 2 points | Code mort confirmé (6 symboles). `GetMatchKillSources`/`GetMatchKillAssists` DÉJÀ corrigés. Angle mort du garde de Campagne structurel mais aucun lecteur fautif aujourd'hui. `match_bomb_stats` absent du snapshot confirmé. |
| 11 | Citations | vraie, plus grave | `computeFKFD` (`profile/queries.go:299-311`) lit des colonnes inexistantes (`killer_xuid`, `victim_xuid`) : erreur de liaison avalée → 0/0, pour TOUS les titres. `citations.engine` : seul effet = `feature-matrix` (admin). Storm : Infinite n'a pas de Storm Rifle ; la maîtrise Carabine à impulsion existe au seed (`seed_citation_data.go:726`) MAIS le user ne la voit pas dans la page Citations (2026-10-16). |
| 12 | Recalcul H5 + démo | vraie | `backfill --perf`, `--assists-model`, `--lusr`, `--citations` codés sur `DefaultSlug` ; `POST /backfill/start` sans titre ; seul `--engagement-scores` accepte `--title`. En prod : routes HTTP admin, pas de backfill hôte. |

## 1. Décisions (fermes)

- **D-1** (user, 2026-10-16) CI d'abord (E1), validée par UNE CI sur la branche ; les étapes suivantes
  passent des gates locaux ; une CI finale (E10). Un push intermédiaire reste permis (runs périmés
  annulés par E1.1).
- **D-2** Parallélisme Go en CI = plusieurs MACHINES, chacune en `-p 1` (contrainte DuckDB par
  machine). Aucune modification de test pour gagner du temps, sauf le découpage de `grammar` par
  liste nommée (`-run` / `-skip` complémentaires, même liste, une seule source).
- **D-3** Le dernier lot Go est le COMPLÉMENT calculé (`go list ./...` moins les motifs explicites) :
  l'union couvre `./...` par construction. Le job de fusion vérifie en plus que l'ensemble des paquets
  du JSONL fusionné égale `go list ./...`.
- **D-4** Fusion des profils de couverture par concaténation (une ligne `mode:`) : `go tool cover`
  additionne les blocs identiques. Le second passage `internal/sync` est supprimé ; l'affichage de la
  couverture de `internal/sync` se lit sur le profil fusionné filtré (affichage, pas gate).
- **D-5** `concurrency` par workflow et ref ; `cancel-in-progress` faux sur `main` et `feat/v75`.
- **D-6** (user, 2026-10-16) La v1 du LUSR est RETIRÉE : recalcul d'exclusion par la v2, la v2
  respecte `is_excluded`, les 8 composantes sont écrites par le chemin v2 (calcul de composantes
  conservé, il ne dépend pas de la note v1), opt-out `LEVELUP_LUSR_CANONICAL=LUSR` supprimé, code v1
  supprimé avec ses tests. Le contrat « le recalculateur ne prend jamais l'écrivain partagé »
  (`TestMatchRecomputer_NeverAcquiresTheSharedWriter`) est REMPLACÉ par l'accès en rafales de la v2,
  documenté dans le commit et le test renommé (retrait de la baseline JSONL au même commit).
- **D-7** (user, 2026-10-16) Position de la victime à la mort : source unique = l'échantillon de
  `kill_positions` (`positionOf`), dans le lot film. Si l'unification fait perdre des contextes de mort
  sur le corpus (ambiguïté de corps), STOP et retour au user avec le compte.
- **D-8** Lot film (E8) : une seule montée de révision par couche touchée, une seule recuisson ciblée
  locale (films KOTH, témoin Harvest, films aux vies sans nom), sur accord du user avant lancement. En
  prod, la recuisson suit la séquence de déploiement connue (recuisson → usage-summary → pad-tiers
  --force → killsource), le superviseur prévient le user.
- **D-9** Vies sans identité : correctifs = règles générales validées sur le corpus, jamais un patch par
  film. Arrêt au 2e lot sans baisse du compte. La lecture juste de `3f048bf3` (Lacuna ou Ritzy) est
  tranchée par le user sur pièces avant tout code.
- **D-10** Tactique : agrégat dans `/tactical/maps` (calage du fond par carte) + image servie par
  `<img loading="lazy">` (cache HTTP ETag existant). Si `api.getBlob` porte un en-tête requis par la
  route, repli sur un chargement par IntersectionObserver (même effet : vignettes visibles seulement).
- **D-11** Participation : cache borné des vérifications POSITIVES seulement (fait immuable), clé
  xuid + match ; jamais un `false` ni une erreur. Le 503 fermé reste sur un échec non caché.
- **D-12** Arrêt : `stop_grace_period: 45s` sur les deux services + budget d'arrêt global de 40 s
  propagé à toutes les phases, y compris celles non bornées aujourd'hui.
- **D-13** Storm : en attente du diagnostic E6.3 ; pas de nouvelle citation pour une arme absente
  d'Infinite.
- **D-14** H5 au déploiement : les passes `--perf`, `--assists-model`, `--lusr`, `--citations`
  deviennent title-aware (CLI `--title` et champ `title` de `POST /backfill/start`) ; l'EXÉCUTION en
  prod reste un geste du user, préparée par une séquence écrite au runbook.

## 2. Étapes

### E1 — CI : ~49 min → ~15-20 min sans retirer de test (entrée 1) — M

Pièces : `.github/workflows/ci.yml:450-646` (job `go-coverage`), `:60-185` (`frontend`),
`scripts/check_test_baseline.sh`, `apps/go-api/scripts/coverage_{filter,check}.sh`,
`archlint/ci_deploy_triggers_test.go`, `archlint/ci_workflows_security_test.go`, `Makefile:341`
(`gate-push`). Mesures de référence : JSONL du run `38938482739` (artefact
`go-baseline-current-jsonl`) ; durées par paquet au journal (section 4).

- [x] E1.1 `concurrency` (D-5) en tête de `ci.yml`.
- [x] E1.2 Lots Go : fichier de définition unique (motifs par lot ; dernier lot = complément, D-3),
  script bash qui rend la liste de paquets d'un lot, équilibré sur les durées du JSONL de référence :
  aucun lot au-delà de 600 s de temps de test. `grammar` coupé en deux par une liste nommée de tests
  lourds (D-2). Chaque lot : `-json -coverprofile -covermode=atomic -coverpkg=./... -tags=integration
  -count=1 -p 1`, même env que l'actuel, artefacts JSONL + profil.
- [x] E1.3 Job de fusion (`needs` : tous les lots, `if: always()`) : contrôle d'ensemble des paquets
  (D-3), `check_test_baseline.sh tests --from-jsonl` sur le JSONL concaténé, fusion des profils (D-4),
  `coverage_filter.sh`, `coverage_check.sh`, rapport HTML, affichage `internal/sync` filtré. Le step
  « go test couverture internal/sync » est supprimé.
- [x] E1.4 `-timeout` : 600 s par paquet partout si chaque processus `grammar` passe sous 600 s
  instrumenté ; commentaire du critère mis à jour (le mémo existe, le découpage le complète).
- [x] E1.5 Vitest en 3 lots (`--shard=i/3 --reporter=blob`), fusion `--merge-reports --coverage`
  (artefact HTML conservé) ; typecheck, lint, knip, build, rasterisation restent dans le job
  `frontend`.
- [x] E1.6 Filtre de chemins sur les branches de travail : job `changes` (action épinglée par SHA),
  web seul → pas de jobs Go, Go seul → pas de jobs web, `openapi.yaml`/`ci.yml`/`scripts/` → tout ;
  CI complète inconditionnelle sur `main`, `feat/v75`, `pull_request`. Garde-rail archlint : tout job
  conditionné par `changes` porte le contournement `main`/`feat/v75`/PR (test rouge prouvé sur un
  texte synthétique). `TestCIPathsIgnoreSubsetOfDeployPathsIgnore` vert.
- [x] E1.7 `make gate-push` : deux voies, film (`./internal/games/halo_infinite/film/...`) en
  processus parallèle à la voie DuckDB `-p 1` ; garde-rail : test archlint qui échoue si un paquet de
  `film/...` (tests compris) dépend du pilote DuckDB. Durée avant/après au journal.
- [x] E1.8 Commentaires du `ci.yml` et de `deploy.yml` (`attente-ci` : « ~25-30 min ») remis aux
  nouvelles durées ; `docs/COMMANDS.md` (EN + FR) si `gate-push` change de forme.

Gate : `go test ./internal/archlint/` ; `make gate-push` local vert ; push de la branche → run CI
vert, durée totale ≤ 20 min, couverture fusionnée 81,0 % ± 0,1, gate baseline vert, contrôle
d'ensemble des paquets vert ; durées par job au journal.

Statut du gate (pilote, 2026-10-17) : VERT. `go test ./internal/archlint/` vert ; `make gate-push` vert
(1612 s, exécutant) ; run CI 38979844734 sur `46f727553` : succès en 15 min 39 s (49 min avant), couverture
fusionnée 81,0 % (identique), 356 paquets exécutés sur 356, gate baseline vert. Durées sous couverture
mesurées (s, paquet le plus long du lot) : grammar-lourd 245, grammar-reste 248, film 130 (replay),
platform 301 (platform/duckdb), sync 261 (internal/sync), reste 44. Fin des jobs : grammar 5 min, film
8 min, platform et reste 10 min, sync 12,5 min (lot le plus long), fusion 1 min 16 s. Le job `changes`
coûte 1 min 48 s avant tout autre job, dont 83 s de recherche du point de divergence avec `main`.
Corrections du pilote après le run (commit E1 suivant) : délai remis à 600 s partout (critère du
fichier de lots tenu : moitiés de grammar sous 450 s) ; base du filtre de chemins = `feat/v75` (comparer
à `main` comptait tout l'écart d'intégration, les deux piles toujours touchées). Correction du pilote
avant le run (`46f727553`) : groupe `concurrency` propre au run sur `main` et `feat/v75` (GitHub ne
garde qu'un run EN ATTENTE par groupe, même sans annulation : un push rapproché aurait annulé le run
dont le déploiement attend le verdict), garde-rail archlint étendu (cas « groupe partagé sur main »).

### E2 — Connexion Xbox et vérification de participation (entrée 3) — S

- [ ] E2.1 Test Vitest rouge (`XboxLoginPage.test.tsx`) : statut `authorized` puis sondage en 404
  `attempt_not_found` → aucun nouveau `start` ; puis garde dans l'effet de récupération
  (`XboxLoginPage.tsx:236`) et arrêt du sondage une fois autorisé. Test de non-régression : la relance
  automatique avant autorisation fonctionne toujours.
- [ ] E2.2 Cache borné des participations positives (D-11) dans
  `wire/registry_participation.go:17-32` ; tests : 2 appels = 1 requête, `false` et erreur non cachés,
  borne de taille respectée, 503 conservé sur échec non caché. Logs `slog` au niveau Debug pour
  l'éviction, rien sur le chemin chaud.

Gate : `npx vitest run src/features/setup` (fichier concerné) ; `go test ./internal/api/...`
(contrat golden compris) ; `make check-types`.

### E3 — Séquence d'arrêt du serveur (entrée 4) — S

- [ ] E3.1 `stop_grace_period: 45s` sur `levelup` et `levelup-demo` (`docker-compose.yml`).
- [ ] E3.2 Budget global de 40 s (D-12) : contexte d'arrêt unique dont chaque phase prend sa part ;
  `workerWG.Wait`, `autoBatchQueue.Close`, `watcherDaemon.Stop` bornés ; `duckdb.CloseAll` toujours
  atteint ; log `slog.WarnContext` nommant la phase qui dépasse.
- [ ] E3.3 Commentaire `main.go:1351` réécrit (35 s au pire avant, budget après ; air : `kill_delay`
  5 s, ignoré sous Windows).
- [ ] E3.4 Test : la somme des bornes d'arrêt reste sous la grâce du compose (lecture du compose par
  le test, échec si la clé disparaît).

Gate : `go test ./cmd/server/ ./internal/archlint/` ; `go vet ./cmd/server/`.

### E4 — LUSR : retrait de la v1 (entrée 5, D-6) — M

Pièces : `sync/match_recomputer.go:7-8,26,146`, `sync/lusr_full_recompute.go:26`,
`sync/skill/skill_v2_canonical.go:41-75`, `sync/engine_postsync_scoring.go:182-202`,
`sync/skill/skill_rating*.go`, `migrations/steps_player_match_skill_rank.go:190-192`.

- [ ] E4.1 Test d'intégration ROUGE : après exclusion d'un match, `match_skill_rank_latest` sert la
  valeur v2, et le match exclu ne compte pas dans la note v2.
- [ ] E4.2 La v2 lit `LoadExcludedMatchIDs` (ou son équivalent partagé) et saute les matchs exclus.
- [ ] E4.3 `MatchRecomputer` appelle `RecomputeLUSRCanonicalForPlayer` sous l'accès en rafales ;
  test du contrat d'écrivain réécrit (D-6).
- [ ] E4.4 Les 8 composantes sont écrites par le chemin v2 (calcul de composantes extrait de la v1
  avant suppression, écriture par `BatchBuilder.AddLUSRComponents` → persister existant, INSERT-only) ;
  test : après un post-sync v2, `lusr_component_history_latest` contient le match.
- [ ] E4.5 Suppression de la v1 : `runLUSRV1UnderRead` et l'étape 2 de `runSkillRatingSteps`, alias
  `BatchComputeLUSR`/`batchComputeLUSR`, `BatchComputeLUSRWithMedals`, `upsertLUSRRatings(Batch)`,
  `writeLUSRComponentHistory` (si remplacé), `loadExistingLUSRStates`, `computeSkillRatingsBatch` et
  le preview v1 (`RunBackfillLUSRDryRun`, `cmd_backfill.go:464`) s'ils n'ont plus d'usage, l'opt-out
  `LUSR` d'`IsLUSRV2Canonical` et de `DefaultLUSRModeIfUnset`, les tests v1 (lignes de baseline
  retirées au même commit). Grep final : aucun appelant restant.
- [ ] E4.6 Mesure avant/après sur une COPIE d'une player DB (serveur arrêté ou copie à froid) : date
  de la dernière ligne de `lusr_component_history` avant, présence du match recalculé après.
- [ ] E4.7 CLAUDE.md, skill `db-schema` et ADR 0024 (amendement court) : v1 supprimée, opt-out retiré.

Gate : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` ;
`go test ./internal/archlint/ ./internal/service/...` ; `bash scripts/check_test_baseline.sh tests`
limité aux paquets touchés (ou le gate E10 si la baseline bouge).

### E5 — Dettes Go du 2026-10-15 (entrée 10) — M

- [ ] E5.1 Code mort supprimé avec tests, dispenses et lignes de baseline :
  `FindMatchesMissingParticipantBits`, `RecomputeAfterARTRebuild` (s'il reste après E4),
  `loadAwardsForScoreboard` + `WithAwardsRepo` + champ `awardsRepo`, `CachedHighlightEventsRepo`
  (+ commentaire de `player_matches_cache.go:119`), paramètre `titleSlug` de `SquadUsualContexts`
  (interface, handler, mocks).
- [ ] E5.2 Constantes de calibrage réexportées : `cmd/diag_lusr_player` importe `skill`
  directement, les 14 réexportations sont retirées (si l'outil ne compile plus après E4 : supprimé).
- [ ] E5.3 Erreurs avalées : `resolveWeaponLabelsOnly`, `ApplyResolutionViews` (erreur remontée),
  `lookupMapImageURL`, `ResolveAssetNamesBulk`, `lookupWeaponMeta`, `loadKnownModesEN`,
  `GetAssistsCoef` dans `computeExpectedAssists` : erreur remontée ou journalisée
  (`observability.LevelUnlessCanceled`) AVANT la dégradation. `GetMatchKillSources`/`Assists` :
  `[~]` déjà corrigés (constat E0).
- [ ] E5.4 DDL de `v_match_full` : une source exportée, trois sites l'importent
  (`steps_shared.go:243`, `sync/schema.go:468`, `ops/seed_demo.go:928`) + garde-rail grep (règle 6).
- [ ] E5.5 Annulation de requête dans la vue match : `match_view_repo.go:114-125` ne journalise,
  ne compte ni ne bascule en direct quand `ctx.Err() != nil` ; `match_view_service.go:373` et
  `match_view_killfeed_medals.go:48` en `LevelUnlessCanceled`. Test : contexte annulé → compteur
  inchangé.
- [ ] E5.6 Garde de Campagne étendu aux lecteurs de `match_registry` sur fenêtre de temps (sans
  `xuid =`) ; mutation prouvée sur `loadLUSRMatchIDsInWindow` ; doublon d'exclusion de
  `selectRecentMatchIDs` retiré s'il n'est plus nécessaire au garde.
- [ ] E5.7 `cmd_reset_bitmasks.go:202` → `matchflags.BackfillParticipants` ; mapper OpenSpartan :
  `EventsLoaded`/`MedalsLoaded` posés selon la présence réelle des données ; `RunForDB` des 4 CLI :
  titre passé explicitement (flag `--title`, défaut inchangé).
- [ ] E5.8 `match_bomb_stats` ajouté au snapshot (`ops/snapshot_export.go`) ; compatibilité des
  snapshots existants vérifiée (un ancien snapshot reste lisible, ou refus documenté et assumé).
- [ ] E5.9 Données : les 157 matchs Infinite du 25-30 mai sans bit skill → `[!]` en E5 avec la
  commande de rattrapage écrite au runbook de déploiement (geste prod) ; « profil sans player DB =
  panne des vues » → test qui prouve une dégradation propre, ou correctif S.

Gate : `go test -tags=integration -p 1` des paquets touchés ; `go test ./internal/archlint/
./internal/api/...` ; `make go-api-lint` (ratchet).

### E6 — Citations et profil (entrée 11) — S

- [ ] E6.1 `computeFKFD` réécrit : événements `kill`/`death` via `HighlightEventsRepo.Load` (repli
  `killer_victim_pairs` → Halo 5 couvert), `analysis.FirstBlood` + nouvel `analysis.FirstDeath`
  (pur, testé), instants négatifs exclus ; erreur journalisée, plus avalée. Tests : Infinite et
  Halo 5 avec fixtures.
- [ ] E6.2 `citations.engine` → `supported` pour Infinite (TOML + repli `adapter_data.go:180`,
  test de parité), en-tête du TOML corrigé.
- [ ] E6.3 Diagnostic « maîtrise Carabine à impulsion absente de la page Citations » : présence et
  `enabled` dans `citation_mappings` (local, sur copie ; prod en lecture, user prévenu), image
  servie, résolution du nom d'arme `weapon_kills:Pulse Carbine`, filtre de la page. Cause nommée
  `fichier:ligne`, correctif S avec test ; même contrôle sur toutes les maîtrises d'armes Infinite
  (liste de celles qui n'apparaissent pas). Storm (D-13) : entrée retirée du backlog une fois la
  page juste.

Gate : `go test ./internal/progression/... ./internal/analysis/... ./internal/games/...` ;
vitest du dossier `features/citations` si touché ; contrôle visuel de la page Citations par le
user (témoin : la maîtrise Carabine à impulsion visible avec son image).

### E7 — Tactique : requêtes du chargement (entrée 2, D-10) — M

- [ ] E7.1 Mesure avant (profil JGtm, données locales) : nombre de requêtes et temps jusqu'à
  l'affichage, au journal.
- [ ] E7.2 Calage du fond ajouté à `TacticalMapCard` (domaine, service, OpenAPI,
  `make generate-types`), absent pour une carte sans fond ; test Go.
- [ ] E7.3 `TacticalMapTile` : plus de `GET .../background` par tuile (calage lu de la carte), image
  en chargement paresseux (D-10) ; plan principal inchangé (mêmes clés de cache). Tests Vitest : une
  grille de N cartes ne déclenche aucune requête de calage, aucune image hors écran.
- [ ] E7.4 Mesure après, au journal (critère : ≤ 10 requêtes au chargement hors images visibles).

Gate : `go test ./internal/api/... ./internal/service/...` ; `make check-types` ; vitest
`features/tactical` ; `npx playwright test` des specs tactiques s'il en existe (PR vers main
seulement en CI) ; contrôle visuel du user.

### E8 — Lot film : position à la mort, score KOTH Lattice, Harvest, vies sans identité (entrées 7, 8, 9, 6) — M+

- [ ] E8.1 Position unique (D-7) : `ContextesDesMorts` lit la position de `positionOf` ; test
  d'égalité à 1e-6 ; tolérance 0,5 m du test `research` retirée ; compte des contextes perdus sur le
  corpus versionné et les films locaux (0 attendu, sinon STOP).
- [ ] E8.2 KOTH Lattice : émissions brutes de `SeriesTotal` autour de l'image 2268 (`26602661`) et
  5346 (`7de0b91d`) ; cause nommée (point jeté, émission hors domaine, camps, manches) ; correctif à
  la source ; critère : `7de0b91d` rend 40 sans contradiction comptée, `26602661` sans descente, les
  autres KOTH locaux inchangés (diff des séries).
- [ ] E8.3 Harvest : `map_id` tiré du registre pour `e449a696` ; nature de la carte (Forge ou native) ;
  recette recos-a (preuve level_id si Forge, `mapModule`, `mapquant-build`, `mapobj-build`,
  relevés de catalogue, test `replaybuild`). Exige l'installation du jeu sur le poste. Si le mvar est
  introuvable : `[!]` justifié et retour au user.
- [ ] E8.4 Vies sans identité (D-9, exécutant Opus annoncé) : outil de mesure par film dans un
  processus séparé (test `research` ou CLI existante étendue, pas de jetable hors dépôt), lancé sur le
  parc APRÈS accord du user (durée estimée annoncée) ; ventilation par build et par cause ; pièces de
  `3f048bf3` présentées au user pour trancher ; correctifs par règle générale ; arrêt au 2e lot sans
  baisse.
- [ ] E8.5 Révisions : une montée par couche touchée (D-8), empreintes et tests de révision à jour ;
  `replay-corpus-gate` vert.
- [ ] E8.6 Recuisson ciblée locale (accord du user) : KOTH, témoin Harvest, films d'E8.4 ; mesures
  au journal ; séquence prod écrite au runbook de déploiement.

Gate : `go test ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/...
./internal/replaybuild/...` ; `make go-api-test-gamefiles` si `internal/himap` touché ; gate du
corpus de rejeu ; contrôle visuel du rejeu EN LECTURE par le user (témoins : `7de0b91d`,
`e449a696`).

### E9 — Halo 5 au déploiement : recalcul des dérivés et démo (entrée 12, D-14) — S à M

- [ ] E9.1 `--perf`, `--assists-model`, `--lusr`, `--citations` honorent `--title` (CLI) ; champ
  `title` de `POST /backfill/start` (OpenAPI + types) ; tests par titre.
- [ ] E9.2 Chemin des sessions (`POST /settings/sessions/recalculate`) vérifié pour Halo 5
  (`PlayerDBPath` avec titre vide) ; corrigé si faux, avec test.
- [ ] E9.3 « Dates de paliers » : source identifiée ou `[!]` justifié (symbole introuvable au
  constat E0).
- [ ] E9.4 Séquence écrite dans le runbook de déploiement (EN) : recalculs H5 par routes admin,
  puis re-seed de la démo prod (`deploy.yml`, job `deploy-demo`). Exécution = geste du user.

Gate : `go test ./cmd/levelup/ ./internal/api/... ./internal/service/...` ; `make check-types`.

### E10 — Clôture

- [ ] E10.1 Une revue adversariale (skill `adversarial-review`, Sonnet, contexte frais) sur le diff
  des étapes à risque (E1, E4, E8) ; constats corrigés ou statués.
- [ ] E10.2 Fusion de `origin/feat/v75` dans la branche si elle a bougé ; gates locaux rejoués.
- [ ] E10.3 Push → CI complète verte (durées au journal).
- [ ] E10.4 `.ai/BACKLOG.md` : contrôle final (les 12 entrées traitées au fil des étapes, aucune
  entrée orpheline) ; `.ai/REGISTRE_REPORTS.md` à jour (lignes Lattice, position à la mort) ; thought_log ;
  `disk-hygiene.ps1 -Lot backlog1016 -Apply`.
- [ ] E10.5 Remise au user : fusion dans `feat/v75` (son geste), séquence de déploiement (E8.6,
  E9.4, E5.9, grandfather du pool déjà prévu).

## 3. Découvertes

- E0 (superviseur) : l'indicateur « Analyse en cours » (maquette validée le 2026-10-16) n'est inscrit
  ni au backlog ni à ce plan. Non traité ici ; à confirmer par le user avant la release.
- E1 (exécutant, 2026-10-17) : des tests Go lisent `apps/web/src` (parité des capacités
  `domain/title/capabilities_parity_test.go` et `capabilities_ts_mirror_test.go`,
  `games/capabilities_front_parity_test.go`, golden TS de `cmd/weapon-sounds`,
  `film/.../objectives/families_parite_ts_test.go`, fixtures de contrat de `film/replay`,
  `archlint/no_hardcoded_base_weapons_test.go`). La règle « web seul -> jobs Go sautés » (E1.6) ne les
  rejoue donc pas sur une branche de travail. Atténué : les chemins `apps/web/src/lib/capabilities/**` et
  `apps/web/src/features/match-replay/**` comptent pour les deux piles. Trou résiduel connu : le balayage
  des littéraux de capacités (`frontCapabilityLiterals`) lit TOUT `apps/web/src`. Rattrapé par la CI
  inconditionnelle sur `feat/v75`, `main` et les PR ; non traité.
- E1 : `scripts/gate-push.ps1` (maillon 2/4) joue encore UNE suite `go test ./...` en série, sans les
  deux voies de `check_test_baseline.sh`. Hors périmètre E1.7 (le plan nomme `make gate-push`) ; docs
  COMMANDS (EN + FR) le disent. Non traité.
- E1 : sur `main`, `concurrency` sans annulation met un run en attente et REMPLACE le run en attente
  précédent (comportement GitHub) : trois pushs rapprochés sur `main` annulent le run du deuxième, dont le
  déploiement (`attente-ci`) se refuserait alors sur une CI « cancelled ». Rare (squash merges espacés de
  plus d'une CI) ; non traité.
- E1 : `go list -tags=integration ./...` compte 356 paquets, le run de référence `38938482739` en avait
  352 : l'arbre a gagné 4 paquets depuis. Le complément (`reste`) les absorbe, sans intervention.

## 4. Journal

- 2026-10-16 : plan écrit après vérification sur pièces (section 0). Décisions user D-1, D-6, D-7 et
  question Storm (→ E6.3). Durées Go par paquet sous couverture (run `38938482739`, s) : grammar ≥ 900
  (coupé par le délai), platform/duckdb 348, sync 266, film/replay 253, persist 101, ops 101,
  api/wire 85, sync/killcollector 63, mapdecoupe 56, archlint 53, killsource 50, cmd/levelup 49,
  halo_infinite/migrations 49, sync/replayartifacts 45, scheduler 36, le reste ≤ 33. Somme ~2 800 s
  pour 346 paquets ; compilation sérialisée ~390 s.
- 2026-10-17 E1 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E1.1 à E1.8 faits, gates locaux
  verts, CI non encore jouée (geste du pilote).
  - Fichiers : `.github/workflows/ci.yml` (concurrency, job `changes`, matrice `go-coverage` + fusion
    `go-coverage-merge`, matrice `web-tests` + fusion `web-coverage-merge`), `.github/ci/go-shards.txt`
    (définition unique des lots), `scripts/ci_go_shard.sh`, `scripts/check_test_baseline.sh` (deux voies),
    garde-rails `archlint/ci_go_shards_test.go`, `ci_paths_filter_test.go`, `film_lane_duckdb_free_test.go`,
    commentaires `deploy.yml` / `Makefile` / runbook, `docs/COMMANDS.md` (EN + FR), `CLAUDE.md`.
  - Lots Go (6, partition exacte de `go list -tags=integration ./...` : 356 paquets ; seul `grammar` est
    dans deux lots) et estimation en secondes de TEST sous couverture (run `38938482739`) :
    `film-grammar-lourd` 1 paquet, `-run` de 6 tests, ~573 s ; `film-grammar-reste` 1 paquet, `-skip` des
    mêmes 6, ~595 s ; `film` 23 paquets (film/... hors grammar : replay 253, killsource 50, ... ; plus
    persist 101 et ops 101) 519 s ; `platform` 8 paquets (platform/duckdb 348, prestige 32, sharedprovider 9,
    halo5, migration 14, halo_infinite/migrations 49, mapdecoupe 56, cmd/levelup 49) 557 s ; `sync` 19
    paquets (sync 266, api/wire 85, killcollector 63, archlint 53, replayartifacts 45, v2 17, skill 12, ...)
    553 s ; `reste` = complément, 305 paquets, 270 s de test mais ~335 s d'édition de liens (1,1 s/paquet,
    estimation) soit ~600 s. Liste lourde : TestSuivanteEgaleLaBoucleDAvantSurTamponsSynthetiques 233 s,
    TestLaStructureDesBobinesTientSesInvariants ~139 s (extrapolé), TestFrameClosureDetailleeRendLaCarte
    DeFrameClosure 78 s, TestGoldenMiniBobineFamilles 44 s, TestLaFermetureDeLaStructureNeDescendPas 40 s,
    TestChaqueLectureCiteUneEtendueDeLaStructure 39 s. Sortie de `ci_go_shard.sh` par lot : voir
    `bash scripts/ci_go_shard.sh --verifier` (6 lots, 356 paquets, complément de 305).
  - Estimation de `grammar` : le run de référence a coupé le paquet à 900 s avec 583 tests terminés
    (869 s). Rapport CI/local sur ces 583 tests : 16,7 (52,1 s locales). Le paquet entier compte 926 tests
    de premier niveau, 70,0 s locales : extrapolation ~1 170 s (le plan disait ~950). Chaque moitié pèse donc
    ~575-595 s, sans marge sous un délai de 600 s : E1.4 -> `-timeout 900s` pour les DEUX lots grammar
    seulement, 600 s ailleurs (plus long processus hors grammar : platform/duckdb, 348 s). Retour à 600 s
    quand une moitié mesurée passe sous 450 s (commentaire de `go-shards.txt`). Les mesures locales de
    Go sont bruitées (machine à 100 % de CPU par d'autres sessions) : seuls les rapports servent.
    Aucun des 6 tests lourds ne partage de mémo de bobine avec le reste du paquet (les mémos sont dans
    `bobines_memo_test.go`, utilisés par les tests de clôture d'image-clé et d'équipes) : le découpage ne
    duplique pas de travail. Union des deux moitiés = 926 tests, sans doublon (vérifié en local).
  - Preuves du script : 9 définitions défectueuses refusées avec le bon message (chevauchement, motif
    périmé, complément non dernier, filtre sans paire, paire de listes différentes, exclusion inutile,
    liste non déclarée, timeout d'un lot inconnu, complément vide) et la paire `run:/skip:` valide acceptée ;
    le Go le rejoue (`TestCIGoShardScriptRefusesBrokenDefinitions`). Contrôle d'ensemble de la fusion
    rejoué sur des JSONL synthétiques issus des vrais lots : nominal vert (356/356) ; un paquet retiré ->
    échec nommé (`internal/watcher`) ; paquet fantôme + lot sans artefact -> échecs nommés.
  - Fusion des profils (D-4), preuve locale (CGO=0, `-covermode=atomic`, `-coverpkg=./internal/domain/title,
    ./internal/games/...`) : A = `domain/title` 87,2 %, B = `games` 25,3 %, exécution unique A+B 35,1 %,
    profils A et B concaténés (une ligne `mode:`) 35,1 % ; A+A concaténé 87,2 % ; blocs couverts : A 234,
    B 500, union 709 = exécution unique 709. `go tool cover` additionne donc bien les blocs dupliqués.
  - Vitest (E1.5) : `--shard=1/3`, `2/3`, `3/3` puis `--merge-reports --coverage` en local : 942 fichiers,
    9 797 tests passés, couverture 32 064/38 726 = 82,79 % statements, identique à un run complet non
    découpé (2 min 36 s ; un lot ~1 min, la fusion 7 s). Le blob contient la couverture brute : la
    fusion produit le rapport HTML. Dossier `.vitest-reports` caché : `include-hidden-files: true` sur
    l'upload (sinon `upload-artifact@v4` ne l'envoie pas), `.gitignore` complété.
  - Filtre de chemins (E1.6) : `dorny/paths-filter` v4.0.3, SHA `ceb8a2b8f2d89434be7ff52d3de7ec3738c5cc9d`
    (tag non annoté, désigne un commit). Décision : sorties BOOLÉENNES par filtre (pas de `list-files`) —
    la liste complète d'une branche éloignée de `main` dépasserait la taille d'une variable d'environnement
    (128 Ko) et ferait échouer `changes`, donc toute la CI. Classement `some-with-excludes` : web, go (hors
    openapi), croisee (openapi + miroirs TS lus par des tests Go), autre (tout le reste), tous. Simulé avec
    le vrai `picomatch` (`dot: true`) sur les filtres lus dans `ci.yml` : web seul -> go=false ; go seul ->
    web=false ; openapi, `.github`, `scripts`, `Makefile`, docs, config, `.ai/baselines`, miroirs -> les deux ;
    aucun fichier -> les deux. Base `main` (défaut de l'action), PAS le push précédent : un run annulé par
    `concurrency` ne doit pas laisser un changement Go non testé derrière un push web. Mutation prouvée :
    retirer `feat/v75` de l'`if` de `go-lint` fait échouer `TestCIJobsDependingOnChangesCarryBypass`.
    `govulncheck` et `npm-audit` restent sans condition (`ci_workflows_security_test.go`). Les jobs de
    fusion se sautent quand leur matrice est sautée (`needs.<matrice>.result != 'skipped'`) et jugent
    sinon avec `!cancelled()`.
  - Outillage : `actionlint` v1.7.7 (avec shellcheck) : 0 remarque sur `ci.yml` (2 remarques SC2012/SC2010
    sur mes scripts corrigées). Parse YAML par `js-yaml` : OK.
  - Gates locaux : `go test ./internal/archlint/` VERT (79 s) ; `golangci-lint` (ratchet) sur archlint : 0
    issue ; `make gate-push` VERT, `rc=0`, 1 612 s au total (lint + web ~2,5 min, suite 1 463 s) ; voies :
    film 22 paquets en 124 s, DuckDB 334 paquets en 1 463 s ; 21 094 tests courants contre 9 398 en
    baseline, aucun manquant, aucun échec. Durée « avant » NON mesurée (un second run de 25 min sur une
    machine chargée par d'autres sessions n'aurait pas été comparable) : le gain attendu est la durée de la
    voie film (~124 s sur 1 587 s en série), ce que le gate ne permet pas de distinguer du bruit. Le gain de
    CI vient des lots, pas du gate local.
  - Incertitudes que seul un vrai run tranchera : (1) durées réelles des lots et de la compilation
    (estimation 1,1 s de lien par paquet ; la mesure sort dans le résumé de `go-coverage-merge`) ; (2) le
    `-timeout 900s` des moitiés de grammar tient-il (estimation extrapolée, 573/595 s) ; (3) couverture
    fusionnée attendue 81,0 % ± 0,1 ; (4) comportement de `dorny/paths-filter` sur un push de branche (fetch
    de `main`, profondeur) ; (5) `upload-artifact` / `download-artifact` `merge-multiple` sur le dossier
    caché et les noms de profil ; (6) l'ordre de compilation parallèle de 6 machines contre le cache
    setup-go (premier run à froid).
