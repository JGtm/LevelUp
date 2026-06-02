# Remédiation revue de code — 2026-06-02

> Suivi des corrections appliquées suite à [.ai/CODE_REVIEW_2026-06-02.md](CODE_REVIEW_2026-06-02.md).
> Branche : `feat/skill-progression-magnitude-scale`. Tout buildé/vet/testé (CGO) ; frontend typecheck/lint/vitest.

## 1. Traité (P0 / P1 — sessions précédentes)

| Réf | Item | Commit |
|---|---|---|
| C1 | RealIP non borné → bypass LoopbackOnly (`/_diag` exposés) | `1d77be546` (+ test) |
| H-A1..A4 | Garde-fou boot prod (`config.Validate`/`SecurityWarnings`, `LEVELUP_ENV`, `TrustProxyHeaders`) ; token clair supprimé | `1d77be546` |
| H-B1 | Rebuild `match_participants` transactionnel + recovery orphelin | `1380e9702` |
| H-B2 | Restore : fin du no-op silencieux + transaction + échappement chemin | `1380e9702` |
| H-B3 | WARN boot si `LEVELUP_PERSIST_BATCH=0` (foot-gun ART) | `18576a407` |
| H-C1 | Data race `DB.sqlDB` → `atomic.Pointer` + test `-race` | `50a30115c` |
| H-D1/D2/D3 | i18n MatchCard + engagement → manifests ; ChartCard réactif palette | `b8c2d4938` |
| #46 | En-têtes de sécurité HTTP (middleware SecurityHeaders) | `1d23f2e96` |
| #69 | Fix healthcheck port (`LEVELUP_API_PORT`) | `1d23f2e96` |

## 2. Traité (cette passe autonome)

| Réf | Item | Commit |
|---|---|---|
| L (analysis/ops/service/domain) | Dead code (`max`, `fmtPct`, `MatchViewRawRow`), magic `outcome==2` → `domain.OutcomeWin`, emojis healthcheck → `[OK]/[KO]`, backup `compression_level` réel | `29a9bfbd5` |
| M duckdb/efficiency | TZ canonique `player_matches_repo` (projection/filtre/ORDER BY via `COALESCE(start_time_utc,…)`) | `47f6f5b67` |
| L duckdb | damage `math.Round` ; suppr consts SQL mortes `Q29TopTeammates`, `Q26gHomePlaylistRanks` | `47f6f5b67` |
| M api/security | Plus de `err.Error()` interne au client sur 5xx (fix au point unique `writeError`/`httpError`, couvre ~77 sites) | `e72617295` |
| L api/multi-title | Commentaire dégradation capability corrigé (503, pas 404) | `e72617295` |
| M concurrency | `MatchQueue` : `seen` marqué après enqueue (plus de perte de matchs) | `8877b3882` |
| M concurrency | `Daemon.running` → `atomic.Bool` + garde double-Start + `cancel`/`rootCtx` sous `playersMu` | `8877b3882` |
| M concurrency | FSM `StateEnteredAt()` synchronisé (provider ne lit plus le champ à nu) | `8877b3882` |
| M security / L docker | Doc variables prod requises (`.env.local.example`) + note isolation volumes docker | `70b92fd4a` |
| M fe-features | Suppr `console.log` debug SynthesisPage ; suppr composants morts `ChallengesCarousel`, `SynthesisCombatProfileSection` | `f4af41b40` |
| L fe-lib | `themeColors.isDark` lisait classe `dark` (jamais posée) → `data-theme` | `f4af41b40` |
| M multi-title | `data_health_check` via PathResolver (plus de `filepath.Join("data","titles",…)` en dur) | `d762443ac` |

## 3. Différé — avec raison (rien d'abandonné en silence)

### a) Refactors lourds (impératif : risque de régression en run autonome non supervisé → lot dédié)
- **Découpe des god-files >500L** : `handlers/media.go` (914), `service/home_service.go`, `domain/match_view.go` (810), `migration/steps_shared.go` (982), `ops/media.go` (952), `ops/seed.go` (910), `platform/duckdb/player_matches_repo.go` (1015), `sync/engine*.go`, `skill_rating.go`, `persist/shared_social_persister.go`. Découpage pur, sans changement fonctionnel — à faire fichier par fichier avec relecture.
- **`api/post_sync_deltas.go`** : extraire `PlayerSnapshotRepo` + table de règles (réduit 2 god-functions).
- **Frontend SRP** : décomposer `SynthesisPage`/`SynthesisOverviewSection`, `SessionComparePage` (~750L), `SquadLayout` (hooks `useSquadSessionSelection`) ; migrer SynthesisPage vers `useLocalFilterBar`.
- **CLI** : consolider ~115 CLIs sous `cmd/levelup` + namespace `cmd/diag/`, helper `cmd/internal/clidb`+`clilog`, archiver les diag de migration obsolètes, nettoyer les `.exe` du working tree.
- **H-D4** : migrer les ~140 ternaires `locale === 'en'` inline vers les manifests + règle ESLint dédiée.
- **Query keys** : centraliser les littéraux dispersés (~10 features) dans `lib/query/keys.ts` + règle ESLint ; trancher `userId` vs `playerSlug` pour prestige.
- **Dédup** : `outcomeKey` central (9 fichiers front) ; `formatNumber/formatDateShort` (_utils vs lib/formatters) ; heatmaps/top-weeks/`median` (analysis) ; helper KPI/home god-functions >80L.
- **analysis purity** : sortir les builders SQL (`sql_fragments.go`, `identity.go`) vers `platform/duckdb` ; retirer `log/slog` de `comeback.go`/`weapon_correlation.go`.

### b) Zone d'un agent concurrent (impératif : `internal/sync`, `internal/scheduler/auto_sync.go` activement édités en parallèle — éviter le conflit)
- `MatchesInserted` incrémenté avant ACK async (engine_batch_path) ; couplage engine↔persist (`BatchPersister`) ; god-files engine.
- `persistPlayerRecordsLegacy` (check boot + WARN) ; `aggregates.go` avale les erreurs de rebuild de vues.
- `auto_sync` : test `-race` parallélisme, expiration des flags, multi-titres (PlayerDBPath par slug) ; `contract_test` de-skip + datasets e2e hétérogènes + capture `player` du mock + tag `combat_write_guard`.
- **Retry-After** (429/503) : nécessite de plumber le header via le type `HTTPError` (sync) + changer la signature `OnHTTPError` (interface) — traverse `internal/sync`.
- `persist_sink` goroutines fire-and-forget (domaine lifecycle W6) ; coordinator dedup asymétrique ; métriques WAL worker.

### c) Décision de design / produit (à trancher par l'équipe)
- **TitleDataAdapter dormant** : `Capabilities()` annonce du supporté pour des `Load*` stubs ; champs `dataAdapter` injectés non lus ; DI `registry.go` hard-gate `slug==halo_infinite`. Choix : finir le câblage canonique OU retirer la façade morte. (Cohérence des 2 systèmes de capabilities idem.)
- **Ownership 404 sur slug inconnu** : changement de comportement d'access-control ; la revue elle-même le conditionne à un test croisé — à valider avec un test d'intégration avant bascule.
- **Timezone DuckDB globale → par-requête** : large, lié à la dette `first_joined` (à cadrer avant prod).
- **Pool handles DuckDB sans éviction** : éviction LRU/TTL + plafond configurable (coordination ref-count délicate).
- **ReplaceAttr redaction logs** : nécessite un handler wrapper (le format compact ne supporte pas ReplaceAttr) ; faible valeur (aucun secret loggé aujourd'hui — audit revue).
- **Migrations sans verrou inter-process** : pré-requis d'orchestration de déploiement (1 seul writer) — documenté.

### d) Nits faible valeur / risque > valeur (laissés tels quels, documentés)
- `ObjectiveScores` pré-check `information_schema` (fonctionnellement correct) ; `perfect_kills` sous-requête corrélée (correcte ; CTE = risque dans une requête de 1015L) ; `ExcludeFriends` `NOT IN`→`NOT EXISTS` ; requêtes joueur non bornées (besoin d'une décision produit sur la fenêtre).
- `RadarChart` memo `buildOption` (perf LOW, ambigu) ; `RankProgressGauge` hex (exception `color-allow` structurelle SVG tolérée CLAUDE.md §20) ; `safeDestPath` TOCTOU ; `Radar []any` typing ; `sql.ErrNoRows` par string dans l'adapter ; `SteamPoller` clé API en URL (code mort non câblé) ; `cmd/get-token` (dev tooling).

## 4. Écarté — finding erroné ou obsolète
- **Consts SQL « mortes » `shared.` (queries_squad.go:80,354)** : `Q30SquadMatches`/`Q42MapStatsForSquadTemplate` sont **utilisées** ; leur préfixe `shared.` est une référence **ATTACH valide** en contexte squad (pas l'anti-pattern cross-DB). Seul `Q29TopTeammates` était réellement mort (supprimé).
- **`seed_demo_media` layout `data/players/`** : le conteneur démo monte **intentionnellement** le layout plat legacy (`docker-compose.yml: ./data/demo/players:/app/data/players:ro`). Le passer en `data/titles/{slug}/players/` casserait la démo — finding inapplicable à ce contexte.
- Déjà traités en P1 : en-têtes HTTP, healthcheck port, titres engagement par défaut, alerte CORS (couverte par `config.Validate`).

---
*Build complet `go build ./...` + `go vet ./internal/...` verts ; `tsc -b` vert. 10 commits cette passe (29a9bfbd5 → d762443ac).*
