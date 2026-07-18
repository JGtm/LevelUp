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
// L'INSERT dynamique est construit en fonction des champs non-nil par
// `player_persister.go::enrichmentFields()` (il n'existe pas de sql_builder.go).
//
// **Extensibilité** : ajouter un enrichment = ajouter un champ pointer +
// 1 entrée dans `enrichmentFields()`. Cf. `doc.go`.
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
	// KillKind : mecanique du kill (Halo 5 natif : weapon/melee/groundpound/
	// shoulderbash). Vide => NULL en base (Infinite ne porte pas cette mecanique).
	// Capture Phase 1 ; exploitation (donut) + backfill = Phase 2.
	KillKind string `json:"kill_kind,omitempty"`
}

// WeaponAccuracyInsert — row pour shared.weapon_accuracy : agrégat par
// (match_id, xuid, weapon_id) des tirs d'une arme sur un match. Reconstruit la
// précision par arme là où le carnage ne la sert pas (Halo 5 : WeaponStats[]
// servi vide → dérivé des events WeaponDrop ; la somme par joueur = carnage
// TotalShotsFired/Landed, validé exact). INSERT pur (table sans index/PK —
// ART-safe, idempotence via l'ancre match_registry comme killer_victim_pairs).
type WeaponAccuracyInsert struct {
	MatchID     string `json:"match_id"`
	XUID        string `json:"xuid"`
	WeaponID    uint64 `json:"weapon_id"`
	ShotsFired  int    `json:"shots_fired"`
	ShotsLanded int    `json:"shots_landed"`
	Drops       int    `json:"drops"` // nb de WeaponDrop avec tirs agrégés (usage de l'arme)
}

// KillerVictimInsert — row pour shared.killer_victim_pairs (forme **par-kill**,
// 1 row par kill event — cf. analysis.ComputeKillerVictimPairs).
//
// KillerGamertag / VictimGamertag / TimeMS sont REQUIS par le match-view
// (tug-of-war, KD timeline, antagonistes — cf. queries_match.go Q20KVPairs qui
// lit ces colonnes). Les laisser vides produit la forme dégradée (kill_count
// seul) qui casse ces vues. Même schéma cible que la complétion legacy
// (EventsCompletionPersister.KVPairCompletion).
type KillerVictimInsert struct {
	MatchID        string  `json:"match_id"`
	KillerXUID     string  `json:"killer_xuid"`
	KillerGamertag string  `json:"killer_gamertag,omitempty"`
	VictimXUID     string  `json:"victim_xuid"`
	VictimGamertag string  `json:"victim_gamertag,omitempty"`
	Count          int     `json:"count"`
	TimeMS         int64   `json:"time_ms,omitempty"`
	WeaponID       *uint64 `json:"weapon_id,omitempty"`
}

// KillPositionInsert — row pour shared.kill_positions (positions monde tueur/
// victime par kill, jointes au kill par (match_id, killer_xuid, time_ms)).
// Coordonnées nullables : un kill peut n'avoir que la position du tueur. Halo 5
// les fournit nativement ; Infinite les laisse nil jusqu'au câblage monde du film.
type KillPositionInsert struct {
	MatchID    string   `json:"match_id"`
	KillerXUID string   `json:"killer_xuid"`
	TimeMS     int      `json:"time_ms"`
	KillerX    *float64 `json:"killer_x,omitempty"`
	KillerY    *float64 `json:"killer_y,omitempty"`
	KillerZ    *float64 `json:"killer_z,omitempty"`
	VictimX    *float64 `json:"victim_x,omitempty"`
	VictimY    *float64 `json:"victim_y,omitempty"`
	VictimZ    *float64 `json:"victim_z,omitempty"`
}

// XUIDAliasInsert — row pour shared.xuid_aliases.
type XUIDAliasInsert struct {
	XUID     string    `json:"xuid"`
	Gamertag string    `json:"gamertag"`
	LastSeen time.Time `json:"last_seen"`
}

// CommendationInsert — row pour shared.match_commendations (commendations NATIVES
// Halo 5 progressées sur un match, AXE B prod-gate). Compteur par-match comme
// medals_earned : Count = Progress − PreviousProgress du delta carnage (> 0).
//
// ART-SAFETY : INSERT OR IGNORE sur la clé naturelle non-mutée
// (match_id, xuid, commendation_id) — jamais d'UPDATE sur Count, aucun index
// secondaire sur une colonne mutée (cf. campagne ART #23046). CommendationID est
// l'UUID natif de commendation (clé naturelle, jamais résolu en numérique côté h5).
type CommendationInsert struct {
	MatchID        string `json:"match_id"`
	XUID           string `json:"xuid"`
	CommendationID string `json:"commendation_id"`
	Count          int    `json:"count"`
	// Progress = total À VIE absolu de la commendation à l'instant de ce match
	// (carnage delta Progress). Le total courant = le Progress du match le plus
	// récent par commendation (jamais SUM(count), qui rate la baseline pré-sync).
	Progress int `json:"progress"`
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
// Schema réel prod (cf. sync/pve.go::PveMatchStatsRow) : 19 cols + PK.
// Le `create_pve_match_stats` migration n'en crée que 14 — les 5 autres
// (sentinel_kills, marine_kills, total_kills, deaths, damage_dealt, pve_bits)
// proviennent d'une migration Python ancienne non répliquée côté Go.
// Test-local patch via ALTER TABLE ADD COLUMN IF NOT EXISTS (cf.
// pve_persister_test.go::openPVETestDB).
//
// PveBits : bitmask de complétion par colonne kills. Set à l'INSERT en mode
// Collect→Persist (plus de UPDATE post-coup).
type PVEMatchStatsInsert struct {
	MatchID        string  `json:"match_id"`
	XUID           string  `json:"xuid"`
	WavesCompleted int     `json:"waves_completed"`
	BossKills      int     `json:"boss_kills"`
	GruntKills     int     `json:"grunt_kills"`
	EliteKills     int     `json:"elite_kills"`
	JackalKills    int     `json:"jackal_kills"`
	BruteKills     int     `json:"brute_kills"`
	HunterKills    int     `json:"hunter_kills"`
	SkimmerKills   int     `json:"skimmer_kills"`
	CrawlerKills   int     `json:"crawler_kills"`
	SoldierKills   int     `json:"soldier_kills"`
	KnightKills    int     `json:"knight_kills"`
	WardenKills    int     `json:"warden_kills"`
	SentinelKills  int     `json:"sentinel_kills"`
	MarineKills    int     `json:"marine_kills"`
	TotalKills     int     `json:"total_kills"`
	Deaths         int     `json:"deaths"`
	DamageDealt    float64 `json:"damage_dealt"`
	PveBits        *int    `json:"pve_bits,omitempty"`
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
//
// Schema cf. internal/migration/steps_metadata.go : (mode_en, lang, name)
// avec PK (mode_en, lang). INSERT OR IGNORE pour préserver les traductions
// existantes (l'edition manuelle n'est pas écrasée par les seeds automatiques).
type ModeNameTranslationInsert struct {
	ModeEN string `json:"mode_en"`
	Lang   string `json:"lang"`
	Name   string `json:"name"`
}
