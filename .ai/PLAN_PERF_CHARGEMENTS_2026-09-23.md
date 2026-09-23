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
- [x] L2.1 annuaire + retrait des jointures (D2.1) + tests de parite des gamertags — Q29 / Q32 /
  Q32b sans `LEFT JOIN v_gamertag_lookup` (`queries_squad.go`) ; annuaire par LECTURE
  (`duckdb/squad_repo_annuaire.go` : une requete alias + participants sur les memes matchs, jambe
  kill-feed seulement pour les xuids restes sans nom), cascade pure `analysis.AnnuaireGamertags.
  Resolve` + `MaskedXuidLabel` (`analysis/identity_annuaire.go`) ; jambe kill-feed de la vue
  extraite en UN generateur partage vue / annuaire (`identity.go`, DDL de la vue inchange a
  l'octet, sha256 9aebfa6b…) ; tests : `identity_annuaire_test.go` + 4 tests d'integration contre
  la VRAIE vue (`squad_repo_annuaire_test.go`) ; ecarts et « une fois par GetPage » : cf. journal
- [x] L2.2 `LoadImpactEvents` unique (D2.2) — `service/teammates/teammates_service_loads.go` :
  `pourLaRequete` (copie du service par requete, lecteur Q32 memoise par ENSEMBLE de matchs) +
  `prechargerImpacts` (section `impact_events_shared`, avant les quatre blocs) ; GetPage +2 L ;
  tests `teammates_service_loads_test.go` (une lecture pour les 4 consommateurs, deux requetes =
  deux lectures) ; mutation « enveloppe retiree » : 5 lectures, rouge
- [x] L2.3 `LoadFor` unique par membre + xuids sans LoadFor (D2.3) — `teammates_service_loads.go` :
  chargeur memoise par (titre, gamertag) + `precharger` (section `squad_members` : coequipiers
  selectionnes toujours, joueur principal quand la population escouade existe — exactement les
  lectures que les blocs faisaient) ; bandeau sequentiel (`loadTeammatesCanonical`, errgroup
  retire) ; intensite : xuids de `resolveSquadScope(...).xuidByPlayer` (`resolveIntensityXUIDs`
  supprime) ; tests : un LoadFor par membre (avant : 9 pour un coequipier), bandeau seul sans
  population, intensite sans chargeur ; mutations (enveloppe retiree, xuids de la page retires) :
  rouges ; effet decide de D2.3 : un coequipier NON suivi a desormais sa ligne d'intensite (cf.
  journal)
- [x] L2.4 journal des morts restreint et partage (D2.4) — `teammates_squad_echange.go` :
  `lireJournalDesMorts` (section `kill_events_shared`) avec `TacticalQuery.Matchs =
  RestreindreAux(habituel)`, lecture unique partagee par l'echange et le nuage (qui la recoit
  deja restreinte) ; `teammates_squad_isolement.go` : `MortsAvecContexte` sur la MEME liste
  blanche ; test `TestBuildSquadEchange_JournalRestreintALaComposition` (une lecture de chaque,
  liste = [m1 m2]) ; mutations (liste retiree de l'une ou l'autre lecture) : rouges ; parite page
  9/9. GAIN NUL TANT QUE L5a N'EST PAS FUSIONNE : sur la copie, KillEvents 1,9 -> 1,8 s et
  MortsAvecContexte 3,3 -> 3,3 s avec la liste — le lecteur tactique l'applique par
  semi-jointure sur l'univers, sous laquelle DuckDB ne pousse pas les fenetres `_latest` (cf.
  D5a.1). « Univers calcule une fois » : non atteignable ici (chaque lecture tactique calcule le
  sien dans `tactical_repo*.go`, perimetre L5a) — cf. journal
- [x] L2.5 `sessionMatchIDs` depuis `filters.sessions` (D2.5) — `sessionMatchIDsDeLaPage`
  (`teammates_service_briefing.go`, a cote des filtres de session) : l'ensemble se lit sur
  `filteredMatches`, qui porte deja les deux regles (labels, comme
  `filterSynthesisByPickedSessions`) ; GetPage -6 L ; test
  `TestGetPage_SessionFilter_ParFiltersSessions` (2 matchs sur 3, meme population que par
  labels), mutation (condition retiree) : rouge ; copie de production : la requete
  `filters.sessions` seule rend desormais EXACTEMENT la page de `picked_squad_session_labels`
  (7 matchs au lieu de 38), les 8 autres scenarios inchanges
- [x] L2.6 usage / formes partages + `replaylabels` hors chemin de requete (D2.6) —
  `squadagg.LireUsage` + `LecturesUsage` (champ `Lectures` optionnel des deux requetes
  d'assemblage : Synthese et Sessions inchangees) ; `teammates_service_usage.go` :
  `loadUsageBlocks` / `lireUsagePartage` (section `usage_shared`), GetPage -3 L ;
  `replaylabels.Catalogue` (`cache.go`) : catalogue lu une fois par (racine, titre) et par
  processus, echec non memorise, lu par les deux assemblages squadagg ; la PREMIERE requete le
  paie encore (un prechargement au boot releverait du cablage, hors perimetre) ; tests : une
  lecture de chaque (avant : deux), `TestCatalogue_*` ; mutations (partage coupe, cache coupe) :
  rouges ; parite page 9/9 ; usage + formes 118 -> 91 ms sur la page de reference
- [x] L2.7 annulation entre sections (D2.7) — `teammates_service_sections.go` (nouveau) : les
  sections de la population escouade sortent de GetPage (`sectionsDeLaPopulation` ->
  `tableauxDeLaPopulation` + `graphesDeLaPopulation`, entree `populationEscouade`, sortie
  `sectionsEscouade`), chacune sous `siVivante` (lancee seulement si `ctx.Err() == nil`) ;
  GetPage verifie `ctx.Err()` avant l'historique du joueur, avant chaque coequipier, apres la
  boucle, apres les sections et apres les blocs d'usage -> `requeteAnnulee` (erreur du contexte
  enveloppee, reponse vide : jamais une page partielle) ; prechargement : controle entre deux
  membres et avant les impacts (`teammates_service_loads.go`) ; blocs d'usage : trois etapes
  sous `siVivante` (`teammates_service_usage.go`) ; GetPage 316 -> 263 L, fichier 575 -> 522 L ;
  tests `teammates_service_sections_test.go` (7 points d'annulation : `context.Canceled`,
  reponse vide, section en cours presente, suivantes absentes, lecture en cours non refaite ;
  temoin non annule ; blocs d'usage d'une requete deja annulee) ; 10 mutations (chaque point de
  controle retire, `siVivante` inconditionnel, prechargement ou blocs d'usage hors garde) :
  rouges ; parite page 9/9 identique a L2.6

Gate : `gofmt` ; `go build ./...` ; `go vet ./...` ; `go test ./internal/service/...
./internal/platform/duckdb/... ./internal/analysis/...` ; `go test -tags=integration -p 1
./internal/platform/duckdb/...` si un repo est touche ; lint paquets touches ; chrono SQL sur
copie (§0) pour Q29 / Q32 / Q32b avant et apres (attendu : secondes → millisecondes).

Journal du lot (2026-09-23, branche `feat/perf-l2` depuis 37cb48167 ; commits 092a43e6b L2.1,
e0df01c9d L2.1 suite, cf6df1232 L2.2, 033f8d2d5 L2.3, f398b6e29 L2.4, 8c61334ee L2.5, 739fbdefc
L2.6, 7176aaa45 L2.7, b02ea5020 gate, puis ce journal) :

- Chrono SQL (copie de `shared_matches_v2.duckdb` et de la base joueur JGtm dans le scratchpad,
  `access_mode=read_only`, `threads=2`, `memory_limit=512MB`, 3 executions, par les VRAIES
  methodes du repo) : Q29 LoadTopTeammates 1,7-2,4 s -> 21 ms (22-99 ms selon la coupe d'egalite
  du top 50) ; Q32 LoadImpactEvents sur les 38 matchs de la composition 2,1-2,4 s -> 11-17 ms ;
  Q32b LoadMainTeamParticipants sur les memes matchs 2,3 s -> 6-7 ms. Sorties Q32 et Q32b
  identiques ligne a ligne, noms compris ; Q29 identique hors groupe d'egalite du bas.
- Page (sonde GetPage sur la copie, meme cablage que le serveur, une execution par scenario ;
  la mesure de reference reste celle du superviseur, §8) : composition de 3 coequipiers (38
  matchs) 17,3-20,2 s -> 6,6 s ; session de 7 matchs 17,6-18,0 s -> 6,5 s ; session par
  `filters.sessions` 17,2-18,0 s -> 6,4 s ; composition exacte 17,2-17,8 s -> 6,5 s ; un
  coequipier (579 matchs) 19,6-25,4 s -> 8,7 s ; periode 18,6-19,6 s -> 8,1 s ; coequipier non
  suivi 16,5 s -> 6,3 s ; sans selection 2,1-2,3 s -> 0,27 s. Reste, sur la session de 7
  matchs : `echange` 3,2 s (le nuage d'isolement y lit le contexte des morts) et
  `kill_events_shared` 1,9 s, les deux lectures tactiques que la liste blanche n'accelere pas
  tant que D5a.1 (L5a) ne restreint pas l'univers avant les fenetres `_latest` ; le reste de la
  page : ~1,4 s. Cible D2.8 sur la copie : « a vide » 0,27 s (<= 1 s) ; « a chaud, 7 matchs »
  6,5 s (> 5 s, dependance L5a).
- Sections de duree ajoutees (feuilles, cf. L1) : `impact_events_shared` (L2.2), `squad_members`
  (L2.3), `kill_events_shared` (L2.4), `usage_shared` (L2.6). PAS de section `annuaire` :
  l'annuaire est une etape de chaque lecture Q29 / Q32 / Q32b, deja dans la section de
  l'appelant (`top_teammates`, `impact_events_shared`, `main_team_allies`) ; une section imbriquee
  y serait comptee deux fois dans `total_ms`. Sa duree sort en DEBUG (`squad_annuaire` :
  `duration_ms`, xuids cherches, restes pour le kill-feed, matchs lus).
- Ecarts a la lettre des decisions, et pourquoi :
  (1) D2.1 « annuaire charge UNE fois par GetPage » -> un annuaire par LECTURE (Q29, Q32, Q32b),
  sur ses xuids et ses matchs : ces trois methodes du port `SquadRepository` servent aussi
  SquadService et les coequipiers de session de l'accueil, qui lisent le gamertag des lignes ;
  changer le contrat du port sortait du perimetre. Par page : trois annuaires, Q32 n'etant plus
  lue qu'une fois (L2.2).
  (2) D2.1 enumere alias, participants, « Joueur #### », bots ; la vue a un niveau de plus, le
  kill-feed (journal canonique et historique) : sans lui, 27 des 124 noms de la copie tombaient
  en « Joueur #### ». Niveau garde, lu seulement pour les xuids restes sans nom et sur les matchs
  ou la lecture les a rencontres ; son SQL est la jambe MEME de la vue (generateur
  `gamertagKillFeedSQL` partage, DDL de la vue inchange a l'octet, sha256 9aebfa6b…) et vit dans
  `analysis` : le garde-rail `TestAucunLecteurNeLitLAncienneTable` interdit
  `FROM killer_victim_pairs` sous `platform/duckdb`.
  (3) Porteurs de badges ex aequo de la matrice d'impact : `topKiller`, `falseBrother` et
  `silentHero` (`analysis/match_impact.go`) donnent le badge au PREMIER participant a egalite,
  dans l'ordre des lignes Q32b — requete sans ORDER BY, avant comme apres. Retirer la jointure a
  change l'ordre physique : 33 matchs touches sur l'ensemble des 9 scenarios, verifies sur la
  copie (29 a egalite de frags max dans l'equipe du main : Bourreau ; 4 a double egalite :
  Faux-frere x3, Heros silencieux x1) ; ex. Bourreau de c25f8e7c passe de Chocoboflor a
  Madina97294 (composition) ; sur « un coequipier » (566 matchs dans la matrice), joueur
  principal : Bourreau 55 -> 50, Faux-frere 59 -> 58, Heros silencieux 11 -> 10. Aucune stat ne
  change ; le departage etait deja arbitraire (stable pour un plan donne : deux executions de la
  base identiques, L2.1 a L2.7 identiques entre elles). Non corrige : decouverte (1).
  (4) D2.3 : un coequipier NON suivi a desormais sa ligne d'intensite (Nilton410 : 15 / 15
  matchs au lieu de 0) — effet voulu de D2.3.
  (5) D2.5 : la requete `filters.sessions` seule rend la page de `picked_squad_session_labels`
  (7 matchs au lieu de 38) — effet voulu.
  (6) D2.4 « l'univers n'est calcule qu'une fois » : hors d'atteinte dans le perimetre, chaque
  lecture tactique calcule le sien dans `tactical_repo*.go` (L5a).
  (7) D2.6 : catalogue d'armes memorise par (racine, titre) et par processus ; la premiere
  requete le paie encore (un prechargement au boot releve du cablage).
  (8) D2.7 : les sections de population sortent de GetPage (`teammates_service_sections.go`)
  pour y poser les points de controle sans l'allonger ; une lecture EN COURS va a son terme (les
  trois lectures communes d'usage notamment : `squadagg.LireUsage` ne consulte pas le contexte
  entre elles), la section suivante n'est pas lancee.
- Parite finale (forme canonique, 9 scenarios, copie) : identique a L2.6 sur les 9. Par rapport a
  la base, chaque difference a sa cause : `impact_matrix` (ecart 3, cinq scenarios),
  `intensity_profile` du non suivi (ecart 4), `filters.sessions` = page de la session de 7 matchs
  (ecart 5) ; `map_heatmap` exclu sur « un coequipier » et « periode » (non-determinisme
  PREEXISTANT : deux executions de la base different). Hors ces sections, pages identiques a la
  base.
- Dette : aucune fonction nouvelle au-dela de 80 L ; GetPage 323 -> 263 L ;
  `teammates_service.go` 579 -> 522 L (toujours au-dela de 500, en baisse) ; aucun autre fichier
  touche ne franchit 500 L (max `teammates_squad_echange.go` 477) ; fonctions preexistantes au-dela
  de 80 L non allongees (buildSquadIntensityProfile 82 = 82, buildSquadPerMinuteStats 107,
  extractSynthesisSessionLabels 82) ; buildSquadIntensityProfile a 7 parametres (limite revive
  `argument-limit` = 7). golangci-lint `--new-from-rev=37cb48167` sur les 5 paquets touches :
  0 issue (ST1023 releve par le gate, corrige en b02ea5020).
- Gate (code final b02ea5020) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ;
  `go vet ./...` 0 ; `go test ./internal/service/... ./internal/platform/duckdb/...
  ./internal/analysis/...` 0 (27 paquets, aucun `--- FAIL:`) ; `go test -tags=integration -p 1
  ./internal/platform/duckdb/...` 0 ; `go test ./internal/archlint/` 0 (dont
  `no_slug_comparison_test.go`) ; les 7 tests de `internal/sync/no_art_patterns_test.go` verts ;
  aucun test renomme ni supprime (22 ajoutes) : baseline JSONL inchangee.
- Mutations jouees (toutes rouges puis restaurees, `cmp` a l'appui) : L2.1 — bot connu rendu en
  xuid brut, xuid inconnu rendu brut au lieu de « Joueur #### », niveau kill-feed saute, libelle
  masque compte en octets (unitaires), jambe kill-feed jamais lue, niveau participants ignore
  (integration, contre la vraie vue) ; L2.2 — enveloppe retiree ; L2.3 — enveloppe retiree,
  xuids de la page retires ; L2.4 — liste blanche retiree de l'une puis l'autre lecture ; L2.5 —
  condition `filters.sessions` retiree ; L2.6 — partage coupe, cache coupe ; L2.7 — dix (chaque
  point de controle de GetPage, les deux du prechargement, `siVivante` inconditionnel,
  prechargement hors garde, blocs d'usage sans garde).
- Decouvertes (consignees, non traitees) : (1) Q32b sans ORDER BY et departage « premier a
  egalite » de `topKiller` / `falseBrother` / `silentHero` : le porteur d'un badge ex aequo
  depend du plan physique de DuckDB ; un ORDER BY (match_id, xuid) ou un departage explicite le
  rendrait stable — changement de sortie a decider ; (2) `map_heatmap` non deterministe d'une
  execution a l'autre sur les grandes populations (un coequipier, periode) ; (3)
  `formes_retenues.matches[].objective.family` alterne zones_koth / zones_strongholds d'une
  execution a l'autre sur le meme match ; (4) Q29 `ORDER BY games_together` sans departage : la
  coupe LIMIT 50 du groupe d'egalite du bas est arbitraire ; (5) `api/handlers/teammates.go:72-75`
  journalise une requete annulee en ERROR (« teammates: erreur service ») et repond 500 — deja le
  cas quand une lecture echouait sur contexte annule ; distinguer `context.Canceled` releve du
  handler (hors perimetre) ; (6) les deux lectures tactiques font 5 des 6,5 s de la page de
  session : D5a.1 (L5a).

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
- [x] L5a.1 univers restreint a la liste (D5a.1) + chrono avant/apres sur copie
- [x] L5a.2 portee / records sur le perimetre (D5a.2) + tests de parite
- [x] L5a.3 evenements une fois par requete (D5a.3)

Gate : comme L2 (paquets touches) ; `-tags=integration -p 1 ./internal/platform/duckdb/...`.

### Journal du lot L5a (2026-09-23, branche `feat/perf-l5a` depuis `feat/perf-chargements` 37cb48167)

- Mesure : COPIE de `shared_matches_v2.duckdb` (1,3 Go) dans le scratchpad `l5a/`, outil temporaire
  `cmd/perfprobe_l5a_tmp/` (supprime avant commit) appelant les VRAIS repos a travers un driver
  chronometre, `access_mode=read_only`, 2 threads, 512 Mo ; joueur JGtm (xuid 2533274823110022,
  1 158 matchs hors Firefight) ; perimetres = 6 derniers matchs, 30, tout l'historique ; seconde
  execution retenue (cache chaud). Les binaires avant/apres sont construits sur le code de base
  (fichiers repos de 37cb48167 remis le temps de la construction) puis sur le code du lot.
- Diagnostic (question de la consigne) : la liste etait deja appliquee EN SQL (`clausePerimetre`
  ajoute `mr.match_id IN (...)` au WHERE de l'univers), pas en Go apres balayage. Mais (a) le EXISTS
  correle sur `match_kill_events_latest` ne la recevait pas, et la fenetre `QUALIFY ... OVER
  (PARTITION BY match_id)` de la vue se calculait sur les 3,95 M lignes a chaque lecture ; (b) les
  equipes, le journal, les positions et l'isolement re-selectionnaient l'univers en sous-requete
  (`IN (SELECT u.match_id FROM (univers) u)`) : univers re-evalue avec son EXISTS (0,45 a 0,5 s par
  requete) et semi-jointure jamais poussee sous les fenetres des vues `_latest`. Meme defaut dans
  `WeaponRangeRepo` : le `e.match_id IN (...)` ne traverse pas la jointure jusqu'a
  `kill_positions_latest` / `kill_openings_latest`. DuckDB ne pousse sous une fenetre qu'un filtre
  CONSTANT sur la cle de partition ; seule une EGALITE `= ?` se propage par la jointure
  (`KillDistanceRepo`, un match : 15 ms contre 10 ms, non touche).
- L5a.1 (`tactical_repo_univers.go:95,178,239,278`, `tactical_repo.go:216,251,287,311`,
  `tactical_repo_isolement.go:71,106`) : liste recopiee DANS le EXISTS (jeton `%PERIMETRE_JOURNAL%`) ;
  equipes, journal, positions et isolement lus sur la liste des matchs RENDUS par l'univers
  (`listeDeLUnivers`), posee sur CHAQUE vue `_latest` (e ; kp + e ; e + c + p). Avant -> apres, ms,
  n = 6 / 30 / 1 158 : `KillEvents` 1 794 / 1 867 / 1 901 -> 59 / 79 / 996 ; `KillEvents` sans liste
  (page Escouade aujourd'hui) 1 827-2 056 -> 1 078-1 250 ; `MortsAvecContexte` 3 260 / 3 361 / 3 529 ->
  80 / 122 / 1 727 ; `Univers` 1 055 / 995 / 1 088 -> 16 / 22 / 389. Detail a 6 matchs avant :
  univers 502, equipes 464, journal 743 ; apres : 12, 2, 44 (isolement : 14, 3, 63).
- L5a.2 (`weapon_range_repo.go:106,164,212,235,268`) : `kp.match_id IN (...)` ajoute a la portee
  (`buildWeaponRangeScope`, liste liee trois fois : `fragSolo`, `e`, `kp`) ; `LoadWeaponRange` et
  `LoadWeaponOpening` lisent les DEUX cotes en UNE requete (« le joueur est tueur OU victime »,
  separation en Go par `separerCotes`, joueurs designes par `joueursDuFiltre` via `appendXUIDFilter`,
  jamais une copie du sous-select `xuid_aliases`). Avant -> apres, ms, 6 / 30 / 1 158 :
  `LoadWeaponRange` 1 324 / 1 449 / 3 692 -> 64 / 88 / 2 024 ; `LoadWeaponOpening` 630 / 734 / 3 025 ->
  60 / 104 / 1 838 ; `LoadMatchRangeKills` 733 / 776 / 2 231-2 963 -> 85 / 165 / 2 126-2 894 (neutre a
  l'historique complet : ecart dans le bruit de six executions). `synthesis_weapon_records.go`
  inchange : la section profite de la lecture unique.
- L5a.3 : page Sessions (`coordination_block.go:109,126,178,202`, `session_page_coordination.go:79,
  90,123`) : journal des morts + appuis des TROIS scopes (session, comparee, reference) en une
  lecture COMPLETEE (`lectureCoordination` : les deux sessions ensemble, puis le seul complement de
  la reference, et seulement si elle sert) ; chaque bloc se decoupe par `coordination.Restreindre`.
  Series temporelles (`timeseries_service.go:357,369,376`, `timeseries_service_sections.go:220,
  226`) : participants lus UNE fois (`lireEquipesDuScope`, section `participants` qui remplace
  `team_sizes`) pour la courbe d'equipe de l'intensite et la parite de la coordination.
  `highlight_events` : deja charges une fois par requete et sur le perimetre (Sessions : un appel
  pour courante + comparee, `attachSessionEventBlocks` ; Series : `loadHighlightEvents`, un appel) —
  constat, rien a changer (8 / 11 / 195 ms). Attribution des 5 s sans journal de Series temporelles :
  dans `attachMigratedSections`, le seul producteur apres la coordination est `range_profiles`
  (`LoadMatchRangeKills` tous joueurs + compte des frags publiables sur le perimetre de la page :
  2,2 a 2,9 s sur copie a l'historique complet) — pas une relecture identique, une lecture VOISINE de
  la portee (cf. decouverte 2) ; traitee par la restriction de L5a.2 (85 / 165 ms a 6 / 30 matchs).
- Scenarios de page (SQL seul, sur copie, meme outil) avant -> apres, ms : Sessions coordination
  (session 6 + comparee 6 + reference 1 158) 7 971-8 258 -> 1 940-2 891 (la variante « trois lectures »
  sur le nouveau code : 1 919-3 017, le partage gagne ~0,1 s, dans le bruit) ; Sessions portee (6 + 6 +
  reference 30) 2 604-2 772 -> 423-539 ; Synthese records (perimetre 6 / 30 / 1 158) 1 755 /
  1 727-1 782 / 4 542-4 608 -> 100-111 / 129-143 / 2 113-2 133 ; Series temporelles (portee + entame +
  roles + coordination, 6 / 30 / 1 158) 5 892-6 008 / 5 922-5 996 / 17 124-21 187 -> 468-474 / 616-619 /
  8 547-9 291.
- Cible D5a.4 (indicative, verifiee par le superviseur §8 bis) : sur un perimetre de session ou de
  periode (<= 30 matchs), ces lectures passent sous 0,7 s par page. Sur TOUT l'historique, elles
  restent : Synthese ~2,1 s (records), Sessions ~2 a 2,9 s (reference d'habituel = periode de la
  page), Series temporelles ~8,5 a 9,3 s — le reste tient a la conception (vues `_latest` a fenetre
  sur tables entieres, lectures voisines non partageables sans changer port/analysis : decouvertes 1
  a 3 et 6).
- Parite : empreintes (multiset ET ordre) identiques avant/apres sur la copie pour `KillEvents`,
  `KillEvents` sans liste, `MortsAvecContexte`, `Univers` (6 / 30 / 1 158) ; multiset identique pour
  `LoadWeaponRange` (gamertag et xuid), `LoadWeaponOpening`, `LoadMatchRangeKills` (6 / 30 / 1 158,
  l'ordre de ces lectures n'a jamais ete deterministe : agregat par hachage, sans ORDER BY) — 22
  comparaisons, 22 identiques. Tests : `tactical_repo_fenetres_test.go` (fenetres `_latest` bornees au
  perimetre pour les quatre lectures, mesure inchangee, gardes de l'isolement),
  `weapon_range_repo_fenetres_test.go` (fenetres bornees pour les trois lectures ; parite une lecture /
  deux lectures sur un corpus qui exerce double frag, unanimite, publiable, bot, suicide, hors scope,
  gamertag a deux xuid, gamertag inconnu, tous les joueurs, pour les deux tables),
  `fenetres_perimetre_helpers_test.go` (outil : note les requetes REELLEMENT envoyees et les rejoue
  sous `EXPLAIN (ANALYZE, FORMAT JSON)`, cardinalite de chaque WINDOW), `session_page_coordination_
  test.go` (une lecture par match ; parite avec trois lectures separees, `reflect.DeepEqual`),
  `timeseries_service_equipes_test.go` (section `participants` appelee une fois, les deux
  consommateurs nourris). Stub `appuisRepoStub` aligne sur le contrat reel (honore la liste).
- Mutations jouees (toutes rouges, puis restaurees depuis copie) : liste retiree du EXISTS (4 rouges :
  fenetres 30 > 6) ; journal en semi-jointure `IN (SELECT unnest([...]))` et contexte des morts idem
  (2 rouges) ; `kp` en semi-jointure (5 rouges : trois lectures du repo) ; suicide absent du cote
  victime et colonne victime neutralisee (parite rouge, 4 cas chacun) ; complement qui relit tout
  (lectures `[[m1 m2] [m1 m2 m3]]` au lieu de `[[m1 m2] [m3]]`, parite rouge) ; decoupage par scope
  retire (2 rouges) ; coordination qui relit les participants et intensite privee de la lecture
  (section `participants` = 2, courbe d'equipe absente).
- Gate : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...` 0 ; `go test
  ./internal/service/... ./internal/platform/duckdb/... ./internal/analysis/...` 0 (27 paquets ok,
  aucun `--- FAIL:`) ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 0 (5 paquets
  ok) ; `golangci-lint run --new-from-rev=37cb48167` sur `internal/platform/duckdb/...` et
  `internal/service/...` : 0 issue. En plus : `./internal/archlint/...` ok, garde-rails SQL de
  `./internal/sync/` ok. Aucun fichier au-dela de 500 L cree ou grossi ; `GetPage` des Series
  temporelles a longueur constante (commentaire resserre d'une ligne).
- Commits : a16e66820 (L5a.1), 6e5b14f80 (L5a.2), c1596ee4d (L5a.3), puis ce journal.
- Perimetre : `session_page_coordination.go` (attachement du bloc coordination de la page Sessions)
  touche en plus des motifs du plan, au meme titre que `session_page_range.go` nomme dans la consigne ;
  tests associes adaptes : `equipment_usage_block_test.go` (signature d'`attachMigratedSections`),
  `kill_measured_scope_test.go` (arguments lies 2N -> 3N), `coordination_block_test.go` (stub).
- Decouvertes du lot (consignees ici, non traitees) : (1) les records de distance (Synthese) n'usent
  que le cote tueur : une lecture d'un seul cote au port ramenerait l'historique complet de ~2,1 s a
  ~1,3 s (`port/weapon_range.go`, hors perimetre) ; (2) Series temporelles : `weapon_range` (portee +
  entame du joueur) et `range_profiles` (tous les joueurs) relisent les memes frags mesures du meme
  scope ; les partager exige la victime sur `analysis.MeasuredKill` ou un port commun (hors
  perimetre) ; (3) Sessions : la portee de la session, de la comparee et de la fenetre de reference
  (qui les contient) sont trois lectures ; les partager exige un total de frags PAR MATCH dans
  `port.MatchRangeRead` (hors perimetre ; cout residuel 0,4 a 0,5 s) ; (4) Series temporelles : les
  blocs usage et formes (`squadagg`, perimetre L2) relisent films, joueurs et participants du meme
  scope (participants : 4 lectures par page avant le lot, 2 apres) ; (5) `TacticalRepo.MortsParCarte`
  (ecran d'entree de l'onglet Tactique) pose la liste sur `mr` seulement : les vues `kp` et `e` se
  calculent sur l'historique entier (meme defaut que D5a.1, page hors lot) ; (6) a l'historique
  complet, une liste `IN` de 1 158 valeurs coute ~0,3 s par vue `_latest` (jointure MARK sur la liste
  sous la fenetre) : pour l'isolement, 1,8 s avec les trois listes contre 1,5 s avec la seule liste de
  `e` (au-dela de ~800 matchs la liste coute plus que la fenetre entiere ; a 100 / 300 / 600 matchs :
  0,16 / 0,43 / 0,82 s contre 0,75 / 0,91 / 1,07 s) ; seul un changement de conception (vues `_latest`
  materialisees) changerait l'ordre de grandeur ; (7) risque de conflit de fusion avec L2 si L2 change
  la signature de `squadagg.BuildSquadFormesBlock` / `BuildEquipmentUsageBlock`, appelees depuis
  `timeseries_service_sections.go` (modifie ici) ; (8) `listeDeLUnivers` lie la liste des matchs de
  l'univers une fois par vue (3 x N arguments pour l'isolement) : mesure jusqu'a 1 158 matchs (3 474
  arguments, 1,7 s), non mesure au-dela — un univers NON restreint (page Escouade tant que D2.4 n'est
  pas fusionne) porte tout l'historique du joueur.

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
- [x] L4b.1 endpoint + service + contrat + tests — port `internal/port/services.go:403`
  (`CompositionSessions(ctx, playerXUID, teammates, exact)`) ; service
  `internal/service/teammates/teammates_service_composition_legere.go:66` (nouveau fichier) :
  Q29 puis `lireComposition` (`:113`, Q30 par coéquipier, `intersectSquadRowsByMatchID`),
  `appliquerCompositionExacte` (`:179`, extraPool, Q32b SOUS L'OPTION seulement,
  `filterExactComposition`), `buildCompositionSessionEntries` ; sans coéquipier
  `sessionsEscouadeDuPrincipal` (`:205`, historique du principal,
  `wrapSessionLabelsAsComposition`) ; jamais une section de la page ; sections de durée
  `top_teammates`, `squad_matches`, `main_team_allies`, `player_matches`,
  `composition_sessions`. Handler `internal/api/handlers/teammates.go:49` (route dans le MÊME
  Mount que la page : mêmes middlewares ownership et titre), `:110` (`MapCapabilityError`
  sonde `match.history`, puis `mapServiceError`), `:134` (composition rognée, vides écartés).
  Contrat : `api/openapi_manual_fragment.yaml:976`, `openapi.yaml` et `generated.ts`
  régénérés, `lib/api/types.ts:1265` (`CompositionSessionsResponse`, re-export du schéma).
  Tests : `teammates_service_composition_legere_test.go:139` (parité avec GetPage sur 16
  scénarios existants du paquet : liste, ordre, `match_count`, `match_count_roster`,
  `excluded_by_exact_composition`, dernière session), `:290` (lectures faites / jamais
  faites), `:346` (annulation), `:366` (sections) ; `handlers/teammates_sessions_test.go`
  (6 cas) ; `page_service_errors_test.go:124` (499, 503, contrat)
- [x] L4b.2 front : hook, sélecteur, ancrage, `enabled` de la requête lourde + tests —
  `features/squad/queries.ts:85` (`useCompositionSessions` : `signal`, `staleTime` 5 min,
  `keepPreviousData`, `retry: false`), `:48` (`signal` transmis par `useTeammates`) ;
  `lib/query/keys.ts:189` (`compositionSessions`, sous le préfixe `teammates`) ;
  `features/squad/useSquadPageRequests.ts` (nouveau) : `:109` (`enabled` de la lourde),
  `:112` (source des sessions), `:126` et `:143` (réconciliation et ré-ancrage, déplacés de
  `SquadLayout` sans changement) ; `squadPending.ts:153` (`pickCompositionSessionsSource`) ;
  `SquadLayout.tsx:162` (531 → 452 L). Preuves : `SquadLayout.requests.test.tsx:204` (à
  froid : une légère puis UNE lourde déjà sur la dernière session), `:225` (composition
  sans session commune vidée avant la lourde), `:237` (clic du rail = une lourde, aucune
  relecture légère), `:262` (légère en échec = séquence L4a) ;
  `useSquadPageRequests.test.tsx:117-176` (verrou de montage, changement de composition,
  bascule de l'option, sélection manuelle valide, sans coéquipier la lourde n'attend pas) ;
  `queries.test.tsx:46-100` (annulation des deux requêtes, URL) ; `squadPending.test.ts:347`
  (source) et `:396` (parité de la décision depuis la légère ou la lourde, scénarios de
  `decideCompositionReanchor`)
- [x] L4b.3 chrono : endpoint léger < 300 ms sur copie — 45 à 197 ms selon la composition
  (détail au journal)

Journal du lot (2026-09-23, branche `feat/perf-l4b` depuis `feat/perf-chargements`
8d016c94a, exécuteur Opus ; commits 87b53622e L4b.1, 382c0fe90 L4b.2, puis ce journal) :

- Chrono (sonde temporaire compilée depuis `cmd/perfprobe_l4b_tmp/`, retirée ; copie de
  `shared_matches_v2.duckdb` du 23/09 16:02 ; `metadata.duckdb` et bases joueurs : copies du
  lot L2, les originales étant tenues par le serveur du checkout principal ; `threads=2`,
  `512MB` = défauts du pool ; câblage de `wire.TeammatesCtx`), 5 exécutions par scénario :
  composition de 3 coéquipiers (7 sessions) 90-197 ms ; la même en composition exacte
  88-149 ms ; un coéquipier (76 sessions) 45-110 ms ; un coéquipier en composition exacte
  (20 sessions, Q32b sur tout l'historique commun : 84-96 ms) 144-196 ms ; coéquipier non
  suivi en composition exacte 70-140 ms ; sans coéquipier 101 ms à froid (process neuf),
  1 ms ensuite (cache de lecture du lot L5b). Sections : `top_teammates` 18-102 ms (bimodal
  20 / 80 ms d'un appel à l'autre), `squad_matches` 24-94 ms, `main_team_allies` 5-96 ms,
  `composition_sessions` ≤ 1 ms. Avant L4b, ces deux champs n'arrivaient qu'avec la page :
  sur la même copie 0,95-1,19 s (3 coéquipiers), 3,8-4,7 s (un coéquipier), 1,2-1,3 s (un
  coéquipier, exacte), 0,16-0,19 s (sans coéquipier). Cible < 300 ms : tenue partout.
- Parité sur données réelles (même sonde) : à Q29 FIGÉ (une lecture servie aux deux chemins),
  les deux champs sont identiques à l'octet sur les 6 scénarios. À Q29 réel, 2 scénarios
  « exacte » sur 6 diffèrent de la page : Q29 rend deux ensembles de 50 xuids DIFFÉRENTS sur
  deux lectures consécutives (coupe arbitraire du groupe d'égalité, découverte (4) du lot L2),
  donc un extraPool différent ; la page elle-même n'est pas stable (5 pages de suite, un
  coéquipier en composition exacte : 4 différentes de la première). Non corrigé (hors
  périmètre) : découverte (1) ci-dessous.
- Précisions d'implémentation (aucune décision rouverte) : (a) réponse : les deux champs
  TOUJOURS présents (liste vide, chaîne vide) là où la page les omet ; (b) `exact` absent =
  false, comme `filter_exact_composition` absent du corps de la page ; (c) composition rognée,
  vides écartés (`?teammates=` = aucun coéquipier) ; (d) la résolution d'un coéquipier (top
  50 sans casse, puis alias) est une seconde copie de celle de `buildTeammateRowWithMatches`
  (`teammates_service_kpis.go`, hors périmètre ; CLAUDE.md n°6 : deux au plus), verrouillée par
  le test de parité ; (e) `pourLaRequete` est appelée : aucune lecture mémorisée par elle
  (Q32, LoadFor) n'est empruntée sur ce chemin aujourd'hui, une lecture partagée qui s'y
  ajouterait passerait par la même mémoire ; l'annuaire de L2 est celui de Q29 et Q32b ;
  (f) Q32b n'est lue que sous l'option : hors option la page ne s'en sert que pour ses
  sections ; (g) front : la décision est retenue par composition (titre, joueur, composition
  triée, option), posée APRÈS l'écriture de l'ancrage dans le store ; sans coéquipier la
  lourde part aussitôt (la décision est `none` par construction) et la légère nourrit
  seulement le sélecteur ; (h) `retry: false` sur la légère : son échec retombe aussitôt sur
  la lourde plutôt qu'après le backoff (1 s + 2 s) ; (i) clé sous le préfixe `teammates`
  (l'invalidation après ajout d'ami la couvre ; l'extraPool dépend des amis), sans locale ni
  filtres ; (j) la clé lourde d'avant l'ancrage existe dans le cache sans avoir été chargée
  (requête désactivée) : les tests comptent les clés CHARGÉES ; (k) infrastructure de test :
  handler MSW par défaut de la route légère (`src/test/handlers.ts`), et le test du lien
  profond neutralise aussi la légère (même intention que pour la lourde) ; (l) les deux tests
  L4a qui attendaient DEUX requêtes lourdes au snap (`SquadLayout.requests.test.tsx`) sont
  réécrits pour l'ordre L4b (une légère, une lourde), la séquence L4a restant exigée, elle,
  quand la légère échoue.
- Gates (code final 382c0fe90) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ;
  `go vet ./...` 0 ; `go test ./internal/service/... ./internal/api/... ./internal/port/...` 0
  (12 paquets, aucun `--- FAIL:`) ; `go run ./cmd/openapi-gen -check` 0 ;
  `node tools/check-generated-types-fresh.mjs` 0 ; golangci-lint 2.12.2
  `--new-from-rev=8d016c94a` sur les 3 paquets touchés : 0 issue ; `npm run typecheck` 0 ;
  `npm run lint` 0 erreur (26 avertissements, tous antérieurs, aucun sur les fichiers du
  lot) ; vitest `src/features/squad src/features/filters src/lib/query src/lib/api` 0 (85
  fichiers, 770 tests). Hors gate : suite web complète 0 (790 fichiers, 8 476 tests ; 3
  fichiers et 19 tests ignorés, antérieurs) ; ratchets knip (0/0/0), imports croisés (7 ≤ 7),
  couleurs, champs, contrat : verts. Aucun test renommé ni supprimé côté Go (baseline JSONL
  inchangée).
- Mutations jouées, toutes rouges puis restaurées (`cmp`) : Go — option exacte ignorée,
  dernière session = la plus ancienne, Q32b lue hors option, sessions solo au lieu d'escouade
  sans coéquipier, contrôle d'annulation retiré, coéquipier introuvable gardé dans
  l'intersection, composition non nettoyée, `MapCapabilityError` retiré, liste nulle non
  normalisée ; web — lourde sans attendre la décision, repli retiré, ancrage sur un
  placeholder, sans coéquipier la lourde attend la légère, `signal` retiré de la lourde puis
  de la légère, option absente de la clé légère, source légère ignorée.
- Découvertes (non traitées, hors périmètre) : (1) Q29 (`LoadTopTeammates`, `ORDER BY
  games_together DESC LIMIT 50` sans départage, `queries_squad.go:54`) : l'ensemble du top 50 change d'une
  lecture à l'autre, donc l'extraPool de la composition exacte, donc les sessions, comptes et
  responsables nommés de la composition exacte ne sont pas reproductibles d'une requête à la
  suivante — page comme lecture légère (mesuré ci-dessus) ; conséquence plus lourde que la
  découverte (4) de L2 ; un départage (`xuid`) le rendrait stable. (2) POST `/pages/teammates`
  ne passe pas par `MapCapabilityError` : une `ErrCapabilityNotSupported` y rendrait 500 (la
  route légère rend 503). (3) `buildCompositionSessionLabels` trie par `StartedAt` avec
  `sort.Slice` sur une tranche issue d'une map : ordre non déterministe entre deux sessions de
  même début (théorique). (4) La lecture légère sans coéquipier lit tout l'historique du
  joueur principal (101 ms à froid) pour n'en garder que les sessions escouade.


## 9 ter. L7 — Carriere : rencontres et rivaux sans `v_gamertag_lookup` (Go) — ajoute le 2026-09-23 a la mesure intermediaire

Decouverte de la mesure intermediaire (campagne sans L4b, serveur du worktree d'integration sur
les donnees reelles) : `GET /pages/career/top-encounters` 10,7 s et `GET /pages/career/rivals`
10,1 s, en parallele, a chaque ouverture de la page Carriere (non capte le matin : mon attente
etait trop courte, ce n'est donc pas une regression). Cause : quatre `LEFT JOIN v_gamertag_lookup`
dans `platform/duckdb/queries_career_encounters.go` (:71, :118, :218, :321), meme defaut que C1,
meme remede que L2 (annuaire par lecture, `squad_repo_annuaire.go`).

Decisions tranchees :
- D7.1 Retirer les quatre jointures ; nommer les lignes par l'annuaire de L2 (`nommerLignes` /
  `annuaireDeLecture` ou une variante partagee, sans dupliquer la cascade : alias, participants
  des matchs de la lecture, kill-feed pour les restes, `Joueur ####`, bots `bid(`).
- D7.2 Parite stricte des gamertags et des compteurs sur les fixtures existantes et sur copie
  (empreintes avant/apres) ; les ex aequo eventuels sont journalises comme pour L2.
- D7.3 Chrono avant/apres sur copie (recette §0) : attendu secondes vers dizaines de ms.
- D7.4 Sections `timing` sur les lectures.

Perimetre : `internal/platform/duckdb/{queries_career_encounters.go,career_repo_encounters.go,
career_repo.go}` (+ `squad_repo_annuaire.go` seulement pour exposer un helper commun, sans
changer son comportement), `internal/service/career_service_encounters.go` si une signature
change, tests associes.

Items :
- [x] L7.1 rencontres (top-encounters) sans jointure + parite + chrono — Q26 sans jointure ni
      colonne de nom (`queries_career_encounters.go:6`) ; `GetTopEncountersGlobal`
      (`career_repo_encounters.go:31`, 102 -> 29 L : `topEncountersQuery`, `scanTopEncounters` :86,
      `encounterFromStats` :116) nomme par l'annuaire de L2 reutilise tel quel (`nommerLignes`,
      `squad_repo_annuaire.go`, seul l'en-tete change) sur les matchs de l'historique du joueur
      (`QMatchsDuJoueurTpl` :26, token Campagne ; `nommerSurLHistorique` :254) ; sections
      `top_encounters` / `top_encounters_annuaire` (:42, :49) ; tests `career_repo_annuaire_test.go:20`
      (un niveau de la cascade par xuid, contre la VRAIE vue), `:92` (ecart nomme : nom hors de
      l'historique), `:121` (sections) ; ratchet `annuaire_ratchet_test.go:41` ; 200 lignes servies
      identiques a l'octet sur la copie (5 joueurs, avec et sans amis) ; 2,6-3,7 s -> 0,82-1,33 s
      (reste : fenetre du kill-feed, cf. journal) ; commit aae43375a
- [x] L7.2 rivaux sans jointure + parite + chrono — Q27 sans jointure, `MIN(kv.match_id) AS
      match_rencontre` (`queries_career_encounters.go:100`, :107) ; `GetRivals`
      (`career_repo_encounters.go:163`) : une connexion, deux lectures, UN annuaire pour les deux
      listes, lignes `rivalLu` (:155) portant le match du duel ou la jambe kill-feed cherche un
      adversaire qu'aucune ligne participant ne connait ; sections `rivals` (2 appels) /
      `rivals_annuaire` (:219, :189) ; test `career_repo_annuaire_test.go:63` (dont x_kfseul, connu du
      seul kill-feed) ; 100 lignes servies identiques a l'octet ; 6,6-7,0 s -> 1,94-2,18 s ; commit
      cc72023b8
- [x] L7.3 autres lecteurs du meme fichier (grep `v_gamertag_lookup` dans `queries_career*.go`,
      `career_repo*.go`, `home_repo*.go`) : traites s'ils sont sur une page, sinon consignes — grep
      etendu par la consigne (Relations, Comparer, Explorer, vue match, Medias, gamertag_repo) ; 2
      lectures traitees, remplacement mecanique et parite tenue : Q10 (`queries_match.go:14`,
      `GetEncounters` `career_repo_encounters.go:286`, liste `/career/encounters` du selecteur de
      composition de l'onglet Tactique) 2,25-2,62 s -> 28-30 ms ; Comparer (`GetLocalStats`
      `compare_repo.go:30`, `localStatsQuery` :87) 2,3-2,6 s -> 9-29 ms par appel ; tests
      `career_repo_annuaire_test.go:167`, `compare_repo_annuaire_test.go:18` ; `home_repo*.go` : aucune
      lecture de la vue ; le reste CONSIGNE avec son cout (journal, (a) a (f)) et fige par le ratchet ;
      commit 0406132ff

Gate : comme L2 (paquets touches) ; `-tags=integration -p 1 ./internal/platform/duckdb/...`.

Journal du lot L7 (2026-09-23, branche `feat/perf-l7` depuis `feat/perf-chargements` 2beeba665 ;
commits aae43375a L7.1, cc72023b8 L7.2, 0406132ff L7.3, aa2ec4880 gate, puis ce journal) :

- Mesure : COPIE de `shared_matches_v2.duckdb` (1,3 Go ; la copie L2 de 12:41 recopiee dans `l7/` :
  la base reelle, tenue par le serveur de mesure, n'a pas ete ouverte), sonde temporaire
  `cmd/perfprobe_l7_tmp/` (jamais commitee) appelant les VRAIS repos a travers un driver chronometre,
  `access_mode=read_only`, 2 threads, 512 Mo ; binaire « avant » construit sur l'arbre de base exporte
  (`git archive 2beeba665`), « apres » sur le worktree ; cinq joueurs suivis (JGtm 1 160 matchs,
  Madina97294 1 275, Chocoboflor 587, XxDaemonGamerxX 39, Nuzzles 7 190). La vue seule
  (`SELECT count(*) FROM v_gamertag_lookup`) : 1,7-1,8 s au repos.
- Chrono avant -> apres (JGtm, deux tours intercales de 3 executions) : rencontres (Q26) 2,6-3,7 s ->
  0,82-1,33 s (Q26 0,84-1,30 s + matchs de l'historique 13-21 ms + annuaire 9-17 ms) ; rivaux (Q27 x 2)
  6,6-7,0 s -> 1,94-2,18 s (Q27 1,06-1,08 s par lecture + 14 ms + 9 ms) ; Q10 2,25-2,62 s -> 28-30 ms ;
  Comparer 2,27-2,59 s -> 9-29 ms par appel (4 appels : 9,3-9,9 s -> 81-87 ms). Cinq joueurs, apres :
  rencontres 0,80-1,37 s, rivaux 1,54-2,24 s (Nuzzles : annuaire 40 ms + jambe kill-feed 42 + 85 ms
  pour ses deux « Joueur #### »), Q10 9-180 ms, Comparer 67-86 ms le bloc de 4.
  ATTENDU « DIZAINES DE MS » NON ATTEINT POUR Q26 ET Q27 : ce qui reste est la fenetre `QUALIFY ...
  OVER (PARTITION BY match_id)` de `match_kill_events_latest` (3,95 M lignes), que le filtre tueur /
  victime ne traverse pas (kv_stats seule : 0,76-1,14 s) — meme defaut que D5a.1 / decouverte (6) de
  L5a, hors des decisions D7 : decouvertes (1) et (2).
- Parite (D7.2) : L7.1 / L7.2 : 300 lignes servies (rencontres avec et sans amis, rivaux, 5 joueurs)
  identiques a l'octet avant/apres (noms, compteurs, ordre), au code final. Au-dela des lignes servies,
  sur TOUS les joueurs croises ou affrontes des cinq joueurs (58 353 couples joueur / croise) : zero
  ecart pour quatre joueurs ; 8 chez Nuzzles, « Joueur #### » la ou la vue trouve un nom hors de ses
  7 190 matchs (ses participants n'ont pas de gamertag), aucun dans une ligne servie — l'ecart nomme de
  D7.1 (participants des matchs de la lecture), fige par `TestCareerRepo_Annuaire_EcartNomme_
  NomHorsHistorique`. Rivaux : les 269 adversaires sans alias ni nom de participant (JGtm, Madina97294,
  Chocoboflor) recoivent le nom de la vue ; un adversaire connu du SEUL kill-feed (aucune ligne
  participant ; 1 sur la copie, chez Nuzzles, nomme par alias) serait reste masque par la lecture
  agregee — d'ou `match_rencontre`. Q10 : lignes communes identiques (noms, compteurs) ; ne different
  que les ex aequo de la coupe LIMIT 50 (`ORDER BY match_count DESC` sans departage, deja differents
  d'une execution a l'autre AVANT le lot : deux ensembles sur deux executions pour JGtm et Nuzzles) —
  JGtm 1 ligne, Madina97294 4, Chocoboflor 2, Nuzzles et XxDaemonGamerxX 0. Comparer : 20 sorties
  (joueur + ses trois co-participants les plus frequents, 5 joueurs) identiques a l'octet ; zero ecart de
  nom sur les 53 061 xuids de la base (la lecture couvre tout l'historique du joueur compare).
- Lectures de la vue CONSIGNEES (L7.3), cout mesure (JGtm, dernier match, un appel) :
  (a) Relations `GetRelations` (Q28 et Q28 scope, `queries_career_encounters.go:146,249`) : 3,4-4,0 s
  (scope 30 matchs : 2,4-2,7 s) ; remplacement mecanique mais parite NON tenue : 2 lignes servies de
  Nuzzles (sur 7 450) passeraient a « Joueur #### », et l'annuaire nommerait jusqu'a 7 450 xuids dont
  2 973 sans alias ni nom ; sans la vue : ~0,9 s (fenetre du kill-feed, comme Q26) + annuaire (non mesure) ;
  (b) vue match : Q12 tableau de score 1,7-2,2 s (3-6 ms sans la jointure, mesure), Q21 evenements 2,2 s,
  Q23 rencontres 2,2 s, Q23b stats de rencontre 3,6 s (dont ~0,9 s de fenetre du kill-feed) — de l'ordre
  de 10 s de vue par ouverture si les quatre sont servies ; Q12 / Q23 / Q23b mecaniques (annuaire du
  match) mais parite NON tenue : 11 couples (match, joueur) sur 93 636, dans 10 matchs, passeraient a
  « Joueur #### » (la vue les nomme hors du match) ; Q21 non mecanique (un xuid absent de la vue y est
  NIL et le service affiche le xuid brut ; l'annuaire rendrait « Joueur #### ») ;
  (c) `GamertagRepo.ResolveGamertags` (evenements de match) : 2,1 s ; non mecanique (le port ne porte
  aucun match, et un xuid inconnu doit y etre ABSENT de la carte) ;
  (d) Explorer `ResolveXUIDByGamertag` : 2,3 s ; non mecanique (recherche par NOM en ILIKE ; l'annuaire
  nomme des xuids) ;
  (e) Medias `loadMatchLobbies` (associations media / match) : une evaluation de la vue par appel
  (1,7-3 s, non mesuree seule) ; page hors de la liste ; lecture par match comme Q12 ;
  (f) hors de la liste de la consigne : heatmap Relations Q29 (`queries_relations_moments.go`, 2,3 s) et
  classement mondial (`leaderboard_world_repo.go`), une evaluation de la vue chacune.
  Le ratchet `TestLecturesDeLaVueDesNoms_Ratchet` (analyse des litteraux SQL, pas des commentaires) fige
  ces 11 lectures fichier par fichier : une lecture ajoutee ou reintroduite le fait echouer, une lecture
  retiree aussi (la table descend).
- Sections de duree (D7.4 ; feuilles ; `TestCareerRepo_Annuaire_SectionsDeDuree`) : `top_encounters`,
  `top_encounters_annuaire`, `rivals` (2 appels), `rivals_annuaire`, `encounters`, `encounters_annuaire`,
  `compare_local_stats`, `compare_local_stats_annuaire` (Comparer : 2 appels par requete, A puis B).
- Ecarts a la lettre, et pourquoi : (1) l'annuaire reste dans `squad_repo_annuaire.go` : `nommerLignes`
  est deja commun au paquet, et un deplacement vers `annuaire_repo.go` laissait six references de
  fichier dans des sources Escouade hors perimetre ; seul son en-tete change (lecteurs, mesures L7), sa
  ligne DEBUG garde son nom (`squad_annuaire`) ; (2) les « matchs de la lecture » d'une lecture agregee
  de la Carriere (et de Comparer) sont ceux de l'historique du joueur (`QMatchsDuJoueurTpl` = le
  `my_history` de Q26, exclusion Campagne comprise ; 2-26 ms) ; les rivaux y ajoutent le match du duel
  par l'acces `match` existant de l'annuaire — aucune ligne de l'annuaire modifiee ; (3) L7.3 depasse la
  liste de fichiers du plan, comme la consigne le demande ; Q10 vit dans `queries_match.go` mais son
  lecteur est `career_repo_encounters.go` ; (4) refactors imposes par la taille des fonctions touchees :
  GetTopEncountersGlobal 102 -> 29 L, GetLocalStats 82 -> 54 L ; (5) Comparer : la jointure
  `xuid_aliases` part avec la vue (elle ne servait que le nom ; xuid PRIMARY KEY : les compteurs ne
  pouvaient pas en etre gonfles).
- Mutations jouees (toutes rouges puis restaurees, `cmp` a l'appui) : jointure reintroduite dans Q26,
  Q27, Q10, Comparer (ratchet) ; rivaux sans le match du duel (x_kfseul masque) ; annuaire sans les matchs
  de l'historique ; rencontres, rivaux, Q10, Comparer jamais nommes ; nemesis hors de l'annuaire ;
  Comparer nomme sur l'historique du joueur du repo au lieu de celui du compare ; jeton Campagne retire
  de `QMatchsDuJoueurTpl` (garde-rail structurel) ; sections `rivals`, `top_encounters_annuaire`,
  `encounters`, `compare_local_stats_annuaire` retirees. Deux mutations d'abord VERTES, tests rendus
  discriminants : « rivaux sans match du duel » (x_kfseul partageait un match avec un autre xuid sans
  nom : son duel est desormais dans ma3, ou aucun autre ne joue) et « Comparer sur l'historique du
  joueur du repo » (x_autre, jamais croise par lui, ajoute).
- Dette : aucun fichier touche ne grossit (queries_career_encounters.go 568 = 568, queries_match.go
  623 -> 620, geles au-dela de 500 L ; les autres sous 500) ; aucune fonction nouvelle au-dela de 80 L ;
  golangci `--new-from-rev=2beeba665` : 1 issue (prealloc, `projeterRivaux`) corrigee en aa2ec4880,
  puis 0 (avec et sans le tag integration).
- Gate (code final aa2ec4880) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ; `go vet ./...`
  0 ; `go test ./internal/service/... ./internal/platform/duckdb/... ./internal/analysis/...
  ./internal/api/...` 0 (32 paquets ok, aucun `--- FAIL:`) ; `go test -tags=integration -p 1
  ./internal/platform/duckdb/...` 0 (5 paquets ok) ; `go test ./internal/archlint/` 0 ; les 8
  garde-rails ART / legacy de `internal/sync` verts ; aucun test renomme ni supprime (7 ajoutes) :
  baseline JSONL inchangee ; l'artefact `data/titles/halo_5/warehouse/metadata.duckdb` recree par
  `./internal/api/...` dans le worktree (decouverte (6) de L1) retire.
- Decouvertes (consignees, non traitees) : (1) Q26 et Q27 paient la fenetre `_latest` du kill-feed sur
  3,95 M lignes (0,8-1,1 s par lecture) ; une liste CONSTANTE des matchs du joueur poussee sous la
  fenetre la ramene a 0,51-0,56 s (JGtm) / 0,21-0,24 s (Nuzzles), mesure ; seule une vue materialisee
  changerait l'ordre de grandeur (cf. L5a (6)) ; (2) GetRivals lit deux fois le meme agregat (ORDER BY
  deaths puis frags) : une lecture et deux tris en Go economiseraient une fenetre (~1 s par requete) ;
  (3) la vue match evalue la vue des noms quatre fois par ouverture, plus ResolveGamertags (cf. (b),
  (c)) ; (4) `EncounterStatsRaw.FirstSeen` / `LastSeenAt` ne sont jamais poses par
  GetTopEncountersGlobal (Q26 lit `first_seen_at` sans le rendre) : les badges temporels (recrue,
  ancien) des rencontres de la Carriere ne peuvent pas s'allumer — preexistant, inchange ; (5) Q10 (coupe
  LIMIT 50) et Q23 (`ORDER BY count_together DESC`) n'ont pas de departage : ensemble et ordre des ex
  aequo arbitraires d'une execution a l'autre (comme Q29, decouverte (4) de L2) ; (6) Q10 n'ecarte pas
  les bots : 93 des 224 lignes servies aux cinq joueurs sont des bots (JGtm : « 343 Chilies » 4e), que le
  selecteur de composition de l'onglet Tactique peut proposer comme coequipiers ; (7) la heatmap
  Relations (Q29) rend un ordre non deterministe (ORDER BY xuid, heure : le jour ne departage pas) ;
  (8) Relations : 2 973 des 7 450 joueurs recurrents de Nuzzles n'ont ni alias ni gamertag de
  participant (import sans gamertag ?) — la vue comme l'annuaire les masquent.
## 9 quater. L8 — Escouade : departages stables et capability sur la page (Go)

Ajoute le 2026-09-23 a la fusion de L4b : decouverte (4) du lot L2, decouvertes (1) et (2) du lot
L4b, et decision « departage stable des badges ex aequo » en attente au journal §12.

Decisions tranchees (superviseur) :
- D8.1 `Q29TopTeammatesSharedTpl` (`internal/platform/duckdb/queries_squad.go`, `ORDER BY
  games_together DESC LIMIT 50`) : ordre deterministe `ORDER BY games_together DESC,
  wins_together DESC, p2.xuid ASC`. Motif : sans departage, la coupe du LIMIT 50 parmi les ex
  aequo varie d'une lecture a l'autre, donc la liste des coequipiers connus a exclure
  (composition exacte) varie aussi, et les sessions / compteurs de la page Escouade et de la
  lecture legere ne sont pas reproductibles (4 pages sur 5 differentes sur donnees reelles, L4b).
- D8.2 `Q32bMainTeamParticipantsTemplate` (meme fichier, sans ORDER BY) : `ORDER BY p.match_id,
  p.xuid`. Les porteurs de badges ex aequo (`topKiller`, `falseBrother`, `silentHero`, premier ex
  aequo dans l'ordre des lignes) dependaient du plan d'execution DuckDB ; la regle devient « le
  plus petit xuid parmi les ex aequo », stable, documentee en commentaire au-dessus du template
  et ici (l'utilisateur pourra choisir une autre regle plus tard). Autres templates de
  `queries_squad.go` consommes en « premier ex aequo » et sans ORDER BY : traites de meme si
  c'est mecanique, sinon consignes.
- D8.3 `internal/api/handlers/teammates.go` : POST `/pages/teammates` passe par
  `MapCapabilityError(ctx, err, "teammates.page")` AVANT `mapServiceError`, comme la route legere
  `GET /pages/teammates/sessions` (L4b) : 503 `capability_not_supported` au lieu de 500 quand
  `match.history` manque. Test handler : capability absente = 503.
- D8.4 Tests : (a) deux lectures consecutives de Q29 et de Q32b sur une base `:memory:` avec des
  ex aequo construits rendent le MEME resultat dans le MEME ordre ; la mutation « ORDER BY
  retire » rend le test rouge de facon fiable (ordre d'insertion different de l'ordre voulu) ;
  (b) tests existants `squad_repo*_test.go` / `teammates_*_test.go` adaptes s'ils figeaient un
  ordre ou un porteur (chaque fixture changee, au journal) ; (c) golden JSON regenere et
  explique s'il change.

Perimetre : `internal/platform/duckdb/queries_squad.go`, `internal/platform/duckdb/squad_repo*.go`
(tests), `internal/api/handlers/teammates.go` (+ tests), `internal/service/teammates/*_test.go`
(fixtures seulement), ce plan (§9 quater). Interdits (L7 en parallele) :
`queries_career_encounters.go`, `career_repo*.go`.

Items :
- [x] L8.1 Q29, ordre total — `queries_squad.go:60` (`ORDER BY games_together DESC,
  wins_together DESC, p2.xuid ASC`), motif en commentaire `:28-32` ; preuve
  `squad_repo_departages_test.go:95` (`TestSquadRepo_Q29_DepartageStable`)
- [x] L8.2 Q32b + autres templates — `queries_squad.go:328` (`ORDER BY p.match_id, p.xuid`), regle
  des ex aequo en commentaire `:306-313` ; preuve `squad_repo_departages_test.go:189`
  (`TestSquadRepo_Q32b_OrdreStableEtPorteursExAequo`). Recensement des 12 constantes de
  `queries_squad.go` : trois autres sans ORDER BY, aucune consommee en « premier ex aequo »,
  laissees telles quelles — `QSquadExpectedWinProbTpl` (`:8`, une ligne par match, lue dans une
  map par `loadExpectedWinProbs`, `squad_repo.go:248`), `Q42MapStatsForSquadSharedTpl` et son
  fragment `Q42MapStatsSquadExtraExclusionFrag` (`:382`, `:406`, agreges par carte dans une map,
  `squad_repo_mapstats.go:44-71`, seul lecteur `LoadMapStatsForSquad`)
- [x] L8.3 MapCapabilityError — `handlers/teammates.go:104` (sonde `teammates.page`, avant
  `mapServiceError`), doc `:87-90` ; test `teammates_test.go:109`
  (`TestTeammatesHandler_CapabilityAbsente_503`)
- [x] L8.4 tests — (a) les trois tests ci-dessus, chacun livre dans le commit du changement qu'il
  prouve ; (b) aucun test existant a adapter, aucune fixture modifiee ; (c) aucun golden
  concerne (detail au journal)

Gate : `gofmt -l ./internal ./cmd` ; `go build ./...` ; `go vet ./...` ; `go test
./internal/platform/duckdb/... ./internal/service/teammates/... ./internal/api/handlers/...` ;
`go test -tags=integration -p 1 ./internal/platform/duckdb/...` ; golangci-lint
`--new-from-rev=436dc7200` sur les paquets touches.

Journal du lot (2026-09-23, branche `feat/perf-l8` depuis `feat/perf-chargements` 436dc7200,
executeur Opus ; commits 077269adc L8.1, beece4926 L8.2, 38fd541ef L8.3, puis ce journal) :

- Q29 : les trois lecteurs (`TeammatesService.GetPage`, `TeammatesService.CompositionSessions`,
  `SquadService`) recoivent le meme top 50 dans le meme ordre a chaque lecture ; la liste
  deroulante, l'extraPool de la composition exacte (`buildExtraPoolXUIDs`), la resolution d'un
  coequipier par gamertag (premier du top 50 sans casse) et `resolveFriendXUIDs` (map gamertag ->
  xuid) ne dependent plus du plan. Seuls changent, a egalite de matchs communs, l'ordre (victoires
  puis xuid) et, a la coupe du LIMIT 50, l'appartenance (les plus petits xuids entrent).
- Q32b : l'ordre des lignes arrive intact jusqu'aux badges (`nommerLignes` nomme en place,
  `squad_repo_annuaire.go:57-76` ; `allyByMatch[...] = append(...)`,
  `teammates_squad_charts_impact_events.go:82` ; snaps dans l'ordre, `:132`) et `topKiller`,
  `silentHero`, `falseBrother` (`analysis/match_impact.go:402`, `:426`, `:463`) retiennent le
  premier a egalite. Regle en vigueur : le plus petit xuid parmi les ex aequo, ordre binaire de
  la chaine (aucune `default_collation` dans le depot). Les autres lecteurs de Q32b sont
  insensibles a l'ordre (`buildMainTeamXUIDSet`, ensemble par match) ou y gagnent un nom
  d'affichage stable (accueil, `sessionCoreTeammates` : premier gamertag vu par cle minuscule).
- Templates sans ORDER BY laisses tels quels (cf. L8.2) : seul effet d'ordre possible, le dernier
  bit de la moyenne flottante `PerfAvg` de Q42, sommee dans l'ordre des lignes (theorique, pas un
  « premier ex aequo »). Templates a ORDER BY PARTIEL (Q32, Q32c) : decouverte (2).
- Effet sur donnees reelles : non mesure (bases de `data/` interdites a ce lot, serveur de mesure
  en cours). Attendu : les porteurs de badges ex aequo des 33 matchs releves au lot L2 (ecart 3)
  deviennent le plus petit xuid, aucune statistique ne change, et la page Escouade comme la
  lecture legere rendent le meme resultat d'une requete a l'autre sur une meme base. Cout non
  mesure : Q29 etait deja un tri borne (deux cles de plus), Q32b trie ses lignes (une par allie
  et par match).
- Tests (D8.4 a) : `squad_repo_departages_test.go` (nouveau, `//go:build integration`, base
  `:memory:` de `newTestPlayerDB`, vraies methodes du repo, aucune DDL recopiee).
  `TestSquadRepo_Q29_DepartageStable` : 54 coequipiers sur six matchs « avec amis », groupes a
  egalite inseres dans un ordre qui n'est ni l'ordre voulu ni son inverse (45 a 3 matchs / 3
  victoires par xuid decroissant ; 3 a 3 matchs dont l'ordre des xuids est l'inverse de celui des
  victoires ; 5 a 2 / 1 inseres 5, 3, 1, 4, 2, coupes par le LIMIT 50 au milieu du groupe) ; deux
  lectures egales, liste attendue ecrite en clair. `TestSquadRepo_Q32b_OrdreStableEtPorteursExAequo` :
  trois matchs inseres mb2, mb3, mb1, dans chacun trois allies a egalite sur le critere d'un badge
  (le plus petit xuid ni premier ni dernier insere) et un adversaire ; deux lectures egales, ordre
  (match_id, xuid) ecrit en clair, et la matrice rejouee (`ComputeMatchImpactFull` sur les lignes
  dans leur ordre, comme `buildSquadImpactMatrix`) donne Heros silencieux, Bourreau et Faux-frere
  ex aequo au plus petit xuid (Bourreau sans egalite en temoin). L8.3 :
  `TestTeammatesHandler_CapabilityAbsente_503`.
- Tests existants (D8.4 b) : aucun a adapter, aucune fixture modifiee — aucun ne figeait un ordre
  ni un porteur (tests Q32b existants compares par cle triee, `squad_repo_main_team_test.go:81-89` ;
  tests d'annuaire : un nom par ligne ; tests du service teammates : depots simules, l'ordre est
  celui que le test fournit). Goldens (D8.4 c) : aucun concerne, aucun fichier golden ni
  `testdata` de l'Escouade dans le depot, les tests de page passent par des depots simules.
- Mutations jouees (toutes restaurees, `cmp` a l'appui) : Q29 — ORDER BY d'avant L8
  (`games_together DESC` seul) : rouge 10/10, et deux lectures consecutives differentes 9/10
  (l'instabilite de production reproduite sur 54 lignes) ; sans `p2.xuid ASC` : rouge 10/10 ;
  sans `wins_together DESC` : rouge 5/5, deterministe (position 45). Q32b — ORDER BY retire :
  rouge 10/10 (ordre et les trois porteurs ex aequo) ; `ORDER BY p.match_id` seul : rouge 10/10.
  L8.3 — bloc `MapCapabilityError` retire : rouge (500 `teammates_error`, marque retryable).
- Gates (code final 38fd541ef) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ;
  `go vet ./...` 0 ; `go test ./internal/platform/duckdb/... ./internal/service/teammates/...
  ./internal/api/handlers/...` 0 (6 paquets, aucun `--- FAIL:`) ; `go test -tags=integration -p 1
  ./internal/platform/duckdb/...` 0 (5 paquets, aucun `--- FAIL:`) ; golangci-lint 2.12.2
  `--new-from-rev=436dc7200` sur `platform/duckdb/...` et `api/handlers/...` : 0 issue, et avec
  `--build-tags=integration` sur `platform/duckdb/...` : 0 issue. Hors gate : `go vet
  -tags=integration` des trois arbres 0 ; `go test ./internal/archlint/` ROUGE, preexistant
  (decouverte 1). Aucun test renomme ni supprime (3 ajoutes) : baseline JSONL inchangee.
  Tailles : `queries_squad.go` 401 -> 417 L, `teammates.go` 142 -> 150 L, `teammates_test.go`
  119 -> 143 L, test nouveau 244 L.
- Decouvertes (non traitees, hors perimetre) :
  (1) `internal/archlint` ROUGE des la base de campagne 436dc7200 : `TestNoNewFrenchLabelLiteral`,
  `api/handlers/teammates.go` porte 3 litteraux accentues pour 1 alloue ; les deux de trop sont les
  tags `doc:` des parametres `teammates` et `exact` de la route legere (`teammates.go:69-70`),
  ajoutes par L4b (87b53622e) et publies dans `openapi.yaml` ; ce lot n'ajoute aucun litteral
  accentue. A reparer avant la CI de cloture (C.3, `go test ./...` avec `-tags=integration`) : le
  ratchet ne se remonte jamais, la correction touche le texte des deux tags donc le contrat et
  les types generes.
  (2) Q32 (`ORDER BY he.match_id, he.time_ms`) et Q32c (`ORDER BY kv.match_id, kv.time_ms`) : ordre
  PARTIEL ; a temps egal, Premier sang et Premiere victime (`firstByTime`), Finisseur et Boulet
  (`lastByTimeFiltered`), Top Gun (tri stable puis premier au seuil) retiennent le premier
  evenement dans l'ordre des lignes, que choisit le plan. Meme classe que Q32b mais hors de la
  lettre de D8.2 (templates AVEC ORDER BY) ; un departage (xuid, event_type) changerait des
  porteurs : a decider.
  (3) `analysis.slowestFirstKillerWithTime` (Touriste, `match_impact.go:374`) parcourt une map : a
  egalite de premier frag, le porteur change d'une execution a l'autre quel que soit l'ordre SQL
  (iteration des maps Go aleatoire) ; `analysis` hors perimetre.
  (4) La description OpenAPI de POST `/pages/teammates` ne mentionne pas le 503
  `capability_not_supported` (celle de GET `/pages/teammates/sessions` le fait) ; le statut 503
  est deja documente (reponse partagee `DbBusy`), seul le texte manque.
  (5) `teammates_service_composition_sessions.go:146-157` : `resolveGamertagFallback` se dit « meme
  repli que la requete SQL Q32b », perime depuis L2 (Q32b ne porte plus de gamertag), et recopie
  la regle de `analysis.MaskedXuidLabel` (en octets, la ou celle-ci compte des runes).
  (6) `squad_repo.go:472-475` : le commentaire de `LoadMainTeamParticipants` est detache en fin de
  fichier ; la fonction (`squad_repo_synthesis.go:15`) n'en a pas.
- Complement superviseur (2026-09-23, commit « perf(l8): archlint — tags doc de la route legere
  sans accent ») : decouverte (1) CORRIGEE. Les deux tags `doc:` de `teammates.go:69-70` sont
  reformules sans caractere accentue, sens identique (« Gamertags des membres de la composition,
  joints par des virgules. Absent ou vide : sessions escouade du joueur principal. » ; « Option
  composition exacte (filter_exact_composition de POST /pages/teammates). Absent : false. ») ;
  `api/openapi.yaml` (4 descriptions) et `apps/web/src/lib/api/generated.ts` (2 commentaires)
  regeneres par les commandes documentees (`go run ./cmd/openapi-gen`, `npm run generate-types`,
  openapi-typescript 7.13.0). Seul litteral accentue restant du fichier : le resume de la page
  (`teammates.go:48`, « Analyse coequipiers » accentue), le 1 que l'allowlist compte deja, non
  modifie. Controles : `openapi-gen -check` 0 et `check-generated-types-fresh.mjs` 0 (avant et
  apres regeneration) ; `go test ./internal/archlint/ ./internal/api/handlers/` 0 ;
  `TestOpenAPIYAMLIsUpToDate` PASS et `go test ./internal/api/` 0 ; aucun `--- FAIL:`.

## 9 quinquies. L9-go — correctifs de la revue adversariale (Go)

Ajoute le 2026-09-23 a la cloture (C.2) : revue adversariale du diff cumule, fan-out aveugle
(revues A, B, D pour le Go ; le lot L9-web traite la revue C en parallele). Decisions tranchees
par le superviseur, dans l'ordre d'execution :

- D9.1 (revue B, P0) : le cache des lectures joueur ne stocke jamais un chargement degrade — ni
  requete terminee pendant le chargement, ni etape best-effort en echec ; un chargement degrade
  n'est ni stocke ni partage aux requetes en attente (elles rechargent). Couvre le P2 de la revue
  D (« un echec ponctuel de metadata est mis en cache 60 s »).
- D9.2 (revue B, P1) : rafale LUSR bornee — zero ecrivain en regime stationnaire conserve (D6.1) ;
  ecrivain rendu puis repris tous les K = 50 matchs ou des 2 s de detention, reprise du reste de
  la file dans le meme cycle ; journal INFO par rafale (`candidates`, `new`, `bursts`).
- D9.3 (revue B, P2) : (a) saisons — une fin de contexte n'est jamais memorisee comme echec
  Waypoint ; (b) db_profiles.json — un instantane lu dans la fenetre de mefiance n'est jamais
  garde ; (c) journaux en DEBUG quand la requete a pris fin (filters_service, data issues
  Escouade, lireComposition).
- D9.4 (revue A, P1) : matrice d'impact — appartenance a l'escouade par xuid.
- D9.5 (revue A, P2) : lecture legere — sous `exact=true`, un echec de Q32b est une erreur
  (503 / 500 par mapServiceError), jamais un 200 au roster non filtre.
- D9.6 (revue D, P1) : Carriere — amis des rencontres resolus sans `v_gamertag_lookup`, en une
  lecture (registre des profils, puis `xuid_aliases`, puis participants de l'historique, ILIKE).
- D9.7 (revue D, P2) : session piquee sans match = aucun match ; `session_id` accepte.
- D9.8 (revue D, P2) : ordres totaux de Q32, Q32c et Q10.
- D9.9 (revue D, P2) : ratchet de l'annuaire sur tout `internal/`, identifiant nu, exceptions
  datees.
- D9.10 (revue D, P2) : `filtersCacheKey` — chaque champ de `port.PlayerMatchFilters` change la
  cle (test par reflexion).

Perimetre : les fichiers cites par les decisions et leurs tests ; ce plan (§9 quinquies). Hors
perimetre : `apps/web` (lot L9-web), `.ai/thought_log.md` (texte de l'entree au rapport).
Invariants : ART (INSERT-only, `_latest`, aucune allowlist), title-agnostic, `slog.*Context`,
fichiers <= 500 L / fonctions <= 80 L (dette gelee).

Items :
- [x] L9.1 cache sans chargement degrade (D9.1) — `platform/duckdb/player_read_cache.go:192`
  (`fetch` pose une sonde de degradation sur le contexte du chargement ; valeur degradee = rendue
  avec `errDegraded` (:216), jamais stockee, marqueur `<cache>_degraded`), `:176` (une requete
  greffee sur un vol degrade ou annule recharge pour son compte ; la porteuse recoit ses lignes
  sans erreur, best-effort comme sans cache), `:237` `noteDegraded`, `:249` `bestEffortFailed`,
  `:259` (requete terminee = degrade d'office). Sites consignes : `filters_repo_asset_names.go:95`,
  `filters_repo_fr_cascades.go:95,102,115,146,161` (deux lectures jusque-la avalees en silence,
  `rows.Err` verifie), `mode_name_tr.go:101`, `match_history_fr_translations.go:224,239`,
  `home_repo_translations.go:215`, `home_repo_translations_canonical.go:109,143` (erreur des
  modes FR jusque-la ignoree), `player_matches_adapter.go:57`. Tests
  `player_read_cache_test.go:323,360,390,433` ; integration `player_read_cache_degraded_test.go:73`
  (panne de metadata, filtres), `:97` (panne, historique enrichi), `:161` (annulations reparties,
  reprise durable de `TestRevB_FilterRowsCachePoisonedByCancelledRequest`) ; commit f4d8c2af5
- [x] L9.2 rafales LUSR bornees (D9.2) — `sync/skill/skill_v2_shared_access.go:75`
  (`defaultLUSRBurstLimits` : 50 matchs, 2 s = `sharedprovider.defaultRWHoldWatchdog`), `:97`
  (`runWriterBursts`, `heldGroups` traverse les rafales :98, INFO `lusr_v2: rafale bornee` :111),
  `:144` (`processShadowBurst` borne) ; `skill_v2_shadow.go:146` ; `skill_v2_watermark.go:146`
  (`bursts` dans `lusr_v2: rafale terminee`) ; tests `skill_v2_bounded_bursts_test.go:70` (300
  candidats), `:123` (horloge pilotee), `:162` (parite en rafales de 2) ; parite existante verte ;
  docs `SYNC_GUIDE.md` EN + FR ; gate integration sync + persist code 0 ; commit f6edd85a3
- [x] L9.3 (a) saisons, (b) db_profiles, (c) journaux (D9.3) — (a) `service/seasons_catalog.go:307`,
  test `seasons_catalog_cache_test.go:307` (annulation, echeance) ; (b) `config/config_players.go:95`,
  test `config_players_cache_test.go:135` (scenario de la revue) ; (c)
  `observability/level.go:20` (`LevelUnlessCanceled`, source unique gardee par
  `level_test.go:45`), `service/filters_service.go:117,163`,
  `teammates/teammates_data_issues.go:31`, `teammates_service_composition_legere.go:136`, test
  `teammates_log_test.go:212` ; commits 778efcb9e, 3a7c21d28 (message DEBUG sans accent, cf. journal)
- [x] L9.4 matrice d'impact par xuid (D9.4) — `teammates/teammates_squad_charts_impact_events.go:94`
  (`resolveSquadScope(...).gtByXUID`, lignes = noms choisis), `teammates_service_sections.go:129` ;
  test `teammates_impact_matrix_membership_test.go:25` (« Madina » / « madina » / « MADINA ») ;
  commit 3b89ac6b2
- [x] L9.5 lecture legere : Q32b illisible = erreur (D9.5) — `teammates_service_intersect.go:346`
  (`lireEquipeAlliee`, seul appel de Q32b), `teammates_service_composition_legere.go:192,206` ;
  test `teammates_service_composition_legere_test.go:418` (`TestRevA_LegereDegradeeSansSignal`
  inverse) ; cas de parite « Q32b en echec : roster non filtre » retire (divergence voulue) ;
  handler inchange (mapServiceError : 503 / 500, `page_service_errors_test.go`) ; commit 5cb42f7ae
- [x] L9.6 Carriere, amis sans la vue (D9.6) — `platform/duckdb/career_repo_friends.go:36`
  (`QAmisParGamertagTpl`), `:61` (`ResolveFriendXUIDs`, une lecture, section `career_friends`) ;
  `service/career_service_encounters.go:189` (registre d'abord, puis la lecture unique),
  `career_service.go:148` (`WithFriendXUIDSources`) ; `api/wire/registry_career.go:73,118` ;
  libelle du ratchet corrige (`annuaire_ratchet_test.go:49`) ; tests
  `career_service_friends_test.go:56,73,86`, integration `career_repo_friends_test.go:29,72` ;
  chrono au journal ; commits f2a4757c1, 54a2244ec (requete enrolee au garde-rail des seeds, cf. journal)
- [x] L9.7 session piquee sans match (D9.7) — `teammates_service_briefing.go:356` (contrat ecrit),
  consommateurs `teammates_service_kpis.go:96`, `teammates_service_briefing.go:68`,
  `teammates_squad_charts_intensity_perminute.go:250` (`!= nil`) ; `session_id` :
  `teammates_service_briefing.go:312` ; tests `teammates_session_filter_test.go:328` (scenario 5),
  `:355`, `teammates_filter_helpers_test.go:141` ; test du relecteur vert ; commit 329c21663
- [x] L9.8 ordres totaux (D9.8) — `queries_squad.go:202` (Q32), `:243` (Q32c),
  `queries_match.go:29` (Q10), regle en commentaire au-dessus de chaque gabarit ; tests
  `squad_repo_departages_test.go:265,302,338` ; commit 145abc97b
- [x] L9.9 ratchet de l'annuaire (D9.9) — `annuaire_ratchet_test.go:38` (table datee : DDL et
  migrations, lectures consignees L7, deux lectures du sync killcollector), `:105` (parcours de
  tout `internal/`, identifiant nu dans tout litteral) ; commit 58ae43598
- [x] L9.10 cle du cache par reflexion (D9.10) — `player_matches_cache_test.go:306` ; commit
  31851e885

Gate : `gofmt -l ./internal ./cmd` ; `go build ./...` ; `go vet ./...` ; `go test ./...` ;
`go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/...` ;
`go run ./cmd/openapi-gen -check` ; `golangci-lint run --new-from-rev=8830aebe2 ./...`.

Journal du lot (2026-09-23, branche `feat/perf-l9go` depuis `feat/perf-chargements` 8830aebe2,
executeur Opus ; un commit par item, puis ce journal) :

- Chrono du point 6 (sonde temporaire compilee hors de l'arbre, jamais commitee : binaire « avant »
  construit sur 5cb42f7ae, « apres » sur f2a4757c1 ; copie de `shared_matches_v2.duckdb` du
  23/09 17:25 — celle de la revue D, recopiee dans le scratchpad du lot —, copies de
  `db_profiles.json` et `player_friends.json` ; `OpenReadOnly`, 2 threads, 512 Mo ;
  `CareerService.GetTopEncounters` de bout en bout, cablage de `wire.Career` reproduit, 3 tours) :
  avant 6,6-19,3 s (JGtm 14,3-19,3 s, XxDaemonGamerxX 6,6-13,0 s,
  Madina97294 7,3-8,9 s, Chocoboflor 7,2-7,7 s ; la vue seule : 1,9-4,8 s PAR AMI, trois amis
  par joueur) ; apres 0,79-0,98 s (registre + lecture) ; la lecture seule des amis, sans
  registre : 7,8-20,7 ms, et le service 0,80-0,98 s. Parite : 12 amis sur 12 resolus au meme
  xuid par le registre, la lecture et la vue ; 40 rencontres servies (4 joueurs x 10) identiques
  a l'octet avant / apres. Ce qui reste (~0,8 s) : Q26 (fenetre `_latest` du kill-feed,
  decouvertes (1) et (2) du lot L7).
- Precisions d'implementation (aucune decision rouverte) : (1) L9.1 — le signal de degradation
  des chargeurs est une SONDE posee par le cache sur le contexte du chargement, renseignee aux
  sites best-effort (`noteDegraded` / `bestEffortFailed`) ; ces sites sont des helpers partages
  par des lecteurs non caches (historique de match, medias, accueil) dont la signature ne change
  pas (sans sonde : sans effet). L'erreur typee `errDegraded` porte le verdict de `fetch` a
  `load` ; non exportee, elle ne sort jamais du cache. Une table absente (base non migree) n'est
  pas une degradation (meme resultat a chaque lecture). (2) L9.1 — les cascades FR carte /
  selection sont extraites sans changement dans `filters_repo_fr_cascades.go` :
  `filters_repo_asset_names.go` (491 L) aurait depasse 500 L. (3) L9.2 — la duree se teste APRES
  chaque match : une rafale peut depasser 2 s de la duree d'un match ; mesure sur la base de test
  : 6 rafales de 50 matchs, 374-401 ms chacune. (4) L9.3 (c) — cinq sites choisissent le niveau
  par la vie de la requete : helper unique `observability.LevelUnlessCanceled` + garde-rail
  (CLAUDE.md n6) plutot que cinq copies du predicat. (5) L9.4 — `mainGamertag` retire de la
  signature (le joueur principal est `s.gamertag`, seul appelant), `teammates` ajoute : six
  parametres comme avant. (6) L9.6 — le registre est `cfg.LoadPlayers` (relu seulement quand le
  fichier change, L5b) ; tous les amis configures de la base de production sont des profils
  suivis : en production, la resolution ne lit plus la base du tout. Ecarts nommes avec la vue
  (tests `career_repo_friends_test.go:72`) : un ancien nom porte dans l'historique du joueur est
  reconnu ; un nom porte seulement hors de son historique ne l'est plus (un joueur jamais croise
  ne peut pas figurer dans ses rencontres) ; homonymes : niveau le plus fort puis plus petit
  xuid (la vue : au hasard). Le niveau kill-feed de la vue n'est pas repris (la decision enumere
  trois niveaux). (7) L9.7 — le `session_id` se lit sur les lignes canoniques
  (`Enrichment.SessionID`), `SynthesisMatchRow` ne portant que le libelle. (8) L9.9 — un nom
  d'etape de migration qui contient l'identifiant sans le nommer seul
  (`upgrade_v_gamertag_lookup_...`) n'est pas compte (`\bv_gamertag_lookup\b`).
- Mutations jouees (toutes rouges puis restaurees, `cmp` a l'appui) : L9.1 — stockage
  inconditionnel (5 tests rouges dont les annulations reparties : 36 empoisonnements / 36),
  partage aux requetes en attente, sonde muette, fin de requete ignoree, sites de consignation
  retires ; L9.2 — `heldGroups` par rafale (parite rouge : 4 traites contre 3), borne de matchs
  retiree (2 rafales au lieu de 6), borne de duree retiree, pas de reprise ; L9.3 — fin de
  contexte memorisee, instantane stocke dans la fenetre, niveau inconditionnel, copie en ligne du
  predicat ; L9.4 — appartenance par nom ; L9.5 — degradation silencieuse, cause perdue (`%v`) ;
  L9.6 — registre ignore, niveau participants retire, comparaison sensible a la casse, bots admis ;
  L9.7 — consommateur `len() > 0`, `session_id` ignore ; L9.8 — ORDER BY d'avant sur Q32, Q32c,
  Q10 (rouge 5/5 chacun) ; L9.9 — lecture ajoutee, gabarit assemble dans un autre paquet ;
  L9.10 — `OrderBy` retire de la cle, champ ajoute au type sans entrer dans la cle.
- Tests des relecteurs rejoues par overlay sur le code final (hors de l'arbre) :
  `TestRevB_FilterRowsCachePoisonedByCancelledRequest` (400 annulations, 0 empoisonnement, 73 s),
  `TestRevB_CancelledFetchMemorisedAsFailure`, `TestRevB_RacyWindowSnapshotTrustedAfterWindow`,
  `TestRevD_SessionPiqueeSansMatch` : verts ; `TestRevA_MatriceImpact_NomQ32bDifferentDuNomChoisi`
  et `TestRevA_LegereDegradeeSansSignal` remplaces par leurs versions durables (signature de la
  matrice changee ; attente inversee pour la lecture legere).
- Dette : aucune fonction nouvelle au-dela de 80 L ni de 5 parametres (`buildSquadImpactMatrix`
  garde ses six) ; `buildSquadImpactMatrix` raccourcie ; fichiers nouveaux sous 500 L. Deux
  fichiers deja au-dela de 500 L grossissent : `service/filters_service.go` 613 -> 614 (ligne
  d'import du helper de niveau) et `platform/duckdb/queries_match.go` 620 -> 622 (deux lignes
  de documentation de l'ordre total demandees par D9.8) ; `filters_repo_asset_names.go`
  491 -> 305 (extraction).
- Gate (code final 54a2244ec) : `gofmt -l ./internal ./cmd` vide ; `go build ./...` 0 ;
  `go vet ./...` 0 ; `go test ./...` 0 (190 paquets ok, 152 sans test, aucun `--- FAIL:`,
  254 s). La premiere passe (code 31851e885) avait releve deux rouges, corrigees : ratchet
  `TestNoNewFrenchLabelLiteral` (le message DEBUG de la fin de contexte, dans
  `seasons_catalog.go`, passe par `c.logger`, hors de l'exclusion `slog.*` : reformule sans
  accent, 3a7c21d28) et cliquet `TestSeedParityEnrollmentRatchet` (`mp.gamertag` de
  `QAmisParGamertagTpl` couverte par aucune requete enrolee : requete enrolee sur
  `seedPlayerSchema`, seuil d'extraction propre dans `seedParityMinRefs`, 54a2244ec).
  `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...
  ./internal/platform/duckdb/...` 0 (17 paquets ok, 648 s, aucun `--- FAIL:`) ; a L9.2,
  sync + persist seuls : 0 (301 s). `go run ./cmd/openapi-gen -check` 0 (aucun handler
  touche). `golangci-lint run --new-from-rev=8830aebe2 ./...` : 0 issue ; avec
  `--build-tags=integration` sur `platform/duckdb` et `sync` : 0 ; `go vet -tags=integration`
  de duckdb, sync, persist, service : 0. Aucun test renomme ni supprime (le cas « Q32b en
  echec » retire est une ligne de la table d'un test de L4b, absent de la baseline JSONL) :
  baseline inchangee. Artefact `data/titles/halo_5/warehouse/metadata.duckdb` (recree par
  `./internal/api/...`, decouverte (6) de L1) retire de l'arbre du worktree.
- Decouvertes (consignees, non traitees) : (1) `service/seasons_catalog.go` : le singleflight
  partage aux requetes en attente le repli (TOML + base vide) d'une porteuse annulee pendant le
  fetch — non memorise desormais, mais servi a ces requetes-la (meme classe que D9.1) ;
  (2) `teammates_service_kpis.go:88` : `teammates_load_squad_matches_failed` reste un ERROR
  inconditionnel sur la page (la degradation qui suit, `issues.add`, passe en DEBUG sur
  annulation) ; (3) `home_repo_translations.go:33,53` (accueil legacy, non cache) :
  `mapImageURLs, _ :=` et `modeNamesFR, _ :=` avalent leurs erreurs sans journal ;
  (4) `filters_repo.go` `hasMVPlayerMatches` rend false sur erreur (repli sur `v_match_full`,
  memes lignes) sans journal ; (5) lecture legere : un coequipier dont Q30 echoue sort
  silencieusement de l'intersection (la page le dit dans `data_issues`, la reponse legere n'en
  porte pas) — meme classe que D9.5, hors de sa lettre ; (6) les deux lectures de la vue du
  sync killcollector (`credit_annuaire.go`, `roster.go`) materialisent la vue a chaque passe
  (cout non mesure) ; (7) saisons : un Waypoint lent (echeance du client HTTP) n'est plus
  memorise — sur base vide, chaque requete authentifiee retente (miroir voulu de
  `privacyFailure`) ; (8) le test d'annulations reparties est probabiliste (la fenetre des
  traductions est courte : 14 chargements degrades sur 150 sur ce poste) ; les tests
  deterministes (unitaires, panne de metadata) sont les gardes de D9.1 ; (9) `go test ./...`
  ecrit des rasters JSON sous `data/cache/replays/*/rasters/` de l'arbre (marqueA.json,
  writer1.json...) : un test du rejeu ecrit sous `data/` du depot au lieu d'un repertoire
  temporaire (meme classe que la decouverte (6) de L1) ; laisses en place (ignores par git).

## 9 sexies. L9-web — correctifs de la revue adversariale (web)

Ajouté le 2026-09-23 à la clôture (C.2, revue adversariale du diff cumulé) : constats web du
relecteur C, reproduits par ses tests `scratchpad/revC/*.revc.test.tsx` (verts = défaut présent).
Lot parallèle de L9-go (`apps/go-api`, hors de ce lot).

Décisions tranchées (superviseur), dans l'ordre :
- D9w.1 (P1) Lien profond vers une session ancienne écrasé par le snap : `useSquadSessionSelection.ts`
  pose la session du lien, puis `useSquadPageRequests.ts` et `squadPending.ts` décidaient un `snap`
  parce que `lastKnownLatestSessionId` (global au store) valait null ou celui d'une autre
  composition. Correction : au PREMIER ancrage d'une composition arrivée par lien profond, décider
  `none` et mémoriser la dernière session de la composition. Test : légère EN SUCCÈS
  `[S2 (3), S1 (2)]`, lien `?session=S1 (2)&teammates=Alice` : l'unique lourde part sur `S1 (2)`,
  le store reste sur `S1 (2)`.
- D9w.2 (P2) Cache léger périmé : `fresh` exige aussi `!light.isFetching` (pas de décision sur une
  donnée en cours de revalidation). Test : retour sur la page avec un cache léger périmé, une seule
  lourde.
- D9w.3 (P2) Sans coéquipier, label au suffixe périmé : la lourde n'attend la légère QUE si une
  session est pickée (réconciliation du suffixe avant la lourde). Test : store sur `S1 (2)`, légère
  `[S1 (4)]`, une seule lourde sur `S1 (4)`.
- D9w.4 (P2) 503 transitoire sur la légère (`retry: false`) : rejouer UNE fois sur 503, délai court,
  avant le repli. Test : 503 puis 200, une légère rejouée, une seule lourde.
- D9w.5 (P2) Capture globale des erreurs : ne rien enregistrer quand `err.name === 'AbortError'`
  (les annulations volontaires évinçaient les vraies erreurs du tampon de 5 entrées joint aux
  tickets). Test.
- D9w.6 (P2) Double requête au montage en dev (StrictMode + `signal`) : pas de changement de code ;
  explication au journal, pour que la mesure de clôture l'exclue.

Périmètre : `apps/web/src/features/squad/{useSquadSessionSelection.ts,useSquadPageRequests.ts,
squadPending.ts,queries.ts}` (+ tests), `apps/web/src/lib/global-capture/install.ts` (+ test), ce
plan (§9 sexies). Aucune string UI, aucune couleur, aucune clé de query nouvelles.

Items :
- [x] L9w.1 lien profond gardé au premier ancrage (D9w.1) — `features/squad/squadPending.ts:118`
  (`pinnedByDeepLink && stillValid` → `none`), champ `:76`, règle documentée `:84-88` ;
  `useSquadPageRequests.ts:185-196` (lien consommé au PREMIER ancrage, retenu seulement si sa
  composition est la courante), `:103` (transmis à la décision ; la dernière session de la
  composition est mémorisée par la branche « pas de snap » existante d'`appliquerAncrage`) ;
  `useSquadSessionSelection.ts:98` (`useSquadDeepLink`, capture au montage partagée, `:152`).
  Preuves : `SquadLayout.deeplink.test.tsx:146` (légère en succès `[S2 (3), S1 (2)]`, lien
  `S1 (2)` + Alice : une légère puis UNE lourde, sur `S1 (2)` dans les deux champs ; store sur
  `S1 (2)`, `lastKnownLatestSessionId` = `S2 (3)`, `isAutoSnappingToLatest` faux ; ancrage
  antérieur nul ou d'une autre composition), `:170` (session du lien inconnue de la composition :
  snap sur sa dernière AVANT l'unique lourde) ; `useSquadPageRequests.test.tsx:240` (après le
  premier ancrage, une nouvelle session arrivée re-snappe : règle à usage unique), `:263` (lien
  d'une autre composition : règles ordinaires) ; `squadPending.deeplink.test.ts:25-58` (6 cas purs,
  dont le témoin sans lien)
- [x] L9w.2 cache léger périmé (D9w.2) — `squadPending.ts:197` (`fresh` de la légère =
  `isEnabled && !isPlaceholderData && !isFetching`), `:146-149` (`LightQueryView`), motif
  `:179-185` ; `useSquadPageRequests.ts:135-159` (vue transmise) ; `isEnabled` en plus de la
  décision : journal (b). Preuves : `SquadLayout.requests.test.tsx:341` (cache léger
  `[S2 (3), S1 (2)]` vieux de 6 min, revalidation `[S3 (1), …]` : la légère PUIS une seule lourde,
  sur `S3 (1)` ; store vide, ou ancré sur `S2 (3)` par la visite précédente) ;
  `squadPending.test.ts:375` (en revalidation : lue, pas fraîche), `:384` (requête fermée : lue,
  pas fraîche)
- [x] L9w.3 sans coéquipier, session pickée (D9w.3) — `useSquadPageRequests.ts:131-132`
  (`attendLaLegere` = coéquipier OU session pickée). Preuves : `SquadLayout.requests.test.tsx:370`
  (store `S1 (2)`, légère `[S1 (4), S0 (1)]`, amis vides : une légère puis UNE lourde sur `S1 (4)`,
  dans les deux champs), `useSquadPageRequests.test.tsx:214` (légère retenue : aucune lourde tant
  qu'elle vole) ; témoins inchangés `useSquadPageRequests.test.tsx:196` et
  `SquadLayout.requests.test.tsx:210` (sans session pickée, la lourde n'attend pas)
- [x] L9w.4 503 rejoué une fois (D9w.4) — `features/squad/queries.ts:84`
  (`retryCompositionSessions` : `failureCount < 1` et statut 503), `:75-77` (`STATUS_DB_BUSY`,
  `COMPOSITION_SESSIONS_RETRY_DELAY_MS` = 500 ms), `:118-119`. Preuves :
  `SquadLayout.requests.test.tsx:386` (503 puis 200 : deux légères puis UNE lourde, sur `S2 (3)`),
  `:399` (503 deux fois : pas de troisième légère, repli L4a) ; `queries.test.tsx:132`, `:139`,
  `:146` (500, 502, erreur réseau : jamais rejoués)
- [x] L9w.5 capture globale sans les annulations (D9w.5) — `lib/global-capture/install.ts:162`
  (garde), `:179` (`isAbortError`, par le nom). Preuves : `install.test.ts:247` (rejet `AbortError`
  simulé : rien d'enregistré, erreur propagée), `:254` (fetch en vol annulé par son signal, fetch de
  l'environnement sous MSW : rien d'enregistré) ; la capture d'une vraie erreur réseau (`:235`,
  antérieur) reste verte
- [x] L9w.6 double requête au montage en dev (D9w.6) — aucun code ; explication au journal (f) ;
  témoin durable `SquadLayout.requests.test.tsx:418` (Escouade sous StrictMode : une légère et une
  lourde, aucune abandonnée)

Gate (depuis `apps/web`, `node_modules\.tmp` purgé avant) : `npm run typecheck` ; `npm run lint`
(0 erreur) ; `npx vitest run src/features/squad src/features/filters src/lib/query src/lib/api
src/lib/global-capture` ; `npx vitest run` (suite complète).

Journal du lot (2026-09-23, branche `feat/perf-l9web` depuis `feat/perf-chargements` 8830aebe2,
exécuteur Opus ; commits 54118fe6f L9w.1 à L9w.4 (+ témoin L9w.6), 7ecdc99bf L9w.5, puis ce journal) :

- Reproductions du relecteur C (configuration recopiée sur ce worktree) : 11 tests verts sur la
  base 8830aebe2 (défauts présents) ; sur le code final, les 5 qui reproduisent un défaut de ce lot
  sont ROUGES (défauts corrigés) et les 6 autres restent verts (comportement dev de StrictMode sur
  Carrière et Séries temporelles, témoin « avec coéquipier », Escouade sous StrictMode). Tests
  durables : 26 écrits AVANT les correctifs, 17 rouges sur la base (les défauts) et 9 témoins
  verts ; un 27e (légère fermée, `squadPending.test.ts:384`) avec le complément de D9w.2 (journal
  (b)), rouge sous la mutation M2b ; tous verts sur le code final.
- Précisions d'implémentation (aucune décision rouverte) :
  (a) D9w.1 — `SquadLayout.tsx` étant hors périmètre, le lien n'y est pas relayé :
  `useSquadPageRequests` le relit par le crochet partagé `useSquadDeepLink` (capture au montage,
  dans le même rendu de `SquadLayout` que `useSquadSessionSelection`). « Premier ancrage » = le
  premier ancrage appliqué par la page, consommé dans tous les cas ; le lien n'y vaut que si sa
  composition (triée) est la composition courante. « Décider `none` » s'applique quand la session du
  lien appartient à la composition (comparaison sans le suffixe « (N) ») ; une session du lien
  inconnue de la composition retombe sur les règles ordinaires (snap sur la dernière, ou « clear »),
  sinon la lourde partirait sur un label qu'aucune ligne ne porte (filtre serveur à l'égalité
  exacte, `filterSynthesisByPickedSessions`) : page vide — c'est le repli que promet le commentaire
  de la réconciliation (« si TOUS sont des zombies, le ré-ancrage reprend la main »). Mémorisation :
  la branche « pas de snap » d'`appliquerAncrage` (inchangée) écrit la dernière session de la
  composition dans `lastKnownLatestSessionId` ; un rechargement garde donc la session du lien
  (sélection épinglée, dernière déjà ancrée). Repli (légère en échec) : le premier ancrage se lit
  dans la réponse lourde, le lien y est gardé de même. Défaut antérieur à la campagne (même
  séquence dans `SquadLayout.tsx` à c89aa4bdc).
  (b) D9w.2 — `!isFetching` seul ne suffisait pas (mesuré : le test restait rouge, requêtes
  `lourde, legere, lourde`) : pendant le verrou de montage (`teammatesReady` faux) la légère est
  DÉSACTIVÉE, donc ni en cours, ni périmée aux yeux de TanStack (`isStale` est faux pour une requête
  désactivée, `@tanstack/query-core` 5.102.8, `queryObserver.js:333`), et sa donnée en cache
  décidait l'ancrage AVANT l'ouverture. `fresh` exige donc aussi `isEnabled` (champ du résultat de
  `useQuery`). À l'ouverture, la revalidation d'une donnée périmée part dans le même rendu
  (`isFetching` optimiste, `shouldFetchOptionally`) : aucun rendu ne décide sur la donnée périmée.
  Effet de bord : sur un cache léger frais (retour avant 5 min), la décision attend l'ouverture du
  verrou (un rendu plus tard), sans requête en plus.
  (c) D9w.3 — `attendLaLegere = hasTeammates || session pickée` ; l'échec de la légère libère
  toujours la lourde aussitôt (repli L4a).
  (d) D9w.4 — délai 500 ms (la moitié du premier délai de rejeu de l'application, 1 s ; le serveur
  annonce `Retry-After: 5`, trop long pour une lecture de quelques dizaines de ms qui a un repli) ;
  seul le 503 est rejoué : 500, 502, 504 et erreur réseau retombent aussitôt sur la lourde, comme
  avant. La règle est portée par la requête et prime sur la politique de l'application (les tests
  tournent sous un client à `retry: false` par défaut).
  (e) D9w.5 — détection par le NOM (`AbortError`), pas par `instanceof DOMException` (un polyfill
  peut rejeter une `Error` nommée). Une annulation avec une raison personnalisée (`abort(raison)`)
  rejette avec cette raison et resterait enregistrée : aucun appelant du dépôt n'en passe (TanStack
  Query appelle `abort()` sans argument, `query.js:225`) ; un dépassement `AbortSignal.timeout`
  (`TimeoutError`) reste enregistré, à dessein.
  (f) D9w.6 — double requête au montage en dev. StrictMode (`main.tsx`) monte, démonte puis remonte
  chaque composant. Quand le dernier observateur d'une requête part, `query-core` ANNULE son fetch
  en vol si la `queryFn` a lu `signal` (`Query.removeObserver` : `#abortSignalConsumed` →
  `retryer.cancel({ revert: true })`, `query.js:138`), puis le ré-abonnement du remontage relance
  la requête : deux fetch, le premier abandonné. Avant L3, les `queryFn` ne lisaient pas `signal` :
  `cancelRetry()` seul, le remontage récupérait la promesse en vol, une seule requête (reproduit par
  le relecteur C : Carrière et Séries temporelles, deux fetch dont le premier abandonné sous
  StrictMode, un seul sans StrictMode ou sans `signal`). Production non touchée : le double montage
  de StrictMode n'existe qu'en développement. L'Escouade y échappe : ses deux requêtes sont
  désactivées au premier rendu (verrou de montage `teammatesReady`, posé par un effet) et ne partent
  qu'au rendu suivant, après le double montage (témoin `SquadLayout.requests.test.tsx:418`, rouge si
  le verrou est retiré). Pour la mesure de clôture C.1 (Vite, donc StrictMode) : exclure des comptes
  la PREMIÈRE requête, abandonnée, de chaque requête à `signal` ACTIVE dès le premier rendu de sa
  page (constaté sur Carrière et Séries temporelles ; même mécanisme pour les autres hooks à
  `signal` du lot L3 : Synthèse, Sessions, détail de session, Accueil, résolution et aperçu des
  filtres solo) — « annulée » dans l'onglet Réseau ; si elle a atteint le serveur, 499
  `client_closed` dans `http.log` — : ni une régression, ni un coût de production.
- Gates (code final 7ecdc99bf, `node_modules\.tmp` purgé) : `npm run typecheck` 0 ; `npm run lint`
  0 (0 erreur, 26 avertissements, tous antérieurs, aucun sur les fichiers du lot) ; `npx vitest run
  src/features/squad src/features/filters src/lib/query src/lib/api src/lib/global-capture` 0 (88
  fichiers, 827 tests) ; suite complète `npx vitest run` 0 (791 fichiers et 8 503 tests verts ; 3
  fichiers et 19 tests ignorés, antérieurs ; 27 tests ajoutés, 1 renommé : « sans coéquipier ni
  session pickée … »). Hors gate : ratchets du pre-push `lint-no-hardcoded-fields`,
  `lint-no-hardcoded-colors`, `lint-cross-feature-imports` (7 ≤ 7), `knip-ratchet` (0/0/0),
  `lint-contract-ratchet` : verts. Tailles : `useSquadPageRequests.ts` 158 → 206 L,
  `squadPending.ts` 192 → 226 L, `useSquadSessionSelection.ts` 203 → 214 L, `queries.ts` 105 →
  121 L, `install.ts` 171 → 187 L ; les cas purs du lien profond vivent dans
  `squadPending.deeplink.test.ts` (nouveau, 68 L) pour garder `squadPending.test.ts` sous 500 L
  (489 L).
- Mutations jouées (13), toutes rouges puis restaurées (fichiers identiques à l'octet aux
  versions finales, empreinte du diff inchangée) : M1 règle du lien retirée de la décision (6
  rouges) ; M1b lien jamais transmis à la décision (3) ; M1c `none` sans condition, session du lien
  inconnue gardée (3) ; M1d lien jamais consommé (1) ; M1e lien appliqué à une autre composition
  (1) ; M2 `fresh` sans `!isFetching` (3) ; M2b `fresh` sans `isEnabled`, forme de la décision
  seule (3) ; M3 sans coéquipier la lourde n'attend jamais la légère (2) ; M4 `retry: false` (4) ;
  M4b 503 rejoué deux fois (2) ; M4c tout 5xx et erreur réseau rejoués (3) ; M5 garde `AbortError`
  retirée (2) ; M6 témoin StrictMode, verrou de montage retiré (1).
- Découvertes (non traitées, hors périmètre) :
  (1) La réconciliation des suffixes (`useSquadPageRequests.ts:165-175`) lit `source.sessions` même
  quand la source n'est pas fraîche (placeholder d'une autre composition, cache léger fermé ou en
  revalidation) : si un suffixe « (N) » a bougé entre deux visites, le label pické est réécrit
  d'après la donnée périmée puis d'après la revalidation (deux écritures du store, donc jusqu'à deux
  résolutions escouade) ; aucune requête lourde en plus (D9w.2).
  (2) Repli sur la lourde (légère en échec) : la fraîcheur de la réponse lourde ignore `isFetching`
  — au retour sur la page avec un cache lourd périmé ET la légère en échec, l'ancrage se décide sur
  la réponse lourde périmée (comportement L4a, hors de la lettre de D9w.2).
  (3) Le lien profond est relu par `useSquadPageRequests` (`useSquadDeepLink`) faute de pouvoir le
  faire passer par `SquadLayout.tsx` (hors périmètre) : deux captures de la même URL dans le même
  rendu. Le relayer par `SquadLayout` (un champ de `SquadPageRequestsInput`) rendrait le flux
  explicite ; si `useSquadPageRequests` était un jour monté après la redirection qui retire la
  query, la règle s'éteindrait sans bruit (`SquadLayout.deeplink.test.tsx:146` la garde).
  (4) `lib/global-capture/install.test.ts` : `_uninstallGlobalCaptureForTests` rend à `window` le
  fetch présent à l'installation, et l'`afterEach` du bloc « wrap fetch » restaure AVANT la
  désinstallation : chaque test à mock laisse ce mock dans `window.fetch` pour le suivant (mesuré
  par une sonde : au début du nouveau test, `window.fetch` est le `vi.fn` du test précédent) ;
  contourné dans le nouveau test par une capture du fetch de l'environnement en `beforeAll`.

## 10. Cloture de campagne (superviseur)

- [x] C.1 mesure de reference (§8 protocole) : Escouade a froid / a chaud / clic rail, Synthese,
      Sessions, Series temporelles, Carriere, Accueil ; tableau avant/apres dans l'etat des lieux —
      fait le 2026-09-23 : mesure finale 17:19-17:22 (tous les lots sauf L9), Accueil hors sync a
      18:34, Carriere hors sync apres L9-go a 19:47-19:50 (trois passages) puis avec le reglage
      local C.4 dans l'environnement du processus a 19:57-19:58 (deux passages) ; tableau au §6 de
      `.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md`
- [x] C.2 revue adversariale du diff cumule `origin/feat/v75..feat/perf-chargements` :
      fan-out aveugle (lentilles : parite des chiffres ; annulation / timeouts / erreurs ;
      caches et invalidation ; front sources de verite et cles de query), deux rondes max — faite
      le 2026-09-23 (quatre relecteurs Opus, lentilles A a D, une ronde : 1 P0, 3 P1, 12 P2, tous
      traites par L9-go (§9 quinquies) et L9-web (§9 sexies) ou consignes au §11)
- [ ] C.3 correctifs, gates complets (`go test ./...`, `-tags=integration -p 1`, typecheck `tsc -b
      --force`, eslint, vitest), push, CI verte au niveau job — correctifs : L9-go, L9-web, C4 ;
      gates rejoues par le superviseur sur l'arbre fusionne le 2026-09-23 au soir : gofmt vide,
      build 0, vet 0, `go test ./...` 190 paquets sans echec, integration `-p 1`
      sync/persist/duckdb/migration 18 paquets sans echec (13 min), `tsc -b --force` 0, eslint
      0 erreur (26 avertissements anterieurs), vitest 791 fichiers / 8 503 tests ; apres C4 :
      build, vet, tests duckdb / api / observability / archlint (`-count=1`), golangci-lint
      `--new-from-rev=origin/feat/v75` (verdicts au §12) ; push et CI : au §12
- [x] C.4 `.env.local` local : `LEVELUP_DUCKDB_THREADS=8`, `LEVELUP_DUCKDB_MEMORY_LIMIT=4GB`
      (poste 16 coeurs / 32 Go ; prod inchangee) — pose le 2026-09-23 a 19:52. DECOUVERTE a la
      pose : `LEVELUP_DUCKDB_MEMORY_LIMIT` / `_THREADS` etaient lues par des variables de paquet
      (`platform/duckdb/db.go`), evaluees avant `config.BootstrapEnvLocal()` : `.env.local` etait
      ignore en silence (serveur relance avec le fichier : budgets et durees inchanges ; relance avec
      les variables dans l'environnement du processus : rencontres 3,1-4,5 s vers 0,8-2,0 s). Lot C4
      (executeur Opus, `feat/perf-c4` 9fa4aa625, fusion 95ea5f010) : les deux bornes sont relues a
      l'ouverture de chaque connexion (`duckMemoryLimit()` / `duckThreads()` dans
      `applyDuckSessionInit` et `BudgetsSnapshot`) ; defauts 512MB / 2 et validation inchanges,
      l'environnement du processus reste prioritaire. Test sans tag
      `TestResourceLimits_EnvSetAfterPackageInitIsHonored` (3 / 300MB poses apres l'init :
      instantane 3 / "300MB", connexion neuve threads=3 et 286.1 MiB), rouge sur l'ancien code
      (mutation : 4 assertions). Gates verts (duckdb defaut + integration, api, observability,
      archlint, golangci 0 issue). Aucune autre variable de paquet du meme genre dans
      `platform/duckdb` ; une ailleurs, consignee au §11 (`LEVELUP_REPLAY_PUBLIC`).
- [ ] C.5 go utilisateur puis fusion dans `feat/v75` ; entree thought_log ; retrait des worktrees
      — entree thought_log de cloture ecrite le 2026-09-23 ; worktrees des lots (l1 a l8, l9web,
      l9go, c4, relecteurs) retires (jonctions d'abord, jamais `--force`) ; reste la fusion sur go
      de l'utilisateur, puis le retrait de `LevelUp-wt-perf` et des branches `feat/perf-*`
- [x] C.6 garde-rails structurels de perf, poses ou verifies a la cloture : (a) test grep interdisant `LEFT JOIN v_gamertag_lookup` dans les templates SQL des pages (queries_squad.go, squad_repo*.go) une fois L2 fusionne ; (b) tests de fenetres bornees par EXPLAIN ANALYZE sur les vues `_latest` (L5a : `tactical_repo_fenetres_test.go`, `weapon_range_repo_fenetres_test.go`) ; (c) ratchet d invalidation des caches de lecture (L5b : `archlint/player_read_cache_invalidation_test.go`) ; (d) garde-rail du filigrane LUSR (L6 : `lusr_watermark_guardrail_test.go`) ; (e) tableau avant/apres des durees par page dans l etat des lieux, meme protocole que le 23/09 matin.
      (jonctions node_modules retirees AVANT, jamais `--force`) — verifie sur pieces le 2026-09-23
      a 18:40 : (a) `annuaire_ratchet_test.go` (L7, etendu par L9-go a tout `internal/`, identifiant
      nu, table datee des occurrences restantes ; plus aucune lecture dans `queries_squad.go` ni
      `squad_repo*.go`), (b) les deux fichiers presents, (c) present, (d) present, (e) §6 de
      l'etat des lieux

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

- (cloture, superviseur, 2026-09-23 soir) Accueil : `GET /pages/home` rend 310 Ko en 1,6 s hors
  sync (historique complet dans une seule reponse) — charge utile a decouper ou a materialiser (plan
  structurel). Carriere : trois passages hors sync apres L9-go, rencontres 1,6 / 4,5 / 3,4 s et
  rivaux 2,1 / 6,5 / 4,1 s — le premier passage (requetes etalees par le socle) est le plus rapide,
  les rechargements lancent 6 lectures DuckDB en meme temps sur 2 threads / 512 Mo et chacune
  ralentit (csrs 18 -> 638 ms, achievements 16 -> 706 ms) : la fenetre `_latest` du kill-feed sur
  tout l'historique reste le cout (plan structurel), le reglage local C.4 en attenue l'effet.
  `platform/duckdb/db_query.go:136` (`logDBError`) journalise un `context canceled` en ERROR
  (« query failed (after recovery) », 3 par ouverture de la Carriere en dev a cause de StrictMode ;
  37 le 2026-09-22 a 18 h, avant la campagne) — meme classe que D9g.3, candidat a
  `observability.LevelUnlessCanceled`. `friends_xp` (bootstrap) tente d'ouvrir EN RW la DB de
  chaque ami non suivi pour la migrer (« chemin introuvable », 5 WARN par bootstrap, deja le matin
  a 10 h) : resoudre l'existence du profil avant d'ouvrir. Chocoboflor est « joueur en echec » a
  chaque cycle d'auto-sync (HTTP 429 de halostats a la decouverte, page 0) : externe a la
  campagne, a surveiller. Le cron des noms d'assets au boot (`asset_name_sweep_cron`, 60 s apres
  le demarrage) et l'expansion du catalogue de playlists (une trentaine de GET Waypoint, 19:47:20
  a 19:47:33) tournent pendant les premieres requetes de l'utilisateur apres un redemarrage.
  Lot C4 : `internal/api/handlers/replay_local_gate.go:74` lit `LEVELUP_REPLAY_PUBLIC` dans une
  variable de paquet a l'init (meme defaut que les bornes DuckDB : ignoree depuis `.env.local`),
  son commentaire presente la lecture unique comme voulue — a trancher. Chocoboflor : au boot,
  `cli_auth: refresh_token mort — reauth_required` (xuid 2535469190789936) : reconnexion SSO
  du compte a faire par l'utilisateur, jamais de re-capture (ADR 0023).

## 12. Journal

- 2026-09-23 : plan ecrit, branche de campagne creee depuis origin/feat/v75 c89aa4bdc,
  vague 1 lancee (L1, L3, L4a, L6).
- 2026-09-23 (apres-midi) : vague 1 fusionnee dans la campagne — L1 (6fba84c96), L3 (b10ec25b3), L6 (aba010a07), L4a (7088906ee, conflit filters/queries.ts resolu par le superviseur : signal de L3 + matchContext/enabled de L4a) ; gates rejoues sur l'arbre fusionne (build/vet/tests Go, typecheck, vitest 105 fichiers / 967 tests) ; vague 2 lancee (L2, L5a, L5b depuis 37cb48167). Reprises pour L4b : signal sur useTeammates (decouverte L3/L4a).
- 2026-09-23 (fin d'apres-midi) : L5a fusionne (1198c721e) et L5b fusionne (e1f173b74, conflit handlers/career.go resolu par le superviseur : requete unique de L5b + mapServiceError de L3) ; gates rejoues sur l'arbre fusionne. L2 encore en cours (relance envoyee a 13:35 apres 30 min sans activite, reprise constatee a 13:57).
- 2026-09-23 (soir) : L2 fusionne (8d016c94a, sans conflit), gates rejoues (build, vet, tests service/duckdb/analysis/archlint/api) ; vague 2 close ; L4b lance depuis 8d016c94a (worktree LevelUp-wt-perf-l4b). Decision utilisateur en attente : departage stable des badges ex aequo (Q32b sans ORDER BY).
- 2026-09-23 (16:30) : L4b fusionne (sans conflit), gates rejoues (build, tests teammates/api/port, typecheck, vitest). Mesure intermediaire 15:39-15:44 (sans L4b) consignee au scratchpad et reprise dans l'etat des lieux a la cloture : Escouade a froid 194 s -> 2,7 s, a chaud 26 s -> 2,8 s, Synthese 6,2 -> 2,7 s, Sessions 6,1 -> 0,3 s, Series temporelles 9,5 -> 0,5 s, Carriere 5,6 -> 1,5 s (hors rencontres/rivaux 10 s : lot L7), Accueil 4,2 -> 1,4 s ; cycle de sync 16 h : 11 bascules au lieu de 1 243. Lot L8 ajoute (departages stables Q29 / Q32b + MapCapabilityError sur POST /pages/teammates).
- 2026-09-23 (17:40) : L7 fusionne (b7bbe593f, conflit du plan resolu) et L8 fusionne (conflit du plan resolu) ; tous les lots sont dans la campagne. Cloture : gates complets, mesure finale, revue adversariale, CI.
- 2026-09-23 (18:30-20:00, cloture) : L9-web fusionne (ef5809933, sans conflit ; thought_log
  ac387b66f), worktree l9web retire ; L9-go fusionne (e1cf9da81 ; conflit du plan : §9 quinquies
  perdu a la resolution puis retabli en 98c79d9d5 avec le thought_log du lot ; arbre `apps/go-api`
  identique a celui du lot) ; Accueil mesure hors sync (2,1 s), Carriere mesuree hors sync apres
  L9-go (rencontres 1,6-4,5 s, rivaux 2,1-6,5 s) puis avec 8 threads / 4 Go (0,8-2,0 s) ;
  decouverte C.4 (reglages DuckDB lus avant `.env.local`) traitee par le lot C4 ; serveurs de
  mesure arretes a 20:00. Gates complets verts sur l arbre fusionne (C.3) ; lot C4 fusionne (95ea5f010, worktree retire) ; entree thought_log de cloture ; push de la branche et CI de cloture : ligne suivante.
- 2026-09-23 (21:05, CI de cloture) : push a9bffbd1c — gitleaks, gate ADR 0021 et Deploy Pre-Check
  verts ; workflow CI : 8 jobs verts (build + test Linux et Windows, lint, contrat OpenAPI, lease
  ADR 0013, OpenAPI lint, front), UN job rouge, « Go Coverage + Baseline non-regression » : 8 tests
  de la baseline absents du run (`internal/platform/duckdb`, TestCachedPlayerMatchesRepo_{7} et
  TestTTLCache_LenAndInvalidateAll — retires par L5b 7cd64d664 avec la reecriture du cache, sans
  retrait de la baseline dans le meme commit, contrairement a la regle). Remede prescrit par
  `scripts/check_test_baseline.sh` : 80 lignes JSONL / exactement 8 paires (Package, Test)
  retirees, paragraphe date dans l'en-tete du script ; le check rejoue en local sur le JSONL du
  run CI (artefact `go-baseline-current-jsonl`) : tous presents, 0 test en echec, 0 paquet en
  echec. Re-push pour la CI.
