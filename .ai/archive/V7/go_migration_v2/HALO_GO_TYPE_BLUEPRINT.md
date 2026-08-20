# HALO_GO_TYPE_BLUEPRINT.md — Blueprint documentaire des types Go canoniques

> Document de préparation à l'implémentation.
> Il traduit le modèle canonique Halo en structures Go cibles, sans écrire encore le code de production.

## Rôle du document

Ce document répond à une question pratique du chantier Go :

une fois le modèle canonique défini, dans quelle forme Go doit-il exister pour rester propre, testable et découplé du legacy Python ?

Il ne remplace pas [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md).
Il le projette dans une forme d'implémentation Go plausible et disciplinée.

## Objectifs

1. éviter que les premiers types Go canoniques soient improvisés au moment de l'implémentation ;
2. prévenir un glissement vers des structs trop proches des payloads bruts SPNKr ;
3. clarifier les packages cibles et les règles de nullabilité ;
4. préparer les signatures d'interface des futurs providers de titre.

## Ce document ne fixe pas encore

1. les types OpenAPI générés côté handlers ;
2. les modèles DuckDB ;
3. les structs internes des analytics produit ;
4. les structs de persistance pour jobs, sync ou settings.

## Placement recommandé dans le futur repo Go

```text
apps/go-api/
  internal/
    halo/
      canonical/
        types.go
        enums.go
        bootstrap.go
      provider/
        provider.go
        errors.go
      titles/
        haloinfinite/
          mapper.go
```

## Règles de modélisation Go

1. Les types canoniques n'importent ni DuckDB, ni packages sync, ni packages UI.
2. Les types canoniques n'utilisent pas de noms de champs hérités directement des tables DuckDB.
3. Les valeurs optionnelles utilisent des pointeurs ou des wrappers explicites quand l'absence doit être distinguée de la valeur zéro.
4. Les slices connues mais vides restent des slices vides, pas des pointeurs vers slice.
5. Les enums orientées contrat utilisent des alias string stables.
6. Les timestamps utilisent `time.Time` en UTC ; la sérialisation JSON vient ensuite dans la couche API.

## Aliases et enums recommandés

```go
type TitleKey string
type ProviderKey string
type CapabilityKey string
type CapabilityStatus string
type CapabilitySeverity string
type EventConfidence string
type AssetKind string

const (
    TitleHaloInfinite TitleKey = "halo_infinite"
)

const (
    CapabilitySupported  CapabilityStatus = "supporte"
    CapabilityDegraded   CapabilityStatus = "degrade"
    CapabilityHidden     CapabilityStatus = "non_expose"
    CapabilityOutOfScope CapabilityStatus = "hors_scope"
)
```

## Types racine recommandés

### 1. Title Runtime Context

```go
type TitleRuntimeContext struct {
    TitleKey                TitleKey                     `json:"title"`
    ProviderKey             ProviderKey                  `json:"provider"`
    CapabilitySchemaVersion string                       `json:"capability_schema_version"`
    Capabilities            map[CapabilityKey]CapabilityStatus `json:"capabilities"`
    Limitations             []CapabilityGap              `json:"limitations,omitempty"`
}
```

### 2. Capability Gap

```go
type CapabilityGap struct {
    CapabilityKey CapabilityKey      `json:"capability_key"`
    ReasonCode    string             `json:"reason_code"`
    Severity      CapabilitySeverity `json:"severity"`
    Message       *string            `json:"message,omitempty"`
    Retryable     *bool              `json:"retryable,omitempty"`
}
```

### 3. Player Identity

```go
type PlayerIdentity struct {
    XUID               string  `json:"xuid"`
    Gamertag           string  `json:"gamertag"`
    GamertagNormalized *string `json:"gamertag_normalized,omitempty"`
    ServiceTag         *string `json:"service_tag,omitempty"`
    EmblemURL          *string `json:"emblem_url,omitempty"`
    AvatarURL          *string `json:"avatar_url,omitempty"`
    IsBot              *bool   `json:"is_bot,omitempty"`
}
```

### 4. Asset Reference

```go
type AssetReference struct {
    Kind         AssetKind         `json:"asset_kind"`
    AssetID      string            `json:"asset_id"`
    VersionID    *string           `json:"version_id,omitempty"`
    DefaultLabel *string           `json:"default_label,omitempty"`
    Labels       map[string]string `json:"labels,omitempty"`
    IconURL      *string           `json:"icon_url,omitempty"`
}
```

### 5. Match History Page et Item

```go
type MatchHistoryPage struct {
    Player       PlayerIdentity    `json:"player"`
    Start        int               `json:"start"`
    Count        int               `json:"count"`
    Items        []MatchHistoryItem `json:"items"`
    HasMore      *bool             `json:"has_more,omitempty"`
    TotalMatches *int              `json:"total_matches,omitempty"`
}

type MatchHistoryItem struct {
    MatchID         string          `json:"match_id"`
    StartedAtUTC    time.Time       `json:"started_at_utc"`
    DurationSeconds *int            `json:"duration_seconds,omitempty"`
    MatchType       *string         `json:"match_type,omitempty"`
    Playlist        *AssetReference `json:"playlist,omitempty"`
    Map             *AssetReference `json:"map,omitempty"`
    GameVariant     *AssetReference `json:"game_variant,omitempty"`
    IsRanked        *bool           `json:"is_ranked,omitempty"`
    IsPVE           *bool           `json:"is_pve,omitempty"`
    Outcome         *string         `json:"outcome,omitempty"`
}
```

### 6. Match Detail

```go
type MatchDetail struct {
    MatchID      string               `json:"match_id"`
    StartedAtUTC time.Time            `json:"started_at_utc"`
    EndedAtUTC   *time.Time           `json:"ended_at_utc,omitempty"`
    Playlist     *AssetReference      `json:"playlist,omitempty"`
    Map          *AssetReference      `json:"map,omitempty"`
    GameVariant  *AssetReference      `json:"game_variant,omitempty"`
    IsRanked     *bool                `json:"is_ranked,omitempty"`
    IsPVE        *bool                `json:"is_pve,omitempty"`
    Participants []MatchParticipant   `json:"participants"`
    Teams        []TeamSnapshot       `json:"teams,omitempty"`
    Skill        []MatchSkillSnapshot `json:"skill,omitempty"`
    Events       []MatchEvent         `json:"events,omitempty"`
    Film         *FilmReference       `json:"film,omitempty"`
    Limitations  []CapabilityGap      `json:"limitations,omitempty"`
}
```

### 7. Match Participant, Team Snapshot et Skill Snapshot

```go
type MatchParticipant struct {
    Identity        PlayerIdentity `json:"identity"`
    TeamID          *int           `json:"team_id,omitempty"`
    RankInMatch     *int           `json:"rank_in_match,omitempty"`
    Outcome         *string        `json:"outcome,omitempty"`
    Score           *int           `json:"score,omitempty"`
    Kills           *int           `json:"kills,omitempty"`
    Deaths          *int           `json:"deaths,omitempty"`
    Assists         *int           `json:"assists,omitempty"`
    Accuracy        *float64       `json:"accuracy,omitempty"`
    DamageDealt     *float64       `json:"damage_dealt,omitempty"`
    DamageTaken     *float64       `json:"damage_taken,omitempty"`
    ShotsFired      *int           `json:"shots_fired,omitempty"`
    ShotsHit        *int           `json:"shots_hit,omitempty"`
    HeadshotKills   *int           `json:"headshot_kills,omitempty"`
    MaxKillingSpree *int           `json:"max_killing_spree,omitempty"`
    PersonalScore   *int           `json:"personal_score,omitempty"`
}

type TeamSnapshot struct {
    TeamID            int      `json:"team_id"`
    Score             *int     `json:"score,omitempty"`
    MMR               *float64 `json:"mmr,omitempty"`
    ParticipantXUIDs  []string `json:"participants_xuids,omitempty"`
}

type MatchSkillSnapshot struct {
    PlayerXUID       string   `json:"player_xuid"`
    TeamMMR          *float64 `json:"team_mmr,omitempty"`
    EnemyMMR         *float64 `json:"enemy_mmr,omitempty"`
    KillsExpected    *float64 `json:"kills_expected,omitempty"`
    KillsStdDev      *float64 `json:"kills_stddev,omitempty"`
    DeathsExpected   *float64 `json:"deaths_expected,omitempty"`
    DeathsStdDev     *float64 `json:"deaths_stddev,omitempty"`
    AssistsExpected  *float64 `json:"assists_expected,omitempty"`
    AssistsStdDev    *float64 `json:"assists_stddev,omitempty"`
}
```

### 8. Match Event, Career, Customization et Film

```go
type MatchEvent struct {
    EventType         string          `json:"event_type"`
    TimeMS            int             `json:"time_ms"`
    PrimaryActorXUID  *string         `json:"primary_actor_xuid,omitempty"`
    SecondaryActorXUID *string        `json:"secondary_actor_xuid,omitempty"`
    Confidence        *EventConfidence `json:"confidence,omitempty"`
    RawHint           *string         `json:"raw_hint,omitempty"`
}

type CareerProgression struct {
    Player      PlayerIdentity          `json:"player"`
    CurrentRank *AssetReference         `json:"current_rank,omitempty"`
    CurrentXP   *int64                  `json:"current_xp,omitempty"`
    NextRank    *AssetReference         `json:"next_rank,omitempty"`
    History     []CareerProgressionItem `json:"history,omitempty"`
}

type CareerProgressionItem struct {
    Rank      AssetReference `json:"rank"`
    AchievedAt *time.Time    `json:"achieved_at,omitempty"`
    XP        *int64         `json:"xp,omitempty"`
}

type CustomizationSnapshot struct {
    Player          PlayerIdentity  `json:"player"`
    ArmorCore       *AssetReference `json:"armor_core,omitempty"`
    SpartanImageURL *string         `json:"spartan_image_url,omitempty"`
    EmblemURL       *string         `json:"emblem_url,omitempty"`
}

type FilmReference struct {
    MatchID       string   `json:"match_id"`
    FilmAvailable bool     `json:"film_available"`
    FilmID        *string  `json:"film_id,omitempty"`
    ChunkManifest *string  `json:"chunk_manifest,omitempty"`
    Limitations   []CapabilityGap `json:"limitations,omitempty"`
}
```

## Interface provider recommandée au niveau doc

Cette interface n'est pas le code final, mais la cible conceptuelle à protéger.

```go
type TitleProvider interface {
    Metadata() TitleRuntimeContext

    LookupByGamertag(ctx context.Context, gamertag string) (*PlayerIdentity, error)
    LookupByXUIDs(ctx context.Context, xuids []string) ([]PlayerIdentity, error)

    GetMatchHistory(ctx context.Context, player string, start int, count int) (*MatchHistoryPage, error)
    GetMatchDetail(ctx context.Context, matchID string, xuids []string) (*MatchDetail, error)

    GetCareerProgression(ctx context.Context, xuid string) (*CareerProgression, error)
    GetCustomization(ctx context.Context, xuid string) (*CustomizationSnapshot, error)

    GetFilmReference(ctx context.Context, matchID string) (*FilmReference, error)
    DownloadFilmChunk(ctx context.Context, url string) ([]byte, error)
}
```

## Règles d'interface

1. Les signatures retournent des types canoniques, jamais des payloads provider bruts.
2. Les erreurs transport, auth et rate limit restent des erreurs provider ; elles ne deviennent pas des faux objets vides.
3. Les données dérivées LevelUp ne doivent pas rentrer dans cette interface.

## Frontière avec les types OpenAPI

Le backend Go aura probablement deux couches de types :

1. types canoniques Halo ;
2. types contractuels OpenAPI de la façade produit.

Règle importante :

les types OpenAPI ne doivent pas devenir le modèle canonique, et le modèle canonique ne doit pas être forcé à refléter des détails de pagination, de wording ou de compat historique du frontend.

Un adaptateur explicite entre `internal/halo/canonical/` et `internal/api/` est donc attendu.

## Frontière avec DuckDB et analytics produit

Les types canoniques ne doivent pas contenir :

1. colonnes `mv_*` ;
2. flags de backfill ;
3. write leases ;
4. calculs de sessions ;
5. LUSR final ;
6. citations ;
7. réconciliation finale des armes comme vérité produit.

Ces surfaces appartiennent aux couches produit, pas au provider Halo.

## Documents compagnons

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour le cadre conceptuel.
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) pour les surfaces réellement supportées aujourd'hui.
3. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) pour la projection bootstrap côté produit.
4. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md) pour la discipline de mapping depuis Halo Infinite.
5. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la projection canonique vers les contrats produit.
6. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) pour le rattachement aux phases du programme.
