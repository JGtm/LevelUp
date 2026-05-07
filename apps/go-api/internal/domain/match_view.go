// Package domain — types pour la vue détail d'un match.
//
// Port Go de apps/api/app/schemas/match_view.py.
// Route : GET /api/v1/players/{slug}/matches/{match_id}
package domain

import "time"

// MatchViewResponse est la réponse complète de la vue match.
type MatchViewResponse struct {
	Header       MatchViewHeader   `json:"header"`
	Rank         MatchViewRank     `json:"rank"`
	SummaryTab   MatchSummaryTab   `json:"summary_tab"`
	CombatTab    MatchCombatTab    `json:"combat_tab"`
	TeamTab      MatchTeamTab      `json:"team_tab"`
	MediaTab     MatchMediaTab     `json:"media_tab"`
	CitationsTab MatchCitationsTab `json:"citations_tab"`
	// Radar (Phase 1 méta-plan § 6.1.3 — chunk MV4.B). Profil de participation
	// 6 axes (Combat / Survival / Support / Score / Objective / Impact) calculé
	// via narrative.ComputeParticipationProfile à partir des personal_score_awards.
	// Vide si awards non disponibles (capability absente ou repo non câblé).
	// Type `[]any` pour éviter une dépendance domain → service ;
	// les éléments concrets sont des service.MatchViewRadarSeries.
	Radar []any `json:"radar,omitempty"`
	// Sprint 54 B : avertissement de privacy.
	PrivacyWarning *MatchPrivacyWarning `json:"privacy_warning,omitempty"`
}

// MatchViewHeader : en-tête du match.
type MatchViewHeader struct {
	MatchID        string     `json:"match_id"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	StartTimeLabel string     `json:"start_time_label"`
	OutcomeCode    *int       `json:"outcome_code,omitempty"`
	OutcomeLabel   string     `json:"outcome_label"`
	// OutcomeColor : valeur hex legacy. Deprecated (anti-pattern CLAUDE.md
	// règle 20 — aucun hex côté backend). Utiliser OutcomeColorToken pour
	// les nouveaux consommateurs front qui appellent tokenCssVar().
	OutcomeColor string `json:"outcome_color"`
	// OutcomeColorToken : token sémantique (SemanticToken : "outcome-win",
	// "outcome-loss", "outcome-draw", "outcome-dnf"). Le front résout via
	// tokenCssVar(token). Empty si outcome inconnu.
	OutcomeColorToken string `json:"outcome_color_token,omitempty"`
	ScoreLabel        string `json:"score_label,omitempty"`
	// DominanceFlag : true si un badge narratif (domination/humiliation/etc.)
	// s'applique à ce match. Maintenu pour compatibilité ascendante avec les
	// consommateurs front V0 qui n'attendent qu'un booléen.
	// Deprecated: utiliser DominanceBadge pour le rendu i18n + couleur token.
	DominanceFlag bool `json:"dominance_flag"`
	// DominanceBadge : badge narratif typé (LabelKey + ColorToken) résolu via
	// narrative.ResolveDominanceBadge. Nil si aucun badge ne s'applique.
	// Phase 1 méta-plan § 6.1.3 — pilote MatchView aligné sur les fondations.
	DominanceBadge *MatchViewDominanceBadge `json:"dominance_badge,omitempty"`
	HadBotTeammate bool                     `json:"had_bot_teammate"`
	MapUI          string                   `json:"map_ui"`
	MapID          string                   `json:"map_id,omitempty"`
	ModeUI         string                   `json:"mode_ui"`
	PlaylistLabel  string                   `json:"playlist_label"`
	PerfDisplay    string                   `json:"performance_display"`
	// PerfColor : valeur hex legacy. Deprecated (cf. OutcomeColor).
	PerfColor *string `json:"performance_color,omitempty"`
	// PerfColorToken : token sémantique perf-tier-1..5 (1=meilleur, 5=pire).
	// Empty si performance score absent.
	PerfColorToken          string `json:"performance_color_token,omitempty"`
	IsExcluded              bool   `json:"is_excluded"`
	PlayableDurationSeconds *int64 `json:"playable_duration_seconds,omitempty"`
	WaypointURL             string `json:"waypoint_url,omitempty"`
	// MapImageURL : URL de l'image de la map (résolue via TitleAssetURLAdapter
	// au boot du service). Nil si capability non supportée par le titre ou
	// asset manquant. Le front affiche un fallback texte (cf. mock C).
	MapImageURL *string `json:"map_image_url,omitempty"`
	// IsFavorite : true si le joueur a marqué ce match comme favori
	// (table shared_social.match_favorites). False par défaut si shared_social
	// indisponible ou erreur de lecture (dégradation gracieuse).
	IsFavorite bool `json:"is_favorite"`
}

// MatchViewRank : rang CSR ou LUSR pour ce match.
type MatchViewRank struct {
	RatingType string   `json:"rating_type"`
	TierLabel  *string  `json:"tier_label,omitempty"`
	NumericVal *float64 `json:"numeric_value,omitempty"`
	DeltaValue *float64 `json:"delta_value,omitempty"`
	IconURL    string   `json:"icon_url,omitempty"`
	// ProgressPct : position dans le sous-tier courant (0.0–1.0).
	// Nil pour Onyx (pas de tier suivant) ou si rating_value absent.
	ProgressPct *float64 `json:"progress_pct,omitempty"`
}

// MatchViewDominanceBadge : badge narratif typé exposé dans le header.
// Mirror frontend du narrative.DominanceBadge Go (LabelKey + ColorToken).
type MatchViewDominanceBadge struct {
	// Flag : valeur numérique de canonical.DominanceFlag (1..5 pour les 5
	// badges narratifs ; 0 ou inconnu n'est pas exposé — le pointeur est nil).
	Flag int `json:"flag"`
	// LabelKey : clé i18n (ex. "narrative.dominance.domination").
	LabelKey string `json:"label_key"`
	// ColorToken : token sémantique pour résolution couleur côté front
	// (ex. "narrative.dominance.win.strong"). Jamais un hex.
	ColorToken string `json:"color_token"`
}

// ---------------------------------------------------------------------------
// Onglet résumé
// ---------------------------------------------------------------------------

// MatchSummaryKpis : KPIs personnels du résumé.
type MatchSummaryKpis struct {
	Kills           *int     `json:"kills,omitempty"`
	Deaths          *int     `json:"deaths,omitempty"`
	Assists         *int     `json:"assists,omitempty"`
	KDA             *float64 `json:"kda,omitempty"`
	DamageDealt     *float64 `json:"damage_dealt,omitempty"`
	AverageLife     string   `json:"average_life,omitempty"`
	HeadshotKills   *int     `json:"headshot_kills,omitempty"`
	MaxKillingSpree *int     `json:"max_killing_spree,omitempty"`
	PerfectKills    *int     `json:"perfect_kills,omitempty"`
	Accuracy        *float64 `json:"accuracy,omitempty"`
	PersonalScore   *int     `json:"personal_score,omitempty"`
	TeamMMR         *float64 `json:"team_mmr,omitempty"`
	EnemyMMR        *float64 `json:"enemy_mmr,omitempty"`
	DeltaMMR        *float64 `json:"delta_mmr,omitempty"`
}

// MatchPersonalResult : résultat personnel du joueur.
type MatchPersonalResult struct {
	OutcomeLabel string `json:"outcome_label"`
	// OutcomeColor : hex legacy (deprecated, cf. MatchViewHeader.OutcomeColor).
	OutcomeColor string `json:"outcome_color"`
	// OutcomeColorToken : token sémantique (cf. MatchViewHeader.OutcomeColorToken).
	OutcomeColorToken string `json:"outcome_color_token,omitempty"`
	Score             *int   `json:"score,omitempty"`
	RankInTeam        *int   `json:"rank_in_team,omitempty"`
}

// MatchExpectedStats : comparaison réel vs attendu + moyennes historiques.
// Halo Infinite : pas d'expected_assists (l'API skill ne fournit que Kills + Deaths).
type MatchExpectedStats struct {
	HasExpectedData bool     `json:"has_expected_data"`
	ExpectedKills   *float64 `json:"expected_kills,omitempty"`
	ExpectedDeaths  *float64 `json:"expected_deaths,omitempty"`
	// Moyennes historiques sur le mode (HistAvg)
	HasHistAvg           bool     `json:"has_hist_avg"`
	HistAvgKills         *float64 `json:"hist_avg_kills,omitempty"`
	HistAvgDeaths        *float64 `json:"hist_avg_deaths,omitempty"`
	HistAvgAssists       *float64 `json:"hist_avg_assists,omitempty"`
	HistAvgSpree         *float64 `json:"hist_avg_spree,omitempty"`
	HistAvgHeadshotKills *float64 `json:"hist_avg_headshot_kills,omitempty"`
	HistAvgPerfectKills  *float64 `json:"hist_avg_perfect_kills,omitempty"`
	HistMatchCount       int      `json:"hist_match_count,omitempty"`
	HistModeCategory     string   `json:"hist_mode_category,omitempty"`
}

// MatchHistAvgRow : ligne brute de Q29 (historique récent pour moyennes).
type MatchHistAvgRow struct {
	Kills           int
	Deaths          int
	Assists         int
	HeadshotKills   *int
	MaxKillingSpree *int
	PerfectKills    int
	PairName        string
	IsFirefight     bool
	IsRanked        bool
}

// MatchMedal : une médaille gagnée dans le match.
type MatchMedal struct {
	MedalNameID int64   `json:"medal_name_id"`
	Name        string  `json:"name"`
	Count       int     `json:"count"`
	Description *string `json:"description,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	Difficulty  string  `json:"difficulty,omitempty"`
}

// MatchCitation : badge de citation associé au match.
type MatchCitation struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Color *string  `json:"color,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// MatchSummaryTab : contenu de l'onglet Résumé.
type MatchSummaryTab struct {
	KPIs           MatchSummaryKpis       `json:"kpis"`
	PersonalResult MatchPersonalResult    `json:"personal_result"`
	Medals         []MatchMedal           `json:"medals"`
	Citations      []MatchCitationSnippet `json:"citations"`
	ExpectedStats  MatchExpectedStats     `json:"expected_stats"`
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
	EventTimeMS *int64  `json:"event_time_ms,omitempty"`
	ActorXUID   *string `json:"actor_xuid,omitempty"`
}

// MatchTugOfWarBin : tranche temporelle de la timeline tug-of-war.
type MatchTugOfWarBin struct {
	BinStart   int `json:"bin_start"`
	BinEnd     int `json:"bin_end"`
	TeamKills  int `json:"team_kills"`
	EnemyKills int `json:"enemy_kills"`
	NetKills   int `json:"net_kills"`
}

// MatchImpactBadge : badge d'impact calculé (premier sang, finisseur…).
//
// Deprecated: utiliser MatchViewImpactRole pour les nouveaux consommateurs
// (8 rôles narratifs typés via narrative.IdentifyImpactRoles, Phase 1
// méta-plan § 6.1.3). Conservé pour rétrocompat avec analysis.ComputeMatchImpactFull
// (4 rôles bilatéral legacy).
type MatchImpactBadge struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	PlayerXUID string `json:"player_xuid,omitempty"`
	Value      string `json:"value,omitempty"`
	// TimeMS : instant (ms depuis le début du match) où le badge a été
	// déclenché, pour les badges event-based uniquement. Nil pour les badges
	// stat-based (top_killer, silent_hero, false_brother) qui n'ont pas de
	// notion de temps.
	TimeMS *int64 `json:"time_ms,omitempty"`
}

// MatchViewImpactRole : rôle narratif attribué via narrative.IdentifyImpactRoles
// (8 rôles : first_blood, clutch_finisher, last_casualty, last_group_kill,
// first_group_death, silent_hero, false_brother, top_killer).
//
// Mirror frontend du narrative.RoleAssignment Go. Rendu via le wrapper
// `<NarrativeBadge>` (Phase 0 méta-plan § 5.4) avec couleur résolue côté
// front via tokenCssVar(ColorToken).
type MatchViewImpactRole struct {
	XUID       string `json:"xuid"`
	RoleKey    string `json:"role_key"`
	LabelKey   string `json:"label_key"`
	ColorToken string `json:"color_token"`
	Inverted   bool   `json:"inverted,omitempty"`
}

// MatchKDTimelinePoint : point K/D sur la timeline du match.
type MatchKDTimelinePoint struct {
	TimeSeconds int `json:"time_seconds"`
	Kills       int `json:"kills"`
	Deaths      int `json:"deaths"`
}

// MatchKillerVictimPair : agrégat killer→victim pour un match (chart
// match_view.18 — "Graphe des antagonistes"). Une entrée par paire unique
// (killer_xuid, victim_xuid) ; KillCount = somme des kills sur ce match.
type MatchKillerVictimPair struct {
	KillerXUID     string `json:"killer_xuid"`
	KillerGamertag string `json:"killer_gamertag"`
	VictimXUID     string `json:"victim_xuid"`
	VictimGamertag string `json:"victim_gamertag"`
	KillCount      int    `json:"kill_count"`
}

// MatchCombatTab : contenu de l'onglet Combat.
type MatchCombatTab struct {
	WeaponKills     []MatchWeaponKill      `json:"weapon_kills"`
	HighlightEvents []MatchHighlightEvent  `json:"highlight_events"`
	TugOfWar        []MatchTugOfWarBin     `json:"tug_of_war"`
	ImpactBadges    []MatchImpactBadge     `json:"impact_badges"`
	KDTimeline      []MatchKDTimelinePoint `json:"kd_timeline"`
	NemesisDuels    []MatchNemesisRow      `json:"nemesis_duels"`

	// KillerVictim : paires killer→victim agrégées pour le chart antagoniste
	// (match_view.18). Vide si killer_victim_pairs n'est pas peuplé.
	KillerVictim []MatchKillerVictimPair `json:"killer_victim,omitempty"`

	// ImpactRoles (Phase 1 méta-plan § 6.1.3 — pilote MatchView aligné
	// fondations narrative). 8 rôles narratifs typés via
	// narrative.IdentifyImpactRoles, en parallèle des 4 ImpactBadges
	// legacy. Vide si events ou scoreboard absents.
	ImpactRoles []MatchViewImpactRole `json:"impact_roles,omitempty"`

	// Cadence (Phase 1 méta-plan § 6.1.3). Cadence intra-match : kills
	// par phase de 60s, 1 série par xuid du scoreboard.
	// Format ChartPointStacked → wrapper `<BarStacked>` côté front.
	// Nil si events absents.
	Cadence *ChartSeries[ChartPointStacked] `json:"cadence,omitempty"`
}

// ---------------------------------------------------------------------------
// Onglet équipe
// ---------------------------------------------------------------------------

// PlayerMedalRow : médaille d'un joueur dans un match (expander scoreboard).
type PlayerMedalRow struct {
	MedalID    int64  `json:"medal_id"`
	Count      int    `json:"count"`
	Label      string `json:"label,omitempty"`
	ImageURL   string `json:"image_url,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
}

// PlayerWeaponKillRow : kills par arme d'un joueur dans un match (expander scoreboard).
type PlayerWeaponKillRow struct {
	WeaponID int64  `json:"weapon_id"`
	Kills    int    `json:"kills"`
	Label    string `json:"label,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// MatchScoreboardSkillRank : skill rank d'un joueur pour ce match (extrait
// allégé de SkillRankRaw, pour le panneau d'expander). Source : match_skill_rank
// table de la player DB du joueur — donc disponible uniquement pour les joueurs
// trackés (main + amis avec player DB).
type MatchScoreboardSkillRank struct {
	RatingType  string   `json:"rating_type"`            // "CSR" ou "LUSR"
	TierLabel   *string  `json:"tier_label,omitempty"`   // ex: "Onyx 1500"
	RatingValue *float64 `json:"rating_value,omitempty"` // valeur numérique
	RatingDelta *float64 `json:"rating_delta,omitempty"` // ±delta vs match précédent
}

// MatchScoreboardRow : ligne du scoreboard.
type MatchScoreboardRow struct {
	XUID     string  `json:"xuid"`
	Gamertag string  `json:"gamertag"`
	TeamSide *string `json:"team_side,omitempty"`
	IsMe     bool    `json:"is_me"`
	IsBot    bool    `json:"is_bot,omitempty"`
	IsMVP    bool    `json:"is_mvp,omitempty"`
	IsLVP    bool    `json:"is_lvp,omitempty"`
	// PerformanceScore : score de performance (0..100) calculé sur l'historique
	// du joueur. Disponible uniquement pour les joueurs trackés (main + amis).
	// Source : player_match_enrichment.performance_score de la player DB.
	PerformanceScore *float64 `json:"performance_score,omitempty"`
	// HadBotTeammate : true si au moins un bot dans l'équipe du joueur (du
	// point de vue de SA player DB). Disponible uniquement pour les joueurs
	// trackés. Source : player_match_enrichment.had_bot_teammate.
	HadBotTeammate bool `json:"had_bot_teammate,omitempty"`
	// SkillRank : rang compétitif (CSR/LUSR) pour ce match. Disponible
	// uniquement pour les joueurs trackés. Source : match_skill_rank.
	SkillRank        *MatchScoreboardSkillRank `json:"skill_rank,omitempty"`
	Rank             *int                      `json:"rank,omitempty"`
	Score            *int                      `json:"score,omitempty"`
	Kills            *int                      `json:"kills,omitempty"`
	Deaths           *int                      `json:"deaths,omitempty"`
	Assists          *int                      `json:"assists,omitempty"`
	KDA              *float64                  `json:"kda,omitempty"`
	Accuracy         *float64                  `json:"accuracy,omitempty"`
	DamageDealt      *float64                  `json:"damage_dealt,omitempty"`
	DamageTaken      *float64                  `json:"damage_taken,omitempty"`
	ShotsFired       *int                      `json:"shots_fired,omitempty"`
	ShotsHit         *int                      `json:"shots_hit,omitempty"`
	AvgLifeSeconds   *float64                  `json:"avg_life_seconds,omitempty"`
	HeadshotKills    *int                      `json:"headshot_kills,omitempty"`
	MaxKillingSpree  *int                      `json:"max_killing_spree,omitempty"`
	PerfectKills     *int                      `json:"perfect_kills,omitempty"`
	GrenadeKills     *int                      `json:"grenade_kills,omitempty"`
	MeleeKills       *int                      `json:"melee_kills,omitempty"`
	PowerWeaponKills *int                      `json:"power_weapon_kills,omitempty"`
	OutcomeLabel     string                    `json:"outcome_label"`
	// Combat yield (V7)
	TopWeaponID         *int64   `json:"top_weapon_id,omitempty"`
	TopWeaponLabel      string   `json:"top_weapon_label,omitempty"`
	OffensiveConversion *float64 `json:"offensive_conversion,omitempty"`
	DefensiveResistance *float64 `json:"defensive_resistance,omitempty"`
	DamagePerKill       *float64 `json:"damage_per_kill,omitempty"`
	DamagePerDeath      *float64 `json:"damage_per_death,omitempty"`
	// Expected vs actual (depuis match_participants).
	// Halo Infinite : pas d'expected_assists (l'API skill ne renvoie que Kills + Deaths).
	ExpectedKills  *float64 `json:"expected_kills,omitempty"`
	ExpectedDeaths *float64 `json:"expected_deaths,omitempty"`
	KillsStdDev    *float64 `json:"kills_stddev,omitempty"`
	DeathsStdDev   *float64 `json:"deaths_stddev,omitempty"`
	// Expander : données per-player chargées en bulk
	Medals      []PlayerMedalRow      `json:"medals,omitempty"`
	WeaponKills []PlayerWeaponKillRow `json:"weapon_kills,omitempty"`
}

// MatchNemesisRow : adversaire fréquent (kills reçus de lui).
type MatchNemesisRow struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	KilledMe int    `json:"killed_me"`
	IKilled  int    `json:"i_killed"`
}

// MatchRosterRow : ligne condensée du roster (tous les participants).
type MatchRosterRow struct {
	XUID        string   `json:"xuid"`
	Gamertag    string   `json:"gamertag"`
	TeamSide    *string  `json:"team_side,omitempty"`
	IsMe        bool     `json:"is_me"`
	IsBot       bool     `json:"is_bot"`
	Kills       *int     `json:"kills,omitempty"`
	Deaths      *int     `json:"deaths,omitempty"`
	Assists     *int     `json:"assists,omitempty"`
	KDA         *float64 `json:"kda,omitempty"`
	DamageDealt *float64 `json:"damage_dealt,omitempty"`
	DamageTaken *float64 `json:"damage_taken,omitempty"`
}

// MatchEncounterRow : historique de rencontres avec un participant du match.
type MatchEncounterRow struct {
	XUID          string `json:"xuid"`
	Gamertag      string `json:"gamertag"`
	CountTogether int    `json:"count_together"`
	IsAlly        bool   `json:"is_ally"`
	IsBot         bool   `json:"is_bot,omitempty"`
	// Découpage du CountTogether en allié vs ennemi sur l'historique commun
	// (mock match_view.14 — affiche "(A:X | E:Y)" sous Rencontres).
	AllyCount  *int `json:"ally_count,omitempty"`
	EnemyCount *int `json:"enemy_count,omitempty"`
	// Winrates calculés sur l'historique commun. nil quand non calculable
	// (W+L == 0 sur le bucket ally ou enemy).
	WinrateAsAlly  *float64 `json:"winrate_as_ally,omitempty"`
	WinrateVsEnemy *float64 `json:"winrate_vs_enemy,omitempty"`
	// K/D croisé : kills_dealt = kills par moi sur ce joueur (toutes occurrences),
	// deaths_suffered = morts subies par moi causées par ce joueur. nil quand
	// killer_victim_pairs est absent du repo.
	KillsDealt     *int `json:"kills_dealt,omitempty"`
	DeathsSuffered *int `json:"deaths_suffered,omitempty"`
	// Date du dernier match commun (toutes occurrences allié + ennemi).
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	// Badges narratifs typés (chunk MV4.C / MV4.C').
	Badges []MatchEncounterBadge `json:"badges,omitempty"`
}

// MatchEncounterBadge : badge narratif typé pour un encounter.
//
// Mirror frontend du narrative.EncounterBadge Go (LabelKey + ColorToken).
// Le détail (winrate, kd_against_me, ordinal value) est exposé dans Detail
// pour permettre au front d'afficher des tooltips contextuels.
type MatchEncounterBadge struct {
	Kind       string         `json:"kind"`        // "ordinal" / "ally_plus" / "tough_enemy"
	LabelKey   string         `json:"label_key"`   // clé i18n (ex. "narrative.encounter.ordinal")
	ColorToken string         `json:"color_token"` // token sémantique
	Detail     map[string]any `json:"detail,omitempty"`
}

// MatchTeamTab : contenu de l'onglet Équipe.
type MatchTeamTab struct {
	Roster     []MatchRosterRow     `json:"roster"`
	Scoreboard []MatchScoreboardRow `json:"scoreboard"`
	Nemesis    []MatchNemesisRow    `json:"nemesis"`
	Encounters []MatchEncounterRow  `json:"encounters"`
}

// ---------------------------------------------------------------------------
// Onglets médias et citations
// ---------------------------------------------------------------------------

// MatchAssociatedMedia : média associé au match (typé).
type MatchAssociatedMedia struct {
	FileID          string  `json:"file_id"`
	FileName        string  `json:"file_name"`
	FilePath        string  `json:"file_path"`
	ThumbnailURL    *string `json:"thumbnail_url,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	CaptureTime     *string `json:"capture_time,omitempty"`
	Liked           bool    `json:"liked"`
}

// MatchMediaTab : contenu de l'onglet Médias.
type MatchMediaTab struct {
	MediaItems []MatchAssociatedMedia `json:"media_items"`
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
	MatchID                 string
	StartTime               *time.Time
	DurationSeconds         *float64
	MapName                 *string
	PairName                *string
	PlaylistName            *string
	IsFirefight             bool
	IsRanked                bool
	PlayableDurationSeconds *int64
	MapAssetID              *string
	GameVariantName         *string
	// PlaylistAssetID : identifiant stable de la playlist (match_registry.playlist_id).
	// Nil pour les matchs custom games sans playlist enregistrée.
	PlaylistAssetID *string
	// MapNameFR / ModeNameFR / PlaylistNameFR : traductions FR enrichies post-scan
	// via asset_translations (map, playlist) et mode_name_tr (mode). Nil si non
	// disponibles — le service retombe alors sur le label brut EN.
	MapNameFR      *string
	ModeNameFR     *string
	PlaylistNameFR *string
	// PairNameFR : traduction FR stockée en DB (match_registry.pair_name_fr).
	// Utilisé comme fallback quand mode_name_tr ne contient pas le mode EN normalisé.
	PairNameFR *string
	// MapImageURL : URL résolue depuis map_images_registry par map_id (stable UUID).
	// Nil si map_id absent du registry — le service retombe sur l'adapter name-based.
	MapImageURL *string
	// Team0Score / Team1Score : score de jeu de chaque équipe (match_registry).
	// Ex: 50/47 pour Slayer, 3/1 pour CTF. Nil si FFA ou custom sans score.
	Team0Score *int16
	Team1Score *int16
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
	TeamMMR           *float64
	EnemyMMR          *float64
	HeadshotKills     *int
	MaxKillingSpree   *int
}

// ScoreboardRaw : données brutes de Q12 (une ligne du scoreboard).
type ScoreboardRaw struct {
	XUID             string
	Gamertag         string
	IsBot            bool
	TeamID           *int
	RankInTeam       *int
	OutcomeCode      int
	PersonalScore    *float64
	Kills            int
	Deaths           int
	Assists          int
	KDA              *float64
	Accuracy         *float64
	TimePlayed       *float64
	TeamMMR          *float64
	EnemyMMR         *float64
	ShotsFired       *int
	ShotsHit         *int
	DamageDealt      *float64
	DamageTaken      *float64
	AvgLifeSeconds   *float64
	HeadshotKills    *int
	MaxKillingSpree  *int
	GrenadeKills     *int
	MeleeKills       *int
	PowerWeaponKills *int
	PerfectKills     int
	TopWeaponID      *int64
	TopWeaponLabel   string
	// Expected stats (kills_expected, deaths_expected, etc. depuis match_participants).
	// Halo Infinite : pas d'assists_expected/stddev (l'API ne les renvoie pas).
	KillsExpected  *float64
	DeathsExpected *float64
	KillsStdDev    *float64
	DeathsStdDev   *float64
}

// BulkMedalRaw : une ligne de Q27 (médailles de tous les joueurs du match).
type BulkMedalRaw struct {
	XUID       string
	MedalID    int64
	Count      int
	Label      string
	Difficulty string
}

// BulkWeaponKillRaw : une ligne de Q28 (kills par arme de tous les joueurs du match).
type BulkWeaponKillRaw struct {
	XUID        string
	WeaponID    int64
	Kills       int
	WeaponLabel string
	NameEN      string // nom EN officiel → lookup WeaponImageURL dans l'adapter
}

// MatchEnrichmentRaw : données brutes de Q18.
type MatchEnrichmentRaw struct {
	PerformanceScore *float64
	IsWithFriends    bool
	IsExcluded       bool
	// DominanceFlag : 0=none, 1=domination, 2=humiliation, 3=remontada,
	// 4=débandade, 5=contre-remontada (cf. canonical.DominanceFlag).
	// Peuplé par sync.BackfillDominanceFlags via engine.RunBackfillComebackBadges.
	DominanceFlag int
}

// MedalRaw : données brutes de Q14.
type MedalRaw struct {
	MedalID     int64
	Count       int
	Label       string
	Description string
	Difficulty  string
}

// EventRaw : données brutes de Q21.
type EventRaw struct {
	EventType string
	TimeMS    *int64
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

// MatchNeighbors : matchs adjacents (prev/next) pour la navigation de la page détail.
type MatchNeighbors struct {
	PreviousMatchID *string `json:"previous_match_id"`
	NextMatchID     *string `json:"next_match_id"`
	CurrentIndex    int     `json:"current_index"`
	TotalMatches    int     `json:"total_matches"`
	// AppliedFilters : echo des filtres effectivement appliqués (Phase 2b).
	// Permet au front de reconstruire un contextLabel localisé sans dupliquer
	// la résolution côté backend. Nil si aucun filtre n'a été passé.
	AppliedFilters *MatchFilterSpec `json:"applied_filters,omitempty"`
}

// SkillRankRaw : données brutes de Q22 (match_skill_rank — player DB).
type SkillRankRaw struct {
	RatingType    string
	TierLabel     *string
	RatingValue   *float64
	RatingDelta   *float64
	PlaylistGroup *string
	// Tier (EN, ex: "Diamond", "Onyx") et SubTier (1..6 ou 0 pour Onyx) sont
	// extraits depuis match_skill_rank. Utilisés par buildRankBlock pour
	// construire l'URL du badge via TitleAssetURLAdapter.CSRRankImageURL.
	// Nil quand absent (LUSR ou rang non encore résolu côté sync).
	Tier    *string
	SubTier *int
}

// EncounterRaw : données brutes de Q23 (participants du match + historique commun).
type EncounterRaw struct {
	XUID          string
	Gamertag      string
	IsBot         bool
	CountTogether int
	IsAlly        bool
}

// EncounterStatsRaw : stats riches par encounter chargees via Q23b
// (chunk MV4.C'). Permet a narrative.ComputeEncounterBadges d'attribuer
// les badges ally_plus + tough_enemy.
//
// L'agregation se fait sur l'historique commun entre le joueur courant
// (myXUID) et tm.xuid. Le service compose ces stats avec les EncounterRaw
// (CountTogether + IsAlly) pour produire les EncounterStats consommees par
// narrative.ComputeEncounterBadges.
type EncounterStatsRaw struct {
	XUID           string
	AllyCount      int
	EnemyCount     int
	WinsAsAlly     int
	LossesAsAlly   int
	WinsVsEnemy    int
	LossesVsEnemy  int
	KillsDealt     int
	DeathsSuffered int
	// LastSeenAt : start_time UTC du match commun le plus récent. Zéro time
	// quand l'historique est vide ou que match_registry n'a pas de start_time
	// pour les matchs concernés.
	LastSeenAt time.Time
}

// MediaAssocRaw : données brutes de Q24 (media_files + media_match_associations).
type MediaAssocRaw struct {
	FileID        string
	FileName      string
	FilePath      string
	ThumbnailPath *string
	CaptureTime   *string
	Liked         bool
}

// ExpectedStatsRaw : données brutes de Q26 (match_participants expected columns).
// Halo Infinite : pas d'assists (l'API ne renvoie que Kills + Deaths).
type ExpectedStatsRaw struct {
	KillsExpected  *float64
	DeathsExpected *float64
	KillsStddev    *float64
	DeathsStddev   *float64
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
