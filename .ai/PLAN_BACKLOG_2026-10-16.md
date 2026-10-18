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
- **D-15** (user, 2026-10-17) Lot film autorisé en entier : décodages de recherche film par film,
  mesure des vies sans nom sur le parc (2 676 films, un processus par film, en série), recuisson
  ciblée en fin d'E8 (serveur local arrêté pendant la recuisson). Toujours un seul décodage à la
  fois, jamais en arrière-plan pendant un autre, jamais de boucle de shell autour d'un CLI qui
  décode (protections `internal/filmproc`).
- **D-16** (user, 2026-10-17, après l'enquête E8.1) Les trois erreurs de lecture des positions du
  coup fatal sont corrigées dans E8 : rattachement par xuid (joueurs renommés), dernière position
  tenue tant que le film n'émet pas de changement (même vie, hors embarquement), position du siège
  pour l'occupant d'un véhicule. La position à la mort (contexte) suit la même lecture. Remplace la
  branche « STOP si perte » de D-7 : l'unification ne doit plus rien perdre.
- **D-17** (user, 2026-10-17) Frag posthume (tueur déjà mort, projectile en vol) : le tueur est
  placé au lieu de sa mort, repéré comme position posthume (repli nommé et compté).
- **D-18** (user, 2026-10-18) `3f048bf3` (Vagabond, 26/02, pas un match de JGtm) : l'API confirme
  deux bots (équipe 1, 18 s ; équipe 0, 9 s ; décalage constant ~44 s entre ses horaires et le chrono
  du film). (a) Le joueur de l'équipe 1 entre 0:52 et 1:01 = 343 Ritzy : lecture actuelle juste.
  (b) La silhouette de 1:44,4-1:44,9 = l'arrivée de 343 Lacuna, en ÉQUIPE 0. Le user : « ce n'est pas
  possible qu'un bot ou un joueur ait deux équipes » → la couleur équipe 1 lue à cet instant est
  présumée être une ERREUR DE LECTURE (équipe héritée de l'occupant précédent de la place) : la
  chercher et la corriger par une règle générale ; à défaut de lecture prouvée, la vie prend l'équipe
  du reste de sa vie (0). Même traitement pour `8c19b7dc` (Isolation, 09/06).
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

- [x] E2.1 Test Vitest rouge (`XboxLoginPage.test.tsx`) : statut `authorized` puis sondage en 404
  `attempt_not_found` → aucun nouveau `start` ; puis garde dans l'effet de récupération
  (`XboxLoginPage.tsx:236`) et arrêt du sondage une fois autorisé. Test de non-régression : la relance
  automatique avant autorisation fonctionne toujours.
- [x] E2.2 Cache borné des participations positives (D-11) dans
  `wire/registry_participation.go:17-32` ; tests : 2 appels = 1 requête, `false` et erreur non cachés,
  borne de taille respectée, 503 conservé sur échec non caché. Logs `slog` au niveau Debug pour
  l'éviction, rien sur le chemin chaud.

Gate : `npx vitest run src/features/setup` (fichier concerné) ; `go test ./internal/api/...`
(contrat golden compris) ; `make check-types`.

Statut du gate (exécutant, 2026-10-17) : VERT. `npx vitest run src/features/auth src/features/setup` : 8 fichiers,
129 tests passés (dont le nouveau test, rouge avant le correctif : `pollCount` 5 au lieu de 1) ;
`go test ./internal/api/...` : api 42 s, handlers 24 s, humacore, middleware, wire tous `ok` ; `make check-types`
(tsc -b) sans erreur ; eslint sur les 3 fichiers web sans remarque ; `golangci-lint run --new-from-merge-base=origin/main`
sur `internal/api/wire` : 0 issue.

### E3 — Séquence d'arrêt du serveur (entrée 4) — S

- [x] E3.1 `stop_grace_period: 45s` sur `levelup` et `levelup-demo` (`docker-compose.yml`).
- [x] E3.2 Budget global de 40 s (D-12) : contexte d'arrêt unique dont chaque phase prend sa part ;
  `workerWG.Wait`, `autoBatchQueue.Close`, `watcherDaemon.Stop` bornés ; `duckdb.CloseAll` toujours
  atteint ; log `slog.WarnContext` nommant la phase qui dépasse.
- [x] E3.3 Commentaire `main.go:1351` réécrit (35 s au pire avant, budget après ; air : `kill_delay`
  5 s, ignoré sous Windows).
- [x] E3.4 Test : la somme des bornes d'arrêt reste sous la grâce du compose (lecture du compose par
  le test, échec si la clé disparaît).

Gate : `go test ./cmd/server/ ./internal/archlint/` ; `go vet ./cmd/server/`.

Statut du gate (exécutant, 2026-10-17) : VERT. `go vet ./cmd/server/` sans sortie ; `go test ./cmd/server/
./internal/archlint/ -count=1` : `ok` (cmd/server 8 s, archlint 68 s, `compose_hardening` compris avec la nouvelle
clé) ; `go test ./internal/platform/coalesce/` `ok` ; `golangci-lint run --new-from-merge-base=origin/main` sur
`cmd/server` et `platform/coalesce` : 0 issue.

### E4 — LUSR : retrait de la v1 (entrée 5, D-6) — M

Pièces : `sync/match_recomputer.go:7-8,26,146`, `sync/lusr_full_recompute.go:26`,
`sync/skill/skill_v2_canonical.go:41-75`, `sync/engine_postsync_scoring.go:182-202`,
`sync/skill/skill_rating*.go`, `migrations/steps_player_match_skill_rank.go:190-192`.

- [x] E4.1 Test d'intégration ROUGE : après exclusion d'un match, `match_skill_rank_latest` sert la
  valeur v2, et le match exclu ne compte pas dans la note v2.
  `TestMatchRecomputer_ExclusionServeLaNoteV2SansLeMatchExclu` (tag `integration`), rouge avant tout
  correctif : 11 matchs sur 12 servis à 1501-1505 (valeur v1) au lieu de 2200 (valeur v2) dans
  `match_skill_rank_latest_by_type`, expérience v2 12 au lieu de 11, mu 30,174 au lieu de 30,087 (référence
  rejouée sans le match exclu). Vert après E4.2 + E4.3. Prémisse corrigée : `match_skill_rank_latest` (vue lue
  par l'interface) ordonne `start_time DESC NULLS LAST` AVANT `written_at` et les lignes v1 n'ont pas de
  `start_time` : la v1 ne la masquait pas ; le masquage était dans `match_skill_rank_latest_by_type` (graphe
  d'évolution, arbitré sur `written_at`). Le test affirme sur les deux vues.
- [x] E4.2 La v2 lit `LoadExcludedMatchIDs` (chargeur existant, sans copie) et saute les matchs exclus :
  retirés dans la sélection sous lecteur (`selectShadowWorkUnderRead`, avant `scorable`), compteur
  `skipped_excluded` dans les deux lignes INFO. Test `TestRunLUSRV2Shadow_Canonical_SkipsExcludedMatches`,
  rouge avant (processed 3, 2 lignes de note pour le match exclu, expérience 3), vert après. Deux tests
  d'échec d'écriture (`WriteFailHoldsWatermark`, `HeldGroupSkipsLaterMatches`) simulaient l'échec par une base
  joueur FERMÉE, que la lecture des exclusions refuse désormais en amont : ils utilisent une base joueur qui
  refuse toute écriture de note (`CHECK (FALSE)`), même intention.
- [x] E4.3 `MatchRecomputer` -> `RecomputeLUSRCanonicalWithAccess` (nouvelle, accès en rafales ; la version à
  handle tenu `RecomputeLUSRCanonicalForPlayer` l'enveloppe en mode épinglé). Contrat d'écrivain retenu : une
  rafale pour la sentinelle de reset (un INSERT), puis les rafales du rejeu v2 (au plus 50 matchs ou 2 s, sélection
  et filigrane sous lecteur) ; jamais en continu ; la note de performance se lit sous un segment de lecture relâché
  avant la première rafale. `TestMatchRecomputer_NeverAcquiresTheSharedWriter` ->
  `TestMatchRecomputer_TakesTheSharedWriterInShortBurstsOnly` (2 prises d'écrivain pour 12 matchs, aucune lecture
  pendant que l'écrivain est tenu, base revenue en lecture seule) ; `..._NoProviderReadsSharedReadOnly` ->
  `..._NoProviderReleasesTheSharedWriterLease`. Lignes de baseline : ces deux tests n'y figuraient pas (0 ligne à
  retirer). Garde archlint `TestNoSharedWriterOnRequestPath` : entrée datée pour `match_recomputer.go` (décision
  D-6) ; `syncRootFileBaseline` 79 -> 78.
- [x] E4.4 Les 8 composantes sont écrites par le chemin v2. Vérification sur pièces : leur calcul
  (`computeCompositeScoreWithBreakdown`) dépend des statistiques du match et de moyennes glissantes du joueur,
  pas de mu/sigma v1 : pas d'arrêt. Nouveau `skill_v2_components.go` (orchestration par match, fenêtres
  glissantes par groupe vides à chaque cycle comme la v1 incrémentale, score d'exploit médailles chargé une fois
  sous lecteur par `skill.WithMedalExploit`), écriture INSERT-only par `AppendOnlyLUSRPersister.PersistComponents`
  (même corps d'INSERT `persistLUSRComponents` que le flux d'ingestion), lecture par
  `lusr_component_history_latest`. Écart au libellé : `BatchBuilder.AddLUSRComponents` -> `PlayerPersister` n'est
  pas utilisable ici (le persister ne fait rien sans ligne d'enrichissement « live » et saute un match déjà
  ingéré) ; `AddLUSRComponents` garde donc zéro appelant de production (candidat E5.1, non traité). Best-effort
  avec compteur `levelup.lusr_v2.component_write_errors_total` et Warn. Test
  `TestPostSyncLUSRV2_WritesTheEightComponents` (par `runSkillRatingSteps`), rouge avant
  (`LUSRUpdated = 0`, aucune composante). Parité prouvée avant la suppression de la v1, test jetable : 40 matchs,
  v1 force contre v2, 275 valeurs `match_id/composante` identiques à 1e-12.
- [x] E4.5 Suppression de la v1 : `BatchComputeLUSR`/`batchComputeLUSR`/`BatchComputeLUSRWithMedals`,
  `computeSkillRatingsBatch`, `upsertLUSRRatings(Batch)`, `writeLUSRComponentHistory`, `loadExistingLUSRStates`,
  `lusrResult`, `applyInactivityDecay`, le preview (`BatchComputeLUSRPreview`, `RunBackfillLUSRDryRun`,
  `levelup backfill --lusr --dry-run` : pas d'équivalent v2 trivial, l'aperçu comparait des mu/sigma v1),
  `runLUSRV1UnderRead` et l'étape 2 de `runSkillRatingSteps`, `IsLUSRV2Enabled`/`IsLUSRV2Canonical` et leurs
  variables d'environnement `LEVELUP_LUSR_V2_ENABLED` / `LEVELUP_LUSR_CANONICAL` (+ `=LUSR`),
  `DefaultLUSRModeIfUnset`, `LogLUSRModeAtBoot` (et l'appel de `cmd/server`), les `Setenv` des CLI, la sentinelle
  dual-row sans condition, et `RecomputeAfterARTRebuild` (+ test : sans appelant de production, il appelait la v1)
  `[~]` E5.1. Docs : `docs/SYNC_GUIDE.md` EN + FR. Tests v1 retirés (28 tests, dont 7 dans la baseline : 40 lignes
  JSONL retirées au même commit, en-tête de `check_test_baseline.sh`). Conservé : `RunFormulaSim` (simulateur
  de formules `--compare-formulas`, TrueSkill v1, n'écrit rien, hors liste du plan) et ce qu'il tient
  (`trueskillUpdate`, `PlayerState`, constantes de calibrage : E5.2). Grep final au journal.
- [x] E4.6 Mesure avant/après sur une COPIE d'une player DB. `cmd/diag_q` impossible : les bases du serveur local
  sont verrouillées de façon exclusive (lecture refusée, y compris par copie). Copie à froid de JGtm prise dans
  la sauvegarde `data/backups/avant_recuisson_98_2026-10-16` (joueur + base partagée du même instant),
  `cmd/recompute_perfnote` (`-mode sql`, puis `-mode lusr` = `RecomputeLUSRCanonicalForPlayer`). Avant :
  `lusr_component_history_latest` 7 137 lignes / 1 028 matchs, dernière ligne 2026-07-18 20:51, 1 153 matchs
  notés LUSR dont 125 sans composantes. Après le rejeu (1 108 matchs traités en 47 s, 24 rafales de ~2 s, 36 matchs
  non deux équipes, 3 déséquilibrés) : 8 011 lignes / 1 153 matchs, dernière ligne du jour du rejeu, 0 match noté
  sans composantes.
- [x] E4.7 CLAUDE.md (règle 10), skill `db-schema` (`lusr_component_history`, les deux vues de
  `match_skill_rank`), ADR 0024 (amendement EN du 2026-10-17), docs/SYNC_GUIDE EN + FR.

Gate : `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` ; `go test ./internal/archlint/
./internal/service/...` ; `go test ./cmd/levelup/ ./cmd/server/` ; `go vet` des paquets touchés ;
`bash scripts/check_test_baseline.sh tests`.

Statut du gate (exécutant, 2026-10-17) : VERT (sorties au journal).

### E5 — Dettes Go du 2026-10-15 (entrée 10) — M

- [x] E5.1 Code mort supprimé avec tests, dispenses et lignes de baseline :
  `FindMatchesMissingParticipantBits` (+ 3 tests + dispense archlint), `RecomputeAfterARTRebuild` (`[~]` supprimé par E4.5),
  `loadAwardsForScoreboard` + `WithAwardsRepo` + champ `awardsRepo` (+ 4 tests), `CachedHighlightEventsRepo` (fichier et
  test entiers, dont `ttlCacheHE`, `highlightEventsCacheKey`, et `cacheMetrics` de `player_matches_cache.go` qui n'avait plus
  d'autre lecteur : le commentaire de `:119` part avec), paramètre `titleSlug` de `SquadUsualContexts` (interface
  `prestige.Service`, handler, mocks, `LazyPrestigeService`). Le paramètre n'était pas mort pour le service interne, qui le
  passait au fournisseur : le titre est désormais `prestige.Deps.TitleSlug`, posé par `serviceAndPlayerDB` depuis le PlayerDB
  résolu (même principe que `BaselineProvider` : l'identité ne traverse pas la frontière), et le test unitaire du service
  affirme le titre reçu par le fournisseur. Ajouts demandés par le journal d'E4 : `BatchBuilder.AddLUSRComponents` (zéro
  appelant de production) supprimé avec `PlayerBatch.LUSRComponents`, son appel dans `PlayerPersister`, son assainissement
  (`SanitizeBatch`) et son test (`persistLUSRComponents` reste : `AppendOnlyLUSRPersister.PersistComponents` l'utilise) ; type
  mort `shadowParticipant` supprimé. Baseline : 128 lignes JSONL / 18 paires retirées, en-tête de
  `check_test_baseline.sh` mis à jour.
- [x] E5.2 Constantes de calibrage réexportées : l'outil compile après E4 ; `cmd/diag_lusr_player/replay.go` importe
  `internal/sync/skill` pour ses 14 constantes (`InitialMU` ... `IndividualMUAlpha`), les 14 réexportations de
  `sync/skill_reexport.go` sont retirées (grep : aucun autre appelant `sync.X`).
- [x] E5.3 Erreurs avalées, toutes journalisées AVANT la dégradation : `resolveWeaponLabelsOnly` (requête, ligne illisible,
  itération), `resolveWeaponMeta` (c'est lui qui rend `lookupWeaponMeta` muet : ligne illisible, identifiant illisible,
  itération ; la requête unifiée en échec était déjà en WARN), `ApplyResolutionViews` (erreur de `v_match_full` REMONTÉE),
  `lookupMapImageURL`, `ResolveAssetNamesBulk` du mode du match, `resolveAssetName`/`resolveAssetNameEN` (même motif, relevé
  en E4.8b), `loadKnownModesEN` (table absente = DEBUG, jamais plus de silence), `GetAssistsCoef` dans
  `computeExpectedAssists` (ErrNoRows = WARN modèle non seedé, autre = ERROR, requête abandonnée = DEBUG). Niveau par le
  nouveau `degradedLookupLevel` (table absente DEBUG, sinon `observability.LevelUnlessCanceled` à WARN). Tests :
  `lookups_degrades_journalises_test.go` (4 résolutions, panne = WARN, table absente = DEBUG),
  `TestApplyResolutionViews_VMatchFullEnEchecRemonte`, `TestComputeExpectedAssists_CoefPopulationnelEnEchec_EstJournalise`.
  `GetMatchKillSources`/`Assists` : `[~]` déjà corrigés (constat E0).
- [x] E5.4 DDL de `v_match_full` : source unique `migration.VMatchFullViewSQL` ; `sync/schema.go` (`sharedViewsSQL`) et
  `ops/seed_demo.go` l'importent ; garde-rail `archlint/v_match_full_ddl_single_source_test.go` (aucun second
  `CREATE [OR REPLACE] VIEW v_match_full` hors de la source dans le code non-test), mutation prouvée (littéral recopié dans
  `sync/schema.go` : échec nommant le fichier), auto-preuve du motif.
- [x] E5.5 `GetMatchMeta` : sur `ctx.Err() != nil` après la lecture du snapshot, ni bascule en direct, ni ERROR, ni
  compteur ; l'erreur remonte. `match_view_service.go` (métadonnées) et `match_view_killfeed_medals.go` en
  `LevelUnlessCanceled`. Tests : `TestGetMatchMeta_ContexteAnnule_NiBasculeNiCompteurNiError` (rouge prouvé sans la garde :
  forceLive armé, compteur 0 -> 1, ERROR), `TestMatchView_MetadonneesEtMedaillesDuFeed_NiveauSelonLaRequete`.
- [x] E5.6 Garde de Campagne étendu : `estLecteurCampagne` reconnaît le registre (`match_registry`, `v_match_full`) lu SEUL sur
  une fenêtre de temps (inégalité ou BETWEEN contre un paramètre ; `<>` n'en est pas une, relevé par un faux positif de
  `killcollector/postsync.go:conditionBacklog`). Mutation prouvée : retirer `excludeCampaignClause` de
  `loadLUSRMatchIDsInWindow` fait échouer le garde. Deux nouveaux lecteurs statués par dispense datée du 2026-10-17
  (`cmd/diag_film_avail`, `cmd/lusr_v2_squad_estimate` : outils opérateur). `selectRecentMatchIDs` : le doublon est CONSERVÉ,
  il est encore nécessaire au garde (mutation : sans lui le garde échoue, le garde ne voit pas que `dc.excludeCampaign`
  filtre l'ensemble final).
- [x] E5.7 `cmd_reset_bitmasks.go` : `matchflags.BackfillParticipants` (copie 512 retirée). Mapper OpenSpartan : `MedalsLoaded`
  et le bit `BackfillBitMedals` ne sont posés que si des médailles sont mappées ; `EventsLoaded` SUPPRIMÉ (écart au libellé :
  la table d'événements est importée dans une passe à part, `MapMatch` ne peut jamais savoir si des événements sont
  chargés, un champ toujours faux serait un champ mort). `RunForDB` -> `RunForTitleDB` avec `--title` (défaut
  `halo_infinite`) dans `apply_shared_migrations` (+ `halo5migrations.Register()`), `backfill-world-player-stats` (aussi pour
  le chemin du shared) et `diag_bot_resolution`, et `slug` déjà porté par `levelup backfill-killsource`
  (`migrerSchemaPartage`, flag `--title` existant).
- [x] E5.8 `match_bomb_stats` ajouté à `sharedSnapshotTitleOwnedRaw` : exportée quand la base la porte, reconstruite avec
  `match_bomb_stats_latest` (`migration.MatchBombStatsLatestViewSQL`) à la lecture. Table NON requise : un cut ancien (sans le
  Parquet) reste lisible, la vue match en sert les colonnes d'Assaut absentes (WARN existant de `loadMatchBombStats`), aucun
  repli live global. Tests : `TestSnapshot_ExporteLesStatsDAssaut` (dernière génération servie, match non prêt exclu ; rouge
  prouvé sans l'ajout), `TestSnapshot_CutAnterieurAuxStatsDAssaut_resteLisible`.
- [!] E5.9a Les 157 matchs Infinite du 25-30 mai sans bit skill (4) : NON traités ici, geste prod. Commande écrite au runbook
  (`docs/RUNBOOK_DEPLOY_CHECKLIST.md`, section Post-deploy) : serveur arrêté, `levelup reset-bitmasks --dry-run` puis
  `--apply` (pose le bit là où `team_mmr` est présent), nouveau `--dry-run` à 0, puis `levelup backfill --skill` pour ce qui
  reste. Aucune exécution : décision et fenêtre de maintenance du user.
- [x] E5.9b « Profil suivi sans player DB = panne des vues » : CONFIRMÉ par un test (l'ouverture par le pool échoue,
  « chemin introuvable », quand le dossier du joueur n'existe pas), donc correctif S : `config.PlayerDBFileExists` (rend true
  en démo) consulté par l'opener des amis (`registry_pages.go`) ; absence => `service.ErrFriendWithoutPlayerDB`, que le
  résolveur traite en absence (DEBUG, ni erreur ni `section_unavailable`) en lisant les autres amis. Une autre panne
  d'ouverture reste une panne remontée. Tests : `TestFriendsExtrasResolver_AmiSansPlayerDB_EstUneAbsence`,
  `TestPlayerDBFileExists`.

Gate : `go test -tags=integration -p 1` des paquets touchés ; `go test ./internal/archlint/
./internal/api/...` ; `make go-api-lint` (ratchet).

Statut du gate (exécutant, 2026-10-17) : VERT (sorties au journal). `[!]` E5.9a : geste prod du user (runbook).

### E6 — Citations et profil (entrée 11) — S

- [x] E6.1 `computeFKFD` réécrit : événements `kill`/`death` via `HighlightEventsRepo.Load` (repli
  `killer_victim_pairs` → Halo 5 couvert), `analysis.FirstBlood` + nouvel `analysis.FirstDeath`
  (pur, testé), instants négatifs exclus ; erreur journalisée, plus avalée. Tests : Infinite et
  Halo 5 avec fixtures. Fait : `internal/progression/profile/first_events.go` (la fenêtre est celle des matchs du
  JOUEUR, Campagne exclue par `campaignExcl`), `analysis.CountFirstKillsAndDeaths` (règle de la vue match, un
  frag ou une mort sans identité ouvre le match sans créditer personne), erreurs remontées à `BuildProfile`
  qui les journalise (`slog.WarnContext`). Le repli du chargeur lit `match_kill_events_latest` (alimenté par les
  couples `killer_victim_pairs` en prod, `persist/kill_events_credit.go`), pas la table des couples elle-même.
- [x] E6.2 `citations.engine` → `supported` pour Infinite (TOML + repli `adapter_data.go:180`,
  test de parité), en-tête du TOML corrigé. Fait : `config/titles/halo_infinite/mappings/capabilities.toml:50`,
  `games/halo_infinite/adapter_data.go:180`, parité `capabilities_parity_test.go` verte. Seul lecteur
  fonctionnel (`games/feature.go:37`, feature-matrix admin) ; aucun test ne figeait l'ancienne valeur pour Infinite.
- [x] E6.3 Diagnostic « maîtrise Carabine à impulsion absente de la page Citations » : présence et
  `enabled` dans `citation_mappings` (local, sur copie ; prod en lecture, user prévenu), image
  servie, résolution du nom d'arme `weapon_kills:Pulse Carbine`, filtre de la page. Cause nommée
  `fichier:ligne`, correctif S avec test ; même contrôle sur toutes les maîtrises d'armes Infinite
  (liste de celles qui n'apparaissent pas). Storm (D-13) : entrée retirée du backlog une fois la
  page juste. Diagnostic fait, AUCUN défaut de code trouvé (journal) ; Storm retirée du backlog.
- [!] E6.3b État de la metadata de l'instance que le user regardait (poste ou prod) : non lisible ici (bases du
  serveur local verrouillées, prod interdite, l'API locale exige une session). Requête exacte au journal pour le
  pilote ; contrôle visuel du user ouvert (témoin : la maîtrise Carabine à impulsion visible avec son image).

Gate : `go test ./internal/progression/... ./internal/analysis/... ./internal/games/...` ;
vitest du dossier `features/citations` si touché ; contrôle visuel de la page Citations par le
user (témoin : la maîtrise Carabine à impulsion visible avec son image).

Statut du gate (exécutant, 2026-10-17) : VERT (sorties au journal). Vitest et `make check-types` non joués :
aucun fichier web touché. Contrôle visuel du user : OUVERT.

### E7 — Tactique : requêtes du chargement (entrée 2, D-10) — M

- [x] E7.1 Mesure avant (profil JGtm, données locales) : nombre de requêtes et temps jusqu'à
  l'affichage, au journal. Établi par le code et le relevé du 16/10 (pas de navigateur ici) : 96 requêtes = 4 fixes + 2 x 46
  cartes ouvrables ; temps jusqu'à l'affichage non mesuré (mesure en navigateur laissée au contrôle visuel du user).
- [x] E7.2 Calage du fond ajouté à `TacticalMapCard` (domaine, service, OpenAPI,
  `make generate-types`), absent pour une carte sans fond ; test Go. Champ `calage_fond`, posé par
  `service/tactical_service_fonds.go` (cartes ouvrables seulement, section de durée `tactical_maps_fonds`).
- [x] E7.3 `TacticalMapTile` : plus de `GET .../background` par tuile (calage lu de la carte), image
  en chargement paresseux (D-10) ; plan principal inchangé (mêmes clés de cache). Tests Vitest : une
  grille de N cartes ne déclenche aucune requête de calage, aucune image hors écran. REPLI de D-10 appliqué
  (image par IntersectionObserver, pas `<img loading="lazy">`) : voir journal.
- [x] E7.4 Mesure après, au journal (critère : ≤ 10 requêtes au chargement hors images visibles). 5 requêtes
  (4 fixes + le calage de la seule carte du plan) ; test Vitest à 46 cartes.

Gate : `go test ./internal/api/... ./internal/service/...` ; `make check-types` ; vitest
`features/tactical` ; `npx playwright test` des specs tactiques s'il en existe (PR vers main
seulement en CI) ; contrôle visuel du user.

Statut du gate (exécutant, 2026-10-17) : VERT (sorties au journal). Aucune spec Playwright ne joue la page
Tactique (seul `e2e/visual/readme-shots.visual.spec.ts` la photographie, serveur requis : non lancé). Mesure en
navigateur et contrôle visuel du user : OUVERTS.

### E8 — Lot film : position à la mort, score KOTH Lattice, Harvest, vies sans identité (entrées 7, 8, 9, 6) — M+

- [x] E8.1 (2026-10-18 : (a) (d) (f) (g) le 17, (b) (c) (e) le 18 par l option B du pilote, cf. journal ; 2 contextes de mort perdus sur 23 326, nommés au journal) — Positions du coup fatal et position à la mort (D-7 amendée par D-16, D-17). Historique :
  première tentative (source unique `positionOf`) arrêtée le 2026-10-17 (151 contextes perdus sur
  22 454), ref `wip/b1016-e8-1-parque` ; enquête (ref `wip/b1016-e8-1-enquete`, journal) : en base,
  8,3 % des tueurs et 5,9 % des victimes sans position, causes = lecture. Travail :
  [x] (a) passe des positions sur le journal fusionné par xuid (joueurs renommés : 0 position sur tout
  le match aujourd'hui, `killRefsFromDeaths(batch.Deaths)` avant fusion) ; commentaire faux de
  `positions.go` (« crédit-seul sans position ») corrigé ;
  [x] (b) dernière position tenue dans la même vie tant que le film n'émet pas de changement (i0 émis
  sur changement seulement), hors embarquement ;
  [x] (c) occupant d'un véhicule : position du siège par la couche véhicules déjà utilisée pour les
  tirs ;
  [x] (d) tueur déjà mort au moment du frag (projectile en vol) : lieu de sa mort, repéré comme
  position posthume (repli nommé, compté) ;
  [x] (e) `ContextesDesMorts` lit la victime par cette même lecture (source unique, test d'égalité à
  1e-6, tolérance 0,5 m retirée) ;
  [x] (f) mesure avant/après sur le corpus versionné et les 237 films à faits locaux (manquants par
  côté et par cause), critère : aucune perte, baisse des manquants conforme à l'enquête ;
  [x] (g) révisions montées une fois (D-8), rattrapage killsource/positions prévu à E8.6 et en prod.
- [x] E8.2 KOTH Lattice : émissions brutes de `SeriesTotal` autour de l'image 2268 (`26602661`) et
  5346 (`7de0b91d`) ; cause nommée (point jeté, émission hors domaine, camps, manches) ; correctif à
  la source ; critère : `7de0b91d` rend 40 sans contradiction comptée, `26602661` sans descente, les
  autres KOTH locaux inchangés (diff des séries).
- [x] E8.3 Harvest (2026-10-18, recette recos-a `24cf38183` / `5c7d2beff`) : `map_id` `ff6b8a12-6b95-4eac-8ee1-9ba4c985c2e0` (28 matchs au registre), carte Forge ; `.mvar` téléchargé par le pilote le 2026-10-18 (cache `data/cache/mvar/ff6b8a12-.../map.mvar`). CANEVAS PROUVÉ par le level_id du `.mvar` : 426470249 (0x196B6B69) -> `fo11_blank`, unicité 1/1 sur les 64 modules installés (`himap.TestPreuveLevelIDCartesDuCache`, ligne ajoutée à `preuvesLevelIDCache` ; même level_id qu'Argyle). `mapModule["Harvest"]`, catalogue de bornes régénéré 84 -> 85 cartes (diff = la seule entrée `harvest`), `film_profiles.json` (bloc `derived` seul), `map_objectives.json` (+34 objectifs, 131 cartes, hors `generated_at` aucune autre ligne), relevés de catalogue (CTF à formes 36 -> 37) et test `replaybuild`. Cuisson vérifiée sans rien publier (`replay-parc-vies`) : `e449a696` (KOTH) et `99b8ba5c` (Slayer), les deux seuls films Harvest du cache, étaient « écartés » (carte hors catalogue), cuisent `ok` (158 et 144 pistes, 0 vie sans nom) ; cf. journal. Ancien statut : `[!]` du 2026-10-17 (mvar introuvable). Texte d origine : `map_id` tiré du registre pour `e449a696` ; nature de la carte (Forge ou native) ;
  recette recos-a (preuve level_id si Forge, `mapModule`, `mapquant-build`, `mapobj-build`,
  relevés de catalogue, test `replaybuild`). Exige l'installation du jeu sur le poste. Si le mvar est
  introuvable : `[!]` justifié et retour au user.
- [x] E8.4 (2026-10-18) : mesure du parc, ventilation, trois règles générales (index des chunks
  hors table, chaîne de contradictions, puis D-18 : corps d un bot arrivé avant sa première déclaration) et révisions faites ; vies sans identité du périmètre 449 -> 15
  sur le parc entier, aucune vie nommée perdue. Reste 15 vies (les 2 de
  `3f048bf3` et `8c19b7dc` sont nommées depuis D-18), sans lecture qui les nomme (bots jamais déclarés, film sans lien
  d'identité, pistes parasites), cf. journal. Texte d'origine : Vies sans identité (D-9, exécutant Opus annoncé) : outil de mesure par film dans un
  processus séparé (test `research` ou CLI existante étendue, pas de jetable hors dépôt), lancé sur le
  parc APRÈS accord du user (durée estimée annoncée) ; ventilation par build et par cause ; pièces de
  `3f048bf3` présentées au user pour trancher ; correctifs par règle générale ; arrêt au 2e lot sans
  baisse.
- [x] E8.5 Révisions (pilote, 2026-10-18) : montées une fois sur tout E8 — `objectives-2026-10-17`
  (E8.2), `isolement-2026-10-17-positions` (E8.1), `SchemaVersion` 99 et `placement-2026-10-17`
  (E8.4), `grammar-2026-10-18` et `killsource-2026-10-18` (E8.1 b/c/e + fin d'E8.4) ; inchangés :
  `profile`, `source`, `SchemaDesFaits`. `replay-corpus-gate --base=HEAD` vert (19/19, 0 perte).
  Décisions du pilote sur les deux restes d'E8.1 : silences après un mouvement TENUS (D-16 à la
  lettre, 16 tueurs reprennent leur trace à 0,02 m médian) ; 2 contextes de mort sur 23 326 non
  placés (deux corps du même joueur à 83 m, refus d'ambiguïté commun avec `kill_positions` ;
  victime attachée à un objet ni véhicule échantillonné ni bipède), comptés, contre 148 gagnés.
- [~] E8.6 Recuisson locale -> déplacée dans la séquence d'après fusion (E10.5) (pilote,
  2026-10-18). Raison : le serveur local tourne sur `feat/v75` (schéma 98, révisions d'avant) ;
  publier des artefacts au schéma 99 et réécrire les positions avec les révisions de cette branche
  avant la fusion mettrait les données du poste en avance sur le code qui les sert. Les mesures
  (parc entier par `replay-parc-vies`, KOTH, positions sur les faits) sont faites sans rien
  publier. Séquence (poste après fusion et redémarrage du serveur, puis prod) : serveur arrêté,
  `levelup backfill-replay --only-existing` (rejeux), `levelup backfill-killsource` (réécrit
  `kill_positions`, `kill_openings`, `match_lives`, `match_death_context`, `match_death_mates`),
  usage-summary, `backfill-pad-tiers --force`. Harvest : `.mvar` téléchargé le 2026-10-18 (serveur
  arrêté, jeton de JGtm, `data/cache/mvar/ff6b8a12-.../map.mvar` et `fo11_blank.mvar`) pour E8.3.

Gate : `go test ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/...
./internal/replaybuild/...` ; `make go-api-test-gamefiles` si `internal/himap` touché ; gate du
corpus de rejeu ; contrôle visuel du rejeu EN LECTURE par le user (témoins : `7de0b91d`,
`e449a696`).
Statut du gate (exécutant, 2026-10-18) : VERT, E8 close (E8.1 à E8.5 statués `[x]`, E8.6 `[~]` déplacé à E10.5). Sorties au journal.
Le contrôle visuel du user (rejeu EN LECTURE de `7de0b91d` et `e449a696`) reste OUVERT jusqu'après la fusion : le rejeu servi
est celui de feat/v75 tant que la recuisson d'après fusion n'est pas jouée (E8.6/E10.5).

### E9 — Halo 5 au déploiement : recalcul des dérivés et démo (entrée 12, D-14) — S à M

- [x] E9.1 `--perf`, `--assists-model`, `--lusr`, `--citations` honorent `--title` (CLI) ; champ
  `title` de `POST /backfill/start` (OpenAPI + types) ; tests par titre. Fait : `cmd/levelup/cmd_backfill_titled.go`
  (un pilote `runTitledPass` et une table de passes remplacent six paires de boucles recopiées ; `cmd_backfill.go`
  730 -> 222 lignes) ; `--title` couvre aussi `--engagement-scores`, `--citations-recompute-all`, `--composite-only`,
  `--compare-formulas` ; `--csr`/`--shared-csr` refusent un autre titre. Migrations des bases DU titre
  (`applyTitleMigrationsOnDB` : `RunForDB` forçait le jeu du titre par défaut). Porte `citations.engine` par la
  capability (CLI : « non supporte pour le titre », rien ouvert ; job HTTP : avertissement `citations_ignorees`,
  y compris sous le scope « tout » ; `SyncEngine.CitationsEngineSupported`). HTTP : `domain.BackfillStartRequest.Title`,
  recherche du joueur restreinte au titre (404 sinon), moteur `NewSyncEngineForTitle` et provider shared DU titre
  (`config.SharedProviderForTitle` : `cfg.SharedProvider` est celui d'Infinite, `AcquireWriter` ignore le chemin) ;
  même correction au replay LUSR admin (`registry_lusr_gaps.go`). Fragment OpenAPI + `generated.ts` + `types.ts`.
- [x] E9.2 Chemin des sessions (`POST /settings/sessions/recalculate`) vérifié pour Halo 5 : FAUX. La route
  appelait `config.PlayerDBPath(cfg, "", gt)` et `config.SharedDBPath(cfg, "")` (titre vide = Infinite) pour CHAQUE
  profil, avec le provider d'Infinite : un profil Halo 5 n'était jamais recalculé et créait une base Infinite vide.
  Corrigé par profil (`sessionsRecalcPaths`, `SharedProviderForTitle`). Tests rouges par mutation :
  `TestSessionsRecalcPaths`, `TestPostRecalculateSessions_ProfilHalo5SurLesBasesDeHalo5`.
- [x] E9.3 « Dates de paliers » : source identifiée = `cmd/backfill_milestone_dates` (`ops.BackfillMilestoneDates`,
  `--data-root data/titles/halo_5 [--apply]`, serveur arrêté ; déjà corrigé pour la Campagne par E2 du plan du
  15/10). Elle n'est PAS dans l'image de prod (Dockerfile : `levelup-server`, `levelup`, `backfill_objective_stats`) :
  l'exécution est une décision du user (étage de build ou abandon : date non dérivable = NULL), écrite au runbook
  (« Decisions still open »).
- [x] E9.4 Séquence écrite dans le runbook de déploiement (EN) : `docs/RUNBOOK_DEPLOY_CHECKLIST.md`, section
  « Post-deploy recomputes (v7.5) », R1 à R10 ordonnées (157 matchs, engagement/perf/assists Halo 5, LUSR v2 des
  deux titres, séquence film d'E8.6, sessions, accords du pool, re-seed de la démo), mode (serveur arrêté / route
  admin), dépendances et durées connues. Rien n'est exécuté. L'ancien item E5 du runbook devient un renvoi vers R1 ;
  sa dernière phrase (`levelup backfill --skill`) citait une commande inexistante : corrigée.

Gate : `go test ./cmd/levelup/ ./internal/api/... ./internal/service/...` ; `make check-types`.
Statut du gate (exécutant, 2026-10-18) : VERT. `go test -count=1 ./cmd/levelup/ ./internal/api/... ./internal/service/... ./internal/config/... ./internal/domain/...` : tous `ok` (cmd/levelup 27 s, internal/api 43 s, handlers 26 s, service 13 s) ; avec `-tags=integration -p 1` sur `./cmd/levelup/ ./internal/config/ ./internal/service/ ./internal/api/handlers/ ./internal/api/wire/` : 5 `ok` ; `go test -count=1 ./internal/sync/ ./internal/archlint/` : `ok` (101 s, 75 s) ; `go vet` des paquets touchés sans sortie ; `golangci-lint run --new-from-rev=HEAD --build-tags=integration ./cmd/levelup/... ./internal/config/... ./internal/api/... ./internal/service/... ./internal/sync/` : 0 issue (après correction d'un goconst) ; `openapi-gen -check` à jour, `tools/check-generated-types-fresh.mjs` OK, `npm run typecheck` (cache `.tmp` purgé) sans erreur ; aucun test renommé ni retiré (baseline JSONL inchangée). Rouge prouvé par mutation : moteur sans titre dans le pilote CLI (`TestRunTitledPass_PassesRealesSurLesBasesDeHalo5` rouge, base partagée d'Infinite créée), porte de capability retirée (`TestRunTitledPass_CitationsNonSupporteesSurHalo5` rouge), chemins des sessions et moteur HTTP sans titre (`TestSessionsRecalcPaths`, `TestPostRecalculateSessions_Profil...`, `TestStartBackfill_TitreRoute` rouges).

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

- E2 (exécutant, 2026-10-17) : `StepDeviceCode.tsx` (onboarding) porte le même effet de récupération sur
  `attempt_not_found` sans test du statut `authorized`/`provisioned`. Le sondage s'arrête désormais au succès
  (hook partagé `useDeviceFlowStatus`), donc le 404 ne lui arrive plus après autorisation ; la garde explicite
  n'y est pas posée (hors périmètre E2, qui nomme `XboxLoginPage`). Non traité.
- E2 : la première vérification d'un couple (joueur, match) reste soumise au budget de lecture (3 s d'attente
  de basculement + 5 s dans `IsParticipant`) : le cache ne sert que les vérifications suivantes. La piste « aligner
  son budget sur celui de la page » de l'entrée de backlog n'est pas dans D-11 ; non traitée.
- E3 (exécutant, 2026-10-17) : le plan (section 0, ligne 4) et D-12 comptaient `watcherDaemon.Stop` parmi les étapes
  non bornées. Il est borné en interne à 8 s (`watcher.stopWaitTimeout`), mais il s'exécutait AVANT le chronomètre
  de la séquence : le pire cas réel était 8 + 35 = 43 s plus les attentes non bornées, déjà au-delà de 40 s. Le
  budget de 40 s compte désormais depuis la demande d'arrêt, watcher compris. Non traité au-delà (constat).
- E3 : `docker-compose.yml` ne porte aucun autre service qui lance le serveur ; les unités systemd ou scripts de
  déploiement hors compose (`packaging/`, `scripts/deploy.sh`) n'ont pas été relus pour un délai d'arrêt propre.
  Non traité.
- E4 (exécutant, 2026-10-17) : la prémisse « une ligne v1 postérieure masque la v2 » était fausse pour la vue lue par
  l'interface (`match_skill_rank_latest` : `start_time DESC NULLS LAST` avant `written_at`, les lignes v1 n'ont pas de
  `start_time`) et vraie pour `match_skill_rank_latest_by_type` (graphe d'évolution). Le correctif et le test portent
  sur les deux vues. Les lignes v1 déjà écrites en prod par d'anciennes exclusions restent dans la table (append-only) :
  invisibles de la vue match, mais servies par le graphe d'évolution tant qu'un rejeu v2 n'a pas écrit une ligne plus
  récente pour ce match. Non traité (un rejeu `levelup backfill --lusr` par joueur les recouvre).
- E4 : `TestNoSharedWriterOnRequestPath` (plan sécurité du 2026-10-16, L5) interdisait au recalcul d'exclusion de
  prendre l'écrivain partagé (DoS par une route que tout compte peut rejouer). La décision D-6 le remplace par des
  rafales courtes : allowlist datée. Risque résiduel à connaître : un compte qui bascule l'exclusion en boucle
  déclenche des rejeux enchaînés (la coalescence n'absorbe que les demandes simultanées) ; chaque rejeu complet de
  JGtm (1 108 matchs) prend 47 s et 24 rafales de ~2 s sur la copie mesurée. Non traité (hors D-6).
- E4 : les matchs que la v2 ne note pas (36 non deux équipes, 3 déséquilibrés sur JGtm) n'ont plus de composantes
  recalculées (la v1 en calculait pour tous les matchs d'une chaîne) ; leurs anciennes composantes restent. Les
  composantes des matchs notés sont complètes. Non traité.
- E4 : `BatchBuilder.AddLUSRComponents` n'a toujours aucun appelant de production (voir E4.4) ; le type
  `shadowParticipant` de `skill_v2_shadow.go` est mort (relevé par `unused`, antérieur à E4) ; le simulateur de
  formules (`RunFormulaSim`) retient du code TrueSkill v1 ; les constantes de calibrage de la v1 (`LUSRMaxDelta`,
  `Inactivity*`, `Tau`, `KElo`...) et leurs réexportations restent (E5.2). Non traités.
- E4 : régression de E3 corrigée parce qu'elle bloquait le gate de baseline de E4 : `cmd/server/shutdown.go` (créé
  par E3) cite `shared_social` et n'était pas dans la liste blanche de
  `TestNoUnauthorizedSharedSocialMention` (`platform/duckdb`) ; le gate d'E3 (`cmd/server`, `archlint`) ne couvrait pas ce
  test. Entrée datée ajoutée à `no_attach_on_social_test.go`.
- E4 : `golangci-lint run --new-from-merge-base=origin/main` sur les paquets touchés remonte 9 remarques toutes dans
  des fichiers non touchés par E4 (`cmd/levelup/cmd_backfill_replay_passe.go`, `migration/steps_*`,
  `sync/invariants`, `sync/killcollector` ; héritées de `feat/v75` par rapport à `main`) ; `--new-from-rev=HEAD`
  (le diff d'E4 seul) : 0 remarque.
- E5 (exécutant, 2026-10-17) : `RunForDB` (titre par défaut forcé) reste l'appel de `cmd/populate-playlists-catalog`,
  `cmd/seed-assists-model`, `cmd/levelup/migrations_cli.go:applyMigrationsOnDB` (treize appelants dans la CLI) et du
  serveur (par construction, une base par titre) : seuls les quatre CLI nommés par E5.7 prennent un titre. Non traité.
- E5 : `cmd_reset_bitmasks.go` recopie encore les constantes `pBit*`, `backfillFlagSkillForReset` et
  `mBitPVEStatsForReset` ; `matchflags` n'exporte que `BackfillParticipants` et `MBitPVEStats` (la copie du PVE pourrait
  suivre). Non traité (hors libellé E5.7).
- E5 : l'ouverture d'une player DB d'un AUTRE joueur par le pool (`config.ResolvePlayer` -> `GetOrOpen`) échoue quand le dossier
  du joueur manque ; seul le chemin des extras d'amis de la vue match le teste maintenant (`PlayerDBFileExists`). Les autres
  lecteurs d'amis ou de coéquipiers (escouade, relations) n'ont pas été relus pour ce cas. Non traité.
- E5 : `platform/duckdb/weapon_resolver.go` garde deux autres `continue` silencieux sur erreur de lecture de ligne (`:280`, résolution par clé hors arsenal, et `:365`, résolution par clé)
  — hors des fonctions nommées par E5.3. Non traité.
- E5 : l'extraction de la démo (`ops/seed_demo*.go`) ne cite pas `match_bomb_stats` (grep) :
  la vue match de la démo n'aurait donc pas ses statistiques d'Assaut. Non vérifié sur une démo, non traité.
- E6 (exécutant, 2026-10-17) : `service/citations_service.go:32` (`GetCitationsPage`) avale l'échec de
  `LoadCitationMappings` (`mappings = nil`) sans journal : une metadata illisible rend une page de citations
  VIDE au lieu d'une erreur. Hors périmètre, non traité.
- E6 : la page Citations n'applique aucun filtre : le corps `{filters: ...}` envoyé par `features/citations/queries.ts`
  n'est pas lu (`handlers/citations.go:85` ne décode que `category`). La barre de filtres de la page est donc sans effet
  sur la liste. Hors périmètre, non traité.
- E6 : le nom des tuiles (`CitationCard.tsx:48`, `truncate` sur 98 px) coupe « Maîtrise du/de la ... » après ~16 caractères :
  les maîtrises d'armes sont indiscernables par le nom seul à l'écran (infobulle = nom complet). Hors périmètre, non traité.
- E6 : le serveur ne rejoue jamais le seed des citations : `cmd/server/prebuilt_metadata.go:34` n'extrait le zip que si la
  base est absente et `deploy.yml` ne lance pas `levelup seed` (commentaire l. 333) ; seule la commande manuelle
  `levelup seed citation-mappings` (`cmd/levelup/cmd_data.go:355`) met à jour une metadata existante.
- E7 (exécutant, 2026-10-17) : `/tactical/maps` fait désormais UNE résolution de fond par carte ouvrable côté serveur
  (`MapBackgroundForMap` : une requête `MapKeysForMap` + `os.Stat` + lecture du sidecar), soit le travail que les 46
  requêtes HTTP faisaient déjà, sans les allers-retours. Coût serveur non mesuré (bases verrouillées) ; la section de
  durée `tactical_maps_fonds` le donnera dans les journaux. Une lecture groupée (une requête pour les N cartes) serait
  l'étape suivante si elle pèse (ADR 0036 : un chargement par requête). Non traité.
- E7 : la page Tactique garde `GET /tactical/{map}/background` pour la carte du plan (1 requête) et pour une carte
  sans fond le plan demande encore l'image (404 nominal) : le calage du plan ne vient pas de la grille (« plan
  principal inchangé »). Une amorce du cache `tacticalMapBackgroundFrame` depuis `calage_fond` supprimerait la requête
  restante. Non traité.
- E7 : `TacticalFond.mesure.test.ts` (instrument de parc, ignoré par défaut) servait le calage par la route ; sa page de
  cartes porte maintenant `calage_fond` (sinon les vignettes n'y demanderaient plus d'image). Corrigé car l'instrument
  aurait cessé de mesurer le chemin de l'image ; il n'a pas été rejoué (données de parc, hors gate).
- E8.4 (exécutant, 2026-10-18) : les chaînes de contradictions publiaient des NOMS FAUX, pas seulement des
  vies sans nom : 43 vies nommées à l'équipe contraire de la feuille sur `0d0bc019`, `6c79ac63`, `d8c7d880`
  (corrigées par la règle des chaînes). `5fbecd0f` porte une chaîne NON résolue (l'index des chunks d'un
  membre est tenu par un joueur non contredit) et 0 vie sans nom : ses noms sont peut-être faux (les
  comptes de morts de la feuille donnent raison aux chunks pour deux membres sur quatre). Non traité.
- E8.4 : le registre propre du collecteur (`killcollector.entreeDuRegistre`) ne reçoit pas la table du
  film : il lit les seuls chunks, donc les corps qu'un joueur porte à l'index de la table (alternance
  d'index, `1e0e7984`) y restent `index_hors_table` dans `match_lives`. Non mesuré, non traité.
- E8.4 : le journal d'E8.2 dit que `facts/fallback` est dans la fermeture d'imports de `facts/objectives` ;
  `go list -deps` dit le contraire (ni `objectives`, ni `killsource`, ni `grammar` ne l'importent). Un
  nouveau repli ne change donc aucune empreinte de couche.
- E8.4 : 2 films du cache échouent à la cuisson, « aucun chunk de données dans le film » (`e869bcdf`,
  `f3e3112f`, format 0, build vide) ; 59 films du cache (2 735) n'ont pas de faits exportés (base du
  serveur verrouillée) et n'ont pas été mesurés. Non traités.
- E8.4 : 6 vies de 3 bots que le tableau de l API porte mais que BOT_METADATA ne déclare jamais, dans un film de 38 s (`5da6fd30`,
  index 8 à 10, trois bots arrivés à 28-30 s) : un appariement par ordre d'arrivée les nommerait, ce
  serait une heuristique ; non traité.
- E8.1 (b) (c) (exécutant, 2026-10-18) : l'entame (`kill_openings`, `BuildKillOpenings`) ne lit pas le bord : un
  côté sans échantillon à l'instant décalé reste nil alors que la tenue le placerait. Non mesuré, non traité.
- E8.1 : `siegesDe` nomme pour un joueur les sièges que le repli du premier occupant lui donne hors de toute vie
  couvrante ; sur `4f77afc1` (victime 2535457217346540 à 1 111 523 ms) il lui donne un second corps vivant à 83 m,
  et la mort perd son lieu (dans `kill_positions` comme au contexte). Non traité.
- E8.1 : des lectures d'occupation attachent un bipède à un parent qui n'est ni un véhicule échantillonné ni un
  bipède (`5676a9ba`, parent 806) ; leur nature n'est pas établie (assassinat ? véhicule hors de la bande ?).
  Les 70 tueurs « à bord d'un véhicule sans échantillon » des faits locaux reprennent à p50 67 m : ils roulaient,
  le nuage `ti=40` ne porte pas leur véhicule. Non traité.
- E8.3 (exécutant, 2026-10-18) : le châssis de véhicule `b857fb95` (`e449a696` 2 vies, `99b8ba5c` 3 vies) n'est pas
  dans la table des familles : vies publiées sans sprite, marqueur neutre (repli `repli_chassis_vehicule_marqueur_neutre`,
  alarme déjà journalisée). Non traité.
- E8.3 : le corpus `-tags=gamefiles` de `internal/himap` n'est pas vert sur ce poste, indépendamment du lot : le dépôt de
  variantes `.ai/re_dump/mapvar` n'existe dans aucun checkout (113 occurrences « dépôt de variantes absent » : `TestPreuveLevelIDCartes`,
  `TestSondeBouillie`, `TestSondeCanevasCartesOrphelines`, ...) et `TestBancCliffhanger` donne ACCORD 64,4 % sous la référence 64,7 %
  (carte native, code `himap` non touché). Les trois cibles `cmd/` du corpus (`film-profiles-build`, `mapfond-build`,
  `mapstruct-build`) sont vertes. Non traité.
- E9 (exécutant, 2026-10-18) : `scheduler/auto_sync_engine.go:55` et `handlers/sync_handler.go:202` branchent `cfg.SharedProvider`
  (provider d'Infinite) sur un moteur construit pour le titre du ctx/de la requête, comme le faisaient `/backfill/start` et le replay
  LUSR admin avant E9. Non vérifié si un sync Halo 5 passe par ces deux chemins (le livesync Halo 5 résout son propre provider) ;
  `config.SharedProviderForTitle` est le correctif prêt. Non traité.
- E9 : `handlePostRecalculateSessions` parcourt `LoadPlayers()` profils `auth_only` compris ; `RecalculatePlayerSessions` ouvre
  (donc crée) leur base joueur. Antérieur à E9, inchangé. `openspartan_post_import_service.go:240` calcule aussi ses chemins sur le titre
  par défaut (import OpenSpartan, Infinite par nature). Non traités.
- E9 : `levelup backfill` n'a pas de `--skill` et le job HTTP déclare la portée `skill` « non implémentée » : les matchs sans donnée de skill
  que `reset-bitmasks` ne marque pas n'ont aucune commande de rattrapage (le runbook de E5 et l'entrée « [données/prod] » du BACKLOG citaient
  une commande inexistante : corrigés). Un rattrapage serait une passe neuve. Non traité.
- E9 : `cmd/levelup/main.go` n'installe pas `migration.SetTitleStepsProvider(halomigrations.StepsFor)` (les autres outils `cmd/` le font) :
  les migrations shared jouées par la CLI (`backfill --engagement-scores`, etc.) n'ont pas les étapes propres au titre. Sans effet sur une base
  déjà migrée au boot du serveur (cas de la prod) ; observable sur une base neuve (le test E9 pose le fournisseur lui-même). Non traité.
- E9 : `docs/COMMANDS.md` documente `backfill-vehicle-takes` comme exigeant `--force` après une re-cuisson ; la séquence film d'E8.6 ne le
  prévoit pas. Si la re-cuisson au schéma 99 change la lecture d'occupation qu'il projette n'est pas établi ici : question posée au pilote (runbook R7).


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
- 2026-10-17 E2 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E2.1 et E2.2 faits, gate vert.
  - E2.1 : cause confirmée sur pièces (`XboxLoginPage.tsx`, effet de récupération sans test du statut ; le serveur
    fait `Rotate` puis `Rebind` au succès, l'ancien sondage reçoit 404 `attempt_not_found`). Test écrit AVANT le
    correctif (`une fois autorisé : ni nouveau start ni nouveau sondage...` : 1er sondage `authorized`, suivants
    404, attente d'un intervalle de 5,6 s) : ROUGE, `expected 5 to be 1` (cinq sondages, relances de `start`
    enchaînées). Correctif : (1) `useDeviceFlowStatus` : `refetchInterval` fonction, `false` dès que le dernier
    statut lu est `authorized`/`provisioned` (prédicat exporté `isDeviceFlowSucceeded`, `setup/queries.ts`) ;
    (2) garde `if (isDeviceFlowSucceeded(status?.status)) return` en tête de l'effet de récupération. Le test de
    non-régression existant `relance automatiquement le flow sur attempt_not_found` reste vert (la relance avant
    autorisation fonctionne). Les deux occurrences du test `authorized || provisioned` du composant utilisent
    désormais le prédicat (règle des deux copies). Après le correctif : vert (`pollCount` 1, `startCount` 1).
  - E2.2 : cache `participationCache` (`wire/participation_cache.go`) : valeur zéro utilisable, clé titre + xuid +
    match (le titre en défense en profondeur : une base partagée par titre), bornée à 4 096 entrées (anneau
    d'insertion, le plus ancien sort ; Debug à l'éviction, rien sur le chemin chaud), `true` seulement : ni `false`
    ni erreur ne sont mémorisés, le 503 fermé du middleware est inchangé sur un échec non caché. Câblé dans
    `ServiceRegistry.MatchParticipation` (champ `participations`). Le profil sans xuid et le profil inconnu
    court-circuitent avant le cache. Tests (`participation_cache_test.go`) : 2 appels = 1 requête, clé titre/joueur/
    match, `false` non mémorisé (puis `true` mémorisé après synchronisation), erreur non mémorisée (y compris un
    `true` rendu avec erreur), borne et éviction du plus ancien, bout en bout avec `RequireMatchParticipation`
    (503, 503, 200, puis 200 servi par le cache base tombée). Pas de test rouge préalable : la brique est neuve,
    l'absence de la brique est une erreur de compilation.
- 2026-10-17 E3 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E3.1 à E3.4 faits, gate vert.
  - Pièces rouvertes avant de coder : séquence `main.go:1342-1484`, `shutdown_recomputes.go`, `watcher/daemon.go:229`
    (Stop borné à 8 s), `persist/queue.go` (`Drain` rend à l'échéance du contexte, `Close` attend `sendWG` sans
    borne), `.air.toml:35-46`. La séquence dépassant 80 lignes, elle est extraite hors de `main()` (fichiers
    `cmd/server/shutdown.go` et `shutdown_budget.go`) ; `main()` ne fait plus que `runShutdown(shutdownDeps{...})`.
  - Mécanisme : `shutdownClock` (départ à la demande d'arrêt, `shutdownBudget` 40 s, `shutdownCloseReserve` 5 s
    gardés pour `closeStores`). Chaque phase (`shutdownPhase{nom, borne, run(ctx)}`) reçoit `min(borne, budget
    restant - réserve)` ; les attentes sans contexte (`watcherDaemon.Stop`, `WaitGroup.Wait`, `Close`,
    `gate.WaitInFlight`) passent par `blocking`. Une phase qui dépasse produit un `slog.WarnContext` « phase hors
    budget » avec `phase`, `allotted`, `limit`, `waited`, `budget_exhausted`, puis la séquence continue. Phases, dans
    l'ordre et bornes : watcher 10 s (borne interne 8 s), http 15 s, recomputes 2 s (`coalesce.DrainWait`),
    scheduler 3 s, gate 5 s, persist.drain 5 s, persist.close 5 s, persist.worker 5 s, checkpoint 5 s ; puis
    `closeStores` (CloseAll, `reg.Close`, `closeShared`, `metaDB.Close`, relevé des fuites), toujours atteint.
    `drainCoalescedRecomputes` prend le contexte de la phase et rend un booléen (son test passe `context.Background()`).
    `shutdown_total_duration_ms` compte maintenant depuis la demande d'arrêt, watcher compris.
  - E3.1 : `stop_grace_period: 45s` sur `levelup` et `levelup-demo`. E3.3 : le commentaire périmé de `main.go`
    (`stop_timeout=20s`) disparaît avec l'extraction ; le contrat est dans le commentaire de `runShutdown` (budget,
    réserve, air : `kill_delay` 5 s ignoré sous Windows où l'arbre est tué) ; les commentaires de
    `coalesce.DrainWait` et de son test (« Docker : 10 s ») sont corrigés.
  - E3.4 : `shutdown_budget_test.go` lit `docker-compose.yml` (yaml.v3) : chaque service exige un
    `stop_grace_period` ≥ `shutdownBudget` + 5 s. ROUGE prouvé en remettant le compose de HEAD : les deux services
    échouent (« sans stop_grace_period : Docker tue le process à 10 s... ») ; vert avec les clés. Autres tests :
    durée accordée jamais au-delà du budget restant ni dans la réserve (horloge simulée) ; phase bloquée nommée en
    Warn et suivante exécutée ; trois phases bloquées : séquence rendue dans total - réserve ; phase à l'heure
    silencieuse ; bornes de phase dans ]0, total - réserve]. `TestShutdown_DrainsRecomputesBeforeCloseAll`
    (analyse de l'arbre) lit désormais `shutdown.go` : HTTP avant recalculs avant scheduler, `shutdownPhases` avant
    `closeStores`, un seul `CloseAll`, un seul `Shutdown` ; mutation prouvée (phases permutées : échec). Aucun test
    renommé ni supprimé : pas de ligne de baseline à retirer.
- 2026-10-17 E4 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E4.1 à E4.7 faits, gates verts.
  - Pièces rouvertes avant de coder : `match_recomputer.go`, `lusr_full_recompute.go`, `skill_v2_*`, `engine_postsync_scoring.go`,
    `skill_rating*.go`, les migrations de `match_skill_rank`. Constat qui corrige le plan (section 0, ligne 5) : la vue servie
    à l'interface ordonne `start_time DESC NULLS LAST` avant `written_at`, donc les lignes v1 (sans `start_time`) ne masquaient
    pas la v2 ; elles dominaient `match_skill_rank_latest_by_type` (graphe d'évolution).
  - Rouge -> vert. E4.1 `TestMatchRecomputer_ExclusionServeLaNoteV2SansLeMatchExclu` : rouge avant correctif, 11 matchs sur 12
    à 1501-1505 (v1) au lieu de 2200 dans `_latest_by_type`, `expérience v2 = 12, attendu 11`, `mu v2 = 30.174013105`
    contre 30.086840769 pour la référence rejouée sans le match exclu ; vert après E4.2 + E4.3. E4.2
    `TestRunLUSRV2Shadow_Canonical_SkipsExcludedMatches` : rouge (`processed = 3, attendu 2`, 2 lignes de note pour le match
    exclu, expérience 3), vert. E4.4 `TestPostSyncLUSRV2_WritesTheEightComponents` : rouge (`LUSRUpdated = 0`), vert.
  - Contrat d'écrivain du recalculateur : rafales courtes uniquement (reset du filigrane = 1 INSERT, puis rafales du rejeu de
    50 matchs ou 2 s), lecture des notes de performance sous un segment de lecture relâché avant la première rafale ;
    `TestMatchRecomputer_TakesTheSharedWriterInShortBurstsOnly` (2 prises pour 12 matchs, 0 lecture sous écrivain, base revenue
    en lecture seule) et `TestMatchRecomputer_NoProviderReleasesTheSharedWriterLease`. API : `RecomputeLUSRCanonicalWithAccess`
    (accès en rafales) ; `RecomputeLUSRCanonicalForPlayer` (handle tenu) l'enveloppe ; options variadiques `skill.ShadowOption`
    (`WithMedalExploit`) sur `RunLUSRV2Shadow*`.
  - Composantes : parité v1/v2 (test jetable retiré) sur 40 matchs, 275 valeurs identiques à 1e-12 avant de supprimer la v1.
    Mesure E4.6 : voir l'item.
  - Grep final (symboles v1, interrupteurs, `RecomputeAfterARTRebuild`) sur `apps docs scripts CLAUDE.md .claude Makefile
    .github docker-compose.yml .env.local.example` : 0 occurrence hors `docs/adr/0024-*.md` (historique + amendement) et
    l'en-tête RETRAIT de `scripts/check_test_baseline.sh`. Motif : `BatchComputeLUSR|batchComputeLUSR|computeSkillRatingsBatch|
    upsertLUSRRatings|writeLUSRComponentHistory|loadExistingLUSRStates|lusrResult|runLUSRV1UnderRead|RunBackfillLUSRDryRun|
    LUSRDryRunReport|LUSRPlaylistPreview|IsLUSRV2Canonical|IsLUSRV2Enabled|DefaultLUSRModeIfUnset|LogLUSRModeAtBoot|
    lusrCanonicalEnvFlag|lusrV2EnvFlag|RecomputeAfterARTRebuild|LEVELUP_LUSR_CANONICAL|LEVELUP_LUSR_V2_ENABLED`.
  - Gate (sorties réelles, un seul `go` à la fois, `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`) :
    `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` : `ok` (sync 185-196 s, sync/skill 7 s,
    sync/v2 17 s, replayartifacts 41 s, persist `ok`) ; `go test ./internal/archlint/ ./internal/service/...` : `ok`
    (archlint 68,5 s ; deux ratchets relevés puis traités : `TestNoSharedWriterOnRequestPath` -> entrée d'allowlist datée,
    `TestSyncRootPackageFrozen` 79 -> 78) ; `go test ./cmd/levelup/ ./cmd/server/` : `ok` (22,8 s et 8 s) ;
    `go vet` des paquets touchés (tags par défaut, `integration`, `psarepro`) et des 42 paquets qui importent `sync`/`sync/skill` :
    sans sortie ; `golangci-lint run --new-from-rev=HEAD` (diff d'E4) avec `--build-tags=integration` : 0 issue ;
    `--new-from-merge-base=origin/main` : 9 remarques, toutes dans des fichiers que E4 ne touche pas (voir Découvertes) ;
    `bash scripts/check_test_baseline.sh tests` : `Baseline : 9391 tests`, `Courant : 21061 tests`, tous les tests de la
    baseline présents, aucun échec de test ni de paquet, `EXIT=0` (voie film 119 s, voie DuckDB 1 396 s). Un premier passage
    de ce script avait échoué sur `TestNoUnauthorizedSharedSocialMention` (régression d'E3, réparée : voir Découvertes).
- 2026-10-17 E5 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E5.1 à E5.8 et E5.9b faits, E5.9a `[!]` (geste prod), gates verts.
  - Pièces rouvertes avant de coder : chaque symbole mort re-grepé (appelants de production : 0), `SquadUsualContexts` (le
    service interne passait bien le titre au fournisseur), `ApplyResolutionViews`, les trois DDL de `v_match_full`,
    `GetMatchMeta`, le garde de la Campagne, `OpenSnapshotShared`, `resolveWeaponMeta`, `config.ResolvePlayer`/`GetOrOpen`.
  - Rouge -> vert / mutations : garde `v_match_full` (littéral recopié dans `sync/schema.go` : échec), `GetMatchMeta` annulé
    (sans la garde : `forceLive` armé, compteur stale-schema 0 -> 1, ERROR journalisé ; avec : rien), garde de la Campagne
    (exclusion de `loadLUSRMatchIDsInWindow` retirée : échec ; exclusion de `selectRecentMatchIDs` retirée : échec, doublon
    conservé), export de snapshot (`tableMatchBombStats` retiré de la liste : `match_bomb_stats_latest` absente),
    `ApplyResolutionViews` sans `match_registry` (rendait nil avant), ami sans dossier de player DB (le pool échouait
    « chemin introuvable » : panne dite pour toute la section avant `PlayerDBFileExists`).
  - Découvertes en route : le critère « registre sur fenêtre de temps » prenait `<> ?` pour une borne (faux positif sur
    `killcollector/postsync.go`), motif resserré et testé ; deux outils opérateur dispensés. Voir section 3 pour le reste.
  - Gate (sorties réelles, un seul `go` à la fois, `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`) :
    `go test -tags=integration -p 1 ./internal/persist/... ./internal/service/... ./internal/prestige/... ./internal/migration/...
    ./internal/ops/... ./internal/openspartan/... ./internal/config/...` : `ok` ; `... ./internal/platform/duckdb/...
    ./internal/api/... ./internal/games/halo_infinite/migrations/...` : `ok` (platform/duckdb 242 s) ; `... ./internal/sync/...` :
    `ok` (sync 185 s) ; `... ./cmd/levelup/ ./cmd/apply_shared_migrations/ ./cmd/diag_bot_resolution/
    ./cmd/backfill-world-player-stats/ ./cmd/diag_lusr_player/ ./internal/archlint/` : `ok` ; `go test ./internal/archlint/
    ./internal/api/...` (tags par défaut) : `ok` ; `go vet -tags=integration` des paquets touchés : sans sortie ;
    `golangci-lint run --new-from-merge-base=origin/main` (paquets touchés, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-backlog1016`) :
    `0 issues` ; `bash scripts/check_test_baseline.sh tests` : `Baseline : 9373 tests` (9 391 - 18), `Courant : 21068 tests`, tous
    les tests de la baseline présents, aucun échec de test ni de paquet, `EXIT=0` (voie film 122 s, voie DuckDB 1 378 s).

- 2026-10-17 E6 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E6.1 et E6.2 faits, E6.3 diagnostiqué sans défaut de code, E6.3b `[!]`, gate vert.
  - E6.1. Pièces rouvertes : `computeFKFD` lisait `he.killer_xuid` / `he.victim_xuid` (colonnes absentes de `highlight_events`,
    qui porte `xuid`) et avalait l'erreur de liaison : 0/0 pour tous les titres. Rouge prouvé : avec l'ancien `computeFKFD`
    restauré (`git show HEAD:`), `TestComputeFKFD_Infinite` et `TestComputeFKFD_Halo5RepliSurLesFrags` échouent (`0 / 0, attendu 1 / 1`) ;
    verts avec le correctif. Tests unitaires purs : `TestFirstDeath`, `TestCountFirstKillsAndDeaths` (match ouvert par le joueur,
    première mort du joueur, instant négatif écarté, tueur vide et tueur absent = bot qui n'ouvre pour personne, médaille ignorée).
    La fenêtre lue est celle des matchs du joueur (`match_participants`) et applique `campaignExcl` (testé : un match Campagne Halo 5
    n'est pas compté). Le test d'intégration Halo 5 a été arrêté par le garde `TestNoRawKillScopeLiteral` (littéraux de portée
    bruts dans un `INSERT INTO match_kill_events`) : constantes `killscope` utilisées.
  - E6.2. Lecteurs de `CapCitationsEngine` relus : `games/feature.go:37` seul (feature-matrix) ; les commentaires `home.go:372`,
    `home_service.go:476`, `NavL2.tsx:40` parlent d'Halo 5 (`not_exposed`, inchangé). `synthetic_title_b` était déjà `supported`.
  - E6.3 DIAGNOSTIC (aucune lecture de la prod ; bases du poste verrouillées : « Device or resource busy » sur `metadata.duckdb`
    et son WAL ; l'API locale (:8000) exige une session et l'origine CSRF, non contournée). Copies à froid utilisées : la
    sauvegarde `data/backups/avant_rattrapage_2026-10-08/metadata.duckdb` (= le zip de production `metadata-prebuilt.zip`, 106
    lignes dont 99 actives) puis la même copie après `ops.SeedCitationMappings` (« 4 insérées, 106 mises à jour », 110 lignes).
    (1) `pulse_carbine_mastery` est présente et `enabled = true` dans les deux copies (et dans la démo locale, 88 lignes),
    catégorie `weapon`, sous-catégorie `Paria`, image `static/commendations/halo_infinite/HI_Commendations_Carabine.png`, 5 paliers.
    (2) Le seed n'est jamais rejoué par le serveur : `cmd/server/prebuilt_metadata.go:34` n'extrait le zip que si la base est
    absente, `deploy.yml` ne lance pas `levelup seed`, seule la commande manuelle `levelup seed citation-mappings`
    (`cmd/levelup/cmd_data.go:355` → `ops/seed.go:328`) met à jour une metadata existante ; la ligne est au seed depuis mars 2026
    (`98c015b82`, `ef25be1e0`) et le seed du poste du 16/10 (E5 du plan des citations) l'a mise à jour. (3) Chaîne rejouée en code
    sur la copie seedée (`CitationsRepo.LoadCitationMappings` → `analysis.MergeCitationTotals` → `BuildCitationsByCategory`, sonde
    jetable supprimée) : 103 citations actives, 6 catégories, la catégorie `weapon` compte 31 tuiles dont `pulse_carbine_mastery`
    (total 0, 5 paliers, image trouvée sur disque) ; aucun filtre côté service (`citations_service.go:25-77`), côté handler (le
    corps `{filters}` n'est pas lu, `handlers/citations.go:85`) ni côté page (`CitationsView`, `normalize.ts` : rien ne masque un
    total 0 ni une sous-catégorie). (4) Résolution du nom : les 32 citations `weapon_kills:` du seed (24 armes, 8 véhicules) se résolvent
    toutes sur la requête de `loadWeaponKeyNames` exécutée sur la copie (`hinf_pulse_carbine` → `Pulse Carbine`, FR « Carabine à
    impulsion ») ; `TestWeaponStatCitations_ResolvableName` et `TestCitationImagePaths_ExistOnDisk` (existants) verts. L'image
    `HI_Commendations_Carabine.png` est suivie par git (100 x 100, lisible), copiée dans l'image Docker (`COPY static`). Aucune maîtrise
    d'arme Infinite du seed n'est invisible ou irrésolue. CONCLUSION : pas de défaut de code ; la seule cause restante est l'état de la
    metadata de l'instance regardée (ligne absente ou `enabled` faux, seed non rejoué). Deux pistes d'affichage non vérifiées, notées
    en section 3 (le nom coupé des tuiles ; la barre de filtres sans effet).
  - REQUÊTE POUR LE PILOTE (poste, serveur arrêté ou copie à froid ; prod : lecture seule, user prévenu) :
    `SELECT citation_name_norm, enabled, category, subcategory, image_path, tier_targets FROM citation_mappings WHERE citation_name_norm IN ('pulse_carbine_mastery', 'paria_weapons_mastery');`
    attendu : 2 lignes, `enabled = true`, `pulse_carbine_mastery` en catégorie `weapon` avec l'image `HI_Commendations_Carabine.png`.
    Si la ligne manque ou `enabled` est faux : `levelup seed citation-mappings` (serveur arrêté, un seul écrivain sur la metadata), sans
    recalcul requis pour l'affichage (les totaux viennent de `match_citations_latest`). Compter aussi : `SELECT count(*), count(*) FILTER
    (WHERE enabled IS NOT FALSE) FROM citation_mappings;` attendu 110 et 103.
  - Storm (D-13) : aucune citation ajoutée ; l'équivalent Infinite est la Carabine à impulsion (`pulse_carbine_mastery`, déjà au seed
    avec son image) ; entrée « [dette/citations-2026-10-16] » retirée du backlog, trois lignes soldées retirées de `REGISTRE_REPORTS.md`.
  - Gate (sorties réelles, un seul `go` à la fois, `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`) : `go test ./internal/progression/...
    ./internal/analysis/... ./internal/games/...` : tous `ok` (66 paquets) ; `go test -tags=integration -p 1 ./internal/progression/...
    ./internal/analysis/` : `ok` (profile 27 s) ; `go test ./internal/archlint/ ./internal/api/... ./internal/domain/title/... ./internal/ops/` :
    `ok` (un premier passage d'archlint a échoué sur `TestNoRawKillScopeLiteral`, corrigé) ; `golangci-lint run --new-from-rev=HEAD~1`
    (`./internal/analysis/ ./internal/progression/profile/ ./internal/games/halo_infinite/`, avec et sans `-tags=integration`) :
    `0 issues`. Aucun test renommé ni retiré : la baseline JSONL n'est pas touchée.
- 2026-10-17 E7 (exécutant, worktree `LevelUp-wt-backlog-1016`) : E7.1 à E7.4 faits, gate vert, contrôle visuel du user ouvert.
  - E7.1 AVANT (sur pièces, sans navigateur). Pièces rouvertes : `TacticalMapsColumn.tsx` monte une `TacticalMapTile` par carte
    ouvrable ; chaque tuile appelait `useTacticalMapBackgroundUrl` (`GET /tactical/{map}/background.png`) et, par `MiniPlan`,
    `useTacticalMapBackgroundFrame` (`GET /tactical/{map}/background`), clés de cache sans paresse. Fixes : `POST
    /filters/match-ids`, `POST /tactical/maps`, coéquipiers (`useCompositionOptions`), raster de la carte du plan = 4. Formule
    2N + 4 ; le relevé du 16/10 (96 requêtes) donne N = 46 cartes ouvrables. N n'a pas été relu sur une copie à froid : la
    sauvegarde `data/backups/avant_recuisson_98_2026-10-16/` (1,8 Go) existe mais aucun client DuckDB n'est installé sur le
    poste et une sonde Go jetable est écartée par la règle du dépôt ; N = 46 reste donc DÉDUIT du relevé (si le relevé
    comptait 6 fixes, N = 45, le témoin du backlog : même ordre de grandeur). Temps jusqu'à l'affichage : non mesuré.
  - E7.2. `domain.TacticalMapCard.CalageFond *replaydoc.MapBackgroundCalibration` (`json:"calage_fond,omitempty"`), posé par
    `TacticalService.poserLesCalagesDeFond` (`service/tactical_service_fonds.go`) depuis `detail.rejeu.MapBackgroundForMap`,
    la MÊME source que la route par carte. Cartes sous le plancher non lues (pas de vignette) ; absence de fond
    (`ErrMapBackgroundNotAvailable`) = pas de champ, sans trace ; autre panne = `WarnContext` nominatif + carte sans calage,
    grille jamais arrêtée ; sans service de rejeu = aucun calage. Section de durée `tactical_maps_fonds`, compte
    `cartes_avec_fond` à la ligne « tactique: cartes jouees ». `make openapi-gen` puis `npm run generate-types` (2 lignes
    dans `generated.ts`) ; `openapi-gen -check` à jour. Tests : `tactical_service_fonds_test.go` (calage posé, carte sans
    fond, carte sous le plancher non lue, panne journalisée sans arrêter la grille, sans service de rejeu).
  - E7.3. CHOIX D-10 : REPLI par IntersectionObserver, pas `<img loading="lazy">`. Vérification demandée : `api.getBlob`
    passe par `sendRequest`, qui pose `X-LevelUp-Title` et `X-LevelUp-Locale` ; le cookie de session suffirait à un `<img>`
    même origine (le proxy Vite `/api` le transmet), mais un `<img>` n'emporte pas l'en-tête de titre, et
    `middleware.resolveTitleSlug` retombe alors sur le titre de la SESSION (puis `halo_infinite`) : un `map_id` est propre à
    un titre, l'image pourrait être résolue sur le mauvais titre, et la garde anti-fuite `X-LevelUp-Title-Resolved` du client
    ne s'appliquerait pas (doc de `api.getBlob`). S'y ajoutent le 404 nominal d'une carte sans fond (console, requête
    perdue) et le partage du cache blob avec le plan principal (clé `tacticalMapBackground(titleSlug, mapId)` inchangée).
    Réalisation : hook `lib/hooks/useVisibleUneFois.ts` (observateur à marge de 100 px, débranché à la première entrée,
    réputé visible sans `IntersectionObserver`), `useTacticalMapBackgroundUrl(playerSlug, mapId, actif = true)` ; la tuile
    passe `actif = visible && carte.calage_fond != null` (une carte sans calage ne demande jamais d'image) ; `MiniPlan`
    lit le cadre dans `carte.calage_fond` (`mapFrame`), plus aucun `useTacticalMapBackgroundFrame` par tuile (le hook reste
    pour `TacticalPlanCard`). Tests Vitest : `TacticalMapsColumn.fonds.test.tsx` (46 cartes montées : 0 appel `get`, 0
    `getBlob` ; 2 vignettes rendues visibles : exactement 2 `getBlob`, aux bonnes URL, une seule fois ; carte sans calage
    visible : 0 ; mini-plan peint sur le calage de la carte, bornes invalides) et `useVisibleUneFois.test.tsx` (5 cas).
    Rouge prouvé par mutation (image demandée sans condition : 3 des 4 tests échouent). `TacticalPage.test.tsx` : la carte du
    chemin nominal du fond porte `calage_fond` ; `TacticalFond.mesure.test.ts` aussi (voir section 3).
  - E7.4 APRÈS. Au chargement : 4 fixes + 1 calage (la carte du plan, `TacticalPlanCard`) = 5 requêtes hors images (critère
    <= 10 tenu), puis une image par vignette ENTRÉE dans le champ de vision (+ celle du plan, même clé de cache que sa
    vignette) au lieu de 46 images + 46 calages. Établi par le code et par le test à 46 cartes, pas en navigateur : la
    mesure réelle (onglet Réseau de `ascension/tactique`, profil JGtm : attendu ~5 requêtes de données + ~10 images selon la
    hauteur de colonne, sous 1 400 px la rangée défile horizontalement) reste au contrôle visuel du user.
  - Gate (sorties réelles, un seul `go` à la fois, `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`) : `go test ./internal/api/...
    ./internal/service/... ./internal/domain/... -count=1` : tout `ok`, 0 échec ; `go test -tags=integration -p 1
    ./internal/service/ ./internal/domain/` : `ok` (service 21 s) ; `go run ./cmd/openapi-gen -check` : à jour ;
    `golangci-lint run --new-from-rev=HEAD ./internal/service/ ./internal/domain/` (avec et sans `-tags=integration`) :
    `0 issues` ; `npm run typecheck` (`tsc -b`) : propre ; `eslint` des 7 fichiers web touchés : propre ; `vitest run
    src/features/tactical src/lib/hooks src/lib/query` : 36 fichiers passés, 1 ignoré (l'instrument de mesure), 425 tests
    passés, 2 ignorés. Aucun test renommé ni retiré : la baseline JSONL n'est pas touchée. Pas de spec Playwright tactique.
- 2026-10-17 E8.1 / E8.2 (exécutant Sonnet, worktree `LevelUp-wt-backlog-1016`). Les mesures ci-dessous n'ont décodé AUCUN film : elles lisent la section statborg et les positions des fichiers de faits du cache du poste (`L1_DEPOT`, 237 films lisibles sur 255, les 18 autres sont au codec/schéma périmé), un fichier à la fois.
  - E8.1 STOP D-7. `ContextesDesMorts` place la victime par `positionOf` (même instant `TempsMS*1000 + origine du film`, même tolérance ± 120 ms, même refus d'ambiguïté que `kill_positions`) ; test d'égalité à 1e-6 et tolérance 0,5 m du test `research` remplacée par l'arrondi de publication (0,02 m) : tout écrit, vert, mais PARQUÉ sur `wip/b1016-e8-1-parque` (ref locale, commit `38d2138a3`, non poussée, non intégrée à cette branche). Compte des contextes PERDUS (victime localisée avant, non localisée après), instrument `death_context_lieu_unique_research_test.go` (sur la ref) : corpus versionné (8 films) 1 110 morts à xuid, 1 054 localisées avant, 1 043 après, 12 PERDUES (1,1 %), 1 gagnée ; faits locaux (237 films) 22 685 morts, 22 454 avant, 22 307 après, 151 PERDUES (0,67 %), 4 gagnées. Cause des 151 + 12 : TOUTES « aucun échantillon du siège de la victime à ± 120 ms » (écart au plus proche échantillon : p50 325 ms, p90 838 ms, max 1 106 ms) ; 0 ambiguïté de corps, 0 « aucun siège nommé ». Écart de lieu sur les communes : p50 0,09 m, p99 0,39 m, max 8,6 m. Constat de contexte : l'origine d'horloge du film et le calage du registre diffèrent de 56 à 123 ms (corpus versionné), c'est la cause du 0,5 m historique. Décision à prendre par le user : accepter ces ~0,7 à 1,1 % de contextes en moins, ou garder le lieu approché (fenêtre d'une seconde) et le tenir égal à celui de `kill_positions` autrement. Si accepté : reprendre la ref (cherry-pick), monter `killcollector.IsolationDecoderRev` (une seule fois, au titre d'E8) et jouer le gate killcollector.
  - E8.2 CAUSE NOMMÉE : émission hors domaine non jetée + départage de `longestRun`. Émissions brutes (faits) du slot d'équipe en défaut : `26602661` slot 8 : 206864 ms A=1 B=0 ; 221567 ms A=-69 B=120 ; 237600 ms A=0 B=5 ; `7de0b91d` slot 8 : 283850 ms A=0 B=5 ; 434821 ms A=1 B=0 ; 551853 ms A=0 B=5. Le registre dit 3-1 et 1-2 : le camp du slot 8 a UN point dans les deux films. `A=0 B=5` passe la borne de domaine (0..250 sur A et B) ; plus tardive que le vrai point, elle gagne le départage de `longestRun` (à longueur égale, la dernière émission) : le point réel est remplacé par 0 (« camp 1 -> 0 » de `26602661`, « camp 0 -> 0 » à 5414 de `7de0b91d`). Ni la manche (1 manche réelle, désignateurs écrits [0 7] et [0 1 3 4]), ni le camp, ni `scoreTicksOf` ne sont en cause. Ce n'est pas une descente de la série (strictement croissante) mais une valeur 0 publiée à la place de 1. Population : sur 21 042 émissions d'équipe du composant 0 (237 films), 20 997 ont B = 0 ; 45 (35 films) ont B != 0, paires illisibles (`(0,5)` 16 fois). CORRECTIF À LA SOURCE : `emissionHorsDomaine(key, slot, v, val)` jette (même compteur `repli_emission_hors_domaine_jetee`) l'émission du score de mode d'un slot d'ÉQUIPE à canal B non nul ; un JOUEUR garde ses canaux B (Oddball ré-émet à A égale quand B bouge). Tests rouge -> vert : `score_equipe_canal_b_test.go` (émissions brutes des deux films : total [0] -> [1], 7de0b91d par manche [0 1] -> [1] ; canal B nul inchangé ; joueur inchangé). SECOND CORRECTIF NÉCESSAIRE AU CRITÈRE : la courbe juste, `7de0b91d` donne 3 points sains aux gains 39, 39, 40 et « valeur la plus fréquente » publiait 39 (contradiction table 40 comptée). `holdThresholdOf` : la valeur la plus fréquente DÉSIGNE le groupe (borne 5/4), le plus grand gain du groupe est le seuil (un intervalle sain perd un tic, n'en gagne jamais ; les fusionnés restent dehors : 73 ou 80 vs 40). Tests : `TestSeuilGardeDeuxTicsPerdusSurTroisIntervalles`, `TestSeuilGardeLeGroupeBorneLeMaximum`. DÉCISION À CONFIRMER PAR LE PILOTE : ce second correctif change l'estimateur du seuil (commentaires de `hill_hold_threshold.go` et de `regulation.toml` mis à jour) ; il est isolé dans `holdThresholdOf` et son test, retirable seul.
  - E8.2 MESURES avant/après (instruments `score_koth_emissions_research_test.go`, 237 films) : 6 films changent de série d'équipe : `03ccbe42` (CTF, 0@96190 -> 1@89134, le 1 manquant revient), `21ece4d8` (KOTH Arène, le 0 parasite à 330291 disparaît), `26602661` (camp 1 : 0@2268 -> 1@1961), `4577fcc4` (Slayer, 0@73929 -> 1@64222), `7de0b91d` (camp 0 : 0@5414 -> 1@4243), `7fce3219` (CTF, 0@488838 -> 1@78989) ; les 231 autres identiques ; tous les scores finaux des 6 films rejoignent le registre. Les 23 films KOTH locaux assemblés comme la cuisson (lignes de joueurs et scores du registre, copie à froid de la base) : seuils inchangés sauf `7de0b91d` 39 (contredit) -> 40 (accord, compteur de contradiction 1 -> 0, points lus 1 -> 3) ; `26602661` : 1961:1 pour le camp 1, plus de point à 0, 4 intervalles sains [40 40 40 40] au lieu de 3 ; `21ece4d8` : la série du camp 0 perd son 0 initial. Aucune recuisson.
  - RÉVISIONS MONTÉES (D-8) par E8.1-E8.3 jusqu'ici : `objectives.Rev` `objectives-2026-09-27` -> `objectives-2026-10-17` (chronique de `rev.go`, golden `objectives_rev.golden` régénéré par la porte gardée, empreinte `e2eaf321...9979b`) ; fixtures de contrat web `replay_schema_98_*.json.gz` et `manifest.json` régénérées (3 occurrences de `objectivesRev` par fixture, rien d'autre ne bouge) ; `film/types/testdata/shapes.golden` régénéré (ligne `revisions`). NON montées : `SchemaVersion` (98, la forme du document ne change pas ; le contenu de `scoreTimeline` est daté par `objectives.Rev`), `grammar`, `profile`, `killsource`, `source`, `IsolationDecoderRev`, `PlacementRev`. Pour E8.4 : si une couche `objectives` doit encore bouger, NE PAS monter à nouveau (même rang `objectives-2026-10-17`, empreinte du golden à recopier) ; `rev.go` ne se hache pas lui-même, donc toucher sa chronique est libre, mais tout autre fichier du paquet `facts/objectives` (hors tests) ou de sa fermeture d'imports (dont `facts/fallback`) change l'empreinte.
  - Gate E8.2 : `go test ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/... ./internal/replaybuild/... ./internal/archlint/ -count=1` : 22 paquets `ok`, 0 échec ; `golangci-lint run --new-from-rev=HEAD` (film/internal/facts, film/replay, et avec `--build-tags research`) : 0 issues. `internal/himap` non touché.
  - E8.3 `[!]` Harvest. `map_id` (registre, copie à froid `avant_recuisson_98_2026-10-16`, `diag_q`) : `ff6b8a12-6b95-4eac-8ee1-9ba4c985c2e0`, 28 matchs (Squad : Slayer 10, CTF 10, KOTH 6, Land Grab 1, Bombe neutre 1) ; `e449a696` en fait partie. NATURE : Forge. Pièces : l'installation du jeu (`D:/SteamLibrary/.../deploy`) ne porte aucun module de ce nom (30 modules `multi`, identiques sous `any`, `ds`, `pc`, tous déjà au catalogue de bornes via `mapModule`), `CartesForge` et `mapModule` ne la citent pas, et aucun module natif ne s'appelle ainsi. Le canevas se prouve par le `level_id` du `.mvar` de la variante (méthode de recos-a `24cf38183`, sonde `himap.TestPreuveLevelIDCartesDuCache`) ; or `data/cache/mvar/` du poste ne porte que 8 variantes (dont les trois de recos-a), pas celle de `ff6b8a12`, et la chaîne des films ne l'a pas déposée (cuisson refusée « hors catalogue » avant le téléchargement). BLOCAGE : la variante ne se télécharge que par l'API UGC avec le jeton d'un joueur (`mapobj-build --map-id`, ADR 0023, rotation possible du refresh token pendant que le serveur local tourne) : geste non joué ici, hors de ce que le brief d'E8 autorise ; aucun canevas n'est deviné (une borne fausse écrirait des coordonnées fausses sans signal). Pour reprendre : `go run ./cmd/mapobj-build --player <gamertag suivi> --map-id ff6b8a12-6b95-4eac-8ee1-9ba4c985c2e0 --save-mvar data/cache/mvar/ff6b8a12-6b95-4eac-8ee1-9ba4c985c2e0` (vérifier l'arborescence de dépôt écrite par `saveVariantFile`), puis ajouter `{"Harvest", "ff6b8a12-...", <canevas>}` à `preuvesLevelIDCache` (`himap/sonde_levelid_cache_gamefiles_test.go`) et `"Harvest"` à `mapModule`, relancer `mapquant-build` et `mapobj-build --from-file`, relevés de catalogue (formes CTF, socles neutres, fond) et test `replaybuild` comme `5c7d2beff`. Le témoin `e449a696` est au score à la SECONDE (un tic = un point) : il ne mesure pas le seuil de garde. Aucun fichier du dépôt modifié par cet item.
- 2026-10-17 E8.1 reprise (exécutant Opus, worktree `LevelUp-wt-backlog-1016`, D-16/D-17). Aucun artefact écrit, aucune base du poste touchée ; un seul film décodé (chronométrage, sous `filmproc.AcquireSolo`). Statut : PARTIEL, (a) (d) (f) (g) faits, (b) (c) (e) arrêtés sur deux décisions à prendre par le user (ci-dessous).
  - (a) FAIT. `collectPositions` place les couples du JOURNAL FUSIONNÉ (`fusionne.Deaths`, identités du crédit), plus la liste pré-fusion du film ; `buildPositionRows` reçoit le journal et en tire les couples ET les morts. Commentaires faux corrigés (`positions.go`, `collector_run.go` : « un kill crédit-seul n'a structurellement aucune position »). Test : `TestCollectPositions_LeJournalFusionneOuvreLaPasse` (un couple résolu par le seul crédit va jusqu'à la résolution de carte ; l'ancienne lecture s'arrêtait à « rien à placer »).
  - (d) FAIT. `replay/killpos_posthume.go` : tueur sans position directe, mort du journal STRICTEMENT avant le frag, aucune vie NOMMÉE à son nom ouverte entre les deux : posé au lieu de sa mort par `positionOf` à l'instant de sa mort ; sinon nil. Repli nommé `repli_position_posthume_du_tueur` (registre `replay/positions`, condition `film_muet`, compteur de la passe, expvar `killsource_repli_position_posthume_du_tueur`), rapport `KillPosReport.Posthumous` / `PosthumousUnplaced`, journal de passe `tueurs_posthumes` / `tueurs_posthumes_sans_lieu`. L'entame (`BuildKillOpenings`) n'applique pas la règle. Entrée unique `replay.EntreeDesPositions` (Positions, Registre, Kills, Morts, OrigineUS, Fallbacks). Tests rouge -> vert : `killpos_posthume_test.go` (posé au lieu de sa mort et compté ; réapparu = non posthume ; même milliseconde = non posthume ; mort non localisable = nil compté à part ; entame sans règle). Limite écrite : une réapparition dans une vie NON nommée n'est pas vue (résidu E8.4).
  - (f) MESURE (instrument `kill_positions_e81_mesure_research_test.go`, couples du journal et lignes `kill_positions_latest` exportés de la copie à froid `data/backups/avant_recuisson_98_2026-10-16`, registre du collecteur `REG_COLLECTEUR=1`) :

    | corpus versionné, 8 films, 1 097 frags | avant (base) | après (a) | après (a)+(d) |
    |---|---|---|---|
    | tueur sans position | 131 | 110 | 80 |
    | victime sans position | 82 | 64 | 64 |
    | ligne absente (aucun des deux) | 34 | 26 | 23 |

    | faits locaux, 233 films, 22 535 frags | avant (base) | après (a) | après (a)+(d) |
    |---|---|---|---|
    | tueur sans position | 1 476 | 837 | 491 |
    | victime sans position | 1 028 | 369 | 369 |
    | ligne absente (aucun des deux) | 685 | 64 | 51 |

    Posthumes : 30 posés + 2 sans lieu (versionné), 346 posés + 10 sans lieu (local ; enquête : 355). PERDUS (côté placé avant, nil après) : 0 tueur, 0 victime contre la base, 0 / 0 contre (a), sur les deux corpus. Écarts > 1 mm sur les côtés placés avant et après : 142 tueurs et 19 victimes (local), max 0,489 m : positions des faits cuits contre balayage du collecteur, pas le placement (même fonction). Causes restantes (classes de l'enquête), local, tueurs : immobile 220, véhicule 200 (+ embarqué 9), en mouvement 51, posthume non placé 10, ambigu 1 ; victimes : immobile 162, véhicule 138 (+ 2), en mouvement 60, sans échantillon avant 6, ambigu 1. Contextes de mort : 0 perdu (`ContextesDesMorts` non modifié). Baisse conforme à l'enquête (678 -> 90 lignes absentes attendues par (a) sur 240 films ; 685 -> 64 mesuré ici sur 233).
  - (b) (c) ARRÊTÉS, DEUX DÉCISIONS. Pièces (instruments `kill_positions_regles_research_test.go` et `kill_positions_cout_vehicules_research_test.go`) :
    1. LA PASSE DES POSITIONS NE LIT AUCUN VÉHICULE. La seule lecture film de « à bord de quel véhicule » est l'occupation `object-parent-state` (i10), lue par la marche des trames ; les événements d'embarquement ne nomment pas le véhicule (`VehicleSlotValid` faux à l'embarquement) et ne couvrent que 1 à 9 cas. Sans i10, « hors embarquement » (b) n'est pas vérifiable : une tenue poserait l'occupant au point où il est monté (enquête : saut p50 35 m). Chronométrage sur `4f77afc1` (BTB, 460 865 positions, contextes neufs) : pont du collecteur 5,35 s ; marche de killsource 2,33 s (déjà payée, ET elle récolte l'occupation : `canalDesMorts.h.rides`, non exposée par `LectureDeKillsource`) ; seconde marche `{Morts}` 5,19 s ; images-clés + nuage `ti=40` 2,88 s ; événements 1,00 s. Option A : seconde marche + `ti=40` dans la passe, environ +8 s par film BTB (doublement de pont + marche), aucune révision de grammaire. Option B : exposer l'occupation de la marche de killsource (modification de `grammar`, montée de `grammar.Rev`, chaîne `facts.Rev`/`killsource.Rev`, faits cuits du parc périmés, fixtures du corpus à régénérer) + `ti=40` (+2,9 s). À trancher par le user.
    2. LES SILENCES SANS LECTURE DE VÉHICULE APRÈS UN MOUVEMENT. Faits locaux, verdict i10 au dernier point du siège : tueurs immobiles 217/220 sans lecture (tenue juste, reprise au même point p50 0,000 m) ; tueurs « en mouvement » 51 : 35 à bord lus (25 avec véhicule à ± 120 ms), 16 sans lecture ; victimes immobiles 166/168 sans lecture ; victimes « en mouvement » 60 : 11 à bord lus, 47 sans lecture ; véhicules (classe de l'enquête) tueurs 202 : 142 placeables, 56 parent sans aucun échantillon `ti=40` (véhicule jamais répliqué, à sa naissance), 4 ; victimes 141 : 75 placeables, 31 parent sans échantillon, 34 dernière lecture « libre ». Les 47 + 16 sans lecture ne se séparent des immobiles que par la vitesse des deux derniers points (heuristique) ; pour les tueurs, la reprise après le silence en départage une partie (11 des 18 exemples à moins de 0,4 m, d'autres à 5-31 m) ; pour les victimes il n'y a pas de reprise (le point suivant est la réapparition, environ 8 s plus tard). D-16 au pied de la lettre (« tenue tant que le film n'émet pas de changement, hors embarquement ») les tiendrait ; la consigne du pilote (« sinon sans position, comptés, pas d'heuristique ») n'a pas de critère lu qui les isole. À trancher par le user (les tenir ; ou lire la vélocité `i1` du dernier record, non capturée par le collecteur ; ou laisser nil ces silences).
  - (e) ARRÊTÉ, dépend de (b) : les 151 contextes perdus de la première tentative sont tous des victimes sans échantillon à ± 120 ms, c'est-à-dire des tenues. Sans (b), la source unique fait perdre des contextes (contraire à D-16). `ContextesDesMorts` et la tolérance 0,5 m du test `research` restent inchangés ; l'instrument parqué `death_context_lieu_unique_research_test.go` n'est pas repris (il reste sur la ref).
  - (g) RÉVISIONS. Montée : `killcollector.IsolationDecoderRev` `isolement-2026-10-14-coequipiers` -> `isolement-2026-10-17-positions` (chronique dans `isolation_facts.go` ; `docs/COMMANDS.md` + `docs/FR/COMMANDS.md`). Pour E8.4/E8.6 : NE PAS remonter cette constante dans E8 (si (b)/(c)/(e) arrivent dans E8 : même valeur, chronique complétée, tant qu'aucun `backfill-killsource` ne l'a publiée). NON montées : `PlacementRev` (placement inchangé, réécrit de toute façon après les vies), `SchemaVersion` (document de rejeu inchangé), `grammar`, `profile`, `source`, `killsource.Rev`, `objectives.Rev` (déjà monté par E8.2). Nouveau nom au registre des replis (`facts/fallback`) : `repli_position_posthume_du_tueur`.
  - Rattrapage à rejouer à E8.6 (local, serveur arrêté) puis en prod : `go run ./cmd/levelup backfill-killsource` (hors ligne, cache de films ; `--status` d'un autre terminal ; reprise en relançant) : tout match dont `match_lives` est sous l'ancienne révision est redécodé, `kill_positions` et `kill_openings` réécrites (append-only, lues par `_latest`). Aucune recuisson de rejeu nécessaire pour E8.1.
- 2026-10-17 (pilote) : E8.1 (b), (c), (e) — décision technique du pilote : option B (exposer
  l'occupation `i10` déjà récoltée par la marche de killsource), pas de seconde marche (+8 s par film
  BTB à chaque synchro, pour toujours). La montée de `grammar.Rev` / `facts.Rev` / `killsource.Rev`
  qu'elle exige est faite UNE fois avec celle d'E8.4 (D-8) : E8.4 passe avant, puis E8.1 (b)(c)(e)
  sur la même révision. Silences après un mouvement sans véhicule lu : re-mesurés une fois
  l'occupation exposée ; ceux qui restent sans lecture de véhicule sont laissés sans position et
  comptés (pas de critère de vitesse).
- 2026-10-17/18 E8.4 (exécutant Opus, worktree `LevelUp-wt-backlog-1016`). Statut : PARTIEL, `[!]` justifié ci-dessous. Aucun artefact publié dans `data/`, aucune base du serveur touchée (une COPIE de la copie à froid `avant_recuisson_98_2026-10-16` a servi à lire `is_firefight` de deux matchs).
  - OUTIL. La mesure du 16/10 était une boucle de shell autour de `replay-build` (`parc_run.sh` d'un scratchpad, faits exportés par `replay-facts-export`). Remplacée par une sous-commande du dépôt, `levelup replay-parc-vies` (`cmd/levelup/cmd_replay_parc_vies*.go`) : un film à la fois dans l'enfant borné `internal/replaychild` (verrou solo en attente bornée, sentinelle, priorité basse), aucune base ouverte, rien écrit sous le dépôt (`SansEcritureDesFaits`), reprise par `films.tsv`, identité et provenance de CHAQUE piste dans `identites/<id>.tsv`, `--comparer <avant> <après>` (vies nommées, perdues, noms et équipes changés, pistes redécoupées). Faits : les 2 676 exports du 16/10, recopiés dans `parc_faits`. Journal enrichi (diagnostic durable) : la contradiction table/chunks porte son détail `xuid:siege->controle`, l'alarme `index_hors_table` les index lus des vies restées sans nom.
  - MESURE AVANT (parc de 2 676 films, binaire de HEAD `48f50eccf` + outil) : 2 660 cuits, 14 écartés (Forge hors catalogue), 2 en échec (aucun chunk de données), 0 panique ; 2 376 vies sans identité sur 43 films, les mêmes 43 que le 16/10. EXCLUS et comptés à part : 5 films de build inconnu (`a349fea8` 413, `47d20b5d` 298, `13b00e35` 285, `50247b26` 244, `03af54c3` 226 = 1 466) et 2 films de mode non géré, Firefight (`edc5daf6` 246, `8c12fd58` Gruntpocalypse 215, `is_firefight` vrai au registre = 461). PÉRIMÈTRE : 36 films, 449 vies. Par build : HI_1_9_0 1 film / 14 vies ; HI_1_10_0 4 / 60 ; HI_1_11_0 8 / 117 ; HI_1_12_0 2 / 22 ; HI_1_13_0 21 / 236.
  - CAUSES (provenance du lien publié + journal enrichi), témoin par cause :
    1. index des chunks hors table, 324 vies (31 films) : la table du film assied un joueur à `i`, les chunks (concordants sur tous les chunks) le lisent à `j`, et ses corps portent `j`, seul ou en alternance avec `i`, sans jamais se chevaucher. Témoin `4aa3a9b5` : « Tru Mr Coop », table 6, chunks 8, aucune vie au rejeu, 17 corps d'index 8 dont 12 sans nom et 5 que le relais donnait à « 343 PardonMy » alors que le corps propre de ce bot vivait au même instant (Coop a changé d'équipe après le départ d'un joueur). Sur les 8 films du parc à contradiction sans vie anonyme, l'exclusion temporelle nommait déjà ces corps du MÊME joueur (12/12 sur `186199b7`).
    2. chaîne de contradictions, 108 vies (7 films) : l'index des chunks d'un joueur est le siège de table d'un autre joueur contredit (ou l'index qu'ils donnent à un joueur que la table n'assied pas, `b1b62f38`, où le complément produisait une collision qui retirait les deux liens). Témoin `0d0bc019` : 4 joueurs 2->19, 17->20, 19->21, 20->22 ; 32 vies sans nom ET 10 vies d'index 19 publiées « misfitXD999 » (équipe 1 à la feuille) dans l'équipe 0.
    3. bot sur un index partagé, juste avant la première déclaration BOT_METADATA du bot qui arrive, 2 vies : `3f048bf3` slot 540, `8c19b7dc` slot 533. Suspendu à la réponse du user (ci-dessous).
    4. bots que BOT_METADATA ne déclare jamais, 6 vies : `5da6fd30` (film de 38 s, 3 bots aux index 8 à 10).
    5. film sans aucun lien d'identité, 7 vies : `72c48587` (table refusée pour vacant intercalé, aucune lecture d'index dans les chunks, aucun record de création lu).
    6. pistes parasites hors plage des bipèdes, 2 vies : `7aa132d1` slots 36 et 292, frames 10504-10520, `sans_record`.
    Aucune vie de la cause « siège de bot » hors du cas 3, ni de la cause « réapparition dans une vie non nommée » (limite écrite à E8.1 (d)) parmi les 449 ; la cause « bot relayé » existe, mais sur des vies NOMMÉES à tort (12, corrigées par la règle 1), pas sur des vies sans nom.
  - RÈGLE 1 (lot 1), `replay/identity_registry_index_des_chunks.go` : l'index `j` des chunks nomme EN PLUS les corps qui le portent quand aucun joueur de la table effective ni aucun bot déclaré ne le tient et qu'aucun de ces corps ne chevauche un corps du même joueur nommé autrement (sinon l'index est retiré, refus compté) ; le pont slot -> index garde l'index que le corps porte (sans quoi l'équipe lue par index se perdait : 144 vies passaient à « sans équipe » au premier essai, corrigé). Repli nommé `repli_index_des_chunks_hors_table` (registre `replay/identites`, compteur branché). La table du film ne perd aucun siège. Effet (44 films à contradiction) : 449 -> 125 vies sans nom, 0 perdue, 12 noms changés (vies que le relais donnait à un bot dont le corps vivait en même temps), 1 équipe changée (-1 -> 0).
  - RÈGLE 2 (lot 2), `replay/identity_registry_chaines.go` : les membres d'une chaîne prennent leur index des chunks dans la table effective (avant le retrait des collisions), leur ancien siège reste un index supplémentaire sous la même garde ; tout ou rien si un index des chunks est tenu par un joueur non contredit. Effet : 125 -> 17, 0 perdue, 94 noms changés, tous sur les corps des membres de chaîne (6 films).
  - CONTRÔLE INDÉPENDANT, pont par morts (la mort qui termine chaque vie, comparée au nom lu) : sur les 44 films, concordances 4 655 -> 5 039 (lot 1) -> 5 178 (lot 2), discordances 488 -> 510 -> 446 (vies nouvellement nommées vérifiées à 94,6 % ; sur les chaînes les discordances tombent de 27 à 13 sur `0d0bc019`, 15 à 4 sur `6c79ac63`, 37 à 27 sur `a3aab482`). Parc entier : concordances 254 069 -> 254 592, discordances 17 937 -> 17 895, 36 films bougent.
  - VALIDATION PARC ENTIER (binaire final, 2 676 films, fin 01:06) : 2 660 comparés, vies sans identité 2 376 -> 1 944 (périmètre 449 -> 17 ; exclus inchangés 1 466 + 461), nommées 318, PERDUES 0, noms changés 106 (12 relais + 94 chaînes, détail par piste dans `parc_cmp_final.txt` du scratchpad), 1 équipe changée (-1 -> 0), aucun changement hors des 31 films fautifs. Résidu par build : HI_1_13_0 4 films / 15 vies, HI_1_11_0 1 / 2. Le lot 1 seul a été mesuré sur les 44 films à contradiction (les deux règles ne se déclenchent que là) et sur 566 films du parc (passe interrompue pour économiser 5 h) : 0 perdue.
  - ARRÊT (D-9) : deux lots, chacun en baisse ; pas de troisième lot : les 15 vies restantes n'ont aucune lecture du film qui les nomme, et les 2 autres attendent le user.
  - QUESTION AU USER, `3f048bf3` (Arena, 4 contre 4, chrono du rejeu, 0:00 au coup d'envoi). Deux bots se sont succédé sur la même place de participant : « 343 Ritzy », équipe 1, entré vers 0:51 à la place de H ImpacT H et reparti vers 1:05-1:09 à l'arrivée de Matisfaction1 ; puis « 343 Lacuna », équipe 0, entré vers 1:40-1:44 à la place de VoguishOcean04 et reparti vers 1:50-1:54 (feuille de match, recalée sur le film à 2 s près). Le film déclare Lacuna à partir de 1:45,1 seulement ; il lit l'entité de Ritzy vivante quand le corps de 0:52 naît.
    (a) Le personnage qui court de 0:52 à 1:01 dans l'équipe 1 : le rejeu le montrait « 343 Lacuna » (équipe 0) avant le 16/10, il le montre « 343 Ritzy » (équipe 1) depuis. Les pièces vont toutes vers Ritzy (Lacuna n'est pas encore dans la partie, et l'équipe est celle de Ritzy). Est-ce bien Ritzy ?
    (b) Le personnage qui n'apparaît qu'une demi-seconde, de 1:44,4 à 1:44,9, juste avant que Lacuna soit déclaré : avant le 16/10 il s'appelait « 343 Lacuna », il est sans nom depuis. Le film le place dans l'équipe 1 alors que Lacuna joue dans l'équipe 0. Deux lectures : c'est l'apparition de Lacuna (vous verriez « 343 Lacuna » une demi-seconde, à la couleur de l'équipe 1 que le film lui donne à cet instant) ; ou ce n'est pas une vraie vie (un corps fantôme à l'arrivée du bot, qui ne devrait pas s'afficher du tout). Laquelle ? La même figure existe sur `8c19b7dc` (1,6 s, juste avant l'arrivée de « 343 Godfather », équipe cohérente cette fois) : la réponse vaudra règle pour les deux.
    Aucun code n'est écrit pour ce cas.
  - RÉVISIONS (D-8, une par couche, aucune déjà montée n'est remontée) : `SchemaVersion` 98 -> 99 (contenu seul, verdict « republier », chronique v99, golden de forme, goldens d'assemblage des 8 builds où seule la ligne de schéma bouge, fixtures de contrat web `replay_schema_99_*` et manifeste régénérés, contenu identique hors numéro de version, plafonds écrits de la chronique 3402 -> 3435 et de `structure_test.go` 1454 -> 1458) ; `killcollector.PlacementRev` `placement-2026-10-16` -> `placement-2026-10-17` (le document interne des portages lit le même registre). NON montées : `grammar.Rev`, `SchemaDesFaits`, `killsource.Rev`, `profile`, `source`, `objectives.Rev` (déjà `objectives-2026-10-17` par E8.2), `IsolationDecoderRev` (déjà par E8.1). Pour E8.1 (b)(c)(e) : AUCUNE montée de `grammar` / faits / `killsource` n'est faite ici, elle reste à faire une fois par l'exécutant suivant ; `SchemaVersion` 99 et `placement-2026-10-17` sont pris, à réutiliser sans remonter tant qu'aucune recuisson ne les a publiés.
  - À REJOUER À E8.6 : recuisson (« republier » depuis les faits suffit pour le schéma 99) des films à contradiction cuits localement et en prod ; `backfill-killsource` pour le placement des vies (`PlacementRev`). Les 36 films fautifs ne sont pas cuits localement aujourd'hui.
  - ÉCART DE MÉTHODE À SIGNALER : un `python3` a servi une fois à éditer trois fichiers Go en cours de session (règle « jamais Python ») ; les fichiers ont été repris à la main et normalisés par `gofmt`, aucun script n'est versionné.
- 2026-10-18 E8.1 (b) (c) (e) (exécutant Opus, worktree `LevelUp-wt-backlog-1016`, D-16, option B du pilote). Aucun artefact écrit dans `data/`, aucune base touchée ; les mesures relisent les faits cuits du cache (`L1_DEPOT`) et les TSV du 17/10 (couples du journal, lignes `kill_positions_latest` de la copie à froid), registre du collecteur (`REG_COLLECTEUR=1`). Statut : `[x]`.
  - OCCUPATION EXPOSÉE (option B). `grammar.LectureDeKillsource.Occupation` rend les lectures `object-parent-state` (i10) que le canal des morts récoltait déjà pendant la marche de killsource (aucun bit lu autrement, aucune seconde marche) ; `killsource.Result.Occupation` (hors sérialisation) les porte au collecteur. Test `TestLOccupationDeLaMarcheDeKillsourceEstCelleDeLaCuisson` (bobine de la marche) : égale à celle de `ScanMarcheDesTramesAvec{Morts}`. Nuage des véhicules : `replay.PositionsDesVehicules` (même lecture que la cuisson, `nuageDesVehicules` partagé avec `decodeFilmVehicleScan`), porte F-1 hors emprise appliquée.
  - RÈGLE, une seule lecture (`replay/killpos_a_l_instant.go`, `lectureALInstant.placer`) pour les deux côtés de `kill_positions`, le lieu du tueur posthume et la victime du contexte de mort : (1) ÉCHANTILLON à ± 120 ms (un seul siège, sinon silence) ; (2) TENUE (b) : sinon le dernier échantillon d'avant l'instant, si un seul siège le porte, si le journal ne porte aucune mort du joueur entre lui (moins la tolérance) et l'instant, et si aucune montée à bord ne court sur le silence ; (3) SIÈGE (c) : une montée (lue depuis `vehicleSeatTol` avant le dernier échantillon, sans descente ni retour du bipède au flux) pose le joueur à la position du véhicule parent à ± 120 ms ; sinon nil, compté `AboardNoVehicle` ; montée puis descente pendant le silence : nil, compté `Disembarked`. Sans lecture du bord (`LectureDuBord.Lue` faux, l'entame), seule la voie (1) s'applique : `kill_openings` inchangée. Comptes : `KillPosReport.Held/Seated/AboardNoVehicle/Disembarked`, journal de passe `cotes_tenus`, `cotes_au_siege`, `cotes_a_bord_sans_vehicule`, `cotes_debarques`, expvar `killsource_positions_silences_sans_lieu`.
  - SILENCES APRÈS UN MOUVEMENT SANS VÉHICULE LU, re-mesurés une fois l'occupation exposée (instrument `kill_positions_silences_research_test.go`, faits locaux) : tueurs à bord 245 (véhicule à ± 120 ms 171, véhicule sans échantillon 70, échantillon après seulement 4), sans lecture d'occupation : même quantum aux deux derniers points 212 (reprise après le silence p50 0,000 m, p90 0,020), lent 7 (reprise <= 0,02 m), en mouvement 16 (reprise p50 0,019 m, p90 5,0, max 6,4) ; victimes à bord 118, débarquées 35, sans lecture : même quantum 153, lent 8, en mouvement 41. DÉCISION D'EXÉCUTION À CONFIRMER PAR LE PILOTE : aucun critère LU ne sépare un silence « après un mouvement » d'un arrêt (seule la vitesse des deux derniers points le fait, et le brief l'exclut) ; ces silences sont donc TENUS comme D-16 le dit (« tant que le film n'émet pas de changement, hors embarquement »), et la reprise montre que les tueurs concernés s'étaient arrêtés (p50 0,02 m). Variante mesurée et non retenue : tenir seulement quand les deux derniers quanta sont égaux (lecture sans seuil) ferait perdre 27 contextes de mort aux faits locaux et 3 au corpus versionné (contraire au critère « 0 contexte perdu »).
  - Tests rouge -> vert (`killpos_a_l_instant_test.go`, mutation « voie 1 seule » : 6 tests rouges) : tenue au dernier échantillon et nil sans lecture du bord ; la tenue ne traverse pas une mort ; siège du véhicule ; à bord sans échantillon du véhicule ; débarqué ; montée suivie d'un retour au flux ; (e) `TestLeLieuDuContexteEstCeluiDeKillPositions` : lieu du contexte = victime de `kill_positions` à 1e-6 pour l'échantillon, la tenue et le siège, sous une origine du film 80 ms avant le calage du registre. `death_context_test.go` : l'entrée de référence pose l'origine et la lecture du bord ; le cas « mort sans lieu » porte la mort qui clôt la vie (la tenue ne la traverse pas). Tolérance 0,5 m de `service/tactical_reel_research_test.go` remplacée par l'arrondi de publication (0,02 m).
  - MESURE `kill_positions` (instrument `kill_positions_e81bc_mesure_research_test.go`) :

    | corpus versionné, 8 films, 1 097 frags | base | (a)+(d) (HEAD) | +(b)(c) |
    |---|---|---|---|
    | tueur sans position | 131 | 80 | 6 |
    | victime sans position | 82 | 64 | 8 |
    | ligne absente | 34 | 23 | 0 |

    | faits locaux, 233 films, 22 535 frags | base | (a)+(d) (HEAD) | +(b)(c) |
    |---|---|---|---|
    | tueur sans position | 1 476 | 491 | 78 |
    | victime sans position | 1 028 | 369 | 85 |
    | ligne absente | 685 | 51 | 2 |

    Côtés tenus 90 / 437, au siège 38 / 253, à bord sans véhicule 11 / 110, débarqués 0 / 35 (versionné / local) ; posthumes posés 32 / 353, sans lieu 0 / 3 (le lieu de leur mort se lit maintenant par la même lecture). PERDUS : 0 tueur, 0 victime contre la base et contre (a)+(d), sur les deux corpus ; aucun côté placé des deux côtés ne bouge de plus d'1 mm. Restent sans position (local) : tueurs à bord sans échantillon du véhicule 74, muets 4 (deux corps 1, déjà mort non posé 3) ; victimes à bord sans véhicule 36, débarquées 35, muettes 14 (deux corps à ± 120 ms 1, aucun échantillon avant 6, mort du journal pendant le silence ou deux corps au dernier point 7).
  - MESURE contextes de mort (instrument repris de la ref parquée, `death_context_lieu_unique_research_test.go`) : corpus versionné 1 110 morts à xuid, localisées avant 1 054, après 1 102, PERDUES 0, gagnées 48 ; faits locaux (247 films) 23 562 morts, avant 23 326, après 23 472, PERDUES 2, gagnées 148 ; écart de lieu sur les communes p50 0,09 m, p99 0,39 m, max 8,6 m (l'origine du film contre le calage du registre). Les 2 pertes, nommées : `4f77afc1` (victime 2535457217346540 à 1 111 523 ms) : deux sièges nommés portent un échantillon à ± 120 ms, 547 à -32 ms et 565 à +5,8 ms, à 83 m l'un de l'autre — `kill_positions` refuse aussi ce lieu (deux corps pour un joueur), le second siège n'étant nommé que par le repli du premier occupant ; `5676a9ba` (victime 2535442484776177 à 401 879 ms) : montée à -632 ms sur un parent (806) qui n'a aucun échantillon, ni de véhicule ni de bipède, descente à +9 ms. Critère « 0 contexte perdu » : tenu sur le corpus versionné, 2/23 326 aux faits locaux, à trancher (le lieu suit `kill_positions`, qui n'en a pas non plus).
  - RÉVISIONS (D-8, une par couche ; à reprendre telles quelles par E8.6) : `grammar.Rev` `grammar-2026-10-16` -> `grammar-2026-10-18` (chronique, golden `grammar_rev.golden`) ; `killsource.Rev` `killsource-2026-10-16` -> `killsource-2026-10-18` (chronique, golden ; la porte `Result.LineByLinePublishable` déplacée telle quelle dans `kill_publication.go` pour le seuil de 500 lignes) ; `objectives.Rev` reste `objectives-2026-10-17` (empreinte recopiée au même rang : seule la valeur amont de `grammar.Rev` change) ; `killcollector.IsolationDecoderRev` reste `isolement-2026-10-17-positions` (chronique complétée, `docs/COMMANDS.md` + FR) ; `SchemaVersion` reste 99 (chronique v99 complétée : `coverage.decoder` publie les deux révisions, verdict du parc `redecoder`) ; `PlacementRev`, `profile`, `source`, `SchemaDesFaits` inchangés. Plafond compagnon de la façade 310 -> 312 (`replay.LectureDuBord`, `replay.PositionsDesVehicules`, justification datée).
- 2026-10-18 E8.4, suite D-18 (exécutant Opus, même worktree). Statut : `[x]`. Aucun artefact publié, aucune base touchée ; décodages un par un (instrument sous `filmproc.AcquireSolo`, passes `replay-parc-vies`).
  - PIÈCES (instrument `replay/equipe_arrivee_bot_research_test.go`, index 8, horloge du film) : `3f048bf3` — « 343 Ritzy » (bid 0, équipe 1 écrite) déclaré de 468,375 s à 481,046 s (premier paquet BOT_METADATA complet sans bot), son entité `ti=9` (slot 1674, équipe 1) lue à UNE image-clé, 468,375 s ; six paquets complets sans bot ensuite ; le corps 540 (record de création, index 8) naît à 523,607 s et finit à 524,107 s (frame 1052) ; « 343 Lacuna » (bid 17, équipe 0 écrite) est déclaré par le paquet SUIVANT, 524,339 s, jusqu'à 525,390 s ; aucune image-clé porteuse entre 508,382 s et 528,386 s, aucune entité `ti=9` de Lacuna. Création du corps à 1:44,7 du chrono, arrivée de Lacuna à l'API (décalage ~44 s) à 1:44,6. `8c19b7dc` — corps 533 né à 1 263,674 s, mort à ~1 265,27 s, « 343 Godfather » (bid 4, équipe 1) déclaré à 1 265,390 s ; aucun paquet entre les deux ; l'index 8 porte aussi « 343 Connmando » (équipe 1, entité lue à 1 633-1 733 s).
  - ERREUR DE LECTURE PROUVÉE (`3f048bf3`) : l'équipe 1 du corps 540 ne vient d'aucune lecture de ce corps ; c'est l'équipe de la TABLE PAR INDEX (`grammar.ScanPlayerTeams`), qui agrège les entités `ti=9` de l'index sur tout le film, donc la seule de Ritzy, parti 42 s plus tôt, transmise par le pont slot -> index (`teamPublication.equipeDuSlot`) à un corps resté sans nom. Deux règles générales : (1) condition c des déclarations (`identity_registry_declarations.go`, `premiereDeclarationApres`) : un corps d'index de bots créé quand aucun bot de l'index n'est déclaré, dont la vie finit avant toute déclaration, est nommé par le PREMIER paquet qui déclare un bot de l'index après sa création, si ce paquet n'en déclare qu'un et si aucune image-clé porteuse ne les sépare (la condition b quand la vie est plus courte que le retard du paquet) ; la piste prend alors l'équipe de l'entrée (Lacuna : 0). (2) Le pont slot -> index se tait sur un index dont un bot déclaré écrit une autre équipe que la table par index (`teamPublication.retenirLesIndexRelayes`), compté dans `coverage.teams.tracksSlotAmbiguous`. Aucun repli : ce sont des lectures. Tests rouge -> vert : `TestPremiereDeclarationApresUneVieCourteNommeLeCorps`, `...NeRetientQueLePremierPaquet` (gardes : image-clé, deux bots au premier paquet), `TestEquipeDuSlotSeTaitSurUnIndexRelaye`, `TestIndexRelayeIgnoreLEquipeNonLue`.
  - VALIDATION : passe ciblée (2 films) puis parc entier (`replay-parc-vies`, binaire de la partie 1, 2 676 films, 12:12 -> 18:34) contre la passe finale du 18/10 (`parc_lot2`) : 2 660 comparés, sans identité 1 944 -> 1 942, NOMMÉES 2 (`3f048bf3` 540 -> « 343 Lacuna », équipe 0 ; `8c19b7dc` 533 -> « 343 Godfather », équipe 1), PERDUES 0, noms changés 0, équipes changées 0 (la règle 2 ne change l'équipe d'aucune autre piste du parc), pistes redécoupées 0 ; 14 écartés et 2 échecs inchangés. Périmètre : 17 -> 15 vies (les 15 du journal du 17/10 : bots jamais déclarés, film sans lien d'identité, pistes parasites). Corpus versionné : goldens d'assemblage inchangés.
  - RÉVISIONS : aucune de plus (contenu du schéma 99, chronique v99 complétée ; la montée de `grammar`/`killsource` vient d E8.1, entrée précédente).
- 2026-10-18 gate E8.1 + E8.4 (sorties réelles, un seul `go` à la fois, `GOCACHE=%LOCALAPPDATA%\go-build-backlog1016`, après la passe du parc) : `go test -count=1 ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/... ./internal/replaybuild/... ./internal/service/... ./internal/archlint/` : 28 paquets `ok`, 0 échec (après régénération des fixtures de bots par `TestBotsDesFixturesDuRejeuRegenerate`, du golden des formes et des fixtures de contrat web : seules les valeurs `grammar-2026-10-18` / `killsource-2026-10-18` changent, vérifié ligne à ligne) ; `go test -tags=integration -p 1 ./internal/sync/killcollector/... ./internal/replaybuild/... ./internal/persist/... ./cmd/levelup/...` : 4 `ok` ; `replay-corpus-gate --base=HEAD` : 19 témoins `ok`, 0 gain, 0 perte, 0 changement (seuls `coverage.decoder.grammarRev` et `killsourceRev` bougent), FIN=0 ; `vitest run src/features/match-replay` (hors sandbox) : 232 fichiers passés, 4 ignorés, 3 244 tests passés ; `golangci-lint run --new-from-rev=HEAD` (film, killcollector, service, archlint, cmd/levelup ; tags par défaut, `integration`, `research`) : 0 issues ×3. Lecture du bord du collecteur contre les faits (`bord_du_collecteur_research_test.go`, `4f77afc1`, un décodage) : occupation 431 = 431 lectures égales, nuage des véhicules 144 385 = 144 385 échantillons égaux — les mesures sur faits valent pour la production. Aucun test renommé ni retiré : baseline JSONL inchangée.
  - À JOUER À E8.6 (local, serveur arrêté, puis prod dans la séquence connue recuisson -> usage-summary -> pad-tiers --force -> killsource, le pilote prévient le user) : RECUISSON des rejeux (artefacts 98 : verdict `redecoder`, les faits persistés portent `grammar-2026-10-16`, le film est relu) ; `levelup backfill-killsource` (hors ligne ; `match_kill_events` sous `killsource-2026-10-16` et `match_lives` sous l'ancienne `IsolationDecoderRev` deviennent candidats : `kill_positions`, `kill_openings`, `match_lives`, `match_death_context`, `match_death_mates` réécrits, append-only, lus par `_latest` ; placement des vies `placement-2026-10-17`).

- 2026-10-18 E8.3 Harvest (exécutant Sonnet, worktree `LevelUp-wt-backlog-1016`). Statut : `[x]`. Aucune requête réseau, aucune authentification ; aucune base du serveur ouverte (le registre a été lu dans la copie à froid `data/backups/avant_recuisson_98_2026-10-16`, par `cmd/diag_q` en lecture seule) ; aucune modification dans le checkout principal.
  - CANEVAS. `TestPreuveLevelIDCartesDuCache` (tag `gamefiles`, `PREUVE_MVAR_CACHE`) avec la ligne `{"Harvest", "ff6b8a12-...", CanevasBlank}` : `level_id 426470249 (0x196B6B69) -> [fo11_blank] | objets 5296 | objectifs 34`, unicité 1/1 sur les 32 level_id distincts des 64 modules (le test échoue si le dossier n'est pas unique ou diffère de l'attendu : la preuve n'est pas posée par hypothèse). Même level_id qu'Argyle (`mapModule`), déjà sur `fo11_blank`. `CartesForge` ne cite pas Harvest (aucun fond produit, hors périmètre) : l'entrée de `mapModule` suffit à la résolution des bornes, comme pour Serenity/Interference/Vacancy.
  - CATALOGUES. `mapquant-build` (installation du jeu présente, exécuté depuis le worktree) : 84 -> 85 cartes, diff = 18 lignes ajoutées, entrée `harvest` (module `fo11_blank`, largeurs 15/15/17), aucune autre carte ne bouge. `film-profiles-build` : seul le bloc `derived` change (84 -> 85 cartes, nouvelle empreinte `991eb9e8...41ef1`), exigé par `TestCatalogueCommisEgaleCatalogueRegenere` ; la révision de profil ne hache pas ce catalogue (recos-a). `mapobj-build --from-file` : copie du `.mvar` nommée `<map_id>_map.mvar` (convention de Serenity/Vacancy, évite un troisième module générique `map`) : 130 -> 131 cartes, 34 objectifs, 28 hachages de libellé non résolus, diff = l'entrée + `generated_at` + la ligne de notes.
  - RELEVÉS. `replay_map_objectives_test.go` : CTF à formes 36 -> 37 (l'assertion par rôle et camp passe sur les 131 cartes) ; `flag_spawn_neutral_label_test.go` inchangé (vert : aucun socle neutre au label ni sans équipe n'est ajouté). `replaybuild_test.go` : `Harvest` -> `fo11_blank` dans `TestResolveMapEntry_SurLeCatalogueLivre`.
  - CUISSON, sans rien publier (`levelup replay-parc-vies`, faits du parc `parc_faits`, cache du poste en `--cache`, catalogues lus dans le worktree, un film à la fois en avant-plan). Avant (passe du parc du 18/10) : `e449a696` et `99b8ba5c` « écarté (carte_hors_catalogue, Harvest) ». Après : `e449a696` (KOTH Squad, HI_1_13_0) `ok`, 158 pistes, 0 sans nom, seuil de garde de la colline lu : 199 points dont 60 écartés, un tic par point (le témoin n'en mesure pas le seuil, cf. ci-dessus) ; `99b8ba5c` (Slayer, HI_1_13_0) `ok`, 144 pistes, 0 sans nom. Ce sont les SEULS films Harvest du cache (les 26 autres matchs du registre n'ont ni faits ni chunks) : la demande de 2-3 autres matchs n'est pas atteignable ; la KOTH est couverte par `e449a696`. Piège noté : poser `LEVELUP_REPO_ROOT` sur le checkout principal fait lire à la commande le catalogue de CE checkout (ancien) et reproduit le refus ; ne pas le poser et passer `--cache`.
  - GATE E8.3 : `go test -count=1 ./internal/himap/... ./cmd/mapquant-build/... ./cmd/mapobj-build/... ./internal/replaybuild/... ./internal/service/... ./internal/archlint/` : tous `ok` (himap 3,5 s, archlint 72 s, service 13 s) ; `golangci-lint run --new-from-rev=HEAD` : 0 issues (tags par défaut, et `gamefiles` sur `internal/himap`). `go-api-test-gamefiles` : `film-profiles-build`, `mapfond-build`, `mapstruct-build` `ok` ; `internal/himap` NON vert pour des causes étrangères au lot (cf. découvertes : dépôt de variantes `.ai/re_dump/mapvar` absent, `TestBancCliffhanger` 64,4 % < 64,7 %), `TestPreuveLevelIDCartesDuCache` (la sonde du lot) `PASS` avec ses 4 cartes.
  - RÉVISIONS : aucune de montée (catalogue de bornes hors `profile`, `PlacementRev`, `SchemaVersion`, `grammar`, `killsource` inchangés). À JOUER APRÈS FUSION (E8.6/E10.5) : `backfill-replay` sur les films Harvest du parc de prod, qui n'avaient aucun rejeu.
- 2026-10-18 clôture d'E8 (exécutant Sonnet, même worktree). Gate d'étape rejoué après E8.3, un seul `go` à la fois : `go test -count=1 ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/... ./internal/replaybuild/...` : 21 paquets `ok`, 0 échec ; `go test -count=1 ./internal/himap/... ./cmd/mapquant-build/... ./cmd/mapobj-build/... ./internal/replaybuild/... ./internal/service/... ./internal/archlint/` : tous `ok` ; `CGO_ENABLED=0 go run ./cmd/replay-corpus-gate --base=HEAD~1` (E8.3 contre l'état d'avant, deux passes de 590 s au plus, reprise par le cache de la base) : 19 témoins `ok`, 0 gain, 0 perte, 0 changement, FIN=0 (les trois témoins les plus récents cuits à froid, dont `f75e7053` KOTH) ; le gate d'E8.1/E8.4 contre `HEAD` (pilote) reste celui qui porte les changements de révision. Items : E8.1 `[x]`, E8.2 `[x]`, E8.3 `[x]`, E8.4 `[x]`, E8.5 `[x]`, E8.6 `[~]` (E10.5). `.ai/BACKLOG.md` : quatre entrées retirées (vies sans identité, écart de position, courbe Lattice, Harvest), deux entrées pour les restes réels (15 vies sans nom ; découvertes d'E8 non traitées), une ligne « Récemment complété » ; `.ai/REGISTRE_REPORTS.md` : lignes Lattice, Harvest et écart de position soldées. Contrôle visuel du user : OUVERT jusqu'après la fusion.
- 2026-10-18 E9 (exécutant Sonnet, même worktree) : E9.1 à E9.4 faits, gate vert (détail sous l'étape). Pièces rouvertes avant de coder : `cmd_backfill.go` (seul `--engagement-scores` portait un titre), `engine_options.go` (`NewSyncEngine` = `DefaultSlug`), `handlers/backfill.go` (moteur sans titre, provider du boot), `handlers/settings.go:518` (titre vide), `config/player_resolver.go` (`cfg.SharedProvider` = titre par défaut), `migration/registry.go` (`RunForDB` force `DefaultSlug`), `Dockerfile` (binaires de l'image), `deploy.yml` (job `deploy-demo`), COMMANDS.md (doctrine des passes film). Écarts au brief : (1) `--title` couvre aussi `--composite-only`, `--citations-recompute-all`, `--compare-formulas` (même code, sinon piège silencieux) ; (2) les passes sélectionnées tournent TOUTES (avant : `--perf --assists-model` ne lançait que `--perf`, les autres retournaient tôt) ; (3) le gate de capability est aux appelants (CLI, job), pas dans `RunBackfillCitations` : les tests de pipeline le pilotent sans manifeste de titre ; (4) le champ `title` n'a pas de schéma OpenAPI (corps lu à la main) : ajouté à la description du fragment et à `types.ts`. Fichiers : voir `git show --stat` du commit `b1016(E9)`.
