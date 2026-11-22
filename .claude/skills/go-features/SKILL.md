---
name: go-features
description: Catalogue des algorithmes Go existants (internal/analysis et sous-paquets, domain/title). À invoquer AVANT d'implémenter un calcul, un indicateur, un agrégat ou un helper de libellé, pour vérifier qu'il n'existe pas déjà.
---

# Skill : go-features — Où chercher avant d'écrire

## Règle fondamentale

Avant de réimplémenter quoi que ce soit, chercher dans `apps/go-api/internal/analysis/` et ses sous-paquets. La quasi-totalité des algorithmes du produit existe déjà.

Trois sous-paquets ont un README catalogue : `temporal/`, `breakdown/`, `narrative/`. Les lire en premier quand la famille correspond.

Ce catalogue donne un chemin et quelques fonctions clés par ligne. Le code fait foi : ouvrir le fichier avant de s'y fier. Les chemins sont relatifs à `internal/analysis/` sauf mention contraire.

## Indicateurs et KPI (règle KDA, ADR 0006)

| Chemin | Fonctions clés |
|---|---|
| `indicators.go` | `AggregateKDA`, `KDR`, `WinRate`, `Accuracy`, `CombatEfficiency` |
| `kpi_stats.go` | `ComputeKPIStats`, `ComputeTeamAvgKPIs` |
| `stats_canonical.go` | agrégats de stats à partir de `canonical.PlayerMatchRow` |
| `expected_fda.go`, `expected_assists.go` | `ExpectedFDA`, `FDADiff` (écart au FDA attendu) |
| `pooled_per_minute.go` | `PooledPerMinute[T]` (taux par minute sur plusieurs matchs) |
| `combat_yield.go` | `ComputeCombatYield`, `FragEquivalents`, `ClassifyCombatProfile` |
| `math_safe.go` | `IsBadFloat`, `SanitizeFloat`, `SafeRatio` (jamais de `+Inf` ni de `NaN` en sortie) |

Réutiliser ces KPI plutôt que de recalculer : KDA n'est jamais un quotient.

## Score d'équipe, manches, prolongation (ADR 0032)

| Chemin | Fonctions clés |
|---|---|
| `team_score_display.go` | `ReadTeamScore`, `TeamScoreLabel`, `FormatTeamScoreLabel` |
| `camps_canonical.go` | `CampsDuMatch`, `EntreeDeScoreCanonique` |
| `overtime.go` | `ComputeOvertime`, `ResolveOvertime` |

## Séries temporelles et tendances

| Chemin | Fonctions clés |
|---|---|
| `temporal/` (README) | `BucketByGranularity`, `FilterByPeriod`, `RollingMean`, `LowessSmooth`, score d'engagement |
| `trends/` | indicateurs de tendance, matrice, séries de victoires et défaites, vue escouade |
| `sessions.go` | `ComputeSessions`, `BuildSessionGroups`, `IsSessionPotentiallyActive` |
| `match_history_avg.go` | `ComputeModeCategory` (moyennes glissantes de l'historique) |
| `rank_progression.go` | `ComputeRankProgressionByChain` |
| `retention.go` | `BorneRetention`, `BorneRetentionDepuis` |

## Ventilations et narration d'un match

| Chemin | Fonctions clés |
|---|---|
| `breakdown/` (README) | ventilations par carte, mode, playlist ; comparaisons |
| `narrative/` (README) | `IdentifyImpactRoles`, badge de dominance, cadence, profil d'intensité, premiers événements |
| `match_impact.go`, `match_impact_thief.go` | `ComputeMatchImpactFull` (badges d'impact) |
| `comeback.go`, `comeback_frag_contrast.go` | `ComputeDominanceFlag`, `ComputeScoreMarginDominance` |
| `tug_of_war.go`, `kd_timeline.go` | `ComputeTugOfWar`, `ComputeKDTimeline` |
| `killer_victim.go` | `ComputeKillerVictimPairs` (appariement tueur et victime) |
| `scoreboard_extremes.go` | `ComputeMVPLVP` |

## Primitives d'événements et de séries

| Chemin | Fonctions clés |
|---|---|
| `longest_run.go` | `LongestRun[T]` (plus longue série vérifiant un prédicat) |
| `max_killing_spree.go` | `ComputeMaxKillingSpree` |
| `first_blood.go` | `FirstBlood`, `FirstDeath`, `CountFirstKillsAndDeaths` |
| `kv_synthetic_events.go` | `SynthesizeKillEventsFromKVPairs`, `MergeAndSortCanonicalEvents` |
| `perfect_kills.go` | décompte des kills parfaits |

## Escouade

| Chemin | Fonctions clés |
|---|---|
| `squad_score.go` | `ComputeSquadPerformanceScore` |
| `squad_profiles.go` | `ComputeParticipationProfile`, `ComputeSquadRecords` |
| `squad_breakdown.go` | `ComputeSquadBreakdown`, `ComputeTopWeeks` |
| `squad_impact.go`, `squad_timeseries.go` | `ComputeImpactSummary`, `ComputeSquadTimeseries` |
| `squadimpact/`, `squadformes/`, `squademprise/` | rôles et historique d'impact, formes, emprise de l'escouade |

## Citations et médailles

| Chemin | Fonctions clés |
|---|---|
| `citations_engine.go` | `ComputeFullMatchCitations`, `ComputeMatchCitations`, `ParseTierMax`, `RegisterCustomDispatcher` |
| `citations_composite.go`, `citation_cumul.go` | citations composites et cumul |
| `citations.go` | `MergeCitationTotals`, `BuildCitationsByCategory` |
| `citation_snippets.go` | `ParseTierTargets`, `ComputeTierProgression`, `BuildCitationSnippets` |
| `medals.go`, `medal_exploit.go` | médailles ; `ComputeMedalExploitScore` |

## Notation et rang

| Chemin | Fonctions clés |
|---|---|
| `skill_v2/` et `internal/sync/skill/` | LUSR v2 (seule version vivante), lissage d'affichage |
| `skill_rating.go` | `ComputeSkillRatingsBatch` (batch hérité, utilisé par `service/stats_service.go`) |
| `home_canonical_skill.go` | `BuildSkillTierLabel`, `SkillBadgeURL`, `CSRTierOrdinal` |
| `internal/sync/performance.go` | score de performance relatif (`analysis/performance_score.go` ne porte plus que des constantes) |

Le CSR est une valeur de l'API : il n'est pas calculé ici.

## Libellés, filtres et modes

| Chemin | Fonctions clés |
|---|---|
| `mode_label.go` | `NormalizeModeLabel`, `ResolveModeUI`, `ExtractKnownMode` |
| `modelabel/` | catégorie de mode, extraction et nettoyage de libellés |
| `internal/games/halo_infinite/mode_category.go` | `InferModeCategoryFromPairName` (propre à Halo Infinite) |
| `playlist_label.go` | `NormalizePlaylistLabel`, `DisplayPlaylistLabel` |
| `match_filter.go` | `OutcomeCodeForLabel`, `IsValidOutcomeLabel` |
| `map_groups.go`, `map_variant.go` | `GroupMapsByBase`, `BaseMapName` (regroupement des variantes d'une carte) |

## Armes

| Chemin | Fonctions clés |
|---|---|
| `weapon_range.go`, `weapon_distance_records.go`, `weapon_opening_delta.go` | portée, records de distance, écart d'ouverture |
| `weapontier/` | paliers d'armes et recoupements |
| `internal/games/weapons/` | registre des armes du titre, libellés |
| `internal/games/halo_infinite/killsource_*.go` | source de dégâts d'une mort et son icône |

## Analyses issues du film et du rejeu

| Chemin | Contenu |
|---|---|
| `tactical/` | placement, géométrie, réapparitions |
| `coordination/` | blocs, isolation, appui |
| `leadshare/` | part de menée |
| `sessionusage/` | usage d'équipement par session |
| `killsummary/` | résumés par match du journal des morts (ADR 0039) |
| `digest/`, `prestigetuning/`, `patterns/`, `relations/` | synthèses, réglage Prestige, schémas de jeu, relations entre joueurs |

Le décodage du film est dans la brique `halo-infinite-film-decoder` (façade `decfilm`, ADR 0034), pas dans `analysis/`. Les temps forts issus du film : `internal/games/halo_infinite/film_highlight_events.go`.

## Accueil, monde, carrière

| Chemin | Contenu |
|---|---|
| `home_*.go`, `home_canonical_*.go` | agrégats de la page d'accueil (`home.go` : helpers transverses) |
| `world_stats.go` | `NormalizeSeasonID`, `AccumulateWorldStats` |
| `xp_estimate.go` | `EstimateCareerXP` |

## Multi-titres (`internal/domain/title/`)

| Chemin | Fonctions clés |
|---|---|
| `registry.go` | `Registry` (`NewRegistry`), `TitleDescriptor`, `PathResolver` (`NewPathResolver`, `NewPathResolverRoots`) |
| `matcher.go` | `MatchByXboxTitleID`, `MatchBySteamAppID`, `MatchPresence` |

## Code sans appelant de production

Ces fonctions existent mais aucun code de production ne les appelle : ne pas les présenter comme réutilisables, et se demander s'il faut les supprimer avant de bâtir dessus.

- `spawn_detection.go` (estimation du début de match dans un film).
- `ComputeAntagonistCounts` (`killer_victim.go`).
- `ComputeModeCategoryAverages` (`match_history_avg.go`).

## Pattern de vérification

```bash
# Chercher si une fonction existe déjà
grep -r "func.*Score\|func.*Rating\|func.*Normalize" apps/go-api/internal/analysis/

# Lister les exports d'un fichier
grep "^func [A-Z]" apps/go-api/internal/analysis/squad_score.go

# Vérifier qu'une fonction a un appelant de production
grep -rn "NomDeLaFonction" apps/go-api --include=*.go | grep -v _test.go
```

## Avant toute implémentation

Si la fonctionnalité semble absente de `analysis/` :

1. Chercher dans les autres paquets de `apps/go-api/internal/` : `service/`, `progression/`, `games/`, `platform/duckdb/`, `sync/`. La logique peut vivre côté lecture ou côté synchronisation.
2. Lire le journal des décisions et les plans actifs (`.ai/PLAN_*.md`, les plans clos sont sous `.ai/archive/`) : le sujet peut être en cours ou déjà tranché.
3. Seulement alors implémenter : dans `internal/analysis/` pour un algorithme pur et commun à tous les titres, dans `internal/games/{slug}/` pour ce qui dépend d'un titre, dans `service/` pour l'orchestration.
