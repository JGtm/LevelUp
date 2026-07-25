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
	// IsPartial / PartialReasons (RC6 — 2026-05-08) : true quand le match a son
	// match_registry peuplé mais qu'au moins une source secondaire critique est
	// vide (scoreboard, events, player stats). Le front peut afficher un bandeau
	// "Sync incomplet — certaines sections sont indisponibles" au lieu d'un
	// crash full-page. Strict 404 reste pour les match_id totalement absents.
	IsPartial      bool     `json:"is_partial,omitempty"`
	PartialReasons []string `json:"partial_reasons,omitempty"`
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
	PerfColorToken string `json:"performance_color_token,omitempty"`
	IsExcluded     bool   `json:"is_excluded"`
	// IsRanked : true si la playlist est classée (CSR officiel). Utilisé côté
	// front pour désactiver le bouton "Exclure" (un match classé ne peut pas
	// être exclu). Source : shared.match_registry.is_ranked.
	IsRanked                bool   `json:"is_ranked"`
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
// ExpectedAssists est calculé à la volée depuis assists_model_coefs (slope×(personal_score+shots_hit)+intercept).
type MatchExpectedStats struct {
	ExpectedKills   *float64 `json:"expected_kills,omitempty"`
	ExpectedDeaths  *float64 `json:"expected_deaths,omitempty"`
	ExpectedAssists *float64 `json:"expected_assists,omitempty"`
	// ExpectedWinProb : proba de victoire pré-match de l'équipe du joueur (LUSR v2,
	// ∈ [0,1]). Alimente la card « Résultat attendu ». Nil si pré-v2 / non disponible.
	ExpectedWinProb *float64 `json:"expected_win_prob,omitempty"`
	// LocallyEstimated : true quand expected_kills/deaths ne viennent PAS de l'API
	// skill (absente, ex. Halo 5) mais d'un modèle local count∝durée (TrueSkill2-like)
	// → le front affiche « Estimé localement ».
	LocallyEstimated bool `json:"locally_estimated,omitempty"`
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

// MatchMedal : une médaille gagnée dans le match.
//
// Icône title-aware : ImageURL (PNG, HINF) OU les champs Sprite* (feuille + offset,
// Halo 5). Mutuellement exclusifs ; mêmes tags JSON que l'Asset Drawer (AssetMeta).
type MatchMedal struct {
	MedalNameID  int64   `json:"medal_name_id"`
	Name         string  `json:"name"`
	Count        int     `json:"count"`
	Description  *string `json:"description,omitempty"`
	ImageURL     string  `json:"image_url,omitempty"`
	Difficulty   string  `json:"difficulty,omitempty"`
	SpriteSheet  string  `json:"sprite_sheet,omitempty"`
	SpriteLeft   int     `json:"sprite_left,omitempty"`
	SpriteTop    int     `json:"sprite_top,omitempty"`
	SpriteWidth  int     `json:"sprite_width,omitempty"`
	SpriteHeight int     `json:"sprite_height,omitempty"`
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
	// Class : axe manipulation de l'arme (shoulder/sidearm/heavy/…), résolu via le
	// registre (BulkWeaponKillRaw). Vide si l'arme est absente du registre. Recolore
	// le breakdown par arme par classe (FragWeaponBreakdown, sunburst v2).
	Class string `json:"class,omitempty"`
}

// MatchHighlightEvent : événement filmé horodaté.
//
// ActorGamertag est le nom à afficher (résolu via v_gamertag_lookup côté repo :
// gère bots `bid(N.0)` → "343 Bot N" et fallback xuid raw). Le front l'affiche
// directement, sans logique de résolution. ActorXUID reste exposé pour les
// callers qui ont besoin de l'ID stable (deep-linking, etc.).
type MatchHighlightEvent struct {
	EventType     string  `json:"event_type"`
	EventTimeMS   *int64  `json:"event_time_ms,omitempty"`
	ActorXUID     *string `json:"actor_xuid,omitempty"`
	ActorGamertag *string `json:"actor_gamertag,omitempty"`
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

	// FragDistribution : répartition hiérarchique des frags v2 (sunburst classe→rôle)
	// du VIEWER (is_me) pour ce match. Classes gun = bulk weapon kills du viewer
	// (registre) ; melee/grenade/spartan + total = compteurs natifs de sa ligne
	// scoreboard. Nil si le viewer n'a aucun kill (le front rend null). Cf.
	// .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P3.
	FragDistribution *FragDistribution `json:"frag_distribution,omitempty"`
}

// ---------------------------------------------------------------------------
// Onglet équipe
// ---------------------------------------------------------------------------

// PlayerMedalRow : médaille d'un joueur dans un match (expander scoreboard).
//
// Icône title-aware (mêmes tags que MatchMedal) : ImageURL (PNG, HINF) OU les champs
// Sprite* (feuille + offset, Halo 5). Mutuellement exclusifs. Sans les champs sprite,
// une médaille Halo 5 (pas de PNG par-médaille) s'affichait vide dans le drawer (GH-5a).
type PlayerMedalRow struct {
	MedalID      int64  `json:"medal_id"`
	Count        int    `json:"count"`
	Label        string `json:"label,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	Difficulty   string `json:"difficulty,omitempty"`
	SpriteSheet  string `json:"sprite_sheet,omitempty"`
	SpriteLeft   int    `json:"sprite_left,omitempty"`
	SpriteTop    int    `json:"sprite_top,omitempty"`
	SpriteWidth  int    `json:"sprite_width,omitempty"`
	SpriteHeight int    `json:"sprite_height,omitempty"`
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
	IconURL     *string  `json:"icon_url,omitempty"`     // URL du badge image (CSR tier)
}

// MatchScoreboardRow : ligne du scoreboard.
type MatchScoreboardRow struct {
	XUID     string  `json:"xuid"`
	Gamertag string  `json:"gamertag"`
	TeamSide *string `json:"team_side,omitempty"`
	// TeamName : libellé d'équipe localisé fourni par le backend (Halo 5 :
	// « Rouge »/« Red » depuis le référentiel team_colors). Vide pour les titres
	// sans référentiel d'équipes (Halo Infinite) → le front retombe sur sa
	// résolution de nom d'équipe existante (Eagle/Cobra).
	TeamName string `json:"team_name,omitempty"`
	// TeamColor : couleur d'identité hex (#RRGGBB) de l'équipe fournie par le backend
	// (Halo 5 : team_colors.color). Vide pour les titres sans référentiel de couleurs
	// (Halo Infinite) → le front retombe sur sa map de couleurs par team_id.
	TeamColor string `json:"team_color,omitempty"`
	IsMe      bool   `json:"is_me"`
	IsBot     bool   `json:"is_bot,omitempty"`
	IsMVP     bool   `json:"is_mvp,omitempty"`
	IsLVP     bool   `json:"is_lvp,omitempty"`
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
	// Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate :
	// ground pound, shoulder bash). nil hors h5 (omitempty) ; affichage gated
	// front via la capability native_kill_mechanics.
	AssassinationKills *int   `json:"assassination_kills,omitempty"`
	GroundPoundKills   *int   `json:"ground_pound_kills,omitempty"`
	ShoulderBashKills  *int   `json:"shoulder_bash_kills,omitempty"`
	OutcomeLabel       string `json:"outcome_label"`
	// Combat yield (V7)
	TopWeaponID         *int64   `json:"top_weapon_id,omitempty"`
	TopWeaponLabel      string   `json:"top_weapon_label,omitempty"`
	OffensiveConversion *float64 `json:"offensive_conversion,omitempty"`
	DefensiveResistance *float64 `json:"defensive_resistance,omitempty"`
	DamagePerKill       *float64 `json:"damage_per_kill,omitempty"`
	DamagePerDeath      *float64 `json:"damage_per_death,omitempty"`
	// Expected vs actual : kills/deaths depuis l'API, assists calculé à la volée.
	ExpectedKills   *float64 `json:"expected_kills,omitempty"`
	ExpectedDeaths  *float64 `json:"expected_deaths,omitempty"`
	ExpectedAssists *float64 `json:"expected_assists,omitempty"`
	// LocallyEstimated : expected K/D issus du modèle local (Halo 5), pas de l'API
	// → le drawer affiche le label « Estimé localement ».
	LocallyEstimated bool     `json:"locally_estimated,omitempty"`
	KillsStdDev      *float64 `json:"kills_stddev,omitempty"`
	DeathsStdDev     *float64 `json:"deaths_stddev,omitempty"`
	// Expander : données per-player chargées en bulk
	Medals      []PlayerMedalRow      `json:"medals,omitempty"`
	WeaponKills []PlayerWeaponKillRow `json:"weapon_kills,omitempty"`
	// Objective : stats objectifs du joueur (CTF/Zones/Oddball). nil hors mode à
	// objectif (Slayer) ou titre non supporté (capability objective_stats absente →
	// table vide). Seuls les champs du bloc du mode joué sont non-nil. Affichage gated
	// front (useCapability('objective_stats')) + data-driven par bloc présent.
	Objective *MatchScoreboardObjective `json:"objective,omitempty"`
}

// MatchScoreboardObjective : stats objectifs par joueur d'un match à objectif.
// Blocs mutuellement exclusifs par mode (CTF / Zones (Strongholds+KOTH) / Oddball /
// Stockpile / Extraction / VIP) : seuls les champs du mode joué sont renseignés (les autres
// nil, omitempty). Totaux équipe/lobby calculés à la LECTURE côté front (SUM par équipe).
// Colonnes verrouillées sur payload réel GetMatchStats (PLAN_V72_OBJECTIVE_STATS.md ;
// Stockpile + Extraction + VIP : V721-02, PLAN_V721_NOTION_BATCH.md).
type MatchScoreboardObjective struct {
	// CTF (CaptureTheFlagStats)
	FlagCaptures             *int     `json:"flag_captures,omitempty"`
	FlagCaptureAssists       *int     `json:"flag_capture_assists,omitempty"`
	FlagGrabs                *int     `json:"flag_grabs,omitempty"`
	FlagSecures              *int     `json:"flag_secures,omitempty"`
	FlagSteals               *int     `json:"flag_steals,omitempty"`
	FlagReturns              *int     `json:"flag_returns,omitempty"`
	FlagCarriersKilled       *int     `json:"flag_carriers_killed,omitempty"`
	FlagReturnersKilled      *int     `json:"flag_returners_killed,omitempty"`
	KillsAsFlagCarrier       *int     `json:"kills_as_flag_carrier,omitempty"`
	KillsAsFlagReturner      *int     `json:"kills_as_flag_returner,omitempty"`
	TimeAsFlagCarrierSeconds *float64 `json:"time_as_flag_carrier_seconds,omitempty"`
	// Zones (ZonesStats — Strongholds + KOTH ; zone_scoring_ticks>0 distingue KOTH)
	ZoneCaptures       *int     `json:"zone_captures,omitempty"`
	ZoneSecures        *int     `json:"zone_secures,omitempty"`
	ZoneOffensiveKills *int     `json:"zone_offensive_kills,omitempty"`
	ZoneDefensiveKills *int     `json:"zone_defensive_kills,omitempty"`
	ZoneScoringTicks   *int     `json:"zone_scoring_ticks,omitempty"`
	TimeInZonesSeconds *float64 `json:"time_in_zones_seconds,omitempty"`
	// Oddball (OddballStats)
	KillsAsSkullCarrier              *int     `json:"kills_as_skull_carrier,omitempty"`
	SkullCarriersKilled              *int     `json:"skull_carriers_killed,omitempty"`
	SkullGrabs                       *int     `json:"skull_grabs,omitempty"`
	SkullScoringTicks                *int     `json:"skull_scoring_ticks,omitempty"`
	TimeAsSkullCarrierSeconds        *float64 `json:"time_as_skull_carrier_seconds,omitempty"`
	LongestTimeAsSkullCarrierSeconds *float64 `json:"longest_time_as_skull_carrier_seconds,omitempty"`
	// Stockpile (StockpileStats — graines d'énergie)
	KillsAsPowerSeedCarrier       *int     `json:"kills_as_power_seed_carrier,omitempty"`
	PowerSeedCarriersKilled       *int     `json:"power_seed_carriers_killed,omitempty"`
	PowerSeedsDeposited           *int     `json:"power_seeds_deposited,omitempty"`
	PowerSeedsStolen              *int     `json:"power_seeds_stolen,omitempty"`
	TimeAsPowerSeedCarrierSeconds *float64 `json:"time_as_power_seed_carrier_seconds,omitempty"`
	TimeAsPowerSeedDriverSeconds  *float64 `json:"time_as_power_seed_driver_seconds,omitempty"`
	// Extraction (ExtractionStats — aucune durée dans ce bloc)
	ExtractionConversionsCompleted *int `json:"extraction_conversions_completed,omitempty"`
	ExtractionConversionsDenied    *int `json:"extraction_conversions_denied,omitempty"`
	ExtractionInitiationsCompleted *int `json:"extraction_initiations_completed,omitempty"`
	ExtractionInitiationsDenied    *int `json:"extraction_initiations_denied,omitempty"`
	SuccessfulExtractions          *int `json:"successful_extractions,omitempty"`
	// VIP (VipStats)
	KillsAsVip              *int     `json:"kills_as_vip,omitempty"`
	VipKills                *int     `json:"vip_kills,omitempty"`
	VipAssists              *int     `json:"vip_assists,omitempty"`
	TimesSelectedAsVip      *int     `json:"times_selected_as_vip,omitempty"`
	MaxKillingSpreeAsVip    *int     `json:"max_killing_spree_as_vip,omitempty"`
	TimeAsVipSeconds        *float64 `json:"time_as_vip_seconds,omitempty"`
	LongestTimeAsVipSeconds *float64 `json:"longest_time_as_vip_seconds,omitempty"`
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
	Kind            string  `json:"kind"`
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
	// NativeCommendations : commendations NATIVES (Halo 5) progressées CE match,
	// affichées TELLES QUELLES (PAS le moteur de citations dérivé d'Infinite —
	// AXE B prod-gate). Vide/omis pour les titres sans commendations natives.
	NativeCommendations []MatchNativeCommendation `json:"native_commendations,omitempty"`
	Medals              []MatchMedal              `json:"medals"`
}

// MatchNativeCommendation : commendation NATIVE (Halo 5) gagnée sur un match.
// Donnée brute : Name vide → le front dégrade en « Commendation {ID} » ; IconURL
// nil → pas d'icône (Phase 1, définitions natives = suite AXE B).
//
// PARITÉ Infinite (anneau de progression + masterisé doré) : les champs de tier
// réutilisent les MÊMES noms JSON que MatchCitationSnippet (progress_pct,
// tier_index, tier_count, next_tier_target, cumulative, is_newly_mastered) afin
// que le front réutilise le composant CitationProgressRing SANS divergence de
// contrat. Calculés read-time depuis Progress (cumul à vie) + tier_targets +
// Count (delta du match) via analysis.ComputeTierProgression.
//
// Les commendations MASTERISÉES AVANT le match (palier final déjà franchi) ne sont
// JAMAIS émises (filtrage backend = parité « masquage des pré-masterisées »
// Infinite). Le front n'a donc qu'à distinguer : anneau de progression (cas normal)
// vs anneau doré + check (IsNewlyMastered = masterisé PENDANT ce match).
type MatchNativeCommendation struct {
	ID      string  `json:"id"`
	Name    string  `json:"name,omitempty"`
	Count   int     `json:"count"`
	IconURL *string `json:"icon_url,omitempty"`
	// ProgressPct : pourcentage de progression vers le prochain palier (0..100),
	// 100 si masterisé. Anneau de progression côté front (CitationProgressRing.pct).
	ProgressPct float64 `json:"progress_pct"`
	// IsNewlyMastered : le palier final a été franchi PENDANT ce match (anneau doré
	// + check côté front). Faux pour une simple progression intermédiaire.
	IsNewlyMastered bool `json:"is_newly_mastered,omitempty"`
	// Cumulative : total absolu À VIE après ce match (= Progress). Affiché « cumul/seuil ».
	Cumulative int `json:"cumulative,omitempty"`
	// TierIndex : nombre de paliers atteints (0 = aucun, TierCount = maîtrisé).
	TierIndex int `json:"tier_index,omitempty"`
	// TierCount : nombre total de paliers (longueur de tier_targets). 0 → pas de
	// paliers connus → le front masque l'anneau (dégradation : icône + count seuls).
	TierCount int `json:"tier_count,omitempty"`
	// NextTierTarget : seuil absolu du prochain palier (0 si maîtrisé).
	NextTierTarget int `json:"next_tier_target,omitempty"`
}
