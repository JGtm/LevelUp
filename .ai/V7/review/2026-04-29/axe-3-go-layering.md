# Axe 3 — Layering & responsabilités (Go)

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/go-api/internal/{api,service,analysis,port,sync,domain,platform,games}/

## Synthèse (3-5 lignes max)

Les fondations sont en place : `port/` dispose d'une trentaine d'interfaces, la plupart des handlers consomment `port.*Service` via `ServiceFactory[S]`, `domain/` est pur (n'importe que `canonical/`), et `analysis/` n'importe ni `service/` ni `sync/` ni `platform/`. Mais la frontière `service/ ↔ platform/` est rompue à plusieurs endroits : `service/media_service.go` et `service/media_index_service.go` ouvrent DuckDB directement (`sql.Open("duckdb", path)`) et font du `os.Stat`/`filepath.Walk`, `service/home_service.go` dépend d'un type concret `duckdb.PersistSink`, et `internal/games/halo_infinite/ranks_loader.go` viole la direction d'import en consommant `platform/duckdb` depuis `games/`. Côté handlers, plusieurs (Bootstrap, Engagement, Lab) prennent des `*service.Foo` concrets au lieu de `port.Foo`, et `handlers/help.go` shell out à `git` + parse markdown — toute logique métier qui devrait vivre dans un service ou un package dédié. Verdict global : **bon** layering au sens MVC mais 3 BLOQUANTS à corriger pour fermer proprement la frontière service/platform.

## Matrice frontières (Go)

| Couche | Dépend de (réel) | Devrait dépendre de | Violations notables |
|---|---|---|---|
| `api/handlers/` | `port/`, `domain/`, `service/`, `sync/`, `config/`, `notify/`, `notifications/`, `platform/{auth,settings,jobs}` | `port/` + `domain/` + middleware + (occasionnellement) `service/` pour les types DI génériques | 3 handlers (Bootstrap, Engagement, Lab) tapent dans le type concret `*service.Foo` ; `help.go` shell out git ; `media.go` fait du file-walk |
| `api/` (server, registry) | `service/`, `platform/duckdb`, `platform/halo`, `notifications`, `games`, `config` | `service/` + `platform/*` (composition root) | OK — c'est l'endroit légitime pour le wiring concret |
| `service/` | `analysis/`, `port/`, `domain/`, `games/`, `platform/halo`, `platform/duckdb` (1 fichier), `platform/jobs` (1 fichier), `ops/`, `database/sql` (2 fichiers) | `analysis/` + `port/` + `domain/` + `games/` uniquement | 2 services ouvrent DuckDB direct (`media_service.go`, `media_index_service.go`) ; `home_service.go` couplé à `duckdb.PersistSink` concret |
| `analysis/` | `domain/`, `games/canonical`, `games/mappings`, `assets/static` (1 fichier), stdlib | `domain/` + `games/canonical` + stdlib | `home.go` tire `assets/static` (URL helper, mineur) ; quelques `slog.Debug/Warn` et un `time.Now()` (sessions.go:171) — impurités locales |
| `sync/` | `analysis/`, `domain/`, `assets/`, `platform/auth`, `database/sql` (direct OK ici) | `analysis/` + `domain/` + `platform/*` | OK — sync est un orchestrateur d'ingestion qui peut légitimement utiliser DuckDB direct |
| `domain/` | `games/canonical` uniquement, stdlib | `games/canonical` + stdlib | OK — propre |
| `port/` | `domain/`, `games/canonical`, `analysis/temporal` (1 ref) | `domain/` + `games/canonical` | OK — aucune dep vers `platform/` |
| `games/` | `canonical/`, `mappings/`, `domain/`, `platform/duckdb` (1 fichier) | `canonical/` + `mappings/` + `domain/` | `halo_infinite/ranks_loader.go` importe `platform/duckdb` (violation directionnelle) |
| `platform/duckdb` | `domain/`, `port/`, `games/canonical`, `assets/`, stdlib | `domain/` + `port/` + stdlib | OK — implémente les ports |

## Constats

### [BLOQUANT] DuckDB ouvert directement dans `service/` (sql.Open contourne le pool)

- **Fichiers** :
  - `apps/go-api/internal/service/media_service.go:279, 319` (méthodes `UploadMedia` / `ReassociateMedia`)
  - `apps/go-api/internal/service/media_index_service.go:236` (`resetPlayerMediaIndex`)
- **Extrait** :
  ```go
  // service/media_service.go:319
  db, err := sql.Open("duckdb", targetPath)
  // service/media_index_service.go:236
  db, err := sql.Open("duckdb", dbPath)
  ```
- **Problème** : `service/` n'a pas le droit d'ouvrir DuckDB, et les ouvertures directes ici contournent le pool de connexions `platform/duckdb/pool.go` + le ref-count `db.go` (dont la criticité a été soulignée le 2026-04-18 dans `project_map.md`). À la moindre concurrence avec Home/Battle Pass, ça refait tomber l'erreur `same database file with a different configuration`.
- **Action** : extraire la logique média/reset dans un nouveau `port.MediaIndexRepository` et `port.MediaUploadRepository` implémentés dans `platform/duckdb/`, et faire passer toutes les ouvertures par `pool.Get(...)`.

### [BLOQUANT] `service/home_service.go` dépend du type concret `duckdb.PersistSink`

- **Fichier:ligne** : `apps/go-api/internal/service/home_service.go:18, 31`
- **Extrait** :
  ```go
  import "levelup/go-api/internal/platform/duckdb"

  type HomeService struct {
      repo port.HomeRepository
      // ...
      sink *duckdb.PersistSink // nil → pas de persistance
  }
  ```
- **Problème** : un service ne devrait dépendre que de `port.*`. Le couplage à `duckdb.PersistSink` rend `HomeService` non-mockable hors environnement DuckDB et empêche d'injecter un sink alternatif (memory, no-op typé, second backend).
- **Action** : définir `port.HomePersistSink` (interface : `Enqueue(ctx, item)` + `Flush(ctx)`) dans `port/` et faire passer `WithPersistSink(port.HomePersistSink)` ; `*duckdb.PersistSink` implémentera l'interface sans changement.

### [BLOQUANT] `internal/games/halo_infinite/ranks_loader.go` importe `platform/duckdb` (cycle directionnel)

- **Fichier:ligne** : `apps/go-api/internal/games/halo_infinite/ranks_loader.go:10, 22`
- **Extrait** :
  ```go
  import "levelup/go-api/internal/platform/duckdb"

  func LoadRankCatalog(ctx context.Context, metaDB *duckdb.DB) (*mappings.RankCatalog, error) {
      // ...
      rows, err := metaDB.Query(ctx, `SELECT rank_id, lang, ... FROM career_rank_translations`)
  }
  ```
- **Problème** : `games/` est censé être un namespace adapter au-dessus du canonique (cf. skill `arch-rules`, README adapters). Importer `platform/duckdb` casse cette frontière et empêche un titre synthétique de stub `LoadRankCatalog` sans DuckDB. C'est aussi un dial direct sur DuckDB hors `platform/` — interdit par la grille.
- **Action** : déplacer le SQL dans `platform/duckdb/metadata_repo.go` (méthode `LoadCareerRankTranslations`), exposer un `port.RankCatalogLoader`, et garder dans `halo_infinite/` uniquement la projection `[]TranslationRow → mappings.RankCatalog`.

### [BLOQUANT] Logique métier (git + extraction markdown) dans `handlers/help.go`

- **Fichier:ligne** : `apps/go-api/internal/api/handlers/help.go:169-271`
- **Extrait** :
  ```go
  func buildFullReleaseHistory(repoRoot, lang string) (string, error) {
      // 1) os.ReadFile README ; 2) git log/show via exec.Command ; 3) parsing What's-new
  }
  func gitLogSHAs(repoRoot, relPath string) ([]string, error) {
      cmd := exec.Command("git", "log", "--all", "--format=%H", "--", relPath)
  }
  ```
- **Problème** : 390 lignes de logique métier (cache disque, exec git, parsing markdown, fallback) dans un handler. Aucune testabilité hors HTTP, aucune frontière de couche.
- **Action** : créer `service/release_notes_service.go` (cache + git + parsing) derrière `port.ReleaseNotesService` ; le handler ne fait plus que `svc.GetReleaseNotes(ctx, lang)` + `writeJSON`.

### [DETTE] Handlers couplés au type concret `*service.Foo` au lieu de `port.Foo`

- **Fichiers** :
  - `apps/go-api/internal/api/handlers/bootstrap.go:13, 17, 34, 38` (`*service.BootstrapService`)
  - `apps/go-api/internal/api/handlers/engagement.go:27, 31` (`*service.PlayerEngagementService`)
  - `apps/go-api/internal/api/handlers/lab.go:15, 19` (`*service.LabService`)
- **Extrait** :
  ```go
  // handlers/bootstrap.go
  type BootstrapHandler struct { svc *service.BootstrapService }
  // engagement.go
  newSvc ServiceFactory[*service.PlayerEngagementService]
  ```
- **Problème** : les interfaces `port.BootstrapService` et `port.LeaderboardService`/`port.CareerService` existent et sont consommées partout ailleurs ; ces 3 handlers cassent le pattern. Les tests doivent construire un vrai `*service.BootstrapService` au lieu d'un mock léger.
- **Action** : changer le champ pour `port.BootstrapService` / `port.PlayerEngagementService` / `port.LabService` (ajouter ces 2 dernières interfaces si manquantes — `EngagementScoreService` existe déjà mais c'est l'autre service player-scoped qui n'est pas porté).

### [DETTE] `handlers/media.go` (791 L) : logique de FS/URL répandue dans le handler

- **Fichier** : `apps/go-api/internal/api/handlers/media.go` (lignes 553-784, fonctions `countMediaInDir`, `resolveCapturesDir`, `filePathToURL`, `urlToFilePath`, `transformMediaURLs`, `ServeMediaFile`)
- **Extrait** :
  ```go
  func (h *MediaHandler) ServeMediaFile(w http.ResponseWriter, r *http.Request) {
      // 80 lignes de chemin candidat → os.Stat → http.ServeFile, dispatchées sur 2 dossiers
  }
  func (h *MediaHandler) filePathToURL(slug, absPath, capturesBase string) string {
      // 30 lignes de filepath.Rel + concat URL
  }
  ```
- **Problème** : 791 L (>500 L), avec 3 responsabilités mélangées (résolution captures dir, conversion URL↔chemin, ServeFile-like). Pas de testabilité hors handler.
- **Action** : extraire un `MediaPathResolver` (ou `service.MediaURLMapper`) qui encapsule capturesBase + repoRoot et fournit `ToURL(slug, abs) → string` + `ToFilePath(slug, url) → string` + `ResolveServeRoots(slug) → []string`. Le handler ne reste qu'un router et un encodeur.

### [DETTE] Logique de friend-diff dans `handlers/settings.go`

- **Fichier:ligne** : `apps/go-api/internal/api/handlers/settings.go:177-240` (`newFriendsAdded`, `friendGamertagsChanged`, `normalizeGamertag`)
- **Extrait** :
  ```go
  func newFriendsAdded(prev, next []string) []string {
      prevSet := make(map[string]struct{}, len(prev))
      for _, gt := range prev {
          prevSet[normalizeGamertag(gt)] = struct{}{}
      }
      // ...
  }
  ```
- **Problème** : opération pure (set diff case-insensitive) qui n'a rien à faire dans un handler. Le thought_log 2026-04-29 mentionne d'ailleurs 11 sub-tests `TestNewFriendsAdded` dépendant de cette fonction.
- **Action** : déplacer dans `analysis/social.go` (ou `analysis/friends.go`) ou plus simplement dans le service `FriendsOrchestrator`. Le handler ne devrait que parser le PATCH et déléguer.

### [DETTE] `service/match_view_service.go` 1213 L — `GetMatchView` 240 L

- **Fichier:ligne** : `apps/go-api/internal/service/match_view_service.go:161-400`
- **Problème** : la fonction `GetMatchView` orchestre 16+ goroutines parallèles, fait du parsing/projection inline (encounter stats, narrative badges), et l'ensemble dépasse largement les seuils de la grille (80 L par fonction, 500 L par fichier). Le fichier est aussi le plus gros service du repo.
- **Action** : casser en sous-fonctions : `loadAllMatchData(ctx, matchID) → matchBundle` (errgroup) + `assembleMatchView(bundle) → MatchViewResponse`. Sortir les sous-builders narrative dans `match_view_narrative.go` (déjà existant) et déplacer la logique 240 L vers ces helpers.

### [DETTE] `analysis/home.go` 1760 L, 57 fonctions

- **Fichier** : `apps/go-api/internal/analysis/home.go`
- **Problème** : god-file mélangeant 8 responsabilités distinctes (KPIs, hero card, spartan identity, highlights principaux + slides, mode/locale labels, helpers couleur, narrative badges, match score). Fonctions pures, donc pas un BLOQUANT, mais maintenance pénible.
- **Action** : découper en `analysis/home/` package avec `kpis.go`, `hero_card.go`, `spartan_identity.go`, `highlights.go`, `labels.go`. Aucune migration de signature publique nécessaire (toutes restent `analysis.*`).

### [DETTE] `sync/engine.go` — `run()` 190 L, `processMatch()`/`runPostSyncPipeline()` également suspects

- **Fichier:ligne** : `apps/go-api/internal/sync/engine.go:330-519, 538, 738`
- **Problème** : la méthode `run()` dépasse 80 L (190 L) et le fichier (948 L) approche le seuil. C'est un orchestrateur central, donc historiquement difficile à découper, mais l'absence de séparation `runDeltaWindow` / `runFullWindow` / `processBatch` rend le code difficile à tester unitairement.
- **Action** : extraire `runWindow(ctx, opts, isDelta) → (windowResult, error)` + `runPostSyncPipeline` déjà extrait. Cibler 80 L par méthode.

### [DETTE] Co-existence `notify/` (Discord) vs `notifications/` (in-app)

- **Fichiers** : `apps/go-api/internal/notify/*.go` (12 fichiers, Discord webhooks, `log.Printf` au lieu de slog) vs `apps/go-api/internal/notifications/*.go` (in-app DB-backed)
- **Problème** : deux packages avec un nom presque identique mais un rôle très différent (intégration sortante Discord vs domaine de notifications applicatives). Le thought_log 2026-04-29 §6 note d'ailleurs que `notify/` utilise encore `log.Printf` partout — dette pré-existante.
- **Action** : renommer/déplacer `notify/` → `platform/discord/` (c'est un canal sortant, pas un domaine), et migrer ses `log.Printf` vers `slog`. `notifications/` reste le domaine in-app.

### [DETTE] `analysis/sessions.go` n'est pas pur (`time.Now()` non injecté)

- **Fichier:ligne** : `apps/go-api/internal/analysis/sessions.go:171`
- **Extrait** :
  ```go
  func IsSessionPotentiallyActive(lastMatchTime time.Time, cutoffHour int) bool {
      now := time.Now().In(lastMatchTime.Location())
      // ...
  }
  ```
- **Problème** : `time.Now()` lu à l'intérieur d'une fonction d'`analysis/` la rend non-déterministe et difficile à tester sans `time.Sleep`. La grille `arch-rules` interdit les side-effects en `analysis/`.
- **Action** : ajouter un paramètre `now time.Time` à la signature, ou passer un `Clock` (pattern injecté). Le caller (service) fournit `time.Now()`.

### [AMÉLIORATION] `analysis/comeback.go` et `analysis/weapon_correlation.go` loggent via `slog`

- **Fichiers** : `analysis/comeback.go:122, 130, 193` ; `analysis/weapon_correlation.go:145, 202, 218`
- **Problème** : `slog.Debug/Warn` dans `analysis/` introduit un side-effect (I/O console). Acceptable techniquement (slog est stdlib), mais la convention « analysis pur » dérape.
- **Action** : soit accepter explicitement (commentaire dans `arch-rules`), soit injecter un `Logger interface { Debug(msg, attrs...) }` mockable. Pas urgent — purement cosmétique.

### [AMÉLIORATION] `*service.Foo` exposé dans le wrapper Engagement (interface manquante)

- **Fichier:ligne** : `apps/go-api/internal/api/handlers/engagement.go:71-78`
- **Extrait** :
  ```go
  case errors.Is(err, service.ErrEngagementMatchNotFound):
  case errors.Is(err, service.ErrEngagementPvENotSupported):
  ```
- **Problème** : le handler dépend de sentinels `service.Err*` ; le pattern courant est de définir les sentinels à côté de l'interface (`port.ErrEngagementMatchNotFound`).
- **Action** : promouvoir les 2 sentinels dans `port/engagement_score.go` et faire pointer le service dessus.

## Cartographie : exemple d'un appel bien layered vs un appel violant

### Bien layered : `GET /api/v1/players/{slug}/pages/career`

```
chi route → CareerHandler.GetCareer (handlers/career.go:32)
  → ServiceFactory[port.CareerService] (interface)
    → service.CareerService.GetCareerPage (service/career_service.go:79)
      → port.CareerRepository.GetLatestRank, GetXPHistory… (interfaces)
        → platform/duckdb.CareerRepo.GetLatestRank (platform/duckdb/career_repo.go)
          → DuckDB (via pool)
      → analysis.* (helpers purs : aucun ici, pas nécessaire)
```
Chaque couche dépend uniquement de la couche immédiatement plus profonde, et toutes les frontières sont des interfaces `port.*`.

### Violant : `POST /api/v1/players/{slug}/media/upload`

```
chi route → MediaHandler.PostUploadMedia (handlers/media.go:313)
  → MediaUploadContextFactory (closure renvoyant port.MediaService + 5 strings de chemin)
    → port.MediaService.UploadMedia(ctx, req) (interface OK)
      → service.MediaService.UploadMedia (service/media_service.go:~270)
        → sql.Open("duckdb", targetPath)               ← BLOQUANT 1
        → os.Stat / filepath.Walk / filepath.Join     ← FS direct dans service/
        → ops.IndexMedia (back-channel via ops/)
      → handlers/media.go:resolveCapturesDir + filePathToURL + urlToFilePath  ← DETTE 2
        → titlePkg.NewPathResolver(repoRoot)
```
Trois couches qui se chevauchent : le handler résout des chemins, le service ouvre la DB, et `ops/` indexe. La frontière handler↔service↔platform est diluée.

## Suivi recommandé

1. **Sprint « media platformization »** : extraire 2 nouveaux ports (`MediaIndexRepository`, `MediaUploadRepository`) et migrer `service/media_*.go` derrière. Couvre les 3 BLOQUANTS service/duckdb/FS et la DETTE handlers/media.go d'un coup.
2. **Hardening DI handlers** : un PR ciblé qui remplace les 3 derniers `*service.Foo` concrets par `port.*` (Bootstrap, Engagement, Lab) + promotion des sentinels associés. Petit effort, gros bénéfice testabilité.
3. **Suivi `games/halo_infinite/ranks_loader.go`** : déplacer le SQL dans `platform/duckdb` au prochain passage sur la couche métadonnées rangs (rejoint le travail saisons / metadata_repo).

Le reste (god-files `analysis/home.go`, `service/match_view_service.go`, `sync/engine.go`) est de la dette de découpage à traiter au fil de l'eau, sans urgence — fichiers fonctionnels et bien testés selon le project_map.

## Constats hors-axe

- **Charts / pré-shape** (axe 1) : aucun nouvel élément.
- **Multi-titres** (axe 2) : `analysis/home.go:24` contient une constante `homeStaticTitleSlug = "halo_infinite"` avec TODO de promotion à l'adapter, à recouper avec l'axe 2.
- **Logging** (transverse) : `notify/notifiers.go` utilise `log.Printf` (déjà noté dans thought_log 2026-04-29 §6). À couvrir dans un éventuel axe « logging discipline ».
- **Tests** : aucune fonction `analysis/` ne casse les tests — l'impureté `time.Now()` dans `sessions.go:171` survit grâce à des fixtures qui posent des matchs « il y a 30 min ». Une régression serait silencieuse.
