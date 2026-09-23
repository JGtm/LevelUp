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
- [ ] L1.1 paquet `timing` + tests
- [ ] L1.2 middleware : seuil + `http_timings` + test (seuil respecte, 2xx lent en INFO, 2xx rapide en DEBUG)
- [ ] L1.3 sections GetPage (26)
- [ ] L1.4 sections filtres, synthese, sessions, series temporelles, coordination
- [ ] L1.5 doc de la variable (EN + FR)

Gate : `gofmt -l` vide ; `go build ./...` ; `go vet ./...` ; `go test ./internal/observability/...
./internal/api/middleware/... ./internal/service/... ./internal/service/teammates/...` ;
`golangci-lint run` sur les paquets touches : 0 issue nouvelle.

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
