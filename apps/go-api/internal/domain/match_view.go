// Package domain — types pour la vue détail d'un match.
//
// Port Go de apps/api/app/schemas/match_view.py.
// Route : GET /api/v1/players/{slug}/matches/{match_id}
package domain

import "time"

// MatchViewResponse est la réponse complète de la vue match.
type MatchViewResponse struct {
	Header      MatchViewHeader   `json:"header"`
	Rank        MatchViewRank     `json:"rank"`
	SummaryTab  MatchSummaryTab   `json:"summary_tab"`
	CombatTab   MatchCombatTab    `json:"combat_tab"`
	TeamTab     MatchTeamTab      `json:"team_tab"`
	MediaTab    MatchMediaTab     `json:"media_tab"`
	CitationsTab MatchCitationsTab `json:"citations_tab"`
}

// MatchViewHeader : en-tête du match.
type MatchViewHeader struct {
	MatchID          string     `json:"match_id"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	StartTimeLabel   string     `json:"start_time_label"`
	OutcomeCode      *int       `json:"outcome_code,omitempty"`
	OutcomeLabel     string     `json:"outcome_label"`
	OutcomeColor     string     `json:"outcome_color"`
	ScoreLabel       string     `json:"score_label,omitempty"`
	DominanceFlag    bool       `json:"dominance_flag"`
	HadBotTeammate   bool       `json:"had_bot_teammate"`
	MapUI            string     `json:"map_ui"`
	MapID            string     `json:"map_id,omitempty"`
	ModeUI           string     `json:"mode_ui"`
	PlaylistLabel    string     `json:"playlist_label"`
	PerfDisplay      string     `json:"performance_display"`
	PerfColor        *string    `json:"performance_color,omitempty"`
}

// MatchViewRank : rang CSR ou LUSR pour ce match.
type MatchViewRank struct {
	RatingType  string   `json:"rating_type"`
	TierLabel   *string  `json:"tier_label,omitempty"`
	NumericVal  *float64 `json:"numeric_value,omitempty"`
	DeltaValue  *float64 `json:"delta_value,omitempty"`
}

// ---------------------------------------------------------------------------
// Onglet résumé
// ---------------------------------------------------------------------------

// MatchSummaryKpis : KPIs personnels du résumé.
type MatchSummaryKpis struct {
	Kills         *int     `json:"kills,omitempty"`
	Deaths        *int     `json:"deaths,omitempty"`
	Assists       *int     `json:"assists,omitempty"`
	KDA           *float64 `json:"kda,omitempty"`
	DamageDealt   *float64 `json:"damage_dealt,omitempty"`
	AverageLife   string   `json:"average_life,omitempty"`
}

// MatchPersonalResult : résultat personnel du joueur.
type MatchPersonalResult struct {
	OutcomeLabel string  `json:"outcome_label"`
	OutcomeColor string  `json:"outcome_color"`
	Score        *int    `json:"score,omitempty"`
	RankInTeam   *int    `json:"rank_in_team,omitempty"`
}

// MatchExpectedStats : comparaison réel vs attendu.
type MatchExpectedStats struct {
	HasExpectedData  bool     `json:"has_expected_data"`
	ExpectedKills    *float64 `json:"expected_kills,omitempty"`
	ExpectedDeaths   *float64 `json:"expected_deaths,omitempty"`
	ExpectedAssists  *float64 `json:"expected_assists,omitempty"`
}

// MatchMedal : une médaille gagnée dans le match.
type MatchMedal struct {
	MedalNameID int64   `json:"medal_name_id"`
	Name        string  `json:"name"`
	Count       int     `json:"count"`
	Description *string `json:"description,omitempty"`
}

// MatchCitation : badge de citation associé au match.
type MatchCitation struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Color *string `json:"color,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// MatchSummaryTab : contenu de l'onglet Résumé.
type MatchSummaryTab struct {
	KPIs           MatchSummaryKpis    `json:"kpis"`
	PersonalResult MatchPersonalResult `json:"personal_result"`
	Medals         []MatchMedal        `json:"medals"`
	Citations      []MatchCitation     `json:"citations"`
	ExpectedStats  MatchExpectedStats  `json:"expected_stats"`
}

// ---------------------------------------------------------------------------
// Onglet combat
// ---------------------------------------------------------------------------

// MatchWeaponKill : kills par arme.
type MatchWeaponKill struct {
	WeaponID    int64  `json:"weapon_id"`
	WeaponLabel string `json:"weapon_label"`
	KillCount   int    `json:"kill_count"`
}

// MatchHighlightEvent : événement filmé horodaté.
type MatchHighlightEvent struct {
	EventType   string  `json:"event_type"`
	TickCount   *int64  `json:"tick_count,omitempty"`
	ActorXUID   *string `json:"actor_xuid,omitempty"`
}

// MatchCombatTab : contenu de l'onglet Combat.
type MatchCombatTab struct {
	WeaponKills     []MatchWeaponKill     `json:"weapon_kills"`
	HighlightEvents []MatchHighlightEvent `json:"highlight_events"`
	Charts          []interface{}         `json:"charts"`
}

// ---------------------------------------------------------------------------
// Onglet équipe
// ---------------------------------------------------------------------------

// MatchScoreboardRow : ligne du scoreboard (19+ colonnes).
type MatchScoreboardRow struct {
	XUID          string   `json:"xuid"`
	Gamertag      string   `json:"gamertag"`
	TeamSide      *string  `json:"team_side,omitempty"`
	IsMe          bool     `json:"is_me"`
	Rank          *int     `json:"rank,omitempty"`
	Kills         *int     `json:"kills,omitempty"`
	Deaths        *int     `json:"deaths,omitempty"`
	Assists       *int     `json:"assists,omitempty"`
	KDA           *float64 `json:"kda,omitempty"`
	Accuracy      *float64 `json:"accuracy,omitempty"`
	DamageDealt   *float64 `json:"damage_dealt,omitempty"`
	DamageTaken   *float64 `json:"damage_taken,omitempty"`
	ShotsFired    *int     `json:"shots_fired,omitempty"`
	ShotsHit      *int     `json:"shots_hit,omitempty"`
	OutcomeLabel  string   `json:"outcome_label"`
}

// MatchNemesisRow : adversaire fréquent (kills reçus de lui).
type MatchNemesisRow struct {
	XUID      string `json:"xuid"`
	Gamertag  string `json:"gamertag"`
	KilledMe  int    `json:"killed_me"`
	IKilled   int    `json:"i_killed"`
}

// MatchTeamTab : contenu de l'onglet Équipe.
type MatchTeamTab struct {
	Scoreboard []MatchScoreboardRow `json:"scoreboard"`
	Nemesis    []MatchNemesisRow    `json:"nemesis"`
}

// ---------------------------------------------------------------------------
// Onglets médias et citations
// ---------------------------------------------------------------------------

// MatchMediaTab : contenu de l'onglet Médias.
type MatchMediaTab struct {
	MediaItems []interface{} `json:"media_items"`
}

// MatchCitationsTab : contenu de l'onglet Citations.
type MatchCitationsTab struct {
	Commendations []MatchCitation `json:"commendations"`
	Medals        []MatchMedal    `json:"medals"`
}

// ---------------------------------------------------------------------------
// Types raw DB (non exportés vers JSON)
// ---------------------------------------------------------------------------

// MatchMetaRaw : données brutes de la requête Q13 (match_registry).
type MatchMetaRaw struct {
	MatchID         string
	StartTime       *time.Time
	DurationSeconds *float64
	MapName         *string
	PairName        *string
	PlaylistName    *string
	IsFirefight     bool
	IsRanked        bool
}

// PlayerMatchStatsRaw : données brutes de Q17 (match_participants filtré par xuid).
type PlayerMatchStatsRaw struct {
	OutcomeCode       int
	TeamID            *int
	RankInTeam        *int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *float64
	AvgLifeSeconds    *float64
	TimePlayedSeconds *float64
	ShotsFired        *int
	ShotsHit          *int
	DamageDealt       *float64
	DamageTaken       *float64
}

// ScoreboardRaw : données brutes de Q12 (une ligne du scoreboard).
type ScoreboardRaw struct {
	XUID          string
	Gamertag      string
	TeamID        *int
	RankInTeam    *int
	OutcomeCode   int
	PersonalScore *float64
	Kills         int
	Deaths        int
	Assists       int
	KDA           *float64
	Accuracy      *float64
	TimePlayed    *float64
	TeamMMR       *float64
	EnemyMMR      *float64
}

// MatchEnrichmentRaw : données brutes de Q18.
type MatchEnrichmentRaw struct {
	PerformanceScore *float64
	IsWithFriends    bool
}

// MedalRaw : données brutes de Q14.
type MedalRaw struct {
	MedalID int64
	Count   int
	Label   string
}

// EventRaw : données brutes de Q21.
type EventRaw struct {
	EventType string
	TickCount *int64
	XUID      *string
}

// WeaponKillRaw : données brutes de Q16.
type WeaponKillRaw struct {
	WeaponID    int64
	WeaponLabel string
	Kills       int
}

// KVPairRaw : données brutes de Q20.
type KVPairRaw struct {
	KillerXUID string
	KillerGT   string
	VictimXUID string
	VictimGT   string
	KillCount  int
	TimeMS     int64
}

// MatchViewRawRow : DEPRECATED — conservé le temps de migrer les appelants.
// Préférer MatchMetaRaw + PlayerMatchStatsRaw.
type MatchViewRawRow = struct {
	MatchID           string
	StartTime         *time.Time
	DurationSeconds   *float64
	MapName           *string
	PairName          *string
	PlaylistName      *string
	IsFirefight       bool
	IsRanked          bool
	OutcomeCode       int
	TeamID            *int
	RankInTeam        *int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *float64
	AvgLifeSeconds    *float64
	TimePlayedSeconds *float64
	ShotsFired        *int
	ShotsHit          *int
	DamageDealt       *float64
	DamageTaken       *float64
}
