package canonical

import "time"

// AssetReference est l'objet canonique pour les assets Halo référencés par
// le produit (maps, playlists, modes, ranks, médailles).
type AssetReference struct {
	Kind         string            // "map", "playlist", "game_variant", "career_rank", ...
	ID           string            // identifiant source stable
	VersionID    string            // optionnel
	DefaultLabel string            // libellé par défaut sans i18n locale avancée
	Labels       map[string]string // locale -> label
	IconURL      string
}

// MatchSummary est la version légère d'un match pour un historique paginé.
type MatchSummary struct {
	MatchID         string
	StartedAtUTC    time.Time
	DurationSeconds *int
	MatchType       MatchType
	Playlist        *AssetReference
	Map             *AssetReference
	GameVariant     *AssetReference
	IsRanked        *bool
	IsPvE           *bool
	Outcome         Outcome
}

// MatchDetail est l'objet canonique central d'un match côté services.
type MatchDetail struct {
	MatchID      string
	StartedAtUTC time.Time
	EndedAtUTC   *time.Time
	Playlist     *AssetReference
	Map          *AssetReference
	GameVariant  *AssetReference
	IsRanked     *bool
	IsPvE        *bool
	MatchType    MatchType
	Participants []MatchParticipant
	Teams        []TeamSnapshot
	Skill        *MatchSkillSnapshot
	Limitations  []CapabilityGap
}

// MatchParticipant représente un joueur d'un match dans le canonique.
type MatchParticipant struct {
	Identity        PlayerIdentity
	TeamID          *int
	RankInMatch     *int
	Outcome         Outcome
	Score           *int
	Kills           *int
	Deaths          *int
	Assists         *int
	HeadshotKills   *int
	Accuracy        *float64
	DamageDealt     *int
	DamageTaken     *int
	ShotsFired      *int
	ShotsHit        *int
	MaxKillingSpree *int
	PersonalScore   *int

	// Champs étendus scoreboard (null si non chargés par LoadMatchScoreboard)
	KDA              *float64
	TimePlayed       *int
	MeleeKills       *int
	GrenadeKills     *int
	PowerWeaponKills *int
	AvgLifeSeconds   *float64
	PerfectKills     *int
	TopWeaponID      *string // effective_weapon_id converti en string
	IsBot            *bool
}

// HighlightEvent est un événement horodaté issu de highlight_events (kill ou death).
// Pour un event "kill", XUID = tueur ; pour "death", XUID = victime.
type HighlightEvent struct {
	EventType string // "kill" | "death"
	TimeMS    int64
	XUID      string
}

// ImpactBadge est un badge d'impact calculé sur les événements d'un match.
type ImpactBadge struct {
	BadgeKey   string // identifiant technique : first_blood, clutch_finisher…
	BadgeFR    string // libellé français affiché
	PlayerXUID string
}

// TeamSnapshot est la vue légère d'une équipe d'un match.
type TeamSnapshot struct {
	TeamID            int
	Score             *int
	MMR               *float64
	ParticipantsXUIDs []string
}

// MatchSkillSnapshot contient les données natives de skill d'un match.
type MatchSkillSnapshot struct {
	PlayerXUID      string
	TeamMMR         *float64
	EnemyMMR        *float64
	KillsExpected   *float64
	KillsStdDev     *float64
	DeathsExpected  *float64
	DeathsStdDev    *float64
	AssistsExpected *float64
	AssistsStdDev   *float64
}

// CapabilityGap représente une limitation explicite reportée à un consommateur.
type CapabilityGap struct {
	CapabilityKey string // ex: "match.skill.snapshot"
	ReasonCode    string // cause normalisée
	Severity      string // "info" | "warning" | "blocking"
	Message       string
	Retryable     bool
}
