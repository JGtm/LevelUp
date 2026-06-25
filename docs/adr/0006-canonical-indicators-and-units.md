# ADR 0006 — Canonical indicators and units (KDA, KDR, WinRate, Accuracy, PerfTier)

**Status** — Proposed (2026-04-29). Triggered by code review axe 1 (BLOQUANT 1, 2) + axe 6 (BLOQUANT seuils PerfTier).

**Deciders** — Guillaume (GS).

## Context

La revue de code 2026-04-29 a identifié plusieurs anti-patterns sur le calcul et l'unité des indicateurs métier :

- **WinRate** : 7+ implémentations Go avec unités divergentes (`0..1` dans Home/Synthesis/SquadV2, `0..100` dans Stats/Teammates/Compare/Timeseries). 14+ sites front font `*100` parfois en double, parfois jamais.
- **Accuracy** : même divergence (`0..1` dans MatchView/Compare, `0..100` dans Home/Squad/MatchCard).
- **KDA** : formule canonique `(K+A)/max(1,D)` documentée dans `apps/web/src/features/help/i18n.ts:326` mais répétée 3× inline en Go (`analysis/performance_score.go`, `sync/performance.go`). Aucun helper centralisé.
- **K/D ratio (KDR)** : 2 recomputes côté front (`TimeseriesKdaBars.tsx:78`, `SynthesisPage.tsx:139-141`), dont 1 mathématiquement faux (`sum(K)/sum(D) ≠ avg(K/D)`).
- **PerfTier (5 paliers)** : 6 implémentations Go avec seuils divergents (`80/60/40` 3-paliers vs `80/65/50/35` 5-paliers). Bug visuel latent : un même score `62` reçoit 3 couleurs différentes selon la surface.
- **Précision décimale** : `toFixed(0)`, `toFixed(1)`, `toFixed(2)`, `Math.round(*1000)/10` selon le module — aucune convention.

Décisions produit déjà actées pendant la revue :
- Aucun recompute KDA/KDR/K/D justifié côté front : la valeur API fait foi partout.
- Convention canonique d'unité : `0..1` côté API (ratio), formatage `*100` + arrondi décimal **uniquement à l'affichage** front.

## Decision

**Centraliser les indicateurs dans `internal/analysis/indicators.go`** avec les formules et unités canoniques suivantes :

```go
package analysis

// KDA = (kills + assists) / max(1, deaths)
func KDA(k, a, d int) float64

// KDR = K/D ratio (sans assists) — différent de KDA
func KDR(k, d int) float64

// WinRate = wins / total — TOUJOURS 0..1
func WinRate(wins, total int) float64

// Accuracy = hits / fired — TOUJOURS 0..1
func Accuracy(hits, fired int) float64

// PerfTier — 5 paliers, seuils canoniques [80, 65, 50, 35]
// score >= 80  → Tier1 (Excellent)
// score >= 65  → Tier2 (Bon)
// score >= 50  → Tier3 (Correct)
// score >= 35  → Tier4 (Faible)
// sinon         → Tier5 (Mauvais)
func PerfTier(score float64) Tier
```

**Conventions appliquées** :

| Élément | Convention | Notes |
|---|---|---|
| Unité côté API | toujours `0..1` (ratio) | Pas de `0..100` dans aucun DTO |
| Formatage côté front | helper `formatPercent(ratio, decimals = 1)` | `apps/web/src/lib/formatters/percent.ts` |
| Précision décimale | 1 par défaut · 2 pour ratios sub-unitaires (KDA, KDR) · 0 pour compteurs | Standardisée |
| PerfTier seuils | `[80, 65, 50, 35]` | Source de vérité = `perfScale` côté front, déjà testée |
| Outcome enum | `OutcomeTie=1, OutcomeWin=2, OutcomeLoss=3, OutcomeDNF=4` | Aligné Go ↔ TS |
| KDA front | jamais recomputé | Consommé tel quel via `m.kda`, `kpis.avg_kda` |
| KDR front | jamais recomputé | Backend doit exposer `kd_ratio` (extension DTO Timeseries en P2.5) |

## Consequences

### Positive

- Source unique de vérité pour les indicateurs.
- Suppression de 7+ implémentations divergentes en Go.
- Bug visuel PerfTier (couleur incorrecte sur Match View) résolu mécaniquement.
- Bug Synthesis sum/sum K/D résolu via exposition `total_kdr` ou consommation `kpis.global_ratio`.
- Tests table-driven complets sur le helper canonical.

### Negative

- **Breaking API** côté front : passage de `0..100` à `0..1` sur plusieurs endpoints (Stats, Teammates, Compare, Timeseries). Mitigation : helper `formatPercent` introduit en même temps, regen `openapi-typescript` casse le build aux endroits incohérents.
- Migration des 7+ sites Go dans une seule passe (P2.3 du plan).
- Extension des DTOs `TimeseriesMatchRow` (ajout `kd_ratio` + `kda`) et `SynthesisOverview` (ajout `total_kdr`) — couplage à coordonner.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **Garder les formules inline + ADR doc-only** | Ne résout pas la divergence des unités ni le bug visuel PerfTier. |
| **Convention `0..100` côté API** | Force le front à ne pas multiplier ; or 50% des sites front s'attendent déjà à du ratio. Migration symétrique sans gain. |
| **PerfTier 3 paliers `[80, 60, 40]`** | Divergent avec `perfScale` côté front (testé, source de vérité). Recours = aligner Go sur TS, pas l'inverse. |
| **Recompute K/D toléré côté front** | Risque divergence Go ↔ TS (déjà cassé dans Synthesis avec sum/sum). API doit faire foi. |

## References

- Code review : `.ai/V7/review/2026-04-29/axe-1-agnosticisme.md` (BLOQUANT 1, 2 amendés), `axe-6-dry.md` (BLOQUANT PerfTier).
- Plan d'action : `PLAN_ACTION.md` P1.2, P2.1, P2.3, P2.5, P2.6.
- Source de vérité PerfTier : `apps/web/src/lib/accessibility/scales/instances.ts:20` + tests `apps/web/src/lib/accessibility/scales/__tests__/instances.test.ts:14-22`.
- Formule KDA documentée : `apps/web/src/features/help/i18n.ts:326`.
