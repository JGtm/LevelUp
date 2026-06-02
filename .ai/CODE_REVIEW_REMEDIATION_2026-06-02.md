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

## 5. Lot A — correction/robustesse backend (passe dédiée, débloquée)

Items initialement différés « zone concurrente » (`internal/sync`/`scheduler`), traités une fois le travail parallèle confirmé comme recherche seule. Investigation multi-agents (7 specs vérifiées) → implémentation → **vérification adversariale** (7 sceptiques, 0 bug bloquant, 3 nits low corrigés).

| Réf | Item | Commit |
|---|---|---|
| A1 | Réconciliation post-Drain : `InsertedMatchIDs`/`MatchesInserted` reconciliés vs `match_registry` si Drain échoue (plus de post-sync ~285s sur matchs fantômes) | `693fb1041` + `e98b4ae21` |
| A2 | `refreshAggregates` → `(created, failed, errors.Join)` + warn agrégé corrélé | `693fb1041` |
| A3 | **Retry-After** (429/503) honoré + backoff exponentiel borné + métriques expvar `levelup.auth_pool.*` | `d53480fdc` + `e98b4ae21` |
| A4 | `persistPlayerRecordsLegacy` : détection one-shot (`sync.Once`, ctx découplé) + WARN `legacy_player_records_upsert_used` | `693fb1041` + `e98b4ae21` |
| A5 | `GetChallenges` synchrone + suppression des 2 méthodes async `Background()` (write-after-CloseAll) | `693fb1041` |
| A6 | **Anti-régression URL `xuid(NNN)` ACTIVE** (`TestContract_…_V1` via `RunDelta` + capture mock) + combat guard dégardé | `3331f1f3c` |
| A7 | `auto_sync` précondition `os.Stat` via slug du profil (garde seulement) | `693fb1041` |

**Correctifs post-vérification** (`e98b4ae21`) : A1 bénéfice-du-doute sur erreur de requête (+ 4 tests) ; A3 métriques comptées seulement si (ré)appliqué + `last_cooldown_seconds` sur extension ; A4 sonde sur `context.Background()`.

**Reste différé du lot A** : `CrossPlayerDedup_V1` (confidence moyenne — dépend du couplage xuid↔PlayerId du mock, à faire avec un `statsBody` explicite).

---

## 6. Lot B — découpe des god-files (>500L) + dédups frontend

### a) Dédups frontend + centralisation query keys
| Réf | Item | Commit |
|---|---|---|
| B-dedup | `outcomeKey` canonique (`lib/outcome-color.ts`) — 3 consommateurs `'draw'` dédupliqués (variante `'tie'` laissée : key-set différent) | `3a0880777` |
| B-dedup | `formatDateShort`/`formatNumber` (`components/charts/_utils.ts`) → réexport + délégation `lib/formatters` (fallback `'-'` préservé) | `9f1ac133a` |
| B-keys | Query keys à risque de drift centralisés en factories (prestige `challengeKeys.list`/`prestigeKeys.meAll` ; `adminKeys` ; `profileKeys.campaignAll` ; helpers `*All` sur le factory central pour notifications/coach/match-history/teammates). Clés feature-privées mono-site laissées (0 drift) | `24a27a0c6` |

### b) Découpe god-files Go (extraction de blocs contigus, **0 changement fonctionnel**, `goimports` + build/vet/gofmt/test verts par fichier)
| Fichier | Avant→Après | Fichiers créés | Commit |
|---|---|---|---|
| `api/handlers/media.go` | 914→409 | `media_serve` + `media_paths` + `media_upload` | `c74516142` |
| `platform/duckdb/player_matches_repo.go` | 1024→325 | `_projection` + `_scan` + `_loaders` | `e8b485871` |
| `service/squad_service_v2.go` | 1006→480 | `_aggregates` + `_intersect` | `c379c5806` |
| `ops/seed.go` | 910→350 | `seed_citation_data` (data table, exemptée 80L) | `a1f36f978` |
| `analysis/squad_breakdown.go` | 903→412 | `_canonical` + `_heatmaps` | `a1f36f978` |
| `platform/duckdb/media_repo_q37_pipeline.go` | 829→233 | `_enrich` + `_transform` | `847ea3002` |
| `platform/duckdb/squad_repo.go` | 762→358 | `_synthesis` + `_mapstats` | `847ea3002` |
| `platform/duckdb/queries_match.go` | 677→447 | `queries_match_detail` | `d60b086ba` |
| `platform/duckdb/queries_home_citations.go` | 676→455 | `queries_citations` | `d60b086ba` |
| `platform/duckdb/queries_career.go` | 618→343 | `queries_career_encounters` | `d60b086ba` |
| `config/config.go` | 671→405 | `config_players` + `config_settings` | `d60b086ba` |
| `platform/lab/provider.go` | 760→324 | `_assets_medals` + `_contracts` | `dc77552d5` |
| `service/media_service.go` | 688→407 | `_upload` + `_build` | `dc77552d5` |
| `service/explorer_service.go` | 670→370 | `_target` + `_convert` | `dc77552d5` |

→ **Tous les fichiers >700L non-sensibles sont désormais <500L.** `seed_citation_data.go` (565L) reste >500 : c'est un littéral de données pur (`[]CitationMapping{...}`), exemption « data table » documentée en tête de fichier.

### c) God-files restants — différés avec raison
- **Refactor comportemental (pas un déplacement mécanique)** : `sync/engine.go` (776) — la méthode `run()` fait ~500L ; la découper = extraire des sous-fonctions du cœur de la boucle sync (risque > déplacement de fichier). `api/server.go` (977) — `NewRouter` est une fonction géante d'assemblage DI/routes. À faire en passe dédiée supervisée.
- **Zone migration (ordre d'`init()` = ordre d'exécution, cf. `registry.go:38`)** : `migration/steps_shared.go` (982), `steps_metadata.go` (640), `steps_player.go` (571). Ces fichiers sont des **manifestes déclaratifs** (suites de `Register(Migration{…})`) ; l'ordre d'enregistrement pilote l'ordre d'exécution au boot. Les splitter risque une régression d'ordre (échec schéma en prod) pour un fichier conceptuellement « data table » (comme `seed_citation_data`). Différé sauf besoin avéré + garde-rail d'ordre.
- **DI/wiring** : `api/registry_pages.go` (725) — assemblage de dépendances ; découpe par domaine possible mais peu de gain de lisibilité, à cadrer.
- **Cohérents 500-700L** : `duckdb/db.go` (624), `progression/profile/service.go` (605), `service/session_page_service.go` (604), `prestige/service.go` (567), `api/post_sync_progression.go` (567), `analysis/match_impact.go` (551), `analysis/skill_rating.go` (531), `duckdb/filters_repo.go` (544), `engagement_score_repo.go` (532), `home_repo_skill_peak.go` (501), `handlers/{sync_handler,backfill,prestige}.go`, `halo/provider.go` (545), `ops/{seed_demo,seed_demo_media}.go`, `sync/{skill_v2_shadow,citations,backfill,backfill_weapons,skill_formula_sim}.go`, `persist/persist_sink.go`, `scheduler/auto_sync.go`, `watcher/daemon.go`. À découper au cas par cas si une responsabilité distincte émerge (pas de découpe mécanique d'un fichier cohérent juste pour le compteur).

### d) Lot zones sensibles (validé utilisateur) + i18n H-D4 (pilote)
Splits **zones sensibles** (sync/persist) — extraction de blocs contigus, vérifiés `go test -race` :
| Fichier | Avant→Après | Fichiers créés | Commit |
|---|---|---|---|
| `sync/skill_rating.go` | 773→341 | `_trueskill` + `_composite` + `_preview` | `d4bcd4b0e` |
| `sync/engine_postsync.go` | 773→368 | `_scoring` + `_csr` | `aaec5f99a` |
| `persist/shared_social_persister.go` | 765→408 | `_batch` (Persist + persist* internals) | `2aaee5ef1` |
| `sync/performance.go` | 672→447 | `_helpers` (rankPerf + metrics + loadHistory) | `2aaee5ef1` |

**H-D4 i18n — pilote Timeseries livré** (`3386513d2`) : 62 ternaires `locale === 'en' ? … : …` inline des 3 onglets (summary/progression/distributions) migrés vers le manifest TOML `timeseries` (`t('timeseries.*')`, clés typées `keyof M` ⇒ typo = erreur tsc). 48 clés ajoutées + réutilisation des existantes ; prop `locale` retirée des 2 onglets devenus indépendants de la locale.
- **Reste H-D4 (différé, ~74 occurrences sur ~38 fichiers)** : à traiter **par feature selon sa convention existante** — TOML manifest (`match-view`, `home`, `explorer`…) OU dict hand-written `i18n.ts` (`ascension`, `notifications`, `settings`, `help`, `squad`…). **Attention** : une fraction des `locale === 'en'` ne sont **pas** des strings traduisibles (normalisation de locale `? 'en' : 'fr'`, code Intl `'en-US'`/`'en-GB'`, sélection de fonction, flags booléens) ni des cibles (occurrences **dans** les `i18n.ts` eux-mêmes). La **règle ESLint** doit donc cibler spécifiquement les ternaires *string-littéral* en JSX, pas tout `locale === 'en'` (sinon faux positifs) — règle custom non triviale, à écrire après migration du gros des `.tsx`.

---
*Build complet `go build ./internal/...` + `go vet ./internal/...` verts ; `tsc -b` vert. Lot A : `go test -race` verts. Lot B (god-files non-sensibles + dédups + query keys) : build/vet/gofmt/test par package verts. Lot zones sensibles : `go test -race ./internal/sync` (suite complète) + `./internal/persist` verts. i18n timeseries : typecheck + lint + vitest (37) verts. Commits : passe initiale 29a9bfbd5 → d762443ac (11) ; lot A 693fb1041 → e98b4ae21 (4) ; lot B 3a0880777 → dc77552d5 ; sensibles + i18n d4bcd4b0e → 2aaee5ef1.*
