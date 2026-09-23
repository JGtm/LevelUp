# Plan : chargements laborieux (Escouade en tete) — campagne perf 2026-09-23

> Source : `.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md` (mesures, causes C1-C9).
> Go utilisateur le 2026-09-23 (« d'accord avec ton plan, pilote des agents Opus »).
> Contrat d'execution : skill `plan-execution` (ordre strict a l'interieur d'un lot, aucun
> item sans statut, zero fix hors perimetre, decouvertes consignees ici, pas differees).
> Branche de campagne : `feat/perf-chargements` (depuis `origin/feat/v75` c89aa4bdc, code
> identique a la base mesuree 43a01721e sur tous les fichiers concernes). Un worktree et une
> branche `feat/perf-<lot>` par lot ; fusion dans la campagne par le superviseur ; fusion
> finale dans `feat/v75` sur go utilisateur apres CI verte.

## 0. Regles communes a tous les lots

- Perimetre FERME par lot (liste des fichiers autorises). Toute autre modification = hors
  perimetre : a consigner en « Decouvertes » du lot, pas a faire.
- Aucun agent ne lance le serveur (`air`, `server.exe`), Vite, un navigateur, un backfill,
  ni n'ouvre les bases reelles sous `data/` en ecriture. Mesure de reference = le
  superviseur, sur le checkout principal, a la fin de chaque vague (protocole §8).
- Chronometrage SQL autorise pour un agent : COPIE de `data/titles/halo_infinite/warehouse/
  shared_matches_v2.duckdb` dans son scratchpad + outil Go temporaire hors de l'arbre
  surveille par air (ou sous `cmd/<lot>_probe_tmp/` supprime avant le commit), ouverture
  `?access_mode=read_only`, `SET threads=2`, `SET memory_limit='512MB'`.
- Commandes `go` : une a la fois dans son worktree, avec
  `PATH=C:\msys64\ucrt64\bin;... CGO_ENABLED=1 CC=C:\msys64\ucrt64\bin\gcc.exe
  GOCACHE=<worktree>\.gocache-<lot>` ; lint avec `GOLANGCI_LINT_CACHE` isole par lot.
- Invariants intouchables : ecritures per-match via `persist.BatchBuilder` (INSERT-only),
  lectures des tables append-only par les vues `_latest`, `PathResolver` pour tout chemin,
  capabilities (jamais `slug == ...`), logs `slog.*Context`, aucune string UI FR en dur,
  fichiers ≤ 500 L / fonctions ≤ 80 L (dette existante gelee par la baseline lint).
- Cloture d'un lot = gates verts + items statues + section du lot mise a jour ici + entree
  `.ai/thought_log.md` + rapport final au superviseur (fait / non fait + justification /
  decouvertes / mutations jouees). Commits sur la branche du lot, message prefixe `perf(<lot>)`.
  Pas de push par les agents.

## 1. Vagues

| Vague | Lots (en parallele) | Condition de depart |
|---|---|---|
| 1 | L1 instrumentation, L3 timeouts/retry/annulation/session, L4a front Escouade, L6 sync | go du 2026-09-23 |
| 2 | L2 Escouade backend, L5a blocs solo, L5b socle | L1 fusionne dans la campagne |
| 3 | L4b endpoint leger + ancrage avant la requete lourde | L2 et L4a fusionnes |
| Cloture | mesure de reference, revue adversariale du diff cumule (fan-out aveugle), correctifs, push + CI, fusion `feat/v75` | tous les lots fusionnes |

## 2. L1 — Instrumentation des durees (Go)

Objectif : voir ou passe le temps sans lancer un profileur ; base des verdicts des lots 2, 5.

Decisions tranchees :
- D1.1 Middleware `internal/api/middleware/slog_logger.go` : seuil `LEVELUP_SLOW_REQUEST_MS`
  (defaut 1000, lu UNE fois au montage). Toute requete dont la duree ≥ seuil est journalisee
  en INFO (meme `msg: "http"`, attributs inchanges + `slow: true`), y compris les 2xx. En
  dessous : comportement actuel (DEBUG / WARN / ERROR).
- D1.2 Paquet `internal/observability/timing` : `type Timings` (mutex, sections ordonnees,
  cumul + nombre d'appels par nom) ; `WithTimings(ctx) (context.Context, *Timings)` ;
  `FromContext(ctx) *Timings` nil-safe (un `*Timings` nil est un no-op) ;
  `(*Timings) Section(name string) func()` (la fonction rendue arrete le chrono) ;
  `(*Timings) Snapshot() []SectionStat` (tri duree decroissante) ; `(*Timings) LogAttrs()
  []any` (`total_ms`, `sections` = "nom=ms" x15 max, `calls` pour les sections appelees
  plus d'une fois). Tests unitaires purs (sans DuckDB).
- D1.3 Le middleware pose `WithTimings` sur CHAQUE requete et, apres le handler, si au moins
  une section existe ET (duree ≥ seuil OU DEBUG active), journalise une ligne INFO
  `http_timings` avec `path`, `duration_ms`, `LogAttrs()`. Les services appellent seulement
  `defer timing.FromContext(ctx).Section("nom")()`.
- D1.4 Sections a poser (liste fermee, noms = fonction appelee) :
  - `service/teammates/teammates_service.go` GetPage : `top_teammates`, `player_matches`,
    `teammate_rows`, `main_team_allies`, `enrich_assets`, `map_stats`, `match_history`,
    `session_timeline`, `map_heatmap`, `impact_matrix`, `per_minute`, `synergy_radar`,
    `intensity_profile`, `performance_series`, `weapon_kills`, `weapon_accuracy`,
    `kill_mechanics`, `first_blood`, `assist_pairs`, `echange`, `range_profiles`,
    `medal_digest`, `briefing_header`, `composition_sessions`, `equipment_usage`, `squad_formes` ;
  - `service/filters_service.go` Resolve : `load_matches`, `resolve_rows`, `season_counts` ;
  - `service/synthesis_service*.go` : une section par etape majeure (chargement canonique,
    enrichissement FR, distribution des frags, records de distance / portee, profil de
    combat, armes, vehicules, objectifs, heatmap) ;
  - `service/session_page_service*.go` et `service/timeseries_service*.go` : une section par
    bloc / section (coordination, portee, evenements, objectifs, LUSR, engagement, ...) ;
  - `service/coordination_block.go` : `kill_events`, `appuis`, `bloc`.
- D1.5 Pas de compteur SQL dans ce lot (hors perimetre). Aucun changement de comportement.

Perimetre (fichiers) : `internal/observability/timing/*` (nouveau), `internal/api/middleware/
slog_logger.go` (+ test), les fichiers de service listes en D1.4 (ajout de `defer` uniquement),
`docs/COMMANDS.md` ou `docs/SYNC_GUIDE.md` : une ligne sur `LEVELUP_SLOW_REQUEST_MS` (EN + FR).

Items :
- [x] L1.1 paquet `timing` + tests — `internal/observability/timing/timing.go` (Timings, WithTimings, FromContext nil-safe, Section, Snapshot, LogAttrs) + `timing_test.go` (11 tests purs sur horloge pilotee, `-race` vert ; mutation « garde nil retiree » = panique detectee)
- [x] L1.2 middleware : seuil + `http_timings` + test (seuil respecte, 2xx lent en INFO, 2xx rapide en DEBUG) — `internal/api/middleware/slog_logger.go` (seuil lu au montage, `slow: true`, 4xx/5xx lents gardent WARN/ERROR, valeur invalide = ERROR + defaut 1000) + 10 tests nouveaux dans `slog_logger_test.go` ; mutations jouees : seuil ramene a 0 (6 tests rouges), porte `http_timings` toujours ouverte (2 rouges), condition « au moins une section » retiree (1 rouge)
- [x] L1.3 sections GetPage (26) — 20 `defer` en tete des builders appeles (16 fichiers `service/teammates/teammates_*.go`, consigne : ne pas allonger GetPage) + 6 sections inline dans GetPage (`top_teammates`, `player_matches`, `map_stats` = appels de repo ; `match_history`, `session_timeline`, `composition_sessions` = fonctions pures sans ctx), GetPage +12 L ; sonde temporaire (supprimee) : 26 sections remontees, `teammate_rows` calls=2 ; lint : 0 issue nouvelle, 3 funlen preexistants a +1 (buildTeammateRowWithMatches 89 L, buildSquadImpactMatrix 84 instructions, buildSquadPerMinuteStats 97 L)
- [x] L1.4 sections filtres, synthese, sessions, series temporelles, coordination — 38 sites de section (table au journal du lot) : filtres 3 (inline dans Resolve), synthese 9, sessions 10 et series temporelles 14 (dont `expected_assists`, un seul site partage par les deux pages), coordination 3 (`kill_events`, `appuis`, `bloc`) ; `defer` dans les builders appeles quand ils ont un ctx, sinon section inline ; sonde temporaire (supprimee) : 9 / 10 / 12 / 2 / 3 sections remontees sur les fixtures existantes (les sections conditionnelles absentes quand leur repo n est pas cable) ; lint : 0 issue nouvelle, funlen preexistants GetSynthesisPage 107->111, GetPage sessions 138->140, GetPage series 136->144
- [x] L1.5 doc de la variable (EN + FR) — `docs/COMMANDS.md` et `docs/FR/COMMANDS.md`, section « Run the app / Lancement » : seuil, defaut 1000, lu au demarrage, `slow: true`, ligne `http_timings` (champs), DEBUG via `LEVELUP_LOGS_FILE_LEVEL=debug`

Gate : `gofmt -l` vide ; `go build ./...` ; `go vet ./...` ; `go test ./internal/observability/...
./internal/api/middleware/... ./internal/service/... ./internal/service/teammates/...` ;
`golangci-lint run` sur les paquets touches : 0 issue nouvelle.

Journal du lot (2026-09-23, branche `feat/perf-l1` depuis 97cc0d0c8 ; commits c5c50f499 L1.1,
108ac8661 L1.2, 0a2ce5c10 L1.3, 4ee02dbcd L1.4, puis L1.5 + ce journal) :

- Lecture retenue de D1.1 : une requete lente 4xx/5xx garde son niveau WARN/ERROR (jamais
  retrogradee en INFO) et porte `slow: true` ; seul un 2xx/3xx lent passe de DEBUG a INFO. Valeur
  invalide ou <= 0 : ERROR journalisee au montage, seuil par defaut 1000.
- `total_ms` = somme des cumuls de TOUTES les sections (y compris au-dela des 15 listees). Les
  sections sont des feuilles (aucune fonction instrumentee n'en appelle une autre, appelants
  recenses) : `duration_ms - total_ms` = temps hors sections (factory de
  service et resolution du joueur dans le handler, calculs purs non listes, encodage, middlewares
  en aval dont l'ecriture de session).
- Placement : `defer` en premiere instruction du builder appele quand il recoit un ctx (consigne :
  ne pas allonger les orchestrateurs deja au-dela du seuil funlen) ; sinon section inline par la
  fonction d'arret (`stop := ...Section(...)` puis `stop()`), la ou un defer ne se declencherait
  qu'en sortie de fonction : appels de repo, fonctions pures sans ctx, blocs d'orchestrateur.
- Fichiers de builders touches au-dela de la liste litterale de D1.4 (defer + import uniquement,
  en application de la consigne « poser les defer dans les builders appeles ») : 16 fichiers
  `service/teammates/teammates_*.go` ; `service/{expected_assists,session_page_frag_distribution,
  session_page_usage,session_page_range,synthesis_weapon_records}.go`.
- Table des sections hors Escouade (nom = fonction chronometree) : filtres `load_matches`
  (LoadMatchesForFilters), `resolve_rows` (ResolveFiltersFromRows), `season_counts` (catalogue +
  BuildSeasonCounts, seulement si le catalogue est cable) ; synthese `player_matches`,
  `enrich_translations` (enrichissement FR), `heatmap`, `combat_profile`, `fun_stats` (awards et
  vehicules), `frag_distribution` (loadTopWeaponKills), `weapon_accuracy` (armes),
  `objective_stats`, `weapon_records` (records de distance) ; sessions `player_matches`,
  `placements` (LUSR / CSR), `objective_index`, `expected_assists`, `frag_distribution`,
  `weapon_accuracy`, `event_blocks`, `lobby_sizes`, `session_usage`, `range_profiles` (portee) ;
  series temporelles `player_matches`, `highlight_events`, `expected_assists`, `tabs` (onglets),
  `objective_stats`, `weapon_kills`, `frag_distribution`, `weapon_accuracy`, `event_blocks`,
  `weapon_range` (portee des engagements), `equipment_usage`, `squad_formes`, `team_sizes`,
  `range_profiles` ; bloc coordination (sessions et series) `kill_events`, `appuis`, `bloc`, sans
  section englobante (elle serait comptee deux fois dans `total_ms`). En mode comparaison, les
  blocs de la session comparee cumulent sous le meme nom (`calls` = 2) ; le journal des morts de
  la reference d'habituel aussi (`kill_events` jusqu'a 3 appels).
- Dette : 0 issue lint nouvelle (51 = 51 sur les trois paquets du gate ; `--new-from-rev=97cc0d0c8` :
  0 issue). Six compteurs funlen preexistants augmentent : buildTeammateRowWithMatches 88->89,
  buildSquadImpactMatrix 83->84 instructions, buildSquadPerMinuteStats 96->97, GetSynthesisPage
  107->111, GetPage sessions 138->140, GetPage series 136->144 ; GetPage Escouade (nolint funlen)
  +12 L. Quatre fichiers deja au-dela de 500 L grossissent : filters_service.go 606->613,
  session_page_service.go 887->894, teammates_service.go 566->579,
  teammates_squad_charts_weapons_perf.go 674->678 ; aucun fichier ne franchit 500 L.
- Gate : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ; `go test` des
  quatre motifs du gate 0 (aucun `--- FAIL:`) ; golangci-lint 0 issue nouvelle. En plus, verts :
  `go test ./internal/archlint/... ./internal/config/... ./internal/api/...`, `-race` sur `timing`.
- Mutations jouees (toutes rouges puis restaurees, `cmp` a l'appui) : garde nil de `Section`
  retiree (panique) ; seuil ramene a 0 (6 tests rouges, dont « seuil non atteint = pas de ligne
  INFO ») ; porte `http_timings` toujours ouverte (2 rouges) ; condition « au moins une section »
  retiree (1 rouge).
- Exemple de ligne produite par un test : `{"level":"INFO","msg":"http_timings",
  "path":"/api/v1/players/x/pages/teammates","duration_ms":6,"total_ms":6,
  "sections":"load=6 build=0","calls":"load=2"}`.
- Decouvertes du lot (consignees ici, non traitees) : (1) `title_slug` de la ligne `http` et les
  compteurs HTTP par titre valent toujours le repli `halo_infinite` pour la chaine racine : le
  middleware est monte avant `TitleExtractor` (`api/server.go:640` puis `:645`), qui ne passe le
  titre qu'a une requete derivee (`middleware/title.go:40-42`) — lecture de code, non mesure ;
  (2) commentaire de doc orphelin en fin de `service/teammates/teammates_service.go` (doc de
  buildBriefingHeaderForTeammatesPage, dont la fonction, `teammates_service_briefing.go`, n'en a
  pas) ; (3) directives `//nolint:PLR0913 — ...` invalides (`service/match_view_builders_team.go:
  47-48`) : avertissement « unknown linters » a chaque golangci-lint ; (4)
  `internal/observability/README.md` ne cite pas le sous-paquet `timing` (hors perimetre) ; (5)
  processus : le scratchpad de session est partage entre agents de lot — mon `lint_baseline.txt`
  a ete ecrase a 11:58 par un fichier du lot L4a (meme nom), reference retrouvee dans ma copie
  normalisee de 11:54 ; un sous-dossier par lot evite la collision ; (6) `go test ./internal/api/...`
  a cree `data/titles/halo_5/warehouse/metadata.duckdb` (12 Ko, ignore par git, 12:21) dans l'arbre
  du worktree : un test existant ecrit sous `data/` du depot au lieu d'un repertoire temporaire ;
  artefact retire apres coup, test non identifie (hors perimetre).

## 3. L3 — Timeouts, retry, annulation, session (Go + web)

Decisions tranchees :
- D3.1 `cmd/server/main.go` : `WriteTimeout` 30 s → 120 s (constante nommee, commentaire :
  date, cause = pages a 25-55 s tronquees le 2026-09-23, nginx `proxy_read_timeout` 300 s
  donc compatible). `ReadTimeout`, `IdleTimeout` inchanges.
- D3.2 Handlers (`internal/api/handlers/*.go`) : un helper `mapServiceError(ctx, err, code)`
  dans `helpers.go` qui rend (a) `errDBBusy()` (503 + Retry-After, existe deja) pour
  `dblease.ErrDBLocked` et `sharedprovider.ErrSwapTimeout` / `ErrSwapFailed`, (b) une erreur
  Huma 499 `client_closed` journalisee en DEBUG (jamais ERROR) pour `context.Canceled`,
  (c) sinon le 500 actuel avec le log ERROR actuel. Applique aux handlers de pages :
  teammates, filters (resolve + match-ids), synthesis, sessions, timeseries, home (garder son
  code `home_page_db_busy`), explorer, career. Contrats OpenAPI (`api/openapi.yaml`, gate
  `openapi-gen -check`) et tests de contrat mis a jour.
- D3.3 Web `lib/api/client.ts` : `get/post(..., { signal })` transmis a `fetch`.
  `app/queryClient.ts` : pas de retry quand `status` ∈ {502, 504} (reponse tronquee ou
  proxy) ; 2 retries conserves sur 500 et 503 (db_busy est transitoire).
- D3.4 Passer `signal` (du contexte `queryFn`) aux hooks de page : `features/filters/queries.ts`
  (resolve + preview), `features/synthesis`, `features/session-detail`, `features/timeseries`,
  `features/career`, `features/home` (les hooks `useQuery` de page, pas les mutations).
  `features/squad/queries.ts` est HORS perimetre (L4a).
- D3.5 `internal/api/middleware/session.go` + `platform/session/store.go` : `Touch` n'ecrit le
  fichier que si la session est modifiee OU si sa derniere ecriture date de plus de 5 min
  (champ `LastPersistedAt` en memoire, pas dans le JSON ; TTL glissant preserve a 5 min pres).
  Test : deux requetes a 1 s d'intervalle = une ecriture.

Perimetre : `cmd/server/main.go`, `internal/api/handlers/helpers.go` + handlers de pages listes
+ leurs tests + `api/openapi.yaml` (regenere), `internal/api/middleware/session.go`,
`internal/platform/session/store.go` (+ tests), `apps/web/src/lib/api/client.ts` (+ test),
`apps/web/src/app/queryClient.ts` (+ test), hooks `queries.ts` des features listees en D3.4.

Items :
- [ ] L3.1 WriteTimeout 120 s + commentaire date
- [ ] L3.2 helper `mapServiceError` + application aux handlers de pages + OpenAPI + tests
- [ ] L3.3 client.ts `signal` + queryClient sans retry 502/504 + tests
- [ ] L3.4 `signal` transmis dans les hooks de page listes
- [ ] L3.5 Touch de session throttle + test

Gate Go : `gofmt`, `go build ./...`, `go vet ./...`, `go test ./internal/api/... ./internal/platform/
session/...`, `go run ./cmd/openapi-gen -check` (ou la commande documentee dans `docs/COMMANDS.md`),
lint paquets touches. Gate web (depuis `apps/web`, `node_modules\.tmp` purge avant) :
`npm run typecheck`, `npm run lint`, `npx vitest run src/lib/api src/app src/features/filters
src/features/synthesis src/features/session-detail src/features/timeseries src/features/career
src/features/home`.

## 4. L4a — Front Escouade : une seule source de verite, pas de requete a vide (web)

Decisions tranchees :
- D4.1 La session pickee de l'escouade vit UNIQUEMENT dans `useSquadFilterStore`
  (`filterContext.sessions.picked_sessions`). `pickedSquadSessionLabels` devient une valeur
  derivee du store ; l'etat local et la cle localStorage `squad-sessions-<slug>` sont
  supprimes (migration une fois au montage : si la cle existe et le store est vide, l'appliquer
  au store puis retirer la cle). `applySessionLabels` = `setSessions` du store. La cle de
  query teammates n'inclut plus `sessionLabels` (le hash du filterContext les couvre) ; le
  corps envoie `picked_squad_session_labels` = `picked_sessions` du store (contrat serveur
  inchange). Consequence attendue et verifiee par test : un snap ou un clic du rail = UNE
  seule nouvelle cle de query, donc une seule requete.
- D4.2 `useTeammates` gagne `enabled` : pas de requete tant que la composition initiale n'est
  pas connue = `selectedGts.length > 0` OU la liste d'amis est resolue (`isSuccess` de
  `useFriendGamertags`, a exposer) et vide (mode exploration sans coequipier : requete
  legitime). Le deep-link (`session`/`teammates` en URL) pose la composition avant la premiere
  requete.
- D4.3 Aperçu et resolve escouade : `useFiltersPreview` n'est actif que si `isDirty`
  (filtres en attente differents du commite) ; sinon `available`, `presetCounts`,
  `seasonCounts` viennent du `resolvedContext` (repli deja en place). Pour que ce repli soit
  equivalent, le resolve du store escouade envoie `match_context: 'squad'` :
  `useFiltersResolve(playerSlug, store, { matchContext })`, et `matchContext` entre dans la
  cle (`queryKeys.filtersResolve`) pour ne jamais confondre solo et escouade.
- D4.4 Resolve solo inutile hors pages Stats : predicat pur `routeShowsSoloFilters(pathname)`
  exporte de `components/shell/shellNavigation.ts` (meme regle que `NavL2` pour rendre
  `FilterOmnibar`), utilise par `NavL2` ET par `PlayerLayout` pour `enabled` du resolve solo
  et de `useFollowLatestSession`. Test : sur `/squad/*`, `/career`, `/home`, aucun resolve solo.
- D4.5 `AssetDrawer` : les trois requetes d'assets ne partent que tiroir ouvert
  (`enabled: isOpen`), sans changer le comportement une fois ouvert.
- D4.6 Rien d'autre : pas de refonte de `decideCompositionReanchor` (L4b), pas de nouvel
  endpoint, pas de changement de contrat serveur, StrictMode conserve.

Perimetre : `apps/web/src/features/squad/{SquadLayout.tsx,SquadFilterBar.tsx,
useSquadFilterBarState.ts,squadPending.ts,queries.ts,SquadContext.ts}` + tests et gardes
associes (`filterBarBoundary.guard.test.ts`, `squadPending.test.ts`, ...), `features/friends/
queries.ts` (exposer `isSuccess`), `features/filters/queries.ts` (option `matchContext` et
`enabled` ; coordonner avec L3 qui y ajoute `signal` : ne toucher que la signature et la cle),
`lib/query/keys.ts`, `components/shell/{NavL2.tsx,shellNavigation.ts}` (+ tests),
`routes/{-$lang}/t/$titleSlug/players/$playerSlug.tsx`, `features/asset-drawer/*`, `stores/
createFilterStore.ts` seulement si D4.1 l'exige (documenter).

Items :
- [ ] L4a.1 source unique de la session (D4.1) + migration localStorage + tests (un snap = une cle)
- [ ] L4a.2 `enabled` de teammates (D4.2) + deep-link + tests
- [ ] L4a.3 aperçu conditionnel + `matchContext` dans resolve et cle (D4.3) + tests
- [ ] L4a.4 predicat `routeShowsSoloFilters` partage NavL2 / PlayerLayout (D4.4) + tests
- [ ] L4a.5 AssetDrawer `enabled: isOpen` (D4.5) + test
- [ ] L4a.6 i18n : aucune string nouvelle ; sinon FR + EN

Gate (depuis `apps/web`, `node_modules\.tmp` purge avant) : `npm run typecheck` (tsc -b) ;
`npm run lint` ; `npx vitest run src/features/squad src/features/filters src/features/friends
src/features/asset-drawer src/stores src/components/shell src/routes src/lib/query`.

## 5. L6 — Synchronisation : zero ecrivain quand rien n'est nouveau (Go)

Decisions tranchees :
- D6.1 `internal/sync/skill/skill_v2_shadow.go` (et `skill_v2_shared_access.go`) : le
  filigrane « deja traite » se lit AVANT toute prise d'ecrivain, sur un lecteur (`SharedReadDB`
  ou snapshot) ; en regime stationnaire (aucun candidat nouveau) : zero acquisition d'ecrivain,
  une ligne INFO `lusr_v2: rien de nouveau` avec `candidates`, `new` = 0.
- D6.2 S'il y a du nouveau : UNE rafale d'ecrivain par joueur et par cycle (les candidats
  nouveaux, par lots de 3 a l'interieur si la transaction l'exige), jamais un ecrivain par
  triplet de candidats.
- D6.3 Invariants ART intacts : INSERT-only, `written_at`, lectures par `_latest`, `persist.*`
  ; aucun UPSERT ; garde-rails `no_art_patterns_test.go` inchanges (aucune allowlist ajoutee).
- D6.4 Hors perimetre : le 503 sur `ErrSwapTimeout` (L3), le cron du classement mondial, les
  derivations de rejeu, la periode de l'auto-sync (`app_settings.json`).

Perimetre : `internal/sync/skill/*` (+ tests), `internal/sync/no_art_patterns_test.go` si un
nouveau fichier doit etre reference (sans allowlist), `docs/SYNC_GUIDE.md` (EN + FR) : un
paragraphe sur le filigrane.

Items :
- [ ] L6.1 filigrane lu avant l'ecrivain, zero writer en stationnaire + test
- [ ] L6.2 une rafale par joueur + test
- [ ] L6.3 ligne INFO par cycle + doc EN/FR

Gate : `gofmt` ; `go build ./...` ; `go vet ./...` ; `go test ./internal/sync/...` ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` (OBLIGATOIRE, code
de sortie 0 verifie) ; lint paquets touches.

## 6. L2 — Escouade backend (Go) — vague 2, apres L1

Decisions tranchees :
- D2.1 Retirer `LEFT JOIN v_gamertag_lookup` de Q29, Q32, Q32b et de tout autre template de
  `queries_squad.go` / `squad_repo*.go` qui le porte (grep). Les gamertags viennent d'un
  annuaire charge UNE fois par `GetPage` : `SELECT xuid, gamertag FROM xuid_aliases WHERE xuid
  IN (...)` sur les xuids distincts rencontres (top 50, lobby des matchs du perimetre), repli
  `match_participants.gamertag` (MAX) sur les memes matchs pour les xuids absents, puis
  `Joueur ####` (meme regle que `analysis.MaskedXuidLabelSQL`) et bots `bid(...)` (meme regle
  que `analysis.BotSQLCase`). Une fonction pure `annuaire.Resolve(xuid) string` testee.
- D2.2 `LoadImpactEvents` : UNE lecture par `GetPage` (memo dans une structure de chargements
  de la requete), partagee par matrice d'impact, intensite, series de performance, premier frag.
- D2.3 `LoadFor` : un seul chargement par gamertag et par `GetPage` (memo), bandeau compris
  (le parallelisme du bandeau devient inutile : le charger d'abord, une fois). L'intensite lit
  les xuids depuis `teammates[].XUID`, plus jamais par `LoadFor`.
- D2.4 Echange et isolement : `TacticalQuery.Matchs = RestreindreAux(ids de
  allSquadRowsForTimeline)` (historique de la composition, pas du main) et une seule lecture
  du journal des morts partagee par les deux blocs (l'univers n'est calcule qu'une fois).
- D2.5 `sessionMatchIDs` construit aussi depuis `req.Filters.Sessions.PickedSessions` (labels
  ou ids, meme regle que `filterSynthesisByPickedSessions`).
- D2.6 Usage et formes : les trois lectures communes (`match_usage_films_latest`,
  `match_usage_players_latest`, `match_participants`) chargees une fois pour les deux blocs ;
  `replaylabels.Load` charge une fois par process (`sync.Once` ou cache par titre), jamais
  sur le chemin de requete.
- D2.7 Annulation : `ctx.Err()` verifie entre les sections de `GetPage` (retour immediat).
- D2.8 Cible indicative, verifiee par le superviseur (§8) : requete a chaud (7 matchs) ≤ 5 s,
  requete a vide ≤ 1 s. Aucune section supprimee du contrat, aucun chiffre change : un test de
  parite (golden ou comparaison avant/apres sur une base de test) par section touchee.

Perimetre : `internal/service/teammates/*`, `internal/service/squadagg/*`, `internal/platform/
duckdb/{queries_squad.go,squad_repo*.go,squad_v2_adapter.go}`, `internal/games/halo_infinite/
replaylabels/*` (cache), `internal/analysis/*` seulement pour une fonction pure d'annuaire,
tests associes. Pas de changement de contrat `domain.TeammatesPageResponse`.

Items :
- [ ] L2.1 annuaire + retrait des jointures (D2.1) + tests de parite des gamertags
- [ ] L2.2 `LoadImpactEvents` unique (D2.2)
- [ ] L2.3 `LoadFor` unique par membre + xuids sans LoadFor (D2.3)
- [ ] L2.4 journal des morts restreint et partage (D2.4)
- [ ] L2.5 `sessionMatchIDs` depuis `filters.sessions` (D2.5)
- [ ] L2.6 usage / formes partages + `replaylabels` hors chemin de requete (D2.6)
- [ ] L2.7 annulation entre sections (D2.7)

Gate : `gofmt` ; `go build ./...` ; `go vet ./...` ; `go test ./internal/service/...
./internal/platform/duckdb/... ./internal/analysis/...` ; `go test -tags=integration -p 1
./internal/platform/duckdb/...` si un repo est touche ; lint paquets touches ; chrono SQL sur
copie (§0) pour Q29 / Q32 / Q32b avant et apres (attendu : secondes → millisecondes).

## 7. L5a — Blocs des pages solo (Go) — vague 2, apres L1

Decisions tranchees :
- D5a.1 Bloc coordination (`service/coordination_block.go` → `TacticalRepository.KillEvents`,
  `tactical_repo_univers.go`) : quand `Matchs` est fourni, l'univers est restreint a la liste
  AVANT le `EXISTS` sur `match_kill_events_latest` (semi-jointure sur la liste, pas un balayage
  de tout `match_registry ⨝ match_participants` du joueur). Meme regle pour `MortsAvecContexte`.
- D5a.2 Bloc portee / records de distance (`WeaponRangeRepo`, `synthesis_weapon_records.go`,
  `match_range_repo.go`) : lectures restreintes au perimetre de la page (liste de match_ids),
  jamais tout l'historique quand une session ou une periode est posee ; une seule lecture
  partagee quand plusieurs sections en ont besoin dans la meme requete.
- D5a.3 Evenements (`highlight_events`, journal des morts) : charges une fois par requete et
  sur le perimetre ; les 5 s sans journal de Series temporelles (10:41:49 → 10:41:54) doivent
  etre attribues par les sections de L1 puis traites si c'est une lecture repetee.
- D5a.4 Cible indicative (§8) : Synthese ≤ 2 s, Sessions ≤ 2 s, Series temporelles ≤ 4 s,
  chiffres affiches inchanges (tests de parite par bloc touche).

Perimetre : `internal/service/{coordination_block.go,synthesis_service*.go,synthesis_weapon_
records.go,session_page_service*.go,timeseries_service*.go}`, `internal/platform/duckdb/
{tactical_repo*.go,weapon_range_repo.go,match_range_repo.go,highlight_events_repo.go,
kill_distance_repo.go}` + tests. Pas de changement de contrat des reponses.

Items :
- [ ] L5a.1 univers restreint a la liste (D5a.1) + chrono avant/apres sur copie
- [ ] L5a.2 portee / records sur le perimetre (D5a.2) + tests de parite
- [ ] L5a.3 evenements une fois par requete (D5a.3)

Gate : comme L2 (paquets touches) ; `-tags=integration -p 1 ./internal/platform/duckdb/...`.

## 8. L5b — Socle (Go) — vague 2, apres L1

Decisions tranchees :
- D5b.1 `service/seasons_catalog.go` : l'echec du fetch live est memorise par titre (backoff
  30 min) et le catalogue resolu est cache en memoire par titre (TTL 1 h, invalide par un
  fetch reussi) : plus aucun appel reseau sur le chemin de requete apres le premier echec.
- D5b.2 `handlers/field_mappings.go` : cache du DTO construit par (titre, locale) avec une cle
  de version (hash des TOML du titre + version du catalogue de saisons) ; l'ETag se compare
  AVANT toute construction.
- D5b.3 `filters/resolve` : cache serveur des lignes `LoadMatchesForFilters` par (xuid, titre)
  TTL 60 s + invalidation a la fin d'un cycle de sync du joueur (le point d'accroche existe :
  `sync.v2: cycle terminé` / `RunPostSync`) ; les trois contextes (solo, squad, apercu) sont
  calcules en Go depuis les memes lignes.
- D5b.4 `LoadPlayerMatches` : cabler `CachedPlayerMatchesRepo` (existe, jamais instancie)
  dans `wire/registry_pages.go` (`playerMatchesAdapterFor`) par (xuid, titre), APRES
  l'enrichissement FR (les lignes cachees sont partagees : aucun consommateur ne doit les
  muter — verifier par grep et, au besoin, copier a la lecture), invalidation a la fin du sync
  du joueur.
- D5b.5 `config/player_resolver.go` : `db_profiles.json` lu au plus une fois par requete
  (cache par mtime), au lieu de deux lectures par appel (~80 par page Escouade).
- D5b.6 Carriere `highlight-matches` : `enrichHighlightMatches` charge les matchs par
  identifiants (une requete `IN`), plus jamais `LoadAll` deux fois.
- D5b.7 `bootstrap` : l'echec de l'appel privacy est memorise 5 min (aujourd'hui seul le succes
  l'est 30 min).

Perimetre : `internal/service/seasons_catalog.go`, `internal/api/handlers/field_mappings.go`,
`internal/service/filters_service.go` + `internal/platform/duckdb/filters_repo.go` (cache),
`internal/api/wire/registry_pages.go` (+ `registry_pages_home.go` si necessaire), `internal/
platform/duckdb/player_matches_cache.go` (+ invalidation), `internal/config/player_resolver.go`,
`internal/service/career_service_highlights.go` + `handlers/career.go`, `internal/service/
bootstrap_service.go`, `internal/platform/halo/privacy_provider.go`, `internal/sync/*` seulement
pour le point d'invalidation (une ligne d'appel), tests associes.

Items :
- [x] L5b.1 saisons : echec memorise + cache (D5b.1) + test — `service/seasons_catalog.go` : catalogue resolu cache par titre (`seasonsCatalogTTL` 1 h, :53), singleflight par titre (:202), copie rendue a chaque lecture (`cloneSeasonCatalog`, :308) ; l'echec live garde le repli TOML + base `seasonsLiveFetchBackoff` 30 min (:57, `recordFetchFailure` :295) : pendant l'attente, cache hit, ni base ni reseau ; au terme, la requete suivante retente et un succes remplace le repli ; echec de lecture de la base jamais cache ; echec SANS jeton dans le contexte (le provider refuse avant tout appel reseau) non memorise. Marqueurs timing `seasons_catalog_hit/miss`. Tests `seasons_catalog_cache_test.go` (7, horloge pilotee)
- [x] L5b.2 field-mappings : cache + ETag avant construction (D5b.2) + test — `api/handlers/field_mappings.go` `handleGet` (:181) : version = empreinte du CONTENU des trois TOML du titre (memorisee par jeu de sets charges, `field_mappings_cache.go` :62-110) + empreinte du catalogue de saisons (:117) ; DTO cache par (titre, locale fr/en) a cette version, If-None-Match compare avant toute construction (:210, :227) ; ETag = hash du corps (stable d'un process a l'autre, change des qu'un TOML ou une saison change). Corrige au passage le defaut de l'ancien `etagFor` (ETag fige au 1er corps par (titre, locale, schema) : une saison decouverte gardait l'ancien ETag). Locales hors fr/en servies sans cache (cache borne). Marqueurs `field_mappings_cache_hit/miss`. Tests `field_mappings_cache_test.go` (5)
- [x] L5b.3 cache filtres + invalidation post-sync (D5b.3) + test — `platform/duckdb/player_read_cache.go` (nouveau) : cache process-wide generique, cle (xuid, titre, chemin de la base, variante), TTL 60 s (:49), FIFO 256, singleflight, generation par (xuid, titre) (un chargement commence avant une invalidation ne remplit pas le cache, et la requete suivante ne se greffe pas sur lui : cle de vol + generation, :153), copie a la lecture, requete vivante qui recharge si le vol partage a ete annule avec la requete qui le portait ; `CachedFiltersRepo` (`filters_repo.go` :34-52) cable dans `wire/registry_pages.go:40`. Les trois contextes (solo, escouade, apercu) partagent une entree : `LoadMatchesForFilters` n'a aucun parametre, le contexte est applique en Go. Invalidation `InvalidatePlayerReadCaches` (:64) : `sync/engine_postsync.go:170` (defer en tete de `runPostSyncPipeline` = fin du post-sync du joueur, V1 et V2, y compris panic recupere), `sync/friends_recompute.go:62` (recalcul is_with_friends hors sync), `platform/duckdb/match_exclusion_repo.go:63` (exclusion). Ratchet `archlint/player_read_cache_invalidation_test.go`. Marqueurs `filter_rows_cache_hit/miss`. Tests `player_read_cache_test.go` (10) + `wire/registry_pages_cache_test.go`
- [x] L5b.4 `CachedPlayerMatchesRepo` cable + invalidation (D5b.4) + test — `platform/duckdb/player_matches_cache.go` refondu sur le cache generique : enveloppe `PlayerMatchesAdapter` donc cache les lignes APRES l'enrichissement FR (:49), une variante par jeu de filtres (`filtersCacheKey`), delegue `LobbySizesAtCompletion` (capacite lue par assertion de type dans SessionPageService, :71), `InvalidatePlayer` du port = invalidation du joueur ; cable dans `wire/registry_pages.go:325` (`playerMatchesAdapterFor`). Grep des consommateurs : Home et Synthese RE-ENRICHISSENT les lignes recues (`EnrichCanonicalAssetTranslations` ecrit `Labels[...]`, `DefaultLabel`, `IconURL` a travers les pointeurs d'AssetReference) — une map partagee entre requetes serait un « concurrent map writes » fatal : `clonePlayerMatchRows` (:91) copie la tranche et les quatre AssetReference ; autres references partagees (aucune ecriture, grep du 2026-09-23), inventaire fige par `TestClonePlayerMatchRows_ReferenceInventory`. Ancien `ttlCache` et `InvalidatePlayer` no-op de l'adapter (debranche) supprimes. Marqueurs `player_matches_cache_hit/miss`. Tests `player_matches_cache_test.go` (11)
- [x] L5b.5 `db_profiles.json` par mtime (D5b.5) + test — `config/config_players.go` `readDBProfiles` (:68) : contenu garde par chemin, relu seulement si horodatage ou taille change ; horodatage de moins de 2 s jamais cru (`dbProfilesRacyWindow`, :62 : deux ecritures dans le meme tic) ; fichier absent = liste vide, recree = relu ; chaque appel rend une liste neuve. Le PATCH des reglages (ecriture atomique, nouvel horodatage) reste visible sans redemarrage. Tests `config_players_cache_test.go` (5)
- [!] L5b.6 highlight-matches par identifiants (D5b.6) + test — PARTIEL. Fait : `api/handlers/career.go` `enrichHighlightSections` (:308) enrichit les deux sections en UNE requete par identifiants (union des match_id, liste blanche `MatchIDs`) : l'historique complet (`MatchHistoryRepo.LoadAll`) n'est plus charge qu'une fois par requete au lieu de deux, et le `LoadPlayerMatches` du meme GetPage passe par le cache L5b.4 ; codes d'erreur inchanges ; parite ligne a ligne avec l'ancien enrichissement section par section testee (`career_highlight_test.go`, oracle = l'ancien code). Non fait : la requete SQL `IN` — `MatchHistoryService.GetPage` calcule sur l'historique COMPLET des colonnes des lignes servies (taux de victoire par carte `computeMapWinRates`, placements LUSR `applyMatchPlacements`) : un chargement par identifiants changerait ces valeurs (parite rompue) et exige port + repo + service hors perimetre §8
- [x] L5b.7 privacy : echec memorise (D5b.7) + test — deux memoires, chacune garde la reponse d'avant : (1) provider `platform/halo/privacy_provider.go` : un echec Waypoint (statut HTTP, reseau, reponse illisible) garde son repli `fetch_error` `PrivacyFailureTTL` 5 min (:39, `privacyFailure` :156), une fin de contexte (annulation, echeance de l'appelant) n'est jamais memorisee ; (2) `/bootstrap` `service/bootstrap_privacy.go` (extrait de bootstrap_service.go) : budget depasse ou erreur du provider = 5 min sans appel live ni attente de 2 s, nil immediat (repli E3 sur le state persiste, ce que la page servait deja apres les 2 s) (:31, :80) ; annulation de la requete non memorisee. Marqueurs `privacy_cache_hit/miss`, `privacy_live_backoff`. Tests `privacy_provider_failure_test.go` (3), `bootstrap_privacy_test.go` (4)

Gate : `gofmt`, build, vet, `go test ./internal/service/... ./internal/api/... ./internal/config/...
./internal/platform/...`, `-tags=integration -p 1 ./internal/sync/...` si le sync est touche,
lint paquets touches.

Journal du lot (2026-09-23, branche `feat/perf-l5b` depuis `feat/perf-chargements` 37cb48167 ; commits
8507e3dd4 L5b.1, 21dc72cc6 L5b.2, 7cd64d664 L5b.3 + L5b.4, 881011db7 L5b.5, cda8b236f L5b.6,
87ef765e1 L5b.7, puis ce journal) :

- Lectures retenues : (a) saisons — la memoire de l'echec live EST le cache : le repli (TOML +
  base vide) est garde jusqu'au terme de l'attente de 30 min, puis la requete suivante retente ; un
  echec sans jeton dans le contexte (aucun appel reseau fait) n'est ni memorise ni cache, sinon une
  requete anonyme bloquerait 30 min une requete authentifiee ; (b) sections timing (point 9 du
  lot) : MARQUEURS de duree nulle `<cache>_hit` / `<cache>_miss` (et `privacy_live_backoff`) — la
  duree du chargement reste portee par la section appelante deja posee par L1 (`load_matches`,
  `season_counts`, `player_matches`...) : une section chronometree a l'interieur serait comptee deux
  fois dans `total_ms` (les sections sont des feuilles) ; `db_profiles.json` sans marqueur
  (`LoadPlayers` ne recoit pas de ctx, ~100 appelants) ; (c) les caches filtres et historique sont
  PROCESS-WIDE dans `platform/duckdb` (la struct du registre vit dans `wire/registry.go`, hors
  perimetre, et l'invalidation doit etre appelable depuis `internal/sync`), cle (xuid, titre, chemin
  de la base, variante) : deux bases qui porteraient le meme joueur (fixtures de tests, demo) ne
  partagent rien ; (d) point 1 du lot, writers de `player_match_enrichment` hors sync verifies :
  recalcul « avec amis » (`sync.RecomputeIsWithFriends`, declenche par le PUT de la liste d'amis)
  invalide ; exclusion (`MatchExclusionRepo.SetExclusion`) invalide par prudence (les lignes cachees
  ne portent pas is_excluded aujourd'hui) ; favoris : `shared_social`, jamais lu par ces deux
  chargeurs, rien a invalider ; import OpenSpartan (couche service) et sync Halo 5 (runner
  `games/halo_5/livesync`, qui ne passe pas par `runPostSyncPipeline`) : TTL 60 s seulement, hors
  perimetre (Decouvertes) ; les CLI (backfill, `recompute-friends`) sont un autre process : TTL ;
  (e) privacy — DEUX memoires pour que chaque cas garde la reponse d'avant : echec rapide = le
  provider resservait deja `fetch_error` (desormais sans appel reseau pendant 5 min) ; budget de 2 s
  depasse = `/bootstrap` servait le state persiste (E3) apres 2 s, il le sert maintenant tout de
  suite pendant 5 min. Memoriser le depassement dans le provider aurait fait servir `fetch_error`
  (et reecrire le state persiste) la ou E3 s'appliquait : ecarte.
- Fichiers touches hors de la liste litterale du §8, justifies : `platform/duckdb/player_read_cache.go`
  (nouveau, cache generique commun L5b.3/L5b.4 — 2 caches sur la meme mecanique, pas de 3e copie du
  ttlCache) ; `platform/duckdb/player_matches_adapter.go` (doc + `InvalidatePlayer` no-op supprime :
  l'adapter n'est plus le port, code mort, regle 7) ; `platform/duckdb/match_exclusion_repo.go` et
  `sync/friends_recompute.go` (point 1, une ligne d'appel chacun) ; `config/config_players.go` (lieu
  de la lecture de `db_profiles.json`, D5b.5) ; `api/handlers/field_mappings_cache.go` et
  `service/bootstrap_privacy.go` (extraits pour tenir les 500 L : field_mappings.go serait monte a
  524 L ; bootstrap_service.go 630 -> 604 L) ; tests `wire/registry_pages_cache_test.go`,
  `archlint/player_read_cache_invalidation_test.go` (ratchet des trois points d'invalidation).
  `internal/sync` : deux lignes d'appel (+ un import), aucune autre modification.
- Dette : 0 issue lint nouvelle (`golangci-lint run --new-from-rev=37cb48167` sur les 8 paquets
  touches : « 0 issues. ») ; aucune fonction nouvelle > 80 L ni > 5 parametres ; fichiers : tous les
  fichiers crees ou modifies < 500 L sauf deux deja au-dela : `sync/engine_postsync.go` 526 -> 527
  (la ligne d'appel), `service/bootstrap_service.go` 630 -> 604 (baisse). Ratchet FR : un message de
  log accentue passe par `h.logger` (hors exclusion `slog.*`) a ete reformule sans accent.
- Gate (codes de sortie) : `gofmt -l ./internal ./cmd` vide (0) ; `go build ./...` 0 ; `go vet ./...`
  0 ; `go test ./internal/service/... ./internal/api/... ./internal/config/... ./internal/platform/...`
  0, aucun `^--- FAIL:` ; `go test -tags=integration -p 1 ./internal/sync/...` 0 (267 s) ;
  `go test ./internal/archlint/...` 0 ; golangci-lint 0 issue nouvelle.
- Mutations jouees (toutes rouges puis restaurees, `cmp` a l'appui) : saisons — echec non memorise,
  echec sans jeton memorise, copie retiree sur hit puis sur miss, cache desactive, singleflight retire
  (20 fetchs au lieu de 1) ; field-mappings — version sans catalogue de saisons, version sans TOML,
  cache jamais servi ; cache lectures — generation ignoree au store (le chargement perime ecrase le
  frais), cle de vol sans generation (la requete post-invalidation attend le vol perime), copie
  retiree, relance de la requete vivante retiree, invalidation publique sans effet, appel post-sync
  retire de `runPostSyncPipeline` (ratchet archlint rouge) ; historique — AssetReference Map non
  copiee (inventaire rouge + « fatal error: concurrent map writes » sur la re-ecriture concurrente),
  invalidation publique sans le cache historique, cablage Filters sur le repo non cache ;
  db_profiles — horodatage ignore, fenetre de mefiance retiree, cache desactive ; highlight-matches —
  union sans la section des pires, HadBotTeammate non propage ; privacy — echec provider non
  memorise, fin de contexte memorisee, depassement de budget non memorise, annulation memorisee.
- Decouvertes (consignees, non traitees) : (1) `platform/duckdb/highlight_events_cache.go` :
  `CachedHighlightEventsRepo` jamais instancie (code mort) et en-tete perime (« meme strategie que
  player_matches_cache.go ... ttlCache existant ... si un 3e cache emerge, extraire un
  ttlCacheGeneric ») : le generique existe (`playerReadCache`) ; `cacheMetrics` n'est plus garde que
  pour lui ; (2) `service/seasons_catalog.go` : `errEmpty` + `var _ = errEmpty` (musee de code mort) ;
  (3) sync Halo 5 (`games/halo_5/livesync/runner.go`) hors `runPostSyncPipeline` : ses lectures
  cachees ne sont invalidees que par le TTL de 60 s — une ligne d'appel en fin de cycle H5
  l'alignerait ; (4) `service/openspartan_post_import_service.go:213` ecrit `player_match_enrichment`
  depuis la couche service sans invalidation (TTL) ; (5) `/bootstrap` persiste un echec Waypoint
  (`fetch_error`, IsPrivate=false, source « waypoint ») comme etat observe (`Build` ->
  `UpsertPrivacyState`), ecrasant le state persiste reel — preexistant ; (6) la decouverte (6) de L1
  persiste : `go test ./internal/api/...` recree `data/titles/halo_5/warehouse/metadata.duckdb`
  (12 Ko, ignore par git) dans l'arbre du worktree — retire apres le gate ; de meme,
  `go test -tags=integration ./internal/sync/...` cree des repertoires VIDES
  `data/cache/{film_chunks,film_manifests,replays/halo_5,replays/halo_infinite}` dans l'arbre (laisses
  en place : vides, ignores, et on ne supprime rien sous `data/cache` sans raison).

## 9. L4b — Endpoint leger + ancrage avant la requete lourde — vague 3, apres L2 et L4a

Decisions tranchees :
- D4b.1 Nouvel endpoint `GET /players/{slug}/pages/teammates/sessions?teammates=a,b,c&exact=true`
  (Huma, meme handler que teammates) rendant `{ composition_sessions, latest_composition_session }`
  calcules par `TeammatesService.CompositionSessions` : intersection Q30 par coequipier +
  filtre composition exacte (Q32b sans jointure) + `buildCompositionSessionEntries`. Cible
  < 300 ms. Contrat OpenAPI + `generate-types` + `lib/api/types.ts` (MANUEL).
- D4b.2 Front : `useCompositionSessions` part des que la composition est connue ; le selecteur
  de sessions et `decideCompositionReanchor` s'alimentent de cette reponse ; la requete lourde
  `useTeammates` n'est activee qu'une fois la decision d'ancrage prise (snap applique ou
  « aucune session commune » ou selection manuelle valide). `composition_sessions` de la
  reponse lourde reste servi (contrat inchange) et sert de repli.
- D4b.3 Cible (§8) : page a froid stable en ≤ 2 requetes lourdes... non : en UNE requete lourde
  (la bonne session des le depart) + une requete legere.

Perimetre : `internal/api/handlers/teammates.go`, `internal/service/teammates/teammates_service_
composition_sessions.go` (+ nouveau fichier), `internal/port/*` (methode), `api/openapi.yaml`,
`apps/web/src/features/squad/*`, `lib/api/types.ts`, `lib/query/keys.ts`, tests.

Items :
- [ ] L4b.1 endpoint + service + contrat + tests
- [ ] L4b.2 front : hook, selecteur, ancrage, `enabled` de la requete lourde + tests
- [ ] L4b.3 chrono : endpoint leger < 300 ms sur copie ou en mesure de reference

## 10. Cloture de campagne (superviseur)

- [ ] C.1 mesure de reference (§8 protocole) : Escouade a froid / a chaud / clic rail, Synthese,
      Sessions, Series temporelles, Carriere, Accueil ; tableau avant/apres dans l'etat des lieux
- [ ] C.2 revue adversariale du diff cumule `origin/feat/v75..feat/perf-chargements` :
      fan-out aveugle (lentilles : parite des chiffres ; annulation / timeouts / erreurs ;
      caches et invalidation ; front sources de verite et cles de query), deux rondes max
- [ ] C.3 correctifs, gates complets (`go test ./...`, `-tags=integration -p 1`, typecheck `tsc -b
      --force`, eslint, vitest), push, CI verte au niveau job
- [ ] C.4 `.env.local` local : `LEVELUP_DUCKDB_THREADS=8`, `LEVELUP_DUCKDB_MEMORY_LIMIT=4GB`
      (poste 16 coeurs / 32 Go ; prod inchangee)
- [ ] C.5 go utilisateur puis fusion dans `feat/v75` ; entree thought_log ; retrait des worktrees
      (jonctions node_modules retirees AVANT, jamais `--force`)

## 8 bis. Protocole de mesure (superviseur uniquement)

Serveur du checkout principal (ou du worktree d'integration avec `LEVELUP_REPO_ROOT` sur le
principal), `LEVELUP_LOGS_FILE_LEVEL=debug`, Vite, instance Chrome du MCP connectee ; pour
chaque page : `navigate_page` + `performance.setResourceTimingBufferSize(5000)` en `initScript`,
attente de stabilite, puis durees serveur dans `logs/http.log` (`path`, `status`, `duration_ms`,
`response_bytes`) et sections `http_timings` (L1). Escouade : premier passage (localStorage
vide = nouvelle instance ou `localStorage.clear()`), rechargement, clic « session precedente ».
Aucun autre process ne tient les bases pendant la mesure.

## 11. Decouvertes (a consigner, pas a traiter)

- (etat des lieux) `CachedPlayerMatchesRepo` jamais instancie ; `QSquadExpectedWinProbTpl` lit
  `match_skill_rank` et non `_latest` ; `ErrSwapTimeout` en 500 ; sans session piquee, periode
  et cascade ne restreignent pas `allSquadRows` (`teammates_service.go:286-291`) ;
  `CareerLiveBudget` inutilise et en-tete de fichier perime ; provider `sharedprovider` passe
  en « error state » pendant la rafale LUSR du 2026-09-23 10:33.

## 12. Journal

- 2026-09-23 : plan ecrit, branche de campagne creee depuis origin/feat/v75 c89aa4bdc,
  vague 1 lancee (L1, L3, L4a, L6).
