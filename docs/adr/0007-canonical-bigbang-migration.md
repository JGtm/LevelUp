# ADR 0007 — Canonical PlayerMatchRow big-bang migration (services produit)

**Status** — Proposed (2026-04-29). Triggered by code review axe 2 (BLOQUANT 4 « canonical non propagé »).

**Deciders** — Guillaume (GS).

## Context

ADR 0002 a posé `canonical.PlayerMatchRow` comme contrat d'échange entre `service/`, `analysis/` et `domain/`. Au 2026-04-29, **3 services seulement** consomment ce type :

- `squad_service_v2.go`
- `match_history_service.go`
- `explorer_service.go`

Les **13 autres services produit** lisent leurs propres `domain.*MatchRow` calqués 1-1 sur les colonnes DuckDB Halo Infinite :

| Service | Row-type lu | Colonnes |
|---|---|---|
| `home_service.go` | `domain.HomeMatchRow` | 40 colonnes Halo |
| `synthesis_service.go` | `domain.SynthesisMatchRow` | similaire |
| `career_service.go` | `domain.CareerMatchRow` + canonical via `LoadEncounters` | partiel |
| `match_view_service.go` | `domain.MatchView*` | hook `WithDataAdapter` mais flux legacy |
| `squad_service.go` (v1) | `domain.SquadMatchRow` | Halo |
| `stats_service.go` | `domain.StatsMatchRow` | Halo |
| `timeseries_service.go` | `domain.TimeseriesMatchRow` | Halo |
| `match_history_service.go` | `canonical.PlayerMatchRow` | OK |
| `explorer_service.go` | `canonical.PlayerMatchRow` | OK |
| `citations_service.go` | `domain.CitationContext` | Halo |
| `engagement_score_service.go` | DuckDB direct | Halo |
| `compare_service.go` | repo DuckDB | titleSlug paramétré |
| `media_service.go` | `domain.Media*` + analysis | catégories Halo hardcodées |
| `leaderboard_service.go` | repo DuckDB | Halo CSR seul |
| `season_pass_service.go` | repo DuckDB | Halo only |
| `prestige/evaluator.go` | repo DuckDB | Halo only |

Conséquence : pour ajouter un titre B (Halo MCC, Halo 5, etc.), il faut **soit forker N row-types soit migrer N services**. Le multi-titres ne fonctionne aujourd'hui que pour Squad V2 + Match History + Explorer.

## Decision

**Migration big-bang** (option A1=B des arbitrages de revue) sur branche dédiée :

- **Branche** : `refactor/canonical-migration-bigbang` issue de `feat/multi-title-static-fs-rescope`.
- **Pilote** : `home_service.go` migré en premier (volume faible, exposition large) → sert de référence aux 12 autres.
- **Sub-PRs** : 1 par service, mergées dans la feature branch (revue progressive). Pas de merge intermédiaire vers `main`.
- **Merge final** : 1 PR groupé vers `feat/multi-title-static-fs-rescope` une fois les 13 services migrés.

**Ordre suggéré** (du plus simple au plus complexe) :

1. `home_service.go` (pilote)
2. `synthesis_service.go`
3. `career_service.go`
4. `stats_service.go`
5. `match_view_service.go`
6. `session_compare_service.go`
7. `timeseries_service.go`
8. `compare_service.go`
9. `engagement_score_service.go`
10. `citations_service.go`
11. `media_service.go`
12. `leaderboard_service.go`
13. `season_pass_service.go`

**Critère done par service** :
- Service consomme `[]canonical.PlayerMatchRow` au lieu de `[]domain.*MatchRow`.
- Repo correspondant expose la donnée canonique via `TitleDataAdapter`.
- DTO HTTP externe **inchangé** (le canonical est interne).
- Tests verts : handler + service (mock `port.Repository`) + analysis (pur).
- Mesure perf avant/après documentée dans le commit (latence + allocation).

**Suppression des types legacy** post-migration : `domain.HomeMatchRow`, `domain.StatsMatchRow`, `domain.SquadMatchRow`, `domain.SynthesisMatchRow`, `domain.CitationContext`, etc.

## Consequences

### Positive

- Alignement total : `canonical.PlayerMatchRow` devient l'unique entrée des services.
- Ajout d'un 2e titre (Halo MCC, Halo 5) ne nécessite plus de modifier les services produit.
- Suppression des 13 row-types parallèles → réduction code + dette.
- Découplage des couches `service/` ↔ `platform/duckdb/` via `TitleDataAdapter`.

### Negative

- **Risque maximal du plan** : 13 services touchés simultanément, ~2-3 semaines effort concentré.
- Régressions possibles non détectées si tests insuffisants (raison pour laquelle P3 « tests fondations » précède P4).
- Perf à mesurer : conversion `domain.*MatchRow` → `canonical.PlayerMatchRow` peut introduire allocations supplémentaires.
- Discovery friction temporaire : un dev qui touche un service en cours de migration doit savoir lequel utiliser.

## Mitigations

1. **Pilote avant le big bang** (P4.1) : `home_service.go` migré seul, validation perf + tests, plan documenté pour les 12 autres.
2. **Tests fondations P3 préalables** : couverture honnête sur handlers/services, tests régression engagement B1-B4.
3. **Sub-PRs incrémentaux** sur la feature branch : revue progressive, merge final unique.
4. **Pas de merge intermédiaire vers main** : la branche peut rester ouverte ~3 semaines sans gêner d'autres travaux.
5. **Mesure perf systématique** : `go test -bench` ou `pprof` avant/après sur chaque service, delta documenté dans le commit.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) Migration progressive 1 service / sprint** | Effort étalé sur 13 sprints (~3 mois). Co-existence prolongée des deux contrats. Risque que la migration s'arrête à mi-chemin (pattern « scaffolding then forget »). |
| **C) Status quo : « canonical pour les nouveaux services seulement »** | Bloque le multi-titres au-delà de Squad/MatchHistory/Explorer. Acte un plafond ~35-40 % de migration. Dette grandit. |
| **D) Génériques per-titre (Go generics)** | ADR 0002 a déjà rejeté cette piste : insuffisant pour le cas. |

## References

- ADR 0002 — `canonical-player-match-row.md` (contrat initial).
- Code review : `axe-2-multi-titres.md` (matrice services / canonical), `axe-1-agnosticisme.md` (BLOQUANT canonical non propagé).
- Plan d'action : `PLAN_ACTION.md` P4 (big bang) + P3 (tests fondations préalables).
- Skill : `.claude/skills/canonical-types/SKILL.md`.
