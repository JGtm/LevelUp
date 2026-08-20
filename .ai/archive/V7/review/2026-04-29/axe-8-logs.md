# Axe 8 — Logs & observabilite

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Perimetre : apps/go-api/internal/{observability, notify, api/middleware, api/handlers, sync, service, platform}/* + apps/web/src/**

## Synthese

Logger : `log/slog` stdlib (Go 1.26), JSON ou Text selon `LEVELUP_LOG_JSON`, niveau pilotable par `LEVELUP_LOG_LEVEL`. La fondation est correcte (689 occurrences slog, init unique dans `cmd/server/main.go`). Trois problemes structurels : (1) le `request_id` est genere mais jamais propage dans le `context.Context`, donc absent de tous les logs business hors middleware HTTP ; (2) le package `internal/notify` reste sur `log.Printf` (29 sites, 3 fichiers) et echappe completement au logger structure ; (3) l'endpoint `/debug/vars` (expvar) n'est pas monte dans le routeur — le package `internal/observability` est mort. Pas de Prometheus / OpenTelemetry / Sentry assume (decision explicite, plan §4.7). Aucun `panic`/`log.Fatal` non justifie en code de prod.

## Compteurs

- Logger principal : `log/slog` stdlib
- `slog.*Context(ctx, ...)` : 474 occurrences / 75 fichiers (forme recommandee)
- `slog.*(...)` sans Context : 215 occurrences / 48 fichiers (forme appauvrie, ratio ~31%)
- `log.Printf` dans `internal/` : 29 occurrences / 3 fichiers (uniquement `internal/notify/{discord,notifiers,version}.go`)
- `fmt.Println` / `fmt.Printf` actifs dans `internal/` : 0 (les 2 hits Grep sont des commentaires d'exemple, voir `validation/gate.go:9` et `ops/healthcheck.go:8`)
- `panic()` hors `cmd/` : 1 (`internal/platform/auth/sisu_provider.go:180` — guard de bug d'utilisation, justifie)
- `log.Fatal` hors `cmd/` : 0
- `os.Exit` hors `cmd/` : 1 (`internal/api/server.go:215` — boot fatal sur asset resolver)
- Erreurs reellement avalees `_ = X.Method(...)` en code de prod : ~10 (le reste, ~205 hits, sont des tests ou des `_ = os.Remove(tmpPath)` post-rollback legitimes)
- `console.log` dans `apps/web/src/` non-test : 10 occurrences, toutes encapsulees dans 3 wrappers `_logger.ts` + 1 `ErrorBoundary.tsx` (pas de console direct)
- Endpoint `/healthz` ou `/readyz` dedie : non (un seul `/health` mixte liveness+readiness+counts)
- Frontend Sentry / LogRocket / Datadog : absent (decision implicite, pas de config)
- `expvar` / `/debug/vars` exposes : non (package `internal/observability` jamais importe ailleurs)

## Constats

### [BLOQUANT] `request_id` jamais propage dans le ctx — debug prod cassed-by-design

`internal/api/middleware/request_id.go:14-22` ecrit l'ID dans `w.Header()` mais ne fait jamais `ctx = context.WithValue(...)`. Le middleware suivant `SlogLogger` re-lit ce header pour logger la ligne d'acces (`slog_logger.go:30`), mais aucun service ou repo en aval n'a moyen de retrouver ce request_id : `internal/ctxkeys/ctxkeys.go` n'expose que `titleSlugKey`, `haloTokensKey`, `haloXUIDKey`. Consequence : impossible de correler une 500 logguee par `SlogLogger` avec les 3-5 lignes `slog.WarnContext` emises par les services et repos pendant la meme requete. Tous les log front + back sont independants.

```go
// request_id.go:14-22 — ecrit le header mais pas le contexte
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get(headerRequestID)
        if id == "" { id = uuid.New().String() }
        w.Header().Set(headerRequestID, id)
        next.ServeHTTP(w, r)
    })
}
```

Correction : injecter dans `ctx` via une cle `ctxkeys.RequestID` + brancher un `slog.Handler` custom qui lit cette cle pour l'ajouter automatiquement dans les attributs de chaque log emis avec `slog.*Context`.

### [BLOQUANT] Package `internal/notify` 100% sur `log.Printf` — 29 sites hors slog

`internal/notify/discord.go` (5 sites L107/135/140/148/154), `notifiers.go` (17 sites L36/44/48/52/58/61/68/70/87/97/107/111/120/268/284/298/318), `version.go` (7 sites L39/51/58/64/76/82/84). Aucune integration avec `slog` : ces logs partent sur stderr en format texte non structure, sans request_id, sans duration, sans niveau (ni Info/Warn/Error). Concretement, en JSON-mode prod, ces lignes ressortent comme du bruit non-JSON dans le flux et ne peuvent pas etre filtrees ni alertees. Cette dette est deja note dans `.ai/thought_log.md [2026-04-29]` ("Dette pre-existante detectee — A traiter en follow-up sur le package complet").

```go
// notifiers.go:36 — pattern sur tout le fichier
log.Printf("[Discord:sync] panic récupéré: %v", r)
// devrait etre :
slog.ErrorContext(ctx, "discord_sync_panic", "err", r)
```

### [BLOQUANT] Package `internal/observability` mort — `/debug/vars` jamais expose

`internal/observability/expvar_metrics.go` definit 4 categories de metriques (`service_duration_ms`, `repo_query_duration_ms`, `cache_hit_ratio`, `error_count`) avec helpers `IncCounter` / `RecordDurationMS` proprement threadsafe. Mais `Grep observability\.` ne trouve aucun import en dehors du package lui-meme (file_with_matches : 2 fichiers — le code et son test). Le routeur `internal/api/server.go` ne monte pas `/debug/vars`. Les 4 categories ne sont jamais incrementees. Resultat : zero observabilite metrique cote backend, le travail expvar est entierement dormant.

### [DETTE] 215 sites `slog.*(...)` non-Context dans des fonctions qui ont un ctx

Ratio 215/689 = 31% des appels slog ne propagent pas le ctx. Exemples critiques (handlers HTTP avec `r.Context()` immediatement dispo) :

- `internal/api/handlers/admin.go` — 10/10 sites (L56/60/84/88/112/116/154/158/171/175) en `slog.Info`/`slog.Error` sans Context
- `internal/api/handlers/watcher_handler.go` — 11 sites
- `internal/api/handlers/user_auth.go` — 12 sites
- `internal/service/match_view_service.go:202-302` — 9 sites `slog.Warn(...)` au sein d'`errgroup.WithContext(ctx)` alors que `gctx` est explicitement disponible (ligne 196), face a 2 sites L245/L252 qui passent bien `gctx`. Incoherence intra-fichier.
- `internal/service/bootstrap_service.go:277` — `slog.Warn` alors que les 2 sites adjacents L68/L75 utilisent `slog.WarnContext`.

Avec un handler slog enrichi par request_id (cf. BLOQUANT 1), ces 215 sites perdent l'attribut request_id. La forme `Context` est obligatoire dans la doc projet (`.claude/skills/arch-rules/SKILL.md` § Logging).

### [DETTE] `/health` mixte liveness + readiness + payload metier

`internal/api/handlers/health.go:31-53` execute `repo.GetMatchCount(ctx)` (avec retour 503 si erreur) + 3 lectures additionnelles (`GetDBVersion`, `GetPlayerCount`, `GetLastSyncAt`). Ce endpoint sert simultanement de :
- liveness (le binaire repond)
- readiness (la DB shared est lisible)
- info dump (uptime, go_version, app_version)

Probleme : un orchestrateur Kubernetes / load balancer doit pouvoir distinguer "le process est vivant mais pas pret" (DB indisponible momentanement) de "le process est mort". En l'etat, un blip DuckDB renvoie 503 et un kill-and-replace est declenche, alors qu'un simple retire-from-LB suffirait. Standard : `/healthz` minimal (HTTP 200 si process up) + `/readyz` qui touche la DB.

### [DETTE] `internal/api/middleware/error_tracker.go` desactive en dur

L66-69 : `func (et *ErrorTracker) Middleware(next http.Handler) http.Handler { return next }` — toute la machinerie d'alerting Discord 500 + taux d'erreur sur fenetre 1min est court-circuitee. Le code "vrai" est conserve sous `middlewareDisabled` (L74, marque `//nolint:unused`). Soit c'est de la dead code museum a supprimer (250L pour rien), soit la fonctionnalite doit etre reactivable via feature flag — le commentaire L65 dit "Reactiver en supprimant le return immediat ci-dessous" mais ne pointe pas vers une raison documentee, et il n'y a pas de variable env / flag pour le piloter en runtime.

### [DETTE] Erreurs avalees silencieusement sur `slog.WarnContext` hot path home

`internal/service/home_service.go` L178/191/199/216/393 : 5 chemins ou un `LoadSpartanIdentity`/`LoadRecentMedia`/`LoadRecentPlaylistRanks`/`GetFavoriteMatchIDs`/`BattlePass persist` echoue, est logge en `slog.WarnContext`, et la home renvoie un payload partiel sans signaler la degradation au client. Pour le debug prod, `slog.Warn` correct, mais le client n'a aucun moyen de savoir que sa home est incomplete (pas de champ `partial: true` ni de `data_freshness`). Note : ce point recoupe l'axe 1 (contrats services), donc constat hors-axe ci-dessous.

### [DETTE] `os.Exit(1)` au boot sur asset resolver — failsafe trop dur

`internal/api/server.go:212-216` : si `assets.New(assetCfg)` echoue, le serveur `os.Exit(1)`. Or l'asset resolver est utilise pour servir les images de maps/medailles/badges — pertes esthetiques, pas une erreur metier. Le mode demo ou un setup neuf sans cache `data/cache/` peut tres bien tomber dessus. Devrait etre `slog.Warn` + degrader gracefully (handler renvoyant 404 sur les assets, le reste de l'API continue). Note : c'est dans `api/server.go` qui est techniquement appele depuis `cmd/server`, donc partiellement defendable, mais l'API package est cense etre la lib HTTP injectee — pas le bootstrap.

### [AMELIORATION] Frontend : pas de SDK observabilite, mais 3 wrappers `_logger.ts` ad-hoc

`apps/web/src/features/filters/_logger.ts`, `features/squad/_logger.ts`, `lib/accessibility/_logger.ts` — meme pattern repete 3x : prefix, dedupe via `Set`, dev-only debug. C'est correct en l'etat (10 console.* total dans le source non-test, tous encapsules), mais a centraliser : creer `apps/web/src/lib/logger.ts` exposant `createLogger(prefix)` ; supprimer les 3 wrappers feature-locaux. Sans Sentry/LogRocket, les erreurs front silencieuses (network, parse) ne remontent jamais cote ops — acceptable si l'app est self-hosted et utilisee par 4 joueurs.

### [AMELIORATION] `panic()` justifie dans `sisu_provider.go:180`

Guard de bug d'utilisation (`Exchange` appele avant `InitDeviceFlow`). Acceptable car le code legacy Python avait le meme guard. Pourrait migrer vers `return ErrFlowNotInitialized` pour ne pas crasher le serveur entier en cas de mauvais ordering UI — risque faible mais reel.

### Constats hors-axe

- (axe 1) `home_service.go` degrade silencieusement sans champ `partial`/`degraded` dans la reponse — meme pattern dans `match_view_service.go:202-302` (10 chemins log+swallow).
- (axe 3) `internal/observability` jamais branche est aussi un probleme de wiring DI : le code cree des metriques mais aucun service ne les lit pour incrementer.

## Cartographie : flux d'un log dans une requete HTTP

```
1. Requete entrante : GET /api/v1/players/JGtm/pages/home
2. middleware.RequestID — genere uuid, ecrit dans w.Header()["X-Request-ID"]
   ⛔ NE TOUCHE PAS le ctx — l'ID est invisible aux couches en aval
3. middleware.CORS / CSRF / RateLimit — pas de log
4. middleware.SlogLogger (debut) — capture start time
5. middleware.WithSession — pas de log courant
6. middleware.TitleExtractor — ecrit ctxkeys.TitleSlug (visible aux couches aval)
7. handler.HomeHandler.Get — utilise r.Context() pour propager au service
   ⚠️ pas de log explicite cote handler
8. service.HomeService.LoadHome — emet 4-5 slog.WarnContext(gctx, ...)
   avec attrs {gamertag, err, match_id} mais SANS request_id
   exemple : slog.WarnContext(gctx, "home: LoadSpartanIdentity failed", "err", err)
9. platform/duckdb.HomeRepo — quasi pas de log (pas d'observation des queries)
   ⛔ aucun timing per-query, aucun observability.RecordDurationMS
10. retour vers handler — writeJSON(payload)
11. middleware.SlogLogger (fin) — emet 1 ligne unique avec :
    {method, path, status, duration_ms, response_bytes, request_id, remote_addr, title_slug}
    NIVEAU : Debug si <400, Warn si 4xx, Error si 5xx
```

Constat : les lignes #8 (services) et #11 (acces HTTP) sont ecrites sur le meme stream stderr mais ne partagent aucune cle commune sauf `match_id`/`gamertag` quand ils existent. Pour reconstruire le flux d'une 500, il faut grep par timestamp et esperer qu'aucune autre requete concurrente n'emet en parallele. En charge, c'est ininterpretable.

## Suivi recommande

1. **Brancher request_id dans le ctx + handler slog enrichi** — ajouter `ctxkeys.WithRequestID` / `ctxkeys.RequestID` ; creer un `slog.Handler` wrapper qui lit `RequestID(ctx)` et l'attache automatiquement a chaque attribut. Ratio cible : 100% des `slog.*Context` portent `request_id`. Effort : ~1 jour.
2. **Migrer `internal/notify` vers slog structure** — les 29 sites `log.Printf` → `slog.InfoContext` / `slog.ErrorContext` avec attributs `op`, `gamertag`, `webhook_status_code`. Effort : ~2h. Recoupe la dette deja documentee dans `thought_log.md [2026-04-29]`.
3. **Decision binaire sur `internal/observability` + `internal/api/middleware/error_tracker.go`** — soit on cable l'expvar (instrumenter 5-10 hot paths repo + monter `/debug/vars`), soit on supprime les deux packages. Le code dormant cree de la confusion (un futur dev pensera que les metriques sont actives) et viole l'anti-pattern "Dead code museum" liste dans `CLAUDE.md`. Effort : 30 min de decision + 1h d'execution.

---

## Amendement post-vérification (2026-04-29)

> Ajout issu de la passe de vérification finale (cf. [verification-finale-scaffolding.md](verification-finale-scaffolding.md)).

### [DETTE] `post_sync_deltas.go` émet des notifications vers 3 routes inexistantes (observability incomplète côté output)

- **Fichier:ligne** : `apps/go-api/internal/api/post_sync_deltas.go:261,277` — `TargetRoute` pointe vers `/players/%s/defis` (route inexistante). Voir aussi le miroir front `apps/web/src/features/notifications/navigation.ts:46,52,55` qui pointe vers les mêmes 3 routes fantômes (`/help/changelog`, `/defis`, `/sync`).
- **Problème** : c'est de l'observability cassée côté **sortie** (pas côté logs structurés, mais notifications utilisateur) : le système émet des deltas et croit pointer vers une page de drill-down qui n'existe pas. Aucune log ne signale ce désalignement parce que la route 404 est traitée côté front. Symptôme cohérent avec le pattern « scaffolding then forget » identifié sur les autres axes.
- **Action** : ajouter un test de contrat (table-driven) qui vérifie que tous les `TargetRoute` émis dans `post_sync_deltas.go` correspondent à des routes existantes dans `routeTree.gen.ts`. Recoupe le constat amendé en axe 4.
