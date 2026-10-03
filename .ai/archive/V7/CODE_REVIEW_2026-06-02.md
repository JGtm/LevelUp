# Revue de code complete - LevelUp (migration Go) - 2026-06-02

> Revue multi-agents (17 relecteurs specialises + verification adversariale des findings critiques/eleves).
> Perimetre : `apps/go-api/` (~218k LOC Go) + `apps/web/` (~113k LOC TS/TSX).
> Resultat brut : 131 findings sur 17 perimetres ; **1 critical, 13 high** (severite ajustee apres verification), 50 medium, 49 low ; 32 findings confirmes en relecture du code reel, 2 refutes/retrogrades.

---

## 1. Verdict d'ensemble

Le codebase est **mature et bien structure** sur ses fondations : discipline `slog` quasi parfaite, chemin d'ecriture anti-ART rigoureux (INSERT-only via `persist`), controle d'acces isole dans un package pur teste, store de tokens exemplaire, sequence de shutdown soignee, couche `analysis/` largement pure et tres bien testee, systeme de couleurs frontend reellement applique. Aucune corruption de donnees active n'a ete trouvee sur le chemin nominal.

Les risques se concentrent sur **trois axes critiques pour un deploiement mondial multi-user** :

1. **Posture de securite "fail-open"** : toute la securite repose sur la configuration d'environnement, **sans aucune validation au boot**. Les defauts (`AuthMode=none`, `SessionSecret=CHANGE_ME_IN_PRODUCTION`, `CORS=localhost`) sont non surs et ne sont surcharges par aucun fichier de deploiement. Une mauvaise config = donnees de tous les joueurs lisibles par n'importe qui.
2. **Surete des donnees sur les chemins de reprise/maintenance** : migration "rebuild" non transactionnelle (DROP+RENAME) sur la table partagee de tous les joueurs, restore qui retourne "Success" sans rien restaurer, chemin legacy ART-unsafe reactivable par flag.
3. **Hypotheses single-tenant cablees en dur** : multi-titres reel uniquement au bord HTTP (la couche data est cablee `halo_infinite`), pool de handles DuckDB non borne, timezone globale unique.

S'y ajoute une dette structurelle (god-files, 115 CLIs) et un i18n incomplet (utilisateurs EN voient du FR).

### Tableau de severite (apres verification)

| Perimetre | C | H | M | L | Note dominante |
|---|:-:|:-:|:-:|:-:|---|
| api / handlers / server | 1 | 1 | 4 | 1 | RealIP bypass + god-router |
| service | 0 | 0 | 6 | 2 | couplage horizontal + SQL inline |
| platform/duckdb | 0 | 0 | 5 | 3 | TZ non canonique + scans non bornes |
| sync + persist | 0 | 1 | 4 | 3 | legacy ART reactivable par flag |
| analysis | 0 | 0 | 4 | 6 | purete violee (SQL/slog) + duplication |
| domain + games + port | 0 | 0 | 5 | 3 | TitleDataAdapter dormant |
| migration/ops/scheduler/watcher | 0 | 2 | 6 | 2 | rebuild non transactionnel + restore no-op |
| cmd/ (CLIs) | 0 | 0 | 5 | 3 | 115 CLIs, bare sql.Open, .exe |
| multi-titres (sweep) | 0 | 0 | 4 | 3 | faux multi-titres en couche data |
| logging + erreurs (sweep) | 0 | 0 | 2 | 4 | slog sans Context + avalements |
| securite + authz | 0 | 3 | 3 | 1 | defauts fail-open sans garde-boot |
| concurrence | 0 | 1 | 4 | 2 | race sqlDB + goroutines non tracees |
| tests | 0 | 0 | 2 | 4 | invariants sync critiques t.Skip |
| frontend features | 0 | 1 | 9 | 1 | i18n FR hardcode + duplication |
| frontend lib/composants | 0 | 1 | 3 | 3 | query keys non centralisees |
| frontend transverse | 0 | 2 | 2 | 2 | 2 systemes i18n |
| readiness deploiement | 0 | 1 | 3 | 3 | defauts non surs + ressources non bornees |

---

## 2. CRITIQUE (1) - a corriger avant exposition publique

### C1 - RealIP non borne contourne LoopbackOnly : endpoints `/_diag/auto-sync` exposes
`apps/go-api/internal/api/server.go:112` + `:562-570` ; `internal/api/middleware/loopback.go:25-39` ; `handlers/admin_auto_sync.go:120-194`

`r.Use(chimiddleware.RealIP)` est applique **globalement** : go-chi RealIP ecrase `r.RemoteAddr` a partir des en-tetes client `True-Client-IP` / `X-Real-IP` / `X-Forwarded-For` **sans aucune allowlist de proxy de confiance** (le godoc go-chi avertit explicitement du risque). Les routes `/_diag/auto-sync/{snapshot,run,probe}` ne sont protegees QUE par `LoopbackOnly`, qui teste `IsLoopback()` sur un `RemoteAddr` deja reecrit. `probe` renvoie l'empreinte SHA256 + head/tail des refresh tokens et declenche une rotation OAuth live ; `run` force un sync.

**Impact** : derriere un CDN/LB en prod, un attaquant externe envoyant `X-Real-IP: 127.0.0.1` atteint des endpoints diagnostic non authentifies (exfiltration d'empreintes de tokens, syncs forces, rotation OAuth). Le meme vecteur fausse le rate-limit par IP et les logs d'audit.

**Remede** : (1) ne pas appliquer RealIP globalement sans CIDR de proxies de confiance configurable ; (2) faire `LoopbackOnly` lire l'adresse TCP reelle (avant RealIP, ou sous-routeur sans RealIP) ; (3) idealement proteger `/_diag/auto-sync` derriere `RequireAuth+RequireAdmin` comme `/admin`.

---

## 3. ELEVES (13)

### A. Securite & configuration fail-open (cluster - meme remede : garde-fou boot)

**H-A1 - Secret de session par defaut = cle publique committee, sans garde-fou** — `internal/config/config.go:241`
`SessionSecret: getEnvOrDefault("LEVELUP_SESSION_SECRET", "CHANGE_ME_IN_PRODUCTION")`. C'est la cle HMAC-SHA256 qui signe les cookies (`platform/session/store.go:175`). Aucun code de boot ne refuse de demarrer si le defaut subsiste, et `isProduction` (donc le flag `Secure` du cookie) en est deduit : secret par defaut => cookie transmis en clair sur HTTP. Vecteur realiste : `Secure=false` -> capture du `session_id` par sniffing -> re-signature triviale avec le secret connu.

**H-A2 - `AuthMode` defaut "none" desactive tout l'ownership multi-user** — `internal/config/config.go:247`
Avec `authMode=="none"`, `authz.Enforced()` -> false, `RequirePlayerOwnership` devient un pass-through total, `RequireAuth`/`RequireAdmin` no-opent. Fail-open : oublier `LEVELUP_AUTH_MODE=xbox` rend toutes les donnees de tous les profils lisibles par n'importe quel visiteur. Aucun avertissement au boot.

**H-A3 - Defauts non surs non surcharges par les fichiers de deploiement** — `internal/config/config.go:241-249` + `docker-compose.yml` + `.env.local.example`
`docker-compose.yml` (service `levelup`) ne definit ni `LEVELUP_AUTH_MODE`, ni `LEVELUP_CORS_ORIGINS`, ni `LEVELUP_SESSION_SECRET` ; `.env.local.example` ne les liste pas. En `AuthMode=none`, `/debug/vars` (expvar) est servi sans auth. Un operateur qui suit les exemples deploie ouvert.

**H-A4 - Refresh token OAuth Microsoft en clair a la racine** — `apps/go-api/token_Madina97294.txt:1`
Refresh token v2 Microsoft reel (`M.C544_SN1.0.U.-...`) d'un profil de prod (Madina97294), en clair dans le working tree. **Correctement gitignore et jamais committe** (verifie), mais expose a une copie accidentelle (backup, .zip support).

> **Remede unifie A1-A4** : un mode `LEVELUP_ENV=production` qui, au boot et hors `DemoMode`, **refuse de demarrer** (`slog.Error` + `os.Exit(1)`) si `SessionSecret==CHANGE_ME` (ou < 32 octets) OU `AuthMode==none` OU `CORSOrigins` contient `localhost`. Generer un `SessionSecret` aleatoire persistant si absent. Decoreler le flag `Secure` d'une variable explicite. Supprimer `token_Madina97294.txt` du disque (reimport via `cmd/token-import`) et rotater le RT cote Azure par precaution. Documenter ces 3 variables comme **requises** dans `.env.local.example` + `docker-compose.yml`.

### B. Surete des donnees (migration / restore / persist)

**H-B1 - Rebuild de `match_participants` non transactionnel : fenetre de perte totale** — `internal/migration/steps_shared_rebuild_match_participants.go:152-165`
La sequence `CREATE __rebuilt AS SELECT ...` ; `DROP TABLE match_participants` ; `ALTER ... RENAME` est executee statement par statement **sans transaction** (le package `internal/migration` ne contient aucun `Begin/Commit/Rollback`). Un crash/coupure entre le DROP et le RENAME laisse la table partagee de **tous les joueurs** definitivement absente, sans recovery (le `__rebuilt` orphelin n'est jamais detecte au boot). Meme pattern dans `steps_player_append_only_match_skill_rank.go:102-103`.
*(Trigger etroit : tourne au boot avant le scheduler, one-shot par DB ; mais impact catastrophique.)*
**Remede** : encadrer le swap dans une transaction explicite — le pattern existe deja en interne dans `cmd/rebuild_mp/main.go:66-105` (Begin -> DROP/RENAME/ADD PK -> validation COUNT -> Commit, Rollback sur erreur). Ajouter un chemin de recovery `__rebuilt` orphelin.

**H-B2 - `RestorePlayer` (replace=false) : faux "Success" sans rien restaurer** — `internal/ops/restore.go:162-176,99-104`
`CREATE TABLE IF NOT EXISTS %q AS SELECT * FROM read_parquet(...)` : si la table existe (cas le plus probable en reprise sur incident), DuckDB n'ecrit rien et ne leve pas d'erreur, mais la table est ajoutee a `TablesLoaded` et `Success=true`. De plus `CTAS` perd la PRIMARY KEY (reintroduit le piege PK/ON CONFLICT), le chemin Parquet est interpole sans echappement, et la boucle n'est pas transactionnelle.
**Remede** : pour `replace=false`, detecter l'existence et refuser ou `INSERT INTO` ; preserver le schema/PK ; placeholder DuckDB pour le chemin ; transaction.

**H-B3 - Chemin legacy `writes.go` ART-unsafe reactivable par flag** — `internal/sync/writes.go:32-180` + `engine_batch_path.go:50-54`
`submitOrInsertMatch` branche sur `batchMode` : poser `LEVELUP_PERSIST_BATCH=0` (documente comme fallback de rollback) reactive `InsertRegistryIfNotExists` / `insertParticipantRow` qui font `ON CONFLICT (...) DO UPDATE` sur `match_registry` et `match_participants` — exactement le pattern ART proscrit. **Aucun log WARN au boot**, et `no_art_patterns_test` ne couvre pas ces deux tables contre la forme `ON CONFLICT DO UPDATE` (elles sont hors `tablesProtegees`).
**Remede** : soit supprimer le chemin legacy, soit le faire pointer vers un `SharedPersister` INSERT-only ; a defaut, `slog.Warn` tres visible au boot si le flag est pose a 0, et documenter en gras "ART-unsafe, jamais en multi-user".

### C. Concurrence

**H-C1 - Data race sur `DB.sqlDB` : swap in-place sans lock cote lecture** — `internal/platform/duckdb/db.go:234,357,432-566,613`
Le champ `sqlDB *sql.DB` (et `closed bool`) est reassigne in-place sous `openDBsMu` dans `openCachedDB`/`Reopen`, mais lu **sans verrou** par `Query`/`Exec`/`QueryRow`/`ExecRecovered`/`QueryRecovered`/`UpsertNoConflict`/`SQLDb`. `WithReopenOnInvalidated` est appele depuis ~20 repos sur des `*DB` partages process-wide. Un Reopen concurrent d'une lecture HTTP en vol = data race (detectable par `go test -race`). *(Sur amd64/arm64 le pointeur aligne ne "tear" pas : consequence realiste = "database is closed" sur happens-before manque, pas un crash garanti.)*
**Remede** : `atomic.Pointer[sql.DB]` (load lock-free en lecture, store au swap), ou `RWMutex` sur la struct `DB`. Ajouter un test `-race` interleavant `Reopen()`/`Query()`.

### D. Frontend - i18n / accessibilite (deploiement mondial)

**H-D1 - `MatchCard` : ~20 libelles FR hardcodes malgre la prop `locale`** — `apps/web/src/components/ui/match-card.tsx` (146,253,280,333-341,395-441,527,545-551)
Composant reutilise (home, career, explorer...). Un seul libelle passe par i18n ; tout le reste ('Map inconnue', 'Escouade'/'Solo', 'Performance', 'frags'/'morts', 'Equipe'/'Adversaires', 'Maitrise !', 'Duree :'...) reste en FR en anglais. L'infra i18n (`formatMessage`/`commonManifest`) est deja en place, seul le cablage manque. (eslint `no-hardcoded-strings` en `warn` => le build passe.)

**H-D2 - `ChartCard` ne re-memoize pas l'option ECharts au changement de palette** — `apps/web/src/components/charts/ChartCard.tsx:94-99`
Le `useMemo` depend de `useThemeVersion()` (observe `data-theme`) mais pas de `useColorPaletteVersion()` (observe les mutations de `:root style` declenchees par `applyPalette()`). Les couleurs hex sont "bakees" dans l'option (le canvas n'accepte pas les CSS vars). Changer la palette d'accessibilite (Okabe-Ito/Cividis/Tol-Bright - **cle pour daltoniens**) sans toggler le theme laisse tous les charts ECharts (Bar/Radar/Timeseries/Heatmap) avec les anciennes couleurs jusqu'au remount. *(Portee plus large que decrite : `useColorPaletteVersion` n'a aucun consommateur.)*
**Remede** : ajouter `useColorPaletteVersion()` dans les deps du `useMemo` de `ChartCard`.

**H-D3 - Sections engagement : FR hardcode sans fallback EN, manifest bypasse** — `apps/web/src/features/engagement/EngagementMatchSection.tsx:64-123` ; `EngagementTimeseriesSection.tsx:71-80`
Composants sans prop `locale` ni hook i18n, chaines FR en dur ('Historique insuffisant...', 'Au-dessus de votre habitude...') alors que `engagement.toml` contient deja les traductions EN. `MatchViewPage` a `locale` mais ne le passe pas a ces composants : un utilisateur EN voit du FR.

**H-D4 - Deux systemes i18n coexistent** — ~30 fichiers `apps/web/src/features/`
140 ternaires inline `locale === 'en' ? ... : ...` (vraies chaines UI traduites a la main, pluralisation non-ICU `match${n>1?'s':''}`) coexistent avec les manifests TOML (source de verite, ADR 0003). Le linter `no-hardcoded-strings` n'inspecte ni les `ConditionalExpression` ni les template literals => ces traductions echappent au garde-fou de parite FR+EN. Risque eleve d'oubli de langue (cf. H-D3).
**Remede** : statuer - texte UI via manifests uniquement ; reserver `locale === 'en'` aux choix non-textuels (`numberLocale`). Regle ESLint dediee.

### E. Architecture / maintenabilite

**H-E1 - `NewRouter` : god-function ~880L mono-bloc** — `internal/api/server.go:79-961`
Une seule fonction (`//nolint:gocyclo`) melange montage des middlewares, chargement des field mappings + 3 adapters multi-titres, **ouverture repetee de `metadata.duckdb` avec retries/sleeps**, construction du `ServiceRegistry`, et l'enregistrement de ~80 routes. Pire : **`os.Exit(1)` ligne 289** dans un constructeur de `http.Handler` (sur echec `assetResolver`) - intestable et empeche un shutdown propre. La signature ne retourne pas d'`error`.
**Remede** : extraire `buildAdapters()`/`buildServiceRegistry()`/`mountRoutes(r, reg)` par domaine ; remonter les erreurs (`return err`) au lieu d'`os.Exit` ; sortir l'I/O DB du builder.

---

## 4. Themes MEDIUM saillants (50 findings)

**Fuites de couches (port pattern / layering)**
- Handlers `progression`/`campaign`/`milestones` instancient directement des repos `platform/duckdb` + mapping DTO inline (`handlers/progression.go:128-225`, `campaign.go:180-184`).
- `post_sync_deltas.go:181-652` : SQL brut multi-tables dans le package `api` + 2 god-functions `nolint`.
- SQL inline + ouverture DB directe dans la couche `service` (`media_service.ReassociateMedia:479-564`, `catalog_fetcher_service`).
- Couplage horizontal `HomeService` -> type concret `*CareerLiveService` (`service/home_service.go:62,141,235`) alors qu'Explorer abstrait correctement le meme service derriere une interface.

**Multi-titres = facade en couche data**
- DI hard-gate `slug==halo_infinite` (`internal/api/registry.go:317-322`).
- `data_health_check` et `auto_sync` resolvent uniquement halo_infinite (`scheduler/data_health_check.go:120-121`, `auto_sync.go:757`).
- `games.TitleDataAdapter` annonce `CapSupported` mais la plupart des `Load*` renvoient `ErrCapabilityNotSupported` (`games/halo_infinite/adapter_data.go:61-95`) ; champ `dataAdapter` injecte mais jamais lu dans 4 services.
- Deux systemes de capabilities paralleles a synchroniser a la main (`title.Capability` vs `games.CapabilityKey`).

**Efficacite DuckDB / correction des donnees**
- `player_matches_repo.go:142,222,228,254,313` ignore le pattern TZ canonique (`r.start_time` brut au lieu de `COALESCE(start_time_utc, ...)`) - **risque de correction des donnees utilisateur reel**.
- Requetes joueur non bornees (full scan `match_participants`/`player_match_enrichment`) qui scaleront mal en multi-user.
- `perfect_kills` en sous-requete correlee par ligne ; constantes SQL mortes avec prefixe cross-DB `shared.` interdit (ADR 0016).

**Concurrence (medium)**
- Data race sur l'etat du `Daemon` watcher (`running`/`cancel`/`rootCtx`) lu depuis les handlers HTTP sans verrou (`watcher/daemon.go`).
- `MatchQueue` : matchs marques `seen` **avant** l'envoi sur channel -> perte silencieuse si la file est pleine (`watcher/match_queue.go:45-87`).
- Goroutines fire-and-forget `PersistChallenges`/`PersistBattlePass` en `context.Background()` non tracees -> ecriture possible apres `duckdb.CloseAll` (`platform/duckdb/persist_sink.go:58-69,517-528`).
- `QueryRecovered` ne protege que la requete initiale, pas l'iteration `rows.Next()`.

**God-files (>500L)** : `domain/match_view.go` (810, DTO purs), `service/home_service.go` (3 concerns), `migration/steps_shared.go` (982), `ops/media.go` (952), `ops/seed.go` (910), `scheduler/auto_sync.go` (840), plusieurs repos duckdb 700-1015L, `sync/engine*` & `skill_rating.go` ~720-773L. ~59 fichiers Go > 500L au total.

**Outillage CLI** : 115 packages CLI (47 sont des `diag_*`/`lusr_v2_*`), ~100 ouvrant DuckDB en bare `sql.Open` (dont des UPDATE ART-risque), `fmt.Print*` au lieu de `slog`, one-shots/POC jamais nettoyes, ~60 `.exe` polluant le working tree (correctement gitignores). Une CLI unifiee `cmd/levelup` existe deja et devrait absorber le reste.

**Gestion d'erreurs / observabilite**
- Compteur `MatchesInserted` incremente **avant** l'ACK async -> le post-sync peut tourner sur des matchs non persistes si `Drain` echoue (`sync/engine_batch_path.go:184-198`).
- `refreshAggregates` avale les echecs de rebuild de vues materialisees (warn + `err=nil`) -> dashboard perime sans signal.
- Fuite de `err.Error()` interne vers le client sur 76 reponses 500.
- 286 `slog.*` sans `Context` en code request-scoped -> correlation `request_id`/`event_id` perdue.
- Absence d'en-tetes de securite HTTP (HSTS, X-Content-Type-Options, X-Frame-Options, CSP).

**Frontend** : query keys non centralisees sur ~10 features (dont divergence `userId` vs `playerSlug` -> fragmentation de cache + invalidation manquee) ; `console.log` de debug expedie en prod (`SynthesisPage.tsx:487`) ; mapping outcome->cle (magic 1/2/3/4) duplique dans 9+ fichiers ; `SynthesisPage` reimplemente le hook `useLocalFilterBar` extrait pour ca ; composants morts (`SynthesisCombatProfileSection`, `ChallengesCarousel`).

**Readiness deploiement** : healthcheck conteneur lit une variable de port inexistante (ne marche que sur 8000 par accident, `cmd/server/main.go:97-101`) ; timezone DuckDB globale unique pour toutes les sessions ; pool de handles DuckDB par joueur **sans eviction** (croissance memoire/FD non bornee) ; cooldown 429/503 fixe a 30s ignorant le header `Retry-After` de l'API Halo.

**Tests** : `contract_test.go` du sync - 10/12 invariants critiques (dedup cross-player, anti-regression URL `xuid(NNN)` du bug 14-jours, isolation des echecs partiels) **t.Skip'd en bloc** en attendant un livrable "D6"/orchestrateur V2 jamais livre -> aucune couverture live sur le chemin de prod V1 ; datasets e2e mono-categorie (Slayer, pas de mix PVE+PVP) ; test d'echange de code OAuth = test fantome (`t.Skip()` inconditionnel apres montage d'un mock complet).

---

## 5. Points forts confirmes

- **Discipline `slog`** : zero `fmt.Println`/`log.Printf` dans `internal/api` et `internal/` (sauf docstrings / CLIs stdout). Cles structurees coherentes, injection auto `request_id`/`event_id`.
- **Chemin anti-ART rigoureux** : ecritures per-match shared+player en INSERT-only atomique via `persist.SharedPersister`/`PlayerPersister`, serialisation des writers par `dblease`, WAL durable avec recovery au boot, idempotence par EXISTS. Persisters testes sous concurrence 10x20.
- **Controle d'acces** : `internal/authz` pur (0 I/O), chokepoint unique `RequirePlayerOwnership`, semantique correcte (admin/owner/403), tres bien teste (owner/foreign/admin/unlinked/demo). `MultiUserTokenStore` exemplaire (0600/0700, write-temp+rename, anti path-traversal, source unique ADR 0023).
- **Lifecycle** : shutdown ordonne `BatchQueue.Drain -> WaitInFlight -> Coordinator.Wait -> duckdb.CloseAll`, detection de fuites de refcount - vraie maitrise des incidents passes.
- **`analysis/`** : majoritairement pur, couverture excellente (temporal 91%, narrative 94%, breakdown 99%, skill_v2 89%).
- **Frontend** : `generated.ts` (6601L) isole derriere une facade unique (`lib/api/types.ts`) ; systeme de tokens couleur reellement applique (commentaires `color-allow` systematiques, aucune classe Tailwind couleur) ; `routeTree.gen.ts` propre.
- **Multi-titres au bord HTTP** : `RequireCapability` honnete (503, jamais de panic), branche sur capability et non sur slug ; `synthetic_title_b` prouve l'isolation.
- **Injection SQL** : requetes parametrees partout (placeholders, allowlists ORDER BY/IN), aucune injection trouvee. Secrets non committes (gitignores, historique propre).

---

## 6. Plan d'action priorise

### P0 - avant toute exposition publique mondiale
1. **Garde-fou boot "production"** (couvre H-A1..A4) : refuser de demarrer hors `DemoMode` si secret/auth/CORS non surs ; documenter les 3 variables comme requises ; generer un secret aleatoire si absent.
2. **C1 RealIP** : borner RealIP aux proxies de confiance + proteger `/_diag/auto-sync` derriere `RequireAuth+RequireAdmin`.
3. **Supprimer `token_Madina97294.txt`** du disque + rotater le RT cote Azure.
4. **H-B1 migration rebuild** : transactionnaliser (reutiliser le pattern `cmd/rebuild_mp`) + recovery `__rebuilt`.
5. **H-B2 restore** : refuser/INSERT sur table existante, transaction, preserver schema/PK.

### P1 - durcissement avant montee en charge
6. **Concurrence** : `atomic.Pointer[sql.DB]` (H-C1) ; tracer les goroutines fire-and-forget (ctx + WG) ; lock etat `Daemon` ; `MatchQueue` ne marquer `seen` qu'apres envoi.
7. **H-B3 legacy ART** : supprimer le chemin ou le rendre INSERT-only + `slog.Warn` boot si flag off.
8. **En-tetes de securite HTTP** (HSTS/CSP/X-Frame-Options/X-Content-Type-Options).
9. **i18n** (H-D1..D4) : cabler `MatchCard`/engagement sur les manifests ; `ChartCard` -> `useColorPaletteVersion` ; statuer sur les ternaires inline + regle ESLint.
10. **Healthcheck** : corriger la variable de port.

### P2 - dette structurelle & maintenabilite
11. **Decouper les god-files** : `NewRouter` -> `buildAdapters`/`buildServiceRegistry`/`mountRoutes` ; `cmd/server/main.go` ; reduire les fuites de layering (handlers/services -> repos).
12. **Decision multi-titres** : finir la couche canonique `TitleDataAdapter` (chemin data) OU documenter explicitement le single-title et retirer la facade dormante.
13. **CLIs** : unifier sous `cmd/levelup`, archiver les `diag_*` obsoletes, nettoyer les `.exe` du working tree, passer les CLIs sur le helper DB canonique + `slog`.
14. **Tests** : de-skip `contract_test.go` (ou livrer l'orchestrateur V2) ; datasets e2e heterogenes PVE+PVP.
15. **Ressources multi-user** : borner le pool de handles DuckDB par joueur (eviction LRU) ; timezone par-session ; respecter `Retry-After`.
16. **Dette connue** : corriger la TZ `first_joined` (~964 matchs decales, casse T0/quit-ordering LUSR).

---

*Revue generee le 2026-06-02. Findings ancres sur du code reel relu ; severites ajustees apres verification adversariale (les findings refutes/retrogrades ont ete ecartes du decompte critical/high).*
