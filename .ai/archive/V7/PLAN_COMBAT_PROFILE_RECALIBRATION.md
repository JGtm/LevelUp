# Plan — Recalibration du profil de combat (3 axes × 5 bandes)

**Date** : 2026-06-20
**Branche** : `feat/combat-profile-recalibration` (depuis main)
**Statut** : Complété (prêt à commit, non déployé). Repère barres = frontière élite mondiale (OC 0.90 / DR 1.65).

## Contexte

Les 4 joueurs suivis affichaient tous le même profil (« Offensif précis / Défensif solide / Engagement modéré »). Diagnostic (investigation data, cf. [[reference_combat_profile_grille_trop_grossiere]]) :
1. Seuils de classification figés calés sur l'élite mondiale → tout le monde « solide » (DR ≥ 1.59 = p90 mondial).
2. Axe « Engagement » basé sur `residual_brut`, une métrique **auto-référencée** (écart à sa propre norme via `coef_team_share`) → ~0 pour tout joueur constant (JGtm). Ce n'est PAS l'engagement absolu.

## Données de référence (world leaders, top-100 mondial, table `world_player_season_stats`, n=1349)

- **DR** : médiane 1.49, p75 1.55, p90 1.60, p99 1.76, max 2.10.
- **OC** : médiane 0.82, p75 0.85, p90 0.87, p99 0.92, max 0.98 (borné — saturation).
- **Engagement absolu** = `pace_joueur / pace_lobby` (colonnes déjà persistées `engagement_pace_player` / `engagement_pace_lobby`). Mesuré : Madina 1.34 > JGtm 0.99 > Chocoboflor 0.87 > XxDaemon 0.71 (conforme au terrain).

## Décisions validées (vocabulaire + bornes)

| Bande | Offensif (OC) | Défensif (DR) | Activité (pace_ratio) |
|---|---|---|---|
| 1 | Dispersé `< 0.78` | Fragile `< 1.20` | Passif `< 0.80` |
| 2 | Irrégulier `0.78–0.81` | Exposé `1.20–1.35` | Discret `0.80–0.92` |
| 3 | Équilibré `0.81–0.85` | Solide `1.35–1.50` | Mesuré `0.92–1.08` |
| 4 | Précis `0.85–0.90` | Résistant `1.50–1.65` | Actif `1.08–1.25` |
| 5 | Chirurgical `> 0.90` | Inébranlable `> 1.65` | Agressif `> 1.25` |

`pace_ratio` agrégé = `mean(engagement_pace_player / engagement_pace_lobby)` sur les matchs où `pace_lobby > 0`.

## Principe de découplage (IMPORTANT)

Les constantes `OffensiveConversionP80=0.83` / `DefensiveResistanceP80=1.59` servent à DEUX usages aujourd'hui :
- **Classification textuelle** (badges) → remplacée par les 15 bandes ci-dessus (nouvelles constantes).
- **Normalisation visuelle des barres** (`NormalizeForBar` Go + 3 répliques front) → c'est le **Lot 2**.

On crée des constantes **séparées** pour les bandes ; le Lot 1 ne touche donc PAS l'échelle des barres.

## Lot 1 — Badges 5 bandes + axe Activité sur pace_ratio (+ glossaire)

**Backend**
- `internal/analysis/combat_yield.go` : nouvelles bornes (slices `ocBands`/`drBands`/`paceBands`), `classifyOffensive`/`classifyDefensive` → 5 bandes, `classifyActivity(avgPaceRatio)` → 5 bandes ; `ClassifyCombatProfile` param `avgResidualBrut` → `avgPaceRatio`. Commentaires « Phase 4 » périmés corrigés.
- `internal/domain/combat_profile.go` : 15 constantes `CombatStyle` ; champ `AvgResidualBrut` → `AvgPaceRatio` (JSON `avg_pace_ratio`).
- `internal/games/canonical/match.go` : + `EngagementPacePlayer *float64`, `EngagementPaceLobby *float64`.
- SELECT enrichment (`platform/duckdb/shared_query_helpers.go` + projection) : lire `engagement_pace_player`, `engagement_pace_lobby`.
- Agrégats `kpi_stats.go`, `service/synthesis_service_builders.go`, `service/session_compare_service.go` : calculer `avgPaceRatio` et le passer à `ClassifyCombatProfile`. `residual_brut` reste calculé/persisté (coaching/patterns) — non touché.

**Frontend**
- Centraliser les 15 labels dans `features/_shared/combatProfileLabels.ts` (consommé par `SynthesisPage.tsx` + `SquadCombatProfileRow.tsx`).
- `lib/api/types.ts` : étendre les unions de styles à 5 valeurs ; `avg_residual_brut` → `avg_pace_ratio`.
- `session-compare/SessionCompareEngagement.tsx` : basculer sur `avg_pace_ratio`.
- Strings i18n FR + EN.

**Glossaire** — `features/help/i18n.ts` : section « Profil de combat » (3 axes + formules + 5 bandes), FR + EN.

## Lot 2 — Recalage des barres visuelles (repère à valider)

Le repère « 100 % » des barres OC/DR (`NormalizeForBar` + `combat-yield-bar.tsx`, `SessionOcdrBars.tsx`, `TimeseriesCombatYield.tsx`) est aujourd'hui 0.83/1.59. À recaler une fois le repère choisi (médiane pro 0.82/1.49 ? frontière élite 0.90/1.65 ?). **À trancher avant implémentation.**

## Consommateurs (carte d'impact)

- Badges (à étendre) : SynthesisPage, SquadCombatProfileRow.
- Barres/charts OC/DR (Lot 2) : combat-yield-bar, SessionOcdrBars, TimeseriesCombatYield, + KpiGrid/SessionCompareOCDR/Explorer (consomment, pas de seuil).
- `residual_brut` (NE PAS casser) : coaching (`progression/coach`), patterns (`analysis/patterns/behavioral_engagement.go`).

## Tests
- `combat_yield_test.go` (5 bandes + classifyActivity ratio), `kpi_stats_test.go`, synthesis/session_compare.
- Front : typecheck + mapping labels.

## Done definition
- 4 joueurs différenciés (JGtm Précis/Exposé/Mesuré, Madina Équilibré/Solide/Agressif, Choco Équilibré/Solide/Discret, Daemon Dispersé/Fragile/Passif).
- `go test ./...` + `go vet` verts ; `npm typecheck/lint` verts.
- Glossaire + thought_log à jour.
