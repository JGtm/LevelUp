# ADR 0010 — Timeseries binning : serveur (Go) plutôt que client (front)

**Status** — Accepted (2026-04-29, finalisé en P7.1 sub-PR D du `PLAN_ACTION.md`).

**Deciders** — Guillaume (GS).

## Context

L'audit axe 1 (BLOQUANT 5) a soulevé que les DTOs Timeseries pré-shapent du ECharts :

- `IntensityHeatmapPoint{X, Y, Count, AvgKD}` : axes X/Y nommés comme du chart, pas comme du métier (`hour_of_day`, `day_of_week`)
- `DistributionBucket{BinStart, BinEnd, Count}` : histogrammes pré-binnés côté Go avant transit JSON

Conséquence du pré-binning : **changer le nombre de bins requiert un appel API** au lieu d'un re-render front sur la donnée brute.

Question : exposer la donnée brute (matchs avec leurs valeurs) et binner côté front pour gagner en flexibilité, ou garder le pré-binning Go pour la perf ?

**Note** : cet ADR sera produit en P7.1 sub-PR D, après le renommage des DTOs (passage `X/Y` → noms métier sémantiques).

## Decision

**Conserver le pré-binning côté Go** pour les histogrammes Timeseries.

Raisons :

1. **Perf transit JSON** : un histogramme typique a ~30-50 bins. La donnée brute serait des milliers de matchs (1 ligne par match) — payload JSON 100× plus lourd.
2. **Charge front** : binner côté front sur 5000+ matchs à chaque re-render serait inutilement coûteux en CPU client.
3. **Cas d'usage actuel** : aucun besoin de re-binning dynamique côté UI. Les bins sont fixes par feature (ex: histogramme KDA = bins de largeur 0.5 entre 0 et 5+).

Les DTOs portent désormais des noms métier (`BucketLower`, `BucketUpper` au lieu de `BinStart`/`BinEnd` ECharts), ce qui décorelle le contrat API du chart consommateur (cf. ADR 0006 + amendement axe 1). Le bucket reste générique (utilisable pour KDA, kills, accuracy, score-per-min, rolling-WR), avec la métrique implicite portée par le tab parent (`kda_buckets`, `kills_buckets`…) plutôt que par le DTO lui-même.

## Couplage assumé documenté

- Le nombre de bins, la largeur, et les bornes sont **figés côté Go** par la fonction de binning du service Timeseries.
- Pour ajouter un nouveau type d'histogramme, le pré-binning Go doit être étendu.
- Si un jour on veut un re-binning dynamique côté UI, alternative : exposer un endpoint dédié qui retourne la donnée brute matchs (reste à designer, pas dans ce scope).

## Consequences

### Positive

- Payload JSON léger (~50 lignes de bins vs ~5000 matchs bruts).
- Re-render front instantané (juste une map sur les bins déjà calculés).
- Cohérent avec le découplage post-renommage : le DTO porte des noms métier, le binning est un détail d'implémentation backend.

### Negative

- Pas de re-binning dynamique côté UI (slider « ajuster la largeur des bins »). Si le besoin émerge, refactor obligatoire.
- Test du calcul de binning uniquement côté Go (les snapshots front ne testent que le rendu sur les bins reçus).

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) Exposer la donnée brute matchs et binner côté front** | Payload JSON 100× plus lourd. Charge CPU client élevée. Aucun cas d'usage actuel justifiant la flexibilité. |
| **B) Binning côté Go + side channel pour la donnée brute (deux endpoints)** | Surdimensionné. Double maintenance pour zéro besoin actuel. |
| **C) GraphQL avec sélection des bins** | Pas de stack GraphQL. Surdimensionné pour le besoin. |

## References

- Code review : `axe-1-agnosticisme.md` (BLOQUANT 5 — DTOs Timeseries / Synthesis pré-shape pour ECharts).
- ADR 0006 — `canonical-indicators-and-units.md` (renommage DTOs, sémantique métier).
- Plan d'action : `PLAN_ACTION.md` P7.1 sub-PR D.
- Code : `apps/go-api/internal/domain/timeseries.go`, `apps/go-api/internal/service/timeseries_service.go`.
