// Package domain — home.go : types pour la page d'accueil Mission Control.
//
// Correspond à l'endpoint GET /api/v1/players/{slug}/pages/home.
// Les types BattlePassResponse et ChallengesResponse sont retournés
// directement par leurs propres endpoints (best-effort, nécessite auth Sprint 15).
package domain

import "time"

// HomeMatchRow / HomeSessionRow ont été déplacés vers `internal/legacymatch`
// (P4.3 finale cleanup) — toutes les références ont migré, plus d'alias dans
// domain.

// HomeMediaRow est une ligne brute chargée depuis Q28 (médias récents).
type HomeMediaRow struct {
	FileName       string
	MatchID        *string
	MatchStartTime *time.Time
}

// HomeSkillPeakRow est la projection brute d'un pic historique CSR ou LUSR
// lu depuis match_skill_rank / player_csr_snapshots.
//
// MeasurementMatchesRemaining (10..0) indique la phase de placement :
//   - nil : champ non renseigné par le repo (legacy / pas de logique placement)
//   - 0   : phase de placement terminée → rating_value + tier valides
//   - >0  : encore N matchs avant la fin du placement → BadgeImageURL doit
//     pointer sur unranked_(10-N).png et RatingValue peut être 0
//
// La même sémantique vaut pour CSR (10 matchs Microsoft, source player_csr_snapshots)
// et pour LUSR (10 matchs par playlist_group, dérivé de match_skill_rank côté local).
type HomeSkillPeakRow struct {
	RatingValue                 float64
	TierLabel                   *string
	BadgeImageURL               *string
	MeasurementMatchesRemaining *int
	// PlacementTotal = seuil placement (5 ou 10). Propagé vers HomeSkillPeakSummary.PlacementTotal.
	PlacementTotal *int
	// Tier (EN, ex. "Gold"/"Diamond"/"Onyx") + SubTier (1..6, 0 pour Onyx) bruts du
	// peak, pour la bande de progression ORDINALE (analysis.SkillTierBand). Le
	// sous-palier est renseigné par chaque système (CSR/LUSR) sur sa propre échelle.
	Tier    string
	SubTier int
}

// HomeSpartanIdentityRow est la projection brute de l'identité record pour la home.
type HomeSpartanIdentityRow struct {
	SpartanID         *string
	BannerImageURL    *string
	EmblemImageURL    *string
	BackdropImageURL  *string
	HighestCSR        *HomeSkillPeakRow
	HighestLUSR       *HomeSkillPeakRow
	RankNumber        int
	RankName          *string
	RankTier          *string
	RankImageURL      *string
	AdornmentImageURL *string
	CurrentXP         int
	XPForNextRank     int
	IsMaxRank         bool
}

// ---------------------------------------------------------------------------
// Blocs Hero Card
// ---------------------------------------------------------------------------

// HeroKPIs contient les KPIs globaux du joueur affichés dans le hero card.
type HeroKPIs struct {
	WinRate                float64  `json:"win_rate"`
	GlobalRatio            *float64 `json:"global_ratio,omitempty"`
	AvgKDA                 *float64 `json:"avg_kda,omitempty"`
	AvgAccuracy            *float64 `json:"avg_accuracy,omitempty"`
	TotalMatches           int      `json:"total_matches"`
	Wins                   int      `json:"wins"`
	Draws                  int      `json:"draws"`
	DNFs                   int      `json:"dnfs"`
	Losses                 int      `json:"losses"`
	TotalPlaytimeSecs      int      `json:"total_playtime_secs"`
	FavoriteWeaponName     string   `json:"favorite_weapon_name,omitempty"`
	FavoriteWeaponKills    int      `json:"favorite_weapon_kills,omitempty"`
	FavoritePlaylistName   string   `json:"favorite_playlist_name,omitempty"`
	FavoritePlaylistCount  int      `json:"favorite_playlist_count,omitempty"`
	AvgOffensiveConversion *float64 `json:"avg_offensive_conversion,omitempty"`
	AvgDefensiveResistance *float64 `json:"avg_defensive_resistance,omitempty"`
	// DmgPerKill / DmgPerDeath : dégâts moyens par frag / par mort, agrégés
	// (Σ damage_dealt / Σ kills, Σ damage_taken / Σ deaths). Nil si dénominateur
	// nul. Affichés à côté du rendement/résistance (parité bande Synthesis).
	DmgPerKill  *float64 `json:"dmg_per_kill,omitempty"`
	DmgPerDeath *float64 `json:"dmg_per_death,omitempty"`
}

// HeroTrend représente la variation des métriques clés sur une fenêtre glissante.
// Calcul : 5 derniers matchs vs 5 précédents.
type HeroTrend struct {
	RatioDelta    *float64 `json:"ratio_delta,omitempty"`
	AccuracyDelta *float64 `json:"accuracy_delta,omitempty"`
	WinRateDelta  *float64 `json:"win_rate_delta,omitempty"`
}

// HomeHeroCard est la carte briefing principale affichée en haut de l'accueil.
type HomeHeroCard struct {
	PlayerName string     `json:"player_name"`
	KPIs       HeroKPIs   `json:"kpis"`
	Trend      *HeroTrend `json:"trend,omitempty"`
}

// HomeCareerRankSummary représente le rang carrière courant dans la home.
type HomeCareerRankSummary struct {
	RankNumber        int     `json:"rank_number"`
	RankTitle         string  `json:"rank_title"`
	NextRankTitle     string  `json:"next_rank_title,omitempty"`
	RankImageURL      *string `json:"rank_image_url,omitempty"`
	AdornmentImageURL *string `json:"adornment_image_url,omitempty"`
	CurrentXP         int     `json:"current_xp"`
	XPForNextRank     int     `json:"xp_for_next_rank"`
	// TotalXP : XP de carrière cumulée (Σ seuils des rangs précédents + current_xp).
	// Affichée au rang max (Héros), où current_xp/xp_for_next valent 0. 0 si le
	// catalog n'a pas les seuils (career_ranks.xp_required).
	TotalXP     int     `json:"total_xp"`
	ProgressPct float64 `json:"progress_pct"`
	IsMaxRank   bool    `json:"is_max_rank"`
}

// HomeSkillPeakSummary représente un pic historique CSR ou LUSR affiché sur la home.
//
// MeasurementMatchesRemaining (10..0) signale au front la phase de placement :
//   - omis (nil) : pas de logique placement disponible (legacy clients tolérés)
//   - 0           : phase terminée → afficher rating_value + tier_label
//   - >0          : afficher "En placement (10-N)/10" et utiliser badge_image_url
//     (qui pointe déjà sur unranked_(10-N).png côté backend)
//
// Valable aussi bien pour CSR (10 matchs Microsoft) que pour LUSR (10 matchs
// par playlist_group). Le front lit ce champ pour décider du rendu sans
// deviner via heuristique.
type HomeSkillPeakSummary struct {
	RatingValue                 float64 `json:"rating_value"`
	TierLabel                   *string `json:"tier_label,omitempty"`
	BadgeImageURL               *string `json:"badge_image_url,omitempty"`
	MeasurementMatchesRemaining *int    `json:"measurement_matches_remaining,omitempty"`
	// PlacementTotal = nombre total de matchs de placement requis pour la
	// saison du peak (5 depuis Season 3, 10 historique). Permet au front
	// d'afficher "En placement (X/N)" avec le bon dénominateur.
	// nil → fallback front à 10 (back-compat clients anciens).
	PlacementTotal *int `json:"placement_total,omitempty"`
	// Progression ORDINALE dans le palier courant (0..100), pour la barre à droite
	// du rating. Calculée par analysis.SkillTierBand (via le sous-palier I..VI,
	// indépendante de l'échelle CSR/LUSR). nil hors phase matured.
	TierProgressPct *float64 `json:"tier_progress_pct,omitempty"`
	// Libellé localisé du palier suivant (extrémité droite de la barre, ex.
	// "Onyx", "Platine"). nil pour Onyx (sommet, pas de palier suivant).
	NextTierLabel *string `json:"next_tier_label,omitempty"`
}

// HomePlaylistRank associe une playlist récente à son dernier rang compétitif connu.
// MeasurementMatchesRemaining est renseigné quand le joueur est en phase de placement
// (CSR ranked, 10→0 matchs restants) pour permettre au front de différencier les états
// de placement (badge unranked_N.png déjà composé côté backend dans BadgeImageURL).
type HomePlaylistRank struct {
	PlaylistName                string   `json:"playlist_name"`
	IsRanked                    bool     `json:"is_ranked"`
	RatingType                  *string  `json:"rating_type,omitempty"` // "CSR" | "LUSR" — nil si aucun rang calculé
	RatingValue                 *float64 `json:"rating_value,omitempty"`
	TierLabel                   *string  `json:"tier_label,omitempty"`
	BadgeImageURL               *string  `json:"badge_image_url,omitempty"`
	MeasurementMatchesRemaining *int     `json:"measurement_matches_remaining,omitempty"`
	// PlacementTotal = seuil placement de la saison du match (5 ou 10).
	// nil → fallback front à 10 (back-compat).
	PlacementTotal *int `json:"placement_total,omitempty"`
	// TierProgressPct : remplissage ORDINAL de la barre (0..100) via le sous-palier
	// (n/6), indépendant de l'échelle CSR/LUSR. Calculé par analysis.SkillTierBand
	// (même bande que le skill peak). nil hors phase matured (placement / sans rang)
	// → pas de barre côté front.
	TierProgressPct *float64 `json:"tier_progress_pct,omitempty"`
	// NextTierLabel : libellé localisé du SOUS-PALIER suivant (extrémité droite de
	// la barre, ex. "Or V", "Platine I"). nil pour Onyx (sommet).
	NextTierLabel *string `json:"next_tier_label,omitempty"`
}

// HomeSpartanIdentity représente le bloc identitaire compact de la home.
type HomeSpartanIdentity struct {
	BannerImageURL   *string                `json:"banner_image_url,omitempty"`
	SpartanID        *string                `json:"spartan_id,omitempty"`
	EmblemImageURL   *string                `json:"emblem_image_url,omitempty"`
	BackdropImageURL *string                `json:"backdrop_image_url,omitempty"`
	HighestCSR       *HomeSkillPeakSummary  `json:"highest_csr,omitempty"`
	HighestLUSR      *HomeSkillPeakSummary  `json:"highest_lusr,omitempty"`
	CareerRank       *HomeCareerRankSummary `json:"career_rank,omitempty"`
}

// ---------------------------------------------------------------------------
// Signaux et highlights
// ---------------------------------------------------------------------------

// HighlightItem est un fait saillant synthétique pour la zone signaux.
// ValueColor indique la couleur sémantique de Value : "positive", "warning", "neutral", "negative" ou "".
// Slides permet à une tuile unique de faire défiler plusieurs faits (ex. « Série »).
// Quand Slides est non vide, Value/Detail portent le premier slide ; le front défile les autres.
//
// TitleKey est la clé i18n (ex: "highlight.title.perf_avg"). Le front la résout.
// DetailKey + DetailParams permettent de rendre un détail traduit avec paramètres.
type HighlightItem struct {
	TitleKey     string           `json:"title_key,omitempty"`
	Title        string           `json:"title,omitempty"` // legacy fallback, ne pas remplir pour les nouvelles tuiles
	Value        string           `json:"value"`
	Detail       string           `json:"detail,omitempty"` // détail déjà localisé (legacy) ou vide
	DetailKey    string           `json:"detail_key,omitempty"`
	DetailParams map[string]any   `json:"detail_params,omitempty"`
	ValueColor   string           `json:"value_color,omitempty"`
	Slides       []HighlightSlide `json:"slides,omitempty"`
}

// HighlightSlide est un fait secondaire dans une tuile à défilement.
type HighlightSlide struct {
	LabelKey     string         `json:"label_key,omitempty"`
	Label        string         `json:"label,omitempty"` // legacy fallback
	Value        string         `json:"value"`
	Detail       string         `json:"detail,omitempty"`
	DetailKey    string         `json:"detail_key,omitempty"`
	DetailParams map[string]any `json:"detail_params,omitempty"`
	ValueColor   string         `json:"value_color,omitempty"`
}

// ---------------------------------------------------------------------------
// Matchs récents
// ---------------------------------------------------------------------------

// RecentMatchItem représente un match récent dans la timeline.
type RecentMatchItem struct {
	MatchID         string     `json:"match_id"`
	Title           string     `json:"title"`
	Detail          string     `json:"detail"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	OutcomeLabel    string     `json:"outcome_label"`
	OutcomeTone     string     `json:"outcome_tone"`
	ScoreLabel      *string    `json:"score_label,omitempty"`
	NarrativeBadges []string   `json:"narrative_badges,omitempty"`
	IsFavorite      bool       `json:"is_favorite"`
	// S56 — champs enrichis pour MatchCard
	MapUI                    *string                `json:"map_ui,omitempty"`
	ModeUI                   *string                `json:"mode_ui,omitempty"`
	PlaylistUI               *string                `json:"playlist_ui,omitempty"`
	Kills                    *int                   `json:"kills,omitempty"`
	Deaths                   *int                   `json:"deaths,omitempty"`
	Assists                  *int                   `json:"assists,omitempty"`
	PerformanceScoreRelative *int                   `json:"performance_score_relative,omitempty"`
	OffensiveConversion      *float64               `json:"offensive_conversion,omitempty"`
	DefensiveResistance      *float64               `json:"defensive_resistance,omitempty"`
	DamageDealt              *float64               `json:"damage_dealt,omitempty"`
	DamageTaken              *float64               `json:"damage_taken,omitempty"`
	MapImageURL              *string                `json:"map_image_url,omitempty"`
	SkillRatingValue         *int                   `json:"skill_rating_value,omitempty"`
	SkillRatingType          *string                `json:"skill_rating_type,omitempty"`
	SkillTierLabel           *string                `json:"skill_tier_label,omitempty"`
	SkillRatingDelta         *float64               `json:"skill_rating_delta,omitempty"`
	SkillPlaylistGroup       *string                `json:"skill_playlist_group,omitempty"`
	SkillRankImageURL        *string                `json:"skill_rank_image_url,omitempty"`
	SkillProgressPct         *float64               `json:"skill_progress_pct,omitempty"`
	SkillPointsInTier        *int                   `json:"skill_points_in_tier,omitempty"`
	KDA                      *float64               `json:"kda,omitempty"`
	DurationSecs             *int                   `json:"duration_secs,omitempty"`
	Accuracy                 *float64               `json:"accuracy,omitempty"`
	AvgLifeSecs              *float64               `json:"avg_life_secs,omitempty"`
	TeamMMR                  *float64               `json:"team_mmr,omitempty"`
	EnemyMMR                 *float64               `json:"enemy_mmr,omitempty"`
	DeltaMMR                 *float64               `json:"delta_mmr,omitempty"`
	IsWithFriends            *bool                  `json:"is_with_friends,omitempty"`
	RankInTeam               *int                   `json:"rank_in_team,omitempty"`
	HeadshotKills            *int                   `json:"headshot_kills,omitempty"`
	PerfectKills             *int                   `json:"perfect_kills,omitempty"`
	TopMedals                []RecentMatchMedal     `json:"top_medals,omitempty"`
	TopCitations             []MatchCitationSnippet `json:"top_citations,omitempty"`
	// SessionLabel : permet au frontend de filtrer recent_matches par session
	// (ex. OutcomeSequenceTape limitée à la dernière session). Bug #6.
	SessionLabel *string `json:"session_label,omitempty"`
	// IsRanked : true si la playlist est classée (CSR officiel).
	// Source : canonical.MatchSummary.IsRanked (issu de match_registry.is_ranked).
	IsRanked bool `json:"is_ranked,omitempty"`
}

// RecentMatchMedal est une médaille compacte pour l'affichage dans MatchCard.
//
// Icône title-aware : ImageURL (PNG, HINF) OU les champs Sprite* (feuille + offset,
// Halo 5). Mutuellement exclusifs ; mêmes tags JSON que l'Asset Drawer (AssetMeta)
// pour réutiliser le rendu front (background-image + background-position).
type RecentMatchMedal struct {
	MedalID      int64  `json:"medal_id"`
	Name         string `json:"name"`
	Count        int    `json:"count"`
	Description  string `json:"description,omitempty"`
	ImageURL     string `json:"image_url"`
	Difficulty   string `json:"difficulty,omitempty"`
	SpriteSheet  string `json:"sprite_sheet,omitempty"`
	SpriteLeft   int    `json:"sprite_left,omitempty"`
	SpriteTop    int    `json:"sprite_top,omitempty"`
	SpriteWidth  int    `json:"sprite_width,omitempty"`
	SpriteHeight int    `json:"sprite_height,omitempty"`
}

// MatchCitationSnippet est une citation progressée dans un match, pour l'affichage MatchCard.
type MatchCitationSnippet struct {
	Key             string  `json:"key"`
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	ImageURL        *string `json:"image_url,omitempty"`
	Delta           int     `json:"delta"`
	ProgressPct     float64 `json:"progress_pct"`
	IsNewlyMastered bool    `json:"is_newly_mastered,omitempty"`
	// Cumulative : total absolu après ce match (ex: 35 sur tiers [10,20,30,50,100]).
	Cumulative int `json:"cumulative,omitempty"`
	// TierIndex : palier atteint (0 = aucun, len(tiers) = maîtrisé).
	TierIndex int `json:"tier_index,omitempty"`
	// TierCount : nombre total de paliers (longueur de tier_targets).
	TierCount int `json:"tier_count,omitempty"`
	// NextTierTarget : seuil absolu du prochain palier (0 si maîtrisé).
	NextTierTarget int `json:"next_tier_target,omitempty"`
}

// HomeMatchCitationRaw est une ligne brute merger depuis match_citations + citation_mappings.
type HomeMatchCitationRaw struct {
	Norm        string
	Display     string
	Description string
	ImagePath   string
	TierTargets string // CSV ex: "10,20,30,50,100"
	Delta       int    // valeur du match (mc.value)
	Cumulative  int    // SUM(value) sur tous les matchs jusqu'à ce match inclus
}

// HomeMatchCommendationRaw est une commendation NATIVE progressée sur un match
// (Halo 5 : shared.match_commendations ⨝ commendation_definitions). Sert à remplir
// le slot TopCitations de la MatchCard pour les titres SANS moteur de citations
// dérivé (citations.engine = not_exposed) mais AVEC commendations natives
// (commendations.native = supported). Cf. enrichMatchesWithCommendations.
type HomeMatchCommendationRaw struct {
	ID      string // UUID natif de la commendation (= MatchCitationSnippet.Key)
	Name    string // nom localisé (commendation_definitions, FR > EN)
	IconURL string // URL icône CDN native (vide si définition absente)
	Count   int    // progression gagnée CE match (Progress − PreviousProgress)
	// Progress : total cumulatif À VIE de la commendation À CE match (valeur absolue
	// de 343, match_commendations.progress). Sert à calculer le palier atteint +
	// masterisé (parité Cumulative des citations Infinite). 0 si la colonne est NULL
	// (lignes écrites avant l'ajout de progress → dégradation : anneau vide).
	Progress int
	// TierTargets : CSV croissant des seuils de paliers de la commendation
	// (commendation_definitions.tier_targets, format Infinite). Vide → pas de paliers
	// connus (Meta/Daily, ou définition pré-tier_targets) → ProgressPct=0.
	TierTargets string
}

// ---------------------------------------------------------------------------
// Résumé de session
// ---------------------------------------------------------------------------

// SessionSummaryItem est le résumé d'une session (solo ou escouade).
type SessionSummaryItem struct {
	SessionLabel         string     `json:"session_label"`
	MatchCount           int        `json:"match_count"`
	WinRate              float64    `json:"win_rate"`
	GlobalRatio          *float64   `json:"global_ratio,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	EndedAt              *time.Time `json:"ended_at,omitempty"`
	Wins                 int        `json:"wins"`
	Losses               int        `json:"losses"`
	Draws                int        `json:"draws"`
	DNFs                 int        `json:"dnfs"`
	AvgPlayerPerformance *float64   `json:"avg_player_performance,omitempty"`
	AvgTeamPerformance   *float64   `json:"avg_team_performance,omitempty"`
	AvgKDA               *float64   `json:"avg_kda,omitempty"`
	DominantPlaylist     *string    `json:"dominant_playlist,omitempty"`
	DominantMode         *string    `json:"dominant_mode,omitempty"`
	// Teammates (sessions escouade) : gamertags des coéquipiers les plus présents
	// dans la session (amis configurés, top 3 = MAX_SELECTION page Escouade). Sert
	// au deep-link card escouade → /squad (pré-sélection de la composition).
	// Vide pour les sessions solo ou si la donnée n'est pas disponible.
	Teammates []string `json:"teammates,omitempty"`
}

// ---------------------------------------------------------------------------
// Médias récents
// ---------------------------------------------------------------------------

// RecentMediaItem est une entrée compacte d'un média récent.
type RecentMediaItem struct {
	Basename       string     `json:"basename"`
	MatchID        *string    `json:"match_id,omitempty"`
	MatchStartTime *time.Time `json:"match_start_time,omitempty"`
}

// ---------------------------------------------------------------------------
// Réponses API
// ---------------------------------------------------------------------------

// HomePageResponse est la réponse agrégée de la page d'accueil Mission Control.
type HomePageResponse struct {
	Hero                HomeHeroCard         `json:"hero"`
	SpartanIdentity     *HomeSpartanIdentity `json:"spartan_identity,omitempty"`
	Highlights          []HighlightItem      `json:"highlights"`
	RecentMatches       []RecentMatchItem    `json:"recent_matches"`
	FavoriteMatches     []RecentMatchItem    `json:"favorite_matches"`
	RecentMedia         []RecentMediaItem    `json:"recent_media"`
	SoloSession         *SessionSummaryItem  `json:"solo_session,omitempty"`
	SquadSession        *SessionSummaryItem  `json:"squad_session,omitempty"`
	SoloSessions        []SessionSummaryItem `json:"solo_sessions,omitempty"`
	SquadSessions       []SessionSummaryItem `json:"squad_sessions,omitempty"`
	HasRankedHistory    bool                 `json:"has_ranked_history"`
	HasUnrankedHistory  bool                 `json:"has_unranked_history"`
	RecentPlaylistRanks []HomePlaylistRank   `json:"recent_playlist_ranks,omitempty"`
}

// BattlePassResponse contient les informations Battle Pass live ou depuis le cache DB.
// available=false avec error_hint="auth_required" tant que l'auth n'est pas portée (Sprint 15).
// SnapshotAt (RFC3339) horodate la donnée affichée — `now` en cas d'appel live réussi,
// `MAX(snapshot_at)` du dernier snapshot persisté en fallback cache.
type BattlePassResponse struct {
	Available   bool    `json:"available"`
	Rank        *int    `json:"rank,omitempty"`
	RewardTrack *string `json:"reward_track,omitempty"`
	Progress    *int    `json:"progress,omitempty"`
	FromCache   bool    `json:"from_cache,omitempty"`
	SnapshotAt  *string `json:"snapshot_at,omitempty"`
	ErrorHint   *string `json:"error_hint,omitempty"`
}

// ChallengeItem représente un défi actif détaillé pour l'accueil.
// IsSquad est nil pour les défis Halo saison (jamais escouade) ;
// true pour les défis Prestige issus d'un SquadChallenge.
type ChallengeItem struct {
	ChallengePath   string   `json:"challenge_path"`
	TrackingID      *string  `json:"tracking_id,omitempty"`
	Title           string   `json:"title"`
	Description     *string  `json:"description,omitempty"`
	ImageURL        *string  `json:"image_url,omitempty"`
	ProgressCurrent *int     `json:"progress_current,omitempty"`
	ProgressTarget  *int     `json:"progress_target,omitempty"`
	ProgressPercent *float64 `json:"progress_percent,omitempty"`
	XPReward        *int     `json:"xp_reward,omitempty"`
	IsSquad         *bool    `json:"is_squad,omitempty"`
}

// ChallengesResponse contient le résumé des défis actifs ou depuis le cache DB.
// available=false avec error_hint="auth_required" tant que l'auth n'est pas portée (Sprint 15).
// SnapshotAt (RFC3339) horodate la donnée affichée — `now` en cas d'appel live réussi,
// `MAX(snapshot_at)` agrégé sur les snapshots du joueur en fallback cache.
type ChallengesResponse struct {
	Available   bool            `json:"available"`
	Total       *int            `json:"total,omitempty"`
	Completed   *int            `json:"completed,omitempty"`
	XPAvailable *int            `json:"xp_available,omitempty"`
	NextExpiry  *string         `json:"next_expiry,omitempty"`
	Items       []ChallengeItem `json:"items,omitempty"`
	FromCache   bool            `json:"from_cache,omitempty"`
	SnapshotAt  *string         `json:"snapshot_at,omitempty"`
	ErrorHint   *string         `json:"error_hint,omitempty"`
}
