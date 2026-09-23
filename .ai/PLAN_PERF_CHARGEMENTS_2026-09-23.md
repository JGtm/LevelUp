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
- [ ] L7.1 rencontres (top-encounters) sans jointure + parite + chrono
- [ ] L7.2 rivaux sans jointure + parite + chrono
- [ ] L7.3 autres lecteurs du meme fichier (grep `v_gamertag_lookup` dans `queries_career*.go`,
      `career_repo*.go`, `home_repo*.go`) : traites s'ils sont sur une page, sinon consignes

Gate : comme L2 (paquets touches) ; `-tags=integration -p 1 ./internal/platform/duckdb/...`.
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
- [ ] C.6 garde-rails structurels de perf, poses ou verifies a la cloture : (a) test grep interdisant `LEFT JOIN v_gamertag_lookup` dans les templates SQL des pages (queries_squad.go, squad_repo*.go) une fois L2 fusionne ; (b) tests de fenetres bornees par EXPLAIN ANALYZE sur les vues `_latest` (L5a : `tactical_repo_fenetres_test.go`, `weapon_range_repo_fenetres_test.go`) ; (c) ratchet d invalidation des caches de lecture (L5b : `archlint/player_read_cache_invalidation_test.go`) ; (d) garde-rail du filigrane LUSR (L6 : `lusr_watermark_guardrail_test.go`) ; (e) tableau avant/apres des durees par page dans l etat des lieux, meme protocole que le 23/09 matin.
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
- 2026-09-23 (fin d'apres-midi) : L5a fusionne (1198c721e) et L5b fusionne (e1f173b74, conflit handlers/career.go resolu par le superviseur : requete unique de L5b + mapServiceError de L3) ; gates rejoues sur l'arbre fusionne. L2 encore en cours (relance envoyee a 13:35 apres 30 min sans activite, reprise constatee a 13:57).
- 2026-09-23 (soir) : L2 fusionne (8d016c94a, sans conflit), gates rejoues (build, vet, tests service/duckdb/analysis/archlint/api) ; vague 2 close ; L4b lance depuis 8d016c94a (worktree LevelUp-wt-perf-l4b). Decision utilisateur en attente : departage stable des badges ex aequo (Q32b sans ORDER BY).
