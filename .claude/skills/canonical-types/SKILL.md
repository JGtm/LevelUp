# Skill : canonical-types — Types canoniques inter-titres (internal/games/canonical/)

## Rôle

Les types canoniques sont la "lingua franca" entre les adapters (`TitleDataAdapter`) et les services produit. Ils sont stables quelle que soit la source de données du titre.

Localisation : `apps/go-api/internal/games/canonical/`

## Types principaux

### Match

```go
// MatchSummary — résumé d'un match (liste, historique)
type MatchSummary struct {
    MatchID         string
    StartedAtUTC    time.Time
    DurationSeconds *int
    MatchType       MatchType        // Ranked, Social, Custom, Firefight
    Playlist        *AssetReference
    Map             *AssetReference
    GameVariant     *AssetReference
    IsRanked        *bool
    IsPvE           *bool
    Outcome         Outcome          // Win, Loss, Tie, DNF
}

// MatchDetail — détail complet (page match)
type MatchDetail struct {
    MatchID      string
    Participants []MatchParticipant
    Teams        []TeamSnapshot
    Skill        *MatchSkillSnapshot
    Limitations  []CapabilityGap    // features dégradées pour ce match
    // + champs de MatchSummary
}

// MatchParticipant — stats d'un joueur dans un match
type MatchParticipant struct {
    Identity      PlayerIdentity
    TeamID        *int
    Outcome       Outcome
    Kills         *int
    Deaths        *int
    Assists       *int
    HeadshotKills *int
    Accuracy      *float64
    DamageDealt   *int
    DamageTaken   *int
    Score         *int
    // ...
}
```

### Joueur

```go
type PlayerIdentity struct {
    XUID               string
    Gamertag           string
    GamertagNormalized string
    EmblemURL          string
    AvatarURL          string
    IsBot              bool
}

type PlayerStats struct {
    Identity      PlayerIdentity
    MatchesPlayed int
    Wins, Losses, Ties int
    WinRate       *float64
    Kills, Deaths, Assists int
    KDR, KDA      *float64
    Accuracy      *float64
}
```

### Carrière

```go
type CareerSnapshot struct {
    Player        PlayerIdentity
    CurrentRank   *AssetReference
    CurrentXP     *int
    XPForNextRank *int
    NextRank      *AssetReference
    History       []CareerHistoryEntry
    HighestCSR    *int
    HighestLUSR   *float64
    IsMaxRank     bool
}

type EncounterRow struct {
    Identity   PlayerIdentity
    MatchCount int
    AsTeammate int
    AsEnemy    int
    AvgKDA     *float64
}
```

### Assets et références

```go
// AssetReference — référence localisée à un asset (mode, map, playlist…)
type AssetReference struct {
    Kind         string
    ID           string
    DefaultLabel string
    Labels       map[string]string  // locale → label (ex: "fr" → "Assassin")
    IconURL      string
}
```

### Scopes et requêtes

```go
type StatsScope struct {
    From, To      time.Time
    PlaylistIDs   []string
    IncludePvE    bool
    IncludeRanked *bool
}

type TimeseriesQuery struct {
    Metric  FieldKey
    Bucket  Bucket    // Day, Week, Month
    From, To time.Time
    GroupBy  []GroupBy
}

type CareerOptions struct {
    IncludeHistory bool
    HistoryLimit   int
}
```

## Enums

```go
type Outcome   string  // OutcomeWin, OutcomeLoss, OutcomeTie, OutcomeDNF
type MatchType string  // MatchTypeRanked, MatchTypeSocial, MatchTypeCustom, MatchTypeFirefight
type RatingType string // RatingTypeCSR, RatingTypeLUSR
type Bucket    string  // BucketDay, BucketWeek, BucketMonth
```

## FieldKey — constantes canoniques

Définies dans `canonical/fields.go`. Groupes principaux :

| Groupe | Exemples de FieldKey |
|---|---|
| combat | `kills`, `deaths`, `assists`, `headshot_kills`, `accuracy`, `kdr`, `kda`, `damage_dealt` |
| match | `match_id`, `started_at_utc`, `duration_seconds`, `outcome`, `personal_score`, `is_ranked` |
| career | `current_rank_id`, `current_xp`, `xp_for_next_rank` |

Ces clés correspondent aux sections dans `config/titles/{slug}/mappings/fields.toml`.

## Règle d'utilisation

Un service produit ne doit **jamais** avoir de colonnes DuckDB title-specific dans son code.
Tout passe par les méthodes `Load*` du `TitleDataAdapter` qui retournent ces types canoniques.

```go
// Correct
summaries, err := data.LoadMatchSummaries(ctx, ids)
// summaries[0].Kills est *int — nullable, dégrader si nil

// Interdit dans un service
conn.Query("SELECT kills FROM match_participants WHERE ...")
```
