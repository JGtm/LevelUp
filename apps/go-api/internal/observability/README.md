# observability

P8.3 (revue 2026-04-29, ADR 0009) — monitoring expvar minimal pour multi-user.

## Pourquoi expvar (pas Prometheus)

Voir `docs/adr/0009-expvar-monitoring-multi-user.md`. Synthèse :

- **Auto-hébergé multi-user** sans cluster d'observabilité dédié.
- expvar est dans la stdlib Go (`/debug/vars`) → 0 dépendance externe.
- Suffisant pour les besoins actuels (latence services, compteurs erreurs).
- Migration vers Prometheus possible plus tard si on déploie sur K8s.

## Métriques exposées

Compteurs publiés sous la clé `levelup` dans `/debug/vars` :

### Durations (`*_duration_ms`)

Émises via `observability.RecordDurationMS(name, ms)` à la fin de chaque appel
service. Chaque entrée stocke `count`, `sum_ms`, `avg_ms`, `max_ms`.

| Métrique | Source |
|---|---|
| `home_get_page` | `service.HomeService.GetHomePage` |
| `career_get_page` | `service.CareerService.GetCareerPage` |
| `match_view_get` | `service.MatchViewService.GetMatchView` |
| `stats_get_page` | `service.StatsService.GetPage` |
| `timeseries_get_page` | `service.TimeseriesService.GetPage` |

### Compteurs (via `IncCounter` / `AddInt`)

Au moment de la rédaction, aucun compteur custom n'est défini hors des
durations. Les `error_count` sont émis via `slog.ErrorContext` (handler
ContextHandler attache `request_id`).

## Consultation

```bash
# En local (mode démo) :
curl http://localhost:8000/debug/vars | jq .levelup

# En prod : auth admin requise (RequireAuth + RequireAdmin).
curl -H "Authorization: Bearer <token>" https://api.example.com/debug/vars | jq .levelup
```

Sortie typique :

```json
{
  "career_get_page": { "count": 142, "sum_ms": 8200, "avg_ms": 57, "max_ms": 312 },
  "home_get_page":   { "count": 89,  "sum_ms": 12500, "avg_ms": 140, "max_ms": 850 }
}
```

## Ajouter une métrique

1. Au début de la fonction à instrumenter, ajouter :
   ```go
   defer func(start time.Time) {
       observability.RecordDurationMS("my_op_name", time.Since(start).Milliseconds())
   }(time.Now())
   ```
2. Mettre à jour ce README avec la nouvelle entrée dans le tableau.
3. Ajouter un test si la métrique est critique : voir
   `expvar_metrics_test.go::TestRecordDurationMS_BasicAggregation`.

## Reset

`observability.Reset()` est exposé pour les tests qui veulent partir d'un état
propre. **Ne pas appeler en prod** — invalide les compteurs cumulés.
