# Journal de decisions — archive 2025-Q2

> Rotation trimestrielle (regle CLAUDE.md).

## [2025-04-25] fix(media): file serving + corrections UI galerie médias

**Statut** : ✅ Complété

**Décision technique** :
- `file_path` en DB = chemin absolu Windows (ex. `C:\Users\...\JGtm\video.mp4`) — inaccessible depuis le navigateur sans endpoint HTTP dédié
- Ajout `GET /api/v1/players/{slug}/media/files/*` dans `handlers/media.go` : sert les fichiers depuis le répertoire captures du joueur avec validation anti-traversal (`path.Clean` + `strings.HasPrefix`)
- `transformMediaURLs()` transforme les chemins absolus en URLs servables dans `GetMediaLibrary` avant sérialisation JSON
- Frontend : date remplace le basename (`6 avr. à 23:43` via `toLocaleDateString + toLocaleTimeString`), suppression badge "Aperçu au survol", GIF/screenshots sans thumbnail utilisent `file_path` (maintenant URL valide), limite home rail 12→20
- Lightbox : basename retiré de la barre d'info

**Résultats** :
- `go build ./...` ✅
- TypeScript : 0 erreur sur `MediaViewer.tsx` et `RecentMediaRail.tsx`

**Note map names** : `map_name_fr` dans `match_registry` doit être rempli par le sync Python pour afficher les noms en français. L'API retourne déjà `COALESCE(map_name_fr, map_name)` — c'est une question de données, pas de code.

## [2025-04-16] Sprint 11 — Accueil/Home + socle provider Halo

**Statut** : Complété

**Décision technique principale** :
- Page Home entièrement read-only depuis DuckDB (Q26/Q27/Q28) — pas de live calls avant Sprint 15.
- `BattlePassResponse` et `ChallengesResponse` retournent `available=false, error_hint="auth_required"` : dégradation explicite et documentée jusqu'au portage MSAL (Sprint 15).
- Provider Halo (`platform/halo/provider.go`) : squelette avec token bucket 60 req/min + retry exponentiel x3. Méthodes `doRequest` prêtes pour Sprint 15.

**Fichiers créés** :
- `internal/domain/home.go` — 12 types domaine (HomeMatchRow, HomeSessionRow, HomeMediaRow, HeroKPIs, HeroTrend, HomeHeroCard, HighlightItem, RecentMatchItem, SessionSummaryItem, RecentMediaItem, HomePageResponse, BattlePassResponse, ChallengesResponse)
- `internal/platform/duckdb/home_repo.go` — HomeRepo (LoadHomeMatches/Q26, LoadHomeSessions/Q27, LoadRecentMedia/Q28)
- `internal/analysis/home.go` — 7 algos stateless (ComputeKPIs, ComputeTrend, BuildHeroCard, BuildHighlights, BuildRecentMatches, BuildSessionSummary, BuildRecentMedia)
- `internal/analysis/home_test.go` — 10 tests
- `internal/platform/halo/provider.go` — HaloProvider skeleton (rate limiter + retry + GetBattlePass/GetChallenges)
- `internal/service/home_service.go` — HomeService.GetHomePage/GetBattlePass/GetChallenges
- `internal/api/handlers/home.go` — 3 handlers GET /pages/home, GET /battlepass, GET /challenges

**Fichiers modifiés** :
- `internal/platform/duckdb/queries.go` — Q26/Q27/Q28 ajoutés
- `internal/port/repository.go` — HomeRepository interface + noopHomeRepo
- `internal/api/server.go` — 3 routes Sprint 11

**Résultats observés** :
- `go build ./...` → PASS (0 erreurs)
- `go test ./internal/analysis/...` → 21/21 PASS (11 sessions + 10 home)
- Commit : `7467e977` sur `feature/go-migration`

**Conclusion** :
Sprint 11 complet. Architecture home : Q26/Q27/Q28 → HomeRepo → HomeService → HomeHandler. Provider Halo prêt pour Sprint 15.
Routes actives :
  GET /api/v1/players/{slug}/pages/home
  GET /api/v1/players/{slug}/battlepass
  GET /api/v1/players/{slug}/challenges
Prochaine étape selon SPRINT_ROADMAP : Sprint 12 (Escouade + Synthèse, ~7-10j). 

---

## [2025-04-16] Sprint 40+41 — Observabilité, scoreboard, weapon parser, healthcheck

**Statut** : Complété

**Décision technique** : Implémenter Sprint 40 (observabilité middleware) et Sprint 41 (scoreboard + weapons + health) en un seul bloc. Sprint 36 T6 (bascule Docker) vérifié déjà fait.

**Sprint 40 — Observabilité (T1+T2+T3)** :
- T1 : `middleware/contract_validate.go` — validation dev-only (LEVELUP_CONTRACT_VALIDATE=1), stdlib JSON, vérifie Content-Type + error shape {code, message, retryable}
- T2+T3 : `middleware/error_tracker.go` — Discord webhook fire-and-forget pour 500, rolling 1-min window error rate alerting >5% avec cooldown 5min
- `config.go` : `DiscordWebhookURL` champ struct + `loadDiscordWebhookURL()` helper (env var + fallback app_settings.json)

**Sprint 41 — Scoreboard + Weapons + Health (T1+T2+T3)** :
- T1 : +10 colonnes dans Q12/ScoreboardRaw/MatchScoreboardRow/Scan/buildTeamTab (shots_fired, shots_hit, damage_dealt, damage_taken, avg_life_seconds, headshot_kills, max_killing_spree, grenade_kills, melee_kills, power_weapon_kills)
- T2 : `halo_client.go` → `GetMatchFilm()` (manifest + chunk download), `backfill_weapons.go` (pipeline complet analysis→DB), `writes.go` → `InsertWeaponKills()` + `MarkWeaponKillsDone()`
- T3 : `HealthResponse` enrichi (+player_count, last_sync_at, uptime, go_version), `BootstrapRepository` interface étendue, `bootstrap_repo.go` → `GetPlayerCount()` + `GetLastSyncAt()`

**Sprint 36 T6** : docker-compose.yml + Dockerfile déjà 100% Go (healthcheck = `/app/levelup-server -health-check`). Marqué ✅.

**Résultats** : go vet OK sur domain, analysis, middleware, port (api/service bloqués par CGo DuckDB Windows — attendu). gofmt appliqué sur tous les fichiers.

**Conclusion** : Sprints 40+41 terminés. Sprint 36 T1 (parity_check = 0 diff) reste ��� — nécessite un run en prod. Prochaine étape : Sprint 42 (Analyse UI avancée + fanout multi-joueur).

---

## [2025-04-17] Mesure locale des coverage gates CI — Sprint 49 closure

### Statut : Complété

### Décision technique
Créer des tests internes (package-level) et CGO pour couvrir les fonctions privées non testées dans handlers, middleware et validation, puis mesurer tous les gates localement au lieu d'attendre la CI GitHub Actions.

### Fichiers créés/modifiés
- `apps/go-api/internal/api/handlers/handlers_extra_test.go` — ajout tests : HomeHandler BattlePass/Challenges NotFound, SquadHandler NotFound/Error, MediaHandler PostUpload 501, MatchHistory Export avec ExportHint
- `apps/go-api/internal/api/handlers/handlers_internal_test.go` — tests fonctions privées helpers (encodeExportToken, formatOptFloat, optStr, etc.)
- `apps/go-api/internal/api/middleware/middleware_internal_test.go` — tests internes : contractResponseWriter, validateErrorShape, errorTrackWriter, discordSimplePayload, checkWindow, notifyError/postDiscord (avec fake webhook), shadowCall (fake Python server), shadowResponseWriter, resolveTitleSlug
- `apps/go-api/internal/validation/compare_cgo_test.go` — tests CGO DuckDB in-memory : listTables, countRows, compareTableCounts, loadMatchIDs, compareMatchIDs, compareBitmasks, ComparePlayerDBs roundtrip + erreurs

### Résultats mesurés

| Package | Avant | Après | Gate | Statut |
|---------|-------|-------|------|--------|
| handlers | 73.7% | **75.4%** | ≥75% | ✅ |
| middleware | 57.5% | **84.6%** | ≥80% | ✅ |
| validation | 52.9% | **88.4%** | ≥70% | ✅ |
| migration | 81.1% | 81.1% | ≥75% | ✅ (déjà OK) |
| platform/duckdb | 75.4% | 75.4% | ≥70% | ✅ (déjà OK) |
| sync | 11.2% | 11.2% | ≥70% | ❌ (dette : nécessite mock API Halo) |

- **Durée suite complète** : ~6s (`go test -tags cgo,integration ./internal/...`)
- **Rapport HTML** : `apps/go-api/coverage.html` généré

### Conclusion
5 gates sur 6 validés localement. Le gate `sync ≥ 70%` reste en dette (11.2%) — nécessiterait une infrastructure de mock API Halo massive, hors scope. SPRINT_ROADMAP.md mis à jour avec les mesures effectives.
