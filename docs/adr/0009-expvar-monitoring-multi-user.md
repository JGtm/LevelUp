# ADR 0009 — Expvar monitoring (multi-user, sans Prometheus)

**Status** — Proposed (2026-04-29). Triggered by code review axe 8 (BLOQUANT `internal/observability` mort) + contexte produit multi-user acté.

**Deciders** — Guillaume (GS).

## Context

Le PLAN_META_FOUNDATIONS_GO §4.7 a explicitement écarté Prometheus / OpenTelemetry pour cette itération du projet. À la place, un squelette de monitoring perf basique a été scaffoldé dans `internal/observability/expvar_metrics.go` :

- Compteurs atomiques thread-safe (`atomic.Int64` + `sync.Map`)
- 4 catégories de métriques prévues :
  - `service_duration_ms{service,status}` — moyenne + max + sum + count
  - `repo_query_duration_ms{query}` — même structure
  - `cache_hit_ratio{cache}` — compteurs hits + misses
  - `error_count{service}` — compteur simple
- Helpers `IncCounter`, `AddInt`, `RecordDurationMS` (à compléter selon scope)
- Publication sous la clé `"levelup"` du JSON renvoyé par `/debug/vars`

**Au 2026-04-29 : code mort**. Aucun service / repo n'appelle ces helpers, `/debug/vars` n'est pas monté, le package est marqué « scaffolding then forget » (axe 8).

Parallèlement, `internal/api/middleware/error_tracker.go` (Sprint 40 T2+T3) implémente un alerting Discord sur HTTP 500 + alerte si taux d'erreur > 5% sur fenêtre 1 min. **Désactivé en dur** ligne 67-69 avec commentaire explicite : « DÉSACTIVÉ EN DUR — l'alerting Discord 500/taux d'erreur n'est pas souhaité ».

**Contexte produit (acté 2026-04-29)** : l'app est destinée au multi-utilisateur (proposé en UI). Cela change la balance coût/bénéfice du monitoring perf basique : sur instance solo, dispensable ; sur instance multi-user, nécessaire pour identifier les hot paths et requêtes lentes sans dépendre d'un stack Prometheus.

## Decision

**1. Brancher `internal/observability/expvar_metrics.go`** :

Instrumentation cible au moment de la première itération :

| Catégorie | Hot paths instrumentés | Effort |
|---|---|---|
| `service_duration_ms` | 6 services principaux (Home, Career, Synthesis, MatchView, Squad, Timeseries) | 2h |
| `repo_query_duration_ms` | 3-5 queries critiques (chargement match registry, scan match_participants, agrégat squad) | 2h |
| `cache_hit_ratio` | si caches en place (TanStack côté front non pertinent ; cache repo Go si présent) | 1h |
| `error_count` | par service principal (incrémenté dans les `slog.ErrorContext`) | 1h |

`/debug/vars` exposé **derrière auth admin** (middleware `RequireAdmin` à créer ou réutiliser). Convention de nommage `<categorie>_<sous_cle>` en snake_case (cf. doc package).

**2. Supprimer `internal/api/middleware/error_tracker.go`** :

La désactivation explicite (« pas souhaité ») est une décision tranchée. On consolide en supprimant le code mort plutôt que de le laisser dormir avec un `//nolint:unused`. Si du jour au lendemain un alerting Discord 500 devient pertinent, il sera rebrandé en module dédié distinct (pas un middleware désactivé en dur).

**3. Documentation** :

Créer `internal/observability/README.md` listant les métriques exposées et comment les consulter (curl `/debug/vars` + jq query exemples).

## Consequences

### Positive

- Visibilité minimum sur la perf prod en multi-user, sans dépendance Prometheus / OpenTelemetry.
- Détection rapide des hot paths et queries lentes avant qu'elles ne deviennent visibles utilisateur.
- Code mort `error_tracker` éliminé (anti-pattern « Dead code museum »).
- Squelette `observability` valorisé après plusieurs mois d'inutilisation.

### Negative

- Instrumentation manuelle à maintenir (pas de monitoring automatique des nouveaux endpoints).
- expvar JSON peut grossir si les sous-clés explosent — convention de nommage stricte requise.
- Pas de séries temporelles ni d'alerting (juste une photo instantanée). Pour ce besoin, il faudra Prometheus à terme.
- 1 jour effort + maintenance continue à chaque ajout de hot path.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) Adopter Prometheus / OpenTelemetry** | Surdimensionné pour le besoin. Stack à déployer/maintenir. Le projet est en pré-production multi-user, pas en SaaS scale. |
| **B) Supprimer `internal/observability` (instance solo)** | Contexte multi-user invalide cette option. |
| **C) Garder le code mort « pour plus tard »** | Anti-pattern « Dead code museum » CLAUDE.md. Crée de la confusion. |
| **D) Réactiver `error_tracker` Discord** | Décision explicite « pas souhaité » — on ne ré-ouvre pas. |

## References

- Code review : `axe-8-logs.md` (BLOQUANT 2 — observability mort + error_tracker désactivé).
- Plan meta : `.ai/V7/PLAN_META_FOUNDATIONS_GO.md` §4.7 (rejet Prometheus/OpenTelemetry).
- Plan d'action : `PLAN_ACTION.md` P1.5 (décision actée), P8.3 (exécution).
- Code : `apps/go-api/internal/observability/expvar_metrics.go`, `apps/go-api/internal/api/middleware/error_tracker.go` (à supprimer).
