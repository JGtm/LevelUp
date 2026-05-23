// Package persist — rows.go : structs pour les tables qui n'ont pas encore
// de Row type dans `internal/domain/`. Définies ici pour démarrer Phase 1
// sans bloquer sur un refactor exhaustif des domain types.
//
// Migration future : ces types pourraient migrer vers `internal/domain/`
// si d'autres packages (analysis) en ont besoin. Pour l'instant, ils sont
// internes à `persist`.

package persist

import "time"

// EnrichmentRow — UNE row complète pour player.player_match_enrichment.
//
// **Property clé** : tous les enrichments computed pour 1 match sont dans
// CE struct. Pas de UPSERT depuis 4 callers différents (comme avant) —
// 1 seul INSERT batch avec tous les champs.
//
// Champs pointers : un champ nil signifie "ne pas écrire cette colonne".
// Le `buildEnrichmentInsertSQL` (sql_builder.go) construit l'INSERT
// dynamiquement en fonction des champs non-nil.
//
// **Extensibilité** : ajouter un enrichment = ajouter un champ pointer +
// 1 branche dans `buildEnrichmentInsertSQL`. Cf. `doc.go`.
type EnrichmentRow struct {
	// PK
	MatchID string `json:"match_id"`

	// Performance score (post-sync compute)
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	PerformanceChain *string  `json:"performance_chain,omitempty"`

	// Dominance flag (computed depuis Steaktacular medals + outcome)
	DominanceFlag *int `json:"dominance_flag,omitempty"`

	// Engagement (computed depuis pace player/team/lobby)
	EngagementScore           *float64 `json:"engagement_score,omitempty"`
	EngagementScoreBrut       *float64 `json:"engagement_score_brut,omitempty"`
	EngagementScoreConfidence *string  `json:"engagement_score_confidence,omitempty"` // "full" / "partial" / "insufficient_history"
	EngagementPacePlayer      *float64 `json:"engagement_pace_player,omitempty"`
	EngagementPaceTeam        *float64 `json:"engagement_pace_team,omitempty"`
	EngagementPaceLobby       *float64 `json:"engagement_pace_lobby,omitempty"`
	EngagementPlayerActivity  *int     `json:"engagement_player_activity,omitempty"` // INTEGER en DB
	ModeCategory              *string  `json:"mode_category,omitempty"`

	// Session (groupage temporel)
	SessionID    *string `json:"session_id,omitempty"`
	SessionLabel *string `json:"session_label,omitempty"`

	// Friends / squad (post-sync compute)
	IsWithFriends      *bool   `json:"is_with_friends,omitempty"`
	TeammatesSignature *string `json:"teammates_signature,omitempty"`
	HadBotTeammate     *bool   `json:"had_bot_teammate,omitempty"`

	// Timestamps
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// HighlightEventInsert — row pour shared.highlight_events.
type HighlightEventInsert struct {
	MatchID     string  `json:"match_id"`
	XUID        *string `json:"xuid,omitempty"`
	EventType   string  `json:"event_type"`
	TimeMS      int     `json:"time_ms"`
	DetailsJSON *string `json:"details_json,omitempty"`
}

// WeaponKillInsert — row pour shared.weapon_kills.
type WeaponKillInsert struct {
	MatchID         string  `json:"match_id"`
	XUID            string  `json:"xuid"`
	TimeMS          int     `json:"time_ms"`
	WeaponID        *uint64 `json:"weapon_id,omitempty"`
	ReconciledAs    *uint64 `json:"reconciled_as,omitempty"`
	DeltaMS         *int    `json:"delta_ms,omitempty"`
	Confidence      string  `json:"confidence"`
	AttributionPath string  `json:"attribution_path"`
	SwapDetected    bool    `json:"swap_detected"`
	DelayedDamage   bool    `json:"delayed_damage"`
	PlayerIndex     *int    `json:"player_index,omitempty"`
}

// KillerVictimInsert — row pour shared.killer_victim_pairs.
type KillerVictimInsert struct {
	MatchID    string  `json:"match_id"`
	KillerXUID string  `json:"killer_xuid"`
	VictimXUID string  `json:"victim_xuid"`
	Count      int     `json:"count"`
	WeaponID   *uint64 `json:"weapon_id,omitempty"`
}

// XUIDAliasInsert — row pour shared.xuid_aliases.
type XUIDAliasInsert struct {
	XUID     string    `json:"xuid"`
	Gamertag string    `json:"gamertag"`
	LastSeen time.Time `json:"last_seen"`
}

// SkillRankInsert — row pour player.match_skill_rank.
type SkillRankInsert struct {
	MatchID         string   `json:"match_id"`
	RatingType      string   `json:"rating_type"` // "LUSR" ou "CSR"
	RatingValue     *float64 `json:"rating_value,omitempty"`
	RatingDeviation *float64 `json:"rating_deviation,omitempty"`
	Tier            *string  `json:"tier,omitempty"`
	TierFR          *string  `json:"tier_fr,omitempty"`
	SubTier         *int     `json:"sub_tier,omitempty"`
	TierLabel       *string  `json:"tier_label,omitempty"`
	RatingDelta     *float64 `json:"rating_delta,omitempty"`
	PlaylistGroup   *string  `json:"playlist_group,omitempty"`
}

// LUSRComponentInsert — row pour player.lusr_component_history (breakdown).
type LUSRComponentInsert struct {
	MatchID       string  `json:"match_id"`
	ComponentName string  `json:"component_name"`
	Value         float64 `json:"value"`
	Weight        float64 `json:"weight"`
}

// CitationInsert — row pour player.match_citations.
type CitationInsert struct {
	MatchID          string  `json:"match_id"`
	CitationNameNorm string  `json:"citation_name_norm"`
	Value            float64 `json:"value"`
}

// PersonalScoreAwardInsert — row pour player.personal_score_awards.
//
// Schema cf. sync/schema.go : id (seq auto), match_id, xuid, award_name,
// award_category, award_count, award_score, created_at (DEFAULT).
type PersonalScoreAwardInsert struct {
	MatchID       string `json:"match_id"`
	XUID          string `json:"xuid"`
	AwardName     string `json:"award_name"`
	AwardCategory string `json:"award_category"`
	AwardCount    int    `json:"award_count"`
	AwardScore    int    `json:"award_score"`
}

// CareerProgressionInsert — row pour player.career_progression.
type CareerProgressionInsert struct {
	XUID             string    `json:"xuid"`
	Rank             int       `json:"rank"`
	RankName         string    `json:"rank_name"`
	RankTier         string    `json:"rank_tier"`
	CurrentXP        int       `json:"current_xp"`
	XPForNextRank    int       `json:"xp_for_next_rank"`
	XPTotal          int       `json:"xp_total"`
	IsMaxRank        bool      `json:"is_max_rank"`
	AdornmentPath    string    `json:"adornment_path"`
	SpartanID        string    `json:"spartan_id"`
	BannerImageURL   string    `json:"banner_image_url"`
	EmblemImageURL   string    `json:"emblem_image_url"`
	BackdropImageURL string    `json:"backdrop_image_url"`
	RecordedAt       time.Time `json:"recorded_at"`
}

// SessionInsert — row pour player.sessions (groupage temporel des matchs).
type SessionInsert struct {
	SessionID  string    `json:"session_id"`
	Label      string    `json:"label"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	MatchCount int       `json:"match_count"`
}

// PVEMatchStatsInsert — row pour shared_pve.pve_match_stats (Firefight).
//
// PveBits : bitmask de complétion par colonne kills (PveBitTotalKills,
// PveBitBossKills, PveBitGrunt…Warden). Calculé par le collecteur au moment
// où il sait quelles kills counts sont fiables. Set à l'INSERT en mode
// Collect→Persist (plus de UPDATE post-coup).
type PVEMatchStatsInsert struct {
	MatchID      string `json:"match_id"`
	XUID         string `json:"xuid"`
	Waves        int    `json:"waves"`
	BossKills    int    `json:"boss_kills"`
	GruntKills   int    `json:"grunt_kills"`
	EliteKills   int    `json:"elite_kills"`
	JackalKills  int    `json:"jackal_kills"`
	BruteKills   int    `json:"brute_kills"`
	HunterKills  int    `json:"hunter_kills"`
	SkimmerKills int    `json:"skimmer_kills"`
	CrawlerKills int    `json:"crawler_kills"`
	SoldierKills int    `json:"soldier_kills"`
	KnightKills  int    `json:"knight_kills"`
	WardenKills  int    `json:"warden_kills"`
	PveBits      *int   `json:"pve_bits,omitempty"`
}

// MatchCSRInsert — row pour shared.match_csrs (CSR de tous les participants
// d'un match ranked, pas seulement le joueur synchronisé). PK (match_id, xuid).
//
// Distincte de SkillRankInsert (qui vit en player.match_skill_rank et stocke
// le CSR/LUSR du joueur synchronisé pour ses propres analyses). match_csrs
// permet d'afficher les ratings des coéquipiers/adversaires dans match view.
type MatchCSRInsert struct {
	MatchID                     string   `json:"match_id"`
	XUID                        string   `json:"xuid"`
	RatingType                  string   `json:"rating_type"` // "CSR" (forcé pour cette table)
	RatingValue                 *float64 `json:"rating_value,omitempty"`
	Tier                        *string  `json:"tier,omitempty"`
	SubTier                     *int     `json:"sub_tier,omitempty"`
	TierLabel                   *string  `json:"tier_label,omitempty"`
	RatingDelta                 *float64 `json:"rating_delta,omitempty"`
	MeasurementMatchesRemaining *int     `json:"measurement_matches_remaining,omitempty"`
	SeasonID                    *string  `json:"season_id,omitempty"`
}

// ModeNameTranslationInsert — row pour metadata.mode_name_tr.
type ModeNameTranslationInsert struct {
	RawName        string `json:"raw_name"`
	Lang           string `json:"lang"`
	TranslatedName string `json:"translated_name"`
}
