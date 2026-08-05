# Journal de decisions — archive 2025-Q4

> Rotation trimestrielle (regle CLAUDE.md).

## [2025-12-15] feat(sprint16+17): Settings/Setup + Jobs longs persistants

**Statut** : Complété (commit d2ac4565)

**Tâche** : Sprint 16 (Settings/Setup — mutations de configuration + création profil joueur) et Sprint 17 (Jobs longs persistants — JobStore + GET /jobs/{job_id} + POST /sync/initial).

**Décisions techniques principales** :

1. **AppSettings struct avec champs `raw`** — `platform/settings/store.go` charge le JSON brut dans `map[string]json.RawMessage` en plus du struct typé, puis re-merge à la sauvegarde. Garantit que les champs inconnus (ex. `doppler_enabled`) ne sont jamais effacés par un PATCH partiel.
2. **`discord_webhook_url` masqué** — stocké dans `AppSettings.DiscordWebhookURL` (internal) mais jamais sérialisé dans `SettingsResponse` — seulement `DiscordWebhookURLPresent: bool`. Règle de sécurité identique au Python.
3. **JobStore thread-safe + persistance JSON** — `platform/jobs/store.go` utilise `sync.RWMutex` + `data/cache/jobs.json`. À l'init, tous les jobs `running`/`queued` → `interrupted` (le process qui les exécutait est mort). TTL 1h pour les jobs terminaux. `newJobID()` basé sur `UnixNano` (simple et efficace).
4. **Single-flight initial_sync** — `FindActiveInitialSync(playerSlug)` cherche un job non terminal par `JobType == "initial_sync"` et `PlayerSlug == slug`. Retourne 409 si actif.
5. **`POST /setup/players` guards** — 403 `can_self_provision`, 409 `no_halo_identity`, 409 `identity_mismatch`. Compare `strings.ToLower()` pour la case-insensitive. Crée/merge dans `db_profiles.json` v2.1.
6. **Handlers stubs Phase 4** — `PostMediaResetIndex` et `StartInitialSync` créent le job et lancent une goroutine stub. Le vrai moteur sera branché en Sprint 18/19. Commentaire `// TODO Sprint 19` explicite.
7. **Bug pré-existant corrigé** — `citations_service.go` : `Items→Citations` et `TotalMedals→TotalCount` (champs domain inexistants, build cassé depuis Sprint 13).

**Résultats observés** :
- `go build ./...` : **0 erreur** (avec toolchain CGo ucrt64)
- `go vet ./...` : **0 warning**
- 11 fichiers modifiés, 1257 insertions, 86 suppressions

**Prochaine étape** : Sprint 18 — Moteur sync minimal (12 mixins, ~13K LOC Python)

---

## [2025-12-01] Sprint 7 + Sprint 8 — Parity script + Explorer + Match View + KV

**Statut** : Complété

**Décision technique principale** :
- Sprint 7 : Script `scripts/parity_check.py` qui compare les 6 endpoints Phase 1 entre le serveur Go et les golden values JSON. Génère `tests/fixtures/parity_report.json` avec diff tolérant (DEFAULT_FLOAT_TOL=0.01).
- Sprint 8 : Port complet de l'Explorer + Match View. Architecture : repos DuckDB → services purs → handlers chi. KV pairs résolus via `shared.v_killer_victim_full` (vue v6 garantie). Algorithme KV pur dans `internal/analysis/killer_victim.go` pour les cas sans vue.
- `formatDateFRLong` ajouté (distinct de `formatDateFR` de match_history_service.go) pour le format "JJ mois AAAA, HH:MM".

**Fichiers créés** :
- `scripts/parity_check.py` (Sprint 7 — script Python de validation de parité)
- `internal/platform/duckdb/queries.go` — Q17-Q21 ajoutées
- `internal/domain/match_view.go` — types JSON response + types raw DB
- `internal/domain/explorer.go` — ExplorerPlayerQueryRequest, CommonMatchRow, CommonMatchRaw
- `internal/domain/chart/base.go` — HaloColors, OkabeIto, OutcomeColor, PerfColor
- `internal/domain/chart/antagonists.go` — AntagonistBarChartData, DuelChartData, ImpactTimelineData, DominanceChartData
- `internal/analysis/killer_victim.go` — ComputeKillerVictimPairs (algo bisect ±toleranceMS), ComputeAntagonistCounts
- `internal/platform/duckdb/match_view_repo.go` — implémente MatchViewRepository (8 méthodes)
- `internal/platform/duckdb/explorer_repo.go` — implémente ExplorerRepository (GetCommonMatches, ResolveXUIDByGamertag)
- `internal/service/match_view_service.go` — GetMatchView : assemble header (outcome+perf colors), summary (KPIs, medals), combat (weapons, events), team (scoreboard, nemesis)
- `internal/service/explorer_service.go` — GetCommonMatches : résolution gamertag → Q19 → were_teammates
- `internal/api/handlers/match_view.go` — GET /players/{slug}/matches/{match_id}
- `internal/api/handlers/explorer.go` — POST /players/{slug}/pages/explorer/player-query
- `internal/port/repository.go` — MatchViewRepository + ExplorerRepository interfaces + noop impls

**Fichiers modifiés** :
- `internal/api/server.go` — routes Sprint 8 ajoutées (matches/{match_id}, pages/explorer/player-query)
- `internal/service/service_test.go` — 10 nouveaux tests (buildScoreLabel, convertMedals, convertCommonMatches, formatDateFRLong)
- `.ai/go_migration_v2/SPRINT_ROADMAP.md` — Sprints 5-8 marqués ✅

**Résultats observés** :
- `go build ./...` → PASS
- `go test ./internal/service/` → 25/25 PASS (15 anciens + 10 nouveaux)

**Conclusion** :
Sprint 7+8 complets. Phase 1 entière terminée. Phase 2 démarrée (Explorer + Match View opérationnels). Prochaine étape : Sprint 9 (Sessions) ou Sprint 10 (Stats/Séries + perf score).

---
