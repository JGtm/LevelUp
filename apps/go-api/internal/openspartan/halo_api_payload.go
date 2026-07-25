// halo_api_payload.go — types miroir des réponses de l'API Halo officielle.
//
// Renommé depuis `models.go` le 2026-07-26 (v7.2.1, V721-11, arbitrage utilisateur).
// Raison : ces structs ne décrivent AUCUN format propre à OpenSpartan — elles
// décrivent le JSON de l'API Halo, qu'OpenSpartan/grunt se contente de stocker
// verbatim dans `MatchStats.ResponseBody`. Le nom `models.go` sous un paquet
// `openspartan` laissait croire l'inverse au premier coup d'œil. Aucun changement
// de comportement : ces types ne sortent jamais du duo `openspartan`/`mapper`.
package openspartan

import (
	"encoding/json"
	"time"
)

// MatchStats mirrors the top-level JSON shape of the Halo Infinite
// match-stats API response, as stored verbatim by OpenSpartan/grunt in
// MatchStats.ResponseBody.
//
// Fields whose Halo API shape is not yet load-bearing for PR 1 are kept as
// json.RawMessage so we can pass them through without committing to a schema
// the mapper PR 2 may refine.
type MatchStats struct {
	MatchID   string    `json:"MatchId"`
	MatchInfo MatchInfo `json:"MatchInfo"`
	Teams     []Team    `json:"Teams"`
	Players   []Player  `json:"Players"`
}

type MatchInfo struct {
	StartTime           time.Time `json:"StartTime"`
	EndTime             time.Time `json:"EndTime"`
	Duration            string    `json:"Duration"`      // ISO 8601, e.g. "PT11M59.2855382S"
	LifecycleMode       int       `json:"LifecycleMode"` // 3 = matchmaking (typical)
	GameVariantCategory int       `json:"GameVariantCategory"`
	LevelID             string    `json:"LevelId"`
	MapVariant          AssetRef  `json:"MapVariant"`
	UgcGameVariant      AssetRef  `json:"UgcGameVariant"`
	Playlist            *AssetRef `json:"Playlist,omitempty"`
	PlaylistExperience  int       `json:"PlaylistExperience,omitempty"`
	PlaylistMapModePair *AssetRef `json:"PlaylistMapModePair,omitempty"`
	ClearanceID         string    `json:"ClearanceId,omitempty"`
	SeasonID            string    `json:"SeasonId,omitempty"`
	PlayableDuration    string    `json:"PlayableDuration"`
	TeamsEnabled        bool      `json:"TeamsEnabled"`
	TeamScoringEnabled  bool      `json:"TeamScoringEnabled"`
	GameplayInteraction int       `json:"GameplayInteraction,omitempty"`
}

type AssetRef struct {
	AssetKind int    `json:"AssetKind"`
	AssetID   string `json:"AssetId"`
	VersionID string `json:"VersionId"`
}

type Team struct {
	TeamID  int             `json:"TeamId"`
	Outcome int             `json:"Outcome"`
	Rank    int             `json:"Rank"`
	Stats   json.RawMessage `json:"Stats,omitempty"`
}

type Player struct {
	PlayerID          string            `json:"PlayerId"`   // "xuid(<digits>)" or "bid(<digits>)" for bots
	PlayerType        int               `json:"PlayerType"` // 1 = human, other = bot (verify per match)
	BotAttributes     json.RawMessage   `json:"BotAttributes,omitempty"`
	LastTeamID        int               `json:"LastTeamId"`
	Outcome           int               `json:"Outcome"`
	Rank              int               `json:"Rank"`
	ParticipationInfo ParticipationInfo `json:"ParticipationInfo"`
	PlayerTeamStats   []PlayerTeamStat  `json:"PlayerTeamStats"`
}

type ParticipationInfo struct {
	FirstJoinedTime        time.Time  `json:"FirstJoinedTime"`
	LastLeaveTime          *time.Time `json:"LastLeaveTime,omitempty"`
	PresentAtBeginning     bool       `json:"PresentAtBeginning"`
	JoinedInProgress       bool       `json:"JoinedInProgress"`
	LeftInProgress         bool       `json:"LeftInProgress"`
	PresentAtCompletion    bool       `json:"PresentAtCompletion"`
	TimePlayed             string     `json:"TimePlayed"`
	ConfirmedParticipation *bool      `json:"ConfirmedParticipation,omitempty"`
}

type PlayerTeamStat struct {
	TeamID int         `json:"TeamId"`
	Stats  StatsBundle `json:"Stats"`
}

// StatsBundle keeps only CoreStats typed (used by PR 2's mapper for medals,
// kill/death, accuracy). Mode/PvP-specific stats are left raw and surfaced as
// json.RawMessage so the mapper can decide what to extract.
type StatsBundle struct {
	CoreStats           CoreStats       `json:"CoreStats"`
	PvpStats            json.RawMessage `json:"PvpStats,omitempty"`
	PveStats            json.RawMessage `json:"PveStats,omitempty"`
	CaptureTheFlagStats json.RawMessage `json:"CaptureTheFlagStats,omitempty"`
	EliminationStats    json.RawMessage `json:"EliminationStats,omitempty"`
	ExtractionStats     json.RawMessage `json:"ExtractionStats,omitempty"`
	InfectionStats      json.RawMessage `json:"InfectionStats,omitempty"`
	OddballStats        json.RawMessage `json:"OddballStats,omitempty"`
	ZonesStats          json.RawMessage `json:"ZonesStats,omitempty"`
	StockpileStats      json.RawMessage `json:"StockpileStats,omitempty"`
	// VipStats : bloc RÉEL du mode Arena:VIP, absent de l'inventaire initial —
	// ajouté V721-02 après capture du payload (GameVariantCategory 23).
	VipStats json.RawMessage `json:"VipStats,omitempty"`
}

// CoreStats is the per-player-per-team core stats block.
// NameId fields for medals are uint32 (filmshell IDs, up to ~4.3 billion).
type CoreStats struct {
	Score                 int          `json:"Score"`
	PersonalScore         int          `json:"PersonalScore"`
	RoundsWon             int          `json:"RoundsWon"`
	RoundsLost            int          `json:"RoundsLost"`
	RoundsTied            int          `json:"RoundsTied"`
	Kills                 int          `json:"Kills"`
	Deaths                int          `json:"Deaths"`
	Assists               int          `json:"Assists"`
	KDA                   float64      `json:"KDA"`
	Suicides              int          `json:"Suicides"`
	Betrayals             int          `json:"Betrayals"`
	AverageLifeDuration   string       `json:"AverageLifeDuration"` // ISO 8601
	GrenadeKills          int          `json:"GrenadeKills"`
	HeadshotKills         int          `json:"HeadshotKills"`
	MeleeKills            int          `json:"MeleeKills"`
	PowerWeaponKills      int          `json:"PowerWeaponKills"`
	ShotsFired            int          `json:"ShotsFired"`
	ShotsHit              int          `json:"ShotsHit"`
	Accuracy              float64      `json:"Accuracy"`
	DamageDealt           int          `json:"DamageDealt"`
	DamageTaken           int          `json:"DamageTaken"`
	CalloutAssists        int          `json:"CalloutAssists"`
	VehicleDestroys       int          `json:"VehicleDestroys"`
	DriverAssists         int          `json:"DriverAssists"`
	Hijacks               int          `json:"Hijacks"`
	EmpAssists            int          `json:"EmpAssists"`
	MaxKillingSpree       int          `json:"MaxKillingSpree"`
	Medals                []MedalAward `json:"Medals,omitempty"`
	PersonalScores        []ScoreAward `json:"PersonalScores,omitempty"`
	Spawns                int          `json:"Spawns,omitempty"`
	ObjectivesCompleted   int          `json:"ObjectivesCompleted,omitempty"`
	DeprecatedDamageDealt float64      `json:"DeprecatedDamageDealt,omitempty"`
	DeprecatedDamageTaken float64      `json:"DeprecatedDamageTaken,omitempty"`
}

type MedalAward struct {
	NameID                    uint32 `json:"NameId"`
	Count                     int    `json:"Count"`
	TotalPersonalScoreAwarded int    `json:"TotalPersonalScoreAwarded"`
}

type ScoreAward struct {
	NameID                    uint32 `json:"NameId"`
	Count                     int    `json:"Count"`
	TotalPersonalScoreAwarded int    `json:"TotalPersonalScoreAwarded"`
}

// PlayerMatchStatsValue mirrors one item of PlayerMatchStats.ResponseBody.$.Value
// — an array, one entry per requested player. Provides advanced stats not in
// MatchStats (TeamMmr, CSR snapshots, StatPerformances expected/stddev).
type PlayerMatchStatsValue struct {
	ID         string                  `json:"Id"` // "xuid(<digits>)"
	ResultCode int                     `json:"ResultCode"`
	Result     *PlayerMatchStatsResult `json:"Result,omitempty"`
}

type PlayerMatchStatsResult struct {
	TeamMmr          float64         `json:"TeamMmr"`
	RankRecap        json.RawMessage `json:"RankRecap,omitempty"`
	StatPerformances json.RawMessage `json:"StatPerformances,omitempty"`
}

// ParsedMatch is the per-match unit yielded by the iterator. It bundles the
// MatchStats payload with any advanced PlayerMatchStats entries we found by
// joining on MatchId, plus the raw JSON bytes for downstream tools that need
// to re-emit fields we did not type.
type ParsedMatch struct {
	MatchID        string
	Stats          MatchStats
	PlayerStats    []PlayerMatchStatsValue
	RawMatchStats  json.RawMessage
	RawPlayerStats json.RawMessage // null if no PlayerMatchStats row joined
}

// Confidence describes how certain DetectOwner is about the returned XUID.
type Confidence int

const (
	// ConfidenceNone means no heuristic produced a candidate.
	ConfidenceNone Confidence = iota
	// ConfidenceLow means a single weak heuristic (e.g. filename hint only)
	// produced a candidate but no corroboration was found.
	ConfidenceLow
	// ConfidenceMedium means at least one strong heuristic produced a candidate.
	ConfidenceMedium
	// ConfidenceHigh means multiple heuristics independently agree on the same XUID.
	ConfidenceHigh
)

func (c Confidence) String() string {
	switch c {
	case ConfidenceHigh:
		return "high"
	case ConfidenceMedium:
		return "medium"
	case ConfidenceLow:
		return "low"
	default:
		return "none"
	}
}
