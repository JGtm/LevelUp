---
name: canonical-types
description: Types canoniques inter-titres (internal/games/canonical/) et contrat TitleDataAdapter.Load*. À invoquer avant de lire, ajouter ou modifier un type, un FieldKey ou un champ servi par un adaptateur de titre, ou d'écrire un service qui consomme des données de match, de carrière ou d'événements.
---

# Skill : canonical-types — Types canoniques inter-titres

## Rôle

Les types canoniques sont la langue commune entre les adaptateurs de titre (`TitleDataAdapter`) et les services produit. Ils restent stables quelle que soit la source de données du titre.

Localisation : `apps/go-api/internal/games/canonical/`. Le code fait foi : lire le fichier avant de s'appuyer sur un champ.

## Carte des types, par famille

| Famille | Fichier | Types |
|---|---|---|
| Match | `match.go` | `MatchSummary`, `MatchParticipant`, `PlayerMatchRow`, `PlayerMatchEnrichment`, `TeamSnapshot`, `SkillSnapshot`, `MatchSkillSnapshot`, `ImpactBadge`, `CapabilityGap`, `HighlightEvent` |
| Récompenses natives | `match.go` | `CommendationDefinition`, `CommendationTotal` |
| Événements de match | `events.go` | `MatchEvent`, `MatchEventTimeline`, `MatchEventOptions`, `MatchEventType`, `KillKind`, `Vec3` |
| Joueur | `identity.go` | `PlayerIdentity`, `PlayerStats` |
| Carrière | `career.go` | `CareerSnapshot`, `CareerHistoryEntry`, `LUSRCheckpoint`, `EncounterRow` |
| Explorer (deux joueurs) | `explorer.go` | `CommonMatchRow`, `CrossKillTally`, `PlayerIntersection`, `PlayerMatchSetStats`, `RecentMatchRow` |
| Assets | `assets.go`, `match.go` | `AssetReference` (match.go), `AssetMeta`, `KillSourceIcon` |
| Catalogue | `catalog.go` | `Experience`, `CanonicalPlaylist`, `CanonicalPair`, `CanonicalMap`, `CanonicalGameVariant` |
| Requêtes | `scopes.go` | `StatsScope`, `TimeseriesQuery`, `CareerOptions` (`DefaultCareerHistoryLimit = 50`) |
| Séries temporelles | `timeseries.go` | `MetricSeries`, `TimeBucket` |
| Enums | `enums.go` | `Outcome`, `MatchType`, `RatingType`, `Bucket`, `GroupBy`, `DominanceFlag`, `HighlightEventType` |
| Clés de champ | `fields.go` | `FieldKey` et ses constantes |
| Catégories | `medal_category.go`, `commendation_category.go` | catégories de médailles et de citations |

Il n'existe pas de type `MatchDetail` : le détail d'un match se compose de plusieurs chargements (voir plus bas).

## Types les plus utilisés

### `MatchSummary` (liste, historique)

Champs de base : `MatchID`, `StartedAtUTC`, `DurationSeconds *int`, `MatchType`, `Playlist`, `Map`, `GameVariant` (tous `*AssetReference`), `IsRanked *bool`, `IsPvE *bool`, `Outcome`.

Champs plus récents, à connaître :
- `PairMode *AssetReference` : le mode du match. Les pages le lisent tel quel, sans repli sur `GameVariant` (un test d'architecture le garde).
- `Teams []TeamSnapshot` : scores d'équipes pour les listes.
- `T0Ms *int64` : décompte d'avant match. `GameplayDurationSeconds()` rend la vraie durée de jeu : l'utiliser plutôt que de soustraire à la main.
- `RoundsTotal *int` : manches jouées (ADR 0032).

### `MatchParticipant` (un joueur dans un match)

`Identity`, `TeamID`, `Outcome`, `Score`, `Kills`, `Deaths`, `Assists`, `HeadshotKills`, `Accuracy`, `DamageDealt`, `DamageTaken`, tous pointeurs. S'y ajoutent les champs du tableau des scores (temps joué, kills de mêlée ou de grenade, arme principale, `IsBot`), les mécaniques propres à Halo 5 et `KillsExpected` / `DeathsExpected`. Un champ à `nil` veut dire « non fourni par ce titre » : dégrader, ne pas afficher 0.

### `PlayerMatchRow` (contrat des services d'agrégation)

Compose `Summary` (`MatchSummary`), `Self` (`MatchParticipant` du joueur principal) et `Enrichment` (`PlayerMatchEnrichment` : session, note de performance, dominance, amis, MMR, `SkillSnapshot`). C'est l'entrée de la plupart des calculs de `internal/analysis/`.

### `PlayerIdentity`

`XUID`, `Gamertag`, `GamertagNormalized`, `ServiceTag`, `EmblemURL`, `AvatarURL`, `IsBot`. Le xuid est une chaîne : un identifiant Halo peut dépasser `int64`.

### `CareerSnapshot`

Rang courant et suivant (`CurrentRank`, `NextRank`, `*AssetReference`), `CurrentXP`, `XPForNextRank`, `XPTotal`, `History []CareerHistoryEntry`, `HighestCSR`, `HighestLUSR`, bornes propres au titre (`XPMax`, `RankMax`, `MaxRank`), état de placement (`MeasurementMatchesRemaining`, `PlacementTotal`). Les bornes à `nil` laissent le service retomber sur ses valeurs par défaut.

### `AssetReference`

`Kind`, `ID`, `VersionID`, `DefaultLabel`, `Labels` (code de langue vers libellé), `IconURL`. Pour choisir un libellé par langue : `Text()` (rend un `i18n.Text`), puis `Pick`. Ne jamais lire `Labels["fr"]` en dur.

### Requêtes

- `StatsScope` : `From`, `To` (UTC), `PlaylistIDs`, `IncludePvE`, `IncludeRanked *bool`. Liste vide = pas de filtre.
- `TimeseriesQuery` : `Metric FieldKey`, `Bucket`, `From`, `To`, `GroupBy []GroupBy`.
- `CareerOptions` : `IncludeHistory`, `HistoryLimit` (0 = 50, pas « tout »).

## Enums

- `Outcome` : `OutcomeWin`, `OutcomeLoss`, `OutcomeTie`, `OutcomeDNF`.
- `MatchType` : `MatchTypeRanked`, `MatchTypeSocial`, `MatchTypeCustom`, `MatchTypeFirefight`, `MatchTypeUnknownMT`.
- `RatingType` : `RatingTypeCSR`, `RatingTypeLUSR`.
- `Bucket` : `BucketDay`, `BucketWeek`, `BucketMonth`.
- `GroupBy` : `GroupByPlaylist`, `GroupByMode`, `GroupByMap`.
- `DominanceFlag` : de `DominanceNone` à `DominanceAbnegation` (8 valeurs).
- `HighlightEventType` : kill, death, assist, medal, finisher, clutch, first_kill, first_death.

Chaque enum a son `IsKnown...` et son `All...` : les utiliser pour valider une entrée.

## FieldKey

Défini dans `canonical/fields.go`. Six groupes :

| Groupe | Exemples |
|---|---|
| combat | `kills`, `deaths`, `assists`, `headshot_kills`, `accuracy`, `kdr`, `kda`, `damage_dealt` |
| match | `match_id`, `started_at_utc`, `duration_seconds`, `outcome`, `personal_score`, `is_ranked` |
| career | `current_rank_id`, `current_xp`, `xp_for_next_rank`, `win_rate` |
| skill | `team_mmr`, `enemy_mmr`, `kills_expected`, `csr_value`, `lusr_value` |
| pve | `waves_completed`, `bosses_killed`, `grunt_kills` |
| derived | `match_count`, `kills_per_min`, `performance_score`, `engagement_score`, `personal_score_per_min` |

Chaque clé correspond à une section `[fields.<clé>]` de `config/titles/{slug}/mappings/fields.toml`. Une section absente veut dire « surface non supportée par ce titre ».

## Politique d'évolution : additive uniquement

- Ajouter un champ ou un type est permis. Supprimer ou renommer un champ est un changement cassant : ADR, dépréciation préalable, message de commit explicite (ADR 0005).
- Une valeur de `FieldKey` ne change jamais. Un test refuse tout renommage ou tout ajout non déclaré.
- Ajouter un `FieldKey` impose trois gestes : l'ajouter dans `AllFieldKeys()`, mettre à jour le golden `canonical/testdata/fields.golden.txt`, et incrémenter `schema_version` dans les TOML de mappings concernés.
- Un champ optionnel est un pointeur : c'est ce qui distingue « inconnu » de « zéro ».

## Charger les données : méthodes `Load*` du `TitleDataAdapter`

Interface dans `internal/games/adapter.go`. Chaque méthode rend un type canonique, ou `ErrCapabilityNotSupported` si le titre n'a pas la capacité (dégrader proprement, jamais de panic).

| Besoin | Méthode |
|---|---|
| Stats d'un joueur | `LoadPlayerStats(ctx, xuid, StatsScope)` |
| Carrière | `LoadCareerSnapshot(ctx, xuid, CareerOptions)` |
| Série temporelle | `LoadTimeseries(ctx, xuid, TimeseriesQuery)` |
| Tableau des scores d'un match | `LoadMatchScoreboard(ctx, matchID)` |
| Temps forts d'un match | `LoadHighlightEvents(ctx, matchID)` |
| Chronologie complète | `LoadMatchEvents(ctx, matchID, MatchEventOptions)` |
| Rencontres, amis | `LoadEncounters`, `LoadFriendsXUIDs` |
| Historique de rating | `LoadLUSRHistory` |
| Explorer | `LoadTargetRecentMatches`, `LoadParticipantStats`, `LoadPlayerIntersection` |

## Règle d'utilisation

Un service ne contient ni SQL ni colonne DuckDB propre à un titre (garde d'architecture sur les services). Les lectures canoniques passent par `TitleDataAdapter.Load*` ; les autres lectures passent par un port de `internal/port/`.

```go
// Correct : kills d'un match, nullable
players, err := data.LoadMatchScoreboard(ctx, matchID)
// players[i].Kills est *int : dégrader si nil

// Interdit dans un service
conn.Query("SELECT kills FROM match_participants WHERE ...")
```

Le package `canonical` importe `internal/i18n` (pour `i18n.Text`) et rien d'autre hors bibliothèque standard.
