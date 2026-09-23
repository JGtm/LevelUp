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
- [x] L3.1 WriteTimeout 120 s + commentaire date — `cmd/server/main.go:90` (constante
  `serverWriteTimeout`, date, cause, nginx 300 s) lue par `http.Server` (`:1478`) ; le
  garde-rail `internal/api/wire/build_queue_writer_budget_test.go` lisait la forme littérale
  `WriteTimeout: N * time.Second` : réaligné sur la constante (gate rouge sinon)
- [x] L3.2 helper `mapServiceError` + application aux handlers de pages + OpenAPI + tests —
  `handlers/helpers.go:90` (499 DEBUG, puis 503 `errDBBusy()` WARN, sinon 500 ERROR) ; 18
  sites (teammates, filters x2, synthesis, sessions, sessions/detail, timeseries,
  explorer x2, career x9) + `home.go:153` `homePageError` (`home_page_db_busy` et
  `home_page_db_recovering` gardés) ; fragment OpenAPI (réponses partagées `DbBusy`,
  `ClientClosed` ; 499 + 503 sur les 17 opérations) puis `openapi.yaml` et `generated.ts`
  régénérés ; tests `map_service_error_test.go`, `page_service_errors_test.go`
- [x] L3.3 client.ts `signal` + queryClient sans retry 502/504 + tests —
  `lib/api/client.ts:272,321,353` ; `app/queryClient.ts:18` ; `client.test.ts` (4 cas),
  `app/queryClient.test.ts` (nouveau)
- [x] L3.4 `signal` transmis dans les hooks de page listes — 13 `useQuery` (filtres 2,
  synthèse 1, détail de session 1, séries temporelles 1, carrière 6, accueil 2) ;
  `features/timeseries/queries.test.tsx` (démontage pendant le calcul = requête abandonnée)
- [x] L3.5 Touch de session throttle + test — `platform/session/store.go:237` (Touch),
  `:429` (persistedRecently), `:101` (persistMark), `:115` (WithClock), `:301` (purge des
  marques) ; `middleware/session.go` (commentaires) ; tests `store_touch_test.go`,
  `store_marks_test.go`, `middleware/session_touch_test.go`

Gate Go : `gofmt`, `go build ./...`, `go vet ./...`, `go test ./internal/api/... ./internal/platform/
session/...`, `go run ./cmd/openapi-gen -check` (ou la commande documentee dans `docs/COMMANDS.md`),
lint paquets touches. Gate web (depuis `apps/web`, `node_modules\.tmp` purge avant) :
`npm run typecheck`, `npm run lint`, `npx vitest run src/lib/api src/app src/features/filters
src/features/synthesis src/features/session-detail src/features/timeseries src/features/career
src/features/home`.

Journal du lot (2026-09-23, branche `feat/perf-l3`, exécuteur Opus) :
- Choix d'exécution dans le cadre des décisions : (a) le 499 se décide sur le contexte DE LA
  REQUÊTE (`ctx.Err()`), testé avant le 503 (une attente de verrou interrompue par
  l'annulation remonte en `ErrDBLocked`) ; un `context.Canceled` interne (errgroup) alors
  que le client attend reste un 500 ; (b) `isDBBusy` réutilise `isSharedSwapContention`
  (home.go), seul prédicat de contention du paquet : il couvre aussi `ErrProviderClosed`,
  dont le contrat de la sentinelle demande 503 ; (c) Accueil : ses deux 503 historiques
  sont testés d'abord, `ErrDBLocked` passe en 503 `db_busy`, le reste par
  `mapServiceError` ; (d) les lignes ERROR propres à chaque handler (attribut joueur) sont
  remplacées par la ligne unique de `mapServiceError` (`code` + `err`) ; la corrélation
  passe par l'`event_id` du contexte et la ligne d'accès qui porte le chemin ; filters,
  sessions, explorer et carrière rendaient leur 500 sans aucun log : ils journalisent
  désormais l'erreur ; (e) `LastPersistedAt` vit dans le Store (table en mémoire `marks`
  par session : last_seen_at PERSISTÉ + empreinte du contenu hors last_seen_at), car
  `domain.SessionData` est hors périmètre ; la marque retient le last_seen_at écrit et non
  l'heure d'écriture (un `Save` de handler qui réécrit un last_seen_at ancien ne dispense
  pas Touch de rafraîchir : sinon une session active pourrait expirer) ; marques oubliées
  par `Delete` et `PurgeExpired` ; (f) les deux tests de concurrence du store écrivent par
  `Save` (un Touch throttlé ne produirait plus la rafale d'écritures qu'ils exigent).
- Gates : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ;
  `go test ./internal/api/... ./internal/platform/session/...` 0 (6 paquets testés, aucun
  `--- FAIL:`) ; `go run ./cmd/openapi-gen -check` 0 et `tools/check-generated-types-fresh.mjs`
  0 ; `golangci-lint` 2.12.2 sur les paquets touchés, `--new-from-rev=HEAD` : 0 issue ;
  `npm run typecheck` 0 ; `npm run lint` 0 erreur (26 avertissements antérieurs, aucun sur
  les fichiers du lot) ; vitest 69 fichiers, 511 tests verts (14 skips antérieurs de
  `SynthesisPage.test.tsx`).
- Mutations jouées, toutes détectées puis restaurées : 502 rejoué ; 503 non rejoué ; 499
  journalisé en ERROR ; branche 499 neutralisée (500 sur les 18 routes, compteur) ; branche
  503 neutralisée ; throttle neutralisé (deux requêtes à 1 s = deux écritures) ; marque sur
  l'heure d'écriture ; `serverWriteTimeout` à 10 s (garde-rail du dépôt d'ouvrier).
- Découvertes (non traitées) : commentaires devenus faux « le serveur ferme l'écriture à
  30 s » (`internal/api/wire/registry_build_queue.go:253,259,311`,
  `internal/sync/replayartifacts/derivations.go:165`) ; 8 handlers d'écriture testent encore
  `errors.Is(err, dblease.ErrDBLocked)` à la main ; `handlers/setup.go:271` avale l'erreur de
  `Touch` (`_ =`) ; la ligne d'accès du middleware classe le 499 en WARN (classe 4xx,
  compteur 4xx) — `slog_logger.go` est au périmètre L1.

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
- [x] L4a.1 source unique de la session (D4.1) + migration localStorage + tests (un snap = une cle)
  — `features/squad/useSquadSessionSelection.ts:178-186` (session lue dans le store,
  `applySessionLabels` = `setSessions`), `:114-135` (migration unique de `squad-sessions-<slug>` :
  appliquee si le store est vide, retiree dans tous les cas), `lib/query/keys.ts:182` (cle teammates
  sans segment sessions), `features/squad/queries.ts` ; etat local + deux effets de synchro supprimes
  de `SquadLayout.tsx` (638 → 531 L). Preuves : `SquadLayout.requests.test.tsx:164` (snap : 2 cles en
  tout, 2 requetes, les deux champs alignes), `:191` (clic du rail : idem), `useSquadSessionSelection.
  test.tsx:152-194` (store = source, migration, store prioritaire, cle illisible).
- [x] L4a.2 `enabled` de teammates (D4.2) + deep-link + tests — `features/squad/queries.ts:26,43`,
  `useSquadSessionSelection.ts:190-201` (`teammatesReady` = etat de montage pose ET composition
  initiale connue), `features/friends/queries.ts:82` (`useFriendGamertagsState` : `isSuccess`,
  `isError`), `SquadLayout.tsx:166,390` (`isPending` : « Chargement… » tant que desactive, jamais
  l'etat vide). Lien profond : composition posee des le premier rendu (`useSquadSessionSelection.ts:
  148`), session posee dans le store avant l'ouverture des requetes (`:114-135`). Preuves :
  `SquadLayout.requests.test.tsx:131,154`, `SquadLayout.deeplink.test.tsx:71`, `useSquadSessionSelection.
  test.tsx:69-133`.
- [x] L4a.3 aperçu conditionnel + `matchContext` dans resolve et cle (D4.3) + tests —
  `features/squad/useSquadFilterBarState.ts:148` (recalage de `pending` pendant le rendu), `:182-183`
  (apercu `enabled: isDirty`, donnee ignoree hors attente), `features/filters/queries.ts:66,72,198`,
  `lib/query/keys.ts:61`, `SquadLayout.tsx:114` (`{ matchContext: 'squad', enabled: mountApplied }`).
  Preuves : `SquadLayout.requests.test.tsx:219`, `features/filters/useFiltersResolve.options.test.tsx`.
- [x] L4a.4 predicat `routeShowsSoloFilters` partage NavL2 / PlayerLayout (D4.4) + tests —
  `components/shell/shellNavigation.ts:55`, `NavL2.tsx:74`, `routes/{-$lang}/t/$titleSlug/players/
  $playerSlug.tsx:73-84`, `features/filters/queries.ts:125` (`enabled` de `useFollowLatestSession`).
  Preuves : `routes/.../$playerSlug.test.tsx:127` (`/squad/synergies`, `/career`, `/home`,
  `/stats/synthesis` : 0 resolve solo), `:137,147,154`, `shellNavigation.test.ts:12-32`,
  `useFollowLatestSession.test.tsx:139`.
- [x] L4a.5 AssetDrawer `enabled: isOpen` (D4.5) + test — `features/asset-drawer/useAssetDrawer.ts:19,31,
  43`, `AssetDrawer.tsx:21-23` ; preuve `features/asset-drawer/AssetDrawer.test.tsx:38,47`.
- [x] L4a.6 i18n : aucune string nouvelle (aucun texte JSX ajoute ; `no-hardcoded-strings` vert).

Gate (depuis `apps/web`, `node_modules\.tmp` purge avant) : `npm run typecheck` (tsc -b) ;
`npm run lint` ; `npx vitest run src/features/squad src/features/filters src/features/friends
src/features/asset-drawer src/stores src/components/shell src/routes src/lib/query`.

Journal du lot (2026-09-23, executeur Opus, branche `feat/perf-l4a` depuis 97cc0d0c8) :
- Commits : f70bb0772 (L4a.1 a L4a.3, Escouade), d011f3ae7 (L4a.4, resolution solo), 32fbc29fc
  (L4a.5, AssetDrawer), puis ce plan. Pas de push (superviseur).
- Gate : `npm run typecheck` (tsc -b, `node_modules\.tmp` purge) 0 ; `npm run lint` 0 (26
  avertissements, les memes fichiers que la base, aucun nouveau) ; vitest (8 dossiers du gate)
  0 : 113 fichiers, 1 032 tests (rejoue sur l'etat final). Hors gate : suite web complete 0 (786
  fichiers, 8 444 tests ; 3 fichiers et 19 tests ignores preexistants), ratchets knip (0/0/0),
  couleurs, champs, imports croises (7 ≤ 7, `squad=>friends` deja autorise) verts.
- Mutations jouees (code restaure apres chaque) : M1 copie locale de la session synchronisee par
  effet → snap et clic du rail rouges ; M1b idem + sessions dans la cle (ancien schema) → 3
  requetes au lieu de 2 (la requete intermediaire mesuree) ; M2 sans attente de la composition →
  requete sans coequipier, rouge ; M4 sans verrou de montage → premiere requete sur la composition
  ou la session restauree, rouge (lien profond, migration) ; M3 apercu toujours actif et M3b recalage
  de `pending` par effet → apercu relance a chaque snap / changement de session, rouges ; M5
  resolve solo toujours actif → 4 routes rouges ; M6 suivi sans garde d'egalite → snap sur un
  resolu perime, rouge ; M7 suivi toujours actif → rouge ; M8 catalogues tiroir ferme → rouge ;
  M9 resolve escouade sans verrou de montage → premiere resolution sur la session restauree, rouge.
- Precisions d'implementation (aucune decision rouverte) : (a) D4.2 : `isSuccess` est expose par un
  hook frere `useFriendGamertagsState` (et `useFriendGamertags` en derive) pour ne pas changer la
  signature lue par trois autres consommateurs et trois mocks de test hors perimetre (vue match,
  Prestige, rejeu) ; un `/friends` en ECHEC vaut
  liste vide resolue (comportement historique de `useFriendGamertags`), sinon la page resterait sur
  « Chargement… » ; vider la composition est un choix qui garde la requete active ; le verrou de
  montage (`mountApplied`) ouvre aussi le resolve escouade, pour que lien profond et migration
  soient poses avant la PREMIERE resolution aussi. (b) D4.3 : le recalage de `pending` sur le
  commite passe dans le rendu (motif « etat du rendu precedent ») — avec l'effet d'origine, un
  commit « sale » relancait l'apercu a chaque snap (M3b) ; la donnee d'apercu est ignoree quand il
  est desactive (placeholder `keepPreviousData` d'une cle precedente). (c) D4.4 :
  `useFollowLatestSession` n'est actif que si le resolu du store EST celui de la requete courante
  (`soloResolvedContext === soloResolve.data`) : hors Stats le store garde un resolu d'une autre
  page ou d'un autre joueur, y snapper en revenant sur Stats enverrait une requete sur une session
  perimee (M6). (d) `createFilterStore.ts` non modifie (D4.1 ne l'exige pas). (e) `SquadLayout` :
  `isPending` remplace `isLoading` (requete desactivee = « Chargement… ») ; `onReset={resetFilters}`
  (le `applySessionLabels([])` suivant etait devenu un no-op) ; deux directives
  `set-state-in-effect` devenues sans objet retirees. (f) Cles : `filtersResolve` gagne un 4e
  segment `matchContext` (« all » pour le solo, corps solo inchange), `teammates` perd son segment
  sessions ; garde `keys.title-slug.guard.test.ts` adapte a la signature.
- Consequences observables : sur l'Escouade, le rail (precedente / suivante, totaux) parcourt les
  seules sessions escouade (resolu en `match_context` 'squad', D4.3) et non plus toutes les
  sessions du joueur. Sur les pages solo, la resolution solo part a l'arrivee sur une page Stats
  (plus pendant les autres pages) : si une session est apparue entre-temps, la page Stats
  interroge une fois avec le filtre persiste puis une fois apres le snap, comme un chargement a
  froid (a regarder a la mesure de cloture C.1).
- Decouvertes (non traitees, hors perimetre) : (1) `useTeammates` ne transmet pas `signal` (L3 D3.4
  exclut `features/squad/queries.ts`, L4a ne le prevoit pas) : une requete teammates devenue
  inutile n'est pas annulee cote client — a arbitrer (L4b ou cloture). (2) Le store escouade
  (`levelup-squad-filter-v1`) est global, pas par joueur : la session pickee d'un joueur est
  proposee au suivant (le re-ancrage converge, au prix d'une requete) ; la memoire par joueur
  `squad-sessions-<slug>` a disparu avec D4.1. (3) `SquadLayout` s'abonne au store escouade entier
  (`useSquadFilterStore()` sans selecteur) : il se re-rend a chaque `resolvedContext`. (4) Strings
  FR en dur preexistantes, non bilingues : « Chargement… » et les replis de `formatError`
  (`SquadLayout.tsx`), « Analyser » (`SquadFilterBar.tsx`). (5) Un test existant de
  `components/shell` emet « Not implemented: navigation to another Document » (jsdom), bruit
  preexistant.

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
- [x] L6.1 filigrane lu avant l'ecrivain, zero writer en stationnaire + test —
  `skill/skill_v2_watermark.go:58` (`selectShadowWorkUnderRead` : candidats, filigranes par
  `loadGroupWatermarks` sur la vue `_latest`, eligibilite des candidats au-dessus, sous le Read
  relache avant toute rafale) ; `skill/skill_v2_shadow.go:136` (aucun notable au-dessus du
  filigrane → retour sans `Write`) ; tests `skill/skill_v2_watermark_test.go:22` (base fichier,
  lecteur attache READ_ONLY : cycle 2 = 0 `Write`, 1 `Read`, 0 ligne ecrite) et `:76` (joueur
  sans candidat).
- [x] L6.2 une rafale par joueur + test — `skill/skill_v2_shared_access.go:67`
  (`runSingleWriterBurst` : un `Write("lusr")` par joueur et par cycle ; lots de 3 retires, aucune
  transaction ne couvre plusieurs matchs) ; tests `skill/skill_v2_shadow_burst_test.go:207`
  (4 neufs → 1 rafale ; 3 neufs sur historique traite → 1 de plus) et
  `skill/skill_v2_parity_test.go:88` (parite des ecritures, 1 rafale).
- [x] L6.3 ligne INFO par cycle + doc EN/FR — `skill/skill_v2_watermark.go:126`
  (`lusr_v2: rien de nouveau`) et `:140` (`lusr_v2: rafale terminee`), `candidates` et `new` dans
  les deux ; `docs/SYNC_GUIDE.md:182` et `docs/FR/SYNC_GUIDE.md:182`.

Gate : `gofmt` ; `go build ./...` ; `go vet ./...` ; `go test ./internal/sync/...` ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` (OBLIGATOIRE, code
de sortie 0 verifie) ; lint paquets touches.

Journal du lot (2026-09-23, branche `feat/perf-l6`, base 97cc0d0c8) :
- Mesure de depart relue dans les journaux du checkout principal : 1 233 `AcquireWriter` au label
  `sync_v2_postsync/lusr` entre 10:32 et 10:34 (`provider.log`), cinq joueurs, 9 416 candidats,
  `processed` = 0 partout (`general.log`) ; le cinquieme joueur (6 414 candidats) a vu sa rafale
  echouer au 232e triplet (provider en erreur a l'ouverture RW), 5 721 candidats reportes.
- Choix d'implementation (dans D6.1, sans rien rouvrir) : « nouveau » = candidat d'une chaine LUSR
  au-dessus du filigrane de son groupe ET notable (predicat partage `classifyLUSREligibility`,
  evalue sur le lecteur, arret au premier notable). Raison mesuree : un joueur sur cinq garde a
  chaque cycle UN match non notable au-dessus de son filigrane (`skipped_non_two_team=1` a tous les
  cycles depuis la veille : equipes differentes de 2, owner sans equipe ou issue non notable) ; au
  seul filigrane, il prendrait une rafale par cycle pour rien.
  Sous l'ecrivain, `processOneShadowMatch` garde tous ses controles (groupe tenu, filigrane relu
  sur le handle RW, eligibilite) : le pre-filtre n'est qu'une optimisation. Attendu sur le cycle
  mesure : 1 rafale (le joueur dont l'ecriture canonique echoue, retente a chaque cycle) au lieu
  de 1 233.
- Regle n°6 : le predicat « deja traite » a desormais trois consommateurs (scoreur, pre-filtre,
  detecteur de trous) → helper unique `lusrWatermarkCovers` (`skill_v2_watermark.go:37`) + garde-rail
  `lusr_watermark_guardrail_test.go` (et test de la frontiere ≤). Constructeur `newShadowRunContext`
  extrait (partage avec la reference du test de parite). `postsyncLUSRBurstChunk` et
  `loadShadowMatchesUnderRead` supprimes (code mort). Echec d'acquisition de l'ecrivain :
  `slog.ErrorContext` (etait WARN ; au plus une fois par joueur et par cycle).
- Invariants ART : aucune ecriture modifiee (INSERT seuls via `SkillV2Repo.UpsertState` et
  `persist.AppendOnlyLUSRPersister`), lectures par `player_skill_state_v2_latest`, aucune allowlist
  touchee (`no_art_patterns_test.go` et `internal/persist/*` inchanges).
- Parite (point d'attention 5) : `TestLUSRV2Shadow_ParityWithLegacyOrchestration` joue
  l'orchestration d'avant (reference figee dans le test) et la nouvelle sur deux bases jumelles,
  historique deja traite + 10 arrivees (hors ordre sous le filigrane, trois equipes, 3 contre 1,
  abandon, sans chaine, groupe neuf, ecriture canonique en echec puis groupe tenu), mode canonique,
  fuite inter-modes et offsets d'escouade actifs : lignes `player_skill_state_v2` et
  `match_skill_rank` identiques (identifiants, ordre, valeurs), 3 traites des deux cotes, 1 rafale.
- Mutations jouees (chacune restauree ensuite, fichiers verifies identiques) : M1 ecrivain pris
  meme sans notable → rouges stationnaire et sans-candidat (`writeCalls = 1`) ; M2 lots de 3 sous
  l'ecrivain → rouges rafale unique (2 puis 3 rafales) et parite (4 traites contre 3 : le groupe
  tenu n'est plus partage d'un lot a l'autre) ; M3 pre-filtre « groupe deja score = deja vu » →
  rouges parite (1 contre 3), rafale et stationnaire ; M4 eligibilite du seul premier candidat →
  rouge parite (0 contre 3) ; M5 ancienne orchestration restauree en production → ecritures
  identiques a la reference (la reference est fidele), rouges rafale (5 rafales), stationnaire et
  sans-candidat.
- Gates : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ;
  `go test ./internal/sync/...` 0 ; `go test -tags=integration -p 1 ./internal/sync/...
  ./internal/persist/...` 0 (440 s, zero `^--- FAIL:`, 12 paquets ok) ; `golangci-lint run
  ./internal/sync/...` : 30 constats avant, les memes 30 apres (0 nouveau), `--new-from-rev=97cc0d0c8`
  → 0 ; en plus `go test ./internal/archlint/` 0 et `go test ./internal/games/halo_5/livesync/
  ./internal/service/` 0.
- Decouvertes (non traitees) : (a) joueur 2535469190789936 : l'ecriture canonique LUSR echoue a
  chaque cycle depuis au moins 09:49 (« Duplicate key id: N violates primary key constraint » sur
  `match_skill_rank` de sa base joueur : sequence `match_skill_rank_id_seq` en retard sur max(id)) —
  deux groupes tenus, 5 matchs sautes par cycle, trou LUSR tant que la sequence n'a pas depasse
  max(id) (109 occurrences dans `general.log`) ; (b) `buildTwoTeamRosters`
  (`skill_v2_shadow.go`) avale ses erreurs SQL (`return nil, nil, false`, `continue`) : une erreur de
  lecture passe pour « non notable » ; (c) type `shadowParticipant` inutilise (`skill_v2_shadow.go:57`,
  signale par `unused`) ; (d) la vue `player_skill_state_v2_latest` arbitre par `MAX(written_at)` sans
  departage par `id` (recette ADR 0026 : `ROW_NUMBER ... written_at DESC, id DESC`) — deux lignes
  pour un meme (xuid, groupe) si deux horodatages sont egaux, `LoadState` en prend une au hasard ;
  (e) le joueur aux 6 414 candidats en a 3 530 sans chaine LUSR alors que le filtre SQL ecarte
  deja `is_ranked` / `is_firefight` : a verifier (pair_name classe Ranked ou Firefight avec des
  drapeaux a FALSE ?) ; (f) l'appelant `internal/sync/engine_postsync_scoring.go:164` journalise en
  WARN l'erreur rendue par le shadow (lecture du filigrane comprise), hors perimetre du lot.

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
- [ ] L5b.1 saisons : echec memorise + cache (D5b.1) + test
- [ ] L5b.2 field-mappings : cache + ETag avant construction (D5b.2) + test
- [ ] L5b.3 cache filtres + invalidation post-sync (D5b.3) + test
- [ ] L5b.4 `CachedPlayerMatchesRepo` cable + invalidation (D5b.4) + test
- [ ] L5b.5 `db_profiles.json` par mtime (D5b.5) + test
- [ ] L5b.6 highlight-matches par identifiants (D5b.6) + test
- [ ] L5b.7 privacy : echec memorise (D5b.7) + test

Gate : `gofmt`, build, vet, `go test ./internal/service/... ./internal/api/... ./internal/config/...
./internal/platform/...`, `-tags=integration -p 1 ./internal/sync/...` si le sync est touche,
lint paquets touches.

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
- 2026-09-23 (apres-midi) : vague 1 fusionnee dans la campagne — L1 (6fba84c96), L3 (b10ec25b3), L6 (aba010a07), L4a (7088906ee, conflit filters/queries.ts resolu par le superviseur : signal de L3 + matchContext/enabled de L4a) ; gates rejoues sur l'arbre fusionne (build/vet/tests Go, typecheck, vitest 105 fichiers / 967 tests) ; vague 2 lancee (L2, L5a, L5b depuis 37cb48167). Reprises pour L4b : signal sur useTeammates (decouverte L3/L4a).
