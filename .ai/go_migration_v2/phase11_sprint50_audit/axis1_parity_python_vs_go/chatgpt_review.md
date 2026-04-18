# Axe 1 · Review ChatGPT — Parité Python↔Go + Streamlit↔React

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | ChatGPT |
| Date du passage | `2026-04-18` |
| SHA Python (worktree `LevelUp`) | `db638c09` |
| SHA Go (worktree `LevelUp-go-migration`) | `93c3cd66` |
| SHA React (même worktree que Go) | `93c3cd66` |
| Durée de l'analyse | `session courante` |
| Corpus analysé (nb fichiers) | `audit ciblé, non exhaustif` |

## Synthèse exécutive (150 mots max)

La branche courante est plus alignée que les audits historiques de Phase 6-9. Côté contrat, les exemptions autrefois centralisées dans `notYetImplemented` ont disparu de `apps/go-api/internal/api/contract_test.go:122`, et le routeur Chi expose bien les endpoints Sprint 49 sensibles comme `citations`, `media`, `synthesis`, `teammates`, `timeseries`, `session-compare` et `last-match/resolve` dans `apps/go-api/internal/api/server.go:163-192`.

Côté UI, toutes les surfaces React majeures sont routées dans `apps/web/src/routes/` et adossées soit à un test unitaire Vitest, soit à une spec Playwright. En revanche, cette passe n'a pas rejoué de diff fonctionnel Python ↔ Go ni Streamlit ↔ React. La parité est donc solide sur le contrat et la présence des surfaces, mais encore partiellement probante sur certains parcours fins côté frontend.

---

## A. Parité API (endpoints HTTP)

> Revue ciblée des endpoints les plus risqués ou historiquement divergents. Cette table ne remplace pas un `parity_check.py` relancé sur 24 endpoints.

| Endpoint | Côté Python (fichier) | Côté Go (fichier) | Parité contrat | Parité payload | Parité status codes | Écart | Classif |
|----------|-----------------------|-------------------|:--------------:|:--------------:|:-------------------:|-------|:-------:|
| `GET /health` | `apps/go-api/api/openapi.yaml:42` | `apps/go-api/internal/api/server.go:90` | ✅ | ✅ | ✅ | Route racine assumée par `servers: /` dans l'OpenAPI. | 🟢 |
| `GET /api/v1/bootstrap` | `apps/go-api/api/openapi.yaml:86` | `apps/go-api/internal/api/server.go:95` | ✅ | ✅ | ✅ | Aucun écart confirmé dans cette passe. | 🟢 |
| `GET /api/v1/players` | `apps/go-api/api/openapi.yaml:112` | `apps/go-api/internal/api/server.go:96` | ✅ | ✅ | ✅ | Aucun écart confirmé dans cette passe. | 🟢 |
| `POST /api/v1/session/context` | `apps/go-api/api/openapi.yaml:133`, `apps/go-api/internal/domain/session.go:42` | `apps/go-api/internal/api/server.go:104`, `apps/go-api/internal/api/handlers/session_context.go:66` | ✅ | ✅ | ✅ | Le contrat actuel expose `available_titles` et `current_title_slug`; pas de divergence OpenAPI constatée. | 🟢 |
| `POST /api/v1/players/{player_slug}/filters/resolve` | `apps/go-api/api/openapi.yaml:165` | `apps/go-api/internal/api/server.go:129` | ✅ | ✅ | ✅ | Aucun écart confirmé dans cette passe. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/citations` | `apps/go-api/api/openapi.yaml:903` | `apps/go-api/internal/api/server.go:167` | ✅ | ✅ | ✅ | Ancienne divergence GET/POST clôturée; plus d'exemption dans `contract_test.go:122`. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/media` | `apps/go-api/api/openapi.yaml:940` | `apps/go-api/internal/api/server.go:171` | ✅ | ✅ | ✅ | Ancienne divergence GET/POST clôturée; plus d'exemption dans `contract_test.go:122`. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/synthesis` | `apps/go-api/api/openapi.yaml:883` | `apps/go-api/internal/api/server.go:163` | ✅ | ✅ | ✅ | Ancienne divergence GET/POST clôturée; plus d'exemption dans `contract_test.go:122`. | 🟢 |
| `GET /api/v1/players/{player_slug}/pages/match-history/export` | `apps/go-api/api/openapi.yaml:305` | `apps/go-api/internal/api/server.go:176` | ✅ | ✅ | ✅ | L'ancienne divergence de méthode n'est plus documentée comme ouverte. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/teammates` | `apps/go-api/api/openapi.yaml:863` | `apps/go-api/internal/api/server.go:180` | ✅ | ✅ | ✅ | Endpoint Sprint 33 bien câblé côté routeur. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/timeseries` | `apps/go-api/api/openapi.yaml:823` | `apps/go-api/internal/api/server.go:184` | ✅ | ✅ | ✅ | Contrat FastAPI-like présent; les routes legacy Go-only restent aussi exposées. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/session-compare` | `apps/go-api/api/openapi.yaml:960` | `apps/go-api/internal/api/server.go:188` | ✅ | ✅ | ✅ | Endpoint présent dans OpenAPI et routeur. | 🟢 |
| `POST /api/v1/players/{player_slug}/pages/last-match/resolve` | `apps/go-api/api/openapi.yaml:980` | `apps/go-api/internal/api/server.go:192` | ✅ | ✅ | ✅ | Endpoint présent dans OpenAPI et routeur. | 🟢 |
| `GET /api/v1/directory/gamertags/search` | `apps/go-api/api/openapi.yaml:333` | `apps/go-api/internal/api/server.go:202` | ✅ | ✅ | ✅ | La route reste enregistrée seulement si `gamertagSvc != nil`; la garantie 503 inconditionnelle n'est pas démontrée ici. | 🟡 |

## B. Parité pages UI (Streamlit → React)

> Cette passe confirme l'existence des surfaces React et leur rattachement route/tests. Elle ne remplace pas une revue écran-par-écran contre Streamlit.

| Page métier | Streamlit (fichier) | React (fichier) | Features couvertes | Features manquantes | Features modernisées | Classif |
|-------------|---------------------|-----------------|-------------------:|:-------------------:|:--------------------:|:-------:|
| Home | `src/ui/pages/home_mission_control.py` | `apps/web/src/features/home/HomePage.tsx:21` | Route, test unitaire, spec Playwright | Diff visuelle 1:1 non rejouée | Shell React data-driven | 🟢 |
| Career | `src/ui/pages/career.py` | `apps/web/src/features/career/CareerPage.tsx:17` | Route, test unitaire, spec Playwright | Diff fonctionnelle fine non rejouée | Navigation React | 🟢 |
| Synthèse | `src/ui/pages/synthesis.py` | `apps/web/src/features/synthesis/SynthesisPage.tsx:151` | Route, test unitaire, spec Playwright | Diff widget-par-widget non rejouée | API page-oriented | 🟢 |
| Match history | `src/ui/pages/match_history.py` | `apps/web/src/features/match-history/MatchHistoryPage.tsx:12` | Route, test unitaire, spec Playwright, export mocké dans MSW | Diff table/colonnes non rejouée | Table React dédiée | 🟢 |
| Match view | `src/ui/pages/match_view.py` + helpers | `apps/web/src/features/match-view/MatchViewPage.tsx:29` | Route, spec Playwright | Pas de test unitaire dédié trouvé | Composition React multi-composants | 🟡 |
| Last match | `src/ui/pages/last_match.py` | `apps/web/src/features/match-view/LastMatchPage.tsx:15` | Route, spec Playwright | Pas de test unitaire dédié trouvé | Intégration route dédiée | 🟡 |
| Explorer | `src/ui/pages/explorer.py` | `apps/web/src/features/explorer/ExplorerPage.tsx:18` | Route, test unitaire, spec Playwright | Diff détail/tri non rejouée | UX React de recherche | 🟢 |
| Session compare | `src/ui/pages/session_compare.py` | `apps/web/src/features/session-compare/SessionComparePage.tsx:93` | Route, spec Playwright | Pas de test unitaire dédié trouvé | Route dédiée dans shell React | 🟡 |
| Sessions (timeseries) | `src/ui/pages/timeseries.py` | `apps/web/src/features/timeseries/TimeseriesPage.tsx:40` | Route, spec Playwright | Pas de test unitaire dédié trouvé | Contrat page-oriented React | 🟡 |
| Squad / Teammates | `src/ui/pages/teammates.py` | `apps/web/src/features/squad/SquadPage.tsx:172` | Route, test unitaire, spec Playwright | Diff sous-modules non rejouée | Surface consolidée React | 🟢 |
| Citations | `src/ui/pages/citations.py` | `apps/web/src/features/citations/CitationsPage.tsx:13` | Route, spec Playwright | Pas de test unitaire dédié trouvé | Intégration React unifiée | 🟡 |
| Media | `src/ui/pages/media_library.py` + `media_v2.py` | `apps/web/src/features/media/MediaPage.tsx:263` | Route, test unitaire, spec Playwright | Diff de pagination/tri non rejouée | Upload et grille React | 🟢 |
| Settings | `src/ui/pages/settings.py` | `apps/web/src/features/settings/SettingsPage.tsx:32` | Route, test unitaire, spec Playwright | Diff formulaire fine non rejouée | Etat draft React | 🟢 |
| Setup wizard | `src/ui/pages/setup_wizard.py` | `apps/web/src/features/setup/SetupPage.tsx:417` | Route, test unitaire, spec Playwright | Diff pas-à-pas non rejouée | Onboarding React | 🟢 |
| Changelog | `N/A Python` | `apps/web/src/features/changelog/ChangelogPage.tsx:10` | Route dédiée | N/A | Nouvelle feature documentée | 🟢 |

## C. Parité algorithmes métier (7 cœurs)

| Algorithme | Python (module) | Go (package) | Golden values vertes ? | Écart observé | Classif |
|------------|-----------------|--------------|:----------------------:|---------------|:-------:|
| Performance score | `src/analysis/performance_score.py` | `apps/go-api/internal/analysis/performance_score.go` | `non relancé ici` | Aucun écart statique constaté dans cette passe; revalidation runtime non faite. | 🟡 |
| LUSR / CSR | `src/analysis/skill_rating.py` | `apps/go-api/internal/analysis/skill_rating.go`, `apps/go-api/internal/sync/skill_rating.go` | `non relancé ici` | Aucun écart statique constaté dans cette passe; revalidation runtime non faite. | 🟡 |
| Sessions | `src/analysis/sessions.py` | `apps/go-api/internal/analysis/` | `non relancé ici` | Aucun écart statique constaté dans cette passe; revalidation runtime non faite. | 🟡 |
| Citations | `src/analysis/...` | `apps/go-api/internal/api/handlers/citations.go` + services associés | `non relancé ici` | Aucun écart statique constaté dans cette passe. | 🟡 |
| Killer/victim | `src/analysis/killer_victim.py` | `apps/go-api/internal/analysis/` | `non relancé ici` | Aucun écart statique constaté dans cette passe. | 🟡 |
| Weapon parser | `src/analysis/weapon_parser.py` | `apps/go-api/internal/analysis/weapon_*.go` | `non relancé ici` | Aucun écart statique constaté dans cette passe. | 🟡 |
| Spawn / comeback detection | `src/analysis/...` | `apps/go-api/internal/analysis/` | `non relancé ici` | Non vérifié directement dans cette passe. | 🟡 |

## D. Parité sync / backfill

| Flux | Python | Go | Écart | Classif |
|------|--------|----|-------|:-------:|
| Sync delta | `scripts/sync.py` + `src/data/sync/engine.py` | `apps/go-api/internal/sync/engine.go` | Aucun écart statique confirmé; non rejoué ici. | 🟡 |
| Backfill (tous flags SyncScope) | `scripts/backfill_data.py` + `scripts/backfill/` | `apps/go-api/internal/sync/backfill.go` + flags associés | Aucun écart statique confirmé; non rejoué ici. | 🟡 |
| Write lease | `src/data/sync/...` | `apps/go-api/internal/platform/duckdb/pool.go` | Non revalidé en exécution durant cette passe. | 🟡 |
| Bitmask `backfill_completed` (18 bits) | historique Python | `apps/go-api/internal/sync/backfill_flags.go` | Aucun écart statique confirmé; non rejoué ici. | 🟡 |

## E. Parité CLI / scripts opérationnels

| Script Python | Équivalent Go/CLI | Couvert ? | Écart | Classif |
|---------------|-------------------|:---------:|-------|:-------:|
| `scripts/sync.py` | `cmd/levelup` + sync backend | partiellement | Équivalence documentaire plausible, non rejouée. | 🟡 |
| `scripts/backup_player.py` | `apps/go-api/internal/ops/backup.go` | partiellement | Non rejoué dans cette passe. | 🟡 |
| `scripts/restore_player.py` | `apps/go-api/internal/ops/restore.go` | partiellement | Non rejoué dans cette passe. | 🟡 |
| `scripts/backfill_data.py` | `apps/go-api/internal/sync/backfill.go` | partiellement | Non rejoué dans cette passe. | 🟡 |
| `scripts/check_env.py` | `apps/go-api/internal/validation/gate.go` + CLI | partiellement | Non rejoué dans cette passe. | 🟡 |

## F. Parité données (schémas DuckDB)

> Cette passe n'a pas rouvert les DBs. Aucun écart statique n'a été observé dans les migrations ou la documentation courante, mais la validation reste documentaire ici.

| Table | Colonnes identiques ? | Types identiques ? | Index/vues identiques ? | Écart | Classif |
|-------|:---------------------:|:------------------:|:-----------------------:|-------|:-------:|
| `match_registry` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `match_participants` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `medals_earned` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `killer_victim_pairs` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `xuid_aliases` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `weapon_kills` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `player_match_enrichment` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `match_skill_rank` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `sessions` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `pve_match_stats` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `weapon_labels` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| `career_ranks` | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |
| Vues (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`) | `non revalidé` | `non revalidé` | `non revalidé` | Pas de divergence statique identifiée dans cette passe. | 🟡 |

## G. Parité i18n

| Aspect | Python | Go/React | Écart | Classif |
|--------|--------|----------|-------|:-------:|
| Nombre de langues | 14 (scope documentaire) | non revérifié dans cette passe | Preuve insuffisante côté React pour conclure 1:1. | 🟡 |
| Source de traduction runtime | DuckDB + traductions UI Python | non revérifié dans cette passe | Pas de divergence statique confirmée, mais pas de validation runtime relancée. | 🟡 |
| Résolution `Accept-Language` | attendue | non revérifiée dans cette passe | Non testé ici. | 🟡 |

## H. Parité observabilité & erreurs

| Aspect | Python | Go/React | Écart | Classif |
|--------|--------|----------|-------|:-------:|
| Notifier Discord | historique Python | `apps/go-api/internal/notify/` | Aucun écart statique confirmé ici. | 🟢 |
| Logs structurés | attendu | `apps/go-api/internal/api/middleware/slog_logger.go` | Présents côté Go. | 🟢 |
| Taxonomie erreurs provider | doc Python/Go | `HALO_PROVIDER_ERROR_TAXONOMY.md` + middleware/services | Aucun écart statique confirmé ici. | 🟢 |

## I. Modernisations volontaires (🟢)

> Lister les écarts Python → Go / Streamlit → React qui sont intentionnels (amélioration, simplification, suppression). Chaque ligne doit avoir une motivation.

| Modernisation | Motivation | Remplacement | Impact utilisateur |
|---------------|------------|--------------|--------------------|
| `Changelog` React dédié | Nouvelle surface produit, pas une dette de parité | `apps/web/src/features/changelog/ChangelogPage.tsx:10` + `apps/web/src/routes/changelog.tsx` | Amélioration, pas de régression Python |
| Routes legacy Go-only encore exposées (`/pages/sessions`, `/pages/stats/query`, `/pages/squad`) | Compatibilité et transition progressive | Contrats FastAPI-like ajoutés en parallèle dans `openapi.yaml` | Impact utilisateur nul si le frontend consomme les routes cibles |
| `SessionContextResponse` title-aware | Support multi-titres | `apps/go-api/internal/domain/session.go:42-48` | Réponse enrichie côté shell React |

## J. Récap classifications

| Niveau | Nombre d'items | Liste des IDs (ou courtes descriptions) |
|--------|:--------------:|-----------------------------------------|
| 🔴 Bloquant | 0 | — |
| 🟠 Majeur | 0 | — |
| 🟡 Mineur | 10 | preuve partielle UI/algorithmes, route gamertag conditionnelle |
| 🟢 Toléré | 8 | exemptions contrat closes, modernisations React/Go, observabilité présente |

## K. Observations libres

Le point le plus fort de cette passe est négatif au bon sens du terme : les anciens constats de rupture de contrat ne sont plus visibles dans l'état courant du dépôt. `apps/go-api/internal/api/contract_test.go:122` ne conserve plus aucune exemption, et `apps/go-api/internal/api/server.go:163-192` expose explicitement les routes Sprint 49 sensibles. En revanche, le Sprint 50 ne peut pas être considéré comme clôturé sur la seule base de cette review : aucune relance de `parity_check.py` n'a été faite ici, et la comparaison Streamlit ↔ React reste probante surtout par existence de surfaces/routes/tests, pas par diff fonctionnel complet.

---

**Fin du template axe 1.**
